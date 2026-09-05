# Product Requirements Document (PRD): Bot WhatsApp Jadwal Kuliah Otomatis (Golang)

## 1. Latar Belakang & Permasalahan (Problem Statement)
* **Permasalahan Utama:** Jadwal kuliah yang dibagikan oleh pihak jurusan/prodi sering kali berupa gambar atau dokumen PDF dengan format penulisan yang disingkat-singkat (menggunakan kode mata kuliah dan inisial kode dosen). 
* **Dampak:** Mahasiswa sering merasa pusing, kesulitan membaca jadwal secara cepat, salah melihat ruangan, atau keliru mengenali dosen pengampu di awal semester.
* **Kebutuhan Pengguna:** Dibutuhkan sebuah sistem pencari informasi jadwal yang cepat, interaktif, dan dapat diakses langsung melalui aplikasi chat harian yang biasa digunakan tanpa harus membuka portal akademik yang lambat atau melihat gambar jadwal yang rumit.

## 2. Solusi Produk (Product Solution)
* Mengembangkan sebuah **Bot WhatsApp Otomatis** berbasis bahasa pemrograman **Go (Golang)** menggunakan *library* `whatsmeow` dan *database* lokal SQLite (`modernc.org/sqlite`).
* Bot ini berfungsi sebagai asisten virtual kelas yang dapat menerjemahkan kode-kode singkatan jadwal menjadi informasi yang *human-readable* (nama matkul lengkap, nama dosen lengkap, waktu, dan ruangan) berdasarkan perintah teks (*command*) dari pengguna.

## 3. Fitur yang Sudah Selesai Diimplementasikan (Completed Features)
* ✅ **Pemuatan Jadwal Terstruktur (`jadwal.json`):** Parsing data mata kuliah, master dosen JTK, ruangan, dan jam perkuliahan secara dinamis.
* ✅ **Pintasan Cek Hari (`!hari ini`, `!besok`, `!senin` - `!jumat`, `!seminggu`):** Akses cepat jadwal kuliah per hari atau seminggu penuh.
* ✅ **Kuliah Berikutnya / Sedang Berlangsung (`!next` / `!sekarang`):** Perhitungan waktu real-time untuk mendeteksi kuliah aktif atau kuliah berikutnya berserta sisa waktu.
* ✅ **Pencarian Lengkap (`!dosen`, `!ruang`, `!matkul`, `!cari`):** Direktori dosen, ruangan, mata kuliah, dan pencarian bebas (*fuzzy search*).
* ✅ **Pengingat Otomatis Grup (`reminder.go`):** Scheduler background yang otomatis mengirim jadwal ke grup terdaftar setiap pukul **06:30 WIB (Senin – Jumat)**.
* ✅ **Strategi Hybrid (Anti-Spam Grup & Fleksibel di DM):** Di grup wajib menggunakan prefix (`!`), di chat pribadi bebas tanpa prefix.
* ✅ **Simulasi Manusiawi (Anti-Ban):** Mengirim indikator *"sedang mengetik..."* (*Typing Presence*) dan reaksi emoji (*Message Reaction* `📅` / `⏰`).
* ✅ **Reload On-The-Fly (`!reload`):** Memperbarui data jadwal tanpa me-restart server Go.
* ✅ **Menu & Kamus Keyword Bertingkat (`!menu` & `!keyword`):** Tampilan ringkas di layar HP dan kamus keyword lengkap.

## 4. Roadmap & Rencana Pengembangan Selanjutnya (Upcoming Development)
### 🎯 Tahap 1: Arsitektur Multi-Class / Multi-Tenant (Skala Jurusan)
* **Penyimpanan Jadwal Modular:** Memisahkan data jadwal per kelas (contoh: `data/jadwal_d4_1a.json`, `data/jadwal_d4_3a.json`, `data/jadwal_d4_3b.json`).
* **Sistem Preferensi Pengguna & Grup (`!setkelas` / `!kelas`):**
  * Setiap mahasiswa atau grup kelas dapat mengunci kelasnya masing-masing (misal: grup 3A di-set `!setkelas D4-3A`, grup 3B di-set `!setkelas D4-3B`).
  * Pengingat otomatis pagi jam 06:30 akan mengirimkan jadwal sesuai kelas yang dipilih oleh grup tersebut.
* **Dukungan Cek Lintas Kelas (*On-Demand Override*):** Mahasiswa bisa mengintip jadwal kelas lain kapan saja (contoh: `!senin 3B` atau `!next 1A`).

### 🎯 Tahap 2: Manajemen Tugas & Deadline Kuliah (`!tugas`)
* Sistem pencatat tugas per mata kuliah/kelas berbasis SQLite/JSON (`!tugas tambah`, `!tugas list`, `!tugas selesai`).

### 🎯 Tahap 3: Pengiriman Dokumen Modul & Silabus PDF (`!modul`)
* Fitur pengiriman file PDF materi/silabus praktikum langsung dari penyimpanan bot ke chat mahasiswa.