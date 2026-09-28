/**
 * web/js/app-sa.js — Area Superadmin (REVISI Figma, tanpa role switcher).
 * Data asli: /api/classes, /api/status. Sisanya draf lokal + empty state
 * sampai endpoint backend (undangan, pengguna, audit, backup, master) ada.
 */
function saApp() {
  return {
    view: 'dashboard',
    drawer: false,
    q: '',

    saNav: [
      { id: 'dashboard', label: 'Dashboard', img: '/assets/icons/home.svg' },
      { id: 'kelas', label: 'Daftar Kelas', img: '/assets/icons/ext-settings-edit.svg' },
      { id: 'status-bot', label: 'Status Bot WhatsApp', img: '/assets/icons/activity.svg' },
      { id: 'antrean', label: 'Antrean Pesan', img: '/assets/icons/bell.svg' },
      { id: 'master-ruangan', label: 'Master Ruangan', img: '/assets/icons/calendar.svg' },
      { id: 'master-matkul', label: 'Master Mata Kuliah', img: '/assets/icons/book.svg' },
      { id: 'pengguna', label: 'Kelola Pengguna', img: '/assets/icons/event.svg' },
      { id: 'backup', label: 'Backup Data', img: '/assets/icons/folder.svg' }
    ],

    botOnline: false,
    kelasList: [],
    totalKelas: 0,
    kelasAktif: '',
    kelasForm: { nama: '', prodi: '', angkatan: '' },
    kelasError: '',
    undangNomor: '',
    undangError: '',
    undangLink: '',
    dukunganAlasan: '',

    toast: { show: false, message: '', timer: null },

    saTitle(id) {
      const item = this.saNav.find(n => n.id === id);
      return item ? item.label : id;
    },

    async initSA() {
      await this.loadPartials([
        ['sa-sidebar', '/partials/sa/sidebar.html'],
        ['sa-topbar', '/partials/sa/topbar.html'],
        ['sa-dashboard', '/partials/sa/view-dashboard.html'],
        ['sa-kelas', '/partials/sa/view-kelas.html'],
        ['sa-undang', '/partials/sa/view-undang.html'],
        ['sa-dukungan', '/partials/sa/view-dukungan.html'],
        ['sa-soon', '/partials/sa/view-soon.html'],
        ['sa-drawer', '/partials/sa/drawer.html'],
        ['sa-toast', '/partials/sa/toast.html']
      ]);
      await this.checkBot();
      await this.loadKelas();
      setInterval(() => this.checkBot(), 30000);
    },

    async loadPartials(slots) {
      await Promise.all(slots.map(async ([id, url]) => {
        try {
          const res = await fetch(url + '?v=20260927b', { cache: 'no-store' });
          if (!res.ok) throw new Error(`HTTP ${res.status}`);
          const el = document.getElementById(id);
          if (el) {
            el.innerHTML = await res.text();
            if (window.Alpine && window.Alpine.initTree) window.Alpine.initTree(el);
          }
        } catch (err) {
          console.error(`Gagal memuat ${url}:`, err);
        }
      }));
    },

    go(v) {
      this.view = v;
      this.drawer = false;
      window.scrollTo({ top: 0 });
    },

    soon(fitur) { this.showToast(`${fitur}: fitur belum tersedia.`); },

    async checkBot() {
      try {
        const st = await API.getStatus();
        if (st) this.botOnline = String(st.bot_connection || '').toLowerCase() === 'connected';
      } catch (e) { this.botOnline = false; }
    },

    async loadKelas() {
      try {
        const data = await API.getClasses();
        if (data) {
          this.kelasList = data.classes || [];
          this.totalKelas = data.total_classes || this.kelasList.length;
        }
      } catch (e) { /* kosong */ }
    },

    bukaUndang(k) { this.kelasAktif = k; this.undangNomor = ''; this.undangError = ''; this.view = 'undang'; window.scrollTo({ top: 0 }); },
    bukaDetail(k) { this.kelasAktif = k; this.view = 'detail'; window.scrollTo({ top: 0 }); },

    simpanKelas() {
      const f = this.kelasForm;
      if (!f.nama.trim()) { this.kelasError = 'Nama kelas wajib diisi.'; return; }
      if (!f.prodi.trim()) { this.kelasError = 'Program studi wajib diisi.'; return; }
      if (!f.angkatan.trim()) { this.kelasError = 'Angkatan wajib diisi.'; return; }
      this.kelasError = '';
      if (!this.kelasList.includes(f.nama.trim())) this.kelasList.push(f.nama.trim());
      this.totalKelas = this.kelasList.length;
      this.kelasForm = { nama: '', prodi: '', angkatan: '' };
      this.view = 'kelas';
      this.showToast('Draf kelas tersimpan lokal — penyimpanan permanen butuh endpoint backend.');
    },

    buatUndangKM() {
      if (!this.undangNomor || this.undangNomor.replace(/\D/g, '').length < 9) {
        this.undangError = 'Nomor WhatsApp calon KM tidak valid.';
        return;
      }
      this.undangError = '';
      this.undangLink = `https://bot-jadwal/undang/km?kelas=${encodeURIComponent(this.kelasAktif)}&wa=${encodeURIComponent(this.undangNomor)}`;
      this.view = 'undang-siap';
      window.scrollTo({ top: 0 });
    },

    masukDukungan() {
      if (!this.kelasAktif) { this.showToast('Pilih kelas tujuan dulu.'); return; }
      if (!this.dukunganAlasan.trim()) { this.showToast('Alasan dukungan wajib diisi.'); return; }
      this.showToast('Mode dukungan penuh butuh endpoint autentikasi backend.');
    },

    copyText(text, okMsg) {
      const done = () => this.showToast(okMsg || 'Tersalin.');
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done).catch(() => this.showToast('Gagal menyalin otomatis.'));
      } else {
        this.showToast('Clipboard tidak tersedia di browser ini.');
      }
    },

    showToast(msg) {
      if (this.toast.timer) clearTimeout(this.toast.timer);
      this.toast.message = msg;
      this.toast.show = true;
      this.toast.timer = setTimeout(() => { this.toast.show = false; }, 3000);
    }
  };
}
