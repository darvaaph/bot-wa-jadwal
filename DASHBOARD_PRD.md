# 📄 Product Requirements Document (PRD)
# Web Admin Dashboard — Bot WhatsApp Jadwal Kuliah & Multi-Kelas

* **Status:** Draft Ready for Design & Specification Review
* **Target Release:** v2.0
* **Target Audience:** UI/UX Designer, Frontend Developer, Bot Maintainer / Komti Kelas
* **Tech Stack Alignment:** Pure Go Embedded Server (`embed.FS`, `net/http`), Vanilla CSS Design System, Responsive HTML5 & Vanilla JS (Zero node_modules on production binary).

---

## 1. 🎯 Visi Produk & Latar Belakang

### 1.1 Problem Statement
1. **Keterbatasan Interaksi Berbasis Chat WhatsApp:**
   * Mengetik perintah panjang seperti `!pindah aljabar | 2026-09-08 13:00 | D102` atau mengedit file `jadwal.json` secara manual di server rentan terhadap *typo*, salah kurung kurawal JSON, dan memerlukan akses SSH/terminal yang tidak ramah bagi sebagian Komti (Ketua Tingkat).
2. **Visibilitas Status Sistem yang Buruk:**
   * Saat koneksi WhatsApp putus atau meminta scan QR Code baru, bot administrator harus membuka terminal server untuk melihat QR terminal yang sering terpotong atau sulit dipindai dari layar laptop.
3. **Pengelolaan Multi-Kelas:**
   * Dengan hadirnya arsitektur Multi-Kelas, administrator perlu melihat gambaran utuh: berapa banyak grup yang terdaftar, kelas mana saja yang aktif, dan memantau tugas antar kelas dalam satu tampilan visual yang terpadu.

### 1.2 Solusi Produk
Membangun **Admin Dashboard Berbasis Web** yang disematkan (*embedded*) langsung di dalam biner Go. Dashboard ini menyediakan antarmuka visual modern, responsif (HP & Desktop), berkecepatan tinggi, dan aman untuk mengelola jadwal, tugas, jadwal pengganti sementara, serta konektivitas bot WhatsApp tanpa memerlukan aplikasi eksternal tambahan.

---

## 2. 👥 Persona Pengguna & Alur Kerja Utama

| Persona | Profil & Kebutuhan | Perangkat Utama | Skenario Pemakaian |
| :--- | :--- | :--- | :--- |
| **Komti / PJ Mata Kuliah** | Mahasiswa yang bertanggung jawab mengabari teman sekelas. Butuh cara cepat (< 30 detik) mencatat tugas dosen atau memindahkan jam kuliah saat sedang di kampus. | Smartphone (Mobile Web) / Laptop | Menggeser jam kuliah praktikum saat dosen mengabari mendadak via WhatsApp, lalu mengecek kalender jadwal pengganti. |
| **Bot Maintainer / Host** | Pengelola server bot (pemilik nomor bot). Butuh memantau status koneksi, scan QR login WhatsApp, dan menambah kelas baru untuk jurusan/angkatan lain. | Laptop / Desktop | Membuka dashboard saat setup awal semester, scan QR login, menambah kurikulum kelas baru, dan mengecek log broadcast pengingat pagi. |

---

## 3. 🎨 Panduan Desain Sistem (*Design System Tokens*)

Untuk memastikan konsistensi dan estetika premium yang dirancang oleh UI/UX designer, dashboard mengadopsi prinsip **Modern Academic Dark Interface** yang bersih, berfokus pada hierarki informasi, dan ramah mata.

### 3.1 Palet Warna (Design Tokens)

