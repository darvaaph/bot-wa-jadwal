# 📖 Panduan Lengkap & Dokumentasi Penggunaan
# Bot WhatsApp Jadwal Kuliah & Deadline Tracker

Selamat datang di panduan resmi penggunaan **Bot WhatsApp Jadwal Kuliah & Asisten Tugas Mahasiswa**. Bot ini dikembangkan menggunakan bahasa pemrograman **Go (Golang)** dengan *library* modern `whatsmeow` dan database SQLite lokal (`modernc.org/sqlite`).

---

## 📑 Daftar Isi
1. [Sekilas Tentang Bot](#1-sekilas-tentang-bot)
2. [Prinsip Desain & Arsitektur Cerdas](#2-prinsip-desain--arsitektur-cerdas)
3. [Daftar Fitur Lengkap](#3-daftar-fitur-lengkap)
4. [Tabel Lengkap Perintah (Cheat Sheet)](#4-tabel-lengkap-perintah-cheat-sheet)
5. [Panduan Penggunaan Fitur Jadwal Kuliah](#5-panduan-penggunaan-fitur-jadwal-kuliah)
6. [Panduan Penggunaan Pengingat Otomatis Pagi (`!reminder`)](#6-panduan-penggunaan-pengingat-otomatis-pagi-reminder)
7. [Panduan Penggunaan Deadline Tracker Tugas (`!tugas`)](#7-panduan-penggunaan-deadline-tracker-tugas-tugas)
8. [Panduan Penggunaan Jadwal Pengganti Sementara (`!pindah`, `!kosong`, `!kuliahganti`, `!libur`)](#8-panduan-penggunaan-jadwal-pengganti-sementara-pindah-kosong-kuliahganti-libur)
9. [Panduan Penggunaan Multi-Kelas (`!kelas` & `!setkelas`)](#9-panduan-penggunaan-multi-kelas-kelas--setkelas)
10. [Perbedaan Akses: Grup vs Chat Pribadi (DM)](#10-perbedaan-akses-grup-vs-chat-pribadi-dm)
11. [Format Penulisan Tenggat Waktu (Natural Time Parser)](#11-format-penulisan-tenggat-waktu-natural-time-parser)
12. [Cara Menjalankan & Menguji Bot di Komputer](#12-cara-menjalankan--menguji-bot-di-komputer)
13. [Kestabilan & Keamanan Sistem (Technical Reliability)](#13-kestabilan--keamanan-sistem-technical-reliability)
14. [Cetak Biru Web Admin Dashboard (Future Roadmap)](#14-cetak-biru-web-admin-dashboard-future-roadmap)

---

## 1. 🚀 Sekilas Tentang Bot

Bot WhatsApp ini bertindak sebagai **asisten virtual kelas** yang memudahkan mahasiswa dalam:
* Mengecek jadwal kuliah harian, mingguan, maupun kuliah yang sedang berlangsung secara instan.
* Menerjemahkan inisial kode dosen dan nama ruangan laboratorium/gedung.
* Mengirimkan broadcast jadwal otomatis setiap hari kuliah pukul **06:30 WIB** ke grup kelas.
* Melacak tugas-tugas kuliah (*deadline tracker*) dengan indikator hitung mundur (*countdown*) dan peringatan otomatis jika ada tugas mendesak (*urgent*).
* Mengakomodasi perubahan jadwal sementara (*overrides*) seperti kelas ditiadakan, pindah jam, atau libur nasional tanpa merusak jadwal master.
* Melayani banyak kelas paralel sekaligus (*multi-tenant*) cukup dengan satu nomor bot WhatsApp.

---

## 2. 🧠 Prinsip Desain & Arsitektur Cerdas

### A. Strategi Hybrid (Anti-Spam Grup & Fleksibel di DM)
* **Di Grup WhatsApp:** Setiap perintah **WAJIB** diawali dengan simbol prefix (`!`, `/`, atau `#`). Hal ini mencegah bot merespons obrolan biasa anggota grup (misal saat ada yang mengobrol kata *"senin"* atau *"besok"*).
* **Di Chat Pribadi (DM):** Mahasiswa bebas mengetik tanpa prefix (contoh: cukup ketik `hari ini`, `besok`, `tugas`, atau `kelas`).

### B. Simulasi Manusiawi & Anti-Ban WhatsApp
* **Typing Indicator (`ChatPresenceComposing`):** Setiap merespons perintah, bot memunculkan status *"sedang mengetik..."* selama 600ms agar interaksi terasa natural dan mengurangi risiko deteksi bot oleh server WhatsApp.
* **Reaksi Emoji Otomatis:**
  * 📅 untuk perintah jadwal perkuliahan.
  * ⏰ untuk perintah pengingat otomatis (`!reminder`).
  * 📝 untuk perintah manajemen tugas (`!tugas`).
  * 🔄 untuk perintah jadwal pengganti (`!pindah`, `!kosong`, `!libur`).
  * 🏫 untuk perintah pengaturan kelas (`!kelas`, `!setkelas`).

### C. Pembatas Garis Pesan Rapi (Mobile Friendly)
* Semua garis pembatas diformat tepat 10 karakter (`──────────`) sehingga tidak terlipat (*wrapping*) berantakan di layar ponsel dengan font besar.

### D. Penyimpanan Data Modular & Persisten
* `data/jadwal/*.json`: Master jadwal perkuliahan modular per kelas (misal: `3a.json`, `3b.json`).
* `tugas.db` (SQLite WAL Mode): Database terpadu untuk catatan tugas (`tasks`), jadwal pengganti sementara (`schedule_overrides`), dan pemetaan kelas grup WhatsApp (`chat_settings`).
* `reminder_groups.json`: Daftar JID grup yang mengaktifkan broadcast pengingat pagi otomatis.
* `sesi_bot.db` (SQLite): Penyimpanan sesi login dan token autentikasi WhatsApp WhatsMeow.

---

## 3. 🌟 Daftar Fitur Lengkap

1. **Jadwal Harian & Mingguan:** Akses jadwal hari tertentu (`!senin`, `!selasa`, dll.), jadwal hari ini (`!hari ini`), jadwal besok (`!besok`), atau seluruh minggu kerja (`!seminggu`).
2. **Kuliah Real-Time (`!next` / `!sekarang`):** Mendeteksi mata kuliah yang sedang aktif saat ini (beserta sisa menit), kuliah yang akan dimulai berikutnya, atau status libur/selesai.
3. **Direktori Lengkap:**
   * `!dosen [Nama/Inisial]` : Mencari data dosen JTK Polban berdasarkan inisial atau nama lengkap.
   * `!ruang [Nama/Kode]` : Mencari letak dan jadwal ruangan tertentu.
   * `!matkul` : Daftar ringkasan mata kuliah semester aktif.
   * `!cari [Kata Kunci]` : Pencarian bebas (*fuzzy search*) di seluruh mata kuliah.
4. **Reload On-The-Fly (`!reload`):** Membaca ulang isi file konfigurasi jadwal seluruh kelas ke memori tanpa perlu mematikan program bot di terminal.
5. **Broadcast Pengingat Pagi (06:30 WIB):** Background worker yang otomatis menyapa grup kelas setiap hari kerja (Senin s.d. Jumat) dengan jadwal hari itu dan daftar tugas mendesak sesuai kelas masing-masing.
6. **Deadline Tracker Tugas (`!tugas`):**
   * Pelacak tugas aktif yang tetap bertahan sampai tenggatnya lewat (tidak dihapus sepihak).
   * Badge urgensi otomatis (`🚨 DEADLINE HARI INI`, `⚠️ DEADLINE BESOK (H-1)`, `⏳ H-X`).
   * Filter cepat `!tugas hari ini` dan `!tugas besok`.
   * Filter per mata kuliah (cth: `!tugas sbd`, `!tugas aljabar`).
   * Perpanjangan tenggat waktu (`!tugas edit [ID] | [tenggat]`).
   * Riwayat dan arsip tugas selesai semesteran (`!tugas riwayat`).
   * Role-based access control (RBAC): Di grup hanya Admin yang dapat mengubah data tugas.
7. **Jadwal Pengganti Sementara (*Schedule Overrides*):**
   * Pergeseran jam kuliah sementara (`!pindah`), pembatalan kelas (`!kosong`), kuliah pengganti (`!kuliahganti`), dan pengumuman libur seharian (`!libur`).
   * Otomatis kedaluwarsa setelah tanggal lewat tanpa perlu diedit balik.
   * Deteksi bentrok jadwal otomatis (*conflict warning*) saat pergeseran jam.
8. **Dukungan Multi-Kelas (*Multi-Tenant*):**
   * 1 bot WhatsApp melayani banyak kelas/jurusan sekaligus.
   * Grup WhatsApp dapat dikunci ke jadwal kelasnya masing-masing (`!setkelas 3A`, `!setkelas 3B`).
   * Broadcast pagi otomatis mengirim jadwal yang sesuai dengan kelas aktif masing-masing grup.

---

## 4. 📊 Tabel Lengkap Perintah (Cheat Sheet)

### Perintah Pengaturan Multi-Kelas (`!kelas`, `!setkelas`, `!resetkelas`)
| Perintah | Di Grup | Di DM Pribadi | Hak Akses di Grup | Penjelasan Singkat |
| :--- | :--- | :--- | :--- | :--- |
| `!daftarkelas` / `!kelas` | `!daftarkelas` | `daftarkelas` | **Semua Anggota** | Melihat status kelas aktif & 19 pilihan kelas yang dikelompokkan per semester |
| `!setkelas [nama_kelas]` | `!setkelas D4-TI-SMT3-A` | `setkelas smt 3 a` | **Khusus Admin** | Menautkan grup/chat ke kelas tertentu (mendukung alias bebas) |
| `!pilihkelas [nama_kelas]` | `!pilihkelas D4-TI-SMT3-A`| `pilihkelas smt 3 a`| **Khusus Admin** | Alias dari perintah `!setkelas` |
| `!resetkelas` | `!resetkelas` | `resetkelas` | **Khusus Admin** | Menghapus setelan kelas chat (kembali ke status *Belum Diatur*) |

### Perintah Jadwal Perkuliahan
| Perintah | Di Grup | Di DM Pribadi | Penjelasan Singkat |
| :--- | :--- | :--- | :--- |
| `!menu` | `!menu` | `menu` | Menampilkan menu navigasi utama yang ringkas |
| `!keyword` / `!help` | `!keyword` | `keyword` | Menampilkan kamus seluruh kata kunci bot |
| `!hari ini` / `!today` | `!hari ini` | `hari ini` | Jadwal kuliah hari ini |
| `!besok` / `!tomorrow` | `!besok` | `besok` | Jadwal kuliah besok hari |
| `!senin` s.d. `!jumat` | `!senin` | `senin` | Jadwal kuliah pada hari yang ditentukan |
| `!seminggu` / `!all` | `!seminggu` | `seminggu` | Rangkuman jadwal kuliah Senin s.d. Jumat |
| `!next` / `!sekarang` | `!next` | `next` | Cek kuliah yang sedang berlangsung / berikutnya |
| `!matkul` | `!matkul` | `matkul` | Daftar seluruh mata kuliah semester ini |
| `!dosen [Inisial/Nama]` | `!dosen MR` | `dosen MR` | Cari informasi dosen (nama, NIP, matkul) |
| `!ruang [Nama/Kode]` | `!ruang D105` | `ruang D105` | Cari jadwal yang memakai ruangan tersebut |
| `!cari [Kata Kunci]` | `!cari basis` | `cari basis` | Cari jadwal berdasarkan kata kunci bebas |
| `!reload` | `!reload` | `reload` | Muat ulang jadwal seluruh kelas dari disk |

### Perintah Pengingat Otomatis (`!reminder`)
| Perintah | Di Grup | Di DM Pribadi | Penjelasan Singkat |
| :--- | :--- | :--- | :--- |
| `!reminder` | `!reminder` | `reminder` | Cek status pengingat di chat/grup saat ini |
| `!reminder on` | `!reminder on` | `reminder on` | Mengaktifkan pengingat jam 06:30 WIB di grup ini |
| `!reminder off` | `!reminder off` | `reminder off` | Mematikan pengingat di grup ini |
| `!reminder test` | `!reminder test`| `reminder test`| Simulasi pengiriman pesan pengingat pagi sekarang |

### Perintah Deadline Tracker Tugas (`!tugas`)
| Perintah | Di Grup | Di DM Pribadi | Hak Akses di Grup |
| :--- | :--- | :--- | :--- |
| `!tugas` | `!tugas` | `tugas` | **Semua Anggota** |
| `!tugas hari ini` | `!tugas hari ini` | `tugas hari ini` | **Semua Anggota** |
| `!tugas besok` | `!tugas besok` | `tugas besok` | **Semua Anggota** |
| `!tugas [matkul]` | `!tugas sbd` | `tugas sbd` | **Semua Anggota** |
| `!tugas riwayat` | `!tugas riwayat` | `tugas riwayat` | **Semua Anggota** |
| `!tugas tambah [M] \| [D] \| [T]` | `!tugas tambah ...` | `tugas tambah ...` | **Khusus Admin** |
| `!tugas edit [ID] \| [Tenggat]` | `!tugas edit 1 \| minggu 23:59` | `tugas edit 1 \| ...` | **Khusus Admin** |
| `!tugas selesai [ID]` | `!tugas selesai 1` | `tugas selesai 1` | **Khusus Admin** |
| `!tugas hapus [ID]` | `!tugas hapus 1` | `tugas hapus 1` | **Khusus Admin** |
| `!tugas bantuan` | `!tugas bantuan` | `tugas bantuan` | **Semua Anggota** |

### Perintah Jadwal Pengganti Sementara (`!pindah`, `!kosong`, `!kuliahganti`, `!libur`)
| Perintah | Di Grup | Di DM Pribadi | Hak Akses di Grup |
| :--- | :--- | :--- | :--- |
| `!pindah [Matkul] \| [Waktu] \| [Ruang]` | `!pindah ...` | `pindah ...` | **Khusus Admin** |
| `!kosong [Matkul] \| [Waktu] \| [Alasan]` | `!kosong ...` | `kosong ...` | **Khusus Admin** |
| `!kuliahganti [Matkul] \| [Waktu] \| [Ruang]`| `!kuliahganti ...`| `kuliahganti ...`| **Khusus Admin** |
| `!libur [Waktu] \| [Keterangan]` | `!libur besok \| HUT RI` | `libur besok \| ...` | **Khusus Admin** |
| `!jadwalganti` | `!jadwalganti` | `jadwalganti` | **Semua Anggota** |
| `!batalganti [ID]` | `!batalganti 1` | `batalganti 1` | **Khusus Admin** |

---

## 5. 📅 Panduan Penggunaan Fitur Jadwal Kuliah

Fitur jadwal kuliah adalah fungsi inti bot yang memungkinkan mahasiswa mengecek agenda perkuliahan kapan saja secara cepat tanpa perlu membuka file PDF atau dokumen gambar manual.

### A. Cek Jadwal Harian & Mingguan
* **Jadwal Hari Ini:**
  * Ketik: `!hari ini` atau `!today`
  * Bot menampilkan daftar lengkap mata kuliah hari ini, jam, ruangan, nama dosen pengampu, dan status kelas pengganti/libur jika ada.
* **Jadwal Besok:**
  * Ketik: `!besok` atau `!tomorrow`
  * Sangat bermanfaat di malam hari untuk mempersiapkan buku, modul praktikum, atau materi kuliah esok hari.
* **Jadwal Hari Tertentu:**
  * Ketik: `!senin`, `!selasa`, `!rabu`, `!kamis`, atau `!jumat`
  * Bot menampilkan jadwal lengkap pada hari yang diminta.
* **Rangkuman Mingguan:**
  * Ketik: `!seminggu` atau `!all`
  * Bot merangkum jadwal perkuliahan dari hari Senin sampai Jumat dalam satu pesan rapi berstruktur.

### B. Melihat Kuliah Real-Time (`!next` / `!sekarang`)
Bot memeriksa waktu saat pesan diterima secara presisi dan mencocokkannya dengan jadwal hari ini:
* **Jika ada kuliah berlangsung:** Bot menampilkan nama mata kuliah, dosen pengampu, ruangan, serta sisa menit sebelum perkuliahan usai.
* **Jika jeda istirahat / kuliah berikutnya masih beberapa waktu lagi:** Bot menampilkan hitung mundur menuju kelas berikutnya (contoh: *Dimulai dalam 45 menit*).
* **Jika semua kuliah hari ini telah usai:** Bot mengonfirmasi bahwa seluruh perkuliahan hari ini sudah selesai.
* **Jika hari libur / akhir pekan:** Bot mengabarkan bahwa hari ini tidak ada agenda perkuliahan.

### C. Direktori Dosen, Ruangan, dan Pencarian Bebas
* **Informasi Dosen (`!dosen [Inisial/Nama]`):**
  * Contoh: `!dosen MR` atau `!dosen supriadi`
  * Menampilkan nama lengkap dosen beserta gelar, NIP/NIDN, dan daftar mata kuliah yang diampu.
* **Informasi Ruangan (`!ruang [Kode/Nama]`):**
  * Contoh: `!ruang D105` atau `!ruang Lab 312`
  * Menampilkan lokasi ruangan dan jadwal penggunaan ruangan tersebut sepanjang minggu.
* **Daftar Seluruh Mata Kuliah (`!matkul`):**
  * Menampilkan ringkasan seluruh mata kuliah yang terdaftar di semester aktif beserta bobot SKS.
* **Pencarian Cerdas Bebas (`!cari [Kata Kunci]`):**
  * Contoh: `!cari basis data` atau `!cari web`
  * Mencari mata kuliah yang relevan di seluruh jadwal kelas.

### D. Memperbarui Master Jadwal On-The-Fly (`!reload`)
Jika terjadi perubahan ruangan atau jam perkuliahan resmi dari jurusan:
1. Buka file JSON kelas yang bersangkutan di folder `data/jadwal/` (misal: `data/jadwal/3a.json` atau `data/jadwal/3b.json`).
2. Edit jam atau ruangan yang diperlukan, lalu simpan file.
3. Di grup WhatsApp atau chat pribadi, ketik:
   ```text
   !reload
   ```
4. Bot akan membaca ulang seluruh file konfigurasi di folder `data/jadwal/` ke dalam memori secara instan. Jadwal baru langsung aktif tanpa perlu mematikan program bot di terminal!

---

## 6. ⏰ Panduan Penggunaan Pengingat Otomatis Pagi (`!reminder`)

Pengingat otomatis pagi berjalan setiap hari **Senin sampai Jumat pukul 06:30 WIB**.

### Cara Mengaktifkan di Grup:
1. Masukkan bot ke dalam grup kelas.
2. Kirim perintah berikut di dalam grup:
   ```text
   !reminder on
   ```
3. Bot akan menyimpan identitas grup ke dalam file `reminder_groups.json`.

### Menguji Tampilan Pengingat:
Untuk memastikan format pengingat sudah sesuai tanpa perlu menunggu jam 06:30 pagi:
```text
!reminder test
```
Bot akan langsung merespons dengan format pesan pengingat pagi lengkap beserta jadwal kuliah hari ini dan seksi peringatan tugas mendesak.

### Integrasi Multi-Kelas pada Broadcast Pagi:
Setiap grup yang mengaktifkan pengingat akan menerima broadcast jadwal yang **terpersonalisasi sesuai kelas aktif grup tersebut** (yang diatur via `!setkelas`). Misalnya, grup kelas 3A akan menerima pengingat jadwal 3A, sedangkan grup kelas 3B menerima pengingat jadwal 3B secara otomatis.

---

## 7. 📝 Panduan Penggunaan Deadline Tracker Tugas (`!tugas`)

Fitur ini dirancang khusus untuk memecahkan masalah umum grup kelas: **tugas yang hilang dari obrolan dan lupa dikerjakan.**

### A. Konsep Tracker: Tetap Terpajang Sampai Tenggat Lewat
Di grup kelas, saat seorang mahasiswa sudah mengumpulkan tugas, tugas tersebut **tidak boleh langsung dihapus**. Mahasiswa lain mungkin belum mengumpulkan. Karena itu:
* Tugas tetap berada di daftar aktif.
* Bot menghitung mundur waktu secara real-time.
* Tugas otomatis diarsipkan jika tenggat waktu sudah terlewati lebih dari 2 hari (H+2).

### B. Format Menambahkan Tugas Baru (`!tugas tambah`)
Format wajib menggunakan pemisah tanda pipa (`|`):
```text
!tugas tambah [Mata Kuliah] | [Deskripsi / Judul Tugas] | [Tenggat Waktu]
```

**Contoh Perintah:**
```text
!tugas tambah SBD | Laporan Praktikum Modul 3 | Jumat 23:59
!tugas tambah Aljabar | Latihan Soal Nilai Eigen | Besok 14:00
!tugas tambah Grafika | Proyek Kelompok OpenGL | 25-09-2026 23:59
```

### C. Format Tampilan Daftar Tugas (`!tugas`):
```text
📋 *DAFTAR TUGAS KELAS*
──────────

*1. [ALJABAR] Latihan Soal Nilai Eigen*
   • Status   : 🚨 *DEADLINE HARI INI* (Sisa ~7 jam)
   • Tenggat  : Hari Ini, 23:59 WIB
   • ID Tugas : #2
   • Oleh     : @628123456789

*2. [SBD] Laporan Praktikum Modul 3*
   • Status   : ⚠️ *DEADLINE BESOK (H-1)*
   • Tenggat  : Besok (Jumat), 23:59 WIB
   • ID Tugas : #1
   • Oleh     : @628123456789

──────────
_Tips: Di grup, tugas tetap terpajang sampai tenggatnya selesai._
```

### D. Filter Cepat Berdasarkan Urgensi
* **Tugas Hari Ini:**
  ```text
  !tugas hari ini
  ```
  *(Hanya menampilkan tugas yang jatuh tempo hari ini)*
* **Tugas Besok:**
  ```text
  !tugas besok
  ```
  *(Hanya menampilkan tugas yang jatuh tempo besok / H-1)*

### E. Filter Tugas per Mata Kuliah & Pencarian Cepat (`!tugas [matkul]` / `!tugas cari [kata]`)
Saat mahasiswa ingin fokus belajar atau mengecek progres 1 mata kuliah tertentu saja (misal menjelang praktikum SBD besok atau kuis Aljabar):
* **Cukup ketik nama atau singkatan matkul:**
  ```text
  !tugas sbd
  !tugas aljabar
  !tugas mtk
  ```
* **Pencarian bebas kata kunci:**
  ```text
  !tugas cari modul 3
  !tugas praktikum
  ```
* **Format Tampilan Luaran:**
  * Header otomatis menyesuaikan: `📋 *DAFTAR TUGAS KELAS: SISTEM BASIS DATA*`
  * Jika belum ada tugas pada matkul tersebut, bot menampilkan pesan melegakan:
    `🎉 *Tidak ada tugas aktif untuk kriteria ini!*`

### F. Mengubah / Memperpanjang Tenggat Tugas (`!tugas edit` / `!tugas mundur`)
Jika dosen memperpanjang atau memundurkan tenggat tugas, Komti tidak perlu menghapus dan membuat ulang tugas dari awal:
* **Ubah Tenggat Waktu Saja:**
  ```text
  !tugas edit [ID] | [Tenggat Baru]
  ```
  *Contoh:*
  ```text
  !tugas edit 1 | Minggu 23:59
  !tugas mundur 2 | 12 sep 20.00
  ```
* **Ubah Deskripsi dan Tenggat Sekaligus:**
  ```text
  !tugas edit [ID] | [Deskripsi Baru] | [Tenggat Baru]
  ```
  *Contoh:*
  ```text
  !tugas edit 1 | Revisi Lapres Modul 1 | Senin 12:00
  ```

### G. Menyelesaikan & Menghapus Tugas
Jika seluruh mahasiswa satu kelas sudah mengumpulkan tugas atau tugas dibatalkan oleh dosen:
* **Tandai Selesai:**
  ```text
  !tugas selesai 1
  ```
* **Hapus Permanen:**
  ```text
  !tugas hapus 1
  ```
*(Ganti angka `1` dengan nomor ID tugas yang tertera pada daftar)*

### H. Riwayat & Arsip Tugas Selesai (`!tugas riwayat` / `!tugas arsip`)
Tugas yang telah ditandai selesai (`!tugas selesai [ID]`) tidak hilang begitu saja, melainkan dipindahkan ke rekam jejak semester:
* **Perintah:**
  ```text
  !tugas riwayat
  !tugas arsip
  ```
* **Format Tampilan:**
  ```text
  📜 *ARSIP & RIWAYAT TUGAS SELESAI*
  ──────────
  *1. ✅ [ALJABAR LINEAR] Latihan Soal Nilai Eigen*
     • Tenggat  : 05 Sep 2026, 23:59 WIB
     • ID Tugas : #2
     • Oleh     : @628123456789
  ```
* **Manfaat:** Sangat berguna bagi mahasiswa menjelang pekan **UTS** dan **UAS** untuk mengulang kembali latihan/tugas yang pernah dikerjakan.

---

## 8. 🔄 Panduan Penggunaan Jadwal Pengganti Sementara (`!pindah`, `!kosong`, `!kuliahganti`, `!libur`)

Dalam dinamika perkuliahan, jadwal seringkali berubah mendadak: dosen berhalangan hadir sehingga kelas ditiadakan, kuliah dimajukan atau diundur jamnya, terdapat kuliah pengganti di hari Sabtu, atau kampus diliburkan karena agenda dies natalis/hari libur nasional.

Fitur **Schedule Overrides** diciptakan untuk mengakomodasi seluruh dinamika tersebut tanpa perlu merusak atau mengubah isi file master jadwal:
* Perubahan hanya berlaku untuk **tanggal/minggu yang ditentukan**.
* **Minggu berikutnya, jadwal otomatis kembali normal** tanpa perlu intervensi manual dari siapapun.
* Terintegrasi langsung dengan pengecekan jadwal harian (`!hari ini`, `!besok`), `!next`, dan pengingat pagi otomatis `!reminder`.

### A. Memindahkan Jam atau Hari Kuliah (`!pindah`)
Digunakan saat dosen meminta jam perkuliahan digeser ke jam lain di hari yang sama, atau digeser ke hari lain pada minggu tersebut.
* **Format Perintah:**
  ```text
  !pindah [Nama Matkul] | [Waktu/Hari Baru & Jam] | [Ruang Baru (Opsional)]
  ```
* **Contoh:**
  * `!pindah aljabar | besok 13:00 | Lab 312`
  * `!pindah sbd | jumat 15:00 - 16:40`
  * `!pindah matdis | 10-09-2026 08:00 - 10:30 | D105`
* **Efek pada Jadwal:**
  * Pada **jadwal asal**: Ditandai coret `~~07:00 - 08:40 WIB~~` dengan status `❌ *KULIAH DIPINDAHKAN* (Dipindah ke: ...)`.
  * Pada **jadwal tujuan**: Otomatis disisipkan blok jadwal baru dengan badge `🔄 [KULIAH PENGGANTI]`.
  * **Minggu depan**: Jadwal otomatis kembali ke jadwal reguler semula.

### B. Menandai Kuliah Ditiadakan / Kosong (`!kosong`)
Digunakan saat jam perkuliahan ditiadakan (misal dosen dinas ke luar kota, sakit, atau kelas diganti tugas mandiri).
* **Format Perintah:**
  ```text
  !kosong [Nama Matkul] | [Hari/Tanggal (Opsional)] | [Alasan (Opsional)]
  ```
* **Contoh:**
  * `!kosong sbd | besok | Dosen dinas luar kota`
  * `!kosong matdis | hari ini`
  * `!kosong aljabar | jumat | Diberi tugas mandiri`
* **Efek:** Pada tanggal yang dipilih, mata kuliah bersangkutan dicoret dan diberi label keterangan alasan ditiadakan.

### C. Menambah Kuliah Pengganti di Hari Libur / Kosong (`!kuliahganti`)
Digunakan untuk menambahkan sesi kuliah baru di luar jadwal reguler (misal kuliah pengganti di hari Sabtu atau hari kerja yang tidak ada mata kuliah tersebut).
* **Format Perintah:**
  ```text
  !kuliahganti [Nama Matkul] | [Hari/Tanggal & Jam] | [Ruang]
  ```
* **Contoh:**
  * `!kuliahganti matdis | sabtu 09:00 - 11:30 | D105`
  * `!kuliahganti sbd | besok 13:00 - 15:30 | Lab 312`
* **Efek:** Bot mengenali sesi perkuliahan tersebut pada tanggal yang ditentukan, menampilkannya saat mahasiswa mengecek `!hari ini`, `!besok`, `!next`, dan menyertakannya di pengingat pagi otomatis.

### D. Menetapkan Hari Libur Seharian (`!libur`)
Digunakan jika ada libur nasional, libur fakultas, atau cuti bersama sehingga seluruh perkuliahan pada hari tersebut ditiadakan secara menyeluruh.
* **Format Perintah:**
  ```text
  !libur [Hari/Tanggal] | [Keterangan/Nama Libur]
  ```
* **Contoh:**
  * `!libur besok | Hari Kemerdekaan RI`
  * `!libur senin | Libur Nasional Maulid Nabi`
  * `!libur 17-08-2026 | HUT RI ke-81`
* **Efek Luar Biasa:**
  * Seluruh perkuliahan pada tanggal tersebut otomatis dinonaktifkan.
  * Mahasiswa yang mengetik `!hari ini`, `!besok`, atau `!next` langsung menerima kartu ucapan libur:
    ```text
    🌴 *HARI INI LIBUR KULIAH*
    Keterangan: Libur Nasional Maulid Nabi
    Selamat beristirahat & berkumpul bersama keluarga! ✨
    ```
  * Pengingat pagi otomatis (06:30 WIB) mengirimkan pesan ucapan selamat berlibur.

### E. Deteksi Bentrok Jadwal Otomatis (*Conflict Warning*) & Konfirmasi Paksa
Saat menjalankan perintah `!pindah` atau `!kuliahganti`, bot secara otomatis memvalidasi apakah jam baru tersebut bertabrakan dengan jadwal mata kuliah lain di hari yang sama:
* **Jika Terjadi Bentrok:** Bot membatalkan perubahan dan menampilkan peringatan bentrok:
  ```text
  ⚠️ *PERINGATAN: TERDETEKSI BENTROK JADWAL!*
  ──────────
  Jam yang dipilih bertabrakan dengan:
  • Matkul : Rekayasa Perangkat Lunak
  • Jam    : 13:00 - 15:30 WIB
  • Ruang  : D105 (Dosen: Pak Bambang)

  Jika memang ingin tetap memindahkan (force), tambahkan kata *paksa* di akhir:
  !pindah aljabar | besok 13:00 | paksa
  ```
* **Opsi Konfirmasi Paksa (`paksa`):** Jika kelas tersebut memang sudah disepakati bersama (misal kelas yang lain sudah kosong), Komti cukup menambahkan kata `paksa` di akhir parameter:
  ```text
  !pindah aljabar | besok 13:00 | Lab 312 | paksa
  !kuliahganti sbd | sabtu 10:00 | Lab 312 | paksa
  ```

### F. Melihat dan Membatalkan Perubahan Jadwal
* **Melihat Seluruh Perubahan Aktif:**
  ```text
  !jadwalganti
  ```
  Bot menampilkan daftar seluruh jadwal pengganti dan status libur yang sedang aktif beserta nomor ID-nya.
* **Membatalkan Perubahan (Kembali Normal Seketika):**
  ```text
  !batalganti [ID]
  ```
  *Contoh:* `!batalganti 1`
  Jadwal langsung kembali normal seketika seperti semula.

---

## 9. 🏫 Panduan Penggunaan Multi-Kelas (`!kelas` & `!setkelas`)

Mulai versi 2.0, bot mendukung arsitektur **Multi-Tenant (Banyak Kelas Paralel)** untuk **D3 & D4 Teknik Informatika (total 19 kelas)**. Satu bot WhatsApp yang sama dapat melayani berbagai kelas paralel sekaligus tanpa saling mencampur atau menimpa jadwal satu sama lain.

### A. Konsep Kerja & Explicit Onboarding (Best Practice)
1. **Explicit Onboarding (Status Awal):** Ketika bot pertama kali dimasukkan ke grup baru atau di-chat secara personal (DM), status kelas obrolan adalah **Belum Diatur**.
   * Jika seseorang mengetik `!hari ini`, `!besok`, `!jadwal`, `!next`, `!matkul`, `!tugas`, atau `!reminder on`, bot tidak akan sembarangan menampilkan jadwal kelas lain.
   * Sebaliknya, bot akan menampilkan pesan ramah **Panduan Onboarding** yang meminta pengguna/admin menentukan kelas terlebih dahulu via `!setkelas [nama_kelas]`.
2. **Format Eksplisit Semester (Bebas Salah Paham):**
   * Format kode kelas distandarkan secara eksplisit: `<PRODI>-SMT<SEMESTER>-<KELAS>` (contoh: `D4-TI-SMT3-A`, `D4-TI-SMT1-B`, `D3-TI-SMT1-A`).
   * **Mengapa demikian?** Jika hanya ditulis `3A`, banyak mahasiswa salah mengira itu adalah "Kelas 3" (Tingkat 3 / Tahun ke-3). Dengan format `SMT3`, status Semester 3 tercantum secara gamblang dan presisi.
   * **Alias Cerdas Tetap Didukung:** Bot dilengkapi pencocok cerdas sehingga pengguna tetap dapat mengetik ringkas seperti `!setkelas smt 3 a`, `!setkelas d4 3 a`, atau bahkan `!setkelas 3a` (otomatis diarahkan ke `D4-TI-SMT3-A`).
3. **Master Jadwal Modular (19 Kelas):** Seluruh kelas D3 dan D4 tersimpan rapi dalam format file JSON di folder `data/jadwal/`:
   * **D4 Sarjana Terapan (12 Kelas):** `D4-TI-SMT1-A` s.d. `1D`, `D4-TI-SMT3-A` s.d. `3D`, `D4-TI-SMT5-A`, `5B`, `D4-TI-SMT7-A`, `7B`.
   * **D3 Diploma (7 Kelas):** `D3-TI-SMT1-A`, `1B`, `D3-TI-SMT3-A`, `3B`, `D3-TI-SMT5-A`, `5B`, `5C`.
4. **Pemetaan Obrolan (*Chat-to-Class Mapping*):** Setiap grup WhatsApp atau obrolan pribadi ditautkan ke kelas tertentu melalui database `tugas.db` (tabel `chat_settings`).
5. **Penyimpanan In-Memory & O(1) Lookup:** Bot menyimpan data seluruh kelas dan pengaturan obrolan dalam memori (*cached*) dengan proteksi `sync.RWMutex`, menjamin pembacaan jadwal tetap instan tanpa lag.

### B. Melihat Status Kelas Aktif & Daftar Pilihan Kelas (Dikelompokkan per Semester)
Untuk mengetahui status kelas aktif di chat ini dan melihat seluruh 19 kelas resmi yang dikelompokkan secara terstruktur:
* **Perintah:**
  ```text
  !daftarkelas
  ```
  *(Alias: `!kelas`)*
* **Format Balasan Bot:**
  ```text
  🏫 *DAFTAR KELAS PERKULIAHAN*
  ──────────
  📌 *Kelas Aktif di Chat Ini:* D4-TI-SMT3-A — _D4 Semester 3 (Transisi) / Kelas 3A_

  Pilihan kelas resmi (dikelompokkan per semester):

  📚 *PROGRAM STUDI D4 TEKNIK INFORMATIKA*
    • *Semester 1:*
      - 🔘 `D4-TI-SMT1-A`
      - 🔘 `D4-TI-SMT1-B`
      - 🔘 `D4-TI-SMT1-C`
      - 🔘 `D4-TI-SMT1-D`
    • *Semester 3:*
      - ✅ *`D4-TI-SMT3-A`* *(Aktif)*
      - 🔘 `D4-TI-SMT3-B`
      - 🔘 `D4-TI-SMT3-C`
      - 🔘 `D4-TI-SMT3-D`
    • *Semester 5:*
      - 🔘 `D4-TI-SMT5-A`
      - 🔘 `D4-TI-SMT5-B`
    • *Semester 7:*
      - 🔘 `D4-TI-SMT7-A`
      - 🔘 `D4-TI-SMT7-B`

  📚 *PROGRAM STUDI D3 TEKNIK INFORMATIKA*
    • *Semester 1:*
      - 🔘 `D3-TI-SMT1-A`
      - 🔘 `D3-TI-SMT1-B`
    • *Semester 3:*
      - 🔘 `D3-TI-SMT3-A`
      - 🔘 `D3-TI-SMT3-B`
    • *Semester 5:*
      - 🔘 `D3-TI-SMT5-A`
      - 🔘 `D3-TI-SMT5-B`
      - 🔘 `D3-TI-SMT5-C`

  ──────────
  💡 *Cara Memilih / Mengatur Kelas:*
  Ketik: `!setkelas [nama_kelas]`
  Contoh: `!setkelas D4-TI-SMT3-A`
  _Tips: Anda juga dapat mengetik santai seperti `!setkelas smt 3 a` atau `!setkelas d4 3 a`._
  ```

### C. Menautkan Grup ke Kelas Tertentu (`!setkelas` / `!pilihkelas`)
Hanya **Admin Grup** (atau pengguna di DM pribadi) yang memiliki wewenang untuk mengubah kelas aktif chat:
* **Format Perintah:**
  ```text
  !setkelas [nama_kelas]
  ```
  *(Alias: `!pilihkelas [nama_kelas]`)*
* **Contoh Penggunaan:**
  * `!setkelas D4-TI-SMT3-A` *(format kanonikal)*
  * `!setkelas smt 3 a` *(alias bebas spasi)*
  * `!setkelas d4 3 a` *(alias prodi + semester)*
  * `!setkelas 3a` *(alias kompatibilitas pendek)*
  * `!setkelas D3-TI-SMT1-A` *(kelas D3)*
  * `!setkelas d3 1 a` *(alias D3 santai)*
* **Karakteristik & Validasi:**
  * Penulisan bersifat *case-insensitive* (`d4-ti-smt3-a` atau `D4-TI-SMT3-A` sama-sama dikenali secara cerdas).
  * Jika nama kelas tidak terdaftar di sistem, bot menolak dan memandu pengguna mengecek `!daftarkelas`.
  * Pengaturan tersimpan permanen di database `tugas.db` (tabel `chat_settings`).

### D. Mereset Pengaturan Kelas (`!resetkelas`)
Jika grup ingin melepaskan kustomisasi kelas dan kembali ke status belum diatur:
* **Format Perintah:**
  ```text
  !resetkelas
  ```
  *(Khusus Admin Grup)*
* Chat akan kembali ke status **Belum Diatur**, dan perintah jadwal berikutnya akan menampilkan panduan onboarding.

### E. Memperbarui Data Jadwal di Server (`!reload`)
Jika ada revisi file jadwal pada server:
1. Perbarui file JSON di folder `data/jadwal/`.
2. Di WhatsApp, kirim perintah:
   ```text
   !reload
   ```
3. Bot akan menyegarkan seluruh 19 kelas dari disk ke memori tanpa perlu me-restart bot.

### F. Pengaruh Pengaturan Kelas terhadap Fitur Lain
Saat kelas aktif di suatu obrolan telah disetel (misal ke `D4-TI-SMT3-A`):
* **Pengecekan Jadwal (`!hari ini`, `!besok`, `!senin`, dll.):** Otomatis menyajikan jadwal kelas tersebut.
* **Kuliah Real-Time (`!next` / `!sekarang`):** Memeriksa perkuliahan kelas tersebut yang sedang berlangsung.
* **Pengingat Tugas (`!tugas`):** Terhubung langsung ke konteks kelas aktif.
* **Pengingat Pagi Otomatis (`!reminder on`):** Broadcast pagi (06:30 WIB) otomatis menyajikan jadwal harian kelas tersebut.
* **Pencarian Dosen & Ruangan (`!dosen`, `!ruang`):** Mencocokkan data pada jadwal kelas tersebut.

---

## 10. 🔒 Perbedaan Akses: Grup vs Chat Pribadi (DM)

Demi menjaga ketertiban grup kelas dari spam atau pengubahan data sepihak yang tidak sah, bot menerapkan sistem kendali akses berbasis peran (*Role-Based Access Control*):

| Fitur / Karakteristik | Di Grup Kelas | Di Chat Pribadi (DM) | Alasan & Rasional |
| :--- | :--- | :--- | :--- |
| **Kebutuhan Simbol Prefix** | Wajib (`!`, `/`, `#`) | Bebas (Bisa tanpa prefix) | Mencegah bot tersulut obrolan santai anggota grup |
| **Cek Jadwal & Status** | Semua Anggota | Bebas | Seluruh mahasiswa berhak mengetahui agenda perkuliahan |
| **Cek Daftar Tugas (`!tugas`)** | Semua Anggota | Bebas | Transparansi daftar tugas bersama untuk seluruh mahasiswa |
| **Tambah / Edit Tugas** | **Khusus Admin Grup** | Bebas | Mencegah tugas fiktif atau spam dari anggota iseng |
| **Selesaikan / Hapus Tugas** | **Khusus Admin Grup** | Bebas | Menjaga agar tugas tidak terhapus sepihak sebelum kumpul |
| **Jadwal Pengganti (`!pindah`, `!libur`)** | **Khusus Admin Grup** | Bebas | Perubahan jadwal kelas harus melalui persetujuan Komti |
| **Pengaturan Kelas (`!setkelas`)** | **Khusus Admin Grup** | Bebas | Mencegah anggota sembarangan mengubah identitas kelas grup |
| **Pengingat Pagi (`!reminder on/off`)** | **Khusus Admin Grup** | Bebas | Hanya pengurus kelas yang mengatur waktu/jadwal siaran |

---

## 11. 🕒 Format Penulisan Tenggat Waktu (Natural Time Parser)

Parser waktu bot dapat mengenali berbagai variasi penulisan bahasa Indonesia:

1. **Kata Relatif Hari:**
   * `hari ini 20:00` / `hari ini 23:59`
   * `besok 12:00` / `besok 14:00`
2. **Nama Hari Kerja:**
   * `senin 23:59`, `selasa 10:00`, `rabu 23:59`, `kamis 15:00`, `jumat 23:59`
3. **Format Tanggal Kalender Standar:**
   * `15-09-2026 23:59`
   * `2026-09-15 23:59`
   * `15/09/2026 23:59`

> **Catatan:** Jika jam tidak dicantumkan (contoh: `!tugas tambah SBD | Lapres | Jumat`), bot otomatis menetapkan batas akhir pada pukul **23:59 WIB**.

---

## 12. 💻 Cara Menjalankan & Menguji Bot di Komputer

### Prasyarat:
* Go (Golang) versi 1.22 atau lebih baru terinstal di komputer/server.
* Koneksi internet aktif untuk menghubungkan klien WhatsApp.

### Menjalankan Bot:
Buka terminal di direktori proyek `f:\Project\bot-jadwal` lalu jalankan:
```bash
go run .
```
Jika baru pertama kali dijalankan atau sesi autentikasi kedaluwarsa, terminal akan menampilkan **QR Code**. Pindai QR Code tersebut melalui menu **Linked Devices (Perangkat Tertaut)** di aplikasi WhatsApp ponsel Anda.

### Menjalankan Pengujian Otomatis (Unit Test):
Bot dilengkapi rangkaian pengujian unit otomatis (*unit test suite*) yang menguji seluruh logika inti tanpa memerlukan koneksi WhatsApp nyata:
```bash
go test -v .
```

Rangkaian 8 Test Suite Komprehensif:
1. `TestOverrideManager`: Pengujian pergeseran jam, kelas kosong, kuliah pengganti, hari libur, dan deteksi bentrok jadwal.
2. `TestSchedule`: Pengujian parser jadwal harian, mingguan, pencarian dosen/ruangan, dan fuzzy search.
3. `TestTaskManager`: Pengujian pembuatan tugas, filter mata kuliah, perpanjangan deadline, dan riwayat tugas SQLite.
4. `TestClassManager`: Pengujian multi-JSON loader, normalisasi kode kelas, validasi fallback, dan hot-reload.
5. `TestChatSettingsManager`: Pengujian tabel relasi `chat_settings` di SQLite, in-memory caching, dan handler `!setkelas`/`!daftarkelas`.
6. `TestReminderIntegration`: Pengujian broadcast jadwal pagi multi-kelas, resolusi kelas aktif per grup, dan pengingat tugas mendesak.
7. `TestDatabaseInit`: Pengujian pembuatan skema tabel SQLite, integritas WAL mode, dan penutupan koneksi bersih.
8. `TestUtils`: Pengujian parser prefix pesan, pembersih perintah, normalisasi waktu, dan fungsi string helper.

Output pengujian yang diharapkan:
```text
=== RUN   TestChatSettingsManager
--- PASS: TestChatSettingsManager (0.01s)
=== RUN   TestClassManager
--- PASS: TestClassManager (0.00s)
=== RUN   TestDatabaseInit
--- PASS: TestDatabaseInit (0.01s)
=== RUN   TestOverrideManager
--- PASS: TestOverrideManager (0.02s)
=== RUN   TestReminderIntegration
--- PASS: TestReminderIntegration (0.00s)
=== RUN   TestSchedule
--- PASS: TestSchedule (0.00s)
=== RUN   TestTaskManager
--- PASS: TestTaskManager (0.04s)
=== RUN   TestUtils
--- PASS: TestUtils (0.00s)
PASS
ok  	bot-jadwal	2.030s
```

### Mengompilasi Biner Mandiri (*Production Binary Build*):
Untuk menjalankan bot tanpa perlu dependensi Go di server produksi:
```bash
go build -v -o bot-jadwal.exe .
```

---

## 13. 🛡️ Kestabilan & Keamanan Sistem (*Technical Reliability*)

### A. Pembersihan Database saat Bot Dimatikan (*Graceful Shutdown*)
Saat bot dihentikan di terminal (`Ctrl + C` atau sinyal OS `SIGTERM`), bot mengeksekusi prosedur penutupan teratur:
1. Menghentikan watchdog supervisor di latar belakang.
2. Memutuskan sambungan WhatsApp secara teratur (`client.Disconnect()`).
3. Menutup seluruh koneksi SQLite (`tugas.db`, `sesi_bot.db`) sehingga operasi commit/checkpoint tersimpan bersih dan aman dari risiko *database is locked* atau *WAL-file leak* di Windows.

### B. Ketahanan Sambungan Internet (*Auto-Reconnect Resilience*)
Koneksi jaringan internet kampus atau server seringkali mengalami gangguan sesaat (seperti socket timeout, WiFi kampus drop, atau EOF). Bot dilengkapi:
* **Event Listener Cerdas:** Mendeteksi pemutusan sambungan (`*events.Disconnected`) dan memicu proses rekoneksi instan di latar belakang.
* **Watchdog Supervisor Goroutine:** Rutinitas pengawas independen yang rutin memantau status aktif bot. Jika bot offline tanpa sengaja, supervisor akan mengeksekusi rekoneksi berkala dengan algoritma **Exponential Backoff** (3s ➔ 6s ➔ 12s ➔ maks 30s) hingga sambungan pulih sempurna.

### C. Performa Tinggi & Akses Memori Aman (*Thread-Safe Concurrency*)
Semua operasi pembacaan jadwal dan pengaturan obrolan kelas menggunakan mekanisme **In-Memory Cache O(1)** yang dilindungi oleh `sync.RWMutex`. Bot mampu melayani puluhan pesan masuk secara simultan tanpa mengalami *race condition*, tanpa membebani disk I/O, dan dengan latensi tanggapan di bawah 5 milidetik.

---

## 14. 🎨 Cetak Biru Web Admin Dashboard (*Future Roadmap*)

Untuk melengkapi kemudahan Komti dan pengurus kelas dalam mengelola tugas, jadwal pengganti, dan pengaturan grup, saat ini telah disusun **Product Requirements Document (PRD)** resmi untuk antarmuka web grafis:

* **Dokumen Spesifikasi Lengkap:** [DASHBOARD_PRD.md](file:///f:/Project/bot-jadwal/DASHBOARD_PRD.md)
* **Konsep Arsitektur:** Server Web tertanam murni (*Pure Go Embedded Server*) menggunakan `net/http` dan `embed.FS`, tanpa memerlukan dependensi Node.js atau server terpisah saat berjalan di produksi.
* **Fitur Utama yang Direncanakan:**
  1. **Overview & System Health:** Status bot online, waktu aktif (*uptime*), total tugas, dan daftar perubahan jadwal aktif.
  2. **Interactive Task Board:** Kanban / Table view manajemen tugas dengan modal tambah, edit tenggat, dan arsip riwayat.
  3. **Schedule & Override Manager:** Kalender visual jadwal kuliah dengan modal cepat untuk `Pindah Jam`, `Kelas Kosong`, dan `Set Hari Libur`.
  4. **Multi-Class & Group Mapping:** Antarmuka visual untuk melihat grup WhatsApp mana saja yang terhubung ke kelas 3A, 3B, atau kelas lainnya.
  5. **Morning Reminder Control:** Pengaturan jam broadcast pagi per grup dan simulasi kirim pesan (*instant test broadcast*).

Dokumen PRD tersebut dirancang secara rinci dengan wireframe layar, token desain modern, struktur REST API, dan panduan desain untuk rekan desainer UI/UX sebelum tahap implementasi kode dilakukan.

---
*Dokumentasi ini disusun dan diperbarui untuk Bot WhatsApp Jadwal Kuliah & Manajemen Tugas Mahasiswa.*
