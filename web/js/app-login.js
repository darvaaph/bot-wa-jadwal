/**
 * web/js/app-login.js — Masuk Pengurus: Nomor WhatsApp + Kata sandi (tanpa email).
 * Login dan pemulihan kata sandi memakai API backend.
 * ?role=km|pj|sa menentukan label dan tujuan setelah autentikasi.
 */
function loginApp() {
  return {
    ready: false,
    role: 'km',
    mode: 'masuk',
    nomor: '',
    sandi: '',
    recoveryToken: '',
    recoveryPassword: '',
    recoveryConfirm: '',
    recoverySent: false,
    lihat: false,
    formError: '',
    fieldError: '',
    ruangKerja: [],
    toast: { show: false, message: '', timer: null },

    get roleLabel() {
      return { km: 'Ketua Murid', pj: 'PJ Mata Kuliah', sa: 'System Admin' }[this.role] || 'Pengurus';
    },

    get areaUrl() {
      return { km: '/km.html', pj: '/pj.html', sa: '/system-admin.html' }[this.role] || '/km.html';
    },

    initLogin() {
      try {
        const params = new URLSearchParams(window.location.search);
        const r = (params.get('role') || 'km').toLowerCase();
        if (['km', 'pj', 'sa'].includes(r)) this.role = r;
        this.recoveryToken = params.get('recovery_token') || params.get('token') || '';
        if (this.recoveryToken && (params.get('mode') === 'recovery' || params.has('recovery_token') || params.has('token'))) this.mode = 'reset';
      } catch (e) { /* default km */ }
      this.ready = true;
    },

    normalizedIdentity() {
      let clean = this.nomor.trim().replace(/[-\s]/g, '');
      if (clean.startsWith('08')) clean = '+62' + clean.slice(1);
      else if (clean.startsWith('62')) clean = '+' + clean;
      return clean;
    },

    async mintaPemulihan() {
      this.formError = '';
      if (!this.nomorValid()) {
        this.formError = 'Masukkan nomor WhatsApp yang digunakan pada akun.';
        return;
      }
      this.loading = true;
      try {
        await API.requestPasswordRecovery(this.normalizedIdentity());
        this.recoverySent = true;
      } catch (err) {
        this.formError = err.status === 429 ? 'Terlalu banyak permintaan. Tunggu beberapa saat lalu coba lagi.' : (err.message || 'Permintaan pemulihan belum dapat diproses.');
      } finally { this.loading = false; }
    },

    async aturUlangSandi() {
      this.formError = '';
      if (this.recoveryPassword.length < 12) {
        this.formError = 'Kata sandi baru minimal 12 karakter.';
        return;
      }
      if (this.recoveryPassword !== this.recoveryConfirm) {
        this.formError = 'Konfirmasi kata sandi belum sama.';
        return;
      }
      this.loading = true;
      try {
        await API.confirmPasswordRecovery(this.recoveryToken, this.recoveryPassword);
        this.mode = 'reset-selesai';
        window.history.replaceState({}, '', '/login.html?role=' + encodeURIComponent(this.role));
      } catch (err) {
        this.formError = err.message || 'Tautan pemulihan tidak valid atau sudah kedaluwarsa.';
      } finally { this.loading = false; }
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
      if (!this.sandi || !this.sandi.trim()) {
        this.fieldError = 'sandi';
        this.formError = 'Kata sandi wajib diisi.';
        return;
      }

      this.loading = true;
      try {
        const clean = this.normalizedIdentity();

        const data = await API.login(clean, this.sandi);
        this.showToast('Berhasil masuk. Mengarahkan...');

        const daftar = (data && data.assignments) || [];
        if (daftar.length > 1) {
          this.ruangKerja = daftar;
          this.mode = 'pilih';
          this.loading = false;
          window.scrollTo({ top: 0 });
          return;
        }

        let targetUrl = this.areaUrl;
        if (daftar.length > 0) {
          targetUrl = this.urlPeran(daftar[0].role);
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

    urlPeran(role) {
      const r = String(role || '').toUpperCase();
      if (r === 'SYSTEM_ADMIN') return '/system-admin.html';
      if (r === 'KM') return '/km.html';
      if (r === 'PJ') return '/pj.html';
      return this.areaUrl;
    },

    labelPeran(role) {
      const r = String(role || '').toUpperCase();
      if (r === 'SYSTEM_ADMIN') return 'System Admin';
      if (r === 'KM') return 'Ketua Murid (KM)';
      if (r === 'PJ') return 'PJ Mata Kuliah';
      return role || 'Pengurus';
    },

    deskripsiPenugasan(a) {
      const bagian = [];
      if (a.class_slug) bagian.push('Kelas ' + a.class_slug);
      if (a.offering_name) bagian.push(a.offering_name);
      else if (String(a.role || '').toUpperCase() === 'KM') bagian.push('Semua mata kuliah');
      return bagian.join(' · ') || 'Ruang kerja pengurus';
    },

    async pilihRuangKerja(a) {
      if (!a || !a.id) return;
      this.loading = true;
      this.formError = '';
      try {
        await API.switchContext(a.id);
        window.location.href = this.urlPeran(a.role);
      } catch (err) {
        this.formError = err.message || 'Gagal berpindah ruang kerja.';
      } finally {
        this.loading = false;
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
