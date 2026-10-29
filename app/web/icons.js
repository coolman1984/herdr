// Inline SVG icon set (24px grid, 1.8 stroke, currentColor) and coloured agent glyphs.
// All markup here is our own constant text, never pane data.
const P = {
  plus: 'M12 5v14M5 12h14',
  shell: 'M4 6l5 5-5 5M11 18h9',
  team: 'M9 11a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7zM2.5 20a6.5 6.5 0 0 1 13 0M16 4.3a3.5 3.5 0 0 1 0 6.4M18 14.5a6.5 6.5 0 0 1 3.5 5.5',
  gauge: 'M12 14l4-4M4.9 19a9 9 0 1 1 14.2 0',
  inbox: 'M3 13l2.5-8h13L21 13v6H3zM3 13h5l1.5 3h5L16 13h5',
  search: 'M11 18a7 7 0 1 0 0-14 7 7 0 0 0 0 14zM20 20l-4-4',
  command: 'M9 6a3 3 0 1 0-3 3h12a3 3 0 1 0-3-3v12a3 3 0 1 0 3-3H6a3 3 0 1 0 3 3z',
  help: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM9.5 9.5a2.5 2.5 0 1 1 3.5 2.3c-.6.3-1 .9-1 1.6v.6M12 17.5v.01',
  x: 'M6 6l12 12M18 6L6 18',
  max: 'M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5',
  min: 'M9 4v5H4M15 4v5h5M9 20v-5H4M15 20v-5h5',
  splitR: 'M4 4h16v16H4zM12 4v16',
  splitD: 'M4 4h16v16H4zM4 12h16',
  copy: 'M9 9h11v11H9zM5 15H4V4h11v1',
  paste: 'M9 4h6v3H9zM8 5H5v16h14V5h-3M9 13h6M9 17h4',
  edit: 'M4 20h4L19 9l-4-4L4 16zM14 6l4 4',
  close: 'M5 7h14M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3',
  more: 'M5 12h.01M12 12h.01M19 12h.01',
  send: 'M4 12l16-8-6 16-3-7z',
  folder: 'M3 6h6l2 2h10v11H3z',
  layout: 'M4 4h16v16H4zM9 4v16M15 4v16',
  board: 'M4 5h16M4 12h16M4 19h10',
  spaces: 'M3 5h8v14H3zM13 5h8v14h-8z',
  power: 'M12 3v9M6.3 6.3a8 8 0 1 0 11.4 0',
  shield: 'M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6z',
  sun: 'M12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM12 2v2M12 20v2M2 12h2M20 12h2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4',
  check: 'M5 12l5 5L20 7',
  git: 'M6 3v12M18 9a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM6 21a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM18 9a9 9 0 0 1-9 9',
  clock: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 7v5l3 2',
  tools: 'M14.7 6.3a4 4 0 0 0-5.4 5.4L3 18l3 3 6.3-6.3a4 4 0 0 0 5.4-5.4l-2.5 2.5-2.8-.7-.7-2.8z',
  task: 'M9 5h10M9 12h10M9 19h10M4 5l1 1 2-2M4 12l1 1 2-2M4 19l1 1 2-2',
  key: 'M7 14a4 4 0 1 1 3.9-5H21v4h-2v3h-3v-3h-5.1A4 4 0 0 1 7 14z',
  bell: 'M6 16V11a6 6 0 1 1 12 0v5l2 2H4zM10 21h4',
  play: 'M7 4l13 8-13 8z',
  stop: 'M6 6h12v12H6z',
  pin: 'M9 4h6l-1 6 3 3H7l3-3zM12 13v8',
  eye: 'M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12zM12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z',
  arrowR: 'M5 12h14M13 6l6 6-6 6',
  doc: 'M6 3h9l4 4v14H6zM14 3v5h5M9 13h7M9 17h7',
};

