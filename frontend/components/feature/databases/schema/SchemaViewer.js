"use client";

import { useState } from "react";
import { useRouter } from "next/navigation"; 
import styles from "./SchemaViewer.module.css";

const schemas = [
  {
    name: "public",
    tables: ["customer", "flight", "booking", "aircraft"],
  },
  {
    name: "log",
    tables: ["activity_log"],
  },
];

const tableData = {
  customer: {
    columns: [
      { name: "customer_id", type: "integer", nullable: "No", key: "PK" },
      { name: "full_name", type: "varchar(100)", nullable: "No", key: "-" },
      { name: "email", type: "varchar(150)", nullable: "No", key: "-" },
      { name: "phone", type: "varchar(20)", nullable: "Yes", key: "-" },
      { name: "created_at", type: "timestamp", nullable: "No", key: "-" },
    ],
    constraints: [
      { name: "customer_pkey", type: "PRIMARY KEY", definition: "(customer_id)" },
      { name: "customer_email_key", type: "UNIQUE", definition: "(email)" },
    ],
    indexes: [
      { name: "customer_pkey", columns: "customer_id", method: "btree", unique: "Yes" },
      { name: "customer_email_key", columns: "email", method: "btree", unique: "Yes" },
    ],
  },
  flight: {
    columns: [
      { name: "flight_id", type: "integer", nullable: "No", key: "PK" },
      { name: "flight_number", type: "varchar(10)", nullable: "No", key: "-" },
      { name: "origin", type: "varchar(3)", nullable: "No", key: "-" },
      { name: "destination", type: "varchar(3)", nullable: "No", key: "-" },
      { name: "departure", type: "timestamp", nullable: "Yes", key: "-" },
      { name: "arrival", type: "timestamp", nullable: "Yes", key: "-" },
      { name: "aircraft_id", type: "integer", nullable: "No", key: "FK" },
    ],
    constraints: [
      { name: "flight_pkey", type: "PRIMARY KEY", definition: "(flight_id)" },
      {
        name: "flight_aircraft_id_fkey",
        type: "FOREIGN KEY",
        definition: "(aircraft_id) REFERENCES public.aircraft(aircraft_id)",
      },
      { name: "flight_time_check", type: "CHECK", definition: "(arrival > departure)" },
    ],
    indexes: [
      { name: "flight_pkey", columns: "flight_id", method: "btree", unique: "Yes" },
      { name: "idx_flight_departure", columns: "departure", method: "btree", unique: "No" },
      { name: "idx_flight_route", columns: "origin, destination", method: "btree", unique: "No" },
    ],
  },
  booking: {
    columns: [
      { name: "booking_id", type: "integer", nullable: "No", key: "PK" },
      { name: "customer_id", type: "integer", nullable: "No", key: "FK" },
      { name: "flight_id", type: "integer", nullable: "No", key: "FK" },
      { name: "seat_number", type: "varchar(5)", nullable: "Yes", key: "-" },
      { name: "price", type: "numeric(10,2)", nullable: "No", key: "-" },
      { name: "booked_at", type: "timestamp", nullable: "No", key: "-" },
    ],
    constraints: [
      { name: "booking_pkey", type: "PRIMARY KEY", definition: "(booking_id)" },
      {
        name: "booking_customer_id_fkey",
        type: "FOREIGN KEY",
        definition: "(customer_id) REFERENCES public.customer(customer_id)",
      },
      {
        name: "booking_flight_id_fkey",
        type: "FOREIGN KEY",
        definition: "(flight_id) REFERENCES public.flight(flight_id)",
      },
      { name: "booking_flight_seat_key", type: "UNIQUE", definition: "(flight_id, seat_number)" },
    ],
    indexes: [
      { name: "booking_pkey", columns: "booking_id", method: "btree", unique: "Yes" },
      { name: "booking_flight_seat_key", columns: "flight_id, seat_number", method: "btree", unique: "Yes" },
      { name: "idx_booking_customer", columns: "customer_id", method: "btree", unique: "No" },
    ],
  },
  aircraft: {
    columns: [
      { name: "aircraft_id", type: "integer", nullable: "No", key: "PK" },
      { name: "model", type: "varchar(50)", nullable: "No", key: "-" },
      { name: "manufacturer", type: "varchar(50)", nullable: "Yes", key: "-" },
      { name: "capacity", type: "integer", nullable: "No", key: "-" },
      { name: "registration_code", type: "varchar(10)", nullable: "No", key: "-" },
    ],
    constraints: [
      { name: "aircraft_pkey", type: "PRIMARY KEY", definition: "(aircraft_id)" },
      { name: "aircraft_registration_key", type: "UNIQUE", definition: "(registration_code)" },
      { name: "aircraft_capacity_check", type: "CHECK", definition: "(capacity > 0)" },
    ],
    indexes: [
      { name: "aircraft_pkey", columns: "aircraft_id", method: "btree", unique: "Yes" },
      { name: "aircraft_registration_key", columns: "registration_code", method: "btree", unique: "Yes" },
    ],
  },
  activity_log: {
    columns: [
      { name: "log_id", type: "bigint", nullable: "No", key: "PK" },
      { name: "customer_id", type: "integer", nullable: "Yes", key: "FK" },
      { name: "action", type: "varchar(50)", nullable: "No", key: "-" },
      { name: "details", type: "jsonb", nullable: "Yes", key: "-" },
      { name: "logged_at", type: "timestamp", nullable: "No", key: "-" },
    ],
    constraints: [
      { name: "activity_log_pkey", type: "PRIMARY KEY", definition: "(log_id)" },
      {
        name: "activity_log_customer_id_fkey",
        type: "FOREIGN KEY",
        definition: "(customer_id) REFERENCES public.customer(customer_id)",
      },
    ],
    indexes: [
      { name: "activity_log_pkey", columns: "log_id", method: "btree", unique: "Yes" },
      { name: "idx_activity_log_logged_at", columns: "logged_at", method: "btree", unique: "No" },
      { name: "idx_activity_log_details", columns: "details", method: "gin", unique: "No" },
    ],
  },
};

