# Audit source code dan alur aplikasi

Tanggal: 2 November 2026. Baseline: `f8883afcfdceb2d58a391d6bac25c14c54549a91` beserta perubahan staged dan unstaged yang sudah ada saat audit.

## Ringkasan

Repositori ini menjalankan bot WhatsApp, portal mahasiswa, dan dashboard PJ/KM/System Admin dalam satu aplikasi Go dengan SQLite. Sistem mengelola kelas, semester, jadwal, tugas, materi, akses pengguna, notifikasi, serta backup dan restore akademik.

Asumsi beban: satu proses aplikasi, beberapa kelas, beberapa pengelola bekerja bersamaan, dan pembacaan portal oleh mahasiswa. Temuan utama tidak membutuhkan skala besar atau beberapa server untuk terjadi.

**Verdict: belum layak dinyatakan aman untuk produksi.** Pengambilalihan akun melalui undangan dan pemberian akses lintas kelas sudah berhasil direproduksi pada database uji. Perbaiki keduanya sebelum optimasi performa atau refactor besar.

Audit menghasilkan 20 temuan prioritas. `P0` berarti perlu penanganan segera; `P1` berarti risiko keamanan/data atau hasil bisnis salah yang harus diperbaiki; `P2` berarti cacat fungsional atau pemeliharaan yang perlu dijadwalkan. Ini bukan penilaian estetika kode.

## Metode dan batas cakupan

- Menggunakan `using-superpowers`, `ponytail-audit`, dan prinsip `ponytail`. Core antislop dibaca; pekerjaan ini tidak merancang atau mengubah tampilan.
- Menginventarisasi source aplikasi, entry point, routes, skema/migrasi, frontend, dan tests; menelusuri jalur berisiko dari route sampai query/transaksi dan pemakainya. Inventaris utama memuat 102 berkas Go non-test yang terlihat melalui pencarian normal dan 10 JavaScript aplikasi; source seed yang terpengaruh aturan ignore juga diperiksa terpisah.
- Membandingkan implementasi dengan `docs/product/BUSINESS_RULES.md`, `ACCESS_CONTROL.md`, serta pedoman operasional. Dokumentasi diperlakukan sebagai kontrak yang perlu dicocokkan dengan kode, bukan bukti bahwa fitur sudah terlaksana.
- Membaca secara mendalam alur autentikasi, undangan, scope, lifecycle tugas, publikasi/partisipasi jadwal, outbox, worker, portal, service worker, draf ujian, serta restore. Bagian lain diperiksa secara selektif melalui struktur, query, pemanggil, dan tests. **Ini audit lintas repositori berbasis risiko, bukan klaim bahwa setiap baris source dan setiap interaksi UI sudah diverifikasi.**
- Tidak membaca `docs/archive/user-interviews/`, rahasia lingkungan, atau database produksi. Tidak mengirim pesan WhatsApp, menjalankan seed pada data asli, atau melakukan restore nyata.
- Temuan berlabel **reproduksi** dibuktikan dengan request HTTP in-process/database sementara atau mock JavaScript. Temuan **statis** memiliki jalur kode dan skenario konkret tetapi belum diuji end-to-end di lingkungan produksi.

## Peta alur dan penilaian

| Alur | Jalur implementasi utama | Penilaian |
|---|---|---|
| Startup | `cmd/bot/main.go` → config → pool DB/migrasi → manager legacy/v1 → worker → HTTP | Konfigurasi produksi divalidasi; wiring legacy dan v1 masih menambah kompleksitas |
| Login dan sesi | routes → `AuthController` → `user_sessions` → `AuthManager` | Berbeda dari implementasi domain `auth.Service`; ada bypass pembatas dan perpanjangan sesi |
| Undangan | KM/Admin → invitation → public accept → user/assignment/session | Dua cacat otorisasi terkonfirmasi |
| Portal | slug → mode LINK/CODE → sesi portal → semester → read model | Gate v1 ada; endpoint legacy dan cache browser mengurangi perlindungannya |
| Tugas | controller → validasi/scope → transaksi task/review/audit → outbox | Archive guard, invariant review, dan penyelarasan notifikasi belum konsisten |
| Jadwal | pattern/event → preview konflik → publish/revoke → portal/bot | Model ada, tetapi resolver portal dan bot berbeda; partisipasi melewati sebagian aturan |
| Materi | controller → scope → query tulis langsung | Scope kelas/offering diperiksa; riwayat perubahan tidak ditulis pada jalur HTTP |
| Notifikasi | producer → `notification_messages` → claim worker → WhatsApp → status/attempt | Transaksi producer belum atomik; pesan usang dan pengiriman ganda masih mungkin |
| Backup/restore | snapshot → checksum/preview → maintenance gate → pre-restore backup → transaksi restore | Desain lebih kuat; ada pesan koreksi yang dapat tersangkut tanpa kanal |
| Frontend | API helper → state per role → partial HTML → service worker | Ada risiko cache privat, kehilangan draf, dan error menjadi daftar kosong |

