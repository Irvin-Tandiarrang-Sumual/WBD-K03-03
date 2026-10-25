"use client";

import Link from "next/link";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { api, unwrapPaged } from "@/lib/api";
import type { AccessType, Database, Group } from "@/lib/types";
import styles from "./GroupsPage.module.css";

export default function GroupsPage() {
  const [groups, setGroups] = useState<Group[]>([]);
  const [databases, setDatabases] = useState<Database[]>([]);
  const [query, setQuery] = useState("");
  const [name, setName] = useState("");
  const [databaseId, setDatabaseId] = useState("");
  const [accessType, setAccessType] = useState<AccessType>("view_only");
  const [error, setError] = useState("");

  function load() {
    Promise.all([api.groups(), api.adminDatabases()])
      .then(([groupResponse, databaseResponse]) => {
        setGroups(unwrapPaged(groupResponse));
        setDatabases(databaseResponse.data || []);
      })
      .catch((requestError: unknown) => {
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Unable to load groups.",
        );
      });
  }

  useEffect(load, []);

  const visibleGroups = useMemo(
    () =>
      groups.filter((group) =>
        group.name.toLowerCase().includes(query.toLowerCase()),
      ),
    [groups, query],
  );

  async function createGroup(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!name.trim() || !databaseId) return;

    try {
      await api.createGroup({
        name: name.trim(),
        databaseAccesses: [
          {
            databaseId: Number(databaseId),
            accessType,
          },
        ],
      });
      setName("");
      load();
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to create group.",
      );
    }
  }

  return (
    <section className={styles.container}>
      <header className={styles.header}>
        <h1>Groups</h1>
      </header>

      {error && <p role="alert">{error}</p>}

      <input
        type="search"
        placeholder="Search group name..."
        value={query}
        onChange={(event) => setQuery(event.target.value)}
      />

      <div className={styles.grid}>
        {visibleGroups.map((group) => (
          <article className={styles.card} key={group.id}>
            <h2>{group.name}</h2>
            <p>{group.databaseAccesses.length} database permissions</p>
            <Link href={`/admin/groups/${group.id}`}>Manage group</Link>
          </article>
        ))}
      </div>

      <form onSubmit={createGroup}>
        <input
          required
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="New group name"
        />
        <select
          required
          value={databaseId}
          onChange={(event) => setDatabaseId(event.target.value)}
        >
          <option value="">Select database</option>
          {databases.map((database) => (
            <option key={database.id} value={database.id}>
              {database.name}
            </option>
          ))}
        </select>
        <select
          value={accessType}
          onChange={(event) => setAccessType(event.target.value as AccessType)}
        >
          <option value="view_only">View only</option>
          <option value="view_edit">View and edit</option>
        </select>
        <button type="submit">Create group</button>
      </form>
    </section>
  );
}
