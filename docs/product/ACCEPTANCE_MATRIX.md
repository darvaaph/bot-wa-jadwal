# Matriks Penerimaan Produk

Status: **belum menjadi gerbang rilis yang lulus**. Setiap baris harus diuji pada backend dan UI dengan database salinan multi-kelas sebelum deployment. `Otomatis` berarti ada uji Go untuk jalur inti; `Manual` berarti bukti lintas perangkat atau operasi nyata masih perlu dicatat. Respons gagal harus mempertahankan data sebelum aksi.

| Flow | Skenario penerimaan | Verifikasi |
|---|---|---|
| UF-ACCESS-001 | Admin membuat kelas dan undangan KM; undangan sekali pakai menghasilkan penugasan aktif | Otomatis + Manual |
| UF-ACCESS-002 | KM mengundang PJ hanya untuk offering kelasnya; konteks tidak bisa diganti penerima | Otomatis + Manual |
| UF-ACCESS-003 | Akun lama menerima penugasan baru tanpa akun ganda | Otomatis |
| UF-ACCESS-004 | Login menampilkan pilihan konteks aktif; pergantian konteks menyegarkan cakupan | Otomatis + Manual |
| UF-ACCESS-005 | Pemulihan akun membatalkan sesi lama dan mencatat alasan | Otomatis |
| UF-ACCESS-006 | PJ/KM pengganti aktif tanpa mencabut pengurus terakhir secara tak sengaja | Otomatis |
| UF-PORTAL-001 | Tautan/kode kelas membuka portal baca saja; kode salah/kedaluwarsa ditolak | Otomatis + Manual |
| UF-PORTAL-002 | Ringkasan, Jadwal, Tugas, Mata Kuliah, Perubahan, Arsip cocok dengan semester terbit | Otomatis + Manual |
| UF-TASK-001 | PJ menyimpan draf, meninjau, menerbitkan; tugas muncul di portal dan satu outbox | Otomatis + Manual |
| UF-TASK-002 | KM mereview versi terbit; keputusan stale ditolak dan riwayat tetap ada | Otomatis |
| UF-TASK-003 | Edit/arsip/pulihkan tugas mengikuti versi, soft delete, dan audit | Otomatis + Manual |
| UF-SCH-001 | Jam mulai + durasi menghasilkan selesai; pratinjau konflik sebelum simpan | Otomatis + Manual |
| UF-SCH-002 | Perubahan sementara melalui draf event, pratinjau, dan publikasi | Otomatis + Manual |
| UF-SCH-003 | Permanen menutup versi asal sehari sebelum tanggal berlaku; sesi sebelum/pada/setelah benar | Otomatis + Manual |
| UF-SCH-004 | KM mencabut event terbit, portal kembali ke pola berlaku, koreksi antre | Otomatis |
| UF-SCH-005 | Peserta lintas kelas menerima/menolak; restore yang menyentuh relasi ini tertahan | Otomatis |
| UF-SEM-001 | Impor JSON memvalidasi lalu membuat semester draf tanpa mengganti semester aktif | Otomatis + Manual |
| UF-SEM-002 | Salin/input manual menghasilkan draf yang bisa ditinjau | Otomatis + Manual |
| UF-ROOM-001 | Kandidat ruang sesuai interval; konfirmasi TU ditampilkan sebagai riwayat | Otomatis + Manual |
| UF-OPS-001 | Bot offline tidak membatalkan publikasi; status pesan tetap terlihat | Otomatis + Manual |
| UF-OPS-002 | Edit bersamaan memunculkan 409; input lokal dipertahankan | Otomatis + Manual |
| UF-OPS-003 | KM meminta backup, Admin mengeksekusi; restore pratinjau, titik pemulihan, rollback, isolasi kelas | Otomatis + Manual |

