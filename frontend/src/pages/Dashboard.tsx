import { useState, useEffect, useMemo } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { toast } from 'react-hot-toast';
import { CaretDown, CaretRight } from '@phosphor-icons/react';
import { useAuth } from '../contexts/AuthContext';
import { todoApi, groupApi } from '../api';
import TodoList from '../components/TodoList';
import StickyBottomInput from '../components/StickyBottomInput';
import EditTodoModal from '../components/EditTodoModal';
import FilterBar, { TodoFilters } from '../components/FilterBar';
import type { Todo, Group, TodoCreate, TodoUpdate } from '../types';

export interface PendingTodo {
  tempId: string;
  title: string;
  group_id: string | null;
  priority: 'low' | 'medium' | 'high';
}

export default function Dashboard() {
  const { user } = useAuth();
  const [todos, setTodos] = useState<Todo[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(true);
  const [editingTodo, setEditingTodo] = useState<Todo | null>(null);
  const [pendingTodos, setPendingTodos] = useState<PendingTodo[]>([]);
  const [newlyCreatedId, setNewlyCreatedId] = useState<string | null>(null);
  const [showCompleted, setShowCompleted] = useState(false);
  const [filters, setFilters] = useState<TodoFilters>({
    searchText: '',
    tags: [],
    groupId: null,
    dateRange: { from: null, to: null },
  });

  useEffect(() => {
    if (user) {
      fetchTodos();
      fetchGroups();
    }
  }, [user]);

  // Get all unique tags from todos for filter dropdown
  const availableTags = useMemo(() => {
    const tagSet = new Set<string>();
    todos.forEach(todo => {
      todo.tags?.forEach(tag => tagSet.add(tag));
    });
    return Array.from(tagSet).sort();
  }, [todos]);

  // Apply filters to todos
  const filteredTodos = useMemo(() => {
    return todos.filter(todo => {
      // Search text filter (title or description)
      if (filters.searchText) {
        const searchLower = filters.searchText.toLowerCase();
        const titleMatch = todo.title.toLowerCase().includes(searchLower);
        const descMatch = todo.description?.toLowerCase().includes(searchLower);
        if (!titleMatch && !descMatch) return false;
      }

      // Tags filter (must have ALL selected tags)
      if (filters.tags.length > 0) {
        const todoTags = todo.tags || [];
        const hasAllTags = filters.tags.every(tag => todoTags.includes(tag));
        if (!hasAllTags) return false;
      }

      // Group filter
      if (filters.groupId && todo.group_id !== filters.groupId) {
        return false;
      }

      // Date range filter (on due_date)
      if (filters.dateRange.from) {
        if (!todo.due_date) return false;
        const dueDate = new Date(todo.due_date);
        const fromDate = new Date(filters.dateRange.from);
        if (dueDate < fromDate) return false;
      }
      if (filters.dateRange.to) {
        if (!todo.due_date) return false;
        const dueDate = new Date(todo.due_date);
        const toDate = new Date(filters.dateRange.to);
        toDate.setHours(23, 59, 59, 999); // End of day
        if (dueDate > toDate) return false;
      }

      return true;
    });
  }, [todos, filters]);

  // Split filtered todos into ongoing and completed
  const ongoingTodos = useMemo(() =>
    filteredTodos.filter(todo => todo.status === 'pending'),
    [filteredTodos]
  );

  const completedTodos = useMemo(() =>
    filteredTodos.filter(todo => todo.status === 'completed'),
    [filteredTodos]
  );

  const fetchTodos = async () => {
    try {
      const data = await todoApi.getAll();
      setTodos(data);
    } catch (error) {
      toast.error('Failed to fetch todos');
    } finally {
      setLoading(false);
    }
  };

  const fetchGroups = async () => {
    try {
      const data = await groupApi.getAll();
      setGroups(data);
    } catch (error) {
      toast.error('Failed to fetch groups');
    }
  };

  const handleCreateTodo = async (todo: TodoCreate) => {
    const tempId = `temp-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

    // 1. Add to pending list at bottom (non-blocking - allows multiple, recently added at bottom)
    const pending: PendingTodo = {
      tempId,
      title: todo.title,
      group_id: todo.group_id,
      priority: todo.priority,
    };
    setPendingTodos(prev => [...prev, pending]);

    // 2. API call runs async - doesn't block next input
    try {
      const newTodo = await todoApi.create(todo);

      // 3. Remove from pending, add real todo at bottom (highest position = bottom of list)
      setPendingTodos(prev => prev.filter(p => p.tempId !== tempId));
      setNewlyCreatedId(newTodo.id);
      setTodos(prev => [...prev, newTodo]);

      // Track analytics
      trackEvent('todo_created', {
        has_due_date: !!todo.due_date,
        has_group: !!todo.group_id,
        priority: todo.priority || 'medium',
      });

      // 4. Clear animation marker after animation completes
      setTimeout(() => setNewlyCreatedId(null), 1500);
    } catch (error) {
      // Remove failed pending todo
      setPendingTodos(prev => prev.filter(p => p.tempId !== tempId));
      toast.error('Failed to create todo');
    }
  };

  const handleStatusChange = async (id: string, status: 'pending' | 'completed') => {
    try {
      await todoApi.update(id, { status });
      setTodos(todos.map(todo =>
        todo.id === id ? { ...todo, status } : todo
      ));
      trackEvent('todo_status_changed', { status });
      toast.success(`Todo marked as ${status}`);
    } catch (error) {
      toast.error('Failed to update todo status');
    }
  };

  const handleDeleteTodo = async (id: string) => {
    try {
      await todoApi.delete(id);
      setTodos(todos.filter(todo => todo.id !== id));
      toast.success('Todo deleted successfully');
    } catch (error) {
      toast.error('Failed to delete todo');
    }
  };

  const handleCreateGroup = async (name: string) => {
    try {
      const colorCode = '#' + Math.floor(Math.random()*16777215).toString(16).padStart(6, '0');
      const newGroup = await groupApi.create({ name, color_code: colorCode });
      setGroups([...groups, newGroup]);
      return newGroup;
    } catch (error) {
      toast.error('Failed to create group');
      throw error;
    }
  };

  const handleEditTodo = async (id: string, todo: TodoUpdate) => {
    try {
      const updatedTodo = await todoApi.update(id, todo);
      setTodos(todos.map(t =>
        t.id === id ? updatedTodo : t
      ));
      toast.success('Todo updated successfully');
    } catch (error) {
      toast.error('Failed to update todo');
    }
  };

  const handleReorder = async (startIndex: number, endIndex: number, isCompleted: boolean = false) => {
    try {
      // Get the appropriate filtered list (ongoing or completed)
      const filteredList = isCompleted ? completedTodos : ongoingTodos;

      // Get the todo being moved from the filtered list
      const movedTodo = filteredList[startIndex];
      if (!movedTodo) return;

      // Create a new filtered list with the reordered item
      const newFilteredList = Array.from(filteredList);
      const [removed] = newFilteredList.splice(startIndex, 1);
      newFilteredList.splice(endIndex, 0, removed);

      // Now rebuild the full todos array maintaining the new order for the filtered section
      // while keeping the other section (completed/ongoing) in their original positions
      const otherList = isCompleted ? ongoingTodos : completedTodos;

      // Combine: ongoing first, then completed (matching our UI display order)
      const newTodos = isCompleted
        ? [...otherList, ...newFilteredList]
        : [...newFilteredList, ...otherList];

      // Update the state immediately for smooth UI
      setTodos(newTodos);

      // Assign new positions to ALL pending todos to maintain order
      const pendingTodosOrdered = newTodos.filter(t => t.status === 'pending');
      const updatedPositions = pendingTodosOrdered.map((todo, index) => ({
        id: todo.id,
        position: ((index + 1) * 1000).toString(),
      }));

      // Update positions in the database (only for pending todos since that's what we reorder)
      if (updatedPositions.length > 0) {
        await todoApi.reorder({ todos: updatedPositions });
      }
    } catch (error) {
      console.error('Reorder error:', error);
      toast.error('Failed to reorder todo');
      // Revert the state on error
      fetchTodos();
    }
  };

  return (
    <div className="h-full flex flex-col bg-surface-light dark:bg-surface-dark relative">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="flex-1 overflow-y-auto px-4 sm:px-6 py-6 pb-32"
      >
        <div className="max-w-4xl mx-auto">
      <div className="mb-6 pl-10 lg:pl-0">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Todos</h1>
        <p className="mt-2 text-gray-600 dark:text-gray-400">
          Manage your tasks and stay organized
        </p>
      </div>

      {/* Filter Bar */}
      <FilterBar
        filters={filters}
        onFiltersChange={setFilters}
        availableTags={availableTags}
        groups={groups}
      />

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-600"></div>
        </div>
      ) : todos.length === 0 && pendingTodos.length === 0 ? (
        <div className="bg-surface-light-muted dark:bg-surface-dark-muted rounded-2xl p-8 text-center">
          <div className="max-w-md mx-auto">
            <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
              <span className="text-3xl">✨</span>
            </div>
            <h3 className="text-lg font-heading text-gray-900 dark:text-white mb-2">
              No tasks yet
            </h3>
            <p className="text-gray-500 dark:text-gray-400 text-sm">
              Start by adding a task below. Try typing something like "Buy groceries tomorrow" - the date will be automatically detected!
            </p>
          </div>
        </div>
      ) : (
        <div className="space-y-6">
          {/* Ongoing Tasks Section */}
          <div>
            <div className="flex items-center gap-2 mb-3">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                Ongoing
              </h2>
              <span className="px-2 py-0.5 text-xs rounded-full bg-primary-100 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400">
                {ongoingTodos.length}
              </span>
            </div>
            {ongoingTodos.length === 0 && pendingTodos.length === 0 ? (
              <div className="bg-surface-light-muted dark:bg-surface-dark-muted rounded-xl p-6 text-center">
                <p className="text-gray-500 dark:text-gray-400 text-sm">
                  {filteredTodos.length === 0 && todos.length > 0
                    ? 'No tasks match your filters'
                    : 'No ongoing tasks. Add one below!'}
                </p>
              </div>
            ) : (
              <TodoList
                todos={ongoingTodos}
                groups={groups}
                pendingTodos={pendingTodos}
                newlyCreatedId={newlyCreatedId}
                onStatusChange={handleStatusChange}
                onDelete={handleDeleteTodo}
                onEdit={setEditingTodo}
                onReorder={handleReorder}
              />
            )}
          </div>

          {/* Completed Tasks Section - Collapsible */}
          {completedTodos.length > 0 && (
            <div>
              <button
                onClick={() => setShowCompleted(!showCompleted)}
                className="flex items-center gap-2 mb-3 group"
              >
                <span className="text-gray-400 group-hover:text-gray-600 dark:group-hover:text-gray-300 transition-colors">
                  {showCompleted ? <CaretDown size={16} weight="bold" /> : <CaretRight size={16} weight="bold" />}
                </span>
                <h2 className="text-lg font-semibold text-gray-500 dark:text-gray-400 group-hover:text-gray-700 dark:group-hover:text-gray-300 transition-colors">
                  Completed
                </h2>
                <span className="px-2 py-0.5 text-xs rounded-full bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400">
                  {completedTodos.length}
                </span>
              </button>

              <AnimatePresence>
                {showCompleted && (
                  <motion.div
                    initial={{ height: 0, opacity: 0 }}
                    animate={{ height: 'auto', opacity: 1 }}
                    exit={{ height: 0, opacity: 0 }}
                    transition={{ duration: 0.2 }}
                    className="overflow-hidden"
                  >
                    <TodoList
                      todos={completedTodos}
                      groups={groups}
                      pendingTodos={[]}
                      newlyCreatedId={null}
                      onStatusChange={handleStatusChange}
                      onDelete={handleDeleteTodo}
                      onEdit={setEditingTodo}
                      onReorder={handleReorder}
                    />
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          )}
        </div>
      )}

        </div>
      </motion.div>

      <EditTodoModal
        isOpen={!!editingTodo}
        onClose={() => setEditingTodo(null)}
        onSubmit={handleEditTodo}
        todo={editingTodo}
        groups={groups}
      />

      <StickyBottomInput
        groups={groups}
        onSubmit={handleCreateTodo}
        onCreateGroup={handleCreateGroup}
      />
    </div>
  );
}
