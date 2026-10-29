// Live screen view, built for 60 fps:
// - rows are parsed once and cached (raw text + colour state in -> html + state out);
// - text reflows: a line longer than the card is wrapped into display rows of the card's width, so a
//   card never scrolls sideways (rule lines of box characters are cut instead);
// - only visible display rows (+ overscan) exist in the DOM, each absolutely placed with a transform;
//   a row element is rewritten only when its html changed;
// - no layout reads on the update path: scroll position and size come from scroll events and
//   a shared ResizeObserver; the shared rAF loop calls paint().
// - bounded: at most MAX_LINES lines; old lines drop off the top and the reading position stays put.
import { parseRows, newState, stateKey, stripAnsi } from './ansi.js';

export const MAX_LINES = 2000;
const OVERSCAN = 12, SHIFT_SEARCH = 600, SEGS = 256;   // row key = line * SEGS + display row in line

const views = new Set();
const ro = new ResizeObserver((entries) => {
  for (const e of entries) {
    const v = e.target._view; if (!v) continue;
    v.h = e.contentRect.height;
    if (v.w !== e.contentRect.width) { v.w = e.contentRect.width; v.fitSoon(); }
    if (v.follow) v.jump = true; v.need();
  }
});

// The monospace advance is measured once, again after the web font has loaded (it changes the width).
let charRatio = 0;
function monoRatio() {
  if (charRatio) return charRatio;
  const s = document.body.appendChild(document.createElement('span'));
  s.style.cssText = 'position:absolute;visibility:hidden;white-space:pre;font:100px var(--font-mono)';
  s.textContent = 'M'.repeat(20);
  charRatio = s.getBoundingClientRect().width / 2000 || 0.6;
  s.remove();
  return charRatio;
}
document.fonts?.ready.then(() => { charRatio = 0; for (const v of views) { v.fs = null; v.fit(); } });

// Font: the base size, grown (at most 1.25x) when the pane has fewer columns than the card fits, never
// shrunk; anything wider than the card wraps.
export const fitState = { on: true, base: 13 };
const PAD = 24;   // .trow left + right padding
export const SCREEN_PAD = PAD;
// One terminal cell at font size fs (row height follows --row-h = round(fs * 1.5)).
export function cellSize(fs = fitState.base) { return { w: fs * monoRatio(), h: Math.round(fs * 1.5) }; }

export class ScreenView {
  constructor(box, requestPaint, signal) {
    this.box = box; this.need = () => { this.dirty = true; requestPaint(this); };
    this.spacer = box.appendChild(document.createElement('div'));
    this.spacer.className = 'spacer';
    this.layer = box.appendChild(document.createElement('div'));
    this.layer.className = 'rows';
    this.raw = []; this.cache = []; this.base = 0; this.vc = 0;
    this.starts = null;   // display row where each line starts; null = recount
    this.rows = new Map(); this.pool = [];
    this.lh = 0; this.h = 0; this.top = 0; this.follow = true; this.jump = false; this.anchor = false; this.dirty = false;
    this.version = 0; this.plainCache = null;
    box._view = this; ro.observe(box); views.add(this);
    box.addEventListener('scroll', () => {
      const t = box.scrollTop;
      if (t === this.top) return;          // our own scroll from paint: nothing new to draw
      this.top = t;
      this.follow = this.top + this.h >= (this.spacerH || 0) - this.lh * 1.5;
      this.need();
    }, { passive: true, signal });
    signal?.addEventListener('abort', () => { ro.unobserve(box); box._view = null; views.delete(this); });
  }
  get lines() { return this.raw.length; }

  // Columns of the real pane (Herdr layout rect width); 0 = unknown -> longest recent line.
  setCols(n) { if (n !== this.cols) { this.cols = n; this.fit(); } }
  guessCols() {
    const now = performance.now();
    if (this.guessAt && now - this.guessAt < 2000) return this.guessed || 0;
    this.guessAt = now;
    let m = 0;
    for (let i = Math.max(0, this.raw.length - 200); i < this.raw.length; i++) { const l = stripAnsi(this.raw[i]).length; if (l > m) m = l; }
    return (this.guessed = m >= 20 ? Math.min(m, 400) : 0);
  }
  fit() { this.fitPending = true; this.need(); }
  // Width changes while a divider or the window is dragged: re-wrap at most every 100 ms, plus once at
  // the end (a re-wrap re-parses every line, too costly for each frame).
  fitSoon() {
    clearTimeout(this.fitTimer);
    const wait = this.fitAt == null ? 0 : 100 - (performance.now() - this.fitAt);   // first size: fit now
    if (wait <= 0) { this.fitAt = performance.now(); this.fit(); }
    else this.fitTimer = setTimeout(() => { this.fitAt = performance.now(); this.fit(); }, wait);
  }
  prepare() {
    if (this.fitPending) { this.fitPending = false; this.applyFit(); }
    if (this.dirty && this.h && !this.lh) this.measureRow();
  }
  applyFit() {
    const base = fitState.base, cols = this.cols || this.guessCols(), sized = this.w > PAD + 40;
    let fs = base;
    if (fitState.on && cols && sized) fs = Math.max(base, Math.min(base * 1.25, Math.floor(((this.w - PAD) / (cols * monoRatio())) * 4) / 4));
    if (fs !== this.fs) {
      this.fs = fs;
      this.fontWrite = { fs, base };
      this.lh = Math.round(fs * 1.5);
      for (const r of this.rows.values()) r._y = -1;
    }
    const vc = sized ? Math.max(20, Math.floor((this.w - PAD) / (fs * monoRatio()))) : 0;
    if (vc !== this.vc) { this.vc = vc; this.reparse(0, Infinity, true); this.touched(); }
  }
  get plain() { return (this.plainCache ??= stripAnsi(this.raw.join('\n'))); }

