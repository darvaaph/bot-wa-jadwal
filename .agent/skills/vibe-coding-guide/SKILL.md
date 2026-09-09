---
name: vibe-coding-guide
description: >-
  Panduan komprehensif, batasan arsitektur (guardrails), praktik terbaik (best practices),
  dan protokol edukasi bagi pengembang mahasiswa saat melakukan vibe coding (pair programming dengan AI)
  pada repositori bot-jadwal (Go, SQLite WAL, embedded Tailwind CDN & Alpine.js).
---

# Vibe Coding Guide & Architectural Guardrails

Panduan operasional dan batasan arsitektur (*guardrails*) untuk seluruh pengembang mahasiswa proyek `bot-jadwal` v2.0 saat berkolaborasi dengan AI (*pair programming / vibe coding*).

Tujuan utama skill ini adalah **menghasilkan kode berstandar industri tanpa merusak arsitektur**, sekaligus **membantu mahasiswa memahami konsep teknis di balik setiap kode yang dibuat**.

---

## 1. Protokol Edukasi (Pedagogical Protocol)

Setiap kali AI Agent menghasilkan, memodifikasi, atau merefaktor kode atas instruksi pengguna, AI **WAJIB** menyertakan penjelasan konsep singkat di akhir respons dengan format:

```markdown
### 💡 Catatan Belajar (Konsep di Balik Kode)
- **Mengapa pendekatan ini dipilih:** [Penjelasan arsitektural singkat]
- **Konsep inti yang dipelajari:** [Pola desain, reaktivitas Alpine.js, query SQLite, atau error handling Go]
- **Potensi kesalahan pemula:** [Kesalahan umum yang harus dihindari terkait topik ini]
```

*Prinsip: Mahasiswa tidak hanya menerima kode jadi, melainkan memahami cara kerja dan alasannya agar siap saat presentasi atau audit kode.*

---

## 2. Batasan Keras Arsitektur (Hard Guardrails)

Seluruh instruksi dari pengembang atau AI **DILARANG KERAS** melanggar batas-batas teknis berikut:

### A. Frontend: Zero-Build & Single Binary
1. **Dilarang Menambah Node.js / NPM:**
   - JANGAN PERNAH menjalankan `npm init`, `npm install`, atau memasukkan `package.json`, `node_modules`, Webpack, Vite, atau framework SPA seperti React/Vue/Angular.
   - Stack frontend adalah **HTML5 murni + Tailwind CSS (via CDN) + Alpine.js (via CDN)**.
2. **Penyematan Biner Go (`embed.FS`):**
   - Seluruh berkas web wajib berada di dalam direktori `web/` (`index.html`, `css/`, `js/`, `assets/`).
   - Berkas disematkan langsung ke dalam biner Go melalui `web/embed.go`. Penambahan berkas di luar struktur ini akan menyebabkan aset tidak terbawa ke biner produksi di server Azure.
3. **Pemisahan Berkas Skrip:**
   - Logika komunikasi HTTP fetch / API disimpan di `web/js/api.js`.
   - Logika state reaktif dan interaktivitas Alpine.js disimpan di `web/js/app.js`.
   - Dilarang menulis blok `<script>` panjang bercampur baur di dalam `index.html`.

### B. Backend: Modular Go & SQLite WAL
1. **Pustaka Standar (Standard Library First):**
   - Gunakan fitur perutean (*routing*) bawaan Go v1.22+ (`net/http` dengan pola `http.ServeMux`).
   - Dilarang menambahkan web framework berat baru (seperti Gin, Fiber, atau Echo) tanpa persetujuan Project Manager.
2. **Pemisahan Berkas Handler (Anti-Monolith):**
   - Berkas `internal/api/server.go` hanya berfungsi sebagai inisialisasi server dan pendaftaran rute.
   - Logika endpoint wajib dipisah ke berkas modular di `internal/api/`:
     - `tasks_handler.go` untuk domain tugas kuliah.
     - `schedule_handler.go` untuk domain jadwal dan kelas.
     - `bot_handler.go` untuk telemetri status WhatsApp dan QR Code.
