// sync-flows.mjs — the sync lifecycle: pairing, upload, hydration, re-login.
// Rig and helpers live in harness.mjs; see its header for why assertions go
// through the real UIs rather than the database.
//
// Usage:  node e2e/sync-flows.mjs [--keep]

import {
  BASE, KEEP, MARKER, BOOT_MS, chromium,
  check, step, failed,
  startServer, waitForServer, adminPage, mintCode, serverView,
  newClient, openCloudTab, enableSync, activate, waitForHealthySync,
  waitForApp, addMarkerTransaction, hasMarker,
} from './harness.mjs';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { spawnSync } from 'node:child_process';

// ---------------------------------------------------------------- scenarios

async function run() {
  const dataDir = mkdtempSync(join(tmpdir(), 'cashflux-e2e-'));
  const server = startServer(dataDir);
  await waitForServer();
  const browser = await chromium.launch();

  try {
    // ---- 1. New account creation + pairing -------------------------------
    console.log(`
[1] new account creation + pairing`);
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
    const stateA = await waitForHealthySync(A.page);
    check('activation signs the device in and reaches the server', stateA === 'synced' || stateA === 'syncing', `state=${stateA}`);

    // ---- 2. The sample dataset must never reach the server ---------------
    console.log(`
[2] sample data is never uploaded`);
    const afterActivate = await serverView(admin);
    check('an activated-but-unpersonalised device uploads nothing', afterActivate.neverSynced,
      `syncedData=${afterActivate.syncedData} syncedAgo=${afterActivate.syncedAgo}`);

    // ---- 3. Real data pairs and lands ------------------------------------
    console.log(`
[3] real data uploads`);
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
    console.log(`
[4] second browser hydrates`);
    const code2 = await mintCode(admin);
    const B = await newClient(browser);
    await openCloudTab(B.page);
    await enableSync(B.page);
    await activate(B.page, code2);
    await B.page.goto(`${BASE}/budget/`, { waitUntil: 'load', timeout: BOOT_MS });
    await waitForApp(B.page);

    // Signed-in is asserted by the indicator EXISTING (it renders nothing at all
    // until a session exists). The state VALUE is deliberately not asserted for this
    // client: several browser contexts from one address trip the bridge's per-client
    // WebSocket upgrade throttle, so a transient "error" here is the harness, not the
    // product — and the two checks below make the real claim anyway, that this
    // browser pulled the snapshot and can see a transaction typed on the other one.
    const stateB = await waitForHealthySync(B.page);
    check('the second browser is signed in', stateB !== null, `state=${stateB}`);
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
    console.log(`
[5] secondary login keeps data`);
    await openCloudTab(A.page);
    // Wait for it rather than sampling once: the pane re-renders as sync activity
    // lands, and the control appears with the signed-in state rather than with the
    // page. A one-shot count here was the suite's own flakiness, not the product's.
    const signOut = A.page.getByText(/^Sign out$/).first();
    await signOut.waitFor({ timeout: 30_000 }).catch(() => {});
    check('a signed-in device offers Sign out', (await signOut.count()) === 1);
    await signOut.click();
    await A.page.waitForTimeout(5000);
    check('signing out returns the activation field', (await A.page.getByTestId('device-link-code').count()) === 1);

    const code3 = await mintCode(admin);
    await openCloudTab(A.page);
    await activate(A.page, code3);
    const stateRelogin = await waitForHealthySync(A.page);
    check('re-activating signs back in', stateRelogin === 'synced' || stateRelogin === 'syncing', `state=${stateRelogin}`);

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

  console.log(`\n${failed() === 0 ? 'ALL SCENARIOS PASSED' : `${failed()} CHECK(S) FAILED`}`);
  process.exit(failed() === 0 ? 0 : 1);
}

run().catch((e) => {
  console.error('sync-flows aborted:', e);
  process.exit(2);
});
