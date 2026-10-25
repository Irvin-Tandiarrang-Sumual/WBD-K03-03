"use client";

import { useEffect, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { LoginResponse } from "@/lib/types";

export default function AdminLayout({ children }: { children: ReactNode }) {
  const router = useRouter();
  const [authorized, setAuthorized] = useState<boolean | null>(null);

  useEffect(() => {
    if (typeof window !== "undefined") {
      const cached = localStorage.getItem("wbd_user");
      if (cached) {
        try {
          const user: LoginResponse = JSON.parse(cached);
          if (user.role !== "admin") {
            setAuthorized(false);
            router.replace("/databases");
            return;
          }
        } catch {
          // ignore error
        }
      }
    }

    api
      .me()
      .then((user) => {
        if (user.role === "admin") {
          setAuthorized(true);
        } else {
          setAuthorized(false);
          router.replace("/databases");
        }
      })
      .catch(() => {
        setAuthorized(false);
        router.replace("/login");
      });
  }, [router]);

  if (authorized === null) {
    return (
      <div
        style={{
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          minHeight: "50vh",
        }}
      >
        <p style={{ color: "#a1a1aa" }}>Checking authorization...</p>
      </div>
    );
  }

  if (!authorized) {
    return (
      <div
        style={{
          maxWidth: "600px",
          margin: "4rem auto",
          padding: "2rem",
          textAlign: "center",
          backgroundColor: "#18181b",
          borderRadius: "8px",
          border: "1px solid #27272a",
        }}
      >
        <h2 style={{ color: "#ef4444", marginBottom: "1rem" }}>
          Access Denied
        </h2>
        <p style={{ color: "#a1a1aa", marginBottom: "1.5rem" }}>
          You need Administrator privileges to access this area.
        </p>
        <button
          onClick={() => router.push("/databases")}
          style={{
            backgroundColor: "#2563eb",
            color: "#fff",
            border: "none",
            padding: "8px 16px",
            borderRadius: "4px",
            cursor: "pointer",
          }}
        >
          Go to Databases
        </button>
      </div>
    );
  }

  return <>{children}</>;
}