```css
:root {
  /* Surface & Background */
  --bg-primary: #0b0f19;         /* Latar belakang utama (Deep Slate/Navy) */
  --bg-surface: #111827;         /* Latar belakang kartu/panel (Dark Neutral) */
  --bg-surface-elevated: #1f2937;/* Modal, dropdown, hover card */
  --border-subtle: rgba(255, 255, 255, 0.08); /* Garis batas kartu & tabel */
  --border-focus: #3b82f6;       /* Garis fokus input */

  /* Typography */
  --text-primary: #f9fafb;       /* Teks judul & data utama */
  --text-secondary: #9ca3af;     /* Label, metadata, subtitle */
  --text-muted: #6b7280;         /* Hint, placeholder, footer */

  /* Brand & Aksen */
  --brand-primary: #10b981;      /* Emerald Green (Karakter WhatsApp & Sukses) */
  --brand-primary-hover: #059669;
  --brand-secondary: #3b82f6;    /* Royal Blue (Aksi sekunder & filter) */
  
  /* Status & Urgensi */
  --status-urgent: #ef4444;       /* Merah: Deadline Hari Ini / Kelas Kosong */
  --status-warning: #f59e0b;      /* Kuning/Amber: Deadline Besok / Jam Pindah */
  --status-success: #10b981;      /* Hijau: Online, Tugas Selesai */
  --status-info: #6366f1;         /* Indigo: Kuliah Pengganti Extra */
}
```

### 3.2 Tipografi & Skala Spasi
* **Font Utama:** `Plus Jakarta Sans` atau `Inter` (Font sans-serif modern yang nyaman dibaca pada teks padat).
* **Font Monospace:** `JetBrains Mono` atau `Fira Code` (Untuk kode matkul, jam perkuliahan, dan log konsol).
* **Skala Spasi (Grid 8-point):**
  * `4px` (xxs), `8px` (xs), `12px` (sm), `16px` (md), `24px` (lg), `32px` (xl), `48px` (xxl).
* **Radius Sudut (*Border Radius*):**
  * Kartu & Panel: `12px`
  * Tombol & Input: `8px`
  * Badge & Tag Status: `9999px` (*Full Pill*)

---

## 4. 🗺️ Informasi Arsitektur & Peta Halaman (*Sitemap*)

```
Admin Dashboard Shell
├── 1. 📊 Telemetry & Overview Hub (Dashboard Utama)
├── 2. 📅 Jadwal Kuliah & Editor Multi-Kelas
├── 3. 🔄 Kalender Jadwal Pengganti & Libur (Overrides)
├── 4. 📝 Papan Manajemen Tugas & Arsip Tenggat
├── 5. 👥 Grup WhatsApp & Broadcast Pengingat Pagi
├── 6. 📱 Gateway Status WhatsApp & Scan QR Code
└── 7. 🔒 Kunci Keamanan Akses (PIN/Session Lock)
```

---

## 5. 📐 Spesifikasi Detail Layar (*Screen-by-Screen Specification*)

### Layar 1: 📊 Telemetry & Overview Hub (Dashboard Utama)

**Tujuan:** Memberikan gambaran kilat kesehatan bot, jadwal perkuliahan hari ini, dan tugas yang paling mendesak dalam hitungan detik.

#### Komponen UI & Layout:
1. **Top Header Bar:**
   * Nama Kampus/Jurusan aktif.
   * Dropdown Switcher Kelas: Mengganti konteks data (contoh: `Kelas 3A` ▾, `Kelas 3B` ▾).
   * Status Pill Koneksi WhatsApp: Dot hijau berkedip `🟢 Terhubung (+62 812-xxxx)` atau merah `🔴 Terputus`.
   * Tombol Aksi Cepat: `[+ Tugas Baru]`, `[+ Pindah Jam]`, `[🔄 Reload Data]`.
2. **Row KPI Telemetri (4 Kartu Stat):**
   * **Kartu 1: Status Kuliah Hari Ini** (Contoh: `3 Sesi Perkuliahan` • *Sesi berikutnya: 13:00 WIB*).
   * **Kartu 2: Tugas Aktif** (Contoh: `4 Tugas Belum Selesai` • *🚨 1 Deadline Hari Ini*).
   * **Kartu 3: Jadwal Pengganti Aktif** (Contoh: `2 Perubahan Minggu Ini`).
   * **Kartu 4: Grup Terdaftar** (Contoh: `5 Grup WhatsApp` • *Reminder 06:30 Aktif*).
