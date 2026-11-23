# Desain Perbaikan UI/UX Pengingat Absen Kuliah

- **Tanggal:** 2026-11-23
- **Ukuran pekerjaan:** Perubahan kecil-terfokus (frontend UI/JS/HTML dan unit test frontend, tanpa perubahan API/database).
- **Status:** Menunggu persetujuan sebelum masuk ke rencana implementasi (`superpowers:writing-plans`).

## 1. Tujuan dan Alasan

Memperbaiki tampilan, aksesibilitas, kejelasan alur, dan keandalan cache pada kontrol **Pengingat Absen Kuliah** di halaman KM (`Pengaturan Kelas`) serta konsistensi tampilan/kontras di panel `System Admin`. Masalah saat ini membuat tombol toggle tampil sebagai pil abu-abu polos tanpa bulatan status, kolom jeda menit berisiko kosong karena mismatch cache aset `immutable`, label tertekan dalam tinggi tetap `h-[44px]`, dan beberapa warna kontrol gagal standar kontras WCAG AA.

## 2. Lingkup, Non-Goal, dan Anti-Goal

- **Target utama:**
  - Switch `Pengingat Absen Kuliah` berfungsi visual dan semantik (`role="switch"`, bulatan thumb aktif/nonaktif terlihat jelas, fokus keyboard `:focus-visible`, area klik minimal `44x44px`).
  - Layout **Pendekatan 1** (panel terpadu 1 baris penuh tanpa kotak bersarang, hanya pemisah garis tipis `border-t border-[#EEEEEE]`), dengan input jeda angka `[ 25 | menit ]` hanya aktif/ditampilkan saat toggle menyala.
  - Copywriting ringkas dalam bahasa Indonesia alami (tanpa istilah "cek akademik" maupun pengulangan rentang), konsisten memakai format `1-60`.
  - Guard state `settingsLoading` / `settingsError` (kartu pengaturan tidak tampil dengan default palsu saat gagal muat atau sedang memuat), serta indikator perubahan belum disimpan di footer kartu `Waktu Pengingat & Zona`.
  - Kontras pesan error pada form edit pengingat di System Admin ditingkatkan agar lolos WCAG AA >= 4.5:1.
  - Cache-buster `?v=` pada aset JS yang berubah diperbarui agar tidak memuat JS lama dari cache `immutable`.
- **Non-goal:**
  - Tidak mengubah kontrak API backend `/api/v1/classes/{slug}/settings`, skema SQLite, maupun struktur umum card `Ringkasan Pagi`, `Pengingat Sore`, `Pengingat Pengganti`, dan `Zona Waktu`.
  - Tidak menambahkan `npm`, `package.json`, atau build step Tailwind baru (`web/css/tailwind.css` tetap statis pre-compiled sesuai ADR-0017).
- **Anti-goal:**
  - Tidak membungkus komponen dengan kotak berborder di dalam kartu (`Container Inception`), tidak memakai kelas Tailwind dinamis yang belum ter-compile di `web/css/tailwind.css`, dan tidak memakai emoji dekoratif pada label/heading.

## 3. Batasan Teknis

