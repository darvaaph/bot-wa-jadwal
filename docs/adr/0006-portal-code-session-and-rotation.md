# ADR-0006 — Sesi dan Rotasi Kode Portal

- Status: Accepted
- Tanggal: 2026-09-29
- Konteks: Kontrak API v1 sudah mewajibkan mode portal `CODE`, sesi berbasis hash, dan invalidasi sesi saat kode dirotasi, tetapi belum menetapkan endpoint untuk pertukaran maupun rotasi kode.
- Keputusan: `POST /api/v1/portal/:slug/session` menukar kode menjadi token portal berumur 30 hari. Token mentah hanya dikembalikan sekali dan database hanya menyimpan hash. Lima kegagalan per kelas dan sumber dalam 15 menit memblokir percobaan berikutnya selama 15 menit. `POST /api/v1/classes/:slug/portal-code/rotate` hanya dapat dipakai KM kelas terkait atau System Admin. Body boleh membawa kode 6–128 karakter; tanpa kode, server membuat kode numerik 8 digit. Rotasi menaikkan `portal_code_version`, mencabut sesi versi lama, dan menulis audit tanpa kode atau hash dalam satu transaksi.
- Konsekuensi: Frontend harus menyimpan token sesi portal sendiri dan mengirimkannya melalui `X-Portal-Token`. Kode hasil rotasi hanya dapat dilihat pada respons rotasi. Rotasi serentak dilindungi oleh pemeriksaan versi saat update.
