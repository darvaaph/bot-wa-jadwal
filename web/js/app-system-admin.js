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
      { title: 'UTAMA', items: [
        { id: 'dashboard', label: 'Ringkasan Sistem', icon: 'sa-grid', img: '/assets/icons/home.svg' },
        { id: 'kelas', label: 'Kelas & Semester', icon: 'sa-cap', img: '/assets/icons/classes.svg', active: ['kelas','buat','detail','undang','undang-siap'] },
        { id: 'pengguna', label: 'Pengguna & Peran', icon: 'sa-users', img: '/assets/icons/people.svg' }
      ] },
      { title: 'DATA AKADEMIK', items: [
        { id: 'master-matkul', label: 'Mata Kuliah', icon: 'sa-book', img: '/assets/icons/book.svg' },
        { id: 'master-ruangan', label: 'Ruangan', icon: 'sa-door', img: '/assets/icons/room.svg' },
        { id: 'master-dosen', label: 'Dosen', icon: 'sa-badge', img: '/assets/icons/people.svg' }
      ] },
      { title: 'DATA & SISTEM', items: [
        { id: 'antrean', label: 'Antrean WhatsApp', icon: 'sa-plane', img: '/assets/icons/message-queue.svg' },
        { id: 'backup', label: 'Cadangan & Audit', icon: 'sa-history', img: '/assets/icons/backup.svg', active: ['backup','audit','pembaruan'] }
      ] }
    ],

    kelasViews: ['kelas', 'buat', 'detail', 'jadwal', 'undang', 'undang-siap'],

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
    lingkupKelas: '',
    kelasMenu: null,
    kelasFilterLanjutan: false,
    kelasSemesterMap: {},

    kelasAktif: '',
    kelasAktifObj: null,
    modalBuatTerbuka: false,
    kelasForm: { prodi: 'D4 Teknik Informatika', angkatan: '2025', rombel: 'A', nama: 'D4-TI 1A', slug: 'd4-ti-1a', portal_access_mode: 'LINK', kustom: false },
    kelasError: '',
    kelasSaving: false,
    kelasStatusConfirm: null,
    detailSemesterList: [],
    detailSemesterLoading: false,
    detailSemesterError: '',
    detailSettings: null,
    detailSettingsError: '',
    portalModeConfirm: null,
    detailMenu: null,
    dukunganFormTerbuka: false,
    portalCodeReveal: '',
    kmPenugasan: null,
    kmPenugasanLoading: false,
    smBuatTerbuka: false,
    smBuatForm: { academic_year: '', term: 'Ganjil', starts_on: '', ends_on: '' },
    smBuatError: '',
    smBuatSaving: false,
    smImporTerbuka: false,
    smImporFile: null,
    smImporHasil: null,
    smImporError: '',
    smImporSaving: false,
    smImporDragOver: false,
    smPreviewMap: {},
    smPratinjau: null,
    smAktifkan: null,
    smHapusKonfirmasi: null,
    smHapusSaving: false,
    smHapusError: '',
    smMenu: null,
    smArsipTerbuka: null,
    smImporSemester: null,

    modalTambahOffering: false,
    modalDaftarOffering: false,
    offeringSemesterTarget: null,
    offeringForm: { course_code: '', activity_type: 'TEORI', display_name: '', lecturer_codes: [] },
    offeringFormError: '',
    offeringFormSaving: false,
    offeringQuickMatkulOpen: false,
    offeringQuickMatkul: { kode: '', nama: '' },
    offeringQuickMatkulSaving: false,
    offeringQuickMatkulError: '',
    offeringQuickDosenOpen: false,
    offeringQuickDosen: { kode: '', nama: '' },
    offeringQuickDosenSaving: false,
    offeringQuickDosenError: '',

    jadwalKelasSlug: '',
    jadwalSemesterId: '',
    jadwalSemester: null,
    jadwalSemesterOptions: [],
    jadwalOfferings: [],
    jadwalPatterns: [],
    jadwalPreviewSem: null,
    jadwalLoading: false,
    jadwalError: '',
    jadwalHari: 0,
    jadwalQ: '',
    jadwalModal: false,
    jadwalForm: { id: '', version: 0, offeringId: '', day: '1', start: '', duration: 100, end: '', roomId: '', link: '', reason: '', effectiveDate: '' },
    jadwalFormError: '',
    jadwalFormSaving: false,
    jadwalPreview: null,
    jadwalPreviewLoading: false,
    jadwalHapus: null,
    jadwalHapusError: '',

    undangNomor: '',
    undangError: '',
    undangLoading: false,
    undangLink: '',

    dukunganAlasan: '',
    dukunganAktif: null,
    dukunganLoading: false,

    penggunaTab: 'akun',
    penggunaQ: '',
    penggunaPage: 1,
    penggunaPerPage: 10,
    undangPenggunaTerbuka: false,
    penggunaMenu: null,
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
    undanganSANomor: '',
    undanganSAError: '',
    undanganSALoading: false,
    undanganSAResult: null,

    modalUndangPengguna: false,
    modalUndangPeran: 'SYSTEM_ADMIN',
    modalUndangKelas: '',
    modalUndangNomor: '',
    modalUndangLoading: false,
    modalUndangError: '',
    modalUndangResult: null,

    modalResetPassword: false,
    resetPasswordTarget: null,
    resetPasswordIsSelf: false,
    resetPasswordForm: { password: '', confirm: '', reason: '', show: false },
    resetPasswordLoading: false,
    resetPasswordError: '',

    modalProfil: false,
    profilTarget: null,

    modalUbahPeran: false,
    ubahPeranTarget: null,
    ubahPeranForm: { role: '', class_slug: '', reason: '' },
    ubahPeranLoading: false,
    ubahPeranError: '',

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
    matkulFormLoading: false,
    matkulQ: '',
    matkulStatusFilter: '',
    matkulEdit: null,
    matkulEditError: '',
    matkulEditLoading: false,
    matkulMenu: null,
    matkulStatusConfirm: null,
    matkulStatusLoading: false,
    modalTambahMatkul: false,
    modalImportMatkul: false,
    importTab: 'jadwal',
    importCsvText: '',
    importFileName: '',
    importFileSize: '',
    importDragOver: false,
    importLoading: false,
    importSuccess: '',
    modalTolakUsulan: false,
    usulanDitolakTarget: null,
    usulanAlasanTolak: '',
    usulanRejectLoading: false,
    dosenList: [],
    dosenLoading: false,
    dosenError: '',
    dosenForm: { kode: '', nama: '' },
    dosenFormError: '',
    dosenQ: '',
    dosenStatusFilter: '',
    dosenEdit: null,
    dosenEditError: '',
    dosenMenu: null,
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

    lingkupLabel() {
      const slug = (this.lingkupKelas || '').trim();
      if (!slug) return 'Seluruh Kampus';
      const target = (this.kelasList || []).find(k => k.slug === slug);
      return target ? target.nama : 'Seluruh Kampus';
    },

    onLingkupChange() {
      const slug = (this.lingkupKelas || '').trim();
      if (!slug) {
        this.resetKelasFilter();
        this.go('kelas');
        return;
      }
      const target = (this.kelasList || []).find(k => k.slug === slug);
      if (target) this.bukaDetail(target);
      else this.go('kelas');
    },

    toggleKelasMenu(slug) {
      this.kelasMenu = this.kelasMenu === slug ? null : slug;
    },

    async muatSemesterSemua() {
      const slugs = (this.kelasList || []).map(k => k.slug).filter(Boolean);
      await Promise.all(slugs.map(slug => this.muatSemesterKelas(slug)));
    },

    async muatSemesterKelas(slug) {
      if (!slug || this.kelasSemesterMap[slug]) return;
      this.kelasSemesterMap[slug] = { loading: true, list: [] };
      try {
        const res = await API.getSemestersResult(slug);
        this.kelasSemesterMap[slug] = { loading: false, list: (res && res.ok && Array.isArray(res.data)) ? res.data : [] };
      } catch (e) {
        this.kelasSemesterMap[slug] = { loading: false, list: [] };
      }
    },

    semesterAktifKelas(slug) {
      const entry = this.kelasSemesterMap[slug];
      const list = (entry && entry.list) || [];
      return list.find(s => String(s.status || '').toUpperCase() === 'ACTIVE') || null;
    },

    semesterTampilKelas(slug) {
      return this.semesterAktifKelas(slug) || (((this.kelasSemesterMap[slug] || {}).list || [])[0]) || null;
    },

    labelSemester(s) {
      if (!s) return '-';
      if (s.name || s.label) return s.name || s.label;
      const t = String(s.term || '');
      const term = t ? t.charAt(0).toUpperCase() + t.slice(1).toLowerCase() : '';
      return [s.academic_year || '', term].filter(Boolean).join(' ') || ('Semester #' + (s.id ?? ''));
    },

    jadwalChipKelas(slug) {
      const entry = this.kelasSemesterMap[slug];
      if (!entry || entry.loading) return null;
      const list = entry.list || [];
      if (!list.length) return { text: 'Belum ada semester', tone: 'none' };
      const aktif = this.semesterAktifKelas(slug);
      if (aktif && aktif.published_at) return { text: 'Jadwal Terbit', tone: 'ok' };
      return { text: 'Draf / Belum Lengkap', tone: 'warn' };
    },

    bannerSemester() {
      const hitung = {};
      const contoh = {};
      (this.kelasList || []).forEach(k => {
        if (String(k.status || '').toUpperCase() !== 'ACTIVE') return;
        const s = this.semesterAktifKelas(k.slug);
        if (!s) return;
        const label = this.labelSemester(s);
        hitung[label] = (hitung[label] || 0) + 1;
        if (!contoh[label]) contoh[label] = k.slug;
      });
      let terbaik = '', jumlah = 0;
      Object.keys(hitung).forEach(label => { if (hitung[label] > jumlah) { jumlah = hitung[label]; terbaik = label; } });
      return { label: terbaik, slug: contoh[terbaik] || '', jumlah };
    },

    kelolaPeriode() {
      const b = this.bannerSemester();
      if (b.slug) { this.kelasMenu = null; this.bukaDetail(b.slug); return; }
      this.showToast('Belum ada semester aktif. Buka detail kelas untuk mengelola periode.');
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

    saCrumb() {
      if (this.view === 'jadwal') {
        return (this.isJadwalDraft && this.isJadwalDraft()) ? 'Jadwal Semester Draf' : 'Jadwal Mingguan';
      }
      if (this.view === 'detail' && this.kelasAktifObj) {
        const nama = this.kelasAktifObj.nama || this.kelasAktifObj.slug || 'Detail';
        return nama + (this.isDukunganUntuk(this.kelasAktifObj.slug) ? ' (Mode Dukungan)' : '');
      }
      const map = {
        dashboard: 'Ringkasan',
        kelas: 'Kelas & Semester', buat: 'Tambah kelas', detail: 'Kelas & Semester', jadwal: 'Jadwal Mingguan',
        undang: 'Undang KM', 'undang-siap': 'Undangan siap',
        pengguna: 'Pengguna & Peran',
        'master-matkul': 'Mata Kuliah', 'master-ruangan': 'Ruangan', 'master-dosen': 'Dosen',
        antrean: 'Antrean WhatsApp',
        backup: 'Cadangan & Audit', audit: 'Cadangan & Audit', pembaruan: 'Cadangan & Audit'
      };
      return map[this.view] || this.saTitle(this.view);
    },

    dukunganKelasSlug() {
      const d = this.dukunganAktif;
      return (d && (d.class_slug || d.slug)) || '';
    },

    dukunganJam() {
      try {
        const d = new Date(this.dukunganAktif && this.dukunganAktif.expires_at);
        if (isNaN(d)) return '-';
        const p = (v) => String(v).padStart(2, '0');
        return p(d.getHours()) + ':' + p(d.getMinutes());
      } catch (e) { return '-'; }
    },

    dukunganSisa() {
      try {
        const d = new Date(this.dukunganAktif && this.dukunganAktif.expires_at);
        const ms = d.getTime() - Date.now();
        if (isNaN(ms)) return '';
        if (ms <= 0) return 'sudah berakhir';
        const mnt = Math.floor(ms / 60000);
        if (mnt < 60) return 'sisa ' + mnt + ' menit';
        return 'sisa ' + Math.floor(mnt / 60) + ' jam ' + (mnt % 60) + ' menit';
      } catch (e) { return ''; }
    },

    masterTotal() {
      return (this.matkulList || []).length + (this.ruangList || []).length + (this.dosenList || []).length;
    },
    metricTotalKelas() {
      return this.botStatusDetails?.classes_summary?.total || this.totalKelas || (this.kelasList || []).length || 0;
    },

    metricActiveKelas() {
      return this.botStatusDetails?.classes_summary?.active || this.activeKelasCount || (this.kelasList || []).filter(k => k.status === 'ACTIVE').length || 0;
    },

    metricTotalUsers() {
      return this.botStatusDetails?.users_summary?.total || 0;
    },

    metricUsersBreakdown() {
      const u = this.botStatusDetails?.users_summary;
      if (u && (u.km_count || u.pj_count || u.admin_count)) {
        return (u.km_count || 0) + ' KM • ' + (u.pj_count || 0) + ' PJ • ' + (u.admin_count || 0) + ' Admin';
      }
      return (u?.active || 0) + ' akun aktif • ' + (u?.suspended || 0) + ' ditangguhkan';
    },

    metricTotalTasks() {
      return this.botStatusDetails?.tasks_summary?.total || 0;
    },

    metricTasksBreakdown() {
      const t = this.botStatusDetails?.tasks_summary;
      return (t?.published || 0) + ' diterbitkan • ' + (t?.draft || 0) + ' draf';
    },

    metricTotalSchedules() {
      const e = this.botStatusDetails?.events_summary;
      return (e?.total_patterns || 0) + (e?.total_events || 0);
    },

    metricSchedulesBreakdown() {
      const e = this.botStatusDetails?.events_summary;
      return (e?.total_patterns || 0) + ' reguler • ' + (e?.total_events || 0) + ' pengganti';
    },

    backupPendingCount() {
      return (this.backupRequests || []).filter(r => String(r.status || '').toUpperCase() === 'PENDING').length;
    },

    perluTindakanCount() {
      let n = 0;
      if ((this.failedCount || 0) > 0) n++;
      if (this.backupPendingCount() > 0) n++;
      return n;
    },

    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed;
      try { localStorage.setItem('asterisk:sidebar:collapsed', this.sidebarCollapsed ? '1' : '0'); } catch (e) {}
    },

    knownViews: ['dashboard', 'kelas', 'buat', 'detail', 'jadwal', 'undang', 'undang-siap', 'pengguna', 'antrean', 'audit', 'pembaruan', 'master-ruangan', 'master-matkul', 'master-dosen', 'backup'],

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

      await AsteriskShell.mount('sa', ['dashboard','kelas','jadwal','undang','dukungan','antrean','backup','pengguna','ruangan','matkul','dosen']);
      await this.loadPartials([
        ['sa-dashboard', '/partials/system-admin/view-dashboard.html'],
        ['sa-kelas', '/partials/system-admin/view-kelas.html'],
        ['sa-jadwal', '/partials/system-admin/view-jadwal.html'],
        ['sa-undang', '/partials/system-admin/view-undang.html'],
        ['sa-dukungan', '/partials/system-admin/view-dukungan.html'],
        ['sa-antrean', '/partials/system-admin/view-antrean.html'],
        ['sa-backup', '/partials/system-admin/view-backup.html'],
        ['sa-pengguna', '/partials/system-admin/view-pengguna.html'],
        ['sa-ruangan', '/partials/system-admin/view-master-ruangan.html'],
        ['sa-matkul', '/partials/system-admin/view-master-matkul.html'],
        ['sa-dosen', '/partials/system-admin/view-master-dosen.html'],
        ['sa-banner', '/partials/system-admin/support-banner.html']
      ]);

      await Promise.all([this.checkBot(), this.loadKelas(), this.loadFailedCount(), this.loadMatkul(), this.loadRuang(), this.loadDosen(), this.loadBackupRequests(), this.muatDukunganAktif()]);
      await this.muatSemesterSemua();
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

    labelPeran(role) {
      const r = String(role || '').toUpperCase();
      if (r === 'SYSTEM_ADMIN') return 'System Admin';
      if (r === 'KM') return 'Ketua Murid';
      if (r === 'PJ') return 'PJ';
      return role || '-';
    },

    kelasPenugasan(a) {
      if (!a) return '-';
      if (String(a.role || '').toUpperCase() === 'SYSTEM_ADMIN') return 'Semua kelas';
      return a.class_slug || a.class_code || this.scopePenugasan(a);
    },

    undanganMenunggu(u) {
      if (!u || String(u.status || '').toUpperCase() !== 'PENDING') return false;
      if (u.is_expired) return false;
      return this.undanganSisaWaktu(u.expires_at) !== 'kedaluwarsa';
    },

    penggunaAktifCount() {
      return (this.penggunaList || []).filter(u => String(u.status || '').toUpperCase() === 'ACTIVE').length;
    },

    undanganMenungguCount() {
      return (this.undanganList || []).filter(u => this.undanganMenunggu(u)).length;
    },

    barisPenggunaCount() {
      return this.totalBarisPengguna();
    },

    semuaBarisPengguna() {
      const rows = [];
      (this.filteredPenugasan() || []).forEach(a => {
        rows.push({ id: 'a-' + a.id, kind: 'assignment', data: a });
      });
      (this.filteredUndangan() || []).forEach(u => {
        rows.push({ id: 'u-' + u.id, kind: 'invitation', data: u });
      });
      return rows;
    },

    totalBarisPengguna() {
      return this.semuaBarisPengguna().length;
    },

    totalHalamanPengguna() {
      return Math.max(1, Math.ceil(this.totalBarisPengguna() / (this.penggunaPerPage || 10)));
    },

    barisPenggunaPaginated() {
      const perPage = this.penggunaPerPage || 10;
      const totalHalaman = this.totalHalamanPengguna();
      const page = Math.max(1, Math.min(this.penggunaPage || 1, totalHalaman));
      const start = (page - 1) * perPage;
      return this.semuaBarisPengguna().slice(start, start + perPage);
    },

    mulaiItemPengguna() {
      const total = this.totalBarisPengguna();
      if (total === 0) return 0;
      const totalHalaman = this.totalHalamanPengguna();
      const page = Math.max(1, Math.min(this.penggunaPage || 1, totalHalaman));
      return (page - 1) * (this.penggunaPerPage || 10) + 1;
    },

    akhirItemPengguna() {
      const total = this.totalBarisPengguna();
      if (total === 0) return 0;
      const totalHalaman = this.totalHalamanPengguna();
      const page = Math.max(1, Math.min(this.penggunaPage || 1, totalHalaman));
      return Math.min(page * (this.penggunaPerPage || 10), total);
    },

    keHalamanPengguna(p) {
      const target = Math.max(1, Math.min(p, this.totalHalamanPengguna()));
      this.penggunaPage = target;
      this.penggunaMenu = null;
    },

    halamanArrayPengguna() {
      const total = this.totalHalamanPengguna();
      const curr = this.penggunaPage;
      if (total <= 7) {
        const pages = [];
        for (let i = 1; i <= total; i++) pages.push(i);
        return pages;
      }
      const pages = [];
      pages.push(1);
      if (curr > 3) pages.push('...');
      const start = Math.max(2, curr - 1);
      const end = Math.min(total - 1, curr + 1);
      for (let i = start; i <= end; i++) {
        pages.push(i);
      }
      if (curr < total - 2) pages.push('...');
      pages.push(total);
      return pages;
    },

    cocokCariPengguna(teks) {
      const q = (this.penggunaQ || '').trim().toLowerCase();
      if (!q) return true;
      return String(teks || '').toLowerCase().includes(q);
    },

    filteredPenugasan() {
      return (this.penugasanList || []).filter(a =>
        this.cocokCariPengguna([a.display_name, a.identity_key, a.role, a.class_slug, a.class_code].filter(Boolean).join(' '))
      );
    },

    filteredUndangan() {
      const list = (this.undanganList || []).filter(u => {
        // Undangan yang sudah diterima (ACCEPTED) sudah terdaftar sebagai penugasan aktif di atas
        if (String(u.status || '').toUpperCase() === 'ACCEPTED') return false;
        return this.cocokCariPengguna([u.invited_identity_key, u.role, u.class_slug].filter(Boolean).join(' '));
      });
      const bobot = (u) => this.undanganMenunggu(u) ? 0 : 1;
      return list.slice().sort((x, y) => bobot(x) - bobot(y));
    },

    penggunaUntukPenugasan(a) {
      if (!a) return null;
      return (this.penggunaList || []).find(u => String(u.identity_key || '') === String(a.identity_key || '')) || null;
    },

    togglePenggunaMenu(id) {
      this.penggunaMenu = this.penggunaMenu === id ? null : id;
    },

    bukaAuditPengguna(identitas) {
      this.penggunaMenu = null;
      this.auditKelas = '';
      this.auditAction = '';
      this.auditEntity = '';
      this.auditEntityId = '';
      this.auditActor = String(identitas || '');
      this.auditSince = '';
      this.auditUntil = '';
      this.go('audit');
    },

    resetSemuaFilterPengguna() {
      this.penggunaQ = '';
      this.penugasanStatus = '';
      this.penugasanRole = '';
      this.penugasanKelas = '';
      this.undanganStatus = '';
      this.undanganRole = '';
      this.undanganKelas = '';
      this.penggunaPage = 1;
      this.muatPenggunaSemua();
    },

    async muatPenggunaSemua() {
      this.penggunaMenu = null;
      await Promise.all([this.loadPengguna(), this.loadPenugasan(), this.loadUndangan()]);
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

    bukaModalUndangPengguna() {
      this.modalUndangPengguna = true;
      this.modalUndangPeran = 'SYSTEM_ADMIN';
      this.modalUndangKelas = (this.kelasList && this.kelasList.length) ? this.kelasList[0].slug : '';
      this.modalUndangNomor = '';
      this.modalUndangLoading = false;
      this.modalUndangError = '';
      this.modalUndangResult = null;
    },

    tutupModalUndangPengguna() {
      this.modalUndangPengguna = false;
      this.modalUndangError = '';
      this.modalUndangResult = null;
    },

    async kirimModalUndangPengguna() {
      const nomor = (this.modalUndangNomor || '').trim();
      const cleanPhone = nomor.replace(/\D/g, '');
      if (cleanPhone.length < 9) {
        this.modalUndangError = 'Nomor WhatsApp tidak valid (minimal 9 digit).';
        return;
      }
      const role = this.modalUndangPeran;
      if ((role === 'KM' || role === 'PJ') && !this.modalUndangKelas) {
        this.modalUndangError = 'Pilih kelas tujuan untuk peran ' + (role === 'KM' ? 'Ketua Murid' : 'PJ') + '.';
        return;
      }

      this.modalUndangError = '';
      this.modalUndangLoading = true;
      try {
        const payload = {
          role: role,
          invited_identity_key: nomor
        };
        if (role !== 'SYSTEM_ADMIN') {
          payload.class_slug = this.modalUndangKelas;
        }
        const res = await API.createInvitation(payload);
        const token = (res && res.token) ? res.token : '';
        const link = token ? `${window.location.origin}/invite.html?token=${encodeURIComponent(token)}` : '';
        this.modalUndangResult = {
          id: res && res.invitation_id,
          expires_at: res && res.expires_at,
          link: link
        };
        await this.loadUndangan();
        if (link) {
          this.copyText(link, 'Tautan undangan disalin ke clipboard.');
        }
      } catch (err) {
        this.modalUndangError = err.message || 'Gagal membuat undangan.';
      } finally {
        this.modalUndangLoading = false;
      }
    },

    async salinLinkAtauBuatBaru(u) {
      if (u._link) {
        this.copyText(u._link, 'Tautan undangan disalin ke clipboard.');
        return;
      }
      try {
        const payload = {
          role: u.role,
          invited_identity_key: u.invited_identity_key
        };
        if (u.class_slug) payload.class_slug = u.class_slug;
        const res = await API.createInvitation(payload);
        const token = (res && res.token) ? res.token : '';
        const link = token ? `${window.location.origin}/invite.html?token=${encodeURIComponent(token)}` : '';
        if (link) {
          u._link = link;
          this.copyText(link, 'Tautan baru dibuat dan disalin ke clipboard.');
          await this.loadUndangan();
        } else {
          this.showToast('Undangan dibuat ulang.');
          await this.loadUndangan();
        }
      } catch (err) {
        this.showToast(err.message || 'Gagal memperbarui tautan undangan.');
      }
    },

    async aktifkanCepatPengguna(a) {
      const u = this.penggunaUntukPenugasan(a);
      if (u) {
        try {
          await API.recoverUser(u.id, 'Diaktifkan kembali oleh Administrator');
          this.showToast('Akun ' + (u.display_name || u.identity_key) + ' berhasil diaktifkan kembali.');
          await this.muatPenggunaSemua();
        } catch (err) {
          this.showToast(err.message || 'Gagal mengaktifkan akun.');
        }
      } else {
        this.mulaiAksiPenugasan(a, 'cabut');
      }
    },

    bukaModalResetPassword(target, isSelf = false) {
      this.penggunaMenu = null;
      this.resetPasswordTarget = target;
      this.resetPasswordIsSelf = isSelf;
      this.resetPasswordForm = {
        password: '',
        confirm: '',
        reason: isSelf ? 'Perubahan kata sandi akun mandiri' : '',
        show: false
      };
      this.resetPasswordLoading = false;
      this.resetPasswordError = '';
      this.modalResetPassword = true;
    },

    tutupModalResetPassword() {
      this.modalResetPassword = false;
      this.resetPasswordTarget = null;
      this.resetPasswordError = '';
    },

    async simpanResetPassword() {
      const f = this.resetPasswordForm;
      const pwd = (f.password || '').trim();
      if (pwd.length < 12) {
        this.resetPasswordError = 'Kata sandi minimal 12 karakter.';
        return;
      }
      if (this.resetPasswordIsSelf && pwd !== (f.confirm || '').trim()) {
        this.resetPasswordError = 'Konfirmasi kata sandi tidak cocok.';
        return;
      }
      if (!this.resetPasswordIsSelf && !((f.reason || '').trim())) {
        this.resetPasswordError = 'Alasan tindakan wajib diisi untuk catatan audit sistem.';
        return;
      }

      this.resetPasswordLoading = true;
      this.resetPasswordError = '';
      try {
        let targetId = null;
        if (this.resetPasswordIsSelf) {
          targetId = this.currentUser ? this.currentUser.id : null;
        } else if (this.resetPasswordTarget) {
          const u = this.resetPasswordTarget.identity_key ? this.penggunaUntukPenugasan(this.resetPasswordTarget) : this.resetPasswordTarget;
          targetId = u ? u.id : this.resetPasswordTarget.id;
        }

        if (!targetId) {
          throw new Error('ID pengguna target tidak ditemukan.');
        }

        const reason = (f.reason || '').trim() || (this.resetPasswordIsSelf ? 'Pembaruan kata sandi akun administrator' : 'Reset kata sandi oleh System Admin');
        await API.recoverUser(targetId, reason, pwd);

        if (this.resetPasswordIsSelf) {
          this.showToast('Kata sandi berhasil diubah! Sesi aktif diperbarui. Mengalihkan ke login...', 4000);
          this.modalResetPassword = false;
          setTimeout(() => {
            this.logout();
          }, 1500);
        } else {
          this.showToast('Kata sandi berhasil di-reset. Sesi lama pengguna telah dicabut.');
          this.modalResetPassword = false;
          await this.muatPenggunaSemua();
        }
      } catch (err) {
        this.resetPasswordError = err.message || 'Gagal mengubah kata sandi.';
      } finally {
        this.resetPasswordLoading = false;
      }
    },

    bukaModalProfil(target) {
      this.penggunaMenu = null;
      this.profilTarget = target || this.currentUser || {};
      this.modalProfil = true;
    },

    tutupModalProfil() {
      this.modalProfil = false;
      this.profilTarget = null;
    },

    bukaModalUbahPeran(a) {
      this.penggunaMenu = null;
      this.ubahPeranTarget = a;
      this.ubahPeranForm = {
        role: a.role || 'KM',
        class_slug: a.class_slug || (this.kelasList && this.kelasList.length ? this.kelasList[0].slug : ''),
        reason: ''
      };
      this.ubahPeranLoading = false;
      this.ubahPeranError = '';
      this.modalUbahPeran = true;
    },

    tutupModalUbahPeran() {
      this.modalUbahPeran = false;
      this.ubahPeranTarget = null;
      this.ubahPeranError = '';
    },

    async simpanUbahPeran() {
      const a = this.ubahPeranTarget;
      if (!a) return;
      const f = this.ubahPeranForm;
      if (!((f.reason || '').trim())) {
        this.ubahPeranError = 'Alasan perubahan peran wajib diisi untuk catatan audit.';
        return;
      }
      this.ubahPeranLoading = true;
      this.ubahPeranError = '';
      try {
        // Cabut penugasan lama terlebih dahulu
        await API.changeAssignmentStatus(a.id, 'cabut', f.reason.trim(), false);
        // Buat undangan / penugasan baru dengan peran dan kelas baru
        const payload = {
          role: f.role,
          invited_identity_key: a.identity_key
        };
        if (f.role !== 'SYSTEM_ADMIN') {
          payload.class_slug = f.class_slug;
        }
        await API.createInvitation(payload);
        this.showToast(`Penugasan ${a.display_name || a.identity_key} dirotasi ke ${this.labelPeran(f.role)}.`);
        this.modalUbahPeran = false;
        await this.muatPenggunaSemua();
      } catch (err) {
        this.ubahPeranError = err.message || 'Gagal merotasi peran pengguna.';
      } finally {
        this.ubahPeranLoading = false;
      }
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
      this.matkulFormLoading = false;
      this.modalTambahMatkul = true;
    },

    tutupModalTambahMatkul() {
      if (this.matkulFormLoading) return;
      this.modalTambahMatkul = false;
      this.matkulFormError = '';
      this.matkulFormLoading = false;
    },

    bukaModalImportMatkul() {
      this.modalImportMatkul = true;
      this.importTab = 'jadwal';
      this.importCsvText = '';
      this.importFileName = '';
      this.importFileSize = '';
      this.importDragOver = false;
      this.importLoading = false;
      this.importError = '';
      this.importSuccess = '';
    },

    tutupModalImportMatkul() {
      this.modalImportMatkul = false;
      this.importLoading = false;
      this.importError = '';
      this.importSuccess = '';
      this.importDragOver = false;
    },

    resetImportFile() {
      this.importCsvText = '';
      this.importFileName = '';
      this.importFileSize = '';
      this.importError = '';
      const fileInput = document.getElementById('sa-import-matkul-file');
      if (fileInput) fileInput.value = '';
    },

    handleImportFileSelect(event) {
      const file = event.target.files && event.target.files[0];
      if (file) {
        this.readFileContent(file);
      }
    },

    handleImportDrop(event) {
      this.importDragOver = false;
      const file = event.dataTransfer && event.dataTransfer.files && event.dataTransfer.files[0];
      if (file) {
        this.readFileContent(file);
      }
    },

    readFileContent(file) {
      this.importError = '';
      if (file.size > 2 * 1024 * 1024) {
        this.importError = 'Ukuran berkas melebihi batas maksimal 2MB.';
        return;
      }
      this.importFileName = file.name;
      this.importFileSize = (file.size / 1024).toFixed(1) + ' KB';
      const reader = new FileReader();
      reader.onload = (e) => {
        this.importCsvText = e.target.result || '';
        const parsed = this.parsedImportCourses();
        if (parsed.length === 0) {
          this.importError = 'Tidak ditemukan baris data mata kuliah yang valid di dalam berkas.';
        }
      };
      reader.onerror = () => {
        this.importError = 'Gagal membaca berkas yang dipilih.';
      };
      reader.readAsText(file);
    },

    unduhTemplateMatkul() {
      const csvContent = "kode,nama_matkul,status\n25IF1101,Algoritma dan Pemrograman,ACTIVE\n25TI1102,Komputasi Kognitif,ACTIVE\n25KU0002,Pendidikan Pancasila,INACTIVE\n";
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.setAttribute('href', url);
      link.setAttribute('download', 'template_mata_kuliah.csv');
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
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
            name: (item.name || item.nama || item.matkul || '').trim(),
            status: ((item.status || 'ACTIVE').toUpperCase() === 'INACTIVE' ? 'INACTIVE' : 'ACTIVE')
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

        const parts = line.split(delimiter).map(p => p.trim());
        if (parts.length >= 2) {
          const code = parts[0];
          // Skip header row if detected
          const lowerCode = code.toLowerCase();
          if (lowerCode === 'kode' || lowerCode === 'code' || lowerCode === 'kode_matkul') {
            continue;
          }
          let name = parts[1];
          let status = 'ACTIVE';
          if (parts.length >= 3) {
            const rawStatus = parts[2].toUpperCase();
            if (rawStatus === 'INACTIVE' || rawStatus === 'TIDAK AKTIF' || rawStatus === 'NONAKTIF') {
              status = 'INACTIVE';
            } else if (rawStatus === 'ACTIVE' || rawStatus === 'AKTIF') {
              status = 'ACTIVE';
            }
          }
          if (code && name && !seen.has(code.toUpperCase())) {
            seen.add(code.toUpperCase());
            out.push({ code, name, status });
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
        this.importFileName = '';
        this.importFileSize = '';
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
      const kode = (f.kode || '').trim().toUpperCase();
      const nama = (f.nama || '').trim();
      if (!kode || !nama) {
        this.matkulFormError = 'Kode dan nama mata kuliah wajib diisi.';
        return;
      }
      this.matkulFormError = '';
      this.matkulFormLoading = true;
      try {
        await API.createMasterCourse({ code: kode, name: nama });
        this.matkulForm = { kode: '', nama: '' };
        await this.loadMatkul();
        this.modalTambahMatkul = false;
        this.showToast('Mata kuliah berhasil ditambahkan.');
      } catch (err) {
        this.matkulFormError = err.message || 'Gagal menambah mata kuliah.';
      } finally {
        this.matkulFormLoading = false;
      }
    },

    mulaiUbahMatkul(m) {
      const st = (m.status || 'ACTIVE').toUpperCase();
      this.matkulEdit = {
        id: m.id,
        kode: m.code,
        nama: m.name || '',
        status: st,
        _originalStatus: st
      };
      this.matkulEditError = '';
      this.matkulEditLoading = false;
    },

    batalUbahMatkul() {
      if (this.matkulEditLoading) return;
      this.matkulEdit = null;
      this.matkulEditError = '';
      this.matkulEditLoading = false;
    },

    async simpanUbahMatkul() {
      const f = this.matkulEdit;
      if (!f) return;
      const nama = (f.nama || '').trim();
      if (!nama) {
        this.matkulEditError = 'Nama mata kuliah wajib diisi.';
        return;
      }
      const status = (f.status || 'ACTIVE').toUpperCase();

      // OPSI A: Jika status diubah dari ACTIVE ke INACTIVE, tampilkan dialog konfirmasi Layar 05
      if (status === 'INACTIVE' && f._originalStatus === 'ACTIVE') {
        this.matkulStatusConfirm = {
          id: f.id,
          kode: f.kode,
          nama: nama,
          dari: 'ACTIVE',
          ke: 'INACTIVE',
          fromModalEdit: true
        };
        return;
      }

      this.matkulEditError = '';
      this.matkulEditLoading = true;
      try {
        await API.patchMasterCourse(f.id, { name: nama, status: status });
        this.showToast(`Mata kuliah ${f.kode} berhasil diperbarui.`);
        this.matkulEdit = null;
        await this.loadMatkul();
      } catch (err) {
        this.matkulEditError = err.message || 'Gagal mengubah mata kuliah.';
      } finally {
        this.matkulEditLoading = false;
      }
    },

    mintaKonfirmasiStatusMatkul(m) {
      const st = String(m.status || 'ACTIVE').toUpperCase();
      this.matkulStatusConfirm = {
        id: m.id,
        kode: m.code,
        nama: m.name,
        dari: st,
        ke: st === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE',
        fromModalEdit: false
      };
    },

    batalKonfirmasiStatusMatkul() {
      if (this.matkulStatusLoading) return;
      this.matkulStatusConfirm = null;
    },

    async jalankanUbahStatusMatkul() {
      const c = this.matkulStatusConfirm;
      if (!c) return;
      this.matkulStatusLoading = true;
      try {
        const payload = { status: c.ke };
        if (c.nama) {
          payload.name = c.nama;
        }
        await API.patchMasterCourse(c.id, payload);
        this.matkulStatusConfirm = null;
        if (c.fromModalEdit) {
          this.matkulEdit = null;
        }
        await this.loadMatkul();
        this.showToast(c.ke === 'ACTIVE'
          ? `Mata kuliah ${c.kode} berhasil diaktifkan.`
          : `Mata kuliah ${c.kode} dinonaktifkan. Nonaktif tidak dipakai untuk penawaran jadwal baru.`);
      } catch (err) {
        this.showToast(err.message || 'Gagal mengubah status mata kuliah.');
      } finally {
        this.matkulStatusLoading = false;
      }
    },

    salinKodeMatkul(code) {
      if (!code) return;
      this.copyText(code, `Kode mata kuliah ${code} disalin.`);
    },

    namaUsulanMatkul(u) {
      if (!u) return 'Mata Kuliah Baru';
      if (u.payload_json) {
        try {
          const p = typeof u.payload_json === 'string' ? JSON.parse(u.payload_json) : u.payload_json;
          if (p && p.name) return p.name;
        } catch (_) {}
      }
      return u.target_code || 'Mata Kuliah Baru';
    },

    formatWaktuRelatif(dateStr) {
      if (!dateStr) return '';
      const d = new Date(dateStr);
      if (isNaN(d.getTime())) return dateStr;
      const now = new Date();
      const diffSec = Math.floor((now - d) / 1000);
      if (diffSec < 60) return 'baru saja';
      const diffMin = Math.floor(diffSec / 60);
      if (diffMin < 60) return `${diffMin} menit lalu`;
      const diffHours = Math.floor(diffMin / 60);
      if (diffHours < 24) return `${diffHours} jam lalu`;
      const diffDays = Math.floor(diffHours / 24);
      if (diffDays < 30) return `${diffDays} hari lalu`;
      return d.toLocaleDateString('id-ID', { day: 'numeric', month: 'short' });
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
      this.importError = '';
      this.importSuccess = '';
      try {
        const res = await API.syncMasterAllJadwal();
        await Promise.all([this.loadMatkul(), this.loadRuang(), this.loadDosen()]);
        const msg = res.message || `Berhasil menyinkronkan ${res.total_synced || 0} master data kampus.`;
        this.importSuccess = msg;
        this.showToast(msg);
      } catch (err) {
        this.importError = err.message || 'Gagal menyinkronkan master data kampus.';
        this.showToast(this.importError);
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
      this.penugasanStatus = String(status || '');
      this.undanganStatus = '';
      this.go('pengguna');
    },

    async go(v) {
      this.pageState = null;
      if (v === 'pembaruan') v = 'audit';
      if (!this.knownViews.includes(v)) { this.showPageError('404'); return; }
      this.view = v;
      this.drawer = false;
      if (v === 'buat') {
        this.view = 'kelas';
        this.bukaModalBuatKelas();
        return;
      }
      if (v === 'antrean') this.loadAntrean();
      if (v === 'backup') { this.backupHasil = null; this.restoreHasil = null; this.loadBackupList(); this.loadBackupRequests(); }
      if (v === 'audit') this.loadAudit();
      if (v === 'pengguna') {
        this.penggunaFilter = '';
        await this.muatPenggunaSemua();
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
      this.detailMenu = null;
      this.dukunganFormTerbuka = false;
      this.portalCodeReveal = '';
      this.kmPenugasan = null;
      this.smPreviewMap = {};
      this.smPratinjau = null;
      this.smAktifkan = null;
      this.smMenu = null;
      this.smArsipTerbuka = null;
      this.smBuatTerbuka = false;
      this.smImporTerbuka = false;
      this.view = 'detail';
      window.scrollTo({ top: 0 });
      this.loadDetailSemester(obj.slug);
      this.loadDetailSettings(obj.slug);
      this.muatKMPenugasan(obj.slug);
    },

    kembaliKeDetail() {
      const slug = this.jadwalKelasSlug || (this.kelasAktifObj && this.kelasAktifObj.slug) || '';
      const target = (this.kelasList || []).find(x => x.slug === slug);
      if (target) this.bukaDetail(target);
      else this.go('kelas');
    },

    jadwalHariNama(n) {
      const m = { 1: 'Senin', 2: 'Selasa', 3: 'Rabu', 4: 'Kamis', 5: 'Jumat', 6: 'Sabtu', 7: 'Minggu' };
      return m[Number(n)] || '-';
    },

    isJadwalDraft() {
      return String((this.jadwalSemester && this.jadwalSemester.status) || '').toUpperCase() === 'DRAFT';
    },

    jadwalOfferingMap() {
      const m = {};
      (this.jadwalOfferings || []).forEach(o => { m[String(o.id)] = o; });
      return m;
    },

    jadwalRow(o) {
      const map = this.jadwalOfferingMap();
      const off = map[String(o.course_offering_id || o.offering_id || '')] || {};
      const lect = Array.isArray(off.lecturers) ? off.lecturers.filter(Boolean).join(', ') : '';
      return {
        id: o.id,
        version: o.version,
        day: o.day_of_week,
        start: o.start_time,
        end: o.end_time,
        room: o.room || '',
        room_id: o.room_id,
        offering: o.offering || o.display_name || off.display_name || '—',
        activity: String(off.activity_type || '').toUpperCase(),
        dosen: lect,
        offering_id: o.course_offering_id || o.offering_id,
        meeting_link: o.meeting_link || '',
      };
    },

    jadwalCountHari(n) {
      return (this.jadwalPatterns || []).filter(p => Number(p.day_of_week) === Number(n)).length;
    },

    jadwalRuanganUnik() {
      const s = new Map();
      (this.jadwalPatterns || []).forEach(p => {
        const code = String(p.room || '').trim();
        if (code && !s.has(code)) s.set(code, code);
      });
      return Array.from(s.values());
    },

    jadwalKesiapan() {
      const p = this.jadwalPreviewSem;
      const conflicts = (p && (p.conflicts || p.Conflicts)) || [];
      const blockers = (p && (p.blockers || p.Blockers)) || [];
      const allIssues = [...blockers, ...conflicts];
      const n = allIssues.length;
      if (n === 0) return { text: 'Kesiapan: 0 Bentrok (Siap Terbit)', tone: 'ok', detail: '' };
      return { text: `Kesiapan: ${n} masalah (Perlu Tinjau)`, tone: 'warn', detail: allIssues.join(' • ') };
    },

    jadwalFiltered() {
      const q = (this.jadwalQ || '').trim().toLowerCase();
      const hari = Number(this.jadwalHari || 0);
      const map = this.jadwalOfferingMap();
      return (this.jadwalPatterns || [])
        .filter(p => !hari || Number(p.day_of_week) === hari)
        .filter(p => {
          if (!q) return true;
          const off = map[String(p.course_offering_id || '')] || {};
          const hay = [p.offering || off.display_name, (off.lecturers || []).join(' '), p.room].filter(Boolean).join(' ').toLowerCase();
          return hay.includes(q);
        })
        .map(p => this.jadwalRow(p))
        .sort((a, b) => (a.day - b.day) || String(a.start).localeCompare(String(b.start)));
    },

    jadwalHariList() {
      const groups = {};
      this.jadwalFiltered().forEach(r => {
        (groups[r.day] = groups[r.day] || []).push(r);
      });
      return [1, 2, 3, 4, 5, 6, 7]
        .filter(d => groups[d] && groups[d].length)
        .map(d => ({ day: d, nama: this.jadwalHariNama(d), rows: groups[d] }));
    },

    async bukaJadwal(kelasSlug, semesterId) {
      const slug = kelasSlug || (this.kelasAktifObj && this.kelasAktifObj.slug) || this.kelasAktif;
      if (!slug) { this.showToast('Pilih kelas dulu.'); return; }
      this.jadwalKelasSlug = slug;
      if (!this.kelasAktifObj || this.kelasAktifObj.slug !== slug) {
        this.kelasAktifObj = (this.kelasList || []).find(x => x.slug === slug) || { slug, nama: slug };
        this.kelasAktif = slug;
      }
      this.view = 'jadwal';
      this.jadwalHari = 0;
      this.jadwalQ = '';
      this.jadwalModal = false;
      this.jadwalHapus = null;
      window.scrollTo({ top: 0 });
      await this.muatJadwal(semesterId);
    },

    async pilihJadwalSemester(id) {
      if (!id || String(id) === String(this.jadwalSemesterId)) return;
      await this.muatJadwal(id);
    },

    async muatJadwal(semesterId) {
      const slug = this.jadwalKelasSlug;
      if (!slug) return;
      this.jadwalLoading = true;
      this.jadwalError = '';
      try {
        const res = await API.getSemestersResult(slug);
        const list = (res && res.ok && Array.isArray(res.data)) ? res.data : [];
        if (!list.length) throw new Error('Belum ada semester untuk kelas ini.');
        this.jadwalSemesterOptions = list;
        const picked = list.find(s => String(s.id) === String(semesterId))
          || list.find(s => String(s.status || '').toUpperCase() === 'ACTIVE')
          || list.find(s => String(s.status || '').toUpperCase() === 'DRAFT')
          || list[0];
        this.jadwalSemesterId = String(picked.id);
        this.jadwalSemester = picked;
        const [offerings, patterns, preview] = await Promise.all([
          API.getSemesterOfferings(picked.id).catch(() => null),
          API.getPatterns().catch(() => []),
          API.previewSemester(slug, picked.id).catch(() => null),
        ]);
        if (offerings === null) throw new Error('Daftar mata kuliah gagal dimuat.');
        this.jadwalOfferings = Array.isArray(offerings) ? offerings : [];
        const ids = new Set(this.jadwalOfferings.map(o => String(o.id)));
        this.jadwalPatterns = (Array.isArray(patterns) ? patterns : []).filter(p => ids.has(String(p.course_offering_id || '')));
        this.jadwalPreviewSem = preview;
      } catch (e) {
        this.jadwalOfferings = [];
        this.jadwalPatterns = [];
        this.jadwalPreviewSem = null;
        this.jadwalError = e.message || 'Jadwal gagal dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.jadwalLoading = false;
      }
    },

    jadwalDurasi() {
      const f = this.jadwalForm;
      const m = String(f.start || '').match(/^(\d{2}):(\d{2})$/);
      if (!m) return 0;
      if (f.end) {
        const n = String(f.end).match(/^(\d{2}):(\d{2})$/);
        if (!n) return 0;
        return (Number(n[1]) * 60 + Number(n[2])) - (Number(m[1]) * 60 + Number(m[2]));
      }
      return Number(f.duration || 0);
    },

    bukaTambahSesi() {
      if ((this.jadwalOfferings || []).length === 0) {
        this.showToast('Semester ini belum punya mata kuliah. Impor kurikulum dulu.');
        return;
      }
      this.jadwalForm = { id: '', version: 0, offeringId: '', day: '1', start: '', duration: 100, end: '', roomId: '', link: '', reason: '', effectiveDate: '' };
      this.jadwalFormError = '';
      this.jadwalPreview = null;
      this.jadwalModal = true;
    },

    bukaUbahSesi(r) {
      this.jadwalForm = { id: String(r.id), version: r.version || 0, offeringId: String(r.offering_id || ''), day: String(r.day || '1'), start: String(r.start || '').slice(0, 5), duration: 0, end: String(r.end || '').slice(0, 5), roomId: r.room_id ? String(r.room_id) : '', link: r.meeting_link || '', reason: '', effectiveDate: '' };
      this.jadwalFormError = '';
      this.jadwalPreview = null;
      this.jadwalModal = true;
    },

    hitungJadwalAkhir() {
      const f = this.jadwalForm;
      if (f.id) return;
      const m = String(f.start || '').match(/^(\d{2}):(\d{2})$/);
      const total = m ? Number(m[1]) * 60 + Number(m[2]) + Number(f.duration || 0) : 0;
      f.end = total > 0 && total < 1440 ? String(Math.floor(total / 60)).padStart(2, '0') + ':' + String(total % 60).padStart(2, '0') : '';
      this.jadwalPreview = null;
    },

    rakitJadwalPayload() {
      const f = this.jadwalForm;
      if (f.id) {
        const p = (this.jadwalPatterns || []).find(x => String(x.id) === String(f.id));
        if (!p) throw new Error('Pola asal belum termuat. Muat ulang daftar.');
        const payload = { version: Number(p.version), day_of_week: Number(f.day), start_time: f.start, duration_min: (() => {
          const a = String(f.start).match(/^(\d{2}):(\d{2})$/);
          const b = String(f.end).match(/^(\d{2}):(\d{2})$/);
          if (!a || !b) return 0;
          return (Number(b[1]) * 60 + Number(b[2])) - (Number(a[1]) * 60 + Number(a[2]));
        })() };
        if (f.roomId) payload.room_id = Number(f.roomId);
        if ((f.link || '').trim()) payload.meeting_link = f.link.trim();
        if (!this.isJadwalDraft()) {
          payload.effective_from = f.effectiveDate;
          payload.reason = (f.reason || '').trim();
        } else if ((f.reason || '').trim()) {
          payload.reason = f.reason.trim();
        }
        return payload;
      }
      const payload = { offering_id: Number(f.offeringId), day_of_week: Number(f.day), start_time: f.start, duration_min: Number(f.duration) };
      if (f.roomId) payload.room_id = Number(f.roomId);
      if ((f.link || '').trim()) payload.meeting_link = f.link.trim();
      return payload;
    },

    async pratinjauJadwal() {
      const f = this.jadwalForm;
      if (!f.id && !f.offeringId) { this.jadwalFormError = 'Pilih mata kuliah dulu.'; return; }
      if (!f.start) { this.jadwalFormError = 'Isi jam mulai yang valid.'; return; }
      if (f.id) {
        if (this.jadwalDurasi() <= 0) { this.jadwalFormError = 'Jam selesai harus setelah jam mulai.'; return; }
        if (!this.isJadwalDraft()) {
          if (!f.effectiveDate) { this.jadwalFormError = 'Isi tanggal mulai berlaku.'; return; }
          if ((f.reason || '').trim().length < 5) { this.jadwalFormError = 'Keterangan minimal 5 karakter.'; return; }
        }
      } else if (!(Number(f.duration) > 0)) { this.jadwalFormError = 'Durasi harus lebih dari 0 menit.'; return; }
      this.jadwalFormError = '';
      this.jadwalPreviewLoading = true;
      try {
        this.jadwalPreview = f.id
          ? await API.previewPattern(f.id, this.rakitJadwalPayload())
          : await API.previewCreatePattern(this.rakitJadwalPayload());
        if (this.jadwalPreview && this.jadwalPreview.can_publish === false) {
          this.jadwalFormError = 'Ada konflik yang menghalangi penyimpanan.';
        }
      } catch (e) {
        this.jadwalFormError = e.message || 'Gagal meninjau jadwal.';
      } finally {
        this.jadwalPreviewLoading = false;
      }
    },

    async simpanJadwal() {
      if (!this.jadwalPreview) { await this.pratinjauJadwal(); if (!this.jadwalPreview) return; }
      if (this.jadwalPreview.can_publish === false) return;
      const f = this.jadwalForm;
      this.jadwalFormSaving = true;
      this.jadwalFormError = '';
      try {
        if (f.id) await API.patchPattern(f.id, this.rakitJadwalPayload());
        else await API.createPattern(this.rakitJadwalPayload());
        this.showToast(f.id ? (this.isJadwalDraft() ? 'Jadwal draf diperbarui.' : 'Jadwal diperbarui.') : 'Sesi perkuliahan ditambahkan.');
        this.jadwalModal = false;
        this.jadwalPreview = null;
        await this.muatJadwal(this.jadwalSemesterId);
      } catch (e) {
        this.jadwalFormError = e.message || 'Gagal menyimpan jadwal.';
      } finally {
        this.jadwalFormSaving = false;
      }
    },

    mintaHapusSesi(r) {
      this.jadwalHapus = { id: r.id, version: r.version, nama: r.offering };
      this.jadwalHapusError = '';
    },

    async jalankanHapusSesi() {
      const h = this.jadwalHapus;
      if (!h) return;
      try {
        await API.deletePattern(h.id, h.version);
        this.showToast(this.isJadwalDraft() ? 'Jadwal draf dihapus.' : 'Jadwal dihapus (tidak aktif mulai hari ini).');
        this.jadwalHapus = null;
        await this.muatJadwal(this.jadwalSemesterId);
      } catch (e) {
        this.jadwalHapusError = e.message || 'Gagal menghapus sesi.';
      }
    },

    async muatKMPenugasan(slug) {
      if (!slug) return;
      this.kmPenugasanLoading = true;
      this.kmPenugasan = null;
      try {
        const list = await API.getAdminAssignments('ACTIVE', 'KM', slug).catch(() => []);
        this.kmPenugasan = (list || [])[0] || null;
      } catch (e) {
        this.kmPenugasan = null;
      } finally {
        this.kmPenugasanLoading = false;
      }
    },

    fmtTanggalID(iso) {
      try {
        const d = new Date(iso);
        if (isNaN(d)) return '-';
        return new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'long', year: 'numeric' }).format(d);
      } catch (e) { return '-'; }
    },

    fmtRentangID(mulai, selesai) {
      try {
        const f = (v) => {
          const d = new Date(v.length === 10 ? v + 'T00:00:00' : v);
          if (isNaN(d)) return '';
          const p = (n) => String(n).padStart(2, '0');
          return p(d.getDate()) + ' ' + ['Jan','Feb','Mar','Apr','Mei','Jun','Jul','Agu','Sep','Okt','Nov','Des'][d.getMonth()] + ' ' + d.getFullYear();
        };
        const a = mulai ? f(String(mulai)) : '';
        const b = selesai ? f(String(selesai)) : '';
        return [a, b].filter(Boolean).join(' – ') || '-';
      } catch (e) { return '-'; }
    },

    portalURL() {
      const slug = this.kelasAktifObj && this.kelasAktifObj.slug ? this.kelasAktifObj.slug : '';
      return slug ? (window.location.origin + '/c/' + encodeURIComponent(slug)) : '';
    },

    async muatPreviewSemesterSA(semId) {
      const slug = this.kelasAktifObj && this.kelasAktifObj.slug ? this.kelasAktifObj.slug : '';
      if (!slug || !semId || this.smPreviewMap[semId]) return;
      this.smPreviewMap[semId] = { loading: true, data: null };
      try {
        const data = await API.previewSemester(slug, semId);
        this.smPreviewMap[semId] = { loading: false, data: data || null };
      } catch (e) {
        this.smPreviewMap[semId] = { loading: false, data: null };
      }
    },

    smJumlah(semId, kunci) {
      const entry = this.smPreviewMap[semId];
      if (!entry || entry.loading || !entry.data) return null;
      const v = entry.data[kunci];
      return typeof v === 'number' ? v : null;
    },

    bukaTambahSesiSM(s) {
      this.smMenu = null;
      const slug = this.kelasAktifObj && this.kelasAktifObj.slug ? this.kelasAktifObj.slug : '';
      if (slug && s && s.id) {
        this.bukaJadwal(slug, s.id);
        return;
      }
      this.bukaPratinjauSM(s);
      this.showToast('Kelola rincian sesi jadwal melalui pratinjau kesiapan semester.');
    },

    bukaPratinjauSM(s) {
      this.smMenu = null;
      const entry = this.smPreviewMap[s.id] || {};
      this.smPratinjau = { sem: s, data: entry.data || null, loading: !!entry.loading };
      if (!entry.data && !entry.loading) {
        this.muatPreviewSemesterSA(s.id).then(() => {
          if (this.smPratinjau && this.smPratinjau.sem && String(this.smPratinjau.sem.id) === String(s.id)) {
            this.smPratinjau = { sem: s, data: (this.smPreviewMap[s.id] || {}).data || null, loading: false };
          }
        });
      }
    },

    mulaiAktifkanSM(s) {
      this.smMenu = null;
      this.smAktifkan = { id: s.id, nama: this.labelSemester(s) };
    },

    async jalankanAktifkanSM() {
      const s = this.smAktifkan;
      const slug = this.kelasAktifObj && this.kelasAktifObj.slug ? this.kelasAktifObj.slug : '';
      if (!s || !slug) return;
      try {
        await API.activateSemester(slug, s.id);
        this.showToast('Semester diaktifkan. Semester aktif lama menjadi arsip.');
        this.smAktifkan = null;
        this.smPreviewMap = {};
        await this.loadDetailSemester(slug);
        await this.muatSemesterSemua();
      } catch (err) {
        this.showToast(err.message || 'Gagal mengaktifkan semester.');
      }
    },

    konfirmasiHapusDraf(s) {
      this.smMenu = null;
      this.smHapusKonfirmasi = { id: s.id, nama: this.labelSemester(s) };
      this.smHapusSaving = false;
      this.smHapusError = '';
    },

    async eksekusiHapusDraf() {
      const h = this.smHapusKonfirmasi;
      const slug = this.kelasAktifObj && this.kelasAktifObj.slug ? this.kelasAktifObj.slug : '';
      if (!h || !slug) return;
      this.smHapusSaving = true;
      this.smHapusError = '';
      try {
        await API.deleteSemesterDraft(slug, h.id);
        this.showToast('Semester draf berhasil dihapus.');
        this.smHapusKonfirmasi = null;
        this.smPreviewMap = {};
        await this.loadDetailSemester(slug);
        await this.muatSemesterSemua();
      } catch (err) {
        this.smHapusError = err.message || 'Gagal menghapus semester draf.';
      } finally {
        this.smHapusSaving = false;
      }
    },

    bukaBuatSemester() {
      const slug = this.kelasAktifObj && this.kelasAktifObj.slug ? this.kelasAktifObj.slug : '';
      if (!slug) { this.showToast('Pilih kelas dulu.'); return; }
      this.smBuatTerbuka = true;
      this.smBuatError = '';
      if (!this.smBuatForm.academic_year) {
        const th = new Date().getFullYear();
        this.smBuatForm.academic_year = th + '/' + (th + 1);
      }
    },

    async simpanSemesterBaru() {
      const slug = this.kelasAktifObj && this.kelasAktifObj.slug ? this.kelasAktifObj.slug : '';
      const f = this.smBuatForm;
      if (!slug) return;
      if (!((f.academic_year || '').trim())) { this.smBuatError = 'Tahun akademik wajib diisi (contoh: 2025/2026).'; return; }
      if (!((f.term || '').trim())) { this.smBuatError = 'Semester (Ganjil/Genap) wajib diisi.'; return; }
      if (!f.starts_on || !f.ends_on) { this.smBuatError = 'Tanggal mulai dan selesai wajib diisi.'; return; }
      if (!(f.ends_on > f.starts_on)) { this.smBuatError = 'Tanggal selesai harus setelah tanggal mulai.'; return; }
      this.smBuatError = '';
      this.smBuatSaving = true;
      try {
        await API.createSemesterDraft(slug, {
          academic_year: f.academic_year.trim(),
          term: f.term.trim().toUpperCase(),
          starts_on: f.starts_on,
          ends_on: f.ends_on
        });
        this.showToast('Semester draf dibuat.');
        this.smBuatTerbuka = false;
        this.smBuatForm = { academic_year: '', term: 'Ganjil', starts_on: '', ends_on: '' };
        await this.loadDetailSemester(slug);
        await this.muatSemesterSemua();
      } catch (err) {
        this.smBuatError = err.message || 'Gagal membuat semester draf.';
      } finally {
        this.smBuatSaving = false;
      }
    },

    bukaTambahOffering(sem) {
      if (!sem || !sem.id) {
        sem = (this.detailSemesterList || []).find(s => String(s.status || '').toUpperCase() === 'DRAFT') || this.jadwalSemester;
      }
      if (!sem || !sem.id) {
        this.showToast('Pilih semester draf terlebih dahulu.');
        return;
      }
      this.offeringSemesterTarget = sem;
      this.offeringForm = {
        course_code: '',
        activity_type: 'TEORI',
        display_name: '',
        lecturer_codes: []
      };
      this.offeringFormError = '';
      this.offeringFormSaving = false;
      this.offeringQuickMatkulOpen = false;
      this.offeringQuickMatkul = { kode: '', nama: '' };
      this.offeringQuickMatkulSaving = false;
      this.offeringQuickMatkulError = '';
      this.offeringQuickDosenOpen = false;
      this.offeringQuickDosen = { kode: '', nama: '' };
      this.offeringQuickDosenSaving = false;
      this.offeringQuickDosenError = '';
      if (!this.matkulList || !this.matkulList.length) this.loadMatkul();
      if (!this.dosenList || !this.dosenList.length) this.loadDosen();
      this.modalTambahOffering = true;
    },

    async simpanQuickMatkul() {
      const q = this.offeringQuickMatkul;
      const kode = (q.kode || '').trim().toUpperCase();
      const nama = (q.nama || '').trim();
      if (!kode || !nama) {
        this.offeringQuickMatkulError = 'Kode dan nama mata kuliah wajib diisi.';
        return;
      }
      this.offeringQuickMatkulSaving = true;
      this.offeringQuickMatkulError = '';
      try {
        await API.createMasterCourse({ code: kode, name: nama });
        await this.loadMatkul();
        this.offeringForm.course_code = kode;
        this.onOfferingCourseSelect();
        this.offeringQuickMatkul = { kode: '', nama: '' };
        this.offeringQuickMatkulOpen = false;
        this.showToast(`Mata kuliah ${kode} berhasil didaftarkan ke Master Data.`);
      } catch (err) {
        this.offeringQuickMatkulError = err.message || 'Gagal mendaftarkan mata kuliah baru ke Master Data.';
      } finally {
        this.offeringQuickMatkulSaving = false;
      }
    },

    async simpanQuickDosen() {
      const q = this.offeringQuickDosen;
      const kode = (q.kode || '').trim().toUpperCase();
      const nama = (q.nama || '').trim();
      if (!kode || !nama) {
        this.offeringQuickDosenError = 'Kode/inisial dan nama dosen wajib diisi.';
        return;
      }
      this.offeringQuickDosenSaving = true;
      this.offeringQuickDosenError = '';
      try {
        await API.createMasterLecturer({ code: kode, full_name: nama });
        await this.loadDosen();
        if (!this.offeringForm.lecturer_codes.includes(kode)) {
          this.offeringForm.lecturer_codes.push(kode);
        }
        this.offeringQuickDosen = { kode: '', nama: '' };
        this.offeringQuickDosenOpen = false;
        this.showToast(`Dosen ${nama} (${kode}) berhasil didaftarkan ke Master Data.`);
      } catch (err) {
        this.offeringQuickDosenError = err.message || 'Gagal mendaftarkan dosen baru ke Master Data.';
      } finally {
        this.offeringQuickDosenSaving = false;
      }
    },

    onOfferingCourseSelect() {
      const code = this.offeringForm.course_code;
      const m = (this.matkulList || []).find(x => x.code === code);
      if (m) {
        const act = this.offeringForm.activity_type === 'PRAKTIKUM' ? ' (Praktikum)' : ' (Teori)';
        this.offeringForm.display_name = (m.name || m.code) + act;
      }
    },

    onOfferingActivityChange() {
      this.onOfferingCourseSelect();
    },

    toggleOfferingLecturer(code) {
      const idx = this.offeringForm.lecturer_codes.indexOf(code);
      if (idx >= 0) this.offeringForm.lecturer_codes.splice(idx, 1);
      else this.offeringForm.lecturer_codes.push(code);
    },

    async simpanOffering() {
      const f = this.offeringForm;
      const sem = this.offeringSemesterTarget;
      if (!sem || !sem.id) {
        this.offeringFormError = 'Semester target tidak ditemukan.';
        return;
      }
      if (!f.course_code) {
        this.offeringFormError = 'Pilih mata kuliah dari Master Data terlebih dahulu.';
        return;
      }
      if (!f.display_name) {
        this.offeringFormError = 'Nama tampilan mata kuliah wajib diisi.';
        return;
      }
      this.offeringFormSaving = true;
      this.offeringFormError = '';
      try {
        await API.createSemesterOffering(sem.id, {
          course_code: f.course_code,
          activity_type: f.activity_type,
          display_name: f.display_name,
          lecturer_codes: f.lecturer_codes
        });
        this.showToast('Mata kuliah berhasil ditambahkan ke semester.');
        this.modalTambahOffering = false;
        if (this.view === 'detail' && this.kelasAktifObj && this.kelasAktifObj.slug) {
          await this.muatDetailKelas(this.kelasAktifObj.slug);
        } else if (this.view === 'jadwal' && this.jadwalSemesterId) {
          await this.muatJadwal(this.jadwalSemesterId);
        }
      } catch (e) {
        this.offeringFormError = (e && e.message) || 'Gagal menambahkan mata kuliah.';
      } finally {
        this.offeringFormSaving = false;
      }
    },

    bukaDaftarOffering() {
      this.modalDaftarOffering = true;
    },

    offeringPatternsCount(offeringId) {
      if (!offeringId) return 0;
      return (this.jadwalPatterns || []).filter(p => String(p.course_offering_id || '') === String(offeringId)).length;
    },

    async hapusOffering(off) {
      if (!off || !off.id) return;
      const count = this.offeringPatternsCount(off.id);
      if (count > 0) {
        this.showToast('Mata kuliah ini masih memiliki ' + count + ' sesi jadwal. Hapus sesinya terlebih dahulu.');
        return;
      }
      if (!confirm('Hapus mata kuliah "' + (off.display_name || off.course_code) + '" dari semester ini?')) {
        return;
      }
      const semId = (this.jadwalSemester && this.jadwalSemester.id) || (this.offeringSemesterTarget && this.offeringSemesterTarget.id);
      if (!semId) {
        this.showToast('ID semester tidak ditemukan.');
        return;
      }
      try {
        await API.deleteSemesterOffering(semId, off.id);
        this.showToast('Mata kuliah dihapus dari semester.');
        if (this.view === 'jadwal' && this.jadwalSemesterId) {
          await this.muatJadwal(this.jadwalSemesterId);
        } else if (this.view === 'detail' && this.kelasAktifObj && this.kelasAktifObj.slug) {
          await this.muatDetailKelas(this.kelasAktifObj.slug);
        }
      } catch (e) {
        this.showToast((e && e.message) || 'Gagal menghapus mata kuliah.');
      }
    },

    bukaImporSemester(sem) {
      if (!this.kelasAktifObj || !this.kelasAktifObj.slug) { this.showToast('Pilih kelas dulu.'); return; }
      const target = sem || (this.detailSemesterList || []).find(s => String(s.status || '').toUpperCase() === 'DRAFT');
      if (!target) { this.showToast('Buat semester draf dulu sebelum mengimpor kurikulum.'); return; }
      this.smImporSemester = target;
      this.smImporTerbuka = true;
      this.smImporFile = null;
      this.smImporHasil = null;
      this.smImporError = '';
      this.smImporDragOver = false;
    },

    fmtFileSize(bytes) {
      if (!bytes || bytes === 0) return '0 B';
      const k = 1024;
      const sizes = ['B', 'KB', 'MB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    },

    pilihFileImporSM(ev) {
      const files = ev && ev.target && ev.target.files;
      this.smImporFile = files && files[0] ? files[0] : null;
      this.smImporHasil = null;
      this.smImporError = '';
      this.smImporDragOver = false;
    },

    dropFileImporSM(ev) {
      this.smImporDragOver = false;
      const files = ev && ev.dataTransfer && ev.dataTransfer.files;
      if (files && files[0]) {
        const file = files[0];
        if (file.name.endsWith('.json') || file.type === 'application/json') {
          this.smImporFile = file;
          this.smImporHasil = null;
          this.smImporError = '';
        } else {
          this.smImporError = 'Hanya berkas format .json yang didukung.';
        }
      }
    },

    resetImporSM() {
      this.smImporFile = null;
      this.smImporHasil = null;
      this.smImporError = '';
      this.smImporDragOver = false;
      if (this.$refs.smImporInput) this.$refs.smImporInput.value = '';
    },

    convertPolbanToCurriculum(doc) {
      if (!doc || typeof doc !== 'object') return null;
      if (Array.isArray(doc.courses) || Array.isArray(doc.offerings) || Array.isArray(doc.schedule_patterns)) {
        return doc;
      }
      if (doc.mata_kuliah && typeof doc.mata_kuliah === 'object') {
        const courses = [];
        for (const [code, name] of Object.entries(doc.mata_kuliah)) {
          courses.push({ code: String(code).trim(), name: String(name).trim() });
        }
        const lecturers = [];
        if (doc.dosen && typeof doc.dosen === 'object') {
          for (const [code, fullName] of Object.entries(doc.dosen)) {
            lecturers.push({ code: String(code).trim(), full_name: String(fullName).trim() });
          }
        }
        const offeringsMap = {};
        const patterns = [];
        const validDays = { senin: 1, selasa: 2, rabu: 3, kamis: 4, jumat: 5, sabtu: 6, minggu: 7 };

        if (Array.isArray(doc.jadwal)) {
          doc.jadwal.forEach(j => {
            const cCode = String(j.kode_matkul || '').trim();
            if (!cCode) return;
            const actType = (j.nama_matkul && /praktikum|lab/i.test(j.nama_matkul)) ? 'PRAKTIKUM' : 'TEORI';
            const key = cCode + '|' + actType;
            if (!offeringsMap[key]) {
              offeringsMap[key] = {
                course_code: cCode,
                activity_type: actType,
                display_name: String(j.nama_matkul || doc.mata_kuliah[cCode] || cCode).trim(),
                lecturer_codes: []
              };
            }
            const lCode = String(j.inisial_dosen || '').trim();
            if (lCode && !offeringsMap[key].lecturer_codes.includes(lCode)) {
              offeringsMap[key].lecturer_codes.push(lCode);
            }

            const dayStr = String(j.hari || '').toLowerCase().trim();
            const dow = validDays[dayStr] || 1;
            const jamParts = String(j.jam || '').split('-');
            if (jamParts.length === 2) {
              const start = jamParts[0].trim();
              const end = jamParts[1].trim();
              const [sh, sm] = start.split(':').map(Number);
              const [eh, em] = end.split(':').map(Number);
              const startMin = (sh || 0) * 60 + (sm || 0);
              const endMin = (eh || 0) * 60 + (em || 0);
              const dur = Math.max(endMin - startMin, 50);

              patterns.push({
                course_code: cCode,
                activity_type: actType,
                day_of_week: dow,
                start_time: start,
                duration_min: dur,
                room_code: j.ruang ? String(j.ruang).trim() : null
              });
            }
          });
        }

        return {
          source_type: 'JSON',
          courses: courses,
          lecturers: lecturers,
          offerings: Object.values(offeringsMap),
          schedule_patterns: patterns
        };
      }
      return doc;
    },

    unduhTemplateKurikulum() {
      const template = {
        source_type: "JSON",
        courses: [
          { code: "25TI1101", name: "Dasar-Dasar Pemrograman" },
          { code: "25TI1102", name: "Komputasi Kognitif" }
        ],
        lecturers: [
          { code: "AB", full_name: "Akhmad Bakhrun, S.Kom, M.T." },
          { code: "AD", full_name: "Dr. Ade Chandra Nugraha, S.Si., M.T." }
        ],
        offerings: [
          {
            course_code: "25TI1101",
            activity_type: "TEORI",
            display_name: "Dasar-Dasar Pemrograman (Teori)",
            lecturer_codes: ["AB"]
          },
          {
            course_code: "25TI1102",
            activity_type: "TEORI",
            display_name: "Komputasi Kognitif (Teori)",
            lecturer_codes: ["AD"]
          }
        ],
        schedule_patterns: [
          {
            course_code: "25TI1101",
            activity_type: "TEORI",
            day_of_week: 1,
            start_time: "07:00",
            duration_min: 100,
            room_code: "D101"
          },
          {
            course_code: "25TI1102",
            activity_type: "TEORI",
            day_of_week: 2,
            start_time: "08:40",
            duration_min: 100,
            room_code: "D102"
          }
        ]
      };
      const blob = new Blob([JSON.stringify(template, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'template_kurikulum.json';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    },

    async validasiImporSM() {
      const target = this.smImporSemester || (this.detailSemesterList || []).find(s => String(s.status || '').toUpperCase() === 'DRAFT');
      if (!target) { this.smImporError = 'Tidak ada semester draf tujuan pada kelas ini.'; return; }
      if (!this.smImporFile) { this.smImporError = 'Pilih berkas JSON kurikulum terlebih dahulu.'; return; }
      let rawObj = null;
      try {
        rawObj = JSON.parse(await this.smImporFile.text());
      } catch (e) {
        this.smImporError = 'Format berkas tidak valid atau bukan JSON standar.';
        return;
      }
      const payload = this.convertPolbanToCurriculum(rawObj) || rawObj;
      this.smImporError = '';
      this.smImporSaving = true;
      try {
        this.smImporHasil = await API.importSemester(target.id, payload);
        if (this.smImporHasil && this.smImporHasil.has_fatal_errors) {
          this.smImporError = 'Validasi ditolak karena ditemukan galat.';
        } else {
          this.showToast('Validasi berkas berhasil. Siap diterapkan ke draf.');
        }
      } catch (err) {
        if (err.errors && Array.isArray(err.errors)) {
          this.smImporHasil = {
            has_fatal_errors: true,
            status: 'INVALID',
            errors: err.errors
          };
          this.smImporError = err.message || 'Validasi ditolak karena ditemukan galat.';
        } else {
          this.smImporHasil = null;
          this.smImporError = err.message || 'Gagal memvalidasi berkas impor.';
        }
      } finally {
        this.smImporSaving = false;
      }
    },

    async terapkanImporSM() {
      const hasil = this.smImporHasil;
      const target = this.smImporSemester || (this.detailSemesterList || []).find(s => String(s.status || '').toUpperCase() === 'DRAFT');
      if (!hasil || !hasil.batch_id) { this.smImporError = 'Validasi berkas terlebih dahulu sebelum menerapkan.'; return; }
      if (hasil.has_fatal_errors) { this.smImporError = 'Masih ada galat fatal. Perbaiki berkas terlebih dahulu.'; return; }
      if (!target) { this.smImporError = 'Tidak ada semester draf tujuan.'; return; }
      this.smImporSaving = true;
      try {
        await API.applySemesterImport(target.id, hasil.batch_id);
        this.showToast('Kurikulum berhasil diterapkan ke semester draf.');
        this.smImporTerbuka = false;
        this.smImporHasil = null;
        this.smImporFile = null;
        this.smPreviewMap = {};
        await this.loadDetailSemester(this.kelasAktifObj.slug);
      } catch (err) {
        this.smImporError = err.message || 'Gagal menerapkan impor ke semester draf.';
      } finally {
        this.smImporSaving = false;
      }
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
          (this.detailSemesterList || []).forEach(s => this.muatPreviewSemesterSA(s.id));
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
        'd4 teknik informatika': 'D4-TI',
        'd3 teknik informatika': 'D3-TI',
        'd3 manajemen informatika': 'D3-MI',
        'd4 akuntansi': 'D4-AK',
        'd3 akuntansi': 'D3-AK',
        'd4 keuangan syariah': 'D4-KS',
        'd3 keuangan dan perbankan': 'D3-KP',
        'd4 administrasi bisnis': 'D4-AB',
        'd3 administrasi bisnis': 'D3-AB',
        'd4 manajemen pemasaran': 'D4-MP',
        'd3 manajemen pemasaran': 'D3-MP',
        'd4 manajemen aset': 'D4-MA',
        'd4 destinasi pariwisata': 'D4-DP',
        'd3 usaha perjalanan wisata': 'D3-UPW',
        'd4 bahasa inggris untuk komunikasi bisnis dan profesional': 'D4-BI',
        'd3 bahasa inggris': 'D3-BI',
        'd4 teknik elektronika': 'D4-TE',
        'd3 teknik elektronika': 'D3-TE',
        'd4 teknik otomasi industri': 'D4-TOI',
        'd4 teknik telekomunikasi': 'D4-TT',
        'd3 teknik telekomunikasi': 'D3-TT',
        'd3 teknik listrik': 'D3-TL',
        'd4 teknologi pembangkit tenaga listrik': 'D4-TPTL',
        'd4 teknik konservasi energi': 'D4-TKE',
        'd3 teknik konversi energi': 'D3-TKE',
        'd4 teknik pendingin dan tata udara': 'D4-TPTU',
        'd3 teknik pendingin dan tata udara': 'D3-TPTU',
        'd4 teknik perancangan jalan dan jembatan': 'D4-TPJJ',
        'd4 teknik perawatan dan perbaikan gedung': 'D4-TPPG',
        'd3 teknik konstruksi sipil': 'D3-TKS',
        'd3 teknik konstruksi gedung': 'D3-TKG',
        'd4 teknik perancangan dan konstruksi mesin': 'D4-TPKM',
        'd4 proses manufaktur': 'D4-PM',
        'd3 teknik mesin': 'D3-TM',
        'd3 teknik aeronautika': 'D3-TA',
        'd4 teknik kimia produksi bersih': 'D4-TKPB',
        'd3 teknik kimia': 'D3-TK',
        'd3 analis kimia': 'D3-AKM'
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
          shortProdi = `${jenjang}-${inisial || sisa.slice(0, 3).toUpperCase()}`;
        } else if (/^[a-zA-Z\s]+$/.test(p) && p.split(/\s+/).length > 1) {
          const words = p.split(/\s+/).filter(w => !['dan', 'untuk', 'atau', 'di', 'ke', 'dari'].includes(w.toLowerCase()));
          shortProdi = words.map(w => w[0] ? w[0].toUpperCase() : '').join('');
        }
      }

      let tingkat = '1';
      if (a) {
        let currYear = (this.activeSemester && this.activeSemester.academic_year)
          ? parseInt(this.activeSemester.academic_year.split('/')[0], 10)
          : new Date().getFullYear();
        let cohort = parseInt(a, 10);
        if (cohort && cohort <= currYear) {
          tingkat = String(Math.max(1, Math.min(4, currYear - cohort + 1)));
        } else {
          tingkat = '1';
        }
      }

      const parts = [];
      if (shortProdi) parts.push(shortProdi);
      if (r) {
        parts.push(`${tingkat}${r}`);
      } else {
        parts.push(`${tingkat}A`);
      }
      return parts.join(' ');
    },

    portalURLPreview() {
      const host = window.location.host || 'domain.com';
      const slug = (this.kelasForm && this.kelasForm.slug) ? this.kelasForm.slug : 'd4-ti-1a';
      return host + '/c/' + slug;
    },

    bukaModalBuatKelas() {
      this.resetKelasForm();
      this.kelasError = '';
      this.modalBuatTerbuka = true;
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
        nama: 'D4-TI 1A',
        slug: 'd4-ti-1a',
        portal_access_mode: 'LINK',
        kustom: false
      };
      this.updateAutoKelas();
    },

    async simpanKelas() {
      const f = this.kelasForm;
      if (!f.kustom || !((f.nama || '').trim())) { this.updateAutoKelas(); }
      if (!((f.prodi || '').trim())) { this.kelasError = 'Program studi wajib diisi.'; return; }
      if (!((f.angkatan || '').trim())) { this.kelasError = 'Tahun angkatan wajib diisi.'; return; }
      const rombel = ((f.rombel || '').trim() || 'A').toUpperCase();
      if (!/^[A-Z0-9]{1,4}$/.test(rombel)) { this.kelasError = 'Rombel wajib 1–4 karakter alfanumerik (contoh: A).'; return; }
      if (!((f.nama || '').trim())) { this.kelasError = 'Nama kelas wajib diisi.'; return; }

      this.kelasError = '';
      this.kelasSaving = true;

      try {
        const cohortNum = parseInt(String(f.angkatan).trim(), 10) || new Date().getFullYear();
        const payload = {
          name: f.nama.trim(),
          code: f.nama.trim(),
          study_program: f.prodi.trim(),
          cohort_year: cohortNum,
          group_label: rombel,
          portal_access_mode: f.portal_access_mode || 'LINK'
        };
        const slugManual = ((f.slug || '').trim());
        if (slugManual) payload.slug = this.slugifyKelas(slugManual);

        const res = await API.createClass(payload);

        this.resetKelasForm();
        this.resetKelasFilter();
        this.modalBuatTerbuka = false;
        await Promise.all([this.loadKelas(), this.checkBot()]);
        await this.muatSemesterSemua();

        if (res && res.portal_code) {
          this.showToast('Kelas ' + (res.code || payload.name) + ' dibuat! Kode PIN Portal: ' + res.portal_code, 7000);
        } else {
          this.showToast('Kelas baru berhasil ditambahkan.');
        }
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
      this.portalCodeReveal = '';
      try {
        const res = await API.rotatePortalCode(this.kelasAktifObj.slug);
        if (res && res.portal_code) {
          this.portalCodeReveal = res.portal_code;
          this.showToast('Kode portal dirotasi. Kode hanya tampil sekali, salin sekarang.');
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

    bukaModalTolakUsulan(u) {
      if (!u) return;
      this.usulanDitolakTarget = u;
      this.usulanAlasanTolak = '';
      this.usulanRejectLoading = false;
      this.modalTolakUsulan = true;
    },

    tutupModalTolakUsulan() {
      if (this.usulanRejectLoading) return;
      this.modalTolakUsulan = false;
      this.usulanDitolakTarget = null;
      this.usulanAlasanTolak = '';
    },

    async prosesTolakUsulan() {
      const u = this.usulanDitolakTarget;
      if (!u) return;
      const note = (this.usulanAlasanTolak || '').trim();
      if (!note) {
        this.showToast('Alasan penolakan wajib diisi untuk KM.');
        return;
      }
      this.usulanRejectLoading = true;
      try {
        await API.decideProposal(u.id, 'reject', note);
        const judul = this.namaUsulanMatkul(u) || u.target_code || 'Mata Kuliah';
        this.showToast(`Usulan "${judul}" telah ditolak.`);
        this.tutupModalTolakUsulan();
        await Promise.all([this.loadUsulan(''), this.loadRuang(), this.loadMatkul()]);
      } catch (err) {
        this.showToast(err.message || 'Gagal menolak usulan.');
      } finally {
        this.usulanRejectLoading = false;
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

    bukaDialogDukungan(k) {
      this.dialogDukunganKelas = k;
      this.kelasAktif = k.slug;
      this.dukunganAlasan = '';
    },

    async jalankanMasukDukungan() {
      if (!this.dialogDukunganKelas) return;
      this.kelasAktif = this.dialogDukunganKelas.slug;
      await this.masukDukungan();
      if (this.dukunganAktif) {
        this.dialogDukunganKelas = null;
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
