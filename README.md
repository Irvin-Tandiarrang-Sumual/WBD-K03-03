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

Frontend dapat diakses melalui:

```text
http://localhost:3000
```

Untuk lingkungan development Docker, data demo dibuat otomatis. Akun yang
dapat digunakan untuk login adalah:

```text
Admin: admin / admin123
User:  user / user123
```

`demo_user` dan `demo_password` adalah kredensial database target development,
bukan akun login aplikasi.

Database demo PostgreSQL `db1_pg` dijalankan oleh Docker Compose dan berisi
tabel contoh `customers` serta `orders` untuk mencoba tampilan schema dan data.
Database MySQL `db2_mysql` juga dijalankan oleh Docker Compose dan berisi tabel
`products` serta `inventory_movements`.

Tiga koneksi tambahan tersedia di halaman Databases untuk pengujian frontend:

- `db3_pg_empty`: PostgreSQL valid tanpa user table untuk menguji empty state.
- `db4_pg_sales`: PostgreSQL berisi tabel customer dan order penjualan.
- `db5_mysql_analytics`: MySQL berisi tabel event dan metrik harian.

Schema ketiganya dibuat oleh file SQL di folder `docker/postgres-connections`
dan `docker/mysql-connections` ketika container pertama kali dibuat.

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

## API Documentation (Swagger)

Aplikasi ini menyediakan dokumentasi API interaktif berbasis Swagger / OpenAPI untuk mempermudah eksplorasi dan pengujian endpoint backend.

### 1. Akses Swagger UI

Pastikan container backend atau server lokal sudah berjalan:
```bash
docker compose up
```

Buka URL berikut pada browser:
```text
http://localhost:8080/swagger/index.html
```

> **Catatan**: Spesifikasi mentah OpenAPI dalam format JSON juga dapat diakses langsung di:
> `http://localhost:8080/swagger/doc.json`

---

### 2. Cara Menguji Endpoint (Dasar)

1. **Pilih Endpoint**: Endpoint dikelompokkan berdasarkan kategori (seperti `auth`, `admin`, `databases`, `migrations`). Klik pada salah satu endpoint untuk melihat rinciannya.
2. **Mulai Testing**: Klik tombol **Try it out** di pojok kanan atas blok endpoint tersebut.
3. **Isi Parameter / Body**: Masukkan parameter query, path parameter, atau request body JSON sesuai contoh skema yang tertera.
4. **Eksekusi**: Klik tombol **Execute** (berwarna biru).
5. **Periksa Respons**: Hasil request akan ditampilkan di bagian **Responses**, meliputi:
   - **HTTP Status Code** (misal `200 OK`, `201 Created`, `400 Bad Request`, `401 Unauthorized`).
   - **Response Body** (JSON respons data / pesan error).
   - **Response Headers**.

---

### 3. Panduan Autentikasi (Cookie-Based Authentication)

Endpoint selain `POST /api/v1/auth/login` memerlukan sesi login aktif menggunakan **Cookie-based authentication** (`session=<token>`). Ikuti panduan langkah demi langkah di bawah ini untuk mengautentikasi Swagger UI:

#### Akun Demo yang Tersedia
- **Admin** (memiliki izin penuh ke seluruh endpoint `/admin/*` dan `/me/*`):
  - **Username**: `admin`
  - **Password**: `admin123`
- **User Biasa** (akses terbatas ke endpoint `/me/*`):
  - **Username**: `user`
  - **Password**: `user123`

#### Langkah-Langkah Autentikasi:
1. **Lakukan Login via Swagger**:
   - Expand endpoint `POST /api/v1/auth/login`.
   - Klik **Try it out**.
   - Masukkan kredensial pada Request body:
     ```json
     {
       "username": "admin",
       "password": "admin123"
     }
     ```
   - Klik **Execute** dan pastikan status respons `200 OK`.

