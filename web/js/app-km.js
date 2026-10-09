/**
 * web/js/app-km.js — Area KM (REVISI Figma, tanpa role switcher).
 * Memakai helper API global dari js/api.js. Backend yang belum ada
 * ditandai jujur di UI (toast + empty state), tidak difake.
 */
function kmApp() {
  return {
    view: 'dashboard',
    drawer: false,
    sidebarCollapsed: false,
    pageState: null,
    dashboardLoading: true,
    q: '',
    unreadCount: 0,
    weekOffset: 0,
    hideEmpty: false,
    activeScheduleDay: '',

    ...AsteriskShell.behavior('km'),

    roleLabel: 'KM',
    contextAssignments: [],
    contextSwitching: false,

    navSections: [
      { title: 'UTAMA', items: [
        { id: 'dashboard', label: 'Dashboard' },
      ] },
      { title: 'AKADEMIK', items: [
        { id: 'tugas', label: 'Tugas' },
        { id: 'jadwal', label: 'Jadwal Kuliah' },
        { id: 'ruangan', label: 'Ruangan' },
        { id: 'materi', label: 'Materi' },
      ] },
      { title: 'KELOLA KELAS', items: [
        { id: 'anggota', label: 'Anggota & PJ' },
        { id: 'pengaturan', label: 'Pengaturan Kelas' },
        { id: 'log', label: 'Riwayat Audit' },
      ] },
    ],

    get nav() { return this.navSections.flatMap(s => s.items); },
    get kmNav() { return this.nav; },
    get roleSub() { return 'Pengelola seluruh kelas'; },

    isActive(item) { const a = item.active || [item.id]; return a.includes(this.view); },

    pinnedNav(id) { return (this.nav || []).find(n => n.id === id) || null; },

    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed;
      try { localStorage.setItem('asterisk:sidebar:collapsed', this.sidebarCollapsed ? '1' : '0'); } catch (e) {}
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
        window.location.href = role === 'PJ' ? '/pj.html' : role === 'SYSTEM_ADMIN' ? '/system-admin.html' : '/km.html';
      } catch (err) {
        this.showToast(err.message || 'Konteks akses tidak tersedia.');
      } finally { this.contextSwitching = false; }
    },

    knownViews: ['dashboard', 'tugas', 'antrean', 'jadwal', 'ruangan', 'materi', 'semester', 'anggota', 'pengaturan', 'usulan', 'monitoring', 'log', 'notifikasi', 'akun'],

    showPageError(status) {
      this.pageState = { status: status };
      this.drawer = false;
      this.view = '__error';
      window.scrollTo({ top: 0 });
    },

    todayName: 'Senin',
    todayFull: '',
    currentTime: '',
    selectedClass: '',
    classSlug: '',
    classList: [],
    botOnline: false,
    fullSchedule: [],
    tasks: [],
    offeringList: [],
    offeringLoading: false,
    offeringState: 'idle',
    semesterId: '',
    semesterStatus: '',
    portalCodeReveal: '',

    tugasSub: 'list',
    tugasTab: 'aktif',
    tugasDari: '', tugasSampai: '',
    tugasLoading: false, tugasListError: '',
    tugasMatkul: 'Semua',
    tugasRentang: '',
    tugasSort: 'dekat',
    tugasForm: { offeringId: '', judul: '', tanggal: '', jam: '', deskripsi: '', kumpul: '', kumpulUrl: '', jenis: 'Individu' },
    tugasError: '',
    tugasDetail: null, tugasReviews: [], tugasDetailLoading: false, tugasDetailTab: 'detail',
    tugasPreview: null, tugasPublishMsg: '', tugasPublishing: false,
    editTugasId: '', editTugasVersion: 0, tugasConflict: null,
    patternsList: [],

    anggotaSub: 'list',
    undang: { offeringId: '', nomor: '' },
    undangError: '',
    undangLink: '',

    // Semester (draf manual/salin, impor JSON, preview, aktivasi, arsip).
    semesterList: [],
    semesterOptions: [],
    semesterLoading: false,
    semesterError: '',
    semesterForm: { academic_year: '', term: 'Ganjil', starts_on: '', ends_on: '', source_semester_id: '' },
    semesterFormError: '',
    semesterFormErrorField: '',
    semesterSaving: false,
    semesterAktif: null,
    semesterAktifAlasan: '',
    semesterPreview: null,
    semesterPreviewLoading: false,
    semesterPreviewError: '',
    imporFile: null,
    imporHasil: null,
    imporError: '',
    imporSaving: false,
    arsipDetail: null,
    arsipOfferings: [],
    arsipLoading: false,
    offeringForm: { course_code: '', activity_type: 'TEORI', display_name: '', lecturer_codes: '' },
    offeringFormError: '',
    offeringSaving: false,
    offeringTargetId: null,

    // Materi (list + tambah; ubah/arsip tunda).
    materiList: [],
    materiLoading: false,
    materiError: '',
    materiFilterOffering: '',
    materiSort: 'terbaru',
    materiFormOpen: false,
    materiForm: { offeringId: '', title: '', material_type: 'DOCUMENT', url: '', description: '' },
    materiFormError: '',
    materiSaving: false,
    editMateriId: null, editMateriVersion: 0,

    // Notifikasi (list + retry + attempts; tiru pola System Admin).
    notifList: [],
    notifLoading: false,
    notifError: '',
    notifFilter: '',
    notifDetailId: null,
    notifAttempts: [],
    notifAttemptsLoading: false,
    notifAttemptsError: '',

    // Penugasan Peran (daftar PJ + undangan + tangguhkan/cabut).
    penugasanList: [],
    penugasanLoading: false,
    penugasanError: '',
    undanganList: [],
    undanganLoading: false,
    undanganError: '',
    penugasanAksi: null,
    penugasanAlasan: '',
    penugasanError: '',
    penugasanSaving: false,
    penugasanPemicu: null,

    reviewId: null,
    reviewMode: 'koreksi',
    reviewNote: '',

    // Pengaturan kelas (ditulis ke server; berlaku pengiriman berikutnya).
    pengaturan: {
      pagi: '06:00',
      sore: '17:00',
      gantiMenit: 60,
      zona: 'Asia/Jakarta'
    },
    pengaturanSaving: false,
    pengaturanFormError: '',
    classSettings: null,
    settingsLoading: false,
    settingsError: '',
    rotatingCode: false,

    toast: { show: false, message: '', timer: null },

    get todayList() {
      return this.fullSchedule.filter(s => s.hari === this.todayName);
    },

    get sapaanWaktu() {
      const now = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
      const h = now.getHours();
      if (h < 11) return 'Selamat pagi';
      if (h < 15) return 'Selamat siang';
      if (h < 18) return 'Selamat sore';
      return 'Selamat malam';
    },

    get currentSession() {
      const now = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
      const pad = n => String(n).padStart(2, '0');
      const nowHM = `${pad(now.getHours())}:${pad(now.getMinutes())}`;
      return (this.todayList || []).find(s => {
        if (!s.timeStart || !s.timeEnd) return false;
        return nowHM >= s.timeStart && nowHM <= s.timeEnd;
      }) || null;
    },

    get nextSession() {
      const now = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
      const pad = n => String(n).padStart(2, '0');
      const nowHM = `${pad(now.getHours())}:${pad(now.getMinutes())}`;
      return (this.todayList || [])
        .filter(s => s.timeStart && s.timeStart > nowHM)
        .slice()
        .sort((a, b) => a.timeStart.localeCompare(b.timeStart))[0] || null;
    },

    isSessionCurrent(s) {
      if (!s || !s.timeStart || !s.timeEnd) return false;
      const now = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
      const pad = n => String(n).padStart(2, '0');
      const nowHM = `${pad(now.getHours())}:${pad(now.getMinutes())}`;
      return nowHM >= s.timeStart && nowHM <= s.timeEnd;
    },

    isSessionPassed(s) {
      if (!s || !s.timeEnd) return false;
      const now = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
      const pad = n => String(n).padStart(2, '0');
      const nowHM = `${pad(now.getHours())}:${pad(now.getMinutes())}`;
      return nowHM > s.timeEnd;
    },

    isSessionNext(s) {
      return !!(this.nextSession && s && String(this.nextSession.id) === String(s.id));
    },

    get replacementCount() {
      return (this.fullSchedule || []).filter(s => String(s.eventKind || '').toUpperCase() === 'REPLACEMENT').length;
    },

    get withUrgency() {
      const rank = { mendesak: 0, mendekati: 1, aman: 2 };
      return [...this.tugasAktif].sort((a, b) => (rank[a.urgency] ?? 2) - (rank[b.urgency] ?? 2));
    },

    get tugasAktif() {
      return (this.tasks || []).filter(t =>
        String(t.publication_status || '').toUpperCase() === 'PUBLISHED' && !t.completed_at && !t.archived_at);
    },

    get antrean() {
      return (this.tasks || [])
        .filter(t => String(t.publication_status || '').toUpperCase() === 'PUBLISHED'
          && String(t.review_state || 'NOT_REVIEWED').toUpperCase() === 'NOT_REVIEWED'
          && !t.completed_at && !t.archived_at)
        .slice().sort((a, b) => this.bandingDeadline(a.deadline_at, b.deadline_at, true));
    },

    get tugasTabCounts() {
      const c = { aktif: 0, draf: 0, review: 0, selesai: 0, terlewat: 0, arsip: 0, selesai_terlewat: 0 };
      (this.tasks || []).forEach(t => {
        if (t.archived_at) { c.arsip++; return; }
        if (t.completed_at) { c.selesai++; c.selesai_terlewat++; return; }
        if (String(t.publication_status || '').toUpperCase() === 'DRAFT') { c.draf++; return; }
        if (String(t.review_state || 'NOT_REVIEWED').toUpperCase() === 'NOT_REVIEWED') { c.review++; return; }
        if (this.lewatDeadline(t)) { c.terlewat++; c.selesai_terlewat++; return; }
        c.aktif++;
      });
      return c;
    },

    get tugasTabList() {
      const dari = this.tugasDari ? new Date(this.tugasDari + 'T00:00:00+07:00') : null;
      const sampai = this.tugasSampai ? new Date(this.tugasSampai + 'T23:59:59+07:00') : null;
      return (this.tasks || []).filter(t => {
        if (this.tugasMatkul !== 'Semua' && t.matkul !== this.tugasMatkul) return false;
        if (t.archived_at) { if (this.tugasTab !== 'arsip') return false; }
        else if (t.completed_at) { if (this.tugasTab !== 'selesai' && this.tugasTab !== 'selesai_terlewat') return false; }
        else if (String(t.publication_status || '').toUpperCase() === 'DRAFT') { if (this.tugasTab !== 'draf') return false; }
        else if (String(t.review_state || 'NOT_REVIEWED').toUpperCase() === 'NOT_REVIEWED') { if (this.tugasTab !== 'review') return false; }
        else if (this.lewatDeadline(t)) { if (this.tugasTab !== 'terlewat' && this.tugasTab !== 'selesai_terlewat') return false; }
        else if (this.tugasTab !== 'aktif') return false;

        if (this.tugasRentang) {
          const d = t.deadline_at ? new Date(t.deadline_at) : null;
          if (d) {
            const diffDays = (d - new Date()) / (3600000 * 24);
            const days = parseInt(this.tugasRentang, 10);
            if (days && (diffDays < -days || diffDays > days)) return false;
          }
        }
        if (dari || sampai) {
          const d = t.deadline_at ? new Date(t.deadline_at) : null;
          if (!d) return false;
          if (dari && d < dari) return false;
          if (sampai && d > sampai) return false;
        }
        if (!this.shellMatchesSearch('tugas', [t.title, t.deskripsi, t.instructions, t.matkul])) return false;
        return true;
      }).slice().sort((a, b) => this.bandingDeadline(a.deadline_at, b.deadline_at, this.tugasSort === 'dekat'));
    },

    tugasTabTitle() {
      if (this.tugasTab === 'review') return 'Daftar Perlu Review';
      if (this.tugasTab === 'draf') return 'Daftar Draf Tugas';
      if (this.tugasTab === 'selesai_terlewat' || this.tugasTab === 'selesai' || this.tugasTab === 'terlewat') return 'Daftar Tugas Selesai & Terlewat';
      if (this.tugasTab === 'arsip') return 'Daftar Arsip Tugas';
      return 'Daftar Tugas Aktif';
    },

    get tugasFilterAktif() { return !!(this.tugasDari || this.tugasSampai || this.tugasRentang || (this.tugasMatkul && this.tugasMatkul !== 'Semua')); },
    hapusTugasFilter() { this.tugasDari = ''; this.tugasSampai = ''; this.tugasRentang = ''; this.tugasMatkul = 'Semua'; },

    pubLabel(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'DRAFT') return 'Draf';
      if (s === 'PUBLISHED') return 'Terbit';
      if (s === 'REVOKED') return 'Publikasi Dicabut';
      return st || '-';
    },

    reviewLabel(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'NOT_REVIEWED') return 'Perlu Review';
      if (s === 'APPROVED') return 'Disetujui';
      if (s === 'CHANGES_REQUESTED') return 'Perlu koreksi';
      if (s === 'REVOKED') return 'Dibatalkan';
      return st || '-';
    },

    fmtDeadlineID(iso) { return API.fmtDeadlineID(iso); },
    lewatDeadline(t) { return API.lewatDeadline(t); },
    waktuDeadline(iso) { return API.waktuDeadline(iso); },

    // Pembanding deadline; tanpa deadline selalu paling bawah.
    bandingDeadline(aIso, bIso, dekat) {
      const da = this.waktuDeadline(aIso), db = this.waktuDeadline(bIso);
      if (isNaN(da) && isNaN(db)) return 0;
      if (isNaN(da)) return 1;
      if (isNaN(db)) return -1;
      return dekat ? da - db : db - da;
    },

    deadlineBadge(iso, completed) { return API.deadlineBadge(iso, completed); },

    fmtDeadlineShort(iso) {
      if (!iso) return '-';
      try {
        const d = new Date(iso);
        if (isNaN(d.getTime())) return String(iso);
        const now = new Date();
        const tomorrow = new Date(now);
        tomorrow.setDate(now.getDate() + 1);
        const isToday = d.toDateString() === now.toDateString();
        const isTomorrow = d.toDateString() === tomorrow.toDateString();
        const timePart = d.toLocaleTimeString('id-ID', {
          timeZone: 'Asia/Jakarta',
          hour: '2-digit',
          minute: '2-digit',
          hour12: false
        }).replace('.', ':') + ' WIB';
        if (isToday) return `Hari ini, ${timePart}`;
        if (isTomorrow) return `Besok, ${timePart}`;
        const dayNames = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
        const dayName = dayNames[d.getDay()];
        return `${dayName}, ${timePart}`;
      } catch (e) {
        return String(iso);
      }
    },

    deadlineRelative(iso) {
      if (!iso) return 'Aktif';
      try {
        const d = new Date(iso);
        if (isNaN(d.getTime())) return 'Aktif';
        const now = new Date();
        const diffH = (d - now) / 3600000;
        if (diffH < 0) return 'Terlewat';
        if (diffH <= 48) {
          const jam = Math.max(1, Math.round(diffH));
          return `Sisa ${jam} jam`;
        }
        const hari = Math.ceil(diffH / 24);
        return `Sisa ${hari} hari`;
      } catch (e) {
        return 'Aktif';
      }
    },

    fmtDeadlineDetailed(iso) {
      if (!iso) return '-';
      try {
        const d = new Date(iso);
        if (isNaN(d.getTime())) return String(iso);
        const now = new Date();
        const tomorrow = new Date(now);
        tomorrow.setDate(now.getDate() + 1);
        const isToday = d.toDateString() === now.toDateString();
        const isTomorrow = d.toDateString() === tomorrow.toDateString();
        const timePart = d.toLocaleTimeString('id-ID', {
          timeZone: 'Asia/Jakarta',
          hour: '2-digit',
          minute: '2-digit',
          hour12: false
        }).replace('.', ':') + ' WIB';
        const dayNames = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
        const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
        const dayName = isToday ? 'Hari ini' : (isTomorrow ? 'Besok' : dayNames[d.getDay()]);
        const datePart = `${d.getDate()} ${monthNames[d.getMonth()]}`;
        return `${dayName}, ${datePart} · ${timePart}`;
      } catch (e) {
        return String(iso);
      }
    },

    urgencyStyle(iso) {
      if (!iso) return { bg: 'bg-[#DCFCE7]', text: 'text-[#166534]', dot: 'bg-[#16A34A]' };
      try {
        const d = new Date(iso);
        if (isNaN(d.getTime())) return { bg: 'bg-[#DCFCE7]', text: 'text-[#166534]', dot: 'bg-[#16A34A]' };
        const now = new Date();
        const diffH = (d - now) / 3600000;
        if (diffH <= 28) {
          return { bg: 'bg-[#FEE2E2]', text: 'text-[#991B1B]', dot: 'bg-[#DC2626]' };
        }
        if (diffH <= 72) {
          return { bg: 'bg-[#FEF3C7]', text: 'text-[#92400E]', dot: 'bg-[#D97706]' };
        }
        return { bg: 'bg-[#DCFCE7]', text: 'text-[#166534]', dot: 'bg-[#16A34A]' };
      } catch (e) {
        return { bg: 'bg-[#DCFCE7]', text: 'text-[#166534]', dot: 'bg-[#16A34A]' };
      }
    },

    waktuRelatif(iso) {
      if (!iso) return '';
      try {
        const d = new Date(iso);
        if (isNaN(d.getTime())) return '';
        const now = new Date();
        const diffMs = now - d;
        const diffMin = Math.round(diffMs / 60000);
        if (diffMin < 1) return 'Baru saja';
        if (diffMin < 60) return `${diffMin} menit yang lalu`;
        const diffH = Math.round(diffMin / 60);
        if (diffH < 24) return `${diffH} jam yang lalu`;
        const diffDay = Math.round(diffH / 24);
        if (diffDay === 1) {
          const pad = n => String(n).padStart(2, '0');
          return `Kemarin, ${pad(d.getHours())}:${pad(d.getMinutes())} WIB`;
        }
        if (diffDay < 7) return `${diffDay} hari yang lalu`;
        return d.toLocaleDateString('id-ID', { timeZone: 'Asia/Jakarta', day: 'numeric', month: 'short' });
      } catch (e) {
        return '';
      }
    },

    get urgent3() { return this.withUrgency.slice(0, 3); },
    get nearCount() { return this.withUrgency.filter(t => t.urgency !== 'aman').length; },

    get matkulOpts() {
      const set = new Set(this.fullSchedule.map(s => s.matkul));
      return Array.from(set);
    },

    get roomList() {
      return Array.from(new Set(this.fullSchedule.map(s => s.ruang).filter(Boolean)));
    },

    get weekDays() {
      const names = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat'];
      const now = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
      const dow = (now.getDay() + 6) % 7;
      const monday = new Date(now);
      monday.setDate(now.getDate() - dow + this.weekOffset * 7);
      return names.map((n, i) => {
        const d = new Date(monday);
        d.setDate(monday.getDate() + i);
        const dateShort = d.toLocaleDateString('id-ID', { day: 'numeric', month: 'short', timeZone: 'Asia/Jakarta' });
        return { name: n, dateNum: d.getDate(), dateShort, full: d };
      });
    },

    get currentScheduleDay() {
      if (this.activeScheduleDay) return this.activeScheduleDay;
      return ['Sabtu', 'Minggu'].includes(this.todayName) ? 'Senin' : this.todayName;
    },

    getActiveDayDate() {
      const found = this.weekDays.find(d => d.name === this.currentScheduleDay);
      return found ? found.dateShort : '';
    },

    get weekLabel() {
      const fmt = (d) => d.getDate() + ' ' + d.toLocaleString('id-ID', { month: 'short', timeZone: 'Asia/Jakarta' });
      const days = this.weekDays;
      return `${fmt(days[0].full)} – ${fmt(days[4].full)} ${days[4].full.getFullYear()}`;
    },

    get slots() {
      const base = ['08', '10', '13', '15'];
      const fromData = (this.fullSchedule || []).map(s => String(s.timeStart || '').split(':')[0].padStart(2, '0')).filter(h => /^\d{2}$/.test(h));
      return Array.from(new Set([...base, ...fromData])).sort();
    },

    get calendarCells() {
      const cells = [];
      for (const slot of this.slots) {
        for (const d of this.weekDays) {
          cells.push({ day: d.name, slot: slot, key: d.name + '-' + slot });
        }
      }
      return cells;
    },

    isToday(dayName) { return dayName === this.todayName; },

    slotKind(matkul) {
      const s = String(matkul || '').toLowerCase();
      if (s.includes('praktikum') || s.includes('praktik')) return 'Praktikum';
      if (s.includes('teori')) return 'Teori';
      return '';
    },

    slotRange(entry, slotHH) {
      if (entry && entry.timeStart) {
        return entry.timeEnd ? `${entry.timeStart}–${entry.timeEnd}` : entry.timeStart;
      }
      return slotHH + '.00';
    },

    daySessions(dayName) {
      const sessions = this.fullSchedule
        .filter(s => s.hari === dayName)
        .filter(s => this.shellMatchesSearch('jadwal', [s.matkul, s.dosen, s.ruang]))
        .slice()
        .sort((a, b) => String(a.timeStart || '').localeCompare(String(b.timeStart || '')));
      if (this.hideEmpty) return sessions;
      return sessions;
    },

    goPindah(dayName, slotHH, session) {
      this.mulaiUbah({ dayName: dayName, slotHH: slotHH, session: session });
    },

    // Samakan kode kelas legacy (mis. D4-TI-SMT3-A) ke slug kanonis v1
    // (mis. d4-ti-smt3-a) memakai active_assignment + daftar classes dari /me.
    resolveClassSlug(me) {
      const asg = (me && me.active_assignment) || {};
      if (asg.role === 'SYSTEM_ADMIN') {
        const requested = new URLSearchParams(window.location.search).get('class');
        if (requested) return requested;
      }
      if (asg.class_slug) return String(asg.class_slug);
      const list = (me && me.classes) || [];
      const code = String(this.selectedClass || '').toLowerCase();
      if (code && Array.isArray(list)) {
        const hit = list.find(c => String(c.slug || '') === code || String(c.code || '').toLowerCase() === code);
        if (hit && hit.slug) return String(hit.slug);
      }
      return String(this.selectedClass || '');
    },

    slotAt(dayName, slotHH) {
      return this.fullSchedule.find(s => s.hari === dayName && parseInt((s.timeStart || '0').split(':')[0], 10) === parseInt(slotHH, 10));
    },

    // ===== Alur Perubahan Jadwal (mengikuti docs/design/flows/SCHEDULE_MANAGEMENT.md) =====
    jadwalSub: 'daftar',
    filtMatkul: '', filtDosen: '', filtRuang: '',
    polaLoading: false, polaError: '',
    polaForm: { id: '', version: 0, offeringId: '', day: '1', start: '', end: '', roomId: '', link: '', effectiveDate: '' },
    polaFormError: '',
    polaPreview: null, polaPreviewPayload: '', polaPreviewLoading: false,
    roomSearch: { date: '', start: '', end: '' }, roomCandidates: null, roomCandidatesLoading: false, roomCandidatesError: '',
    roomHistory: [], roomHistoryLoading: false, roomHistoryError: '',
    ubahForm: { offeringId: '', kind: 'REPLACEMENT', scope: 'sementara', originPatternId: '', originDate: '', date: '', day: '1', start: '', end: '', roomId: '', link: '', reason: '', effectiveDate: '', participantIds: '', conflictReason: '' },
    ubahFormError: '',
    draftEvent: null,
    previewData: null, previewLoading: false, previewError: '',
    eventsList: [], eventsLoading: false, eventsError: '',
    eventFilter: 'semua',
    selectedEvent: null, revokeReason: '', revokeError: '',
    eventDetailLoading: false, eventDetailError: '',
    roomCands: [], roomCandsLoading: false,
    // Konfirmasi TU: satu objek (open/room/form/saving/error).
    tuConfirm: { open: false, room: null, form: { status: 'CONFIRMED', name: '', note: '' }, saving: false, error: '' },

    get polaRooms() {
      const map = new Map();
      (this.patternsList || []).forEach(p => {
        if (p.room_id && !map.has(String(p.room_id))) {
          map.set(String(p.room_id), { id: String(p.room_id), code: p.room || p.room_code || ('Ruang ' + p.room_id) });
        }
      });
      return Array.from(map.values());
    },

    get scopePatterns() {
      const ids = new Set((this.offeringList || []).map(o => String(o.id)));
      if (ids.size === 0) return this.patternsList || [];
      return (this.patternsList || []).filter(p => ids.has(String(p.course_offering_id || '')));
    },

    async pilihSemesterJadwal(id) {
      const sid = String(id || '');
      if (!sid || sid === String(this.semesterId)) return;
      this.semesterId = sid;
      const opt = (this.semesterOptions || []).find(s => String(s.id) === sid);
      this.semesterStatus = opt ? String(opt.status || '').toUpperCase() : '';
      this.offeringLoading = true;
      try {
        const offerings = await API.getSemesterOfferings(sid).catch(() => null);
        if (offerings === null) { this.offeringState = 'error'; this.offeringList = []; return; }
        this.offeringList = Array.isArray(offerings) ? offerings : [];
        this.offeringState = this.offeringList.length > 0 ? 'ok' : 'empty-offering';
      } finally {
        this.offeringLoading = false;
      }
    },

    get filteredPola() {
      const qm = this.filtMatkul.toLowerCase(), qd = this.filtDosen.toLowerCase(), qr = this.filtRuang.toLowerCase();
      return this.scopePatterns.filter(p => {
        const mk = String(p.display_name || p.course_name || p.offering || '').toLowerCase();
        const ds = String(p.dosen || p.lecturer || '').toLowerCase();
        const rg = String(p.room || p.room_code || '').toLowerCase();
        return (!qm || mk.includes(qm)) && (!qd || ds.includes(qd)) && (!qr || rg.includes(qr));
      });
    },

    get polaFilterActive() { return !!(this.filtMatkul || this.filtDosen || this.filtRuang); },

    get filteredEvents() {
      if (this.eventFilter === 'semua') return this.eventsList;
      return this.eventsList.filter(e => String(e.lifecycle_status || '').toUpperCase() === this.eventFilter);
    },

    get blockingConflicts() { return (this.previewData && this.previewData.conflicts || []).filter(c => c.blocking); },
    get overrideConflicts() { return (this.previewData && this.previewData.conflicts || []).filter(c => !c.blocking); },

    polaHariName(num) {
      const names = { 1: 'Senin', 2: 'Selasa', 3: 'Rabu', 4: 'Kamis', 5: 'Jumat', 6: 'Sabtu', 7: 'Minggu' };
      return names[Number(num)] || '-';
    },

    kindLabel(kind) {
      const k = String(kind || '').toUpperCase();
      if (k === 'REPLACEMENT') return 'Kelas Pengganti';
      if (k === 'EXTRA') return 'Kelas Tambahan';
      if (k === 'HOLIDAY') return 'Hari Libur';
      if (k === 'SESSION_CANCELLED') return 'Sesi Dibatalkan';
      return kind || '-';
    },

    kindDescription(kind) {
      const k = String(kind || '').toUpperCase();
      if (k === 'REPLACEMENT') return 'Pindahkan satu sesi ke waktu baru.';
      if (k === 'EXTRA') return 'Tambahkan sesi di luar jadwal rutin.';
      if (k === 'HOLIDAY') return 'Tandai waktu libur pada tanggal tertentu.';
      if (k === 'SESSION_CANCELLED') return 'Batalkan satu sesi yang terjadwal.';
      return '';
    },

    statusLabel(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'DRAFT') return 'Draf';
      if (s === 'PUBLISHED') return 'Terbit';
      if (s === 'REVOKED') return 'Publikasi Dicabut';
      return st || '-';
    },

    canEditPattern(p) { return true; },

    isDraftSemester() { return String(this.semesterStatus || '').toUpperCase() === 'DRAFT'; },

    hapusPolaFilter() { this.filtMatkul = ''; this.filtDosen = ''; this.filtRuang = ''; },

    bukaDaftar() { this.jadwalSub = 'daftar'; window.scrollTo({ top: 0 }); },

    mulaiUbah(prefill) {
      const f = this.ubahForm;
      f.offeringId = (prefill && prefill.offeringId) || f.offeringId || '';
      f.kind = (prefill && prefill.kind) || 'REPLACEMENT';
      f.scope = 'sementara';
      f.originPatternId = ''; f.originDate = ''; f.date = ''; f.day = '1'; f.start = ''; f.end = '';
      f.roomId = ''; f.link = ''; f.reason = ''; f.effectiveDate = ''; f.participantIds = ''; f.conflictReason = '';
      if (prefill && prefill.session) {
        const s = prefill.session;
        const pat = (this.patternsList || []).find(p =>
          String(p.display_name || p.course_name || p.offering || '') === String(s.matkul));
        if (pat) {
          f.originPatternId = String(pat.id);
          if (pat.course_offering_id) f.offeringId = String(pat.course_offering_id);
        }
        if (s.timeStart) f.start = s.timeStart.slice(0, 5);
        if (s.timeEnd) f.end = s.timeEnd.slice(0, 5);
      } else if (prefill && prefill.dayName) {
        const hit = this.slotAt(prefill.dayName, prefill.slotHH || '');
        if (hit) {
          const pat = (this.patternsList || []).find(p =>
            String(p.display_name || p.course_name || p.offering || '') === String(hit.matkul));
          if (pat) {
            f.originPatternId = String(pat.id);
            if (pat.course_offering_id) f.offeringId = String(pat.course_offering_id);
          }
          if (hit.timeStart) f.start = hit.timeStart.slice(0, 5);
          if (hit.timeEnd) f.end = hit.timeEnd.slice(0, 5);
        }
      }
      this.ubahFormError = '';
      this.view = 'jadwal'; this.jadwalSub = 'ubah';
      window.scrollTo({ top: 0 });
    },

    mulaiTambahPola() {
	  this.polaForm = { id: '', version: 0, offeringId: '', day: '1', start: '', end: '', duration: 100, roomId: '', link: '', effectiveDate: '' };
      this.polaFormError = '';
      this.polaPreview = null;
      this.view = 'jadwal'; this.jadwalSub = 'pola';
      window.scrollTo({ top: 0 });
    },

    editPola(p) {
      this.polaForm = { id: String(p.id), version: p.version || 0, offeringId: String(p.course_offering_id || ''), day: String(p.day_of_week || '1'), start: (p.start_time || '').slice(0, 5), end: (p.end_time || '').slice(0, 5), roomId: p.room_id ? String(p.room_id) : '', link: p.meeting_link || '', effectiveDate: '' };
      this.polaFormError = '';
      this.view = 'jadwal'; this.jadwalSub = 'pola';
      window.scrollTo({ top: 0 });
    },

    async hapusPola(p) {
      if (!p.id || !p.version) { this.showToast('Data pola tidak lengkap.'); return; }
      const isDraf = this.isDraftSemester();
      const ok = window.confirm(isDraf
        ? 'Hapus jadwal draf ini? Baris dihapus sungguhan dan tidak dapat dikembalikan.'
        : 'Hapus jadwal tetap ini? Jadwal tidak aktif mulai hari ini. Versi lama tetap tersimpan.');
      if (!ok) return;
      try {
        await API.deletePattern(p.id, p.version);
        this.showToast(isDraf ? 'Jadwal draf dihapus.' : 'Jadwal tetap dihapus (tidak aktif mulai hari ini).');
        this.patternsList = await API.getPatterns().catch(() => []);
      } catch (e) {
        this.showToast(e.message || 'Gagal menghapus jadwal tetap.');
      }
    },

    async hapusDrafPerubahan(ev) {
      if (!ev.id || !ev.version) { this.showToast('Data draf tidak lengkap.'); return; }
      const ok = window.confirm('Hapus draf perubahan jadwal ini? Tindakan ini tidak dapat dibatalkan.');
      if (!ok) return;
      try {
        await API.deleteTeachingEvent(ev.id, ev.version);
        this.showToast('Draf perubahan jadwal dihapus.');
        await this.loadEvents();
      } catch (e) {
        this.showToast(e.message || 'Gagal menghapus draf perubahan.');
      }
    },

    durasiMenit(start, end) {
      const m = String(start || '').match(/(\d{1,2}):(\d{2})/);
      const n = String(end || '').match(/(\d{1,2}):(\d{2})/);
      if (!m || !n) return 0;
      return (parseInt(n[1], 10) * 60 + parseInt(n[2], 10)) - (parseInt(m[1], 10) * 60 + parseInt(m[2], 10));
    },

    hitungAkhirPola() {
      const f = this.polaForm;
      const parts = String(f.start || '').match(/^(\d{2}):(\d{2})$/);
      const total = parts ? Number(parts[1]) * 60 + Number(parts[2]) + Number(f.duration || 0) : 0;
      f.end = total > 0 && total < 1440 ? String(Math.floor(total / 60)).padStart(2, '0') + ':' + String(total % 60).padStart(2, '0') : '';
      this.polaPreview = null;
    },

    payloadPola() {
      const f = this.polaForm;
      const payload = { offering_id: Number(f.offeringId), day_of_week: Number(f.day), start_time: f.start, duration_min: Number(f.duration) };
      if (f.roomId) payload.room_id = Number(f.roomId);
      if ((f.link || '').trim()) payload.meeting_link = f.link.trim();
      return payload;
    },

    async terbitPola() {
      if (!this.polaPreview || !this.polaPreview.can_publish) return;
	  if (this.polaPreviewPayload !== JSON.stringify(this.payloadPola())) { this.polaFormError = 'Form berubah. Muat ulang pratinjau sebelum menerbitkan.'; return; }
      this.polaPreviewLoading = true; this.polaFormError = '';
      try {
        await API.createPattern(this.payloadPola());
        this.showToast('Jadwal tetap ditambahkan.');
        this.polaPreview = null;
        await this.loadPatterns();
        this.jadwalSub = 'daftar';
      } catch (e) { this.polaFormError = e.message || 'Gagal menerbitkan jadwal.'; }
      finally { this.polaPreviewLoading = false; }
    },

    async simpanPola() {
      const f = this.polaForm;
      if (!f.offeringId) { this.polaFormError = 'Pilih mata kuliah di kelas ini dulu.'; return; }
	  if (!f.start || !f.end) { this.polaFormError = 'Isi jam mulai dan durasi yang valid.'; return; }
      const dur = this.durasiMenit(f.start, f.end);
      if (dur <= 0) { this.polaFormError = 'Jam selesai harus setelah jam mulai.'; return; }
      if (f.id) {
        this.mulaiUbah({ offeringId: f.offeringId });
        Object.assign(this.ubahForm, { scope: 'permanen', originPatternId: String(f.id), day: String(f.day),
          start: f.start, end: f.end, roomId: f.roomId, link: f.link, effectiveDate: f.effectiveDate });
        return;
      }
      this.polaFormError = '';
      try {
		this.polaPreviewLoading = true;
		this.polaPreviewPayload = JSON.stringify(this.payloadPola());
		this.polaPreview = await API.previewCreatePattern(this.payloadPola());
      } catch (err) {
		this.polaFormError = err.message || 'Gagal meninjau jadwal tetap.';
	  } finally { this.polaPreviewLoading = false; }
    },

    async loadPatterns() {
      this.polaLoading = true; this.polaError = '';
      try {
        this.patternsList = await API.getPatterns();
      } catch (e) {
        this.patternsList = []; this.polaError = 'Jadwal tetap belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.polaLoading = false;
      }
    },

    async loadEvents() {
      this.eventsLoading = true; this.eventsError = '';
      try {
        this.eventsList = await API.getTeachingEvents();
      } catch (e) {
        this.eventsList = []; this.eventsError = 'Perubahan jadwal belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.eventsLoading = false;
      }
    },

    validasiUbah() {
      const f = this.ubahForm;
      if (!f.offeringId) return 'Pilih mata kuliah di kelas ini dulu.';
      if (f.scope === 'permanen') {
        if (!f.originPatternId) return 'Pilih jadwal tetap yang akan diganti.';
        if (!f.start || !f.end || this.durasiMenit(f.start, f.end) <= 0) return 'Isi jam mulai dan selesai yang valid.';
        if (this.isDraftSemester()) return '';
        if (!f.effectiveDate) return 'Isi tanggal mulai berlaku.';
        if (!f.reason || f.reason.trim().length < 5) return 'Keterangan minimal 5 karakter.';
        return '';
      }
      if ((f.kind === 'REPLACEMENT' || f.kind === 'SESSION_CANCELLED') && !f.originPatternId) return 'Pilih jadwal semula untuk kelas pengganti atau sesi yang dibatalkan.';
      if ((f.kind === 'REPLACEMENT' || f.kind === 'SESSION_CANCELLED') && !f.originDate) return 'Isi tanggal kejadian asal.';
      if (f.kind !== 'SESSION_CANCELLED' && (!f.date || !f.start || !f.end)) return 'Tanggal serta jam mulai dan selesai wajib diisi.';
      if (f.kind !== 'SESSION_CANCELLED' && this.durasiMenit(f.start, f.end) <= 0) return 'Jam selesai harus setelah jam mulai.';
      if (!f.reason || f.reason.trim().length < 5) return 'Keterangan minimal 5 karakter.';
      if (f.scope === 'permanen' && !f.effectiveDate) return 'Isi tanggal mulai berlaku untuk perubahan permanen.';
      return '';
    },

    rakitPayloadPermanen() {
      const f = this.ubahForm;
      const pattern = (this.patternsList || []).find(p => String(p.id) === String(f.originPatternId));
      if (!pattern) throw new Error('Jadwal tetap asal belum termuat. Muat ulang daftar jadwal.');
      const payload = { version: Number(pattern.version), day_of_week: Number(f.day), start_time: f.start,
        duration_min: this.durasiMenit(f.start, f.end) };
      if (!this.isDraftSemester()) {
        payload.effective_from = f.effectiveDate;
        payload.reason = f.reason.trim();
      } else if ((f.reason || '').trim()) {
        payload.reason = f.reason.trim();
      }
      if (f.roomId) payload.room_id = Number(f.roomId);
      if ((f.link || '').trim()) payload.meeting_link = f.link.trim();
      return payload;
    },

    pilihPolaAsal() {
      const p = (this.patternsList || []).find(item => String(item.id) === String(this.ubahForm.originPatternId));
      if (!p || this.ubahForm.scope !== 'permanen') return;
      this.ubahForm.day = String(p.day_of_week || 1);
      this.ubahForm.start = String(p.start_time || '').slice(0, 5);
      this.ubahForm.end = String(p.end_time || '').slice(0, 5);
      this.ubahForm.roomId = p.room_id ? String(p.room_id) : '';
      this.ubahForm.link = p.meeting_link || '';
    },

    rakitPayloadUbah() {
      const f = this.ubahForm;
      const origin = (this.patternsList || []).find(p => String(p.id) === String(f.originPatternId));
      let dateISO = f.date, startHM = f.start, endHM = f.end;
      if (f.kind === 'SESSION_CANCELLED' && origin) {
        dateISO = f.originDate;
        startHM = String(origin.start_time || '').slice(0, 5);
        endHM = String(origin.end_time || '').slice(0, 5);
      }
      const payload = {
        owner_offering_id: Number(f.offeringId),
        event_kind: f.kind,
        starts_at: `${dateISO}T${startHM}:00+07:00`,
        ends_at: `${dateISO}T${endHM}:00+07:00`,
        reason: f.reason.trim()
      };
      if ((f.kind === 'REPLACEMENT' || f.kind === 'SESSION_CANCELLED') && f.originPatternId) {
        payload.origin_pattern_id = Number(f.originPatternId);
        payload.origin_date = f.originDate;
      }
      if (f.roomId) payload.room_id = Number(f.roomId);
      if ((f.link || '').trim()) payload.meeting_link = f.link.trim();
      const parts = String(f.participantIds || '').split(',').map(s => Number(String(s).trim())).filter(n => n > 0);
      if (parts.length > 0) payload.participant_offering_ids = parts;
      return payload;
    },

    async simpanDrafPerubahan() {
      const err = this.validasiUbah();
      if (err) { this.ubahFormError = err; return; }
      this.ubahFormError = '';
      try {
        const draf = await API.createTeachingEventDraft(this.rakitPayloadUbah());
        this.draftEvent = draf;
        await this.loadEvents();
        this.showToast('Draf perubahan jadwal tersimpan. Draf hanya terlihat oleh pengurus.');
        this.jadwalSub = 'daftar';
      } catch (e) {
        this.ubahFormError = e.message || 'Gagal menyimpan draf.';
      }
    },

    async tinjauPerubahan() {
      const err = this.validasiUbah();
      if (err) { this.ubahFormError = err; return; }
      this.ubahFormError = ''; this.previewError = '';
      try {
        if (this.ubahForm.scope === 'permanen') {
          this.draftEvent = { id: Number(this.ubahForm.originPatternId), permanent: true };
          this.jadwalSub = 'tinjau';
          window.scrollTo({ top: 0 });
          await this.muatPratinjau();
          return;
        }
        const draf = await API.createTeachingEventDraft(this.rakitPayloadUbah());
        this.draftEvent = draf;
        this.jadwalSub = 'tinjau';
        window.scrollTo({ top: 0 });
        await this.muatPratinjau();
      } catch (e) {
        this.ubahFormError = e.message || 'Gagal membuat draf untuk ditinjau.';
      }
    },

    async muatPratinjau() {
      if (!this.draftEvent || !this.draftEvent.id) { this.previewError = 'Draf belum tersedia.'; return; }
      this.previewLoading = true; this.previewError = '';
      try {
        if (this.draftEvent.permanent) {
          this.previewData = await API.previewPattern(this.draftEvent.id, this.rakitPayloadPermanen());
          return;
        }
        this.previewData = await API.previewTeachingEvent(this.draftEvent.id);
        if (!this.previewData) this.previewError = 'Pratinjau belum dapat dimuat. Coba lagi.';
        if (this.previewData && this.previewData.new && this.previewData.new.starts_at) {
          await this.cariKandidatRuang(this.previewData.new.starts_at, this.previewData.new.ends_at).catch(() => {});
        }
      } catch (e) {
        this.previewError = 'Pratinjau belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.previewLoading = false;
      }
    },

    async terbitkanPerubahan() {
      if (!this.draftEvent || !this.draftEvent.id) return;
      if (this.blockingConflicts.length > 0) { this.showToast('Konflik pemblokir harus diselesaikan dulu.'); return; }
      if (this.draftEvent.permanent) {
        try {
          await API.patchPattern(this.draftEvent.id, this.rakitPayloadPermanen());
          this.patternsList = await API.getPatterns();
          this.showToast(this.isDraftSemester() ? 'Jadwal draf diperbarui.' : 'Jadwal tetap baru berlaku sesuai tanggal pilihan.');
          this.jadwalSub = 'daftar';
        } catch (e) { this.showToast(e.message || 'Gagal menerbitkan perubahan permanen.'); }
        return;
      }
      if (this.overrideConflicts.length > 0 && !(this.ubahForm.conflictReason || '').trim()) {
        this.showToast('Isi alasan pengecualian konflik sebelum menerbitkan.'); return;
      }
      // Peringatan: jika ruangan dipilih tapi belum konfirmasi TU
      if (this.ubahForm.roomId && this.draftEvent && !this.draftEvent.room_id) {
        const confirm = window.confirm('Anda memilih ruangan tapi belum mencatat konfirmasi TU. Lanjutkan tanpa konfirmasi TU? (Ruangan mungkin tidak tersedia saat publish)');
        if (!confirm) return;
      }
      try {
        await API.publishTeachingEvent(this.draftEvent.id, (this.ubahForm.conflictReason || '').trim() || null, this.draftEvent.version || 1);
        this.showToast('Perubahan jadwal terbit.');
        await this.loadEvents();
        const found = (this.eventsList || []).find(e => String(e.id) === String(this.draftEvent.id));
        this.selectedEvent = found || null;
        this.jadwalSub = 'detail';
        window.scrollTo({ top: 0 });
      } catch (e) {
        this.showToast(e.message || 'Gagal menerbitkan perubahan.');
      }
    },

    ubahLagi() { this.jadwalSub = 'ubah'; window.scrollTo({ top: 0 }); },

    async bukaDetail(ev) {
      this.selectedEvent = ev;
      this.revokeReason = ''; this.revokeError = '';
      this.eventDetailError = '';
      this.jadwalSub = 'detail';
      window.scrollTo({ top: 0 });
      this.eventDetailLoading = true;
      try {
        const detail = await API.getTeachingEvent(ev.id);
        this.selectedEvent = Object.assign({}, detail && detail.event || ev, {
          participants: detail && detail.participations || [],
          confirmations: detail && detail.confirmations || []
        });
      } catch (err) {
        this.eventDetailError = err.message || 'Detail perubahan jadwal gagal dimuat.';
      } finally { this.eventDetailLoading = false; }
    },

    async lanjutkanDraf(ev) {
      this.draftEvent = ev;
      const f = this.ubahForm;
      f.offeringId = String(ev.offering_id || f.offeringId || '');
      f.kind = ev.event_kind || 'REPLACEMENT';
      f.reason = ev.reason || '';
      f.date = String(ev.starts_at || '').slice(0, 10);
      f.start = String(ev.starts_at || '').slice(11, 16);
      f.end = String(ev.ends_at || '').slice(11, 16);
      this.jadwalSub = 'tinjau';
      window.scrollTo({ top: 0 });
      await this.muatPratinjau();
    },

    async cabutPublikasi() {
      const ev = this.selectedEvent;
      if (!ev) return;
      if (!this.revokeReason || this.revokeReason.trim().length < 5) {
        this.revokeError = 'Isi alasan pencabutan minimal 5 karakter.'; return;
      }
      this.revokeError = '';
      try {
        await API.revokeTeachingEvent(ev.id, this.revokeReason.trim(), ev.version || 0);
        this.showToast('Publikasi dicabut. Jadwal semula berlaku kembali.');
        await this.loadEvents();
        const found = (this.eventsList || []).find(e => String(e.id) === String(ev.id));
        this.selectedEvent = found || null;
      } catch (e) {
        this.revokeError = e.message || 'Gagal mencabut publikasi.';
      }
    },

    async aksiPartisipasi(ev, action) {
      try {
        await API.participateTeachingEvent(ev.id, action);
        const label = action === 'accept' ? 'diterima' : (action === 'decline' ? 'ditolak' : 'dilepas');
        this.showToast(`Partisipasi kelas ${label}.`);
        await this.loadEvents();
        await this.bukaDetail(ev);
      } catch (e) {
        this.showToast(e.message || 'Gagal menyimpan keputusan partisipasi.');
      }
    },

    async cariKandidatRuang(startsAt, endsAt) {
      if (!startsAt || !endsAt) return;
      this.roomCandsLoading = true;
      try {
        const data = await API.getRoomAvailability(startsAt, endsAt);
        this.roomCands = Array.isArray(data) ? data : (data && data.candidates) ? data.candidates : [];
      } catch (e) {
        this.roomCands = [];
      } finally {
        this.roomCandsLoading = false;
      }
    },

    bukaKonfirmasiTU(room) {
      this.tuConfirm.room = room;
      this.tuConfirm.form = { status: 'CONFIRMED', name: '', note: '' };
      this.tuConfirm.error = '';
      this.tuConfirm.open = true;
    },

    tutupKonfirmasiTU() {
      this.tuConfirm = { open: false, room: null, form: { status: 'CONFIRMED', name: '', note: '' }, saving: false, error: '' };
    },

    async simpanKonfirmasiTU() {
      const tc = this.tuConfirm;
      if (!this.draftEvent || !this.draftEvent.id || !tc.room) {
        this.showToast('Data draf atau ruangan tidak tersedia.'); return;
      }
      if (!((tc.form.name || '').trim())) {
        this.showToast('Nama petugas TU wajib diisi.'); return;
      }
      tc.error = '';
      tc.saving = true;
      try {
        await API.confirmTeachingEventRoom(this.draftEvent.id, {
          room_id: Number(tc.room.id),
          confirmation_status: tc.form.status,
          external_contact: tc.form.name.trim(),
          note: tc.form.note || undefined
        });
        this.showToast(tc.form.status === 'CONFIRMED' ? 'Konfirmasi TU disetujui.' : 'Konfirmasi TU ditolak.');
        this.tutupKonfirmasiTU();
        await this.muatPratinjau();
        const current = (this.eventsList || []).find(e => String(e.id) === String(this.draftEvent.id));
        if (current) await this.bukaDetail(current);
      } catch (e) {
        tc.error = e.message || 'Gagal menyimpan konfirmasi TU.';
      } finally {
        tc.saving = false;
      }
    },

    async initKM() {
      try { this.sidebarCollapsed = localStorage.getItem('asterisk:sidebar:collapsed') === '1'; } catch (e) {}
      window.addEventListener('offline', () => { this.showPageError('offline'); });
      window.addEventListener('online', () => { if (this.pageState && this.pageState.status === 'offline') window.location.reload(); });
      const token = API.getAuthToken ? API.getAuthToken() : localStorage.getItem('access_token');
      if (!token) {
        window.location.replace('/login.html?role=km');
        return;
      }
      try {
        const me = await API.getMe();
        if (!me || !me.user) {
          localStorage.removeItem('access_token');
          window.location.replace('/login.html?role=km');
          return;
        }
        const role = me.active_assignment && me.active_assignment.role;
        if (role === 'PJ') {
          window.location.replace('/pj.html');
          return;
        }
        if (role && role !== 'KM' && role !== 'SYSTEM_ADMIN') {
          window.location.replace('/login.html?role=km');
          return;
        }
        this.currentUser = me.user;
        this.activeRole = role || null;
        this.meCache = me;
        this.contextAssignments = Array.isArray(me.assignments) ? me.assignments : [];
        this.classSlug = this.resolveClassSlug(me);
        this.selectedClass = this.classSlug;
      } catch (e) {
        localStorage.removeItem('access_token');
        window.location.replace('/login.html?role=km');
        return;
      }

      await AsteriskShell.mount('km', ['dashboard', 'tugas', 'jadwal', 'rooms', 'materi', 'semester', 'anggota', 'antrean', 'monitor', 'notif', 'pengaturan', 'usulan']);
      await this.loadPartials([
        ['km-dashboard', '/partials/km/view-dashboard.html'],
        ['km-tugas', '/partials/km/view-tugas.html'],
        ['km-jadwal', '/partials/km/view-jadwal.html'],
        ['km-rooms', '/partials/common/view-rooms.html'],
        ['km-materi', '/partials/km/view-materi.html'],
        ['km-semester', '/partials/km/view-semester.html'],
        ['km-anggota', '/partials/km/view-anggota.html'],
        ['km-antrean', '/partials/km/view-antrean.html'],
        ['km-monitor', '/partials/km/view-monitor.html'],
        ['km-notif', '/partials/km/view-notif.html'],
        ['km-pengaturan', '/partials/km/view-pengaturan.html'],
        ['km-usulan', '/partials/km/view-usulan.html'],
      ]);
      this.updateClock();
      setInterval(() => this.updateClock(), 1000);
      await this.checkBot();
      await this.loadClasses();
      await this.loadOfferings();
      await this.loadSchedule();
      await this.loadTasks();
      await this.loadPatterns();
      await this.loadEvents();
      await this.loadSemesters().catch(() => {});
      await this.loadMateri().catch(() => {});
      await this.loadNotifikasi().catch(() => {});
      await this.loadAuditLog().catch(() => {});
      this.auditPreviewList = (this.auditList || []).slice(0, 5);
      await this.loadUndanganKM().catch(() => {});
      await this.loadPengaturanKelas().catch(() => {});
      setInterval(() => this.checkBot(), 30000);
      this.dashboardLoading = false;
    },

    async loadPartials(slots) {
      // Alpine v3 auto-init node baru via MutationObserver.
      // Jangan panggil Alpine.initTree manual di sini: menyebabkan x-for ter-render 2x.
      await Promise.all(slots.map(async ([id, url]) => {
        try {
          const res = await fetch(url + '?v=' + encodeURIComponent(AsteriskShell.version), { cache: 'no-store' });
          if (!res.ok) throw new Error(`HTTP ${res.status}`);
          const el = document.getElementById(id);
          if (el) {
            el.innerHTML = await res.text();
          }
        } catch (err) {
          console.error(`Gagal memuat ${url}:`, err);
        }
      }));
    },

    updateClock() {
      try {
        const fmt = new Intl.DateTimeFormat('id-ID', {
          timeZone: 'Asia/Jakarta', weekday: 'long', day: 'numeric',
          month: 'long', year: 'numeric', hour: '2-digit',
          minute: '2-digit', second: '2-digit', hour12: false
        });
        const parts = fmt.formatToParts(new Date());
        const get = (t) => { const p = parts.find(x => x.type === t); return p ? p.value : ''; };
        let day = get('weekday') || 'Senin';
        day = day.charAt(0).toUpperCase() + day.slice(1);
        this.todayName = day;
        if (!this.activeScheduleDay) {
          this.activeScheduleDay = ['Sabtu', 'Minggu'].includes(day) ? 'Senin' : day;
        }
        this.todayFull = `${day}, ${get('day')} ${get('month')} ${get('year')}`;
        this.currentTime = `${get('hour')}:${get('minute')}:${get('second')} WIB`;
      } catch (e) {
        this.todayName = 'Senin';
        if (!this.activeScheduleDay) this.activeScheduleDay = 'Senin';
        this.todayFull = 'Senin';
      }
    },

    bukaAntreanReview() {
      this.go('tugas');
      this.tugasTab = 'review';
    },

    go(v) {
      this.pageState = null;
      if (v === 'tugas-tambah') {
        this.mulaiTambahTugas();
        return;
      }
      if (!this.knownViews.includes(v)) { this.showPageError('404'); return; }
      this.view = v;
      this.drawer = false;
      if (v === 'tugas') this.tugasSub = 'list';
      if (v === 'anggota') { this.anggotaSub = 'list'; this.loadPenugasan(); this.loadUndanganKM(); }
      if (v === 'semester') this.loadSemesters();
      if (v === 'materi') this.loadMateri();
      if (v === 'ruangan') this.loadRoomHistory();
      if (v === 'notifikasi' || v === 'monitoring') this.loadNotifikasi();
      if (v === 'log') this.loadAuditLog();
      if (v === 'pengaturan') { this.loadKanalSaya(); this.loadPengaturanKelas(); this.loadBackupSaya(); }
      if (v === 'usulan') { this.loadUsulanTarget(); this.loadUsulanSaya(); }
      window.scrollTo({ top: 0 });
    },

    soon(fitur) { this.showToast(`${fitur}: fitur belum tersedia.`); },

    async loadRoomCandidates() {
      const f = this.roomSearch;
      if (!f.date || !f.start || !f.end || f.end <= f.start) { this.roomCandidatesError = 'Isi tanggal dan interval waktu yang valid.'; return; }
      this.roomCandidatesLoading = true; this.roomCandidatesError = ''; this.roomCandidates = null;
      try { this.roomCandidates = await API.getRoomAvailability(f.date + 'T' + f.start + ':00+07:00', f.date + 'T' + f.end + ':00+07:00');
        if (!this.roomCandidates) throw new Error('Kandidat ruangan gagal dimuat.'); }
      catch (e) { this.roomCandidatesError = e.message || 'Kandidat ruangan gagal dimuat.'; }
      finally { this.roomCandidatesLoading = false; }
    },

    async loadRoomHistory() {
      this.roomHistoryLoading = true; this.roomHistoryError = '';
      try { this.roomHistory = await API.getRoomConfirmations(); }
      catch (e) { this.roomHistory = []; this.roomHistoryError = e.message || 'Riwayat konfirmasi gagal dimuat.'; }
      finally { this.roomHistoryLoading = false; }
    },

    async checkBot() {
      try {
        const st = await API.getStatus();
        if (st) this.botOnline = String(st.bot_connection || '').toLowerCase() === 'connected';
      } catch (e) { this.botOnline = false; }
    },

    async loadClasses() {
      const scopedClass = this.meCache ? this.resolveClassSlug(this.meCache) : '';
      this.selectedClass = scopedClass || '';
      try {
        const data = await API.getClasses();
        const classes = (data && data.classes) || [];
        this.classList = classes.filter(c => {
          const slug = typeof c === 'string' ? c : c.slug;
          return !scopedClass || slug === scopedClass;
        });
      } catch (e) { this.classList = scopedClass ? [scopedClass] : []; }
      this.classSlug = scopedClass;
    },

    async loadSchedule() {
      try {
        const [patterns, events] = await Promise.all([API.getPatterns(), API.getTeachingEvents()]);
        const seen = new Set();
        const list = [];
        (patterns || []).forEach((s, i) => {
          const entry = {
            id: `pattern-${s.id || i}`, hari: this.polaHariName(s.day_of_week),
            jam: `${String(s.start_time || '').slice(0, 5)} - ${String(s.end_time || '').slice(0, 5)}`,
            matkul: s.display_name || s.offering || s.course_name || 'Mata Kuliah',
            dosen: s.lecturer || s.dosen || '', ruang: s.room || s.room_code || '',
            timeStart: String(s.start_time || '').slice(0, 5), timeEnd: String(s.end_time || '').slice(0, 5)
          };
          const key = `${entry.hari}|${entry.timeStart}|${entry.matkul}|${entry.ruang || ''}`;
          if (seen.has(key)) return;
          seen.add(key);
          list.push(entry);
        });
        (events || []).filter(e => String(e.lifecycle_status || '').toUpperCase() === 'PUBLISHED').forEach((e, i) => {
          const start = new Date(e.starts_at), end = new Date(e.ends_at);
          if (Number.isNaN(start.getTime())) return;
          const hari = start.toLocaleDateString('id-ID', { weekday: 'long', timeZone: 'Asia/Jakarta' });
          const hm = d => d.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit', hour12: false, timeZone: 'Asia/Jakarta' });
          list.push({ id: `event-${e.id || i}`, hari, jam: `${hm(start)} - ${hm(end)}`, matkul: e.offering || 'Mata Kuliah', dosen: '', ruang: e.room || '', timeStart: hm(start), timeEnd: hm(end), eventKind: e.event_kind });
        });
        this.fullSchedule = list;
      } catch (e) { this.fullSchedule = []; }
    },

    async loadTasks() {
      this.tugasLoading = true; this.tugasListError = '';
      try {
        const raw = await API.getAllTasks('');
        this.tasks = (raw || []).map(t => {
          const u = this.deadlineBadge(t.deadline_at, t.completed_at);
          return { id: t.id, matkul: t.matkul, deskripsi: t.deskripsi, title: t.title,
                   instructions: t.instructions, deadline: this.fmtDeadlineID(t.deadline_at),
                   deadline_at: t.deadline_at, task_type: t.task_type,
                   submission_text: t.submission_text, submission_url: t.submission_url,
                   publication_status: t.publication_status, review_state: t.review_state,
                   version: t.version, completed_at: t.completed_at, archived_at: t.archived_at,
                   offering_id: t.offering_id,
                   urgency: u.level, countdown: u.badge };
        });
      } catch (e) {
        this.tasks = [];
        this.tugasListError = 'Tugas belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.tugasLoading = false;
      }
    },

    async loadOfferings() {
      this.offeringLoading = true;
      this.offeringState = 'loading';
      try {
        if (!this.selectedClass) { this.offeringState = 'no-class'; this.offeringList = []; return; }
        const slug = this.classSlug || this.selectedClass;
        const hasil = await API.getSemestersResult(slug).catch(() => ({ ok: false, status: 0, data: [] }));
        const list = Array.isArray(hasil.data) ? hasil.data : [];
        if (!hasil.ok && hasil.status === 404) {
          this.offeringState = 'no-v1-class'; this.offeringList = []; this.semesterId = ''; this.semesterStatus = ''; return;
        }
        if (!hasil.ok) {
          this.offeringState = 'error'; this.offeringList = []; this.semesterId = ''; this.semesterStatus = ''; return;
        }
        const active = list.find(s => s.status === 'ACTIVE') || list.find(s => String(s.status || '').toUpperCase() === 'DRAFT') || list[0];
        if (!active) {
          this.offeringState = 'empty-semester'; this.offeringList = []; this.semesterId = ''; this.semesterStatus = ''; return;
        }
        this.semesterOptions = list;
        this.semesterId = String(active.id);
        this.semesterStatus = String(active.status || '').toUpperCase();
        const offerings = await API.getSemesterOfferings(active.id).catch(() => null);
        if (offerings === null) {
          this.offeringState = 'error'; this.offeringList = []; return;
        }
        this.offeringList = Array.isArray(offerings) ? offerings : [];
        this.offeringState = this.offeringList.length > 0 ? 'ok' : 'empty-offering';
      } catch (e) {
        this.offeringList = [];
        this.offeringState = 'error';
      } finally {
        this.offeringLoading = false;
      }
    },

    pesanOffering() {
      if (this.offeringState === 'no-v1-class') return 'Kelas ' + (this.selectedClass || 'ini') + ' belum terdaftar di database (kode: NOT_FOUND). Minta Administrator membuat kelas tersebut, lalu siapkan semester dan mata kuliah.';
      if (this.offeringState === 'empty-semester') return 'Belum ada semester untuk kelas ' + (this.selectedClass || 'ini') + '. Minta Administrator menyiapkan semester dan mata kuliah di database.';
      if (this.offeringState === 'empty-offering') return 'Semester aktif belum memiliki mata kuliah. Tambahkan mata kuliah ke semester dahulu.';
      if (this.offeringState === 'error') return 'Daftar mata kuliah gagal dimuat. Periksa koneksi lalu coba lagi.';
      if (this.offeringState === 'no-class') return 'Kelas belum termuat. Muat ulang halaman.';
      return '';
    },

    offeringDisplay(id) {
      const found = (this.offeringList || []).find(o => String(o.id) === String(id));
      return found ? (found.display_name || found.course_code || '') : '';
    },

    parseDeadlineID(tanggal, jam) { return API.parseDeadlineID(tanggal, jam); },

    previewMatkulTag() {
      const o = (this.offeringList || []).find(x => String(x.id) === String(this.tugasForm.offeringId));
      if (o) {
        if (o.course_name) return String(o.course_name).toUpperCase();
        if (o.course_code) return String(o.course_code).toUpperCase();
        if (o.display_name) return String(o.display_name.split('(')[0] || o.display_name).trim().toUpperCase();
      }
      return 'MATA KULIAH';
    },

    previewDeadlineFormatted() {
      if (!this.tugasForm.tanggal || !this.tugasForm.jam) return 'Pilih tanggal & jam';
      const iso = this.parseDeadlineID(this.tugasForm.tanggal, this.tugasForm.jam);
      if (!iso) return `${this.tugasForm.tanggal} · ${this.tugasForm.jam} WIB`;
      try {
        const d = new Date(iso);
        if (isNaN(d.getTime())) return `${this.tugasForm.tanggal} · ${this.tugasForm.jam} WIB`;
        const hari = d.toLocaleString('id-ID', { timeZone: 'Asia/Jakarta', weekday: 'long' });
        const tgl = d.toLocaleString('id-ID', { timeZone: 'Asia/Jakarta', day: 'numeric' });
        const bln = d.toLocaleString('id-ID', { timeZone: 'Asia/Jakarta', month: 'short' });
        const jam = String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0');
        return `${hari}, ${tgl} ${bln} · ${jam} WIB`;
      } catch (e) {
        return `${this.tugasForm.tanggal} · ${this.tugasForm.jam} WIB`;
      }
    },

    fmtDeadlineLongID(iso) {
      if (!iso) return '-';
      try {
        const d = new Date(iso);
        if (isNaN(d.getTime())) return String(iso);
        const hari = d.toLocaleString('id-ID', { timeZone: 'Asia/Jakarta', weekday: 'long' });
        const tgl = d.toLocaleString('id-ID', { timeZone: 'Asia/Jakarta', day: 'numeric' });
        const bln = d.toLocaleString('id-ID', { timeZone: 'Asia/Jakarta', month: 'long' });
        const thn = d.toLocaleString('id-ID', { timeZone: 'Asia/Jakarta', year: 'numeric' });
        const jam = String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0');
        return `${hari}, ${tgl} ${bln} ${thn} · ${jam} WIB`;
      } catch (e) {
        return String(iso);
      }
    },

    previewCountdown(customIso) {
      const iso = customIso || (this.tugasForm.tanggal && this.tugasForm.jam ? this.parseDeadlineID(this.tugasForm.tanggal, this.tugasForm.jam) : '');
      if (!iso) return { text: 'Belum diatur', bg: 'bg-slate-100', color: 'text-slate-600' };
      try {
        const d = new Date(iso);
        const now = new Date();
        const diffMs = d.getTime() - now.getTime();
        const diffDays = Math.ceil(diffMs / (1000 * 60 * 60 * 24));
        if (diffDays < 0) {
          return { text: 'Sudah lewat', bg: 'bg-[#FEE2E2]', color: 'text-[#DC2626]' };
        }
        if (diffDays === 0) {
          return { text: 'Hari ini', bg: 'bg-[#FEF3C7]', color: 'text-[#D97706]' };
        }
        if (diffDays === 1) {
          return { text: 'Besok', bg: 'bg-[#FEF3C7]', color: 'text-[#D97706]' };
        }
        if (diffDays <= 3) {
          return { text: `Sisa ${diffDays} hari`, bg: 'bg-[#FEF3C7]', color: 'text-[#D97706]' };
        }
        return { text: `Sisa ${diffDays} hari`, bg: 'bg-[#DCFCE7]', color: 'text-[#15803D]' };
      } catch (e) {
        return { text: 'Aktif', bg: 'bg-[#DCFCE7]', color: 'text-[#15803D]' };
      }
    },

    urgencyOf(label) {
      try {
        const s = String(label || '');
        const hm = s.match(/(\d{1,2})[:.](\d{2})/);
        if (!hm) return { level: 'aman', badge: 'Aktif' };
        const now = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
        const target = new Date(now);
        target.setHours(parseInt(hm[1], 10), parseInt(hm[2], 10), 0, 0);
        if (/besok/i.test(s)) target.setDate(target.getDate() + 1);
        else if (!/hari ini/i.test(s)) {
          const dm = s.match(/(\d{1,2})\s+(Jan|Feb|Mar|Apr|Mei|Jun|Jul|Agu|Sep|Okt|Nov|Des)/i);
          if (dm) {
            const months = { jan: 0, feb: 1, mar: 2, apr: 3, mei: 4, jun: 5, jul: 6, agu: 7, sep: 8, okt: 9, nov: 10, des: 11 };
            target.setMonth(months[dm[2].slice(0, 3).toLowerCase()]);
            target.setDate(parseInt(dm[1], 10));
            if (target < new Date(now.getTime() - 86400000)) target.setFullYear(target.getFullYear() + 1);
          }
        }
        const diffH = (target - now) / 3600000;
        if (diffH < 0) return { level: 'mendesak', badge: 'Terlewat' };
        if (diffH < 24) return { level: 'mendesak', badge: 'Besok' };
        if (diffH <= 72) return { level: 'mendekati', badge: `H-${Math.ceil(diffH / 24)}` };
        return { level: 'aman', badge: 'Aktif' };
      } catch (e) {
        return { level: 'aman', badge: 'Aktif' };
      }
    },

    // Tugas — tambah + tinjau + terbit + ubah + selesai + arsip via API asli.
    mulaiTambahTugas() {
      this.tugasForm = { offeringId: '', judul: '', tanggal: '', jam: '', deskripsi: '', kumpul: '', kumpulUrl: '', jenis: 'Individu' };
      this.tugasError = '';
      this.editTugasId = ''; this.editTugasVersion = 0; this.tugasConflict = null;
      this.tugasSub = 'tambah';
      this.view = 'tugas';
      window.scrollTo({ top: 0 });
    },

    validasiTugasTerbit() {
      const f = this.tugasForm;
      const offeringId = f.offeringId || '';
      if (!offeringId) return 'Pilih mata kuliah di kelas ini dulu.';
      if (!f.judul || f.judul.trim().length < 5) return 'Judul tugas minimal 5 karakter.';
      if (!f.tanggal || !f.jam) return 'Tanggal dan jam deadline wajib diisi.';
      if (!f.deskripsi || f.deskripsi.trim().length < 5) return 'Instruksi tugas minimal 5 karakter.';
      if (!((f.kumpul || '').trim() || (f.kumpulUrl || '').trim())) return 'Isi tempat pengumpulan (keterangan atau tautan).';
      if (!this.parseDeadlineID(f.tanggal, f.jam)) return 'Format tanggal atau jam tidak dikenali. Pakai YYYY-MM-DD dan HH:MM.';
      return '';
    },

    rakitTugasPayload(saveAs) {
      const f = this.tugasForm;
      const payload = {
        offering_id: Number(f.offeringId),
        title: f.judul.trim(),
        instructions: f.deskripsi.trim(),
        save_as: saveAs
      };
      const deadlineAt = (f.tanggal && f.jam) ? this.parseDeadlineID(f.tanggal, f.jam) : '';
      if (deadlineAt) payload.deadline_at = deadlineAt;
      if (f.jenis) payload.task_type = f.jenis;
      if ((f.kumpul || '').trim()) payload.submission_text = f.kumpul.trim();
      if ((f.kumpulUrl || '').trim()) payload.submission_url = f.kumpulUrl.trim();
      return payload;
    },

    async terbitTugas() {
      if (!this.tugasPreview) { this.tinjauTugas(); return; }
      this.tugasError = '';
      this.tugasPublishing = true;
      try {
        const res = await API.createTask(this.rakitTugasPayload('published'));
        const newId = res && res.data && res.data.id;
        await this.loadTasks();
        this.tugasPreview = null;
        this.tugasSub = 'list';
        if (newId) {
          await this.bukaDetailTugas(newId);
          this.tugasPublishMsg = 'Tugas diterbitkan.';
        } else {
          this.view = 'tugas';
          this.showToast('Tugas diterbitkan.');
        }
      } catch (err) {
        this.tugasError = err.message || 'Gagal menerbitkan tugas di server.';
        this.tugasSub = 'tinjau';
      } finally {
        this.tugasPublishing = false;
      }
    },

    async simpanDraf() {
      const f = this.tugasForm || {};
      if (!f.offeringId) { this.showToast('Pilih mata kuliah di kelas ini dulu.'); return; }
      if (!f.judul || f.judul.trim().length < 5) { this.showToast('Judul tugas minimal 5 karakter.'); return; }
      try {
        await API.createTask(this.rakitTugasPayload('draft'));
        await this.loadTasks();
        this.tugasTab = 'draf';
        this.tugasSub = 'list';
        this.showToast('Draf tersimpan di server.');
        this.view = 'tugas';
      } catch (err) {
        this.showToast(err.message || 'Gagal menyimpan draf di server.');
      }
    },

    tinjauTugas() {
      const err = this.validasiTugasTerbit();
      if (err) { this.tugasError = err; window.scrollTo({ top: 0 }); return; }
      this.tugasError = '';
      const f = this.tugasForm;
      const o = (this.offeringList || []).find(x => String(x.id) === String(f.offeringId));
      let dosen = 'Tim Dosen';
      if (o) {
        if (Array.isArray(o.lecturers) && o.lecturers.length > 0) {
          dosen = o.lecturers.join(', ');
        } else if (o.display_name && o.display_name.includes('·')) {
          dosen = o.display_name.split('·')[1].trim();
        } else if (o.display_name && o.display_name.includes('-')) {
          dosen = o.display_name.split('-')[1].trim();
        }
      }
      this.tugasPreview = {
        offeringId: f.offeringId,
        matkul: this.offeringDisplay(f.offeringId),
        dosen: dosen,
        judul: f.judul.trim(),
        deadline_at: this.parseDeadlineID(f.tanggal, f.jam),
        deskripsi: f.deskripsi.trim(),
        jenis: f.jenis || '',
        kumpul: (f.kumpul || '').trim(),
        kumpulUrl: (f.kumpulUrl || '').trim()
      };
      this.tugasSub = 'tinjau';
      window.scrollTo({ top: 0 });
    },

    async bukaDetailTugas(id) {
      this.tugasDetailLoading = true;
      this.tugasDetail = null; this.tugasReviews = [];
      this.tugasDetailTab = 'detail'; this.tugasConflict = null;
      this.tugasPublishMsg = '';
      this.tugasSub = 'detail';
      this.view = 'tugas';
      window.scrollTo({ top: 0 });
      try {
        const d = await API.getTaskDetail(id);
        const info = (d && (d.task || d)) || null;
        if (!info) { this.showToast('Tugas tidak ditemukan.'); this.tugasSub = 'list'; return; }
        const u = this.deadlineBadge(info.deadline_at, info.completed_at || info.is_completed);
        this.tugasDetail = {
          id: info.id, matkul: info.offering || info.matkul || '', title: info.title || '',
          deskripsi: info.instructions || info.deskripsi || '', instructions: info.instructions || '',
          deadline_at: info.deadline_at, deadline: this.fmtDeadlineID(info.deadline_at),
          task_type: info.task_type || '', submission_text: info.submission_text || '',
          submission_url: info.submission_url || '',
          publication_status: info.publication_status || info.status || 'DRAFT',
          review_state: info.review_state || info.review_status || 'NOT_REVIEWED',
          version: info.version || 0,
          completed_at: info.completed_at || (info.is_completed ? true : null),
          archived_at: info.archived_at || (info.is_archived ? true : null),
          offering_id: info.offering_id || info.course_offering_id || '',
          urgency: u.level, countdown: u.badge
        };
        const revs = (d && d.reviews) || [];
        this.tugasReviews = revs.map(r => ({
          reviewer: r.reviewer || 'KM', decision: r.decision || '',
          note: r.note || '', version: r.task_version || r.version || '',
          waktu: r.created_at || ''
        }));
      } catch (e) {
        this.showToast('Gagal memuat detail tugas dari server.');
        this.tugasSub = 'list';
      } finally {
        this.tugasDetailLoading = false;
      }
    },

    mulaiUbahTugas() {
      const d = this.tugasDetail;
      if (!d) return;
      const dl = d.deadline_at ? new Date(d.deadline_at) : null;
      const pad = (n) => String(n).padStart(2, '0');
      let tgl = '', jam = '';
      if (dl && !isNaN(dl)) {
        tgl = `${dl.getFullYear()}-${pad(dl.getMonth() + 1)}-${pad(dl.getDate())}`;
        jam = `${pad(dl.getHours())}:${pad(dl.getMinutes())}`;
      }
      this.tugasForm = { offeringId: String(d.offering_id || ''), judul: d.title || '', tanggal: tgl, jam: jam,
        deskripsi: d.instructions || d.deskripsi || '', kumpul: d.submission_text || '',
        kumpulUrl: d.submission_url || '', jenis: d.task_type || 'Individu' };
      this.tugasError = '';
      this.editTugasId = d.id; this.editTugasVersion = d.version || 0;
      this.tugasConflict = null;
      this.tugasSub = 'ubah';
      window.scrollTo({ top: 0 });
    },

    async simpanUbahTugas() {
      if (!this.editTugasId) return;
      const err = this.validasiTugasTerbit();
      if (err) { this.tugasError = err; window.scrollTo({ top: 0 }); return; }
      this.tugasError = '';
      try {
        const payload = this.rakitTugasPayload(undefined);
        delete payload.offering_id;
        delete payload.save_as;
        payload.version = Number(this.editTugasVersion) || 0;
        await API.updateTask(this.editTugasId, payload);
        await this.loadTasks();
        this.showToast('Perubahan tersimpan sebagai versi baru.');
        await this.bukaDetailTugas(this.editTugasId);
      } catch (e) {
        if (e.code === 'VERSION_CONFLICT') {
          this.tugasConflict = (e.payload && (e.payload.current_data || e.payload.data)) || null;
        } else {
          this.tugasError = e.message || 'Gagal menyimpan perubahan.';
        }
      }
    },

    async muatUlangVersiTugas() {
      if (!this.editTugasId) return;
      try {
        const d = await API.getTaskDetail(this.editTugasId);
        const info = (d && (d.task || d)) || null;
        if (info) {
          this.editTugasVersion = info.version || 0;
          this.tugasConflict = null;
          this.showToast('Versi terbaru dimuat. Periksa sebelum menyimpan ulang.');
        }
      } catch (e) {
        this.showToast('Gagal memuat versi terbaru.');
      }
    },

    async tandaiSelesai(id, version) {
      try {
        await API.completeTask(id, version);
        await this.loadTasks();
        this.showToast('Tugas ditandai selesai.');
        if (this.tugasSub === 'detail') await this.bukaDetailTugas(id);
      } catch (e) {
        this.showToast(e.message || 'Gagal menandai selesai.');
        await this.loadTasks();
      }
    },

    async arsipkanTugas(id, version) {
      try {
        await API.deleteTask(id, version);
        await this.loadTasks();
        this.showToast('Tugas diarsipkan.');
        if (this.tugasSub === 'detail') { this.tugasSub = 'list'; }
      } catch (e) {
        this.showToast(e.message || 'Gagal mengarsipkan tugas.');
        await this.loadTasks();
      }
    },

    hapusTugas(id) {
      const t = (this.tasks || []).find(x => String(x.id) === String(id));
      this.arsipkanTugas(id, t && t.version);
    },

    async pulihkanTugas(id, version) {
      try {
        await API.restoreTask(id, version);
        await this.loadTasks();
        this.showToast('Tugas dipulihkan dari arsip.');
      } catch (e) {
        this.showToast(e.message || 'Gagal memulihkan tugas.');
        await this.loadTasks();
      }
    },

    // Undang PJ — via POST /api/v1/invitations (butuh semester_id + offering_id).
    async buatUndangPJ() {
      const offeringId = (this.undang && this.undang.offeringId) || '';
      if (!offeringId) { this.undangError = 'Mata kuliah di kelas ini wajib dipilih.'; return; }
      if (!this.undang.nomor || this.undang.nomor.replace(/\D/g, '').length < 9) {
        this.undangError = 'Nomor WhatsApp calon PJ tidak valid.';
        return;
      }
      if (!this.semesterId) { this.undangError = 'Semester aktif tidak ditemukan untuk kelas ini.'; return; }
      this.undangError = '';
      try {
        const res = await API.createInvitation({
          role: 'PJ',
          class_slug: this.classSlug || this.selectedClass,
          semester_id: Number(this.semesterId),
          offering_id: Number(offeringId),
          invited_identity_key: this.undang.nomor.trim()
        });
        const token = (res && res.token) ? res.token : '';
        this.undangLink = `${window.location.origin}/invite.html?token=${encodeURIComponent(token)}`;
        this.anggotaSub = 'siap';
      } catch (err) {
        this.undangError = err.message || 'Gagal membuat undangan PJ.';
      }
    },

    // Pemeriksaan KM — via POST /api/v1/tasks/{id}/reviews (terikat versi).
    reviewConflict: null,

    async setujuiTugas(id) {
      this.reviewConflict = null;
      try {
        const detail = await API.getTaskDetail(id).catch(() => null);
        const version = detail && (detail.version || (detail.task && detail.task.version)) || 0;
        await API.reviewTask(id, { decision: 'APPROVED', task_version: version });
        await this.loadTasks();
        this.showToast('Tugas disetujui.');
      } catch (err) {
        if (err.code === 'VERSION_CONFLICT') {
          this.reviewConflict = { id: id };
          this.showToast('Versi tugas berubah. Muat versi terbaru sebelum memutuskan.');
          await this.loadTasks();
        } else {
          this.showToast(err.message || 'Gagal menyimpan persetujuan.');
        }
      }
    },
    async kirimReview(id) {
      const targetId = id || this.reviewId;
      if (!targetId) return;
      const decision = this.reviewMode === 'batal' ? 'REVOKED' : 'CHANGES_REQUESTED';
      if (!this.reviewNote.trim()) { this.showToast('Catatan wajib untuk koreksi/pembatalan.'); return; }
      this.reviewConflict = null;
      try {
        const detail = await API.getTaskDetail(targetId).catch(() => null);
        const version = detail && (detail.version || (detail.task && detail.task.version)) || 0;
        await API.reviewTask(targetId, { decision: decision, note: this.reviewNote.trim(), task_version: version });
        this.reviewNote = '';
        this.reviewId = null;
        await this.loadTasks();
        this.showToast('Keputusan pemeriksaan tersimpan di server.');
      } catch (err) {
        if (err.code === 'VERSION_CONFLICT') {
          this.reviewConflict = { id: targetId };
          this.showToast('Versi tugas berubah. Muat versi terbaru sebelum memutuskan.');
          await this.loadTasks();
        } else {
          this.showToast(err.message || 'Gagal menyimpan keputusan pemeriksaan.');
        }
      }
    },

    async reviewAksi(id, mode) {
      const isBatal = mode === 'batal';
      const promptMsg = isBatal
        ? 'Alasan penolakan / pembatalan tugas (wajib):'
        : 'Catatan koreksi untuk PJ mata kuliah (wajib):';
      const note = window.prompt(promptMsg);
      if (note === null) return;
      if (!note.trim()) {
        this.showToast('Catatan wajib diisi untuk ' + (isBatal ? 'penolakan tugas.' : 'minta koreksi.'));
        return;
      }
      this.reviewMode = mode;
      this.reviewNote = note.trim();
      await this.kirimReview(id);
    },

    async muatUlangAntrean() {
      this.reviewConflict = null;
      this.reviewId = null;
      await this.loadTasks();
      this.showToast('Versi terbaru dimuat.');
    },

    // Keputusan dari layar detail (menyebut versi yang ditinjau).
    async putuskanDetail(decision) {
      const d = this.tugasDetail;
      if (!d) return;
      if ((decision === 'CHANGES_REQUESTED' || decision === 'REVOKED') && !(this.reviewNote || '').trim()) {
        this.showToast('Catatan wajib untuk koreksi/pembatalan.'); return;
      }
      try {
        await API.reviewTask(d.id, { decision: decision, note: (this.reviewNote || '').trim() || undefined, task_version: d.version || 0 });
        this.reviewNote = '';
        await this.loadTasks();
        this.showToast(decision === 'APPROVED' ? 'Tugas disetujui untuk versi ini.' : 'Keputusan pemeriksaan tersimpan di server.');
        await this.bukaDetailTugas(d.id);
      } catch (e) {
        if (e.code === 'VERSION_CONFLICT') {
          this.tugasConflict = { id: d.id };
          this.showToast('Versi tugas berubah. Muat versi terbaru sebelum memutuskan.');
          await this.loadTasks();
        } else {
          this.showToast(e.message || 'Gagal menyimpan keputusan.');
        }
      }
    },

    auditList: [], auditLoading: false, auditError: '',
    auditFilter: { action: '', entity_type: '', actor: '', since: '', until: '' },
    auditDetailId: null, auditHasMore: false, auditLoadingMore: false,
    auditPreviewList: [],

    fmtWaktuID(iso) { return API.fmtWaktuID(iso); },

    labelAksiAudit(action) {
      const a = String(action || '').toUpperCase();
      const map = {
        UPDATE_PORTAL_MODE: 'Mode portal diperbarui',
        UPDATE_CLASS_SETTINGS: 'Pengaturan kelas diperbarui',
        ROTATE_PORTAL_CODE: 'Kode portal diputar',
        CREATE_TASK: 'Tugas dibuat',
        UPDATE_TASK: 'Tugas diperbarui',
        REVIEW_TASK: 'Tugas diperiksa',
        COMPLETE_TASK: 'Tugas diselesaikan',
        ARCHIVE_TASK: 'Tugas diarsipkan',
        RESTORE_TASK: 'Tugas dipulihkan',
        CREATE_PATTERN: 'Jadwal ditambahkan',
        UPDATE_PATTERN: 'Jadwal diperbarui',
        DELETE_PATTERN: 'Jadwal dihapus',
        CREATE_TEACHING_EVENT: 'Perubahan jadwal dibuat',
        PUBLISH_EVENT: 'Perubahan diterbitkan',
        REVOKE_EVENT: 'Publikasi dicabut',
        CREATE_BACKUP: 'Cadangan data dibuat',
        ASSIGN_ROLE: 'Peran ditetapkan',
        SUSPEND_ASSIGNMENT: 'Peran ditangguhkan',
        REVOKE_INVITATION: 'Undangan dibatalkan',
        LINK_CHANNEL: 'Kanal WhatsApp terhubung',
        REVOKE_CHANNEL: 'Kanal WhatsApp dilepas',
        BOT_TEST_MESSAGE: 'Uji kirim pesan bot'
      };
      if (map[a]) return map[a];
      if (!action) return '-';
      return String(action).replace(/_/g, ' ').toLowerCase().replace(/^\w/, c => c.toUpperCase());
    },

    async loadAuditLog(more) {
      const isMore = !!more;
      if (isMore) {
        this.auditLoadingMore = true;
      } else {
        this.auditLoading = true; this.auditError = ''; this.auditDetailId = null;
      }
      try {
        const params = this.rakitAuditParams(isMore ? this.auditList.length : 0);
        const rows = await API.getAudit(params).catch(() => null);
        const list = Array.isArray(rows) ? rows : [];
        this.auditList = isMore ? [...this.auditList, ...list] : list;
        this.auditHasMore = list.length >= 50;
      } catch (e) {
        if (!isMore) {
          this.auditList = [];
          this.auditError = 'Riwayat belum dapat dimuat. Periksa koneksi lalu coba lagi.';
        } else {
          this.showToast('Gagal memuat riwayat berikutnya.');
        }
      } finally {
        this.auditLoading = false; this.auditLoadingMore = false;
      }
    },

    rakitAuditParams(offset) {
      const f = this.auditFilter || {};
      const params = { limit: 50 };
      if ((f.action || '').trim()) params.action = f.action.trim().toUpperCase();
      if ((f.entity_type || '').trim()) params.entity_type = f.entity_type.trim().toUpperCase();
      if ((f.actor || '').trim()) params.actor = f.actor.trim();
      if (f.since) params.since = String(f.since).length === 16 ? f.since + ':00+07:00' : f.since;
      if (f.until) params.until = String(f.until).length === 16 ? f.until + ':00+07:00' : f.until;
      if (offset > 0) params.offset = offset;
      return params;
    },

    auditFilterCount() {
      const f = this.auditFilter || {};
      return ['action', 'entity_type', 'actor', 'since', 'until'].filter(k => (f[k] || '').trim()).length;
    },

    resetAuditFilter() {
      this.auditFilter = { action: '', entity_type: '', actor: '', since: '', until: '' };
    },

    prettyJSON(v) {
      if (v === null || v === undefined || v === '') return '—';
      try {
        const obj = typeof v === 'string' ? JSON.parse(v) : v;
        return JSON.stringify(obj, null, 2);
      } catch (e) {
        return String(v);
      }
    },

    get notifGagal() {
      return (this.notifList || []).filter(n => ['FAILED', 'CANCELLED'].includes(String(n.status || '').toUpperCase())).length;
    },

    get notifMenunggu() {
      return (this.notifList || []).filter(n => ['PENDING', 'PROCESSING'].includes(String(n.status || '').toUpperCase())).length;
    },

    get attentionCount() {
      return this.antrean.length + this.notifGagal + this.undanganMenunggu + this.drafPerubahan.length;
    },

    get undanganMenunggu() {
      return (this.undanganList || []).length;
    },

    get drafPerubahan() {
      return (this.eventsList || []).filter(e => String(e.lifecycle_status || '').toUpperCase() === 'DRAFT');
    },

    get publikasiTerbaru() {
      const acts = ['PUBLISH_EVENT', 'CREATE_TEACHING_EVENT', 'CREATE_TASK'];
      return (this.auditList || []).filter(a => acts.includes(String(a.action || '').toUpperCase())).slice(0, 3);
    },

    get activeSemesterLabel() {
      const s = (this.semesterList || []).find(x => String(x.status || '').toUpperCase() === 'ACTIVE');
      if (s) return `Semester ${s.term || ''} ${s.academic_year || ''}`.trim();
      if (this.semesterId) return 'Semester aktif dimuat';
      return '';
    },

    get auditPreview() {
      const src = (this.auditPreviewList && this.auditPreviewList.length ? this.auditPreviewList : this.auditList) || [];
      return src.slice(0, 5);
    },

    get materiTampil() {
      const f = String(this.materiFilterOffering || '');
      const list = this.materiList || [];
      const filtered = (!f || f === 'semua')
        ? list.slice()
        : (f === 'umum'
          ? list.filter(m => m.offering_id === null || m.offering_id === undefined || String(m.offering_id) === '')
          : list.filter(m => String(m.offering_id || '') === f));
      const terbaru = this.materiSort !== 'terlama';
      return filtered.sort((a, b) => {
        const ta = a.created_at ? new Date(a.created_at).getTime() : 0;
        const tb = b.created_at ? new Date(b.created_at).getTime() : 0;
        return terbaru ? (tb - ta) : (ta - tb);
      });
    },

    get materiCountUmum() {
      return (this.materiList || []).filter(m => m.offering_id === null || m.offering_id === undefined || String(m.offering_id) === '').length;
    },

    materiCountOffering(id) {
      return (this.materiList || []).filter(m => String(m.offering_id || '') === String(id)).length;
    },

    materiOfferingName(m) {
      if (!m || m.offering_id === null || m.offering_id === undefined || String(m.offering_id) === '') return 'Umum kelas';
      return this.offeringDisplay(m.offering_id) || 'Mata kuliah';
    },

    sumberMateri(m) {
      const url = String((m && m.url) || '').trim();
      if (!url) return String((m && m.description) || '').trim() ? 'Catatan' : 'Tautan';
      let host = '';
      try { host = new URL(url).hostname.replace(/^www\./, ''); } catch (e) { host = ''; }
      const path = url.split('?')[0];
      const ext = (path.split('.').pop() || '').toUpperCase();
      if (/^[A-Z0-9]{2,5}$/.test(ext) && !/^(COM|ID|ORG|NET|IO|DEV|APP|ME|LINK|GLYPH|US)$/.test(ext)) return ext;
      if (host) {
        if (host.includes('drive.google')) return 'Tautan Google Drive';
        if (host.includes('docs.google')) return 'Tautan Google Docs';
        return 'Tautan ' + host;
      }
      return 'Tautan';
    },

    fmtTanggalSingkat(iso) {
      try {
        const d = new Date(iso);
        if (!iso || isNaN(d)) return '—';
        const tgl = d.toLocaleDateString('id-ID', { timeZone: 'Asia/Jakarta', day: 'numeric', month: 'short' });
        return tgl.replace('.', '');
      } catch (e) { return '—'; }
    },

    semesterLabel(s) {
      if (!s) return '-';
      return `${s.academic_year || ''} · ${s.term || ''}`.trim();
    },

    semesterStatusLabel(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'DRAFT') return 'Draf';
      if (s === 'ACTIVE') return 'Aktif';
      if (s === 'ARCHIVED') return 'Arsip';
      return st || '-';
    },

    async loadSemesters() {
      const slug = this.classSlug || this.selectedClass;
      if (!slug) { this.semesterList = []; this.semesterError = 'Kelas belum termuat.'; return; }
      this.semesterLoading = true; this.semesterError = '';
      try {
        const hasil = await API.getSemestersResult(slug).catch(() => ({ ok: false, status: 0, data: [] }));
        if (!hasil.ok && hasil.status === 404) {
          this.semesterList = [];
          this.semesterError = 'Kelas belum terdaftar di database. Minta System Admin membuat kelas.';
          return;
        }
        if (!hasil.ok) {
          this.semesterList = [];
          this.semesterError = 'Daftar semester belum dapat dimuat. Periksa koneksi lalu coba lagi.';
          return;
        }
        this.semesterList = Array.isArray(hasil.data) ? hasil.data : [];
        const active = this.semesterList.find(s => String(s.status).toUpperCase() === 'ACTIVE');
        if (active) this.semesterId = String(active.id);
      } catch (e) {
        this.semesterList = [];
        this.semesterError = 'Daftar semester belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.semesterLoading = false;
      }
    },

    async buatSemesterDraf() {
      const f = this.semesterForm;
      const slug = this.classSlug || this.selectedClass;
      this.semesterFormErrorField = '';
      if (!slug) { this.semesterFormError = 'Kelas belum termuat.'; return; }
      if (!((f.academic_year || '').trim())) { this.semesterFormErrorField = 'academic_year'; this.semesterFormError = 'Tahun ajaran wajib diisi (contoh 2024/2025).'; return; }
      if (!((f.term || '').trim())) { this.semesterFormError = 'Semester (Ganjil/Genap) wajib diisi.'; return; }
      if (!f.starts_on || !f.ends_on) { this.semesterFormError = 'Tanggal mulai dan selesai wajib diisi.'; return; }
      if (!(f.ends_on > f.starts_on)) { this.semesterFormError = 'Tanggal selesai harus setelah tanggal mulai.'; return; }
      this.semesterFormError = '';
      this.semesterSaving = true;
      try {
        const payload = {
          academic_year: f.academic_year.trim(),
          term: f.term.trim(),
          starts_on: f.starts_on,
          ends_on: f.ends_on
        };
        if (f.source_semester_id) payload.source_semester_id = Number(f.source_semester_id);
        const res = await API.createSemesterDraft(slug, payload);
        this.showToast(f.source_semester_id ? 'Semester draf disalin dari semester sebelumnya.' : 'Semester draf dibuat.');
        this.semesterForm = { academic_year: '', term: 'Ganjil', starts_on: '', ends_on: '', source_semester_id: '' };
        await this.loadSemesters();
        await this.loadOfferings();
        if (res && res.id) await this.muatPreviewSemester(res.id);
      } catch (e) {
        this.semesterFormError = e.message || 'Gagal membuat semester draf.';
      } finally {
        this.semesterSaving = false;
      }
    },

    async muatPreviewSemester(id) {
      const slug = this.classSlug || this.selectedClass;
      if (!slug || !id) return;
      this.semesterPreviewLoading = true; this.semesterPreviewError = ''; this.semesterPreview = null;
      try {
        this.semesterPreview = await API.previewSemester(slug, id);
        if (!this.semesterPreview) this.semesterPreviewError = 'Pratinjau belum dapat dimuat. Coba lagi.';
      } catch (e) {
        this.semesterPreviewError = 'Pratinjau belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.semesterPreviewLoading = false;
      }
    },

    pilihFileImpor(ev) {
      const files = ev && ev.target && ev.target.files;
      this.imporFile = files && files[0] ? files[0] : null;
      this.imporHasil = null; this.imporError = '';
    },

    async validasiImpor(id) {
      if (!this.imporFile) { this.imporError = 'Pilih file JSON kurikulum dulu.'; return; }
      let payload = null;
      try {
        payload = JSON.parse(await this.imporFile.text());
      } catch (e) {
        this.imporError = 'File bukan JSON valid.';
        return;
      }
      this.imporError = ''; this.imporSaving = true;
      try {
        this.imporHasil = await API.importSemester(id, payload);
        if (this.imporHasil && this.imporHasil.has_fatal_errors) {
          this.imporError = 'Impor ditolak: perbaiki baris bertanda GALAT lalu validasi ulang.';
        } else {
          this.showToast('Validasi lolos. Terapkan untuk menyimpan.');
        }
      } catch (e) {
        this.imporHasil = null;
        this.imporError = e.message || 'Gagal memvalidasi impor.';
      } finally {
        this.imporSaving = false;
      }
    },

    async terapkanImpor(id) {
      const hasil = this.imporHasil;
      if (!hasil || !hasil.batch_id) { this.imporError = 'Validasi dulu sebelum menerapkan.'; return; }
      if (hasil.has_fatal_errors) { this.imporError = 'Masih ada GALAT. Perbaiki dulu sebelum menerapkan.'; return; }
      this.imporSaving = true;
      try {
        await API.applySemesterImport(id, hasil.batch_id);
        this.showToast('Impor diterapkan ke semester draf.');
        this.imporHasil = null; this.imporFile = null;
        await this.loadSemesters();
        await this.loadOfferings();
        await this.muatPreviewSemester(id);
      } catch (e) {
        this.imporError = e.message || 'Gagal menerapkan impor.';
      } finally {
        this.imporSaving = false;
      }
    },

    async simpanOfferingManual(semId) {
      const f = this.offeringForm;
      if (!semId) return;
      if (!((f.course_code || '').trim())) { this.offeringFormError = 'Kode mata kuliah wajib diisi.'; return; }
      if (!((f.display_name || '').trim())) { this.offeringFormError = 'Nama tampilan wajib diisi.'; return; }
      this.offeringFormError = '';
      this.offeringSaving = true;
      try {
        await API.createSemesterOffering(semId, {
          course_code: f.course_code.trim(),
          activity_type: (f.activity_type || 'TEORI').toUpperCase(),
          display_name: f.display_name.trim(),
          lecturer_codes: String(f.lecturer_codes || '').split(',').map(s => s.trim()).filter(Boolean)
        });
        this.showToast('Mata kuliah ditambahkan ke draf.');
        this.offeringForm = { course_code: '', activity_type: 'TEORI', display_name: '', lecturer_codes: '' };
        this.offeringTargetId = null;
        await this.loadOfferings();
        await this.muatPreviewSemester(semId);
      } catch (e) {
        this.offeringFormError = e.message || 'Gagal menambah mata kuliah.';
      } finally {
        this.offeringSaving = false;
      }
    },

    async bukaArsipSemester(s) {
      this.arsipDetail = s;
      this.arsipOfferings = [];
      this.arsipLoading = true;
      try {
        this.arsipOfferings = await API.getSemesterOfferings(s.id).catch(() => []) || [];
      } finally {
        this.arsipLoading = false;
      }
    },

    mulaiAktifkanSemester(s) {
      this.semesterAktif = s;
    },

    async jalankanAktifkanSemester() {
      const s = this.semesterAktif;
      if (!s) return;
      const slug = this.classSlug || this.selectedClass;
      try {
        await API.activateSemester(slug, s.id);
        this.showToast('Semester diaktifkan. Semester aktif lama menjadi arsip.');
        this.semesterAktif = null;
        await this.loadSemesters();
        await this.loadOfferings();
      } catch (e) {
        this.showToast(e.message || 'Gagal mengaktifkan semester.');
      }
    },

    async loadMateri() {
      const slug = this.classSlug || this.selectedClass;
      if (!slug) { this.materiList = []; return; }
      this.materiLoading = true; this.materiError = '';
      try {
        const data = await API.getMaterials(slug, '');
        if (data === null) {
          this.materiList = [];
          this.materiError = 'Materi belum dapat dimuat. Periksa koneksi lalu coba lagi.';
          return;
        }
        this.materiList = Array.isArray(data) ? data : [];
      } catch (e) {
        this.materiList = [];
        this.materiError = 'Materi belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.materiLoading = false;
      }
    },

    async simpanMateri() {
      const f = this.materiForm;
      const slug = this.classSlug || this.selectedClass;
      if (!slug) { this.materiFormError = 'Kelas belum termuat.'; return; }
      if (!((f.title || '').trim()) || String(f.title).trim().length < 3) { this.materiFormError = 'Judul materi minimal 3 karakter.'; return; }
      this.materiFormError = '';
      this.materiSaving = true;
      try {
        if (this.editMateriId) {
          await API.updateMaterial(this.editMateriId, {
            version: Number(this.editMateriVersion) || 0,
            title: f.title.trim(),
            material_type: (f.material_type || 'OTHER').toUpperCase(),
            url: (f.url || '').trim(),
            description: (f.description || '').trim()
          });
          this.showToast('Materi diperbarui.');
        } else {
          const payload = {
            class_slug: slug,
            title: f.title.trim(),
            material_type: (f.material_type || 'OTHER').toUpperCase()
          };
          if (f.offeringId) payload.offering_id = Number(f.offeringId);
          if ((f.url || '').trim()) payload.url = f.url.trim();
          if ((f.description || '').trim()) payload.description = f.description.trim();
          await API.createMaterial(payload);
          this.showToast('Materi tersimpan.');
        }
        this.materiForm = { offeringId: '', title: '', material_type: 'DOCUMENT', url: '', description: '' };
        this.editMateriId = null; this.editMateriVersion = 0;
        this.materiFormOpen = false;
        await this.loadMateri();
      } catch (e) {
        if (e.code === 'VERSION_CONFLICT') {
          this.materiFormError = 'Versi berubah di server. Muat ulang lalu ubah kembali.';
          await this.loadMateri();
        } else {
          this.materiFormError = e.message || 'Gagal menyimpan materi.';
        }
      } finally {
        this.materiSaving = false;
      }
    },

    mulaiUbahMateri(m) {
      this.materiForm = {
        offeringId: m.offering_id ? String(m.offering_id) : '',
        title: m.title || '', material_type: m.material_type || 'DOCUMENT',
        url: m.url || '', description: m.description || ''
      };
      this.materiFormError = '';
      this.editMateriId = m.id; this.editMateriVersion = m.version || 0;
      this.materiFormOpen = true;
      window.scrollTo({ top: 0 });
    },

    batalUbahMateri() {
      this.materiForm = { offeringId: '', title: '', material_type: 'DOCUMENT', url: '', description: '' };
      this.editMateriId = null; this.editMateriVersion = 0;
      this.materiFormError = '';
      this.materiFormOpen = false;
    },

    async arsipkanMateri(m) {
      if (!m || !m.id) return;
      if (!window.confirm('Arsipkan materi "' + (m.title || '') + '"? Materi tidak tampil di daftar.')) return;
      try {
        await API.archiveMaterial(m.id, m.version || 0);
        this.showToast('Materi diarsipkan.');
        await this.loadMateri();
      } catch (e) {
        this.showToast(e.message || 'Gagal mengarsipkan materi.');
        await this.loadMateri();
      }
    },

    materiTipeLabel(t) {
      const s = String(t || '').toUpperCase();
      if (s === 'DOCUMENT') return 'Dokumen';
      if (s === 'MEETING') return 'Tautan rapat';
      if (s === 'REPOSITORY') return 'Repositori';
      if (s === 'PORTAL') return 'Portal';
      return 'Lainnya';
    },

    async loadNotifikasi() {
      this.notifLoading = true; this.notifError = '';
      try {
        const extra = {};
        const slug = this.classSlug || this.selectedClass;
        const list = await API.getNotifications(this.notifFilter || '', 50, extra).catch((e) => { throw e; });
        this.notifList = Array.isArray(list) ? list.filter(n => {
          if (!slug) return true;
          if (!n.class_slug) return true;
          return String(n.class_slug) === String(slug);
        }) : [];
      } catch (e) {
        this.notifList = [];
        this.notifError = 'Notifikasi belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.notifLoading = false;
      }
    },

    statusNotifLabelKM(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'PENDING') return 'Menunggu';
      if (s === 'PROCESSING') return 'Diproses';
      if (s === 'SENT') return 'Terkirim';
      if (s === 'FAILED') return 'Gagal';
      if (s === 'CANCELLED') return 'Dibatalkan';
      return st || '-';
    },

    notifDapatDiulangKM(st) {
      const s = String(st || '').toUpperCase();
      return s === 'FAILED' || s === 'CANCELLED';
    },

    async toggleNotifDetailKM(id) {
      if (this.notifDetailId === id) { this.notifDetailId = null; return; }
      this.notifDetailId = id;
      this.notifAttempts = [];
      this.notifAttemptsError = '';
      await this.loadNotifAttemptsKM(id);
    },

    async loadNotifAttemptsKM(id) {
      this.notifAttemptsLoading = true; this.notifAttemptsError = '';
      try {
        this.notifAttempts = await API.getNotificationAttempts(id) || [];
      } catch (e) {
        this.notifAttempts = [];
        this.notifAttemptsError = 'Riwayat percobaan gagal dimuat.';
      } finally {
        this.notifAttemptsLoading = false;
      }
    },

    async ulangiPesanKM(id) {
      try {
        await API.retryNotification(id);
        this.showToast('Notifikasi dijadwalkan ulang.');
        await this.loadNotifikasi();
      } catch (e) {
        this.showToast(e.message || 'Gagal menjadwalkan ulang.');
      }
    },

    async loadPenugasan() {
      const slug = this.classSlug || this.selectedClass;
      this.penugasanLoading = true; this.penugasanError = '';
      try {
        const list = await API.getAdminAssignments('', '', slug || '').catch(() => { throw new Error('load'); });
        this.penugasanList = (Array.isArray(list) ? list : []).filter(a => {
          if (slug && a.class_slug) return String(a.class_slug) === String(slug);
          return true;
        });
      } catch (e) {
        this.penugasanList = [];
        this.penugasanError = 'Daftar penugasan belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.penugasanLoading = false;
      }
    },

    async loadUndanganKM() {
      const slug = this.classSlug || this.selectedClass;
      this.undanganLoading = true; this.undanganError = '';
      try {
        // Undangan KM dibuat System Admin dan di luar wewenang KM
        // (cabut oleh KM ditolak backend 403) — tampilkan PJ saja.
        const list = await API.getAdminInvitations('PENDING', 'PJ', slug || '').catch(() => { throw new Error('load'); });
        this.undanganList = (Array.isArray(list) ? list : []).filter(u => {
          if (String(u.role || '').toUpperCase() !== 'PJ') return false;
          if (slug && u.class_slug) return String(u.class_slug) === String(slug);
          return true;
        });
      } catch (e) {
        this.undanganList = [];
        this.undanganError = '';
      } finally {
        this.undanganLoading = false;
      }
    },

    penugasanLabel(role) {
      const r = String(role || '').toUpperCase();
      if (r === 'KM') return 'Ketua Murid';
      if (r === 'PJ') return 'PJ Mata Kuliah';
      return role || '-';
    },

    get penugasanPJ() {
      return this.penugasanList.filter(a => String(a.role || '').toUpperCase() === 'PJ');
    },

    get penugasanKM() {
      return this.penugasanList.filter(a => String(a.role || '').toUpperCase() === 'KM');
    },

    mulaiAksiPenugasan(a, aksi, pemicu) {
      const status = String(a.status || '').toUpperCase();
      if (String(a.role || '').toUpperCase() !== 'PJ') return;
      if (aksi === 'tangguhkan' && status !== 'ACTIVE') return;
      if (aksi === 'cabut' && status !== 'ACTIVE' && status !== 'SUSPENDED') return;
      this.penugasanAksi = {
        id: a.id,
        aksi: aksi,
        nama: a.display_name || a.username || ('#' + a.id),
        mataKuliah: a.offering_display || a.course_name || a.course_code || 'Mata kuliah belum tercatat',
        kode: a.course_code || ''
      };
      this.penugasanAlasan = '';
      this.penugasanError = '';
      this.penugasanPemicu = pemicu || null;
      this.$nextTick(() => {
        const dialog = document.getElementById('km-penugasan-dialog');
        if (dialog && !dialog.open) dialog.showModal();
      });
    },

    tutupAksiPenugasan() {
      if (this.penugasanSaving) return;
      const dialog = document.getElementById('km-penugasan-dialog');
      if (dialog?.open) dialog.close();
      else this.selesaikanAksiPenugasan();
    },

    selesaikanAksiPenugasan() {
      const pemicu = this.penugasanPemicu;
      this.penugasanAksi = null;
      this.penugasanAlasan = '';
      this.penugasanError = '';
      this.penugasanPemicu = null;
      this.$nextTick(() => {
        if (pemicu?.isConnected) pemicu.focus();
        else document.getElementById('km-penugasan-heading')?.focus();
      });
    },

    async jalankanAksiPenugasan() {
      const a = this.penugasanAksi;
      if (!a || this.penugasanSaving) return;
      if (!((this.penugasanAlasan || '').trim())) {
        this.penugasanError = 'Isi alasan tindakan terlebih dahulu.';
        document.getElementById('km-penugasan-alasan')?.focus();
        return;
      }
      this.penugasanSaving = true;
      this.penugasanError = '';
      try {
        await API.changeAssignmentStatus(a.id, a.aksi, this.penugasanAlasan.trim(), false);
        this.penugasanSaving = false;
        this.tutupAksiPenugasan();
        await this.loadPenugasan();
        document.getElementById('km-penugasan-heading')?.focus();
        this.showToast(a.aksi === 'cabut' ? 'Penugasan PJ dicabut.' : 'Penugasan PJ ditangguhkan.');
      } catch (e) {
        this.penugasanError = e.message || 'Gagal mengubah penugasan. Coba lagi.';
      } finally {
        this.penugasanSaving = false;
      }
    },

    async cabutUndanganKM(id) {
      try {
        await API.revokeInvitation(id, 'Dicabut KM');
        this.showToast('Undangan dicabut.');
        await this.loadUndanganKM();
      } catch (e) {
        this.showToast(e.message || 'Gagal mencabut undangan.');
      }
    },

    async loadPengaturanKelas() {
      const slug = this.classSlug || this.selectedClass;
      if (!slug) return;
      this.settingsLoading = true; this.settingsError = '';
      try {
        const s = await API.getClassSettings(slug).catch(() => null);
        this.classSettings = s;
        if (!s) { this.settingsError = 'Pengaturan kelas belum dapat dimuat.'; return; }
        if (s.morning_reminder_time) this.pengaturan.pagi = s.morning_reminder_time;
        if (s.afternoon_reminder_time) this.pengaturan.sore = s.afternoon_reminder_time;
        if (s.replacement_reminder_minutes !== undefined && s.replacement_reminder_minutes !== null) {
          this.pengaturan.gantiMenit = Number(s.replacement_reminder_minutes);
        }
        if (s.timezone) this.pengaturan.zona = s.timezone;
      } catch (e) {
        this.settingsError = 'Pengaturan kelas belum dapat dimuat.';
      } finally {
        this.settingsLoading = false;
      }
    },

    async simpanPengaturan() {
      const slug = this.classSlug || this.selectedClass;
      if (!slug) return;
      const hhmm = (v) => /^\d{2}:\d{2}$/.test(String(v || '')) && Number(String(v).slice(0, 2)) <= 23 && Number(String(v).slice(3)) <= 59;
      if (!hhmm(this.pengaturan.pagi) || !hhmm(this.pengaturan.sore)) {
        this.pengaturanFormError = 'Jam pengingat harus HH:MM (00:00–23:59).';
        return;
      }
      const menit = Number(this.pengaturan.gantiMenit);
      if (!Number.isInteger(menit) || menit < 0 || menit > 1440) {
        this.pengaturanFormError = 'Pengingat pengganti harus 0–1440 menit.';
        return;
      }
      this.pengaturanFormError = '';
      this.pengaturanSaving = true;
      try {
        await API.updateClassSettings(slug, {
          version: this.classSettings && this.classSettings.version,
          timezone: (this.pengaturan.zona || '').trim() || undefined,
          morning_reminder_time: this.pengaturan.pagi,
          afternoon_reminder_time: this.pengaturan.sore,
          replacement_reminder_minutes: menit
        });
        this.showToast('Pengaturan kelas disimpan. Berlaku pengiriman berikutnya.');
        await this.loadPengaturanKelas();
      } catch (e) {
        if (e.code === 'VERSION_CONFLICT') {
          this.pengaturanFormError = 'Versi berubah di server. Muat ulang lalu simpan kembali.';
          await this.loadPengaturanKelas();
        } else {
          this.pengaturanFormError = e.message || 'Gagal menyimpan pengaturan.';
        }
      } finally {
        this.pengaturanSaving = false;
      }
    },

    async rotasiKodeKelas() {
      const slug = this.classSlug || this.selectedClass;
      if (!slug) return;
      this.rotatingCode = true;
      try {
        const result = await API.rotatePortalCode(slug);
        this.portalCodeReveal = result && result.portal_code || '';
        this.showToast('Kode kelas dirotasi. Salin kode baru sebelum menutup dialog.');
        await this.loadPengaturanKelas();
      } catch (e) {
        this.showToast(e.message || 'Gagal merotasi kode kelas.');
      } finally {
        this.rotatingCode = false;
      }
    },

    tutupKodePortal() {
      this.portalCodeReveal = '';
    },

    salinKodePortal() {
      if (!this.portalCodeReveal) return;
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(this.portalCodeReveal).then(() => this.showToast('Kode portal disalin.')).catch(() => this.showToast('Kode tidak dapat disalin otomatis.'));
      } else this.showToast('Clipboard tidak didukung browser ini.');
    },

    async setModePortal(mode) {
      const slug = this.classSlug || this.selectedClass;
      if (!slug) return;
      try {
        await API.setPortalMode(slug, mode, 'Diubah KM');
        this.showToast(mode === 'CODE' ? 'Portal memakai kode kelas.' : 'Portal memakai tautan.');
        await this.loadPengaturanKelas();
      } catch (e) {
        this.showToast(e.message || 'Gagal mengubah mode portal.');
      }
    },

    mintaResetSandi() {
      this.showToast('Minta bantuan System Admin untuk mengatur ulang kata sandi.');
    },

    backupList: [], backupLoading: false, backupError: '',
    backupFormOpen: false,
    backupForm: { scope: 'kelas', semester_id: '', reason: '' },
    backupFormError: '', backupSaving: false,

    labelStatusBackup(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'PENDING') return 'Menunggu Admin';
      if (s === 'PROCESSING') return 'Sedang dibuat';
      if (s === 'EXECUTED') return 'Cadangan dibuat';
      if (s === 'REJECTED') return 'Ditolak';
      if (s === 'CREATING') return 'Diproses';
      if (s === 'READY') return 'Siap';
      if (s === 'VERIFIED') return 'Terverifikasi';
      if (s === 'FAILED') return 'Gagal';
      return st || '-';
    },

    async loadBackupSaya() {
      const slug = this.classSlug || this.selectedClass;
      if (!slug) { this.backupList = []; return; }
      this.backupLoading = true; this.backupError = '';
      try {
        this.backupList = await API.getBackupRequests() || [];
      } catch (e) {
        this.backupList = [];
        this.backupError = 'Daftar cadangan belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.backupLoading = false;
      }
    },

    async mintaBackup() {
      const f = this.backupForm;
      const slug = this.classSlug || this.selectedClass;
      if (!slug) { this.backupFormError = 'Kelas belum termuat.'; return; }
      if (!((f.reason || '').trim()) || String(f.reason).trim().length < 5) {
        this.backupFormError = 'Alasan minimal 5 karakter.';
        return;
      }
      this.backupFormError = '';
      this.backupSaving = true;
      try {
        const payload = { class_slug: slug, reason: f.reason.trim() };
        if (f.scope === 'semester') {
          if (!f.semester_id) { this.backupFormError = 'Pilih semester untuk cakupan semester.'; return; }
          payload.semester_id = Number(f.semester_id);
        }
        await API.createBackupRequest(payload);
        this.showToast('Permintaan cadangan tercatat dan menunggu System Admin.');
        this.backupForm = { scope: 'kelas', semester_id: '', reason: '' };
        this.backupFormOpen = false;
        await this.loadBackupSaya();
      } catch (e) {
        this.backupFormError = e.message || 'Gagal meminta cadangan.';
      } finally {
        this.backupSaving = false;
      }
    },

    copyText(text, okMsg) {
      const done = () => this.showToast(okMsg || 'Tersalin.');
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done).catch(() => this.showToast('Gagal menyalin otomatis.'));
      } else {
        this.showToast('Clipboard tidak tersedia di browser ini.');
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
        window.location.href = '/login.html';
      }, 500);
    },

    showToast(msg) {
      if (this.toast.timer) clearTimeout(this.toast.timer);
      this.toast.message = msg;
      this.toast.show = true;
      this.toast.timer = setTimeout(() => { this.toast.show = false; }, 3000);
    },

    // ---- Kanal WhatsApp ----
    async loadKanalSaya() {
      this.kanalLoading = true; this.kanalError = '';
      try {
        this.kanalList = await API.getChannels('', '');
      } catch (e) {
        this.kanalList = [];
        this.kanalError = 'Daftar kanal belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.kanalLoading = false;
      }
    },

    labelStatusKanal(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'ACTIVE') return 'Aktif';
      if (s === 'DISCONNECTED') return 'Terputus';
      if (s === 'REVOKED') return 'Dilepas';
      return st || '-';
    },

    async tautkanKanalSaya() {
      const f = this.kanalForm;
      if (!((f.jid || '').trim()) || !(f.jid || '').includes('@')) { this.kanalFormError = 'JID grup wajib diisi (minta via perintah !kanal di grup).'; return; }
      this.kanalFormError = '';
      this.kanalSaving = true;
      try {
        const res = await API.linkChannel(f.jid.trim(), '', (f.nama || '').trim());
        this.showToast(res && res.changed === false ? 'Kanal sudah tertaut ke kelas ini.' : 'Kanal berhasil ditautkan.');
        this.kanalForm = { jid: '', nama: '' };
        await this.loadKanalSaya();
      } catch (err) {
        this.kanalFormError = err.message || 'Gagal menautkan kanal.';
      } finally {
        this.kanalSaving = false;
      }
    },

    mulaiLepasKanalSaya(k) {
      this.kanalLepas = { id: k.id, nama: k.display_name || k.jid };
      this.kanalAlasan = '';
    },

    async jalankanLepasKanalSaya() {
      const k = this.kanalLepas;
      if (!k) return;
      if (!((this.kanalAlasan || '').trim())) {
        this.showToast('Isi alasan pelepasan terlebih dahulu.');
        return;
      }
      try {
        await API.revokeChannel(k.id, this.kanalAlasan.trim());
        this.showToast(`Kanal ${k.nama} dilepas.`);
        this.kanalLepas = null;
        await this.loadKanalSaya();
      } catch (err) {
        this.showToast(err.message || 'Gagal melepas kanal.');
      }
    },

    // ---- Kanal WhatsApp kelas (KM menautkan grupnya sendiri) ----
    kanalList: [],
    kanalLoading: false,
    kanalError: '',
    kanalForm: { jid: '', nama: '' },
    kanalFormError: '',
    kanalSaving: false,
    kanalLepas: null,
    kanalAlasan: '',

    // ---- Usulan koreksi master (KM mengusulkan, System Admin memutuskan) ----
    usulanList: [],
    usulanLoading: false,
    usulanError: '',
    usulanFilterKind: '',
    usulanFilterStatus: '',
    usulanTargetRuang: [],
    usulanTargetMatkul: [],
    usulanForm: { kind: 'ROOM', targetId: '', kode: '', nama: '', gedung: '', tipe: '', kapasitas: '', catatan: '' },
    usulanFormError: '',
    usulanSaving: false,

    async loadUsulanTarget() {
      try {
        const [ruang, matkul] = await Promise.all([
          API.getMasterRooms('ACTIVE').catch(() => []),
          API.getMasterCourses('ACTIVE').catch(() => []),
        ]);
        this.usulanTargetRuang = Array.isArray(ruang) ? ruang : [];
        this.usulanTargetMatkul = Array.isArray(matkul) ? matkul : [];
      } catch (e) {
        this.usulanTargetRuang = [];
        this.usulanTargetMatkul = [];
      }
    },

    async loadUsulanSaya() {
      this.usulanLoading = true; this.usulanError = '';
      try {
        this.usulanList = await API.getProposals(this.usulanFilterStatus || '', this.usulanFilterKind || '');
      } catch (e) {
        this.usulanList = [];
        this.usulanError = 'Daftar usulan belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.usulanLoading = false;
      }
    },

    usulanTargetList() {
      const arr = this.usulanForm.kind === 'ROOM' ? this.usulanTargetRuang : this.usulanTargetMatkul;
      return (arr || []).map(x => ({ id: x.id, label: (x.code || '') + ' · ' + (x.name || '') }));
    },

    labelStatusUsulan(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'PENDING') return 'Menunggu';
      if (s === 'APPROVED') return 'Disetujui';
      if (s === 'REJECTED') return 'Ditolak';
      return st || '-';
    },

    async kirimUsulan() {
      const f = this.usulanForm;
      const isBaru = !((f.targetId || '').toString().trim());
      this.usulanFormError = '';
      const payload = {};
      if (f.kind === 'ROOM') {
        if (isBaru) {
          if (!((f.kode || '').trim())) { this.usulanFormError = 'Kode wajib diisi untuk ruangan baru.'; return; }
          payload.code = f.kode.trim();
        }
        if ((f.nama || '').trim()) payload.name = f.nama.trim();
        if ((f.gedung || '').trim()) payload.building = f.gedung.trim();
        if ((f.tipe || '').trim()) payload.room_type = f.tipe.trim();
        if ((f.kapasitas || '') !== '') {
          if (!(/^\d+$/.test(String(f.kapasitas).trim()))) { this.usulanFormError = 'Kapasitas wajib angka bulat ≥ 0.'; return; }
          payload.capacity = Number(String(f.kapasitas).trim());
        }
        if (Object.keys(payload).length === 0) { this.usulanFormError = 'Isi minimal satu field yang diusulkan.'; return; }
      } else {
        if (!((f.nama || '').trim())) { this.usulanFormError = 'Nama mata kuliah wajib diisi.'; return; }
        payload.name = f.nama.trim();
        if (isBaru) {
          if (!((f.kode || '').trim())) { this.usulanFormError = 'Kode wajib diisi untuk mata kuliah baru.'; return; }
          payload.code = f.kode.trim();
        }
      }
      const body = { kind: f.kind, payload, note: (f.catatan || '').trim() };
      if (!isBaru) body.target_id = Number(f.targetId);
      this.usulanSaving = true;
      try {
        await API.createProposal(body);
        this.showToast('Usulan terkirim. System Admin akan meninjau.');
        this.usulanForm = { kind: 'ROOM', targetId: '', kode: '', nama: '', gedung: '', tipe: '', kapasitas: '', catatan: '' };
        await this.loadUsulanSaya();
      } catch (err) {
        this.usulanFormError = err.message || 'Gagal mengirim usulan.';
      } finally {
        this.usulanSaving = false;
      }
    }
  };
}
