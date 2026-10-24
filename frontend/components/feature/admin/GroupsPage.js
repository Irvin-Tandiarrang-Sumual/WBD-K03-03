"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import styles from "./GroupsPage.module.css";

const initialGroups = [
  { id: "backend", name: "Backend Team", description: "Developers and service administrators", members: 6, databases: 3, color: "blue" },
  { id: "analitik", name: "Analytics Team", description: "Access for reporting needs", members: 4, databases: 2, color: "purple" },
  { id: "qa", name: "Internal QA", description: "Validate changes before release", members: 3, databases: 1, color: "orange" },
];

export default function GroupsPage() {
  const [groups, setGroups] = useState(initialGroups);
  const [query, setQuery] = useState("");
  const [showModal, setShowModal] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  const visibleGroups = useMemo(
    () => groups.filter((group) => `${group.name} ${group.description}`.toLowerCase().includes(query.toLowerCase())),
    [groups, query]
  );

  const createGroup = (event) => {
    event.preventDefault();
    if (!name.trim()) return;
    setGroups((current) => [
      ...current,
      {
        id: name.toLowerCase().replace(/[^a-z0-9]+/g, "-"),
        name: name.trim(),
        description: description.trim() || "No description yet",
        members: 0,
        databases: 0,
        color: "green",
      },
    ]);
    setName("");
    setDescription("");
    setShowModal(false);
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div className={styles.breadcrumb}>
          <strong className = {styles.code}>Groups</strong>
        </div>
      </div>

      <div className={styles.layout}>

        <main className={styles.main}>
          <div className={styles.tabs}>
            <div className={`${styles.tab} ${styles.tabActive}`}>
              Group List ({visibleGroups.length})
            </div>
          </div>

          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "24px" }}>
            <input
              type="text"
              placeholder="Search group name..."
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              className={styles.searchInput}
              style={{ width: "100%", maxWidth: "300px", margin: 0 }}
            />
            <button className={styles.primaryButton} onClick={() => setShowModal(true)}>
              + New Group
            </button>
          </div>

          <div className={styles.grid}>
            {visibleGroups.map((group) => (
              <article className={styles.card} key={group.id}>
                <div className={styles.cardTop}>
                  <span className={`${styles.groupIcon} ${styles[group.color]}`}>
                    {group.name.slice(0, 1)}
                  </span>
                  <button className={styles.moreButton}>•••</button>
                </div>
                <h2>{group.name}</h2>
                <p>{group.description}</p>
                <div className={styles.metrics}>
                  <div>
                    <strong>{group.members}</strong>
                    <span>Members</span>
                  </div>
                  <div>
                    <strong>{group.databases}</strong>
                    <span>Databases</span>
                  </div>
                </div>
                <Link href={`/admin/groups/${group.id}`} className={styles.openLink}>
                  Manage group <span>→</span>
                </Link>
              </article>
            ))}
            
            {visibleGroups.length === 0 && (
              <div className={styles.empty}>No groups found.</div>
            )}
          </div>
        </main>
      </div>

      {showModal && (
        <div className={styles.backdrop} onMouseDown={(event) => event.target === event.currentTarget && setShowModal(false)}>
          <form className={styles.modal} onSubmit={createGroup}>
            <div className={styles.modalHeader}>
              <div>
                <p className={styles.eyebrow}>NEW GROUP</p>
                <h2>Create a new group</h2>
              </div>
              <button type="button" onClick={() => setShowModal(false)} className={styles.close}>
                ×
              </button>
            </div>
            
            <div className={styles.formGrid}>
              <label>
                Group name
                <input required value={name} onChange={(event) => setName(event.target.value)} placeholder="Example: Finance Team" />
              </label>
              <label>
                Description
                <input value={description} onChange={(event) => setDescription(event.target.value)} placeholder="Purpose of this group" />
              </label>
            </div>

            <div className={styles.modalActions}>
              <button type="button" className={styles.secondaryButton} onClick={() => setShowModal(false)}>
                Cancel
              </button>
              <button className={styles.primaryButton}>Create group</button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
}
