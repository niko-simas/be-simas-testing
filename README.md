# SIMAS Backend

Backend API untuk Sistem Informasi Manajemen Siswa.

## Tech Stack

Go 1.26 · Gin · GORM · PostgreSQL · Redis · JWT · bcrypt · SMTP

## Struktur Direktori

```
cmd/api/                   Entrypoint
internal/
  auth/                    JWT, Google OAuth, password & OTP helper
  config/                  Load environment variables
  handler/                 HTTP handler
  infrastructure/mailer/   SMTP sender + template email
  middleware/              Auth, rate limit, logger
  model/                   Struct user & DTO
  repository/              Query database (GORM)
  service/                 Business logic
migrations/                SQL migration
postman/                   Collection & environment Postman
markdown/                  Dokumentasi planning & spesifikasi MVP
```

## Setup

DB_HOST=localhost
DB_PORT=5432
DB_NAME=simas_db
DB_USERNAME=postgres
DB_PASSWORD=

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

JWT_SECRET=

MAIL_HOST=
MAIL_PORT=587
MAIL_USERNAME=
MAIL_PASSWORD=
MAIL_FROM=
MAIL_FROM_NAME=SIMAS
MAIL_ENCRYPTION=tls

WEB_GOOGLE_CLIENT_ID=
MOBILE_GOOGLE_CLIENT_ID=
ANDROID_PACKAGE_NAME=
```

**1. Jalankan**

```bash
go run ./cmd/api
```

Server aktif di `http://localhost:8080`. Migration berjalan otomatis saat startup.

## Endpoint

Base URL: `/api/v1`

### Auth Mobile

| Method | Path | Auth | Keterangan |
|---|---|---|---|
| POST | `/auth/mobile/register` | - | Daftar akun baru, kirim OTP ke email |
| POST | `/auth/mobile/verify-otp` | - | Verifikasi kode OTP |
| POST | `/auth/mobile/resend-otp` | - | Kirim ulang OTP |
| POST | `/auth/mobile/login` | - | Login email + password |
| POST | `/auth/mobile/oauth` | - | Login via Google ID token |
| POST | `/auth/mobile/oauth/complete` | - | Lengkapi nomor WhatsApp setelah OAuth |
| POST | `/api/v1/auth/refresh` | - | Rotasi access + refresh token |
| POST | `/auth/logout` | Bearer | Blacklist access token + revoke refresh token |
| GET | `/api/v1/mobile/status-verifikasi` | Bearer | Cek status onboarding user |

### Auth Internal (Admin Pusat, Sekolah, Pendidik)

| Method | Path | Auth | Keterangan |
|---|---|---|---|
| POST | `/auth/pusat/register` | Bearer super_admin | Daftar super_admin baru (hanya oleh super_admin yang sudah aktif) |
| POST | `/auth/pusat/login` | - | Login super_admin (tanpa force update) |
| POST | `/auth/internal/login` | - | Login admin_sekolah / tenaga_pendidik (dengan force update) |
| PUT | `/auth/internal/update-password` | Bearer temp_token | Update password wajib setelah login pertama |

**Perbedaan role:**
- `super_admin` (SIMAS) → login langsung dapat session (tidak ada force password update).
- `admin_sekolah` / `tenaga_pendidik` → login pertama balik `202` dengan `temp_token`, wajib update password via `PUT /auth/internal/update-password`.

### Super Admin Login

```
POST /auth/pusat/login
```

```json
{
  "email": "admin.pusat@simas.com",
  "password": "SimasAdmin2024!"
}
```

Response `200`:

```json
{
  "account_status": "active",
  "onboarding_status": "completed",
  "access_token": "<jwt>",
  "refresh_token": "<jwt>",
  "expires_in": 900,
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "admin.pusat@simas.com",
    "name": "Administrator Pusat",
    "role": "super_admin",
    "is_password_updated": true
  }
}
```

### Register Super Admin

Hanya super_admin yang sudah aktif dapat membuat super_admin baru. Endpoint dilindungi oleh middleware `RequireAuth` + `RequireRole("super_admin")`.

```
POST /auth/pusat/register
Authorization: Bearer <access_token_super_admin>
```

```json
{
  "email": "newadmin@simas.com",
  "name": "New Administrator",
  "password": "NewPass123!"
}
```

Response `201`:

```json
{
  "account_status": "active",
  "onboarding_status": "completed",
  "access_token": "<jwt>",
  "refresh_token": "<jwt>",
  "expires_in": 900,
  "user": {
    "id": "...",
    "email": "newadmin@simas.com",
    "name": "New Administrator",
    "role": "super_admin",
    "is_password_updated": true
  }
}
```

### Internal Login (Force Update)

```
POST /auth/internal/login
```

```json
{
  "email": "admin@sekolah.sch.id",
  "password": "DefaultPass123!"
}
```

**Login pertama** → Response `202`:

```json
{
  "account_state": "require_password_update",
  "temp_token": "<jwt-temp-token>",
  "expires_in": 900
}
```

**Setelah force update** → Response `200`: sama seperti super admin login.

### Force Update Password

