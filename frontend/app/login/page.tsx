"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import styles from "../page.module.css";
import { api } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setLoading(true);

    try {
      await api.login(username, password);
      router.push("/databases");
      router.refresh();
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to sign in.",
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className={styles.page}>
      <main className={styles.main}>
        <div className={styles.intro}>
          <h1>
            <code className={styles.code}>Login</code>
          </h1>
        </div>

        <form onSubmit={handleSubmit} className={styles.intro}>
          <label htmlFor="username">Username</label>
          <input
            id="username"
            type="text"
            required
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            placeholder="Input username"
            className={styles.searchInput}
          />

          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            required
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="Input password"
            className={styles.searchInput}
          />

          {error && <p role="alert">{error}</p>}

          <button
            type="submit"
            disabled={loading}
            className={styles.primary}
          >
            {loading ? "Signing in..." : "Sign in"}
          </button>
        </form>
      </main>
    </div>
  );
}
