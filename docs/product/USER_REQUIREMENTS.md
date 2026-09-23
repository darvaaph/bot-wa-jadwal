# User Requirements Bot Jadwal

## Status Dokumen

| Atribut | Nilai |
|---|---|
| Versi | 1.0 |
| Status | Approved |
| Pemilik | Tim Bot Jadwal |
| Terakhir diperbarui | 23 September 2026 |
| Acuan keputusan | [Product Definition versi 0.3](PRODUCT_DEFINITION.md) |
| Sumber riset | [Wawancara 11 pengguna](../user-interviews/README.md) |
| Turunan sistem | [Functional Requirements v1.0](FUNCTIONAL_REQUIREMENTS.md) |

Dokumen ini mencatat kebutuhan dari sudut pandang pengguna Bot Jadwal. Setiap requirement memiliki ID permanen agar dapat ditelusuri ke PRD, aturan bisnis, desain, implementasi, dan pengujian. Kriteria keberhasilan menjelaskan hasil yang harus dirasakan pengguna tanpa menetapkan detail implementasi yang belum diperlukan.

## 1. Tujuan

User Requirements digunakan untuk memastikan fitur dibangun karena kebutuhan pengguna atau keputusan produk yang telah disetujui. Dokumen ini menjadi penghubung antara hasil wawancara dan spesifikasi produk. Detail endpoint, tabel database, framework, serta pilihan algoritma tidak dibahas di sini.

## 2. Aktor

| Aktor | Kebutuhan Utama |
|---|---|
| Mahasiswa | Melihat jadwal, tugas, perubahan, ruangan, dan materi dengan cepat |
| PJ Mata Kuliah | Mengelola tugas dan jadwal mata kuliah yang ditugaskan tanpa command |
| Ketua Murid | Mengelola seluruh mata kuliah dalam kelas dan mengoreksi perubahan yang keliru |
| System Admin | Mengelola kelas, akses awal, semester, master data, dan pemulihan sistem |

## 3. Prioritas

- `Must`: diperlukan agar produk menyelesaikan masalah utama dan dapat digunakan.
- `Should`: bernilai tinggi, tetapi dapat dikerjakan setelah alur inti stabil.
- `Could`: peningkatan lanjutan yang tidak menghalangi penggunaan utama.

Semua requirement dalam dokumen ini berstatus `Approved`. Prioritas menentukan urutan implementasi, bukan tingkat persetujuannya.

## 4. Akses dan Identitas

| ID | User Requirement | Aktor | Sumber | Prioritas | Kriteria Keberhasilan | Aturan Terkait |
|---|---|---|---|---|---|---|
| UR-ACCESS-001 | Mahasiswa membutuhkan akses cepat untuk melihat informasi kelas tanpa membuat akun. | Mahasiswa | Product Definition OQ-001 | Must | Mahasiswa dapat membuka portal kelas melalui tautan atau kode kelas; akses hanya-baca; fungsi administrasi dan informasi sensitif tidak terlihat. | BR-ACCESS-001 |
| UR-ACCESS-002 | Pengurus membutuhkan akun terverifikasi agar hanya orang yang ditunjuk dapat mengubah data. | PJ, KM | Product Definition OQ-002 | Must | Akun pengurus dibuat melalui undangan; pengguna yang belum terautentikasi tidak dapat mengubah data; WhatsApp dapat digunakan untuk verifikasi atau pemulihan. | BR-ACCESS-001, BR-ACCESS-003 |
| UR-ACCESS-003 | PJ membutuhkan akses yang terbatas pada mata kuliah yang menjadi tanggung jawabnya. | PJ | Bima, Imam KM | Must | PJ hanya dapat melihat fungsi pengelolaan untuk mata kuliah yang ditugaskan; permintaan perubahan di luar cakupan ditolak; pencabutan peran langsung menghentikan akses. | BR-ACCESS-002, BR-ACCESS-003 |
| UR-ACCESS-004 | KM membutuhkan wewenang untuk mengelola seluruh mata kuliah pada kelas yang ditugaskan kepadanya. | KM | Bima, Imam KM | Must | KM dapat mengelola jadwal, tugas, PJ, dan koreksi pada kelasnya; KM tidak memperoleh akses ke kelas lain tanpa penugasan tambahan. | BR-CLASS-001 |
| UR-ACCESS-005 | Satu pengguna membutuhkan dukungan beberapa peran apabila bertanggung jawab pada lebih dari satu kelas atau semester. | PJ, KM | Product Definition OQ-009 | Should | Peran dan cakupan ditentukan per kelas serta semester; pengguna dapat berpindah konteks tanpa membuat akun baru; izin setiap konteks tetap terpisah. | BR-ACCESS-004 |
| UR-ACCESS-006 | Pengelola sistem membutuhkan cara memulihkan akses kelas ketika KM atau PJ kehilangan akun. | System Admin | Product Definition OQ-002 | Must | System Admin dapat memulihkan atau memindahkan akses; tindakan pemulihan tercatat; data kelas tidak berubah akibat pergantian akun. | BR-AUDIT-001 |

