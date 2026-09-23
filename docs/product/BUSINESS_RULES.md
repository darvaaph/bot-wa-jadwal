# Business Rules Bot Jadwal

## Status Dokumen

| Atribut | Nilai |
|---|---|
| Versi | 2.0.0 |
| Status | Approved |
| Pemilik | Tim Bot Jadwal |
| Terakhir diperbarui | 23 September 2026 |
| Acuan | [Product Definition](PRODUCT_DEFINITION.md), [User Requirements](USER_REQUIREMENTS.md), [Access Control](ACCESS_CONTROL.md), [User Flows](USER_FLOWS.md), dan [PRD](../PRD.md) |
| Turunan sistem | [Functional Requirements](FUNCTIONAL_REQUIREMENTS.md) |

Dokumen ini menetapkan aturan yang harus selalu dipenuhi oleh antarmuka, API, bot WhatsApp, proses latar belakang, dan database. Dokumen ini menjelaskan keputusan bisnis, bukan struktur tabel atau pilihan framework.

## 1. Cara Membaca Aturan

- `Wajib` berarti sistem harus menegakkan ketentuan tersebut.
- `Dilarang` berarti permintaan harus ditolak atau tidak ditampilkan.
- `Boleh` berarti tindakan tersedia jika syarat dan izin terpenuhi.
- Semua pemeriksaan izin dijalankan di backend. Menyembunyikan tombol saja tidak memenuhi aturan.
- ID aturan tidak digunakan kembali. Aturan yang tidak berlaku diberi status `Deprecated` dan menunjuk aturan pengganti.

## 2. Akses dan Identitas

### BR-ACCESS-001 Perubahan Memerlukan Autentikasi

Mahasiswa boleh membaca portal kelas tanpa akun melalui tautan atau kode kelas. Setiap operasi tambah, ubah, publikasi, pembatalan, penghapusan, pemulihan, dan perubahan akses wajib menggunakan akun aktif serta sesi yang valid.

### BR-ACCESS-002 Cakupan PJ

PJ hanya boleh mengelola penawaran mata kuliah yang tercantum pada penugasan aktifnya. Pemeriksaan wajib mencocokkan `class_id`, `semester_id`, dan `course_offering_id`. Permintaan di luar cakupan ditolak tanpa mengungkap keberadaan data.

### BR-ACCESS-003 Pencabutan Peran

Role assignment yang ditangguhkan, dicabut, belum mencapai `valid_from`, atau telah melewati `valid_until` tidak boleh dipakai untuk operasi baru. Pergantian semester tidak menonaktifkan KM. Course offering semester arsip membuat cakupan PJ hanya-baca. Pencabutan tidak menghapus data maupun identitas pelaku pada audit log.

### BR-ACCESS-004 Beberapa Peran dalam Satu Akun

Satu akun boleh memiliki beberapa penugasan. Izin dihitung dari konteks aktif, bukan dari peran terluas yang pernah dimiliki pengguna. Antarmuka wajib menampilkan peran, kelas, dan semester aktif sebelum pengguna melakukan perubahan.

### BR-ACCESS-005 Akun Pengurus Melalui Undangan

Pendaftaran publik untuk PJ, KM, dan System Admin dilarang. Undangan hanya dapat digunakan sekali, memiliki masa berlaku, dapat dicabut, dan terikat pada identitas serta cakupan yang ditetapkan pemberi akses. Pengiriman ulang membatalkan token lama.

### BR-ACCESS-006 Akses Dukungan System Admin

System Admin menjalankan perubahan akademik hanya untuk dukungan, insiden, atau pemulihan. Sistem wajib meminta alasan, mencatat cakupan, dan menyimpan pelaku. System Admin pertama dibuat melalui provisioning, bukan endpoint publik.

## 3. Kelas, Portal, dan Semester

### BR-CLASS-001 Isolasi Kelas

Setiap data operasional wajib memiliki `class_id`. Membaca, menulis, mengimpor, mengekspor, dan memulihkan data tidak boleh mencampurkan kelas. Identitas kelas bersifat permanen dan tidak berubah ketika semester berganti.

