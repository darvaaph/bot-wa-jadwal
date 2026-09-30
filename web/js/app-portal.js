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
      await this.loadClasses();
      await this.loadSchedule();
      await this.loadTasks();
    },

    async loadPartials(slots) {
      await Promise.all(slots.map(async ([id, url]) => {
        try {
          const res = await fetch(url + '?v=20260927d', { cache: 'no-store' });
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
        localStorage.setItem('portal_pin_at', String(Date.now()));
        this.unlocked = true;
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
      this.unlocked = false;
      this.pin = '';
      this.view = 'dashboard';
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
      try {
        const data = await API.getClasses();
        if (data && data.default_class) { this.selectedClass = data.default_class; return; }
      } catch (e) { /* fallback */ }
      try {
        const st = await API.getStatus();
        if (st && st.default_class) this.selectedClass = st.default_class;
      } catch (e) { /* kosong */ }
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
        const raw = await API.getTasks(this.selectedClass);
        this.tasks = (raw || []).map(t => {
          const u = this.urgencyOf(t.deadline);
          return { id: t.id, matkul: t.matkul, deskripsi: t.deskripsi,
                   deadline: t.deadline, urgency: u.level, countdown: u.badge };
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