function KeyBadge({ value }) {
  if (value === "PK") return <span className={styles.badgePk}>[PK]</span>;
  if (value === "FK") return <span className={styles.badgeFk}>[FK]</span>;
  return <>{value}</>;
}

function ColumnsTable({ rows }) {
  return (
    <table className={styles.dataTable}>
      <thead>
        <tr>
          <th>Name</th>
          <th>Type</th>
          <th>Nullable</th>
          <th>Key</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((col) => (
          <tr key={col.name}>
            <td>{col.name}</td>
            <td>{col.type}</td>
            <td>{col.nullable}</td>
            <td>
              <KeyBadge value={col.key} />
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function ConstraintsTable({ rows }) {
  return (
    <table className={styles.dataTable}>
      <thead>
        <tr>
          <th>Name</th>
          <th>Type</th>
          <th>Definition</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((c) => (
          <tr key={c.name}>
            <td>{c.name}</td>
            <td>{c.type}</td>
            <td>{c.definition}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function IndexesTable({ rows }) {
  return (
    <table className={styles.dataTable}>
      <thead>
        <tr>
          <th>Name</th>
          <th>Columns</th>
          <th>Method</th>
          <th>Unique</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((idx) => (
          <tr key={idx.name}>
            <td>{idx.name}</td>
            <td>{idx.columns}</td>
            <td>{idx.method}</td>
            <td>{idx.unique}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export default function SchemaViewer({ dbName }) {
  const router = useRouter(); 
  
  const [searchTable, setSearchTable] = useState("");
  const [activeTable, setActiveTable] = useState("flight");
  const [activeTab, setActiveTab] = useState("Columns");

  const current = tableData[activeTable];
  const rows = current
    ? {
        Columns: current.columns,
        Constraints: current.constraints,
        Indexes: current.indexes,
      }[activeTab]
    : null;

  const renderContent = () => {
    if (!rows || rows.length === 0) {
      return (
        <p className={styles.emptyState}>
          Data for {activeTable} {activeTab} is not available.
        </p>
      );
    }
    if (activeTab === "Columns") return <ColumnsTable rows={rows} />;
    if (activeTab === "Constraints") return <ConstraintsTable rows={rows} />;
    return <IndexesTable rows={rows} />;
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <p className={styles.breadcrumb}>
          <strong>{dbName}</strong> 
        </p>
        
        <div className={styles.actions}>
          <button 
            onClick={() => router.push(`/databases/${dbName}/migrations/new?dbName=${dbName}`)}
            className={styles.secondaryButton}
          >
            Migrate
          </button>
          
          <button 
            onClick={() => router.push(`/databases/${dbName}/tables/${activeTable}`)}
            className={styles.primaryButton}
          >
            View Data ({activeTable})
          </button>
        </div>
      </div>

      <div className={styles.layout}>
        <aside className={styles.sidebar}>
          <input
            type="text"
            placeholder="Find table..."
            value={searchTable}
            onChange={(e) => setSearchTable(e.target.value)}
            className={styles.searchInput}
          />

          <div className={styles.schemaList}>
            {schemas.map((schema) => {
              const filteredTables = schema.tables.filter((t) =>
                t.toLowerCase().includes(searchTable.toLowerCase())
              );

              if (filteredTables.length === 0) return null;

              return (
                <div key={schema.name} className={styles.schemaGroup}>
                  <div className={styles.schemaTitle}>▼ {schema.name}</div>
                  <div className={styles.tableList}>
                    {filteredTables.map((table) => (
                      <div
                        key={table}
                        onClick={() => setActiveTable(table)}
                        className={`${styles.tableItem} ${
                          activeTable === table ? styles.tableItemActive : ""
                        }`}
                      >
                        {table}
                      </div>
                    ))}
                  </div>
                </div>
              );
            })}
          </div>
        </aside>

        <main className={styles.main}>
          <div className={styles.tabs}>
            {["Columns", "Constraints", "Indexes"].map((tab) => (
              <div
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`${styles.tab} ${
                  activeTab === tab ? styles.tabActive : ""
                }`}
              >
                {tab}
              </div>
            ))}
          </div>

          <div className={styles.tableContainer}>{renderContent()}</div>
        </main>
      </div>
    </div>
  );
}
