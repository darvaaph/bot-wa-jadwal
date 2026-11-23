const { test } = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const path = require('node:path');
const root = path.resolve(__dirname, '../..');
const source = file => fs.readFileSync(path.join(root, file), 'utf8');

function appContext(extra = {}) {
  return vm.createContext({ URL, URLSearchParams, console, document: { cookie: '' }, window: {}, ...extra });
}

test('all tasks use one request and propagate authentication/network failures', async () => {
  let calls = [];
  const ctx = appContext({ fetch: async url => { calls.push(url); return { ok: false, status: 401 }; } });
  vm.runInContext(source('web/js/api.js'), ctx);
  await assert.rejects(ctx.window.API.getAllTasks(1), e => e.status === 401 && e.code === 'UNAUTHORIZED');
  assert.equal(calls.length, 1);
  assert.match(calls[0], /tab=semua/);
  ctx.fetch = async () => { throw new Error('offline'); };
  await assert.rejects(ctx.window.API.getAllTasks(1), /offline/);
  ctx.fetch = async () => ({ ok: true, json: async () => ({ data: [] }) });
  assert.equal((await ctx.window.API.getAllTasks(1)).length, 0);
});

test('task screens show load errors instead of successful empty state', async () => {
  for (const role of ['km', 'pj']) {
    const ctx = appContext({ AsteriskShell: { behavior: () => ({}) }, examSessionBehavior: () => ({}), localStorage: { getItem: () => '' }, API: { getAllTasks: async () => { throw Object.assign(new Error('expired'), { code: 'UNAUTHORIZED' }); } } });
    vm.runInContext(source('web/js/app-' + role + '.js'), ctx);
    const factory = ctx[role + 'App'];
    assert.equal(typeof factory, 'function');
    const app = factory();
    app.updateAppBadge = () => {};
    await app.loadTasks();
    assert.match(app.tugasListError, /Sesi berakhir/);
    assert.equal(app.tugasLoading, false);
  }
});

test('exam update failures preserve saved identity, version, and form', async () => {
  for (const status of [401, 409, 503]) {
    let deletes = 0;
    const ctx = appContext({ API: {
      createTeachingEventDraft: async (payload, id) => {
        assert.equal(id, 9); assert.equal(payload.version, 4);
        throw Object.assign(new Error(status === 409 ? 'Draf berubah' : 'Simpan gagal'), { status });
      },
      deleteTeachingEvent: async () => { deletes++; }
    } });
    vm.runInContext(source('web/js/exam-session.js'), ctx);
    const app = ctx.examSessionBehavior();
    app.getMatchingPattern = () => null;
    app.examDraft = { id: 9, version: 4, reason: 'Saved draft' };
    app.examDraftIsNew = false;
    app.examForm = { offeringId: '1', date: '2026-11-09', start: '08:00', end: '09:00', kind: 'EXAM_UTS', reason: 'New input' };
    const draft = app.examDraft;
    await app.simpanUjian(false);
    assert.equal(app.examDraft, draft);
    assert.equal(app.examForm.reason, 'New input');
    assert.equal(deletes, 0);
    assert.ok(app.examError);
  }
});

test('failed draft cleanup retains the draft for recovery', async () => {
  const ctx = appContext({ API: { deleteTeachingEvent: async () => { throw new Error('offline'); } }, document: { getElementById: () => ({ open: false }) } });
  vm.runInContext(source('web/js/exam-session.js'), ctx);
  const app = ctx.examSessionBehavior();
  app.examDraft = { id: 9, version: 4 }; app.examDraftIsNew = true;
  await app.tutupDialogUjian();
  assert.equal(app.examDraft.id, 9);
  assert.equal(app.examError, 'offline');
});

test('existing invitations use the recipient login and submit no new credential', async () => {
  let received;
  const ctx = appContext({ API: { acceptInvitation: async (...args) => { received = args; return { existing_account: true }; } }, window: { location: { search: '?token=fixture-token&existing=1' }, scrollTo() {} } });
  vm.runInContext(source('web/js/app-invite.js'), ctx);
  const app = ctx.inviteApp(); app.initInvite();
  assert.equal(app.existingAccount, true);
  assert.equal(app.loginUrl(), '/login.html?invite_token=fixture-token');
  await app.aktifkan();
  assert.equal(app.done, true);
  assert.deepEqual(Array.from(received), ['fixture-token', '', '']);
});