3. **Widget "Next Lecture Banner" (Sesi Kuliah Aktif/Berikutnya):**
   * Menampilkan kartu sorotan: Nama Mata Kuliah, Dosen, Jam, Ruang, dan *Countdown Timer* (cth: *"Sedang berlangsung (Sisa 45 menit)"* atau *"Mulai dalam 1 jam 15 menit"*).
4. **Two-Column Split View:**
   * **Kolom Kiri (60%):** Ringkasan Jadwal Kuliah Hari Ini & Besok dengan status normal vs dipindahkan.
   * **Kolom Kanan (40%):** Widget *Urgent Task Radar* (Daftar tugas terdekat dengan badge hitung mundur tenggat).

---

### Layar 2: 📅 Jadwal Kuliah & Editor Multi-Kelas

**Tujuan:** Menampilkan visualisasi kurikulum mingguan (Senin s.d. Jumat) dan memungkinkan Komti memodifikasi jam, dosen, ruangan, atau menambah kelas baru tanpa menyentuh file JSON.

#### Komponen UI & Layout:
1. **Class Management Bar:**
   * Tab Pemilih Kelas: `[ 3A (Aktif) ]` `[ 3B ]` `[ + Tambah Kelas Baru ]`.
   * Tombol **"Unggah File JSON"** & **"Ekspor JSON"**.
2. **Weekly Timetable Grid (Visual Senin - Jumat):**
   * Kolom Hari: Senin, Selasa, Rabu, Kamis, Jumat.
   * Kartu Matkul pada setiap hari:
     * Label Jam (cth: `07:00 - 08:40 WIB`).
     * Judul Mata Kuliah + Tipe (`(Teori)` / `(Praktikum)`).
     * Nama Dosen + Inisial Avatar (cth: `MR - Muhammad Rizqi, M.T.`).
     * Tag Ruangan (cth: `D102-Lab. MT`).
     * Aksi Baris Kartu: Tombol Edit (Pensil) & Pindahkan (Panah).
3. **Drawer / Modal "Edit / Tambah Sesi Kuliah":**
   * Input Dropdown Mata Kuliah (auto-complete dari daftar master matkul).
   * Input Tipe Sesi: Radio Button `Teori` / `Praktikum`.
   * Input Hari: Dropdown `Senin` s.d. `Jumat`.
   * Input Jam Mulai & Jam Selesai: Input time mask `HH:MM`.
   * Input Dosen Pengampu: Dropdown inisial dosen (otomatis mengisi nama lengkap).
   * Input Ruangan: Text input dengan suggestions (`D101`, `Lab RPL`, dll.).
   * Tombol: `[Batal]` dan `[Simpan Perubahan]`.
4. **Modal "Tambah Kelas Baru":**
   * Input ID Kelas: Text input (cth: `3C` atau `TI-1A`).
   * Input Nama Kampus/Sub-Jurusan (cth: `D4 Semester 3 / Kelas 3C`).
   * Opsi Awal: `[Duplikasi dari Kelas 3A]` atau `[Mulai dari Jadwal Kosong]`.

---

### Layar 3: 🔄 Kalender Jadwal Pengganti & Libur (*Overrides*)

**Tujuan:** Mengelola perubahan dinamis yang mengikat pada tanggal tertentu (*Date-Specific*) seperti kuliah ditiadakan, digeser ke hari lain, atau libur nasional.

#### Komponen UI & Layout:
1. **Filter Status & Tipe:**
   * Filter Tipe: `[Semua]` `[Pindah Jam]` `[Kuliah Kosong]` `[Kuliah Pengganti]` `[Libur Seharian]`.
   * Filter Waktu: `[Perubahan Aktif (Mendatang)]` `[Riwayat Lewat]`.
2. **Tabel Data Jadwal Pengganti:**
   * Kolom 1: **Tanggal Berlaku** (Format: `Senin, 08 Sep 2026`).
   * Kolom 2: **Tipe Perubahan** (Badge warna: Kuning = RESCHEDULE, Merah = CANCEL, Biru = EXTRA, Hijau = HOLIDAY).
   * Kolom 3: **Mata Kuliah** (Nama matkul lengkap).
   * Kolom 4: **Jadwal Asal ➔ Jadwal Baru** (cth: `Senin 07:00 ➔ Selasa 13:00 [Lab 312]`).
   * Kolom 5: **Keterangan / Alasan** (cth: *"Dosen dinas luar kota"*).
   * Kolom 6: **Dibuat Oleh** (JID Pembuat / Web Admin).
   * Kolom 7: **Aksi**: Tombol `[Batalkan (Hapus)]` dengan dialog konfirmasi.
