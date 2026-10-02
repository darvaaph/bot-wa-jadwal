# Arah visual Bot Jadwal

Dokumen ini adalah sumber nilai visual untuk Portal Kelas, Ketua Murid, PJ Mata Kuliah, dan System Admin. Nilai yang dipakai di kode berada di `web/css/tokens.css`; perubahan warna atau ukuran bersama dimulai dari sana.

## Identitas

Antarmuka harus terasa langsung dan tenang, dengan informasi kelas dan semester lebih menonjol daripada hiasan. Bahasa antarmuka adalah Bahasa Indonesia. Asterisk yang sudah ada tetap menjadi penanda identitas; jangan membuat identitas atau ilustrasi baru tanpa kebutuhan produk yang jelas.

Palet terkunci: primer `#3965FB`, hover `#2f54d6`, primer muda `#E9EAFF`, tinta `#1F1F1F`, garis `#EEEEEE`, teks sekunder `#667085`, bahaya `#FF6C48`, kanvas KM `#F8FAFC`, dan sukses `#E1FFB7`. Ilustrasi yang sudah ada memakai `#9AB7FD`, `#C9F5E9`, `#FDE5A8`, `#E5D5FF`, dan `#7192F8`. Warna pendukung yang ditemukan di UI lama dicatat sebagai token kompatibilitas agar migrasi tidak mengubah tampilan diam-diam.

## Aturan pemakaian

- Gunakan kelas semantik dari `tokens.css` seperti `bg-primary`, `text-muted`, dan `border-border`, termasuk variasi status yang tercatat di sana. Jangan menambah kelas warna `-[#...]` baru.
- `--surface` untuk bidang konten, `--canvas` untuk latar lembut, `--text` untuk isi, `--primary` untuk aksi dan penanda aktif. Status tetap menyertakan teks atau ikon, bukan warna saja.
- Spasi, radius, bayangan, dan skala huruf tersedia sebagai variabel di `tokens.css`. Token ini menjadi dasar primitive Fase 2; utilitas Tailwind lama yang memakai ukuran tetap masih ada selama migrasi.
- Kontrol utama menargetkan 44 × 44 px, fokus harus terlihat, dan informasi tidak boleh terpotong pada layar 360 px. Pengujian perilaku ini dilakukan saat shell dan halaman dimigrasikan pada Fase 1–4.
- Poppins dipakai untuk judul dan Montserrat untuk isi selama font tersedia; fallback sistem dipakai saat luring. Penggantian sumber font eksternal adalah pekerjaan fase berikutnya.

Keputusan tata letak: sidebar desktop 248 px menjaga label tetap terbaca, sementara mode 76 px memberi ruang untuk data tabel. Pada ponsel, navigasi bawah empat tujuan mempertahankan akses ke operasi utama tanpa mengorbankan area baca.

## Inventaris shell sebelum unifikasi

| Bagian | Salinan saat ini | Catatan migrasi Fase 1 |
| --- | --- | --- |
| Sidebar | `partials/portal/sidebar.html`, `partials/km/sidebar.html`, `partials/system-admin/sidebar.html`, `partials/common/sidebar.html` (PJ) | Empat implementasi; menu dan identitas berbeda per peran. |
| Topbar | `partials/portal/topbar.html`, `partials/km/topbar.html`, `partials/system-admin/topbar.html`, `partials/common/topbar.html` (PJ) | Search, bell, dan identitas belum seragam. |
| Drawer | `partials/portal/drawer.html`, `partials/system-admin/drawer.html`, `partials/common/drawer.html` (KM/PJ) | Overlay dan daftar menu perlu satu kontrak. |
| Bottombar | `partials/portal/bottombar.html`, `partials/system-admin/bottombar.html`, `partials/common/bottombar.html` (KM/PJ) | Empat tujuan perlu satu pola aktif. |
| Toast | `partials/portal/toast.html`, `partials/system-admin/toast.html`, `partials/common/toast.html` (KM/PJ) | Posisi dan pengumuman perlu disatukan. |
| Keadaan muat/galat | `partials/common/skeleton-dashboard.html`, `skeleton-dashboard-sa.html`, `state-error.html`; slot `*-state` di tiap shell | Variasi per peran dan halaman belum satu API. |
| Kerangka halaman | `index.html`, `km.html`, `pj.html`, `system-admin.html` | Sidebar, topbar, main, drawer, bottombar, dan toast dirangkai ulang per halaman. |

## Batas Fase 0

Fase ini hanya memusatkan nilai visual dan mencatat duplikasi. Shell, sumber font, inline style, kontrak navigasi, serta perilaku responsif yang ada belum diubah; itu pekerjaan Fase 1–4 setelah review.
