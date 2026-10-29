// Dialogs, drawer panels and the command palette (P0 + P1 + P2 features around the grid).
import { api, missing, why } from './api.js';
import { el, $, toast, fail, dialog, confirm, ask, copyText, ago, OrderedQueue, knownNumber, patchContent } from './ui.js';
import { fitState } from './screen.js';
import * as Q from './parity.js';
import * as G from './guard.js';

let X; // context from app.js (state + helpers); set by init()
export function init(ctx) { X = ctx; }

// ---------- look: theme, font size, density ----------
export function applyLook() {
  const L = X ? X.LS : { get: (k, d) => { try { return JSON.parse(localStorage.getItem('deck.' + k)) ?? d; } catch { return d; } } };
  const r = document.documentElement;
  // ?theme=light|dark|hc wins once and is remembered like a palette choice.
  const asked = new URLSearchParams(location.search).get('theme');
  if (['dark', 'light', 'hc'].includes(asked) && X && L.get('theme', 'dark') !== asked) X.LS.set('theme', asked);
  r.dataset.theme = ['dark', 'light', 'hc'].includes(asked) ? asked : L.get('theme', 'dark');
  r.dataset.density = L.get('density', 'normal');
  const fs = L.get('font', 13);
  r.style.setProperty('--mono-size', fs + 'px');
  r.style.setProperty('--row-h', Math.round(fs * 1.5) + 'px');
  fitState.base = fs; fitState.on = L.get('fit', true);
}
export function setLook(k, v) {
  X.LS.set(k, v); applyLook();
  for (const p of X.S.panes.values()) { const c = X.card(p.key); if (c) { c.view.fs = null; c.view.fit(); c.view.resetMetrics(); } X.schedule(p.key); }
}

// ---------- empty state ----------
export function renderEmpty(box, down) {
  box.textContent = '';
  if (down) {
    box.append(el('div', { class: 'empty-card' },
      el('h2', { text: 'Herdr is not running' }),
      el('p', { class: 'muted', text: X.S.herdr.error || 'The Deck cannot reach the Herdr session.' }),
      el('button', { class: 'primary', text: 'Start Herdr', 'data-w': '1', onclick: () => X.act('Start Herdr', () => api.herdrStart(), 'Launcher started; waiting for Herdr...') }),
      el('p', { class: 'muted small', text: 'The Deck checks again by itself and shows the panes when Herdr is up.' })));
  } else {
    box.append(el('div', { class: 'empty-card' },
      el('h2', { text: 'No panes yet' }),
      el('p', { class: 'muted', text: 'Herdr is running but has no panes in this session.' }),
      el('div', { class: 'row' }, el('button', { class: 'primary', text: '+ Agent', onclick: startAgent, 'data-w': '1' }),
        el('button', { text: '+ Shell', onclick: newShell, 'data-w': '1' }))));
  }
}

// ---------- drawer (one right-side panel at a time) ----------
let openPanel = null, opener = null;
// The button that opens a panel ([data-drawer]) shows it as pressed while the panel is open.
function markOpen(name) {
  for (const b of document.querySelectorAll('[data-drawer]')) if (b.getAttribute('role') !== 'menuitem') b.setAttribute('aria-pressed', String(b.dataset.drawer === name));
}
export function drawer(name, title, render, live = false) {
  const d = $('#drawer');
  if (openPanel?.name === name && !d.hidden) { closeDrawer(); return; }
  const a = document.activeElement;
  if (a && a !== document.body && !d.contains(a)) opener = a;   // focus goes back here on close
  drawerAbort?.abort();
  $('#drawer-body').replaceChildren();
  openPanel = { name, render, live };
  $('#drawer-title').textContent = title;
  d.hidden = false;
  markOpen(name);
  paintDrawer();
  $('#drawer-close').focus();
}
let drawerGeneration = 0, drawerAbort = null;
async function paintDrawer() {
  drawerAbort?.abort(); drawerAbort = new AbortController();
  const generation = ++drawerGeneration, panel = openPanel, signal = drawerAbort.signal;
  const body = $('#drawer-body'), stage = el('div');
  try {
    await panel.render(stage, signal);
    if (signal.aborted || generation !== drawerGeneration || openPanel !== panel) return;
    body.className = stage.className || '';
    if (panel.painted) patchContent(body, stage);
    else { body.replaceChildren(...stage.childNodes); panel.painted = true; }
  } catch (e) { if (!signal.aborted && generation === drawerGeneration) body.append(el('p', { class: 'err', text: why(e) })); }
}
export function closeDrawer() {
  const d = $('#drawer'), a = document.activeElement, hadFocus = !a || a === document.body || d.contains(a);
  drawerAbort?.abort(); drawerGeneration++; clearTimeout(paintTimer); paintTimer = 0; d.hidden = true; openPanel = null;
  markOpen(null);
  // Focus returns to the opener (a hidden menu item -> its menu button), only if it was in the panel.
  const back = opener?.closest?.('[role="menu"]') ? $('#btn-more') : opener;
  opener = null;
  if (hadFocus && back?.isConnected) back.focus();
}
// Live panels repaint at most every 500 ms (streaming panes must not rebuild a panel each frame).
let lastPaint = 0, paintTimer = 0;
export function refreshOpen(force) {
  if (!openPanel || $('#drawer').hidden || (!openPanel.live && !force) || document.hidden) return;
  const wait = force ? 0 : 500 - (performance.now() - lastPaint);
  if (wait <= 0) { clearTimeout(paintTimer); paintTimer = 0; lastPaint = performance.now(); paintDrawer(); }
  else if (!paintTimer) paintTimer = setTimeout(() => { paintTimer = 0; refreshOpen(); }, wait);
}
addEventListener('keydown', (ev) => { if (ev.key === 'Escape' && openPanel && !document.querySelector('dialog[open]') && !$('#drawer').hidden && $('#drawer').contains(document.activeElement)) closeDrawer(); });
setTimeout(() => { const b = $('#drawer-close'); if (b) b.onclick = closeDrawer; });

