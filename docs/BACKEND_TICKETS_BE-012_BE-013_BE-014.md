# Backend Security Tickets: BE-012 sampai BE-014

Dokumen ini merinci pekerjaan security hardening untuk rate limiting, konfigurasi token dan cookie, serta kebijakan CORS dan HTTP security headers. Setiap ticket memuat baseline repository, keputusan desain, acceptance criteria, test matrix, Definition of Done, dan prompt implementasi.

## Ringkasan

| ID | Judul | Prioritas | Kondisi saat ini | Dependensi |
|---|---|---:|---|---|
| BE-012 | Rate Limiting Endpoint Sensitif | P0 | Login sudah persisten, portal limiter masih in-memory, endpoint sensitif lain belum memakai policy terpusat | BE-013 untuk keyed hashing dan trusted proxy policy |
| BE-013 | Finalisasi Konfigurasi Token dan Cookie Produksi | P1 | Secure cookie sudah terhubung, `AuthHashKey` belum dipakai server API | BE-012 memakai hash key yang sama |
| BE-014 | Security Headers dan CORS Production Policy | P1 | Same-host CORS dan tiga header dasar tersedia, allowlist production dan CSP belum tersedia | BE-013 untuk environment dan proxy awareness |

Urutan implementasi yang disarankan:

1. Tetapkan model environment, secret, dan trusted proxy pada BE-013.
2. Implementasikan rate limiter BE-012 dengan source identity dari BE-013.
3. Finalisasi CORS serta security headers melalui BE-014.

## Baseline repository

Kondisi berikut terverifikasi saat dokumen dibuat:

