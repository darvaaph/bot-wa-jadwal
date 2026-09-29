# Panduan Integrasi REST API v1 untuk Frontend Developer

Dokumen ini berisi spesifikasi teknis lengkap REST API v1 untuk pengembang Frontend (Web Dashboard & Portal Mahasiswa). Semua endpoint telah aktif dan diverifikasi pada server backend.

---

## 1. Konvensi Dasar & Global

### 1.1 Base URL & Versi
- **Base URL:** `http://localhost:8080/api/v1`
- **Format Data:** JSON (`Content-Type: application/json; charset=utf-8`)
- **Zona Waktu & Tanggal:**
  - Waktu tulis (Request): Format ISO 8601 / RFC 3339 UTC (contoh: `2026-10-01T23:59:00Z`) atau `YYYY-MM-DD HH:MM`.
  - Waktu baca (Response): Format RFC 3339. Konversi ke waktu lokal pengguna (`Asia/Jakarta` / WIB) dilakukan di sisi klien.

### 1.2 Header Autentikasi
Untuk endpoint terproteksi (Dashboard Pengelola KM, PJ, dan Admin), kirimkan token Bearer:
```http
Authorization: Bearer <access_token>
```

### 1.3 Format Amplop Respons (Envelope)

#### A. Respons Berhasil (HTTP 200, 201)
```json
{
  "status": "success",
  "data": { ... } // atau array [...]
}
```

#### B. Respons Gagal (HTTP 400, 401, 403, 404, 409, 422, 500)
```json
{
  "status": "error",
  "error": {
    "code": "VALIDATION",
    "message": "Pesan deskriptif dalam Bahasa Indonesia",
    "details": { ... } // Opsional: informasi tambahan (misal versi konflik)
  }
}
```

#### Daftar Kode Error Standar:
| Kode Error | HTTP Status | Keterangan |
|---|---|---|
| `UNAUTHENTICATED` | 401 | Token Bearer tidak disertakan, format salah, atau kedaluwarsa. |
| `FORBIDDEN` | 403 | Peran pengguna (Role) tidak memiliki wewenang pada entitas/cakupan tersebut. |
| `NOT_FOUND` | 404 | Data yang diminta tidak ditemukan di database. |
| `VALIDATION` | 422 / 400 | Data input tidak lengkap, format salah, atau melanggar aturan bisnis. |
| `VERSION_CONFLICT` | 409 | Terjadi konflik versi (optimistic locking). Data telah diubah pengguna lain. |
| `RATE_LIMITED` | 429 | Batas percobaan terlampaui (misal: 5 kali gagal login dalam 15 menit). |

---

## 2. Autentikasi & Manajemen Sesi (`/api/v1/auth/*`)

### 2.1 Login Pengelola
`POST /api/v1/auth/login`
- **Hak Akses:** Publik (Rate-limited: 5 kegagalan per 15 menit).
- **Request Body:**
```json
{
  "identity_key": "+6281234567890",
  "password": "password123"
}
```
- **Response (200 OK):**
```json
{
  "status": "success",
  "data": {
    "token": "eyJhbGciOi...",
    "token_type": "Bearer",
    "expires_at": "2026-09-29T20:30:00Z",
    "need_context_choice": false,
    "assignments": [
      {
        "id": 1,
        "role": "KM",
        "class_id": 1,
        "class_code": "D4-TI-2024-A",
        "class_slug": "d4-ti-2024-a"
      }
    ]
  }
}
```
> **Catatan Frontend:** Jika `need_context_choice: true`, arahkan pengguna untuk memilih peran/kelas aktif terlebih dahulu menggunakan endpoint `switch-context`.

