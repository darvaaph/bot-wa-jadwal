# Backend Completion Tickets: BE-007, BE-008, BE-010, dan BE-011

Dokumen ini merinci pekerjaan untuk mematangkan kontrak schedule pattern, task lifecycle, notification retry, dan backup/restore. Setiap ticket memuat kondisi repository saat dokumen dibuat, ruang lingkup, acceptance criteria, test matrix, Definition of Done, dan prompt implementasi.

## Ringkasan

| ID | Judul | Prioritas | Kondisi saat ini | Dependensi |
|---|---|---:|---|---|
| BE-007 | Finalisasi Kontrak Schedule Pattern | P1 | Implementasi dasar tersedia, perlu verifikasi lengkap dan integrasi dengan BE-006 | BE-006 untuk conflict engine terpusat |
| BE-008 | Hardening Task Lifecycle dan Review | P1 | Versioning dasar tersedia, concurrency dan konsistensi review perlu dikunci | BE-009 untuk audit writer kanonis |
| BE-010 | Rapikan Notification Retry Lifecycle | P1 | Retry terbatas ke status tertentu, response dan concurrency masih perlu dirapikan | BE-005 dan BE-009 |
| BE-011 | Finalisasi Semantics Backup dan Restore | P1 | Endpoint saat ini hanya memverifikasi backup, sedangkan requirement produk juga menyebut full restore | Keputusan ADR wajib |

Urutan implementasi yang disarankan:

1. BE-007 setelah interface conflict engine BE-006 stabil.
2. BE-008 setelah bentuk audit writer BE-009 disepakati.
3. BE-010 setelah durable outbox BE-005 tersedia.
4. BE-011 setelah keputusan verify-only atau full restore dicatat dalam ADR.

## Baseline repository

Kondisi berikut sudah ada pada branch saat dokumen dibuat:

- Commit `3febb1c` memperketat conflict, version, dan idempotency jadwal.
- Commit `a038054` memperketat kontrak task, semester, dan material.
- Schedule pattern create dan patch sudah menerima `lecturer_ids`.
- Schedule pattern patch sudah meminta `version` dan membuat baris versi baru dalam transaksi.
- Complete, archive, dan restore task sudah meminta `version` serta menaikkan version secara atomik.
- Notification retry sudah membatasi status awal ke `FAILED` dan `CANCELLED`.
- `POST /api/v1/restores` saat ini menghitung ulang checksum dan menandai backup sebagai `VERIFIED`. Endpoint belum mengganti database aktif.

Baseline tersebut belum otomatis menutup ticket. Setiap ticket tetap memerlukan pengujian terhadap seluruh acceptance criteria dan integrasi dengan BE-005, BE-006, serta BE-009.

## Aturan umum implementasi

- Baca `AGENTS.md` sebelum mengubah kode.
- Gunakan kontrak API v1 dan ADR accepted sebagai sumber keputusan utama.
- Jangan mempercayai class, semester, offering, atau user ID dari client sebagai bukti akses.
- Gunakan SQLite sementara pada test. Jangan menyentuh database atau sesi WhatsApp di `storage/`.
- Jangan menelan error dari query, `RowsAffected`, audit, atau `Commit`.
- Tambahkan regression test sebelum atau bersama implementasi.
- Jalankan `gofmt` hanya pada file Go yang berubah.
- Jangan commit atau push kecuali diminta.
- Jalankan gate berikut sebelum menyatakan ticket selesai:

```powershell
go test -count=1 ./...
go vet ./...
go build -o bin/bot ./cmd/bot
git diff --check
```

Referensi utama:

- [API v1](api/API_V1.md)
- [Business Rules](product/BUSINESS_RULES.md)
- [Functional Requirements](product/FUNCTIONAL_REQUIREMENTS.md)
- [Data Model](product/DATA_MODEL.md)
- [Access Control](product/ACCESS_CONTROL.md)
- [Backend Schema](spec/be-v1-schema.md)

---

## BE-007: Finalisasi Kontrak Schedule Pattern

### Status

Sebagian besar mekanisme dasar sudah diimplementasikan. Ticket belum ditutup karena pemeriksaan konflik masih tersebar di handler dan belum ada bukti lengkap untuk concurrency, rollback, effective range, serta parity dengan conflict engine BE-006.

### Masalah

Schedule pattern merupakan sumber jadwal reguler. Perubahan permanen tidak boleh menghapus riwayat. Backend harus menutup pattern lama dan membuat pattern baru secara atomik. Kegagalan validasi, conflict check, audit, insert versi baru, atau commit tidak boleh meninggalkan pattern lama dalam keadaan tertutup tanpa pengganti.

Kontrak juga harus menjamin:

