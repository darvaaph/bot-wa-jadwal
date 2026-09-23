# Information Architecture Bot Jadwal

## Status Dokumen

| Atribut | Nilai |
|---|---|
| Versi | 1.0 |
| Status | Draft untuk ditinjau |
| Pemilik | Tim Bot Jadwal |
| Terakhir diperbarui | 23 September 2026 |
| Acuan | [Product Definition](PRODUCT_DEFINITION.md), [Access Control](ACCESS_CONTROL.md), [User Flows](USER_FLOWS.md), [Business Rules](BUSINESS_RULES.md), [Functional Requirements](FUNCTIONAL_REQUIREMENTS.md), dan [PRD](../PRD.md) |

Dokumen ini menetapkan pembagian area aplikasi, hierarki halaman, navigasi, konteks, penamaan, dan hubungan antarkonten. Pola URL bersifat konseptual dan dapat disesuaikan pada API serta routing implementation tanpa mengubah struktur informasi.

## 1. Tujuan dan Prinsip

Bot Jadwal menggunakan mode antarmuka `Operate`: pengguna datang untuk menemukan informasi atau menyelesaikan tugas dengan cepat. Arsitektur informasi mengikuti prinsip berikut:

1. Mahasiswa langsung masuk ke informasi kelas tanpa melewati area pengelola.
2. Pengurus selalu melihat konteks peran, kelas, semester, dan mata kuliah sebelum mengubah data.
3. Jadwal, tugas, dan materi menjadi objek utama. Fitur administratif tidak boleh menghalangi tugas harian.
4. Status publikasi, persetujuan, pembatalan, dan pengiriman WhatsApp ditampilkan terpisah.
5. Navigasi hanya menampilkan tujuan yang benar-benar tersedia bagi peran aktif.

## 2. Ruang Aplikasi

Aplikasi dibagi menjadi empat ruang dengan batas akses yang berbeda.

| Ruang | Pengguna | Akses | Tujuan |
|---|---|---|---|
| Portal Kelas | Mahasiswa dan pengunjung dengan akses kelas | Tanpa akun, hanya-baca | Melihat informasi akademik satu kelas |
| Area Pengelola | PJ dan KM | Login dan role assignment aktif | Mengelola data dalam konteks kelas |
| System Admin | System Admin | Login dengan autentikasi lebih kuat | Mengelola data lintas kelas dan operasi sistem |
| Akses Akun | Calon atau pengurus aktif | Token undangan atau proses pemulihan | Aktivasi, login, dan pemulihan akun |

Portal Kelas dan Area Pengelola bukan variasi dari halaman yang sama. Portal mengutamakan konsumsi informasi, sedangkan Area Pengelola mengutamakan perubahan data dan audit.

Struktur ini menggantikan konsep empat tab tunggal pada `docs/DASHBOARD_PRD.md`. Dokumen tersebut merekam dashboard prototipe sebelum model akses berbasis peran disetujui. Status WhatsApp sekarang berada di area System Admin, sedangkan PJ dan KM melihat status pengiriman pada publikasi terkait.

## 3. Model Konteks

Hierarki konteks menjadi dasar navigasi dan penyaringan data.

```text
Sistem
└── Kelas Permanen
    └── Semester Kelas
        ├── Penawaran Mata Kuliah
        │   ├── Jadwal
        │   ├── Tugas
        │   ├── Materi
        │   └── Penugasan PJ
        ├── Perubahan Jadwal
        └── Riwayat Kelas
```

Aturan konteks:

- Mahasiswa memilih kelas melalui URL atau kode. Semester aktif menjadi konteks awal.
- PJ memilih role assignment jika memiliki lebih dari satu kelas atau mata kuliah.
- KM memilih kelas dan semester. Pilihan mata kuliah berfungsi sebagai filter, bukan batas akses tambahan.
- System Admin bekerja dalam konteks global. Saat membuka kelas untuk dukungan, sistem menampilkan banner mode dukungan dan meminta alasan sebelum perubahan akademik.
- Perpindahan konteks mengatur ulang filter halaman yang tidak berlaku, tetapi tidak mengubah konteks akun lain.

## 4. Peta Situs

