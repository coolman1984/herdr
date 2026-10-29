import { upsertJob } from './ui.js';
// Parity with today's Herdr setup (tasks/herdr-deck-parity.md A) + the "better" features (B):
// Ctrl+B chords, folder picker, columns, layout builder, board, spaces, stop-all, agents hub,
// tools, Herdr split layout + drag resize/swap, global search, composer chips, done strip.
import { api, missing, why } from './api.js';
import { el, $, toast, fail, dialog, confirm, confirmTyped, ask, ago } from './ui.js';
import { icon } from './icons.js';
import * as P from './panels.js';
import * as G from './guard.js';

let X;
export function init(ctx) {
  X = ctx;
  const grid = $('#grid');
  grid.addEventListener('dragover', (ev) => {
    if (!ev.dataTransfer.types.includes('text/deck-pane')) return;
    const c = ev.target.closest('.card'); if (!c) return;
    ev.preventDefault(); ev.dataTransfer.dropEffect = 'move';
    if (dropOn !== c) { dropOn?.classList.remove('drop'); dropOn = c; c.classList.add('drop'); }
  });
  grid.addEventListener('dragleave', (ev) => { if (dropOn && !dropOn.contains(ev.relatedTarget)) { dropOn.classList.remove('drop'); dropOn = null; } });
  grid.addEventListener('drop', (ev) => {
    const from = ev.dataTransfer.getData('text/deck-pane'), c = ev.target.closest('.card');
    dropOn?.classList.remove('drop'); dropOn = null;
    if (!from || !c || c.dataset.key === from) return;
    ev.preventDefault();
    X.act('Swap panes', () => api.swap(from, c.dataset.key), 'Panes swapped');
  });
}
let dropOn = null;
const cur = () => X.S.panes.get(X.currentKey());
const field = (label, input) => el('label', { class: 'field' }, el('span', { text: label }), input);

// ---------- Ctrl+B chords (same keys as the terminal) ----------
export const CHORDS = [
  ['f', 'Open folder -> new space', () => folderPicker('')],
  ['F', 'Open folder + Claude', () => folderPicker('claude')],
  ['t', 'Open folder + team (Claude left, Codex right)', () => folderPicker('team')],
  ['V', 'Split into N equal columns', () => columns()],
  ['B', 'Layout builder', () => layoutBuilder()],
  ['a', 'Agent board (blocked first)', () => board()],
  ['S', 'Spaces side by side / separate', () => spaces()],
  ['i', 'Shortcuts + guide', () => help()],
  ['Q', 'Stop everything (typed confirm)', () => stopAll()],
  ['C', 'Claude on the least-used account', () => nextClaude()],
  ['v', 'Split right', () => split(cur(), 'right')],
  ['-', 'Split down', () => split(cur(), 'down')],
  ['x', 'Close pane', () => cur() && X.closePane(cur().key)],
  ['z', 'Zoom pane', () => cur() && X.zoom(cur().key)],
  ['r', 'Resize mode (arrows, Esc)', () => resizeMode()],
  ['h', 'Focus pane left', () => focusDir('left')], ['j', 'Focus pane down', () => focusDir('down')],
  ['k', 'Focus pane up', () => focusDir('up')], ['l', 'Focus pane right', () => focusDir('right')],
  ['ArrowLeft', 'Focus pane left', () => focusDir('left')], ['ArrowDown', 'Focus pane down', () => focusDir('down')],
  ['ArrowUp', 'Focus pane up', () => focusDir('up')], ['ArrowRight', 'Focus pane right', () => focusDir('right')],
  ['H', 'Swap with left neighbour', () => swapDir('left')], ['J', 'Swap with lower neighbour', () => swapDir('down')],
  ['K', 'Swap with upper neighbour', () => swapDir('up')], ['L', 'Swap with right neighbour', () => swapDir('right')],
  ['Tab', 'Next pane', () => X.moveCard(1)], ['Shift+Tab', 'Previous pane', () => X.moveCard(-1)],
  ['c', 'New tab', () => newTab()], ['n', 'Next tab / space', () => X.railStep(1)], ['p', 'Previous tab / space', () => X.railStep(-1)],
  ['1-9', 'Go to tab 1..9', null],
  ['T', 'Rename tab', () => renameTab()], ['X', 'Close tab', () => closeTab()],
  ['N', 'New space (folder picker)', () => folderPicker('')], ['W', 'Rename space', () => renameSpace()], ['D', 'Close space', () => closeSpace()],
  ['w', 'Space picker', () => P.palette('space ')], ['g', 'Go to any space, tab or agent', () => P.palette('focus ')],
  ['o', 'Jump to the agent that just notified', () => jumpNotified()],
  ['P', 'Rename pane', () => cur() && X.renamePane(cur().key)],
  ['[', 'Copy mode: select text in the screen', () => copyMode()], ['e', 'Open scrollback as Markdown', () => cur() && P.exportMd(cur(), X.plainOf(cur().key))],
  ['b', 'Hide / show the sidebar', () => X.toggleRail()], ['s', 'Settings (look)', () => settings()],
  ['R', 'Reload Herdr config', () => X.act('Reload config', () => api.reload(), 'Herdr config reloaded')],
  ['q', 'Detach: close this window (Herdr keeps running)', () => detach()],
  ['m', 'Agents hub (herdr-agents commands)', () => agentsHub()], ['.', 'Tools (check, headless, trial, sync)', () => tools()],
  ['/', 'Search all screens', () => search()], ['?', 'Keys overlay', () => keysOverlay()],
];
const CHORD_MAP = new Map(CHORDS.filter((c) => c[2]).map((c) => [c[0], c[2]]));
let chord = false, chordTimer = 0, resizing = null;
function chordHint(text) {
  const h = $('#chord');
  h.hidden = !text; h.textContent = text || '';
}
export function chordKey(ev) {
  if (resizing) return resizeKey(ev);
  if (ev.ctrlKey && !ev.shiftKey && !ev.altKey && (ev.key === 'b' || ev.key === 'B')) {
    ev.preventDefault(); chord = true; chordHint('Ctrl+B …  (? lists the keys, Esc cancels)');
    clearTimeout(chordTimer); chordTimer = setTimeout(() => { chord = false; chordHint(''); }, 2500);
    return true;
  }
  if (!chord || ['Shift', 'Control', 'Alt', 'Meta'].includes(ev.key)) return false;
  ev.preventDefault(); chord = false; clearTimeout(chordTimer); chordHint('');
  if (ev.key === 'Escape') return true;
  const k = ev.key === 'Tab' && ev.shiftKey ? 'Shift+Tab' : ev.key;
  if (/^[1-9]$/.test(k)) { goTab(+k); return true; }
  const fn = CHORD_MAP.get(k);
  if (fn) fn(); else toast(`Ctrl+B ${k} is not bound. Ctrl+B ? lists the keys.`, 'warn');
  return true;
}

