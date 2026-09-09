---
name: ticket-creator
description: >-
  Gunakan skill ini setiap kali pengguna meminta untuk membuat, merumuskan, atau merevisi tiket tugas
  (GitHub Issues / Backlog) untuk proyek bot-jadwal, baik untuk peran Backend Developer, Frontend Developer,
  UI/UX Designer, maupun QA / Software Tester.
---

# Ticket Creator — Bot WhatsApp & Web Admin Dashboard v2.0

Panduan operasional dan standar perumusan tiket tugas (GitHub Issues) untuk tim pengembang mahasiswa proyek `bot-jadwal`.

---

## 1. Konteks Arsitektur Proyek
Setiap tiket yang dibuat wajib selaras dengan fondasi teknis proyek saat ini:
* **Backend:** Bahasa Go (v1.22+), SQLite dengan WAL mode (`storage/tugas.db`), modular layout di `internal/` (`api`, `schedule`, `task`, `link`, `chat`, `reminder`, `bot`).
* **Frontend:** Single-page dashboard di `web/` menggunakan **HTML5 + Tailwind CSS + Alpine.js**, disematkan langsung ke dalam biner Go via `embed.FS` (`web/embed.go`).
* **WhatsApp Client:** Whatsmeow Multi-Device dengan isolasi sesi (`storage/sesi_bot.db`).
* **Lingkungan Eksekusi Lokal:**
  * Mode Web: `go run ./cmd/bot -web-only` (untuk frontend & desain, aman tanpa WhatsApp).
  * Mode Dev WA: `go run ./cmd/bot -session storage/sesi_dev.db` (untuk backend & QA menggunakan nomor testing terpisah).
* **Produksi:** Microsoft Azure Linux VM via `systemd`.

---

## 2. Aturan Format Output (Wajib Dipatuhi)
1. **Bebas AI Slop & Tanpa Banjir Emoji:** Gunakan bahasa teknik profesional yang ringkas, jelas, dan lugas. Hindari pembuka bertele-tele dan hindari penggunaan emoji berlebihan.
2. **Satu Blok Utuh Siap Salin:** Seluruh draf isi tiket wajib dibungkus dalam **satu blok kode menggunakan empat backticks** (` ````markdown ` ) agar blok kode JSON di dalamnya tidak terpecah dan pengguna bisa menyalin dengan satu kali klik.
3. **Pemisahan File (Anti-Conflict):** Untuk tugas backend atau frontend yang dikerjakan oleh 2 orang berbeda, selalu arahkan pengerjaan pada file terpisah (contoh: `tasks_handler.go` vs `schedule_handler.go`).

---

## 3. Template Standar per Peran

### A. Template: BACKEND (BE)
Gunakan format ini untuk tugas pembuatan endpoint API, query database SQLite, atau logika data:

````markdown
### Konteks
[Latar belakang fitur dan kebutuhan bisnis/sistem]

### Lingkup Berkas
- [File baru atau file yang dimodifikasi, sertakan path relatif]

### Spesifikasi Endpoint
- **Method & Path:** `[GET/POST/PUT/DELETE] /api/[nama-endpoint]`
- **Header:** `Content-Type: application/json`
- **Request Body:**
```json
{
  "field": "type"
}
```
- **Validasi:** [Aturan validasi input dan penanganan error 400]
- **Response Success (200/201):**
```json
{
  "status": "success",
  "data": {}
}
```
- **Response Error (400/404/500):**
```json
{
  "status": "error",
  "message": "Deskripsi kesalahan"
}
```

### Kriteria Penyelesaian (Definition of Done)
- [ ] Handler diimplementasikan pada berkas modular terpisah.
- [ ] Validasi input berfungsi dengan benar.
- [ ] Unit test mencakup skenario sukses dan skenario gagal.
- [ ] Seluruh unit test lulus (`go test ./...`).
````

---

### B. Template: FRONTEND (FE)
Gunakan format ini untuk pembuatan halaman web, kartu komponen, modal pop-up, dan integrasi API:

````markdown
### Konteks & Acuan Desain
- **Fitur:** [Nama fitur / komponen UI]
- **Acuan Desain:** [Link Frame Figma atau referensi PRD]

### Lingkup Berkas
- `web/index.html` (atau file pendukung di `web/css/`, `web/js/`)

### Spesifikasi Tampilan & Interaksi
- **Perilaku Responsif:** Menyesuaikan tata letak pada layar HP (< 640px) dan Desktop (>= 768px).
- **Integrasi REST API:** Memanggil endpoint `[METHOD] /api/[endpoint]` via `web/js/api.js`.
- **Interaktivitas (Alpine.js):** [Logika buka/tutup modal, tab switching, atau reactive state]
- **Handling UI:** Indikator loading saat fetch dan feedback notifikasi toast saat aksi berhasil.

### Kriteria Penyelesaian (Definition of Done)
- [ ] Tampilan presisi mengikuti acuan desain Figma.
- [ ] Responsif dan nyaman digunakan di layar HP (thumb-friendly).
- [ ] Berfungsi normal saat dijalankan via `go run ./cmd/bot -web-only`.
- [ ] Bebas dari pesan error pada inspect console browser.
````

---

### C. Template: UI/UX DESIGN
Gunakan format ini untuk perancangan visual, wireframe, prototype, dan design system di Figma:

````markdown
### Konteks & Alur Pengguna
- **Persona:** [Komti Kelas / Mahasiswa / Administrator]
- **Kebutuhan:** [Alur yang ingin dicapai pengguna, misal: mencatat tugas baru secara cepat dari HP]

### Target Layar & Resolusi
- **Mobile (Prioritas Utama):** Frame 390px (iPhone/Android modern).
- **Desktop:** Frame 1440px.

### Kebutuhan Komponen & Variasi State
- **Komponen Inti:** [Daftar komponen visual yang dibutuhkan]
- **Variasi State Wajib:**
  - *Default State* (tampilan data normal)
  - *Empty State* (tampilan saat data belum ada)
  - *Active / Error State* (validasi form dan pesan gagal)

### Kriteria Penyelesaian (Definition of Done)
- [ ] Menggunakan palet warna dan tipografi sesuai design tokens PRD.
- [ ] Komponen menggunakan fitur Auto-Layout Figma.
- [ ] Tersedia mockup lengkap versi Desktop dan versi Mobile.
- [ ] Tautan Figma disematkan di tiket dan siap diinspeksi oleh Frontend.
````

---

### D. Template: TESTER / QA
Gunakan format ini untuk pembuatan skenario pengujian, audit fungsional, dan pencarian bug:

````markdown
### Konteks Pengujian
[Modul atau fitur yang akan diuji fungsionalitasnya]

### Lingkungan Uji
- **Platform:** [Mobile Browser / Desktop Browser / WhatsApp Sandbox]
- **Mode Eksekusi:** [Mode -web-only / Mode -session dev]

### Skenario Pengujian (Test Cases)
1. **Kasus Normal (Happy Path):**
   - Langkah uji dan hasil yang diharapkan.
2. **Kasus Negatif & Validasi:**
   - Uji input kosong, format tanggal tidak valid, atau karakter khusus.
3. **Kasus Responsivitas / Multi-Device:**
   - Pengujian tampilan dan interaksi tombol pada layar HP fisik.

### Kriteria Penyelesaian (Definition of Done)
- [ ] Seluruh skenario pengujian selesai dieksekusi.
- [ ] Setiap temuan bug didokumentasikan di GitHub Issues dengan langkah reproduksi lengkap.
- [ ] Memberikan konfirmasi kelayakan rilis kepada Project Manager.
````
