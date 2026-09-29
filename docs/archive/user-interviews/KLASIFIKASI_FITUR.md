# Klasifikasi dan Analisis Relevansi Fitur: Bot Jadwal v2.0 & Web Dashboard Admin

Dokumen ini mengklasifikasikan seluruh usulan fitur yang muncul dari 11 sesi wawancara pengguna (Ketua Murid dan PJ Mata Kuliah). Seluruh usulan dipilah ke dalam kategori modul sistem, kemudian dievaluasi berdasarkan relevansinya terhadap tujuan utama sistem: **otomatisasi informasi perkuliahan, pengelolaan tugas terpusat, dan eliminasi kerumitan operasional kelas**.

---

## 1. Kriteria Penilaian Relevansi

Penilaian relevansi fitur didasarkan pada 3 parameter:
1. **Dampak Langsung (*Problem-Solution Fit*):** Apakah fitur secara langsung menyelesaikan masalah utama (obrolan grup menumpuk, lupa tugas, jadwal bentrok/dadakan, dan command bot yang rumit)?
2. **Kesesuaian Ruang Lingkup (*Domain Scope*):** Apakah fitur berfokus pada ranah jadwal dan tugas akademik kelas, atau melebar ke urusan sosial dan organisasi non-kuliah?
3. **Kelayakan Teknis (*Technical Feasibility & Complexity*):** Apakah fitur dapat diimplementasikan secara efisien pada arsitektur Go, SQLite WAL, Tailwind CSS, Alpine.js, dan integrasi WhatsApp yang ada?

---

## 2. Matriks Klasifikasi Seluruh Usulan Fitur

### A. Modul Manajemen Jadwal Perkuliahan

| Fitur yang Diusulkan | Pengusul | Status Relevansi | Alasan dan Pertimbangan Teknis |
|---|---|---|---|
| **Dukungan Jadwal Pengganti Dinamis** | Anindya, Kemal, Bima, Imam KM | **Sangat Relevan (Prioritas Utama)** | Bot lama hanya membaca jadwal statis. Penanganan kelas pengganti adalah kebutuhan paling mendesak bagi seluruh PJ. |
| **Pembedaan Jadwal Sementara vs Permanen (*Auto-Revert*)** | Imam KM | **Sangat Relevan (Prioritas Utama)** | Kelas pengganti umumnya hanya berlaku 1 sesi. Fitur *auto-revert* mengembalikan jadwal reguler secara otomatis tanpa perlu intervensi manual ulang. |
| **Mode Draf Perubahan Jadwal (*Draft State*)** | Giza | **Sangat Relevan (Prioritas Utama)** | PJ sering menerima info tentatif. Draf memungkinkan data disimpan lebih awal dan baru dipublikasikan setelah dosen memberi kepastian. |
| **Kalkulasi Otomatis Jam Selesai Kuliah** | Faqih | **Sangat Relevan (Prioritas Utama)** | Sederhana diterapkan di antarmuka web (jam awal + durasi jam pelajaran/SKS = jam selesai). Mencegah salah ketik waktu. |
| **Formulir Input Berurutan dengan Pilihan Dropdown** | Hana, Giza, Kemal | **Sangat Relevan (Prioritas Utama)** | Mengurangi salah ketik (*human error*). Dropdown untuk mata kuliah, kelas, dan jam pelajaran mempercepat proses input data. |
| **Kalender Visual yang Bisa Diklik per Tanggal** | Anindya, Hana, Afzhal | **Relevan (Fase Lanjutan / Sekunder)** | Representasi kalender visual sangat membantu peninjauan jadwal bulanan. Untuk tahap awal (MVP), tampilan tabel dan list per hari dapat diterapkan terlebih dahulu sebelum kalender grafis penuh. |
| **Penyuntingan Jadwal Langsung di Tabel (*Inline Editable Table*)** | Kemal | **Relevan (Fase Lanjutan / Alternatif UI)** | Berguna untuk manipulasi cepat banyak data sekaligus ala spreadsheet, namun implementasi form modal/pop-up lebih aman dari risiko tersenggol atau salah ubah data (*unintended keystrokes*). |
| **Interaksi Geser-Lepas (*Drag and Drop*) pada Kalender** | Afzhal | **Tidak Perlu / Kurang Relevan** | Kompleksitas pembuatan UI drag-and-drop kalender cukup tinggi di Alpine.js/mobile web, sementara frekuensi geser jadwal per minggu relatif rendah. Cukup menggunakan pemilih tanggal/jam biasa. |