### BR-CLASS-002 Portal Mahasiswa

Portal mahasiswa bersifat hanya-baca. Mode akses kelas adalah tautan atau kode kelas. Sesi kode disimpan di server dengan versi kode; rotasi kode membatalkan seluruh sesi versi lama. Kode aktif membuka semester terbit, termasuk arsip hanya-baca, tetapi tidak dapat digunakan sebagai kredensial administrasi. Audit log, data akun, nomor telepon, dan konfigurasi bot tidak boleh ditampilkan.

### BR-SEM-001 Cakupan Semester

Pola jadwal, teaching event, tugas, materi khusus offering, penawaran mata kuliah, dan penugasan PJ wajib dapat ditelusuri ke semester. Materi kelas umum boleh tidak memiliki semester. Operasi pada semester arsip bersifat hanya-baca, kecuali pemulihan System Admin yang tercatat.

### BR-SEM-002 Semester Aktif Tunggal

Satu kelas hanya boleh memiliki satu semester aktif. Semester wajib memiliki `starts_on` dan `ends_on` yang valid. Semester baru dibuat sebagai draf dan tidak terlihat sebagai semester berjalan sebelum aktivasi berhasil.

### BR-SEM-003 Aktivasi dan Pengarsipan

Aktivasi semester baru, pengisian `published_at`, dan pengarsipan semester lama harus terjadi dalam satu transaksi. Kegagalan salah satu langkah membatalkan seluruh aktivasi. Riwayat semester lama yang pernah aktif tetap dapat dibaca melalui portal kelas.

### BR-SEM-004 Wewenang Aktivasi

Hanya KM aktif pada kelas tersebut atau System Admin yang boleh mengaktifkan semester. PJ tidak boleh mengaktifkan semester. Penugasan PJ dari semester lama tidak otomatis aktif pada semester baru.

### BR-SEM-005 Sumber Data Semester

Semester boleh disiapkan melalui impor JSON, salinan semester sebelumnya, atau input manual. Semua metode menghasilkan draf yang harus dipreview. Impor yang memiliki kesalahan pemblokir dilarang menulis data aktif atau meninggalkan data parsial.

## 4. Jadwal Perkuliahan

### BR-SCH-001 Jenis dan Dampak Perubahan

`schedule_patterns` mendefinisikan pola reguler. `teaching_events` mendefinisikan kejadian pengganti, tambahan, libur, atau sesi yang dibatalkan. Event tambahan boleh tidak memiliki pola asal; event pengganti dan pembatalan wajib menunjuk pola serta tanggal asal. Perubahan permanen membuat versi pola baru sejak tanggal berlaku.

### BR-SCH-002 Preview Sebelum Publikasi

Perubahan jadwal wajib melewati preview yang menampilkan mata kuliah, data lama, data baru, jenis perubahan, waktu berlaku, dan penerima. Draf tidak terlihat oleh mahasiswa dan tidak membuat notifikasi.

### BR-SCH-003 Publikasi Idempoten

Satu tindakan publikasi memiliki identitas unik. Percobaan ulang dengan identitas yang sama tidak boleh membuat publikasi atau notifikasi kedua. Perubahan isi setelah publikasi harus dibuat sebagai pembaruan atau koreksi yang dapat ditelusuri.

### BR-SCH-004 Publikasi oleh PJ

PJ boleh memublikasikan perubahan jadwal untuk mata kuliah dalam penugasan aktifnya tanpa persetujuan KM. Sistem wajib mencatat versi data, pelaku, waktu, dan nilai sebelum serta sesudah.

### BR-SCH-005 Pencabutan Publikasi oleh KM

Hanya KM pada kelas terkait yang boleh mencabut teaching event yang sudah terbit. PJ dapat membuat koreksi baru atau meminta KM melakukan pencabutan. Lifecycle publikasi berubah menjadi `REVOKED`, menyimpan alasan, dan tidak menghapus publikasi awal. `SESSION_CANCELLED` tetap berarti sesi akademik dibatalkan.

### BR-SCH-006 Koreksi kepada Mahasiswa

