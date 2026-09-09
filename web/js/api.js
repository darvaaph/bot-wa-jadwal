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
        default_class: 'TI-2A',
        classes: ['TI-2A']
      };
    }
  },

  /**
   * Ambil daftar tugas aktif (/api/tasks)
   */
  async getTasks(classId = '') {
    try {
      const res = await fetch(`${API_BASE}/api/tasks?class=${encodeURIComponent(classId)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return await res.json();
    } catch (err) {
      console.warn('API getTasks not available yet, using local store/mock');
      // Mengambil dari localStorage jika backend /api/tasks belum diaktifkan
      const local = localStorage.getItem('bot_tasks');
      if (local) return JSON.parse(local);
      return [
        {
          id: 1,
          matkul: 'SISTEM BASIS DATA',
          deskripsi: 'Kerjakan soal latihan normalisasi 1NF sampai 3NF pada modul halaman 42.',
          deadline: 'Jumat, 11 Sep 23:59 WIB'
        },
        {
          id: 2,
          matkul: 'JARINGAN KOMPUTER',
          deskripsi: 'Simulasi subnetting VLSM menggunakan Cisco Packet Tracer dan submit file .pkt.',
          deadline: 'Senin, 14 Sep 12:00 WIB'
        }
      ];
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
      console.warn('Backend /api/tasks POST not ready yet, saving to localStorage');
      const tasks = await this.getTasks();
      const newTask = {
        id: Date.now(),
        ...taskData
      };
      tasks.push(newTask);
      localStorage.setItem('bot_tasks', JSON.stringify(tasks));
      return { status: 'success', task: newTask };
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
      console.warn('Backend /api/tasks DELETE not ready yet, deleting from localStorage');
      let tasks = await this.getTasks();
      tasks = tasks.filter(t => t.id !== taskId);
      localStorage.setItem('bot_tasks', JSON.stringify(tasks));
      return { status: 'success' };
    }
  }
};
