// sync-flows.mjs — end-to-end coverage for the embedded CashFlux sync lifecycle.
//
// These are the flows a person actually walks through, asserted through the two real
// UIs rather than through internals: the admin console at /admin/cashflux is the
// ORACLE for server state (it reports synced bytes and last-sync time per account),
// and the CashFlux client at /budget/ is the subject. Nothing here reads the SQLite
// file directly — if the console can't show it, the operator can't see it either, and
// a test that peeks behind the UI would pass while the operator is still in the dark.
//
// Every run is hermetic: its own port, its own site DB, its own CashFlux data dir,
// its own fresh owner account. It never touches a running instance.
//
// Usage:
//   node e2e/sync-flows.mjs            # all scenarios
//   node e2e/sync-flows.mjs --keep     # leave the server up afterwards for poking
//
// Requires: bin/server.exe and web/cashflux/bin/main.wasm built (scripts/dev.sh,
// scripts/build-cashflux.sh), plus playwright resolvable from the CashFlux checkout.

import { spawn, spawnSync } from 'node:child_process';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { chromium } from 'file:///C:/Users/mreca/Desktop/CashFlux/node_modules/playwright/index.mjs';

const PORT = Number(process.env.E2E_PORT || 8211);
const BASE = `http://127.0.0.1:${PORT}`;
const KEEP = process.argv.includes('--keep');
const OWNER = { user: 'e2eowner', pass: 'e2e-password-123' };
// Unique per run so a stale profile or leftover data can never make a check pass.
const MARKER = `E2E-MARKER-${process.pid}-${Date.now().toString(36)}`;

// Generous but bounded: the CashFlux wasm bundle is ~94MB uncompressed, so a cold
// first paint genuinely takes ~15s on this machine. Every wait below is an explicit
// condition with a timeout, never a bare sleep hoping something finished.
const BOOT_MS = 180_000;
const APP_READY_MS = 90_000;

let failures = 0;
let scenario = '';
const step = (msg) => console.log(`   · ${msg}`);
const check = (label, ok, detail = '') => {
  console.log(`   ${ok ? 'PASS' : 'FAIL'}  ${label}${detail ? ` — ${detail}` : ''}`);
  if (!ok) failures++;
};

// ---------------------------------------------------------------- server harness

function startServer(dataDir) {
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
    },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  proc.stdout.on('data', (b) => process.env.E2E_VERBOSE && process.stdout.write(`[srv] ${b}`));
  proc.stderr.on('data', (b) => process.env.E2E_VERBOSE && process.stdout.write(`[srv!] ${b}`));
  return proc;
}

