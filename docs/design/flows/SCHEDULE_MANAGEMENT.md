# Paket Desain Manajemen Jadwal

## Status Dokumen

| Atribut | Nilai |
|---|---|
| Versi | 1.1.0 |
| Status | Approved untuk eksplorasi UI/UX |
| Pemilik | Tim Bot Jadwal |
| Terakhir diperbarui | 24 September 2026 |
| Panduan global | [Panduan Desain pen.dev](../PENCIL_CONTEXT.md) |
| Inventaris | [Inventaris Layar](../SCREEN_INVENTORY.md) |
| Data contoh | [Data Contoh untuk Prototype](../PROTOTYPE_DATA.md) |
| Sumber kanonis | FR-SCH-001 sampai FR-SCH-008; BR-SCH-001 sampai BR-SCH-009; UF-SCH-001 sampai UF-SCH-005 |

Paket ini mengarahkan agent pen.dev untuk membuat desain alur jadwal reguler dan perubahan jadwal. Jika paket ini tidak menetapkan perilaku tertentu, ikuti dokumen sumber yang ditautkan dan jangan menebak keputusan produk.

## 1. Tujuan Vertical Slice

Desain harus mencakup:

1. PJ atau KM mengelola jadwal reguler melalui formulir.
2. PJ menyimpan draf perubahan, memeriksa pratinjau dan konflik, lalu dapat menerbitkan tanpa menunggu persetujuan KM.
3. KM dapat mencabut publikasi yang keliru, melihat jadwal yang kembali berlaku, dan mengirim koreksi.
4. Satu perubahan dapat melibatkan kelas lain setelah KM kelas peserta menerima undangan.
5. Mahasiswa melihat jadwal efektif dengan jenis sesi yang jelas.

Pola jadwal reguler berada pada `schedule_patterns`. Perubahan aktual berada pada `teaching_events`. Nama tersebut hanya dipakai dalam anotasi data. Antarmuka memakai istilah `Perubahan Jadwal`. Perubahan sementara tidak mengubah pola reguler. Perubahan permanen membuat versi pola baru sejak tanggal berlaku.

## 2. Peran dan Izin

| Aksi | Mahasiswa | PJ | KM | System Admin |
|---|:---:|:---:|:---:|:---:|
| Melihat jadwal terbit | Ya | Ya | Ya | Dukungan |
| Membuat atau mengubah pola reguler | Tidak | Mata kuliah sendiri | Seluruh kelas | Dukungan |
| Membuat dan menyimpan draf perubahan jadwal | Tidak | Mata kuliah sendiri | Seluruh kelas | Dukungan |
| Menerbitkan perubahan jadwal | Tidak | Mata kuliah sendiri | Seluruh kelas | Dukungan |
| Mencabut perubahan yang sudah terbit | Tidak | Tidak | Kelasnya | Dukungan |
| Mengelola partisipasi kelas peserta | Tidak | Tidak | Kelas peserta | Dukungan |

PJ hanya melihat mata kuliah pada penugasannya. KM kelas peserta boleh menerima, menolak, melepas, atau meminta undangan ulang untuk partisipasi kelasnya. KM peserta tidak mengubah waktu, ruangan, jenis, alasan, atau status publikasi utama. System Admin memakai fungsi dukungan dengan alasan dan audit.

## 3. Jenis dan Status Jadwal

### Jenis data

- `schedule_patterns`: jadwal reguler yang berulang pada rentang tanggal efektif.
- `teaching_events`: perubahan aktual yang mengganti, menambah, atau membatalkan satu sesi, serta hari libur. Label antarmukanya adalah `Perubahan Jadwal`.
- Perubahan permanen menutup versi pola lama dan membuat versi baru. Riwayat versi tetap tersedia.

### Jenis perubahan jadwal

| Nilai data | Label antarmuka | Makna |
|---|---|---|
| `REPLACEMENT` | Kelas Pengganti | Mengganti sesi reguler tertentu |
| `EXTRA` | Kelas Tambahan | Menambah sesi; boleh tanpa pola asal |
| `HOLIDAY` | Hari Libur | Menandai libur akademik sesuai keputusan kelas |
| `SESSION_CANCELLED` | Sesi Dibatalkan | Sesi akademik tidak berlangsung |

Label tersebut harus memakai teks atau ikon, tidak hanya warna.

### Lifecycle publikasi

