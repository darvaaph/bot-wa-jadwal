/**
 * web/js/app-pj.js — Area PJ (REVISI Figma).
 * Cakupan PJ: hanya mata kuliah yang ditugaskan (dipilih lokal, disimpan
 * di perangkat sampai endpoint penugasan backend ada).
 */
function pjApp() {
  return {
    view: 'dashboard',
    drawer: false,
    q: '',
    weekOffset: 0,
    pjMatkul: localStorage.getItem('pj_matkul') || '',
    offeringId: localStorage.getItem('pj_offering_id') || '',
    offeringList: [],
    offeringLoading: false,

    roleLabel: 'PJ',

    navSections: [
      { title: 'PJ', items: [
        { id: 'dashboard', label: 'Dashboard', img: '/assets/icons/home.svg' },
      ] },
      { title: 'AKADEMIK', items: [
        { id: 'tugas', label: 'Tugas', img: '/assets/icons/tasks.svg', active: ['tugas', 'tambah', 'preview', 'konfirmasi', 'terbit'] },
        { id: 'jadwal', label: 'Jadwal', img: '/assets/icons/calendar.svg', active: ['jadwal', 'pindah', 'perubahan'] },
        { id: 'dosen', label: 'Dosen berhalangan', img: '/assets/icons/book.svg' },
        { id: 'materi', label: 'Materi', img: '/assets/icons/folder.svg' },
      ] },
      { title: 'LAINNYA', items: [
        { id: 'status', label: 'Status pemeriksaan', img: '/assets/icons/ext-check.svg' },
        { id: 'notifikasi', label: 'Notifikasi', img: '/assets/icons/bell.svg' },
        { id: 'pengaturan', label: 'Pengaturan', img: '/assets/icons/settings.svg' },
        { id: 'akun', label: 'Akun', img: '/assets/icons/event.svg' },
      ] },
    ],

    get nav() { return this.navSections.flatMap(s => s.items); },
    get pjNav() { return this.nav; },
    get roleSub() { return this.pjMatkul || 'Mata kuliah belum dipilih'; },

    isActive(item) { const a = item.active || [item.id]; return a.includes(this.view); },

    todayName: 'Senin',
    todayFull: '',
    currentTime: '',
    selectedClass: '',
    botOnline: false,
    fullSchedule: [],
    tasks: [],

    tugasForm: { judul: '', tanggal: '', jam: '', deskripsi: '', kumpul: '' },
    tugasError: '',
    terbitOk: false,

    dosen: { tanggal: '', jam: '', mode: 'online', hariGanti: '', jamGanti: '', alasan: '', link: '' },

    pengaturan: {
      pagi: localStorage.getItem('pj_rem_pagi') || '06:00',
      sore: localStorage.getItem('pj_rem_sore') || '17:00'
    },
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
      return [...this.tasks].sort((a, b) => (rank[a.urgency] ?? 2) - (rank[b.urgency] ?? 2));
    },

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
      return `${fmt(days[0].full)}–${fmt(days[4].full)} ${days[4].full.getFullYear()}`;
    },

    slotSaya(dayName, slotHH) {
      return this.scopeList.find(s => s.hari === dayName && parseInt((s.timeStart || '0').split(':')[0], 10) === parseInt(slotHH, 10));
    },

    async initPJ() {
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
        ['pj-topbar', '/partials/common/topbar.html'],
        ['pj-dashboard', '/partials/pj/view-dashboard.html'],
        ['pj-tugas', '/partials/pj/view-tugas.html'],
        ['pj-jadwal', '/partials/pj/view-jadwal.html'],
        ['pj-dosen', '/partials/pj/view-dosen.html'],
        ['pj-status', '/partials/pj/view-status.html'],
        ['pj-notif', '/partials/pj/view-notif.html'],
        ['pj-akun', '/partials/pj/view-akun.html'],
        ['pj-drawer', '/partials/common/drawer.html'],
        ['pj-toast', '/partials/common/toast.html']
      ]);
      this.updateClock();
      setInterval(() => this.updateClock(), 1000);
      await this.checkBot();
      await this.loadClasses();
      await this.loadOfferings();
      await this.loadSchedule();
      await this.loadTasks();
      this.patternsList = await API.getPatterns().catch(() => []);
      setInterval(() => this.checkBot(), 30000);
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
      this.view = v;
      this.drawer = false;
      window.scrollTo({ top: 0 });
    },

    soon(fitur) { this.showToast(`${fitur}: fitur belum tersedia.`); },

    simpanMatkul() {
      localStorage.setItem('pj_matkul', this.pjMatkul);
      this.showToast(this.pjMatkul ? `Cakupan: ${this.pjMatkul}` : 'Cakupan dikosongkan.');
    },

    async loadOfferings() {
      this.offeringLoading = true;
      try {
        if (!this.selectedClass) return;
        const semesters = await API.getSemesters(this.selectedClass).catch(() => []);
        const list = Array.isArray(semesters) ? semesters : [];
        const active = list.find(s => s.status === 'ACTIVE') || list[0];
        if (!active) { this.offeringList = []; return; }
        const offerings = await API.getSemesterOfferings(active.id).catch(() => []);
        this.offeringList = Array.isArray(offerings) ? offerings : [];
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
      } finally {
        this.offeringLoading = false;
      }
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
        const raw = await API.getTasks(this.offeringId || '');
        const list = (raw || []).map(t => {
          const u = this.urgencyOf(t.deadline_at || t.deadline);
          return { id: t.id, matkul: t.matkul, deskripsi: t.deskripsi,
                   deadline: t.deadline, title: t.title, version: t.version,
                   urgency: u.level, countdown: u.badge };
        });
        if (this.pjMatkul) {
          this.tasks = list.filter(t => !t.matkul || t.matkul === this.pjMatkul);
          if (this.tasks.length === 0) this.tasks = list;
        } else {
          this.tasks = list;
        }
      } catch (e) { this.tasks = []; }
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
      const hh = String(hm[1]).padStart(2, '0');
      return `${datePart}T${hh}:${hm[2]}:00+07:00`;
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

    mulaiTambah() {
      if (!this.offeringId) { this.showToast('Pilih mata kuliah (offering) yang ditugaskan dulu di Dashboard.'); return; }
      this.tugasForm = { judul: '', tanggal: '', jam: '', deskripsi: '', kumpul: '' };
      this.tugasError = '';
      this.terbitOk = false;
      this.view = 'tambah';
      window.scrollTo({ top: 0 });
    },

    async terbitTugas() {
      const f = this.tugasForm;
      if (!this.offeringId) { this.tugasError = 'Pilih mata kuliah (offering) di Dashboard dulu.'; return; }
      if (!f.judul || f.judul.trim().length < 5) { this.tugasError = 'Judul tugas minimal 5 karakter.'; return; }
      if (!f.tanggal || !f.jam) { this.tugasError = 'Tanggal dan jam deadline wajib diisi.'; return; }
      if (!f.deskripsi || f.deskripsi.trim().length < 5) { this.tugasError = 'Deskripsi tugas minimal 5 karakter.'; return; }
      const deadlineAt = this.parseDeadlineID(f.tanggal, f.jam);
      if (!deadlineAt) { this.tugasError = 'Format tanggal atau jam tidak dikenali. Pakai YYYY-MM-DD dan HH:MM.'; return; }
      this.tugasError = '';
      try {
        await API.createTask({
          offering_id: Number(this.offeringId),
          title: f.judul.trim(),
          instructions: f.deskripsi.trim(),
          deadline_at: deadlineAt,
          submission_text: (f.kumpul || '').trim() || undefined,
          save_as: 'published'
        });
      } catch (err) {
        this.tugasError = err.message || 'Gagal menerbitkan tugas di server.';
        return;
      }
      await this.loadTasks();
      this.terbitOk = true;
      this.showToast('Tugas diterbitkan.');
      this.view = 'tugas';
    },

    async simpanDrafTugas() {
      const f = this.tugasForm || {};
      if (!this.offeringId) { this.showToast('Pilih mata kuliah (offering) di Dashboard dulu.'); return; }
      if (!f.judul || f.judul.trim().length < 5) { this.showToast('Judul tugas minimal 5 karakter.'); return; }
      try {
        const payload = {
          offering_id: Number(this.offeringId),
          title: f.judul.trim(),
          instructions: (f.deskripsi || '').trim(),
          save_as: 'draft'
        };
        const deadlineAt = (f.tanggal && f.jam) ? this.parseDeadlineID(f.tanggal, f.jam) : '';
        if (deadlineAt) payload.deadline_at = deadlineAt;
        if ((f.kumpul || '').trim()) payload.submission_text = f.kumpul.trim();
        await API.createTask(payload);
        await this.loadTasks();
        this.showToast('Draf tersimpan di server.');
        this.view = 'tugas';
      } catch (err) {
        this.showToast(err.message || 'Gagal menyimpan draf di server.');
      }
    },

    async hapusTugas(id) {
      try {
        const detail = await API.getTaskDetail(id).catch(() => null);
        const version = detail && (detail.version || (detail.task && detail.task.version));
        await API.deleteTask(id, version);
        await this.loadTasks();
        this.showToast('Tugas diarsipkan.');
      } catch (err) {
        this.showToast(err.message || 'Gagal mengarsipkan tugas.');
      }
    },

    async lihatTugas(t) {
      try {
        const detail = await API.getTaskDetail(t.id);
        const info = detail && (detail.task || detail);
        this.showToast(info && info.title ? `Tugas: ${info.title}` : 'Detail tugas dimuat.');
      } catch (e) {
        this.showToast('Gagal memuat detail tugas dari server.');
      }
    },

    previewDosen() {
      if (!this.dosen.tanggal || !this.dosen.alasan) {
        this.showToast('Lengkapi tanggal dan alasan dulu.');
        return;
      }
      this.showToast('Preview diperbarui.');
    },

    dosenMessage() {
      const f = this.dosen;
      const judul = f.mode === 'ganti' ? 'Jadwal diganti hari' : 'Perkuliahan dialihkan online';
      return `INFO PERKULIAHAN • ${this.selectedClass}\n${judul}\n${this.pjMatkul}\n${f.tanggal}${f.mode === 'ganti' && f.hariGanti ? ' → ' + f.hariGanti + ' ' + f.jamGanti : ''}\n${f.alasan}${f.link ? '\nTautan pertemuan: ' + f.link : ''}`;
    },

    copyDosen() { this.copyText(this.dosenMessage(), 'Teks pengumuman tersalin.'); },

    async publishDosen() {
      const f = this.dosen;
      const matkulTarget = this.offeringName() || this.pjMatkul || '';
      if (!this.offeringId || !matkulTarget || !f.tanggal || !f.alasan) {
        this.showToast('Pilih offering, lengkapi tanggal dan alasan dulu.');
        return;
      }

      this.showToast('Menyimpan perubahan jadwal...');
      try {
        if (!this.patternsList || this.patternsList.length === 0) {
          this.patternsList = await API.getPatterns().catch(() => []);
        }

        const targetLower = matkulTarget.toLowerCase();
        const matched = (this.patternsList || []).find(p => {
          const name = String(p.display_name || p.course_name || '').toLowerCase();
          return name.includes(targetLower) || targetLower.includes(name);
        });

        // Parse tanggal ke format ISO YYYY-MM-DD
        let dateIso = new Date().toISOString().slice(0, 10);
        const dm = String(f.tanggal).match(/(\d{1,2})\s+([A-Za-z]+)\s*(\d{4})?/);
        if (/^\d{4}-\d{2}-\d{2}$/.test(String(f.tanggal).trim())) {
          dateIso = String(f.tanggal).trim();
        } else if (dm) {
          const months = { jan:'01',feb:'02',mar:'03',apr:'04',mei:'05',jun:'06',jul:'07',agu:'08',sep:'09',okt:'10',nov:'11',des:'12' };
          const mKey = dm[2].toLowerCase().slice(0, 3);
          const month = months[mKey] || '01';
          const year = dm[3] || new Date().getFullYear();
          dateIso = `${year}-${month}-${String(dm[1]).padStart(2, '0')}`;
        }

        // Parse rentang waktu
        const parts = String(f.jamGanti || f.jam || '08:00 - 09:40').split('-').map(x => x.trim().replace('.', ':'));
        let startH = parts[0] || '08:00';
        let endH = parts[1] || '09:40';
        if (startH.length === 4) startH = '0' + startH;
        if (endH.length === 4) endH = '0' + endH;

        const startsAt = `${dateIso}T${startH}:00+07:00`;
        const endsAt = `${dateIso}T${endH}:00+07:00`;
        const reasonText = (f.alasan || 'Dosen berhalangan') + (f.link ? ' | Tautan: ' + f.link : '');
        const kind = (f.mode === 'ganti' && matched) ? 'REPLACEMENT' : 'EXTRA';

        const payload = {
          owner_offering_id: Number(this.offeringId),
          event_kind: kind,
          starts_at: startsAt,
          ends_at: endsAt,
          reason: reasonText
        };
        if (kind === 'REPLACEMENT' && matched) {
          payload.origin_pattern_id = matched.id;
          payload.origin_date = dateIso;
        }
        if (matched && matched.room_id) {
          payload.room_id = matched.room_id;
        }

        const draft = await API.createTeachingEventDraft(payload);
        if (draft && draft.id) {
          await API.publishTeachingEvent(draft.id).catch(() => null);
        }

        this.copyText(this.dosenMessage(), 'Tersimpan ke database & pengumuman WhatsApp tersalin!');
        await this.loadSchedule();
      } catch (err) {
        this.copyText(this.dosenMessage(), 'Pengumuman tersalin. (Status DB: ' + (err.message || 'Koneksi lokal') + ')');
      }
    },

    simpanPengaturan() {
      localStorage.setItem('pj_rem_pagi', this.pengaturan.pagi);
      localStorage.setItem('pj_rem_sore', this.pengaturan.sore);
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
    }
  };
}
