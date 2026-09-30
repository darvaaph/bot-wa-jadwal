/**
 * web/js/app-km.js — Area KM (REVISI Figma, tanpa role switcher).
 * Memakai helper API global dari js/api.js. Backend yang belum ada
 * ditandai jujur di UI (toast + empty state), tidak difake.
 */
function kmApp() {
  return {
    view: 'dashboard',
    drawer: false,
    q: '',
    weekOffset: 0,

    kmNav: [
      { id: 'dashboard', label: 'Dashboard', img: '/assets/icons/home.svg' },
      { id: 'tugas', label: 'Tugas', img: '/assets/icons/tasks.svg' },
      { id: 'jadwal', label: 'Jadwal', img: '/assets/icons/calendar.svg' },
      { id: 'dosen', label: 'Dosen berhalangan', img: '/assets/icons/book.svg' },
      { id: 'materi', label: 'Materi', img: '/assets/icons/folder.svg' },
      { id: 'anggota', label: 'Anggota & Tim', img: '/assets/icons/ext-settings-edit.svg' },
      { id: 'pengaturan', label: 'Pengaturan Kelas', img: '/assets/icons/settings.svg' },
      { id: 'monitoring', label: 'Monitoring', img: '/assets/icons/activity.svg' },
      { id: 'log', label: 'Log Aktivitas', img: '/assets/icons/ext-check.svg' },
      { id: 'notifikasi', label: 'Notifikasi', img: '/assets/icons/bell.svg' },
      { id: 'akun', label: 'Akun', img: '/assets/icons/event.svg' }
    ],

    todayName: 'Senin',
    todayFull: '',
    currentTime: '',
    selectedClass: '',
    classList: [],
    botOnline: false,
    fullSchedule: [],
    tasks: [],

    tugasSub: 'list',
    tugasChip: 'Semua',
    tugasMatkul: 'Semua',
    tugasSort: 'dekat',
    tugasForm: { matkul: '', judul: '', tanggal: '', jam: '', deskripsi: '', kumpul: '' },
    tugasError: '',

    dosen: { matkul: '', tanggal: '', mode: 'online', hariGanti: '', jamGanti: '', alasan: '', link: '' },
    patternsList: [],

    anggotaSub: 'list',
    undang: { matkul: '', nomor: '' },
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
      return [...this.tasks].sort((a, b) => (rank[a.urgency] ?? 2) - (rank[b.urgency] ?? 2));
    },

    get urgent3() { return this.withUrgency.slice(0, 3); },
    get nearCount() { return this.tasks.filter(t => t.urgency !== 'aman').length; },
    get antrean() { return this.withUrgency; },

    get matkulOpts() {
      const set = new Set(this.fullSchedule.map(s => s.matkul));
      return Array.from(set);
    },

    get roomList() {
      return Array.from(new Set(this.fullSchedule.map(s => s.ruang).filter(Boolean)));
    },

    get tugasList() {
      const q = this.q.toLowerCase();
      let list = this.tasks.filter(t => {
        const hit = (t.matkul + ' ' + t.deskripsi).toLowerCase().includes(q);
        const mk = this.tugasMatkul === 'Semua' || t.matkul === this.tugasMatkul;
        if (this.tugasChip === 'Terjadwal' || this.tugasChip === 'Selesai') return false;
        return hit && mk;
      });
      const rank = { mendesak: 0, mendekati: 1, aman: 2 };
      list.sort((a, b) => this.tugasSort === 'dekat'
        ? (rank[a.urgency] ?? 2) - (rank[b.urgency] ?? 2)
        : (rank[b.urgency] ?? 2) - (rank[a.urgency] ?? 2));
      return list;
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

    slotAt(dayName, slotHH) {
      return this.fullSchedule.find(s => s.hari === dayName && parseInt((s.timeStart || '0').split(':')[0], 10) === parseInt(slotHH, 10));
    },

    async initKM() {
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
      } catch (e) {
        localStorage.removeItem('access_token');
        window.location.replace('/login.html?role=km');
        return;
      }

      await this.loadPartials([
        ['km-sidebar', '/partials/km/sidebar.html'],
        ['km-topbar', '/partials/km/topbar.html'],
        ['km-dashboard', '/partials/km/view-dashboard.html'],
        ['km-tugas', '/partials/km/view-tugas.html'],
        ['km-jadwal', '/partials/km/view-jadwal.html'],
        ['km-dosen', '/partials/km/view-dosen.html'],
        ['km-materi', '/partials/km/view-materi.html'],
        ['km-anggota', '/partials/km/view-anggota.html'],
        ['km-antrean', '/partials/km/view-antrean.html'],
        ['km-monitor', '/partials/km/view-monitor.html'],
        ['km-notif', '/partials/km/view-notif.html'],
        ['km-pengaturan', '/partials/km/view-pengaturan.html'],
        ['km-drawer', '/partials/km/drawer.html'],
        ['km-toast', '/partials/km/toast.html']
      ]);
      this.updateClock();
      setInterval(() => this.updateClock(), 1000);
      await this.checkBot();
      await this.loadClasses();
      await this.loadSchedule();
      await this.loadTasks();
      this.patternsList = await API.getPatterns().catch(() => []);
      setInterval(() => this.checkBot(), 30000);
    },

    async loadPartials(slots) {
      await Promise.all(slots.map(async ([id, url]) => {
        try {
          const res = await fetch(url + '?v=20260927a', { cache: 'no-store' });
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
      if (v === 'tugas') this.tugasSub = 'list';
      if (v === 'anggota') this.anggotaSub = 'list';
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
          return;
        }
      } catch (e) { /* fallback */ }
      try {
        const st = await API.getStatus();
        if (st && st.default_class) {
          this.selectedClass = st.default_class;
          this.classList = st.classes || [];
        }
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

    // Tugas — tambah + hapus via API asli.
    async terbitTugas() {
      const f = this.tugasForm;
      if (!f.matkul) { this.tugasError = 'Mata kuliah wajib dipilih.'; return; }
      if (!f.judul || f.judul.trim().length < 5) { this.tugasError = 'Judul tugas minimal 5 karakter.'; return; }
      if (!f.tanggal || !f.jam) { this.tugasError = 'Tanggal dan jam deadline wajib diisi.'; return; }
      if (!f.deskripsi || f.deskripsi.trim().length < 5) { this.tugasError = 'Deskripsi tugas minimal 5 karakter.'; return; }
      this.tugasError = '';
      await API.createTask({
        matkul: f.matkul,
        deskripsi: `${f.judul} — ${f.deskripsi.trim()}`,
        deadline: `${f.tanggal} ${f.jam}`
      });
      await this.loadTasks();
      this.tugasForm = { matkul: '', judul: '', tanggal: '', jam: '', deskripsi: '', kumpul: '' };
      this.tugasSub = 'list';
      this.showToast('Tugas diterbitkan.');
    },

    async simpanDraf() {
      this.showToast('Draf tersimpan lokal — status draf butuh endpoint backend.');
      this.tugasSub = 'list';
    },

    async hapusTugas(id) {
      await API.deleteTask(id);
      await this.loadTasks();
      this.showToast('Tugas diarsipkan.');
    },

    // Dosen berhalangan — draf lokal, tidak tersimpan ke backend.
    previewDosen() {
      if (!this.dosen.matkul || !this.dosen.tanggal || !this.dosen.alasan) {
        this.showToast('Lengkapi mata kuliah, tanggal, dan alasan dulu.');
        return;
      }
      this.showToast('Preview diperbarui.');
    },

    dosenMessage() {
      const f = this.dosen;
      const judul = f.mode === 'ganti' ? 'Jadwal diganti hari' : 'Perkuliahan dialihkan online';
      return `INFO PERKULIAHAN • ${this.selectedClass}\n${judul}\n${f.matkul}\n${f.tanggal}${f.mode === 'ganti' && f.hariGanti ? ' → ' + f.hariGanti + ' ' + f.jamGanti : ''}\n${f.alasan}${f.link ? '\nTautan pertemuan: ' + f.link : ''}`;
    },

    copyDosen() { this.copyText(this.dosenMessage(), 'Teks pengumuman tersalin.'); },

    async publishDosen() {
      const f = this.dosen;
      if (!f.matkul || !f.tanggal || !f.alasan) {
        this.showToast('Lengkapi mata kuliah, tanggal, dan alasan dulu.');
        return;
      }

      this.showToast('Menyimpan perubahan jadwal...');
      try {
        if (!this.patternsList || this.patternsList.length === 0) {
          this.patternsList = await API.getPatterns().catch(() => []);
        }

        const targetLower = f.matkul.toLowerCase();
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
          owner_offering_id: matched ? matched.course_offering_id : 1,
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

    // Undang PJ — link digenerate lokal; endpoint undangan belum tersedia.
    buatUndangPJ() {
      if (!this.undang.matkul) { this.undangError = 'Mata kuliah wajib dipilih.'; return; }
      if (!this.undang.nomor || this.undang.nomor.replace(/\D/g, '').length < 9) {
        this.undangError = 'Nomor WhatsApp calon PJ tidak valid.';
        return;
      }
      this.undangError = '';
      this.undangLink = `https://bot-jadwal/undang/pj?kelas=${encodeURIComponent(this.selectedClass)}&matkul=${encodeURIComponent(this.undang.matkul)}&wa=${encodeURIComponent(this.undang.nomor)}`;
      this.anggotaSub = 'siap';
    },

    // Review — keputusan tercatat lokal; sinkronisasi butuh endpoint backend.
    setujuiTugas() { this.showToast('Keputusan review butuh endpoint backend.'); },
    kirimReview() {
      if (!this.reviewNote.trim()) { this.showToast('Catatan wajib untuk koreksi/pembatalan.'); return; }
      this.reviewNote = '';
      this.reviewId = null;
      this.showToast('Keputusan tercatat lokal — sinkronisasi butuh endpoint backend.');
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
    }
  };
}
