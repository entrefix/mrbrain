import client from './client';

export interface AIProvider {
  id: string;
  user_id: string;
  name: string;
  provider_type: 'openai' | 'anthropic' | 'google' | 'custom';
  base_url: string;
  api_key_masked?: string;
  selected_model?: string | null;
  is_default: boolean;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface AIProviderModel {
  id: string;
  provider_id: string;
  model_id: string;
  model_name: string;
  created_at: string;
}

export interface AIProviderCreate {
  name: string;
  provider_type: 'openai' | 'anthropic' | 'google' | 'custom';
  base_url: string;
  api_key: string;
  is_default?: boolean;
}

export interface AIProviderUpdate {
  name?: string;
  base_url?: string;
  api_key?: string;
  selected_model?: string | null;
  is_default?: boolean;
  is_enabled?: boolean;
}

export interface TestConnectionRequest {
  provider_type: 'openai' | 'anthropic' | 'google' | 'custom';
  base_url: string;
  api_key: string;
}

export interface TestConnectionResponse {
  success: boolean;
  message: string;
  models?: string[];
}

const aiProviderApi = {
  getAll: async (): Promise<AIProvider[]> => {
    const response = await client.get('/ai-providers');
    return response.data;
  },

  getById: async (id: string): Promise<AIProvider> => {
    const response = await client.get(`/ai-providers/${id}`);
    return response.data;
  },

  create: async (data: AIProviderCreate): Promise<AIProvider> => {
    const response = await client.post('/ai-providers', data);
    return response.data;
  },

  update: async (id: string, data: AIProviderUpdate): Promise<AIProvider> => {
    const response = await client.put(`/ai-providers/${id}`, data);
    return response.data;
  },

  delete: async (id: string): Promise<void> => {
    await client.delete(`/ai-providers/${id}`);
  },

  testConnection: async (data: TestConnectionRequest): Promise<TestConnectionResponse> => {
    const response = await client.post('/ai-providers/test', data);
    return response.data;
  },

  fetchModels: async (id: string): Promise<AIProviderModel[]> => {
    const response = await client.post(`/ai-providers/${id}/fetch-models`);
    return response.data;
  },

  getModels: async (id: string): Promise<AIProviderModel[]> => {
    const response = await client.get(`/ai-providers/${id}/models`);
    return response.data;
  },
};

export default aiProviderApi;

// Default base URLs for each provider type
export const DEFAULT_BASE_URLS: Record<string, string> = {
  openai: 'https://api.openai.com/v1',
  anthropic: 'https://api.anthropic.com/v1',
  google: 'https://generativelanguage.googleapis.com/v1beta',
  custom: '',
};

export const PROVIDER_LABELS: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  google: 'Google',
  custom: 'Custom (OpenAI-compatible)',
};