```text
Bot Jadwal
├── Portal Kelas /c/{classSlug}
│   ├── Ringkasan
│   ├── Jadwal
│   ├── Tugas
│   │   └── Detail Tugas
│   ├── Mata Kuliah
│   │   └── Detail Mata Kuliah dan Materi
│   ├── Perubahan Terbaru
│   └── Arsip Semester
├── Area Pengelola /app
│   ├── Ringkasan
│   ├── Jadwal
│   │   ├── Jadwal Reguler
│   │   ├── Perubahan Jadwal
│   │   └── Detail dan Riwayat Jadwal
│   ├── Tugas
│   │   ├── Daftar Tugas
│   │   ├── Draf dan Persetujuan
│   │   └── Detail dan Riwayat Tugas
│   ├── Materi
│   ├── Ruangan
│   ├── Anggota dan Peran
│   ├── Semester
│   ├── Riwayat Perubahan
│   └── Pengaturan Kelas
├── System Admin /admin
│   ├── Ringkasan Sistem
│   ├── Kelas
│   │   └── Detail Kelas
│   ├── Pengguna dan Penugasan
│   ├── Master Mata Kuliah
│   ├── Master Ruangan
│   ├── Antrean Notifikasi
│   ├── Backup dan Pemulihan
│   ├── Audit Global
│   └── Status Sistem dan WhatsApp
└── Akun /auth
    ├── Login
    ├── Aktivasi Undangan
    ├── Lupa Kata Sandi
    ├── Atur Ulang Kata Sandi
    └── Akses Ditolak
```

## 5. Portal Kelas

### 5.1 Navigasi Portal

Navigasi utama portal terdiri dari `Ringkasan`, `Jadwal`, `Tugas`, dan `Mata Kuliah`. `Perubahan Terbaru` tampil sebagai bagian Ringkasan dan memiliki halaman daftar melalui tautan `Lihat Semua`. Arsip semester berada pada pemilih semester, bukan navigasi utama.

Pada mobile, empat tujuan utama menggunakan bottom navigation. Pada desktop, tujuan yang sama tampil di header. Portal tidak menampilkan menu akun, administrasi, audit, atau status bot.

### 5.2 Inventaris Halaman Portal

| Halaman | Tujuan | Isi Utama | Aksi Utama |
|---|---|---|---|
| Ringkasan | Menjawab apa yang perlu diketahui sekarang | Jadwal hari ini, perubahan terbaru, tugas terdekat, konteks kelas | Buka detail jadwal atau tugas |
| Jadwal | Melihat jadwal efektif per hari atau minggu | Jadwal reguler yang sudah digabung dengan perubahan aktif | Pilih tanggal, buka detail |
| Tugas | Menemukan tugas berdasarkan urgensi | Hari Ini, Minggu Ini, Mendatang, Terlewat, filter mata kuliah | Buka detail tugas |
| Detail Tugas | Membaca seluruh ketentuan tugas | Judul, instruksi, deadline, mata kuliah, tempat pengumpulan | Buka tautan pengumpulan |
| Mata Kuliah | Menelusuri informasi per mata kuliah | Daftar mata kuliah semester aktif | Buka detail mata kuliah |
| Detail Mata Kuliah | Mengumpulkan jadwal, materi, dan tautan terkait | Jadwal reguler, dosen, materi, tugas aktif | Buka materi atau tugas |
| Perubahan Terbaru | Menelusuri perubahan yang sudah terbit | Perubahan jadwal dan koreksi dalam urutan waktu | Buka perubahan terkait |
| Arsip Semester | Membaca informasi lama yang dipublikasikan | Semester, jadwal, tugas, materi arsip | Ganti semester arsip |

### 5.3 Prioritas Ringkasan Portal

Urutan informasi pada Ringkasan adalah:

1. Konteks kelas dan tanggal.
2. Jadwal yang sedang berlangsung atau berikutnya.
3. Perubahan yang berlaku hari ini.
4. Tugas dengan deadline terdekat.
5. Tautan menuju daftar lengkap.

Jika tidak ada jadwal hari ini, halaman tetap menampilkan perubahan terbaru dan tugas. Status bot tidak ditampilkan karena mahasiswa tidak dapat menindaklanjutinya dan portal tetap menjadi sumber informasi saat bot offline.

## 6. Area Pengelola

### 6.1 Kerangka Navigasi

Area Pengelola memakai satu app shell dengan tiga lapisan:

1. Context switcher untuk peran, kelas, semester, dan mata kuliah PJ.
2. Navigasi utama untuk Ringkasan, Jadwal, Tugas, dan menu Lainnya.
3. Navigasi lokal berupa tab di dalam objek, seperti `Detail`, `Riwayat`, dan `Notifikasi`.

Pada mobile, bottom navigation memuat `Ringkasan`, `Jadwal`, `Tugas`, dan `Lainnya`. Tombol tambah berada pada header atau bagian awal halaman terkait, bukan menjadi tujuan navigasi. Pada desktop, seluruh tujuan yang diizinkan tampil pada sidebar.

### 6.2 Menu Berdasarkan Peran

