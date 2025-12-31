import { useState, useEffect, useMemo } from 'react';
import { AnimatePresence } from 'framer-motion';
import { toast } from 'react-hot-toast';
import { Gear, Plus, Cpu, Trash, Database } from '@phosphor-icons/react';
import { useAuth } from '../contexts/AuthContext';
import { memoryApi, todoApi, ragApi, aiProviderApi, userDataApi } from '../api';
import type { Memory, Todo, RAGAskResponse, RAGSearchResult } from '../types';
import type { AIProvider, AIProviderModel, AIProviderCreate, DataStats } from '../api';
import UnifiedGrid from '../components/UnifiedGrid';
import CreateNoteModal from '../components/CreateNoteModal';
import NoteDetailModal from '../components/NoteDetailModal';
import ImportModal from '../components/ImportModal';
import ProfileDropdown from '../components/ProfileDropdown';
import PillSwitch from '../components/PillSwitch';
import AskAIResults from '../components/AskAIResults';
import AIProviderCard from '../components/AIProviderCard';
import AIProviderForm from '../components/AIProviderForm';
import DeleteConfirmationModal from '../components/DeleteConfirmationModal';

interface Message {
  id: string;
  type: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  isLoading?: boolean;
  sources?: RAGSearchResult[];
}

type ActiveTab = 'mems' | 'todos';

// Citation type for grid display
type CitationItem = {
  document: RAGSearchResult['document'];
  isCitation: boolean;
};

// Pending item type for optimistic UI
type PendingMemory = Memory & { isProcessing?: boolean };
type PendingTodo = Todo & { isProcessing?: boolean };