test('draft API updates one resource with expected version and exposes conflicts', async () => {
  let request;
  const ctx = appContext({ fetch: async (url, options) => { request = { url, options }; return { ok: false, status: 409, json: async () => ({ error: { message: 'Draf berubah' } }) }; } });
  vm.runInContext(source('web/js/api.js'), ctx);
  await assert.rejects(ctx.window.API.createTeachingEventDraft({ version: 4 }, 9), e => e.status === 409 && e.code === 'CONFLICT');
  assert.equal(request.url, '/api/v1/teaching-events/9');
  assert.equal(request.options.method, 'PATCH');
  assert.equal(JSON.parse(request.options.body).version, 4);
});

test('service worker ignores every API request across accounts and removes old caches', async () => {
  const listeners = {}; let cacheReads = 0; let cacheWrites = 0; const deleted = [];
  const swCode = source('web/sw.js');
  const cacheNameMatch = swCode.match(/const\s+CACHE_NAME\s*=\s*['"]([^'"]+)['"]/);
  const currentCache = cacheNameMatch ? cacheNameMatch[1] : 'asterisk-pwa-v17';
  const ctx = vm.createContext({ URL, console,
    self: { location: { origin: 'https://app.test' }, addEventListener: (name, fn) => { listeners[name] = fn; }, clients: { claim: async () => {} }, skipWaiting() {} },
    caches: { keys: async () => ['asterisk-pwa-v14', 'asterisk-pwa-v15', 'asterisk-pwa-v16', currentCache], delete: async key => deleted.push(key), match: async () => { cacheReads++; }, open: async () => ({ put: async () => { cacheWrites++; } }) },
    fetch: async () => { throw new Error('offline'); }
  });
  vm.runInContext(swCode, ctx);
  for (const account of ['first', 'second']) {
    for (const endpoint of ['/api/v1/auth/me', '/api/v1/tasks', '/api/v1/portal/a/tasks']) {
      let handled = false;
      listeners.fetch({ request: { method: 'GET', mode: 'cors', url: 'https://app.test' + endpoint, headers: { account } }, respondWith() { handled = true; } });
      assert.equal(handled, false);
    }
  }
  let activation;
  listeners.activate({ waitUntil(p) { activation = p; } }); await activation;
  assert.deepEqual(deleted, ['asterisk-pwa-v14', 'asterisk-pwa-v15', 'asterisk-pwa-v16']);
  assert.equal(cacheReads, 0); assert.equal(cacheWrites, 0);
  ctx.fetch = async () => ({ status: 200, headers: { get: () => 'no-store' }, clone() { return this; } });
  let response;
  listeners.fetch({ request: { method: 'GET', mode: 'cors', url: 'https://app.test/js/api.js' }, respondWith(p) { response = p; } });
  await response;
  assert.equal(cacheWrites, 0);
});

test('portal keeps a successful empty week and locks on expired access without legacy fallback', async () => {
  const ctx = appContext({ AsteriskShell: { behavior: () => ({}) }, navigator: {}, localStorage: { getItem: () => '', removeItem() {} }, API: { getPortalSchedule: async () => ({ days: { '2026-11-09': [] } }) } });
  vm.runInContext(source('web/js/app-portal.js'), ctx);
  const app = ctx.portalApp();
  Object.defineProperty(app, 'selectedClassSlug', { value: 'fixture' });
  Object.defineProperty(app, 'tanggalPekan', { value: Object.fromEntries(['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'].map(day => [day, '2026-11-09'])) });
  await app.loadJadwalPekan();
  assert.equal(app.jadwalError, '');
  assert.equal(app.fullSchedule.length, 0);
  ctx.API.getPortalSchedule = async () => { throw Object.assign(new Error('expired'), { status: 401 }); };
  app.unlocked = true; app.fullSchedule = [{ title: 'previous class' }];
  await app.loadJadwalPekan();
  assert.equal(app.unlocked, false);
  assert.equal(app.fullSchedule.length, 0);
  assert.ok(app.jadwalError);
});