### 2.2 Informasi Profil & Konteks Aktif
`GET /api/v1/auth/me`
- **Header:** `Authorization: Bearer <token>`
- **Response (200 OK):**
```json
{
  "status": "success",
  "data": {
    "user": {
      "id": 1,
      "identity_key": "+6281234567890",
      "display_name": "Ketua Mahasiswa TI-1A"
    },
    "active_assignment": {
      "id": 1,
      "role": "KM",
      "class_id": 1,
      "class_code": "D4-TI-2024-A",
      "class_slug": "d4-ti-2024-a"
    },
    "classes": [
      {
        "id": 1,
        "code": "D4-TI-2024-A",
        "slug": "d4-ti-2024-a",
        "study_program": "Teknik Informatika",
        "cohort_year": 2024,
        "group_label": "A",
        "status": "ACTIVE"
      }
    ]
  }
}
```

### 2.3 Ganti Peran / Konteks Aktif
`POST /api/v1/auth/switch-context`
- **Header:** `Authorization: Bearer <token>`
- **Request Body:**
```json
{
  "role_assignment_id": 2
}
```
- **Response (200 OK):** Mengembalikan token Bearer baru dengan konteks peran yang telah diperbarui. Simpan token baru ini di `localStorage`/`sessionStorage`.

### 2.4 Logout
`POST /api/v1/auth/logout`
- **Header:** `Authorization: Bearer <token>`
- **Response (200 OK):**
```json
{
  "status": "success",
  "data": {
    "revoked": true
  }
}
```

### 2.5 Daftar Kelas Sesuai Cakupan

`GET /api/v1/classes`

- Bearer KM mengembalikan hanya kelas pada konteks aktifnya.
- Bearer System Admin mengembalikan seluruh kelas.
- `X-Portal-Token` mengembalikan hanya kelas yang terikat pada sesi portal tersebut.
- Bearer PJ ditolak dengan `403`; permintaan tanpa Bearer maupun portal token ditolak dengan `401`.

### 2.6 Rotasi Kode Portal Kelas

`POST /api/v1/classes/:slug/portal-code/rotate`

- **Auth:** KM pada kelas tersebut atau System Admin.
- **Request Body:** `{}` agar server membuat kode 8 digit, atau `{"code":"kode-baru"}` untuk menentukan kode sepanjang 6–128 karakter.
- **Response:** `portal_code`, `portal_code_version`, `portal_access_mode`, dan `reveal_once:true`.
- Tampilkan atau salin `portal_code` saat respons diterima. Nilai mentah tidak dapat diminta kembali dari backend.
- Rotasi langsung mencabut seluruh sesi portal versi sebelumnya.

### 2.7 Cutover Endpoint Tugas Legacy

`GET /api/tasks` masih tersedia sementara untuk pembacaan kompatibilitas dan mengambil data dari model v1. Responsnya memiliki header `Deprecation: true` serta `Link` menuju `/api/v1/tasks`.

`POST /api/tasks` dan `DELETE /api/tasks/:id` telah dihentikan dan selalu mengembalikan `410 Gone`. Seluruh perubahan tugas wajib menggunakan endpoint `/api/v1/tasks` dengan sesi pengelola yang valid.

---

## 3. Portal Mahasiswa (`/api/v1/portal/:slug/*`)

Endpoint pada modul ini bersifat **read-only**. Kelas mode `LINK` dapat diakses langsung tanpa token login. Kelas mode `CODE` memerlukan sesi portal terbatas. `:slug` adalah slug kelas (contoh: `d4-ti-2024-a` atau `d4-ti-2024-b`).

### 3.0 Membuka Portal Mode CODE

Tukar kode kelas menjadi token sesi:

```http
POST /api/v1/portal/:slug/session
Content-Type: application/json

{"code":"123456"}
```

Respons `201 Created` mengembalikan `portal_token` dan `expires_at`. Simpan token hanya di browser yang membutuhkannya, lalu kirim pada seluruh pembacaan portal melalui `X-Portal-Token`. Token tidak memberikan hak akses ke endpoint pengelola.

```http
X-Portal-Token: <portal_token>
```