export default function Unified() {
  const { user } = useAuth();
  const [activeTab, setActiveTab] = useState<ActiveTab>('mems');
  const [showSettings, setShowSettings] = useState(false);
  
  // Data states
  const [memories, setMemories] = useState<(Memory | PendingMemory)[]>([]);
  const [todos, setTodos] = useState<(Todo | PendingTodo)[]>([]);
  const [memoryCitations, setMemoryCitations] = useState<CitationItem[]>([]);
  const [todoCitations, setTodoCitations] = useState<CitationItem[]>([]);
  
  // UI states
  const [loading, setLoading] = useState(true);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false);
  const [isImportModalOpen, setIsImportModalOpen] = useState(false);
  const [selectedItem, setSelectedItem] = useState<Memory | Todo | CitationItem | null>(null);
  
  // Ask AI states
  const [askMessages, setAskMessages] = useState<Message[]>([]);
  const [askInputValue, setAskInputValue] = useState('');
  const [isAskLoading, setIsAskLoading] = useState(false);

  // Load initial data
  useEffect(() => {
    if (user) {
      fetchData();
    }
  }, [user]);

  const fetchData = async () => {
    try {
      const [memoriesData, todosData] = await Promise.all([
        memoryApi.getAll(),
        todoApi.getAll(),
      ]);
      setMemories(memoriesData);
      setTodos(todosData);
    } catch (error) {
      toast.error('Failed to fetch data');
    } finally {
      setLoading(false);
    }
  };

  // Get items to display based on active tab
  // Only show citations if they exist, otherwise show all memories
  const displayItems = useMemo(() => {
    if (activeTab === 'mems') {
      // If there are citations, only show citations (clear grid)
      if (memoryCitations.length > 0) {
        return memoryCitations;
      }
      // Otherwise show all memories
      return memories;
    } else {
      // If there are citations, only show citations (clear grid)
      if (todoCitations.length > 0) {
        return todoCitations;
      }
      // Otherwise show all todos
      return todos;
    }
  }, [activeTab, memories, todos, memoryCitations, todoCitations]);

  // Handle create memory
  const handleCreateNote = async (title: string, content: string) => {
    // Close modal immediately and show processing indicator
    setIsCreateModalOpen(false);
    
    // Create temporary pending item for optimistic UI
    const tempId = `temp-${Date.now()}`;
    
    try {
      if (activeTab === 'mems') {
        // For memories, use content only - backend generates summary automatically
        const memoryContent = content.trim();
        if (!memoryContent) {
          toast.error('Content is required');
          return;
        }
        
        // Auto-generate summary from content (first line, truncated)
        const autoSummary = memoryContent.split('\n')[0].trim();
        const summary = autoSummary.length > 100 ? autoSummary.substring(0, 100) + '...' : autoSummary;
        
        const pendingItem: PendingMemory = {
          id: tempId,
          user_id: user?.id || '',
          content: memoryContent,
          summary: summary || null,
          category: 'Uncategorized',
          url: null,
          url_title: null,
          url_content: null,
          is_archived: false,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          isProcessing: true,
        };
        
        // Add to grid with processing indicator
        setMemories((prev) => [pendingItem, ...prev]);
        
        // Create via API - only send content, backend will generate summary
        const newMemory = await memoryApi.create({ content: memoryContent });
        
        // Replace pending with real item
        setMemories((prev) => prev.map((m) => m.id === tempId ? newMemory : m));
        toast.success('Memory created');
      } else {
        // For todos, use title as title and content as description
        const todoTitle = title.trim() || content.split('\n')[0].substring(0, 50) || 'Untitled Todo';
        const pendingItem: PendingTodo = {
          id: tempId,
          user_id: user?.id || '',
          title: todoTitle,
          description: content.trim() || null,
          due_date: null,
          priority: 'medium' as const,
          status: 'pending' as const,
          position: '0',
          tags: [],
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          group_id: null,
          isProcessing: true,
        };
        
        // Add to grid with processing indicator
        setTodos((prev) => [...prev, pendingItem]);
        
        // Create via API
        const newTodo = await todoApi.create({
          title: todoTitle,
          description: content.trim() || null,
        });
        
        // Replace pending with real item
        setTodos((prev) => prev.map((t) => t.id === tempId ? newTodo : t));
        toast.success('Todo created');
      }
    } catch (error) {
      // Remove failed pending item
      if (activeTab === 'mems') {
        setMemories((prev) => prev.filter((m) => m.id !== tempId));
      } else {
        setTodos((prev) => prev.filter((t) => t.id !== tempId));
      }
      toast.error('Failed to create item');
      throw error;
    }
  };

  // Handle update item
  const handleUpdateItem = async (id: string, data: any) => {
    try {
      if (activeTab === 'mems') {
        // Check if it's a citation (read-only)
        const isCitation = memoryCitations.some(c => c.document.id === id);
        if (isCitation) {
          toast.error('Citations are read-only');
          return;
        }
        const updated = await memoryApi.update(id, data);
        setMemories((prev) => prev.map((m) => (m.id === id ? updated : m)));
      } else {
        // Check if it's a citation (read-only)
        const isCitation = todoCitations.some(c => c.document.id === id);
        if (isCitation) {
          toast.error('Citations are read-only');
          return;
        }
        const updated = await todoApi.update(id, data);
        setTodos((prev) => prev.map((t) => (t.id === id ? updated : t)));
      }
    } catch (error) {
      toast.error('Failed to update item');
      throw error;
    }
  };

  // Handle delete item
  const handleDeleteItem = async (id: string) => {
    try {
      if (activeTab === 'mems') {
        // Check if it's a citation (can remove from citations)
        const isCitation = memoryCitations.some(c => c.document.id === id);
        if (isCitation) {
          setMemoryCitations((prev) => prev.filter((c) => c.document.id !== id));
          toast.success('Citation removed');
          return;
        }
        await memoryApi.delete(id);
        setMemories((prev) => prev.filter((m) => m.id !== id));
        toast.success('Memory deleted');
      } else {
        // Check if it's a citation (can remove from citations)
        const isCitation = todoCitations.some(c => c.document.id === id);
        if (isCitation) {
          setTodoCitations((prev) => prev.filter((c) => c.document.id !== id));
          toast.success('Citation removed');
          return;
        }
        await todoApi.delete(id);
        setTodos((prev) => prev.filter((t) => t.id !== id));
        toast.success('Todo deleted');
      }
    } catch (error) {
      toast.error('Failed to delete item');
      throw error;
    }
  };

  // Handle convert memory to todo
  const handleConvertMemory = async (memory: Memory) => {
    try {
      await memoryApi.convertToTodo(memory.id);
      toast.success('Converted to todo!');
      setMemories((prev) => prev.filter((m) => m.id !== memory.id));
      await fetchData(); // Refresh todos
    } catch (error) {
      toast.error('Failed to convert');
    }
  };

  // Handle import
  const handleImportComplete = (importedMemories: Memory[]) => {
    setMemories((prev) => [...importedMemories, ...prev]);
    toast.success(`Imported ${importedMemories.length} memories`);
  };

  // Submit query to Ask AI
  const submitQuery = async (query: string, skipUserMessage: boolean = false) => {
    // Add user message (unless regenerating)
    if (!skipUserMessage) {
      const userMessage: Message = {
        id: Date.now().toString(),
        type: 'user',
        content: query,
        timestamp: new Date(),
      };
      setAskMessages((prev) => [...prev, userMessage]);
    }

    // Add loading message
    const loadingId = (Date.now() + 1).toString();
    setAskMessages((prev) => [
      ...prev,
      {
        id: loadingId,
        type: 'assistant',
        content: '',
        timestamp: new Date(),
        isLoading: true,
      },
    ]);
    setIsAskLoading(true);

    try {
      const response: RAGAskResponse = await ragApi.ask({
        question: query,
        max_context: 5,
      });

      // Update loading message with response
      setAskMessages((prev) =>
        prev.map((msg) =>
          msg.id === loadingId
            ? {
                ...msg,
                content: response.answer,
                sources: response.sources,
                isLoading: false,
              }
            : msg
        )
      );

      // Extract and split citations - CLEAR completely before adding new ones
      // Clear existing citations first
      setMemoryCitations([]);
      setTodoCitations([]);
      
      // Only add citations if sources exist
      if (response.sources && response.sources.length > 0) {
        const memoryCitations = response.sources
          .filter((s) => s.document.content_type === 'memory')
          .map((s) => ({ document: s.document, isCitation: true }));
        
        const todoCitations = response.sources
          .filter((s) => s.document.content_type === 'todo')
          .map((s) => ({ document: s.document, isCitation: true }));

        // Set new citations (grid will be cleared and only show these)
        setMemoryCitations(memoryCitations);
        setTodoCitations(todoCitations);
      }
    } catch (error) {
      console.error('Ask AI error:', error);
      toast.error('Failed to process your question');
      setAskMessages((prev) => prev.filter((msg) => msg.id !== loadingId));
    } finally {
      setIsAskLoading(false);
    }
  };

  // Handle Ask AI submit
  const handleAskSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!askInputValue.trim() || isAskLoading) return;

    const query = askInputValue.trim();
    setAskInputValue('');
    await submitQuery(query);
  };

  // Handle regenerate - resubmit last query
  const handleRegenerateQuery = (lastQuery: string) => {
    // Remove the last assistant message (the one we're regenerating)
    setAskMessages((prev) => {
      const newMessages = [...prev];
      // Find and remove the last assistant message
      for (let i = newMessages.length - 1; i >= 0; i--) {
        if (newMessages[i].type === 'assistant' && !newMessages[i].isLoading) {
          newMessages.splice(i, 1);
          break;
        }
      }
      return newMessages;
    });
    
    // Resubmit the query without adding a new user message
    submitQuery(lastQuery, true);
  };

  // Handle item click
  const handleItemClick = (item: Memory | Todo | { document: RAGSearchResult['document']; isCitation?: boolean } | PendingMemory | PendingTodo) => {
    // Don't open detail modal for pending items
    if ('isProcessing' in item && item.isProcessing) {
      return;
    }
    // Normalize citation items - ensure isCitation is always boolean
    const normalizedItem: Memory | Todo | CitationItem = 'document' in item
      ? { document: item.document, isCitation: item.isCitation ?? true }
      : item as Memory | Todo;
    setSelectedItem(normalizedItem);
    setIsDetailModalOpen(true);
  };

  // Get logo - using text for now
  const Logo = () => (
    <button
      onClick={() => {
        setActiveTab('mems');
        setShowSettings(false);
        // Clear citations when navigating to mems
        setMemoryCitations([]);
        setTodoCitations([]);
      }}
      className="flex items-center gap-2 hover:opacity-80 transition-opacity cursor-pointer"
    >
      <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-primary-400 to-primary-600 flex items-center justify-center">
        <span className="text-white text-sm font-bold">M</span>
      </div>
      <span className="text-xl font-heading text-gray-800 dark:text-white hidden sm:inline">Mr. Brain</span>
    </button>
  );

  // Settings Content Component
  const SettingsContent = () => {
    const [providers, setProviders] = useState<AIProvider[]>([]);
    const [providerModels, setProviderModels] = useState<Record<string, AIProviderModel[]>>({});
    const [loadingProviders, setLoadingProviders] = useState(true);
    const [showProviderForm, setShowProviderForm] = useState(false);
    const [editingProvider, setEditingProvider] = useState<AIProvider | null>(null);
    const [refreshingModels, setRefreshingModels] = useState<string | null>(null);
    const [dataStats, setDataStats] = useState<DataStats | null>(null);
    const [loadingStats, setLoadingStats] = useState(false);
    const [showClearMemoriesModal, setShowClearMemoriesModal] = useState(false);
    const [showClearAllModal, setShowClearAllModal] = useState(false);

    useEffect(() => {
      fetchProviders();
      fetchDataStats();
    }, []);

    const fetchProviders = async () => {
      try {
        const data = await aiProviderApi.getAll();
        setProviders(data);
        for (const provider of data) {
          try {
            const models = await aiProviderApi.getModels(provider.id);
            setProviderModels(prev => ({ ...prev, [provider.id]: models }));
          } catch {}
        }
      } catch (error) {
        toast.error('Failed to load AI providers');
      } finally {
        setLoadingProviders(false);
      }
    };

    const fetchDataStats = async () => {
      setLoadingStats(true);
      try {
        const stats = await userDataApi.getStats();
        setDataStats(stats);
      } catch (error) {
        console.error('Failed to fetch data stats:', error);
      } finally {
        setLoadingStats(false);
      }
    };

    const handleCreateProvider = async (data: AIProviderCreate) => {
      try {
        const newProvider = await aiProviderApi.create(data);
        setProviders(prev => [...prev, newProvider]);
        setShowProviderForm(false);
        toast.success('AI provider added successfully');
        try {
          const models = await aiProviderApi.fetchModels(newProvider.id);
          setProviderModels(prev => ({ ...prev, [newProvider.id]: models }));
        } catch {}
      } catch (error) {
        toast.error('Failed to add AI provider');
        throw error;
      }
    };

    const handleUpdateProvider = async (data: AIProviderCreate) => {
      if (!editingProvider) return;
      try {
        const updated = await aiProviderApi.update(editingProvider.id, {
          name: data.name,
          base_url: data.base_url,
          api_key: data.api_key || undefined,
          is_default: data.is_default,
        });
        setProviders(prev => prev.map(p => p.id === updated.id ? updated : p));
        setEditingProvider(null);
        toast.success('AI provider updated successfully');
      } catch (error) {
        toast.error('Failed to update AI provider');
        throw error;
      }
    };

    const handleDeleteProvider = async (id: string) => {
      try {
        await aiProviderApi.delete(id);
        setProviders(prev => prev.filter(p => p.id !== id));
        toast.success('AI provider deleted');
      } catch (error) {
        toast.error('Failed to delete AI provider');
      }
    };

    const handleSetDefault = async (id: string) => {
      try {
        const updated = await aiProviderApi.update(id, { is_default: true });
        setProviders(prev => prev.map(p => ({
          ...p,
          is_default: p.id === updated.id,
        })));
        toast.success('Default provider updated');
      } catch (error) {
        toast.error('Failed to set default provider');
      }
    };

    const handleSelectModel = async (providerId: string, modelId: string) => {
      try {
        const updated = await aiProviderApi.update(providerId, { selected_model: modelId });
        setProviders(prev => prev.map(p => p.id === updated.id ? updated : p));
        toast.success('Model selected');
      } catch (error) {
        toast.error('Failed to select model');
      }
    };

    const handleRefreshModels = async (providerId: string) => {
      setRefreshingModels(providerId);
      try {
        const models = await aiProviderApi.fetchModels(providerId);
        setProviderModels(prev => ({ ...prev, [providerId]: models }));
        toast.success(`Found ${models.length} models`);
      } catch (error) {
        toast.error('Failed to fetch models');
      } finally {
        setRefreshingModels(null);
      }
    };

    const handleClearMemories = async () => {
      try {
        const result = await userDataApi.clearMemories();
        toast.success(`Deleted ${result.memories_deleted} memories`);
        await fetchDataStats();
        // Refresh memories in parent component
        const memoriesData = await memoryApi.getAll();
        setMemories(memoriesData);
      } catch (error) {
        toast.error('Failed to clear memories');
        throw error;
      }
    };

    const handleClearAllData = async () => {
      try {
        const result = await userDataApi.clearAll();
        toast.success(
          `Deleted ${result.memories_deleted} memories, ${result.todos_deleted} todos, and ${result.custom_groups_deleted} groups`
        );
        await fetchDataStats();
        // Refresh data in parent component
        const [memoriesData, todosData] = await Promise.all([
          memoryApi.getAll(),
          todoApi.getAll(),
        ]);
        setMemories(memoriesData);
        setTodos(todosData);
      } catch (error) {
        toast.error('Failed to clear all data');
        throw error;
      }
    };

    return (
      <>
        <div className="max-w-2xl mx-auto">
          <div className="mb-8">
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Settings</h1>
            <p className="mt-2 text-gray-600 dark:text-gray-400">
              Manage your account preferences and settings
            </p>
          </div>

          <div className="space-y-6">
            {/* AI Providers Section */}
            <div className="bg-surface-light-muted dark:bg-surface-dark-muted rounded-2xl overflow-hidden">
              <div className="p-6">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-xl bg-secondary-100 dark:bg-secondary-900/30 flex items-center justify-center">
                      <Cpu size={20} weight="regular" className="text-secondary-600 dark:text-secondary-400" />
                    </div>
                    <div>
                      <h2 className="text-lg font-heading text-gray-900 dark:text-white">
                        AI Providers
                      </h2>
                      <p className="text-sm text-gray-500 dark:text-gray-400">
                        Configure AI services for todo summarization
                      </p>
                    </div>
                  </div>
                  <button
                    onClick={() => setShowProviderForm(true)}
                    className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-xl hover:bg-primary-700 transition-colors"
                  >
                    <Plus size={16} weight="bold" />
                    Add Provider
                  </button>
                </div>

                {loadingProviders ? (
                  <div className="py-8 text-center text-gray-500 dark:text-gray-400">
                    Loading providers...
                  </div>
                ) : providers.length === 0 ? (
                  <div className="py-8 text-center">
                    <p className="text-gray-500 dark:text-gray-400 mb-4">
                      No AI providers configured yet.
                    </p>
                    <p className="text-sm text-gray-400 dark:text-gray-500">
                      Add an AI provider to enable automatic todo summarization and tagging.
                    </p>
                  </div>
                ) : (
                  <div className="space-y-4">
                    {providers.map(provider => (
                      <AIProviderCard
                        key={provider.id}
                        provider={provider}
                        models={providerModels[provider.id] || []}
                        onEdit={() => setEditingProvider(provider)}
                        onDelete={() => handleDeleteProvider(provider.id)}
                        onSetDefault={() => handleSetDefault(provider.id)}
                        onSelectModel={(modelId) => handleSelectModel(provider.id, modelId)}
                        onRefreshModels={() => handleRefreshModels(provider.id)}
                        refreshingModels={refreshingModels === provider.id}
                      />
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* Data Management */}
            <div className="bg-surface-light-muted dark:bg-surface-dark-muted rounded-2xl overflow-hidden">
              <div className="p-6">
                <div className="flex items-center gap-3 mb-6">
                  <div className="w-10 h-10 rounded-xl bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
                    <Database size={20} weight="regular" className="text-red-600 dark:text-red-400" />
                  </div>
                  <div>
                    <h2 className="text-lg font-heading text-gray-900 dark:text-white">
                      Data Management
                    </h2>
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      Manage and delete your stored data
                    </p>
                  </div>
                </div>

                {loadingStats ? (
                  <div className="py-8 text-center text-gray-500 dark:text-gray-400">
                    Loading data statistics...
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div className="p-4 bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700">
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <h3 className="font-medium text-gray-900 dark:text-white mb-1">
                            Clear All Memories
                          </h3>
                          <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                            Delete all your saved memories and their embeddings
                          </p>
                          {dataStats && (
                            <p className="text-xs text-gray-500 dark:text-gray-500">
                              {dataStats.memory_count} {dataStats.memory_count === 1 ? 'memory' : 'memories'} stored
                            </p>
                          )}
                        </div>
                        <button
                          onClick={() => setShowClearMemoriesModal(true)}
                          disabled={!dataStats || dataStats.memory_count === 0}
                          className="px-4 py-2 text-sm font-medium text-red-600 dark:text-red-400 border border-red-300 dark:border-red-800 rounded-xl hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                        >
                          <Trash size={16} weight="regular" />
                          Clear Memories
                        </button>
                      </div>
                    </div>

                    <div className="p-4 bg-white dark:bg-gray-800 rounded-xl border border-red-300 dark:border-red-800">
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <h3 className="font-medium text-red-600 dark:text-red-400 mb-1">
                            Clear All Data
                          </h3>
                          <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                            Delete all todos, memories, and custom groups. AI providers and default groups will be kept.
                          </p>
                          {dataStats && (
                            <div className="text-xs text-gray-500 dark:text-gray-500 space-y-0.5">
                              <p>{dataStats.memory_count} memories</p>
                              <p>{dataStats.todo_count} todos</p>
                              <p>{dataStats.custom_group_count} custom groups</p>
                            </div>
                          )}
                        </div>
                        <button
                          onClick={() => setShowClearAllModal(true)}
                          disabled={!dataStats || (dataStats.memory_count === 0 && dataStats.todo_count === 0 && dataStats.custom_group_count === 0)}
                          className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-xl hover:bg-red-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                        >
                          <Trash size={16} weight="bold" />
                          Clear All
                        </button>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>

          </div>
        </div>

        {/* Provider Form Modal */}
        <AnimatePresence>
          {(showProviderForm || editingProvider) && (
            <AIProviderForm
              provider={editingProvider}
              onSubmit={editingProvider ? handleUpdateProvider : handleCreateProvider}
              onCancel={() => {
                setShowProviderForm(false);
                setEditingProvider(null);
              }}
            />
          )}
        </AnimatePresence>

        {/* Clear Memories Confirmation Modal */}
        {dataStats && (
          <DeleteConfirmationModal
            isOpen={showClearMemoriesModal}
            onClose={() => setShowClearMemoriesModal(false)}
            onConfirm={handleClearMemories}
            title="Clear All Memories"
            message="This will permanently delete all your memories and their vector embeddings. This action cannot be undone."
            stats={[
              { label: 'Memories', count: dataStats.memory_count },
            ]}
          />
        )}

        {/* Clear All Data Confirmation Modal */}
        {dataStats && (
          <DeleteConfirmationModal
            isOpen={showClearAllModal}
            onClose={() => setShowClearAllModal(false)}
            onConfirm={handleClearAllData}
            title="Clear All Data"
            message="This will permanently delete all your todos, memories, and custom groups. AI providers and default groups will be preserved. This action cannot be undone."
            stats={[
              { label: 'Memories', count: dataStats.memory_count },
              { label: 'Todos', count: dataStats.todo_count },
              { label: 'Custom Groups', count: dataStats.custom_group_count },
            ]}
          />
        )}
      </>
    );
  };

  return (
    <div className="h-full w-full flex flex-col bg-surface-light dark:bg-surface-dark relative">
      {/* Header - Translucent, hovering over grid (no border, no shadow, seamless blend with grid) */}
      <div className="sticky top-0 z-20 bg-surface-light/60 dark:bg-surface-dark/60 backdrop-blur-md px-4 sm:px-6 py-3">
        <div className="flex items-center justify-between">
          <Logo />
          
          {/* Centered Pill Switch - Always visible, but unhighlighted when settings is active */}
          <div className="absolute left-1/2 transform -translate-x-1/2">
            <PillSwitch 
              activeTab={activeTab} 
              onTabChange={(tab) => {
                setActiveTab(tab);
                setShowSettings(false); // Close settings when switching tabs
              }}
              isSettingsActive={showSettings}
            />
          </div>
          
          <div className="flex items-center gap-4">
            <button
              onClick={() => {
                setShowSettings(!showSettings);
                if (!showSettings) {
                  // Clear citations when opening settings
                  setMemoryCitations([]);
                  setTodoCitations([]);
                }
              }}
              className={`p-2 rounded-lg transition-colors ${
                showSettings
                  ? 'bg-primary-100 dark:bg-primary-900/40 text-primary-600 dark:text-primary-400'
                  : 'hover:bg-gray-100/80 dark:hover:bg-gray-800/80 text-gray-600 dark:text-gray-400'
              }`}
              title="Settings"
            >
              <Gear size={20} weight="regular" />
            </button>
            
            <ProfileDropdown />
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 overflow-hidden flex flex-col relative">
        {showSettings ? (
          /* Settings View */
          <div className="flex-1 overflow-y-auto px-4 sm:px-6 pt-6 pb-6">
            <SettingsContent />
          </div>
        ) : (
          /* Grid Area - with subtle top padding so initial elements aren't hidden, but can scroll under header */
          <div className="flex-1 overflow-y-auto pt-6 pb-32">
            {loading ? (
              <div className="flex items-center justify-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-600"></div>
              </div>
            ) : (
              <UnifiedGrid
                items={displayItems}
                type={activeTab === 'mems' ? 'memory' : 'todo'}
                onCreateClick={() => setIsCreateModalOpen(true)}
                onImportClick={() => setIsImportModalOpen(true)}
                onItemClick={handleItemClick}
              />
            )}
          </div>
        )}

        {/* Ask AI Area - Floating over grid (grouped expand + query, stack effect on grid) - Hidden when settings is active */}
        {!showSettings && (
          <div className="absolute bottom-0 left-0 right-0 z-10">
            <AskAIResults
              messages={askMessages}
              isLoading={isAskLoading}
              inputValue={askInputValue}
              onInputChange={setAskInputValue}
              onSubmit={handleAskSubmit}
              onRegenerate={handleRegenerateQuery}
            />
          </div>
        )}
      </div>

      {/* Modals */}
      <CreateNoteModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onSubmit={handleCreateNote}
        type={activeTab === 'mems' ? 'memory' : 'todo'}
      />

      <NoteDetailModal
        isOpen={isDetailModalOpen}
        onClose={() => {
          setIsDetailModalOpen(false);
          setSelectedItem(null);
        }}
        item={selectedItem}
        type={activeTab === 'mems' ? 'memory' : 'todo'}
        onUpdate={handleUpdateItem}
        onDelete={handleDeleteItem}
        onConvert={activeTab === 'mems' ? handleConvertMemory : undefined}
      />

      <ImportModal
        isOpen={isImportModalOpen}
        onClose={() => setIsImportModalOpen(false)}
        onImportComplete={handleImportComplete}
      />
    </div>
  );
}

