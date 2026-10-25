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
import styles from "./MigrationDiff.module.css";

type ViewMode = "Schema Diff" | "Full Script";

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
  const [leftMigrationId, setLeftMigrationId] = useState("");
  const [rightMigrationId, setRightMigrationId] = useState("");
  const [rollbackTargetId, setRollbackTargetId] = useState("");
  const [comparison, setComparison] = useState<MigrationComparison | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>("Schema Diff");
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
      const loaded = migrationResponse.data ?? [];
      const latest = loaded.at(-1);
      const previous = loaded.at(-2) ?? latest;
      const latestSuccessful = [...loaded]
        .reverse()
        .find((migration) => migration.status === "success");

      setDatabase(db);
      setMigrations(loaded);
      setResolvedDatabaseName(db.name);
      setLeftMigrationId(previous ? String(previous.id) : "");
      setRightMigrationId(latest ? String(latest.id) : "");
      setRollbackTargetId(latestSuccessful ? String(latestSuccessful.id) : "");
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
    () => migrations.filter((migration) => migration.status === "success"),
    [migrations],
  );

  const diffLines = useMemo<DiffLine[]>(() => {
    const leftLines = leftMigration?.sql_up.split("\n") ?? [];
    const rightLines = rightMigration?.sql_up.split("\n") ?? [];
    const lineCount = Math.max(leftLines.length, rightLines.length);

    return Array.from({ length: lineCount }, (_, index) => {
      const left = leftLines[index] ?? "";
      const right = rightLines[index] ?? "";
      return { index, left, right, equal: left === right };
    }).filter((line) => viewMode === "Full Script" || !line.equal);
  }, [leftMigration, rightMigration, viewMode]);

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
      const result = await api.compareMigrations(
        databaseId,
        leftMigrationId,
        rightMigrationId,
      );
      setComparison(result);
      toast.info("Perbandingan schema migrasi berhasil ditampilkan.", "Schema Diff");
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

    // Filter added or modified SQL lines from the right migration
    const addedLines = diffLines
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

      const explainableRegex = /^(SELECT|WITH|INSERT|UPDATE|DELETE|VALUES|TABLE)\b/i;
      const ddlRegex = /^(CREATE|ALTER|DROP|TRUNCATE|RENAME)\b/i;

      const targetQuery = statements.find((stmt) => explainableRegex.test(stmt));

      if (!targetQuery && statements.some((stmt) => ddlRegex.test(stmt))) {
        const ddlStatements = statements.filter((stmt) => ddlRegex.test(stmt));
        setPlanAnalysis({
          isAnalyze: analyze,
          rawText:
            `-- Analisis Statis DDL (${ddlStatements.length} statement)\n` +
            `-- Catatan: SQL EXPLAIN pada mesin database hanya berlaku untuk query SELECT / DML.\n\n` +
            ddlStatements.map((s, idx) => `${idx + 1}. ${s};`).join("\n"),
          scanType: "DDL Schema Operation (Metadata Lock)",
          indexUsed: /INDEX/i.test(sql) ? "Pembuatan / Modifikasi Index" : "N/A (DDL)",
          joinStrategy: "Tidak ada (Operasi Skema)",
        });
        toast.info(
          "Statement DDL terdeteksi. Menampilkan analisis statis operasi skema.",
          "Analisis DDL",
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
          `Migration versi ${created.version} gagal: ${created.error_message || "Unknown database error"}`,
        );
        toast.error(`Migrasi versi ${created.version} gagal: ${created.error_message}`, "Migrasi Gagal");
      } else {
        setMigrationQuery("");
        setPlanAnalysis(null);
        setNotice(`Migrasi versi ${created.version} berhasil dieksekusi.`);
        toast.success(`Migrasi versi ${created.version} berhasil dieksekusi secara otomatis!`, "Migrasi Berhasil");
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
    return <div className={styles.message}>Memuat data migrasi database...</div>;
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
          <h1 className={styles.title}>Schema Diff & Migration Management</h1>
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

      {/* Access Permission Warning Banner if View Only */}
      {!isViewEdit && (
        <div
          style={{
            backgroundColor: "rgba(59, 130, 246, 0.1)",
            border: "1px solid rgba(59, 130, 246, 0.3)",
            borderRadius: "8px",
            padding: "0.85rem 1rem",
            color: "#93c5fd",
            fontSize: "0.875rem",
            display: "flex",
            alignItems: "center",
            gap: "0.5rem",
          }}
        >
          <span>👁️</span>
          <span>
            <strong>Mode Akses View Only:</strong> Anda dapat meninjau schema diff dan melihat execution plan, namun eksekusi migrasi dan rollback skema dibatasi hanya untuk anggota grup dengan akses <strong>View & Edit</strong>.
          </span>
        </div>
      )}

      {error && <div className={styles.error}>{error}</div>}
      {notice && <div className={styles.notice}>{notice}</div>}

      {/* Section 1: Schema Diff & History */}
      <section className={styles.panel}>
        <div className={styles.sectionHeader}>
          <div>
            <h2 className={styles.sectionTitle}>Bandingkan Schema (Schema Diff)</h2>
            <p className={styles.helpText}>
              Bandingkan dua kondisi versi schema migrasi untuk melihat perbedaan struktur secara visual.
            </p>
          </div>
          <div className={styles.topBar}>
            <button
              type="button"
              className={styles.secondaryButton}
              onClick={() =>
                setViewMode((current) =>
                  current === "Schema Diff" ? "Full Script" : "Schema Diff",
                )
              }
            >
              Mode: {viewMode}
            </button>
          </div>
        </div>

        {migrations.length < 2 ? (
          <div className={styles.message}>
            Minimal dibutuhkan 2 versi migrasi untuk melakukan perbandingan schema diff. Saat ini terdapat {migrations.length} versi.
          </div>
        ) : (
          <>
            <div className={styles.compareControls}>
              <label>
                Versi Pertama (Kiri)
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
                Versi Kedua (Kanan)
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
                  Bandingkan
                </button>
                {diffLines.length > 0 && rightMigration && (
                  <button
                    type="button"
                    className={styles.secondaryButton}
                    onClick={handleUseDiffAsCandidate}
                    title="Salin perubahan schema ini sebagai kandidat SQL migrasi"
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
                  Status SQL Up:{" "}
                  <strong style={{ color: comparison.sql_up_equal ? "#86efac" : "#fca5a5" }}>
                    {comparison.sql_up_equal ? "Identik (Sama)" : "Berbeda (Terdapat Perubahan)"}
                  </strong>
                </span>
                <span>
                  Status SQL Down:{" "}
                  <strong style={{ color: comparison.sql_down_equal ? "#86efac" : "#fca5a5" }}>
                    {comparison.sql_down_equal ? "Identik" : "Berbeda"}
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

            <div className={styles.diffWrapper}>
              <div className={styles.diffHeader}>
                <strong>Versi {leftMigration?.version ?? "-"}</strong>
                <strong>Versi {rightMigration?.version ?? "-"}</strong>
              </div>
              <div className={styles.diffContainer}>
                {diffLines.length === 0 ? (
                  <div className={styles.emptyDiff}>Tidak ada perbedaan SQL di antara kedua versi.</div>
                ) : (
                  <>
                    <div className={styles.editor}>
                      {diffLines.map((line) => (
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
                      {diffLines.map((line) => (
                        <div
                          key={line.index}
                          className={line.equal ? styles.lineUnchanged : styles.lineAdded}
                        >
                          <span className={styles.lineNumber}>{line.index + 1}</span>
                          {line.right || " "}
                        </div>
                      ))}
                    </div>
                  </>
                )}
              </div>
            </div>
          </>
        )}
      </section>

      {/* Section 2: Migration Generation & Execution Plan Analysis */}
      <section className={styles.panel}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: "0.5rem" }}>
          <div>
            <h2 className={styles.sectionTitle}>Eksekusi Migrasi & Analisis Execution Plan</h2>
            <p className={styles.helpText}>
              Tinjau kandidat SQL migrasi di bawah. Jalankan EXPLAIN untuk menganalisis execution plan sebelum dieksekusi.
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
          placeholder="Contoh: ALTER TABLE customers ADD COLUMN phone VARCHAR(30); atau CREATE TABLE ..."
          rows={4}
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