- `lecturer_ids` hanya berisi dosen yang terhubung ke offering;
- PJ hanya mengelola offering assignment aktifnya;
- KM hanya mengelola offering dalam kelas assignment aktifnya;
- perpindahan offering tidak menembus scope;
- optimistic locking menghasilkan `409 VERSION_CONFLICT`;
- conflict engine yang sama dipakai oleh create, patch, preview, dan publish;
- versi serta effective range dapat dihitung secara deterministik.

### Tujuan

Menjadikan create dan patch schedule pattern aman terhadap stale update, kegagalan transaksi, konflik jadwal, dan akses lintas scope. Riwayat pattern lama harus tetap dapat dipakai untuk menghitung jadwal pada tanggal sebelum versi baru berlaku.

### Ruang lingkup

1. Validasi request create dan patch dengan aturan yang sama.
2. Wajibkan `offering_id`, `day_of_week`, `start_time`, `duration_min`, dan lecturer sesuai kontrak.
3. Hitung `end_time` di server.
4. Validasi setiap lecturer terhadap `offering_lecturers`.
5. Ambil class dan semester melalui relasi offering, bukan dari payload.
6. Terapkan scope PJ, KM, dan System Admin secara eksplisit.
7. Gunakan BE-006 conflict engine untuk room, lecturer, class, dan offering overlap.
8. Wajibkan `version > 0` pada patch.
9. Tutup pattern lama dan buat pattern baru pada satu transaksi.
10. Tulis audit pada transaksi yang sama melalui BE-009.
11. Kembalikan ID pattern baru, version baru, dan effective range yang aktif.
12. Pertahankan jadwal historis berdasarkan pattern lama.

### Di luar ruang lingkup

- Menghapus pattern lama secara fisik.
- Mengubah teaching event menjadi schedule pattern.
- Membuat UI editor jadwal.
- Mengimplementasikan ulang conflict query di handler.

### Aturan versioning dan effective range

- Pattern baru dimulai dari `effective_from` yang valid pada semester offering.
- Pattern lama diakhiri sebelum pattern baru mulai. Gunakan semantik tanggal yang terdokumentasi dan uji batas hari secara eksplisit.
- Patch dengan version stale tidak boleh menutup pattern lama.
- Insert versi baru yang gagal harus membatalkan perubahan `effective_until` pattern lama.
- Audit gagal harus membatalkan kedua perubahan.
- Dua patch dengan version yang sama tidak boleh sama-sama berhasil.
- Response sukses mengembalikan version baru yang dapat langsung dipakai frontend.

### Aturan scope

| Aktor | Akses |
|---|---|
| PJ | Hanya offering dari assignment aktifnya |
| KM | Offering pada kelas assignment aktifnya |
| System Admin | Scope global sesuai operasi dukungan yang terdokumentasi |
| GUEST atau session tanpa assignment | Ditolak |

Perubahan `offering_id` pada patch hanya diizinkan jika offering tujuan masih berada dalam scope aktor dan semester yang valid. Jika produk tidak membutuhkan perpindahan offering, tolak perubahan tersebut dan dokumentasikan kontraknya.

### Response minimum

Response patch sukses harus menyediakan data yang cukup bagi frontend:

```json
{
  "id": 123,
  "replaces_pattern_id": 122,
  "version": 4,
  "effective_from": "2026-10-01",
  "effective_until": null
}
```

Envelope final mengikuti helper response API v1.

### Acceptance criteria

- [ ] Create menerima dan memvalidasi `lecturer_ids`.
- [ ] Lecturer yang tidak terhubung ke offering menghasilkan `422`.
- [ ] Patch tanpa version atau dengan version nol menghasilkan `422`.
- [ ] Patch stale menghasilkan `409` beserta `current_version`.
- [ ] Patch sukses menutup pattern lama dan membuat satu pattern baru.
- [ ] Kegagalan insert pattern baru mempertahankan pattern lama tetap aktif.
- [ ] Kegagalan audit mempertahankan pattern lama tetap aktif dan tidak membuat pattern baru.
- [ ] Dua patch bersamaan dengan version sama menghasilkan tepat satu pemenang.
- [ ] Perpindahan offering mematuhi scope aktor.
- [ ] Room, lecturer, class, dan offering conflicts diperiksa oleh BE-006.
- [ ] Pattern historis tetap digunakan untuk tanggal sebelum effective version baru.
- [ ] Response mengembalikan ID dan version pattern baru.
- [ ] Tidak ada query conflict bisnis yang diduplikasi di handler.

### Test matrix

| Skenario | Hasil yang diharapkan |
|---|---|
| Create dengan lecturer valid | Pattern dan relasi dosen tersimpan |
| Create dengan lecturer di luar offering | `422 VALIDATION` |
| PJ memakai offering lain | `403 FORBIDDEN` |
| KM memakai offering kelas lain | `403 FORBIDDEN` |
| Patch tanpa version | `422 VALIDATION` |
| Patch dengan stale version | `409 VERSION_CONFLICT` |
| Dua patch version sama | Satu sukses, satu conflict |
| Insert versi baru dipaksa gagal | Pattern lama tetap aktif |
| Audit dipaksa gagal | Seluruh mutation rollback |
| Room atau lecturer overlap | Ditolak sesuai hasil BE-006 |
| Interval bersebelahan | Tidak dianggap overlap |
| Tanggal sebelum versi baru | Jadwal memakai pattern lama |
| Tanggal sesudah versi baru | Jadwal memakai pattern baru |

