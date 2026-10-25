import { adminApi } from "@/lib/api/admin";
import { authApi } from "@/lib/api/auth";
import { unwrapPaged } from "@/lib/api/client";
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
  createUser: adminApi.createUser,
  setUserRole: adminApi.setUserRole,
  deleteUser: adminApi.deleteUser,
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

export { unwrapPaged };
