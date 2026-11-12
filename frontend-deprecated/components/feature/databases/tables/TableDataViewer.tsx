"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { QueryResponse, QueryRow } from "@/lib/types";
import styles from "./TableDataViewer.module.css";

const PAGE_SIZE = 100;

function readRows(body: QueryResponse): QueryRow[] {
  return body.data?.rows || [];
}

export default function TableDataViewer() {
  const router = useRouter();
  const params = useParams<{ title: string; table: string }>();
  const databaseId = params.title;
  const tableName = params.table;
  const [rows, setRows] = useState<QueryRow[]>([]);
  const [error, setError] = useState("");
  const [page, setPage] = useState(1);
  const [totalRows, setTotalRows] = useState(0);

  const validIdentifier = useMemo(
    () => /^[A-Za-z0-9_$]+$/.test(tableName || ""),
    [tableName],
  );

  useEffect(() => {
    if (!databaseId || !validIdentifier) return;

    const offset = (page - 1) * PAGE_SIZE;
    let cancelled = false;

    Promise.all([
      api.query(databaseId, `SELECT COUNT(*) AS total FROM ${tableName}`),
      api.query(
        databaseId,
        `SELECT * FROM ${tableName} LIMIT ${PAGE_SIZE} OFFSET ${offset}`,
      ),
    ])
      .then(([countBody, dataBody]) => {
        if (cancelled) return;
        const countValue = readRows(countBody)[0]?.total;
        const parsedTotal = Number(countValue ?? 0);
        setTotalRows(Number.isFinite(parsedTotal) ? parsedTotal : 0);
        setRows(readRows(dataBody));
        setError("");
      })
      .catch((requestError: unknown) => {
        if (cancelled) return;
        setRows([]);
        setTotalRows(0);
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Unable to load table data.",
        );
      });

    return () => {
      cancelled = true;
    };
  }, [databaseId, page, tableName, validIdentifier]);

  const columns = rows.length > 0 ? Object.keys(rows[0]) : [];
  const displayError = validIdentifier ? error : "Invalid table name.";
  const totalPages = Math.max(1, Math.ceil(totalRows / PAGE_SIZE));
  const firstRow = totalRows === 0 ? 0 : (page - 1) * PAGE_SIZE + 1;
  const lastRow = Math.min(page * PAGE_SIZE, totalRows);

  return (
    <section className={styles.container}>
      <header className={styles.header}>
        <div>
          <h1>{tableName}</h1>
          <p>Maximum {PAGE_SIZE} rows per page</p>
        </div>
        <button
          type="button"
          className={styles.secondaryButton}
          onClick={() => router.push(`/databases/${databaseId}/schema`)}
        >
          Back to Schema
        </button>
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

      {!error && (
        <nav className={styles.pagination} aria-label="Table data pagination">
          <span className={styles.entryCount}>
            Showing {firstRow}-{lastRow} of {totalRows} rows
          </span>
          <div className={styles.pageControls}>
            <button
              type="button"
              className={styles.pageButton}
              onClick={() => setPage((current) => Math.max(1, current - 1))}
              disabled={page === 1}
            >
              Previous
            </button>
            <span className={styles.pageNumber}>
              Page {page} of {totalPages}
            </span>
            <button
              type="button"
              className={styles.pageButton}
              onClick={() =>
                setPage((current) => Math.min(totalPages, current + 1))
              }
              disabled={page >= totalPages}
            >
              Next
            </button>
          </div>
        </nav>
      )}
    </section>
  );
}
