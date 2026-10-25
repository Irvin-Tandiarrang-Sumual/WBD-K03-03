"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, unwrapPaged } from "@/lib/api";
import type { User } from "@/lib/types";

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .users()
      .then((body) => setUsers(unwrapPaged(body)))
      .catch((requestError: unknown) => {
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Gagal memuat daftar pengguna.",
        );
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  return (
    <div
      style={{
        maxWidth: "1000px",
        margin: "0 auto",
        padding: "2rem 1.5rem",
        display: "flex",
        flexDirection: "column",
        gap: "1.5rem",
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          flexWrap: "wrap",
          gap: "1rem",
          borderBottom: "1px solid #27272a",
          paddingBottom: "1rem",
        }}
      >
        <div>
          <Link
            href="/admin"
            style={{
              color: "#a1a1aa",
              fontSize: "0.85rem",
              textDecoration: "none",
              display: "inline-block",
              marginBottom: "0.25rem",
            }}
          >
            ← Kembali ke Admin Dashboard
          </Link>
          <h1 style={{ fontSize: "1.75rem", fontWeight: 800, color: "#fafafa" }}>
            User Management
          </h1>
        </div>
      </div>

      {error && (
        <div
          style={{
            backgroundColor: "rgba(239, 68, 68, 0.1)",
            border: "1px solid rgba(239, 68, 68, 0.3)",
            color: "#fca5a5",
            padding: "0.75rem 1rem",
            borderRadius: "8px",
            fontSize: "0.875rem",
          }}
          role="alert"
        >
          {error}
        </div>
      )}

      <div
        style={{
          backgroundColor: "#121215",
          border: "1px solid #27272a",
          borderRadius: "12px",
          overflow: "hidden",
        }}
      >
        {loading ? (
          <div style={{ padding: "2rem", textAlign: "center", color: "#a1a1aa" }}>
            Memuat pengguna...
          </div>
        ) : (
          <table
            style={{
              width: "100%",
              borderCollapse: "collapse",
              textAlign: "left",
              fontSize: "0.875rem",
            }}
          >
            <thead>
              <tr style={{ backgroundColor: "#18181b", borderBottom: "1px solid #27272a" }}>
                <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600 }}>ID</th>
                <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600 }}>Username</th>
                <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600 }}>Role</th>
              </tr>
            </thead>
            <tbody>
              {users.map((user) => (
                <tr
                  key={user.id}
                  style={{
                    borderBottom: "1px solid #1f1f23",
                    transition: "background 0.15s ease",
                  }}
                >
                  <td style={{ padding: "0.75rem 1rem", color: "#71717a" }}>
                    #{user.id}
                  </td>
                  <td style={{ padding: "0.75rem 1rem", color: "#fafafa", fontWeight: 600 }}>
                    👤 {user.username}
                  </td>
                  <td style={{ padding: "0.75rem 1rem" }}>
                    <span
                      style={{
                        fontSize: "0.75rem",
                        padding: "2px 8px",
                        borderRadius: "10px",
                        backgroundColor:
                          user.role === "admin"
                            ? "rgba(124, 58, 237, 0.15)"
                            : "rgba(37, 99, 235, 0.15)",
                        color: user.role === "admin" ? "#c084fc" : "#60a5fa",
                        border: `1px solid ${
                          user.role === "admin"
                            ? "rgba(124, 58, 237, 0.3)"
                            : "rgba(37, 99, 235, 0.3)"
                        }`,
                        fontWeight: 700,
                        textTransform: "uppercase",
                      }}
                    >
                      {user.role}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
