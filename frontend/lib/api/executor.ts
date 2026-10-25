import type { QueryResponse } from "@/lib/types";
import { request } from "@/lib/api/client";

export const executorApi = {
  query: (databaseId: number | string, query: string) =>
    request<QueryResponse>(`/api/v1/me/databases/${databaseId}/query`, {
      method: "POST",
      body: JSON.stringify({ query }),
    }),

  explain: (
    databaseId: number | string,
    query: string,
    analyze = false,
  ) =>
    request<QueryResponse>(
      `/api/v1/me/databases/${databaseId}/explain?analyzer=${analyze}`,
      {
        method: "POST",
        body: JSON.stringify({ query }),
      },
    ),
};
