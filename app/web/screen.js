// Live screen view, built for 60 fps:
// - rows are parsed once and cached (raw text + colour state in -> html + state out);
// - only visible rows (+ overscan) exist in the DOM, each absolutely placed with a transform;
//   a row element is rewritten only when its html changed;
// - no layout reads on the update path: scroll position and size come from scroll events and
//   a shared ResizeObserver; the shared rAF loop calls paint().
// - bounded: at most MAX_LINES rows; old rows drop off the top.
import { parseLine, newState, stateKey, stripAnsi } from './ansi.js';

export const MAX_LINES = 2000;
const OVERSCAN = 12, SHIFT_SEARCH = 600;

const ro = new ResizeObserver((entries) => {
  for (const e of entries) { const v = e.target._view; if (v) { v.h = e.contentRect.height; v.need(); } }
});

export class ScreenView {
  constructor(box, requestPaint, signal) {
    this.box = box; this.need = () => { this.dirty = true; requestPaint(this); };
    this.spacer = box.appendChild(document.createElement('div'));
    this.spacer.className = 'spacer';
    this.layer = box.appendChild(document.createElement('div'));
    this.layer.className = 'rows';
    this.raw = []; this.cache = []; this.base = 0;
    this.rows = new Map(); this.pool = [];
    this.lh = 0; this.h = 0; this.top = 0; this.follow = true; this.jump = false; this.dirty = false;
    this.version = 0; this.plainCache = null;
    box._view = this; ro.observe(box);
    box.addEventListener('scroll', () => {
      const t = box.scrollTop;
      if (t === this.top) return;          // our own scroll from paint: nothing new to draw
      this.top = t;
      this.follow = this.top + this.h >= this.raw.length * this.lh - this.lh * 1.5;
      this.need();
    }, { passive: true, signal });
    signal?.addEventListener('abort', () => { ro.unobserve(box); box._view = null; });
  }
  get lines() { return this.raw.length; }
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
      this.reparse(Math.max(0, start - (this.raw.length < start ? 0 : 0)), start + lines.length);
    }
    this.touched();
    return true;
  }
  dropTop(k) {
    this.raw.splice(0, k); this.cache.splice(0, k); this.base += k;
    this.reparse(0, 0);
  }
  // Re-parse from row i0; stop at the first row at or after `until` whose text and start state match.
  reparse(i0, until) {
    const raw = this.raw, cache = this.cache, n = raw.length;
    let st = i0 > 0 && cache[i0 - 1] ? { ...cache[i0 - 1].end } : newState();
    for (let i = i0; i < n; i++) {
      const c = cache[i], k0 = stateKey(st);
      if (c && c.raw === raw[i] && c.k0 === k0) { if (i >= until) return; st = { ...c.end }; continue; }
      const html = parseLine(raw[i], st);
      cache[i] = { raw: raw[i], k0, html, end: { ...st } };
    }
  }
  touched() {
    this.plainCache = null; this.version++;
    if (this.follow) this.jump = true;
    this.need();
  }

  apply(lines) {
    const old = this.raw;
    // Rows dropped off the top (scrollback moving): find the old index of the new first row.
    if (old.length && lines.length && old[0] !== lines[0]) {
      const lim = Math.min(old.length - 1, SHIFT_SEARCH);
      for (let k = 1; k <= lim; k++) {
        if (old[k] === lines[0] && (lines.length < 2 || old[k + 1] === lines[1] || k + 1 >= old.length)) {
          this.cache.splice(0, k); this.base += k; break;
        }
      }
    }
    let st = newState(), changed = false;
    const cache = this.cache, n = lines.length;
    for (let i = 0; i < n; i++) {
      const c = cache[i], k0 = stateKey(st);
      if (c && c.raw === lines[i] && c.k0 === k0) { st = { ...c.end }; continue; }
      const html = parseLine(lines[i], st);
      cache[i] = { raw: lines[i], k0, html, end: { ...st } };
      changed = true;
    }
    if (cache.length > n) { cache.length = n; changed = true; }
    if (!changed && n === old.length) { this.raw = lines; return false; }
    this.raw = lines; this.plainCache = null; this.version++;
    if (this.follow) this.jump = true;
    this.need();
    return true;
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
    if (!this.lh) this.measureRow();
    const total = this.raw.length * this.lh;
    if (this.spacerH !== total) { this.spacer.style.height = total + 'px'; this.spacerH = total; }
    return true;
  }
  paintScroll() {
    if (!this.jump || !this.dirty || !this.h) return;
    this.jump = false;
    this.top = Math.max(0, this.spacerH - this.h);
    this.box.scrollTop = this.top;
  }
  paint() {
    if (!this.dirty || !this.h) return;
    this.dirty = false;
    const lh = this.lh, n = this.raw.length;
    const first = Math.max(0, Math.floor(this.top / lh) - OVERSCAN);
    const last = Math.min(n, Math.ceil((this.top + (this.h || 600)) / lh) + OVERSCAN);
    for (const [abs, r] of this.rows) {
      const i = abs - this.base;
      if (i < first || i >= last) { this.rows.delete(abs); r.hidden = true; this.pool.push(r); }
    }
    for (let i = first; i < last; i++) {
      const abs = i + this.base;
      let r = this.rows.get(abs);
      if (!r) {
        r = this.pool.pop() || this.layer.appendChild(document.createElement('div'));
        r.className = 'row'; r.hidden = false; r._y = -1; r._h = null;
        this.rows.set(abs, r);
      }
      const html = this.cache[i].html;
      if (r._h !== html) { r.innerHTML = html; r._h = html; }
      const y = i * lh;
      if (r._y !== y) { r.style.transform = `translateY(${y}px)`; r._y = y; }
    }
    // Keep the pool small: at most one screen of spare rows.
    while (this.pool.length > 80) this.pool.pop().remove();
  }
  scrollToLine(n) {
    if (!this.lh) this.measureRow();
    this.follow = false;
    this.top = Math.max(0, n * this.lh - this.h / 2);
    this.box.scrollTop = this.top;
    this.flash = n; this.need();
    requestAnimationFrame(() => {
      const r = this.rows.get(n + this.base);
      if (r) { r.classList.add('hit'); setTimeout(() => r.classList.remove('hit'), 1600); }
    });
  }
  domRows() { return this.rows.size; }
}
