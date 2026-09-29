# Dokumentasi Hasil Riset Pengguna: Bot Jadwal v2.0 & Web Dashboard Admin

Dokumen ini merupakan ringkasan eksekutif, indeks data, dan sintesis temuan dari 11 sesi wawancara pengguna (*user interviews*) yang dilakukan terhadap Pengurus Kelas (Ketua Murid / KM) dan Penanggung Jawab Mata Kuliah (PJ Matkul) untuk pengembangan sistem **Bot Jadwal v2.0** dan **Web Admin Dashboard**.

---

## 1. Daftar dan Indeks Berkas Responden

Berikut adalah 11 dokumen transkripsi wawancara lengkap:

| No | Nama Responden | Peran / Jabatan | Label / ID | Pewawancara | Berkas Dokumentasi |
|---|---|---|---|---|---|
| 01 | **Iman** | PJ SDB (Sistem Basis Data) Teori | [19] | jocelyn | [01_iman_pj_matkul_sdb_teori.md](01_iman_pj_matkul_sdb_teori.md) |
| 02 | **Arsel** | PJ OS (Sistem Operasi) Praktek | [19] | jocelyn | [02_arsel_pj_matkul_os_praktek.md](02_arsel_pj_matkul_os_praktek.md) |
| 03 | **Irfan** | PJ OS (Sistem Operasi) Teori | [19] | jocelyn | [03_irfan_pj_matkul_os_teori.md](03_irfan_pj_matkul_os_teori.md) |
| 04 | **Afzhal** | PJ Matdis (Matematika Diskrit) Praktek | [19] | Faqih23 | [04_afzhal_pj_matkul_matdis_praktek.md](04_afzhal_pj_matkul_matdis_praktek.md) |
| 05 | **Faqih** | PJ SDB (Sistem Basis Data) Praktek | [18] | Faqih23 | [05_faqih_pj_matkul_sdb_praktek.md](05_faqih_pj_matkul_sdb_praktek.md) |
| 06 | **Imam** | Ketua Murid / KM (Ketua Kelas) | [19] | jocelyn | [06_iman_km.md](06_iman_km.md) |
| 07 | **Hana** | PJ Alin (Aljabar Linier) Teori | [19] | jocelyn | [07_hana_pj_matkul_alin_teori.md](07_hana_pj_matkul_alin_teori.md) |
| 08 | **Anindya** | PJ Alin (Aljabar Linier) Praktik | [19] | jocelyn | [08_anindya_pj_matkul_alin_praktik.md](08_anindya_pj_matkul_alin_praktik.md) |
| 09 | **Giza** | PJ Alin (Aljabar Linier) Teori | [19] | jocelyn | [09_giza_pj_matkul_alin_teori.md](09_giza_pj_matkul_alin_teori.md) |
| 10 | **Kemal** | PJ PLP Teori | [20] | jocelyn | [10_kemal_pj_matkul_plp_teori.md](10_kemal_pj_matkul_plp_teori.md) |
| 11 | **Bima** | PJ PLP Praktik | [21] | jocelyn | [11_bima_pj_matkul_plp_praktik.md](11_bima_pj_matkul_plp_praktik.md) |

---

## 2. Sintesis Masalah Utama (Pain Points)

Berdasarkan analisis silang dari seluruh responden, terdapat lima hambatan operasional utama yang dihadapi pengurus kelas dan mahasiswa saat ini:

### 1. Perintah Teks Bot di WhatsApp Sulit Dihafal dan Memenuhi Grup (*Chat Clutter & CLI Friction*)
* **Masalah:** Menginput data jadwal dan tugas menggunakan perintah teks (*command*) bot langsung di dalam grup obrolan WhatsApp dinilai menyulitkan (Giza, Hana, Kemal, Faqih). Format teks yang kaku sering memicu salah ketik (*human error*).
* **Dampak:** Riwayat percakapan grup WhatsApp menjadi penuh dengan perintah bot (Imam KM, Bima), sehingga pengumuman penting mudah tertimbun dan terlewat oleh mahasiswa.

### 2. Kesulitan Mencari Ruangan Kosong untuk Kelas Pengganti
* **Masalah:** Hampir seluruh responden (Imam KM, Kemal, Faqih, Anindya, Arsel, Irfan) menyebutkan bahwa mencari ruangan kosong ketika ada kuliah pengganti adalah proses paling merepotkan dan memakan waktu.
* **Dampak:** PJ dan KM harus mencocokkan jadwal mahasiswa, kesediaan dosen, mengecek ruangan fisik kampus satu per satu, serta berkoordinasi secara manual dengan Tata Usaha (TU).

### 3. Keterbatasan Bot Lama dalam Menangani Perubahan Jadwal
* **Masalah:** Bot WhatsApp generasi sebelumnya hanya membaca jadwal perkuliahan reguler yang bersifat statis (Anindya, Kemal).
* **Dampak:** Kuliah pengganti atau penyesuaian jam kuliah dadakan tidak muncul di pesan bot, sehingga data bot menjadi tidak akurat saat terjadi perubahan jadwal.

### 4. Beban Mengingatkan Mahasiswa Berulang Kali (*Reminder Fatigue*)
* **Masalah:** PJ menghabiskan banyak waktu dan tenaga untuk mengingatkan tugas kuliah berkali-kali kepada mahasiswa sekelas (Hana, Bima, Afzhal).
* **Dampak:** Mahasiswa tetap sering lupa atau melewatkan deadline karena informasi tugas tercecer di grup WA, Microsoft Teams, dan materi slide dosen yang menumpuk.

