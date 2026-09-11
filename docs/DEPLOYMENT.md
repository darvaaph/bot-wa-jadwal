# ☁️ Panduan Server & Deployment (DevOps Guide)
# Bot WhatsApp Jadwal Kuliah & Asisten Mahasiswa

Dokumen ini ditujukan khusus untuk pengelola server / tim yang bertugas melakukan *maintenance*, *deployment*, dan konfigurasi server cloud produksi (Microsoft Azure).

---

## 📑 Daftar Isi
1. [Informasi Server Produksi](#1-informasi-server-produksi)
2. [Cara Masuk ke Server (SSH)](#2-cara-masuk-ke-server-ssh)
3. [Alur Update Kodingan (Deploy Update)](#3-alur-update-kodingan-deploy-update)
4. [Skenario Update Jadwal & Jam Pengingat](#4-skenario-update-jadwal--jam-pengingat)
5. [Perintah Kendali Layanan (`systemd`)](#5-perintah-kendali-layanan-systemd)
6. [Panduan Testing Aman (Mencegah Bentrok Sesi)](#6-panduan-testing-aman-mencegah-bentrok-sesi)
7. [Konfigurasi Zona Waktu (WIB)](#7-konfigurasi-zona-waktu-wib)
8. [Struktur File di Server](#8-struktur-file-di-server)

---

## 1. 🖥️ Informasi Server Produksi

* **Cloud Provider:** Microsoft Azure (Azure for Students)
* **Tipe VM:** `Standard_B2ats_v2` (2 vCPU, 1 GiB RAM, AMD EPYC)
* **Sistem Operasi:** Ubuntu Server 24.04 LTS (x64)
* **Region:** Malaysia West (Kuala Lumpur)
* **Alamat IP Publik:** `<IP_SERVER_PRODUKSI>` *(Hubungi PM/DevOps Lead untuk kredensial)*
* **Port Terbuka:**
  * `22` (SSH - Akses remote terminal)
  * `80` (HTTP - Akses web dashboard)
  * `443` (HTTPS - Akses web dashboard aman)

---

## 2. 🔑 Cara Masuk ke Server (SSH)

Buka **PowerShell** atau **Terminal** di laptop:
```bash
ssh <USER_SSH>@<IP_SERVER_PRODUKSI>
```
* Masukkan password server saat diminta.
* Untuk keluar dari server: ketik `exit` atau tekan `Ctrl + D`.

---

## 3. 🚀 Alur Update Kodingan (Deploy Update)

Gunakan alur 3 langkah ini setiap kali ada pembaruan fitur atau perbaikan kode Go (`.go`):

### Langkah 1: Compile untuk Linux (di PowerShell Laptop)
Pastikan berada di folder proyek `F:\Project\bot-jadwal`:
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bot-jadwal ./cmd/bot
```
*(Perintah ini menghasilkan file biner mandiri `bot-jadwal` khusus arsitektur Linux x64).*

### Langkah 2: Kirim File ke Server (di PowerShell Laptop)
```powershell
scp bot-jadwal <USER_SSH>@<IP_SERVER_PRODUKSI>:~/bot-jadwal.new
```
> 💡 **Kenapa diunggah sebagai `bot-jadwal.new`?**  
> Karena bot sedang aktif berjalan di memori server. Menimpa file biner yang sedang berjalan secara langsung di Linux dapat memicu error *"Text file busy"*. Mengunggahnya sebagai `.new` lalu me-*replace*-nya via perintah `mv` adalah praktik terbaik (aman 100%).

### Langkah 3: Pasang & Restart di Server (di Terminal SSH)
Masuk ke terminal server (`ssh <USER_SSH>@<IP_SERVER_PRODUKSI>`), lalu jalankan:
```bash
mv bot-jadwal.new bot-jadwal
chmod +x bot-jadwal
sudo systemctl restart bot-jadwal
```

Verifikasi status bot:
```bash
sudo systemctl status bot-jadwal
```
*(Pastikan statusnya berwarna hijau: `Active: active (running)`).*

---

## 4. 📂 Skenario Update Jadwal & Jam Pengingat

### A. Jika Hanya Mengubah Jadwal Kuliah (`data/jadwal/*.json`)
*(Tidak perlu compile Go ulang, cukup kirim foldernya)*:
1. 💻 **Di Laptop (PowerShell):**
   ```powershell
   scp -r data/jadwal <USER_SSH>@<IP_SERVER_PRODUKSI>:~/data/
   ```
2. ☁️ **Di Server (SSH):**
   ```bash
   sudo systemctl restart bot-jadwal
   ```
   *(Atau dari WhatsApp cukup ketik `!reload` jika fitur reload aktif).*

### B. Jika Hanya Mengubah Jam Pengingat (`reminder_groups.json`)
1. 💻 **Di Laptop (PowerShell):**
   ```powershell
   scp reminder_groups.json <USER_SSH>@<IP_SERVER_PRODUKSI>:~/
   ```
2. ☁️ **Di Server (SSH):**
   ```bash
   sudo systemctl restart bot-jadwal
   ```

---

## 5. 🛠️ Perintah Kendali Layanan (`systemd`)

Bot dikelola sebagai background service Linux bernama `bot-jadwal.service`. File konfigurasinya berada di `/etc/systemd/system/bot-jadwal.service`.

| Kebutuhan | Perintah di Server | Penjelasan |
| :--- | :--- | :--- |
| **Cek Status Bot** | `sudo systemctl status bot-jadwal` | Cek apakah bot aktif (hijau), mati, atau error. *(Tekan **`q`** untuk keluar).* |
| **Restart Bot** | `sudo systemctl restart bot-jadwal` | Memuat ulang aplikasi bot (setelah upload file baru). |
| **Matikan Bot** | `sudo systemctl stop bot-jadwal` | Mematikan bot sementara (misal saat testing lokal). |
| **Nyalakan Bot** | `sudo systemctl start bot-jadwal` | Menyalakan kembali bot di latar belakang. |
| **Intip Log Realtime** | `journalctl -u bot-jadwal -f` | Menonton log pesan masuk dan aktivitas bot live. *(Tekan **`Ctrl + C`** untuk keluar).* |

---

## 6. 🛡️ Panduan Testing Aman (Mencegah Bentrok Sesi)

> ⚠️ **PERINGATAN SANGAT PENTING TENTANG LOGOUT:**  
> **JANGAN PERNAH menekan tombol "Keluar / Logout" di aplikasi WhatsApp HP pada sesi bot.**  
> Menekan logout di WhatsApp akan mencabut kunci enkripsi akun secara permanen dari server Meta, sehingga sesi di server cloud akan mati total dan mewajibkan scan QR ulang.

### Alur Testing Fitur Baru (Jika Menggunakan Nomor WhatsApp yang Sama):
1. ☁️ **Di Server:** Matikan bot sementara agar tidak berebut koneksi soket:
   ```bash
   sudo systemctl stop bot-jadwal
   ```
2. 💻 **Di Laptop:** Jalankan bot secara lokal:
   ```powershell
   go run .
   ```
   Lakukan testing chat ke bot dari HP sampai fitur terbukti bekerja normal.
3. 💻 **Di Laptop:** Matikan bot laptop dengan menekan **`Ctrl + C`**.
4. 💻 **Di Laptop:** Compile & kirim file terbaru ke server:
   ```powershell
   $env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bot-jadwal .
   scp bot-jadwal <USER_SSH>@<IP_SERVER_PRODUKSI>:~/bot-jadwal.new
   ```
5. ☁️ **Di Server:** Pasang & nyalakan kembali bot server:
   ```bash
   mv bot-jadwal.new bot-jadwal
   chmod +x bot-jadwal
   sudo systemctl start bot-jadwal
   ```

### Testing Logika Offline (Disarankan):
Untuk menguji parsing waktu, format jadwal, atau modul tugas, jalankan unit test tanpa perlu membuka WhatsApp:
```powershell
go test -v ./...
```

---

## 7. 🕒 Konfigurasi Zona Waktu (WIB)

Server Azure secara default menggunakan zona waktu UTC. Server ini sudah dikonfigurasi ke **WIB (Asia/Jakarta)** agar pengingat jadwal pagi (pukul 06:00 WIB) dan kalkulasi hari kuliah sinkron dengan jam lokal mahasiswa.

Jika suatu saat zona waktu ter-reset:
```bash
sudo timedatectl set-timezone Asia/Jakarta
sudo systemctl restart bot-jadwal
```
Periksa dengan mengetik: `date` (harus berakhiran `WIB`).

---

## 8. 📁 Struktur File di Server

Di direktori home server (`/home/<USER_SSH>/`):
```text
/home/<USER_SSH>/
├── bot-jadwal             # File biner aplikasi hasil compile (Go)
├── data/
│   ├── jadwal/            # 19 file master jadwal JSON per kelas
│   └── jadwal.json        # Konfigurasi kurikulum default
└── storage/               # Direktori runtime state (terisolasi)
    ├── reminder_groups.json # Daftar JID grup terdaftar pengingat pagi
    ├── sesi_bot.db        # Database SQLite sesi login WhatsMeow
    └── tugas.db           # Database SQLite tugas, setting kelas, link & override
```
