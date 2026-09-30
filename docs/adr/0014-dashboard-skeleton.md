# ADR-0014 — Skeleton Dashboard

- Status: Accepted
- Tanggal: 2026-10-01
- Konteks: Muat awal dashboard menampilkan angka nol lalu melompat saat data tiba; teks `Memuat...` tidak menahan ruang. Alternatif: spinner penuh, teks saja, skeleton per halaman.
- Keputusan: (1) Satu partial `web/partials/common/skeleton-dashboard.html` (bar judul + 3 kartu + panel, `animate-pulse` flat tanpa shimmer) dipakai 4 dashboard. (2) Flag `dashboardLoading` (awal `true`, mati di akhir `init*`) khusus muat awal; refresh kecil tetap pakai teks/spinner inline yang ada agar tidak kedip. (3) Skeleton dimuat susulan setelah partial dashboard karena slotnya bersarang. (4) `role="status"` + teks sr-only agar screen reader tahu yang dimuat.
- Konsekuensi: pola siap dipakai ulang untuk list/tabel; `dashboardLoading` tidak menutupi galat (state galat ADR-0013 tetap prioritas via `view`). Uji: throttle network → kerangka tampil dulu, lalu konten; tanpa JS rusak tidak ada (skeleton butuh Alpine, konten statis tetap nol).
