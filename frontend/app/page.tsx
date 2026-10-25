"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { LoginResponse } from "@/lib/types";

export default function Home() {
  const router = useRouter();

  useEffect(() => {
    // 1. Cek dari localStorage untuk navigasi instan
    if (typeof window !== "undefined") {
      const cached = localStorage.getItem("wbd_user");
      if (cached) {
        try {
          const user: LoginResponse = JSON.parse(cached);
          if (user.role === "admin") {
            router.replace("/admin");
          } else {
            router.replace("/databases");
          }
          return;
        } catch {
          // ignore parsing error
        }
      }
    }

    // 2. Verifikasi dengan API me jika tidak ada di localStorage
    api
      .me()
      .then((user) => {
        if (user.role === "admin") {
          router.replace("/admin");
        } else {
          router.replace("/databases");
        }
      })
      .catch(() => {
        // Belum login, langsung ke login page
        router.replace("/login");
      });
  }, [router]);

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        alignItems: "center",
        minHeight: "70vh",
        gap: "1rem",
        color: "#a1a1aa",
        fontSize: "0.95rem",
      }}
    >
      <div
        style={{
          width: "36px",
          height: "36px",
          border: "3px solid #27272a",
          borderTop: "3px solid #3b82f6",
          borderRadius: "50%",
          animation: "spin 0.8s linear infinite",
        }}
      />
      <style>{`
        @keyframes spin {
          0% { transform: rotate(0deg); }
          100% { transform: rotate(360deg); }
        }
      `}</style>
      <p>Memeriksa sesi dan mengarahkan...</p>
    </div>
  );
}