// ---------- P0.8 + Agent ----------
const EFFORTS = ['', 'low', 'medium', 'high', 'xhigh', 'max'];
export async function startAgent(preset) {
  const profiles = X.S.profiles;
  if (!profiles.length) return toast('No agent profiles from the backend (agents.json)', 'error');
  const cur = X.S.panes.get(X.currentKey());
  const prof = el('select', { 'aria-label': 'Profile' }, profiles.map((p) => el('option', { value: p.name, text: `${p.name} (${p.kind})` })));
  const model = el('input', { 'aria-label': 'Model', list: 'models', spellcheck: false });
  const models = el('datalist', { id: 'models' }, [...new Set(profiles.map((p) => p.model).filter(Boolean))].map((m) => el('option', { value: m })));
  const effort = el('select', { 'aria-label': 'Effort' }, EFFORTS.map((e) => el('option', { value: e, text: e || '(profile default)' })));
  const account = el('input', { 'aria-label': 'Account', list: 'accts', spellcheck: false, size: 8 });
  const accts = el('datalist', { id: 'accts' });
  const cwd = el('input', { 'aria-label': 'Folder', value: cur?.cwd || X.S.root, spellcheck: false });
  const where = el('select', { 'aria-label': 'Where' }, el('option', { value: 'split', text: 'Split right of the current pane' }), el('option', { value: 'new', text: 'New pane (Herdr decides)' }));
  const sync = () => {
    const p = profiles.find((x) => x.name === prof.value) || {};
    model.value = p.model || ''; model.placeholder = p.model ? '' : '(CLI default)';
    effort.value = EFFORTS.includes(p.effort) ? p.effort : '';
    const list = p.accounts || (p.kind === 'claude' ? ['2030', '2040', '1984'] : []);
    accts.replaceChildren(...list.map((a) => el('option', { value: a })));
    account.value = ''; account.placeholder = p.kind === 'claude' || p.kind === 'codex' ? 'default sign-in' : 'no account support';
    account.disabled = !(p.kind === 'claude' || p.kind === 'codex' || list.length);
  };
  prof.onchange = sync;
  if (preset) prof.value = preset;
  sync();
  const f = (label, input) => el('label', { class: 'field' }, el('span', { text: label }), input);
  const v = await dialog({ title: 'Start agent', body: el('div', { class: 'form2' }, f('Profile', prof), f('Model', model), models, f('Effort', effort), f('Account', account), accts, f('Folder', cwd), f('Where', where)),
    actions: [{ label: 'Cancel', value: null }, { label: 'Start', value: 'ok', kind: 'primary' }],
    check: () => (cwd.value.trim() ? '' : 'Folder is required') });
  if (v !== 'ok') return null;
  const body = { profile: prof.value, model: model.value.trim(), effort: effort.value, account: account.value, cwd: cwd.value.trim(), key: where.value === 'split' && cur ? cur.key : '' };
  return X.act('Start agent', () => api.start(body), `Starting ${body.profile}...`);
}
export async function newShell() {
  const cur = X.S.panes.get(X.currentKey());
  return X.act('New shell', () => (cur ? api.split(cur.key, cur.cwd) : api.shell(X.S.root)), 'Shell opened');
}

// ---------- P1.1 Paste to pane ----------
export async function pasteToPane(p) {
  if (!p) return toast('No pane selected', 'warn');
  const ta = el('textarea', { rows: 5, class: 'mono', 'aria-label': 'Text to send', placeholder: 'Paste here (Ctrl+V or right-click > Paste)', spellcheck: false });
  const v = await dialog({ title: `Paste to ${X.nameOf(p)}`,
    body: el('div', {}, el('p', { class: 'muted small', text: 'Sent exactly as written. "Send text" does not press Enter; you decide.' }), ta),
    actions: [{ label: 'Cancel', value: null }, { label: 'Send text', value: 'text', kind: 'primary' }, { label: 'Send + Enter', value: 'enter' }],
    check: () => (ta.value ? '' : 'Nothing to send') });
  if (!v) return;
  const text = ta.value;
  const r = await X.act('Paste to pane', () => api.text(p.key, text), `${text.length} characters sent to ${X.nameOf(p)}`);
  if (r && v === 'enter') X.act('Enter', () => api.keys(p.key, ['enter']), 'Enter sent');
}

