# Product Requirements Document Bot Jadwal v2.1

## Status Dokumen

| Atribut | Nilai |
|---|---|
| Versi | 2.1.0 |
| Status | Approved |
| Tanggal | 23 September 2026 |
| Pemilik | Tim Bot Jadwal |
| Target awal | Semester Ganjil 2026/2027 |
| Acuan produk | [Product Definition v0.3](product/PRODUCT_DEFINITION.md) |
| Acuan kebutuhan | [User Requirements v1.0](product/USER_REQUIREMENTS.md) |
| Sumber riset | [Wawancara 11 pengguna](user-interviews/README.md) |

PRD ini menetapkan hasil yang harus disediakan produk, batas ruang lingkup, prioritas, serta kriteria keberhasilan. Product Definition menjadi sumber keputusan alur dan aturan. User Requirements menjadi sumber kebutuhan pengguna. Detail tabel database, endpoint, dan keputusan implementasi dibahas dalam dokumen teknis terpisah.

## 1. Ringkasan Produk

Bot Jadwal membantu mahasiswa menerima jadwal perkuliahan, perubahan jadwal, tugas, tautan, dan pengingat melalui WhatsApp. Versi saat ini masih banyak bergantung pada command. Hasil wawancara menunjukkan bahwa keluaran bot umumnya mudah dibaca, tetapi pengurus kesulitan menghafal format input, informasi sering terlambat diperbarui, dan aktivitas bot menambah kepadatan percakapan grup.

Bot Jadwal v2.1 membagi tanggung jawab sistem sebagai berikut:

- Portal web menjadi tempat mahasiswa membaca informasi kelas.
- Dashboard pengelola menjadi tempat PJ dan KM mengelola data.
- Backend Go dan SQLite menjadi sumber data utama.
- Bot WhatsApp menjadi kanal siaran, pengingat, koreksi, dan jalur darurat.
- System Admin mengelola konfigurasi lintas kelas dan pemulihan akses.

## 2. Masalah Pengguna

1. PJ harus menghafal command panjang untuk memperbarui tugas dan jadwal.
2. Informasi akademik tersebar di WhatsApp, Teams, Classroom, email, dan slide dosen.
3. Pesan penting mudah tenggelam di grup kelas yang ramai.
4. Bot lama belum menyajikan siklus jadwal pengganti secara lengkap.
5. Mencari ruangan kosong membutuhkan pencocokan manual dan konfirmasi TU.
6. PJ harus mengingatkan mahasiswa berulang kali.
7. Kewenangan PJ, KM, dan pengelola sistem belum dibatasi dengan jelas.
8. Identitas kelas, semester, dan arsip semester lama belum dimodelkan secara konsisten.

## 3. Persona dan Peran

### 3.1 Mahasiswa

Mahasiswa membutuhkan akses cepat tanpa akun untuk melihat jadwal, tugas, perubahan, ruangan, dan materi kelas. Mahasiswa memiliki akses hanya-baca melalui tautan atau kode kelas.

### 3.2 PJ Mata Kuliah

PJ mengelola tugas, tautan, dan jadwal untuk mata kuliah yang ditugaskan. PJ dapat menyimpan draf dan langsung memublikasikan perubahan dalam cakupannya. Setiap publikasi dicatat.

### 3.3 Ketua Murid

KM mengelola semua mata kuliah pada kelas yang ditugaskan, menunjuk PJ, mengaktifkan semester, memantau perubahan, dan membatalkan publikasi yang keliru. Pembatalan tidak menghapus riwayat.

### 3.4 System Admin

System Admin membuat kelas, menunjuk KM awal, mengelola master ruangan, menangani pemulihan akses, dan memantau sistem lintas kelas. Peran ini tidak digunakan untuk pekerjaan harian kelas kecuali dukungan atau pemulihan.

## 4. Tujuan Produk

1. Memindahkan aktivitas administrasi rutin dari command WhatsApp ke dashboard web.
2. Memungkinkan PJ mencatat tugas dari ponsel dalam kurang dari 30 detik.
3. Menyediakan satu sumber informasi jadwal, tugas, perubahan, ruangan, dan materi.
4. Memisahkan data berdasarkan kelas dan semester.
5. Mendukung jadwal reguler, pengganti, tambahan, libur, dan pembatalan.
6. Menerapkan hak akses berdasarkan peran, kelas, semester, dan mata kuliah.
7. Mengirimkan siaran WhatsApp yang ringkas, tepat waktu, dan tidak ganda.
8. Menjaga arsip dan audit log agar perubahan dapat ditelusuri dan dipulihkan.

