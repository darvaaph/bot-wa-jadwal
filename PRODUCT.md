# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Go application with an embedded plain HTML dashboard, CDN-hosted Tailwind CSS and Alpine.js, SQLite storage, and WhatsApp integration through whatsmeow. The repository does not use npm or a frontend build step.

## Users

- Mahasiswa membuka portal kelas dari ponsel untuk melihat jadwal, tugas, perubahan, ruangan, dan tautan yang diizinkan tanpa membuat akun.
- PJ Mata Kuliah memperbarui tugas, materi, dan jadwal untuk mata kuliah yang ditugaskan pada semester tertentu.
- Ketua Murid mengelola seluruh mata kuliah, PJ, semester, dan koreksi pada kelasnya.
- System Admin membuat kelas, menunjuk KM awal, mengelola master data lintas kelas, memulihkan akses, dan memantau sistem.

## Product Purpose

Bot Jadwal menyediakan satu sumber informasi kelas yang lebih mudah diperbarui dan dibaca daripada command serta riwayat percakapan WhatsApp. Dashboard web menjadi tempat membaca dan mengelola data, sedangkan WhatsApp menjadi kanal siaran, pengingat, dan jalur darurat.

## Positioning

Produk memusatkan metadata jadwal dan tugas per kelas serta semester, membatasi perubahan berdasarkan penugasan pengurus, dan mempertahankan riwayat publikasi serta koreksi. Produk bukan Learning Management System dan tidak menyimpan nilai atau berkas tugas mahasiswa.

## Operating Context

PJ dan KM paling sering memakai ponsel ketika menerima informasi dari dosen di kelas atau grup WhatsApp. Mahasiswa membuka tautan kelas dari pesan WhatsApp. Satu akun pengurus dapat memiliki beberapa konteks kelas, semester, atau mata kuliah. Gangguan koneksi bot tidak boleh menghentikan akses dashboard atau publikasi web.

## Capabilities and Constraints

- Portal mahasiswa hanya-baca melalui tautan atau kode kelas.
- Akun pengurus dibuat melalui undangan. PJ dan KM menggunakan satu halaman login.
- Kelas memiliki identitas permanen dan tepat satu semester aktif.
- Semester dapat disiapkan melalui impor JSON, salinan semester lama, atau input manual.
- PJ dapat memublikasikan tugas dan perubahan jadwal dalam cakupannya. KM dapat membatalkan perubahan jadwal kelasnya.
- Data lintas kelas harus terisolasi. Tindakan penting masuk audit log.
- Rekomendasi ruangan berasal dari data internal dan tetap memerlukan konfirmasi Tata Usaha.
- Antarmuka harus mobile-first, dapat digunakan dengan keyboard, dan tidak menyampaikan status hanya melalui warna.

## Brand Commitments

Nama produk adalah Bot Jadwal. Nada antarmuka harus langsung, tenang, dan mudah dipahami oleh mahasiswa serta pengurus kelas. Arah visual yang berlaku tercatat dalam [DESIGN.md](DESIGN.md).

## Evidence on Hand

- Wawancara 11 pengguna pada `docs/user-interviews/`.
- Product Definition, User Requirements, Access Control, User Flows, Business Rules, dan Functional Requirements pada `docs/product/`.
- Implementasi backend, API awal, bot WhatsApp, data jadwal JSON, dan pengujian Go dalam repository.
- Belum ada testimoni, metrik penggunaan produksi, atau aset merek final yang boleh ditampilkan sebagai bukti publik.

## Product Principles

1. Informasi kelas harus lebih cepat ditemukan daripada di riwayat WhatsApp.
2. Operasi umum pengurus harus dapat diselesaikan dari ponsel tanpa menghafal command.
3. Konteks kelas, semester, mata kuliah, dan peran harus selalu jelas.
4. Perubahan tentatif tidak boleh tersiar, sedangkan publikasi dan koreksi harus dapat ditelusuri.
5. Data lama diarsipkan dan dipulihkan, bukan ditimpa atau dihapus tanpa jejak.

## Accessibility & Inclusion

Kontrol utama memiliki target sentuh minimal 44 x 44 piksel. Navigasi dan dialog dapat digunakan dengan keyboard, fokus terlihat, label dapat dibaca teknologi bantu, serta status memakai teks atau ikon selain warna. Bahasa utama produk adalah Bahasa Indonesia.
