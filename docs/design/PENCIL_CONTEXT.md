# Panduan Desain pen.dev

## Status Dokumen

| Atribut | Nilai |
|---|---|
| Versi | 1.3.0 |
| Status | Approved untuk eksplorasi UI/UX (Design Tokens v2.0) |
| Pemilik | Tim Bot Jadwal |
| Terakhir diperbarui | 24 September 2026 |
| Cakupan | Panduan global yang selalu diberikan kepada agent pen.dev |
| Sumber | [Product Definition](../product/PRODUCT_DEFINITION.md), [Functional Requirements](../product/FUNCTIONAL_REQUIREMENTS.md), [Access Control](../product/ACCESS_CONTROL.md), [User Flows](../product/USER_FLOWS.md), dan [Information Architecture](../product/INFORMATION_ARCHITECTURE.md) |

Dokumen ini adalah panduan ringkas untuk eksplorasi desain di pen.dev. Dokumen ini tidak menggantikan kebutuhan produk yang sudah disetujui dan tidak boleh dipakai untuk mengubah aturan produk.

## 1. Produk

Bot Jadwal membantu satu atau beberapa kelas mengelola jadwal perkuliahan, perubahan sesi, tugas, materi, pengingat WhatsApp, semester, dan pengurus. Aplikasi web mengurangi ketergantungan pada command WhatsApp. Portal mahasiswa berfokus pada konsumsi informasi, sedangkan area pengelola berfokus pada input, publikasi, koreksi, dan audit.

## 2. Pengguna dan Akses

| Aktor | Akses | Cakupan utama |
|---|---|---|
| Mahasiswa | Tanpa akun, melalui tautan atau kode kelas | Membaca jadwal, tugas, materi, perubahan, dan arsip semester terbit |
| PJ Mata Kuliah | Login dan penugasan aktif | Mengelola mata kuliah yang ditugaskan |
| Ketua Murid (KM) | Login dan penugasan aktif | Mengelola seluruh mata kuliah, semester, PJ, dan pengaturan pada kelasnya |
| System Admin | Login dengan sesi lebih singkat | Administrasi global, master data, audit, dukungan, backup, dan pemulihan |

Sebelum pengguna mengubah data, antarmuka harus selalu menunjukkan kelas, semester, peran, dan mata kuliah yang sedang dikelola. Jangan memakai label `Konteks` atau `Ganti Konteks` pada antarmuka. Gunakan `Sedang mengelola` dan `Pindah kelas atau peran`. Navigasi dan aksi yang tidak diizinkan tidak ditampilkan. Backend tetap menjadi penentu izin final.

## 3. Ruang Aplikasi

1. `Portal Kelas`: hanya-baca, mobile-first, tanpa akun mahasiswa.
2. `Area Pengelola`: app shell untuk PJ dan KM.
3. `System Admin`: area terpisah untuk operasi lintas kelas.
4. `Akses Akun`: login, aktivasi undangan, pemulihan, serta pemilihan kelas dan peran.

Portal dan Area Pengelola bukan variasi dari layar yang sama. Portal mengutamakan jadwal dan tenggat. Area Pengelola mengutamakan pekerjaan tertunda, formulir, pratinjau, status publikasi, serta riwayat.

## 4. Navigasi Utama

### Portal Kelas

- Ringkasan
- Jadwal
- Tugas
- Mata Kuliah dan Materi
- Perubahan Terbaru
- Arsip Semester

### Area Pengelola

- Ringkasan
- Jadwal
- Tugas
- Lainnya: Materi, Ruangan, Anggota dan Peran, Semester, Riwayat Perubahan, Pengaturan Kelas

Pada mobile, gunakan bottom navigation untuk `Ringkasan`, `Jadwal`, `Tugas`, dan `Lainnya`. Pada desktop, gunakan sidebar. Tombol tambah merupakan aksi halaman, bukan item navigasi.

### System Admin