test('portal loads schedule and tasks in two requests and ignores a stale week response', async () => {
  const calls = [];
  const ctx = appContext({
    AsteriskShell: { behavior: () => ({}) },
    navigator: {},
    localStorage: { getItem: () => '' },
    fetch: url => new Promise(resolve => calls.push({ url, resolve }))
  });
  vm.runInContext(source('web/js/api.js'), ctx);
  ctx.API = ctx.window.API;
  vm.runInContext(source('web/js/app-portal.js'), ctx);
  const app = ctx.portalApp();
  Object.defineProperty(app, 'selectedClassSlug', { value: 'fixture' });
  const names = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
  const date = day => '2026-11-' + String(day).padStart(2, '0');
  Object.defineProperty(app, 'tanggalPekan', {
    get: () => Object.fromEntries(names.map((name, i) => [name, date(9 + i + app.weekOffset * 7)]))
  });
  app.normalisasiEfektif = items => items;
  const reply = (call, data) => call.resolve({ ok: true, json: async () => ({ data }) });
  const schedule = (start, title) => ({
    days: Object.fromEntries(names.map((_, i) => [date(start + i), i === 0 ? [{ title }] : []]))
  });
  const oldWeek = app.loadJadwalPekan();
  const tasks = app.loadTugasPortal();
  assert.equal(calls.length, 2);
  assert.match(calls[0].url, /schedule\?date=2026-11-09&days=6$/);
  assert.match(calls[1].url, /tasks\?group=semua$/);
  reply(calls[1], { hari_ini: [], minggu_ini: [], mendatang: [], terlewat: [] });
  await tasks;
  assert.equal(app.tugasError, '');
  app.weekOffset++;
  const newWeek = app.loadJadwalPekan();
  reply(calls[2], schedule(16, 'latest'));
  await newWeek;
  reply(calls[0], schedule(9, 'stale'));
  await oldWeek;
  assert.equal(app.fullSchedule[0].title, 'latest');
  assert.equal(app.jadwalLoading, false);
});

test('km settings refuses submit before classSettings loads and restores attendance minutes on toggle', async () => {
  let updateCalls = 0;
  const ctx = appContext({
    AsteriskShell: { behavior: () => ({}) },
    examSessionBehavior: () => ({}),
    localStorage: { getItem: () => '' },
    API: {
      getClassSettings: async () => { throw new Error('network error'); },
      updateClassSettings: async () => { updateCalls++; }
    }
  });
  vm.runInContext(source('web/js/app-km.js'), ctx);
  const app = ctx.kmApp();
  app.classSlug = 'd4-ti-2024-a';
  app.showToast = () => {};

  await app.loadPengaturanKelas();
  assert.equal(app.classSettings, null);
  assert.ok(app.settingsError);

  await app.simpanPengaturan();
  assert.equal(updateCalls, 0);
  assert.ok(app.pengaturanFormError);

  app.pengaturan.absenMenit = 40;
  app.onAbsenMenitInput();
  app.pengaturan.absenAktif = false;
  app.onToggleAbsen();
  app.pengaturan.absenMenit = '';
  app.pengaturan.absenAktif = true;
  app.onToggleAbsen();
  assert.equal(app.pengaturan.absenMenit, 40);
});

test('system admin settings rejects empty attendance minutes and bubble avoids slop emoji', async () => {
  let updateCalls = 0;
  const ctx = appContext({
    AsteriskShell: { behavior: () => ({}) },
    localStorage: { getItem: () => '' },
    API: {
      updateClassSettings: async () => { updateCalls++; }
    }
  });
  vm.runInContext(source('web/js/app-system-admin.js'), ctx);
  const app = ctx.systemAdminApp();
  app.kelasAktifObj = { slug: 'd4-ti-2024-a' };
  app.detailSettings = { version: 1 };
  app.pengingatForm = {
    morning_reminder_time: '06:00',
    afternoon_reminder_time: '17:00',
    replacement_reminder_minutes: 60,
    attendance_reminder_minutes: ''
  };

  await app.simpanPengingat();
  assert.equal(updateCalls, 0);
  assert.ok(app.pengingatError);

  const bubble = app.getNotifBubbleData({
    event_type: 'ATTENDANCE_REMINDER',
    payload_json: JSON.stringify({ course: 'Basis Data', starts_at: '08:00 - 10:00 WIB', room: 'LAB-1' }),
    scheduled_at: '2026-10-06T01:25:00Z'
  });
  assert.equal(bubble.title, 'PENGINGAT ABSEN KULIAH');
  assert.equal(bubble.title.includes('📋'), false);
});

