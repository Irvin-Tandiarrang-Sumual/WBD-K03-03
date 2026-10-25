"use client";

import { useEffect, useState, useMemo } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { Database, Migration, QueryResponse, QueryRow } from "@/lib/types";
import ConfirmModal from "@/components/ui/ConfirmModal";
import ToastContainer from "@/components/ui/Toast";
import { useToast } from "@/lib/hooks/useToast";
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
  const [tableData, setTableData] = useState<QueryRow[]>([]);
  const [dataPage, setDataPage] = useState(1);
  const [totalRows, setTotalRows] = useState(0);
  const [loadingData, setLoadingData] = useState(false);
  const [schemaError, setSchemaError] = useState("");
  // SQL Console tab states
  const [sqlQuery, setSqlQuery] = useState("");
  const [queryExecuting, setQueryExecuting] = useState(false);
  const [queryResult, setQueryResult] = useState<QueryResponse | null>(null);
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

  // Load columns when activeTable changes
  useEffect(() => {
    if (!activeTable) return;

    const escapedTable = activeTable.replaceAll("'", "''");

    api

      .query(
        databaseId,
        `SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_schema NOT IN ('pg_catalog', 'information_schema', 'mysql', 'performance_schema', 'sys') AND table_name = '${escapedTable}' ORDER BY ordinal_position`,
      )

      .then((body) => {
        setColumns(readRows(body));
      })
      .catch((err: unknown) => {
        setSchemaError(
          err instanceof Error ? err.message : "Unable to load table columns.",
        );
      });
  }, [databaseId, activeTable]);

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

  function handleRunQuery(explain = false) {
    if (!sqlQuery.trim()) return;
    if (!explain && isDestructiveQuery(sqlQuery)) {
      setPendingExplain(false);
      setConfirmModalOpen(true);
      return;
    }
    executeSqlQuery(explain);
  }

  async function executeSqlQuery(explain = false) {
    setQueryExecuting(true);
    setQueryError("");
    setQueryResult(null);

    try {
      let res: QueryResponse;
      if (explain) {
        res = await api.explain(databaseId, sqlQuery.trim());
        toast.info("Query execution plan berhasil diperoleh.", "Explain Plan");
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
                {/* 1. Table Schema Card */}

                <div className={styles.contentCard}>
                  <div
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "center",
                    }}

                  >
                    <h2 className={styles.sectionTitle}>
                      Skema Tabel: <code>{activeTable}</code>
                    </h2>
                    <span style={{ fontSize: "0.85rem", color: "#a1a1aa" }}>
                      {columns.length} Kolom
                    </span>
                  </div>
                  <div className={styles.tableWrapper}>
                    <table className={styles.dataTable}>
                      <thead>
                        <tr>
                          <th>Nama Kolom</th>
                          <th>Tipe Data</th>
                          <th>Nullable</th>
                        </tr>
                      </thead>
                      <tbody>
                        {columns.map((col, idx) => (
                          <tr key={idx}>
                            <td>
                              <strong>
                                {String(col.column_name || col.COLUMN_NAME || "")}
                              </strong>
                            </td>
                            <td>
                              <code>
                                {String(col.data_type || col.DATA_TYPE || "")}
                              </code>
                            </td>
                            <td>
                              {String(
                                col.is_nullable || col.IS_NULLABLE || "",
                              ).toUpperCase() === "YES" ? (
                                <span style={{ color: "#a1a1aa" }}>YES</span>
                              ) : (
                                <span style={{ color: "#3b82f6" }}>NO</span>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
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
          <div
            className={isViewEdit ? styles.alertSuccess : styles.alertInfo}
          >
            {isViewEdit
              ? "Hak Akses: View & Edit. Anda dapat menjalankan query SELECT serta modifikasi data (INSERT, UPDATE, DELETE)."
              : "Hak Akses: View Only. Sesuai grup Anda, Anda hanya diizinkan menjalankan query baca (SELECT)."}
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
              Ketik Query SQL:
            </label>
            <textarea
              id="sql-input"

              className={styles.sqlTextarea}

              placeholder={
                isViewEdit
                  ? "Contoh: SELECT * FROM customers LIMIT 10; atau UPDATE customers SET name='New Name' WHERE id=1;"
                  : "Contoh: SELECT * FROM customers LIMIT 10;"
              }

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
            <div style={{ display: "flex", gap: "0.5rem" }}>
              <button
                type="button"

                className={styles.secondaryButton}

                onClick={() => handleRunQuery(true)}

                disabled={queryExecuting || !sqlQuery.trim()}

              >
                Explain Query
              </button>
              <button
                type="button"

                className={styles.primaryButton}

                onClick={() => handleRunQuery(false)}

                disabled={queryExecuting || !sqlQuery.trim()}

              >
                {queryExecuting ? "Menjalankan..." : "Jalankan Query"}
              </button>
            </div>
          </div>
          {queryError && (
            <div className={styles.alertError}>
              <strong>Error:</strong> {queryError}
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
