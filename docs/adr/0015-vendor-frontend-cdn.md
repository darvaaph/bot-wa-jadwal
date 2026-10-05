# ADR-0015 — Vendor Tailwind dan Alpine Same-Origin

- Status: Accepted
- Tanggal: 2026-10-01
- Konteks: Tailwind Play CDN dan Alpine dimuat dari `cdn.tailwindcss.com`/`cdn.jsdelivr.net`. Browser HP dengan pelindung pelacakan (terlihat di DuckDuckGo) memblokir domain tersebut: CSS utilitas tidak jalan, sidebar desktop tampil di HP, layout rusak. Di laptop tanpa pemblokir semua normal sehingga sulit direproduksi.
- Keputusan: (1) Vendor kedua bundle ke `web/js/vendor/` (disematkan via `embed.go`), halaman memuat dari origin sendiri. (2) Tanpa langkah build/npm: berkas disalin apa adanya, diperbarui manual bila perlu. (3) CSP `script-src` diketatkan (host CDN dihapus). Google Fonts tetap CDN (hanya tipografi, degradasi aman). Keputusan font ini digantikan oleh ADR-0016.
- Konsekuensi: biner +452KB; pembaruan versi manual. Uji: blokir domain CDN di DevTools → layout tetap benar; `TestWebKnownPage`-setara untuk `/js/vendor/*` 200.