// ---------- pane geometry (Herdr layout rects when known, else grid order) ----------
function rectsOfTab(tab) {
  const lay = X.S.layouts.find((l) => l.tab_id === tab);
  return lay ? new Map(lay.panes.map((p) => [X.S.session + '|' + p.pane_id, p.rect])) : null;
}
function neighbour(key, dir) {
  const p = X.S.panes.get(key); if (!p) return null;
  const rects = rectsOfTab(p.tab_id);
  if (rects && rects.get(key)) {
    const a = rects.get(key); let best = null, bestScore = -1;
    for (const [k, b] of rects) {
      if (k === key) continue;
      const touch = dir === 'right' ? b.x >= a.x + a.width - 1 && b.x <= a.x + a.width + 1 : dir === 'left' ? b.x + b.width >= a.x - 1 && b.x + b.width <= a.x + 1
        : dir === 'down' ? b.y >= a.y + a.height - 1 && b.y <= a.y + a.height + 1 : b.y + b.height >= a.y - 1 && b.y + b.height <= a.y + 1;
      if (!touch) continue;
      const ov = dir === 'left' || dir === 'right' ? Math.min(a.y + a.height, b.y + b.height) - Math.max(a.y, b.y) : Math.min(a.x + a.width, b.x + b.width) - Math.max(a.x, b.x);
      if (ov > bestScore) { bestScore = ov; best = k; }
    }
    if (best) return best;
  }
  const list = X.visiblePanes(), i = list.findIndex((q) => q.key === key);
  const j = i + (dir === 'right' || dir === 'down' ? 1 : -1);
  return list[j]?.key || null;
}
function focusDir(dir) {
  const k = X.currentKey(); const n = k && neighbour(k, dir);
  if (!n) return toast('No pane in that direction', 'warn');
  X.S.current = n; if (X.S.zoom) X.zoom(n); else requestAnimationFrame(() => X.card(n)?.title.focus());
}
function swapDir(dir) {
  const k = X.currentKey(); const n = k && neighbour(k, dir);
  if (!n) return toast('No pane in that direction', 'warn');
  X.act('Swap panes', () => api.swap(k, n), 'Panes swapped');
}
function resizeMode() {
  const k = X.currentKey(); if (!k) return;
  resizing = k; chordHint(`Resize ${X.nameOf(X.S.panes.get(k))}: arrows grow/shrink, Esc to finish`);
}
function resizeKey(ev) {
  const dirs = { ArrowLeft: 'left', ArrowRight: 'right', ArrowUp: 'up', ArrowDown: 'down', h: 'left', l: 'right', k: 'up', j: 'down' };
  ev.preventDefault();
  if (ev.key === 'Escape' || ev.key === 'Enter') { resizing = null; chordHint(''); return true; }
  const d = dirs[ev.key];
  if (d) api.resize(resizing, d, 2).catch((e) => { resizing = null; chordHint(''); fail('Resize')(new Error(why(e))); });
  return true;
}

// ---------- Herdr layout view: real split rects, drag dividers to resize ----------
export function placeLayout(grid, lay, cards) {
  const A = lay.area || { x: 0, y: 0, width: 1, height: 1 };
  const pct = (v, total) => (v / total) * 100;
  const inLayout = new Set();
  for (const lp of lay.panes || []) {
    const key = X.S.session + '|' + lp.pane_id, c = cards.get(key), r = lp.rect;
    if (!c || !r) continue;
    inLayout.add(key);
    c.root.hidden = false;
    c.root.style.cssText = `left:calc(${pct(r.x - A.x, A.width)}% + 4px);top:calc(${pct(r.y - A.y, A.height)}% + 4px);width:calc(${pct(r.width, A.width)}% - 8px);height:calc(${pct(r.height, A.height)}% - 8px)`;
    if (r.x + r.width < A.x + A.width - 0.5) grid.append(divider(key, 'v', pct(r.x + r.width - A.x, A.width), pct(r.y - A.y, A.height), pct(r.height, A.height), A));
    if (r.y + r.height < A.y + A.height - 0.5) grid.append(divider(key, 'h', pct(r.y + r.height - A.y, A.height), pct(r.x - A.x, A.width), pct(r.width, A.width), A));
  }
  for (const [k, c] of cards) if (!inLayout.has(k)) c.root.hidden = true;
}
function divider(key, axis, at, from, len, A) {
  const d = el('div', { class: 'divider ' + axis, role: 'separator', 'aria-orientation': axis === 'v' ? 'vertical' : 'horizontal',
    'aria-label': 'Drag to resize', tabindex: '-1' });
  d.style.cssText = axis === 'v' ? `left:${at}%;top:${from}%;height:${len}%` : `top:${at}%;left:${from}%;width:${len}%`;
  d.addEventListener('pointerdown', (ev) => {
    ev.preventDefault(); d.setPointerCapture(ev.pointerId);
    const box = d.parentElement.getBoundingClientRect(); // one read at drag start
    const cell = axis === 'v' ? box.width / A.width : box.height / A.height;
    const start = axis === 'v' ? ev.clientX : ev.clientY; let delta = 0;
    d.classList.add('drag');
    const move = (e) => { delta = (axis === 'v' ? e.clientX : e.clientY) - start; d.style.transform = axis === 'v' ? `translateX(${delta}px)` : `translateY(${delta}px)`; };
    const up = () => {
      d.removeEventListener('pointermove', move); d.classList.remove('drag'); d.style.transform = '';
      const cells = Math.round(delta / cell);
      if (cells) X.act('Resize', () => api.resize(key, axis === 'v' ? (cells > 0 ? 'right' : 'left') : (cells > 0 ? 'down' : 'up'), Math.abs(cells)), `Resized by ${Math.abs(cells)} cells`);
    };
    d.addEventListener('pointermove', move);
    d.addEventListener('pointerup', up, { once: true });
  });
  return d;
}
export function dragSwap(head, key) {
  head.addEventListener('dragstart', (ev) => {
    if (ev.target.closest('button,input,textarea')) { ev.preventDefault(); return; }
    ev.dataTransfer.setData('text/deck-pane', key); ev.dataTransfer.effectAllowed = 'move';
  });
}

