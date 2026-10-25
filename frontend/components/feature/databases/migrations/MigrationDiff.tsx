"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { Migration, MigrationComparison } from "@/lib/types";
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

  const loadData = useCallback(async () => {
    try {
      const [migrationResponse, database] = await Promise.all([
        api.migrations(databaseId, 1, 100),
        api.database(databaseId),
      ]);
      const loaded = migrationResponse.data ?? [];
      const latest = loaded.at(-1);
      const previous = loaded.at(-2) ?? latest;
      const latestSuccessful = [...loaded]
        .reverse()
        .find((migration) => migration.status === "success");

      setMigrations(loaded);
      setResolvedDatabaseName(database.name);
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
      setError("Select two migrations to compare.");
      return;
    }
    if (leftMigrationId === rightMigrationId) {
      setError("Choose two different migration versions.");
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
    } catch (compareError) {
      setError(errorMessage(compareError));
    } finally {
      setBusy(false);
    }
  };

  const handleExecute = async () => {
    const sql = migrationQuery.trim();
    if (!sql) {
      setError("Enter an SQL migration before executing it.");
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
          `Migration version ${created.version} failed and was recorded: ${created.error_message || "Unknown database error"}`,
        );
      } else {
        setMigrationQuery("");
        setNotice(`Migration version ${created.version} executed successfully.`);
      }
    } catch (executeError) {
      setError(errorMessage(executeError));
    } finally {
      setBusy(false);
    }
  };

  const handleRollback = async () => {
    if (!rollbackTargetId) {
      setError("Select a target migration for rollback.");
      return;
    }

    setBusy(true);
    setError("");
    setNotice("");
    try {
      const rolledBack = await api.rollback(
        Number(databaseId),
        Number(rollbackTargetId),
      );
      setNotice(
        rolledBack.length > 0
          ? `${rolledBack.length} migration(s) rolled back successfully.`
          : "Database is already at the selected migration.",
      );
      await loadData();
    } catch (rollbackError) {
      setError(errorMessage(rollbackError));
    } finally {
      setBusy(false);
    }
  };

  if (loading) {
    return <div className={styles.message}>Loading migrations...</div>;
  }

  return (
    <div className={styles.container}>
      <div className={styles.headingRow}>
        <div>
          <p className={styles.breadcrumb}>{resolvedDatabaseName}</p>
          <h1 className={styles.title}>Database migrations</h1>
        </div>
        <button
          type="button"
          className={styles.secondaryButton}
          onClick={() => router.push(`/databases/${databaseId}/schema`)}
        >
          Back to Schema
        </button>
      </div>

      {error && <div className={styles.error}>{error}</div>}
      {notice && <div className={styles.notice}>{notice}</div>}

      <section className={styles.panel}>
        <div className={styles.sectionHeader}>
          <div>
            <h2 className={styles.sectionTitle}>Migration history</h2>
            <p className={styles.helpText}>
              These versions are loaded from the backend migration records.
            </p>
          </div>
          <select
            value={viewMode}
            onChange={(event) => setViewMode(event.target.value as ViewMode)}
            className={styles.dropdown}
          >
            <option value="Schema Diff">Changed lines</option>
            <option value="Full Script">Full SQL script</option>
          </select>
        </div>

        {migrations.length === 0 ? (
          <div className={styles.emptyDiff}>
            No migration has been recorded for this database yet.
          </div>
        ) : (
          <>
            <div className={styles.compareControls}>
              <label>
                First version
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
                      Version {migration.version} ({migration.status})
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Second version
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
                      Version {migration.version} ({migration.status})
                    </option>
                  ))}
                </select>
              </label>
              <button
                type="button"
                className={styles.primaryButton}
                onClick={handleCompare}
                disabled={busy || migrations.length < 2}
              >
                Compare
              </button>
            </div>

            {comparison && (
              <div className={styles.comparisonSummary}>
                <span>SQL up: {comparison.sql_up_equal ? "equal" : "different"}</span>
                <span>SQL down: {comparison.sql_down_equal ? "equal" : "different"}</span>
              </div>
            )}

            {(leftMigration?.error_message || rightMigration?.error_message) && (
              <div className={styles.migrationErrors}>
                {leftMigration?.error_message && (
                  <span>
                    Version {leftMigration.version}: {leftMigration.error_message}
                  </span>
                )}
                {rightMigration?.error_message && (
                  <span>
                    Version {rightMigration.version}: {rightMigration.error_message}
                  </span>
                )}
              </div>
            )}

            <div className={styles.diffWrapper}>
              <div className={styles.diffHeader}>
                <strong>Version {leftMigration?.version ?? "-"}</strong>
                <strong>Version {rightMigration?.version ?? "-"}</strong>
              </div>
              <div className={styles.diffContainer}>
                {diffLines.length === 0 ? (
                  <div className={styles.emptyDiff}>No SQL differences found.</div>
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

      <section className={styles.panel}>
        <h2 className={styles.sectionTitle}>Execute migration</h2>
        <p className={styles.helpText}>
          Enter one supported SQL statement. The backend executes it, generates the
          rollback SQL, and stores the resulting migration version.
        </p>
        <textarea
          className={styles.rollbackTextarea}
          value={migrationQuery}
          onChange={(event) => setMigrationQuery(event.target.value)}
          placeholder="Example: ALTER TABLE customers ADD COLUMN phone VARCHAR(30);"
        />
        <button
          type="button"
          className={styles.primaryButton}
          onClick={handleExecute}
          disabled={busy || !migrationQuery.trim()}
        >
          {busy ? "Processing..." : "Execute Migration"}
        </button>
      </section>

      <section className={styles.panel}>
        <h2 className={styles.sectionTitle}>Rollback</h2>
        <p className={styles.helpText}>
          Roll back every migration newer than the selected target version.
        </p>
        <div className={styles.actionRow}>
          <select
            value={rollbackTargetId}
            onChange={(event) => setRollbackTargetId(event.target.value)}
            className={styles.dropdown}
            disabled={successfulMigrations.length === 0}
          >
            {successfulMigrations.map((migration) => (
              <option key={migration.id} value={migration.id}>
                Version {migration.version}
              </option>
            ))}
          </select>
          <button
            type="button"
            className={styles.dangerButton}
            onClick={handleRollback}
            disabled={busy || successfulMigrations.length === 0}
          >
            Rollback to Version
          </button>
        </div>
      </section>
    </div>
  );
}
