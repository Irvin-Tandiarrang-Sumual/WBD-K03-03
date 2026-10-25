"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { api, unwrapPaged } from "@/lib/api";
import {
  clearStoredCurrentUser,
  getStoredCurrentUser,
} from "@/lib/api/auth";
import type { Group, User, UserRole } from "@/lib/types";
import styles from "./UsersPage.module.css";

const ROLE_OPTIONS: UserRole[] = ["user", "admin"];

export default function UsersPage() {
  const router = useRouter();
  const [users, setUsers] = useState<User[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [newUserRole, setNewUserRole] = useState<UserRole>("user");
  const [selectedGroupIds, setSelectedGroupIds] = useState<number[]>([]);
  const [creating, setCreating] = useState(false);
  const [updatingUserId, setUpdatingUserId] = useState<number | null>(null);
  const [error, setError] = useState("");

  function load() {
    Promise.all([api.users(), api.groups()])
      .then(([usersBody, groupsBody]) => {
        setUsers(unwrapPaged(usersBody));
        setGroups(unwrapPaged(groupsBody));
      })
      .catch((requestError: unknown) => {
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Unable to load users.",
        );
      });
  }

  useEffect(load, []);

  const adminCount = useMemo(
    () => users.filter((user) => user.role === "admin").length,
    [users],
  );

  function toggleGroupSelection(groupId: number) {
    setSelectedGroupIds((current) =>
      current.includes(groupId)
        ? current.filter((id) => id !== groupId)
        : [...current, groupId],
    );
  }

  async function createUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!username.trim() || !password) return;

    if (selectedGroupIds.length === 0) {
      setError("Please select at least one group for the new user.");
      return;
    }

    setError("");
    setCreating(true);

    try {
      let createdUser = await api.createUser({
        username: username.trim(),
        password,
        groupIds: selectedGroupIds,
      });

      if (newUserRole === "admin") {
        createdUser = await api.setUserRole(createdUser.id, "admin");
      }

      setUsername("");
      setPassword("");
      setNewUserRole("user");
      setSelectedGroupIds([]);
      setUsers((current) => [...current, createdUser]);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to create user.",
      );
    } finally {
      setCreating(false);
    }
  }

  async function updateUserRole(userId: number, role: UserRole) {
    const targetUser = users.find((user) => user.id === userId);
    if (targetUser?.role === "admin" && role !== "admin" && adminCount <= 1) {
      setError("At least one admin is required.");
      return;
    }

    setError("");
    setUpdatingUserId(userId);

    try {
      const updatedUser = await api.setUserRole(userId, role);
      const currentUser = getStoredCurrentUser();

      if (currentUser && currentUser.id === userId) {
        clearStoredCurrentUser();
        router.push("/login");
        router.refresh();
        return;
      }

      try {
        await api.users(1, 1);
      } catch {
        clearStoredCurrentUser();
        router.push("/login");
        router.refresh();
        return;
      }

      setUsers((current) =>
        current.map((user) => (user.id === userId ? updatedUser : user)),
      );
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to update user role.",
      );
    } finally {
      setUpdatingUserId(null);
    }
  }

  return (
    <section className={styles.container}>
      <h1>Users</h1>
      {error && <p role="alert">{error}</p>}
      <table className={styles.dataTable}>
        <thead>
          <tr>
            <th>ID</th>
            <th>Username</th>
            <th>Role</th>
          </tr>
        </thead>
        <tbody>
          {users.map((user) => {
            const isLastAdmin = user.role === "admin" && adminCount <= 1;

            return (
              <tr key={user.id}>
                <td>{user.id}</td>
                <td>{user.username}</td>
                <td>
                  <select
                    aria-label={`Role for ${user.username}`}
                    className={styles.roleSelect}
                    value={user.role}
                    disabled={updatingUserId === user.id || isLastAdmin}
                    title={
                      isLastAdmin
                        ? "Minimal harus ada 1 admin"
                        : undefined
                    }
                    onChange={(event) =>
                      void updateUserRole(
                        user.id,
                        event.target.value as UserRole,
                      )
                    }
                  >
                    {ROLE_OPTIONS.map((roleOption) => (
                      <option key={roleOption} value={roleOption}>
                        {roleOption}
                      </option>
                    ))}
                  </select>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>

      <form onSubmit={createUser} className={styles.createForm}>
        <h2>Create User</h2>
        <input
          required
          value={username}
          onChange={(event) => setUsername(event.target.value)}
          placeholder="Username"
        />
        <input
          required
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          placeholder="Password"
        />
        <select
          aria-label="New user role"
          value={newUserRole}
          onChange={(event) => setNewUserRole(event.target.value as UserRole)}
        >
          {ROLE_OPTIONS.map((roleOption) => (
            <option key={roleOption} value={roleOption}>
              Role: {roleOption}
            </option>
          ))}
        </select>

        <fieldset className={styles.groupFieldset}>
          <legend>Assign Groups (minimal 1)</legend>
          <div className={styles.checkGrid}>
            {groups.map((group) => (
              <label key={group.id} className={styles.checkLabel}>
                <input
                  type="checkbox"
                  checked={selectedGroupIds.includes(group.id)}
                  onChange={() => toggleGroupSelection(group.id)}
                />
                {group.name}
              </label>
            ))}
          </div>
        </fieldset>

        <button type="submit" disabled={creating}>
          {creating ? "Creating..." : "Create user"}
        </button>
      </form>
    </section>
  );
}
