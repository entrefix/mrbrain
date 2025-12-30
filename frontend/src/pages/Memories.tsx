import { useState, useEffect, useMemo } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { toast } from 'react-hot-toast';
import {
  SquaresFour,
  List,
  MagnifyingGlass,
  X,
  BookmarkSimple,
  PaperPlaneTilt,
  ArrowSquareOut,
  Trash,
  CheckSquare,
  CaretDown,
  CaretRight,
  ArrowsClockwise,
  Globe,
  BookOpen,
  Coffee,
  FilmSlate,
  Book,
  Lightning,
  MapPin,
  ShoppingBag,
  Users,
  Trophy,
  ChatCircle,
  Folder,
} from '@phosphor-icons/react';
import { useAuth } from '../contexts/AuthContext';
import { memoryApi } from '../api';
import type {
  Memory,
  MemoryCategory,
  MemoryDigest,
  MemoryStats,
} from '../types';
import { formatDistanceToNow } from 'date-fns';
import { useDebounce } from '../hooks/useDebounce';

type ViewMode = 'timeline' | 'categories';

// Pending memory interface for optimistic UI
export interface PendingMemory {
  tempId: string;
  content: string;
}

const CATEGORY_ICONS: Record<string, React.ElementType> = {
  Websites: Globe,
  Food: Coffee,
  Movies: FilmSlate,
  Books: Book,
  Ideas: Lightning,
  Places: MapPin,
  Products: ShoppingBag,
  People: Users,
  Learnings: Trophy,
  Quotes: ChatCircle,
  Uncategorized: Folder,
};

