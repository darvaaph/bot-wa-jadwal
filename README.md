# 🤖 Bot WhatsApp Jadwal Kuliah & Deadline Tracker

Bot WhatsApp asisten kelas cerdas yang dibangun menggunakan bahasa **Go (Golang)**, library `whatsmeow`, dan database SQLite. Bot ini berjalan secara mandiri selama 24 jam nonstop di cloud server (Azure Virtual Machine Linux).

---

## 📑 Daftar Isi
1. [Cheat Sheet Perintah Server (Azure VM)](#1--cheat-sheet-perintah-server-azure-vm)
2. [Alur Update Kodingan (Deploy Update)](#2--alur-update-kodingan-deploy-update)
3. [Panduan Testing & Development Aman (Anti-Bentrok)](#3--panduan-testing--development-aman-anti-bentrok)
4. [Cheat Sheet Perintah Bot WhatsApp](#4--cheat-sheet-perintah-bot-whatsapp)
5. [Struktur File Penting](#5--struktur-file-penting)

---

## 1. 🖥️ Cheat Sheet Perintah Server (Azure VM)

### A. Masuk ke Server dari Laptop (SSH)
Buka PowerShell atau CMD di laptop Anda:
```bash
ssh darvajago@85.211.182.189
```
* Masukkan password akun Azure Anda saat diminta.
* Untuk keluar dari server: ketik `exit` atau tekan `Ctrl + D`.

### B. Mengendalikan Bot di Server (`systemd`)
Setelah masuk ke server via SSH, gunakan perintah berikut untuk mengatur bot:

| Kebutuhan | Perintah di Server |
| :--- | :--- |
| **Cek Status Bot (Hidup/Mati/RAM)** | `sudo systemctl status bot-jadwal` |
| **Nyalakan Bot** | `sudo systemctl start bot-jadwal` |
| **Matikan Bot Sementara** | `sudo systemctl stop bot-jadwal` |
| **Restart Bot (setelah upload file baru)** | `sudo systemctl restart bot-jadwal` |
| **Melihat Log Chat / Aktivitas Realtime** | `journalctl -u bot-jadwal -f` |

> 💡 **Tips Navigasi Terminal:**
> * Saat melihat status (`systemctl status`) dan tertahan di tulisan `(END)`, tekan tombol **`q`** di keyboard untuk keluar.
> * Saat melihat log siaran langsung (`journalctl -f`), tekan **`Ctrl + C`** untuk berhenti mengintip log.

---

## 2. 🚀 Alur Update Kodingan (Deploy Update)

Jika Anda menambah fitur baru atau mengubah kodingan di laptop:

### Langkah 1: Compile untuk Linux (di PowerShell Laptop)
Buka PowerShell di folder proyek `F:\Project\bot-jadwal`:
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bot-jadwal-linux .
```

### Langkah 2: Kirim Hanya File Aplikasinya ke Server (di PowerShell Laptop)
```powershell
scp bot-jadwal-linux darvajago@85.211.182.189:~/bot-jadwal
```
*(Perintah ini langsung menimpa file aplikasi lama tanpa menyentuh atau merusak database tugas/jadwal di server).*

### Langkah 3: Restart Bot di Server (di Terminal SSH)
```bash
sudo systemctl restart bot-jadwal
```
Update langsung aktif dalam 5 detik!

---

## 3. 🛡️ Panduan Testing & Development Aman (Anti-Bentrok)

> ⚠️ **PERINGATAN PENTING TENTANG LOGOUT:**  
> **JANGAN PERNAH menekan "Keluar / Logout" di WhatsApp HP pada sesi bot.**  
> Jika Anda logout resmi di WA, sesi login di server Azure akan ikut terhapus permanen dan Anda harus scan QR ulang.

### Cara Testing Live yang Benar (Jika Pakai 1 Nomor WA yang Sama):
1. **Matikan server sementara:**
   ```bash
   sudo systemctl stop bot-jadwal
   ```
2. **Jalankan & tes kodingan di laptop:**
   ```powershell
   go run .
   ```
   Chat bot dari HP untuk menguji fitur baru sampai berhasil.
3. **Matikan bot di laptop:** Tekan `Ctrl + C` di PowerShell laptop.
4. **Kirim update ke server & nyalakan kembali:**
   ```powershell
   $env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bot-jadwal-linux .
   scp bot-jadwal-linux darvajago@85.211.182.189:~/bot-jadwal
   ```
   Lalu di server:
   ```bash
   sudo systemctl start bot-jadwal
   ```

### Cara Testing Logika Tanpa Konek WhatsApp (Unit Test Offline):
Jalankan perintah ini di laptop Anda kapan saja:
```powershell
go test -v ./...
```
Perintah ini akan menguji seluruh logika perintah, jadwal, dan kalkulasi tugas secara instan tanpa perlu koneksi internet atau WhatsApp.

---

## 4. 📱 Cheat Sheet Perintah Bot WhatsApp

*(Di grup gunakan prefix `!`, `/`, atau `#`. Di chat pribadi/DM bisa langsung ketik tanpa prefix).*

### A. Jadwal Kuliah
* `!jadwal` atau `!jadwal hari ini` ➔ Jadwal kuliah hari ini.
* `!jadwal besok` ➔ Jadwal kuliah besok.
* `!jadwal sekarang` ➔ Mata kuliah yang sedang berlangsung saat ini.
* `!jadwal senin` s/d `!jadwal jumat` ➔ Jadwal hari tertentu.
* `!jadwal mingguan` ➔ Jadwal lengkap dari Senin sampai Jumat.

### B. Manajemen Tugas (Deadline Tracker)
* `!tugas` ➔ Menampilkan daftar seluruh tugas aktif beserta hitung mundur (*countdown*).
* `!tugas tambah <Nama Tugas> | <Matkul> | <Deadline>` ➔ Menambah tugas baru.
  * *Contoh:* `!tugas tambah Laporan Praktikum | Basis Data | 12/09 23:59`
* `!tugas selesai <ID>` ➔ Menandai tugas sebagai selesai.
* `!tugas hapus <ID>` ➔ Menghapus tugas dari database.

### C. Pengingat Otomatis & Pengaturan Kelas
* `!reminder on` / `!reminder off` ➔ Mengaktifkan / mematikan broadcast jadwal otomatis setiap pagi pukul 06:30 WIB.
* `!kelas` ➔ Melihat daftar kelas yang tersedia di sistem.
* `!setkelas <KODE_KELAS>` ➔ Mengatur kelas untuk grup tersebut (contoh: `!setkelas D4-TI-SMT3-A`).

---

## 5. 📂 Struktur File Penting

* `bot-jadwal` (Linux Binary): File aplikasi utama yang berjalan di server.
* `sesi_bot.db`: Database SQLite penyimpanan token sesi login WhatsApp WhatsMeow.
* `tugas.db`: Database SQLite utama penyimpanan tugas, setting grup, dan jadwal pengganti sementara.
* `jadwal.json` & `data/jadwal/*.json`: File konfigurasi jadwal mata kuliah per kelas.
* `reminder_groups.json`: Daftar ID grup WhatsApp yang mengaktifkan pengingat jadwal pagi.
* `/etc/systemd/system/bot-jadwal.service`: File konfigurasi background service di Linux.
