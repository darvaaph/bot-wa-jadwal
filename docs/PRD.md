# Product Requirements Document (PRD)
# Bot Jadwal v2.0 & Web Admin Dashboard

* **Status:** Draft disetujui untuk implementasi
* **Versi:** 2.0.0
* **Tanggal:** 21 September 2026
* **Penulis:** Tim Pengembang bot-jadwal (Berdasarkan Riset 11 Responden Pengguna)
* **Target Rilis:** Semester Ganjil 2026/2027
* **Teknologi Utama:** Go (Backend REST API & Embedded Web Server), SQLite WAL Mode, Tailwind CSS CDN, Alpine.js, WhatsApp Automation Engine

---

## 1. Ringkasan Eksekutif dan Latar Belakang

Sistem `bot-jadwal` versi 1.0 telah digunakan oleh mahasiswa kelas untuk menerima informasi jadwal perkuliahan harian melalui obrolan grup WhatsApp. Namun, evaluasi operasional dan riset mendalam terhadap 11 responden (Ketua Murid dan Penanggung Jawab Mata Kuliah) mengungkapkan sejumlah kelemahan mendasar:

1. **Obrolan Grup Menumpuk (*Chat Clutter*):** Interaksi penambahan tugas dan pembaruan jadwal dilakukan melalui perintah teks (*command*) manual langsung di dalam grup WhatsApp. Hal ini mengotori ruang obrolan, memicu miskomunikasi, dan membuat pengumuman penting tenggelam.
2. **Keterbatasan Bot pada Jadwal Reguler:** Bot generasi pertama hanya mampu membaca jadwal perkuliahan statis. Ketika terjadi kuliah pengganti atau pergeseran jadwal dadakan, informasi di bot menjadi tidak akurat karena sistem tidak mendukung jadwal pengganti dinamis.
3. **Kelelahan Mengingatkan Tugas (*Reminder Fatigue*):** PJ Matkul harus mengingatkan mahasiswa berkali-kali secara manual melalui chat grup, sementara informasi deadline sering tersebar di berbagai tempat (WhatsApp, Microsoft Teams, Google Classroom, dan slide dosen).
4. **Pencarian Ruangan Kosong yang Merepotkan:** Mengatur kelas pengganti memakan waktu lama karena PJ harus mencocokkan jadwal dosen, jadwal mahasiswa, dan memeriksa ketersediaan ruangan kampus fisik ke pihak Tata Usaha (TU).
5. **Ketiadaan Pembatasan Hak Akses (*Role Ambiguity*):** Tidak ada batas wewenang yang jelas antara siapa yang berhak mengubah jadwal dan tugas untuk mata kuliah tertentu, sehingga memicu risiko salah ubah data (*human error*).

Untuk mengatasi masalah tersebut, **Bot Jadwal v2.0** memindahkan seluruh aktivitas input, manipulasi data, dan konfigurasi dari chat WhatsApp ke sebuah **Web Admin Dashboard** yang ringan, responsif mobile, dan terstruktur. Bot WhatsApp kini murni bertindak sebagai mesin siaran (*broadcast engine*) dan pengingat otomatis satu arah ke grup kelas.

---

## 2. Persona Pengguna

Berdasarkan riset pengguna terhadap 11 responden, pengguna sistem dibagi menjadi tiga kelompok persona utama:

### Persona 1: Ketua Murid / KM (Pengawas & Koordinator Utama)
* **Representasi Responden:** Imam (KM) `[19]`
* **Karakteristik:** Bertanggung jawab atas koordinasi seluruh perkuliahan di kelas, hubungan dengan dosen, dan perizinan ruangan dengan Tata Usaha (TU).
* **Tujuan:**
  * Memastikan seluruh jadwal perkuliahan dan kelas pengganti berjalan tertib.
  * Memiliki kendali penuh (*full access*) untuk memantau, memvalidasi, dan mengedit data seluruh mata kuliah.
  * Meniadakan obrolan grup yang berantakan akibat teks perintah bot.
