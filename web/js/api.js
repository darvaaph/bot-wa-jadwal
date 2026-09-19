/**
 * web/js/api.js — REST API Client Wrapper
 * Berfungsi sebagai jembatan komunikasi antara Web Dashboard dan Backend Go.
 */

const API_BASE = '';

const API = {
  /**
   * Cek kesehatan server API (/api/health)
   */
  async getHealth() {
    try {
      const res = await fetch(`${API_BASE}/api/health`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return await res.json();
    } catch (err) {
      console.warn('API getHealth error:', err);
      return { status: 'offline', uptime: '0s' };
    }
  },

  /**
   * Ambil status telemetri bot dan sistem kelas (/api/status)
   */
  async getStatus() {
    try {
      const res = await fetch(`${API_BASE}/api/status`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return await res.json();
    } catch (err) {
      console.warn('API getStatus error:', err);
      return {
        status: 'mock',
        bot_connection: 'connected',
        total_classes: 1,
        default_class: 'D4-TI-SMT3-A',
        classes: ['D4-TI-SMT3-A']
      };
    }
  },

  /**
   * Ambil daftar seluruh kelas yang terdaftar di ClassManager (/api/classes)
   */
  async getClasses() {
    try {
      const res = await fetch(`${API_BASE}/api/classes`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json = await res.json();
      return json.data || { total: 0, classes: [], default_class: '' };
    } catch (err) {
      console.warn('API getClasses error:', err);
      return {
        total: 1,
        classes: ['D4-TI-SMT3-A'],
        default_class: 'D4-TI-SMT3-A'
      };
    }
  },

  /**
   * Ambil jadwal kuliah mingguan berdasarkan kode kelas (/api/schedule)
   */
  async getSchedule(classId = '', day = '') {
    try {
      let url = `${API_BASE}/api/schedule?class=${encodeURIComponent(classId)}`;
      if (day && day !== 'Semua') {
        url += `&day=${encodeURIComponent(day)}`;
      }
      const res = await fetch(url);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json = await res.json();
      return json.data || { schedule: [], total: 0, class: classId };
    } catch (err) {
      console.warn('API getSchedule error:', err);
      return { schedule: [], total: 0, class: classId };
    }
  },

  /**
   * Ambil daftar tugas aktif yang terfilter per kelas (/api/tasks)
   */
  async getTasks(classId = '') {
    try {
      const res = await fetch(`${API_BASE}/api/tasks?class=${encodeURIComponent(classId)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json = await res.json();
      if (Array.isArray(json.data)) return json.data;
      if (Array.isArray(json)) return json;
      return [];
    } catch (err) {
      console.warn('API getTasks error:', err);
      const local = localStorage.getItem('bot_tasks_' + classId) || localStorage.getItem('bot_tasks');
      if (local) {
        try { return JSON.parse(local); } catch (e) {}
      }
      return [];
    }
  },

  /**
   * Simpan catatan tugas baru (/api/tasks)
   */
  async createTask(taskData) {
    try {
      const res = await fetch(`${API_BASE}/api/tasks`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(taskData)
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return await res.json();
    } catch (err) {
      console.warn('Backend /api/tasks POST fallback to localStorage', err);
      const classId = taskData.class_id || 'default';
      const tasks = await this.getTasks(classId);
      const newTask = {
        id: Date.now(),
        ...taskData
      };
      tasks.push(newTask);
      localStorage.setItem('bot_tasks_' + classId, JSON.stringify(tasks));
      return { status: 'success', data: newTask };
    }
  },

  /**
   * Hapus / Tandai selesai catatan tugas (/api/tasks/{id})
   */
  async deleteTask(taskId) {
    try {
      const res = await fetch(`${API_BASE}/api/tasks/${taskId}`, { method: 'DELETE' });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return await res.json();
    } catch (err) {
      console.warn('Backend /api/tasks DELETE fallback to localStorage', err);
      return { status: 'success' };
    }
  }
};
