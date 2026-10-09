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
    dashboardRefreshing: false,
    dashboardData: null,
    dashboardError: '',
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
    semesterStatus: '',

    ...AsteriskShell.behavior('pj'),

    roleLabel: 'PJ',
    contextAssignments: [],
    contextSwitching: false,

    navSections: [
      { title: 'UTAMA', items: [
        { id: 'dashboard', label: 'Dashboard', icon: 'space_dashboard' },
      ] },
      { title: 'AKADEMIK', items: [
        { id: 'tugas', label: 'Tugas Dikelola', icon: 'assignment', active: ['tugas', 'tambah', 'tinjau-tugas', 'detail-tugas', 'ubah-tugas', 'preview', 'konfirmasi', 'terbit'] },
        { id: 'jadwal', label: 'Jadwal Kuliah', icon: 'calendar_month', active: ['jadwal', 'pindah', 'perubahan'] },
        { id: 'materi', label: 'Materi', icon: 'menu_book' },
      ] },
    ],

    pjNavIcon(item) {
      const icon = item && item.icon;
      const paths = {
        'space_dashboard': '<svg width="20" height="20" viewBox="0 0 14 14" fill="currentColor" class="w-5 h-5 shrink-0" aria-hidden="true" focusable="false"><path d="M2.95996 12.36963q-0.55371 0-0.94336-0.38623-0.38623-0.38965-0.38623-0.94336l0-8.08008q0-0.55371 0.38623-0.93994 0.38965-0.38965 0.94336-0.38965l8.08008 0q0.55371 0 0.93994 0.38965 0.38965 0.38623 0.38965 0.93994l0 8.08008q0 0.55371-0.38965 0.94336-0.38623 0.38623-0.93994 0.38623l-8.08008 0z m0-1.32959l3.45557 0 0-8.08008-3.45557 0 0 8.08008z m4.62451 0l3.45557 0 0-4.04004-3.45557 0 0 4.04004z m0-5.20557l3.45557 0 0-2.87451-3.45557 0 0 2.87451z"/></svg>',
        'assignment': '<svg width="20" height="20" viewBox="0 0 14 14" fill="currentColor" class="w-5 h-5 shrink-0" aria-hidden="true" focusable="false"><path d="M2.95996 12.36963q-0.55029 0-0.93994-0.38965-0.38965-0.38965-0.38965-0.93994l0-8.08008q0-0.55029 0.38623-0.93994 0.38965-0.38965 0.93994-0.38965l2.33447 0q0.21191-0.52979 0.67334-0.84766 0.46484-0.31787 1.03223-0.31787 0.56738 0 1.02881 0.31787 0.46484 0.31787 0.67676 0.84766l2.33447 0q0.55029 0 0.93994 0.38965 0.38965 0.38965 0.38965 0.93994l0 8.08008q0 0.55029-0.38965 0.93994-0.38965 0.38965-0.93994 0.38965l-8.08008 0z m0-1.32959l8.08008 0 0-8.08008-8.08008 0 0 8.08008z m1.75-1.16553l2.86426 0q0.24609 0 0.41357-0.16748 0.16748-0.16748 0.16748-0.41699 0-0.24609-0.16748-0.41358-0.16748-0.16748-0.41357-0.16748l-2.86426 0q-0.24951 0-0.41699 0.16748-0.16748 0.16748-0.16748 0.41358 0 0.24951 0.16748 0.41699 0.16748 0.16748 0.41699 0.16748z m0-2.29004l4.58008 0q0.24951 0 0.41699-0.16748 0.16748-0.16748 0.16748-0.41699 0-0.24951-0.16748-0.41699-0.16748-0.16748-0.41699-0.16748l-4.58008 0q-0.24951 0-0.41699 0.16748-0.16748 0.16748-0.16748 0.41699 0 0.24951 0.16748 0.41699 0.16748 0.16748 0.41699 0.16748z m0-2.29345l4.58008 0q0.24951 0 0.41699-0.16748 0.16748-0.16748 0.16748-0.41358 0-0.24951-0.16748-0.41699-0.16748-0.16748-0.41699-0.16748l-4.58008 0q-0.24951 0-0.41699 0.16748-0.16748 0.16748-0.16748 0.41699 0 0.24951 0.16748 0.41699 0.16748 0.16748 0.41699 0.16748z m2.29004-2.78565q0.20166 0 0.33154-0.12988 0.1333-0.1333 0.1333-0.33496 0-0.20166-0.1333-0.33155-0.12988-0.1333-0.33154-0.1333-0.20166 0-0.33496 0.1333-0.12988 0.12988-0.12988 0.33155 0 0.20166 0.12988 0.33496 0.1333 0.12988 0.33496 0.12988z m-4.04004 8.53467l0-8.08008 0 8.08008z"/></svg>',
        'calendar_month': '<svg width="20" height="20" viewBox="0 0 14 14" fill="currentColor" class="w-5 h-5 shrink-0" aria-hidden="true" focusable="false"><path d="M2.95996 12.95068q-0.55371 0-0.94336-0.38623-0.38623-0.38965-0.38623-0.93994l0-8.08349q0-0.55029 0.38623-0.93653 0.38965-0.38965 0.94336-0.38965l0.54004 0 0-0.54345q0-0.26318 0.18115-0.44092 0.18115-0.18115 0.44434-0.18115 0.26318 0 0.44092 0.18115 0.18115 0.17773 0.18115 0.44092l0 0.54345 4.50488 0 0-0.54345q0-0.26318 0.18115-0.44092 0.18115-0.18115 0.44434-0.18115 0.26318 0 0.44092 0.18115 0.18115 0.17773 0.18115 0.44092l0 0.54345 0.54004 0q0.55371 0 0.93994 0.38965 0.38965 0.38623 0.38965 0.93653l0 8.08349q0 0.55029-0.38965 0.93994-0.38623 0.38623-0.93994 0.38623l-8.08008 0z m0-1.32617l8.08008 0 0-5.79004-8.08008 0 0 5.79004z m0-6.95898l8.08008 0 0-1.12451-8.08008 0 0 1.12451z m0 0l0-1.12451 0 1.12451z m4.04004 3.52734q-0.25977 0-0.4375-0.17432-0.17432-0.17432-0.17432-0.43408 0-0.25977 0.17432-0.43408 0.17773-0.17773 0.4375-0.17773 0.25977 0 0.43408 0.17773 0.17773 0.17432 0.17774 0.43408 0 0.25977-0.17774 0.43408-0.17432 0.17432-0.43408 0.17432z m-2.33447 0q-0.25977 0-0.43408-0.17432-0.17432-0.17432-0.17432-0.43408 0-0.25977 0.17432-0.43408 0.17432-0.17773 0.43408-0.17773 0.25977 0 0.43408 0.17773 0.17773 0.17432 0.17773 0.43408 0 0.25977-0.17773 0.43408-0.17432 0.17774-0.43408 0.17773z m4.66894 0q-0.25635 0-0.43408-0.17432-0.17773-0.17773-0.17773-0.43408 0-0.25635 0.17773-0.43067 0.17773-0.17773 0.43408-0.17773 0.25635 0 0.43067 0.17773 0.17773 0.17432 0.17773 0.43408 0 0.25977-0.17432 0.43408-0.17432 0.17774-0.43408 0.17773z m-2.33447 2.33447q-0.25977 0-0.4375-0.17773-0.17432-0.17773-0.17432-0.43408 0-0.25635 0.17432-0.43067 0.17773-0.17773 0.4375-0.17773 0.25977 0 0.43408 0.17432 0.17773 0.17432 0.17774 0.43408 0 0.25635-0.17774 0.43408-0.17432 0.17774-0.43408 0.17773z m-2.33447 0q-0.25977 0-0.43408-0.17773-0.17432-0.17773-0.17432-0.43408 0-0.25635 0.17432-0.43067 0.17432-0.17773 0.43408-0.17773 0.25977 0 0.43408 0.17432 0.17773 0.17773 0.17773 0.43408 0 0.25977-0.17773 0.43408-0.17432 0.17774-0.43408 0.17773z m4.66894 0q-0.25635 0-0.43408-0.17773-0.17773-0.17773-0.17773-0.43408 0-0.25635 0.17773-0.43067 0.17773-0.17773 0.43408-0.17773 0.25635 0 0.43067 0.17432 0.17773 0.17432 0.17773 0.43408 0 0.25977-0.17432 0.43408-0.17432 0.17774-0.43408 0.17773z"/></svg>',
        'menu_book': '<svg width="20" height="20" viewBox="0 0 14 14" fill="currentColor" class="w-5 h-5 shrink-0" aria-hidden="true" focusable="false"><path d="M3.79053 9.46094q0.68701 0 1.33642 0.15381 0.64941 0.15381 1.28858 0.46142l0-5.7456q-0.59814-0.35205-1.26807-0.52637-0.66992-0.17432-1.35693-0.17432-0.52295 0-1.04248 0.10254-0.51611 0.09912-0.99805 0.3042l0 5.77637q0.50928-0.17432 1.01172-0.26319 0.50586-0.08887 1.02881-0.08886z m3.79394 0.61523q0.63916-0.30762 1.28858-0.46142 0.64941-0.15381 1.33642-0.15381 0.52295 0 1.02539 0.08886 0.50586 0.08887 1.01514 0.26319l0-5.77637q-0.48193-0.20508-1.00146-0.3042-0.51611-0.10254-1.03907-0.10254-0.68701 0-1.35693 0.17432-0.66992 0.17432-1.26807 0.52637l0 5.7456z m-0.57422 1.53125q-0.2085 0-0.3999-0.05127-0.19141-0.05127-0.35547-0.14355-0.56738-0.33154-1.18945-0.50586-0.62207-0.17432-1.2749-0.17432-0.59814 0-1.1792 0.16406-0.57764 0.16065-1.11084 0.44776-0.35547 0.18457-0.69043-0.01709-0.33496-0.20166-0.33496-0.59131l0-7.02393q0-0.2085 0.09912-0.39306 0.09912-0.18457 0.29736-0.27686 0.68018-0.34863 1.41162-0.51611 0.73145-0.1709 1.49707-0.1709 0.85449 0 1.66455 0.21875 0.81348 0.21875 1.55518 0.6665 0.74512-0.44434 1.55518-0.66308 0.81006-0.22217 1.66455-0.22217 0.76563 0 1.49707 0.1709 0.73145 0.16748 1.41162 0.51611 0.19824 0.09229 0.29736 0.27686 0.09912 0.18457 0.09912 0.39306l0 7.07862q0 0.37256-0.33496 0.55713-0.33154 0.18457-0.69043-0.00342-0.5332-0.28711-1.11426-0.44776-0.57764-0.16406-1.17578-0.16406-0.64941 0-1.26465 0.17774-0.61523 0.17432-1.17578 0.50244-0.16406 0.09229-0.35888 0.14355-0.19141 0.05127-0.39991 0.05127z m1.15528-6.36767q0-0.12646 0.09228-0.25977 0.09229-0.13672 0.2085-0.18115 0.42383-0.14697 0.84765-0.21875 0.42725-0.0752 0.89551-0.0752 0.29053 0 0.57422 0.0376 0.28369 0.03418 0.56055 0.09229 0.1333 0.03076 0.22558 0.15039 0.0957 0.11621 0.09571 0.26318 0 0.24609-0.16065 0.3623-0.16064 0.11279-0.40673 0.05469-0.2085-0.04102-0.43067-0.06152-0.22217-0.02393-0.45801-0.02393-0.37939 0-0.7417 0.07178-0.3623 0.06836-0.69726 0.18799-0.26318 0.09912-0.43408-0.01709-0.1709-0.11621-0.1709-0.38281z m0 3.20947q0-0.12646 0.09228-0.25977 0.09229-0.13672 0.2085-0.18115 0.42383-0.14697 0.84765-0.22217 0.42725-0.0752 0.89551-0.07519 0.29053 0 0.57422 0.0376 0.28369 0.0376 0.56055 0.0957 0.1333 0.03076 0.22558 0.14697 0.0957 0.11621 0.09571 0.26318 0 0.24951-0.16065 0.36573-0.16064 0.11279-0.40673 0.05468-0.2085-0.04443-0.43067-0.06494-0.22217-0.02393-0.45801-0.02392-0.37939 0-0.7417 0.06494-0.3623 0.06494-0.69726 0.18115-0.26318 0.10254-0.43408-0.00683-0.1709-0.11279-0.1709-0.37598z m0-1.60303q0-0.12988 0.09228-0.26318 0.09229-0.13672 0.2085-0.18115 0.42383-0.14355 0.84765-0.21875 0.42725-0.0752 0.89551-0.0752 0.29053 0 0.57422 0.0376 0.28369 0.0376 0.56055 0.0957 0.1333 0.02734 0.22558 0.14697 0.0957 0.11621 0.09571 0.26319 0 0.24951-0.16065 0.36572-0.16064 0.11279-0.40673 0.05469-0.2085-0.04443-0.43067-0.06494-0.22217-0.02393-0.45801-0.02393-0.37939 0-0.7417 0.07178-0.3623 0.07178-0.69726 0.18799-0.26318 0.10254-0.43408-0.01368-0.1709-0.11963-0.1709-0.38281z"/></svg>'
      };
      return paths[icon] || '';
    },

    pjBreadcrumbs() {
      const c = [];
      c.push({ label: 'Sistem', action: () => this.go('dashboard'), clickable: true });
      const kelas = this.dashboardData?.class?.code || this.dashboardData?.class?.label || this.selectedClass || 'D4-TI 1A';
      c.push({ label: kelas, action: () => this.go('dashboard'), clickable: false });

      if (this.view === 'dashboard') {
        c.push({ label: 'Ruang Kerja PJ', active: true });
      } else if (['tugas', 'tambah', 'detail-tugas', 'ubah-tugas', 'preview', 'konfirmasi', 'terbit'].includes(this.view)) {
        if (this.view === 'tugas') {
          c.push({ label: 'Tugas Dikelola', active: true });
        } else {
          c.push({ label: 'Tugas Dikelola', action: () => this.go('tugas'), clickable: true });
          if (this.view === 'tambah') {
            c.push({ label: 'Tambah Draf Baru', active: true });
          } else if (this.view === 'ubah-tugas') {
            const id = this.editTugasId || this.tugasDetail?.id || '';
            const isFix = this.tugasKoreksi || (this.tugasDetail && String(this.tugasDetail.review_state || '').toUpperCase() === 'CHANGES_REQUESTED');
            c.push({ label: isFix ? `Perbaiki Draf #${id || ''}`.trim() : `Ubah Draf #${id || ''}`.trim(), active: true });
          } else if (this.view === 'detail-tugas') {
            c.push({ label: 'Detail Tugas', active: true });
          } else if (this.view === 'preview') {
            c.push({ label: 'Pratinjau Pesan', active: true });
          } else if (this.view === 'konfirmasi') {
            c.push({ label: 'Konfirmasi Pengajuan', active: true });
          } else {
            c.push({ label: this.view, active: true });
          }
        }
      } else if (['jadwal', 'pindah', 'perubahan'].includes(this.view)) {
        if (this.view === 'jadwal') {
          c.push({ label: 'Jadwal Kuliah', active: true });
        } else {
          c.push({ label: 'Jadwal Kuliah', action: () => this.go('jadwal'), clickable: true });
          c.push({ label: this.view === 'pindah' ? 'Pindah Jadwal' : 'Perubahan Jadwal', active: true });
        }
      } else if (this.view === 'materi') {
        c.push({ label: 'Materi Kuliah', active: true });
      } else if (this.view === 'akun') {
        c.push({ label: 'Pengaturan Akun', active: true });
      } else if (this.view === 'audit') {
        c.push({ label: 'Riwayat Perubahan', active: true });
      } else {
        c.push({ label: (this.nav?.find(item => item.id === this.view)?.label) || this.view, active: true });
      }
      return c;
    },

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
    tugasSubmitting: false,
    tugasSubmittingDraf: false,
    terbitOk: false,
    tugasTab: 'aktif',
    tugasDari: '', tugasSampai: '',
    tugasLoading: false, tugasListError: '',
    tugasDetail: null, tugasReviews: [], tugasDetailLoading: false, tugasDetailTab: 'detail',
    tugasPreview: null, tugasPublishMsg: '',
    taskDelivery: [], eventDelivery: [], deliveryLoading: false, deliveryError: '',
    editTugasId: '', editTugasVersion: 0, tugasConflict: null,
    initialInstructions: '',

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

    tabLabel(t) {
      if (t === 'draf' || t === 'draf_review') return 'Draf & Review';
      if (t === 'selesai') return 'Selesai';
      if (t === 'arsip') return 'Arsip';
      return 'Aktif';
    },

    get tugasTabCounts() {
      const c = { aktif: 0, draf: 0, selesai: 0, arsip: 0 };
      (this.tasks || []).forEach(t => {
        if (t.archived_at) { c.arsip++; return; }
        if (t.completed_at) { c.selesai++; return; }
        const isDraft = String(t.publication_status || '').toUpperCase() === 'DRAFT';
        const isReview = String(t.review_state || '').toUpperCase() === 'NOT_REVIEWED' || String(t.review_state || '').toUpperCase() === 'CHANGES_REQUESTED';
        if (isDraft || isReview) { c.draf++; return; }
        c.aktif++;
      });
      return c;
    },

    get tugasTabList() {
      const dari = this.tugasDari ? new Date(this.tugasDari + 'T00:00:00+07:00') : null;
      const sampai = this.tugasSampai ? new Date(this.tugasSampai + 'T23:59:59+07:00') : null;
      return (this.tasks || []).filter(t => {
        const isDraft = String(t.publication_status || '').toUpperCase() === 'DRAFT';
        const isReview = String(t.review_state || '').toUpperCase() === 'NOT_REVIEWED' || String(t.review_state || '').toUpperCase() === 'CHANGES_REQUESTED';
        if (t.archived_at) {
          if (this.tugasTab !== 'arsip') return false;
        } else if (t.completed_at) {
          if (this.tugasTab !== 'selesai') return false;
        } else if (isDraft || isReview) {
          if (this.tugasTab !== 'draf' && this.tugasTab !== 'draf_review') return false;
        } else {
          if (this.tugasTab !== 'aktif') return false;
        }
        if (dari || sampai) {
          const d = t.deadline_at ? new Date(t.deadline_at) : null;
          if (!d) return false;
          if (dari && d < dari) return false;
          if (sampai && d > sampai) return false;
        }
        if (!this.shellMatchesSearch('tugas', [t.title, t.deskripsi, t.instructions, t.matkul])) return false;
        return true;
      }).slice().sort((a, b) => String(a.deadline_at || '').localeCompare(String(b.deadline_at || '')));
    },

    get tugasFilterAktif() { return !!(this.tugasDari || this.tugasSampai); },
    hapusTugasFilter() { this.tugasDari = ''; this.tugasSampai = ''; },

    get tugasKoreksi() {
      if (this.dashboardData && this.dashboardData.correction && this.dashboardData.correction.task_id) {
        return this.dashboardData.correction;
      }
      const rev = (this.tasks || []).find(t => String(t.review_state || '').toUpperCase() === 'CHANGES_REQUESTED');
      if (rev) {
        return {
          task_id: rev.id,
          task_title: rev.title || rev.deskripsi,
          reviewer_name: 'Ketua Murid',
          note: 'Instruksi tugas perlu disesuaikan sebelum disetujui KM.'
        };
      }
      return null;
    },

    async perbaikiKoreksi(taskId) {
      if (!taskId) return;
      await this.bukaDetailTugas(taskId);
      this.mulaiUbahTugas();
    },

    tugasDosen(t) {
      if (this.dashboardData && this.dashboardData.offering && Array.isArray(this.dashboardData.offering.lecturers) && this.dashboardData.offering.lecturers.length > 0) {
        return this.dashboardData.offering.lecturers.join(', ');
      }
      const sched = (this.fullSchedule || []).find(s => s.dosen && (!t || !t.matkul || s.matkul === t.matkul));
      if (sched && sched.dosen) return sched.dosen;
      return 'Dosen Pengampu';
    },

    tugasTempat(t) {
      if (!t) return 'LMS Kampus';
      if (t.submission_text && t.submission_text.trim()) return t.submission_text.trim();
      if (t.submission_url && t.submission_url.trim()) return t.submission_url.trim();
      return 'LMS Kampus';
    },

    countdownLabel(iso, completed) {
      if (completed) return 'Selesai';
      if (!iso) return 'Aktif';
      try {
        const d = new Date(iso);
        if (isNaN(d)) return 'Aktif';
        const now = new Date();
        const diffH = Math.round((d - now) / 3600000);
        if (diffH < 0) {
          const daysOver = Math.abs(Math.floor(diffH / 24));
          return daysOver > 0 ? `Terlewat ${daysOver} Hari` : `Terlewat ${Math.abs(diffH)} Jam`;
        }
        if (diffH < 48) {
          return `Sisa ${diffH} Jam`;
        }
        const days = Math.round(diffH / 24);
        return `Sisa ${days} Hari`;
      } catch (e) {
        return 'Aktif';
      }
    },

    countdownClass(t) {
      if (t.completed_at) return 'bg-emerald-50 text-emerald-800 border-emerald-200';
      if (!t.deadline_at) return 'bg-slate-100 text-slate-700 border-slate-200';
      try {
        const d = new Date(t.deadline_at);
        if (isNaN(d)) return 'bg-slate-100 text-slate-700 border-slate-200';
        const diffH = (d - new Date()) / 3600000;
        if (diffH < 0) return 'bg-rose-50 text-rose-800 border-rose-200';
        if (diffH < 48) return 'bg-amber-50 text-amber-900 border-amber-200';
        return 'bg-slate-100 text-slate-700 border-slate-200';
      } catch (e) {
        return 'bg-slate-100 text-slate-700 border-slate-200';
      }
    },

    formatDeadlineShort(iso) {
      if (!iso) return '—';
      try {
        const d = new Date(iso);
        if (isNaN(d)) return String(iso);
        const now = new Date();
        const isToday = d.toDateString() === now.toDateString();
        const tomorrow = new Date(now);
        tomorrow.setDate(now.getDate() + 1);
        const isTomorrow = d.toDateString() === tomorrow.toDateString();

        const datePart = d.toLocaleDateString('id-ID', {
          timeZone: 'Asia/Jakarta',
          day: 'numeric',
          month: 'short'
        });
        const timePart = d.toLocaleTimeString('id-ID', {
          timeZone: 'Asia/Jakarta',
          hour: '2-digit',
          minute: '2-digit',
          hour12: false
        }).replace('.', ':') + ' WIB';

        let prefix = '';
        if (isToday) prefix = 'Hari ini, ';
        else if (isTomorrow) prefix = 'Besok, ';

        return `${prefix}${datePart} · ${timePart}`;
      } catch (e) {
        return String(iso);
      }
    },

    pubLabel(st, rev) {
      const s = String(st || '').toUpperCase();
      if (s === 'DRAFT') return 'Draf';
      if (s === 'PUBLISHED') {
        if (String(rev || '').toUpperCase() === 'NOT_REVIEWED') return 'Diajukan';
        return 'Terbit';
      }
      if (s === 'REVOKED') return 'Publikasi Dicabut';
      return st || '-';
    },

    reviewLabel(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'NOT_REVIEWED') return 'Menunggu Review KM';
      if (s === 'APPROVED') return 'Disetujui KM';
      if (s === 'CHANGES_REQUESTED') return 'Perlu Perbaikan';
      if (s === 'REVOKED') return 'Dibatalkan';
      return st || '-';
    },

    formatSubmittedTime(iso) {
      if (!iso) return 'baru saja';
      try {
        const d = new Date(iso);
        if (isNaN(d)) return String(iso);
        const now = new Date();
        const isToday = d.toDateString() === now.toDateString();
        const yesterday = new Date(now);
        yesterday.setDate(now.getDate() - 1);
        const isYesterday = d.toDateString() === yesterday.toDateString();

        const timePart = d.toLocaleTimeString('id-ID', {
          timeZone: 'Asia/Jakarta',
          hour: '2-digit',
          minute: '2-digit',
          hour12: false
        }).replace('.', ':') + ' WIB';

        if (isToday) return 'hari ini, ' + timePart;
        if (isYesterday) return 'kemarin, ' + timePart;

        const datePart = d.toLocaleDateString('id-ID', {
          timeZone: 'Asia/Jakarta',
          day: 'numeric',
          month: 'short'
        });
        return `${datePart}, ${timePart}`;
      } catch (e) {
        return String(iso);
      }
    },

    tugasMetaSub(t) {
      if (!t) return '';
      const rev = String(t.review_state || '').toUpperCase();
      const isDraft = String(t.publication_status || '').toUpperCase() === 'DRAFT';
      const matkul = t.matkul || this.offeringName() || 'Mata Kuliah';
      if (rev === 'CHANGES_REQUESTED') {
        return `${matkul} · Koreksi instruksi diminta Ketua Murid`;
      }
      if (isDraft) {
        return `${matkul} · Disimpan sebagai draf ${this.formatSubmittedTime(t.created_at)}`;
      }
      if (rev === 'NOT_REVIEWED') {
        return `${matkul} · Diajukan ke KM ${this.formatSubmittedTime(t.created_at)}`;
      }
      return `${matkul} · Pengumpulan: ${this.tugasTempat(t)} · Dosen: ${this.tugasDosen(t)}`;
    },

    tugasIconBoxClass(t) {
      if (!t) return 'bg-blue-50 text-blue-600';
      const rev = String(t.review_state || '').toUpperCase();
      const isDraft = String(t.publication_status || '').toUpperCase() === 'DRAFT';
      if (rev === 'CHANGES_REQUESTED') {
        return 'bg-[#FFF1F2] text-[#E11D48]';
      }
      if (isDraft || rev === 'NOT_REVIEWED') {
        return 'bg-[#F1F5F9] text-[#64748B]';
      }
      return 'bg-blue-50 text-blue-600';
    },

    reviewBadgeClass(t) {
      const st = String((t && (t.review_state || t.review_status)) || '').toUpperCase();
      if (st === 'CHANGES_REQUESTED') {
        return 'bg-[#FFF1F2] text-[#9F1239] border border-[#FDA4AF]';
      }
      if (st === 'NOT_REVIEWED') {
        return 'bg-[#FEF3C7] text-[#78350F] border border-[#FCD34D]';
      }
      if (st === 'APPROVED') {
        return 'bg-[#ECFDF5] text-[#065F46] border border-[#A7F3D0]';
      }
      return 'bg-slate-100 text-slate-700 border border-slate-200';
    },

    pubBadgeClass(t) {
      const st = String((t && (t.publication_status || t.status)) || '').toUpperCase();
      if (st === 'DRAFT') {
        return 'bg-[#F1F5F9] text-[#334155]';
      }
      return 'bg-[#EFF6FF] text-[#1D4ED8]';
    },

    tugasActionLabel(t) {
      if (!t) return 'Detail';
      const rev = String(t.review_state || '').toUpperCase();
      const isDraft = String(t.publication_status || '').toUpperCase() === 'DRAFT';
      if (rev === 'CHANGES_REQUESTED') return 'Perbaiki Draf';
      if (isDraft || rev === 'NOT_REVIEWED') return 'Edit Draf';
      return 'Detail';
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
        const y = d.getFullYear();
        const m = String(d.getMonth() + 1).padStart(2, '0');
        const dt = String(d.getDate()).padStart(2, '0');
        return { name: n, dateNum: d.getDate(), full: d, dateStr: `${y}-${m}-${dt}` };
      });
    },

    get weekLabel() {
      const days = this.weekDays;
      const startDay = days[0].dateNum;
      const endDay = days[4].dateNum;
      const month = days[4].full.toLocaleString('id-ID', { month: 'long', timeZone: 'Asia/Jakarta' });
      const year = days[4].full.getFullYear();
      return `${startDay} – ${endDay} ${month} ${year}`;
    },

    get pekanNum() {
      return Math.max(1, 6 + this.weekOffset);
    },

    activeSemesterLabel() {
      if (this.semesterData && this.semesterData.active) {
        const s = this.semesterData.active;
        return `${s.term === 'ODD' ? 'Semester Ganjil' : 'Semester Genap'} ${s.academic_year || ''} (Aktif)`;
      }
      return 'Semester Ganjil 2026/2027 (Aktif)';
    },

    isMatkulSaya(s) {
      if (!s) return false;
      if (this.offeringId && s.offeringId && String(s.offeringId) === String(this.offeringId)) return true;
      const sm = String(s.matkul || '').trim().toLowerCase();
      const pm = String(this.pjMatkul || '').trim().toLowerCase();
      if (pm && sm && (sm === pm || sm.includes(pm) || pm.includes(sm))) return true;
      return false;
    },

    sesiType(s) {
      if (!s) return 'Teori';
      if (s.eventKind === 'REPLACEMENT') return 'Pengganti';
      const m = String(s.matkul || '').toLowerCase();
      const act = String(s.activityType || '').toLowerCase();
      if (m.includes('praktikum') || m.includes('praktik') || act.includes('practic')) return 'Praktikum';
      return 'Teori';
    },

    sesiTypeBadgeClass(s) {
      const type = this.sesiType(s);
      if (type === 'Pengganti') return 'bg-[#FEF3C7] text-[#92400E] border border-[#FDE68A]';
      if (type === 'Praktikum') return 'bg-[#F5F3FF] text-[#6D28D9] border border-[#DDD6FE]';
      return 'bg-[#EFF6FF] text-[#1D4ED8] border border-[#BFDBFE]';
    },

    daySessionCountLabel(dayName) {
      const cnt = this.daySessions(dayName).length;
      if (cnt === 0) return '0 Sesi';
      return `${cnt} Sesi`;
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
      return (this.fullSchedule || [])
        .filter(s => s.hari === dayName)
        .filter(s => this.shellMatchesSearch('jadwal', [s.matkul, s.dosen, s.ruang]))
        .slice()
        .sort((a, b) => String(a.timeStart || '').localeCompare(String(b.timeStart || '')));
    },

    goPindah(dayName, slotHH) {
      this.mulaiUbah({ dayName: dayName, slotHH: slotHH, kind: 'REPLACEMENT' });
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
      return this.daySessions(dayName).find(s => parseInt((s.timeStart || '0').split(':')[0], 10) === parseInt(slotHH, 10));
    },

    // ===== Alur Perubahan Jadwal (mengikuti docs/design/flows/SCHEDULE_MANAGEMENT.md) =====
    // SCR-SCH-001 daftar pola · SCR-SCH-002 form · SCR-SCH-003 tinjau
    // SCR-SCH-004 detail · SCR-SCH-005 partisipasi kelas (KM pemilik)
    jadwalSub: 'daftar',
    schedViewMode: 'calendar', // 'calendar' | 'list'
    scheduleLoading: false,
    scheduleError: '',
    showRoomQuickCheck: false,
    showLabGuide: false,
    roomQuickDate: '',
    roomQuickStart: '08:40',
    roomQuickEnd: '10:20',
    roomQuickBuilding: '',
    roomQuickLoading: false,
    roomQuickResults: [],
    roomQuickError: '',
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
    showPenggantiModal: false,
    penggantiSubmitting: false,
    penggantiError: '',
    penggantiRoomLoading: false,
    penggantiRoomCandidates: [],
    penggantiForm: {
      isSessionLocked: false,
      originSessionDisplay: '',
      originKey: '',
      originPatternId: '',
      originDate: '',
      newDate: '',
      startTime: '08:40',
      endTime: '11:10',
      roomId: '',
      tuStatus: 'PENDING',
      tuContact: '',
      note: ''
    },

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

    get pjAvailableOriginSessions() {
      const list = [];
      const patterns = this.scopePatterns || [];
      const days = this.weekDays || [];
      for (const p of patterns) {
        const dow = Number(p.day_of_week);
        if (dow >= 1 && dow <= 5 && days[dow - 1]) {
          const d = days[dow - 1];
          const start = (p.start_time || '').slice(0, 5);
          const end = (p.end_time || '').slice(0, 5);
          const dateFormatted = d.dateNum + ' ' + d.full.toLocaleString('id-ID', { month: 'short', timeZone: 'Asia/Jakarta' });
          list.push({
            key: `${p.id}|${d.dateStr}|${start}|${end}`,
            patternId: p.id,
            date: d.dateStr,
            start: start,
            end: end,
            label: `${this.polaHariName(dow)}, ${dateFormatted} (${start} – ${end})`
          });
        } else {
          const start = (p.start_time || '').slice(0, 5);
          const end = (p.end_time || '').slice(0, 5);
          list.push({
            key: `${p.id}||${start}|${end}`,
            patternId: p.id,
            date: '',
            start: start,
            end: end,
            label: `${this.polaHariName(dow)} (${start} – ${end})`
          });
        }
      }
      return list;
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

    isDraftSemester() { return String(this.semesterStatus || '').toUpperCase() === 'DRAFT'; },

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

    bukaCekRuangan() {
      const now = new Date();
      const pad = n => String(n).padStart(2, '0');
      this.roomQuickDate = `${now.getFullYear()}-${pad(now.getMonth()+1)}-${pad(now.getDate())}`;
      this.showRoomQuickCheck = true;
      this.cariRuanganCepat();
    },

    async cariRuanganCepat() {
      this.roomQuickLoading = true;
      this.roomQuickError = '';
      try {
        const d = this.roomQuickDate || new Date().toISOString().slice(0, 10);
        const startsAt = `${d}T${this.roomQuickStart || '08:40'}:00Z`;
        const endsAt = `${d}T${this.roomQuickEnd || '10:20'}:00Z`;
        const [rooms, candidates] = await Promise.all([
          API.getMasterRooms('ACTIVE').catch(() => []),
          API.getRoomAvailability(startsAt, endsAt).catch(() => [])
        ]);
        const candidateSet = new Set((candidates || []).map(c => String(c.id || c.room_id)));
        
        let list = (rooms || []).map(r => {
          const isCandidate = candidateSet.size === 0 || candidateSet.has(String(r.id));
          return {
            id: r.id,
            code: r.code,
            name: r.name,
            building: r.building || 'Gedung Kuliah',
            capacity: r.capacity || 40,
            available: isCandidate
          };
        });

        if (this.roomQuickBuilding) {
          list = list.filter(r => String(r.building).toLowerCase().includes(this.roomQuickBuilding.toLowerCase()));
        }
        this.roomQuickResults = list;
      } catch (e) {
        this.roomQuickError = 'Gagal memuat ketersediaan ruangan.';
        this.roomQuickResults = [];
      } finally {
        this.roomQuickLoading = false;
      }
    },

    pilihRuanganCepat(r) {
      this.showRoomQuickCheck = false;
      this.bukaModalPengganti({
        roomId: r.id,
        date: this.roomQuickDate,
        start: this.roomQuickStart,
        end: this.roomQuickEnd
      });
    },

    bukaModalPengganti(context) {
      this.penggantiError = '';
      this.penggantiSubmitting = false;
      this.penggantiRoomCandidates = [];
      this.penggantiRoomLoading = false;

      let patId = '';
      let origDate = '';
      let start = '';
      let end = '';
      let isLocked = false;
      let display = '';
      let roomId = '';

      if (context && context.session) {
        const s = context.session;
        const d = context.day;
        start = (s.timeStart || '').slice(0, 5);
        end = (s.timeEnd || '').slice(0, 5);
        origDate = (d && d.dateStr) || s.date || '';
        roomId = s.roomId ? String(s.roomId) : '';

        const pat = (this.patternsList || []).find(p =>
          String(p.display_name || p.course_name || p.offering || '') === String(s.matkul)
        );
        if (pat) patId = String(pat.id);

        isLocked = true;
        display = `${s.hari || ''}, ${(d && d.dateNum) || ''} (${start} – ${end})`;
      } else if (context && context.roomId) {
        roomId = String(context.roomId);
        if (context.date) origDate = context.date;
        if (context.start) start = context.start;
        if (context.end) end = context.end;
      }

      this.penggantiForm = {
        isSessionLocked: isLocked,
        originSessionDisplay: display,
        originKey: patId ? `${patId}|${origDate}|${start}|${end}` : '',
        originPatternId: patId,
        originDate: origDate,
        newDate: origDate || '',
        startTime: start || '08:40',
        endTime: end || '11:10',
        roomId: roomId,
        tuStatus: 'PENDING',
        tuContact: '',
        note: ''
      };

      if (!isLocked && !this.penggantiForm.originKey && this.pjAvailableOriginSessions.length > 0) {
        const first = this.pjAvailableOriginSessions[0];
        this.penggantiForm.originKey = first.key;
        this.penggantiForm.originPatternId = String(first.patternId);
        this.penggantiForm.originDate = first.date;
        if (!this.penggantiForm.newDate) this.penggantiForm.newDate = first.date;
        if (first.start) this.penggantiForm.startTime = first.start;
        if (first.end) this.penggantiForm.endTime = first.end;
      }

      this.showPenggantiModal = true;
      this.onPenggantiTimeOrDateChange();
    },

    tutupModalPengganti() {
      this.showPenggantiModal = false;
      this.penggantiError = '';
    },

    onOriginKeyChange() {
      if (!this.penggantiForm.originKey) return;
      const [patId, origDate, start, end] = this.penggantiForm.originKey.split('|');
      this.penggantiForm.originPatternId = patId || '';
      this.penggantiForm.originDate = origDate || '';
      if (!this.penggantiForm.newDate && origDate) this.penggantiForm.newDate = origDate;
      if (start) this.penggantiForm.startTime = start;
      if (end) this.penggantiForm.endTime = end;
      this.onPenggantiTimeOrDateChange();
    },

    async onPenggantiTimeOrDateChange() {
      const f = this.penggantiForm;
      if (!f.newDate || !f.startTime || !f.endTime) return;
      if (this.durasiMenit(f.startTime, f.endTime) <= 0) return;

      const startsAt = `${f.newDate}T${f.startTime}:00+07:00`;
      const endsAt = `${f.newDate}T${f.endTime}:00+07:00`;

      this.penggantiRoomLoading = true;
      try {
        const data = await API.getRoomAvailability(startsAt, endsAt);
        this.penggantiRoomCandidates = Array.isArray(data) ? data : (data && data.candidates) ? data.candidates : [];
      } catch (e) {
        this.penggantiRoomCandidates = [];
      } finally {
        this.penggantiRoomLoading = false;
      }
    },

    async kirimPengajuanPengganti() {
      const f = this.penggantiForm;
      this.penggantiError = '';

      if (!this.offeringId) {
        this.penggantiError = 'Mata kuliah penugasan PJ belum teridentifikasi.';
        return;
      }
      if (!f.originPatternId || !f.originDate) {
        this.penggantiError = 'Pilih sesi perkuliahan asal yang akan digantikan.';
        return;
      }
      if (!f.newDate) {
        this.penggantiError = 'Pilih tanggal perkuliahan baru.';
        return;
      }
      if (!f.startTime || !f.endTime) {
        this.penggantiError = 'Isi jam mulai dan jam selesai.';
        return;
      }
      if (this.durasiMenit(f.startTime, f.endTime) <= 0) {
        this.penggantiError = 'Jam selesai harus lebih besar daripada jam mulai.';
        return;
      }
      if (!f.note || f.note.trim().length < 5) {
        this.penggantiError = 'Catatan pengajuan minimal 5 karakter (tulis alasan atau kebutuhan sesi pengganti).';
        return;
      }

      this.penggantiSubmitting = true;
      try {
        const payloadEvent = {
          owner_offering_id: Number(this.offeringId),
          event_kind: 'REPLACEMENT',
          starts_at: `${f.newDate}T${f.startTime}:00+07:00`,
          ends_at: `${f.newDate}T${f.endTime}:00+07:00`,
          origin_pattern_id: Number(f.originPatternId),
          origin_date: f.originDate,
          reason: f.note.trim()
        };
        if (f.roomId) {
          payloadEvent.room_id = Number(f.roomId);
        }

        const draf = await API.createTeachingEventDraft(payloadEvent);
        const eventId = draf && draf.id;

        if (eventId && f.roomId) {
          try {
            await API.confirmTeachingEventRoom(eventId, {
              room_id: Number(f.roomId),
              confirmation_status: f.tuStatus || 'PENDING',
              external_contact: f.tuContact ? f.tuContact.trim() : undefined,
              note: f.note ? f.note.trim() : undefined
            });
          } catch (errConfirm) {
            console.warn('Gagal mencatat konfirmasi ruangan TU:', errConfirm);
          }
        }

        this.showToast('Pengajuan kuliah pengganti berhasil diajukan ke Ketua Murid.');
        this.tutupModalPengganti();
        await Promise.all([
          this.loadSchedule(),
          this.loadEvents()
        ]);
      } catch (err) {
        this.penggantiError = err.message || 'Gagal mengajukan kuliah pengganti. Periksa tanggal dan koneksi Anda.';
      } finally {
        this.penggantiSubmitting = false;
      }
    },

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

      await AsteriskShell.mount('pj', ['dashboard', 'tugas', 'jadwal', 'rooms', 'materi', 'semester', 'audit', 'status', 'akun']);
      await this.loadPartials([
        ['pj-dashboard', '/partials/pj/view-dashboard.html'],
        ['pj-tugas', '/partials/pj/view-tugas.html'],
        ['pj-jadwal', '/partials/pj/view-jadwal.html'],
        ['pj-rooms', '/partials/common/view-rooms.html'],
        ['pj-materi', '/partials/pj/view-materi.html'],
        ['pj-semester', '/partials/pj/view-semester.html'],
        ['pj-audit', '/partials/pj/view-audit.html'],
        ['pj-status', '/partials/pj/view-status.html'],
        ['pj-akun', '/partials/pj/view-akun.html'],
      ]);
      this.updateClock();
      setInterval(() => this.updateClock(), 1000);
      await this.checkBot();
      await this.loadPJDashboard();
      this.loadClasses().catch(() => {});
      this.loadOfferings().catch(() => {});
      this.loadSchedule().catch(() => {});
      this.loadTasks().catch(() => {});
      this.loadPatterns().catch(() => {});
      this.loadEvents().catch(() => {});
      this.loadSemesterPJ().catch(() => {});
      setInterval(() => this.checkBot(), 30000);
      this.dashboardLoading = false;
    },

    async loadPJDashboard() {
      this.dashboardError = '';
      if (!this.dashboardData) this.dashboardLoading = true;
      else this.dashboardRefreshing = true;
      try {
        const data = await API.getPJDashboard(this.offeringId || '');
        this.dashboardData = data;
        if (data.class) {
          this.selectedClass = data.class.label || data.class.code;
          this.classSlug = data.class.slug;
        }
        if (data.offering && data.offering.id) {
          this.offeringId = String(data.offering.id);
          this.pjMatkul = data.offering.display_name;
          try {
            localStorage.setItem('pj_offering_id', this.offeringId);
            localStorage.setItem('pj_matkul', this.pjMatkul);
          } catch (e) {}
        }
        if (Array.isArray(data.offerings)) {
          this.offeringList = data.offerings;
        }
        this.unreadCount = (data.correction && data.correction.needed) ? data.correction.count : 0;
      } catch (e) {
        this.dashboardError = e.message || 'Ringkasan ruang kerja PJ belum dapat dimuat.';
      } finally {
        this.dashboardLoading = false;
        this.dashboardRefreshing = false;
      }
    },

    async gantiOffering(offId) {
      if (!offId || String(offId) === String(this.offeringId)) return;
      this.offeringId = String(offId);
      try {
        localStorage.setItem('pj_offering_id', this.offeringId);
      } catch (e) {}
      await this.loadPJDashboard();
      this.loadTasks().catch(() => {});
      this.loadSchedule().catch(() => {});
    },

    pjTime(iso) {
      const date = new Date(iso);
      if (Number.isNaN(date.getTime())) return '-';
      return new Intl.DateTimeFormat('id-ID', {
        hour: '2-digit', minute: '2-digit', hour12: false,
        timeZone: this.dashboardData?.class?.timezone || 'Asia/Jakarta'
      }).format(date);
    },
    kmTime(iso) { return this.pjTime(iso); },

    pjDeadlineBadge(deadlineAt) {
      if (!deadlineAt) return { text: '—', class: 'bg-slate-100 text-slate-700' };
      const d = new Date(deadlineAt);
      if (isNaN(d)) return { text: '—', class: 'bg-slate-100 text-slate-700' };
      const diffMs = d.getTime() - Date.now();
      const diffH = Math.round(diffMs / (1000 * 60 * 60));
      if (diffMs < 0) {
        return { text: 'Terlewat', class: 'bg-rose-50 text-rose-700 border border-rose-200' };
      }
      if (diffH <= 48) {
        return { text: `Sisa ${diffH} Jam`, class: 'bg-[#FFFBEB] text-[#92400E] border border-[#FDE68A]' };
      }
      const diffDays = Math.ceil(diffH / 24);
      return { text: `Sisa ${diffDays} Hari`, class: 'bg-emerald-50 text-emerald-800 border border-emerald-200' };
    },

    waktuRelatif(iso) {
      if (!iso) return 'Baru saja';
      try {
        const d = new Date(iso);
        if (isNaN(d)) return 'Baru saja';
        const sec = Math.floor((Date.now() - d.getTime()) / 1000);
        if (sec < 60) return 'Baru saja';
        const min = Math.floor(sec / 60);
        if (min < 60) return `${min} menit lalu`;
        const jam = Math.floor(min / 60);
        if (jam < 24) return `${jam} jam lalu`;
        const hari = Math.floor(jam / 24);
        return `${hari} hari lalu`;
      } catch (e) {
        return 'Baru saja';
      }
    },

    perbaikiTugas(taskId) {
      if (!taskId) return;
      this.bukaDetailTugas(taskId);
    },

    pjInitials() {
      const name = this.currentUser?.display_name || 'PJ';
      const parts = name.trim().split(/\s+/).filter(Boolean);
      if (!parts.length) return 'PJ';
      if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
      return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
    },

    pjReviewCount() {
      const corr = this.dashboardData?.correction?.count || 0;
      const pend = this.dashboardData?.metrics?.pending_review_count || 0;
      if (corr > 0 || pend > 0) return corr > 0 ? corr : pend;
      const needFix = (this.tasks || []).filter(t => String(t.review_state || '').toUpperCase() === 'CHANGES_REQUESTED').length;
      if (needFix > 0) return needFix;
      const waiting = (this.tasks || []).filter(t => String(t.review_state || '').toUpperCase() === 'NOT_REVIEWED').length;
      return waiting > 0 ? waiting : 0;
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
        if (!this.selectedClass) { this.offeringState = 'no-class'; this.offeringList = []; this.semesterStatus = ''; return; }
        const slug = this.classSlug || this.selectedClass;
        const hasil = await API.getSemestersResult(slug).catch(() => ({ ok: false, status: 0, data: [] }));
        const list = Array.isArray(hasil.data) ? hasil.data : [];
        if (!hasil.ok && hasil.status === 404) {
          this.offeringState = 'no-v1-class'; this.offeringList = []; this.semesterStatus = ''; return;
        }
        if (!hasil.ok) {
          this.offeringState = 'error'; this.offeringList = []; this.semesterStatus = ''; return;
        }
        const active = list.find(s => s.status === 'ACTIVE') || list.find(s => String(s.status || '').toUpperCase() === 'DRAFT') || list[0];
        if (!active) { this.offeringState = 'empty-semester'; this.offeringList = []; this.semesterStatus = ''; return; }
        this.semesterStatus = String(active.status || '').toUpperCase();
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
      this.scheduleLoading = true;
      this.scheduleError = '';
      try {
        const [rawPatterns, rawEvents] = await Promise.all([
          API.getPatterns({ scope: 'class' }).catch(() => []),
          API.getTeachingEvents({ scope: 'class' }).catch(() => [])
        ]);
        const seen = new Set();
        const list = [];
        (rawPatterns || []).forEach((s, i) => {
          const entry = {
            id: `pattern-${s.id || i}`,
            offeringId: s.course_offering_id,
            hari: this.polaHariName(s.day_of_week),
            jam: `${String(s.start_time || '').slice(0, 5)} - ${String(s.end_time || '').slice(0, 5)}`,
            matkul: s.display_name || s.offering || s.course_name || 'Mata Kuliah',
            dosen: s.lecturer || s.dosen || '',
            ruang: s.room || s.room_code || '',
            timeStart: String(s.start_time || '').slice(0, 5),
            timeEnd: String(s.end_time || '').slice(0, 5),
            activityType: s.activity_type || ''
          };
          const key = `${entry.hari}|${entry.timeStart}|${entry.matkul}|${entry.ruang || ''}`;
          if (seen.has(key)) return;
          seen.add(key);
          list.push(entry);
        });
        (rawEvents || []).filter(e => String(e.lifecycle_status || '').toUpperCase() === 'PUBLISHED').forEach((e, i) => {
          const start = new Date(e.starts_at), end = new Date(e.ends_at);
          if (Number.isNaN(start.getTime())) return;
          const hari = start.toLocaleDateString('id-ID', { weekday: 'long', timeZone: 'Asia/Jakarta' });
          const hm = d => d.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit', hour12: false, timeZone: 'Asia/Jakarta' });
          let originDateNote = '';
          if (e.origin_occurrence_date) {
            try {
              const od = new Date(e.origin_occurrence_date);
              originDateNote = od.toLocaleDateString('id-ID', { weekday: 'long', day: 'numeric', month: 'short', timeZone: 'Asia/Jakarta' });
            } catch (err) { originDateNote = e.origin_occurrence_date; }
          }
          list.push({
            id: `event-${e.id || i}`,
            offeringId: e.offering_id,
            hari,
            jam: `${hm(start)} - ${hm(end)}`,
            matkul: e.offering || 'Mata Kuliah',
            dosen: '',
            ruang: e.room || '',
            timeStart: hm(start),
            timeEnd: hm(end),
            eventKind: e.event_kind,
            originDateNote
          });
        });

        // Fallback jika database pola kosong: muat dari data legacy kelas jika ada
        if (list.length === 0 && this.classSlug) {
          const legacyItems = await API.fetchLegacySchedule(this.classSlug).catch(() => []);
          (legacyItems || []).forEach((item, idx) => {
            const parts = (item.jam || '').split('-');
            list.push({
              id: `leg-${idx}`,
              hari: item.hari,
              jam: item.jam,
              matkul: item.matkul,
              dosen: item.dosen,
              ruang: item.ruang,
              timeStart: (parts[0] || '').trim(),
              timeEnd: (parts[1] || '').trim()
            });
          });
        }

        this.fullSchedule = list;
      } catch (e) {
        this.fullSchedule = [];
        this.scheduleError = 'Jadwal perkuliahan belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.scheduleLoading = false;
      }
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
                   created_at: t.created_at || null,
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
      this.tugasForm = { judul: '', tanggal: '', jam: '23:59', deskripsi: '', kumpul: '', kumpulUrl: '', jenis: 'Individu' };
      this.tugasError = '';
      this.tugasSubmitting = false;
      this.tugasSubmittingDraf = false;
      this.terbitOk = false;
      this.editTugasId = ''; this.editTugasVersion = 0; this.tugasConflict = null;
      this.view = 'tambah';
      window.scrollTo({ top: 0 });
    },

    previewDeadlineText() {
      const f = this.tugasForm;
      if (!f || !f.tanggal) return 'Tenggat: Belum ditentukan';
      try {
        const parts = f.tanggal.split('-');
        if (parts.length === 3) {
          const d = new Date(Number(parts[0]), Number(parts[1]) - 1, Number(parts[2]));
          if (!isNaN(d.getTime())) {
            const dayName = d.toLocaleDateString('id-ID', { weekday: 'long' });
            const monthName = d.toLocaleDateString('id-ID', { month: 'short' });
            const jamStr = f.jam ? `${f.jam} WIB` : '23:59 WIB';
            return `Tenggat: ${dayName}, ${Number(parts[2])} ${monthName} ${parts[0]} · ${jamStr}`;
          }
        }
      } catch (e) {}
      return `Tenggat: ${f.tanggal} ${f.jam ? ('· ' + f.jam + ' WIB') : ''}`;
    },

    validasiTugasTerbit() {
      const f = this.tugasForm;
      if (!this.offeringId) return 'Pilih mata kuliah di kelas ini di Dashboard dulu.';
      if (!f.judul || f.judul.trim().length < 5) return 'Judul tugas minimal 5 karakter.';
      if (!f.tanggal || !f.jam) return 'Tanggal dan jam deadline wajib diisi.';
      if (!f.deskripsi || f.deskripsi.trim().length < 5) return 'Instruksi tugas minimal 5 karakter.';
      if (!((f.kumpul || '').trim() || (f.kumpulUrl || '').trim())) return 'Isi tempat pengumpulan (keterangan tempat atau tautan URL).';
      if (!this.parseDeadlineID(f.tanggal, f.jam)) return 'Format tanggal atau jam tidak valid. Gunakan format YYYY-MM-DD dan HH:MM.';
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
      if (!f.judul || f.judul.trim().length < 5) {
        this.tugasError = 'Judul tugas minimal 5 karakter untuk disimpan sebagai draf.';
        this.showToast(this.tugasError);
        return;
      }
      this.tugasError = '';
      this.tugasSubmittingDraf = true;
      try {
        await API.createTask(this.rakitTugasPayload('draft'));
        await this.loadTasks();
        this.tugasTab = 'draf';
        this.showToast('Draf tugas berhasil disimpan di server database.');
        this.view = 'tugas';
      } catch (err) {
        this.tugasError = err.message || 'Gagal menyimpan draf di server.';
        this.showToast(this.tugasError);
      } finally {
        this.tugasSubmittingDraf = false;
      }
    },
    async simpanDraf() { return this.simpanDrafTugas(); },

    async ajukanKeKM() {
      const err = this.validasiTugasTerbit();
      if (err) {
        this.tugasError = err;
        window.scrollTo({ top: 0, behavior: 'smooth' });
        return;
      }
      this.tugasError = '';
      this.tugasSubmitting = true;
      try {
        const res = await API.createTask(this.rakitTugasPayload('published'));
        const newId = res && res.data && res.data.id;
        await this.loadTasks();
        this.terbitOk = true;
        this.tugasPublishMsg = 'Tugas berhasil diajukan ke Ketua Murid untuk ditinjau.';
        this.tugasPreview = null;
        this.showToast('Draf tugas berhasil diajukan ke Ketua Murid.');
        if (newId) {
          await this.bukaDetailTugas(newId);
        } else {
          this.tugasTab = 'draf';
          this.view = 'tugas';
        }
      } catch (err) {
        this.tugasError = err.message || 'Gagal mengajukan draf tugas ke server.';
        window.scrollTo({ top: 0, behavior: 'smooth' });
      } finally {
        this.tugasSubmitting = false;
      }
    },

    tinjauTugas() {
      return this.ajukanKeKM();
    },

    async terbitTugas() {
      return this.ajukanKeKM();
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
          reviewer: r.reviewer || 'Ketua Murid', decision: r.decision || '',
          note: r.note || '', version: r.task_version || r.version || '',
          waktu: this.formatSubmittedTime(r.created_at) || r.created_at || ''
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
      this.initialInstructions = (d.instructions || d.deskripsi || '').trim();
      this.tugasError = '';
      this.editTugasId = d.id; this.editTugasVersion = d.version || 0;
      this.tugasConflict = null;
      this.view = 'ubah-tugas';
      window.scrollTo({ top: 0 });
    },

    isInstruksiDiubah() {
      return (this.tugasForm.deskripsi || '').trim() !== this.initialInstructions && (this.tugasForm.deskripsi || '').trim() !== '';
    },

    tugasTargetVersi() {
      return (Number(this.editTugasVersion) || (this.tugasDetail && this.tugasDetail.version) || 1) + 1;
    },

    latestReviewKM() {
      if (this.tugasReviews && this.tugasReviews.length > 0) {
        const rev = this.tugasReviews.find(r => String(r.decision || '').toUpperCase() === 'CHANGES_REQUESTED') || this.tugasReviews[0];
        return rev;
      }
      if (this.tugasKoreksi) {
        return {
          reviewer: this.tugasKoreksi.reviewer_name || 'Ketua Murid',
          note: this.tugasKoreksi.note || 'Instruksi tugas perlu disesuaikan sebelum disetujui KM.',
          version: (this.tugasDetail && this.tugasDetail.version) || 1,
          waktu: 'Kemarin, 19:40 WIB'
        };
      }
      return null;
    },

    async simpanUbahTugas(targetMode) {
      if (!this.editTugasId) return;
      if (this.tugasSubmitting) return;

      const isCurrentDraft = this.tugasDetail && String(this.tugasDetail.publication_status || '').toUpperCase() === 'DRAFT';
      const wantPublish = targetMode === 'published' || (!targetMode && !isCurrentDraft);

      if (wantPublish) {
        const err = this.validasiTugasTerbit();
        if (err) { this.tugasError = err; window.scrollTo({ top: 0 }); return; }
      } else {
        if (!this.tugasForm.judul || !this.tugasForm.judul.trim()) {
          this.tugasError = 'Judul tugas wajib diisi.';
          window.scrollTo({ top: 0 });
          return;
        }
      }
      this.tugasError = '';
      this.tugasSubmitting = true;
      try {
        const payload = this.rakitTugasPayload(wantPublish ? 'published' : 'draft');
        delete payload.offering_id;
        payload.version = Number(this.editTugasVersion) || 0;
        await API.updateTask(this.editTugasId, payload);
        await this.loadTasks();
        if (wantPublish) {
          this.showToast('Tugas berhasil diajukan ke Ketua Murid untuk ditinjau!');
        } else {
          this.showToast('Perubahan draf berhasil disimpan.');
        }
        await this.bukaDetailTugas(this.editTugasId);
      } catch (e) {
        if (e.code === 'VERSION_CONFLICT') {
          this.tugasConflict = (e.payload && (e.payload.current_data || e.payload.data)) || null;
        } else {
          this.tugasError = e.message || 'Gagal menyimpan perubahan.';
        }
      } finally {
        this.tugasSubmitting = false;
      }
    },

    async ajukanDrafKeKM() {
      const d = this.tugasDetail;
      if (!d || !d.id) return;
      if (this.tugasSubmitting) return;

      // Validasi kelengkapan data draf sebelum diajukan ke KM
      if (!d.title || !d.title.trim()) {
        this.showToast('Judul tugas wajib diisi sebelum diajukan.');
        this.mulaiUbahTugas();
        return;
      }
      if (!d.instructions || !d.instructions.trim()) {
        this.showToast('Instruksi tugas wajib diisi sebelum diajukan.');
        this.mulaiUbahTugas();
        return;
      }
      if (!d.deadline_at) {
        this.showToast('Tenggat tugas wajib diisi sebelum diajukan.');
        this.mulaiUbahTugas();
        return;
      }
      const hasSub = (d.submission_text && d.submission_text.trim()) || (d.submission_url && d.submission_url.trim());
      if (!hasSub) {
        this.showToast('Tempat atau tautan pengumpulan wajib diisi sebelum diajukan.');
        this.mulaiUbahTugas();
        return;
      }

      this.tugasSubmitting = true;
      try {
        await API.updateTask(d.id, {
          version: Number(d.version) || 0,
          save_as: 'published'
        });
        await this.loadTasks();
        this.showToast('Tugas berhasil diajukan ke Ketua Murid untuk ditinjau!');
        await this.bukaDetailTugas(d.id);
      } catch (e) {
        if (e.code === 'VERSION_CONFLICT') {
          this.showToast('Versi data tugas telah berubah di server. Memuat ulang...');
          await this.bukaDetailTugas(d.id);
        } else {
          this.showToast(e.message || 'Gagal mengajukan draf ke Ketua Murid.');
        }
      } finally {
        this.tugasSubmitting = false;
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
