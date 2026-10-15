#!/usr/bin/env node
'use strict';
/*
 * The page and the API it calls, in real headless Chrome.
 *
 *   node test/browser/run.js              build and check this checkout
 *   node test/browser/run.js --root DIR   build and check another checkout,
 *                                         e.g. a worktree of the first commit
 *
 * Needs Go (`go` on PATH, or GO=path/to/go) and Chrome (found in the usual
 * places, or CHROME=path/to/chrome). No npm dependencies.
 *
 * Checks named "regression:" fail against the first commit.
 */
const { spawn, spawnSync } = require('node:child_process');
const { mkdtempSync, readdirSync, readFileSync, statSync } = require('node:fs');
const net = require('node:net');
const os = require('node:os');
const path = require('node:path');
const { Browser, findChrome, removeDir, sleep } = require('./cdp');

const args = process.argv.slice(2);
const rootArg = args.indexOf('--root');
const root = path.resolve(rootArg >= 0 ? args[rootArg + 1] : path.join(__dirname, '..', '..'));
const GO = process.env.GO || 'go';

/* --- the server ------------------------------------------------------- */

function freePort() {
  return new Promise((resolve, reject) => {
    const srv = net.createServer();
    srv.on('error', reject);
    srv.listen(0, '127.0.0.1', () => {
      const { port } = srv.address();
      srv.close(() => resolve(port));
    });
  });
}

async function startServer(work) {
  const exe = path.join(work, process.platform === 'win32' ? 'sugen.exe' : 'sugen');
  const build = spawnSync(GO, ['build', '-o', exe, './cmd/server'], { cwd: root, encoding: 'utf8' });
  if (build.status !== 0) throw new Error(`go build failed in ${root}:\n${build.stderr || build.error}`);
  const port = await freePort();
  const proc = spawn(exe, [], { env: { ...process.env, PORT: String(port) }, stdio: 'ignore' });
  const origin = `http://localhost:${port}`;
  for (let k = 0; k < 100; k++) {
    try {
      if ((await fetch(origin + '/')).ok) return { proc, origin };
    } catch {
      /* not listening yet */
    }
    await sleep(100);
  }
  proc.kill();
  throw new Error('server did not start');
}

/* --- page helpers ----------------------------------------------------- */

/*
 * Installed before the page's own scripts.
 *
 * Responses to requests whose URL contains window.__delay.match are held
 * back window.__delay.ms, so a check can make an earlier request answer
 * after a later one.
 *
 * Headless Chrome has no clipboard to read back, so writeText records what
 * the page hands it in window.__clipboard. That is the part the page
 * decides; what the OS does with it is not this project's code.
 */
const PAGE_HOOKS = `
  window.__delay = null;
  const realFetch = window.fetch;
  window.fetch = async (...a) => {
    const d = window.__delay;
    const res = await realFetch(...a);
    if (d && String(a[0]).includes(d.match)) await new Promise((r) => setTimeout(r, d.ms));
    return res;
  };
  window.__clipboard = null;
  Object.defineProperty(navigator, 'clipboard', {
    value: { writeText: async (t) => { window.__clipboard = String(t); } },
  });
`;

async function openPage(browser, origin) {
  const page = browser.page;
  await page.send('Page.addScriptToEvaluateOnNewDocument', { source: PAGE_HOOKS });
  await page.navigate(origin + '/');
  await loaded(page);
  return page;
}

/* Both boxes have left "Loading...". */
const loaded = (page) => page.waitFor(`
  document.getElementById('password').textContent !== 'Loading...' &&
  document.getElementById('passphrase').textContent !== 'Loading...'`);

const text = (page, id) => page.eval(`document.getElementById(${JSON.stringify(id)}).textContent`);

/* Set an input the way a person does: the value, then the events it fires. */
const setValue = (page, id, value) => page.eval(`(() => {
  const el = document.getElementById(${JSON.stringify(id)});
  el.value = ${JSON.stringify(String(value))};
  el.dispatchEvent(new Event('input', { bubbles: true }));
  el.dispatchEvent(new Event('change', { bubbles: true }));
})()`);

const click = (page, id) => page.eval(`document.getElementById(${JSON.stringify(id)}).click()`);

/* Wait until a box satisfies pred; on timeout, say what it held instead. */
async function expectBox(page, id, pred, what, timeoutMs = 5000) {
  const end = Date.now() + timeoutMs;
  let last;
  while (Date.now() < end) {
    last = await text(page, id);
    if (pred(last)) return last;
    await sleep(50);
  }
  throw new Error(`#${id}: expected ${what}, got ${JSON.stringify(last)}`);
}

/* The notification the page shows after Copy; the newest one wins. */
const lastNotification = (page) => page.waitFor(`(() => {
  const n = document.querySelectorAll('.notification');
  return n.length ? n[n.length - 1].textContent : '';
})()`);