- Ringkasan Sistem
- Kelas
- Pengguna dan Penugasan
- Master Mata Kuliah
- Master Ruangan
- Antrean Notifikasi
- Backup dan Pemulihan
- Audit Global
- Status Sistem dan WhatsApp

## 5. Model Interaksi

- Operasi publikasi mengikuti urutan input, validasi, pratinjau, hasil, dan riwayat perubahan. Jangan menambahkan dialog konfirmasi setelah layar pratinjau.
- Publikasi web tidak bergantung pada keberhasilan WhatsApp.
- Status data dan status pengiriman pesan ditampilkan terpisah.
- Aksi berisiko menjelaskan dampaknya dan meminta konfirmasi.
- Konflik edit mempertahankan input pengguna dan menawarkan pemuatan versi terbaru.
- URL detail tetap stabil ketika objek berpindah ke arsip.

## 6. Bahasa Antarmuka dan Istilah Kanonis

| Konsep | Nilai atau label |
|---|---|
| Publikasi tugas | `DRAFT`, `PUBLISHED`, `REVOKED` |
| Pemeriksaan tugas (`review_state` pada data) | `NOT_REVIEWED`, `APPROVED`, `CHANGES_REQUESTED`, `REVOKED` |
| Kondisi tugas | Selesai dari `completed_at`; Terlewat dihitung dari tenggat, bukan status tersimpan |
| Perubahan jadwal (`teaching_event` pada data) | `DRAFT`, `PUBLISHED`, `REVOKED` |
| Partisipasi lintas kelas | `PENDING`, `ACCEPTED`, `DECLINED`, `REMOVED` |
| Semester | `DRAFT`, `ACTIVE`, `ARCHIVED` |
| Notifikasi | `PENDING`, `PROCESSING`, `SENT`, `FAILED` |

Kode status dan nama tabel hanya boleh muncul dalam anotasi desain, bukan sebagai teks utama untuk pengguna.

| Istilah internal | Teks antarmuka | Contoh pemakaian |
|---|---|---|
| konteks aktif | Sedang mengelola | `D4 TI 2024 A · Semester 3 · Ketua Murid` |
| ganti konteks | Pindah kelas atau peran | Tombol pada menu akun |
| pemilih konteks | Pilih kelas dan peran | Judul setelah pengguna dengan beberapa akses masuk |
| teaching event | Perubahan jadwal | Nama area untuk kelas pengganti, tambahan, libur, dan pembatalan sesi |
| offering | Mata kuliah di kelas ini | Pilihan mata kuliah yang dapat dikelola |
| owner offering | Mata kuliah penyelenggara | Label saat perubahan melibatkan kelas lain |
| participant offering | Kelas peserta | Label kelas lain yang diundang |
| review | Pemeriksaan KM | Nama proses internal pada halaman pengelola |
| Perlu Review | Perlu diperiksa KM | Status tugas yang baru diterbitkan PJ |
| Reset Filter | Hapus Semua Filter | Tindakan saat hasil filter kosong |
| access denied | Tidak memiliki akses | Judul keadaan tanpa izin |
| conflict edit | Ada perubahan yang lebih baru | Judul ketika versi data berbeda |
| room candidate | Perlu konfirmasi TU | Status ruangan yang belum dikonfirmasi |

Gunakan istilah berikut secara konsisten: `Ketua Murid (KM)`, `PJ Mata Kuliah`, `Perubahan Jadwal`, `Terbit`, `Publikasi Dicabut`, `Sesi Dibatalkan`, `Perlu diperiksa KM`, `Perlu konfirmasi TU`, dan `Riwayat Perubahan`.

### Prinsip penulisan

