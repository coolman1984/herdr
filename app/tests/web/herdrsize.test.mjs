import test from 'node:test';
import assert from 'node:assert/strict';
import { browser } from './dom.mjs';
browser();
const { areaCells, sizeReporter, PANE_FRAME } = await import('../../web/herdrsize.js');

const cell = { w: 8, h: 20 };
const box = (left, top, width, height) => ({ left, top, width, height });

test('two cards side by side: Herdr width = both cards in cells + pane frames (3 cols, 2 rows each)', () => {
  // 824 px screen - 24 px padding = 800 px = 100 cells each; 400 px = 20 rows
  const a = areaCells([box(0, 0, 824, 400), box(840, 0, 824, 400)], cell);
  assert.deepEqual(a, { cols: 2 * (100 + PANE_FRAME.cols), rows: 20 + PANE_FRAME.rows });
});

test('2x2 grid: first row gives the columns, first column the rows', () => {
  const a = areaCells([box(0, 0, 424, 300), box(440, 0, 424, 300), box(0, 320, 424, 300), box(440, 320, 424, 300)], cell);
  assert.deepEqual(a, { cols: 2 * (50 + PANE_FRAME.cols), rows: 2 * (15 + PANE_FRAME.rows) });
});

test('no cards or no font metrics: nothing to report; tiny or huge areas are clamped', () => {
  assert.equal(areaCells([], cell), null);
  assert.equal(areaCells([box(0, 0, 500, 500)], { w: 0, h: 20 }), null);
  assert.deepEqual(areaCells([box(0, 0, 30, 30)], cell), { cols: 20, rows: 5 });
  assert.deepEqual(areaCells([box(0, 0, 99999, 99999)], cell), { cols: 1000, rows: 500 });
});

test('reporter sends a size once, again only when it changes or after a reset', async () => {
  let size = { cols: 100, rows: 30 };
  const sent = [];
  const r = sizeReporter(() => size, (c, w) => sent.push([c, w]));
  r.now(); r.now();
  size = { cols: 120, rows: 30 }; r.now();
  r.reset(); r.now();
  assert.deepEqual(sent, [[100, 30], [120, 30], [120, 30]]);
});