| Tujuan | PJ | KM | Catatan |
|---|:---:|:---:|---|
| Ringkasan | Ya | Ya | Isi disesuaikan dengan cakupan |
| Jadwal | Ya | Ya | PJ hanya mata kuliah penugasan |
| Tugas | Ya | Ya | Antrean approval hanya untuk KM |
| Materi | Ya | Ya | Sesuai cakupan |
| Ruangan | Ya | Ya | Hasil tetap memerlukan TU |
| Anggota dan Peran | Tidak | Ya | KM mengelola PJ kelasnya |
| Semester | Baca konteks | Kelola | PJ tidak dapat mengaktifkan semester |
| Riwayat Perubahan | Lingkup sendiri | Seluruh kelas | Mengikuti access control |
| Pengaturan Kelas | Tidak | Ya | Mode portal, kode, waktu pengingat, approval tugas |

### 6.3 Inventaris Halaman Pengelola

| Halaman | Isi dan Struktur | Aksi Utama |
|---|---|---|
| Ringkasan | Konteks aktif, tindakan tertunda, jadwal hari ini, tugas terdekat, publikasi terbaru, status notifikasi bermasalah | Lanjutkan draf, buka item bermasalah |
| Jadwal Reguler | Daftar per hari dengan filter mata kuliah, dosen, dan ruangan | Tambah jadwal, ubah jadwal |
| Perubahan Jadwal | Tab `Draf`, `Terbit`, `Dibatalkan`, `Selesai` | Buat perubahan, publikasikan |
| Detail Jadwal | Data efektif, versi reguler, perubahan terkait, konflik, notifikasi, audit | Ubah, buat perubahan, batalkan jika KM |
| Daftar Tugas | Tab `Aktif`, `Draf`, `Menunggu`, `Selesai`, `Terlewat`, `Arsip` | Tambah tugas, filter |
| Detail Tugas | Isi tugas, status, versi, notifikasi, audit | Ubah, publikasikan, arsipkan, pulihkan |
| Draf dan Persetujuan | Draf milik pengguna dan antrean approval KM | Lanjutkan, setujui, kembalikan |
| Materi | Daftar berdasarkan mata kuliah dan jenis tautan | Tambah, ubah, arsipkan |
| Ruangan | Pencarian tanggal dan waktu, kandidat, keterbatasan data | Catat konfirmasi TU |
| Anggota dan Peran | KM aktif, PJ per mata kuliah, undangan, penugasan tidak aktif | Undang, tangguhkan, ganti, cabut |
| Semester | Semester aktif, draf, dan arsip | Buat, impor, salin, review, aktifkan |
| Riwayat Perubahan | Kronologi dengan filter pelaku, objek, tindakan, dan waktu | Buka perbandingan versi |
| Pengaturan Kelas | Akses portal, rotasi kode, approval tugas, waktu pengingat | Simpan pengaturan |

### 6.4 Ringkasan Berdasarkan Peran

Ringkasan PJ mengutamakan draf miliknya, perubahan atau tugas yang perlu diselesaikan, dan mata kuliah penugasannya. Ringkasan KM mengutamakan antrean approval, publikasi terbaru seluruh kelas, undangan, konflik, dan kegagalan notifikasi. Informasi lintas kelas tidak digabung dalam satu daftar tanpa label kelas yang jelas.

## 7. System Admin

### 7.1 Navigasi Admin

System Admin menggunakan area terpisah agar fungsi global tidak bercampur dengan pekerjaan harian kelas. Navigasi desktop menggunakan sidebar. Pada mobile, fungsi inspeksi tetap tersedia, tetapi operasi backup, restore, master data massal, dan audit global dapat mengarahkan pengguna ke tampilan yang lebih sesuai untuk layar lebar tanpa memblokir tindakan darurat.

### 7.2 Inventaris Halaman Admin

| Halaman | Tujuan | Aksi Utama |
|---|---|---|
| Ringkasan Sistem | Melihat kelas bermasalah, antrean gagal, status bot, dan kejadian akses | Buka masalah terkait |
| Kelas | Mencari kelas berdasarkan program, angkatan, rombel, dan status | Buat kelas |
| Detail Kelas | Melihat semester, KM, mode portal, dan ringkasan kesehatan | Undang KM, masuk mode dukungan |
| Pengguna dan Penugasan | Menelusuri akun serta role assignment lintas kelas | Tangguhkan akun, pulihkan akses |
| Master Mata Kuliah | Menjaga identitas mata kuliah yang dipakai lintas kelas | Tambah, ubah, nonaktifkan |
| Master Ruangan | Menjaga daftar ruangan dan status aktif | Tambah, ubah, nonaktifkan |
| Antrean Notifikasi | Melihat pesan Pending, Processing, Sent, dan Failed | Coba ulang pesan gagal |
| Backup dan Pemulihan | Menentukan kelas, semester, paket, serta titik pemulihan | Buat backup, validasi, restore |
| Audit Global | Menelusuri tindakan keamanan dan dukungan | Filter, buka detail |
| Status Sistem dan WhatsApp | Melihat kesehatan aplikasi, koneksi bot, dan proses pengiriman | Sambungkan ulang sesuai prosedur |