## 5. Kelas dan Semester

| ID | User Requirement | Aktor | Sumber | Prioritas | Kriteria Keberhasilan | Aturan Terkait |
|---|---|---|---|---|---|---|
| UR-CLASS-001 | Pengguna membutuhkan pemisahan informasi yang jelas antara satu kelas dan kelas lainnya. | Semua aktor | Product Definition bagian 7 | Must | Jadwal, tugas, pengurus, dan materi hanya muncul pada kelas yang dipilih; perubahan pada satu kelas tidak memengaruhi kelas lain. | BR-CLASS-001 |
| UR-CLASS-002 | KM membutuhkan identitas kelas yang tetap meskipun semester berubah. | KM | Product Definition bagian 7 | Must | Kelas tetap dapat dikenali berdasarkan program, angkatan, dan rombel; pergantian semester tidak membuat kelas baru yang terputus dari riwayatnya. | BR-CLASS-001 |
| UR-SEM-001 | KM membutuhkan cara menyiapkan semester baru tanpa menghapus data semester sebelumnya. | KM | Product Definition OQ-004 | Must | Hanya satu semester aktif per kelas; semester sebelumnya menjadi arsip hanya-baca; jadwal dan tugas lama tetap dapat ditelusuri. | BR-SEM-002, BR-SEM-003, BR-SEM-004 |
| UR-SEM-002 | Pengelola membutuhkan beberapa cara untuk menyiapkan jadwal semester baru. | KM, System Admin | Diskusi produk dan kebutuhan pergantian semester | Must | Pengguna dapat memilih impor JSON, menyalin semester sebelumnya, atau mengisi jadwal manual; data dapat diperiksa sebelum semester diaktifkan. | BR-SEM-003, BR-SEM-004 |
| UR-SEM-003 | Pengelola membutuhkan umpan balik yang jelas ketika data impor semester tidak valid. | KM, System Admin | Product Definition bagian 13 | Must | Sistem menunjukkan data yang bermasalah; semester tidak dapat diaktifkan sebelum kesalahan diselesaikan; kegagalan impor tidak meninggalkan data parsial tanpa konfirmasi. | BR-SEM-001, BR-SEM-004 |

## 6. Jadwal Perkuliahan

