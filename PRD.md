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
* ✅ **Deadline Tracker & Pengingat Tugas Otomatis (`!tugas` - SQLite `tugas.db`):**
  * Pencatatan tugas perkuliahan dengan parser tenggat waktu fleksibel (misal: `hari ini 23:59`, `besok 14:00`, `Jumat 23:59`).
  * Badge urgensi otomatis (*countdown*): `🚨 DEADLINE HARI INI`, `⚠️ DEADLINE BESOK (H-1)`, `⚠️ H-X`.
  * Filter cepat: `!tugas hari ini` dan `!tugas besok`.
  * Tugas di grup tetap terpajang sampai tenggatnya lewat (auto-archive setelah H+2) sehingga mahasiswa yang belum mengumpulkan tidak kehilangan info tugas.
  * Role-based access control (RBAC): Di grup hanya Admin grup (Komti/Wakil) yang dapat menambah, menyelesaikan, atau menghapus tugas. Di DM bebas untuk to-do list pribadi.
  * Anti-duplikasi tugas cerdas untuk mencegah entri ganda.
  * Integrasi otomatis peringatan tugas mendesak (*urgent*) ke dalam pesan pengingat jadwal pagi pukul 06:30 WIB.
  * Filter tugas per mata kuliah (`!tugas [matkul]`, contoh: `!tugas sbd`, `!tugas aljabar`).
  * Perpanjangan / edit tenggat tugas langsung (`!tugas edit [ID] | [tenggat]`).
  * Riwayat & arsip rekam jejak tugas semesteran yang telah selesai (`!tugas riwayat` / `!tugas arsip`).
* ✅ **Jadwal Pengganti Sementara (*Schedule Overrides* - `override.go`):**
  * Pindah jam/hari kuliah sementara (`!pindah`), kuliah kosong/ditiadakan (`!kosong`), dan kuliah pengganti di hari libur (`!kuliahganti`).
  * Peringatan bentrok jadwal otomatis (*conflict warning*) saat pemindahan jadwal dengan opsi paksa (`paksa`).
  * Fitur pengumuman libur seharian (`!libur`) yang terintegrasi dengan ucapan selamat libur pada pengingat pagi otomatis.
  * Manajemen pembatalan instan (`!jadwalganti` & `!batalganti [ID]`).
* ✅ **Kestabilan & Keamanan Sistem (*Technical Reliability*):**
  * Pembersihan database saat bot dimatikan (*Graceful Shutdown* pada `Ctrl + C` / SIGTERM) untuk mencegah database lock & WAL leak di Windows.
  * Ketahanan sambungan internet (*Auto-Reconnect Resilience*) dengan background watchdog supervisor dan algoritma Exponential Backoff.
  * Rangkaian pengujian unit test otomatis 100% lulus (**PASS**).

## 4. Roadmap & Rencana Pengembangan Selanjutnya (To-Do List)
Daftar rencana pengembangan fitur mikro (*Quality of Life*) dan jangka menengah dicatat secara rinci di:
👉 **[TODO.md](file:///f:/Project/bot-jadwal/TODO.md)**

### Ringkasan Rencana Fitur:
1. 🔗 **Tautan Penting Kelas (`!link` / `!drive`):** Direktori link Google Drive materi, Zoom perkuliahan, dan presensi SIAKAD.
2. ⏰ **Kustomisasi Jam Pengingat Pagi (`!reminder jam [HH:MM]`):** Waktu broadcast pengingat pagi yang dapat disesuaikan per grup.
3. 📋 **Format Teks Bersih Siap Salin (`!salin` / `!rekap`):** Format minimalis siap forward untuk grup angkatan atau dosen.
4. 📢 **Papan Pengumuman Komti (`!info` / `!pengumuman`):** Pin pesan penting mendadak dari dosen agar tidak tenggelam di grup.
5. 🎯 **Skala Jurusan / Multi-Tenant (`!setkelas`):** Dukungan jadwal multi-kelas (D4-1A, D4-3A, dll.) — *Selesai 100%*.
6. 🖥️ **Web Admin Dashboard:** Antarmuka web visual untuk monitoring WhatsApp, manajemen tugas, dan editor jadwal. Dokumen spesifikasi desain lengkap dapat dilihat pada:
   👉 **[DASHBOARD_PRD.md](file:///f:/Project/bot-jadwal/DASHBOARD_PRD.md)**
