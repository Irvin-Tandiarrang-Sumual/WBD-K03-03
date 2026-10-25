"use client";

import { useMemo, useState } from "react";
import styles from "./ConnectionsPage.module.css";

const MODAL_TYPES = { CREATE: "create", EDIT: "edit", TEST: "test" };
const CONNECTION_STATUS = { CONNECTED: "Connected", FAILED: "Connection failed" };

/** @typedef {{ id: number, name: string, host: string, port: string, database: string, status: string, user: string }} Connection */
/** @typedef {{ name: string, host: string, port: string, database: string, user: string, password: string }} ConnectionForm */
/** @typedef {{ type: "create" | "edit" | "test", id?: number }} ModalState */

/** @type {Connection[]} */
const INITIAL_CONNECTIONS = [
  { id: 1, name: "Airport Production", host: "db-prod.internal", port: "5432", database: "airportdb", status: CONNECTION_STATUS.CONNECTED, user: "db_admin" },
  { id: 2, name: "Inventory Read Replica", host: "inventory.internal", port: "5432", database: "inventory_prod", status: CONNECTION_STATUS.CONNECTED, user: "reporter" },
  { id: 3, name: "Legacy HR", host: "10.24.8.19", port: "5432", database: "legacy_hr", status: CONNECTION_STATUS.FAILED, user: "hr_service" },
];

/** @returns {ConnectionForm} */
function createEmptyForm() {
  return { name: "", host: "", port: "5432", database: "", user: "", password: "" };
}

/** @param {Connection} connection @returns {ConnectionForm} */
function getFormFromConnection(connection) {
  return { name: connection.name, host: connection.host, port: connection.port, database: connection.database, user: connection.user, password: "" };
}

/** @param {Connection} connection @param {string} query */
function matchesSearch(connection, query) {
  return `${connection.name} ${connection.host} ${connection.database}`.toLowerCase().includes(query.toLowerCase());
}

function ConnectionStats({ connections }) {
  const connectedCount = connections.filter((connection) => connection.status === CONNECTION_STATUS.CONNECTED).length;

  return (
    <div className={styles.stats}>
      <div className={styles.statBox}><span>Total connections</span><strong>{connections.length}</strong></div>
      <div className={styles.statBox}><span>Connected</span><strong className={styles.good}>{connectedCount}</strong></div>
      <div className={styles.statBox}><span>Needs attention</span><strong className={styles.bad}>{connections.length - connectedCount}</strong></div>
    </div>
  );
}

function ConnectionActions({ connection, onEdit, onRemove, onTest }) {
  return (
    <div className={styles.rowActions}>
      <button className={styles.secondaryButton} onClick={() => onTest(connection)}>Test connection</button>
      <button className={styles.secondaryButton} onClick={() => onEdit(connection)}>Edit</button>
      <button className={styles.secondaryButton} onClick={() => onRemove(connection.id)}>Delete</button>
    </div>
  );
}

