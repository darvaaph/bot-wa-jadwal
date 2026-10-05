# Arah visual Bot Jadwal

Dokumen ini adalah sumber nilai visual untuk Portal Kelas, Ketua Murid, PJ Mata Kuliah, dan System Admin. Nilai yang dipakai di kode berada di `web/css/tokens.css`; perubahan warna atau ukuran bersama dimulai dari sana.

## Identitas

Antarmuka harus terasa langsung dan tenang, dengan informasi kelas dan semester lebih menonjol daripada hiasan. Bahasa antarmuka adalah Bahasa Indonesia. Asterisk yang sudah ada tetap menjadi penanda identitas; jangan membuat identitas atau ilustrasi baru tanpa kebutuhan produk yang jelas.

Palet terkunci: primer `#3965FB`, hover `#2f54d6`, primer muda `#E9EAFF`, tinta `#1F1F1F`, garis pemisah `#EEEEEE`, garis kartu `#D7DDE7`, garis kontrol `#CBD5E1`, teks sekunder `#667085`, bahaya `#FF6C48`, kanvas dashboard `#F2F5FA`, dan sukses `#E1FFB7`. Ilustrasi yang sudah ada memakai `#9AB7FD`, `#C9F5E9`, `#FDE5A8`, `#E5D5FF`, dan `#7192F8`. Warna pendukung yang ditemukan di UI lama dicatat sebagai token kompatibilitas agar migrasi tidak mengubah tampilan diam-diam.

## Aturan pemakaian

- Gunakan kelas semantik dari `tokens.css` seperti `bg-primary`, `text-muted`, dan `border-border`, termasuk variasi status yang tercatat di sana. Jangan menambah kelas warna `-[#...]` baru.
- `--surface` untuk bidang konten, `--canvas` untuk latar lembut, `--text` untuk isi, `--primary` untuk aksi dan penanda aktif. Status tetap menyertakan teks atau ikon, bukan warna saja.
- Spasi, radius, bayangan, dan skala huruf tersedia sebagai variabel di `tokens.css`. Token ini menjadi dasar primitive Fase 2; utilitas Tailwind lama yang memakai ukuran tetap masih ada selama migrasi.
- Kontrol utama menargetkan 44 × 44 px, fokus harus terlihat, dan informasi tidak boleh terpotong pada layar 360 px. Pengujian perilaku ini dilakukan saat shell dan halaman dimigrasikan pada Fase 1–4.
- Poppins dipakai untuk judul, Montserrat untuk isi, dan JetBrains Mono untuk data monospace. Berkas WOFF2 dan Material Symbols disimpan lokal di `web/assets/fonts/` dengan fallback sistem bila font tidak dapat dimuat.

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

## Fase 1: shell bersama

`web/partials/common/shell.html` sekarang memuat sidebar, topbar, drawer, navigasi bawah, toast, kerangka muat, dan state galat untuk empat peran. `web/js/shell.js` memasang slot halaman dan menyediakan perilaku navigasi, fokus drawer, identitas peran, serta empat tujuan navigasi bawah; setiap aplikasi memasok `navSections` dan data konteksnya. Banner Mode Dukungan System Admin berada di `partials/system-admin/support-banner.html` karena hanya peran itu yang memiliki aksi dukungan.

Keputusan tata letak: konten memakai kanvas netral yang sama agar peran tidak terasa seperti aplikasi berbeda, sementara teks identitas tetap menyebut peran dan kelas aktif. Pada lebar ponsel, pencarian mengecil dalam ruang yang tersedia dan navigasi bawah tetap empat kolom dengan target sentuh 44 px.

## Fase 2: primitive

`web/css/components.css` adalah API class untuk komponen dasar. Pakai class dasar sekali per elemen; modifier hanya menyatakan maksud, bukan ukuran lokal.

| Komponen | Class dasar | Modifier atau aturan |
| --- | --- | --- |
| Button | `ui-button` | `ui-button--primary`, `--outline`, `--secondary`, `--ghost`, `--danger` |
| Input, select, textarea | `ui-field` | `aria-invalid="true"` untuk galat; textarea punya tinggi minimum |
| Card | `ui-card` | `ui-card--raised` hanya bila elevasi membantu hierarki |
| Table | `ui-table-wrap` + `ui-table` | Pembungkus menggulir horizontal jika data memang lebar |
| Pagination | `ui-pagination` | Tombol aktif memakai `aria-current="page"` |
| Badge | `ui-badge` | `--primary`, `--success`, `--danger`; label status tetap wajib |
| Dialog, drawer, toast | `ui-dialog`, `ui-drawer`, `ui-toast` | Dialog native memakai `showModal()`; drawer memakai fokus terkurung |
| Empty, skeleton, form error | `ui-empty`, `ui-skeleton`, `ui-form-error` | Error dihubungkan ke field lewat `aria-describedby` |

Ikon navigasi shell memakai SVG sebagai mask `ui-icon--*` sehingga warna mengikuti `currentColor` tanpa filter. Inline `font-family` telah diganti dengan `font-display`; body tetap menggunakan font isi dari `style.css`.

## Fase 3: halaman per peran

View Portal, KM, PJ, dan System Admin kini memakai `ui-button` pada aksi dan navigasi lokal, `ui-field` pada input teks/select/textarea, `ui-card` pada bidang konten, serta `ui-badge` dan `ui-empty` pada status yang sesuai. Input khusus seperti checkbox, unggahan berkas, dan digit PIN tetap memakai kontrol native yang ukurannya sesuai tugas; dua tabel master tetap memiliki daftar kartu di mobile dan `ui-table-wrap` di desktop. Login, undangan, dan 404 memakai primitive yang sama.

Keputusan tata letak: form pengaturan dan filter lanjutan menjadi satu kolom di ponsel agar label serta isi field tetap terbaca, lalu bertambah kolom saat ruangnya cukup. Daftar tugas dan jadwal tetap berupa kartu pada ponsel, sedangkan data tabular yang membutuhkan banyak kolom dibungkus gulir pada layar yang lebih lebar.

## Fase 4: QA dan aset offline

`web/css/fonts.css` memuat Poppins, Montserrat, JetBrains Mono, dan subset Material Symbols dari `web/assets/fonts/`. Lisensi serta daftar ikon yang masuk subset dicatat di `web/assets/fonts/README.md`; tujuh halaman utama tidak lagi meminta stylesheet Google Fonts. Referensi aset lokal pada halaman utama menggunakan versi cache `20261003` yang sama.