Jika event yang dicabut sudah disiarkan, sistem wajib membuat notifikasi koreksi yang menghubungkan pesan lama, pencabutan, dan jadwal yang kembali berlaku. Pencabutan kedua terhadap event yang sama ditolak.

### BR-SCH-007 Konflik Jadwal

Sistem wajib memeriksa benturan kelas, waktu, dan ruangan berdasarkan data internal sebelum penyimpanan. Konflik ditampilkan bersama data yang bertabrakan. Jika konflik bukan pemblokir, pengguna wajib mengonfirmasi dan menulis alasan. Hasil pemeriksaan ruangan tidak menggantikan konfirmasi resmi TU.

### BR-SCH-008 Perkuliahan Lintas Kelas

Satu teaching event boleh terhubung ke beberapa course offering. Satu kelas wajib menjadi pemilik. KM kelas peserta harus menerima partisipasi sebelum event tampil pada kelasnya. KM peserta boleh melepas kelasnya tetapi tidak boleh mengubah waktu, tempat, jenis, atau lifecycle acara utama.

## 5. Tugas dan Materi

### BR-TASK-001 Data Minimum Tugas

Tugas yang diterbitkan wajib memiliki mata kuliah, judul, instruksi, deadline, dan tempat atau tautan pengumpulan. Deadline menggunakan zona waktu kelas. Draf boleh belum lengkap, tetapi tidak boleh dipublikasikan sebelum lolos validasi.

### BR-TASK-002 Deadline dan Arsip

Tugas yang melewati deadline berubah menjadi `OVERDUE`, bukan dihapus. Tugas yang ditandai selesai berubah menjadi `COMPLETED`. Pengarsipan mengisi `archived_at` tanpa mengganti status hasil sehingga tugas tetap dapat dicari dan diaudit.

### BR-TASK-003 Review Retrospektif

PJ boleh langsung memublikasikan tugas. KM wajib memiliki alur review retrospektif dan dapat memberi keputusan `APPROVED`, `CHANGES_REQUESTED`, atau `REVOKED`. `CHANGES_REQUESTED` mengembalikan tugas ke `DRAFT`; `REVOKED` mengakhiri publikasi. Keduanya menarik tugas dari portal dan wajib memiliki catatan. Riwayat review bersifat append-only.

### BR-TASK-004 Perubahan Tugas Terbit

PJ boleh mengubah tugas terbit dalam cakupannya dan KM boleh mengubah seluruh tugas pada kelasnya. Sistem wajib menyimpan versi sebelumnya. Perubahan deadline atau tempat pengumpulan yang memengaruhi mahasiswa harus menghasilkan pembaruan notifikasi sesuai kebijakan kelas.

### BR-TASK-005 Materi dan Tautan

Materi dan tautan wajib memiliki `class_id`. Materi umum kelas boleh tidak memiliki course offering; materi mata kuliah wajib menunjuk course offering yang kelasnya sama. PJ hanya boleh membuat materi pada offering penugasannya, sedangkan KM boleh membuat materi umum kelas.

## 6. Notifikasi WhatsApp

### BR-NOTIF-001 Portal sebagai Sumber Status

Publikasi data web tidak bergantung pada keberhasilan WhatsApp. Jika bot offline, data tetap terbit di portal dan notifikasi masuk antrean. Antarmuka pengelola wajib membedakan status data terbit dari status pesan terkirim.

### BR-NOTIF-002 Pengiriman Tepat Satu Secara Logis

Setiap notifikasi memiliki kunci idempotensi. Percobaan ulang memakai catatan yang sama dan tidak membuat pesan bisnis baru. Sistem mencatat status `Pending`, `Processing`, `Sent`, atau `Failed`, jumlah percobaan, serta kesalahan terakhir.

### BR-NOTIF-003 Isi Pesan

Pesan jadwal harian memuat mata kuliah, jam, dosen, dan ruangan. Pesan perubahan membandingkan data lama dengan data baru serta menyebut jenis dan waktu berlaku. Pesan tugas memuat mata kuliah, judul, deadline, dan tautan detail. Informasi panjang tetap berada di portal.

