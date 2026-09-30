/**
 * web/js/app-login.js — Masuk Pengurus: Nomor WhatsApp + Kata sandi (tanpa email).
 * Verifikasi butuh endpoint backend — form tervalidasi lokal lalu jujur toast.
 * ?role=km|pj|sa menentukan label + area tujuan pratinjau.
 */
function loginApp() {
  return {
    role: 'km',
    mode: 'masuk',
    nomor: '',
    sandi: '',
    lihat: false,
    terkirim: false,
    formError: '',
    fieldError: '',
    toast: { show: false, message: '', timer: null },

    selectedDemoRole: '',
    demoAccounts: {
      sa: { nomor: '081111111111', sandi: 'password123', label: 'System Admin (Semua Akses)', role: 'sa' },
      km: { nomor: '081234567890', sandi: 'password123', label: 'Ketua Murid (KM)', role: 'km' },
      pj: { nomor: '081298765432', sandi: 'password123', label: 'PJ Mata Kuliah (PJ)', role: 'pj' }
    },

    get roleLabel() {
      return { km: 'Ketua Murid', pj: 'PJ Mata Kuliah', sa: 'System Admin' }[this.role] || 'Pengurus';
    },

    get areaUrl() {
      return { km: '/km.html', pj: '/pj.html', sa: '/system-admin.html' }[this.role] || '/km.html';
    },

    pilihDemo(roleKey, autoSubmit = false) {
      const acc = this.demoAccounts[roleKey];
      if (!acc) return;
      this.selectedDemoRole = roleKey;
      this.nomor = acc.nomor;
      this.sandi = acc.sandi;
      this.role = acc.role;
      this.formError = '';
      this.fieldError = '';
      this.showToast(`Akun demo ${acc.label} terisi.`);
      if (autoSubmit) {
        this.masuk();
      }
    },

    initLogin() {
      try {
        const params = new URLSearchParams(window.location.search);
        const r = (params.get('role') || 'km').toLowerCase();
        if (['km', 'pj', 'sa'].includes(r)) this.role = r;
      } catch (e) { /* default km */ }
    },

    nomorValid() {
      return this.nomor.replace(/\D/g, '').length >= 9;
    },

    loading: false,

    async masuk() {
      this.formError = '';
      this.fieldError = '';
      if (!this.nomorValid()) {
        this.fieldError = 'nomor';
        this.formError = 'Nomor WhatsApp tidak valid (minimal 9 digit).';
        return;
      }
      if (!this.sandi || this.sandi.length < 8) {
        this.fieldError = 'sandi';
        this.formError = 'Kata sandi minimal 8 karakter.';
        return;
      }

      this.loading = true;
      try {
        let clean = this.nomor.trim().replace(/[-\s]/g, '');
        if (clean.startsWith('08')) {
          clean = '+62' + clean.slice(1);
        } else if (clean.startsWith('62')) {
          clean = '+' + clean;
        }

        const data = await API.login(clean, this.sandi);
        this.showToast('Berhasil masuk. Mengarahkan...');

        let targetUrl = this.areaUrl;
        if (data && data.assignments && data.assignments.length > 0) {
          const role = (data.assignments[0].role || '').toUpperCase();
          if (role === 'SYSTEM_ADMIN') {
            targetUrl = '/system-admin.html';
          } else if (role === 'KM') {
            targetUrl = '/km.html';
          } else if (role === 'PJ') {
            targetUrl = '/pj.html';
          }
        }

        setTimeout(() => {
          window.location.href = targetUrl;
        }, 600);
      } catch (err) {
        this.formError = err.message || 'Gagal masuk. Periksa kembali nomor dan kata sandi Anda.';
        this.showToast(this.formError);
      } finally {
        this.loading = false;
      }
    },

    kirimPulih() {
      this.formError = '';
      if (!this.nomorValid()) {
        this.formError = 'Nomor WhatsApp tidak valid (minimal 9 digit).';
        return;
      }
      this.terkirim = true;
    },

    showToast(msg) {
      if (this.toast.timer) clearTimeout(this.toast.timer);
      this.toast.message = msg;
      this.toast.show = true;
      this.toast.timer = setTimeout(() => { this.toast.show = false; }, 3000);
    }
  };
}
