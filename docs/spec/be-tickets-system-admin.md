# Tiket Backend Area System Admin

> Sumber: hasil bedah + batch 1–3 frontend System Admin (Okt 2026).
> Glosarium memakai `CONTEXT.md`. Acuan spek: `docs/product/INFORMATION_ARCHITECTURE.md` (§7.2, §10–12),
> `docs/product/FUNCTIONAL_REQUIREMENTS.md`, `docs/product/ACCESS_CONTROL.md`.
> Status frontend: setiap tiket yang belum tersedia dinyatakan jujur di UI
> (empty-state / catatan keterbatasan) — frontend tidak mengarang data.

## Urutan saran

1. BE-01 + BE-02 (membuka tab Penugasan/Undangan yang kini placeholder)
2. BE-10 + BE-11 (melengkapi Backup dan Pemulihan)
3. BE-04 (dukungan sejati) + BE-12 (filter audit waktu/pelaku)
4. BE-05, BE-06, BE-07, BE-08, BE-09, BE-03, BE-13

---

## Akses, peran, undangan

### BE-01 — Daftar + siklus Penugasan Peran

> Status: **Selesai** — `GET /api/v1/admin/assignments`, `POST .../assignments/{id}/suspend|revoke`
> (`internal/api/v1/admin_assignments.go`, route di `internal/api/routes.go`),
> tes di `internal/api/admin_assignments_test.go`. Frontend belum diwiring (tab masih placeholder).

- **Problem:** `GET /api/v1/admin/users` hanya kembalikan `id/identity_key/display_name/status/roles`
  (`internal/api/v1/admin_controller.go:67-74`). Tab Penugasan tak bisa tampilkan
  scope kelas/mata kuliah, status, dan masa berlaku.
- **Kebutuhan:** `GET` assignments (pengguna, peran, scope kelas/mata kuliah,
  status `ACTIVE/SUSPENDED/REVOKED`, `valid_from/until`) + aksi tangguhkan/cabut
  penugasan (terpisah dari tangguhkan akun). Guard ganti KM tunggal tetap di frontend.
- **Acuan:** FR-ACCESS-007, IA §7.2.
- **Frontend kini:** `web/partials/system-admin/view-pengguna.html` tab Penugasan = empty-state jujur.

### BE-02 — Daftar + cabut Undangan

> Status: **Selesai** — `GET /api/v1/admin/invitations` (+ filter derivasi `status=EXPIRED`),
> `POST .../invitations/{id}/revoke` (file dan tes yang sama dengan BE-01).
> Frontend belum diwiring (tab masih placeholder).

- **Problem:** hanya ada buat (`POST /api/v1/invitations`) dan terima; tak ada daftar,
  status, maupun cabut. Kirim ulang sudah membatalkan token lama
  (`internal/api/v1/auth_controller.go:947-951`).
- **Kebutuhan:** `GET` invitations (status `PENDING/ACCEPTED/EXPIRED/REVOKED` + kedaluwarsa)
  + revoke. TTL 7 hari sudah benar (`invitationTTL`, `auth_controller.go:959`).
- **Acuan:** FR-ACCESS-003, IA §7.2.
- **Frontend kini:** tab Undangan = empty-state jujur + info TTL + tombol buat dari Daftar Kelas.

### BE-03 — Undangan System Admin tambahan

- **Problem:** `POST /api/v1/invitations` menolak role selain KM/PJ
  (`internal/api/v1/auth_controller.go:914-918`), padahal kebijakan membolehkan
  System Admin tambahan.
- **Kebutuhan:** izinkan role `SYSTEM_ADMIN` sesuai kebijakan tim.
- **Acuan:** ACCESS_CONTROL §6.1.

### BE-04 — Konteks dukungan sejati (break-glass)

> Status: **Selesai sebagian** — tabel `support_grants` (migrasi 010) + hibah 60 mnt
> (`POST enter` alasan min 10, `POST exit`, `GET active`; supersede + kedaluwarsa
> beraudit `SUPPORT_EXIT`) + wiring banner server. Tes di
> `internal/api/admin_support_test.go`.
> **Batas jujur**: atribusi `grant_id` per mutasi akademik belum ada — mutasi SA
> tetap lewat hak global dan terkorelasi via timeline `SUPPORT_ENTER/EXIT`.
> Lanjutan: plumbing konteks dukungan ke audit tiap handler akademik.

- **Problem:** `masukDukungan()` hanya flag frontend + alasan wajib + banner global.
  Tak ada konteks dukungan di server.
- **Kebutuhan:** endpoint masuk dukungan (kelas + alasan min. 10 karakter → konteks
  sementara beraudit); setiap mutasi selama mode tercatat sebagai akses dukungan.
- **Acuan:** FR-ACCESS-004, ACCESS_CONTROL §8.

## Kelas

### BE-05 — Ubah mode Portal Kelas

> Status: **Selesai** — `GET .../classes/{slug}/settings`,
> `PATCH .../portal-mode` (LINK/CODE + audit, CODE wajib hash aktif) + wiring
> Detail Kelas (mode/zona/versi + tombol + rotasi). Tes di
> `internal/api/admin_portal_master_test.go`.

