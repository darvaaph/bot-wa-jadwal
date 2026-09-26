/**
 * web/js/app.js — Alpine.js Dashboard (tema Figma Asterisk)
 * Shell + Dashboard + Jadwal + Dosen berhalangan wired API asli.
 * AREA KEMAL: tab Tugas (lihat marker KEMAL di index.html).
 */

function dashboardApp() {
  return {
    activeView: 'dashboard',
    drawerOpen: false,
    sidebarCollapsed: localStorage.getItem('asterisk_sidebar') === 'collapsed',
    searchQuery: '',

    navItems: [
      { id: 'dashboard', label: 'Dashboard', icon: 'home', img: '/assets/icons/home.svg' },
      { id: 'monitoring', label: 'Monitoring', icon: 'monitoring', img: '/assets/icons/activity.svg' },
      { id: 'tugas', label: 'Tugas', icon: 'assignment', img: '/assets/icons/tasks.svg' },
      { id: 'jadwal', label: 'Jadwal', icon: 'calendar_month', img: '/assets/icons/calendar.svg' },
      { id: 'dosen', label: 'Dosen berhalangan', icon: 'book', img: '/assets/icons/book.svg' },
      { id: 'materi', label: 'Materi', icon: 'folder', img: '/assets/icons/folder.svg' },
      { id: 'notifikasi', label: 'Notifikasi', icon: 'notifications', img: '/assets/icons/bell.svg' },
      { id: 'event', label: 'Event', icon: 'event', img: '/assets/icons/event.svg' },
      { id: 'pengaturan', label: 'Pengaturan', icon: 'settings', img: '/assets/icons/settings.svg' },
      { id: 'profil', label: 'Profil', icon: 'person', img: '' }
    ],

    // Clock & date (WIB)
    currentTime: '00:00:00 WIB',
    todayName: 'Senin',
    todayFull: '',

    // Backend state
    selectedClass: '',
    classList: [],
    botOnline: false,
    fullSchedule: [],
    tasks: [],

    // Jadwal week nav
    weekOffset: 0,

    // Dosen berhalangan form
    dosenForm: { matkul: '', tanggal: '', jam: '', mode: 'online', alasan: '', link: '' },

    toast: { show: false, message: '', timer: null },

    // ---------- Computed ----------
    get todaySchedule() {
      return this.fullSchedule.filter(s => s.hari === this.todayName);
    },

    get urgentTasks() {
      const rank = { mendesak: 0, mendekati: 1, aman: 2 };
      return [...this.tasks]
        .sort((a, b) => (rank[a.urgency] ?? 2) - (rank[b.urgency] ?? 2))
        .slice(0, 3);
    },

    get urgentCount() {
      return this.tasks.filter(t => t.urgency !== 'aman').length;
    },

    get matkulList() {
      const set = new Set(this.fullSchedule.map(s => s.matkul));
      return Array.from(set);
    },

    get roomList() {
      const set = new Set(this.fullSchedule.map(s => s.ruang).filter(Boolean));
      return Array.from(set);
    },

    // Senin–Jumat minggu berjalan (+offset), tanggal real
    get weekDays() {
      const names = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat'];
      const now = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
      const dow = (now.getDay() + 6) % 7; // Senin=0
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
      const year = days[4].full.getFullYear();
      return `${fmt(days[0].full)}–${fmt(days[4].full)} ${year}`;
    },

    viewTitle(id) {
      const item = this.navItems.find(n => n.id === id);
      return item ? item.label : id;
    },

    sessionAt(dayName, slotHH) {
      return this.fullSchedule.find(s => s.hari === dayName && parseInt((s.timeStart || '0').split(':')[0], 10) === parseInt(slotHH, 10));
    },

    // ---------- Init ----------
    async initDashboard() {
      await this.loadPartials();
      this.updateClock();
      setInterval(() => this.updateClock(), 1000);
      await this.checkBotHealth();
      await this.loadClasses();
      await this.loadSchedule();
      await this.loadTasks();
      setInterval(() => this.checkBotHealth(), 30000);
    },

    // Muat partials HTML per bagian lalu daftarkan ke Alpine.
    // ASSET_V: naikkan tiap ada perubahan partial agar browser tidak pakai cache lama.
    async loadPartials() {
      const ASSET_V = '20260926c';
      const slots = [
        ['slot-sidebar', '/partials/sidebar.html'],
        ['slot-topbar', '/partials/topbar.html'],
        ['slot-dashboard', '/partials/view-dashboard.html'],
        ['slot-jadwal', '/partials/view-jadwal.html'],
        ['slot-dosen', '/partials/view-dosen.html'],
        ['slot-tugas', '/partials/view-tugas.html'],
        ['slot-soon', '/partials/view-soon.html'],
        ['slot-drawer', '/partials/drawer.html'],
        ['slot-toast', '/partials/toast.html']
      ];
      await Promise.all(slots.map(async ([id, url]) => {
        try {
          const res = await fetch(url + '?v=' + ASSET_V, { cache: 'no-store' });
          if (!res.ok) throw new Error(`HTTP ${res.status}`);
          const html = await res.text();
          const el = document.getElementById(id);
          if (el) {
            el.innerHTML = html;
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
          timeZone: 'Asia/Jakarta', weekday: 'long', day: 'numeric',
          month: 'long', year: 'numeric',
          hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false
        });
        const parts = fmt.formatToParts(new Date());
        const get = (t) => { const p = parts.find(x => x.type === t); return p ? p.value : ''; };
        let day = get('weekday') || 'Senin';
        day = day.charAt(0).toUpperCase() + day.slice(1);
        this.todayName = day;
        this.todayFull = `${day}, ${get('day')} ${get('month')} ${get('year')}`;
        this.currentTime = `${get('hour')}:${get('minute')}:${get('second')} WIB`;
      } catch (e) {
        const now = new Date();
        const map = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
        this.todayName = map[now.getDay()];
        this.todayFull = this.todayName;
        const pad = (n) => String(n).padStart(2, '0');
        this.currentTime = `${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())} WIB`;
      }
    },

    switchView(v) {
      this.activeView = v;
      this.drawerOpen = false;
      window.scrollTo({ top: 0 });
    },

    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed;
      try {
        localStorage.setItem('asterisk_sidebar', this.sidebarCollapsed ? 'collapsed' : 'open');
      } catch (e) { /* abaikan */ }
    },

    soon(fitur) {
      this.showToast(`${fitur}: fitur belum tersedia.`);
    },

    searchGo() {
      // Query dishare ke tab Tugas (milik Kemal) — cukup pindah view.
      this.switchView('tugas');
    },

    shiftWeek(n) {
      this.weekOffset += n;
    },

    // ---------- API ----------
    async checkBotHealth() {
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
          return;
        }
      } catch (e) { /* fallback di bawah */ }
      try {
        const st = await API.getStatus();
        if (st && st.default_class) {
          this.selectedClass = st.default_class;
          this.classList = st.classes || [];
        }
      } catch (e) { /* tetap kosong */ }
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
        if (diffH < 24) return { level: 'mendesak', badge: diffH < 1 ? 'Besok' : `Sisa ${Math.ceil(diffH)} Jam` };
        if (diffH <= 72) return { level: 'mendekati', badge: `H-${Math.ceil(diffH / 24)}` };
        return { level: 'aman', badge: 'Aktif' };
      } catch (e) {
        return { level: 'aman', badge: 'Aktif' };
      }
    },

    // ---------- Dosen berhalangan (draf lokal + salin; publish butuh endpoint BE) ----------
    previewDosen() {
      if (!this.dosenForm.matkul || !this.dosenForm.tanggal || !this.dosenForm.alasan) {
        this.showToast('Lengkapi mata kuliah, tanggal, dan alasan dulu.');
        return;
      }
      this.showToast('Preview diperbarui — cek panel kanan.');
    },

    dosenMessage() {
      const f = this.dosenForm;
      const judul = f.mode === 'ganti' ? 'Jadwal diganti hari' : 'Perkuliahan dialihkan online';
      return `INFO PERKULIAHAN • ${this.selectedClass}\n${judul}\n${f.matkul}\n${f.tanggal}${f.jam ? ' • ' + f.jam + ' WIB' : ''}\n${f.alasan}${f.link ? '\nTautan pertemuan: ' + f.link : ''}`;
    },

    copyDosen() {
      this.copyText(this.dosenMessage(), 'Teks pengumuman tersalin.');
    },

    publishDosen() {
      if (!this.dosenForm.matkul || !this.dosenForm.tanggal || !this.dosenForm.alasan) {
        this.showToast('Lengkapi mata kuliah, tanggal, dan alasan dulu.');
        return;
      }
      this.copyText(this.dosenMessage(), 'Tersalin — publish permanen butuh endpoint backend.');
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
