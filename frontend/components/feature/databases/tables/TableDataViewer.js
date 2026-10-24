"use client";

import { useState } from "react";
import { useRouter, useParams } from "next/navigation";
import styles from "./TableDataViewer.module.css";

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

const generateMockData = (tableName) => {
  return Array.from({ length: 125 }, (_, i) => ({
    id: i + 1,
    data_name: `${tableName}_record_${i + 1}`,
    created_at: "2026-10-24T09:00:00.000Z", 
    status: i % 2 === 0 ? "Active" : "Inactive",
  }));
};
const mockTableData = {
  customer: generateMockData("customer"),
  flight: generateMockData("flight"),
  booking: generateMockData("booking"),
  aircraft: generateMockData("aircraft"),
  activity_log: generateMockData("activity_log"),
};

export default function TableDataViewer() {
  const router = useRouter();
  const params = useParams();

  const dbTitle = params?.title || "database";
  const activeTable = params?.table || "customer";

  const [searchTable, setSearchTable] = useState("");
  const [currentPage, setCurrentPage] = useState(1);
  const rowsPerPage = 50;

  const handleTableSelect = (table) => {
    setCurrentPage(1);
    router.push(`/databases/${dbTitle}/tables/${table}`);
  };

  const data = mockTableData[activeTable] || [];
  const totalPages = Math.ceil(data.length / rowsPerPage);
  const paginatedData = data.slice(
    (currentPage - 1) * rowsPerPage,
    currentPage * rowsPerPage
  );

  const columns = data.length > 0 ? Object.keys(data[0]) : [];

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <p className={styles.breadcrumb}>
          <strong>{activeTable}</strong>
        </p>

        <button
          onClick={() => router.push(`/databases/${dbTitle}/schema`)}
          className={styles.primaryButton}
        >
          View Schema
        </button>
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
                        onClick={() => handleTableSelect(table)}
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
          <div className={styles.tableContainer}>
            {paginatedData.length === 0 ? (
              <p className={styles.emptyState}>No data available for {activeTable}.</p>
            ) : (
              <table className={styles.dataTable}>
                <thead>
                  <tr>
                    {columns.map((col) => (
                      <th key={col}>
                        {col}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {paginatedData.map((row, rowIndex) => (
                    <tr key={rowIndex}>
                      {columns.map((col) => (
                        <td key={col}>
                          {row[col]}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>

          {totalPages > 1 && (
            <div className={styles.pagination}>
              <span className={styles.entryCount}>
                Showing {(currentPage - 1) * rowsPerPage + 1} to {Math.min(currentPage * rowsPerPage, data.length)} of {data.length} entries
              </span>
              <div className={styles.pageControls}>
                <button
                  onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                  className={styles.pageButton}
                >
                  Prev
                </button>
                <span className={styles.pageNumber}>
                  Page {currentPage} of {totalPages}
                </span>
                <button
                  onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                  disabled={currentPage === totalPages}
                  className={styles.pageButton}
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
