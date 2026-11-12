"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
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
      const user = await api.login(username, password);
      if (user.role === "admin") {
        router.push("/admin");
      } else {
        router.push("/databases");
      }
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Gagal masuk. Periksa kembali username dan password Anda.",
      );
    } finally {
      setLoading(false);
    }
  }

  function handleQuickFill(u: string, p: string) {
    setUsername(u);
    setPassword(p);
    setError("");
  }

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        alignItems: "center",
        minHeight: "75vh",
        padding: "1.5rem",
      }}
    >
      <div
        style={{
          width: "100%",
          maxWidth: "420px",
          backgroundColor: "#121215",
          border: "1px solid #27272a",
          borderRadius: "12px",
          padding: "2.25rem",
          boxShadow: "0 10px 30px rgba(0, 0, 0, 0.5)",
          display: "flex",
          flexDirection: "column",
          gap: "1.5rem",
        }}
      >
        <div style={{ textAlign: "center" }}>
          <div style={{ fontSize: "2rem", marginBottom: "0.5rem" }}>🔐</div>
          <h1 style={{ fontSize: "1.5rem", fontWeight: 700, color: "#fafafa" }}>
            Masuk ke Wedebe
          </h1>
          <p
            style={{
              color: "#a1a1aa",
              fontSize: "0.875rem",
              marginTop: "0.25rem",
            }}
          >
            Silakan masukkan kredensial akun Anda
          </p>
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

        <form
          onSubmit={handleSubmit}
          style={{ display: "flex", flexDirection: "column", gap: "1rem" }}
        >
          <div>
            <label
              htmlFor="username"
              style={{
                display: "block",
                marginBottom: "0.4rem",
                fontSize: "0.85rem",
                color: "#e4e4e7",
                fontWeight: 500,
              }}
            >
              Username
            </label>
            <input
              id="username"
              type="text"
              required
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="Masukkan username"
              style={{
                width: "100%",
                padding: "0.65rem 0.85rem",
                backgroundColor: "#09090b",
                border: "1px solid #3f3f46",
                borderRadius: "6px",
                color: "#fafafa",
                fontSize: "0.9rem",
                outline: "none",
              }}
            />
          </div>

          <div>
            <label
              htmlFor="password"
              style={{
                display: "block",
                marginBottom: "0.4rem",
                fontSize: "0.85rem",
                color: "#e4e4e7",
                fontWeight: 500,
              }}
            >
              Password
            </label>
            <input
              id="password"
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Masukkan password"
              style={{
                width: "100%",
                padding: "0.65rem 0.85rem",
                backgroundColor: "#09090b",
                border: "1px solid #3f3f46",
                borderRadius: "6px",
                color: "#fafafa",
                fontSize: "0.9rem",
                outline: "none",
              }}
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            style={{
              marginTop: "0.5rem",
              backgroundColor: "#2563eb",
              color: "#fff",
              border: "none",
              padding: "0.75rem",
              borderRadius: "6px",
              fontSize: "0.95rem",
              fontWeight: 600,
              cursor: loading ? "not-allowed" : "pointer",
              opacity: loading ? 0.7 : 1,
              transition: "background 0.15s ease",
            }}
          >
            {loading ? "Sedang Masuk..." : "Sign In"}
          </button>
        </form>

        <div style={{ borderTop: "1px solid #27272a", paddingTop: "1.25rem" }}>
          <p
            style={{
              fontSize: "0.75rem",
              color: "#71717a",
              marginBottom: "0.6rem",
              textTransform: "uppercase",
              letterSpacing: "0.5px",
              fontWeight: 600,
              textAlign: "center",
            }}
          >
            Klik Cepat Akun Demo:
          </p>
          <div style={{ display: "flex", gap: "0.5rem" }}>
            <button
              type="button"
              onClick={() => handleQuickFill("admin", "admin123")}
              style={{
                flex: 1,
                backgroundColor: "rgba(124, 58, 237, 0.15)",
                border: "1px solid rgba(124, 58, 237, 0.4)",
                color: "#c084fc",
                padding: "0.5rem",
                borderRadius: "6px",
                fontSize: "0.8rem",
                fontWeight: 600,
                cursor: "pointer",
              }}
            >
              👑 Admin (admin123)
            </button>
            <button
              type="button"
              onClick={() => handleQuickFill("user", "user123")}
              style={{
                flex: 1,
                backgroundColor: "rgba(37, 99, 235, 0.15)",
                border: "1px solid rgba(37, 99, 235, 0.4)",
                color: "#60a5fa",
                padding: "0.5rem",
                borderRadius: "6px",
                fontSize: "0.8rem",
                fontWeight: 600,
                cursor: "pointer",
              }}
            >
              👤 User (user123)
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
