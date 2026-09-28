const MockData = {
  classes() {
    return [
          { id: 1, name: 'D4 TI 2024 A', code: 'D4 TI 2024 A', slug: 'd4-ti-2024-a', semester: 'Semester 3' },
          { id: 2, name: 'D4 TI 2024 B', code: 'D4 TI 2024 B', slug: 'd4-ti-2024-b', semester: 'Semester 3' }
    ];
  },
  offerings() {
    return [
          { id: 1, code: 'PLP-T', name: 'Pemrograman Lanjut', type: 'Teori', pj: 'Pengguna Contoh A' },
          { id: 2, code: 'PLP-P', name: 'Pemrograman Lanjut', type: 'Praktik', pj: 'Pengguna Contoh B' },
          { id: 3, code: 'SBD-T', name: 'Sistem Basis Data', type: 'Teori', pj: 'Pengguna Contoh C' },
          { id: 4, code: 'ALIN-T', name: 'Aljabar Linear', type: 'Teori', pj: 'Pengguna Contoh D' },
          { id: 5, code: 'OS-T', name: 'Sistem Operasi', type: 'Teori', pj: 'Pengguna Contoh E' }
    ];
  },
  tasks() {
    return [
          {
            id: 1,
            course_offering_id: 3,
            course_code: 'SBD-T',
            course_name: 'Sistem Basis Data',
            title: 'Normalisasi Skema Perpustakaan',
            instructions: 'Ubah tabel transaksi ke bentuk normal ketiga. Sertakan alasan untuk setiap pemisahan tabel.',
            deadline_at: '2026-09-25T21:00:00+07:00',
            submission_target: 'Tautan LMS kelas',
            status: 'PUBLISHED',
            review_status: 'NOT_REVIEWED',
            is_completed: false,
            creator_name: 'Pengguna Contoh C'
          },
          {
            id: 2,
            course_offering_id: 2,
            course_code: 'PLP-P',
            course_name: 'Pemrograman Lanjut',
            title: 'Refaktor Modul Inventaris',
            instructions: 'Implementasikan pola Repository dan Unit of Work pada modul inventaris barang kampus.',
            deadline_at: '2026-09-28T20:00:00+07:00',
            submission_target: 'Repositori GitHub Kelas',
            status: 'DRAFT',
            review_status: 'NOT_REVIEWED',
            is_completed: false,
            creator_name: 'Pengguna Contoh B'
          },
          {
            id: 3,
            course_offering_id: 4,
            course_code: 'ALIN-T',
            course_name: 'Aljabar Linear',
            title: 'Latihan Ruang Vektor',
            instructions: 'Kerjakan soal latihan bab 4 nomor 1 sampai 15 pada buku pegangan.',
            deadline_at: '2026-10-01T18:00:00+07:00',
            submission_target: 'Tautan LMS kelas',
            status: 'PUBLISHED',
            review_status: 'APPROVED',
            is_completed: false,
            creator_name: 'Pengguna Contoh D'
          },
          {
            id: 4,
            course_offering_id: 5,
            course_code: 'OS-T',
            course_name: 'Sistem Operasi',
            title: 'Ringkasan Manajemen Proses',
            instructions: 'Tuliskan esai ringkas mengenai algoritma penjadwalan CPU Round Robin vs Shortest Job First.',
            deadline_at: '2026-09-23T19:00:00+07:00',
            submission_target: 'Google Form Pengumpulan',
            status: 'PUBLISHED',
            review_status: 'APPROVED',
            is_completed: true,
            creator_name: 'Pengguna Contoh E'
          }
    ];
  },
  schedule() {
    return [
          { hari: 1, waktu: '08.00 - 09.40 WIB', kode: 'PLP-T', mata_kuliah: 'Pemrograman Lanjut', jenis: 'Teori', pengajar: 'Dosen Contoh A', ruangan: 'R. Teori 3', berlaku: '26 Agu - 18 Des 2026', status_label: 'Reguler' },
          { hari: 2, waktu: '10.00 - 11.40 WIB', kode: 'SBD-T', mata_kuliah: 'Sistem Basis Data', jenis: 'Teori', pengajar: 'Dosen Contoh B', ruangan: 'R. Teori 2', berlaku: '26 Agu - 18 Des 2026', status_label: 'Kelas Pengganti' },
          { hari: 3, waktu: '13.00 - 15.30 WIB', kode: 'PLP-P', mata_kuliah: 'Pemrograman Lanjut', jenis: 'Praktik', pengajar: 'Dosen Contoh C', ruangan: 'Lab 301', berlaku: '26 Agu - 18 Des 2026', status_label: 'Reguler' },
          { hari: 4, waktu: '08.00 - 09.40 WIB', kode: 'ALIN-T', mata_kuliah: 'Aljabar Linear', jenis: 'Teori', pengajar: 'Dosen Contoh D', ruangan: 'R. Teori 1', berlaku: '26 Agu - 18 Des 2026', status_label: 'Reguler' },
          { hari: 5, waktu: '09.00 - 10.40 WIB', kode: 'OS-T', mata_kuliah: 'Sistem Operasi', jenis: 'Teori', pengajar: 'Dosen Contoh E', ruangan: 'R. Teori 4', berlaku: '26 Agu - 18 Des 2026', status_label: 'Reguler' }
    ];
  },
  events() {
    return [
          {
            jenis: 'Kelas Pengganti',
            mata_kuliah: 'Sistem Basis Data, Teori',
            jadwal_semula: 'Selasa, 29 September 2026, 10.00 - 11.40 WIB, R. Teori 2',
            jadwal_baru: 'Rabu, 30 September 2026, 15.30 - 17.10 WIB, Lab 302',
            alasan: 'Dosen menghadiri kegiatan program studi',
            status_publikasi: 'Terbit',
            status_wa: 'Menunggu dikirim'
          },
          {
            jenis: 'Kelas Tambahan',
            mata_kuliah: 'Pemrograman Lanjut, Praktik',
            jadwal_semula: 'Sesi Tambahan Materi Inventaris',
            jadwal_baru: 'Sabtu, 3 Oktober 2026, 09.00 - 11.30 WIB, Lab 301',
            alasan: 'Pendalaman materi sebelum evaluasi tengah semester',
            status_publikasi: 'Draf',
            status_wa: 'Belum dijadwalkan'
          }
    ];
  }
};
