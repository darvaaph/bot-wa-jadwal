function readCookie(name) {
  const parts = document.cookie ? document.cookie.split(';') : [];
  for (const part of parts) {
    const trimmed = part.trim();
    if (trimmed.startsWith(name + '=')) {
      return decodeURIComponent(trimmed.slice(name.length + 1));
    }
  }
  return '';
}

function getAuthToken() {
  if (typeof localStorage === 'undefined') return '';
  return localStorage.getItem('access_token') || localStorage.getItem('token') || '';
}

function setAuthToken(token) {
  if (typeof localStorage === 'undefined') return;
  if (token) {
    localStorage.setItem('access_token', token);
  } else {
    localStorage.removeItem('access_token');
    localStorage.removeItem('token');
  }
}

function authHeaders(extra) {
  const headers = Object.assign({ 'Content-Type': 'application/json' }, extra || {});
  const token = getAuthToken();
  if (token) headers['Authorization'] = 'Bearer ' + token;
  const csrf = readCookie('bot_jadwal_csrf');
  if (csrf) headers['X-CSRF-Token'] = csrf;
  return headers;
}

function mutationHeaders(extra) {
  return authHeaders(extra);
}

const BotApi = {
  async getClasses() {
    try {
      const res = await fetch('/api/classes', { credentials: 'same-origin' });
      if (res.ok) {
        const json = await res.json();
        if (json.data) return json.data;
      }
    } catch (e) {}
    try {
      const st = await this.getStatus();
      if (st && st.classes) {
        return {
          default_class: st.default_class,
          total_classes: st.total_classes,
          classes: st.classes
        };
      }
    } catch (e) {}
    return null;
  },

  async getTasks(classId) {
    try {
      const res = await fetch('/api/v1/tasks?class_id=' + encodeURIComponent(classId || ''), { credentials: 'same-origin' });
      if (res.ok) {
        const json = await res.json();
        if (json.data && json.data.length > 0) {
          return json.data.map(item => ({
            id: item.id,
            course_offering_id: item.course_offering_id,
            course_code: item.course_code || 'TUGAS',
            course_name: item.course_name || 'Mata Kuliah',
            matkul: item.course_name || item.course_code || 'Mata Kuliah',
            title: item.title,
            deskripsi: item.instructions || item.title,
            instructions: item.instructions,
            deadline: item.deadline_at,
            deadline_at: item.deadline_at,
            submission_target: item.submission_text || item.submission_url || 'LMS Kampus',
            status: item.publication_status || 'DRAFT',
            review_status: item.review_state || 'NOT_REVIEWED',
            is_done: !!item.completed_at,
            is_completed: !!item.completed_at,
            creator_name: 'PJ Mata Kuliah'
          }));
        }
      }
    } catch (e) {}

    // Fallback ke endpoint legacy /api/tasks?class=...
    try {
      const legRes = await fetch('/api/tasks?class=' + encodeURIComponent(classId || ''), { credentials: 'same-origin' });
      if (legRes.ok) {
        const legJson = await legRes.json();
        return (legJson.data || []).map(item => ({
          id: item.id,
          matkul: item.matkul,
          deskripsi: item.deskripsi,
          deadline: item.deadline,
          is_done: !!item.is_done
        }));
      }
    } catch (e) {}
    return [];
  },

  async createTask(payload) {
    const isLegacy = payload && (payload.matkul || payload.deskripsi || payload.deadline) && !payload.course_offering_id;
    const url = isLegacy ? '/api/tasks' : '/api/v1/tasks';
    const res = await fetch(url, {
      method: 'POST',
      credentials: 'same-origin',
      headers: mutationHeaders(),
      body: JSON.stringify(payload)
    });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Gagal menyimpan tugas di server.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    return await res.json();
  },

  async reviewTask(taskId, body) {
    const res = await fetch('/api/v1/tasks/' + taskId + '/reviews', {
      method: 'POST',
      credentials: 'same-origin',
      headers: mutationHeaders(),
      body: JSON.stringify(body)
    });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Gagal menyimpan hasil pemeriksaan di server.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    return true;
  },

  async completeTask(taskId) {
    const res = await fetch('/api/v1/tasks/' + taskId + '/complete', {
      method: 'PATCH',
      credentials: 'same-origin',
      headers: mutationHeaders()
    });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Gagal menandai selesai di server.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    return true;
  },

  async getTaskDetail(taskId) {
    const res = await fetch('/api/v1/tasks/' + taskId, { credentials: 'same-origin' });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (!res.ok) return null;
    const json = await res.json();
    return json.data || null;
  },

  async getTaskReviews(taskId) {
    const res = await fetch('/api/v1/tasks/' + taskId + '/reviews', { credentials: 'same-origin' });
    if (!res.ok) return null;
    const json = await res.json();
    return json.data || [];
  },

  async publishTask(taskId) {
    const res = await fetch('/api/v1/tasks/' + taskId + '/publish', {
      method: 'POST',
      credentials: 'same-origin',
      headers: mutationHeaders()
    });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (res.status === 409) {
      const err = new Error('Versi data sudah berubah, muat ulang sebelum menyimpan.');
      err.code = 'VERSION_CONFLICT';
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Gagal mempublikasikan tugas.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    const json = await res.json();
    return json.data;
  },

  async updateTask(taskId, payload) {
    const res = await fetch('/api/v1/tasks/' + taskId, {
      method: 'PUT',
      credentials: 'same-origin',
      headers: mutationHeaders(),
      body: JSON.stringify(payload)
    });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (res.status === 409) {
      const err = new Error('Versi data sudah berubah, muat ulang sebelum menyimpan.');
      err.code = 'VERSION_CONFLICT';
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Gagal mengubah tugas.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    const json = await res.json();
    return json.data;
  },

  async archiveTask(taskId, unarchive) {
    const res = await fetch('/api/v1/tasks/' + taskId + (unarchive ? '/unarchive' : '/archive'), {
      method: 'POST',
      credentials: 'same-origin',
      headers: mutationHeaders()
    });
    if (!res.ok) {
      const err = new Error('Gagal mengubah status arsip.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    const json = await res.json();
    return json.data;
  },

  async deleteTaskV1(taskId) {
    const res = await fetch('/api/v1/tasks/' + taskId, {
      method: 'DELETE',
      credentials: 'same-origin',
      headers: mutationHeaders()
    });
    if (!res.ok) {
      const err = new Error('Gagal menghapus tugas.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    return true;
  },

  async restoreTask(taskId, reason) {
    const res = await fetch('/api/v1/tasks/' + taskId + '/restore', {
      method: 'POST',
      credentials: 'same-origin',
      headers: mutationHeaders(),
      body: JSON.stringify({ reason: reason })
    });
    if (!res.ok) {
      const err = new Error('Gagal memulihkan tugas.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    const json = await res.json();
    return json.data;
  },

  async getMaterials(classId, offeringId) {
    let url = '/api/v1/materials?class_id=' + classId;
    if (offeringId) url += '&course_offering_id=' + offeringId;
    const res = await fetch(url, { credentials: 'same-origin' });
    if (!res.ok) return null;
    const json = await res.json();
    return json.data || [];
  },

  async createMaterial(payload) {
    const res = await fetch('/api/v1/materials', {
      method: 'POST',
      credentials: 'same-origin',
      headers: mutationHeaders(),
      body: JSON.stringify(payload)
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error) || 'Gagal menyimpan materi.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    const json = await res.json();
    return json.data;
  },

  async createClass(payload) {
    const res = await fetch('/api/v1/classes', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error) || 'Gagal membuat kelas.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return (await res.json()).data;
  },

  async getSemesters(classId) {
    const res = await fetch('/api/v1/classes/' + classId + '/semesters', { credentials: 'same-origin' });
    if (!res.ok) return null;
    return (await res.json()).data || [];
  },

  async createSemesterDraft(classId, payload) {
    const res = await fetch('/api/v1/classes/' + classId + '/semesters/draft', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error) || 'Gagal membuat semester draf.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return (await res.json()).data;
  },

  async previewSemester(classId, semesterId) {
    const res = await fetch('/api/v1/classes/' + classId + '/semesters/' + semesterId + '/preview', { credentials: 'same-origin' });
    if (!res.ok) return null;
    return (await res.json()).data;
  },

  async activateSemester(classId, semesterId, version) {
    const res = await fetch('/api/v1/classes/' + classId + '/semesters/' + semesterId + '/activate', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ version: version || null })
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error) || 'Gagal mengaktifkan semester.');
      err.code = res.status === 409 ? 'NOT_READY' : 'SAVE_FAILED'; throw err;
    }
    return true;
  },

  async importSemester(classId, payload) {
    const res = await fetch('/api/v1/classes/' + classId + '/semesters/import', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    const json = await res.json().catch(() => null);
    if (res.status === 422) {
      const err = new Error((json && json.error) || 'Impor ditolak.');
      err.code = 'IMPORT_INVALID'; err.errors = json && json.errors ? json.errors : []; throw err;
    }
    if (!res.ok) {
      const err = new Error((json && json.error) || 'Gagal mengimpor semester.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return json.data;
  },

  async verifyPortalCode(slug, code) {
    const res = await fetch('/api/portal/' + encodeURIComponent(slug) + '/verify-code', {
      method: 'POST', credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ code: code })
    });
    if (res.status === 429) {
      const err = new Error('Terlalu banyak percobaan, coba lagi nanti.');
      err.code = 'RATE_LIMITED'; throw err;
    }
    if (!res.ok) {
      const err = new Error('Kode kelas tidak valid.');
      err.code = 'INVALID_CODE'; throw err;
    }
    return (await res.json()).data;
  },

  async getTeachingEvents(classId, filters) {
    let url = '/api/v1/teaching-events?class_id=' + classId;
    if (filters && filters.lifecycle) url += '&lifecycle=' + encodeURIComponent(filters.lifecycle);
    if (filters && filters.kind) url += '&kind=' + encodeURIComponent(filters.kind);
    const res = await fetch(url, { credentials: 'same-origin' });
    if (!res.ok) return null;
    return (await res.json()).data || [];
  },

  async getTeachingEventDetail(eventId) {
    const res = await fetch('/api/v1/teaching-events/' + eventId, { credentials: 'same-origin' });
    if (!res.ok) return null;
    return (await res.json()).data;
  },

  async previewTeachingEvent(eventId) {
    const res = await fetch('/api/v1/teaching-events/' + eventId + '/preview', { credentials: 'same-origin' });
    if (!res.ok) return null;
    return (await res.json()).data;
  },

  async createTeachingEventDraft(payload) {
    const res = await fetch('/api/v1/teaching-events/draft', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error) || 'Gagal membuat draf event.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return (await res.json()).data;
  },

  async publishTeachingEvent(eventId, conflictOverrideReason) {
    const res = await fetch('/api/v1/teaching-events/' + eventId + '/publish', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ conflict_override_reason: conflictOverrideReason || null })
    });
    if (res.status === 409) {
      const err = new Error('Konflik memblokir publikasi, periksa preview.');
      err.code = 'CONFLICT'; throw err;
    }
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error) || 'Gagal mempublikasikan event.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return (await res.json()).data;
  },

  async revokeTeachingEvent(eventId, reason) {
    const res = await fetch('/api/v1/teaching-events/' + eventId + '/revoke', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify({ reason: reason })
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error) || 'Gagal mencabut publikasi.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return (await res.json()).data;
  },

  async getNotifications(classId, status) {
    let url = '/api/v1/notifications?class_id=' + classId;
    if (status) url += '&status=' + encodeURIComponent(status);
    const res = await fetch(url, { credentials: 'same-origin' });
    if (!res.ok) return null;
    return (await res.json()).data || [];
  },

  async retryNotification(messageId) {
    const res = await fetch('/api/v1/notifications/' + messageId + '/retry', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders()
    });
    if (!res.ok) {
      const err = new Error('Gagal menjadwalkan ulang notifikasi.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return true;
  },

  async getRooms(status) {
    const res = await fetch('/api/v1/rooms?status=' + encodeURIComponent(status || 'ACTIVE'), { credentials: 'same-origin' });
    if (!res.ok) return null;
    return (await res.json()).data || [];
  },

  async getRoomAvailability(date, start, end) {
    const res = await fetch('/api/v1/rooms/availability?date=' + encodeURIComponent(date) + '&start=' + encodeURIComponent(start) + '&end=' + encodeURIComponent(end), { credentials: 'same-origin' });
    if (!res.ok) return null;
    return (await res.json()).data;
  },

  async getAudit(params) {
    const qs = new URLSearchParams(params || {}).toString();
    const res = await fetch('/api/v1/audit' + (qs ? '?' + qs : ''), { credentials: 'same-origin' });
    if (!res.ok) return null;
    return (await res.json()).data || [];
  },

  async getClassSettings(classId) {
    const res = await fetch('/api/v1/classes/' + classId + '/settings', { credentials: 'same-origin' });
    if (!res.ok) return null;
    return (await res.json()).data;
  },

  async updateClassSettings(classId, payload) {
    const res = await fetch('/api/v1/classes/' + classId + '/settings', {
      method: 'PATCH', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    if (res.status === 409) {
      const err = new Error('Pengaturan berubah di tempat lain, muat ulang sebelum menyimpan.');
      err.code = 'VERSION_CONFLICT'; throw err;
    }
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error) || 'Gagal menyimpan pengaturan.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return true;
  },

  async issueRecovery(identityKey, reason) {
    const res = await fetch('/api/v1/admin/recovery/issue', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ identity_key: identityKey, reason: reason })
    });
    if (!res.ok) {
      const err = new Error('Gagal menerbitkan token pemulihan.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return (await res.json()).data;
  },

  async getAdminStatus() {
    const res = await fetch('/api/v1/admin/status', {
      headers: authHeaders(),
      credentials: 'same-origin'
    });
    if (!res.ok) return null;
    const json = await res.json();
    return json.data || null;
  },

  async getSystemStatus() {
    return this.getAdminStatus();
  },

  async getV1Classes() {
    const res = await fetch('/api/v1/classes', {
      headers: authHeaders(),
      credentials: 'same-origin'
    });
    if (!res.ok) return null;
    const json = await res.json();
    return json.data || null;
  },

  async updateClassStatus(slug, status) {
    const res = await fetch('/api/v1/classes/' + encodeURIComponent(slug), {
      method: 'PATCH',
      headers: authHeaders(),
      credentials: 'same-origin',
      body: JSON.stringify({ status: status })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || 'Gagal memperbarui status kelas.');
      throw err;
    }
    return json.data;
  },

  async createInvitation(payload) {
    const res = await fetch('/api/v1/invitations', {
      method: 'POST',
      headers: authHeaders(),
      credentials: 'same-origin',
      body: JSON.stringify(payload)
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const msg = (json && json.error && json.error.message) || (json && json.message) || 'Gagal membuat undangan.';
      const err = new Error(msg);
      err.code = (json && json.error && json.error.code) || 'INVITATION_FAILED';
      throw err;
    }
    return json.data;
  },

  async rotatePortalCode(slug, code) {
    const body = code ? { code: code } : {};
    const res = await fetch('/api/v1/classes/' + encodeURIComponent(slug) + '/portal-code/rotate', {
      method: 'POST',
      headers: authHeaders(),
      credentials: 'same-origin',
      body: JSON.stringify(body)
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || 'Gagal merotasi kode portal.');
      throw err;
    }
    return json.data;
  },

  async getMe() {
    const res = await fetch('/api/v1/auth/me', {
      headers: authHeaders(),
      credentials: 'same-origin'
    });
    if (!res.ok) return null;
    const json = await res.json();
    return json.data || null;
  },

  async createBackup(payload) {
    const res = await fetch('/api/v1/backups', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload || {})
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error && body.error.message) || (body && body.error) || 'Gagal membuat backup.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return (await res.json()).data;
  },

  async restoreBackup(backupId, reason) {
    const res = await fetch('/api/v1/restores', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify({ backup_id: backupId, reason: reason })
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error && body.error.message) || (body && body.error) || 'Gagal memverifikasi restore.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return true;
  },

  async getStatus() {
    const res = await fetch('/api/status', { credentials: 'same-origin' });
    if (!res.ok) return null;
    return await res.json();
  },

  async getPortalTasks(slug) {
    const res = await fetch('/api/portal/' + encodeURIComponent(slug) + '/tasks?group=all', { credentials: 'same-origin' });
    if (!res.ok) return null;
    const json = await res.json();
    if (!json.data || !json.data.tugas) return null;
    return json.data.tugas.map(item => {
      let deadlineAt = '';
      const parts = String(item.tenggat || '').split(' ');
      if (parts.length === 2) {
        deadlineAt = parts[0] + 'T' + parts[1].replace('.', ':') + ':00+07:00';
      }
      return {
        id: item.id,
        course_offering_id: null,
        course_code: item.kode_mata_kuliah || 'TUGAS',
        course_name: item.mata_kuliah || 'Mata Kuliah',
        title: item.judul,
        instructions: item.instruksi,
        deadline_at: deadlineAt,
        submission_target: item.tempat_pengumpulan || 'LMS Kampus',
        status: 'PUBLISHED',
        review_status: 'APPROVED',
        is_completed: false,
        creator_name: 'PJ Mata Kuliah'
      };
    });
  },

  async getSchedule(slug, dateStr) {
    try {
      const res = await fetch('/api/schedule?class=' + encodeURIComponent(slug || '') + '&day=' + encodeURIComponent(dateStr || 'all'), { credentials: 'same-origin' });
      if (res.ok) {
        const json = await res.json();
        if (json.data) return json.data;
      }
    } catch (e) {}

    // Fallback ke endpoint portal v1
    try {
      const res = await fetch('/api/v1/portal/' + encodeURIComponent(slug) + '/schedule?date=' + encodeURIComponent(dateStr || ''), { credentials: 'same-origin' });
      if (res.ok) {
        const json = await res.json();
        if (json.data && json.data.jadwal) return json.data.jadwal;
      }
    } catch (e) {}
    return [];
  },

  async deleteTask(taskId) {
    try {
      const res = await fetch('/api/tasks/' + taskId, {
        method: 'DELETE',
        credentials: 'same-origin',
        headers: mutationHeaders()
      });
      if (res.ok) return true;
    } catch (e) {}
    return this.deleteTaskV1(taskId);
  },

  async getChanges(slug, limit) {
    const res = await fetch('/api/portal/' + encodeURIComponent(slug) + '/changes?limit=' + (limit || 20), { credentials: 'same-origin' });
    if (!res.ok) return null;
    const json = await res.json();
    if (!json.data || !json.data.perubahan) return null;
    return json.data.perubahan;
  },

  async getSession() {
    const res = await fetch('/api/v1/auth/session', { credentials: 'same-origin' });
    if (res.status === 401) return { authenticated: false };
    if (res.status === 503) return { unavailable: true };
    if (!res.ok) return null;
    const json = await res.json();
    if (!json.data) return null;
    return {
      authenticated: true,
      user: json.data.user,
      activeRoleAssignmentId: json.data.active_role_assignment_id,
      assignments: json.data.assignments || []
    };
  },

  async login(identityKey, password, roleAssignmentId) {
    const body = { identity_key: identityKey, password: password };
    if (roleAssignmentId) body.role_assignment_id = roleAssignmentId;
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      credentials: 'same-origin',
      headers: mutationHeaders(),
      body: JSON.stringify(body)
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Identitas atau kata sandi tidak valid.');
      err.code = res.status === 401 ? 'INVALID_CREDENTIALS' : 'LOGIN_FAILED';
      err.payload = json && json.data ? json.data : null;
      throw err;
    }
    if (json && json.data && json.data.token) {
      setAuthToken(json.data.token);
    }
    return json.data;
  },

  async switchContext(roleAssignmentId) {
    const res = await fetch('/api/v1/auth/switch-context', {
      method: 'POST',
      credentials: 'same-origin',
      headers: mutationHeaders(),
      body: JSON.stringify({ role_assignment_id: roleAssignmentId })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error('Konteks akses tidak tersedia.');
      err.code = 'SWITCH_FAILED';
      throw err;
    }
    if (json && json.data && json.data.token) {
      setAuthToken(json.data.token);
    }
    return json.data;
  },

  async logout() {
    try {
      await fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'same-origin',
        headers: mutationHeaders()
      });
    } catch (e) {
      // Session cleanup is best-effort; local state is cleared regardless.
    }
    setAuthToken(null);
  },

  getAuthToken: getAuthToken,
  setAuthToken: setAuthToken
};

// Ekspor global untuk komponen Alpine.js (app-km.js, app.js, app-sa.js)
if (typeof window !== 'undefined') {
  window.BotApi = BotApi;
  window.API = BotApi;
}
if (typeof module !== 'undefined' && module.exports) {
  module.exports = BotApi;
}
