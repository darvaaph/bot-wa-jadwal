/**
 * web/js/app-portal.js — Portal mahasiswa (REVISI Figma): hanya-baca, tanpa akun.
 * Akses kelas diverifikasi ulang oleh backend saat portal dibuka.
 */
function portalApp() {
  return {
    unlocked: localStorage.getItem('portal_pin_ok') === '1',
    pin: '',
    pinError: '',
    view: 'dashboard',
    drawer: false,
    sidebarCollapsed: false,
    pageState: null,
    dashboardLoading: true,
    q: '',
    unreadCount: 0,
    hari: 'Senin',
    tugasMatkul: 'Semua',

    portalNav: [
      { id: 'dashboard', label: 'Ringkasan', img: '/assets/icons/home.svg' },
      { id: 'jadwal', label: 'Jadwal', img: '/assets/icons/calendar.svg' },
      { id: 'tugas', label: 'Tugas', img: '/assets/icons/tasks.svg' },
      { id: 'courses', label: 'Mata Kuliah', img: '/assets/icons/folder.svg' }
    ],

    todayName: 'Senin',
    todayFull: '',
    semesterLabel: '',
    selectedClass: '',
    selectedProdi: 'D4',
    selectedSemester: '3',
    selectedAbjad: 'A',
    classList: [],
    comboboxOpen: false,
    classQuery: '',
    classFilterProdi: 'ALL',
    gateStep: 'select',
    checkingAccess: false,
    fullSchedule: [],
    jadwalEfektif: [],
    jadwalCacheHariIni: [],
    jadwalLoading: false,
    jadwalError: '',
    tasks: [],
    tugasGrup: { hari_ini: [], minggu_ini: [], mendatang: [], terlewat: [] },
    tugasTab: 'mendatang',
    tugasMatkul: 'Semua',
    tugasLoading: false,
    tugasError: '',
    materiList: [],
    materiLoading: false,
    courses: [], coursesLoading: false, coursesError: '', selectedCourse: null,
    semesterList: [],
    semesterLoading: false,
    archiveSelected: null,
    archiveLoading: false,
    archiveError: '',
    archiveData: { schedule: [], tasks: [], materials: [], changes: [] },
    perubahanList: [],
    perubahanLoading: false,
    detailTugas: {},
    detailMateri: [],

    toast: { show: false, message: '', timer: null },

    get todayList() {
      return this.jadwalEfektifHariIni;
    },

    // Tanggal ISO tiap hari Senin–Jumat pada pekan berjalan (zona WIB).
    get tanggalPekan() {
      const out = {};
      try {
        const names = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat'];
        const nowWib = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
        const dow = (nowWib.getDay() + 6) % 7;
        const monday = new Date(nowWib);
        monday.setDate(nowWib.getDate() - dow);
        names.forEach((n, i) => {
          const d = new Date(monday);
          d.setDate(monday.getDate() + i);
          const pad = (x) => String(x).padStart(2, '0');
          out[n] = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
        });
      } catch (e) {}
      return out;
    },

    get jadwalEfektifHariIni() {
      return (this.jadwalCacheHariIni || []).slice().sort((a, b) =>
        String(a.timeStart || '').localeCompare(String(b.timeStart || '')));
    },

    get withUrgency() {
      const rank = { mendesak: 0, mendekati: 1, aman: 2 };
      return [...this.tasks].sort((a, b) => (rank[a.urgency] ?? 2) - (rank[b.urgency] ?? 2));
    },

    get urgent3() { return this.withUrgency.slice(0, 3); },
    get nearCount() { return this.tasks.filter(t => t.urgency !== 'aman').length; },

    get matkulOpts() {
      const dariTugas = (this.tasks || []).map(t => t.matkul).filter(Boolean);
      const dariJadwal = (this.fullSchedule || []).map(s => s.matkul).filter(Boolean);
      return Array.from(new Set([...dariTugas, ...dariJadwal]));
    },

    get tugasTabList() {
      const list = this.tugasGrup[this.tugasTab] || [];
      return list.filter(t => this.tugasMatkul === 'Semua' || t.matkul === this.tugasMatkul);
    },

    get jadwalHari() {
      return (this.jadwalEfektif || []).slice().sort((a, b) =>
        String(a.timeStart || '').localeCompare(String(b.timeStart || '')));
    },

    get perubahanTerbaru() { return (this.perubahanList || []).slice(0, 3); },

    async bukaArsipSemester(semester) {
      if (!semester || String(semester.status || '').toUpperCase() !== 'ARCHIVED') return;
      this.archiveSelected = semester;
      this.archiveLoading = true;
      this.archiveError = '';
      this.archiveData = { schedule: [], tasks: [], materials: [], changes: [] };
      try {
        const start = new Date(`${semester.starts_on}T00:00:00+07:00`);
        const end = new Date(`${semester.ends_on}T23:59:59+07:00`);
        const days = [];
        if (!Number.isNaN(start.getTime())) {
          for (let i = 0; i < 7; i++) {
            const d = new Date(start); d.setDate(start.getDate() + i);
            if (d > end) break;
            days.push(d.toISOString().slice(0, 10));
          }
        }
        const [schedules, tasks, materials, changes] = await Promise.all([
          Promise.all(days.map(d => API.getPortalSchedule(this.selectedClassSlug, d, semester.id))),
          API.getPortalTasks(this.selectedClassSlug, '', semester.id),
          API.getPortalMaterials(this.selectedClassSlug, semester.id),
          API.getChanges(this.selectedClassSlug, semester.id)
        ]);
        this.archiveData = {
          schedule: schedules.flatMap(x => x && Array.isArray(x.items) ? x.items : []),
          tasks: tasks || [], materials: materials || [], changes: changes || []
        };
      } catch (err) {
        this.archiveError = err.message || 'Arsip semester belum dapat dimuat.';
      } finally { this.archiveLoading = false; }
    },

    tutupArsipSemester() {
      this.archiveSelected = null;
      this.archiveError = '';
    },

    badgeJadwal(kind) {
      const k = String(kind || '').toUpperCase();
      if (k === 'PENGGANTI') return { label: 'Kelas Pengganti', cls: 'bg-amber-50 text-amber-900 border-amber-300', icon: 'swap_horiz' };
      if (k === 'TAMBAHAN') return { label: 'Kelas Tambahan', cls: 'bg-[#E9EAFF] text-[#3965FB] border-[#3965FB]/30', icon: 'add_circle' };
      if (k === 'LIBUR') return { label: 'Diliburkan', cls: 'bg-red-50 text-red-700 border-red-200', icon: 'event_busy' };
      if (k === 'DIBATALKAN') return { label: 'Sesi Dibatalkan', cls: 'bg-red-50 text-red-700 border-red-200', icon: 'cancel' };
      return { label: 'Pola Jadwal', cls: 'bg-[#E1FFB7] text-green-800 border-green-200', icon: 'event' };
    },

    fmtDeadlineID(iso) { return API.fmtDeadlineID(iso); },
    deadlineBadge(iso) { return API.deadlineBadge(iso); },

    dashboardPartialsLoaded: false,

    async ensureDashboardPartials() {
      if (this.dashboardPartialsLoaded) return;
      await this.loadPartials([
        ['portal-sidebar', '/partials/portal/sidebar.html'],
        ['portal-state', '/partials/common/state-error.html'],
        ['portal-topbar', '/partials/portal/topbar.html'],
        ['portal-dashboard', '/partials/portal/view-dashboard.html'],
        ['portal-tugas', '/partials/portal/view-tugas.html'],
        ['portal-jadwal', '/partials/portal/view-jadwal.html'],
        ['portal-materi', '/partials/portal/view-materi.html'],
        ['portal-courses', '/partials/portal/view-courses.html'],
        ['portal-perubahan', '/partials/portal/view-perubahan.html'],
        ['portal-arsip', '/partials/portal/view-arsip.html'],
        ['portal-drawer', '/partials/portal/drawer.html'],
        ['portal-bottombar', '/partials/portal/bottombar.html']
      ]);
      await this.loadPartials([
        ['portal-skeleton', '/partials/common/skeleton-dashboard.html']
      ]);
      this.dashboardPartialsLoaded = true;
    },

    async initPortal() {
      try { this.sidebarCollapsed = localStorage.getItem('asterisk:sidebar:collapsed') === '1'; } catch (e) {}
      window.addEventListener('offline', () => { this.showPageError('offline'); });
      window.addEventListener('online', () => { if (this.pageState && this.pageState.status === 'offline') window.location.reload(); });
      await this.loadClasses();
      await this.loadPartials([
        ['portal-gate', '/partials/portal/gate.html'],
        ['portal-toast', '/partials/portal/toast.html']
      ]);

      if (!this.unlocked) {
        return;
      }

      const slug = this.selectedClassSlug;
      const token = localStorage.getItem('portal_class') === slug ? localStorage.getItem('portal_token') : '';
      try {
        const res = await fetch('/api/v1/portal/' + encodeURIComponent(slug) + '/summary', {
          credentials: 'same-origin',
          headers: token ? { 'X-Portal-Token': token } : {}
        });
        if (!res.ok) {
          localStorage.removeItem('portal_pin_ok');
          this.unlocked = false;
          this.gateStep = res.status === 401 ? 'pin' : 'select';
          return;
        }
      } catch (err) {
        this.unlocked = false;
        this.showPageError('offline');
        return;
      }

      await this.ensureDashboardPartials();
      this.updateClock();
      const wd = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
      const today = wd[new Date().getDay()];
      this.hari = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat'].includes(today) ? today : 'Senin';
      await Promise.all([
        this.loadSchedule(),
        this.loadJadwalEfektif(),
        this.loadTugasPortal(),
        this.loadMateri(),
        this.loadPerubahan(),
        this.loadSemester()
      ]);
      this.dashboardLoading = false;
    },

    async loadPartials(slots) {
      // Alpine v3 auto-init node baru via MutationObserver; jangan initTree manual (render ganda).
      await Promise.all(slots.map(async ([id, url]) => {
        try {
          const res = await fetch(url + '?v=20261010', { cache: 'no-store' });
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
          timeZone: 'Asia/Jakarta', weekday: 'long', day: 'numeric', month: 'long', year: 'numeric'
        });
        const parts = fmt.formatToParts(new Date());
        const get = (t) => { const p = parts.find(x => x.type === t); return p ? p.value : ''; };
        let day = get('weekday') || 'Senin';
        day = day.charAt(0).toUpperCase() + day.slice(1);
        this.todayName = day;
        this.todayFull = `${day}, ${get('day')} ${get('month')} ${get('year')}`;
      } catch (e) {
        this.todayName = 'Senin';
        this.todayFull = 'Senin';
      }
    },

    get selectedClassSlug() {
      return (this.selectedClass || 'd4-ti-2024-a').toLowerCase().replace(/\s+/g, '-');
    },

    loadingPin: false,

    async masukKelas() {
      if (!this.selectedClass) {
        this.showToast('Pilih kelas terlebih dahulu.');
        return;
      }
      this.checkingAccess = true;
      this.pinError = '';

      try {
        const slug = this.selectedClassSlug;
        const savedToken = localStorage.getItem('portal_token');
        const savedClass = localStorage.getItem('portal_class');
        const tokenToSend = (savedClass === slug && savedToken) ? savedToken : '';

        // Cek apakah kelas ini membutuhkan kode akses v1
        const res = await fetch('/api/v1/portal/' + encodeURIComponent(slug) + '/summary', {
          credentials: 'same-origin',
          headers: tokenToSend ? { 'X-Portal-Token': tokenToSend } : {}
        });

        if (res.ok) {
          // Akses langsung diizinkan (mode LINK atau sesi token valid)
          localStorage.setItem('portal_pin_ok', '1');
          localStorage.setItem('portal_class', slug);
          this.unlocked = true;
          this.gateStep = 'select';
          this.showToast(`Selamat datang di Portal ${this.selectedClass}!`);
          await this.ensureDashboardPartials();
          this.updateClock();
          await Promise.all([this.loadSchedule(), this.loadJadwalEfektif(), this.loadTugasPortal(), this.loadMateri(), this.loadPerubahan(), this.loadSemester()]);
          this.dashboardLoading = false;
          return;
        }

        if (res.status === 401) {
          // Kelas ini memerlukan PIN (mode CODE)
          this.gateStep = 'pin';
          this.pin = '';
          return;
        }

        // Jika kelas tidak dikenal portal (404): jangan buka; tampilkan error jujur.
        // Status lain (500, dsb): masalah server/koneksi, juga jangan buka diam-diam.
        this.showToast(res.status === 404
          ? 'Kelas tidak terdaftar di portal. Periksa kode kelas atau hubungi KM.'
          : 'Portal belum dapat diakses. Periksa koneksi lalu coba lagi.');
        return;
      } catch (err) {
        // Fallback jika offline/error: tetap terkunci agar tidak tampil portal kosong.
        this.showToast('Tidak dapat terhubung ke portal. Periksa koneksi lalu coba lagi.');
        return;
      } finally {
        this.checkingAccess = false;
      }
    },

    kembaliKePilihKelas() {
      this.gateStep = 'select';
      this.pin = '';
      this.pinError = '';
    },

    async bukaPortal() {
      const code = this.pin.trim();
      if (code.length < 6 || code.length > 128) {
        this.pinError = 'Kode akses harus terdiri dari 6 sampai 128 karakter.';
        return;
      }
      this.pinError = '';
      this.loadingPin = true;

      try {
        const slug = this.selectedClassSlug;
        const res = await API.verifyPortalCode(slug, code);

        if (res && res.portal_token) {
          localStorage.setItem('portal_token', res.portal_token);
        }
        localStorage.setItem('portal_pin_ok', '1');
        localStorage.setItem('portal_class', slug);
        localStorage.setItem('portal_pin_at', String(Date.now()));
        this.unlocked = true;
        this.gateStep = 'select';
        this.showToast('Kode akses terverifikasi. Selamat datang di Portal Kelas!');

        await this.ensureDashboardPartials();
        this.updateClock();
        await Promise.all([this.loadSchedule(), this.loadJadwalEfektif(), this.loadTugasPortal(), this.loadMateri(), this.loadPerubahan(), this.loadSemester()]);
        this.dashboardLoading = false;
      } catch (err) {
        this.pinError = err.message || 'Kode akses tidak valid. Periksa kembali kode dari grup kelas.';
        this.showToast(this.pinError);
      } finally {
        this.loadingPin = false;
      }
    },

    kunciPortal() {
      localStorage.removeItem('portal_pin_ok');
      localStorage.removeItem('portal_token');
      localStorage.removeItem('portal_class');
      this.unlocked = false;
      this.gateStep = 'select';
      this.pin = '';
      this.pinError = '';
      this.view = 'dashboard';
    },

    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed;
      try { localStorage.setItem('asterisk:sidebar:collapsed', this.sidebarCollapsed ? '1' : '0'); } catch (e) {}
    },

    knownViews: ['dashboard', 'tugas', 'detail-tugas', 'jadwal', 'courses', 'course-detail', 'materi', 'detail-materi', 'perubahan', 'arsip'],

    showPageError(status) {
      this.pageState = { status: status };
      this.drawer = false;
      this.view = '__error';
      window.scrollTo({ top: 0 });
    },

    async pilihKelas(kelas) {
      if (!kelas || kelas === this.selectedClass) return;
      this.selectedClass = kelas;
      this.syncSegmentsFromClass();
      this.gateStep = 'select';
      this.pin = '';
      this.pinError = '';

      if (this.unlocked) {
        const savedClass = localStorage.getItem('portal_class');
        this.unlocked = (savedClass === this.selectedClassSlug && localStorage.getItem('portal_pin_ok') === '1');
        if (this.unlocked) {
          await this.ensureDashboardPartials();
          this.loadSchedule();
          this.loadJadwalEfektif();
          this.loadTugasPortal();
          this.loadMateri();
          this.loadPerubahan();
          this.loadSemester();
        }
      }

      try {
        const newUrl = '/c/' + this.selectedClassSlug;
        if (window.location.pathname !== newUrl) {
          window.history.replaceState({}, '', newUrl);
        }
      } catch (e) {}
    },

    parseClass(code) {
      if (!code) return { prodi: 'D4', smt: '3', abjad: 'A' };
      const m = String(code).match(/^([A-Za-z0-9]+)-[A-Za-z0-9]+-SMT(\d+)-([A-Za-z0-9]+)$/i);
      if (m) {
        return { prodi: m[1].toUpperCase(), smt: m[2], abjad: m[3].toUpperCase() };
      }
      return { prodi: 'D4', smt: '3', abjad: 'A' };
    },

    get selectedClassMeta() {
      if (!this.selectedClass) return { prodi: 'D4', smt: '3', abjad: 'A', label: 'Belum dipilih' };
      const p = this.parseClass(this.selectedClass);
      const prodiName = p.prodi === 'D4' ? 'D4 Teknik Informatika' : (p.prodi === 'D3' ? 'D3 Teknik Informatika' : p.prodi);
      return {
        ...p,
        prodiName,
        label: `${prodiName} · Semester ${p.smt} · Kelas ${p.abjad}`
      };
    },

    toggleCombobox() {
      this.comboboxOpen = !this.comboboxOpen;
      if (this.comboboxOpen) {
        setTimeout(() => {
          const inp = document.getElementById('search-class-input');
          if (inp) inp.focus();
        }, 50);
      }
    },

    get flatClassGroups() {
      const q = this.classQuery.trim().toLowerCase();
      const groups = [
        {
          title: 'D4 Teknik Informatika',
          prodi: 'D4',
          badge: 'Sarjana Terapan',
          items: this.classList.filter(c => this.parseClass(c).prodi === 'D4')
        },
        {
          title: 'D3 Teknik Informatika',
          prodi: 'D3',
          badge: 'Diploma Tiga',
          items: this.classList.filter(c => this.parseClass(c).prodi === 'D3')
        }
      ];

      return groups.map(g => {
        if (this.classFilterProdi !== 'ALL' && g.prodi !== this.classFilterProdi) {
          return { ...g, items: [] };
        }
        const matched = g.items.filter(c => {
          if (!q) return true;
          const p = this.parseClass(c);
          const searchStr = `${c} ${p.prodi} ${p.smt} ${p.abjad} semester ${p.smt} kelas ${p.abjad} smt${p.smt} ${p.smt}${p.abjad}`.toLowerCase();
          return searchStr.includes(q);
        }).map(c => {
          const p = this.parseClass(c);
          return {
            code: c,
            smt: p.smt,
            abjad: p.abjad,
            label: `Semester ${p.smt} · Kelas ${p.abjad}`
          };
        });
        return { ...g, items: matched };
      }).filter(g => g.items.length > 0);
    },

    get totalFilteredCount() {
      return this.flatClassGroups.reduce((acc, g) => acc + g.items.length, 0);
    },

    selectAndCloseClass(c) {
      this.pilihKelas(c);
      this.comboboxOpen = false;
      this.classQuery = '';
    },

    get availableProdis() {
      const list = this.classList.map(c => this.parseClass(c).prodi);
      const unique = Array.from(new Set(list));
      return unique.length ? unique : ['D4', 'D3'];
    },

    get availableSemesters() {
      const list = this.classList
        .map(c => this.parseClass(c))
        .filter(p => p.prodi === this.selectedProdi)
        .map(p => p.smt);
      const unique = Array.from(new Set(list)).sort((a, b) => Number(a) - Number(b));
      return unique.length ? unique : ['1', '3', '5', '7'];
    },

    get availableAbjads() {
      const list = this.classList
        .map(c => this.parseClass(c))
        .filter(p => p.prodi === this.selectedProdi && p.smt === this.selectedSemester)
        .map(p => p.abjad);
      const unique = Array.from(new Set(list)).sort();
      return unique.length ? unique : ['A', 'B', 'C', 'D'];
    },

    setProdi(prodi) {
      this.selectedProdi = prodi;
      const semList = this.availableSemesters;
      if (!semList.includes(this.selectedSemester)) {
        this.selectedSemester = semList[0] || '1';
      }
      const abjadList = this.availableAbjads;
      if (!abjadList.includes(this.selectedAbjad)) {
        this.selectedAbjad = abjadList[0] || 'A';
      }
      this.syncClassFromSegments();
    },

    setSemester(smt) {
      this.selectedSemester = smt;
      const abjadList = this.availableAbjads;
      if (!abjadList.includes(this.selectedAbjad)) {
        this.selectedAbjad = abjadList[0] || 'A';
      }
      this.syncClassFromSegments();
    },

    setAbjad(abjad) {
      this.selectedAbjad = abjad;
      this.syncClassFromSegments();
    },

    syncClassFromSegments() {
      const targetPattern = new RegExp(`^${this.selectedProdi}-.*-SMT${this.selectedSemester}-${this.selectedAbjad}$`, 'i');
      const found = this.classList.find(c => targetPattern.test(c));
      const targetClass = found || `${this.selectedProdi}-TI-SMT${this.selectedSemester}-${this.selectedAbjad}`;
      this.pilihKelas(targetClass);
    },

    syncSegmentsFromClass() {
      if (!this.selectedClass) return;
      const p = this.parseClass(this.selectedClass);
      this.selectedProdi = p.prodi;
      this.selectedSemester = p.smt;
      this.selectedAbjad = p.abjad;
    },

    go(v) {
      this.pageState = null;
      if (!this.knownViews.includes(v)) { this.showPageError('404'); return; }
      this.view = v;
      this.drawer = false;
      if (v === 'jadwal') this.loadJadwalEfektif();
      if (v === 'courses') this.loadCourses();
      if (v === 'arsip') this.loadSemester();
      window.scrollTo({ top: 0 });
    },

    soon(fitur) { this.showToast(`${fitur}: fitur belum tersedia.`); },

    async loadCourses() {
      this.coursesLoading = true; this.coursesError = '';
      try { this.courses = await API.getPortalCourses(this.selectedClassSlug); }
      catch (e) { this.courses = []; this.coursesError = e.message || 'Mata kuliah gagal dimuat.'; }
      finally { this.coursesLoading = false; }
    },

    bukaCourse(course) { this.selectedCourse = course; this.go('course-detail'); },

    async loadClasses() {
      const pathMatch = window.location.pathname.match(/\/c\/([^/]+)/);
      const urlParams = new URLSearchParams(window.location.search);
      const slugFromUrl = (pathMatch && pathMatch[1]) || urlParams.get('c') || urlParams.get('kelas');

      try {
        const data = await API.getClasses();
        if (data && Array.isArray(data.classes)) {
          this.classList = data.classes;
        }
        if (slugFromUrl) {
          this.selectedClass = decodeURIComponent(slugFromUrl);
        } else if (data && data.default_class) {
          this.selectedClass = data.default_class;
        }
      } catch (e) { /* fallback */ }

      if (!this.selectedClass) {
        try {
          const st = await API.getStatus();
          if (st && st.default_class) this.selectedClass = st.default_class;
          if (st && Array.isArray(st.classes)) this.classList = st.classes;
        } catch (e) { /* kosong */ }
      }

      if (!this.selectedClass && this.classList.length > 0) {
        this.selectedClass = this.classList[0];
      }
      this.syncSegmentsFromClass();

      const savedClass = localStorage.getItem('portal_class');
      if (savedClass && savedClass === this.selectedClassSlug) {
        this.unlocked = localStorage.getItem('portal_pin_ok') === '1';
      } else if (savedClass && savedClass !== this.selectedClassSlug) {
        this.unlocked = false;
      }
    },

    normalisasiEfektif(items, hariLabel) {
      return (items || []).map((it, i) => ({
        id: it.id || `ef-${i}`,
        hari: hariLabel || '',
        kind: it.kind || 'REGULER',
        matkul: it.offering || it.title || 'Mata Kuliah',
        dosen: Array.isArray(it.lecturers) ? it.lecturers.join(', ') : (it.lecturers || ''),
        ruang: it.room || '',
        link: it.meeting_link || '',
        timeStart: (it.starts_at || '').slice(0, 5),
        timeEnd: (it.ends_at || '').slice(0, 5)
      }));
    },

    async loadSchedule() {
      try {
        const raw = await API.getSchedule(this.selectedClass, 'all');
        this.fullSchedule = (raw || []).map((s, i) => {
          const parts = String(s.jam || '').split('-').map(x => x.trim().replace('.', ':'));
          return {
            id: `sch-${i}`, hari: s.hari, jam: s.jam, matkul: s.matkul,
            dosen: s.dosen, ruang: s.ruang,
            timeStart: parts[0] || '', timeEnd: parts[1] || ''
          };
        });
      } catch (e) { this.fullSchedule = []; }
    },

    // Jadwal efektif (pola + perubahan terbit) untuk satu hari.
    async loadJadwalEfektif() {
      this.jadwalLoading = true; this.jadwalError = '';
      const slug = this.selectedClassSlug;
      const tanggal = (this.tanggalPekan && this.tanggalPekan[this.hari]) || '';
      try {
        const data = await API.getPortalSchedule(slug, tanggal);
        if (data && Array.isArray(data.items)) {
          this.jadwalEfektif = this.normalisasiEfektif(data.items, this.hari);
        } else {
          this.jadwalEfektif = this.fullSchedule.filter(s => s.hari === this.hari).map(s => ({
            id: s.id, hari: s.hari, kind: 'REGULER', matkul: s.matkul,
            dosen: s.dosen || '', ruang: s.ruang || '',
            timeStart: s.timeStart || '', timeEnd: s.timeEnd || ''
          }));
        }
      } catch (e) {
        this.jadwalEfektif = [];
        this.jadwalError = 'Jadwal belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.jadwalLoading = false;
      }
      // Cache khusus hari ini untuk dashboard.
      try {
        const tglHariIni = (this.tanggalPekan && this.tanggalPekan[this.todayName]) || '';
        if (this.hari === this.todayName) {
          this.jadwalCacheHariIni = this.jadwalEfektif.slice();
        } else {
          const data = await API.getPortalSchedule(slug, tglHariIni);
          if (data && Array.isArray(data.items)) {
            this.jadwalCacheHariIni = this.normalisasiEfektif(data.items, this.todayName);
          } else {
            this.jadwalCacheHariIni = this.fullSchedule.filter(s => s.hari === this.todayName);
          }
        }
      } catch (e) {
        this.jadwalCacheHariIni = this.fullSchedule.filter(s => s.hari === this.todayName);
      }
    },

    pilihHari(d) {
      this.hari = d;
      this.loadJadwalEfektif();
    },

    async loadTugasPortal() {
      this.tugasLoading = true; this.tugasError = '';
      const slug = this.selectedClassSlug;
      try {
        const grup = ['hari_ini', 'minggu_ini', 'mendatang', 'terlewat'];
        const hasil = await Promise.all(grup.map(g =>
          API.getPortalTasks(slug, g)
            .then(data => ({ ok: true, data: data }))
            .catch(err => ({ ok: false, err: err }))
        ));
        if (hasil.every(h => !h.ok)) {
          const st = hasil[0].err && hasil[0].err.status;
          this.tugasGrup = { hari_ini: [], minggu_ini: [], mendatang: [], terlewat: [] };
          this.tasks = [];
          this.tugasError = st === 404
            ? 'Kelas tidak ditemukan di portal. Periksa kode kelas atau hubungi KM.'
            : st === 401
              ? 'Kelas ini dilindungi kode akses. Kunci portal lalu masukkan kode dari grup kelas.'
              : 'Tugas belum dapat dimuat. Periksa koneksi lalu coba lagi.';
          return;
        }
        const petakan = (arr) => (arr || []).map(t => {
          const deadline = t.deadline_at;
          const u = this.deadlineBadge(deadline);
          return { id: t.id, matkul: t.matkul || t.offering || 'Mata Kuliah',
                   title: t.title || '', deskripsi: t.deskripsi || t.instructions || t.title || '',
                   instructions: t.instructions || '',
                   deadline: this.fmtDeadlineID(deadline), deadline_at: deadline,
                   urgency: u.level, countdown: u.badge };
        });
        this.tugasGrup = {
          hari_ini: petakan(hasil[0].ok ? hasil[0].data : []),
          minggu_ini: petakan(hasil[1].ok ? hasil[1].data : []),
          mendatang: petakan(hasil[2].ok ? hasil[2].data : []),
          terlewat: petakan(hasil[3].ok ? hasil[3].data : [])
        };
        const gabung = new Map();
        [...this.tugasGrup.mendatang, ...this.tugasGrup.minggu_ini, ...this.tugasGrup.hari_ini, ...this.tugasGrup.terlewat]
          .forEach(t => { if (!gabung.has(String(t.id))) gabung.set(String(t.id), t); });
        this.tasks = Array.from(gabung.values());
      } catch (e) {
        this.tugasGrup = { hari_ini: [], minggu_ini: [], mendatang: [], terlewat: [] };
        this.tasks = [];
        this.tugasError = 'Tugas belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.tugasLoading = false;
      }
    },

    async loadTasks() { await this.loadTugasPortal(); },

    async bukaDetailTugas(t) {
      this.detailTugas = { memuat: true };
      this.detailMateri = [];
      this.view = 'detail-tugas';
      window.scrollTo({ top: 0 });
      try {
        const d = await API.getPortalTaskDetail(this.selectedClassSlug, t.id);
        const info = (d && (d.task || d)) || null;
        if (!info) throw { code: 'NOT_FOUND' };
        const u = this.deadlineBadge(info.deadline_at);
        this.detailTugas = {
          id: info.id, matkul: info.offering || '', title: info.title || '',
          instructions: info.instructions || '',
          deadline: this.fmtDeadlineID(info.deadline_at),
          task_type: info.task_type || '',
          submission_text: info.submission_text || '', submission_url: info.submission_url || '',
          version: info.version || '', is_completed: !!info.is_completed,
          urgency: u.level, countdown: u.badge
        };
        this.detailMateri = (d && d.materials) || [];
      } catch (e) {
        if (e && e.code === 'NOT_FOUND') { this.showPageError('404'); return; }
        if (e && e.code === 'LOAD_FAILED') { this.showPageError('500'); return; }
        this.detailTugas = { hilang: true };
        this.detailMateri = [];
      }
    },

    async loadMateri() {
      this.materiLoading = true;
      try {
        this.materiList = await API.getPortalMaterials(this.selectedClassSlug).catch(() => []);
      } catch (e) {
        this.materiList = [];
      } finally {
        this.materiLoading = false;
      }
    },

    get materiGrup() {
      const grup = {};
      (this.materiList || []).forEach(m => {
        const kunci = m.material_type || 'Lainnya';
        if (!grup[kunci]) grup[kunci] = [];
        grup[kunci].push(m);
      });
      return Object.keys(grup).sort().map(k => ({ jenis: k, items: grup[k] }));
    },

    async loadPerubahan() {
      this.perubahanLoading = true;
      try {
        this.perubahanList = await API.getChanges(this.selectedClassSlug).catch(() => null) || [];
      } catch (e) {
        this.perubahanList = [];
      } finally {
        this.perubahanLoading = false;
      }
    },

    async loadSemester() {
      this.semesterLoading = true;
      try {
        this.semesterList = await API.getPortalSemesters(this.selectedClassSlug).catch(() => []);
      } catch (e) {
        this.semesterList = [];
      } finally {
        this.semesterLoading = false;
      }
    },

    labelStatusSemester(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'ACTIVE') return 'Aktif';
      if (s === 'DRAFT') return 'Draf';
      if (s === 'ARCHIVED') return 'Arsip';
      return st || '-';
    },

    labelJenisUbah(kind) {
      const k = String(kind || '').toUpperCase();
      if (k === 'REPLACEMENT') return 'Kelas Pengganti';
      if (k === 'EXTRA') return 'Kelas Tambahan';
      if (k === 'HOLIDAY') return 'Hari Libur';
      if (k === 'SESSION_CANCELLED') return 'Sesi Dibatalkan';
      return kind || '-';
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

    showToast(msg) {
      if (this.toast.timer) clearTimeout(this.toast.timer);
      this.toast.message = msg;
      this.toast.show = true;
      this.toast.timer = setTimeout(() => { this.toast.show = false; }, 3000);
    }
  };
}