3. **Modal "Buat Jadwal Pengganti Baru":**
   * Tab Pemilih Mode:
     * Mode A: **Pindah Jam / Hari** (Pilih matkul normal ➔ Tentukan tanggal & jam baru ➔ Deteksi bentrok otomatis).
     * Mode B: **Kuliah Ditiadakan / Kosong** (Pilih tanggal & sesi kuliah yang ditiadakan ➔ Masukkan alasan).
     * Mode C: **Kuliah Pengganti Tambahan** (Matkul, tanggal baru, rentang jam, ruangan).
     * Mode D: **Umumkan Libur Seharian** (Tanggal libur ➔ Keterangan hari libur, misal Hari Kemerdekaan).

---

### Layar 4: 📝 Papan Manajemen Tugas & Arsip Tenggat

**Tujuan:** Melacak tugas akademik mahasiswa, batas waktu penyerahan (*deadline*), dan rekam jejak tugas yang telah diselesaikan.

#### Komponen UI & Layout:
1. **Control Header:**
   * Tab Tampilan: `[Tampilan Tabel / List]` atau `[Papan Kanban (Berdasarkan Urgensi)]`.
   * Dropdown Filter Matkul: `Semua Mata Kuliah` ▾.
   * Pencarian: Input cari kata kunci tugas.
   * Tombol: `[+ Tambah Catatan Tugas]`.
2. **Kategori Kanban / Urgensi List:**
   * **Kolom 1: 🚨 Mendesak (Hari Ini & Besok / H-0 & H-1)**
   * **Kolom 2: ⚠️ Minggu Ini (H-2 s.d. H-7)**
   * **Kolom 3: 📅 Mendatang (> H-7)**
   * **Kolom 4: ✅ Selesai / Arsip**
3. **Komponen Kartu Tugas:**
   * Tag Mata Kuliah dengan warna khas per matkul.
   * Judul & Instruksi Tugas.
   * Badge Hitung Mundur Waktu (cth: `🚨 8 Jam Tersisa` atau `⚠️ H-2 (Jumat, 23:59 WIB)`).
   * Info Pembuat Catatan (Komti / Mahasiswa).
   * Tombol Aksi Cepat: `[✓ Tandai Selesai]`, `[Perpanjang Tenggat]`, `[Hapus]`.
4. **Modal "Tambah / Edit Catatan Tugas":**
   * Dropdown Mata Kuliah (terhubung ke kurikulum kelas aktif).
   * Deskripsi / Instruksi Tugas (Textarea dengan dukungan bullet points).
   * Date & Time Picker untuk Deadline.
   * *Preview Real-Time*: Menampilkan bagaimana notifikasi teks bot WhatsApp akan terlihat di chat grup.

---

### Layar 5: 👥 Grup WhatsApp & Broadcast Pengingat Pagi

**Tujuan:** Mengatur grup-grup WhatsApp yang menerima broadcast pengingat pagi otomatis jam 06:30 WIB dan mengaitkan masing-masing grup ke kelasnya.

#### Komponen UI & Layout:
1. **Global Reminder Scheduler Settings:**
   * Toggle: `[Pengingat Otomatis Aktif / Nonaktif]`.
   * Pengaturan Waktu Broadcast: Jam `[06]` : `[30]` WIB (Senin – Jumat).
