# Attendance Reminder UI/UX Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Note: per repository `git-team-flow` rules, leave final git commit/push to the user instead of auto-committing.

**Goal:** Fix broken toggle rendering, cache-version drift, WCAG contrast failures, responsive height clipping, loading/dirty state gaps, and microcopy issues for the Attendance Reminder UI across KM (`Pengaturan Kelas`) and System Admin (`Detail Kelas`).

**Architecture:** Replace the uncompiled Tailwind `peer`/`after:*` checkbox hack in `web/partials/km/view-pengaturan.html` with a flattened, single-row `role="switch"` + conditional `menit` delay sub-row (`Pendekatan 1`) using only utility classes verified in `web/css/tailwind.css`, add `toggleAbsen()` and computed `pengaturanDirty` state in `web/js/app-km.js`, bump immutable asset cache busters in `web/km.html` and `web/system-admin.html`, and raise alert contrast in `web/partials/system-admin/view-kelas.html`. All behaviors and CSS invariants are pinned by automated Node tests in `tests/frontend/remediation.test.cjs` plus Go embedded-asset verification.

**Tech Stack:** Go (`web/embed.go`), plain HTML + Alpine.js v3.14.3, precompiled static Tailwind v3.4.17 (`web/css/tailwind.css`) + `tokens.css`/`components.css`, Node built-in test runner (`node --test`).

