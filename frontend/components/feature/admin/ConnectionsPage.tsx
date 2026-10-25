"use client";

import { FormEvent, useEffect, useState } from "react";
import { api, unwrapPaged } from "@/lib/api";
import type { CreateDatabaseInput, Database, DatabaseType } from "@/lib/types";
import styles from "./ConnectionsPage.module.css";

const emptyForm: CreateDatabaseInput = {
  name: "",
  host: "",
  port: 5432,
  user: "",
  password: "",
  type: "postgresql",
};

export default function ConnectionsPage() {
  const [databases, setDatabases] = useState<Database[]>([]);
  const [form, setForm] = useState<CreateDatabaseInput>(emptyForm);
  const [error, setError] = useState("");

  function load() {
    api.adminDatabases()
      .then((body) => setDatabases(unwrapPaged(body)))
      .catch((requestError: unknown) => {
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Unable to load databases.",
        );
      });
  }

  useEffect(load, []);

  function updateForm(field: keyof CreateDatabaseInput, value: string) {
    setForm((current) => ({
      ...current,
      [field]: field === "port" ? Number(value) : value,
    }));
  }

  async function createDatabase(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");

    try {
      await api.createDatabase(form);
      setForm(emptyForm);
      load();
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to create database.",
      );
    }
  }

  async function deleteDatabase(id: number) {
    try {
      await api.deleteDatabase(id);
      setDatabases((current) => current.filter((database) => database.id !== id));
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to delete database.",
      );
    }
  }

  return (
    <section className={styles.container}>
      <h1>Database connections</h1>
      {error && <p role="alert">{error}</p>}

      <form onSubmit={createDatabase} className={styles.form}>
        <input
          required
          placeholder="Name"
          value={form.name}
          onChange={(event) => updateForm("name", event.target.value)}
        />
        <input
          required
          placeholder="Host"
          value={form.host}
          onChange={(event) => updateForm("host", event.target.value)}
        />
        <input
          required
          type="number"
          min="1"
          max="65535"
          placeholder="Port"
          value={form.port}
          onChange={(event) => updateForm("port", event.target.value)}
        />
        <input
          required
          placeholder="User"
          value={form.user}
          onChange={(event) => updateForm("user", event.target.value)}
        />
        <input
          required
          type="password"
          placeholder="Password"
          value={form.password}
          onChange={(event) => updateForm("password", event.target.value)}
        />
        <select
          value={form.type}
          onChange={(event) =>
            updateForm("type", event.target.value as DatabaseType)
          }
        >
          <option value="postgresql">PostgreSQL</option>
          <option value="mysql">MySQL</option>
        </select>
        <button type="submit">Add database</button>
      </form>

      <div className={styles.grid}>
        {databases.map((database) => (
          <article key={database.id} className={styles.card}>
            <h2>{database.name}</h2>
            <p>
              {database.type} at {database.host}:{database.port}
            </p>
            <button type="button" onClick={() => deleteDatabase(database.id)}>
              Delete
            </button>
          </article>
        ))}
      </div>
    </section>
  );
}