- Login membatasi lima kegagalan dalam 15 menit dan memblokir selama 15 menit.
- Percobaan login disimpan pada `login_attempts`.
- Handler login masih membuat source hash dari `r.RemoteAddr` lengkap. Port client sementara dapat menghasilkan hash berbeda.
- Beberapa insert `login_attempts` pada handler masih mengabaikan error.
- Package `internal/auth` memiliki service dengan keyed hash, tetapi server API belum memakai service tersebut.
- Portal CODE membatasi lima kegagalan dalam 15 menit, tetapi counter dan block disimpan dalam map proses.
- Restart proses atau beberapa instance server menghasilkan state rate limit portal yang berbeda.
- `BOT_JADWAL_AUTH_HASH_KEY` sudah dibaca ke `Config.AuthHashKey`, tetapi belum diteruskan ke `api.Server` atau auth service aktif.
- Cookie `bv1` sudah memakai `HttpOnly`, `SameSite=Lax`, dan `Secure` berdasarkan `BOT_JADWAL_SECURE_COOKIES`.
- CORS saat ini mengizinkan origin dengan hostname yang sama dan memantulkan origin tersebut bersama credentials.
- Middleware sudah mengirim `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, dan `Referrer-Policy: same-origin`.
- Content Security Policy, Permissions Policy, HSTS, dan allowlist origin production belum tersedia.
- Frontend memuat Tailwind, Alpine.js, dan Google Fonts dari CDN. CSP harus mempertimbangkan dependency tersebut tanpa memakai wildcard.

## Aturan umum implementasi

- Baca `AGENTS.md` sebelum mengubah kode.
- Jangan menyimpan password, token, kode portal, invitation token, recovery code, IP mentah, atau credential dalam log dan tabel percobaan.
- Jangan mempercayai `X-Forwarded-For` atau `Forwarded` kecuali request berasal dari proxy yang dikonfigurasi sebagai trusted.
- Gunakan response error generik pada endpoint yang dapat dipakai untuk enumerasi akun, kelas, atau token.
- Gunakan database sementara pada test.
- Jangan menyentuh `.env`, database, atau sesi WhatsApp pada `storage/`.
- Tambahkan regression test sebelum atau bersama implementasi.
- Jangan commit atau push kecuali diminta.
- Jalankan gate berikut sebelum menutup ticket:

```powershell
go test -count=1 ./...
go vet ./...
go build -o bin/bot ./cmd/bot
git diff --check
```

Referensi utama:

- [API v1](api/API_V1.md)
- [Access Control](product/ACCESS_CONTROL.md)
- [Business Rules](product/BUSINESS_RULES.md)
- [Functional Requirements](product/FUNCTIONAL_REQUIREMENTS.md)
- [Data Model](product/DATA_MODEL.md)
- [Deployment](DEPLOYMENT.md)
- [ADR Opaque Session](adr/0003-opaque-session-wa-identity.md)

---

## BE-012: Rate Limiting Endpoint Sensitif

### Status

Sebagian tersedia. Login sudah memakai penyimpanan database. Portal CODE sudah memiliki policy sesuai kontrak, tetapi state masih lokal pada proses. Invitation acceptance, recovery, backup/restore, serta admin mutation belum memakai rate limiter terpusat.

### Masalah

Rate limiting yang hanya hidup di memori dapat dilewati dengan restart atau distribusi request ke beberapa instance. Source identity yang memakai `RemoteAddr` lengkap juga dapat berubah karena port client berbeda pada setiap koneksi.

Endpoint sensitif tidak boleh memakai aturan yang tersebar karena hal berikut:

- limit dan response dapat berbeda antar handler;
- satu handler dapat menyimpan source mentah sedangkan handler lain memakai hash;
- trusted proxy header dapat dipalsukan jika server mempercayainya tanpa daftar proxy;
- counter dapat hilang ketika aplikasi restart;
- database error dapat diabaikan sehingga limiter gagal terbuka tanpa diketahui;
- response dapat membocorkan apakah identity, class slug, invitation, atau token tersedia.

### Tujuan

Menyediakan rate limiter terpusat, persisten, dan dapat diuji. Limiter harus memakai subject serta source yang dinormalisasi dan di-hash dengan key rahasia. Semua instance aplikasi harus melihat state yang sama.

### Endpoint dalam scope

| Endpoint atau operasi | Subject minimum | Policy |
|---|---|---|
| Login | Identity + source | Pertahankan 5 gagal per 15 menit, blokir 15 menit |
| Portal CODE exchange | Class + source | Pertahankan 5 gagal per 15 menit, blokir 15 menit |
| Invitation acceptance | Token fingerprint + source | Tentukan limit melalui ADR atau konfigurasi |
| Recovery publik jika tersedia | Identity atau token fingerprint + source | Tentukan limit melalui ADR atau konfigurasi |
| Backup dan restore | Actor + source | Batasi frekuensi operasi mahal dan berisiko |
| Suspend dan recover user | Actor + source | Batasi abuse tanpa mengubah authorization |
| Portal code rotation | Actor + class | Batasi rotasi berulang |

Jangan membuat route recovery baru sebagai bagian ticket ini. Terapkan policy hanya pada endpoint yang benar-benar tersedia. Jika endpoint belum tersedia, dokumentasikan policy yang harus dipakai ketika route tersebut dibuat.

### Rancangan data

Gunakan satu tabel generik baru atau abstraksi persisten yang setara. Contoh logical fields:

```text
id
policy_key
subject_hash
source_hash
outcome
attempted_at
blocked_until
created_at
```

Aturan data:

- `policy_key` memakai konstanta, misalnya `AUTH_LOGIN` dan `PORTAL_CODE_EXCHANGE`.
- `subject_hash` dan `source_hash` memakai HMAC-SHA256 dengan key dari BE-013.
- Nilai mentah tidak disimpan.
- Outcome memakai enum terbatas seperti `SUCCESS`, `FAILURE`, dan `BLOCKED`.
- Tambahkan index untuk policy, subject, source, serta waktu.
- Cleanup data lama mengikuti retention yang terdokumentasi.
- Migration database lama dan fresh schema menghasilkan bentuk yang sama.

`login_attempts` dapat dipertahankan untuk compatibility atau dimigrasikan ke service generik. Jangan membuat dua limiter aktif untuk request login yang sama.

### Source identity dan trusted proxy

Default source berasal dari alamat peer `RemoteAddr` setelah port dihapus dan alamat IP dinormalisasi.

Header proxy hanya boleh dipakai ketika peer langsung termasuk `TRUSTED_PROXY_CIDRS`:

- `Forwarded` atau `X-Forwarded-For` diproses dengan aturan yang terdokumentasi.
- Ambil hop client berdasarkan rantai proxy tepercaya, bukan selalu elemen pertama tanpa validasi.
- Header dari peer yang tidak tepercaya diabaikan.
- IPv4 dan IPv6 dinormalisasi sebelum hashing.
- Test harus membuktikan spoofed forwarding header tidak mengubah source identity.

### Policy dan response

- Policy login dan portal mengikuti angka yang sudah dibakukan dokumentasi.
- Threshold untuk endpoint lain harus berasal dari konfigurasi atau ADR. Jangan menyisipkan angka arbitrer langsung ke handler.
- Response rate limit memakai `429 TOO_MANY_REQUESTS`.
- Tambahkan header `Retry-After` tanpa mengungkap counter internal.
- Error login, portal, invitation, dan recovery tetap generik.
- Success dapat mereset window hanya jika business rule menyatakannya. Riwayat security event tidak dihapus.
- Authorization selalu diperiksa. Rate limiting tidak menggantikan role dan scope validation.

### Failure policy

Tentukan perilaku ketika limiter storage gagal:

- Login, portal code, invitation acceptance, dan recovery publik sebaiknya gagal tertutup dengan error layanan generik.
- Operasi admin yang membutuhkan database utama juga mengembalikan error jika limiter tidak dapat diperiksa.
- Catat error operasional tanpa subject atau source mentah.
- Jangan diam-diam melanjutkan request seolah limit belum tercapai.

### Acceptance criteria

- [ ] Login tetap memakai policy 5 kegagalan per 15 menit dan blokir 15 menit.
- [ ] Portal CODE memakai policy yang sama sesuai kontrak.
- [ ] Counter portal bertahan setelah service atau server dibuat ulang.
- [ ] Dua instance aplikasi membaca state limiter yang sama.
- [ ] Port client yang berbeda tidak mengubah source identity.
- [ ] IPv4 dan IPv6 dinormalisasi sebelum hashing.
- [ ] Forwarding header dari peer tidak tepercaya diabaikan.
- [ ] Forwarding header dari trusted proxy diproses sesuai policy.
- [ ] Database tidak menyimpan identity, source address, token, atau kode mentah.
- [ ] Request terblokir menghasilkan `429` dan `Retry-After`.
- [ ] Slug, identity, dan token tidak dapat dienumerasi dari perbedaan response.
- [ ] Invitation acceptance dan endpoint sensitif lain memakai policy registry terpusat.
- [ ] Limiter storage error tidak diabaikan.
- [ ] Concurrent requests tidak melampaui threshold akibat race.
- [ ] Cleanup atau retention security attempt terdokumentasi.

### Test matrix

| Skenario | Hasil yang diharapkan |
|---|---|
| Login gagal sampai threshold | Request berikutnya `429` |
| Portal code salah sampai threshold | Request berikutnya `429` |
| Server dibuat ulang | Block portal masih aktif |
| Dua service memakai database sama | Limit dihitung bersama |
| Source `IP:portA` dan `IP:portB` | Source hash sama |
| IPv6 dengan representasi setara | Source hash sama |
| Spoofed `X-Forwarded-For` dari peer biasa | Header diabaikan |
| Request melalui trusted proxy | Client source diturunkan dengan benar |
| Subject valid dan tidak valid | Pesan gagal tidak membedakan keberadaan data |
| Dua request pada batas threshold | Tidak keduanya lolos |
| Limiter query dipaksa gagal | Request tidak dilanjutkan diam-diam |
| Window berakhir | Request dapat dicoba kembali |
| Successful login | Riwayat tetap ada, window mengikuti business rule |

### Definition of Done

- Semua endpoint dalam scope memakai satu policy registry dan limiter service.
- Portal tidak lagi bergantung pada map in-memory sebagai sumber kebenaran.
- Source identity dan proxy trust memiliki implementasi tunggal.
- Tidak ada raw identity, IP, token, atau code pada storage dan log.
- Migration upgrade serta fresh database lulus.
- Unit test, integration test, concurrency test, vet, build, dan diff check lulus.

### Prompt implementasi

```text
Kerjakan ticket BE-012 Rate Limiting Endpoint Sensitif pada repository bot-wa-jadwal.

