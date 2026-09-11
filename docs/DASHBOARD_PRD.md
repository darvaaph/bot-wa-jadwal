# 📱 Panduan Fitur & Spesifikasi Produk (PRD)
# Web Admin Dashboard — bot-jadwal v2.0

Panduan ini disusun untuk memberikan gambaran menyeluruh bagi tim **UI/UX Designer** dan **Frontend Developer** mengenai seluruh fitur, alur pengguna (*user journey*), dan kebutuhan komponen antarmuka yang akan dibangun pada Web Admin Dashboard.

---

## 1. 👥 Persona & Kebutuhan Pengguna

1. **Ketua Kelas & PJ Mata Kuliah (Pengelola Data / Admin):**
   - Mengakses dashboard dari ponsel pintar (*smartphone*) saat berada di ruang kelas kampus.
   - Membutuhkan antarmuka yang cepat untuk mencatat tugas baru dari dosen dalam waktu **kurang dari 20 detik** tanpa ribet.
   - Membutuhkan kemudahan menggeser jadwal perkuliahan atau menandai kelas kosong secara visual.
2. **Mahasiswa Kelas (Pengguna Informasi):**
   - Membuka dashboard untuk mengecek jadwal kuliah hari ini, ruang kelas, dan tautan Zoom perkuliahan.
   - Melihat daftar tugas kuliah yang sedang aktif dan mengecek tugas apa saja yang tenggat waktunya (deadline) sudah dekat.
3. **Administrator Teknis:**
   - Memantau apakah koneksi bot WhatsApp di server cloud sedang aktif atau membutuhkan pemindaian ulang QR Code tanpa perlu membuka terminal Linux (SSH).

---

## 2. 🗺️ Struktur Navigasi Utama Dashboard

Dashboard dirancang sebagai aplikasi satu halaman (*Single Page Dashboard*) dengan **4 tab navigasi utama**:

```text
┌────────────────────────────────────────────────────────────────────────┐
│                   NAVIGASI WEB ADMIN DASHBOARD                         │
├───────────────┬───────────────────┬──────────────────┬─────────────────┤
│ 1. RINGKASAN  │ 2. JADWAL KULIAH  │ 3. CATATAN TUGAS │ 4. GATEWAY WA   │
│  (Overview)   │    (Schedule)     │     (Tasks)      │  (Bot Status)   │
└───────────────┴───────────────────┴──────────────────┴─────────────────┘
```

- **Tampilan Desktop (1440px):** Navigasi berupa Sidebar di sisi kiri atau Header navigasi di sisi atas.
- **Tampilan Mobile (390px):** Navigasi berupa Bottom Navigation Bar (di bawah layar) atau Drawer Menu (hamburger) yang ramah dioperasikan dengan satu jempol (*thumb-friendly*).

---

## 3. 🔍 Rincian Fitur per Halaman

---

### Tab 1: Ringkasan Perkuliahan *(Overview)*
> **Fokus:** Informasi instan dalam 3 detik pertama saat web dibuka.