// ---------- splits, tabs, spaces ----------
export function split(p, direction) {
  if (!p) return toast('No pane selected', 'warn');
  return X.act(`Split ${direction}`, () => api.splitDir(p.key, direction, p.cwd), `Split ${direction}`);
}
function goTab(n) {
  const p = cur(); const ws = X.S.filter.ws || p?.workspace_id || X.S.workspaces[0]?.workspace_id;
  const tabs = X.S.tabs.filter((t) => (t.workspace_id || t.parent_workspace) === ws);
  const t = tabs[n - 1]; if (!t) return toast(`No tab ${n} in this space`, 'warn');
  X.setFilter(ws, t.tab_id);
}
const curTab = () => X.S.tabs.find((t) => t.tab_id === (X.S.filter.tab || cur()?.tab_id));
const curWs = () => X.S.workspaces.find((w) => w.workspace_id === (X.S.filter.ws || cur()?.workspace_id));
function newTab() { const w = curWs(); if (!w) return toast('Pick a space first', 'warn'); X.act('New tab', () => api.tabCreate(w.workspace_id, cur()?.cwd || '')); }
async function renameTab() {
  const t = curTab(); if (!t) return toast('Pick a tab first', 'warn');
  const v = await ask('Rename tab', 'Tab name', t.label || ''); if (v?.trim()) X.act('Rename tab', () => api.tabRename(t.tab_id, v.trim()));
}
async function closeTab() {
  const t = curTab(); if (!t) return toast('Pick a tab first', 'warn');
  const n = [...X.S.panes.values()].filter((p) => p.tab_id === t.tab_id);
  if (await confirm('Close tab?', `Tab ${t.label || t.tab_id} with ${n.length} pane(s):\n${n.map((p) => '  ' + X.nameOf(p)).join('\n')}\n\nAgents in it stop.`, 'Close tab'))
    X.act('Close tab', () => api.tabClose(t.tab_id));
}
async function renameSpace() {
  const w = curWs(); if (!w) return toast('Pick a space first', 'warn');
  const v = await ask('Rename space', 'Space name', w.label || ''); if (v?.trim()) X.act('Rename space', () => api.wsRename(w.workspace_id, v.trim()));
}
async function closeSpace() {
  const w = curWs(); if (!w) return toast('Pick a space first', 'warn');
  const n = [...X.S.panes.values()].filter((p) => p.workspace_id === w.workspace_id);
  if (await confirm('Close space?', `Space ${w.label || w.workspace_id} with ${n.length} pane(s). Every agent in it stops.`, 'Close space'))
    X.act('Close space', () => api.wsClose(w.workspace_id));
}
function jumpNotified() {
  const k = X.S.lastNotified;
  if (!k || !X.S.panes.has(k)) return toast('No agent has notified yet', 'warn');
  X.setFilter(null, null); X.zoom(k);
}
function copyMode() {
  const c = X.card(X.currentKey()); if (!c) return;
  c.screenBox.focus(); toast('Copy mode: select text with the mouse or Shift+arrows, Ctrl+C copies. Ctrl+Shift+C copies the whole screen.', 'info', 6000);
}
function detach() {
  toast('Closing the Deck window. Herdr and every agent keep running.', 'info');
  setTimeout(() => window.close(), 400);
}
async function settings() {
  const L = X.LS;
  const theme = el('select', { 'aria-label': 'Theme' }, ['dark', 'light', 'hc'].map((t) => el('option', { value: t, text: { dark: 'Dark', light: 'Light', hc: 'High contrast' }[t], selected: L.get('theme', 'dark') === t })));
  const density = el('select', { 'aria-label': 'Density' }, ['normal', 'compact'].map((t) => el('option', { value: t, text: t, selected: L.get('density', 'normal') === t })));
  const font = el('input', { type: 'number', min: 10, max: 22, value: L.get('font', 13), 'aria-label': 'Screen font size' });
  const v = await dialog({ title: 'Settings', body: el('div', { class: 'form2' }, field('Theme', theme), field('Density', density), field('Screen font (px)', font)),
    actions: [{ label: 'Cancel', value: null }, { label: 'Apply', value: 'ok', kind: 'primary' }] });
  if (v !== 'ok') return;
  P.setLook('theme', theme.value); P.setLook('density', density.value); P.setLook('font', Math.max(10, Math.min(22, +font.value || 13)));
}

// ---------- A5-A7 folder picker ----------
export async function folderPicker(agent) {
  const path = el('input', { 'aria-label': 'Folder path', spellcheck: false, value: cur()?.cwd || X.S.root });
  const recent = el('ul', { class: 'list folders', 'aria-label': 'Recent folders' });
  const browse = el('ul', { class: 'list folders', 'aria-label': 'Sub-folders' });
  const note = el('p', { class: 'muted small' });
  const choose = (p) => { path.value = p; list(p); };
  const list = async (p) => {
    browse.textContent = '';
    try {
      const r = await api.foldersList(p);
      if (r.parent != null) browse.append(el('li', {}, el('button', { type: 'button', class: 'linkish', onclick: () => choose(r.parent) }, icon('folder'), ' .. (up)')));
      for (const d of r.dirs || []) {
        const full = r.path ? r.path.replace(/[\\/]?$/, '\\') + d : d;
        browse.append(el('li', {}, el('button', { type: 'button', class: 'linkish', onclick: () => choose(full) }, icon('folder'), ' ' + d)));
      }
      note.textContent = '';
    } catch (e) { note.textContent = 'Browse: ' + why(e) + ' You can still type a path.'; }
  };
  try {
    const r = await api.foldersRecent();
    if (r.last && !cur()) path.value = r.last;
    for (const f of r.folders || []) recent.append(el('li', {}, el('button', { type: 'button', class: 'linkish', onclick: () => choose(f) }, icon('clock'), ' ' + f)));
  } catch (e) {
    const seen = [...new Set([...X.S.panes.values()].map((p) => p.cwd).filter(Boolean))];
    for (const f of seen) recent.append(el('li', {}, el('button', { type: 'button', class: 'linkish', onclick: () => choose(f) }, icon('folder'), ' ' + f)));
    note.textContent = 'Recent list from open panes (' + (missing(e) ? 'backend endpoint missing' : e.message) + ').';
  }
  list(path.value);
  const acts = [{ label: 'Cancel', value: null }, { label: 'Open', value: '', kind: agent === '' ? 'primary' : '' },
    { label: 'Open + Claude', value: 'claude', kind: agent === 'claude' ? 'primary' : '' }, { label: 'Open + team', value: 'team', kind: agent === 'team' ? 'primary' : '' }];
  const v = await dialog({ title: 'Open folder', wide: true,
    body: el('div', {}, field('Folder', path), el('div', { class: 'cols2' }, el('div', {}, el('h3', { text: 'Recent' }), recent), el('div', {}, el('h3', { text: 'Browse' }), browse)), note,
      el('p', { class: 'muted small', text: 'Open = new space there (Ctrl+B F). + Claude starts Claude in it (Ctrl+B Shift+F). + team = Claude left, Codex right (Ctrl+B T).' })),
    actions: acts, check: () => (path.value.trim() ? '' : 'Type or pick a folder') });
  if (v === null) return;
  X.act('Open folder', () => api.wsOpen(path.value.trim(), v), `Opening ${path.value.trim()}${v ? ' + ' + v : ''}`);
}

