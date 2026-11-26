import { api, setCSRFToken } from './api_client';
import { mapBackendUserToPlatformUser } from './user_service';
import type { PlatformUser } from '~/types/auth';

export interface AuthUserDTO {
  id: number;
  username: string;
  role: 'admin' | 'user';
  csrf_token?: string;
  csrfToken?: string;
}

export function mapAuthUserToPlatformUser(dto: AuthUserDTO): PlatformUser {
  return mapBackendUserToPlatformUser(dto);
}

export const authService = {
  async login(username: string, password: string): Promise<PlatformUser> {
    const data = await api.post<AuthUserDTO>('/auth/login', {
      username: username.trim(),
      password,
    });

    const token = data.csrfToken || data.csrf_token;
    if (token) {
      setCSRFToken(token);
    }

    return mapAuthUserToPlatformUser(data);
  },

  async getMe(): Promise<PlatformUser> {
    const data = await api.get<AuthUserDTO>('/auth/me');

    const token = data.csrfToken || data.csrf_token;
    if (token) {
      setCSRFToken(token);
    }

    return mapAuthUserToPlatformUser(data);
  },

  async logout(): Promise<void> {
    try {
      await api.post('/auth/logout');
    } finally {
      setCSRFToken('');
    }
  },
};