export default function Memories() {
  const { user } = useAuth();
  const [memories, setMemories] = useState<Memory[]>([]);
  const [categories, setCategories] = useState<MemoryCategory[]>([]);
  const [stats, setStats] = useState<MemoryStats | null>(null);
  const [digest, setDigest] = useState<MemoryDigest | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>('timeline');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [inputValue, setInputValue] = useState('');
  const [pendingMemories, setPendingMemories] = useState<PendingMemory[]>([]);
  const [showDigest, setShowDigest] = useState(false);
  const [digestLoading, setDigestLoading] = useState(false);
  const [expandedCategory, setExpandedCategory] = useState<string | null>(null);
  const [isSearching, setIsSearching] = useState(false);
  const [urlPreviewMemory, setUrlPreviewMemory] = useState<Memory | null>(null);

  // Debounce search query for server-side search
  const debouncedSearchQuery = useDebounce(searchQuery, 300);

  useEffect(() => {
    if (user) {
      fetchData();
    }
  }, [user]);

  // Server-side search effect
  useEffect(() => {
    if (!user) return;

    // Skip during initial load
    if (loading) return;

    const performSearch = async () => {
      if (!debouncedSearchQuery.trim()) {
        // No search query - fetch all memories
        try {
          const memoriesData = await memoryApi.getAll();
          setMemories(memoriesData);
        } catch (error) {
          // Silently handle - data already loaded from initial fetch
        }
        return;
      }

      // Search via API for full-text search
      setIsSearching(true);
      try {
        const results = await memoryApi.search({
          query: debouncedSearchQuery.trim(),
          category: selectedCategory || undefined,
          limit: 100,
        });
        setMemories(results);
      } catch (error) {
        toast.error('Search failed');
      } finally {
        setIsSearching(false);
      }
    };

    performSearch();
  }, [debouncedSearchQuery, selectedCategory, user, loading]);

  const fetchData = async () => {
    try {
      const [memoriesData, categoriesData, statsData] = await Promise.all([
        memoryApi.getAll(),
        memoryApi.getCategories(),
        memoryApi.getStats(),
      ]);
      setMemories(memoriesData);
      setCategories(categoriesData);
      setStats(statsData);
    } catch (error) {
      toast.error('Failed to fetch memories');
    } finally {
      setLoading(false);
    }
  };

  const fetchDigest = async (regenerate = false) => {
    setDigestLoading(true);
    try {
      const digestData = regenerate
        ? await memoryApi.generateDigest()
        : await memoryApi.getDigest();
      setDigest(digestData);
    } catch (error) {
      toast.error('Failed to fetch digest');
    } finally {
      setDigestLoading(false);
    }
  };

  const handleCreateMemory = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim()) return;

    const tempId = `temp-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    const content = inputValue.trim();

    // 1. Clear input immediately (non-blocking UX)
    setInputValue('');

    // 2. Add to pending list (shows at top with loading indicator)
    setPendingMemories((prev) => [{ tempId, content }, ...prev]);

    try {
      const newMemory = await memoryApi.create({ content });
      // 3. Remove pending, add real memory at top
      setPendingMemories((prev) => prev.filter((p) => p.tempId !== tempId));
      setMemories((prev) => [newMemory, ...prev]);
      // Refresh stats in background (non-blocking)
      memoryApi.getStats().then(setStats);
    } catch (error) {
      // Remove failed pending memory
      setPendingMemories((prev) => prev.filter((p) => p.tempId !== tempId));
      toast.error('Failed to save memory');
    }
  };

  const handleDeleteMemory = async (id: string) => {
    try {
      await memoryApi.delete(id);
      setMemories((prev) => prev.filter((m) => m.id !== id));
      toast.success('Memory deleted');
      const statsData = await memoryApi.getStats();
      setStats(statsData);
    } catch (error) {
      toast.error('Failed to delete memory');
    }
  };

  const handleConvertToTodo = async (memory: Memory) => {
    try {
      await memoryApi.convertToTodo(memory.id);
      toast.success('Converted to todo!');
    } catch (error) {
      toast.error('Failed to convert to todo');
    }
  };

  // Filter memories based on search and category
  const filteredMemories = useMemo(() => {
    return memories.filter((memory) => {
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        const contentMatch = memory.content.toLowerCase().includes(query);
        const summaryMatch = memory.summary?.toLowerCase().includes(query);
        const urlTitleMatch = memory.url_title?.toLowerCase().includes(query);
        if (!contentMatch && !summaryMatch && !urlTitleMatch) return false;
      }
      if (selectedCategory && memory.category !== selectedCategory) {
        return false;
      }
      return true;
    });
  }, [memories, searchQuery, selectedCategory]);

  // Group memories by date for timeline view
  const groupedMemories = useMemo(() => {
    const groups: Record<string, Memory[]> = {
      Today: [],
      Yesterday: [],
      'This Week': [],
      Earlier: [],
    };

    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const yesterday = new Date(today.getTime() - 24 * 60 * 60 * 1000);
    const weekAgo = new Date(today.getTime() - 7 * 24 * 60 * 60 * 1000);

    filteredMemories.forEach((memory) => {
      const createdAt = new Date(memory.created_at);
      if (createdAt >= today) {
        groups['Today'].push(memory);
      } else if (createdAt >= yesterday) {
        groups['Yesterday'].push(memory);
      } else if (createdAt >= weekAgo) {
        groups['This Week'].push(memory);
      } else {
        groups['Earlier'].push(memory);
      }
    });

    return groups;
  }, [filteredMemories]);

  // Group memories by category for category view
  const memoriesByCategory = useMemo(() => {
    const grouped: Record<string, Memory[]> = {};
    filteredMemories.forEach((memory) => {
      const cat = memory.category || 'Uncategorized';
      if (!grouped[cat]) {
        grouped[cat] = [];
      }
      grouped[cat].push(memory);
    });
    return grouped;
  }, [filteredMemories]);

  const getCategoryColor = (categoryName: string): string => {
    const category = categories.find((c) => c.name === categoryName);
    return category?.color_code || '#6B7280';
  };

  const getCategoryIcon = (categoryName: string) => {
    return CATEGORY_ICONS[categoryName] || Folder;
  };

  const renderMemoryCard = (memory: Memory) => {
    const CategoryIcon = getCategoryIcon(memory.category);
    const categoryColor = getCategoryColor(memory.category);

    return (
      <motion.div
        key={memory.id}
        layout
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -20 }}
        className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-4 hover:shadow-md transition-shadow"
      >
        <div className="flex items-start justify-between gap-3">
          <div className="flex-1 min-w-0">
            {/* Category badge */}
            <div className="flex items-center gap-2 mb-2">
              <span
                className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full"
                style={{
                  backgroundColor: `${categoryColor}20`,
                  color: categoryColor,
                }}
              >
                <CategoryIcon className="w-3 h-3" />
                {memory.category}
              </span>
              <span className="text-xs text-gray-400">
                {formatDistanceToNow(new Date(memory.created_at), {
                  addSuffix: true,
                })}
              </span>
            </div>

            {/* Content */}
            <p className="text-gray-900 dark:text-white text-sm leading-relaxed">
              {memory.content}
            </p>

            {/* Summary */}
            {memory.summary && (
              <p className="mt-2 text-xs text-gray-500 dark:text-gray-400 italic">
                {memory.summary}
              </p>
            )}

            {/* URL preview - clickable to show modal */}
            {memory.url && (
              <button
                onClick={() => setUrlPreviewMemory(memory)}
                className="mt-3 w-full flex items-center gap-2 p-2 bg-gray-50 dark:bg-gray-800 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors text-left"
              >
                <ArrowSquareOut size={16} weight="regular" className="text-gray-400 flex-shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="text-xs font-medium text-gray-700 dark:text-gray-300 truncate">
                    {memory.url_title || memory.url}
                  </p>
                  {memory.url_content && (
                    <p className="text-xs text-gray-500 dark:text-gray-400 truncate">
                      {memory.url_content}
                    </p>
                  )}
                </div>
              </button>
            )}
          </div>

          {/* Actions */}
          <div className="flex flex-col gap-1">
            <button
              onClick={() => handleConvertToTodo(memory)}
              className="p-1.5 text-gray-400 hover:text-primary-600 dark:hover:text-primary-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded transition-colors"
              title="Convert to todo"
            >
              <CheckSquare size={16} weight="regular" />
            </button>
            <button
              onClick={() => handleDeleteMemory(memory.id)}
              className="p-1.5 text-gray-400 hover:text-red-600 dark:hover:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded transition-colors"
              title="Delete"
            >
              <Trash size={16} weight="regular" />
            </button>
          </div>
        </div>
      </motion.div>
    );
  };

  return (
    <div className="h-full flex flex-col bg-surface-light dark:bg-surface-dark relative">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="flex-1 overflow-y-auto px-4 sm:px-6 py-6 pb-32"
      >
        <div className="max-w-4xl mx-auto">
          {/* Header */}
          <div className="mb-6 pl-10 lg:pl-0">
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
              Memories
            </h1>
        <p className="mt-2 text-gray-600 dark:text-gray-400">
          Capture ideas, links, and thoughts - AI organizes them for you
        </p>
      </div>

      {/* Stats */}
      {stats && (
        <div className="grid grid-cols-3 gap-4 mb-6">
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
            <p className="text-2xl font-bold text-gray-900 dark:text-white">
              {stats.total}
            </p>
            <p className="text-xs text-gray-500">Total Memories</p>
          </div>
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
            <p className="text-2xl font-bold text-gray-900 dark:text-white">
              {stats.this_week}
            </p>
            <p className="text-xs text-gray-500">This Week</p>
          </div>
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
            <p className="text-2xl font-bold text-gray-900 dark:text-white">
              {Object.keys(stats.by_category).length}
            </p>
            <p className="text-xs text-gray-500">Categories</p>
          </div>
        </div>
      )}

      {/* Weekly Digest */}
      <div className="mb-6">
        <button
          onClick={() => {
            if (!showDigest && !digest) {
              fetchDigest();
            }
            setShowDigest(!showDigest);
          }}
          className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-primary-600 dark:hover:text-primary-400"
        >
          <BookOpen size={16} weight="regular" />
          Weekly Digest
          {showDigest ? (
            <CaretDown size={16} weight="bold" />
          ) : (
            <CaretRight size={16} weight="bold" />
          )}
        </button>

        <AnimatePresence>
          {showDigest && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: 'auto', opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              className="overflow-hidden"
            >
              <div className="mt-3 bg-gradient-to-br from-primary-50 to-purple-50 dark:from-primary-900/20 dark:to-purple-900/20 rounded-lg border border-primary-200 dark:border-primary-800 p-4">
                {digestLoading ? (
                  <div className="flex items-center justify-center py-8">
                    <div className="animate-spin rounded-full h-6 w-6 border-t-2 border-b-2 border-primary-600"></div>
                  </div>
                ) : digest ? (
                  <>
                    <div className="flex items-center justify-between mb-3">
                      <span className="text-xs text-gray-500">
                        {digest.week_start} - {digest.week_end}
                      </span>
                      <button
                        onClick={() => fetchDigest(true)}
                        className="text-xs text-primary-600 hover:text-primary-700 flex items-center gap-1"
                      >
                        <ArrowsClockwise size={12} weight="regular" />
                        Regenerate
                      </button>
                    </div>
                    <p className="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap">
                      {digest.digest_content}
                    </p>
                  </>
                ) : (
                  <p className="text-sm text-gray-500 text-center py-4">
                    No digest available yet. Add some memories first!
                  </p>
                )}
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Search and View Toggle */}
      <div className="flex items-center gap-3 mb-4">
        <div className="flex-1 relative">
          {isSearching ? (
            <div className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 border-2 border-primary-500 border-t-transparent rounded-full animate-spin" />
          ) : (
            <MagnifyingGlass size={16} weight="regular" className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          )}
          <input
            type="text"
            placeholder="Search memories..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-9 pr-8 py-2 text-sm border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery('')}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
            >
              <X size={16} weight="regular" />
            </button>
          )}
        </div>

        <div className="flex items-center bg-surface-light-muted dark:bg-surface-dark-muted rounded-xl p-1">
          <button
            onClick={() => setViewMode('timeline')}
            className={`p-2 rounded-lg transition-all duration-200 ${
              viewMode === 'timeline'
                ? 'bg-white dark:bg-surface-dark-elevated shadow-subtle text-primary-600'
                : 'text-gray-500 hover:text-gray-700'
            }`}
            title="Timeline view"
          >
            <List size={16} weight="regular" />
          </button>
          <button
            onClick={() => setViewMode('categories')}
            className={`p-2 rounded-lg transition-all duration-200 ${
              viewMode === 'categories'
                ? 'bg-white dark:bg-surface-dark-elevated shadow-subtle text-primary-600'
                : 'text-gray-500 hover:text-gray-700'
            }`}
            title="Category view"
          >
            <SquaresFour size={16} weight="regular" />
          </button>
        </div>
      </div>

      {/* Category Filter Chips */}
      <div className="flex flex-wrap gap-2 mb-6">
        <button
          onClick={() => setSelectedCategory(null)}
          className={`px-3 py-1.5 text-xs font-medium rounded-full transition-colors ${
            selectedCategory === null
              ? 'bg-primary-600 text-white'
              : 'bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700'
          }`}
        >
          All
        </button>
        {categories
          .filter((c) => stats?.by_category[c.name])
          .map((category) => {
            const Icon = getCategoryIcon(category.name);
            return (
              <button
                key={category.id}
                onClick={() =>
                  setSelectedCategory(
                    selectedCategory === category.name ? null : category.name
                  )
                }
                className={`inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-full transition-colors ${
                  selectedCategory === category.name
                    ? 'text-white'
                    : 'hover:opacity-80'
                }`}
                style={{
                  backgroundColor:
                    selectedCategory === category.name
                      ? category.color_code
                      : `${category.color_code}20`,
                  color:
                    selectedCategory === category.name
                      ? 'white'
                      : category.color_code,
                }}
              >
                <Icon className="w-3 h-3" />
                {category.name}
                <span className="ml-1 opacity-70">
                  ({stats?.by_category[category.name] || 0})
                </span>
              </button>
            );
          })}
      </div>

      {/* Content */}
      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-600"></div>
        </div>
      ) : memories.length === 0 && pendingMemories.length === 0 ? (
        <div className="bg-surface-light-muted dark:bg-surface-dark-muted rounded-2xl p-8 text-center">
          <div className="max-w-md mx-auto">
            <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
              <BookmarkSimple size={32} weight="regular" className="text-primary-600 dark:text-primary-400" />
            </div>
            <h3 className="text-lg font-heading text-gray-900 dark:text-white mb-2">
              No memories yet
            </h3>
            <p className="text-gray-500 dark:text-gray-400 text-sm">
              Start capturing your thoughts, links, and ideas below. AI will
              automatically organize them into categories!
            </p>
          </div>
        </div>
      ) : viewMode === 'timeline' ? (
        /* Timeline View */
        <div className="space-y-6">
          {/* Pending Memories - show at top with loading indicator */}
          {pendingMemories.length > 0 && (
            <div>
              <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">
                Processing...
              </h3>
              <div className="space-y-3">
                {pendingMemories.map((pm) => (
                  <motion.div
                    key={pm.tempId}
                    initial={{ opacity: 0, y: -20 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-4 opacity-70"
                  >
                    <div className="flex items-center gap-2 mb-2">
                      <div className="w-3 h-3 border-2 border-primary-500 border-t-transparent rounded-full animate-spin" />
                      <span className="text-xs text-gray-400">AI is categorizing...</span>
                    </div>
                    <p className="text-gray-600 dark:text-gray-300 text-sm">{pm.content}</p>
                  </motion.div>
                ))}
              </div>
            </div>
          )}

          {Object.entries(groupedMemories).map(
            ([period, periodMemories]) =>
              periodMemories.length > 0 && (
                <div key={period}>
                  <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">
                    {period}
                  </h3>
                  <div className="space-y-3">
                    <AnimatePresence>
                      {periodMemories.map(renderMemoryCard)}
                    </AnimatePresence>
                  </div>
                </div>
              )
          )}
        </div>
      ) : (
        /* Category View */
        <div className="space-y-4">
          {Object.entries(memoriesByCategory).map(
            ([categoryName, categoryMemories]) => {
              const Icon = getCategoryIcon(categoryName);
              const color = getCategoryColor(categoryName);
              const isExpanded = expandedCategory === categoryName;

              return (
                <div
                  key={categoryName}
                  className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden"
                >
                  <button
                    onClick={() =>
                      setExpandedCategory(isExpanded ? null : categoryName)
                    }
                    className="w-full flex items-center justify-between p-4 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      <div
                        className="w-10 h-10 rounded-lg flex items-center justify-center"
                        style={{ backgroundColor: `${color}20` }}
                      >
                        <Icon className="w-5 h-5" style={{ color }} />
                      </div>
                      <div className="text-left">
                        <p className="font-medium text-gray-900 dark:text-white">
                          {categoryName}
                        </p>
                        <p className="text-xs text-gray-500">
                          {categoryMemories.length} memories
                        </p>
                      </div>
                    </div>
                    {isExpanded ? (
                      <CaretDown size={20} weight="bold" className="text-gray-400" />
                    ) : (
                      <CaretRight size={20} weight="bold" className="text-gray-400" />
                    )}
                  </button>

                  <AnimatePresence>
                    {isExpanded && (
                      <motion.div
                        initial={{ height: 0, opacity: 0 }}
                        animate={{ height: 'auto', opacity: 1 }}
                        exit={{ height: 0, opacity: 0 }}
                        className="overflow-hidden"
                      >
                        <div className="p-4 pt-0 space-y-3">
                          {categoryMemories.map(renderMemoryCard)}
                        </div>
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>
              );
            }
          )}
        </div>
      )}

      {/* URL Preview Modal */}
      <AnimatePresence>
        {urlPreviewMemory && urlPreviewMemory.url && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4"
            onClick={() => setUrlPreviewMemory(null)}
          >
            <motion.div
              initial={{ scale: 0.95, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.95, opacity: 0 }}
              className="bg-white dark:bg-gray-900 rounded-xl shadow-xl max-w-lg w-full max-h-[80vh] overflow-hidden"
              onClick={(e) => e.stopPropagation()}
            >
              {/* Header */}
              <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
                <div className="flex items-center gap-2">
                  <Globe size={20} weight="regular" className="text-primary-600" />
                  <h3 className="font-heading text-gray-900 dark:text-white">
                    URL Preview
                  </h3>
                </div>
                <button
                  onClick={() => setUrlPreviewMemory(null)}
                  className="p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
                >
                  <X size={20} weight="regular" />
                </button>
              </div>

              {/* Content */}
              <div className="p-4 space-y-4 overflow-y-auto max-h-[60vh]">
                {/* URL Title */}
                <div>
                  <p className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">
                    Title
                  </p>
                  <p className="text-gray-900 dark:text-white">
                    {urlPreviewMemory.url_title || 'No title available'}
                  </p>
                </div>

                {/* URL */}
                <div>
                  <p className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">
                    URL
                  </p>
                  <p className="text-primary-600 dark:text-primary-400 text-sm break-all">
                    {urlPreviewMemory.url}
                  </p>
                </div>

                {/* AI Summary */}
                {urlPreviewMemory.url_content && (
                  <div>
                    <p className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">
                      AI Summary
                    </p>
                    <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-3">
                      <p className="text-gray-700 dark:text-gray-300 text-sm leading-relaxed">
                        {urlPreviewMemory.url_content}
                      </p>
                    </div>
                  </div>
                )}

                {/* Original Memory */}
                <div>
                  <p className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">
                    Your Note
                  </p>
                  <p className="text-gray-600 dark:text-gray-400 text-sm italic">
                    "{urlPreviewMemory.content}"
                  </p>
                </div>
              </div>

              {/* Footer */}
              <div className="flex items-center justify-end gap-3 p-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
                <button
                  onClick={() => setUrlPreviewMemory(null)}
                  className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                >
                  Close
                </button>
                <a
                  href={urlPreviewMemory.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 rounded-xl transition-colors"
                >
                  <ArrowSquareOut size={16} weight="regular" />
                  Open URL
                </a>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

        </div>
      </motion.div>

      {/* Quick Capture Input */}
      <div className="absolute bottom-0 left-0 right-0 z-20">
        <div className="max-w-4xl mx-auto px-3 sm:px-4 pb-4 sm:pb-6">
          <form
            onSubmit={handleCreateMemory}
            className="bg-white/90 dark:bg-surface-dark-elevated/90 backdrop-blur-glass rounded-2xl shadow-float border border-gray-200/50 dark:border-gray-800/50"
          >
            <div className="flex items-center gap-2 p-3 pb-2 sm:pb-3">
              {/* Icon */}
              <div className="flex-shrink-0 w-8 h-8 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
                <PaperPlaneTilt size={16} weight="bold" className="text-primary-600 dark:text-primary-400" />
              </div>

              {/* Text input */}
              <div className="flex-1 relative min-w-0">
                <input
                  type="text"
                  value={inputValue}
                  onChange={(e) => setInputValue(e.target.value)}
                  placeholder="Capture a memory... (e.g., Found a great coffee shop on Main St)"
                  className="w-full bg-transparent border-none outline-none text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 text-sm relative z-10 leading-[1.5]"
                />
              </div>

              {/* Submit button */}
              <button
                type="submit"
                disabled={!inputValue.trim()}
                className="px-4 py-1.5 bg-primary-600 text-white text-sm font-medium rounded-xl hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                Add
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