const PASSWORD_CHARS = /^[a-zA-Z0-9~!@#$%^&\-=_+?]+$/;
const TOO_SHORT = 'password length too short for the specified minimum requirements';

/* --- checks ----------------------------------------------------------- */

const checks = [];
const check = (name, fn, opts = {}) => checks.push({ name, fn, opts });

check('the page shows a 12-character password and a two-word passphrase', async (page) => {
  await expectBox(page, 'password', (t) => t.length === 12 && PASSWORD_CHARS.test(t), 'a 12-character password');
  await expectBox(page, 'passphrase', (t) => t.split(' ').length === 2, 'two words');
});

check('Copy puts the shown password on the clipboard', async (page) => {
  const pw = await expectBox(page, 'password', (t) => PASSWORD_CHARS.test(t), 'a password');
  await click(page, 'copy_password');
  const note = await lastNotification(page);
  if (note !== 'Password copied to clipboard!') throw new Error(`notification: ${JSON.stringify(note)}`);
  const clip = await page.eval('window.__clipboard');
  if (clip !== pw) throw new Error(`clipboard got ${JSON.stringify(clip)}, box shows ${JSON.stringify(pw)}`);
});

check('regression: settings with no room show the reason, not an empty box', async (page) => {
  // 8 characters, at least 3 digits and 5 special characters: no room for
  // the two letters. The original server returned 10 characters here. Once
  // it refused, the page set the box to result.password, which is
  // undefined, and the box went blank.
  await setValue(page, 'password_range', 8);
  await setValue(page, 'minimum_digits_in_password', 3);
  await setValue(page, 'minimum_special_characters_in_password', 5);
  await expectBox(page, 'password', (t) => t === TOO_SHORT, 'the server\'s reason');
});

check('regression: Copy refuses while the box shows a reason', async (page) => {
  await setValue(page, 'password_range', 8);
  await setValue(page, 'minimum_digits_in_password', 3);
  await setValue(page, 'minimum_special_characters_in_password', 5);
  await expectBox(page, 'password', (t) => !PASSWORD_CHARS.test(t), 'no password');
  await click(page, 'copy_password');
  const note = await lastNotification(page);
  const clip = await page.eval('window.__clipboard');
  if (note !== 'Failed to copy password.' || clip !== null) {
    throw new Error(`notification ${JSON.stringify(note)}, clipboard got ${JSON.stringify(clip)}`);
  }
});

check('regression: a switched-off minimum does not count against the length', async (page) => {
  // Unticking Numbers disables "Minimum digits" (3) but the page still sends
  // it. 8 characters with 6 special characters fit once it is ignored.
  await setValue(page, 'password_range', 8);
  await click(page, 'numbers_in_password');
  await setValue(page, 'minimum_special_characters_in_password', 6);
  // Earlier answers (Numbers still on) also have 8 characters; wait for
  // one that matches the final settings.
  await expectBox(page, 'password', (t) => t.length === 8 && PASSWORD_CHARS.test(t) &&
    !/[0-9]/.test(t) && t.replace(/[a-zA-Z]/g, '').length >= 6,
  '8 characters, no digits, at least 6 special');
});

check('regression: Regenerate leaves the length label telling the truth', async (page) => {
  await setValue(page, 'password_range', 20);
  await expectBox(page, 'password', (t) => t.length === 20, 'a 20-character password');
  const before = await text(page, 'password');
  await click(page, 'reset_button');
  await expectBox(page, 'password', (t) => t !== before, 'a new password');
  const label = await text(page, 'password_range_label');
  const pw = await text(page, 'password');
  if (label !== 'Length: 20' || pw.length !== 20) {
    throw new Error(`label ${JSON.stringify(label)} over a ${pw.length}-character password`);
  }
});

check('regression: a slow earlier password response cannot replace a newer one', async (page) => {
  await page.eval(`window.__delay = { match: 'passwordLength=10&', ms: 800 }`);
  await setValue(page, 'password_range', 10);
  await sleep(50);
  await setValue(page, 'password_range', 16);
  await expectBox(page, 'password', (t) => t.length === 16, 'a 16-character password');
  await sleep(1200); // the held-back answer for 10 arrives now
  const pw = await text(page, 'password');
  if (pw.length !== 16) throw new Error(`slider at 16, box shows ${pw.length} characters: ${pw}`);
});

check('regression: a slow earlier passphrase response cannot replace a newer one', async (page) => {
  await page.eval(`window.__delay = { match: 'totalWords=3', ms: 800 }`);
  await setValue(page, 'passphrase_phrase_range', 3);
  await sleep(50);
  await setValue(page, 'passphrase_phrase_range', 4);
  await expectBox(page, 'passphrase', (t) => t.split(' ').length === 4, 'four words');
  await sleep(1200);
  const phrase = await text(page, 'passphrase');
  if (phrase.split(' ').length !== 4) throw new Error(`slider at 4, box shows ${JSON.stringify(phrase)}`);
});

check('regression: the page reads the form only once Chrome has restored it', async (page) => {
  // Back/forward without the bfcache reloads the page, and Chrome restores
  // the boxes' ticks and the slider after load, firing no change events.
  // The page read the form at DOMContentLoaded, before that: the minimum
  // inputs stayed enabled under unticked boxes (and, since the page flipped
  // them on each change, were backwards from then on), the label kept the
  // default length, and the first password was made from the defaults.
  await setValue(page, 'password_range', 20);
  await click(page, 'numbers_in_password');
  await click(page, 'special_characters_in_password');
  await expectBox(page, 'password', (t) => /^[a-zA-Z]{20}$/.test(t), '20 letters');
  await page.navigate('about:blank');
  await page.eval('history.back()');
  await page.once('Page.loadEventFired');
  await loaded(page);
  await sleep(300);
  const state = () => page.eval(`[
    document.getElementById('numbers_in_password').checked,
    document.getElementById('minimum_digits_in_password').disabled,
    document.getElementById('special_characters_in_password').checked,
    document.getElementById('minimum_special_characters_in_password').disabled,
    document.getElementById('password_range').value]`);
  const restored = await state();
  if (restored[0] || restored[2] || restored[4] !== '20') {
    throw new Error(`precondition: Chrome did not restore the form (${restored})`);
  }
  if (!restored[1] || !restored[3]) throw new Error(`boxes unticked, but minimum inputs disabled = ${restored[1]}, ${restored[3]}`);
  const label = await text(page, 'password_range_label');
  const pw = await text(page, 'password');
  if (label !== 'Length: 20' || !/^[a-zA-Z]{20}$/.test(pw)) {
    throw new Error(`restored to 20, letters only; label ${JSON.stringify(label)}, password ${JSON.stringify(pw)}`);
  }
  await click(page, 'numbers_in_password');
  await click(page, 'special_characters_in_password');
  const ticked = await state();
  if (ticked[1] || ticked[3]) throw new Error(`boxes ticked, but minimum inputs disabled = ${ticked[1]}, ${ticked[3]}`);
}, { extraArgs: ['--disable-features=BackForwardCache'] });

check('a page restored whole from the back/forward cache keeps what it showed', async (page) => {
  const before = [await text(page, 'password'), await text(page, 'passphrase')];
  await page.navigate('about:blank');
  await page.eval('history.back()');
  await page.waitFor(`location.pathname === '/' && document.readyState === 'complete'`);
  const persisted = await page.eval(`performance.getEntriesByType('navigation')[0].type`);
  await sleep(300);
  const after = [await text(page, 'password'), await text(page, 'passphrase')];
  if (after[0] !== before[0] || after[1] !== before[1]) {
    throw new Error(`navigation ${persisted}: ${JSON.stringify(before)} became ${JSON.stringify(after)}`);
  }
});

check('regression: no generated secret is left in Chrome\'s disk cache', async (page, browser) => {
  const shown = [await text(page, 'password'), await text(page, 'passphrase')];
  await setValue(page, 'password_range', 24);
  shown.push(await expectBox(page, 'password', (t) => t.length === 24, 'a 24-character password'));
  await browser.close(); // flushes the cache to disk
  const files = walk(browser.profile).filter((f) => /cache/i.test(f));
  const blob = Buffer.concat(files.map((f) => readFileSync(f))).toString('latin1');
  // Positive control: the page's own script is cached, so the cache is on
  // disk and searchable, and an absence below means something.
  if (!blob.includes('async function updatePassword')) throw new Error('control: password.js is not in the disk cache');
  const json = (s) => JSON.stringify(s).slice(1, -1).replace(/&/g, '\\u0026');
  const found = shown.filter((s) => blob.includes(s) || blob.includes(json(s)));
  if (found.length) throw new Error(`found in the disk cache: ${found.map((s) => JSON.stringify(s)).join(', ')}`);
}, { keepProfile: true });

function walk(dir) {
  const out = [];
  for (const name of readdirSync(dir)) {
    const p = path.join(dir, name);
    let s;
    try {
      s = statSync(p);
    } catch {
      continue;
    }
    if (s.isDirectory()) out.push(...walk(p));
    else if (s.size < 50 * 1024 * 1024) out.push(p);
  }
  return out;
}

/* --- runner ----------------------------------------------------------- */

(async () => {
  if (!findChrome()) {
    console.log('Chrome not found; set CHROME to its path.');
    process.exit(1);
  }
  const work = mkdtempSync(path.join(os.tmpdir(), 'sugen-browser-'));
  const { proc, origin } = await startServer(work);
  console.log(`# ${root}`);
  let failed = 0;
  try {
    for (const { name, fn, opts } of checks) {
      const browser = await Browser.launch({ extraArgs: opts.extraArgs, keepProfile: opts.keepProfile });
      try {
        const page = await openPage(browser, origin);
        await fn(page, browser);
        console.log(`ok - ${name}`);
      } catch (err) {
        failed++;
        console.log(`not ok - ${name}\n  # ${String(err.message).split('\n')[0]}`);
      } finally {
        await browser.close();
        if (opts.keepProfile) await removeDir(browser.profile);
      }
    }
  } finally {
    proc.kill();
    await sleep(300);
    await removeDir(work);
  }
  console.log(`\n${checks.length - failed} of ${checks.length} passed`);
  process.exit(failed ? 1 : 0);
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