---

### B. Modul Manajemen Tugas Kuliah

| Fitur yang Diusulkan | Pengusul | Status Relevansi | Alasan dan Pertimbangan Teknis |
|---|---|---|---|
| **Form Tambah & Perbarui Tugas Terstruktur** | Seluruh Responden (11 orang) | **Sangat Relevan (Prioritas Utama)** | Fondasi utama sistem. Menghilangkan proses input manual teks panjang di chat WhatsApp. |
| **Kolom Tempat/Portal Pengumpulan Tugas** | Kemal, Bima | **Sangat Relevan (Prioritas Utama)** | Mahasiswa sering bingung di mana tugas harus dikumpulkan (Google Classroom, LMS, MS Teams, email, atau fisik). Kolom ini wajib ada. |
| **Pengelompokan Deadline (*Deadline Buckets*)** | Bima, Faqih, Afzhal, Hana | **Sangat Relevan (Prioritas Utama)** | Memilah tugas berdasarkan urgensi waktu (Hari Ini, Minggu Ini, Mendatang, Terlewat) membantu mahasiswa menentukan prioritas pengerjaan. |
| **Pencarian dan Filter Tugas Multifaktor** | Iman, Irfan, Afzhal, Hana, Anindya | **Sangat Relevan (Prioritas Utama)** | Filter berdasarkan mata kuliah dan urutan deadline terdekat mempermudah pelacakan tugas aktif. |
| **Pelacakan Status Penyelesaian Tugas (*Task Done Tracker*)** | Afzhal, Irfan | **Relevan (Sekunder)** | Berguna untuk menandai tugas yang sudah lewat/selesai di dashboard admin agar tampilan tetap rapi. |
| **Alur Verifikasi dan Persetujuan Tugas oleh KM (*Approval Flow*)** | Faqih | **Relevan (Opsional / Peran Tertentu)** | Dapat diakomodasi melalui konsep status draf tugas yang dapat disetujui KM sebelum notifikasi disiarkan. |

---

### C. Modul Pengingat & Notifikasi WhatsApp

| Fitur yang Diusulkan | Pengusul | Status Relevansi | Alasan dan Pertimbangan Teknis |
|---|---|---|---|
| **Notifikasi Perubahan Jadwal Komparatif (*Schedule Diff*)** | Kemal | **Sangat Relevan (Prioritas Utama)** | Format notifikasi menyajikan perbandingan data lama vs data baru (`Sebelum -> Menjadi`). Menghilangkan miskomunikasi di grup. |
| **Jadwal Pengingat Tugas Sore Hari (*Post-Class Timing*)** | Bima | **Sangat Relevan (Prioritas Utama)** | Mengirimkan reminder tugas aktif setiap sore hari (pukul 16.30 atau 17.00 saat mahasiswa pulang kuliah) sangat tepat waktu. |
| **Jadwal Siaran Rutin Pagi Hari (*Daily Morning Schedule*)** | Arsel, Afzhal, Anindya | **Sangat Relevan (Prioritas Utama)** | Siaran otomatis pukul 06.00 atau 06.30 WIB untuk jadwal kuliah hari ini dan pengingat kelas pertama. |
| **Format Siaran Ringkas dengan Tautan Web (*Deep-link*)** | Faqih, Hana, Bima | **Sangat Relevan (Prioritas Utama)** | Pesan WhatsApp dibuat padat (hanya nama matkul, deadline, dan ringkasan). Detail lengkap dan lampiran diakses via tautan web agar grup tidak penuh. |
| **Dua Tahap Notifikasi Kelas Pengganti** | Kemal, Anindya | **Sangat Relevan (Prioritas Utama)** | Tahap 1: Notifikasi instan saat perubahan disimpan. Tahap 2: Notifikasi pengingat otomatis menjelang jam pelaksanaan kelas pengganti. |
| **Otomatisasi Pin dan Unpin Pesan WhatsApp (*Auto-Pin Message*)** | Afzhal | **Tidak Perlu / Terkendala Teknis** | Mayoritas pustaka bot WhatsApp (seperti whatsmeow atau Baileys) memiliki keterbatasan atau risiko tinggi (*banned*) saat memanipulasi aksi admin seperti pin/unpin di grup resmi. Pesan siaran terstruktur sudah cukup efektif. |
| **Saluran Khusus Pengumuman Terpisah (*Dedicated Announcement Channel*)** | Bima | **Relevan (Alternatif Kebijakan Grup)** | Bukan fitur kode aplikasi, melainkan implementasi operasional kelas (membuat grup khusus pengumuman satu arah di WhatsApp di mana bot menjadi admin). |

