# TICKET-008: Dashboard Ringkasan Agregat (SCR-APP-002, IA §6.4)

## Scope
Dashboard KM agregat: Perlu Review, Undangan menunggu (lintas kelas), Konflik jadwal, Notifikasi gagal. `IA §6.4`: "Ringkasan KM: perlu-review, undangan lintas kelas, konflik, notif gagal, publikasi terbaru."

## Backend Status
- **SUDAH ADA**: 
  - `GET /api/v1/tasks` (filter `review_state=NOT_REVIEWED`) → `antrean.length`
  - `GET /api/v1/admin/invitations?status=PENDING&class_slug=` → undangan pending
  - `GET /api/v1/notifications?status=FAILED&class_slug=` → notif gagal
  - `GET /api/v1/teaching-events?status=DRAFT` + preview konflik → konflik

## Frontend Status
- `view-dashboard.html`: kartu "Perlu perhatian" (Review + Notif gagal) + Ringkasan 4 kartu (Tugas, Jadwal, Materi, Bot)
- **BELUM**: Agregat Undangan menunggu, Konflik jadwal, Publikasi terbaru

## Required (P2)
- [ ] **Frontend `app-km.js`** getters:
  - `get undanganMenunggu()` → count `GET /admin/invitations?status=PENDING` (filter kelas KM)
  - `get konflikJadwal()` → count preview Konflik blocking dari draf event (butuh endpoint preview batch atau client-side hitung dari `eventsList` filter `DRAFT` + `conflicts.blocking`)
  - `get publikasiTerbaru()` → 5 `teaching_events` `PUBLISHED` terbaru + 5 `tasks` `PUBLISHED` terbaru
- [ ] **Frontend `view-dashboard.html`**: Tambah di section "Perlu perhatian":
  - Kartu "Undangan menunggu" (badge count + link ke `anggota` tab undangan)
  - Kartu "Konflik jadwal" (badge count + link ke `jadwal` tab tinjau)
  - Kartu "Publikasi terbaru" (list 3 terbaru event/task + link)
- [ ] **Frontend**: Load data di `initKM()` → `loadUndanganKM()`, `loadEvents()` (sudah), tambah `loadPublikasiTerbaru()` jika perlu endpoint terpisah

## Acceptance Criteria
- Dashboard menampilkan 5 kartu "Perlu perhatian": Perlu Review, Undangan menunggu, Konflik jadwal, Notifikasi gagal, Publikasi terbaru
- Setiap kartu: count badge + CTA link ke halaman detail
- Data load async, tidak blokir render dashboard
- Error handling graceful (kartu kosong + toast error)

## Related
- `docs/product/INFORMATION_ARCHITECTURE.md` §6.4
- `docs/design/SCREEN_INVENTORY.md` SCR-APP-002