Baca AGENTS.md, docs/api/API_V1.md bagian auth dan portal, docs/product/ACCESS_CONTROL.md, docs/product/BUSINESS_RULES.md bagian akses, docs/product/FUNCTIONAL_REQUIREMENTS.md bagian FR-ACCESS, docs/product/DATA_MODEL.md bagian login_attempts dan portal_sessions, serta BE-013 jika sudah tersedia.

Baseline:
- Login sudah menyimpan login_attempts dan memiliki limit 5 gagal per 15 menit dengan blokir 15 menit.
- Handler login masih memakai hash dari RemoteAddr lengkap sehingga port client dapat membypass source bucket.
- Portal CODE memiliki policy yang sama tetapi menyimpan failure dan block di map in-memory.
- AuthHashKey sudah dibaca config tetapi belum dipakai server API.

Implementasikan rate limiter terpusat dan persisten:
1. Buat policy registry untuk login, portal code, invitation acceptance, recovery yang tersedia, backup/restore, admin suspend/recover, dan portal code rotation.
2. Pertahankan threshold login dan portal yang sudah terdokumentasi. Threshold endpoint lain harus berasal dari config atau ADR, bukan angka tersembunyi di handler.
3. Simpan event limiter tanpa raw identity, IP, code, atau token. Gunakan HMAC-SHA256 dengan AuthHashKey dari BE-013.
4. Normalisasi peer IP dengan menghapus port serta mengkanonisasi IPv4 dan IPv6.
5. Percayai Forwarded atau X-Forwarded-For hanya jika peer langsung berada pada TRUSTED_PROXY_CIDRS.
6. Ganti limiter portal in-memory dengan storage bersama yang bertahan setelah restart dan bekerja pada beberapa instance.
7. Gunakan operasi database atomik agar concurrent request tidak melewati threshold.
8. Response block memakai 429, error code TOO_MANY_REQUESTS, dan Retry-After.
9. Pesan gagal harus generik agar akun, class, invitation, dan recovery token tidak dapat dienumerasi.
10. Jangan mengabaikan error pencatatan atau pemeriksaan limiter.
11. Jangan membuat endpoint baru di luar scope.

