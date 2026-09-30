/**
 * web/js/app-portal.js — Portal mahasiswa (REVISI Figma): hanya-baca, tanpa akun.
 * Gerbang PIN disimpan lokal 30 hari sebagai pratinjau; verifikasi butuh endpoint backend.
 */
function portalApp() {
  return {
    unlocked: localStorage.getItem('portal_pin_ok') === '1',
    pin: '',
    pinError: '',
    view: 'dashboard',
    drawer: false,
    q: '',
    hari: 'Senin',
    tugasChip: 'Semua',
    tugasMatkul: 'Semua',

    portalNav: [
      { id: 'dashboard', label: 'Dashboard', img: '/assets/icons/home.svg' },
      { id: 'tugas', label: 'Tugas', img: '/assets/icons/tasks.svg' },
      { id: 'jadwal', label: 'Jadwal', img: '/assets/icons/calendar.svg' },
      { id: 'materi', label: 'Materi', img: '/assets/icons/folder.svg' },
      { id: 'notifikasi', label: 'Notifikasi', img: '/assets/icons/bell.svg' }
    ],

    todayName: 'Senin',
    todayFull: '',
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
    tasks: [],
    detailTugas: {},

    toast: { show: false, message: '', timer: null },

    get todayList() {
      return this.fullSchedule.filter(s => s.hari === this.todayName);
    },

    get withUrgency() {
      const rank = { mendesak: 0, mendekati: 1, aman: 2 };
      return [...this.tasks].sort((a, b) => (rank[a.urgency] ?? 2) - (rank[b.urgency] ?? 2));
    },

    get urgent3() { return this.withUrgency.slice(0, 3); },
    get nearCount() { return this.tasks.filter(t => t.urgency !== 'aman').length; },

    get matkulOpts() {
      return Array.from(new Set(this.fullSchedule.map(s => s.matkul)));
    },

    get tugasList() {
      const q = this.q.toLowerCase();
      return this.withUrgency.filter(t => {
        const hit = (t.matkul + ' ' + t.deskripsi).toLowerCase().includes(q);
        const mk = this.tugasMatkul === 'Semua' || t.matkul === this.tugasMatkul;
        if (this.tugasChip === 'Terjadwal' || this.tugasChip === 'Selesai') return false;
        return hit && mk;
      });
    },

    get jadwalHari() {
      return this.fullSchedule.filter(s => s.hari === this.hari);
    },

    async initPortal() {
      await this.loadClasses();
      await this.loadPartials([
        ['portal-gate', '/partials/portal/gate.html'],
        ['portal-sidebar', '/partials/portal/sidebar.html'],
        ['portal-topbar', '/partials/portal/topbar.html'],
        ['portal-dashboard', '/partials/portal/view-dashboard.html'],
        ['portal-tugas', '/partials/portal/view-tugas.html'],
        ['portal-jadwal', '/partials/portal/view-jadwal.html'],
        ['portal-materi', '/partials/portal/view-materi.html'],
        ['portal-notif', '/partials/portal/view-notif.html'],
        ['portal-drawer', '/partials/portal/drawer.html'],
        ['portal-toast', '/partials/portal/toast.html']
      ]);
      this.updateClock();
      const wd = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
      const today = wd[new Date().getDay()];
      this.hari = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat'].includes(today) ? today : 'Senin';
      await this.loadSchedule();
      await this.loadTasks();
    },

    async loadPartials(slots) {
      await Promise.all(slots.map(async ([id, url]) => {
        try {
          const res = await fetch(url + '?v=20261002', { cache: 'no-store' });
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
          await Promise.all([this.loadSchedule(), this.loadTasks()]);
          return;
        }

        if (res.status === 401) {
          // Kelas ini memerlukan PIN (mode CODE)
          this.gateStep = 'pin';
          this.pin = '';
          return;
        }

        // Jika status 404 (kelas berbasis data jadwal reguler), langsung masuk ke portal
        localStorage.setItem('portal_pin_ok', '1');
        localStorage.setItem('portal_class', slug);
        this.unlocked = true;
        this.gateStep = 'select';
        await Promise.all([this.loadSchedule(), this.loadTasks()]);
      } catch (err) {
        // Fallback jika offline/error
        localStorage.setItem('portal_pin_ok', '1');
        localStorage.setItem('portal_class', this.selectedClassSlug);
        this.unlocked = true;
        this.gateStep = 'select';
        await Promise.all([this.loadSchedule(), this.loadTasks()]);
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
      if (!/^\d{6}$/.test(code)) {
        this.pinError = 'Kode akses harus 6 digit angka.';
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

        await Promise.all([this.loadSchedule(), this.loadTasks()]);
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

    pilihKelas(kelas) {
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
          this.loadSchedule();
          this.loadTasks();
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
      this.view = v;
      this.drawer = false;
      window.scrollTo({ top: 0 });
    },

    soon(fitur) { this.showToast(`${fitur}: fitur belum tersedia.`); },

    bukaDetailTugas(t) {
      this.detailTugas = t;
      this.view = 'detail-tugas';
      window.scrollTo({ top: 0 });
    },

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

    async loadTasks() {
      try {
        const slug = this.selectedClassSlug;
        const portal = await API.getPortalTasks(slug).catch(() => null);
        const raw = portal && portal.length ? portal : await API.getTasks(this.selectedClass);
        this.tasks = (raw || []).map(t => {
          const deadline = t.deadline_at || t.deadline;
          const u = this.urgencyOf(deadline);
          return { id: t.id,
                   matkul: t.course_name || t.matkul,
                   deskripsi: t.title ? (t.title + (t.instructions ? ' — ' + t.instructions : '')) : t.deskripsi,
                   deadline: deadline, urgency: u.level, countdown: u.badge };
        });
      } catch (e) { this.tasks = []; }
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
