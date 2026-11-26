# Database Management Platform (WBD-K03-03)

Web-based database management platform to dynamically manage database connections, enforce group-based Role-Based Access Control (RBAC), execute schema migrations with multi-group consensus approval, analyze query execution plans (*EXPLAIN / EXPLAIN ANALYZE*), and recover batch DML executions (*quarantine auto-recovery*).

<p align="center">
  <img src="assets/images/oogway-quotes.jpg" alt="Master Oogway" width="250" align="center"/>
</p>

---

## Tech Stack

- **Backend**: Go 1.23, Gin Web Framework, GORM, PostgreSQL (`pgx`), MySQL (`go-sql-driver/mysql`), Swagger/OpenAPI.
- **Frontend**: React 19, React Router v7 (SPA Mode), Vite, Tailwind CSS, TypeScript.
- **Database Storage**:
  - `internal_db`: PostgreSQL 15 (platform metadata, users, groups, and migration history).
  - `demo_postgres`: PostgreSQL 15 engine (target databases: `db1_pg`, `db3_analytics`, `db4_billing`).
  - `demo_mysql`: MySQL 8.4 engine (target database: `db2_mysql`).
- **Infrastructure**: Docker & Docker Compose.

---

## Usage Guide

### 1. Prerequisites
- Docker Engine / Docker Desktop & Docker Compose v2+.

### 2. Environment Configuration & Data Seeding
Copy the environment template in the project root:
```bash
cp .env.example .env
```
> **Automatic Seeding on Docker Startup:**
> In `.env`, the `SEED_TEST_DATA` variable controls startup behavior:
> - `SEED_TEST_DATA=true` *(default)*: Backend automatically runs schema migrations and seeds test data (users, groups, demo database connections, migration history, and consensus drafts).
> - `SEED_TEST_DATA=false`: Backend only runs internal schema migrations without seeding mock test data (clean database).

