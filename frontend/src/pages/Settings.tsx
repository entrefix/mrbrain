import { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { toast } from 'react-hot-toast';
import { useTheme } from '../contexts/ThemeContext';
import { Sun, Moon, Plus, Cpu, Trash, Database } from '@phosphor-icons/react';
import { aiProviderApi, userDataApi } from '../api';
import type { AIProvider, AIProviderModel, AIProviderCreate, DataStats } from '../api';
import AIProviderCard from '../components/AIProviderCard';
import AIProviderForm from '../components/AIProviderForm';
import DeleteConfirmationModal from '../components/DeleteConfirmationModal';

export default function Settings() {
  const [emailNotifications, setEmailNotifications] = useState(true);
  const [pushNotifications, setPushNotifications] = useState(true);

  // AI Provider state
  const [providers, setProviders] = useState<AIProvider[]>([]);
  const [providerModels, setProviderModels] = useState<Record<string, AIProviderModel[]>>({});
  const [loadingProviders, setLoadingProviders] = useState(true);
  const [showProviderForm, setShowProviderForm] = useState(false);
  const [editingProvider, setEditingProvider] = useState<AIProvider | null>(null);
  const [refreshingModels, setRefreshingModels] = useState<string | null>(null);

  // Data Management state
  const [dataStats, setDataStats] = useState<DataStats | null>(null);
  const [loadingStats, setLoadingStats] = useState(false);
  const [showClearMemoriesModal, setShowClearMemoriesModal] = useState(false);
  const [showClearAllModal, setShowClearAllModal] = useState(false);

  // Fetch providers on mount
  useEffect(() => {
    fetchProviders();
    fetchDataStats();
  }, []);

  const fetchProviders = async () => {
    try {
      const data = await aiProviderApi.getAll();
      setProviders(data);

      // Fetch models for each provider
      for (const provider of data) {
        try {
          const models = await aiProviderApi.getModels(provider.id);
          setProviderModels(prev => ({ ...prev, [provider.id]: models }));
        } catch {
          // Ignore errors for individual providers
        }
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

      // Fetch models for the new provider
      try {
        const models = await aiProviderApi.fetchModels(newProvider.id);
        setProviderModels(prev => ({ ...prev, [newProvider.id]: models }));
      } catch {
        // Ignore if model fetch fails
      }
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
    } catch (error) {
      toast.error('Failed to clear all data');
      throw error;
    }
  };

  const handleSaveSettings = () => {
    toast.success('Settings saved successfully');
  };

  return (
    <div className="h-full flex flex-col bg-surface-light dark:bg-surface-dark relative">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="flex-1 overflow-y-auto px-4 sm:px-6 py-6"
      >
        <div className="max-w-2xl mx-auto">
          <div className="mb-8 pl-10 lg:pl-0">
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
                {/* Clear All Memories */}
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

                {/* Clear All Data */}
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

        {/* Theme & Notifications */}
        <div className="bg-surface-light-muted dark:bg-surface-dark-muted rounded-2xl">

          <div className="px-6 py-4 bg-surface-light dark:bg-surface-dark-elevated rounded-b-2xl">
            <button
              onClick={handleSaveSettings}
              className="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-xl text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 transition-colors"
            >
              Save Settings
            </button>
          </div>
        </div>
        </div>
        </div>
      </motion.div>

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
    </div>
  );
}
