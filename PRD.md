# Product Requirements Document (PRD): Bot WhatsApp Jadwal Kuliah Otomatis (Golang)

## 1. Latar Belakang & Permasalahan (Problem Statement)
* **Permasalahan Utama:** Jadwal kuliah yang dibagikan oleh pihak jurusan/prodi sering kali berupa gambar atau dokumen PDF dengan format penulisan yang disingkat-singkat (menggunakan kode mata kuliah dan inisial kode dosen). 
* **Dampak:** Mahasiswa sering merasa pusing, kesulitan membaca jadwal secara cepat, salah melihat ruangan, atau keliru mengenali dosen pengampu di awal semester.
* **Kebutuhan Pengguna:** Dibutuhkan sebuah sistem pencari informasi jadwal yang cepat, interaktif, dan dapat diakses langsung melalui aplikasi chat harian yang biasa digunakan tanpa harus membuka portal akademik yang lambat atau melihat gambar jadwal yang rumit.

## 2. Solusi Produk (Product Solution)
* Mengembangkan sebuah **Bot WhatsApp Otomatis** berbasis bahasa pemrograman **Go (Golang)** menggunakan *library* `whatsmeow` dan *database* lokal SQLite (`modernc.org/sqlite`).
* Bot ini berfungsi sebagai asisten virtual kelas yang dapat menerjemahkan kode-kode singkatan jadwal menjadi informasi yang *human-readable* (nama matkul lengkap, nama dosen lengkap, waktu, dan ruangan) berdasarkan perintah teks (*command*) dari pengguna.

## 3. Ruang Lingkup Fitur (Scope & Features)
* **Penyimpanan Data Terstruktur (`jadwal.json`):** Seluruh data jadwal harian, mata kuliah, dosen, dan ruangan disimpan dalam format JSON agar mudah dibaca dan dikelola oleh program.
* **Perintah Cek Jadwal (`/jadwal [hari]`):** Pengguna dapat meminta jadwal spesifik hari tertentu (contoh: `/jadwal senin`).
* **Pencarian Berdasarkan Dosen atau Ruangan:** Memungkinkan pengguna mencari jadwal berdasarkan kode dosen atau lokasi ruangan.
* **Sesi Mandiri (Persistent Session):** Bot menggunakan SQLite murni Go sehingga tidak memerlukan *scan* QR code berulang kali saat bot direstart.

## 4. Cara Kerja Sistem (Workflow / Architecture)
1. **Inisiasi & Autentikasi:** Program Go berjalan, memuat data sesi dari `sesi_bot.db`, dan terhubung ke WhatsApp (menyediakan QR code sekali di awal untuk *pairing*).
2. **Pemuatan Data (Data Loading):** Saat bot menyala, program membaca file `jadwal.json` ke dalam memori aplikasi (*struct* Go).
3. **Event Listener (Pesan Masuk):** *Library* `whatsmeow` mendengarkan *event* pesan masuk dari pengguna.
4. **Command Processing:** 
   * Bot memvalidasi teks pesan yang masuk (mengabaikan huruf besar/kecil).
   * Bot mencocokkan perintah dengan data yang ada di memori.
5. **Response Delivery:** Bot mengirimkan balasan terstruktur kembali ke chat pengguna secara instan.

## 5. Target Pengembangan Selanjutnya (Next Action Items for AI Agent)
* Membuat struktur *file* `jadwal.json` yang menampung data jadwal kuliah lengkap beserta pemetaan kodenya.
* Menulis fungsi *parser* di Go untuk membaca file JSON tersebut ke dalam variabel *struct*.
* Mengimplementasikan fungsi *Event Handler* pada `whatsmeow` untuk menangkap teks masuk dan merespons perintah pengguna secara dinamis.

*Catatan Pembelajaran:* Setiap memberikan potongan kode baru, AI Agent wajib menyertakan penjelasan logika singkat di baliknya agar mahasiswa IT yang mengembangkan proyek ini dapat memahami alur kerjanya dengan baik.