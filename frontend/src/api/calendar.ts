import client from './client';

export interface CalendarStatus {
  connected: boolean;
  calendar_email?: string | null;
  is_enabled?: boolean;
}

const calendarApi = {
  getStatus: async (): Promise<CalendarStatus> => {
    const response = await client.get('/calendar/status');
    return response.data;
  },

  initiateOAuth: async (): Promise<{ auth_url: string }> => {
    const response = await client.post('/calendar/connect');
    return response.data;
  },

  disconnect: async (): Promise<void> => {
    await client.delete('/calendar/disconnect');
  },
};

export default calendarApi;