## 5. Batasan Ruang Lingkup

Produk ini tidak:

- menggantikan LMS atau mengelola nilai;
- menerima unggahan berkas jawaban mahasiswa;
- mengelola kegiatan non-akademik seperti HIMA atau acara kelas;
- membuat pembagian kelompok otomatis;
- membuat polling yang sudah tersedia di WhatsApp;
- menjamin ketersediaan ruangan tanpa konfirmasi TU;
- terintegrasi dengan sistem pusat kampus tanpa akses data resmi;
- menyediakan akun individual mahasiswa pada fase awal;
- menyediakan drag and drop kalender atau auto-pin WhatsApp pada MVP.

## 6. Ruang Lingkup Fungsional

### Epic 1 Akses dan Identitas

- **FR-ACCESS-001:** Mahasiswa dapat membuka portal kelas melalui tautan atau kode kelas tanpa akun.
- **FR-ACCESS-002:** Portal mahasiswa bersifat hanya-baca dan tidak menampilkan fungsi administrasi, audit log, nomor pengurus, atau data sensitif.
- **FR-ACCESS-003:** PJ dan KM masuk melalui akun yang dibuat dari undangan.
- **FR-ACCESS-004:** WhatsApp dapat digunakan untuk verifikasi atau pemulihan, tetapi bukan satu-satunya cara masuk.
- **FR-ACCESS-005:** Hak akses pengguna ditetapkan per kelas, semester, mata kuliah, dan peran.
- **FR-ACCESS-006:** Satu akun dapat memiliki peran berbeda pada beberapa kelas atau semester.
- **FR-ACCESS-007:** Pencabutan peran langsung menghentikan akses perubahan data.
- **FR-ACCESS-008:** System Admin dapat memulihkan atau memindahkan akses tanpa mengubah data kelas.

### Epic 2 Kelas dan Semester

- **FR-CLASS-001:** Kelas memiliki identitas permanen berdasarkan program studi, angkatan, dan rombel, bukan nomor semester.
- **FR-CLASS-002:** Semua data operasional terikat pada kelas agar tidak tercampur dengan kelas lain.
- **FR-SEM-001:** Setiap kelas hanya memiliki satu semester aktif.
- **FR-SEM-002:** Aktivasi semester baru mengarsipkan semester sebelumnya tanpa menghapus data.
- **FR-SEM-003:** KM atau System Admin dapat membuat semester melalui impor JSON, salin semester lama, atau input manual.
- **FR-SEM-004:** Impor menampilkan preview dan kesalahan sebelum data diaktifkan.
- **FR-SEM-005:** Kegagalan impor tidak meninggalkan data parsial tanpa konfirmasi.
- **FR-SEM-006:** Arsip semester lama tersedia dalam mode hanya-baca.

### Epic 3 Jadwal Perkuliahan

- **FR-SCH-001:** Jadwal reguler memuat mata kuliah, jenis teori atau praktik, dosen, hari, jam mulai, durasi, jam selesai, dan ruangan.
- **FR-SCH-002:** Pengelola dapat membuat dan memperbarui jadwal secara manual melalui form terpandu.
- **FR-SCH-003:** Jam selesai dihitung dari jam mulai dan durasi, lalu dapat diperiksa sebelum disimpan.
- **FR-SCH-004:** Sistem membedakan jadwal reguler, pengganti, tambahan, libur, dan dibatalkan.
- **FR-SCH-005:** Perubahan dapat ditandai sementara atau permanen.
- **FR-SCH-006:** Jadwal sementara berlaku pada sesi tertentu dan tidak menimpa jadwal reguler secara permanen.
- **FR-SCH-007:** PJ dapat menyimpan perubahan sebagai draf yang tidak terlihat oleh mahasiswa dan tidak memicu notifikasi.
- **FR-SCH-008:** Preview menampilkan jadwal lama, jadwal baru, jenis perubahan, dan penerima sebelum publikasi.
- **FR-SCH-009:** PJ dapat memublikasikan perubahan untuk mata kuliah yang ditugaskan tanpa persetujuan KM.
- **FR-SCH-010:** KM dapat memublikasikan dan membatalkan perubahan untuk semua mata kuliah pada kelasnya.
- **FR-SCH-011:** Pembatalan mempertahankan riwayat dan mengirim koreksi apabila perubahan sebelumnya telah disiarkan.
- **FR-SCH-012:** Publikasi ulang dengan identitas yang sama tidak mengirim notifikasi ganda.

