# Panduan Desain: Alur Undangan, Aktivasi Akun, dan Login (Batch 1)

Panduan ini adalah acuan kerja untuk tim UI/UX dalam merancang proses pembuatan akun pengurus (KM & PJ), halaman aktivasi dari WhatsApp, halaman login, dan perpindahan peran.

Agar tim UI/UX memiliki konteks visual yang jelas dan tombol modal tidak terkesan "melayang tanpa halaman induk", dokumen ini juga menyertakan kerangka dasar navigasi (*sidebar*) untuk Admin dan KM.

---

## 1. Alur Lengkap: Dari Undangan Sampai Masuk Dashboard

### Alur Ketua Murid (KM)
1. **Admin membuka menu Kelas:** Admin melihat daftar kelas, lalu menekan tombol **"Undang KM"** pada kelas yang belum ada pengurusnya.
2. **Admin memasukkan nomor WA calon KM:** Muncul pop-up modal, Admin mengisi nomor WhatsApp, dan sistem membuatkan tautan undangan khusus.
3. **Kirim via WhatsApp:** Admin mengirimkan tautan tersebut ke WhatsApp calon KM.
4. **KM membuka tautan:** Calon KM mengklik tautan dari chat WhatsApp, lalu terbuka **Halaman Aktivasi Akun KM**.
5. **KM membuat kata sandi:** Di halaman tersebut terlihat jelas informasi kelasnya. KM mengisi nama dan membuat kata sandi baru.
6. **Masuk ke dashboard:** Setelah klik tombol simpan, akun aktif dan KM langsung masuk ke ruang kerja kelasnya.

### Alur Penanggung Jawab (PJ)
1. **KM membuka menu Anggota & Tim:** Di dashboard kelasnya, KM melihat daftar mata kuliah semester ini, lalu menekan tombol **"Undang PJ"** pada mata kuliah terkait.
2. **KM memasukkan nomor WA calon PJ:** Muncul pop-up modal, KM mengisi nomor WhatsApp calon PJ, dan sistem membuatkan tautan undangan khusus.
3. **Kirim via WhatsApp:** KM membagikan tautan undangan ke WhatsApp calon PJ.
4. **PJ membuka tautan:** Calon PJ mengklik tautan di WhatsApp, lalu terbuka **Halaman Aktivasi Akun PJ**.
5. **PJ melihat tugasnya & membuat kata sandi:** Di layar langsung tertera informasi: *"Anda diundang sebagai Penanggung Jawab (PJ) mata kuliah Pemrograman Web di Kelas D4 TI 2024 A"*. PJ mengisi nama dan kata sandi baru.
6. **Masuk ke dashboard:** Setelah tersimpan, PJ langsung masuk ke dashboard yang khusus menampilkan mata kuliah miliknya saja.

---

## 2. Tampilan Sisi Admin: Layar Induk & Modal Undang KM

### Kerangka Menu Sidebar Admin
Tampilan Admin menggunakan layout web desktop dengan menu di sebelah kiri.
*Daftar menu di sidebar:*
* Dashboard Utama
* **Daftar Kelas** $\rightarrow$ *(FOKUS BATCH 1 KITA ADA DI MENU INI)*
* Status Bot WhatsApp *(dikerjakan di batch berikutnya)*
* Antrean Pesan *(dikerjakan di batch berikutnya)*
* Master Ruangan *(dikerjakan di batch berikutnya)*
* Master Mata Kuliah *(dikerjakan di batch berikutnya)*
* Kelola Pengguna *(dikerjakan di batch berikutnya)*
* Backup Data *(dikerjakan di batch berikutnya)*

---

### Halaman Induk Admin: Layar "Daftar Kelas"
*Tempat tombol pemicu modal undangan KM berada.*

* **Tata Letak:**
  - Di bagian atas: Judul halaman **"Daftar Kelas"** dan tombol `+ Buat Kelas Baru`.
  - Area utama: Tabel yang memuat daftar seluruh kelas kampus.
* **Kolom Tabel:**
  - Nama Kelas (misal: *D4 TI 2024 A*, *D4 TI 2024 B*).
  - Program Studi & Angkatan.
  - Ketua Murid (KM) Aktif.
  - Aksi.
* **Kondisi Tombol:**
  - Jika kelas sudah punya KM $\rightarrow$ Tombol aksi: `Detail Kelas`.
  - Jika kelas **belum ada KM** $\rightarrow$ Tombol aksi: **`+ Undang KM`**.