Kode salah menggunakan respons `401` generik. Setelah lima kegagalan dalam 15 menit, percobaan berikutnya mendapat `429` selama 15 menit.

### 3.1 Ringkasan Dashboard Kelas (Summary)
`GET /api/v1/portal/:slug/summary?date=YYYY-MM-DD`
- **Query Params:** `date` (opsional, default hari ini).
- **Response (200 OK):**
```json
{
  "status": "success",
  "data": {
    "class": {
      "code": "D4-TI-2024-A",
      "slug": "d4-ti-2024-a",
      "program": "D4 Teknik Informatika",
      "cohort": 2024,
      "group": "A"
    },
    "date": "2026-09-28",
    "day_name": "Senin",
    "active_semester": {
      "academic_year": "2024/2025",
      "term": "GANJIL"
    },
    "ongoing_event": null,
    "next_event": {
      "course": "Pemrograman Web",
      "time": "08:00 - 10:00 WIB",
      "room": "LAB-1",
      "kind": "REGULAR"
    },
    "today_schedule": [
      {
        "id": 1,
        "course": "Pemrograman Web",
        "starts_at": "08:00",
        "ends_at": "10:00",
        "room": "LAB-1",
        "kind": "REGULAR",
        "lecturers": "Budi Santoso, M.Kom."
      }
    ],
    "nearest_tasks": [
      {
        "id": 4,
        "course": "Pemrograman Web",
        "title": "Tugas 1: Desain Responsif",
        "deadline_at": "2026-10-02T23:59:00Z"
      }
    ]
  }
}
```

### 3.2 Jadwal Kuliah Harian
`GET /api/v1/portal/:slug/schedule?date=YYYY-MM-DD`
- Menggabungkan pola reguler (`schedule_patterns`) dan kejadian terkini (`teaching_events`).
- Menghasilkan label status: `REGULAR`, `REPLACEMENT` (kuliah pengganti), `EXTRA` (kuliah tambahan), atau `SESSION_CANCELLED` (dibatalkan).

### 3.3 Daftar Tugas Aktif Mahasiswa
`GET /api/v1/portal/:slug/tasks?group=hari_ini|minggu_ini|mendatang|terlewat&offering_id=&q=`
- Mengembalikan daftar tugas berstatus `PUBLISHED` yang belum diarsipkan.

### 3.4 Detail Tugas
`GET /api/v1/portal/:slug/tasks/:id`
- Mengembalikan instruksi lengkap, batas waktu, berkas/link materi pendukung, dan tautan pengumpulan tugas.

### 3.5 Riwayat Perubahan Jadwal Terkini
`GET /api/v1/portal/:slug/changes?since=YYYY-MM-DD`
- Menampilkan pengumuman resmi kuliah pengganti atau pembatalan sesi yang telah terbit.

### 3.6 Berkas & Tautan Penting Kelas
`GET /api/v1/portal/:slug/materials?offering_id=`
- Menampilkan link Google Drive, modul kuliah, grup WhatsApp, repository GitHub, atau link Zoom/Meet.

---

## 4. Manajemen Tugas Kuliah (`/api/v1/tasks/*`)

Khusus untuk peran **PJ (Penanggung Jawab Matkul)** dan **KM (Ketua Mahasiswa)**.

### 4.1 Mengambil Daftar Tugas Berdasarkan Tab
`GET /api/v1/tasks?tab=aktif|draf|review|selesai|terlewat|arsip&offering_id=`
- **Tab Filter:**
  - `aktif`: Tugas terbit (`PUBLISHED`) yang belum selesai dan belum melewati tenggat.
  - `draf`: Tugas yang masih dalam penyusunan oleh PJ/KM (`DRAFT`).
  - `review`: Tugas yang diajukan oleh PJ dan menunggu persetujuan KM (`NOT_REVIEWED`).
  - `selesai`: Tugas yang telah ditandai selesai (`completed_at != null`).
  - `terlewat`: Tugas yang telah melewati batas waktu pengumpulan.
  - `arsip`: Tugas yang telah diarsipkan.

