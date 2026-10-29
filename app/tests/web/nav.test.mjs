import test from 'node:test';
import assert from 'node:assert/strict';
import { browser, NodeStub } from './dom.mjs';
const env = browser();
const U = await import('../../web/ui.js');
const P = await import('../../web/panels.js');
const Q = await import('../../web/parity.js');
const S = { safe: false, writeBlocked: false, panes: new Map(), profiles: [], root: 'D:/repo', session: 's', workspaces: [], tabs: [], layouts: [] };
const ctx = { S, LS: { get: (k, d) => d, set() {} }, nameOf: (p) => p.name || p.key, statusOf: (p) => p.agent_status, schedule() {}, card: () => null, currentKey: () => null, plainOf: () => '' };
P.init(ctx); Q.init(ctx);
const key = (k, extra = {}) => Object.assign(new Event('keydown', { bubbles: true }), { key: k, preventDefault() {}, stopImmediatePropagation() {} }, extra);
const btn = (id, attrs = {}) => { const b = document.querySelector('#' + id); for (const [k, v] of Object.entries(attrs)) b.setAttribute(k, v); return b; };

test('More menu: opens on the button, arrows move, Esc closes and returns focus, an item click closes', () => {
  const trigger = btn('btn-more', { 'aria-expanded': 'false' }), menu = btn('more-menu', { role: 'menu' });
  const a = new NodeStub('button'), b = new NodeStub('button'); menu.append(a, b); menu.hidden = true;
  U.bindMenu(trigger, menu);
  trigger.dispatchEvent(new Event('click'));
  assert.equal(menu.hidden, false); assert.equal(trigger.getAttribute('aria-expanded'), 'true'); assert.equal(document.activeElement, a);
  document.dispatchEvent(key('ArrowDown')); assert.equal(document.activeElement, b);
  document.dispatchEvent(key('ArrowDown')); assert.equal(document.activeElement, a);   // wraps
  document.dispatchEvent(key('Escape'));
  assert.equal(menu.hidden, true); assert.equal(trigger.getAttribute('aria-expanded'), 'false'); assert.equal(document.activeElement, trigger);
  // The trigger toggles; after closing, arrows no longer move focus (listeners removed).
  // (Item click closing needs real bubbling: checked in the browser probe.)
  trigger.dispatchEvent(new Event('click')); assert.equal(menu.hidden, false);
  trigger.dispatchEvent(new Event('click')); assert.equal(menu.hidden, true); assert.equal(document.activeElement, trigger);
  document.dispatchEvent(key('ArrowDown')); assert.equal(document.activeElement, trigger);
});

test('panel buttons show pressed while their panel is open; focus returns to the opener on close', async () => {
  const usage = btn('btn-usage', { 'data-drawer': 'usage' }), inbox = btn('btn-inbox', { 'data-drawer': 'inbox' });
  document.querySelector('#drawer').hidden = true;
  usage.focus();
  P.drawer('usage', 'Usage', () => {});
  assert.equal(usage.getAttribute('aria-pressed'), 'true'); assert.equal(inbox.getAttribute('aria-pressed'), 'false');
  assert.equal(document.activeElement, document.querySelector('#drawer-close'));
  document.querySelector('#drawer').append(document.querySelector('#drawer-close'));   // the close button lives in the panel
  P.closeDrawer();
  assert.equal(usage.getAttribute('aria-pressed'), 'false'); assert.equal(document.activeElement, usage);
  // Opened from a hidden menu item: focus goes to the menu button.
  const item = new NodeStub('button'); document.querySelector('#more-menu').append(item); item.focus();
  P.drawer('board', 'Agent board', () => {}); P.closeDrawer();
  assert.equal(document.activeElement, document.querySelector('#btn-more'));
  // Closing while the user works elsewhere does not steal focus.
  usage.focus(); P.drawer('usage', 'Usage', () => {}); const other = new NodeStub('textarea'); document.body.append(other); other.focus();
  P.closeDrawer(); assert.equal(document.activeElement, other);
});

test('the help button, ? and Ctrl+B ? open the same panel', () => {
  document.querySelector('#drawer').hidden = true;
  Q.help(); const viaButton = document.querySelector('#drawer-title').textContent; P.closeDrawer();
  Q.keysOverlay(); const viaKey = document.querySelector('#drawer-title').textContent; P.closeDrawer();
  assert.equal(viaKey, viaButton);
  assert.ok(Q.CHORDS.some(([k, d]) => k === '?' && d === 'Keys and guide'));
});
