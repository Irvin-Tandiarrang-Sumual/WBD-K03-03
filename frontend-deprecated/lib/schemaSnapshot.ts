import { api } from "@/lib/api";
import { readCell } from "@/lib/schemaMetadata";
import type { Migration, QueryRow } from "@/lib/types";

export interface SchemaColumnSnapshot {
  name: string;
  type: string;
  nullable: boolean;
}

export interface SchemaIndexSnapshot {
  name: string;
  unique: boolean;
  primary: boolean;
}

export interface SchemaTableSnapshot {
  schema?: string;
  columns: SchemaColumnSnapshot[];
  indexes?: SchemaIndexSnapshot[];
}

export interface SchemaSnapshot {
  tables: Record<string, SchemaTableSnapshot>;
}

export interface ColumnDiffChange {
  kind: "added" | "removed" | "modified";
  tableName: string;
  columnName: string;
  before?: SchemaColumnSnapshot;
  after?: SchemaColumnSnapshot;
  description: string;
}

export interface IndexDiffChange {
  kind: "added" | "removed";
  tableName: string;
  indexName: string;
  index: SchemaIndexSnapshot;
  description: string;
}

export interface TableDiffChange {
  tableName: string;
  status: "added" | "removed" | "modified";
  addedColumns: SchemaColumnSnapshot[];
  removedColumns: SchemaColumnSnapshot[];
  modifiedColumns: {
    before: SchemaColumnSnapshot;
    after: SchemaColumnSnapshot;
  }[];
  addedIndexes: SchemaIndexSnapshot[];
  removedIndexes: SchemaIndexSnapshot[];
}

export interface SnapshotDiffSummaryItem {
  type: "table_added" | "table_removed" | "column_added" | "column_removed" | "column_modified" | "index_added" | "index_removed";
  tableName: string;
  targetName: string;
  detail: string;
  badgeText: string;
}

export interface SnapshotDiffResult {
  isIdentical: boolean;
  summaryItems: SnapshotDiffSummaryItem[];
  tableDiffs: TableDiffChange[];
}

function readRows(body: { data?: { rows?: QueryRow[] } }): QueryRow[] {
  return Array.isArray(body.data?.rows) ? body.data.rows : [];
}

export function cloneSnapshot(snapshot: SchemaSnapshot): SchemaSnapshot {
  return JSON.parse(JSON.stringify(snapshot)) as SchemaSnapshot;
}

export async function fetchLiveDatabaseSnapshot(
  databaseId: string,
  dbType?: string,
): Promise<SchemaSnapshot> {
  const isMysql = String(dbType || "").toLowerCase() === "mysql";

  const tablesQuery = isMysql
    ? "SELECT table_schema, table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE' ORDER BY table_name"
    : "SELECT table_schema, table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys') ORDER BY table_schema, table_name";

  const columnsQuery = isMysql
    ? "SELECT table_schema, table_name, column_name, data_type, is_nullable FROM information_schema.columns WHERE table_schema = DATABASE() ORDER BY table_name, ordinal_position"
    : "SELECT table_schema, table_name, column_name, data_type, is_nullable FROM information_schema.columns WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys') ORDER BY table_schema, table_name, ordinal_position";

  const indexesQuery = isMysql
    ? "SELECT table_schema, table_name, index_name, CASE WHEN non_unique = 0 THEN 'YES' ELSE 'NO' END AS is_unique, CASE WHEN index_name = 'PRIMARY' THEN 'YES' ELSE 'NO' END AS is_primary FROM information_schema.statistics WHERE table_schema = DATABASE() ORDER BY table_name, index_name, seq_in_index"
    : "SELECT schemaname AS table_schema, tablename AS table_name, indexname AS index_name, CASE WHEN indexdef ILIKE 'CREATE UNIQUE%' THEN 'YES' ELSE 'NO' END AS is_unique, CASE WHEN indexname LIKE '%_pkey' THEN 'YES' ELSE 'NO' END AS is_primary FROM pg_indexes WHERE schemaname NOT IN ('pg_catalog', 'information_schema') ORDER BY schemaname, tablename, indexname";

  const [tablesRes, columnsRes, indexesRes] = await Promise.all([
    api.query(databaseId, tablesQuery).catch(() => ({ data: { rows: [] } })),
    api.query(databaseId, columnsQuery).catch(() => ({ data: { rows: [] } })),
    api.query(databaseId, indexesQuery).catch(() => ({ data: { rows: [] } })),
  ]);

  const tables: Record<string, SchemaTableSnapshot> = {};

  for (const row of readRows(tablesRes)) {
    const name = readCell(row, "table_name");
    const schema = readCell(row, "table_schema");
    if (!name) continue;
    tables[name] = {
      schema: schema || undefined,
      columns: [],
      indexes: [],
    };
  }

  for (const row of readRows(columnsRes)) {
    const tableName = readCell(row, "table_name");
    const colName = readCell(row, "column_name");
    const dataType = readCell(row, "data_type");
    const isNullable = readCell(row, "is_nullable").toUpperCase() === "YES" || readCell(row, "is_nullable") === "1" || readCell(row, "is_nullable").toLowerCase() === "true";
    const schema = readCell(row, "table_schema");
    if (!tableName || !colName) continue;

    if (!tables[tableName]) {
      tables[tableName] = {
        schema: schema || undefined,
        columns: [],
        indexes: [],
      };
    }
    tables[tableName].columns.push({
      name: colName,
      type: dataType,
      nullable: isNullable,
    });
  }

  for (const row of readRows(indexesRes)) {
    const tableName = readCell(row, "table_name");
    const indexName = readCell(row, "index_name");
    const isUnique = ["YES", "TRUE", "1"].includes(readCell(row, "is_unique").toUpperCase());
    const isPrimary = ["YES", "TRUE", "1"].includes(readCell(row, "is_primary").toUpperCase());
    if (!tableName || !indexName || !tables[tableName]) continue;

    const existingIndexes = tables[tableName].indexes ?? [];
    if (!existingIndexes.some((idx) => idx.name === indexName)) {
      existingIndexes.push({
        name: indexName,
        unique: isUnique,
        primary: isPrimary,
      });
      tables[tableName].indexes = existingIndexes;
    }
  }

  return { tables };
}

