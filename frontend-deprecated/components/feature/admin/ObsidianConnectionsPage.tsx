"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api, unwrapPaged } from "@/lib/api";
import type { CreateDatabaseInput, Database, DatabaseType } from "@/lib/types";
import ConfirmModal from "@/components/ui/ConfirmModal";
import ToastContainer from "@/components/ui/Toast";
import { useToast } from "@/lib/hooks/useToast";

export interface ServerHost {
  id: string; // key: `${host}:${port}:${type}`
  host: string;
  port: number;
  user: string;
  type: DatabaseType;
  label: string;
}

export default function ConnectionsPage() {
  const [databases, setDatabases] = useState<Database[]>([]);
  const [selectedHostKey, setSelectedHostKey] = useState<string>("");
  const [databaseName, setDatabaseName] = useState("");
  const [loading, setLoading] = useState(false);

  // Modal State
  const [modalOpen, setModalOpen] = useState(false);
  const [modalLoading, setModalLoading] = useState(false);
  const [targetDelete, setTargetDelete] = useState<{ id: number; name: string } | null>(null);

  // Toast System
  const { toasts, removeToast, toast } = useToast();

  function load() {
    api
      .adminDatabases()
      .then((body) => {
        const dbs = unwrapPaged(body);
        setDatabases(dbs);
      })
      .catch((requestError: unknown) => {
        toast.error(
          requestError instanceof Error
            ? requestError.message
            : "Gagal memuat koneksi database.",
          "Gagal Memuat Data",
        );
      });
  }

  useEffect(() => {
    load();
  }, []);

  // Dynamically extract unique server hosts directly from the database records
  const availableHosts: ServerHost[] = useMemo(() => {
    const map = new Map<string, ServerHost>();
    for (const db of databases) {
      const key = `${db.host}:${db.port}:${db.type}`;
      if (!map.has(key)) {
        const typeIcon = db.type === "postgresql" ? "🐘" : "🐬";
        map.set(key, {
          id: key,
          host: db.host,
          port: db.port,
          user: db.user,
          type: db.type as DatabaseType,
          label: `${typeIcon} ${db.type.toUpperCase()} Server (${db.host}:${db.port}) - User: ${db.user}`,
        });
      }
    }
    return Array.from(map.values());
  }, [databases]);

  // Sync selected host key when hosts are loaded
  useEffect(() => {
    if (availableHosts.length > 0) {
      if (!selectedHostKey || !availableHosts.some((h) => h.id === selectedHostKey)) {
        setSelectedHostKey(availableHosts[0].id);
      }
    }
  }, [availableHosts, selectedHostKey]);

  const currentHost = availableHosts.find((h) => h.id === selectedHostKey) || availableHosts[0];

  async function createDatabase(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const cleanName = databaseName.trim();
    if (!cleanName || !currentHost) return;

    setLoading(true);

    const payload: CreateDatabaseInput = {
      host: currentHost.host,
      port: currentHost.port,
      user: currentHost.user,
      password: "", // backend automatically reuses credential from existing database on this host
      type: currentHost.type,
      name: cleanName,
    };

    try {
      await api.createDatabase(payload);
      toast.success(
        `Database "${cleanName}" berhasil dibuat secara dinamis di server ${currentHost.host}:${currentHost.port} dan didaftarkan!`,
        "Database Berhasil Dibuat",
      );
      setDatabaseName("");
      load();
    } catch (requestError) {
      toast.error(
        requestError instanceof Error
          ? requestError.message
          : "Gagal membuat database.",
        "Gagal Membuat Database",
      );
    } finally {
      setLoading(false);
    }
  }

  function promptDeleteDatabase(id: number, name: string) {
    setTargetDelete({ id, name });
    setModalOpen(true);
  }

  async function handleConfirmDelete() {
    if (!targetDelete) return;

    setModalLoading(true);
    try {
      await api.deleteDatabase(targetDelete.id);
      setDatabases((prev) => prev.filter((d) => d.id !== targetDelete.id));
      toast.success(
        `Database "${targetDelete.name}" berhasil dihapus dari sistem.`,
        "Database Dihapus",
      );
      setModalOpen(false);
      setTargetDelete(null);
    } catch (requestError) {
      toast.error(
        requestError instanceof Error
          ? requestError.message
          : "Gagal menghapus database. Pastikan tidak ada grup yang masih mengarah ke database ini.",
        "Gagal Menghapus Database",
      );
      setModalOpen(false);
    } finally {
      setModalLoading(false);
    }
  }

  return (
    <div
      style={{
        maxWidth: "1000px",
        margin: "0 auto",
        padding: "2rem 1.5rem",
        display: "flex",
        flexDirection: "column",
        gap: "2rem",
      }}
    >
      <ToastContainer toasts={toasts} onDismiss={removeToast} />

      <ConfirmModal
        isOpen={modalOpen}
        title="Hapus Database"
        message={`Apakah Anda yakin ingin menghapus database "${targetDelete?.name}"? Tindakan ini hanya diizinkan jika tidak ada access group yang mengarah ke database ini.`}
        confirmText="Ya, Hapus Database"
        variant="danger"
        isLoading={modalLoading}
        onConfirm={handleConfirmDelete}
        onCancel={() => setModalOpen(false)}
      />

      {/* Header */}
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
          <h1 style={{ fontSize: "1.75rem", fontWeight: 800, color: "#fafafa" }}>
            Database Connections
          </h1>
          <p style={{ color: "#a1a1aa", fontSize: "0.9rem", marginTop: "0.25rem" }}>
            Pilih server target dari database dan buat database baru secara dinamis.
          </p>
        </div>
        <Link
          href="/admin"
          style={{
            backgroundColor: "#18181b",
            border: "1px solid #3f3f46",
            color: "#e4e4e7",
            padding: "8px 16px",
            borderRadius: "6px",
            fontSize: "0.875rem",
            fontWeight: 500,
            textDecoration: "none",
          }}
        >
          ← Kembali ke Admin Control
        </Link>
      </div>

      {/* Form Buat Database Baru */}
      <section
        style={{
          backgroundColor: "#121215",
          border: "1px solid #27272a",
          borderRadius: "12px",
          padding: "1.75rem",
          display: "flex",
          flexDirection: "column",
          gap: "1.25rem",
        }}
      >
        <div>
          <h2 style={{ fontSize: "1.2rem", fontWeight: 700, color: "#fafafa" }}>
            + Buat Database Baru
          </h2>
          <p style={{ color: "#a1a1aa", fontSize: "0.875rem", marginTop: "0.25rem" }}>
            ⚡ <strong>Pembuatan Dinamis:</strong> Pilih host server dari daftar server aktif di database di bawah dan masukkan nama database baru. Host dan port otomatis terkunci mengikuti server yang ada di database. Sistem akan otomatis mengeksekusi <code style={{ color: "#86efac" }}>CREATE DATABASE</code> di engine target jika database belum ada.
          </p>
        </div>

        <form
          onSubmit={createDatabase}
          style={{
            display: "flex",
            flexDirection: "column",
            gap: "1.25rem",
          }}
        >
          {/* Pilihan Host Dropdown (Fetched from Database) */}
          <div>
            <label
              htmlFor="host-select"
              style={{
                display: "block",
                fontSize: "0.85rem",
                color: "#e4e4e7",
                fontWeight: 600,
                marginBottom: "0.4rem",
              }}
            >
              Pilih Server / Host (Diambil Langsung dari Database)
            </label>
            <select
              id="host-select"
              value={selectedHostKey}
              onChange={(e) => setSelectedHostKey(e.target.value)}
              disabled={availableHosts.length === 0}
              style={{
                width: "100%",
                padding: "0.75rem 0.85rem",
                backgroundColor: "#09090b",
                border: "1px solid #3b82f6",
                borderRadius: "6px",
                color: "#fafafa",
                fontSize: "0.95rem",
                outline: "none",
                fontWeight: 500,
              }}
            >
              {availableHosts.length === 0 ? (
                <option value="">Memuat server dari database...</option>
              ) : (
                availableHosts.map((opt) => (
                  <option key={opt.id} value={opt.id}>
                    {opt.label} [Dari Database]
                  </option>
                ))
              )}
            </select>
          </div>

          {/* Info Badge Server Terkunci */}
          {currentHost && (
            <div
              style={{
                backgroundColor: "rgba(39, 39, 42, 0.4)",
                border: "1px solid #27272a",
                borderRadius: "8px",
                padding: "0.85rem 1rem",
                display: "grid",
                gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))",
                gap: "0.75rem",
                fontSize: "0.8rem",
              }}
            >
              <div>
                <span style={{ color: "#71717a", display: "block" }}>Host Server:</span>
                <strong style={{ color: "#fafafa" }}>{currentHost.host}</strong>
              </div>
              <div>
                <span style={{ color: "#71717a", display: "block" }}>Port (Terkunci):</span>
                <strong style={{ color: "#60a5fa" }}>{currentHost.port}</strong>
              </div>
              <div>
                <span style={{ color: "#71717a", display: "block" }}>Engine Tipe:</span>
                <strong style={{ color: "#c084fc", textTransform: "uppercase" }}>{currentHost.type}</strong>
              </div>
              <div>
                <span style={{ color: "#71717a", display: "block" }}>User:</span>
                <strong style={{ color: "#fafafa" }}>{currentHost.user}</strong>
              </div>
            </div>
          )}

          {/* Nama Database Baru */}
          <div>
            <label
              htmlFor="db-name"
              style={{
                display: "block",
                fontSize: "0.85rem",
                color: "#60a5fa",
                fontWeight: 700,
                marginBottom: "0.4rem",
              }}
            >
              Nama Database Baru yang Ingin Dibuat (Wajib)
            </label>
            <input
              id="db-name"
              required
              placeholder="e.g. ganteng_db, toko_online, perpustakaan_db"
              value={databaseName}
              onChange={(e) => setDatabaseName(e.target.value)}
              style={{
                width: "100%",
                padding: "0.75rem 0.85rem",
                backgroundColor: "#09090b",
                border: "1px solid rgba(59, 130, 246, 0.5)",
                borderRadius: "6px",
                color: "#fafafa",
                fontSize: "0.95rem",
                outline: "none",
              }}
            />
            <span style={{ fontSize: "0.75rem", color: "#71717a", marginTop: "0.35rem", display: "block" }}>
              Gunakan huruf, angka, dan underscore (_). Database fisik ini akan dibuat otomatis di server yang dipilih ({currentHost ? `${currentHost.host}:${currentHost.port}` : "server"}).
            </span>
          </div>

          <div>
            <button
              type="submit"
              disabled={loading || !databaseName.trim() || !currentHost}
              style={{
                backgroundColor: "#2563eb",
                color: "#ffffff",
                padding: "0.75rem 1.75rem",
                borderRadius: "8px",
                border: "none",
                fontWeight: 700,
                fontSize: "0.95rem",
                cursor: loading || !databaseName.trim() || !currentHost ? "not-allowed" : "pointer",
                opacity: loading || !databaseName.trim() || !currentHost ? 0.6 : 1,
                boxShadow: "0 2px 10px rgba(37, 99, 235, 0.3)",
                transition: "all 0.15s ease",
              }}
            >
              {loading ? "Memproses Pembuatan Database..." : "+ Buat Database Sekarang"}
            </button>
          </div>
        </form>
      </section>

      {/* Daftar Database Aktif */}
      <section
        style={{
          backgroundColor: "#121215",
          border: "1px solid #27272a",
          borderRadius: "12px",
          padding: "1.75rem",
          display: "flex",
          flexDirection: "column",
          gap: "1.25rem",
        }}
      >
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <h2 style={{ fontSize: "1.2rem", fontWeight: 700, color: "#fafafa" }}>
              Daftar Database Aktif
            </h2>
            <p style={{ color: "#a1a1aa", fontSize: "0.85rem", marginTop: "0.25rem" }}>
              Seluruh database yang terdaftar dalam sistem.
            </p>
          </div>
          <span
            style={{
              fontSize: "0.8rem",
              padding: "4px 10px",
              borderRadius: "12px",
              backgroundColor: "rgba(59, 130, 246, 0.1)",
              color: "#60a5fa",
              border: "1px solid rgba(59, 130, 246, 0.2)",
              fontWeight: 600,
            }}
          >
            {databases.length} Database Terdaftar
          </span>
        </div>

        {databases.length === 0 ? (
          <div
            style={{
              padding: "2.5rem 1rem",
              textAlign: "center",
              color: "#71717a",
              border: "1px dashed #27272a",
              borderRadius: "8px",
            }}
          >
            Belum ada koneksi database yang terdaftar.
          </div>
        ) : (
          <div style={{ overflowX: "auto" }}>
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
                  <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600 }}>Nama Database</th>
                  <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600 }}>Tipe Server</th>
                  <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600 }}>Host & Port</th>
                  <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600 }}>User</th>
                  <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600, textAlign: "right" }}>Aksi</th>
                </tr>
              </thead>
              <tbody>
                {databases.map((db) => (
                  <tr
                    key={db.id}
                    style={{
                      borderBottom: "1px solid #1f1f23",
                      transition: "background 0.15s ease",
                    }}
                  >
                    <td style={{ padding: "0.85rem 1rem", color: "#71717a" }}>#{db.id}</td>
                    <td style={{ padding: "0.85rem 1rem", color: "#fafafa", fontWeight: 600 }}>
                      <Link
                        href={`/databases/${db.id}`}
                        style={{
                          color: "#60a5fa",
                          textDecoration: "none",
                          fontWeight: 700,
                        }}
                      >
                        🗄️ {db.name}
                      </Link>
                    </td>
                    <td style={{ padding: "0.85rem 1rem" }}>
                      <span
                        style={{
                          fontSize: "0.75rem",
                          padding: "2px 8px",
                          borderRadius: "10px",
                          backgroundColor:
                            db.type === "postgresql"
                              ? "rgba(59, 130, 246, 0.15)"
                              : "rgba(245, 158, 11, 0.15)",
                          color: db.type === "postgresql" ? "#60a5fa" : "#fbbf24",
                          border: `1px solid ${
                            db.type === "postgresql"
                              ? "rgba(59, 130, 246, 0.3)"
                              : "rgba(245, 158, 11, 0.3)"
                          }`,
                          fontWeight: 700,
                          textTransform: "uppercase",
                        }}
                      >
                        {db.type}
                      </span>
                    </td>
                    <td
                      style={{
                        padding: "0.85rem 1rem",
                        color: "#a1a1aa",
                        fontFamily: "monospace",
                        fontSize: "0.8rem",
                      }}
                    >
                      {db.host}:{db.port}
                    </td>
                    <td style={{ padding: "0.85rem 1rem", color: "#e4e4e7" }}>{db.user}</td>
                    <td style={{ padding: "0.85rem 1rem", textAlign: "right" }}>
                      <button
                        onClick={() => promptDeleteDatabase(db.id, db.name)}
                        style={{
                          backgroundColor: "rgba(239, 68, 68, 0.1)",
                          color: "#f87171",
                          border: "1px solid rgba(239, 68, 68, 0.3)",
                          padding: "6px 12px",
                          borderRadius: "6px",
                          fontSize: "0.75rem",
                          fontWeight: 600,
                          cursor: "pointer",
                          transition: "all 0.15s ease",
                        }}
                      >
                        Hapus Database
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
