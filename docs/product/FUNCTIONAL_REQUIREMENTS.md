# Functional Requirements Bot Jadwal

## Status Dokumen

| Atribut | Nilai |
|---|---|
| Versi | 2.0.0 |
| Status | Approved |
| Pemilik | Tim Bot Jadwal |
| Terakhir diperbarui | 23 September 2026 |
| Acuan | [User Requirements](USER_REQUIREMENTS.md), [Business Rules](BUSINESS_RULES.md), [Access Control](ACCESS_CONTROL.md), [User Flows](USER_FLOWS.md), dan [PRD](../PRD.md) |
| Turunan antarmuka | [Information Architecture](INFORMATION_ARCHITECTURE.md) |
| Turunan data | [Data Model](DATA_MODEL.md) |
| Matriks | [Traceability Matrix](TRACEABILITY.md) |

Dokumen ini menerjemahkan kebutuhan pengguna dan aturan bisnis menjadi perilaku sistem yang dapat dibangun dan diuji. Detail endpoint, skema database, komponen antarmuka, dan teknologi implementasi ditentukan pada dokumen teknis setelah requirement ini disetujui.

## 1. Konvensi

- `FR-*` adalah ID permanen functional requirement.
- `Must`, `Should`, dan `Could` mengikuti prioritas pada User Requirements.
- Kata `harus` menyatakan perilaku wajib. Kata `dapat` menyatakan kemampuan yang tersedia jika izin dan prasyarat terpenuhi.
- Acceptance criteria adalah batas minimum kelulusan, bukan daftar seluruh test case.
- Sistem harus menolak permintaan yang gagal autentikasi, otorisasi, validasi, atau pemeriksaan versi tanpa menyimpan perubahan parsial.

## 2. Akses dan Identitas

### FR-ACCESS-001 Akses Portal Kelas

| Atribut | Ketentuan |
|---|---|
| Aktor | Mahasiswa |
| Prioritas | Must |
| Sumber | UR-ACCESS-001, UR-CLASS-001; BR-CLASS-002; UF-PORTAL-001 |
| Requirement | Sistem harus menyediakan portal hanya-baca melalui slug kelas. Sesuai konfigurasi kelas, sistem langsung memberikan akses melalui tautan atau meminta kode kelas. |
| Acceptance criteria | Kode yang benar membuat sesi portal database dengan `access_code_version`; rotasi kode mencabut sesi versi lama; kode salah tidak membuka data; kode aktif membuka semester terbit termasuk arsip hanya-baca; portal tidak menyediakan operasi perubahan atau data sensitif. |

### FR-ACCESS-002 Login Pengurus

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM, System Admin |
| Prioritas | Must |
| Sumber | UR-ACCESS-002; BR-ACCESS-001; UF-ACCESS-004 |
| Requirement | Sistem harus menyediakan satu halaman login menggunakan identitas akun dan kata sandi untuk seluruh pengurus. Login normal tidak boleh bergantung pada koneksi WhatsApp. |
| Acceptance criteria | Kredensial valid membuat sesi server; lima kegagalan dalam 15 menit mengunci login 15 menit; sesi PJ/KM dibatasi 2 jam idle atau 24 jam total; sesi System Admin dibatasi 30 menit idle atau 8 jam total; akun tanpa penugasan aktif tidak dapat membuka area pengelola. |

### FR-ACCESS-003 Undangan dan Aktivasi Akun

| Atribut | Ketentuan |
|---|---|
| Aktor | System Admin, KM, calon pengurus |
| Prioritas | Must |
| Sumber | UR-ACCESS-002, UR-ACCESS-003, UR-ACCESS-004; BR-ACCESS-005; UF-ACCESS-001, UF-ACCESS-002, UF-ACCESS-003 |
| Requirement | System Admin dapat mengundang KM untuk kelas tertentu. KM dapat mengundang PJ untuk mata kuliah dan semester dalam cakupannya. Penerima dapat membuat akun baru atau menambahkan penugasan ke akun yang cocok. |
| Acceptance criteria | Undangan menampilkan peran dan cakupan yang tidak dapat diubah penerima; token hanya dapat digunakan sekali; token kedaluwarsa atau dicabut ditolak; pengiriman ulang membatalkan token lama. |

### FR-ACCESS-004 Otorisasi Berbasis Cakupan

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM, System Admin |
| Prioritas | Must |
| Sumber | UR-ACCESS-003, UR-ACCESS-004; BR-ACCESS-002, BR-ACCESS-006; UF-ACCESS-004 |
| Requirement | Backend harus memeriksa sesi, role assignment aktif, kelas, jenis aksi, serta course offering untuk PJ pada setiap permintaan perubahan. Aksi dukungan System Admin harus meminta alasan. |
| Acceptance criteria | PJ hanya dapat mengubah mata kuliah penugasannya; KM hanya dapat mengubah kelas penugasannya; penolakan tidak mengungkap keberadaan data di luar cakupan; aksi dukungan memiliki audit log. |

