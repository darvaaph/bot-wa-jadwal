# Panduan Desain: Ruang Kerja Pengurus — PJ, KM, dan System Admin (Batch 3)

Panduan ini adalah acuan kerja untuk tim UI/UX dalam merancang tampilan dashboard pengurus setelah login. Dokumen ini menjelaskan perbedaan hak akses dan tampilan antara **Penanggung Jawab (PJ)**, **Ketua Murid (KM)**, dan **System Administrator**.

---

## 1. Peta Perbedaan Hak Akses & Menu

Agar tim UI/UX tidak bingung membedakan menu untuk masing-masing peran, berikut perbandingannya:

| Fitur / Menu | Penanggung Jawab (PJ) | Ketua Murid (KM) | System Administrator |
|---|---|---|---|
| **Cakupan Kerja** | Terbatas pada 1–2 mata kuliah yang ditugaskan | Mengelola seluruh mata kuliah di 1 kelas | Mengelola seluruh kampus & sistem |
| **Kelola Tugas** | Menambah & mengubah tugas pada mata kuliahnya saja | Melihat semua tugas kelas + **Antrean Pemeriksaan Tugas dari PJ** | Hanya melihat riwayat tugas jika ada kendala |
| **Kelola Jadwal** | Mengusulkan kelas pengganti pada jam mata kuliahnya | Mengatur jadwal reguler mingguan & jadwal kelas pengganti | Mengatur data master ruangan kuliah kampus |
| **Kelola Anggota** | Tidak memiliki menu ini | Mengundang, mengganti, atau mencabut PJ | Membuat kelas & mengundang KM |
| **Pengaturan Kelas** | Tidak memiliki menu ini | Mengganti kode PIN kelas & waktu notifikasi WhatsApp | Konfigurasi server & bot WhatsApp |

---

## 2. Tampilan Ruang Kerja Penanggung Jawab (PJ)

PJ adalah mahasiswa yang bertugas mengelola informasi untuk mata kuliah tertentu saja (misal: *Pemrograman Web*).

### A. Navigasi Menu PJ (Sidebar Ringkas)
1. **Ringkasan PJ (Dashboard Beranda)**
2. **Kelola Tugas** (Fokus utama PJ)
3. **Perubahan Jadwal** (Khusus jam mata kuliahnya)
4. **Materi Mata Kuliah**

---

### B. Layar Penting pada Tampilan PJ

#### 1. Form Tambah / Ubah Tugas
*Halaman tempat PJ memasukkan tugas baru untuk kelas.*
* **Field Mata Kuliah (Otomatis Terkunci):** Dropdown hanya menampilkan mata kuliah yang dipegang oleh PJ tersebut. PJ tidak bisa memilih mata kuliah milik orang lain.
* **Field Input Tugas:**
  - Judul Tugas (misal: *Tugas 2 - Perancangan ERD*).
  - Deskripsi & Instruksi Pengerjaan (editor teks untuk instruksi lengkap).
  - Tanggal & Jam Deadline (lengkap dengan pemilih tanggal/waktu).
  - Tempat Pengumpulan (Input link Google Drive / Classroom atau catatan teks fisik).
* **Dua Tombol Aksi di Bawah Form:**
  - Tombol **"Simpan Draf"**: Tugas disimpan secara privat dan belum muncul di portal mahasiswa.
  - Tombol **"Publikasikan Tugas"**: Tugas langsung muncul di portal mahasiswa dan otomatis masuk ke antrean pemeriksaan KM.

#### 2. Status Tugas di Tampilan PJ
Setiap tugas di daftar tugas PJ memiliki 2 penanda status yang jelas:
* **Status Publikasi:** `Draf` (belum tayang) atau `Terbit` (sudah bisa dibaca mahasiswa).
* **Status Pemeriksaan KM:**
  - `Menunggu Pemeriksaan`: Tugas sudah terbit tapi belum dicek KM.
  - `Disetujui`: KM sudah memeriksa dan menyetujui tugas.
  - `Perlu Koreksi`: KM meminta PJ merevisi instruksi atau deadline tugas (ada catatan revisi dari KM).

---

## 3. Tampilan Ruang Kerja Ketua Murid (KM)

KM adalah pengelola utama kelas. Tampilannya lebih lengkap karena KM bertanggung jawab atas seluruh mata kuliah dan jadwal kelas.

### A. Navigasi Menu KM
1. **Ringkasan Kelas (Dashboard Utama)**
2. **Kelola Tugas** (Dilengkapi tab *Antrean Pemeriksaan*)
3. **Kelola Jadwal** (Jadwal reguler mingguan & jadwal pengganti)
4. **Anggota & Tim** (Tempat mengundang & mengelola PJ)
5. **Materi & Tautan Kelas**
6. **Pengaturan Kelas** (PIN portal & setelan WhatsApp)

---

### B. Layar Penting pada Tampilan KM

#### 1. Layar Antrean Pemeriksaan Tugas (*Review Queue*)
*Tempat KM meninjau tugas-tugas yang baru saja dipublikasikan oleh para PJ.*
* **Tujuan:** Memastikan tidak ada bentrok deadline antar-tugas dan instruksi tugas sudah jelas bagi mahasiswa sekelas.
* **Tampilan Antrean:**
  - Berupa daftar kartu tugas dari berbagai mata kuliah yang statusnya *Menunggu Pemeriksaan*.
  - Menampilkan nama mata kuliah, nama PJ pembuatnya, judul tugas, dan deadline.