| ID | User Requirement | Aktor | Sumber | Prioritas | Kriteria Keberhasilan | Aturan Terkait |
|---|---|---|---|---|---|---|
| UR-SCH-001 | Mahasiswa membutuhkan tampilan jadwal hari ini yang menggunakan bahasa dan informasi yang mudah dipahami. | Mahasiswa | Iman, Hana, Anindya, Kemal | Must | Jadwal menampilkan nama mata kuliah, jenis, jam mulai dan selesai, dosen, serta ruangan; pengguna tidak perlu memahami kode internal. | BR-CLASS-001, BR-SEM-001 |
| UR-SCH-002 | Pengurus membutuhkan formulir untuk menambah atau mengubah jadwal tanpa command WhatsApp. | PJ, KM | Iman, Hana, Giza, Kemal | Must | Pengguna dapat memilih mata kuliah, tanggal atau hari, jam, durasi, dosen, dan ruangan melalui input terpandu; jadwal dapat dibuat manual. | BR-ACCESS-002, BR-SCH-002 |
| UR-SCH-003 | Pengurus membutuhkan cara membedakan perubahan jadwal sementara dan permanen. | PJ, KM | Imam KM, Anindya, Kemal | Must | Perubahan sementara berlaku pada sesi atau tanggal tertentu; jadwal reguler kembali berlaku setelah sesi selesai; perubahan permanen diperlihatkan sebagai perubahan jadwal rutin. | BR-SCH-001 |
| UR-SCH-004 | PJ membutuhkan tempat menyimpan perubahan tentatif sebelum dipublikasikan. | PJ | Giza | Must | Perubahan dapat disimpan sebagai draf; draf tidak terlihat oleh mahasiswa dan tidak memicu notifikasi; PJ dapat melanjutkan atau menghapus draf. | BR-SCH-002 |
| UR-SCH-005 | PJ membutuhkan preview untuk memeriksa perubahan sebelum publikasi. | PJ | Iman, Arsel, Faqih, Hana, Irfan | Must | Preview menampilkan mata kuliah, jadwal lama, jadwal baru, jenis perubahan, dan penerima; PJ dapat kembali memperbaiki data sebelum memublikasikan. | BR-SCH-002 |
| UR-SCH-006 | PJ membutuhkan kemampuan memublikasikan perubahan untuk mata kuliahnya tanpa menunggu persetujuan KM. | PJ | Keputusan OQ-005 | Must | PJ dapat memublikasikan perubahan dalam cakupannya; publikasi tercatat; perubahan langsung tampil di portal kelas dan memicu siaran yang sesuai. | BR-SCH-004, BR-AUDIT-001 |
| UR-SCH-007 | KM membutuhkan kemampuan membatalkan perubahan jadwal yang keliru pada kelasnya. | KM | Keputusan OQ-005 | Must | KM dapat membatalkan perubahan yang masih berlaku; riwayat publikasi tidak dihapus; mahasiswa menerima koreksi jika informasi sebelumnya sudah disiarkan. | BR-SCH-005, BR-SCH-006 |
| UR-SCH-008 | Pengguna membutuhkan tampilan yang membedakan jadwal reguler, pengganti, tambahan, libur, dan dibatalkan. | Semua aktor | Anindya, Kemal, Imam KM | Must | Setiap jenis perubahan memiliki label dan penjelasan yang jelas; tampilan tidak hanya mengandalkan warna; jadwal lama dan baru tidak tertukar. | BR-SCH-001 |

## 7. Tugas dan Deadline

