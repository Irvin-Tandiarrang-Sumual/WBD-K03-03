# End-to-End (E2E) Testing Guide (Playwright)

Automated end-to-end browser test suite for the Wedebe platform using Playwright, TypeScript, and Docker.

The suite validates realistic multi-user journeys including role-based authentication, migration drafting and dry-run, multi-leader consensus approval, real-time Server-Sent Events (SSE) progress streaming, and batch quarantine recovery across desktop and mobile viewports (iPhone 14 / 375px).

---

## 1. How to Run Tests

### Option A: Docker Container Mode (Zero-Install)
Runs tests inside the official Playwright container (`mcr.microsoft.com/playwright:v1.50.0-noble`). No local Node.js or browser installation required.

```bash
# Run E2E tests in Docker:
make test-e2e-docker

# Run with clean database volumes:
make test-e2e-fresh
```

The test report is saved to `./e2e/playwright-report/index.html` on the host machine.

---

### Option B: Local Host Mode (For UI and Interactive Debugging)

#### 1. Prerequisites
Ensure the application stack is running:
```bash
docker compose up -d
```

#### 2. Install Dependencies
```bash
cd e2e
npm install
npx playwright install chromium
```

#### 3. Run Tests
```bash
# Playwright UI Mode (Interactive)
npm run test:ui

# Headless terminal execution
npm test

# Headed mode (Visible browser window)
npm run test:headed

# Debug mode (Playwright Inspector)
npm run test:debug

# View last test HTML report
npm run report
```

---

## 2. Architecture and Test Fixtures

The suite uses custom fixtures defined in `e2e/fixtures/test-fixtures.ts` with strict TypeScript typing (zero `any`):

| Fixture | Role | Seed Account | Scenario Scope |
|---|---|---|---|
| `adminPage` | Admin | `admin` : `admin123` | User and group governance, database connections, draft execution. |
| `leaderPage` | Engineering Leader | `leader` : `leader123` | Draft creation, DDL dry-run, engineering group approval vote. |
| `userPage` | Engineering Member | `user` : `user123` | Schema inspection, SQL queries, draft proposal submission. |
| `leaderQaPage` | QA Leader | `leader_qa` : `qa123` | Second leader for multi-group unanimous consensus. |
| `auditorLeadPage` | Auditor Leader | `auditor_lead` : `auditor123` | Read-only access verification (mutation blocked). |

---

## 3. Test Suites & User Journeys (17 Scenarios)

1. **`01_auth_access_control.spec.ts`**:
   - `REG-01`: Successful login and HTTP-only session cookie issuance.
   - `REG-02`: Invalid credentials rejection with validation message.
   - `REG-04`: Logout and session invalidation.
2. **`02_drafting_dryrun.spec.ts`**:
   - `TC-E2E-DFT-001`: Migration proposal submission and syntax checks.
   - `SEC-02`: PostgreSQL wire-protocol dry-run (`conn.Prepare`) error detection before execution.
3. **`03_approval_quorum.spec.ts`**:
   - `APR-01`: Multi-leader consensus voting across groups.
   - `APR-02`: Unanimous rule (single reject marks proposal as rejected).
   - `APR-07`: Vote isolation and weight parity across groups.
4. **`04_sse_async_execution.spec.ts`**:
   - `ASY-01`: Asynchronous submission to background worker pool.
   - `CON-01`: Real-time SSE streaming for execution progress and status transitions.
5. **`05_batch_quarantine_recovery.spec.ts`**:
   - `BAT-01`: Automatic transition to quarantine state (`HTTP 423 Locked`) upon partial batch failure.
   - `BAT-02`: Recovery through Resume and Revert actions.
6. **`06_migration_history_lifecycle.spec.ts`**:
   - `HIS-01`: Target database version history retrieval.
   - `HIS-02`: Cross-version schema comparison (Schema Diff).
   - `HIS-03`: Rollback draft proposal creation with reverse DDL.
   - `HIS-04`: Forward pointer re-apply (Redo).
7. **`07_admin_governance.spec.ts`**:
   - `ADM-01`: User management, group membership updates, and database connection lifecycle.
