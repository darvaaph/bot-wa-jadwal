# ADR-0013 — State Galat Halaman (500/Luring/404)

- Status: Accepted
- Tanggal: 2026-10-01
- Konteks: Galat hanya tampil sebagai toast/teks inline; tidak ada pola visual untuk gagal total (server 5xx, putus koneksi, halaman/data tak ditemukan). Desain menetapkan 3 state: lingkaran `#E9EAFF` + angka/ikon `#3965FB`, judul Poppins, tombol `Coba lagi`/`Kembali ke Dashboard`.
- Keputusan: (1) Satu partial `web/partials/common/state-error.html` dipakai 4 area, dirender saat `view === '__error'` + `pageState.status`. (2) Tiap app punya `pageState`, `knownViews`, `showPageError(status)`; `go()` membersihkan `pageState` dan menampilkan 404 untuk view tak dikenal. (3) Listener `offline` menampilkan state luring, `online` memuat ulang jika status luring. (4) `Coba lagi` = `location.reload()`; `Kembali ke Dashboard` = `go('dashboard')`. (5) Data 404/500 dipetakan dari `err.code` API (`NOT_FOUND` → 404, `LOAD_FAILED` → 500); galat level-daftar tetap inline, hanya konteks level-halaman (detail, gerbang) memakai state penuh.
- Konsekuensi: pola baru wajib dipakai untuk pemuatan level-halaman berikutnya; `knownViews` harus dijaga saat menambah view. Uji: `go('tak-ada')` → 404; DevTools offline → luring; detail tugas terhapus → 404.
- Catatan 2026-10-01: path halaman yang tidak ada di server (mis. salah ketik `*.html`) tidak mencapai SPA. `routes.go` kini menyajikan `web/404.html` mandiri (tanpa JS) dengan status 404 untuk path `*.html`/tanpa ekstensi; API dan aset tetap 404 semula. Uji: `TestWebUnknownHTML_ServesCustom404`.
