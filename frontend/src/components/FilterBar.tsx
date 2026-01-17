import { useState, useRef, useEffect } from 'react';
import { FiSearch, FiX, FiFilter, FiChevronDown, FiCalendar, FiTag, FiFolder } from 'react-icons/fi';
import { motion, AnimatePresence } from 'framer-motion';
import type { Group } from '../types';

export interface TodoFilters {
  searchText: string;
  tags: string[];
  groupId: string | null;
  dateRange: {
    from: string | null;
    to: string | null;
  };
}

interface FilterBarProps {
  filters: TodoFilters;
  onFiltersChange: (filters: TodoFilters) => void;
  availableTags: string[];
  groups: Group[];
}

export default function FilterBar({
  filters,
  onFiltersChange,
  availableTags,
  groups,
}: FilterBarProps) {
  const [showFilters, setShowFilters] = useState(false);
  const [showTagDropdown, setShowTagDropdown] = useState(false);
  const [showGroupDropdown, setShowGroupDropdown] = useState(false);
  const filterRef = useRef<HTMLDivElement>(null);

  // Close dropdowns when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (filterRef.current && !filterRef.current.contains(event.target as Node)) {
        setShowTagDropdown(false);
        setShowGroupDropdown(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const activeFilterCount = [
    filters.tags.length > 0,
    filters.groupId !== null,
    filters.dateRange.from !== null || filters.dateRange.to !== null,
  ].filter(Boolean).length;

  const handleTagToggle = (tag: string) => {
    const newTags = filters.tags.includes(tag)
      ? filters.tags.filter(t => t !== tag)
      : [...filters.tags, tag];
    onFiltersChange({ ...filters, tags: newTags });
  };

  const handleGroupSelect = (groupId: string | null) => {
    onFiltersChange({ ...filters, groupId });
    setShowGroupDropdown(false);
  };

  const handleDateChange = (field: 'from' | 'to', value: string) => {
    onFiltersChange({
      ...filters,
      dateRange: { ...filters.dateRange, [field]: value || null },
    });
  };

  const clearFilters = () => {
    onFiltersChange({
      searchText: '',
      tags: [],
      groupId: null,
      dateRange: { from: null, to: null },
    });
  };

  const hasActiveFilters = filters.searchText || filters.tags.length > 0 ||
    filters.groupId || filters.dateRange.from || filters.dateRange.to;

  const selectedGroup = groups.find(g => g.id === filters.groupId);

  return (
    <div ref={filterRef} className="mb-4">
      {/* Search and Filter Toggle Row */}
      <div className="flex items-center gap-2">
        {/* Search Input */}
        <div className="flex-1 relative">
          <FiSearch className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 w-4 h-4" />
          <input
            type="text"
            value={filters.searchText}
            onChange={(e) => onFiltersChange({ ...filters, searchText: e.target.value })}
            placeholder="Search todos..."
            className="w-full pl-9 pr-8 py-2 text-sm rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-black text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          />
          {filters.searchText && (
            <button
              onClick={() => onFiltersChange({ ...filters, searchText: '' })}
              className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
            >
              <FiX className="w-4 h-4" />
            </button>
          )}
        </div>

        {/* Filter Toggle Button */}
        <button
          onClick={() => setShowFilters(!showFilters)}
          className={`flex items-center gap-1.5 px-3 py-2 text-sm rounded-lg border transition-colors ${
            showFilters || activeFilterCount > 0
              ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20 text-primary-600 dark:text-primary-400'
              : 'border-gray-200 dark:border-gray-700 bg-white dark:bg-black text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-900'
          }`}
        >
          <FiFilter className="w-4 h-4" />
          <span>Filters</span>
          {activeFilterCount > 0 && (
            <span className="ml-1 px-1.5 py-0.5 text-xs rounded-full bg-primary-500 text-white">
              {activeFilterCount}
            </span>
          )}
        </button>

        {/* Clear Filters */}
        {hasActiveFilters && (
          <button
            onClick={clearFilters}
            className="px-3 py-2 text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
          >
            Clear
          </button>
        )}
      </div>

      {/* Expanded Filters */}
      <AnimatePresence>
        {showFilters && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            transition={{ duration: 0.15 }}
          >
            <div className="mt-3 p-3 bg-gray-50 dark:bg-gray-900/50 rounded-lg border border-gray-200 dark:border-gray-800">
              <div className="flex flex-wrap gap-3">
                {/* Tags Filter */}
                <div className="relative">
                  <button
                    onClick={() => {
                      setShowTagDropdown(!showTagDropdown);
                      setShowGroupDropdown(false);
                    }}
                    className="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-black hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                  >
                    <FiTag className="w-4 h-4 text-gray-400" />
                    <span className="text-gray-700 dark:text-gray-300">
                      {filters.tags.length > 0 ? `${filters.tags.length} tags` : 'Tags'}
                    </span>
                    <FiChevronDown className="w-4 h-4 text-gray-400" />
                  </button>

                  <AnimatePresence>
                    {showTagDropdown && (
                      <motion.div
                        initial={{ opacity: 0, y: -10 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, y: -10 }}
                        className="absolute top-full left-0 mt-1 w-56 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 z-50"
                      >
                        <div className="p-2 max-h-48 overflow-y-auto">
                          {availableTags.length === 0 ? (
                            <p className="text-sm text-gray-500 dark:text-gray-400 px-2 py-1">
                              No tags available
                            </p>
                          ) : (
                            availableTags.map(tag => (
                              <label
                                key={tag}
                                className="flex items-center gap-2 px-2 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-700 rounded cursor-pointer"
                              >
                                <input
                                  type="checkbox"
                                  checked={filters.tags.includes(tag)}
                                  onChange={() => handleTagToggle(tag)}
                                  className="w-4 h-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                                />
                                <span className="text-sm text-gray-700 dark:text-gray-300">
                                  #{tag}
                                </span>
                              </label>
                            ))
                          )}
                        </div>
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>

                {/* Group Filter */}
                <div className="relative">
                  <button
                    onClick={() => {
                      setShowGroupDropdown(!showGroupDropdown);
                      setShowTagDropdown(false);
                    }}
                    className="flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-black hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                  >
                    {selectedGroup ? (
                      <>
                        <span
                          className="w-3 h-3 rounded-full"
                          style={{ backgroundColor: selectedGroup.color_code }}
                        />
                        <span className="text-gray-700 dark:text-gray-300">
                          {selectedGroup.name}
                        </span>
                      </>
                    ) : (
                      <>
                        <FiFolder className="w-4 h-4 text-gray-400" />
                        <span className="text-gray-700 dark:text-gray-300">Group</span>
                      </>
                    )}
                    <FiChevronDown className="w-4 h-4 text-gray-400" />
                  </button>

                  <AnimatePresence>
                    {showGroupDropdown && (
                      <motion.div
                        initial={{ opacity: 0, y: -10 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, y: -10 }}
                        className="absolute top-full left-0 mt-1 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 z-50"
                      >
                        <div className="py-1">
                          <button
                            onClick={() => handleGroupSelect(null)}
                            className={`w-full px-4 py-2 text-left text-sm hover:bg-gray-50 dark:hover:bg-gray-700 ${
                              !filters.groupId ? 'bg-gray-50 dark:bg-gray-700' : ''
                            }`}
                          >
                            <span className="text-gray-600 dark:text-gray-400">All groups</span>
                          </button>
                          {groups.map(group => (
                            <button
                              key={group.id}
                              onClick={() => handleGroupSelect(group.id)}
                              className={`w-full px-4 py-2 text-left text-sm hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center gap-2 ${
                                filters.groupId === group.id ? 'bg-gray-50 dark:bg-gray-700' : ''
                              }`}
                            >
                              <span
                                className="w-3 h-3 rounded-full"
                                style={{ backgroundColor: group.color_code }}
                              />
                              <span className="text-gray-700 dark:text-gray-300">{group.name}</span>
                            </button>
                          ))}
                        </div>
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>

                {/* Date Range Filter */}
                <div className="flex items-center gap-2">
                  <FiCalendar className="w-4 h-4 text-gray-400" />
                  <input
                    type="date"
                    value={filters.dateRange.from || ''}
                    onChange={(e) => handleDateChange('from', e.target.value)}
                    className="px-2 py-1.5 text-sm rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-black text-gray-700 dark:text-gray-300"
                    placeholder="From"
                  />
                  <span className="text-gray-400">to</span>
                  <input
                    type="date"
                    value={filters.dateRange.to || ''}
                    onChange={(e) => handleDateChange('to', e.target.value)}
                    className="px-2 py-1.5 text-sm rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-black text-gray-700 dark:text-gray-300"
                    placeholder="To"
                  />
                </div>
              </div>

              {/* Active Filter Tags */}
              {(filters.tags.length > 0 || filters.groupId || filters.dateRange.from || filters.dateRange.to) && (
                <div className="mt-3 flex flex-wrap gap-2">
                  {filters.tags.map(tag => (
                    <span
                      key={tag}
                      className="inline-flex items-center gap-1 px-2 py-1 text-xs rounded-full bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400"
                    >
                      #{tag}
                      <button
                        onClick={() => handleTagToggle(tag)}
                        className="hover:text-purple-800 dark:hover:text-purple-200"
                      >
                        <FiX className="w-3 h-3" />
                      </button>
                    </span>
                  ))}
                  {filters.groupId && selectedGroup && (
                    <span
                      className="inline-flex items-center gap-1 px-2 py-1 text-xs rounded-full"
                      style={{
                        backgroundColor: selectedGroup.color_code + '20',
                        color: selectedGroup.color_code,
                      }}
                    >
                      {selectedGroup.name}
                      <button
                        onClick={() => handleGroupSelect(null)}
                        className="hover:opacity-70"
                      >
                        <FiX className="w-3 h-3" />
                      </button>
                    </span>
                  )}
                  {(filters.dateRange.from || filters.dateRange.to) && (
                    <span className="inline-flex items-center gap-1 px-2 py-1 text-xs rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400">
                      {filters.dateRange.from || '...'} - {filters.dateRange.to || '...'}
                      <button
                        onClick={() => onFiltersChange({ ...filters, dateRange: { from: null, to: null } })}
                        className="hover:text-blue-800 dark:hover:text-blue-200"
                      >
                        <FiX className="w-3 h-3" />
                      </button>
                    </span>
                  )}
                </div>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
