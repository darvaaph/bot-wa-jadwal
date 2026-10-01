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
        if (json.classes) return json;
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

  mapTaskItem(item) {
    return {
      id: item.id,
      course_offering_id: item.offering_id,
      offering_id: item.offering_id,
      course_code: item.offering || 'TUGAS',
      course_name: item.offering || 'Mata Kuliah',
      matkul: item.offering || 'Mata Kuliah',
      title: item.title,
      deskripsi: item.instructions || item.title,
      instructions: item.instructions,
      deadline: item.deadline_at,
      deadline_at: item.deadline_at,
      task_type: item.task_type || '',
      submission_text: item.submission_text || '',
      submission_url: item.submission_url || '',
      submission_target: item.submission_text || item.submission_url || 'LMS Kampus',
      status: item.publication_status || 'DRAFT',
      publication_status: item.publication_status || 'DRAFT',
      review_status: item.review_state || 'NOT_REVIEWED',
      review_state: item.review_state || 'NOT_REVIEWED',
      version: item.version || 0,
      completed_at: item.completed_at || null,
      archived_at: item.archived_at || null,
      is_done: !!item.completed_at,
      is_completed: !!item.completed_at,
      is_archived: !!item.archived_at,
      creator_name: 'PJ Mata Kuliah'
    };
  },

  async getTasks(offeringId, tab) {
    const params = new URLSearchParams();
    if (offeringId && /^\d+$/.test(String(offeringId))) params.set('offering_id', String(offeringId));
    if (tab) params.set('tab', tab);
    const qs = params.toString();
    try {
      const res = await fetch('/api/v1/tasks' + (qs ? '?' + qs : ''), {
        headers: authHeaders(),
        credentials: 'same-origin'
      });
      if (res.ok) {
        const json = await res.json();
        if (json.data && json.data.length > 0) {
          return json.data.map(item => this.mapTaskItem(item));
        }
      }
    } catch (e) {}

    // Fallback ke endpoint legacy /api/tasks?class=... (untuk portal publik tanpa auth)
    try {
      const classSlug = offeringId && !/^\d+$/.test(String(offeringId)) ? String(offeringId) : '';
      const legRes = await fetch('/api/tasks?class=' + encodeURIComponent(classSlug), { credentials: 'same-origin' });
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
    // Backend hanya punya POST /api/v1/tasks (legacy POST /api/tasks = 410 Gone).
    const res = await fetch('/api/v1/tasks', {
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
    if (res.status === 409) {
      const json = await res.json().catch(() => null);
      const err = new Error((json && json.error && json.error.message) || 'Versi tugas telah berubah. Muat ulang untuk melihat revisi terbaru.');
      err.code = 'VERSION_CONFLICT';
      err.payload = json && json.data ? json.data : json;
      throw err;
    }
    if (!res.ok) {
      const json = await res.json().catch(() => null);
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal menyimpan hasil pemeriksaan di server.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    const json = await res.json().catch(() => null);
    return (json && json.data) || true;
  },

  // Mengambil seluruh tab server paralel lalu menggabung (dedupe per id).
  // Backend tidak punya tab "semua", jadi gabungkan: aktif, draf, review, selesai, terlewat, arsip.
  async getAllTasks(offeringId) {
    const tabs = ['aktif', 'draf', 'review', 'selesai', 'terlewat', 'arsip'];
    const results = await Promise.all(tabs.map(t => this.getTasks(offeringId, t).catch(() => [])));
    const seen = new Map();
    results.flat().forEach(t => {
      if (t && t.id != null && !seen.has(String(t.id))) seen.set(String(t.id), t);
    });
    return Array.from(seen.values());
  },

  async completeTask(taskId, version) {
    let v = version;
    if (!v) {
      const d = await this.getTaskDetail(taskId).catch(() => null);
      v = d && (d.version || (d.task && d.task.version));
    }
    const res = await fetch('/api/v1/tasks/' + taskId + '/complete', {
      method: 'POST',
      credentials: 'same-origin',
      headers: mutationHeaders(),
      body: JSON.stringify({ version: v || 0 })
    });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (res.status === 409) {
      throw this.versionConflictErr('Versi tugas telah berubah. Muat ulang sebelum menandai selesai.');
    }
    if (!res.ok) {
      const err = new Error('Gagal menandai selesai di server.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    return true;
  },

  async getTaskDetail(taskId) {
    const res = await fetch('/api/v1/tasks/' + taskId, {
      headers: authHeaders(),
      credentials: 'same-origin'
    });
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
    // Tidak ada GET /api/v1/tasks/{id}/reviews — review ikut di GetTaskDetail.
    const d = await this.getTaskDetail(taskId).catch(() => null);
    if (!d) return [];
    if (Array.isArray(d)) return d;
    return d.reviews || [];
  },

  async publishTask(taskId, version) {
    // Tidak ada POST /api/v1/tasks/{id}/publish — publish via PATCH save_as=published.
    let v = version;
    if (!v) {
      const d = await this.getTaskDetail(taskId).catch(() => null);
      v = d && (d.version || (d.task && d.task.version)) || 0;
    }
    return this.updateTask(taskId, { version: v, save_as: 'published' });
  },

  async updateTask(taskId, payload) {
    const res = await fetch('/api/v1/tasks/' + taskId, {
      method: 'PATCH',
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

  versionConflictErr(fallbackMsg) {
    const err = new Error(fallbackMsg);
    err.code = 'VERSION_CONFLICT';
    return err;
  },

  async archiveTask(taskId, unarchive, version) {
    // Backend: POST .../archive dan POST .../restore (tidak ada /unarchive).
    const endpoint = unarchive ? '/restore' : '/archive';
    let v = version;
    if (!v) {
      const d = await this.getTaskDetail(taskId).catch(() => null);
      v = d && (d.version || (d.task && d.task.version)) || 0;
    }
    const res = await fetch('/api/v1/tasks/' + taskId + endpoint, {
      method: 'POST',
      credentials: 'same-origin',
      headers: mutationHeaders(),
      body: JSON.stringify({ version: v || 0 })
    });
    if (res.status === 409) {
      throw this.versionConflictErr('Versi tugas telah berubah. Muat ulang sebelum mengubah status.');
    }
    if (!res.ok) {
      const err = new Error('Gagal mengubah status arsip.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    const json = await res.json();
    return json.data;
  },

  async deleteTaskV1(taskId, version) {
    // Backend tidak punya DELETE /api/v1/tasks/{id} — hapus = arsip.
    return this.archiveTask(taskId, false, version);
  },

  async restoreTask(taskId, version) {
    let v = version;
    if (!v || typeof v !== 'number') {
      const d = await this.getTaskDetail(taskId).catch(() => null);
      v = d && (d.version || (d.task && d.task.version)) || 0;
    }
    const res = await fetch('/api/v1/tasks/' + taskId + '/restore', {
      method: 'POST',
      credentials: 'same-origin',
      headers: mutationHeaders(),
      body: JSON.stringify({ version: v || 0 })
    });
    if (res.status === 409) {
      throw this.versionConflictErr('Versi tugas telah berubah. Muat ulang sebelum memulihkan.');
    }
    if (!res.ok) {
      const err = new Error('Gagal memulihkan tugas.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    const json = await res.json();
    return json.data;
  },

  async getMaterials(classSlug, offeringId) {
    let url = '/api/v1/materials?class_slug=' + encodeURIComponent(classSlug || '');
    if (offeringId) url += '&offering_id=' + encodeURIComponent(offeringId);
    const res = await fetch(url, { headers: authHeaders(), credentials: 'same-origin' });
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

  async updateMaterial(id, payload) {
    const res = await fetch('/api/v1/materials/' + encodeURIComponent(id), {
      method: 'PATCH',
      credentials: 'same-origin',
      headers: mutationHeaders(),
      body: JSON.stringify(payload)
    });
    const body = await res.json().catch(() => null);
    if (res.status === 409) {
      const err = new Error((body && body.error && body.error.message) || (body && body.error) || 'Versi materi berubah di server.');
      err.code = 'VERSION_CONFLICT'; err.payload = body;
      throw err;
    }
    if (!res.ok) {
      const err = new Error((body && body.error && body.error.message) || (body && body.error) || 'Gagal mengubah materi.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    return body.data;
  },

  async archiveMaterial(id, version) {
    const res = await fetch('/api/v1/materials/' + encodeURIComponent(id) + '?version=' + encodeURIComponent(version || 0), {
      method: 'DELETE',
      credentials: 'same-origin',
      headers: mutationHeaders()
    });
    const body = await res.json().catch(() => null);
    if (res.status === 409) {
      const err = new Error((body && body.error && body.error.message) || (body && body.error) || 'Versi materi berubah di server.');
      err.code = 'VERSION_CONFLICT'; err.payload = body;
      throw err;
    }
    if (!res.ok) {
      const err = new Error((body && body.error && body.error.message) || (body && body.error) || 'Gagal mengarsipkan materi.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    return body.data;
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

  async getSemesters(classSlug) {
    const r = await this.getSemestersResult(classSlug);
    return r.ok ? r.data : null;
  },

  // Versi mentah agar pemanggil bisa membedakan 404 (kelas belum terdaftar)
  // dari gagal jaringan/server.
  async getSemestersResult(classSlug) {
    try {
      const res = await fetch('/api/v1/classes/' + encodeURIComponent(classSlug) + '/semesters', {
        headers: authHeaders(),
        credentials: 'same-origin'
      });
      if (!res.ok) return { ok: false, status: res.status, data: [] };
      const json = await res.json().catch(() => null);
      return { ok: true, status: res.status, data: (json && json.data) || [] };
    } catch (e) {
      return { ok: false, status: 0, data: [] };
    }
  },

  async createSemesterDraft(classSlug, payload) {
    const res = await fetch('/api/v1/classes/' + encodeURIComponent(classSlug) + '/semesters', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error) || 'Gagal membuat semester draf.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return (await res.json()).data;
  },

  async previewSemester(classSlug, semesterId) {
    const res = await fetch('/api/v1/classes/' + encodeURIComponent(classSlug) + '/semesters/' + encodeURIComponent(semesterId) + '/preview', {
      headers: authHeaders(), credentials: 'same-origin'
    });
    if (!res.ok) return null;
    const json = await res.json().catch(() => null);
    return (json && json.data) || null;
  },

  async createSemesterOffering(semesterId, payload) {
    const res = await fetch('/api/v1/semesters/' + encodeURIComponent(semesterId) + '/offerings', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal menambah mata kuliah.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return json.data;
  },

  async applySemesterImport(semesterId, batchId) {
    const res = await fetch('/api/v1/semesters/' + encodeURIComponent(semesterId) + '/import-apply', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify({ batch_id: batchId })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error) || 'Gagal menerapkan impor.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return json.data;
  },

  async activateSemester(classSlug, semesterId) {
    const res = await fetch('/api/v1/classes/' + encodeURIComponent(classSlug) + '/semesters/' + encodeURIComponent(semesterId) + '/activate', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ confirm: true })
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error) || 'Gagal mengaktifkan semester.');
      err.code = res.status === 409 ? 'NOT_READY' : 'SAVE_FAILED'; throw err;
    }
    return true;
  },

  async getSemesterOfferings(semesterId) {
    // null = gagal dimuat (bedakan dari [] = semester tanpa mata kuliah).
    try {
      const res = await fetch('/api/v1/semesters/' + encodeURIComponent(semesterId) + '/offerings', {
        headers: authHeaders(),
        credentials: 'same-origin'
      });
      if (!res.ok) return null;
      const json = await res.json().catch(() => null);
      return (json && json.data) || [];
    } catch (e) {
      return null;
    }
  },

  async importSemester(semesterId, payload) {
    const res = await fetch('/api/v1/semesters/' + encodeURIComponent(semesterId) + '/import-validate', {
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
    const res = await fetch('/api/v1/portal/' + encodeURIComponent(slug) + '/session', {
      method: 'POST', credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ code: code })
    });
    if (res.status === 429) {
      const err = new Error('Terlalu banyak percobaan, coba lagi nanti (15 menit).');
      err.code = 'RATE_LIMITED'; throw err;
    }
    if (!res.ok) {
      const err = new Error('Kode kelas tidak valid.');
      err.code = 'INVALID_CODE'; throw err;
    }
    return (await res.json()).data;
  },

  async getPatterns(params) {
    const qs = new URLSearchParams(params || {}).toString();
    const res = await fetch('/api/v1/schedule/patterns' + (qs ? '?' + qs : ''), {
      headers: authHeaders(),
      credentials: 'same-origin'
    });
    if (!res.ok) return [];
    const json = await res.json().catch(() => null);
    return (json && json.data) || [];
  },

  async createPattern(payload) {
    const res = await fetch('/api/v1/schedule/patterns', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal menyimpan pola jadwal.');
      err.code = 'SAVE_FAILED'; err.payload = json; throw err;
    }
    return json.data;
  },

  async patchPattern(patternId, payload) {
    const res = await fetch('/api/v1/schedule/patterns/' + encodeURIComponent(patternId), {
      method: 'PATCH', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    const json = await res.json().catch(() => null);
    if (res.status === 409) {
      const err = new Error((json && json.error && json.error.message) || 'Ada perubahan yang lebih baru. Muat ulang sebelum menyimpan.');
      err.code = 'VERSION_CONFLICT'; err.payload = json; throw err;
    }
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal mengubah pola jadwal.');
      err.code = 'SAVE_FAILED'; err.payload = json; throw err;
    }
    return json.data;
  },

  async deletePattern(patternId, version) {
    const url = '/api/v1/schedule/patterns/' + encodeURIComponent(patternId) + '?version=' + encodeURIComponent(version);
    const res = await fetch(url, {
      method: 'DELETE', credentials: 'same-origin', headers: mutationHeaders()
    });
    const json = await res.json().catch(() => null);
    if (res.status === 409) {
      const err = new Error((json && json.error && json.error.message) || 'Versi pola jadwal tidak cocok. Muat ulang sebelum menghapus.');
      err.code = 'VERSION_CONFLICT'; err.payload = json; throw err;
    }
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal menghapus pola jadwal.');
      err.code = 'SAVE_FAILED'; err.payload = json; throw err;
    }
    return json.data;
  },

  async deleteTeachingEvent(eventId, version) {
    const url = '/api/v1/teaching-events/' + encodeURIComponent(eventId) + '?version=' + encodeURIComponent(version);
    const res = await fetch(url, {
      method: 'DELETE', credentials: 'same-origin', headers: mutationHeaders()
    });
    const json = await res.json().catch(() => null);
    if (res.status === 409) {
      const err = new Error((json && json.error && json.error.message) || 'Versi kejadian jadwal tidak cocok. Muat ulang sebelum menghapus.');
      err.code = 'VERSION_CONFLICT'; err.payload = json; throw err;
    }
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal menghapus draf perubahan jadwal.');
      err.code = 'SAVE_FAILED'; err.payload = json; throw err;
    }
    return json.data;
  },

  async getTeachingEvents(filters) {
    // Backend: GET /api/v1/teaching-events?status= (DRAFT/PUBLISHED/REVOKED).
    let url = '/api/v1/teaching-events';
    if (filters && (filters.status || filters.lifecycle)) {
      url += '?status=' + encodeURIComponent(filters.status || filters.lifecycle);
    }
    const res = await fetch(url, { credentials: 'same-origin', headers: authHeaders() });
    if (!res.ok) return null;
    return (await res.json()).data || [];
  },

  async getTeachingEventDetail(eventId) {
    // Backend tidak punya GET detail — cari dari list.
    const list = await this.getTeachingEvents().catch(() => null);
    if (Array.isArray(list)) return list.find(e => String(e.id) === String(eventId)) || null;
    return null;
  },

  async previewTeachingEvent(eventId) {
    const res = await fetch('/api/v1/teaching-events/' + eventId + '/preview', {
      method: 'POST',
      credentials: 'same-origin',
      headers: authHeaders()
    });
    if (!res.ok) return null;
    return (await res.json()).data;
  },

  async createTeachingEventDraft(payload) {
    const res = await fetch('/api/v1/teaching-events', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error && body.error.message) || (body && body.error) || 'Gagal membuat draf event.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return (await res.json()).data;
  },

  async publishTeachingEvent(eventId, conflictOverrideReason, version) {
    const headers = Object.assign(mutationHeaders(), {
      'Idempotency-Key': 'pub-event-' + eventId + '-' + Date.now()
    });
    const res = await fetch('/api/v1/teaching-events/' + eventId + '/publish', {
      method: 'POST', credentials: 'same-origin', headers: headers,
      body: JSON.stringify({
        version: version || 1,
        conflict_override_reason: conflictOverrideReason || null
      })
    });
    if (res.status === 409) {
      const err = new Error('Konflik memblokir publikasi, periksa preview.');
      err.code = 'CONFLICT'; throw err;
    }
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error && body.error.message) || (body && body.error) || 'Gagal mempublikasikan event.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return (await res.json()).data;
  },

  async participateTeachingEvent(eventId, action) {
    const res = await fetch('/api/v1/teaching-events/' + eventId + '/participation', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ action: action })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal menyimpan keputusan partisipasi.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return json.data;
  },

  async confirmTeachingEventRoom(eventId, payload) {
    const res = await fetch('/api/v1/teaching-events/' + eventId + '/room-confirmations', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal mencatat konfirmasi ruangan.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return json.data;
  },

  async revokeTeachingEvent(eventId, reason, version) {
    let v = version;
    if (!v) {
      const list = await this.getTeachingEvents().catch(() => []);
      const found = (list || []).find(e => String(e.id) === String(eventId));
      v = (found && found.version) || 0;
    }
    const res = await fetch('/api/v1/teaching-events/' + eventId + '/revoke', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ reason: reason, version: v || 0 })
    });
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      const err = new Error((body && body.error) || 'Gagal mencabut publikasi.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return (await res.json()).data;
  },

  async getNotifications(status, limit, extra) {
    // Backend: GET /api/v1/notifications?status=&class_id=&event_type=&since=&until=&limit=&offset=.
    let url = '/api/v1/notifications';
    const qs = new URLSearchParams();
    if (status) qs.set('status', status);
    if (limit) qs.set('limit', String(limit));
    if (extra) {
      if (extra.class_id) qs.set('class_id', String(extra.class_id));
      if (extra.event_type) qs.set('event_type', extra.event_type);
      if (extra.since) qs.set('since', extra.since);
      if (extra.until) qs.set('until', extra.until);
      if (extra.offset) qs.set('offset', String(extra.offset));
    }
    if ([...qs].length) url += '?' + qs.toString();
    const res = await fetch(url, { headers: authHeaders(), credentials: 'same-origin' });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Antrean notifikasi gagal dimuat.');
      err.code = 'LOAD_FAILED';
      err.status = res.status;
      throw err;
    }
    const json = await res.json().catch(() => null);
    return (json && json.data) || [];
  },

  async getChannels(classSlug, status) {
    let url = '/api/v1/whatsapp-channels';
    const qs = new URLSearchParams();
    if (classSlug) qs.set('class_slug', classSlug);
    if (status) qs.set('status', status);
    if ([...qs].length) url += '?' + qs.toString();
    const res = await fetch(url, { headers: authHeaders(), credentials: 'same-origin' });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Daftar kanal gagal dimuat.');
      err.code = 'LOAD_FAILED';
      err.status = res.status;
      throw err;
    }
    const json = await res.json().catch(() => null);
    return (json && json.data) || [];
  },

  async linkChannel(jid, classSlug, displayName) {
    const res = await fetch('/api/v1/whatsapp-channels', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ jid: jid, class_slug: classSlug, display_name: displayName || '' })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal menautkan kanal.');
      err.code = res.status === 409 ? 'CONFLICT' : 'SAVE_FAILED';
      err.status = res.status;
      throw err;
    }
    return json.data;
  },

  async revokeChannel(id, reason) {
    const res = await fetch('/api/v1/whatsapp-channels/' + encodeURIComponent(id) + '/revoke', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ reason: reason || '' })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal melepas kanal.');
      err.code = res.status === 409 ? 'CONFLICT' : 'SAVE_FAILED';
      err.status = res.status;
      throw err;
    }
    return json.data;
  },

  async getNotificationAttempts(id) {    const res = await fetch('/api/v1/notifications/' + encodeURIComponent(id) + '/attempts', {
      headers: authHeaders(), credentials: 'same-origin'
    });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Riwayat percobaan gagal dimuat.');
      err.code = 'LOAD_FAILED';
      err.status = res.status;
      throw err;
    }
    const json = await res.json().catch(() => null);
    return (json && json.data) || [];
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

  async getRooms() {
    // Tidak ada GET /api/v1/rooms — gunakan kandidat dengan rentang hari ini.
    const now = new Date();
    const s = now.toISOString();
    const e = new Date(now.getTime() + 3600000).toISOString();
    return this.getRoomAvailability(s, e);
  },

  async getRoomAvailability(startsAt, endsAt) {
    // Backend: GET /api/v1/rooms/candidates?starts_at=&ends_at= (RFC3339).
    const res = await fetch('/api/v1/rooms/candidates?starts_at=' + encodeURIComponent(startsAt) + '&ends_at=' + encodeURIComponent(endsAt), {
      headers: authHeaders(),
      credentials: 'same-origin'
    });
    if (!res.ok) return null;
    return (await res.json()).data;
  },

  async getAudit(params) {
    const qs = new URLSearchParams(params || {}).toString();
    const res = await fetch('/api/v1/audit' + (qs ? '?' + qs : ''), {
      headers: authHeaders(),
      credentials: 'same-origin'
    });
    if (!res.ok) return null;
    return (await res.json()).data || [];
  },

  async getClassSettings(slug) {
    const res = await fetch('/api/v1/classes/' + encodeURIComponent(slug) + '/settings', {
      headers: authHeaders(),
      credentials: 'same-origin'
    });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (!res.ok) return null;
    const json = await res.json().catch(() => null);
    return (json && json.data) || null;
  },

  async setPortalMode(slug, mode, reason) {
    const res = await fetch('/api/v1/classes/' + encodeURIComponent(slug) + '/portal-mode', {
      method: 'PATCH', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ mode: mode, reason: reason || '' })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal mengubah mode portal.');
      err.code = res.status === 422 ? 'VALIDATION' : 'SAVE_FAILED';
      err.status = res.status;
      throw err;
    }
    return json.data;
  },

  async updateClassSettings(slug, payload) {
    const res = await fetch('/api/v1/classes/' + encodeURIComponent(slug) + '/settings', {
      method: 'PATCH', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify(payload || {})
    });
    const json = await res.json().catch(() => null);
    if (res.status === 409) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Versi pengaturan berubah di server.');
      err.code = 'VERSION_CONFLICT'; err.payload = json; throw err;
    }
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal menyimpan pengaturan kelas.');
      err.code = 'SAVE_FAILED'; err.status = res.status;
      throw err;
    }
    return json.data;
  },

  async getProposals(status, kind) {
    let url = '/api/v1/master/proposals';
    const qs = new URLSearchParams();
    if (status) qs.set('status', status);
    if (kind) qs.set('kind', kind);
    if ([...qs].length) url += '?' + qs.toString();
    const res = await fetch(url, { headers: authHeaders(), credentials: 'same-origin' });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Daftar usulan gagal dimuat.');
      err.code = 'LOAD_FAILED';
      err.status = res.status;
      throw err;
    }
    const json = await res.json().catch(() => null);
    return (json && json.data) || [];
  },

  async decideProposal(id, keputusan, reviewNote) {
    const res = await fetch('/api/v1/master/proposals/' + encodeURIComponent(id) + '/' + keputusan, {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ review_note: reviewNote || '' })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal memutuskan usulan.');
      err.code = res.status === 409 ? 'CONFLICT' : 'SAVE_FAILED';
      err.status = res.status;
      throw err;
    }
    return json.data;
  },

  async createProposal(payload) {
    const res = await fetch('/api/v1/master/proposals', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify(payload || {})
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal mengirim usulan.');
      err.code = 'SAVE_FAILED';
      err.status = res.status;
      throw err;
    }
    return json.data;
  },

  async issueRecovery() {
    const err = new Error('Pemulihan akun via API belum tersedia.');
    err.code = 'NOT_IMPLEMENTED';
    throw err;
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

  async getBackups(classSlug, status) {
    let url = '/api/v1/backups';
    const qs = new URLSearchParams();
    if (classSlug) qs.set('class_slug', classSlug);
    if (status) qs.set('status', status);
    if ([...qs].length) url += '?' + qs.toString();
    const res = await fetch(url, { headers: authHeaders(), credentials: 'same-origin' });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Daftar cadangan gagal dimuat.');
      err.code = 'LOAD_FAILED';
      err.status = res.status;
      throw err;
    }
    const json = await res.json().catch(() => null);
    return (json && json.data) || [];
  },

  async restoreBackup(backupId, reason) {
    const res = await fetch('/api/v1/restores', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify({ backup_id: backupId, reason: reason })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal memverifikasi restore.');
      err.code = 'SAVE_FAILED'; throw err;
    }
    return (json && json.data) || true;
  },

  async getStatus() {
    const res = await fetch('/api/status', { credentials: 'same-origin' });
    if (!res.ok) return null;
    return await res.json();
  },

  portalHeaders() {
    const headers = {};
    try {
      const savedToken = localStorage.getItem('portal_token');
      const savedClass = localStorage.getItem('portal_class');
      if (savedToken && savedClass) headers['X-Portal-Token'] = savedToken;
    } catch (e) {}
    return headers;
  },

  // Jadwal efektif portal untuk satu tanggal (pola + perubahan terbit).
  // Mengembalikan {date, items} atau null bila kelas tak ditemukan / akses ditolak.
  async getPortalSchedule(slug, dateStr) {
    try {
      const res = await fetch('/api/v1/portal/' + encodeURIComponent(slug) + '/schedule' + (dateStr ? '?date=' + encodeURIComponent(dateStr) : ''), {
        credentials: 'same-origin',
        headers: this.portalHeaders()
      });
      if (!res.ok) return null;
      const json = await res.json();
      return json.data || null;
    } catch (e) {
      return null;
    }
  },

  async getPortalSemesters(slug) {
    try {
      const res = await fetch('/api/v1/portal/' + encodeURIComponent(slug) + '/semesters', {
        credentials: 'same-origin',
        headers: this.portalHeaders()
      });
      if (!res.ok) return [];
      const json = await res.json();
      return json.data || [];
    } catch (e) {
      return [];
    }
  },

  async getPortalTaskDetail(slug, taskId) {
    const headers = this.portalHeaders();
    const res = await fetch('/api/v1/portal/' + encodeURIComponent(slug) + '/tasks/' + encodeURIComponent(taskId), {
      credentials: 'same-origin',
      headers: headers
    });
    if (res.status === 404) {
      const err = new Error('Tugas ini sudah tidak terbit. Minta informasi terbaru kepada PJ atau KM kelas.');
      err.code = 'NOT_FOUND';
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Detail tugas belum dapat dimuat. Periksa koneksi lalu coba lagi.');
      err.code = 'LOAD_FAILED';
      throw err;
    }
    const json = await res.json();
    return json.data || null;
  },

  async getPortalMaterials(slug) {
    try {
      const res = await fetch('/api/v1/portal/' + encodeURIComponent(slug) + '/materials', {
        credentials: 'same-origin',
        headers: this.portalHeaders()
      });
      if (!res.ok) return [];
      const json = await res.json();
      return json.data || [];
    } catch (e) {
      return [];
    }
  },

  async getPortalTasks(slug, group) {    const valid = ['hari_ini', 'minggu_ini', 'mendatang', 'terlewat'];
    const g = valid.includes(group) ? group : '';
    const headers = {};
    try {
      const savedToken = localStorage.getItem('portal_token');
      const savedClass = localStorage.getItem('portal_class');
      if (savedToken && (!slug || savedClass === slug)) headers['X-Portal-Token'] = savedToken;
    } catch (e) {}
    const res = await fetch('/api/v1/portal/' + encodeURIComponent(slug) + '/tasks' + (g ? '?group=' + g : ''), {
      credentials: 'same-origin',
      headers: headers
    });
    if (res.status === 404) {
      const err = new Error('Kelas tidak ditemukan di portal. Periksa kode kelas atau hubungi KM.');
      err.code = 'CLASS_NOT_FOUND';
      err.status = 404;
      throw err;
    }
    if (res.status === 401) {
      const err = new Error('Kelas ini dilindungi kode akses. Masukkan kode 6 digit dari grup kelas.');
      err.code = 'PORTAL_UNAUTHORIZED';
      err.status = 401;
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Tugas belum dapat dimuat. Periksa koneksi lalu coba lagi.');
      err.code = 'LOAD_FAILED';
      err.status = res.status;
      throw err;
    }
    const json = await res.json();
    const arr = Array.isArray(json.data) ? json.data : [];
    return arr.map(item => {
      return {
        id: item.id,
        course_offering_id: null,
        course_code: item.offering || 'TUGAS',
        course_name: item.offering || 'Mata Kuliah',
        title: item.title,
        instructions: item.instructions,
        deadline_at: item.deadline_at,
        submission_target: item.submission_text || item.submission_url || 'LMS Kampus',
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

    // Fallback ke endpoint portal v1 (shape: {data:{items:[]}})
    try {
      const res = await fetch('/api/v1/portal/' + encodeURIComponent(slug) + '/schedule?date=' + encodeURIComponent(dateStr || ''), { credentials: 'same-origin' });
      if (res.ok) {
        const json = await res.json();
        if (json.data && json.data.items) return json.data.items;
      }
    } catch (e) {}
    return [];
  },

  async deleteTask(taskId, version) {
    // Legacy DELETE = 410 — langsung arsip via v1.
    return this.deleteTaskV1(taskId, version);
  },

  async getChanges(slug) {
    const res = await fetch('/api/v1/portal/' + encodeURIComponent(slug) + '/changes', { credentials: 'same-origin' });
    if (!res.ok) return null;
    const json = await res.json();
    return json.data || null;
  },

  async getSession() {
    // GET /api/v1/auth/me hanya mengembalikan {user, active_assignment, classes}.
    // Daftar assignments lengkap hanya ada di respons login(); gunakan itu untuk pilih konteks.
    const res = await fetch('/api/v1/auth/me', { headers: authHeaders(), credentials: 'same-origin' });
    if (res.status === 401) return { authenticated: false };
    if (res.status === 503) return { unavailable: true };
    if (!res.ok) return null;
    const json = await res.json();
    if (!json.data) return null;
    const active = json.data.active_assignment || null;
    return {
      authenticated: true,
      user: json.data.user,
      activeAssignment: active,
      activeRoleAssignmentId: active && active.id,
      classes: json.data.classes || []
    };
  },

  async login(identityKey, password) {
    const body = { identity_key: identityKey, password: password };
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

  async acceptInvitation(token, password, displayName) {
    const res = await fetch('/api/v1/invitations/accept', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: token, password: password, display_name: displayName || '' })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Undangan tidak valid atau telah digunakan.');
      err.code = (json && json.error && json.error.code) || 'INVITE_FAILED';
      err.status = res.status;
      throw err;
    }
    return json.data;
  },

  async getAdminUsers(status) {
    let url = '/api/v1/admin/users';
    if (status) url += '?status=' + encodeURIComponent(status);
    const res = await fetch(url, { headers: authHeaders(), credentials: 'same-origin' });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (!res.ok) return [];
    const json = await res.json().catch(() => null);
    return (json && json.data) || [];
  },

  async suspendUser(userId, reason) {
    const res = await fetch('/api/v1/admin/users/' + encodeURIComponent(userId) + '/suspend', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ reason: reason || '' })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal menangguhkan pengguna.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    return json.data;
  },

  async recoverUser(userId, reason) {
    const res = await fetch('/api/v1/admin/users/' + encodeURIComponent(userId) + '/recover', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ reason: reason || '' })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal memulihkan pengguna.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    return json.data;
  },

  async getMasterRooms(status) {
    let url = '/api/v1/master/rooms';
    if (status) url += '?status=' + encodeURIComponent(status);
    const res = await fetch(url, { headers: authHeaders(), credentials: 'same-origin' });
    if (!res.ok) return [];
    const json = await res.json().catch(() => null);
    return (json && json.data) || [];
  },

  async createMasterRoom(payload) {
    const res = await fetch('/api/v1/master/rooms', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal menambah ruangan.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    return json.data;
  },

  async patchMasterRoom(id, payload) {
    const res = await fetch('/api/v1/master/rooms/' + encodeURIComponent(id), {
      method: 'PATCH', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    if (!res.ok) {
      const json = await res.json().catch(() => null);
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal mengubah ruangan.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    return true;
  },

  async getMasterCourses(status) {
    let url = '/api/v1/master/courses';
    if (status) url += '?status=' + encodeURIComponent(status);
    const res = await fetch(url, { headers: authHeaders(), credentials: 'same-origin' });
    if (!res.ok) return [];
    const json = await res.json().catch(() => null);
    return (json && json.data) || [];
  },

  async createMasterCourse(payload) {
    const res = await fetch('/api/v1/master/courses', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal menambah mata kuliah.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    return json.data;
  },

  async patchMasterCourse(id, payload) {
    const res = await fetch('/api/v1/master/courses/' + encodeURIComponent(id), {
      method: 'PATCH', credentials: 'same-origin', headers: mutationHeaders(), body: JSON.stringify(payload)
    });
    if (!res.ok) {
      const json = await res.json().catch(() => null);
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal mengubah mata kuliah.');
      err.code = 'SAVE_FAILED';
      throw err;
    }
    return true;
  },

  async getAdminAssignments(status, role, classSlug) {
    let url = '/api/v1/admin/assignments';
    const qs = new URLSearchParams();
    if (status) qs.set('status', status);
    if (role) qs.set('role', role);
    if (classSlug) qs.set('class_slug', classSlug);
    if ([...qs].length) url += '?' + qs.toString();
    const res = await fetch(url, { headers: authHeaders(), credentials: 'same-origin' });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Daftar penugasan gagal dimuat.');
      err.code = 'LOAD_FAILED';
      err.status = res.status;
      throw err;
    }
    const json = await res.json().catch(() => null);
    return (json && json.data) || [];
  },

  async changeAssignmentStatus(id, aksi, reason, force) {
    const endpoint = aksi === 'cabut' ? 'revoke' : 'suspend';
    const res = await fetch('/api/v1/admin/assignments/' + encodeURIComponent(id) + '/' + endpoint, {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ reason: reason || '', force: !!force })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal mengubah penugasan.');
      err.code = res.status === 409 ? 'CONFLICT' : 'SAVE_FAILED';
      err.status = res.status;
      throw err;
    }
    return json.data;
  },

  async suspendAssignment(id, reason, force) {
    return this.changeAssignmentStatus(id, 'tangguhkan', reason, force);
  },

  async revokeAssignment(id, reason, force) {
    return this.changeAssignmentStatus(id, 'cabut', reason, force);
  },

  async supportEnter(classSlug, reason) {
    const res = await fetch('/api/v1/admin/support/enter', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ class_slug: classSlug, reason: reason })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal masuk Mode Dukungan.');
      err.code = 'SAVE_FAILED';
      err.status = res.status;
      throw err;
    }
    return json.data;
  },

  async supportExit(reason) {
    const res = await fetch('/api/v1/admin/support/exit', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ reason: reason || '' })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal keluar Mode Dukungan.');
      err.code = 'SAVE_FAILED';
      err.status = res.status;
      throw err;
    }
    return json.data;
  },

  async supportActive() {
    const res = await fetch('/api/v1/admin/support/active', {
      headers: authHeaders(), credentials: 'same-origin'
    });
    if (!res.ok) return null;
    const json = await res.json().catch(() => null);
    return (json && json.data) || null;
  },

  async testBotMessage(to, text) {
    const res = await fetch('/api/v1/admin/bot/test-message', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ to: to, text: text })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal mengirim pesan uji.');
      err.code = 'SAVE_FAILED';
      err.status = res.status;
      throw err;
    }
    return json.data;
  },

  async getAdminInvitations(status, role, classSlug) {
    let url = '/api/v1/admin/invitations';
    const qs = new URLSearchParams();
    if (status) qs.set('status', status);
    if (role) qs.set('role', role);
    if (classSlug) qs.set('class_slug', classSlug);
    if ([...qs].length) url += '?' + qs.toString();
    const res = await fetch(url, { headers: authHeaders(), credentials: 'same-origin' });
    if (res.status === 401) {
      const err = new Error('Sesi berakhir atau belum masuk.');
      err.code = 'UNAUTHORIZED';
      throw err;
    }
    if (!res.ok) {
      const err = new Error('Daftar undangan gagal dimuat.');
      err.code = 'LOAD_FAILED';
      err.status = res.status;
      throw err;
    }
    const json = await res.json().catch(() => null);
    return (json && json.data) || [];
  },

  async revokeInvitation(id, reason) {
    const res = await fetch('/api/v1/admin/invitations/' + encodeURIComponent(id) + '/revoke', {
      method: 'POST', credentials: 'same-origin', headers: mutationHeaders(),
      body: JSON.stringify({ reason: reason || '' })
    });
    const json = await res.json().catch(() => null);
    if (!res.ok) {
      const err = new Error((json && json.error && json.error.message) || (json && json.error) || 'Gagal mencabut undangan.');
      err.code = res.status === 409 ? 'CONFLICT' : 'SAVE_FAILED';
      err.status = res.status;
      throw err;
    }
    return json.data;
  },

  getAuthToken: getAuthToken,
  setAuthToken: setAuthToken
};

// Ekspor global untuk komponen Alpine.js (app-portal.js, app-km.js, app-pj.js, app-login.js, app-system-admin.js)
if (typeof window !== 'undefined') {
  window.BotApi = BotApi;
  window.API = BotApi;
}
if (typeof module !== 'undefined' && module.exports) {
  module.exports = BotApi;
}
