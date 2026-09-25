function portalApp() {
  return {
    activeTab: 'ringkasan',
    currentRole: 'mahasiswa',
    selectedClassId: 1,
    tempClassId: 1,
    isLoading: false,

    availableClasses: MockData.classes(),
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
      switchContext: false,
      taskDetail: false
    },

    activeDetailTask: null,

    toast: {
      show: false,
      message: '',
      type: 'success'
    },

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
      return 'Mahasiswa';
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
      let list = this.tasks.filter(t => t.status === 'PUBLISHED');

      if (this.taskFilter === 'MENDESAK') {
        list = list.filter(t => !t.is_completed && isTaskUrgent(t.deadline_at));
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

    initApp() {
      const now = new Date();
      const day = now.getDay();
      this.selectedDay = (day >= 1 && day <= 5) ? day : 5;

      this.fetchClassesFromAPI();
      this.fetchTasksFromAPI();
      this.fetchScheduleForSelected();
      this.fetchTodaySchedule();
      this.fetchChangesFromAPI();
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

    closeAllModals() {
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
        const data = await BotApi.getPortalTasks(this.selectedClassSlug);
        if (data) {
          this.tasks = data;
          this.tasksLive = true;
        } else {
          this.tasksLive = false;
        }
      } catch (e) {
        this.tasksLive = false;
      } finally {
        this.isLoading = false;
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

    openTaskDetail(task) {
      this.activeDetailTask = task;
      this.modals.taskDetail = true;
    },

    openSwitchContextModal() {
      this.tempClassId = this.selectedClassId;
      this.modals.switchContext = true;
    },

    applySwitchContext() {
      this.selectedClassId = this.tempClassId;
      this.modals.switchContext = false;
      this.showToast('Kelas aktif diubah ke ' + this.selectedClassName + '.');
      this.fetchTasksFromAPI();
      this.fetchScheduleForSelected();
      this.fetchTodaySchedule();
      this.fetchChangesFromAPI();
    }
  };
}
