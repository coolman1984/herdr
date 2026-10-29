import test from 'node:test';
import assert from 'node:assert/strict';
import { browser, NodeStub } from './dom.mjs';
const env = browser();
const { initSplits, layoutGrid, clearGrid } = await import('../../web/split.js');
const { ScreenView } = await import('../../web/screen.js');

const store = new Map();
const LS = { get: (k, d) => store.has(k) ? store.get(k) : d, set: (k, v) => store.set(k, v) };
const styled = (n) => { const props = {}; n.style = { props, setProperty(k, v) { props[k] = v; }, removeProperty(k) { delete props[k]; } }; return n; };
styled(document.body);
initSplits(LS);
const ev = (type, extra = {}) => Object.assign(new Event(type, { bubbles: true }), { button: 0, clientX: 0, clientY: 0, preventDefault() {}, stopPropagation() {} }, extra);
const gridOf = (w = 1012, h = 612) => { const g = styled(new NodeStub('div')); g.rect = { width: w, height: h }; return g; };
const dividers = (g, axis) => g.querySelectorAll('.gd').filter((d) => d.classList.contains(axis) || d.className.split(' ').includes(axis));

test('grid: rows and columns follow the card count; 1 card in 2 columns fills the width', () => {
  store.clear();
  const g = gridOf();
  layoutGrid(g, 4, 2);
  assert.equal(dividers(g, 'v').length, 1); assert.equal(dividers(g, 'h').length, 1);
  assert.equal(g.style.props['--gcols'], 'minmax(0, 500.00fr) minmax(0, 500.00fr)');
  assert.equal(g.style.props['--grows'], 'minmax(0, 500.00fr) minmax(0, 500.00fr)');
  layoutGrid(g, 2, 2);
  assert.equal(dividers(g, 'h').length, 0); assert.equal(g.style.props['--grows'], 'minmax(0, 1000.00fr)');
  layoutGrid(g, 1, 2);
  assert.equal(g.querySelectorAll('.gd').length, 0); assert.equal(g.style.props['--gcols'], 'minmax(0, 1000.00fr)');
  clearGrid(g);
  assert.equal(g.style.props['--gcols'], undefined);
});

test('grid: a pointer drag moves the column line, keeps the minimum and is saved per column count', () => {
  store.clear();
  const g = gridOf(); layoutGrid(g, 2, 2);
  const d = dividers(g, 'v')[0];
  d.dispatchEvent(ev('pointerdown', { clientX: 100 }));
  window.dispatchEvent(ev('pointermove', { clientX: 200 })); env.frame();
  assert.equal(g.style.props['--gcols'], 'minmax(0, 600.00fr) minmax(0, 400.00fr)');
  window.dispatchEvent(ev('pointerup', { clientX: 200 }));
  assert.deepEqual(store.get('gridSizes').c2, [0.6, 0.4]);
  // Too far: the right column keeps 360 px of the 1000 free px.
  d.dispatchEvent(ev('pointerdown', { clientX: 0 }));
  window.dispatchEvent(ev('pointerup', { clientX: 5000 }));
  assert.deepEqual(store.get('gridSizes').c2, [0.64, 0.36]);
  // A new render with the same counts keeps the saved split; 3 columns has its own.
  const g2 = gridOf(); layoutGrid(g2, 3, 3);
  assert.equal(g2.style.props['--gcols'].split(') ').length, 3);
  assert.ok(g2.style.props['--gcols'].startsWith('minmax(0, 333.33fr)'));
});

test('grid: arrow keys move a focused divider, double-click resets it', () => {
  store.clear();
  const g = gridOf(); layoutGrid(g, 4, 2);
  const h = dividers(g, 'h')[0];
  h.dispatchEvent(ev('keydown', { key: 'ArrowDown' }));
  assert.deepEqual(store.get('gridSizes').r2, [0.52667, 0.47333]);   // 16 px of 600 free px
  h.dispatchEvent(ev('keydown', { key: 'ArrowLeft' }));                // wrong axis: ignored
  assert.deepEqual(store.get('gridSizes').r2, [0.52667, 0.47333]);
  h.dispatchEvent(ev('dblclick'));
  assert.equal(store.get('gridSizes').r2, undefined);
  assert.equal(g.style.props['--grows'], 'minmax(0, 500.00fr) minmax(0, 500.00fr)');
});

test('rail and drawer widths: keys and drags are clamped, saved and reset', () => {
  store.clear();
  const rail = document.querySelector('#rail-split'), drawer = document.querySelector('#drawer-split');
  rail.dispatchEvent(ev('keydown', { key: 'ArrowRight' }));
  assert.equal(store.get('railW'), 248); assert.equal(document.body.style.props['--rail-w'], '248px');
  rail.dispatchEvent(ev('pointerdown', { clientX: 0 }));
  window.dispatchEvent(ev('pointerup', { clientX: 900 }));   // window 800 px: at most half
  assert.equal(store.get('railW'), 400);
  rail.dispatchEvent(ev('dblclick'));
  assert.equal(store.get('railW'), null); assert.equal(document.body.style.props['--rail-w'], undefined);
  // The drawer grows to the left.
  drawer.dispatchEvent(ev('pointerdown', { clientX: 500 }));
  window.dispatchEvent(ev('pointerup', { clientX: 400 }));
  assert.equal(store.get('drawerW'), 440);   // 380 + 100, capped at innerWidth - 360 = 440
  drawer.dispatchEvent(ev('keydown', { key: 'ArrowRight', shiftKey: true }));
  assert.equal(store.get('drawerW'), 376);
});

test('screen refit on width change is immediate first, then at most every 100 ms', () => {
  const box = new NodeStub(), view = new ScreenView(box, () => {}, new AbortController().signal);
  const before = env.timers.size;
  view.fitSoon(); assert.equal(view.fitPending, true); assert.equal(env.timers.size, before);
  view.fitPending = false; view.fitSoon();
  assert.equal(view.fitPending, false); assert.equal(env.timers.size, before + 1);
});