### 3. Running the Application
Start all services from the root directory:
```bash
docker compose up --build -d
```
Wait until all containers are healthy. Access the services at:
- **Frontend Web Console**: [http://localhost:3000](http://localhost:3000)
- **Backend API Server**: [http://localhost:8080](http://localhost:8080)
- **Interactive Swagger Docs**: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

### 4. Default Seed Accounts

| Username | Password | Role | Group & Access Level | Testing Scenario |
|---|---|---|---|---|
| `admin` | `admin123` | **Admin** | `admin` (`view_edit`) | Full access: user management, group RBAC, database connections, and direct execution. |
| `leader` | `leader123` | **User** | `engineering` (Leader) | Voting rights for migration proposals (engineering group consensus). |
| `user` | `user123` | **User** | `engineering` (Member) | Regular developer: submit drafts and query team databases. |
| `leader_qa` | `leaderqa123` | **User** | `qa-team` (Leader) | Second voting leader for multi-group consensus validation. |
| `auditor_lead` | `auditor123` | **User** | `auditors` (Leader) | `view_only` access (read & explain queries, restricted from schema edits). |

### 5. Demo Target Databases

| Database | Engine | Initial Seed State | Testing Scenario |
|---|---|---|---|
| `db1_pg` | PostgreSQL | Normal (`v3_applied`) | Reversible migration history (eligible for safe rollback). |
| `db2_mysql` | MySQL | Normal (`v3_applied`) | Irreversible migration (destructive column drop). |
| `db3_analytics`| PostgreSQL | Post-Rollback (`v1`) | Active `Forward` pointer for Re-apply / Redo testing. |
| `db4_billing` | PostgreSQL | **Quarantined (423)** | Partial batch DML failure, ready for Resume or Revert batch recovery. |

### 6. Key Commands

```bash
# Run internal database schema migration
make migrate        # or: cd backend && go run ./cmd/migrate

# Re-run data seeding
make seed           # or: cd backend && go run ./cmd/seed

# Run backend unit tests
make test           # or: cd backend && go test ./...

# Run frontend typecheck
cd frontend-spa && npx tsc --noEmit

# Stop containers
docker compose down

# Stop containers and wipe database volumes
docker compose down -v
```

---

## API Endpoints

PostgreSQL batch migrations support a single `UPDATE` with a single-column integer primary key as its batch column. End SQL up and SQL down with a predicate such as `AND id BETWEEN ? AND ?` or `WHERE id BETWEEN ? AND ?`. The worker binds the exact keys for each batch and stores commit evidence alongside the target changes. Revert uses the approved SQL down in reverse batch order and limits it to keys actually changed by SQL up.

Migration jobs are stored in the platform PostgreSQL database. When workers restart, pending jobs remain available for execution. Each database can have only one active job. A worker holds a PostgreSQL advisory lock while it processes the job, and PostgreSQL releases the lock if the worker connection ends.

If a worker disconnects mid-execution, the next worker checks whether the draft already has a recorded outcome. When the outcome is still unclear, the job is held as `uncertain` and blocks new jobs on that database. Verify the actual changes on the target against the draft and any committed checkpoints before resuming work. Automatic reconciliation of uncertain outcomes is still pending.

All protected endpoints require session cookie (`session=<token>`) and CSRF validation headers, except `/api/v1/health` and `/api/v1/auth/login`.

### 1. Authentication (`/auth`)
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/auth/login` | User login, set session cookie & CSRF token |
| `POST` | `/api/v1/auth/logout` | Terminate active user session |
| `GET` | `/api/v1/auth/me` | Get current user profile |

### 2. Admin Panel (`/admin`)
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/admin/users` | List all users (paginated) |
| `POST` | `/api/v1/admin/users` | Create new user |
| `GET` / `PATCH` / `DELETE` | `/api/v1/admin/users/:id` | Get detail, update, or delete user |
| `PATCH` | `/api/v1/admin/users/:id/role` | Update user role (`admin` / `user`) |
| `GET` / `POST` | `/api/v1/admin/groups` | List or create access group |
| `GET` / `PATCH` / `DELETE` | `/api/v1/admin/groups/:id` | Get detail, update database access, or delete group |
| `POST` / `DELETE` | `/api/v1/admin/groups/:id/users/:user_id` | Add or remove user from group |
| `GET` / `POST` | `/api/v1/admin/databases` | List registered databases or register new database |
| `GET` / `DELETE` | `/api/v1/admin/databases/:id` | Get detail or delete database connection |

### 3. Database Exploration & Queries (`/me/databases`)
| Method | Endpoint | Access Level | Description |
|---|---|---|---|
| `GET` | `/api/v1/me/databases` | — | List databases accessible to current user |
| `GET` | `/api/v1/me/databases/:database_id` | `view_only` | Get database detail & table schema metadata |
| `POST` | `/api/v1/me/databases/:database_id/query` | `view_only` | Execute custom SQL query |
| `POST` | `/api/v1/me/databases/:database_id/explain` | `view_only` | Analyze query execution plan (`EXPLAIN`) |
| `POST` | `/api/v1/me/databases/:database_id/explain-analyze` | `view_only` | Analyze actual query execution plan (`EXPLAIN ANALYZE`) |

### 4. Migration Management & Consensus Drafts (`/me/databases` & `/me/drafts`)
| Method | Endpoint | Access Level | Description |
|---|---|---|---|
| `GET` | `/api/v1/me/databases/:database_id/migrations` | `view_only` | Database migration version history |
| `GET` | `/api/v1/me/databases/:database_id/migrations/:id` | `view_only` | Get single migration detail |
| `GET` | `/api/v1/me/databases/:database_id/migrations/compare` | `view_only` | Compare two migration versions (*Schema Diff*) |
| `POST` | `/api/v1/me/databases/:database_id/migrations` | `view_edit` | Directly execute schema migration |
| `POST` | `/api/v1/me/databases/:database_id/drafts` | `view_edit` | Submit new migration draft proposal |
| `GET` | `/api/v1/me/databases/:database_id/drafts` | `view_only` | List draft proposals for database |
| `POST` | `/api/v1/me/databases/:database_id/rollback-draft` | `view_edit` | Create draft proposal for rollback |
| `POST` | `/api/v1/me/databases/:database_id/forward-draft` | `view_edit` | Create draft proposal for forward / re-apply |
| `POST` | `/api/v1/me/databases/:database_id/migration-drafts/:draft_id/vote` | Leader | Vote (*approve* / *reject*) on draft proposal |
| `POST` | `/api/v1/me/databases/:database_id/migration-drafts/:draft_id/execute` | `view_edit` | Execute approved consensus draft |

### 5. Batch DML Recovery & Quarantine Recovery (`/me/drafts`)
| Method | Endpoint | Access Level | Description |
|---|---|---|---|
| `GET` | `/api/v1/me/drafts/:id` | `view_only` | Get draft status & quarantine isolation details |
| `POST` | `/api/v1/me/drafts/:id/resume` | `view_edit` | Resume interrupted batch DML execution |
| `POST` | `/api/v1/me/drafts/:id/revert-batch` | `view_edit` | Revert and rollback executed batch DML |
| `GET` | `/api/v1/me/drafts/:id/progress` | `view_only` | Get current batch execution progress |
| `GET` | `/api/v1/me/drafts/:id/progress/stream` | `view_only` | Server-Sent Events (SSE) real-time stream of batch progress |

---

## Contributors

| NIM | Name |
|---|---|
| 13524006 | Gabriella Botimada Lubis |
| 13524030 | Irvin Tandiarrang Sumual |
| 13524033 | Ray Owen Martin |
