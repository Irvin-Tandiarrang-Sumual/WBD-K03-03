import { api } from './api_client';
import { cleanEntityId, generateUserInitials } from '~/utils/formatters';
import type { PlatformUser, AppRole } from '~/types/auth';

export interface BackendUserResponse {
  id: number;
  username: string;
  role: string;
}

export const AVATAR_COLORS = [
  'bg-[#264D3B] text-[#F5EEDD]',
  'bg-[#1E6B8C] text-[#F5EEDD]',
  'bg-[#F3B738] text-[#3A241A]',
  'bg-[#822433] text-[#F5EEDD]',
];

export function mapBackendUserToPlatformUser(user: BackendUserResponse): PlatformUser {
  const initials = generateUserInitials(user.username);
  const nameParts = user.username.split(/[._-]/);
  const displayName = nameParts
    .map((p) => p.charAt(0).toUpperCase() + p.slice(1))
    .join(' ');

  return {
    id: String(user.id),
    username: user.username,
    displayName: displayName || user.username,
    email: `${user.username.toLowerCase()}@wedebe.dev`,
    role: user.role === 'admin' ? 'admin' : 'user',
    avatarInitials: initials,
    avatarColor: AVATAR_COLORS[user.id % AVATAR_COLORS.length],
  };
}

export const userService = {
  async getUsers(params: { page?: number; limit?: number } = { page: 1, limit: 100 }): Promise<PlatformUser[]> {
    const query = new URLSearchParams();
    if (params.page) query.set('page', String(params.page));
    if (params.limit) query.set('limit', String(params.limit));
    const qs = query.toString() ? `?${query.toString()}` : '';

    const res = await api.get<BackendUserResponse[] | { data: BackendUserResponse[] }>(`/admin/users${qs}`);
    const items = Array.isArray(res) ? res : (res && Array.isArray((res as any).data) ? (res as any).data : []);
    return items.map(mapBackendUserToPlatformUser);
  },

  async getUserById(id: string | number): Promise<PlatformUser> {
    const rawId = cleanEntityId(id, 'usr-');
    const res = await api.get<BackendUserResponse>(`/admin/users/${rawId}`);
    return mapBackendUserToPlatformUser(res);
  },

  async createUser(payload: {
    username: string;
    password?: string;
    role?: 'admin' | 'user';
    groupIds: number[];
  }): Promise<PlatformUser> {
    const defaultPassword = payload.password || `${payload.username}123`;
    const res = await api.post<BackendUserResponse>('/admin/users', {
      username: payload.username.trim().toLowerCase(),
      password: defaultPassword,
      groupIds: payload.groupIds.length > 0 ? payload.groupIds : [1],
    });

    if (payload.role && payload.role !== 'user' && res && res.id) {
      await api.patch<BackendUserResponse>(`/admin/users/${res.id}/role`, { role: payload.role });
      res.role = payload.role;
    }

    return mapBackendUserToPlatformUser(res);
  },

  async updateUser(id: string | number, payload: { username?: string; password?: string }): Promise<PlatformUser> {
    const rawId = cleanEntityId(id, 'usr-');
    const res = await api.patch<BackendUserResponse>(`/admin/users/${rawId}`, payload);
    return mapBackendUserToPlatformUser(res);
  },

  async updateUserRole(id: string | number, role: 'admin' | 'user'): Promise<PlatformUser> {
    const rawId = cleanEntityId(id, 'usr-');
    const res = await api.patch<BackendUserResponse>(`/admin/users/${rawId}/role`, { role });
    return mapBackendUserToPlatformUser(res);
  },

  async deleteUser(id: string | number): Promise<void> {
    const rawId = cleanEntityId(id, 'usr-');
    await api.delete(`/admin/users/${rawId}`);
  },
};
