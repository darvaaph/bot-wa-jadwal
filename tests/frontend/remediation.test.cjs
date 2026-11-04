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
  const ctx = vm.createContext({ URL, console,
    self: { location: { origin: 'https://app.test' }, addEventListener: (name, fn) => { listeners[name] = fn; }, clients: { claim: async () => {} }, skipWaiting() {} },
    caches: { keys: async () => ['asterisk-pwa-v14', 'asterisk-pwa-v15', 'asterisk-pwa-v16', 'asterisk-pwa-v17'], delete: async key => deleted.push(key), match: async () => { cacheReads++; }, open: async () => ({ put: async () => { cacheWrites++; } }) },
    fetch: async () => { throw new Error('offline'); }
  });
  vm.runInContext(source('web/sw.js'), ctx);
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
