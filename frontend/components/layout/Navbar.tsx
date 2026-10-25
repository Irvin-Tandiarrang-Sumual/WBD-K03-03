"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { LoginResponse } from "@/lib/types";

export default function Navbar() {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<LoginResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    function syncFromStorage() {
      if (typeof window !== "undefined") {
        const cached = localStorage.getItem("wbd_user");
        if (cached) {
          try {
            setUser(JSON.parse(cached));
          } catch {
            setUser(null);
          }
        } else {
          setUser(null);
        }
      }
    }

    // 1. Baca langsung dari storage untuk render instan
    syncFromStorage();

    // 2. Verifikasi status sesi sekali ke server
    api
      .me()
      .then((userData) => {
        setUser(userData);
      })
      .catch(() => {
        setUser(null);
      })
      .finally(() => {
        setLoading(false);
      });

    // 3. Listener hanya sinkronisasi dari storage saat ada event login/logout
    window.addEventListener("auth-changed", syncFromStorage);
    return () => window.removeEventListener("auth-changed", syncFromStorage);
  }, []);

  async function handleLogout() {
    await api.logout();
    setUser(null);
    router.push("/login");
  }

  const isActive = (href: string) =>
    href === "/" ? pathname === "/" : pathname.startsWith(href);

  // Arah logo brand bergantung pada role dan status login
  const brandHref = !user
    ? "/login"
    : user.role === "admin"
      ? "/admin"
      : "/databases";

  return (
    <header className="navbar">
      <div className="container">
        <Link href={brandHref} className="brand">
          <span style={{ fontSize: "1.2rem" }}>🗄️</span>
          <span className="brandBadge">Wedebe</span>
        </Link>

        {/* Menu Navigasi Utama: Hanya tampil jika pengguna telah terautentikasi */}
        <nav className="nav-links" aria-label="Main Navigation">
          {!loading && user && (
            <>
              {user.role === "admin" && (
                <Link
                  href="/admin"
                  className="nav-link"
                  aria-current={isActive("/admin") ? "page" : undefined}
                >
                  Admin Control
                </Link>
              )}
              <Link
                href="/databases"
                className="nav-link"
                aria-current={isActive("/databases") ? "page" : undefined}
              >
                Databases
              </Link>
            </>
          )}
        </nav>

        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "0.85rem",
          }}
        >
          {!loading && user ? (
            <>
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "0.5rem",
                  fontSize: "0.875rem",
                  backgroundColor: "rgba(255, 255, 255, 0.05)",
                  padding: "4px 10px",
                  borderRadius: "20px",
                  border: "1px solid rgba(255, 255, 255, 0.1)",
                }}
              >
                <span style={{ color: "#fafafa", fontWeight: 600 }}>
                  {user.username}
                </span>
                <span
                  style={{
                    fontSize: "0.7rem",
                    padding: "2px 7px",
                    borderRadius: "10px",
                    backgroundColor:
                      user.role === "admin" ? "#7c3aed" : "#2563eb",
                    color: "#fff",
                    textTransform: "uppercase",
                    fontWeight: 700,
                    letterSpacing: "0.5px",
                  }}
                >
                  {user.role}
                </span>
              </div>
              <button
                onClick={handleLogout}
                style={{
                  background: "transparent",
                  border: "1px solid #3f3f46",
                  color: "#f87171",
                  padding: "5px 12px",
                  borderRadius: "6px",
                  cursor: "pointer",
                  fontSize: "0.85rem",
                  fontWeight: 500,
                  transition: "all 0.15s ease",
                }}
                onMouseOver={(e) => {
                  e.currentTarget.style.backgroundColor = "rgba(239, 68, 68, 0.1)";
                  e.currentTarget.style.borderColor = "#ef4444";
                }}
                onMouseOut={(e) => {
                  e.currentTarget.style.backgroundColor = "transparent";
                  e.currentTarget.style.borderColor = "#3f3f46";
                }}
              >
                Sign Out
              </button>
            </>
          ) : !loading ? (
            <Link
              href="/login"
              style={{
                background: "#2563eb",
                color: "#fff",
                padding: "6px 16px",
                borderRadius: "6px",
                textDecoration: "none",
                fontSize: "0.875rem",
                fontWeight: 600,
                transition: "background 0.15s ease",
              }}
              onMouseOver={(e) => (e.currentTarget.style.background = "#1d4ed8")}
              onMouseOut={(e) => (e.currentTarget.style.background = "#2563eb")}
            >
              Sign In
            </Link>
          ) : null}
        </div>
      </div>
    </header>
  );
}
