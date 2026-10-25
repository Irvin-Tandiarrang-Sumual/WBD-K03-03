# Database Management Platform (WBD-K03-03)

Platform manajemen database berbasis web  untuk mengelola koneksi database secara dinamis, mengontrol hak akses pengguna berbasis grup (*Role-Based Access Control*), mengeksekusi dan membandingkan migrasi skema (*Schema Diff & Migration*), menganalisis rencana eksekusi query (*EXPLAIN / EXPLAIN ANALYZE*), serta melakukan *automated rollback*.

---
## Features
1. **Dynamic Database Creation**:
2. **Group-Based Access Control**:
3. **Schema Diff & Candidate Migration Generation**:
4. **Query Execution Plan Analysis (EXPLAIN / EXPLAIN ANALYZE)**:
5. **Automated Schema Rollback**:
---

## Tech Stack & Architecture

- **Backend**: Go 1.23, Gin Web Framework, GORM ORM, PostgreSQL Driver (`pgx`), MySQL Driver (`go-sql-driver/mysql`), Swaggo (OpenAPI/Swagger).
- **Frontend**: Next.js 15, React 19, TypeScript, CSS Modules.
- **Database Storage**:
  - `internal_db`: PostgreSQL 15 untuk menyimpan data pengguna, grup, metadata database, dan riwayat migrasi.
  - `demo_postgres`: Engine PostgreSQL 15 target demo.
  - `demo_mysql`: Engine MySQL 8.4 target demo.
- **Infrastructure**: Docker & Docker Compose.

---

## Usage

### 1. Prasyarat
- Docker Engine atau Docker Desktop aktif.
- Docker Compose v2+.

### 2. Konfigurasi Lingkungan (`.env`)
Salin file konfigurasi contoh di root direktori proyek:
```bash
cp .env.example .env
```
Pastikan variabel lingkungan seperti port dan kredensial database internal sudah sesuai.

### 3. Menjalankan Seluruh Service
Jalankan perintah berikut dari direktori root proyek:
```bash
docker compose up --build -d
```