## Must fix

### 1. [P0] Undangan PJ dapat mengambil alih akun System Admin

**Bukti:** reproduksi HTTP. Lokasi: [auth_controller.go:1406](F:/Project/bot-wa-jadwal/internal/api/v1/auth_controller.go:1406), pembuatan undangan di [baris 1146](F:/Project/bot-wa-jadwal/internal/api/v1/auth_controller.go:1146).

**Alur dan masalah:** KM mengundang nomor akun Admin yang sudah ada menjadi PJ pada offering kelasnya. KM menerima token undangan dari respons dan dapat menggunakannya sendiri. `AcceptInvitation` melakukan upsert yang mengganti `password_hash`, `display_name`, dan status akun yang sudah ada tanpa membuktikan kepemilikan akun. Login dengan sandi baru lalu switch ke assignment Admin berhasil. `session_version` juga tetap 1 dalam reproduksi.

**Perbaikan minimum:** pisahkan aktivasi akun baru dari penambahan role akun lama. Akun lama harus login sebagai identitas penerima atau menjalani verifikasi kepemilikan independen; undangan tidak boleh mengganti sandi/status akun lama. Reset sandi harus melalui alur pemulihan yang mencabut sesi.

**Jika dibiarkan:** pemilik akses KM dapat memperoleh akses global. **Tes wajib:** undangan ke akun existing tidak mengubah sandi, status, atau sesi; penerimaan oleh identitas berbeda ditolak.

### 2. [P1] KM dapat membuat PJ untuk offering kelas lain

**Bukti:** reproduksi HTTP. Lokasi: [auth_controller.go:1199](F:/Project/bot-wa-jadwal/internal/api/v1/auth_controller.go:1199), [middleware/auth.go:102](F:/Project/bot-wa-jadwal/internal/api/middleware/auth.go:102).

**Alur dan masalah:** payload memakai `class_slug` milik KM tetapi `semester_id` dan `offering_id` milik kelas lain. Handler hanya mewajibkan kedua ID terisi, tanpa memverifikasi rantai offering → semester → kelas. Foreign key masing-masing kolom tidak memeriksa hubungan antarkolom. Assignment diterima middleware, lalu PJ hasil undangan berhasil membuat tugas di offering kelas lain.

**Perbaikan minimum:** validasi hubungan ketiga ID sebelum membuat undangan, ulangi saat penerimaan, dan validasi scope assignment saat autentikasi. Gunakan pemeriksaan yang sama untuk semua penulisan assignment.

**Jika dibiarkan:** isolasi antarkelas dapat ditembus tanpa peran Admin. **Tes wajib:** kombinasi class/semester/offering silang ditolak pada create, accept, dan penggunaan sesi.

### 3. [P1] Service worker menyimpan respons API privat tanpa pemisahan pengguna

**Bukti:** reproduksi mock JavaScript membuktikan `/api/v1/auth/me` tetap masuk cache ketika respons memiliki `Cache-Control: no-store`. Lokasi: [sw.js:95](F:/Project/bot-wa-jadwal/web/sw.js:95), [api.js:1566](F:/Project/bot-wa-jadwal/web/js/api.js:1566).

**Alur dan masalah:** semua GET `/api/` yang sukses dimasukkan ke cache bersama dan dipakai ketika jaringan gagal. Tidak ada pengecualian autentikasi, pemeriksaan `no-store`, atau pembersihan cache saat logout. Header tersebut tidak dipakai oleh kode sebagai syarat sebelum `cache.put`.

