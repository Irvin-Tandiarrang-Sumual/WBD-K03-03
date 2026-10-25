"use client";

import { ChangeEvent, useEffect, useMemo, useState } from "react";
import ButtonLink from "@/components/ui/ButtonLink";
import { api, unwrapPaged } from "@/lib/api";
import type { Database } from "@/lib/types";
import styles from "./DatabaseList.module.css";

const ITEMS_PER_PAGE = 8;

export default function DatabaseList() {
  const [databases, setDatabases] = useState<Database[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [searchQuery, setSearchQuery] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;

    api.databases(currentPage, ITEMS_PER_PAGE)
      .then((body) => {
        if (!active) return;
        setDatabases(unwrapPaged(body));
        setTotalPages(Math.max(1, body.paging?.total_page || 1));
      })
      .catch((requestError: unknown) => {
        if (active) {
          setError(
            requestError instanceof Error
              ? requestError.message
              : "Unable to load databases.",
          );
        }
      });

    return () => {
      active = false;
    };
  }, [currentPage]);

  const visibleDatabases = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();

    return databases.filter((database) =>
      `${database.name} ${database.host} ${database.type}`
        .toLowerCase()
        .includes(query),
    );
  }, [databases, searchQuery]);

  function handleSearch(event: ChangeEvent<HTMLInputElement>) {
    setSearchQuery(event.target.value);
    setCurrentPage(1);
  }

  return (
    <section className={styles.wrapper}>
      <header className={styles.header}>
        <h1>
          <span className={styles.titleBadge}>Databases</span>
        </h1>
      </header>

      <div className={styles.controls}>
        <p className={styles.pageInfo}>
          Showing page {currentPage} of {totalPages}
        </p>
        <input
          type="search"
          placeholder="Find databases..."
          value={searchQuery}
          onChange={handleSearch}
          className={styles.searchInput}
          aria-label="Search databases"
        />
      </div>

      {error && <p role="alert">{error}</p>}

      <div className={styles.grid}>
        {visibleDatabases.map((database) => (
          <article key={database.id} className={styles.card}>
            <h2 className={styles.cardTitle}>{database.name}</h2>
            <div className={styles.cardBody}>
              <p className={styles.cardText}>
                Type: <strong>{database.type}</strong>
              </p>
              <p className={styles.cardText}>
                Endpoint:{" "}
                <strong>
                  {database.host}:{database.port}
                </strong>
              </p>
            </div>
            <div className={styles.cardFooter}>
              <ButtonLink
                href={`/databases/${database.id}/schema`}
                variant="ghost"
                size="sm"
                className={styles.openButton}
              >
                Open
              </ButtonLink>
            </div>
          </article>
        ))}
        {!error && visibleDatabases.length === 0 && <p>No databases found.</p>}
      </div>

      <nav className={styles.pagination} aria-label="Database pagination">
        <button
          className={styles.button}
          onClick={() => setCurrentPage((page) => Math.max(1, page - 1))}
          disabled={currentPage === 1}
        >
          Previous
        </button>
        <button
          className={styles.button}
          onClick={() =>
            setCurrentPage((page) => Math.min(totalPages, page + 1))
          }
          disabled={currentPage === totalPages}
        >
          Next
        </button>
      </nav>
    </section>
  );
}
