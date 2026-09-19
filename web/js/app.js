/**
 * web/js/app.js — Alpine.js Dashboard App Component
 * Mengatur seluruh state reaktif antarmuka Dashboard Admin v2.0
 */

function dashboardApp() {
  return {
    activeTab: 'overview',
    mobileMenuOpen: false,
    taskModalOpen: false,
    
    tabTitles: {
      overview: 'Ringkasan Sistem & Jadwal Hari Ini',
      jadwal: 'Jadwal Kuliah Mingguan',
      tugas: 'Catatan Tugas & Praktikum',
      bot: 'Gateway & Koneksi WhatsApp'
    },

    // Clock & Date
    currentTime: '00:00:00 WIB',
    todayName: 'Senin',

    // Telemetry & Metrics
    availableClasses: [],
    selectedClass: localStorage.getItem('bot_selected_class') || 'D4-TI-SMT3-A',
    botOnline: true,
    botStatusText: 'Terhubung',
    metrics: {
      totalMatkul: 0,
      activeTasks: 0,
      totalGroups: 1
    },

    // Schedules
    selectedDayFilter: 'Semua',
    fullSchedule: [],

    // Tasks State
    tasks: [],
    newTask: {
      class_id: '',
      matkul: '',
      deskripsi: '',
      deadline: ''
    },

    // Toast State
    toast: {
      show: false,
      message: '',
      timer: null
    },

    // Computed / Getters
    get todaySchedule() {
      return this.fullSchedule.filter(s => s.hari.toLowerCase() === this.todayName.toLowerCase());
    },

    get filteredSchedule() {
      if (this.selectedDayFilter === 'Semua') {
        return this.fullSchedule;
      }
      return this.fullSchedule.filter(s => s.hari.toLowerCase() === this.selectedDayFilter.toLowerCase());
    },

    get uniqueMatkulList() {
      const set = new Set(this.fullSchedule.map(s => s.matkul || s.nama_matkul).filter(Boolean));
      return Array.from(set);
    },

    // Initialization
    async initDashboard() {
      this.updateClock();
      setInterval(() => this.updateClock(), 1000);

      // Load available classes from API
      await this.loadClasses();

      // Load schedule and tasks for current class
      await Promise.all([
        this.loadSchedule(),
        this.loadTasks(),
        this.checkBotHealth()
      ]);

      // Poll bot health every 30s
      setInterval(() => this.checkBotHealth(), 30000);
    },

    updateClock() {
      const now = new Date();
      const hariMap = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
      this.todayName = hariMap[now.getDay()];

      const pad = (n) => String(n).padStart(2, '0');
      const timeStr = `${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())} WIB`;
      this.currentTime = `${this.todayName}, ${timeStr}`;
    },

    async loadClasses() {
      const res = await API.getClasses();
      if (res && Array.isArray(res.classes) && res.classes.length > 0) {
        this.availableClasses = res.classes;
        // Jika kelas yang tersimpan di localStorage tidak ada di daftar kelas, fallback ke default
        if (!this.availableClasses.includes(this.selectedClass)) {
          this.selectedClass = res.default_class || this.availableClasses[0];
          localStorage.setItem('bot_selected_class', this.selectedClass);
        }
      } else {
        this.availableClasses = [this.selectedClass];
      }
    },

    async onClassChange(newClass) {
      if (!newClass || newClass === this.selectedClass) return;
      this.selectedClass = newClass;
      localStorage.setItem('bot_selected_class', newClass);

      await Promise.all([
        this.loadSchedule(),
        this.loadTasks()
      ]);

      this.showToast(`🏫 Beralih ke kelas ${newClass}`);
    },

    async loadSchedule() {
      const res = await API.getSchedule(this.selectedClass);
      if (res && Array.isArray(res.schedule)) {
        this.fullSchedule = res.schedule.map(s => ({
          hari: s.hari,
          jam: s.jam,
          kode_matkul: s.kode_matkul || '',
          nama_matkul: s.nama_matkul || s.matkul || '',
          matkul: s.nama_matkul || s.matkul || '',
          inisial_dosen: s.inisial_dosen || '',
          dosen: s.dosen || '',
          ruang: s.ruang || '-'
        }));
      } else {
        this.fullSchedule = [];
      }
      this.metrics.totalMatkul = this.uniqueMatkulList.length;
    },

    async loadTasks() {
      this.tasks = await API.getTasks(this.selectedClass);
      this.metrics.activeTasks = this.tasks.length;
    },

    async checkBotHealth() {
      const status = await API.getStatus();
      if (status && status.status === 'ok') {
        this.botOnline = (status.bot_connection === 'connected');
        this.botStatusText = this.botOnline ? 'Terhubung (Online)' : 'Standby / Disconnected';
        if (status.total_classes) {
          this.metrics.totalGroups = status.total_classes;
        }
      }
    },

    openTaskModal() {
      this.newTask = {
        class_id: this.selectedClass,
        matkul: '',
        deskripsi: '',
        deadline: ''
      };
      this.taskModalOpen = true;
    },

    async saveNewTask() {
      if (!this.newTask.matkul || !this.newTask.deskripsi) {
        alert('Mohon pilih mata kuliah dan isi deskripsi tugas.');
        return;
      }

      if (!this.newTask.deadline) {
        this.newTask.deadline = 'Segera';
      }

      const payload = {
        class_id: this.newTask.class_id || this.selectedClass,
        matkul: this.newTask.matkul,
        deskripsi: this.newTask.deskripsi,
        deadline: this.newTask.deadline
      };

      await API.createTask(payload);
      await this.loadTasks();

      this.taskModalOpen = false;
      this.showToast('✅ Tugas baru berhasil disimpan & disinkronkan!');
    },

    async deleteTask(id) {
      if (!confirm('Tandai tugas ini sudah selesai atau hapus?')) return;
      await API.deleteTask(id);
      await this.loadTasks();
      this.showToast('🗑️ Catatan tugas berhasil diarsipkan.');
    },

    showToast(msg) {
      if (this.toast.timer) clearTimeout(this.toast.timer);
      this.toast.message = msg;
      this.toast.show = true;
      this.toast.timer = setTimeout(() => {
        this.toast.show = false;
      }, 3000);
    }
  };
}
