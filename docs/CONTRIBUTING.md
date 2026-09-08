# 🤝 Panduan Kontribusi Tim (Contributing Guide)
# Bot WhatsApp Jadwal Kuliah & Web Admin Dashboard

Selamat datang di tim pengembang! Dokumen ini adalah panduan standar operasional untuk seluruh anggota tim—baik **UI/UX Designer**, **Frontend Developer**, **Backend Developer**, maupun **QA / Software Tester**—agar alur kerja kita terstruktur, rapi, dan mencerminkan kultur *engineering* di dunia kerja nyata/startup.

---

## 📑 Daftar Isi
1. [Setup Lingkungan Pengembangan Lokal](#1-setup-lingkungan-pengembangan-lokal)
2. [Alur Kerja Git (Git Workflow)](#2-alur-kerja-git-git-workflow)
3. [Standar Penulisan Pesan Commit (Conventional Commits)](#3-standar-penulisan-pesan-commit-conventional-commits)
4. [Kriteria Penyelesaian Pekerjaan (Definition of Done)](#4-kriteria-penyelesaian-pekerjaan-definition-of-done)
5. [Panduan Kerja Khusus per Peran](#5-panduan-kerja-khusus-per-peran)
6. [Etika & Saluran Komunikasi Tim](#6-etika--saluran-komunikasi-tim)

---

## 1. 🚀 Setup Lingkungan Pengembangan Lokal

Seluruh anggota tim yang berkontribusi pada kode wajib memastikan proyek dapat dijalankan di komputer masing-masing.

### Prasyarat:
* **Go (Golang):** Minimal versi 1.22 atau yang lebih baru ([Unduh di golang.org](https://golang.org)).
* **Git:** Terpasang dan terkonfigurasi dengan nama & email Anda.

### Langkah Memulai:
1. Clone repositori:
   ```bash
   git clone https://github.com/darvaaph/bot-wa-jadwal.git
   cd bot-wa-jadwal
   ```
2. Unduh seluruh dependensi modul Go:
   ```bash
   go mod download
   ```
3. Jalankan pengujian unit otomatis untuk memastikan semua modul berfungsi normal:
   ```bash
   go test -v ./...
   ```
4. Menjalankan bot dan server API web lokal:
   ```bash
   go run ./cmd/bot
   ```
   > ⚠️ **Catatan Penting Login WhatsApp:**  
   > Jika bot server produksi sedang aktif, koordinasikan dengan Project Manager sebelum melakukan scan QR Code WhatsApp agar sesi login server tidak terganggu. Anda dapat menguji logika modul dan API web tanpa harus terhubung ke WhatsApp.

---

## 2. 🌿 Alur Kerja Git (Git Workflow)

Kami menggunakan alur **GitHub Flow** yang berbasis *Pull Request* (PR). **Dilarang keras melakukan commit atau push langsung ke branch `main`!**

### Langkah Membuat Fitur Baru:
1. Pastikan branch `main` lokal Anda selalu ter-update:
   ```bash
   git checkout main
   git pull origin main
   ```
2. Buat branch baru dengan format penamaan sesuai jenis pekerjaan:
   * Fitur baru: `feat/nama-fitur` (contoh: `feat/task-api-crud`, `feat/dashboard-navbar`)
   * Perbaikan bug: `fix/nama-bug` (contoh: `fix/deadline-parser`, `fix/table-overflow`)
   * Dokumentasi: `docs/nama-dokumen` (contoh: `docs/api-spec`)
   * Pengujian: `test/nama-modul` (contoh: `test/link-manager`)
   * Refactoring: `refactor/nama-modul`

   ```bash
   git checkout -b feat/task-api-crud
   ```
3. Lakukan perubahan kode, pastikan `go test ./...` tetap lulus, lalu commit.
4. Push branch ke remote:
   ```bash
   git push origin feat/task-api-crud
   ```
5. Buka **Pull Request (PR)** di GitHub menuju branch `main`.
6. Isi template PR yang telah disediakan dan minta *Review* ke Project Manager / Peer Reviewer.

---

## 3. 📝 Standar Penulisan Pesan Commit (Conventional Commits)

Gunakan format standar industri agar riwayat perubahan (*changelog*) mudah dibaca:

```text
<tipe>(<lingkup opsional>): <deskripsi ringkas dalam kalimat imperatif>
```

### Tipe Commit yang Didukung:
* `feat`: Penambahan fitur baru (contoh: `feat(api): tambah endpoint POST /api/tasks`)
* `fix`: Perbaikan bug atau kesalahan logika (contoh: `fix(schedule): perbaiki perhitungan kuliah berikutnya pada hari libur`)
* `docs`: Pembaruan atau penambahan dokumentasi (contoh: `docs: perbarui panduan setup lokal`)
* `refactor`: Perubahan struktur kode tanpa mengubah fungsionalitas (contoh: `refactor(chat): ekstrak helper admin ke package chat`)
* `test`: Penambahan atau perbaikan unit test (contoh: `test(link): tambah uji kasus duplicate URL`)
* `chore`: Tugas pemeliharaan, update dependencies, atau build script (contoh: `chore: perbarui aturan .gitignore`)

---

## 4. ✅ Kriteria Penyelesaian Pekerjaan (Definition of Done / DoD)

Sebuah Pull Request (PR) baru dapat di-*merge* ke branch utama jika memenuhi checklist berikut:
- [ ] **Lolos Uji Otomatis:** Menjalankan `go test -v ./...` dan seluruh test berstatus `PASS`.
- [ ] **Bebas Linting/Vet Error:** Menjalankan `go vet ./...` tanpa pesan kesalahan.
- [ ] **Git Hygiene:** Tidak ada file biner (`*.exe`, `bot-jadwal`), file database (`*.db`), atau file arsip yang ikut ter-commit.
- [ ] **Dokumentasi / Komentar:** Kode baru yang kompleks memiliki komentar penjelas yang jelas dalam bahasa Indonesia yang baik.
- [ ] **Code Review:** Mendapatkan minimal 1 persetujuan (*Approved*) dari Project Manager atau rekan tim.

---

## 5. 👥 Panduan Kerja Khusus per Peran

### 🎨 UI/UX Designer
* **Fokus Utama:** Merancang antarmuka Web Admin Dashboard yang modern, intuitif, dan responsif (Mobile & Desktop) berdasarkan spesifikasi [DASHBOARD_PRD.md](DASHBOARD_PRD.md).
* **Deliverable:**
  - File desain Figma terorganisir (komponen, auto-layout, prototype interaktif).
  - Mengikuti palet warna *Modern Academic Dark Interface* yang tercantum di PRD.
  - Dokumentasi aset (ikon, SVG, font tokens) yang siap diserahkan ke Frontend Developer.

### 💻 Frontend Developer
* **Fokus Utama:** Menerjemahkan rancangan Figma dari UI/UX Designer menjadi halaman web responsif dan mengintegrasikannya dengan REST API backend.
* **Lingkup Kerja:** Direktori `web/` (atau starter frontend yang disepakati).
* **Prinsip Utama:**
  - Desain harus mobile-friendly (Komti kelas banyak mengakses dari HP saat di kampus).
  - Gunakan penanganan status loading, error alert, dan konfirmasi aksi (misal modal konfirmasi sebelum menghapus tugas).

### ⚙️ Backend Developer (Go)
* **Fokus Utama:** Mengembangkan endpoint REST API di `internal/api/`, optimasi database SQLite, dan memastikan performa backend tinggi.
* **Prinsip Utama:**
  - Selalu perhatikan *concurrency* (gunakan connection pool `database.InitDB` dan SQLite WAL mode).
  - Terapkan validasi input ketat pada setiap request JSON.
  - Wajib menyertakan unit test (`*_test.go`) untuk setiap handler atau fungsi baru.

### 🧪 QA / Software Tester
* **Fokus Utama:** Memastikan bot WhatsApp dan Web Dashboard bekerja tanpa celah kegagalan.
* **Lingkup Kerja:**
  - Menyusun skenario pengujian fungsional dan *edge cases* (misal: input format tanggal aneh di chat, bentrok jam kuliah, karakter khusus pada nama tugas).
  - Melaporkan bug secara terstruktur di GitHub Issues menggunakan template **Bug Report**.
  - Melakukan verifikasi ulang setelah bug diperbaiki oleh developer.

---

## 6. 💬 Etika & Saluran Komunikasi Tim

* **Transparan & Asinkron:** Jika mengalami kendala teknis (*blocker*), jangan ragu untuk bertanya di forum tim. Lebih baik bertanya daripada terhambat berhari-hari.
* **Saling Menghargai:** Berikan kritik yang konstruktif saat melakukan *Code Review* di GitHub PR.
* **Komitmen Waktu:** Alokasikan 3–5 jam per minggu sesuai kesepakatan awal untuk menjaga ritme sprint tim.
