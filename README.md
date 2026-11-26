# Database Management Platform (WBD-K03-03)

Platform manajemen database berbasis web untuk mengelola koneksi database secara dinamis, mengontrol hak akses pengguna berbasis grup (*Role-Based Access Control*), mengeksekusi migrasi skema dengan konsensus persetujuan multi-grup, analisis rencana eksekusi query (*EXPLAIN / EXPLAIN ANALYZE*), serta pemulihan batch DML (*quarantine auto-recovery*).

<p align="center">
  <img src="assets/images/oogway-quotes.jpg" alt="Master Oogway" width="250" align="center"/>
</p>

---

## 🛠️ Tech Stack

- **Backend**: Go 1.23, Gin Web Framework, GORM, PostgreSQL (`pgx`), MySQL (`go-sql-driver/mysql`), Swagger/OpenAPI.
- **Frontend**: React 19, React Router v7 (SPA Mode), Vite, Tailwind CSS, TypeScript.
- **Database Storage**:
  - `internal_db`: PostgreSQL 15 (metadata platform, user, grup, dan riwayat migrasi).
  - `demo_postgres`: Engine PostgreSQL 15 (database target: `db1_pg`, `db3_analytics`, `db4_billing`).
  - `demo_mysql`: Engine MySQL 8.4 (database target: `db2_mysql`).
- **Infrastruktur**: Docker & Docker Compose.

---

## Usage

### 1. Prasyarat
- Docker Engine / Docker Desktop & Docker Compose v2+.

### 2. Konfigurasi Lingkungan & Data Seeding
Salin file konfigurasi `.env` di root direktori proyek:
```bash
cp .env.example .env
```
> **Pengaturan Seeding Otomatis di Startup Docker:**
> Di dalam file `.env`, terdapat variabel `SEED_TEST_DATA`:
> - `SEED_TEST_DATA=true` *(default)*: Backend akan otomatis menjalankan migrasi skema sekaligus mengisi (*seeding*) data uji coba (user, grup, koneksi database demo, riwayat migrasi, dan draft konsensus).
> - `SEED_TEST_DATA=false`: Backend hanya akan menjalankan migrasi skema tabel internal tanpa data uji coba awal (database bersih).

