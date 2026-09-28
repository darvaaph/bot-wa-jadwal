# Plan Eksekusi API v1 — Horizontal Berlapis (L0→L3)

> Keputusan terkunci: ADR-0001 s.d 0005 + `CONTEXT.md` + `docs/api/API_V1.md` (spec beku).
> Guardrails: tanpa npm/`package.json`, `embed.FS` tetap, `go test -v ./...` + `go vet ./...` + `gofmt` tiap PR,
> branch `feat/<kode>-<slug>`, Conventional Commits, dilarang push ke `main`,
> runtime di `storage/` (`bot_v1.db` baru; `tugas.db` lama read-only sampai cutover; `sesi_bot.db` jangan disentuh).

## L0 — Spec beku (selesai saat file ini + API_V1 + ADR di-merge)

- [x] `CONTEXT.md`, `docs/adr/0001-0005`, `docs/api/API_V1.md`
- Gate: review FE+QA tanda tangan bahwa envelope, field tugas, dan kode error tidak akan diubah di L1–L2.

## L1 — Skema + seed pilot (DB `storage/bot_v1.db` baru)

1. `feat/be-v1-schema` — migrasi `internal/database/migrate_v1.go` (tanpa hapus tabel lama di DB lain):
   `classes, class_settings, semesters, users, role_assignments, role_invitations, login_attempts, user_sessions, portal_sessions, recovery_tokens, courses, course_offerings, lecturers, offering_lecturers, rooms, schedule_patterns, teaching_events, teaching_event_offerings, room_confirmations, tasks, task_reviews, materials, whatsapp_channels, notification_messages, notification_attempts, audit_logs, import_batches, import_errors, backup_records` + FK + CHECK status + indeks `DATA_MODEL §11`. Test: FK aktif, 1 ACTIVE/semester per kelas, 1 OWNER/event, `end>start`.
2. `feat/be-v1-seed` — `cmd/seed-v1` + `migration_manifest.json` (1–2 kelas pilot, mapping SMT→angkatan manual) + impor courses/lecturers/offerings/patterns + checksum sumber vs target + antrean resolusi (offering tak cocok → pending, jangan dipaksa). Test: hitung baris + buka 1 portal schedule dari DB.
3. Gate L1: `go test ./internal/database/...`, backup `bot_v1.db`, demo baca langsung dari SQLite.

## L2 — Endpoint + auth + shim (satu lapis, tanpa FE)

4. `feat/be-v1-auth` — bcrypt, `login_attempts` + blokir 5/15 mnt, sesi opaque + rotasi switch-context + TTL FR + middleware `RequireAuth/Scope` + audit auth. Test: token salah 401, lintas-kelas 403, revoke efektif, pesan login generik.
5. `feat/be-v1-portal` — 6 GET portal §2 (summary/schedule/tasks/detail/changes/materials) + mode LINK/CODE + label non-warna + urut deadline. Test: arsip hanya-baca 410, kelas lain tak bocor, overdue tetap muncul.
6. `feat/be-v1-schedule` — patterns CRUD + events draf/preview/publish/revoke/participation + konflik blocking vs override + `Idempotency-Key` + UTC simpan / WIB tampil. Test: PJ lintas offering 403, KM revoke butuh reason, retry publish tak ganda, WA gagal ≠ batal.
7. `feat/be-v1-tasks` — draf/publish/patch-version/reviews-transaksional/complete/archive/restore + validasi publish minimum. Test: `409` versi basi + diff, `CHANGES_REQUESTED→DRAFT`, `REVOKED` tarik portal, versi lama teraudit.
8. `feat/be-v1-shim` — shim 7 endpoint lama → model baru + header `Deprecation: true` + audit `LEGACY_SHIM` + 501 terstruktur untuk fitur v1.1. Test: kontrak lama tak pecah + bot WA baca DB baru untuk kelas pilot.
9. Gate L2: `go test -v ./...`, `go vet ./...`, uji Insomnia seluruh tabel §1–6 API_V1, bot WA smoke test 1 grup pilot.

## L3 — QA + cutover frontend sekaligus (FE menunggu s.d sini per ADR-0005)

10. `feat/qa-v1-acceptance` — QA tulis test-case dari acceptance `FR-ACCESS/SCH/TASK/SEM` + uji 390px + keyboard/Escape + state loading/empty/error/denied/offline/conflict. Bug blocker → kembali ke tiket L2 terkait.
11. `feat/fe-v1-cutover` — ganti `web/js/api.js` ke `/api/v1` (hapus fallback localStorage untuk path v1), tampilkan konteks aktif + 409-diff + 501-v1.1, hapus stub `view-tugas.html` Kemal. Test: Chrome DevTools mobile + skenario publish PJ → review KM → portal update.
12. `feat/ops-v1-cutover` — pointer DSN bot → `bot_v1.db` untuk kelas pilot + titik pemulihan + runbook rollback (kembali ke `tugas.db`), rotasi kode portal, akun KM/PJ pilot via undangan. Go/no-go bareng PM.
13. Gate L3 (DoD rilis): semua gate L1–L2 hijau, UAT 1 kelas pilot lulus, audit publish/revoke lengkap (pelaku, waktu, before/after), tidak ada data lintas kelas bocor, rollback pernah dicoba di staging/dev.

## Perintah verifikasi tiap PR

```bash
go test -v ./...
go vet ./...
gofmt -w <file-yang-diubah>
go run ./cmd/bot -web-only   # dashboard + REST tanpa WA
```

## Risiko aktif (dipilih sadar)

- DB baru (ADR-0004): cutover ganda — mitigasi pointer DSN + backup + checksum.
- Horizontal + FE menunggu (ADR-0005): FE idle — mitigasi spec beku L0; ubah kontrak = ADR baru.