```
PUT /auth/internal/update-password
Authorization: Bearer <temp_token>
```

```json
{
  "old_password": "DefaultPass123!",
  "new_password": "NewPass123!",
  "confirm_password": "NewPass123!"
}
```

Response `200`:

```json
{
  "account_status": "active",
  "access_token": "<jwt>",
  "refresh_token": "<jwt>",
  "expires_in": 900,
  "user": {
    "role": "admin_sekolah",
    "is_password_updated": true
  }
}
```

Catatan:
- `temp_token` berlaku 15 menit, hanya bisa digunakan di endpoint force-update.
- `temp_token` ditolak di endpoint operasional lain (`/auth/me`, `/mobile/...`).
- Setelah update password, `temp_token` dicabut otomatis.

### Register

```
POST /auth/mobile/register
```

```json
{
  "nama_lengkap": "Budi Santoso",
  "email": "budi@example.com",
  "no_whatsapp": "081234567890",
  "password": "Password123!"
}
```

Response `201`:

```json
{
  "message": "verification_code_sent",
  "email": "budi@example.com",
  "expires_in": 180
}
```

### Verify OTP

```
POST /auth/mobile/verify-otp
```

```json
{
  "email": "budi@example.com",
  "otp_code": "841285"
}
```

Response `200`:

```json
{
  "account_status": "active",
  "onboarding_status": "need_kyc",
  "access_token": "<jwt>",
  "refresh_token": "<jwt>",
  "expires_in": 900
}
```

### Login

```
POST /auth/mobile/login
```

```json
{
  "email": "budi@example.com",
  "password": "Password123!"
}
```

Response `200`: sama seperti verify OTP.

### Google OAuth

```
POST /auth/mobile/oauth
```

```json
{
  "google_token": "<google-id-token>"
}
```

User baru atau tanpa WhatsApp → `202`:

```json
{
  "account_state": "require_whatsapp",
  "temp_user_id": "<id>",
  "expires_in": 600
}
```

User existing → `200`: sama seperti login.

### Complete OAuth

```
POST /auth/mobile/oauth/complete
```

```json
{
  "temp_user_id": "<id>",
  "no_whatsapp": "081234567890"
}
```

Response `200`: sama seperti login.

### Refresh Token

```
POST /auth/refresh
```

```json
{
  "refresh_token": "<refresh-token>"
}
```

Response `200`: token baru, token lama dirotasi.

### Logout

```
POST /auth/logout
Authorization: Bearer <access-token>
```

Response `200`:

```json
{ "message": "logged_out" }
```

### Status Verifikasi

```
GET /mobile/status-verifikasi
Authorization: Bearer <access-token>
```

Response `200`:

```json
{
  "status": "need_kyc",
  "account_status": "active"
}
```

## Error Response

```json
{
  "code": "invalid_otp",
  "message": "Kode OTP salah atau sudah kedaluwarsa"
}
```

| Code | HTTP | Keterangan |
|---|---|---|
| `invalid_request` | 400 | Payload tidak valid |
| `weak_password` | 400 | Password tidak memenuhi syarat |
| `invalid_otp` | 400 | OTP salah atau expired |
| `invalid_whatsapp` | 400 | Format nomor WhatsApp salah |
| `invalid_session` | 400 | Temp OAuth session tidak valid |
| `invalid_credentials` | 401 | Email atau password salah |
| `invalid_google_token` | 401 | Google ID token tidak valid |
| `email_not_verified` | 403 | Email belum diverifikasi |
| `account_inactive` | 403 | Akun belum aktif |
| `forbidden` | 403 | Role tidak sesuai |
| `not_internal` | 403 | Akun tidak memiliki akses update password |
| `password_mismatch` | 400 | Konfirmasi password tidak cocok |
| `same_password` | 400 | Password baru sama dengan password lama |
| `invalid_temp_token` | 401 | Temp token tidak valid atau sudah kedaluwarsa |
| `email_already_exists` | 409 | Email sudah terdaftar |
| `resend_cooldown` | 429 | Cooldown resend aktif |
| `rate_limit_exceeded` | 429 | Terlalu banyak percobaan |

## Kebijakan

### Rate Limit

| Endpoint | Maks | Window |
|---|---|---|
| `register` | 5x | 15 menit |
| `verify_otp` | 10x | 10 menit |
| `resend_otp` | 3x | 10 menit |
| `login` | 10x | 15 menit |
| `oauth` | 10x | 15 menit |
| `refresh` | 20x | 15 menit |

Konfigurasi: `internal/middleware/rate_limit.go`

### OTP & Token

| Aturan | Nilai |
|---|---|
| Panjang OTP | 6 digit |
| OTP expired | 3 menit |
| Max percobaan OTP | 5x per kode |
| Cooldown resend | 60 detik |
| Access token | 15 menit |
| Refresh token | 7 hari |
| Temp token (force update) | 15 menit |
| OAuth pending session | 10 menit |

### Role

