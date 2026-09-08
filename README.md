# 🤖 Bot WhatsApp Jadwal Kuliah & Deadline Tracker

Bot WhatsApp asisten kelas cerdas yang dibangun menggunakan bahasa **Go (Golang)**, library `whatsmeow`, dan database SQLite. Bot ini berjalan secara mandiri selama 24 jam nonstop di cloud server (**Microsoft Azure VM Linux**).

---

## 📌 Peta Panduan Cepat
* [1. Di Mana Saya Mengetik Perintah? (Penting!)](#1-di-mana-saya-mengetik-perintah-penting)
* [2. Alur Update Kodingan (Hanya 3 Langkah Mudah)](#2-alur-update-kodingan-hanya-3-langkah-mudah)
* [3. Skenario Update Lainnya (Jadwal & Jam Reminder)](#3-skenario-update-lainnya)
* [4. Tabel Perintah Kendali Server (Systemd)](#4-tabel-perintah-kendali-server-systemd)
* [5. Cara Testing Aman di Laptop (Anti-Bentrok)](#5-cara-testing-aman-di-laptop-anti-bentrok)
* [6. Cheat Sheet Perintah Chat Bot WhatsApp](#6-cheat-sheet-perintah-chat-bot-whatsapp)

---

## 1. ⚠️ Di Mana Saya Mengetik Perintah? (Penting!)

Jangan sampai tertukar antara terminal laptop dan terminal server:

| Ikon | Tempat | Cara Membuka | Keterangan |
| :---: | :--- | :--- | :--- |
| 💻 | **Laptop (Windows)** | Buka **PowerShell** di laptop, masuk ke folder project (`cd F:\Project\bot-jadwal`). | Untuk compile program, testing lokal, dan mengirim file (`scp`). |
| ☁️ | **Server (Linux)** | Di PowerShell/CMD ketik: `ssh darvajago@85.211.182.189`. | Untuk mengontrol bot, restart, dan cek status via `systemctl`. |

---

## 2. 🚀 Alur Update Kodingan (Hanya 3 Langkah Mudah)

Gunakan alur ini setiap kali Anda selesai menambah fitur atau mengedit file kodingan `.go` di laptop:

### Langkah 1: Compile untuk Linux
💻 **Ketik di PowerShell Laptop:**
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bot-jadwal .
```
*(Perintah ini menghasilkan file aplikasi Linux bernama `bot-jadwal`).*

### Langkah 2: Kirim File ke Server
💻 **Ketik di PowerShell Laptop:**
```powershell
scp bot-jadwal darvajago@85.211.182.189:~/
```
*(Masukkan password server Anda. File lama di server otomatis tertimpa file baru).*

### Langkah 3: Restart Bot di Server
☁️ **Ketik di Terminal SSH Server:**
```bash
chmod +x bot-jadwal
sudo systemctl restart bot-jadwal
```
🎉 **Selesai!** Bot di server langsung memakai kodingan terbaru dalam hitungan detik.

---

## 3. 📂 Skenario Update Lainnya

### A. Jika Hanya Mengubah Jadwal Kuliah (`data/jadwal/*.json`)
*(Tidak perlu compile Go, cukup kirim foldernya saja)*:
1. 💻 **Di Laptop (PowerShell):**
   ```powershell
   scp -r data/jadwal darvajago@85.211.182.189:~/data/
   ```
2. ☁️ **Di Server (SSH):**
   ```bash
   sudo systemctl restart bot-jadwal
   ```

### B. Jika Hanya Mengubah Jam Pengingat (`reminder_groups.json`)
1. 💻 **Di Laptop (PowerShell):**
   ```powershell
   scp reminder_groups.json darvajago@85.211.182.189:~/
   ```
2. ☁️ **Di Server (SSH):**
   ```bash
   sudo systemctl restart bot-jadwal
   ```

---

## 4. 🛠️ Tabel Perintah Kendali Server (`systemd`)

Ketik perintah-perintah ini di **☁️ Terminal SSH Server**:

| Kebutuhan | Perintah di Server | Penjelasan |
| :--- | :--- | :--- |
| **Cek Status Bot** | `sudo systemctl status bot-jadwal` | Melihat apakah bot sedang hidup (hijau), mati, atau error. *(Tekan tombol **`q`** untuk keluar dari tampilan).* |
| **Restart Bot** | `sudo systemctl restart bot-jadwal` | Mematikan lalu langsung menyalakan bot lagi (dipakai setelah update file). |
| **Matikan Bot** | `sudo systemctl stop bot-jadwal` | Mematikan bot sementara (misal saat mau testing di laptop). |
| **Nyalakan Bot** | `sudo systemctl start bot-jadwal` | Menyalakan bot kembali. |
| **Intip Chat Realtime** | `journalctl -u bot-jadwal -f` | Menonton log pesan masuk/keluar secara live. *(Tekan **`Ctrl + C`** untuk keluar).* |
| **Keluar dari Server** | `exit` | Menutup sambungan SSH dan kembali ke laptop. |

---

## 5. 🛡️ Cara Testing Aman di Laptop (Anti-Bentrok)

> ⚠️ **ATURAN EMAS:**  
> **JANGAN PERNAH menekan tombol "Keluar / Logout" di WhatsApp HP pada sesi bot.**  
> Jika Anda logout di WA, sesi login di server Azure akan ikut terhapus permanen dan harus scan QR ulang.

### Alur Testing Live (Jika Menggunakan 1 Nomor WhatsApp yang Sama):
1. ☁️ **Di Server:** Matikan bot sementara agar tidak berebut sesi:
   ```bash
   sudo systemctl stop bot-jadwal
   ```
2. 💻 **Di Laptop:** Nyalakan dan uji coba di PowerShell:
   ```powershell
   go run .
   ```
   Lakukan testing chat ke bot dari HP sampai fitur terbukti berjalan lancar.
3. 💻 **Di Laptop:** Matikan bot laptop dengan menekan **`Ctrl + C`**.
4. 💻 **Di Laptop:** Compile dan upload hasil kodingan baru:
   ```powershell
   $env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bot-jadwal .
   scp bot-jadwal darvajago@85.211.182.189:~/
   ```
5. ☁️ **Di Server:** Nyalakan kembali bot server:
   ```bash
   sudo systemctl start bot-jadwal
   ```

### Testing Logika Cepat Tanpa WhatsApp (Unit Test Offline):
💻 **Di Laptop:**
```powershell
go test -v ./...
```
Menguji seluruh rumus waktu, jadwal, dan logika tugas dalam 1 detik tanpa perlu membuka WhatsApp.

---

## 6. 📱 Cheat Sheet Perintah Chat Bot WhatsApp

*(Di grup gunakan prefix `!`, `/`, atau `#`. Di chat pribadi/DM bisa langsung ketik tanpa prefix).*

### A. Jadwal Perkuliahan
* `!jadwal` atau `!jadwal hari ini` ➔ Jadwal kuliah hari ini.
* `!jadwal besok` ➔ Jadwal kuliah besok.
* `!jadwal sekarang` ➔ Mata kuliah yang sedang berlangsung saat ini.
* `!jadwal senin` s/d `!jadwal jumat` ➔ Jadwal kuliah hari tertentu.
* `!seminggu` ➔ Jadwal lengkap Senin sampai Jumat.

### B. Deadline Tracker (Tugas Kuliah)
* `!tugas` ➔ Menampilkan seluruh tugas aktif dengan countdown waktu.
* `!tugas tambah <Matkul> | <Deskripsi Tugas> | <Deadline>` ➔ Menambah tugas baru (mendukung sesi Teori & Praktikum).  
  *Contoh:* `!tugas tambah SBD praktikum | Laporan Modul 3 | 12/09 23:59` atau `!tugas tambah Alin teori | Resume Bab 2 | Besok 14:00`
* `!tugas selesai <ID>` ➔ Menandai tugas telah selesai.
* `!tugas hapus <ID>` ➔ Menghapus tugas dari database.

### C. Pengingat Otomatis & Pengaturan Kelas
* `!reminder on` / `!reminder off` ➔ Mengaktifkan / mematikan pesan jadwal otomatis setiap pagi pukul **06:00 WIB**.
* `!reminder` ➔ Melihat status pengingat di grup saat ini.
* `!kelas` ➔ Melihat daftar kelas yang tersedia di sistem.
* `!setkelas <KODE_KELAS>` ➔ Mengatur kelas untuk grup tersebut (contoh: `!setkelas D4-TI-SMT3-A`).

### D. Tautan Penting Kelas (Drive / Zoom)
* `!link` atau `!tautan` ➔ Menampilkan seluruh tautan penting kelas yang dikelompokkan per kategori.
* `!drive` ➔ Shortcut instan tautan Google Drive / OneDrive materi kuliah.
* `!zoom` atau `!gmeet` ➔ Shortcut instan tautan kuliah daring aktif.
* `!link cari <kata>` ➔ Mencari tautan berdasarkan judul atau deskripsi.
* `!link tambah <Judul> | <URL> (| <Catatan>)` ➔ Menambah tautan baru (*Khusus Admin di grup*).  
  *Contoh:* `!link tambah Drive Materi | https://s.id/drive-d4a | Folder lengkap`
* `!link hapus <ID>` ➔ Menghapus tautan (*Khusus Admin di grup*).

