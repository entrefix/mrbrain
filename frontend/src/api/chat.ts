import client from './client';

export interface ChatThread {
  id: string;
  user_id: string;
  created_at: string;
  updated_at: string;
}

export interface ChatMessage {
  id: string;
  thread_id: string;
  role: 'user' | 'assistant';
  content: string;
  mode?: string;
  sources?: string;
  created_at: string;
}

export interface ChatThreadResponse {
  thread: ChatThread;
  messages: ChatMessage[];
}

export interface ChatThreadsResponse {
  threads: ChatThread[];
}

export interface ChatMessageCreateRequest {
  role: 'user' | 'assistant';
  content: string;
  mode?: string;
  sources?: string;
}

export const chatApi = {
  getActiveThread: async (): Promise<ChatThreadResponse> => {
    const response = await client.get('/chat/threads/active');
    return response.data;
  },

  getAllThreads: async (): Promise<ChatThreadsResponse> => {
    const response = await client.get('/chat/threads');
    return response.data;
  },

  getThread: async (threadId: string): Promise<ChatThreadResponse> => {
    const response = await client.get(`/chat/threads/${threadId}`);
    return response.data;
  },

  createThread: async (): Promise<{ thread: ChatThread }> => {
    const response = await client.post('/chat/threads');
    return response.data;
  },

  addMessage: async (threadId: string, message: ChatMessageCreateRequest): Promise<{ message: ChatMessage }> => {
    const response = await client.post(`/chat/threads/${threadId}/messages`, message);
    return response.data;
  },

  deleteThread: async (threadId: string): Promise<void> => {
    await client.delete(`/chat/threads/${threadId}`);
  },
};