Kerjakan secara test-first. Gunakan database SQLite sementara. Uji restart service, dua service dengan database sama, concurrent threshold, perubahan port client, IPv6, spoofed forwarding header, trusted proxy, window expiry, limiter storage failure, dan response enumeration.

Jalankan gofmt, go test -count=1 ./..., go vet ./..., go build -o bin/bot ./cmd/bot, dan git diff --check. Laporkan policy registry final, migration, proxy trust model, hasil test, serta risiko tersisa. Jangan commit atau push kecuali diminta.
```

---

## BE-013: Finalisasi Konfigurasi Token dan Cookie Produksi

### Status

Cookie auth sudah memakai `HttpOnly`, `SameSite=Lax`, dan flag `Secure` dari konfigurasi. `BOT_JADWAL_AUTH_HASH_KEY` sudah dibaca, tetapi belum diteruskan ke auth service aktif. Startup juga belum membedakan validasi development dan production secara tegas.

### Masalah

Konfigurasi security yang tersedia tetapi tidak digunakan memberi rasa aman palsu. Deployment production juga tidak boleh berjalan dengan secret kosong, cookie tidak aman, atau trusted proxy yang tidak didefinisikan dengan jelas.

Token mentah hanya boleh tersedia pada response yang memang membuat token. Token tidak boleh masuk log, audit, error, atau query parameter kecuali compatibility portal yang sudah dibakukan dan risikonya didokumentasikan.

### Tujuan

Membuat konfigurasi security eksplisit, tervalidasi saat startup, dan konsisten pada login, switch context, logout, portal session, invitation, recovery, serta rate limiting.

### Konfigurasi yang disarankan

| Environment variable | Fungsi | Production |
|---|---|---|
| `BOT_JADWAL_ENV` | `development`, `test`, atau `production` | Wajib bernilai `production` |
| `BOT_JADWAL_AUTH_HASH_KEY` | Key HMAC untuk identity dan source fingerprint | Wajib, minimal 32 byte entropy memadai |
| `BOT_JADWAL_SECURE_COOKIES` | Menambahkan flag Secure pada cookie auth | Wajib `true` |
| `BOT_JADWAL_ALLOWED_ORIGINS` | Allowlist origin untuk BE-014 | Wajib eksplisit |
| `BOT_JADWAL_TRUSTED_PROXY_CIDRS` | Proxy yang boleh memasok client forwarding header | Opsional, default kosong |
| `BOT_JADWAL_PUBLIC_BASE_URL` | Origin publik canonical untuk validasi deployment dan link | Wajib jika deployment membutuhkannya |

Nama final dapat disesuaikan, tetapi satu nama harus dipakai konsisten oleh kode dan dokumentasi.

### Aturan hash key

- Gunakan key untuk HMAC fingerprint identity, source, dan subject limiter.
- Jangan memakai key sebagai password encryption key atau session token.
- Session token acak dapat tetap dicari melalui hash kriptografis satu arah karena token memiliki entropy tinggi.
- Jangan mencetak key atau turunannya ke log.
- Perubahan key harus memiliki konsekuensi yang terdokumentasi. Fingerprint attempt lama tidak lagi cocok setelah rotasi.
- Production startup gagal jika key kosong atau terlalu pendek.
- Development boleh memakai key eksplisit dari environment. Jangan membuat default hard-coded yang dapat terbawa ke production.

### Aturan token

- Gunakan CSPRNG untuk token session, invitation, portal, dan recovery.
- Simpan hanya hash token.
- Kembalikan raw token hanya sekali pada response pembuatannya.
- Gunakan perbandingan constant-time untuk secret yang dibandingkan di aplikasi.
- Rotasi context menghasilkan token baru dan mencabut token lama.
- Suspend atau session version change membatalkan token lama.
- Jangan mencatat Authorization header, cookie, request body secret, atau raw token pada logger.
- Query parameter `portal_token` harus dianggap compatibility path. Hindari memasukkannya ke access log dan arahkan frontend memakai `X-Portal-Token`.

### Aturan cookie

- `HttpOnly=true`.
- `Secure=true` pada production.
- `SameSite=Lax` dipertahankan selama frontend dan API berada pada site yang sama.
- Jika arsitektur memerlukan cross-site cookie, buat threat model dan CSRF protection sebelum memakai `SameSite=None`.
- Path cookie harus sesempit yang masih mendukung seluruh route auth.
- Login, switch context, dan logout memakai atribut cookie yang konsisten.
- Logout menghapus cookie dengan atribut path, secure, dan SameSite yang cocok.
- Jangan memakai Domain cookie kecuali deployment multi-subdomain memang membutuhkannya.

### Startup validation

Tambahkan validasi sebelum server menerima traffic:

- Environment production dikenali secara eksplisit.
- Hash key valid.
- Secure cookie aktif.
- Public base URL memakai HTTPS.
- Allowed origins valid dan tidak memakai wildcard bersama credentials.
- Trusted proxy CIDR dapat diparse.
- Storage directory tersedia.
- Error menyebut nama konfigurasi yang salah tanpa mencetak secret.

### Acceptance criteria

- [ ] `AuthHashKey` benar-benar digunakan oleh auth dan limiter service.
- [ ] Production startup gagal jika hash key kosong atau tidak valid.
- [ ] Production startup gagal jika secure cookie nonaktif.
- [ ] Production startup gagal jika origin atau proxy CIDR malformed.
- [ ] Development dan test dapat memakai konfigurasi eksplisit yang aman untuk environment tersebut.
- [ ] Login, switch context, dan logout memakai atribut cookie konsisten.
- [ ] Raw session, portal, invitation, dan recovery token tidak masuk database, log, atau audit.
- [ ] Source dan identity fingerprint memakai HMAC, bukan plain hash nilai berentropi rendah.
- [ ] Context switch mencabut token lama.
- [ ] Suspend dan session version change membatalkan session lama.
- [ ] Portal token query tidak dicatat oleh application logger.
- [ ] Dokumentasi environment menjelaskan pembuatan secret tanpa menaruh nilainya pada repository.
- [ ] `.env`, secret, database, session WA, dan binary tetap tidak terlacak Git.

### Test matrix

| Skenario | Hasil yang diharapkan |
|---|---|
| Production tanpa hash key | Startup gagal |
| Production dengan key pendek | Startup gagal |
| Production secure cookie false | Startup gagal |
| Development dengan config eksplisit | Startup berhasil |
| Allowed origin malformed | Startup gagal |
| Trusted proxy CIDR malformed | Startup gagal |
| Login sukses | Cookie HttpOnly, Secure sesuai environment, SameSite benar |
| Switch context | Token dan cookie lama tidak berlaku |
| Logout | Cookie dihapus dengan atribut cocok |
| Database diperiksa | Hanya hash token tersimpan |
| Log diperiksa | Tidak memuat token, cookie, password, atau code |
| Hash subject sama | Fingerprint deterministik dengan key sama |
| Hash subject dengan key berbeda | Fingerprint berbeda |

### Definition of Done

- Tidak ada field konfigurasi security yang dibaca tetapi tidak digunakan.
- Startup validation berjalan sebelum server dan worker menerima traffic.
- Auth handler memakai service atau helper hashing kanonis.
- Token dan cookie tests mencakup login, switch, logout, revoke, dan expiry.
- Deployment documentation memuat seluruh variable dan failure mode.
- Test, vet, build, dan diff check lulus.

### Prompt implementasi

```text
Kerjakan ticket BE-013 Finalisasi Konfigurasi Token dan Cookie Produksi pada repository bot-wa-jadwal.