// ---------- A8 columns ----------
export async function columns() {
  const p = cur(); if (!p) return toast('No pane selected', 'warn');
  const n = el('input', { type: 'number', min: 2, max: 8, value: 4, 'aria-label': 'Count' });
  const dir = el('select', { 'aria-label': 'Direction' }, el('option', { value: 'right', text: 'columns (side by side)' }), el('option', { value: 'down', text: 'rows (stacked)' }));
  const v = await dialog({ title: `Split ${X.nameOf(p)} into equal parts`, body: el('div', { class: 'form2' }, field('Count', n), field('As', dir)),
    actions: [{ label: 'Cancel', value: null }, { label: 'Split', value: 'ok', kind: 'primary' }], check: () => (+n.value >= 2 && +n.value <= 8 ? '' : '2 to 8') });
  if (v === 'ok') X.act('Columns', () => api.columns(p.key, +n.value, dir.value), `Split into ${n.value}`);
}

// ---------- A9 layout builder ----------
export async function layoutBuilder() {
  const S = X.S;
  const count = el('input', { type: 'number', min: 1, max: 6, value: 2, 'aria-label': 'Columns' });
  const where = el('select', { 'aria-label': 'Open in' }, el('option', { value: 0, text: 'new tab in this space' }), el('option', { value: 1, text: 'a new space' }));
  const name = el('input', { 'aria-label': 'Name', value: 'work' });
  const presetSel = el('select', { 'aria-label': 'Preset' }, el('option', { value: '', text: '(none)' }));
  const preview = el('div', { class: 'lb-preview', 'aria-hidden': 'true' });
  const rowsBox = el('div', { class: 'lb-rows' });
  let presets = [];
  const runsOptions = () => [el('option', { value: 'shell|', text: 'PowerShell (empty)' }),
    ...S.profiles.map((p) => el('option', { value: 'new|' + p.name, text: `New: ${p.name}  [${p.model || 'default model'}]` })),
    ...[...S.panes.values()].filter((p) => p.name || p.agent).map((p) => el('option', { value: 'existing|' + p.pane_id, text: `Running: ${X.nameOf(p)} - ${p.agent_status}` }))];
  const rows = [];
  const draw = () => {
    const n = Math.max(1, Math.min(6, +count.value || 1));
    while (rows.length < n) {
      const f = el('input', { 'aria-label': `Column ${rows.length + 1} folder`, value: rows[rows.length - 1]?.folder.value || cur()?.cwd || X.S.root, spellcheck: false });
      const runs = el('select', { 'aria-label': `Column ${rows.length + 1} runs` }, runsOptions());
      const model = el('input', { 'aria-label': `Column ${rows.length + 1} model`, placeholder: 'model (optional)' });
      const account = el('input', { 'aria-label': `Column ${rows.length + 1} account`, placeholder: 'account', size: 6 });
      runs.onchange = () => { const [t] = runs.value.split('|'); model.disabled = account.disabled = t !== 'new'; f.disabled = t === 'existing'; paintPreview(); };
      runs.onchange();
      rows.push({ folder: f, runs, model, account, row: el('div', { class: 'lb-row' }, el('b', { text: rows.length + 1 }), f, runs, model, account) });
    }
    rows.length = n;
    rowsBox.replaceChildren(...rows.map((r) => r.row));
    paintPreview();
  };
  const paintPreview = () => preview.replaceChildren(...rows.map((r, i) => el('div', {}, el('b', { text: i + 1 }), el('span', { text: r.runs.selectedOptions[0]?.text.split('  ')[0] || '' }))));
  count.oninput = draw;
  const plan = () => ({ Name: name.value.trim() || 'work', Where: +where.value, WorkspaceId: X.S.filter.ws || cur()?.workspace_id || '',
    Columns: rows.map((r) => { const [Type, Value] = r.runs.value.split('|'); return { Folder: r.folder.value.trim(), Type, Value, Model: r.model.value.trim(), Account: r.account.value.trim() }; }) });
  presetSel.onchange = () => {
    const p = presets.find((x) => x.name === presetSel.value)?.plan; if (!p) return;
    const cols = p.Columns || p.columns || [];
    count.value = cols.length; rows.length = 0; draw();
    cols.forEach((c, i) => {
      rows[i].folder.value = c.Folder || c.folder || '';
      const t = c.Type || (c.runs ? 'new' : 'shell'), v = c.Value || c.runs || '';
      rows[i].runs.value = `${t}|${v}`; if (rows[i].runs.selectedIndex < 0) rows[i].runs.selectedIndex = 0;
      rows[i].model.value = c.Model || c.model || ''; rows[i].account.value = c.Account || c.account || ''; rows[i].runs.onchange();
    });
    if (p.Name || p.name) name.value = p.Name || p.name;
  };
  const note = el('p', { class: 'muted small' });
  api.layouts().then((r) => { presets = r.presets || []; for (const p of presets) presetSel.append(el('option', { value: p.name, text: p.name })); })
    .catch((e) => { note.textContent = 'Presets: ' + why(e); });
  draw();
  const v = await dialog({ title: 'Layout builder', wide: true,
    body: el('div', {}, el('div', { class: 'form2' }, field('Columns', count), field('Open in', where), field('Name', name), field('Preset', presetSel)),
      preview, el('div', { class: 'lb-head muted small' }, el('span', { text: '#' }), el('span', { text: 'Folder' }), el('span', { text: 'Runs' }), el('span', { text: 'Model' }), el('span', { text: 'Account' })),
      rowsBox, note),
    actions: [{ label: 'Cancel', value: null }, { label: 'Save preset…', value: 'save' }, { label: 'Open layout', value: 'ok', kind: 'primary' }],
    check: () => (rows.every((r) => r.folder.disabled || r.folder.value.trim()) ? '' : 'Every column needs a folder') });
  if (v === 'save') {
    const n = await ask('Save preset', 'Preset name', name.value);
    if (n?.trim()) X.act('Save preset', () => api.layoutSave(n.trim(), plan()), 'Preset saved: ' + n.trim());
  } else if (v === 'ok') X.act('Open layout', () => api.layoutBuild(plan()), 'Building the layout…');
}