### 3. Menjalankan Aplikasi
Jalankan seluruh service dari direktori root:
```bash
docker compose up --build -d
```
Tunggu hingga semua kontainer berstatus *healthy*. Akses service di:
- **Frontend Web Console**: [http://localhost:3000](http://localhost:3000)
- **Backend API Server**: [http://localhost:8080](http://localhost:8080)
- **Interactive Swagger Docs**: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

### 4. Akun Pengujian Bawaan (Seed Data)

| Username | Password | Role | Group & Hak Akses | Deskripsi Skenario Uji |
|---|---|---|---|---|
| `admin` | `admin123` | **Admin** | `admin` (`view_edit`) | Akses penuh: manajemen user, grup, koneksi database, dan eksekusi langsung. |
| `leader` | `leader123` | **User** | `engineering` (Leader) | Hak voting proposal migrasi (konsensus grup engineering). |
| `user` | `user123` | **User** | `engineering` (Member) | Pengguna reguler untuk mengajukan draft & query database tim. |
| `leader_qa` | `leaderqa123` | **User** | `qa-team` (Leader) | Hak voting kedua untuk validasi multi-group consensus. |
| `auditor_lead` | `auditor123` | **User** | `auditors` (Leader) | Hak akses `view_only` (read & explain query, tidak bisa edit skema). |

### 5. Basis Data Target Demo

| Database | Engine | Status Awal di Seed | Skenario Pengujian |
|---|---|---|---|
| `db1_pg` | PostgreSQL | Normal (`v3_applied`) | Riwayat migrasi reversibel (bisa rollback aman). |
| `db2_mysql` | MySQL | Normal (`v3_applied`) | Migrasi ireversibel (destructive drop column). |
| `db3_analytics`| PostgreSQL | Post-Rollback (`v1`) | Memiliki pointer `Forward` aktif untuk uji Re-apply / Redo. |
| `db4_billing` | PostgreSQL | **Quarantined (423)** | Batch DML gagal parsial, siap diuji untuk Resume atau Revert batch. |

### 6. Perintah Operasional Penting

```bash
make migrate        # atau: cd backend && go run ./cmd/migrate

make seed           # atau: cd backend && go run ./cmd/seed

make test           # atau: cd backend && go test ./...

cd frontend-spa && npx tsc --noEmit

docker compose down

docker compose down -v
```

---

## Daftar Endpoint API

Semua endpoint dilindungi cookie sesi (`session=<token>`) dan validasi header CSRF, kecuali `/api/v1/health` dan `/api/v1/auth/login`.

### 1. Autentikasi (`/auth`)
| Method | Endpoint | Deskripsi |
|---|---|---|
| `POST` | `/api/v1/auth/login` | Login user, set cookie sesi & token CSRF |
| `POST` | `/api/v1/auth/logout` | Mengakhiri sesi aktif |
| `GET` | `/api/v1/auth/me` | Profil pengguna saat ini |

### 2. Panel Admin (`/admin`)
| Method | Endpoint | Deskripsi |
|---|---|---|
| `GET` | `/api/v1/admin/users` | List seluruh user (paginated) |
| `POST` | `/api/v1/admin/users` | Buat user baru |
| `GET` / `PATCH` / `DELETE` | `/api/v1/admin/users/:id` | Detail, update, atau hapus user |
| `PATCH` | `/api/v1/admin/users/:id/role` | Ubah role user (`admin` / `user`) |
| `GET` / `POST` | `/api/v1/admin/groups` | List atau buat access group baru |
| `GET` / `PATCH` / `DELETE` | `/api/v1/admin/groups/:id` | Detail, update akses database, atau hapus grup |
| `POST` / `DELETE` | `/api/v1/admin/groups/:id/users/:user_id` | Tambah atau keluarkan user dari grup |
| `GET` / `POST` | `/api/v1/admin/databases` | List database terdaftar atau daftarkan DB baru |
| `GET` / `DELETE` | `/api/v1/admin/databases/:id` | Detail atau hapus koneksi database |

### 3. Eksplorasi Database & Query Pengguna (`/me/databases`)
| Method | Endpoint | Hak Akses | Deskripsi |
|---|---|---|---|
| `GET` | `/api/v1/me/databases` | — | Daftar database yang diizinkan untuk user |
| `GET` | `/api/v1/me/databases/:database_id` | `view_only` | Detail & metadata skema tabel database |
| `POST` | `/api/v1/me/databases/:database_id/query` | `view_only` | Eksekusi kustom query SQL |
| `POST` | `/api/v1/me/databases/:database_id/explain` | `view_only` | Analisis query plan (`EXPLAIN`) |
| `POST` | `/api/v1/me/databases/:database_id/explain-analyze` | `view_only` | Analisis query plan aktual (`EXPLAIN ANALYZE`) |

### 4. Manajemen Migrasi & Draft Konsensus (`/me/databases` & `/me/drafts`)
| Method | Endpoint | Hak Akses | Deskripsi |
|---|---|---|---|
| `GET` | `/api/v1/me/databases/:database_id/migrations` | `view_only` | Riwayat versi migrasi database |
| `GET` | `/api/v1/me/databases/:database_id/migrations/:id` | `view_only` | Detail satu versi migrasi |
| `GET` | `/api/v1/me/databases/:database_id/migrations/compare` | `view_only` | Perbandingan dua versi migrasi (*Schema Diff*) |
| `POST` | `/api/v1/me/databases/:database_id/migrations` | `view_edit` | Eksekusi langsung migrasi skema |
| `POST` | `/api/v1/me/databases/:database_id/drafts` | `view_edit` | Ajukan proposal draft migrasi baru |
| `GET` | `/api/v1/me/databases/:database_id/drafts` | `view_only` | Daftar proposal draft pada database |
| `POST` | `/api/v1/me/databases/:database_id/rollback-draft` | `view_edit` | Buat draft proposal untuk rollback |
| `POST` | `/api/v1/me/databases/:database_id/forward-draft` | `view_edit` | Buat draft proposal untuk re-apply / forward |
| `POST` | `/api/v1/me/databases/:database_id/migration-drafts/:draft_id/vote` | Leader | Berikan suara (*approve* / *reject*) pada proposal draft |
| `POST` | `/api/v1/me/databases/:database_id/migration-drafts/:draft_id/execute` | `view_edit` | Eksekusi draft yang telah disetujui konsensus |

### 5. Pemulihan Batch DML & Quarantine Recovery (`/me/drafts`)
| Method | Endpoint | Hak Akses | Deskripsi |
|---|---|---|---|
| `GET` | `/api/v1/me/drafts/:id` | `view_only` | Ambil detail status draft & isolasi karantina |
| `POST` | `/api/v1/me/drafts/:id/resume` | `view_edit` | Lanjutkan eksekusi batch DML yang sempat terhenti |
| `POST` | `/api/v1/me/drafts/:id/revert-batch` | `view_edit` | Batalkan dan pulihkan batch DML yang telah berjalan |
| `GET` | `/api/v1/me/drafts/:id/progress` | `view_only` | Ambil status progres eksekusi batch saat ini |
| `GET` | `/api/v1/me/drafts/:id/progress/stream` | `view_only` | Server-Sent Events (SSE) realtime stream progres batch |

---

## Kontributor

| NIM | Nama |
|---|---|
| 13524006 | Gabriella Botimada Lubis |
| 13524030 | Irvin Tandiarrang Sumual |
| 13524033 | Ray Owen Martin |
