# PENCIL.DEV DESIGN SYSTEM BLUEPRINT & CONTEXT
> **Panduan Konteks & Spesifikasi Desain untuk Pencil.dev (Redesign Bot Jadwal v2.0)**  
> *Gunakan dokumen ini sebagai Master Context Prompt di Pencil.dev agar desain baru konsisten, rapi, tetap mempertahankan identitas warna asli, dan bebas halusinasi fitur.*

---

## 1. ATURAN EMAS ANTI-HALUSINASI (STRICT GUARDRAILS)

Saat mendesain ulang (redesign) antarmuka ini di Pencil.dev, ikuti aturan mutlak berikut:

1. **JANGAN Menambah Fitur Asing**:
   - DILARANG menambahkan fitur *e-commerce*, pembayaran SPP, sistem keranjang, absensi GPS / selfie, forum chatting publik mahasiswa, grading/penilaian nilai ujian mahasiswa, atau feed media sosial.
   - Fokus aplikasi HANYA pada: **Jadwal Kuliah (Reguler & Perubahan)**, **Tugas Kuliah & Deadline**, **Materi Perkuliahan**, **Notifikasi WhatsApp Bot Otomatis**, dan **Tata Kelola Kampus/Kelas**.
2. **JANGAN Mengurangi Aksi/Fungsi Kunci**:
   - Fitur esensial seperti: *Switch Role / Ganti Konteks*, *Rotasi Kode Portal Kelas*, *Review KM terhadap Draf PJ*, *1-Click Sync Master Data dari Berkas Jadwal*, *Audit Trail*, dan *Dukungan Kelas oleh Admin* WAJIB tetap ada.
3. **Bahasa Antarmuka**:
   - 100% menggunakan **Bahasa Indonesia** yang formal, jelas, dan ramah akademis perguruan tinggi (misal: "Ketua Murid", "PJ Mata Kuliah", "Ruangan", "Dosen Pengampu", "Tenggat Waktu", "Draf", "Publikasikan").
4. **Tujuan Utama Redesign**:
   - Menghilangkan inkonsistensi layout (misal: sebelumnya ada form statis abu-abu di atas tabel, penempatan tombol aksi yang tidak seragam, atau spacing yang acak).
   - Memperkuat hierarki visual, kenyamanan keterbacaan data (table di desktop, card di mobile), responsive grid, konsistensi dialog/modal, dan *empty state* yang informatif.

---

## 2. DESIGN TOKENS & VISUAL IDENTITY (PALET TERKUNCI)

Pencil.dev **WAJIB** menggunakan palet warna dan nilai token yang sudah ada, TIDAK BOLEH mengganti warna primer menjadi ungu, cyan, atau gradien AI generik.

### A. Palet Warna (Color Palette)
- **Primary / Brand**: `#3965FB` (Royal Blue / Biru Utama)
- **Primary Hover**: `#2f54d6`
- **Primary Soft (Surface Aksen)**: `#E9EAFF` (Biru muda lembut untuk badge / icon wrapper)
- **Primary Faint**: `#EEF2FF`
- **Tinta / Heading Text**: `#1F1F1F` (Dark Ink netral)
- **Deep Ink**: `#1D1B3A`
- **Secondary / Muted Text**: `#667085` (Slate Grey untuk deskripsi, subjudul, dan metadata)
- **Border / Garis Pemisah**: `#EEEEEE` (Divider subtle), `#D7DDE7` (Border kartu), `#CBD5E1` (Border input/control)
- **Surface / Card Background**: `#FFFFFF` (Putih bersih)
- **Surface Subtle**: `#F6F7FB` (Background baris hover / container preview)
- **Canvas / App Background**: `#F2F5FA` (Abu-abu kebiruan netral latar belakang dashboard)
- **Success / Status Aktif**: `#E1FFB7` (Mint green badge), teks `#166534` (Hijau tua terbaca)
- **Warning / Perlu Review**: `#FCD484` (Soft Gold), teks `#92400E` (Amber gelap)
- **Danger / Draf Ditolak / Hapus**: `#FF6C48` (Coral Red), teks `#991B1B` (Merah gelap)

### B. Tipografi (Typography)
- **Display & Headings**: `Poppins`, semi-bold & bold (Modern, tegas, berwibawa akademis).
- **Body & Controls**: `Montserrat` atau `Inter`, regular & medium (Sangat nyaman dibaca pada teks kecil).
- **Kode & Data Monospace**: `JetBrains Mono` (Untuk kode kelas 6 karakter, kode matkul `25IF1101`, inisial dosen `MR`, jam kuliah `08:00 - 09:40`).

### C. Spacing, Radius & Elevation
- **Border Radius**: 
  - Kartu & Modal: `rounded-2xl` (16px) atau `rounded-xl` (12px).
  - Tombol & Field Input: `rounded-xl` (10pxâ€“12px).
  - Badge Status: `rounded-full` (9999px pill).
- **Shadow**: Subtil dan halus (`shadow-sm`, `0 8px 24px rgb(15 23 42 / 0.08)`), tidak kotor atau gelap berlebihan.
- **Target Sentuh Minimum (Accessibility)**: Minimal **44 Ã— 44 px** untuk semua tombol aksi, icon button, dan item menu navigasi bawah.

---

## 3. STRUKTUR PERAN & HAK AKSES (USER ROLES MATRIX)

Aplikasi memiliki **4 Aktor / Role** dengan cakupan kerja yang sangat spesifik:

```
â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”
â”‚                        STRUKTUR ROLE APLIKASI                          â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚ 1. MAHASISWA      â”‚ â€¢ Portal Publik / Mahasiswa umum kelas             â”‚
â”‚    (Portal Read)  â”‚ â€¢ Tanpa login password! Hanya input Kode Kelas 6-karâ”‚
â”‚                   â”‚ â€¢ Akses: Lihat jadwal hari ini/minggu ini, tugas,  â”‚
â”‚                   â”‚   unduh materi, riwayat perubahan jadwal.          â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¼â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚ 2. KETUA MURID    â”‚ â€¢ Pemimpin & Pengelola Administrasi Kelas          â”‚
â”‚    (KM)           â”‚ â€¢ Login via No. WA + Kata Sandi                   â”‚
â”‚                   â”‚ â€¢ Akses: Review & Approve draf tugas dari PJ,      â”‚
â”‚                   â”‚   buat/terbitkan tugas, kelola jadwal pengganti,   â”‚
â”‚                   â”‚   rotasi kode portal kelas, kelola anggota PJ.     â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¼â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚ 3. PJ MATA KULIAH â”‚ â€¢ Penanggung Jawab Matkul Spesifik di Kelas       â”‚
â”‚    (PJ)           â”‚ â€¢ Login via No. WA + Kata Sandi                   â”‚
â”‚                   â”‚ â€¢ Akses: Buat draf tugas untuk matkulnya, ajukan  â”‚
â”‚                   â”‚   review ke KM, usulkan perubahan jadwal matkul,   â”‚
â”‚                   â”‚   konfirmasi ruangan kuliah dengan dosen.          â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¼â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚ 4. SYSTEM ADMIN   â”‚ â€¢ Administrator Kampus / Seluruh Sistem            â”‚
â”‚    (Super Admin)  â”‚ â€¢ Login via No. WA / Admin + Kata Sandi            â”‚
â”‚                   â”‚ â€¢ Akses: Kelola seluruh kelas kampus, pengguna &   â”‚
â”‚                   â”‚   penugasan peran, Master Matkul, Master Ruangan,  â”‚
â”‚                   â”‚   Master Dosen (1-Click Sync 106 data POLBAN),     â”‚
â”‚                   â”‚   antrean notifikasi bot WA, backup & audit log.   â”‚
â””â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”´â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜
```