  // Full screen text from the backend.
  set(text) {
    if (text === this.lastText) return false;
    this.lastText = text;
    const lines = text.replace(/\r\n?/g, '\n').split('\n');
    if (lines.length > MAX_LINES) lines.splice(0, lines.length - MAX_LINES);
    return this.apply(lines);
  }
  // Row patch from the backend (API.md "screen_patch"): splice(start, delete, ...lines), only on top of
  // the version it was made from. Returns false when the versions do not match (caller resyncs).
  // Cost is O(changed rows): rows before `start` are untouched, rows after it are only re-parsed
  // until their colour state matches the cache again.
  patch(pt) {
    if (this.sv !== pt.from_version) return false;
    const { start, lines } = pt, del = pt.delete || 0;
    this.sv = pt.version; this.lastText = null;
    if (start === 0 && del && !lines.length) this.dropTop(del);
    else {
      this.raw.splice(start, del, ...lines);
      this.cache.splice(start, del, ...new Array(lines.length));
      if (this.raw.length > MAX_LINES) this.dropTop(this.raw.length - MAX_LINES);
      this.reparse(start, start + lines.length);
    }
    this.touched();
    return true;
  }
  dropTop(k) {
    this.keepPlace(k);
    this.raw.splice(0, k); this.cache.splice(0, k); this.base += k;
    this.reparse(0, 0);
  }
  // k lines leave the top: when the owner reads older output, move the scroll position up by their
  // display rows so the same text stays in view.
  keepPlace(k) {
    if (this.follow || !this.lh) return;
    let gone = 0;
    for (let i = 0; i < k && i < this.cache.length; i++) gone += this.cache[i]?.segs.length || 1;
    if (!gone) return;
    this.top = Math.max(0, this.top - gone * this.lh); this.anchor = true;
  }
  parse(line, st) { return parseRows(line, st, this.vc); }
  // Re-parse from line i0; stop at the first line at or after `until` whose text, start state and
  // width match the cache (all = true: re-parse every line, e.g. after a width change).
  reparse(i0, until, all = false) {
    const raw = this.raw, cache = this.cache, n = raw.length, vc = this.vc;
    let st = i0 > 0 && cache[i0 - 1] ? { ...cache[i0 - 1].end } : newState();
    for (let i = i0; i < n; i++) {
      const c = cache[i], k0 = stateKey(st);
      if (!all && c && c.raw === raw[i] && c.k0 === k0 && c.vc === vc) { if (i >= until) return; st = { ...c.end }; continue; }
      const segs = this.parse(raw[i], st);
      cache[i] = { raw: raw[i], k0, vc, segs, end: { ...st } };
    }
  }
  touched() {
    this.plainCache = null; this.version++; this.starts = null;
    if (this.follow) this.jump = true;
    this.need();
  }

  apply(lines) {
    const old = this.raw;
    let shifted = false;
    // Lines dropped off the top (scrollback moving): find the old index of the new first line.
    if (old.length && lines.length && old[0] !== lines[0]) {
      const lim = Math.min(old.length - 1, SHIFT_SEARCH);
      for (let k = 1; k <= lim; k++) {
        if (old[k] === lines[0] && (lines.length < 2 || old[k + 1] === lines[1] || k + 1 >= old.length)) {
          this.keepPlace(k); this.cache.splice(0, k); this.base += k; shifted = true; break;
        }
      }
    }
    let st = newState(), changed = false;
    const cache = this.cache, n = lines.length, vc = this.vc;
    for (let i = 0; i < n; i++) {
      const c = cache[i], k0 = stateKey(st);
      if (c && c.raw === lines[i] && c.k0 === k0 && c.vc === vc) { st = { ...c.end }; continue; }
      const segs = this.parse(lines[i], st);
      cache[i] = { raw: lines[i], k0, vc, segs, end: { ...st } };
      changed = true;
    }
    if (cache.length > n) { cache.length = n; changed = true; }
    if (!changed && !shifted && n === old.length) { this.raw = lines; return false; }
    this.raw = lines;
    this.touched();
    return true;
  }

