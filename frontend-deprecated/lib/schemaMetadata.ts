import { sanitizeSqlQuery } from "@/lib/api/executor";
import type { QueryRow } from "@/lib/types";

export interface TableConstraintItem {
  constraintName: string;
  constraintType: string;
  columnName: string;
  foreignReference: string;
}

export interface TableIndexItem {
  indexName: string;
  uniqueness: string;
  indexType: string;
  columnNames: string;
  definition?: string;
}

export function readCell(row: QueryRow, name: string): string {
  const key = Object.keys(row).find(
    (k) => k.toLowerCase() === name.toLowerCase(),
  );
  if (!key || row[key] === null || row[key] === undefined) {
    return "";
  }
  return String(row[key]);
}

export function buildColumnsQuery(escapedTable: string, dbType?: string): string {
  const isMysql = String(dbType || "").toLowerCase() === "mysql";
  if (isMysql) {
    return `SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = '${escapedTable}' ORDER BY ordinal_position`;
  }
  return `SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys') AND table_name = '${escapedTable}' ORDER BY ordinal_position`;
}

export function buildConstraintsQuery(escapedTable: string, dbType?: string): string {
  const isMysql = String(dbType || "").toLowerCase() === "mysql";
  if (isMysql) {
    return `SELECT tc.CONSTRAINT_NAME AS constraint_name, tc.CONSTRAINT_TYPE AS constraint_type, COALESCE(kcu.COLUMN_NAME, '') AS column_name, CASE WHEN kcu.REFERENCED_TABLE_NAME IS NOT NULL THEN CONCAT(kcu.REFERENCED_TABLE_NAME, '.', kcu.REFERENCED_COLUMN_NAME) ELSE '' END AS foreign_reference FROM information_schema.TABLE_CONSTRAINTS tc LEFT JOIN information_schema.KEY_COLUMN_USAGE kcu ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME AND tc.TABLE_SCHEMA = kcu.TABLE_SCHEMA AND tc.TABLE_NAME = kcu.TABLE_NAME WHERE tc.TABLE_SCHEMA = DATABASE() AND tc.TABLE_NAME = '${escapedTable}' ORDER BY tc.CONSTRAINT_TYPE, tc.CONSTRAINT_NAME, kcu.ORDINAL_POSITION`;
  }
  return `SELECT tc.constraint_name, tc.constraint_type, COALESCE(kcu.column_name, '') AS column_name, CASE WHEN tc.constraint_type = 'FOREIGN KEY' AND ccu.table_name IS NOT NULL THEN ccu.table_name || '.' || ccu.column_name ELSE '' END AS foreign_reference FROM information_schema.table_constraints tc LEFT JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema AND tc.table_name = kcu.table_name LEFT JOIN information_schema.constraint_column_usage ccu ON tc.constraint_type = 'FOREIGN KEY' AND tc.constraint_name = ccu.constraint_name AND tc.table_schema = ccu.table_schema WHERE tc.table_schema NOT IN ('pg_catalog', 'information_schema') AND tc.table_name = '${escapedTable}' AND tc.constraint_name NOT LIKE '%_not_null' ORDER BY tc.constraint_type, tc.constraint_name, kcu.ordinal_position`;
}

export function buildIndexesQuery(escapedTable: string, dbType?: string): string {
  const isMysql = String(dbType || "").toLowerCase() === "mysql";
  if (isMysql) {
    return `SELECT INDEX_NAME AS index_name, CASE WHEN NON_UNIQUE = 0 THEN 'UNIQUE' ELSE 'NON-UNIQUE' END AS uniqueness, INDEX_TYPE AS index_type, GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX SEPARATOR ', ') AS column_names FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = '${escapedTable}' GROUP BY INDEX_NAME, NON_UNIQUE, INDEX_TYPE ORDER BY INDEX_NAME`;
  }
  return `SELECT indexname AS index_name, CASE WHEN indexdef ILIKE 'CREATE UNIQUE%' THEN 'UNIQUE' ELSE 'NON-UNIQUE' END AS uniqueness, COALESCE(substring(indexdef FROM 'USING ([a-zA-Z0-9_]+)'), 'btree') AS index_type, COALESCE(substring(indexdef FROM '\\((.+)\\)$'), '') AS column_names, indexdef AS index_definition FROM pg_indexes WHERE schemaname NOT IN ('pg_catalog', 'information_schema') AND tablename = '${escapedTable}' ORDER BY indexname`;
}

export function parseConstraintsRows(rows: QueryRow[]): TableConstraintItem[] {
  return rows.map((row) => ({
    constraintName: readCell(row, "constraint_name"),
    constraintType: readCell(row, "constraint_type").toUpperCase(),
    columnName: readCell(row, "column_name"),
    foreignReference: readCell(row, "foreign_reference"),
  }));
}

export function parseIndexesRows(rows: QueryRow[]): TableIndexItem[] {
  return rows.map((row) => ({
    indexName: readCell(row, "index_name"),
    uniqueness: readCell(row, "uniqueness").toUpperCase() || "NON-UNIQUE",
    indexType: readCell(row, "index_type").toUpperCase() || "BTREE",
    columnNames: readCell(row, "column_names"),
    definition: readCell(row, "index_definition") || undefined,
  }));
}

export function isSelectOnlyQuery(rawSql: string): boolean {
  const cleaned = sanitizeSqlQuery(rawSql, true);
  if (!cleaned || cleaned.includes(";")) {
    return false;
  }
  return /^(SELECT|WITH)(?:[\s*(\r\n\t])/i.test(cleaned);
}