**Perbaikan minimum:** keluarkan API pengelola dan data portal terlindungi dari cache offline; bersihkan cache API lama saat aktivasi versi baru. Jika offline data diperlukan, rancang scope identitas dan pencabutannya secara eksplisit.

**Jika dibiarkan:** browser bersama dapat menampilkan data pengguna sebelumnya saat offline; rotasi akses tidak menghapus salinan lokal. Skenario lintas akun belum dijalankan di browser nyata. **Tes wajib:** logout/rotasi token lalu offline tidak mengembalikan data privat lama.

### 4. [P1] Endpoint tugas legacy melewati kode akses portal

**Bukti:** reproduksi HTTP. Lokasi: [routes.go:22](F:/Project/bot-wa-jadwal/internal/api/routes.go:22), [legacy.go:269](F:/Project/bot-wa-jadwal/internal/api/legacy/legacy.go:269).

**Alur dan masalah:** portal kelas mode CODE menolak pembaca tanpa sesi, tetapi `GET /api/tasks` tanpa autentikasi masih mengembalikan judul, mata kuliah, kelas, dan deadline tugasnya. Tanpa filter kelas, query menggabungkan tugas lintas kelas; tidak membatasi semester aktif/terpublikasi.

**Perbaikan minimum:** hentikan endpoint baca legacy yang tidak diperlukan atau jalankan gate portal dan filter semester yang sama dengan v1.

**Jika dibiarkan:** mode CODE memberi kesan perlindungan yang tidak berlaku pada seluruh jalur baca. **Tes wajib:** kelas CODE tidak terbaca lewat endpoint alternatif tanpa sesi sah.

### 5. [P1] Permintaan yang diblokir justru membuka kembali percobaan login

**Bukti:** reproduksi HTTP. Lokasi: [auth_controller.go:176](F:/Project/bot-wa-jadwal/internal/api/v1/auth_controller.go:176).

**Alur dan masalah:** `loginBlocked` membaca 32 catatan terbaru, termasuk `BLOCKED`. Setelah lima kegagalan, 32 permintaan yang ditolak menggeser semua catatan FAILURE keluar dari hasil query. Permintaan selanjutnya kembali memeriksa sandi sebelum 15 menit berlalu.

**Perbaikan minimum:** hitung FAILURE dalam jendela waktu sejak SUCCESS atau simpan `blocked_until` yang tidak bergantung pada banyaknya baris BLOCKED. Jangan menerapkan batas jumlah baris sebelum menghitung kegagalan relevan.

**Jika dibiarkan:** pembatas brute force dapat dilewati secara deterministik. **Tes wajib:** ratusan request selama blokir tetap mendapat penolakan hingga waktunya habis.

### 6. [P2] Pergantian konteks memperpanjang batas absolut sesi

**Bukti:** reproduksi HTTP. Lokasi: [auth_controller.go:729](F:/Project/bot-wa-jadwal/internal/api/v1/auth_controller.go:729).

**Alur dan masalah:** sesi berumur 23 jam dengan sisa satu jam mendapat masa hidup baru 24 jam setelah switch context. Handler membuat sesi baru dengan `created_at` sekarang dan `expiresAt = now + TTL`, berbeda dari batas absolut dalam kontrak akses.

**Perbaikan minimum:** pertahankan awal sesi/rantai sesi dan gunakan batas paling awal antara expiry lama dan batas peran tujuan. Rotasi token dan pencabutan token lama dilakukan atomik.

**Jika dibiarkan:** sesi aktif dapat diperpanjang berulang tanpa login ulang. **Tes wajib:** switch dekat expiry tidak memperpanjang batas absolut; kegagalan insert tidak meninggalkan rotasi setengah jadi.

### 7. [P1] Mode dukungan Admin tidak menjadi syarat mutasi akademik

**Bukti:** reproduksi HTTP pada PATCH tugas tanpa support grant. Lokasi: [task_controller.go:535](F:/Project/bot-wa-jadwal/internal/api/v1/task_controller.go:535), [routes.go:87](F:/Project/bot-wa-jadwal/internal/api/routes.go:87).