Status WhatsApp tidak menjadi tab utama Area Pengelola. PJ dan KM cukup melihat status pengiriman pada publikasi terkait. Telemetri dan penyambungan ulang hanya tersedia untuk System Admin.

## 8. Akses Akun

| Halaman | Informasi Wajib | Hasil |
|---|---|---|
| Login | Identitas akun, kata sandi, tautan pemulihan | Masuk ke satu konteks atau pemilih konteks |
| Aktivasi Undangan | Pemberi akses, peran, kelas, semester, mata kuliah jika PJ | Membuat akun atau menambah penugasan |
| Lupa Kata Sandi | Identitas akun tanpa mengonfirmasi keberadaan akun | Mengirim instruksi pemulihan |
| Atur Ulang Kata Sandi | Status token dan input kata sandi baru | Mencabut token lama dan kembali ke login |
| Akses Ditolak | Alasan umum dan konteks aktif | Kembali, ganti konteks, atau login |

Halaman undangan tidak mengizinkan penerima mengubah peran atau cakupan. Pesan login dan pemulihan tidak boleh mengungkap apakah identitas tertentu terdaftar.

## 9. Pola Navigasi dan Hubungan Konten

### 9.1 Tautan Kontekstual

- Jadwal menautkan mata kuliah, perubahan aktif, ruangan, dan audit terkait.
- Tugas menautkan mata kuliah, materi pendukung, tempat pengumpulan, versi, dan notifikasi.
- Mata kuliah menautkan jadwal reguler, tugas aktif, materi, dosen, dan PJ.
- Publikasi menautkan status pengiriman WhatsApp dan koreksi yang menggantikannya.
- Audit menautkan kembali ke objek, tetapi tidak memberikan izin tambahan untuk membuka objek tersebut.

### 9.2 Breadcrumb

Breadcrumb digunakan pada halaman detail dengan kedalaman lebih dari dua tingkat, misalnya `Kelas > Semester > Jadwal > Detail`. Pada mobile, breadcrumb diringkas menjadi tombol kembali dengan label induk. Context switcher tidak menggantikan breadcrumb karena keduanya menjawab pertanyaan yang berbeda.

### 9.3 URL dan Deep Link

Setiap detail yang dikirim melalui WhatsApp harus memiliki URL stabil. URL wajib membuka kelas dan objek yang benar setelah pemeriksaan akses. Jika objek dipindahkan ke arsip, URL tetap membuka detail arsip. Jika akses tidak valid, sistem mengarahkan ke kode kelas atau login lalu kembali ke tujuan awal.

## 10. Pencarian, Filter, dan Pengurutan

| Area | Pencarian atau Filter | Urutan Awal |
|---|---|---|
| Jadwal | Tanggal, hari, mata kuliah, dosen, jenis perubahan | Waktu mulai |
| Tugas | Mata kuliah, status, rentang deadline | Deadline terdekat |
| Materi | Mata kuliah, jenis, kata kunci | Terbaru diperbarui |
| Anggota | Peran, mata kuliah, status penugasan | Nama |
| Riwayat | Pelaku, objek, tindakan, rentang waktu | Terbaru |
| Kelas Admin | Program, angkatan, rombel, status | Identitas kelas |
| Notifikasi | Kelas, jenis, status, waktu | Terbaru |

Filter aktif selalu terlihat dan dapat dihapus satu per satu atau direset. Filter yang tidak berlaku setelah pergantian konteks dihapus dengan pemberitahuan singkat.

## 11. State Halaman

Setiap halaman data harus mendefinisikan state berikut:

| State | Perilaku |
|---|---|
| Loading | Mempertahankan kerangka halaman dan menunjukkan bagian yang sedang dimuat |
| Empty awal | Menjelaskan bahwa data belum dibuat dan menawarkan aksi jika pengguna berwenang |
| Empty hasil filter | Menjelaskan bahwa tidak ada hasil dan menawarkan reset filter |
| Error | Menyebut data yang gagal dimuat serta menyediakan coba ulang |
| Permission denied | Tidak membuka data dan menawarkan kembali atau ganti konteks |
| Offline bot | Menegaskan bahwa data web tetap tersimpan dan pesan sedang mengantre |
| Conflict edit | Menampilkan versi terbaru dan mempertahankan input pengguna |
| Success | Mengonfirmasi objek dan status baru, bukan hanya menampilkan pesan generik |