async function waitForServer() {
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

async function adminPage(browser, { firstRun }) {
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

async function openCashfluxTab(page) {
  const tab = page.getByText('cashflux', { exact: false }).first();
  if (await tab.count()) await tab.click();
  else await page.goto(`${BASE}/admin/cashflux`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(3000);
}

async function mintCode(page) {
  await openCashfluxTab(page);
  await page.getByText(/^Generate (code|another)$/).first().click();
  await page.waitForTimeout(2500);
  const m = (await page.locator('body').innerText()).match(/\b(\d{6})\b/);
  if (!m) throw new Error('no activation code appeared in the admin console');
  return m[1];
}

// serverView scrapes the console's own reporting — the operator's view of the truth.
async function serverView(page) {
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

async function newClient(browser) {
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

async function openCloudTab(page) {
  await page.goto(`${BASE}/budget/settings`, { waitUntil: 'load', timeout: BOOT_MS });
  await waitForApp(page);
  const cloud = page.getByText(/^cloud$/i).first();
  if (await cloud.count()) { await cloud.click(); await page.waitForTimeout(2500); }
}

// waitForApp waits for the wasm shell to actually paint, instead of guessing a duration.
async function waitForApp(page) {
  await page.waitForFunction(
    () => !!document.querySelector('[data-testid="sync-pulse"], .topbar, .rail-nav'),
    null,
    { timeout: APP_READY_MS },
  );
  await page.waitForTimeout(2500);
}

async function enableSync(page) {
  if (await page.getByTestId('sync-discovery-ok').count()) return;
  const toggles = page.locator('input[type=checkbox], [role=switch]');
  for (let i = 0; i < (await toggles.count()); i++) {
    await toggles.nth(i).click({ force: true }).catch(() => {});
    await page.waitForTimeout(2500);
    if (await page.getByTestId('sync-discovery-ok').count()) return;
  }
  throw new Error('could not turn sync on / discovery never succeeded');
}

async function activate(page, code) {
  await page.getByTestId('device-link-code').fill(code);
  await page.getByTestId('device-link-submit').click();
  // The pull that follows may reload the page; either way give it room to land.
  await page.waitForTimeout(15_000);
}

const syncState = (page) => page.getByTestId('sync-pulse').getAttribute('data-sync-state').catch(() => null);

// addMarkerTransaction writes a uniquely-named transaction. This is the real
// end-to-end payload: asserting that a SPECIFIC record a person entered on one
// device shows up on another proves data retrieval in a way that byte counts and
// status labels cannot — both of those were green in this suite while a second
// browser was still showing its own untouched demo data.
async function addMarkerTransaction(page, marker) {
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
async function hasMarker(page, marker) {
  await page.goto(`${BASE}/budget/transactions`, { waitUntil: 'load', timeout: BOOT_MS });
  await waitForApp(page);
  await page.waitForTimeout(3000);
  return (await page.locator('body').innerText()).includes(marker);
}

// ---------------------------------------------------------------- scenarios

async function run() {
  const dataDir = mkdtempSync(join(tmpdir(), 'cashflux-e2e-'));
  const server = startServer(dataDir);
  await waitForServer();
  const browser = await chromium.launch();

  try {
    // ---- 1. New account creation + pairing -------------------------------
    scenario = 'new account creation + pairing';
    console.log(`\n[1] ${scenario}`);
    const admin = await adminPage(browser, { firstRun: true });
    const code1 = await mintCode(admin);
    step(`minted ${code1}`);

    const before = await serverView(admin);
    check('minting creates exactly one account', before.users === 1, `users=${before.users}`);
    check('a brand-new account reports "never synced"', before.neverSynced, before.text.includes('never synced') ? '' : `syncedAgo=${before.syncedAgo}`);

    const A = await newClient(browser);
    await openCloudTab(A.page);
    await enableSync(A.page);
    check('an unactivated device shows no sync indicator', (await A.page.getByTestId('sync-pulse').count()) === 0);
    await activate(A.page, code1);
    check('activation signs the device in', (await syncState(A.page)) !== null, `state=${await syncState(A.page)}`);

    // ---- 2. The sample dataset must never reach the server ---------------
    scenario = 'sample data is never uploaded';
    console.log(`\n[2] ${scenario}`);
    const afterActivate = await serverView(admin);
    check('an activated-but-unpersonalised device uploads nothing', afterActivate.neverSynced,
      `syncedData=${afterActivate.syncedData} syncedAgo=${afterActivate.syncedAgo}`);

    // ---- 3. Real data pairs and lands ------------------------------------
    scenario = 'real data uploads';
    console.log(`\n[3] ${scenario}`);
    await A.page.goto(`${BASE}/budget/`, { waitUntil: 'load', timeout: BOOT_MS });
    await waitForApp(A.page);
    const banner = A.page.getByTestId('sample-data-banner');
    check('the sample banner is showing before personalising', (await banner.count()) === 1);
    await A.page.getByTestId('sample-dismiss').click(); // marks the dataset as the user's own
    await A.page.waitForTimeout(4000);
    check('dismissing clears the sample banner on this device',
      (await A.page.getByTestId('sample-data-banner').count()) === 0);

    await addMarkerTransaction(A.page, MARKER);
    check('the marker transaction is on the first device', await hasMarker(A.page, MARKER), MARKER);
    await A.page.getByTestId('sync-pulse').click().catch(() => {});
    await A.page.waitForTimeout(12_000);

    const afterPush = await serverView(admin);
    check('personalised data reaches the server', !afterPush.neverSynced && afterPush.syncedData !== '0 B',
      `syncedData=${afterPush.syncedData}`);
    check('the console reports when it last synced', /just now|m ago|h ago/.test(afterPush.syncedAgo),
      `syncedAgo=${afterPush.syncedAgo}`);

    // ---- 4. A second browser hydrates ------------------------------------
    scenario = 'second browser hydrates';
    console.log(`\n[4] ${scenario}`);
    const code2 = await mintCode(admin);
    const B = await newClient(browser);
    await openCloudTab(B.page);
    await enableSync(B.page);
    await activate(B.page, code2);
    await B.page.goto(`${BASE}/budget/`, { waitUntil: 'load', timeout: BOOT_MS });
    await waitForApp(B.page);

    check('the second browser is signed in', (await syncState(B.page)) !== null, `state=${await syncState(B.page)}`);
    // An applying pull calls reloadPage(), so a load count that never grew proves the
    // remote snapshot was never applied — which separates "hydrate failed outright"
    // from "hydrated, but the sample flag survived the import".
    const loadsB = await B.page.evaluate(() => Number(sessionStorage.getItem('__loads') || 0));
    check('the second browser applied the remote snapshot (the pull reloaded it)', loadsB >= 2, `loads=${loadsB}`);
    const flagProbe = await B.page.evaluate(() =>
      (window.cashfluxStoreGet ? window.cashfluxStoreGet('cashflux:sampleActive') : 'no-bridge'));
    check('the second browser shows NO sample banner after hydrating',
      (await B.page.getByTestId('sample-data-banner').count()) === 0, `browserstore flag=${JSON.stringify(flagProbe)}`);
    check('the second browser sees the transaction entered on the first',
      await hasMarker(B.page, MARKER), MARKER);

    const afterHydrate = await serverView(admin);
    check('hydrating did not overwrite the server copy',
      !afterHydrate.neverSynced && afterHydrate.syncedData === afterPush.syncedData,
      `before=${afterPush.syncedData} after=${afterHydrate.syncedData}`);
    check('both devices share one account', afterHydrate.users === 1, `users=${afterHydrate.users}`);

    // ---- 5. Secondary login on a device that already has data ------------
    scenario = 'secondary login keeps data';
    console.log(`\n[5] ${scenario}`);
    await openCloudTab(A.page);
    const signOut = A.page.getByText(/^Sign out$/).first();
    check('a signed-in device offers Sign out', (await signOut.count()) === 1);
    await signOut.click();
    await A.page.waitForTimeout(5000);
    check('signing out returns the activation field', (await A.page.getByTestId('device-link-code').count()) === 1);

    const code3 = await mintCode(admin);
    await openCloudTab(A.page);
    await activate(A.page, code3);
    check('re-activating signs back in', (await syncState(A.page)) !== null, `state=${await syncState(A.page)}`);

    const afterRelogin = await serverView(admin);
    check('a secondary login does not lose server data',
      !afterRelogin.neverSynced && afterRelogin.syncedData === afterPush.syncedData,
      `before=${afterPush.syncedData} after=${afterRelogin.syncedData}`);
    check('a secondary login does not create a second account', afterRelogin.users === 1, `users=${afterRelogin.users}`);
  } finally {
    if (!KEEP) {
      await browser.close().catch(() => {});
      // Argument array, no shell: nothing here is interpolated into a command string.
      spawnSync('taskkill', ['/PID', String(server.pid), '/F', '/T'], { stdio: 'ignore' });
      try { rmSync(dataDir, { recursive: true, force: true }); } catch { /* windows lock */ }
    } else {
      console.log(`\n(--keep) server still running at ${BASE}, data in ${dataDir}`);
    }
  }

  console.log(`\n${failures === 0 ? 'ALL SCENARIOS PASSED' : `${failures} CHECK(S) FAILED`}`);
  process.exit(failures === 0 ? 0 : 1);
}

run().catch((e) => {
  console.error(`\nsuite aborted during "${scenario}":`, e);
  process.exit(2);
});