### Definition of Done

- Create dan patch pattern memakai service conflict yang sama.
- Seluruh mutation dan audit berjalan pada transaksi yang sama.
- Tidak ada jalur yang menutup pattern lama sebelum seluruh prasyarat lulus.
- Regression test mencakup stale version, concurrency, scope, conflict, dan rollback.
- Dokumentasi request dan response sesuai implementasi.
- Test, vet, build, dan diff check lulus.

### Prompt implementasi

```text
Kerjakan ticket BE-007 Finalisasi Kontrak Schedule Pattern pada repository bot-wa-jadwal.

Baca AGENTS.md, docs/api/API_V1.md bagian schedule patterns, docs/product/BUSINESS_RULES.md bagian BR-SCH-001 sampai BR-SCH-009, docs/product/FUNCTIONAL_REQUIREMENTS.md bagian FR-SCH-001 sampai FR-SCH-006, docs/product/ACCESS_CONTROL.md, dan docs/product/DATA_MODEL.md. Periksa juga pekerjaan BE-006 dan BE-009 jika sudah tersedia.

Baseline saat ini sudah memiliki lecturer_ids, version check, transaksi patch, dan pembuatan pattern versi baru. Jangan menulis ulang tanpa membuktikan gap. Audit implementasi aktual terhadap acceptance criteria ticket ini.

Selesaikan hal berikut:
1. Satukan validasi create dan patch pada service domain yang dapat diuji.
2. Validasi lecturer_ids terhadap offering_lecturers.
3. Ambil scope class dan semester dari offering, bukan dari payload.
4. Tegakkan scope PJ, KM, dan System Admin.
5. Gunakan conflict engine BE-006. Hapus query konflik duplikat dari handler.
6. Patch wajib version lebih dari nol dan memakai optimistic update pada pattern lama.
7. Tutup pattern lama, buat pattern baru, salin relasi dosen, dan tulis audit BE-009 dalam satu transaksi.
8. Kegagalan pada langkah apa pun harus rollback penuh.
9. Pastikan effective_from dan effective_until menghasilkan jadwal historis yang benar.
10. Response sukses mengembalikan ID serta version pattern baru.

Kerjakan secara test-first. Tambahkan integration test SQLite sementara untuk lecturer invalid, PJ dan KM lintas scope, omitted version, stale version, dua patch concurrent, insert failure, audit failure, conflict, serta pemilihan pattern lama dan baru berdasarkan tanggal. Jangan menyentuh storage produksi.

Jalankan gofmt pada file yang berubah, go test -count=1 ./..., go vet ./..., go build -o bin/bot ./cmd/bot, dan git diff --check. Laporkan file yang berubah, aturan effective range final, bukti rollback, dan risiko tersisa. Jangan commit atau push kecuali diminta.
```

---

## BE-008: Hardening Task Lifecycle dan Review

### Status

Version wajib dan optimistic update untuk complete, archive, dan restore sudah tersedia. Ticket masih memerlukan penguncian concurrent review, konsistensi `review_state` dengan `task_reviews`, response conflict yang lengkap, serta migrasi audit ke BE-009.

### Masalah

Task memiliki dua kelompok state yang berbeda:

- publication lifecycle seperti `DRAFT`, `PUBLISHED`, dan `REVOKED`;
- marker operasional seperti `completed_at`, `archived_at`, dan soft delete.

Review KM juga terkait dengan versi task tertentu. Backend tidak boleh menganggap approval versi lama berlaku setelah PJ mengubah isi tugas. Dua request review atau lifecycle yang berjalan bersamaan tidak boleh menghasilkan state dan audit yang saling bertentangan.

### Tujuan

Menjamin seluruh perubahan task memakai optimistic locking, menjaga riwayat review per versi, dan melakukan mutation, review insert, audit, serta outbox dalam transaksi yang sama jika operasi memicu notifikasi.

### Ruang lingkup

1. Create dan patch task sesuai scope offering.
2. Patch selalu memakai `WHERE id = ? AND version = ?`.
3. Perubahan yang terlihat pengguna menaikkan version.
4. Review mengacu ke `task_version` aktif.
5. `CHANGES_REQUESTED` dan `REVOKED` wajib memiliki note non-kosong.
6. Review concurrent terhadap versi sama menghasilkan satu keputusan yang sah sesuai policy.
7. Complete, archive, dan restore meminta version positif.
8. Lifecycle update menaikkan version secara atomik.
9. Restore membersihkan pasangan soft delete secara konsisten.
10. Audit memakai BE-009 dan outbox memakai BE-005 bila relevan.
11. Semua endpoint menerapkan scope PJ, KM, dan System Admin sesuai dokumentasi.

