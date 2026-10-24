"use client";

import { useState } from "react";
import styles from "./MigrationDiff.module.css";

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

const generateMockLines = (version) => {
  const lines = [];
  
  Object.entries(tableData).forEach(([tableName, data]) => {
    lines.push(`CREATE TABLE ${tableName} (`);
    data.columns.forEach((col, idx) => {
      const isLast = idx === data.columns.length - 1 && data.constraints.length === 0;
      lines.push(`  ${col.name} ${col.type}${col.nullable === "No" ? " NOT NULL" : ""}${isLast ? "" : ","}`);
    });
    data.constraints.forEach((cons, idx) => {
      const isLast = idx === data.constraints.length - 1;
      lines.push(`  CONSTRAINT ${cons.name} ${cons.type} ${cons.definition}${isLast ? "" : ","}`);
    });
    lines.push(`);`);
    lines.push("");
  });

  while (lines.length < 80) {
    lines.push(`-- Empty padding line ${lines.length + 1}`);
  }

  if (version >= 2) {
    lines.push(`INSERT INTO customer (customer_id, full_name, email) VALUES (1, 'John Doe', 'john@example.com');`);
    lines.push(`INSERT INTO aircraft (aircraft_id, model, capacity) VALUES (1, 'Boeing 737', 180);`);
  } else {
    lines.push(`-- No insert operations available in version ${version}`);
    lines.push(`-- Reserved for insert operations`);
  }

  if (version >= 3) {
    lines.push(`UPDATE customer SET phone = '+62812345678' WHERE customer_id = 1;`);
  } else {
    lines.push(`-- No update operations available in version ${version}`);
  }

  if (version >= 4) {
    lines.push(`DELETE FROM booking WHERE customer_id = 1;`);
  } else {
    lines.push(`-- No delete operations available in version ${version}`);
  }

  while (lines.length < 100) {
    lines.push(`-- End of script line ${lines.length + 1}`);
  }

  return lines.slice(0, 100);
};

const INITIAL_VERSIONS = {
  1: generateMockLines(1),
  2: generateMockLines(2),
  3: generateMockLines(3),
  4: generateMockLines(4),
};

