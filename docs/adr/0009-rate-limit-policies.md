# ADR-0009 — Kebijakan Rate Limit Terpusat

- Status: Accepted.
- Tanggal: 2026-09-29
- Konteks: Limit tersebar (login di handler, portal di map in-memory) dengan angka dan subject berbeda. Source dari `RemoteAddr` lengkap dapat berubah karena port client. Header proxy dipercaya tanpa daftar proxy.
- Keputusan:
  - Satu registry (`internal/ratelimit`, `DefaultPolicies`) dan satu tabel persisten `security_attempts` untuk seluruh endpoint sensitif. `login_attempts` dipertahankan untuk kompatibilitas audit login; request login hanya memakai satu limiter aktif (handler memakai `login_attempts` dengan semantik yang sama).
  - Threshold: `AUTH_LOGIN` 5 gagal/15 mnt blokir 15 mnt; `PORTAL_CODE_EXCHANGE` 5/15 mnt blokir 15 mnt; `INVITE_ACCEPT` 10/15 mnt blokir 30 mnt; `BACKUP_RESTORE` 10/60 mnt blokir 30 mnt; `ADMIN_MUTATION` 10/15 mnt blokir 15 mnt; `PORTAL_CODE_ROTATE` 5/15 mnt blokir 15 mnt.
  - Jendela dihitung sejak keberhasilan terakhir (success mereset window); riwayat tidak dihapus. Untuk operasi tanpa konsep gagal (backup, rotasi, mutasi admin), setiap keberhasilan menutup jendela sehingga frekuensi operasi mahal tetap dibatasi.
  - Subject + source hanya sebagai HMAC-SHA256 keyed (`BOT_JADWAL_AUTH_HASH_KEY`); tanpa key valid dipakai kunci efemeral (tidak stabil lintas restart; production wajib key valid).
  - Source = IP peer tanpa port (IPv4/IPv6 dikanonisasi). Header `Forwarded`/`X-Forwarded-For` hanya dipakai bila peer langsung ada pada `BOT_JADWAL_TRUSTED_PROXY_CIDRS`; jika tidak, header diabaikan.
  - Pemeriksaan memakai transaksi serial agar concurrent request tidak melewati threshold. Response blokir `429 TOO_MANY_REQUESTS` + `Retry-After`; pesan gagal generik (anti-enumerasi).
  - Storage error → gagal tertutup (`503 SERVICE_UNAVAILABLE` generik), tidak pernah dilanjutkan diam-diam. Error operasional dicatat tanpa subject/source mentah.
  - Retensi 30 hari via `Purge`; cleanup terjadwal di luar aplikasi.
- Konsekuensi: dua instance berbagi state lewat database. Rotasi hash key menginvalidasi fingerprint lama (limit mengendur sementara, bukan mengencang). Test memakai database sementara dan policy window pendek via registry injeksi.
