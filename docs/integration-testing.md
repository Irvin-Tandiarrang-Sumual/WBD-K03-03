# Integration Testing Guide (Go & Testcontainers)

Integration tests validate real database drivers, transaction boundaries, advisory lock managers, and background queue workers against ephemeral database containers (**PostgreSQL 15** and **MySQL 8.4**) using **Testcontainers-Go**.

---

## 1. Prerequisites

- **Docker Engine / Docker Desktop**: Must be running on the host machine to allow Testcontainers to spin up ephemeral database instances.
- **Go 1.23+**: Installed locally.

---

## 2. How to Run

From the repository root:
```bash
# Run all integration tests via Makefile
make test-integration

# Run integration tests with coverage profiling
make test-integration-cover
```

Or execute directly from the `backend/` directory:
```bash
cd backend
go test -v -tags=integration ./test/integration/... -count=1
```

---

## 3. Test Suites & Coverage Breakdown

The integration suite consists of 8 comprehensive test suites:

1. **`auth_journey_test.go`**: Validates session cookie issuance, multi-role authentication, CSRF token validation (`X-CSRF-Token`), and logout invalidation against internal PostgreSQL tables.
2. **`admin_rbac_journey_test.go`**: Verifies Role-Based Access Control (RBAC) enforcement across database groups, verifying that `view_only` users cannot execute mutations, and validates group membership assignments.
3. **`postgres_target_journey_test.go`**: Tests the PostgreSQL driver (`pgx`), executing multi-stage schema migrations, syntax validation, and reverse DDL generation on real target PostgreSQL instances.
4. **`mysql_target_journey_test.go`**: Tests the MySQL 8.4 driver, version tracking in MySQL metadata tables, and non-transactional DDL execution behavior.
5. **`draft_journey_test.go`**: Tests the migration proposal lifecycle, multi-leader consensus voting (unanimous 100% approval requirement), and execution of approved drafts.
6. **`concurrency_race_test.go`**: Tests concurrent migration executions against the same database, verifying race conditions on PostgreSQL advisory locks (`pg_try_advisory_lock`), and ensuring conflicting requests return `HTTP 409 Conflict`.
7. **`batch_recovery_test.go`**: Simulates partial execution failures during large batch DML updates, automatic transition to quarantine status (`HTTP 423 Locked`), and verifies recovery via `Resume` and `Revert`.
8. **`migration_queue_test.go`**: Validates background worker reliability, job queue polling (`migration_jobs`), worker heartbeats, and idempotent job re-execution after abrupt worker restarts.
