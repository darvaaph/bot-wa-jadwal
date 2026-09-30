/**
 * web/js/app-invite.js — Aktivasi undangan pengurus (KM/PJ) dari tautan WhatsApp.
 * Backend: POST /api/v1/invitations/accept {token, password, display_name}.
 * Accept tidak mengembalikan sesi, jadi setelah berhasil pengguna diarahkan masuk.
 */
function inviteApp() {
  return {
    token: '',
    nama: '',
    sandi: '',
    konfirmasi: '',
    lihat: false,
    loading: false,
    formError: '',
    done: false,

    initInvite() {
      try {
        const params = new URLSearchParams(window.location.search);
        this.token = (params.get('token') || '').trim();
      } catch (e) {
        this.token = '';
      }
    },

    async aktifkan() {
      this.formError = '';
      if (!this.token) {
        this.formError = 'Tautan undangan tidak lengkap. Minta tautan baru kepada pengundang.';
        return;
      }
      if (!this.nama || this.nama.trim().length < 3) {
        this.formError = 'Isi nama lengkap (minimal 3 karakter).';
        return;
      }
      if (!this.sandi || this.sandi.length < 12) {
        this.formError = 'Kata sandi minimal 12 karakter.';
        return;
      }
      if (this.sandi !== this.konfirmasi) {
        this.formError = 'Konfirmasi kata sandi tidak sama.';
        return;
      }
      this.loading = true;
      try {
        const res = await API.acceptInvitation(this.token, this.sandi, this.nama.trim());
        if (res && res.token) {
          API.setAuthToken(res.token);
          const daftar = (res && res.assignments) || [];
          const peran = daftar.length > 0 ? String(daftar[0].role || '').toUpperCase() : '';
          const tujuan = peran === 'SYSTEM_ADMIN' ? '/system-admin.html' : (peran === 'KM' ? '/km.html' : '/pj.html');
          window.location.href = tujuan;
          return;
        }
        this.done = true;
        window.scrollTo({ top: 0 });
      } catch (err) {
        const msg = String((err && err.message) || '');
        if (/kedaluwarsa/i.test(msg)) {
          this.formError = 'Tautan undangan sudah kedaluwarsa. Minta tautan baru kepada pengundang.';
        } else if (/digunakan|tidak valid/i.test(msg)) {
          this.formError = 'Tautan undangan sudah tidak berlaku atau sudah digunakan. Minta tautan baru kepada pengundang.';
        } else if (err && err.status === 429) {
          this.formError = 'Terlalu banyak percobaan. Tunggu beberapa saat lalu coba lagi.';
        } else {
          this.formError = msg || 'Gagal mengaktifkan akun. Periksa koneksi lalu coba lagi.';
        }
      } finally {
        this.loading = false;
      }
    }
  };
}