- `DRAFT`: hanya terlihat oleh pengurus berwenang, tidak muncul di portal dan tidak memicu notifikasi.
- `PUBLISHED`: tampil dalam jadwal efektif dan dapat memicu notifikasi.
- `REVOKED`: publikasi ditarik oleh KM, tetapi riwayat tetap tersedia.

`SESSION_CANCELLED` adalah jenis sesi akademik, bukan status publikasi. Sesi yang dibatalkan dapat tetap `PUBLISHED` agar mahasiswa melihat pembatalannya.

### Partisipasi lintas kelas

`PENDING` dapat menjadi `ACCEPTED`, `DECLINED`, atau `REMOVED`. `ACCEPTED` dapat menjadi `REMOVED`. Undangan ulang mengubah `DECLINED` atau `REMOVED` menjadi `PENDING` dan dicatat. Perubahan hanya tampil di kelas peserta setelah partisipasi `ACCEPTED`.

## 4. Layar yang Harus Dibuat

Gunakan ID dan nama dari inventaris. Buat frame desktop dan mobile untuk setiap layar utama.

### SCR-SCH-001 Pola Jadwal Reguler

- Tampilkan pola berdasarkan hari dengan filter mata kuliah, dosen, dan ruangan.
- Tampilkan mata kuliah, jenis, hari, jam, dosen, ruangan, dan tanggal efektif.
- Sediakan `Tambah Jadwal`, `Ubah`, dan `Ubah Jadwal Permanen` sesuai izin.
- Pada PJ, mata kuliah mengikuti pilihan kelas dan peran atau hanya menawarkan mata kuliah penugasannya.
- Empty state menjelaskan jadwal reguler belum tersedia dan memberi aksi hanya kepada pengurus berwenang.

### SCR-SCH-002 Form Perubahan Jadwal

Gunakan satu halaman dengan bagian `Jenis Perubahan`, `Jadwal Semula`, `Jadwal Baru`, dan `Keterangan`. Tampilkan bagian `Jadwal Semula` hanya untuk kelas pengganti atau sesi yang dibatalkan. Field yang tidak berlaku untuk jenis pilihan pengguna tidak ditampilkan.

Form memuat:

- kelas serta semester yang sedang dikelola;
- mata kuliah dan satu mata kuliah penyelenggara (`OWNER` pada data);
- jenis perubahan;
- pola asal dan tanggal kejadian asal untuk `REPLACEMENT` atau `SESSION_CANCELLED`;
- tanggal serta jam mulai dan selesai;
- ruangan dan keterangan;
- pilihan `Simpan sebagai Draf` dan `Tinjau Perubahan`.

Kelas tambahan (`EXTRA` pada data) boleh tidak memiliki pola asal. Waktu input memakai zona waktu kelas pemilik. Perubahan harus berada dalam rentang semester kelas pemilik. Form perubahan permanen meminta tanggal mulai berlaku dan menjelaskan sesi mana yang memakai versi lama atau baru.

### SCR-SCH-003 Pratinjau dan Konflik Jadwal

- Bandingkan jadwal lama dan jadwal baru jika perubahan mengganti atau mengubah sesi.
- Untuk kelas tambahan, tampilkan ringkasan sesi baru tanpa membuat jadwal lama seolah-olah diganti.
- Tampilkan jenis perubahan, waktu berlaku, kelas penerima, dan mata kuliah kelas peserta.
- Tampilkan konflik beserta data yang berbenturan.
- Konflik pemblokir menonaktifkan publikasi dan menjelaskan penyebabnya.
- Konflik nonpemblokir meminta konfirmasi dan alasan. Simpan sebagai `conflict_override_reason`.
- Jika ruangan masih kandidat, jelaskan bahwa PJ atau KM perlu mencatat konfirmasi manual TU bila ruangan diwajibkan.
- Sediakan `Ubah Lagi` dan `Terbitkan Perubahan`.

### SCR-SCH-004 Detail Perubahan Jadwal

Gunakan bagian `Detail`, `Riwayat`, dan `Notifikasi` jika sesuai dengan hierarki konten. Tampilkan status publikasi, jenis perubahan, pelaku publikasi, waktu, alasan, pola asal, kelas pemilik, partisipasi, dan perubahan versi.

