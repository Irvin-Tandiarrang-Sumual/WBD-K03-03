"use client";

import { useEffect, useState, useMemo } from "react";
import Link from "next/link";
import { api, unwrapPaged } from "@/lib/api";
import type { AccessType, Database, Group, User } from "@/lib/types";
import ConfirmModal from "@/components/ui/ConfirmModal";
import ToastContainer from "@/components/ui/Toast";
import { useToast } from "@/lib/hooks/useToast";

interface GroupDetailProps {
  groupId: string;
}

export default function GroupDetail({ groupId }: GroupDetailProps) {
  const [group, setGroup] = useState<Group | null>(null);
  const [databases, setDatabases] = useState<Database[]>([]);
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [groupNameInput, setGroupNameInput] = useState("");
  const [isSavingName, setIsSavingName] = useState(false);
  const [loading, setLoading] = useState(true);

  // Add Database State
  const [selectedDbToAdd, setSelectedDbToAdd] = useState<string>("");
  const [selectedAccessToAdd, setSelectedAccessToAdd] = useState<AccessType>("view_only");
  const [isAddingDb, setIsAddingDb] = useState(false);

  // Add User State
  const [selectedUserToAdd, setSelectedUserToAdd] = useState<string>("");
  const [isAddingUser, setIsAddingUser] = useState(false);

  // Modal State
  const [modalOpen, setModalOpen] = useState(false);
  const [modalLoading, setModalLoading] = useState(false);
  const [modalConfig, setModalConfig] = useState<{
    title: string;
    message: string;
    action: () => Promise<void>;
  } | null>(null);

  // Toast System
  const { toasts, removeToast, toast } = useToast();

  async function loadData() {
    try {
      const [groupRes, dbRes, userRes] = await Promise.all([
        api.group(groupId),
        api.adminDatabases(),
        api.users(),
      ]);
      setGroup(groupRes);
      setGroupNameInput(groupRes.name);
      setDatabases(dbRes.data || []);
      setAllUsers(unwrapPaged(userRes));
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Gagal memuat rincian grup.",
        "Gagal Memuat Data",
      );
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadData();
  }, [groupId]);

  // Databases not yet assigned to this group
  const unassignedDatabases = useMemo(() => {
    if (!group) return [];
    const assignedIds = new Set(group.databaseAccesses.map((a) => a.databaseId));
    return databases.filter((db) => !assignedIds.has(db.id));
  }, [group, databases]);

  function getDatabaseInfo(dbId: number) {
    return databases.find((db) => db.id === dbId);
  }

  // 1. Rename Group
  async function handleSaveGroupName(e: React.FormEvent) {
    e.preventDefault();
    if (!group || !groupNameInput.trim() || groupNameInput.trim() === group.name) return;

    setIsSavingName(true);
    try {
      const updated = await api.updateGroup(group.id, {
        name: groupNameInput.trim(),
      });
      setGroup(updated);
      toast.success(
        `Nama grup berhasil diubah menjadi "${updated.name}"!`,
        "Grup Diperbarui",
      );
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Gagal mengubah nama grup.",
        "Gagal Memperbarui Nama",
      );
    } finally {
      setIsSavingName(false);
    }
  }

  // 2. Toggle or Update Database Access Type
  async function handleToggleAccess(dbId: number, currentAccess: AccessType) {
    if (!group) return;
    const nextAccess: AccessType = currentAccess === "view_only" ? "view_edit" : "view_only";
    const newAccesses = group.databaseAccesses.map((acc) => ({
      databaseId: acc.databaseId,
      accessType: acc.databaseId === dbId ? nextAccess : acc.accessType,
    }));

    try {
      const updated = await api.updateGroup(group.id, {
        databaseAccesses: newAccesses,
      });
      setGroup(updated);
      toast.success(
        `Hak akses database #${dbId} berhasil diubah menjadi ${
          nextAccess === "view_edit" ? "View & Edit" : "View Only"
        }!`,
        "Izin Akses Berhasil Diubah",
      );
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Gagal mengubah hak akses database.",
        "Gagal Memperbarui Izin",
      );
    }
  }

  // 3. Add Database to Group
  async function handleAddDatabase(e: React.FormEvent) {
    e.preventDefault();
    if (!group || !selectedDbToAdd) return;

    setIsAddingDb(true);
    const dbId = Number(selectedDbToAdd);
    const newAccesses = [
      ...group.databaseAccesses.map((a) => ({
        databaseId: a.databaseId,
        accessType: a.accessType,
      })),
      {
        databaseId: dbId,
        accessType: selectedAccessToAdd,
      },
    ];

    try {
      const updated = await api.updateGroup(group.id, {
        databaseAccesses: newAccesses,
      });
      setGroup(updated);
      setSelectedDbToAdd("");
      setSelectedAccessToAdd("view_only");
      const dbInfo = getDatabaseInfo(dbId);
      toast.success(
        `Database "${dbInfo?.name || dbId}" berhasil ditambahkan ke grup dengan akses ${
          selectedAccessToAdd === "view_edit" ? "View & Edit" : "View Only"
        }!`,
        "Database Ditambahkan",
      );
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Gagal menambahkan database ke grup.",
        "Gagal Menambahkan Database",
      );
    } finally {
      setIsAddingDb(false);
    }
  }

  // 4. Remove Database from Group
  function promptRemoveDatabase(dbId: number) {
    if (!group) return;
    if (group.databaseAccesses.length <= 1) {
      toast.error(
        "Grup wajib memiliki minimal 1 database yang dapat diakses! Database ini tidak dapat dihapus.",
        "Aksi Ditolak",
      );
      return;
    }

    const dbInfo = getDatabaseInfo(dbId);
    const dbName = dbInfo ? `${dbInfo.name} (${dbInfo.type})` : `Database #${dbId}`;

    setModalConfig({
      title: "Hapus Hak Akses Database",
      message: `Apakah Anda yakin ingin mencabut izin akses grup "${group.name}" ke ${dbName}? Seluruh user di grup ini tidak akan dapat mengakses database tersebut lagi.`,
      action: async () => {
        const newAccesses = group.databaseAccesses
          .filter((a) => a.databaseId !== dbId)
          .map((a) => ({
            databaseId: a.databaseId,
            accessType: a.accessType,
          }));

        const updated = await api.updateGroup(group.id, {
          databaseAccesses: newAccesses,
        });
        setGroup(updated);
        toast.success(
          `Akses ke database "${dbInfo?.name || dbId}" telah dicabut dari grup.`,
          "Akses Dicabut",
        );
      },
    });
    setModalOpen(true);
  }

  // 5. Add User to Group
  async function handleAddUser(e: React.FormEvent) {
    e.preventDefault();
    if (!group || !selectedUserToAdd) return;

    setIsAddingUser(true);
    const userId = Number(selectedUserToAdd);
    const targetUser = allUsers.find((u) => u.id === userId);

    try {
      await api.addUserToGroup(group.id, userId);
      setSelectedUserToAdd("");
      toast.success(
        `User "${targetUser?.username || userId}" berhasil dimasukkan ke grup "${group.name}"!`,
        "User Ditambahkan ke Grup",
      );
      loadData();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Gagal menambahkan user ke grup.",
        "Gagal Menambahkan User",
      );
    } finally {
      setIsAddingUser(false);
    }
  }

  // 6. Remove User from Group
  function promptRemoveUser(userId: number, username: string) {
    if (!group) return;

    setModalConfig({
      title: "Keluarkan Pengguna dari Grup",
      message: `Apakah Anda yakin ingin mengeluarkan pengguna "${username}" dari grup "${group.name}"? Pengguna akan kehilangan akses ke seluruh database grup ini (kecuali jika tergabung di grup lain).`,
      action: async () => {
        await api.removeUserFromGroup(group.id, userId);
        toast.success(
          `User "${username}" telah dikeluarkan dari grup "${group.name}".`,
          "User Dikeluarkan",
        );
        loadData();
      },
    });
    setModalOpen(true);
  }

  async function handleExecuteModalAction() {
    if (!modalConfig) return;
    setModalLoading(true);
    try {
      await modalConfig.action();
      setModalOpen(false);
      setModalConfig(null);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Terjadi kesalahan saat mengeksekusi tindakan.",
        "Aksi Gagal",
      );
      setModalOpen(false);
    } finally {
      setModalLoading(false);
    }
  }

  if (loading) {
    return (
      <div
        style={{
          maxWidth: "1000px",
          margin: "0 auto",
          padding: "3rem 1.5rem",
          color: "#a1a1aa",
          textAlign: "center",
        }}
      >
        Memuat rincian grup...
      </div>
    );
  }

  if (!group) {
    return (
      <div
        style={{
          maxWidth: "1000px",
          margin: "0 auto",
          padding: "3rem 1.5rem",
          color: "#f87171",
          textAlign: "center",
        }}
      >
        Grup tidak ditemukan.{" "}
        <Link href="/admin/groups" style={{ color: "#3b82f6", textDecoration: "underline" }}>
          Kembali ke Access Groups
        </Link>
      </div>
    );
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
        title={modalConfig?.title || "Konfirmasi Tindakan"}
        message={modalConfig?.message || ""}
        confirmText="Lanjutkan"
        variant="danger"
        isLoading={modalLoading}
        onConfirm={handleExecuteModalAction}
        onCancel={() => setModalOpen(false)}
      />

      {/* Navigation Breadcrumb */}
      <div>
        <Link
          href="/admin/groups"
          style={{
            color: "#a1a1aa",
            fontSize: "0.85rem",
            textDecoration: "none",
            display: "inline-flex",
            alignItems: "center",
            gap: "0.5rem",
            marginBottom: "0.5rem",
            transition: "color 0.15s ease",
          }}
          onMouseEnter={(e) => (e.currentTarget.style.color = "#fafafa")}
          onMouseLeave={(e) => (e.currentTarget.style.color = "#a1a1aa")}
        >
          ← Kembali ke Access Groups
        </Link>
        <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", flexWrap: "wrap" }}>
          <h1 style={{ fontSize: "1.75rem", fontWeight: 800, color: "#fafafa", margin: 0 }}>
            {group.name}
          </h1>
          <span
            style={{
              fontSize: "0.75rem",
              padding: "2px 8px",
              borderRadius: "6px",
              backgroundColor: "rgba(59, 130, 246, 0.15)",
              color: "#60a5fa",
              border: "1px solid rgba(59, 130, 246, 0.3)",
              fontWeight: 700,
            }}
          >
            Group ID: #{group.id}
          </span>
        </div>
      </div>

      {/* Section 1: Edit Group Name */}
      <section
        style={{
          backgroundColor: "#121215",
          border: "1px solid #27272a",
          borderRadius: "12px",
          padding: "1.25rem 1.5rem",
        }}
      >
        <h2 style={{ fontSize: "1.1rem", fontWeight: 700, color: "#fafafa", marginBottom: "0.75rem" }}>
          Pengaturan Grup
        </h2>
        <form
          onSubmit={handleSaveGroupName}
          style={{
            display: "flex",
            gap: "0.75rem",
            alignItems: "center",
            flexWrap: "wrap",
          }}
        >
          <input
            type="text"
            value={groupNameInput}
            onChange={(e) => setGroupNameInput(e.target.value)}
            placeholder="Nama Grup"
            required
            style={{
              flex: 1,
              minWidth: "240px",
              padding: "0.6rem 0.85rem",
              borderRadius: "8px",
              border: "1px solid #27272a",
              backgroundColor: "#18181b",
              color: "#fafafa",
              fontSize: "0.9rem",
              outline: "none",
            }}
          />
          <button
            type="submit"
            disabled={isSavingName || !groupNameInput.trim() || groupNameInput.trim() === group.name}
            style={{
              padding: "0.6rem 1.25rem",
              borderRadius: "8px",
              border: "none",
              backgroundColor:
                groupNameInput.trim() === group.name || !groupNameInput.trim()
                  ? "#27272a"
                  : "#2563eb",
              color:
                groupNameInput.trim() === group.name || !groupNameInput.trim()
                  ? "#71717a"
                  : "#ffffff",
              fontSize: "0.875rem",
              fontWeight: 600,
              cursor:
                groupNameInput.trim() === group.name || !groupNameInput.trim()
                  ? "not-allowed"
                  : "pointer",
              transition: "all 0.15s ease",
            }}
          >
            {isSavingName ? "Menyimpan..." : "Simpan Perubahan Nama"}
          </button>
        </form>
      </section>

      {/* Section 2: Database Access Permissions (Access Control) */}
      <section
        style={{
          backgroundColor: "#121215",
          border: "1px solid #27272a",
          borderRadius: "12px",
          overflow: "hidden",
        }}
      >
        <div
          style={{
            padding: "1.25rem 1.5rem",
            borderBottom: "1px solid #27272a",
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            flexWrap: "wrap",
            gap: "0.75rem",
          }}
        >
          <div>
            <h2 style={{ fontSize: "1.1rem", fontWeight: 700, color: "#fafafa", margin: 0 }}>
              Daftar Database & Hak Akses (Access Control)
            </h2>
            <p style={{ margin: "0.25rem 0 0", fontSize: "0.825rem", color: "#a1a1aa" }}>
              Kelola database yang dapat diakses oleh grup ini beserta tingkat hak izinnya.
            </p>
          </div>
          <span
            style={{
              fontSize: "0.75rem",
              padding: "2px 8px",
              borderRadius: "10px",
              backgroundColor: "rgba(59, 130, 246, 0.1)",
              color: "#60a5fa",
              border: "1px solid rgba(59, 130, 246, 0.2)",
            }}
          >
            {group.databaseAccesses.length} Database Terhubung
          </span>
        </div>

        {/* Database List Table */}
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
                <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600 }}>Database</th>
                <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600 }}>Host & Port</th>
                <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600 }}>Tipe Database</th>
                <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600 }}>Izin Akses</th>
                <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600, textAlign: "right" }}>Aksi</th>
              </tr>
            </thead>
            <tbody>
              {group.databaseAccesses.map((access) => {
                const dbInfo = getDatabaseInfo(access.databaseId);
                const isViewEdit = access.accessType === "view_edit";

                return (
                  <tr
                    key={access.databaseId}
                    style={{
                      borderBottom: "1px solid #1f1f23",
                      transition: "background 0.15s ease",
                    }}
                  >
                    <td style={{ padding: "0.85rem 1rem", color: "#fafafa", fontWeight: 600 }}>
                      <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                        <span style={{ fontSize: "1.1rem" }}>🗄️</span>
                        <span>{dbInfo ? dbInfo.name : `Database #${access.databaseId}`}</span>
                      </div>
                    </td>
                    <td style={{ padding: "0.85rem 1rem", color: "#a1a1aa", fontFamily: "monospace", fontSize: "0.8rem" }}>
                      {dbInfo ? `${dbInfo.host}:${dbInfo.port}` : "—"}
                    </td>
                    <td style={{ padding: "0.85rem 1rem" }}>
                      {dbInfo ? (
                        <span
                          style={{
                            fontSize: "0.75rem",
                            padding: "2px 8px",
                            borderRadius: "10px",
                            backgroundColor:
                              dbInfo.type === "postgresql"
                                ? "rgba(59, 130, 246, 0.15)"
                                : "rgba(245, 158, 11, 0.15)",
                            color: dbInfo.type === "postgresql" ? "#60a5fa" : "#fbbf24",
                            border: `1px solid ${
                              dbInfo.type === "postgresql"
                                ? "rgba(59, 130, 246, 0.3)"
                                : "rgba(245, 158, 11, 0.3)"
                            }`,
                            fontWeight: 700,
                            textTransform: "uppercase",
                          }}
                        >
                          {dbInfo.type}
                        </span>
                      ) : (
                        "—"
                      )}
                    </td>
                    <td style={{ padding: "0.85rem 1rem" }}>
                      <span
                        style={{
                          fontSize: "0.75rem",
                          padding: "3px 8px",
                          borderRadius: "6px",
                          fontWeight: 700,
                          backgroundColor: isViewEdit
                            ? "rgba(16, 185, 129, 0.15)"
                            : "rgba(59, 130, 246, 0.15)",
                          color: isViewEdit ? "#34d399" : "#60a5fa",
                          border: `1px solid ${
                            isViewEdit
                              ? "rgba(16, 185, 129, 0.3)"
                              : "rgba(59, 130, 246, 0.3)"
                          }`,
                          display: "inline-block",
                        }}
                      >
                        {isViewEdit ? "✓ VIEW & EDIT" : "👁 VIEW ONLY"}
                      </span>
                    </td>
                    <td style={{ padding: "0.85rem 1rem", textAlign: "right" }}>
                      <div style={{ display: "inline-flex", gap: "0.5rem" }}>
                        <button
                          type="button"
                          onClick={() => handleToggleAccess(access.databaseId, access.accessType)}
                          title={`Ubah izin menjadi ${isViewEdit ? "View Only" : "View & Edit"}`}
                          style={{
                            padding: "0.4rem 0.75rem",
                            borderRadius: "6px",
                            border: "1px solid #3f3f46",
                            backgroundColor: "#18181b",
                            color: isViewEdit ? "#60a5fa" : "#34d399",
                            fontSize: "0.75rem",
                            fontWeight: 600,
                            cursor: "pointer",
                            transition: "all 0.15s ease",
                          }}
                        >
                          ⇄ Ubah ke {isViewEdit ? "View Only" : "View & Edit"}
                        </button>
                        <button
                          type="button"
                          onClick={() => promptRemoveDatabase(access.databaseId)}
                          title="Hapus Database dari Grup"
                          style={{
                            padding: "0.4rem 0.65rem",
                            borderRadius: "6px",
                            border: "1px solid rgba(239, 68, 68, 0.3)",
                            backgroundColor: "rgba(239, 68, 68, 0.1)",
                            color: "#f87171",
                            fontSize: "0.75rem",
                            fontWeight: 600,
                            cursor: "pointer",
                            transition: "all 0.15s ease",
                          }}
                        >
                          Cabut Akses
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>

        {/* Add Database to Group Form */}
        <div
          style={{
            padding: "1.25rem 1.5rem",
            backgroundColor: "#18181b",
            borderTop: "1px solid #27272a",
          }}
        >
          <h3 style={{ fontSize: "0.95rem", fontWeight: 700, color: "#fafafa", marginBottom: "0.75rem" }}>
            + Hubungkan Database Baru ke Grup Ini
          </h3>
          {unassignedDatabases.length === 0 ? (
            <p style={{ margin: 0, fontSize: "0.85rem", color: "#71717a" }}>
              Seluruh database yang tersedia di server telah dihubungkan ke grup ini.
            </p>
          ) : (
            <form
              onSubmit={handleAddDatabase}
              style={{
                display: "flex",
                gap: "0.75rem",
                alignItems: "center",
                flexWrap: "wrap",
              }}
            >
              <select
                value={selectedDbToAdd}
                onChange={(e) => setSelectedDbToAdd(e.target.value)}
                required
                style={{
                  flex: 2,
                  minWidth: "220px",
                  padding: "0.55rem 0.75rem",
                  borderRadius: "8px",
                  border: "1px solid #27272a",
                  backgroundColor: "#121215",
                  color: "#fafafa",
                  fontSize: "0.875rem",
                  outline: "none",
                }}
              >
                <option value="">-- Pilih Database Tersedia --</option>
                {unassignedDatabases.map((db) => (
                  <option key={db.id} value={db.id}>
                    {db.name} ({db.type.toUpperCase()} @ {db.host}:{db.port})
                  </option>
                ))}
              </select>

              <select
                value={selectedAccessToAdd}
                onChange={(e) => setSelectedAccessToAdd(e.target.value as AccessType)}
                style={{
                  flex: 1,
                  minWidth: "160px",
                  padding: "0.55rem 0.75rem",
                  borderRadius: "8px",
                  border: "1px solid #27272a",
                  backgroundColor: "#121215",
                  color: "#fafafa",
                  fontSize: "0.875rem",
                  outline: "none",
                }}
              >
                <option value="view_only">View Only (Hanya SELECT)</option>
                <option value="view_edit">View & Edit (CRUD Penuh)</option>
              </select>

              <button
                type="submit"
                disabled={isAddingDb || !selectedDbToAdd}
                style={{
                  padding: "0.55rem 1.25rem",
                  borderRadius: "8px",
                  border: "none",
                  backgroundColor: !selectedDbToAdd ? "#27272a" : "#2563eb",
                  color: !selectedDbToAdd ? "#71717a" : "#ffffff",
                  fontSize: "0.875rem",
                  fontWeight: 600,
                  cursor: !selectedDbToAdd ? "not-allowed" : "pointer",
                  transition: "all 0.15s ease",
                }}
              >
                {isAddingDb ? "Menambahkan..." : "+ Tambah ke Grup"}
              </button>
            </form>
          )}
        </div>
      </section>

      {/* Section 3: Group Member Management */}
      <section
        style={{
          backgroundColor: "#121215",
          border: "1px solid #27272a",
          borderRadius: "12px",
          padding: "1.25rem 1.5rem",
          display: "flex",
          flexDirection: "column",
          gap: "1.25rem",
        }}
      >
        <div>
          <h2 style={{ fontSize: "1.1rem", fontWeight: 700, color: "#fafafa", margin: 0 }}>
            Pengguna Anggota Grup
          </h2>
          <p style={{ margin: "0.25rem 0 0", fontSize: "0.825rem", color: "#a1a1aa" }}>
            Tambahkan atau kelola pengguna yang menjadi anggota grup ini agar mewarisi izin akses database di atas.
          </p>
        </div>

        {/* Add User to Group Form */}
        <form
          onSubmit={handleAddUser}
          style={{
            display: "flex",
            gap: "0.75rem",
            alignItems: "center",
            flexWrap: "wrap",
          }}
        >
          <select
            value={selectedUserToAdd}
            onChange={(e) => setSelectedUserToAdd(e.target.value)}
            required
            style={{
              flex: 1,
              minWidth: "240px",
              padding: "0.55rem 0.75rem",
              borderRadius: "8px",
              border: "1px solid #27272a",
              backgroundColor: "#18181b",
              color: "#fafafa",
              fontSize: "0.875rem",
              outline: "none",
            }}
          >
            <option value="">-- Pilih Pengguna untuk Dimasukkan --</option>
            {allUsers.map((user) => (
              <option key={user.id} value={user.id}>
                👤 {user.username} ({user.role.toUpperCase()}) - ID #{user.id}
              </option>
            ))}
          </select>

          <button
            type="submit"
            disabled={isAddingUser || !selectedUserToAdd}
            style={{
              padding: "0.55rem 1.25rem",
              borderRadius: "8px",
              border: "none",
              backgroundColor: !selectedUserToAdd ? "#27272a" : "#2563eb",
              color: !selectedUserToAdd ? "#71717a" : "#ffffff",
              fontSize: "0.875rem",
              fontWeight: 600,
              cursor: !selectedUserToAdd ? "not-allowed" : "pointer",
              transition: "all 0.15s ease",
            }}
          >
            {isAddingUser ? "Memproses..." : "+ Tambahkan User ke Grup"}
          </button>
        </form>

        {/* Quick hint for user management */}
        <div
          style={{
            backgroundColor: "#18181b",
            borderRadius: "8px",
            border: "1px solid #27272a",
            padding: "0.85rem 1rem",
            fontSize: "0.825rem",
            color: "#a1a1aa",
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            flexWrap: "wrap",
            gap: "0.5rem",
          }}
        >
          <span>
            💡 Pengguna yang dimasukkan ke grup ini akan langsung mendapatkan hak akses database sesuai konfigurasi di atas.
          </span>
          <Link
            href="/admin/users"
            style={{
              color: "#60a5fa",
              textDecoration: "none",
              fontWeight: 600,
            }}
          >
            Buka User Management →
          </Link>
        </div>
      </section>
    </div>
  );
}