2. **Tabel Daftar Grup Terdaftar:**
   * Kolom 1: **Nama Grup WhatsApp** (cth: *"D4 Teknik Informatika 3A Official"*).
   * Kolom 2: **JID Grup** (ID WhatsApp, cth: `120363xxx@g.us`).
   * Kolom 3: **Kelas Tertaut (*Binding*)**: Dropdown pemilih kelas (`3A`, `3B`, dll.).
   * Kolom 4: **Status Broadcast**: Badge Hijau `Aktif` atau Abu-abu `Nonaktif`.
   * Kolom 5: **Tanggal Ditambahkan**.
   * Kolom 6: **Aksi**:
     * Tombol `[🧪 Uji Kirim Broadcast Sekarang]` (Mengirim pesan simulasi langsung ke grup tersebut).
     * Tombol `[Hapus dari Daftar Reminder]`.
3. **Panel Preview "Pesan Pengingat Pagi":**
   * Menampilkan simulasi tampilan gelembung chat WhatsApp yang akan diterima grup terpilih pada hari ini (menampilkan jadwal hari ini + alert tugas mendesak jika ada).

---

### Layar 6: 📱 Gateway Status WhatsApp & Scan QR Code

**Tujuan:** Menyediakan kontrol koneksi WhatsApp tanpa membuka konsol/terminal.

#### Komponen UI & Layout:
1. **Status Banner:**
   * State A: **🟢 WhatsApp Terhubung**
     * Menampilkan Nomor HP Bot (cth: `+62 812-3456-7890`), Nama Profil, Push Name, dan Platform Client.
     * Tombol `[Putuskan Sesi / Logout]` (dengan konfirmasi keamanan).
   * State B: **🟡 Menunggu Scan QR Code**
     * Menampilkan kartu visual berisi **QR Code tajam beresolusi tinggi** dengan animasi pulse refresh timer (otomatis refresh QR setiap 20 detik jika belum di-scan).
     * Panduan 3 langkah mudah: *"Buka WhatsApp di HP > Perangkat Tertaut > Tautkan Perangkat"*.
   * State C: **🔴 Sambungan Terputus / Reconnecting**
     * Menampilkan status Watchdog Auto-Reconnect dengan *exponential backoff countdown*.
2. **Log Konsol Aktivitas Live (Mini Console):**
   * Kotak terminal hitam (`JetBrains Mono`) yang menampilkan 50 log terakhir (Pesan masuk, perintah yang diproses, status broadcast) dengan auto-scroll.

---

### Layar 7: 🔒 Kunci Keamanan Akses (PIN / Session Gate)

**Tujuan:** Mencegah akses tidak berwenang saat dashboard dibuka melalui IP publik VPS kampus/kosan.

#### Komponen UI & Layout:
* Layar minimalis berlatar belakang gelap dengan kartu otentikasi di tengah (*Center Card*).
* Input PIN / Password Admin (Masked password dengan tombol lihat).
* Proteksi Brute-force: Kunci sementara selama 5 menit jika salah memasukkan PIN 5 kali berturut-turut.
* Fitur "Ingat Saya di Perangkat Ini" (Token cookie HttpOnly 7 hari).

---

## 6. 🔄 State Machine & Error Handling UI

Untuk menjamin antarmuka yang berkualitas tinggi (*anti-AI slop*), setiap layar dan kartu data wajib memiliki **4 status kondisi UI**:

| Status UI | Panduan Tampilan bagi UI/UX Designer |
| :--- | :--- |
| **1. Loading State** | Menggunakan **Skeleton Screens** berbentuk balok abu-abu berkedip halus (*shimmer animation*), BUKAN spinner putar sederhana di tengah layar kosong. |
| **2. Empty State** | Jika data belum ada (misal: belum ada tugas aktif), tampilkan ilustrasi ikon minimalis, pesan ramah (cth: *"Hore! Tidak ada deadline tugas yang tercatat"*), dan tombol CTA (cth: `[+ Buat Catatan Tugas]`). |
| **3. Error State** | Jika gagal menghubungi server atau database terkunci, tampilkan banner peringatan merah dengan tombol `[Coba Lagi]` dan pesan penjelasan yang mudah dimengerti. |
| **4. Success Feedback** | Setiap operasi (simpan, edit, hapus) memunculkan **Toast Notification** di pojok kanan atas dengan animasi slide-in durasi 3 detik. |

---

## 7. 🔌 Spesifikasi Kontrak REST API (Backend Endpoint)

