import { useState, useRef, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { toast } from 'react-hot-toast';
import {
  PaperPlaneTilt,
  MagnifyingGlass,
  ChatCircle,
  CheckSquare,
  BookmarkSimple,
  CircleNotch,
  Info,
  CaretDown,
  CaretUp,
  Warning,
} from '@phosphor-icons/react';
import { ragApi } from '../api';
import type { RAGSearchResult, RAGAskResponse } from '../types';

interface Message {
  id: string;
  type: 'user' | 'assistant' | 'search-results';
  content: string;
  sources?: RAGSearchResult[];
  timestamp: Date;
  isLoading?: boolean;
}

type Mode = 'ask' | 'search';

export default function Chat() {
  // Separate message histories for each mode
  const [askMessages, setAskMessages] = useState<Message[]>([]);
  const [searchMessages, setSearchMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [mode, setMode] = useState<Mode>('ask');
  const [showSources, setShowSources] = useState<Record<string, boolean>>({});
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Get current messages based on mode
  const messages = mode === 'ask' ? askMessages : searchMessages;
  const setMessages = mode === 'ask' ? setAskMessages : setSearchMessages;

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  useEffect(() => {
    // Focus input when switching modes
    inputRef.current?.focus();
  }, [mode]);

  const generateId = () => Math.random().toString(36).substring(7);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim() || isLoading) return;

    const query = inputValue.trim();
    setInputValue('');

    // Add user message
    const userMessage: Message = {
      id: generateId(),
      type: 'user',
      content: query,
      timestamp: new Date(),
    };
    setMessages((prev) => [...prev, userMessage]);

    // Add loading message
    const loadingId = generateId();
    setMessages((prev) => [
      ...prev,
      {
        id: loadingId,
        type: 'assistant',
        content: '',
        timestamp: new Date(),
        isLoading: true,
      },
    ]);
    setIsLoading(true);

    try {
      if (mode === 'ask') {
        const response: RAGAskResponse = await ragApi.ask({
          question: query,
          max_context: 5,
        });

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
      } else {
        const results = await ragApi.search({
          query,
          limit: 10,
        });

        setSearchMessages((prev) =>
          prev.map((msg) =>
            msg.id === loadingId
              ? {
                  ...msg,
                  type: 'search-results',
                  content: `Found ${results.length} result${results.length !== 1 ? 's' : ''}`,
                  sources: results,
                  isLoading: false,
                }
              : msg
          )
        );
      }
    } catch (error) {
      console.error('Error:', error);
      toast.error('Failed to process your request');
      setMessages((prev) => prev.filter((msg) => msg.id !== loadingId));
    } finally {
      setIsLoading(false);
      inputRef.current?.focus();
    }
  };

  const toggleSources = (messageId: string) => {
    setShowSources((prev) => ({
      ...prev,
      [messageId]: !prev[messageId],
    }));
  };

  const getContentTypeIcon = (type: string) => {
    return type === 'todo' ? CheckSquare : BookmarkSimple;
  };

  return (
    <div className="h-full flex flex-col bg-surface-light dark:bg-surface-dark relative">
      {/* Sticky Header */}
      <div className="sticky top-0 z-10 bg-surface-light/80 dark:bg-surface-dark/80 backdrop-blur-glass border-b border-gray-100 dark:border-gray-800/50 px-4 sm:px-6 py-4">
        <div className="max-w-3xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-3 pl-10 lg:pl-0">
            <ChatCircle size={24} weight="regular" className="text-primary-600" />
            <h1 className="text-xl font-heading text-gray-800 dark:text-white">
              Ask Your Data
            </h1>
          </div>

          {/* Mode Toggle */}
          <div className="flex bg-surface-light-muted dark:bg-surface-dark-muted rounded-xl p-1">
            <button
              onClick={() => setMode('ask')}
              className={`flex items-center gap-2 px-3 sm:px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                mode === 'ask'
                  ? 'bg-white dark:bg-surface-dark-elevated text-primary-600 shadow-subtle'
                  : 'text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200'
              }`}
            >
              <ChatCircle size={16} weight="regular" />
              <span className="hidden sm:inline">Ask</span>
            </button>
            <button
              onClick={() => setMode('search')}
              className={`flex items-center gap-2 px-3 sm:px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                mode === 'search'
                  ? 'bg-white dark:bg-surface-dark-elevated text-primary-600 shadow-subtle'
                  : 'text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200'
              }`}
            >
              <MagnifyingGlass size={16} weight="regular" />
              <span className="hidden sm:inline">Search</span>
            </button>
          </div>
        </div>

        {/* Temporary notice */}
        <div className="max-w-3xl mx-auto flex items-center gap-2 mt-3 px-3 py-2 bg-amber-50 dark:bg-amber-900/20 border border-amber-200/50 dark:border-amber-800/50 rounded-xl">
          <Warning size={16} weight="regular" className="text-amber-600 dark:text-amber-400 flex-shrink-0" />
          <p className="text-xs text-amber-700 dark:text-amber-300">
            This chat is temporary and will be cleared when you leave the page.
          </p>
        </div>
      </div>

      {/* Messages Area */}
      <div className="flex-1 overflow-y-auto px-4 sm:px-6 py-4 pb-32 bg-surface-light dark:bg-surface-dark">
        {messages.length === 0 ? (
          <div className="h-full flex flex-col items-center justify-center text-center">
            <div className="w-16 h-16 bg-primary-100 dark:bg-primary-900/30 rounded-2xl flex items-center justify-center mb-4">
              {mode === 'ask' ? (
                <ChatCircle size={32} weight="regular" className="text-primary-600 dark:text-primary-400" />
              ) : (
                <MagnifyingGlass size={32} weight="regular" className="text-primary-600 dark:text-primary-400" />
              )}
            </div>
            <h2 className="text-lg font-heading text-gray-800 dark:text-white mb-2">
              {mode === 'ask' ? 'Ask anything about your data' : 'Search your knowledge base'}
            </h2>
            <p className="text-gray-500 dark:text-gray-400 max-w-md text-sm">
              {mode === 'ask'
                ? 'I can answer questions by searching through your todos and memories.'
                : 'Search semantically across all your saved information.'}
            </p>
            <div className="mt-6 flex flex-wrap gap-2 justify-center">
              {mode === 'ask' ? (
                <>
                  <SuggestionChip
                    text="What are my pending tasks?"
                    onClick={() => setInputValue('What are my pending tasks?')}
                  />
                  <SuggestionChip
                    text="What places have I saved?"
                    onClick={() => setInputValue('What places have I saved?')}
                  />
                </>
              ) : (
                <>
                  <SuggestionChip text="coffee" onClick={() => setInputValue('coffee')} />
                  <SuggestionChip text="ideas" onClick={() => setInputValue('ideas')} />
                </>
              )}
            </div>
          </div>
        ) : (
          <div className="max-w-3xl mx-auto space-y-4">
            <AnimatePresence>
              {messages.map((message, index) => (
                <motion.div
                  key={message.id}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -10 }}
                  className={`flex ${message.type === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                  {message.type === 'user' ? (
                    <div className="max-w-[85%] bg-primary-600 text-white rounded-2xl rounded-br-md px-4 py-3">
                      <p className="text-sm">{message.content}</p>
                    </div>
                  ) : message.isLoading ? (
                    <div className="max-w-[85%] bg-surface-light-muted dark:bg-surface-dark-muted border border-gray-200/50 dark:border-gray-700/50 rounded-2xl rounded-bl-md px-4 py-3">
                      <div className="flex items-center gap-2 text-gray-500">
                        <CircleNotch size={16} weight="regular" className="animate-spin" />
                        <span className="text-sm">Thinking...</span>
                      </div>
                    </div>
                  ) : (
                    <div className="max-w-[85%] bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-2xl rounded-bl-md px-4 py-3">
                      {message.type === 'search-results' ? (
                        <SearchResultsView
                          message={message}
                          getContentTypeIcon={getContentTypeIcon}
                        />
                      ) : (
                        <AskResultView
                          message={message}
                          showSources={showSources}
                          toggleSources={toggleSources}
                          getContentTypeIcon={getContentTypeIcon}
                        />
                      )}
                    </div>
                  )}
                </motion.div>
              ))}
            </AnimatePresence>
            <div ref={messagesEndRef} />
          </div>
        )}
      </div>

      {/* Sticky Bottom Input */}
      <div className="absolute bottom-0 left-0 right-0 z-20">
        <div className="max-w-3xl mx-auto px-3 sm:px-4 pb-4 sm:pb-6">
          <form
            onSubmit={handleSubmit}
            className="bg-white/90 dark:bg-surface-dark-elevated/90 backdrop-blur-glass rounded-2xl shadow-float border border-gray-200/50 dark:border-gray-800/50"
          >
            <div className="flex items-center gap-2 p-3">
              {/* Icon */}
              <div className="flex-shrink-0 w-8 h-8 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
                {mode === 'ask' ? (
                  <ChatCircle size={16} weight="bold" className="text-primary-600 dark:text-primary-400" />
                ) : (
                  <MagnifyingGlass size={16} weight="bold" className="text-primary-600 dark:text-primary-400" />
                )}
              </div>

              {/* Text input */}
              <div className="flex-1 relative min-w-0">
                <input
                  ref={inputRef}
                  type="text"
                  value={inputValue}
                  onChange={(e) => setInputValue(e.target.value)}
                  placeholder={mode === 'ask' ? 'Ask a question about your data...' : 'Search your todos and memories...'}
                  className="w-full bg-transparent border-none outline-none text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 text-sm"
                  disabled={isLoading}
                />
              </div>

              {/* Submit button */}
              <button
                type="submit"
                disabled={!inputValue.trim() || isLoading}
                className="px-4 py-1.5 bg-primary-600 text-white text-sm font-medium rounded-xl hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
              >
                {isLoading ? (
                  <CircleNotch size={16} weight="regular" className="animate-spin" />
                ) : (
                  <PaperPlaneTilt size={16} weight="bold" />
                )}
                <span className="hidden sm:inline">{mode === 'ask' ? 'Ask' : 'Search'}</span>
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}

// Search results component
function SearchResultsView({
  message,
  getContentTypeIcon,
}: {
  message: Message;
  getContentTypeIcon: (type: string) => React.ElementType;
}) {
  return (
    <div>
      <p className="text-gray-600 dark:text-gray-300 text-sm mb-3">{message.content}</p>
      {message.sources && message.sources.length > 0 && (
        <div className="space-y-2">
          {message.sources.map((source, index) => {
            const Icon = getContentTypeIcon(source.document.content_type);
            return (
              <div
                key={source.document.id}
                className="p-3 bg-gray-50 dark:bg-gray-900 rounded-lg"
              >
                <div className="flex items-start gap-2">
                  <Icon className="w-4 h-4 text-gray-400 mt-0.5 flex-shrink-0" />
                  <div className="flex-1 min-w-0">
                    <p className="font-medium text-gray-800 dark:text-white text-sm">
                      {source.document.title || source.document.content.substring(0, 60)}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 line-clamp-2">
                      {source.document.content}
                    </p>
                    <div className="flex items-center gap-2 mt-2">
                      <span className="text-xs px-2 py-0.5 bg-gray-200 dark:bg-gray-700 rounded text-gray-600 dark:text-gray-300">
                        {source.document.content_type}
                      </span>
                      <span className="text-xs text-primary-600 dark:text-primary-400 font-medium">
                        #{index + 1}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

// Ask result component
function AskResultView({
  message,
  showSources,
  toggleSources,
  getContentTypeIcon,
}: {
  message: Message;
  showSources: Record<string, boolean>;
  toggleSources: (id: string) => void;
  getContentTypeIcon: (type: string) => React.ElementType;
}) {
  return (
    <div>
      <p className="text-gray-800 dark:text-white whitespace-pre-wrap text-sm">
        {message.content}
      </p>
      {message.sources && message.sources.length > 0 && (
        <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-700">
          <button
            onClick={() => toggleSources(message.id)}
            className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
          >
            <Info size={16} weight="regular" />
            <span>{message.sources.length} sources</span>
            {showSources[message.id] ? (
              <CaretUp size={16} weight="bold" />
            ) : (
              <CaretDown size={16} weight="bold" />
            )}
          </button>
          {showSources[message.id] && (
            <div className="mt-2 space-y-2">
              {message.sources.map((source, index) => {
                const Icon = getContentTypeIcon(source.document.content_type);
                const title = source.document.title || source.document.content.substring(0, 50);
                return (
                  <div
                    key={source.document.id}
                    className="p-2 bg-gray-50 dark:bg-gray-900 rounded-md text-sm"
                  >
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-primary-600 font-medium">#{index + 1}</span>
                      <Icon className="w-3 h-3 text-gray-400" />
                      <span className="text-gray-700 dark:text-gray-300 font-medium text-xs">
                        {title.substring(0, 50)}
                        {title.length > 50 ? '...' : ''}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function SuggestionChip({ text, onClick }: { text: string; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="px-4 py-2 bg-surface-light-muted dark:bg-surface-dark-muted border border-gray-200/50 dark:border-gray-700/50 rounded-xl text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-surface-dark-elevated transition-all duration-200"
    >
      {text}
    </button>
  );
}