// ---------- P1.2 Attention inbox + notifications ----------
const inboxItems = new Map(); // key -> {t, kind}
export function inboxNote(p, before, now) {
  if (now === 'blocked') inboxItems.set(p.key, { t: Date.now(), kind: 'blocked' });
  else if (before === 'working' && (now === 'idle' || now === 'done')) inboxItems.set(p.key, { t: Date.now(), kind: 'finished' });
  else if (inboxItems.get(p.key)?.kind === 'blocked') inboxItems.delete(p.key);
  else if (now === 'working') inboxItems.delete(p.key);
}
export function inboxCount() { for (const k of inboxItems.keys()) if (!X.S.panes.has(k)) inboxItems.delete(k); return inboxItems.size; }
export function inbox() {
  drawer('inbox', 'Attention inbox', (b) => {
    const list = [...inboxItems].filter(([k]) => X.S.panes.has(k)).sort((a, b2) => (a[1].kind !== b2[1].kind ? (a[1].kind === 'blocked' ? -1 : 1) : b2[1].t - a[1].t));
    const sound = X.LS.get('sound', false);
    b.append(el('div', { class: 'row' },
      el('button', { text: notifyOn() ? 'Notifications on' : 'Turn on Windows notifications', disabled: notifyOn(), onclick: askNotify }),
      el('label', {}, el('input', { type: 'checkbox', checked: sound, onchange: (e) => X.LS.set('sound', e.target.checked) }), ' sound')));
    if (!list.length) b.append(el('p', { class: 'muted', text: 'Nothing needs you. Blocked agents come first here.' }));
    b.append(el('ul', { class: 'list' }, list.map(([k, it]) => {
      const p = X.S.panes.get(k);
      return el('li', { class: 'li ' + it.kind },
        el('span', { class: 'dot', 'data-s': X.statusOf(p) }),
        el('div', { class: 'grow' }, el('strong', { text: X.nameOf(p) }), ' ', el('span', { class: 'muted small', text: `${it.kind} · ${ago(it.t)} ago` }),
          el('div', { class: 'mono small clip', text: X.lastLine(p) })),
        el('button', { text: 'Show', onclick: () => { X.setFilter(null, null); X.zoom(k); } }),
        it.kind === 'finished' && el('button', { text: 'Dismiss', onclick: () => { inboxItems.delete(k); X.schedule(); } }));
    })));
    b.append(el('p', { class: 'muted small', text: 'The Deck never answers a blocked agent. Open it and decide yourself.' }));
  }, true);
}
const notifyOn = () => 'Notification' in window && Notification.permission === 'granted';
function askNotify() { if ('Notification' in window) Notification.requestPermission().then(() => X.schedule()); else toast('Notifications are not supported here', 'warn'); }
export function notify(title, body) {
  if (X?.LS.get('sound', false)) beep();
  if (notifyOn() && (document.hidden || !document.hasFocus())) { try { new Notification(title, { body, tag: title }); } catch (e) { console.warn('notification failed', e); } }
}
let actx = null;
function beep() {
  try {
    actx ||= new AudioContext();
    const o = actx.createOscillator(), g = actx.createGain();
    o.frequency.value = 660; g.gain.value = 0.05; o.connect(g); g.connect(actx.destination); o.start(); o.stop(actx.currentTime + 0.15);
  } catch (e) { console.warn('sound failed', e); }
}

// ---------- P1.3 Task launcher ----------
export function tasks() {
  drawer('tasks', 'Tasks', async (b, signal) => {
    const { tasks: list = [] } = await api.tasks({ signal });
    const prev = el('pre', { class: 'preview mono small', text: 'Pick a task to preview it.' });
    let file = '';
    const send = el('button', { class: 'primary', text: 'Launch… (readiness check)', disabled: true, 'data-w': '1', onclick: () => G.launch(file) });
    const fresh = el('button', { text: 'Citation audit', disabled: true, onclick: () => { prev.replaceWith(G.auditBlock(prev.textContent)); } });
    b.append(el('ul', { class: 'list' }, list.map((t) => el('li', {}, el('button', { class: 'linkish', text: t.title || t.file, onclick: async () => {
      file = t.file; send.disabled = fresh.disabled = false;
      prev.textContent = 'Loading...';
      try { prev.textContent = (await api.taskRead(t.file)).text; } catch (e) { prev.textContent = why(e); }
    } }), el('span', { class: 'muted small', text: ' ' + (t.mtime || '') })))),
    el('div', { class: 'row' }, send, fresh), prev);
  });
}

// ---------- P1.4 Broadcast / compare ----------
export async function broadcast(ta) {
  const keys = X.broadcastKeys(), text = ta.value;
  if (!keys.length || !text.trim()) return toast('Select panes (checkbox or X) and write a prompt', 'warn');
  const names = keys.map((k) => X.nameOf(X.S.panes.get(k))).join(', ');
  if (!(await confirm('Broadcast prompt?', `To ${keys.length} panes: ${names}\n\n${text}`, 'Send to all', false))) return;
  const res = await Promise.all(keys.map((k) => G.sendGuarded(k, text)));
  if (res.every(Boolean)) ta.value = '';
}
export function compare() {
  const keys = X.broadcastKeys();
  if (keys.length < 2) return toast('Select at least two panes to compare', 'warn');
  drawer('compare', 'Compare answers', (b) => {
    b.classList.add('wide');
    b.append(el('div', { class: 'compare' }, keys.filter((k) => X.S.panes.has(k)).map((k) => {
      const p = X.S.panes.get(k), c = X.card(k);
      return el('section', {}, el('h3', {}, el('span', { class: 'dot', 'data-s': X.statusOf(p) }), ' ', X.nameOf(p)),
        el('pre', { class: 'mono small preview', text: (c ? c.view.plain : '').split('\n').slice(-80).join('\n') }));
    })));
  }, true);
}

