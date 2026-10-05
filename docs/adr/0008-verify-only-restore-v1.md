# ADR-0008 — Restore v1 Bersifat Verifikasi Saja

- Status: Accepted.
- Tanggal: 2026-09-29
- Konteks: `docs/api/API_V1.md` mendefinisikan `POST /api/v1/restores` sebagai verifikasi berkas fisik dan checksum, sedangkan `docs/product/FUNCTIONAL_REQUIREMENTS.md` bagian FR-OPS-002 membahas restore terbatas cakupan dengan rollback. Tanpa keputusan eksplisit, frontend, operator, dan response API memakai istilah "pulihkan" untuk operasi yang tidak mengganti database aktif.
- Keputusan: untuk API v1, `POST /api/v1/restores` adalah verify-only (Opsi A ticket BE-011). Endpoint memverifikasi path dalam storage backup yang dikonfigurasi (tolak traversal/symlink escape), checksum, format SQLite, kompatibilitas schema, dan metadata scope; lalu menandai backup sebagai `VERIFIED` dalam transaksi dengan audit BE-009. Endpoint tidak mengganti database aktif. Response menyatakan `restore_performed: false` dan status akhir `VERIFIED` berarti lolos verifikasi, bukan database sudah dipulihkan. Full restore menjadi ticket operasi dan versi API terpisah.
- Konsekuensi: endpoint ini tetap berlabel "Verifikasi Arsip" dan memerlukan alasan. Sejak ADR-0009, paket akademik v2 dapat diverifikasi di sini tanpa mutasi data; pemulihan nyata hanya melalui `POST /api/v1/backups/:id/restore-execute`. Arsip SQLite v1 tetap hanya dapat diverifikasi.
