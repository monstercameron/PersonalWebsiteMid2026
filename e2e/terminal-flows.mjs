// terminal-flows.mjs — end-to-end checks for the interactive terminal.
//
// Everything here is driven through the real UI with real keystrokes: the point of most of these
// features is what happens when a person presses a key, and a test that calls the Go function
// directly would have passed for every one of the defects this suite now guards.
//
// Run against an already-running server (the dev server is fine):
//   bash scripts/build.sh && LISTEN_ADDR=127.0.0.1:8095 ./bin/server.exe &
//   node e2e/terminal-flows.mjs
// Override the target with E2E_BASE. Screenshots land in e2e/shots/ with --shots.

import { mkdirSync } from 'node:fs';
import { chromium } from 'file:///C:/Users/mreca/Desktop/CashFlux/node_modules/playwright/index.mjs';

const BASE = process.env.E2E_BASE || 'http://127.0.0.1:8095';
const SHOTS = process.argv.includes('--shots');
const SHOT_DIR = 'e2e/shots';

// Headless renders this page at roughly 0.4fps (DEVLOG 2026-07-29), so full-speed typing silently
// drops keystrokes. Every keystroke in this file goes through type() with a delay for that reason.
const TYPE_DELAY = 35;
const BOOT_MS = 120_000;

let failures = 0;
const check = (label, ok, detail = '') => {
  console.log(`   ${ok ? 'PASS' : 'FAIL'}  ${label}${detail ? ` — ${detail}` : ''}`);
  if (!ok) failures++;
};
const step = (m) => console.log(`\n== ${m}`);

/** boot opens a page and waits for the terminal to finish booting. */
async function boot(browser, { url = BASE, viewport = { width: 1280, height: 1600 } } = {}) {
  const page = await browser.newPage({ viewport });
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: BOOT_MS });
  await page.waitForSelector('#term-input', { timeout: BOOT_MS });
  await page.waitForFunction(
    () => document.querySelector('#term-body')?.textContent?.includes('session ready'),
    { timeout: BOOT_MS },
  );
  return page;
}

/**
 * run types a command and waits for the terminal to actually settle: the scrollback has grown and
 * the input has been cleared. A fixed sleep here was flaky — this page renders at roughly 0.4fps
 * headless, so "400ms" is somewhere between one frame and none, and a later keystroke could land
 * against a render that had not committed yet.
 */
async function run(page, cmd) {
  const before = await page.evaluate(() => document.querySelector('#term-body').childElementCount);
  await page.click('#term-body');
  await page.fill('#term-input', '');
  await page.type('#term-input', cmd, { delay: TYPE_DELAY });
  await page.keyboard.press('Enter');
  await page.waitForFunction(
    (n) => document.querySelector('#term-body').childElementCount > n,
    before, { timeout: 30_000 },
  ).catch(() => {});
  await settle(page);
}

/** settle waits for the input line to be empty and idle. */
async function settle(page) {
  await page.waitForFunction(
    () => document.querySelector('#term-input')?.value === '',
    null, { timeout: 15_000 },
  ).catch(() => {});
}

/**
 * pressKey sends a key and waits for the input to change, returning its new value. Reading the
 * value immediately after the press is the race that made the history checks flaky.
 */
async function pressKey(page, key) {
  const before = await page.inputValue('#term-input');
  await page.keyboard.press(key);
  await page.waitForFunction(
    (p) => document.querySelector('#term-input')?.value !== p,
    before, { timeout: 10_000 },
  ).catch(() => {});
  return page.inputValue('#term-input');
}

/** body returns the terminal's visible text. */
const body = (page) => page.textContent('#term-body');