### Invariant task

- `reviewed_version` hanya menunjuk versi yang benar-benar memiliki review relevan.
- Perubahan oleh PJ membuat versi baru berstatus `NOT_REVIEWED`.
- Perubahan oleh KM yang otomatis disetujui membuat review `APPROVED` untuk versi baru dalam transaksi yang sama.
- Review untuk version stale ditolak dengan `409`.
- `completed_at` dan `archived_at` tidak mengganti `publication_status`.
- Restore tidak menghapus audit pengarsipan atau penghapusan sebelumnya.
- Task terhapus atau diarsipkan tidak kembali ke daftar aktif sampai restore berhasil.

### Kebijakan concurrency review

Implementasi harus memilih dan mendokumentasikan satu aturan:

1. Hanya satu review final boleh dibuat untuk satu task version, atau
2. Riwayat beberapa keputusan diizinkan tetapi transisi state harus memakai compare-and-swap yang memastikan hanya satu transisi dari state awal yang menang.

Jangan mengandalkan pemeriksaan status sebelum transaksi tanpa predicate pada update. Unique constraint harus mendukung policy yang dipilih.

### Acceptance criteria

- [ ] Patch tanpa version atau version nol menghasilkan `422`.
- [ ] Patch stale menghasilkan `409` dengan current version dan current data yang relevan.
- [ ] Complete, archive, dan restore selalu membutuhkan version positif.
- [ ] Semua lifecycle update memakai predicate version dan menaikkan version.
- [ ] `CHANGES_REQUESTED` dan `REVOKED` tanpa note menghasilkan `422`.
- [ ] Review untuk task version stale menghasilkan `409`.
- [ ] Dua review concurrent tidak menghasilkan state akhir yang ambigu.
- [ ] Patch PJ mengatur versi baru menjadi `NOT_REVIEWED`.
- [ ] Patch KM membuat review `APPROVED` untuk versi baru dalam transaksi yang sama.
- [ ] Complete dan archive tidak mengubah publication lifecycle.
- [ ] Restore membersihkan `deleted_at` dan `deleted_by_user_id` bersama-sama.
- [ ] Kegagalan audit atau outbox membuat mutation rollback sesuai transaction boundary.
- [ ] Scope lintas offering dan lintas kelas ditolak.

### Test matrix

| Skenario | Hasil yang diharapkan |
|---|---|
| PATCH tanpa version | `422 VALIDATION` |
| PATCH stale | `409 VERSION_CONFLICT` |
| Dua PATCH version sama | Satu sukses, satu conflict |
| Review task version lama | `409 VERSION_CONFLICT` |
| Dua review bersamaan | Satu state transition yang sah |
| `CHANGES_REQUESTED` tanpa note | `422 VALIDATION` |
| `REVOKED` dengan note whitespace | `422 VALIDATION` |
| Patch PJ | Version naik, review state `NOT_REVIEWED` |
| Patch KM | Version naik dan review `APPROVED` tersimpan |
| Complete task | `completed_at` terisi, publication status tetap |
| Archive task | `archived_at` terisi, publication status tetap |
| Restore task | Marker archive dan soft delete dibersihkan sesuai kontrak |
| Audit dipaksa gagal | Mutation dan review rollback |
| PJ mengakses offering lain | `403 FORBIDDEN` |
| KM mengakses kelas lain | `403 FORBIDDEN` |

### Definition of Done

- Semua task mutation memakai optimistic locking.
- Review state dapat dijelaskan dari task version dan row `task_reviews`.
- Concurrency test membuktikan hanya transisi sah yang berhasil.
- Mutation, review, audit, dan outbox memakai transaction boundary yang terdokumentasi.
- Tidak ada physical delete pada task v1.
- Test, vet, build, dan diff check lulus.

### Prompt implementasi

