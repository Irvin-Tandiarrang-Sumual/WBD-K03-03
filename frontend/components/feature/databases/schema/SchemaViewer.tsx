"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { QueryRow } from "@/lib/types";
import styles from "./SchemaViewer.module.css";

interface SchemaViewerProps {
  databaseId: string;
}

function readRows(body: { data?: { rows?: QueryRow[] } }): QueryRow[] {
  return body.data?.rows || [];
}

export default function SchemaViewer({ databaseId }: SchemaViewerProps) {
  const router = useRouter();
  const [tables, setTables] = useState<string[]>([]);
  const [activeTable, setActiveTable] = useState("");
  const [columns, setColumns] = useState<QueryRow[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    api.query(
      databaseId,
      "SELECT table_schema, table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE' ORDER BY table_schema, table_name",
    )
      .then((body) => {
        const names = readRows(body)
          .map((row) => String(row.table_name || row.TABLE_NAME || ""))
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
    if (!activeTable) return;

    const escapedTable = activeTable.replaceAll("'", "''");
    api.query(
      databaseId,
      `SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = '${escapedTable}' ORDER BY ordinal_position`,
    )
      .then((body) => setColumns(readRows(body)))
      .catch((requestError: unknown) => {
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Unable to load columns.",
        );
      });
  }, [databaseId, activeTable]);

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <p className={styles.breadcrumb}>
          <strong>{activeTable || "Schema"}</strong>
        </p>
        <button
          onClick={() =>
            router.push(`/databases/${databaseId}/tables/${activeTable}`)
          }
          className={styles.primaryButton}
          disabled={!activeTable}
        >
          View Data
        </button>
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
                <tr key={`${String(column.column_name)}-${index}`}>
                  <td>{String(column.column_name || "")}</td>
                  <td>{String(column.data_type || "")}</td>
                  <td>{String(column.is_nullable || "")}</td>
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
