"use client";

import Link from "next/link";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { api, unwrapPaged } from "@/lib/api";
import type { AccessType, Database, Group } from "@/lib/types";
import ConfirmModal from "@/components/ui/ConfirmModal";
import ToastContainer from "@/components/ui/Toast";
import { useToast } from "@/lib/hooks/useToast";

export default function GroupsPage() {
  const [groups, setGroups] = useState<Group[]>([]);
  const [databases, setDatabases] = useState<Database[]>([]);
  const [query, setQuery] = useState("");
  const [name, setName] = useState("");
  const [databaseId, setDatabaseId] = useState("");
  const [accessType, setAccessType] = useState<AccessType>("view_only");
  const [creating, setCreating] = useState(false);

  // Modal State
  const [modalOpen, setModalOpen] = useState(false);
  const [modalLoading, setModalLoading] = useState(false);
  const [targetDelete, setTargetDelete] = useState<{ id: number; name: string } | null>(null);

  // Toast System
  const { toasts, removeToast, toast } = useToast();

  function load() {
    Promise.all([api.groups(), api.adminDatabases()])
      .then(([groupResponse, databaseResponse]) => {
        setGroups(unwrapPaged(groupResponse));
        setDatabases(databaseResponse.data || []);
      })
      .catch((requestError: unknown) => {
        toast.error(
          requestError instanceof Error
            ? requestError.message
            : "Gagal memuat daftar grup.",
          "Gagal Memuat Data",
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
    setCreating(true);

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
      toast.success(`Access Group "${name}" berhasil dibuat!`, "Group Dibuat");
      setName("");
      load();
    } catch (requestError) {
      toast.error(
        requestError instanceof Error
          ? requestError.message
          : "Gagal membuat grup.",
        "Gagal Membuat Group",
      );
    } finally {
      setCreating(false);
    }
  }

  function promptDeleteGroup(id: number, name: string) {
    setTargetDelete({ id, name });
    setModalOpen(true);
  }

  async function handleConfirmDelete() {
    if (!targetDelete) return;

    setModalLoading(true);
    try {
      await api.deleteGroup(targetDelete.id);
      setGroups((prev) => prev.filter((g) => g.id !== targetDelete.id));
      toast.success(
        `Access group "${targetDelete.name}" berhasil dihapus.`,
        "Group Dihapus",
      );
      setModalOpen(false);
      setTargetDelete(null);
    } catch (requestError) {
      toast.error(
        requestError instanceof Error
          ? requestError.message
          : "Gagal menghapus group.",
        "Penghapusan Gagal",
      );
      setModalOpen(false);
    } finally {
      setModalLoading(false);
    }
  }

  async function handleToggleAccess(group: Group, dbId: number, currentAccess: AccessType) {
    const nextAccess: AccessType = currentAccess === "view_only" ? "view_edit" : "view_only";
    const newAccesses = group.databaseAccesses.map((acc) => ({
      databaseId: acc.databaseId,
      accessType: acc.databaseId === dbId ? nextAccess : acc.accessType,
    }));

    try {
      await api.updateGroup(group.id, {
        databaseAccesses: newAccesses,
      });
      toast.success(
        `Hak akses grup "${group.name}" berhasil diubah menjadi ${nextAccess === "view_edit" ? "View & Edit" : "View Only"}!`,
        "Izin Akses Berhasil Diubah",
      );
      load();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Gagal mengubah hak akses grup.",
        "Gagal Memperbarui Izin",
      );
    }
  }

  function getDatabaseName(id: number) {
    const found = databases.find((db) => db.id === id);
    return found ? found.name : `Database #${id}`;
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
      {/* Toast Pop-up Notifications */}
      <ToastContainer toasts={toasts} onDismiss={removeToast} />

      {/* Confirmation Modal */}
      <ConfirmModal
        isOpen={modalOpen}
        title="Konfirmasi Hapus Access Group"
        message={
          targetDelete
            ? `Apakah Anda yakin ingin menghapus access group "${targetDelete.name}"?\n\nAnggota yang tergabung dalam grup ini akan kehilangan hak akses ke database yang terkait.`
            : ""
        }
        confirmText="Ya, Hapus Group"
        cancelText="Batal"
        variant="danger"
        isLoading={modalLoading}
        onConfirm={handleConfirmDelete}
        onCancel={() => {
          if (!modalLoading) {
            setModalOpen(false);
            setTargetDelete(null);
          }
        }}
      />

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
            ← Kembali ke Admin Control
          </Link>
          <h1 style={{ fontSize: "1.75rem", fontWeight: 800, color: "#fafafa" }}>
            Access Groups
          </h1>
          <p style={{ color: "#a1a1aa", fontSize: "0.875rem", marginTop: "0.25rem" }}>
            Kelola pembagian grup pengguna dan ubah hak akses database (<code style={{ color: "#fde047" }}>view_only</code> ⇄ <code style={{ color: "#86efac" }}>view_edit</code>).
          </p>
        </div>
      </div>

      {/* Form Buat Group Baru */}
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
            + Buat Access Group Baru
          </h2>
          <p style={{ color: "#a1a1aa", fontSize: "0.85rem", marginTop: "0.2rem" }}>
            Setiap group wajib memiliki minimal satu database yang dapat diakses oleh anggotanya.
          </p>
        </div>

        <form
          onSubmit={createGroup}
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
            gap: "1rem",
          }}
        >
          <div>
            <label
              htmlFor="group-name"
              style={{ display: "block", fontSize: "0.8rem", color: "#a1a1aa", marginBottom: "0.3rem" }}
            >
              Nama Group
            </label>
            <input
              id="group-name"
              required
              placeholder="e.g. Finance, Backend Team, Interns"
              value={name}
              onChange={(e) => setName(e.target.value)}
              style={{
                width: "100%",
                padding: "0.6rem 0.75rem",
                backgroundColor: "#09090b",
                border: "1px solid #3f3f46",
                borderRadius: "6px",
                color: "#fafafa",
                fontSize: "0.875rem",
              }}
            />
          </div>

          <div>
            <label
              htmlFor="group-db"
              style={{ display: "block", fontSize: "0.8rem", color: "#a1a1aa", marginBottom: "0.3rem" }}
            >
              Database Target (Wajib)
            </label>
            <select
              id="group-db"
              required
              value={databaseId}
              onChange={(e) => setDatabaseId(e.target.value)}
              style={{
                width: "100%",
                padding: "0.6rem 0.75rem",
                backgroundColor: "#09090b",
                border: "1px solid #3f3f46",
                borderRadius: "6px",
                color: "#fafafa",
                fontSize: "0.875rem",
              }}
            >
              <option value="">-- Pilih Database Target --</option>
              {databases.map((database) => (
                <option key={database.id} value={database.id}>
                  {database.type === "mysql" ? "🐬 " : "🐘 "}
                  {database.name} ({database.type})
                </option>
              ))}
            </select>
          </div>

          <div>
            <label
              htmlFor="group-access"
              style={{ display: "block", fontSize: "0.8rem", color: "#a1a1aa", marginBottom: "0.3rem" }}
            >
              Tingkat Izin Awal
            </label>
            <select
              id="group-access"
              value={accessType}
              onChange={(e) => setAccessType(e.target.value as AccessType)}
              style={{
                width: "100%",
                padding: "0.6rem 0.75rem",
                backgroundColor: "#09090b",
                border: "1px solid #3f3f46",
                borderRadius: "6px",
                color: "#fafafa",
                fontSize: "0.875rem",
              }}
            >
              <option value="view_only">View only (Hanya Baca / SELECT)</option>
              <option value="view_edit">View & Edit (Bisa Mutasi / Migrasi)</option>
            </select>
          </div>

          <div style={{ gridColumn: "1 / -1", marginTop: "0.5rem" }}>
            <button
              type="submit"
              disabled={creating}
              style={{
                backgroundColor: "#2563eb",
                color: "#fff",
                border: "none",
                padding: "0.65rem 1.5rem",
                borderRadius: "6px",
                fontWeight: 600,
                fontSize: "0.875rem",
                cursor: creating ? "not-allowed" : "pointer",
                opacity: creating ? 0.7 : 1,
              }}
            >
              {creating ? "Membuat Group..." : "+ Buat Group"}
            </button>
          </div>
        </form>
      </section>

      {/* List Groups */}
      <section style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: "0.5rem" }}>
          <h2 style={{ fontSize: "1.2rem", fontWeight: 700, color: "#fafafa" }}>
            Daftar Groups ({groups.length})
          </h2>
          <input
            type="search"
            placeholder="🔍 Cari nama grup..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            style={{
              padding: "0.5rem 0.75rem",
              backgroundColor: "#121215",
              border: "1px solid #27272a",
              borderRadius: "6px",
              color: "#fafafa",
              fontSize: "0.85rem",
              width: "250px",
            }}
          />
        </div>

        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(320px, 1fr))",
            gap: "1.25rem",
          }}
        >
          {visibleGroups.map((group) => (
            <article
              key={group.id}
              style={{
                backgroundColor: "#121215",
                border: "1px solid #27272a",
                borderRadius: "10px",
                padding: "1.25rem",
                display: "flex",
                flexDirection: "column",
                gap: "0.85rem",
              }}
            >
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                <h3 style={{ fontSize: "1.1rem", fontWeight: 700, color: "#fafafa" }}>
                  👥 {group.name}
                </h3>
                <button
                  type="button"
                  onClick={() => promptDeleteGroup(group.id, group.name)}
                  aria-label={`Hapus grup ${group.name}`}
                  style={{
                    backgroundColor: "transparent",
                    border: "none",
                    color: "#71717a",
                    fontSize: "0.85rem",
                    cursor: "pointer",
                    padding: "2px 6px",
                    borderRadius: "4px",
                  }}
                  onMouseOver={(e) => (e.currentTarget.style.color = "#ef4444")}
                  onMouseOut={(e) => (e.currentTarget.style.color = "#71717a")}
                >
                  🗑️
                </button>
              </div>

              {/* Izin Database Terdaftar & Tombol Ubah Akses */}
              <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
                <span style={{ fontSize: "0.75rem", color: "#a1a1aa", textTransform: "uppercase", letterSpacing: "0.5px", fontWeight: 600 }}>
                  Akses Database:
                </span>
                {group.databaseAccesses.map((acc) => {
                  const isEdit = acc.accessType === "view_edit";
                  const dbName = getDatabaseName(acc.databaseId);
                  return (
                    <div
                      key={acc.databaseId}
                      style={{
                        backgroundColor: "#18181b",
                        border: "1px solid #27272a",
                        borderRadius: "6px",
                        padding: "0.5rem 0.75rem",
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "center",
                        gap: "0.5rem",
                      }}
                    >
                      <div style={{ minWidth: 0, flex: 1 }}>
                        <div style={{ color: "#fafafa", fontSize: "0.85rem", fontWeight: 600, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                          🗄️ {dbName}
                        </div>
                        <span
                          style={{
                            fontSize: "0.7rem",
                            padding: "2px 6px",
                            borderRadius: "10px",
                            backgroundColor: isEdit ? "rgba(34, 197, 94, 0.15)" : "rgba(234, 179, 8, 0.15)",
                            color: isEdit ? "#86efac" : "#fde047",
                            border: `1px solid ${isEdit ? "rgba(34, 197, 94, 0.3)" : "rgba(234, 179, 8, 0.3)"}`,
                            fontWeight: 700,
                            display: "inline-block",
                            marginTop: "0.2rem",
                          }}
                        >
                          {isEdit ? "View & Edit" : "View Only"}
                        </span>
                      </div>

                      {/* Tombol Toggle Cepat Ubah Akses */}
                      <button
                        type="button"
                        onClick={() => handleToggleAccess(group, acc.databaseId, acc.accessType)}
                        title={`Klik untuk ubah menjadi ${isEdit ? "View Only" : "View & Edit"}`}
                        style={{
                          backgroundColor: "transparent",
                          border: "1px solid #3f3f46",
                          color: isEdit ? "#fde047" : "#86efac",
                          padding: "4px 8px",
                          borderRadius: "4px",
                          fontSize: "0.75rem",
                          fontWeight: 600,
                          cursor: "pointer",
                          whiteSpace: "nowrap",
                        }}
                        onMouseOver={(e) => {
                          e.currentTarget.style.backgroundColor = "rgba(255, 255, 255, 0.05)";
                        }}
                        onMouseOut={(e) => {
                          e.currentTarget.style.backgroundColor = "transparent";
                        }}
                      >
                        ⇄ Ubah ke {isEdit ? "View Only" : "View & Edit"}
                      </button>
                    </div>
                  );
                })}
              </div>

              <div style={{ marginTop: "auto", paddingTop: "0.5rem", borderTop: "1px solid #27272a" }}>
                <Link
                  href={`/admin/groups/${group.id}`}
                  style={{
                    color: "#60a5fa",
                    fontSize: "0.85rem",
                    fontWeight: 600,
                    textDecoration: "none",
                  }}
                >
                  Kelola Anggota & Detail →
                </Link>
              </div>
            </article>
          ))}

          {visibleGroups.length === 0 && (
            <p style={{ color: "#a1a1aa", fontSize: "0.9rem" }}>
              Tidak ada grup yang ditemukan.
            </p>
          )}
        </div>
      </section>
    </div>
  );
}
