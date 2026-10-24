"use client";

import { useMemo, useState } from "react";
import styles from "./UsersPage.module.css";

const initialUsers = [
  { id: 1, username: "owen", name: "Owen Wijaya", groups: ["Tim Backend"], status: "Aktif", admin: false },
  { id: 2, username: "sari", name: "Sari Kusuma", groups: ["Tim Analitik"], status: "Aktif", admin: false },
  { id: 3, username: "admin", name: "Platform Admin", groups: ["Tim Backend", "Tim Analitik"], status: "Aktif", admin: true },
  { id: 4, username: "bima", name: "Bima Pratama", groups: ["Tim Backend"], status: "Nonaktif", admin: false },
];

const groupOptions = ["Tim Backend", "Tim Analitik", "QA Internal"];

export default function UsersPage() {
  const [users, setUsers] = useState(initialUsers);
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("Semua status");
  const [modal, setModal] = useState(null);
  const [form, setForm] = useState({ username: "", name: "", password: "", groups: [], admin: false });

  const filteredUsers = useMemo(() => {
    return users.filter((user) => {
      const matchesQuery = `${user.username} ${user.name} ${user.groups.join(" ")}`.toLowerCase().includes(query.toLowerCase());
      return matchesQuery && (statusFilter === "Semua status" || user.status === statusFilter);
    });
  }, [query, statusFilter, users]);

  const openCreate = () => {
    setForm({ username: "", name: "", password: "", groups: [], admin: false });
    setModal({ type: "create" });
  };

  const openEdit = (user) => {
    setForm({ username: user.username, name: user.name, password: "", groups: user.groups, admin: user.admin });
    setModal({ type: "edit", userId: user.id });
  };

  const saveUser = (event) => {
    event.preventDefault();
    if (!form.username.trim() || !form.name.trim()) return;

    if (modal.type === "create") {
      setUsers((current) => [
        ...current,
        { id: Date.now(), username: form.username.trim(), name: form.name.trim(), groups: form.groups, status: "Aktif", admin: form.admin },
      ]);
    } else {
      setUsers((current) =>
        current.map((user) =>
          user.id === modal.userId
            ? { ...user, username: form.username.trim(), name: form.name.trim(), groups: form.groups, admin: form.admin }
            : user
        )
      );
    }
    setModal(null);
  };

  const toggleStatus = (id) => {
    setUsers((current) =>
      current.map((user) =>
        user.id === id ? { ...user, status: user.status === "Aktif" ? "Nonaktif" : "Aktif" } : user
      )
    );
  };

  const toggleGroup = (group) => {
    setForm((current) => ({
      ...current,
      groups: current.groups.includes(group)
        ? current.groups.filter((item) => item !== group)
        : [...current.groups, group],
    }));
  };

  return (
    <section className={styles.page}>
      <div className={styles.topbar}>
        <div>
          <h1 className = {styles.code}>User Control</h1>
          <p className={styles.subtitle}>Control user and groups.</p>
        </div>
        <button className={styles.primaryButton} onClick={openCreate}>
          + Tambah User
        </button>
      </div>

      <div className={styles.stats}>
        <div className={styles.statCard}>
          <span>Total user</span>
          <strong>{users.length}</strong>
          <small>Semua akun terdaftar</small>
        </div>
        <div className={styles.statCard}>
          <span>Aktif</span>
          <strong>{users.filter((user) => user.status === "Aktif").length}</strong>
          <small className={styles.successText}>Siap digunakan</small>
        </div>
        <div className={styles.statCard}>
          <span>Platform admin</span>
          <strong>{users.filter((user) => user.admin).length}</strong>
          <small>Akses pengelolaan penuh</small>
        </div>
      </div>

      <div className={styles.panel}>
        <div className={styles.toolbar}>
          <div className={styles.searchWrap}>
            <span>⌕</span>
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Cari username, nama, atau group..."
            />
          </div>
          <select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}>
            <option>Semua status</option>
            <option>Aktif</option>
            <option>Nonaktif</option>
          </select>
          <span className={styles.resultCount}>{filteredUsers.length} user</span>
        </div>
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Username</th>
                <th>Nama</th>
                <th>Group</th>
                <th>Status</th>
                <th className={styles.actionHeader}>Aksi</th>
              </tr>
            </thead>
            <tbody>
              {filteredUsers.map((user) => (
                <tr key={user.id}>
                  <td>
                    <div className={styles.userCell}>
                      <span className={styles.avatar}>{user.name.slice(0, 1)}</span>
                      <div>
                        <strong>{user.username}</strong>
                        {user.admin && <span className={styles.adminLabel}>ADMIN</span>}
                      </div>
                    </div>
                  </td>
                  <td>{user.name}</td>
                  <td>
                    <div className={styles.badgeList}>
                      {user.groups.map((group) => (
                        <span className={styles.groupBadge} key={group}>
                          {group}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td>
                    <span className={`${styles.status} ${user.status === "Aktif" ? styles.statusActive : styles.statusInactive}`}>
                      <i /> {user.status}
                    </span>
                  </td>
                  <td>
                    <div className={styles.actions}>
                      <button onClick={() => openEdit(user)}>Edit</button>
                      <button
                        className={user.status === "Aktif" ? styles.dangerAction : styles.activateAction}
                        onClick={() => toggleStatus(user.id)}
                      >
                        {user.status === "Aktif" ? "Nonaktifkan" : "Aktifkan"}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {filteredUsers.length === 0 && (
            <div className={styles.empty}>Tidak ada user yang cocok dengan pencarian.</div>
          )}
        </div>
      </div>

      {modal && (
        <div className={styles.modalBackdrop} onMouseDown={(event) => event.target === event.currentTarget && setModal(null)}>
          <form className={styles.modal} onSubmit={saveUser}>
            <div className={styles.modalHeader}>
              <div>
                <h2>{modal.type === "create" ? "Add user" : "Edit user"}</h2>
              </div>
              <button type="button" className={styles.closeButton} onClick={() => setModal(null)}>
                ×
              </button>
            </div>

            <label>
              Username
              <input
                required
                value={form.username}
                onChange={(event) => setForm({ ...form, username: event.target.value })}
                placeholder="contoh: nama.user"
              />
            </label>
            
            <label>
              Nama lengkap
              <input
                required
                value={form.name}
                onChange={(event) => setForm({ ...form, name: event.target.value })}
                placeholder="Nama user"
              />
            </label>
            
            {modal.type === "create" && (
              <label>
                Password
                <input
                  required
                  type="password"
                  value={form.password}
                  onChange={(event) => setForm({ ...form, password: event.target.value })}
                  placeholder="Minimal 8 karakter"
                />
              </label>
            )}

            <fieldset>
              <legend>Group</legend>
              <div className={styles.checkGrid}>
                {groupOptions.map((group) => (
                  <label className={styles.checkLabel} key={group}>
                    <input
                      type="checkbox"
                      checked={form.groups.includes(group)}
                      onChange={() => toggleGroup(group)}
                    />
                    {group}
                  </label>
                ))}
              </div>
            </fieldset>

            <label 
              className={styles.switchRow} 
              style={{ display: "flex", flexDirection: "row", alignItems: "center", gap: "8px", marginTop: "16px", justifyContent: "flex-start" }}
            >
              <input
                type="checkbox"
                checked={form.admin}
                onChange={(event) => setForm({ ...form, admin: event.target.checked })}
              />
              <span style={{ fontWeight: 500 }}>Platform Admin</span>
            </label>

            <div className={styles.modalActions}>
              <button type="button" className={styles.secondaryButton} onClick={() => setModal(null)}>
                Batal
              </button>
              <button className={styles.primaryButton}>
                {modal.type === "create" ? "Buat user" : "Simpan perubahan"}
              </button>
            </div>
          </form>
        </div>
      )}
    </section>
  );
}