| ID | User Requirement | Aktor | Sumber | Prioritas | Kriteria Keberhasilan | Aturan Terkait |
|---|---|---|---|---|---|---|
| UR-TASK-001 | PJ membutuhkan cara menambahkan tugas tanpa menghafal format command. | PJ | Iman, Faqih, Hana, Giza, Kemal | Must | Form dapat digunakan dari ponsel; PJ hanya dapat memilih mata kuliah dalam cakupannya; tugas dapat disimpan atau dipublikasikan tanpa menulis sintaks command. | BR-ACCESS-002, BR-TASK-001 |
| UR-TASK-002 | Mahasiswa membutuhkan informasi tugas yang lengkap dan tidak tersebar di banyak platform. | Mahasiswa | Seluruh responden | Must | Tugas memuat mata kuliah, judul, instruksi, deadline, serta tempat atau tautan pengumpulan; detail dapat dibuka dari satu portal kelas. | BR-TASK-001 |
| UR-TASK-003 | Mahasiswa membutuhkan pengelompokan tugas berdasarkan tingkat urgensi. | Mahasiswa | Bima, Faqih, Afzhal, Hana | Must | Tugas dapat dilihat dalam kelompok Hari Ini, Minggu Ini, Mendatang, dan Terlewat; urutan deadline terdekat tersedia. | BR-TASK-002 |
| UR-TASK-004 | Pengguna membutuhkan pencarian dan penyaringan tugas berdasarkan konteks yang relevan. | Mahasiswa, PJ, KM | Iman, Irfan, Afzhal, Hana, Anindya | Should | Pengguna dapat menyaring berdasarkan mata kuliah, status, rentang deadline, dan jenis tugas tanpa kehilangan konteks kelas aktif. | BR-CLASS-001, BR-SEM-001 |
| UR-TASK-005 | PJ membutuhkan kemampuan memublikasikan tugas pada mata kuliahnya dengan alur approval yang dapat diatur per kelas. | PJ, KM | Faqih dan keputusan OQ-003 | Must | Publikasi langsung menjadi perilaku default; KM dapat mengaktifkan approval tugas untuk kelasnya; status tugas menunjukkan apakah masih draf, menunggu, atau sudah terbit. | BR-TASK-003 |
| UR-TASK-006 | Pengurus membutuhkan cara menyelesaikan atau mengarsipkan tugas tanpa menghilangkan riwayat. | PJ, KM | Afzhal, Irfan | Should | Tugas dapat ditandai selesai; tugas terlewat atau selesai berpindah dari daftar aktif ke arsip; data tetap dapat ditelusuri. | BR-TASK-002, BR-AUDIT-001 |
| UR-TASK-007 | Pengguna membutuhkan tempat terpusat untuk tautan materi dan referensi per mata kuliah. | Mahasiswa, PJ | Afzhal, Anindya | Should | Materi dan tautan dapat dikelompokkan berdasarkan mata kuliah; tautan dapat dibuka dari detail tugas atau halaman materi; data tidak memenuhi pesan WhatsApp. | BR-CLASS-001, BR-SEM-001 |

## 8. Notifikasi WhatsApp

| ID | User Requirement | Aktor | Sumber | Prioritas | Kriteria Keberhasilan | Aturan Terkait |
|---|---|---|---|---|---|---|
| UR-NOTIF-001 | Mahasiswa membutuhkan pengingat jadwal otomatis pada pagi hari. | Mahasiswa | Arsel, Afzhal, Anindya, Irfan | Must | Pesan memuat jadwal hari itu, jam, dosen, dan ruangan; jadwal pengganti yang berlaku ikut ditampilkan. | BR-SCH-001 |
| UR-NOTIF-002 | Mahasiswa membutuhkan pengingat tugas pada sore hari setelah perkuliahan. | Mahasiswa | Bima, Hana | Must | Pesan menyoroti tugas aktif berdasarkan deadline; waktu pengiriman dapat dikonfigurasi untuk kelas; tugas terlewat dibedakan dengan jelas. | BR-TASK-002 |
| UR-NOTIF-003 | Mahasiswa membutuhkan notifikasi perubahan jadwal yang menunjukkan informasi lama dan baru. | Mahasiswa | Kemal | Must | Pesan membandingkan jadwal sebelum dan sesudah; pembatalan menghasilkan pesan koreksi; pesan menyebut waktu perubahan berlaku. | BR-SCH-006 |
| UR-NOTIF-004 | Pengguna membutuhkan pesan WhatsApp yang ringkas dengan tautan menuju detail web. | Semua aktor | Faqih, Hana, Bima | Must | Pesan hanya memuat informasi utama; rincian panjang tersedia melalui tautan kelas; pesan tidak mengharuskan mahasiswa menjalankan command. | Product Definition bagian 12 |
| UR-NOTIF-005 | Pengelola membutuhkan jaminan bahwa gangguan bot tidak menggandakan atau menghilangkan notifikasi. | PJ, KM, System Admin | Product Definition bagian 13 | Must | Publikasi tetap tersimpan ketika bot offline; notifikasi dapat dikirim setelah koneksi pulih; satu publikasi tidak menghasilkan siaran ganda. | BR-SCH-003, BR-AUDIT-001 |

## 9. Ruangan

