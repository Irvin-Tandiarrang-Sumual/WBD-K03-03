# Database Management Platform (WBD-K03-03)

Web-based database management platform to dynamically manage database connections, enforce group-based Role-Based Access Control (RBAC), execute schema migrations with multi-group consensus approval, analyze query execution plans (*EXPLAIN / EXPLAIN ANALYZE*), and recover batch DML executions (*quarantine auto-recovery*).

<p align="center">
  <img src="assets/images/oogway-quotes.jpg" alt="Master Oogway" width="250" align="center"/>
</p>

---

## Tech Stack

- **Backend**: Go 1.25.5, Gin Web Framework, GORM, PostgreSQL (`pgx`), MySQL (`go-sql-driver/mysql`), Swagger/OpenAPI.
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
| `admin` | `admin123` | **Admin** | `admin` (`view_edit`) | User management, group RBAC, database connections, and execution of approved drafts. |
| `leader` | `leader123` | **User** | `engineering` (Leader) | Voting rights for migration proposals (engineering group consensus). |
| `user` | `user123` | **User** | `engineering` (Member) | Regular developer: submit drafts and query team databases. |
| `leader_qa` | `qa123` | **User** | `qa-team` (Leader) | Second voting leader for multi-group consensus validation. |
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

## Observability

From the repository root, start the application and observability stack together. Set `OBSERVABILITY_CONFIG_DIR=./deploy/observability` and use backend port `8080` for the default Prometheus target.

```bash
export OBSERVABILITY_CONFIG_DIR=./deploy/observability
docker compose -f docker-compose.yml -f deploy/observability/docker-compose.yaml -f deploy/observability/app.override.yaml up --build -d
```

In PowerShell, set the variable with `$env:OBSERVABILITY_CONFIG_DIR = './deploy/observability'`. Open Grafana at `http://localhost:3001` to view the migration dashboard. In Explore, use Loki with `{service_name="wbd-backend"} | json` to find application logs and follow a trace ID into Jaeger. Jaeger is also available at `http://localhost:16686`, and Prometheus at `http://localhost:9090`.

The backend writes JSON to stdout and copies it to a shared volume when `APP_LOG_PATH` is set. The Collector forwards those records to Loki and sends application traces to Jaeger. SQL audit records contain the operation and execution metadata while redacting the statement contents and parameter values.

An observed PostgreSQL demo run processed 20 rows in three batches. This excerpt from Loki is for the last batch.

```json
{
  "event": "sql.batch.chunk.complete",
  "status": "success",
  "correlation_id": "wbd-demo-logging-1795736006689804900-execute",
  "trace_id": "2d871c5c8a4bf49c87f479caa88d2dcf",
  "span_id": "5deeb60f818c6532",
  "job_id": 1,
  "batch_number": 3,
  "attempt": 1,
  "rows": 6,
  "query": "UPDATE [redacted]"
}
```

The same trace ID in Jaeger contained the execute request, `queue.submit`, `queue.lock`, `worker.execute_task`, `db.batch.prepare`, and three `db.batch.chunk` spans. Prometheus reported `migration_queue_jobs{status="done"} = 1`, `migration_batch_rows_processed_total{operation="execute"} = 20`, and `db_pool_connections_idle{dialect="postgresql"} = 1`.

---

## Performance Testing

Run these tests against a freshly seeded local demo stack. The script creates and approves its own drafts using the demo accounts. Run the stress test before the large dataset benchmark so both measurements use an otherwise idle queue.

```bash
k6 run -e BASE_URL=http://localhost:8080 backend/test/stress/migrations.js
```

Each phase lasts 14 seconds. Two virtual users establish the baseline, the peak ramps to 12, and recovery brings the load back to two. At each level k6 submits three approved drafts together, two for one PostgreSQL database and one for another. It retries expected HTTP 409 responses and counts a job as complete when its draft reaches `applied`. The thresholds are API p95 below 500 ms, unexpected errors below 1%, and three completed jobs per phase.

### Observed k6 run

The API figures cover database-list requests and job submissions. Request throughput uses the 14-second phase window. Job throughput uses the time from submission to the last observed completion in each group. Mean queue wait and worker duration come from application metrics.

| Phase | API p50 | API p95 | Requests/s | Unexpected errors | Mean queue wait | Mean worker duration | Jobs completed/s |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Baseline, 2 VUs | 21.90 ms | 41.48 ms | 11.64 | 0% | 53.62 ms | 217.58 ms | 3.51 |
| Peak, ramp to 12 VUs | 14.78 ms | 41.20 ms | 60.14 | 0% | 53.47 ms | 230.02 ms | 2.77 |
| Recovery, ramp to 2 VUs | 14.99 ms | 43.72 ms | 23.07 | 0% | 58.65 ms | 188.50 ms | 4.97 |

API p95 stayed within 3 ms across phases. At peak, request throughput was about five times the baseline and jobs took slightly longer. Recovery includes the ramp down, which accounts for some of the extra requests compared with the baseline. All nine jobs finished, with five expected HTTP 409 responses before retry. Each group contains only three jobs, so job throughput describes this run.

### Large batch migration benchmark

On a fresh demo database, load the dataset and time the full-table UPDATE. The script rolls back that UPDATE so the batch uses the same rows and starting values.

```bash
docker compose exec -T demo_postgres psql -U demo_user -d db1_pg -v rows=200000 < backend/test/stress/batch_dataset.sql
k6 run -e BASE_URL=http://localhost:8080 -e BENCHMARK_ROWS=200000 backend/test/stress/migrations.js
```

On Docker Desktop with 12 available CPUs and about 7.5 GiB of RAM, the full-table UPDATE of 200,000 rows took **5.46 s**. The worker processed the same rows in **20 committed batches** of 10,000 in **22.82 s**. A database query confirmed that all 200,000 rows ended with `score = 1`. Batching took longer in this run and recorded progress after every commit. The plain UPDATE measurement excludes commit. A separate 700,000-row run also finished, with a 19.65 s full-table UPDATE and 147.05 s from queue submission to job completion.

## Static Analysis

The [CodeQL workflow](.github/workflows/codeql.yml) scans Go and JavaScript/TypeScript when GitHub Actions runs it on `main` or `develop`, their pull requests, or via manual dispatch. CodeQL scan evidence is pending the first run on a repository with code scanning enabled. Use the Actions run and Security alerts to record the rule, severity, location, impact, and decision for each finding. For fixes, record the commit and the rerun result.

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
