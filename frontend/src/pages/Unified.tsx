import { useState, useEffect, useMemo } from 'react';
import { AnimatePresence } from 'framer-motion';
import { toast } from 'react-hot-toast';
import { Gear, Plus, Cpu, Trash, Database, GridFour, List } from '@phosphor-icons/react';
import { useAuth } from '../contexts/AuthContext';
import { memoryApi, todoApi, ragApi, aiProviderApi, userDataApi, chatApi } from '../api';
import type { Memory, Todo, RAGAskResponse, RAGSearchResult, UploadJobStatusResponse, AskMode } from '../types';
import type { AIProvider, AIProviderModel, AIProviderCreate, DataStats } from '../api';
import { trackEvent } from '../utils/analytics';
import UnifiedGrid from '../components/UnifiedGrid';
import UnifiedList from '../components/UnifiedList';
import CreateNoteModal from '../components/CreateNoteModal';
import NoteDetailModal from '../components/NoteDetailModal';
import ImportModal from '../components/ImportModal';
import OnboardingModal from '../components/OnboardingModal';
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
  mode?: AskMode; // Store the mode this message was created with
}

type ActiveTab = 'mems' | 'todos';
type ViewMode = 'grid' | 'list';

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
  const [viewMode, setViewMode] = useState<ViewMode>(() => {
    // Load from localStorage, default to list on mobile, grid on desktop
    const saved = localStorage.getItem('todo_view_mode');
    if (saved === 'list' || saved === 'grid') return saved;
    // Default: list on mobile (< 640px), grid on desktop
    const isMobile = typeof window !== 'undefined' && window.innerWidth < 640;
    return isMobile ? 'list' : 'grid';
  });
  
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
  const [askMode, setAskMode] = useState<AskMode>('memories');
  const [currentThreadId, setCurrentThreadId] = useState<string | null>(null);

  // Upload status
  const [uploadStatus, setUploadStatus] = useState<UploadJobStatusResponse | null>(null);

  // Onboarding
  const [showOnboarding, setShowOnboarding] = useState(() => {
    return !localStorage.getItem('memlane_onboarding_complete');
  });

  // Apply fixed positioning for full-height layout (only on Unified page)
  useEffect(() => {
    document.body.style.position = 'fixed';
    document.body.style.width = '100%';
    document.body.style.height = '100%';
    document.body.style.overflow = 'hidden';
    document.documentElement.style.overflow = 'hidden';
    document.documentElement.style.height = '100%';
    
    // Cleanup: restore normal scrolling when component unmounts
    return () => {
      document.body.style.position = '';
      document.body.style.width = '';
      document.body.style.height = '';
      document.body.style.overflow = '';
      document.documentElement.style.overflow = '';
      document.documentElement.style.height = '';
    };
  }, []);

  // Save view mode to localStorage when it changes
  useEffect(() => {
    if (activeTab === 'todos') {
      localStorage.setItem('todo_view_mode', viewMode);
    }
  }, [viewMode, activeTab]);

  // Load initial data
  useEffect(() => {
    if (user) {
      fetchData();
      loadChatThread();
    }
  }, [user]);

  // Load chat thread on mount
  const loadChatThread = async () => {
    try {
      const response = await chatApi.getActiveThread();
      setCurrentThreadId(response.thread.id);
      
      // Convert backend messages to frontend Message format
      const loadedMessages: Message[] = response.messages.map((msg) => ({
        id: msg.id,
        type: msg.role as 'user' | 'assistant',
        content: msg.content,
        timestamp: new Date(msg.created_at),
        mode: msg.mode as AskMode | undefined,
        sources: msg.sources ? JSON.parse(msg.sources) : undefined,
      }));
      
      setAskMessages(loadedMessages);
    } catch (error) {
      console.error('Failed to load chat thread:', error);
      // If no thread exists, one will be created on first message
    }
  };

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
      // Otherwise show all memories sorted by position
      return [...memories].sort((a, b) => {
        return parseInt(a.position) - parseInt(b.position);
      });
    } else {
      // If there are citations, only show citations (clear grid)
      if (todoCitations.length > 0) {
        return todoCitations;
      }
      // Sort todos: pending first, completed last
      const sortedTodos = [...todos].sort((a, b) => {
        // Only sort actual Todo items, not PendingTodo
        if ('isProcessing' in a || 'isProcessing' in b) {
          return 0; // Keep processing items in place
        }
        
        const todoA = a as Todo;
        const todoB = b as Todo;
        
        // Pending todos come first
        if (todoA.status === 'pending' && todoB.status === 'completed') return -1;
        if (todoA.status === 'completed' && todoB.status === 'pending') return 1;
        
        // Within same status, sort by position (for pending) or updated_at (for completed)
        if (todoA.status === 'pending' && todoB.status === 'pending') {
          return parseInt(todoA.position) - parseInt(todoB.position);
        }
        
        // Completed todos: most recently completed first
        if (todoA.status === 'completed' && todoB.status === 'completed') {
          return new Date(todoB.updated_at).getTime() - new Date(todoA.updated_at).getTime();
        }
        
        return 0;
      });
      
      return sortedTodos;
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
          position: '0',
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

        // Track analytics
        trackEvent('memory_created', {
          has_category: newMemory.category !== 'Uncategorized',
          category: newMemory.category,
        });

        // Auto-clear Ask AI results when adding new memory (if there's an active search)
        if (askMessages.length > 0) {
          setAskMessages([]);
          setMemoryCitations([]);
          setTodoCitations([]);
        }

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
        
        // Track analytics
        trackEvent('todo_created', {
          has_due_date: !!newTodo.due_date,
          has_group: !!newTodo.group_id,
          priority: newTodo.priority || 'medium',
        });
        
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

  // Handle quick status toggle for todos
  const handleQuickStatusToggle = async (todoId: string, currentStatus: 'pending' | 'completed') => {
    const newStatus: 'pending' | 'completed' = currentStatus === 'pending' ? 'completed' : 'pending';
    await handleUpdateItem(todoId, { status: newStatus });
  };

  // Handle memory reorder
  const handleMemoryReorder = async (reorderedMemories: Memory[]) => {
    try {
      // Calculate positions: 1000, 2000, 3000...
      const reorderData = reorderedMemories.map((memory, index) => ({
        id: memory.id,
        position: String((index + 1) * 1000),
      }));

      // Call API to save new positions
      await memoryApi.reorder({ memories: reorderData });
      
      // Update local state with memories that have new positions
      const updatedMemories = memories.map((memory) => {
        const positionData = reorderData.find(r => r.id === memory.id);
        if (positionData) {
          // Update position for reordered memories
          const reorderedMemory = reorderedMemories.find(m => m.id === memory.id);
          return {
            ...(reorderedMemory || memory),
            position: positionData.position,
          };
        }
        // Keep memories that weren't reordered unchanged
        return memory;
      });
      
      setMemories(updatedMemories);
    } catch (error) {
      console.error('Failed to reorder memories:', error);
      toast.error('Failed to reorder memories');
      // Refresh to get correct order from server
      await fetchData();
    }
  };

  // Handle todo reorder
  const handleTodoReorder = async (reorderedTodos: Todo[]) => {
    try {
      // Separate pending and completed from reordered todos to maintain their separation
      const pendingTodos = reorderedTodos.filter(t => t.status === 'pending');
      const completedTodos = reorderedTodos.filter(t => t.status === 'completed');
      
      // Calculate positions: pending get 1000, 2000, 3000... and completed get higher positions
      const reorderData: Array<{ id: string; position: string }> = [];
      
      // Add pending todos with positions starting at 1000
      pendingTodos.forEach((todo, index) => {
        reorderData.push({
          id: todo.id,
          position: String((index + 1) * 1000),
        });
      });
      
      // Add completed todos with positions after pending (e.g., if 3 pending, start at 4000)
      const completedStartPosition = (pendingTodos.length + 1) * 1000;
      completedTodos.forEach((todo, index) => {
        reorderData.push({
          id: todo.id,
          position: String(completedStartPosition + (index * 1000)),
        });
      });

      // Call API to save new positions
      await todoApi.reorder({ todos: reorderData });
      
      // Update local state with todos that have new positions
      const updatedTodos = todos.map((todo) => {
        const positionData = reorderData.find(r => r.id === todo.id);
        if (positionData) {
          // Update position for reordered todos
          const reorderedTodo = reorderedTodos.find(t => t.id === todo.id);
          return {
            ...(reorderedTodo || todo),
            position: positionData.position,
          };
        }
        // Keep todos that weren't reordered unchanged
        return todo;
      });
      
      setTodos(updatedTodos);
    } catch (error) {
      console.error('Failed to reorder todos:', error);
      toast.error('Failed to reorder todos');
      // Refresh to get correct order from server
      await fetchData();
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

    // Auto-clear Ask AI results when importing (if there's an active search)
    if (askMessages.length > 0) {
      setAskMessages([]);
      setMemoryCitations([]);
      setTodoCitations([]);
    }
  };

  // Parse command from query (only slash commands: /save, /memory, /todo, /task)
  const parseCommand = (query: string): { command: 'memory' | 'todo' | null; content: string } => {
    const trimmed = query.trim();
    
    // Memory commands - Slash commands only
    if (trimmed.startsWith('/save ') || trimmed.startsWith('/memory ') || trimmed.startsWith('/remember ') || trimmed.startsWith('/note ')) {
      return { command: 'memory', content: trimmed.replace(/^\/save\s+|\/memory\s+|\/remember\s+|\/note\s+/, '').trim() };
    }
    
    // Todo commands - Slash commands
    if (trimmed.startsWith('/todo ') || trimmed.startsWith('/task ') || trimmed.startsWith('/add ')) {
      return { command: 'todo', content: trimmed.replace(/^\/todo\s+|\/task\s+|\/add\s+/, '').trim() };
    }
    
    // No command detected - will proceed with RAG query
    return { command: null, content: query };
  };

  // Submit query to Ask AI
  const submitQuery = async (query: string, skipUserMessage: boolean = false) => {
    // Check for save commands first - BEFORE any RAG processing
    const { command, content } = parseCommand(query);
    
    // Debug logging in development
    if (import.meta.env.DEV) {
      console.log('[Command Parser] Query:', query);
      console.log('[Command Parser] Detected command:', command);
      console.log('[Command Parser] Extracted content:', content);
    }
    
    if (command === 'memory' && content) {
      // Save as memory directly
      console.log('[Command Parser] Saving as memory:', content);
      await handleSaveAsMemory(content);
      return; // Don't proceed with RAG query
    }
    
    if (command === 'todo' && content) {
      // Save as todo directly
      console.log('[Command Parser] Saving as todo:', content);
      await handleSaveAsTodo(content);
      return; // Don't proceed with RAG query
    }
    
    // Clear citations when starting a NEW query
    if (!skipUserMessage) {
      setMemoryCitations([]);
      setTodoCitations([]);
    }

    const loadingId = (Date.now() + 1).toString();

    // Ensure we have a thread ID
    let threadId = currentThreadId;
    if (!threadId) {
      try {
        const threadResponse = await chatApi.getActiveThread();
        threadId = threadResponse.thread.id;
        setCurrentThreadId(threadId);
      } catch (error) {
        console.error('Failed to get/create thread:', error);
        // Continue without persistence if thread creation fails
      }
    }

    // For new queries: add new user message + loading (keep existing messages for stack)
    // For regenerate: just add loading message to existing
    if (!skipUserMessage) {
      const userMessage: Message = {
        id: Date.now().toString(),
        type: 'user',
        content: query,
        timestamp: new Date(),
        mode: askMode, // Store the mode with the user message
      };
      
      // Save user message to backend
      if (threadId) {
        try {
          await chatApi.addMessage(threadId, {
            role: 'user',
            content: query,
            mode: askMode,
          });
        } catch (error) {
          console.error('Failed to save user message:', error);
        }
      }
      
      // Add new messages to existing (stack effect)
      setAskMessages((prev) => [
        ...prev,
        userMessage,
        {
          id: loadingId,
          type: 'assistant',
          content: '',
          timestamp: new Date(),
          isLoading: true,
          mode: askMode, // Store the mode with the assistant message too
        },
      ]);
    } else {
      // Regenerate: just add loading message to existing
      setAskMessages((prev) => [
        ...prev,
        {
          id: loadingId,
          type: 'assistant',
          content: '',
          timestamp: new Date(),
          isLoading: true,
          mode: askMode, // Store the mode
        },
      ]);
    }
    setIsAskLoading(true);

    try {
      // Track chat query
    trackEvent('chat_query_submitted', {
        mode: askMode,
        has_conversation_history: askMessages.length > 0,
      });

      const response: RAGAskResponse = await ragApi.ask({
        question: query,
        max_context: 5,
        mode: askMode,
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

      // Save assistant message to backend
      if (threadId) {
        try {
          await chatApi.addMessage(threadId, {
            role: 'assistant',
            content: response.answer,
            mode: askMode,
            sources: response.sources ? JSON.stringify(response.sources) : undefined,
          });
        } catch (error) {
          console.error('Failed to save assistant message:', error);
        }
      }

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

  // Handle save as memory from chat
  const handleSaveAsMemory = async (content: string) => {
    if (!content.trim()) {
      toast.error('No content to save');
      return;
    }

    try {
      const newMemory = await memoryApi.createFromChat({
        content: content.trim(),
      });
      
      // Add to memories list
      setMemories((prev) => [newMemory, ...prev]);
      
      // Track analytics
      trackEvent('memory_saved_from_chat', {
        has_category: newMemory.category !== 'Uncategorized',
        category: newMemory.category,
      });
      
      toast.success('Saved as memory');
    } catch (error) {
      console.error('Failed to save memory from chat:', error);
      toast.error('Failed to save as memory');
    }
  };

  // Handle save as todo from chat
  const handleSaveAsTodo = async (content: string) => {
    if (!content.trim()) {
      toast.error('No content to save');
      return;
    }

    try {
      const newTodo = await todoApi.createFromChat({
        content: content.trim(),
      });
      
      // Add to todos list
      setTodos((prev) => [...prev, newTodo]);
      
      // Track analytics
      trackEvent('todo_saved_from_chat', {
        has_due_date: !!newTodo.due_date,
        has_group: !!newTodo.group_id,
        priority: newTodo.priority || 'medium',
      });
      
      toast.success('Saved as todo');
    } catch (error) {
      console.error('Failed to save todo from chat:', error);
      toast.error('Failed to save as todo');
    }
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
        <img src="/logo-white.png" alt="Mr.Brain" className="w-6 h-6" />
      </div>
      <span className="text-xl font-heading text-gray-800 dark:text-white hidden sm:inline">memlane</span>
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
          
          <div className="flex items-center gap-2 sm:gap-4">
            {/* View Toggle - Only show for todos, hidden on mobile */}
            {activeTab === 'todos' && !showSettings && (
              <div className="hidden sm:flex items-center gap-1 bg-surface-light-muted dark:bg-surface-dark-muted rounded-lg p-1">
                <button
                  onClick={() => setViewMode('grid')}
                  className={`p-1.5 rounded transition-colors ${
                    viewMode === 'grid'
                      ? 'bg-primary-600 text-white dark:bg-primary-500'
                      : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200'
                  }`}
                  title="Grid View"
                >
                  <GridFour size={18} weight="regular" />
                </button>
                <button
                  onClick={() => setViewMode('list')}
                  className={`p-1.5 rounded transition-colors ${
                    viewMode === 'list'
                      ? 'bg-primary-600 text-white dark:bg-primary-500'
                      : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200'
                  }`}
                  title="List View"
                >
                  <List size={18} weight="regular" />
                </button>
              </div>
            )}
            
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
          <div className="flex-1 overflow-y-auto pt-6 pb-40 sm:pb-32">
            {loading ? (
              <div className="flex items-center justify-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-600"></div>
              </div>
            ) : (
              <>
                {activeTab === 'todos' && viewMode === 'list' ? (
                  <UnifiedList
                    items={displayItems}
                    type="todo"
                    onCreateClick={() => setIsCreateModalOpen(true)}
                    onImportClick={() => setIsImportModalOpen(true)}
                    onItemClick={handleItemClick}
                    onItemUpdate={handleQuickStatusToggle}
                    onReorder={handleTodoReorder}
                    uploadStatus={uploadStatus}
                  />
                ) : (
                  <UnifiedGrid
                    items={displayItems}
                    type={activeTab === 'mems' ? 'memory' : 'todo'}
                    onCreateClick={() => setIsCreateModalOpen(true)}
                    onImportClick={() => setIsImportModalOpen(true)}
                    onItemClick={handleItemClick}
                    onItemUpdate={activeTab === 'todos' ? handleQuickStatusToggle : undefined}
                    onReorder={activeTab === 'todos' ? handleTodoReorder : activeTab === 'mems' ? handleMemoryReorder : undefined}
                    uploadStatus={uploadStatus}
                  />
                )}
              </>
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
              onSaveAsMemory={handleSaveAsMemory}
              onSaveAsTodo={handleSaveAsTodo}
              onClear={async () => {
                // Delete current thread and create a new one
                if (currentThreadId) {
                  try {
                    await chatApi.deleteThread(currentThreadId);
                  } catch (error) {
                    console.error('Failed to delete thread:', error);
                  }
                }
                
                // Create new thread
                try {
                  const newThread = await chatApi.createThread();
                  setCurrentThreadId(newThread.thread.id);
                } catch (error) {
                  console.error('Failed to create new thread:', error);
                  setCurrentThreadId(null);
                }
                
                setAskMessages([]);
                setMemoryCitations([]);
                setTodoCitations([]);
              }}
              mode={askMode}
              onModeChange={setAskMode}
            />
          </div>
        )}
      </div>

      {/* Modals */}
      <CreateNoteModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onSubmit={handleCreateNote}
        onImageCreated={async () => {
          // Refresh memories when an image is uploaded and processed
          const memoriesData = await memoryApi.getAll();
          setMemories(memoriesData);
        }}
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
        onUploadStatusChange={setUploadStatus}
      />

      <OnboardingModal
        isOpen={showOnboarding}
        onClose={() => setShowOnboarding(false)}
      />
    </div>
  );
}

