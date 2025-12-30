import client from './client';
import type { RAGSearchResult, RAGAskResponse, RAGStats } from '../types';

export interface RAGSearchParams {
  query: string;
  content_types?: ('todo' | 'memory')[];
  limit?: number;
  semantic_weight?: number;
}

export interface RAGAskParams {
  question: string;
  content_types?: ('todo' | 'memory')[];
  max_context?: number;
}

export const ragApi = {
  search: async (params: RAGSearchParams): Promise<RAGSearchResult[]> => {
    const response = await client.post('/rag/search', params);
    return response.data.results;
  },

  ask: async (params: RAGAskParams): Promise<RAGAskResponse> => {
    const response = await client.post('/rag/ask', params);
    return response.data;
  },

  indexAll: async (): Promise<{ indexed: number }> => {
    const response = await client.post('/rag/index');
    return response.data;
  },

  getStats: async (): Promise<RAGStats> => {
    const response = await client.get('/rag/stats');
    return response.data;
  },
};