---

## 3.1 MATRIKS FITUR LENGKAP & LOGIKA SISTEM (DEEP FEATURE BREAKDOWN)

Berikut adalah daftar seluruh fitur nyata dan logika alur yang ada di dalam sistem saat ini:

### 1. FITUR PORTAL PUBLIK MAHASISWA
- **Akses Tanpa Akun / Password**: Mahasiswa cukup memasukkan 6 karakter Kode Akses Kelas (misal: `IF241A`) atau mengakses tautan portal kelas.
- **Ringkasan Hari Ini**: Menampilkan status sesi kuliah yang sedang berlangsung atau sesi berikutnya hari ini (mata kuliah, jam, ruangan, dosen).
- **Jadwal Kuliah Efektif**: Tab tampilan Mingguan & Harian. Otomatis menggabungkan jadwal reguler dengan sesi kuliah pengganti / jam geser aktif.
- **Daftar Tugas & Filter**: Pengelompokan tugas berdasarkan deadline: *Hari Ini*, *Minggu Ini*, *Mendatang*, *Overdue (Terlewat)*, dan *Selesai*.
- **Detail Tugas**: Menampilkan deskripsi lengkap instruksi pengerjaan, format berkas, tempat pengumpulan (link Google Form / Classroom / Drive).
- **Materi Perkuliahan**: Akses modul ajar, slide presentasi, dan tautan file per mata kuliah.
- **Arsip Semester**: Mahasiswa dapat beralih ke semester sebelumnya untuk melihat kembali tugas dan materi lama dalam mode *Read-Only*.

### 2. FITUR MANAJEMEN TUGAS & WORKFLOW REVIEW (TASK LIFECYCLE)
- **Pembuatan Draf oleh PJ**: PJ Mata Kuliah hanya dapat membuat draf tugas untuk mata kuliah yang resmi ditugaskan kepadanya.
- **Pengajuan ke KM**: Setelah PJ mengisi judul, deskripsi, jenis tugas (Individu/Kelompok), deadline tanggal & jam, PJ mengajukan tugas ke Ketua Murid.
- **Alur Review KM (KM Review Modal)**:
  - Tugas berstatus `Menunggu Review KM`.
  - KM membuka dialog peninjauan tugas dengan 2 opsi keputusan:
    1. **Setujui & Publikasikan**: Tugas langsung terbit di Portal Mahasiswa dan otomatis memicu siaran pesan WhatsApp bot ke grup kelas.
    2. **Minta Perbaikan**: KM wajib menuliskan catatan revisi/alasan, status tugas kembali menjadi `Draf`, ditarik dari portal, dan dikembalikan ke PJ.
- **Penandaan Selesai & Pengarsipan**: Pengurus dapat menandai tugas yang sudah lewat batas waktu menjadi *Selesai* atau mengarsipkannya agar tidak memenuhi tampilan beranda.

### 3. FITUR JADWAL KULIAH, PERUBAHAN & BENTROK (SCHEDULE & CONFLICT DETECTION)
- **Pola Jadwal Mingguan (Reguler)**: Pengaturan jam kuliah tetap semester (Hari, Jam Mulai, Jam Selesai, Dosen Pengampu, Ruangan).
- **Perubahan Jadwal (Teaching Events)**:
  - **Kuliah Pengganti (Make-up Class)**: Menjadwalkan ulang sesi kuliah yang tertunda ke hari/jam lain.
  - **Sesi Tambahan**: Kuliah tambahan menjelang UTS/UAS.
  - **Libur / Ditiadakan**: Menandai sesi tertentu libur tanpa menghapus jadwal reguler permanen.
- **Pemeriksaan Bentrok Otomatis (*Conflict Checker*)**: Sistem mendeteksi bentrok jadwal jika ada 2 mata kuliah di kelas yang sama atau ruangan yang sama pada jam yang bertubrukan.
- **Perkuliahan Lintas Kelas (*Cross-Class Offering*)**: Satu sesi kuliah gabungan dapat diikuti oleh rombel kelas lain setelah disetujui KM kelas penerima.

### 4. FITUR RUANGAN, DOSEN & SINKRONISASI JADWAL POLBAN
- **Master Ruangan**: Data kode ruangan (misal: `D102-Lab. MT`), nama gedung (`Gedung D`, `Gedung H`), tipe ruangan (Kelas Teori / Lab), dan kapasitas kursi.
- **Pencarian Kandidat Ruangan**: Mencari ruangan kosong yang tidak terpakai pada rentang jam tertentu.
- **Pencatatan Konfirmasi TU Ruangan**: Mencatat status konfirmasi izin ruangan (`PENDING_TU`, `CONFIRMED` dengan nama petugas TU, atau `REJECTED`).
- **Master Dosen**: Data inisial dosen (`MR`, `AD`, `ZA`), nama lengkap beserta gelar akademik (`Dr. Ade Chandra Nugraha, S.Si., M.T.`), dan status keaktifan.
- **1-Click Sync Master Data (106 Data POLBAN)**:
  - Tombol otomatis untuk memindai berkas kurikulum resmi di `data/jadwal/*.json` dan mengekstrak:
    - **41 Mata Kuliah Unik**
    - **18 Ruangan Kuliah Unik**
    - **47 Dosen Pengampu Unik**
  - Tombol terpadu `[ ðŸ”„ Sinkron Semua Master (106) ]` untuk menyinkronkan seluruhnya dalam 1 kali klik.

### 5. FITUR OTOMASI WHATSAPP BOT
- **WhatsApp Web Gateway**: Integrasi bot WA tanpa ketergantungan pihak ketiga berbayar (status QR pairing, Terhubung, atau Terputus).
- **Siaran Otomatis (Auto Broadcast)**:
  - *Ringkasan Pagi*: Jadwal kuliah hari ini otomatis dikirim ke grup kelas setiap pukul 06:30.
  - *Pengumuman Tugas Baru*: Notifikasi instan saat KM menyetujui & menerbitkan tugas baru.
  - *Pengingat Tenggat Waktu*: Pengingat H-1 deadline tugas otomatis dikirim setiap pukul 19:00 malam.
  - *Pengumuman Kuliah Pengganti*: Siaran seketika jika ada jadwal pengganti atau pergeseran jam kuliah.
- **Antrean Notifikasi (Outbox Panel)**: System Admin dapat memantau status antrean pesan (`Pending`, `Sent`, `Failed`) dan memicu pengiriman ulang (*Retry*).

