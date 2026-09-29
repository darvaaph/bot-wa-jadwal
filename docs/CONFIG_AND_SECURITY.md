# 🔐 Panduan Konfigurasi & Keamanan Produksi (Security Reference v3.0)
# Bot WhatsApp Jadwal Kuliah & Web Admin Dashboard

Dokumen ini adalah **referensi kanonis konfigurasi lingkungan (*environment variables*), arsitektur keamanan, proteksi kriptografi, dan manajemen rahasia (*secret lifecycle*)** untuk Bot Jadwal v3.0.

---

## 📑 Daftar Isi
1. [Prinsip Desain Keamanan](#1-prinsip-desain-keamanan)
2. [Matriks Variabel Lingkungan (Environment Variables)](#2-matriks-variabel-lingkungan-environment-variables)
3. [Validasi Startup Fail-Closed](#3-validasi-startup-fail-closed)
4. [Manajemen Kunci Rahasia & Daur Hidup Secret](#4-manajemen-kunci-rahasia--daur-hidup-secret)
5. [Kebijakan Rate Limiting Terpusat (ADR-0009)](#5-kebijakan-rate-limiting-terpusat-adr-0009)
6. [Kebijakan Cookie Sesi & Proteksi CSRF (ADR-0010)](#6-kebijakan-cookie-sesi--proteksi-csrf-adr-0010)
7. [Kebijakan CORS & HTTP Security Headers (BE-014)](#7-kebijakan-cors--http-security-headers-be-014)
8. [Jaringan, Reverse Proxy & Validasi IP Klien](#8-jaringan-reverse-proxy--validasi-ip-klien)
9. [Sanitasi Log & Pencegahan Kebocoran Data (Data Hygiene)](#9-sanitasi-log--pencegahan-kebocoran-data-data-hygiene)

---

## 1. 🛡️ Prinsip Desain Keamanan

Arsitektur keamanan Bot Jadwal berlandaskan lima pilar utama:
1. **Fail-Closed by Default:** Pada lingkungan `production`, aplikasi menolak booting jika salah satu variabel keamanan hilang, kosong, atau diisi secara ceroboh.
2. **Zero-Trust Identity Fingerprinting:** Semua pelacakan sumber rate limiter dan identitas sesi menggunakan HMAC-SHA256 ber-garam (*keyed hashing*) sehingga alamat IP atau ID perangkat tidak dapat diekstraksi balik oleh pihak ketiga.
3. **Defense-in-Depth:** Perlindungan berlapis mulai dari reverse proxy CIDR restriction, exact-match CORS, strict Content Security Policy (CSP), HTTP Strict Transport Security (HSTS), hingga isolasi multi-tenant antar-kelas.
4. **No Plaintext Secret in Storage:** Tidak ada password, token otentikasi, atau kunci HMAC yang disimpan dalam bentuk teks polos baik di database, berkas log, maupun repositori Git.
5. **No Blind Fallback:** Tidak ada penurunan standar keamanan diam-diam ke default development saat berjalan di server cloud.

---

## 2. 📋 Matriks Variabel Lingkungan (Environment Variables)

Aplikasi membaca konfigurasi lingkungan melalui variabel berikut pada file `internal/config/config.go`:

| Nama Variabel | Lingkungan | Wajib? | Tipe Data / Format | Nilai Default | Contoh Aman (Non-Secret) | Penjelasan & Dampak |
|:---|:---:|:---:|:---|:---|:---|:---|
| `BOT_JADWAL_ENV` | Semua | Opsional | Enum: `development`, `test`, `production` | `development` | `production` | Menentukan mode kepatuhan keamanan. Pada `production`, validasi ketat diberlakukan. |
| `BOT_JADWAL_AUTH_HASH_KEY` | Prod | **Wajib di Prod** | Hex string / secret (min. 32 byte) | String kosong | `<GENERATE_MIN_32_BYTES_HEX>` | Kunci HMAC untuk sidik jari identity, rate limit subject hashing, dan CSRF verification. |
| `BOT_JADWAL_SECURE_COOKIES` | Prod | **Wajib di Prod** | Boolean: `true`, `false` | `false` | `true` | Mengharuskan cookie otentikasi `bv1` bertanda `Secure`. Wajib bernilai `true` saat HTTPS aktif. |
| `BOT_JADWAL_ALLOWED_ORIGINS` | Prod | Opsional | Comma-separated URLs | String kosong | `https://jadwal.kelas.ac.id` | Allowlist origin CORS eksak (skema, host, port). Dilarang menggunakan wildcard `*` di produksi. |
| `BOT_JADWAL_TRUSTED_PROXY_CIDRS` | Prod | Opsional | Comma-separated CIDR blocks | `127.0.0.1/32` | `127.0.0.1/32,10.0.0.0/8` | Daftar subnet IP reverse proxy yang dipercaya memasok header `X-Forwarded-For`. |
| `BOT_JADWAL_PUBLIC_BASE_URL` | Prod | Opsional | Canonical HTTPS URL | String kosong | `https://jadwal.kelas.ac.id` | URL publik resmi server untuk pembentukan tautan portal dan webhook callback. |
| `BOT_JADWAL_ADMIN_PASSWORD` | CLI | Opsional (CLI) | String min. 12 karakter | String kosong | `<GENERATE_PASSWORD_12_CHARS>` | Hanya dibaca saat provisioning administrator pertama via `cmd/provision-admin`. |
| `PORT` | Semua | Opsional | Integer port TCP | `8080` | `8080` | Port listening HTTP REST API server lokal. |
| `STORAGE_DIR` | Semua | Opsional | File path lokal | `storage` | `storage` | Direktori tempat database SQLite dan file sesi WhatsApp disimpan. |

---

## 3. 🚫 Validasi Startup Fail-Closed

Pada saat booting di lingkungan produksi (`BOT_JADWAL_ENV=production`), fungsi inisialisasi konfigurasi `config.ValidateForProduction()` di `internal/config/config.go` mengeksekusi pemeriksaan berkut sebelum membuka soket jaringan:

1. **`BOT_JADWAL_AUTH_HASH_KEY`**:
   - Tidak boleh kosong.
   - Panjang kunci minimal **32 byte**.
   - Menolak nilai default berbahaya (misal: `"change-me"`, `"secret"`, `"12345678901234567890123456789012"`).
2. **`BOT_JADWAL_SECURE_COOKIES`**:
   - Wajib bernilai boolean `true`. Server menolak berjalan dengan cookie `Secure=false` di mode produksi.
3. **`BOT_JADWAL_ALLOWED_ORIGINS`**:
   - Jika didefinisikan, setiap entri origin wajib memuat skema `https://` yang valid dan dilarang mengandung wildcard (`*`).
4. **`BOT_JADWAL_TRUSTED_PROXY_CIDRS`**:
   - Setiap entri wajib berupa notasi CIDR IP yang valid (misal `127.0.0.1/32` atau `10.0.0.0/8`).
5. **`BOT_JADWAL_PUBLIC_BASE_URL`**:
   - Jika didefinisikan, wajib menggunakan protokol `https://`.

> ⚠️ **Kegagalan Validasi:**  
> Jika salah satu aturan di atas dilanggar, aplikasi akan mencetak pesan kesalahan diagnostik yang jelas ke `stderr` dan keluar (*exit code 1*), mencegah layanan online dalam keadaan rentan.

---

## 4. 🔑 Manajemen Kunci Rahasia & Daur Hidup Secret

### A. Pembuatan Kunci Baru (Key Generation)
Kunci HMAC dan password administratif harus dihasilkan menggunakan generator bilangan acak berstandar kriptografi (*cryptographically secure pseudorandom number generator / CSPRNG*):

```bash
# 1. Menghasilkan kunci HMAC 32-byte (64 karakter heksadesimal)
openssl rand -hex 32

# 2. Menghasilkan password acak kuat (24 karakter alfabetik + simbol)
openssl rand -base64 18
```

### B. Injeksi Secret ke Lingkungan Server
* **Dilarang Keras:** Menyimpan kunci rahasia di dalam repositori Git, berkas `.env` publik, atau skrip build.
* **Metode Resmi Produksi:** Kunci diinjeksikan secara eksklusif melalui berkas unit `systemd` di `/etc/systemd/system/bot-jadwal.service` dengan hak akses terbatas:
  ```bash
  sudo chown root:root /etc/systemd/system/bot-jadwal.service
  sudo chmod 600 /etc/systemd/system/bot-jadwal.service
  ```

### C. Prosedur Rotasi Kunci (Secret Rotation)
Rotasi `BOT_JADWAL_AUTH_HASH_KEY` disarankan dilakukan secara berkala (misal tiap semester atau bila terjadi mutasi tim DevOps):
1. **Dampak Rotasi:**
   - Seluruh sesi login browser yang aktif akan dibatalkan (*session invalidation*) sehingga pengurus harus login kembali.
   - Bucket persistent rate limiting lama tidak berlaku lagi dan di-reset bersih secara otomatis.
   - Data jadwal perkuliahan dan catatan tugas di database SQLite **TIDAK TERPENGARUH**.
2. **Langkah Eksekusi:**
   ```bash
   # 1. Hasilkan kunci baru di terminal server
   NEW_KEY=$(openssl rand -hex 32)
   
   # 2. Perbarui nilai di /etc/systemd/system/bot-jadwal.service
   sudo nano /etc/systemd/system/bot-jadwal.service
   
   # 3. Muat ulang daemon dan restart bot
   sudo systemctl daemon-reload
   sudo systemctl restart bot-jadwal
   ```

---

## 5. ⏱️ Kebijakan Rate Limiting Terpusat (ADR-0009)

Untuk melindungi server dari serangan *brute force*, *credential stuffing*, dan *denial-of-service (DoS)*, aplikasi menerapkan **Persistent SQLite Rate Limiter** di `internal/ratelimit/` dengan tabel `security_attempts`.

### Daftar 8 Kebijakan Resmi:

| Policy Name | Target Endpoint | Batas Maksimum | Jendela Waktu | Kunci Identitas Subjek | Tindakan Saat Melanggar |
|:---|:---|:---:|:---:|:---|:---|
| `auth_login` | `POST /api/v1/auth/login` | 5 percobaan | 15 Menit | HMAC(IP + Identity) | HTTP 429 Too Many Requests + Lockout |
| `portal_passcode` | `POST /api/v1/portal/passcode` | 5 percobaan | 15 Menit | HMAC(IP + ClassID) | HTTP 429 + Transient Lockout |
| `portal_read` | `GET /api/v1/portal/*` | 60 request | 1 Menit | HMAC(IP Address) | HTTP 429 Rate Limit Exceeded |
| `portal_token` | `POST /api/v1/portal/token` | 10 request | 5 Menit | HMAC(IP Address) | HTTP 429 Rate Limit Exceeded |
| `task_mutation` | `POST, PUT, DELETE /api/v1/tasks/*` | 30 mutasi | 1 Menit | HMAC(UserID / Session) | HTTP 429 + Header Retry-After |
| `schedule_override` | `POST, PUT /api/v1/schedule/overrides` | 20 mutasi | 1 Menit | HMAC(UserID / Session) | HTTP 429 + Header Retry-After |
| `backup_operation` | `POST /api/v1/backup/*` | 5 operasi | 10 Menit | HMAC(UserID / Session) | HTTP 429 + Audit Log Incident |
| `invitation_consume`| `POST /api/v1/invitations/accept` | 10 percobaan| 15 Menit | HMAC(IP Address) | HTTP 429 Rate Limit Exceeded |

Setiap respons HTTP 429 selalu menyertakan header standar `Retry-After: <detik>` dan isi pesan JSON terstruktur `{ "error": { "code": "RATE_LIMIT_EXCEEDED", "message": "..." } }`.

---

## 6. 🍪 Kebijakan Cookie Sesi & Proteksi CSRF (ADR-0010)

### Atribut Cookie Sesi (`bv1`):
* `HttpOnly`: **Ya** (mencegah pencurian token sesi via serangan Cross-Site Scripting / XSS).
* `Secure`: **Ya di produksi** (hanya ditransmisikan melalui sambungan HTTPS terenkripsi).
* `SameSite`: **Lax** (memberikan keseimbangan antara proteksi CSRF dan navigasi tautan antar-situs).
* `Path`: `/` (berlaku untuk seluruh endpoint API dan dashboard).
* `Max-Age`: `86400` (24 jam sesi aktif).

### Proteksi Mutasi Bebas CSRF:
Seluruh mutasi data via HTTP method tidak aman (`POST`, `PUT`, `PATCH`, `DELETE`) dilindungi oleh middleware anti-CSRF di `internal/api/server.go`:
1. **Validasi Origin / Referer:** Memverifikasi bahwa header `Origin` atau `Referer` cocok dengan exact-origin server yang diizinkan.
2. **Custom Request Header Requirement:** Permintaan mutasi yang menggunakan autentikasi cookie wajib menyertakan header `X-Requested-With: XMLHttpRequest` yang tidak dapat dikirim secara otomatis oleh form HTML sederhana pihak ketiga.

---

## 7. 🛡️ Kebijakan CORS & HTTP Security Headers (BE-014)

Semua respons HTTP dari server backend Go secara otomatis diinjeksi dengan header keamanan modern:

```http
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; img-src 'self' data: https:; font-src 'self' https: data:; connect-src 'self';
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: geolocation=(), camera=(), microphone=()
Cache-Control: no-store, no-cache, must-revalidate, proxy-revalidate (pada endpoint sensitif & API v1)
```

### Kebijakan CORS:
* **Same-Host First:** Dashboard yang disematkan langsung dilayani dari origin yang sama, sehingga browser tidak memerlukan preflight CORS kompleks.
* **Exact Origin Allowlist:** Jika dashboard frontend di-host pada subdomain berbeda, daftarkan origin tersebut pada `BOT_JADWAL_ALLOWED_ORIGINS`. Server akan menolak origin liar secara eksplisit (tidak menggunakan `Access-Control-Allow-Origin: *`).

---

## 8. 🌐 Jaringan, Reverse Proxy & Validasi IP Klien

Server backend Go beroperasi di belakang reverse proxy lokal (Nginx / Caddy).

### Mencegah IP Spoofing:
Banyak penyerang mencoba memalsukan IP dengan memanipulasi header `X-Forwarded-For`. Backend Bot Jadwal memeriksa subnet IP koneksi langsung:
* Hanya jika koneksi TCP berasal dari IP yang terdaftar dalam `BOT_JADWAL_TRUSTED_PROXY_CIDRS` (default `127.0.0.1/32`), header `X-Forwarded-For` atau `Forwarded` akan dipercaya untuk menentukan IP klien.
* Jika koneksi berasal dari IP luar, backend akan mengabaikan header forwarder dan menggunakan alamat IP TCP fisik sebagai sumber identitas.

---

## 9. 🧹 Sanitasi Log & Pencegahan Kebocoran Data (Data Hygiene)

### Aturan Redaksi Log:
1. **Dilarang Mencatat Secret:** Nilai password, token bearer, cookie autentikasi, dan kunci HMAC tidak boleh dicetak ke berkas log `stdout`/`stderr` maupun `journalctl`.
2. **Penyamaran Identitas:** Alamat email pada log diagnostik disamarkan sebagian (contoh: `ad***@example.test`).
3. **Penyimpanan Runtime Terisolasi:** Direktori `storage/` diatur dengan izin hak akses `700` (`rwx------`) milik user Linux yang menjalankan bot, mencegah pembacaan berkas SQLite oleh akun pengguna lain di server.
4. **Retensi Log:** Log journalctl otomatis dirotasi oleh sistem operasi Ubuntu dengan batas kapasitas maksimum 100 MB (`SystemMaxUse=100M`).
