function examSessionBehavior() {
  return {
    examStep: 1, // 1 = Form Input, 2 = Pratinjau & Konfirmasi
    examForm: { kind: 'EXAM_UTS', offeringId: '', patternId: '', originDate: '', date: '', start: '', end: '', roomId: '', reason: '' },
    examPatterns: [], examBusy: false, examError: '', examDraft: null, examDraftIsNew: false, examPreview: null,
    examLecturerConfirmed: true, examRoomConfirmed: false, examRoomContact: '', examOverride: '',

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
      if (!this.examForm.patternId) return 'Ujian mandiri (di luar jam kuliah rutin)';
      const p = (this.examPatterns || []).find(x => String(x.id) === String(this.examForm.patternId));
      if (!p) return 'Menggantikan pertemuan kuliah rutin';
      return `Menggantikan kuliah ${this.polaHariName(p.day_of_week)}, ${String(p.start_time || '').slice(0, 5)} - ${String(p.end_time || '').slice(0, 5)}` +
        (this.examForm.originDate ? ` (${this.examForm.originDate})` : '');
    },

    formatExamConflict(c) {
      if (!c) return { title: 'Bentrok Jadwal', detail: '' };
      const code = String(c.code || '').toUpperCase();
      let cleanMsg = String(c.message || '')
        .replace(/^[A-Z_]+:\s*/i, '')
        .replace(/^(offering yang sama|kelas|dosen|ruangan)\s+(memiliki|mengajar|dipakai)\s+sesi\s+lain\s+pada\s+interval\s+tersebut:?\s*/i, '')
        .replace(/^\((.*)\)$/, '$1')
        .trim();

      if (code === 'OFFERING_TIME_OVERLAP') {
        const course = c.course_name || cleanMsg || this.getExamOfferingName();
        return {
          title: 'Bentrok Kuliah Rutin Sendiri',
          detail: `Beririsan dengan jadwal rutin ${course}. Pilih jadwal tersebut pada kolom "Jadwal kuliah yang digantikan".`
        };
      }
      if (code === 'CLASS_TIME_OVERLAP') {
        const course = c.course_name || cleanMsg || 'mata kuliah lain';
        return {
          title: 'Bentrok Jadwal Kelas',
          detail: `Kelas sedang mengikuti sesi ${course} pada jam ini.`
        };
      }
      if (code === 'LECTURER_TIME_OVERLAP') {
        const target = (c.class_name && c.course_name) ? `${c.class_name} (${c.course_name})` : (cleanMsg || 'kelas lain');
        return {
          title: 'Bentrok Jadwal Dosen',
          detail: `Dosen pengampu sedang mengajar di ${target}.`
        };
      }
      if (code === 'ROOM_TIME_OVERLAP') {
        const target = (c.class_name && c.course_name) ? `${c.class_name} — ${c.course_name}` : (cleanMsg || 'sesi kelas lain');
        return {
          title: 'Digunakan Kelas Lain',
          detail: target
        };
      }
      return {
        title: 'Perhatian Jadwal',
        detail: cleanMsg || c.message || code
      };
    },

    kembaliKeForm() {
      this.examStep = 1;
      this.examError = '';
    },

    async tutupDialogUjian() {
      const dlg = document.getElementById('exam-dialog');
      if (dlg && dlg.open) dlg.close();
      if (this.examDraft && this.examDraftIsNew) {
        const draftId = this.examDraft.id;
        const draftVer = this.examDraft.version || 1;
        try {
          await API.deleteTeachingEvent(draftId, draftVer);
          this.examDraft = null;
          this.examDraftIsNew = false;
          this.examPreview = null;
          if (this.loadEvents) await this.loadEvents();
        } catch (e) {
          this.examError = e.message || 'Draf belum terhapus. Coba lagi dari daftar perubahan.';
          if (this.showToast) this.showToast(this.examError);
        }
      }
    },

    pilihMatkulUjian() {
      const patterns = (this.examPatterns || []).filter(p =>
        String(p.course_offering_id || p.offering_id) === String(this.examForm.offeringId)
      );
      if (patterns.length > 0) {
        this.examForm.patternId = String(patterns[0].id);
        this.pilihAsalUjian();
      } else {
        this.examForm.patternId = '';
        this.examForm.originDate = '';
      }
    },

    pilihTanggalUjian() {
      if (!this.examForm.date) return;
      const parts = this.examForm.date.split('-');
      if (parts.length !== 3) return;
      const d = new Date(Number(parts[0]), Number(parts[1]) - 1, Number(parts[2]));
      const jsDay = d.getDay();
      const dow = jsDay === 0 ? 7 : jsDay;

      if (this.examForm.patternId) {
        const p = (this.examPatterns || []).find(x => String(x.id) === String(this.examForm.patternId));
        if (p && (Number(p.day_of_week) === dow || !this.examForm.originDate)) {
          this.examForm.originDate = this.examForm.date;
        }
      } else {
        const match = this.getMatchingPattern();
        if (match) {
          this.examForm.patternId = String(match.id);
          this.examForm.originDate = this.examForm.date;
          if (!this.examForm.start) this.examForm.start = String(match.start_time || '').slice(0, 5);
          if (!this.examForm.end) this.examForm.end = String(match.end_time || '').slice(0, 5);
          if (!this.examForm.roomId && match.room_id) this.examForm.roomId = String(match.room_id);
        }
      }
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
      this.examDraft = null; this.examDraftIsNew = false; this.examPreview = null; this.examError = '';
      this.examLecturerConfirmed = true; this.examRoomConfirmed = false; this.examRoomContact = ''; this.examOverride = '';
      if (this.tutupModalDetailSesi) this.tutupModalDetailSesi();
      if (this.tutupDetailSesi) this.tutupDetailSesi();
      document.getElementById('exam-dialog').showModal();
      this.examBusy = true;
      try {
        this.examPatterns = await API.getPatterns();
        if (this.examForm.patternId) {
          const p = this.examPatterns.find(p => String(p.id) === this.examForm.patternId);
          if (p) {
            this.examForm.offeringId = String(p.course_offering_id || p.offering_id);
            this.examForm.roomId = p.room_id ? String(p.room_id) : '';
          }
        } else if (this.examForm.offeringId) {
          this.pilihMatkulUjian();
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
        this.examDraftIsNew = false;
        this.examPreview = await API.previewTeachingEvent(e.id);
        if (!this.examPreview) throw new Error('Pratinjau gagal dimuat. Coba tinjau lagi.');
        if (detail.confirmations && detail.confirmations.length) {
          const c = detail.confirmations.find(x => String(x.room_id) === String(e.room_id) && x.confirmation_status === 'CONFIRMED');
          if (c) {
            this.examRoomConfirmed = true;
            this.examRoomContact = c.contact || c.external_contact || '';
          }
        }
        this.examStep = 2;
      } catch (err) { this.examError = err.message || 'Draf ujian gagal dimuat.'; }
      finally { this.examBusy = false; }
    },

    pilihAsalUjian() {
      const p = (this.examPatterns || []).find(p => String(p.id) === String(this.examForm.patternId));
      if (!p) { this.examForm.originDate = ''; return; }
      this.examForm.start = String(p.start_time || '').slice(0, 5);
      this.examForm.end = String(p.end_time || '').slice(0, 5);
      this.examForm.roomId = p.room_id ? String(p.room_id) : '';
      const day = (this.weekDays || []).find(d => d.name === this.polaHariName(p.day_of_week));
      if (day && day.dateStr) {
        this.examForm.originDate = day.dateStr;
        this.examForm.date = day.dateStr;
      } else if (this.examForm.date && !this.examForm.originDate) {
        this.examForm.originDate = this.examForm.date;
      }
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

    isRoutineRoom() {
      if (!this.examForm.patternId || !this.examForm.roomId) return false;
      const p = (this.examPatterns || []).find(x => String(x.id) === String(this.examForm.patternId));
      if (!p || !p.room_id) return false;
      if (String(p.room_id) !== String(this.examForm.roomId)) return false;
      const conflicts = (this.examPreview && this.examPreview.conflicts) || [];
      if (conflicts.some(c => c.code === 'ROOM_TIME_OVERLAP')) return false;
      return true;
    },

    async simpanUjian(preview) {
      if (this.examBusy) return;
      const f = this.examForm;
      // Auto-bind ke pola kuliah rutin di hari yang sama bila jam beririsan dan user lupa memilih pola asal
      if (!f.patternId && f.date && f.start && f.end) {
        const match = this.getMatchingPattern();
        if (match) {
          const pStart = String(match.start_time || '').slice(0, 5);
          const pEnd = String(match.end_time || '').slice(0, 5);
          if (!pStart || !pEnd || (f.start < pEnd && f.end > pStart)) {
            f.patternId = String(match.id);
            f.originDate = f.date;
          }
        }
      }
      if (f.patternId && !f.originDate && f.date) {
        f.originDate = f.date;
      }
      if (!f.offeringId || !f.date || !f.start || !f.end || f.start >= f.end || (f.patternId && !f.originDate)) {
        this.examError = 'Lengkapi mata kuliah, tanggal, jam selesai setelah jam mulai, dan tanggal asal bila memakai jadwal rutin.';
        return;
      }
      this.examBusy = true; this.examError = '';
      const wasExistingSavedDraft = Boolean(this.examDraft && !this.examDraftIsNew);
      try {
        const payload = { owner_offering_id: Number(f.offeringId), event_kind: f.kind,
          starts_at: new Date(`${f.date}T${f.start}:00+07:00`).toISOString(),
          ends_at: new Date(`${f.date}T${f.end}:00+07:00`).toISOString(),
          room_id: f.roomId ? Number(f.roomId) : null, reason: f.reason.trim() || null };
        if (f.patternId) { payload.origin_pattern_id = Number(f.patternId); payload.origin_date = f.originDate; }
        if (this.examDraft) payload.version = this.examDraft.version;
        const saved = await API.createTeachingEventDraft(payload, this.examDraft && this.examDraft.id);
        this.examDraft = saved;
        if (preview) {
          this.examDraftIsNew = !wasExistingSavedDraft;
          this.examPreview = await API.previewTeachingEvent(this.examDraft.id);
          if (!this.examPreview) throw new Error('Pratinjau gagal dimuat. Draf sudah tersimpan, coba tinjau lagi.');
          this.examStep = 2;
        } else {
          this.examDraftIsNew = false;
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
      if (conflicts.some(c => c.blocking)) {
        this.examError = 'Bentrok jadwal dosen/kelas memblokir publikasi. Klik "Ubah Data Ujian" untuk menyesuaikan waktu.';
        return;
      }

      const routine = this.isRoutineRoom();
      if (this.examForm.roomId) {
        if (routine) {
          if (!this.examLecturerConfirmed) {
            this.examError = 'Centang persetujuan Dosen Pengampu sebelum menerbitkan jadwal ujian.';
            return;
          }
        } else {
          if (!this.examRoomConfirmed || !this.examRoomContact.trim()) {
            this.examError = 'Catat konfirmasi ruangan dari TU dan nama kontaknya sebelum publikasi.';
            return;
          }
        }
      }

      if (conflicts.some(c => !c.blocking && c.code !== 'ROOM_CONFIRMATION_REQUIRED') && !this.examOverride.trim()) {
        this.examError = 'Isi alasan pengecualian pemakaian ruangan.';
        return;
      }

      this.examBusy = true;
      this.examError = '';
      try {
        if (this.examForm.roomId) {
          const contact = routine ? 'Dosen Pengampu' : this.examRoomContact.trim();
          const note = routine ? 'Ruangan rutin kelas (persetujuan dosen)' : 'Konfirmasi ruang ujian TU';
          await API.confirmTeachingEventRoom(this.examDraft.id, {
            room_id: Number(this.examForm.roomId),
            confirmation_status: 'CONFIRMED',
            external_contact: contact,
            note: note
          });
        }
        const detail = await API.getTeachingEvent(this.examDraft.id);
        this.examDraft = detail.event;
        await API.publishTeachingEvent(this.examDraft.id, this.examOverride.trim() || null, detail.event.version);
        this.examDraftIsNew = false;
        document.getElementById('exam-dialog').close();
        this.showToast('Sesi ujian berhasil diterbitkan.');
        await Promise.all([this.loadEvents(), this.loadSchedule()]);
      } catch (err) {
        this.examError = err.message || 'Ujian belum berhasil diterbitkan. Draf tetap tersimpan.';
      } finally {
        this.examBusy = false;
      }
    }
  };
}
