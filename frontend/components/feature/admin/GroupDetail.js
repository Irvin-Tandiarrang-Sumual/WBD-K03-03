"use client";

import Link from "next/link";
import { useState } from "react";
import styles from "./GroupDetail.module.css";

const groups = {
  backend: { name: "Backend Team", description: "Developers and service administrators" },
  analitik: { name: "Analytics Team", description: "Access for reporting needs" },
  qa: { name: "Internal QA", description: "Validate changes before release" },
};

const allUsers = [
  { name: "Owen Wijaya", username: "owen" },
  { name: "Sari Kusuma", username: "sari" },
  { name: "Platform Admin", username: "admin" },
  { name: "Bima Pratama", username: "bima" },
];

const databases = ["airportdb", "inventory_prod", "legacy_hr", "analytics_staging"];

export default function GroupDetail({ groupId }) {
  const group = groups[groupId] || groups.backend;
  const [members, setMembers] = useState(groupId === "analitik" ? ["sari", "admin"] : ["owen", "admin"]);
  const [access, setAccess] = useState(
    groupId === "analitik"
      ? { airportdb: "View Only", analytics_staging: "View + Edit" }
      : { airportdb: "View + Edit", inventory_prod: "View Only" }
  );
  const [memberQuery, setMemberQuery] = useState("");
  const [showMemberPicker, setShowMemberPicker] = useState(false);
  const [showAccessModal, setShowAccessModal] = useState(false);
  const [selectedDatabase, setSelectedDatabase] = useState(databases[0]);
  const [selectedLevel, setSelectedLevel] = useState("View Only");

  const availableUsers = allUsers.filter(
    (user) => !members.includes(user.username) && `${user.name} ${user.username}`.toLowerCase().includes(memberQuery.toLowerCase())
  );
  const connectedDatabases = Object.entries(access);

  const addMember = (username) => {
    setMembers((current) => [...current, username]);
    setShowMemberPicker(false);
    setMemberQuery("");
  };

  const giveAccess = (event) => {
    event.preventDefault();
    setAccess((current) => ({ ...current, [selectedDatabase]: selectedLevel }));
    setShowAccessModal(false);
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div className={styles.breadcrumb}>
          <strong className = {styles.code}>
            {group.name}
          </strong>
        </div>
        <div className={`${styles.tab} ${styles.tabActive}`}>
              {group.description}
        </div>

        <div className={styles.actions}>
          <button className={styles.primaryButton} onClick={() => setShowAccessModal(true)}>
            + Grant database access
          </button>
        </div>
      </div>

      <div className={styles.layout}>
        <aside className={styles.sidebar}>
          <div className={styles.sectionHeader}>
            <h3>Members ({members.length})</h3>
            <button className={styles.secondaryButton} onClick={() => setShowMemberPicker((value) => !value)}>
              + Add
            </button>
          </div>

          {showMemberPicker && (
            <div className={styles.picker}>
              <input
                autoFocus
                value={memberQuery}
                onChange={(event) => setMemberQuery(event.target.value)}
                placeholder="Search users..."
                className={styles.searchInput}
              />
              {availableUsers.map((user) => (
                <button key={user.username} onClick={() => addMember(user.username)} className={styles.pickerItem}>
                  <span className={styles.avatar}>{user.name.slice(0, 1)}</span>
                  <span>
                    <strong>{user.name}</strong>
                    <br />
                    <small>@{user.username}</small>
                  </span>
                  <b>+</b>
                </button>
              ))}
              {availableUsers.length === 0 && (
                <p className={styles.noResult}>All users are already members or no users were found.</p>
              )}
            </div>
          )}

          <div className={styles.memberList}>
            {members.map((username) => {
              const user = allUsers.find((item) => item.username === username);
              return (
                <div className={styles.memberRow} key={username}>
                  <span className={styles.avatar}>{user.name.slice(0, 1)}</span>
                  <div>
                    <strong>{user.name}</strong>
                    <br />
                    <small>@{user.username}</small>
                  </div>
                  <button
                    className={styles.revoke}
                    onClick={() => setMembers((current) => current.filter((item) => item !== username))}
                  >
                    Remove
                  </button>
                </div>
              );
            })}
          </div>
        </aside>

        <main className={styles.main} style={{ marginTop: "32px" }}>
          <div className={styles.tabs}>
            <div className={`${styles.tab} ${styles.tabActive}`}>Database Access ({connectedDatabases.length})</div>
          </div>

          <div className={styles.tableContainer}>
            <table className={styles.dataTable} style={{ margin: "0 auto", width: "100%", maxWidth: "800px" }}>
              <thead>
                <tr>
                  <th>Database</th>
                  <th>Access level</th>
                  <th>Connection status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {connectedDatabases.map(([database, level]) => (
                  <tr key={database}>
                    <td>
                      <strong className={styles.databaseName}>◈ {database}</strong>
                    </td>
                    <td>
                      <select
                        className={`${styles.accessSelect} ${level === "View + Edit" ? styles.editLevel : ""}`}
                        value={level}
                        onChange={(event) =>
                          setAccess((current) => ({ ...current, [database]: event.target.value }))
                        }
                      >
                        <option>No access</option>
                        <option>View Only</option>
                        <option>View + Edit</option>
                      </select>
                    </td>
                    <td>
                      <span className={styles.connected}>
                        <i /> Connected
                      </span>
                    </td>
                    <td>
                      <button
                        className={styles.secondaryButton}
                        onClick={() =>
                          setAccess((current) => {
                            const next = { ...current };
                            delete next[database];
                            return next;
                          })
                        }
                      >
                        Revoke access
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </main>
      </div>

      {showAccessModal && (
        <div
          className={styles.backdrop}
          onMouseDown={(event) => event.target === event.currentTarget && setShowAccessModal(false)}
        >
          <form className={styles.modal} onSubmit={giveAccess}>
            <div className={styles.modalHeader}>
              <div>
                <p className={styles.eyebrow}>DATABASE ACCESS</p>
                <h2>Grant database access</h2>
              </div>
              <button type="button" className={styles.close} onClick={() => setShowAccessModal(false)}>
                ×
              </button>
            </div>

            <div className={styles.formGrid}>
              <label>
                Database
                <select value={selectedDatabase} onChange={(event) => setSelectedDatabase(event.target.value)}>
                  {databases.map((database) => (
                    <option key={database} value={database}>
                      {database}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Access level
                <select value={selectedLevel} onChange={(event) => setSelectedLevel(event.target.value)}>
                  <option>View Only</option>
                  <option>View + Edit</option>
                </select>
              </label>
            </div>

            <div className={styles.modalActions}>
              <button type="button" className={styles.secondaryButton} onClick={() => setShowAccessModal(false)}>
                Cancel
              </button>
              <button className={styles.primaryButton}>Save access</button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
}
