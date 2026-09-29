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

    get roleLabel() {
      return { km: 'Ketua Murid', pj: 'PJ Mata Kuliah', sa: 'Superadmin' }[this.role] || 'Pengurus';
    },

    get areaUrl() {
      return { km: '/km.html', pj: '/pj.html', sa: '/superadmin.html' }[this.role] || '/km.html';
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

    masuk() {
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
      this.showToast('Verifikasi masuk butuh endpoint backend.');
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
