import { useRef, useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { toast } from 'react-hot-toast';
import ReactMarkdown from 'react-markdown';
import { ChatCircle, CircleNotch, PaperPlaneTilt, CaretUp, CaretDown, Copy, ArrowsClockwise } from '@phosphor-icons/react';
import type { RAGAskResponse } from '../types';

interface Message {
  id: string;
  type: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  isLoading?: boolean;
}

interface AskAIResultsProps {
  messages: Message[];
  isLoading: boolean;
  inputValue: string;
  onInputChange: (value: string) => void;
  onSubmit: (e: React.FormEvent) => void;
  onRegenerate?: (lastQuery: string) => void;
  onCitationsExtracted?: (citations: RAGAskResponse['sources']) => void;
}

export default function AskAIResults({
  messages,
  isLoading,
  inputValue,
  onInputChange,
  onSubmit,
  onRegenerate,
  onCitationsExtracted,
}: AskAIResultsProps) {
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [showScrollUp, setShowScrollUp] = useState(false);
  const [showScrollDown, setShowScrollDown] = useState(false);

  const hasMessages = messages.length > 0;
  const MAX_HEIGHT = '400px'; // Maximum height for expand area

  // Check scroll position for arrow visibility
  useEffect(() => {
    const container = messagesContainerRef.current;
    if (!container || !hasMessages) {
      setShowScrollUp(false);
      setShowScrollDown(false);
      return;
    }

    const checkScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = container;
      const canScroll = scrollHeight > clientHeight;
      setShowScrollUp(canScroll && scrollTop > 50);
      setShowScrollDown(canScroll && scrollTop < scrollHeight - clientHeight - 50);
    };

    // Check immediately and after a short delay to account for layout
    checkScroll();
    const timeoutId = setTimeout(checkScroll, 100);
    
    container.addEventListener('scroll', checkScroll);
    return () => {
      clearTimeout(timeoutId);
      container.removeEventListener('scroll', checkScroll);
    };
  }, [hasMessages, messages]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const scrollUp = () => {
    messagesContainerRef.current?.scrollTo({
      top: 0,
      behavior: 'smooth',
    });
  };

  const scrollDown = () => {
    messagesContainerRef.current?.scrollTo({
      top: messagesContainerRef.current.scrollHeight,
      behavior: 'smooth',
    });
  };

  // Get last user query for regenerate
  const getLastUserQuery = () => {
    for (let i = messages.length - 1; i >= 0; i--) {
      if (messages[i].type === 'user') {
        return messages[i].content;
      }
    }
    return null;
  };

  const handleCopy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      toast.success('Copied to clipboard');
    } catch (error) {
      toast.error('Failed to copy');
    }
  };

  const handleRegenerate = () => {
    const lastQuery = getLastUserQuery();
    if (lastQuery && onRegenerate) {
      onRegenerate(lastQuery);
    }
  };

  return (
    <div className="p-6">
      <div className="max-w-3xl mx-auto">
        {/* Unified Container - Expand area and input as one component */}
        <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl border border-gray-200/50 dark:border-gray-700/50 backdrop-blur-sm overflow-hidden">
          {/* Results Area - Expands upward from within the same container */}
          <AnimatePresence>
            {hasMessages && (
              <motion.div
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: 'auto', opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                transition={{
                  height: { duration: 0.4, ease: [0.4, 0, 0.2, 1] },
                  opacity: { duration: 0.3 },
                }}
                style={{ maxHeight: MAX_HEIGHT }}
                className="relative overflow-hidden border-b border-gray-200/50 dark:border-gray-700/50"
              >
                <div
                  ref={messagesContainerRef}
                  className="overflow-y-auto px-6 py-6 scrollbar-hide"
                  style={{ maxHeight: MAX_HEIGHT }}
                >
                  <div className="space-y-6">
                    {messages.map((message, index) => (
                      <motion.div
                        key={message.id}
                        initial={{ opacity: 0, y: 10 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{
                          delay: index * 0.05,
                          duration: 0.2,
                          ease: [0.4, 0, 0.2, 1],
                        }}
                        className={`flex ${message.type === 'user' ? 'justify-end' : 'justify-start'}`}
                      >
                        {message.type === 'user' ? (
                          <motion.div
                            initial={{ scale: 0.9, opacity: 0 }}
                            animate={{ scale: 1, opacity: 1 }}
                            className="bg-gray-100 dark:bg-gray-700 rounded-lg px-4 py-2.5"
                          >
                            <p className="text-sm text-gray-900 dark:text-white">{message.content}</p>
                          </motion.div>
                        ) : message.isLoading ? (
                          <motion.div
                            initial={{ scale: 0.9, opacity: 0 }}
                            animate={{ scale: 1, opacity: 1 }}
                            className="flex items-center gap-2 text-gray-500 dark:text-gray-400"
                          >
                            <CircleNotch size={16} weight="regular" className="animate-spin" />
                            <span className="text-sm">Thinking...</span>
                          </motion.div>
                        ) : (
                          <div className="flex flex-col gap-2 max-w-[85%]">
                            <div className="text-gray-900 dark:text-white">
                              <ReactMarkdown
                                components={{
                                  // Style headings
                                  h1: ({ node, ...props }) => <h1 className="text-lg font-bold mb-2 mt-4 first:mt-0" {...props} />,
                                  h2: ({ node, ...props }) => <h2 className="text-base font-bold mb-2 mt-3 first:mt-0" {...props} />,
                                  h3: ({ node, ...props }) => <h3 className="text-sm font-semibold mb-1.5 mt-2 first:mt-0" {...props} />,
                                  // Style paragraphs
                                  p: ({ node, ...props }) => <p className="mb-2 last:mb-0 text-sm leading-relaxed" {...props} />,
                                  // Style lists
                                  ul: ({ node, ...props }) => <ul className="list-disc list-inside mb-2 space-y-1 text-sm" {...props} />,
                                  ol: ({ node, ...props }) => <ol className="list-decimal list-inside mb-2 space-y-1 text-sm" {...props} />,
                                  li: ({ node, ...props }) => <li className="text-sm" {...props} />,
                                  // Style code blocks
                                  code: ({ node, className, children, ...props }: any) => {
                                    const match = /language-(\w+)/.exec(className || '');
                                    const isInline = !match;
                                    return isInline ? (
                                      <code className="bg-gray-100 dark:bg-gray-800 px-1.5 py-0.5 rounded text-xs font-mono" {...props}>
                                        {children}
                                      </code>
                                    ) : (
                                      <code className="block bg-gray-100 dark:bg-gray-800 p-2 rounded text-xs font-mono overflow-x-auto mb-2" {...props}>
                                        {children}
                                      </code>
                                    );
                                  },
                                  pre: ({ node, children, ...props }: any) => (
                                    <pre className="bg-gray-100 dark:bg-gray-800 p-2 rounded mb-2 overflow-x-auto" {...props}>
                                      {children}
                                    </pre>
                                  ),
                                  // Style links
                                  a: ({ node, ...props }) => <a className="text-primary-600 dark:text-primary-400 hover:underline" {...props} />,
                                  // Style blockquotes
                                  blockquote: ({ node, ...props }) => <blockquote className="border-l-4 border-gray-300 dark:border-gray-600 pl-4 italic my-2" {...props} />,
                                  // Style strong and emphasis
                                  strong: ({ node, ...props }) => <strong className="font-semibold" {...props} />,
                                  em: ({ node, ...props }) => <em className="italic" {...props} />,
                                  // Style horizontal rule
                                  hr: ({ node, ...props }) => <hr className="my-3 border-gray-200 dark:border-gray-700" {...props} />,
                                }}
                              >
                                {message.content}
                              </ReactMarkdown>
                            </div>
                            {/* Interaction Icons - Only Copy and Regenerate for AI messages */}
                            <div className="flex items-center gap-3 text-gray-400 dark:text-gray-500">
                              <button
                                onClick={() => handleCopy(message.content)}
                                className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors"
                                title="Copy"
                              >
                                <Copy size={14} weight="regular" />
                              </button>
                              <button
                                onClick={handleRegenerate}
                                disabled={isLoading || !getLastUserQuery()}
                                className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                title="Regenerate"
                              >
                                <ArrowsClockwise size={14} weight="regular" />
                              </button>
                            </div>
                          </div>
                        )}
                      </motion.div>
                    ))}
                    <div ref={messagesEndRef} />
                  </div>
                </div>

                {/* Scroll Navigation Arrows - Bottom Right */}
                <div className="absolute bottom-4 right-4 flex flex-col gap-2 z-10">
                  <AnimatePresence>
                    {showScrollUp && (
                      <motion.button
                        initial={{ opacity: 0, scale: 0.8, y: 10 }}
                        animate={{ opacity: 1, scale: 1, y: 0 }}
                        exit={{ opacity: 0, scale: 0.8, y: 10 }}
                        onClick={scrollUp}
                        className="p-2 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-full shadow-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                        title="Scroll up"
                      >
                        <CaretUp size={18} weight="bold" className="text-gray-600 dark:text-gray-300" />
                      </motion.button>
                    )}
                    {showScrollDown && (
                      <motion.button
                        initial={{ opacity: 0, scale: 0.8, y: -10 }}
                        animate={{ opacity: 1, scale: 1, y: 0 }}
                        exit={{ opacity: 0, scale: 0.8, y: -10 }}
                        onClick={scrollDown}
                        className="p-2 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-full shadow-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                        title="Scroll down"
                      >
                        <CaretDown size={18} weight="bold" className="text-gray-600 dark:text-gray-300" />
                      </motion.button>
                    )}
                  </AnimatePresence>
                </div>
              </motion.div>
            )}
          </AnimatePresence>

          {/* Input Area - Part of the same container, always visible with more padding */}
          <div className="p-6">
            <form onSubmit={onSubmit}>
              <div className="flex items-center gap-3">
                <ChatCircle size={20} weight="regular" className="text-primary-600 dark:text-primary-400 flex-shrink-0" />
                <input
                  ref={inputRef}
                  type="text"
                  value={inputValue}
                  onChange={(e) => onInputChange(e.target.value)}
                  placeholder="Ask AI about your memories and todos..."
                  disabled={isLoading}
                  className="flex-1 bg-transparent border-none outline-none text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 text-sm"
                />
                <button
                  type="submit"
                  disabled={!inputValue.trim() || isLoading}
                  className="p-2 bg-primary-600 text-white rounded-full hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  {isLoading ? (
                    <CircleNotch size={18} weight="regular" className="animate-spin" />
                  ) : (
                    <PaperPlaneTilt size={18} weight="bold" />
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}
