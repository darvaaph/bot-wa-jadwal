# 📢 Panduan & Materi Open Recruitment Tim Proyek
# Bot WhatsApp Jadwal Kuliah & Web Admin Dashboard

Dokumen ini berisi cetak biru lengkap pelaksanaan **Open Recruitment** untuk membangun tim pengembang perdana. Sebagai Project Manager (PM), Anda dapat menggunakan dokumen ini sebagai acuan menyusun formulir pendaftaran, pesan broadcast, dan panduan wawancara calon anggota tim.

---

## 📑 Daftar Isi
1. [Tentang Proyek (Elevator Pitch)](#1-tentang-proyek-elevator-pitch)
2. [Formasi & Peran yang Dibutuhkan](#2-formasi--peran-yang-dibutuhkan)
3. [Keuntungan Bergabung (Benefits & Value Proposition)](#3-keuntungan-bergabung-benefits--value-proposition)
4. [Jadwal & Alur Seleksi](#4-jadwal--alur-seleksi)
5. [Draf Formulir Pendaftaran (Google Form / Tally)](#5-draf-formulir-pendaftaran-google-form--tally)
6. [Draf Pesan Broadcast / Publikasi](#6-draf-pesan-broadcast--publikasi)

---

## 1. 🎯 Tentang Proyek (Elevator Pitch)

> **"Bot Jadwal & Asisten Kelas Pintar"** adalah ekosistem digital asisten mahasiswa yang menggabungkan kemudahan interaksi WhatsApp dengan keandalan Web Admin Dashboard. Bot ini telah berjalan aktif di cloud (Microsoft Azure) melayani informasi jadwal harian otomatis, pelacak tenggat tugas (*deadline tracker*), perpustakaan tautan perkuliahan, dan fleksibilitas jadwal pengganti (*override*) untuk 19 kelas perkuliahan.
>
> Proyek ini sedang bertransformasi menuju **Versi 2.0 (Web Admin Dashboard Visual)** untuk memberikan pengalaman manajemen kelas yang jauh lebih mudah dan modern bagi Ketua Tingkat (Komti) dan pengurus kelas.

---

## 2. 👥 Formasi & Peran yang Dibutuhkan

Total kebutuhan tim baru: **4–5 orang**.

### 1. 🎨 UI/UX Designer (1 Orang)
* **Tanggung Jawab:**
  - Menerjemahkan spesifikasi produk di [DASHBOARD_PRD.md](DASHBOARD_PRD.md) menjadi desain antarmuka web (Figma).
  - Merancang sistem desain (Design Tokens, Typography, Dark-Mode Component Library).
  - Menyusun prototype interaktif versi Desktop dan Mobile Web yang intuitif dan ramah pengguna.
* **Kriteria:**
  - Memahami dasar-dasar UI/UX Design dan mahir menggunakan Figma.
  - Paham konsep *responsive layout*, *visual hierarchy*, dan *design system*.
  - Nilai tambah jika memiliki portofolio desain web/dashboard.

### 2. 💻 Frontend Developer (1–2 Orang)
* **Tanggung Jawab:**
  - Mengimplementasikan desain Figma ke dalam antarmuka web modern dan responsif.
  - Menghubungkan antarmuka web ke endpoint REST API backend.
  - Menjaga performa loading cepat dan interaksi pengguna yang mulus.
* **Kriteria:**
  - Menguasai HTML5, CSS3/Tailwind CSS, dan JavaScript (ES6+).
  - Paham cara mengonsumsi REST API (`fetch` / `axios`) dan menangani *state* antarmuka.
  - Familiar dengan Git dan alur kerja kolaborasi GitHub.

### 3. ⚙️ Backend Developer (Golang) (1 Orang)
* **Tanggung Jawab:**
  - Melengkapi dan mengoptimalkan endpoint REST API di modul `internal/api/`.
  - Mengelola query database SQLite (WAL mode) untuk operasi CRUD tugas, jadwal, tautan, dan setting kelas.
  - Menulis rangkaian pengujian unit otomatis (*unit test*) untuk memastikan reliabilitas sistem.
* **Kriteria:**
  - Memahami dasar-dasar pemrograman Go (Golang) dan konsep RESTful API.
  - Memahami dasar basis data relasional (SQL).
  - Tertarik mendalami arsitektur perangkat lunak modular (*Standard Go Project Layout*).

### 4. 🧪 QA / Software Tester (1 Orang)
* **Tanggung Jawab:**
  - Merancang skenario pengujian fungsional untuk bot WhatsApp dan Web Dashboard.
  - Melakukan pengujian intensif (*black-box testing*, *edge cases*, *input validation*).
  - Mendokumentasikan dan melaporkan bug secara terstruktur di GitHub Issues.
* **Kriteria:**
  - Memiliki ketelitian tinggi dan rasa ingin tahu yang besar terhadap celah kegagalan sistem.
  - Mampu menuliskan laporan bug yang jelas (*Steps to Reproduce*, *Expected vs Actual Result*).
  - Tertarik mempelajari dasar-dasar automated testing.

---

## 3. 🎁 Keuntungan Bergabung (Benefits & Value Proposition)

Bagi mahasiswa yang bergabung, ini bukan sekadar tugas kelompok biasa, melainkan **batu loncatan portofolio riil**:
1. 🚀 **Proyek Nyata & Berjalan di Produksi:** Kode yang Anda buat bukan sekadar tugas yang disimpan di folder laptop, melainkan aktif di-deploy di Microsoft Azure dan digunakan oleh mahasiswa nyata.
2. 💼 **Pengalaman Kultur Industri (Startup Simulator):** Belajar alur kerja profesional: Git Flow (Pull Request & Code Review), Clean Architecture, Definition of Done, dan komunikasi tim asinkron.
3. ⭐ **Portofolio Unggulan untuk CV & LinkedIn:** Memiliki kontribusi nyata di repositori open source yang dapat diverifikasi oleh rekruter saat melamar magang atau kerja.
4. 📜 **Sertifikat Kontributor & Rekomendasi:** Mendapatkan sertifikat kontribusi resmi serta rekomendasi personal dari Project Manager/Lead Developer.

---

## 4. 📅 Jadwal & Alur Seleksi

* **Pendaftaran Dibuka:** [Tentukan Tanggal, cth: 10 – 17 September 2026]
* **Review Berkas & Portofolio:** [cth: 18 – 19 September 2026]
* **Wawancara Santai (Online 15 Menit):** [cth: 20 – 21 September 2026]
* **Pengumuman Tim & Onboarding Kickoff:** [cth: 22 September 2026]
* **Durasi Proyek (Sprint v2.0):** 4 – 6 Minggu (estimasi komitmen 3–5 jam/minggu).

---

## 5. 📋 Draf Formulir Pendaftaran (Google Form / Tally)

### Bagian 1: Data Diri
1. Nama Lengkap:
2. NIM & Jurusan / Angkatan:
3. Nomor WhatsApp & Username Discord / Telegram:
4. Link Profil LinkedIn (opsional):

### Bagian 2: Minat & Kemampuan
5. **Posisi yang diminati:**
   - [ ] UI/UX Designer
   - [ ] Frontend Developer
   - [ ] Backend Developer (Go)
   - [ ] QA / Software Tester
6. **Link Portofolio / Karya Terbaik:**
   *(Link GitHub untuk Developer, link Figma/Dribbble untuk Designer, atau dokumen/proyek lampau untuk QA).*
7. **Ceritakan pengalaman teknis atau proyek yang pernah Anda kerjakan:** (1–2 paragraf ringkas).

### Bagian 3: Komitmen & Visi
8. **Berapa jam per minggu yang realistis dapat Anda alokasikan untuk proyek ini?**
   - ( ) 3 – 5 jam / minggu
   - ( ) 5 – 8 jam / minggu
   - ( ) > 8 jam / minggu
9. **Apa motivasi utama Anda ingin bergabung di proyek Bot Jadwal & Web Dashboard ini?**
10. **Pertanyaan / Hal yang ingin Anda tanyakan ke tim inisiator (opsional):**

---

## 6. 📢 Draf Pesan Broadcast / Publikasi

*(Dapat disesuaikan untuk disebar di grup WhatsApp angkatan, channel Discord kampus, atau media sosial).*

```text
🚀 [CALL FOR TEAM] Proyek Asisten Kelas & Web Admin Dashboard v2.0! 🤖✨

Halo teman-teman! 👋
Kalian pasti sudah familiar dengan bot WhatsApp jadwal kuliah yang biasa mengingatkan jadwal dan tugas di kelas kita. 

Untuk semester ini, proyek asisten kelas ini sedang bersiap naik kelas ke Versi 2.0 dengan membangun "Web Admin Dashboard Visual" yang akan di-deploy di cloud server Microsoft Azure! ☁️

Buat kalian yang ingin mengasah skill, belajar alur kerja standar industri (Git Pull Request, Code Review, Modular Architecture), dan membangun PORTFOLIO NYATA untuk modal magang/kerja, yuk gabung ke tim inti pengembang!

🔥 Posisi yang Dibuka:
1. 🎨 UI/UX Designer (Figma, Design System, Responsive Mockup)
2. 💻 Frontend Developer (HTML/CSS/Tailwind/JS, REST API Integration)
3. ⚙️ Backend Developer (Golang, SQLite WAL, REST API Server)
4. 🧪 QA / Software Tester (Black-box testing, bug reporting, test cases)

💡 Yang Akan Kalian Dapatkan:
✅ Pengalaman nyata mengelola aplikasi live di server cloud.
✅ Simulasi kultur kerja tim startup/software house (GitHub Flow & Sprint).
✅ Kontribusi nyata yang sangat bernilai di CV dan LinkedIn.
✅ Sertifikat kontributor resmi dari inisiator proyek.

📌 Syarat Utama:
- Mahasiswa aktif yang memiliki kemauan belajar tinggi & komunikatif.
- Memiliki komitmen waktu 3–5 jam per minggu.
- Memiliki dasar kemampuan sesuai peran yang dipilih.

🔗 Link Pendaftaran:
👉 [MASUKKAN LINK GOOGLE FORM / TALLY DI SINI]

⏳ Batas Pendaftaran: [Tentukan Tanggal, misal: Minggu, 20 September 2026]
Kuota sangat terbatas (hanya 1–2 orang per peran) agar koordinasi tim tetap solid dan intensif.

Sampai jumpa di tim, mari kita bangun portofolio keren bersama! 🚀
```