export default function MigrationDiff({ databaseName = "database" }) {

  const [viewMode, setViewMode] = useState("Schema Diff");
  const [leftVersion, setLeftVersion] = useState(1);
  const [rightVersion, setRightVersion] = useState(2);
  const [versionsData, setVersionsData] = useState(INITIAL_VERSIONS);
  const [versionsList, setVersionsList] = useState([1, 2, 3, 4]);
  const [migrationQuery, setMigrationQuery] = useState("");

  const linesToRender = [];
  for (let i = 0; i < 100; i++) {
    const leftLine = versionsData[leftVersion]?.[i];
    const rightLine = versionsData[rightVersion]?.[i];
    const isDiff = leftLine !== rightLine;
    if (viewMode === "Full Script" || isDiff) {
      linesToRender.push(i);
    }
  }

  const handleMigrateQuery = () => {
    const addedLines = [];
    for (let i = 0; i < 100; i++) {
      const leftLine = versionsData[leftVersion]?.[i];
      const rightLine = versionsData[rightVersion]?.[i];
      
      if (
        leftLine !== rightLine && 
        rightLine && 
        !rightLine.startsWith("-- Empty padding") && 
        !rightLine.startsWith("-- End of script") &&
        !rightLine.startsWith("-- No")
      ) {
        addedLines.push(rightLine);
      }
    }
    
    setMigrationQuery(addedLines.length > 0 ? addedLines.join("\n") : "-- No actionable query changes found between these versions");
  };

  const handleExecute = () => {
    const nextVersion = Math.max(...versionsList) + 1;
    const newMockLines = generateMockLines(nextVersion);
    
    if (migrationQuery) {
      newMockLines[85] = `-- Executed migration generated in version ${nextVersion}:`;
      const queryLines = migrationQuery.split("\n");
      for (let i = 0; i < queryLines.length && i < 10; i++) {
        newMockLines[86 + i] = queryLines[i];
      }
    }

    setVersionsData(prev => ({ ...prev, [nextVersion]: newMockLines }));
    setVersionsList(prev => [...prev, nextVersion]);
    setRightVersion(nextVersion);
  };

  return (
    <div className={styles.container}>
      <div className={styles.diffHeader}>
        <p className={styles.breadcrumb}>
          <strong>{databaseName}</strong>
        </p>
      </div>
      <div className={styles.topBar}>
        <select
          value={viewMode}
          onChange={(e) => setViewMode(e.target.value)}
          className={styles.dropdown}
        >
          <option value="Schema Diff">Schema Diff</option>
          <option value="Full Script">Full Script</option>
        </select>
      </div>

      <div className={styles.diffWrapper}>
        <div className={styles.diffHeader}>
          <div className={styles.versionSelectWrapper}>
            <select
              value={leftVersion}
              onChange={(e) => setLeftVersion(Number(e.target.value))}
              className={styles.dropdown}
            >
              {versionsList.map((v) => (
                <option key={v} value={v}>
                  Version {v}
                </option>
              ))}
            </select>
          </div>
          <div className={styles.versionSelectWrapper}>
            <select
              value={rightVersion}
              onChange={(e) => setRightVersion(Number(e.target.value))}
              className={styles.dropdown}
            >
              {versionsList.map((v) => (
                <option key={v} value={v}>
                  Version {v}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className={styles.diffContainer}>
          {linesToRender.length === 0 ? (
            <div className={styles.emptyDiff}>No differences found between versions.</div>
          ) : (
            <>
              <div className={styles.editor}>
                {linesToRender.map((i) => {
                  const leftLine = versionsData[leftVersion]?.[i];
                  const rightLine = versionsData[rightVersion]?.[i];
                  const isMatch = leftLine === rightLine;
                  return (
                    <div
                      key={i}
                      className={isMatch ? styles.lineUnchanged : styles.lineRemoved}
                    >
                      <span className={styles.lineNumber}>{i + 1}</span>
                      {leftLine}
                    </div>
                  );
                })}
              </div>

              <div className={styles.editor}>
                {linesToRender.map((i) => {
                  const leftLine = versionsData[leftVersion]?.[i];
                  const rightLine = versionsData[rightVersion]?.[i];
                  const isMatch = leftLine === rightLine;
                  return (
                    <div
                      key={i}
                      className={isMatch ? styles.lineUnchanged : styles.lineAdded}
                    >
                      <span className={styles.lineNumber}>{i + 1}</span>
                      {rightLine}
                    </div>
                  );
                })}
              </div>
            </>
          )}
        </div>
      </div>

      <div style={{ display: 'flex', gap: '10px', marginTop: '1rem', marginBottom: '1rem' }}>
        <button 
          onClick={handleMigrateQuery} 
          style={{ padding: '8px 16px', cursor: 'pointer', backgroundColor: '#0056b3', color: 'white', border: 'none', borderRadius: '4px' }}
        >
          Migrate Query
        </button>
        <button 
          onClick={handleExecute} 
          style={{ padding: '8px 16px', cursor: 'pointer', backgroundColor: '#0056b3', color: 'white', border: 'none', borderRadius: '4px' }}
        >
          Execute
        </button>
      </div>

      <div className={styles.rollbackSection} style={{ marginTop: '1.5rem', paddingBottom: '100px' }}>
        <h3 className={styles.rollbackTitle}>Migration Query Result</h3>
        <textarea
          className={styles.rollbackTextarea}
          value={migrationQuery}
          onChange={(e) => setMigrationQuery(e.target.value)}
          placeholder="Migration query will appear here when you click 'Migrate Query'..."
        />
      </div>
    </div>
  );
}