### FR-ACCESS-005 Pemilihan dan Pergantian Konteks

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM, System Admin |
| Prioritas | Should |
| Sumber | UR-ACCESS-005; BR-ACCESS-004; UF-ACCESS-003, UF-ACCESS-004 |
| Requirement | Sistem harus memuat seluruh penugasan aktif akun dan memungkinkan pengguna berpindah konteks tanpa membuat akun baru atau login ulang. |
| Acceptance criteria | Pengguna dengan satu konteks langsung diarahkan ke dashboard; pengguna dengan beberapa konteks dapat memilih peran, kelas, semester, dan mata kuliah; setiap halaman pengelola menampilkan konteks aktif. |

### FR-ACCESS-006 Sesi dan Pemulihan Akun

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM, System Admin |
| Prioritas | Must |
| Sumber | UR-ACCESS-006; BR-ACCESS-003; UF-ACCESS-005 |
| Requirement | Pengurus dapat memulihkan akun melalui verifikasi WhatsApp, kode pemulihan, atau bantuan manual System Admin setelah verifikasi identitas. Sistem harus dapat mencabut sesi dari server. |
| Acceptance criteria | Token yang kedaluwarsa atau telah digunakan ditolak; pemulihan berhasil mencabut token lama dan sesi yang ditentukan kebijakan; pemulihan manual meminta alasan; seluruh tindakan dicatat. |

### FR-ACCESS-007 Siklus dan Pergantian Peran

| Atribut | Ketentuan |
|---|---|
| Aktor | KM, System Admin |
| Prioritas | Must |
| Sumber | UR-ACCESS-003, UR-ACCESS-004, UR-ACCESS-006; BR-ACCESS-003; UF-ACCESS-006 |
| Requirement | Pemberi akses dapat menangguhkan atau mencabut role assignment. KM mengelola PJ pada kelasnya, sedangkan System Admin mengelola KM. Undangan dan role assignment memakai lifecycle terpisah. |
| Acceptance criteria | Undangan memakai `PENDING`, `ACCEPTED`, `EXPIRED`, atau `REVOKED`; role assignment memakai `ACTIVE`, `SUSPENDED`, atau `REVOKED`; KM tetap aktif lintas semester sesuai `valid_from` dan `valid_until`; pencabutan menghentikan operasi baru tanpa mengubah riwayat. |

## 3. Kelas dan Semester

### FR-CLASS-001 Pengelolaan Kelas

| Atribut | Ketentuan |
|---|---|
| Aktor | System Admin |
| Prioritas | Must |
| Sumber | UR-CLASS-001, UR-CLASS-002; BR-CLASS-001; UF-ACCESS-001 |
| Requirement | System Admin dapat membuat kelas dengan identitas permanen yang memuat program studi, angkatan, dan rombel, serta menetapkan slug dan mode akses portal. |
| Acceptance criteria | Identitas dan slug unik; pergantian semester tidak membuat identitas kelas baru; perubahan kelas tidak memengaruhi data kelas lain. |

### FR-CLASS-002 Isolasi Data Kelas

| Atribut | Ketentuan |
|---|---|
| Aktor | Semua aktor |
| Prioritas | Must |
| Sumber | UR-CLASS-001; BR-CLASS-001 |
| Requirement | Sistem harus membatasi setiap query dan operasi data menggunakan kelas pada konteks atau akses portal yang valid. |
| Acceptance criteria | Jadwal, tugas, materi, anggota, dan audit dari kelas lain tidak muncul; permintaan dengan pengenal kelas yang tidak cocok ditolak; backup dan restore tidak mencampurkan kelas. |

### FR-SEM-001 Pembuatan Semester Draf

| Atribut | Ketentuan |
|---|---|
| Aktor | KM, System Admin |
| Prioritas | Must |
| Sumber | UR-SEM-001, UR-SEM-002; BR-SEM-001, BR-SEM-002, BR-SEM-005; UF-SEM-001, UF-SEM-002 |
| Requirement | Pengguna dapat membuat semester draf dengan tanggal mulai dan selesai melalui impor JSON, menyalin struktur semester sebelumnya, atau input manual. Salinan tidak membawa tugas, publikasi, atau penugasan PJ aktif secara otomatis. |
| Acceptance criteria | Tanggal selesai harus setelah tanggal mulai; semester draf tidak menggantikan semester aktif; pengguna dapat meninjau dan mengubah mata kuliah serta jadwal; seluruh data draf memiliki kelas dan semester yang benar. |

### FR-SEM-002 Validasi Impor JSON

| Atribut | Ketentuan |
|---|---|
| Aktor | KM, System Admin |
| Prioritas | Must |
| Sumber | UR-SEM-002, UR-SEM-003; BR-SEM-005; UF-SEM-001 |
| Requirement | Sistem harus memvalidasi struktur, field wajib, mata kuliah, dosen, waktu, ruangan, dan duplikasi sebelum menerima hasil impor. |
| Acceptance criteria | Kesalahan ditampilkan per baris dan field; pengguna dapat mengunggah perbaikan; kesalahan pemblokir mencegah aktivasi; impor gagal tidak meninggalkan data aktif atau data parsial tanpa konfirmasi. |

