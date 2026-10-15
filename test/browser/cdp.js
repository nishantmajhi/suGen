'use strict';
/*
 * Headless Chrome over the DevTools protocol, on a pipe rather than a port,
 * with no dependencies. Messages on the pipe are JSON, each terminated by a
 * NUL byte; Chrome reads from our fd 3 and writes to our fd 4.
 *
 * Trimmed from `Browser Extensions/PDF Page Capture/tests/browser/cdp.js`
 * elsewhere in this archive.
 */
const { spawn } = require('node:child_process');
const { existsSync, mkdtempSync, rmSync } = require('node:fs');
const path = require('node:path');
const os = require('node:os');

const CHROME_CANDIDATES = [
  process.env.CHROME,
  'C:/Program Files/Google/Chrome/Application/chrome.exe',
  'C:/Program Files (x86)/Google/Chrome/Application/chrome.exe',
  process.env.LOCALAPPDATA && path.join(process.env.LOCALAPPDATA, 'Google/Chrome/Application/chrome.exe'),
  '/usr/bin/google-chrome',
  '/usr/bin/chromium',
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
].filter(Boolean);

function findChrome() {
  return CHROME_CANDIDATES.find((c) => existsSync(c)) || null;
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

class Browser {
  /*
   * The profile is a fresh temporary directory, deleted on close unless
   * keepProfile is set (the disk-cache check reads it afterwards).
   */
  static async launch({ profileRoot = os.tmpdir(), extraArgs = [], keepProfile = false } = {}) {
    const chrome = findChrome();
    if (!chrome) return null;
    const profile = mkdtempSync(path.join(profileRoot, 'sugen-profile-'));
    const proc = spawn(chrome, [
      '--headless=new',
      '--remote-debugging-pipe',
      '--no-first-run',
      '--no-default-browser-check',
      `--user-data-dir=${profile}`,
      ...extraArgs,
      'about:blank',
    ], { stdio: ['ignore', 'ignore', 'pipe', 'pipe', 'pipe'] });
    proc.stderr.resume();
    const browser = new Browser(proc, profile, keepProfile);
    const [page] = await browser.targets('page');
    browser.page = await browser.attach(page.targetId);
    await browser.page.send('Page.enable');
    await browser.page.send('Runtime.enable');
    return browser;
  }

  constructor(proc, profile, keepProfile) {
    this.proc = proc;
    this.profile = profile;
    this.keepProfile = keepProfile;
    this.nextId = 1;
    this.pending = new Map();
    this.listeners = new Set();
    this.exited = new Promise((r) => proc.on('exit', r));
    let buf = Buffer.alloc(0);
    proc.stdio[4].on('data', (chunk) => {
      buf = Buffer.concat([buf, chunk]);
      let i;
      while ((i = buf.indexOf(0)) !== -1) {
        const msg = JSON.parse(buf.subarray(0, i).toString('utf8'));
        buf = buf.subarray(i + 1);
        if (msg.id && this.pending.has(msg.id)) {
          const { resolve, reject, method } = this.pending.get(msg.id);
          this.pending.delete(msg.id);
          if (msg.error) reject(new Error(`${method}: ${msg.error.message}`));
          else resolve(msg.result);
        } else {
          for (const l of this.listeners) l(msg);
        }
      }
    });
  }

  send(method, params = {}, sessionId) {
    const id = this.nextId++;
    const msg = { id, method, params };
    if (sessionId) msg.sessionId = sessionId;
    this.proc.stdio[3].write(JSON.stringify(msg) + '\0');
    return new Promise((resolve, reject) => this.pending.set(id, { resolve, reject, method }));
  }

  on(fn) {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  async targets(type) {
    const { targetInfos } = await this.send('Target.getTargets');
    return type ? targetInfos.filter((t) => t.type === type) : targetInfos;
  }

  async attach(targetId) {
    const { sessionId } = await this.send('Target.attachToTarget', { targetId, flatten: true });
    return new Session(this, sessionId);
  }

  async close() {
    if (this.closed) return;
    this.closed = true;
    try {
      await Promise.race([this.send('Browser.close'), sleep(2000)]);
    } catch {
      /* already gone */
    }
    await Promise.race([this.exited, sleep(5000)]);
    if (this.proc.exitCode === null) this.proc.kill();
    if (!this.keepProfile) await removeDir(this.profile);
  }
}

/* Chrome's child processes can hold files for a moment after exit. */
async function removeDir(dir) {
  for (let k = 0; k < 40; k++) {
    try {
      rmSync(dir, { recursive: true, force: true });
      return;
    } catch {
      await sleep(250);
    }
  }
}

class Session {
  constructor(browser, id) {
    this.browser = browser;
    this.id = id;
  }

  send(method, params) {
    return this.browser.send(method, params, this.id);
  }

  /* Wait for the next event of this name on this session. */
  once(method, timeoutMs = 15000) {
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => { off(); reject(new Error(`no ${method}`)); }, timeoutMs);
      const off = this.browser.on((msg) => {
        if (msg.sessionId === this.id && msg.method === method) {
          clearTimeout(timer);
          off();
          resolve(msg.params);
        }
      });
    });
  }

  async navigate(url) {
    const loaded = this.once('Page.loadEventFired');
    await this.send('Page.navigate', { url });
    await loaded;
  }

  /* Evaluate an expression and return its value; rejections become errors. */
  async eval(expression) {
    const r = await this.send('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true });
    if (r.exceptionDetails) {
      const d = r.exceptionDetails;
      throw new Error((d.exception && d.exception.description) || d.text);
    }
    return r.result.value;
  }

  /* Poll an expression until it is truthy. */
  async waitFor(expression, timeoutMs = 10000, interval = 50) {
    const end = Date.now() + timeoutMs;
    let last;
    while (Date.now() < end) {
      last = await this.eval(expression);
      if (last) return last;
      await sleep(interval);
    }
    throw new Error(`timed out waiting for: ${expression}`);
  }
}

module.exports = { Browser, findChrome, removeDir, sleep };
