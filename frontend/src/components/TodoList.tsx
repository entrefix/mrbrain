import { useState, useEffect } from 'react';
import { format, formatDistanceToNow } from 'date-fns';
import { motion, AnimatePresence } from 'framer-motion';
import { FiCalendar, FiFlag, FiEdit2, FiTrash2, FiMenu, FiClock } from 'react-icons/fi';
import { DragDropContext, Droppable, Draggable, DropResult } from 'react-beautiful-dnd';
import type { Todo, Group } from '../types';
import type { PendingTodo } from '../pages/Dashboard';

// Helper to format relative time (2d ago, 5m ago, etc.)
function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffSeconds / 60);
  const diffHours = Math.floor(diffMinutes / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffSeconds < 60) return 'just now';
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  if (diffDays < 30) return `${Math.floor(diffDays / 7)}w ago`;
  return formatDistanceToNow(date, { addSuffix: true });
}

interface TodoListProps {
  todos: Todo[];
  groups: Group[];
  pendingTodos: PendingTodo[];
  newlyCreatedId: string | null;
  onStatusChange: (id: string, status: 'pending' | 'completed') => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onEdit: (todo: Todo) => void;
  onReorder: (startIndex: number, endIndex: number) => Promise<void>;
}

const getItemStyle = (isDragging: boolean, draggableStyle: any) => ({
  ...draggableStyle,
  userSelect: 'none',
  background: isDragging ? 'rgba(var(--color-primary-500), 0.1)' : 'transparent',
});

const getListStyle = (isDraggingOver: boolean) => ({
  background: isDraggingOver ? 'rgba(var(--color-primary-500), 0.05)' : 'transparent',
  padding: '4px',
  borderRadius: '8px',
});

const TodoDroppable = ({ children }: { children: any }) => {
  const [enabled, setEnabled] = useState(false);

  useEffect(() => {
    const animation = requestAnimationFrame(() => setEnabled(true));
    return () => {
      cancelAnimationFrame(animation);
      setEnabled(false);
    };
  }, []);

  if (!enabled) return null;

  return (
    <Droppable droppableId="droppable">
      {children}
    </Droppable>
  );
};

// Pending Todo Card - same style as regular card, just with blinking loader
const PendingTodoCard = ({ todo, groups }: { todo: PendingTodo; groups: Group[] }) => {
  const group = groups.find(g => g.id === todo.group_id);

  return (
    <motion.div
      initial={{ opacity: 0, y: -20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, height: 0, marginBottom: 0 }}
      transition={{ duration: 0.3 }}
      className="bg-white dark:bg-black rounded-lg border border-gray-200 dark:border-gray-800"
    >
      <div className="py-2.5 px-3">
        {/* Main row: same structure as TodoCard */}
        <div className="flex items-center gap-2">
          {/* Drag handle placeholder (disabled) */}
          <div className="flex-shrink-0 opacity-30">
            <FiMenu className="w-4 h-4 text-gray-400" />
          </div>

          {/* Disabled checkbox */}
          <input
            type="checkbox"
            disabled
            className="flex-shrink-0 w-4 h-4 rounded border-gray-300 opacity-30"
          />

          {/* Title with blinking dot */}
          <div className="flex-1 min-w-0">
            <span className="text-gray-500 dark:text-gray-400 truncate block">
              {todo.title}
              <span className="inline-block w-2 h-2 ml-2 rounded-full bg-primary-500 animate-pulse" />
            </span>
          </div>

          {/* Group badge if set */}
          {group && (
            <span
              className="flex-shrink-0 text-xs px-1.5 py-0.5 rounded-full whitespace-nowrap"
              style={{
                backgroundColor: group.color_code + '20',
                color: group.color_code
              }}
            >
              {group.name}
            </span>
          )}
        </div>

        {/* Processing indicator */}
        <div className="mt-1 ml-10 text-xs text-gray-400">
          Processing...
        </div>
      </div>
    </motion.div>
  );
};