### 4.2 Membuat Tugas Baru
`POST /api/v1/tasks`
- **Header:** `Authorization: Bearer <token>`
- **Request Body (Draft):**
```json
{
  "offering_id": 1,
  "title": "Tugas Analisis Algoritma",
  "instructions": "Kerjakan soal 1-5 di buku teks",
  "deadline_at": "2026-10-05T23:59:00Z",
  "save_as": "draft"
}
```
- **Request Body (Publish Langsung):**
```json
{
  "offering_id": 1,
  "title": "Tugas Praktikum 2",
  "instructions": "Upload kode sumber ke GitHub dan submit URL",
  "deadline_at": "2026-10-05T23:59:00Z",
  "task_type": "INDIVIDU",
  "submission_url": "https://classroom.google.com/test",
  "save_as": "published"
}
```
> **Aturan Bisnis:** 
> - Menyimpan sebagai `published` mewajibkan: `title`, `instructions`, `deadline_at`, dan salah satu dari `submission_url` atau `submission_text`.
> - Jika dibuat oleh PJ, status menjadi `PUBLISHED` + `NOT_REVIEWED` (masuk antrean review KM).
> - Jika dibuat oleh KM, otomatis berstatus `APPROVED` dan langsung disiarkan ke bot WhatsApp kelas.

### 4.3 Detail Tugas & Riwayat Review
`GET /api/v1/tasks/:id`
- Mengembalikan data tugas, nilai `version`, daftar ulasan KM (`reviews`), dan status publikasi.

### 4.4 Memperbarui Tugas (Optimistic Locking)
`PATCH /api/v1/tasks/:id`
- **Header:** `Authorization: Bearer <token>`
- **Request Body:**
```json
{
  "version": 1,
  "title": "Tugas Praktikum 2 (Revisi)",
  "instructions": "Perbaikan instruksi soal nomor 3",
  "deadline_at": "2026-10-06T23:59:00Z",
  "save_as": "published"
}
```
> **PENTING (Optimistic Locking):**
> Frontend **wajib** menyertakan field `version` sesuai versi data saat ini.
> Jika server merespons **HTTP 409 Conflict** (`VERSION_CONFLICT`), berarti ada pengguna lain yang telah menyimpan perubahan. Frontend wajib menampilkan jendela komparasi (Diff Modal) agar input pengguna tidak tertimpa tanpa sengaja.

### 4.5 Peninjauan & Persetujuan Tugas oleh KM
`POST /api/v1/tasks/:id/reviews`
- **Hak Akses:** Hanya KM / System Admin.
- **Request Body:**
```json
{
  "decision": "APPROVED", // Pilihan: "APPROVED", "CHANGES_REQUESTED", "REVOKED"
  "task_version": 1,
  "note": "Instruksi sudah jelas dan disetujui."
}
```
> **Efek Keputusan:**
> - `APPROVED`: Tugas disetujui, tampil di Portal Mahasiswa, dan otomatis mengantre notifikasi ke bot WhatsApp kelas.
> - `CHANGES_REQUESTED`: Status tugas dikembalikan ke `DRAFT` agar PJ dapat merevisi.
> - `REVOKED`: Tugas ditarik dari publikasi.

### 4.6 Menandai Selesai, Arsip, dan Pemulihan
- `POST /api/v1/tasks/:id/complete` ➔ Menandai tugas telah selesai (`completed_at`).
- `POST /api/v1/tasks/:id/archive` ➔ Mengarsipkan tugas lampau (`archived_at`).
- `POST /api/v1/tasks/:id/restore` ➔ Mengembalikan tugas dari arsip/selesai.
- `DELETE /api/v1/tasks/:id` ➔ Hapus lunak tugas (soft-delete beraudit).

