import { useState } from 'react';
import { motion } from 'framer-motion';
import { DotsSixVertical } from '@phosphor-icons/react';
import type { Memory, Todo, RAGSearchResult, Status } from '../types';
import { formatDateForDisplay, getDateUrgency } from '../utils/dateParser';

interface UnifiedCardProps {
  item: Memory | Todo | { document: RAGSearchResult['document']; isCitation?: boolean };
  type: 'memory' | 'todo';
  onClick: () => void;
  onStatusToggle?: (todoId: string, currentStatus: Status) => void;
  isDragging?: boolean;
  dragHandleProps?: {
    draggable: boolean;
    onDragStart: (e: React.DragEvent) => void;
    onDragEnd: (e: React.DragEvent) => void;
  };
}

export default function UnifiedCard({ item, type, onClick, onStatusToggle, isDragging, dragHandleProps }: UnifiedCardProps) {
  const getTitle = () => {
    if ('document' in item) {
      // Citation from RAG
      return item.document.title || item.document.content.substring(0, 50);
    }
    if (type === 'memory') {
      const memory = item as Memory;
      // For memories, use summary or first part of content as title
      return memory.summary || memory.content.substring(0, 50);
    } else {
      const todo = item as Todo;
      return todo.title;
    }
  };

  const getSummary = () => {
    if ('document' in item) {
      // Citation from RAG
      return item.document.content.substring(0, 150);
    }
    if (type === 'memory') {
      const memory = item as Memory;
      return memory.content.length > 150 ? memory.content.substring(0, 150) + '...' : memory.content;
    } else {
      const todo = item as Todo;
      return todo.description || '';
    }
  };

  // Extract todo-specific data
  const todo = type === 'todo' && !('document' in item) ? (item as Todo) : null;
  const isCompleted = todo?.status === 'completed';
  const dueDate = todo?.due_date;
  const status = todo?.status;

  // Track if we're dragging to prevent click events
  const [isDraggingState, setIsDraggingState] = useState(false);

  const title = getTitle();
  const summary = getSummary();
  const isCitation = 'document' in item && (item as any).isCitation;
  const isProcessing = 'isProcessing' in item && (item as any).isProcessing;

  // Get date urgency for styling
  const dateUrgency = dueDate ? getDateUrgency(dueDate) : null;
  const formattedDueDate = dueDate ? formatDateForDisplay(new Date(dueDate)) : null;

  // Handle checkbox click
  const handleCheckboxClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (todo && onStatusToggle && !isProcessing) {
      onStatusToggle(todo.id, status as Status);
    }
  };

  // Get due date styling based on urgency
  const getDueDateClass = () => {
    switch (dateUrgency) {
      case 'overdue':
        return 'text-red-600 dark:text-red-400';
      case 'today':
        return 'text-orange-600 dark:text-orange-400';
      case 'this-week':
        return 'text-yellow-600 dark:text-yellow-400';
      case 'future':
        return 'text-gray-500 dark:text-gray-400';
      default:
        return 'text-gray-500 dark:text-gray-400';
    }
  };

  const cardContent = (
    <div
      draggable={dragHandleProps?.draggable || false}
      onDragStart={(e) => {
        setIsDraggingState(true);
        if (dragHandleProps?.onDragStart) {
          dragHandleProps.onDragStart(e);
        }
      }}
      onDragEnd={(e) => {
        // Small delay to prevent click event after drag
        setTimeout(() => setIsDraggingState(false), 100);
        if (dragHandleProps?.onDragEnd) {
          dragHandleProps.onDragEnd(e);
        }
      }}
      onMouseDown={(e) => {
        // Track mouse down to help distinguish click vs drag
        if (dragHandleProps?.draggable) {
          const startX = e.clientX;
          const startY = e.clientY;
          const handleMouseMove = (moveEvent: MouseEvent) => {
            const deltaX = Math.abs(moveEvent.clientX - startX);
            const deltaY = Math.abs(moveEvent.clientY - startY);
            if (deltaX > 5 || deltaY > 5) {
              setIsDraggingState(true);
            }
            document.removeEventListener('mousemove', handleMouseMove);
          };
          document.addEventListener('mousemove', handleMouseMove);
        }
      }}
      className={`bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-4 hover:shadow-md transition-all relative group ${
        isCitation ? 'ring-2 ring-primary-500/50' : ''
      } ${isProcessing ? 'opacity-70' : ''} ${isCompleted ? 'opacity-60' : ''} ${isDragging ? 'opacity-50 scale-95' : ''} cursor-pointer`}
      onClick={() => {
        // Don't trigger click if we just finished dragging
        if (!isDraggingState) {
          onClick();
        }
      }}
    >
      {isProcessing && (
        <div className="absolute inset-0 flex items-center justify-center bg-white/50 dark:bg-gray-900/50 rounded-lg backdrop-blur-sm z-10">
          <div className="flex items-center gap-2 text-primary-600 dark:text-primary-400">
            <div className="w-4 h-4 border-2 border-primary-600 border-t-transparent rounded-full animate-spin" />
            <span className="text-xs font-medium">Processing...</span>
          </div>
        </div>
      )}
      
      <div className={`mb-2 flex items-start gap-3 ${type === 'todo' && todo && !isCitation ? '' : ''}`}>
        {/* Circle indicator for todos */}
        {type === 'todo' && todo && !isCitation && (
          <button
            onClick={handleCheckboxClick}
            disabled={isProcessing}
            className={`mt-0.5 flex-shrink-0 z-20 w-4 h-4 rounded-full border-2 flex items-center justify-center transition-all hover:scale-110 disabled:opacity-50 disabled:cursor-not-allowed ${
              isCompleted
                ? 'bg-primary-600 border-primary-600 dark:bg-primary-500 dark:border-primary-500'
                : 'border-gray-300 dark:border-gray-600 hover:border-primary-500 dark:hover:border-primary-400 bg-white dark:bg-gray-900'
            }`}
            aria-label={isCompleted ? 'Mark as pending' : 'Mark as completed'}
          >
            {isCompleted && (
              <div className="w-1.5 h-1.5 rounded-full bg-white" />
            )}
          </button>
        )}
        
        <div className="flex-1 min-w-0">
          <h3 className={`text-sm font-semibold line-clamp-3 leading-tight ${
            isCompleted
              ? 'line-through text-gray-500 dark:text-gray-500'
              : 'text-gray-900 dark:text-white'
          }`}>
            {title}
          </h3>
          
          {/* Due Date display */}
          {type === 'todo' && todo && !isCitation && formattedDueDate && (
            <div className="mt-2">
              <span className={`text-xs font-medium ${getDueDateClass()}`}>
                {formattedDueDate}
              </span>
            </div>
          )}
          
          {summary && (
            <p className={`text-xs line-clamp-6 mt-2 ${
              isCompleted
                ? 'text-gray-500 dark:text-gray-500'
                : 'text-gray-600 dark:text-gray-400'
            }`}>
              {summary}
            </p>
          )}
        </div>
      </div>
      
      {isCitation && (
        <div className="mt-2 pt-2 border-t border-gray-200 dark:border-gray-700">
          <span className="text-xs text-primary-600 dark:text-primary-400 font-medium">
            From Ask AI
          </span>
        </div>
      )}
      
      {/* Drag handle for todos - shown on hover, positioned on right */}
      {type === 'todo' && todo && !isCitation && dragHandleProps && (
        <div
          onClick={(e) => e.stopPropagation()}
          onMouseDown={(e) => e.stopPropagation()}
          className="absolute top-3 right-3 z-20 p-1 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 cursor-grab active:cursor-grabbing transition-opacity opacity-0 group-hover:opacity-100"
          aria-label="Drag to reorder"
        >
          <DotsSixVertical size={16} weight="bold" />
        </div>
      )}
    </div>
  );

  // Wrap in motion.div for animations if not dragging
  if (dragHandleProps?.draggable) {
    return cardContent;
  }

  return (
    <motion.div
      layout
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
    >
      {cardContent}
    </motion.div>
  );
}