Baca AGENTS.md, docs/adr/0003-opaque-session-wa-identity.md, docs/product/ACCESS_CONTROL.md, docs/product/BUSINESS_RULES.md bagian session, docs/product/DATA_MODEL.md bagian user_sessions dan portal_sessions, README.md, serta docs/DEPLOYMENT.md.

Baseline:
- BOT_JADWAL_AUTH_HASH_KEY sudah dibaca ke Config.AuthHashKey tetapi belum dipakai server API.
- Cookie bv1 sudah HttpOnly, SameSite=Lax, dan Secure berdasarkan BOT_JADWAL_SECURE_COOKIES.
- Package internal/auth memiliki service yang mensyaratkan hash key minimal 32 byte, tetapi handler aktif masih memiliki hashing dan rate-limit sendiri.

Selesaikan hal berikut:
1. Tambahkan mode environment yang eksplisit dan startup validation sebelum server menerima traffic.
2. Production wajib memiliki AuthHashKey valid, SecureCookies=true, HTTPS public base URL, serta allowed origins eksplisit.
3. Teruskan AuthHashKey ke auth dan rate-limit service. Gunakan HMAC-SHA256 untuk fingerprint identity dan source.
4. Jangan memakai hash key untuk mengenkripsi password atau sebagai session token.
5. Pastikan semua token dibuat dengan CSPRNG, database hanya menyimpan hash, dan raw token hanya dikembalikan sekali.
6. Satukan atribut cookie login, switch context, dan logout.
7. Jangan mengubah SameSite menjadi None tanpa threat model dan CSRF protection.
8. Jangan log Authorization, Cookie, password, portal code, invitation token, recovery token, atau query portal_token.
9. Validasi BOT_JADWAL_ALLOWED_ORIGINS dan BOT_JADWAL_TRUSTED_PROXY_CIDRS.
10. Dokumentasikan rotasi hash key dan dampaknya terhadap fingerprint lama.
11. Perbarui README serta DEPLOYMENT tanpa menambahkan secret nyata atau .env ke Git.