**Spec:** [2026-11-23-attendance-reminder-ui-design.md](file:///f:/proyek-lain/bot-wa-jadwal/docs/superpowers/specs/2026-11-23-attendance-reminder-ui-design.md)

## Global Constraints

- Do not add `npm`, `package.json`, CDN scripts, or any frontend CSS/JS build step (`AGENTS.md`, `ADR-0017`).
- Every Tailwind utility class added to HTML partials must already exist in `web/css/tailwind.css` (or custom stylesheets `tokens.css` / `components.css` / `style.css`). Never use `peer`, `peer-checked:*`, `after:*`, `sm:w-40`, or `sm:ml-auto`.
- Follow `ANTI_SLOP_GUIDELINES.md` and `antislop`: flat card hierarchy (`pt-4 border-t border-[#EEEEEE]`, zero nested bordered cards), no fixed `h-[44px]` around wrappable switch text, minimum `44x44px` hit target on interactive controls, non-text contrast >= 3.0:1 (`#64748B` OFF track = 4.76:1), text contrast >= 4.5:1 (`#B45309` warning = 5.02:1, `#B42318` error = 6.57:1), natural Indonesian copy without "cek akademik", duplicate ranges, or em dashes (`—`).
- Bump `?v=20261123` on `app-km.js` in `web/km.html` and `app-system-admin.js` in `web/system-admin.html` so `routes.go`'s 1-year `immutable` caching cannot serve stale JS alongside `no-store` partials.

## Review Focus

- Static CSS compilation drift: uncompiled utility classes in `view-pengaturan.html` or `view-kelas.html` silently render unstyled in production; Task 2 pins an automated class-resolution check against repo CSS.
- Immutable cache mismatch: leaving `?v=20261108` or `?v=20261026` unbumped pairs old JS with new HTML partials; Task 1 tests both HTML entry files for updated `?v=` stamps past `20261108`.
- JavaScript `Number("") === 0` coercion in `pengaturanDirty`: cleared input (`""` / whitespace) on active attendance reminder when saved value is `0` or `25` must still evaluate dirty; Task 1 tests this explicitly.
- Unloaded / failed `classSettings` state: `view-pengaturan.html` must hide the settings grid (`!settingsLoading && !settingsError && classSettings`) and `pengaturanDirty` must return `false` while `classSettings` is null; Task 1 & Task 2 cover both.
- Contrast & mobile overflow regression: `#pengingatError` in System Admin (`text-danger` `#FF6C48` = 2.80:1) fails WCAG AA, and fixed `h-[44px]` / long `whitespace-nowrap` text overflows at 320px-375px or 200% zoom; Task 2 verifies `#B42318`, `min-h-[44px]`, and short `menit` unit markup.

---

### Task 1: KM Switch/Dirty State Helpers and Entry Cache Busters

**Files:**
- Modify: `web/js/app-km.js:4841-4858`
- Modify: `web/km.html:28`
- Modify: `web/system-admin.html:26`
- Test: `tests/frontend/remediation.test.cjs:184-249`

**Interfaces:**
- Consumes existing `kmApp().pengaturan`, `kmApp().classSettings`, `kmApp().onToggleAbsen()`, `kmApp().onAbsenMenitInput()` in `web/js/app-km.js`.
- Produces `toggleAbsen(): void` and getter `get pengaturanDirty(): boolean` on `kmApp()` object literal, plus `?v=20261123` script URLs in `web/km.html` and `web/system-admin.html`.

- [ ] **Step 1: Write failing tests for `toggleAbsen()`, `pengaturanDirty`, and `?v=` cache busters**

Add two tests to `tests/frontend/remediation.test.cjs` after line 249:
1. `'km toggleAbsen flips state cleanly and pengaturanDirty detects unsaved or cleared values without "" -> 0 coercion'`:
   - Instantiate `kmApp()` in `appContext`.
   - Assert `app.pengaturanDirty === false` when `app.classSettings === null`.
   - Load matching `classSettings = { version: 1, morning_reminder_time: '06:00', afternoon_reminder_time: '17:00', replacement_reminder_minutes: 60, timezone: 'Asia/Jakarta', attendance_reminder_minutes: 25 }` with default `app.pengaturan` -> assert `app.pengaturanDirty === false`.
   - Call `app.toggleAbsen()` -> assert `app.pengaturan.absenAktif === false` and `app.pengaturanDirty === true`; call `app.toggleAbsen()` again -> assert `app.pengaturan.absenAktif === true`, `app.pengaturan.absenMenit === 25`, and `app.pengaturanDirty === false`.
   - Test `0` coercion edge case: set `app.classSettings = { ...base, attendance_reminder_minutes: 0 }`, `app.pengaturan.absenAktif = false`, `app.pengaturan.absenMenit = 25` -> assert `app.pengaturanDirty === false`. Then toggle `app.toggleAbsen()` ON and clear `app.pengaturan.absenMenit = ''` -> assert `app.pengaturanDirty === true`.
2. `'html entrypoints bump immutable cache buster for modified KM and System Admin scripts'`:
   - Read `web/km.html` and assert `/js/app-km.js?v=20261123` (or `>= 20261123`) is present (`assert.match`).
   - Read `web/system-admin.html` and assert `/js/app-system-admin.js?v=20261123` (or `>= 20261123`) is present.

- [ ] **Step 2: Run frontend tests to verify failure**

Run: `node --test tests/frontend/remediation.test.cjs`
Expected: FAIL (`app.toggleAbsen is not a function`, `?v=20261123` regex mismatch).

- [ ] **Step 3: Implement `toggleAbsen()`, `get pengaturanDirty()`, and cache-buster bumps**

1. In `web/js/app-km.js` right after `onAbsenMenitInput()` (around line 4857):
   - Add `toggleAbsen()` which flips `this.pengaturan.absenAktif = !this.pengaturan.absenAktif` and delegates to `this.onToggleAbsen()`.
   - Add `get pengaturanDirty()`:
     - Return `false` if `!this.classSettings`.
     - Compare normalized saved values (`morning_reminder_time || '06:00'`, `afternoon_reminder_time || '17:00'`, `replacement_reminder_minutes ?? 60`, `timezone || 'Asia/Jakarta'`, and `attendance_reminder_minutes` parsed as integer `0..60` defaulting to `25`) against current `this.pengaturan` values.
     - Normalize numeric inputs (`gantiMenit`, and `absenMenit` when `absenAktif` is true) by trimming strings and mapping `''` to `NaN` so `Number('')` never coerces to `0`.
2. In `web/km.html:28`, update `<script src="/js/app-km.js?v=20261123"></script>`.
3. In `web/system-admin.html:26`, update `<script src="/js/app-system-admin.js?v=20261123"></script>`.

- [ ] **Step 4: Run frontend tests to verify Task 1 passes**

Run: `node --test tests/frontend/remediation.test.cjs`
Expected: PASS (all existing 11 tests + 2 new Task 1 tests pass).

---

### Task 2: Unified Attendance Reminder Layout (`Pendekatan 1`), Copy Cleanup, Loading Guard, and Contrast Fixes

**Files:**
- Modify: `web/partials/km/view-pengaturan.html:66,145-200`
- Modify: `web/partials/system-admin/view-kelas.html:880-881`
- Test: `tests/frontend/remediation.test.cjs`

**Interfaces:**
- Consumes `settingsLoading`, `settingsError`, `classSettings`, `pengaturan.absenAktif`, `pengaturan.absenMenit`, `toggleAbsen()`, `onAbsenMenitInput()`, `pengaturanDirty`, `pengaturanSaving`, `simpanPengaturan()` from Task 1 / `kmApp()`.
- Consumes `#pengingatError` and `pengingatForm.attendance_reminder_minutes` in `web/partials/system-admin/view-kelas.html`.

- [ ] **Step 1: Write failing static HTML/CSS verification test**

Add test `'attendance reminder UI uses compiled switch classes, high-contrast colors, clean copy, and loading guard'` to `tests/frontend/remediation.test.cjs`:
- Extract the `Waktu Pengingat & Zona` / attendance section of `web/partials/km/view-pengaturan.html` and `Jadwal Notifikasi Otomatis` of `web/partials/system-admin/view-kelas.html`.
- Assert `view-pengaturan.html` main 2-column grid checks `x-show="!settingsLoading && !settingsError && classSettings"`.
- Assert `km-absen-aktif` is a `<button` with `role="switch"`, `:aria-checked`, `@click="toggleAbsen()"`, `min-h-[44px]`, `min-w-[44px]`, `bg-[#3965FB]`, `bg-[#64748B]`, `focus-visible:outline-2`.
- Assert `km-absen-menit` container uses `x-show="pengaturan.absenAktif"`, has `placeholder="25"`, short unit `>menit<`, and no `Menit setelah kuliah dimulai`.
- Assert footer displays `pengaturanDirty` notice (`Ada perubahan belum disimpan.`).
- Assert no disallowed uncompiled/slop strings in the KM attendance block: zero occurrences of `peer`, `after:`, `sm:w-40`, `sm:ml-auto`, `cek akademik`, or em dash `\u2014`.
- Collect every static Tailwind class token inside the modified KM attendance block (`<!-- Row Pengingat Absen -->` through `simpanPengaturan()` footer) and System Admin `#pengingatError` line, and verify each token resolves in `web/css/tailwind.css` or `web/css/tokens.css` / `components.css` / `style.css` (using proper Tailwind CSS selector escaping check, ignoring Alpine attribute names).
- Assert `view-kelas.html` uses `text-[#B42318]` (not `text-danger`) on the `pengingatError` `<p>` and uses ASCII hyphen `1-60 menit` in the attendance reminder label.

- [ ] **Step 2: Run frontend tests to verify failure**

Run: `node --test tests/frontend/remediation.test.cjs`
Expected: FAIL on old `peer`/`after:` checkbox markup, missing `role="switch"`, missing grid `x-show`, `cek akademik` string, and `text-danger` in `view-kelas.html`.

- [ ] **Step 3: Implement `Pendekatan 1` in `view-pengaturan.html` and contrast/copy fix in `view-kelas.html`**

1. In `web/partials/km/view-pengaturan.html`:
   - Line 66: add `x-show="!settingsLoading && !settingsError && classSettings" x-cloak` to the 2-column grid container.
   - Replace lines 145-185 (`<!-- Row Pengingat Absen -->`) with flattened single-row unified panel (`pt-4 border-t border-[#EEEEEE] flex flex-col gap-3`):
     - Top row (`flex items-start justify-between gap-3`):
       - Left (`min-w-0 flex-1`): `<span id="km-absen-label" class="text-[12px] font-semibold text-[#1D1B3A]">Pengingat Absen Kuliah</span>` and `<p id="km-absen-desc" class="text-[11px] leading-relaxed mt-0.5" :class="pengaturan.absenAktif ? 'text-[#667085]' : 'text-[#B45309] font-medium'" x-text="pengaturan.absenAktif ? 'Kirim pengingat presensi otomatis ke grup WhatsApp saat kuliah berlangsung.' : 'Nonaktif. Pengingat presensi otomatis tidak dikirim.'">Kirim pengingat presensi otomatis ke grup WhatsApp saat kuliah berlangsung.</p>`
       - Right (`shrink-0`): `<button type="button" id="km-absen-aktif" role="switch" :aria-checked="String(pengaturan.absenAktif)" aria-labelledby="km-absen-label" aria-describedby="km-absen-desc" @click="toggleAbsen()" class="inline-flex items-center justify-center min-h-[44px] min-w-[44px] -mr-2 rounded-[10px] cursor-pointer focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#3965FB] focus-visible:outline-offset-2">` containing visual track `<span class="relative inline-flex items-center h-[24px] w-[40px] rounded-full p-[2px] transition-colors duration-200 ease-in-out" :class="pengaturan.absenAktif ? 'bg-[#3965FB]' : 'bg-[#64748B]'">` and thumb `<span aria-hidden="true" class="pointer-events-none inline-block h-[20px] w-[20px] transform rounded-full bg-white shadow-sm ring-0 transition duration-200 ease-in-out" :class="pengaturan.absenAktif ? 'translate-x-4' : 'translate-x-0'"></span></span></button>`.
     - Conditional delay row (`<div x-show="pengaturan.absenAktif" x-cloak class="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 pt-1">`):
       - Left (`min-w-0`): `<label for="km-absen-menit" class="text-[12px] font-semibold text-[#1D1B3A]">Jeda Pengingat</label>` and `<p id="km-absen-menit-help" class="text-[11px] text-[#667085] leading-relaxed">Dikirim 1-60 menit setelah sesi kuliah dimulai.</p>`
       - Right input wrapper (`w-full sm:w-[170px] min-h-[44px] px-3 flex items-center justify-between gap-2 bg-white border border-[#D7DDE7] rounded-[10px] focus-within:border-[#3965FB] focus-within:ring-2 focus-within:ring-[#3965FB]/20 transition-all shrink-0`):
         - `<input id="km-absen-menit" type="number" min="1" max="60" step="1" placeholder="25" aria-describedby="km-absen-menit-help" x-model.number="pengaturan.absenMenit" @input="onAbsenMenitInput()" class="w-16 min-w-0 font-mono text-[13px] font-medium text-[#1D1B3A] bg-transparent border-none p-0 focus:outline-none focus:ring-0" />`
         - `<span class="text-[11px] text-[#667085] whitespace-nowrap select-none font-medium">menit</span>`
   - Update card footer (line 189) to `flex flex-col sm:flex-row sm:items-center justify-between gap-3` with `<span x-show="pengaturanDirty" x-cloak role="status" class="text-[12px] font-medium text-[#B45309]">Ada perubahan belum disimpan.</span>` and `ml-auto` on the `Simpan Pengaturan` button.
2. In `web/partials/system-admin/view-kelas.html`:
   - Line 880: change `(0 = nonaktif, 1–60 menit)` to `(0 = nonaktif, 1-60 menit)`.
   - Line 881: replace `text-danger` with `text-[#B42318]`.

- [ ] **Step 4: Run full frontend, contrast, Go test, and static checks**

Run:
1. `node --test tests/frontend/remediation.test.cjs` -> Expected: PASS (14/14 passing, 0 skipped).
2. `go test -count=1 ./...` -> Expected: PASS across all packages (verifying `web/embed.go` embeds updated HTML/JS cleanly).
3. `go vet ./...` -> Expected: exit 0 with no issues.
4. `python C:\Users\DARVA\.gemini\config\skills\antislop-human\contrast-check.py "#64748B" "#FFFFFF"` -> Expected: >= 3.0:1 PASS (`4.76:1`).
5. `python C:\Users\DARVA\.gemini\config\skills\antislop-human\contrast-check.py "#B42318" "#FFFFFF"` -> Expected: >= 4.5:1 PASS (`6.57:1`).
6. Run `antislop` checklist review across changed lines (copy, mobile width/no overflow, contrast, interactive states, keyboard `:focus-visible`).