---

### Modal Pop-up: "Undang KM"
*Muncul di tengah layar saat tombol `+ Undang KM` di tabel kelas diklik.*

* **Tujuan:** Membuat tautan undangan untuk Ketua Murid baru.
* **Elemen Tampilan:**
  - **Judul Modal:** "Undang Ketua Murid (KM)"
  - **Info Kelas (Terkunci):** Menampilkan nama kelas yang dipilih (misal: *D4 Teknik Informatika 2024 A*).
  - **Input Field:** Nomor WhatsApp calon KM (contoh: `081234567890`).
  - **Tombol Utama:** "Buat Tautan Undangan"
* **Tampilan Setelah Tombol Diklik (Hasil Sukses):**
  - Kotak tautan berisi link undangan: `app.kampus.ac.id/invite/km-abc123xyz`
  - Tombol **"Salin Tautan"** (Copy Link).
  - Tombol cepat **"Kirim ke WhatsApp"** (membuka WhatsApp dengan draf pesan siap kirim).
  - Keterangan kecil: *"Tautan hanya berlaku 7 hari dan otomatis hangus setelah akun diaktifkan."*

---

## 3. Tampilan Sisi KM: Layar Induk & Modal Undang PJ

### Kerangka Menu Navigasi KM (Ketua Murid)
Tampilan KM menggunakan navigasi kelas:
* Ringkasan Kelas
* Kelola Jadwal *(dikerjakan di batch berikutnya)*
* Kelola Tugas *(dikerjakan di batch berikutnya)*
* Materi & Tautan *(dikerjakan di batch berikutnya)*
* **Anggota & Tim** $\rightarrow$ *(FOKUS BATCH 1 KITA ADA DI MENU INI)*
* Pengaturan Kelas *(dikerjakan di batch berikutnya)*

---

### Halaman Induk KM: Layar "Anggota & Tim"
*Tempat tombol pemicu modal undangan PJ berada.*

* **Tata Letak:**
  - Judul halaman: **"Anggota & Penanggung Jawab Mata Kuliah"**.
  - Menampilkan daftar atau kartu mata kuliah yang ada di semester aktif kelas ini.
* **Contoh Daftar Mata Kuliah:**
  1. *Basis Data Lanjut* $\rightarrow$ Status: PJ Aktif (*Siti Rahma*) $\rightarrow$ Tombol `Ganti PJ`.
  2. *Pemrograman Web* $\rightarrow$ Status: **Belum ada PJ** $\rightarrow$ Tombol **`+ Undang PJ`**.
  3. *Algoritma Pemrograman* $\rightarrow$ Status: **Belum ada PJ** $\rightarrow$ Tombol **`+ Undang PJ`**.

---

### Modal Pop-up: "Undang PJ"
*Muncul saat KM menekan tombol `+ Undang PJ` di salah satu mata kuliah.*

* **Tujuan:** Menghubungkan mahasiswa dengan mata kuliah yang akan dikelolanya.
* **Elemen Tampilan:**
  - **Judul Modal:** "Undang Penanggung Jawab Mata Kuliah"
  - **Pilihan Mata Kuliah (Dropdown):** Otomatis terpilih mata kuliah yang diklik tadi, tetapi KM tetap bisa menggantinya jika perlu.
  - **Input Field:** Nomor WhatsApp calon PJ.
  - **Tombol Utama:** "Buat Tautan Undangan"
* **Tampilan Setelah Tombol Diklik:**
  - Kotak tautan undangan khusus PJ: `app.kampus.ac.id/invite/pj-web-xyz789`
  - Tombol **"Salin Tautan"** dan tombol **"Kirim ke WhatsApp"**.
  - Catatan: *"PJ yang diundang hanya memiliki hak akses untuk mengelola tugas dan jadwal pada mata kuliah yang dipilih."*

---

## 4. Halaman Aktivasi Akun (Terbuka saat Link WhatsApp Diklik)

Kedua link di atas akan membuka halaman aktivasi di browser ponsel atau laptop calon pengurus.

