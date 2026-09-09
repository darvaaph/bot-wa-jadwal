---
name: git-team-flow
description: >-
  Panduan alur kerja kolaborasi Git tim mahasiswa, pembuatan branch per tiket,
  standar Conventional Commits, dan protokol Pull Request (PR).
  Secara proaktif memicu panduan Git setiap kali fitur selesai dikerjakan
  tanpa pernah melakukan auto-commit liar oleh AI.
---

# Git Team Flow & Collaboration Guide

Panduan operasional dan standar kolaborasi Git untuk seluruh tim pengembang mahasiswa proyek `bot-jadwal`. 

Skill ini memastikan repositori tetap bersih, mencegah konflik berkas (*merge conflict*), mengajarkan etika Git profesional, dan **memandu mahasiswa secara proaktif setiap kali sebuah tugas atau fitur selesai dikerjakan**.

---

## 1. Aturan Emas: DILARANG AUTO-COMMIT OLEH AI

> [!IMPORTANT]
> **AI Agent DILARANG KERAS menjalankan perintah `git commit` atau `git push` secara otomatis / mandiri.**
> 
> Seluruh perintah Git wajib disajikan dalam bentuk blok kode perintah terminal yang jelas dan terstruktur agar **dieksekusi sendiri oleh mahasiswa**.
> 
> **Alasan Pedagogis:** Mahasiswa harus memegang kendali penuh (*ownership*) atas identitas commit mereka, memahami berkas apa saja yang masuk ke Git, dan terbiasa dengan terminal Git industri.

---

## 2. Pemantik Proaktif Selesai Fitur (Completion Trigger)

AI Agent **TIDAK BOLEH** menunggu mahasiswa menyebut kata "git" baru memberikan panduan.

Setiap kali suatu tiket, fitur, atau perbaikan bug selesai dikerjakan dan diverifikasi (misalnya mahasiswa berkata: *"udah jalan"*, *"fiturnya udah selesai"*, *"kodenya udah dites"*, atau AI selesai memandu implementasi):

AI **WAJIB secara proaktif menutup responnya dengan panduan penyimpanan Git** menggunakan format berikut:

```markdown
---
### 🚀 Fitur Selesai! Saatnya Menyimpan ke Git

Kerja bagus! Sebelum beralih ke tugas lain, simpan pekerjaanmu ke Git dengan aman agar tidak hilang dan siap di-review oleh Project Manager:

1. **Buka terminal dan periksa status berkas:**
   ```bash
   git status
   ```

2. **Tambahkan berkas yang terkait saja (hindari git add . jika ada file sampah):**
   ```bash
   git add <path/ke/berkas-yang-diubah>
   ```

3. **Buat pesan commit profesional:**
   ```bash
   git commit -m "<tipe>(<lingkup>): <deskripsi singkat>"
   ```

4. **Kirim branch fiturnya ke GitHub:**
   ```bash
   git push -u origin <nama-branch-fitur>
   ```

5. **Buka Pull Request (PR) di GitHub** dan gunakan draf deskripsi yang disediakan di bawah.
```

---

## 3. Standar Penamaan Branch (Sesuai Tiket)

**DILARANG KERAS** membuat commit atau melakukan *push* langsung ke branch `main`. Seluruh pengerjaan wajib berada di branch fitur masing-masing.

### Format Penamaan Branch:
- **Fitur Baru (Feature):**
  `feat/<kode-tiket>-<deskripsi-singkat>`
  *Contoh:*
  - `feat/be-1-tasks-api`
  - `feat/fe-2-schedule-table`
  - `feat/ui-3-task-modal-design`
- **Perbaikan Bug (Bugfix):**
  `fix/<kode-tiket>-<deskripsi-singkat>`
  *Contoh:*
  - `fix/fe-3-modal-validation`
  - `fix/be-2-schedule-day-filter`
- **Dokumentasi & Pengujian:**
  `docs/<topik>` atau `test/qa-<topik>`
  *Contoh:*
  - `docs/qa-1-test-cases-bot`

### Alur Memulai Branch Baru:
```bash
# 1. Selalu pastikan branch main lokal sinkron dengan pusat
git checkout main
git pull origin main

# 2. Buat dan berpindah ke branch fitur baru
git checkout -b <nama-branch-fitur>
```

---

## 4. Standar Pesan Commit (Conventional Commits)

Format pesan commit wajib mengikuti pola:
`<tipe>(<lingkup>): <deskripsi imperatif huruf kecil>`

### Pilihan Tipe:
- `feat`: Menambah fitur baru (misal: endpoint baru, komponen tabel baru).
- `fix`: Memperbaiki bug atau kesalahan logika.
- `docs`: Mengubah atau menambah dokumentasi (README, PRD, dokumen QA).
- `style`: Perubahan visual/CSS tanpa mengubah logika fungsional.
- `refactor`: Perapian kode tanpa mengubah perilaku fungsional.
- `test`: Menambah atau memperbaiki unit test.

### Contoh Pesan Commit yang Benar:
- `feat(api): implement GET and POST /api/tasks handlers for BE-1`
- `feat(web): add weekly schedule table component for FE-2`
- `fix(modal): correct deadline validation logic for FE-3`
- `docs(qa): draft master test cases spreadsheet for QA-1`

*Hindari pesan commit tidak informatif seperti: "update", "fix bug", "bismillah", "selesai".*

---

## 5. Draf Deskripsi Pull Request (PR) Siap Salin

Setiap kali mahasiswa siap membuka Pull Request ke branch `main`, AI wajib menyediakan draf deskripsi PR dengan struktur berikut:

```markdown
### Ringkasan Perubahan
[Jelaskan secara ringkas dalam 1-2 kalimat apa yang dilakukan pada PR ini]

### Tiket Terkait
Closes #[nomor-issue-di-github]

### Berkas yang Diubah / Ditambah
- `[path/ke/berkas1]`
- `[path/ke/berkas2]`

### Ceklis Mandiri Pengembang (Self-Verification)
- [ ] Kode sudah diuji dan berjalan normal di lingkungan lokal.
- [ ] Tidak ada dependensi terlarang (bebas npm/node_modules).
- [ ] Backend: seluruh unit test lulus (`go test ./...`).
- [ ] Frontend: konsol browser bersih dari error JavaScript.
- [ ] Tidak ada berkas database (`.db`) atau berkas sementara yang ter-commit.
```

---

## 6. Penanganan Jika Terjadi Konflik (Merge Conflict Prevention)

Jika branch `main` telah diperbarui oleh anggota tim lain saat mahasiswa masih bekerja di branch fiturnya:

```bash
# Sinkronkan branch fitur dengan main terbaru
git fetch origin
git merge origin/main
```
Jika terjadi konflik, AI bertugas memandu mahasiswa membaca penanda `<<<<<<< HEAD`, `=======`, dan `>>>>>>>` secara tenang dan objektif tanpa menghapus kode milik rekan setim.