### 6. FITUR KEAMANAN, MULTI-ROLE & TATA KELOLA KELAS
- **Ganti Konteks / Switch Role**: 1 nomor WhatsApp dapat memiliki banyak penugasan (contoh: KM di D4-1A dan PJ di D4-1B). Pengguna dapat berpindah kelas/peran dengan 1 klik tanpa harus login ulang.
- **Rotasi Kode Akses Kelas (Instant Revoke)**: KM dapat mengganti kode 6-karakter kelas kapan saja jika kode tersebar luas. Sesi lama seketika dicabut dari server.
- **Mode Dukungan System Admin (Class Support Mode)**: Administrator dapat masuk sementara ke ruang kelas manapun untuk membantu konfigurasi teknis, ditandai dengan banner oranye khusus di bagian atas layar.
- **Audit Trail Komprehensif**: Mencatat seluruh aktivitas mutasi data (siapa mengubah apa, waktu, nilai sebelum, dan nilai sesudah).
- **Cadangan & Pemulihan Database (Backup & Restore)**: Unduh salinan database SQLite secara instan dan pemulihan data terverifikasi.

---

## 4. APP SHELL & LAYOUT ARCHITECTURE

Pencil.dev harus menggunakan layout responsif yang konsisten di semua layar:

### A. Tampilan Desktop (>= 1024px)
- **Sidebar Kiri (248px)**:
  - Header: Logo Asterisk + Nama "Bot Jadwal", Badge Role ("System Admin" / "Ketua Murid" / "PJ Mata Kuliah").
  - Context Selector: Dropdown ganti peran / ganti kelas aktif (misal jika user adalah KM di 1A tapi PJ di 1B).
  - Navigasi Menu: Dikelompokkan dengan label kategori rapi (misal: UTAMA, AKADEMIK, DATA & SISTEM, PENGATURAN).
  - Footer Sidebar: Profil akun, nomor WA, tombol Keluar.
- **Top Bar**:
  - Tombol hamburger (toggle collapse 76px / 248px).
  - Nama halaman saat ini + Breadcrumbs.
  - Quick action / status WhatsApp Bot (ikon WhatsApp hijau jika terhubung, merah jika putus).
- **Main Canvas (`bg-[#F2F5FA]`)**:
  - Max-width 1280px / full-fluid container dengan padding `p-6` sampai `p-10`.
  - Konten utama dibungkus dalam Kartu Putih Bersih (`ui-card bg-white rounded-2xl border border-[#D7DDE7] shadow-sm`).

### B. Tampilan Mobile (< 768px)
- **Top Bar Mobile**: Logo + Judul Layar + Hamburger Drawer Menu.
- **Konten Utama**: Menggunakan kartu vertikal (card-based layout), tabel diubah menjadi daftar kartu agar tidak perlu scroll horizontal berlebihan.
- **Bottom Navigation Bar (4 Tab Utama - Sticky Bawah, min 44px touch)**:
  - Memudahkan jempol tangan berpindah antara 4 fitur tersering digunakan (misal pada KM: Beranda, Tugas, Jadwal, Menu Lain).

---

## 5. INVENTARIS LAYAR DETAIL (SCREEN SPECIFICATION)

Pencil.dev harus merancang layar-layar berikut tanpa menambah atau menguranginya:

### MODUL 1: AUTENTIKASI & AKSES
1. **SCR-AUTH-001: Halaman Login Pengurus - [100% AUDITED & APPROVED]**
   - Elemen: Form No. WhatsApp (otomatis normalisasi format Indonesia), Kata Sandi (show/hide toggle), Tombol "Masuk", Link "Lupa Sandi", Info akses mahasiswa ("Mahasiswa? Masuk lewat Portal Kelas tanpa akun").
2. **SCR-AUTH-002: Pilih Kelas & Peran (Switch Role / Multi-Context Selector)**
   - Muncul jika 1 nomor WA memiliki beberapa penugasan (contoh: KM di D4-1A sekaligus PJ di D4-1B).
   - Elemen: Kartu pilihan peran, badge kelas, status penugasan aktif, tombol "Masuk ke Kelas Ini".
3. **SCR-AUTH-003: Aktivasi Undangan Pengurus - [100% AUDITED & APPROVED]**
   - Form aktivasi saat KM mengundang PJ baru via tautan WhatsApp.
4. **SCR-PORTAL-AUTH: Masukkan Kode Kelas Mahasiswa - [100% AUDITED & APPROVED]**
   - Halaman bersih, ramah seluler. Input 6-karakter kode kelas (misal: `IF241A`), tombol "Buka Portal Kelas".

---

### MODUL 2: PORTAL MAHASISWA (PUBLIC ACCESS)
1. **SCR-PORTAL-001: Beranda Mahasiswa (Ringkasan Kelas)**
   - Header kelas (Contoh: "D4 - Teknik Informatika 1A"), indikator semester aktif.
   - Banner jadwal hari ini (matkul sedang berlangsung, ruang, dosen, jam).
   - Kartu tugas deadline terdekat (hitung mundur hari).
2. **SCR-PORTAL-002: Jadwal Kuliah Lengkap**
   - Toggle view: Tampilan Mingguan (Seninâ€“Jumat) & Tampilan Harian.
   - Badge sesi: "Reguler", "Kuliah Pengganti", "Ditiadakan / Libur".
3. **SCR-PORTAL-003: Daftar Tugas & Deadline Mahasiswa**
   - Filter tab: "Semua", "Minggu Ini", "Mendatang", "Selesai".
   - Kartu tugas: Judul, mata kuliah, tenggat waktu (tanggal & jam), status pengumpulan, tombol lihat detail.
4. **SCR-PORTAL-004: Detail Tugas Mahasiswa**
   - Rincian deskripsi tugas, lampiran/tautan materi, format pengumpulan.
5. **SCR-PORTAL-005: Materi Kuliah**
   - Daftar materi berdasarkan mata kuliah, tautan Google Drive / modul pembelajaran.

---

### MODUL 3: KETUA MURID (KM) WORKSPACE
1. **SCR-KM-001: Dashboard KM**
   - Metric cards: Jumlah Tugas Aktif, Tugas Menunggu Review KM, Jadwal Minggu Ini, Status WA Gateway.
   - Kotak perhatian: "Ada 2 usulan tugas dari PJ yang perlu diperiksa".
2. **SCR-KM-002: Manajemen Tugas Kelas**
   - Header: Tombol `+ Buat Tugas Baru`.
   - Filter tabs: `Aktif`, `Draf`, `Menunggu Review KM`, `Selesai`, `Diarsipkan`.
   - Tabel/List Tugas dengan kolom: Judul, Matkul, Pembuat (PJ/KM), Deadline, Status, Aksi (Ubah, Hapus, Publikasikan).
   - **Modal Review KM**: Preview draf dari PJ -> Tombol "Setujui & Publikasikan (Kirim WA)" vs "Minta Perbaikan (Beri Catatan)".
3. **SCR-KM-003: Manajemen Jadwal Kuliah & Perubahan**
   - Pola jadwal mingguan kelas.
   - Form penambahan jadwal pengganti / pergeseran jam kuliah.
   - Peringatan deteksi bentrok jadwal otomatis (*Conflict Checker*).