```text
Kerjakan ticket BE-008 Hardening Task Lifecycle dan Review pada repository bot-wa-jadwal.

Baca AGENTS.md, docs/api/API_V1.md bagian task dan review, docs/product/BUSINESS_RULES.md bagian BR-TASK, docs/product/FUNCTIONAL_REQUIREMENTS.md bagian FR-TASK, docs/product/DATA_MODEL.md, dan docs/product/ACCESS_CONTROL.md. Periksa BE-005 dan BE-009 jika sudah tersedia.

Baseline saat ini sudah mewajibkan version pada complete, archive, dan restore serta memakai optimistic update. Verifikasi perilaku aktual sebelum mengubahnya. Fokuskan pekerjaan pada gap, khususnya concurrency review, konsistensi reviewed_version, rollback, dan audit kanonis.

Selesaikan hal berikut:
1. Semua PATCH dan lifecycle request wajib version lebih dari nol.
2. Semua mutation memakai predicate version dan menaikkan version bila data terlihat pengguna berubah.
3. Stale version menghasilkan 409 dengan current_version dan current_data yang dibutuhkan frontend.
4. Review harus menunjuk task_version aktif.
5. CHANGES_REQUESTED dan REVOKED wajib note non-kosong.
6. Tentukan dan dokumentasikan policy concurrent review. Gunakan predicate database atau unique constraint agar hanya transisi sah yang menang.
7. Patch PJ membuat versi baru NOT_REVIEWED.
8. Patch KM membuat APPROVED review untuk versi baru dalam transaksi yang sama.
9. Complete, archive, dan restore tidak mengubah publication_status secara tidak sengaja.
10. Restore membersihkan deleted_at dan deleted_by_user_id secara konsisten tanpa menghapus audit lama.
11. Gunakan audit writer BE-009 dan transactional outbox BE-005 bila mutation memicu notifikasi.
12. Tegakkan scope PJ dan KM dari assignment aktif.

Kerjakan secara test-first. Tambahkan test untuk omitted version, zero version, stale version, concurrent patch, concurrent review, review version stale, note kosong, PJ dan KM lintas scope, audit failure rollback, serta state complete, archive, dan restore. Gunakan SQLite temporary dan jangan menyentuh storage produksi.

Jalankan gofmt, go test -count=1 ./..., go vet ./..., go build -o bin/bot ./cmd/bot, dan git diff --check. Laporkan policy concurrency final, file yang berubah, bukti database state, dan risiko tersisa. Jangan commit atau push kecuali diminta.
```

---

## BE-010: Rapikan Notification Retry Lifecycle

### Status

Handler saat ini membatasi retry ke `FAILED` dan `CANCELLED`, memakai conditional update, dan menulis audit dalam transaksi. Handler masih menghitung serta mengembalikan `attempt_number` sebelum worker benar-benar melakukan pengiriman. BE-010 harus menghilangkan makna attempt yang menyesatkan dan mengunci perilaku concurrent retry.

### Masalah

`notification_attempts` harus merepresentasikan percobaan delivery nyata. Request retry hanya menjadwalkan ulang pesan. Request tersebut tidak boleh membuat attempt palsu atau mengklaim bahwa nomor attempt tertentu sudah dimulai.

Dua retry yang bersamaan juga tidak boleh menjadwalkan dua delivery. Retry terhadap pesan yang sedang diproses, sudah terkirim, atau belum gagal harus ditolak secara konsisten.

### Tujuan

Menjadikan retry sebagai transisi status atomik pada message yang sama. Worker tetap menjadi satu-satunya komponen yang membuat `notification_attempts` saat delivery benar-benar dimulai.

### Ruang lingkup

1. Definisikan status yang dapat diretry.
2. Gunakan compare-and-swap pada status pesan.
3. Jangan membuat atau memesan attempt saat request retry.
4. Jangan mengembalikan `attempt_number` yang belum ada.
5. Worker menentukan nomor attempt pada saat klaim delivery.
6. Lindungi concurrent retry dan concurrent worker claim.
7. Tulis audit retry melalui BE-009.
8. Gunakan outbox dan reconciliation BE-005 untuk pesan tanpa channel.
9. Pastikan scope KM dan System Admin.
10. Dokumentasikan response sebagai scheduled, bukan sent atau processed.

### Keputusan status

Baseline backlog mengizinkan retry untuk `FAILED` dan `CANCELLED`. Implementasi final harus memastikan:

- `FAILED` dapat kembali ke `PENDING`.
- `CANCELLED` hanya dapat diretry jika cancellation memang reversibel menurut penyebabnya.
- `SUPERSEDED` tidak dapat diretry karena isinya sudah usang.
- `SENT` tidak dapat diretry melalui endpoint yang sama karena berisiko mengirim duplikat.
- `PROCESSING` tidak dapat diretry manual.
- Pesan yang menunggu konfigurasi channel tetap berada pada state yang ditentukan BE-005.

Jika `CANCELLED` ternyata mencakup pembatalan permanen akibat data usang, batasi retry ke `FAILED` dan perbarui API v1 serta collection. Jangan membiarkan satu status mempunyai dua arti.

### Response minimum

Response sukses harus menjelaskan penjadwalan ulang, bukan attempt yang belum terjadi:

```json
{
  "id": 10,
  "status": "PENDING",
  "scheduled_at": "2026-09-29T10:00:00Z",
  "retry_scheduled": true
}
```

### Acceptance criteria