### FR-SEM-003 Preview dan Aktivasi Semester

| Atribut | Ketentuan |
|---|---|
| Aktor | KM, System Admin |
| Prioritas | Must |
| Sumber | UR-SEM-001, UR-SEM-003; BR-SEM-002, BR-SEM-003, BR-SEM-004; UF-SEM-001, UF-SEM-002 |
| Requirement | Sistem harus menampilkan ringkasan mata kuliah, jadwal, konflik, dan kelengkapan sebelum aktivasi. Aktivasi semester baru harus sekaligus mengarsipkan semester aktif sebelumnya. |
| Acceptance criteria | PJ tidak dapat mengaktifkan semester; satu kelas tidak pernah memiliki dua semester aktif; aktivasi mengisi `published_at`; kegagalan transaksi mempertahankan semester aktif lama; aktivasi masuk audit log. |

### FR-SEM-004 Akses Semester Arsip

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM, System Admin, mahasiswa jika dipublikasikan |
| Prioritas | Must |
| Sumber | UR-SEM-001; BR-SEM-001, BR-SEM-003 |
| Requirement | Sistem harus mempertahankan semester lama sebagai arsip hanya-baca dengan jadwal, tugas, materi, penugasan, dan riwayatnya. |
| Acceptance criteria | Operasi perubahan biasa pada semester arsip ditolak; mahasiswa dengan kode kelas aktif dapat membaca semester arsip yang berstatus terbit; data yang belum pernah dipublikasikan tetap tersembunyi; pemulihan administratif dicatat terpisah. |

## 4. Jadwal Perkuliahan

### FR-SCH-001 Tampilan Jadwal Efektif

| Atribut | Ketentuan |
|---|---|
| Aktor | Semua aktor |
| Prioritas | Must |
| Sumber | UR-SCH-001, UR-SCH-003, UR-SCH-008; BR-SCH-001; UF-PORTAL-002 |
| Requirement | Sistem harus mengembangkan `schedule_patterns` pada tanggal yang dipilih lalu menerapkan `teaching_events` terbit untuk menghasilkan jadwal efektif. |
| Acceptance criteria | Setiap entri menampilkan nama mata kuliah, jenis, jam mulai dan selesai, dosen, serta ruangan; label membedakan reguler, pengganti, tambahan, libur, dan dibatalkan tanpa mengandalkan warna saja. |

### FR-SCH-002 Pengelolaan Jadwal Reguler Manual

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM |
| Prioritas | Must |
| Sumber | UR-SCH-002; BR-ACCESS-002, BR-SEM-001; UF-SCH-001 |
| Requirement | Pengurus dapat menambah dan mengubah jadwal reguler menggunakan mata kuliah, jenis, dosen, hari, jam mulai, durasi, serta ruangan. Sistem menghitung jam selesai. |
| Acceptance criteria | PJ hanya dapat memilih mata kuliah penugasannya; field wajib divalidasi; sistem memeriksa konflik sebelum menyimpan; perubahan masuk audit log. |

### FR-SCH-003 Draf Perubahan Jadwal

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM |
| Prioritas | Must |
| Sumber | UR-SCH-003, UR-SCH-004; BR-SCH-001, BR-SCH-002; UF-SCH-002 |
| Requirement | Pengurus dapat membuat, mengubah, dan menghapus draf `teaching_event` untuk sesi pengganti, tambahan, libur, atau pembatalan akademik sebelum publikasi. |
| Acceptance criteria | Draf tidak terlihat di portal dan tidak memicu pesan; event tambahan boleh tanpa pola asal; event pengganti atau pembatalan menunjuk pola dan tanggal kejadian asal; perubahan permanen membuat versi pola baru. |

### FR-SCH-004 Preview dan Pemeriksaan Konflik

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM |
| Prioritas | Must |
| Sumber | UR-SCH-005; BR-SCH-002, BR-SCH-007; UF-SCH-002, UF-SCH-003 |
| Requirement | Sebelum publikasi, sistem harus menampilkan data lama, data baru, jenis perubahan, waktu berlaku, penerima, serta benturan kelas, waktu, atau ruangan yang diketahui. |
| Acceptance criteria | Pengguna dapat kembali memperbaiki input; konflik pemblokir mencegah publikasi; konflik nonpemblokir meminta konfirmasi dan alasan; hasil ruangan menyebut perlunya konfirmasi TU. |

### FR-SCH-005 Publikasi Perubahan Jadwal

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM |
| Prioritas | Must |
| Sumber | UR-SCH-006; BR-SCH-003, BR-SCH-004; UF-SCH-002, UF-SCH-003 |
| Requirement | PJ dapat memublikasikan teaching event pada mata kuliah penugasannya dan KM pada seluruh mata kuliah kelasnya. Publikasi mengubah lifecycle menjadi `PUBLISHED`, memperbarui portal, dan membuat notifikasi dengan identitas unik. |
| Acceptance criteria | Publikasi tidak memerlukan persetujuan KM; pelaku dan versi tersimpan; percobaan ulang dengan identitas sama tidak membuat data atau pesan ganda; kegagalan WhatsApp tidak membatalkan publikasi web. |

