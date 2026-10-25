"use client";

import { ChangeEvent, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, unwrapPaged } from "@/lib/api";
import type { Database } from "@/lib/types";
import styles from "./DatabaseList.module.css";

const ITEMS_PER_PAGE = 6;

export default function DatabaseList() {
  const router = useRouter();
  const [databases, setDatabases] = useState<Database[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [searchQuery, setSearchQuery] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;

    if (typeof window !== "undefined" && !localStorage.getItem("wbd_user")) {
      router.replace("/login");
      return;
    }

    setLoading(true);

    api
      .databases(currentPage, ITEMS_PER_PAGE)
      .then((body) => {
        if (!active) return;
        setDatabases(unwrapPaged(body));
        setTotalPages(Math.max(1, body.paging?.total_page || 1));
        setError("");
      })
      .catch((requestError: unknown) => {
        if (active) {
          const msg =
            requestError instanceof Error
              ? requestError.message
              : "Gagal memuat database.";
          setError(msg);
          if (
            msg.toLowerCase().includes("unauthorized") ||
            msg.toLowerCase().includes("authentication")
          ) {
            router.replace("/login");
          }
        }
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [currentPage, router]);

  const visibleDatabases = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    if (!query) return databases;

    return databases.filter((database) =>
      `${database.name} ${database.host} ${database.type}`
        .toLowerCase()
        .includes(query),
    );
  }, [databases, searchQuery]);

  function handleSearch(event: ChangeEvent<HTMLInputElement>) {
    setSearchQuery(event.target.value);
    setCurrentPage(1);
  }

  const isAuthError = error.toLowerCase().includes("authentication") || error.toLowerCase().includes("unauthorized");

  return (
    <section className={styles.wrapper}>
      <header className={styles.header}>
        <h1 className={styles.titleBadge}>Daftar Database Anda</h1>
        <p style={{ color: "#a1a1aa", marginTop: "0.5rem", fontSize: "0.95rem" }}>
          Database yang dapat Anda kelola sesuai grup dan tingkat izin yang ditetapkan.
        </p>
      </header>

      {isAuthError ? (
        <div
          style={{
            maxWidth: "500px",
            margin: "2rem auto",
            backgroundColor: "#121215",
            border: "1px solid #27272a",
            borderRadius: "12px",
            padding: "2rem",
            textAlign: "center",
            display: "flex",
            flexDirection: "column",
            gap: "1rem",
            alignItems: "center",
          }}
        >
          <div style={{ fontSize: "2rem" }}>🔒</div>
          <h2 style={{ color: "#fafafa", fontSize: "1.25rem" }}>
            Autentikasi Diperlukan
          </h2>
          <p style={{ color: "#a1a1aa", fontSize: "0.9rem" }}>
            Silakan masuk terlebih dahulu untuk melihat database yang Anda miliki aksesnya.
          </p>
          <Link
            href="/login"
            style={{
              backgroundColor: "#2563eb",
              color: "#fff",
              padding: "8px 20px",
              borderRadius: "6px",
              fontWeight: 600,
              fontSize: "0.9rem",
              textDecoration: "none",
            }}
          >
            Masuk Sekarang
          </Link>
        </div>
      ) : (
        <>
          <div className={styles.controls}>
            <input
              type="search"
              placeholder="🔍 Cari database berdasarkan nama, host, engine..."
              value={searchQuery}
              onChange={handleSearch}
              className={styles.searchInput}
              aria-label="Cari databases"
            />
            <p className={styles.pageInfo}>
              Halaman {currentPage} dari {totalPages} ({databases.length} database)
            </p>
          </div>

          {error && <p className={styles.alertError} role="alert">{error}</p>}

          {loading ? (
            <div style={{ textAlign: "center", padding: "3rem", color: "#a1a1aa" }}>
              Memuat database...
            </div>
          ) : (
            <div className={styles.grid}>
              {visibleDatabases.map((database) => {
                const isViewEdit = database.access_type === "view_edit";
                return (
                  <article key={database.id} className={styles.card}>
                    <div
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "flex-start",
                        gap: "0.5rem",
                      }}
                    >
                      <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                        <span style={{ fontSize: "1.3rem" }}>
                          {database.type === "mysql" ? "🐬" : "🐘"}
                        </span>
                        <h2 className={styles.cardTitle}>{database.name}</h2>
                      </div>
                      {database.access_type && (
                        <span
                          style={{
                            fontSize: "0.75rem",
                            padding: "3px 8px",
                            borderRadius: "12px",
                            backgroundColor: isViewEdit ? "rgba(34, 197, 94, 0.15)" : "rgba(234, 179, 8, 0.15)",
                            color: isViewEdit ? "#86efac" : "#fde047",
                            border: `1px solid ${isViewEdit ? "rgba(34, 197, 94, 0.3)" : "rgba(234, 179, 8, 0.3)"}`,
                            fontWeight: 700,
                          }}
                        >
                          {isViewEdit ? "View & Edit" : "View Only"}
                        </span>
                      )}
                    </div>

                    <div className={styles.cardBody}>
                      <div className={styles.cardText}>
                        <span>Engine:</span>
                        <strong style={{ textTransform: "capitalize" }}>{database.type}</strong>
                      </div>
                      <div className={styles.cardText}>
                        <span>Host & Port:</span>
                        <code>
                          {database.host}:{database.port}
                        </code>
                      </div>
                      <div className={styles.cardText}>
                        <span>User:</span>
                        <span>{database.user}</span>
                      </div>
                    </div>

                    <div className={styles.cardFooter}>
                      <Link
                        href={`/databases/${database.id}`}
                        className={styles.openButton}
                      >
                        Buka Detail & Data →
                      </Link>
                    </div>
                  </article>
                );
              })}

              {!error && visibleDatabases.length === 0 && (
                <div style={{ gridColumn: "1 / -1", textAlign: "center", padding: "3rem", color: "#a1a1aa" }}>
                  <p>Tidak ada database yang ditemukan.</p>
                </div>
              )}
            </div>
          )}

          {totalPages > 1 && (
            <nav className={styles.pagination} aria-label="Navigasi database">
              <button
                className={styles.button}
                onClick={() => setCurrentPage((page) => Math.max(1, page - 1))}
                disabled={currentPage === 1}
              >
                ← Sebelumnya
              </button>
              <span>
                Halaman {currentPage} dari {totalPages}
              </span>
              <button
                className={styles.button}
                onClick={() =>
                  setCurrentPage((page) => Math.min(totalPages, page + 1))
                }
                disabled={currentPage === totalPages}
              >
                Selanjutnya →
              </button>
            </nav>
          )}
        </>
      )}
    </section>
  );
}
