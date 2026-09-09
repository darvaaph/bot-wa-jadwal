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
    selectedClass: 'TI-2A',
    botOnline: true,
    botStatusText: 'Terhubung',
    metrics: {
      totalMatkul: 8,
      activeTasks: 2,
      totalGroups: 1
    },

    // Schedules
    selectedDayFilter: 'Semua',
    fullSchedule: [
      { hari: 'Senin', jam: '07:30 - 10:00', matkul: 'ALGORITMA & STRUKTUR DATA', dosen: 'Dr. Ir. Budi Santoso, M.Kom', ruang: 'Lab Komputer 2' },
      { hari: 'Senin', jam: '10:15 - 12:45', matkul: 'SISTEM BASIS DATA', dosen: 'Siti Rahmawati, M.T', ruang: 'D304' },
      { hari: 'Selasa', jam: '08:00 - 10:30', matkul: 'JARINGAN KOMPUTER', dosen: 'Ahmad Fauzi, S.Kom., M.Cs', ruang: 'Lab Jaringan' },
      { hari: 'Selasa', jam: '13:00 - 15:30', matkul: 'REKAYASA PERANGKAT LUNAK', dosen: 'Dian Permata, M.Kom', ruang: 'E201' },
      { hari: 'Rabu', jam: '07:30 - 10:00', matkul: 'MATEMATIKA DISKRIT', dosen: 'Prof. Hendra Wijaya', ruang: 'D202' },
      { hari: 'Kamis', jam: '09:00 - 11:30', matkul: 'PEMROGRAMAN BERORIENTASI OBJEK', dosen: 'Rudi Hermawan, M.T', ruang: 'Lab Komputer 1' },
      { hari: 'Jumat', jam: '07:30 - 09:30', matkul: 'BAHASA INGGRIS TEKNIK', dosen: 'Sarah Jenkins, M.Pd', ruang: 'D101' }
    ],

    // Tasks State
    tasks: [],
    newTask: {
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
      return this.fullSchedule.filter(s => s.hari === this.selectedDayFilter);
    },

    get uniqueMatkulList() {
      const set = new Set(this.fullSchedule.map(s => s.matkul));
      return Array.from(set);
    },

    // Initialization
    async initDashboard() {
      this.updateClock();
      setInterval(() => this.updateClock(), 1000);

      // Load tasks
      await this.loadTasks();

      // Poll bot health
      await this.checkBotHealth();
      setInterval(() => this.checkBotHealth(), 30000); // Tiap 30 detik
    },

    updateClock() {
      const now = new Date();
      const hariMap = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
      this.todayName = hariMap[now.getDay()];

      const pad = (n) => String(n).padStart(2, '0');
      const timeStr = `${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())} WIB`;
      this.currentTime = `${this.todayName}, ${timeStr}`;
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
        if (status.default_class) {
          this.selectedClass = status.default_class;
        }
      }
    },

    openTaskModal() {
      this.newTask = { matkul: '', deskripsi: '', deadline: '' };
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

      await API.createTask(this.newTask);
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