### FR-SCH-006 Versi Jadwal Permanen

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM |
| Prioritas | Must |
| Sumber | UR-SCH-003, UR-SCH-006; BR-SCH-001, BR-SCH-004; UF-SCH-003 |
| Requirement | Publikasi perubahan permanen harus menutup versi `schedule_pattern` lama dan mengaktifkan versi baru sejak tanggal berlaku tanpa menghapus riwayat. |
| Acceptance criteria | Tanggal berlaku harus berada pada semester aktif; sesi sebelum tanggal tersebut tetap menggunakan versi lama; konflik dengan perubahan sementara harus diselesaikan sebelum publikasi. |

### FR-SCH-007 Pencabutan Publikasi dan Koreksi

| Atribut | Ketentuan |
|---|---|
| Aktor | KM |
| Prioritas | Must |
| Sumber | UR-SCH-007; BR-SCH-005, BR-SCH-006; UF-SCH-004 |
| Requirement | KM dapat mencabut teaching event terbit pada kelasnya setelah melihat jadwal yang akan kembali berlaku dan memasukkan alasan. Pembatalan sesi akademik tetap direpresentasikan sebagai `SESSION_CANCELLED`, bukan pencabutan publikasi. |
| Acceptance criteria | Publikasi awal tetap tersimpan; lifecycle berubah menjadi `REVOKED`; pencabutan kedua ditolak; portal kembali menampilkan jadwal yang berlaku; informasi yang sudah disiarkan menghasilkan notifikasi koreksi. |

### FR-SCH-008 Perkuliahan Lintas Kelas

| Atribut | Ketentuan |
|---|---|
| Aktor | KM kelas pemilik dan KM kelas peserta |
| Prioritas | Must |
| Sumber | UR-SCH-009; BR-SCH-008 |
| Requirement | Satu teaching event dapat ditautkan ke beberapa course offering tanpa menduplikasi event. Satu kelas menjadi pemilik dan setiap kelas peserta memiliki status partisipasi sendiri. |
| Acceptance criteria | KM pemilik mengubah acara utama; partisipasi baru berstatus `PENDING`; KM peserta dapat `ACCEPTED`, `DECLINED`, atau `REMOVED`; event hanya tampil pada kelas peserta setelah diterima; KM peserta dapat melepas kelasnya tanpa mengubah acara utama. |

## 5. Tugas dan Materi

### FR-TASK-001 Pembuatan Draf Tugas

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM |
| Prioritas | Must |
| Sumber | UR-TASK-001; BR-TASK-001; UF-TASK-001 |
| Requirement | Pengurus dapat membuat draf tugas melalui formulir yang memuat mata kuliah, judul, instruksi, deadline, jenis, dan tempat atau tautan pengumpulan. |
| Acceptance criteria | Mata kuliah dibatasi menurut cakupan; draf dapat disimpan sebelum lengkap; validasi gagal mempertahankan input dan menunjukkan field bermasalah. |

### FR-TASK-002 Preview dan Publikasi Tugas

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM |
| Prioritas | Must |
| Sumber | UR-TASK-002, UR-TASK-005; BR-TASK-001, BR-TASK-003; UF-TASK-001 |
| Requirement | Sistem harus menampilkan preview dan memvalidasi data minimum sebelum tugas dapat dipublikasikan. |
| Acceptance criteria | Publikasi langsung menghasilkan status `PUBLISHED`; tugas muncul di portal dengan detail lengkap; sistem mencatat pelaku, membuat review state awal `NOT_REVIEWED`, dan menjadwalkan pengingat. |

### FR-TASK-003 Review Tugas oleh KM

| Atribut | Ketentuan |
|---|---|
| Aktor | KM, PJ |
| Prioritas | Must |
| Sumber | UR-TASK-005; BR-TASK-003; UF-TASK-002 |
| Requirement | KM harus dapat mereview tugas yang sudah dipublikasikan PJ tanpa menghambat publikasi awal. KM dapat menyetujui, meminta koreksi, atau membatalkan. |
| Acceptance criteria | Setiap keputusan menambah `task_review`; `CHANGES_REQUESTED` mengembalikan tugas ke `DRAFT`; `REVOKED` mengakhiri publikasi; keduanya menarik tugas dari portal dan mewajibkan catatan; versi yang berubah saat ditinjau harus dimuat ulang sebelum keputusan. |

### FR-TASK-004 Daftar, Pengelompokan, dan Filter Tugas

| Atribut | Ketentuan |
|---|---|
| Aktor | Mahasiswa, PJ, KM |
| Prioritas | Must untuk pengelompokan; Should untuk filter |
| Sumber | UR-TASK-002, UR-TASK-003, UR-TASK-004; BR-TASK-002; UF-PORTAL-002 |
| Requirement | Sistem harus mengelompokkan tugas aktif menjadi Hari Ini, Minggu Ini, Mendatang, dan Terlewat, serta menyediakan filter mata kuliah, status, dan rentang deadline. |
| Acceptance criteria | Urutan default memakai deadline terdekat; filter tidak mengubah konteks kelas; tugas yang melewati deadline muncul sebagai `Overdue`, bukan hilang. |