Kerjakan secara test-first. Tambahkan config tests untuk development, test, dan production; cookie tests untuk login, switch, logout; database test untuk hashed token; logger redaction test; serta HMAC determinism test.

Jalankan gofmt, go test -count=1 ./..., go vet ./..., go build -o bin/bot ./cmd/bot, dan git diff --check. Laporkan variable final, startup failure cases, jalur token yang diaudit, dan risiko tersisa. Jangan commit atau push kecuali diminta.
```

---

## BE-014: Security Headers dan CORS Production Policy

### Status

Middleware saat ini mendukung preflight, memantulkan origin dengan hostname sama, mengizinkan credentials, serta mengirim tiga header dasar. Policy tersebut membantu development lintas port, tetapi belum cukup eksplisit untuk production.

### Masalah

Pemeriksaan hostname saja tidak membatasi scheme dan port. Origin lain pada host yang sama dapat menerima CORS credentials. Deployment melalui reverse proxy juga dapat membuat host comparison sulit diprediksi.

Security headers belum memiliki Content Security Policy. Frontend saat ini memakai dependency CDN dan inline behavior, sehingga CSP harus dirancang berdasarkan resource nyata. Policy tidak boleh memakai wildcard luas hanya agar halaman tetap berjalan.

### Tujuan

Membuat kebijakan origin dan header yang berbeda secara eksplisit antara development dan production. Browser hanya menerima credential response untuk origin terpercaya, sementara frontend statis tetap dapat memuat resource yang memang digunakan.

### CORS policy

Production:

- Gunakan exact-origin allowlist dari konfigurasi.
- Origin terdiri dari scheme, hostname, dan port.
- Jangan memakai `*` bersama `Access-Control-Allow-Credentials: true`.
- Origin yang tidak ada pada allowlist tidak menerima header allow-origin.
- Preflight memvalidasi requested method dan requested headers.
- Hanya method dan header yang benar-benar digunakan API yang diizinkan.
- Tambahkan `Vary: Origin` tanpa menimpa nilai `Vary` lain.
- Gunakan `Access-Control-Max-Age` hanya jika policy stabil.
- Request tanpa Origin, misalnya bot atau same-origin non-browser, tetap diproses sesuai auth normal.

Development:

- Boleh mengizinkan daftar origin localhost eksplisit pada port frontend yang dikonfigurasi.
- Jangan menganggap semua port pada hostname yang sama otomatis tepercaya.
- Policy development tidak boleh aktif saat environment production.

### Cookie authentication dan CSRF

CORS bukan perlindungan CSRF. Jika mutation menerima cookie auth dari lebih dari satu origin:

- validasi Origin untuk request mutation;
- pertahankan SameSite yang sesuai;
- tambahkan CSRF token jika arsitektur menjadi cross-site;
- Bearer-only client tidak boleh dipaksa memakai CSRF token tanpa alasan.

Keputusan final harus dicatat dalam threat model atau ADR singkat.

### Security headers minimum

| Header | Policy minimum |
|---|---|
| `X-Content-Type-Options` | `nosniff` |
| `Referrer-Policy` | `same-origin` atau policy yang lebih ketat dan kompatibel |
| `X-Frame-Options` | `DENY` untuk compatibility |
| `Content-Security-Policy` | `frame-ancestors 'none'` dan source list eksplisit |
| `Permissions-Policy` | Nonaktifkan capability browser yang tidak digunakan |
| `Strict-Transport-Security` | Hanya production HTTPS setelah deployment dipastikan benar |
| `Cache-Control` | `no-store` pada response yang membawa token atau data auth sensitif |

### Content Security Policy

Inventaris resource frontend saat ini mencakup:

- script dari `cdn.tailwindcss.com`;
- Alpine.js dari `cdn.jsdelivr.net`;
- font stylesheet dari `fonts.googleapis.com`;
- font files dari `fonts.gstatic.com`;
- koneksi API pada origin aplikasi;
- link keluar ke WhatsApp yang tidak perlu masuk `connect-src` jika hanya navigasi biasa.

Implementasi CSP harus:

- memakai source list spesifik, bukan `*`;
- menetapkan `default-src 'self'`;
- menetapkan `object-src 'none'`;
- menetapkan `base-uri 'self'`;
- menetapkan `frame-ancestors 'none'`;
- membatasi `connect-src` ke API yang diperlukan;
- mencatat setiap kebutuhan `unsafe-inline` atau `unsafe-eval` sebagai risiko nyata;
- memakai nonce atau hash untuk inline script yang dapat dikendalikan;
- memulai dengan `Content-Security-Policy-Report-Only` jika enforcement langsung akan merusak frontend;
- memiliki test dan smoke test browser sebelum dipindahkan ke enforcement.

Repository mengharuskan frontend tetap plain HTML dengan dependency CDN. Jangan menambahkan npm atau frontend build step sebagai bagian ticket ini.

### Logging dan observability

- Log penolakan origin tanpa cookie, Authorization, token, atau query secret.
- Bedakan malformed origin dari origin yang tidak diizinkan.
- Jangan memantulkan origin asing pada response error.
- Sediakan metric atau counter agregat jika infrastructure sudah mendukungnya. Jangan menambah dependency observability baru hanya untuk ticket ini.

### Acceptance criteria

- [ ] Production memakai exact-origin allowlist.
- [ ] Scheme, hostname, dan port semuanya dibandingkan.
- [ ] Origin asing tidak menerima `Access-Control-Allow-Origin` atau credentials.
- [ ] Wildcard tidak digunakan bersama credentials.
- [ ] Preflight method dan headers yang sah menghasilkan `204` dengan header benar.
- [ ] Preflight method atau header asing ditolak secara konsisten.
- [ ] `Vary: Origin` digabung dengan nilai existing.
- [ ] Development origin hanya berasal dari konfigurasi eksplisit.
- [ ] Cookie-authenticated mutation memiliki keputusan CSRF yang terdokumentasi.
- [ ] Header `nosniff`, referrer, frame, permissions, dan CSP tersedia.
- [ ] HSTS hanya dikirim pada production HTTPS.
- [ ] Response login, switch context, portal session, dan invitation token memakai `Cache-Control: no-store`.
- [ ] CSP memuat source CDN yang benar-benar digunakan tanpa wildcard.
- [ ] CSP diuji terhadap halaman KM, Superadmin, dan app tanpa console violation yang tidak diterima.
- [ ] Tidak ada token atau credential pada log penolakan CORS.

### Test matrix

| Skenario | Hasil yang diharapkan |
|---|---|
| Origin production valid | Header allow-origin exact dan credentials tersedia |
| Host sama, port tidak terdaftar | Tidak diizinkan |
| Host sama, scheme berbeda | Tidak diizinkan |
| Origin asing | Tidak mendapat CORS credential headers |
| Request tanpa Origin | Diproses normal |
| Preflight method valid | `204` dengan allow method yang tepat |
| Preflight method asing | Ditolak atau tanpa izin sesuai policy |
| Preflight header valid | Header diizinkan |
| Preflight header asing | Ditolak |
| Existing `Vary` | Nilai tetap ada dan Origin ditambahkan |
| Production HTTPS | HSTS tersedia |
| Development HTTP | HSTS tidak dikirim |
| Login atau portal session response | `Cache-Control: no-store` |
| Halaman frontend dengan CSP report-only | Hanya violation yang sudah dianalisis |
| CSP enforcement | Halaman dan aksi utama tetap bekerja |

### Definition of Done

- CORS tidak lagi mengambil keputusan hanya dari hostname request.
- Configuration parser menolak origin malformed.
- Development dan production policy memiliki test terpisah.
- Security headers diterapkan konsisten pada HTML, asset, dan API sesuai kebutuhan.
- CSP rollout memiliki hasil report-only dan smoke test browser.
- Dokumentasi deployment menjelaskan origin, proxy, HTTPS, HSTS, CSP, dan CSRF.
- Test, vet, build, browser smoke test, dan diff check lulus.

### Prompt implementasi

```text
Kerjakan ticket BE-014 Security Headers dan CORS Production Policy pada repository bot-wa-jadwal.