---

### D. Modul Tata Kelola Pengguna & Hak Akses (RBAC)

| Fitur yang Diusulkan | Pengusul | Status Relevansi | Alasan dan Pertimbangan Teknis |
|---|---|---|---|
| **Pembatasan Hak Akses Berbasis Mata Kuliah (RBAC)** | Bima, Imam KM | **Sangat Relevan (Prioritas Utama)** | PJ Matkul hanya memiliki hak edit untuk mata kuliah yang diampunya. KM memiliki hak penuh (*full access*) untuk seluruh mata kuliah. Menjamin integritas data. |
| **Pencatatan Riwayat Perubahan (*Audit Trail / Change Log*)** | Hana, Anindya, Kemal, Iman | **Sangat Relevan (Prioritas Utama)** | Mencatat riwayat pengubahan jadwal atau tugas (siapa yang mengubah, kapan, dan perubahan apa yang dilakukan) untuk transparansi antar-pengurus. |
| **Panduan Awal Pengguna Baru (*Onboarding Tutorial / Empty State*)** | Imam KM | **Relevan (Sekunder)** | Banner panduan atau *tooltip* pengoperasian awal di web dashboard membantu PJ baru yang berganti setiap semester. |
| **Statistik Penggunaan Mingguan di Halaman Depan Dashboard** | Faqih | **Tidak Perlu / Kurang Relevan** | Statistik jumlah klik atau aktivitas mingguan tidak memberikan nilai tambah fungsional bagi mahasiswa maupun PJ. Mengalihkan fokus dari jadwal dan tugas utama. |

---

### E. Modul Manajemen Ruangan Kampus

| Fitur yang Diusulkan | Pengusul | Status Relevansi | Alasan dan Pertimbangan Teknis |
|---|---|---|---|
| **Pencatatan Ruangan Kelas per Jadwal** | Seluruh Responden | **Sangat Relevan (Prioritas Utama)** | Data ruangan wajib ditampilkan di web dashboard dan siaran notifikasi WhatsApp. |
| **Pelacak Ketersediaan Ruangan Kosong Antar-Kelas Internal** | Kemal, Faqih, Anindya, Arsel, Irfan | **Relevan (Fase Lanjutan / Bertahap)** | Sistem dapat memeriksa jadwal seluruh kelas yang terdaftar pada database bot internal untuk mengetahui ruangan mana yang sedang kosong. |
| **Integrasi Sistem Manajemen Ruangan Terpusat Seluruh Kampus / TU** | Kemal, Giza, Imam KM | **Tidak Perlu / Di Luar Cakupan** | Sistem bot-jadwal adalah aplikasi mandiri tingkat kelas/angkatan. Integrasi langsung ke basis data kampus/TU membutuhkan birokrasi perizinan dan API resmi universitas yang berada di luar kendali pengurus kelas. |

---

### F. Modul Ekstra & Pendukung Belajar