### FR-TASK-005 Perubahan Tugas Terbit

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM |
| Prioritas | Must |
| Sumber | UR-TASK-002, UR-TASK-005; BR-TASK-004; UF-TASK-003 |
| Requirement | Pengurus berwenang dapat mengubah tugas terbit. Sistem harus menyimpan versi sebelum dan sesudah serta menilai kebutuhan pembaruan notifikasi. |
| Acceptance criteria | PJ hanya mengubah mata kuliahnya; perubahan deadline atau tempat pengumpulan yang memengaruhi mahasiswa membuat pembaruan sesuai kebijakan; versi lama tetap dapat diaudit. |

### FR-TASK-006 Penyelesaian, Arsip, dan Pemulihan

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM, System Admin sesuai cakupan |
| Prioritas | Should |
| Sumber | UR-TASK-006, UR-OPS-003; BR-TASK-002, BR-OPS-002; UF-TASK-003 |
| Requirement | Pengurus dapat menandai hasil tugas, mengarsipkan melalui `archived_at`, melakukan soft delete, dan memulihkan tugas sesuai kewenangan. |
| Acceptance criteria | Status hasil `PUBLISHED`, `COMPLETED`, atau `OVERDUE` tidak ditimpa oleh pengarsipan; data tidak hilang dari audit; pemulihan tidak menghapus catatan penghapusan. |

### FR-TASK-007 Materi dan Tautan Mata Kuliah

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM, mahasiswa |
| Prioritas | Should |
| Sumber | UR-TASK-007; BR-TASK-005 |
| Requirement | Pengurus dapat menambah materi atau tautan dengan scope kelas umum atau course offering tertentu. Mahasiswa dapat membukanya dari halaman materi, mata kuliah, atau detail tugas jika akses diizinkan. |
| Acceptance criteria | `class_id` selalu wajib dan `course_offering_id` opsional; PJ hanya dapat membuat materi pada offering penugasannya; KM dapat membuat materi umum kelas; pengarsipan semester tidak menghapus materi lama. |

## 6. Notifikasi WhatsApp

### FR-NOTIF-001 Ringkasan Jadwal Harian

| Atribut | Ketentuan |
|---|---|
| Aktor | Sistem, mahasiswa sebagai penerima |
| Prioritas | Must |
| Sumber | UR-NOTIF-001; BR-NOTIF-003 |
| Requirement | Sistem harus membuat ringkasan jadwal harian pada waktu pagi yang dikonfigurasi untuk kelas. Jadwal efektif harus memperhitungkan perubahan aktif. |
| Acceptance criteria | Pesan memuat mata kuliah, jam, dosen, dan ruangan; sesi pengganti menggantikan informasi reguler yang tidak berlaku; satu kelas tidak menerima dua ringkasan untuk tanggal yang sama. |

### FR-NOTIF-002 Pengingat Tugas

| Atribut | Ketentuan |
|---|---|
| Aktor | Sistem, mahasiswa sebagai penerima |
| Prioritas | Must |
| Sumber | UR-NOTIF-002; BR-TASK-002, BR-NOTIF-003 |
| Requirement | Sistem harus membuat pengingat tugas aktif pada waktu sore yang dikonfigurasi untuk kelas berdasarkan deadline dan status terkini. |
| Acceptance criteria | Pesan membedakan tugas aktif dan terlewat; tugas selesai atau diarsipkan tidak masuk daftar aktif; pesan memuat tautan detail portal. |

### FR-NOTIF-003 Pesan Perubahan dan Koreksi Jadwal

| Atribut | Ketentuan |
|---|---|
| Aktor | Sistem, mahasiswa sebagai penerima |
| Prioritas | Must |
| Sumber | UR-NOTIF-003, UR-NOTIF-004; BR-SCH-006, BR-NOTIF-003; UF-SCH-002, UF-SCH-004 |
| Requirement | Sistem harus membuat pesan setelah perubahan jadwal dipublikasikan dan pesan koreksi setelah pembatalan atau koreksi terhadap informasi yang telah disiarkan. |
| Acceptance criteria | Pesan membandingkan jadwal lama dan baru, menyebut jenis serta waktu berlaku, dan menyediakan tautan detail; pesan koreksi terhubung ke publikasi yang dikoreksi. |

### FR-NOTIF-004 Antrean dan Idempotensi

