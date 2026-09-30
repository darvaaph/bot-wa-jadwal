# CONTEXT.md — Glosarium Kanonis Bot Jadwal

> Sumber: `docs/product/INFORMATION_ARCHITECTURE.md §13`, `docs/product/ACCESS_CONTROL.md`,
> `docs/product/DATA_MODEL.md`, `docs/PRD.md`. File ini hanya glosarium, tanpa detail implementasi.

| Istilah (gunakan) | Jangan gunakan | Arti |
|---|---|---|
| Ketua Murid (KM) | Admin kelas, Komti | Pengurus scope `CLASS`: seluruh mata kuliah pada kelasnya, lintas semester selama assignment aktif |
| PJ Mata Kuliah | Admin mata kuliah | Pengurus scope `COURSE_OFFERING`: satu mata kuliah pada satu semester kelas |
| System Admin | Admin sistem (umum) | Pengurus scope `GLOBAL`: kelas/semester lintas kelas + dukungan break-glass beraudit |
| Kelas | Grup, room chat | Identitas permanen: program studi + angkatan + rombel (`code`, `slug`) |
| Semester Kelas | Semester (tanpa kelas) | Periode `DRAFT/ACTIVE/ARCHIVED` milik satu kelas; hanya satu `ACTIVE` per kelas |
| Course Offering | Mata kuliah, kelas mata kuliah | Pelaksanaan satu mata kuliah (teori/praktik terpisah via `activity_type`) pada satu semester |
| Pola Jadwal | Jadwal reguler | Aturan berulang (`schedule_patterns`: hari, jam, ruangan, rentang efektif, versi) |
| Kejadian Perkuliahan | Override, schedule change | Kejadian aktual (`teaching_events`): `REPLACEMENT/EXTRA/HOLIDAY/SESSION_CANCELLED` + lifecycle `DRAFT/PUBLISHED/REVOKED` |
| Terbit | Aktif (untuk status publikasi) | Data sudah `PUBLISHED` dan tampil di portal |
| Publikasi Dicabut | Kelas Dibatalkan | Lifecycle `REVOKED` oleh KM; riwayat publikasi tetap ada |
| Sesi Dibatalkan | Publikasi Dicabut | Akademik tidak berlangsung (`SESSION_CANCELLED`), bukan pencabutan info |
| Perlu Review | Menunggu Persetujuan | Tugas sudah terbit oleh PJ (`NOT_REVIEWED`); review KM retrospektif, tidak menghambat terbit |
| Ruangan Kandidat | Ruangan Kosong | Hasil pencarian internal, wajib konfirmasi TU (`room_confirmations`) sebelum publish |
| Riwayat Perubahan | Log | Kronologi audit + versi, bukan sekadar log teks |
| Portal Kelas | Dashboard mahasiswa | Halaman hanya-baca `/c/{slug}` tanpa akun (mode `LINK` atau `CODE`) |
| Area Pengelola | Dashboard admin | Halaman `/app` untuk PJ/KM login berbasis role assignment |
| Penugasan Peran | Role, jabatan | `role_assignments`: user + role + scope + status `ACTIVE/SUSPENDED/REVOKED` + masa berlaku |
| Undangan | Invite link | `role_invitations` sekali pakai: `PENDING/ACCEPTED/EXPIRED/REVOKED`, scope tidak dapat diubah penerima |
| Sesi Pengurus | Login, token | `user_sessions`: token hash + `active_role_assignment_id` + salinan `session_version`, dapat dicabut server |
| Sesi Portal | Akses kelas | `portal_sessions`: token hash + `access_code_version`, tanpa hak ubah |
| Tugas (versi) | Task, PR, deskripsi saja | `tasks`: `title/instructions/deadline_at/submission_*` + `publication_status/review_state/version/completed_at/archived_at/deleted_at` |
| Materi | Link, file | `materials`: selalu punya `class_id`, opsional `course_offering_id/task_id` |
| Notifikasi | Broadcast, reminder | `notification_messages` + `notification_attempts`, idempoten via `idempotency_key`, status terpisah dari data akademik |
| Kanal WhatsApp | Grup WA, scope_jid | `whatsapp_channels`: alamat JID sebagai kanal, bukan pemilik data akademik |
| Sidebar | Menu kiri, navbar | Navigasi desktop tetap di kiri (248px), berisi logo + menu + identitas peran |
| Sidebar Ciut | Sidebar mini, icon bar | Mode icon-only 76px: label/judul seksi/teks identitas disembunyikan, ikon tetap + tooltip |
| Logo Asterisk (pemicu) | Logo, asterisk | Tombol logo di atas sidebar untuk menciutkan/melebarkan navigasi |
| State Galat Halaman | Error page, empty state | Tampilan penuh saat halaman gagal: `500` (server), `offline` (luring), `404` (tak ditemukan) via `pageState` |
