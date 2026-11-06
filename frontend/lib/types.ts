export type DatabaseType = "postgresql" | "mysql";
export type AccessType = "view_only" | "view_edit";
export type UserRole = "admin" | "user" | string;
export type MigrationStatus = "success" | "fail" | string;

export interface Database {
  id: number;
  name: string;
  host: string;
  port: number;
  user: string;
  type: DatabaseType | string;
  access_type?: AccessType;
  accessType?: AccessType;
}

export interface DatabaseAccess {
  databaseId: number;
  accessType: AccessType;
}

export interface Group {
  id: number;
  name: string;
  databaseAccesses: DatabaseAccess[];
}

export interface User {
  id: number;
  username: string;
  role: UserRole;
}

export interface Migration {
  id: number;
  databaseId?: number;
  database_id: number;
  version: string;
  status: MigrationStatus;
  sqlUp?: string;
  sql_up: string;
  sqlDown?: string;
  sql_down: string;
  errorMessage?: string;
  error_message: string;
  snapshot?: unknown;
  userId?: number;
  user_id: number;
}

export interface Paging {
  currentPage?: number;
  current_page?: number;
  page?: number;
  limit?: number;
  totalPage?: number;
  total_page?: number;
  totalData?: number;
  total_data?: number;
}

export interface PagedResponse<T> {
  data: T[];
  paging?: Paging;
}

export interface QueryRow {
  [column: string]: unknown;
}

export interface QueryResult {
  rows?: QueryRow[];
  columns?: string[];
  [key: string]: unknown;
}

export interface QueryResponse {
  data?: QueryResult;
  [key: string]: unknown;
}

export interface MigrationComparison {
  firstMigration?: Migration;
  first_migration: Migration;
  secondMigration?: Migration;
  second_migration: Migration;
  sqlUpEqual?: boolean;
  sql_up_equal: boolean;
  sqlDownEqual?: boolean;
  sql_down_equal: boolean;
}

export interface LoginResponse {
  id: number;
  username: string;
  role: UserRole;
}

export interface CreateDatabaseInput {
  host: string;
  port: number;
  user: string;
  password: string;
  type: DatabaseType;
  name: string;
}

export interface CreateUserInput {
  username: string;
  password: string;
  groupIds: number[];
}

export interface CreateGroupInput {
  name: string;
  databaseAccesses: DatabaseAccess[];
}

export interface UpdateGroupInput {
  name?: string;
  databaseAccesses?: DatabaseAccess[];
}
