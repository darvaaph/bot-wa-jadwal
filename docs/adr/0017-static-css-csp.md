# ADR-0017 — CSS statis dan pengetatan CSP bertahap

- Status: Accepted
- Tanggal: 2026-10-14
- Konteks: Tailwind Play dijalankan saat halaman dibuka dan memerlukan kelonggaran CSP. Skrip tema, pemeriksaan token lama, serta CSS kritis login juga berada langsung di HTML. Pemeriksaan token lama tidak sesuai dengan autentikasi cookie.
- Keputusan: Tailwind v3.4.17 dihasilkan sekali menjadi `web/css/tailwind.css` dari HTML/JS repo dan disematkan dalam biner. Runtime Tailwind dilepas. Skrip tema dan CSS login menjadi berkas lokal. Pemeriksaan token `localStorage` pada halaman KM, PJ, dan System Admin dihapus karena inisialisasi aplikasi sudah memeriksa sesi melalui `/api/v1/auth/me`.
- CSP sekarang membatasi skrip, stylesheet, dan font ke origin sendiri serta menolak skrip inline. Pengecualian `unsafe-eval` sementara masih dibutuhkan oleh Alpine standar. Uji browser dengan build Alpine CSP 3.17.4 memperlihatkan ekspresi pada portal tidak didukung, terutama optional chaining; karena itu penggantian tersebut belum diterapkan. Rilis publik memerlukan migrasi ekspresi dan uji semua peran sebelum `unsafe-eval` dapat dihapus.
- Tidak ada npm, `package.json`, atau build frontend pada proses menjalankan atau merilis aplikasi. Pembaruan CSS adalah pemeliharaan aset vendored dan hasilnya dikomit.
