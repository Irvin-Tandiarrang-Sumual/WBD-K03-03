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
| **Stress & Performance** | Pengujian | Uji k6 untuk melihat respons API dan antrean migrasi ketika beban naik lalu turun. | [docs/stress-testing.md](docs/stress-testing.md) |
| **Observability** | Infrastructure | Centralized monitoring with Prometheus, Grafana dashboards, Loki log queries, and Jaeger distributed tracing. | [docs/observability.md](docs/observability.md) |
| **Security SAST** | Security | Static Application Security Testing with CodeQL, automated CI workflows, and OWASP/CWE evaluations. | [docs/security.md](docs/security.md) |
| **Demo Setup** | Guide | Seed accounts, target database profiles (`db1` to `db4`), and interactive scenario walkthroughs. | [docs/demo.md](docs/demo.md) |

---

## Uji Beban Penuh k6 Milestone 2

Jalankan profil penuh pada stack demo lokal yang sudah diisi data awal. [Skrip k6](stress/stress_suite.js) menguji daftar database, riwayat migrasi, query, dan explain sambil mengirim draf migrasi yang sudah disetujui ke database yang sama maupun berbeda. Skrip memantau status setiap job yang diterima hingga selesai. Job dihitung berhasil ketika statusnya `applied`.

```powershell
$env:OBSERVABILITY_CONFIG_DIR = './deploy/observability'
$compose = @('-f', 'docker-compose.yml', '-f', 'deploy/observability/docker-compose.yaml', '-f', 'deploy/observability/app.override.yaml')
$results = (New-Item -ItemType Directory -Path (Join-Path $env:TEMP ("wbd-k6-" + (Get-Date -Format 'yyyyMMdd-HHmmss')))).FullName
docker compose -p wbd-bonus-demo @compose --profile stress run --rm --no-deps -v "${results}:/results" -e STRESS_PROFILE=full k6 run --out json=/results/full.json stress_suite.js
```

VU adalah pengguna yang disimulasikan k6. Pada [profil penuh di repo](stress/config/environments.js), jumlah pengguna untuk skenario API dinaikkan bertahap lalu diturunkan kembali. Batas uji API adalah **p95 < 800 ms**, **p99 < 2.000 ms**, dan **error tak terduga < 3%**. P50 adalah waktu respons tengah. P95 menunjukkan waktu yang mencakup 95% request, jadi membantu melihat kelambatan yang tidak tergambar dari nilai tengah.

### Hasil profil penuh

Pengujian dilakukan di Docker lokal pada 27 November 2026 dengan commit `f8fc148`. Dari **26.688 request API**, p95 keseluruhan mencapai **914 ms**. P99 sebesar **1,87 detik** memenuhi batas 2 detik. k6 keluar dengan kode 99 karena p95 melewati target. Semua **55 job yang diterima mencapai `applied`**.

| Fase | VU API | API p50 | API p95 | Request/detik | Error tak terduga | Job diterima / selesai | Waktu antre p95 | Durasi worker p50 | Job selesai/detik |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Beban awal, 30 dtk | 2 → 10 | 35 ms | 123 ms | 9,90 | 0 / 297 | 20 / 20 | 170 ms | 624 ms | 0,667 |
| Di atas normal, 60 dtk | 10 → 30 | 34 ms | 145 ms | 106,98 | 0 / 6.419 | 35 / 35 | 100 ms | 356 ms | 0,583 |
| Naik ke puncak, 60 dtk | 30 → 75 | 153 ms | 551 ms | 153,93 | 0 / 9.236 | 0 / 0 | — | — | 0 |
| Bertahan di puncak, 30 dtk | 75 | 460 ms | 2.113 ms | 102,80 | 1 / 3.084 (0,032%) | 0 / 0 | — | — | 0 |
| Penurunan beban, 30 dtk | 75 → 10 | 338 ms | 1.289 ms | 128,33 | 0 / 3.850 | 0 / 0 | — | — | 0 |
| Pemulihan, 45 dtk | 10 → 0 | 25 ms | 224 ms | 84,44 | 0 / 3.800 | 0 / 0 | — | — | 0 |

Angka per fase diambil dari output JSON k6 dan dibagi durasi fase untuk menghitung throughput. Job selesai dikelompokkan berdasarkan fase pengajuannya. Waktu antre dihitung dari waktu job tersimpan hingga worker mulai. Durasi worker dihitung dari log mulai hingga status akhir job tersimpan. Median dari awal polling hingga job terkonfirmasi selesai adalah **689 ms**.