| Atribut | Ketentuan |
|---|---|
| Aktor | Sistem, System Admin |
| Prioritas | Must |
| Sumber | UR-NOTIF-005, UR-OPS-001; BR-NOTIF-001, BR-NOTIF-002, BR-NOTIF-004; UF-OPS-001 |
| Requirement | Sistem harus menyimpan notifikasi dalam antrean dengan kunci idempotensi dan status pengiriman. Gangguan bot tidak boleh membatalkan publikasi web. |
| Acceptance criteria | Status tersedia sebagai `Pending`, `Processing`, `Sent`, atau `Failed`; percobaan ulang tidak membuat pesan bisnis baru; perubahan sebelum pengiriman mengganti atau membatalkan pesan usang; System Admin dapat melihat kegagalan dan mencoba ulang. |

### FR-NOTIF-005 Konfigurasi Waktu Pengingat

| Atribut | Ketentuan |
|---|---|
| Aktor | KM |
| Prioritas | Must |
| Sumber | UR-NOTIF-001, UR-NOTIF-002, UR-NOTIF-004 |
| Requirement | KM dapat mengatur waktu pengiriman ringkasan pagi dan pengingat sore untuk kelasnya dengan zona waktu kelas. |
| Acceptance criteria | Perubahan hanya memengaruhi kelas terkait; sistem menampilkan waktu dan zona waktu aktif; jadwal baru berlaku pada proses pengiriman berikutnya. |

### FR-NOTIF-006 Pengingat Kelas Pengganti

| Atribut | Ketentuan |
|---|---|
| Aktor | Sistem, mahasiswa sebagai penerima |
| Prioritas | Must |
| Sumber | UR-NOTIF-006; BR-NOTIF-005 |
| Requirement | Sistem harus menjadwalkan pengingat sebelum teaching event berjenis pengganti yang masih berstatus `PUBLISHED`. |
| Acceptance criteria | Waktu pengingat mengikuti konfigurasi kelas; event `REVOKED` tidak dikirim; perubahan waktu event memperbarui pesan yang masih tertunda; idempotency key mencegah pengiriman ganda. |

## 7. Ruangan

### FR-ROOM-001 Pencarian Kandidat Ruangan

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM |
| Prioritas | Should |
| Sumber | UR-ROOM-001, UR-ROOM-002; BR-ROOM-001; UF-ROOM-001 |
| Requirement | Pengurus dapat mencari kandidat ruangan berdasarkan tanggal, jam mulai, dan jam selesai menggunakan jadwal internal yang tersedia. |
| Acceptance criteria | Hasil mengecualikan benturan yang diketahui, menampilkan sumber serta waktu pembaruan data, dan menyatakan bahwa pengguna harus mengonfirmasi ke TU; hasil kosong menjelaskan konflik atau keterbatasan data. |

### FR-ROOM-002 Pencatatan Konfirmasi TU

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM |
| Prioritas | Should |
| Sumber | UR-ROOM-002; BR-ROOM-003; UF-ROOM-001 |
| Requirement | Pengurus membuat teaching event draf sebelum menghubungi TU secara manual, lalu mencatat hasil konfirmasi pada draf tersebut. |
| Acceptance criteria | Catatan menyimpan status, ruangan, nama petugas atau keterangan sumber, catatan, pelaku pencatat, dan waktu; hanya hasil `CONFIRMED` yang memenuhi syarat publikasi ketika ruangan diperlukan; catatan tidak mengubah master ruangan. |

### FR-ROOM-003 Pengelolaan Master Ruangan

| Atribut | Ketentuan |
|---|---|
| Aktor | System Admin, KM sebagai pengusul |
| Prioritas | Should |
| Sumber | UR-ROOM-003; BR-ROOM-002 |
| Requirement | System Admin dapat menambah, mengubah, dan menonaktifkan ruangan. KM dapat mengirim usulan koreksi tanpa mengubah master secara langsung. |
| Acceptance criteria | Perubahan master masuk audit log; ruangan nonaktif tidak tersedia untuk jadwal baru; referensi pada jadwal lama tetap dapat ditampilkan. |

## 8. Audit, Operasional, dan Pemulihan

### FR-AUDIT-001 Pencatatan Audit

| Atribut | Ketentuan |
|---|---|
| Aktor | Sistem |
| Prioritas | Must |
| Sumber | UR-AUDIT-001, UR-AUDIT-002; BR-AUDIT-001 |
| Requirement | Sistem harus mencatat tindakan yang ditetapkan pada BR-AUDIT-001 dengan pengguna, role assignment aktif atau snapshot konteks sistem, waktu, cakupan, objek, serta data sebelum dan sesudah jika relevan. |
| Acceptance criteria | Pengguna multi-role menghasilkan audit sesuai konteks yang dipilih; publikasi dan pencabutan memiliki catatan terpisah yang saling terhubung; perubahan peran mempertahankan identitas pemberi serta penerima; kegagalan transaksi tidak menghasilkan catatan sukses. |

### FR-AUDIT-002 Penelusuran Audit Berdasarkan Cakupan

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM, System Admin |
| Prioritas | Must |
| Sumber | UR-AUDIT-001, UR-AUDIT-002; BR-AUDIT-001 |
| Requirement | Sistem harus menyediakan daftar dan detail audit sesuai cakupan akses: PJ untuk mata kuliahnya, KM untuk kelasnya, dan System Admin untuk seluruh kelas. |
| Acceptance criteria | Pengguna dapat memfilter waktu, pelaku, dan jenis tindakan; PJ tidak melihat audit mata kuliah lain; nilai sebelum dan sesudah dapat dibandingkan bila tersedia. |

