import client from './client';
import {
  Memory,
  MemoryCategory,
  MemoryDigest,
  MemoryCreate,
  MemoryUpdate,
  MemorySearchParams,
  MemoryToTodoParams,
  MemoryStats,
  MemoryFileUploadResponse,
  WebSearchResult,
  Todo,
} from '../types';

export const memoryApi = {
  getAll: async (limit = 50, offset = 0): Promise<Memory[]> => {
    const response = await client.get('/memories', { params: { limit, offset } });
    return response.data.memories;
  },

  getById: async (id: string): Promise<Memory> => {
    const response = await client.get(`/memories/${id}`);
    return response.data.memory;
  },

  create: async (data: MemoryCreate): Promise<Memory> => {
    const response = await client.post('/memories', data);
    return response.data.memory;
  },

  update: async (id: string, data: MemoryUpdate): Promise<Memory> => {
    const response = await client.put(`/memories/${id}`, data);
    return response.data.memory;
  },

  delete: async (id: string): Promise<void> => {
    await client.delete(`/memories/${id}`);
  },

  getCategories: async (): Promise<MemoryCategory[]> => {
    const response = await client.get('/memories/categories');
    return response.data.categories;
  },

  getByCategory: async (category: string, limit = 50, offset = 0): Promise<Memory[]> => {
    const response = await client.get(`/memories/category/${encodeURIComponent(category)}`, {
      params: { limit, offset },
    });
    return response.data.memories;
  },

  search: async (params: MemorySearchParams): Promise<Memory[]> => {
    const response = await client.post('/memories/search', params);
    return response.data.memories;
  },

  convertToTodo: async (id: string, data?: MemoryToTodoParams): Promise<Todo> => {
    const response = await client.post(`/memories/${id}/to-todo`, data || {});
    return response.data.todo;
  },

  getDigest: async (): Promise<MemoryDigest | null> => {
    const response = await client.get('/memories/digest');
    return response.data.digest;
  },

  generateDigest: async (): Promise<MemoryDigest> => {
    const response = await client.post('/memories/digest/generate');
    return response.data.digest;
  },

  webSearch: async (query: string): Promise<WebSearchResult[]> => {
    const response = await client.post('/memories/web-search', { query });
    return response.data.results;
  },

  getStats: async (): Promise<MemoryStats> => {
    const response = await client.get('/memories/stats');
    return response.data;
  },

  uploadFile: async (file: File): Promise<MemoryFileUploadResponse> => {
    const formData = new FormData();
    formData.append('file', file);

    const response = await client.post('/memories/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    return response.data;
  },

  // New async upload endpoints
  startUploadJob: async (file: File) => {
    const formData = new FormData();
    formData.append('file', file);

    const response = await client.post('/memories/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    return response.data;
  },

  getUploadJobStatus: async (jobId: string) => {
    const response = await client.get(`/memories/upload/jobs/${jobId}`);
    return response.data;
  },

  reorder: async (data: { memories: Array<{ id: string; position: string }> }): Promise<void> => {
    await client.put('/memories/reorder', data);
  },

  uploadImage: async (file: File): Promise<{ memory: Memory; vision_result: { content: string; summary: string; category: string; tags: string[] } }> => {
    const formData = new FormData();
    formData.append('image', file);

    const response = await client.post('/memories/upload-image', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    return response.data;
  },
};
