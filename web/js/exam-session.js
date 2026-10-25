function examSessionBehavior() {
  return {
    examForm: { kind: 'EXAM_UTS', offeringId: '', patternId: '', originDate: '', date: '', start: '', end: '', roomId: '', reason: '' },
    examPatterns: [], examBusy: false, examError: '', examDraft: null, examPreview: null,
    examRoomConfirmed: false, examRoomContact: '', examOverride: '',

    async bukaUjian(session, day) {
      if (session && !session.patternId) return;
      if (this.roleLabel === 'PJ' && session && String(session.offeringId) !== String(this.offeringId)) return;
      const date = (day && day.dateStr) || (session && session.dateStr) ||
        ((this.weekDays || []).find(d => d.name === (session && session.hari)) || {}).dateStr || '';
      this.examForm = { kind: 'EXAM_UTS', offeringId: String((session && session.offeringId) || this.offeringId || ''),
        patternId: String((session && session.patternId) || ''), originDate: session ? date : '', date,
        start: (session && session.timeStart) || '', end: (session && session.timeEnd) || '', roomId: '', reason: '' };
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

    async simpanUjian(preview) {
      if (this.examBusy) return;
      const f = this.examForm;
      if (!f.offeringId || !f.date || !f.start || !f.end || f.start >= f.end || (f.patternId && !f.originDate)) {
        this.examError = 'Lengkapi mata kuliah, tanggal, jam selesai setelah jam mulai, dan tanggal asal bila memakai jadwal rutin.';
        return;
      }
      this.examBusy = true; this.examError = '';
      try {
        if (!this.examDraft) {
          const payload = { owner_offering_id: Number(f.offeringId), event_kind: f.kind,
            starts_at: new Date(`${f.date}T${f.start}:00+07:00`).toISOString(),
            ends_at: new Date(`${f.date}T${f.end}:00+07:00`).toISOString(),
            room_id: f.roomId ? Number(f.roomId) : null, reason: f.reason.trim() || null };
          if (f.patternId) { payload.origin_pattern_id = Number(f.patternId); payload.origin_date = f.originDate; }
          this.examDraft = await API.createTeachingEventDraft(payload);
        }
        if (preview) {
          this.examPreview = await API.previewTeachingEvent(this.examDraft.id);
          if (!this.examPreview) throw new Error('Pratinjau gagal dimuat. Draf sudah tersimpan, coba tinjau lagi.');
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
      if (conflicts.some(c => c.blocking)) { this.examError = 'Konflik memblokir publikasi. Selesaikan benturan jadwal terlebih dahulu.'; return; }
      if (this.examForm.roomId && (!this.examRoomConfirmed || !this.examRoomContact.trim())) {
        this.examError = 'Catat konfirmasi ruangan dari TU dan nama kontaknya sebelum publikasi.'; return;
      }
      if (conflicts.some(c => !c.blocking && c.code !== 'ROOM_CONFIRMATION_REQUIRED') && !this.examOverride.trim()) {
        this.examError = 'Isi alasan pengecualian konflik.'; return;
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
        this.showToast('Sesi ujian diterbitkan.');
        await Promise.all([this.loadEvents(), this.loadSchedule()]);
      } catch (err) { this.examError = err.message || 'Ujian belum berhasil diterbitkan. Draf tetap tersimpan.'; }
      finally { this.examBusy = false; }
    }
  };
}
