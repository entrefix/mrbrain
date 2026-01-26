import client from './client';
import { Todo, TodoCreate, TodoUpdate } from '../types';

export interface TodoReorderRequest {
  todos: Array<{
    id: string;
    position: string;
  }>;
}

export const todoApi = {
  getAll: async (): Promise<Todo[]> => {
    const response = await client.get('/todos');
    return response.data.todos;
  },

  getById: async (id: string): Promise<Todo> => {
    const response = await client.get(`/todos/${id}`);
    return response.data.todo;
  },

  create: async (data: TodoCreate): Promise<Todo> => {
    const response = await client.post('/todos', data);
    return response.data.todo;
  },

  createFromChat: async (data: {
    content: string;
    title?: string;
    description?: string;
    due_date?: string;
    priority?: 'low' | 'medium' | 'high';
    group_id?: string;
  }): Promise<Todo> => {
    const response = await client.post('/todos/from-chat', data);
    return response.data.todo;
  },

  update: async (id: string, data: TodoUpdate): Promise<Todo> => {
    const response = await client.put(`/todos/${id}`, data);
    return response.data.todo;
  },

  delete: async (id: string): Promise<void> => {
    await client.delete(`/todos/${id}`);
  },

  reorder: async (data: TodoReorderRequest): Promise<void> => {
    await client.put('/todos/reorder', data);
  },
};