- **Problem:** `PATCH /api/v1/classes/{slug}` hanya menerima status; tak ada endpoint
  ganti mode `LINK`/`CODE`. UI tampilkan default server (`LINK`, `Asia/Jakarta`) readonly.
- **Kebutuhan:** endpoint ubah `portal_access_mode` (+ audit).
- **Acuan:** FR-CLASS-001, IA §7.2.

## Master + ruangan

### BE-06 — Audit perubahan master

> Status: **Selesai** — `CREATE/UPDATE_MASTER_ROOM/COURSE` + before/after di semua
> tulis master; KM tulis tetap 403. Terlihat di Audit Global (`entity_type`
> `MASTER_ROOM`/`MASTER_COURSE`).

- **Problem:** `PATCH /api/v1/master/rooms/{id}` dan `/courses/{id}` tak tulis audit.
- **Kebutuhan:** setiap tambah/ubah/nonaktif master masuk Riwayat Perubahan.
- **Acuan:** FR-ROOM-003 ("Perubahan master masuk audit log").

### BE-07 — Usulan koreksi KM

> Status: **Selesai sebagian** — tabel `master_proposals` (migrasi 011) + alur penuh
> (KM usul, SA setujui/diterapkan/tolak + audit ganda) + panel review di kedua
> halaman master SA. Tes di `internal/api/admin_portal_master_test.go`.
> **Belum**: form usul di Area KM (`km.html`) — KM kini via API langsung.
> Diketahui: pencarian kandidat hanya tampilkan ruangan ACTIVE (penegakan utama);
> impor mentoleransi kode tak dikenal; konfirmasi TU adalah override manusia.

- **Problem:** tak ada jalur KM mengusulkan koreksi master tanpa ubah langsung.
- **Kebutuhan:** endpoint usulan + persetujuan/penolakan System Admin.
- **Acuan:** FR-ROOM-003.

## Notifikasi

### BE-08 — Filter server antrean

- **Problem:** `GET /api/v1/notifications` hanya filter `status` (+ `limit/offset`);
  filter kelas/jenis/b Waktu dikerjakan lokal di `filteredNotif()`.
- **Kebutuhan:** filter `class_id`, `event_type`, rentang waktu + sort di server.
- **Acuan:** IA §10, FR-NOTIF-004.

### BE-09 — Detail pesan

- **Problem:** respons tak memuat penerima, `idempotency_key`, dan riwayat percobaan;
  UI hanya tampilkan payload mentah.
- **Kebutuhan:** sertakan penerima, `idempotency_key`, dan percobaan per pesan.
- **Acuan:** FR-NOTIF-004, IA §9.1.

## Backup / restore

### BE-10 — Daftar cadangan

> Status: **Selesai** — `GET /api/v1/backups` (filter class_slug/status; KM kelasnya;
> tanpa artifact_ref) + wiring daftar di UI. Tes di `internal/api/admin_backups_test.go`.

- **Problem:** tak ada `GET` backups; UI suruh pengguna mencatat ID manual.
- **Kebutuhan:** daftar cadangan per kelas (ID, status, checksum, waktu, alasan).
- **Acuan:** FR-OPS-002.

### BE-11 — Cakupan + restore aktual

> Status: **Selesai sebagian (batas ADR-0008)** — cakupan semester (`semester_id`
> tervalidasi + tersimpan + diverifikasi), cek relasi `foreign_key_check`, respons
> verify menggema cakupan, audit create/verify memuat alasan + semester tanpa path,
> KM luar cakupan 404, wiring UI (select semester + daftar + verifikasi per baris).
> **Tidak dikerjakan**: paket per semester, titik pemulihan, restore pengganti DB —
> restore aktual tetap tiket operasi/API terpisah per ADR-0008 (ganti file DB live
> butuh prosedur downtime, bukan operasi HTTP).

- **Problem:** `POST /api/v1/backups` hanya snapshot per kelas (tanpa semester/paket);
  `POST /api/v1/restores` verify-only tanpa ubah DB (ADR-0008).
- **Kebutuhan:** pilih semester/paket, tinjau cakupan, titik pemulihan sebelum restore,
  verifikasi relasi + kelas tujuan, rollback saat gagal, audit.
- **Acuan:** FR-OPS-002, IA §7.2.
- **Frontend kini:** copy verify-only jujur ("Database aktif tidak diubah").

## Audit

### BE-12 — Filter waktu + pelaku

> Status: **Selesai** — `since/until` (RFC3339/YYYY-MM-DD, presisi detik UTC;
> tanggal saja = akhir hari untuk `until`), `actor` (ID atau identity_key),
> `entity_id`, slug asing untuk KM 404 + wiring filter + reset di Audit Global.
> Lanjutan di luar tiket: cakupan PJ atas audit matkulnya (route kini KM/SA saja).

- **Problem:** `GET /api/v1/audit` hanya `class_slug/action/entity_type`.
- **Kebutuhan:** rentang waktu + pelaku (+ `entity_id` bila memungkinkan).
- **Acuan:** FR-AUDIT-002.

## Bot / sistem

### BE-13 — Sambung ulang via web

- **Problem:** tak ada endpoint QR/test-kirim/reconnect; UI hanya prosedur terminal-server.
- **Kebutuhan:** penyambungan ulang + uji kirim untuk System Admin sesuai prosedur.
- **Acuan:** IA §7.2.
