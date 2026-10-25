"use client";

import { useEffect, useState, useMemo } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { Database, Migration, QueryResponse, QueryRow } from "@/lib/types";
import ConfirmModal from "@/components/ui/ConfirmModal";
import ToastContainer from "@/components/ui/Toast";
import { useToast } from "@/lib/hooks/useToast";
import { parseExecutionPlan, type ExecutionPlanAnalysis } from "@/lib/explainPlan";
import {
  buildColumnsQuery,
  buildConstraintsQuery,
  buildIndexesQuery,
  isSelectOnlyQuery,
  parseConstraintsRows,
  parseIndexesRows,
  readCell,
  type TableConstraintItem,
  type TableIndexItem,
} from "@/lib/schemaMetadata";
import styles from "./DatabaseDetailView.module.css";
interface DatabaseDetailViewProps {
  databaseId: string;
}
function readRows(body: QueryResponse): QueryRow[] {
  return body.data?.rows || [];
}

const PAGE_SIZE = 50;

export default function DatabaseDetailView({
  databaseId,
}: DatabaseDetailViewProps) {
  const router = useRouter();

  // Database info
  const [db, setDb] = useState<Database | null>(null);
  const [dbError, setDbError] = useState("");
  // Navigation tab
  const [activeTab, setActiveTab] = useState<"schema" | "sql" | "migrations">(
    "schema",
  );

  // Schema & Data tab states
  const [tables, setTables] = useState<string[]>([]);
  const [activeTable, setActiveTable] = useState("");
  const [columns, setColumns] = useState<QueryRow[]>([]);
  const [constraints, setConstraints] = useState<TableConstraintItem[]>([]);
  const [indexes, setIndexes] = useState<TableIndexItem[]>([]);
  const [tableData, setTableData] = useState<QueryRow[]>([]);
  const [dataPage, setDataPage] = useState(1);
  const [totalRows, setTotalRows] = useState(0);
  const [loadingData, setLoadingData] = useState(false);
  const [schemaError, setSchemaError] = useState("");
  // SQL Console tab states
  const [sqlQuery, setSqlQuery] = useState("");
  const [queryExecuting, setQueryExecuting] = useState(false);
  const [queryResult, setQueryResult] = useState<QueryResponse | null>(null);
  const [planAnalysis, setPlanAnalysis] = useState<ExecutionPlanAnalysis | null>(null);
  const [queryError, setQueryError] = useState("");
  // Migrations tab states
  const [migrations, setMigrations] = useState<Migration[]>([]);
  const [migrationsLoading, setMigrationsLoading] = useState(false);
  const [migrationsError, setMigrationsError] = useState("");
  // Load database metadata
  useEffect(() => {
    if (typeof window !== "undefined" && !localStorage.getItem("wbd_user")) {
      router.replace("/login");

      return;
    }

    api

      .database(databaseId)

      .then((data) => {
        setDb(data);
      })
      .catch((err: unknown) => {
        const msg =
          err instanceof Error ? err.message : "Failed to load database info.";
        setDbError(msg);

        if (
          msg.toLowerCase().includes("unauthorized") ||

          msg.toLowerCase().includes("authentication")
        ) {
          router.replace("/login");
        }
      });
  }, [databaseId, router]);

  // Load tables
  useEffect(() => {
    api

      .query(
        databaseId,
        "SELECT table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys') ORDER BY table_name",
      )

      .then((body) => {
        const names = readRows(body)

          .map((row) => String(row.table_name || row.TABLE_NAME || ""))

          .filter(Boolean);

        setTables(names);

        if (names.length > 0 && !activeTable) {
          setActiveTable(names[0]);
        }
      })
      .catch((err: unknown) => {
        setSchemaError(
          err instanceof Error ? err.message : "Unable to list tables.",
        );
      });
  }, [databaseId, activeTable]);

  // Load columns, constraints, and indexes when activeTable changes
  useEffect(() => {
    if (!activeTable) return;

    const escapedTable = activeTable.replaceAll("'", "''");
    const dbType = db?.type;

    api
      .query(databaseId, buildColumnsQuery(escapedTable, dbType))
      .then((body) => {
        setColumns(readRows(body));
      })
      .catch((err: unknown) => {
        setSchemaError(
          err instanceof Error ? err.message : "Unable to load table columns.",
        );
      });

    api
      .query(databaseId, buildConstraintsQuery(escapedTable, dbType))
      .then((body) => {
        setConstraints(parseConstraintsRows(readRows(body)));
      })
      .catch(() => {
        setConstraints([]);
      });

    api
      .query(databaseId, buildIndexesQuery(escapedTable, dbType))
      .then((body) => {
        setIndexes(parseIndexesRows(readRows(body)));
      })
      .catch(() => {
        setIndexes([]);
      });
  }, [databaseId, activeTable, db?.type]);

  // Load table rows when activeTable or dataPage changes
  useEffect(() => {
    if (!activeTable) return;

    const validIdentifier = /^[A-Za-z0-9_$]+$/.test(activeTable);

    if (!validIdentifier) return;

    setLoadingData(true);

    const offset = (dataPage - 1) * PAGE_SIZE;

    Promise.all([
      api.query(databaseId, `SELECT COUNT(*) AS total FROM ${activeTable}`),
      api.query(
        databaseId,
        `SELECT * FROM ${activeTable} LIMIT ${PAGE_SIZE} OFFSET ${offset}`,
      ),
    ])

      .then(([countRes, dataRes]) => {
        const total = Number(readRows(countRes)[0]?.total ?? 0);
        setTotalRows(Number.isFinite(total) ? total : 0);
        setTableData(readRows(dataRes));
      })
      .catch((err: unknown) => {
        setTableData([]);

        setTotalRows(0);

        setSchemaError(
          err instanceof Error ? err.message : "Unable to load table data.",
        );
      })

      .finally(() => {
        setLoadingData(false);
      });
  }, [databaseId, activeTable, dataPage]);

  // Load migrations when switching to migrations tab
  useEffect(() => {
    if (activeTab === "migrations") {
      setMigrationsLoading(true);

      api

        .migrations(databaseId)

        .then((body) => {
          setMigrations(body.data || []);
        })
        .catch((err: unknown) => {
          setMigrationsError(
            err instanceof Error ? err.message : "Failed to load migrations.",
          );
        })

        .finally(() => {
          setMigrationsLoading(false);
        });
    }
  }, [activeTab, databaseId]);

  // Handle SQL execution
  // Toast & Modal Confirmation for Dangerous Actions
  const { toasts, removeToast, toast } = useToast();
  const [confirmModalOpen, setConfirmModalOpen] = useState(false);
  const [pendingExplain, setPendingExplain] = useState(false);
  const [targetRollback, setTargetRollback] = useState<{ id: number; version: string } | null>(null);
  const [rollbackModalOpen, setRollbackModalOpen] = useState(false);
  const [rollbackLoading, setRollbackLoading] = useState(false);

  async function handleConfirmRollback() {
    if (!targetRollback) return;
    setRollbackLoading(true);
    try {
      await api.rollback(Number(databaseId), targetRollback.id);
      toast.success(`Rollback ke migrasi versi ${targetRollback.version} berhasil dieksekusi!`, "Rollback Berhasil");
      const refreshed = await api.migrations(databaseId, 1, 100);
      setMigrations(refreshed.data || []);
      setRollbackModalOpen(false);
      setTargetRollback(null);
    } catch (err: unknown) {
      toast.error(
        err instanceof Error ? err.message : "Gagal melakukan rollback.",
        "Rollback Gagal",
      );
      setRollbackModalOpen(false);
    } finally {
      setRollbackLoading(false);
    }
  }

  // Check if query contains potentially destructive statements
  function isDestructiveQuery(q: string): boolean {
    const upper = q.toUpperCase();
    return (
      upper.includes("DROP ") ||
      upper.includes("TRUNCATE ") ||
      upper.includes("DELETE FROM") ||
      upper.includes("ALTER TABLE")
    );
  }

  function handleRunQuery(explain = false, analyze = false) {
    if (!sqlQuery.trim()) return;
    if (!isSelectOnlyQuery(sqlQuery)) {
      const msg =
        "SQL Console hanya mengizinkan query SELECT (baik untuk akses View Only maupun View & Edit). Untuk perubahan skema atau mutasi data (INSERT, UPDATE, DELETE, CREATE, ALTER, DROP), silakan jalankan melalui menu Migrations.";
      setQueryError(msg);
      setQueryResult(null);
      setPlanAnalysis(null);
      toast.warning(msg, "Hanya Query SELECT Diizinkan");
      return;
    }
    void executeSqlQuery(explain, analyze);
  }

  async function executeSqlQuery(explain = false, analyze = false) {
    setQueryExecuting(true);
    setQueryError("");
    setQueryResult(null);
    setPlanAnalysis(null);

    try {
      let res: QueryResponse;
      if (explain) {
        res = await api.explain(databaseId, sqlQuery.trim(), analyze);
        setPlanAnalysis(parseExecutionPlan(res, analyze, db?.type));
        toast.info(
          `Hasil ${analyze ? "EXPLAIN ANALYZE" : "EXPLAIN"} execution plan berhasil diperoleh.`,
          "Explain Plan",
        );
      } else {
        res = await api.query(databaseId, sqlQuery.trim());
        toast.success("Query berhasil dieksekusi!", "Eksekusi Berhasil");
      }
      setQueryResult(res);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Query execution failed.";
      setQueryError(msg);
      toast.error(msg, "Query Gagal Dieksekusi");
    } finally {
      setQueryExecuting(false);
      setConfirmModalOpen(false);
    }
  }

  const queryColumns = useMemo(() => {
    if (queryResult?.data?.columns && queryResult.data.columns.length > 0) {
      return queryResult.data.columns;
    }

    const rows = readRows(queryResult || {});
    return rows.length > 0 ? Object.keys(rows[0]) : [];
  }, [queryResult]);

  const queryRows = useMemo(() => {
    return readRows(queryResult || {});
  }, [queryResult]);

  const isViewEdit = db?.access_type === "view_edit";

  const totalDataPages = Math.max(1, Math.ceil(totalRows / PAGE_SIZE));

  return (
    <div className={styles.container}>
      {/* Toast Pop-up Notifications */}
      <ToastContainer toasts={toasts} onDismiss={removeToast} />
      {/* Confirmation Modal for Destructive Queries */}
      <ConfirmModal
        isOpen={confirmModalOpen}
        title="Konfirmasi Tindakan Berbahaya"
        message={`Query Anda mengandung perintah destruktif (DROP / TRUNCATE / DELETE / ALTER):
"${sqlQuery.trim().substring(0, 150)}${sqlQuery.trim().length > 150 ? "..." : ""}"
Apakah Anda yakin ingin mengeksekusi query ini pada database?`}
        confirmText="Ya, Jalankan Query"
        cancelText="Batal"
        variant="danger"
        isLoading={queryExecuting}
        onConfirm={() => executeSqlQuery(pendingExplain)}
        onCancel={() => setConfirmModalOpen(false)}
      />

      {/* Confirmation Modal for Rollback */}
      <ConfirmModal
        isOpen={rollbackModalOpen}
        title="Konfirmasi Rollback Migrasi"
        message={`Apakah Anda yakin ingin melakukan rollback ke migrasi versi ${targetRollback?.version}? Perubahan skema yang lebih baru akan dibatalkan secara otomatis melalui SQL down.`}
        confirmText="Ya, Jalankan Rollback"
        cancelText="Batal"
        variant="danger"
        isLoading={rollbackLoading}
        onConfirm={handleConfirmRollback}
        onCancel={() => setRollbackModalOpen(false)}
      />
      {/* Top Bar with Navigation & Title */}

      <div className={styles.topBar}>
        <div className={styles.titleArea}>
          <Link
            href="/databases"

            className={styles.secondaryButton}
            style={{ textDecoration: "none" }}
          >
            ← Kembali ke Daftar
          </Link>
          <h1 className={styles.dbTitle}>{db?.name || `Database #${databaseId}`}</h1>
          {db?.access_type && (
            <span
              className={`${styles.badge} ${
                isViewEdit ? styles.badgeEdit : styles.badgeView
              }`}

            >
              {isViewEdit ? "View & Edit" : "View Only"}
            </span>
          )}
        </div>
      </div>
      {dbError && <div className={styles.alertError}>{dbError}</div>}

      {/* Database Metadata Banner */}

      {db && (
        <div className={styles.metaGrid}>
          <div className={styles.metaItem}>
            Engine: <strong>{db.type}</strong>
          </div>
          <div className={styles.metaItem}>
            Endpoint:{" "}
            <strong>
              {db.host}:{db.port}
            </strong>
          </div>
          <div className={styles.metaItem}>
            Username: <strong>{db.user}</strong>
          </div>
          <div className={styles.metaItem}>
            Hak Akses:{" "}
            <strong style={{ color: isViewEdit ? "#86efac" : "#fde047" }}>
              {isViewEdit ? "View & Edit (Bisa Edit/Query)" : "View Only (Hanya Lihat/SELECT)"}
            </strong>
          </div>
        </div>
      )}

      {/* Navigation Tabs */}

      <nav className={styles.tabs} aria-label="Database sections">
        <button
          className={`${styles.tabBtn} ${
            activeTab === "schema" ? styles.tabBtnActive : ""
          }`}

          onClick={() => setActiveTab("schema")}

        >
          Schema & Data
        </button>
        <button
          className={`${styles.tabBtn} ${
            activeTab === "sql" ? styles.tabBtnActive : ""
          }`}

          onClick={() => setActiveTab("sql")}

        >
          SQL Console
        </button>
        <button
          className={`${styles.tabBtn} ${
            activeTab === "migrations" ? styles.tabBtnActive : ""
          }`}

          onClick={() => setActiveTab("migrations")}

        >
          Migrations ({migrations.length})
        </button>
      </nav>
      {/* Tab 1: Schema & Data Viewer */}
      {activeTab === "schema" && (
        <div className={styles.layoutTwoCol}>
          {/* Tables Sidebar */}

          <aside className={styles.sidebar}>
            <div className={styles.sidebarTitle}>Tabel ({tables.length})</div>
            {tables.length === 0 ? (
              <p style={{ color: "#71717a", fontSize: "0.85rem" }}>
                Tidak ada tabel ditemukan.
              </p>
            ) : (
              <div className={styles.tableList}>
                {tables.map((tbl) => (
                  <button
                    key={tbl}

                    type="button"

                    className={`${styles.tableItem} ${
                      activeTable === tbl ? styles.tableItemActive : ""
                    }`}

                    onClick={() => {
                      setActiveTable(tbl);

                      setDataPage(1);
                    }}

                  >
                    📄 {tbl}
                  </button>
                ))}
              </div>
            )}
          </aside>
          {/* Table Details: Schema and Data */}
          <div style={{ display: "flex", flexDirection: "column", gap: "1.5rem" }}>
            {schemaError && (
              <div className={styles.alertError}>{schemaError}</div>
            )}

            {activeTable ? (
              <>
                {/* 1. Table Schema Card: Columns, Constraints, and Indexes */}

                <div className={styles.contentCard}>
                  <div
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "center",
                      flexWrap: "wrap",
                      gap: "0.5rem",
                    }}
                  >
                    <h2 className={styles.sectionTitle}>
                      Skema Tabel: <code>{activeTable}</code>
                    </h2>
                    <div style={{ display: "flex", gap: "0.6rem", fontSize: "0.8rem", color: "#a1a1aa" }}>
                      <span>{columns.length} Kolom</span>
                      <span>•</span>
                      <span>{constraints.length} Constraint</span>
                      <span>•</span>
                      <span>{indexes.length} Index</span>
                    </div>
                  </div>

                  {/* Columns Table */}
                  <div>
                    <h3 style={{ fontSize: "0.9rem", fontWeight: 700, color: "#e4e4e7", marginBottom: "0.5rem" }}>
                      📋 Daftar Kolom
                    </h3>
                    <div className={styles.tableWrapper}>
                      <table className={styles.dataTable}>
                        <thead>
                          <tr>
                            <th>Nama Kolom</th>
                            <th>Tipe Data</th>
                            <th>Nullable</th>
                            <th>Default</th>
                            <th>Constraint Kolom</th>
                          </tr>
                        </thead>
                        <tbody>
                          {columns.map((col, idx) => {
                            const colName = readCell(col, "column_name");
                            const colType = readCell(col, "data_type");
                            const colNullable = readCell(col, "is_nullable").toUpperCase();
                            const colDefault = readCell(col, "column_default");
                            const colConstraints = constraints.filter(
                              (c) => c.columnName.toLowerCase() === colName.toLowerCase(),
                            );

                            return (
                              <tr key={idx}>
                                <td>
                                  <strong>{colName}</strong>
                                </td>
                                <td>
                                  <code>{colType}</code>
                                </td>
                                <td>
                                  {colNullable === "YES" ? (
                                    <span style={{ color: "#a1a1aa" }}>YES</span>
                                  ) : (
                                    <span style={{ color: "#3b82f6", fontWeight: 600 }}>NO</span>
                                  )}
                                </td>
                                <td>
                                  {colDefault ? (
                                    <code style={{ fontSize: "0.78rem", color: "#a1a1aa" }}>{colDefault}</code>
                                  ) : (
                                    <span style={{ color: "#52525b" }}>—</span>
                                  )}
                                </td>
                                <td>
                                  {colConstraints.length === 0 ? (
                                    <span style={{ color: "#52525b" }}>—</span>
                                  ) : (
                                    <div style={{ display: "flex", gap: "0.35rem", flexWrap: "wrap" }}>
                                      {colConstraints.map((c, cIdx) => {
                                        const isPk = c.constraintType.includes("PRIMARY");
                                        const isFk = c.constraintType.includes("FOREIGN");
                                        const isUq = c.constraintType.includes("UNIQUE");
                                        return (
                                          <span
                                            key={cIdx}
                                            style={{
                                              fontSize: "0.7rem",
                                              padding: "2px 6px",
                                              borderRadius: "4px",
                                              fontWeight: 700,
                                              backgroundColor: isPk
                                                ? "rgba(234, 179, 8, 0.15)"
                                                : isFk
                                                ? "rgba(59, 130, 246, 0.15)"
                                                : isUq
                                                ? "rgba(168, 85, 247, 0.15)"
                                                : "rgba(161, 161, 170, 0.15)",
                                              color: isPk
                                                ? "#fde047"
                                                : isFk
                                                ? "#60a5fa"
                                                : isUq
                                                ? "#c084fc"
                                                : "#d4d4d8",
                                              border: `1px solid ${
                                                isPk
                                                  ? "rgba(234, 179, 8, 0.35)"
                                                  : isFk
                                                  ? "rgba(59, 130, 246, 0.35)"
                                                  : isUq
                                                  ? "rgba(168, 85, 247, 0.35)"
                                                  : "rgba(161, 161, 170, 0.3)"
                                              }`,
                                            }}
                                            title={c.foreignReference ? `References ${c.foreignReference}` : c.constraintName}
                                          >
                                            {isPk ? "🔑 PK" : isFk ? `🔗 FK → ${c.foreignReference}` : isUq ? "✨ UNIQUE" : c.constraintType}
                                          </span>
                                        );
                                      })}
                                    </div>
                                  )}
                                </td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
                  </div>

                  {/* Constraints Table */}
                  <div style={{ marginTop: "0.5rem" }}>
                    <h3 style={{ fontSize: "0.9rem", fontWeight: 700, color: "#e4e4e7", marginBottom: "0.5rem" }}>
                      🔒 Constraints ({constraints.length})
                    </h3>
                    {constraints.length === 0 ? (
                      <p style={{ color: "#71717a", fontSize: "0.85rem", margin: 0 }}>
                        Tidak ada constraint terdaftar pada tabel ini.
                      </p>
                    ) : (
                      <div className={styles.tableWrapper}>
                        <table className={styles.dataTable}>
                          <thead>
                            <tr>
                              <th>Nama Constraint</th>
                              <th>Tipe</th>
                              <th>Kolom</th>
                              <th>Referensi (Foreign Key)</th>
                            </tr>
                          </thead>
                          <tbody>
                            {constraints.map((c, idx) => {
                              const isPk = c.constraintType.includes("PRIMARY");
                              const isFk = c.constraintType.includes("FOREIGN");
                              const isUq = c.constraintType.includes("UNIQUE");
                              return (
                                <tr key={`${c.constraintName}-${c.columnName}-${idx}`}>
                                  <td>
                                    <code>{c.constraintName}</code>
                                  </td>
                                  <td>
                                    <span
                                      style={{
                                        fontSize: "0.72rem",
                                        padding: "2px 8px",
                                        borderRadius: "6px",
                                        fontWeight: 700,
                                        backgroundColor: isPk
                                          ? "rgba(234, 179, 8, 0.15)"
                                          : isFk
                                          ? "rgba(59, 130, 246, 0.15)"
                                          : isUq
                                          ? "rgba(168, 85, 247, 0.15)"
                                          : "rgba(161, 161, 170, 0.15)",
                                        color: isPk
                                          ? "#fde047"
                                          : isFk
                                          ? "#60a5fa"
                                          : isUq
                                          ? "#c084fc"
                                          : "#e4e4e7",
                                      }}
                                    >
                                      {c.constraintType}
                                    </span>
                                  </td>
                                  <td>
                                    <strong>{c.columnName || "—"}</strong>
                                  </td>
                                  <td>
                                    {c.foreignReference ? (
                                      <code style={{ color: "#60a5fa" }}>{c.foreignReference}</code>
                                    ) : (
                                      <span style={{ color: "#52525b" }}>—</span>
                                    )}
                                  </td>
                                </tr>
                              );
                            })}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </div>

                  {/* Indexes Table */}
                  <div style={{ marginTop: "0.5rem" }}>
                    <h3 style={{ fontSize: "0.9rem", fontWeight: 700, color: "#e4e4e7", marginBottom: "0.5rem" }}>
                      ⚡ Indexes ({indexes.length})
                    </h3>
                    {indexes.length === 0 ? (
                      <p style={{ color: "#71717a", fontSize: "0.85rem", margin: 0 }}>
                        Tidak ada index terdaftar pada tabel ini.
                      </p>
                    ) : (
                      <div className={styles.tableWrapper}>
                        <table className={styles.dataTable}>
                          <thead>
                            <tr>
                              <th>Nama Index</th>
                              <th>Kolom</th>
                              <th>Tipe Index</th>
                              <th>Uniqueness</th>
                            </tr>
                          </thead>
                          <tbody>
                            {indexes.map((idxItem, i) => {
                              const isUnique = idxItem.uniqueness === "UNIQUE";
                              return (
                                <tr key={`${idxItem.indexName}-${i}`}>
                                  <td>
                                    <code>{idxItem.indexName}</code>
                                  </td>
                                  <td>
                                    <strong>{idxItem.columnNames || "—"}</strong>
                                  </td>
                                  <td>
                                    <code>{idxItem.indexType}</code>
                                  </td>
                                  <td>
                                    <span
                                      style={{
                                        fontSize: "0.72rem",
                                        padding: "2px 8px",
                                        borderRadius: "6px",
                                        fontWeight: 700,
                                        backgroundColor: isUnique
                                          ? "rgba(16, 185, 129, 0.15)"
                                          : "rgba(161, 161, 170, 0.12)",
                                        color: isUnique ? "#34d399" : "#a1a1aa",
                                      }}
                                    >
                                      {idxItem.uniqueness}
                                    </span>
                                  </td>
                                </tr>
                              );
                            })}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </div>
                </div>
                {/* 2. Table Data Viewer Card */}

                <div className={styles.contentCard}>
                  <div
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "center",
                      flexWrap: "wrap",
                      gap: "0.5rem",
                    }}

                  >
                    <h2 className={styles.sectionTitle}>
                      Isi Data: <code>{activeTable}</code>
                    </h2>
                    <span style={{ fontSize: "0.85rem", color: "#a1a1aa" }}>
                      Total {totalRows} baris
                    </span>
                  </div>
                  {loadingData ? (
                    <p style={{ color: "#a1a1aa", fontSize: "0.9rem" }}>
                      Memuat data...
                    </p>
                  ) : tableData.length === 0 ? (
                    <p style={{ color: "#a1a1aa", fontSize: "0.9rem" }}>
                      Tabel ini belum memiliki data.
                    </p>
                  ) : (
                    <div className={styles.tableWrapper}>
                      <table className={styles.dataTable}>
                        <thead>
                          <tr>
                            {Object.keys(tableData[0]).map((col) => (
                              <th key={col}>{col}</th>
                            ))}
                          </tr>
                        </thead>
                        <tbody>
                          {tableData.map((row, rowIdx) => (
                            <tr key={rowIdx}>
                              {Object.keys(tableData[0]).map((col) => (
                                <td key={col}>
                                  {row[col] !== null && row[col] !== undefined

                                    ? String(row[col])
                                    : <em style={{ color: "#71717a" }}>NULL</em>}
                                </td>
                              ))}
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}

                  {/* Pagination for table data */}

                  {totalRows > PAGE_SIZE && (
                    <div className={styles.pagination}>
                      <span>
                        Halaman {dataPage} dari {totalDataPages}
                      </span>
                      <div style={{ display: "flex", gap: "0.5rem" }}>
                        <button
                          type="button"

                          className={styles.secondaryButton}

                          onClick={() =>
                            setDataPage((p) => Math.max(1, p - 1))
                          }

                          disabled={dataPage === 1}

                        >
                          Sebelumnya
                        </button>
                        <button
                          type="button"

                          className={styles.secondaryButton}

                          onClick={() =>
                            setDataPage((p) =>
                              Math.min(totalDataPages, p + 1),
                            )
                          }

                          disabled={dataPage >= totalDataPages}

                        >
                          Selanjutnya
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              </>
            ) : (
              <div className={styles.contentCard}>
                <p style={{ color: "#a1a1aa" }}>
                  Pilih salah satu tabel di sebelah kiri untuk melihat skema dan datanya.
                </p>
              </div>
            )}
          </div>
        </div>
      )}
      {/* Tab 2: SQL Console */}
      {activeTab === "sql" && (
        <div className={styles.contentCard}>
          <div className={styles.alertInfo}>
            {isViewEdit
              ? "SQL Console khusus untuk query baca (SELECT). Anda memiliki akses View & Edit — untuk melakukan perubahan skema atau modifikasi data (CREATE, ALTER, DROP, INSERT, UPDATE, DELETE), silakan gunakan menu Migrations."
              : "Hak Akses: View Only. Anda dapat menjalankan query baca (SELECT) dan menganalisis execution plan pada SQL Console ini."}
          </div>
          <div>
            <label
              htmlFor="sql-input"

              style={{
                display: "block",
                marginBottom: "0.5rem",
                color: "#e4e4e7",
                fontWeight: 600,
                fontSize: "0.9rem",
              }}

            >
              Ketik Query SQL (Hanya SELECT):
            </label>
            <textarea
              id="sql-input"

              className={styles.sqlTextarea}

              placeholder="Contoh: SELECT * FROM customers LIMIT 10;"

              value={sqlQuery}

              onChange={(e) => setSqlQuery(e.target.value)}
            />
          </div>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              flexWrap: "wrap",
              gap: "0.5rem",
            }}

          >
            <div style={{ display: "flex", gap: "0.5rem" }}>
              {activeTable && (
                <button
                  type="button"

                  className={styles.secondaryButton}
                  style={{ fontSize: "0.8rem", padding: "4px 8px" }}
                  onClick={() =>
                    setSqlQuery(`SELECT * FROM ${activeTable} LIMIT 10;`)
                  }

                >
                  SELECT {activeTable}
                </button>
              )}
            </div>
            <div style={{ display: "flex", gap: "0.5rem", flexWrap: "wrap" }}>
              <button
                type="button"
                className={styles.secondaryButton}
                onClick={() => handleRunQuery(true, false)}
                disabled={queryExecuting || !sqlQuery.trim()}
              >
                🔍 Explain Query
              </button>
              <button
                type="button"
                className={styles.secondaryButton}
                onClick={() => handleRunQuery(true, true)}
                disabled={queryExecuting || !sqlQuery.trim()}
              >
                ⚡ Explain Analyze
              </button>
              <button
                type="button"
                className={styles.primaryButton}
                onClick={() => handleRunQuery(false, false)}
                disabled={queryExecuting || !sqlQuery.trim()}
              >
                {queryExecuting ? "Menjalankan..." : "Jalankan Query"}
              </button>
            </div>
          </div>
          {queryError && (
            <div className={styles.alertError} style={{ display: "flex", justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: "0.75rem" }}>
              <div>
                <strong>Error:</strong> {queryError}
              </div>
              {isViewEdit && queryError.includes("Migrations") && (
                <button
                  type="button"
                  className={styles.secondaryButton}
                  style={{ fontSize: "0.8rem", padding: "4px 10px" }}
                  onClick={() => router.push(`/databases/${databaseId}/migrations/new`)}
                >
                  Buka Menu Migrations →
                </button>
              )}
            </div>
          )}

          {planAnalysis && (
            <div
              style={{
                backgroundColor: "#09090b",
                border: "1px solid #27272a",
                borderRadius: "8px",
                padding: "1rem",
                marginTop: "0.75rem",
                marginBottom: "0.75rem",
              }}
            >
              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "center",
                  marginBottom: "0.75rem",
                  borderBottom: "1px solid #27272a",
                  paddingBottom: "0.5rem",
                }}
              >
                <span style={{ fontWeight: 700, color: "#60a5fa", fontSize: "0.9rem" }}>
                  📊 Hasil Analisis Query Execution Plan ({planAnalysis.isAnalyze ? "EXPLAIN ANALYZE" : "EXPLAIN"})
                </span>
                <button
                  type="button"
                  onClick={() => setPlanAnalysis(null)}
                  style={{
                    background: "none",
                    border: "none",
                    color: "#71717a",
                    cursor: "pointer",
                    fontSize: "0.8rem",
                  }}
                >
                  ✕ Tutup Plan
                </button>
              </div>
              <div
                style={{
                  display: "grid",
                  gridTemplateColumns: "repeat(auto-fit, minmax(170px, 1fr))",
                  gap: "0.75rem",
                  marginBottom: "0.75rem",
                }}
              >
                <div style={{ background: "#18181b", padding: "0.6rem 0.75rem", borderRadius: "6px", border: "1px solid #27272a" }}>
                  <div style={{ fontSize: "0.72rem", color: "#a1a1aa" }}>Scan Strategy</div>
                  <div style={{ fontSize: "0.85rem", fontWeight: 700, color: planAnalysis.scanType?.includes("Seq") || planAnalysis.scanType?.includes("ALL") ? "#fbbf24" : "#34d399", marginTop: "2px" }}>
                    {planAnalysis.scanType || "Standard / Mixed"}
                  </div>
                </div>
                <div style={{ background: "#18181b", padding: "0.6rem 0.75rem", borderRadius: "6px", border: "1px solid #27272a" }}>
                  <div style={{ fontSize: "0.72rem", color: "#a1a1aa" }}>Index Digunakan</div>
                  <div style={{ fontSize: "0.85rem", fontWeight: 700, color: "#e4e4e7", marginTop: "2px" }}>
                    {planAnalysis.indexUsed ? `✓ ${planAnalysis.indexUsed}` : "Tidak ada / Full Scan"}
                  </div>
                </div>
                <div style={{ background: "#18181b", padding: "0.6rem 0.75rem", borderRadius: "6px", border: "1px solid #27272a" }}>
                  <div style={{ fontSize: "0.72rem", color: "#a1a1aa" }}>Join Strategy</div>
                  <div style={{ fontSize: "0.85rem", fontWeight: 700, color: "#e4e4e7", marginTop: "2px" }}>
                    {planAnalysis.joinStrategy || "Single Table / None"}
                  </div>
                </div>
                {planAnalysis.estimatedCost && (
                  <div style={{ background: "#18181b", padding: "0.6rem 0.75rem", borderRadius: "6px", border: "1px solid #27272a" }}>
                    <div style={{ fontSize: "0.72rem", color: "#a1a1aa" }}>Estimated Cost</div>
                    <div style={{ fontSize: "0.85rem", fontWeight: 700, color: "#c084fc", marginTop: "2px" }}>
                      {planAnalysis.estimatedCost}
                    </div>
                  </div>
                )}
                {planAnalysis.actualTime && (
                  <div style={{ background: "#18181b", padding: "0.6rem 0.75rem", borderRadius: "6px", border: "1px solid #27272a" }}>
                    <div style={{ fontSize: "0.72rem", color: "#a1a1aa" }}>Waktu Aktual</div>
                    <div style={{ fontSize: "0.85rem", fontWeight: 700, color: "#38bdf8", marginTop: "2px" }}>
                      {planAnalysis.actualTime}
                    </div>
                  </div>
                )}
                {planAnalysis.rowsProcessed && (
                  <div style={{ background: "#18181b", padding: "0.6rem 0.75rem", borderRadius: "6px", border: "1px solid #27272a" }}>
                    <div style={{ fontSize: "0.72rem", color: "#a1a1aa" }}>Estimasi Baris</div>
                    <div style={{ fontSize: "0.85rem", fontWeight: 700, color: "#f472b6", marginTop: "2px" }}>
                      {planAnalysis.rowsProcessed} rows
                    </div>
                  </div>
                )}
              </div>
              <pre
                style={{
                  backgroundColor: "#000000",
                  color: "#a7f3d0",
                  padding: "0.75rem",
                  borderRadius: "6px",
                  fontSize: "0.8rem",
                  fontFamily: "monospace",
                  overflowX: "auto",
                  margin: 0,
                  border: "1px solid #27272a",
                  whiteSpace: "pre-wrap",
                }}
              >
                {planAnalysis.rawText}
              </pre>
            </div>
          )}

          {queryResult && (
            <div>
              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  marginBottom: "0.5rem",
                  fontSize: "0.875rem",
                  color: "#a1a1aa",
                }}

              >
                <span>Hasil Eksekusi:</span>
                <span>{queryRows.length} baris dikembalikan</span>
              </div>
              {queryRows.length > 0 ? (
                <div className={styles.tableWrapper}>
                  <table className={styles.dataTable}>
                    <thead>
                      <tr>
                        {queryColumns.map((col) => (
                          <th key={col}>{col}</th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {queryRows.map((row, rIdx) => (
                        <tr key={rIdx}>
                          {queryColumns.map((col) => (
                            <td key={col}>
                              {row[col] !== null && row[col] !== undefined

                                ? String(row[col])
                                : <em style={{ color: "#71717a" }}>NULL</em>}
                            </td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className={styles.alertSuccess}>
                  Query berhasil dijalankan tanpa data baris yang dikembalikan (atau mutasi selesai).
                </div>
              )}
            </div>
          )}
        </div>
      )}
      {/* Tab 3: Migrations */}
      {activeTab === "migrations" && (
        <div className={styles.contentCard}>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              flexWrap: "wrap",
              gap: "1rem",
            }}

          >
            <div>
              <h2 className={styles.sectionTitle}>Riwayat Migrasi</h2>
              <p style={{ color: "#a1a1aa", fontSize: "0.85rem", margin: "0.25rem 0 0" }}>
                Kelola versi skema dan migrasi database.
              </p>
            </div>
            {isViewEdit ? (
              <button
                type="button"

                className={styles.primaryButton}

                onClick={() =>
                  router.push(`/databases/${databaseId}/migrations/new`)
                }

              >
                + Buat Migrasi Baru
              </button>
            ) : (
              <span
                style={{
                  fontSize: "0.8rem",
                  color: "#fde047",
                  backgroundColor: "#713f12",
                  padding: "4px 8px",
                  borderRadius: "4px",
                }}

              >
                Membuat migrasi butuh izin View & Edit
              </span>
            )}
          </div>
          {migrationsError && (
            <div className={styles.alertError}>{migrationsError}</div>
          )}

          {migrationsLoading ? (
            <p style={{ color: "#a1a1aa" }}>Memuat riwayat migrasi...</p>
          ) : migrations.length === 0 ? (
            <p style={{ color: "#a1a1aa" }}>Belum ada file migrasi pada database ini.</p>
          ) : (
            <div className={styles.tableWrapper}>
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Versi</th>
                    <th>Status</th>
                    <th>User ID</th>
                    <th>Aksi</th>
                  </tr>
                </thead>
                <tbody>
                  {migrations.map((m) => (
                    <tr key={m.id}>
                      <td>{m.id}</td>
                      <td>
                        <strong>{m.version}</strong>
                      </td>
                      <td>
                        <span
                          style={{
                            color: m.status === "success" ? "#86efac" : "#fca5a5",
                            fontWeight: 600,
                          }}

                        >
                          {m.status}
                        </span>
                      </td>
                      <td>{m.user_id}</td>
                      <td>
                        {isViewEdit && (
                          <button
                            type="button"
                            className={styles.secondaryButton}
                            style={{ fontSize: "0.75rem", padding: "2px 6px" }}
                            onClick={() => {
                              setTargetRollback({ id: m.id, version: m.version });
                              setRollbackModalOpen(true);
                            }}
                          >
                            Rollback ke sini
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