Setelah seluruh kontainer berstatus *healthy*, aplikasi dapat diakses di:
- **Frontend Web App**: [http://localhost:3000](http://localhost:3000)
- **Backend API Server**: [http://localhost:8080](http://localhost:8080)
- **Interactive Swagger UI**: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

### 4. Akun Demo Bawaan
Sistem secara otomatis menginisialisasi akun awal untuk pengujian:

| Role | Username | Password | Deskripsi |
|---|---|---|---|
| **Admin** | `admin` | `admin123` | Memiliki akses penuh ke panel admin, user management, group access control, dan koneksi database. |
| **Regular User** | `user` | `user123` | Pengguna biasa dengan akses database yang dibatasi oleh grup keanggotaannya. |

> **Catatan Kredensial Engine Demo:**
> - PostgreSQL Demo: `demo_postgres:5432` (User: `demo_user`, Password: `demo_password`, Database: `db1_pg`).
> - MySQL Demo: `demo_mysql:3306` (User: `demo_user`, Password: `demo_password`, Database: `db2_mysql`).

### 5. Rebuild
Jika ada pembaruan kode pada backend atau frontend:
```bash
docker compose build --no-cache app frontend && docker compose up -d
```

### 6. Menghentikan Service
Untuk menghentikan kontainer:
```bash
docker compose down
```
Untuk menghentikan dan mereset seluruh volume data database:
```bash
docker compose down -v
```

---

## Testing

### 1. Menjalankan Unit Test Backend
Dari direktori root proyek:
```bash
make test
# atau langsung dari folder backend:
cd backend && go test ./...
```

Target Makefile lainnya yang tersedia:
```bash
make test-unit        # Menjalankan unit test
make test-unit-cover  # Menjalankan unit test dan menghasilkan laporan coverage di backend/coverage.out
make test-integration # Menjalankan integration test
make test-all         # Menjalankan seluruh unit test dan integration test
```

### 2. Menjalankan Typecheck Frontend
Dari direktori `frontend`:
```bash
cd frontend && npx tsc --noEmit
```

---

## 📖 Dokumentasi API & Swagger

Dokumentasi API interaktif tersedia melalui Swagger UI di:
```text
http://localhost:8080/swagger/index.html
```

### Panduan Autentikasi Swagger (Cookie-Based Authentication)
Endpoint selain `POST /api/v1/auth/login` memerlukan cookie sesi (`session=<token>`):
1. Buka endpoint `POST /api/v1/auth/login` di Swagger UI.
2. Klik **Try it out**, masukkan username `admin` dan password `admin123`, lalu klik **Execute**.
3. Buka Developer Tools browser (`F12`), masuk ke tab **Application / Storage** → **Cookies** → salin nilai (*Value*) dari cookie bernama `session`.
4. Klik tombol **Authorize** di pojok kanan atas halaman Swagger UI.
5. Masukkan nilai:
   ```text
   session=<TOKEN_SESI_ANDA>
   ```
6. Klik **Authorize** lalu **Close**. Seluruh endpoint terproteksi kini dapat diuji langsung dari Swagger.

### Memperbarui Swagger Doc
Jika menambahkan endpoint atau mengubah struct request/response:
```bash
cd backend
swag init -g cmd/api/main.go -o docs/ --parseDependency --parseInternal
```

---

## 📡 Daftar Endpoint API

### 1. Autentikasi & Akun (`/auth`)
| Method | Endpoint | Deskripsi | Parameter / Body |
|---|---|---|---|
| `POST` | `/api/v1/auth/login` | Login pengguna dan inisialisasi cookie sesi | `{ "username": "...", "password": "..." }` |
| `POST` | `/api/v1/auth/logout` | Mengakhiri sesi pengguna aktif | — |
| `GET` | `/api/v1/auth/me` | Mengambil data profil user yang sedang login | — |

### 2. Panel Admin: Manajemen Pengguna (`/admin/users`)
| Method | Endpoint | Deskripsi | Catatan |
|---|---|---|---|
| `GET` | `/api/v1/admin/users` | Mengambil daftar seluruh pengguna (paginated) | Query: `page`, `limit` |
| `GET` | `/api/v1/admin/users/:userId` | Mengambil detail satu pengguna berdasarkan ID | Path: `:userId` |
| `POST` | `/api/v1/admin/users` | Membuat pengguna baru | Body: `username`, `password`, `groupIds` |
| `PATCH` | `/api/v1/admin/users/:userId` | Memperbarui data pengguna | Body: `username`, `password` |
| `DELETE` | `/api/v1/admin/users/:userId` | Menghapus pengguna | Path: `:userId` |
| `PATCH` | `/api/v1/admin/users/:userId/role` | Mengubah role pengguna (`admin` / `user`) | Body: `role` |

### 3. Panel Admin: Manajemen Grup & Access Control (`/admin/groups`)
| Method | Endpoint | Deskripsi | Catatan |
|---|---|---|---|
| `GET` | `/api/v1/admin/groups` | Mengambil daftar seluruh access group | Query: `page`, `limit` |
| `GET` | `/api/v1/admin/groups/:groupId` | Mengambil detail grup beserta daftar hak akses database | Path: `:groupId` |
| `POST` | `/api/v1/admin/groups` | Membuat grup baru | Body: `name`, `databaseAccesses` |
| `PATCH` | `/api/v1/admin/groups/:groupId` | Mengubah nama grup atau hak akses database (`view_only` $\leftrightarrow$ `view_edit`) | Body: `name`, `databaseAccesses` |
| `DELETE` | `/api/v1/admin/groups/:groupId` | Menghapus access group | Path: `:groupId` |
| `POST` | `/api/v1/admin/groups/:groupId/users/:userId` | Memasukkan user ke dalam grup | Path: `:groupId`, `:userId` |
| `DELETE` | `/api/v1/admin/groups/:groupId/users/:userId` | Mengeluarkan user dari grup | Path: `:groupId`, `:userId` |

### 4. Panel Admin: Manajemen Database Server (`/admin/databases`)
| Method | Endpoint | Deskripsi | Catatan |
|---|---|---|---|
| `GET` | `/api/v1/admin/databases` | Mengambil seluruh database yang terdaftar | Query: `page`, `limit` |
| `GET` | `/api/v1/admin/databases/:id` | Mengambil detail database berdasarkan ID | Path: `:id` |
| `POST` | `/api/v1/admin/databases` | Membuat database baru secara dinamis pada engine target | Body: `host`, `port`, `user`, `password`, `type`, `name` |
| `DELETE` | `/api/v1/admin/databases/:id` | Menghapus database terdaftar (dicegah jika masih memiliki grup yang terhubung) | Path: `:id` |

### 5. Pengguna: Eksplorasi Database, Query & Migrasi (`/me/databases`)
| Method | Endpoint | Deskripsi | Hak Akses Minimal |
|---|---|---|---|
| `GET` | `/api/v1/me/databases` | Mengambil daftar database yang dapat diakses pengguna | — |
| `GET` | `/api/v1/me/databases/:database_id` | Mengambil detail database | `view_only` |
| `POST` | `/api/v1/me/databases/:database_id/query` | Menjalankan query SQL kustom | `view_only` |
| `POST` | `/api/v1/me/databases/:database_id/explain` | Menganalisis rencana eksekusi query (`EXPLAIN` / `EXPLAIN ANALYZE`) | `view_only` |
| `GET` | `/api/v1/me/databases/:database_id/migrations` | Mengambil daftar riwayat versi migrasi skema | `view_only` |
| `POST` | `/api/v1/me/databases/:database_id/migrations` | Mengeksekusi migrasi skema baru | `view_edit` |
| `GET` | `/api/v1/me/databases/:database_id/migrations/compare` | Membandingkan dua versi migrasi (Schema Diff) | `view_only` |
| `GET` | `/api/v1/me/databases/:database_id/migrations/:migration_id` | Mengambil detail satu versi migrasi | `view_only` |
| `POST` | `/api/v1/me/databases/migrations/rollback` | Melakukan rollback skema ke versi target | `view_edit` |

---

## 👥 Kontributor

| NIM | Nama |
|---|---|
| 13524006 | Gabriella Botimada Lubis |
| 13524030 | Irvin Tandiarrang Sumual |
| 13524033 | Ray Owen Martin |