### 5. Pembagian Tanggung Jawab Belum Terstruktur (*Role Ambiguity*)
* **Masalah:** Sistem belum memiliki delegasi tugas yang terstruktur (Imam KM, Bima). Siapa pun yang memiliki akses dapat mengutak-atik data tanpa batasan mata kuliah, atau sebaliknya, tidak ada yang merasa bertanggung jawab melakukan update berkala.

---

## 3. Matriks Kebutuhan dan Fitur Solutif yang Diusulkan Responden

Berikut adalah rangkuman fitur yang diajukan langsung oleh para responden beserta pengusulnya:

| Kebutuhan / Fitur Solutif | Deskripsi Implementasi | Diusulkan Oleh |
|---|---|---|
| **Pembatasan Hak Akses Berbasis Peran (RBAC)** | PJ Matkul hanya berhak mengubah/menambah jadwal dan tugas untuk mata kuliah miliknya sendiri. KM memiliki hak akses penuh ke seluruh mata kuliah kelas. | Bima, Imam KM |
| **Pembedaan Jadwal Sementara vs Permanen (*Auto-Revert*)** | Jadwal kelas pengganti yang bersifat sementara memiliki masa berlaku dan otomatis kembali (*auto-revert*) ke jadwal reguler semula setelah kelas selesai. | Imam KM |
| **Format Siaran Komparatif (*Schedule Diff Broadcast*)** | Notifikasi perubahan jadwal di WhatsApp menyajikan perbandingan data lama vs data baru berdampingan (`Sebelum: [Jadwal Lama] -> Menjadi: [Jadwal Baru]`). | Kemal |
| **Mode Draf (*Draft State / Staging*)** | PJ dapat menyimpan rencana perubahan jadwal sebagai draf sambil menunggu kepastian dosen, lalu menekan tombol *publish* setelah disetujui. | Giza |
| **Pencarian dan Rekomendasi Ruangan Kosong** | Sistem memindai jadwal pemakaian seluruh kelas untuk merekomendasikan ruangan yang kosong pada slot hari dan jam yang dipilih. | Kemal, Faqih, Anindya, Arsel |
| **Format Pesan Ringkas dengan Tautan Web (*Deep-link*)** | Pesan WhatsApp dibuat ringkas dan padat. Detail instruksi dan tautan berkas diarahkan ke web dashboard agar tidak memicu *chat clutter*. | Faqih, Hana, Bima |
| **Pengelompokan Deadline (*MS Teams Style*)** | Daftar tugas dikelompokkan berdasarkan rentang waktu: Hari Ini, Minggu Ini, Mendatang, dan Terlewat. | Bima, Faqih, Afzhal |
| **Formulir Terpandu dengan Dropdown dan Kalender** | Input data menggunakan formulir dengan pilihan *dropdown* (kelas, ruangan, jam) serta kalender visual yang bisa diklik per tanggal. | Hana, Giza, Kemal, Anindya |
| **Waktu Pengingat Sore Hari (*Post-Class Timing*)** | Notifikasi pengingat tugas harian otomatis dijadwalkan setiap sore setelah jam kuliah selesai (pulang kuliah). | Bima |
| **Repositori Berkas dan Tautan Materi Kuliah** | Tab khusus di web dashboard untuk mengumpulkan tautan Google Drive, slide presentasi dosen, dan modul per mata kuliah. | Afzhal, Anindya |
| **Alur Verifikasi oleh KM (*Approval Flow*)** | Input tugas baru oleh PJ dapat diverifikasi atau di-ACC oleh KM terlebih dahulu sebelum disiarkan secara massal ke grup kelas. | Faqih |
| **Panduan Awal Pengguna (*Onboarding Tutorial*)** | Tampilan petunjuk atau panduan cara kerja dashboard saat pertama kali dibuka oleh admin baru. | Imam KM |

---

## 4. Implikasi Desain untuk Arsitektur Bot Jadwal v2.0

### A. Pembagian Peran Komponen Sistem
1. **Web Admin Dashboard (Desktop & Mobile Web):**
   * Berfungsi sebagai pusat operasional bagi KM dan PJ Matkul.
   * Seluruh aktivitas input data, manajemen jadwal kuliah, pembuatan tugas, dan pencarian ruangan dilakukan di antarmuka web yang rapi.
   * Menghilangkan ketergantungan pada pengetikan perintah teks (*command*) manual di chat WhatsApp.
2. **WhatsApp Bot Service (Broadcaster & Notification Engine):**
   * Berfungsi sebagai mesin notifikasi otomatis satu arah ke grup kelas atau saluran pengumuman.
   * Mengirimkan jadwal kuliah harian setiap pagi.
   * Mengirimkan pengingat tugas aktif setiap sore hari (pulang kuliah).
   * Mengirimkan pesan siaran instan dengan format komparatif (*diff*) setiap kali terjadi perubahan jadwal atau penambahan tugas baru dari web dashboard.

### B. Entitas Basis Data yang Dibutuhkan (SQLite)
* `users` (id, name, role: `km` / `pj`, subject_id, password/token).
* `subjects` (id, code, name, lecturer, default_day, default_start_time, default_duration, default_room).
* `schedules` (id, subject_id, type: `regular` / `replacement`, date, start_time, end_time, room, is_temporary, status: `draft` / `published`).
* `tasks` (id, subject_id, title, description, submission_target, deadline, status: `active` / `completed`).
* `rooms` (id, name, capacity, building).
* `change_logs` (id, entity_type, entity_id, user_id, before_state, after_state, timestamp).