| Requirement | Skenario penerimaan | Verifikasi |
|---|---|---|
| FR-ACCESS-001 | Portal hanya menampilkan data terbit pada kelas yang sah | Otomatis |
| FR-ACCESS-002 | Login gagal/berhasil dan rate limit tidak membocorkan akun | Otomatis |
| FR-ACCESS-003 | Token undangan sekali pakai, kedaluwarsa, dan revoke | Otomatis |
| FR-ACCESS-004 | PJ offering, KM kelas, Admin global; objek asing ditolak | Otomatis |
| FR-ACCESS-005 | Pilihan konteks mengubah seluruh query dan mutasi berikutnya | Otomatis + Manual |
| FR-ACCESS-006 | Sesi kedaluwarsa, suspend, dan recovery membatalkan token | Otomatis |
| FR-ACCESS-007 | Peran terakhir terlindungi dan jejak pergantian dapat dibaca | Otomatis |
| FR-CLASS-001 | Pembuatan, edit, dan arsip kelas memvalidasi identitas unik | Otomatis + Manual |
| FR-CLASS-002 | Data kelas A tidak tampil atau berubah saat konteks B | Otomatis |
| FR-SEM-001 | Semester baru dimulai sebagai draf | Otomatis |
| FR-SEM-002 | Impor invalid memberi daftar galat tanpa data setengah jadi | Otomatis |
| FR-SEM-003 | Pratinjau aktivasi dan perpindahan semester bersifat atomik | Otomatis + Manual |
| FR-SEM-004 | Arsip hanya dibaca bila pernah dipublikasikan | Otomatis + Manual |
| FR-SCH-001 | Jadwal efektif memilih pola versi berlaku dan event terbit | Otomatis |
| FR-SCH-002 | Form manual menghitung selesai, validasi konflik, dan pratinjau | Otomatis + Manual |
| FR-SCH-003 | Draf perubahan tidak muncul di portal | Otomatis |
| FR-SCH-004 | Konflik ruangan/offering/dosen ditandai konsisten di pratinjau dan publish | Otomatis |
| FR-SCH-005 | Publikasi atomik; kegagalan WA tidak membatalkan data terbit | Otomatis |
| FR-SCH-006 | `effective_from` hari ini..akhir semester, versi lama ditutup, 409 untuk stale/ulang | Otomatis |
| FR-SCH-007 | Revoke menghasilkan koreksi tanpa menghapus pesan lama | Otomatis |
| FR-SCH-008 | Owner tepat satu dan partisipasi kelas lain dijaga | Otomatis |
| FR-TASK-001 | Draf boleh belum lengkap, publikasi wajib lengkap | Otomatis |
| FR-TASK-002 | Preview dan publish mengikuti peran serta versi | Otomatis + Manual |
| FR-TASK-003 | Review KM append-only dan versi stale ditolak | Otomatis |
| FR-TASK-004 | Filter tugas memisahkan aktif, draf, selesai, arsip | Otomatis + Manual |
| FR-TASK-005 | Edit tugas terbit memperbarui versi dan status review/notifikasi | Otomatis |
| FR-TASK-006 | Selesai, arsip, soft delete, pulihkan adalah state terpisah | Otomatis |
| FR-TASK-007 | Detail Mata Kuliah menyatukan materi, dosen, jadwal, tugas; materi tertutup tidak bocor | Otomatis + Manual |
| FR-NOTIF-001 | Ringkasan harian sekali per jadwal dan kelas | Otomatis |
| FR-NOTIF-002 | Pengingat tugas mengikuti deadline dan konfigurasi kelas | Otomatis |
| FR-NOTIF-003 | Perubahan/revoke/restore menerbitkan satu koreksi baru bila perlu | Otomatis |
| FR-NOTIF-004 | Outbox idempoten, retry tidak menduplikasi pesan lama | Otomatis |
| FR-NOTIF-005 | Waktu pengingat bersumber dari backend, bukan perangkat PJ | Otomatis + Manual |
| FR-NOTIF-006 | Pengingat kelas pengganti hanya untuk event terbit | Otomatis |
| FR-ROOM-001 | Kandidat bebas bentrok pada interval yang dipilih | Otomatis + Manual |
| FR-ROOM-002 | Kontak, status, dan riwayat konfirmasi TU tersimpan sesuai cakupan | Otomatis + Manual |
| FR-ROOM-003 | Master ruangan hanya dapat diubah Admin | Otomatis |
| FR-AUDIT-001 | Mutasi terbit, restore, dan penugasan mempunyai aktor/alasan | Otomatis |
| FR-AUDIT-002 | PJ/KM/Admin hanya membaca audit sesuai cakupan | Otomatis |
| FR-AUDIT-003 | Audit dan review tidak bisa diubah/hapus lewat aplikasi | Otomatis |
| FR-OPS-001 | Database akademik tetap dapat dibaca saat bot offline | Otomatis |
| FR-OPS-002 | Backup v2 akademik saja; restore multi-kelas, lintas kelas ditolak, rollback dan prepoint terbukti | Otomatis + Manual |
| FR-OPS-003 | Data baru sesudah backup tidak aktif tetapi jejaknya tetap ada | Otomatis |
| FR-OPS-004 | Versi stale mengembalikan 409 tanpa mutasi parsial | Otomatis |
| FR-UX-001 | Form utama dapat digunakan pada ponsel dan desktop tanpa overflow | Manual |
| FR-UX-002 | Setiap data punya loading/kosong/gagal/berhasil/akses ditolak | Manual |
| FR-UX-003 | Pengurus baru menemukan aksi utama dan konteks tanpa bantuan | Manual |
| FR-UX-004 | Keyboard, fokus, label, dan kontras dapat dipakai | Manual |

