# TICKET-007: Timezone + Waktu Pengingat (FR-ACCESS-001, FR-NOTIF-005, SCR-SET-001)

## Scope
KM atur timezone kelas, waktu pengingat pagi/sore, replacement reminder minutes. Di `class_settings` table: `timezone`, `morning_reminder_time`, `afternoon_reminder_time`, `replacement_reminder_minutes`.

## Backend Status
- **SUDAH ADA**: `GET /api/v1/classes/{slug}/settings` (return timezone, portal_access_mode, portal_code_version)
- **BELUM ADA**: `PATCH /api/v1/classes/{slug}/settings` — update `morning_reminder_time`, `afternoon_reminder_time`, `replacement_reminder_minutes`, `timezone`
- **SUDAH ADA**: `notify.go` baca settings per kelas untuk pengiriman reminder

## Frontend Status
- `view-pengaturan.html`: tampilkan waktu pengingat read-only + tombol "Minta perubahan ke admin"
- **BELUM**: Form edit timezone + pagi/sore + replacement minutes

## Required (P2)
- [ ] **Backend**: `PATCH /api/v1/classes/{slug}/settings` body `{ timezone?, morning_reminder_time?, afternoon_reminder_time?, replacement_reminder_minutes? }` — validasi format HH:MM, timezone valid IANA, reminder_minutes 0-1440
- [ ] **Backend**: Audit log `UPDATE_CLASS_SETTINGS` dengan before/after
- [ ] **Frontend `view-pengaturan.html`**: ganti read-only jadi form edit:
  - Timezone: select IANA (default Asia/Jakarta)
  - Ringkasan pagi: time input (default 06:00)
  - Pengingat sore: time input (default 17:00)
  - Pengingat pengganti: number input menit (default 60)
  - Tombol "Simpan" → `API.updateClassSettings()` + toast
- [ ] **Frontend `app-km.js`**: `updateClassSettings(payload)` call endpoint, `loadPengaturanKelas()` reload
- [ ] **Frontend**: Hapus tombol "Minta perubahan ke admin" + `infoPengingat()` toast

## Acceptance Criteria
- KM bisa atur timezone kelas (IANA select), pagi/sore (HH:MM), pengganti (menit)
- Perubahan berlaku pengiriman berikutnya (notify.go baca DB per kelas)
- Validasi: timezone valid, HH:MM format, menit 0-1440
- Audit log before/after tersimpan
- Toast "Pengaturan kelas disimpan" on success

## Related
- `docs/product/FUNCTIONAL_REQUIREMENTS.md` FR-NOTIF-005
- `docs/product/ACCESS_CONTROL.md` §8.4 (KM atur pengingat)
- `docs/design/SCREEN_INVENTORY.md` SCR-SET-001
- `internal/notify/notify.go` (baca settings per kelas)
- `internal/api/v1/portal_controller.go` (GetClassSettings)