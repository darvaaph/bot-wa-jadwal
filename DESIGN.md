# Design Direction: Bot Jadwal Admin Dashboard

> Reading this as: Academic administrative dashboard for university students and class leaders, in a Clean Slate Minimalist style, dial ENERGY 1 / RHYTHM 2 / MOTION 1.

| Atribut | Nilai |
|---|---|
| Versi | 2.0.0 |
| Status | Approved |
| Terakhir diperbarui | 23 September 2026 |

---

## 1. Identitas & Karakter Desain

- **Produk:** Dashboard Web Manajemen Jadwal Kuliah, Tugas, dan WhatsApp Gateway.
- **Audiens:** Mahasiswa, PJ Mata Kuliah, Ketua Murid (KM), dan System Admin.
- **Kepribadian:** Jelas, efisien, tenang, dan profesional (mengutamakan keterbacaan data jadwal tanpa distraksi visual).
- **Masalah yang diselesaikan dari desain lama:** Menghilangkan palet hijau neon yang menyilaukan mata (*eyesore*), menghapus bayangan/glow berlebih, dan merapikan hirarki visual tabel agar nyaman dipandang lama.

---

## 2. Palet Warna (Slate & Subtle Accent)

Mengikuti prinsip R-29 (maksimal 2-3 warna inti + 1 warna aksen terukur).

### Warna Permukaan (Surface / Background)
- **Base Background:** `#0b0f17` (Deep Slate: gelap netral, tidak pekat gulita, tidak memantulkan cahaya berlebih)
- **Card / Secondary Surface:** `#121824` (Subtle elevated slate)
- **Container / Inner Box:** `#182234` (Tertiary surface untuk input, baris tabel, atau kotak detail)
- **Border Dekoratif:** `#1e293b` (Slate-800 halus untuk pemisah noninteraktif)
- **Border Kontrol:** `#52627a` (rasio 3.09:1 terhadap `#0b0f17` untuk batas input dan kontrol)

### Warna Tipografi (Kontras Teruji WCAG AA)
- **Teks Utama (Primary):** `#f8fafc` (Slate-50, kontras > 12:1 terhadap background)
- **Teks Sekunder (Muted):** `#94a3b8` (Slate-400, kontras > 4.5:1 untuk label & sub-informasi)
- **Teks Lemah (Tertiary/Meta):** `#7c8ba1` (rasio 5.54:1 terhadap `#0b0f17`, aman untuk teks normal)

### Warna Aksen Utama
- **Brand Accent:** `#3b82f6` (Blue-500 / Indigo-500: tenang, terpercaya, bukan neon)
- Digunakan hanya pada titik fokus: tombol aksi utama, tab aktif, dan status koneksi bot.

### Warna Status Semantik (Tanpa Neon, Menggunakan Opasitas Lembut)
- **Normal / Tersimpan:** Emerald lembut (`bg-emerald-500/10 text-emerald-400 border-emerald-500/20`)
- **Pindah Jadwal / Pending:** Amber lembut (`bg-amber-500/10 text-amber-400 border-amber-500/20`)
- **Kelas Kosong / Batal / Hapus:** Rose lembut (`bg-rose-500/10 text-rose-400 border-rose-500/20`)
- **Kelas Tambahan:** Sky lembut (`bg-sky-500/10 text-sky-400 border-sky-500/20`)

---

## 3. Tipografi & Hirarki Teks

- **UI & Body Text:** `Plus Jakarta Sans`, sans-serif (huruf bersih dengan x-height proporsional).
- **Data Monospace (Jam, Kode Kelas, Tanggal, Telemetri):** `JetBrains Mono`, monospace (memastikan digit jam dan tanggal sejajar secara tabular).
- **Ukuran & Berat Huruf:**
  - H1 / Brand: 14px, Bold (rapi di header/sidebar, tidak mendominasi).
  - H2 / Section Title: 18px, SemiBold.
  - Metrik Utama: 24px, Bold tabular.
  - Body / Tabel: 13px - 14px, Regular & Medium.
  - Badge & Meta: 11px - 12px, Medium/Mono.

---

## 4. Bentuk, Sudut, dan Elevasi (Border Radius & Shadows)

- **Border Radius Terukur (R-11):**
  - Kartu utama & Modal: `rounded-2xl` (16px).
  - Tombol aksi & Dropdown: `rounded-xl` (10px).
  - Badge & Filter chips: `rounded-lg` (8px).
  - *Dilarang membuat semua elemen berbentuk pil kapsul tanpa tujuan.*
- **Elevasi & Bayangan (R-12):**
  - Hindari bayangan lembut mengambang (*floating blur*) di setiap kartu.
  - Kedalaman dibangun menggunakan perbedaan warna latar belakang (`#0b0f17` vs `#121824`) dan border 1px halus (`#1e293b`).
  - Bayangan hanya digunakan pada Modal dialog untuk memisahkannya dari overlay latar belakang (`shadow-2xl`).

---

## 5. Antislop Dials & Motion

- **ENERGY (1 / Calm):** Tampilan tenang dan utilitarian. Informasi langsung tersampaikan tanpa dekorasi visual artifisial (tanpa mesh gradient, tanpa particle orb, tanpa strip warna kiri dekoratif).
- **RHYTHM (2 / Structured):** Struktur bagian teratur dan konsisten antara Ringkasan, Jadwal, dan Tugas, dengan tata letak tabular yang mudah dipindai mata.
- **MOTION (1 / Subtle Transitions):**
  - Transisi warna dan opasitas cepat (150ms `ease-out`) saat hover tombol dan ganti tab.
  - Tidak ada animasi fade-up melompat-lompat atau efek bounce yang memperlambat alur kerja.

---

## 6. Standar Aksesibilitas & Fungsionalitas

- Seluruh tombol dan area klik minimal berukuran 44 x 44 piksel.
- Status fokus keyboard terlihat jelas (`focus:ring-2 focus:ring-blue-500 focus:outline-none`).
- Setiap tombol dan kontrol memiliki fungsi nyata (tidak ada kontrol mati atau navigasi palsu).
- Modal dapat ditutup dengan tombol Batal, klik di luar, maupun tombol Escape.
- Setiap tampilan data menyediakan state loading, empty, error, success, dan permission denied dengan teks yang tidak bergantung pada warna.
- Teks harus tetap terbaca dan tidak terpotong pada zoom 200 persen.

## 7. Changelog

### 2.0.0, 23 September 2026

- Menyelaraskan audiens dengan peran produk yang disetujui.
- Mengganti warna metadata yang gagal WCAG AA dan membedakan border dekoratif dari batas kontrol.
- Menetapkan target sentuh minimum 44 x 44 piksel serta state dan zoom aksesibel.
