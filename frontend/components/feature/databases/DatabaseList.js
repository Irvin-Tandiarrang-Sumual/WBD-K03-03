"use client";

import { useMemo, useState } from "react";
import ButtonLink from "@/components/ui/ButtonLink";
import styles from "./DatabaseList.module.css";

const ITEMS_PER_PAGE = 4;

/** @typedef {{ id: number, title: string, access: string, connection: string, href: string }} Database */

/** @type {Database[]} */
const DATABASES = [
  "apples", "bananas", "cherries", "dates", "eagles",
  "flowers", "grass", "hats", "igloos", "jacks",
  "kangaroos", "leaves", "mangos", "nuts", "ovens",
  "pandas", "queens", "rashes", "trolls", "undeads",
].map((title, index) => ({
  id: index + 1,
  title,
  access: index % 2 === 0 ? "view and edit" : "view only",
  connection: index % 3 === 0 ? "not connected" : "connected",
  href: `/databases/${title}/schema`,
}));

/** @param {Database[]} databases @param {string} query */
function filterDatabases(databases, query) {
  const normalizedQuery = query.trim().toLowerCase();
  return databases.filter((database) => database.title.toLowerCase().includes(normalizedQuery));
}

export default function DatabaseList() {
  const [currentPage, setCurrentPage] = useState(1);
  const [searchQuery, setSearchQuery] = useState("");

  const filteredDatabases = useMemo(
    () => filterDatabases(DATABASES, searchQuery),
    [searchQuery]
  );
  const totalPages = Math.max(1, Math.ceil(filteredDatabases.length / ITEMS_PER_PAGE));
  const startIndex = (currentPage - 1) * ITEMS_PER_PAGE;
  const visibleDatabases = filteredDatabases.slice(startIndex, startIndex + ITEMS_PER_PAGE);

  const handleSearch = (event) => {
    setSearchQuery(event.target.value);
    setCurrentPage(1);
  };

  const goToPreviousPage = () => setCurrentPage((page) => Math.max(1, page - 1));
  const goToNextPage = () => setCurrentPage((page) => Math.min(totalPages, page + 1));

  return (
    <section className={styles.wrapper}>
      <header className={styles.header}>
        <h1><span className={styles.titleBadge}>Databases</span></h1>
      </header>

      <div className={styles.controls}>
        <p className={styles.pageInfo}>Showing page {currentPage} of {totalPages}</p>
        <input
          type="search"
          placeholder="Find databases..."
          value={searchQuery}
          onChange={handleSearch}
          className={styles.searchInput}
          aria-label="Search databases"
        />
      </div>

      <div className={styles.grid}>
        {visibleDatabases.map((database) => (
          <article key={database.id} className={styles.card}>
            <h2 className={styles.cardTitle}>{database.title}</h2>
            <div className={styles.cardBody}>
              <p className={styles.cardText}>
                Access: <strong className={database.access === "view and edit" ? styles.accessEdit : styles.accessView}>{database.access}</strong>
              </p>
              <p className={styles.cardText}>
                Status: <strong className={database.connection === "connected" ? styles.statusConnected : styles.statusDisconnected}>{database.connection}</strong>
              </p>
            </div>
            <div className={styles.cardFooter}>
              <ButtonLink href={database.href} variant="ghost" size="sm" className={styles.openButton}>
                Open
              </ButtonLink>
            </div>
          </article>
        ))}
      </div>

      <nav className={styles.pagination} aria-label="Database pagination">
        <button className={styles.button} onClick={goToPreviousPage} disabled={currentPage === 1}>
          Previous
        </button>
        <button className={styles.button} onClick={goToNextPage} disabled={currentPage === totalPages}>
          Next
        </button>
      </nav>
    </section>
  );
}