- PJ melihat aksi edit atau koreksi sesuai cakupannya, tetapi tidak melihat `Cabut Publikasi`.
- KM kelas pemilik melihat `Cabut Publikasi` pada perubahan berstatus `PUBLISHED`.
- Draf dapat dilanjutkan atau dihapus oleh pengurus berwenang.
- Status publikasi dan status WhatsApp harus terpisah.

### SCR-SCH-005 Partisipasi Lintas Kelas

- KM kelas pemilik memilih satu mata kuliah sebagai penyelenggara (`OWNER` pada data), lalu menambahkan mata kuliah dari kelas lain sebagai peserta.
- Sistem membuat status `PENDING` untuk peserta.
- KM peserta melihat kelas pemilik, mata kuliah, waktu, ruangan, jenis, dan semester perubahan.
- KM peserta dapat `Terima`, `Tolak`, atau `Lepas Kelas` sesuai status.
- KM pemilik dapat mengundang ulang setelah `DECLINED` atau `REMOVED`.
- Jelaskan bahwa peserta tidak dapat mengubah perubahan utama.
- Jangan menampilkan perubahan di portal kelas peserta sebelum status `ACCEPTED`.

### SCR-PORTAL-003 Jadwal Mahasiswa

- Tampilkan jadwal harian dan mingguan dengan jadwal efektif.
- Setiap item menampilkan mata kuliah, jenis, jam, dosen, dan ruangan jika tersedia.
- Bedakan reguler, pengganti, tambahan, hari libur, dan sesi dibatalkan melalui label yang mudah dipahami.
- Perubahan terbaru dapat membuka detail sesi.
- Hanya tampilkan perubahan yang `PUBLISHED` dan, untuk kelas peserta, partisipasinya `ACCEPTED`.

### SCR-ROOM-001 Kandidat dan Konfirmasi Ruangan

Layar ini prioritas lanjutan, tetapi sediakan pola pendukung jika dibutuhkan oleh alur perubahan jadwal:

- Hasil pencarian ruangan adalah kandidat berdasarkan data internal.
- Tampilkan waktu pembaruan data dan keterangan bahwa pengguna perlu mengonfirmasi ke TU.
- Pengurus membuat draf perubahan jadwal sebelum menghubungi TU.
- Catatan manual memuat status, ruangan, nama petugas atau sumber, waktu pencatatan, dan catatan.
- Hasil hanya dianggap terkonfirmasi jika status `CONFIRMED`.

## 5. Alur yang Harus Digambarkan

### A. Jadwal reguler manual: UF-SCH-001

```text
Pola Jadwal -> Tambah/Ubah -> Pilih mata kuliah dan dosen -> Isi hari, jam, durasi, ruangan
-> Sistem hitung jam selesai dan cek konflik -> Tinjau -> Simpan -> Riwayat Perubahan
```

### B. Kelas pengganti atau tambahan oleh PJ: UF-SCH-002

```text
Pilih pola asal (jika ada) -> Buat Perubahan -> Pilih jenis dan waktu
-> Cek konflik -> Tinjau -> Simpan sebagai Draf atau Terbitkan
-> Portal diperbarui -> Notifikasi dikirim atau masuk antrean
```

Publikasi PJ tidak menunggu persetujuan KM. Kegagalan WhatsApp tidak membatalkan publikasi web. Percobaan ulang dengan identitas publikasi yang sama tidak boleh membuat data atau pesan ganda.

### C. Perubahan permanen: UF-SCH-003

```text
Pilih pola -> Ubah permanen -> Tentukan data dan tanggal mulai
-> Tinjau dampak sesi -> Selesaikan konflik perubahan sementara
-> Tinjau -> Terbitkan -> Tutup versi lama dan aktifkan versi baru
```

Sesi sebelum tanggal berlaku tetap memakai versi lama. Tanggal berlaku harus berada pada semester aktif.

### D. Pencabutan oleh KM: UF-SCH-004

```text
Detail perubahan terbit -> Cabut Publikasi -> Lihat jadwal yang akan berlaku kembali
-> Isi alasan -> Konfirmasi -> Lifecycle REVOKED -> Portal dan notifikasi koreksi diperbarui
```

Pencabutan kedua ditolak. Jika jadwal dasar berubah sejak publikasi, KM harus memeriksa versi terbaru. Koreksi yang sudah disiarkan menghubungkan pesan lama, pencabutan, dan jadwal yang berlaku kembali.

### E. Perkuliahan lintas kelas: UF-SCH-005

