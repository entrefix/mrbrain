import client from './client';
import { User } from '../types';

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
}

export const authApi = {
  register: async (data: RegisterRequest): Promise<User> => {
    const response = await client.post('/auth/register', data);
    return response.data.user;
  },

  login: async (data: LoginRequest): Promise<User> => {
    const response = await client.post('/auth/login', data);
    return response.data.user;
  },

  logout: async (): Promise<void> => {
    await client.post('/auth/logout');
  },

  me: async (): Promise<User> => {
    const response = await client.get('/auth/me');
    return response.data.user;
  },
};
