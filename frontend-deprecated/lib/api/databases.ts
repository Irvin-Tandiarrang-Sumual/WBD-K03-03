import type {
  CreateDatabaseInput,
  Database,
  PagedResponse,
} from "@/lib/types";
import { request } from "@/lib/api/client";

interface DatabaseResponse {
  data: Database;
}

export const databaseApi = {
  list: (page = 1, limit = 20) =>
    request<PagedResponse<Database>>(
      `/api/v1/me/databases?page=${page}&limit=${limit}`,
    ),

  get: (id: number | string) =>
    request<DatabaseResponse>(`/api/v1/me/databases/${id}`).then(
      (response) => response.data,
    ),

  listForAdmin: (page = 1, limit = 20) =>
    request<PagedResponse<Database>>(
      `/api/v1/admin/databases?page=${page}&limit=${limit}`,
    ),

  create: (payload: CreateDatabaseInput) =>
    request<DatabaseResponse>("/api/v1/admin/databases", {
      method: "POST",
      body: JSON.stringify(payload),
    }).then((response) => response.data),

  remove: (id: number | string) =>
    request<void>(`/api/v1/admin/databases/${id}`, {
      method: "DELETE",
    }),
};
