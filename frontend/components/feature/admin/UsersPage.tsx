"use client";

import { useEffect, useState } from "react";
import { api, unwrapPaged } from "@/lib/api";
import type { User } from "@/lib/types";
import styles from "./UsersPage.module.css";

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    api.users()
      .then((body) => setUsers(unwrapPaged(body)))
      .catch((requestError: unknown) => {
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Unable to load users.",
        );
      });
  }, []);

  return (
    <section className={styles.container}>
      <h1>Users</h1>
      {error && <p role="alert">{error}</p>}
      <table className={styles.dataTable}>
        <thead>
          <tr>
            <th>ID</th>
            <th>Username</th>
            <th>Role</th>
          </tr>
        </thead>
        <tbody>
          {users.map((user) => (
            <tr key={user.id}>
              <td>{user.id}</td>
              <td>{user.username}</td>
              <td>{user.role}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