---

## 5. Jadwal & Kuliah Pengganti (`/api/v1/teaching-events/*`)

### 5.1 Mengambil Daftar Kejadian Perkuliahan
`GET /api/v1/teaching-events?status=draft|published|revoked&from=&to=`
- Digunakan untuk menampilkan tab Draf, Terbit, atau Dicabut pada dashboard jadwal.

### 5.2 Mengajukan Kuliah Pengganti Baru
`POST /api/v1/teaching-events`
- **Request Body:**
```json
{
  "owner_offering_id": 1,
  "event_kind": "REPLACEMENT", // Pilihan: "REPLACEMENT", "EXTRA", "SESSION_CANCELLED"
  "starts_at": "2026-10-05T08:00:00Z",
  "ends_at": "2026-10-05T10:00:00Z",
  "origin_pattern_id": 2, // ID pola reguler yang digantikan (jika REPLACEMENT)
  "origin_occurrence_date": "2026-10-02",
  "room_id": 1,
  "reason": "Dosen menghadiri seminar nasional"
}
```

### 5.3 Pratinjau Deteksi Konflik & Komparasi
`POST /api/v1/teaching-events/:id/preview`
- Memeriksa ketersediaan ruangan, bentrok jadwal dosen, atau tabrakan jam kelas sebelum diterbitkan.
- **Response Data:**
```json
{
  "status": "success",
  "data": {
    "has_conflict": false,
    "conflicts": [],
    "comparison": {
      "before": "Jumat, 02 Okt 2026 (08:00 - 10:00 WIB) di R.301",
      "after": "Senin, 05 Okt 2026 (08:00 - 10:00 WIB) di LAB-1"
    }
  }
}
```

### 5.4 Menerbitkan Jadwal Pengganti (Publish)
`POST /api/v1/teaching-events/:id/publish`
- Status berubah menjadi `PUBLISHED`.
- Otomatis masuk ke antrean worker bot WhatsApp untuk disiarkan ke grup kelas.

### 5.5 Mencabut / Membatalkan Jadwal (Revoke)
`POST /api/v1/teaching-events/:id/revoke`
- **Request Body:**
```json
{
  "reason": "Dosen berhalangan hadir kembali, kuliah pengganti dibatalkan"
}
```
- Status berubah menjadi `REVOKED`. Sistem otomatis menyiarkan notifikasi pembatalan ke grup WhatsApp kelas dan mengembalikan jadwal portal ke pola reguler.

---

## 6. Modul Materi & Berkas (`/api/v1/materials/*`)

### 6.1 Mengambil Daftar Materi
`GET /api/v1/materials?offering_id=`

### 6.2 Menambahkan Materi Baru
`POST /api/v1/materials`
- **Request Body:**
```json
{
  "offering_id": 1,
  "title": "Slide Pertemuan 4: Normalisasi Basis Data",
  "material_type": "DOCUMENT", // "DOCUMENT", "MEETING", "REPOSITORY", "PORTAL"
  "url": "https://drive.google.com/file/d/test",
  "description": "Pelajari bentuk 1NF s.d 3NF"
}
```

---

## 7. Modul Administrasi & Operasional v1.1+