  // Display row index where each line starts (starts[n] = total display rows). O(lines), only after a change.
  layout() {
    if (this.starts) return this.starts;
    const n = this.cache.length, s = new Int32Array(n + 1);
    for (let i = 0; i < n; i++) s[i + 1] = s[i] + (this.cache[i]?.segs.length || 1);
    return (this.starts = s);
  }
  lineAt(r) {   // the line that holds display row r
    const s = this.layout();
    let lo = 0, hi = s.length - 2;
    while (lo < hi) { const mid = (lo + hi + 1) >> 1; if (s[mid] <= r) lo = mid; else hi = mid - 1; }
    return lo;
  }

  measureRow() {
    // One read per view lifetime (and after a font change): the fixed row height.
    this.lh = parseFloat(getComputedStyle(this.box).getPropertyValue('--row-h')) || 18;
  }
  resetMetrics() { this.lh = 0; for (const r of this.rows.values()) { r._y = -1; } this.need(); }

  // The shared rAF loop calls the three passes for ALL dirty views in order: sizes, then scroll
  // positions (at most one forced layout per frame for every card together), then rows.
  paintSize() {
    if (!this.dirty || !this.h) return false;   // hidden card: stay dirty, paint when it gets a size
    if (this.fontWrite) {
      const { fs, base } = this.fontWrite; this.fontWrite = null;
      if (fs === base) { this.box.style.removeProperty('--mono-size'); this.box.style.removeProperty('--row-h'); }
      else { this.box.style.setProperty('--mono-size', fs + 'px'); this.box.style.setProperty('--row-h', Math.round(fs * 1.5) + 'px'); }
    }
    const s = this.layout(), total = s[s.length - 1] * this.lh;
    if (this.spacerH !== total) { this.spacer.style.height = total + 'px'; this.spacerH = total; }
    return true;
  }
  paintScroll() {
    if (!this.dirty || !this.h) return;
    if (this.jump) { this.jump = false; this.anchor = false; this.top = Math.max(0, this.spacerH - this.h); this.box.scrollTop = this.top; }
    else if (this.anchor) { this.anchor = false; this.box.scrollTop = this.top; }
  }
  paint() {
    if (!this.dirty || !this.h) return;
    this.dirty = false;
    const lh = this.lh, s = this.layout(), total = s[s.length - 1];
    const first = Math.max(0, Math.floor(this.top / lh) - OVERSCAN);
    const last = Math.min(total, Math.ceil((this.top + (this.h || 600)) / lh) + OVERSCAN);
    const lo = total ? this.lineAt(first) : 0, hi = total ? this.lineAt(Math.max(first, last - 1)) : -1;
    for (const [key, r] of this.rows) {
      const i = Math.floor(key / SEGS) - this.base, y = i >= 0 && i < s.length - 1 ? s[i] + key % SEGS : -1;
      if (y < first || y >= last || key % SEGS >= (this.cache[i]?.segs.length || 1)) { this.rows.delete(key); r.hidden = true; this.pool.push(r); }
    }
    for (let i = lo; i <= hi; i++) {
      const segs = this.cache[i].segs;
      for (let j = 0; j < segs.length && j < SEGS; j++) {
        const row = s[i] + j;
        if (row < first || row >= last) continue;
        const key = (i + this.base) * SEGS + j;
        let r = this.rows.get(key);
        if (!r) {
          r = this.pool.pop() || this.layer.appendChild(document.createElement('div'));
          r.className = 'trow'; r.hidden = false; r._y = -1; r._h = null;
          this.rows.set(key, r);
        }
        if (r._h !== segs[j]) { r.innerHTML = segs[j]; r._h = segs[j]; }
        const y = row * lh;
        if (r._y !== y) { r.style.transform = `translateY(${y}px)`; r._y = y; }
      }
    }
    // Keep the pool small: at most one screen of spare rows.
    while (this.pool.length > 80) this.pool.pop().remove();
  }
  scrollToLine(n) {
    if (!this.lh) this.measureRow();
    this.follow = false;
    this.top = Math.max(0, this.layout()[Math.min(n, this.raw.length)] * this.lh - this.h / 2);
    this.box.scrollTop = this.top;
    this.flash = n; this.need();
    requestAnimationFrame(() => {
      const r = this.rows.get((n + this.base) * SEGS);
      if (r) { r.classList.add('hit'); setTimeout(() => r.classList.remove('hit'), 1600); }
    });
  }
  domRows() { return this.rows.size; }
}
