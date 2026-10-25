# FEAT-SCH-EXAM-01: sesi ujian granular

Target: v3.1-pilot. Implementasi mengikuti model event dan otorisasi yang sudah ada.

| Tiket | Pekerjaan | Bukti penerimaan |
| --- | --- | --- |
| BE-015.1 | Migrasi 016 dan baseline menerima EXAM_UTS/EXAM_UAS | Data, foreign key, indeks, tautan rapat, dan versi lama bertahan |
| BE-015.2 | Validasi domain/API: origin opsional tetapi harus berpasangan dan milik offering | Uji draf, input salah, hak KM/PJ, publikasi dan konflik |
| BE-015.3 | Resolusi bot/portal mengganti hanya occurrence asal | Ujian di tempat, pindah hari, mandiri, dan revoke |
| BE-015.4 | Label WhatsApp dan pengingat H-1 ujian | Uji formatter pagi/sore tanpa mengirim pesan nyata |
| FE-016.1 | Modal Tandai Ujian KM/PJ, UTS/UAS, tanggal/jam, ruang, catatan | Draf, pratinjau, konfirmasi publikasi, kegagalan, keyboard |
| FE-016.2 | Badge ujian dan catatan portal | Portal publik, jadwal campuran, light/dark, lebar 390 px |
| QA-017 | Regresi dan verifikasi UI | go test ./..., go vet ./..., build, laporan interaksi |

## Keputusan implementasi

- Nomor 010 sudah dipakai; migrasi baru menggunakan 016.
- Ujian dengan origin mengganti satu pola pada satu tanggal; tanpa origin menambah sesi mandiri.
- Catatan menyimpan sifat ujian dan pengawas, sesuai payload tiket; tidak menebak pengawas dari dosen.
- Publikasi tetap mengikuti konfirmasi ruang TU dan pemeriksaan konflik yang sudah berlaku.
- Antislop diterapkan selama pengerjaan, sesuai pilihan pengguna.
- Arah visual mengikuti DESIGN.md: ENERGY 1 / RHYTHM 1 / MOTION 1.
- Badge amber/merah menyatakan status ujian; label teks menjaga makna tanpa bergantung pada warna.
- Font, spasi, kontrol, dan dialog mengikuti primitive proyek untuk keterbacaan serta konsistensi KM/PJ/portal.
- Pengujian menggunakan database terisolasi; sesi WhatsApp produksi tidak digunakan.