**Alur dan masalah:** route menerima SYSTEM_ADMIN dan controller membatasi kelas/offering hanya untuk KM/PJ. Support grant dapat mengisi konteks kelas, tetapi tidak diwajibkan dan tidak menjadi pembatas pada jalur tersebut.

**Perbaikan minimum:** untuk aksi akademik Admin, wajibkan grant aktif yang cocok dengan kelas target, alasan dukungan, dan identitas grant pada audit. Terapkan pada setiap operasi akademik yang relevan.

**Jika dibiarkan:** pembatasan dukungan dalam UI dan ACCESS_CONTROL tidak berlaku bagi pemanggil API langsung. **Tes wajib:** tanpa grant, grant kedaluwarsa, dan grant kelas lain ditolak.

### 8. [P1] Data semester arsip masih dapat diubah

**Bukti:** reproduksi HTTP: PATCH tugas semester ARCHIVED menghasilkan 200 dan mengubah judul. Lokasi: [task_controller.go:535](F:/Project/bot-wa-jadwal/internal/api/v1/task_controller.go:535), [task_controller.go:199](F:/Project/bot-wa-jadwal/internal/api/v1/task_controller.go:199).

**Alur dan masalah:** kepemilikan offering diperiksa, tetapi status semester tidak menjadi syarat mutasi. Filter semester pada daftar tugas hanya menyembunyikan data; itu tidak melindungi endpoint berdasarkan ID. Aktivasi semester baru mengarsipkan semester lama tanpa otomatis mencabut seluruh assignment PJ lama.

**Perbaikan minimum:** satu guard mutasi akademik terhadap status kelas/semester, diterapkan di dalam transaksi pada create, edit, review, status, materi, dan jadwal. Jalur restore yang sah perlu aturan tersendiri.

**Jika dibiarkan:** riwayat semester berubah setelah dianggap hanya-baca. **Tes wajib:** semua endpoint mutasi terhadap semester arsip ditolak, termasuk sesi PJ lama.

### 9. [P2] Complete/archive menaikkan reviewed_version tanpa review yang sesuai

**Bukti:** reproduksi HTTP dan query DB. Lokasi: [task_controller.go:1056](F:/Project/bot-wa-jadwal/internal/api/v1/task_controller.go:1056).

**Alur dan masalah:** setelah review APPROVED v1, complete mengubah task menjadi v2 dan `reviewed_version=2`, tetapi tidak menambahkan `task_reviews` v2. Status APPROVED tidak lagi memiliki bukti keputusan untuk versi aktif, melanggar BR-TASK-006.

**Perbaikan minimum:** pisahkan versi isi yang direview dari versi konkurensi status, atau tetapkan aturan transisi review dan catat bukti yang sah secara atomik. Jangan sekadar menyalin keputusan atas nama reviewer.

**Jika dibiarkan:** audit mengatakan versi tertentu disetujui padahal tidak pernah direview. **Tes wajib:** setiap reviewed_version memiliki review yang cocok setelah complete, uncomplete, archive, dan restore.

### 10. [P1] Publikasi dan pembuatan outbox belum atomik

**Bukti:** penelusuran statis. Lokasi: [task_controller.go:351](F:/Project/bot-wa-jadwal/internal/api/v1/task_controller.go:351), [schedule_controller.go:1119](F:/Project/bot-wa-jadwal/internal/api/v1/schedule_controller.go:1119), [common/outbox.go:17](F:/Project/bot-wa-jadwal/internal/api/common/outbox.go:17).

**Alur dan masalah:** transaksi data di-commit lebih dulu, kemudian `QueueNotification` menulis outbox. Helper tidak mengembalikan error ke caller. Crash atau kegagalan DB di antara dua tahap meninggalkan data terbit tanpa intent notifikasi, meskipun request bisnis telah berhasil.

**Perbaikan minimum:** tulis intent melalui transaksi bisnis yang sama dan propagasikan error. Helper `EnqueueTx` yang tersedia juga perlu diperbaiki sebelum dipakai: pembacaannya masih memakai `s.db`, bukan executor transaksi, sehingga tidak melihat seluruh state transaksi dan dapat menunggu koneksi sendiri pada pool satu koneksi.