### FR-AUDIT-003 Retensi dan Integritas Riwayat

| Atribut | Ketentuan |
|---|---|
| Aktor | Sistem, System Admin |
| Prioritas | Must |
| Sumber | UR-AUDIT-003; BR-AUDIT-002 |
| Requirement | Sistem harus mempertahankan audit log dan arsip selama kelas aktif ditambah sekurangnya satu tahun akademik serta mencegah perubahan melalui operasi aplikasi biasa. |
| Acceptance criteria | Pengarsipan semester tidak menghapus audit; soft delete tidak menghapus riwayat; penghapusan setelah retensi memerlukan prosedur dan otorisasi terpisah. |

### FR-OPS-001 Pemisahan Data dan Koneksi Bot

| Atribut | Ketentuan |
|---|---|
| Aktor | System Admin |
| Prioritas | Must |
| Sumber | UR-OPS-001; BR-OPS-004; UF-OPS-001 |
| Requirement | Sistem harus menyimpan data akademik secara terpisah dari sesi WhatsApp sehingga nomor, sesi, atau koneksi bot dapat diganti tanpa mengubah data kelas. |
| Acceptance criteria | Putusnya bot tidak menghalangi login atau publikasi web; penyambungan sesi baru menggunakan kembali data kelas; hanya status pengiriman pesan yang terpengaruh. |

### FR-OPS-002 Backup dan Restore Terbatas Cakupan

| Atribut | Ketentuan |
|---|---|
| Aktor | System Admin, KM sebagai pemohon |
| Prioritas | Must |
| Sumber | UR-OPS-002; BR-OPS-003; UF-OPS-003 |
| Requirement | System Admin dapat membuat backup dan menjalankan restore untuk kelas serta semester yang dipilih setelah meninjau cakupan dan memasukkan alasan. |
| Acceptance criteria | Paket yang tidak cocok ditolak; sistem membuat titik pemulihan sebelum restore; relasi dan kelas tujuan diverifikasi; kegagalan mengembalikan keadaan awal; hasil masuk audit log. |

### FR-OPS-003 Soft Delete dan Pemulihan Data

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM, System Admin sesuai cakupan |
| Prioritas | Must |
| Sumber | UR-OPS-003; BR-OPS-002; UF-TASK-003 |
| Requirement | Sistem harus menggunakan soft delete untuk data penting dan menyediakan pemulihan sesuai kewenangan selama masa pemulihan yang dikonfigurasi. |
| Acceptance criteria | Tindakan meminta konfirmasi; data terhapus tidak muncul pada daftar aktif; pemulihan mengembalikan data tanpa menghapus audit penghapusan; penghapusan permanen tidak tersedia melalui alur biasa. |

### FR-OPS-004 Deteksi Konflik Edit

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM |
| Prioritas | Should |
| Sumber | UR-OPS-004; BR-OPS-001; UF-OPS-002 |
| Requirement | Sistem harus membandingkan versi data saat pengguna menyimpan perubahan terhadap versi yang terakhir dibaca. |
| Acceptance criteria | Simpan dari versi lama ditolak; sistem menampilkan versi terkini dan perubahan pengguna; input pengguna tidak hilang; sistem tidak menggabungkan perubahan yang konflik secara diam-diam. |

## 9. Pengalaman Pengguna dan Aksesibilitas

### FR-UX-001 Form Mobile untuk Operasi Umum

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM |
| Prioritas | Must |
| Sumber | UR-UX-001; UF-TASK-001, UF-SCH-002 |
| Requirement | Form tugas dan jadwal harus dapat diselesaikan pada layar ponsel tanpa command, hafalan sintaks, atau scroll horizontal. |
| Acceptance criteria | Target sentuh kontrol utama minimal 44 x 44 piksel; input memiliki label; nilai pilihan dibatasi menurut konteks; pengguna dapat menyelesaikan alur normal dengan satu tangan. Target waktu kurang dari 30 detik diuji melalui usability test. |

### FR-UX-002 Status Halaman dan Umpan Balik

| Atribut | Ketentuan |
|---|---|
| Aktor | Semua aktor |
| Prioritas | Must |
| Sumber | UR-UX-002 |
| Requirement | Setiap halaman yang memuat atau mengubah data harus menyediakan keadaan loading, empty, error, success, dan permission denied yang sesuai. |
| Acceptance criteria | Pesan menyebut tindakan yang gagal dan langkah berikutnya; kegagalan tidak menghapus input tanpa peringatan; tombol tidak menampilkan status sukses sebelum penyimpanan berhasil. |

### FR-UX-003 Panduan Awal Pengurus

