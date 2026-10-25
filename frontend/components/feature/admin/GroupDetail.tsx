"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { AccessType, Group } from "@/lib/types";
import styles from "./GroupDetail.module.css";

interface GroupDetailProps {
  groupId: string;
}

const ACCESS_OPTIONS: AccessType[] = ["view_only", "view_edit"];

export default function GroupDetail({ groupId }: GroupDetailProps) {
  const [group, setGroup] = useState<Group | null>(null);
  const [updatingDatabaseId, setUpdatingDatabaseId] = useState<number | null>(
    null,
  );
  const [error, setError] = useState("");

  useEffect(() => {
    api.group(groupId)
      .then(setGroup)
      .catch((requestError: unknown) => {
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Unable to load group.",
        );
      });
  }, [groupId]);

  async function updateAccessType(databaseId: number, accessType: AccessType) {
    if (!group) return;

    setError("");
    setUpdatingDatabaseId(databaseId);

    const updatedAccesses = group.databaseAccesses.map((access) =>
      access.databaseId === databaseId ? { ...access, accessType } : access,
    );

    try {
      const updatedGroup = await api.updateGroup(group.id, {
        databaseAccesses: updatedAccesses,
      });
      setGroup(updatedGroup);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Unable to update group access.",
      );
    } finally {
      setUpdatingDatabaseId(null);
    }
  }

  if (!group && error) return <p role="alert">{error}</p>;
  if (!group) return <p>Loading group...</p>;

  return (
    <section className={styles.container}>
      <h1>{group.name}</h1>
      <p>Group ID: {group.id}</p>
      {error && <p role="alert">{error}</p>}
      <table className={styles.dataTable}>
        <thead>
          <tr>
            <th>Database ID</th>
            <th>Access type</th>
          </tr>
        </thead>
        <tbody>
          {group.databaseAccesses.map((access) => (
            <tr key={access.databaseId}>
              <td>{access.databaseId}</td>
              <td>
                <select
                  aria-label={`Access type for database ${access.databaseId}`}
                  className={styles.accessSelect}
                  value={access.accessType}
                  disabled={updatingDatabaseId === access.databaseId}
                  onChange={(event) =>
                    void updateAccessType(
                      access.databaseId,
                      event.target.value as AccessType,
                    )
                  }
                >
                  {ACCESS_OPTIONS.map((accessOption) => (
                    <option key={accessOption} value={accessOption}>
                      {accessOption}
                    </option>
                  ))}
                </select>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
