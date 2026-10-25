import type {
  Migration,
  MigrationComparison,
  PagedResponse,
} from "@/lib/types";
import { request } from "@/lib/api/client";

interface MigrationResponse<T> {
  data: T;
}

export const migrationApi = {
  list: (databaseId: number | string, page = 1, limit = 20) =>
    request<PagedResponse<Migration>>(
      `/api/v1/me/databases/${databaseId}/migrations?page=${page}&limit=${limit}`,
    ),

  create: (databaseId: number | string, sqlUp: string) =>
    request<MigrationResponse<Migration>>(
      `/api/v1/me/databases/${databaseId}/migrations`,
      {
        method: "POST",
        body: JSON.stringify({ sql_up: sqlUp }),
      },
    ).then((response) => response.data),

  compare: (
    databaseId: number | string,
    firstMigrationId: number | string,
    secondMigrationId: number | string,
  ) =>
    request<MigrationResponse<MigrationComparison>>(
      `/api/v1/me/databases/${databaseId}/migrations/compare?migration_id_1=${firstMigrationId}&migration_id_2=${secondMigrationId}`,
    ).then((response) => response.data),

  rollback: (databaseId: number, targetMigrationId: number) =>
    request<MigrationResponse<Migration[]>>(
      "/api/v1/me/databases/migrations/rollback",
      {
        method: "POST",
        body: JSON.stringify({
          database_id: databaseId,
          target_migration_id: targetMigrationId,
        }),
      },
    ).then((response) => response.data),
};