// ---------- P1.5 Usage ----------
const bar = (label, pct) => el('div', { class: 'bar-row' }, el('span', { class: 'bl', text: label }),
  el('span', { class: 'bar', role: 'meter', 'aria-valuenow': pct, 'aria-valuemin': 0, 'aria-valuemax': 100, 'aria-label': label },
    el('span', { style: `width:${Math.max(0, Math.min(100, pct))}%`, 'data-hot': pct >= 80 ? '1' : null })), el('span', { class: 'bv', text: pct + '%' }));
export function usage() {
  drawer('usage', 'Usage and limits', async (b, signal) => {
    b.append(el('p', { class: 'muted', text: 'Loading (herdr-agents usage can take a few seconds)...' }));
    const u = await api.usage({ signal });
    b.textContent = '';
    if (!u.accounts && (u.output || u.text)) u.accounts = parseUsage(u.output || u.text);
    u.text = u.text || u.output;
    for (const a of u.accounts || []) b.append(el('h3', { text: 'Account ' + a.name + (a.main ? ' (main sign-in)' : '') }), a.five_hour != null ? bar('5 hours', a.five_hour) : '', a.weekly != null ? bar('weekly', a.weekly) : '');
    if (!(u.accounts || []).length) b.append(el('p', { class: 'muted small', text: 'No percentages found in the output; see the raw text.' }));
    if (u.codex) b.append(el('h3', { text: 'Codex' }), bar('5 hours', u.codex.five_hour ?? 0), bar('weekly', u.codex.weekly ?? 0));
    const tok = [...X.S.panes.values()].filter((p) => p.tokens && Object.keys(p.tokens).length);
    if (tok.length) b.append(el('h3', { text: 'Tokens per pane' }), el('ul', { class: 'list' }, tok.map((p) => el('li', {}, el('strong', { text: X.nameOf(p) }), ' ',
      el('span', { class: 'mono small', text: Object.entries(p.tokens).map(([k, v]) => k + ' ' + v).join(' · ') })))));
    if (u.text) b.append(el('details', {}, el('summary', { text: 'Raw output' }), el('pre', { class: 'mono small preview', text: u.text })));
    const top = (u.accounts || []).map((a) => a.five_hour ?? 0);
    if (top.length) $('#usage-n').textContent = Math.max(...top) + '%';
  });
}

// "herdr-agents usage" prints a table: `2030*  16%  Mon 11:50  11%  Fri 13:00 ...` (account, 5h %,
// reset, weekly %, ...; * = main sign-in). Older output used "5h N%" / "week N%" words; both are read.
const USAGE_ROW = /^(\S+?)\*?\s+(\d{1,3})%\s+.+?\s(\d{1,3})%\s/;
export function parseUsage(text) {
  const out = [];
  for (const line of text.split('\n')) {
    const row = USAGE_ROW.exec(line);
    if (row) { out.push({ name: row[1], five_hour: +row[2], weekly: +row[3], main: /^\S+\*/.test(line), line: line.trim() }); continue; }
    const five = /5\s*-?h(?:our)?s?\D{0,12}?(\d{1,3})\s*%/i.exec(line), week = /week(?:ly)?\D{0,12}?(\d{1,3})\s*%/i.exec(line);
    if (!five && !week) continue;
    const name = (/\b(\d{4})\b/.exec(line) || /^\s*([\w.-]+)/.exec(line) || ['', '?'])[1];
    out.push({ name, five_hour: five ? +five[1] : null, weekly: week ? +week[1] : null, line: line.trim() });
  }
  return out;
}

// ---------- P1.6 Health (rail widget, polled only while visible) ----------
let healthTimer = 0, healthMissing = false;
async function health() {
  const box = $('#health');
  try {
    const h = await api.health();
    G.setHealth(h);
    box.textContent = '';
    const idle = [...X.S.panes.values()].filter((p) => ['idle', 'done'].includes(X.statusOf(p)) && (p.name || p.agent));
    box.append(el('h3', { text: 'Health' }), h.ram_used_pct == null ? el('div', { text: 'RAM unknown' }) : bar('RAM', h.ram_used_pct),
      ...(h.disks || []).map((d) => el('div', { class: 'small', text: d.free_gb == null ? `${d.drive} unknown` : `${d.drive} ${knownNumber(d.free_gb)} GB free` })),
      el('div', { class: 'small hp' }),
      idle.length ? el('button', { class: 'small-btn', text: 'Close finished panes', 'data-w': '1', onclick: () => closeFinished(idle) }) : '');
    X.schedule();
  } catch (e) {
    box.textContent = '';
    box.append(el('h3', { text: 'Health' }), el('div', { class: 'small muted', text: missing(e) ? 'not available (backend endpoint missing)' : e.message }));
    if (missing(e)) healthMissing = true;
  }
}
async function closeFinished(idle) {
  const text = idle.map((p) => `${p.pane_id}  ${X.nameOf(p)}  (${X.statusOf(p)})`).join('\n');
  if (!(await confirm('Close finished panes?', `These ${idle.length} panes close (their agents stop):\n\n${text}`, `Close ${idle.length} panes`))) return;
  const keys = idle.map((p) => p.key);
  X.act('Close finished panes', () => api.closeFinished(keys).catch((e) => {
    if (!missing(e)) throw e;
    return Promise.all(keys.map((k) => api.close(k))).then(() => ({ message: `Closed ${keys.length} panes` }));
  }));
}
export function pause() { clearInterval(healthTimer); healthTimer = 0; }
export function resume() { if (!healthTimer && !document.hidden) { health(); healthTimer = setInterval(() => !healthMissing && !document.hidden && health(), 10000); } }

