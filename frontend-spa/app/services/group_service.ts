import { api } from './api_client';
import type { DbPermission } from '~/types/database';
import type { AccessGroup, GroupDbAccess } from '~/types/group';

export interface BackendDatabaseAccess {
  databaseId: number;
  accessType: string;
}

export interface BackendGroupResponse {
  id: number;
  name: string;
  leaderId: number;
  memberIds?: number[];
  databaseAccesses: BackendDatabaseAccess[];
}

export function mapBackendGroupToAccessGroup(g: BackendGroupResponse, allMemberIds?: string[]): AccessGroup {
  const dbAccesses: GroupDbAccess[] = (g.databaseAccesses || []).map((da) => ({
    databaseId: String(da.databaseId),
    permission: (da.accessType === 'view_edit' ? 'view_edit' : 'view_only') as DbPermission,
  }));

  const members = new Set<string>();
  if (g.leaderId) {
    members.add(String(g.leaderId));
  }
  if (Array.isArray(g.memberIds)) {
    g.memberIds.forEach((id) => members.add(String(id)));
  }
  if (allMemberIds) {
    allMemberIds.forEach((m) => members.add(m));
  }

  return {
    id: String(g.id),
    name: g.name,
    description: `Access group managing ${dbAccesses.length} database target(s)`,
    leaderId: String(g.leaderId),
    memberIds: Array.from(members),
    databaseAccess: dbAccesses,
  };
}

export const groupService = {
  async getGroups(params: { page?: number; limit?: number } = { page: 1, limit: 100 }): Promise<AccessGroup[]> {
    const query = new URLSearchParams();
    if (params.page) query.set('page', String(params.page));
    if (params.limit) query.set('limit', String(params.limit));
    const qs = query.toString() ? `?${query.toString()}` : '';

    const res = await api.get<BackendGroupResponse[] | { data: BackendGroupResponse[] }>(`/admin/groups${qs}`);
    const items: BackendGroupResponse[] = Array.isArray(res)
      ? res
      : (res && Array.isArray((res as any).data) ? (res as any).data : []);
    return items.map((g) => mapBackendGroupToAccessGroup(g));
  },

  async getGroupById(id: string | number): Promise<AccessGroup> {
    const rawId = typeof id === 'string' && id.startsWith('grp-') ? id.replace('grp-', '') : id;
    const res = await api.get<BackendGroupResponse>(`/admin/groups/${rawId}`);
    return mapBackendGroupToAccessGroup(res);
  },

  async createGroup(payload: {
    name: string;
    leaderId: number;
    databaseAccesses: Array<{ databaseId: number; accessType: string }>;
  }): Promise<AccessGroup> {
    const res = await api.post<BackendGroupResponse>('/admin/groups', payload);
    return mapBackendGroupToAccessGroup(res);
  },

  async updateGroup(
    id: string | number,
    payload: {
      name?: string;
      leaderId?: number;
      databaseAccesses?: Array<{ databaseId: number; accessType: string }>;
    }
  ): Promise<AccessGroup> {
    const rawId = typeof id === 'string' && id.startsWith('grp-') ? id.replace('grp-', '') : id;
    const res = await api.patch<BackendGroupResponse>(`/admin/groups/${rawId}`, payload);
    return mapBackendGroupToAccessGroup(res);
  },

  async deleteGroup(id: string | number): Promise<void> {
    const rawId = typeof id === 'string' && id.startsWith('grp-') ? id.replace('grp-', '') : id;
    await api.delete(`/admin/groups/${rawId}`);
  },

  async addUserToGroup(groupId: string | number, userId: string | number): Promise<void> {
    const rawGroupId = typeof groupId === 'string' && groupId.startsWith('grp-') ? groupId.replace('grp-', '') : groupId;
    const rawUserId = typeof userId === 'string' && userId.startsWith('usr-') ? userId.replace('usr-', '') : userId;
    await api.post(`/admin/groups/${rawGroupId}/users/${rawUserId}`);
  },

  async removeUserFromGroup(
    groupId: string | number,
    userId: string | number,
    replacementLeaderId?: string | number
  ): Promise<void> {
    const rawGroupId = typeof groupId === 'string' && groupId.startsWith('grp-') ? groupId.replace('grp-', '') : groupId;
    const rawUserId = typeof userId === 'string' && userId.startsWith('usr-') ? userId.replace('usr-', '') : userId;
    let url = `/admin/groups/${rawGroupId}/users/${rawUserId}`;
    if (replacementLeaderId !== undefined && replacementLeaderId !== null && replacementLeaderId !== '') {
      const rawRepId = typeof replacementLeaderId === 'string' && replacementLeaderId.startsWith('usr-')
        ? replacementLeaderId.replace('usr-', '')
        : replacementLeaderId;
      url += `?replacement_leader_id=${rawRepId}`;
    }
    await api.delete(url);
  },
};
