# 📋 To-Do List & Roadmap Pengembangan Bot WhatsApp Jadwal Kuliah

Dokumen ini mencatat daftar ide dan rencana pengembangan fitur masa depan yang dirancang agar tetap **ringan, hemat sumber daya, dan memecahkan masalah nyata sehari-hari** tanpa *overengineering*.

---

## 🚀 Backlog Fitur Mikro & Quality of Life (Siap Dikerjakan Kapan Saja)

### 1. 🔗 Daftar Tautan Penting Kelas (`!link` / `!drive`)
* **Masalah:** Mahasiswa sering bertanya berulang-ulang terkait link Google Drive materi, link Zoom dosen, link presensi SIAKAD, atau repositori GitHub kelas. Deskripsi grup WhatsApp seringkali karakternya terbatas atau terpotong di ponsel tertentu.
* **Rencana Solusi:**
  * Komti/Admin dapat menambahkan tautan:
    `!link tambah Drive | https://s.id/drive-kelas`
    `!link tambah Zoom Matdis | https://zoom.us/j/123456`
  * Mahasiswa cukup mengetik `!link` atau `!drive` untuk melihat seluruh daftar tautan aktif.
* **Tingkat Kompleksitas:** 🟢 *Sangat Rendah* (File JSON atau 1 tabel SQLite kecil).

---

### 2. ⏰ Kustomisasi Jam Pengingat Pagi (`!reminder jam [HH:MM]`)
* **Masalah:** Jam pengingat pagi saat ini bersifat seragam pukul `06:30 WIB`. Pada beberapa kelas atau hari tertentu, perkuliahan dimulai pukul `07:00` (sehingga butuh reminder lebih awal di `05:45` atau `06:00`), sementara kelas lain baru mulai pukul `09:00` sehingga reminder dinilai terlalu pagi.
* **Rencana Solusi:**
  * Komti dapat mengatur jam broadcast spesifik untuk grupnya:
    `!reminder jam 06:00` atau `!reminder jam 07:15`
  * Tersimpan dinamis per grup pada `reminder_groups.json`.
* **Tingkat Kompleksitas:** 🟢 *Sangat Rendah* (Menambah kolom konfigurasi `Hour` & `Minute` per grup).

---

### 3. 📋 Format Teks Bersih Siap Salin / Forward (`!salin` / `!rekap`)
* **Masalah:** Komti sering diminta dosen atau ketua angkatan untuk meneruskan rekap jadwal kuliah atau tugas minggu ini ke grup gabungan. Format standar bot saat ini memiliki banyak tombol bantuan dan emoji interaktif yang kurang formal jika diforward mentah-mentah ke dosen.
* **Rencana Solusi:**
  * Perintah `!salin besok` atau `!rekap minggu ini`.
  * Bot merespons dengan satu blok teks minimalis, bersih, rapi, dan sopan tanpa tombol navigasi bot yang siap langsung di-forward.
* **Tingkat Kompleksitas:** 🟢 *Nol Backend Baru* (Hanya variasi string formatter).

---

### 4. 📢 Papan Pengumuman / Info Kilat Komti (`!info` / `!pengumuman`)
* **Masalah:** Pesan penting dari dosen (misal: *"Besok praktikum wajib bawa modul bab 4 cetak & jas lab"*) sering cepat tenggelam tertimbun obrolan santai dan stiker di grup.
* **Rencana Solusi:**
  * Komti memasang pengumuman aktif:
    `!info Besok wajib bawa jas lab & modul bab 4 cetak`
  * Mahasiswa kapan saja bisa mengetik `!info` untuk membaca info terakhir dari Komti.
  * Catatan ini otomatis diselipkan 1 baris pada pengingat pagi pukul 06:30 WIB.
* **Tingkat Kompleksitas:** 🟢 *Sangat Rendah* (Menyimpan 1 baris teks pengumuman aktif).

---

## 🎯 Roadmap Arsitektur Multi-Kelas (Multi-Tenant Support)