// ---------- P1.7 Team presets ----------
const TEAMS = [
  { name: 'Claude + Codex', profiles: ['claude', 'codex'] },
  { name: 'Claude + Codex + Gemini', profiles: ['claude', 'codex', 'gemini'] },
  { name: 'Codex + Gemini (no Claude)', profiles: ['codex', 'gemini'] },
];
export async function team() {
  const have = new Set(X.S.profiles.map((p) => p.name));
  const pick = el('div', { class: 'list' }, TEAMS.map((t, i) => el('label', { class: 'li' }, el('input', { type: 'radio', name: 'team', value: i, checked: i === 0 }),
    ` ${t.name}`, el('span', { class: 'muted small', text: ' - ' + t.profiles.map((p) => (have.has(p) ? p : p + ' (missing)')).join(', ') }))));
  const cwd = el('input', { 'aria-label': 'Folder', value: X.S.panes.get(X.currentKey())?.cwd || X.S.root });
  const v = await dialog({ title: 'Start a team', body: el('div', {}, pick, el('label', { class: 'field' }, el('span', { text: 'Folder' }), cwd),
    el('p', { class: 'muted small', text: 'Each agent starts with its profile defaults, one after another. Research teams run headless outside the Deck (owner rule).' })),
    actions: [{ label: 'Cancel', value: null }, { label: 'Start team', value: 'ok', kind: 'primary' }] });
  if (v !== 'ok') return;
  const t = TEAMS[+pick.querySelector('input:checked').value];
  for (const prof of t.profiles) {
    if (!have.has(prof)) { toast(`Profile ${prof} is not in agents.json; skipped`, 'warn'); continue; }
    const cur = X.currentKey();
    const r = await X.act('Start ' + prof, () => api.start({ profile: prof, model: '', effort: '', account: '', cwd: cwd.value.trim(), key: cur || '' }), `Starting ${prof}...`);
    if (!r) break;
  }
}

// ---------- P1.8 Sign-in helper ----------
export function signin() {
  drawer('signin', 'Sign-in helper', async (b, signal) => {
    const { tools = [] } = await api.signin({ signal });
    b.append(el('p', { class: 'muted small', text: 'The Deck never types or stores secrets. "Open login pane" starts the tool\'s own login in a pane; you finish it there (use Paste to pane for codes).' }));
    b.append(el('ul', { class: 'list' }, tools.map((t) => el('li', { class: 'li' },
      el('span', { class: 'dot', 'data-s': t.signed_in === true ? 'done' : t.signed_in === false ? 'blocked' : 'unknown' }),
      el('div', { class: 'grow' }, el('strong', { text: t.tool + (t.account ? ' ' + t.account : '') }), ' ',
        el('span', { class: 'muted small', text: (t.signed_in === true ? 'signed in' : t.signed_in === false ? 'NOT signed in' : 'sign-in state unknown') + (t.detail ? ' - ' + t.detail : '') })),
      el('button', { text: 'Open login pane', 'data-w': '1', onclick: () => X.act('Open login pane', () => api.signinOpen(t.tool, t.account || '')) })))));
  });
}

