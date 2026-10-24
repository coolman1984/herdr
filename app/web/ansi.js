// ANSI SGR -> safe HTML. Pane text is untrusted: every character is escaped, only our own
// <span class> / style="color:#rrggbb" markup is produced. Other escape sequences are dropped.
const ESC_RE = /\x1b(?:\[([0-?]*)[ -/]*([@-~])|\][^\x07\x1b]*(?:\x07|\x1b\\)|[@-Z\\-_])/g;
const HTML = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' };
const esc = (s) => s.replace(/[&<>"]/g, (c) => HTML[c]);

const CUBE = [0, 95, 135, 175, 215, 255];
function c256(n) {
  if (n < 16) return { cls: n };
  if (n >= 232) { const v = 8 + (n - 232) * 10; return { hex: hex(v, v, v) }; }
  n -= 16;
  return { hex: hex(CUBE[Math.floor(n / 36)], CUBE[Math.floor(n / 6) % 6], CUBE[n % 6]) };
}
const hex = (r, g, b) => '#' + [r, g, b].map((v) => (v & 255).toString(16).padStart(2, '0')).join('');

function applySgr(st, params) {
  const p = params === '' ? [0] : params.split(';').map((x) => parseInt(x || '0', 10));
  for (let i = 0; i < p.length; i++) {
    const n = p[i];
    if (n === 0) { st.fg = st.bg = null; st.b = st.d = st.i = st.u = st.r = false; }
    else if (n === 1) st.b = true;
    else if (n === 2) st.d = true;
    else if (n === 3) st.i = true;
    else if (n === 4) st.u = true;
    else if (n === 7) st.r = true;
    else if (n === 22) st.b = st.d = false;
    else if (n === 23) st.i = false;
    else if (n === 24) st.u = false;
    else if (n === 27) st.r = false;
    else if (n >= 30 && n <= 37) st.fg = { cls: n - 30 };
    else if (n >= 90 && n <= 97) st.fg = { cls: n - 82 };
    else if (n === 39) st.fg = null;
    else if (n >= 40 && n <= 47) st.bg = { cls: n - 40 };
    else if (n >= 100 && n <= 107) st.bg = { cls: n - 92 };
    else if (n === 49) st.bg = null;
    else if (n === 38 || n === 48) {
      let c = null;
      if (p[i + 1] === 5) { c = c256(p[i + 2] | 0); i += 2; }
      else if (p[i + 1] === 2) { c = { hex: hex(p[i + 2] | 0, p[i + 3] | 0, p[i + 4] | 0) }; i += 4; }
      if (n === 38) st.fg = c; else st.bg = c;
    }
  }
}

function open(st) {
  let fg = st.fg, bg = st.bg;
  if (st.r) { [fg, bg] = [bg || { cls: 'bg' }, fg || { cls: 'fg' }]; }
  const cls = [], sty = [];
  if (fg) fg.hex ? sty.push('color:' + fg.hex) : cls.push('f' + fg.cls);
  if (bg) bg.hex ? sty.push('background:' + bg.hex) : cls.push('b' + bg.cls);
  if (st.b) cls.push('ab'); if (st.d) cls.push('ad'); if (st.i) cls.push('ai'); if (st.u) cls.push('au');
  if (!cls.length && !sty.length) return '';
  return '<span' + (cls.length ? ' class="' + cls.join(' ') + '"' : '') + (sty.length ? ' style="' + sty.join(';') + '"' : '') + '>';
}

export const newState = () => ({ fg: null, bg: null, b: false, d: false, i: false, u: false, r: false });
const ck = (c) => (c ? (c.hex || c.cls) : '-');
export const stateKey = (st) => ck(st.fg) + '/' + ck(st.bg) + (st.b ? 'b' : '') + (st.d ? 'd' : '') + (st.i ? 'i' : '') + (st.u ? 'u' : '') + (st.r ? 'r' : '');
const MAX_COLS = 2000;

// Parses one raw line starting in colour state st (mutated to the line's end state). Very long
// lines are clipped with a marker; "Copy screen" still copies the full text.
export function parseLine(raw, st) {
  let clipped = false, cols = 0, last = 0, m;
  raw = raw.replace(/ +$/, '');   // terminals pad rows with spaces; they only add a scroll bar
  if (raw.length > MAX_COLS * 3) { raw = raw.slice(0, MAX_COLS * 3); clipped = true; }
  let html = open(st), span = html !== '';
  const text = (t) => {
    if (cols + t.length > MAX_COLS) { t = t.slice(0, Math.max(0, MAX_COLS - cols)); clipped = true; }
    cols += t.length; html += esc(t);
  };
  ESC_RE.lastIndex = 0;
  while ((m = ESC_RE.exec(raw))) {
    text(raw.slice(last, m.index)); last = ESC_RE.lastIndex;
    if (m[2] === 'm') {
      if (span) html += '</span>';
      applySgr(st, m[1]);
      const o = open(st); html += o; span = o !== '';
    }
  }
  text(raw.slice(last));
  if (span) html += '</span>';
  if (clipped) html += '<span class="clip-more" title="Line clipped here; Copy screen has the full line">…</span>';
  return html;
}

// One HTML string per line; colour state carries across lines.
export function ansiLines(text) {
  const st = newState();
  return text.replace(/\r\n?/g, '\n').split('\n').map((l) => parseLine(l, st));
}

export const stripAnsi = (text) => text.replace(ESC_RE, '').replace(/\r\n?/g, '\n');
