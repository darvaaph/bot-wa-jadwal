# ADR-0005 — Eksekusi Horizontal Berlapis, Frontend Menunggu Backend 100%

- Status: Accepted
- Tanggal: 2026-09-28
- Konteks: Opsi: (A) vertical slice per tiket (frontend cepat nyambung), (B) horizontal (semua DB → semua endpoint → frontend), (C) sekaligus. Pemilik memilih (B)+(tunggu backend jadi) agar kontrak tidak berubah-ubah di tengah jalan.
- Keputusan: Jalankan 4 lapis berurutan dengan gate QA: L0 spec beku (file ini + OpenAPI) → L1 skema + seed pilot → L2 endpoint + auth + shim → L3 QA + cutover frontend sekaligus. Frontend tidak mock paralel; sebagai gantinya L0 harus benar-benar beku (field `title/instructions/deadline_at`, envelope, kode error) agar tunggu tidak sia-sia.
- Konsekuensi: Frontend idle selama L1–L2. Mitigasi: L0 direview FE+QA sebelum L1 mulai; perubahan kontrak setelah L0 wajib ADR baru + bump `openapi minor`. Keuntungan: integrasi akhir sekali jalan, tanpa drift mock vs nyata.