* **Kebutuhan Kunci:**
  * Fitur jadwal pengganti sementara dengan otomatis kembali (*auto-revert*) ke jadwal reguler.
  * Alur validasi atau draf sebelum perubahan jadwal diumumkan massal ke grup.
  * Log riwayat perubahan untuk melacak aktivitas pengubahan data oleh para PJ.

### Persona 2: Penanggung Jawab Mata Kuliah / PJ Matkul (Operator Spesifik)
* **Representasi Responden:** 
  * Hana & Giza (PJ Alin Teori), Anindya (PJ Alin Praktik)
  * Kemal (PJ PLP Teori), Bima (PJ PLP Praktik)
  * Faqih (PJ SDB Praktik), Iman (PJ SDB Teori)
  * Arsel (PJ OS Praktek), Irfan (PJ OS Teori), Afzhal (PJ Matdis Praktek)
* **Karakteristik:** Mahasiswa yang bertugas mengurus satu mata kuliah tertentu, mencatat tugas dari dosen, dan mengumumkan deadline kepada kelas. Sering mengakses web dari ponsel di ruang kelas.
* **Tujuan:**
  * Mencatat tugas baru dan deadline dalam waktu singkat (< 30 detik) tanpa ribet menghafal format perintah teks bot.
  * Menghilangkan beban mengingatkan teman-teman sekelas berkali-kali.
* **Kebutuhan Kunci:**
  * Formulir input terpandu berbasis dropdown dan kolom tempat pengumpulan tugas.
  * Hak akses terisolasi pada mata kuliah yang menjadi tanggung jawabnya agar tidak bersenggolan dengan mata kuliah lain.
  * Mode draf untuk mencatat perubahan jadwal tentatif sambil menunggu kepastian dosen.

### Persona 3: Mahasiswa Kelas (Penerima Manfaat / End-User)
* **Karakteristik:** Seluruh anggota kelas yang membutuhkan kepastian jadwal dan tugas kuliah sehari-hari.
* **Tujuan:**
  * Mengetahui jadwal kuliah hari ini dan lokasi ruangan kelas sebelum berangkat ke kampus.
  * Mengetahui daftar tugas aktif yang harus dikumpulkan dalam waktu dekat agar tidak terlambat mengumpulkan.
* **Kebutuhan Kunci:**
  * Siaran WhatsApp yang ringkas, jelas, dan tidak membanjiri ruang obrolan.
  * Notifikasi perubahan jadwal yang menunjukkan perbandingan jadwal lama vs jadwal baru (*diff*).
  * Pengingat tugas harian yang dikirim pada sore hari (saat pulang kuliah).
  * Tautan web langsung (*deep-link*) untuk membaca rincian instruksi tugas tanpa harus mencari dokumen yang tercecer.

---

## 3. Tujuan Produk (Goals) & Batasan Ruang Lingkup (Non-Goals)

### A. Tujuan Produk (Goals)
1. **Memindahkan 100% Interaksi Admin ke Web:** Tidak ada lagi perintah teks input (`/tambah`, `!jadwal`, dll) di dalam grup WhatsApp. Semua input dilakukan via web dashboard.
2. **Otomatisasi Siaran Jadwal & Pengingat Tugas:** Menjalankan dua siklus pengingat terjadwal otomatis: pagi hari (pukul 06.00 WIB untuk jadwal hari ini) dan sore hari (pukul 17.00 WIB untuk pengingat tugas aktif).
3. **Mendukung Siklus Hidup Jadwal Pengganti (*Full Lifecycle*):** Mendukung pencatatan jadwal pengganti dinamis, status draf, notifikasi perubahan komparatif, dan kembali otomatis ke jadwal semula (*auto-revert*).
4. **Menerapkan Hak Akses Berbasis Peran (RBAC):** Membatasi hak edit PJ hanya pada mata kuliahnya masing-masing, sementara KM memegang wewenang penuh.
5. **Format Pesan Ringkas & Terstruktur:** Pesan siaran di WhatsApp hanya memuat informasi esensial dan menyediakan tautan web menuju dashboard untuk rincian lengkap.

