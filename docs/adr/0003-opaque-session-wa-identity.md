# ADR-0003 — Opaque Session Token, Identitas Nomor WA, bcrypt + TTL FR

- Status: Accepted
- Tanggal: 2026-09-28
- Konteks: Alternatif: JWT stateless vs opaque token DB. `DATA_MODEL §4` sudah mendesain `users/role_assignments/user_sessions/portal_sessions/login_attempts/recovery_tokens` dengan `session_version`, `active_role_assignment_id`, rotasi token saat ganti konteks. JWT menyulitkan pencabutan segera + konteks multi-role.
- Keputusan: Opaque token acak 32 byte (hash SHA-256 di DB, token mentah hanya saat login). `identity_key` = nomor WA ternormalisasi (selaras bot + recovery WA `FR-ACCESS-006`). Password hanya `password_hash` bcrypt. Kirim via `Authorization: Bearer` + opsi cookie httpOnly SameSite=Lax untuk Alpine. TTL: PJ/KM 2 jam idle / 24 jam absolut; System Admin 30 mnt idle / 8 jam absolut. Rate-limit login 5 gagal/15 mnt per identitas+sumber → blokir 15 mnt + catat `login_attempts`. Portal default `LINK`; mode `CODE` memakai `portal_sessions` + `access_code_version`.
- Konsekuensi: Setiap mutasi wajib cek sesi + assignment aktif + scope di backend (bukan di UI). Perlu tabel sesi + middleware auth + job sapu sesi kedaluwarsa.
