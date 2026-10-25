import { adminApi } from "@/lib/api/admin";
import { authApi } from "@/lib/api/auth";
import { databaseApi } from "@/lib/api/databases";
import { executorApi } from "@/lib/api/executor";
import { migrationApi } from "@/lib/api/migrations";

export const api = {
  ...authApi,
  me: authApi.me,
  databases: databaseApi.list,
  adminDatabases: databaseApi.listForAdmin,
  database: databaseApi.get,
  createDatabase: databaseApi.create,
  deleteDatabase: databaseApi.remove,
  users: adminApi.listUsers,
  groups: adminApi.listGroups,
  group: adminApi.getGroup,
  createGroup: adminApi.createGroup,
  updateGroup: adminApi.updateGroup,
  deleteGroup: adminApi.deleteGroup,
  addUserToGroup: adminApi.addUserToGroup,
  removeUserFromGroup: adminApi.removeUserFromGroup,
  query: executorApi.query,
  explain: executorApi.explain,
  migrations: migrationApi.list,
  createMigration: migrationApi.create,
  compareMigrations: migrationApi.compare,
  rollback: migrationApi.rollback,
};

export { request, unwrapPaged } from "@/lib/api/client";
export { adminApi, authApi, databaseApi, executorApi, migrationApi };