* **Tombol Aksi KM pada Tiap Tugas:**
  - Tombol Hijau: **"Setujui"** $\rightarrow$ Status tugas menjadi disetujui.
  - Tombol Oranye: **"Minta Koreksi"** $\rightarrow$ Muncul pop-up untuk menuliskan pesan ke PJ (misal: *"Tolong deadlinenya diundur 1 hari karena hari Kamis ada UTS"*). Tugas otomatis ditarik sementara dari portal mahasiswa agar diperbaiki oleh PJ.
  - Tombol Merah: **"Batalkan Tugas"** $\rightarrow$ Untuk tugas yang salah buat atau dibatalkan dosen.

#### 2. Layar Kelola Jadwal (Jadwal Reguler & Perubahan)
* **Tab 1: Pola Jadwal Reguler:** Tabel jadwal mingguan kelas dari Senin sampai Jumat.
* **Tab 2: Perubahan Jadwal (Kelas Pengganti / Libur):**
  - Tombol `+ Buat Perubahan Jadwal`.
  - Pilihan jenis perubahan:
    - *Kelas Pengganti* (mengganti jam kuliah yang kosong).
    - *Kelas Tambahan* (sesi kuliah ekstra).
    - *Diliburkan / Kosong* (dosen berhalangan hadir).
  - KM bisa memilih ruangan pengganti dan tanggal pelaksanaan.

#### 3. Layar Pengaturan Kelas
* **Pengaturan Akses Portal:**
  - Toggle pilihan: *"Gunakan Link Terbuka"* atau *"Wajibkan Kode PIN 6-Digit"*.
  - Jika mode PIN aktif: Menampilkan 6 digit PIN kelas saat ini dan tombol **"Ganti Kode PIN Baru"** (jika PIN lama bocor ke luar kelas).
* **Pengaturan Pesan WhatsApp:**
  - Jam pengiriman pengingat jadwal otomatis (misal: kirim setiap jam 06:30 pagi).
  - Waktu pengingat tugas (misal: kirim pengingat H-1 sebelum deadline jam 19:00).

---

## 4. Tampilan Ruang Kerja System Administrator

Administrator adalah pengelola tingkat kampus yang berfokus pada kestabilan sistem, bot WhatsApp, dan master data.

### A. Navigasi Menu Admin
1. **Dashboard Ringkasan Sistem**
2. **Daftar Kelas & KM**
3. **Status Bot WhatsApp**
4. **Antrean Pesan**
5. **Master Ruangan Kampus**
6. **Master Mata Kuliah Kampus**
7. **Kelola Pengguna Sistem**
8. **Backup & Pemulihan Data**

---

### B. Layar Penting pada Tampilan Administrator

#### 1. Layar Mode Dukungan Kelas (*Support Mode*)
*Ketika ada KM yang kesulitan atau ada komplain kelas, Admin bisa langsung melihat isi kelas tersebut.*
* **Cara Membuka:** Admin buka menu **Daftar Kelas** $\rightarrow$ klik kelas terkait $\rightarrow$ klik tombol **"Lihat Tampilan Kelas (Mode Dukungan)"**.
* **Tampilan:**
  - Admin langsung melihat halaman kelas persis seperti yang dilihat KM (jadwal dan tugas).
  - Di bagian paling atas ada banner penanda berwarna kuning/oranye:
    > **Mode Dukungan Admin:** Anda sedang melihat data *Kelas D4 TI 2024 A*. Semua perubahan yang Anda buat dicatat atas nama Admin.
  - Ada tombol **"Keluar dari Mode Dukungan"** untuk kembali ke panel admin utama.

#### 2. Layar Status Bot WhatsApp & Scan QR
*Pusat kendali koneksi bot WhatsApp kampus.*
* **Status Terhubung:** Menampilkan nomor HP bot server, status baterai/server, dan tombol uji kirim pesan.
* **Status Terputus:** Muncul kode QR WhatsApp Web besar di tengah layar agar Admin bisa langsung scan dari HP server untuk menghubungkan kembali.

#### 3. Layar Master Ruangan Kampus
*Daftar seluruh ruangan kuliah yang bisa dipakai untuk kelas pengganti.*
* Tabel memuat: Nama Gedung, Lantai, Nama Ruangan (misal: *Lab Komputer 3*), Kapasitas Kursi, dan Fasilitas (Proyektor, AC, Komputer).
* Tombol `+ Tambah Ruangan Baru` dan tombol ubah/nonaktifkan ruangan.

---

## 5. Ringkasan Desain untuk Tim UI/UX

1. **Kerangka Layout & Navigasi Responsif:**
   - **Tampilan Desktop:** Menggunakan struktur layout standar: **Sidebar Menu di Kiri** + **Bar Atas (Header)** + **Area Konten Utama di Kanan**.
   - **Tampilan HP (Mobile):** Menggunakan **Bottom Navigation Bar** dengan 4 menu utama: `Ringkasan`, `Jadwal`, `Tugas`, dan menu `Lainnya` (membuka menu drawer/pop-up untuk Anggota, Materi, dan Pengaturan).
2. **Pembeda Peran yang Jelas:**
   - Cantumkan badge peran dan nama kelas di pojok kanan atas samping profil (misal: `PJ - Pemrograman Web` atau `KM - TI 2024 A`).
3. **Pembeda Status yang Tegas:**
   - Jangan hanya gunakan warna, sertakan teks status yang jelas (`Draf`, `Terbit`, `Menunggu Review`, `Disetujui`).
