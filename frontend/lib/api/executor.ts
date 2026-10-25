import type { QueryResponse } from "@/lib/types";
import { request } from "@/lib/api/client";

export function sanitizeSqlQuery(rawQuery: string, stripExplainPrefix = false): string {
  // Remove leading SQL line comments (-- ...) and block comments (/* ... */)
  let cleaned = rawQuery.trim();
  while (cleaned.startsWith("--") || cleaned.startsWith("/*")) {
    if (cleaned.startsWith("--")) {
      const newlineIdx = cleaned.indexOf("\n");
      cleaned = newlineIdx === -1 ? "" : cleaned.slice(newlineIdx + 1).trim();
    } else {
      const endCommentIdx = cleaned.indexOf("*/");
      cleaned = endCommentIdx === -1 ? "" : cleaned.slice(endCommentIdx + 2).trim();
    }
  }

  // Strip trailing semicolons and whitespace so single statements ending with ';' pass validation
  cleaned = cleaned.replace(/;+\s*$/, "").trim();

  if (stripExplainPrefix) {
    cleaned = cleaned
      .replace(/^explain\s*\([^)]*\)\s*/i, "")
      .replace(/^explain\s+analyze\s+/i, "")
      .replace(/^explain\s+/i, "")
      .trim();
  }

  return cleaned;
}

export const executorApi = {
  query: (databaseId: number | string, query: string) =>
    request<QueryResponse>(`/api/v1/me/databases/${databaseId}/query`, {
      method: "POST",
      body: JSON.stringify({ query: sanitizeSqlQuery(query, false) }),
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
        body: JSON.stringify({ query: sanitizeSqlQuery(query, true) }),
      },
    ),
};