Baca AGENTS.md, docs/product/ACCESS_CONTROL.md, docs/api/API_V1.md, docs/DEPLOYMENT.md, BE-013, serta seluruh web/index.html, web/superadmin.html, web/app.html, CSS, dan JavaScript yang memuat resource eksternal.

Baseline:
- Middleware CORS saat ini membandingkan hostname origin dengan request host dan dapat mengizinkan port atau scheme lain pada host yang sama.
- Credentials diaktifkan untuk origin yang lolos pemeriksaan tersebut.
- X-Content-Type-Options, X-Frame-Options, dan Referrer-Policy sudah tersedia.
- CSP, Permissions-Policy, HSTS, dan production origin allowlist belum tersedia.
- Frontend memakai CDN-hosted Tailwind, Alpine.js, dan Google Fonts. Jangan menambahkan npm atau build step.

Selesaikan hal berikut:
1. Tambahkan exact-origin allowlist dari konfigurasi BE-013. Bandingkan scheme, hostname, dan port.
2. Pisahkan policy development dan production. Production tidak boleh memakai wildcard bersama credentials atau implicit same-host-any-port.
3. Validasi preflight requested method dan headers.
4. Gabungkan Vary: Origin tanpa menimpa header Vary lain.
5. Dokumentasikan dan implementasikan perlindungan CSRF untuk cookie-authenticated mutation sesuai arsitektur origin final.
6. Pertahankan nosniff, frame deny, dan referrer policy.
7. Tambahkan Permissions-Policy dan CSP dengan source list eksplisit.
8. Inventaris resource CDN nyata. Jangan memakai wildcard agar CSP cepat lulus.
9. Gunakan nonce atau hash untuk inline script yang dapat dikendalikan. Catat kebutuhan unsafe-inline atau unsafe-eval yang tidak dapat dihilangkan karena CDN runtime.
10. Mulai CSP dalam report-only jika enforcement langsung merusak frontend, lalu pindahkan ke enforcement setelah smoke test.
11. Kirim HSTS hanya pada production HTTPS dengan proxy trust yang benar.
12. Tambahkan Cache-Control: no-store pada response yang membawa token atau data auth sensitif.
13. Jangan log Authorization, Cookie, portal_token, atau secret ketika origin ditolak.

