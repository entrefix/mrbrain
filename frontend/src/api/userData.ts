import client from './client';

export interface DataStats {
  memory_count: number;
  todo_count: number;
  custom_group_count: number;
}

export interface ClearMemoriesResult {
  memories_deleted: number;
  success: boolean;
  error_message?: string;
}

export interface ClearAllResult {
  memories_deleted: number;
  todos_deleted: number;
  custom_groups_deleted: number;
  success: boolean;
  error_message?: string;
}

export const userDataApi = {
  getStats: async (): Promise<DataStats> => {
    const response = await client.get('/user/data/stats');
    return response.data;
  },

  clearMemories: async (): Promise<ClearMemoriesResult> => {
    const response = await client.post('/user/data/clear-memories');
    return response.data;
  },

  clearAll: async (): Promise<ClearAllResult> => {
    const response = await client.post('/user/data/clear-all');
    return response.data;
  },
};