- [ ] Retry hanya menerima status yang diputuskan dalam kontrak.
- [ ] Retry memakai row yang sama dan tidak membuat message baru.
- [ ] Request retry tidak membuat `notification_attempts`.
- [ ] Response tidak mengembalikan nomor attempt yang belum tersimpan.
- [ ] Worker membuat attempt saat delivery benar-benar dimulai.
- [ ] Dua retry bersamaan menghasilkan satu sukses dan satu conflict.
- [ ] Retry bersamaan dengan worker claim tidak menghasilkan dua delivery.
- [ ] `SENT`, `PROCESSING`, `PENDING`, dan `SUPERSEDED` ditolak.
- [ ] Pesan lintas kelas tidak dapat diretry oleh KM.
- [ ] Audit retry tersimpan dalam transaksi yang sama dengan perubahan status.
- [ ] Audit gagal membuat status pesan tetap seperti semula.
- [ ] Idempotency key dan message ID tidak berubah.

### Test matrix

| Skenario | Hasil yang diharapkan |
|---|---|
| Retry FAILED | Baris sama menjadi PENDING |
| Retry CANCELLED yang reversibel | Baris sama menjadi PENDING |
| Retry SENT | Ditolak |
| Retry PROCESSING | Ditolak |
| Retry PENDING | Ditolak |
| Retry SUPERSEDED | Ditolak |
| Dua retry bersamaan | Satu sukses, satu conflict |
| Retry dan worker claim bersamaan | Maksimal satu delivery claim |
| Audit dipaksa gagal | Status tidak berubah |
| KM kelas lain | `403 FORBIDDEN` |
| Admin global | Diizinkan |
| Setelah worker mulai mengirim | Satu attempt baru dengan nomor unik |

### Definition of Done

- Endpoint retry hanya menjadwalkan ulang message.
- Hanya worker yang membuat attempt.
- Response tidak mengklaim attempt sebelum delivery dimulai.
- Concurrency test membuktikan tidak ada double scheduling atau double send.
- State machine notifikasi terdokumentasi dan konsisten dengan BE-005.
- Test, vet, build, dan diff check lulus.

### Prompt implementasi

```text
Kerjakan ticket BE-010 Rapikan Notification Retry Lifecycle pada repository bot-wa-jadwal.

Baca AGENTS.md, docs/api/API_V1.md bagian notifications, docs/product/BUSINESS_RULES.md bagian BR-NOTIF, docs/product/FUNCTIONAL_REQUIREMENTS.md bagian FR-NOTIF, docs/product/DATA_MODEL.md bagian notification_messages dan notification_attempts, serta pekerjaan BE-005 dan BE-009 jika sudah tersedia.

Baseline saat ini sudah membatasi retry ke FAILED dan CANCELLED serta memakai conditional update. Masalah yang masih terlihat: handler menghitung dan mengembalikan attempt_number sebelum worker melakukan delivery. notification_attempts hanya boleh mewakili percobaan pengiriman nyata.

Selesaikan hal berikut:
1. Putuskan status retry final. FAILED wajib didukung. CANCELLED hanya didukung jika status tersebut benar-benar reversibel; SUPERSEDED, SENT, PROCESSING, dan PENDING harus ditolak.
2. Retry mengubah status message yang sama menjadi PENDING dengan conditional update.
3. Jangan insert, reserve, atau mengembalikan attempt_number pada request retry.
4. Response menyatakan retry_scheduled dan scheduled_at.
5. Worker menjadi satu-satunya pembuat notification_attempts ketika delivery dimulai.
6. Claim worker dan retry harus aman terhadap concurrency.
7. Dua request retry bersamaan menghasilkan tepat satu sukses.
8. Pertahankan message ID dan idempotency key.
9. Tegakkan scope KM dan System Admin.
10. Gunakan audit writer BE-009. Audit failure harus rollback perubahan status.
11. Integrasikan state menunggu channel dari BE-005.

Kerjakan secara test-first. Tambahkan test untuk setiap status, dua concurrent retry, retry versus worker claim, scope lintas kelas, audit rollback, dan attempt creation oleh worker. Verifikasi database, bukan hanya status HTTP.

Jalankan gofmt, go test -count=1 ./..., go vet ./..., go build -o bin/bot ./cmd/bot, dan git diff --check. Laporkan state machine final, perubahan response, file yang berubah, serta bukti concurrency. Jangan commit atau push kecuali diminta.
```

---

## BE-011: Finalisasi Semantics Backup dan Restore

### Status

Backup melalui `VACUUM INTO` dan verifikasi checksum sudah tersedia. `POST /api/v1/restores` saat ini tidak mengganti database. Endpoint hanya membaca berkas, membandingkan checksum, mengubah status menjadi `VERIFIED`, dan menulis audit.

Dokumentasi belum sepenuhnya selaras:

- `API_V1.md` mendefinisikan endpoint restore sebagai verifikasi berkas fisik dan checksum.
- FR-OPS-002 menyatakan System Admin dapat menjalankan restore terbatas cakupan dengan rollback jika gagal.

Perbedaan ini harus diselesaikan melalui ADR sebelum implementer mengubah data produksi.

### Tujuan