Data mentah memuat satu timeout ketika beban bertahan di puncak. Metrik error bawaan mencatat 0% karena request berstatus 0 ikut dihitung sebagai sukses. Tabel memasukkan timeout itu dan memisahkan **1.578 respons HTTP 409** serta **80 respons HTTP 423**. Penghitung `lock_conflicts` di skrip menghitung 409 dua kali dan mencakup pengajuan berulang. Untuk waktu antre, tabel memakai catatan job dan log worker karena metrik k6 mencatat nol.

Pada dua fase awal, API bisa melayani lebih banyak request per detik dengan waktu respons yang masih dekat dengan beban awal. Saat beban dipertahankan di puncak, jumlah request yang dilayani per detik turun dari sekitar 154 ke 103, sementara p95 naik dari 551 ms ke 2,11 detik. Artinya, saat beban tinggi bertahan, pengguna mulai menunggu lebih lama dan target respons terlewati. Setelah beban turun, p95 membaik ke 224 ms, masih di atas 123 ms pada beban awal. Satu request setelah tes selesai merespons dalam 9,72 ms. Waktu sampai respons kembali stabil belum diukur.

Seluruh 55 job yang diterima masuk pada dua fase awal dan selesai. Pada fase berikutnya, termasuk saat beban puncak, tidak ada job baru yang diterima. Karena itu, pengukuran waktu antre dan durasi worker hanya mencakup dua fase awal.

## Bukti dan Analisis CodeQL

CodeQL membaca kode untuk mencari pola yang berpotensi menimbulkan celah keamanan. Pemindaian Go dan JavaScript/TypeScript selesai dan hasil tersedia di Actions. [Bukti run CodeQL di GitHub Actions](https://github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/actions/runs/45959900310)

| Bahasa | File yang dipindai | Temuan |
|---|---:|---:|
| Go | 135 dari 182 file | 11 |
| JavaScript dan TypeScript | 14 dari 14 JS, 136 dari 136 TS | 0 |

Di backend Go, terdapat temuan terkait dua hal. Pertama, aplikasi bisa membaca ID migrasi sebagai angka yang keliru pada sistem 32-bit. Kedua, sebagian input ikut membentuk perintah database, sehingga perlu dipastikan perintah tetap sesuai tujuan pengguna. Beberapa lokasi sudah punya pemeriksaan yang mencegah dampak tersebut. Untuk sisi web, scan ini melaporkan nol temuan pada file yang dipindai.

| Rule dan skor keamanan | Lokasi | Dampak dan keputusan |
|---|---|---|
| `go/incorrect-integer-conversion`, tinggi (8,1), 2 temuan | `backend/internal/adapters/http/handlers/migration_handler.go:184–185` | ID migrasi berpotensi terpotong pada sistem 32-bit. Pembacaan ID sekarang menolak angka yang melebihi ukuran `uint` pada sistem yang dipakai. |
| `go/incorrect-integer-conversion`, tinggi (8,1), 2 temuan | `backend/pkg/ginutil/ginutil.go:20,23` | Jalur ini sudah mengecek hasil konversi bolak-balik dan menolak nilai yang berubah sebelum mengembalikan ID. Dari peninjauan kode, kedua temuan ini dinilai false positive. |
| `go/sql-injection`, tinggi (8,8), 3 temuan | `backend/internal/adapters/driver/mysql/connect.go:110,136` dan `backend/internal/adapters/driver/postgresql/connect.go:100` | Nama database masuk ke SQL pembuatan database. Jalur pemanggil sudah membatasi nama ke huruf, angka, dan garis bawah. Ketiga temuan dinilai false positive pada jalur yang ditinjau. |
| `go/sql-injection`, tinggi (8,8), 2 temuan | `backend/internal/adapters/driver/mysql/connect.go:143,147` | Nama akun sekarang diperlakukan sebagai nama akun MySQL yang dikutip dengan benar. Pemberian hak otomatis dibatasi ke database yang didaftarkan. Integration test memverifikasi bahwa akun dapat memakai database baru tanpa hak di tingkat server. |
| `go/sql-injection`, tinggi (8,8), 2 temuan | `backend/internal/adapters/driver/mysql/executor_repository.go:162` dan `backend/internal/adapters/driver/postgresql/executor_repository.go:163` | SQL Console memeriksa izin dan memvalidasi query di service. Driver sekarang menghentikan eksekusi jika transaksi baca gagal dibuka. Karena fitur ini memang menjalankan SQL yang dimasukkan pengguna, hasil scan ulang masih perlu ditinjau. |


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
