import { useRef, useEffect, useState, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { toast } from 'react-hot-toast';
import ReactMarkdown from 'react-markdown';
import {
  ChatCircle,
  CircleNotch,
  PaperPlaneTilt,
  CaretUp,
  CaretDown,
  Copy,
  ArrowsClockwise,
  X,
  Brain,
  Minus,
  Plus,
  Globe,
  Intersect,
} from '@phosphor-icons/react';
import type { RAGAskResponse, RAGSearchResult, AskMode } from '../types';

interface Message {
  id: string;
  type: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  isLoading?: boolean;
  sources?: RAGSearchResult[];
  mode?: AskMode; // The mode this message was created with
}

// Card represents a Q&A pair
interface CardData {
  id: string;
  question: string;
  answer: string | null;
  sources?: RAGSearchResult[];
  timestamp: Date;
  status: 'searching' | 'answered' | 'error';
  mode: AskMode; // The mode this card was created with
}

interface AskAIResultsProps {
  messages: Message[];
  isLoading: boolean;
  inputValue: string;
  onInputChange: (value: string) => void;
  onSubmit: (e: React.FormEvent) => void;
  onRegenerate?: (lastQuery: string) => void;
  onClear?: () => void;
  onCitationsExtracted?: (citations: RAGAskResponse['sources']) => void;
  mode: AskMode;
  onModeChange: (mode: AskMode) => void;
}

// Modal for expanded card content
interface ExpandedCardModalProps {
  card: CardData | null;
  onClose: () => void;
  onCopy: (text: string) => void;
  onRegenerate: (question: string) => void;
  isLoading: boolean;
}

// Mode configuration with colors
const modes: { id: AskMode; label: string; description: string }[] = [
  { id: 'memories', label: 'Memories', description: 'Search your memories' },
  { id: 'hybrid', label: 'Hybrid', description: 'Memories + Internet' },
  { id: 'internet', label: 'Internet', description: 'Search the web' },
  { id: 'llm', label: 'AI Only', description: 'Direct AI response' },
];

// Mode-specific color schemes
const modeColors: Record<AskMode, {
  button: string;
  buttonHover: string;
  icon: string;
  iconBg: string;
  ring: string;
  text: string;
  border: string;
  cardRing: string;
}> = {
  memories: {
    button: 'bg-primary-600',
    buttonHover: 'hover:bg-primary-700',
    icon: 'text-primary-600 dark:text-primary-400',
    iconBg: 'bg-primary-100 dark:bg-primary-900/30 hover:bg-primary-200 dark:hover:bg-primary-900/50',
    ring: 'focus:ring-primary-500',
    text: 'text-primary-600 dark:text-primary-400',
    border: 'border-primary-500',
    cardRing: 'ring-primary-500/20',
  },
  internet: {
    button: 'bg-blue-600',
    buttonHover: 'hover:bg-blue-700',
    icon: 'text-blue-600 dark:text-blue-400',
    iconBg: 'bg-blue-100 dark:bg-blue-900/30 hover:bg-blue-200 dark:hover:bg-blue-900/50',
    ring: 'focus:ring-blue-500',
    text: 'text-blue-600 dark:text-blue-400',
    border: 'border-blue-500',
    cardRing: 'ring-blue-500/20',
  },
  hybrid: {
    button: 'bg-violet-600',
    buttonHover: 'hover:bg-violet-700',
    icon: 'text-violet-600 dark:text-violet-400',
    iconBg: 'bg-violet-100 dark:bg-violet-900/30 hover:bg-violet-200 dark:hover:bg-violet-900/50',
    ring: 'focus:ring-violet-500',
    text: 'text-violet-600 dark:text-violet-400',
    border: 'border-violet-500',
    cardRing: 'ring-violet-500/20',
  },
  llm: {
    button: 'bg-emerald-600',
    buttonHover: 'hover:bg-emerald-700',
    icon: 'text-emerald-600 dark:text-emerald-400',
    iconBg: 'bg-emerald-100 dark:bg-emerald-900/30 hover:bg-emerald-200 dark:hover:bg-emerald-900/50',
    ring: 'focus:ring-emerald-500',
    text: 'text-emerald-600 dark:text-emerald-400',
    border: 'border-emerald-500',
    cardRing: 'ring-emerald-500/20',
  },
};

const getModeIcon = (mode: AskMode, size: number = 20) => {
  switch (mode) {
    case 'memories':
      return <Brain size={size} weight="regular" />;
    case 'internet':
      return <Globe size={size} weight="regular" />;
    case 'hybrid':
      return <Intersect size={size} weight="regular" />;
    case 'llm':
      return <ChatCircle size={size} weight="regular" />;
    default:
      return <Brain size={size} weight="regular" />;
  }
};

// Expanded Card Modal Component
function ExpandedCardModal({ card, onClose, onCopy, onRegenerate, isLoading }: ExpandedCardModalProps) {
  if (!card) return null;

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="fixed inset-0 z-[1000] flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm"
      onClick={onClose}
    >
      <motion.div
        initial={{ scale: 0.95, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        exit={{ scale: 0.95, opacity: 0 }}
        transition={{ type: 'spring', stiffness: 300, damping: 30 }}
        className="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl border border-gray-200 dark:border-gray-700 max-w-2xl w-full max-h-[80vh] overflow-hidden flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Modal Header */}
        <div className="p-4 border-b border-gray-200 dark:border-gray-700 flex items-start gap-3">
          <div className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center ${modeColors[card.mode].iconBg.split(' ').slice(0, 2).join(' ')}`}>
            <span className={modeColors[card.mode].icon}>
              {getModeIcon(card.mode, 16)}
            </span>
          </div>
          <h3 className="flex-1 text-base font-medium text-gray-900 dark:text-white leading-relaxed">
            {card.question}?
          </h3>
          <button
            onClick={onClose}
            className="p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            <X size={18} weight="bold" />
          </button>
        </div>

        {/* Modal Body - Scrollable */}
        <div className="flex-1 overflow-y-auto p-4">
          <div className="text-gray-800 dark:text-gray-200">
            <ReactMarkdown
              components={{
                h1: ({ node, ...props }) => <h1 className="text-lg font-bold mb-2 mt-4 first:mt-0" {...props} />,
                h2: ({ node, ...props }) => <h2 className="text-base font-bold mb-2 mt-3 first:mt-0" {...props} />,
                h3: ({ node, ...props }) => <h3 className="text-sm font-semibold mb-1.5 mt-2 first:mt-0" {...props} />,
                p: ({ node, ...props }) => <p className="mb-3 last:mb-0 text-sm leading-relaxed" {...props} />,
                ul: ({ node, ...props }) => <ul className="list-disc list-inside mb-3 space-y-1.5 text-sm" {...props} />,
                ol: ({ node, ...props }) => <ol className="list-decimal list-inside mb-3 space-y-1.5 text-sm" {...props} />,
                li: ({ node, ...props }) => <li className="text-sm" {...props} />,
                code: ({ node, className, children, ...props }: any) => {
                  const match = /language-(\w+)/.exec(className || '');
                  const isInline = !match;
                  return isInline ? (
                    <code className="bg-gray-100 dark:bg-gray-700 px-1.5 py-0.5 rounded text-xs font-mono" {...props}>
                      {children}
                    </code>
                  ) : (
                    <code className="block bg-gray-100 dark:bg-gray-700 p-3 rounded text-xs font-mono overflow-x-auto mb-3" {...props}>
                      {children}
                    </code>
                  );
                },
                pre: ({ node, children, ...props }: any) => (
                  <pre className="bg-gray-100 dark:bg-gray-700 p-3 rounded mb-3 overflow-x-auto" {...props}>
                    {children}
                  </pre>
                ),
                a: ({ node, ...props }) => <a className="text-primary-600 dark:text-primary-400 hover:underline" {...props} />,
                blockquote: ({ node, ...props }) => <blockquote className="border-l-4 border-gray-300 dark:border-gray-600 pl-4 italic my-3" {...props} />,
                strong: ({ node, ...props }) => <strong className="font-semibold" {...props} />,
                em: ({ node, ...props }) => <em className="italic" {...props} />,
              }}
            >
              {card.answer || ''}
            </ReactMarkdown>
          </div>

          {/* Sources */}
          {card.sources && card.sources.length > 0 && (
            <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
              <h4 className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">
                Sources ({card.sources.length})
              </h4>
              <div className="space-y-2">
                {card.sources.map((source, idx) => (
                  <div
                    key={idx}
                    className="text-xs text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-gray-700/50 rounded-lg p-2"
                  >
                    <span className="font-medium">{source.document.content_type}:</span>{' '}
                    {source.document.title || source.document.content?.slice(0, 100)}
                    {source.document.metadata?.url && (
                      <a
                        href={source.document.metadata.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="ml-1 text-primary-600 dark:text-primary-400 hover:underline"
                      >
                        ↗
                      </a>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Modal Footer - Actions */}
        <div className="p-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-end gap-2">
          <button
            onClick={() => onCopy(card.answer || '')}
            className="px-3 py-1.5 text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors flex items-center gap-1.5"
          >
            <Copy size={14} weight="regular" />
            Copy
          </button>
          <button
            onClick={() => {
              onRegenerate(card.question);
              onClose();
            }}
            disabled={isLoading}
            className="px-3 py-1.5 text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors flex items-center gap-1.5 disabled:opacity-50"
          >
            <ArrowsClockwise size={14} weight="regular" />
            Regenerate
          </button>
          <button
            onClick={onClose}
            className={`px-3 py-1.5 text-sm text-white rounded-lg transition-colors ${modeColors[card.mode].button} ${modeColors[card.mode].buttonHover}`}
          >
            Close
          </button>
        </div>
      </motion.div>
    </motion.div>
  );
}

export default function AskAIResults({
  messages,
  isLoading,
  inputValue,
  onInputChange,
  onSubmit,
  onRegenerate,
  onClear,
  onCitationsExtracted,
  mode,
  onModeChange,
}: AskAIResultsProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [activeCardIndex, setActiveCardIndex] = useState(0);
  const [modalCard, setModalCard] = useState<CardData | null>(null);
  const [isMinimized, setIsMinimized] = useState(false);
  const [showModeDropup, setShowModeDropup] = useState(false);
  const modeButtonRef = useRef<HTMLButtonElement>(null);

  // Convert messages to cards (pair user questions with AI answers)
  const cards: CardData[] = [];
  for (let i = 0; i < messages.length; i++) {
    const msg = messages[i];
    if (msg.type === 'user') {
      // Find the corresponding assistant response
      const nextMsg = messages[i + 1];
      const card: CardData = {
        id: msg.id,
        question: msg.content,
        answer: null,
        timestamp: msg.timestamp,
        status: 'searching',
        mode: msg.mode || 'memories', // Use the mode from the message, default to memories
      };

      if (nextMsg && nextMsg.type === 'assistant') {
        if (nextMsg.isLoading) {
          card.status = 'searching';
        } else {
          card.answer = nextMsg.content;
          card.sources = nextMsg.sources;
          card.status = 'answered';
        }
        i++; // Skip the assistant message in next iteration
      }

      cards.unshift(card); // Add to beginning (newest first)
    }
  }

  const hasCards = cards.length > 0;

  // Reorder cards so active card is first (front) for display
  const getDisplayCards = useCallback(() => {
    if (cards.length === 0) return [];
    const result = [...cards];
    const activeCard = result.splice(activeCardIndex, 1)[0];
    return [activeCard, ...result]; // Active first, others behind
  }, [cards, activeCardIndex]);

  const displayCards = getDisplayCards();

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!hasCards) return;

      // Only handle if not focused on input
      if (document.activeElement === inputRef.current) return;

      if (e.key === 'ArrowUp' && activeCardIndex < cards.length - 1) {
        e.preventDefault();
        setActiveCardIndex((prev) => prev + 1);
      } else if (e.key === 'ArrowDown' && activeCardIndex > 0) {
        e.preventDefault();
        setActiveCardIndex((prev) => prev - 1);
      } else if (e.key === 'Escape') {
        setModalCard(null);
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [hasCards, activeCardIndex, cards.length]);

  // Reset active card when new card is added
  useEffect(() => {
    if (cards.length > 0) {
      setActiveCardIndex(0);
    }
  }, [cards.length]);

  // Auto-expand when loading (new message being sent)
  useEffect(() => {
    if (isLoading) {
      setIsMinimized(false);
    }
  }, [isLoading]);

  const handleClearAll = useCallback(() => {
    if (onClear) {
      onClear();
    }
  }, [onClear]);

  const handleCopy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      toast.success('Copied to clipboard');
    } catch (error) {
      toast.error('Failed to copy');
    }
  };

  const handleRegenerate = (question: string) => {
    if (onRegenerate) {
      onRegenerate(question);
    }
  };

  const goToPrev = () => {
    setActiveCardIndex((prev) => Math.min(prev + 1, cards.length - 1));
  };

  const goToNext = () => {
    setActiveCardIndex((prev) => Math.max(prev - 1, 0));
  };

  // Processing messages based on mode
  const getProcessingMessages = (currentMode: AskMode) => {
    switch (currentMode) {
      case 'memories':
        return ['Searching your brain...', 'Finding connections...', 'Retrieving memories...', 'Analyzing your data...'];
      case 'internet':
        return ['Searching the web...', 'Fetching results...', 'Reading pages...', 'Gathering information...'];
      case 'hybrid':
        return ['Searching memories and web...', 'Combining sources...', 'Cross-referencing data...', 'Building context...'];
      case 'llm':
        return ['Thinking...', 'Processing...', 'Generating response...', 'Formulating answer...'];
      default:
        return ['Processing...'];
    }
  };

  const getRandomProcessingMessage = () => {
    const messages = getProcessingMessages(mode);
    return messages[Math.floor(Math.random() * messages.length)];
  };

  // Handle Tab key to cycle modes when input is focused
  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Tab' && !e.shiftKey) {
      e.preventDefault();
      const currentIndex = modes.findIndex(m => m.id === mode);
      const nextIndex = (currentIndex + 1) % modes.length;
      onModeChange(modes[nextIndex].id);
    }
  };

  // Close dropup when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (modeButtonRef.current && !modeButtonRef.current.contains(e.target as Node)) {
        setShowModeDropup(false);
      }
    };

    if (showModeDropup) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [showModeDropup]);

  return (
    <div className="p-3 sm:p-6">
      <div className="max-w-3xl mx-auto">
        {/* Controls Row - Minimize in center, Clear all on right */}
        <AnimatePresence>
          {hasCards && (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: 10 }}
              className="flex items-center justify-center mb-3 relative"
            >
              {/* Minimize/Expand Button - Center */}
              <button
                onClick={() => setIsMinimized(!isMinimized)}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-600 hover:text-gray-800 dark:text-gray-300 dark:hover:text-gray-100 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 border border-gray-200 dark:border-gray-600 rounded-full transition-colors shadow-sm"
              >
                {isMinimized ? (
                  <>
                    <Plus size={12} weight="bold" />
                    Show {cards.length} card{cards.length !== 1 ? 's' : ''}
                  </>
                ) : (
                  <>
                    <Minus size={12} weight="bold" />
                    Minimize
                  </>
                )}
              </button>

              {/* Clear All Button - Absolute Right */}
              <button
                onClick={handleClearAll}
                className="absolute right-0 inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 bg-red-50 dark:bg-red-900/30 hover:bg-red-100 dark:hover:bg-red-900/50 border border-red-200 dark:border-red-800 rounded-full transition-colors shadow-sm"
              >
                <X size={12} weight="bold" />
                Clear all
              </button>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Unified Container */}
        <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl border border-gray-200/50 dark:border-gray-700/50 backdrop-blur-sm overflow-visible">
          {/* Cards Section - Only show when not minimized */}
          <AnimatePresence>
            {hasCards && !isMinimized && (
              <motion.div
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: 'auto' }}
                exit={{ opacity: 0, height: 0 }}
                className="relative"
              >
                {/* Card Stack Area - with more bottom margin from input */}
                <div className="p-4 pb-6 relative" style={{ minHeight: '300px' }}>
                  {/* Render cards in stack (max 5 visible) */}
                  {displayCards.slice(0, 5).map((card, displayIndex) => {
                    const isActive = displayIndex === 0;
                    const stackOffset = displayIndex * 10;
                    const scale = 1 - displayIndex * 0.02;
                    const zIndex = 50 - displayIndex;
                    // More aggressive blur/fade for background cards
                    const cardOpacity = isActive ? 1 : Math.max(0.3, 1 - displayIndex * 0.25);
                    const blur = isActive ? 0 : displayIndex * 1.5;

                    return (
                      <motion.div
                        key={card.id}
                        layoutId={card.id}
                        initial={false}
                        animate={{
                          y: stackOffset,
                          scale,
                          opacity: cardOpacity,
                          filter: `blur(${blur}px)`,
                        }}
                        transition={{ type: 'spring', stiffness: 300, damping: 30 }}
                        style={{ zIndex }}
                        onClick={() => {
                          // Find original index of this card
                          const originalIndex = cards.findIndex((c) => c.id === card.id);
                          if (originalIndex !== -1) {
                            if (isActive && card.status === 'answered') {
                              // If clicking active card that has an answer, open modal
                              setModalCard(card);
                            } else {
                              // Otherwise, make it active
                              setActiveCardIndex(originalIndex);
                            }
                          }
                        }}
                        className={`absolute inset-x-4 bg-white dark:bg-gray-800 rounded-xl border overflow-hidden cursor-pointer transition-colors ${
                          isActive
                            ? `${modeColors[card.mode].border} ring-2 ${modeColors[card.mode].cardRing}`
                            : 'border-gray-200 dark:border-gray-700'
                        }`}
                      >
                        {/* Card Header - Question */}
                        <div className="p-4 border-b border-gray-100 dark:border-gray-700">
                          <div className="flex items-start gap-3">
                            <div className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center ${modeColors[card.mode].iconBg.split(' ').slice(0, 2).join(' ')}`}>
                              <span className={modeColors[card.mode].icon}>
                                {getModeIcon(card.mode, 16)}
                              </span>
                            </div>
                            <h3 className="flex-1 text-sm font-medium text-gray-900 dark:text-white leading-relaxed">
                              {card.question}?
                            </h3>
                          </div>
                        </div>

                        {/* Card Body - Answer */}
                        <div className="p-4 max-h-40 overflow-hidden relative">
                          {card.status === 'searching' ? (
                            <div className="flex items-center gap-3 text-gray-500 dark:text-gray-400 py-4">
                              <CircleNotch size={18} weight="regular" className={`animate-spin ${modeColors[card.mode].text}`} />
                              <span className="text-sm">{getRandomProcessingMessage()}</span>
                            </div>
                          ) : card.status === 'error' ? (
                            <p className="text-sm text-red-500 py-2">Failed to get answer. Please try again.</p>
                          ) : (
                            <div className="text-gray-800 dark:text-gray-200">
                              <div className="line-clamp-5">
                                <ReactMarkdown
                                  components={{
                                    h1: ({ node, ...props }) => <h1 className="text-lg font-bold mb-2 mt-4 first:mt-0" {...props} />,
                                    h2: ({ node, ...props }) => <h2 className="text-base font-bold mb-2 mt-3 first:mt-0" {...props} />,
                                    h3: ({ node, ...props }) => <h3 className="text-sm font-semibold mb-1.5 mt-2 first:mt-0" {...props} />,
                                    p: ({ node, ...props }) => <p className="mb-2 last:mb-0 text-sm leading-relaxed" {...props} />,
                                    ul: ({ node, ...props }) => <ul className="list-disc list-inside mb-2 space-y-1 text-sm" {...props} />,
                                    ol: ({ node, ...props }) => <ol className="list-decimal list-inside mb-2 space-y-1 text-sm" {...props} />,
                                    li: ({ node, ...props }) => <li className="text-sm" {...props} />,
                                    code: ({ node, className, children, ...props }: any) => {
                                      const match = /language-(\w+)/.exec(className || '');
                                      const isInline = !match;
                                      return isInline ? (
                                        <code className="bg-gray-100 dark:bg-gray-700 px-1.5 py-0.5 rounded text-xs font-mono" {...props}>
                                          {children}
                                        </code>
                                      ) : (
                                        <code className="block bg-gray-100 dark:bg-gray-700 p-2 rounded text-xs font-mono overflow-x-auto mb-2" {...props}>
                                          {children}
                                        </code>
                                      );
                                    },
                                    pre: ({ node, children, ...props }: any) => (
                                      <pre className="bg-gray-100 dark:bg-gray-700 p-2 rounded mb-2 overflow-x-auto" {...props}>
                                        {children}
                                      </pre>
                                    ),
                                    a: ({ node, ...props }) => <a className="text-primary-600 dark:text-primary-400 hover:underline" {...props} />,
                                    blockquote: ({ node, ...props }) => <blockquote className="border-l-4 border-gray-300 dark:border-gray-600 pl-4 italic my-2" {...props} />,
                                    strong: ({ node, ...props }) => <strong className="font-semibold" {...props} />,
                                    em: ({ node, ...props }) => <em className="italic" {...props} />,
                                  }}
                                >
                                  {card.answer || ''}
                                </ReactMarkdown>
                              </div>

                              {/* Click to expand indicator - gradient fade */}
                              {card.answer && card.answer.length > 200 && (
                                <div className="absolute bottom-0 left-0 right-0 h-12 bg-gradient-to-t from-white dark:from-gray-800 to-transparent pointer-events-none flex items-end justify-center pb-1">
                                  <span className={`text-xs font-medium ${modeColors[card.mode].text} opacity-70`}>
                                    Click to expand
                                  </span>
                                </div>
                              )}
                            </div>
                          )}
                        </div>

                        {/* Card Footer - Actions & Sources */}
                        {card.status === 'answered' && (
                          <div className="px-4 pb-3 flex items-center justify-between">
                            <div className="flex items-center gap-2">
                              {card.sources && card.sources.length > 0 && (
                                <span className="text-xs text-gray-400 dark:text-gray-500">
                                  {card.sources.length} source{card.sources.length !== 1 ? 's' : ''}
                                </span>
                              )}
                            </div>
                            <div className="flex items-center gap-1">
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleCopy(card.answer || '');
                                }}
                                className="p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors"
                                title="Copy"
                              >
                                <Copy size={14} weight="regular" />
                              </button>
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleRegenerate(card.question);
                                }}
                                disabled={isLoading}
                                className="p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors disabled:opacity-50"
                                title="Regenerate"
                              >
                                <ArrowsClockwise size={14} weight="regular" />
                              </button>
                            </div>
                          </div>
                        )}
                      </motion.div>
                    );
                  })}

                  {/* Overlay Navigation - positioned on top right of cards */}
                  {cards.length > 1 && (
                    <div className="absolute top-4 right-6 z-[60] flex flex-col items-center gap-1 bg-white/90 dark:bg-gray-800/90 backdrop-blur-sm rounded-full py-2 px-1.5 shadow-lg border border-gray-200/50 dark:border-gray-700/50">
                      <button
                        onClick={goToPrev}
                        disabled={activeCardIndex >= cards.length - 1}
                        className="p-1.5 rounded-full hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                        title="Previous card"
                      >
                        <CaretUp size={16} weight="bold" className="text-gray-600 dark:text-gray-400" />
                      </button>
                      <span className="text-xs text-gray-500 dark:text-gray-400 font-medium">
                        {activeCardIndex + 1}/{cards.length}
                      </span>
                      <button
                        onClick={goToNext}
                        disabled={activeCardIndex <= 0}
                        className="p-1.5 rounded-full hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                        title="Next card"
                      >
                        <CaretDown size={16} weight="bold" className="text-gray-600 dark:text-gray-400" />
                      </button>
                    </div>
                  )}
                </div>

                {/* Divider */}
                <div className="border-t border-gray-100 dark:border-gray-700" />
              </motion.div>
            )}
          </AnimatePresence>

          {/* Input Row */}
          <div className="p-3 sm:p-5 relative z-[100]">
            <form onSubmit={onSubmit}>
              <div className="flex items-center gap-3">
                {/* Mode Switcher Button */}
                <div className="relative flex-shrink-0 z-[100]">
                  <button
                    ref={modeButtonRef}
                    type="button"
                    onClick={() => setShowModeDropup(!showModeDropup)}
                    className={`p-2 rounded-lg transition-colors ${modeColors[mode].icon} ${modeColors[mode].iconBg}`}
                    title={`Mode: ${modes.find(m => m.id === mode)?.label || mode} (Press Tab to cycle)`}
                  >
                    {getModeIcon(mode, 20)}
                  </button>

                  {/* Mode Dropup Menu */}
                  <AnimatePresence>
                    {showModeDropup && (
                      <motion.div
                        initial={{ opacity: 0, y: 10, scale: 0.95 }}
                        animate={{ opacity: 1, y: 0, scale: 1 }}
                        exit={{ opacity: 0, y: 10, scale: 0.95 }}
                        transition={{ duration: 0.15 }}
                        className="absolute bottom-full left-0 mb-2 w-48 bg-white dark:bg-gray-800 rounded-xl shadow-xl border border-gray-200 dark:border-gray-700 overflow-hidden z-[200]"
                      >
                        <div className="py-1">
                          {modes.map((m) => {
                            const colors = modeColors[m.id];
                            const isSelected = mode === m.id;
                            return (
                              <button
                                key={m.id}
                                type="button"
                                onClick={() => {
                                  onModeChange(m.id);
                                  setShowModeDropup(false);
                                }}
                                className={`w-full px-3 py-2.5 flex items-center gap-3 text-left transition-colors ${
                                  isSelected
                                    ? `${colors.iconBg.split(' ')[0]} ${colors.iconBg.split(' ')[1]}`
                                    : 'hover:bg-gray-50 dark:hover:bg-gray-700/50'
                                } ${isSelected ? colors.text : 'text-gray-700 dark:text-gray-300'}`}
                              >
                                <span className={isSelected ? colors.icon : 'text-gray-500 dark:text-gray-400'}>
                                  {getModeIcon(m.id, 18)}
                                </span>
                                <div>
                                  <div className="text-sm font-medium">{m.label}</div>
                                  <div className="text-xs text-gray-500 dark:text-gray-400">{m.description}</div>
                                </div>
                              </button>
                            );
                          })}
                        </div>
                        <div className="px-3 py-2 bg-gray-50 dark:bg-gray-700/50 border-t border-gray-200 dark:border-gray-700">
                          <span className="text-xs text-gray-500 dark:text-gray-400">Press Tab to cycle modes</span>
                        </div>
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>

                <input
                  ref={inputRef}
                  type="text"
                  value={inputValue}
                  onChange={(e) => onInputChange(e.target.value)}
                  onKeyDown={handleInputKeyDown}
                  placeholder={
                    mode === 'memories' ? 'Ask about your memories ...' :
                    mode === 'internet' ? 'Search the web...' :
                    mode === 'hybrid' ? 'Search memories and the web...' :
                    'Ask anything...'
                  }
                  disabled={isLoading}
                  className="flex-1 bg-transparent border-none outline-none text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 text-sm"
                />
                <button
                  type="submit"
                  disabled={!inputValue.trim() || isLoading}
                  className={`p-2 text-white rounded-full focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors ${modeColors[mode].button} ${modeColors[mode].buttonHover} ${modeColors[mode].ring}`}
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

      {/* Expanded Card Modal */}
      <AnimatePresence>
        {modalCard && (
          <ExpandedCardModal
            card={modalCard}
            onClose={() => setModalCard(null)}
            onCopy={handleCopy}
            onRegenerate={handleRegenerate}
            isLoading={isLoading}
          />
        )}
      </AnimatePresence>
    </div>
  );
}
