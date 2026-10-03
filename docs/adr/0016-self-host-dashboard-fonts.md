# ADR-0016 — Font Dashboard dari Origin Sendiri

- Status: Accepted
- Tanggal: 2026-10-03
- Konteks: ADR-0015 masih membiarkan Google Fonts dari CDN. Dashboard kini perlu memuat font dan ikon secara konsisten ketika layanan eksternal tidak tersedia.
- Keputusan: Poppins, Montserrat, JetBrains Mono, dan subset Material Symbols disimpan di `web/assets/fonts/`, dimuat melalui `web/css/fonts.css`, dan disematkan oleh `web/embed.go`. Keputusan ini menggantikan bagian Google Fonts dalam ADR-0015; keputusan vendor Tailwind dan Alpine tetap berlaku.
- Konsekuensi: Berkas font serta lisensi menjadi bagian biner Go. Daftar ikon dan asal aset dicatat di `web/assets/fonts/README.md`; perubahan ikon memerlukan pembaruan subset dan dokumentasinya.