```text
KM pemilik membuat perubahan dan memilih mata kuliah penyelenggara
-> Tambah mata kuliah kelas lain sebagai peserta (PENDING)
-> KM peserta meninjau -> Terima atau Tolak
-> Perubahan tampil di portal peserta setelah ACCEPTED
```

Tolak atau lepas kelas tidak mengubah perubahan utama. Undangan ulang menghasilkan `PENDING` dan masuk audit.

## 6. Keadaan Layar yang Wajib Dibuat

| State | Perilaku desain |
|---|---|
| Jadwal belum dibuat | Jelaskan jadwal belum dibuat dan beri aksi sesuai izin |
| Filter tanpa hasil | Tampilkan filter aktif dan `Hapus Semua Filter` |
| Memuat | Pertahankan kelas, semester, peran, dan mata kuliah yang sedang dikelola |
| Gagal memuat | Jelaskan data yang gagal dimuat dan sediakan `Coba Lagi` |
| Validation | Pertahankan input dan tandai field bermasalah |
| Conflict blocking | Cegah simpan atau publikasi dan jelaskan benturan |
| Conflict override | Minta konfirmasi serta alasan sebelum lanjut |
| Draf | Terlihat bagi pengurus berwenang, tidak untuk mahasiswa |
| Terbit, WhatsApp tertunda | Jadwal sudah terbit di web; pesan masih mengantre |
| Publikasi dicabut | Riwayat tersedia dan portal menunjukkan jadwal yang berlaku |
| Menunggu jawaban kelas peserta | Hanya kelas pemilik melihat perubahan; peserta belum melihat di portal |
| Diterima kelas peserta | Perubahan tampil pada kelas peserta |
| Ditolak atau kelas dilepas | Perubahan tersembunyi untuk kelas tersebut; tawarkan undangan ulang kepada pemilik |
| Perlu konfirmasi TU | Sebut kandidat dan perlunya konfirmasi manual TU |
| Sesi berakhir | Pertahankan input dan minta login kembali |
| Tidak memiliki akses | Jangan membuka data; tawarkan `Kembali` atau `Pindah kelas atau peran` |
| Ada perubahan yang lebih baru | Tampilkan versi terbaru tanpa membuang input pengguna |

## 7. Arahan Visual dan Responsif

- Ikuti [Panduan Desain pen.dev](../PENCIL_CONTEXT.md), termasuk Design Tokens v2.0 yang sudah disetujui. Jangan kembali ke arahan grayscale lama.
- Tampilkan jadwal sebagai garis waktu atau daftar hari hanya jika hubungan waktu lebih mudah dipindai. Hindari tabel padat pada mobile.
- Buat desktop dan mobile. Di mobile, detail sesi dapat dibuka dari item jadwal tanpa menyembunyikan waktu, mata kuliah, dan statusnya.
- Gunakan label teks dan ikon untuk jenis serta status. Warna tidak boleh menjadi satu-satunya pembeda.
- Kontrol sentuh minimal 44 x 44 piksel. Dialog dapat ditutup dengan Escape dan mengembalikan fokus.
- Jangan menambahkan logo, nama institusi, dosen, atau data nyata yang tidak disediakan.

## 8. Prompt Awal pen.dev

