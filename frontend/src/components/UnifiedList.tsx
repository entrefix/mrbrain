import { useState } from 'react';
import { motion } from 'framer-motion';
import { Plus, UploadSimple, CircleNotch, CheckCircle, XCircle } from '@phosphor-icons/react';
import UnifiedCard from './UnifiedCard';
import type { Memory, Todo, RAGSearchResult, UploadJobStatusResponse } from '../types';

interface UnifiedListProps {
  items: (Memory | Todo | { document: RAGSearchResult['document']; isCitation?: boolean })[];
  type: 'memory' | 'todo';
  onCreateClick: () => void;
  onImportClick: () => void;
  onItemClick: (item: Memory | Todo | { document: RAGSearchResult['document']; isCitation?: boolean }) => void;
  onItemUpdate?: (todoId: string, currentStatus: 'pending' | 'completed') => void;
  onReorder?: (reorderedItems: Todo[]) => void;
  uploadStatus?: UploadJobStatusResponse | null;
}

export default function UnifiedList({ items, type, onCreateClick, onImportClick, onItemClick, onItemUpdate, onReorder, uploadStatus }: UnifiedListProps) {
  const isUploading = uploadStatus && uploadStatus.status !== 'completed' && uploadStatus.status !== 'failed';
  const isCompleted = uploadStatus?.status === 'completed';
  const isFailed = uploadStatus?.status === 'failed';
  
  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);

  // Only enable drag for todos
  const isTodoType = type === 'todo';

  const handleDragStart = (e: React.DragEvent, index: number) => {
    if (!isTodoType || !onReorder) return;
    const item = items[index];
    if (!('id' in item) || 'document' in item) return; // Skip citations
    setDraggedIndex(index);
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', String(index));
    // Create a custom drag image
    if (e.currentTarget instanceof HTMLElement) {
      const dragImage = e.currentTarget.closest('div[class*="motion"]') as HTMLElement;
      if (dragImage) {
        e.dataTransfer.setDragImage(dragImage, e.clientX - dragImage.getBoundingClientRect().left, e.clientY - dragImage.getBoundingClientRect().top);
      }
    }
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    if (!isTodoType || draggedIndex === null || draggedIndex === index) return;
    const item = items[index];
    if (!('id' in item) || 'document' in item) return; // Skip citations
    
    // Check if we're trying to move a completed item above a pending item
    const draggedItem = items[draggedIndex];
    if ('id' in draggedItem && !('document' in draggedItem)) {
      const draggedTodo = draggedItem as Todo;
      const dropTodo = item as Todo;
      
      if (draggedTodo.status === 'completed' && dropTodo.status === 'pending') {
        e.dataTransfer.dropEffect = 'none';
        return;
      }
    }
    
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    setDragOverIndex(index);
  };

  const handleDragLeave = () => {
    setDragOverIndex(null);
  };

  const handleDrop = (e: React.DragEvent, dropIndex: number) => {
    e.preventDefault();
    if (!isTodoType || !onReorder || draggedIndex === null || draggedIndex === dropIndex) {
      setDraggedIndex(null);
      setDragOverIndex(null);
      return;
    }

    const draggedItem = items[draggedIndex];
    const dropItem = items[dropIndex];
    
    // Skip if either is a citation
    if (!('id' in draggedItem) || 'document' in draggedItem || !('id' in dropItem) || 'document' in dropItem) {
      setDraggedIndex(null);
      setDragOverIndex(null);
      return;
    }

    const draggedTodo = draggedItem as Todo;
    const dropTodo = dropItem as Todo;
    
    // Prevent completed items from moving above pending items
    if (draggedTodo.status === 'completed' && dropTodo.status === 'pending') {
      setDraggedIndex(null);
      setDragOverIndex(null);
      return;
    }

    // Filter to get only todos (not citations)
    const todoItems = items.filter((item): item is Todo => 'id' in item && !('document' in item));
    
    // Find indices in the todoItems array
    const draggedTodoIndex = todoItems.findIndex(t => t.id === draggedTodo.id);
    const dropTodoIndex = todoItems.findIndex(t => t.id === dropTodo.id);
    
    if (draggedTodoIndex === -1 || dropTodoIndex === -1) {
      setDraggedIndex(null);
      setDragOverIndex(null);
      return;
    }

    // Create new array with reordered items
    const newItems = [...todoItems];
    const [removed] = newItems.splice(draggedTodoIndex, 1);
    newItems.splice(dropTodoIndex, 0, removed);

    // Call reorder handler with reordered todos
    if (onReorder) {
      onReorder(newItems);
    }
    
    setDraggedIndex(null);
    setDragOverIndex(null);
  };

  const handleDragEnd = () => {
    setDraggedIndex(null);
    setDragOverIndex(null);
  };
  
  return (
    <div className="px-6 sm:px-16 lg:px-24 xl:px-32 2xl:px-40">
      <div className="max-w-4xl mx-auto space-y-3">
        {/* Create Button */}
        <motion.button
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          onClick={onCreateClick}
          className="w-full bg-white dark:bg-gray-900 rounded-lg border-2 border-dashed border-gray-300 dark:border-gray-700 p-4 hover:border-primary-500 dark:hover:border-primary-400 hover:bg-primary-50/50 dark:hover:bg-primary-900/10 transition-all cursor-pointer flex items-center justify-center gap-2"
        >
          <Plus size={20} weight="bold" className="text-gray-400 dark:text-gray-500" />
          <p className="text-sm font-medium text-gray-600 dark:text-gray-400">
            {type === 'memory' ? 'Create Memory' : 'Create Todo'}
          </p>
        </motion.button>

        {/* Import Button - Only show for memories */}
        {type === 'memory' && (
          <motion.button
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            onClick={isUploading ? undefined : onImportClick}
            disabled={isUploading || false}
            className={`w-full bg-white dark:bg-gray-900 rounded-lg border-2 border-dashed p-3 transition-all flex items-center justify-center gap-2 ${
              isUploading
                ? 'border-primary-400 dark:border-primary-600 bg-primary-50/50 dark:bg-primary-900/20 cursor-default'
                : isCompleted
                ? 'border-green-400 dark:border-green-600 bg-green-50/50 dark:bg-green-900/20 cursor-default'
                : isFailed
                ? 'border-red-400 dark:border-red-600 bg-red-50/50 dark:bg-red-900/20 cursor-default'
                : 'border-gray-300 dark:border-gray-700 hover:border-primary-500 dark:hover:border-primary-400 hover:bg-primary-50/50 dark:hover:bg-primary-900/10 cursor-pointer'
            }`}
          >
            {isUploading ? (
              <>
                <CircleNotch size={16} weight="bold" className="text-primary-500 dark:text-primary-400 animate-spin" />
                <p className="text-sm font-medium text-primary-600 dark:text-primary-400">
                  {uploadStatus?.progress || 0}%
                </p>
              </>
            ) : isCompleted ? (
              <>
                <CheckCircle size={16} weight="bold" className="text-green-500 dark:text-green-400" />
                <p className="text-sm font-medium text-green-600 dark:text-green-400">
                  Done!
                </p>
              </>
            ) : isFailed ? (
              <>
                <XCircle size={16} weight="bold" className="text-red-500 dark:text-red-400" />
                <p className="text-sm font-medium text-red-600 dark:text-red-400">
                  Failed
                </p>
              </>
            ) : (
              <>
                <UploadSimple size={16} weight="bold" className="text-gray-400 dark:text-gray-500" />
                <p className="text-sm font-medium text-gray-600 dark:text-gray-400">
                  Import
                </p>
              </>
            )}
          </motion.button>
        )}

        {/* Items List */}
        {items.map((item, index) => {
          const itemId = 'id' in item ? item.id : ('document' in item ? item.document.id : `citation-${index}`);
          const isTodo = isTodoType && 'id' in item && !('document' in item);
          const todoItem = isTodo ? (item as Todo) : null;
          const isTodoCompleted = todoItem?.status === 'completed' || false;
          const isDragging = draggedIndex === index;
          const isDragOver = dragOverIndex === index;
          
          // Only enable drag for pending todos
          const dragHandleProps = isTodo && onReorder && !isTodoCompleted ? {
            draggable: true,
            onDragStart: (e: React.DragEvent) => {
              e.stopPropagation();
              handleDragStart(e, index);
            },
            onDragEnd: (e: React.DragEvent) => {
              e.stopPropagation();
              handleDragEnd();
            },
          } : undefined;

          return (
            <motion.div
              key={itemId}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.02 }}
              className={isDragOver ? 'ring-2 ring-primary-500 rounded-lg' : ''}
              onDragOver={isTodo ? (e) => handleDragOver(e, index) : undefined}
              onDragLeave={isTodo ? handleDragLeave : undefined}
              onDrop={isTodo ? (e) => handleDrop(e, index) : undefined}
            >
              <UnifiedCard
                item={item}
                type={type}
                onClick={() => onItemClick(item)}
                onStatusToggle={onItemUpdate}
                isDragging={isDragging}
                dragHandleProps={dragHandleProps}
              />
            </motion.div>
          );
        })}
      </div>
    </div>
  );
}

