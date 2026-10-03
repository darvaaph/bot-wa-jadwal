const AsteriskShell = (() => {
  const version = new URL(document.currentScript.src).searchParams.get('v') || '';

  async function mount(prefix, views) {
    const root = document.getElementById('app-shell');
    if (!root) throw new Error('Slot shell tidak ditemukan');
    try {
      const response = await fetch(`/partials/common/shell.html?v=${version}`, { cache: 'no-store' });
      if (!response.ok) throw new Error(`Shell HTTP ${response.status}`);
      const template = await response.text();
      const slots = views.map(name => `<div id="${prefix}-${name}"></div>`).join('\n');
      root.innerHTML = template.replaceAll('{{prefix}}', prefix).replace('{{viewSlots}}', slots);
    } catch (error) {
      root.innerHTML = '<main class="min-h-screen flex flex-col items-center justify-center gap-4 px-4 text-center" role="alert"><h1 class="text-[24px] font-bold">Halaman belum dapat dimuat</h1><p>Periksa koneksi, lalu coba lagi.</p><button type="button" class="min-h-[44px] px-6 rounded-xl bg-primary text-white font-semibold">Coba lagi</button></main>';
      root.querySelector('button').addEventListener('click', () => location.reload());
      throw error;
    }
  }

  function behavior(role) {
    return {
      shellRoleKey: role,
      shellDrawerTrigger: null,
      shellNavSections() { return this.navSections || []; },
      shellIdentity() {
        if (role === 'portal') return { label: 'Portal Kelas', detail: this.selectedClass || 'Pilih kelas', sub: this.semesterLabel || 'Hanya lihat', short: 'PK' };
        if (role === 'sa') return { label: 'System Admin', detail: this.currentUser?.display_name || 'Pengelola sistem', sub: this.currentUser?.identity_key || 'Akses global', short: 'SA' };
        const semester = this.activeSemesterLabel || '';
        const scope = this.roleSub || '';
        return { label: role === 'km' ? 'Ketua Murid' : 'PJ Mata Kuliah', detail: this.selectedClass || 'Pilih kelas', sub: [semester, scope].filter(Boolean).join(' · '), short: role.toUpperCase() };
      },
      shellBottomNav() {
        if (role === 'portal') return [
          { id: 'dashboard', label: 'Ringkasan', icon: 'dashboard' },
          { id: 'jadwal', label: 'Jadwal', icon: 'calendar_month' },
          { id: 'tugas', label: 'Tugas', icon: 'assignment' },
          { id: 'courses', label: 'Mata Kuliah', icon: 'menu_book' }
        ];
        if (role === 'sa') return [
          { id: 'dashboard', label: 'Ringkasan', icon: 'dashboard' },
          { id: 'kelas', label: 'Kelas', icon: 'domain' },
          { id: 'antrean', label: 'Antrean', icon: 'forward_to_inbox' },
          { id: '__more', label: 'Lainnya', icon: 'more_horiz' }
        ];
        return [
          { id: 'dashboard', label: 'Ringkasan', icon: 'dashboard' },
          { id: 'jadwal', label: 'Jadwal', icon: 'calendar_month' },
          { id: 'tugas', label: 'Tugas', icon: 'assignment' },
          { id: '__more', label: 'Lainnya', icon: 'more_horiz' }
        ];
      },
      shellBellTarget() { return role === 'sa' ? 'antrean' : role === 'portal' ? 'perubahan' : role === 'km' ? 'notifikasi' : 'status'; },
      shellAttentionCount() { return role === 'sa' ? (this.failedCount || 0) : role === 'portal' ? (this.unreadCount || 0) : (this.attentionCount || 0); },
      shellIsActive(item) { return (item.active || [item.id]).includes(this.view); },
      shellBottomIsActive(item) {
        if (this.view === '__error') return false;
        const activeNav = this.shellNavSections().flatMap(section => section.items)
          .find(navItem => this.shellIsActive(navItem));
        const activeId = activeNav?.id || this.view;
        if (item.id === '__more') {
          return role !== 'portal' && !!activeNav && !this.shellBottomNav().some(bottomItem => bottomItem.id === activeId);
        }
        return item.id === activeId;
      },
      shellMatchesSearch(view, fields) {
        if (this.view !== view) return true;
        const query = String(this.q || '').trim().toLowerCase();
        return !query || fields.some(value => String(value || '').toLowerCase().includes(query));
      },
      shellIconClass(item) { return 'ui-icon--' + (item.img || '').split('/').pop().replace(/\.svg$/, ''); },
      shellGo(item, event) {
        if (item.id === '__more') return this.shellOpenDrawer(event);
        if (role === 'sa' && item.id === 'kelas' && this.resetKelasFilter) this.resetKelasFilter();
        const fromDrawer = this.drawer;
        if (fromDrawer) this.drawer = false;
        const result = this.go(item.id);
        if (fromDrawer) {
          const focusMain = () => this.$nextTick(() => requestAnimationFrame(() => {
            document.querySelector('#main-content')?.focus();
          }));
          Promise.resolve(result).then(focusMain, focusMain);
        }
      },
      shellSearch() {
        if (role === 'sa') this.go('kelas');
        else if (this.view !== 'tugas' && this.view !== 'jadwal') this.go('tugas');
      },
      shellOpenDrawer(event) {
        this.shellDrawerTrigger = event?.currentTarget || document.activeElement;
        this.drawer = true;
        this.$nextTick(() => document.querySelector('#app-shell [data-drawer-close]')?.focus());
      },
      shellCloseDrawer() {
        if (!this.drawer) return;
        this.drawer = false;
        this.$nextTick(() => requestAnimationFrame(() => {
          const trigger = this.shellDrawerTrigger?.isConnected
            ? this.shellDrawerTrigger : document.querySelector('#app-shell [aria-label="Buka menu"]');
          trigger?.focus();
        }));
      },
      shellTrapDrawer(event) {
        if (event.key !== 'Tab') return;
        const drawer = event.currentTarget;
        const controls = [...drawer.querySelectorAll('button:not([disabled]), select:not([disabled]), a[href], input:not([disabled])')].filter(el => el.offsetParent !== null);
        if (!controls.length) return;
        const first = controls[0], last = controls[controls.length - 1];
        if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
        else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
      }
    };
  }

  return { mount, behavior, version };
})();
