# Dokumentasi Lengkap SIMAS API — Registrasi Sekolah & Alur Verifikasi

> Dokumen ini menjelaskan alur lengkap registrasi sekolah, verifikasi oleh Super Admin, tracking progress pengajuan, dan referensi seluruh endpoint API terkait.

---

## Daftar Isi

1. [Ringkasan Sistem](#1-ringkasan-sistem)
2. [State Machine Status Sekolah](#2-state-machine-status-sekolah)
3. [Alur Registrasi Sekolah (End-to-End)](#3-alur-registrasi-sekolah-end-to-end)
4. [Spesifikasi Upload File](#4-spesifikasi-upload-file)
5. [Referensi API Endpoint](#5-referensi-api-endpoint)
6. [Autentikasi & Otorisasi](#6-autentikasi--otorisasi)
7. [Collection Postman](#7-collection-postman)
8. [Kode Error & Handling](#8-kode-error--handling)

---

## 1. Ringkasan Sistem

SIMAS (Sistem Informasi Manajemen Sekolah) menyediakan alur **2-step registration** untuk sekolah baru:

| Tahap | Aksi User | Endpoint | Auth |
|-------|-----------|----------|------|
| **Step 1** | Isi data sekolah (JSON) | `POST /api/v1/sekolah` | ❌ Public |
| **Step 2** | Upload 4 dokumen legalitas (multipart/form-data) | `POST /api/v1/sekolah/{id}/dokumen` | ❌ Public |
| **Step 3** | Super Admin verifikasi (Setuju / Tolak) | `PUT /api/v1/sekolah/{id}/verifikasi` | ✅ Bearer (super_admin) |
| **Tracking** | Cek progress kapan saja | `GET /api/v1/sekolah/progress?tracking_code=...` | ❌ Public |

Setelah pengajuan **diterima** (status = `aktif`), sistem secara otomatis:
1. Membuat akun `admin_sekolah`
2. Mengirimkan kredensial login via email ke sekolah
3. Sekolah dapat login melalui `POST /auth/internal/login`

---

## 2. State Machine Status Sekolah

```
┌─────────────┐     upload dokumen      ┌─────────────────────┐
│  pengajuan  │ ──────────────────────▶ │ menunggu_verifikasi │
└─────────────┘                         └─────────────────────┘
                                               │
                                               │ Super Admin verifikasi
                          ┌────────────────────┼────────────────────┐
                          │ Setuju (aktif)     │                    │ Tolak (ditolak)
                          ▼                    │                    ▼
                   ┌─────────────┐             │            ┌─────────────┐
                   │    aktif    │             │            │   ditolak   │
                   └─────────────┘             │            └─────────────┘
                          │                    │                    │
                          │ nonaktif           │                    │ pengajuan ulang
                          ▼                    │                    ▼
                   ┌─────────────┐             │            ┌─────────────┐
                   │  nonaktif   │ ────────────┘            │  pengajuan  │
                   └─────────────┘                          └─────────────┘
```

| Status | Label | Keterangan |
|--------|-------|------------|
| `pengajuan` | Pengajuan Diterima | Data sekolah sudah tersimpan, menunggu upload dokumen |
| `menunggu_verifikasi` | Menunggu Verifikasi Admin | 4 dokumen legalitas sudah diupload, menunggu review Super Admin |
| `aktif` | Aktif | Pengajuan **diterima**. Akun admin_sekolah dibuat & kredensial dikirim via email |
| `ditolak` | Ditolak | Pengajuan **ditolak**. Admin dapat mengajukan ulang. |
| `nonaktif` | Nonaktif | Sekolah yang sudah aktif dinonaktifkan (jarang digunakan) |

---

## 3. Alur Registrasi Sekolah (End-to-End)

### Step 1 — Pengisian Data Sekolah

User mengisi form data sekolah di halaman awal registrasi.

**Endpoint:** `POST /api/v1/sekolah`
**Content-Type:** `application/json`
**Auth:** Tidak diperlukan

**Request Body:**
```json
{
  "nama": "SMA Negeri 1 Jakarta",
  "npsn": "20123456",
  "jenjang": "sma_smk_ma_mak",
  "alamat": "Jl. Sudirman No. 1",
  "provinsi_id": 31,
  "kabupaten_id": 3173,
  "kecamatan_id": 3173010,
  "desa_id": 3173010001,
  "kode_pos": "10210",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "email": "sman1jakarta@example.com",
  "telepon": "021-1234567",
  "website": "https://sman1jakarta.sch.id",
  "yayasan": "Yayasan Pendidikan Jakarta",
  "kepala_sekolah_nama": "Dr. Budi Santoso, M.Pd.",
  "kepala_sekolah_nip": "196512311990011001",
  "tanggal_berdiri": "1980-07-01",
  "sk_pendirian": "SK.123/1980"
}
```

**Response 201 Created:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tracking_code": "SMAB3C9D2E1",
  "status": "pengajuan",
  "status_label": "Pengajuan Diterima",
  "message": "Simpan tracking_code untuk melihat progress pengajuan. Silakan upload dokumen legalitas di endpoint /sekolah/{id}/dokumen"
}
```

> ⚠️ **PENTING:** Simpan `tracking_code` untuk tracking progress. Tidak ada autentikasi untuk tracking.

---

### Step 2 — Upload Dokumen Legalitas

User melanjutkan ke halaman berikutnya untuk upload 4 dokumen wajib.

**Endpoint:** `POST /api/v1/sekolah/{sekolah_id}/dokumen`
**Content-Type:** `multipart/form-data`
**Auth:** Tidak diperlukan

**Form Fields:**

| Field | Type | Deskripsi | Wajib |
|-------|------|-----------|-------|
| `akta_pendirian` | file | Akta Pendirian Yayasan/Badan Hukum (PDF) | ✅ |
| `nib` | file | Nomor Induk Berusaha (PDF) | ✅ |
| `sk_pendirian` | file | SK Pendirian Sekolah (PDF) | ✅ |
| `siop` | file | Surat Izin Operasional (PDF) | ✅ |

**Response 200 OK:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tracking_code": "SMAB3C9D2E1",
  "status": "menunggu_verifikasi",
  "status_label": "Menunggu Verifikasi Admin",
  "message": "Dokumen berhasil diupload. Pengajuan sedang diverifikasi oleh admin SIMAS."
}
```

Setelah upload berhasil:
- Status otomatis berubah dari `pengajuan` → `menunggu_verifikasi`
- Email notifikasi dikirim ke semua Super Admin
- State log tercatat otomatis

---

### Step 3 — Tracking Progress (Public)

User dapat mengecek status pengajuan kapan saja menggunakan `tracking_code`.

**Endpoint:** `GET /api/v1/sekolah/progress?tracking_code={tracking_code}`
**Auth:** Tidak diperlukan

**Response 200 OK:**
```json
{
  "tracking_code": "SMAB3C9D2E1",
  "nama": "SMA Negeri 1 Jakarta",
  "status": "menunggu_verifikasi",
  "status_label": "Menunggu Verifikasi Admin",
  "email": "sman1jakarta@example.com",
  "created_at": "2024-01-15T08:30:00Z",
  "updated_at": "2024-01-15T09:15:00Z",
  "state_logs": [
    {
      "status_baru": "pengajuan",
      "status_label": "Pengajuan Diterima",
      "catatan": null,
      "created_at": "2024-01-15T08:30:00Z"
    },
    {
      "status_baru": "menunggu_verifikasi",
      "status_label": "Menunggu Verifikasi Admin",
      "catatan": "Dokumen legalitas telah diupload",
      "created_at": "2024-01-15T09:15:00Z"
    }
  ]
}
```

---

### Step 4 — Verifikasi oleh Super Admin

Super Admin login, melihat list sekolah yang menunggu verifikasi, dan memutuskan.

#### A. Setuju (Terima)

**Endpoint:** `PUT /api/v1/sekolah/{sekolah_id}/verifikasi`
**Auth:** `Bearer {super_admin_token}`

**Request Body:**
```json
{
  "status_baru": "aktif",
  "catatan": "Dokumen lengkap dan valid"
}
```

**Yang terjadi setelah disetujui:**
1. Status berubah ke `aktif`
2. Sistem auto-generate password random untuk `admin_sekolah`
3. Akun user dengan role `admin_sekolah` dibuat
4. Email kredensial login dikirim ke email sekolah
5. State log dicatat

#### B. Tolak

**Request Body:**
```json
{
  "status_baru": "ditolak",
  "catatan": "NPSN tidak terdaftar di Dapodik"
}
```

**Yang terjadi setelah ditolak:**
1. Status berubah ke `ditolak`
2. Email penolakan dikirim ke sekolah berisi alasan
3. Sekolah dapat mengajukan ulang nanti

---

### Step 5 — Pengajuan Ulang (Jika Ditolak)

Admin Sekolah login, melihat sekolahnya ditolak, dan mengajukan ulang.

**Endpoint:** `POST /api/v1/sekolah/saya/pengajuan-ulang`
**Auth:** `Bearer {admin_sekolah_token}`

**Request Body:** (sama dengan `SekolahRequest` — update data jika perlu)
```json
{
  "nama": "SMA Negeri 1 Jakarta (Revisi)",
  "npsn": "20123457",
  "jenjang": "sma_smk_ma_mak",
  "email": "sman1jakarta@example.com",
  ...
}
```

**Ketentuan:**
- Hanya boleh diajukan ulang jika status sebelumnya = `ditolak`
- Setelah pengajuan ulang, status kembali ke `pengajuan`
- Admin harus upload ulang dokumen legalitas via `POST /api/v1/sekolah/{id}/dokumen`

---

### Step 6 — Sekolah Aktif (Operational)

Setelah status `aktif`, admin sekolah dapat:

1. **Login** menggunakan kredensial yang dikirim via email
2. **Akses data sekolah** via `GET /api/v1/sekolah/saya`
3. **Update data sekolah** via `PUT /api/v1/sekolah/saya`
4. **Kelola tenaga pendidik** (CRUD + jadwal mengajar)

---

## 4. Spesifikasi Upload File

### Ketentuan File

| Aturan | Detail |
|--------|--------|
| **Format** | PDF only (header file harus diawali `%PDF`) |
| **Ukuran Maksimal** | 3 MB per file (`3 * 1024 * 1024` bytes) |
| **Jumlah File** | 4 file wajib (akta_pendirian, nib, sk_pendirian, siop) |
| **Body Type** | `multipart/form-data` |
| **Storage** | Upload langsung ke S3/MinIO via backend |

### Cara Upload di Postman

1. Pilih method `POST`
2. Pilih tab **Body** → pilih **form-data**
3. Tambahkan 4 key dengan **type = File**:
   - `akta_pendirian` → pilih file PDF
   - `nib` → pilih file PDF
   - `sk_pendirian` → pilih file PDF
   - `siop` → pilih file PDF
4. Tidak perlu set `Content-Type` header secara manual — Postman akan otomatis set `multipart/form-data` dengan boundary

### Error Upload

| Error | Kode Response | Penyebab |
|-------|--------------|----------|
| `incomplete_legalitas` | 400 | Salah satu dari 4 file tidak diupload |
| `file_too_large` | 400 | File melebihi 3 MB |
| `invalid_file_format` | 400 | File bukan PDF (header tidak dimulai `%PDF`) |

---

## 5. Referensi API Endpoint

### Public Endpoints (No Auth)

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| `POST` | `/api/v1/sekolah` | Step 1: Pengajuan baru sekolah (data JSON) |
| `POST` | `/api/v1/sekolah/{id}/dokumen` | Step 2: Upload dokumen legalitas (multipart) |
| `GET` | `/api/v1/sekolah/progress?tracking_code=` | Tracking status pengajuan |
| `GET` | `/api/v1/sekolah/jenjang` | List master data jenjang pendidikan |

### Super Admin Endpoints (Bearer + role: super_admin)

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| `GET` | `/api/v1/sekolah` | List semua sekolah (filter status, search, pagination) |
| `GET` | `/api/v1/sekolah/{id}` | Detail sekolah + state log history |
| `GET` | `/api/v1/sekolah/{id}/dokumen-legalitas` | Generate presigned URL untuk preview/download dokumen |
| `PUT` | `/api/v1/sekolah/{id}/verifikasi` | Setuju / Tolak pengajuan sekolah |
| `PUT` | `/api/v1/sekolah/{id}` | Update data sekolah |

### Admin Sekolah Endpoints (Bearer + role: admin_sekolah)

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| `GET` | `/api/v1/sekolah/saya` | Lihat data sekolah sendiri |
| `PUT` | `/api/v1/sekolah/saya` | Update data dasar sekolah |
| `POST` | `/api/v1/sekolah/saya/pengajuan-ulang` | Ajukan ulang setelah ditolak |

### Internal Auth Endpoints

| Method | Endpoint | Deskripsi | Auth |
|--------|----------|-----------|------|
| `POST` | `/auth/pusat/login` | Login Super Admin | ❌ |
| `POST` | `/auth/pusat/register` | Register Super Admin baru | ✅ super_admin |
| `POST` | `/auth/internal/login` | Login Admin Sekolah / Pendidik | ❌ |
| `PUT` | `/auth/internal/update-password` | Force update password (first login) | ✅ temp_token |

### Mobile Auth Endpoints (Wali Murid)

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| `POST` | `/api/v1/auth/mobile/register` | Register dengan email + password + WhatsApp |
| `POST` | `/api/v1/auth/mobile/verify-otp` | Verifikasi email via OTP |
| `POST` | `/api/v1/auth/mobile/resend-otp` | Kirim ulang OTP |
| `POST` | `/api/v1/auth/mobile/login` | Login setelah verifikasi |
| `POST` | `/api/v1/auth/mobile/oauth` | Google OAuth login |
| `POST` | `/api/v1/auth/mobile/oauth/complete` | Lengkapi data setelah OAuth (WhatsApp) |

---

## 6. Autentikasi & Otorisasi

### Role System

| Role | Akses |
|------|-------|
| `super_admin` | Full access: verifikasi sekolah, register admin baru, lihat semua data |
| `admin_sekolah` | Kelola sekolah sendiri + kelola pendidik + jadwal |
| `tenaga_pendidik` | Lihat profile sendiri + jadwal mengajar |
| `wali_murid` | Mobile app: status verifikasi, data anak |

### Token Flow

```
1. Login → dapat access_token (15 menit) + refresh_token (7 hari)
2. Kirim access_token di header: Authorization: Bearer {token}
3. Jika access_token expired → refresh via POST /api/v1/auth/refresh
4. Jika refresh_token invalid/expired → login ulang
```

### Force Update Password (Internal Auth)

Karyawan/admin sekolah yang baru dibuat oleh Super Admin **wajib** mengganti password saat pertama kali login:

```
1. POST /auth/internal/login → Response 403 + force_update: true + temp_token
2. PUT /auth/internal/update-password (pakai temp_token) → Response 200 + token baru
3. Login ulang dengan password baru → Response 200 + token normal
```

---

## 7. Collection Postman

File: `postman/simas-full-collection.json`

### Struktur Folder

```
SIMAS — Full API Collection
├── MOBILE AUTH (9 requests)
│   ├── 1. Register
│   ├── 2. Verify OTP
│   ├── 3. Resend OTP
│   ├── 4. Login Setelah Verifikasi
│   ├── 5. Refresh Token
│   ├── 6. Logout
│   ├── 7. Status Verifikasi
│   ├── 8. Google OAuth
│   └── 9. Complete OAuth (WA)
│
├── INTERNAL AUTH (8 requests)
│   ├── 0. Register Super Admin (Protected)
│   ├── 1. Super Admin Login (Normal)
│   ├── 2. /auth/me dengan Access Token
│   ├── 3. Refresh Token
│   ├── 4. Logout
│   ├── 5. Internal Login — Admin Sekolah (Force Update)
│   ├── 6. Force Update — Berhasil
│   └── 7. Internal Login Setelah Force Update (Normal)
│
├── UTILITY (1 request)
│   └── Health Check
│
└── SEKOLAH & PENDIDIK
    ├── 01. Public Sekolah (3 requests)
    │   ├── List Jenjang Pendidikan
    │   ├── Pengajuan Sekolah Baru (JSON body)
    │   └── Upload Dokumen Legalitas (multipart/form-data)
    │
    ├── 02. Super Admin (6 requests)
    │   ├── List Sekolah (filter & pagination)
    │   ├── Detail Sekolah + State Log
    │   ├── Get Dokumen Legalitas (Presigned URL)
    │   ├── Verifikasi — Setuju (pengajuan → aktif)
    │   ├── Verifikasi — Tolak
    │   └── Update Data Sekolah
    │
    ├── 03. Admin Sekolah (3 requests)
    │   ├── Get My Sekolah
    │   ├── Update My Sekolah
    │   └── Pengajuan Ulang (setelah ditolak)
    │
    ├── 04. Pendidik Management (9 requests)
    │   ├── Tambah Pendidik
    │   ├── List Pendidik
    │   ├── Detail Pendidik
    │   ├── Update Pendidik
    │   ├── Update Status Pendidik (nonaktifkan)
    │   ├── Tambah Jadwal Mengajar
    │   ├── List Jadwal Pendidik
    │   ├── Update Jadwal
    │   └── Hapus Jadwal
    │
    └── 05. Tenaga Pendidik (4 requests)
        ├── Get My Profile
        ├── Update My Profile
        ├── Get Jadwal Harian
        └── Get Jadwal Minggu Ini
```

### Collection Variables

| Variable | Default | Keterangan |
|----------|---------|------------|
| `base_url` | `http://localhost:8080` | Base URL API |
| `email` | `budi@example.com` | Email testing mobile auth |
| `password` | `Password123!` | Password testing |
| `access_token` | *(kosong)* | Auto-set setelah login |
| `refresh_token` | *(kosong)* | Auto-set setelah login |
| `sekolah_id` | *(kosong)* | Auto-set setelah pengajuan sekolah |
| `tracking_code` | *(kosong)* | Untuk tracking progress |
| `admin_email` | `admin.pusat@simas.com` | Super Admin login |
| `internal_email` | `admin.sekolah@simas.com` | Admin Sekolah login |

---

## 8. Kode Error & Handling

### Sekolah Module

| Kode Error | HTTP Status | Penyebab |
|------------|-------------|----------|
| `sekolah_exists` | 409 | Email atau NPSN sudah terdaftar |
| `not_found` | 404 | Sekolah tidak ditemukan |
| `invalid_jenjang` | 400 | Jenjang tidak valid (bukan `sd_mi`, `smp_mts`, `sma_smk_ma_mak`) |
| `invalid_status` | 400 | Transisi status tidak valid |
| `incomplete_legalitas` | 400 | 4 dokumen legalitas belum lengkap diupload |
| `file_too_large` | 400 | File melebihi 3 MB |
| `invalid_file_format` | 400 | File bukan format PDF |
| `not_allowed` | 400 | Pengajuan ulang hanya untuk status `ditolak` |
| `invalid_tracking` | 404 | Kode tracking tidak ditemukan |

### Auth Module

| Kode Error | HTTP Status | Penyebab |
|------------|-------------|----------|
| `invalid_credentials` | 401 | Email atau password salah |
| `email_not_verified` | 403 | Email belum diverifikasi via OTP |
| `invalid_otp` | 400 | Kode OTP salah atau expired |
| `user_exists` | 409 | Email sudah terdaftar |
| `force_update_password` | 403 | Password harus diupdate (first login) |

---

## Appendix: Flow Diagram Registrasi Sekolah (Text)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         HALAMAN REGISTRASI SEKOLAH                           │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 1: ISI DATA SEKOLAH                                                   │
│  POST /api/v1/sekolah (JSON)                                                │
│  Response: { id, tracking_code, status: "pengajuan" }                         │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 2: UPLOAD DOKUMEN LEGALITAS                                           │
│  POST /api/v1/sekolah/{id}/dokumen (multipart/form-data)                    │
│  - akta_pendirian.pdf (max 3MB)                                              │
│  - nib.pdf (max 3MB)                                                         │
│  - sk_pendirian.pdf (max 3MB)                                                │
│  - siop.pdf (max 3MB)                                                        │
│  Response: { status: "menunggu_verifikasi" }                                  │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  STEP 3: TRACKING PROGRESS (boleh dicek kapan saja)                        │
│  GET /api/v1/sekolah/progress?tracking_code=...                             │
│  Response: { status, status_label, state_logs[] }                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  SUPER ADMIN DASHBOARD                                                       │
│  - List sekolah dengan status "menunggu_verifikasi"                          │
│  - Review dokumen via presigned URL                                          │
│  - Verifikasi: PUT /api/v1/sekolah/{id}/verifikasi                           │
│    ├── Setuju → status: "aktif" → auto-create admin_sekolah                  │
│    └── Tolak  → status: "ditolak" → email alasan penolakan                   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                    ┌─────────────────┴─────────────────┐
                    ▼                                   ▼
┌──────────────────────────────┐        ┌──────────────────────────────┐
│  DITERIMA (aktif)            │        │  DITOLAK (ditolak)           │
│  - Email kredensial dikirim   │        │  - Email alasan dikirim      │
│  - Admin Sekolah bisa login   │        │  - Bisa pengajuan ulang      │
│  - Kelola sekolah & pendidik  │        │    via /sekolah/saya/        │
│                               │        │    pengajuan-ulang           │
└──────────────────────────────┘        └──────────────────────────────┘
```

---

## Catatan Teknis

- **Swagger UI** tersedia di: `http://localhost:8080/swagger/index.html`
- **Build docs:** `swag init -g cmd/api/main.go`
- **State log** tersimpan di tabel `sekolah_state_log` untuk audit trail
- **File storage** menggunakan S3/MinIO dengan key pattern: `sekolah/{sekolah_id}/legalitas/{field}.pdf`
- **Presigned URL** untuk dokumen valid selama 15 menit

---

*Dokumen ini terakhir diperbarui: September 2026*
