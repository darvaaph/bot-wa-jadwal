# Paket Desain Manajemen Tugas

## Status Dokumen

| Atribut | Nilai |
|---|---|
| Versi | 1.1.0 |
| Status | Approved untuk eksplorasi UI/UX |
| Pemilik | Tim Bot Jadwal |
| Terakhir diperbarui | 23 September 2026 |
| Panduan global | [Panduan Desain pen.dev](../PENCIL_CONTEXT.md) |
| Inventaris | [Inventaris Layar](../SCREEN_INVENTORY.md) |
| Data contoh | [Data Contoh untuk Prototype](../PROTOTYPE_DATA.md) |
| Sumber kanonis | FR-TASK-001 sampai FR-TASK-006; BR-TASK-001 sampai BR-TASK-006; UF-TASK-001 sampai UF-TASK-003 |

Paket ini memberi agent pen.dev seluruh panduan yang diperlukan untuk mendesain siklus tugas dari input PJ sampai konsumsi mahasiswa. Jangan membaca dokumen lain kecuali ada keputusan yang tidak dapat ditemukan di paket ini.

## 1. Tujuan Vertical Slice

Desain harus menunjukkan satu tugas yang:

1. dibuat dan ditinjau oleh PJ sebelum terbit;
2. langsung dipublikasikan ke portal mahasiswa;
3. masuk antrean pemeriksaan KM tanpa menghambat publikasi;
4. dapat disetujui, ditarik untuk koreksi, atau dibatalkan oleh KM;
5. dapat diubah, diselesaikan, dan diarsipkan tanpa kehilangan riwayat.

## 2. Peran dan Izin

| Aksi | Mahasiswa | PJ | KM |
|---|:---:|:---:|:---:|
| Melihat tugas terbit | Ya | Ya | Ya |
| Membuat dan mengubah draf | Tidak | Mata kuliah sendiri | Seluruh kelas |
| Memublikasikan | Tidak | Mata kuliah sendiri | Seluruh kelas |
| Memeriksa setelah publikasi | Tidak | Tidak | Ya |
| Meminta koreksi atau membatalkan | Tidak | Tidak | Ya |
| Menandai selesai dan mengarsipkan | Tidak | Mata kuliah sendiri | Seluruh kelas |
| Melihat riwayat | Tidak | Cakupan sendiri | Seluruh kelas |

PJ tidak melihat tombol pemeriksaan KM. Mahasiswa tidak melihat kontrol perubahan, status pemeriksaan internal, audit, atau status pengiriman WhatsApp.

## 3. Model Status

- `publication_status`: `DRAFT`, `PUBLISHED`, atau `REVOKED`.
- `review_state`: `NOT_REVIEWED`, `APPROVED`, `CHANGES_REQUESTED`, atau `REVOKED`.
- `Selesai` berasal dari `completed_at`.
- `Terlewat` dihitung ketika tugas belum selesai dan deadline telah lewat.
- `Arsip` berasal dari `archived_at` dan tidak mengganti status publikasi atau penyelesaian.
- Pemeriksaan KM hanya berlaku untuk versi tugas yang sama. Perubahan oleh PJ membuat versi baru `NOT_REVIEWED`; perubahan oleh KM membuat versi baru langsung disetujui.

Antarmuka tidak boleh menggabungkan seluruh dimensi tersebut menjadi satu badge yang ambigu.

## 4. Field Tugas

| Field | Ketentuan desain |
|---|---|
| Mata kuliah | Terisi dari mata kuliah yang sedang dikelola PJ atau dapat dipilih KM dari kelas aktif |
| Judul | Wajib sebelum publikasi |
| Instruksi | Wajib sebelum publikasi; dukung teks panjang yang mudah dibaca |
| Deadline | Wajib; tampilkan tanggal, waktu, dan zona waktu kelas |
| Jenis tugas | Pilihan terstruktur, tetapi nilai final mengikuti data implementasi |
| Tempat pengumpulan | Minimal salah satu dari keterangan teks atau URL |
| Materi terkait | Opsional dan tidak boleh menghalangi publikasi |

Draf boleh belum lengkap. Pratinjau dan publikasi harus menunjukkan field wajib yang masih kurang tanpa menghapus input.

## 5. Layar yang Harus Dibuat

### SCR-TASK-001 Daftar Tugas Pengelola

Struktur minimum:

- header dengan ringkasan `Sedang mengelola` yang memuat kelas, semester, dan peran;
- tombol `Tambah Tugas` jika pengguna berwenang;
- tab `Aktif`, `Draf`, `Perlu diperiksa KM`, `Selesai`, `Terlewat`, dan `Arsip`;
- filter mata kuliah, status, dan rentang deadline;
- urutan default deadline terdekat;
- setiap item menampilkan mata kuliah, judul, tenggat, publikasi, pemeriksaan KM, dan kondisi selesai atau terlambat;
- tab `Perlu diperiksa KM` hanya tersedia untuk KM.

