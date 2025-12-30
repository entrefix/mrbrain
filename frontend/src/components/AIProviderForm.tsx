import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { FiX, FiCheck, FiLoader, FiAlertCircle } from 'react-icons/fi';
import type { AIProvider, AIProviderCreate, TestConnectionResponse } from '../api/aiProviders';
import { DEFAULT_BASE_URLS, PROVIDER_LABELS } from '../api/aiProviders';
import { aiProviderApi } from '../api';

interface AIProviderFormProps {
  provider?: AIProvider | null;
  onSubmit: (data: AIProviderCreate) => Promise<void>;
  onCancel: () => void;
}

type ProviderType = 'openai' | 'anthropic' | 'google' | 'custom';

export default function AIProviderForm({ provider, onSubmit, onCancel }: AIProviderFormProps) {
  const isEditing = !!provider;

  const [name, setName] = useState(provider?.name || '');
  const [providerType, setProviderType] = useState<ProviderType>(provider?.provider_type || 'openai');
  const [baseUrl, setBaseUrl] = useState(provider?.base_url || DEFAULT_BASE_URLS.openai);
  const [apiKey, setApiKey] = useState('');
  const [isDefault, setIsDefault] = useState(provider?.is_default || false);

  const [loading, setLoading] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<TestConnectionResponse | null>(null);
  const [error, setError] = useState('');

  // Update base URL when provider type changes
  useEffect(() => {
    if (!isEditing) {
      setBaseUrl(DEFAULT_BASE_URLS[providerType] || '');
    }
  }, [providerType, isEditing]);

  // Auto-generate name based on provider type
  useEffect(() => {
    if (!isEditing && !name) {
      setName(PROVIDER_LABELS[providerType] || 'Custom Provider');
    }
  }, [providerType, isEditing, name]);

  const handleTest = async () => {
    if (!baseUrl || !apiKey) {
      setError('Base URL and API key are required');
      return;
    }

    setTesting(true);
    setTestResult(null);
    setError('');

    try {
      const result = await aiProviderApi.testConnection({
        provider_type: providerType,
        base_url: baseUrl,
        api_key: apiKey,
      });
      setTestResult(result);
    } catch (err) {
      setError('Failed to test connection');
    } finally {
      setTesting(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!name || !baseUrl || (!isEditing && !apiKey)) {
      setError('Please fill in all required fields');
      return;
    }

    setLoading(true);
    setError('');

    try {
      await onSubmit({
        name,
        provider_type: providerType,
        base_url: baseUrl,
        api_key: apiKey,
        is_default: isDefault,
      });
    } catch (err) {
      setError('Failed to save provider');
    } finally {
      setLoading(false);
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="fixed inset-0 z-50 overflow-y-auto"
    >
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="fixed inset-0 bg-black/50" onClick={onCancel} />

        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 0.95 }}
          className="relative w-full max-w-lg bg-white dark:bg-gray-800 rounded-xl shadow-xl"
        >
          {/* Header */}
          <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
              {isEditing ? 'Edit AI Provider' : 'Add AI Provider'}
            </h2>
            <button
              onClick={onCancel}
              className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700"
            >
              <FiX size={20} />
            </button>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit} className="p-6 space-y-4">
            {/* Provider Type */}
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Provider Type
              </label>
              <div className="grid grid-cols-2 gap-2">
                {(['openai', 'anthropic', 'google', 'custom'] as const).map((type) => (
                  <button
                    key={type}
                    type="button"
                    onClick={() => setProviderType(type)}
                    className={`px-4 py-2 text-sm rounded-lg border transition-colors ${
                      providerType === type
                        ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300'
                        : 'border-gray-200 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700'
                    }`}
                  >
                    {PROVIDER_LABELS[type]}
                  </button>
                ))}
              </div>
            </div>

            {/* Name */}
            <div>
              <label htmlFor="name" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Display Name
              </label>
              <input
                type="text"
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="My OpenAI Account"
                className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
            </div>

            {/* Base URL */}
            <div>
              <label htmlFor="baseUrl" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Base URL
              </label>
              <input
                type="url"
                id="baseUrl"
                value={baseUrl}
                onChange={(e) => setBaseUrl(e.target.value)}
                placeholder="https://api.openai.com/v1"
                className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 focus:ring-2 focus:ring-primary-500 focus:border-transparent font-mono text-sm"
              />
            </div>

            {/* API Key */}
            <div>
              <label htmlFor="apiKey" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                API Key {isEditing && <span className="text-gray-400">(leave blank to keep current)</span>}
              </label>
              <input
                type="password"
                id="apiKey"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder={isEditing ? '********' : 'sk-...'}
                required={!isEditing}
                className="w-full px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 focus:ring-2 focus:ring-primary-500 focus:border-transparent font-mono text-sm"
              />
            </div>

            {/* Default checkbox */}
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="isDefault"
                checked={isDefault}
                onChange={(e) => setIsDefault(e.target.checked)}
                className="w-4 h-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              <label htmlFor="isDefault" className="text-sm text-gray-700 dark:text-gray-300">
                Set as default provider
              </label>
            </div>

            {/* Test Connection */}
            <div className="pt-2">
              <button
                type="button"
                onClick={handleTest}
                disabled={testing || !baseUrl || !apiKey}
                className="w-full px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center justify-center gap-2"
              >
                {testing ? (
                  <>
                    <FiLoader className="animate-spin" size={16} />
                    Testing connection...
                  </>
                ) : (
                  'Test Connection'
                )}
              </button>

              {testResult && (
                <div
                  className={`mt-2 p-3 rounded-lg text-sm ${
                    testResult.success
                      ? 'bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300'
                      : 'bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    {testResult.success ? <FiCheck /> : <FiAlertCircle />}
                    <span>{testResult.message}</span>
                  </div>
                  {testResult.success && testResult.models && testResult.models.length > 0 && (
                    <div className="mt-2 text-xs">
                      <span className="font-medium">Available models:</span>{' '}
                      {testResult.models.slice(0, 5).join(', ')}
                      {testResult.models.length > 5 && ` +${testResult.models.length - 5} more`}
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Error */}
            {error && (
              <div className="p-3 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300 text-sm flex items-center gap-2">
                <FiAlertCircle />
                {error}
              </div>
            )}

            {/* Actions */}
            <div className="flex gap-3 pt-4">
              <button
                type="button"
                onClick={onCancel}
                className="flex-1 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={loading}
                className="flex-1 px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center justify-center gap-2"
              >
                {loading ? (
                  <>
                    <FiLoader className="animate-spin" size={16} />
                    Saving...
                  </>
                ) : (
                  isEditing ? 'Save Changes' : 'Add Provider'
                )}
              </button>
            </div>
          </form>
        </motion.div>
      </div>
    </motion.div>
  );
}
