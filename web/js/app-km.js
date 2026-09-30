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

    roleLabel: 'KM',

    navSections: [
      { title: 'KM', items: [
        { id: 'dashboard', label: 'Dashboard', img: '/assets/icons/home.svg' },
      ] },
      { title: 'AKADEMIK', items: [
        { id: 'tugas', label: 'Tugas', img: '/assets/icons/tasks.svg' },
        { id: 'antrean', label: 'Antrean pemeriksaan', img: '/assets/icons/ext-check.svg' },
        { id: 'jadwal', label: 'Jadwal', img: '/assets/icons/calendar.svg' },
        { id: 'materi', label: 'Materi', img: '/assets/icons/folder.svg' },
      ] },
      { title: 'KELOLA KELAS', items: [
        { id: 'anggota', label: 'Anggota & Tim', img: '/assets/icons/ext-settings-edit.svg' },
        { id: 'pengaturan', label: 'Pengaturan Kelas', img: '/assets/icons/settings.svg' },
        { id: 'usulan', label: 'Usulan Master', img: '/assets/icons/book.svg' },
      ] },
      { title: 'LAINNYA', items: [
        { id: 'monitoring', label: 'Monitoring', img: '/assets/icons/activity.svg' },
        { id: 'log', label: 'Log Aktivitas', img: '/assets/icons/ext-check.svg' },
        { id: 'notifikasi', label: 'Notifikasi', img: '/assets/icons/bell.svg' },
        { id: 'akun', label: 'Akun', img: '/assets/icons/event.svg' },
      ] },
    ],

    get nav() { return this.navSections.flatMap(s => s.items); },
    get kmNav() { return this.nav; },
    get roleSub() { return 'Pengelola seluruh kelas'; },

    isActive(item) { const a = item.active || [item.id]; return a.includes(this.view); },

    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed;
      try { localStorage.setItem('asterisk:sidebar:collapsed', this.sidebarCollapsed ? '1' : '0'); } catch (e) {}
    },

    knownViews: ['dashboard', 'tugas', 'antrean', 'jadwal', 'materi', 'anggota', 'pengaturan', 'usulan', 'monitoring', 'log', 'notifikasi', 'akun'],

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

    tugasSub: 'list',
    tugasTab: 'aktif',
    tugasDari: '', tugasSampai: '',
    tugasLoading: false, tugasListError: '',
    tugasMatkul: 'Semua',
    tugasSort: 'dekat',
    tugasForm: { offeringId: '', judul: '', tanggal: '', jam: '', deskripsi: '', kumpul: '', kumpulUrl: '', jenis: 'Individu' },
    tugasError: '',
    tugasDetail: null, tugasReviews: [], tugasDetailLoading: false, tugasDetailTab: 'detail',
    tugasPreview: null, tugasPublishMsg: '',
    editTugasId: '', editTugasVersion: 0, tugasConflict: null,
    patternsList: [],

    anggotaSub: 'list',
    undang: { offeringId: '', nomor: '' },
    undangError: '',
    undangLink: '',

    reviewId: null,
    reviewMode: 'koreksi',
    reviewNote: '',

    // Pengaturan (lokal sampai endpoint tersedia)
    pengaturan: {
      pagi: localStorage.getItem('km_rem_pagi') || '06:00',
      sore: localStorage.getItem('km_rem_sore') || '17:00'
    },

    toast: { show: false, message: '', timer: null },

    get todayList() {
      return this.fullSchedule.filter(s => s.hari === this.todayName);
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
          && String(t.review_state || 'NOT_REVIEWED').toUpperCase() === 'NOT_REVIEWED' && !t.archived_at)
        .slice().sort((a, b) => String(a.deadline_at || '').localeCompare(String(b.deadline_at || '')));
    },

    get tugasTabCounts() {
      const c = { aktif: 0, draf: 0, review: 0, selesai: 0, terlewat: 0, arsip: 0 };
      (this.tasks || []).forEach(t => {
        if (t.archived_at) { c.arsip++; return; }
        if (t.completed_at) { c.selesai++; return; }
        if (String(t.publication_status || '').toUpperCase() === 'DRAFT') { c.draf++; return; }
        if (String(t.review_state || 'NOT_REVIEWED').toUpperCase() === 'NOT_REVIEWED') c.review++;
        if (this.lewatDeadline(t)) { c.terlewat++; return; }
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
        else if (t.completed_at) { if (this.tugasTab !== 'selesai') return false; }
        else if (String(t.publication_status || '').toUpperCase() === 'DRAFT') { if (this.tugasTab !== 'draf') return false; }
        else if (this.tugasTab === 'review') {
          if (String(t.review_state || 'NOT_REVIEWED').toUpperCase() !== 'NOT_REVIEWED') return false;
        }
        else if (this.lewatDeadline(t)) { if (this.tugasTab !== 'terlewat') return false; }
        else if (this.tugasTab !== 'aktif') return false;
        if (dari || sampai) {
          const d = t.deadline_at ? new Date(t.deadline_at) : null;
          if (!d) return false;
          if (dari && d < dari) return false;
          if (sampai && d > sampai) return false;
        }
        return true;
      }).slice().sort((a, b) => this.tugasSort === 'dekat'
        ? String(a.deadline_at || '').localeCompare(String(b.deadline_at || ''))
        : String(b.deadline_at || '').localeCompare(String(a.deadline_at || '')));
    },

    get tugasFilterAktif() { return !!(this.tugasDari || this.tugasSampai || (this.tugasMatkul && this.tugasMatkul !== 'Semua')); },
    hapusTugasFilter() { this.tugasDari = ''; this.tugasSampai = ''; this.tugasMatkul = 'Semua'; },

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

    fmtDeadlineID(iso) {
      try {
        const d = new Date(iso);
        if (isNaN(d)) return String(iso || '-');
        return d.toLocaleString('id-ID', { timeZone: 'Asia/Jakarta', weekday: 'long', day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false }) + ' WIB';
      } catch (e) { return String(iso || '-'); }
    },

    lewatDeadline(t) {
      try {
        if (!t || !t.deadline_at || t.completed_at) return false;
        return new Date(t.deadline_at) < new Date();
      } catch (e) { return false; }
    },

    deadlineBadge(iso, completed) {
      try {
        if (completed) return { level: 'aman', badge: 'Selesai' };
        const d = new Date(iso);
        if (isNaN(d)) return { level: 'aman', badge: 'Aktif' };
        const diffH = (d - new Date()) / 3600000;
        if (diffH < 0) return { level: 'mendesak', badge: 'Terlewat' };
        if (diffH < 24) return { level: 'mendesak', badge: 'Besok' };
        if (diffH <= 72) return { level: 'mendekati', badge: `H-${Math.ceil(diffH / 24)}` };
        return { level: 'aman', badge: 'Aktif' };
      } catch (e) {
        return { level: 'aman', badge: 'Aktif' };
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
        return { name: n, dateNum: d.getDate(), full: d };
      });
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
      return this.fullSchedule
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

    slotAt(dayName, slotHH) {
      return this.fullSchedule.find(s => s.hari === dayName && parseInt((s.timeStart || '0').split(':')[0], 10) === parseInt(slotHH, 10));
    },

    // ===== Alur Perubahan Jadwal (mengikuti docs/design/flows/SCHEDULE_MANAGEMENT.md) =====
    jadwalSub: 'daftar',
    filtMatkul: '', filtDosen: '', filtRuang: '',
    polaLoading: false, polaError: '',
    polaForm: { id: '', version: 0, offeringId: '', day: '1', start: '', end: '', roomId: '', link: '', effectiveDate: '' },
    polaFormError: '',
    ubahForm: { offeringId: '', kind: 'REPLACEMENT', scope: 'sementara', originPatternId: '', originDate: '', date: '', start: '', end: '', roomId: '', link: '', reason: '', effectiveDate: '', participantIds: '', conflictReason: '' },
    ubahFormError: '',
    draftEvent: null,
    previewData: null, previewLoading: false, previewError: '',
    eventsList: [], eventsLoading: false, eventsError: '',
    eventFilter: 'semua',
    selectedEvent: null, revokeReason: '', revokeError: '',
    roomCands: [], roomCandsLoading: false,
    roomConfirm: { roomId: '', name: '', note: '' },

    get polaRooms() {
      const map = new Map();
      (this.patternsList || []).forEach(p => {
        if (p.room_id && !map.has(String(p.room_id))) {
          map.set(String(p.room_id), { id: String(p.room_id), code: p.room || p.room_code || ('Ruang ' + p.room_id) });
        }
      });
      return Array.from(map.values());
    },

    get scopePatterns() { return this.patternsList || []; },

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

    fmtWaktuID(iso) {
      try {
        const d = new Date(iso);
        return d.toLocaleString('id-ID', { timeZone: 'Asia/Jakarta', weekday: 'long', day: 'numeric', month: 'long', hour: '2-digit', minute: '2-digit', hour12: false });
      } catch (e) { return String(iso || '-'); }
    },

    canEditPattern(p) { return true; },

    hapusPolaFilter() { this.filtMatkul = ''; this.filtDosen = ''; this.filtRuang = ''; },

    bukaDaftar() { this.jadwalSub = 'daftar'; window.scrollTo({ top: 0 }); },

    mulaiUbah(prefill) {
      const f = this.ubahForm;
      f.offeringId = (prefill && prefill.offeringId) || f.offeringId || '';
      f.kind = (prefill && prefill.kind) || 'REPLACEMENT';
      f.scope = 'sementara';
      f.originPatternId = ''; f.originDate = ''; f.date = ''; f.start = ''; f.end = '';
      f.roomId = ''; f.link = ''; f.reason = ''; f.effectiveDate = ''; f.participantIds = ''; f.conflictReason = '';
      if (prefill && prefill.dayName) {
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
      this.polaForm = { id: '', version: 0, offeringId: '', day: '1', start: '', end: '', roomId: '', link: '', effectiveDate: '' };
      this.polaFormError = '';
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

    async simpanPola() {
      const f = this.polaForm;
      if (!f.offeringId) { this.polaFormError = 'Pilih mata kuliah di kelas ini dulu.'; return; }
      if (!f.start || !f.end) { this.polaFormError = 'Jam mulai dan jam selesai wajib diisi.'; return; }
      const dur = this.durasiMenit(f.start, f.end);
      if (dur <= 0) { this.polaFormError = 'Jam selesai harus setelah jam mulai.'; return; }
      this.polaFormError = '';
      try {
        if (f.id) {
          const payload = { version: Number(f.version) || 0, offering_id: Number(f.offeringId), day_of_week: Number(f.day), start_time: f.start, duration_min: dur };
          if (f.roomId) payload.room_id = Number(f.roomId);
          if ((f.link || '').trim()) payload.meeting_link = f.link.trim();
          if (f.effectiveDate) payload.effective_from = f.effectiveDate;
          await API.patchPattern(f.id, payload);
          this.showToast('Jadwal reguler diperbarui.');
        } else {
          const payload = { offering_id: Number(f.offeringId), day_of_week: Number(f.day), start_time: f.start, duration_min: dur };
          if (f.roomId) payload.room_id = Number(f.roomId);
          if ((f.link || '').trim()) payload.meeting_link = f.link.trim();
          await API.createPattern(payload);
          this.showToast('Jadwal reguler ditambahkan.');
        }
        this.patternsList = await API.getPatterns().catch(() => []);
        this.jadwalSub = 'daftar';
      } catch (err) {
        this.polaFormError = err.message || 'Gagal menyimpan pola jadwal.';
      }
    },

    async loadEvents() {
      this.eventsLoading = true; this.eventsError = '';
      try {
        this.eventsList = await API.getTeachingEvents().catch(() => []) || [];
      } catch (e) {
        this.eventsList = []; this.eventsError = 'Perubahan jadwal belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.eventsLoading = false;
      }
    },

    validasiUbah() {
      const f = this.ubahForm;
      if (!f.offeringId) return 'Pilih mata kuliah di kelas ini dulu.';
      if ((f.kind === 'REPLACEMENT' || f.kind === 'SESSION_CANCELLED') && !f.originPatternId) return 'Pilih jadwal semula untuk kelas pengganti atau sesi yang dibatalkan.';
      if ((f.kind === 'REPLACEMENT' || f.kind === 'SESSION_CANCELLED') && !f.originDate) return 'Isi tanggal kejadian asal.';
      if (f.kind !== 'SESSION_CANCELLED' && (!f.date || !f.start || !f.end)) return 'Tanggal serta jam mulai dan selesai wajib diisi.';
      if (f.kind !== 'SESSION_CANCELLED' && this.durasiMenit(f.start, f.end) <= 0) return 'Jam selesai harus setelah jam mulai.';
      if (!f.reason || f.reason.trim().length < 5) return 'Keterangan minimal 5 karakter.';
      if (f.scope === 'permanen' && !f.effectiveDate) return 'Isi tanggal mulai berlaku untuk perubahan permanen.';
      return '';
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
      if (this.overrideConflicts.length > 0 && !(this.ubahForm.conflictReason || '').trim()) {
        this.showToast('Isi alasan pengecualian konflik sebelum menerbitkan.'); return;
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

    bukaDetail(ev) {
      this.selectedEvent = ev;
      this.revokeReason = ''; this.revokeError = '';
      this.jadwalSub = 'detail';
      window.scrollTo({ top: 0 });
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

    async catatKonfirmasiRuang() {
      if (!this.draftEvent || !this.draftEvent.id || !this.roomConfirm.roomId) {
        this.showToast('Pilih ruangan dan lengkapi catatan konfirmasi dulu.'); return;
      }
      try {
        await API.confirmTeachingEventRoom(this.draftEvent.id, {
          room_id: Number(this.roomConfirm.roomId),
          confirmation_status: 'CONFIRMED',
          confirmed_by: this.roomConfirm.name || undefined,
          note: this.roomConfirm.note || undefined
        });
        this.showToast('Konfirmasi ruangan tercatat.');
      } catch (e) {
        this.showToast(e.message || 'Gagal mencatat konfirmasi.');
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
        this.classSlug = this.resolveClassSlug(me);
      } catch (e) {
        localStorage.removeItem('access_token');
        window.location.replace('/login.html?role=km');
        return;
      }

      await this.loadPartials([
        ['km-sidebar', '/partials/common/sidebar.html'],
        ['km-state', '/partials/common/state-error.html'],
        ['km-topbar', '/partials/common/topbar.html'],
        ['km-dashboard', '/partials/km/view-dashboard.html'],
        ['km-tugas', '/partials/km/view-tugas.html'],
        ['km-jadwal', '/partials/km/view-jadwal.html'],
        ['km-materi', '/partials/km/view-materi.html'],
        ['km-anggota', '/partials/km/view-anggota.html'],
        ['km-antrean', '/partials/km/view-antrean.html'],
        ['km-monitor', '/partials/km/view-monitor.html'],
        ['km-notif', '/partials/km/view-notif.html'],
        ['km-pengaturan', '/partials/km/view-pengaturan.html'],
        ['km-usulan', '/partials/km/view-usulan.html'],
        ['km-drawer', '/partials/common/drawer.html'],
        ['km-bottombar', '/partials/common/bottombar.html'],
        ['km-toast', '/partials/common/toast.html']
      ]);
      // Skeleton dimuat susulan: targetnya berada di dalam partial dashboard.
      await this.loadPartials([
        ['km-skeleton', '/partials/common/skeleton-dashboard.html'],
      ]);
      this.updateClock();
      setInterval(() => this.updateClock(), 1000);
      await this.checkBot();
      await this.loadClasses();
      await this.loadOfferings();
      await this.loadSchedule();
      await this.loadTasks();
      this.patternsList = await API.getPatterns().catch(() => []);
      await this.loadEvents();
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
      if (v === 'tugas-tambah') {
        this.mulaiTambahTugas();
        return;
      }
      if (!this.knownViews.includes(v)) { this.showPageError('404'); return; }
      this.view = v;
      this.drawer = false;
      if (v === 'tugas') this.tugasSub = 'list';
      if (v === 'anggota') this.anggotaSub = 'list';
      if (v === 'log') this.loadAuditLog();
      if (v === 'usulan') { this.loadUsulanTarget(); this.loadUsulanSaya(); }
      window.scrollTo({ top: 0 });
    },

    soon(fitur) { this.showToast(`${fitur}: fitur belum tersedia.`); },

    async checkBot() {
      try {
        const st = await API.getStatus();
        if (st) this.botOnline = String(st.bot_connection || '').toLowerCase() === 'connected';
      } catch (e) { this.botOnline = false; }
    },

    async loadClasses() {
      try {
        const data = await API.getClasses();
        if (data && data.default_class) {
          this.selectedClass = data.default_class;
          this.classList = data.classes || [];
        }
      } catch (e) { /* fallback */ }
      try {
        const st = await API.getStatus();
        if (st && st.default_class) {
          if (!this.selectedClass) this.selectedClass = st.default_class;
          if (st.classes) this.classList = st.classes;
        }
      } catch (e) { /* kosong */ }
      if (this.meCache) {
        const s = this.resolveClassSlug(this.meCache);
        if (s) this.classSlug = s;
      }
    },

    async loadSchedule() {
      try {
        const raw = await API.getSchedule(this.selectedClass, 'all');
        const seen = new Set();
        const list = [];
        (raw || []).forEach((s, i) => {
          const parts = String(s.jam || '').split('-').map(x => x.trim().replace('.', ':'));
          const entry = {
            id: `sch-${i}`, hari: s.hari, jam: s.jam, matkul: s.matkul,
            dosen: s.dosen, ruang: s.ruang,
            timeStart: parts[0] || '', timeEnd: parts[1] || ''
          };
          const key = `${entry.hari}|${entry.timeStart}|${entry.matkul}|${entry.ruang || ''}`;
          if (seen.has(key)) return;
          seen.add(key);
          list.push(entry);
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
          this.offeringState = 'no-v1-class'; this.offeringList = []; this.semesterId = ''; return;
        }
        if (!hasil.ok) {
          this.offeringState = 'error'; this.offeringList = []; this.semesterId = ''; return;
        }
        const active = list.find(s => s.status === 'ACTIVE') || list[0];
        if (!active) {
          this.offeringState = 'empty-semester'; this.offeringList = []; this.semesterId = ''; return;
        }
        this.semesterId = String(active.id);
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

    parseDeadlineID(tanggal, jam) {
      const t = String(tanggal || '').trim();
      const j = String(jam || '').trim().replace('.', ':');
      let datePart = '';
      if (/^\d{4}-\d{2}-\d{2}$/.test(t)) {
        datePart = t;
      } else {
        const dm = t.match(/(\d{1,2})\s+([A-Za-z]+)\s*(\d{4})?/);
        if (!dm) return '';
        const months = { jan: '01', feb: '02', mar: '03', apr: '04', mei: '05', jun: '06', jul: '07', agu: '08', sep: '09', okt: '10', nov: '11', des: '12' };
        const month = months[dm[2].toLowerCase().slice(0, 3)] || '';
        if (!month) return '';
        const year = dm[3] || new Date().getFullYear();
        datePart = `${year}-${month}-${String(dm[1]).padStart(2, '0')}`;
      }
      const hm = j.match(/(\d{1,2})[:.](\d{2})/);
      if (!hm) return '';
      return `${datePart}T${String(hm[1]).padStart(2, '0')}:${hm[2]}:00+07:00`;
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
      try {
        const res = await API.createTask(this.rakitTugasPayload('published'));
        const newId = res && res.data && res.data.id;
        await this.loadTasks();
        this.tugasPublishMsg = 'Tugas diterbitkan.';
        this.tugasPreview = null;
        this.tugasSub = 'list';
        if (newId) {
          await this.bukaDetailTugas(newId);
        } else {
          this.view = 'tugas';
          this.showToast(this.tugasPublishMsg);
        }
      } catch (err) {
        this.tugasError = err.message || 'Gagal menerbitkan tugas di server.';
        this.tugasSub = 'tinjau';
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
      this.tugasPreview = {
        matkul: this.offeringDisplay(f.offeringId),
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
      this.tugasPublishMsg = this.tugasPublishMsg || '';
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

    fmtWaktuID(iso) {
      try {
        const d = new Date(iso);
        if (isNaN(d)) return String(iso || '-');
        return d.toLocaleString('id-ID', { timeZone: 'Asia/Jakarta', day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false }) + ' WIB';
      } catch (e) { return String(iso || '-'); }
    },

    labelAksiAudit(action) {
      const a = String(action || '').toUpperCase();
      const map = {
        CREATE_TASK: 'Tugas dibuat', REVIEW_TASK: 'Tugas diperiksa',
        COMPLETE_TASK: 'Tugas selesai', ARCHIVE_TASK: 'Tugas diarsipkan', RESTORE_TASK: 'Tugas dipulihkan',
        CREATE_PATTERN: 'Jadwal ditambahkan', CREATE_TEACHING_EVENT: 'Perubahan dibuat',
        PUBLISH_EVENT: 'Perubahan diterbitkan', REVOKE_EVENT: 'Publikasi dicabut',
        CREATE_BACKUP: 'Cadangan dibuat', ASSIGN_ROLE: 'Peran ditetapkan'
      };
      return map[a] || action || '-';
    },

    async loadAuditLog() {
      this.auditLoading = true; this.auditError = '';
      try {
        this.auditList = await API.getAudit({ limit: 50 }).catch(() => null) || [];
      } catch (e) {
        this.auditList = [];
        this.auditError = 'Riwayat belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.auditLoading = false;
      }
    },

    simpanPengaturan() {
      localStorage.setItem('km_rem_pagi', this.pengaturan.pagi);
      localStorage.setItem('km_rem_sore', this.pengaturan.sore);
      this.showToast('Pengaturan tersimpan di perangkat ini.');
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