```text
Read the attached files in this order:
1. PENCIL_CONTEXT.md for global product, access, terminology, and visual constraints.
2. SCHEDULE_MANAGEMENT.md for the authoritative schedule scope.
3. PROTOTYPE_DATA.md for populated examples and state copy.
4. SCREEN_INVENTORY.md only to confirm screen IDs and names.

Create the Schedule Management vertical slice as an editable responsive web UI exploration v0.1. Design only the schedule screens specified in SCHEDULE_MANAGEMENT.md. Do not create task, account, member, semester, or system admin screens unless they are shown as context inside the schedule app shell.

Generate desktop and mobile frames for SCR-SCH-001, SCR-SCH-002, SCR-SCH-003, SCR-SCH-004, SCR-SCH-005, and SCR-PORTAL-003. Add SCR-ROOM-001 only as a supporting flow for room candidate and manual TU confirmation.

Show these flows: manual regular schedule management; PJ saves or publishes a replacement or extra event; permanent schedule change with effective date; KM revokes a published event and sees the schedule that becomes effective; and cross-class participation from PENDING to ACCEPTED or DECLINED.

Follow the role permissions, event kinds, publication lifecycle, participant rules, conflict handling, semester bounds, and time-zone rules exactly. PJ may publish without KM approval. SESSION_CANCELLED is an academic event kind, not a publication status. Keep web publication status separate from WhatsApp delivery status. Students only see PUBLISHED events, and cross-class events only after ACCEPTED.

Use the approved Design Tokens v2.0 in PENCIL_CONTEXT.md. Keep status meaning clear with text labels and icons, not color alone. Do not introduce colors outside the approved tokens.

Use Indonesian interface copy and the fictional dataset in docs/design/PROTOTYPE_DATA.md. Use populated examples as the main frames and generate empty states as separate variants. Do not invent features, approval steps, roles, brand assets, institution names, real people, or statistics.

Do not use the visible labels Konteks, Ganti Konteks, Teaching Event, Offering, Review, Reset Filter, Access Denied, or Conflict Edit. Use the plain-language interface terms defined in PENCIL_CONTEXT.md.

Organize frames by actor and flow. Name every frame with its SCR ID, actor, breakpoint, and state. Reuse components for schedule items, change forms, conflict notices, status labels, room confirmation, participation, and history. Do not generate production code.
```

## 9. Data untuk Prototype

Gunakan bagian jadwal pada [Data Contoh untuk Prototype](../PROTOTYPE_DATA.md). Dataset tersebut menyediakan jadwal reguler, perubahan yang sudah terbit, draf kelas tambahan, konflik ruangan, kandidat ruangan, dan pesan keadaan layar. Pakai keadaan berisi sebagai frame utama dan buat keadaan kosong sebagai variasi terpisah.

Jangan menyimpulkan jadwal aktual dari file JSON repository kecuali pengguna secara khusus meminta jadwal tersebut dipakai sebagai fixture desain.

## 10. Checklist Pemeriksaan

- [ ] Pola reguler dan event aktual ditampilkan sebagai konsep berbeda.
- [ ] Perubahan sementara tidak tampak mengubah jadwal reguler permanen.
- [ ] Perubahan permanen menjelaskan tanggal efektif dan versi pola.
- [ ] Draf tidak tampil di portal dan tidak membuat notifikasi.
- [ ] PJ dapat menerbitkan tanpa persetujuan KM.
- [ ] Hanya KM yang dapat mencabut event terbit.
- [ ] `SESSION_CANCELLED` berbeda dari `REVOKED`.
- [ ] Konflik pemblokir dan nonpemblokir memiliki hasil interaksi berbeda.
- [ ] Lintas kelas menggunakan satu mata kuliah penyelenggara (`OWNER` pada data) dan persetujuan KM peserta.
- [ ] Status WhatsApp tidak disamakan dengan lifecycle publikasi.
- [ ] Mobile dan desktop tersedia dengan informasi penting yang sama.
- [ ] Seluruh frame memakai Design Tokens v2.0 dan diberi label eksplorasi.
- [ ] Frame utama memakai data contoh berisi dan state kosong tersedia sebagai variasi terpisah.
- [ ] Istilah teknis hanya muncul dalam anotasi, bukan sebagai label antarmuka.

## 11. Referensi Kanonis

- Kebutuhan pengguna: UR-SCH-001 sampai UR-SCH-009.
- Functional requirements: FR-SCH-001 sampai FR-SCH-008.
- Business rules: BR-SCH-001 sampai BR-SCH-009, BR-ROOM-001 sampai BR-ROOM-003, dan BR-NOTIF-001 sampai BR-NOTIF-005.
- User flows: UF-SCH-001 sampai UF-SCH-005.
- Data model: bagian jadwal dan ruangan pada `docs/product/DATA_MODEL.md`.

## 12. Changelog

### 1.1.0, 24 September 2026

- Mengganti istilah teknis dengan copy perubahan jadwal yang lebih mudah dipahami.
- Menghapus arahan grayscale yang bertentangan dengan Design Tokens v2.0.
- Menautkan kumpulan data contoh bersama dan meminta frame utama berisi data.

### 1.0.0, 24 September 2026

- Membuat paket desain jadwal reguler, perubahan jadwal, konflik, pencabutan, lintas kelas, dan konfirmasi ruangan.
- Menambahkan prompt pen.dev yang membatasi cakupan serta memakai grayscale sampai palet disetujui.
