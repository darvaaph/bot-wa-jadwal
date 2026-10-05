# ADR-0011 — Pemetaan Nama Peran FE vs BE vs URL

- Status: Accepted
- Tanggal: 2026-09-30
- Konteks: BE memakai `KM`, `PJ`, `SYSTEM_ADMIN` (uppercase, exact-match di `RequireRole`). FE punya 3 kebutuhan berbeda: label user (Bahasa Indonesia), konstanta logika (harus sama persis dengan BE), dan URL/file publik yang sudah terlanjur jadi bookmark (`/km.html`, `/pj.html`, `/superadmin.html`, `?role=km|pj|sa`).
- Keputusan: Tiga lapis dipisah. (1) Label UI selalu nama panjang Indonesia: `Ketua Murid`, `PJ Mata Kuliah`, `System Admin` (ikut `CONTEXT.md`). (2) Logika JS (`active_assignment.role`, perbandingan role, payload API) selalu `KM`/`PJ`/`SYSTEM_ADMIN` persis ikut BE. (3) Nama file readable: `system-admin.html`, `app-system-admin.js`, `partials/system-admin/`; `km` dan `pj` tetap karena singkatan resmi glosarium. URL lama `/superadmin.html` dialihkan permanen (301) ke `/system-admin.html`, bukan dihapus.
- Konsekuensi: Perlu satu titik mapping di `web/js/api.js` + komentar di `km/pj/superadmin.html`. Bookmark dan link WhatsApp lama tetap jalan. Uji: guard halaman menolak role yang salah (`PJ` tidak bisa buka `km.html`).