### SCR-TASK-002 Form Tugas

Gunakan satu halaman dengan urutan `Informasi tugas`, `Tenggat`, lalu `Pengumpulan`. Sediakan variasi tambah, ubah, validasi gagal, sesi berakhir, dan ada perubahan yang lebih baru. Pada mobile gunakan satu kolom dan letakkan tindakan setelah field terakhir. Tindakan utama adalah `Tinjau Sebelum Terbit`; tindakan sekunder adalah `Simpan sebagai Draf`.

Jika pengguna mengubah tugas yang sudah terbit, jelaskan bahwa versi bertambah dan mahasiswa mungkin menerima notifikasi pembaruan. Untuk PJ, jelaskan bahwa versi baru perlu diperiksa KM.

### SCR-TASK-003 Pratinjau Publikasi

Pratinjau menampilkan seluruh informasi sebagaimana akan terlihat oleh mahasiswa, kelas dan mata kuliah, serta ringkasan dampak. Sediakan `Ubah Lagi` dan `Terbitkan Tugas`.

Setelah PJ memublikasikan, tampilkan hasil `Tugas diterbitkan dan perlu diperiksa KM`. Setelah KM memublikasikan, tampilkan hasil `Tugas diterbitkan dan sudah diperiksa untuk versi ini`.

### SCR-TASK-004 Detail dan Riwayat Tugas

Gunakan navigasi lokal `Detail`, `Riwayat`, dan `Notifikasi`. Bagian detail menampilkan status publikasi, pemeriksaan KM, tenggat, penyelesaian, serta arsip secara terpisah. Riwayat memperlihatkan versi, pelaku, waktu, perubahan, dan hasil pemeriksaan.

Aksi mengikuti peran:

- PJ: Ubah, Tandai Selesai, Arsipkan.
- KM: Ubah, Periksa Tugas, Tandai Selesai, Arsipkan, dan tindakan pembatalan sesuai status.

### SCR-TASK-005 Antrean Pemeriksaan KM

Tampilkan tugas `PUBLISHED` dengan status pemeriksaan `NOT_REVIEWED`, diurutkan dari tenggat terdekat. Setiap item menunjukkan PJ, mata kuliah, versi, waktu publikasi, dan tenggat. Sediakan perbandingan jika terdapat versi sebelumnya.

### SCR-TASK-006 Dialog Hasil Pemeriksaan

- `Setujui`: konfirmasi singkat; catatan opsional.
- `Minta Koreksi`: catatan wajib; jelaskan bahwa tugas ditarik dari portal dan kembali menjadi draf.
- `Batalkan`: catatan wajib; jelaskan bahwa publikasi dicabut dan notifikasi tertunda dibatalkan.

Jika versi berubah ketika dialog terbuka, jangan menerapkan keputusan. Tampilkan conflict state dan aksi `Muat Versi Terbaru`.

### SCR-PORTAL-004 Daftar Tugas Mahasiswa

Kelompokkan menjadi `Hari Ini`, `Minggu Ini`, `Mendatang`, dan `Terlewat`. Gunakan urutan deadline terdekat. Tampilkan mata kuliah, judul, deadline, dan kondisi selesai jika informasi tersebut memang ditujukan kepada kelas. Jangan tampilkan review state internal.

### SCR-PORTAL-005 Detail Tugas Mahasiswa

Tampilkan mata kuliah, judul, instruksi, deadline beserta zona waktu, jenis, tempat pengumpulan, dan materi terkait. Tugas yang tidak lagi terbit tidak membuka isi lama sebagai tugas aktif; tampilkan pesan koreksi yang sesuai tanpa membuka audit internal.

## 6. Alur Utama

```text
PJ: Daftar Tugas
  -> Tambah Tugas
  -> Isi Form
  -> Preview
  -> Publikasikan
  -> Detail Tugas: Terbit, Perlu diperiksa KM

Mahasiswa: Daftar Tugas
  -> Detail Tugas Terbit

KM: Antrean Pemeriksaan
  -> Detail Tugas
  -> Setujui | Minta Koreksi | Batalkan
  -> Hasil dan Riwayat
```

## 7. Keadaan Layar yang Wajib Dibuat

