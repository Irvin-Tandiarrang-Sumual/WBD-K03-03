import type {
  Migration,
  MigrationComparison,
  PagedResponse,
} from "@/lib/types";
import { request } from "@/lib/api/client";

interface MigrationResponse<T> {
  data: T;
}

function normalizeMigration(m: any): Migration {
  if (!m) return m;
  return {
    ...m,
    databaseId: m.databaseId ?? m.database_id,
    database_id: m.databaseId ?? m.database_id,
    sqlUp: m.sqlUp ?? m.sql_up,
    sql_up: m.sqlUp ?? m.sql_up,
    sqlDown: m.sqlDown ?? m.sql_down,
    sql_down: m.sqlDown ?? m.sql_down,
    errorMessage: m.errorMessage ?? m.error_message,
    error_message: m.errorMessage ?? m.error_message,
    userId: m.userId ?? m.user_id,
    user_id: m.userId ?? m.user_id,
  };
}

export const migrationApi = {
  list: (databaseId: number | string, page = 1, limit = 20) =>
    request<PagedResponse<Migration>>(
      `/api/v1/me/databases/${databaseId}/migrations?page=${page}&limit=${limit}`,
    ).then((res) => ({
      ...res,
      data: (res.data || []).map(normalizeMigration),
    })),

  create: (databaseId: number | string, sqlUp: string) =>
    request<MigrationResponse<Migration>>(
      `/api/v1/me/databases/${databaseId}/migrations`,
      {
        method: "POST",
        body: JSON.stringify({ sqlUp }),
      },
    ).then((response) => normalizeMigration(response.data)),

  compare: (
    databaseId: number | string,
    firstMigrationId: number | string,
    secondMigrationId: number | string,
  ) =>
    request<MigrationResponse<any>>(
      `/api/v1/me/databases/${databaseId}/migrations/compare?migration_id_1=${firstMigrationId}&migration_id_2=${secondMigrationId}`,
    ).then((response) => {
      const d = response.data;
      const first = normalizeMigration(d.firstMigration ?? d.first_migration);
      const second = normalizeMigration(d.secondMigration ?? d.second_migration);
      const upEq = d.sqlUpEqual ?? d.sql_up_equal;
      const downEq = d.sqlDownEqual ?? d.sql_down_equal;
      return {
        firstMigration: first,
        first_migration: first,
        secondMigration: second,
        second_migration: second,
        sqlUpEqual: upEq,
        sql_up_equal: upEq,
        sqlDownEqual: downEq,
        sql_down_equal: downEq,
      };
    }),

  rollback: (databaseId: number, targetMigrationId: number) =>
    request<MigrationResponse<Migration[]>>(
      "/api/v1/me/databases/migrations/rollback",
      {
        method: "POST",
        body: JSON.stringify({
          databaseId,
          targetMigrationId,
        }),
      },
    ).then((response) => (response.data || []).map(normalizeMigration)),
};
