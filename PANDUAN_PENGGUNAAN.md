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
8. [Perbedaan Akses: Grup vs Chat Pribadi (DM)](#8-perbedaan-akses-grup-vs-chat-pribadi-dm)
9. [Format Penulisan Tenggat Waktu (Natural Time Parser)](#9-format-penulisan-tenggat-waktu-natural-time-parser)
10. [Cara Menjalankan & Menguji Bot di Komputer](#10-cara-menjalankan--menguji-bot-di-komputer)

---

## 1. 🚀 Sekilas Tentang Bot

Bot WhatsApp ini bertindak sebagai **asisten virtual kelas** yang memudahkan mahasiswa dalam:
* Mengecek jadwal kuliah harian, mingguan, maupun kuliah yang sedang berlangsung secara instan.
* Menerjemahkan inisial kode dosen dan nama ruangan laboratorium/gedung.
* Mengirimkan broadcast jadwal otomatis setiap hari kuliah pukul **06:30 WIB** ke grup kelas.
* Melacak tugas-tugas kuliah (*deadline tracker*) dengan indikator hitung mundur (*countdown*) dan peringatan otomatis jika ada tugas mendesak (*urgent*).

---

## 2. 🧠 Prinsip Desain & Arsitektur Cerdas

### A. Strategi Hybrid (Anti-Spam Grup & Fleksibel di DM)
* **Di Grup WhatsApp:** Setiap perintah **WAJIB** diawali dengan simbol prefix (`!`, `/`, atau `#`). Hal ini mencegah bot merespons obrolan biasa anggota grup (misal saat ada yang mengobrol kata *"senin"* atau *"besok"*).
* **Di Chat Pribadi (DM):** Mahasiswa bebas mengetik tanpa prefix (contoh: cukup ketik `hari ini`, `besok`, atau `tugas`).

### B. Simulasi Manusiawi & Anti-Ban WhatsApp
* **Typing Indicator (`ChatPresenceComposing`):** Setiap merespons perintah, bot memunculkan status *"sedang mengetik..."* selama 600ms agar interaksi terasa natural dan mengurangi risiko deteksi bot oleh server WhatsApp.
* **Reaksi Emoji Otomatis:**
  * 📅 untuk perintah jadwal perkuliahan.
  * ⏰ untuk perintah pengingat otomatis (`!reminder`).
  * 📝 untuk perintah manajemen tugas (`!tugas`).

### C. Pembatas Garis Pesan Rapi (Mobile Friendly)
* Semua garis pembatas diformat tepat 10 karakter (`──────────`) sehingga tidak terlipat (*wrapping*) berantakan di layar ponsel dengan font besar.

### D. Penyimpanan Data Modular & Persisten
* `jadwal.json`: Data master mata kuliah, dosen, ruangan, dan jam.
* `reminder_groups.json`: Daftar JID grup yang mengaktifkan broadcast pengingat pagi.
* `tugas.db` (SQLite): Database penyimpanan tugas perkuliahan dan to-do list pribadi.

---

## 3. 🌟 Daftar Fitur Lengkap

1. **Jadwal Harian & Mingguan:** Akses jadwal hari tertentu (`!senin`, `!selasa`, dll.), jadwal hari ini (`!hari ini`), jadwal besok (`!besok`), atau seluruh minggu kerja (`!seminggu`).
2. **Kuliah Real-Time (`!next` / `!sekarang`):** Mendeteksi mata kuliah yang sedang aktif saat ini (beserta sisa menit), kuliah yang akan dimulai berikutnya, atau status libur/selesai.
3. **Direktori Lengkap:**
   * `!dosen [Nama/Inisial]` : Mencari data dosen JTK Polban berdasarkan inisial atau nama lengkap.
   * `!ruang [Nama/Kode]` : Mencari letak dan jadwal ruangan tertentu.
   * `!matkul` : Daftar ringkasan mata kuliah semester aktif.
   * `!cari [Kata Kunci]` : Pencarian bebas (*fuzzy search*) di seluruh mata kuliah.
4. **Reload On-The-Fly (`!reload`):** Membaca ulang isi file `jadwal.json` ke memori tanpa perlu mematikan program bot di terminal.
5. **Broadcast Pengingat Pagi (06:30 WIB):** Background worker yang otomatis menyapa grup kelas setiap hari kerja (Senin s.d. Jumat) dengan jadwal hari itu dan daftar tugas mendesak.
6. **Deadline Tracker Tugas (`!tugas`):**
   * Pelacak tugas aktif yang tetap bertahan sampai tenggatnya lewat (tidak dihapus sepihak).
   * Badge urgensi otomatis (`🚨 DEADLINE HARI INI`, `⚠️ DEADLINE BESOK (H-1)`, `⏳ H-X`).
   * Filter cepat `!tugas hari ini` dan `!tugas besok`.
   * Proteksi anti-duplikasi tugas cerdas.
   * Role-based access control (RBAC): Di grup hanya Admin yang dapat mengubah data tugas.

---

## 4. 📊 Tabel Lengkap Perintah (Cheat Sheet)

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
| `!reload` | `!reload` | `reload` | Muat ulang jadwal dari `jadwal.json` |

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
| `!tugas tambah [M] \| [D] \| [T]` | `!tugas tambah ...` | `tugas tambah ...` | **Khusus Admin** |
| `!tugas edit [ID] \| [Tenggat]` | `!tugas edit 1 \| minggu 23:59` | `tugas edit 1 \| ...` | **Khusus Admin** |
| `!tugas selesai [ID]` | `!tugas selesai 1` | `tugas selesai 1` | **Khusus Admin** |
| `!tugas hapus [ID]` | `!tugas hapus 1` | `tugas hapus 1` | **Khusus Admin** |
| `!tugas bantuan` | `!tugas bantuan` | `tugas bantuan` | **Semua Anggota** |

### Perintah Jadwal Pengganti Sementara (`!pindah`, `!kosong`, `!kuliahganti`)
| Perintah | Di Grup | Di DM Pribadi | Hak Akses di Grup |
| :--- | :--- | :--- | :--- |
| `!pindah [Matkul] \| [Waktu] \| [Ruang]` | `!pindah ...` | `pindah ...` | **Khusus Admin** |
| `!kosong [Matkul] \| [Waktu] \| [Alasan]` | `!kosong ...` | `kosong ...` | **Khusus Admin** |
| `!kuliahganti [Matkul] \| [Waktu] \| [Ruang]`| `!kuliahganti ...`| `kuliahganti ...`| **Khusus Admin** |
| `!jadwalganti` | `!jadwalganti` | `jadwalganti` | **Semua Anggota** |
| `!batalganti [ID]` | `!batalganti 1` | `batalganti 1` | **Khusus Admin** |

---

## 5. 📅 Panduan Penggunaan Fitur Jadwal Kuliah

### A. Melihat Kuliah yang Sedang Berlangsung (`!next` / `!sekarang`)
Bot memeriksa waktu saat pesan diterima dan mencocokkan dengan jadwal hari ini:
* **Jika ada kuliah berlangsung:** Bot menampilkan nama matkul, dosen, ruangan, dan sisa waktu sebelum kelas usai.
* **Jika kuliah berikutnya masih beberapa jam lagi:** Bot menampilkan hitung mundur menuju kelas tersebut.
* **Jika semua kuliah hari ini selesai:** Bot mengonfirmasi bahwa sesi perkuliahan telah berakhir.

### B. Memperbarui Jadwal Permanen (`!reload`)
Jika terjadi perubahan ruangan atau jam kuliah resmi untuk seterusnya:
1. Buka file `jadwal.json` di komputer server.
2. Ubah data jam atau ruangan yang bersangkutan, lalu simpan file.
3. Di grup WhatsApp atau chat pribadi ke bot, kirim pesan:
   ```text
   !reload
   ```
4. Bot akan merespons konfirmasi bahwa file `jadwal.json` telah berhasil dimuat ulang ke memori. Semua perintah berikutnya langsung menggunakan jadwal terbaru tanpa perlu restart bot!

### C. Jadwal Pengganti Sementara (*Schedule Override*: `!pindah`, `!kosong`, `!kuliahganti`)
Untuk perubahan mendadak yang **hanya berlaku pada satu tanggal/minggu saja**, gunakan fitur jadwal pengganti agar jadwal permanen di `jadwal.json` tidak rusak dan otomatis kembali normal di minggu depan:

#### 1. Memindahkan Jam atau Hari Kuliah (`!pindah`)
* **Format:** `!pindah [Matkul] | [Hari/Tanggal & Jam Baru] | [Ruang (Opsional)]`
* **Contoh:**
  * `!pindah aljabar | besok 13:00 | Lab 312`
  * `!pindah sbd | jumat 15:00 - 16:40`
* **Efek:** 
  * Pada **hari asal**: jadwal asli dicoret `~~07:00 - 08:40 WIB~~` dengan status `❌ *KULIAH DIPINDAHKAN* (Dipindah ke: ...)`.
  * Pada **hari tujuan**: otomatis disisipkan dengan badge `🔄 [KULIAH PENGGANTI]`.
  * **Minggu depan**: otomatis kembali ke jadwal normal tanpa perlu diedit balik.

#### 2. Menandai Kuliah Ditiadakan / Kosong (`!kosong`)
* **Format:** `!kosong [Matkul] | [Hari/Tanggal (Opsional)] | [Alasan (Opsional)]`
* **Contoh:**
  * `!kosong sbd | besok | Dosen dinas luar kota`
  * `!kosong matdis | hari ini`
* **Efek:** Pada tanggal tersebut jadwal dicoret dengan keterangan alasan ditiadakan.

#### 3. Menambah Kuliah Pengganti di Hari Libur / Kosong (`!kuliahganti`)
* **Format:** `!kuliahganti [Matkul] | [Hari/Tanggal & Jam] | [Ruang]`
* **Contoh:**
  * `!kuliahganti matdis | sabtu 09:00 - 11:30 | D105`
* **Efek:** Bot akan mengenali adanya kuliah tambahan pada hari tersebut dan menyertakannya di pengingat.

#### 4. Cek dan Batalkan Perubahan Jadwal
* `!jadwalganti` : Melihat daftar seluruh perubahan jadwal sementara yang masih aktif.
* `!batalganti [ID]` : Membatalkan perubahan jadwal (jadwal langsung kembali normal seketika).

#### 5. Deteksi Bentrok Jadwal Otomatis (*Conflict Warning*)
Saat memindahkan kelas (`!pindah`) atau membuat kuliah pengganti (`!kuliahganti`), bot secara otomatis memeriksa apakah jam yang dipilih bertabrakan dengan jadwal mata kuliah lain pada tanggal tersebut:
* **Jika Terjadi Bentrok:** Bot membatalkan aksi dan menampilkan rincian mata kuliah, jam, ruangan, serta dosen yang bertabrakan.
* **Opsi Konfirmasi Paksa:** Jika kelas tersebut memang sudah disepakati (misal kelas lain sudah kosong), Komti dapat menambahkan kata `paksa` di akhir:
  ```text
  !pindah aljabar | selasa 09:00 | paksa
  !kuliahganti sbd | sabtu 10:00 | Lab 312 | paksa
  ```

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


---

## 8. 🔒 Perbedaan Akses: Grup vs Chat Pribadi (DM)

| Fitur / Karakteristik | Di Grup Kelas | Di Chat Pribadi (DM) |
| :--- | :--- | :--- |
| **Kebutuhan Simbol Prefix** | Wajib (`!`, `/`, `#`) | Opsional (Bisa tanpa prefix) |
| **Cakupan Tugas (`!tugas`)** | Bersama sekelas (*Public Class Board*) | Catatan to-do list pribadi (*Private*) |
| **Hak Tambah Tugas** | **Khusus Admin Grup** (Komti/Wakil) | Bebas (Setiap mahasiswa) |
| **Hak Selesaikan/Hapus Tugas** | **Khusus Admin Grup** | Bebas (Milik sendiri) |
| **Tujuan Pembatasan** | Mencegah spam & tugas palsu di grup | Fleksibilitas pencatatan mandiri |

---

## 9. 🕒 Format Penulisan Tenggat Waktu (Natural Time Parser)

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

## 10. 💻 Cara Menjalankan & Menguji Bot di Komputer

### Prasyarat:
* Go (Golang) versi 1.22 atau lebih baru terinstal.
* Koneksi internet aktif untuk menghubungkan klien WhatsApp.

### Menjalankan Bot:
Buka terminal di direktori proyek `f:\Project\bot-jadwal` lalu jalankan:
```bash
go run .
```
Jika baru pertama kali dijalankan atau sesi habis, terminal akan menampilkan **QR Code**. Pindai QR Code tersebut melalui menu **Linked Devices (Perangkat Tertaut)** di aplikasi WhatsApp ponsel Anda.

### Menjalankan Pengujian Otomatis (Unit Test):
Untuk memastikan seluruh logika jadwal dan database tugas berfungsi sempurna:
```bash
go test -v .
```
Output yang diharapkan:
```text
=== RUN   TestSchedule
--- PASS: TestSchedule (0.00s)
=== RUN   TestTaskManager
--- PASS: TestTaskManager (0.02s)
PASS
ok  	bot-jadwal	1.415s
```

---
*Dokumentasi ini disusun dan diperbarui untuk Bot WhatsApp Jadwal Kuliah & Manajemen Tugas Mahasiswa.*