// ---------- P2 panels ----------
export function changes(p) {
  drawer('changes', 'Changes - ' + X.nameOf(p), async (b, signal) => {
    const r = await api.git(p.key, { signal });
    b.append(el('p', { class: 'muted small', text: p.cwd }), el('pre', { class: 'mono small preview', text: r.text || '(no changes)' }));
  });
}
function textViewer(name, title, listFn, readFn) {
  drawer(name, title, async (b, signal) => {
    const search = el('input', { type: 'search', placeholder: 'Search', 'aria-label': 'Search' });
    const out = el('pre', { class: 'mono small preview' });
    let text = '';
    const show = () => {
      const q = search.value.trim().toLowerCase();
      out.textContent = !q ? text : text.split('\n').filter((l) => l.toLowerCase().includes(q)).join('\n') || '(no match)';
    };
    search.oninput = show;
    const items = await listFn();
    const auditBtn = el('button', { text: 'Citation audit', onclick: () => { const t = out.textContent; out.replaceChildren(G.auditBlock(text || t)); } });
    b.append(el('div', { class: 'row' }, auditBtn, items.map((it) => el('button', { text: it.label, onclick: async () => {
      out.textContent = 'Loading...';
      try { text = await readFn(it.id); show(); } catch (e) { out.textContent = why(e); }
    } }))), search, out);
  });
}
export function runs() {
  textViewer('runs', 'Run history', async () => ((await api.runs()).runs || []).map((r) => ({ label: r.title || r.file, id: r.file })),
    async (f) => (await api.runRead(f)).text);
}
export function docs() {
  textViewer('docs', 'Docs hub', async () => ['HISTORY', 'LESSONS', 'AGENT-ROSTER', 'TEAM-PLAYBOOK'].map((n) => ({ label: n, id: n })),
    async (n) => (await api.docs(n)).text);
}
export async function addLesson() {
  const ta = el('textarea', { rows: 5, 'aria-label': 'Lesson text', placeholder: 'One durable fact: what, why, how to apply.' });
  const v = await dialog({ title: 'Add lesson (append-only to LESSONS.md)', body: ta,
    actions: [{ label: 'Cancel', value: null }, { label: 'Append', value: 'ok', kind: 'primary' }], check: () => (ta.value.trim().length > 10 ? '' : 'Write the lesson first') });
  if (v === 'ok') X.act('Add lesson', () => api.lesson(ta.value.trim()), 'Lesson appended');
}
export function check() {
  drawer('check', 'Check runner', (b) => {
    const out = el('div', { class: 'mono small' });
    const run = el('button', { class: 'primary', text: 'Run check.ps1', onclick: async () => {
      run.disabled = true; out.textContent = 'Running...';
      try {
        const r = await api.check();
        out.textContent = '';
        for (const l of r.lines || []) out.append(el('div', { class: 'chk ' + (/^\s*FAIL/.test(l) ? 'fail' : /^\s*WARN/.test(l) ? 'warn' : /^\s*OK/.test(l) ? 'ok' : ''), text: l }));
      } catch (e) { out.textContent = why(e); }
      run.disabled = false;
    } });
    b.append(run, out);
  });
}
export function timeline() {
  drawer('timeline', 'Activity today (seen by this window)', (b) => {
    const t = X.S.timeline.slice().reverse();
    if (!t.length) b.append(el('p', { class: 'muted', text: 'No events yet. Started / blocked / finished / closed appear here.' }));
    b.append(el('ul', { class: 'list' }, t.map((e) => el('li', {}, el('span', { class: 'mono small', text: new Date(e.t).toLocaleTimeString() + '  ' }),
      el('strong', { text: e.name }), ' ' + e.what))));
  }, true);
}
export function exportMd(p, text) {
  const md = `# ${X.nameOf(p)} (${p.pane_id})\n\n- folder: ${p.cwd}\n- status: ${X.statusOf(p)}\n- saved: ${new Date().toISOString()}\n\n\`\`\`text\n${text}\n\`\`\`\n`;
  const a = el('a', { href: URL.createObjectURL(new Blob([md], { type: 'text/markdown' })), download: `${X.nameOf(p).replace(/[^\w.-]+/g, '_')}-screen.md` });
  document.body.append(a); a.click(); a.remove();
  setTimeout(() => URL.revokeObjectURL(a.href), 5000);
  toast('Screen saved as Markdown (browser download)');
}
export const asContext = (p, text) => `Screen of ${X.nameOf(p)} (${p.pane_id}, ${p.cwd}):\n\`\`\`text\n${text.split('\n').slice(-200).join('\n')}\n\`\`\`\n`;

// ---------- P2.13 layouts ----------
async function saveLayout() {
  const name = await ask('Save layout', 'Layout name', 'my layout');
  if (!name) return;
  const all = X.LS.get('layouts', {});
  all[name] = { cols: X.S.cols, filter: X.S.filter, hidden: [...X.S.hidden], pinned: [...X.S.pinned] };
  X.LS.set('layouts', all); toast('Layout saved: ' + name);
}
function loadLayout(name) {
  const l = X.LS.get('layouts', {})[name];
  if (!l) return;
  X.S.hidden = new Set(l.hidden); X.S.pinned = new Set(l.pinned);
  X.LS.set('hiddenPanes', l.hidden); X.LS.set('pinned', l.pinned);
  X.setCols(l.cols); X.setFilter(l.filter.ws, l.filter.tab); toast('Layout loaded: ' + name);
}

// ---------- P2.15 prompt library ----------
async function saveSnippet() {
  const c = X.card(X.currentKey());
  const ta = el('textarea', { rows: 4, 'aria-label': 'Snippet text', value: c?.ta.value || '' });
  const nm = el('input', { 'aria-label': 'Name' });
  const v = await dialog({ title: 'Save prompt snippet', body: el('div', {}, el('label', { class: 'field' }, el('span', { text: 'Name' }), nm),
    el('label', { class: 'field' }, el('span', { text: 'Text ({name} = variable)' }), ta)),
    actions: [{ label: 'Cancel', value: null }, { label: 'Save', value: 'ok', kind: 'primary' }], check: () => (nm.value.trim() && ta.value.trim() ? '' : 'Name and text are required') });
  if (v !== 'ok') return;
  const all = X.LS.get('snippets', {}); all[nm.value.trim()] = ta.value; X.LS.set('snippets', all); toast('Snippet saved');
}
async function useSnippet(name) {
  let text = X.LS.get('snippets', {})[name] || '';
  for (const v of [...new Set([...text.matchAll(/\{(\w+)\}/g)].map((m) => m[1]))]) {
    const val = await ask('Snippet variable', v);
    if (val == null) return;
    text = text.split('{' + v + '}').join(val);
  }
  const c = X.card(X.currentKey());
  if (!c) return copyText(text);
  c.ta.value = text; c.ta.dispatchEvent(new Event('input')); c.ta.focus();
}

// ---------- P2.16 raw terminal mode ----------
const rawQueues = new OrderedQueue((key, text, keys) => api.raw(key, text, keys),
  (key) => !X.S.safe && !X.S.writeBlocked && X.S.panes.has(key) && !X.S.panes.get(key)?.stale);