4. **SCR-KM-004: Anggota & Penugasan PJ Matkul**
   - Daftar PJ per mata kuliah di kelas tersebut.
   - Tombol "Undang PJ Baru" (menghasilkan tautan undangan WA).
5. **SCR-KM-005: Pengaturan Portal Kelas**
   - Kode akses kelas aktif 6-karakter.
   - Tombol "Rotasi / Ganti Kode Baru" (jika kode bocor).
   - Pengaturan jam pengingat notifikasi WA (misal: H-1 pukul 19:00).
6. **SCR-KM-006: Riwayat Audit Kelas**
   - Log aktivitas pengurus: Siapa mengubah apa, kapan, dan riwayat revisi tugas.

---

### MODUL 4: PENANGGUNG JAWAB MATA KULIAH (PJ)
1. **SCR-PJ-001: Dashboard PJ**
   - Daftar mata kuliah yang diampu oleh PJ tersebut.
   - Status tugas yang sedang berjalan untuk mata kuliahnya.
2. **SCR-PJ-002: Kelola Draf Tugas Matkul**
   - Form pembuatan tugas: Judul, Deskripsi, Tenggat Waktu, Tautan Sumber.
   - Tombol "Simpan sebagai Draf" atau "Ajukan ke KM untuk Diperiksa".
3. **SCR-PJ-003: Konfirmasi Jadwal & Ruangan Kuliah**
   - Pengaturan sesi kuliah mata kuliahnya, pemilihan dosen pengampu, konfirmasi ketersediaan ruangan kelas/lab.

---

### MODUL 5: SYSTEM ADMIN (SUPER ADMINISTRATOR)
1. **SCR-ADM-001: Ringkasan Sistem (Pusat Kendali Sistem / Health & Overview) - [100% AUDITED & APPROVED]**
   - **Header**: Judul "Pusat kendali sistem", subtitle, tombol aksi `[ â±ï¸ Lihat audit ]` dan `[ ðŸ”„ Sinkron semua master (106) ]`.
   - **Hero Card Gateway WhatsApp**: Latar Dark Navy (`#1D1B3A`), badge `LAYANAN NORMAL`, headline status bot, quick stats perangkat terhubung & antrean pesan.
   - **4 Stat Cards (KPI Telemetri Global via GET /api/v1/admin/status)**:
     - Total Kelas: "19" (19 kelas aktif berjalan)
     - Pengguna & Peran: "24" (19 KM â€¢ 4 PJ â€¢ 1 Admin)
     - Tugas Dipantau: "38" (35 diterbitkan â€¢ 3 draf)
     - Jadwal Aktif: "84" (81 reguler â€¢ 3 pengganti)
   - **Grid Bawah (2 Kolom)**:
     - Kiri (60%): Master Data Berkas Jadwal (41 mata kuliah, 18 ruangan kampus, 47 dosen pengampu, badge Siap Disinkronkan).
     - Kanan (40%): Perlu Tindakan (Badge 2 Item: Pesan gagal dikirim & Cadangan berkala).
2. **SCR-ADM-002: Master Mata Kuliah (Daftar & Tata Kelola Kurikulum) - [100% AUDITED & APPROVED]**
   - **A. SCR-ADM-002A: Layar Utama & Tabel Master Matkul**:
     - *Header*: Judul "Master Mata Kuliah", badge count total terdaftar (`41 Mata Kuliah`), subtitle deskripsi kurikulum POLBAN, tombol aksi kanan: `[ ☁️ Impor / Sinkron ]` (outline) dan `[ + Tambah Mata Kuliah ]` (primer biru `#3965FB`).
     - *Panel Perhatian Usulan KM*: Card berlatar `#E9EAFF` (border `#3965FB`/20, p-4, rounded-xl) jika ada usulan baru dari KM (`GET /api/v1/master/proposals?kind=COURSE&status=PENDING`): info nama matkul usulan, kelas KM pengusul, tombol `[ Setujui ]` (hijau) dan `[ Tolak... ]` (coral/merah).
     - *Filter & Search Toolbar*: Field cari kode/nama matkul (ikon kaca pembesar), filter dropdown status (`Semua Status`, `Aktif`, `Nonaktif`).
     - *Tabel Desktop (5 Kolom Konsisten)*:
       1. `KODE`: Font JetBrains Mono bold (`25IF1101`, `25TI1101`).
       2. `NAMA MATA KULIAH`: Teks nama lengkap matkul (Poppins/Inter semibold).
       3. `STATUS`: Badge pill (`Aktif` mint green `#E1FFB7` teks `#166534` vs `Nonaktif` abu-abu).
       4. `TERAKHIR DIUBAH`: Timestamp tanggal pembaruan data.
       5. `AKSI`: Tombol trigger menu `···` (ghost button 44px).
     - *Mobile Responsive*: Card list per matkul dengan layout vertikal, target tap min 44px.
   - **B. SCR-ADM-002-MENU: Dropdown Aksi Baris `···`**:
     - Popover card (rounded-xl, shadow-lg, border `#D7DDE7`, w-48):
       - Item 1: `✏️ Ubah Mata Kuliah` (membuka modal ubah).
       - Item 2: `🚫 Nonaktifkan / Aktifkan` (toggle soft status via modal konfirmasi).
       - Item 3: `🕒 Lihat Riwayat Audit` (melihat snapshot sebelum/sesudah perubahan).
   - **C. SCR-ADM-002C: Modal Tambah Mata Kuliah Baru**:
     - Modal pop-up tengah (max-w-md 480px, rounded-2xl, p-6).
     - Field:
       1. `Kode Mata Kuliah`: Input text uppercase (misal: `25IF1101`, font mono, validasi unik).
       2. `Nama Mata Kuliah`: Input text nama resmi matkul.
       3. `Status Default`: Toggle pill Aktif (default ON).
     - Footer: Tombol `[ Batal ]` & `[ Simpan Mata Kuliah ]`.
   - **D. SCR-ADM-002D: Modal Ubah Mata Kuliah (Kode Immutable 🔒)**:
     - Modal pop-up tengah (max-w-md 480px, rounded-2xl, p-6).
     - Field:
       1. `Kode Mata Kuliah`: Readonly / disabled box dengan ikon 🔒 gembok dan hint text abu-abu: *"Kode mata kuliah bersifat permanen untuk menjaga integritas riwayat jadwal semester."*
       2. `Nama Mata Kuliah`: Editable input text nama matkul.
       3. `Status Mata Kuliah`: Segmented control pilihan `Aktif` vs `Nonaktif`.
     - Footer: Tombol `[ Batal ]` & `[ Simpan Perubahan ]`.
   - **E. SCR-ADM-002E: Modal Tolak Usulan KM (Review Rejection)**:
     - Modal pop-up tengah (max-w-md 480px, rounded-2xl, p-6).
     - Header: Judul "Tolak Usulan Mata Kuliah", icon peringatan merah.
     - Ringkasan Usulan: Box preview kode matkul, nama matkul usulan, pengusul (KM D4-1A), dan catatan KM.
     - Field Wajib: Textarea `Alasan Penolakan (Catatan Review)` minimal 5 karakter (*"Tuliskan alasan penolakan agar Ketua Murid memahami penyesuaian kurikulum..."*).
     - Footer: Tombol `[ Batal ]` & `[ Konfirmasi Tolak Usulan ]` (coral red).
   - **F. SCR-ADM-002F: Modal Konfirmasi Nonaktifkan Mata Kuliah**:
     - Modal dialog konfirmasi (max-w-md, rounded-2xl, p-6).
     - Header: Judul "Nonaktifkan Mata Kuliah?", icon tanda seru amber.
     - Body: Teks konfirmasi bahwa matkul tidak akan muncul pada penawaran kelas semester baru, namun riwayat jadwal kelas yang sudah berjalan tetap utuh dan aman.
     - Footer: Tombol `[ Batal ]` & `[ Nonaktifkan ]` (amber/merah).
   - **G. SCR-ADM-002G: Modal Impor & Sinkronisasi (Dual Tabs)**:
     - Modal pop-up tengah (max-w-lg 540px, rounded-2xl, p-6).
     - Tab Navigation: `Impor Teks Bebas / JSON` | `Sinkron Otomatis Jadwal POLBAN`.
     - *Tab 1 (Impor Bebas / JSON)*:
       - Dropzone / Textarea JSON masal format array: `[{"code":"25IF1101","name":"..."},...]`.
       - Tombol `[ 📥 Unduh Contoh Format JSON ]` di pojok kanan atas.
       - Tombol aksi: `[ Validasi & Impor ]`.
     - *Tab 2 (Sinkronisasi Otomatis Jadwal POLBAN)*:
       - Opsi 1: `[ 🔄 Ekstrak Matkul dari Berkas Jadwal (41) ]` (Memindai 19 berkas jadwal POLBAN).
       - Opsi 2: `[ ⚡ Sinkronkan Semua Master Kampus (106) ]` (Sinkron terpadu 41 Matkul + 18 Ruangan + 47 Dosen).
       - *Box Jaminan Integritas & Keamanan Data*: Card berlatar `#EEF2FF` dengan ikon perisai biru:
         *"Sinkronisasi bersifat aman & idempotent (merge-upsert). Data yang ditambahkan secara manual tidak akan terhapus, data tidak akan menduplikasi jika dijalankan berulang kali, dan status nonaktif yang Anda tentukan tetap terlindungi."*

