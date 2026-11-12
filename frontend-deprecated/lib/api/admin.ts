import type {
  CreateGroupInput,
  CreateUserInput,
  Group,
  PagedResponse,
  UpdateGroupInput,
  User,
  UserRole,
} from "@/lib/types";
import { request } from "@/lib/api/client";

export const adminApi = {
  listUsers: (page = 1, limit = 20) =>
    request<PagedResponse<User>>(
      `/api/v1/admin/users?page=${page}&limit=${limit}`,
    ),

  createUser: (payload: CreateUserInput) =>
    request<User>("/api/v1/admin/users", {
      method: "POST",
      body: JSON.stringify(payload),
    }),

  setUserRole: (userId: number | string, role: UserRole) =>
    request<User>(`/api/v1/admin/users/${userId}/role`, {
      method: "PATCH",
      body: JSON.stringify({ role }),
    }),

  deleteUser: (userId: number | string) =>
    request<void>(`/api/v1/admin/users/${userId}`, {
      method: "DELETE",
    }),

  listGroups: (page = 1, limit = 20) =>
    request<PagedResponse<Group>>(
      `/api/v1/admin/groups?page=${page}&limit=${limit}`,
    ),

  getGroup: (id: number | string) =>
    request<Group>(`/api/v1/admin/groups/${id}`),

  createGroup: (payload: CreateGroupInput) =>
    request<Group>("/api/v1/admin/groups", {
      method: "POST",
      body: JSON.stringify(payload),
    }),

  updateGroup: (groupId: number | string, payload: UpdateGroupInput) =>
    request<Group>(`/api/v1/admin/groups/${groupId}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    }),

  deleteGroup: (groupId: number | string) =>
    request<void>(`/api/v1/admin/groups/${groupId}`, {
      method: "DELETE",
    }),

  addUserToGroup: (groupId: number | string, userId: number | string) =>
    request<void>(`/api/v1/admin/groups/${groupId}/users/${userId}`, {
      method: "POST",
    }),

  removeUserFromGroup: (groupId: number | string, userId: number | string) =>
    request<void>(`/api/v1/admin/groups/${groupId}/users/${userId}`, {
      method: "DELETE",
    }),
};