### Epic 4 Tugas dan Materi

- **FR-TASK-001:** Form tugas memuat mata kuliah, judul, instruksi, deadline, serta tempat atau tautan pengumpulan.
- **FR-TASK-002:** PJ hanya dapat memilih mata kuliah dalam cakupan penugasannya.
- **FR-TASK-003:** PJ dapat menyimpan draf atau langsung memublikasikan tugas pada mata kuliahnya.
- **FR-TASK-004:** KM dapat mengaktifkan kebijakan approval tugas untuk kelasnya jika diperlukan.
- **FR-TASK-005:** Daftar tugas dikelompokkan menjadi Hari Ini, Minggu Ini, Mendatang, dan Terlewat.
- **FR-TASK-006:** Pengguna dapat menyaring tugas berdasarkan mata kuliah, status, rentang deadline, dan jenis tugas.
- **FR-TASK-007:** Tugas selesai dan terlewat berpindah ke arsip tanpa menghapus riwayat.
- **FR-TASK-008:** Materi dan tautan dapat dikelompokkan berdasarkan mata kuliah serta dihubungkan ke detail tugas.
- **FR-TASK-009:** Mahasiswa belum dapat menandai penyelesaian tugas secara pribadi sampai akun mahasiswa tersedia.

### Epic 5 Notifikasi WhatsApp

- **FR-NOTIF-001:** Sistem mengirim jadwal hari itu pada pagi hari sesuai waktu yang dikonfigurasi.
- **FR-NOTIF-002:** Jadwal pengganti yang berlaku ikut ditampilkan dalam siaran pagi.
- **FR-NOTIF-003:** Sistem mengirim pengingat tugas pada sore hari sesuai waktu yang dikonfigurasi.
- **FR-NOTIF-004:** Publikasi perubahan jadwal mengirim perbandingan data sebelum dan sesudah.
- **FR-NOTIF-005:** Jadwal pengganti mendapatkan pengingat sebelum pelaksanaan.
- **FR-NOTIF-006:** Pesan hanya memuat informasi utama dan tautan menuju detail web.
- **FR-NOTIF-007:** Ketika bot offline, publikasi tetap tersimpan dan notifikasi menunggu dalam antrean.
- **FR-NOTIF-008:** Pengiriman antrean menggunakan identitas unik agar pesan tidak terkirim dua kali.
- **FR-NOTIF-009:** Pembatalan perubahan yang sudah disiarkan mengirim pesan koreksi.

### Epic 6 Ruangan

- **FR-ROOM-001:** System Admin mengelola master ruangan lintas kelas dengan masukan dari KM.
- **FR-ROOM-002:** PJ dan KM dapat mencari kandidat ruangan berdasarkan tanggal dan rentang waktu.
- **FR-ROOM-003:** Rekomendasi menggunakan jadwal yang tersedia di dalam sistem.
- **FR-ROOM-004:** Hasil selalu diberi label sebagai rekomendasi internal yang perlu dikonfirmasi ke TU.
- **FR-ROOM-005:** Pengguna dapat mencatat hasil konfirmasi ruangan.

### Epic 7 Audit dan Pemulihan

- **FR-AUDIT-001:** Tambah, ubah, hapus, publikasi, pembatalan, pemulihan, dan perubahan peran dicatat.
- **FR-AUDIT-002:** Audit log memuat pelaku, peran, waktu, kelas, entitas, tindakan, data sebelum, dan data sesudah.
- **FR-AUDIT-003:** PJ melihat log dalam cakupannya, KM melihat log kelasnya, dan System Admin melihat lintas kelas.
- **FR-AUDIT-004:** Audit log dan arsip disimpan selama kelas aktif ditambah sekurangnya satu tahun akademik.
- **FR-RECOVERY-001:** Data penting menggunakan penghapusan yang dapat dipulihkan sesuai kewenangan.
- **FR-RECOVERY-002:** Sistem mendeteksi ketika data telah berubah sejak pengguna membukanya dan mencegah penimpaan tanpa peringatan.
- **FR-RECOVERY-003:** Data dapat dicadangkan dan dipulihkan per kelas serta semester.
- **FR-RECOVERY-004:** Pergantian nomor atau sesi WhatsApp tidak menghapus data akademik.

## 7. Struktur Informasi

