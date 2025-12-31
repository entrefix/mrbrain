import { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X } from '@phosphor-icons/react';
import { useDebounce } from '../hooks/useDebounce';

interface CreateNoteModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (title: string, content: string) => Promise<void>;
  type: 'memory' | 'todo';
}

export default function CreateNoteModal({ isOpen, onClose, onSubmit, type }: CreateNoteModalProps) {
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [autoTitle, setAutoTitle] = useState('');

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

  // Debounced auto-save
  const debouncedContent = useDebounce(content, 300);
  const debouncedTitle = useDebounce(title, 300);

  useEffect(() => {
    if (isOpen && (debouncedContent.trim() || debouncedTitle.trim())) {
      // Auto-save logic would go here if needed
      // For now, we'll save on submit
    }
  }, [debouncedContent, debouncedTitle, isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!content.trim() && !title.trim()) return;

    // Close modal immediately (optimistic UI)
    const finalTitle = title.trim() || autoTitle;
    const finalContent = content.trim();
    
    // Reset form and close
    setTitle('');
    setContent('');
    setAutoTitle('');
    onClose();
    
    // Submit in background
    try {
      await onSubmit(finalTitle, finalContent);
    } catch (error) {
      console.error('Failed to create memory:', error);
      // Error handling is done in parent component
    }
  };

  const handleClose = () => {
    setTitle('');
    setContent('');
    setAutoTitle('');
    onClose();
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
          initial={{ scale: 0.95, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          exit={{ scale: 0.95, opacity: 0 }}
          onClick={(e) => e.stopPropagation()}
          className="bg-white dark:bg-gray-900 rounded-2xl shadow-xl w-full max-w-2xl max-h-[80vh] flex flex-col"
        >
          {/* Header */}
          <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
              Create {type === 'memory' ? 'Memory' : 'Todo'}
            </h2>
            <button
              onClick={handleClose}
              className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
            >
              <X size={20} weight="regular" className="text-gray-500 dark:text-gray-400" />
            </button>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit} className="flex-1 flex flex-col overflow-hidden">
            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Title {!title.trim() && autoTitle && (
                    <span className="text-xs text-gray-500">(auto-generated)</span>
                  )}
                </label>
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder={autoTitle || 'Enter title...'}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                />
                {!title.trim() && autoTitle && (
                  <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    Will use: "{autoTitle}"
                  </p>
                )}
              </div>

              <div className="flex-1">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Content
                </label>
                <textarea
                  value={content}
                  onChange={(e) => setContent(e.target.value)}
                  placeholder="Start typing..."
                  rows={10}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-none"
                />
              </div>
            </div>

            {/* Footer */}
            <div className="p-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-end gap-3">
              <button
                type="button"
                onClick={handleClose}
                className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isSaving || (!content.trim() && !title.trim())}
                className="px-4 py-2 text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isSaving ? 'Creating...' : 'Create'}
              </button>
            </div>
          </form>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  );
}