3. **Keamanan Database SQLite:**
   - Berkas database: `storage/tugas.db` (dijalankan dengan `PRAGMA journal_mode=WAL`).
   - **Wajib Gunakan Prepared Statements:** Selalu gunakan parameter binding tanda tanya (`?`) pada setiap query `db.Query` atau `db.Exec`. Dilarang melakukan penggabungan string (*string concatenation* atau `fmt.Sprintf`) untuk menyusun query SQL demi mencegah celah **SQL Injection**.

### C. Keamanan Lingkungan Eksekusi & WhatsApp Gateway
1. **Pengerjaan Frontend & Desain:**
   - Selalu jalankan dengan mode tanpa WhatsApp:
     ```bash
     go run ./cmd/bot -web-only
     ```
   - Mode ini mengaktifkan web server di port 8080 secara terisolasi tanpa menghubungkan ke WhatsApp.
2. **Pengerjaan Backend & Pengujian WhatsApp:**
   - **DILARANG KERAS** menyentuh atau menghapus berkas sesi produksi: `storage/sesi_bot.db`.
   - Pengembang dan QA wajib menggunakan berkas sesi pengujian terpisah:
     ```bash
     go run ./cmd/bot -session storage/sesi_dev.db
     ```
   - Sesi pengujian wajib dipasangkan (*paired*) ke nomor WhatsApp pengujian khusus di grup sandbox, bukan grup resmi kelas.

---

## 3. Praktik Terbaik Standar Industri (Best Practices)

### A. Standar Kode Go (Idiomatic Go)
- **Eksplisit Error Handling:** Selalu tangani error secara eksplisit.
  ```go
  if err != nil {
      log.Printf("[ERROR] gagal memproses data: %v", err)
      http.Error(w, `{"status":"error","message":"Gagal memproses data"}`, http.StatusInternalServerError)
      return
  }
  ```
  *Jangan gunakan `_ = err` atau membiarkan error terlewat tanpa logging.*
- **Konsistensi Format Respon JSON:**
  - Respon Berhasil:
    ```json
    {
      "status": "success",
      "data": { ... }
    }
    ```
  - Respon Gagal:
    ```json
    {
      "status": "error",
      "message": "Deskripsi kesalahan yang ramah pengguna"
    }
    ```
- **Status Code Semantik:**
  - `200 OK`: Pengambilan atau pembaruan data berhasil.
  - `201 Created`: Pembuatan entitas baru berhasil (misal: tambah tugas baru).
  - `400 Bad Request`: Validasi input tidak sesuai (misal: field wajib kosong).
  - `404 Not Found`: Data yang diminta tidak ditemukan di database.
  - `500 Internal Server Error`: Terjadi kesalahan pada level server/database.

### B. Standar Kode Frontend (Alpine.js & Tailwind CSS)
- **Penanganan Status UI (UI States):**
  Setiap komponen data wajib menangani 4 status:
  1. *Loading State:* Indikator spinner atau skeleton saat data sedang diambil.
  2. *Success / Default State:* Data tampil rapi sesuai layout.
  3. *Empty State:* Tampilan ramah dan instruktif jika array data kosong (misal: belum ada tugas).
  4. *Error State:* Pesan feedback jika koneksi ke backend gagal.
- **Aksesibilitas & Desain Sentuh Jempol (Thumb-Friendly):**
  - Ukuran area sentuh (*touch target*) untuk tombol aksi minimal berukuran **44 x 44 piksel** (`p-2.5`, `min-h-[44px]`).
  - Uji tata letak pada breakpoint mobile (`< 640px`) dan desktop (`>= 768px`).

---

## 4. Prosedur Verifikasi Mandiri Sebelum Commit

Sebelum membuat commit atau membuka Pull Request (PR), pengembang wajib melakukan verifikasi mandiri:

1. **Pengembang Backend:**
   - Jalankan seluruh pengujian unit Go:
     ```bash
     go test -v ./...
     ```
   - Pastikan kode dapat dikompilasi tanpa error:
     ```bash
     go build ./cmd/bot
     ```
2. **Pengembang Frontend:**
   - Buka halaman di peramban dan periksa *Inspect Element Console*.
   - Pastikan konsol **100% bersih dari pesan error JavaScript** berwarna merah.
   - Uji responsivitas pada mode emulasi layar HP (lebar 390px).
3. **Kebersihan Git:**
   - Pastikan tidak ada berkas temporer, log, atau berkas database `.db` yang tidak sengaja masuk ke `git status`.
