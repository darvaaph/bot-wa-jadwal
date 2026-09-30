/**
 * web/js/app-system-admin.js — Pengendali Area System Admin
 * Terhubung langsung ke REST API v1: /api/v1/admin/status, /api/v1/classes,
 * /api/v1/invitations, dan /api/v1/auth/*.
 */
function systemAdminApp() {
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

    currentUser: null,
    authModal: false,
    authForm: { identityKey: '', password: '' },
    authLoading: false,
    authError: '',

    botOnline: false,
    botStatusDetails: null,

    kelasList: [],
    totalKelas: 0,
    activeKelasCount: 0,
    kelasLoading: false,

    kelasAktif: '',
    kelasAktifObj: null,
    kelasForm: { nama: '', prodi: 'Teknik Informatika', angkatan: '2025' },
    kelasError: '',
    kelasSaving: false,

    undangNomor: '',
    undangError: '',
    undangLoading: false,
    undangLink: '',

    dukunganAlasan: '',

    notifFilter: '',
    notifList: [],
    notifLoading: false,
    notifError: '',

    backupForm: { kelas: '', alasan: '' },
    backupError: '',
    backupHasil: null,
    backupLoading: false,
    restoreForm: { id: '', alasan: '', paham: false },
    restoreError: '',
    restoreLoading: false,

    auditList: [],
    auditLoading: false,
    auditError: '',
    auditKelas: '',

    toast: { show: false, message: '', timer: null },

    get kmAktifCount() {
      return this.kelasList.filter(k => k.statusKM === 'active').length;
    },
    get kelasTanpaKMCount() {
      return this.kelasList.filter(k => k.statusKM === 'none').length;
    },
    get undanganMenungguCount() {
      return this.kelasList.filter(k => k.statusKM === 'pending').length;
    },

    getBotStatusLabel() {
      const st = (this.botStatusDetails && this.botStatusDetails.bot_connection) || '';
      if (this.botOnline || st === 'connected') return 'Terhubung (Aktif)';
      if (st === 'waiting_qr') return 'Menunggu Pindai QR di Terminal';
      if (st === 'reconnecting') return 'Menyambung Ulang ke WhatsApp...';
      if (st === 'uninitialized') return 'Mode Web-Only (WhatsApp Tidak Aktif)';
      return st ? `Status: ${st}` : 'Terputus';
    },

    getBotStatusColor() {
      const st = (this.botStatusDetails && this.botStatusDetails.bot_connection) || '';
      if (this.botOnline || st === 'connected') return 'bg-emerald-500 animate-pulse';
      if (st === 'waiting_qr' || st === 'reconnecting') return 'bg-amber-500 animate-pulse';
      return 'bg-[#FF6C48]';
    },

    filteredKelas() {
      const query = (this.q || '').trim().toLowerCase();
      if (!query) return this.kelasList;
      return this.kelasList.filter(k =>
        (k.nama && k.nama.toLowerCase().includes(query)) ||
        (k.prodi && k.prodi.toLowerCase().includes(query)) ||
        (k.angkatan && k.angkatan.toLowerCase().includes(query)) ||
        (k.kmName && k.kmName.toLowerCase().includes(query)) ||
        (k.kmPhone && k.kmPhone.toLowerCase().includes(query))
      );
    },

    saTitle(id) {
      const item = this.saNav.find(n => n.id === id);
      return item ? item.label : id;
    },

    async initSystemAdmin() {
      const isAuthed = await this.checkAuth();
      if (!isAuthed) {
        localStorage.removeItem('access_token');
        window.location.replace('/login.html?role=sa');
        return;
      }

      await this.loadPartials([
        ['sa-sidebar', '/partials/system-admin/sidebar.html'],
        ['sa-topbar', '/partials/system-admin/topbar.html'],
        ['sa-dashboard', '/partials/system-admin/view-dashboard.html'],
        ['sa-kelas', '/partials/system-admin/view-kelas.html'],
        ['sa-undang', '/partials/system-admin/view-undang.html'],
        ['sa-dukungan', '/partials/system-admin/view-dukungan.html'],
        ['sa-antrean', '/partials/system-admin/view-antrean.html'],
        ['sa-backup', '/partials/system-admin/view-backup.html'],
        ['sa-soon', '/partials/system-admin/view-soon.html'],
        ['sa-drawer', '/partials/system-admin/drawer.html'],
        ['sa-toast', '/partials/system-admin/toast.html'],
        ['sa-auth', '/partials/system-admin/auth-modal.html']
      ]);

      await Promise.all([this.checkBot(), this.loadKelas()]);

      setInterval(() => {
        if (this.currentUser) this.checkBot();
      }, 30000);
    },

    async loadPartials(slots) {
      // Alpine v3 auto-init node baru via MutationObserver; jangan initTree manual (render ganda).
      await Promise.all(slots.map(async ([id, url]) => {
        try {
          const res = await fetch(url + '?v=20261008', { cache: 'no-store' });
          if (!res.ok) throw new Error(`HTTP ${res.status}`);
          const el = document.getElementById(id);
          if (el) {
            el.innerHTML = await res.text();
          }
        } catch (err) {
          console.error(`Gagal memuat partial ${url}:`, err);
        }
      }));
    },

    async checkAuth() {
      try {
        const token = API.getAuthToken ? API.getAuthToken() : localStorage.getItem('access_token');
        if (!token) return false;
        const me = await API.getMe();
        if (me && me.user) {
          const role = me.active_assignment && me.active_assignment.role;
          if (role && role !== 'SYSTEM_ADMIN') return false;
          this.currentUser = me.user;
          this.activeRole = role || null;
          return true;
        }
      } catch (e) {}
      return false;
    },

    async login() {
      if (!this.authForm.identityKey.trim() || !this.authForm.password.trim()) {
        this.authError = 'Nomor identitas dan kata sandi wajib diisi.';
        return;
      }
      this.authLoading = true;
      this.authError = '';
      try {
        const res = await API.login(this.authForm.identityKey.trim(), this.authForm.password.trim());
        if (res && res.token) {
          this.authModal = false;
          await this.checkAuth();
          await Promise.all([this.checkBot(), this.loadKelas()]);
          this.showToast('Berhasil masuk sebagai System Admin.');
        }
      } catch (err) {
        this.authError = err.message || 'Gagal masuk. Periksa kembali kredensial Anda.';
      } finally {
        this.authLoading = false;
      }
    },

    async logout() {
      try {
        await API.logout();
      } catch (e) {}
      localStorage.removeItem('access_token');
      localStorage.removeItem('token');
      this.currentUser = null;
      this.kelasList = [];
      this.totalKelas = 0;
      this.showToast('Sesi telah ditutup. Mengarahkan ke login...');
      setTimeout(() => {
        window.location.href = '/login.html';
      }, 500);
    },

    statusNotifLabel(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'PENDING') return 'Menunggu';
      if (s === 'PROCESSING') return 'Diproses';
      if (s === 'SENT') return 'Terkirim';
      if (s === 'FAILED') return 'Gagal';
      return st || '-';
    },

    fmtWaktuID(iso) {
      try {
        const d = new Date(iso);
        if (isNaN(d)) return String(iso || '-');
        return d.toLocaleString('id-ID', { timeZone: 'Asia/Jakarta', day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false }) + ' WIB';
      } catch (e) { return String(iso || '-'); }
    },

    async loadAntrean() {
      this.notifLoading = true; this.notifError = '';
      try {
        const params = { limit: 50 };
        if (this.notifFilter) params.status = this.notifFilter;
        this.notifList = await API.getNotifications(this.notifFilter || '', 50).catch(() => null) || [];
      } catch (e) {
        this.notifList = [];
        this.notifError = 'Antrean pesan belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.notifLoading = false;
      }
    },

    async ulangiPesan(id) {
      try {
        await API.retryNotification(id);
        this.showToast('Pengiriman ulang dijadwalkan.');
        await this.loadAntrean();
      } catch (err) {
        this.showToast(err.message || 'Gagal menjadwalkan ulang.');
      }
    },

    async loadAudit() {
      this.auditLoading = true; this.auditError = '';
      try {
        const params = { limit: 50 };
        if (this.auditKelas) params.class_slug = this.auditKelas;
        this.auditList = await API.getAudit(params).catch(() => null) || [];
      } catch (e) {
        this.auditList = [];
        this.auditError = 'Riwayat belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.auditLoading = false;
      }
    },

    async buatBackup() {
      const f = this.backupForm;
      if (!f.kelas) { this.backupError = 'Pilih kelas untuk dicadangkan.'; return; }
      this.backupError = ''; this.backupHasil = null;
      this.backupLoading = true;
      try {
        const payload = { class_slug: f.kelas };
        if ((f.alasan || '').trim()) payload.reason = f.alasan.trim();
        this.backupHasil = await API.createBackup(payload);
        this.showToast('Cadangan berhasil dibuat.');
      } catch (err) {
        this.backupError = err.message || 'Gagal membuat cadangan.';
      } finally {
        this.backupLoading = false;
      }
    },

    async pulihkanBackup() {
      const f = this.restoreForm;
      if (!f.id) { this.restoreError = 'Isi ID cadangan yang akan dipulihkan.'; return; }
      if (!((f.alasan || '').trim())) { this.restoreError = 'Isi alasan pemulihan.'; return; }
      if (!f.paham) { this.restoreError = 'Centang pernyataan pemahaman dampak terlebih dahulu.'; return; }
      this.restoreError = '';
      this.restoreLoading = true;
      try {
        await API.restoreBackup(f.id, f.alasan.trim());
        this.showToast('Pemulihan diverifikasi server.');
        this.restoreForm = { id: '', alasan: '', paham: false };
      } catch (err) {
        this.restoreError = err.message || 'Gagal memulihkan cadangan.';
      } finally {
        this.restoreLoading = false;
      }
    },

    alasanModul(id) {
      if (id === 'master-ruangan') return 'Daftar dan pengelolaan ruangan kampus belum tersedia di API. Hubungi pengembang backend untuk endpoint master ruangan.';
      if (id === 'master-matkul') return 'Daftar dan pengelolaan mata kuliah kampus belum tersedia di API. Hubungi pengembang backend untuk endpoint master mata kuliah.';
      if (id === 'pengguna') return 'Daftar pengguna belum tersedia di API (aksi tangguhkan/pulihkan sudah ada, tetapi tidak ada endpoint daftar). Hubungi pengembang backend.';
      return 'Modul ini belum tersedia.';
    },

    go(v) {
      this.view = v;
      this.drawer = false;
      if (v === 'antrean') this.loadAntrean();
      if (v === 'backup') { this.backupHasil = null; }
      if (v === 'pembaruan') this.loadAudit();
      window.scrollTo({ top: 0 });
    },

    soon(fitur) {
      this.showToast(`${fitur}: fitur dijadwalkan pada fase berikutnya.`);
    },

    async checkBot() {
      try {
        const st = await API.getAdminStatus();
        if (st) {
          this.botStatusDetails = st;
          this.botOnline = String(st.bot_connection || '').toLowerCase() === 'connected';
          if (st.classes_summary) {
            this.totalKelas = st.classes_summary.total;
            this.activeKelasCount = st.classes_summary.active;
          }
          return;
        }
      } catch (e) {}

      // Fallback ke endpoint status publik jika admin status gagal
      try {
        const stPub = await API.getStatus();
        if (stPub) {
          this.botOnline = String(stPub.bot_connection || '').toLowerCase() === 'connected';
        }
      } catch (e) {
        this.botOnline = false;
      }
    },

    async loadKelas() {
      this.kelasLoading = true;
      try {
        const data = await API.getV1Classes();
        if (data && Array.isArray(data.classes)) {
          this.kelasList = data.classes.map(k => ({
            id: k.slug || k.code,
            slug: k.slug,
            nama: k.code,
            prodi: k.program || 'Teknik Informatika',
            angkatan: String(k.cohort || '2024'),
            group: k.group || 'A',
            status: k.status || 'ACTIVE',
            statusKM: k.status_km || (k.km_name ? 'active' : (k.pending_phone ? 'pending' : 'none')),
            kmName: k.km_name || '',
            kmPhone: k.km_phone || k.pending_phone || ''
          }));
          if (!this.totalKelas) {
            this.totalKelas = this.kelasList.length;
          }
        }
      } catch (err) {
        if (err.message && err.message.includes('401')) {
          this.authModal = true;
        }
      } finally {
        this.kelasLoading = false;
      }
    },

    bukaUndang(k) {
      const obj = typeof k === 'object' ? k : (this.kelasList || []).find(x => x.slug === k || x.nama === k);
      if (!obj) { this.showToast('Kelas tidak ditemukan di daftar.'); return; }
      this.kelasAktifObj = obj;
      this.kelasAktif = obj.slug;
      this.undangNomor = obj.kmPhone || '';
      this.undangError = '';
      this.view = 'undang';
      window.scrollTo({ top: 0 });
    },

    bukaDetail(k) {
      const obj = typeof k === 'object' ? k : (this.kelasList || []).find(x => x.slug === k || x.nama === k);
      if (!obj || !obj.slug) { this.showToast('Kelas tidak ditemukan di daftar.'); return; }
      this.kelasAktifObj = obj;
      this.kelasAktif = obj.slug;
      this.view = 'detail';
      window.scrollTo({ top: 0 });
    },

    async simpanKelas() {
      const f = this.kelasForm;
      if (!f.nama.trim()) { this.kelasError = 'Nama kelas wajib diisi.'; return; }
      if (!f.prodi.trim()) { this.kelasError = 'Program studi wajib diisi.'; return; }
      if (!f.angkatan.trim()) { this.kelasError = 'Angkatan wajib diisi.'; return; }

      this.kelasError = '';
      this.kelasSaving = true;

      try {
        const cohortNum = parseInt(f.angkatan.trim(), 10) || new Date().getFullYear();
        await API.createClass({
          name: f.nama.trim(),
          study_program: f.prodi.trim(),
          cohort_year: cohortNum
        });

        this.kelasForm = { nama: '', prodi: 'Teknik Informatika', angkatan: '2025' };
        await Promise.all([this.loadKelas(), this.checkBot()]);
        this.view = 'kelas';
        this.showToast('Kelas baru berhasil ditambahkan.');
      } catch (err) {
        this.kelasError = err.message || 'Gagal menyimpan kelas baru.';
      } finally {
        this.kelasSaving = false;
      }
    },

    async ubahStatusKelas(targetStatus) {
      if (!this.kelasAktifObj || !this.kelasAktifObj.slug) return;
      try {
        await API.updateClassStatus(this.kelasAktifObj.slug, targetStatus);
        this.kelasAktifObj.status = targetStatus;
        await this.loadKelas();
        this.showToast(`Status kelas ${this.kelasAktifObj.nama} berhasil diubah ke ${targetStatus}.`);
      } catch (err) {
        this.showToast(err.message || 'Gagal memperbarui status kelas.');
      }
    },

    async rotasiKodePortal() {
      if (!this.kelasAktifObj || !this.kelasAktifObj.slug) return;
      try {
        const res = await API.rotatePortalCode(this.kelasAktifObj.slug);
        if (res && res.portal_code) {
          this.showToast(`Kode portal berhasil dirotasi: ${res.portal_code}`);
        } else {
          this.showToast('Kode portal kelas berhasil dirotasi.');
        }
      } catch (err) {
        this.showToast(err.message || 'Gagal merotasi kode portal.');
      }
    },

    async buatUndangKM() {
      const cleanPhone = (this.undangNomor || '').replace(/\D/g, '');
      if (cleanPhone.length < 9) {
        this.undangError = 'Nomor WhatsApp calon KM tidak valid (minimal 9 digit).';
        return;
      }

      this.undangError = '';
      this.undangLoading = true;

      try {
        const slug = this.kelasAktifObj ? (this.kelasAktifObj.slug || this.kelasAktifObj.id) : '';
        const res = await API.createInvitation({
          role: 'KM',
          class_slug: slug,
          invited_identity_key: this.undangNomor.trim()
        });

        const token = (res && res.token) ? res.token : '';
        this.undangLink = `${window.location.origin}/invite.html?token=${encodeURIComponent(token)}`;
        this.view = 'undang-siap';
        await this.loadKelas();
        this.showToast('Tautan undangan KM berhasil dibuat.');
        window.scrollTo({ top: 0 });
      } catch (err) {
        this.undangError = err.message || 'Gagal membuat undangan KM.';
      } finally {
        this.undangLoading = false;
      }
    },

    masukDukungan() {
      const target = (this.kelasList || []).find(k => k.slug === this.kelasAktif);
      if (!target) { this.showToast('Pilih kelas tujuan dulu.'); return; }
      if (!(this.dukunganAlasan || '').trim()) { this.showToast('Alasan dukungan wajib diisi.'); return; }
      this.showToast(`Dukungan untuk ${target.nama} dicatat. Mode masuk via API belum tersedia.`);
    },

    copyText(text, okMsg) {
      const done = () => this.showToast(okMsg || 'Tersalin ke clipboard.');
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done).catch(() => this.showToast('Gagal menyalin otomatis.'));
      } else {
        this.showToast('Clipboard tidak didukung browser ini.');
      }
    },

    async logout() {
      try {
        await API.logout();
      } catch (e) {}
      localStorage.removeItem('access_token');
      localStorage.removeItem('token');
      this.showToast('Berhasil keluar. Mengarahkan ke login...');
      setTimeout(() => {
        window.location.href = '/login.html?role=sa';
      }, 500);
    },

    showToast(msg) {
      if (this.toast.timer) clearTimeout(this.toast.timer);
      this.toast.message = msg;
      this.toast.show = true;
      this.toast.timer = setTimeout(() => { this.toast.show = false; }, 3000);
    }
  };
}
