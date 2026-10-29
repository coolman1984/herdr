// User-sized areas: left rail width, right drawer width, card columns and rows in grid view.
// Sizes live in LS ('railW', 'drawerW', 'gridSizes' = { c<cols>: [fractions], r<rows>: [fractions] }),
// so each column/row count keeps its own split. Double-click a divider resets it; arrow keys move a
// focused divider. Drags write CSS variables once per frame; the cards' ResizeObserver refits the text.
const GAP = 12, STEP = 16, MIN_COL = 360, MIN_ROW = 240;
let LS = null, sizes = null;
const root = document.body;
const clamp = (v, lo, hi) => Math.max(lo, Math.min(hi, v));
const sum = (a) => a.reduce((s, x) => s + x, 0);

// One pointer drag: move(dx, dy) every frame, end() once. Listeners sit on window, so the drag
// survives the pointer leaving the 12-px handle (no pointer capture needed).
function drag(handle, axis, onStart, move, end) {
  handle.addEventListener('pointerdown', (ev) => {
    if (ev.button !== 0) return;
    ev.preventDefault();
    const x0 = ev.clientX, y0 = ev.clientY;
    onStart();
    let frame = 0, last = null;
    handle.classList.add('drag'); root.classList.add('resizing', axis === 'v' ? 'col' : 'row');
    const mv = (e) => {
      last = e;
      if (!frame) frame = requestAnimationFrame(() => { frame = 0; move(last.clientX - x0, last.clientY - y0); });
    };
    const up = (e) => {
      window.removeEventListener('pointermove', mv); window.removeEventListener('pointerup', up); window.removeEventListener('pointercancel', up);
      if (frame) { cancelAnimationFrame(frame); frame = 0; }
      move(e.clientX - x0, e.clientY - y0);
      handle.classList.remove('drag'); root.classList.remove('resizing', 'col', 'row');
      end();
    };
    window.addEventListener('pointermove', mv); window.addEventListener('pointerup', up); window.addEventListener('pointercancel', up);
  });
}
// Arrow keys on a focused divider: step(+-STEP px); Home/double-click: reset().
function keys(handle, axis, step, reset) {
  handle.addEventListener('keydown', (ev) => {
    const back = axis === 'v' ? 'ArrowLeft' : 'ArrowUp', fwd = axis === 'v' ? 'ArrowRight' : 'ArrowDown';
    if (ev.key === back || ev.key === fwd) { ev.preventDefault(); ev.stopPropagation(); step((ev.key === fwd ? 1 : -1) * (ev.shiftKey ? STEP * 4 : STEP)); }
    else if (ev.key === 'Home') { ev.preventDefault(); ev.stopPropagation(); reset(); }
  });
  handle.addEventListener('dblclick', (ev) => { ev.preventDefault(); reset(); });
}

// ---------- rail and drawer: a pixel width in a CSS variable on <body> ----------
function side(handle, name, cssVar, def, min, maxOf, dir) {
  const cur = () => LS.get(name, null) ?? def;
  const set = (w, save) => {
    w = Math.round(clamp(w, min, maxOf()));
    root.style.setProperty(cssVar, w + 'px');
    handle.setAttribute('aria-valuenow', String(w));
    if (save) LS.set(name, w);
    return w;
  };
  let start = 0, now = 0;
  drag(handle, 'v', () => { start = cur(); now = start; }, (dx) => { now = set(start + dir * dx, false); }, () => set(now, true));
  keys(handle, 'v', (px) => set(cur() + dir * px, true), () => { LS.set(name, null); root.style.removeProperty(cssVar); handle.setAttribute('aria-valuenow', String(def)); });
  handle.setAttribute('aria-valuemin', String(min));
  return () => { const w = LS.get(name, null); if (w == null) root.style.removeProperty(cssVar); else set(w, false); };
}
let applySides = () => {};
export function initSplits(ls) {
  LS = ls;
  const rail = side(document.querySelector('#rail-split'), 'railW', '--rail-w', 232, 140, () => innerWidth * 0.5, 1);
  const drawer = side(document.querySelector('#drawer-split'), 'drawerW', '--drawer-w', 380, 280, () => innerWidth - 360, -1);
  applySides = () => { rail(); drawer(); };
  applySizes();
}
// Re-read saved sizes (after the backend preferences arrive).
export function applySizes() { sizes = null; state = null; applySides(); }