3. **SCR-ADM-003: Master Ruangan (Manajemen Ruangan Perkuliahan Kampus) - [100% AUDITED & APPROVED]**
   - **A. Layar Utama Master Ruangan (SCR-ADM-003A)**:
     - Header: Judul "Master Ruangan", count badge pill "18 ruangan", deskripsi fungsional, tombol aksi kanan: `[ 🔄 Impor / Sinkron ]` (outline) & `[ + Tambah Ruangan ]` (primary `#3965FB`).
     - Banner Konteks: Biru muda `#EEF2FF`, border `#C7D7FE`: *"Data ruangan menjadi rujukan saat menyusun jadwal kelas dan memilih lokasi perkuliahan."*
     - Banner Usulan KM Menunggu Persetujuan: Amber box `#FEF3C7`, outline `#FCD34D`, icon lonceng, detail usulan (misal: `D117-Lab. IoT` oleh KM D4-TI 1B), kutipan catatan KM, tombol `[ Setujui ]` (hijau `#15803D`) & `[ Tolak... ]` (outline merah `#DC2626`).
     - Filter Bar 4 Kontrol: Search input (`Cari kode, nama, atau gedung ruangan...`), dropdown Gedung (`Semua gedung`, `Gedung D`, `Gedung H`), dropdown Jenis (`Semua jenis`, `Teori`, `Laboratorium`), dropdown Status (`Semua status`, `Aktif`, `Nonaktif`).
     - Tabel Desktop 6 Kolom:
       * `RUANGAN`: Kode tebal (misal `D101-Kelas`, `D102-Lab. MT`) + subtitle nama ruangan.
       * `GEDUNG`: Gedung D / Gedung H.
       * `KAPASITAS`: Format angka kursi (misal `32 kursi`).
       * `JENIS`: `Teori` atau `Laboratorium`.
       * `STATUS`: Badge pill hijau mint `Aktif` (`#E1FFB7`) atau abu-abu `Nonaktif`.
       * `AKSI`: Tombol pill `[ ✏️ Ubah ]` (lavender `#E9EAFF`, text `#3965FB`) + Tombol menu cepat `[ ··· ]`.
     - Footer: Keterangan *"Menampilkan 4 dari 18 ruangan · contoh hasil pemindaian POLBAN."*
   - **B. SCR-ADM-003C: Modal Tambah Ruangan Baru**:
     - Modal pop-up tengah (max-w-lg 540px, rounded-2xl, p-6).
     - Box Peringatan Permanen: Background `#EEF2FF`, border `#C7D7FE`, icon info biru: *"Kode ruangan bersifat permanen (immutable) dan tidak dapat diubah setelah disimpan untuk menjaga relasi jadwal kelas. Pastikan format sesuai standar POLBAN (contoh: D102-Lab. MT)."*
     - Baris 1: `Kode ruangan *` (placeholder mono: `D102-Lab. MT atau RT-01`) & `Nama ruangan *` (`Lab. Multimedia`).
     - Baris 2: `Gedung (opsional)` (`Gedung D`), `Tipe ruangan (opsional)` (`Laboratorium`), `Kapasitas (opsional)` (placeholder `32` dengan suffix `kursi`).
     - Baris 3: Status ruangan switch toggle interaktif.
     - Footer: Tombol `[ Batal ]` & `[ Simpan Ruangan ]`.
   - **C. SCR-ADM-003D: Modal Ubah Ruangan**:
     - Modal pop-up tengah (width: 540px, rounded-2xl).
     - Field `Kode ruangan`: Terkunci / Disabled (🔒) berlatar `#F6F7FB`, teks mono `#667085` + helper text: *"Kode bersifat permanen dan tidak dapat diubah."*
     - Field Nama, Gedung, Tipe, Kapasitas: Bebas diperbarui.
     - Footer Kiri: Aksi cepat teks merah `Nonaktifkan ruangan`.
     - Footer Kanan: Tombol `[ Batal ]` & `[ Simpan Perubahan ]`.
   - **D. SCR-ADM-003E: Modal Tolak Usulan Ruangan**:
     - Modal dialog pop-up (width: 480px, rounded-2xl, p-6).
     - Header: Icon bulatan merah muda `#FEE4E2` dengan silang merah `#D92D20`, judul "Tolak Usulan Ruangan".
     - Summary Card: Ringkasan `D117-Lab. IoT`, pengusul KM, spesifikasi gedung/tipe/kapasitas, dan kutipan catatan KM.
     - Form Wajib: Textarea `Alasan Penolakan (Catatan Review) *` min. 5 karakter untuk audit log & notifikasi WhatsApp ke KM.
     - Footer: Tombol `[ Batal ]` & `[ Tolak Usulan ]` (merah bahaya `#D92D20`).
   - **E. SCR-ADM-003F: Modal Konfirmasi Nonaktifkan Ruangan**:
     - Modal dialog pop-up (width: 440px, rounded-2xl, p-6).
     - Header: Icon bulatan amber `#FEF0C7` dengan tanda seru `#DC6803`, judul "Nonaktifkan Ruangan?".
     - Body: Pertanyaan konfirmasi spesifik + Card 3 poin jaminan integritas (riwayat semester lalu tetap aman, tidak muncul di semester baru, bisa diaktifkan kembali).
     - Footer: Tombol `[ Batal ]` & `[ Nonaktifkan ]` (`#D92D20`).
   - **F. SCR-ADM-003G: Modal Impor & Sinkronisasi Ruangan (Dual-Tab)**:
     - *Tab 1 (Impor Format JSON)*: Textarea payload array JSON masal, link aksi `[ 📥 Unduh Format JSON ]`, box petunjuk validasi (code wajib & unik, idempotent upsert), live counter hijau `✓ 2 ruangan terdeteksi valid`, tombol `[ Validasi & Impor ]`.
     - *Tab 2 (Sinkronisasi Otomatis POLBAN)*:
       * Opsi A: `[ 🔄 Sinkronkan Ruangan Saja ]` (Memindai berkas jadwal, ekstrak 18 ruangan resmi POLBAN di Gedung D & H via `POST /api/v1/master/rooms/sync-jadwal`).
       * Opsi B: `[ ⚡ Sinkronkan Semua Master (106) ]` (Sinkron terpadu 41 matkul + 18 ruangan + 47 dosen via `POST /api/v1/master/sync-all`).
       * Safety Banner `#EEF2FF`: Jaminan integritas merge-upsert aman, data manual terlindungi, status nonaktif tidak tertimpa.