Menghilangkan ambiguitas antara verifikasi backup dan pemulihan database. Frontend, operator, dokumentasi, status database, dan response API harus menyatakan efek yang benar.

### Decision gate

Pilih satu jalur dan catat dalam ADR.

#### Opsi A: Verify-only untuk API v1

Rekomendasi untuk scope saat ini karena sesuai kontrak frozen `API_V1.md` dan risiko operasionalnya lebih rendah.

- `POST /api/v1/restores` memverifikasi artifact, checksum, schema compatibility, dan scope metadata.
- Endpoint tidak mengganti database aktif.
- Response menyatakan `restore_performed: false`.
- Status backup menjadi `VERIFIED`, bukan `RESTORED`.
- Frontend memakai label Verifikasi Backup, bukan Pulihkan Database.
- Full restore menjadi ticket operasi dan versi API terpisah.

#### Opsi B: Full restore

Pilih hanya jika produk membutuhkan pemulihan database melalui aplikasi pada milestone ini.

- Server masuk maintenance mode.
- Sistem menolak mutation baru dan menunggu request aktif selesai.
- Sistem membuat recovery point dari database aktif.
- Sistem memverifikasi checksum, schema version, foreign key, dan scope package.
- Sistem menutup koneksi database aktif.
- Sistem mengganti file secara atomik pada filesystem yang sama.
- Sistem membuka database hasil restore dan menjalankan health check.
- Jika health check gagal, sistem mengembalikan recovery point.
- Sistem mencatat hasil operasi pada audit dan log operasional tanpa secret.
- Runbook menjelaskan recovery jika proses berhenti pada setiap fase.

### Ruang lingkup bersama

1. Validasi `backup_id` dan reason sesuai keputusan produk.
2. Pastikan artifact path tetap berada di storage backup yang dikonfigurasi.
3. Tolak path traversal dan symlink escape.
4. Verifikasi checksum sebelum perubahan state.
5. Verifikasi format SQLite dan schema compatibility.
6. Verifikasi metadata class atau scope jika backup bersifat terbatas.
7. Gunakan status backup yang memiliki arti tunggal.
8. Tulis audit melalui BE-009.
9. Jangan masukkan isi backup atau credential ke response dan audit.
10. Sinkronkan dokumentasi frontend, API collection, dan runbook.

### Model status yang perlu diputuskan

Status minimum harus memiliki definisi jelas:

| Status | Makna |
|---|---|
| `CREATING` | Snapshot sedang dibuat |
| `READY` | Artifact selesai dibuat dan checksum tersimpan |
| `VERIFIED` | Artifact telah lolos verifikasi, database belum dipulihkan |
| `RESTORING` | Full restore sedang berjalan, hanya untuk Opsi B |
| `RESTORED` | Full restore selesai dan health check lulus, hanya untuk Opsi B |
| `FAILED` | Operasi terakhir gagal dengan error operasional yang tercatat |

Constraint schema, API response, dan frontend harus memakai definisi yang sama.

### Acceptance criteria Opsi A

- [ ] ADR menetapkan endpoint sebagai verify-only untuk v1.
- [ ] Response menyatakan `restore_performed: false`.
- [ ] Status akhir `VERIFIED` tidak diklaim sebagai database sudah dipulihkan.
- [ ] Frontend dan dokumentasi memakai istilah verifikasi backup.
- [ ] Checksum mismatch menghasilkan `422 CHECKSUM_MISMATCH`.
- [ ] Artifact hilang menghasilkan error yang tidak membocorkan path internal ke client.
- [ ] Schema atau scope package yang tidak cocok ditolak.
- [ ] Reason dan audit tersimpan sesuai kontrak.
- [ ] Postman dan Insomnia memverifikasi semantics yang sama.

### Acceptance criteria Opsi B

- [ ] ADR menetapkan threat model, maintenance mode, dan recovery sequence.
- [ ] Hanya System Admin yang dapat memulai restore.
- [ ] Reason non-kosong wajib.
- [ ] Sistem membuat recovery point sebelum mengganti database.
- [ ] Sistem memverifikasi checksum, schema, foreign key, dan scope.
- [ ] Mutation baru ditolak selama maintenance.
- [ ] File replacement atomik atau memiliki recovery protocol yang setara.
- [ ] Health check dijalankan pada database hasil restore.
- [ ] Kegagalan mengembalikan database awal.
- [ ] Restart atau crash pada setiap fase dapat dipulihkan melalui runbook.
- [ ] Audit mencatat requested, started, succeeded, atau failed tanpa secret.
- [ ] Integration test berjalan pada direktori temporary.

### Test matrix bersama

| Skenario | Hasil yang diharapkan |
|---|---|
| Backup valid | Artifact dan checksum cocok |
| Checksum berubah | `422 CHECKSUM_MISMATCH` |
| Artifact hilang | `404` generik tanpa path internal |
| Path di luar storage backup | Ditolak |
| File bukan SQLite | Ditolak |
| Schema version tidak kompatibel | Ditolak |
| Scope class tidak cocok | Ditolak |
| Non-admin memanggil endpoint | `403 FORBIDDEN` |
| Audit dipaksa gagal | Status metadata rollback |
| Request diulang | Hasil idempoten sesuai operation state |