async function main() {
  if (SHOTS) mkdirSync(SHOT_DIR, { recursive: true });
  const browser = await chromium.launch();

  // ---------------------------------------------------------------- boot + honest metrics
  step('boot and measured metrics');
  let page = await boot(browser);
  let text = await body(page);
  check('boot log rendered', text.includes('session ready'));
  check('neofetch splash rendered', text.includes('earl cameron'));

  // The defect: these two lines used to be hardcoded strings ("wasm · 4.2 mb", "14 ms").
  check('wasm size is measured, not invented', /wasm\s+[\d.]+\s*(KB|MB|B)/.test(text), text.match(/wasm[^\n·]*/)?.[0]);
  check('hydration time reported', /hydrated in \d+ ms/.test(text));
  await page.waitForFunction(
    () => /round trip|unreachable/.test(document.querySelector('#term-body')?.textContent || ''),
    { timeout: 30_000 },
  ).catch(() => {});
  text = await body(page);
  check('tunnel latency measured over a real round trip', /\d+ ms round trip/.test(text),
    text.match(/[\w-]* ?\d+ ms round trip/)?.[0]);

  // ---------------------------------------------------------------- a11y floor
  step('accessibility floor');
  check('scrollback is a live log region',
    (await page.getAttribute('#term-body', 'role')) === 'log' &&
    (await page.getAttribute('#term-body', 'aria-live')) === 'polite');
  check('input has an accessible name', !!(await page.getAttribute('#term-input', 'aria-label')));
  check('the page does not steal focus on load',
    await page.evaluate(() => document.activeElement === document.body || document.activeElement === null || document.activeElement?.id === 'term-input'));
  check('traffic lights are real buttons',
    (await page.evaluate(() => document.querySelector('#term-expand')?.tagName)) === 'BUTTON');
  check('traffic lights are labelled', !!(await page.getAttribute('#term-expand', 'aria-label')));

  // ---------------------------------------------------------------- history + readline
  step('history recall and line editing');
  await run(page, 'help');
  await run(page, 'about');
  await page.click('#term-input');
  let v = await pressKey(page, 'ArrowUp');
  check('ArrowUp recalls the last command', v === 'about', v);
  v = await pressKey(page, 'ArrowUp');
  check('ArrowUp walks further back', v === 'help', v);
  v = await pressKey(page, 'ArrowDown');
  check('ArrowDown walks forward', v === 'about', v);
  v = await pressKey(page, 'ArrowDown');
  check('ArrowDown past the newest restores the empty draft', v === '', v);

  await page.type('#term-input', 'open cashflux', { delay: TYPE_DELAY });
  v = await pressKey(page, 'Alt+Backspace');
  check('Alt+Backspace kills the previous word', v === 'open ', JSON.stringify(v));
  v = await pressKey(page, 'Control+u');
  check('Ctrl+U kills to the start of the line', v === '', JSON.stringify(v));

  await page.type('#term-input', 'garbage', { delay: TYPE_DELAY });
  v = await pressKey(page, 'Control+c');
  check('Ctrl+C abandons the line', v === '', JSON.stringify(v));
  check('Ctrl+C echoes ^C', (await body(page)).includes('^C'));

  await page.keyboard.press('Control+l');
  await page.waitForTimeout(300);
  check('Ctrl+L clears the screen', !(await body(page)).includes('session ready'));

  // history must include portfolio programs, which never reach the shell
  await run(page, 'history');
  text = await body(page);
  check('history records portfolio programs', text.includes('help') && text.includes('about'));

  // ---------------------------------------------------------------- discovery
  step('discovery: chips, tab completion, did-you-mean');
  await page.click('button[data-cmd="projects"]');
  await page.waitForTimeout(600);
  check('a chip runs its command', (await body(page)).includes('CashFlux'));

  await page.click('#term-input');
  await page.type('#term-input', 'neofet', { delay: TYPE_DELAY });
  await page.keyboard.press('Tab');
  check('Tab completes a command', (await page.inputValue('#term-input')).trim() === 'neofetch');
  await page.fill('#term-input', '');

  await run(page, 'projets');
  check('a typo becomes a suggestion', (await body(page)).includes('did you mean `projects`'));

  // ---------------------------------------------------------------- pipes into programs
  step('portfolio programs in pipelines');
  await run(page, 'projects | grep GoWebComponents');
  text = await body(page);
  check('a portfolio program pipes into grep', text.includes('GoWebComponents'));

  // ---------------------------------------------------------------- filesystem recovery
  step('filesystem is recoverable');
  await run(page, 'rm -r notes');
  await run(page, 'ls');
  check('notes can be deleted in-session', !(await body(page)).includes('notes/'));
  await run(page, 'reset');
  await run(page, 'ls');
  check('reset restores the recruiter notes', (await body(page)).includes('notes/'));

  // ---------------------------------------------------------------- theme
  step('theme switching');
  const bgBefore = await page.evaluate(() =>
    getComputedStyle(document.documentElement).getPropertyValue('--t-bg'));
  await run(page, 'theme paper');
  const bgAfter = await page.evaluate(() =>
    getComputedStyle(document.documentElement).getPropertyValue('--t-bg'));
  check('theme changes the palette', bgBefore !== bgAfter, `${bgBefore.trim()} -> ${bgAfter.trim()}`);
  await run(page, 'theme aubergine');

  // ---------------------------------------------------------------- live server facts
  step('stats come off the server');
  await run(page, 'stats');
  await page.waitForFunction(
    () => /uptime|tunnel unavailable/.test(document.querySelector('#term-body')?.textContent || ''),
    { timeout: 30_000 },
  ).catch(() => {});
  text = await body(page);
  check('stats reports a build and uptime', /build/.test(text) && /uptime/.test(text));
  check('stats reports requests served', /requests/.test(text));

  // ---------------------------------------------------------------- bench
  step('bench measures the runtime');
  await run(page, 'bench');
  await page.waitForFunction(
    () => /string build/.test(document.querySelector('#term-body')?.textContent || ''),
    { timeout: 60_000 },
  ).catch(() => {});
  text = await body(page);
  check('bench reports rates', /ops\/sec/.test(text));
  check('bench ran every case', /integer math/.test(text) && /wasm↔js call/.test(text));

  // ---------------------------------------------------------------- contact over gRPC
  step('contact sends over the tunnel');
  await run(page, 'contact');
  check('contact asks for a name', (await body(page)).includes('your name') ||
    (await page.textContent('#term-body')).includes('three questions'));
  await run(page, 'E2E Tester');
  await run(page, 'not-an-email');
  check('a bad email is rejected before it is sent', (await body(page)).includes("doesn't look like an email"));
  await run(page, 'e2e@example.com');
  await run(page, 'automated end-to-end check, please ignore');
  await page.waitForFunction(
    () => /Thanks|couldn't|too many|tunnel/.test(document.querySelector('#term-body')?.textContent || ''),
    { timeout: 30_000 },
  ).catch(() => {});
  text = await body(page);
  check('the message was accepted by the server', /Thanks/.test(text),
    text.split('\n').slice(-3).join(' ').slice(0, 120));

  // ---------------------------------------------------------------- share
  step('share produces a working deep link');
  await run(page, 'about');
  await run(page, 'share');
  text = await body(page);
  const shared = text.match(/https?:\/\/[^\s]*\?cmd=[^\s]+/)?.[0];
  check('share prints a deep link', !!shared, shared);
  check('the link carries the last command', !!shared && shared.includes('cmd=about'));
  await run(page, 'share rm -rf ~');
  check('share refuses a command a link cannot run',
    (await body(page)).includes('will not autorun'));

  // ---------------------------------------------------------------- resume opens the document
  step('resume opens the résumé');
  const popupPromise = page.waitForEvent('popup', { timeout: 15_000 }).catch(() => null);
  await run(page, 'resume');
  const popup = await popupPromise;
  check('resume opens /resume', !!popup && new URL(popup.url()).pathname === '/resume',
    popup ? popup.url() : 'no popup');
  if (popup) await popup.close();

  // ---------------------------------------------------------------- man + hidden commands
  step('man pages and the hidden commands');
  await run(page, 'man ls');
  check('man still works', (await body(page)).includes('list directory contents'));

  await run(page, 'cowsay hello recruiters');
  text = await body(page);
  check('cowsay draws the cow', text.includes('^__^') && text.includes('hello recruiters'));

  await run(page, 'sudo rm -rf /');
  check('sudo has an answer', (await body(page)).includes('not in the sudoers file'));

  await run(page, 'fortune');
  check('fortune says something', (await body(page)).length > 0);

  await run(page, 'htop');
  check('htop lists the real processes', (await body(page)).includes('grpctunnel'));

  // sl animates, then settles on the finished train
  await run(page, 'sl');
  await page.waitForTimeout(2500);
  check('sl runs and finishes', (await body(page)).includes('===='));

  // ---------------------------------------------------------------- second-tier coreutils
  // `uname` is here because its absence is what prompted them: a visitor typed it, the shell did
  // not have it, and the did-you-mean suggested `anime`.
  step('coreutils');
  await run(page, 'uname -a');
  text = await body(page);
  check('uname exists and says what this really is', /Wasm .*js\/wasm/.test(text));
  check('uname does not title-case GOOS into "Js"', !text.includes('Js portfolio'));

  await run(page, 'cal');
  text = await body(page);
  check('cal renders a month grid', text.includes('Su Mo Tu We Th Fr Sa'));
  check('cal names today', /today: \w+ \d+ \w+/.test(text));

  await run(page, 'free');
  check('free reports the real Go heap', (await body(page)).includes('Go heap'));

  await run(page, 'tree notes');
  text = await body(page);
  check('tree walks the filesystem', text.includes('experience.md') && text.includes('files'));

  await run(page, 'echo terminal | base64 | base64 -d');
  check('base64 round-trips through a pipeline', (await body(page)).includes('terminal'));

  await run(page, 'factor 84');
  check('factor factors', (await body(page)).includes('84: 2 2 3 7'));

  await run(page, 'seq 100000');
  check('seq is bounded so it cannot lock the tab', (await body(page)).includes('stopped at'));

  await run(page, 'sleep 5');
  check('sleep refuses to block the event loop', (await body(page)).includes('not blocking the event loop'));

  // ping does real round trips over the tunnel
  await run(page, 'ping');
  await page.waitForFunction(
    () => /rtt min\/avg\/max/.test(document.querySelector('#term-body')?.textContent || ''),
    { timeout: 30_000 },
  ).catch(() => {});
  text = await body(page);
  check('ping measures real round trips', /rtt min\/avg\/max = \d+\/\d+\/\d+ ms/.test(text),
    text.match(/rtt min\/avg\/max[^\n]*/)?.[0]);

  // ---------------------------------------------------------------- snake
  step('snake');
  await page.click('#term-body');
  await page.fill('#term-input', '');
  await page.type('#term-input', 'snake', { delay: TYPE_DELAY });
  await page.keyboard.press('Enter');
  await page.waitForFunction(
    () => /score \d+/.test(document.querySelector('#term-body')?.textContent || ''),
    { timeout: 20_000 },
  ).catch(() => {});
  check('snake renders a board', /score \d+/.test(await body(page)));
  await page.keyboard.press('q');
  await page.waitForTimeout(800);
  check('q quits the game', (await body(page)).includes('quit — score'));
  // The game installs a document key listener; if it leaks, the arrow keys stop reaching history.
  await page.click('#term-input');
  await page.fill('#term-input', '');
  const afterGame = await pressKey(page, 'ArrowUp');
  check('arrow keys return to history after the game', afterGame === 'snake', afterGame);
  await page.fill('#term-input', '');

  // ---------------------------------------------------------------- tour
  step('guided tour');
  await page.click('#term-body');
  await page.fill('#term-input', '');
  await page.type('#term-input', 'tour', { delay: TYPE_DELAY });
  await page.keyboard.press('Enter');
  // The tour types for the visitor: the input fills without anyone touching the keyboard.
  const typedItself = await page.waitForFunction(
    () => (document.querySelector('#term-input')?.value || '').length > 0,
    { timeout: 30_000 },
  ).then(() => true).catch(() => false);
  check('the tour types on its own', typedItself);
  // ...and hands control back the moment the visitor intervenes.
  await page.keyboard.press('Control+c');
  await page.waitForTimeout(2500);
  const afterStop = await body(page);
  await page.waitForTimeout(2500);
  check('the tour stops when the visitor takes over',
    (await body(page)).length === afterStop.length);

  // ---------------------------------------------------------------- fullscreen + focus
  step('fullscreen modal');
  await page.click('#term-expand');
  await page.waitForTimeout(500);
  check('expands to a modal dialog', (await page.getAttribute('#term-frame', 'role')) === 'dialog');
  check('modal is announced as modal', (await page.getAttribute('#term-frame', 'aria-modal')) === 'true');
  if (SHOTS) await page.screenshot({ path: `${SHOT_DIR}/desktop-expanded.png`, fullPage: false });
  await page.keyboard.press('Escape');
  await page.waitForTimeout(500);
  check('Escape shrinks it', (await page.getAttribute('#term-frame', 'role')) !== 'dialog');
  check('focus returns to the launch control',
    await page.evaluate(() => document.activeElement?.id === 'term-launch'));

  if (SHOTS) {
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({ path: `${SHOT_DIR}/desktop-full.png`, fullPage: true });
  }
  await page.close();

  // ---------------------------------------------------------------- deep links
  step('deep links');
  page = await boot(browser, { url: `${BASE}/?cmd=open%20cashflux` });
  await page.waitForFunction(
    () => /Local-first budgeting/.test(document.querySelector('#term-body')?.textContent || ''),
    { timeout: 30_000 },
  ).catch(() => {});
  text = await body(page);
  check('a deep link runs its command', text.includes('Local-first budgeting'));
  check('a deep link opens the terminal', (await page.getAttribute('#term-frame', 'role')) === 'dialog');
  await page.close();

  page = await boot(browser, { url: `${BASE}/?cmd=rm%20-rf%20~` });
  await page.waitForTimeout(1500);
  text = await body(page);
  check('a destructive deep link is refused', text.includes('only read-only commands run automatically'));
  check('the refused command did not run', !text.includes('No such file'));
  await page.close();

  // ---------------------------------------------------------------- mobile
  step('mobile');
  page = await boot(browser, { viewport: { width: 390, height: 844 } });
  check('command chips are reachable without a keyboard',
    await page.isVisible('button[data-cmd="tour"]'));
  const overflows = await page.evaluate(() =>
    document.documentElement.scrollWidth > document.documentElement.clientWidth + 1);
  check('page does not scroll sideways', !overflows);
  await page.click('button[data-cmd="help"]');
  await page.waitForTimeout(800);
  check('a chip works on mobile', (await body(page)).includes('available programs'));
  if (SHOTS) await page.screenshot({ path: `${SHOT_DIR}/mobile-full.png`, fullPage: true });
  await page.click('#term-expand');
  await page.waitForTimeout(600);
  if (SHOTS) await page.screenshot({ path: `${SHOT_DIR}/mobile-expanded.png` });
  await page.close();

  // ---------------------------------------------------------------- SSR fallback
  step('server-rendered fallback (no wasm)');
  const noJs = await browser.newContext({ javaScriptEnabled: false });
  const plain = await noJs.newPage();
  await plain.goto(BASE, { waitUntil: 'domcontentloaded' });
  const ssr = await plain.textContent('body');
  check('the recruiter notes are server-rendered', ssr.includes('EXPERIENCE') && ssr.includes('SKILLS'));
  check('the briefing names the employer', ssr.includes('UKG'));
  check('the terminal is absent without wasm, and the page still works',
    !ssr.includes('session ready') && ssr.includes('Senior software engineer'));

  // The briefing is collapsed by default, and the disclosure must work with no JavaScript at all —
  // it is the fallback for visitors whose wasm never runs, so it cannot depend on scripting.
  check('the briefing is collapsed by default',
    (await plain.getAttribute('#briefing details', 'open')) === null);
  check('its heading survives collapsing (document outline, crawlers)',
    (await plain.isVisible('#briefing summary h2')));
  check('the content is hidden while collapsed',
    !(await plain.isVisible('#briefing pre')));
  await plain.click('#briefing summary');
  await plain.waitForTimeout(400);
  check('it opens without JavaScript', await plain.isVisible('#briefing pre'));
  check('the experience document is readable once open',
    (await plain.innerText('#briefing')).includes('UKG — Software Engineer'));
  await noJs.close();

  await browser.close();
  console.log(`\n${failures === 0 ? 'ALL PASS' : `${failures} FAILURE(S)`}`);
  process.exit(failures === 0 ? 0 : 1);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
