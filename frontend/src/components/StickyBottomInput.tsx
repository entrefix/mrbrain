import { useState, useRef, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Plus, CalendarBlank, Flag, X, Check } from '@phosphor-icons/react';
import { format } from 'date-fns';
import type { Group, TodoCreate } from '../types';
import { getTextSegments, parseDateFromText, extractTitleWithoutDate, dateToInputFormat, formatDateForDisplay } from '../utils/dateParser';

interface StickyBottomInputProps {
  groups: Group[];
  onSubmit: (todo: TodoCreate) => Promise<void>;
  onCreateGroup: (name: string) => Promise<Group>;
}

export default function StickyBottomInput({ groups, onSubmit, onCreateGroup }: StickyBottomInputProps) {
  const [title, setTitle] = useState('');
  const [dueDate, setDueDate] = useState('');
  const [priority, setPriority] = useState<'low' | 'medium' | 'high'>('medium');
  const [groupId, setGroupId] = useState<string>('');
  const [loading, setLoading] = useState(false);

  // Popover states
  const [showDatePicker, setShowDatePicker] = useState(false);
  const [showPriorityPicker, setShowPriorityPicker] = useState(false);
  const [showGroupPicker, setShowGroupPicker] = useState(false);
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

  // Handle click outside to close popovers
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setShowDatePicker(false);
        setShowPriorityPicker(false);
        setShowGroupPicker(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Parse date from title as user types
  const parsedDate = parseDateFromText(title);
  const textSegments = getTextSegments(title);

  // Update due date when natural language date is detected
  useEffect(() => {
    if (parsedDate?.date) {
      setDueDate(dateToInputFormat(parsedDate.date));
    }
  }, [parsedDate?.date?.getTime()]);

  const handleSubmit = async () => {
    if (!title.trim()) return;

    setLoading(true);
    try {
      const cleanedTitle = extractTitleWithoutDate(title);

      await onSubmit({
        title: cleanedTitle,
        due_date: dueDate || null,
        priority,
        group_id: groupId || null,
      });

      // Reset form
      setTitle('');
      setDueDate('');
      setPriority('medium');
      // Keep the group selection
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
      setShowDatePicker(false);
      setShowPriorityPicker(false);
      setShowGroupPicker(false);
    }
  };

  const handleCreateGroup = async () => {
    if (!newGroupName.trim()) return;
    setCreatingGroup(true);
    try {
      const newGroup = await onCreateGroup(newGroupName.trim());
      setGroupId(newGroup.id);
      setNewGroupName('');
      setShowGroupPicker(false);
    } finally {
      setCreatingGroup(false);
    }
  };

  const setQuickDate = (type: 'today' | 'tomorrow' | 'nextWeek') => {
    const now = new Date();
    let date: Date;

    switch (type) {
      case 'today':
        date = now;
        date.setHours(17, 0, 0, 0); // 5 PM
        break;
      case 'tomorrow':
        date = new Date(now);
        date.setDate(date.getDate() + 1);
        date.setHours(9, 0, 0, 0); // 9 AM
        break;
      case 'nextWeek':
        date = new Date(now);
        date.setDate(date.getDate() + 7);
        date.setHours(9, 0, 0, 0);
        break;
    }

    setDueDate(dateToInputFormat(date));
    setShowDatePicker(false);
  };

  const selectedGroup = groups.find(g => g.id === groupId);

  const getPriorityLabel = (p: string) => {
    switch (p) {
      case 'high': return 'H';
      case 'medium': return 'M';
      case 'low': return 'L';
      default: return 'M';
    }
  };

  const getPriorityColor = (p: string) => {
    switch (p) {
      case 'high': return 'text-red-500 bg-red-50 dark:bg-red-900/30 border-red-200 dark:border-red-800';
      case 'medium': return 'text-yellow-500 bg-yellow-50 dark:bg-yellow-900/30 border-yellow-200 dark:border-yellow-800';
      case 'low': return 'text-green-500 bg-green-50 dark:bg-green-900/30 border-green-200 dark:border-green-800';
      default: return 'text-gray-500';
    }
  };

  const formatShortDate = (dateStr: string) => {
    if (!dateStr) return '--';
    try {
      const date = new Date(dateStr);
      return format(date, 'MMM d');
    } catch {
      return '--';
    }
  };

  // Render button components for reuse in both mobile and desktop layouts
  const DateButtonContent = ({ isMobile = false }: { isMobile?: boolean }) => (
    <div className="relative">
      <button
        onClick={() => {
          setShowDatePicker(!showDatePicker);
          setShowPriorityPicker(false);
          setShowGroupPicker(false);
        }}
        className={`flex items-center gap-1 ${isMobile ? 'px-3 py-2.5 min-h-[44px]' : 'px-2 py-1.5'} rounded-lg text-xs border transition-colors ${
          dueDate
            ? 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/30 border-blue-200 dark:border-blue-800'
            : 'text-gray-400 border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800'
        }`}
      >
        <CalendarBlank size={16} weight="regular" />
        <span className={`font-medium ${isMobile ? '' : 'hidden sm:inline'}`}>
          {dueDate ? formatShortDate(dueDate) : (isMobile ? '--' : '')}
        </span>
      </button>

      {/* Date picker popover */}
      <AnimatePresence>
        {showDatePicker && (
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 10 }}
            className="absolute bottom-full right-0 mb-2 w-64 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 p-3 z-30"
          >
            {/* Quick date buttons */}
            <div className="flex gap-2 mb-3">
              <button
                onClick={() => setQuickDate('today')}
                className="flex-1 px-2 py-2 text-xs rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
              >
                Today
              </button>
              <button
                onClick={() => setQuickDate('tomorrow')}
                className="flex-1 px-2 py-2 text-xs rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
              >
                Tomorrow
              </button>
              <button
                onClick={() => setQuickDate('nextWeek')}
                className="flex-1 px-2 py-2 text-xs rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300"
              >
                Next Week
              </button>
            </div>

            {/* DateTime input */}
            <input
              type="datetime-local"
              value={dueDate}
              onChange={(e) => setDueDate(e.target.value)}
              className="w-full text-sm px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-700 bg-transparent text-gray-700 dark:text-gray-300"
            />

            {/* Clear button */}
            {dueDate && (
              <button
                onClick={() => {
                  setDueDate('');
                  setShowDatePicker(false);
                }}
                className="mt-2 w-full px-3 py-2 text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
              >
                Clear date
              </button>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );

  const PriorityButtonContent = ({ isMobile = false }: { isMobile?: boolean }) => (
    <div className="relative">
      <button
        onClick={() => {
          setShowPriorityPicker(!showPriorityPicker);
          setShowDatePicker(false);
          setShowGroupPicker(false);
        }}
        className={`flex items-center gap-1 ${isMobile ? 'px-3 py-2.5 min-h-[44px]' : 'px-2 py-1.5'} rounded-lg text-xs border transition-colors ${getPriorityColor(priority)}`}
      >
        <Flag size={16} weight="regular" />
        <span className="font-medium">{getPriorityLabel(priority)}</span>
      </button>

      {/* Priority picker popover */}
      <AnimatePresence>
        {showPriorityPicker && (
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 10 }}
            className="absolute bottom-full right-0 mb-2 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 p-2 flex gap-1 z-30"
          >
            {(['low', 'medium', 'high'] as const).map((p) => (
              <button
                key={p}
                onClick={() => {
                  setPriority(p);
                  setShowPriorityPicker(false);
                }}
                className={`px-3 py-2 text-xs rounded-lg border capitalize transition-colors ${
                  priority === p
                    ? getPriorityColor(p)
                    : 'text-gray-500 border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700'
                }`}
              >
                {p}
              </button>
            ))}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );

  const GroupButtonContent = ({ isMobile = false }: { isMobile?: boolean }) => (
    <div className="relative">
      <button
        onClick={() => {
          setShowGroupPicker(!showGroupPicker);
          setShowDatePicker(false);
          setShowPriorityPicker(false);
        }}
        className={`flex items-center gap-1.5 ${isMobile ? 'px-3 py-2.5 min-h-[44px]' : 'px-2 py-1.5'} rounded-lg text-xs border transition-colors ${
          selectedGroup
            ? 'border-gray-200 dark:border-gray-700'
            : 'text-gray-400 border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800'
        }`}
      >
        <span
          className="w-3 h-3 rounded-full flex-shrink-0"
          style={{ backgroundColor: selectedGroup?.color_code || '#9CA3AF' }}
        />
        {selectedGroup && (
          <span className={`text-gray-700 dark:text-gray-300 truncate ${isMobile ? 'max-w-[80px]' : 'max-w-[60px] hidden sm:inline'}`}>
            {selectedGroup.name}
          </span>
        )}
      </button>

      {/* Group picker popover */}
      <AnimatePresence>
        {showGroupPicker && (
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 10 }}
            className="absolute bottom-full right-0 mb-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 overflow-hidden z-30"
          >
            <div className="py-1 max-h-40 overflow-y-auto">
              <button
                onClick={() => {
                  setGroupId('');
                  setShowGroupPicker(false);
                }}
                className={`w-full px-3 py-2.5 text-left text-xs hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center gap-2 ${
                  !groupId ? 'bg-gray-50 dark:bg-gray-700' : ''
                }`}
              >
                <span className="w-3 h-3 rounded-full bg-gray-300 dark:bg-gray-600" />
                <span className="text-gray-500 dark:text-gray-400">No group</span>
              </button>
              {groups.map((group) => (
                <button
                  key={group.id}
                  onClick={() => {
                    setGroupId(group.id);
                    setShowGroupPicker(false);
                  }}
                  className={`w-full px-3 py-2.5 text-left text-xs hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center gap-2 ${
                    group.id === groupId ? 'bg-gray-50 dark:bg-gray-700' : ''
                  }`}
                >
                  <span
                    className="w-3 h-3 rounded-full flex-shrink-0"
                    style={{ backgroundColor: group.color_code }}
                  />
                  <span className="text-gray-700 dark:text-gray-300 truncate">{group.name}</span>
                  {group.id === groupId && (
                    <Check size={12} weight="bold" className="text-primary-500 ml-auto flex-shrink-0" />
                  )}
                </button>
              ))}
            </div>

            {/* Create new group */}
            <div className="border-t border-gray-200 dark:border-gray-700 p-2">
              <div className="flex gap-1">
                <input
                  type="text"
                  value={newGroupName}
                  onChange={(e) => setNewGroupName(e.target.value)}
                  placeholder="New group"
                  className="flex-1 text-xs px-2 py-2 rounded border border-gray-200 dark:border-gray-600 bg-transparent text-gray-700 dark:text-gray-300"
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
                  className="px-3 py-2 text-xs bg-primary-600 text-white rounded hover:bg-primary-700 disabled:opacity-50"
                >
                  +
                </button>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );

  return (
    <div ref={containerRef} className="absolute bottom-0 left-0 right-0 z-20">
      <div className="max-w-4xl mx-auto px-3 sm:px-4 pb-4 sm:pb-6">
        <div className="bg-white/90 dark:bg-surface-dark-elevated/90 backdrop-blur-glass rounded-2xl shadow-float border border-gray-200/50 dark:border-gray-800/50">
          {/* Row 1: Input field */}
          <div className="flex items-center gap-2 p-3 pb-2 sm:pb-3">
            {/* Plus icon */}
            <div className="flex-shrink-0 w-8 h-8 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
              <Plus size={16} weight="bold" className="text-primary-600 dark:text-primary-400" />
            </div>

            {/* Text input with date highlighting */}
            <div className="flex-1 relative min-w-0">
              {/* Highlight overlay for detected dates */}
              <div className="absolute inset-0 pointer-events-none overflow-hidden whitespace-pre text-sm leading-[1.5]" style={{ padding: '0' }}>
                {textSegments.map((segment, idx) => (
                  <span
                    key={idx}
                    className={segment.isDate
                      ? 'bg-blue-100 dark:bg-blue-900/40 rounded-sm'
                      : ''
                    }
                    style={{ color: 'transparent' }}
                  >
                    {segment.text}
                  </span>
                ))}
              </div>
              <input
                ref={inputRef}
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Add a task..."
                className="w-full bg-transparent border-none outline-none text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 text-sm relative z-10 leading-[1.5]"
              />
            </div>

            {/* Desktop: buttons inline */}
            <div className="hidden sm:flex items-center gap-2">
              <DateButtonContent />
              <PriorityButtonContent />
              <GroupButtonContent />
              <button
                onClick={handleSubmit}
                disabled={loading || !title.trim()}
                className="px-4 py-1.5 bg-primary-600 text-white text-sm font-medium rounded-lg hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {loading ? '...' : 'Add'}
              </button>
            </div>
          </div>

          {/* Row 2: Mobile buttons */}
          <div className="flex sm:hidden items-center gap-2 px-3 pb-3">
            <DateButtonContent isMobile />
            <PriorityButtonContent isMobile />
            <GroupButtonContent isMobile />
            <button
              onClick={handleSubmit}
              disabled={loading || !title.trim()}
              className="flex-1 px-4 py-2.5 min-h-[44px] bg-primary-600 text-white text-sm font-medium rounded-lg hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {loading ? '...' : 'Add Task'}
            </button>
          </div>

          {/* Detected date indicator */}
          <AnimatePresence>
            {parsedDate?.date && (
              <motion.div
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: 'auto', opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                className="overflow-hidden"
              >
                <div className="px-3 pb-2 flex items-center gap-2 text-xs text-blue-600 dark:text-blue-400">
                  <CalendarBlank size={12} weight="regular" className="flex-shrink-0" />
                  <span>Due: {formatDateForDisplay(parsedDate.date)}</span>
                  <button
                    onClick={() => setDueDate('')}
                    className="ml-1 text-blue-400 hover:text-blue-600 dark:hover:text-blue-300"
                  >
                    <X size={12} weight="regular" />
                  </button>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>
    </div>
  );
}
