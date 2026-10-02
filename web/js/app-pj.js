/**
 * web/js/app-pj.js — Area PJ (REVISI Figma).
 * Cakupan PJ: hanya mata kuliah yang ditugaskan (dipilih lokal, disimpan
 * di perangkat sampai endpoint penugasan backend ada).
 */
function pjApp() {
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
    pjMatkul: localStorage.getItem('pj_matkul') || '',
    offeringId: localStorage.getItem('pj_offering_id') || '',
    offeringList: [],

    materiList: [],
    materiLoading: false,
    materiError: '',
    materiSort: 'terbaru',
    materiFormOpen: false,
    materiForm: { title: '', material_type: 'DOCUMENT', url: '', description: '' },
    materiFormError: '',
    materiSaving: false,
    editMateriId: null, editMateriVersion: 0,
    offeringLoading: false,
    offeringState: 'idle',

    roleLabel: 'PJ',
    contextAssignments: [],
    contextSwitching: false,

    navSections: [
      { title: 'PJ', items: [
        { id: 'dashboard', label: 'Dashboard', img: '/assets/icons/home.svg' },
      ] },
      { title: 'AKADEMIK', items: [
        { id: 'tugas', label: 'Tugas', img: '/assets/icons/tasks.svg', active: ['tugas', 'tambah', 'tinjau-tugas', 'detail-tugas', 'ubah-tugas', 'preview', 'konfirmasi', 'terbit'] },
        { id: 'jadwal', label: 'Jadwal', img: '/assets/icons/calendar.svg', active: ['jadwal', 'pindah', 'perubahan'] },
        { id: 'ruangan', label: 'Ruangan', img: '/assets/icons/event.svg' },
        { id: 'materi', label: 'Materi', img: '/assets/icons/folder.svg' },
      ] },
      { title: 'LAINNYA', items: [
        { id: 'semester', label: 'Semester', img: '/assets/icons/event.svg' },
        { id: 'audit', label: 'Riwayat Perubahan', img: '/assets/icons/activity.svg' },
        { id: 'status', label: 'Status pemeriksaan', img: '/assets/icons/ext-check.svg' },
        { id: 'akun', label: 'Akun', img: '/assets/icons/event.svg' },
      ] },
    ],

    get nav() { return this.navSections.flatMap(s => s.items); },
    get pjNav() { return this.nav; },
    get roleSub() { return this.pjMatkul || 'Mata kuliah belum dipilih'; },
    // Disediakan agar topbar/drawer bersama tetap hidup di area PJ.
    get antrean() { return []; },
    get notifGagal() { return 0; },
    get attentionCount() { return 0; },
    get activeSemesterLabel() { return ''; },

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
        window.location.href = role === 'KM' ? '/km.html' : role === 'SYSTEM_ADMIN' ? '/system-admin.html' : '/pj.html';
      } catch (err) {
        this.showToast(err.message || 'Konteks akses tidak tersedia.');
      } finally { this.contextSwitching = false; }
    },

    knownViews: ['dashboard', 'tugas', 'tambah', 'tinjau-tugas', 'detail-tugas', 'ubah-tugas', 'preview', 'konfirmasi', 'terbit', 'jadwal', 'pindah', 'perubahan', 'ruangan', 'materi', 'semester', 'audit', 'status', 'akun'],

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
    semesterList: [], semesterLoading: false, semesterError: '',
    auditList: [], auditLoading: false, auditError: '', auditDetailId: null,
    botOnline: false,
    fullSchedule: [],
    tasks: [],

    tugasForm: { judul: '', tanggal: '', jam: '', deskripsi: '', kumpul: '', kumpulUrl: '', jenis: 'Individu' },
    tugasError: '',
    terbitOk: false,
    tugasTab: 'aktif',
    tugasDari: '', tugasSampai: '',
    tugasLoading: false, tugasListError: '',
    tugasDetail: null, tugasReviews: [], tugasDetailLoading: false, tugasDetailTab: 'detail',
    tugasPreview: null, tugasPublishMsg: '',
    taskDelivery: [], eventDelivery: [], deliveryLoading: false, deliveryError: '',
    editTugasId: '', editTugasVersion: 0, tugasConflict: null,

    patternsList: [],

    toast: { show: false, message: '', timer: null },

    get matkulOpts() {
      return Array.from(new Set(this.fullSchedule.map(s => s.matkul)));
    },

    get scopeList() {
      if (!this.pjMatkul) return this.fullSchedule;
      return this.fullSchedule.filter(s => s.matkul === this.pjMatkul);
    },

    get todayList() {
      return this.scopeList.filter(s => s.hari === this.todayName);
    },

    get withUrgency() {
      const rank = { mendesak: 0, mendekati: 1, aman: 2 };
      return [...this.tugasAktif].sort((a, b) => (rank[a.urgency] ?? 2) - (rank[b.urgency] ?? 2));
    },

    // Tugas aktif = terbit, belum selesai, tidak diarsip (cakupan matkul sendiri).
    get tugasAktif() {
      return (this.tasks || []).filter(t =>
        String(t.publication_status || '').toUpperCase() === 'PUBLISHED' && !t.completed_at && !t.archived_at);
    },

    get tugasTabCounts() {
      const c = { aktif: 0, draf: 0, selesai: 0, terlewat: 0, arsip: 0 };
      (this.tasks || []).forEach(t => {
        if (t.archived_at) { c.arsip++; return; }
        if (t.completed_at) { c.selesai++; return; }
        if (String(t.publication_status || '').toUpperCase() === 'DRAFT') { c.draf++; return; }
        if (this.lewatDeadline(t)) { c.terlewat++; return; }
        c.aktif++;
      });
      return c;
    },

    get tugasTabList() {
      const dari = this.tugasDari ? new Date(this.tugasDari + 'T00:00:00+07:00') : null;
      const sampai = this.tugasSampai ? new Date(this.tugasSampai + 'T23:59:59+07:00') : null;
      return (this.tasks || []).filter(t => {
        if (t.archived_at) { if (this.tugasTab !== 'arsip') return false; }
        else if (t.completed_at) { if (this.tugasTab !== 'selesai') return false; }
        else if (String(t.publication_status || '').toUpperCase() === 'DRAFT') { if (this.tugasTab !== 'draf') return false; }
        else if (this.lewatDeadline(t)) { if (this.tugasTab !== 'terlewat') return false; }
        else if (this.tugasTab !== 'aktif') return false;
        if (dari || sampai) {
          const d = t.deadline_at ? new Date(t.deadline_at) : null;
          if (!d) return false;
          if (dari && d < dari) return false;
          if (sampai && d > sampai) return false;
        }
        return true;
      }).slice().sort((a, b) => String(a.deadline_at || '').localeCompare(String(b.deadline_at || '')));
    },

    get tugasFilterAktif() { return !!(this.tugasDari || this.tugasSampai); },
    hapusTugasFilter() { this.tugasDari = ''; this.tugasSampai = ''; },

    pubLabel(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'DRAFT') return 'Draf';
      if (s === 'PUBLISHED') return 'Terbit';
      if (s === 'REVOKED') return 'Publikasi Dicabut';
      return st || '-';
    },

    reviewLabel(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'NOT_REVIEWED') return 'Perlu diperiksa KM';
      if (s === 'APPROVED') return 'Disetujui';
      if (s === 'CHANGES_REQUESTED') return 'Perlu koreksi';
      if (s === 'REVOKED') return 'Dibatalkan';
      return st || '-';
    },

    fmtDeadlineID(iso) { return API.fmtDeadlineID(iso); },
    lewatDeadline(t) { return API.lewatDeadline(t); },
    deadlineBadge(iso, completed) { return API.deadlineBadge(iso, completed); },

    get tugasSaya() {
      if (!this.pjMatkul) return [];
      return this.withUrgency.filter(t => t.matkul === this.pjMatkul);
    },

    get urgentSaya() { return this.tugasSaya.slice(0, 3); },
    get nearSaya() { return this.tugasSaya.filter(t => t.urgency !== 'aman').length; },
    get perluCount() {
      return this.tugasSaya.filter(t => t.urgency === 'mendesak').length;
    },

    get roomList() {
      return Array.from(new Set(this.scopeList.map(s => s.ruang).filter(Boolean)));
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
        return { name: n, dateNum: d.getDate(), full: d };
      });
    },

    get weekLabel() {
      const fmt = (d) => d.getDate() + ' ' + d.toLocaleString('id-ID', { month: 'short', timeZone: 'Asia/Jakarta' });
      const days = this.weekDays;
      return `${fmt(days[0].full)} – ${fmt(days[4].full)} ${days[4].full.getFullYear()}`;
    },

    // Slot jam unik ascending. Fixed baseline agar grid stabil saat data kosong,
    // digabung dengan jam unik dari data lalu didedupe + sort.
    get slots() {
      const base = ['08', '10', '13', '15'];
      const fromData = (this.fullSchedule || []).map(s => String(s.timeStart || '').split(':')[0].padStart(2, '0')).filter(h => /^\d{2}$/.test(h));
      return Array.from(new Set([...base, ...fromData])).sort();
    },

    // Flat cell list (slot-major) agar template hanya 1x loop.
    // Menghindari nested <template x-for> yang rapuh di Alpine + double-init.
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
      return this.scopeList
        .filter(s => s.hari === dayName)
        .slice()
        .sort((a, b) => String(a.timeStart || '').localeCompare(String(b.timeStart || '')));
    },

    goPindah(dayName, slotHH) {
      this.mulaiUbah({ dayName: dayName, slotHH: slotHH });
    },

    // Samakan kode kelas legacy (mis. D4-TI-SMT3-A) ke slug kanonis v1
    // (mis. d4-ti-smt3-a) memakai active_assignment + daftar classes dari /me.
    resolveClassSlug(me) {
      const asg = (me && me.active_assignment) || {};
      if (asg.class_slug) return String(asg.class_slug);
      const list = (me && me.classes) || [];
      const code = String(this.selectedClass || '').toLowerCase();
      if (code && Array.isArray(list)) {
        const hit = list.find(c => String(c.slug || '') === code || String(c.code || '').toLowerCase() === code);
        if (hit && hit.slug) return String(hit.slug);
      }
      return String(this.selectedClass || '');
    },

    slotSaya(dayName, slotHH) {
      return this.scopeList.find(s => s.hari === dayName && parseInt((s.timeStart || '0').split(':')[0], 10) === parseInt(slotHH, 10));
    },

    // ===== Alur Perubahan Jadwal (mengikuti docs/design/flows/SCHEDULE_MANAGEMENT.md) =====
    // SCR-SCH-001 daftar pola · SCR-SCH-002 form · SCR-SCH-003 tinjau
    // SCR-SCH-004 detail · SCR-SCH-005 partisipasi kelas (KM pemilik)
    jadwalSub: 'daftar',
    filtMatkul: '', filtDosen: '', filtRuang: '',
    polaLoading: false, polaError: '',
    polaForm: { id: '', version: 0, offeringId: '', day: '1', start: '', end: '', roomId: '', link: '', effectiveDate: '' },
    polaFormError: '',
    polaPreview: null, polaPreviewPayload: '', polaPreviewLoading: false,
    roomSearch: { date: '', start: '', end: '' }, roomCandidates: null, roomCandidatesLoading: false, roomCandidatesError: '',
    roomHistory: [], roomHistoryLoading: false, roomHistoryError: '',
    ubahForm: { kind: 'REPLACEMENT', scope: 'sementara', originPatternId: '', originDate: '', date: '', day: '1', start: '', end: '', roomId: '', link: '', reason: '', effectiveDate: '', participantIds: '', conflictReason: '' },
    ubahFormError: '',
    draftEvent: null,
    previewData: null, previewLoading: false, previewError: '',
    eventsList: [], eventsLoading: false, eventsError: '',
    eventFilter: 'semua',
    selectedEvent: null, revokeReason: '', revokeError: '',
    eventDetailLoading: false, eventDetailError: '',
    roomCands: [], roomCandsLoading: false,
    roomConfirm: { roomId: '', status: 'PENDING', name: '', note: '' },

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
      if (!this.pjMatkul) return this.patternsList || [];
      return (this.patternsList || []).filter(p =>
        String(p.display_name || p.course_name || p.offering || '') === String(this.pjMatkul));
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

    statusLabel(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'DRAFT') return 'Draf';
      if (s === 'PUBLISHED') return 'Terbit';
      if (s === 'REVOKED') return 'Publikasi Dicabut';
      return st || '-';
    },

    fmtWaktuID(iso) { return API.fmtWaktuID(iso); },

    canEditPattern(p) {
      if (!this.offeringId) return false;
      return String(p.course_offering_id || '') === String(this.offeringId);
    },

    hapusPolaFilter() { this.filtMatkul = ''; this.filtDosen = ''; this.filtRuang = ''; },

    bukaDaftar() { this.jadwalSub = 'daftar'; window.scrollTo({ top: 0 }); },

    mulaiUbah(prefill) {
      const f = this.ubahForm;
      f.kind = (prefill && prefill.kind) || 'REPLACEMENT';
      f.scope = 'sementara';
      f.originPatternId = ''; f.originDate = ''; f.date = ''; f.day = '1'; f.start = ''; f.end = '';
      f.roomId = ''; f.link = ''; f.reason = ''; f.effectiveDate = ''; f.participantIds = ''; f.conflictReason = '';
      if (prefill && prefill.dayName) {
        const hit = this.slotSaya(prefill.dayName, prefill.slotHH || '');
        if (hit) {
          const pat = (this.patternsList || []).find(p =>
            String(p.display_name || p.course_name || p.offering || '') === String(hit.matkul));
          if (pat) f.originPatternId = String(pat.id);
          if (hit.timeStart) f.start = hit.timeStart.slice(0, 5);
          if (hit.timeEnd) f.end = hit.timeEnd.slice(0, 5);
        }
      }
      this.ubahFormError = '';
      this.view = 'jadwal'; this.jadwalSub = 'ubah';
      window.scrollTo({ top: 0 });
    },

    mulaiTambahPola() {
      this.polaForm = { id: '', version: 0, offeringId: this.offeringId || '', day: '1', start: '', end: '', duration: 100, roomId: '', link: '', effectiveDate: '' };
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
        this.patternsList = await API.getPatterns();
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
        this.mulaiUbah();
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
      try { this.patternsList = await API.getPatterns(); }
      catch (e) { this.patternsList = []; this.polaError = e.message || 'Jadwal tetap gagal dimuat.'; }
      finally { this.polaLoading = false; }
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
      if (!this.offeringId) return 'Pilih mata kuliah di kelas ini di Dashboard dulu.';
      if (f.scope === 'permanen') {
        if (!f.originPatternId) return 'Pilih jadwal tetap yang akan diganti.';
        if (!f.effectiveDate) return 'Isi tanggal mulai berlaku.';
        if (!f.start || !f.end || this.durasiMenit(f.start, f.end) <= 0) return 'Isi jam mulai dan selesai yang valid.';
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
        duration_min: this.durasiMenit(f.start, f.end), effective_from: f.effectiveDate, reason: f.reason.trim() };
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
        owner_offering_id: Number(this.offeringId),
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
          this.showToast('Jadwal tetap baru berlaku sesuai tanggal pilihan.');
          this.jadwalSub = 'daftar';
        } catch (e) { this.showToast(e.message || 'Gagal menerbitkan perubahan permanen.'); }
        return;
      }
      if (this.overrideConflicts.length > 0 && !(this.ubahForm.conflictReason || '').trim()) {
        this.showToast('Isi alasan pengecualian konflik sebelum menerbitkan.'); return;
      }
      try {
        await API.publishTeachingEvent(this.draftEvent.id, (this.ubahForm.conflictReason || '').trim() || null, this.draftEvent.version || 1);
        this.showToast('Perubahan jadwal terbit.');
        await this.loadEvents();
        const found = (this.eventsList || []).find(e => String(e.id) === String(this.draftEvent.id));
        this.selectedEvent = found || null;
        this.loadDelivery('TEACHING_EVENT', this.draftEvent.id);
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
      this.loadDelivery('TEACHING_EVENT', ev.id);
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
      f.kind = ev.event_kind || 'REPLACEMENT';
      f.reason = ev.reason || '';
      f.date = String(ev.starts_at || '').slice(0, 10);
      f.start = String(ev.starts_at || '').slice(11, 16);
      f.end = String(ev.ends_at || '').slice(11, 16);
      this.jadwalSub = 'tinjau';
      window.scrollTo({ top: 0 });
      await this.muatPratinjau();
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

    async catatKonfirmasiRuang() {
      if (!this.draftEvent || !this.draftEvent.id || !this.roomConfirm.roomId) {
        this.showToast('Pilih ruangan dan lengkapi catatan konfirmasi dulu.'); return;
      }
      try {
        await API.confirmTeachingEventRoom(this.draftEvent.id, {
          room_id: Number(this.roomConfirm.roomId),
          confirmation_status: this.roomConfirm.status,
          external_contact: this.roomConfirm.name || undefined,
          note: this.roomConfirm.note || undefined
        });
        this.showToast('Konfirmasi ruangan tercatat.');
        const current = (this.eventsList || []).find(e => String(e.id) === String(this.draftEvent.id));
        if (current) await this.bukaDetail(current);
      } catch (e) {
        this.showToast(e.message || 'Gagal mencatat konfirmasi.');
      }
    },

    async initPJ() {
      try { this.sidebarCollapsed = localStorage.getItem('asterisk:sidebar:collapsed') === '1'; } catch (e) {}
      window.addEventListener('offline', () => { this.showPageError('offline'); });
      window.addEventListener('online', () => { if (this.pageState && this.pageState.status === 'offline') window.location.reload(); });
      const token = API.getAuthToken ? API.getAuthToken() : localStorage.getItem('access_token');
      if (!token) {
        window.location.replace('/login.html?role=pj');
        return;
      }
      try {
        const me = await API.getMe();
        if (!me || !me.user) {
          localStorage.removeItem('access_token');
          window.location.replace('/login.html?role=pj');
          return;
        }
        const role = me.active_assignment && me.active_assignment.role;
        if (role === 'KM') {
          window.location.replace('/km.html');
          return;
        }
        if (role && role !== 'PJ' && role !== 'SYSTEM_ADMIN') {
          window.location.replace('/login.html?role=pj');
          return;
        }
        this.currentUser = me.user;
        this.activeRole = role || null;
        this.meCache = me;
        this.contextAssignments = Array.isArray(me.assignments) ? me.assignments : [];
        this.classSlug = this.resolveClassSlug(me);
        this.selectedClass = this.classSlug;
        const assignedOffering = me.active_assignment && me.active_assignment.offering_id;
        if (assignedOffering && !this.offeringId) {
          this.offeringId = String(assignedOffering);
          localStorage.setItem('pj_offering_id', this.offeringId);
        }
      } catch (e) {
        localStorage.removeItem('access_token');
        window.location.replace('/login.html?role=pj');
        return;
      }

      await this.loadPartials([
        ['pj-sidebar', '/partials/common/sidebar.html'],
        ['pj-state', '/partials/common/state-error.html'],
        ['pj-topbar', '/partials/common/topbar.html'],
        ['pj-dashboard', '/partials/pj/view-dashboard.html'],
        ['pj-tugas', '/partials/pj/view-tugas.html'],
        ['pj-jadwal', '/partials/pj/view-jadwal.html'],
        ['pj-rooms', '/partials/common/view-rooms.html'],
        ['pj-materi', '/partials/pj/view-materi.html'],
        ['pj-semester', '/partials/pj/view-semester.html'],
        ['pj-audit', '/partials/pj/view-audit.html'],
        ['pj-status', '/partials/pj/view-status.html'],
        ['pj-akun', '/partials/pj/view-akun.html'],
        ['pj-drawer', '/partials/common/drawer.html'],
        ['pj-bottombar', '/partials/common/bottombar.html'],
        ['pj-toast', '/partials/common/toast.html']
      ]);
      // Skeleton dimuat susulan: targetnya berada di dalam partial dashboard.
      await this.loadPartials([
        ['pj-skeleton', '/partials/common/skeleton-dashboard.html'],
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
      await this.loadSemesterPJ();
      setInterval(() => this.checkBot(), 30000);
      this.dashboardLoading = false;
    },

    async loadPartials(slots) {
      // Alpine v3 auto-init node baru via MutationObserver.
      // Jangan panggil Alpine.initTree manual di sini: menyebabkan x-for ter-render 2x.
      await Promise.all(slots.map(async ([id, url]) => {
        try {
          const res = await fetch(url + '?v=20261013', { cache: 'no-store' });
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
        this.todayFull = `${day}, ${get('day')} ${get('month')} ${get('year')}`;
        this.currentTime = `${get('hour')}:${get('minute')}:${get('second')} WIB`;
      } catch (e) {
        this.todayName = 'Senin';
        this.todayFull = 'Senin';
      }
    },

    go(v) {
      this.pageState = null;
      if (!this.knownViews.includes(v)) { this.showPageError('404'); return; }
      this.view = v;
      this.drawer = false;
      if (v === 'materi') this.loadMateri();
      if (v === 'ruangan') this.loadRoomHistory();
      if (v === 'semester') this.loadSemesterPJ();
      if (v === 'audit') this.loadAuditPJ();
      window.scrollTo({ top: 0 });
    },

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

    semesterLabelPJ(s) {
      return `${s.academic_year || ''} · ${s.term || ''}`;
    },

    async loadSemesterPJ() {
      this.semesterLoading = true; this.semesterError = '';
      try {
        const result = await API.getSemestersResult(this.classSlug || this.selectedClass);
        if (!result.ok) throw new Error(result.status === 403 ? 'Konteks ini tidak diizinkan melihat semester.' : 'Semester gagal dimuat.');
        this.semesterList = Array.isArray(result.data) ? result.data : [];
      } catch (err) {
        this.semesterList = []; this.semesterError = err.message || 'Semester gagal dimuat.';
      } finally { this.semesterLoading = false; }
    },

    async loadAuditPJ() {
      this.auditLoading = true; this.auditError = '';
      try {
        const rows = await API.getAudit({ limit: 50 });
        if (!Array.isArray(rows)) throw new Error('Riwayat tidak tersedia.');
        this.auditList = rows;
      } catch (err) {
        this.auditList = []; this.auditError = err.message || 'Riwayat perubahan gagal dimuat.';
      } finally { this.auditLoading = false; }
    },

    labelAuditPJ(action) {
      return String(action || '').replaceAll('_', ' ').toLowerCase().replace(/^./, c => c.toUpperCase());
    },

    soon(fitur) { this.showToast(`${fitur}: fitur belum tersedia.`); },

    simpanMatkul() {
      localStorage.setItem('pj_matkul', this.pjMatkul);
      this.showToast(this.pjMatkul ? `Cakupan: ${this.pjMatkul}` : 'Cakupan dikosongkan.');
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
          this.offeringState = 'no-v1-class'; this.offeringList = []; return;
        }
        if (!hasil.ok) {
          this.offeringState = 'error'; this.offeringList = []; return;
        }
        const active = list.find(s => s.status === 'ACTIVE') || list[0];
        if (!active) { this.offeringState = 'empty-semester'; this.offeringList = []; return; }
        const offerings = await API.getSemesterOfferings(active.id).catch(() => null);
        if (offerings === null) { this.offeringState = 'error'; this.offeringList = []; return; }
        const assignedOffering = this.meCache && this.meCache.active_assignment && this.meCache.active_assignment.offering_id;
        this.offeringList = (Array.isArray(offerings) ? offerings : []).filter(o =>
          assignedOffering && String(o.id) === String(assignedOffering));
        this.offeringState = this.offeringList.length > 0 ? 'ok' : 'empty-offering';
        const ids = this.offeringList.map(o => String(o.id));
        if (this.offeringId && !ids.includes(String(this.offeringId))) {
          this.offeringId = '';
          localStorage.removeItem('pj_offering_id');
        }
        if (!this.offeringId && this.offeringList.length === 1) {
          this.offeringId = String(this.offeringList[0].id);
          localStorage.setItem('pj_offering_id', this.offeringId);
        }
        this.syncMatkulFromOffering();
      } catch (e) {
        this.offeringList = [];
        this.offeringState = 'error';
      } finally {
        this.offeringLoading = false;
      }
    },

    pesanOffering() {
      if (this.offeringState === 'no-v1-class') return 'Kelas ' + (this.selectedClass || 'ini') + ' belum terdaftar di database (kode: NOT_FOUND). Minta Administrator membuat kelas tersebut, lalu siapkan semester dan mata kuliah.';
      if (this.offeringState === 'empty-semester') return 'Belum ada semester untuk kelas ini. Minta Administrator menyiapkan semester dan mata kuliah.';
      if (this.offeringState === 'empty-offering') return 'Semester aktif belum memiliki mata kuliah.';
      if (this.offeringState === 'error') return 'Daftar mata kuliah gagal dimuat. Periksa koneksi lalu coba lagi.';
      if (this.offeringState === 'no-class') return 'Kelas belum termuat. Muat ulang halaman.';
      return '';
    },

    simpanOffering() {
      if (this.offeringId) {
        localStorage.setItem('pj_offering_id', String(this.offeringId));
      } else {
        localStorage.removeItem('pj_offering_id');
      }
      this.syncMatkulFromOffering();
      this.loadTasks();
    },

    syncMatkulFromOffering() {
      const found = (this.offeringList || []).find(o => String(o.id) === String(this.offeringId));
      const name = found ? (found.display_name || found.course_code || '') : '';
      if (name) {
        this.pjMatkul = name;
        localStorage.setItem('pj_matkul', name);
      }
    },

    offeringName() {
      const found = (this.offeringList || []).find(o => String(o.id) === String(this.offeringId));
      return found ? (found.display_name || '') : (this.pjMatkul || '');
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
        const [rawPatterns, rawEvents] = await Promise.all([API.getPatterns(), API.getTeachingEvents()]);
        const raw = (rawPatterns || []).filter(p => !this.offeringId || String(p.course_offering_id) === String(this.offeringId));
        const seen = new Set();
        const list = [];
        (raw || []).forEach((s, i) => {
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
        (rawEvents || []).filter(e => String(e.lifecycle_status || '').toUpperCase() === 'PUBLISHED' && (!this.offeringId || String(e.offering_id) === String(this.offeringId))).forEach((e, i) => {
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
        const raw = await API.getAllTasks(this.offeringId || '');
        let list = (raw || []).map(t => {
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
        if (this.pjMatkul) {
          const scoped = list.filter(t => !t.matkul || t.matkul === this.pjMatkul);
          list = scoped.length > 0 ? scoped : list;
        }
        this.tasks = list;
      } catch (e) {
        this.tasks = [];
        this.tugasListError = 'Tugas belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.tugasLoading = false;
      }
    },

    parseDeadlineID(tanggal, jam) { return API.parseDeadlineID(tanggal, jam); },

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

    mulaiTambah() {
      if (!this.offeringId) { this.showToast('Pilih mata kuliah di kelas ini yang ditugaskan dulu di Dashboard.'); return; }
      this.tugasForm = { judul: '', tanggal: '', jam: '', deskripsi: '', kumpul: '', kumpulUrl: '', jenis: 'Individu' };
      this.tugasError = '';
      this.terbitOk = false;
      this.editTugasId = ''; this.editTugasVersion = 0; this.tugasConflict = null;
      this.view = 'tambah';
      window.scrollTo({ top: 0 });
    },

    validasiTugasTerbit() {
      const f = this.tugasForm;
      if (!this.offeringId) return 'Pilih mata kuliah di kelas ini di Dashboard dulu.';
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
        offering_id: Number(this.offeringId),
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

    async simpanDrafTugas() {
      const f = this.tugasForm || {};
      if (!this.offeringId) { this.showToast('Pilih mata kuliah di kelas ini di Dashboard dulu.'); return; }
      if (!f.judul || f.judul.trim().length < 5) { this.showToast('Judul tugas minimal 5 karakter.'); return; }
      try {
        await API.createTask(this.rakitTugasPayload('draft'));
        await this.loadTasks();
        this.tugasTab = 'draf';
        this.showToast('Draf tersimpan di server.');
        this.view = 'tugas';
      } catch (err) {
        this.showToast(err.message || 'Gagal menyimpan draf di server.');
      }
    },
    async simpanDraf() { return this.simpanDrafTugas(); },

    tinjauTugas() {
      const err = this.validasiTugasTerbit();
      if (err) { this.tugasError = err; window.scrollTo({ top: 0 }); return; }
      this.tugasError = '';
      const f = this.tugasForm;
      this.tugasPreview = {
        matkul: this.offeringName() || this.pjMatkul || '',
        judul: f.judul.trim(),
        deadline_at: this.parseDeadlineID(f.tanggal, f.jam),
        deskripsi: f.deskripsi.trim(),
        jenis: f.jenis || '',
        kumpul: (f.kumpul || '').trim(),
        kumpulUrl: (f.kumpulUrl || '').trim()
      };
      this.view = 'tinjau-tugas';
      window.scrollTo({ top: 0 });
    },

    async terbitTugas() {
      if (!this.tugasPreview) { this.tinjauTugas(); return; }
      this.tugasError = '';
      try {
        const res = await API.createTask(this.rakitTugasPayload('published'));
        const newId = res && res.data && res.data.id;
        await this.loadTasks();
        this.terbitOk = true;
        this.tugasPublishMsg = 'Tugas diterbitkan dan perlu diperiksa KM.';
        this.tugasPreview = null;
        if (newId) {
          await this.bukaDetailTugas(newId);
        } else {
          this.view = 'tugas';
          this.showToast(this.tugasPublishMsg);
        }
      } catch (err) {
        this.tugasError = err.message || 'Gagal menerbitkan tugas di server.';
        this.view = 'tinjau-tugas';
      }
    },

    async bukaDetailTugas(id) {
      this.tugasDetailLoading = true;
      this.tugasDetail = null; this.tugasReviews = [];
      this.tugasDetailTab = 'detail'; this.tugasConflict = null;
      this.tugasPublishMsg = this.tugasPublishMsg || '';
      this.view = 'detail-tugas';
      this.loadDelivery('TASK', id);
      window.scrollTo({ top: 0 });
      try {
        const d = await API.getTaskDetail(id);
        const info = (d && (d.task || d)) || null;
        if (!info) { this.showPageError('404'); return; }
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
          completed_at: info.completed_at || null, archived_at: info.archived_at || null,
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
        this.view = 'tugas';
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
      this.tugasForm = { judul: d.title || '', tanggal: tgl, jam: jam, deskripsi: d.instructions || d.deskripsi || '',
        kumpul: d.submission_text || '', kumpulUrl: d.submission_url || '', jenis: d.task_type || 'Individu' };
      this.tugasError = '';
      this.editTugasId = d.id; this.editTugasVersion = d.version || 0;
      this.tugasConflict = null;
      this.view = 'ubah-tugas';
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
        this.showToast('Perubahan tersimpan sebagai versi baru dan perlu diperiksa KM.');
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
        if (this.view === 'detail-tugas') await this.bukaDetailTugas(id);
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
        if (this.view === 'detail-tugas') this.view = 'tugas';
      } catch (e) {
        this.showToast(e.message || 'Gagal mengarsipkan tugas.');
        await this.loadTasks();
      }
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

    lihatTugas(t) { this.bukaDetailTugas(t.id); },

    async loadDelivery(kind, id) {
      this.deliveryLoading = true; this.deliveryError = '';
      try {
        const rows = await API.getPublicationDelivery(kind, id);
        if (kind === 'TASK') this.taskDelivery = rows;
        else this.eventDelivery = rows;
      } catch (e) { this.deliveryError = e.message || 'Status pengiriman gagal dimuat.'; }
      finally { this.deliveryLoading = false; }
    },

    // Materi — lingkup offering penugasan PJ.
    get materiTampil() {
      const list = (this.materiList || []).slice();
      const terbaru = this.materiSort !== 'terlama';
      return list.sort((a, b) => {
        const ta = a.created_at ? new Date(a.created_at).getTime() : 0;
        const tb = b.created_at ? new Date(b.created_at).getTime() : 0;
        return terbaru ? (tb - ta) : (ta - tb);
      });
    },

    materiTipeLabel(t) {
      const s = String(t || '').toUpperCase();
      if (s === 'DOCUMENT') return 'Dokumen';
      if (s === 'MEETING') return 'Tautan rapat';
      if (s === 'REPOSITORY') return 'Repositori';
      if (s === 'PORTAL') return 'Portal';
      return 'Lainnya';
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

    async loadMateri() {
      const slug = this.classSlug || this.selectedClass;
      if (!slug) { this.materiList = []; return; }
      this.materiLoading = true; this.materiError = '';
      try {
        const data = await API.getMaterials(slug, this.offeringId || '');
        if (data === null) {
          this.materiList = [];
          this.materiError = 'Materi belum dapat dimuat. Periksa koneksi lalu coba lagi.';
          return;
        }
        const list = Array.isArray(data) ? data : [];
        this.materiList = this.offeringId
          ? list.filter(m => String(m.offering_id || '') === String(this.offeringId))
          : list;
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
      if (!this.offeringId) { this.materiFormError = 'Pilih mata kuliah penugasan di Dashboard dulu.'; return; }
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
            offering_id: Number(this.offeringId),
            title: f.title.trim(),
            material_type: (f.material_type || 'OTHER').toUpperCase()
          };
          if ((f.url || '').trim()) payload.url = f.url.trim();
          if ((f.description || '').trim()) payload.description = f.description.trim();
          await API.createMaterial(payload);
          this.showToast('Materi tersimpan.');
        }
        this.materiForm = { title: '', material_type: 'DOCUMENT', url: '', description: '' };
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
        title: m.title || '', material_type: m.material_type || 'DOCUMENT',
        url: m.url || '', description: m.description || ''
      };
      this.materiFormError = '';
      this.editMateriId = m.id; this.editMateriVersion = m.version || 0;
      this.materiFormOpen = true;
      window.scrollTo({ top: 0 });
    },

    batalUbahMateri() {
      this.materiForm = { title: '', material_type: 'DOCUMENT', url: '', description: '' };
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
    }
  };
}
