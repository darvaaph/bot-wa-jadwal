# Prosedur Restore Penuh (Operasional, dengan Downtime)

> Status: prosedural. `POST /api/v1/restores` adalah verify-only per ADR-0008
> dan tidak mengganti database aktif. Penggantian berkas database hanya lewat
> prosedur di bawah ini, oleh operator, saat jendela henti yang diumumkan.
> Glosarium memakai `CONTEXT.md`.

## Prasyarat

1. ID cadangan dari `GET /api/v1/backups` (catat `id`, `checksum`, `class_id`,
   `semester_id`, `status`).
2. Verifikasi lolos: `POST /api/v1/restores {"backup_id": <id>, "reason": "<alasan>"}`
   → `status: VERIFIED`, `restore_performed: false`, checksum cocok.
3. Jendela henti diumumkan; pengurus diminta tidak mempublikasi selama restore.

## Langkah

1. Hentikan proses bot (`systemd stop` / hentikan biner) agar tak ada penulis.
2. Salin database live ke tempat aman (titik pemulihan darurat):
   `cp storage/sesi_bot.db storage/manual-backup-<tanggal>.db`
3. Catat checksum live (`sha256sum`) untuk audit.
4. Ambil `artifact_ref` + `checksum` dari `backup_records` untuk ID cadangan
   (akses DB langsung di host database, hanya baca):
   `SELECT artifact_ref, checksum FROM backup_records WHERE id = <id>;`
   Jalur berkas tidak boleh keluar dari host (jangan ditempel ke chat/tiket).
5. Cocokkan checksum berkas dengan catatan. Berhenti bila beda.
6. Salin artefak menimpa database live. Jangan hapus artefak sumber.
7. Nyalakan ulang bot; login sebagai System Admin.
8. Periksa: `GET /api/v1/admin/status`, buka Portal Kelas + Detail Kelas yang
   dipulihkan, dan `GET /api/v1/audit?action=VERIFY_RESTORE_BACKUP`.
9. Catat di berkas log operasional (mis. `storage/restore-ops.log`, satu baris
   per restore): `waktu | operator | backup_id | checksum | alasan`.
   Contoh: `2026-10-02T03:10:00Z | admin-utama | 12 | ab12.. | pulihkan semester Ganjil`.

## Rollback

Bila langkah 7–8 gagal: hentikan bot, kembalikan salinan langkah 2, nyalakan
ulang, verifikasi ulang. Jangan menghapus salinan darurat sebelum jendela
pemantauan (min. 1x24 jam) selesai.

## Larangan

- Jangan restore ke kelas yang salah: cocokkan `class_id`/`semester_id`
  catatan dengan target sebelum langkah 6.
- Jangan restore dari berkas tanpa catatan `backup_records` + checksum cocok.
- Jangan lakukan saat bot berjalan (risiko DB terkunci/rusak di Windows).