| ID | User Requirement | Aktor | Sumber | Prioritas | Kriteria Keberhasilan | Aturan Terkait |
|---|---|---|---|---|---|---|
| UR-ROOM-001 | PJ dan KM membutuhkan bantuan menemukan kandidat ruangan kosong untuk kelas pengganti. | PJ, KM | Arsel, Irfan, Faqih, Anindya, Kemal, Imam KM | Should | Pengguna dapat memilih tanggal dan rentang waktu lalu melihat ruangan yang tidak tercatat sedang dipakai; hasil memperhitungkan jadwal kelas yang tersedia di sistem. | BR-ROOM-001 |
| UR-ROOM-002 | Pengguna membutuhkan kejelasan bahwa rekomendasi ruangan belum menggantikan konfirmasi resmi. | PJ, KM | Imam KM, Giza, Kemal | Must | Hasil diberi label sebagai rekomendasi data internal; antarmuka mengingatkan pengguna untuk mengonfirmasi ke TU; pengguna dapat mencatat hasil konfirmasi. | BR-ROOM-001 |
| UR-ROOM-003 | System Admin membutuhkan satu tempat untuk memelihara data ruangan lintas kelas. | System Admin | Keputusan OQ-010 | Should | System Admin dapat menambah, memperbarui, dan menonaktifkan ruangan; KM dapat memberikan masukan tanpa mengubah master data secara langsung. | BR-ROOM-002 |

## 10. Audit dan Akuntabilitas

| ID | User Requirement | Aktor | Sumber | Prioritas | Kriteria Keberhasilan | Aturan Terkait |
|---|---|---|---|---|---|---|
| UR-AUDIT-001 | Pengurus membutuhkan riwayat untuk mengetahui siapa yang mengubah data dan apa yang berubah. | PJ, KM | Arsel, Hana, Anindya, Kemal, Iman | Must | Riwayat menampilkan pelaku, waktu, jenis tindakan, data sebelum, dan data sesudah; cakupan tampilan mengikuti hak akses pengguna. | BR-AUDIT-001 |
| UR-AUDIT-002 | KM membutuhkan kemampuan menelusuri publikasi dan pembatalan tanpa kehilangan bukti perubahan sebelumnya. | KM | Keputusan OQ-005 | Must | Pembatalan membuat catatan baru dan tidak menghapus publikasi lama; hubungan antara publikasi dan koreksi dapat dilihat. | BR-SCH-005, BR-AUDIT-001 |
| UR-AUDIT-003 | Pengelola membutuhkan riwayat yang tetap tersedia selama masa operasional kelas. | KM, System Admin | Keputusan OQ-008 | Must | Audit log dan arsip tersedia selama kelas aktif ditambah sekurangnya satu tahun akademik; kebijakan penghapusan tidak berjalan tanpa otorisasi. | BR-AUDIT-002 |

## 11. Pengalaman Pengguna dan Aksesibilitas

| ID | User Requirement | Aktor | Sumber | Prioritas | Kriteria Keberhasilan | Aturan Terkait |
|---|---|---|---|---|---|---|
| UR-UX-001 | PJ membutuhkan alur cepat yang dapat digunakan dengan satu tangan dari ponsel. | PJ | Seluruh responden dan PRD | Must | Operasi umum memiliki target sentuh yang memadai; form tidak membutuhkan zoom horizontal; penambahan tugas normal dapat selesai dalam kurang dari 30 detik. | Product Definition prinsip produk |
| UR-UX-002 | Pengguna membutuhkan status yang jelas ketika data sedang dimuat, kosong, gagal, tersimpan, atau menunggu publikasi. | Semua aktor | DASHBOARD_PRD dan Product Definition | Must | Setiap halaman memiliki loading, empty, error, success, dan permission-denied state yang dapat dipahami; kegagalan tidak menghilangkan input pengguna tanpa peringatan. | Product Definition bagian 4 |
| UR-UX-003 | Pengurus baru membutuhkan panduan awal agar dapat menggunakan dashboard tanpa pelatihan teknis. | PJ, KM | Imam KM | Should | Pengguna baru melihat penjelasan peran, cakupan akses, dan tindakan utama; panduan dapat dibuka kembali; pengguna tidak diwajibkan menghafal command. | Product Definition bagian 8 |
| UR-UX-004 | Pengguna membutuhkan tampilan yang dapat dipahami tanpa mengandalkan warna saja. | Semua aktor | Kebutuhan aksesibilitas dan DESIGN.md | Must | Status memakai teks atau ikon selain warna; fokus keyboard terlihat; kontrol penting memiliki label yang jelas; navigasi dapat digunakan pada perangkat mobile dan desktop. | Product Definition prinsip produk |