State memakai teks dan ikon, tidak hanya warna. Aksi berisiko seperti pembatalan, pencabutan peran, restore, dan penghapusan memerlukan dialog konfirmasi yang menyebut dampaknya.

## 12. Responsivitas dan Aksesibilitas

- Mobile menjadi konteks utama bagi portal, PJ, dan KM. Informasi terpenting tampil sebelum filter lanjutan.
- Tabel berubah menjadi daftar berlabel pada layar sempit. Data penting tidak disembunyikan hanya karena ruang terbatas.
- Bottom navigation dibatasi empat tujuan. Tujuan lain masuk ke `Lainnya` dengan judul yang jelas.
- Target sentuh minimal 44 x 44 piksel. Navigasi keyboard mengikuti urutan visual dan fokus selalu terlihat.
- Drawer, dialog, dan pemilih konteks dapat ditutup dengan Escape serta mengembalikan fokus ke pemicu.
- Label status menggunakan Bahasa Indonesia yang konsisten dan tetap dapat dipahami pembaca layar.

## 13. Terminologi Antarmuka

| Gunakan | Hindari | Alasan |
|---|---|---|
| Ketua Murid (KM) | Admin kelas, Komti secara bergantian | Menjaga nama peran resmi dalam produk |
| PJ Mata Kuliah | Admin mata kuliah | Menjelaskan cakupan tanggung jawab |
| Perubahan Jadwal | Override | Bahasa pengguna |
| Terbit | Aktif untuk seluruh status | Membedakan publikasi dari status semester atau peran |
| Dibatalkan | Dihapus | Riwayat publikasi tetap ada |
| Menunggu Persetujuan | Pending tanpa penjelasan | Status dapat dipahami tanpa istilah teknis |
| Ruangan Kandidat | Ruangan Kosong | Data internal belum menjadi konfirmasi resmi |
| Riwayat Perubahan | Log | Lebih mudah dipahami pengguna umum |

## 14. Prioritas Implementasi Informasi

### MVP

1. Akses akun dan context switcher.
2. Portal Ringkasan, Jadwal, Tugas, dan Detail Tugas.
3. Area Pengelola untuk Jadwal, Tugas, draf, publikasi, dan pembatalan KM.
4. Semester aktif, input manual, impor JSON, serta arsip.
5. Antrean notifikasi, status pengiriman, dan audit dasar.
6. Pengelolaan anggota serta peran.

### Fase Lanjutan

1. Materi dan tautan terpusat.
2. Pencarian kandidat ruangan dan konfirmasi TU.
3. Backup serta restore dari antarmuka.
4. Audit global dan filter lanjutan.
5. Status sistem yang lebih rinci.

Pengelompokan fase tidak mengubah aturan akses atau struktur konteks. Halaman yang belum tersedia tidak ditampilkan sebagai navigasi mati.

## 15. Ketertelusuran

| Area IA | Functional Requirement Utama |
|---|---|
| Portal Kelas | FR-ACCESS-001, FR-SCH-001, FR-TASK-004, FR-TASK-007 |
| Akses Akun | FR-ACCESS-002 sampai FR-ACCESS-007 |
| Konteks dan Kelas | FR-ACCESS-005, FR-CLASS-001, FR-CLASS-002 |
| Semester | FR-SEM-001 sampai FR-SEM-004 |
| Jadwal | FR-SCH-001 sampai FR-SCH-007 |
| Tugas dan Materi | FR-TASK-001 sampai FR-TASK-007 |
| Notifikasi | FR-NOTIF-001 sampai FR-NOTIF-005 |
| Ruangan | FR-ROOM-001 sampai FR-ROOM-003 |
| Audit dan Operasional | FR-AUDIT-001 sampai FR-OPS-004 |
| State dan Aksesibilitas | FR-UX-001 sampai FR-UX-004 |

## 16. Kriteria Selesai

Information Architecture siap menjadi dasar wireframe jika setiap functional requirement memiliki lokasi antarmuka, setiap halaman memiliki pengguna serta tujuan yang jelas, semua jalur perubahan memperlihatkan konteks aktif, dan state loading, empty, error, permission denied, serta conflict telah memiliki tempat. Perubahan struktur yang memengaruhi akses, status, atau alur wajib memperbarui dokumen sumber dan wireframe terkait.