let rawBuf = '', rawTimer = 0, rawTarget = null, rawSwitch = Promise.resolve();
export function clearRaw() { clearTimeout(rawTimer); rawBuf = ''; rawQueues.clear(); if (X?.S.raw) { X.card(X.S.raw)?.root.classList.remove('raw'); X.S.raw = null; $('#rawbar').hidden = true; } }
export function forgetPane(key) { inboxItems.delete(key); rawQueues.forget(key); if (X.S.raw === key) clearRaw(); }
export function rawMode(c) {
  rawSwitch = rawSwitch.then(async () => {
    const old = X.S.raw;
    await endRaw();
    if (old === c.key || X.S.safe || X.S.writeBlocked || X.S.panes.get(c.key)?.stale) return;
    X.S.raw = c.key; rawTarget = c.key;
    $('#rawbar').hidden = false;
    $('#rawbar-text').textContent = `You are typing into ${X.nameOf(X.S.panes.get(c.key))} - every key goes to the pane. Leave: Ctrl+Shift+X or the button.`;
    $('#rawbar-stop').onclick = () => endRaw().catch(fail('Typing'));
    c.root.classList.add('raw'); c.screenBox.focus();
  }).catch(fail('Typing'));
  return rawSwitch;
}
async function endRaw() {
  const key = X.S.raw;
  X.card(key)?.root.classList.remove('raw');
  X.S.raw = null; $('#rawbar').hidden = true;
  await flushRaw(); if (key) await rawQueues.drain(key);
  rawTarget = null;
}
function flushRaw(keys = []) {
  clearTimeout(rawTimer);
  const text = rawBuf, key = rawTarget; rawBuf = '';
  if (!key || (!text && !keys.length)) return Promise.resolve();
  return rawQueues.push(key, text, keys);
}
const RAW = { Enter: 'enter', Escape: 'esc', Tab: 'tab', Backspace: 'backspace', ArrowUp: 'up', ArrowDown: 'down', ArrowLeft: 'left', ArrowRight: 'right', ' ': 'space' };
export function rawKey(ev, c) {
  if (X.S.raw !== c.key || ev.isComposing || ev.keyCode === 229) return;
  if (ev.ctrlKey && ev.shiftKey && /x/i.test(ev.key)) { ev.preventDefault(); endRaw().catch(fail('Typing')); return; }
  if ((ev.ctrlKey || ev.metaKey) && ev.key.toLowerCase() === 'v') return;
  let k = null;
  if (ev.ctrlKey && !ev.altKey && /^[a-z]$/i.test(ev.key)) k = 'ctrl+' + ev.key.toLowerCase();
  else if (ev.shiftKey && ev.key === 'Tab') k = 'shift+tab';
  else if (/^F([1-9]|1[0-2])$/.test(ev.key)) k = ev.key.toLowerCase();
  else if (RAW[ev.key] && !(ev.key === ' ' && rawBuf)) k = RAW[ev.key];
  ev.preventDefault();
  if (k) { flushRaw([k]).catch(fail('Typing')); return; }
  if ([...ev.key].length === 1 && !ev.ctrlKey && !ev.metaKey) { rawBuf += ev.key; clearTimeout(rawTimer); rawTimer = setTimeout(() => flushRaw().catch(fail('Typing')), 60); }
}
addEventListener('compositionend', (ev) => {
  if (!X?.S.raw || !X.card(X.S.raw)?.screenBox.contains(ev.target)) return;
  rawBuf += ev.data || ''; flushRaw().catch(fail('Typing'));
});
addEventListener('paste', (ev) => {
  if (!X?.S.raw || !X.card(X.S.raw)?.screenBox.contains(document.activeElement)) return;
  ev.preventDefault(); rawBuf += ev.clipboardData.getData('text'); flushRaw().catch(fail('Typing'));
});