## 12. Operasional dan Pemulihan

| ID | User Requirement | Aktor | Sumber | Prioritas | Kriteria Keberhasilan | Aturan Terkait |
|---|---|---|---|---|---|---|
| UR-OPS-001 | Pengelola membutuhkan data kelas tetap aman ketika nomor atau sesi WhatsApp berubah. | KM, System Admin | Arsitektur dan implementasi saat ini | Must | Pergantian sesi bot tidak menghapus jadwal, tugas, pengurus, atau riwayat; koneksi baru dapat memakai kembali data kelas. | Product Definition bagian 13 |
| UR-OPS-002 | System Admin membutuhkan kemampuan mencadangkan dan memulihkan data per kelas serta semester. | System Admin | Product Definition bagian 13 | Must | Data dapat diekspor dan dipulihkan dengan cakupan yang jelas; proses pemulihan tidak mencampur kelas; hasil pemulihan dapat diverifikasi. | BR-CLASS-001, BR-SEM-001 |
| UR-OPS-003 | Pengurus membutuhkan perlindungan terhadap penghapusan data yang tidak disengaja. | PJ, KM | Product Definition bagian 13 | Must | Data penting dapat dipulihkan sesuai kewenangan; penghapusan dan pemulihan tercatat; data tidak langsung hilang permanen dari tindakan biasa. | BR-AUDIT-001 |
| UR-OPS-004 | Pengguna membutuhkan perlindungan ketika dua pengurus memperbarui data yang sama. | PJ, KM | Product Definition bagian 13 | Should | Pengguna diberi tahu jika data telah berubah sejak dibuka; pengguna dapat melihat versi terbaru sebelum menimpa; perubahan yang ditolak tidak hilang tanpa penjelasan. | BR-AUDIT-001 |

## 13. Kebutuhan yang Ditunda

| ID | Kebutuhan | Alasan Penundaan | Kondisi Peninjauan Ulang |
|---|---|---|---|
| DEF-001 | Mahasiswa menandai tugas selesai secara pribadi | Portal mahasiswa belum menggunakan akun individual. | Ditinjau ketika autentikasi mahasiswa direncanakan. |
| DEF-002 | Drag and drop kalender | Kompleksitas tinggi dan bukan kebutuhan utama untuk input cepat di ponsel. | Ditinjau setelah form dan kalender dasar tervalidasi. |
| DEF-003 | Auto-pin pesan WhatsApp | Bergantung pada kemampuan dan risiko integrasi WhatsApp. | Ditinjau jika dukungan resmi dan aman tersedia. |
| DEF-004 | Integrasi ruangan langsung dengan TU | Membutuhkan izin dan sumber data eksternal yang belum tersedia. | Ditinjau ketika kampus menyediakan akses data resmi. |
| DEF-005 | Agenda organisasi dan pembagian kelompok | Di luar fokus jadwal dan tugas akademik. | Ditinjau melalui produk atau modul terpisah. |

## 14. Ketertelusuran

Requirement pada dokumen ini harus dipetakan ke artefak berikut ketika dibuat:

```text
Wawancara atau Keputusan Produk
              ↓
       User Requirement
              ↓
      Functional Requirement
              ↓
        User Flow dan UI
              ↓
        API dan Data Model
              ↓
           Test Case
```

Requirement dianggap siap diimplementasikan ketika memiliki aturan bisnis yang jelas, alur pengguna, acceptance criteria teknis, desain state, dan test case. Perubahan requirement tidak menghapus ID lama. Requirement yang tidak lagi berlaku diberi status `Deprecated` dan menunjuk ke penggantinya.