```text
Portal Kelas
├── Ringkasan
├── Jadwal
├── Tugas
├── Materi dan Tautan
└── Perubahan Terbaru

Area Pengelola
├── Kelola Jadwal
├── Kelola Tugas
├── Draf dan Persetujuan
├── Ruangan
├── Anggota dan Peran
└── Riwayat Perubahan

System Admin
├── Daftar Kelas
├── Semester
├── Pengguna
├── Master Mata Kuliah
├── Master Ruangan
└── Status Sistem
```

Detail visual, komponen, responsive behavior, dan state antarmuka mengikuti `DESIGN.md` serta spesifikasi UI yang akan diselaraskan terpisah.

## 8. Kebutuhan Pengalaman Pengguna

- Antarmuka memprioritaskan layar ponsel dengan target sentuh minimal 44 x 44 piksel.
- Form tidak memerlukan scroll horizontal pada lebar layar 390 piksel.
- Penambahan tugas normal dapat diselesaikan dalam kurang dari 30 detik.
- Setiap halaman menyediakan keadaan loading, kosong, gagal, berhasil, dan akses ditolak.
- Kegagalan penyimpanan tidak menghilangkan input tanpa peringatan.
- Status tidak disampaikan melalui warna saja.
- Fokus keyboard terlihat dan semua kontrol penting memiliki label.
- Pengurus baru mendapatkan panduan awal yang dapat dibuka kembali.
- Jadwal menggunakan nama mata kuliah dan bahasa manusiawi, bukan hanya kode internal.

## 9. Kebutuhan Non-Fungsional

### 9.1 Kinerja

- Target muat awal portal di bawah 1,5 detik pada jaringan seluler 4G dalam kondisi pengujian yang ditetapkan.
- Target respons penyimpanan operasi umum di bawah 300 milidetik, tidak termasuk waktu pengiriman WhatsApp.
- Kegagalan memenuhi target dicatat melalui telemetri agar dapat dianalisis.

### 9.2 Keamanan

- Semua mutasi data memerlukan autentikasi dan pemeriksaan izin di backend.
- Kata sandi tidak disimpan dalam bentuk asli.
- Sesi dapat dicabut ketika akun, perangkat, atau peran tidak lagi berlaku.
- Undangan dan kode pemulihan memiliki masa berlaku serta hanya dapat digunakan sesuai kebijakan.
- Data sensitif tidak dikirim ke portal mahasiswa hanya-baca.
- Endpoint mutasi dilindungi dari pemalsuan permintaan dan penyalahgunaan berulang.

### 9.3 Keandalan

- Operasi database mendukung akses web dan bot secara bersamaan tanpa kehilangan data.
- Publikasi web tidak bergantung pada koneksi aktif WhatsApp.
- Pengiriman notifikasi dapat dicoba ulang tanpa menghasilkan duplikasi.
- Aktivasi semester, impor jadwal, dan pemulihan data tidak boleh meninggalkan kondisi setengah selesai.

### 9.4 Deployability

- Aplikasi tetap menggunakan backend Go, SQLite, dan aset web tersemat dalam satu distribusi aplikasi.
- Frontend tidak memerlukan npm atau proses build terpisah.
- Data akademik, audit, dan sesi WhatsApp tetap dipisahkan agar pemulihan sesi tidak memengaruhi data kelas.

## 10. Model Domain Konseptual

```mermaid
erDiagram
    USER ||--o{ ROLE_ASSIGNMENT : receives
    CLASS ||--o{ CLASS_SEMESTER : has
    CLASS_SEMESTER ||--o{ COURSE_OFFERING : contains
    SUBJECT ||--o{ COURSE_OFFERING : offered_as
    COURSE_OFFERING ||--o{ ROLE_ASSIGNMENT : scopes_PJ
    COURSE_OFFERING ||--o{ REGULAR_SCHEDULE : schedules
    REGULAR_SCHEDULE ||--o{ SCHEDULE_CHANGE : changes
    COURSE_OFFERING ||--o{ TASK : owns
    COURSE_OFFERING ||--o{ RESOURCE_LINK : owns
    ROOM ||--o{ REGULAR_SCHEDULE : used_by
    USER ||--o{ AUDIT_LOG : performs
    CLASS_SEMESTER ||--o{ AUDIT_LOG : records
    NOTIFICATION ||--o{ DELIVERY_ATTEMPT : retries
```

Model ini bersifat konseptual. Nama tabel, atribut, indeks, dan strategi migrasi ditetapkan dalam Data Model serta ERD teknis.

## 11. Prioritas Rilis

### MVP

