# Database Management Platform

## Description

## Usage

### Prerequisites

- Docker Engine atau Docker Desktop
- Docker Compose

### Configuration

Make sure `.env` ada di root:

```text
WBD-K03-03/
├── .env
├── docker-compose.yml
├── backend/
└── frontend/
```

Sesuai nilai konfigurasi database dan server di `.env` sebelum menjalankan
aplikasi. Opsi lain adalah dengan

```bash
cp .env.example .env
```

### Start App

Jalankan perintah berikut dari root project:

```bash
docker compose up --build
```

Backend dapat diakses melalui:

```text
http://localhost:8080
```

Jika ingin membangun ulang image tanpa menggunakan cache:

```bash
docker compose build --no-cache app
docker compose up
```

### Stop App

Untuk menghentikan container:

```bash
docker compose down
```

Untuk menghapus volume dan seluruh data database:
```bash
docker compose down -v
```

## API Endpoints
### Access Control
| Method | Endpoint | Function | Notes |
|---|---|---|---|
| POST | `api/v1/auth/login` | Mengautentikasi pengguna | Body : `username` dan `password` |
| POST | `api/v1/auth/logout` | Keluar dari web app | |

### Admin Platform
| Method | Endpoint | Function | Notes |
|---|---|---|---|
| GET | `api/v1/admin/users` | Ambil semua user |  |
| GET | `api/v1/admin/users/:id` | Ambil detail satu user | |
| POST | `api/v1/admin/users` | Buat user baru | Body: `username`, `password`, `role`, dan `groups` |
| DELETE | `api/v1/admin/users/:id` | Hapus satu user | Wajib menyertakan `:id` |
| GET | `api/v1/admin/groups` | Ambil semua group |  |
| GET | `api/v1/admin/groups/:id` | Ambil detail satu group | |
| POST | `api/v1/admin/groups` | Buat group baru | Body: `database_ids` berisi ID database yang terhubung, `access_type` juga |
| PATCH | `api/v1/admin/groups/:id` | Update sebagian data group | Body dapat berisi `access_type` baru (`view_only` atau `view_edit`) |
| DELETE | `api/v1/admin/groups/:id` | Hapus satu group | Wajib menyertakan `:id` |
| POST | `api/v1/admin/groups/:group_id/users` | Hubungkan user ke group | Kirim `user_id` dan `group_id` di request body |
| DELETE | `api/v1/admin/groups/:group_id/users/:user_id` | Putus hubungan user-group |  |
| GET | `api/v1/admin/databases` | Ambil semua database |  |
| GET | `api/v1/admin/databases/:id` | Ambil satu database | Menggunakan parameter `:id` |
| POST | `api/v1/admin/databases` | Buat database baru | Body: `port`, `host`, `user`, `password`, `type`, dan `name` |
| DELETE | `api/v1/admin/databases/:id` | Hapus satu database | Memperbaiki typo dan menambahkan `:id` |

### Database Management & Migration
| Method | Endpoint | Function | Notes / Features |
|---|---|---|---|
| GET | `api/v1/me/databases` | Daftar database | Menampilkan semua database yang dapat diakses user |
| GET | `api/v1/me/databases/:database_id` | Detail database | Menampilkan detail database dan isi datanya |
| POST | `api/v1/me/databases/:database_id/query` | Menjalankan SQL `SELECT` | request body : `SQL SELECT` |
| POST | `api/v1/me/databases/:database_id/explain` | Menganalisis query plan | Gunakan query parameter `analyzer=true` untuk menjalankan `EXPLAIN ANALYZE` |
| GET | `api/v1/me/databases/:database_id/migrations` | Daftar migrasi | Menampilkan riwayat atau daftar file migrasi |
| POST | `api/v1/me/databases/:database_id/migrations` | Menjalankan migrasi baru | Membuat dan mengeksekusi migrasi baru pada database |
| POST | `api/v1/me/databases/migrations/rollback` | Melakukan rollback migrasi | Mengembalikan database ke versi migrasi yang dipilih |
| GET | `api/v1/me/databases/:database_id/migrations/compare` | Membandingkan dua migrasi | Gunakan query parameter `migration_id_1` dan `migration_id_2` untuk menentukan dua migrasi yang dibandingkan |
| GET | `api/v1/me/databases/:database_id/migrations/:migration_id` | Detail migrasi | Menampilkan detail satu file migrasi |


## Contributors
| NIM | NIM |
|--- | ---|
| 13524006 | Gabriella Botimada Lubis |
| 13524030 | Irvin Tandiarrang Sumual |
| 13524033 | Ray Owen Martin |