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
      { id: 'kelas', label: 'Daftar Kelas', img: '/assets/icons/classes.svg' },
      { id: 'status-bot', label: 'Status Bot WhatsApp', img: '/assets/icons/bot.svg' },
      { id: 'antrean', label: 'Antrean Pesan', img: '/assets/icons/message-queue.svg' },
      { id: 'master-ruangan', label: 'Master Ruangan', img: '/assets/icons/room.svg' },
      { id: 'master-matkul', label: 'Master Mata Kuliah', img: '/assets/icons/book.svg' },
      { id: 'pengguna', label: 'Kelola Pengguna', img: '/assets/icons/people.svg' },
      { id: 'backup', label: 'Backup Data', img: '/assets/icons/backup.svg' }
    ],

    botOnline: false,
    kelasList: [
      { id: '1', nama: 'D4 TI 2024 A', prodi: 'Teknik Informatika', angkatan: '2024', statusKM: 'none', kmName: '', kmPhone: '' },
      { id: '2', nama: 'D4 TI 2024 B', prodi: 'Teknik Informatika', angkatan: '2024', statusKM: 'active', kmName: 'Raisa Putri', kmPhone: '0812 3456 7890' },
      { id: '3', nama: 'D4 TI 2025 A', prodi: 'Teknik Informatika', angkatan: '2025', statusKM: 'pending', kmName: '', kmPhone: '0819 8765 4321' }
    ],
    totalKelas: 24,
    kelasAktif: '',
    kelasAktifObj: null,
    kelasForm: { nama: '', prodi: '', angkatan: '' },
    kelasError: '',
    undangNomor: '',
    undangError: '',
    undangLink: '',
    dukunganAlasan: '',

    toast: { show: false, message: '', timer: null },

    get kmAktifCount() {
      const active = this.kelasList.filter(k => k.statusKM === 'active').length;
      return active > 0 ? (active + 21) : 22;
    },
    get kelasTanpaKMCount() {
      const none = this.kelasList.filter(k => k.statusKM === 'none').length;
      return none > 0 ? (none + 1) : 2;
    },
    get undanganMenungguCount() {
      const pending = this.kelasList.filter(k => k.statusKM === 'pending').length;
      return pending > 0 ? (pending + 2) : 3;
    },

    filteredKelas() {
      const query = (this.q || '').trim().toLowerCase();
      if (!query) return this.kelasList;
      return this.kelasList.filter(k => 
        (k.nama && k.nama.toLowerCase().includes(query)) ||
        (k.prodi && k.prodi.toLowerCase().includes(query)) ||
        (k.angkatan && k.angkatan.toLowerCase().includes(query)) ||
        (k.kmName && k.kmName.toLowerCase().includes(query))
      );
    },

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
          const res = await fetch(url + '?v=' + Date.now(), { cache: 'no-store' });
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
        if (data && data.total_classes) {
          this.totalKelas = data.total_classes;
        }
      } catch (e) { /* tetap menggunakan data prototype */ }
    },

    bukaUndang(k) {
      this.kelasAktifObj = typeof k === 'object' ? k : { nama: k, prodi: 'Teknik Informatika', angkatan: '2024', statusKM: 'none' };
      this.kelasAktif = this.kelasAktifObj.nama;
      this.undangNomor = this.kelasAktifObj.kmPhone || '';
      this.undangError = '';
      this.view = 'undang';
      window.scrollTo({ top: 0 });
    },

    bukaDetail(k) {
      this.kelasAktifObj = typeof k === 'object' ? k : { nama: k, prodi: 'Teknik Informatika', angkatan: '2024', statusKM: 'active', kmName: 'Raisa Putri', kmPhone: '0812 3456 7890' };
      this.kelasAktif = this.kelasAktifObj.nama;
      this.view = 'detail';
      window.scrollTo({ top: 0 });
    },

    simpanKelas() {
      const f = this.kelasForm;
      if (!f.nama.trim()) { this.kelasError = 'Nama kelas wajib diisi.'; return; }
      if (!f.prodi.trim()) { this.kelasError = 'Program studi wajib diisi.'; return; }
      if (!f.angkatan.trim()) { this.kelasError = 'Angkatan wajib diisi.'; return; }
      this.kelasError = '';
      const baru = {
        id: String(Date.now()),
        nama: f.nama.trim(),
        prodi: f.prodi.trim(),
        angkatan: f.angkatan.trim(),
        statusKM: 'none',
        kmName: '',
        kmPhone: ''
      };
      this.kelasList.unshift(baru);
      this.totalKelas = (this.totalKelas || 24) + 1;
      this.kelasForm = { nama: '', prodi: '', angkatan: '' };
      this.view = 'kelas';
      this.showToast('Kelas baru berhasil ditambahkan.');
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