// Individual Todo Card Component
const TodoCard = ({
  todo,
  groups,
  isNewlyCreated,
  onStatusChange,
  onDelete,
  onEdit,
  dragHandleProps,
}: {
  todo: Todo;
  groups: Group[];
  isNewlyCreated: boolean;
  onStatusChange: (id: string, status: 'pending' | 'completed') => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onEdit: (todo: Todo) => void;
  dragHandleProps: any;
}) => {
  const [displayTitle, setDisplayTitle] = useState(isNewlyCreated ? '' : todo.title);

  // Typewriter animation for newly created todos
  useEffect(() => {
    if (isNewlyCreated && todo.title) {
      let i = 0;
      setDisplayTitle('');
      const timer = setInterval(() => {
        i++;
        setDisplayTitle(todo.title.slice(0, i));
        if (i >= todo.title.length) {
          clearInterval(timer);
        }
      }, 40);
      return () => clearInterval(timer);
    } else {
      setDisplayTitle(todo.title);
    }
  }, [todo.title, isNewlyCreated]);

  const getPriorityColor = (priority: Todo['priority']) => {
    switch (priority) {
      case 'high': return 'text-red-500';
      case 'medium': return 'text-yellow-500';
      case 'low': return 'text-green-500';
      default: return 'text-gray-400';
    }
  };

  const group = groups.find(g => g.id === todo.group_id);

  return (
    <motion.div
      initial={isNewlyCreated ? { opacity: 0, y: -10, scale: 0.98 } : false}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.3 }}
      className={`group bg-white dark:bg-black rounded-lg border transition-all duration-200 ${
        isNewlyCreated
          ? 'border-primary-300 dark:border-primary-700 shadow-md shadow-primary-100 dark:shadow-primary-900/20'
          : 'border-gray-200 dark:border-gray-800 hover:border-gray-300 dark:hover:border-gray-700'
      }`}
    >
      <div className="py-2.5 px-3">
        {/* Main row: drag handle, checkbox, title, due date, actions */}
        <div className="flex items-center gap-2">
          {/* Drag handle */}
          <div
            {...dragHandleProps}
            className="flex-shrink-0 cursor-grab active:cursor-grabbing opacity-40 hover:opacity-100 transition-opacity"
          >
            <FiMenu className="w-4 h-4 text-gray-400" />
          </div>

          {/* Checkbox */}
          <input
            type="checkbox"
            checked={todo.status === 'completed'}
            onChange={() => onStatusChange(todo.id, todo.status === 'completed' ? 'pending' : 'completed')}
            className="flex-shrink-0 w-4 h-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 focus:ring-offset-0"
          />

          {/* Title */}
          <div className="flex-1 min-w-0">
            <span className={`font-medium truncate block ${
              todo.status === 'completed'
                ? 'text-gray-400 line-through'
                : 'text-gray-900 dark:text-white'
            }`}>
              {displayTitle}
              {isNewlyCreated && displayTitle.length < todo.title.length && (
                <span className="inline-block w-0.5 h-4 bg-primary-500 animate-pulse ml-0.5" />
              )}
            </span>
          </div>

          {/* Due date (compact) */}
          {todo.due_date && (
            <span className="flex-shrink-0 text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap flex items-center gap-1">
              <FiCalendar className="w-3 h-3" />
              {format(new Date(todo.due_date), 'MMM d, h:mm a')}
            </span>
          )}

          {/* Inline action buttons - visible on hover */}
          <div className="flex-shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
            <button
              onClick={() => onEdit(todo)}
              className="p-1.5 rounded hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
              title="Edit"
            >
              <FiEdit2 className="w-3.5 h-3.5" />
            </button>
            <button
              onClick={() => onDelete(todo.id)}
              className="p-1.5 rounded hover:bg-red-50 dark:hover:bg-red-900/20 text-gray-400 hover:text-red-500 transition-colors"
              title="Delete"
            >
              <FiTrash2 className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>

        {/* Second row: tags, group, priority, created time */}
        {(todo.tags?.length > 0 || todo.group_id || todo.priority || todo.created_at) && (
          <div className="mt-1.5 ml-10 flex items-center gap-2 flex-wrap">
            {/* AI Tags */}
            {todo.tags?.map((tag, idx) => (
              <span
                key={idx}
                className="text-xs text-purple-600 dark:text-purple-400"
              >
                #{tag}
              </span>
            ))}

            {/* Spacer */}
            {todo.tags?.length > 0 && (todo.group_id || todo.priority) && (
              <span className="text-gray-300 dark:text-gray-700">•</span>
            )}

            {/* Group badge */}
            {group && (
              <span
                className="text-xs px-1.5 py-0.5 rounded-full"
                style={{
                  backgroundColor: group.color_code + '20',
                  color: group.color_code
                }}
              >
                {group.name}
              </span>
            )}

            {/* Priority flag */}
            {todo.priority && todo.priority !== 'medium' && (
              <FiFlag className={`w-3 h-3 ${getPriorityColor(todo.priority)}`} />
            )}

            {/* Created time - relative */}
            {todo.created_at && (
              <>
                <span className="text-gray-300 dark:text-gray-700">•</span>
                <span className="text-xs text-gray-400 dark:text-gray-500 flex items-center gap-1">
                  <FiClock className="w-3 h-3" />
                  {formatRelativeTime(todo.created_at)}
                </span>
              </>
            )}
          </div>
        )}
      </div>
    </motion.div>
  );
};

export default function TodoList({
  todos,
  groups,
  pendingTodos,
  newlyCreatedId,
  onStatusChange,
  onDelete,
  onEdit,
  onReorder
}: TodoListProps) {
  const handleDragEnd = (result: DropResult) => {
    if (!result.destination) return;

    const sourceIndex = result.source.index;
    const destinationIndex = result.destination.index;

    if (sourceIndex === destinationIndex) return;

    onReorder(sourceIndex, destinationIndex);
  };

  return (
    <div className="space-y-2">
      {/* Todo List */}
      <DragDropContext onDragEnd={handleDragEnd}>
        <TodoDroppable>
          {(provided: any, snapshot: any) => (
            <div
              {...provided.droppableProps}
              ref={provided.innerRef}
              style={getListStyle(snapshot.isDraggingOver)}
              className="space-y-2"
            >
              {todos.map((todo, index) => (
                <Draggable
                  key={todo.id}
                  draggableId={todo.id}
                  index={index}
                >
                  {(provided: any, snapshot: any) => (
                    <div
                      ref={provided.innerRef}
                      {...provided.draggableProps}
                      style={getItemStyle(snapshot.isDragging, provided.draggableProps.style)}
                    >
                      <TodoCard
                        todo={todo}
                        groups={groups}
                        isNewlyCreated={todo.id === newlyCreatedId}
                        onStatusChange={onStatusChange}
                        onDelete={onDelete}
                        onEdit={onEdit}
                        dragHandleProps={provided.dragHandleProps}
                      />
                    </div>
                  )}
                </Draggable>
              ))}
              {provided.placeholder}
            </div>
          )}
        </TodoDroppable>
      </DragDropContext>

      {/* Pending Todos - shown at bottom while processing (recently added at bottom) */}
      <AnimatePresence>
        {pendingTodos.map(pending => (
          <PendingTodoCard key={pending.tempId} todo={pending} groups={groups} />
        ))}
      </AnimatePresence>
    </div>
  );
}
