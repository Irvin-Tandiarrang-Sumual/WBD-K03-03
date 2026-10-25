import type {
  CreateDatabaseInput,
  Database,
  PagedResponse,
} from "@/lib/types";
import { request } from "@/lib/api/client";

export const databaseApi = {
  list: (page = 1, limit = 20) =>
    request<PagedResponse<Database>>(
      `/api/v1/me/databases?page=${page}&limit=${limit}`,
    ),

  get: (id: number | string) =>
    request<Database>(`/api/v1/me/databases/${id}`),

  listForAdmin: (page = 1, limit = 20) =>
    request<PagedResponse<Database>>(
      `/api/v1/admin/databases?page=${page}&limit=${limit}`,
    ),

  create: (payload: CreateDatabaseInput) =>
    request<Database>("/api/v1/admin/databases", {
      method: "POST",
      body: JSON.stringify(payload),
    }),

  remove: (id: number | string) =>
    request<void>(`/api/v1/admin/databases/${id}`, {
      method: "DELETE",
    }),
};
