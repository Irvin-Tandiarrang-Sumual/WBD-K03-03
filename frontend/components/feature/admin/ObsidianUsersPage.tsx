"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, unwrapPaged } from "@/lib/api";
import {
  clearStoredCurrentUser,
  getStoredCurrentUser,
} from "@/lib/api/auth";
import type { Group, User, UserRole } from "@/lib/types";
import ConfirmModal from "@/components/ui/ConfirmModal";
import ToastContainer from "@/components/ui/Toast";
import { useToast } from "@/lib/hooks/useToast";

const ROLE_OPTIONS: UserRole[] = ["user", "admin"];

export default function UsersPage() {
  const router = useRouter();
  const [users, setUsers] = useState<User[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  // Create User Form State
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [newUserRole, setNewUserRole] = useState<UserRole>("user");
  const [selectedGroupIds, setSelectedGroupIds] = useState<number[]>([]);
  const [creating, setCreating] = useState(false);

  // Update Role & Delete State
  const [updatingUserId, setUpdatingUserId] = useState<number | null>(null);
  const [targetDelete, setTargetDelete] = useState<User | null>(null);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const { toasts, removeToast, toast } = useToast();

  function loadData() {
    setLoading(true);
    setError("");
    Promise.all([api.users(1, 100), api.groups(1, 100)])
      .then(([usersBody, groupsBody]) => {
        setUsers(unwrapPaged(usersBody));
        const loadedGroups = unwrapPaged(groupsBody);
        setGroups(loadedGroups);
        if (loadedGroups.length === 1) {
          setSelectedGroupIds((current) =>
            current.length === 0 ? [loadedGroups[0].id] : current,
          );
        }
      })
      .catch((requestError: unknown) => {
        const msg =
          requestError instanceof Error
            ? requestError.message
            : "Gagal memuat daftar pengguna.";
        setError(msg);
        toast.error(msg, "Gagal Memuat Data");
      })
      .finally(() => {
        setLoading(false);
      });
  }

  useEffect(() => {
    loadData();
  }, []);

  const adminCount = useMemo(
    () => users.filter((u) => u.role === "admin").length,
    [users],
  );

  const filteredUsers = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return users;
    return users.filter(
      (u) =>
        u.username.toLowerCase().includes(q) ||
        String(u.role).toLowerCase().includes(q) ||
        String(u.id).includes(q),
    );
  }, [users, searchQuery]);

  function toggleGroupSelection(groupId: number) {
    setSelectedGroupIds((current) =>
      current.includes(groupId)
        ? current.filter((id) => id !== groupId)
        : [...current, groupId],
    );
  }

  async function handleCreateUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmedUsername = username.trim();
    if (!trimmedUsername || !password) {
      toast.warning("Username dan password wajib diisi.", "Input Tidak Lengkap");
      return;
    }

    if (selectedGroupIds.length === 0) {
      const msg = "Pilih minimal 1 Access Group untuk pengguna baru.";
      setError(msg);
      toast.warning(msg, "Pilih Access Group");
      return;
    }

    setError("");
    setCreating(true);

    try {
      let createdUser = await api.createUser({
        username: trimmedUsername,
        password,
        groupIds: selectedGroupIds,
      });

      if (newUserRole === "admin") {
        createdUser = await api.setUserRole(createdUser.id, "admin");
      }

      setUsers((current) => [...current, createdUser]);
      setUsername("");
      setPassword("");
      setNewUserRole("user");
      setSelectedGroupIds(groups.length === 1 ? [groups[0].id] : []);
      toast.success(
        `Pengguna "${createdUser.username}" (${createdUser.role.toUpperCase()}) berhasil dibuat!`,
        "User Berhasil Dibuat",
      );
    } catch (requestError) {
      const msg =
        requestError instanceof Error
          ? requestError.message
          : "Gagal membuat pengguna baru.";
      setError(msg);
      toast.error(msg, "Gagal Membuat User");
    } finally {
      setCreating(false);
    }
  }

  async function handleUpdateRole(userId: number, role: UserRole) {
    const targetUser = users.find((u) => u.id === userId);
    if (!targetUser || targetUser.role === role) return;

    if (targetUser.role === "admin" && role !== "admin" && adminCount <= 1) {
      const msg = "Sistem wajib memiliki minimal 1 Admin aktif.";
      setError(msg);
      toast.warning(msg, "Aksi Ditolak");
      return;
    }

    setError("");
    setUpdatingUserId(userId);

    try {
      const updatedUser = await api.setUserRole(userId, role);
      const currentUser = getStoredCurrentUser();

      if (currentUser && currentUser.id === userId && role !== "admin") {
        clearStoredCurrentUser();
        router.push("/login");
        router.refresh();
        return;
      }

      setUsers((current) =>
        current.map((u) => (u.id === userId ? updatedUser : u)),
      );
      toast.success(
        `Role pengguna "${updatedUser.username}" berhasil diubah menjadi ${updatedUser.role.toUpperCase()}!`,
        "Role Diperbarui",
      );
    } catch (requestError) {
      const msg =
        requestError instanceof Error
          ? requestError.message
          : "Gagal mengubah role pengguna.";
      setError(msg);
      toast.error(msg, "Gagal Mengubah Role");
    } finally {
      setUpdatingUserId(null);
    }
  }

  async function handleConfirmDelete() {
    if (!targetDelete) return;
    setDeleteLoading(true);
    setError("");

    try {
      await api.deleteUser(targetDelete.id);
      setUsers((current) => current.filter((u) => u.id !== targetDelete.id));
      toast.success(
        `Pengguna "${targetDelete.username}" berhasil dihapus.`,
        "User Dihapus",
      );
      setDeleteModalOpen(false);
      setTargetDelete(null);
    } catch (requestError) {
      const msg =
        requestError instanceof Error
          ? requestError.message
          : "Gagal menghapus pengguna.";
      setError(msg);
      toast.error(msg, "Gagal Menghapus User");
      setDeleteModalOpen(false);
    } finally {
      setDeleteLoading(false);
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
        gap: "1.75rem",
      }}
    >
      <ToastContainer toasts={toasts} onDismiss={removeToast} />

      <ConfirmModal
        isOpen={deleteModalOpen}
        title="Konfirmasi Hapus Pengguna"
        message={
          targetDelete
            ? `Apakah Anda yakin ingin menghapus pengguna "${targetDelete.username}" (#${targetDelete.id})? Sesi aktif pengguna ini akan segera dicabut.`
            : ""
        }
        confirmText="Ya, Hapus User"
        cancelText="Batal"
        variant="danger"
        isLoading={deleteLoading}
        onConfirm={handleConfirmDelete}
        onCancel={() => {
          if (!deleteLoading) {
            setDeleteModalOpen(false);
            setTargetDelete(null);
          }
        }}
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
          <p style={{ color: "#a1a1aa", fontSize: "0.875rem", marginTop: "0.25rem" }}>
            Buat pengguna baru, tetapkan grup akses awal, dan kelola peran (<code style={{ color: "#60a5fa" }}>user</code> ⇄ <code style={{ color: "#c084fc" }}>admin</code>).
          </p>
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

      {/* Create User Form Card */}
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
            + Buat Pengguna Baru (Create User)
          </h2>
          <p style={{ color: "#a1a1aa", fontSize: "0.85rem", marginTop: "0.2rem" }}>
            Setiap pengguna wajib tergabung ke minimal satu Access Group saat dibuat.
          </p>
        </div>

        <form
          onSubmit={handleCreateUser}
          style={{
            display: "flex",
            flexDirection: "column",
            gap: "1.25rem",
          }}
        >
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
              gap: "1rem",
            }}
          >
            <div>
              <label
                htmlFor="new-username"
                style={{
                  display: "block",
                  fontSize: "0.8rem",
                  color: "#a1a1aa",
                  marginBottom: "0.35rem",
                  fontWeight: 600,
                }}
              >
                Username
              </label>
              <input
                id="new-username"
                required
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="Contoh: budi_santoso"
                style={{
                  width: "100%",
                  backgroundColor: "#09090b",
                  border: "1px solid #27272a",
                  borderRadius: "8px",
                  padding: "0.6rem 0.75rem",
                  color: "#fafafa",
                  fontSize: "0.875rem",
                }}
              />
            </div>

            <div>
              <label
                htmlFor="new-password"
                style={{
                  display: "block",
                  fontSize: "0.8rem",
                  color: "#a1a1aa",
                  marginBottom: "0.35rem",
                  fontWeight: 600,
                }}
              >
                Password
              </label>
              <input
                id="new-password"
                required
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Maksimal 72 karakter"
                style={{
                  width: "100%",
                  backgroundColor: "#09090b",
                  border: "1px solid #27272a",
                  borderRadius: "8px",
                  padding: "0.6rem 0.75rem",
                  color: "#fafafa",
                  fontSize: "0.875rem",
                }}
              />
            </div>

            <div>
              <label
                htmlFor="new-role"
                style={{
                  display: "block",
                  fontSize: "0.8rem",
                  color: "#a1a1aa",
                  marginBottom: "0.35rem",
                  fontWeight: 600,
                }}
              >
                Peran (Role)
              </label>
              <select
                id="new-role"
                aria-label="New user role"
                value={newUserRole}
                onChange={(e) => setNewUserRole(e.target.value as UserRole)}
                style={{
                  width: "100%",
                  backgroundColor: "#09090b",
                  border: "1px solid #27272a",
                  borderRadius: "8px",
                  padding: "0.6rem 0.75rem",
                  color: "#fafafa",
                  fontSize: "0.875rem",
                }}
              >
                <option value="user">USER (Pengguna Standar)</option>
                <option value="admin">ADMIN (Administrator Sistem)</option>
              </select>
            </div>
          </div>

          {/* Group Selection */}
          <div
            style={{
              backgroundColor: "#09090b",
              border: "1px solid #27272a",
              borderRadius: "8px",
              padding: "1rem",
            }}
          >
            <div
              style={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "center",
                flexWrap: "wrap",
                gap: "0.5rem",
                marginBottom: "0.75rem",
              }}
            >
              <span style={{ fontSize: "0.825rem", fontWeight: 600, color: "#e4e4e7" }}>
                Pilih Access Group (Minimal 1) — Terpilih: {selectedGroupIds.length}
              </span>
              {groups.length > 0 && (
                <div style={{ display: "flex", gap: "0.5rem" }}>
                  <button
                    type="button"
                    onClick={() => setSelectedGroupIds(groups.map((g) => g.id))}
                    style={{
                      background: "transparent",
                      border: "1px solid #27272a",
                      color: "#a1a1aa",
                      borderRadius: "6px",
                      padding: "2px 8px",
                      fontSize: "0.75rem",
                      cursor: "pointer",
                    }}
                  >
                    Pilih Semua
                  </button>
                  <button
                    type="button"
                    onClick={() => setSelectedGroupIds([])}
                    style={{
                      background: "transparent",
                      border: "1px solid #27272a",
                      color: "#a1a1aa",
                      borderRadius: "6px",
                      padding: "2px 8px",
                      fontSize: "0.75rem",
                      cursor: "pointer",
                    }}
                  >
                    Reset
                  </button>
                </div>
              )}
            </div>

            {groups.length === 0 ? (
              <div style={{ fontSize: "0.85rem", color: "#fbbf24" }}>
                ⚠️ Belum ada Access Group yang tersedia. Silakan{" "}
                <Link href="/admin/groups" style={{ color: "#60a5fa", textDecoration: "underline" }}>
                  buat Access Group terlebih dahulu
                </Link>{" "}
                sebelum menambahkan pengguna baru.
              </div>
            ) : (
              <div
                style={{
                  display: "grid",
                  gridTemplateColumns: "repeat(auto-fill, minmax(210px, 1fr))",
                  gap: "0.6rem",
                }}
              >
                {groups.map((group) => {
                  const checked = selectedGroupIds.includes(group.id);
                  return (
                    <label
                      key={group.id}
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: "0.6rem",
                        padding: "0.55rem 0.75rem",
                        borderRadius: "8px",
                        border: `1px solid ${checked ? "#3b82f6" : "#27272a"}`,
                        backgroundColor: checked ? "rgba(59, 130, 246, 0.12)" : "#121215",
                        color: checked ? "#fafafa" : "#a1a1aa",
                        fontSize: "0.85rem",
                        fontWeight: checked ? 600 : 400,
                        cursor: "pointer",
                        transition: "all 0.15s ease",
                      }}
                    >
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() => toggleGroupSelection(group.id)}
                        style={{ accentColor: "#3b82f6", cursor: "pointer" }}
                      />
                      <span style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                        👥 {group.name}
                      </span>
                    </label>
                  );
                })}
              </div>
            )}
          </div>

          <div style={{ display: "flex", justifyContent: "flex-end" }}>
            <button
              type="submit"
              disabled={creating || groups.length === 0}
              style={{
                backgroundColor: creating || groups.length === 0 ? "#27272a" : "#2563eb",
                color: "#ffffff",
                border: "none",
                borderRadius: "8px",
                padding: "0.65rem 1.35rem",
                fontSize: "0.875rem",
                fontWeight: 700,
                cursor: creating || groups.length === 0 ? "not-allowed" : "pointer",
              }}
            >
              {creating ? "Membuat Pengguna..." : "+ Buat Pengguna"}
            </button>
          </div>
        </form>
      </section>

      {/* Users Table Card */}
      <div
        style={{
          backgroundColor: "#121215",
          border: "1px solid #27272a",
          borderRadius: "12px",
          overflow: "hidden",
        }}
      >
        <div
          style={{
            padding: "1rem 1.25rem",
            borderBottom: "1px solid #27272a",
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            flexWrap: "wrap",
            gap: "0.75rem",
          }}
        >
          <div style={{ fontWeight: 700, color: "#fafafa", fontSize: "1rem" }}>
            Daftar Pengguna ({filteredUsers.length})
          </div>
          <input
            type="search"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Cari username atau role..."
            style={{
              backgroundColor: "#09090b",
              border: "1px solid #27272a",
              borderRadius: "8px",
              padding: "0.45rem 0.75rem",
              color: "#fafafa",
              fontSize: "0.825rem",
              minWidth: "220px",
            }}
          />
        </div>

        {loading ? (
          <div style={{ padding: "2rem", textAlign: "center", color: "#a1a1aa" }}>
            Memuat pengguna...
          </div>
        ) : filteredUsers.length === 0 ? (
          <div style={{ padding: "2rem", textAlign: "center", color: "#a1a1aa" }}>
            Tidak ada pengguna yang cocok dengan pencarian.
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
                <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600 }}>Ubah Role</th>
                <th style={{ padding: "0.75rem 1rem", color: "#a1a1aa", fontWeight: 600, textAlign: "right" }}>Aksi</th>
              </tr>
            </thead>
            <tbody>
              {filteredUsers.map((user) => {
                const isLastAdmin = user.role === "admin" && adminCount <= 1;

                return (
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
                    <td style={{ padding: "0.75rem 1rem" }}>
                      <select
                        aria-label={`Role for ${user.username}`}
                        value={user.role}
                        disabled={updatingUserId === user.id || isLastAdmin}
                        title={
                          isLastAdmin
                            ? "Minimal harus ada 1 admin aktif"
                            : undefined
                        }
                        onChange={(e) =>
                          void handleUpdateRole(
                            user.id,
                            e.target.value as UserRole,
                          )
                        }
                        style={{
                          backgroundColor: "#09090b",
                          border: "1px solid #27272a",
                          borderRadius: "6px",
                          padding: "0.35rem 0.6rem",
                          color: isLastAdmin ? "#71717a" : "#fafafa",
                          fontSize: "0.8rem",
                          cursor: isLastAdmin ? "not-allowed" : "pointer",
                        }}
                      >
                        {ROLE_OPTIONS.map((roleOption) => (
                          <option key={roleOption} value={roleOption}>
                            {roleOption.toUpperCase()}
                          </option>
                        ))}
                      </select>
                    </td>
                    <td style={{ padding: "0.75rem 1rem", textAlign: "right" }}>
                      <button
                        type="button"
                        disabled={isLastAdmin}
                        title={
                          isLastAdmin
                            ? "Tidak dapat menghapus satu-satunya admin"
                            : "Hapus pengguna"
                        }
                        onClick={() => {
                          setTargetDelete(user);
                          setDeleteModalOpen(true);
                        }}
                        style={{
                          backgroundColor: isLastAdmin
                            ? "rgba(39, 39, 42, 0.4)"
                            : "rgba(239, 68, 68, 0.12)",
                          border: `1px solid ${
                            isLastAdmin
                              ? "#27272a"
                              : "rgba(239, 68, 68, 0.3)"
                          }`,
                          color: isLastAdmin ? "#52525b" : "#fca5a5",
                          borderRadius: "6px",
                          padding: "0.3rem 0.65rem",
                          fontSize: "0.78rem",
                          fontWeight: 600,
                          cursor: isLastAdmin ? "not-allowed" : "pointer",
                        }}
                      >
                        Hapus
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
