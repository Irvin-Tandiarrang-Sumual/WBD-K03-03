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
  database_id: number;
  version: string;
  status: MigrationStatus;
  sql_up: string;
  sql_down: string;
  error_message: string;
  user_id: number;
}

export interface Paging {
  current_page?: number;
  page?: number;
  limit?: number;
  total_page?: number;
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
  first_migration: Migration;
  second_migration: Migration;
  sql_up_equal: boolean;
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

export interface CreateGroupInput {
  name: string;
  databaseAccesses: DatabaseAccess[];
}