test('km toggleAbsen flips state cleanly and pengaturanDirty detects unsaved or cleared values without "" -> 0 coercion', async () => {
  const ctx = appContext({
    AsteriskShell: { behavior: () => ({}) },
    examSessionBehavior: () => ({}),
    localStorage: { getItem: () => '' },
    API: {}
  });
  vm.runInContext(source('web/js/app-km.js'), ctx);
  const app = ctx.kmApp();

  assert.equal(app.pengaturanDirty, false);

  const baseSettings = {
    version: 1,
    morning_reminder_time: '06:00',
    afternoon_reminder_time: '17:00',
    replacement_reminder_minutes: 60,
    timezone: 'Asia/Jakarta',
    attendance_reminder_minutes: 25
  };
  app.classSettings = { ...baseSettings };
  assert.equal(app.pengaturanDirty, false);

  app.toggleAbsen();
  assert.equal(app.pengaturan.absenAktif, false);
  assert.equal(app.pengaturanDirty, true);

  app.toggleAbsen();
  assert.equal(app.pengaturan.absenAktif, true);
  assert.equal(app.pengaturan.absenMenit, 25);
  assert.equal(app.pengaturanDirty, false);

  app.classSettings = { ...baseSettings, attendance_reminder_minutes: 0 };
  app.pengaturan.absenAktif = false;
  app.pengaturan.absenMenit = 25;
  assert.equal(app.pengaturanDirty, false);

  app.toggleAbsen();
  app.pengaturan.absenMenit = '';
  assert.equal(app.pengaturanDirty, true);
});

test('html entrypoints bump immutable cache buster for modified KM and System Admin scripts', () => {
  const kmHtml = source('web/km.html');
  const saHtml = source('web/system-admin.html');
  const kmMatch = kmHtml.match(/\/js\/app-km\.js\?v=(\d{8})/);
  const saMatch = saHtml.match(/\/js\/app-system-admin\.js\?v=(\d{8})/);
  assert.ok(kmMatch && Number(kmMatch[1]) >= 20261123, 'web/km.html must bump app-km.js cache buster >= 20261123');
  assert.ok(saMatch && Number(saMatch[1]) >= 20261123, 'web/system-admin.html must bump app-system-admin.js cache buster >= 20261123');
});