- Seluruh utility class HTML/Alpine harus sudah tersedia di [tailwind.css](file:///f:/proyek-lain/bot-wa-jadwal/web/css/tailwind.css), [tokens.css](file:///f:/proyek-lain/bot-wa-jadwal/web/css/tokens.css), [style.css](file:///f:/proyek-lain/bot-wa-jadwal/web/css/style.css), atau [components.css](file:///f:/proyek-lain/bot-wa-jadwal/web/css/components.css). Kelas `peer`, `peer-checked:*`, dan `after:*` terbukti tidak ada di `tailwind.css` sehingga dilarang dipakai.
- `web/css/components.css` menerapkan `font-size: 16px !important` pada `<input>` di layar `<= 767px`; kontrol angka dan label satuannya tidak boleh memakai teks inline panjang dengan `whitespace-nowrap` yang memicu overflow di lebar `320px-375px`.
- `internal/api/routes.go` menandai aset berparameter `?v=` sebagai `Cache-Control: public, max-age=31536000, immutable`, sedangkan `loadPartials()` mengambil HTML partial dengan `cache: 'no-store'`. Setiap modifikasi `web/js/app-km.js` atau `web/js/app-system-admin.js` wajib menaikkan query `?v=` pada halaman entry-nya.
- Mengikuti aturan `antislop` langsung selama pengerjaan (termasuk kontras non-teks >= 3:1, teks normal >= 4.5:1, dan larangan em dash pada UI/teks baru).

## 4. Pendekatan yang Dipertimbangkan

1. **Pendekatan 1 (Dipilih): Blok terpadu 1 baris penuh dengan pemisah garis tipis**
   - Header blok berisi judul, keterangan status ringkas, dan `<button type="button" role="switch">` di sisi kanan.
   - Saat aktif (`pengaturan.absenAktif === true`), sub-baris di bawahnya menampilkan label `Jeda Pengingat`, deskripsi singkat 1 baris, dan kotak input `w-full sm:w-[170px]` berisi `<input type="number" ... placeholder="25">` dengan satuan ringkas `menit`.
   - **Alasan dipilih:** Memisahkan secara tegas mental model tombol switch vs kolom ketik angka, menghilangkan kolom mati setengah baris saat nonaktif, menjaga hierarki flat sesuai `ANTI_SLOP_GUIDELINES.md`, dan paling lapang pada layar ponsel kecil.
2. **Pendekatan 2: Tetap grid 2 kolom (switch kiri, input menit kanan)**
   - Memperbaiki CSS switch di kolom kiri dan meringkas unit di kolom kanan sambil menjaga struktur grid 2 kolom.
   - **Alasan tidak dipilih:** Menyisakan kotak disabled selebar setengah kartu saat nonaktif dan tetap menampilkan kontrol switch dalam bingkai mirip field teks.
3. **Pendekatan 3 (Ditolak): Menghapus switch dan menyamakan dengan input angka `0 = nonaktif`**
   - Bertentangan dengan kebutuhan produk bahwa pengingat absen bersifat opsional dengan toggle eksplisit dan default aktif `25` menit.

## 5. Keputusan Rancangan yang Sudah Dikunci

- **KM HTML (`web/partials/km/view-pengaturan.html`):**
  - Tambahkan `x-show="!settingsLoading && !settingsError && classSettings"` dengan `x-cloak` pada grid utama pengaturan kelas agar kartu tidak tampil dengan data default palsu saat loading atau gagal muat.
  - Ganti baris `Pengingat Absen Kuliah` menjadi satu blok terpadu selebar kartu dengan pemisah `pt-4 border-t border-[#EEEEEE]` (tanpa nested border box):
    - Switch: `<button type="button" id="km-absen-aktif" role="switch" :aria-checked="String(pengaturan.absenAktif)" ... @click="toggleAbsen()">` dengan area tekan `min-h-[44px] min-w-[44px]` membungkus track visual `w-[40px] h-[24px]` (`bg-[#3965FB]` saat aktif, `bg-[#64748B]` saat nonaktif dengan kontras 4.76:1) dan thumb `w-[20px] h-[20px] bg-white` (`translate-x-4` / `translate-x-0`), serta `focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#3965FB] focus-visible:outline-offset-2`.
    - Copy status & helper:
      - Saat aktif: `"Kirim pengingat presensi ke grup WhatsApp saat kuliah berlangsung."`
      - Saat nonaktif (`text-[#B45309]`): `"Nonaktif. Pengingat presensi otomatis tidak dikirim."`
    - Sub-baris jeda (`x-show="pengaturan.absenAktif"`): label `Jeda Pengingat`, deskripsi `"Dikirim 1-60 menit setelah sesi kuliah dimulai."`, input `#km-absen-menit` (`min="1"`, `max="60"`, `step="1"`, `placeholder="25"`, satuan ringkas `"menit"`).
  - Footer kartu `Waktu Pengingat & Zona` menampilkan penanda `"Ada perubahan belum disimpan."` (`text-[#B45309]`) ketika `pengaturanDirty` bernilai `true`.
- **KM State & Cache (`web/js/app-km.js`, `web/km.html`):**
  - Tambahkan method `toggleAbsen()` yang membalik `pengaturan.absenAktif` lalu memanggil `onToggleAbsen()` agar keyboard/click handler `role="switch"` tetap kompatibel dengan `onToggleAbsen()` di test eksisting.
  - Tambahkan getter `get pengaturanDirty()` yang membandingkan nilai form `pengaturan` saat ini terhadap `classSettings` yang terakhir dimuat.
  - Naikkan `?v=` pada `app-km.js` di `web/km.html` menjadi `?v=20261123`.
- **System Admin (`web/partials/system-admin/view-kelas.html`, `web/system-admin.html`):**
  - Ganti kelas `text-danger` (`#FF6C48`, rasio 2.80:1) pada `#pengingatError` (`role="alert"`) menjadi `text-[#B42318]` (rasio 6.57:1).
  - Rapikan label angka pengingat absen pada form System Admin agar konsisten memakai tanda hubung biasa (`0 = nonaktif, 1-60 menit`).
  - Naikkan `?v=` pada `app-system-admin.js` di `web/system-admin.html` menjadi `?v=20261123` (mengikuti perubahan validasi absen yang sudah ada di branch ini).
- **Verifikasi & Tes (`tests/frontend/remediation.test.cjs`):**
  - Tambahkan test untuk `toggleAbsen()`, `pengaturanDirty` (aktif/nonaktif dan kembali bersih saat dibatalkan), serta pengecekan statis bahwa seluruh kelas pada blok baru di `view-pengaturan.html` dan `view-kelas.html` tersedia di stylesheet repo dan tidak memakai kelas `peer` / `after:*` yang tidak ter-compile.

## 6. Yang Diserahkan ke Implementer

- Penyesuaian detail jarak vertikal (`gap-*`, `pt-*`), `aria-labelledby`/`aria-describedby`, serta struktur exact assertion di test file selama seluruh batasan CSS dan kontrak behavior di atas terpenuhi.