Dashboard berkomunikasi dengan server Go menggunakan RESTful JSON endpoints berikut:

### Telemetri & Sistem
* `GET /api/v1/system/status` ➔ Status WA, uptime, memory usage, kelas default, jam reminder.
* `GET /api/v1/system/qr` ➔ Mengambil QR Code terkini (Data URL / base64 image).
* `POST /api/v1/system/logout` ➔ Memutuskan koneksi sesi WA.
* `POST /api/v1/system/reload` ➔ Memanggil `classManager.ReloadAll()`.

### Kelas & Jadwal
* `GET /api/v1/classes` ➔ Daftar seluruh kelas yang tersedia (`["3A", "3B"]`).
* `GET /api/v1/classes/:id/schedule` ➔ Mengambil seluruh jadwal matkul & dosen kelas `:id`.
* `POST /api/v1/classes` ➔ Menambahkan kelas baru ke direktori `data/jadwal/`.
* `PUT /api/v1/classes/:id/schedule` ➔ Memperbarui daftar mata kuliah / jadwal kelas `:id`.

### Tugas (Tasks)
* `GET /api/v1/tasks?scope_jid=&status=active` ➔ Mengambil daftar tugas dengan filter.
* `POST /api/v1/tasks` ➔ Membuat catatan tugas baru.
* `PUT /api/v1/tasks/:id` ➔ Memperbarui deskripsi atau deadline tugas.
* `PATCH /api/v1/tasks/:id/done` ➔ Menandai tugas selesai (`is_done = 1`).
* `DELETE /api/v1/tasks/:id` ➔ Menghapus catatan tugas.

### Jadwal Pengganti (Overrides)
* `GET /api/v1/overrides?scope_jid=` ➔ Daftar jadwal pengganti aktif & riwayat.
* `POST /api/v1/overrides` ➔ Membuat override baru (pindah, kosong, ekstra, libur).
* `DELETE /api/v1/overrides/:id` ➔ Menghapus/membatalkan jadwal pengganti.

### Grup & Reminder
* `GET /api/v1/groups` ➔ Daftar grup di `reminder_groups.json` beserta kelas tertautnya.
* `PUT /api/v1/groups/:jid/class` ➔ Mengubah binding kelas grup di `chat_settings`.
* `POST /api/v1/groups/:jid/test-reminder` ➔ Mengirim broadcast pengingat simulasi ke grup.

---

## 8. 📱 Pertimbangan Responsif (Mobile-First Ergonomics)

1. **Navigasi Bawah pada Ponsel (*Mobile Bottom Navigation Bar*):**
   * Pada layar HP (< 768px), sidebar desktop berganti menjadi bilah navigasi bawah (*Bottom Nav*) berisi 4 menu utama: `Dashboard`, `Jadwal`, `Tugas`, `Grup`.
2. **Ukuran Area Sentuh (*Touch Targets*):**
   * Semua tombol aksi minimal memiliki ukuran area sentuh **44px × 44px** agar nyaman ditekan dengan jempol saat berjalan di lorong kampus.
3. **Tabel Responsif:**
   * Di desktop berbentuk tabel horizontal lengkap. Di ponsel otomatis berubah menjadi kartu bertumpuk (*Stacked Cards*).

---

## 9. 🚀 Rencana Rilis & Milestone Desain

1. **Fase Desain UI/UX (Oleh Rekan Designer):**
   * Pembuatan Moodboard & Design Tokens (Figma).
   * Wireframing Low-Fidelity (Alur alih halaman).
   * High-Fidelity Mockups (Desktop 1440px & Mobile 390px).
   * Interactive Prototype (Alur tambah tugas & pindah jam kuliah).
2. **Fase Review & Alignment:**
   * Pemeriksaan bersama untuk memastikan seluruh field data cocok 100% dengan kontrak API backend Go.
3. **Fase Implementasi Frontend & Integrasi Backend:**
   * Pembuatan template HTML/CSS/JS di folder `web/`.
   * Penyematan biner melalui `embed.FS` di `main.go`.
   * Pengujian end-to-end melalui browser dan verifikasi pesan WhatsApp.