4. **SCR-ADM-004: Master Dosen (Manajemen Dosen Pengampu POLBAN) - [100% AUDITED & APPROVED]**
   - **A. Layar Utama Master Dosen (SCR-ADM-004A)**:
     * *Empty State (SCR-ADM-004A-EMPTY)*: Ilustrasi kartu ID kosong `#EEF2FF`, teks "Belum Ada Data Dosen", deskripsi panduan inisial unik 2–4 huruf kapital, tombol aksi ganda: `[ ⚡ Sinkronkan 47 Dosen POLBAN ]` (Primary `#3965FB`) dan `[ + Tambah Manual ]` (Outline `#D7DDE7`).
     * *Filled State (SCR-ADM-004A-FILLED)*:
       - Header: Judul "Master Dosen", badge pill "47 dosen" (`#F1F5F9`), deskripsi, tombol aksi kanan: `[ 🔄 Impor / Sinkron ]` (outline) & `[ + Tambah Dosen ]` (primary `#3965FB`).
       - Callout Edukasi Pengampu: Biru muda `#EEF2FF`, border `#C7D7FE`: *"Dosen yang aktif dapat dipilih sebagai pengampu mata kuliah saat jadwal disusun."*
       - Filter Bar: Search input (`Cari nama atau kode dosen...`) + dropdown Status (`Semua status`, `Aktif`, `Nonaktif`).
       - Tabel Desktop 4 Kolom: `KODE` (JetBrains Mono bold 14px, e.g. `AD`, `BW`, `HA`, `PH`, `SD`), `NAMA LENGKAP & GELAR` (Geist medium 13px), `STATUS` (Badge pill hijau mint `Aktif` `#E1FFB7` / `#166534`), `AKSI` (Tombol `[ ✏️ Ubah ]` lavender `#E9EAFF` + Menu cepat `[ ··· ]`).
       - Footer: Menampilkan catatan *"Menampilkan 5 dari 47 dosen · hasil pemindaian kurikulum POLBAN"* & pagination `‹ 1 2 3 … 10 ›`.
   - **B. SCR-ADM-004C: Modal Tambah Dosen Baru**:
     * Modal dialog pop-up tengah (width: 500px, rounded-2xl 16px, background `#FFFFFF`, shadow-xl).
     * Info Box Immutability: Background `#EEF2FF`, border `#C7D7FE`, ikon info: *"Kode inisial dosen bersifat permanen (immutable) dan tidak dapat diubah setelah disimpan untuk menjaga relasi penugasan jadwal. Gunakan format inisial resmi POLBAN (2–4 huruf kapital, contoh: AD, BW, HA)."*
     * Field 1: `Inisial / Kode Dosen *` (placeholder mono: `Contoh: AD atau BW` + helper format resmi).
     * Field 2: `Nama Lengkap & Gelar Akademik *` (placeholder: `Dr. Ade Chandra Nugraha, S.Si., M.T.` + helper pencetakan dokumen).
     * Field 3: Status Dosen card container dengan toggle switch ON `#3965FB`.
     * Footer: Tombol `[ Batal ]` (Ghost) & `[ Simpan Dosen ]` (Solid primary `#3965FB`).
   - **C. SCR-ADM-004D: Modal Ubah Dosen**:
     * Modal dialog pop-up tengah (width: 500px, rounded-2xl).
     * Field `Inisial / Kode Dosen`: Terkunci / Disabled (🔒) berlatar `#F2F5FA`, font JetBrains Mono bold `"AD"` + badge `Terkunci 🔒` + helper: *"Inisial bersifat permanen dan tidak dapat diubah agar relasi jadwal kuliah tidak terputus."*
     * Field `Nama Lengkap & Gelar Akademik *`: State fokus border `#3965FB`, nilai terisi editable.
     * Field `Status Dosen`: Card container dengan toggle switch ON `#3965FB`.
     * Shortcut bahaya pojok kiri bawah: Teks merah `#D92D20` `"Nonaktifkan Dosen Ini"`.
     * Footer: Tombol `[ Batal ]` & `[ Simpan Perubahan ]` (Primary `#3965FB`).
   - **D. SCR-ADM-004E: Modal Konfirmasi Nonaktifkan Dosen**:
     * Modal dialog bahaya pop-up tengah (width: 460px, rounded-2xl, shadow-2xl).
     * Header: Avatar lingkaran merah muda `#FEE4E2` (44px) dengan ikon warning `#D92D20`, judul "Nonaktifkan Dosen?", pertanyaan spesifik menyebutkan nama & kode dosen.
     * Callout Jaminan Keamanan Data: Background `#FEF3F2`, border `#FECDCA`, 3 poin jaminan tegas `#B42318` (1. Jadwal Kuliah Aman, 2. Penyusunan Jadwal Baru, 3. Dapat Dipulihkan).
     * Footer: Tombol `[ Batal ]` (Outline netral `#D7DDE7`) & `[ Ya, Nonaktifkan Dosen ]` (Solid danger `#D92D20`).
   - **E. SCR-ADM-004G: Modal Impor & Sinkronisasi Dosen Dual-Tab**:
     * Modal dialog pop-up tengah (width: 620px, rounded-2xl, shadow-2xl).
     * Navigasi Dual-Tab: Tab 1 "Impor JSON Masal" & Tab 2 (Aktif) "⚡ Sinkronisasi Kurikulum POLBAN".
     * Edukasi Sistem: Callout `#EEF2FF` menjelaskan pemindaian data kurikulum `data/jadwal/*.json`, penyaringan inisial tim teaching bertanda `+` menjadi 47 dosen unik, serta jaminan Idempotent Safe.
     * Opsi 1 (Dosen Saja): Card border `#D7DDE7`, badge "47 Dosen Terdeteksi", tombol `[ ⚡ Sinkronkan Dosen ]` (Primary `#3965FB`) via `POST /api/v1/master/sync-jadwal`.
     * Opsi 2 (Semua Master): Card dashed border `#D7DDE7`, deskripsi 41 Matkul + 18 Ruangan + 47 Dosen, tombol `[ 🔄 Sinkronkan Semua ]` (Outline `#3965FB`) via `POST /api/v1/master/sync-all`.
     * Footer: Tombol `[ Tutup ]` (Background `#F2F5FA`, border `#D7DDE7`).

