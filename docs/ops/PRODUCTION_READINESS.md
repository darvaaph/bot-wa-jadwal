# Kesiapan produksi

Status 14 Oktober 2026: **belum disetujui untuk produksi**. Perbaikan jalur data, antrean notifikasi, dan pemeriksaan kesiapan sudah ada di source code. Persetujuan rilis memerlukan uji staging dengan akun WhatsApp dan data kelas nyata yang terisolasi.

## Arsitektur yang berlaku

- `storage/bot_v1.db` menyimpan kelas, semester, jadwal efektif, tugas, materi, kanal WhatsApp, pengaturan pengingat, dan antrean notifikasi.
- `storage/tugas.db` mendukung fitur kompatibilitas lama. Jangan pakai database ini sebagai sumber pemetaan kanal atau pengingat produksi.
- Grup hanya terhubung ke kelas saat `whatsapp_channels.status = 'ACTIVE'`. System Admin menautkan atau mencabutnya melalui dashboard. Perubahan terbaca pada pesan berikutnya.
- Beberapa perintah baca bot masih mempunyai jalur kompatibilitas JSON/`tugas.db` untuk kelas yang belum tersedia di v1. Manifest migrasi saat ini hanya memuat **2 kelas pilot** dari **19 berkas jadwal kelas**. Sebelum rilis semua kelas sasaran, cocokkan kelas, semester, jadwal, dan tugas di `bot_v1.db`; jika belum dimigrasikan, hasil bot dan portal dapat berbeda.
- Scheduler v1 membuat ringkasan pagi, pengingat tugas sore, dan pengingat kuliah pengganti dalam `notification_messages`. Worker yang sama mengirimnya lewat WhatsApp. Pengingat pagi dan sore bisa menyusul maksimal satu jam setelah waktu setelan jika proses baru hidup kembali. Entri yang tertinggal dalam status `PROCESSING` akan dikembalikan ke antrean setelah 10 menit. Pengiriman bersifat **at least once**: bila proses mati setelah WhatsApp menerima pesan tetapi sebelum status DB tersimpan, pesan dapat terkirim ulang.
- Portal materi hanya menampilkan respons API. Tidak ada materi contoh otomatis.
- Frontend memakai Tailwind yang sudah menjadi CSS statis dan font lokal. Skrip/style inline pada halaman utama sudah dipindahkan ke berkas lokal, sehingga CSP tidak lagi mengizinkan skrip inline. Alpine versi saat ini masih mengevaluasi ekspresi HTML sehingga `script-src` masih memerlukan `unsafe-eval`.

## Konfigurasi sebelum menjalankan mode produksi

Set `BOT_JADWAL_ENV=production`, `BOT_JADWAL_AUTH_HASH_KEY` minimal 32 byte acak, `BOT_JADWAL_SECURE_COOKIES=true`, `BOT_JADWAL_PUBLIC_BASE_URL` sebagai origin HTTPS tanpa path, dan masukkan origin tersebut ke `BOT_JADWAL_ALLOWED_ORIGINS`. Atur `BOT_JADWAL_TRUSTED_PROXY_CIDRS` bila aplikasi berada di belakang reverse proxy. Simpan nilai rahasia di pengelola secret; jangan commit `.env`, DB, atau sesi WhatsApp.

Gunakan `STORAGE_DIR` untuk direktori persisten yang bisa ditulis proses. Jalankan hanya satu instance pengirim WhatsApp untuk satu sesi dan database. Gunakan `-session storage/sesi_dev.db` serta nomor uji saat staging; biarkan `storage/sesi_bot.db` produksi tetap utuh.

`GET /api/health` memeriksa proses HTTP hidup. `GET /api/ready` mengembalikan 200 hanya bila database v1 dapat diakses, migrasi terbaru sudah terpasang, dan WhatsApp tersambung pada mode bot. Pada `-web-only`, koneksi WhatsApp tidak diperiksa. Endpoint ini tidak membuktikan bahwa kelas, semester, atau akun admin sudah disiapkan; gate data dan UAT tetap wajib.

## Gate sebelum rilis

1. Pada salinan data staging, jalankan `go test -count=1 ./...`, `go vet ./...`, dan build biner untuk target host.
2. Jalankan `-web-only` dengan `STORAGE_DIR` baru. Pastikan `/api/ready` = 200. Provision System Admin pertama dan siapkan kelas/semester uji sebelum mencoba login dan portal. Portal kelas kosong tidak boleh menampilkan materi palsu; terbitkan satu materi lalu pastikan hanya materi itu yang muncul.
3. Jalankan mode bot dengan sesi uji. Periksa `/api/ready` sebelum dan sesudah koneksi WhatsApp, tautkan grup uji ke kelas A, kirim `!jadwal` dan `!portal`, ubah tautan ke kelas B, lalu cabut. Pastikan pesan setelah setiap perubahan menggunakan kelas yang tepat atau memberi panduan onboarding.
4. Atur waktu pengingat kelas uji pada menit berikutnya. Periksa hanya satu `notification_messages` per jenis dan tanggal, isi ringkasan sesuai jadwal efektif, dan `notification_attempts` mencatat ID pesan dari provider.
5. Uji pemulihan worker dengan entri `PROCESSING` lama, pesan tanpa kanal, dan bot offline. Pastikan tidak ada pengiriman ke kanal yang dicabut.
6. Uji backup dan restore pada salinan terisolasi, lalu lakukan UAT di desktop serta ponsel untuk peran System Admin, KM, PJ, dan mahasiswa.
7. **Blokir rilis publik sampai `unsafe-eval` hilang dari CSP.** Tailwind runtime dan skrip/style inline sudah dilepas, tetapi build Alpine CSP resmi belum kompatibel dengan ekspresi pada partial saat ini (misalnya optional chaining dan callback dalam atribut). Migrasikan ekspresi itu ke metode JavaScript dan uji semua peran di browser sebelum mengganti Alpine standar dengan build CSP. `style-src-attr 'unsafe-inline'` masih diperlukan untuk style dinamis pada beberapa sheet/modal; tinjau dan kurangi setelah migrasi skrip.
8. Hitung seluruh kelas sasaran dan pastikan perintah bot yang dipakai mahasiswa membaca kelas v1 yang sama dengan portal. Selama masih ada kelas yang memakai jalur kompatibilitas, jangan nyatakan bot dan portal tersinkron penuh.

Jangan menandai gate selesai hanya dari tes unit. Catat versi commit, lingkungan, hasil uji, dan penanggung jawab sebelum deploy.