// ---------- A10 live agent board (from live state, no extra polling) ----------
const ORDER = { blocked: 0, error: 1, working: 2, idle: 3, done: 4, unknown: 5 };
export function board() {
  P.drawer('board', 'Agent board', (b) => {
    const agents = [...X.S.panes.values()].filter((p) => p.name || p.agent).sort((a, c) => ORDER[X.statusOf(a)] - ORDER[X.statusOf(c)]);
    if (!agents.length) b.append(el('p', { class: 'muted', text: 'No agents running.' }));
    const ws = (id) => X.S.workspaces.find((w) => w.workspace_id === id)?.label || id;
    b.append(el('table', { class: 'board' }, el('thead', {}, el('tr', {}, ['', 'Agent', 'State', 'Space', 'Model', 'ctx', 'Last line', ''].map((h) => el('th', { text: h })))),
      el('tbody', {}, agents.map((p) => el('tr', { 'data-s': X.statusOf(p) },
        el('td', {}, el('span', { class: 'dot', 'data-s': X.statusOf(p) })), el('td', {}, el('strong', { text: X.nameOf(p) })),
        el('td', { text: X.statusOf(p) }), el('td', { text: ws(p.workspace_id) }), el('td', { class: 'mono small', text: p.tokens?.model || '' }),
        el('td', { class: 'mono small', text: p.tokens?.ctx || '' }), el('td', { class: 'mono small clip', text: X.lastLine(p) }),
        el('td', {}, el('button', { text: 'Show', onclick: () => { X.setFilter(null, null); X.zoom(p.key); } })))))));
  }, true);
}

// ---------- A11 spaces side by side ----------
export async function spaces() {
  const boxes = X.S.workspaces.map((w) => el('label', { class: 'li' }, el('input', { type: 'checkbox', value: w.workspace_id, checked: true }), ' ' + (w.label || w.workspace_id)));
  const note = el('p', { class: 'muted small', text: 'The top one becomes the left column. "Separate again" puts each group back in its own space.' });
  const v = await dialog({ title: 'Spaces side by side', body: el('div', {}, el('div', { class: 'list' }, boxes), note),
    actions: [{ label: 'Cancel', value: null }, { label: 'Separate again', value: 'sep' }, { label: 'Side by side', value: 'join', kind: 'primary' }],
    check: (val) => (val === 'join' && boxes.filter((b) => b.querySelector('input').checked).length < 2 ? 'Tick at least two spaces' : '') });
  if (v === 'join') X.act('Side by side', () => api.spacesJoin(boxes.map((b) => b.querySelector('input')).filter((i) => i.checked).map((i) => i.value)));
  else if (v === 'sep') X.act('Separate again', () => api.spacesSeparate());
}

// ---------- A13 stop everything, A14 next-account Claude ----------
export async function stopAll() {
  if (X.S.safe) return toast('Safe mode is on: writes are disabled', 'warn');
  const agents = [...X.S.panes.values()].filter((p) => p.name || p.agent);
  const ok = await confirmTyped('Stop Herdr for real?', `Every agent (${agents.length}) and every pane (${X.S.panes.size}) stops.\nClosing the Deck window does NOT do this; it is only for a real stop.`, 'stop');
  if (ok) X.act('Stop everything', () => api.stopAll('stop'), 'Stopping Herdr…');
}
export function nextClaude() {
  const p = cur();
  return X.act('Claude on next account', () => api.start({ profile: 'next', model: '', effort: '', account: '', cwd: p?.cwd || X.S.root, key: p?.key || '' }), 'Starting Claude on the least-used account…');
}

// ---------- A12 help: chords + Herdr keys + Deck keys + GUIDE.md, one search ----------
const DECK_KEYS = [['Ctrl+K', 'Command palette'], ['Ctrl+Shift+F', 'Search all screens'], ['?', 'Keys overlay'], ['1 / 2 / 3', 'Grid columns'], ['L', 'Grid / Herdr layout view'],
  ['J / K or ] / [', 'Next / previous card'], ['Z or double-click title', 'Zoom (Esc returns)'], ['/ or P', 'Message box of the current card'],
  ['Enter / Shift+Enter', 'Send / new line'], ['↑ / ↓ in an empty box', 'Message history'], ['X', 'Select for broadcast'], ['I', 'Attention inbox'],
  ['Alt+↑ / Alt+↓', 'Previous / next space or tab'], ['Ctrl+Shift+V', 'Paste to pane'], ['Ctrl+Shift+C', 'Copy screen'], ['Ctrl+Shift+X', 'Leave raw typing mode'],
  ['Drag a card title onto another', 'Swap the two panes'], ['Drag a divider (layout view)', 'Resize the split']];