### BR-NOTIF-004 Koreksi Sebelum Pesan Terkirim

Jika data berubah saat pesan lama masih mengantre, sistem tidak boleh mengirim informasi yang sudah salah sebagai pesan terbaru. Sistem mengganti pesan tertunda atau membatalkannya lalu membuat pesan koreksi dengan hubungan audit yang jelas.

### BR-NOTIF-005 Pengingat Kelas Pengganti

Teaching event pengganti yang masih `PUBLISHED` wajib menghasilkan pengingat sebelum waktu mulai. Perubahan waktu memperbarui pesan tertunda. Event `REVOKED` membatalkan pesan tertunda, dan percobaan ulang memakai kunci idempotensi yang sama.

## 7. Ruangan

### BR-ROOM-001 Rekomendasi Internal

Status ruangan kosong hanya berarti tidak ada benturan pada data yang tersedia di sistem. Hasil wajib menampilkan waktu pembaruan data dan peringatan untuk mengonfirmasi ke TU. Sistem dilarang menyatakan ruangan pasti tersedia sebelum konfirmasi.

### BR-ROOM-002 Master Ruangan

Hanya System Admin yang boleh menambah, mengubah, atau menonaktifkan master ruangan lintas kelas. KM boleh mengirim usulan koreksi. Menonaktifkan ruangan tidak menghapus referensi pada jadwal lama.

### BR-ROOM-003 Hasil Konfirmasi

PJ atau KM membuat teaching event draf sebelum menghubungi TU. TU tidak memiliki akun dan konfirmasi tetap berlangsung di luar aplikasi. Hasil konfirmasi manual wajib terikat pada draf dan memuat status, ruangan, nama petugas atau keterangan sumber, catatan, pelaku pencatat, serta waktu. Catatan tidak mengubah master ruangan secara otomatis.

## 8. Audit, Integritas, dan Pemulihan

### BR-AUDIT-001 Tindakan yang Dicatat

Sistem wajib mencatat tambah, ubah, hapus, pulihkan, publikasi, pencabutan, pembatalan sesi akademik, aktivasi semester, perubahan peran, pemulihan akun, rotasi kode kelas, dan aksi dukungan. Catatan memuat pengguna, role assignment atau snapshot konteks sistem, waktu, cakupan, objek, tindakan, serta nilai sebelum dan sesudah jika relevan.

### BR-AUDIT-002 Retensi

Audit log dan arsip disimpan selama kelas aktif ditambah sekurangnya satu tahun akademik. Proses biasa tidak boleh mengubah atau menghapus audit log. Penghapusan setelah masa retensi memerlukan kebijakan dan otorisasi terpisah.

### BR-OPS-001 Konflik Edit

Setiap data yang dapat diubah bersama wajib memiliki versi. Penyimpanan dari versi lama ditolak tanpa menimpa data terbaru. Sistem menampilkan versi terkini dan mempertahankan input pengguna agar dapat disalin atau diperbaiki.

### BR-OPS-002 Penghapusan dan Pemulihan

Tindakan hapus biasa menggunakan soft delete untuk data penting. Pemulihan mengaktifkan kembali data sesuai izin tanpa menghapus catatan penghapusan. Penghapusan permanen hanya tersedia melalui prosedur administratif terpisah.

### BR-OPS-003 Backup dan Restore

Backup dan restore wajib menyebut kelas serta semester yang dicakup. Sistem memvalidasi paket, membuat titik pemulihan sebelum restore, dan menolak paket yang tidak cocok dengan tujuan. Kegagalan restore harus mengembalikan keadaan sebelum proses.

### BR-OPS-004 Data Akademik Terpisah dari Sesi Bot

Pergantian nomor, sesi, atau koneksi WhatsApp tidak boleh menghapus atau memindahkan data akademik. Koneksi bot hanya memengaruhi kemampuan pengiriman pesan.

## 9. Transisi Status yang Diizinkan

