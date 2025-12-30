import client from './client';
import { Group, GroupCreate, GroupUpdate } from '../types';

export const groupApi = {
  getAll: async (): Promise<Group[]> => {
    const response = await client.get('/groups');
    return response.data.groups;
  },

  getById: async (id: string): Promise<Group> => {
    const response = await client.get(`/groups/${id}`);
    return response.data.group;
  },

  create: async (data: GroupCreate): Promise<Group> => {
    const response = await client.post('/groups', data);
    return response.data.group;
  },

  update: async (id: string, data: GroupUpdate): Promise<Group> => {
    const response = await client.put(`/groups/${id}`, data);
    return response.data.group;
  },

  delete: async (id: string): Promise<void> => {
    await client.delete(`/groups/${id}`);
  },
};
