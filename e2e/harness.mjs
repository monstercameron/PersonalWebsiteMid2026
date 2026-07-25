// harness.mjs — shared rig for the end-to-end suites.
//
// Every suite that imports this gets a HERMETIC instance: its own port, its own
// site DB, its own CashFlux data dir, its own fresh owner account. None of it ever
// touches a running server, so a suite can be run while the real one is up.
//
// Assertions in the suites go through the two REAL UIs rather than the SQLite
// file: the admin console is the oracle for server state, and if the console
// cannot show something then the operator cannot see it either — a test that peeks
// behind the UI passes while the operator is still in the dark.

import { spawn, spawnSync } from 'node:child_process';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { chromium } from 'file:///C:/Users/mreca/Desktop/CashFlux/node_modules/playwright/index.mjs';

export const PORT = Number(process.env.E2E_PORT || 8211);
export const BASE = `http://127.0.0.1:${PORT}`;
export const KEEP = process.argv.includes('--keep');
export const OWNER = { user: 'e2eowner', pass: 'e2e-password-123' };
// Unique per run so a stale profile or leftover data can never make a check pass.
export const MARKER = `E2E-MARKER-${process.pid}-${Date.now().toString(36)}`;

// Generous but bounded: the CashFlux wasm bundle is ~94MB uncompressed, so a cold
// first paint genuinely takes ~15s on this machine. Every wait below is an explicit
// condition with a timeout, never a bare sleep hoping something finished.
export const BOOT_MS = 180_000;
export const APP_READY_MS = 90_000;

export let failures = 0;
export let scenario = '';
export const step = (msg) => console.log(`   · ${msg}`);
export const check = (label, ok, detail = '') => {
  console.log(`   ${ok ? 'PASS' : 'FAIL'}  ${label}${detail ? ` — ${detail}` : ''}`);
  if (!ok) failures++;
};

// ---------------------------------------------------------------- server harness