5. **SCR-ADM-005: Manajemen Kelas & Semester (Terdiri dari 3 Sub-Layar)**
   - **A. SCR-ADM-005A: Daftar Kelas (Grid Kelas)**
     - Header: Judul "Daftar Kelas", subtitle, tombol `[ + Buat kelas baru ]`.
     - Filter Bar 5 Kolom: Input Filter Prodi, Angkatan, Rombel, Dropdown Status (`Semua`, `ACTIVE`, `INACTIVE`, `ARCHIVED`), Tombol `Reset`.
     - Grid Kartu Kelas (3 Kolom di Desktop, 1 Kolom di Mobile):
       - Header Kartu (Soft Lavender `#E9EAFF`): Nama Kelas (misal: `D4-1A`), Prodi, Angkatan, Rombel, Slug URL portal, Status Badge Pill.
       - Body Kartu (Putih): Status KM (Nama KM, No WA, atau label "Undangan Menunggu 7 hari" / "Belum Ditetapkan").
       - Footer Kartu: Tombol `[ Lihat detail ]` atau tombol `[ Undang KM ]`.
   - **B. SCR-ADM-005B: Buat Kelas Baru (Formulir Terstandarisasi) - [100% AUDITED & APPROVED]**
     - Form Input:
       1. Program Studi (Dropdown datalist rekomendasi prodi POLBAN: D4 TI, D3 MI, dll).
       2. Angkatan (Input number, misal: `2025`).
       3. Rombel / Kelas (Input text uppercase, misal: `A` atau `B`).
     - Panel Tinjauan Otomatis: Live preview generate nama kelas (`D4 TI 2025 A`) dan slug URL (`d4-ti-2025-a`), dengan tombol toggle "Sesuaikan manual" bila ingin kustom.
     - Mode Portal Default: `LINK` (akses langsung via tautan) atau `CODE` (butuh 6 digit kode).
     - Tombol `[ Simpan kelas ]` (Primer) dan `[ Batal ]` (Outline).
   - **C. SCR-ADM-005C: Detail Kelas & Semester (Pusat Kendali Kelas) - [100% AUDITED & APPROVED]**
     - *Banner Mode Dukungan*: Muncul oranye di atas jika Admin sedang mengelola kelas dalam mode support.
     - Header: Nama Kelas, Identitas Prodi/Angkatan/Rombel, Slug portal, Status Badge.
     - *Kartu 1: Informasi Pengurus (KM)*: Nama KM, WhatsApp, Status KM, tombol `[ Ganti / Undang Ulang KM ]`.
     - *Kartu 2: Keamanan & Akses Portal*: Mode Portal aktif (`Tautan` / `Kode`), Versi kode portal, tombol `[ Mode Tautan ]`, `[ Mode Kode ]`, dan `[ Rotasi Kode Portal ]`.
     - *Bagian 3: Daftar Semester Kelas*:
       - Tabel/List Semester: Nama Semester (misal: `2024/2025 Ganjil`), Rentang Tanggal (`2024-09-01 â†’ 2025-01-31`), Badge Status (`ACTIVE` hijau, `DRAFT` kuning, `ARCHIVED` abu-abu).
       - Tombol muat ulang semester.
     - *Aksi Cepat Footer*: Tombol `[ Lihat Riwayat Perubahan kelas ]`, `[ Lihat Antrean Notifikasi ]`, dan tombol bahaya `[ Nonaktifkan Kelas ]` / `[ Arsipkan Kelas ]`.
   - **D. SCR-ADM-005D: Modal Impor Kurikulum Semester (3 States) - [100% AUDITED & APPROVED]**
     - Komponen modal pop-up tengah (max-w-md 480px, rounded-2xl, p-6, shadow-2xl).
     - Header: Judul "Impor Kurikulum", tombol close "âœ•", subtitle deskripsi tujuan impor draf semester.
     - *State 1 (Awal / Pilih Berkas)*: Dropzone dashed dengan ikon dokumen, helper text format JSON POLBAN di kiri, tautan `[ ðŸ“¥ Unduh Contoh Template ]` di kanan, tombol Validasi & Terapkan disabled.
     - *State 2 (Validasi Berhasil / READY)*: Box berkas terpilih (nama berkas .json, ukuran KB, tombol ganti), box hijau status `READY` (jumlah matkul, dosen, pola jadwal tanpa bentrok ruangan), tombol `[ ðŸ”„ Validasi Ulang ]` (outline) dan `[ ðŸš€ Terapkan ke Draf ]` (primer biru aktif).
     - *State 3 (Validasi Ditolak / Galat)*: Box berkas terpilih, box merah peringatan fatal galat (nomor baris bentrok ruangan/dosen tidak terdaftar), tombol `[ Periksa Berkas ]` aktif, tombol `[ Terapkan ke Draf ]` terkunci disabled.
   - **E. SCR-ADM-005E: Modal Tambah Sesi Perkuliahan (Master Data Picker) - [100% AUDITED & APPROVED]**
     - Komponen modal pop-up tengah (max-w-lg 540px, rounded-2xl, p-6, shadow-2xl).
     - Header: Judul "Tambah Sesi Perkuliahan", tombol close "âœ•", subtitle integrasi Master Data.
     - Field 1: Combobox Mata Kuliah Kurikulum + Segmented Control Jenis Sesi (Teori / Praktikum / Praktik).
     - Field 2: Combobox Dosen Pengampu (terhubung Master Dosen POLBAN).
     - Field 3: Grid 3 kolom (Dropdown Hari, Input Jam Mulai, Input Jam Selesai).
     - Field 4: Combobox Ruangan Kuliah + Live Availability Checker:
       * State A (Tersedia): Box hijau mint status ketersediaan ruangan & dosen.
       * State B (Bentrok): Box amber peringatan jadwal bentrok dengan kelas lain di jam yang sama (tombol simpan terkunci/disabled).
     - Footer: Tombol Batal & Tombol [ + Simpan ke Jadwal ].
    - **F. SCR-SCH-001: Rincian Jadwal Tetap Mingguan (Pola Jadwal Kelas - Active & Draft States) - [100% AUDITED & APPROVED]**
      - Breadcrumb: Sistem / Kelas & Semester / D4-TI 1A / Jadwal Mingguan.
      - Header & Metrik: 8 Matkul Kurikulum | 12 Sesi Mingguan | 4 Ruangan Digunakan | Badge Kesiapan (0 Bentrok).
      - Toolbar: Segmented tab filter hari (Semua Hari, Senin, Selasa, dst.) + Pencarian cepat / Tambah Sesi.
      - Konten Jadwal Harian: Kartu per hari (Senin, Selasa, dst.) dengan jam (JetBrains Mono), badge Teori/Praktikum, dosen pengampu, ruangan, tombol Ubah & Hapus.
      - State 1 (Semester Aktif): Tombol utama header [+ Tambah Sesi Perkuliahan].
      - State 2 (Semester Draf): Badge DRAFT (Persiapan), banner amber peringatan draf, tombol utama header [🚀 Terbitkan & Aktifkan], tombol [+ Tambah Sesi] di toolbar hari.
