"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { Group } from "@/lib/types";
import styles from "./GroupDetail.module.css";

interface GroupDetailProps {
  groupId: string;
}

export default function GroupDetail({ groupId }: GroupDetailProps) {
  const [group, setGroup] = useState<Group | null>(null);
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

  if (error) return <p role="alert">{error}</p>;
  if (!group) return <p>Loading group...</p>;

  return (
    <section className={styles.container}>
      <h1>{group.name}</h1>
      <p>Group ID: {group.id}</p>
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
              <td>{access.accessType}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
