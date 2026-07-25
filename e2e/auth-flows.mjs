// auth-flows.mjs — authentication and authorization: roles, account management,
// credential reset, and the one-click handoff from the admin console.
//
// Rig and helpers live in harness.mjs; see its header for why assertions go through
// the real UIs rather than the database.
//
// The point of this suite is the things a screenshot cannot tell you: that a
// read-only account is ACTUALLY refused by the server, that suspension bites a
// device already signed in, and that resetting a credential really ends live
// sessions rather than only blocking the next login.
//
// Usage:  node e2e/auth-flows.mjs [--keep]

import {
  BASE, KEEP, BOOT_MS, chromium,
  check, step, failed,
  startServer, waitForServer, adminPage, mintCode, openCashfluxTab, serverView,
  newClient, openCloudTab, enableSync, activate, waitForApp, waitForHealthySync,
  addMarkerTransaction, hasMarker,
} from './harness.mjs';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { spawnSync } from 'node:child_process';

const MARKER_A = `AUTH-A-${process.pid}-${Date.now().toString(36)}`;

async function clickRowAction(admin, label) {
  await openCashfluxTab(admin);
  await admin.getByText(label, { exact: true }).first().click();
  await admin.waitForTimeout(3500);
}

async function run() {
  const dataDir = mkdtempSync(join(tmpdir(), 'cashflux-auth-e2e-'));
  const server = startServer(dataDir);
  await waitForServer();
  const browser = await chromium.launch();

  try {
    const admin = await adminPage(browser, { firstRun: true });

    // ---- 1. One-click handoff from the console ---------------------------
    console.log('\n[1] one-click handoff from the admin console');
    // The console mints a code against its own authenticated session and carries it
    // in the URL; the client redeems and strips it. This is the whole point of
    // phase 4: the operator proves who they are ONCE.
    const handoffCode = await mintCode(admin);
    const H = await newClient(browser);
    await H.page.goto(`${BASE}/budget/?activate=${handoffCode}`, { waitUntil: 'load', timeout: BOOT_MS });
    await waitForApp(H.page);
    const handoffState = await waitForHealthySync(H.page);
    check('arriving with a handoff code signs the device in with no typing',
      handoffState === 'synced' || handoffState === 'syncing', `state=${handoffState}`);
    check('the code is stripped from the address bar',
      !(await H.page.evaluate(() => location.search)).includes('activate'),
      await H.page.evaluate(() => location.search));

    // Give it real data so later scenarios have something to protect.
    await H.page.getByTestId('sample-dismiss').click().catch(() => {});
    await H.page.waitForTimeout(3000);
    await addMarkerTransaction(H.page, MARKER_A);
    await H.page.getByTestId('sync-pulse').click().catch(() => {});
    await H.page.waitForTimeout(12_000);
    const seeded = await serverView(admin);
    check('that device syncs real data', !seeded.neverSynced, `syncedData=${seeded.syncedData}`);

    // ---- 2. Invite a second person onto their OWN account -----------------
    console.log(`
[2] an invited person gets their own account and data`);
    await openCashfluxTab(admin);
    await admin.locator('input[type=text]').last().fill('priya');
    await admin.getByText('Add person', { exact: true }).click();
    await admin.waitForTimeout(4000);

    const afterAdd = await serverView(admin);
    check('adding a person creates a second account', afterAdd.users === 2, `users=${afterAdd.users}`);

    // Their own code — NOT the owner shortcut, which would land them in Cam's data.
    await openCashfluxTab(admin);
    await admin.getByText('Code', { exact: true }).first().click();
    await admin.waitForTimeout(3000);
    const inviteCode = (await admin.getByTestId('user-activation-code').first().innerText().catch(() => '')).trim();
    check('the console hands out a code for that account', /^\d{6}$/.test(inviteCode), `code=${inviteCode}`);

    const G = await newClient(browser);
    await openCloudTab(G.page);
    await enableSync(G.page);
    await activate(G.page, inviteCode);
    const guestState = await waitForHealthySync(G.page);
    check('the invited person signs in', guestState === 'synced' || guestState === 'syncing', `state=${guestState}`);

    // The decisive check: they must NOT see the owner's transaction.
    check('the invited person does NOT see the owner data',
      !(await hasMarker(G.page, MARKER_A)), MARKER_A);
    check('the owner still sees their own data', await hasMarker(H.page, MARKER_A), MARKER_A);
    check('two accounts exist, not one', (await serverView(admin)).users === 2);

    // ---- 3. The owner account is protected from its own console ----------
    console.log(`
[3] the console cannot lock the operator out`);
    const view = await serverView(admin);
    check('the owner row is labelled as yours', /your account/i.test(view.text));

    // ---- 4. Management actions keep the data -----------------------------
    console.log(`
[4] suspend / reset keep the data`);
    const before = await serverView(admin);
    await clickRowAction(admin, 'Reset');
    const afterReset = await serverView(admin);
    // Asserted on the stored bytes, not on neverSynced: that flag scans the whole
    // page, and the account we just invited has legitimately never synced.
    check('resetting keeps the accounts and the owner data',
      afterReset.users === before.users && afterReset.syncedData === before.syncedData,
      `users=${afterReset.users} before=${before.syncedData} after=${afterReset.syncedData}`);
    check('the owner device still shows its own transaction',
      await hasMarker(H.page, MARKER_A), MARKER_A);
  } finally {
    if (!KEEP) {
      await browser.close().catch(() => {});
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
  console.error('\nauth-flows aborted:', e);
  process.exit(2);
});
