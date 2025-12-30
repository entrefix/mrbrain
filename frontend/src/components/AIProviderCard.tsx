import { useState } from 'react';
import { motion } from 'framer-motion';
import { FiEdit2, FiTrash2, FiCheck, FiX, FiStar, FiRefreshCw, FiChevronDown } from 'react-icons/fi';
import { SiOpenai, SiGoogle } from 'react-icons/si';
import type { AIProvider, AIProviderModel } from '../api/aiProviders';
import { PROVIDER_LABELS } from '../api/aiProviders';

interface AIProviderCardProps {
  provider: AIProvider;
  models: AIProviderModel[];
  onEdit: () => void;
  onDelete: () => void;
  onSetDefault: () => void;
  onSelectModel: (modelId: string) => void;
  onRefreshModels: () => void;
  refreshingModels: boolean;
}

export default function AIProviderCard({
  provider,
  models,
  onEdit,
  onDelete,
  onSetDefault,
  onSelectModel,
  onRefreshModels,
  refreshingModels,
}: AIProviderCardProps) {
  const [showModels, setShowModels] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const getProviderIcon = () => {
    switch (provider.provider_type) {
      case 'openai':
        return <SiOpenai className="text-green-500" size={20} />;
      case 'anthropic':
        return (
          <div className="w-5 h-5 rounded bg-[#D4A27F] flex items-center justify-center text-white text-xs font-bold">
            A
          </div>
        );
      case 'google':
        return <SiGoogle className="text-blue-500" size={20} />;
      default:
        return (
          <div className="w-5 h-5 rounded bg-gray-500 flex items-center justify-center text-white text-xs font-bold">
            C
          </div>
        );
    }
  };

  return (
    <motion.div
      layout
      className={`bg-white dark:bg-gray-800 rounded-lg border ${
        provider.is_default
          ? 'border-primary-500 ring-1 ring-primary-500'
          : 'border-gray-200 dark:border-gray-700'
      } overflow-hidden`}
    >
      {/* Header */}
      <div className="p-4">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            {getProviderIcon()}
            <div>
              <div className="flex items-center gap-2">
                <h3 className="font-medium text-gray-900 dark:text-white">{provider.name}</h3>
                {provider.is_default && (
                  <span className="px-2 py-0.5 text-xs bg-primary-100 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400 rounded-full">
                    Default
                  </span>
                )}
                {!provider.is_enabled && (
                  <span className="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-500 rounded-full">
                    Disabled
                  </span>
                )}
              </div>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                {PROVIDER_LABELS[provider.provider_type]}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-1">
            {!provider.is_default && (
              <button
                onClick={onSetDefault}
                className="p-2 text-gray-400 hover:text-yellow-500 hover:bg-yellow-50 dark:hover:bg-yellow-900/20 rounded-lg transition-colors"
                title="Set as default"
              >
                <FiStar size={16} />
              </button>
            )}
            <button
              onClick={onEdit}
              className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
              title="Edit"
            >
              <FiEdit2 size={16} />
            </button>
            {confirmDelete ? (
              <div className="flex items-center gap-1">
                <button
                  onClick={onDelete}
                  className="p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
                  title="Confirm delete"
                >
                  <FiCheck size={16} />
                </button>
                <button
                  onClick={() => setConfirmDelete(false)}
                  className="p-2 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                  title="Cancel"
                >
                  <FiX size={16} />
                </button>
              </div>
            ) : (
              <button
                onClick={() => setConfirmDelete(true)}
                className="p-2 text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
                title="Delete"
              >
                <FiTrash2 size={16} />
              </button>
            )}
          </div>
        </div>

        {/* Info */}
        <div className="mt-3 space-y-1 text-sm text-gray-500 dark:text-gray-400">
          <p className="truncate">
            <span className="text-gray-400">URL:</span> {provider.base_url}
          </p>
          <p>
            <span className="text-gray-400">API Key:</span> {provider.api_key_masked}
          </p>
          {provider.selected_model && (
            <p>
              <span className="text-gray-400">Model:</span>{' '}
              <span className="text-gray-700 dark:text-gray-300">{provider.selected_model}</span>
            </p>
          )}
        </div>
      </div>

      {/* Model selection */}
      <div className="border-t border-gray-100 dark:border-gray-700">
        <button
          onClick={() => setShowModels(!showModels)}
          className="w-full px-4 py-3 flex items-center justify-between text-sm text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
        >
          <span>
            {models.length > 0
              ? `${models.length} models available`
              : 'Click to fetch models'}
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={(e) => {
                e.stopPropagation();
                onRefreshModels();
              }}
              disabled={refreshingModels}
              className={`p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-600 ${
                refreshingModels ? 'animate-spin' : ''
              }`}
              title="Refresh models"
            >
              <FiRefreshCw size={14} />
            </button>
            <FiChevronDown
              className={`transition-transform ${showModels ? 'rotate-180' : ''}`}
            />
          </div>
        </button>

        {showModels && (
          <div className="px-4 pb-4">
            {models.length === 0 ? (
              <p className="text-sm text-gray-500 dark:text-gray-400 py-2">
                No models cached. Click refresh to fetch available models.
              </p>
            ) : (
              <div className="max-h-48 overflow-y-auto space-y-1">
                {models.map((model) => (
                  <button
                    key={model.id}
                    onClick={() => onSelectModel(model.model_id)}
                    className={`w-full px-3 py-2 text-left text-sm rounded-lg transition-colors ${
                      provider.selected_model === model.model_id
                        ? 'bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300'
                        : 'hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300'
                    }`}
                  >
                    <span className="font-mono text-xs">{model.model_name}</span>
                    {provider.selected_model === model.model_id && (
                      <FiCheck className="inline ml-2 text-primary-500" size={14} />
                    )}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </motion.div>
  );
}