- Judul halaman menyebut objek atau pekerjaan pengguna, misalnya `Buat Perubahan Jadwal`, bukan `Form Event`.
- Tombol menyebut hasil tindakan, misalnya `Tinjau Sebelum Terbit`, `Terbitkan Tugas`, atau `Simpan sebagai Draf`.
- Jangan memakai campuran Bahasa Indonesia dan Inggris jika padanan Indonesianya jelas.
- Jangan menampilkan istilah basis data, singkatan status, atau nama lifecycle sebagai label utama.
- Pesan gagal menjelaskan bagian yang gagal dan tindakan berikutnya.
- Pesan berhasil menyebut objek yang berubah tanpa mengulang seluruh isi formulir.

## 7. Keadaan Layar Wajib

Setiap layar data harus memiliki desain untuk:

- memuat, dengan nama data yang sedang dimuat;
- belum ada data, dengan tindakan yang sesuai izin;
- filter tanpa hasil, dengan tindakan `Hapus Semua Filter`;
- gagal memuat, dengan tindakan `Coba Lagi`;
- tidak memiliki akses, tanpa membocorkan data;
- ada perubahan yang lebih baru, tanpa membuang input pengguna;
- berhasil, dengan nama objek dan status barunya;
- pesan WhatsApp tertunda jika layar menampilkan status pengiriman.

Status harus memakai teks atau ikon selain warna. Jangan menampilkan kontrol mati, navigasi menuju halaman yang belum ada, statistik palsu, atau konten akademik yang tampak nyata tanpa label contoh.

## 8. Panduan Visual & Design System (Design Tokens)

Status visual: **Approved Design Reference (v2.0)** mengacu pada sistem desain *editorial-tech* yang sudah divalidasi. Agent pen.dev **WAJIB** menerapkan token warna, tipografi, dan anatomi komponen berikut secara konsisten pada seluruh frame desktop dan mobile.

### A. Palet Warna (Color Tokens)

| Token | Nilai HEX | Penggunaan |
|---|---|---|
| `bg-canvas` | `#F3F1ED` | Background dasar halaman/canvas (warm off-white/subtle cream, bukan abu-abu dingin). |
| `surface-dark` | `#20232B` | Sidebar desktop, mobile app bar, dan dark contrast cards ("Fokus Hari Ini", "Tugas Mendesak"). |
| `surface-light` | `#FFFFFF` | Permukaan kartu utama, modal dialog, formulir input, dan tabel data. |
| `primary-brand` | `#315CBE` | Tombol aksi utama, item navigasi aktif, brand icon box, link aktif, dan elemen terpilih. |
| `primary-tint` | `#E8EDF7` | Background highlight lembut, active tab mobile, badge info netral, dan avatar pengurus. |
| `primary-light` | `#AFC3E8` | Teks sekunder pada background gelap `#20232B`, dot status WhatsApp, dan sub-label. |
| `text-primary` | `#252525` | Teks judul, label field, isi tabel, dan teks keterbacaan tinggi pada background terang. |
| `text-muted` | `#666A73` | Teks deskripsi, subtitle, placeholder, dan metadata sekunder. |
| `border-subtle` | `#D8D5CF` | Outline kartu terang dan garis pemisah dekoratif. Jangan gunakan untuk batas kontrol interaktif. |
| `border-control` | `#8B8780` | Batas input dan kontrol interaktif pada background terang. Rasio 3,57:1 terhadap putih. |
| `border-dark` | `#3B404B` | Garis pemisah dekoratif pada permukaan gelap `#20232B`. |
| `border-dark-control` | `#747B88` | Batas kontrol interaktif pada permukaan gelap. Rasio 3,69:1 terhadap `#20232B`. |
| `warning-bg` | `#FFF1CF` | Background alert jadwal pengganti, badge draf/proses, dan callout WhatsApp broadcast. |
| `warning-border` | `#D4A64A` / `#946200` | Border dan outline aksen warning/perhatian. |
| `warning-text` | `#946200` | Teks dan ikon peringatan, status draf, badge proses. |
| `danger-text` | `#A53A32` | Teks deadline mendesak, tombol tolak, aksi keluar/logout, dan error state. |
| `danger-on-dark` | `#F58B82` | Teks atau ikon bahaya pada `surface-dark`. Rasio 6,66:1 terhadap `#20232B`. |
| `danger-bg` | `#FBE9E7` | Background badge status "MENDESAK" dan callout error. |
| `success-text` | `#32704A` | Teks status selesai, indikator sukses, dan badge terbit/terkirim. |
| `success-bg` | `#E8F5E9` | Background badge sukses atau selesai. |