| Atribut | Ketentuan |
|---|---|
| Aktor | PJ, KM |
| Prioritas | Should |
| Sumber | UR-UX-003 |
| Requirement | Pada akses pertama, sistem harus menjelaskan peran, cakupan, konteks aktif, serta tindakan utama dan menyediakan cara membuka panduan kembali. |
| Acceptance criteria | PJ melihat mata kuliah yang dapat dikelola; KM melihat cakupan kelas; pengguna dapat melewati dan membuka ulang panduan; panduan tidak mewajibkan command WhatsApp. |

### FR-UX-004 Aksesibilitas Interaksi

| Atribut | Ketentuan |
|---|---|
| Aktor | Semua aktor |
| Prioritas | Must |
| Sumber | UR-UX-004 |
| Requirement | Antarmuka harus dapat digunakan dengan keyboard dan pembaca layar dasar serta tidak menyampaikan status hanya melalui warna. |
| Acceptance criteria | Urutan fokus mengikuti tampilan; fokus terlihat; kontrol memiliki nama yang dapat dibaca teknologi bantu; dialog dapat ditutup dengan Escape; status memakai teks atau ikon selain warna. |

## 10. Aturan Lintas Fitur

Aturan berikut berlaku pada seluruh functional requirement:

1. Backend memeriksa autentikasi dan cakupan pada setiap operasi perubahan.
2. Operasi yang mengubah beberapa data terkait harus bersifat atomik atau dapat dikembalikan ke keadaan sebelum proses.
3. Waktu ditampilkan dalam zona waktu kelas dan disimpan dengan informasi zona waktu yang tidak ambigu.
4. Operasi publikasi, pencabutan publikasi, pembatalan sesi akademik, penghapusan, pemulihan, dan perubahan peran harus menghasilkan audit log.
5. Sistem tidak boleh menampilkan data dari kelas atau semester lain akibat manipulasi URL, payload, atau konteks antarmuka.
6. Non-functional requirement untuk keamanan, kinerja, keandalan, dan deployment tetap mengikuti [PRD bagian 9](../PRD.md#9-kebutuhan-non-fungsional).

## 11. Ringkasan Ketertelusuran

| Kelompok FR | User Requirement | Business Rule | User Flow |
|---|---|---|---|
| `FR-ACCESS` | UR-ACCESS-001 sampai UR-ACCESS-006 | BR-ACCESS-001 sampai BR-ACCESS-006, BR-CLASS-002 | UF-ACCESS-001 sampai UF-ACCESS-006, UF-PORTAL-001 |
| `FR-CLASS`, `FR-SEM` | UR-CLASS-001 sampai UR-SEM-003 | BR-CLASS-001 sampai BR-SEM-005 | UF-ACCESS-001, UF-SEM-001 sampai UF-SEM-002 |
| `FR-SCH` | UR-SCH-001 sampai UR-SCH-009 | BR-SCH-001 sampai BR-SCH-008 | UF-PORTAL-002, UF-SCH-001 sampai UF-SCH-005 |
| `FR-TASK` | UR-TASK-001 sampai UR-TASK-007 | BR-TASK-001 sampai BR-TASK-005, BR-OPS-002 | UF-PORTAL-002, UF-TASK-001 sampai UF-TASK-003 |
| `FR-NOTIF` | UR-NOTIF-001 sampai UR-NOTIF-006 | BR-NOTIF-001 sampai BR-NOTIF-005 | UF-SCH-002, UF-SCH-004, UF-OPS-001 |
| `FR-ROOM` | UR-ROOM-001 sampai UR-ROOM-003 | BR-ROOM-001 sampai BR-ROOM-003 | UF-ROOM-001 |
| `FR-AUDIT`, `FR-OPS` | UR-AUDIT-001 sampai UR-OPS-004 | BR-AUDIT-001 sampai BR-OPS-004 | UF-OPS-001 sampai UF-OPS-003 |
| `FR-UX` | UR-UX-001 sampai UR-UX-004 | Aturan lintas fitur dan PRD bagian 8 | Flow terkait pada setiap operasi |

## 12. Kriteria Siap Implementasi

Functional requirement siap masuk desain dan engineering jika:

1. User requirement, business rule, dan user flow sumbernya dapat dilacak.
2. Aktor, izin, prasyarat, input, hasil, dan kondisi penolakan sudah jelas.
3. Acceptance criteria dapat diubah menjadi test case tanpa menebak keputusan produk.
4. Nilai konfigurasi yang belum tetap memiliki pemilik keputusan.
5. Perubahan status data dan kebutuhan audit telah ditentukan.

Perubahan requirement tidak menghapus ID lama. Requirement yang tidak lagi berlaku diberi status `Deprecated` dan menunjuk ID penggantinya.

Matriks per requirement tersedia pada [Traceability Matrix](TRACEABILITY.md).

## 13. Changelog

### 2.0.0, 23 September 2026

- Menetapkan dokumen ini sebagai satu-satunya sumber ID `FR-*`.
- Menambah perkuliahan lintas kelas dan pengingat kelas pengganti.
- Menyelaraskan lifecycle jadwal, review tugas, sesi portal, semester arsip, konfirmasi TU, materi umum, dan konteks audit.