### B. Batasan Ruang Lingkup (Non-Goals)
1. **Bukan Pengganti Learning Management System (LMS):** Sistem tidak melayani pengumpulan berkas tugas fisik mahasiswa, pengunggahan file tugas besar, atau sistem penilaian nilai (*grading*). Sistem hanya mencatat metadata tugas dan tautan portal pengumpulan.
2. **Tidak Mengelola Acara Non-Akademik Eksternal:** Sistem tidak memfasilitasi pencatatan agenda organisasi mahasiswa (HIMA), buka bersama, atau acara santai di luar perkuliahan akademik agar jadwal tetap fokus.
3. **Tidak Ada Generator Kelompok Tugas:** Sistem tidak memuat modul pembagian kelompok otomatis (*group generator*).
4. **Tidak Ada Polling Internal di Web:** Fitur jajak pendapat tetap memanfaatkan polling bawaan WhatsApp yang sudah tersedia di grup.
5. **Tidak Terintegrasi dengan Sistem Birokrasi Pusat Universitas:** Sistem beroperasi mandiri di level kelas/angkatan tanpa integrasi API resmi ke server kampus atau Tata Usaha pusat.

---

## 4. Kebutuhan Fungsional (Functional Requirements)

### Epic 1: Autentikasi dan Manajemen Hak Akses (RBAC)
* **Deskripsi:** Mengamankan akses web dashboard dan memastikan isolasi kewenangan antar-pengurus.
* **Kebutuhan:**
  * **FR-1.1:** Sistem mendukung dua tingkatan peran: `Ketua Murid (KM)` dan `Penanggung Jawab Mata Kuliah (PJ)`.
  * **FR-1.2:** Pengguna dengan peran KM memiliki wewenang penuh (membaca, menambah, mengedit, menghapus, mempublikasikan) untuk seluruh mata kuliah kelas.
  * **FR-1.3:** Pengguna dengan peran PJ hanya memiliki wewenang edit pada mata kuliah spesifik yang ditugaskan kepadanya. Pada mata kuliah lain, PJ hanya memiliki hak baca (*read-only*).
  * **FR-1.4:** Sistem menerapkan mekanisme autentikasi berbasis sesi atau token JWT yang aman pada backend Go.
  * **FR-1.5:** Mahasiswa umum dapat mengakses halaman utama dashboard secara publik (tanpa login) untuk melihat jadwal dan daftar tugas dengan status *read-only*.

### Epic 2: Manajemen Jadwal Perkuliahan & Kelas Pengganti
* **Deskripsi:** Pengelolaan jadwal perkuliahan harian, mingguan, dan kelas pengganti dinamis.
* **Kebutuhan:**
  * **FR-2.1 (Jadwal Reguler):** Menyimpan jadwal rutin mingguan (Senin sampai Jumat) mencakup nama mata kuliah, jenis (teori/praktik), dosen, ruangan, jam mulai, dan durasi SKS/jam pelajaran.
  * **FR-2.2 (Kalkulasi Otomatis Jam Selesai):** Saat menginput jam mulai dan durasi jam pelajaran, antarmuka web secara otomatis mengkalkulasikan jam selesai perkuliahan.
  * **FR-2.3 (Jadwal Pengganti Dinamis):** Admin dapat membuat jadwal pengganti (*replacement class*) untuk menggantikan sesi perkuliahan tertentu pada tanggal dan jam spesifik.
  * **FR-2.4 (Pembedaan Sementara vs Permanen):** Admin dapat menandai apakah perubahan jadwal bersifat sementara (hanya berlaku pada tanggal tertentu) atau permanen (mengubah jadwal mingguan seterusnya).
  * **FR-2.5 (Kembali Otomatis / Auto-Revert):** Setelah tanggal dan jam pelaksanaan jadwal sementara terlewati, sistem secara otomatis mengembalikan jadwal perkuliahan ke jadwal reguler awal tanpa tindakan manual.
  * **FR-2.6 (Mode Draf / Draft State):** Admin dapat menyimpan perubahan jadwal dengan status `Draf`. Jadwal berstatus draf tidak akan disiarkan ke WhatsApp hingga admin menekan tombol `Publikasikan`.
  * **FR-2.7 (Status Kelas Kosong / Diliburkan):** Admin dapat menandai sesi perkuliahan tertentu sebagai "Diliburkan" atau "Ditiadakan".