function keyRows() {
  return [...CHORDS.map(([k, d]) => ['Ctrl+B  ' + (k.length === 1 && k !== k.toLowerCase() ? 'Shift+' + k : k), d, 'chord']), ...DECK_KEYS.map(([k, d]) => [k, d, 'deck'])];
}
export function keysOverlay() { help(true); }
export function help(keysOnly) {
  P.drawer('help', keysOnly ? 'Keys' : 'Shortcuts and guide', async (b) => {
    const q = el('input', { type: 'search', placeholder: 'Type to filter keys and the guide', 'aria-label': 'Filter' });
    const tbl = el('tbody');
    const guide = el('div', { class: 'guide small' });
    let guideText = '';
    const paint = () => {
      const s = q.value.trim().toLowerCase();
      tbl.replaceChildren(...keyRows().filter(([k, d]) => !s || (k + ' ' + d).toLowerCase().includes(s)).map(([k, d, t]) =>
        el('tr', {}, el('th', {}, el('kbd', { text: k })), el('td', { text: d }), el('td', { class: 'muted small', text: t }))));
      if (!keysOnly) {
        const parts = guideText.split(/\n(?=#{1,4} )/);
        const hits = s ? parts.filter((p) => p.toLowerCase().includes(s)) : parts;
        guide.replaceChildren(...hits.slice(0, 40).map((p) => el('pre', { class: 'guide-sec', text: p.trim() })));
      }
    };
    q.oninput = paint;
    b.append(q, el('table', { class: 'keys-t' }, tbl), keysOnly ? '' : el('h3', { text: 'Guide (docs/GUIDE.md)' }), guide);
    paint(); q.focus();
    if (!keysOnly) {
      try { guideText = (await api.docs('GUIDE')).text || ''; paint(); } catch (e) { guide.textContent = 'Guide: ' + why(e); }
    }
  });
}

// ---------- A16 agents hub (herdr-agents commands) ----------
export const jobs = new Map();
export async function syncJobs() {
  try { const r = await api.jobs(); for (const j of r.jobs || []) upsertJob(jobs, j); P.refreshOpen(true); }
  catch (e) { toast('Jobs reconciliation failed: ' + e.message, 'warn'); }
}
export function forgetPane(key) { sent.delete(key); worker?.postMessage({ drop: key }); }
export function onResult(d) {
  const j = upsertJob(jobs, { ...d, done: true });
  toast((j ? j.label + ': ' : '') + (d.message || (d.ok ? 'done' : 'failed')), d.ok ? 'ok' : 'error');
  if (j) { j.done = true; j.ok = d.ok; j.output = (d.output || d.message || '').slice(-32768); P.refreshOpen(true); }
}
const HUB = [
  ['status', 'Status', 'get'], ['tools', 'Tools', 'get'], ['ask', 'Ask', 'form'], ['read', 'Read', 'name'], ['diff', 'Diff', 'name'],
  ['fanout', 'Fan-out', 'form'], ['collect', 'Collect', 'get'], ['history', 'History', 'get'], ['team', 'Team', 'form'],
  ['accounts', 'Accounts', 'get'], ['usage', 'Usage', 'usage'], ['login', 'Sign-in', 'signin'], ['handoff', 'Handoff', 'form'],
];
let hubTab = 'status';
export function agentsHub(tab) {
  syncJobs();
  if (tab) hubTab = tab;
  P.drawer('agents', 'Agents hub', async (b, signal) => {
    const tabs = el('div', { class: 'tabs', role: 'tablist' }, HUB.map(([id, label]) => el('button', { role: 'tab', 'aria-selected': String(id === hubTab), text: label,
      onclick: () => { hubTab = id; P.refreshOpen(true); } })));
    const out = el('div', { class: 'hub-body', 'data-drawer-key': 'hub-' + hubTab });
    b.append(tabs, out);
    await hubBody(out, signal);
    if (jobs.size) b.append(el('h3', { text: 'Jobs' }), el('ul', { class: 'list' }, [...jobs.values()].reverse().slice(0, 10).map((j) => el('li', {},
      el('span', { class: 'dot', 'data-s': j.done ? (j.ok ? 'done' : 'error') : 'working' }), ' ' + j.label + ' ', el('span', { class: 'muted small', text: ago(j.t) + ' ago' }),
      j.output ? el('pre', { class: 'mono small preview', text: j.output }) : ''))));
  }, false);
}
const agentNames = () => [...X.S.panes.values()].filter((p) => p.name).map((p) => p.name);
async function hubBody(out, signal) {
  const [id, , kind] = HUB.find((h) => h[0] === hubTab);
  const pre = el('pre', { class: 'mono small preview', text: 'Loading…' });
  const show = async (fn) => { out.append(pre); try { const r = await fn(); pre.textContent = r.output || r.text || JSON.stringify(r, null, 2); } catch (e) { pre.textContent = why(e); } };
  if (kind === 'get') return show(() => (id === 'accounts' ? api.accounts() : api.agentsGet(id)));
  if (kind === 'usage') { P.usage(); return; }
  if (kind === 'signin') { P.signin(); return; }
  const nameSel = (multi) => el('select', { 'aria-label': 'Agent', multiple: multi || null, size: multi ? 5 : null }, agentNames().map((n) => el('option', { value: n, text: n })));
  if (kind === 'name') {
    const s = nameSel();
    out.append(el('div', { class: 'row' }, s, el('button', { text: id === 'diff' ? 'Show diff' : 'Read answer', onclick: () => { pre.remove(); show(() => api.agentsGet(id, { name: s.value })); } })));
    return;
  }
  if (id === 'ask' || id === 'fanout') {
    const s = nameSel(id === 'fanout');
    const ta = el('textarea', { rows: 5, 'aria-label': 'Task', placeholder: 'Task text, or: Read CLAUDE.md and AGENTS.md, then do the task in tasks/x.md' });
    const fresh = el('input', { type: 'checkbox' });
    out.append(field(id === 'ask' ? 'Agent' : 'Agents (Ctrl+click for several)', s), ta, id === 'ask' ? el('label', {}, fresh, ' -Fresh (new conversation)') : '',
      el('div', { class: 'row' }, el('button', { class: 'primary', text: id === 'ask' ? 'Ask and wait' : 'Fan out', 'data-w': '1', onclick: async () => {
        const names = [...s.selectedOptions].map((o) => o.value);
        if (!names.length || !ta.value.trim()) return toast('Pick agent(s) and write the task', 'warn');
        const r = await X.act(id, () => (id === 'ask' ? api.agentsPost('ask', { name: names[0], text: ta.value, fresh: fresh.checked }) : api.agentsPost('fanout', { names, text: ta.value })), 'Sent; the answer arrives here');
        if (r?.id) { upsertJob(jobs, { id: r.id, label: `${id} ${names.join(', ')}`, t: Date.now() }); P.refreshOpen(true); }
      } })));
    return;
  }
  if (id === 'team') { out.append(el('button', { class: 'primary', text: 'Team presets…', onclick: () => P.team() })); return; }
  if (id === 'handoff') {
    const ta = el('textarea', { rows: 8, 'aria-label': 'Handoff text', placeholder: 'State, open work, next steps (tasks/HANDOFF-TEMPLATE.md sections)' });
    out.append(ta, el('button', { class: 'primary', text: 'Write handoff file', 'data-w': '1', onclick: () => X.act('Handoff', () => api.agentsPost('handoff', { text: ta.value })) }));
  }
}

// ---------- A17 tools ----------
export function tools() {
  P.drawer('tools', 'Tools', (b) => {
    const prof = el('select', { 'aria-label': 'Headless profile' }, X.S.profiles.filter((p) => p.kind === 'cline').map((p) => el('option', { value: p.name, text: p.name })));
    const ta = el('textarea', { rows: 3, 'aria-label': 'Headless task' });
    b.append(
      el('h3', { text: 'Setup check' }), el('button', { onclick: () => P.check() }, icon('check'), ' Run check.ps1'),
      el('h3', { text: 'Headless run (herdr-headless.ps1)' }), field('Cline profile', prof), ta,
      el('button', { 'data-w': '1', onclick: async () => {
        if (!ta.value.trim()) return toast('Write the task', 'warn');
        const r = await X.act('Headless', () => api.headless(prof.value, ta.value), 'Running headless; result arrives as a toast and in the Agents hub');
        if (r?.id) upsertJob(jobs, { id: r.id, label: 'headless ' + prof.value, t: Date.now() });
      } }, icon('play'), ' Run headless'),
      el('h3', { text: 'Skills' }), el('button', { 'data-w': '1', onclick: () => X.act('Sync skills', () => api.syncSkills()) }, icon('tools'), ' sync-skills.ps1'),
      el('h3', { text: 'Specs and docs' }),
      el('div', { class: 'row' }, el('button', { onclick: () => P.tasks() }, icon('task'), ' Tasks'), el('button', { onclick: () => P.docs() }, icon('doc'), ' Docs hub'),
        el('button', { onclick: () => P.runs() }, icon('clock'), ' Run history'), el('button', { onclick: () => P.addLesson() }, icon('edit'), ' Add lesson'),
        el('button', { onclick: () => pipeline() }, icon('arrowR'), ' Pipeline')),
      el('p', { class: 'muted small', text: 'Model trial (herdr-trial.ps1) runs long and spends tokens: start it from a pane (+ Shell) so you can watch it.' }));
  });
}

// ---------- B: global search across every screen (web worker keeps typing at 60 fps) ----------
let worker = null; const sent = new Map();
function syncWorker() {
  worker ||= new Worker(new URL('./search.worker.js', import.meta.url));
  for (const k of [...sent.keys()]) if (!X.S.panes.has(k)) { worker.postMessage({ drop: k }); sent.delete(k); }
  for (const p of X.S.panes.values()) {
    const c = X.card(p.key); const v = c ? c.view.version : -1;
    if (sent.get(p.key) === v) continue;
    sent.set(p.key, v);
    worker.postMessage({ doc: { key: p.key, name: X.nameOf(p), text: c ? c.view.plain : (p.screen || '').replace(/\x1b\[[0-9;?]*[ -/]*[@-~]/g, '') } });
  }
}
export function search(initial = '') {
  P.drawer('search', 'Search all screens', (b) => {
    const q = el('input', { type: 'search', value: initial, placeholder: 'Text or /regex/ - searches every pane', 'aria-label': 'Search all screens' });
    const res = el('ul', { class: 'list hits', 'aria-live': 'polite' });
    let t = 0, seq = 0;
    const run = () => {
      clearTimeout(t);
      t = setTimeout(() => {
        syncWorker();
        const my = ++seq;
        worker.onmessage = (ev) => {
          if (ev.data.seq !== my) return;
          res.replaceChildren(...(ev.data.hits.length ? ev.data.hits.map((h) => el('li', {}, el('button', { class: 'linkish hit', onclick: () => jumpTo(h.key, h.n) },
            el('strong', { text: h.name }), el('span', { class: 'muted small', text: ` line ${h.n + 1}` }), el('span', { class: 'mono small clip', text: h.line })))) : [el('li', { class: 'muted', text: q.value ? 'No match.' : '' })]));
        };
        worker.postMessage({ q: q.value, seq: my });
      }, 120);
    };
    q.oninput = run;
    b.append(q, el('div', { class: 'row' }, el('button', { text: 'Find errors', onclick: () => { q.value = '/error|failed|exception|panic|traceback|denied/'; run(); } }),
      el('button', { text: 'Find blocked prompts', onclick: () => { q.value = '/trust|approve|allow|y\\/n|\\(y\\)|sign in|log in/'; run(); } })), res);
    q.focus(); if (initial) run();
  });
}
function jumpTo(key, n) {
  X.setFilter(null, null);
  if (X.S.zoom !== key) X.zoom(key);
  requestAnimationFrame(() => requestAnimationFrame(() => X.card(key)?.view.scrollToLine(n)));
}

// ---------- B: composer chips (task file, folder, snippet) + pending sends ----------
export function attachChip(c, type, value) {
  (c.chipList ||= []).push({ type, value });
  paintChips(c);
}
export function clearChips(c) { c.chipList = []; paintChips(c); }
function paintChips(c) {
  const chips = (c.chipList || []).map((ch, i) => el('span', { class: 'achip' }, icon(ch.type === 'task' ? 'task' : ch.type === 'folder' ? 'folder' : 'doc'), ch.value,
    el('button', { type: 'button', class: 'x', 'aria-label': 'Remove ' + ch.value, text: '×', onclick: () => { c.chipList.splice(i, 1); paintChips(c); } })));
  if (c.pending) chips.push(el('span', { class: 'achip pend' }, el('span', { class: 'spin', 'aria-hidden': 'true' }), `sending${c.pending > 1 ? ' ×' + c.pending : ''}…`));
  c.attach.replaceChildren(...chips);
  c.attach.hidden = !chips.length;
}
export function pending(c, d) { c.pending = Math.max(0, (c.pending || 0) + d); paintChips(c); }
export function composeText(c) {
  const pre = (c.chipList || []).map((ch) => (ch.type === 'task' ? `Read CLAUDE.md and AGENTS.md, then do the task in ${ch.value}.` : ch.type === 'folder' ? `Work in the folder ${ch.value}.` : ch.value));
  return [...pre, c.ta.value].filter((s) => s.trim()).join('\n');
}
export async function attachTask(c) {
  let list = [];
  try { list = (await api.tasks()).tasks || []; } catch (e) { return toast('Tasks: ' + why(e), 'error'); }
  const sel = el('select', { 'aria-label': 'Task file', size: Math.min(12, list.length + 1) }, list.map((t) => el('option', { value: t.file, text: t.file })));
  const v = await dialog({ title: 'Attach a task file', body: sel, actions: [{ label: 'Cancel', value: null }, { label: 'Attach', value: 'ok', kind: 'primary' }],
    check: () => (sel.value ? '' : 'Pick a task') });
  if (v === 'ok') { attachChip(c, 'task', sel.value); c.ta.focus(); }
}

// ---------- B: status sparkline, done strip, reviewer, pipeline ----------
export function spark(box, hist) {
  const h = hist.slice(-24); if (!h.length) { box.replaceChildren(); return; }
  const now = Date.now();
  box.replaceChildren(...h.map((e, i) => {
    const dur = ((h[i + 1]?.t || now) - e.t) / 1000;
    const s = el('span', { 'data-s': e.s, title: `${e.s} ${new Date(e.t).toLocaleTimeString()}` });
    s.style.flexGrow = String(Math.max(1, Math.min(60, Math.round(dur / 10))));
    return s;
  }));
}
export function doneStrip(c, p, show) {
  c.done.hidden = !show;
  if (!show) return;
  c.done.replaceChildren(el('span', {}, icon('check'), ' Finished'),
    el('button', { type: 'button', onclick: () => G.proof(p) }, icon('check'), ' Proof'),
    el('button', { type: 'button', 'data-w': '1', onclick: () => toReviewer(p) }, icon('arrowR'), ' Send to reviewer'),
    el('button', { type: 'button', onclick: () => P.check() }, icon('check'), ' Run check.ps1'),
    el('button', { type: 'button', onclick: () => P.changes(p) }, icon('git'), ' Show diff'),
    el('button', { type: 'button', class: 'icon', 'aria-label': 'Dismiss', onclick: () => { X.S.finished.delete(p.key); X.schedule(p.key); } }, icon('x')));
}
export async function toReviewer(p, reviewerKey) {
  const others = [...X.S.panes.values()].filter((q) => q.key !== p.key && (q.name || q.agent));
  if (!others.length) return toast('No other agent to review; start one with + Agent', 'warn');
  const sel = el('select', { 'aria-label': 'Reviewer' }, others.map((q) => el('option', { value: q.key, text: X.nameOf(q), selected: q.key === reviewerKey })));
  const answer = X.plainOf(p.key).split('\n').slice(-60).join('\n');
  const ta = el('textarea', { rows: 8, class: 'mono small', 'aria-label': 'Review request',
    value: `Review the work ${X.nameOf(p)} just finished in ${p.cwd}. Its final screen:\n\`\`\`text\n${answer}\n\`\`\`\nCheck it against its task, run the checks, and list findings ordered by severity.` });
  const v = await dialog({ title: 'Send to reviewer', wide: true, body: el('div', {}, field('Reviewer', sel), ta),
    actions: [{ label: 'Cancel', value: null }, { label: 'Send', value: 'ok', kind: 'primary' }] });
  if (v === 'ok') G.sendGuarded(sel.value, ta.value);
}
const pipe = { writer: '', reviewer: '', task: '', step: 0 };
export function pipeline() {
  P.drawer('pipeline', 'Pipeline: writer → reviewer → final gate', (b) => {
    const agents = [...X.S.panes.values()].filter((q) => q.name || q.agent);
    const pick = (role) => el('select', { 'aria-label': role, onchange: (e) => { pipe[role] = e.target.value; } },
      el('option', { value: '', text: '(pick)' }), agents.map((q) => el('option', { value: q.key, text: X.nameOf(q), selected: pipe[role] === q.key })));
    const task = el('input', { 'aria-label': 'Task file', value: pipe.task, placeholder: 'tasks/x.md', oninput: (e) => { pipe.task = e.target.value; } });
    const st = (k) => (X.S.panes.get(pipe[k]) ? X.statusOf(X.S.panes.get(pipe[k])) : '-');
    b.append(el('ol', { class: 'pipe' },
      el('li', {}, el('b', { text: 'Writer ' }), pick('writer'), ' ', task, ' ', el('span', { class: 'muted small', text: st('writer') }),
        el('button', { 'data-w': '1', onclick: () => { const w = X.S.panes.get(pipe.writer); if (!w || !pipe.task) return toast('Pick the writer and the task file', 'warn');
          G.sendGuarded(w.key, `Read CLAUDE.md and AGENTS.md, then do the task in ${pipe.task}`); } }, 'Send task')),
      el('li', {}, el('b', { text: 'Reviewer ' }), pick('reviewer'), ' ', el('span', { class: 'muted small', text: st('reviewer') }),
        el('button', { 'data-w': '1', onclick: () => { const w = X.S.panes.get(pipe.writer); if (!w) return toast('Pick the writer', 'warn'); toReviewer(w, pipe.reviewer); } }, 'Send writer\'s answer')),
      el('li', {}, el('b', { text: 'Final gate ' }), el('button', { onclick: () => P.check() }, 'Run check.ps1'), ' ',
        el('button', { onclick: () => { const w = X.S.panes.get(pipe.writer); if (w) P.changes(w); } }, 'Show diff'))),
      el('p', { class: 'muted small', text: 'docs/TEAM-PLAYBOOK.md: one writer, an independent reviewer, then the lead runs the gate and commits. Nothing here is sent without your click.' }));
  }, true);
}

// ---------- perf hook ----------
let perfHook = null;
export function setPerfHook(fn) { perfHook = fn; }
export function frameDone(ms) { if (perfHook) perfHook(ms); }