export function startServer(dataDir) {
  const proc = spawn('./bin/server.exe', [], {
    cwd: process.cwd(),
    env: {
      ...process.env,
      LISTEN_ADDR: `127.0.0.1:${PORT}`,
      BASE_URL: BASE,
      DB_PATH: join(dataDir, 'site.db'),
      CASHFLUX_DATA_DIR: join(dataDir, 'cashflux'),
      ADMIN_SECRET: 'e2e-secret-stable',
      // Deliberately NO ADMIN_PASSWORD: that would also switch on the /budget/
      // password gate (config.go falls BudgetPassword back to it), adding a step
      // these scenarios aren't about.
      ADMIN_PASSWORD: '',
      ADMIN_USERNAME: '',
      // The tunnel throttles per CLIENT, and every browser context this rig opens
      // — the admin console, device A, device B — shares 127.0.0.1, so the server
      // sees one very busy client rather than three ordinary ones. The default cap
      // of 8 connections is ample for a real device and nowhere near enough for a
      // rig that also churns connections across sign-out and re-activation; the
      // last scenario in sync-flows runs after all of it and was the one that paid.
      // Raised HERE, in the test environment, rather than in the product: the
      // throttle is doing its job, this rig is simply not what it is defending
      // against. Anything that would fail in production still fails here.
      CASHFLUX_SERVER_GRPC_MAX_CONNECTIONS_PER_CLIENT: '256',
      CASHFLUX_SERVER_GRPC_MAX_UPGRADES_PER_CLIENT_PER_MINUTE: '1000',
    },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  proc.stdout.on('data', (b) => process.env.E2E_VERBOSE && process.stdout.write(`[srv] ${b}`));
  proc.stderr.on('data', (b) => process.env.E2E_VERBOSE && process.stdout.write(`[srv!] ${b}`));
  return proc;
}

export async function waitForServer() {
  for (let i = 0; i < 60; i++) {
    try {
      const r = await fetch(`${BASE}/v1/version`);
      if (r.ok) return true;
    } catch { /* not up yet */ }
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error('server never came up');
}

// ---------------------------------------------------------------- admin console

export async function adminPage(browser, { firstRun }) {
  const page = await (await browser.newContext({ viewport: { width: 1280, height: 900 } })).newPage();
  await page.goto(`${BASE}/admin`, { waitUntil: 'networkidle', timeout: BOOT_MS });
  await page.waitForTimeout(3500);
  await page.locator('input').nth(0).fill(OWNER.user);
  await page.locator('input').nth(1).fill(OWNER.pass);
  if (firstRun) {
    await page.getByText('Create account', { exact: true }).click();
    await page.waitForTimeout(3000);
    const cont = page.getByText(/Continue to admin/i).first();
    if (await cont.count()) { await cont.click(); await page.waitForTimeout(2500); }
  } else {
    await page.locator('button', { hasText: 'Sign in' }).first().click();
    await page.waitForTimeout(3500);
  }
  return page;
}

export async function openCashfluxTab(page) {
  const tab = page.getByText('cashflux', { exact: false }).first();
  if (await tab.count()) await tab.click();
  else await page.goto(`${BASE}/admin/cashflux`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(3000);
}

export async function mintCode(page) {
  await openCashfluxTab(page);
  await page.getByText(/^Generate (code|another)$/).first().click();
  await page.waitForTimeout(2500);
  const m = (await page.locator('body').innerText()).match(/\b(\d{6})\b/);
  if (!m) throw new Error('no activation code appeared in the admin console');
  return m[1];
}

// serverView scrapes the console's own reporting — the operator's view of the truth.
export async function serverView(page) {
  await page.reload({ waitUntil: 'networkidle' });
  await page.waitForTimeout(2000);
  await openCashfluxTab(page);
  const text = await page.locator('body').innerText();
  const syncedData = (text.match(/SYNCED DATA\s*\n\s*([^\n]+)/i) || [])[1] || '';
  const users = Number((text.match(/USERS \((\d+)\)/i) || [])[1] || 0);
  const neverSynced = /never synced/i.test(text);
  // Anchored to the shapes userSyncedLabel actually emits. A loose /synced (.+)/
  // matched the "SYNCED DATA" storage-tile heading instead of the user row.
  const syncedAgoMatch = text.match(/synced (just now|\d+[mh] ago|[A-Z][a-z]{2} \d+)/);
  const syncedAgo = syncedAgoMatch ? syncedAgoMatch[1] : '';
  return { syncedData: syncedData.trim(), users, neverSynced, syncedAgo: syncedAgo.trim(), text };
}

// ---------------------------------------------------------------- cashflux client

export async function newClient(browser) {
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 900 } });
  const page = await ctx.newPage();
  // Count page loads so a test can prove a pull APPLIED (an applying pull reloads).
  await page.addInitScript(() => {
    const n = Number(sessionStorage.getItem('__loads') || 0) + 1;
    sessionStorage.setItem('__loads', String(n));
  });
  page.on('pageerror', (e) => console.log(`   ! pageerror: ${String(e).slice(0, 160)}`));
  // Surface the app's own ERROR/WARN lines. A silent failure inside the wasm (a
  // failed import, a deferred decrypt) otherwise shows up only as a wrong-looking
  // screen, with the reason sitting unread in the console.
  page.on('console', (m) => {
    const t = m.text();
    if (/\[ERROR\]|\[WARN\]/.test(t)) console.log(`   ! app: ${t.slice(0, 200)}`);
  });
  return { ctx, page };
}

// openCloudTab navigates straight to the Cloud tab's own route and waits for the
// pane to actually be there.
//
// It used to load /budget/settings and click a "Cloud" tab IF one happened to be on
// screen already, then sleep 2500ms. Both halves were wrong. The conditional click
// meant that whenever the tab list had not rendered yet the click was silently
// skipped, the helper returned successfully, and the scenario ran its assertions
// against the plain Settings page — reporting "a signed-in device offers Sign out"
// as a product failure when the truth was that nothing had ever opened the pane
// holding that control. The flat sleep then decided how long rendering was allowed
// to take. /settings/:tab is a real route, so ask for the pane directly and wait for
// evidence it arrived.
export async function openCloudTab(page) {
  // Retry the navigation: an applying pull calls reloadPage() inside the app, which
  // can interrupt a goto issued at the same moment. That is the product behaving
  // correctly — a hydrating device is supposed to reload — so the test accommodates
  // it rather than treating it as a failure.
  for (let attempt = 0; ; attempt++) {
    try {
      await page.goto(`${BASE}/budget/settings/cloud`, { waitUntil: 'load', timeout: BOOT_MS });
      break;
    } catch (e) {
      if (attempt >= 2 || !/interrupted by another navigation/.test(String(e))) throw e;
      await page.waitForTimeout(3000);
    }
  }
  await waitForApp(page);
  // The pane looks different in each of its states — sync off, discovering, paired,
  // locked — so key off the fact that ANY of its own controls rendered rather than
  // one state's control. sync-pulse is excluded deliberately: it is the top-bar
  // indicator, present on every screen, and matching it would mean this helper
  // returned successfully from pages that are not the Cloud pane at all.
  await page.waitForFunction(
    () => !!document.querySelector('[data-testid^="sync-"]:not([data-testid="sync-pulse"])')
      || !!document.querySelector('[data-testid="device-link-code"]')
      || /Sign out/.test(document.querySelector('#app')?.innerText || ''),
    null,
    { timeout: 45_000 },
  );
}

// waitForApp waits for the wasm shell to actually paint, instead of guessing a duration.
export async function waitForApp(page) {
  await page.waitForFunction(
    () => !!document.querySelector('[data-testid="sync-pulse"], .topbar, .rail-nav'),
    null,
    { timeout: APP_READY_MS },
  );
  await page.waitForTimeout(2500);
}

export async function enableSync(page) {
  if (await page.getByTestId('sync-discovery-ok').count()) return;
  const toggles = page.locator('input[type=checkbox], [role=switch]');
  for (let i = 0; i < (await toggles.count()); i++) {
    await toggles.nth(i).click({ force: true }).catch(() => {});
    await page.waitForTimeout(2500);
    if (await page.getByTestId('sync-discovery-ok').count()) return;
  }
  throw new Error('could not turn sync on / discovery never succeeded');
}

export async function activate(page, code) {
  // Fill and submit, retrying across reloads the app performs on its own.
  //
  // A pull that applies a remote snapshot calls reloadPage(), and a sign-out
  // immediately followed by a re-activation is exactly when one is most likely to be
  // in flight. When that reload lands between locating the field and filling it, the
  // handle goes stale and fill() times out — reported as "the activation field is
  // missing" when it was there before the reload and is there again after. Re-open
  // the pane and try again rather than calling a reload a failure.
  for (let attempt = 0; ; attempt++) {
    try {
      await page.getByTestId('device-link-code').fill(code, { timeout: 30_000 });
      await page.getByTestId('device-link-submit').click({ timeout: 15_000 });
      break;
    } catch (e) {
      if (attempt >= 2) throw e;
      await page.waitForTimeout(4000);
      await openCloudTab(page);
    }
  }
  // The pull that follows may reload the page; either way give it room to land.
  await page.waitForTimeout(15_000);
}

export const syncState = (page) => page.getByTestId('sync-pulse').getAttribute('data-sync-state').catch(() => null);

// waitForHealthySync polls until the client reports a state that means "signed in
// and talking to the server", or gives up.
//
// Sampling the state once was both flaky and too weak: the indicator can legitimately
// still be mounting when we look, and "error" — which a bare non-null check accepts —
// is exactly the state a failed sign-in produces. Several browser contexts hammering
// one server also trip its per-client upgrade throttle, so a transient "error" on the
// way to "synced" is normal here and worth waiting through rather than asserting on.
export async function waitForHealthySync(page, timeoutMs = 60_000) {
  const healthy = new Set(['synced', 'syncing']);
  const deadline = Date.now() + timeoutMs;
  let last = null;
  while (Date.now() < deadline) {
    last = await syncState(page);
    if (last && healthy.has(last)) return last;
    await page.waitForTimeout(2000);
  }
  return last;
}

// addMarkerTransaction writes a uniquely-named transaction. This is the real
// end-to-end payload: asserting that a SPECIFIC record a person entered on one
// device shows up on another proves data retrieval in a way that byte counts and
// status labels cannot — both of those were green in this suite while a second
// browser was still showing its own untouched demo data.
export async function addMarkerTransaction(page, marker) {
  await page.goto(`${BASE}/budget/transactions`, { waitUntil: 'load', timeout: BOOT_MS });
  await waitForApp(page);
  await page.getByTestId('add-transaction-btn').click();
  await page.waitForTimeout(1500);
  await page.getByTestId('txn-add-amount').fill('42.42');
  await page.getByTestId('txn-add-desc').fill(marker);
  await page.getByTestId('txn-add-another').click();
  await page.waitForTimeout(3000);
  await page.keyboard.press('Escape');
  await page.waitForTimeout(2500);
}

// hasMarker looks for the marker anywhere in the transactions screen.
// hasMarker polls rather than sleeping a fixed interval. The app deliberately defers
// sync until the renderer is idle, so "how long until a hydrated row is on screen"
// is a range, not a constant — a flat sleep turns that range into a coin flip and
// reports the harness's impatience as a product failure. Polling asserts exactly the
// same thing (the row IS there) without pretending to know when.
export async function hasMarker(page, marker, timeoutMs = 30_000) {
  await page.goto(`${BASE}/budget/transactions`, { waitUntil: 'load', timeout: BOOT_MS });
  await waitForApp(page);
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    if ((await page.locator('body').innerText()).includes(marker)) return true;
    if (Date.now() > deadline) return false;
    await page.waitForTimeout(1000);
  }
}


// failed lets a suite report its own exit status without importing module state
// back and forth: check() increments the shared counter, this reads it.
export const failed = () => failures;
export function bumpFailure() { failures++; }
export { chromium };