### B. Tipografi Ganda (Typography Pairing)

1. **Font Primer UI (`Funnel Sans`, system-ui, sans-serif):**
   - Digunakan untuk: seluruh judul halaman (`Page Title`), nama tombol, copy penjelasan, label form input, dan konten tabel.
   - Karakter: modern, humanis, sangat mudah dibaca.
2. **Font Teknis Monospace (`IBM Plex Mono`, monospace):**
   - Digunakan untuk:
     - Jam dan waktu (contoh: `16.00 WIB`, `Senin, 13.30 - 15.20`).
     - Status pendek yang perlu dipindai cepat (contoh: `Reguler`, `Berubah`, `Mendesak`, `Proses`, `Menunggu`, `Belum diperiksa`).
     - Kode mata kuliah dan kelas (contoh: `ALIN-T`, `PLP-P`, `OS-T`, `SCR-SCH-005`).
     - Inisial avatar pengurus (contoh: `IM`, `HA`, `BJ`).
     - Label teknis singkat yang memang perlu dipindai cepat (contoh: kode kelas dan versi). Hindari label Inggris atau huruf kapital dekoratif.

### C. Anatomi Komponen Kunci

1. **Brand Mark:**
   - Bujur sangkar `w-[36px] h-[36px]` dengan background `#315CBE`.
   - Di dalamnya memuat teks inisial `BJ` warna putih tebal (`IBM Plex Mono` bold).
   - Di sampingnya teks produk `bot-jadwal` (bold) dan nama area yang sedang dibuka, misalnya `Area Pengelola`.
2. **Sidebar Desktop (`#20232B`):**
   - Item menu navigasi berukuran tinggi 44-48px dengan padding `0 14px`.
   - Item aktif: background `#315CBE` dengan teks dan ikon putih.
   - Item non-aktif: background transparan dengan teks `#E8EDF7` dan ikon `#AFC3E8`.
   - **Status WhatsApp:** Kotak di bagian bawah sidebar dengan outline `#3B404B`, indikator bulat `#AFC3E8`, teks `Terhubung`, serta jadwal pesan berikutnya (`Pesan berikutnya 17.00 WIB`).
   - **Kartu Akun:** Kotak avatar inisial (`PC`), nama `Pengguna Contoh`, peran (`Ketua Murid`), dan tombol `Pindah kelas atau peran`.
3. **Dark Contrast Cards (`#20232B`):**
   - Digunakan pada "Fokus Hari Ini" (mobile) dan "Tugas Mendesak" (desktop) untuk menciptakan titik fokus hierarki visual instan.
   - Teks judul putih tebal, waktu tenggat menggunakan `IBM Plex Mono` warna emas (`#D5A647`) atau `danger-on-dark` (`#F58B82`).
4. **Tombol Aksi (Buttons):**
   - **Primary Action:** Background `#315CBE`, teks `#FFFFFF` bold, sudut `rounded-[4px]`.
   - **Secondary / Outline Action:** Background `#FFFFFF`, `outline: 1px solid #252525` atau `border-control`, teks `#252525`.
   - **Destructive / Rejection Action:** Background transparan atau putih, `outline: 1px solid #A53A32`, teks `#A53A32`.
   - DILARANG menggunakan tombol hitam pekat generik tanpa fungsi semantik.