| Role | Keterangan |
|---|---|
| `super_admin` | Administrator platform SIMAS. Login normal, tidak ada force update. |
| `admin_sekolah` | Admin satuan pendidikan. Dibuat oleh sistem, wajib update password saat pertama login. |
| `tenaga_pendidik` | Guru / wali kelas. Dibuat oleh sistem, wajib update password saat pertama login. |
| `wali_murid` | Orang tua / wali murid. Daftar mandiri via mobile, tidak ada force update. |

### Password

Minimal 8 karakter, wajib mengandung huruf besar, huruf kecil, angka, dan karakter khusus.

## Database

### Tabel `users`

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID PK | |
| `email` | VARCHAR(255) UNIQUE | Lowercase |
| `name` | VARCHAR(255) | |
| `google_id` | VARCHAR(255) UNIQUE | NULL jika bukan OAuth |
| `avatar_url` | TEXT | |
| `provider` | VARCHAR(50) | `email` atau `google` |
| `password_hash` | TEXT | NULL untuk OAuth |
| `no_whatsapp` | VARCHAR(20) | Format `62xxx` |
| `role` | VARCHAR(50) | `wali_murid` |
| `account_status` | VARCHAR(40) | `pending_verification` / `active` |
| `onboarding_status` | VARCHAR(40) | `need_kyc` / `need_child` / `waiting_verification` / `active` |
| `email_verified` | BOOLEAN | |
| `email_verified_at` | TIMESTAMP | |
| `is_password_updated` | BOOLEAN | `true` untuk super_admin, `false` untuk akun internal baru |
| `last_login_at` | TIMESTAMP | |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

### Tabel `refresh_tokens`

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID PK | |
| `user_id` | UUID FK | Referensi `users.id` |
| `token_hash` | TEXT UNIQUE | SHA256 hash |
| `expires_at` | TIMESTAMP | |
| `revoked_at` | TIMESTAMP | NULL jika aktif |
| `replaced_by_id` | UUID | ID token pengganti |
| `created_at` | TIMESTAMP | |

## Status Implementasi

**Selesai**

- Register manual dengan OTP email
- Verify, resend OTP
- Login email/password
- Google OAuth mobile + WhatsApp completion
- Super admin login (tanpa force update)
- Internal login dengan force password update (admin_sekolah, tenaga_pendidik)
- Refresh token rotation
- Logout dengan Redis blacklist
- Status verifikasi
- Rate limiting per IP
- Password strength validation
- Role guard middleware
- HTML email template OTP
- Request ID dan structured logging
- Migration otomatis saat startup

**Belum**

- Unit test
- KYC (upload KTP dan foto wajah)
- Pencarian sekolah
- Submit data anak
- Verifikasi siswa oleh sekolah
- Dashboard presensi
- Modul sekolah (B2B)
- Modul pendidik

## Testing dengan Postman

Import collection dan environment berikut:

```
postman/simas-full-collection.json   → Full API Collection (Mobile + Internal + Utility)
postman/simas-local.env.json         → Environment variables (Base URL, token, dll)
```

Pilih environment **SIMAS — Local** sebelum menjalankan request.

### Folder Collection

| Folder | Keterangan |
|---|---|
| **MOBILE AUTH** | Registrasi Wali Murid (register, verify OTP, login, refresh, logout, OAuth, status) |
| **INTERNAL AUTH** | Autentikasi Internal (super_admin login, force password update flow) |
| **UTILITY** | Health check |

### Environment Variables

| Variable | Default | Keterangan |
|---|---|---|
| `base_url` | `http://localhost:8080` | Base URL API |
| `email` | `budi@example.com` | Email untuk test register mobile |
| `password` | `Password123!` | Password untuk test |
| `admin_email` | `admin.pusat@simas.com` | Email super_admin |
| `admin_password` | `SimasAdmin2024!` | Password super_admin |
| `internal_email` | `admin@sekolah.sch.id` | Email admin_sekolah / pendidik |
| `internal_password` | `DefaultPass123!` | Password internal awal |
| `access_token` | *(empty)* | Auto-populated setelah login |
| `refresh_token` | *(empty)* | Auto-populated setelah login |
| `temp_token` | *(empty)* | Auto-populated saat force update |
| `otp_code` | *(empty)* | Manual input dari email |

### Urutan Testing (Mobile Auth)

1. **Register** → Email terkirim, isi `otp_code` di environment
2. **Verify OTP** → Akun aktif, simpan token
3. **Refresh Token** → Rotasi token
4. **Logout** → Blacklist token
5. **Token Lama** → Harus 401 (revoked)

### Urutan Testing (Internal Auth — Force Update)

1. **Super Admin Login** → 200 + session langsung
2. **Internal Login** → 202 + `temp_token` (akun baru)
3. **Force Update Password** → 200 + session (setelah update)
4. **Temp Token Ditolak** → 401 di endpoint operasional

## Catatan

- Environment variables dan `firebase_sdk/` tidak masuk version control
- Google client secret hanya ada di backend
- OTP dan password tidak pernah muncul di log atau response
- Refresh token disimpan sebagai SHA256 hash di database
- Logo email: ganti `LOGO_URL` di `internal/infrastructure/mailer/template.go`
