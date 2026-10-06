# PEDOMAN ANTI-SLOP UI/UX (BOT JADWAL v2.0)
> **Standar Kurasi Visual & Pencegahan Klise AI Generator untuk Seluruh Antarmuka Sistem**  
> *Rujukan Resmi Desain: Menghentikan kebiasaan buruk AI (Rainbow Slop, Container Inception, Duplikasi Informasi, dsb.) agar antarmuka tetap bersih, berwibawa, dan berstandar enterprise akademik.*

---

## 1. Prinsip Utama Anti-Slop (/antislop-ui)

> *"Kepadatan informasi (information density) yang baik lahir dari **keteraturan tipografi, spasi (whitespace), dan garis kisi yang presisi**, BUKAN dari menumpuk banyak kotak berlatar warna-warni."*

1. **Monokromatik Tenang, Warna Hanyalah Aksen**:
   - Background canvas dan kartu utama harus bersih (`#FFFFFF` dan `#F2F5FA`).
   - Warna aksen (biru `#3965FB`, hijau `#15803D`, merah `#D92D20`, amber `#B45309`) HANYA digunakan pada status pill kecil, teks penting, atau tombol aksi utama. Dilarang mewarnai seluruh background baris tabel dengan warna pelangi.
2. **Satu Level Kedalaman (Flattened Hierarchy)**:
   - Hindari membuat kotak di dalam kotak di dalam kotak. Cukup gunakan garis pembatas `border-b border-[#F2F4F7]` atau `divide-y`.
3. **Tipografi Menggantikan Kontainer**:
   - Gunakan ukuran teks, ketebalan font (*font weight*), dan warna kontras untuk membedakan judul, subjudul, dan metadata. Jangan setiap teks diberi kotak pembungkus sendiri.

---

## 2. Katalog 9 Pola AI Slop & Solusi Perbaikannya

Berikut adalah 9 fenomena AI slop yang diidentifikasi dari sesi desain nyata beserta solusi wajibnya:

| No | Nama Pola AI Slop | Karakteristik Buruk (Gejala) | Dampak Negatif | Solusi Wajib Antislop |
|:---:|---|---|---|---|
| **1** | **Container Inception (Nesting Hell)** | Membungkus komponen ke dalam kotak, di dalam kotak, di dalam kotak lagi tanpa alasan hierarki yang jelas. | Halaman terlihat sesak, vertikal height membengkak, mata cepat lelah. | **Hapus kontainer berlapis.** Gabungkan teks rincian/alasan langsung ke dalam kolom tabel atau baris datar dengan `border-b` halus. |
| **2** | **Rainbow UI Slop (Multi-Pastel Overload)** | Menggunakan terlalu banyak warna pastel cerah sekaligus pada satu tampilan (kuning, biru, merah, ungu bertumpuk). | UI terlihat seperti aplikasi mainan anak-anak, bukan sistem administrasi kampus resmi. | **Batasi warna ke status badge kecil.** Latar belakang baris tetap netral monokromatik (putih atau `#F8FAFC`). |
| **3** | **Checklist Clutter (Duplikasi Informasi)** | Menambahkan kotak petunjuk/checklist panjang di bawah form yang hanya mengulang teks yang sudah tertulis di atasnya. | Menghabiskan 30–40% tinggi modal hanya untuk mengulang hal yang sama berkali-kali. | **Hapus kotak checklist redundan.** Cukup letakkan helper text ringkas 1 baris (`Min. 12 karakter`) tepat di atas/bawah kolom input. |
| **4** | **Gratuitous Emoji UI Slop** | Menempelkan emoji teks sembarangan pada tombol/filter (`Kritis ⚠️`, `Dukungan 🛡️`, `Hari Ini 📅`). | Mengesankan template murahan AI, mengorbankan kesan profesionalitas akademis. | **Gunakan teks murni yang bersih** (`Kritis`, `Dukungan`, `Hari Ini`) atau gunakan ikon SVG vektor resmi dengan resolusi tajam. |
| **5** | **Redundant Status Sub-Labels** | Menambahkan sub-keterangan status centang ganda yang bertubrukan dengan kolom status di sebelahnya. | Duplikasi informasi visual yang membingungkan dan tidak berdasar pada kolom backend. | **Hapus sub-label ganda.** Kolom tanggal cukup berisi jam/tanggal murni (`sent_at`), status keberhasilan cukup diwakili kolom `STATUS`. |
| **6** | **Debug Dump Representation** | Menampilkan data komparasi seperti output terminal kasar programmer (`1 -> 3 sesi`, hash terpotong di rounded box). | Sulit dibaca manusia awam, merusak estetika desain antarmuka modern. | Ubah menjadi **Tabel 3 Kolom Ramping** (*Komponen \| Data Cadangan \| Data Aktif*) atau tabel diff ala inspeksi Git yang rapi. |
| **7** | **Banner Hijacking (Feature Dominance)** | Membuat banner fitur darurat/insidental dengan ukuran raksasa yang memakan 30% layar dan memotong alur halaman. | Mengaburkan hierarki visual halaman utama dan mendominasi konten harian. | **Rampingkan fitur darurat.** Pindahkan menjadi tombol aksi kompak di header kanan atas atau inline alert strip 1 baris. |
| **8** | **Broken Font Ligature ("Huruf T" Slop)** | Memasukkan nama ikon webfont teks yang jika font-nya gagal termuat atau terpotong, akan merender karakter teks mentahnya (misal `toggle_off` terpotong jadi `T`). | Pengguna bingung melihat huruf teks/glif misterius di samping tombol aksi. | **Gunakan Inline SVG mandiri.** SVG vektor tidak bergantung pada webfont eksternal dan anti-rusak. |
| **9** | **Redundansi Aksi / Tombol Ganda** | Menaruh dua kontrol pembatalan sekaligus di layar yang sama tanpa alur UX yang disengaja (misal `✕` di header dan `Batal` di bawah pada form kecil). | Membingungkan hierarki penyelesaian tindakan. | Untuk dialog formulir input pendek, **fokuskan alur ke tombol aksi footer** (`[ Batal ]` berdampingan dengan `[ Simpan ]`). |

---

## 3. Checklist Kurasi Desain (Pre-Approval Checklist)

Sebelum menyetujui (*approve*) frame desain baru atau kode HTML hasil ekspor, periksa checklist berikut:

- [ ] **Bebas Container Inception**: Apakah ada kartu di dalam kartu yang sebenarnya bisa digabung atau dipisahkan dengan garis border tipis saja?
- [ ] **Bebas Efek Pelangi**: Apakah warna latar belakang didominasi putih/netral, dan warna aksen hanya ada pada badge kecil?
- [ ] **Bebas Duplikasi Teks**: Apakah ada instruksi formulir yang diulang lebih dari 1 kali di layar yang sama?
- [ ] **Bebas Emoji Liar**: Apakah tombol, filter, dan chip menggunakan teks murni atau SVG resmi, bukan emoji teks sistem?
- [ ] **Bebas Ikon Teks Mentah**: Apakah seluruh ikon menggunakan SVG langsung atau font Material Symbols yang terbukti memiliki fallback aman?
- [ ] **Struktur Komparasi Rapi**: Apakah data perbandingan (diff, audit, restore) disajikan dalam format tabel kisi yang jelas, bukan teks mentah bertumpuk?
- [ ] **Ergonomi Aksi Jelas**: Apakah tombol primer, sekunder, dan destruktif memiliki pembeda kontras warna yang tegas sesuai standar WCAG AA?