| Entitas | Dari | Ke | Pelaku atau Pemicu |
|---|---|---|---|
| Undangan | `PENDING` | `ACCEPTED`, `EXPIRED`, atau `REVOKED` | Penerima, waktu, atau pemberi akses |
| Role assignment | `ACTIVE` | `SUSPENDED` atau `REVOKED` | Pemberi akses berwenang |
| Role assignment | `SUSPENDED` | `ACTIVE` atau `REVOKED` | Pemberi akses berwenang |
| Teaching event | `DRAFT` | `PUBLISHED` | PJ atau KM sesuai cakupan |
| Teaching event | `PUBLISHED` | `REVOKED` | KM kelas pemilik |
| Partisipasi event | `PENDING` | `ACCEPTED`, `DECLINED`, atau `REMOVED` | KM kelas peserta |
| Tugas | `DRAFT` | `PUBLISHED` | PJ atau KM |
| Tugas | `PUBLISHED` | `DRAFT` | KM meminta koreksi |
| Tugas | `PUBLISHED` | `COMPLETED`, `OVERDUE`, atau `REVOKED` | Pengurus, deadline, atau KM |
| Review tugas | `NOT_REVIEWED` | `APPROVED`, `CHANGES_REQUESTED`, atau `REVOKED` | KM |
| Semester | `DRAFT` | `ACTIVE` | KM atau System Admin |
| Semester | `ACTIVE` | `ARCHIVED` | Aktivasi semester pengganti |

Transisi di luar tabel ditolak kecuali dokumen ini diperbarui. Status arsip tidak menghapus entitas atau riwayatnya.

## 10. Nilai Konfigurasi yang Masih Perlu Ditetapkan

Nilai berikut tidak mengubah aturan bisnis, tetapi wajib ditentukan sebelum implementasi produksi:

| Konfigurasi | Pemilik Keputusan |
|---|---|
| Masa berlaku undangan | Tim produk dan keamanan |
| Batas percobaan kode kelas | Tim keamanan |
| Jadwal pengingat pagi dan sore | KM per kelas, dengan nilai awal sistem |
| Jumlah serta jeda percobaan ulang notifikasi | Tim operasional |
| Masa pemulihan soft delete | Tim produk dan operasional |

## 11. Ketertelusuran

| Kelompok Aturan | Requirement dan Flow Utama |
|---|---|
| `BR-ACCESS`, `BR-CLASS` | UR-ACCESS-001 sampai UR-ACCESS-006; UF-ACCESS-001 sampai UF-PORTAL-001 |
| `BR-SEM` | UR-SEM-001 sampai UR-SEM-003; UF-SEM-001 sampai UF-SEM-002 |
| `BR-SCH` | UR-SCH-001 sampai UR-SCH-009; UF-SCH-001 sampai UF-SCH-005 |
| `BR-TASK` | UR-TASK-001 sampai UR-TASK-007; UF-TASK-001 sampai UF-TASK-003 |
| `BR-NOTIF` | UR-NOTIF-001 sampai UR-NOTIF-006; UF-OPS-001 |
| `BR-ROOM` | UR-ROOM-001 sampai UR-ROOM-003; UF-ROOM-001 |
| `BR-AUDIT`, `BR-OPS` | UR-AUDIT-001 sampai UR-AUDIT-003; UR-OPS-001 sampai UR-OPS-004; UF-OPS-001 sampai UF-OPS-003 |

## 12. Kriteria Selesai Business Rule

Sebuah aturan siap diterjemahkan menjadi functional requirement dan test case jika aktor, kondisi awal, tindakan, hasil, penolakan, status data, audit, dan referensinya dapat diuji. Perubahan aturan yang memengaruhi akses, status, atau publikasi wajib diikuti pembaruan pada Product Definition, User Requirements, Access Control, User Flows, PRD, dan test case terkait.

## 13. Changelog

### 2.0.0, 23 September 2026

- Memisahkan undangan dari role assignment dan menetapkan cakupan KM lintas semester.
- Mengganti model perubahan jadwal dengan pola reguler serta teaching event dan menambah aturan lintas kelas.
- Menetapkan review tugas retrospektif, pengingat kelas pengganti, konfirmasi TU berbasis draf, materi umum, dan audit role context.