### Epic 3: Manajemen Tugas Kuliah (Assignment Tracker)
* **Deskripsi:** Pencatatan, pemantauan, dan penyaringan tugas kuliah aktif.
* **Kebutuhan:**
  * **FR-3.1 (Formulir Tugas Terstruktur):** Formulir penambahan tugas wajib memuat kolom:
    1. Mata Kuliah (pilihan dropdown)
    2. Judul / Topik Tugas (teks ringkas)
    3. Deskripsi Tugas / Instruksi (area teks)
    4. Tenggat Waktu (pemilih tanggal dan jam deadline)
    5. Tempat Pengumpulan Tugas (contoh: link Google Classroom, LMS, MS Teams, atau fisik)
  * **FR-3.2 (Pengelompokan Deadline / Buckets):** Daftar tugas pada dashboard dikelompokkan secara visual ke dalam 4 kategori batas waktu:
    1. *Hari Ini* (deadline jatuh pada hari yang sama)
    2. *Minggu Ini* (deadline dalam 1 sampai 7 hari ke depan)
    3. *Mendatang* (deadline > 7 hari ke depan)
    4. *Terlewat / Selesai* (arsip tugas)
  * **FR-3.3 (Filter dan Pengurutan):** Pengguna dapat menyaring tugas berdasarkan mata kuliah dan mengurutkan berdasarkan deadline terdekat.
  * **FR-3.4 (Penyelesaian Tugas):** Admin dapat menandai tugas sebagai "Selesai" untuk mengarsipkannya dari daftar tugas aktif.

### Epic 4: Mesin Siaran & Notifikasi WhatsApp
* **Deskripsi:** Otomatisasi pengiriman pesan informatif ke grup WhatsApp kelas.
* **Kebutuhan:**
  * **FR-4.1 (Siaran Jadwal Harian Pagi):** Sistem otomatis mengirimkan pesan jadwal perkuliahan hari ini setiap pagi pukul 06.00 WIB. Memuat: daftar mata kuliah hari ini, jam, ruangan, nama dosen, dan tautan daring jika ada.
  * **FR-4.2 (Siaran Pengingat Tugas Sore):** Sistem otomatis mengirimkan daftar tugas aktif setiap sore pukul 17.00 WIB (setelah jam kuliah selesai), menyoroti tugas yang memiliki deadline dalam waktu dekat.
  * **FR-4.3 (Notifikasi Perubahan Jadwal Komparatif / Diff):** Ketika jadwal pengganti dipublikasikan dari dashboard, sistem seketika mengirimkan notifikasi siaran dengan format komparatif:
    * Mata kuliah yang berubah
    * Jadwal Lama (`Sebelum: Hari, Jam, Ruangan`)
    * Jadwal Baru (`Menjadi: Hari, Jam, Ruangan`)
    * Keterangan dari dosen/PJ
  * **FR-4.4 (Format Pesan Ringkas & Tautan Dalam / Deep-link):** Pesan bot dirancang padat tanpa deskripsi panjang yang memenuhi layar. Di bagian bawah pesan selalu disertakan tautan langsung menuju web dashboard: `Detail lengkap: https://[domain-dashboard]/tugas/[id]`.
  * **FR-4.5 (Pengingat Menjelang Kelas Pengganti):** Khusus kelas pengganti, sistem mengirimkan notifikasi pengingat tambahan 2 jam sebelum kelas dimulai agar mahasiswa tidak lupa hadir.

### Epic 5: Manajemen Ruangan Kampus
* **Deskripsi:** Pencatatan dan pemantauan ketersediaan ruangan kelas fisik.
* **Kebutuhan:**
  * **FR-5.1 (Katalog Ruangan):** Menyimpan daftar ruangan perkuliahan kelas (contoh: Lab 301, Lab 302, R. Teori 3, Auditorium).
  * **FR-5.2 (Pelacak Ruangan Kosong Internal):** Berdasarkan basis data seluruh jadwal perkuliahan yang tercatat di sistem, dashboard menyediakan fitur pencarian untuk melihat ruangan mana saja yang sedang tidak terpakai pada hari dan rentang jam tertentu.

