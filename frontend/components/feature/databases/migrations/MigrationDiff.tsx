"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { api, unwrapPaged } from "@/lib/api";
import type { Migration } from "@/lib/types";
import styles from "./MigrationDiff.module.css";

interface MigrationDiffProps {
  databaseId: string;
}

export default function MigrationDiff({ databaseId }: MigrationDiffProps) {
  const [migrations, setMigrations] = useState<Migration[]>([]);
  const [sql, setSql] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const loadMigrations = useCallback(() => {
    api.migrations(databaseId)
      .then((body) => setMigrations(unwrapPaged(body)))
      .catch((requestError: unknown) => {
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Unable to load migrations.",
        );
      });
  }, [databaseId]);

  useEffect(() => {
    if (databaseId) loadMigrations();
  }, [databaseId, loadMigrations]);

  async function createMigration(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setLoading(true);

    try {
      await api.createMigration(databaseId, sql);
      setSql("");
      loadMigrations();
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to create migration.",
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className={styles.container}>
      <h1>Migration history</h1>

      {error && <p role="alert">{error}</p>}

      <form onSubmit={createMigration}>
        <label htmlFor="migration-sql">SQL migration</label>
        <textarea
          id="migration-sql"
          required
          value={sql}
          onChange={(event) => setSql(event.target.value)}
          placeholder="CREATE TABLE ..."
        />
        <button type="submit" disabled={loading}>
          {loading ? "Running..." : "Run migration"}
        </button>
      </form>

      <div className={styles.tableWrapper}>
        <table className={styles.dataTable}>
          <thead>
            <tr>
              <th>Version</th>
              <th>Status</th>
              <th>SQL up</th>
              <th>SQL down</th>
            </tr>
          </thead>
          <tbody>
            {migrations.map((migration) => (
              <tr key={migration.id}>
                <td>{migration.version}</td>
                <td>{migration.status}</td>
                <td>
                  <code>{migration.sql_up}</code>
                </td>
                <td>
                  <code>{migration.sql_down}</code>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