| Method & Path | Hak Akses | Keterangan |
|---|---|---|
| `GET /api/v1/rooms/candidates?starts_at=&ends_at=&capacity=` | KM / PJ | Mencari ruangan kosong yang tidak bentrok pada jam tersebut. |
| `POST /api/v1/teaching-events/:id/room-confirmations` | KM | Konfirmasi persetujuan penggunaan ruangan dari Tata Usaha/Pengelola Lab. |
| `GET /api/v1/notifications?status=PENDING\|SENT\|FAILED` | Admin / KM | Melihat antrean status pengiriman pesan broadcast WhatsApp. |
| `POST /api/v1/notifications/:id/retry` | Admin / KM | Menjadwalkan ulang pesan `FAILED`/`CANCELLED` menjadi `PENDING` (`{retry_scheduled:true}`, tanpa `attempt_number`). |
| `GET /api/v1/audit?entity_type=&action=&limit=` | Admin / KM | Melihat log jejak audit perubahan data penting. |
| `POST /api/v1/backups` | Admin | Membuat backup basis data SQLite target v1 secara instan. |
| `POST /api/v1/restores` | Admin | Verifikasi Backup (verify-only, ADR-0008): checksum, format, schema, scope → `VERIFIED` + `restore_performed:false`. Database aktif tidak diganti. |
| `GET /api/v1/admin/status` | Admin | Telemetri kesehatan bot, koneksi WhatsApp, dan metrik sistem. |
| `POST /api/v1/admin/users/:id/suspend` | Admin | Menonaktifkan akun pengguna bermasalah. |
| `POST /api/v1/admin/users/:id/recover` | Admin | Mengaktifkan kembali akun pengguna. |

---

## 8. Contoh Implementasi Klien Frontend (JavaScript Fetch Wrapper)

Berikut adalah contoh fungsi pembantu (helper) untuk integrasi di Frontend (Tailwind + Alpine.js):

```javascript
// web/js/api.js - HTTP Client Terpusat
const API_BASE = '/api/v1';

export async function request(path, options = {}) {
  const token = localStorage.getItem('access_token');
  const headers = {
    'Content-Type': 'application/json',
    ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
    ...options.headers,
  };

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  });

  const resJson = await response.json();

  if (!response.ok) {
    // Tangani 401 Unauthorized: Sesi habis
    if (response.status === 401) {
      localStorage.removeItem('access_token');
      window.location.href = '/login.html';
      throw new Error('Sesi telah berakhir. Silakan login kembali.');
    }

    // Tangani 409 Conflict: Optimistic Locking
    if (response.status === 409) {
      const err = new Error(resJson.error?.message || 'Data telah diubah oleh pengguna lain');
      err.code = 'VERSION_CONFLICT';
      err.details = resJson.error?.details;
      throw err;
    }

    throw new Error(resJson.error?.message || 'Terjadi kesalahan sistem');
  }

  return resJson.data;
}

// Contoh Penggunaan:
// 1. Ambil Summary Portal
export const getPortalSummary = (slug, date) => 
  request(`/portal/${slug}/summary${date ? `?date=${date}` : ''}`);

// 2. Simpan Tugas dengan Optimistic Locking
export async function saveTask(taskData) {
  try {
    if (taskData.id) {
      return await request(`/tasks/${taskData.id}`, {
        method: 'PATCH',
        body: JSON.stringify(taskData),
      });
    } else {
      return await request('/tasks', {
        method: 'POST',
        body: JSON.stringify(taskData),
      });
    }
  } catch (err) {
    if (err.code === 'VERSION_CONFLICT') {
      alert('Konflik Versi: Data ini telah diubah oleh orang lain. Halaman akan menyajikan data terbaru.');
      // Munculkan modal perbandingan / diff
    } else {
      alert(`Gagal menyimpan tugas: ${err.message}`);
    }
  }
}
```

---

## 9. File Koleksi Postman & Insomnia

Koleksi lengkap siap pakai tersedia di direktori repositori:
- **Postman:** [`docs/api/bot-jadwal-v1.postman_collection.json`](file:///f:/Project/bot-jadwal/docs/api/bot-jadwal-v1.postman_collection.json)
- **Insomnia:** [`docs/api/bot-jadwal-v1.insomnia_collection.json`](file:///f:/Project/bot-jadwal/docs/api/bot-jadwal-v1.insomnia_collection.json)
- **Panduan Pengujian Postman:** [`docs/api/PANDUAN_TESTING_POSTMAN.md`](file:///f:/Project/bot-jadwal/docs/api/PANDUAN_TESTING_POSTMAN.md)
