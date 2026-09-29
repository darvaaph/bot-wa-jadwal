# ☁️ Panduan Server, Deployment & Disaster Recovery (DevOps Runbook v3.0)
# Bot WhatsApp Jadwal Kuliah & Web Admin Dashboard

Dokumen ini adalah **runbook operasional kanonis** untuk pengelola server dan tim DevOps yang bertugas melakukan *maintenance*, *deployment*, konfigurasi lingkungan produksi, pencadangan (*backup*), dan pemulihan darurat (*disaster recovery*) pada server cloud Microsoft Azure.

---

## 📑 Daftar Isi
1. [Informasi Server Produksi](#1-informasi-server-produksi)
2. [Akses Remote Server (SSH) & Prasyarat Keamanan](#2-akses-remote-server-ssh--prasyarat-keamanan)
3. [SOP Deployment Standar 6-Langkah (Atomic & Immutable)](#3-sop-deployment-standar-6-langkah-atomic--immutable)
4. [Skenario Pembaruan Data Non-Biner (Jadwal & Reminder)](#4-skenario-pembaruan-data-non-biner-jadwal--reminder)
5. [Kendali Layanan Sistem (`systemd`)](#5-kendali-layanan-sistem-systemd)
6. [Panduan Pengujian Lokal Aman (Mencegah Bentrok Sesi)](#6-panduan-pengujian-lokal-aman-mencegah-bentrok-sesi)
7. [Konfigurasi Zona Waktu (WIB)](#7-konfigurasi-zona-waktu-wib)
8. [Struktur Berkas & Direktori di Server](#8-struktur-berkas--direktori-di-server)
9. [Konfigurasi Environment & Hardening Produksi](#9-konfigurasi-environment--hardening-produksi)
10. [SOP Rollback Darurat (Rollback Runbook)](#10-sop-rollback-darurat-rollback-runbook)

---

## 1. 🖥️ Informasi Server Produksi

* **Cloud Provider:** Microsoft Azure (Azure for Students)
* **Tipe VM:** `Standard_B2ats_v2` (2 vCPU, 1 GiB RAM, AMD EPYC)
* **Sistem Operasi:** Ubuntu Server 24.04 LTS (x64)
* **Region:** Southeast Asia / Malaysia West
* **Alamat IP Publik:** `<IP_SERVER_PRODUKSI>` *(Lihat catatan kredensial aman tim DevOps)*
* **Port Layanan Terbuka:**
  * `22` (SSH - Akses remote terminal terautentikasi)
  * `80` (HTTP - Reverse proxy Nginx / Let's Encrypt challenge)
  * `443` (HTTPS - Akses web dashboard terenkripsi TLS)
  * `8080` (Lokal 127.0.0.1 - Backend Bot Go, dibatasi hanya untuk reverse proxy lokal)

---

## 2. 🔑 Akses Remote Server (SSH) & Prasyarat Keamanan

Buka **PowerShell** (Windows) atau **Terminal** (macOS/Linux):
```bash
ssh <USER_SSH>@<IP_SERVER_PRODUKSI>
```
* Autentikasi disarankan menggunakan SSH Key (`~/.ssh/id_ed25519`).
* Dilarang melakukan login langsung sebagai `root`; gunakan user non-root dengan hak akses `sudo`.
* Untuk keluar dari sesi terminal server: ketik `exit` atau tekan `Ctrl + D`.

---

## 3. 🚀 SOP Deployment Standar 6-Langkah (Atomic & Immutable)

Setiap rilis biner baru wajib mengikuti prosedur 6-langkah berikut untuk memastikan verifikasi integritas checksum SHA-256, ketersediaan snapshot rollback, dan zero corrupt binary runtime:

### Langkah 1: Kompilasi Biner Linux & Buat Hash SHA-256 (di Laptop Pengembang)
Pastikan berada di root folder repositori proyek, branch rilis bersih, dan seluruh automated test telah lulus:
```powershell
# 1. Kompilasi biner mandiri Linux amd64
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -trimpath -ldflags="-s -w" -o bot-jadwal ./cmd/bot
Remove-Item Env:GOOS; Remove-Item Env:GOARCH; Remove-Item Env:CGO_ENABLED

# 2. Buat berkas verifikasi checksum SHA-256
$hash = (Get-FileHash -Algorithm SHA256 .\bot-jadwal).Hash.ToLower()
"$hash  bot-jadwal" | Out-File -Encoding ascii -NoNewline .\bot-jadwal.sha256
Write-Host "Biner berhasil dikompilasi. SHA-256: $hash"
```

### Langkah 2: Unggah Biner & Checksum ke Server via SCP
```powershell
scp bot-jadwal <USER_SSH>@<IP_SERVER_PRODUKSI>:~/bot-jadwal.new
scp bot-jadwal.sha256 <USER_SSH>@<IP_SERVER_PRODUKSI>:~/bot-jadwal.sha256
```
> ℹ️ **Alasan Penamaan `bot-jadwal.new`:**
> Biner bot yang sedang aktif berjalan di Linux dilindungi oleh kernel (*text file busy*). Mengunggah artefak ke berkas sementara `.new` mencegah kegagalan transfer di tengah jalan dan menjamin penggantian biner dilakukan secara atomik via perintah `mv`.

### Langkah 3: Verifikasi Integritas Checksum di Server (SSH)
Masuk ke terminal server (`ssh <USER_SSH>@<IP_SERVER_PRODUKSI>`) dan validasi keutuhan berkas:
```bash
sha256sum -c bot-jadwal.sha256
```
*Output wajib menunjukkan:* `bot-jadwal: OK`.
Jika gagal (*FAILED*), **JANGAN LANJUTKAN DEPLOYMENT**. Hapus berkas `.new` dan ulangi proses transfer.

### Langkah 4: Pencadangan Pra-Deploy (Pre-Deploy Backup Snapshot)
Sebelum mengganti biner atau memulai migrasi, buat salinan biner aktif saat ini dan cadangan database SQLite:
```bash
# 1. Backup biner saat ini sebagai fallback instan
cp bot-jadwal bot-jadwal.prev

# 2. Buat snapshot database SQLite dengan aman
BACKUP_DIR="storage/backups/pre_deploy_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

if command -v sqlite3 >/dev/null 2>&1; then
    sqlite3 storage/bot_v1.db ".backup '$BACKUP_DIR/bot_v1.db'"
    [ -f storage/tugas.db ] && sqlite3 storage/tugas.db ".backup '$BACKUP_DIR/tugas.db'"
else
    cp storage/bot_v1.db* "$BACKUP_DIR/" 2>/dev/null || true
    cp storage/tugas.db* "$BACKUP_DIR/" 2>/dev/null || true
fi
echo "Pre-deploy snapshot tersimpan di: $BACKUP_DIR"
```

### Langkah 5: Penggantian Atomik & Restart Layanan
```bash
mv bot-jadwal.new bot-jadwal
chmod 755 bot-jadwal
sudo systemctl restart bot-jadwal
```

### Langkah 6: Verifikasi Kesehatan Pasca-Deploy (Multi-Tier Health Check)
Segera periksa kesehatan sistem dalam 60 detik pertama:
```bash
# 1. Cek status service di systemd
sudo systemctl status bot-jadwal --no-pager

# 2. Uji endpoint REST API liveness probe
curl -fsS http://127.0.0.1:8080/api/health || echo "ERROR: Health check gagal!"

# 3. Uji endpoint telemetri status bot
curl -fsS http://127.0.0.1:8080/api/status

# 4. Periksa log booting dan inisialisasi database
sudo journalctl -u bot-jadwal -n 30 --no-pager
```
*Kriteria Sukses:*
- Service berada dalam status `Active: active (running)`.
- HTTP `/api/health` mengembalikan kode `200 OK` dengan status `healthy`.
- Log tidak mencatat fatal panic, database invariant corruption, atau failure startup validation.

---

## 4. 📂 Skenario Pembaruan Data Non-Biner (Jadwal & Reminder)

Jika perubahan hanya menyangkut data kurikulum atau jam pengingat tanpa perubahan kode Go:

### A. Pembaruan Jadwal Kuliah Master (`data/jadwal/*.json`)
1. 💻 **Di Laptop (PowerShell):**
   ```powershell
   scp -r data/jadwal/* <USER_SSH>@<IP_SERVER_PRODUKSI>:~/data/jadwal/
   ```
2. ☁️ **Di Server (SSH):**
   ```bash
   sudo systemctl restart bot-jadwal
   ```
   *(Atau dari WhatsApp, administrator dapat memicu pemuatan ulang via perintah `!reload`).*

### B. Pembaruan Grup Pengingat Pagi (`reminder_groups.json`)
1. 💻 **Di Laptop (PowerShell):**
   ```powershell
   scp reminder_groups.json <USER_SSH>@<IP_SERVER_PRODUKSI>:~/
   ```
2. ☁️ **Di Server (SSH):**
   ```bash
   sudo systemctl restart bot-jadwal
   ```

---

## 5. 🛠️ Kendali Layanan Sistem (`systemd`)

Layanan bot dikelola sebagai daemon Linux menggunakan unit systemd `/etc/systemd/system/bot-jadwal.service`.

| Kebutuhan Operasional | Perintah di Terminal Server | Keterangan |
| :--- | :--- | :--- |
| **Cek Status Service** | `sudo systemctl status bot-jadwal` | Memeriksa apakah bot aktif running, pid, dan konsumsi memori. |
| **Restart Service** | `sudo systemctl restart bot-jadwal` | Memuat ulang biner atau konfigurasi lingkungan. |
| **Hentikan Layanan** | `sudo systemctl stop bot-jadwal` | Mematikan bot (misal saat maintenance darurat). |
| **Nyalakan Layanan** | `sudo systemctl start bot-jadwal` | Menjalankan bot kembali di background. |
| **Pantau Log Realtime** | `sudo journalctl -u bot-jadwal -f` | Streaming log output aplikasi live (`Ctrl + C` untuk keluar). |
| **Lihat Log Terakhir** | `sudo journalctl -u bot-jadwal -n 100 --no-pager` | Menampilkan 100 baris log terakhir tanpa paginasi. |

---

## 6. 🛡️ Panduan Pengujian Lokal Aman (Mencegah Bentrok Sesi)

> ⚠️ **PERINGATAN SANGAT PENTING MENGENAI SESI WHATSAPP:**
> **JANGAN PERNAH menekan tombol "Keluar / Logout" pada perangkat tertaut WhatsApp di ponsel bot.**
> Menekan logout di WhatsApp akan mencabut kunci kriptografi sesi secara permanen dari server Meta. Hal ini menyebabkan sesi di cloud mati total dan mewajibkan pemindaian QR code ulang secara fisik oleh pemegang nomor.

### Alur Kerja Pengujian Lokal:

Untuk menguji perubahan kode di komputer pengembang tanpa mengganggu server produksi:

1. **Pengujian Frontend & REST API Dashboard (Sangat Direkomendasikan):**
   Gunakan mode `-web-only` agar aplikasi tidak menghubungkan klien WhatsApp sama sekali:
   ```powershell
   go run ./cmd/bot -web-only
   ```
   * Dashboard dapat diakses di `http://localhost:8080`.
   * Database runtime lokal terisolasi di `storage/`.
   * WhatsApp nonaktif total; server cloud produksi terlindungi penuh dari risiko perebutan soket.

2. **Pengujian Integrasi WhatsApp (Gunakan Sesi Dev Terpisah):**
   Jika wajib menguji perintah WhatsApp, gunakan opsi `-session` dengan berkas sesi dev dan nomor uji coba cadangan:
   ```powershell
   go run ./cmd/bot -session storage/sesi_dev.db
   ```
   * Scan QR Code yang muncul di terminal menggunakan **nomor WhatsApp testing cadangan**.
   * **DILARANG KERAS menyalin atau membuka berkas sesi produksi `storage/sesi_bot.db` secara lokal!**

3. **Pengujian Unit Otomatis (Offline):**
   Jalankan seluruh test suite unit tanpa memerlukan koneksi jaringan maupun WhatsApp:
   ```powershell
   go test -v ./...
   ```

---

## 7. 🕒 Konfigurasi Zona Waktu (WIB)

Aplikasi sangat bergantung pada zona waktu **WIB (Asia/Jakarta)** untuk kalkulasi hari kuliah, filter jadwal harian, penentuan tenggat waktu tugas pukul 23:59 WIB, dan broadcast otomatis pukul 06:00 WIB.

Pastikan server produksi Azure menggunakan zona waktu Asia/Jakarta:
```bash
sudo timedatectl set-timezone Asia/Jakarta
```
Verifikasi dengan menjalankan perintah:
```bash
date
# Output wajib berakhiran WIB, contoh: Tue Sep 29 18:20:00 WIB 2026
```

---

## 8. 📁 Struktur Berkas & Direktori di Server

Berikut bagan struktur berkas kanonis pada direktori home server (`/home/<USER_SSH>/`):

```text
/home/<USER_SSH>/
├── bot-jadwal                 # Biner aktif aplikasi (Linux ELF 64-bit)
├── bot-jadwal.prev            # Cadangan biner versi sebelumnya (untuk instant rollback)
├── bot-jadwal.sha256          # Checksum SHA-256 rilis aktif
├── data/
│   ├── jadwal/                # 19 berkas JSON kurikulum master per kelas
│   └── jadwal.json            # Konfigurasi kurikulum default
└── storage/                   # Direktori runtime state (akses terisolasi chmod 700)
    ├── bot_v1.db              # Database SQLite utama v3.0 (tasks, users, audit, rate limit)
    ├── bot_v1.db-wal          # SQLite Write-Ahead Log file
    ├── bot_v1.db-shm          # SQLite Shared Memory index file
    ├── tugas.db               # Database SQLite pendukung / legacy compatibility
    ├── sesi_bot.db            # Database sesi autentikasi WhatsMeow
    ├── reminder_groups.json   # Konfigurasi preferensi broadcast grup
    └── backups/               # Direktori snapshot database pra-deploy dan rotasi backup
        ├── pre_deploy_20260929_180000/
        └── ...
```

---

## 9. 🔐 Konfigurasi Environment & Hardening Produksi

Aplikasi membaca konfigurasi lingkungan melalui environment variables. Konfigurasi ini didefinisikan secara aman pada unit file systemd di `/etc/systemd/system/bot-jadwal.service` dalam blok `[Service]`:

```ini
[Unit]
Description=Bot WhatsApp Jadwal Kuliah & Web Admin Dashboard v3.0
After=network.target

[Service]
Type=simple
User=<USER_SSH>
WorkingDirectory=/home/<USER_SSH>
ExecStart=/home/<USER_SSH>/bot-jadwal
Restart=always
RestartSec=5s

# Konfigurasi Environment & Keamanan Produksi
Environment="BOT_JADWAL_ENV=production"
Environment="BOT_JADWAL_AUTH_HASH_KEY=<GENERATE_MIN_32_BYTES_HEX>"
Environment="BOT_JADWAL_SECURE_COOKIES=true"
Environment="BOT_JADWAL_ALLOWED_ORIGINS=https://jadwal.kelasanda.com"
Environment="BOT_JADWAL_TRUSTED_PROXY_CIDRS=127.0.0.1/32,10.0.0.0/8"
Environment="BOT_JADWAL_PUBLIC_BASE_URL=https://jadwal.kelasanda.com"
Environment="STORAGE_DIR=storage"
Environment="PORT=8080"

# Isolasi Keamanan Proses Linux
ProtectSystem=full
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

### Penjelasan Variabel Konfigurasi Produksi:
* `BOT_JADWAL_ENV`: Wajib diset ke `production`. Server akan mengeksekusi validasi *fail-closed* saat startup; jika konfigurasi keamanan belum memenuhi kriteria, server akan menolak berjalan demi mencegah celah kerentanan.
* `BOT_JADWAL_AUTH_HASH_KEY`: Kunci rahasia minimal 32 byte untuk fungsi HMAC-SHA256 (sidik jari perangkat, subject hashing, dan persistent rate limiting). **JANGAN PERNAH menyimpan kunci ini di repositori Git**. Buat kunci acak di terminal server:
  ```bash
  openssl rand -hex 32
  ```
* `BOT_JADWAL_SECURE_COOKIES`: Wajib bernilai `true` pada deployment HTTPS agar cookie otentikasi browser (`bv1`) dilengkapi atribut `Secure; SameSite=Lax`.
* `BOT_JADWAL_ALLOWED_ORIGINS`: Daftar origin CORS terpercaya dipisahkan tanda koma (skema, domain, dan port exact match tanpa wildcard `*`).
* `BOT_JADWAL_TRUSTED_PROXY_CIDRS`: Daftar CIDR IP reverse proxy tepercaya (misal Nginx lokal `127.0.0.1/32`). Mencegah pemalsuan identitas IP klien pada header `X-Forwarded-For`.
* `BOT_JADWAL_PUBLIC_BASE_URL`: URL publik kanonis aplikasi (wajib skema HTTPS).

---

## 10. 🚨 SOP Rollback Darurat (Rollback Runbook)

Jika setelah deployment ditemukan anomali kritis, prosedur rollback harus segera dieksekusi sebelum batas waktu observasi (*Decision Deadline*: maksimal **15 menit** pasca cutover).

### Kondisi Pemicu Rollback (Rollback Triggers):
1. **Crash Loop:** Service `bot-jadwal` gagal berjalan stabil atau terus me-restart otomatis.
2. **Health Check Failure:** Endpoint `/api/health` mengembalikan error 500 atau timeout lebih dari 30 detik.
3. **Migrasi Database Rusak:** Terjadi kegagalan migrasi SQLite atau rusaknya integritas relasi tabel data.
4. **Celah Keamanan Kritis:** Ditemukan kegagalan autentikasi, bypass otorisasi role, atau header keamanan hilang.
5. **Malfungsi Notifikasi:** Terjadi pesan berulang tanpa henti (*infinite spam loop*) ke grup WhatsApp.

### Prosedur Rollback Langkah Demi Langkah:

#### 1. Hentikan Layanan yang Bermasalah
```bash
sudo systemctl stop bot-jadwal
```

#### 2. Kembalikan Biner ke Versi Sebelumnya
```bash
cp bot-jadwal.prev bot-jadwal
chmod 755 bot-jadwal
```

#### 3. Pulihkan Snapshot Database (Jika Schema Mengalami Kerusakan)
Jika migrasi database versi baru merusak data atau struktur schema:
```bash
LATEST_BACKUP=$(ls -td storage/backups/pre_deploy_* | head -n 1)
if [ -n "$LATEST_BACKUP" ]; then
    echo "Memulihkan database dari snapshot: $LATEST_BACKUP"
    cp "$LATEST_BACKUP"/bot_v1.db* storage/
    [ -f "$LATEST_BACKUP/tugas.db" ] && cp "$LATEST_BACKUP"/tugas.db* storage/
fi
```

#### 4. Nyalakan Kembali Layanan
```bash
sudo systemctl start bot-jadwal
```

#### 5. Verifikasi Pemulihan Sistem
```bash
# Periksa status service dan health check
sudo systemctl status bot-jadwal --no-pager
curl -fsS http://127.0.0.1:8080/api/health
sudo journalctl -u bot-jadwal -n 20 --no-pager
```

#### 6. Komunikasi & Eskalasi Insiden
1. Beritahukan tim pengembang dan koordinator kelas di grup komunikasi teknis:
   > *"Rollback versi rilis telah berhasil dieksekusi pada pukul HH:MM WIB. Sistem telah kembali stabil ke versi sebelumnya. Penyelidikan akar masalah (RCA) sedang berlangsung."*
2. Kumpulkan log insiden untuk analisis penyebab utama:
   ```bash
   sudo journalctl -u bot-jadwal --since "1 hour ago" > ~/storage/incident_$(date +%Y%m%d_%H%M%S).log
   ```