6. **SCR-ADM-006: Manajemen Pengguna & Penugasan Peran**
   - Daftar nomor WA pengguna, nama lengkap, role yang diemban, status akun (Aktif / Ditangguhkan), tombol Reset Sandi.
7. **SCR-ADM-007: Antrean Pesan WhatsApp (Broadcast & Notification Outbox) - [100% AUDITED & APPROVED]**
   - **Header**: Judul "Antrean WhatsApp", deskripsi fungsional, tombol aksi kanan: `[ ðŸ”„ Perbarui antrean ]`.
   - **3-Tab Navigation**:
     - `Menunggu` (Status PENDING/PROCESSING)
     - `Gagal` (Status FAILED/CANCELLED, badge indikator merah angka error, tombol aksi `[ Coba lagi ]` via `POST /api/v1/notifications/{id}/retry`)
     - `Riwayat` (Status SENT, 128 riwayat pesan terkirim, pagination 1-13)
   - **Tabel 6 Kolom Konsisten**: `PENERIMA` (Grup WA / Nomor target), `JENIS PESAN` (Pengingat jadwal, tugas baru, kuliah pengganti), `WAKTU` (Dijadwalkan / Terakhir dicoba / Terkirim pada), `PERCOBAAN` (Attempts), `STATUS` (Badge Menunggu / Gagal / Terkirim), `AKSI` (Lihat / Coba lagi).
   - **Modal Detail Pesan Notifikasi**: Grid metadata 2 kolom, bubble chat WhatsApp preview realistis (#EFEAE2, bubble putih/hijau, markdown WA), accordion raw JSON payload.
8. **SCR-ADM-008: Cadangan Data & Audit Global - [100% AUDITED & APPROVED]**
   - **Kartu Atas (Cadangan Data)**:
     - Header: Judul "Cadangan data", subtitle, tombol `[ â˜ï¸ Buat cadangan ]`.
     - Daftar File Cadangan: Item dengan icon db/file, nama file timestamp, checksum SHA-256 (format `sha256: 7d2fâ€¢â€¢â€¢â€¢9a1e`), cakupan kampus/kelas, alasan pencadangan, waktu & nama admin, badge hijau `VERIFIED`, tombol aksi `Unduh` & `Pulihkan`.
   - **Kartu Bawah (Audit Global)**:
     - Header: Judul "Audit Global", subtitle, filter bar pencarian tindakan/aktor/objek + dropdown `Semua tindakan` + `Semua periode`.
     - Tabel 5 Kolom: `WAKTU` | `AKTOR` | `TINDAKAN` | `OBJEK` | `ALASAN / RINCIAN`.
     - Badges Tindakan: Kuning `SUPPORT_ENTER`, Biru `CREATE_BACKUP`, Hijau `ACTIVATE_SEMESTER`, Ungu `INVITE_KM`.
     - Footer: Pagination log aktivitas.
   - **Modal Buat Cadangan Data**:
     - Cakupan: Radio/Dropdown pilihan `Seluruh Kampus (Global Database)` vs kelas tertentu (D4-TI 1A, D3-MI 2B).
     - Semester: Pilihan semester opsional.
     - Alasan Cadangan: Wajib min. 5 karakter (sesuai SQLite schema constraint `CHECK (length(trim(reason)) >= 5)`).
     - Info box keamanan SHA-256 terenkripsi di folder `storage/backups`.

---

## 6. POLA INTERAKSI & KOMPONEN STANDAR (UI PATTERNS)

Untuk memastikan konsistensi layout di seluruh halaman:

1. **Header Halaman**:
   - Selalu ada Judul Utama (`text-[28px] font-bold text-ink`) + Badge Jumlah Data + Subtitle Deskripsi 1 baris di kiri.
   - Tombol Aksi Utama selalu berada di sisi kanan atas (di mobile otomatis wrap rapi di bawah judul).
2. **Filter & Search Bar**:
   - Terdiri dari Search Field berikon pencarian di kiri + Dropdown Filter Status di kanan. Tinggi field konsisten 44px.
3. **Empty State Pattern**:
   - Jika data kosong: Ikon bulat besar dengan background `#E9EAFF`, teks judul bold, deskripsi singkat 2 baris, dan tombol aksi pemulihan utama (contoh: "Sinkronkan Sekarang" atau "+ Tambah Pertama Kali").
4. **Modal Dialog Pattern**:
   - Backdrop hitam transparan dengan blur halus (`bg-black/50 backdrop-blur-sm`).
   - Kartu modal di tengah: Header modal dengan tombol close `X` (44px), body modal dengan spacing form `gap-4`, footer modal dengan tombol Aksi Utama (Primer) dan tombol Batal (Ghost) berjajar rapi.

---

## 7. PROMPT INSTRUKSI SIAP PAKAI UNTUK PENCIL.DEV

Ketika Anda membuka Pencil.dev, salin dan masukkan perintah prompt berikut:

> *"Bertindaklah sebagai Senior UI/UX Designer. Saya melampirkan konteks arsitektur dan sistem desain lengkap dari aplikasi Bot Jadwal (Portal Kampus & Asisten Akademik). Tolong buatkan redesign antarmuka dengan ketentuan ketat:*
> 1. *Gunakan palet warna terkunci: Primary `#3965FB`, Soft `#E9EAFF`, Canvas `#F2F5FA`, Card `#FFFFFF`, Ink `#1F1F1F`, Muted `#667085`, Success `#E1FFB7`, Warning `#FCD484`, Danger `#FF6C48`.*
> 2. *Jaga konsistensi layout desktop (Sidebar 248px + Main Card) dan mobile (Card list + 4-Tab Bottombar min 44px).*
> 3. *Jangan berhalusinasi atau menambahkan fitur e-commerce / chatting sosial baru. Rancang persis modul yang tertera: Portal Mahasiswa, Ketua Murid (KM), PJ Mata Kuliah, dan System Admin.*
> 4. *Mulai dengan meredesign Layar [Sebutkan nama layar, misal: Master Data System Admin atau Dashboard KM] lengkap dengan Header aksi, Filter Bar, Tampilan Tabel Desktop, Tampilan Kartu Mobile, serta Modal Impor & Tambah Data."*