**Jika dibiarkan:** notifikasi hilang tanpa antrean yang bisa di-retry. **Tes wajib:** injeksi kegagalan insert outbox membatalkan publikasi; retry idempoten tidak menghasilkan intent ganda.

### 11. [P1] Edit dan pencabutan tidak menyingkirkan notifikasi usang

**Bukti:** reproduksi pencabutan event mempertahankan pengumuman lama berstatus PENDING; penelusuran PATCH tugas. Lokasi: [schedule_controller.go:1285](F:/Project/bot-wa-jadwal/internal/api/v1/schedule_controller.go:1285), [task_controller.go:766](F:/Project/bot-wa-jadwal/internal/api/v1/task_controller.go:766).

**Alur dan masalah:** ketika bot offline, pengumuman event mengantre. Revoke hanya menambah pesan pembatalan; pengumuman lama tetap boleh dikirim. PATCH tugas tidak membuat pembaruan notifikasi maupun mengganti payload lama. Deadline yang sudah dikoreksi dapat tetap diumumkan memakai nilai sebelumnya.

**Perbaikan minimum:** saat edit/revoke, tandai intent lama superseded/cancelled dan buat koreksi versi baru dalam transaksi yang sama. Producer dan worker perlu menyepakati versi entitas yang masih berlaku.

**Jika dibiarkan:** mahasiswa menerima informasi salah setelah data di portal sudah benar. **Tes wajib:** publish → edit/revoke saat offline → worker online hanya mengirim informasi yang masih berlaku.

### 12. [P1] Worker mengabaikan kegagalan pencatatan setelah pesan terkirim

**Bukti:** penelusuran statis. Lokasi: [notification_worker.go:251](F:/Project/bot-wa-jadwal/internal/bot/notification_worker.go:251).

**Alur dan masalah:** setelah WhatsApp sukses, UPDATE SENT dan INSERT attempt memakai context batch yang sama dan errornya diabaikan. Jika timeout/DB error terjadi pada tahap itu, baris dapat tetap PROCESSING. Reaper mengembalikannya ke PENDING dan pengiriman berikutnya dapat mengulang pesan yang sudah diterima pengguna.

**Perbaikan minimum:** simpan identitas pengiriman yang stabil sebelum mengirim, tangani hasil persist secara eksplisit, dan selesaikan pencatatan dengan context finalisasi terbatas yang tidak langsung ikut batal. Jangan menyatakan jaminan exactly-once tanpa rekonsiliasi provider.

**Jika dibiarkan:** pesan ganda atau status UI yang salah setelah gangguan. **Tes wajib:** sender sukses lalu context dibatalkan/DB gagal, kemudian reap dan restart worker.

### 13. [P1] Jadwal bot dan portal memakai aturan efektif yang berbeda

**Bukti:** penelusuran statis. Lokasi: [v1_service.go:350](F:/Project/bot-wa-jadwal/internal/bot/v1_service.go:350), [v1_service.go:388](F:/Project/bot-wa-jadwal/internal/bot/v1_service.go:388), [portal_controller.go:1400](F:/Project/bot-wa-jadwal/internal/api/v1/portal_controller.go:1400).

**Alur dan masalah:** resolver bot menekan pola asal hanya untuk ujian. Kuliah Senin yang dipindah ke Rabu dapat tetap tampil reguler pada Senin, sedangkan portal menekan pola asal REPLACEMENT. Query event bot juga hanya mengambil OWNER; kelas PARTICIPANT yang ACCEPTED tidak memperoleh event yang tampil di portalnya.

**Perbaikan minimum:** satu resolver jadwal efektif untuk bot, portal, dan ringkasan notifikasi; formatter pesan tetap terpisah.

**Jika dibiarkan:** mahasiswa memperoleh jawaban berbeda untuk kelas/tanggal yang sama. **Tes wajib:** parity bot/portal untuk replacement lintas hari, cancelled, holiday, ujian, dan peserta accepted.

### 14. [P1] Menyimpan revisi draf ujian dapat menghapus draf tersimpan

**Bukti:** reproduksi mock JavaScript: DELETE berhasil, CREATE gagal, lalu `examDraft` menjadi null. Lokasi: [exam-session.js:267](F:/Project/bot-wa-jadwal/web/js/exam-session.js:267).