// ---------- grid: column and row fractions ----------
const saved = () => (sizes ??= { ...(LS.get('gridSizes', {}) || {}) });
function fracs(key, n) {
  const a = saved()[key];
  return Array.isArray(a) && a.length === n && a.every((x) => x > 0) ? a.slice() : new Array(n).fill(1 / n);
}
const tracks = (f) => f.map((x) => `minmax(0, ${(x * 1000).toFixed(2)}fr)`).join(' ');
// Divider k (1-based) sits in the gap after track k: offset = free * (f1..fk) + (k - 1) gaps + half a gap.
const at = (f, k) => `calc((100% - ${(f.length - 1) * GAP}px) * ${sum(f.slice(0, k)).toFixed(5)} + ${(k - 1) * GAP + GAP / 2}px)`;

let state = null;   // { grid, cols: [..], rows: [..], ck, rk } of the last layout
function paint() {
  const { grid, cols, rows } = state;
  grid.style.setProperty('--gcols', tracks(cols));
  grid.style.setProperty('--grows', tracks(rows));
  grid.style.setProperty('--grid-min', (rows.length * MIN_ROW + (rows.length - 1) * GAP) + 'px');
  for (const d of grid.querySelectorAll('.gd')) {
    const k = +d.dataset.k, f = d.classList.contains('v') ? cols : rows;
    d.style[d.classList.contains('v') ? 'left' : 'top'] = at(f, k);
    d.setAttribute('aria-valuenow', String(Math.round(sum(f.slice(0, k)) * 100)));
  }
}
function gridDivider(axis, k) {
  const d = document.createElement('div');
  d.className = 'divider gd ' + axis; d.dataset.k = String(k);
  Object.assign(d, { tabIndex: 0 });
  d.setAttribute('role', 'separator'); d.setAttribute('aria-orientation', axis === 'v' ? 'vertical' : 'horizontal');
  d.setAttribute('aria-label', (axis === 'v' ? 'Column' : 'Row') + ' size (arrows move, double-click resets)');
  d.setAttribute('aria-valuemin', '0'); d.setAttribute('aria-valuemax', '100');
  const list = () => axis === 'v' ? state.cols : state.rows;
  const key = () => axis === 'v' ? state.ck : state.rk;
  // Move the line between track k and k+1 by px; both keep at least the minimum size.
  const shift = (base, px, total) => {
    const f = base.slice(), free = total - (f.length - 1) * GAP, min = Math.min((axis === 'v' ? MIN_COL : MIN_ROW) / free, (f[k - 1] + f[k]) / 2);
    const pair = f[k - 1] + f[k], a = clamp(f[k - 1] + px / free, min, pair - min);
    f[k - 1] = a; f[k] = pair - a;
    return f;
  };
  const total = () => { const r = state.grid.getBoundingClientRect(); return axis === 'v' ? r.width : r.height; };   // one read per gesture
  const put = (f) => { if (axis === 'v') state.cols = f; else state.rows = f; paint(); };
  const save = () => { saved()[key()] = list().map((x) => +x.toFixed(5)); LS.set('gridSizes', saved()); };
  let base = null, size = 0;
  drag(d, axis, () => { base = list(); size = total(); }, (dx, dy) => put(shift(base, axis === 'v' ? dx : dy, size)), save);
  keys(d, axis, (px) => { put(shift(list(), px, total())); save(); }, () => {
    const n = list().length; delete saved()[key()]; LS.set('gridSizes', saved()); put(new Array(n).fill(1 / n));
  });
  return d;
}
// Called on every grid render: n visible cards, cols chosen; layout view and zoom pass n = 0 / 1.
export function layoutGrid(grid, n, cols) {
  const c = Math.max(1, Math.min(cols, n)), r = Math.max(1, Math.ceil(n / c));
  const ck = 'c' + c, rk = 'r' + r;
  if (!state || state.grid !== grid || state.ck !== ck || state.rk !== rk) state = { grid, ck, rk, cols: fracs(ck, c), rows: fracs(rk, r) };
  const want = [...Array(c - 1)].map((_, i) => 'v' + (i + 1)).concat([...Array(r - 1)].map((_, i) => 'h' + (i + 1)));
  const have = [...grid.querySelectorAll('.gd')];
  if (have.map((d) => (d.classList.contains('v') ? 'v' : 'h') + d.dataset.k).join() !== want.join()) {
    for (const d of have) d.remove();
    for (const w of want) grid.append(gridDivider(w[0], +w.slice(1)));
  }
  paint();
}
// Herdr layout view: no grid dividers, no fractions.
export function clearGrid(grid) {
  for (const d of grid.querySelectorAll('.gd')) d.remove();
  for (const v of ['--gcols', '--grows', '--grid-min']) grid.style.removeProperty(v);
  state = null;
}