// One sprite of <symbol>s in the page; each icon is a tiny <svg><use>.
const NS = 'http://www.w3.org/2000/svg';
const sprite = document.createElementNS(NS, 'svg');
sprite.setAttribute('style', 'display:none'); sprite.setAttribute('aria-hidden', 'true');
sprite.innerHTML = Object.entries(P).map(([k, d]) => `<symbol id="i-${k}" viewBox="0 0 24 24"><path d="${d}"/></symbol>`).join('');
document.body.prepend(sprite);
export function icon(name, cls = '') {
  const s = document.createElementNS(NS, 'svg');
  s.setAttribute('class', 'ic ' + cls);
  s.setAttribute('aria-hidden', 'true');
  const u = document.createElementNS(NS, 'use');
  u.setAttribute('href', '#i-' + (P[name] ? name : 'more'));
  s.append(u);
  return s;
}

// Agent glyph: a filled tool-colour circle + a white mark (the only colour in a card header). kind = Herdr agent kind / profile kind.
const GLYPH = {
  claude: ['#d97757', '<path d="M12 4v16M4 12h16M6.3 6.3l11.4 11.4M17.7 6.3L6.3 17.7" stroke="#fff" stroke-width="2.2" stroke-linecap="round"/>'],
  codex: ['#10a37f', '<path d="M12 4l7 4v8l-7 4-7-4V8z" fill="none" stroke="#fff" stroke-width="2"/>'],
  agy: ['#5b7cfa', '<path d="M12 3c.8 4.7 4.3 8.2 9 9-4.7.8-8.2 4.3-9 9-.8-4.7-4.3-8.2-9-9 4.7-.8 8.2-4.3 9-9z" fill="#fff"/>'],
  gemini: ['#5b7cfa', '<path d="M12 3c.8 4.7 4.3 8.2 9 9-4.7.8-8.2 4.3-9 9-.8-4.7-4.3-8.2-9-9 4.7-.8 8.2-4.3 9-9z" fill="#fff"/>'],
  cline: ['#14b8a6', 'C'], grok: ['#6b7280', 'G'], cursor: ['#2a2f3a', 'Cu'], opencode: ['#a855f7', 'O'],
  shell: ['#3a4150', '<path d="M7 8l4 4-4 4M13 16h4" fill="none" stroke="#e6e9ef" stroke-width="2" stroke-linecap="round"/>'],
};
export function glyph(kind) {
  const [bg, mark] = GLYPH[kind] || (kind ? ['#4b5563', ((kind.match(/[a-z0-9]/i) || ['?'])[0]).toUpperCase()] : GLYPH.shell);
  const s = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  s.setAttribute('viewBox', '0 0 24 24'); s.setAttribute('class', 'glyph'); s.setAttribute('aria-hidden', 'true');
  const inner = mark.startsWith('<') ? mark
    : `<text x="12" y="16.5" text-anchor="middle" font-size="${mark.length > 1 ? 10 : 13}" font-weight="700" fill="#fff" font-family="Segoe UI,system-ui,sans-serif">${mark}</text>`;
  s.innerHTML = `<circle cx="12" cy="12" r="12" fill="${bg}"/>` + inner;
  return s;
}

// Context ring: pct 0-100 (null = unknown).
export function ring(pct) {
  const s = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  s.setAttribute('viewBox', '0 0 36 36'); s.setAttribute('class', 'ring'); s.setAttribute('aria-hidden', 'true');
  const v = pct == null ? 0 : Math.max(0, Math.min(100, pct));
  const hot = v >= 90 ? ' hot' : v >= 75 ? ' warm' : '';
  s.innerHTML = `<circle cx="18" cy="18" r="15" class="ring-bg"/><circle cx="18" cy="18" r="15" class="ring-fg${hot}" pathLength="100" stroke-dasharray="${v} 100" transform="rotate(-90 18 18)"/>` +
    `<text x="18" y="22" text-anchor="middle">${pct == null ? '–' : Math.round(v)}</text>`;
  return s;
}