- [x] **Fase 1: Multi-JSON Loader (`ClassManager`):** *(Selesai)*
  * Direktori [data/jadwal/](file:///f:/Project/bot-jadwal/data/jadwal/) berisi [3a.json](file:///f:/Project/bot-jadwal/data/jadwal/3a.json) dan [3b.json](file:///f:/Project/bot-jadwal/data/jadwal/3b.json).
  * [ClassManager](file:///f:/Project/bot-jadwal/class_manager.go) dengan in-memory cache, normalisasi cerdas, hot-reload, fallback otomatis, dan 100% lulus unit test ([class_manager_test.go](file:///f:/Project/bot-jadwal/class_manager_test.go)).
- [x] **Fase 2: Tabel `chat_settings` di SQLite (`ChatSettingsManager`):** *(Selesai)*
  * Menyimpan relasi `scope_jid -> class_id` pada `tugas.db` melalui [chat_settings.go](file:///f:/Project/bot-jadwal/chat_settings.go).
  * Write-through cache in-memory untuk pembacaan instan O(1), aman konkurensi (`sync.RWMutex`), dan 100% lulus unit test ([chat_settings_test.go](file:///f:/Project/bot-jadwal/chat_settings_test.go)).
- [x] **Fase 3: Perintah `!setkelas` & `!daftarkelas`:** *(Selesai)*
  * Handler `HandleCommand` di [chat_settings.go](file:///f:/Project/bot-jadwal/chat_settings.go) untuk `!daftarkelas`, `!setkelas`, dan `!resetkelas`.
  * Proteksi otorisasi admin grup di WhatsApp grup & kebebasan pengaturan di DM pribadi.
  * Pembaruan tampilan menu bantuan `!menu` dan `!keyword` di [schedule.go](file:///f:/Project/bot-jadwal/schedule.go).
  * 100% lulus unit test skenario perintah ([chat_settings_test.go](file:///f:/Project/bot-jadwal/chat_settings_test.go)).
- [x] **Fase 4: Integrasi ke Dispatcher Pesan & Pengingat Pagi:** *(Selesai)*
  * Integrasi penuh resolusi jadwal dinamis per chat/grup di [main.go](file:///f:/Project/bot-jadwal/main.go) (`activeJadwal`).
  * Integrasi perintah kelas dan hot-reload seluruh kelas (`!reload`) di WhatsApp.
  * Personalisasi broadcast jadwal pagi 06:30 WIB per kelas grup pada [reminder.go](file:///f:/Project/bot-jadwal/reminder.go).
  * 100% lulus integrasi test [reminder_test.go](file:///f:/Project/bot-jadwal/reminder_test.go) dan kompilasi biner `bot-jadwal.exe` sukses.
- [x] **Ekstraksi Lengkap 19 Kelas D3 & D4 Teknik Informatika:** *(Selesai)*
  * 12 Kelas D4 Sarjana Terapan (`D4-TI-1A` s.d. `1D`, `D4-TI-3A` s.d. `3D`, `D4-TI-5A`, `5B`, `D4-TI-7A`, `7B`).
  * 7 Kelas D3 Diploma (`D3-TI-1A`, `1B`, `D3-TI-3A`, `3B`, `D3-TI-5A`, `5B`, `5C`).
  * Integrasi 47 inisial dosen (termasuk dosen umum/bahasa dan tim teaching) serta 41 mata kuliah dengan label Teori/Praktikum.
- [x] **Explicit Onboarding (Best Practice WhatsApp Bot):** *(Selesai)*
  * Status awal chat baru adalah `Belum Diatur` tanpa asumsi kelas tertentu.
  * Perintah jadwal (`!hari ini`, `!besok`, `!jadwal`, `!next`, `!matkul`, dll.), tugas (`!tugas`), dan override langsung menampilkan pesan panduan onboarding.
  * `!daftarkelas` menampilkan `📌 Kelas Aktif di Chat Ini: ⚠️ *Belum Diatur*`.
  * `!reminder on` menolak aktif sebelum kelas ditentukan oleh admin grup.
  * 17 Unit test suite **100% PASS** dan kompilasi biner `bot-jadwal.exe` sukses.
- [ ] **Cek Jadwal Lintas Kelas (*Cross-Class Peek*):**
  * Mahasiswa dapat mengintip jadwal kelas lain kapan saja (contoh: `!jadwal senin 3b` atau `!next 3b`).
- [ ] **Distribusi Dokumen Silabus / Modul Praktikum PDF (`!modul`):**
  * Bot dapat mengirimkan file dokumen praktikum secara langsung dari penyimpanan lokal server ke chat mahasiswa.
- [ ] **Web Admin Dashboard (Embedded Pure Go Server):**
  * Dokumen PRD spesifikasi lengkap untuk desainer UI/UX telah selesai dirancang di [DASHBOARD_PRD.md](file:///f:/Project/bot-jadwal/DASHBOARD_PRD.md).
  * Siap masuk ke tahap perancangan visual Figma (Moodboard, Design Tokens, Wireframe, High-Fidelity UI, Prototype).

---

## ✅ Fitur yang Sudah Selesai Diimplementasikan (Completed)

- [x] Parser Jadwal Cerdas & Pencarian Fleksibel (`!hari ini`, `!besok`, `!next`, `!matkul`, `!dosen`, `!ruang`, `!cari`).
- [x] Jadwal Pengganti Sementara (*Schedule Overrides*): `!pindah`, `!kosong`, `!kuliahganti`, `!jadwalganti`, `!batalganti`.
- [x] Deteksi Bentrok Jadwal Otomatis (*Schedule Conflict Warning*) saat pemindahan kelas dengan opsi konfirmasi paksa (`paksa`).
- [x] Fitur Pengumuman Libur Seharian (`!libur`) dengan integrasi otomatis ke ucapan libur pengingat pagi 06:30 WIB.
- [x] Pengingat Otomatis Pagi (*Morning Reminder Cron*) Senin–Jumat 06:30 WIB dengan peringatan tugas mendesak.
- [x] Deadline Tracker & Pengelola Tugas SQLite (`!tugas`, `!tugas tambah`, `!tugas edit`, `!tugas selesai`, `!tugas hapus`).
- [x] Parser Waktu Alami Indonesia (nama bulan `sep`/`september`, kata relatif `hari ini`/`besok`, format jam saja `22.22`).
- [x] Validasi Mata Kuliah Cerdas & Rekomendasi Alias Pintar (`sbd`, `aljabar`, `mtk`, `matdis`, `umum`).
- [x] Filter Tugas per Mata Kuliah (`!tugas [matkul]`, contoh: `!tugas sbd`, `!tugas aljabar`).
- [x] Riwayat & Rekam Jejak Tugas Selesai Sepanjang Semester (`!tugas riwayat` / `!tugas arsip`).
- [x] Pembersihan Database saat Bot Dimatikan (*Graceful Shutdown* pada `Ctrl + C` / SIGTERM) untuk mencegah database lock & WAL leak di Windows.
- [x] Ketahanan Sambungan Internet (*Auto-Reconnect Resilience*) dengan Watchdog Supervisor & Exponential Backoff.
- [x] Pengujian Unit Test Komprehensif (100% PASS pada seluruh 8 test suite: `TestChatSettingsManager`, `TestClassManager`, `TestDatabaseInit`, `TestOverrideManager`, `TestReminderIntegration`, `TestSchedule`, `TestTaskManager`, `TestUtils`).