### Test tambahan Opsi B

| Skenario | Hasil yang diharapkan |
|---|---|
| Mutation saat maintenance | Ditolak sementara |
| Failure sebelum file replacement | Database aktif tidak berubah |
| Failure setelah replacement | Recovery point dipulihkan |
| Database hasil restore gagal dibuka | Rollback ke database awal |
| Health check gagal | Rollback dan status FAILED |
| Process restart saat RESTORING | Recovery mengikuti journal atau runbook |

### Definition of Done

- ADR accepted menetapkan verify-only atau full restore.
- API, functional requirements, data model, frontend guide, dan collection tidak saling bertentangan.
- Response tidak melebih-lebihkan efek operasi.
- Path, checksum, schema, scope, authorization, dan audit memiliki test.
- Opsi B tidak dianggap selesai tanpa recovery drill.
- Test, vet, build, dan diff check lulus.

### Prompt implementasi

```text
Kerjakan ticket BE-011 Finalisasi Semantics Backup dan Restore pada repository bot-wa-jadwal.

Baca AGENTS.md, docs/api/API_V1.md bagian backup dan restore, docs/product/FUNCTIONAL_REQUIREMENTS.md bagian FR-OPS-002, docs/product/BUSINESS_RULES.md bagian operasional, docs/product/DATA_MODEL.md bagian backup_records, docs/DEPLOYMENT.md, serta ADR yang relevan.

Jangan langsung mengimplementasikan penggantian database. Ada konflik kontrak: API_V1.md saat ini mendefinisikan POST /api/v1/restores sebagai verifikasi file dan checksum, sedangkan FR-OPS-002 membahas restore penuh dengan recovery. Buat atau perbarui ADR lebih dahulu dan nyatakan opsi yang dipilih.

Rekomendasi untuk v1 adalah verify-only:
1. Pertahankan POST /api/v1/restores sebagai verifikasi artifact.
2. Verifikasi checksum, format SQLite, schema compatibility, foreign key, dan scope metadata.
3. Pastikan artifact path berada di storage backup yang dikonfigurasi dan tidak dapat keluar melalui traversal atau symlink.
4. Ubah status menjadi VERIFIED dalam transaksi dengan audit BE-009.
5. Response wajib menyatakan restore_performed=false.
6. Jangan mengembalikan path filesystem internal kepada client.
7. Sinkronkan API_V1, functional requirements, DOKUMENTASI_API_FRONTEND, Postman, Insomnia, dan label frontend agar semuanya menyebut verifikasi backup.
8. Buat ticket terpisah untuk full restore jika masih dibutuhkan.

Jika pemilik produk memilih full restore, implementasikan maintenance mode, recovery point otomatis, checksum dan schema validation, penghentian mutation baru, atomic replacement, reopen dan health check, rollback, operation journal, audit state, serta runbook. Semua test wajib memakai direktori dan database temporary. Jangan pernah menguji terhadap storage produksi.

Tambahkan test untuk checksum mismatch, artifact hilang, path escape, file non-SQLite, schema incompatible, scope mismatch, non-admin, audit rollback, dan idempotent repeat. Untuk full restore, tambahkan failure injection pada setiap fase dan recovery drill.

Jalankan gofmt, go test -count=1 ./..., go vet ./..., go build -o bin/bot ./cmd/bot, dan git diff --check. Laporkan ADR yang dipilih, semantics endpoint final, file yang berubah, bukti test, dan risiko operasional tersisa. Jangan commit atau push kecuali diminta.
```

---

## Gate penyelesaian empat ticket

Keempat ticket dinyatakan selesai jika:

- [ ] Pattern versioning aman terhadap stale update, concurrency, dan rollback.
- [ ] Task lifecycle dan review mempunyai state transition yang konsisten per version.
- [ ] Notification retry hanya menjadwalkan ulang dan worker menjadi satu-satunya pembuat attempt.
- [ ] Backup verification dan full restore tidak lagi memakai istilah atau status yang ambigu.
- [ ] Semua mutation penting memakai audit writer BE-009.
- [ ] Semua notifikasi memakai durable outbox BE-005.
- [ ] Semua conflict check memakai engine BE-006.
- [ ] Tidak ada akses lintas class atau offering.
- [ ] Dokumentasi API dan collection pengujian selaras.
- [ ] `go test -count=1 ./...` lulus.
- [ ] `go vet ./...` lulus.
- [ ] `go build -o bin/bot ./cmd/bot` lulus.
- [ ] `git diff --check` lulus.
- [ ] Test race dijalankan pada environment dengan CGO aktif, atau keterbatasannya dicatat.