test('attendance reminder UI uses compiled switch classes, high-contrast colors, clean copy, and loading guard', () => {
  const kmHtml = source('web/partials/km/view-pengaturan.html');
  const saHtml = source('web/partials/system-admin/view-kelas.html');
  const twCss = source('web/css/tailwind.css');
  const customCss = source('web/css/tokens.css') + '\n' + source('web/css/components.css') + '\n' + source('web/css/style.css');

  const gridLine = kmHtml.match(/<div[^>]*lg:grid-cols-2[^>]*>/);
  assert.ok(gridLine, 'KM settings 2-column grid must exist');
  assert.match(gridLine[0], /x-show="!settingsLoading && !settingsError && classSettings"/);

  const kmStart = kmHtml.indexOf('<!-- Row Pengingat Absen -->');
  const kmEnd = kmHtml.indexOf('<!-- Card Kanan:');
  assert.ok(kmStart !== -1 && kmEnd > kmStart, 'KM attendance reminder section boundaries must exist');
  const kmBlock = kmHtml.slice(kmStart, kmEnd);

  assert.match(kmBlock, /<button[^>]*id="km-absen-aktif"[^>]*role="switch"/s);
  assert.match(kmBlock, /:aria-checked="String\(pengaturan\.absenAktif\)"/);
  assert.match(kmBlock, /@click="toggleAbsen\(\)"/);
  assert.match(kmBlock, /min-h-\[44px\]/);
  assert.match(kmBlock, /min-w-\[44px\]/);
  assert.match(kmBlock, /bg-\[#3965FB\]/);
  assert.match(kmBlock, /bg-\[#64748B\]/);
  assert.match(kmBlock, /focus-visible:outline-2/);
  assert.match(kmBlock, /x-show="pengaturan\.absenAktif"/);
  assert.match(kmBlock, /<input[^>]*id="km-absen-menit"[^>]*placeholder="25"/s);
  assert.match(kmBlock, />\s*menit\s*<\/span>/);
  assert.match(kmBlock, /x-show="pengaturanDirty"/);
  assert.match(kmBlock, /Ada perubahan belum disimpan\./);

  for (const forbidden of ['peer', 'after:', 'sm:w-40', 'sm:ml-auto', 'cek akademik', 'Menit setelah kuliah dimulai', '\u2014']) {
    assert.equal(kmBlock.includes(forbidden), false, `disallowed token found in KM attendance block: ${forbidden}`);
  }
  const switchBtnMatch = kmBlock.match(/<button[^>]*id="km-absen-aktif"[^>]*>/s);
  assert.ok(switchBtnMatch && !/(?:^|\s)h-\[44px\](?=\s|")/.test(switchBtnMatch[0]), 'switch button must not lock fixed h-[44px]');

  const saStart = saHtml.indexOf('aria-labelledby="sa-pengingat-title"');
  const saEnd = saHtml.indexOf('</section>', saStart);
  assert.ok(saStart !== -1 && saEnd > saStart, 'System Admin reminder section must exist');
  const saBlock = saHtml.slice(saStart, saEnd);
  assert.match(saBlock, /\(0 = nonaktif, 1-60 menit\)/);
  const saErrorLine = saBlock.match(/<p[^>]*x-show="pengingatError"[^>]*>/);
  assert.ok(saErrorLine, 'System Admin pengingatError alert element must exist');
  assert.match(saErrorLine[0], /text-\[#B42318\]/);
  assert.equal(saErrorLine[0].includes('text-danger'), false, 'System Admin error must not use low-contrast text-danger');

  const twEscape = cls => '.' + cls.replace(/[:/\[\].#'"]/g, ch => '\\' + ch);
  const extractClassTokens = fragment => {
    const tokens = new Set();
    for (const m of fragment.matchAll(/(?<!:)\bclass="([^"]+)"/g)) {
      for (const tok of m[1].trim().split(/\s+/)) if (tok) tokens.add(tok);
    }
    for (const m of fragment.matchAll(/:class="([^"]+)"/g)) {
      for (const q of m[1].matchAll(/'([^']+)'/g)) {
        for (const tok of q[1].trim().split(/\s+/)) if (tok) tokens.add(tok);
      }
    }
    return [...tokens];
  };

  for (const tok of [...extractClassTokens(kmBlock), ...extractClassTokens(saErrorLine[0])]) {
    const sel = twEscape(tok);
    const inTw = twCss.includes(sel + '{') || twCss.includes(sel + ':');
    const inCustom = customCss.includes(sel + ' ') || customCss.includes(sel + '{') || customCss.includes(sel + ':');
    assert.ok(inTw || inCustom, `CSS class not found in precompiled stylesheets: ${tok}`);
  }
});

test('km loadPengaturanKelas keeps existing settings visible on background refresh or refresh failure and sets error when slug missing', async () => {
  let resolveRefresh;
  let callCount = 0;
  const toasts = [];
  const ctx = appContext({
    AsteriskShell: { behavior: () => ({}) },
    examSessionBehavior: () => ({}),
    localStorage: { getItem: () => '' },
    API: {
      getClassSettings: async () => {
        callCount++;
        if (callCount === 1) {
          return {
            slug: 'd4-ti-2025-a',
            version: 1,
            morning_reminder_time: '06:00',
            afternoon_reminder_time: '17:00',
            replacement_reminder_minutes: 60,
            attendance_reminder_minutes: 25,
            timezone: 'Asia/Jakarta'
          };
        }
        return new Promise((resolve, reject) => {
          resolveRefresh = reject;
        });
      }
    }
  });
  vm.runInContext(source('web/js/app-km.js'), ctx);
  const app = ctx.kmApp();
  app.showToast = msg => toasts.push(msg);

  await app.loadPengaturanKelas();
  assert.ok(app.settingsError, 'missing slug with no cached settings must set settingsError');

  app.classSlug = 'd4-ti-2025-a';
  await app.loadPengaturanKelas();
  assert.equal(app.settingsError, '');
  assert.equal(app.settingsLoading, false);
  assert.equal(app.classSettings.version, 1);

  const refreshPromise = app.loadPengaturanKelas();
  assert.equal(app.settingsLoading, false, 'background refresh must not set settingsLoading and collapse grid');
  assert.ok(app.classSettings, 'existing classSettings must stay intact during refresh');

  resolveRefresh(new Error('offline'));
  await refreshPromise;
  assert.equal(app.classSettings.version, 1, 'failed refresh must preserve existing classSettings');
  assert.equal(app.settingsError, '', 'failed background refresh must not blank out loaded screen');
  assert.ok(toasts.length > 0, 'failed background refresh should notify user via toast');
});
