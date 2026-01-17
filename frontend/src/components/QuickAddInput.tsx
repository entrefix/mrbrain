import { useState, useRef, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { FiPlus, FiCalendar, FiFlag, FiFolder, FiX, FiChevronDown } from 'react-icons/fi';
import type { Group, TodoCreate } from '../types';
import { getTextSegments, parseDateFromText, extractTitleWithoutDate, dateToInputFormat, formatDateForDisplay } from '../utils/dateParser';

interface QuickAddInputProps {
  groups: Group[];
  onSubmit: (todo: TodoCreate) => Promise<void>;
  onCreateGroup: (name: string) => Promise<Group>;
}

export default function QuickAddInput({ groups, onSubmit, onCreateGroup }: QuickAddInputProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [dueDate, setDueDate] = useState('');
  const [priority, setPriority] = useState<'low' | 'medium' | 'high'>('medium');
  const [groupId, setGroupId] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [showGroupDropdown, setShowGroupDropdown] = useState(false);
  const [newGroupName, setNewGroupName] = useState('');
  const [creatingGroup, setCreatingGroup] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Set default group to "Personal" on mount
  useEffect(() => {
    const personalGroup = groups.find(g => g.name.toLowerCase() === 'personal');
    if (personalGroup && !groupId) {
      setGroupId(personalGroup.id);
    }
  }, [groups, groupId]);

  // Handle click outside to collapse
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        if (!title.trim() && isExpanded) {
          setIsExpanded(false);
        }
        setShowGroupDropdown(false);
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [title, isExpanded]);

  // Parse date from title as user types
  const parsedDate = parseDateFromText(title);
  const textSegments = getTextSegments(title);

  // Update due date when natural language date is detected
  useEffect(() => {
    if (parsedDate?.date) {
      setDueDate(dateToInputFormat(parsedDate.date));
    }
  }, [parsedDate?.date?.getTime()]);

  const handleFocus = () => {
    setIsExpanded(true);
  };

  const handleSubmit = async () => {
    if (!title.trim()) return;

    setLoading(true);
    try {
      // Frontend is now the source of truth for date parsing
      // Send cleaned title (without date text) and parsed date to backend
      const cleanedTitle = extractTitleWithoutDate(title);

      await onSubmit({
        title: cleanedTitle,  // Clean title without date text
        description: description || undefined,
        due_date: dueDate || null,  // Use frontend-parsed date
        priority,
        group_id: groupId || null,
      });

      // Reset form
      setTitle('');
      setDescription('');
      setDueDate('');
      setPriority('medium');
      // Keep the group selection
      setIsExpanded(false);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
    if (e.key === 'Escape') {
      if (!title.trim()) {
        setIsExpanded(false);
      }
    }
  };

  const handleCreateGroup = async () => {
    if (!newGroupName.trim()) return;
    setCreatingGroup(true);
    try {
      const newGroup = await onCreateGroup(newGroupName.trim());
      setGroupId(newGroup.id);
      setNewGroupName('');
      setShowGroupDropdown(false);
    } finally {
      setCreatingGroup(false);
    }
  };

  const selectedGroup = groups.find(g => g.id === groupId);

  const getPriorityColor = (p: string) => {
    switch (p) {
      case 'high': return 'text-red-500 bg-red-50 dark:bg-red-900/20';
      case 'medium': return 'text-yellow-500 bg-yellow-50 dark:bg-yellow-900/20';
      case 'low': return 'text-green-500 bg-green-50 dark:bg-green-900/20';
      default: return 'text-gray-500 bg-gray-50 dark:bg-gray-900/20';
    }
  };

  return (
    <div ref={containerRef} className="mb-6">
      <motion.div
        layout
        className="bg-white dark:bg-black rounded-xl shadow-sm border border-gray-200 dark:border-gray-800 overflow-hidden"
      >
        {/* Main input row */}
        <div className="p-4">
          <div className="flex items-center gap-3">
            <div className="flex-shrink-0">
              <div className="w-8 h-8 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
                <FiPlus className="text-primary-600 dark:text-primary-400" />
              </div>
            </div>

            <div className="flex-1 relative">
              {/* Highlighted input overlay */}
              {isExpanded && title && (
                <div className="absolute inset-0 pointer-events-none flex items-center text-lg">
                  {textSegments.map((segment, idx) => (
                    <span
                      key={idx}
                      className={segment.isDate
                        ? 'bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-400 rounded px-1'
                        : 'text-transparent'
                      }
                    >
                      {segment.text}
                    </span>
                  ))}
                </div>
              )}

              <input
                ref={inputRef}
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                onFocus={handleFocus}
                onKeyDown={handleKeyDown}
                placeholder="Add a new task... (try 'Buy groceries tomorrow')"
                className="w-full text-lg bg-transparent border-none outline-none text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500"
              />
            </div>

            {/* Quick action buttons when collapsed */}
            {!isExpanded && (
              <button
                onClick={() => {
                  setIsExpanded(true);
                  inputRef.current?.focus();
                }}
                className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 p-2"
              >
                <FiChevronDown />
              </button>
            )}
          </div>

          {/* Parsed date indicator */}
          {isExpanded && parsedDate?.date && (
            <div className="mt-2 ml-11 flex items-center gap-2 text-sm text-blue-600 dark:text-blue-400">
              <FiCalendar className="flex-shrink-0" />
              <span>Due: {formatDateForDisplay(parsedDate.date)}</span>
            </div>
          )}
        </div>

        {/* Expanded options */}
        <AnimatePresence>
          {isExpanded && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: 'auto', opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="border-t border-gray-100 dark:border-gray-800"
            >
              {/* Description */}
              <div className="px-4 py-3 border-b border-gray-100 dark:border-gray-800">
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Add description (optional)"
                  rows={2}
                  className="w-full text-sm bg-transparent border-none outline-none resize-none text-gray-700 dark:text-gray-300 placeholder-gray-400 dark:placeholder-gray-500"
                />
              </div>

              {/* Options row */}
              <div className="px-4 py-3 flex items-center gap-3 flex-wrap">
                {/* Group selector */}
                <div className="relative">
                  <button
                    type="button"
                    onClick={() => setShowGroupDropdown(!showGroupDropdown)}
                    className="flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                  >
                    {selectedGroup ? (
                      <>
                        <span
                          className="w-3 h-3 rounded-full"
                          style={{ backgroundColor: selectedGroup.color_code }}
                        />
                        <span className="text-gray-700 dark:text-gray-300">{selectedGroup.name}</span>
                      </>
                    ) : (
                      <>
                        <FiFolder className="text-gray-400" />
                        <span className="text-gray-500">No group</span>
                      </>
                    )}
                    <FiChevronDown className="text-gray-400" />
                  </button>

                  {/* Group dropdown */}
                  <AnimatePresence>
                    {showGroupDropdown && (
                      <motion.div
                        initial={{ opacity: 0, y: -10 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, y: -10 }}
                        className="absolute top-full left-0 mt-1 w-56 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 z-50"
                      >
                        <div className="py-1 max-h-48 overflow-y-auto">
                          <button
                            onClick={() => {
                              setGroupId('');
                              setShowGroupDropdown(false);
                            }}
                            className="w-full px-4 py-2 text-left text-sm hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center gap-2"
                          >
                            <span className="w-3 h-3 rounded-full bg-gray-300 dark:bg-gray-600" />
                            <span className="text-gray-600 dark:text-gray-400">No group</span>
                          </button>
                          {groups.map((group) => (
                            <button
                              key={group.id}
                              onClick={() => {
                                setGroupId(group.id);
                                setShowGroupDropdown(false);
                              }}
                              className={`w-full px-4 py-2 text-left text-sm hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center gap-2 ${
                                group.id === groupId ? 'bg-gray-50 dark:bg-gray-700' : ''
                              }`}
                            >
                              <span
                                className="w-3 h-3 rounded-full"
                                style={{ backgroundColor: group.color_code }}
                              />
                              <span className="text-gray-700 dark:text-gray-300">{group.name}</span>
                              {group.is_default && (
                                <span className="text-xs text-gray-400">(Default)</span>
                              )}
                            </button>
                          ))}
                        </div>

                        {/* Create new group */}
                        <div className="border-t border-gray-200 dark:border-gray-700 p-2">
                          <div className="flex gap-2">
                            <input
                              type="text"
                              value={newGroupName}
                              onChange={(e) => setNewGroupName(e.target.value)}
                              placeholder="New group name"
                              className="flex-1 text-sm px-3 py-1.5 rounded border border-gray-200 dark:border-gray-600 bg-transparent text-gray-700 dark:text-gray-300"
                              onKeyDown={(e) => {
                                if (e.key === 'Enter') {
                                  e.preventDefault();
                                  handleCreateGroup();
                                }
                              }}
                            />
                            <button
                              onClick={handleCreateGroup}
                              disabled={creatingGroup || !newGroupName.trim()}
                              className="px-3 py-1.5 text-sm bg-primary-600 text-white rounded hover:bg-primary-700 disabled:opacity-50"
                            >
                              {creatingGroup ? '...' : 'Add'}
                            </button>
                          </div>
                        </div>
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>

                {/* Due date picker */}
                <div className="flex items-center gap-2">
                  <FiCalendar className="text-gray-400" />
                  <input
                    type="datetime-local"
                    value={dueDate}
                    onChange={(e) => setDueDate(e.target.value)}
                    className="text-sm px-2 py-1 rounded border border-gray-200 dark:border-gray-700 bg-transparent text-gray-700 dark:text-gray-300"
                  />
                  {dueDate && (
                    <button
                      onClick={() => setDueDate('')}
                      className="text-gray-400 hover:text-gray-600"
                    >
                      <FiX size={14} />
                    </button>
                  )}
                </div>

                {/* Priority selector */}
                <div className="flex items-center gap-1">
                  <FiFlag className="text-gray-400 mr-1" />
                  {(['low', 'medium', 'high'] as const).map((p) => (
                    <button
                      key={p}
                      onClick={() => setPriority(p)}
                      className={`px-2 py-1 text-xs rounded capitalize ${
                        priority === p
                          ? getPriorityColor(p)
                          : 'text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800'
                      }`}
                    >
                      {p}
                    </button>
                  ))}
                </div>

                {/* Spacer */}
                <div className="flex-1" />

                {/* Submit button */}
                <button
                  onClick={handleSubmit}
                  disabled={loading || !title.trim()}
                  className="px-4 py-1.5 bg-primary-600 text-white text-sm font-medium rounded-lg hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  {loading ? 'Adding...' : 'Add Task'}
                </button>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </motion.div>
    </div>
  );
}
