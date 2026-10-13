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
    botOnline: false,
    contextAssignments: [],
    contextSwitching: false,
    meCache: null,
    hari: 'Senin',
    tugasMatkul: 'Semua',

    ...AsteriskShell.behavior('portal'),

    navSections: [
      {
        title: '', items: [
          { id: 'dashboard', label: 'Beranda', icon: 'home' },
          { id: 'jadwal', label: 'Jadwal Kuliah', icon: 'calendar_month' },
          { id: 'tugas', label: 'Tugas', icon: 'assignment', active: ['tugas', 'detail-tugas'] },
          { id: 'materi', label: 'Materi', icon: 'folder' }
        ]
      }
    ],

    portalNavIcon(item) {
      const key = item && item.icon;
      const open = '<svg width="19" height="19" viewBox="0 0 14 14" fill="currentColor" class="w-[19px] h-[19px] shrink-0" aria-hidden="true" focusable="false">';
      const paths = {
        'home': '<path d="M3.5 11.08447l2.15332 0 0-2.96338q0-0.19824 0.1333-0.33496 0.13672-0.13672 0.33838-0.13672l1.75 0q0.20166 0 0.33496 0.13672 0.13672 0.13672 0.13672 0.33496l0 2.96338 2.15332 0 0-5.07226q0-0.08887-0.04102-0.16065-0.0376-0.0752-0.10595-0.12988l-3.14112-2.36865q-0.08887-0.07861-0.21191-0.07862-0.12305 0-0.21191 0.07862l-3.14112 2.36865q-0.06836 0.05469-0.10937 0.12988-0.0376 0.07178-0.0376 0.16065l0 5.07226z m-0.58447 0l0-5.07226q0-0.22217 0.09912-0.42041 0.10254-0.20166 0.28027-0.33155l3.14112-2.3789q0.24609-0.18799 0.56054-0.18799 0.31787 0 0.56738 0.18799l3.14112 2.3789q0.17773 0.12988 0.27685 0.33155 0.10254 0.19824 0.10254 0.42041l0 5.07226q0 0.23242-0.17431 0.40674-0.17432 0.17432-0.41016 0.17432l-2.26611 0q-0.20166 0-0.33838-0.1333-0.1333-0.13672-0.1333-0.33838l0-2.95996-1.52442 0 0 2.95996q0 0.20166-0.13672 0.33838-0.1333 0.1333-0.33496 0.1333l-2.26611 0q-0.23584 0-0.41016-0.17432-0.17432-0.17432-0.17431-0.40674z"/>',
        'calendar_month': '<path d="M3.27441 12.25q-0.3999 0-0.66992-0.27002-0.27002-0.27002-0.27002-0.67334l0-7.44775q0-0.40332 0.27002-0.67334 0.27002-0.27002 0.66992-0.27002l1.03223 0 0-0.98438q0-0.13672 0.08887-0.22558 0.08887-0.08887 0.22558-0.08887 0.13672 0 0.22559 0.08887 0.08887 0.08887 0.08887 0.22558l0 0.98438 4.17334 0 0-1.0083q0-0.12305 0.08203-0.20508 0.08545-0.08545 0.20849-0.08545 0.12646 0 0.2085 0.08545 0.08545 0.08203 0.08545 0.20508l0 1.0083 1.03223 0q0.3999 0 0.66992 0.27002 0.27002 0.27002 0.27002 0.67334l0 7.44775q0 0.40332-0.27002 0.67334-0.27002 0.27002-0.66992 0.27002l-7.45118 0z m0-0.58447l7.45118 0q0.1333 0 0.24609-0.10938 0.11279-0.11279 0.11279-0.24951l0-5.11328-8.16894 0 0 5.11328q0 0.13672 0.11279 0.24951 0.11279 0.10938 0.24609 0.10938z m-0.35888-6.05664l8.16894 0 0-1.75q0-0.1333-0.11279-0.2461-0.11279-0.11279-0.24609-0.11279l-7.45118 0q-0.1333 0-0.24609 0.11279-0.11279 0.11279-0.11279 0.2461l0 1.75z m0 0l0-1.75q0-0.1333 0-0.2461 0-0.11279 0-0.11279 0 0 0 0.11279 0 0.11279 0 0.2461l0 1.75z m4.08447 2.64892q-0.18115 0-0.31445-0.1333-0.1333-0.13672-0.1333-0.31787 0-0.18115 0.1333-0.31445 0.1333-0.1333 0.31445-0.1333 0.18115 0 0.31445 0.1333 0.1333 0.1333 0.1333 0.31445 0 0.18115-0.1333 0.31787-0.1333 0.1333-0.31445 0.1333z m-2.33447 0q-0.18115 0-0.31446-0.1333-0.1333-0.13672-0.1333-0.31787 0-0.18115 0.1333-0.31445 0.1333-0.1333 0.31446-0.1333 0.18115 0 0.31445 0.1333 0.13672 0.1333 0.1333 0.31445 0 0.18115-0.13672 0.31787-0.1333 0.1333-0.31445 0.1333z m4.66894 0q-0.18115 0-0.31787-0.1333-0.1333-0.13672-0.1333-0.31787 0-0.18115 0.1333-0.31445 0.13672-0.1333 0.31787-0.1333 0.18115 0 0.31446 0.1333 0.1333 0.1333 0.1333 0.31445 0 0.18115-0.1333 0.31787-0.1333 0.1333-0.31446 0.1333z m-2.33447 2.24219q-0.18115 0-0.31445-0.1333-0.1333-0.1333-0.31445-0.1333-0.18115 0-0.31446 0.1333-0.1333 0.13672-0.31445 0.13672 0.18115 0 0.31445 0.13672 0.1333 0.1333 0.31446 0 0.18115-0.1333 0.31445-0.1333 0.31445-0.1333z m-2.33447 0q-0.18115 0-0.31446-0.1333-0.1333-0.1333-0.31445 0-0.18115 0.1333-0.31446 0.1333-0.13672 0.31446-0.13672 0.18115 0 0.31445 0.13672 0.13672 0.1333 0.13672 0.31446 0 0.18115-0.13672 0.31445-0.1333 0.1333-0.31445 0.1333z m4.66894 0q-0.18115 0-0.31787-0.1333-0.1333-0.1333-0.31445 0-0.18115 0.1333-0.31446 0.13672-0.13672 0.31787-0.13672 0.18115 0 0.31446 0.1333 0.1333 0.1333 0.31446 0 0.18115-0.1333 0.31445-0.1333 0.1333-0.31446 0.1333z"/>',
        'assignment': '<path d="M3.27441 11.66553q-0.38623 0-0.66308-0.27686-0.27686-0.27686-0.27686-0.66308l0-7.45118q0-0.38623 0.27686-0.66308 0.27686-0.27686 0.66308-0.27686l2.74121 0q-0.07861-0.44775 0.21534-0.80664 0.29395-0.3623 0.77588-0.3623 0.47852 0 0.77246 0.3623 0.29395 0.35889 0.20507 0.80664l2.74122 0q0.38623 0 0.66308 0.27686 0.27686 0.27686 0.66308 0.66308l0 7.45118q0 0.38623-0.27686 0.66308-0.27686 0.27686-0.66308 0.27686l-7.45118 0z m0-0.58106l7.45118 0q0.1333 0 0.24609-0.11279 0.11279-0.11279 0.11279-0.24609l0-7.45118q0-0.1333-0.11279-0.24609-0.11279-0.11279-0.24609-0.11279l-7.45118 0q-0.1333 0-0.24609 0.11279-0.11279 0.11279-0.11279 0.24609l0 7.45118q0 0.1333 0.11279 0.24609 0.11279 0.11279 0.24609 0.11279z m1.39112-1.59277l2.91894 0q0.12305 0 0.20508-0.08545 0.08545-0.08545 0.08545-0.2085 0-0.12305-0.08545-0.20507-0.08203-0.08545-0.20508-0.08545l-2.91894 0q-0.12305 0-0.2085 0.08545-0.08203 0.08203-0.08203 0.20507 0 0.12646 0.08203 0.21192 0.08545 0.08203 0.2085 0.08203z m0-2.20117l4.66894 0q0.12305 0 0.20508-0.08203 0.08545-0.08545 0.08545-0.2085 0-0.12305-0.08545-0.20508-0.08203-0.08545-0.20508-0.08545l-4.66894 0q-0.12305 0-0.2085 0.08545-0.08203 0.08203-0.08203 0.20508 0 0.12305 0.08203 0.2085 0.08545 0.08203 0.2085 0.08203z m0-2.19776l4.66894 0q0.12305 0 0.20508-0.08203 0.08545-0.08545 0.08545-0.20849 0-0.12646-0.08545-0.2085-0.08203-0.08545-0.20508-0.08545l-4.66894 0q-0.12305 0-0.2085 0.08545-0.08203 0.08203-0.08203 0.2085 0 0.12305 0.08203 0.20849 0.08545 0.08203 0.2085 0.08203z m2.33447-2.50195q0.18799 0 0.31104-0.12305 0.12646-0.12305 0.12646-0.31445 0-0.18799-0.12646-0.31103-0.12305-0.12646-0.31104-0.12647-0.18799 0-0.31445 0.12647-0.12305 0.12305-0.12305 0.31103 0 0.19141 0.12305 0.31445 0.12646 0.12305 0.31445 0.12305z m-4.08447 8.49365q0 0 0-0.11279 0-0.11279 0-0.24609l0-7.45118q0-0.1333 0-0.24609 0-0.11279 0-0.11279 0 0 0 0.11279 0 0.11279 0 0.24609l0 7.45118q0 0.1333 0 0.24609 0 0.11279 0 0.11279z"/>',
        'folder': '<path d="M2.69336 11.08447q-0.40332 0-0.67334-0.27002-0.27002-0.27002-0.27002-0.67334l0-6.28222q0-0.40332 0.27002-0.67334 0.27002-0.27002 0.67334-0.27002l2.51221 0q0.18799 0 0.36572 0.07861 0.17773 0.0752 0.3042 0.20166l0.88867 0.88867 4.54248 0q0.40332 0 0.67334 0.27002 0.27002 0.27002 0.27002 0.66992l0 5.1167q0 0.40332-0.27002 0.67334-0.27002 0.27002-0.67334 0.27002l-8.61328 0z m0-0.58447l8.61328 0q0.15723 0 0.25635-0.09912 0.10254-0.10254 0.10254-0.25977l0-5.1167q0-0.15723-0.10254-0.25634-0.09912-0.10254-0.25635-0.10254l-4.77832 0-1.06299-1.06299q-0.05811-0.05811-0.11963-0.07861-0.06152-0.02393-0.12988-0.02393l-2.52246 0q-0.15723 0-0.25977 0.10254-0.09912 0.09912-0.09912 0.25635l0 6.28222q0 0.15723 0.09912 0.25977 0.10254 0.09912 0.25977 0.09912z m-0.35889 0q0 0 0-0.09912 0-0.10254 0-0.25977l0-6.28222q0-0.15723 0-0.25635 0-0.10254 0-0.10254 0 0 0 0.02393 0 0.02051 0 0.07861l0 1.06299q0 0 0 0.10254 0 0.09912 0 0.25634l0 5.1167q0 0.15723 0 0.25977 0 0.09912 0 0.09912z"/>'
      };
      return open + (paths[key] || '') + '</svg>';
    },

    todayName: 'Senin',
    todayFull: '',
    semesterLabel: '',
    selectedClass: '',
    selectedProdi: 'D4',
    selectedSemester: '3',
    selectedAbjad: 'A',
    classList: [],
    classModalOpen: false,
    comboboxOpen: false,
    classQuery: '',
    classFilterProdi: 'ALL',
    gateStep: 'pin',
    checkingAccess: false,
    fullSchedule: [],
    jadwalEfektif: [],
    jadwalCacheHariIni: [],
    jadwalPekan: { Senin: [], Selasa: [], Rabu: [], Kamis: [], Jumat: [], Sabtu: [] },
    weekOffset: 0,
    modeTampilan: 'kalender',
    mobileJadwalMode: 'per_hari',
    detailModalOpen: false,
    detailSesi: null,
    modalRiwayatOpen: false,
    pekanHariListDesktop: ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat'],
    pekanHariListMobile: ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'],
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
    filterMateriMatkul: 'SEMUA',
    sortMateri: 'terbaru',
    detailMateriModalOpen: false,
    selectedMateri: null,
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

    // Tanggal ISO tiap hari Senin–Minggu pada pekan terpilih (berdasarkan weekOffset, zona WIB).
    get tanggalPekan() {
      const out = {};
      try {
        const names = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu'];
        const nowWib = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
        const dow = (nowWib.getDay() + 6) % 7;
        const monday = new Date(nowWib);
        monday.setDate(nowWib.getDate() - dow + (this.weekOffset * 7));
        names.forEach((n, i) => {
          const d = new Date(monday);
          d.setDate(monday.getDate() + i);
          const pad = (x) => String(x).padStart(2, '0');
          out[n] = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
        });
      } catch (e) { }
      return out;
    },

    get tanggalPekanFormatted() {
      const out = {};
      const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
      const names = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu'];
      names.forEach(n => {
        const iso = this.tanggalPekan[n];
        if (!iso) { out[n] = ''; return; }
        const parts = iso.split('-');
        const day = parseInt(parts[2], 10);
        const m = parseInt(parts[1], 10) - 1;
        out[n] = `${day} ${months[m] || ''}`;
      });
      return out;
    },

    get pekanNum() {
      try {
        if (this.semesterList && this.semesterList.length) {
          const active = this.semesterList.find(s => String(s.status || '').toUpperCase() === 'ACTIVE');
          if (active && active.starts_on) {
            const start = new Date(active.starts_on);
            const now = new Date();
            const diffDays = Math.floor((now - start) / (24 * 3600 * 1000));
            const w = Math.floor(diffDays / 7) + 1;
            if (w >= 1 && w <= 20) return w;
          }
        }
      } catch (e) { }
      return 6; // fallback sesuai Figma: Pekan Ke-6
    },

    get pekanAktifNum() {
      return Math.max(1, this.pekanNum + (this.weekOffset || 0));
    },

    get rentangPekanLabel() {
      try {
        const sen = this.tanggalPekan['Senin'];
        const min = this.tanggalPekan['Minggu'];
        if (!sen || !min) return 'Pekan Ini';
        const d1 = new Date(sen + 'T00:00:00+07:00');
        const d2 = new Date(min + 'T00:00:00+07:00');
        const mNames = ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember'];
        if (d1.getMonth() === d2.getMonth() && d1.getFullYear() === d2.getFullYear()) {
          return `${d1.getDate()} – ${d2.getDate()} ${mNames[d1.getMonth()]} ${d1.getFullYear()}`;
        } else if (d1.getFullYear() === d2.getFullYear()) {
          return `${d1.getDate()} ${mNames[d1.getMonth()]} – ${d2.getDate()} ${mNames[d2.getMonth()]} ${d2.getFullYear()}`;
        }
        return `${d1.getDate()} ${mNames[d1.getMonth()]} ${d1.getFullYear()} – ${d2.getDate()} ${mNames[d2.getMonth()]} ${d2.getFullYear()}`;
      } catch (e) {
        return 'Pekan Ini';
      }
    },

    get rentangPekanSingkatLabel() {
      try {
        const sen = this.tanggalPekan['Senin'];
        const min = this.tanggalPekan['Minggu'];
        if (!sen || !min) return `Pekan ${this.pekanAktifNum}`;
        const d1 = new Date(sen + 'T00:00:00+07:00');
        const d2 = new Date(min + 'T00:00:00+07:00');
        const mShort = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
        if (d1.getMonth() === d2.getMonth()) {
          return `${d1.getDate()} - ${d2.getDate()} ${mShort[d1.getMonth()]} (Pekan ${this.pekanAktifNum})`;
        }
        return `${d1.getDate()} ${mShort[d1.getMonth()]} - ${d2.getDate()} ${mShort[d2.getMonth()]} (Pekan ${this.pekanAktifNum})`;
      } catch (e) {
        return `Pekan ${this.pekanAktifNum}`;
      }
    },

    get tanggalHariTerpilihFormat() {
      try {
        const iso = this.tanggalPekan[this.hari];
        if (!iso) return this.hari;
        const d = new Date(iso + 'T00:00:00+07:00');
        const mNames = ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember'];
        return `${this.hari}, ${d.getDate()} ${mNames[d.getMonth()]} ${d.getFullYear()}`;
      } catch (e) {
        return this.hari;
      }
    },

    get activeTasksCount() {
      return (this.tasks || []).length || 2;
    },

    get hasSaturdayClass() {
      return Array.isArray(this.jadwalPekan['Sabtu']) && this.jadwalPekan['Sabtu'].length > 0;
    },

    get pekanHariListAgenda() {
      if (this.hasSaturdayClass) {
        return ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
      }
      return ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat'];
    },

    formatTanggalLengkap(h) {
      try {
        const iso = this.tanggalPekan[h];
        if (!iso) return h;
        const d = new Date(iso + 'T00:00:00+07:00');
        const mNames = ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember'];
        return `${h}, ${d.getDate()} ${mNames[d.getMonth()]} ${d.getFullYear()}`;
      } catch (e) {
        return h;
      }
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
      return list.filter(t => (this.tugasMatkul === 'Semua' || t.matkul === this.tugasMatkul)
        && this.shellMatchesSearch('tugas', [t.title, t.deskripsi, t.instructions, t.matkul]));
    },

    get jadwalHari() {
      return (this.jadwalEfektif || []).filter(s => this.shellMatchesSearch('jadwal', [s.matkul, s.dosen, s.ruang])).sort((a, b) =>
        String(a.timeStart || '').localeCompare(String(b.timeStart || '')));
    },

    getSesiLiveStatus(s, dayName) {
      if (!s) return { label: 'Akan Datang', class: 'bg-[#EFF6FF] text-[#1D4ED8]' };
      const now = new Date();
      const todayIndo = now.toLocaleDateString('id-ID', { weekday: 'long', timeZone: 'Asia/Jakarta' });
      const hari = s.hari || dayName || this.hari;
      if (hari !== todayIndo || (this.weekOffset !== undefined && this.weekOffset !== 0)) {
        return { label: 'Akan Datang', class: 'bg-[#EFF6FF] text-[#1D4ED8]' };
      }
      const timeParts = new Intl.DateTimeFormat('en-GB', {
        timeZone: 'Asia/Jakarta',
        hour: 'numeric',
        minute: 'numeric',
        hour12: false
      }).formatToParts(now);
      const h = parseInt(timeParts.find(p => p.type === 'hour')?.value || '0', 10);
      const m = parseInt(timeParts.find(p => p.type === 'minute')?.value || '0', 10);
      const nowMin = h * 60 + m;

      const toMins = (t) => {
        if (!t) return -1;
        const [hh, mm] = String(t).replace('.', ':').split(':').map(Number);
        return (isNaN(hh) || isNaN(mm)) ? -1 : hh * 60 + mm;
      };

      const start = toMins(s.timeStart);
      const end = toMins(s.timeEnd);
      if (start >= 0 && end >= 0) {
        if (nowMin >= start && nowMin <= end) {
          return { label: '● Sedang Berlangsung Saat Ini', class: 'bg-[#DCFCE7] text-[#15803D]' };
        } else if (nowMin > end) {
          return { label: 'Selesai Hari Ini', class: 'bg-[#F1F5F9] text-[#64748B]' };
        }
      }
      return { label: 'Akan Datang Hari Ini', class: 'bg-[#EFF6FF] text-[#1D4ED8]' };
    },

    isSesiOngoing(s, dayName) {
      if (!s) return false;
      const now = new Date();
      const todayIndo = now.toLocaleDateString('id-ID', { weekday: 'long', timeZone: 'Asia/Jakarta' });
      const hari = s.hari || dayName || this.hari;
      if (hari !== todayIndo || (this.weekOffset !== undefined && this.weekOffset !== 0)) {
        return false;
      }
      const timeParts = new Intl.DateTimeFormat('en-GB', {
        timeZone: 'Asia/Jakarta',
        hour: 'numeric',
        minute: 'numeric',
        hour12: false
      }).formatToParts(now);
      const h = parseInt(timeParts.find(p => p.type === 'hour')?.value || '0', 10);
      const m = parseInt(timeParts.find(p => p.type === 'minute')?.value || '0', 10);
      const nowMin = h * 60 + m;

      const toMins = (t) => {
        if (!t) return -1;
        const [hh, mm] = String(t).replace('.', ':').split(':').map(Number);
        return (isNaN(hh) || isNaN(mm)) ? -1 : hh * 60 + mm;
      };

      const start = toMins(s.timeStart);
      const end = toMins(s.timeEnd);
      if (start < 0 || end < 0) return false;
      return nowMin >= start && nowMin <= end;
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
      if (k === 'TAMBAHAN') return { label: 'Kelas Tambahan', cls: 'bg-primary-soft text-primary border-primary/30', icon: 'add_circle' };
      if (k === 'LIBUR') return { label: 'Diliburkan', cls: 'bg-red-50 text-red-700 border-red-200', icon: 'event_busy' };
      if (k === 'DIBATALKAN') return { label: 'Sesi Dibatalkan', cls: 'bg-red-50 text-red-700 border-red-200', icon: 'cancel' };
      return { label: 'Pola Jadwal', cls: 'bg-success text-green-800 border-green-200', icon: 'event' };
    },

    fmtDeadlineID(iso) { return API.fmtDeadlineID(iso); },
    deadlineBadge(iso) { return API.deadlineBadge(iso); },

    dashboardPartialsLoaded: false,

    async ensureDashboardPartials() {
      if (this.dashboardPartialsLoaded) return;
      await this.loadPartials([
        ['portal-dashboard', '/partials/portal/view-dashboard.html'],
        ['portal-tugas', '/partials/portal/view-tugas.html'],
        ['portal-jadwal', '/partials/portal/view-jadwal.html'],
        ['portal-materi', '/partials/portal/view-materi.html'],
        ['portal-courses', '/partials/portal/view-courses.html'],
        ['portal-perubahan', '/partials/portal/view-perubahan.html'],
        ['portal-arsip', '/partials/portal/view-arsip.html']
      ]);
      this.dashboardPartialsLoaded = true;
    },

    async initPortal() {
      try { this.sidebarCollapsed = localStorage.getItem('asterisk:sidebar:collapsed') === '1'; } catch (e) { }
      window.addEventListener('offline', () => { this.showPageError('offline'); });
      window.addEventListener('online', () => { if (this.pageState && this.pageState.status === 'offline') window.location.reload(); });
      await this.loadClasses();
      await this.loadPartials([['portal-gate', '/partials/portal/gate.html']]);
      await AsteriskShell.mount('portal', ['dashboard', 'tugas', 'jadwal', 'materi', 'courses', 'perubahan', 'arsip']);

      await this.masukKelas();
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
          this.gateStep = 'pin';
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
      this.classModalOpen = true;
    },

    async pilihKelasModal(cls) {
      this.selectedClass = cls;
      this.pin = '';
      this.pinError = '';
      this.classModalOpen = false;
      const slug = (cls || '').toLowerCase().replace(/\s+/g, '-');
      try {
        window.history.replaceState({}, '', '/c/' + encodeURIComponent(slug));
      } catch (e) { }
      await this.masukKelas();
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
        this.gateStep = 'pin';
        this.showToast('Kode akses terverifikasi. Selamat datang di Portal Kelas!');

        await this.ensureDashboardPartials();
        this.updateClock();
        await Promise.all([this.loadSchedule(), this.loadJadwalEfektif(), this.loadTugasPortal(), this.loadMateri(), this.loadPerubahan(), this.loadSemester()]);
        this.dashboardLoading = false;
      } catch (err) {
        if (err && err.code === 'RATE_LIMITED') {
          this.pinError = 'Terlalu banyak percobaan salah. Silakan tunggu 15 menit.';
        } else {
          this.pinError = 'Kode akses salah atau telah diperbarui oleh Ketua Murid. Silakan minta kode terbaru di grup kelas.';
        }
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
      this.gateStep = 'pin';
      this.pin = '';
      this.pinError = '';
      this.view = 'dashboard';
    },

    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed;
      try { localStorage.setItem('asterisk:sidebar:collapsed', this.sidebarCollapsed ? '1' : '0'); } catch (e) { }
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
      this.gateStep = 'pin';
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
      } catch (e) { }
    },

    parseClass(code) {
      if (!code) return { prodi: 'D4', smt: '3', abjad: 'A' };
      const str = String(code).trim();
      const m = str.match(/^([A-Za-z0-9]+)-[A-Za-z0-9]+-SMT(\d+)-([A-Za-z0-9]+)$/i);
      if (m) {
        return { prodi: m[1].toUpperCase(), smt: m[2], abjad: m[3].toUpperCase() };
      }
      const mYear = str.match(/^([A-Za-z0-9]+)-[A-Za-z0-9]+-(\d{4})-([A-Za-z0-9]+)$/i);
      if (mYear) {
        return { prodi: mYear[1].toUpperCase(), smt: '1', abjad: mYear[3].toUpperCase() };
      }
      const mShort = str.match(/^([A-Za-z0-9]+)-[A-Za-z0-9]+-(\d+)([A-Za-z0-9]+)$/i);
      if (mShort) {
        return { prodi: mShort[1].toUpperCase(), smt: mShort[2], abjad: mShort[3].toUpperCase() };
      }
      return { prodi: 'D4', smt: '3', abjad: 'A' };
    },

    get selectedClassFormatted() {
      if (!this.selectedClass) return 'D4-TI 1A';
      const raw = String(this.selectedClass).trim();
      const mYear = raw.match(/^([A-Za-z0-9]+)-([A-Za-z0-9]+)-(\d{4})-([A-Za-z0-9]+)$/i);
      if (mYear) {
        return `${mYear[1].toUpperCase()}-${mYear[2].toUpperCase()} 1${mYear[4].toUpperCase()}`;
      }
      const mSmt = raw.match(/^([A-Za-z0-9]+)-([A-Za-z0-9]+)-SMT(\d+)-([A-Za-z0-9]+)$/i);
      if (mSmt) {
        return `${mSmt[1].toUpperCase()}-${mSmt[2].toUpperCase()} ${mSmt[3]}${mSmt[4].toUpperCase()}`;
      }
      const mShort = raw.match(/^([A-Za-z0-9]+)-([A-Za-z0-9]+)-([0-9]+[A-Za-z]+)$/i);
      if (mShort) {
        return `${mShort[1].toUpperCase()}-${mShort[2].toUpperCase()} ${mShort[3].toUpperCase()}`;
      }
      if (/^[A-Za-z0-9]+-[A-Za-z0-9]+\s+[0-9A-Za-z]+$/i.test(raw)) {
        return raw.toUpperCase();
      }
      return raw;
    },

    get selectedClassMeta() {
      if (!this.selectedClass) return { prodi: 'D4', smt: '3', abjad: 'A', label: 'D4-TI 1A' };
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
      if (v === 'perubahan') this.loadPerubahan();
      if (v === 'materi') this.loadMateri();
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
          const match = (this.classList || []).find(c => c.toLowerCase().replace(/\s+/g, '-') === slugFromUrl.toLowerCase());
          this.selectedClass = match || decodeURIComponent(slugFromUrl);
        } else if (data && data.default_class) {
          this.selectedClass = data.default_class;
        }
      } catch (e) { /* fallback */ }

      this.gateStep = 'pin';

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
        activityType: String(it.activity_type || 'THEORY').toUpperCase(),
        matkul: it.offering || it.title || 'Mata Kuliah',
        code: it.code || '',
        dosen: Array.isArray(it.lecturers) ? it.lecturers.join(', ') : (it.lecturers || ''),
        ruang: it.room || '',
        link: it.meeting_link || '',
        timeStart: (it.starts_at || '').slice(0, 5),
        timeEnd: (it.ends_at || '').slice(0, 5),
        pj: it.pj || '',
        pjPhone: it.pj_phone || '',
        originDate: it.origin_occurrence_date || '',
        reason: it.reason || ''
      }));
    },

    fmtAsalPengganti(isoDate, reason) {
      if (isoDate) {
        try {
          const d = new Date(isoDate + 'T00:00:00+07:00');
          const days = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
          const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
          return `Pengganti sesi ${days[d.getDay()]}, ${d.getDate()} ${months[d.getMonth()]} ${d.getFullYear()}`;
        } catch (e) { }
      }
      if (reason) return reason;
      return 'Kuliah pengganti resmi';
    },

    getInisial(name) {
      if (!name) return '—';
      const clean = String(name).replace(/^(bpk\.?|ibu\.?|dr\.?|dra\.?|drs\.?)\s+/i, '').trim();
      const parts = clean.split(/[\s,]+/).filter(Boolean);
      if (parts.length === 0) return '—';
      if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
      return (parts[0][0] + parts[1][0]).toUpperCase();
    },

    get tugasTerkaitDetailSesi() {
      if (!this.detailSesi || !this.detailSesi.matkul) return [];
      const target = this.detailSesi.matkul.toLowerCase().trim();
      return (this.tasks || []).filter(t => {
        if (!t || !t.matkul) return false;
        const tm = t.matkul.toLowerCase().trim();
        return tm === target || target.includes(tm) || tm.includes(target);
      });
    },

    chatWhatsAppPJ(s) {
      if (!s) return;
      const raw = s.pjPhone || '';
      const clean = raw.replace(/[^0-9]/g, '');
      if (clean && clean.length >= 9) {
        const text = encodeURIComponent(`Halo PJ ${s.matkul}, saya mahasiswa ingin menanyakan perihal perkuliahan.`);
        window.open(`https://wa.me/${clean}?text=${text}`, '_blank', 'noopener,noreferrer');
      } else {
        this.showToast(`Kontak PJ ${s.pj ? '(' + s.pj + ')' : ''} dapat dihubungi melalui grup WhatsApp kelas.`);
      }
    },

    salinInfoSesi(s) {
      if (!s) return;
      const text = `📅 *Jadwal Perkuliahan: ${s.matkul}*\n` +
        `🗓️ Hari: ${this.formatTanggalLengkap(s.hari)}\n` +
        `⏰ Waktu: ${s.timeStart} – ${s.timeEnd} WIB\n` +
        `🏢 Ruangan: ${s.ruang ? 'Ruang ' + s.ruang : (s.link ? 'Kuliah Daring Zoom' : '—')}\n` +
        `👨‍🏫 Dosen: ${s.dosen || '—'}\n` +
        `👤 PJ Matkul: ${s.pj || 'Belum ditugaskan'}` +
        (s.link ? `\n🔗 Link Zoom: ${s.link}` : '') +
        (s.kind === 'PENGGANTI' ? `\n⚠️ Keterangan: ${this.fmtAsalPengganti(s.originDate, s.reason)}` : '');
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => {
          this.showToast('Informasi sesi berhasil disalin ke clipboard!');
        }).catch(() => {
          this.showToast('Gagal menyalin informasi sesi.');
        });
      } else {
        this.showToast('Informasi sesi berhasil disalin.');
      }
    },

    async prevWeek() {
      this.weekOffset--;
      await this.loadJadwalPekan();
    },

    async nextWeek() {
      this.weekOffset++;
      await this.loadJadwalPekan();
    },

    async mingguIni() {
      this.weekOffset = 0;
      this.hari = this.todayName;
      await this.loadJadwalPekan();
    },

    lihatJadwalBesok() {
      const days = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
      const curIdx = days.indexOf(this.hari);
      const nextIdx = (curIdx + 1) % days.length;
      if (nextIdx === 0) {
        this.weekOffset++;
        this.hari = 'Senin';
        this.loadJadwalPekan();
      } else {
        this.pilihHari(days[nextIdx]);
      }
    },

    bukaDetailSesi(s) {
      if (!s) return;
      this.detailSesi = s;
      this.detailModalOpen = true;
    },

    tutupDetailSesi() {
      this.detailModalOpen = false;
      this.detailSesi = null;
    },

    bukaRiwayatPerubahan() {
      this.modalRiwayatOpen = true;
      if (!this.perubahanList || this.perubahanList.length === 0) {
        this.loadPerubahan();
      }
    },

    tutupRiwayatPerubahan() {
      this.modalRiwayatOpen = false;
    },

    salinRingkasanSesi(s) {
      if (!s) return;
      const tgl = this.tanggalPekanFormatted[s.hari] || s.hari;
      const teks = `[Jadwal Kuliah - ${s.matkul}]\n` +
        `📅 Hari/Tgl: ${s.hari}, ${tgl}\n` +
        `⏰ Waktu: ${s.timeStart} – ${s.timeEnd} WIB\n` +
        `📍 Tempat: ${s.ruang ? 'Ruang ' + s.ruang : (s.link ? 'Kuliah Daring' : '—')}\n` +
        `👨‍🏫 Dosen: ${s.dosen || '—'}\n` +
        (s.pj ? `👤 PJ: ${s.pj}\n` : '') +
        (s.kind === 'PENGGANTI' ? `ℹ️ Keterangan: ${this.fmtAsalPengganti(s.originDate, s.reason)}\n` : '') +
        (s.link ? `🔗 Link: ${s.link}\n` : '');
      if (navigator && navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(teks).then(() => {
          this.showToast('Ringkasan sesi berhasil disalin ke papan klip!');
        }).catch(() => {
          this.showToast('Ringkasan sesi disalin.');
        });
      } else {
        this.showToast('Ringkasan sesi disalin.');
      }
    },

    hubungiPJ(s) {
      const name = (s && s.pj) || 'PJ Kelas';
      const matkul = (s && s.matkul) || '';
      const msg = encodeURIComponent(`Halo ${name}, saya ingin bertanya mengenai perkuliahan ${matkul}.`);
      window.open(`https://wa.me/?text=${msg}`, '_blank');
    },

    getTugasTerkait(matkul) {
      if (!matkul || !Array.isArray(this.tasks)) return [];
      const m = String(matkul).toLowerCase().trim();
      return this.tasks.filter(t => {
        const tm = String(t.matkul || '').toLowerCase().trim();
        return tm && (tm.includes(m) || m.includes(tm));
      }).slice(0, 2);
    },

    getStatusWaktu(s) {
      if (!s || !s.timeStart) return '';
      try {
        const timeParts = new Intl.DateTimeFormat('en-GB', {
          timeZone: 'Asia/Jakarta',
          hour: 'numeric',
          minute: 'numeric',
          hour12: false
        }).formatToParts(new Date());
        const h = parseInt(timeParts.find(p => p.type === 'hour')?.value || '0', 10);
        const m = parseInt(timeParts.find(p => p.type === 'minute')?.value || '0', 10);
        const cur = h * 60 + m;
        const [h1, m1] = String(s.timeStart || '').replace('.', ':').split(':').map(Number);
        const [h2, m2] = String(s.timeEnd || '').replace('.', ':').split(':').map(Number);
        const start = h1 * 60 + m1;
        const end = h2 * 60 + m2;
        if (cur >= start && cur <= end) return 'Sedang berlangsung';
        if (cur < start) return 'Akan datang';
        return 'Selesai';
      } catch (e) {
        return '';
      }
    },

    getInitials(name) {
      if (!name) return 'DS';
      const cleaned = String(name)
        .replace(/^(Bpk\.|Ibu\.|Pak\.|Dr\.|Prof\.|Ir\.)\s*/i, '')
        .replace(/,\s*[A-Za-z.\s]+$/i, '')
        .replace(/\(.*\)/g, '')
        .trim();
      const parts = cleaned.split(/\s+/).filter(Boolean);
      if (parts.length >= 2) {
        return (parts[0][0] + parts[1][0]).toUpperCase();
      }
      if (parts.length === 1 && parts[0].length >= 2) {
        return parts[0].slice(0, 2).toUpperCase();
      }
      return (cleaned[0] || 'D').toUpperCase();
    },

    getKodeMatkul(s) {
      if (!s) return 'TI2104';
      if (s.kode) return s.kode;
      const m = String(s.matkul || '').toLowerCase();
      if (m.includes('aljabar')) return 'TI2104';
      if (m.includes('web')) return 'TI2105';
      if (m.includes('basis data')) return 'TI2103';
      if (m.includes('objek') || m.includes('pbo')) return 'TI2102';
      return 'TI2101';
    },

    formatWaktuPublikasi(iso) {
      if (!iso) return 'Terbit oleh KM';
      try {
        const d = new Date(iso);
        if (isNaN(d)) return String(iso);
        const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
        const day = d.getDate();
        const month = months[d.getMonth()];
        const hh = String(d.getHours()).padStart(2, '0');
        const mm = String(d.getMinutes()).padStart(2, '0');
        return `${day} ${month}, ${hh}:${mm} WIB oleh KM`;
      } catch (e) {
        return 'Terbit oleh KM';
      }
    },

    formatWaktuEfektif(ch) {
      if (!ch) return '';
      try {
        const start = ch.starts_at ? new Date(ch.starts_at) : null;
        const end = ch.ends_at ? new Date(ch.ends_at) : null;
        const days = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
        const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
        if (start && !isNaN(start)) {
          const dayName = days[start.getDay()];
          const d = start.getDate();
          const m = months[start.getMonth()];
          const y = start.getFullYear();
          const h1 = String(start.getHours()).padStart(2, '0');
          const m1 = String(start.getMinutes()).padStart(2, '0');
          let jamStr = `${h1}:${m1}`;
          if (end && !isNaN(end)) {
            const h2 = String(end.getHours()).padStart(2, '0');
            const m2 = String(end.getMinutes()).padStart(2, '0');
            jamStr += ` - ${h2}:${m2}`;
          }
          jamStr += ' WIB';
          const ruangStr = ch.room ? ` • Ruang ${ch.room}` : '';
          return `${dayName}, ${d} ${m} ${y} • ${jamStr}${ruangStr}`;
        }
      } catch (e) { }
      return ch.room ? `Ruang ${ch.room}` : '';
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

    // Jadwal efektif (pola + perubahan terbit) untuk seluruh pekan.
    async loadJadwalPekan() {
      this.jadwalLoading = true;
      this.jadwalError = '';
      const slug = this.selectedClassSlug;
      const days = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
      try {
        const results = await Promise.all(days.map(d => {
          const tgl = this.tanggalPekan[d];
          return API.getPortalSchedule(slug, tgl);
        }));

        let totalV1Items = results.reduce((sum, r) => sum + ((r && Array.isArray(r.items)) ? r.items.length : 0), 0);
        if (totalV1Items === 0 && (!this.fullSchedule || this.fullSchedule.length === 0)) {
          await this.loadSchedule();
        }
        const useLegacyFallback = totalV1Items === 0 && (this.fullSchedule || []).length > 0;

        const newPekan = {};
        days.forEach((d, idx) => {
          const res = results[idx];
          if (!useLegacyFallback && res && Array.isArray(res.items)) {
            newPekan[d] = this.normalisasiEfektif(res.items, d);
          } else {
            newPekan[d] = (this.fullSchedule || []).filter(s => s.hari === d).map(s => ({
              id: s.id, hari: s.hari, kind: 'REGULER', activityType: 'THEORY',
              matkul: s.matkul, dosen: s.dosen || '', ruang: s.ruang || '',
              timeStart: s.timeStart || '', timeEnd: s.timeEnd || '',
              link: '', pj: '', originDate: '', reason: ''
            }));
          }
        });
        this.jadwalPekan = newPekan;
        this.jadwalEfektif = this.jadwalPekan[this.hari] || [];
        if (this.weekOffset === 0) {
          this.jadwalCacheHariIni = this.jadwalPekan[this.todayName] || [];
        }
      } catch (e) {
        this.jadwalError = 'Jadwal belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.jadwalLoading = false;
      }
    },

    async loadJadwalEfektif() {
      await this.loadJadwalPekan();
    },

    pilihHari(d) {
      this.hari = d;
      this.jadwalEfektif = this.jadwalPekan[d] || [];
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
          return {
            id: t.id, matkul: t.matkul || t.offering || 'Mata Kuliah',
            title: t.title || '', deskripsi: t.deskripsi || t.instructions || t.title || '',
            instructions: t.instructions || '',
            deadline: this.fmtDeadlineID(deadline), deadline_at: deadline,
            urgency: u.level, countdown: u.badge
          };
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

    get defaultSampleMateri() {
      return [
        // 7 berkas - Sistem Basis Data
        {
          id: 101,
          title: 'Slide Normalisasi Database: 1NF hingga BCNF',
          course_name: 'Sistem Basis Data',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/18k9aZ-Normalisasi-BCNF-Rev.pdf',
          description: 'Slide materi pertemuan ke-4 mengenai aturan pembentukan 1NF, 2NF, 3NF hingga Boyce-Codd Normal Form (BCNF). Mohon pelajari contoh dekomposisi tabel sebelum praktikum pekan depan.',
          uploader_name: 'Farhan Pratama (PJ)',
          date_display: '06 Okt',
        },
        {
          id: 102,
          title: 'Rekaman Kuliah Sesi Pengganti: DDL & DML SQL',
          course_name: 'Sistem Basis Data',
          material_type: 'MEETING',
          url: 'https://drive.google.com/file/d/1B9c8x-Rekaman-Kuliah-SQL-DDL-DML',
          description: 'Rekaman kuliah sesi pengganti daring mata kuliah Sistem Basis Data untuk materi Data Definition Language (DDL) dan Data Manipulation Language (DML).',
          uploader_name: 'Farhan Pratama (PJ)',
          date_display: '03 Okt',
        },
        {
          id: 103,
          title: 'Slide Pemodelan Konseptual: Entity Relationship Diagram (ERD)',
          course_name: 'Sistem Basis Data',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1C4d2e-ERD-Konseptual.pdf',
          description: 'Materi representasi model data ERD: entitas kuat, entitas lemah, kardinalitas relasi, dan mapping ERD ke skema relasional.',
          uploader_name: 'Dosen Pengampu',
          date_display: '28 Sep',
        },
        {
          id: 104,
          title: 'Modul Praktikum 02: Manajemen Tabel & Constraint SQL',
          course_name: 'Sistem Basis Data',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1D5f6g-Praktikum-SQL-Constraint.pdf',
          description: 'Panduan praktikum laboratorium basis data sintaks CREATE TABLE, ALTER TABLE, PRIMARY KEY, FOREIGN KEY, dan CHECK constraint.',
          uploader_name: 'Farhan Pratama (PJ)',
          date_display: '24 Sep',
        },
        {
          id: 105,
          title: 'Kumpulan Kueri Contoh: Join & Subquery PostgreSQL',
          course_name: 'Sistem Basis Data',
          material_type: 'REPOSITORY',
          url: 'https://github.com/d4ti-polban/sbd-queries-join-subquery',
          description: 'Kumpulan skrip SQL demonstrasi inner join, left join, full outer join, dan korelasi subquery untuk latihan mandiri.',
          uploader_name: 'Farhan Pratama (PJ)',
          date_display: '20 Sep',
        },
        {
          id: 106,
          title: 'Slide Pengenalan Relational DBMS & Arsitektur PostgreSQL',
          course_name: 'Sistem Basis Data',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1E6g7h-Intro-RDBMS-Postgres.pdf',
          description: 'Konsep dasar ACID, storage engine, dan arsitektur client-server sistem manajemen basis data relasional.',
          uploader_name: 'Dosen Pengampu',
          date_display: '15 Sep',
        },
        {
          id: 107,
          title: 'Latihan Mandiri: Normalisasi Kasus Transaksi Penjualan',
          course_name: 'Sistem Basis Data',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1F7h8i-Latihan-Normalisasi-Transaksi.pdf',
          description: 'Studi kasus faktur belanja supermarket untuk didekomposisi dari unnormalized form (UNF) hingga 3NF.',
          uploader_name: 'Farhan Pratama (PJ)',
          date_display: '12 Sep',
        },

        // 6 berkas - Pemrograman Berorientasi Objek
        {
          id: 201,
          title: 'Starter Repository: Template Proyek Java OOP',
          course_name: 'Pemrograman Berorientasi Objek',
          material_type: 'REPOSITORY',
          url: 'https://github.com/d4ti-polban/oop-java-starter-template',
          description: 'Starter repository untuk proyek pemrograman berorientasi objek menggunakan Java. Sudah dilengkapi modul Gradle dan template boilerplate.',
          uploader_name: 'Farhan Pratama (PJ)',
          date_display: '28 Sep',
        },
        {
          id: 202,
          title: 'Slide Inheritance & Polymorphism dalam Java',
          course_name: 'Pemrograman Berorientasi Objek',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1G8i9j-Java-Inheritance-Poly.pdf',
          description: 'Konsep pewarisan kelas, method overriding, overloading, dynamic method dispatch, dan interface implementation.',
          uploader_name: 'Dosen Pengampu',
          date_display: '26 Sep',
        },
        {
          id: 203,
          title: 'Rekaman Sesi Lab: Implementasi Abstract Class & Interface',
          course_name: 'Pemrograman Berorientasi Objek',
          material_type: 'MEETING',
          url: 'https://drive.google.com/file/d/1H9j0k-Lab-Abstract-Interface',
          description: 'Demonstrasi live coding pembuatan hierarki objek, antarmuka kontrak, dan sistem pembayaran elektronik.',
          uploader_name: 'Farhan Pratama (PJ)',
          date_display: '22 Sep',
        },
        {
          id: 204,
          title: 'Slide Encapsulation & Access Modifiers',
          course_name: 'Pemrograman Berorientasi Objek',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1I0k1l-Encapsulation-Access.pdf',
          description: 'Prinsip information hiding, penggunaan private, protected, public, serta pembuatan getter-setter yang konsisten.',
          uploader_name: 'Dosen Pengampu',
          date_display: '18 Sep',
        },
        {
          id: 205,
          title: 'Modul Praktikum Java Collections Framework',
          course_name: 'Pemrograman Berorientasi Objek',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1J1l2m-Java-Collections-Guide.pdf',
          description: 'Panduan penggunaan List (ArrayList, LinkedList), Set (HashSet), dan Map (HashMap) dalam aplikasi Java.',
          uploader_name: 'Farhan Pratama (PJ)',
          date_display: '14 Sep',
        },
        {
          id: 206,
          title: 'Slide Pengenalan Objek, Kelas, & Siklus Hidup Instansiasi',
          course_name: 'Pemrograman Berorientasi Objek',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1K2m3n-Java-Class-Object-Lifecycle.pdf',
          description: 'Dasar paradigma berorientasi objek: cetak biru kelas, instansiasi memori heap/stack, dan alokasi objek.',
          uploader_name: 'Dosen Pengampu',
          date_display: '08 Sep',
        },

        // 5 berkas - Matematika Diskrit
        {
          id: 301,
          title: 'Slide Graf & Pohon Rentang Minimum (MST)',
          course_name: 'Matematika Diskrit',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1L3n4o-Graf-Spanning-Tree.pdf',
          description: 'Teori graf dasar, matriks ketetanggaan, dan algoritma penentuan MST (Kruskal dan Prim).',
          uploader_name: 'Dosen Pengampu',
          date_display: '02 Okt',
        },
        {
          id: 302,
          title: 'Modul Latihan Relasi Ekivalensi & Poset',
          course_name: 'Matematika Diskrit',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1M4o5p-Relasi-Poset-Latihan.pdf',
          description: 'Sifat-sifat relasi biner, relasi kesetaraan, partisi himpunan, dan diagram Hasse untuk partially ordered set.',
          uploader_name: 'Farhan Pratama (PJ)',
          date_display: '27 Sep',
        },
        {
          id: 303,
          title: 'Slide Logika Proposisional & Inferensi Deduktif',
          course_name: 'Matematika Diskrit',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1N5p6q-Logika-Proposisi.pdf',
          description: 'Tabel kebenaran, hukum aljabar proposisi, bentuk normal konjungtif (CNF), dan kaidah inferensi valid.',
          uploader_name: 'Dosen Pengampu',
          date_display: '21 Sep',
        },
        {
          id: 304,
          title: 'Rekaman Kuliah: Teknik Pembuktian & Induksi Matematika',
          course_name: 'Matematika Diskrit',
          material_type: 'MEETING',
          url: 'https://drive.google.com/file/d/1O6q7r-Rekaman-Induksi-Matematika',
          description: 'Penjelasan pembuktian langsung, kontraposisi, kontradiksi, dan langkah basis serta langkah induktif.',
          uploader_name: 'Farhan Pratama (PJ)',
          date_display: '16 Sep',
        },
        {
          id: 305,
          title: 'Slide Kombinatorika & Prinsip Sarang Burung Merpati',
          course_name: 'Matematika Diskrit',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1P7r8s-Kombinatorika-Pigeonhole.pdf',
          description: 'Kaidah perkalian, permutasi, kombinasi, dan aplikasi Pigeonhole Principle dalam komputasi.',
          uploader_name: 'Dosen Pengampu',
          date_display: '10 Sep',
        },

        // 4 berkas - Arsitektur Komputer
        {
          id: 401,
          title: 'Slide Arsitektur Instruction Set & Datapath MIPS',
          course_name: 'Arsitektur Komputer',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1Q8s9t-MIPS-Datapath.pdf',
          description: 'Format instruksi MIPS 32-bit (R/I/J type), alur data prosesor single-cycle, dan sinyal kendali ALU.',
          uploader_name: 'Dosen Pengampu',
          date_display: '01 Okt',
        },
        {
          id: 402,
          title: 'Simulator MIPS MARS & Kumpulan Skrip Contoh',
          course_name: 'Arsitektur Komputer',
          material_type: 'REPOSITORY',
          url: 'https://github.com/d4ti-polban/mars-mips-sample-code',
          description: 'Panduan instalasi MARS simulator dan contoh implementasi aritmatika biner dalam bahasa assembly.',
          uploader_name: 'Farhan Pratama (PJ)',
          date_display: '25 Sep',
        },
        {
          id: 403,
          title: 'Slide Hirarki Memori & Manajemen Cache',
          course_name: 'Arsitektur Komputer',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1R9t0u-Memory-Hierarchy-Cache.pdf',
          description: 'Prinsip lokalitas waktu dan ruang, pemetaan cache direct vs associative, serta kebijakan write-through/back.',
          uploader_name: 'Dosen Pengampu',
          date_display: '19 Sep',
        },
        {
          id: 404,
          title: 'Slide Arsitektur Von Neumann & Siklus Instruksi',
          course_name: 'Arsitektur Komputer',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1S0u1v-Von-Neumann-Cycle.pdf',
          description: 'Komponen dasar von Neumann: register CPU, bus sistem, dan siklus fetch-decode-execute.',
          uploader_name: 'Dosen Pengampu',
          date_display: '11 Sep',
        },

        // 2 berkas - Umum Kelas (Silabus & RPS)
        {
          id: 501,
          title: 'Silabus Perkuliahan & Kontrak Belajar D4-TI 1A',
          course_name: 'Umum Kelas (Silabus & RPS)',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1T1v2w-Silabus-Kontrak-Kuliah.pdf',
          description: 'Dokumen rujukan kurikulum, aturan presensi, bobot evaluasi nilai, dan tata tertib perkuliahan semester ganjil.',
          uploader_name: 'Pengurus Kelas',
          date_display: '05 Sep',
        },
        {
          id: 502,
          title: 'Rencana Pembelajaran Semester (RPS) Terpadu Kurikulum 2024',
          course_name: 'Umum Kelas (Silabus & RPS)',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1U2w3x-RPS-Terpadu-Kurikulum.pdf',
          description: 'Rincian capaian pembelajaran mata kuliah (CPMK) seluruh mata kuliah wajib D4 Teknik Informatika tingkat 1.',
          uploader_name: 'Pengurus Kelas',
          date_display: '03 Sep',
        },

        // 4 berkas pelengkap lainnya
        {
          id: 601,
          title: 'Slide Modul Aljabar Linear: Eliminasi Gauss-Jordan & Matriks Invers',
          course_name: 'Aljabar Linear',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1V3x4y-Alin-Gauss-Jordan.pdf',
          description: 'Operasi baris elementer, eliminasi Gauss-Jordan, determinan, dan pencarian matriks balikan.',
          uploader_name: 'Farhan Pratama (PJ)',
          date_display: '29 Sep',
        },
        {
          id: 602,
          title: 'Repositori Praktikum Aljabar Linear dengan NumPy',
          course_name: 'Aljabar Linear',
          material_type: 'REPOSITORY',
          url: 'https://github.com/d4ti-polban/alin-numpy-labs',
          description: 'Kumpulan notebook praktikum komputasi vektor dan operasi matriks menggunakan pustaka Python NumPy.',
          uploader_name: 'Farhan Pratama (PJ)',
          date_display: '23 Sep',
        },
        {
          id: 603,
          title: 'Slide Technical Writing & Standar Penulisan Laporan',
          course_name: 'Bahasa Inggris',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1W4y5z-English-Tech-Writing.pdf',
          description: 'Pedoman penulisan laporan teknis praktikum, gaya sitasi IEEE, dan terminologi teknologi informasi formal.',
          uploader_name: 'Dosen Pengampu',
          date_display: '17 Sep',
        },
        {
          id: 604,
          title: 'Panduan Etika Akademik & Keselamatan Kerja Lab Komputer',
          course_name: 'Etika Profesi',
          material_type: 'DOCUMENT',
          url: 'https://drive.google.com/file/d/1X5z6a-K3-Etika-Lab.pdf',
          description: 'Prosedur operasional baku penggunaan laboratorium komputer, hak cipta software, dan pencegahan plagiarisme kode.',
          uploader_name: 'Pengurus Kelas',
          date_display: '07 Sep',
        },
      ];
    },

    async loadMateri() {
      this.materiLoading = true;
      this.materiError = '';
      try {
        const dbItems = await API.getPortalMaterials(this.selectedClassSlug).catch(() => []);
        if (dbItems && dbItems.length > 0) {
          const sample = this.defaultSampleMateri;
          const titles = new Set(dbItems.map(d => (d.title || '').toLowerCase().trim()));
          const filteredSample = sample.filter(s => !titles.has((s.title || '').toLowerCase().trim()));
          this.materiList = [...dbItems, ...filteredSample];
        } else {
          this.materiList = [...this.defaultSampleMateri];
        }
      } catch (e) {
        this.materiList = [...this.defaultSampleMateri];
      } finally {
        this.materiLoading = false;
      }
    },

    get materiCategories() {
      const all = this.materiList || [];
      const totalCount = all.length;

      const umumCount = all.filter(m => {
        const c = (m.course_name || '').toLowerCase();
        return !m.course_name || c.includes('umum') || c.includes('silabus') || c.includes('rps');
      }).length;

      const list = [
        { key: 'SEMUA', label: 'Semua Materi', shortLabel: 'Semua', count: totalCount },
        { key: 'UMUM', label: 'Umum Kelas (Silabus & RPS)', shortLabel: 'Umum Kelas', count: umumCount },
      ];

      const courseDefs = [
        { key: 'Sistem Basis Data', label: 'Sistem Basis Data', shortLabel: 'Basis Data' },
        { key: 'Pemrograman Berorientasi Objek', label: 'Pemrograman Berorientasi Objek', shortLabel: 'PBO' },
        { key: 'Matematika Diskrit', label: 'Matematika Diskrit', shortLabel: 'Matdis' },
        { key: 'Arsitektur Komputer', label: 'Arsitektur Komputer', shortLabel: 'Arsitektur' },
      ];

      const foundKeys = new Set(courseDefs.map(c => c.key.toLowerCase()));

      courseDefs.forEach(c => {
        const count = all.filter(m => (m.course_name || '').trim().toLowerCase() === c.key.toLowerCase()).length;
        list.push({ ...c, count });
      });

      all.forEach(m => {
        const cName = (m.course_name || '').trim();
        if (cName && !foundKeys.has(cName.toLowerCase()) && !cName.toLowerCase().includes('umum')) {
          foundKeys.add(cName.toLowerCase());
          const count = all.filter(item => (item.course_name || '').trim().toLowerCase() === cName.toLowerCase()).length;
          list.push({
            key: cName,
            label: cName,
            shortLabel: cName.length > 12 ? cName.split(' ')[0] : cName,
            count,
          });
        }
      });

      return list;
    },

    get filteredMateriList() {
      let list = [...(this.materiList || [])];
      const filter = this.filterMateriMatkul;

      if (filter === 'UMUM') {
        list = list.filter(m => {
          const c = (m.course_name || '').toLowerCase();
          return !m.course_name || c.includes('umum') || c.includes('silabus') || c.includes('rps');
        });
      } else if (filter && filter !== 'SEMUA') {
        list = list.filter(m => (m.course_name || '').trim().toLowerCase() === filter.toLowerCase());
      }

      if (this.sortMateri === 'terlama') {
        list.sort((a, b) => (a.id || 0) - (b.id || 0));
      } else {
        list.sort((a, b) => (b.id || 0) - (a.id || 0));
      }

      return list;
    },

    get currentMateriHeading() {
      const count = this.filteredMateriList.length;
      if (this.filterMateriMatkul === 'SEMUA') {
        return `Semua Berkas Materi (${count} Berkas)`;
      }
      if (this.filterMateriMatkul === 'UMUM') {
        return `Umum Kelas (Silabus & RPS) (${count} Berkas)`;
      }
      return `${this.filterMateriMatkul} (${count} Berkas)`;
    },

    getMateriType(m) {
      if (!m) return 'Slide';
      const t = String(m.material_type || '').toUpperCase();
      const title = String(m.title || '').toLowerCase();
      if (t === 'MEETING' || title.includes('rekaman') || title.includes('video')) return 'Video';
      if (t === 'REPOSITORY' || title.includes('repositori') || title.includes('starter') || title.includes('template')) return 'Repositori';
      if (t === 'DOCUMENT') {
        if (title.includes('slide')) return 'Slide';
        return 'Dokumen';
      }
      if (title.includes('slide')) return 'Slide';
      return 'Slide';
    },

    getMateriSubtitle(m) {
      if (!m) return '';
      if (m.subtitle_display) return m.subtitle_display;
      const course = m.course_name || 'Umum Kelas';
      const type = this.getMateriType(m);
      if (type === 'Video') return `${course} · Tautan Google Drive`;
      if (type === 'Repositori') return `${course} · GitHub`;

      let format = 'PDF';
      if (m.url && !m.url.includes('.pdf') && !m.url.includes('drive')) format = 'Dokumen';
      const uploader = m.uploader_name ? `Diunggah oleh ${m.uploader_name}` : 'Diunggah oleh PJ (Farhan)';
      return `${course} · ${format} · ${uploader}`;
    },

    getMateriDate(m) {
      if (!m) return '06 Okt';
      if (m.date_display) return m.date_display;
      if (m.created_at) {
        try {
          const d = new Date(m.created_at);
          if (!isNaN(d.getTime())) {
            const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
            const pad = (n) => String(n).padStart(2, '0');
            return `${pad(d.getDate())} ${months[d.getMonth()]}`;
          }
        } catch (e) { }
      }
      return '06 Okt';
    },

    getMateriModalMeta(m) {
      if (!m) return '';
      const type = this.getMateriType(m);
      let format = 'Dokumen PDF';
      if (type === 'Video') format = 'Tautan Video (Google Drive)';
      else if (type === 'Repositori') format = 'Repositori Kode (GitHub)';

      const uploader = m.uploader_name || 'Farhan Pratama (PJ)';
      return `Format: ${format} · Diunggah oleh ${uploader}`;
    },

    cleanUrlDisplay(url) {
      if (!url) return 'drive.google.com/...';
      let clean = url.replace(/^https?:\/\//i, '');
      if (clean.length > 42) {
        clean = clean.substring(0, 39) + '...';
      }
      return clean;
    },

    bukaDetailMateri(m) {
      this.selectedMateri = m;
      this.detailMateriModalOpen = true;
    },

    tutupDetailMateri() {
      this.detailMateriModalOpen = false;
      this.selectedMateri = null;
    },

    salinTautanMateri(url) {
      if (!url) {
        this.showToast('Tautan belum tersedia.');
        return;
      }
      if (navigator && navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(url).then(() => {
          this.showToast('Tautan berkas berhasil disalin!');
        }).catch(() => {
          this.showToast('Tautan berkas berhasil disalin!');
        });
      } else {
        this.showToast('Tautan berkas berhasil disalin!');
      }
    },

    bukaTautanLangsung(url) {
      if (!url) {
        this.showToast('Tautan berkas belum tersedia.');
        return;
      }
      window.open(url, '_blank', 'noopener,noreferrer');
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
