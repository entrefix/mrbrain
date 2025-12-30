// User types
export interface User {
  id: string;
  email: string;
  full_name: string | null;
  theme: string;
  created_at: string;
}

// Todo types
export type Priority = 'low' | 'medium' | 'high';
export type Status = 'pending' | 'completed';

export interface Todo {
  id: string;
  user_id: string;
  group_id: string | null;
  title: string;
  description: string | null;
  due_date: string | null;
  priority: Priority;
  status: Status;
  position: string;
  tags: string[];
  created_at: string;
  updated_at: string;
}

export interface TodoCreate {
  title: string;
  description?: string | null;
  due_date?: string | null;
  priority?: Priority;
  group_id?: string | null;
}

export interface TodoUpdate {
  title?: string;
  description?: string | null;
  due_date?: string | null;
  priority?: Priority;
  status?: Status;
  group_id?: string | null;
  position?: string;
  tags?: string[];
}

// Group types
export interface Group {
  id: string;
  user_id: string | null;
  name: string;
  color_code: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface GroupCreate {
  name: string;
  color_code?: string;
}

export interface GroupUpdate {
  name?: string;
  color_code?: string;
}

// Memory types
export interface Memory {
  id: string;
  user_id: string;
  content: string;
  summary: string | null;
  category: string;
  url: string | null;
  url_title: string | null;
  url_content: string | null;
  is_archived: boolean;
  created_at: string;
  updated_at: string;
}

export interface MemoryCategory {
  id: string;
  user_id: string | null;
  name: string;
  color_code: string;
  icon: string | null;
  is_system: boolean;
  created_at: string;
}

export interface MemoryDigest {
  id: string;
  user_id: string;
  week_start: string;
  week_end: string;
  digest_content: string;
  created_at: string;
}

export interface MemoryCreate {
  content: string;
}

export interface MemoryUpdate {
  content?: string;
  category?: string;
  is_archived?: boolean;
}

export interface MemorySearchParams {
  query?: string;
  category?: string;
  date_from?: string;
  date_to?: string;
  limit?: number;
  offset?: number;
}

export interface MemoryToTodoParams {
  title?: string;
  description?: string;
  priority?: Priority;
  group_id?: string;
}

export interface MemoryStats {
  total: number;
  by_category: Record<string, number>;
  this_week: number;
  this_month: number;
}

export interface WebSearchResult {
  title: string;
  url: string;
  snippet: string;
}

// RAG types
export interface RAGDocument {
  id: string;
  content_type: 'todo' | 'memory';
  content_id: string;
  user_id: string;
  title: string;
  content: string;
  metadata: Record<string, string>;
}

export interface RAGSearchResult {
  document: RAGDocument;
  score: number;
  match_type: string;
  highlights: string[];
}

export interface RAGAskResponse {
  answer: string;
  sources: RAGSearchResult[];
  model: string;
}

export interface RAGStats {
  total_documents: number;
  todos_indexed: number;
  memories_indexed: number;
  fts_enabled: boolean;
}

// Legacy type exports for backward compatibility during migration
export type Database = {
  public: {
    Tables: {
      todos: {
        Row: Todo;
        Insert: TodoCreate & { user_id: string };
        Update: TodoUpdate;
      };
      groups: {
        Row: Group;
        Insert: GroupCreate & { user_id: string | null };
        Update: GroupUpdate;
      };
    };
  };
};
