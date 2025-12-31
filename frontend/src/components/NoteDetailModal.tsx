import { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, Trash, CheckSquare, BookmarkSimple } from '@phosphor-icons/react';
import { useDebounce } from '../hooks/useDebounce';
import type { Memory, Todo, RAGSearchResult } from '../types';

interface NoteDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  item: Memory | Todo | { document: RAGSearchResult['document']; isCitation?: boolean } | null;
  type: 'memory' | 'todo';
  onUpdate: (id: string, data: any) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onConvert?: (item: Memory) => Promise<void>;
}

export default function NoteDetailModal({
  isOpen,
  onClose,
  item,
  type,
  onUpdate,
  onDelete,
  onConvert,
}: NoteDetailModalProps) {
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);

  useEffect(() => {
    if (item) {
      if ('document' in item) {
        // Citation - read-only
        setTitle(item.document.title || item.document.content.substring(0, 50));
        setContent(item.document.content || '');
      } else if (type === 'memory') {
        const memory = item as Memory;
        setTitle(memory.summary || memory.content.split('\n')[0].substring(0, 50));
        setContent(memory.content || '');
      } else {
        const todo = item as Todo;
        setTitle(todo.title || '');
        setContent(todo.description || '');
      }
      setHasChanges(false);
    }
  }, [item, type]);

  const debouncedTitle = useDebounce(title, 300);
  const debouncedContent = useDebounce(content, 300);

  useEffect(() => {
    if (!item || !isOpen || isSaving) return;
    if ('document' in item) return; // Citations are read-only

    const originalTitle = type === 'memory' 
      ? (item as Memory).summary || ''
      : (item as Todo).title || '';
    const originalContent = type === 'memory'
      ? (item as Memory).content || ''
      : (item as Todo).description || '';

    if (debouncedTitle !== originalTitle || debouncedContent !== originalContent) {
      setHasChanges(true);
      handleAutoSave();
    }
  }, [debouncedTitle, debouncedContent, item, type, isOpen]);

  const handleAutoSave = async () => {
    if (!item || 'document' in item) return;
    if (!hasChanges) return;

    setIsSaving(true);
    try {
      if (type === 'memory') {
        await onUpdate((item as Memory).id, {
          content: debouncedContent,
          summary: debouncedTitle || undefined,
        });
      } else {
        await onUpdate((item as Todo).id, {
          title: debouncedTitle,
          description: debouncedContent || null,
        });
      }
      setHasChanges(false);
    } catch (error) {
      console.error('Auto-save failed:', error);
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!item || 'document' in item) return;
    if (window.confirm('Are you sure you want to delete this item?')) {
      await onDelete((item as Memory | Todo).id);
      onClose();
    }
  };

  const handleConvert = async () => {
    if (!item || type !== 'memory' || 'document' in item) return;
    if (onConvert) {
      await onConvert(item as Memory);
      onClose();
    }
  };

  if (!isOpen || !item) return null;

  const isCitation = 'document' in item;
  const itemId = 'id' in item ? item.id : item.document.id;

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      >
        <motion.div
          initial={{ scale: 0.95, opacity: 0, y: 20 }}
          animate={{ scale: 1, opacity: 1, y: 0 }}
          exit={{ scale: 0.95, opacity: 0, y: 20 }}
          onClick={(e) => e.stopPropagation()}
          className="bg-white dark:bg-gray-900 rounded-2xl shadow-xl w-full max-w-3xl max-h-[90vh] flex flex-col"
        >
          {/* Header */}
          <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
            <div className="flex items-center gap-3">
              {type === 'memory' ? (
                <BookmarkSimple size={24} weight="regular" className="text-primary-600 dark:text-primary-400" />
              ) : (
                <CheckSquare size={24} weight="regular" className="text-primary-600 dark:text-primary-400" />
              )}
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                {isCitation ? 'Citation' : type === 'memory' ? 'Memory' : 'Todo'}
              </h2>
              {isSaving && (
                <span className="text-xs text-gray-500 dark:text-gray-400">Saving...</span>
              )}
              {hasChanges && !isSaving && (
                <span className="text-xs text-primary-600 dark:text-primary-400">Unsaved changes</span>
              )}
            </div>
            <div className="flex items-center gap-2">
              {!isCitation && type === 'memory' && onConvert && (
                <button
                  onClick={handleConvert}
                  className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
                  title="Convert to Todo"
                >
                  <CheckSquare size={20} weight="regular" className="text-gray-500 dark:text-gray-400" />
                </button>
              )}
              {!isCitation && (
                <button
                  onClick={handleDelete}
                  className="p-2 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
                  title="Delete"
                >
                  <Trash size={20} weight="regular" className="text-red-600 dark:text-red-400" />
                </button>
              )}
              <button
                onClick={onClose}
                className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
              >
                <X size={20} weight="regular" className="text-gray-500 dark:text-gray-400" />
              </button>
            </div>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-y-auto p-6">
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Title
                </label>
                {isCitation ? (
                  <p className="text-gray-900 dark:text-white font-medium">
                    {title || 'Untitled'}
                  </p>
                ) : (
                  <input
                    type="text"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                  />
                )}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Content
                </label>
                {isCitation ? (
                  <div className="prose dark:prose-invert max-w-none">
                    <p className="text-gray-700 dark:text-gray-300 whitespace-pre-wrap">
                      {content}
                    </p>
                  </div>
                ) : (
                  <textarea
                    value={content}
                    onChange={(e) => setContent(e.target.value)}
                    rows={15}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-none"
                  />
                )}
              </div>

              {!isCitation && type === 'memory' && 'category' in item && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Category
                  </label>
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    {(item as Memory).category}
                  </p>
                </div>
              )}
            </div>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  );
}