### Epic 6: Riwayat Perubahan (Audit Trail / Change Log)
* **Deskripsi:** Transparansi pencatatan aktivitas perubahan data.
* **Kebutuhan:**
  * **FR-6.1:** Sistem mencatat setiap aksi penambahan, perubahan, dan penghapusan jadwal atau tugas ke dalam tabel `change_logs`.
  * **FR-6.2:** Data yang dicatat meliputi: Nama pengguna, peran, jenis aksi, entitas yang diubah, waktu perubahan, serta status sebelum dan sesudah perubahan.
  * **FR-6.3:** Riwayat perubahan dapat dilihat oleh KM dan PJ pada menu Riwayat di dashboard.

---

## 5. Kebutuhan Antarmuka Pengguna (UI/UX Specifications)

### A. Struktur Navigasi Dashboard
Dashboard menggunakan model Single Page Application (SPA) responsif dengan 4 tab navigasi utama:
1. **Ringkasan (Overview):** Kartu kuliah hari ini, widget mini tugas mendesak (< 24 jam), dan indikator status bot.
2. **Jadwal Kuliah (Schedule):** Tampilan jadwal Senin sampai Jumat dengan pemilih hari, penanda kelas pengganti (badge kuning), dan tombol ubah jadwal (khusus admin).
3. **Daftar Tugas (Tasks):** Pengelompokan tugas berdasarkan deadline, filter mata kuliah, tombol tambah tugas, dan modal formulir input.
4. **Ruangan & Riwayat (Rooms & Logs):** Pencarian slot ruangan kosong internal dan log riwayat pembaruan sistem.

### B. Pedoman Tampilan Mobile-First
* Seluruh tombol aksi utama dan input formulir memiliki target sentuh minimal **44 x 44 piksel** agar nyaman dioperasikan satu jempol pada layar smartphone (390 x 844 px).
* Modal formulir input pada layar mobile menggunakan gaya *bottom sheet* (muncul dari bawah layar) dengan tombol simpan yang mudah dijangkau.
* Navigasi mobile diletakkan di bagian bawah layar (*bottom navigation bar*).

---

## 6. Kebutuhan Non-Fungsional (Non-Functional Requirements)

1. **Kinerja & Kecepatan:**
   * Waktu muat awal (*initial load*) halaman web dashboard di bawah 1,5 detik pada jaringan seluler 4G.
   * Aksi penyimpanan formulir jadwal dan tugas merespons dalam waktu kurang dari 300 ms.
2. **Keandalan Basis Data:**
   * Basis data SQLite wajib berjalan dalam **WAL Mode (Write-Ahead Logging)** dengan opsi `busy_timeout` terkonfigurasi untuk mendukung konkurensi pembacaan oleh web dan penulisan oleh bot/admin secara simultan tanpa galat *database locked*.
3. **Penyajian Aset Terintegrasi (Self-Contained Deployment):**
   * Seluruh berkas antarmuka web (HTML, CSS, JS) di-embed langsung ke dalam *binary* Go menggunakan pustaka bawaan `embed` Go, menghasilkan satu berkas *executable* tunggal yang siap dijalankan di server tanpa ketergantungan eksternal yang rumit.
4. **Keamanan Data:**
   * Proteksi CSRF pada seluruh endpoint mutasi data (POST, PUT, DELETE).
   * Kata sandi akun pengurus disimpan menggunakan algoritma hashing standar industri (`bcrypt` atau `Argon2`).
5. **Kepatuhan Penulisan (Copywriting Hygiene):**
   * Seluruh teks antarmuka, pesan siaran WhatsApp, dan dokumentasi sistem mematuhi aturan anti-slop: bebas dari kata-kata klise/buzzword kosong, tidak menggunakan em dash (`—`), dan mengutamakan fakta spesifik yang akurat.

---

## 7. Model Data Utama (Database Schema Overview)