#### Data & Komponen yang Ditampilkan:
1. **Header Sambutan:** Menampilkan hari, tanggal saat ini, dan kelas perkuliahan yang aktif (misal: *D4 Teknik Informatika - Kelas 3A*).
2. **Kartu Kuliah Hari Ini (Today's Class Card):**
   - Menampilkan mata kuliah yang sedang berlangsung atau yang akan datang berikutnya hari ini.
   - Memuat: Nama mata kuliah, jam mulai–selesai, ruangan kelas, dan inisial/nama dosen.
   - Tombol cepat: **"Buka Link Kuliah"** (jika kelas berstatus daring via Zoom/Meet).
3. **Widget Mini Tugas Mendesak (Urgent Tasks):**
   - Daftar 2–3 tugas dengan tenggat waktu paling dekat (< 24 jam).
   - Tautan cepat bertuliskan *"Lihat Semua Tugas"* menuju Tab 3.
4. **Indikator Koneksi WhatsApp:**
   - Badge status kecil di pojok: Hijau (*"Bot Aktif"*) atau Merah (*"Bot Terputus"*).

---

### Tab 2: Jadwal Kuliah Mingguan *(Schedule)*
> **Fokus:** Akses jadwal Senin–Jumat yang terstruktur rapi dan mudah dibaca.

#### Data & Komponen yang Ditampilkan:
1. **Filter Hari (Day Selector):**
   - Tombol pilihan hari: **Senin, Selasa, Rabu, Kamis, Jumat**.
   - Menekan tombol hari akan langsung menyaring daftar mata kuliah pada hari tersebut tanpa reload halaman.
2. **Daftar / Tabel Mata Kuliah:**
   - Setiap mata kuliah ditampilkan dalam bentuk kartu (*card*) atau baris tabel yang rapi.
   - Informasi tiap jadwal:
     - **Jam:** Contoh `07:30 - 09:10 WIB`.
     - **Mata Kuliah:** Nama lengkap matkul beserta label kategori (`Teori` atau `Praktikum`).
     - **Dosen:** Inisial & nama dosen pengampu.
     - **Ruangan:** Ruang kelas (misal: `Lab 302` atau `R. Teori 3`).
     - **Aksi:** Tombol salin tautan kuliah daring (Zoom / Google Meet).
3. **Penanda Jadwal Pengganti (Schedule Override):**
   - Jika ada mata kuliah yang digeser jamnya atau ditiadakan, kartu diberi penanda khusus:
     - Badge Kuning: *"Jadwal Pengganti"* (menampilkan jam/hari baru).
     - Badge Abu-abu: *"Kelas Kosong / Diliburkan"*.

---

### Tab 3: Manajemen Catatan Tugas *(Tasks)* — *Fitur Paling Utama*
> **Fokus:** Pencatatan cepat, pemantauan tenggat waktu, dan kejelasan status.

#### Data & Komponen yang Ditampilkan:
1. **Tombol Utama "+ Tambah Tugas Baru":**
   - Tombol aksi utama yang mencolok. Pada layar HP, tombol ini wajib diletakkan pada posisi yang mudah dijangkau ibu jari (*floating action button* atau tombol lebar di atas daftar).
2. **Daftar Kartu Tugas (Task Cards):**
   - Disusun otomatis berdasarkan tenggat waktu terdekat.
   - Setiap kartu memuat:
     - **Nama Mata Kuliah:** (Contoh: *Sistem Basis Data*).
     - **Deskripsi Tugas:** (Contoh: *Laporan Praktikum Modul 2 - Bab 3*).
     - **Tenggat Waktu (Deadline):** Format waktu terbaca manusia (Contoh: *Jumat, 12 Sep 2026 - 23:59 WIB*).
     - **Badge Urgensi Deadline (Wajib 3 Warna):**
       - 🔴 **Merah:** Tenggat < 24 jam atau telah terlewat (*Mendesak*).
       - 🟡 **Kuning/Oranye:** Tenggat 1–3 hari ke depan (*Perhatian*).
       - 🟢 **Hijau / Netral:** Tenggat masih > 3 hari (*Aman*).
     - **Tombol Aksi Kartu:**
       - Tombol checklist: Tandai tugas telah selesai.
       - Tombol tempat sampah: Hapus tugas.
3. **Modal Pop-up Formulir Tambah Tugas:**
   - Muncul saat tombol "+ Tambah Tugas" ditekan.
   - Input yang dibutuhkan:
     1. Pilihan Mata Kuliah *(Dropdown select).*
     2. Deskripsi / Judul Tugas *(Textarea / text input).*
     3. Tanggal & Jam Deadline *(Datetime picker).*
   - Tombol **"Batal"** dan tombol **"Simpan Tugas"**.
   - Aksesibilitas: Dapat ditutup via klik di luar modal (backdrop), tombol Batal, atau tombol `Escape`.
4. **Variasi Status Layar (States) yang Wajib Didesain:**
   - **Default State:** Daftar kartu tugas terisi data.
   - **Empty State:** Jika tidak ada tugas aktif, tampilkan ilustrasi/ikon ramah dengan teks: *"Belum ada tugas aktif untuk kelas ini"* beserta tombol pemicu "+ Tambah Tugas".
   - **Error / Validation State:** Border merah pada form jika pengguna menekan tombol simpan dengan kolom kosong.

---

### Tab 4: Status WhatsApp Gateway & Scan QR *(Bot Status)*
> **Fokus:** Telemetri server bot dan penyambungan ulang (*re-pairing*) tanpa SSH terminal.

#### Data & Komponen yang Ditampilkan:
1. **Panel Status Koneksi:**
   - Indikator visual status:
     - 🟢 **Connected / Aktif:** Menampilkan nama bot, nomor WhatsApp yang tersambung, dan lama waktu aktif (*uptime*).
     - 🔴 **Disconnected / Terputus:** Menampilkan peringatan bahwa bot sedang offline.
2. **Area Tampilan QR Code (Live QR Container):**
   - Jika bot dalam status terputus / belum login, area ini akan menampilkan gambar **QR Code hitam-putih**.
   - Dilengkapi petunjuk singkat: *"Buka WhatsApp di HP > Perangkat Tertaut > Tautkan Perangkat > Pindai QR Code di atas"*.
   - Tombol **"Muat Ulang QR"** jika QR code kedaluwarsa.

---

## 4. 📐 Standar Desain untuk UI/UX & Frontend

1. **Mobile-First (Prioritas Layar HP):**
   - Target frame mobile di Figma: **390 x 844 px** (iPhone / Android modern).
   - Ukuran target sentuh (*touch target*) untuk seluruh tombol dan input minimal **44 x 44 px** agar tidak salah pencet oleh jempol.
2. **Resolusi Desktop:**
   - Target frame desktop di Figma: **1440 x 900 px**.
   - Manfaatkan ruang desktop dengan tata letak kartu berbentuk grid (2–3 kolom) yang rapi.
3. **Kebebasan Eksplorasi Visual:**
   - Desainer UI/UX dan Frontend memiliki kebebasan penuh dalam menentukan palet warna, tipografi, dan gaya visual (nuansa terang/gelap, kartu modern, clean academic).
   - Yang terpenting adalah informasi hierarkis jelas, mudah dibaca, dan kontras warna memenuhi standar kenyamanan mata.
