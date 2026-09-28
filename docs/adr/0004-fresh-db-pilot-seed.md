# ADR-0004 — DB Baru Bersih + Seed 1–2 Kelas Pilot

- Status: Accepted
- Tanggal: 2026-09-28
- Konteks: Opsi: (A) in-place bertahap di `storage/tugas.db`, (B) DB baru bersih + impor ulang, (C) tambal skema lama `scope_jid`. (A) paling aman untuk bot, tapi pemilik memilih (B) agar skema target murni tanpa warisan `scope_jid`.
- Keputusan: Bangun `storage/bot_v1.db` baru dengan skema target (tahap 1–5 `DATA_MODEL §15`). Seed hanya 1–2 kelas pilot (mis. `D4-TI-2024-A` + satu lagi) dari `data/jadwal/*.json` via skrip impor + `migration_manifest` eksplisit (program, angkatan, rombel, tahun akademik, semester aktif awal). `DATA_MODEL §14 ayat 2` melarang menebak angkatan dari `SMT3` — mapping harus ditulis manusia di manifest. DB lama `storage/tugas.db` dipertahankan read-only selama transisi; bot WA tetap menunjuk DB lama sampai cutover diumumkan.
- Konsekuensi: Risiko cutover (dua DB hidup paralel). Wajib: checksum jumlah sumber vs target, antrean resolusi offering tak cocok (jangan dipaksakan), titik pemulihan sebelum cutover, rollback = kembalikan pointer DSN ke DB lama.
