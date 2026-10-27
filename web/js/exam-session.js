function examSessionBehavior() {
  return {
    examStep: 1, // 1 = Form Input, 2 = Pratinjau & Konfirmasi
    examForm: { kind: 'EXAM_UTS', offeringId: '', patternId: '', originDate: '', date: '', start: '', end: '', roomId: '', reason: '' },
    examPatterns: [], examBusy: false, examError: '', examDraft: null, examPreview: null,
    examRoomConfirmed: false, examRoomContact: '', examOverride: '',

    getExamOfferingName() {
      const o = (this.offeringList || []).find(x => String(x.id) === String(this.examForm.offeringId));
      return o ? (o.display_name || o.course_name) : 'Mata Kuliah';
    },

    getExamRoomName() {
      if (!this.examForm.roomId) return 'Belum ditentukan / Daring';
      const r = (this.polaRooms || []).find(x => String(x.id) === String(this.examForm.roomId));
      return r ? (r.code || r.name) : 'Ruangan #' + this.examForm.roomId;
    },

    getExamPatternLabel() {
      if (!this.examForm.patternId) return 'Sesi tambahan (di luar jam kuliah rutin)';
      const p = (this.examPatterns || []).find(x => String(x.id) === String(this.examForm.patternId));
      if (!p) return 'Menggantikan sesi kuliah rutin';
      return `Menggantikan sesi ${this.polaHariName(p.day_of_week)}, ${p.start_time} - ${p.end_time}` +
        (this.examForm.originDate ? ` (tanggal ${this.examForm.originDate})` : '');
    },

    kembaliKeForm() {
      this.examStep = 1;
      this.examError = '';
    },

    async bukaUjian(session, day) {
      if (session && !session.patternId) return;
      if (this.roleLabel === 'PJ' && session && String(session.offeringId) !== String(this.offeringId)) return;
      const date = (day && day.dateStr) || (session && session.dateStr) ||
        ((this.weekDays || []).find(d => d.name === (session && session.hari)) || {}).dateStr || '';
      this.examForm = { kind: 'EXAM_UTS', offeringId: String((session && session.offeringId) || this.offeringId || ''),
        patternId: String((session && session.patternId) || ''), originDate: session ? date : '', date,
        start: (session && session.timeStart) || '', end: (session && session.timeEnd) || '', roomId: '', reason: '' };
      this.examStep = 1;
      this.examDraft = null; this.examPreview = null; this.examError = '';
      this.examRoomConfirmed = false; this.examRoomContact = ''; this.examOverride = '';
      if (this.tutupModalDetailSesi) this.tutupModalDetailSesi();
      if (this.tutupDetailSesi) this.tutupDetailSesi();
      document.getElementById('exam-dialog').showModal();
      this.examBusy = true;
      try {
        this.examPatterns = await API.getPatterns();
        const p = this.examPatterns.find(p => String(p.id) === this.examForm.patternId);
        if (p) {
          this.examForm.offeringId = String(p.course_offering_id || p.offering_id);
          this.examForm.roomId = p.room_id ? String(p.room_id) : '';
        }
      } catch (err) { this.examError = err.message || 'Jadwal asal gagal dimuat. Tutup lalu coba lagi.'; }
      finally { this.examBusy = false; }
    },

    async bukaDrafUjian(ev) {
      await this.bukaUjian();
      this.examBusy = true;
      try {
        const detail = await API.getTeachingEvent(ev.id);
        const e = detail.event;
        const local = raw => new Date(raw).toLocaleString('sv-SE', { timeZone: 'Asia/Jakarta' });
        const start = local(e.starts_at), end = local(e.ends_at);
        this.examForm = { kind: e.event_kind, offeringId: String(e.owner_offering_id || ev.owner_offering_id || this.offeringId || ''),
          patternId: String(e.origin_pattern_id || e.origin_schedule_pattern_id || ''), originDate: e.origin_occurrence_date || '',
          date: start.slice(0, 10), start: start.slice(11, 16), end: end.slice(11, 16),
          roomId: e.room_id ? String(e.room_id) : '', reason: e.reason || '' };
        this.examDraft = e;
        this.examPreview = await API.previewTeachingEvent(e.id);
        if (!this.examPreview) throw new Error('Pratinjau gagal dimuat. Coba tinjau lagi.');
        this.examStep = 2;
      } catch (err) { this.examError = err.message || 'Draf ujian gagal dimuat.'; }
      finally { this.examBusy = false; }
    },

    pilihAsalUjian() {
      const p = this.examPatterns.find(p => String(p.id) === this.examForm.patternId);
      if (!p) { this.examForm.originDate = ''; return; }
      this.examForm.start = String(p.start_time || '').slice(0, 5);
      this.examForm.end = String(p.end_time || '').slice(0, 5);
      this.examForm.roomId = p.room_id ? String(p.room_id) : '';
      const day = (this.weekDays || []).find(d => d.name === this.polaHariName(p.day_of_week));
      this.examForm.originDate = (day && day.dateStr) || '';
      this.examForm.date = this.examForm.originDate;
    },

    getMatchingPattern() {
      if (!this.examForm.date || !this.examForm.offeringId) return null;
      const parts = this.examForm.date.split('-');
      if (parts.length !== 3) return null;
      const d = new Date(Number(parts[0]), Number(parts[1]) - 1, Number(parts[2]));
      const jsDay = d.getDay();
      const dow = jsDay === 0 ? 7 : jsDay;
      const patterns = (this.examPatterns || []).filter(p =>
        String(p.course_offering_id || p.offering_id) === String(this.examForm.offeringId) &&
        Number(p.day_of_week) === dow
      );
      if (patterns.length === 0) return null;
      if (patterns.length === 1 || !this.examForm.start) return patterns[0];
      const exactMatch = patterns.find(p => String(p.start_time || '').startsWith(this.examForm.start));
      return exactMatch || patterns[0];
    },

    terapkanPolaCocok() {
      const p = this.getMatchingPattern();
      if (!p) return;
      this.examForm.patternId = String(p.id);
      this.examForm.originDate = this.examForm.date;
      this.examForm.start = String(p.start_time || '').slice(0, 5);
      this.examForm.end = String(p.end_time || '').slice(0, 5);
      if (p.room_id) this.examForm.roomId = String(p.room_id);
    },

    hasBlockingConflicts() {
      return Boolean(this.examPreview && (this.examPreview.conflicts || []).some(c => c.blocking));
    },

    async simpanUjian(preview) {
      if (this.examBusy) return;
      const f = this.examForm;
      if (!f.offeringId || !f.date || !f.start || !f.end || f.start >= f.end || (f.patternId && !f.originDate)) {
        this.examError = 'Lengkapi mata kuliah, tanggal, jam selesai setelah jam mulai, dan tanggal asal bila memakai jadwal rutin.';
        return;
      }
      this.examBusy = true; this.examError = '';
      try {
        if (this.examDraft) {
          try {
            await API.deleteTeachingEvent(this.examDraft.id, this.examDraft.version || 1);
          } catch (e) { /* ignore cleanup error */ }
          this.examDraft = null;
        }
        const payload = { owner_offering_id: Number(f.offeringId), event_kind: f.kind,
          starts_at: new Date(`${f.date}T${f.start}:00+07:00`).toISOString(),
          ends_at: new Date(`${f.date}T${f.end}:00+07:00`).toISOString(),
          room_id: f.roomId ? Number(f.roomId) : null, reason: f.reason.trim() || null };
        if (f.patternId) { payload.origin_pattern_id = Number(f.patternId); payload.origin_date = f.originDate; }
        this.examDraft = await API.createTeachingEventDraft(payload);
        if (preview) {
          this.examPreview = await API.previewTeachingEvent(this.examDraft.id);
          if (!this.examPreview) throw new Error('Pratinjau gagal dimuat. Draf sudah tersimpan, coba tinjau lagi.');
          this.examStep = 2;
        } else {
          document.getElementById('exam-dialog').close();
          this.showToast('Draf ujian tersimpan. Lanjutkan publikasi dari daftar perubahan.');
        }
        await this.loadEvents();
      } catch (err) { this.examError = err.message || 'Draf ujian gagal disimpan.'; }
      finally { this.examBusy = false; }
    },

    async terbitkanUjian() {
      if (this.examBusy || !this.examDraft || !this.examPreview) return;
      const conflicts = this.examPreview.conflicts || [];
      if (conflicts.some(c => c.blocking)) { this.examError = 'Konflik memblokir publikasi. Klik tombol "Ubah Data Ujian" untuk memindahkan jam/tanggal.'; return; }
      if (this.examForm.roomId && (!this.examRoomConfirmed || !this.examRoomContact.trim())) {
        this.examError = 'Catat konfirmasi ruangan dari TU dan nama kontaknya sebelum publikasi.'; return;
      }
      if (conflicts.some(c => !c.blocking && c.code !== 'ROOM_CONFIRMATION_REQUIRED') && !this.examOverride.trim()) {
        this.examError = 'Isi alasan pengecualian konflik ruangan/sesi.'; return;
      }
      this.examBusy = true; this.examError = '';
      try {
        if (this.examForm.roomId) {
          await API.confirmTeachingEventRoom(this.examDraft.id, { room_id: Number(this.examForm.roomId),
            confirmation_status: 'CONFIRMED', external_contact: this.examRoomContact.trim(), note: 'Konfirmasi ruang ujian' });
        }
        const detail = await API.getTeachingEvent(this.examDraft.id);
        await API.publishTeachingEvent(this.examDraft.id, this.examOverride.trim() || null, detail.event.version);
        document.getElementById('exam-dialog').close();
        this.showToast('Sesi ujian berhasil diterbitkan.');
        await Promise.all([this.loadEvents(), this.loadSchedule()]);
      } catch (err) { this.examError = err.message || 'Ujian belum berhasil diterbitkan. Draf tetap tersimpan.'; }
      finally { this.examBusy = false; }
    }
  };
}