## Aturan akses yang harus diulang untuk setiap aksi

| Aturan dari `ACCESS_CONTROL.md` | Penerimaan |
|---|---|
| §5 Portal | Tanpa akun hanya baca kelas/semester terbit; kode sesi invalid 401/403; endpoint mutasi tertolak |
| §6 Undangan | Admin→KM, KM→PJ kelasnya; token dipakai sekali dan tidak dapat mengubah scope |
| §7 Sesi | Suspend, revoke, recovery, dan perpindahan konteks segera mengubah hak akses |
| §8.1 Informasi akademik | Mahasiswa hanya publikasi, PJ hanya offering, KM kelas, Admin sesuai dukungan |
| §8.2 Tugas | PJ membuat/ubah/terbit di offering, KM review/koreksi kelas, mahasiswa tak bisa mutasi |
| §8.3 Jadwal | PJ perubahan di offering; KM boleh revoke; versi permanen tunduk tanggal dan scope |
| §8.4 Pengguna/peran | KM tidak bisa mengundang KM/Admin; Admin kelola global |
| §8.5 Operasional | KM hanya meminta backup; Admin eksekusi/restore; kandidat ruang dan audit dibatasi scope |
| §9 Backend | Semua penolakan terjadi di server sekalipun UI disiasati; objek asing 404 generik jika perlu |
| §10–12 Siklus dan konflik peran | Hak lama hilang saat dicabut; pemilik multi-peran memilih konteks aktif eksplisit |
| §13–15 Audit/galat | Aksi sensitif beralasan, input invalid 422, versi stale 409, data tidak berubah saat gagal |

Gerbang rilis: isi bukti per baris (nama tes, hasil, dan tangkapan layar ponsel/desktop), jalankan `go test ./...` dan `go vet ./...`, lalu lakukan restore pada **salinan database** dan pulihkan lagi memakai `pre_restore_backup_id`. Jangan menyatakan production-ready sebelum seluruh baris lulus.

Verifikasi 2026-10-02: `go test ./...` dan `go vet ./...` lulus. Restore kelas diuji pada salinan `storage/bot_v1.db` yang dimigrasikan ke skema 15 di direktori sementara: perubahan tugas kembali ke isi backup, lalu titik pemulihan yang dibuat sebelum restore dipakai untuk mengembalikan perubahan itu. Uji otomatis juga memastikan dosen penawaran yang muncul setelah backup tidak lagi aktif. Salinan dan paket uji telah dibersihkan. Bukti UI KM/PJ/Admin/portal pada ponsel dan desktop serta seluruh baris manual masih harus dilengkapi sebelum gerbang rilis dinyatakan lulus.