function ConnectionsTable({ connections, onEdit, onRemove, onTest }) {
  return (
    <table className={styles.dataTable}>
      <thead>
        <tr><th>Connection name</th><th>Host : Port</th><th>Database</th><th>Status</th><th>Actions</th></tr>
      </thead>
      <tbody>
        {connections.length === 0 ? (
          <tr><td colSpan="5" className={styles.emptyState}>No connections found.</td></tr>
        ) : connections.map((connection) => (
          <tr key={connection.id}>
            <td>
              <div className={styles.connectionName}>
                <span className={styles.dbIcon}>◈</span>
                <div><strong>{connection.name}</strong><br /><small>{connection.user}</small></div>
              </div>
            </td>
            <td><code>{connection.host}:{connection.port}</code></td>
            <td><code>{connection.database}</code></td>
            <td>
              <span className={`${styles.status} ${connection.status === CONNECTION_STATUS.CONNECTED ? styles.connected : styles.failed}`}>
                <i /> {connection.status}
              </span>
            </td>
            <td><ConnectionActions connection={connection} onEdit={onEdit} onRemove={onRemove} onTest={onTest} /></td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function ConnectionModal({ modal, form, testResult, onChange, onClose, onSubmit, onTest }) {
  const title = modal.type === MODAL_TYPES.CREATE ? "Add connection" : modal.type === MODAL_TYPES.TEST ? "Test connection" : "Edit connection";
  const fields = [
    ["name", "Display name", "Airport Production", true],
    ["database", "Database name", "airportdb", true],
    ["host", "Host", "db.internal", true],
    ["port", "Port", "", false],
    ["user", "Username", "postgres", true],
    ["password", "Password", "Will not be shown again", false],
  ];

  return (
    <div className={styles.backdrop} onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <form className={styles.modal} onSubmit={onSubmit}>
        <div className={styles.modalHeader}>
          <div><p className={styles.eyebrow}>DATABASE ENDPOINT</p><h2>{title}</h2></div>
          <button type="button" className={styles.close} onClick={onClose}>×</button>
        </div>
        <div className={styles.formGrid}>
          {fields.map(([field, label, placeholder, required]) => (
            <label key={field}>
              {label}
              <input
                required={required}
                type={field === "password" ? "password" : "text"}
                value={form[field]}
                onChange={(event) => onChange(field, event.target.value)}
                placeholder={placeholder}
              />
            </label>
          ))}
        </div>
        {testResult && (
          <div className={`${styles.testResult} ${testResult === "success" ? styles.testSuccess : styles.testError}`}>
            <span>{testResult === "success" ? "✓" : "!"}</span>
            {testResult === "success" ? "Connection successful. Database is accessible." : "Connection failed. Check host, port, and credentials."}
          </div>
        )}
        <div className={styles.modalActions}>
          <button type="button" className={styles.testButton} onClick={onTest}>Test connection</button>
          {modal.type !== MODAL_TYPES.TEST && <>
            <button type="button" className={styles.secondaryButton} onClick={onClose}>Cancel</button>
            <button className={styles.primaryButton}>Save</button>
          </>}
        </div>
      </form>
    </div>
  );
}

export default function ConnectionsPage() {
  const [connections, setConnections] = useState(INITIAL_CONNECTIONS);
  const [query, setQuery] = useState("");
  /** @type {[ModalState | null, Function]} */
  const [modal, setModal] = useState(null);
  const [form, setForm] = useState(createEmptyForm);
  const [testResult, setTestResult] = useState(null);

  const visibleConnections = useMemo(() => connections.filter((connection) => matchesSearch(connection, query)), [connections, query]);
  const closeModal = () => setModal(null);
  const updateFormField = (field, value) => setForm((current) => ({ ...current, [field]: value }));

  const openCreate = () => { setForm(createEmptyForm()); setTestResult(null); setModal({ type: MODAL_TYPES.CREATE }); };
  const openEdit = (connection) => { setForm(getFormFromConnection(connection)); setTestResult(null); setModal({ type: MODAL_TYPES.EDIT, id: connection.id }); };
  const openTest = (connection) => { setForm(getFormFromConnection(connection)); setTestResult(null); setModal({ type: MODAL_TYPES.TEST, id: connection.id }); };
  const testConnection = () => setTestResult(form.host.includes("10.24") ? "error" : "success");

  const saveConnection = (event) => {
    event.preventDefault();
    if (!modal || !form.name || !form.host || !form.database) return;

    if (modal.type === MODAL_TYPES.CREATE) {
      setConnections((current) => [...current, { ...form, id: Date.now(), status: CONNECTION_STATUS.CONNECTED }]);
    } else {
      setConnections((current) => current.map((connection) => connection.id === modal.id ? { ...connection, ...form, status: connection.status } : connection));
    }
    closeModal();
  };

  const removeConnection = (id) => setConnections((current) => current.filter((connection) => connection.id !== id));

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div className={styles.breadcrumb}><strong className={styles.code}>Database Connections</strong></div>
        <div className={styles.actions}><button className={styles.primaryButton} onClick={openCreate}>+ Add connection</button></div>
      </div>
      <div className={styles.layout}>
        <aside className={styles.sidebar}>
          <input type="text" placeholder="Search connections or databases..." value={query} onChange={(event) => setQuery(event.target.value)} className={styles.searchInput} />
          <ConnectionStats connections={connections} />
        </aside>
        <main className={styles.main}>
          <div className={styles.tabs}><div className={`${styles.tab} ${styles.tabActive}`}>Connection list ({visibleConnections.length})</div></div>
          <div className={styles.tableContainer}><ConnectionsTable connections={visibleConnections} onEdit={openEdit} onRemove={removeConnection} onTest={openTest} /></div>
        </main>
      </div>
      {modal && <ConnectionModal modal={modal} form={form} testResult={testResult} onChange={updateFormField} onClose={closeModal} onSubmit={saveConnection} onTest={testConnection} />}
    </div>
  );
}