export function formatSnapshotAsBaselineSQL(snapshot: SchemaSnapshot): string {
  const tableNames = Object.keys(snapshot.tables).sort();
  if (tableNames.length === 0) {
    return "-- Initial Schema Snapshot (Empty Database)";
  }

  const blocks = tableNames.map((tableName) => {
    const tbl = snapshot.tables[tableName];
    const colLines = tbl.columns.map((col, idx) => {
      const nullText = col.nullable ? "NULL" : "NOT NULL";
      const comma = idx < tbl.columns.length - 1 ? "," : "";
      return `  ${col.name} ${col.type} ${nullText}${comma}`;
    });
    return `CREATE TABLE ${tableName} (\n${colLines.join("\n")}\n);`;
  });

  return `-- Initial Schema Snapshot (Migration #1 - Describe Tables)\n${blocks.join("\n\n")}`;
}

export function parseMigrationSnapshot(raw: unknown): SchemaSnapshot | null {
  if (!raw) return null;
  try {
    const obj = typeof raw === "string" ? JSON.parse(raw) : raw;
    if (obj && typeof obj === "object" && "tables" in obj && obj.tables && typeof obj.tables === "object") {
      return obj as SchemaSnapshot;
    }
  } catch {
    // ignore invalid JSON
  }
  return null;
}

/**
 * Applies a single SQL statement onto a SchemaSnapshot in-place (forward execution).
 */