**Alur dan masalah:** frontend menghapus draf lama sebelum membuat penggantinya. Kegagalan validasi/jaringan pada CREATE menghilangkan draf yang sebelumnya sudah aman. Jika DELETE gagal, error diabaikan dan CREATE tetap dijalankan, sehingga arah kegagalan lain berpotensi menyisakan duplikat.

**Perbaikan minimum:** gunakan endpoint update draf dengan expected version dalam satu transaksi. Jangan menggunakan delete/create sebagai operasi edit dari browser.

**Jika dibiarkan:** pengguna kehilangan draf atau mendapat draf ganda. **Tes wajib:** kegagalan simpan mempertahankan draf lama beserta identitas dan detailnya.

### 15. [P2] Batas semester event dibandingkan dengan tanggal UTC

**Bukti:** penelusuran statis. Lokasi: [conflict.go:99](F:/Project/bot-wa-jadwal/internal/schedule/conflict.go:99), [conflict.go:111](F:/Project/bot-wa-jadwal/internal/schedule/conflict.go:111).

**Alur dan masalah:** engine telah membaca timezone kelas, tetapi pemeriksaan rentang semester memakai `StartsAt.Format` tanpa konversi ke timezone kelas. Event 1 September pukul 00:30 WIB yang dikirim sebagai UTC terbaca sebagai 31 Agustus dan ditolak pada semester yang mulai 1 September. Peserta juga memerlukan timezone kelas masing-masing.

**Perbaikan minimum:** bandingkan tanggal lokal owner/peserta dengan batas semester masing-masing; tentukan juga kebijakan event yang berakhir melewati batas periode.

**Jika dibiarkan:** event sah pada batas hari ditolak atau event di luar periode diterima. **Tes wajib:** awal/akhir semester dengan timestamp UTC dan offset lokal yang mewakili instant sama.

### 16. [P2] Partisipasi kelas gabungan melewati state machine dan audit

**Bukti:** penelusuran statis. Lokasi: [schedule_controller.go:1815](F:/Project/bot-wa-jadwal/internal/api/v1/schedule_controller.go:1815).

**Alur dan masalah:** endpoint langsung UPDATE status tanpa syarat status sebelumnya, validasi periode semester saat accept, maupun audit/responded_by/responded_at. Kelas yang sudah leave dapat kembali accept tanpa undangan baru. Cabang Admin mengubah seluruh PARTICIPANT event sekaligus.

**Perbaikan minimum:** targetkan satu partisipasi yang sah, periksa transisi sebelumnya dan cakupannya, validasi tanggal, lalu catat respons serta audit dalam transaksi.

**Jika dibiarkan:** keikutsertaan dapat berubah di luar alur undangan dan sulit ditelusuri. **Tes wajib:** REMOVED → ACCEPTED ditolak tanpa reinvite; respons satu kelas tidak mengubah kelas lain.

### 17. [P2] Pesan koreksi restore tanpa kanal tidak pernah direkonsiliasi

**Bukti:** penelusuran statis dua query yang tidak cocok. Lokasi: [scoped_restore.go:422](F:/Project/bot-wa-jadwal/internal/backup/scoped_restore.go:422), [outbox.go:140](F:/Project/bot-wa-jadwal/internal/notify/outbox.go:140).

**Alur dan masalah:** ketika restore mengubah data terpublikasi dan kelas belum punya kanal, koreksi dibuat dengan channel NULL tetapi `scheduled_at=now`. Reconciler hanya memasangkan pesan yang channel DAN scheduled_at keduanya NULL. Worker mensyaratkan channel terisi, sehingga pesan ini tersangkut meskipun kanal kemudian ditambahkan.

**Perbaikan minimum:** gunakan kontrak enqueue yang sama; tanpa kanal, scheduled_at juga NULL, atau revisi reconciler agar konsisten dengan arti waktu jadwal.

**Jika dibiarkan:** koreksi pemulihan tidak sampai kepada kelas. **Tes wajib:** restore tanpa kanal → hubungkan kanal → koreksi menjadi dapat diproses.

## Should fix

### 18. [P2] Aturan bisnis diduplikasi antara controller dan service