| State | Perilaku desain |
|---|---|
| Belum ada data | Jelaskan belum ada tugas dan tampilkan `Tambah Tugas` hanya jika diizinkan |
| Filter tanpa hasil | Tampilkan filter aktif dan `Hapus Semua Filter` |
| Memuat | Pertahankan struktur halaman serta ringkasan kelas dan peran |
| Gagal memuat | Sebut bagian yang gagal dimuat dan tampilkan `Coba Lagi` |
| Validation | Tunjukkan field bermasalah, pesan dekat field, serta ringkasan jika diperlukan |
| Session expired | Pertahankan input dan arahkan login ulang |
| Tidak memiliki akses | Jangan membuka data; tawarkan `Kembali` atau `Pindah kelas atau peran` |
| Ada perubahan yang lebih baru | Pertahankan input, tampilkan versi terbaru, dan tawarkan salin atau muat ulang |
| Bot offline | Tugas tetap terbit; status pesan menunjukkan antrean |
| Success | Sebut tugas dan status barunya |
| Versi berubah saat diperiksa | Blokir keputusan dan minta KM memuat versi terbaru |

## 8. Responsif dan Aksesibilitas

- Buat setiap layar pada lebar mobile dan desktop.
- Pada mobile, filter lanjutan dapat memakai drawer. Filter aktif tetap terlihat setelah drawer ditutup.
- Daftar desktop dapat memakai tabel hanya jika versi mobile berubah menjadi daftar berlabel.
- Deadline tidak hanya dibedakan dengan warna.
- Dialog dapat ditutup dengan Escape dan mengembalikan fokus.
- Urutan fokus mengikuti urutan visual.
- Kontrol sentuh minimal 44 x 44 piksel.
- Pesan error terhubung dengan field dan dapat dibaca screen reader.

## 9. Data Contoh untuk Prototype

Gunakan bagian tugas pada [Data Contoh untuk Prototype](../PROTOTYPE_DATA.md). Data tersebut menyediakan daftar aktif, draf, tugas yang perlu diperiksa, tugas terlewat, detail tugas, serta pesan keadaan layar. Pakai keadaan berisi sebagai frame utama dan buat keadaan kosong sebagai variasi terpisah.

Jangan membuat nama kampus, dosen, mahasiswa, statistik kelas, logo, atau foto profil.

## 10. Prompt Awal pen.dev

```text
Read these files in order:
1. docs/design/PENCIL_CONTEXT.md
2. docs/design/flows/TASK_MANAGEMENT.md
3. docs/design/PROTOTYPE_DATA.md

Create the Task Management vertical slice as an editable UI exploration v0.1. Generate all screens listed in section 5 for mobile and desktop. Start from the manager app shell, then the student portal views. Use Indonesian UI copy and the labeled fictional data in docs/design/PROTOTYPE_DATA.md. Use populated examples as the main frames and generate empty states as separate variants.

Follow permissions, publication status, review state, version rules, and required states exactly. Do not invent roles, approval steps, features, brand assets, statistics, or implementation technology. Keep review state hidden from students. Keep web publication separate from WhatsApp delivery status.

Do not use the visible labels Konteks, Ganti Konteks, Teaching Event, Offering, Review, Reset Filter, Access Denied, or Conflict Edit. Use the plain-language interface terms defined in PENCIL_CONTEXT.md.

Organize frames by actor and flow. Name every frame with its SCR ID and state. Reuse components for the class-and-role switcher, navigation, status labels, filters, task rows, form fields, preview, dialog, feedback, and history. Do not generate production code yet.
```

## 11. Checklist Pemeriksaan

- [ ] PJ dapat menyimpan draf dan memublikasikan tanpa approval awal.
- [ ] KM memiliki antrean pemeriksaan dan tiga keputusan yang benar.
- [ ] Mahasiswa hanya melihat tugas yang masih terbit.
- [ ] Pemeriksaan KM selalu menyebut versi yang ditinjau.
- [ ] Publikasi, review, selesai, terlambat, dan arsip tidak digabung menjadi satu status.
- [ ] Perubahan PJ dan KM menghasilkan konsekuensi review yang berbeda.
- [ ] Semua state wajib tersedia.
- [ ] Mobile dan desktop tersedia tanpa kehilangan informasi penting.
- [ ] Copy memakai istilah kanonis dan Bahasa Indonesia.
- [ ] Hasil tetap berlabel eksplorasi, bukan desain final.
- [ ] Frame utama memakai data contoh berisi dan state kosong tersedia sebagai variasi terpisah.

## 12. Changelog

### 1.1.0, 24 September 2026

- Mengganti istilah review dan konteks dengan copy antarmuka yang lebih langsung.
- Menautkan kumpulan data contoh bersama dan meminta frame utama berisi data.

### 1.0.0, 23 September 2026

- Membuat paket context khusus vertical slice tugas untuk pen.dev.
- Menetapkan layar, izin, status, field, alur, state, data contoh, prompt, dan checklist review.
