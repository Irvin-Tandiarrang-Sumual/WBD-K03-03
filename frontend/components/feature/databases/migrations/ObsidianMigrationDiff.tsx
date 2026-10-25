"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { Database, Migration, MigrationComparison, QueryResponse } from "@/lib/types";
import ConfirmModal from "@/components/ui/ConfirmModal";
import ToastContainer from "@/components/ui/Toast";
import { useToast } from "@/lib/hooks/useToast";
import { sanitizeSqlQuery } from "@/lib/api/executor";
import { parseExecutionPlan, type ExecutionPlanAnalysis } from "@/lib/explainPlan";
import {
  diffSchemaSnapshots,
  enrichMigrationsWithSnapshots,
  fetchLiveDatabaseSnapshot,
  type SchemaSnapshot,
} from "@/lib/schemaSnapshot";
import styles from "./MigrationDiff.module.css";

type ViewMode = "Snapshot Diff" | "Snapshot JSON" | "SQL Script";

interface MigrationDiffProps {
  databaseId: string;
  databaseName?: string;
}

interface DiffLine {
  index: number;
  left: string;
  right: string;
  equal: boolean;
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "Unexpected request error";
}

export default function MigrationDiff({
  databaseId,
  databaseName,
}: MigrationDiffProps) {
  const router = useRouter();
  const [database, setDatabase] = useState<Database | null>(null);
  const [resolvedDatabaseName, setResolvedDatabaseName] = useState(
    databaseName ?? `Database ${databaseId}`,
  );
  const [migrations, setMigrations] = useState<Migration[]>([]);
  const [snapshotsById, setSnapshotsById] = useState<Record<number, SchemaSnapshot>>({});
  const [leftMigrationId, setLeftMigrationId] = useState("");
  const [rightMigrationId, setRightMigrationId] = useState("");
  const [rollbackTargetId, setRollbackTargetId] = useState("");
  const [comparison, setComparison] = useState<MigrationComparison | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>("Snapshot Diff");
  const [migrationQuery, setMigrationQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  // Explain execution plan state
  const [explainLoading, setExplainLoading] = useState(false);
  const [planAnalysis, setPlanAnalysis] = useState<ExecutionPlanAnalysis | null>(null);

  // Modal State
  const [modalOpen, setModalOpen] = useState(false);
  const [modalLoading, setModalLoading] = useState(false);

  // Toast System
  const { toasts, removeToast, toast } = useToast();

  const loadData = useCallback(async () => {
    try {
      const [migrationResponse, db] = await Promise.all([
        api.migrations(databaseId, 1, 100),
        api.database(databaseId),
      ]);
      const rawLoaded = migrationResponse.data ?? [];
      const liveSnapshot = await fetchLiveDatabaseSnapshot(databaseId, db.type);
      const { migrations: enriched, snapshotsById: enrichedSnapshots } =
        enrichMigrationsWithSnapshots(rawLoaded, liveSnapshot, Number(databaseId));

      const latest = enriched.at(-1);
      const previous = enriched.at(-2) ?? latest;
      const latestRealSuccessful = [...rawLoaded]
        .reverse()
        .find((migration) => migration.status === "success");

      setDatabase(db);
      setMigrations(enriched);
      setSnapshotsById(enrichedSnapshots);
      setResolvedDatabaseName(db.name);
      setLeftMigrationId(previous ? String(previous.id) : "");
      setRightMigrationId(latest ? String(latest.id) : "");
      setRollbackTargetId(
        latestRealSuccessful ? String(latestRealSuccessful.id) : "",
      );
      setComparison(null);
      setError("");
    } catch (loadError) {
      setError(errorMessage(loadError));
    } finally {
      setLoading(false);
    }
  }, [databaseId]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  // Access control: View Edit vs View Only
  const isViewEdit = database?.access_type === "view_edit";

  const leftMigration = useMemo(
    () => migrations.find((migration) => String(migration.id) === leftMigrationId),
    [leftMigrationId, migrations],
  );
  const rightMigration = useMemo(
    () => migrations.find((migration) => String(migration.id) === rightMigrationId),
    [migrations, rightMigrationId],
  );
  const successfulMigrations = useMemo(
    () => migrations.filter((migration) => migration.status === "success" && migration.id > 0),
    [migrations],
  );

  const leftSnapshot = useMemo(
    () => (leftMigration ? snapshotsById[leftMigration.id] ?? null : null),
    [leftMigration, snapshotsById],
  );
  const rightSnapshot = useMemo(
    () => (rightMigration ? snapshotsById[rightMigration.id] ?? null : null),
    [rightMigration, snapshotsById],
  );

  const snapshotDiff = useMemo(
    () => diffSchemaSnapshots(leftSnapshot, rightSnapshot),
    [leftSnapshot, rightSnapshot],
  );

  const leftJsonText = useMemo(
    () => JSON.stringify(leftSnapshot ?? { tables: {} }, null, 2),
    [leftSnapshot],
  );
  const rightJsonText = useMemo(
    () => JSON.stringify(rightSnapshot ?? { tables: {} }, null, 2),
    [rightSnapshot],
  );

  const jsonDiffLines = useMemo<DiffLine[]>(() => {
    const leftLines = leftJsonText.split("\n");
    const rightLines = rightJsonText.split("\n");
    const lineCount = Math.max(leftLines.length, rightLines.length);
    return Array.from({ length: lineCount }, (_, index) => {
      const left = leftLines[index] ?? "";
      const right = rightLines[index] ?? "";
      return { index, left, right, equal: left === right };
    });
  }, [leftJsonText, rightJsonText]);

  const sqlDiffLines = useMemo<DiffLine[]>(() => {
    const leftLines = leftMigration?.sql_up.split("\n") ?? [];
    const rightLines = rightMigration?.sql_up.split("\n") ?? [];
    const lineCount = Math.max(leftLines.length, rightLines.length);

    return Array.from({ length: lineCount }, (_, index) => {
      const left = leftLines[index] ?? "";
      const right = rightLines[index] ?? "";
      return { index, left, right, equal: left === right };
    });
  }, [leftMigration, rightMigration]);

  const handleCompare = async () => {
    if (!leftMigrationId || !rightMigrationId) {
      setError("Pilih dua versi migrasi untuk dibandingkan.");
      return;
    }
    if (leftMigrationId === rightMigrationId) {
      setError("Pilih dua versi migrasi yang berbeda.");
      return;
    }

    setBusy(true);
    setError("");
    setNotice("");
    try {
      if (Number(leftMigrationId) > 0 && Number(rightMigrationId) > 0) {
        const result = await api.compareMigrations(
          databaseId,
          leftMigrationId,
          rightMigrationId,
        );
        setComparison(result);
      } else if (leftMigration && rightMigration) {
        setComparison({
          first_migration: leftMigration,
          second_migration: rightMigration,
          sql_up_equal: leftMigration.sql_up === rightMigration.sql_up,
          sql_down_equal: leftMigration.sql_down === rightMigration.sql_down,
        });
      }
      toast.info(
        `Ditemukan ${snapshotDiff.summaryItems.length} perubahan struktur pada perbandingan snapshot.`,
        "Schema Snapshot Diff",
      );
    } catch (compareError) {
      const msg = errorMessage(compareError);
      setError(msg);
      toast.error(msg, "Gagal Membandingkan");
    } finally {
      setBusy(false);
    }
  };

  // Generate candidate SQL from diff and populate to editor
  const handleUseDiffAsCandidate = () => {
    if (!rightMigration) return;

    const addedLines = sqlDiffLines
      .filter((d) => !d.equal && d.right.trim())
      .map((d) => d.right);

    const candidateSql = addedLines.length > 0 ? addedLines.join("\n") : rightMigration.sql_up;
    setMigrationQuery(candidateSql);
    setPlanAnalysis(null);
    toast.success("Kandidat SQL migrasi berhasil disalin ke editor di bawah untuk ditinjau.", "Kandidat SQL Disalin");
  };

  // Query Execution Plan Analysis (EXPLAIN or EXPLAIN ANALYZE)
  const handleExplain = async (analyze: boolean) => {
    const sql = migrationQuery.trim();
    if (!sql) {
      toast.warning("Masukkan query SQL terlebih dahulu untuk dianalisis.", "Query Kosong");
      return;
    }

    setExplainLoading(true);
    try {
      const statements = sql
        .split(";")
        .map((stmt) => sanitizeSqlQuery(stmt, true))
        .filter(Boolean);

      const explainableRegex = /^(SELECT|WITH)\b/i;
      const dmlOrDdlRegex = /^(CREATE|ALTER|DROP|TRUNCATE|RENAME|INSERT|UPDATE|DELETE)\b/i;

      const targetQuery = statements.find((stmt) => explainableRegex.test(stmt));

      if (!targetQuery && statements.some((stmt) => dmlOrDdlRegex.test(stmt))) {
        const matchedStatements = statements.filter((stmt) => dmlOrDdlRegex.test(stmt));
        setPlanAnalysis({
          isAnalyze: analyze,
          rawText:
            `-- Analisis Statis Migrasi (${matchedStatements.length} statement)\n` +
            `-- Catatan: SQL EXPLAIN berfokus pada rencana eksekusi query baca (SELECT).\n\n` +
            matchedStatements.map((s, idx) => `${idx + 1}. ${s};`).join("\n"),
          scanType: "Migration Operation (Schema / DML)",
          indexUsed: /INDEX/i.test(sql) ? "Pembuatan / Modifikasi Index" : "N/A (Migration)",
          joinStrategy: "Tidak ada (Operasi Migrasi)",
        });
        toast.info(
          "Statement migrasi terdeteksi. Menampilkan analisis statis operasi migrasi.",
          "Analisis Migrasi",
        );
        return;
      }

      const queryToExplain = targetQuery || sanitizeSqlQuery(sql, true);
      const response: QueryResponse = await api.explain(databaseId, queryToExplain, analyze);
      setPlanAnalysis(parseExecutionPlan(response, analyze, database?.type));

      toast.info(
        `Hasil ${analyze ? "EXPLAIN ANALYZE" : "EXPLAIN"} execution plan berhasil diperoleh.`,
        "Execution Plan Analisis",
      );
    } catch (explainErr) {
      toast.error(errorMessage(explainErr), "Analisis Plan Gagal");
    } finally {
      setExplainLoading(false);
    }
  };

  const handleExecute = async () => {
    if (!isViewEdit) {
      toast.error("Hanya pengguna dengan hak akses View & Edit yang dapat mengeksekusi migrasi.", "Akses Ditolak");
      return;
    }

    const sql = migrationQuery.trim();
    if (!sql) {
      setError("Masukkan SQL migrasi sebelum mengeksekusinya.");
      return;
    }

    setBusy(true);
    setError("");
    setNotice("");
    try {
      const created = await api.createMigration(databaseId, sql);
      await loadData();
      if (created.status === "fail") {
        setError(
          `Migrasi versi ${created.version} gagal dan seluruh perubahan dalam batch ini telah di-rollback otomatis (Atomic): ${created.error_message || "Unknown database error"}`,
        );
        toast.error(
          `Migrasi versi ${created.version} gagal (seluruh query di-rollback otomatis): ${created.error_message}`,
          "Migrasi Gagal & Di-Rollback",
        );
      } else {
        setMigrationQuery("");
        setPlanAnalysis(null);
        setNotice(`Migrasi versi ${created.version} berhasil dieksekusi secara atomik dan snapshot terbaru telah disimpan.`);
        toast.success(`Migrasi versi ${created.version} berhasil dieksekusi & snapshot diperbarui!`, "Migrasi Berhasil");
      }
    } catch (executeError) {
      const msg = errorMessage(executeError);
      setError(msg);
      toast.error(msg, "Eksekusi Migrasi Gagal");
    } finally {
      setBusy(false);
    }
  };

  const promptRollback = () => {
    if (!isViewEdit) {
      toast.error("Hanya pengguna dengan hak akses View & Edit yang dapat melakukan rollback.", "Akses Ditolak");
      return;
    }
    if (!rollbackTargetId) {
      setError("Pilih versi target migrasi untuk rollback.");
      return;
    }
    setModalOpen(true);
  };

  const handleRollbackConfirm = async () => {
    setModalLoading(true);
    setError("");
    setNotice("");
    try {
      const rolledBack = await api.rollback(
        Number(databaseId),
        Number(rollbackTargetId),
      );
      const msg = rolledBack.length > 0
        ? `${rolledBack.length} migrasi berhasil di-rollback.`
        : "Database sudah berada pada versi migrasi yang dipilih.";
      setNotice(msg);
      toast.success(msg, "Rollback Berhasil");
      await loadData();
      setModalOpen(false);
    } catch (rollbackError) {
      const msg = errorMessage(rollbackError);
      setError(msg);
      toast.error(msg, "Rollback Gagal");
      setModalOpen(false);
    } finally {
      setModalLoading(false);
    }
  };

  if (loading) {
    return <div className={styles.message}>Memuat data migrasi & snapshot database...</div>;
  }

  return (
    <div className={styles.container}>
      <ToastContainer toasts={toasts} onDismiss={removeToast} />

      <ConfirmModal
        isOpen={modalOpen}
        title="Konfirmasi Rollback Migrasi"
        message={`Apakah Anda yakin ingin melakukan rollback ke versi migrasi terpilih? Seluruh perubahan skema yang lebih baru akan dibatalkan melalui SQL down.`}
        confirmText="Ya, Jalankan Rollback"
        variant="danger"
        isLoading={modalLoading}
        onConfirm={handleRollbackConfirm}
        onCancel={() => setModalOpen(false)}
      />

      {/* Top Header & Breadcrumb */}
      <div className={styles.headingRow}>
        <div>
          <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", flexWrap: "wrap" }}>
            <p className={styles.breadcrumb}>{resolvedDatabaseName}</p>
            {database?.access_type && (
              <span
                style={{
                  fontSize: "0.75rem",
                  padding: "2px 8px",
                  borderRadius: "6px",
                  fontWeight: 700,
                  backgroundColor: isViewEdit
                    ? "rgba(16, 185, 129, 0.15)"
                    : "rgba(59, 130, 246, 0.15)",
                  color: isViewEdit ? "#34d399" : "#60a5fa",
                  border: `1px solid ${
                    isViewEdit
                      ? "rgba(16, 185, 129, 0.3)"
                      : "rgba(59, 130, 246, 0.3)"
                  }`,
                }}
              >
                {isViewEdit ? "✓ HAK AKSES: VIEW & EDIT" : "👁 HAK AKSES: VIEW ONLY (READ-ONLY)"}
              </span>
            )}
          </div>
          <h1 className={styles.title}>Schema Snapshot Diff & Migration Management</h1>
        </div>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button
            type="button"
            className={styles.secondaryButton}
            onClick={() => router.push(`/databases/${databaseId}/schema`)}
          >
            Lihat Schema Viewer
          </button>
          <button
            type="button"
            className={styles.secondaryButton}
            onClick={() => router.push(`/databases/${databaseId}`)}
          >
            ← Kembali ke Database
          </button>
        </div>
      </div>

      {/* Access Notice for View Only users */}
      {!isViewEdit && (
        <div
          style={{
            backgroundColor: "rgba(59, 130, 246, 0.1)",
            border: "1px solid rgba(59, 130, 246, 0.3)",
            color: "#93c5fd",
            padding: "0.75rem 1rem",
            borderRadius: "8px",
            fontSize: "0.875rem",
            display: "flex",
            alignItems: "center",
            gap: "0.6rem",
          }}
        >
          <span>👁️</span>
          <span>
            <strong>Mode Akses View Only:</strong> Anda dapat meninjau perbandingan snapshot schema dan melihat execution plan, namun eksekusi migrasi dan rollback skema dibatasi hanya untuk anggota grup dengan akses <strong>View & Edit</strong>.
          </span>
        </div>
      )}

      {error && <div className={styles.error}>{error}</div>}
      {notice && <div className={styles.notice}>{notice}</div>}

      {/* Section 1: Schema Snapshot Diff & History */}
      <section className={styles.panel}>
        <div className={styles.sectionHeader}>
          <div>
            <h2 className={styles.sectionTitle}>Bandingkan Schema Snapshot (Schema Diff)</h2>
            <p className={styles.helpText}>
              Migrasi pertama merepresentasikan <strong>Initial Schema Snapshot (Describe Tables)</strong>. Bandingkan dua versi untuk melihat perubahan tabel, kolom baru/dihapus, maupun JSON snapshot.
            </p>
          </div>
          <div className={styles.topBar} style={{ display: "flex", gap: "0.4rem", flexWrap: "wrap" }}>
            {(["Snapshot Diff", "Snapshot JSON", "SQL Script"] as ViewMode[]).map((mode) => (
              <button
                key={mode}
                type="button"
                className={viewMode === mode ? styles.primaryButton : styles.secondaryButton}
                onClick={() => setViewMode(mode)}
                style={{ fontSize: "0.8rem", padding: "0.45rem 0.85rem" }}
              >
                {mode === "Snapshot Diff"
                  ? "🔍 Perubahan Snapshot"
                  : mode === "Snapshot JSON"
                  ? "{ } Snapshot JSON"
                  : "📜 SQL Script"}
              </button>
            ))}
          </div>
        </div>

        {/* Version Selectors */}
        <div className={styles.compareControls}>
          <label>
            Versi Pertama (Snapshot Awal / Kiri)
            <select
              value={leftMigrationId}
              onChange={(event) => {
                setLeftMigrationId(event.target.value);
                setComparison(null);
              }}
              className={styles.dropdown}
            >
              {migrations.map((migration) => (
                <option key={migration.id} value={migration.id}>
                  Versi {migration.version} ({migration.status})
                </option>
              ))}
            </select>
          </label>
          <label>
            Versi Kedua (Snapshot Target / Kanan)
            <select
              value={rightMigrationId}
              onChange={(event) => {
                setRightMigrationId(event.target.value);
                setComparison(null);
              }}
              className={styles.dropdown}
            >
              {migrations.map((migration) => (
                <option key={migration.id} value={migration.id}>
                  Versi {migration.version} ({migration.status})
                </option>
              ))}
            </select>
          </label>
          <div style={{ display: "flex", gap: "0.5rem" }}>
            <button
              type="button"
              className={styles.primaryButton}
              onClick={handleCompare}
              disabled={busy || migrations.length < 2}
            >
              Bandingkan Snapshot
            </button>
            {rightMigration && rightMigration.id > 0 && (
              <button
                type="button"
                className={styles.secondaryButton}
                onClick={handleUseDiffAsCandidate}
                title="Salin perubahan SQL ini sebagai kandidat migrasi"
                style={{
                  borderColor: "#3b82f6",
                  color: "#60a5fa",
                }}
              >
                📋 Jadikan Kandidat Migrasi
              </button>
            )}
          </div>
        </div>

        {comparison && (
          <div className={styles.comparisonSummary}>
            <span>
              Status Snapshot:{" "}
              <strong style={{ color: snapshotDiff.isIdentical ? "#86efac" : "#fde047" }}>
                {snapshotDiff.isIdentical
                  ? "Identik (Tidak Ada Perubahan Skema)"
                  : `${snapshotDiff.summaryItems.length} Perubahan Terdeteksi`}
              </strong>
            </span>
            <span>
              Status SQL Up:{" "}
              <strong style={{ color: comparison.sql_up_equal ? "#86efac" : "#fca5a5" }}>
                {comparison.sql_up_equal ? "Identik" : "Berbeda"}
              </strong>
            </span>
          </div>
        )}

        {(leftMigration?.error_message || rightMigration?.error_message) && (
          <div className={styles.migrationErrors}>
            {leftMigration?.error_message && (
              <span>
                Versi {leftMigration.version}: {leftMigration.error_message}
              </span>
            )}
            {rightMigration?.error_message && (
              <span>
                Versi {rightMigration.version}: {rightMigration.error_message}
              </span>
            )}
          </div>
        )}

        {/* Mode 1: Visual Snapshot Diff (Tables, Columns, Indexes) */}
        {viewMode === "Snapshot Diff" && (
          <div
            style={{
              backgroundColor: "#09090b",
              border: "1px solid #27272a",
              borderRadius: "10px",
              padding: "1.25rem",
              display: "flex",
              flexDirection: "column",
              gap: "1.25rem",
            }}
          >
            <div
              style={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "center",
                flexWrap: "wrap",
                gap: "0.5rem",
                borderBottom: "1px solid #27272a",
                paddingBottom: "0.75rem",
              }}
            >
              <div>
                <strong style={{ color: "#fafafa", fontSize: "0.95rem" }}>
                  Perbandingan Snapshot: Versi {leftMigration?.version ?? "-"} ➔ Versi {rightMigration?.version ?? "-"}
                </strong>
                <p style={{ color: "#a1a1aa", fontSize: "0.8rem", margin: "0.2rem 0 0" }}>
                  Menampilkan penambahan/penghapusan tabel, kolom baru pada tabel, perubahan tipe data, serta index.
                </p>
              </div>
              <span
                style={{
                  fontSize: "0.78rem",
                  padding: "4px 10px",
                  borderRadius: "999px",
                  fontWeight: 700,
                  backgroundColor: snapshotDiff.isIdentical
                    ? "rgba(16, 185, 129, 0.12)"
                    : "rgba(59, 130, 246, 0.15)",
                  color: snapshotDiff.isIdentical ? "#34d399" : "#60a5fa",
                }}
              >
                {snapshotDiff.summaryItems.length} Perubahan Struktur
              </span>
            </div>

            {leftMigrationId === rightMigrationId ? (
              <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
                <div style={{ color: "#a1a1aa", fontSize: "0.875rem" }}>
                  Menampilkan isi <strong>Snapshot Versi {rightMigration?.version}</strong> ({Object.keys(rightSnapshot?.tables ?? {}).length} tabel terdaftar). Pilih dua versi berbeda di atas untuk melihat perbedaan antar-snapshot.
                </div>
                <div
                  style={{
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))",
                    gap: "1rem",
                  }}
                >
                  {Object.entries(rightSnapshot?.tables ?? {}).map(([tableName, tbl]) => (
                    <div
                      key={tableName}
                      style={{
                        backgroundColor: "#121215",
                        border: "1px solid #27272a",
                        borderRadius: "8px",
                        padding: "1rem",
                      }}
                    >
                      <div style={{ fontWeight: 700, color: "#60a5fa", marginBottom: "0.5rem" }}>
                        📄 Tabel <code>{tableName}</code> ({tbl.columns.length} kolom)
                      </div>
                      <ul style={{ listStyle: "none", padding: 0, margin: 0, display: "flex", flexDirection: "column", gap: "0.3rem" }}>
                        {tbl.columns.map((c) => (
                          <li key={c.name} style={{ fontSize: "0.8rem", color: "#e4e4e7", display: "flex", justifyContent: "space-between" }}>
                            <span><strong>{c.name}</strong></span>
                            <code style={{ color: "#a1a1aa" }}>{c.type} {c.nullable ? "(NULL)" : "(NOT NULL)"}</code>
                          </li>
                        ))}
                      </ul>
                    </div>
                  ))}
                </div>
              </div>
            ) : snapshotDiff.isIdentical ? (
              <div style={{ color: "#a1a1aa", fontSize: "0.875rem", padding: "0.5rem 0" }}>
                ✓ Struktur tabel, kolom, dan index pada kedua snapshot identik (atau migrasi hanya berisi perubahan baris data DML).
              </div>
            ) : (
              <>
                {/* High-level Change Feed (e.g., "Kolom baru di tabel A: ...") */}
                <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
                  <div style={{ fontSize: "0.8rem", fontWeight: 700, color: "#a1a1aa", textTransform: "uppercase" }}>
                    Ringkasan Perubahan Snapshot:
                  </div>
                  {snapshotDiff.summaryItems.map((item, idx) => {
                    const isAdd = item.type.endsWith("_added");
                    const isRemove = item.type.endsWith("_removed");
                    return (
                      <div
                        key={idx}
                        style={{
                          display: "flex",
                          alignItems: "center",
                          gap: "0.75rem",
                          padding: "0.6rem 0.85rem",
                          borderRadius: "8px",
                          backgroundColor: isAdd
                            ? "rgba(16, 185, 129, 0.1)"
                            : isRemove
                            ? "rgba(239, 68, 68, 0.1)"
                            : "rgba(234, 179, 8, 0.1)",
                          border: `1px solid ${
                            isAdd
                              ? "rgba(16, 185, 129, 0.3)"
                              : isRemove
                              ? "rgba(239, 68, 68, 0.3)"
                              : "rgba(234, 179, 8, 0.3)"
                          }`,
                        }}
                      >
                        <span
                          style={{
                            fontSize: "0.7rem",
                            fontWeight: 800,
                            padding: "2px 8px",
                            borderRadius: "4px",
                            backgroundColor: isAdd
                              ? "rgba(16, 185, 129, 0.2)"
                              : isRemove
                              ? "rgba(239, 68, 68, 0.2)"
                              : "rgba(234, 179, 8, 0.2)",
                            color: isAdd ? "#34d399" : isRemove ? "#f87171" : "#facc15",
                            whiteSpace: "nowrap",
                          }}
                        >
                          {item.badgeText}
                        </span>
                        <span style={{ fontSize: "0.875rem", color: "#fafafa", fontWeight: 500 }}>
                          {item.detail}
                        </span>
                      </div>
                    );
                  })}
                </div>

                {/* Per-Table Breakdown */}
                <div
                  style={{
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))",
                    gap: "1rem",
                  }}
                >
                  {snapshotDiff.tableDiffs.map((tDiff) => (
                    <div
                      key={tDiff.tableName}
                      style={{
                        backgroundColor: "#121215",
                        border: "1px solid #27272a",
                        borderRadius: "8px",
                        padding: "1rem",
                        display: "flex",
                        flexDirection: "column",
                        gap: "0.6rem",
                      }}
                    >
                      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                        <strong style={{ color: "#fafafa", fontSize: "0.95rem" }}>
                          📄 Tabel <code>{tDiff.tableName}</code>
                        </strong>
                        <span
                          style={{
                            fontSize: "0.72rem",
                            fontWeight: 700,
                            color:
                              tDiff.status === "added"
                                ? "#34d399"
                                : tDiff.status === "removed"
                                ? "#f87171"
                                : "#60a5fa",
                          }}
                        >
                          {tDiff.status === "added"
                            ? "Tabel Baru"
                            : tDiff.status === "removed"
                            ? "Tabel Dihapus"
                            : "Struktur Berubah"}
                        </span>
                      </div>

                      {tDiff.addedColumns.map((col) => (
                        <div key={`add-${col.name}`} style={{ fontSize: "0.82rem", color: "#86efac" }}>
                          + Kolom baru: <strong>{col.name}</strong> (<code>{col.type}</code>, {col.nullable ? "NULL" : "NOT NULL"})
                        </div>
                      ))}
                      {tDiff.removedColumns.map((col) => (
                        <div key={`rem-${col.name}`} style={{ fontSize: "0.82rem", color: "#fca5a5" }}>
                          - Kolom dihapus: <strong>{col.name}</strong> (<code>{col.type}</code>)
                        </div>
                      ))}
                      {tDiff.modifiedColumns.map((mod) => (
                        <div key={`mod-${mod.after.name}`} style={{ fontSize: "0.82rem", color: "#fde047" }}>
                          ~ Kolom diubah: <strong>{mod.after.name}</strong> (<code>{mod.before.type}</code> ➔ <code>{mod.after.type}</code>)
                        </div>
                      ))}
                      {tDiff.addedIndexes.map((idx) => (
                        <div key={`add-idx-${idx.name}`} style={{ fontSize: "0.82rem", color: "#38bdf8" }}>
                          + Index baru: <code>{idx.name}</code> ({idx.unique ? "UNIQUE" : "NON-UNIQUE"})
                        </div>
                      ))}
                      {tDiff.removedIndexes.map((idx) => (
                        <div key={`rem-idx-${idx.name}`} style={{ fontSize: "0.82rem", color: "#f87171" }}>
                          - Index dihapus: <code>{idx.name}</code>
                        </div>
                      ))}
                    </div>
                  ))}
                </div>
              </>
            )}
          </div>
        )}

        {/* Mode 2: Side-by-Side Developer JSON Snapshot */}
        {viewMode === "Snapshot JSON" && (
          <div className={styles.diffWrapper}>
            <div className={styles.diffHeader}>
              <strong>Snapshot JSON — Versi {leftMigration?.version ?? "-"}</strong>
              <strong>Snapshot JSON — Versi {rightMigration?.version ?? "-"}</strong>
            </div>
            <div className={styles.diffContainer}>
              <div className={styles.editor}>
                {jsonDiffLines.map((line) => (
                  <div
                    key={line.index}
                    className={line.equal ? styles.lineUnchanged : styles.lineRemoved}
                  >
                    <span className={styles.lineNumber}>{line.index + 1}</span>
                    {line.left || " "}
                  </div>
                ))}
              </div>
              <div className={styles.editor}>
                {jsonDiffLines.map((line) => (
                  <div
                    key={line.index}
                    className={line.equal ? styles.lineUnchanged : styles.lineAdded}
                  >
                    <span className={styles.lineNumber}>{line.index + 1}</span>
                    {line.right || " "}
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* Mode 3: SQL Script Diff */}
        {viewMode === "SQL Script" && (
          <div className={styles.diffWrapper}>
            <div className={styles.diffHeader}>
              <strong>SQL Up — Versi {leftMigration?.version ?? "-"}</strong>
              <strong>SQL Up — Versi {rightMigration?.version ?? "-"}</strong>
            </div>
            <div className={styles.diffContainer}>
              <div className={styles.editor}>
                {sqlDiffLines.map((line) => (
                  <div
                    key={line.index}
                    className={line.equal ? styles.lineUnchanged : styles.lineRemoved}
                  >
                    <span className={styles.lineNumber}>{line.index + 1}</span>
                    {line.left || " "}
                  </div>
                ))}
              </div>
              <div className={styles.editor}>
                {sqlDiffLines.map((line) => (
                  <div
                    key={line.index}
                    className={line.equal ? styles.lineUnchanged : styles.lineAdded}
                  >
                    <span className={styles.lineNumber}>{line.index + 1}</span>
                    {line.right || " "}
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </section>

      {/* Section 2: Migration Generation & Execution Plan Analysis */}
      <section className={styles.panel}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: "0.5rem" }}>
          <div>
            <h2 className={styles.sectionTitle}>Eksekusi Migrasi Baru (Multi-Query Atomic) & Analisis Execution Plan</h2>
            <p className={styles.helpText}>
              Masukkan satu atau beberapa perintah migrasi dipisahkan dengan titik koma (<code>;</code>) seperti CREATE TABLE, ALTER TABLE, INSERT, UPDATE, atau DELETE. Seluruh query dieksekusi sebagai 1 migrasi atomik — jika ada satu query gagal, perubahan sebelumnya otomatis di-rollback ke kondisi awal.
            </p>
          </div>
          <span
            style={{
              fontSize: "0.75rem",
              color: isViewEdit ? "#86efac" : "#fca5a5",
              fontWeight: 600,
            }}
          >
            {isViewEdit ? "Izin Eksekusi: Aktif" : "Izin Eksekusi: Diblokir (View Only)"}
          </span>
        </div>

        <textarea
          className={styles.rollbackTextarea}
          value={migrationQuery}
          onChange={(event) => setMigrationQuery(event.target.value)}
          placeholder={"Contoh multi-query dalam 1 migrasi:\nCREATE TABLE orders (id INT PRIMARY KEY, customer_id INT);\nALTER TABLE orders ADD COLUMN total INT;\nINSERT INTO orders (id, customer_id, total) VALUES (1, 10, 50000);"}
          rows={5}
          style={{ fontFamily: "monospace", fontSize: "0.875rem" }}
        />

        {/* Action Buttons: Explain Plan & Execute */}
        <div style={{ display: "flex", gap: "0.75rem", flexWrap: "wrap", alignItems: "center" }}>
          <button
            type="button"
            className={styles.secondaryButton}
            onClick={() => handleExplain(false)}
            disabled={explainLoading || !migrationQuery.trim()}
            style={{ display: "inline-flex", alignItems: "center", gap: "0.4rem" }}
          >
            🔍 Analisis Plan (EXPLAIN)
          </button>

          <button
            type="button"
            className={styles.secondaryButton}
            onClick={() => handleExplain(true)}
            disabled={explainLoading || !migrationQuery.trim()}
            style={{ display: "inline-flex", alignItems: "center", gap: "0.4rem" }}
          >
            ⚡ EXPLAIN ANALYZE
          </button>

          <button
            type="button"
            className={styles.primaryButton}
            onClick={handleExecute}
            disabled={busy || !isViewEdit || !migrationQuery.trim()}
            style={{
              backgroundColor: isViewEdit ? "#2563eb" : "#3f3f46",
              cursor: isViewEdit && migrationQuery.trim() ? "pointer" : "not-allowed",
              marginLeft: "auto",
            }}
          >
            {busy ? "Mengeksekusi..." : "+ Eksekusi Migrasi"}
          </button>
        </div>

        {/* Execution Plan Analysis Card */}
        {planAnalysis && (
          <div
            style={{
              marginTop: "0.75rem",
              backgroundColor: "#0d1117",
              border: "1px solid #30363d",
              borderRadius: "8px",
              padding: "1rem",
              display: "flex",
              flexDirection: "column",
              gap: "0.75rem",
            }}
          >
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <span style={{ fontSize: "0.875rem", fontWeight: 700, color: "#58a6ff" }}>
                📊 Query Execution Plan ({planAnalysis.isAnalyze ? "EXPLAIN ANALYZE" : "EXPLAIN"})
              </span>
              <button
                type="button"
                onClick={() => setPlanAnalysis(null)}
                style={{
                  background: "none",
                  border: "none",
                  color: "#8b949e",
                  cursor: "pointer",
                  fontSize: "0.8rem",
                }}
              >
                Tutup ✕
              </button>
            </div>

            {/* Quick Metrics Badges */}
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))",
                gap: "0.5rem",
              }}
            >
              {planAnalysis.scanType && (
                <div style={{ backgroundColor: "#161b22", padding: "0.5rem 0.75rem", borderRadius: "6px" }}>
                  <span style={{ fontSize: "0.7rem", color: "#8b949e", display: "block" }}>Jenis Scan:</span>
                  <strong style={{ fontSize: "0.8rem", color: "#f0883e" }}>{planAnalysis.scanType}</strong>
                </div>
              )}
              {planAnalysis.indexUsed && (
                <div style={{ backgroundColor: "#161b22", padding: "0.5rem 0.75rem", borderRadius: "6px" }}>
                  <span style={{ fontSize: "0.7rem", color: "#8b949e", display: "block" }}>Penggunaan Index:</span>
                  <strong style={{ fontSize: "0.8rem", color: "#3fb950" }}>{planAnalysis.indexUsed}</strong>
                </div>
              )}
              {planAnalysis.joinStrategy && (
                <div style={{ backgroundColor: "#161b22", padding: "0.5rem 0.75rem", borderRadius: "6px" }}>
                  <span style={{ fontSize: "0.7rem", color: "#8b949e", display: "block" }}>Join Strategy:</span>
                  <strong style={{ fontSize: "0.8rem", color: "#d2a8ff" }}>{planAnalysis.joinStrategy}</strong>
                </div>
              )}
              {planAnalysis.estimatedCost && (
                <div style={{ backgroundColor: "#161b22", padding: "0.5rem 0.75rem", borderRadius: "6px" }}>
                  <span style={{ fontSize: "0.7rem", color: "#8b949e", display: "block" }}>Estimated Cost:</span>
                  <strong style={{ fontSize: "0.8rem", color: "#58a6ff" }}>{planAnalysis.estimatedCost}</strong>
                </div>
              )}
              {planAnalysis.actualTime && (
                <div style={{ backgroundColor: "#161b22", padding: "0.5rem 0.75rem", borderRadius: "6px" }}>
                  <span style={{ fontSize: "0.7rem", color: "#8b949e", display: "block" }}>Actual Time:</span>
                  <strong style={{ fontSize: "0.8rem", color: "#7ee787" }}>{planAnalysis.actualTime}</strong>
                </div>
              )}
              {planAnalysis.rowsProcessed && (
                <div style={{ backgroundColor: "#161b22", padding: "0.5rem 0.75rem", borderRadius: "6px" }}>
                  <span style={{ fontSize: "0.7rem", color: "#8b949e", display: "block" }}>Jumlah Baris:</span>
                  <strong style={{ fontSize: "0.8rem", color: "#e6edf3" }}>{planAnalysis.rowsProcessed} rows</strong>
                </div>
              )}
            </div>

            {/* Raw Execution Plan Output */}
            <pre
              style={{
                margin: 0,
                padding: "0.75rem",
                backgroundColor: "#161b22",
                border: "1px solid #21262d",
                borderRadius: "6px",
                color: "#e6edf3",
                fontSize: "0.8rem",
                fontFamily: "monospace",
                overflowX: "auto",
                whiteSpace: "pre-wrap",
                lineHeight: 1.4,
              }}
            >
              {planAnalysis.rawText}
            </pre>
          </div>
        )}
      </section>

      {/* Section 3: Rollback */}
      <section className={styles.panel}>
        <div>
          <h2 className={styles.sectionTitle}>Rollback Migrasi Skema</h2>
          <p className={styles.helpText}>
            Kembalikan skema database ke versi tertentu. Backend akan otomatis menjalankan SQL down secara bertahap.
          </p>
        </div>
        <div className={styles.actionRow}>
          <select
            value={rollbackTargetId}
            onChange={(event) => setRollbackTargetId(event.target.value)}
            className={styles.dropdown}
            disabled={successfulMigrations.length === 0 || !isViewEdit}
          >
            {successfulMigrations.map((migration) => (
              <option key={migration.id} value={migration.id}>
                Versi {migration.version}
              </option>
            ))}
          </select>
          <button
            type="button"
            className={styles.dangerButton}
            onClick={promptRollback}
            disabled={busy || successfulMigrations.length === 0 || !isViewEdit}
            style={{
              cursor: isViewEdit && successfulMigrations.length > 0 ? "pointer" : "not-allowed",
              opacity: isViewEdit && successfulMigrations.length > 0 ? 1 : 0.5,
            }}
          >
            Rollback ke Versi Terpilih
          </button>
        </div>
      </section>
    </div>
  );
}