export function applySqlToSnapshot(snapshot: SchemaSnapshot, sqlScript: string): void {
  const statements = sqlScript
    .split(";")
    .map((s) => s.trim())
    .filter(Boolean);

  for (const stmt of statements) {
    // 1. CREATE TABLE <name> (col1 type1, col2 type2, ...)
    const createTableMatch = stmt.match(
      /^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_."`]+)\s*\(([\s\S]+)\)$/i,
    );
    if (createTableMatch) {
      const rawTableName = createTableMatch[1].replace(/["`]/g, "").split(".").pop() || "";
      const body = createTableMatch[2];
      const columns: SchemaColumnSnapshot[] = [];
      const lines = body
        .split(",")
        .map((l) => l.trim())
        .filter(Boolean);

      for (const line of lines) {
        if (/^(CONSTRAINT|PRIMARY\s+KEY|FOREIGN\s+KEY|UNIQUE|CHECK)\b/i.test(line)) {
          continue;
        }
        const parts = line.split(/\s+/);
        if (parts.length >= 2) {
          const colName = parts[0].replace(/["`]/g, "");
          const colType = parts[1].toLowerCase();
          const nullable = !/\bNOT\s+NULL\b|\bPRIMARY\s+KEY\b/i.test(line);
          columns.push({ name: colName, type: colType, nullable });
        }
      }
      if (rawTableName) {
        snapshot.tables[rawTableName] = {
          columns,
          indexes: [],
        };
      }
      continue;
    }

    // 2. DROP TABLE <name>
    const dropTableMatch = stmt.match(
      /^DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?([a-zA-Z0-9_."`]+)/i,
    );
    if (dropTableMatch) {
      const rawTableName = dropTableMatch[1].replace(/["`]/g, "").split(".").pop() || "";
      delete snapshot.tables[rawTableName];
      continue;
    }

    // 3. ALTER TABLE <table> ADD [COLUMN] <col> <type>
    const addColMatch = stmt.match(
      /^ALTER\s+TABLE\s+([a-zA-Z0-9_."`]+)\s+ADD\s+(?:COLUMN\s+)?([a-zA-Z0-9_"`]+)\s+([a-zA-Z0-9_()]+)(.*)$/i,
    );
    if (addColMatch) {
      const tableName = addColMatch[1].replace(/["`]/g, "").split(".").pop() || "";
      const colName = addColMatch[2].replace(/["`]/g, "");
      const colType = addColMatch[3].toLowerCase();
      const rest = addColMatch[4] || "";
      const nullable = !/\bNOT\s+NULL\b/i.test(rest);

      if (!snapshot.tables[tableName]) {
        snapshot.tables[tableName] = { columns: [], indexes: [] };
      }
      const existing = snapshot.tables[tableName].columns.find(
        (c) => c.name.toLowerCase() === colName.toLowerCase(),
      );
      if (!existing) {
        snapshot.tables[tableName].columns.push({
          name: colName,
          type: colType,
          nullable,
        });
      }
      continue;
    }

    // 4. ALTER TABLE <table> DROP [COLUMN] <col>
    const dropColMatch = stmt.match(
      /^ALTER\s+TABLE\s+([a-zA-Z0-9_."`]+)\s+DROP\s+(?:COLUMN\s+)?([a-zA-Z0-9_"`]+)/i,
    );
    if (dropColMatch) {
      const tableName = dropColMatch[1].replace(/["`]/g, "").split(".").pop() || "";
      const colName = dropColMatch[2].replace(/["`]/g, "");
      if (snapshot.tables[tableName]) {
        snapshot.tables[tableName].columns = snapshot.tables[
          tableName
        ].columns.filter((c) => c.name.toLowerCase() !== colName.toLowerCase());
      }
      continue;
    }

    // 5. CREATE [UNIQUE] INDEX <idx> ON <table> (...)
    const createIdxMatch = stmt.match(
      /^CREATE\s+(UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_"`]+)\s+ON\s+([a-zA-Z0-9_."`]+)/i,
    );
    if (createIdxMatch) {
      const isUnique = Boolean(createIdxMatch[1]);
      const idxName = createIdxMatch[2].replace(/["`]/g, "");
      const tableName = createIdxMatch[3].replace(/["`]/g, "").split(".").pop() || "";
      if (snapshot.tables[tableName]) {
        const indexes = snapshot.tables[tableName].indexes ?? [];
        if (!indexes.some((i) => i.name.toLowerCase() === idxName.toLowerCase())) {
          indexes.push({ name: idxName, unique: isUnique, primary: false });
          snapshot.tables[tableName].indexes = indexes;
        }
      }
      continue;
    }

    // 6. DROP INDEX <idx> [ON <table>]
    const dropIdxMatch = stmt.match(
      /^DROP\s+INDEX\s+(?:IF\s+EXISTS\s+)?([a-zA-Z0-9_"`]+)(?:\s+ON\s+([a-zA-Z0-9_."`]+))?/i,
    );
    if (dropIdxMatch) {
      const idxName = dropIdxMatch[1].replace(/["`]/g, "");
      for (const tbl of Object.values(snapshot.tables)) {
        if (tbl.indexes) {
          tbl.indexes = tbl.indexes.filter(
            (i) => i.name.toLowerCase() !== idxName.toLowerCase(),
          );
        }
      }
    }
  }
}

/**
 * Ensures every migration has a SchemaSnapshot and synthesizes a Baseline Migration (Version 1 / 0)
 * if the database does not yet have an initial describe-tables migration.
 */
export function enrichMigrationsWithSnapshots(
  rawMigrations: Migration[],
  liveSnapshot: SchemaSnapshot,
  databaseId: number,
): {
  migrations: Migration[];
  snapshotsById: Record<number, SchemaSnapshot>;
} {
  const snapshotsById: Record<number, SchemaSnapshot> = {};

  // Check if any migration already has a backend snapshot
  for (const m of rawMigrations) {
    const parsed = parseMigrationSnapshot(m.snapshot);
    if (parsed) {
      snapshotsById[m.id] = parsed;
    }
  }

  // Walk backwards from liveSnapshot for successful migrations that don't have a stored snapshot
  const successful = rawMigrations.filter((m) => m.status === "success");
  let cursorSnapshot = cloneSnapshot(liveSnapshot);

  for (let i = successful.length - 1; i >= 0; i--) {
    const current = successful[i];
    if (!snapshotsById[current.id]) {
      snapshotsById[current.id] = cloneSnapshot(cursorSnapshot);
    } else {
      cursorSnapshot = cloneSnapshot(snapshotsById[current.id]);
    }
    // Step backwards using sql_down to get the state prior to `current`
    if (current.sql_down && !current.sql_down.startsWith("--")) {
      const beforeCurrent = cloneSnapshot(cursorSnapshot);
      applySqlToSnapshot(beforeCurrent, current.sql_down);
      cursorSnapshot = beforeCurrent;
    }
  }

  // For failed migrations, assign the snapshot of the preceding migration
  let lastKnown = cloneSnapshot(cursorSnapshot);
  for (const m of rawMigrations) {
    if (snapshotsById[m.id]) {
      lastKnown = cloneSnapshot(snapshotsById[m.id]);
    } else {
      snapshotsById[m.id] = cloneSnapshot(lastKnown);
    }
  }

  // Check if the first migration is already an initial baseline snapshot
  const firstMigration = rawMigrations[0];
  const hasBaselineFirstMigration =
    firstMigration &&
    (firstMigration.sql_up.includes("Initial Schema Snapshot") ||
      firstMigration.sql_down.includes("Baseline schema snapshot"));

  if (hasBaselineFirstMigration) {
    return {
      migrations: rawMigrations,
      snapshotsById,
    };
  }

  // Otherwise, create the virtual/initial Migration #1 (Baseline Snapshot - Describe Tables)
  // representing the initial tables before subsequent migrations
  const baselineSnapshot = cloneSnapshot(cursorSnapshot);
  const baselineSql = formatSnapshotAsBaselineSQL(baselineSnapshot);
  const baselineMigration: Migration = {
    id: -1,
    database_id: databaseId,
    version: rawMigrations.length === 0 ? "1 (Initial Snapshot)" : "0 (Initial Snapshot)",
    status: "success",
    sql_up: baselineSql,
    sql_down: "-- Baseline schema snapshot (initial state)",
    error_message: "",
    snapshot: baselineSnapshot,
    user_id: 1,
  };
  snapshotsById[baselineMigration.id] = baselineSnapshot;

  return {
    migrations: [baselineMigration, ...rawMigrations],
    snapshotsById,
  };
}

/**
 * Compares two SchemaSnapshots and returns structured table/column/index differences.
 */
export function diffSchemaSnapshots(
  left: SchemaSnapshot | null | undefined,
  right: SchemaSnapshot | null | undefined,
): SnapshotDiffResult {
  const leftTables = left?.tables ?? {};
  const rightTables = right?.tables ?? {};
  const allTableNames = Array.from(
    new Set([...Object.keys(leftTables), ...Object.keys(rightTables)]),
  ).sort();

  const summaryItems: SnapshotDiffSummaryItem[] = [];
  const tableDiffs: TableDiffChange[] = [];

  for (const tableName of allTableNames) {
    const lTable = leftTables[tableName];
    const rTable = rightTables[tableName];

    // Table Added
    if (!lTable && rTable) {
      tableDiffs.push({
        tableName,
        status: "added",
        addedColumns: rTable.columns,
        removedColumns: [],
        modifiedColumns: [],
        addedIndexes: rTable.indexes ?? [],
        removedIndexes: [],
      });
      summaryItems.push({
        type: "table_added",
        tableName,
        targetName: tableName,
        detail: `Tabel baru "${tableName}" ditambahkan (${rTable.columns.length} kolom)`,
        badgeText: `+ TABEL BARU`,
      });
      for (const col of rTable.columns) {
        summaryItems.push({
          type: "column_added",
          tableName,
          targetName: col.name,
          detail: `Kolom baru di tabel ${tableName}: ${col.name} (${col.type}${col.nullable ? ", NULL" : ", NOT NULL"})`,
          badgeText: `+ KOLOM BARU`,
        });
      }
      continue;
    }

    // Table Removed
    if (lTable && !rTable) {
      tableDiffs.push({
        tableName,
        status: "removed",
        addedColumns: [],
        removedColumns: lTable.columns,
        modifiedColumns: [],
        addedIndexes: [],
        removedIndexes: lTable.indexes ?? [],
      });
      summaryItems.push({
        type: "table_removed",
        tableName,
        targetName: tableName,
        detail: `Tabel "${tableName}" dihapus (${lTable.columns.length} kolom)`,
        badgeText: `- TABEL DIHAPUS`,
      });
      continue;
    }

    if (!lTable || !rTable) continue;

    // Compare Columns within existing table
    const addedColumns: SchemaColumnSnapshot[] = [];
    const removedColumns: SchemaColumnSnapshot[] = [];
    const modifiedColumns: { before: SchemaColumnSnapshot; after: SchemaColumnSnapshot }[] = [];

    const lColsByName = new Map(
      lTable.columns.map((c) => [c.name.toLowerCase(), c]),
    );
    const rColsByName = new Map(
      rTable.columns.map((c) => [c.name.toLowerCase(), c]),
    );

    for (const rCol of rTable.columns) {
      const lCol = lColsByName.get(rCol.name.toLowerCase());
      if (!lCol) {
        addedColumns.push(rCol);
        summaryItems.push({
          type: "column_added",
          tableName,
          targetName: rCol.name,
          detail: `Kolom baru di tabel ${tableName}: ${rCol.name} (${rCol.type}${rCol.nullable ? ", NULL" : ", NOT NULL"})`,
          badgeText: `+ KOLOM BARU`,
        });
      } else if (
        lCol.type.toLowerCase() !== rCol.type.toLowerCase() ||
        lCol.nullable !== rCol.nullable
      ) {
        modifiedColumns.push({ before: lCol, after: rCol });
        summaryItems.push({
          type: "column_modified",
          tableName,
          targetName: rCol.name,
          detail: `Kolom diubah di tabel ${tableName}.${rCol.name}: ${lCol.type} (${lCol.nullable ? "NULL" : "NOT NULL"}) → ${rCol.type} (${rCol.nullable ? "NULL" : "NOT NULL"})`,
          badgeText: `~ KOLOM DIUBAH`,
        });
      }
    }

    for (const lCol of lTable.columns) {
      if (!rColsByName.has(lCol.name.toLowerCase())) {
        removedColumns.push(lCol);
        summaryItems.push({
          type: "column_removed",
          tableName,
          targetName: lCol.name,
          detail: `Kolom dihapus di tabel ${tableName}: ${lCol.name} (${lCol.type})`,
          badgeText: `- KOLOM DIHAPUS`,
        });
      }
    }

    // Compare Indexes within existing table
    const lIndexes = lTable.indexes ?? [];
    const rIndexes = rTable.indexes ?? [];
    const lIdxByName = new Map(lIndexes.map((i) => [i.name.toLowerCase(), i]));
    const rIdxByName = new Map(rIndexes.map((i) => [i.name.toLowerCase(), i]));

    const addedIndexes: SchemaIndexSnapshot[] = [];
    const removedIndexes: SchemaIndexSnapshot[] = [];

    for (const rIdx of rIndexes) {
      if (!lIdxByName.has(rIdx.name.toLowerCase())) {
        addedIndexes.push(rIdx);
        summaryItems.push({
          type: "index_added",
          tableName,
          targetName: rIdx.name,
          detail: `Index baru di tabel ${tableName}: ${rIdx.name} (${rIdx.unique ? "UNIQUE" : "NON-UNIQUE"})`,
          badgeText: `+ INDEX BARU`,
        });
      }
    }

    for (const lIdx of lIndexes) {
      if (!rIdxByName.has(lIdx.name.toLowerCase())) {
        removedIndexes.push(lIdx);
        summaryItems.push({
          type: "index_removed",
          tableName,
          targetName: lIdx.name,
          detail: `Index dihapus di tabel ${tableName}: ${lIdx.name}`,
          badgeText: `- INDEX DIHAPUS`,
        });
      }
    }

    if (
      addedColumns.length > 0 ||
      removedColumns.length > 0 ||
      modifiedColumns.length > 0 ||
      addedIndexes.length > 0 ||
      removedIndexes.length > 0
    ) {
      tableDiffs.push({
        tableName,
        status: "modified",
        addedColumns,
        removedColumns,
        modifiedColumns,
        addedIndexes,
        removedIndexes,
      });
    }
  }

  return {
    isIdentical: summaryItems.length === 0,
    summaryItems,
    tableDiffs,
  };
}

