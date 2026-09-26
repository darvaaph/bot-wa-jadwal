function pengelolaApp() {
  return {
    activeTab: 'ringkasan',
    currentRole: 'km',
    tempRole: 'km',
    selectedClassId: 1,
    tempClassId: 1,
    isLoading: false,
    isSubmitting: false,

    authChecked: false,
    authRequired: false,
    demoMode: false,
    authUser: null,
    authAssignments: [],
    activeAssignmentId: null,
    pickedAssignmentId: null,
    pendingAssignments: [],
    loginForm: { identity_key: '', password: '' },
    authError: '',

    availableClasses: MockData.classes(),
    courseOfferings: MockData.offerings(),
    tasks: MockData.tasks(),
    tasksLive: false,
    taskFilter: 'SEMUA',
    taskSearch: '',

    weekDays: ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat'],
    selectedDay: 5,
    fullSchedule: MockData.schedule(),
    teachingEvents: MockData.events(),
    dayCache: {},
    changesLive: false,

    modals: {
      createTask: false,
      reviewTask: false,
      switchContext: false,
      taskDetail: false
    },

    activeDetailTask: null,
    activeReviewTask: null,

    newTaskForm: {
      course_offering_id: '',
      title: '',
      instructions: '',
      deadline_at: '',
      submission_target: 'Tautan LMS kelas'
    },

    reviewForm: {
      taskId: null,
      decision: 'APPROVED',
      note: ''
    },

    toast: {
      show: false,
      message: '',
      type: 'success'
    },

    waStatus: 'Memeriksa...',
    todayFormatted: 'Jumat, 25 September 2026',

    get selectedClassName() {
      const cls = this.availableClasses.find(c => c.id == this.selectedClassId);
      return cls ? cls.name : 'D4 TI 2024 A';
    },

    get selectedClassSlug() {
      const cls = this.availableClasses.find(c => c.id == this.selectedClassId);
      if (cls && cls.slug) return cls.slug;
      return 'd4-ti-2024-a';
    },

    get selectedSemesterName() {
      return 'Semester 3 (2026/2027 Ganjil)';
    },

    get roleDisplayTitle() {
      if (this.currentRole === 'km') return 'Ketua Murid';
      if (this.currentRole === 'pj') return 'PJ Mata Kuliah';
      return 'Mahasiswa';
    },

    get roleDisplayBadge() {
      if (this.authUser) return this.authUser.display_name;
      if (this.currentRole === 'km') return 'Ketua Murid · Akses Penuh';
      if (this.currentRole === 'pj') return 'PJ Mata Kuliah';
      return 'Portal Mahasiswa';
    },

    get publishedTasks() {
      return this.tasks.filter(t => t.status === 'PUBLISHED' && !t.is_completed);
    },

    get urgentTask() {
      const list = this.publishedTasks.filter(t => !t.is_completed);
      return list.length > 0 ? list[0] : null;
    },

    dateForWeekday(dayNumber) {
      const now = new Date();
      const iso = now.getDay() === 0 ? 7 : now.getDay();
      const base = new Date(now.getFullYear(), now.getMonth(), now.getDate());
      base.setDate(base.getDate() + (dayNumber - iso));
      const pad = (n) => String(n).padStart(2, '0');
      return base.getFullYear() + '-' + pad(base.getMonth() + 1) + '-' + pad(base.getDate());
    },

    todayStr() {
      const now = new Date();
      const pad = (n) => String(n).padStart(2, '0');
      return now.getFullYear() + '-' + pad(now.getMonth() + 1) + '-' + pad(now.getDate());
    },

    get todaySchedule() {
      const cached = this.dayCache[this.todayStr()];
      if (cached && cached.isLive) {
        return cached.items.map(item => ({
          jam_mulai: item.jam_mulai,
          jam_selesai: item.jam_selesai,
          mata_kuliah: item.mata_kuliah,
          ruangan: item.ruangan,
          label: item.label,
          dosen: item.dosen || []
        }));
      }
      return this.fullSchedule
        .filter(item => item.hari === this.selectedDay)
        .map(item => ({
          jam_mulai: item.waktu.split(' - ')[0],
          jam_selesai: item.waktu.split(' - ')[1] ? item.waktu.split(' - ')[1].replace(' WIB', '') : '',
          mata_kuliah: item.mata_kuliah,
          ruangan: item.ruangan,
          label: item.status_label,
          dosen: [item.pengajar]
        }));
    },

    get todayScheduleIsLive() {
      const cached = this.dayCache[this.todayStr()];
      return !!(cached && cached.isLive);
    },

    get selectedDaySchedule() {
      const cached = this.dayCache[this.dateForWeekday(this.selectedDay)];
      if (cached && cached.isLive) {
        return cached.items.map(item => ({
          waktu: item.jam_mulai + ' - ' + item.jam_selesai + ' WIB',
          kode: '',
          mata_kuliah: item.mata_kuliah,
          jenis: item.jenis,
          status_label: item.label,
          pengajar: (item.dosen || []).join(', '),
          ruangan: item.ruangan,
          berlaku: this.selectedSemesterName
        }));
      }
      return this.fullSchedule.filter(item => item.hari === this.selectedDay);
    },

    get scheduleIsLive() {
      const cached = this.dayCache[this.dateForWeekday(this.selectedDay)];
      return !!(cached && cached.isLive);
    },

    get filteredTasks() {
      let list = [...this.tasks];

      if (this.taskFilter === 'MENDESAK') {
        list = list.filter(t => !t.is_completed && isTaskUrgent(t.deadline_at));
      } else if (this.taskFilter === 'DRAFT') {
        list = list.filter(t => t.status === 'DRAFT');
      } else if (this.taskFilter === 'SELESAI') {
        list = list.filter(t => t.is_completed);
      }

      if (this.taskSearch.trim() !== '') {
        const q = this.taskSearch.toLowerCase();
        list = list.filter(t =>
          t.title.toLowerCase().includes(q) ||
          (t.course_name && t.course_name.toLowerCase().includes(q)) ||
          (t.course_code && t.course_code.toLowerCase().includes(q))
        );
      }

      return list;
    },

    get reviewPendingTasks() {
      return this.tasks.filter(t => t.review_status === 'NOT_REVIEWED');
    },

    get pendingReviewCount() {
      return this.reviewPendingTasks.length;
    },

    async initApp() {
      const now = new Date();
      const day = now.getDay();
      this.selectedDay = (day >= 1 && day <= 5) ? day : 5;

      await this.bootstrapAuth();
      if (this.authRequired && !this.authUser) {
        this.authChecked = true;
        return;
      }
      this.authChecked = true;
      await this.refreshAll();
    },

    async bootstrapAuth() {
      try {
        const session = await BotApi.getSession();
        if (session && session.authenticated) {
          this.applyPrincipal(session.user, session.assignments, session.activeRoleAssignmentId);
          return;
        }
        if (session && session.unavailable) {
          this.demoMode = true;
          return;
        }
      } catch (e) {
        // Network failure keeps the login gate closed with an error below.
      }
      this.authRequired = true;
    },

    applyPrincipal(user, assignments, activeAssignmentId) {
      this.authUser = user;
      this.authAssignments = assignments || [];
      this.activeAssignmentId = activeAssignmentId || null;
      this.pickedAssignmentId = this.activeAssignmentId;
      this.authRequired = false;
      this.demoMode = false;
      const active = this.authAssignments.find(a => a.id === activeAssignmentId) || this.authAssignments[0];
      if (active) {
        this.currentRole = String(active.role || '').toLowerCase() === 'pj' ? 'pj' : 'km';
        this.tempRole = this.currentRole;
      }
    },

    assignmentLabel(assignment) {
      const role = assignment.role === 'PJ' ? 'PJ Mata Kuliah' : 'Ketua Murid';
      let scope = '';
      if (assignment.class_id) {
        const cls = this.availableClasses.find(c => c.id == assignment.class_id);
        scope = cls ? cls.name : ('Kelas #' + assignment.class_id);
      }
      if (assignment.course_offering_id) {
        scope += ' · Offering #' + assignment.course_offering_id;
      }
      return role + (scope ? ' — ' + scope : '');
    },

    async doLogin() {
      this.authError = '';
      if (!this.loginForm.identity_key || !this.loginForm.password) {
        this.authError = 'Identitas dan kata sandi wajib diisi.';
        return;
      }
      this.isSubmitting = true;
      try {
        const data = await BotApi.login(this.loginForm.identity_key, this.loginForm.password);
        this.loginForm.password = '';
        if (data.context_selection_required && data.assignments && data.assignments.length > 1) {
          this.authUser = data.user;
          this.authAssignments = data.assignments;
          this.pickedAssignmentId = data.assignments[0].id;
          this.modals.switchContext = true;
          return;
        }
        this.applyPrincipal(data.user, data.assignments, data.active_role_assignment_id);
        await this.refreshAll();
      } catch (e) {
        this.authError = e.message || 'Gagal masuk. Periksa kembali identitas Anda.';
      } finally {
        this.isSubmitting = false;
      }
    },

    async doLogout() {
      await BotApi.logout();
      this.authUser = null;
      this.authAssignments = [];
      this.authRequired = true;
      this.currentRole = 'km';
      this.showToast('Anda telah keluar. Silakan masuk kembali.');
    },

    async refreshAll() {
      await this.fetchClassesFromAPI();
      await this.fetchTasksFromAPI();
      await this.fetchBotStatus();
      await this.fetchScheduleForSelected();
      await this.fetchTodaySchedule();
      await this.fetchChangesFromAPI();
    },

    switchTab(tabName) {
      this.activeTab = tabName;
      window.scrollTo({ top: 0, behavior: 'smooth' });
    },

    selectDay(dayNumber) {
      this.selectedDay = dayNumber;
      this.fetchScheduleForSelected();
    },

    showToast(message, type = 'success') {
      this.toast.message = message;
      this.toast.type = type;
      this.toast.show = true;
      setTimeout(() => { this.toast.show = false; }, 4000);
    },

    showSessionExpired() {
      this.authUser = null;
      this.authRequired = true;
      this.showToast('Sesi berakhir atau belum masuk. Silakan masuk kembali.', 'error');
    },

    closeAllModals() {
      this.modals.createTask = false;
      this.modals.reviewTask = false;
      this.modals.switchContext = false;
      this.modals.taskDetail = false;
    },

    resetTaskFilters() {
      this.taskFilter = 'SEMUA';
      this.taskSearch = '';
    },

    getScheduleCountForDay(dayNumber) {
      return this.fullSchedule.filter(i => i.hari === dayNumber).length;
    },

    async fetchClassesFromAPI() {
      try {
        const data = await BotApi.getClasses();
        if (data) this.availableClasses = data;
      } catch (e) {
        // Keep prototype data on fallback
      }
    },

    async fetchTasksFromAPI() {
      try {
        this.isLoading = true;
        const data = await BotApi.getTasks(this.selectedClassId);
        if (data) {
          this.tasks = data;
          this.tasksLive = true;
        } else {
          this.tasksLive = false;
        }
      } catch (e) {
        this.tasksLive = false;
        if (e && e.code === 'UNAUTHORIZED') {
          this.showSessionExpired();
        } else {
          this.showToast('Gagal memuat tugas dari server. Menampilkan data contoh.', 'error');
        }
      } finally {
        this.isLoading = false;
      }
    },

    async fetchBotStatus() {
      try {
        const json = await BotApi.getStatus();
        if (json && json.bot_connection) {
          this.waStatus = json.bot_connection === 'connected' ? 'Terhubung' : 'Standby';
        } else {
          this.waStatus = 'Tidak diketahui';
        }
      } catch (e) {
        this.waStatus = 'Tidak diketahui';
      }
    },

    async fetchDaySchedule(dateStr) {
      try {
        const data = await BotApi.getSchedule(this.selectedClassSlug, dateStr);
        if (data && data.jadwal) {
          this.dayCache[dateStr] = { isLive: true, items: data.jadwal };
          return true;
        }
      } catch (e) {
        // Fall through to prototype data below.
      }
      if (!this.dayCache[dateStr]) {
        this.dayCache[dateStr] = { isLive: false, items: [] };
      } else {
        this.dayCache[dateStr].isLive = false;
      }
      return false;
    },

    async fetchScheduleForSelected() {
      await this.fetchDaySchedule(this.dateForWeekday(this.selectedDay));
    },

    async fetchTodaySchedule() {
      await this.fetchDaySchedule(this.todayStr());
    },

    async fetchChangesFromAPI() {
      try {
        const items = await BotApi.getChanges(this.selectedClassSlug, 20);
        if (items) {
          this.teachingEvents = items.map(item => ({
            jenis: item.label || 'Perubahan Jadwal',
            mata_kuliah: item.mata_kuliah,
            jadwal_semula: item.jadwal_semula || '',
            jadwal_baru: (item.jadwal_baru || '') + (item.ruangan ? ', ' + item.ruangan : ''),
            alasan: item.keterangan || '',
            status_publikasi: 'Terbit',
            status_wa: ''
          }));
          this.changesLive = true;
          return;
        }
      } catch (e) {
        // Fall through to prototype data below.
      }
      this.changesLive = false;
    },

    async refreshData() {
      await this.fetchTasksFromAPI();
      await this.fetchScheduleForSelected();
      await this.fetchTodaySchedule();
      await this.fetchChangesFromAPI();
      this.showToast('Data berhasil diperbarui dari server.');
    },

    openCreateTaskModal() {
      this.newTaskForm = {
        course_offering_id: this.courseOfferings[0] ? this.courseOfferings[0].id : '',
        title: '',
        instructions: '',
        deadline_at: '2026-09-30T23:59',
        submission_target: 'Tautan LMS kelas'
      };
      this.modals.createTask = true;
    },

    wibToUtcIso(localValue) {
      const raw = String(localValue || '').trim();
      if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(raw)) return '';
      const asUtc = new Date(raw + ':00+07:00');
      if (Number.isNaN(asUtc.getTime())) return '';
      return asUtc.toISOString();
    },

    async submitCreateTask() {
      if (!this.newTaskForm.title || !this.newTaskForm.course_offering_id) {
        this.showToast('Judul dan mata kuliah wajib diisi.', 'error');
        return;
      }
      const deadlineIso = this.wibToUtcIso(this.newTaskForm.deadline_at);
      if (!deadlineIso) {
        this.showToast('Tenggat waktu tidak valid.', 'error');
        return;
      }
      this.isSubmitting = true;

      const payload = {
        course_offering_id: parseInt(this.newTaskForm.course_offering_id),
        title: this.newTaskForm.title,
        instructions: this.newTaskForm.instructions,
        deadline_at: deadlineIso,
        task_type: 'INDIVIDUAL',
        submission_text: this.newTaskForm.submission_target
      };

      try {
        await BotApi.createTask(payload);
        this.showToast('Tugas baru berhasil disimpan dan diterbitkan.');
        this.modals.createTask = false;
        await this.fetchTasksFromAPI();
      } catch (err) {
        if (err && err.code === 'UNAUTHORIZED') {
          this.modals.createTask = false;
          this.showSessionExpired();
        } else {
          this.showToast('Gagal menyimpan tugas. Periksa isian lalu coba lagi.', 'error');
        }
      } finally {
        this.isSubmitting = false;
      }
    },

    openReviewModal(task, defaultDecision = 'APPROVED') {
      this.activeReviewTask = task;
      this.reviewForm = {
        taskId: task.id,
        decision: defaultDecision,
        note: ''
      };
      this.modals.reviewTask = true;
    },

    async submitReviewModal() {
      if (!this.reviewForm.taskId) return;
      if (this.reviewForm.decision !== 'APPROVED' && !String(this.reviewForm.note || '').trim()) {
        this.showToast('Catatan wajib diisi untuk koreksi atau pembatalan.', 'error');
        return;
      }
      this.isSubmitting = true;

      try {
        await BotApi.reviewTask(this.reviewForm.taskId, {
          decision: this.reviewForm.decision,
          note: this.reviewForm.note ? this.reviewForm.note : null
        });
        this.showToast('Hasil pemeriksaan KM berhasil disimpan.');
        this.modals.reviewTask = false;
        await this.fetchTasksFromAPI();
      } catch (err) {
        if (err && err.code === 'UNAUTHORIZED') {
          this.modals.reviewTask = false;
          this.showSessionExpired();
        } else {
          this.showToast('Gagal menyimpan hasil pemeriksaan. Catatan Anda tetap tersimpan di formulir ini.', 'error');
        }
      } finally {
        this.isSubmitting = false;
      }
    },

    async submitDirectReview(taskId, decision) {
      this.isSubmitting = true;
      try {
        await BotApi.reviewTask(taskId, { decision: decision, note: null });
        this.showToast('Tugas berhasil disetujui dan diterbitkan.');
        await this.fetchTasksFromAPI();
      } catch (err) {
        if (err && err.code === 'UNAUTHORIZED') {
          this.showSessionExpired();
        } else {
          this.showToast('Gagal menyimpan persetujuan di server.', 'error');
        }
      } finally {
        this.isSubmitting = false;
      }
    },

    async markTaskComplete(taskId) {
      try {
        await BotApi.completeTask(taskId);
        this.showToast('Tugas ditandai selesai.');
        await this.fetchTasksFromAPI();
      } catch (err) {
        if (err && err.code === 'UNAUTHORIZED') {
          this.showSessionExpired();
        } else {
          this.showToast('Gagal menandai selesai di server.', 'error');
        }
      }
    },

    openTaskDetail(task) {
      this.activeDetailTask = task;
      this.modals.taskDetail = true;
    },

    openSwitchContextModal() {
      this.tempRole = this.currentRole;
      this.tempClassId = this.selectedClassId;
      this.pickedAssignmentId = this.activeAssignmentId;
      this.pendingAssignments = [];
      this.modals.switchContext = true;
    },

    async applySwitchContext() {
      if (this.authUser && !this.demoMode) {
        if (!this.pickedAssignmentId) {
          this.showToast('Pilih konteks akses terlebih dahulu.', 'error');
          return;
        }
        try {
          await BotApi.switchContext(this.pickedAssignmentId);
          const session = await BotApi.getSession();
          if (session && session.authenticated) {
            this.applyPrincipal(session.user, session.assignments, session.activeRoleAssignmentId);
          }
          this.modals.switchContext = false;
          this.showToast('Konteks akses diubah ke ' + this.roleDisplayTitle + '.');
          await this.refreshAll();
        } catch (err) {
          this.showToast('Gagal mengganti konteks akses.', 'error');
        }
        return;
      }
      if (this.tempRole === 'mahasiswa') {
        window.location.href = './index.html';
        return;
      }
      this.currentRole = this.tempRole;
      this.selectedClassId = this.tempClassId;
      this.modals.switchContext = false;
      this.showToast('Wewenang aktif diubah ke ' + this.roleDisplayTitle + '.');
      this.fetchTasksFromAPI();
    }
  };
}