5. **Indikator Hak Akses Visual (RBAC Cues):**
   - Mode PJ Mata Kuliah wajib menampilkan pembeda eksplisit:
     - Item yang di luar wewenang: diberi label `Hanya lihat` warna netral (`#666A73`).
     - Item yang dalam wewenang PJ: diberi label `Dapat diubah` warna biru (`#315CBE`), dan saat dipilih memiliki outline `#315CBE` dengan background lembut `#E8EDF7`.
6. **Halaman Ubah Jadwal dengan Perbandingan:**
   - Sesi yang diubah menampilkan dropdown mata kuliah.
   - Pilihan toggle `Sementara` vs `Permanen`.
   - Pada desktop, kotak perbandingan terbagi dua kolom. Pada mobile, tampilkan `Sebelum` lalu `Menjadi` secara vertikal:
     - Kolom kiri: **SEBELUM** (ikon gembok terkunci, data jadwal asli yang tidak bisa diubah).
     - Kolom kanan: **MENJADI** (wajib diisi, input tanggal dengan ikon kalender, jam mulai dan jam selesai otomatis, pilihan ruangan).
   - Banner callout WhatsApp: background `#FFF1CF`, teks `#946200`, menjelaskan bahwa perubahan akan disiarkan ke grup kelas saat dipublikasikan.
   - Tombol footer ganda: `Simpan sebagai Draf` (outline) dan `Tinjau Perubahan` (biru `#315CBE`). Publikasi dilakukan dari layar pratinjau.
7. **Mobile View (Portal Mahasiswa & Mobile Shell):**
   - Lebar standar: 390px (iPhone viewport).
   - App bar atas gelap `#20232B` dengan brand mark `BJ`.
   - Filter pill status fleksibel: `Semua` (aktif gelap `#20232B`), `Mendesak`, `Proses`, `Selesai` (`#32704A`).
   - Bottom navigation bar: tinggi ~68px, background `#FFFFFF`, border atas `#D8D5CF`, item aktif menggunakan kartu soft `#E8EDF7` dengan teks dan ikon `#315CBE`.

---

## 9. Batas Implementasi Frontend

Repository saat ini memakai HTML biasa pada folder `web/`, Alpine.js, dan Tailwind melalui CDN. Jangan menambahkan React, Vue, Next.js, npm, `package.json`, atau build step frontend. Seluruh komponen pen.dev harus dapat diimplementasikan dengan kelas Tailwind CSS standar dan atribut Alpine.js (`x-data`, `x-show`, `@click`).

## 10. Aturan untuk Agent

1. Baca dokumen ini, satu paket alur, dan `PROTOTYPE_DATA.md` sebelum mendesain.
2. Terapkan secara ketat **Design Tokens pada Bagian 8** (warna, tipografi Funnel Sans + IBM Plex Mono, dan gaya kartu).
3. Jangan pernah kembali ke gaya monokrom/grayscale polos kecuali secara eksplisit diminta untuk wireframe mentah.
4. Jangan membaca seluruh folder `docs` kecuali paket vertical slice menunjuk bagian tertentu.
5. Jangan menambah role, fitur, status, atau persetujuan yang tidak tertulis.
6. Jika sumber tampak bertentangan, jangan menebak. Catat ID requirement yang bertentangan.
7. Gunakan Bahasa Indonesia untuk seluruh copy antarmuka.
8. Isi frame utama dengan [data contoh untuk prototype](PROTOTYPE_DATA.md). Frame kosong dibuat sebagai variasi terpisah, bukan sebagai satu-satunya hasil.
9. Hasilkan versi mobile dan desktop yang responsif.
10. Sertakan state wajib (empty, loading, error) dan variasi izin yang relevan.

## 11. Arah UX dan Alasan Keputusan

**Design Read:** aplikasi operasional untuk mahasiswa dan pengurus yang paling sering memakai ponsel, dengan gaya editorial-tech yang tenang. Dial: `ENERGY 1 / RHYTHM 2 / MOTION 1`.

