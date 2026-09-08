# 🤖 Bot WhatsApp Jadwal Kuliah & Asisten Mahasiswa

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Library](https://img.shields.io/badge/WhatsApp-whatsmeow-25D366?style=flat&logo=whatsapp)](https://github.com/tulir/whatsmeow)
[![Database](https://img.shields.io/badge/Database-SQLite_WAL-003B57?style=flat&logo=sqlite)](https://sqlite.org)
[![Server](https://img.shields.io/badge/Cloud-Microsoft_Azure-0078D4?style=flat&logo=microsoftazure)](https://azure.microsoft.com)

Bot WhatsApp asisten kelas cerdas yang dirancang untuk membantu mahasiswa mengecek jadwal kuliah harian/mingguan, melacak tenggat waktu tugas (*deadline tracker*), mengelola tautan penting kelas (*link manager*), mengakomodasi jadwal pengganti sementara (*overrides*), dan mengirimkan broadcast pengingat pagi otomatis setiap hari kuliah pukul **06:00 WIB**.

---

## 📚 Navigasi Dokumentasi Tim

Dokumentasi proyek ini telah dipisahkan secara modular agar rapi dan mudah dibaca oleh setiap anggota tim:

* ☁️ **[DEPLOYMENT.md](file:///f:/Project/bot-jadwal/DEPLOYMENT.md) ➔ Panduan Server & Operasi DevOps**  
  *(Wajib dibaca bagi pengelola server: IP Azure, akses SSH, alur compile Linux, update server, perintah systemd, dan konfigurasi timezone).*
* 📖 **[PANDUAN_PENGGUNAAN.md](file:///f:/Project/bot-jadwal/PANDUAN_PENGGUNAAN.md) ➔ Panduan Lengkap Penggunaan Bot**  
  *(Dokumentasi komprehensif seluruh fitur, sintaks perintah, format deadline natural, dan aturan hak akses admin).*
* 🏗️ **[ARCHITECTURE.md](file:///f:/Project/bot-jadwal/ARCHITECTURE.md) ➔ Cetak Biru Arsitektur Teknis**  
  *(Diagram alur pesan, integrasi SQLite WAL mode, multi-tenant class resolution, dan perancangan modul).*
* 📋 **[DASHBOARD_PRD.md](file:///f:/Project/bot-jadwal/DASHBOARD_PRD.md) ➔ PRD Web Admin Dashboard**  
  *(Rencana pengembangan dashboard web admin visual untuk manajemen jadwal dan tugas).*

---

## 🚀 Memulai Pengembangan Lokal (Local Development)

### 1. Prasyarat Sistem
* **Go (Golang):** Minimal versi 1.22 atau yang lebih baru.
* **Git:** Untuk manajemen version control.

### 2. Menjalankan di Komputer Lokal
1. Buka terminal di folder proyek:
   ```bash
   cd F:\Project\bot-jadwal
   ```
2. Unduh seluruh dependensi Go:
   ```bash
   go mod download
   ```
3. Jalankan bot:
   ```bash
   go run ./cmd/bot
   ```
4. Jika pertama kali dijalankan, terminal akan merender **QR Code**. Buka WhatsApp di HP $\rightarrow$ **Perangkat Tertaut** $\rightarrow$ Scan QR tersebut.

> ⚠️ **Catatan Penting Saat Testing:**  
> Jika bot server produksi sedang aktif, matikan bot server sementara sebelum menjalankan bot secara lokal agar tidak terjadi perebutan koneksi WhatsApp. Lihat panduan lengkapnya di [DEPLOYMENT.md](file:///f:/Project/bot-jadwal/DEPLOYMENT.md).

### 3. Menjalankan Unit Test
Proyek ini dilengkapi rangkaian pengujian unit otomatis untuk memverifikasi logika parsing jadwal, tugas, dan manajemen tautan tanpa perlu terhubung ke WhatsApp:
```bash
go test -v ./...
```

---

## 📱 Ringkasan Perintah Chat Bot (Cheat Sheet)

*(Di grup WhatsApp gunakan prefix `!`, `/`, atau `#`. Di chat pribadi/DM bisa langsung diketik tanpa prefix).*

### 1. Jadwal Perkuliahan
* `!jadwal` atau `!jadwal hari ini` ➔ Menampilkan jadwal kuliah hari ini.
* `!jadwal besok` ➔ Menampilkan jadwal kuliah besok.
* `!jadwal sekarang` ➔ Menampilkan mata kuliah yang sedang berlangsung saat ini.
* `!jadwal senin` s/d `!jadwal jumat` ➔ Menampilkan jadwal pada hari tertentu.
* `!seminggu` ➔ Menampilkan jadwal lengkap Senin sampai Jumat.
* `!dosen [nama/kode]` ➔ Mencari jadwal berdasarkan inisial atau nama dosen.
* `!ruang [nama]` ➔ Mencari jadwal berdasarkan nama ruangan/lab.

### 2. Manajemen Tugas & Deadline Tracker
* `!tugas` ➔ Menampilkan daftar tugas aktif beserta hitung mundur (*countdown*).
* `!tugas tambah <Nama Tugas> | <Matkul> | <Deadline>` ➔ Menambah tugas baru.  
  *Contoh:* `!tugas tambah Laporan Praktikum | Basis Data | 12/09 23:59`
* `!tugas selesai <ID>` ➔ Menandai tugas sebagai selesai.
* `!tugas hapus <ID>` ➔ Menghapus tugas dari sistem.

### 3. Tautan Penting Kelas (!link)
* `!link` / `!tautan` ➔ Menampilkan seluruh link penting (Drive, Zoom/Meet, Portal, Repository).
* `!drive` ➔ Menampilkan link penyimpanan materi Google Drive kelas.
* `!zoom` / `!meet` ➔ Menampilkan link perkuliahan daring aktif.
* `!link tambah <Judul> | <URL>` ➔ Menambah tautan baru *(Admin Grup)*.
* `!link hapus <ID>` ➔ Menghapus tautan *(Admin Grup)*.

### 4. Pengingat Pagi & Pengaturan Multi-Kelas
* `!reminder on` / `!reminder off` ➔ Mengaktifkan / mematikan broadcast jadwal pagi otomatis (**06:00 WIB**).
* `!reminder` ➔ Melihat status pengingat di obrolan saat ini.
* `!kelas` ➔ Menampilkan daftar kelas yang tersedia dan kelas aktif.
* `!setkelas <KODE_KELAS>` ➔ Mengatur kelas untuk grup tersebut *(Admin Grup)*.

---

## 📂 Struktur Direktori Proyek

```text
bot-jadwal/
├── cmd/
│   └── bot/
│       └── main.go          # Titik masuk utama aplikasi (bot WhatsApp & REST API server)
├── internal/
│   ├── config/              # Konfigurasi aplikasi & auto-migration file runtime
│   ├── database/            # Inisialisasi SQLite connection pool WAL mode
│   ├── util/                # Helper murni (waktu WIB, parser tanggal alami, FindDataDir)
│   ├── schedule/            # ScheduleEngine, ClassManager, & OverrideManager
│   ├── task/                # TaskManager SQLite & parser tenggat tugas
│   ├── link/                # LinkManager SQLite & categorizer tautan kelas
│   ├── chat/                # ChatSettingsManager & GroupAdminResolver
│   ├── reminder/            # Cron broadcast pagi otomatis (06:00 WIB)
│   ├── bot/                 # whatsmeow client, quoted reply, & message event dispatcher
│   └── api/                 # HTTP REST API server untuk Web Admin Dashboard
├── data/
│   ├── jadwal/              # Berkas master JSON kurikulum per kelas
│   └── jadwal.json          # Berkas kurikulum default
├── storage/                 # Direktori terisolasi untuk file runtime (di-ignore oleh git)
│   ├── tugas.db             # Database SQLite aplikasi (tugas, link, setting kelas, override)
│   ├── sesi_bot.db          # Database sesi login WhatsMeow
│   └── reminder_groups.json # File JSON preferensi grup pengingat
├── DEPLOYMENT.md            # Dokumentasi operasional server Azure & DevOps
├── PANDUAN_PENGGUNAAN.md    # Panduan komprehensif fitur untuk pengguna
├── ARCHITECTURE.md          # Cetak biru arsitektur teknis sistem
├── DASHBOARD_PRD.md         # PRD Web Admin Dashboard
└── README.md                # Berkas ringkasan proyek ini
```

---

## 👥 Kontribusi Tim
1. Pastikan selalu membuat *branch* baru untuk fitur baru: `git checkout -b feat/nama-fitur`.
2. Selalu jalankan `go test -v ./...` sebelum melakukan *commit* atau *merge*.
3. Untuk memperbarui server produksi setelah perubahan di-*merge* ke `main`, ikuti petunjuk rilis di **[DEPLOYMENT.md](file:///f:/Project/bot-jadwal/DEPLOYMENT.md)**.
