"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, unwrapPaged } from "@/lib/api";
import {
  buildColumnsQuery,
  buildConstraintsQuery,
  buildIndexesQuery,
  parseConstraintsRows,
  parseIndexesRows,
  readCell,
  type TableConstraintItem,
  type TableIndexItem,
} from "@/lib/schemaMetadata";
import type { Database, QueryRow } from "@/lib/types";
import styles from "./SchemaViewer.module.css";

interface SchemaViewerProps {
  databaseId: string;
}

function readRows(body: { data?: { rows?: QueryRow[] } }): QueryRow[] {
  return Array.isArray(body.data?.rows) ? body.data.rows : [];
}

export default function SchemaViewer({ databaseId }: SchemaViewerProps) {
  const router = useRouter();
  const [tables, setTables] = useState<string[]>([]);
  const [activeTable, setActiveTable] = useState("");
  const [columns, setColumns] = useState<QueryRow[]>([]);
  const [constraints, setConstraints] = useState<TableConstraintItem[]>([]);
  const [indexes, setIndexes] = useState<TableIndexItem[]>([]);
  const [database, setDatabase] = useState<Database | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api
      .databases(1, 100)
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
          .map((row) => readCell(row, "table_name"))
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

    api
      .query(databaseId, buildColumnsQuery(escapedTable, databaseType))
      .then((body) => setColumns(readRows(body)))
      .catch((requestError: unknown) => {
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Unable to load columns.",
        );
      });

    api
      .query(databaseId, buildConstraintsQuery(escapedTable, databaseType))
      .then((body) => setConstraints(parseConstraintsRows(readRows(body))))
      .catch(() => setConstraints([]));

    api
      .query(databaseId, buildIndexesQuery(escapedTable, databaseType))
      .then((body) => setIndexes(parseIndexesRows(readRows(body))))
      .catch(() => setIndexes([]));
  }, [databaseId, activeTable, database]);

  const canMigrate = Boolean(
    database && (database.access_type ?? database.accessType) === "view_edit",
  );

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

        <main className={styles.main} style={{ display: "flex", flexDirection: "column", gap: "1.5rem" }}>
          <div>
            <h3 style={{ margin: "0 0 0.5rem", fontSize: "0.95rem" }}>
              Columns ({columns.length})
            </h3>
            <table className={styles.dataTable}>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Type</th>
                  <th>Nullable</th>
                  <th>Default</th>
                </tr>
              </thead>
              <tbody>
                {columns.map((column, index) => (
                  <tr key={`${readCell(column, "column_name")}-${index}`}>
                    <td>{readCell(column, "column_name")}</td>
                    <td>{readCell(column, "data_type")}</td>
                    <td>{readCell(column, "is_nullable")}</td>
                    <td>{readCell(column, "column_default") || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {columns.length === 0 && !error && (
              <p className={styles.emptyState}>No columns found.</p>
            )}
          </div>

          <div>
            <h3 style={{ margin: "0 0 0.5rem", fontSize: "0.95rem" }}>
              Constraints ({constraints.length})
            </h3>
            {constraints.length === 0 ? (
              <p className={styles.emptyState}>No constraints found.</p>
            ) : (
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>Constraint Name</th>
                    <th>Type</th>
                    <th>Column</th>
                    <th>Foreign Reference</th>
                  </tr>
                </thead>
                <tbody>
                  {constraints.map((c, index) => (
                    <tr key={`${c.constraintName}-${c.columnName}-${index}`}>
                      <td>{c.constraintName}</td>
                      <td>{c.constraintType}</td>
                      <td>{c.columnName || "—"}</td>
                      <td>{c.foreignReference || "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>

          <div>
            <h3 style={{ margin: "0 0 0.5rem", fontSize: "0.95rem" }}>
              Indexes ({indexes.length})
            </h3>
            {indexes.length === 0 ? (
              <p className={styles.emptyState}>No indexes found.</p>
            ) : (
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>Index Name</th>
                    <th>Columns</th>
                    <th>Type</th>
                    <th>Uniqueness</th>
                  </tr>
                </thead>
                <tbody>
                  {indexes.map((idxItem, index) => (
                    <tr key={`${idxItem.indexName}-${index}`}>
                      <td>{idxItem.indexName}</td>
                      <td>{idxItem.columnNames || "—"}</td>
                      <td>{idxItem.indexType}</td>
                      <td>{idxItem.uniqueness}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </main>
      </div>
    </div>
  );
}