**Bukti:** penelusuran pemanggil. Lokasi: [auth/service.go:381](F:/Project/bot-wa-jadwal/internal/auth/service.go:381) dibanding [auth_controller.go:699](F:/Project/bot-wa-jadwal/internal/api/v1/auth_controller.go:699); [task/lifecycle.go:139](F:/Project/bot-wa-jadwal/internal/task/lifecycle.go:139) dibanding [task_controller.go:535](F:/Project/bot-wa-jadwal/internal/api/v1/task_controller.go:535).

**Alur dan masalah:** route HTTP menulis aturan bisnis sendiri meskipun domain service mempunyai operasi setara. Contoh konkret: SwitchContext domain mempertahankan batas absolut, handler HTTP justru memperpanjangnya. Pengujian domain yang lulus tidak menjamin API menggunakan perilaku itu. Ada pula dua implementasi pengiriman antrean, `notify.ProcessDue` dan `NotificationWorker.ProcessPending`.

**Perbaikan minimum:** pilih satu implementasi per operasi bisnis dan pindahkan caller secara bertahap dengan tes kontrak HTTP. Controller cukup parsing, auth context, pemanggilan operasi, dan pemetaan respons. Jangan langsung menghapus service tanpa memeriksa caller legacy/migrasi/tests.

**Jika dibiarkan:** setiap perubahan aturan perlu diperbaiki di beberapa tempat dan mudah tidak seragam. **Lean:** pengurangan baris mungkin besar, tetapi belum dihitung sebagai diff yang aman; tidak ada angka penghematan atau penghapusan dependency yang diklaim.

### 19. [P2] Pengambilan semua tugas menelan error dan menjalankan enam request

**Bukti:** reproduksi mock JavaScript. Lokasi: [api.js:273](F:/Project/bot-wa-jadwal/web/js/api.js:273).

**Alur dan masalah:** `getAllTasks` meminta enam tab, mengubah semua kegagalan menjadi `[]`, lalu menggabungkan hasil. Enam kegagalan autentikasi/jaringan menghasilkan daftar kosong yang terlihat sukses. Tab aktif/review/terlewat juga saling beririsan sehingga pekerjaan query dan transfer berulang.

**Perbaikan minimum:** propagasikan error yang menentukan keberhasilan pemuatan dan sediakan query daftar yang benar-benar dibutuhkan layar, dengan filter/pagination server bila datanya bertambah. Tidak perlu framework frontend atau build step baru.

**Jika dibiarkan:** pengguna mengira tugas hilang dan server mengerjakan pembacaan berulang. **Tes wajib:** 401 dan network failure menghasilkan error state; satu tab gagal tidak diam-diam dianggap hasil lengkap.

### 20. [P2] CRUD materi melalui HTTP tidak mencatat audit perubahan

**Bukti:** penelusuran statis jalur tulis. Lokasi: [academic_controller.go:1426](F:/Project/bot-wa-jadwal/internal/api/v1/academic_controller.go:1426), [baris 1533](F:/Project/bot-wa-jadwal/internal/api/v1/academic_controller.go:1533), [baris 1634](F:/Project/bot-wa-jadwal/internal/api/v1/academic_controller.go:1634).

**Alur dan masalah:** create, patch, dan delete/archive materi menulis langsung ke DB dan mengembalikan sukses tanpa audit perubahan. Repository materi memiliki dukungan audit, tetapi jalur HTTP ini tidak menggunakannya. Perubahan URL atau penghilangan materi tidak mempunyai snapshot aktor/sebelum/sesudah yang setara dengan tugas.

**Perbaikan minimum:** satukan mutasi materi dan audit di satu transaksi, menggunakan satu operasi domain yang dipanggil HTTP.

**Jika dibiarkan:** koreksi materi sulit ditelusuri dan kontrak audit tidak konsisten. **Tes wajib:** setiap mutasi menghasilkan audit ber-scope yang tepat; kegagalan audit membatalkan mutasi.

## Hasil verifikasi

