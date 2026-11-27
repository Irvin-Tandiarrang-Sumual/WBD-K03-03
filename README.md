# Database Management Platform (Wedebe - WBD-K03-03)

Web-based database management platform to dynamically manage database connections, enforce group-based Role-Based Access Control (RBAC), execute schema migrations with multi-group consensus approval, analyze query execution plans (EXPLAIN / EXPLAIN ANALYZE), and recover batch DML executions with automatic quarantine.

<p align="center">
  <img src="assets/images/oogway-quotes.jpg" alt="Master Oogway" width="250" align="center"/>
</p>

---

## Table of Contents

- [Tech Stack](#tech-stack)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Environment Configuration](#environment-configuration)
  - [Running the Application](#running-the-application)
  - [Operational Commands](#operational-commands)
- [Technical Documentation (`docs/`)](#technical-documentation-docs)
  - [Unit Testing](docs/unit-testing.md)
  - [Integration Testing](docs/integration-testing.md)
  - [End-to-End (E2E) Testing](docs/e2e-testing.md)
  - [Performance & Stress Testing](docs/stress-testing.md)
  - [Observability & Tracing](docs/observability.md)
  - [Security SAST (CodeQL)](docs/security.md)
  - [Demo Setup & Walkthroughs](docs/demo.md)
- [API Endpoints](#api-endpoints)
  - [1. Authentication (`/auth`)](#1-authentication-auth)
  - [2. System Health (`/health`)](#2-system-health-health)
  - [3. Admin Governance (`/admin`)](#3-admin-governance-admin)
  - [4. Database Exploration & Queries (`/me/databases`)](#4-database-exploration--queries-medatabases)
  - [5. Schema Migrations (`/me/databases/:id/migrations`)](#5-schema-migrations-medatabasesidmigrations)
  - [6. Migration Proposals & Consensus Drafts](#6-migration-proposals--consensus-drafts)
  - [7. Batch DML Recovery & Quarantine (`/drafts`)](#7-batch-dml-recovery--quarantine-drafts)
- [Contributors](#contributors)

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

## Getting Started

### Prerequisites
- Docker Engine / Docker Desktop & Docker Compose v2+.

### Environment Configuration
Copy the sample environment configuration file:
```bash
cp .env.example .env
```
The `SEED_TEST_DATA` variable in `.env` controls database initialization:
- `SEED_TEST_DATA=true` (default): Automatically runs internal migrations and seeds mock demo data (users, groups, target databases, migration history, and drafts).
- `SEED_TEST_DATA=false`: Runs internal migrations only, leaving a clean database.

### Running the Application
Start all application services from the root directory:
```bash
docker compose up --build -d
```
Wait until all containers report healthy. Access the services at:
- **Frontend Web Console**: [http://localhost:3000](http://localhost:3000)
- **Backend API Server**: [http://localhost:8080](http://localhost:8080)
- **Interactive Swagger Docs**: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

### Operational Commands

```bash
# Run internal database schema migrations manually
make migrate

# Re-run initial demo data seeding
make seed

# Stop all application containers
docker compose down

# Stop all containers and wipe database volumes
docker compose down -v
```

---

## Technical Documentation (`docs/`)

Detailed documentation is organized by category in the [`docs/`](docs/) directory:

| Document | Category | Description | Link |
|---|---|---|---|
| **Unit Testing** | Testing | Go unit tests across 22 packages (parsers, hazard classification, DDL reverse generators) and TypeScript verification. | [docs/unit-testing.md](docs/unit-testing.md) |
| **Integration Testing** | Testing | Ephemeral container testing (PostgreSQL 15 & MySQL 8.4) with Testcontainers-Go covering 8 journey test suites. | [docs/integration-testing.md](docs/integration-testing.md) |
| **End-to-End Testing** | Testing | Playwright browser automation covering 17 critical user journeys across desktop and mobile (375px) viewports. | [docs/e2e-testing.md](docs/e2e-testing.md) |
| **Stress & Performance** | Testing | k6 load suite testing concurrency up to 75 VUs, advisory lock contention, worker queues, and MTTR. | [docs/stress-testing.md](docs/stress-testing.md) |
| **Observability** | Infrastructure | Centralized monitoring with Prometheus, Grafana dashboards, Loki log queries, and Jaeger distributed tracing. | [docs/observability.md](docs/observability.md) |
| **Security SAST** | Security | Static Application Security Testing with CodeQL, automated CI workflows, and OWASP/CWE evaluations. | [docs/security.md](docs/security.md) |
| **Demo Setup** | Guide | Seed accounts, target database profiles (`db1` to `db4`), and interactive scenario walkthroughs. | [docs/demo.md](docs/demo.md) |

---

## API Endpoints

All protected endpoints require an encrypted session cookie (`session=<token>`) and CSRF validation headers (`X-CSRF-Token`), except `/api/v1/health` and `/api/v1/auth/login`.

### 1. Authentication (`/auth`)
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/auth/login` | Authenticate user, issue session cookie and CSRF token |
| `POST` | `/api/v1/auth/logout` | Terminate active user session and invalidate token |
| `GET` | `/api/v1/auth/me` | Retrieve profile of the currently authenticated user |

### 2. System Health (`/health`)
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/health` | Service health status check |

### 3. Admin Governance (`/admin`)
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/admin/users` | List all registered users (paginated) |
| `POST` | `/api/v1/admin/users` | Register a new user |
| `GET` | `/api/v1/admin/users/:id` | Retrieve user details by ID |
| `PATCH` | `/api/v1/admin/users/:id` | Update user profile details |
| `DELETE` | `/api/v1/admin/users/:id` | Delete a user account |
| `PATCH` | `/api/v1/admin/users/:id/role` | Update user platform role (`admin` / `user`) |
| `GET` | `/api/v1/admin/groups` | List all access groups |
| `POST` | `/api/v1/admin/groups` | Create a new access group |
| `GET` | `/api/v1/admin/groups/:id` | Retrieve group details and database permissions |
| `PATCH` | `/api/v1/admin/groups/:id` | Update group details and database mappings |
| `DELETE` | `/api/v1/admin/groups/:id` | Delete an access group |
| `POST` | `/api/v1/admin/groups/:id/users/:user_id` | Add a user to an access group |
| `DELETE` | `/api/v1/admin/groups/:id/users/:user_id` | Remove a user from an access group |
| `GET` | `/api/v1/admin/databases` | List all registered target databases |
| `POST` | `/api/v1/admin/databases` | Connect and register a new target database |
| `GET` | `/api/v1/admin/databases/:id` | Retrieve database connection details |
| `DELETE` | `/api/v1/admin/databases/:id` | Delete a target database connection |

### 4. Database Exploration & Queries (`/me/databases`)
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/me/databases` | List databases accessible to current user |
| `GET` | `/api/v1/me/databases/:database_id` | Retrieve database metadata and table schemas |
| `POST` | `/api/v1/me/databases/:database_id/query` | Execute custom read-only SQL queries |
| `POST` | `/api/v1/me/databases/:database_id/explain` | View query execution plan (`EXPLAIN`) |
| `POST` | `/api/v1/me/databases/:database_id/explain-analyze` | View actual query execution plan (`EXPLAIN ANALYZE`) |

### 5. Schema Migrations (`/me/databases/:id/migrations`)
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/me/databases/:database_id/migrations` | Retrieve database migration history |
| `GET` | `/api/v1/me/databases/:database_id/migrations/:migration_id` | View single migration version details |
| `GET` | `/api/v1/me/databases/:database_id/migrations/compare` | Compare schema diff between two migration versions |
| `POST` | `/api/v1/me/databases/:database_id/migrations` | Directly apply a schema migration |

### 6. Migration Proposals & Consensus Drafts
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/me/databases/:database_id/drafts` | Submit a new migration proposal draft |
| `GET` | `/api/v1/me/databases/:database_id/drafts` | List draft proposals for a database |
| `POST` | `/api/v1/me/databases/:database_id/rollback-draft` | Submit a rollback draft proposal with reverse DDL |
| `POST` | `/api/v1/me/databases/:database_id/forward-draft` | Submit a forward / re-apply draft proposal |
| `POST` | `/api/v1/me/databases/:database_id/migration-drafts/:draft_id/vote` | Cast vote (`approve` / `reject`) on a proposal |
| `POST` | `/api/v1/me/databases/:database_id/migration-drafts/:draft_id/execute` | Execute an approved consensus draft |
| `GET` | `/api/v1/drafts/:id` | Retrieve draft details, status, and isolation state |
| `POST` | `/api/v1/drafts/:id/approvals` | Submit consensus approval or rejection vote |
| `POST` | `/api/v1/drafts/:id/execute` | Submit approved draft for asynchronous execution |

### 7. Batch DML Recovery & Quarantine (`/drafts`)
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/drafts/:id/resume` | Resume interrupted batch DML execution from last checkpoint |
| `POST` | `/api/v1/drafts/:id/revert-batch` | Roll back committed chunks of a failed batch draft in reverse order |
| `GET` | `/api/v1/drafts/:id/progress` | Check batch execution progress percentage and row counts |
| `GET` | `/api/v1/drafts/:id/progress/stream` | Real-time Server-Sent Events (SSE) execution progress stream |

---

## Contributors

| Student ID | Name |
|---|---|
| 13524006 | Gabriella Botimada Lubis |
| 13524030 | Irvin Tandiarrang Sumual |
| 13524033 | Ray Owen Martin |