```text
┌───────────────────────┐        ┌─────────────────────────┐
│         users         │        │        subjects         │
├───────────────────────┤        ├─────────────────────────┤
│ id (PK)               │        │ id (PK)                 │
│ username              │   ┌───<│ code                    │
│ password_hash         │   │    │ name                    │
│ role (km/pj)          │   │    │ lecturer                │
│ subject_id (FK, opt)  │>──┘    │ default_room            │
└───────────────────────┘        └─────────────────────────┘
                                              │
              ┌───────────────────────────────┴───────────────────────────────┐
              ▼                                                               ▼
┌─────────────────────────┐                                     ┌─────────────────────────┐
│        schedules        │                                     │          tasks          │
├─────────────────────────┤                                     ├─────────────────────────┤
│ id (PK)                 │                                     │ id (PK)                 │
│ subject_id (FK)         │                                     │ subject_id (FK)         │
│ type (regular/override) │                                     │ title                   │
│ date (YYYY-MM-DD, opt)  │                                     │ description             │
│ day_of_week (1-7)       │                                     │ submission_target       │
│ start_time (HH:MM)      │                                     │ deadline (DATETIME)     │
│ end_time (HH:MM)        │                                     │ status (active/done)    │
│ room                    │                                     │ created_by (FK users)   │
│ is_temporary (BOOLEAN)  │                                     │ created_at (DATETIME)   │
│ status (draft/published)│                                     └─────────────────────────┘
│ note                    │
└─────────────────────────┘
              │
              ▼
┌─────────────────────────┐
│       change_logs       │
├─────────────────────────┤
│ id (PK)                 │
│ user_id (FK)            │
│ entity_type             │
│ entity_id               │
│ action (create/upd/del) │
│ diff_summary            │
│ timestamp (DATETIME)    │
└─────────────────────────┘
```

---

## 8. Template Pesan Siaran WhatsApp

### Template 1: Siaran Rutin Jadwal Pagi (Pukul 06.00 WIB)
```text
*JADWAL KULIAH HARI INI*
Hari: [Hari], [Tanggal]
Kelas: [Nama Kelas]

1. *[Mata Kuliah 1]* ([Teori/Praktik])
   Pukul: [Jam Mulai] - [Jam Selesai] WIB
   Ruang: [Nama Ruangan]
   Dosen: [Nama Dosen]
   [Link Daring jika ada]

2. *[Mata Kuliah 2]* ([Teori/Praktik])
   Pukul: [Jam Mulai] - [Jam Selesai] WIB
   Ruang: [Nama Ruangan]
   Dosen: [Nama Dosen]

Detail lengkap dan tugas aktif:
https://[domain-dashboard]
```

### Template 2: Siaran Rutin Pengingat Tugas Sore (Pukul 17.00 WIB)
```text
*PENGINGAT TUGAS KULIAH AKTIF*
Hari: [Hari], [Tanggal]

[Daftar Tugas dengan Urutan Deadline Terdekat]:
1. *[Nama Mata Kuliah]* - [Judul Tugas]
   Deadline: [Hari], [Tanggal] ([Jam] WIB)
   Tempat Kumpul: [Google Classroom / LMS / Email]
   Sisa Waktu: [X hari / Y jam lagi]

2. *[Nama Mata Kuliah]* - [Judul Tugas]
   Deadline: [Hari], [Tanggal] ([Jam] WIB)
   Tempat Kumpul: [Google Classroom / LMS / Email]
   Sisa Waktu: [X hari / Y jam lagi]

Lihat deskripsi dan berkas tugas:
https://[domain-dashboard]/tasks
```

### Template 3: Notifikasi Perubahan Jadwal Komparatif (*Schedule Diff*)
```text
*PEMBERITAHUAN PERUBAHAN JADWAL*
Mata Kuliah: [Nama Mata Kuliah] ([Teori/Praktik])
Dosen: [Nama Dosen]

*Perubahan*:
Sebelum : [Hari Lama], [Jam Lama] WIB ([Ruangan Lama])
Menjadi : [Hari Baru], [Tanggal Baru], [Jam Baru] WIB ([Ruangan Baru])

Status: [Kelas Pengganti Sementara / Perubahan Permanen]
Keterangan: [Pesan dari dosen atau PJ]

Periksa pembaruan jadwal di web dashboard:
https://[domain-dashboard]
```
