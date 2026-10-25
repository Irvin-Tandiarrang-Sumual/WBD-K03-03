"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, unwrapPaged } from "@/lib/api";
import type { Database, QueryRow } from "@/lib/types";
import styles from "./SchemaViewer.module.css";

interface SchemaViewerProps {
  databaseId: string;
}

function readRows(body: { data?: { rows?: QueryRow[] } }): QueryRow[] {
  return Array.isArray(body.data?.rows) ? body.data.rows : [];
}

function readValue(row: QueryRow, name: string): unknown {
  const matchingKey = Object.keys(row).find(
    (key) => key.toLowerCase() === name.toLowerCase(),
  );
  return matchingKey ? row[matchingKey] : undefined;
}

export default function SchemaViewer({ databaseId }: SchemaViewerProps) {
  const router = useRouter();
  const [tables, setTables] = useState<string[]>([]);
  const [activeTable, setActiveTable] = useState("");
  const [columns, setColumns] = useState<QueryRow[]>([]);
  const [database, setDatabase] = useState<Database | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api.databases(1, 100)
      .then((body) => {
        const accessibleDatabase = unwrapPaged(body).find(
          (item) => String(item.id) === databaseId,
        );
        if (!accessibleDatabase) {
          throw new Error("Database not found.");
        }
        setDatabase(accessibleDatabase);
        const databaseType = String(accessibleDatabase.type).toLowerCase();
        return api.query(
          databaseId,
          databaseType === "mysql"
            ? "SELECT table_schema, table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE' ORDER BY table_name"
            : "SELECT table_schema, table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema NOT IN ('pg_catalog', 'information_schema') ORDER BY table_schema, table_name",
        );
      })
      .then((body) => {
        const names = readRows(body)
          .map((row) => String(readValue(row, "table_name") || ""))
          .filter(Boolean);
        setTables(names);
        setActiveTable(names[0] || "");
      })
      .catch((requestError: unknown) => {
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Unable to load schema.",
        );
      });
  }, [databaseId]);

  useEffect(() => {
    if (!activeTable || !database) {
      return;
    }

    const escapedTable = activeTable.replaceAll("'", "''");
    const databaseType = String(database.type).toLowerCase();

    api.query(
      databaseId,
      databaseType === "mysql"
        ? `SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = '${escapedTable}' ORDER BY ordinal_position`
        : `SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_schema NOT IN ('pg_catalog', 'information_schema') AND table_name = '${escapedTable}' ORDER BY ordinal_position`,
    )
      .then((body) => setColumns(readRows(body)))
      .catch((requestError: unknown) => {
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Unable to load columns.",
        );
      });
  }, [databaseId, activeTable, database]);

  const canMigrate = Boolean(database && database.accessType === "view_edit");

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <p className={styles.breadcrumb}>
          <strong>{activeTable || "Schema"}</strong>
        </p>
        <div className={styles.actions}>
          <button
            type="button"
            onClick={() =>
              router.push(`/databases/${databaseId}/migrations/new`)
            }
            className={styles.secondaryButton}
            disabled={!canMigrate}
          >
            Migrate
          </button>
          <button
            type="button"
            onClick={() =>
              router.push(`/databases/${databaseId}/tables/${activeTable}`)
            }
            className={styles.primaryButton}
            disabled={!activeTable}
          >
            View Data
          </button>
        </div>
      </div>

      {error && <p role="alert">{error}</p>}

      <div className={styles.layout}>
        <aside className={styles.sidebar}>
          <div className={styles.schemaTitle}>Tables</div>
          {tables.map((table) => (
            <button
              key={table}
              type="button"
              onClick={() => setActiveTable(table)}
              className={`${styles.tableItem} ${
                activeTable === table ? styles.tableItemActive : ""
              }`}
            >
              {table}
            </button>
          ))}
        </aside>

        <main className={styles.main}>
          <table className={styles.dataTable}>
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Nullable</th>
              </tr>
            </thead>
            <tbody>
              {columns.map((column, index) => (
                <tr key={`${String(readValue(column, "column_name"))}-${index}`}>
                  <td>{String(readValue(column, "column_name") || "")}</td>
                  <td>{String(readValue(column, "data_type") || "")}</td>
                  <td>{String(readValue(column, "is_nullable") || "")}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {columns.length === 0 && !error && (
            <p className={styles.emptyState}>No columns found.</p>
          )}
        </main>
      </div>
    </div>
  );
}