| Keputusan | Alasan |
|---|---|
| Satu aksi utama per layar | Pengguna dapat langsung melihat langkah berikutnya tanpa membandingkan banyak tombol setara. |
| Ringkasan `Sedang mengelola` selalu terlihat | Perubahan data harus jelas berlaku untuk kelas, semester, dan peran yang mana. |
| Preview sebelum publikasi | Pengurus dapat memeriksa dampak tanpa menambah dialog konfirmasi yang berulang. |
| Detail lanjutan dibuka saat diperlukan | Layar ponsel tetap ringkas, sementara informasi audit dan notifikasi tetap tersedia. |
| Kartu gelap hanya untuk hal paling mendesak | Kontras menandai satu fokus, bukan menjadi dekorasi di seluruh halaman. |
| Tema terang sebagai dasar | Mahasiswa dan pengurus paling sering membuka aplikasi di kelas atau lingkungan kampus yang terang; permukaan gelap tetap dipakai terbatas sebagai penanda fokus. |
| Funnel Sans untuk teks dan IBM Plex Mono untuk kode atau waktu | Teks tetap ramah dibaca, sedangkan data teknis mudah dipindai. |
| Biru sebagai aksen tindakan | Warna utama menunjukkan pilihan aktif dan tindakan utama tanpa memenuhi seluruh layar. |
| Gerak terbatas pada transisi keadaan | Produk dipakai untuk bekerja cepat; animasi tidak boleh memperlambat pemindaian. |

Urutan alur utama harus terbaca sebagai `pilih pekerjaan → isi data → tinjau dampak → terbitkan → lihat hasil`. Jangan meminta konfirmasi kedua setelah layar pratinjau kecuali tindakan menghapus atau mencabut publikasi memiliki dampak yang tidak dapat dibatalkan langsung.

## 12. Sumber Berdasarkan Pertanyaan

| Pertanyaan | Sumber |
|---|---|
| Apa yang harus tersedia? | `FUNCTIONAL_REQUIREMENTS.md` |
| Siapa boleh melakukan apa? | `ACCESS_CONTROL.md` dan `BUSINESS_RULES.md` |
| Bagaimana urutan interaksi? | `USER_FLOWS.md` |
| Halaman dan navigasinya apa? | `INFORMATION_ARCHITECTURE.md` |
| Field dan hubungan datanya apa? | `DATA_MODEL.md` |
| Mengapa fitur dibutuhkan? | `USER_REQUIREMENTS.md` dan `PRODUCT_DEFINITION.md` |

## 13. Changelog

### 1.3.0, 24 September 2026

- Mengganti istilah antarmuka yang teknis dengan label yang menjelaskan kelas, peran, dan tindakan pengguna.
- Menetapkan arah UX, dial desain, serta alasan keputusan visual dan alur.
- Menautkan data contoh bersama agar frame utama tidak tergenerate sebagai layar kosong.

### 1.2.0, 24 September 2026

- Mengadopsi Visual Design System & Design Tokens resmi v2.0 (*editorial-tech*) berbasis referensi tervalidasi.
- Menetapkan palet warna eksplisit (`#F3F1ED`, `#20232B`, `#315CBE`, `#FFF1CF`, `#A53A32`, `#32704A`).
- Menetapkan typography pairing (`Funnel Sans` + `IBM Plex Mono`).
- Menambahkan anatomi komponen kunci: Dark Contrast Cards, Widget Telemetri Bot WhatsApp, Side-by-Side Schedule Comparison, dan Mobile App Bar & Bottom Navigation.

### 1.1.0, 24 September 2026

- Mengunci eksplorasi awal pada grayscale setelah generator memilih warna aksen tanpa arahan produk.

### 1.0.0, 23 September 2026

- Membuat panduan global untuk eksplorasi desain berbasis kebutuhan produk.
- Menetapkan batas akses, navigasi, keadaan layar, istilah, arahan visual sementara, dan stack frontend.
