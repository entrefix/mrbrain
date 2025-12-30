import { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { toast } from 'react-hot-toast';
import { useTheme } from '../contexts/ThemeContext';
import { Sun, Moon, Plus, Cpu } from '@phosphor-icons/react';
import { aiProviderApi } from '../api';
import type { AIProvider, AIProviderModel, AIProviderCreate } from '../api/aiProviders';
import AIProviderCard from '../components/AIProviderCard';
import AIProviderForm from '../components/AIProviderForm';

export default function Settings() {
  const { theme, toggleTheme } = useTheme();
  const [emailNotifications, setEmailNotifications] = useState(true);
  const [pushNotifications, setPushNotifications] = useState(true);

  // AI Provider state
  const [providers, setProviders] = useState<AIProvider[]>([]);
  const [providerModels, setProviderModels] = useState<Record<string, AIProviderModel[]>>({});
  const [loadingProviders, setLoadingProviders] = useState(true);
  const [showProviderForm, setShowProviderForm] = useState(false);
  const [editingProvider, setEditingProvider] = useState<AIProvider | null>(null);
  const [refreshingModels, setRefreshingModels] = useState<string | null>(null);

  // Fetch providers on mount
  useEffect(() => {
    fetchProviders();
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

        {/* Theme & Notifications */}
        <div className="bg-surface-light-muted dark:bg-surface-dark-muted rounded-2xl">
          <div className="p-6 space-y-6">
            <div>
              <h2 className="text-lg font-heading text-gray-900 dark:text-white mb-4">
                Theme Preferences
              </h2>
              <button
                onClick={toggleTheme}
                className="flex items-center px-4 py-3 rounded-xl text-gray-600 dark:text-gray-300 hover:bg-surface-light dark:hover:bg-surface-dark-elevated transition-all duration-200"
              >
                {theme === 'dark' ? (
                  <Sun size={20} weight="regular" className="mr-3" />
                ) : (
                  <Moon size={20} weight="regular" className="mr-3" />
                )}
                {theme === 'dark' ? 'Light Mode' : 'Dark Mode'}
              </button>
            </div>

            <div>
              <h2 className="text-lg font-heading text-gray-900 dark:text-white mb-4">
                Notifications
              </h2>
              <div className="space-y-4">
                <label className="flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={emailNotifications}
                    onChange={(e) => setEmailNotifications(e.target.checked)}
                    className="w-5 h-5 rounded-lg border-gray-300 text-primary-600 focus:ring-primary-500 focus:ring-offset-0"
                  />
                  <span className="ml-3 text-gray-700 dark:text-gray-300">
                    Email Notifications
                  </span>
                </label>
                <label className="flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={pushNotifications}
                    onChange={(e) => setPushNotifications(e.target.checked)}
                    className="w-5 h-5 rounded-lg border-gray-300 text-primary-600 focus:ring-primary-500 focus:ring-offset-0"
                  />
                  <span className="ml-3 text-gray-700 dark:text-gray-300">
                    Push Notifications
                  </span>
                </label>
              </div>
            </div>
          </div>

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
    </div>
  );
}
