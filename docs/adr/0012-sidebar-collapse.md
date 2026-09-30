# ADR-0012 — Sidebar Ciut via Logo Asterisk

- Status: Accepted
- Tanggal: 2026-10-01
- Konteks: Sidebar desktop 248px (`portal`, `common` untuk KM/PJ, `system-admin`) memakan ruang konten. Permintaan: klik logo asterisk menciutkan hingga tersisa ikon. Alternatif: tombol chevron terpisah, tombol di topbar, sidebar hilang total.
- Keputusan: (1) Logo jadi `<button>` toggle (`toggleSidebar()`, `aria-expanded`, `aria-label`, `title`). (2) Mode ciut 76px icon-only: label/judul seksi/teks footer hidden via `.sidebar-collapsed`, ikon tetap + tooltip `title`, logo tampil bintang saja via container `overflow-hidden` 32px. (3) Footer sisakan ikon logout/lock. (4) Status disimpan `localStorage asterisk:sidebar:collapsed`, dibaca di `init*()`. (5) Berlaku 4 area, mobile (`drawer`/`bottombar`) tidak berubah. Transisi `transition-all duration-200`, padding konten `md:pl-[248px]` ↔ `md:pl-[76px]`.
- Konsekuensi: Satu pola Alpine + CSS di `style.css` untuk 3 partial. Perlu `sidebarCollapsed` di tiap `*App()`. Uji: klik logo 2x bolak-balik, reload ingat status, keyboard Enter/Space jalan, tooltip muncul saat ciut.
