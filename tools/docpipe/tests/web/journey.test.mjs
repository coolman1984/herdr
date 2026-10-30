// Real-browser journey test for the docpipe web app (Playwright + Chromium, node --test).
//   node --test tools/docpipe/tests/web/journey.test.mjs
// Waits on the page's own readiness marker (data-ready="1"), never on network idle.
import { test, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { spawn, execFileSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const here = dirname(fileURLToPath(import.meta.url));
const pkg = join(here, '..', '..');
const shots = process.env.SHOTS || join(pkg, '..', '..', 'tasks', 'docpipe-web-shots');
const py = process.platform === 'win32' ? 'python' : 'python3';
let work, browser, servers = [];

async function serve(db) {
  const port = 20000 + Math.floor(Math.random() * 20000);
  const p = spawn(py, ['-m', 'docpipe', 'serve', '--db', db, '--port', String(port)], { cwd: pkg, stdio: 'ignore' });
  servers.push(p);
  const url = `http://127.0.0.1:${port}`;
  for (let i = 0; i < 100; i++) {
    try { if ((await fetch(url + '/healthz')).ok) return url; } catch { }
    await new Promise(r => setTimeout(r, 100));
  }
  throw new Error('server did not start');
}

async function ready(page) { await page.waitForSelector('main[data-ready="1"]', { timeout: 10000 }); }

function luminance(rgb) {
  const [r, g, b] = rgb.match(/\d+/g).slice(0, 3).map(Number).map(v => { v /= 255; return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}
const contrast = (a, b) => { const [x, y] = [luminance(a), luminance(b)].sort((p, q) => q - p); return (x + 0.05) / (y + 0.05); };

let url, emptyUrl;
before(async () => {
  work = mkdtempSync(join(tmpdir(), 'docpipe-web-'));
  execFileSync(py, [join(here, 'make_demo.py'), work, '20000'], { stdio: 'ignore' });
  url = await serve(join(work, 'demo.db'));
  emptyUrl = await serve(join(work, 'empty.db'));
  browser = await chromium.launch();
  mkdirSync(shots, { recursive: true });
});
after(async () => { await browser?.close(); servers.forEach(s => s.kill()); });

for (const [name, viewport] of [['phone', { width: 390, height: 844 }], ['desktop', { width: 1280, height: 800 }]]) {
  for (const scheme of ['light', 'dark']) {
    test(`dashboard on ${name} (${scheme}): totals, no sideways scroll, readable contrast`, async () => {
      const ctx = await browser.newContext({ viewport, colorScheme: scheme, locale: 'ar-EG' });
      const page = await ctx.newPage();
      await page.goto(url + '/');
      await ready(page);
      assert.equal(await page.getAttribute('html', 'dir'), 'rtl');
      const totals = await page.$$eval('[data-total]', els => els.map(e => e.dataset.total));
      assert.ok(totals.includes('2150.75'), 'sales total 2150.75 shown: ' + totals);
      const ledger = 20000 * 0 + Array.from({ length: 20000 }, (_, i) => (i % 1000) + 0.5).reduce((a, b) => a + b, 0);
      assert.ok(totals.some(t => Number(t) === ledger), `ledger total ${ledger} shown: ${totals}`);
      const fields = await page.$$eval('.card .muted:first-child', els => els.map(e => e.textContent));
      assert.ok(!fields.includes('client'), 'a text column (client names with numbers in them) is not a numeric total: ' + fields);
      assert.equal(totals.length, 2, 'exactly the two money fields: ' + totals);
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
      assert.ok(overflow <= 1, `page scrolls sideways by ${overflow}px`);
      for (const sel of ['main h1', '.card .muted', 'header a', 'td']) {
        const [fg, bg] = await page.$eval(sel, el => {
          const cs = getComputedStyle(el); let n = el, bg = 'rgba(0, 0, 0, 0)';
          while (n && (bg === 'rgba(0, 0, 0, 0)' || bg === 'transparent')) { bg = getComputedStyle(n).backgroundColor; n = n.parentElement; }
          return [cs.color, bg];
        });
        assert.ok(contrast(fg, bg) >= 4.5, `${sel} contrast ${contrast(fg, bg).toFixed(2)} < 4.5 (${fg} on ${bg})`);
      }
      await page.screenshot({ path: join(shots, `dashboard-${name}-${scheme}.png`), fullPage: false });
      await ctx.close();
    });
  }
}

test('keyboard only: skip link, open a file, values carry their source', async () => {
  const page = await browser.newPage();
  await page.goto(url + '/');
  await ready(page);
  await page.keyboard.press('Tab');
  assert.equal(await page.evaluate(() => document.activeElement.className), 'skip');
  await page.keyboard.press('Enter');
  let found = false;
  for (let i = 0; i < 30 && !found; i++) {
    await page.keyboard.press('Tab');
    found = await page.evaluate(() => (document.activeElement.getAttribute('href') || '').startsWith('/file/'));
  }
  assert.ok(found, 'a file link is reachable by Tab');
  await Promise.all([page.waitForURL(/\/file\//), page.keyboard.press('Enter')]);
  await ready(page);
  const locs = await page.$$eval('td code', els => els.map(e => e.textContent));
  assert.ok(locs.some(l => /!(A|B)\d+$/.test(l) || /^p\d+$/.test(l) || /^s\d+:/.test(l)), 'locations shown: ' + locs.slice(0, 5));
  const outline = await page.evaluate(() => { const a = document.querySelector('a'); a.focus(); return getComputedStyle(a).outlineStyle; });
  assert.notEqual(outline, 'none', 'focus is visible');
  await page.close();
});

test('search: loading state on a slow network, results, then an error state', async () => {
  const ctx = await browser.newContext({ viewport: { width: 390, height: 844 } });
  const page = await ctx.newPage();
  await page.route('**/api/search*', async route => { await new Promise(r => setTimeout(r, 1200)); await route.continue(); });
  await page.goto(url + '/search');
  await ready(page);
  await page.fill('#q', 'القاهرة');
  await page.click('button[type=submit]');
  await page.waitForSelector('#results[aria-busy="true"] [role="status"]', { timeout: 2000 });
  await page.waitForSelector('#results[data-ready="1"]:not([aria-busy]) table', { timeout: 10000 });
  assert.ok((await page.textContent('#results')).includes('مبيعات!A2'));
  await page.screenshot({ path: join(shots, 'search-phone.png') });
  await page.unroute('**/api/search*');
  await page.route('**/api/search*', route => route.abort());
  await page.fill('#q', '1250.5');
  await page.click('button[type=submit]');
  await page.waitForSelector('#results [role="alert"]', { timeout: 5000 });
  await ctx.close();
});

test('empty database, unknown page, English and reduced motion', async () => {
  const page = await browser.newPage();
  await page.goto(emptyUrl + '/');
  await ready(page);
  assert.ok(await page.$('[role="status"]'), 'empty state shown');
  const resp = await page.goto(url + '/file/99999');
  assert.equal(resp.status(), 404);
  await page.goto(url + '/?lang=en');
  await ready(page);
  assert.equal(await page.getAttribute('html', 'dir'), 'ltr');
  await page.emulateMedia({ reducedMotion: 'reduce' });
  const tr = await page.$eval('.card', el => getComputedStyle(el).transitionDuration);
  assert.equal(tr, '0s', 'no motion when reduced motion is asked');
  assert.ok(await page.$('main') && await page.$('header nav[aria-label]') && await page.$('label[for="q"]') === null, 'landmarks present');
  await page.goto(url + '/search?lang=en');
  assert.ok(await page.$('label[for="q"]'), 'search field has a label');
  await page.close();
});

test('performance on a slowed CPU with 20,000 records', async () => {
  const ctx = await browser.newContext({ viewport: { width: 390, height: 844 } });
  const page = await ctx.newPage();
  const cdp = await ctx.newCDPSession(page);
  await cdp.send('Emulation.setCPUThrottlingRate', { rate: 4 });
  const t0 = Date.now();
  await page.goto(url + '/');
  await ready(page);
  const ms = Date.now() - t0;
  const nav = await page.evaluate(() => { const n = performance.getEntriesByType('navigation')[0]; return { dcl: Math.round(n.domContentLoadedEventEnd), size: n.transferSize, nodes: document.getElementsByTagName('*').length }; });
  console.log(`dashboard, 4x CPU slowdown: ready in ${ms} ms, DOMContentLoaded ${nav.dcl} ms, ${nav.size} bytes, ${nav.nodes} nodes`);
  assert.ok(ms < 3000, `dashboard took ${ms} ms`);
  assert.ok(nav.nodes < 1500, `${nav.nodes} DOM nodes`);
  await page.goto(url + '/file/2');
  await ready(page);
  const rows = await page.$$eval('tbody tr', r => r.length);
  assert.equal(rows, 100, 'large file is paged, 100 rows per page');
  await ctx.close();
});