| Pemeriksaan | Hasil |
|---|---|
| `go test ./...` pada kondisi awal | Lulus seluruh package yang dijalankan |
| `go vet ./...` | Lulus |
| Delapan probe HTTP dengan `httptest` dan database `t.TempDir()` | Semua berhasil mereproduksi kondisi cacat yang diuji |
| Mock JavaScript: draf ujian | Terbukti DELETE mendahului CREATE gagal |
| Mock JavaScript: service worker | Terbukti respons API `no-store` tetap dikirim ke cache |
| Mock JavaScript: getAllTasks | Terbukti enam error menjadi hasil sukses kosong |
| `go test -race` pada bot/chat/notify/schedule/auth | Tidak berjalan: CGO nonaktif dan compiler C tidak tersedia |

Delapan probe HTTP mencakup takeover akun, undangan lintas kelas, mutasi arsip/dukungan, invariant review, bypass portal legacy, perpanjangan sesi, bypass blokir login, dan notifikasi event yang sudah dicabut. Probe bersifat pembuktian cacat: PASS berarti perilaku bermasalah teramati, bukan berarti aplikasi aman. Pada penyusunan awal satu probe memakai nama kolom fixture yang salah; fixture diperbaiki dan seluruh delapan probe kemudian dijalankan kembali dengan hasil di atas.

Berkas probe sementara dihapus setelah audit. Source aplikasi, tests milik pengguna, staging area, dan database produksi tidak diubah oleh audit. Laporan ini adalah artefak yang ditambahkan.

`go test ./...` juga menemukan package lokal di `storage/exam-qa-20261025/seed` karena cakupan perintah Go mencakup folder itu; tidak ada pembacaan database/sesi runtime yang dilakukan secara sengaja. Paket backup diuji melalui pengujian API restore yang sudah ada, sehingga label `[no test files]` di package backup bukan bukti tidak adanya cakupan restore.

## Bagian yang sudah baik

- Query parameterized dan transaksi sudah digunakan di banyak jalur tulis.
- Sesi server-side memakai hash token; cookie pengelola HttpOnly dan konfigurasi produksi memvalidasi secure cookie/origin.
- Cukup banyak pengujian API, scope, migrasi, konflik, restore, dan outbox sudah tersedia.
- Backup memakai snapshot transaksi, checksum, preview token, pre-restore backup, dan maintenance gate.
- Version check pada banyak operasi membantu mencegah kehilangan perubahan karena dua pengelola menyimpan bersamaan.
- Frontend tetap memakai aset lokal tanpa kebutuhan npm/build step baru.

Fondasi ini layak dipertahankan. Masalah terbesar adalah aturan yang berbeda antarjalur aktif, bukan kebutuhan mengganti stack.

## Urutan perbaikan yang disarankan

1. Tutup takeover dan scope undangan (#1–2), lalu tambahkan tes keamanan di route HTTP nyata. Periksa penggunaan akun/undangan existing bila sistem sudah pernah diekspos; audit ini sendiri tidak memeriksa log produksi.
2. Tutup jalur baca alternatif dan cache privat (#3–4), perbaiki limiter/sesi serta support dan archive guard (#5–8).
3. Benahi invariant review dan alur publish/outbox/worker (#9–12), lalu samakan hasil jadwal bot/portal (#13).
4. Hilangkan delete/create draf dan rapikan timezone, partisipasi, serta koreksi restore (#14–17).
5. Konsolidasikan implementasi domain berdasarkan kasus regresi yang sudah jelas (#18), perbaiki pemuatan frontend dan audit materi (#19–20).

Jangan memulai dengan rewrite menyeluruh. Untuk setiap perbaikan: tambahkan satu regresi melalui jalur yang benar-benar dipakai, perbaiki akar masalah, jalankan test relevan, kemudian suite dan vet. Setelah correctness stabil, ukur query/latensi sebelum mengubah pool SQLite atau menambah cache.

## Belum diverifikasi

Tidak menjalankan koneksi WhatsApp nyata, pengiriman provider, restore terhadap data produksi, uji beban terukur, browser end-to-end, audit aksesibilitas/visual, race detector yang berhasil, maupun audit kerentanan dependency/CVE. Source vendored tidak dibaca baris demi baris. Tidak semua cabang parser legacy, import, migration/seed, CSS, dan partial HTML diperiksa secara penuh. Temuan statis perlu regresi terarah saat perbaikan; lulus test yang ada tidak membuktikan ketiadaan bug lain.
