/**
 * web/js/app-system-admin.js — Pengendali Area System Admin
 * Terhubung langsung ke REST API v1: /api/v1/admin/status, /api/v1/classes,
 * /api/v1/invitations, dan /api/v1/auth/*.
 */
function systemAdminApp() {
  return {
    view: 'dashboard',
    drawer: false,
    sidebarCollapsed: false,
    pageState: null,
    dashboardLoading: true,
    q: '',
    unreadCount: 0,

    ...AsteriskShell.behavior('sa'),

    navSections: [
      { title: 'SISTEM', items: [
        { id: 'dashboard', label: 'Ringkasan Sistem', img: '/assets/icons/home.svg' },
        { id: 'kelas', label: 'Daftar Kelas', img: '/assets/icons/classes.svg', active: ['kelas','buat','detail','undang','undang-siap'] },
        { id: 'pengguna', label: 'Pengguna dan Penugasan', img: '/assets/icons/people.svg' },
        { id: 'dukungan', label: 'Mode Dukungan', img: '/assets/icons/settings.svg' }
      ] },
      { title: 'OPERASIONAL', items: [
        { id: 'antrean', label: 'Antrean Notifikasi', img: '/assets/icons/message-queue.svg' },
        { id: 'kanal', label: 'Kanal WhatsApp', img: '/assets/icons/bell.svg' },
        { id: 'audit', label: 'Audit Global', img: '/assets/icons/activity.svg', active: ['audit','pembaruan'] },
        { id: 'status-bot', label: 'Status Sistem', img: '/assets/icons/bot.svg' }
      ] },
      { title: 'DATA & SISTEM', items: [
        { id: 'master-ruangan', label: 'Master Ruangan', img: '/assets/icons/room.svg' },
        { id: 'master-matkul', label: 'Master Mata Kuliah', img: '/assets/icons/book.svg' },
        { id: 'master-dosen', label: 'Master Dosen', img: '/assets/icons/people.svg' },
        { id: 'backup', label: 'Backup dan Pemulihan', img: '/assets/icons/backup.svg' }
      ] }
    ],

    kelasViews: ['kelas', 'buat', 'detail', 'undang', 'undang-siap'],

    currentUser: null,
    meCache: null,
    contextAssignments: [],
    contextSwitching: false,
    sessionRedirecting: false,

    botOnline: false,
    botStatusDetails: null,

    ujiPesanTo: '',
    ujiPesanText: '',
    ujiPesanLoading: false,
    ujiPesanError: '',
    ujiPesanHasil: null,

    kelasList: [],
    totalKelas: 0,
    activeKelasCount: 0,
    kelasLoading: false,
    kelasFilter: { prodi: '', angkatan: '', rombel: '', status: '' },

    kelasAktif: '',
    kelasAktifObj: null,
    kelasForm: { prodi: 'D4 Teknik Informatika', angkatan: '2025', rombel: 'A', nama: 'D4 TI 2025 A', slug: 'd4-ti-2025-a', kustom: false },
    kelasError: '',
    kelasSaving: false,
    kelasStatusConfirm: null,
    detailSemesterList: [],
    detailSemesterLoading: false,
    detailSemesterError: '',
    detailSettings: null,
    detailSettingsError: '',
    portalModeConfirm: null,

    undangNomor: '',
    undangError: '',
    undangLoading: false,
    undangLink: '',

    dukunganAlasan: '',
    dukunganAktif: null,
    dukunganLoading: false,

    penggunaTab: 'akun',
    penggunaList: [],
    penggunaLoading: false,
    penggunaError: '',
    penggunaFilter: '',
    penggunaAksi: null,
    penggunaAlasan: '',

    penugasanList: [],
    penugasanLoading: false,
    penugasanError: '',
    penugasanStatus: '',
    penugasanRole: '',
    penugasanKelas: '',
    penugasanAksi: null,
    penugasanAlasan: '',
    penugasanForce: false,
    penugasanGuard: false,

    undanganList: [],
    undanganLoading: false,
    undanganError: '',
    undanganStatus: '',
    undanganRole: '',
    undanganKelas: '',
    undanganAksi: null,
    undanganAlasan: '',
    undanganSANomor: '',
    undanganSAError: '',
    undanganSALoading: false,
    undanganSAResult: null,

    ruangList: [],
    ruangLoading: false,
    ruangError: '',
    ruangForm: { kode: '', nama: '', gedung: '', tipe: '', kapasitas: '' },
    ruangFormError: '',
    ruangQ: '',
    ruangStatusFilter: '',
    ruangEdit: null,
    ruangEditError: '',
    ruangStatusConfirm: null,
    modalTambahRuang: false,
    modalImportRuang: false,
    importRuangTab: 'jadwal',
    importRuangCsvText: '',
    importRuangLoading: false,
    importRuangError: '',
    importRuangSuccess: '',

    matkulList: [],
    matkulLoading: false,
    matkulError: '',
    matkulForm: { kode: '', nama: '' },
    matkulFormError: '',
    matkulQ: '',
    matkulStatusFilter: '',
    matkulEdit: null,
    matkulEditError: '',
    matkulStatusConfirm: null,
    modalTambahMatkul: false,
    modalImportMatkul: false,
    importTab: 'jadwal',
    importCsvText: '',
    importLoading: false,
    importError: '',
    importSuccess: '',
    dosenList: [],
    dosenLoading: false,
    dosenError: '',
    dosenForm: { kode: '', nama: '' },
    dosenFormError: '',
    dosenQ: '',
    dosenStatusFilter: '',
    dosenEdit: null,
    dosenEditError: '',
    dosenStatusConfirm: null,
    modalTambahDosen: false,
    modalImportDosen: false,
    importDosenTab: 'jadwal',
    importDosenCsvText: '',
    importDosenLoading: false,
    importDosenError: '',
    importDosenSuccess: '',
    syncAllLoading: false,

    usulanList: [],
    usulanLoading: false,
    usulanError: '',
    usulanKeputusan: null,
    usulanCatatan: '',

    notifFilter: '',
    notifKelas: '',
    notifJenis: '',
    notifSince: '',
    notifUntil: '',
    notifDetailId: null,
    notifAttempts: [],
    notifAttemptsLoading: false,
    notifAttemptsError: '',

    kanalList: [],
    kanalLoading: false,
    kanalError: '',
    kanalKelas: '',
    kanalForm: { jid: '', kelas: '', nama: '' },
    kanalFormError: '',
    kanalSaving: false,
    kanalLepas: null,
    kanalAlasan: '',
    notifList: [],
    notifLoading: false,
    notifLoadingMore: false,
    notifError: '',
    notifFilterOpen: (typeof window !== 'undefined' ? window.innerWidth >= 768 : true),
    notifAdvancedOpen: false,
    notifLimit: 50,
    notifHasMore: false,
    failedCount: 0,

    backupForm: { kelas: '', semester: '', alasan: '' },
    backupError: '',
    backupHasil: null,
    backupLoading: false,
    backupList: [],
    backupRequests: [], backupRequestsLoading: false, backupRequestsError: '', backupRequestExecuting: 0,
    backupListLoading: false,
    backupListError: '',
    backupSemesterList: [],
    restoreForm: { id: '', alasan: '', paham: false },
    restoreError: '',
    restoreHasil: null,
    restoreLoading: false,
    scopedRestorePreview: null,
    scopedRestoreResult: null,
    scopedRestoreLoading: false,
    selectedBackupRestorable: false,

    auditList: [],
    auditLoading: false,
    auditLoadingMore: false,
    auditError: '',
    auditKelas: '',
    auditAction: '',
    auditEntity: '',
    auditEntityId: '',
    auditActor: '',
    auditSince: '',
    auditUntil: '',
    auditFilterOpen: (typeof window !== 'undefined' ? window.innerWidth >= 768 : true),
    auditAdvancedOpen: false,
    auditLimit: 50,
    auditOffset: 0,
    auditHasMore: false,
    auditActionOptions: ['SUSPEND_USER', 'RECOVER_USER', 'ROTATE_PORTAL_CODE', 'SUPPORT_ENTER', 'SUPPORT_EXIT', 'INVITE_ROLE', 'ASSIGN_ROLE', 'SUSPEND_ROLE', 'REVOKE_ROLE', 'UPDATE_CLASS_STATUS', 'CREATE_BACKUP', 'VERIFY_RESTORE', 'LOGIN', 'LOGOUT'],
    auditEntityOptions: ['USER', 'ROLE_ASSIGNMENT', 'ROLE_INVITATION', 'CLASS', 'CLASS_SETTINGS', 'BACKUP', 'PORTAL_SESSION', 'TASK', 'TEACHING_EVENT', 'MATERIAL'],

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
      if (this.botOnline || st === 'connected') return 'bg-emerald-500';
      if (st === 'waiting_qr' || st === 'reconnecting') return 'bg-amber-500';
      return 'bg-danger';
    },

    async kirimUjiPesan() {
      const to = (this.ujiPesanTo || '').trim();
      const text = (this.ujiPesanText || '').trim();
      if (!to || !text) { this.ujiPesanError = 'Isi JID kanal tujuan dan teks pesan.'; return; }
      if (text.length > 500) { this.ujiPesanError = 'Teks maksimal 500 karakter.'; return; }
      this.ujiPesanError = '';
      this.ujiPesanHasil = null;
      this.ujiPesanLoading = true;
      try {
        this.ujiPesanHasil = await API.testBotMessage(to, text);
        this.showToast('Pesan uji terkirim.');
      } catch (err) {
        this.ujiPesanError = err.message || 'Gagal mengirim pesan uji.';
      } finally {
        this.ujiPesanLoading = false;
      }
    },

    filteredKelas() {
      const query = (this.q || '').trim().toLowerCase();
      const f = this.kelasFilter || {};
      const fp = (f.prodi || '').trim().toLowerCase();
      const fa = (f.angkatan || '').trim();
      const fr = (f.rombel || '').trim().toUpperCase();
      const fs = (f.status || '').trim().toUpperCase();
      const fkm = (f.statusKM || '').trim();
      return (this.kelasList || []).filter(k => {
        if (fp && !String(k.prodi || '').toLowerCase().includes(fp)) return false;
        if (fa && !String(k.angkatan || '').toLowerCase().includes(fa.toLowerCase())) return false;
        if (fr && String(k.group || '').toUpperCase() !== fr) return false;
        if (fs && String(k.status || '').toUpperCase() !== fs) return false;
        if (fkm && String(k.statusKM || '') !== fkm) return false;
        if (!query) return true;
        return (
          (k.nama && k.nama.toLowerCase().includes(query)) ||
          (k.slug && k.slug.toLowerCase().includes(query)) ||
          (k.prodi && k.prodi.toLowerCase().includes(query)) ||
          (k.angkatan && k.angkatan.toLowerCase().includes(query)) ||
          (k.group && k.group.toLowerCase().includes(query)) ||
          (k.kmName && k.kmName.toLowerCase().includes(query)) ||
          (k.kmPhone && k.kmPhone.toLowerCase().includes(query))
        );
      });
    },

    resetKelasFilter() {
      this.q = '';
      this.kelasFilter = { prodi: '', angkatan: '', rombel: '', status: '', statusKM: '' };
    },

    filterKelasTanpaKM() {
      this.resetKelasFilter();
      this.kelasFilter.statusKM = 'none';
      this.go('kelas');
    },

    filteredNotif() {
      const fk = (this.notifKelas || '').trim();
      const fj = (this.notifJenis || '').trim();
      return (this.notifList || []).filter(n => {
        if (fk && String(n.class_id ?? '') !== fk) return false;
        if (fj && String(n.event_type || '') !== fj) return false;
        return true;
      });
    },

    notifKelasOptions() {
      // Label jujur: slug bila backend mengirimnya, kalau tidak "Kelas #id".
      // Opsi dibangun dari data termuat (limit) — bukan daftar kelas global.
      const seen = new Map();
      (this.notifList || []).forEach(n => {
        const id = String(n.class_id ?? '');
        if (!id || seen.has(id)) return;
        seen.set(id, n.class_slug || ('Kelas #' + id));
      });
      return Array.from(seen.entries()).map(([id, label]) => ({ id, label }));
    },

    saTitle(id) {
      const item = this.navSections.flatMap(section => section.items).find(n => n.id === id);
      return item ? item.label : id;
    },

    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed;
      try { localStorage.setItem('asterisk:sidebar:collapsed', this.sidebarCollapsed ? '1' : '0'); } catch (e) {}
    },

    knownViews: ['dashboard', 'kelas', 'buat', 'detail', 'undang', 'undang-siap', 'pengguna', 'dukungan', 'antrean', 'kanal', 'audit', 'status-bot', 'master-ruangan', 'master-matkul', 'master-dosen', 'backup'],

    showPageError(status) {
      this.pageState = { status: status };
      this.drawer = false;
      this.view = '__error';
      window.scrollTo({ top: 0 });
    },

    async initSystemAdmin() {
      try { this.sidebarCollapsed = localStorage.getItem('asterisk:sidebar:collapsed') === '1'; } catch (e) {}
      window.addEventListener('offline', () => { this.showPageError('offline'); });
      window.addEventListener('online', () => { if (this.pageState && this.pageState.status === 'offline') window.location.reload(); });
      const isAuthed = await this.checkAuth();
      if (!isAuthed) {
        this.redirectToLogin();
        return;
      }

      await AsteriskShell.mount('sa', ['dashboard','kelas','undang','dukungan','antrean','kanal','backup','pengguna','ruangan','matkul','dosen','soon']);
      await this.loadPartials([
        ['sa-dashboard', '/partials/system-admin/view-dashboard.html'],
        ['sa-kelas', '/partials/system-admin/view-kelas.html'],
        ['sa-undang', '/partials/system-admin/view-undang.html'],
        ['sa-dukungan', '/partials/system-admin/view-dukungan.html'],
        ['sa-antrean', '/partials/system-admin/view-antrean.html'],
        ['sa-kanal', '/partials/system-admin/view-kanal.html'],
        ['sa-backup', '/partials/system-admin/view-backup.html'],
        ['sa-pengguna', '/partials/system-admin/view-pengguna.html'],
        ['sa-ruangan', '/partials/system-admin/view-master-ruangan.html'],
        ['sa-matkul', '/partials/system-admin/view-master-matkul.html'],
        ['sa-dosen', '/partials/system-admin/view-master-dosen.html'],
        ['sa-soon', '/partials/system-admin/view-soon.html'],
        ['sa-banner', '/partials/system-admin/support-banner.html']
      ]);

      await Promise.all([this.checkBot(), this.loadKelas(), this.loadFailedCount(), this.muatDukunganAktif()]);
      this.dashboardLoading = false;

      setInterval(() => {
        if (this.currentUser) { this.checkBot(); this.loadFailedCount(); this.muatDukunganAktif(); }
      }, 30000);
    },

    async loadPartials(slots) {
      // Alpine v3 auto-init node baru via MutationObserver; jangan initTree manual (render ganda).
      await Promise.all(slots.map(async ([id, url]) => {
        try {
          const res = await fetch(url + '?v=' + encodeURIComponent(AsteriskShell.version), { cache: 'no-store' });
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
          this.meCache = me;
          this.contextAssignments = Array.isArray(me.assignments) ? me.assignments : [];
          return true;
        }
      } catch (e) {}
      return false;
    },

    contextLabel(a) {
      const role = String(a && a.role || '').toUpperCase();
      const scope = a && (a.offering_name || a.class_slug) || 'Global';
      return `${role === 'SYSTEM_ADMIN' ? 'System Admin' : role} · ${scope}`;
    },

    async switchContextById(id) {
      const activeId = this.meCache && this.meCache.active_assignment && this.meCache.active_assignment.id;
      if (!id || this.contextSwitching || String(id) === String(activeId)) return;
      this.contextSwitching = true;
      try {
        const result = await API.switchContext(Number(id));
        const chosen = this.contextAssignments.find(a => String(a.id) === String(id));
        const role = String((result && (result.role || result.active_role)) || (chosen && chosen.role) || '').toUpperCase();
        window.location.href = role === 'KM' ? '/km.html' : role === 'PJ' ? '/pj.html' : '/system-admin.html';
      } catch (err) {
        this.showToast(err.message || 'Konteks akses tidak tersedia.');
      } finally { this.contextSwitching = false; }
    },

    redirectToLogin() {
      if (this.sessionRedirecting) return;
      this.sessionRedirecting = true;
      API.setAuthToken(null);
      this.currentUser = null;
      window.location.replace('/login.html?role=sa');
    },

    statusNotifLabel(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'PENDING') return 'Menunggu';
      if (s === 'PROCESSING') return 'Diproses';
      if (s === 'SENT') return 'Terkirim';
      if (s === 'FAILED') return 'Gagal';
      if (s === 'CANCELLED') return 'Dibatalkan';
      return st || '-';
    },

    notifDapatDiulang(st) {
      const s = String(st || '').toUpperCase();
      return s === 'FAILED' || s === 'CANCELLED';
    },

    toggleNotifDetail(id) {
      const membuka = this.notifDetailId !== id;
      this.notifDetailId = membuka ? id : null;
      if (membuka) this.loadNotifAttempts(id);
    },

    async loadKanal() {
      this.kanalLoading = true; this.kanalError = '';
      try {
        this.kanalList = await API.getChannels(this.kanalKelas || '', '');
      } catch (e) {
        this.kanalList = [];
        this.kanalError = (e && e.code === 'UNAUTHORIZED')
          ? 'Sesi berakhir. Masuk kembali lalu coba lagi.'
          : 'Daftar kanal belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.kanalLoading = false;
      }
    },

    resetKanalFilter() {
      this.kanalKelas = '';
    },

    labelStatusKanal(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'ACTIVE') return 'Aktif';
      if (s === 'DISCONNECTED') return 'Terputus';
      if (s === 'REVOKED') return 'Dilepas';
      return st || '-';
    },

    async tautkanKanal() {
      const f = this.kanalForm;
      if (!((f.jid || '').trim()) || !(f.jid || '').includes('@')) { this.kanalFormError = 'JID grup wajib diisi (minta via perintah !kanal di grup).'; return; }
      if (!f.kelas) { this.kanalFormError = 'Pilih kelas tujuan.'; return; }
      this.kanalFormError = '';
      this.kanalSaving = true;
      try {
        const res = await API.linkChannel(f.jid.trim(), f.kelas, (f.nama || '').trim());
        this.showToast(res && res.changed === false ? 'Kanal sudah tertaut ke kelas ini.' : 'Kanal berhasil ditautkan.');
        this.kanalForm = { jid: '', kelas: '', nama: '' };
        await this.loadKanal();
      } catch (err) {
        this.kanalFormError = err.message || 'Gagal menautkan kanal.';
      } finally {
        this.kanalSaving = false;
      }
    },

    mulaiLepasKanal(k) {
      this.kanalLepas = { id: k.id, nama: (k.display_name || k.jid) + ' · ' + (k.class_slug || '') };
      this.kanalAlasan = '';
    },

    async jalankanLepasKanal() {
      const k = this.kanalLepas;
      if (!k) return;
      if (!((this.kanalAlasan || '').trim())) {
        this.showToast('Isi alasan pelepasan terlebih dahulu.');
        return;
      }
      try {
        await API.revokeChannel(k.id, this.kanalAlasan.trim());
        this.showToast(`Kanal ${k.nama} dilepas. Pesan PENDING-nya tetap yatim.`);
        this.kanalLepas = null;
        await this.loadKanal();
      } catch (err) {
        this.showToast(err.message || 'Gagal melepas kanal.');
      }
    },

    async loadNotifAttempts(id) {
      this.notifAttemptsLoading = true;
      this.notifAttemptsError = '';
      this.notifAttempts = [];
      try {
        const list = await API.getNotificationAttempts(id) || [];
        // Abaikan hasil basi bila pengguna sudah membuka detail lain.
        if (this.notifDetailId !== id) return;
        this.notifAttempts = list;
      } catch (e) {
        if (this.notifDetailId !== id) return;
        this.notifAttemptsError = (e && e.code === 'UNAUTHORIZED')
          ? 'Sesi berakhir. Masuk kembali lalu coba lagi.'
          : 'Riwayat percobaan belum dapat dimuat.';
      } finally {
        this.notifAttemptsLoading = false;
      }
    },

    prettyPayload(json) {
      if (json == null || json === '') return '-';
      if (typeof json === 'object') {
        try { return JSON.stringify(json, null, 2); } catch (e) { return String(json); }
      }
      const s = String(json);
      try { return JSON.stringify(JSON.parse(s), null, 2); } catch (e) { return s; }
    },

    notifJenisOptions() {
      const seen = new Set();
      (this.notifList || []).forEach(n => {
        const t = String(n.event_type || '');
        if (t && !seen.has(t)) seen.add(t);
      });
      return Array.from(seen).sort();
    },

    notifFilterCount() {
      let n = 0;
      if ((this.notifFilter || '').trim()) n++;
      if ((this.notifKelas || '').trim()) n++;
      if ((this.notifJenis || '').trim()) n++;
      if ((this.notifSince || '').trim()) n++;
      if ((this.notifUntil || '').trim()) n++;
      return n;
    },

    notifChips() {
      const out = [];
      if ((this.notifFilter || '').trim()) out.push({ key: 'notifFilter', label: 'Status', value: this.statusNotifLabel(this.notifFilter) });
      if ((this.notifKelas || '').trim()) out.push({ key: 'notifKelas', label: 'Kelas', value: this.notifKelasLabelById(this.notifKelas) });
      if ((this.notifJenis || '').trim()) out.push({ key: 'notifJenis', label: 'Jenis', value: this.notifJenis.trim() });
      if ((this.notifSince || '').trim()) out.push({ key: 'notifSince', label: 'Sejak', value: this.notifSince.trim() });
      if ((this.notifUntil || '').trim()) out.push({ key: 'notifUntil', label: 'Sampai', value: this.notifUntil.trim() });
      return out;
    },

    removeNotifChip(key) {
      if (key && key in this) this[key] = '';
      this.loadAntrean();
    },

    notifAdvancedCount() {
      let n = 0;
      if ((this.notifKelas || '').trim()) n++;
      if ((this.notifSince || '').trim()) n++;
      if ((this.notifUntil || '').trim()) n++;
      return n;
    },

    notifKelasLabelById(id) {
      const opt = (this.notifKelasOptions() || []).find(o => String(o.id) === String(id));
      return opt ? opt.label : ('Kelas #' + id);
    },

    notifKelasLabel(n) {
      if (!n) return '-';
      return n.class_slug || ('Kelas #' + (n.class_id ?? '-'));
    },

    notifPenerima(n) {
      if (!n) return '— (tanpa kanal terdaftar)';
      if (n.channel_name) return n.channel_name + (n.channel_jid ? ' · ' + n.channel_jid : '');
      if (n.channel_jid) return n.channel_jid;
      return '— (tanpa kanal terdaftar)';
    },

    bukaAuditEntitas(entityType) {
      this.auditKelas = '';
      this.auditAction = '';
      this.auditEntity = String(entityType || '');
      this.auditEntityId = '';
      this.auditActor = '';
      this.auditSince = '';
      this.auditUntil = '';
      this.go('audit');
    },

    fmtWaktuID(iso) { return API.fmtWaktuID(iso); },

    async loadPengguna() {
      this.penggunaLoading = true; this.penggunaError = '';
      try {
        this.penggunaList = await API.getAdminUsers(this.penggunaFilter || '');
      } catch (e) {
        this.penggunaList = [];
        this.penggunaError = 'Daftar pengguna belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.penggunaLoading = false;
      }
    },

    mulaiAksiPengguna(u, aksi) {
      if (aksi === 'tangguhkan' && this.isAkunSendiri(u)) {
        this.showToast('Tidak dapat menangguhkan akun Anda sendiri.');
        return;
      }
      this.penggunaAksi = { id: u.id, nama: u.display_name || u.identity_key, aksi: aksi };
      this.penggunaAlasan = '';
    },

    async jalankanAksiPengguna() {
      const a = this.penggunaAksi;
      if (!a) return;
      if (!((this.penggunaAlasan || '').trim())) {
        this.showToast('Isi alasan tindakan terlebih dahulu.');
        return;
      }
      try {
        if (a.aksi === 'tangguhkan') {
          await API.suspendUser(a.id, this.penggunaAlasan.trim());
          this.showToast('Akun ditangguhkan dan sesinya dicabut.');
        } else {
          await API.recoverUser(a.id, this.penggunaAlasan.trim());
          this.showToast('Akun dipulihkan.');
        }
        this.penggunaAksi = null;
        await this.loadPengguna();
      } catch (err) {
        this.showToast(err.message || 'Gagal memproses tindakan.');
      }
    },

    isAkunSendiri(u) {
      return !!(this.currentUser && u && String(u.id) === String(this.currentUser.id));
    },

    isPenugasanAktifSendiri(a) {
      const aktif = this.meCache && this.meCache.active_assignment && this.meCache.active_assignment.id;
      return !!(aktif && a && String(a.id) === String(aktif));
    },

    labelStatusPengguna(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'ACTIVE') return 'Aktif';
      if (s === 'SUSPENDED') return 'Ditangguhkan';
      if (s === 'REVOKED') return 'Dicabut';
      return st || '-';
    },

    labelStatusPenugasan(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'ACTIVE') return 'Aktif';
      if (s === 'SUSPENDED') return 'Ditangguhkan';
      if (s === 'REVOKED') return 'Dicabut';
      return st || '-';
    },

    labelStatusUndangan(st, isExpired) {
      const s = String(st || '').toUpperCase();
      if (s === 'PENDING' && isExpired) return 'Kedaluwarsa';
      if (s === 'PENDING') return 'Menunggu';
      if (s === 'ACCEPTED') return 'Diterima';
      if (s === 'EXPIRED') return 'Kedaluwarsa';
      if (s === 'REVOKED') return 'Dicabut';
      return st || '-';
    },

    undanganSisaWaktu(expiresAt) {
      // Label relatif sisa berlaku; '' bila tak dapat di-parse.
      try {
        const t = new Date(expiresAt);
        if (isNaN(t)) return '';
        const ms = t.getTime() - Date.now();
        if (ms <= 0) return 'kedaluwarsa';
        const h = Math.floor(ms / 3600000);
        if (h < 1) return 'kurang dari 1 jam lagi';
        if (h < 24) return h + ' jam lagi';
        const d = Math.floor(h / 24);
        return d + ' hari lagi';
      } catch (e) { return ''; }
    },

    scopePenugasan(a) {
      if (!a) return '-';
      if (a.role === 'SYSTEM_ADMIN') return 'Global';
      const parts = [];
      if (a.class_slug || a.class_code) parts.push(a.class_slug || a.class_code);
      if (a.semester_label) parts.push(a.semester_label);
      if (a.course_code || a.course_name) parts.push(a.course_code || a.course_name);
      else if (a.offering_display) parts.push(a.offering_display);
      return parts.length ? parts.join(' · ') : '-';
    },

    resetPenugasanFilter() {
      this.penugasanStatus = '';
      this.penugasanRole = '';
      this.penugasanKelas = '';
    },

    resetUndanganFilter() {
      this.undanganStatus = '';
      this.undanganRole = '';
      this.undanganKelas = '';
    },

    async loadPenugasan() {
      this.penugasanLoading = true; this.penugasanError = '';
      try {
        this.penugasanList = await API.getAdminAssignments(this.penugasanStatus || '', this.penugasanRole || '', this.penugasanKelas || '');
      } catch (e) {
        this.penugasanList = [];
        this.penugasanError = (e && e.code === 'UNAUTHORIZED')
          ? 'Sesi berakhir. Masuk kembali lalu coba lagi.'
          : 'Daftar penugasan belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.penugasanLoading = false;
      }
    },

    mulaiAksiPenugasan(a, aksi) {
      if (this.isPenugasanAktifSendiri(a)) {
        this.showToast('Tidak dapat mengubah penugasan aktif Anda sendiri.');
        return;
      }
      this.penugasanAksi = {
        id: a.id,
        nama: (a.display_name || a.identity_key) + ' · ' + a.role,
        aksi,
        kelasSlug: a.class_slug || '',
        peran: a.role
      };
      this.penugasanAlasan = '';
      this.penugasanForce = false;
      // Guard proaktif: KM terakhir yang masih aktif di kelasnya.
      this.penugasanGuard = String(a.role || '').toUpperCase() === 'KM'
        && String(a.status || '').toUpperCase() === 'ACTIVE'
        && (this.penugasanList || []).filter(x =>
          x.id !== a.id
          && String(x.role || '').toUpperCase() === 'KM'
          && String(x.status || '').toUpperCase() === 'ACTIVE'
          && (x.class_slug || '') === (a.class_slug || '')
        ).length === 0;
    },

    async jalankanAksiPenugasan() {
      const a = this.penugasanAksi;
      if (!a) return;
      if (!((this.penugasanAlasan || '').trim())) {
        this.showToast('Isi alasan tindakan terlebih dahulu.');
        return;
      }
      try {
        const res = await API.changeAssignmentStatus(a.id, a.aksi, this.penugasanAlasan.trim(), this.penugasanForce);
        const sesi = res && res.revoked_sessions != null ? ` (${res.revoked_sessions} sesi dicabut)` : '';
        this.showToast(a.aksi === 'cabut' ? `Penugasan ${a.nama} dicabut${sesi}.` : `Penugasan ${a.nama} ditangguhkan${sesi}.`);
        this.penugasanAksi = null;
        this.penugasanForce = false;
        this.penugasanGuard = false;
        await this.loadPenugasan();
      } catch (err) {
        if (err && err.status === 409 && /seluruh KM/i.test(err.message || '')) {
          this.penugasanGuard = true;
        }
        this.showToast(err.message || 'Gagal memproses tindakan.');
      }
    },

    undangPenggantiKM() {
      const a = this.penugasanAksi;
      const slug = a && a.kelasSlug ? a.kelasSlug : '';
      const target = (this.kelasList || []).find(k => k.slug === slug);
      if (!target) { this.showToast('Kelas tidak ditemukan di daftar. Muat ulang Daftar Kelas.'); return; }
      this.penugasanAksi = null;
      this.bukaUndang(target);
    },

    async loadUndangan() {
      this.undanganLoading = true; this.undanganError = '';
      try {
        this.undanganList = await API.getAdminInvitations(this.undanganStatus || '', this.undanganRole || '', this.undanganKelas || '');
      } catch (e) {
        this.undanganList = [];
        this.undanganError = (e && e.code === 'UNAUTHORIZED')
          ? 'Sesi berakhir. Masuk kembali lalu coba lagi.'
          : 'Daftar undangan belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.undanganLoading = false;
      }
    },

    mulaiAksiUndangan(u) {
      this.undanganAksi = { id: u.id, nama: (u.invited_identity_key || '') + ' · ' + u.role };
      this.undanganAlasan = '';
    },

    async jalankanAksiUndangan() {
      const a = this.undanganAksi;
      if (!a) return;
      if (!((this.undanganAlasan || '').trim())) {
        this.showToast('Isi alasan pencabutan terlebih dahulu.');
        return;
      }
      try {
        await API.revokeInvitation(a.id, this.undanganAlasan.trim());
        this.showToast(`Undangan ${a.nama} dicabut. Buat tautan baru via Daftar Kelas bila masih dibutuhkan.`);
        this.undanganAksi = null;
        await this.loadUndangan();
      } catch (err) {
        this.showToast(err.message || 'Gagal mencabut undangan.');
      }
    },

    async buatUndanganSA() {
      const nomor = (this.undanganSANomor || '').trim();
      if (nomor.replace(/\D/g, '').length < 9) {
        this.undanganSAError = 'Nomor WhatsApp calon System Admin tidak valid (minimal 9 digit).';
        return;
      }
      this.undanganSAError = '';
      this.undanganSAResult = null;
      this.undanganSALoading = true;
      try {
        const res = await API.createInvitation({ role: 'SYSTEM_ADMIN', invited_identity_key: nomor });
        const token = res && res.token ? res.token : '';
        const link = token ? `${window.location.origin}/invite.html?token=${encodeURIComponent(token)}` : '';
        this.undanganSAResult = {
          id: res && res.invitation_id,
          expires_at: res && res.expires_at,
          link
        };
        this.showToast('Undangan System Admin dibuat (berlaku 7 hari, sekali pakai).');
        this.undanganSANomor = '';
        await this.loadUndangan();
        if (link) this.copyText(link, 'Tautan undangan SA disalin.');
      } catch (err) {
        this.undanganSAError = err.message || 'Gagal membuat undangan System Admin.';
      } finally {
        this.undanganSALoading = false;
      }
    },

    kirimUlangUndangan(u) {
      const slug = u && u.class_slug ? u.class_slug : '';
      const target = (this.kelasList || []).find(k => k.slug === slug);
      if (!target) {
        // Tanpa dead-end: bawa ke Daftar Kelas agar admin pilih kelas manual.
        this.go('kelas');
        this.showToast('Pilih kelas undangan di Daftar Kelas, lalu buat tautan baru.');
        return;
      }
      this.bukaUndang(target);
      this.undangNomor = u.invited_identity_key || '';
      this.showToast('Nomor terisi dari undangan lama. Buat tautan baru untuk membatalkan token lama.');
    },

    labelStatusKelas(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'ACTIVE') return 'Aktif';
      if (s === 'INACTIVE') return 'Nonaktif';
      if (s === 'ARCHIVED') return 'Arsip';
      return st || '-';
    },

    filteredRuang() {
      const q = (this.ruangQ || '').trim().toLowerCase();
      if (!q) return this.ruangList || [];
      return (this.ruangList || []).filter(r =>
        (r.code && r.code.toLowerCase().includes(q)) ||
        (r.name && r.name.toLowerCase().includes(q)) ||
        (r.building && r.building.toLowerCase().includes(q))
      );
    },

    async loadRuang() {
      this.ruangLoading = true; this.ruangError = '';
      try {
        this.ruangList = await API.getMasterRooms(this.ruangStatusFilter || '');
      } catch (e) {
        this.ruangList = [];
        this.ruangError = 'Daftar ruangan belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.ruangLoading = false;
      }
    },

    bukaModalTambahRuang() {
      this.ruangForm = { kode: '', nama: '', gedung: '', tipe: '', kapasitas: '' };
      this.ruangFormError = '';
      this.modalTambahRuang = true;
    },

    tutupModalTambahRuang() {
      this.modalTambahRuang = false;
      this.ruangFormError = '';
    },

    bukaModalImportRuang() {
      this.modalImportRuang = true;
      this.importRuangTab = 'jadwal';
      this.importRuangCsvText = '';
      this.importRuangLoading = false;
      this.importRuangError = '';
      this.importRuangSuccess = '';
    },

    tutupModalImportRuang() {
      this.modalImportRuang = false;
      this.importRuangLoading = false;
      this.importRuangError = '';
      this.importRuangSuccess = '';
    },

    async sinkronRuangDariJadwal() {
      this.importRuangLoading = true;
      this.importRuangError = '';
      this.importRuangSuccess = '';
      try {
        const res = await API.syncMasterRoomsJadwal();
        this.importRuangSuccess = res.message || `Berhasil menyinkronkan ${res.total_synced || 0} ruangan dari berkas jadwal.`;
        await this.loadRuang();
        this.showToast(`Berhasil menyinkronkan ${res.total_synced || 0} ruangan.`);
      } catch (err) {
        this.importRuangError = err.message || 'Gagal menyinkronkan ruangan dari jadwal.';
      } finally {
        this.importRuangLoading = false;
      }
    },

    async tambahRuang() {
      const f = this.ruangForm;
      if (!((f.kode || '').trim())) { this.ruangFormError = 'Kode ruangan wajib diisi.'; return; }
      if ((f.kapasitas || '') !== '' && !(/^\d+$/.test(String(f.kapasitas).trim()))) { this.ruangFormError = 'Kapasitas wajib angka bulat ≥ 0.'; return; }
      this.ruangFormError = '';
      try {
        const payload = { code: f.kode.trim() };
        if ((f.nama || '').trim()) payload.name = f.nama.trim();
        if ((f.gedung || '').trim()) payload.building = f.gedung.trim();
        if ((f.tipe || '').trim()) payload.room_type = f.tipe.trim();
        if ((f.kapasitas || '') !== '') payload.capacity = Number(String(f.kapasitas).trim());
        await API.createMasterRoom(payload);
        this.ruangForm = { kode: '', nama: '', gedung: '', tipe: '', kapasitas: '' };
        await this.loadRuang();
        this.modalTambahRuang = false;
        this.showToast('Ruangan ditambahkan.');
      } catch (err) {
        this.ruangFormError = err.message || 'Gagal menambah ruangan.';
      }
    },

    mulaiUbahRuang(r) {
      const cap = r.capacity;
      this.ruangEdit = { id: r.id, kode: r.code, nama: r.name || '', gedung: r.building || '', tipe: r.room_type || '', kapasitas: (cap === null || cap === undefined || cap === '') ? '' : String(cap) };
      this.ruangEditError = '';
    },

    batalUbahRuang() {
      this.ruangEdit = null;
      this.ruangEditError = '';
    },

    async simpanUbahRuang() {
      const f = this.ruangEdit;
      if (!f) return;
      if ((f.kapasitas || '') !== '' && !(/^\d+$/.test(String(f.kapasitas).trim()))) { this.ruangEditError = 'Kapasitas wajib angka bulat ≥ 0.'; return; }
      this.ruangEditError = '';
      try {
        const payload = {
          name: (f.nama || '').trim(),
          building: (f.gedung || '').trim(),
          room_type: (f.tipe || '').trim()
        };
        if ((f.kapasitas || '') !== '') payload.capacity = Number(String(f.kapasitas).trim());
        await API.patchMasterRoom(f.id, payload);
        this.showToast(`Ruangan ${f.kode} diubah.`);
        this.ruangEdit = null;
        await this.loadRuang();
      } catch (err) {
        this.ruangEditError = err.message || 'Gagal mengubah ruangan.';
      }
    },

    mintaKonfirmasiStatusRuang(r) {
      this.ruangStatusConfirm = { id: r.id, kode: r.code, dari: String(r.status || '').toUpperCase(), ke: String(r.status).toUpperCase() === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE' };
    },

    batalKonfirmasiStatusRuang() {
      this.ruangStatusConfirm = null;
    },

    async jalankanUbahStatusRuang() {
      const c = this.ruangStatusConfirm;
      if (!c) return;
      try {
        await API.patchMasterRoom(c.id, { status: c.ke });
        this.ruangStatusConfirm = null;
        await this.loadRuang();
        this.showToast(c.ke === 'ACTIVE' ? 'Ruangan diaktifkan.' : 'Ruangan dinonaktifkan. Jadwal lama tetap tampil; ruangan nonaktif tak dipakai untuk jadwal baru.');
      } catch (err) {
        this.showToast(err.message || 'Gagal mengubah status ruangan.');
      }
    },

    filteredMatkul() {
      const q = (this.matkulQ || '').trim().toLowerCase();
      if (!q) return this.matkulList || [];
      return (this.matkulList || []).filter(m =>
        (m.code && m.code.toLowerCase().includes(q)) ||
        (m.name && m.name.toLowerCase().includes(q))
      );
    },

    async loadMatkul() {
      this.matkulLoading = true; this.matkulError = '';
      try {
        this.matkulList = await API.getMasterCourses(this.matkulStatusFilter || '');
      } catch (e) {
        this.matkulList = [];
        this.matkulError = 'Daftar mata kuliah belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.matkulLoading = false;
      }
    },

    bukaModalTambahMatkul() {
      this.matkulForm = { kode: '', nama: '' };
      this.matkulFormError = '';
      this.modalTambahMatkul = true;
    },

    tutupModalTambahMatkul() {
      this.modalTambahMatkul = false;
      this.matkulFormError = '';
    },

    bukaModalImportMatkul() {
      this.modalImportMatkul = true;
      this.importTab = 'jadwal';
      this.importCsvText = '';
      this.importLoading = false;
      this.importError = '';
      this.importSuccess = '';
    },

    tutupModalImportMatkul() {
      this.modalImportMatkul = false;
      this.importLoading = false;
      this.importError = '';
      this.importSuccess = '';
    },

    parsedImportCourses() {
      const text = (this.importCsvText || '').trim();
      if (!text) return [];
      if (text.startsWith('[') || text.startsWith('{')) {
        try {
          const parsed = JSON.parse(text);
          const list = Array.isArray(parsed) ? parsed : (parsed.courses || []);
          return list.map(item => ({
            code: (item.code || item.kode || '').trim(),
            name: (item.name || item.nama || item.matkul || '').trim()
          })).filter(item => item.code && item.name);
        } catch (_) {}
      }
      const lines = text.split(/\r?\n/);
      const out = [];
      const seen = new Set();
      for (let rawLine of lines) {
        const line = rawLine.trim();
        if (!line || line.startsWith('#') || line.startsWith('//')) continue;
        let delimiter = ',';
        if (line.includes('\t')) delimiter = '\t';
        else if (line.includes(';') && !line.includes(',')) delimiter = ';';
        else if (line.includes('|')) delimiter = '|';

        const parts = line.split(delimiter);
        if (parts.length >= 2) {
          const code = parts[0].trim();
          const name = parts.slice(1).join(delimiter).trim();
          if (code && name && !seen.has(code.toUpperCase())) {
            seen.add(code.toUpperCase());
            out.push({ code, name });
          }
        }
      }
      return out;
    },

    async sinkronMatkulDariJadwal() {
      this.importLoading = true;
      this.importError = '';
      this.importSuccess = '';
      try {
        const res = await API.syncMasterCoursesJadwal();
        this.importSuccess = res.message || `Berhasil menyinkronkan ${res.total_synced || 0} mata kuliah dari berkas jadwal.`;
        await this.loadMatkul();
        this.showToast(`Berhasil menyinkronkan ${res.total_synced || 0} mata kuliah.`);
      } catch (err) {
        this.importError = err.message || 'Gagal menyinkronkan mata kuliah dari jadwal.';
      } finally {
        this.importLoading = false;
      }
    },

    async prosesImportCsvMatkul() {
      const courses = this.parsedImportCourses();
      if (!courses.length) {
        this.importError = 'Tidak ada baris data mata kuliah yang valid. Pastikan format: KODE, NAMA MATA KULIAH';
        return;
      }
      this.importLoading = true;
      this.importError = '';
      this.importSuccess = '';
      try {
        const res = await API.bulkCreateMasterCourses(courses);
        this.importSuccess = res.message || `Berhasil mengimpor ${res.total_imported || 0} mata kuliah.`;
        this.importCsvText = '';
        await this.loadMatkul();
        this.showToast(`Berhasil mengimpor ${res.total_imported || 0} mata kuliah.`);
      } catch (err) {
        this.importError = err.message || 'Gagal mengimpor mata kuliah.';
      } finally {
        this.importLoading = false;
      }
    },

    async tambahMatkul() {
      const f = this.matkulForm;
      if (!((f.kode || '').trim()) || !((f.nama || '').trim())) {
        this.matkulFormError = 'Kode dan nama mata kuliah wajib diisi.';
        return;
      }
      this.matkulFormError = '';
      try {
        await API.createMasterCourse({ code: f.kode.trim(), name: f.nama.trim() });
        this.matkulForm = { kode: '', nama: '' };
        await this.loadMatkul();
        this.modalTambahMatkul = false;
        this.showToast('Mata kuliah ditambahkan.');
      } catch (err) {
        this.matkulFormError = err.message || 'Gagal menambah mata kuliah.';
      }
    },

    mulaiUbahMatkul(m) {
      this.matkulEdit = { id: m.id, kode: m.code, nama: m.name || '' };
      this.matkulEditError = '';
    },

    batalUbahMatkul() {
      this.matkulEdit = null;
      this.matkulEditError = '';
    },

    async simpanUbahMatkul() {
      const f = this.matkulEdit;
      if (!f) return;
      if (!((f.nama || '').trim())) { this.matkulEditError = 'Nama mata kuliah wajib diisi.'; return; }
      this.matkulEditError = '';
      try {
        await API.patchMasterCourse(f.id, { name: f.nama.trim() });
        this.showToast(`Mata kuliah ${f.kode} diubah.`);
        this.matkulEdit = null;
        await this.loadMatkul();
      } catch (err) {
        this.matkulEditError = err.message || 'Gagal mengubah mata kuliah.';
      }
    },

    mintaKonfirmasiStatusMatkul(m) {
      this.matkulStatusConfirm = { id: m.id, kode: m.code, dari: String(m.status || '').toUpperCase(), ke: String(m.status).toUpperCase() === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE' };
    },

    batalKonfirmasiStatusMatkul() {
      this.matkulStatusConfirm = null;
    },

    async jalankanUbahStatusMatkul() {
      const c = this.matkulStatusConfirm;
      if (!c) return;
      try {
        await API.patchMasterCourse(c.id, { status: c.ke });
        this.matkulStatusConfirm = null;
        await this.loadMatkul();
        this.showToast(c.ke === 'ACTIVE' ? 'Mata kuliah diaktifkan.' : 'Mata kuliah dinonaktifkan. Penawaran lama tetap tampil; nonaktif tak dipakai untuk penawaran baru.');
      } catch (err) {
        this.showToast(err.message || 'Gagal mengubah status.');
      }
    },

    filteredDosen() {
      const q = (this.dosenQ || '').trim().toLowerCase();
      if (!q) return this.dosenList || [];
      return (this.dosenList || []).filter(d =>
        (d.code && d.code.toLowerCase().includes(q)) ||
        (d.full_name && d.full_name.toLowerCase().includes(q)) ||
        (d.name && d.name.toLowerCase().includes(q))
      );
    },

    async loadDosen() {
      this.dosenLoading = true; this.dosenError = '';
      try {
        this.dosenList = await API.getMasterLecturers(this.dosenStatusFilter || '');
      } catch (e) {
        this.dosenList = [];
        this.dosenError = 'Daftar dosen belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.dosenLoading = false;
      }
    },

    bukaModalTambahDosen() {
      this.dosenForm = { kode: '', nama: '' };
      this.dosenFormError = '';
      this.modalTambahDosen = true;
    },

    tutupModalTambahDosen() {
      this.modalTambahDosen = false;
      this.dosenFormError = '';
    },

    async tambahDosen() {
      const f = this.dosenForm;
      if (!((f.kode || '').trim()) || !((f.nama || '').trim())) {
        this.dosenFormError = 'Inisial dan nama dosen wajib diisi.';
        return;
      }
      this.dosenFormError = '';
      try {
        await API.createMasterLecturer({ code: f.kode.trim(), full_name: f.nama.trim() });
        this.dosenForm = { kode: '', nama: '' };
        this.modalTambahDosen = false;
        await this.loadDosen();
        this.showToast('Dosen ditambahkan.');
      } catch (err) {
        this.dosenFormError = err.message || 'Gagal menambah dosen.';
      }
    },

    mulaiUbahDosen(d) {
      this.dosenEdit = { id: d.id, kode: d.code, nama: d.full_name || d.name || '' };
      this.dosenEditError = '';
    },

    batalUbahDosen() {
      this.dosenEdit = null;
      this.dosenEditError = '';
    },

    async simpanUbahDosen() {
      const f = this.dosenEdit;
      if (!f) return;
      if (!((f.nama || '').trim())) { this.dosenEditError = 'Nama dosen wajib diisi.'; return; }
      this.dosenEditError = '';
      try {
        await API.patchMasterLecturer(f.id, { full_name: f.nama.trim() });
        this.showToast(`Data dosen ${f.kode} diubah.`);
        this.dosenEdit = null;
        await this.loadDosen();
      } catch (err) {
        this.dosenEditError = err.message || 'Gagal mengubah data dosen.';
      }
    },

    mintaKonfirmasiStatusDosen(d) {
      this.dosenStatusConfirm = {
        id: d.id,
        kode: d.code,
        nama: d.full_name || d.name,
        dari: String(d.status || '').toUpperCase(),
        ke: String(d.status).toUpperCase() === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE'
      };
    },

    batalKonfirmasiStatusDosen() {
      this.dosenStatusConfirm = null;
    },

    async jalankanUbahStatusDosen() {
      const c = this.dosenStatusConfirm;
      if (!c) return;
      try {
        await API.patchMasterLecturer(c.id, { status: c.ke });
        this.dosenStatusConfirm = null;
        await this.loadDosen();
        this.showToast(c.ke === 'ACTIVE' ? 'Dosen diaktifkan.' : 'Dosen dinonaktifkan.');
      } catch (err) {
        this.showToast(err.message || 'Gagal mengubah status dosen.');
      }
    },

    bukaModalImportDosen() {
      this.modalImportDosen = true;
      this.importDosenTab = 'jadwal';
      this.importDosenCsvText = '';
      this.importDosenLoading = false;
      this.importDosenError = '';
      this.importDosenSuccess = '';
    },

    tutupModalImportDosen() {
      this.modalImportDosen = false;
      this.importDosenLoading = false;
      this.importDosenError = '';
      this.importDosenSuccess = '';
    },

    parsedImportDosen() {
      const text = (this.importDosenCsvText || '').trim();
      if (!text) return [];
      if (text.startsWith('[') || text.startsWith('{')) {
        try {
          const parsed = JSON.parse(text);
          const list = Array.isArray(parsed) ? parsed : (parsed.lecturers || parsed.dosen || []);
          return list.map(item => ({
            code: (item.code || item.kode || item.inisial || '').trim(),
            full_name: (item.full_name || item.name || item.nama || '').trim()
          })).filter(item => item.code && item.full_name);
        } catch (_) {}
      }
      const lines = text.split(/\r?\n/);
      const out = [];
      const seen = new Set();
      for (let rawLine of lines) {
        const line = rawLine.trim();
        if (!line || line.startsWith('#') || line.startsWith('//')) continue;
        let delimiter = ',';
        if (line.includes('\t')) delimiter = '\t';
        else if (line.includes(';') && !line.includes(',')) delimiter = ';';
        else if (line.includes('|')) delimiter = '|';

        const parts = line.split(delimiter);
        if (parts.length >= 2) {
          const code = parts[0].trim();
          const fullName = parts.slice(1).join(delimiter).trim();
          if (code && fullName && !seen.has(code.toUpperCase())) {
            seen.add(code.toUpperCase());
            out.push({ code, full_name: fullName });
          }
        }
      }
      return out;
    },

    async sinkronDosenDariJadwal() {
      this.importDosenLoading = true;
      this.importDosenError = '';
      this.importDosenSuccess = '';
      try {
        const res = await API.syncMasterLecturersJadwal();
        this.importDosenSuccess = res.message || `Berhasil menyinkronkan ${res.total_synced || 0} dosen dari berkas jadwal.`;
        await this.loadDosen();
        this.showToast(`Berhasil menyinkronkan ${res.total_synced || 0} dosen.`);
      } catch (err) {
        this.importDosenError = err.message || 'Gagal menyinkronkan dosen dari jadwal.';
      } finally {
        this.importDosenLoading = false;
      }
    },

    async prosesImportCsvDosen() {
      const lecturers = this.parsedImportDosen();
      if (!lecturers.length) {
        this.importDosenError = 'Tidak ada baris data dosen yang valid. Format: INISIAL, NAMA LENGKAP & GELAR';
        return;
      }
      this.importDosenLoading = true;
      this.importDosenError = '';
      this.importDosenSuccess = '';
      try {
        const res = await API.bulkCreateMasterLecturers(lecturers);
        this.importDosenSuccess = res.message || `Berhasil mengimpor ${res.total_imported || 0} dosen.`;
        this.importDosenCsvText = '';
        await this.loadDosen();
        this.showToast(`Berhasil mengimpor ${res.total_imported || 0} dosen.`);
      } catch (err) {
        this.importDosenError = err.message || 'Gagal mengimpor dosen.';
      } finally {
        this.importDosenLoading = false;
      }
    },

    async sinkronSemuaMaster() {
      this.syncAllLoading = true;
      try {
        const res = await API.syncMasterAllJadwal();
        await Promise.all([this.loadMatkul(), this.loadRuang(), this.loadDosen()]);
        this.showToast(res.message || `Berhasil menyinkronkan ${res.total_synced || 0} master data kampus.`);
      } catch (err) {
        this.showToast(err.message || 'Gagal menyinkronkan master data kampus.');
      } finally {
        this.syncAllLoading = false;
      }
    },

    async loadAntrean() {
      this.notifLoading = true; this.notifError = '';
      this.notifHasMore = false;
      try {
        const extra = this.notifParams(0);
        this.notifList = await API.getNotifications(this.notifFilter || '', this.notifLimit, extra) || [];
        this.notifHasMore = (this.notifList || []).length >= this.notifLimit;
        this.updateFailedCount();
      } catch (e) {
        this.notifList = [];
        this.notifError = (e && e.code === 'UNAUTHORIZED')
          ? 'Sesi berakhir. Masuk kembali lalu coba lagi.'
          : 'Antrean notifikasi belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.notifLoading = false;
      }
    },

    notifParams(offset) {
      const extra = {};
      if ((this.notifKelas || '').trim()) extra.class_id = this.notifKelas.trim();
      if ((this.notifJenis || '').trim()) extra.event_type = this.notifJenis.trim();
      const since = this.dateTimeLocalParam(this.notifSince);
      const until = this.dateTimeLocalParam(this.notifUntil);
      if (since) extra.since = since;
      if (until) extra.until = until;
      if (offset) extra.offset = offset;
      return extra;
    },

    async loadAntreanMore() {
      if (this.notifLoadingMore || !this.notifHasMore) return;
      this.notifLoadingMore = true;
      try {
        const offset = (this.notifList || []).length;
        const extra = this.notifParams(offset);
        const rows = await API.getNotifications(this.notifFilter || '', this.notifLimit, extra) || [];
        this.notifList = [...(this.notifList || []), ...rows];
        this.notifHasMore = rows.length >= this.notifLimit;
        this.updateFailedCount();
      } finally {
        this.notifLoadingMore = false;
      }
    },

    resetNotifFilter() {
      this.notifFilter = '';
      this.notifKelas = '';
      this.notifJenis = '';
      this.notifSince = '';
      this.notifUntil = '';
      this.notifHasMore = false;
    },

    async loadFailedCount() {
      try {
        const list = await API.getNotifications('FAILED', 50).catch(() => null) || [];
        this.failedCount = Array.isArray(list) ? list.length : 0;
        this.unreadCount = this.failedCount;
      } catch (e) {
        this.failedCount = 0;
        this.unreadCount = 0;
      }
    },

    updateFailedCount() {
      // Badge lonceng mencerminkan hasil filter FAILED penuh (limit 50).
      // Filter lain tak menyentuh badge; angka global dijaga loadFailedCount berkala.
      if ((this.notifFilter || '') === 'FAILED') {
        this.failedCount = (this.notifList || []).length;
        this.unreadCount = this.failedCount;
      }
    },

    async ulangiPesan(id) {
      try {
        await API.retryNotification(id);
        this.showToast('Pengiriman ulang dijadwalkan.');
        await Promise.all([this.loadAntrean(), this.loadFailedCount()]);
      } catch (err) {
        this.showToast(err.message || 'Gagal menjadwalkan ulang.');
      }
    },

    auditDateTimeParam(v) {
      return this.dateTimeLocalParam(v);
    },

    dateTimeLocalParam(v) {
      const s = (v || '').trim();
      if (!s) return '';
      // datetime-local dibaca sebagai zona lokal browser -> kirim instan UTC
      // agar filter server (UTC) tepat tanpa selisih zona.
      const d = new Date(s.length === 16 ? s + ':00' : s);
      if (!isNaN(d)) return d.toISOString();
      return s;
    },

    resetAuditFilter() {
      this.auditKelas = '';
      this.auditAction = '';
      this.auditEntity = '';
      this.auditEntityId = '';
      this.auditActor = '';
      this.auditSince = '';
      this.auditUntil = '';
      this.auditOffset = 0;
      this.auditHasMore = false;
    },

    auditFilterCount() {
      let n = 0;
      if ((this.auditKelas || '').trim()) n++;
      if ((this.auditAction || '').trim()) n++;
      if ((this.auditEntity || '').trim()) n++;
      if ((this.auditEntityId || '').trim()) n++;
      if ((this.auditActor || '').trim()) n++;
      if ((this.auditSince || '').trim()) n++;
      if ((this.auditUntil || '').trim()) n++;
      return n;
    },

    auditChips() {
      const out = [];
      if ((this.auditKelas || '').trim()) out.push({ key: 'auditKelas', label: 'Kelas', value: this.auditKelas.trim() });
      if ((this.auditAction || '').trim()) out.push({ key: 'auditAction', label: 'Aksi', value: this.auditAction.trim() });
      if ((this.auditEntity || '').trim()) out.push({ key: 'auditEntity', label: 'Entitas', value: this.auditEntity.trim() });
      if ((this.auditEntityId || '').trim()) out.push({ key: 'auditEntityId', label: 'ID', value: this.auditEntityId.trim() });
      if ((this.auditActor || '').trim()) out.push({ key: 'auditActor', label: 'Pelaku', value: this.auditActor.trim() });
      if ((this.auditSince || '').trim()) out.push({ key: 'auditSince', label: 'Sejak', value: this.auditSince.trim() });
      if ((this.auditUntil || '').trim()) out.push({ key: 'auditUntil', label: 'Sampai', value: this.auditUntil.trim() });
      return out;
    },

    removeAuditChip(key) {
      if (key && key in this) this[key] = '';
      this.loadAudit();
    },

    auditAdvancedCount() {
      let n = 0;
      if ((this.auditEntityId || '').trim()) n++;
      if ((this.auditActor || '').trim()) n++;
      if ((this.auditSince || '').trim()) n++;
      if ((this.auditUntil || '').trim()) n++;
      return n;
    },

    setAuditPreset(name) {
      this.resetAuditFilter();
      if (name === 'dukungan') {
        this.auditAction = 'SUPPORT_ENTER';
      } else if (name === 'kritis') {
        this.auditAction = 'ROTATE_PORTAL_CODE';
      } else if (name === 'hari-ini') {
        const d = new Date();
        d.setHours(0, 0, 0, 0);
        const pad = (v) => String(v).padStart(2, '0');
        this.auditSince = d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate()) + 'T00:00';
      }
      this.loadAudit();
    },

    auditSeverity(action) {
      const a = String(action || '').toUpperCase();
      if (['SUSPEND_USER', 'SUSPEND_ROLE', 'REVOKE_ROLE', 'ROTATE_PORTAL_CODE', 'REVOKE_TEACHING_EVENT', 'REVOKE', 'DELETE'].includes(a)) return 'Kritis';
      if (['SUPPORT_ENTER', 'SUPPORT_EXIT'].includes(a)) return 'Dukungan';
      return 'Biasa';
    },

    auditSeverityClass(action) {
      const s = this.auditSeverity(action);
      if (s === 'Kritis') return 'bg-red-50 border-red-200 text-red-700';
      if (s === 'Dukungan') return 'bg-amber-50 border-amber-300 text-amber-800';
      return 'bg-slate-100 border-slate-200 text-slate-600';
    },

    auditActorLabel(a) {
      if (!a) return 'Sistem';
      return (a.actor_name || 'Sistem') + (a.actor_role ? ' · ' + a.actor_role : '');
    },

    auditEntityLabel(a) {
      if (!a) return '-';
      let s = a.entity_type || '-';
      if (a.entity_id) s += ' #' + a.entity_id;
      return s;
    },

    parseAuditJSON(v) {
      if (v == null || v === '') return null;
      if (typeof v === 'object') return v;
      try {
        const p = JSON.parse(v);
        return (p && typeof p === 'object') ? p : { value: p };
      } catch (e) {
        return { value: String(v) };
      }
    },

    auditChanges(a) {
      const b = this.parseAuditJSON(a ? a.before_json : null) || {};
      const af = this.parseAuditJSON(a ? a.after_json : null) || {};
      const keys = Array.from(new Set([...Object.keys(b), ...Object.keys(af)])).slice(0, 20);
      return keys.map(k => {
        const bv = b[k] === undefined ? '-' : JSON.stringify(b[k]);
        const av = af[k] === undefined ? '-' : JSON.stringify(af[k]);
        return { key: k, before: String(bv).slice(0, 160), after: String(av).slice(0, 160), changed: bv !== av };
      });
    },

    auditParams(offset) {
      const params = { limit: this.auditLimit, offset: offset || 0 };
      if ((this.auditKelas || '').trim()) params.class_slug = this.auditKelas.trim();
      if ((this.auditAction || '').trim()) params.action = this.auditAction.trim();
      if ((this.auditEntity || '').trim()) params.entity_type = this.auditEntity.trim();
      if ((this.auditEntityId || '').trim()) params.entity_id = this.auditEntityId.trim();
      if ((this.auditActor || '').trim()) params.actor = this.auditActor.trim();
      const since = this.auditDateTimeParam(this.auditSince);
      const until = this.auditDateTimeParam(this.auditUntil);
      if (since) params.since = since;
      if (until) params.until = until;
      return params;
    },

    async loadAudit() {
      this.auditLoading = true; this.auditError = '';
      this.auditOffset = 0; this.auditHasMore = false;
      try {
        const rows = await API.getAudit(this.auditParams(0)).catch(() => null) || [];
        this.auditList = rows;
        this.auditHasMore = rows.length >= this.auditLimit;
      } catch (e) {
        this.auditList = [];
        this.auditError = 'Riwayat Perubahan belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.auditLoading = false;
      }
    },

    async loadAuditMore() {
      if (this.auditLoadingMore || !this.auditHasMore) return;
      this.auditLoadingMore = true;
      try {
        const offset = (this.auditList || []).length;
        const rows = await API.getAudit(this.auditParams(offset)).catch(() => null) || [];
        this.auditList = [...(this.auditList || []), ...rows];
        this.auditOffset = offset;
        this.auditHasMore = rows.length >= this.auditLimit;
      } finally {
        this.auditLoadingMore = false;
      }
    },

    labelStatusBackup(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'VERIFIED') return 'Terverifikasi';
      if (s === 'READY') return 'Siap';
      if (s === 'CREATING') return 'Dibuat';
      if (s === 'FAILED') return 'Gagal';
      return st || '-';
    },

    async buatBackup() {
      const f = this.backupForm;
      if (!f.kelas) { this.backupError = 'Pilih kelas untuk dicadangkan.'; return; }
      if (!((f.alasan || '').trim())) { this.backupError = 'Isi alasan pencadangan. Alasan tercatat di Riwayat Perubahan.'; return; }
      this.backupError = ''; this.backupHasil = null;
      this.backupLoading = true;
      try {
        const payload = { class_slug: f.kelas, reason: f.alasan.trim() };
        if (f.semester) payload.semester_id = Number(f.semester);
        this.backupHasil = await API.createBackup(payload);
        this.showToast('Cadangan berhasil dibuat.');
        await this.loadBackupList();
      } catch (err) {
        this.backupError = err.message || 'Gagal membuat cadangan.';
      } finally {
        this.backupLoading = false;
      }
    },

    async onBackupKelasChange() {
      this.backupForm.semester = '';
      this.backupSemesterList = [];
      const slug = (this.backupForm.kelas || '').trim();
      if (!slug) return;
      try {
        const res = await API.getSemestersResult(slug);
        if (res && res.ok && Array.isArray(res.data)) this.backupSemesterList = res.data;
      } catch (e) {}
    },

    async loadBackupList() {
      this.backupListLoading = true; this.backupListError = '';
      try {
        this.backupList = await API.getBackups('', '');
      } catch (e) {
        this.backupList = [];
        this.backupListError = (e && e.code === 'UNAUTHORIZED')
          ? 'Sesi berakhir. Masuk kembali lalu coba lagi.'
          : 'Daftar cadangan belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.backupListLoading = false;
      }
    },

    async loadBackupRequests() {
      this.backupRequestsLoading = true; this.backupRequestsError = '';
      try { this.backupRequests = await API.getBackupRequests(); }
      catch (e) { this.backupRequests = []; this.backupRequestsError = e.message || 'Permintaan cadangan gagal dimuat.'; }
      finally { this.backupRequestsLoading = false; }
    },

    async executeBackupRequest(item) {
      this.backupRequestExecuting = item.id;
      try {
        await API.executeBackupRequest(item.id);
        this.showToast('Cadangan permintaan berhasil dibuat.');
        await Promise.all([this.loadBackupRequests(), this.loadBackupList()]);
      } catch (e) { this.backupRequestsError = e.message || 'Gagal mengeksekusi permintaan.'; }
      finally { this.backupRequestExecuting = 0; }
    },

    pilihCadanganUntukVerifikasi(b) {
      this.restoreForm.id = String(b.id || '');
      this.selectedBackupRestorable = Boolean(b.restorable);
      this.restoreError = '';
      this.restoreHasil = null;
      this.scopedRestorePreview = null;
      this.scopedRestoreResult = null;
      if (b.restorable) this.previewRestore(b.id);
    },

    async previewRestore(id) {
      this.scopedRestoreLoading = true; this.restoreError = ''; this.scopedRestorePreview = null;
      try { this.scopedRestorePreview = await API.previewScopedRestore(id); }
      catch (e) { this.restoreError = e.message || 'Pratinjau pemulihan gagal.'; }
      finally { this.scopedRestoreLoading = false; }
    },

    async executeRestore() {
      const f = this.restoreForm;
      const preview = this.scopedRestorePreview;
      if (!preview || !preview.can_restore) { this.restoreError = 'Selesaikan keterkaitan lintas kelas lalu muat ulang pratinjau.'; return; }
      if (!f.alasan.trim() || !f.paham) { this.restoreError = 'Isi alasan dan konfirmasi dampak pemulihan.'; return; }
      this.scopedRestoreLoading = true; this.restoreError = '';
      try {
        this.scopedRestoreResult = await API.executeScopedRestore(preview.backup_id, f.alasan.trim(), preview.preview_token);
        this.scopedRestorePreview = null;
        this.showToast('Data akademik berhasil dipulihkan.');
        await this.loadBackupList();
      } catch (e) { this.restoreError = e.message || 'Pemulihan gagal.'; }
      finally { this.scopedRestoreLoading = false; }
    },

    async pulihkanBackup() {
      const f = this.restoreForm;
      if (!f.id) { this.restoreError = 'Isi ID cadangan yang akan diverifikasi.'; return; }
      if (!((f.alasan || '').trim())) { this.restoreError = 'Isi alasan verifikasi.'; return; }
      if (!f.paham) { this.restoreError = 'Centang pernyataan pemahaman dampak terlebih dahulu.'; return; }
      this.restoreError = ''; this.restoreHasil = null;
      this.restoreLoading = true;
      try {
        this.restoreHasil = await API.restoreBackup(f.id, f.alasan.trim());
        this.showToast('Berkas cadangan terverifikasi. Database aktif tidak diubah.');
        this.restoreForm = { id: '', alasan: '', paham: false };
      } catch (err) {
        this.restoreError = err.message || 'Gagal memverifikasi cadangan.';
      } finally {
        this.restoreLoading = false;
      }
    },

    bukaKelasStatus(status) {
      this.resetKelasFilter();
      this.kelasFilter.status = String(status || '');
      this.go('kelas');
    },

    bukaPenggunaStatus(status) {
      this.penggunaTab = 'akun';
      this.penggunaFilter = String(status || '');
      this.go('pengguna');
    },

    async go(v) {
      this.pageState = null;
      if (v === 'pembaruan') v = 'audit';
      if (!this.knownViews.includes(v)) { this.showPageError('404'); return; }
      this.view = v;
      this.drawer = false;
      if (v === 'buat') { this.kelasError = ''; this.updateAutoKelas(); }
      if (v === 'antrean') this.loadAntrean();
      if (v === 'kanal') this.loadKanal();
      if (v === 'backup') { this.backupHasil = null; this.restoreHasil = null; this.loadBackupList(); this.loadBackupRequests(); }
      if (v === 'audit') this.loadAudit();
      if (v === 'pengguna') {
        await this.loadPengguna();
        if (this.penggunaTab === 'penugasan') await this.loadPenugasan();
        if (this.penggunaTab === 'undangan') await this.loadUndangan();
      }
      if (v === 'master-ruangan') { this.loadRuang(); this.loadUsulan('ROOM'); }
      if (v === 'master-matkul') { this.loadMatkul(); this.loadUsulan('COURSE'); }
      if (v === 'master-dosen') { this.loadDosen(); }
      window.scrollTo({ top: 0 });
    },

    isKelasView() {
      return (this.kelasViews || []).includes(this.view);
    },

    lihatKelasBermasalah(jenis) {
      this.resetKelasFilter();
      if (jenis === 'tanpa-km') {
        this.kelasFilter.statusKM = 'none';
      } else if (jenis === 'undangan-pending') {
        this.kelasFilter.statusKM = 'pending';
      }
      this.view = 'kelas';
      this.drawer = false;
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
        if (err.status === 401) this.redirectToLogin();
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
      this.kelasStatusConfirm = null;
      this.view = 'detail';
      window.scrollTo({ top: 0 });
      this.loadDetailSemester(obj.slug);
      this.loadDetailSettings(obj.slug);
    },

    async loadDetailSemester(slug) {
      if (!slug) return;
      this.detailSemesterLoading = true;
      this.detailSemesterError = '';
      this.detailSemesterList = [];
      try {
        const res = await API.getSemestersResult(slug);
        if (res && res.ok) {
          this.detailSemesterList = Array.isArray(res.data) ? res.data : [];
        } else if (res && res.status === 404) {
          this.detailSemesterError = 'Kelas ini belum memiliki semester terdaftar.';
        } else {
          this.detailSemesterError = 'Semester kelas belum dapat dimuat. Periksa koneksi lalu coba lagi.';
        }
      } catch (e) {
        this.detailSemesterError = 'Semester kelas belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.detailSemesterLoading = false;
      }
    },

    bukaAuditKelas(slug) {
      this.auditKelas = slug || '';
      this.auditAction = '';
      this.auditEntity = '';
      this.auditEntityId = '';
      this.auditActor = '';
      this.auditSince = '';
      this.auditUntil = '';
      this.go('audit');
    },

    bukaAntreanKelas(classId) {
      this.notifFilter = '';
      this.notifKelas = String(classId || '');
      this.go('antrean');
    },

    slugifyKelas(nama) {
      return String(nama || '').trim().toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/^-+|-+$/g, '')
        .slice(0, 80);
    },

    daftarProdi() {
      const defaults = [
        // Jurusan Teknik Komputer dan Informatika
        'D4 Teknik Informatika',
        'D3 Teknik Informatika',

        // Jurusan Teknik Elektro
        'D4 Teknik Elektronika',
        'D4 Teknik Otomasi Industri',
        'D4 Teknik Telekomunikasi',
        'D3 Teknik Elektronika',
        'D3 Teknik Listrik',
        'D3 Teknik Telekomunikasi',

        // Jurusan Teknik Mesin
        'D4 Teknik Perancangan dan Konstruksi Mesin',
        'D4 Proses Manufaktur',
        'D3 Teknik Mesin',
        'D3 Teknik Aeronautika',

        // Jurusan Teknik Sipil
        'D4 Teknik Perancangan Jalan dan Jembatan',
        'D4 Teknik Perawatan dan Perbaikan Gedung',
        'D3 Teknik Konstruksi Sipil',
        'D3 Teknik Konstruksi Gedung',
        'S2 Terapan Rekayasa Infrastruktur',

        // Jurusan Teknik Refrigerasi dan Tata Udara
        'D4 Teknik Pendingin dan Tata Udara',
        'D3 Teknik Pendingin dan Tata Udara',

        // Jurusan Teknik Konversi Energi
        'D4 Teknologi Pembangkit Tenaga Listrik',
        'D4 Teknik Konservasi Energi',
        'D3 Teknik Konversi Energi',

        // Jurusan Teknik Kimia
        'D4 Teknik Kimia Produksi Bersih',
        'D3 Teknik Kimia',
        'D3 Analis Kimia',

        // Jurusan Akuntansi
        'D4 Akuntansi',
        'D4 Keuangan Syariah',
        'D4 Akuntansi Manajemen Pemerintahan',
        'D3 Akuntansi',
        'D3 Keuangan dan Perbankan',
        'S2 Terapan Keuangan dan Perbankan Syariah',

        // Jurusan Administrasi Niaga
        'D4 Administrasi Bisnis',
        'D4 Manajemen Pemasaran',
        'D4 Manajemen Aset',
        'D4 Destinasi Pariwisata',
        'D3 Administrasi Bisnis',
        'D3 Manajemen Pemasaran',
        'D3 Usaha Perjalanan Wisata',
        'S2 Terapan Pemasaran, Inovasi, dan Teknologi',

        // Jurusan Bahasa Inggris
        'D4 Bahasa Inggris untuk Komunikasi Bisnis dan Profesional',
        'D3 Bahasa Inggris'
      ];
      const fromList = (this.kelasList || []).map(k => k.prodi).filter(Boolean);
      return Array.from(new Set([...defaults, ...fromList]));
    },

    deriveKelasName(prodi, angkatan, rombel) {
      const p = (prodi || '').trim();
      const a = (angkatan || '').trim();
      const r = (rombel || '').trim().toUpperCase();
      if (!p && !a && !r) return '';

      const pLower = p.toLowerCase();
      const mapKhusus = {
        'd4 teknik informatika': 'D4 TI',
        'd3 teknik informatika': 'D3 TI',
        'd4 akuntansi': 'D4 AK',
        'd3 akuntansi': 'D3 AK',
        'd4 keuangan syariah': 'D4 KS',
        'd3 keuangan dan perbankan': 'D3 KP',
        'd4 administrasi bisnis': 'D4 AB',
        'd3 administrasi bisnis': 'D3 AB',
        'd4 manajemen pemasaran': 'D4 MP',
        'd3 manajemen pemasaran': 'D3 MP',
        'd4 manajemen aset': 'D4 MA',
        'd4 destinasi pariwisata': 'D4 DP',
        'd3 usaha perjalanan wisata': 'D3 UPW',
        'd4 bahasa inggris untuk komunikasi bisnis dan profesional': 'D4 BIKBP',
        'd3 bahasa inggris': 'D3 BI',
        'd4 teknik elektronika': 'D4 TE',
        'd3 teknik elektronika': 'D3 TE',
        'd4 teknik otomasi industri': 'D4 TOI',
        'd4 teknik telekomunikasi': 'D4 TT',
        'd3 teknik telekomunikasi': 'D3 TT',
        'd3 teknik listrik': 'D3 TL',
        'd4 teknologi pembangkit tenaga listrik': 'D4 TPTL',
        'd4 teknik konservasi energi': 'D4 TKE',
        'd3 teknik konversi energi': 'D3 TKE',
        'd4 teknik pendingin dan tata udara': 'D4 TPTU',
        'd3 teknik pendingin dan tata udara': 'D3 TPTU',
        'd4 teknik perancangan jalan dan jembatan': 'D4 TPJJ',
        'd4 teknik perawatan dan perbaikan gedung': 'D4 TPPG',
        'd3 teknik konstruksi sipil': 'D3 TKS',
        'd3 teknik konstruksi gedung': 'D3 TKG',
        'd4 teknik perancangan dan konstruksi mesin': 'D4 TPKM',
        'd4 proses manufaktur': 'D4 PM',
        'd3 teknik mesin': 'D3 TM',
        'd3 teknik aeronautika': 'D3 TA',
        'd4 teknik kimia produksi bersih': 'D4 TKPB',
        'd3 teknik kimia': 'D3 TK',
        'd3 analis kimia': 'D3 AKM'
      };

      let shortProdi = mapKhusus[pLower];
      if (!shortProdi) {
        shortProdi = p;
        const matchJenjang = p.match(/^(D[1-4]|S[1-3])\s+(.+)$/i);
        if (matchJenjang) {
          const jenjang = matchJenjang[1].toUpperCase();
          const sisa = matchJenjang[2].trim();
          const words = sisa.split(/\s+/).filter(w => !['dan', 'untuk', 'atau', 'di', 'ke', 'dari'].includes(w.toLowerCase()));
          const inisial = words.map(w => w[0] ? w[0].toUpperCase() : '').join('');
          shortProdi = `${jenjang} ${inisial || sisa.slice(0, 3).toUpperCase()}`;
        } else if (/^[a-zA-Z\s]+$/.test(p) && p.split(/\s+/).length > 1) {
          const words = p.split(/\s+/).filter(w => !['dan', 'untuk', 'atau', 'di', 'ke', 'dari'].includes(w.toLowerCase()));
          shortProdi = words.map(w => w[0] ? w[0].toUpperCase() : '').join('');
        }
      }

      const parts = [];
      if (shortProdi) parts.push(shortProdi);
      if (a) parts.push(a);
      if (r) parts.push(r);
      return parts.join(' ');
    },

    updateAutoKelas() {
      if (this.kelasForm.kustom) return;
      const nama = this.deriveKelasName(this.kelasForm.prodi, this.kelasForm.angkatan, this.kelasForm.rombel);
      this.kelasForm.nama = nama;
      this.kelasForm.slug = this.slugifyKelas(nama);
    },

    toggleKustomKelas() {
      this.kelasForm.kustom = !this.kelasForm.kustom;
      if (!this.kelasForm.kustom) {
        this.updateAutoKelas();
      }
    },

    resetKelasForm() {
      this.kelasForm = {
        prodi: 'D4 Teknik Informatika',
        angkatan: '2025',
        rombel: 'A',
        nama: 'D4 TI 2025 A',
        slug: 'd4-ti-2025-a',
        kustom: false
      };
      this.updateAutoKelas();
    },

    async simpanKelas() {
      const f = this.kelasForm;
      if (!f.kustom || !((f.nama || '').trim())) { this.updateAutoKelas(); }
      if (!((f.prodi || '').trim())) { this.kelasError = 'Program studi wajib diisi.'; return; }
      if (!((f.angkatan || '').trim())) { this.kelasError = 'Angkatan wajib diisi.'; return; }
      const rombel = ((f.rombel || '').trim() || 'A').toUpperCase();
      if (!/^[A-Z0-9]{1,4}$/.test(rombel)) { this.kelasError = 'Rombel wajib 1–4 karakter huruf/angka (contoh: A).'; return; }
      if (!((f.nama || '').trim())) { this.kelasError = 'Nama kelas wajib diisi.'; return; }

      this.kelasError = '';
      this.kelasSaving = true;

      try {
        const cohortNum = parseInt(String(f.angkatan).trim(), 10) || new Date().getFullYear();
        const payload = {
          name: f.nama.trim(),
          study_program: f.prodi.trim(),
          cohort_year: cohortNum,
          group_label: rombel
        };
        const slugManual = ((f.slug || '').trim());
        if (slugManual) payload.slug = this.slugifyKelas(slugManual);
        await API.createClass(payload);

        this.resetKelasForm();
        this.resetKelasFilter();
        await Promise.all([this.loadKelas(), this.checkBot()]);
        this.view = 'kelas';
        this.showToast('Kelas baru berhasil ditambahkan.');
      } catch (err) {
        this.kelasError = err.message || 'Gagal menyimpan kelas baru.';
      } finally {
        this.kelasSaving = false;
      }
    },

    mintaKonfirmasiStatus(targetStatus) {
      if (!this.kelasAktifObj || !this.kelasAktifObj.slug) return;
      this.kelasStatusConfirm = {
        dari: this.kelasAktifObj.status || 'ACTIVE',
        ke: targetStatus,
        nama: this.kelasAktifObj.nama
      };
    },

    batalKonfirmasiStatus() {
      this.kelasStatusConfirm = null;
    },

    async jalankanUbahStatus() {
      const c = this.kelasStatusConfirm;
      if (!c || !this.kelasAktifObj || !this.kelasAktifObj.slug) return;
      await this.ubahStatusKelas(c.ke);
      this.kelasStatusConfirm = null;
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
        await this.loadDetailSettings(this.kelasAktifObj.slug);
      } catch (err) {
        this.showToast(err.message || 'Gagal merotasi kode portal.');
      }
    },

    async loadDetailSettings(slug) {
      if (!slug) return;
      this.detailSettings = null;
      this.detailSettingsError = '';
      try {
        this.detailSettings = await API.getClassSettings(slug);
        if (!this.detailSettings) this.detailSettingsError = 'Pengaturan kelas belum dapat dimuat.';
      } catch (e) {
        this.detailSettingsError = 'Pengaturan kelas belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      }
    },

    mintaKonfirmasiModePortal(mode) {
      if (!this.kelasAktifObj || !this.kelasAktifObj.slug) return;
      const dari = this.detailSettings ? this.detailSettings.portal_access_mode : '?';
      if (dari === mode) {
        this.showToast(`Mode portal sudah ${mode}.`);
        return;
      }
      this.portalModeConfirm = { ke: mode, dari, nama: this.kelasAktifObj.nama };
    },

    batalKonfirmasiModePortal() {
      this.portalModeConfirm = null;
    },

    async jalankanUbahModePortal() {
      const c = this.portalModeConfirm;
      if (!c) return;
      this.portalModeConfirm = null;
      await this.ubahModePortal(c.ke);
    },

    async ubahModePortal(mode) {
      if (!this.kelasAktifObj || !this.kelasAktifObj.slug) return;
      const alasan = this.isDukunganUntuk(this.kelasAktifObj.slug) && this.dukunganAktif
        ? (this.dukunganAktif.reason || '') : '';
      try {
        const res = await API.setPortalMode(this.kelasAktifObj.slug, mode, alasan);
        if (res && res.changed === false) {
          this.showToast(`Mode portal sudah ${mode}.`);
        } else {
          this.showToast(mode === 'LINK' ? 'Mode tautan aktif. Kode lama tak berlaku.' : 'Mode kode aktif.');
        }
        await this.loadDetailSettings(this.kelasAktifObj.slug);
      } catch (err) {
        this.showToast(err.message || 'Gagal mengubah mode portal.');
      }
    },

    async loadUsulan(kind) {
      this.usulanLoading = true; this.usulanError = '';
      try {
        this.usulanList = await API.getProposals('PENDING', kind || '');
      } catch (e) {
        this.usulanList = [];
        this.usulanError = (e && e.code === 'UNAUTHORIZED')
          ? 'Sesi berakhir. Masuk kembali lalu coba lagi.'
          : 'Daftar usulan belum dapat dimuat.';
      } finally {
        this.usulanLoading = false;
      }
    },

    usulanKindList(kind) {
      return (this.usulanList || []).filter(u => !kind || u.kind === kind);
    },

    mulaiKeputusanUsulan(u, keputusan) {
      this.usulanKeputusan = { id: u.id, keputusan, judul: (u.target_code || u.kind) + ' · ' + (u.class_slug || '') };
      this.usulanCatatan = '';
    },

    async jalankanKeputusanUsulan() {
      const k = this.usulanKeputusan;
      if (!k) return;
      if (k.keputusan === 'reject' && !((this.usulanCatatan || '').trim())) {
        this.showToast('Catatan penolakan wajib diisi.');
        return;
      }
      try {
        await API.decideProposal(k.id, k.keputusan, (this.usulanCatatan || '').trim());
        this.showToast(k.keputusan === 'approve' ? `Usulan ${k.judul} disetujui dan diterapkan.` : `Usulan ${k.judul} ditolak.`);
        this.usulanKeputusan = null;
        await Promise.all([this.loadUsulan(''), this.loadRuang(), this.loadMatkul()]);
      } catch (err) {
        this.showToast(err.message || 'Gagal memutuskan usulan.');
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

    async masukDukungan() {
      const target = (this.kelasList || []).find(k => k.slug === this.kelasAktif);
      if (!target) { this.showToast('Pilih kelas tujuan dulu.'); return; }
      const alasan = (this.dukunganAlasan || '').trim();
      if (alasan.length < 10) { this.showToast('Alasan dukungan minimal 10 karakter.'); return; }
      this.dukunganLoading = true;
      try {
        this.dukunganAktif = await API.supportEnter(target.slug, alasan);
        this.dukunganAlasan = '';
        this.showToast(`Mode Dukungan aktif untuk ${target.nama} (60 menit).`);
        window.scrollTo({ top: 0 });
      } catch (err) {
        this.showToast(err.message || 'Gagal masuk Mode Dukungan.');
      } finally {
        this.dukunganLoading = false;
      }
    },

    async keluarDukungan() {
      this.dukunganLoading = true;
      try {
        await API.supportExit('');
        this.dukunganAktif = null;
        this.dukunganAlasan = '';
        this.showToast('Mode Dukungan dimatikan.');
      } catch (err) {
        this.showToast(err.message || 'Gagal keluar Mode Dukungan.');
      } finally {
        this.dukunganLoading = false;
      }
    },

    async muatDukunganAktif() {
      try {
        this.dukunganAktif = await API.supportActive();
      } catch (e) {
        this.dukunganAktif = null;
      }
    },

    isDukunganUntuk(slug) {
      return !!(this.dukunganAktif && (this.dukunganAktif.class_slug === slug || this.dukunganAktif.slug === slug));
    },

    namaDukungan() {
      const d = this.dukunganAktif;
      if (!d) return '';
      const slug = d.class_slug || d.slug || '';
      const k = (this.kelasList || []).find(x => x.slug === slug);
      return k ? k.nama : (d.class_code || slug);
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
      this.currentUser = null;
      this.kelasList = [];
      this.totalKelas = 0;
      this.dukunganAktif = null;
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
