# ADR-0002 — Versioning /api/v1 + Shim Legacy

- Status: Accepted
- Tanggal: 2026-09-28
- Konteks: 7 endpoint lama tanpa versi (`/api/tasks`, `/api/schedule`, ...) dipakai bot WA + dashboard lama + `web/js/api.js` dengan fallback localStorage. Perubahan breaking akan memecah keduanya saat migrasi `DATA_MODEL`.
- Keputusan: Bekukan kontrak lama. Tambah `/api/v1/*` baru sebagai kontrak kanonis. Endpoint lama dipertahankan sebagai shim read-only (GET) + tulis terbatas (POST/DELETE tasks) yang diterjemahkan ke model baru, bertanda header `Deprecation: true` + log. Hapus hanya via migrasi terpisah setelah frontend pindah.
- Konsekuensi: Duplikasi tipis di `internal/api/` (shim vs v1). Keuntungan: rollback aman, bot WA tidak mati saat cutover DB.