2. **Dapatkan Token Sesi**:
   Ambil string token sesi dari cookie browser melalui salah satu cara berikut:
   - **Opsi A (Melalui DevTools Storage / Cookies — Disarankan)**:
     1. Buka Developer Tools browser (`F12` atau klik kanan → *Inspect*).
     2. Buka tab **Application** (Chrome/Edge/Brave) atau **Storage** (Firefox).
     3. Pada panel kiri, expand menu **Cookies** → klik `http://localhost:8080`.
     4. Salin (copy) teks nilai pada kolom **Value** dari cookie dengan nama `session`.
   - **Opsi B (Melalui DevTools Network)**:
     1. Buka tab **Network** pada Developer Tools.
     2. Klik request `login` yang baru saja dieksekusi.
     3. Pada panel kanan, lihat bagian **Response Headers**.
     4. Cari header `Set-Cookie:` yang berisi format `session=<TOKEN>; ...` dan salin token tersebut.

3. **Pasang Token ke Swagger UI (Authorize)**:
   - Gulir ke paling atas halaman Swagger UI, lalu klik tombol 🔒 **Authorize** (di pojok kanan atas).
   - Pada modal dialog **CookieAuth (apiKey)**, ketikkan nilai dengan format:
     ```text
     session=<TOKEN>
     ```
     *(Penting: Harus menyertakan awalan `session=` diikuti nilai token yang sudah disalin)*
   - Klik tombol **Authorize**, lalu klik **Close**.
   - Ikon gembok 🔒 pada endpoint yang terproteksi sekarang akan berstatus terkunci/aktif. Semua request berikutnya akan otomatis membawa header cookie tersebut.

4. **Logout**:
   - Untuk mengakhiri sesi, eksekusi endpoint `POST /api/v1/auth/logout`.
   - Buka kembali modal 🔒 **Authorize** di pojok kanan atas, lalu klik tombol **Logout** untuk menghapus token dari Swagger UI.

---

### 4. Update documentation

Jika Anda melakukan perubahan pada route handler, struct request/response, atau menambahkan anotasi Swagger baru di backend:

1. Pastikan CLI `swag` sudah terpasang:
   ```bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ```

2. Jalankan perintah pembuatan dokumentasi dari dalam direktori `backend/`:
   ```bash
   cd backend
   swag init -g cmd/api/main.go -o docs/ --parseDependency --parseInternal
   ```

3. File `docs.go`, `swagger.json`, dan `swagger.yaml` di dalam folder `backend/docs/` akan diperbarui secara otomatis. Refresh browser Anda untuk melihat perubahan terbaru pada Swagger UI.

## Unit Testing

Unit test backend dapat dijalankan dengan satu command dari root project:

```bash
make test
```

Target `test` pada [Makefile](./Makefile) root berpindah ke folder `backend`
dan menjalankan seluruh unit test Go dengan:

```text
go test ./...
```

Integration test yang membutuhkan PostgreSQL eksternal tidak termasuk dalam
command unit test. Untuk menjalankan seluruh unit test dan integration test,
gunakan:

```bash
make test-all
```

Target lain yang tersedia:

```bash
make test-unit        # Sama dengan make test
make test-unit-cover  # Unit test dengan laporan coverage
make test-integration # Integration test
```

Target `test-unit-cover` menjalankan package yang memiliki unit test dan
menghasilkan laporan coverage di `backend/coverage.out`. Laporan tersebut juga
menampilkan total coverage seluruh package yang dipilih.

### Run with Docker

From the repository root, copy `.env.example` to `.env`:
```bash
Copy-Item .env.example .env
```
then run:

```bash
docker compose up --build
```

The frontend will be available at `http://localhost:3000` by default.

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
| POST | `api/v1/admin/groups` | Buat group baru | Body: `name` dan `databases`, setiap database memiliki `database_id` dan `access_type` |
| PATCH | `api/v1/admin/groups/:id` | Update sebagian data group | Body dapat berisi `name` atau `databases`, setiap database memiliki `database_id` dan `access_type` |
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