// ---------- P0.11 command palette ----------
function commands() {
  const S = X.S, out = [];
  const add = (label, run, hint = '') => out.push({ label, run, hint });
  for (const p of S.panes.values()) {
    const n = X.nameOf(p);
    add(`Focus ${n}`, () => { X.setFilter(null, null); X.zoom(p.key); }, p.pane_id);
    add(`Ask ${n}`, () => { X.setFilter(null, null); X.schedule(); requestAnimationFrame(() => { const c = X.card(p.key); c?.ta.focus(); }); }, 'prompt box');
    if (p.name) add(`Stop ${n}`, () => X.stopAgent(p), 'confirm');
    add(`Close pane ${n}`, () => X.closePane(p.key), 'confirm');
    add(`Paste to ${n}`, () => pasteToPane(p), 'send text');
  }
  add('+ Agent', () => startAgent(), 'start an agent');
  for (const pr of S.profiles) add(`+ Agent ${pr.name}`, () => startAgent(pr.name), pr.model);
  add('+ Shell', newShell); add('Team presets', team); add('Attention inbox', inbox, 'I');
  add('Tasks', tasks, 'task launcher'); add('Launch a task (readiness check)…', () => G.launch(), 'guarded'); add('Launch queue and limits', G.queuePanel);
  for (const p of S.panes.values()) if (p.name || p.agent) add(`Proof checklist ${X.nameOf(p)}`, () => G.proof(p), 'evidence'); add('Usage and limits', usage); add('Sign-in helper', signin);
  add('Docs hub', docs, 'HISTORY, LESSONS...'); add('Open docs: LESSONS', docs); add('Add lesson', addLesson, 'append-only');
  add('Run history', runs); add('Check runner (check.ps1)', check); add('Activity timeline', timeline);
  add('Reload Herdr config', () => X.act('Reload config', () => api.reload(), 'Herdr config reloaded'), 'herdr server reload-config');
  add('Select all visible panes', () => { for (const p of X.filteredPanes()) S.selected.add(p.key); X.schedule(); for (const p of X.filteredPanes()) X.schedule(p.key); }, 'broadcast');
  add('Compare selected panes', compare);
  for (const n of [1, 2, 3]) add(`${n} column${n > 1 ? 's' : ''}`, () => X.setCols(n), String(n));
  add('Theme: dark', () => setLook('theme', 'dark')); add('Theme: light', () => setLook('theme', 'light')); add('Theme: high contrast', () => setLook('theme', 'hc'));
  add('Font bigger', () => setLook('font', Math.min(22, X.LS.get('font', 13) + 1))); add('Font smaller', () => setLook('font', Math.max(10, X.LS.get('font', 13) - 1)));
  add(X.LS.get('fit', true) ? 'Screen: fit to width OFF (fixed font)' : 'Screen: fit to width ON', () => setLook('fit', !X.LS.get('fit', true)));
  add('Density: compact', () => setLook('density', 'compact')); add('Density: normal', () => setLook('density', 'normal'));
  add(S.safe ? 'Safe mode: off (allow writes)' : 'Safe mode: on (read-only)', () => X.setSafe(!S.safe));
  add('Save layout', saveLayout);
  for (const n of Object.keys(X.LS.get('layouts', {}))) add(`Load layout ${n}`, () => loadLayout(n));
  add('Show hidden panes', () => { S.hidden.clear(); X.LS.set('hiddenPanes', []); X.schedule(); });
  add('Save prompt box as snippet', saveSnippet);
  for (const n of Object.keys(X.LS.get('snippets', {}))) add(`Snippet: ${n}`, () => useSnippet(n), 'insert into prompt');
  for (const [k, label, fn] of Q.CHORDS) if (fn && !/^Arrow|^[hjklHJKL]$/.test(k)) add(label, fn, 'Ctrl+B ' + k);
  for (const w of S.workspaces) add(`Space ${w.label || w.workspace_id}`, () => X.setFilter(w.workspace_id, null), 'go to space');
  for (const t of S.tabs) add(`Focus tab ${t.label || t.tab_id}`, () => X.setFilter(t.workspace_id || t.parent_workspace, t.tab_id), 'tab');
  add('View: Herdr layout (real splits)', () => X.setView('layout'), 'L'); add('View: grid', () => X.setView('grid'), 'L');
  add('Pipeline (writer, reviewer, gate)', Q.pipeline); add('Search all screens', () => Q.search(), 'Ctrl+Shift+F');
  add('Find errors in all screens', () => Q.search('/error|failed|exception|panic|traceback|denied/'));
  add('Keyboard shortcuts', () => Q.keysOverlay(), '?');
  return out;
}
function score(q, s) {
  s = s.toLowerCase(); let i = 0, sc = 0, last = -2;
  for (const ch of q) { const j = s.indexOf(ch, i); if (j < 0) return -1; sc += j === last + 1 ? 3 : 1; if (j === 0 || s[j - 1] === ' ') sc += 2; last = j; i = j + 1; }
  return sc - s.length / 100;
}
export function palette(prefix = '') {
  if (document.querySelector('dialog.pal')) return;
  const all = commands();
  const inp = el('input', { role: 'combobox', 'aria-expanded': 'true', 'aria-controls': 'pal-list', 'aria-label': 'Command', placeholder: 'Type a command - Enter runs, Esc closes', spellcheck: false });
  const ul = el('ul', { id: 'pal-list', role: 'listbox', class: 'pal-list' });
  let shown = [], sel = 0;
  const d = el('dialog', { class: 'dlg pal', 'aria-label': 'Command palette' }, inp, ul);
  const paint = () => {
    const q = inp.value.trim().toLowerCase();
    shown = q ? all.map((c) => [score(q, c.label), c]).filter((x) => x[0] >= 0).sort((a, b) => b[0] - a[0]).map((x) => x[1]) : all;
    shown = shown.slice(0, 60); sel = Math.min(sel, Math.max(0, shown.length - 1));
    ul.textContent = '';
    shown.forEach((c, i) => ul.append(el('li', { role: 'option', id: 'pal-' + i, 'aria-selected': i === sel ? 'true' : 'false', class: i === sel ? 'on' : '',
      onmousedown: (e) => { e.preventDefault(); run(i); } }, el('span', { text: c.label }), el('span', { class: 'muted small', text: c.hint }))));
    inp.setAttribute('aria-activedescendant', shown.length ? 'pal-' + sel : '');
    ul.children[sel]?.scrollIntoView({ block: 'nearest' });
  };
  const run = (i) => { const c = shown[i]; d.close(); if (c) c.run(); };
  inp.addEventListener('input', () => { sel = 0; paint(); });
  inp.addEventListener('keydown', (e) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); sel = Math.min(shown.length - 1, sel + 1); paint(); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); sel = Math.max(0, sel - 1); paint(); }
    else if (e.key === 'Enter') { e.preventDefault(); run(sel); }
  });
  d.addEventListener('close', () => d.remove());
  inp.value = prefix;
  document.body.append(d); d.showModal(); paint(); inp.focus();
}