1. Portal kelas hanya-baca tanpa akun mahasiswa.
2. Login undangan untuk PJ, KM, dan System Admin.
3. Pemisahan kelas, semester, mata kuliah, dan hak akses.
4. Jadwal reguler serta perubahan sementara atau permanen.
5. Draf, preview, publikasi PJ, pembatalan KM, dan audit log.
6. Form tugas, deadline buckets, filter, serta arsip.
7. Siaran pagi, pengingat sore, diff jadwal, koreksi, antrean, dan pencegahan duplikasi.
8. Impor JSON, input manual, aktivasi, dan arsip semester.
9. Backup dan pemulihan dasar.

### Fase Lanjutan

1. Pencarian kandidat ruangan kosong.
2. Materi dan tautan terpusat.
3. Approval tugas yang dapat dikonfigurasi per kelas.
4. Deteksi edit bersamaan yang lebih rinci.
5. Penyempurnaan onboarding dan telemetri operasional.

### Ditunda

1. Akun individual mahasiswa dan tracker penyelesaian pribadi.
2. Drag and drop kalender.
3. Auto-pin pesan WhatsApp.
4. Integrasi resmi dengan sistem ruangan TU.
5. Agenda non-akademik dan pembagian kelompok.

## 12. Ukuran Keberhasilan

- PJ dapat membuat tugas dari ponsel tanpa command dalam waktu median kurang dari 30 detik.
- Jadwal pengganti tampil di portal dan siaran bot pada hari pelaksanaan.
- Tidak ada perubahan lintas kelas atau mata kuliah tanpa izin.
- Pergantian semester tidak menghapus data semester sebelumnya.
- Publikasi dan percobaan ulang tidak menghasilkan notifikasi ganda.
- Setiap perubahan penting memiliki pelaku, waktu, data sebelum, dan data sesudah.
- Pembatalan KM mempertahankan riwayat dan menghasilkan koreksi jika diperlukan.
- Penggantian sesi WhatsApp tidak menghapus data akademik.

## 13. Template Siaran WhatsApp

### 13.1 Jadwal Pagi

```text
*JADWAL KULIAH HARI INI*
[Hari], [Tanggal] | [Nama Kelas]

1. *[Mata Kuliah]* ([Teori/Praktik])
   [Jam Mulai] - [Jam Selesai] WIB
   [Ruangan] | [Nama Dosen]

Detail jadwal dan tugas:
https://[domain]/c/[kelas]
```

### 13.2 Pengingat Tugas Sore

```text
*PENGINGAT TUGAS*
[Hari], [Tanggal]

1. *[Mata Kuliah]* | [Judul]
   Deadline: [Tanggal dan Jam]
   Tempat: [Portal atau Lokasi]

Lihat detail:
https://[domain]/c/[kelas]/tasks
```

### 13.3 Perubahan Jadwal

```text
*PERUBAHAN JADWAL*
[Mata Kuliah] ([Teori/Praktik])

Sebelum: [Hari, Jam, Ruangan]
Menjadi: [Hari, Tanggal, Jam, Ruangan]
Status: [Sementara/Permanen]
Keterangan: [Keterangan]

Detail:
https://[domain]/c/[kelas]/schedule
```

### 13.4 Koreksi atau Pembatalan

```text
*KOREKSI JADWAL*
[Mata Kuliah]

Perubahan sebelumnya dibatalkan oleh KM.
Jadwal yang berlaku: [Hari, Tanggal, Jam, Ruangan]
Alasan: [Alasan]

Detail:
https://[domain]/c/[kelas]/schedule
```

## 14. Ketertelusuran

| Kelompok User Requirement | Epic PRD |
|---|---|
| UR-ACCESS | Epic 1 Akses dan Identitas |
| UR-CLASS, UR-SEM | Epic 2 Kelas dan Semester |
| UR-SCH | Epic 3 Jadwal Perkuliahan |
| UR-TASK | Epic 4 Tugas dan Materi |
| UR-NOTIF | Epic 5 Notifikasi WhatsApp |
| UR-ROOM | Epic 6 Ruangan |
| UR-AUDIT, UR-OPS | Epic 7 Audit dan Pemulihan |
| UR-UX | Kebutuhan Pengalaman Pengguna |

Setiap functional requirement harus dipetakan ke user flow, desain, API, data model, dan test case sebelum implementasi dinyatakan siap. Perubahan pada requirement yang telah disetujui mengikuti mekanisme perubahan dalam Product Definition.
