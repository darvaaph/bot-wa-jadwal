# ADR-0010 — Kebijakan CORS, CSRF, dan HTTP Security Headers

- Status: Accepted
- Tanggal: 2026-09-29
- Konteks:
  - Sebelumnya, middleware CORS hanya membandingkan hostname origin dengan request host sehingga port dan scheme yang berbeda dapat menerima credential headers.
  - Server belum memiliki Content Security Policy (CSP), Permissions-Policy, dan HSTS untuk mode production.
  - Kredensial cookie (`bv1`) yang dipakai dashboard web rentan terhadap serangan CSRF jika mutasi lintas origin diizinkan tanpa validasi.
  - Frontend dibangun tanpa npm/build step, menggunakan vanilla HTML dengan CDN-hosted Tailwind, Alpine.js, dan Google Fonts.
- Keputusan:
  - **CORS Production Allowlist Eksplisit**: Production HANYA mengizinkan origin yang terdaftar di `BOT_JADWAL_ALLOWED_ORIGINS` atau `BOT_JADWAL_PUBLIC_BASE_URL`. Perbandingan mencakup scheme, hostname, dan port secara tepat (exact match). Tidak menggunakan wildcard `*` bersama credentials. Origin yang tidak terdaftar tidak menerima header CORS sama sekali.
  - **CORS Development Policy**: Mode development dan test mengizinkan origin localhost (`localhost`, `127.0.0.1`, `::1`) pada skema http/https.
  - **Preflight Validation**: Header `Access-Control-Request-Method` dibatasi hanya pada method yang didukung (`GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD`). Header `Access-Control-Request-Headers` divalidasi terhadap allowlist. Preflight yang tidak sah ditolak dengan status error yang sesuai.
  - **Vary Header**: `Origin` digabungkan ke header `Vary` tanpa menimpa header existing.
  - **Proteksi CSRF untuk Cookie-Authenticated Mutations**: Request mutasi (`POST`, `PUT`, `PATCH`, `DELETE`) yang membawa cookie `bv1` tanpa header `Authorization` (Bearer) divalidasi origin/referer-nya. Jika origin/referer tidak cocok dengan allowlist (atau tidak ada di production), request ditolak dengan `403 Forbidden`. Request dengan header Bearer token (API client, bot WhatsApp) dilewati tanpa CSRF check karena browser tidak mengirim Bearer token otomatis secara cross-origin.
  - **HTTP Security Headers**:
    - `X-Content-Type-Options: nosniff`
    - `X-Frame-Options: DENY`
    - `Referrer-Policy: same-origin`
    - `Permissions-Policy: camera=(), microphone=(), geolocation=(), payment=()`
    - `Content-Security-Policy`:
      `default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.tailwindcss.com https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'self';`
      *(Kebutuhan `'unsafe-inline'` dan `'unsafe-eval'` didokumentasikan karena sifat Alpine.js v3 dynamic evaluation serta injeksi stylesheet dinamis Tailwind CDN tanpa build tool).*
    - `Strict-Transport-Security: max-age=31536000; includeSubDomains` hanya dikirim ketika berjalan di mode production di atas HTTPS (koneksi TLS langsung atau via proxy terpercaya dengan header `X-Forwarded-Proto: https`).
  - **No-Store Caching**: Seluruh endpoint autentikasi dan pembuatan token (`/api/v1/auth/login`, `/api/v1/auth/switch-context`, `/api/v1/portal/:slug/session`, `/api/v1/invitations`, `/api/v1/invitations/accept`) selalu mengirim header `Cache-Control: no-store`.
  - **Privacy Log**: Penolakan origin tidak pernah mencatat cookie, header Authorization, password, atau token dalam log server.
- Konsekuensi:
  - Deployment production wajib mengonfigurasi `BOT_JADWAL_ALLOWED_ORIGINS` dengan tepat jika web dashboard diakses dari domain terpisah.
  - Mutasi via cookie aman dari CSRF browser tanpa memerlukan token CSRF terpisah untuk client API murni.