### A. Halaman Aktivasi KM
* **Kartu Info Penugasan (Read-Only di Bagian Atas):**
  - Judul: *"Selamat Datang di Bot Jadwal!"*
  - Badge Peran: **Ketua Murid (KM)**
  - Nama Kelas: **D4 Teknik Informatika 2024 A**
  - Semester: **Semester Ganjil 2026/2027**
  - Nomor WhatsApp Terdaftar: (Nomor yang diundang Admin)
* **Formulir Aktivasi:**
  - Input: Nama Lengkap / Nama Tampilan.
  - Input: Kata Sandi Baru (minimal 12 karakter).
  - Input: Konfirmasi Kata Sandi.
  - Tombol: **"Aktifkan Akun & Masuk Kelas"**.
* **Kondisi Tautan Hangus (Lewat 7 Hari atau Sudah Dipakai):**
  - Form disembunyikan.
  - Pesan jelas: *"Tautan undangan sudah kedaluwarsa atau sudah digunakan. Silakan hubungi Administrator untuk meminta tautan baru."*

---

### B. Halaman Aktivasi PJ
* **Kartu Info Penugasan (Sangat Jelas & Menonjol):**
  - Judul: *"Undangan Penanggung Jawab Mata Kuliah"*
  - Pesan utama: **"Anda diundang sebagai PJ Pemrograman Web di Kelas D4 Teknik Informatika 2024 A"**
  - Keterangan: *"Diundang oleh Ketua Murid (KM)"*
* **Formulir Aktivasi:**
  - Input: Nama Lengkap.
  - Input: Kata Sandi Baru (minimal 12 karakter).
  - Input: Konfirmasi Kata Sandi.
  - Tombol: **"Aktifkan Akun & Masuk"**.
* **Opsi Tambahan (Jika PJ Sudah Punya Akun Sebelumnya):**
  - Di bawah form ada tombol: *"Sudah punya akun sebelumnya? Masuk di sini untuk menambahkan mata kuliah ini ke akun Anda."*
* **Kondisi Tautan Hangus:**
  - Pesan jelas: *"Tautan undangan sudah tidak berlaku. Silakan hubungi Ketua Murid (KM) kelas Anda untuk meminta tautan baru."*

---

## 5. Halaman Login Pengurus
*Halaman masuk harian untuk Admin, KM, dan PJ yang sudah memiliki akun aktif.*

* **Elemen Tampilan:**
  - Judul: **"Masuk Pengurus"**
  - Input: Nomor WhatsApp.
  - Input: Kata Sandi (ada tombol intip kata sandi).
  - Tombol: **"Masuk"**.
  - Link: *"Lupa kata sandi?"*.
  - **Catatan Edukasi di Bagian Bawah:**
    > *"Mahasiswa tidak perlu login. Untuk melihat jadwal kuliah dan daftar tugas, silakan buka tautan portal kelas Anda."*
* **State Error:**
  - Salah kata sandi: *"Nomor atau kata sandi tidak sesuai"*.
  - Salah 5 kali berturut-turut: *"Terlalu banyak percobaan gagal. Akun dibekukan sementara selama 15 menit."*

---

## 6. Layar Pemilihan Peran & Kelas (Context Switcher)
*Hanya muncul tepat setelah login **jika** satu orang memiliki lebih dari satu peran/tugas.*
*(Contoh kasus: Mahasiswa yang menjadi KM di kelasnya sendiri, sekaligus menjadi PJ Asisten di kelas adik tingkat).*

* **Isi Tampilan:**
  - Judul: **"Pilih Ruang Kerja"**
  - Sub-judul: *"Akun Anda terdaftar pada beberapa peran. Pilih ruang kerja yang ingin Anda kelola:"*
  - **Pilihan Kartu:**
    - **Kartu 1:** Ketua Murid (KM) — *Kelas D4 TI 2024 A (Semua Mata Kuliah)* $\rightarrow$ Tombol *"Buka Ruang Kerja"*.
    - **Kartu 2:** Penanggung Jawab (PJ) — *Kelas D4 TI 2025 B (Mata Kuliah: Pemrograman Web)* $\rightarrow$ Tombol *"Buka Ruang Kerja"*.
* **Komponen di Header / Navigasi Atas Dashboard:**
  - Di sudut kanan atas dashboard (samping profil pengguna), terdapat indikator peran aktif saat ini (misal: `KM - D4 TI 2024 A`).
  - Ada menu dropdown: **"Ganti Peran / Kelas"** agar pengguna bisa berpindah kelas kapan saja tanpa perlu logout.
