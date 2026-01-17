import { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Image } from '@phosphor-icons/react';
import { toast } from 'react-hot-toast';
import { memoryApi } from '../api';

interface CreateNoteModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (title: string, content: string) => Promise<void>;
  onImageCreated?: () => void; // Callback when image creates a memory directly
  type: 'memory' | 'todo';
}

export default function CreateNoteModal({ isOpen, onClose, onSubmit, onImageCreated, type }: CreateNoteModalProps) {
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [autoTitle, setAutoTitle] = useState('');
  const [isImageUploading, setIsImageUploading] = useState(false);
  const contentRef = useRef<HTMLTextAreaElement>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);

  // Auto-generate title from content
  useEffect(() => {
    if (!title.trim() && content.trim()) {
      const firstLine = content.split('\n')[0].trim();
      const generatedTitle = firstLine.length > 50 ? firstLine.substring(0, 50) + '...' : firstLine;
      setAutoTitle(generatedTitle);
    } else {
      setAutoTitle('');
    }
  }, [content, title]);

  // Focus textarea when modal opens
  useEffect(() => {
    if (isOpen && contentRef.current) {
      setTimeout(() => contentRef.current?.focus(), 100);
    }
  }, [isOpen]);

  // Reset form when modal closes
  useEffect(() => {
    if (!isOpen) {
      setTitle('');
      setContent('');
      setAutoTitle('');
      setIsSaving(false);
    }
  }, [isOpen]);

  // Keyboard shortcut: Cmd/Ctrl + Enter to save
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
        e.preventDefault();
        handleSave();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, content, title, type]);

  const handleSave = async () => {
    // For memories, only content is required. For todos, either title or content is required.
    if (type === 'memory' && !content.trim()) {
      onClose();
      return;
    }
    if (type === 'todo' && !content.trim() && !title.trim()) {
      onClose();
      return;
    }

    setIsSaving(true);
    const finalTitle = type === 'todo' ? (title.trim() || autoTitle) : '';
    const finalContent = content.trim();
    
    try {
      await onSubmit(finalTitle, finalContent);
      // Reset form after successful save (keep modal open for rapid entry)
      setTitle('');
      setContent('');
      setAutoTitle('');
    } catch (error) {
      console.error('Failed to create memory:', error);
      // Error handling is done in parent component
    } finally {
      setIsSaving(false);
    }
  };

  const handleClose = async () => {
    // Auto-save on close if there's content
    if (type === 'memory' && content.trim()) {
      await handleSave();
    } else if (type === 'todo' && (content.trim() || title.trim())) {
      await handleSave();
    } else {
      onClose();
    }
  };

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file type
    const validTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/gif', 'image/webp'];
    if (!validTypes.includes(file.type)) {
      toast.error('Invalid image type. Supported: JPG, PNG, GIF, WebP');
      e.target.value = '';
      return;
    }

    // Validate file size (max 10MB)
    const maxSize = 10 * 1024 * 1024;
    if (file.size > maxSize) {
      toast.error('Image too large. Maximum size is 10MB');
      e.target.value = '';
      return;
    }

    setIsImageUploading(true);

    try {
      const result = await memoryApi.uploadImage(file);

      // Show success message
      const categoryInfo = result.vision_result.category !== 'Uncategorized'
        ? ` (${result.vision_result.category})`
        : '';
      toast.success(`Image processed${categoryInfo}! Notes extracted.`);

      // Close modal and notify parent to refresh
      onClose();
      onImageCreated?.();
    } catch (error: any) {
      const errorMsg = error.response?.data?.error || 'Failed to process image';
      toast.error(errorMsg);
    } finally {
      setIsImageUploading(false);
      e.target.value = '';
    }
  };

  if (!isOpen) return null;

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm"
        onClick={handleClose}
      >
        <motion.div
          initial={{ scale: 0.95, opacity: 0, y: 20 }}
          animate={{ scale: 1, opacity: 1, y: 0 }}
          exit={{ scale: 0.95, opacity: 0, y: 20 }}
          transition={{ duration: 0.2, ease: [0.4, 0, 0.2, 1] }}
          onClick={(e) => e.stopPropagation()}
          className="bg-white dark:bg-gray-900 rounded-lg shadow-2xl w-full max-w-2xl max-h-[85vh] flex flex-col"
          style={{
            boxShadow: '0 8px 16px rgba(0, 0, 0, 0.15), 0 0 0 1px rgba(0, 0, 0, 0.05)',
          }}
        >
          {/* Content Area - Google Keep style */}
          <div className="flex-1 flex flex-col overflow-hidden">
            {/* Title field - Only show for todos, not for memories */}
            {type === 'todo' && (
              <div className="px-6 pt-6 pb-3">
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder={autoTitle || 'Title'}
                  className="w-full text-lg font-medium bg-transparent border-none outline-none text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 resize-none"
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault();
                      contentRef.current?.focus();
                    }
                  }}
                />
              </div>
            )}

            {/* Content textarea */}
            <div className="flex-1 px-6 py-4 overflow-y-auto">
              <textarea
                ref={contentRef}
                value={content}
                onChange={(e) => setContent(e.target.value)}
                placeholder={type === 'memory' ? 'Start typing your memory...' : 'Start typing...'}
                className="w-full min-h-[250px] text-sm bg-transparent border-none outline-none text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 resize-none"
                style={{ 
                  fontFamily: 'inherit',
                  lineHeight: '1.6',
                }}
              />
            </div>

            {/* Footer - Google Keep style */}
            <div className="px-6 py-3 flex items-center justify-between border-t border-gray-100 dark:border-gray-800">
              {/* Image Upload - Only for memories */}
              <div className="flex items-center gap-2">
                {type === 'memory' && (
                  <label className={`group relative ${isImageUploading ? 'cursor-not-allowed' : 'cursor-pointer'}`}>
                    <input
                      ref={imageInputRef}
                      type="file"
                      accept="image/jpeg,image/jpg,image/png,image/gif,image/webp"
                      onChange={handleImageUpload}
                      className="hidden"
                      disabled={isImageUploading || isSaving}
                    />
                    <div className={`flex items-center gap-2 px-3 py-2 text-sm font-medium rounded transition-colors ${
                      isImageUploading
                        ? 'text-gray-400 dark:text-gray-500'
                        : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800'
                    }`}>
                      {isImageUploading ? (
                        <div className="w-4 h-4 border-2 border-primary-500 border-t-transparent rounded-full animate-spin" />
                      ) : (
                        <Image size={18} weight="regular" />
                      )}
                      <span className="hidden sm:inline">
                        {isImageUploading ? 'Processing...' : 'Upload Image'}
                      </span>
                    </div>
                    {/* Tooltip */}
                    {!isImageUploading && (
                      <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2 py-1 bg-gray-900 dark:bg-gray-700 text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap pointer-events-none">
                        Extract notes from image
                      </div>
                    )}
                  </label>
                )}
              </div>

              <button
                type="button"
                onClick={handleClose}
                disabled={isSaving || isImageUploading}
                className="px-5 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                {isSaving ? 'Saving...' : (
                  <>
                    Save
                    <kbd className="hidden sm:inline-flex items-center gap-0.5 px-1.5 py-0.5 text-[10px] font-medium text-gray-400 dark:text-gray-500 bg-gray-100 dark:bg-gray-800 rounded border border-gray-200 dark:border-gray-700">
                      <span>{typeof navigator !== 'undefined' && navigator.platform?.includes('Mac') ? '⌘' : 'Ctrl'}</span>
                      <span>↵</span>
                    </kbd>
                  </>
                )}
              </button>
            </div>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  );
}

