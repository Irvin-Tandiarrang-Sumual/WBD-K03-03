"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { api } from "@/lib/api";
import type { QueryRow } from "@/lib/types";
import styles from "./TableDataViewer.module.css";

interface QueryBody {
  data?: {
    rows?: QueryRow[];
  };
}

export default function TableDataViewer() {
  const params = useParams<{ title: string; table: string }>();
  const databaseId = params.title;
  const tableName = params.table;
  const [rows, setRows] = useState<QueryRow[]>([]);
  const [error, setError] = useState("");

  const validIdentifier = useMemo(
    () => /^[A-Za-z0-9_$]+$/.test(tableName || ""),
    [tableName],
  );

  useEffect(() => {
    if (!databaseId || !validIdentifier) return;

    api.query(databaseId, `SELECT * FROM ${tableName} LIMIT 100`)
      .then((body: QueryBody) => setRows(body.data?.rows || []))
      .catch((requestError: unknown) => {
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Unable to load table data.",
        );
      });
  }, [databaseId, tableName, validIdentifier]);

  const columns = rows.length > 0 ? Object.keys(rows[0]) : [];
  const displayError = validIdentifier ? error : "Invalid table name.";

  return (
    <section className={styles.container}>
      <header className={styles.header}>
        <h1>{tableName}</h1>
        <p>Showing up to 100 rows</p>
      </header>

      {displayError && <p role="alert">{displayError}</p>}

      {!error && rows.length === 0 && <p>No rows found.</p>}

      {rows.length > 0 && (
        <div className={styles.tableWrapper}>
          <table className={styles.dataTable}>
            <thead>
              <tr>
                {columns.map((column) => (
                  <th key={column}>{column}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, index) => (
                <tr key={index}>
                  {columns.map((column) => (
                    <td key={column}>{String(row[column] ?? "")}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