| Fitur yang Diusulkan | Pengusul | Status Relevansi | Alasan dan Pertimbangan Teknis |
|---|---|---|---|
| **Repositori Berkas & Tautan Materi Kuliah (*Resource Hub*)** | Afzhal, Anindya | **Relevan (Fase Lanjutan / Pelengkap)** | Tab sederhana di dashboard untuk menyimpan kumpulan link Google Drive, materi PPT, atau modul per mata kuliah agar tidak tercecer. |
| **Pencatatan Acara Non-Akademik (HIMA, Bukber, Rapat Kelas)** | Faqih | **Tidak Perlu / Kurang Relevan** | Sistem difokuskan secara murni untuk jadwal akademik perkuliahan dan tugas kuliah. Menggabungkan agenda organisasi mahasiswa (HIMA) atau acara santai (bukber) berisiko mengotori kalender dan pesan bot. |
| **Fitur Pembagian dan Pemetaan Kelompok Mahasiswa (*Group Mapping*)** | Faqih | **Tidak Perlu / Di Luar Cakupan** | Logika pembuat kelompok tugas (*group generator*) berada di luar ruang lingkup aplikasi penjadwalan (*scheduling system*). Proses ini lebih efektif dilakukan manual atau menggunakan alat terpisah. |
| **Fitur Polling Pemilihan Waktu Pengganti di Dashboard** | Kemal | **Tidak Perlu / Redundan** | Fitur polling sudah tersedia secara bawaan (*native*) dan sangat mudah digunakan langsung di dalam aplikasi WhatsApp. Membuat modul polling terpisah di web dashboard bersifat redundan dan memakan waktu pengembangan. |

---

## 3. Rangkuman Pemisahan Fitur

### Kelompok 1: Fitur Relevan & Esensial (Masuk Ruang Lingkup v2.0)
1. **Dukungan Jadwal Pengganti dengan *Auto-Revert*** (otomatis kembali ke jadwal reguler).
2. **Mode Draf Perubahan Jadwal** (persiapan perubahan sebelum disetujui dosen).
3. **Formulir Terpandu Berbasis Dropdown & Kalkulasi Otomatis Jam Selesai**.
4. **Formulir Tambah & Kelola Tugas** (dilengkapi kolom tempat/link pengumpulan).
5. **Pemisahan Kategori Deadline Tugas** (Hari Ini, Minggu Ini, Mendatang).
6. **Hak Akses Berbasis Peran (RBAC)** (KM = Full, PJ = Terbatas pada matkulnya).
7. **Pencatatan Riwayat Perubahan (*Audit Trail / Change Log*)**.
8. **Siaran Notifikasi Komparatif (*Schedule Diff*) di WhatsApp**.
9. **Dua Siklus Siaran Rutin Harian**: Siaran pagi (Jadwal hari ini) dan Siaran sore (Reminder tugas aktif).
10. **Pesan WhatsApp Ringkas dengan Tautan Menuju Web Dashboard**.

### Kelompok 2: Fitur Relevan untuk Pengembangan Lanjutan (Fase 2 / Secondary)
1. **Kalender Visual Interaktif** (tampilan kalender bulanan grafis di dashboard).
2. **Katalog Ketersediaan Ruangan Kosong** (berdasarkan basis data kelas internal).
3. **Repositori Tautan Materi Kuliah** (tab link Google Drive / modul per mata kuliah).
4. **Panduan Pengguna Baru (*Onboarding Banner / Empty State Guide*)**.

### Kelompok 3: Fitur Tidak Perlu / Di Luar Cakupan (Dieliminasi)
1. **Pencatatan Event Non-Akademik (HIMA / Bukber):** Mengaburkan fokus utama sistem perkuliahan.
2. **Generator Pembagian Kelompok Mahasiswa:** Di luar lingkup fungsional sistem penjadwalan.
3. **Modul Polling Internal:** Redundan karena fitur polling WhatsApp native jauh lebih praktis.
4. **Otomatisasi Pin Pesan WhatsApp:** Keterbatasan teknis API bot dan risiko pemblokiran akun WhatsApp.
5. **Integrasi Basis Data TU / Kampus Terpusat:** Di luar kendali sistem mandiri kelas.
6. **Statistik Penggunaan Mingguan:** Tidak memberikan dampak nyata bagi penyelesaian masalah perkuliahan.
7. **Interaksi Drag-and-Drop Kalender:** Terlalu kompleks untuk kebutuhan manipulasi jadwal yang tidak terlalu sering dilakukan.
