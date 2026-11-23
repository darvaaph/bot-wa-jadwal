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
    inviteToken: '',
    recoveryPassword: '',
    recoveryConfirm: '',
    recoverySent: false,
    lihat: false,
    formError: '',
    fieldError: '',
    ruangKerja: [],
    toast: { show: false, message: '', timer: null },
    currentTheme: 'light',
    isDarkMode: false,

    get roleLabel() {
      return { km: 'Ketua Murid', pj: 'PJ Mata Kuliah', sa: 'System Admin' }[this.role] || 'Pengurus';
    },

    get areaUrl() {
      return { km: '/km.html', pj: '/pj.html', sa: '/system-admin.html' }[this.role] || '/km.html';
    },

    initLogin() {
      this.initTheme();
      try {
        const params = new URLSearchParams(window.location.search);
        this.inviteToken = params.get('invite_token') || '';
        const r = (params.get('role') || 'km').toLowerCase();
        if (['km', 'pj', 'sa'].includes(r)) this.role = r;
        this.recoveryToken = params.get('recovery_token') || params.get('token') || '';
        if (this.recoveryToken && (params.get('mode') === 'recovery' || params.has('recovery_token') || params.has('token'))) this.mode = 'reset';
      } catch (e) { /* default km */ }
      this.ready = true;
    },

    initTheme() {
      document.documentElement.classList.remove('dark');
      this.currentTheme = 'light';
      this.isDarkMode = false;
    },

    applyTheme(theme, animate = false) {
      this.currentTheme = 'light';
      this.isDarkMode = false;
      document.documentElement.classList.remove('dark');
    },

    cycleTheme() {
      this.applyTheme('light', false);
    },

    themeLabel() {
      return 'Terang';
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
        if (this.inviteToken) {
          window.location.href = '/invite.html?token=' + encodeURIComponent(this.inviteToken) + '&existing=1';
          return;
        }
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

    formatKelas(raw) {
      if (!raw) return '';
      const m = String(raw).trim().match(/^([A-Za-z0-9]+)-([A-Za-z0-9]+)-(\d{4})-([A-Za-z0-9]+)$/i);
      if (m) {
        const cohort = parseInt(m[3], 10);
        const currentYear = new Date().getFullYear();
        const tingkat = Math.max(1, Math.min(4, currentYear - cohort + 1));
        return `${m[1].toUpperCase()}-${m[2].toUpperCase()} ${tingkat}${m[4].toUpperCase()}`;
      }
      return String(raw).toUpperCase();
    },

    deskripsiPenugasan(a) {
      const bagian = [];
      const kelas = this.formatKelas(a.class_name || a.class_slug);
      if (kelas) bagian.push('Kelas ' + kelas);
      if (a.offering_name) bagian.push(a.offering_name);
      else if (String(a.role || '').toUpperCase() === 'KM') bagian.push('Koordinator Kelas');
      return bagian.join(' · ') || 'Ruang kerja pengurus';
    },

    roleIconSvg(role) {
      const r = String(role || '').toUpperCase();
      if (r === 'KM') {
        return `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3 17L5 7L9.5 12L12 5L14.5 12L19 7L21 17H3Z"/><path d="M3 17V20C3 20.6 3.4 21 4 21H20C20.6 21 21 20.6 21 20V17"/><circle cx="12" cy="13" r="1.25" fill="currentColor" stroke="none"/></svg>`;
      }
      if (r === 'PJ') {
        return `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3L2 8.5L12 14L22 8.5L12 3Z"/><path d="M6 11V16.5C6 18.5 8.7 20.5 12 20.5C15.3 20.5 18 18.5 18 16.5V11"/><path d="M22 8.5V15.5C22 16 21.6 16.5 21 16.5H20"/><circle cx="20" cy="17" r="1" fill="currentColor"/></svg>`;
      }
      return `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 2L4 5.5V11.5C4 16.8 7.4 21.7 12 23C16.6 21.7 20 16.8 20 11.5V5.5L12 2Z"/><circle cx="12" cy="11.5" r="2.5"/><path d="M12 6.5V9M12 14V16.5M7 11.5H9.5M14.5 11.5H17"/></svg>`;
    },

    roleBadgeClasses(role) {
      const r = String(role || '').toUpperCase();
      if (r === 'KM') return 'bg-blue-50 text-[#3965FB] border-blue-200';
      if (r === 'PJ') return 'bg-emerald-50 text-[#059669] border-emerald-200';
      return 'bg-indigo-50 text-[#6366F1] border-indigo-200';
    },

    roleCardBorderClass(role) {
      const r = String(role || '').toUpperCase();
      if (r === 'KM') return 'hover:border-[#3965FB] focus-visible:ring-[#3965FB]/20';
      if (r === 'PJ') return 'hover:border-[#059669] focus-visible:ring-[#059669]/20';
      return 'hover:border-[#6366F1] focus-visible:ring-[#6366F1]/20';
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

    bukaPortalMahasiswa() {
      try {
        const saved = localStorage.getItem('portal_class');
        if (saved) {
          window.location.href = '/c/' + encodeURIComponent(saved);
          return;
        }
      } catch (e) { }
      window.location.href = '/c/';
    },

    showToast(msg) {
      if (this.toast.timer) clearTimeout(this.toast.timer);
      this.toast.message = msg;
      this.toast.show = true;
      this.toast.timer = setTimeout(() => { this.toast.show = false; }, 3000);
    }
  };
}