Kerjakan secara test-first. Tambahkan table-driven tests untuk origin valid, host sama dengan port berbeda, scheme berbeda, origin asing, no-Origin, preflight method/header valid dan invalid, Vary merge, HSTS development/production, no-store, dan header security. Jalankan browser smoke test pada halaman KM, Superadmin, serta app dan periksa console CSP.

Jalankan gofmt, go test -count=1 ./..., go vet ./..., go build -o bin/bot ./cmd/bot, dan git diff --check. Laporkan allowlist final, CSP final dan setiap relaxation, keputusan CSRF, hasil browser test, serta risiko tersisa. Jangan commit atau push kecuali diminta.
```

---

## Gate penyelesaian tiga ticket

BE-012 sampai BE-014 dinyatakan selesai jika:

- [ ] Rate limit login dan portal persisten serta aman pada beberapa instance.
- [ ] Endpoint sensitif memakai policy registry yang terdokumentasi.
- [ ] Source identity tidak dapat dibypass dengan perubahan port atau spoofed forwarding header.
- [ ] Production tidak dapat start dengan hash key kosong atau cookie non-Secure.
- [ ] Raw token, password, code, identity, dan source address tidak masuk log atau database security event.
- [ ] CORS production memakai exact-origin allowlist.
- [ ] Cookie credentials hanya tersedia untuk origin terpercaya.
- [ ] CSRF policy untuk cookie-authenticated mutation terdokumentasi.
- [ ] CSP dan security headers lulus browser smoke test.
- [ ] Login dan response pembuat token memakai `Cache-Control: no-store`.
- [ ] Dokumentasi deployment dan environment sesuai implementasi.
- [ ] `go test -count=1 ./...` lulus.
- [ ] `go vet ./...` lulus.
- [ ] `go build -o bin/bot ./cmd/bot` lulus.
- [ ] `git diff --check` lulus.
- [ ] Security review menyetujui proxy trust, secret handling, CORS, CSP, dan CSRF policy.
