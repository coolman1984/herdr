// Herdr Deck frontend: store, event stream, pane grid / Herdr layout, rail, keyboard. No framework.
import { api, initApi, MOCK } from './api.js';
import { el, $, toast, fail, confirm, ask, copyText, sameValues, queuePatch, openMenu } from './ui.js';
import { ScreenView } from './screen.js';
import { icon, glyph, ring } from './icons.js';
import * as P from './panels.js';
import * as Q from './parity.js';
import * as G from './guard.js';

let prefs = {}, prefsTimer = 0, prefsReady = false, prefsWrite = Promise.resolve();
function savePreferences() {
  const snapshot = JSON.parse(JSON.stringify(prefs));
  prefsWrite = prefsWrite.catch(() => {}).then(() => api.savePrefs(snapshot));
  prefsWrite.catch(fail('Preferences'));
}
const LS = {
  get(k, d) { if (Object.hasOwn(prefs, k)) return prefs[k]; try { const v = localStorage.getItem('deck.' + k); return v == null ? d : JSON.parse(v); } catch { return d; } },
  set(k, v) { prefs[k] = v; if (prefsReady) { clearTimeout(prefsTimer); prefsTimer = setTimeout(savePreferences, 500); } try { localStorage.setItem('deck.' + k, JSON.stringify(v)); } catch { /* storage blocked: the setting lasts for this run only */ } },
};

const capHistory = (history) => Object.fromEntries(Object.entries(history || {}).slice(-100).map(([key, rows]) =>
  [key, Array.isArray(rows) ? rows.slice(-50).map((text) => String(text).slice(0, 32768)) : []]));

export const S = {
  session: '', root: '', paneCols: new Map(), herdr: { running: true, error: '' }, panes: new Map(), workspaces: [], tabs: [], profiles: [], layouts: [],
  filter: LS.get('filter', { ws: null, tab: null }), cols: LS.get('cols', 2), view: LS.get('view', 'grid'), zoom: null, current: null,
  selected: new Set(), connected: false, lastUpdate: 0, safe: false, rail: LS.get('rail', true),
  hidden: new Set(LS.get('hiddenPanes', [])), pinned: new Set(LS.get('pinned', [])),
  queue: new Map(), history: capHistory(LS.get('history', {})), timeline: [], hist: new Map(), finished: new Set(),
  raw: null, loaded: false, lastNotified: null, alarmAt: new Map(),
};

const STATUS = ['blocked', 'error', 'working', 'idle', 'done', 'unknown'];
const statusOf = (p) => (p.blocked ? 'blocked' : p.error ? 'error' : STATUS.includes(p.agent_status) ? p.agent_status : 'unknown');
const tok = (p) => p.tokens || {};
const nameOf = (p) => p.label || tok(p).name || p.name || p.agent || (p.title && p.title.slice(0, 40)) || p.pane_id;
const kindOf = (p) => p.agent || (S.profiles.find((x) => x.name === p.name?.replace(/-\d+$/, ''))?.kind) || '';
const isAgent = (p) => !!(p.name || p.agent);
const folder = (cwd) => (cwd || '').replace(/[\\/]+$/, '');
const worst = (list) => { for (const s of STATUS) if (list.includes(s)) return s; return 'unknown'; };
const ctxPct = (p) => { const m = /(\d+(?:\.\d+)?)\s*%/.exec(tok(p).ctx || tok(p).context || ''); return m ? +m[1] : null; };

// ---------- write guard (safe mode is enforced by the backend too) ----------
async function act(what, fn, okMsg) {
  if (S.safe || S.writeBlocked || S.panes.get(currentKey())?.stale) { toast('Writes disabled: safe mode or stale connection', 'warn'); return null; }
  try {
    const r = await fn();
    toast(okMsg || (r && (r.message || r.output)) || what + ' done', 'ok');
    return r || {};
  } catch (e) { if (['input_unknown', 'input_not_empty', 'agent_exited'].includes(e.code)) toast(e.message, 'warn', 9000); else fail(what)(e); return null; }
}

// ---------- render scheduling: only changed cards, once per frame ----------
export const cards = new Map();
const dirty = new Set();
let structDirty = true, frame = 0;
const paintSet = new Set();
function requestPaint(v) { paintSet.add(v); if (!frame) frame = requestAnimationFrame(flush); }
function schedule(key) {
  if (key) dirty.add(key); else structDirty = true;
  if (!frame) frame = requestAnimationFrame(flush);
}
function flush() {
  frame = 0;
  const t0 = performance.now();
  const struct = structDirty, anyCard = dirty.size > 0;
  if (structDirty) { structDirty = false; renderRail(); renderMain(); }
  for (const n of document.querySelectorAll('[data-w]')) { const off = !!(S.safe || S.writeBlocked || S.panes.get(n.closest('.card')?.dataset.key)?.stale); if (n.disabled !== off) n.disabled = off; }
  for (const k of dirty) if (cards.has(k)) updateCard(cards.get(k), S.panes.get(k));
  dirty.clear();
  // Screens before the top bar: setting scrollTop forces a layout, and it must not find the rest of
  // the page dirty (profiled: 2.3 s of 10 s in paintScroll while the top bar was rewritten first).
  const pending = [...paintSet]; paintSet.clear();
  for (const v of pending) v.prepare();
  const views = pending.filter((v) => v.paintSize());
  for (const v of views) v.paintScroll();
  for (const v of views) v.paint();
  if (struct || anyCard) { renderTop(); P.refreshOpen(); }
  renderStatus();
  Q.frameDone(performance.now() - t0);
}

// ---------- state merge ----------
export function mergePane(np) {
  const key = np.key || (S.session + '|' + np.pane_id);
  const old = S.panes.get(key);
  const before = old ? statusOf(old) : null;
  const topologyChanged = old && ((np.workspace_id !== undefined && np.workspace_id !== old.workspace_id) || (np.tab_id !== undefined && np.tab_id !== old.tab_id));
  const wasAgent = old && isAgent(old);
  const pt = np.screen_patch;
  if (pt) delete np.screen_patch;
  const p = old ? Object.assign(old, np, { key }) : { tokens: {}, screen: '', ...np, key };
  // Patches queue in order (several can arrive before the next frame); a full screen replaces them.
  if (typeof np.screen === 'string') { p.pq = null; p.patchBytes = 0; p.needsResync = false; }
  if (pt && !p.needsResync && !queuePatch(p, pt)) resync('screen patch queue overflow for ' + key);
  if (!old) { S.panes.set(key, p); structDirty = true; if (S.loaded && isAgent(p)) { log(p, 'started'); G.onNewPane(p); } }
  if (topologyChanged) structDirty = true;
  if (old && !wasAgent && isAgent(p)) G.onNewPane(p);
  noteStatus(p, before);
  schedule(key);
}
function removePane(key) {
  const p = S.panes.get(key);
  if (p) log(p, 'closed');
  S.panes.delete(key); S.selected.delete(key); S.queue.delete(key); S.finished.delete(key);
  S.hist.delete(key); S.alarmAt.delete(key); S.hidden.delete(key); S.pinned.delete(key); S.paneCols.delete(key);
  delete S.history[key]; if (p?.name) delete S.history[p.name];
  dirty.delete(key); G.forgetPane(key); P.forgetPane(key); Q.forgetPane(key);
  LS.set('hiddenPanes', [...S.hidden]); LS.set('pinned', [...S.pinned]); LS.set('history', S.history);
  const proofs = LS.get('proof', {}); delete proofs[key]; if (p?.name) delete proofs[p.name]; LS.set('proof', proofs);
  const c = cards.get(key); if (c) { c.closeMenu?.(); paintSet.delete(c.view); c.ac.abort(); c.root.remove(); cards.delete(key); }
  if (S.current === key) S.current = null;
  if (S.lastNotified === key) S.lastNotified = null;
  if (S.zoom === key) S.zoom = null;
  schedule();
}
export function applyState(st) {
  S.session = st.session || S.session;
  if (st.herdr) S.herdr = st.herdr;
  for (const prop of ['workspaces', 'tabs']) if (st[prop] && !sameValues(S[prop], st[prop])) { S[prop] = st[prop]; structDirty = true; }
  if (st.layouts && !sameValues(S.layouts, st.layouts)) {
    structDirty = true;
    S.layouts = st.layouts;
    // Real column count of each pane (Herdr rect width) for fit-to-width; refit only the cards that changed.
    const next = new Map();
    for (const l of S.layouts || []) for (const lp of l.panes || []) if (lp.rect?.width > 0) next.set(S.session + '|' + lp.pane_id, Math.round(lp.rect.width));
    for (const [k, c] of cards) if (next.get(k) !== S.paneCols.get(k)) c.view.setCols(next.get(k) || 0);
    S.paneCols = next;
  }
  if (st.profiles) S.profiles = st.profiles;
  if (st.lead) S.lead = st.lead;
  if (st.root) S.root = st.root;
  if (st.read_only != null) setSafe(!!st.read_only, false);
  if (st.error) toast('Backend: ' + st.error, 'error');
  if (st.panes) {
    S.awaitingState = false; S.restarted = false;
    const seen = new Set();
    for (const np of st.panes) { mergePane({ ...np, stale: !!np.stale || st.herdr?.running === false }); seen.add(np.key || (S.session + '|' + np.pane_id)); }
    for (const k of [...S.panes.keys()]) if (!seen.has(k)) removePane(k);
  }
  S.loaded = true; S.lastUpdate = Date.now();
  updateWriteState();
  if (!frame) frame = requestAnimationFrame(flush);
  sendVisible();
}

// status transitions: toasts, notifications, inbox, timeline, sparkline, queued prompts
function noteStatus(p, before) {
  const now = statusOf(p);
  if (before === now) return;
  const h = S.hist.get(p.key) || []; h.push({ s: now, t: Date.now() }); if (h.length > 40) h.shift(); S.hist.set(p.key, h);
  const lastAlarm = S.alarmAt.get(p.key) || 0;
  if (now === 'blocked' && Date.now() - lastAlarm < 30000) {
    // Same blocked episode (status flapping, API-REQUESTS R17): no second toast / notification.
  } else if (now === 'blocked') {
    S.alarmAt.set(p.key, Date.now());
    toast(`${nameOf(p)} is BLOCKED and needs you (the Deck never answers for you)`, 'blocked');
    P.notify(`${nameOf(p)} needs you`, lastLine(p)); S.lastNotified = p.key;
    log(p, 'blocked');
  } else if (before === 'working' && (now === 'idle' || now === 'done')) {
    log(p, 'finished'); S.finished.add(p.key); S.lastNotified = p.key;
    P.notify(`${nameOf(p)} finished`, lastLine(p));
  }
  if (now === 'working') S.finished.delete(p.key);
  P.inboxNote(p, before, now);
  if (now === 'idle' && S.queue.get(p.key)?.length) {
    const text = S.queue.get(p.key).shift();
    G.sendGuarded(p.key, text).then((r) => { if (!r && !G.isUncertain(p.key)) S.queue.get(p.key)?.unshift(text); });
  }
}
function log(p, what) { S.timeline.push({ t: Date.now(), name: nameOf(p), key: p.key, what }); if (S.timeline.length > 500) S.timeline.shift(); }
function lastLine(p) {
  const v = cards.get(p.key)?.view;
  const lines = (v ? v.plain : (p.screen || '')).split('\n').map((l) => l.trim()).filter(Boolean);
  return (lines[lines.length - 1] || '').slice(0, 120);
}
const plainOf = (key) => cards.get(key)?.view.plain ?? '';

// ---------- event stream: stays open while hidden (backend uses it as the window's heartbeat) ----------
let closeEvents = null, retry = 0;
function connect() {
  if (closeEvents) return;
  closeEvents = api.events(['hello', 'settings', 'state', 'panes', 'diff', 'herdr', 'error', 'result'], {
    onOpen() { S.connected = true; S.awaitingState = true; S.downSince = 0; retry = 0; Q.syncJobs(); updateWriteState(); schedule(); },
    onDown(err) { S.connected = false; S.awaitingState = true; S.downSince ||= Date.now(); S.downMsg = err.message; updateWriteState(); schedule(); },
    onEvent(name, d) {
      S.lastUpdate = Date.now();
      window.__deckPerf?.msg();
      if (name === 'hello') { S.clientId = d.client_id; lastVis = null; sendVisible(); }
      else if (name === 'settings') setSafe(!!d.read_only, false);
      else if (name === 'state') applyState(d);
      else if (name === 'panes' || name === 'diff') {
        for (const p of d.panes || []) mergePane(p);
        for (const k of d.removed || []) removePane(k);
        if (d.workspaces || d.tabs || d.layouts) applyState({ workspaces: d.workspaces, tabs: d.tabs, layouts: d.layouts });
      } else if (name === 'herdr') { S.herdr = d; updateWriteState(); schedule(); }
      else if (name === 'error') toast(d.message || 'Backend error', 'error');
      else if (name === 'result') Q.onResult(d);
      if (!S.connected) { S.connected = true; S.downSince = 0; updateWriteState(); schedule(); }
    },
  });
}
// A missed patch means our copy is stale: fetch one full state (at most one at a time).
let resyncing = null;
function resync(why) {
  if (resyncing) return;
  console.warn('resync:', why);
  resyncing = api.state().then(applyState, (e) => toast('Resync failed: ' + e.message, 'error')).finally(() => { resyncing = null; });
}
async function loadState() {
  try { applyState(await api.state()); }
  catch (e) {
    S.connected = false; S.downMsg = 'Backend not reachable: ' + e.message; schedule();
    setTimeout(loadState, Math.min(10000, 1000 * 2 ** retry++));
  }
}
document.addEventListener('visibilitychange', () => {
  if (document.hidden) { lastVis = ''; sendVisible(); P.pause(); }
  else { schedule(); sendVisible(); P.resume(); for (const c of cards.values()) c.hash = ''; for (const k of S.panes.keys()) schedule(k); }
});

// ---------- visible panes -> backend polls them faster ----------
let visTimer = 0, lastVis = '';
function sendVisible() {
  clearTimeout(visTimer);
  visTimer = setTimeout(() => {
    if (!S.clientId) return;
    const keys = document.hidden ? [] : visiblePanes().map((p) => p.key), sig = S.clientId + ':' + keys.join(',');
    if (sig === lastVis) return;
    lastVis = sig;
    api.visible(keys, S.clientId).catch((e) => { lastVis = ''; toast('Visible-panes update failed: ' + e.message, 'error'); });
  }, 150);
}

// ---------- rail: spaces, tabs (sidebar rows like Herdr's settings.toml) ----------
function setFilter(ws, tab) { S.filter = { ws, tab }; S.zoom = null; LS.set('filter', S.filter); schedule(); sendVisible(); }
function railItem(label, sub, status, active, onclick, cls = '') {
  return el('li', {}, el('button', { class: 'rail-item ' + cls + (active ? ' active' : ''), 'aria-current': active ? 'true' : null, onclick },
    el('span', { class: 'dot', 'data-s': status, 'aria-hidden': 'true' }),
    el('span', { class: 'rl' }, el('span', { class: 'rl1', text: label }), sub ? el('span', { class: 'rl2', text: sub }) : ''),
    el('span', { class: 'sr', text: ' status ' + status })));
}
function renderRail() {
  document.body.classList.toggle('no-rail', !S.rail);
  const nav = $('#rail-nav'); nav.textContent = '';
  const all = [...S.panes.values()];
  const ul = el('ul', { role: 'list' });
  ul.append(railItem(`All panes`, `${all.length} panes`, worst(all.map(statusOf)), !S.filter.ws, () => setFilter(null, null), 'all'));
  for (const w of S.workspaces) {
    const wp = all.filter((p) => p.workspace_id === w.workspace_id);
    const wt = w.tokens || {};
    const sub = [wt.branch || w.branch, wt.git_status || w.git_status].filter(Boolean).join(' ');
    ul.append(railItem(w.label || w.workspace_id, sub || `${wp.length} panes`, worst(wp.map(statusOf)),
      S.filter.ws === w.workspace_id && !S.filter.tab, () => setFilter(w.workspace_id, null), 'ws'));
    for (const t of S.tabs.filter((t) => (t.workspace_id || t.parent_workspace) === w.workspace_id)) {
      const tp = wp.filter((p) => p.tab_id === t.tab_id);
      ul.append(railItem(t.label || t.tab_id, '', worst(tp.map(statusOf)), S.filter.tab === t.tab_id, () => setFilter(w.workspace_id, t.tab_id), 'tab'));
      for (const p of tp.filter(isAgent)) {   // agent rows: state icon, name, model, ctx (Herdr sidebar rows)
        ul.append(el('li', {}, el('button', { class: 'rail-agent', onclick: () => { setFilter(w.workspace_id, t.tab_id); zoom(p.key); } },
          el('span', { class: 'dot', 'data-s': statusOf(p) }), el('span', { class: 'ra1', text: nameOf(p) }),
          el('span', { class: 'ra2', text: [tok(p).model, tok(p).ctx].filter(Boolean).join(' · ') }))));
      }
    }
  }
  nav.append(ul);
}
function railStep(d) {
  const items = [[null, null], ...S.workspaces.flatMap((w) => [[w.workspace_id, null],
    ...S.tabs.filter((t) => (t.workspace_id || t.parent_workspace) === w.workspace_id).map((t) => [w.workspace_id, t.tab_id])])];
  const i = items.findIndex(([w, t]) => w === S.filter.ws && t === S.filter.tab);
  const [w, t] = items[(i + d + items.length) % items.length];
  setFilter(w, t);
}

// ---------- main area: grid or Herdr's real split layout ----------
const paneOrder = (a, b) => (S.pinned.has(b.key) - S.pinned.has(a.key)) || a.pane_id.localeCompare(b.pane_id, undefined, { numeric: true });
function filteredPanes() {
  return [...S.panes.values()].filter((p) => !S.hidden.has(p.key) && (!S.filter.ws || p.workspace_id === S.filter.ws) &&
    (!S.filter.tab || p.tab_id === S.filter.tab)).sort(paneOrder);
}
function visiblePanes() { return S.zoom && S.panes.has(S.zoom) ? [S.panes.get(S.zoom)] : filteredPanes(); }
function layoutTab() {
  const tab = S.filter.tab || (S.filter.ws && (S.workspaces.find((w) => w.workspace_id === S.filter.ws)?.active_tab_id));
  return tab ? S.layouts.find((l) => l.tab_id === tab) : null;
}

function renderMain() {
  const grid = $('#grid'), empty = $('#empty');
  const down = !S.herdr.running && !S.panes.size;
  empty.hidden = !down && !(S.loaded && S.panes.size === 0);
  if (!empty.hidden) P.renderEmpty(empty, down);
  grid.hidden = !empty.hidden;
  for (const [k, c] of cards) if (!S.panes.has(k)) { c.ac.abort(); c.root.remove(); cards.delete(k); }
  const list = visiblePanes();
  const lay = S.view === 'layout' && !S.zoom ? layoutTab() : null;
  grid.classList.toggle('zoomed', !!S.zoom);
  grid.classList.toggle('herdr-layout', !!lay);
  grid.style.setProperty('--cols', S.zoom ? 1 : S.cols);
  for (const b of document.querySelectorAll('[data-view]')) b.setAttribute('aria-pressed', String(b.dataset.view === S.view));
  for (const b of document.querySelectorAll('[data-cols]')) b.setAttribute('aria-pressed', String(+b.dataset.cols === S.cols && S.view === 'grid'));
  const keep = new Set(list.map((p) => p.key));
  for (const [k, c] of cards) c.root.hidden = !keep.has(k);
  let prev = null;
  for (const p of list) {
    let c = cards.get(p.key);
    if (!c) { c = makeCard(p); cards.set(p.key, c); updateCard(c, p); }
    c.root.hidden = false;
    const want = prev ? prev.nextSibling : grid.firstChild;
    if (want !== c.root) grid.insertBefore(c.root, want);
    prev = c.root;
    c.root.style.cssText = '';
  }
  for (const h of grid.querySelectorAll('.divider')) h.remove();
  if (lay) Q.placeLayout(grid, lay, cards);
  const hiddenN = [...S.panes.keys()].filter((k) => S.hidden.has(k)).length;
  $('#grid-note').textContent = S.view === 'layout' && !lay && !S.zoom ? 'Herdr layout view: pick a tab in the left rail (or a space) to see its real splits.'
    : list.length === 0 && S.panes.size ? 'No panes in this view.' : hiddenN ? `${hiddenN} hidden pane(s) - Ctrl+K "Show hidden panes"` : '';
}

const KEYS = [['Esc', 'esc'], ['Ctrl+C', 'ctrl+c'], ['Enter', 'enter'], ['↑', 'up', 'Up'], ['↓', 'down', 'Down'], ['Tab', 'tab'], ['y', 'y'], ['n', 'n']];
const ibtn = (name, label, onclick, extra = {}) => el('button', { type: 'button', class: 'icon', 'aria-label': label, 'data-tip': label, onclick, ...extra }, icon(name));
function makeCard(p) {
  const key = p.key, c = { key, hash: '', ac: new AbortController(), pending: 0 };
  const pane = () => S.panes.get(key);
  c.sel = el('input', { type: 'checkbox', class: 'sel', 'aria-label': 'Select for broadcast (X)', 'data-tip': 'Select (X)',
    onchange: () => { c.sel.checked ? S.selected.add(key) : S.selected.delete(key); renderTop(); } });
  c.glyph = el('span', { class: 'gl' });
  c.title = el('button', { class: 'ttl', 'data-tip': 'Zoom (Z / double-click)', onclick: () => zoom(key) });
  c.chips = el('span', { class: 'chips' });
  c.ring = el('span', { class: 'ctx' });
  c.st = el('span', { class: 'st-chip' });
  c.sub = el('div', { class: 'card-sub' });
  c.spark = el('div', { class: 'spark', 'aria-hidden': 'true' });
  // No spacer: .tt grows, so the name gets all free width before it is cut.
  const head = el('header', { class: 'card-h', draggable: 'true' }, c.sel, c.glyph, el('div', { class: 'tt' }, el('div', { class: 'tt1' }, c.title, c.chips), c.sub),
    c.ring, c.st,
    ibtn('splitR', 'Split right (Ctrl+B V)', () => Q.split(pane(), 'right'), { 'data-w': '1' }),
    ibtn('splitD', 'Split down (Ctrl+B -)', () => Q.split(pane(), 'down'), { 'data-w': '1' }),
    c.zbtn = ibtn('max', 'Zoom (Z)', () => zoom(key)));
  head.addEventListener('dblclick', (ev) => { if (!ev.target.closest('button,input')) zoom(key); });
  Q.dragSwap(head, key);
  const scr = el('div', { class: 'screen', tabindex: '0', role: 'log', 'aria-live': 'off' });
  c.view = new ScreenView(scr, requestPaint, c.ac.signal); c.screenBox = scr;
  c.view.setCols(S.paneCols.get(key) || 0);
  scr.addEventListener('keydown', (ev) => P.rawKey(ev, c));
  c.err = el('div', { class: 'card-err', role: 'alert' });
  c.done = el('div', { class: 'card-done', hidden: true });
  c.ta = el('textarea', { rows: 1, class: 'pin', placeholder: 'Message · Enter sends', title: 'Enter sends · Shift+Enter new line · ↑ on an empty box = history', spellcheck: 'false', 'data-w': '1' });
  c.ta.addEventListener('keydown', (ev) => promptKeys(ev, c));
  c.ta.addEventListener('input', () => autosize(c.ta));
  c.attach = el('div', { class: 'attach' });
  c.form = el('form', { class: 'composer', onsubmit: (ev) => { ev.preventDefault(); sendPrompt(c); } },
    c.attach, el('div', { class: 'comp-row' }, c.ta,
      ibtn('task', 'Attach a task file', () => Q.attachTask(c)), ibtn('folder', 'Attach the folder', () => Q.attachChip(c, 'folder', pane().cwd)),
      ibtn('paste', 'Paste to pane (Ctrl+Shift+V)', () => P.pasteToPane(pane()), { 'data-w': '1' }),
      el('button', { type: 'submit', class: 'send', 'aria-label': 'Send (Enter)', 'data-tip': 'Send (Enter)', 'data-w': '1' }, icon('send'))));
  const keys = el('div', { class: 'keys', role: 'toolbar', 'aria-label': 'Keys' },
    KEYS.map(([label, k, name]) => el('button', { type: 'button', text: label, 'data-w': '1', class: 'key', 'aria-label': 'Send key ' + (name || label),
      onclick: () => act('Key ' + label, () => api.keys(key, [k]), `${name || label} sent to ${nameOf(pane())}`) })),
    el('span', { class: 'grow' }),
    ibtn('copy', 'Copy screen (Ctrl+Shift+C)', () => copyText(c.view.plain)),
    ibtn('edit', 'Rename (Ctrl+B Shift+P)', () => renamePane(key), { 'data-w': '1' }),
    ibtn('close', 'Close pane (Ctrl+B X)', () => closePane(key), { 'data-w': '1', class: 'icon danger-t' }),
    moreMenu(c));
  c.root = el('article', { class: 'card', 'data-key': key, onfocusin: () => { S.current = key; }, onmousedown: () => { S.current = key; } },
    head, c.spark, scr, c.err, c.done, c.form, keys);
  return c;
}
function moreMenu(c) {
  const key = c.key, p = () => S.panes.get(key);
  const item = (ic, label, fn, w) => el('button', { type: 'button', role: 'menuitem', 'data-w': w ? '1' : null,
    disabled: !!(w && (S.safe || S.writeBlocked || p()?.stale)), onclick: () => { if (!w || !(S.safe || S.writeBlocked || p()?.stale)) fn(); } }, icon(ic), label);
  const trigger = ibtn('more', 'More actions', () => { c.closeMenu?.(); c.closeMenu = openMenu(trigger, menu()); }, { 'aria-haspopup': 'menu', 'aria-expanded': 'false' });
  const menu = () => el('div', { class: 'menu', role: 'menu' },
      ...KEYS.map(([label, k]) => item('key', 'Send key ' + label, () => act('Key ' + label, () => api.keys(key, [k])), true)),
      item('splitR', 'Split right', () => Q.split(p(), 'right'), true),
      item('splitD', 'Split down', () => Q.split(p(), 'down'), true),
      item('copy', 'Copy screen', () => copyText(c.view.plain)),
      item('edit', 'Rename pane', () => renamePane(key), true),
      item('close', 'Close pane', () => closePane(key), true),
      item('stop', 'Stop agent', () => stopAgent(p()), true),
      item('eye', 'Focus in Herdr', () => act('Focus', () => api.paneFocus(key).catch((e) => (p().name ? api.focus(p().name) : api.tabFocus(p().tab_id)))), true),
      item('max', "Herdr's own zoom", () => act('Herdr zoom', () => api.paneZoom(key)), true),
      item('key', 'Type into pane (raw mode)', () => P.rawMode(c), true),
      item('clock', 'Queue prompt for when idle', () => queuePrompt(key), true),
      item('check', 'Proof checklist', () => G.proof(p())),
      item('arrowR', 'Send answer to a reviewer', () => Q.toReviewer(p()), true),
      item('git', 'Changes in this folder', () => P.changes(p())),
      item('doc', 'Export screen as Markdown', () => P.exportMd(p(), c.view.plain)),
      item('copy', 'Copy as prompt context', () => copyText(P.asContext(p(), c.view.plain))),
      item('pin', 'Pin / unpin', () => toggleSet(S.pinned, key, 'pinned')),
      item('eye', 'Hide this pane', () => toggleSet(S.hidden, key, 'hiddenPanes')));
  return trigger;
}
function toggleSet(set, key, name) { set.has(key) ? set.delete(key) : set.add(key); LS.set(name, [...set]); schedule(); sendVisible(); }
function autosize(ta) { ta.style.height = 'auto'; const h = ta.scrollHeight; ta.style.height = Math.min(h, 120) + 'px'; ta.classList.toggle('tall', h > 120); }

const chip = (text, cls = '') => el('span', { class: 'chip ' + cls, text });
function updateCard(c, p) {
  if (!p) return;
  // Screen first: the header, failure card and last line read the updated text.
  // A full snapshot sets the version even when its text is unchanged (a resync must repair the version).
  if (typeof p.screen === 'string') { c.view.set(p.screen); c.view.sv = p.screen_version; }
  if (p.pq) {
    const q = p.pq; p.pq = null; p.patchBytes = 0;
    for (const pt of q) if (!c.view.patch(pt)) { resync(`screen version ${c.view.sv} != ${pt.from_version} for ${p.pane_id}`); break; }
  }
  p.screen = undefined;   // the view owns the text now; keep no second copy per pane
  if (!S.paneCols.has(p.key)) c.view.fit();   // no layout width: guess from the text (throttled to 2 s)
  c.root.classList.toggle('stale', !!p.stale || !!S.writeBlocked);
  for (const n of c.root.querySelectorAll('[data-w]')) { const off = !!(S.safe || S.writeBlocked || p.stale); if (n.disabled !== off) n.disabled = off; }
  const s = statusOf(p), t = tok(p), name = nameOf(p), kind = kindOf(p);
  const hist = S.hist.get(p.key) || [];
  const h = [s, name, kind, JSON.stringify(t), p.cwd, p.title, p.error, S.queue.get(p.key)?.length || 0, S.selected.has(p.key), S.finished.has(p.key), hist.length, S.zoom === p.key, G.isUncertain(p.key)].join('|');
  if (h !== c.hash) {
    c.hash = h;
    c.root.dataset.s = s;
    c.glyph.replaceChildren(glyph(kind || (isAgent(p) ? '?' : 'shell')));
    c.title.textContent = name;
    c.title.setAttribute('aria-label', `${name}, ${s}. Zoom`);
    const effort = S.profiles.find((x) => x.name === p.name?.replace(/-\d+$/, ''))?.effort;
    c.chips.replaceChildren(...[t.model && chip(t.model, 'model'), effort && chip(effort, 'effort'), t.account && chip('acct ' + t.account, 'acct'),
      t.limit && chip(t.limit, /(9\d|100)%/.test(t.limit) ? 'hot' : ''), S.pinned.has(p.key) && chip('pinned'),
      S.queue.get(p.key)?.length && chip(`queued ${S.queue.get(p.key).length}`, 'q')].filter(Boolean));
    const pct = ctxPct(p);
    c.ring.replaceChildren(...(t.ctx || pct != null ? [ring(pct), el('span', { class: 'ctx-t' }, el('b', { text: (t.ctx || '').replace(/^ctx\s*/, '') || (pct + '%') }), t.out ? el('span', { text: 'out ' + t.out }) : '')] : []));
    c.ring.title = [t.ctx, t.in && 'in ' + t.in, t.out && 'out ' + t.out, t.cost].filter(Boolean).join(' · ');
    c.st.textContent = s; c.st.dataset.s = s; c.st.title = s;
    c.sub.textContent = [t.path || folder(p.cwd), p.title && p.title !== name ? p.title : '', t.in && 'in ' + t.in, t.cost].filter(Boolean).join('  ·  ');
    c.sub.title = c.sub.textContent;
    c.sel.checked = S.selected.has(p.key);
    c.ta.setAttribute('aria-label', 'Message for ' + name);
    // A card whose agent exited stays an agent card: its text is refused, never run in the shell left behind.
    if (isAgent(p)) c.everAgent = true;
    c.ta.placeholder = c.everAgent && !isAgent(p) ? 'Agent exited · messages are not sent to the shell' : isAgent(p) ? 'Message · Enter sends' : 'Shell command · Enter runs';
    c.screenBox.setAttribute('aria-label', 'Screen of ' + name);
    c.zbtn.replaceChildren(icon(S.zoom === p.key ? 'min' : 'max'));
    Q.spark(c.spark, hist);
    Q.doneStrip(c, p, S.finished.has(p.key) && s !== 'working');
    G.failureCard(c, p);
  }
}

// ---------- pane actions ----------
function zoom(key) {
  S.zoom = S.zoom === key ? null : key; S.current = key; schedule(); sendVisible();
  requestAnimationFrame(() => cards.get(key)?.title.focus());
}
async function sendPrompt(c) {
  const text = Q.composeText(c);
  if (!text.trim()) return;
  const p = S.panes.get(c.key);
  const hk = p.name || p.key, h = (S.history[hk] ||= []);
  if (h[h.length - 1] !== text) { h.push(text.slice(0, 32768)); if (h.length > 50) h.shift(); while (Object.keys(S.history).length > 100) delete S.history[Object.keys(S.history)[0]]; LS.set('history', S.history); }
  // Local-first: the box clears at once and a pending chip shows; a failure puts the text back.
  c.histI = null; c.ta.value = ''; Q.clearChips(c); autosize(c.ta);
  if (S.safe || S.writeBlocked || p.stale) { toast('Writes disabled: pane or connection is stale', 'warn'); c.ta.value = text; return; }
  Q.pending(c, +1);
  const r = await G.sendGuarded(c.key, text, !isAgent(p) && !c.everAgent ? 'shell' : 'agent');
  Q.pending(c, -1);
  // A failed send puts the text back; an UNKNOWN delivery does not (the failure card offers Resend).
  if (!r && !G.isUncertain(c.key) && !c.ta.value) { c.ta.value = text; autosize(c.ta); }
}
function promptKeys(ev, c) {
  if (ev.key === 'Enter' && !ev.shiftKey && !ev.isComposing) { ev.preventDefault(); sendPrompt(c); return; }
  const p = S.panes.get(c.key), h = S.history[p?.name || c.key] || [];
  const atStart = c.ta.selectionStart === 0 && c.ta.selectionEnd === 0;
  if (ev.key === 'ArrowUp' && h.length && (c.ta.value === '' || (c.histI != null && atStart))) {
    ev.preventDefault(); c.histI = c.histI == null ? h.length - 1 : Math.max(0, c.histI - 1); c.ta.value = h[c.histI]; autosize(c.ta);
  } else if (ev.key === 'ArrowDown' && c.histI != null) {
    ev.preventDefault(); c.histI++; if (c.histI >= h.length) { c.histI = null; c.ta.value = ''; } else c.ta.value = h[c.histI]; autosize(c.ta);
  }
}
async function renamePane(key) {
  const p = S.panes.get(key);
  const v = await ask('Rename pane', `New label for ${p.pane_id}`, p.label || nameOf(p));
  if (v != null && v.trim()) act('Rename', () => api.rename(key, v.trim()), 'Renamed to ' + v.trim());
}
async function closePane(key) {
  const p = S.panes.get(key);
  if (!p) return;
  if (S.safe) return toast('Safe mode is on: writes are disabled', 'warn');
  const ok = await confirm('Close pane?', `${p.pane_id}  ${nameOf(p)}\n${folder(p.cwd)}\n\n${isAgent(p) ? 'This stops the agent running in it.' : 'The shell in it closes.'}`, 'Close pane');
  if (ok) act('Close pane', () => api.close(key), `Closed ${nameOf(p)}`);
}
async function stopAgent(p) {
  if (!p?.name) return toast('This pane has no named agent; use Close instead', 'warn');
  if (await confirm('Stop agent?', `Stop ${p.name} (${p.pane_id})?\nOnly agents started by herdr-agents can be stopped this way.`, 'Stop agent')) act('Stop agent', () => api.stop(p.name), `Stopped ${p.name}`);
}
async function queuePrompt(key) {
  const v = await ask('Queue prompt', `Sent to ${nameOf(S.panes.get(key))} the next time it is idle (never when blocked)`);
  if (v && v.trim()) { if (!S.queue.has(key)) S.queue.set(key, []); S.queue.get(key).push(v.trim()); schedule(key); toast('Prompt queued'); }
}
const card = (key) => cards.get(key);
function currentKey() {
  const a = document.activeElement?.closest?.('.card');
  return a ? a.dataset.key : S.current && S.panes.has(S.current) ? S.current : filteredPanes()[0]?.key;
}

// ---------- top bar + status line (runs every frame while panes stream: write only what changed) ----------
const setText = (n, v) => { v = String(v); if (n && n.textContent !== v) n.textContent = v; };
const setAttr = (n, k, v) => { if (n.getAttribute(k) !== v) n.setAttribute(k, v); };
let lastCounts = '';
function renderTop() {
  const list = [...S.panes.values()];
  const blocked = list.filter((p) => statusOf(p) === 'blocked').length;
  const n = P.inboxCount();
  const ib = $('#btn-inbox'), inboxN = $('#inbox-n');
  setText(inboxN, n); if (inboxN.hidden !== !n) inboxN.hidden = !n;
  ib.classList.toggle('alarm', blocked > 0);
  setAttr(ib, 'aria-label', `Inbox, ${n} items, ${blocked} blocked`);
  setText($('#session'), MOCK ? 'mock' : (S.session.split(/[\\/]/).slice(-2, -1)[0] || S.session || ''));
  if ($('#session').title !== S.session) $('#session').title = S.session;
  if ($('#bcast').hidden !== (S.selected.size === 0)) $('#bcast').hidden = S.selected.size === 0;
  setText($('#bcast-n'), `${S.selected.size} selected`);
  const counts = {}; for (const p of list) { const s = statusOf(p); counts[s] = (counts[s] || 0) + 1; }
  const sig = JSON.stringify(counts);
  if (sig !== lastCounts) {
    lastCounts = sig;
    $('#counts').replaceChildren(...['blocked', 'working', 'idle', 'done'].filter((s) => counts[s]).map((s) =>
      el('span', { class: 'count', 'data-s': s, 'aria-label': `${counts[s]} ${s}` }, el('span', { class: 'dot', 'data-s': s }), String(counts[s]), el('span', { class: 'cw', text: ' ' + s }))));
  }
  const title = (blocked ? `(${blocked}!) ` : '') + 'Herdr Deck';
  if (document.title !== title) document.title = title;
  setText($('#health .hp'), `${list.length} panes, ${list.filter((p) => isAgent(p) && ['idle', 'done'].includes(statusOf(p))).length} idle agents`);
  for (const c of cards.values()) if (c.sel.checked !== S.selected.has(c.key)) c.sel.checked = S.selected.has(c.key);
}
function renderStatus() {
  const parts = [S.connected ? 'connected' : (S.downMsg || 'connecting...'), `${S.panes.size} panes`, S.safe && 'SAFE MODE (read-only)', MOCK && 'MOCK DATA'];
  setText($('#status-text'), parts.filter(Boolean).join('  ·  '));
  const ok = S.connected ? '1' : '0';
  if ($('#statusbar').dataset.ok !== ok) $('#statusbar').dataset.ok = ok;
  G.transportBanner(!S.writeBlocked && (S.connected || !S.loaded), S.restarted ? 'Deck stopped or restarted - reopen it from the Herdr launcher' : S.herdr.running === false ? 'Herdr offline - panes are stale; writes disabled' : S.downMsg);
}
let ageTimer = 0;
function ageTick() {
  if (document.hidden) { ageTimer = 0; return; }
  const age = S.lastUpdate ? Math.round((Date.now() - S.lastUpdate) / 1000) : null;
  $('#status-age').textContent = age == null ? '' : `updated ${age} s ago`;
  ageTimer = setTimeout(ageTick, 1000);
}
document.addEventListener('visibilitychange', () => { if (!document.hidden && !ageTimer) ageTick(); });

function setCols(n) { S.cols = n; S.view = 'grid'; S.zoom = null; LS.set('cols', n); LS.set('view', 'grid'); schedule(); }
function setView(v) { S.view = v; S.zoom = null; LS.set('view', v); schedule(); sendVisible(); }
export function setSafe(v, push = true) {
  const was = S.safe;
  if (!push && was === v) return;
  S.safe = v; if (v) P.clearRaw(); document.body.classList.toggle('safe', v); $('#safe').checked = v; updateWriteState(); schedule();
  if (push && was !== v) api.readOnly(v).then(() => toast(v ? 'Safe mode on: every write is refused' : 'Safe mode off', 'ok'))
    .catch((e) => { S.safe = was; document.body.classList.toggle('safe', was); $('#safe').checked = was; fail('Safe mode')(e); });
}
function updateWriteState() {
  S.writeBlocked = !!S.restarted || !!S.awaitingState || S.herdr.running === false || (S.loaded && !S.connected);
  document.body.classList.toggle('writes-blocked', S.writeBlocked);
  if (S.writeBlocked) P.clearRaw();
  for (const n of document.querySelectorAll('[data-w]')) { const off = !!(S.safe || S.writeBlocked); if (n.disabled !== off) n.disabled = off; }
  for (const [key, c] of cards) { c.root.classList.toggle('stale', !!S.panes.get(key)?.stale || S.writeBlocked); for (const n of c.root.querySelectorAll('[data-w]')) { const off = !!(S.safe || S.writeBlocked || S.panes.get(key)?.stale); if (n.disabled !== off) n.disabled = off; } }
}
let restartProbe = false;
setInterval(async () => {
  if (!S.downSince || Date.now() - S.downSince <= 10000 || restartProbe) return;
  restartProbe = true;
  try { await api.state(); }
  catch { S.restarted = true; updateWriteState(); schedule(); }
  finally { restartProbe = false; }
}, 2000);
const broadcastKeys = () => [...S.selected].filter((k) => S.panes.has(k));
function toggleRail() { S.rail = !S.rail; LS.set('rail', S.rail); schedule(); }

// ---------- global keys (Ctrl+B chords live in parity.js) ----------
const typing = (t) => t instanceof Element && !!(t.closest('input,textarea,select,[contenteditable]') || t.closest('dialog'));
document.addEventListener('keydown', (ev) => {
  if (ev.defaultPrevented) return;
  if (Q.chordKey(ev)) return;
  const k = ev.key, mod = ev.ctrlKey || ev.metaKey;
  if (mod && !ev.shiftKey && (k === 'k' || k === 'K')) { ev.preventDefault(); P.palette(); return; }
  if (mod && ev.shiftKey && (k === 'F' || k === 'f')) { ev.preventDefault(); Q.search(); return; }
  if (mod && ev.shiftKey && (k === 'V' || k === 'v')) { ev.preventDefault(); const key = currentKey(); if (key) P.pasteToPane(S.panes.get(key)); return; }
  if (mod && ev.shiftKey && (k === 'C' || k === 'c') && !window.getSelection().toString()) {
    const key = currentKey(); if (key) { ev.preventDefault(); copyText(cards.get(key).view.plain); } return;
  }
  if (ev.altKey && (k === 'ArrowUp' || k === 'ArrowDown')) { ev.preventDefault(); railStep(k === 'ArrowUp' ? -1 : 1); return; }
  if (S.raw || typing(ev.target) || mod || ev.altKey) return;
  if (k === 'Escape' && S.zoom) { zoom(S.zoom); return; }
  if (k === '?') { ev.preventDefault(); Q.keysOverlay(); }
  else if (k === '1' || k === '2' || k === '3') setCols(+k);
  else if (k === 'l' || k === 'L') setView(S.view === 'layout' ? 'grid' : 'layout');
  else if (k === ']' || k === '[' || k === 'j' || k === 'k') moveCard(k === ']' || k === 'j' ? 1 : -1);
  else if (k === 'z' || k === 'Z') { const key = currentKey(); if (key) zoom(key); }
  else if (k === '/' || k === 'p') { const key = currentKey(); if (key) { ev.preventDefault(); cards.get(key).ta.focus(); } }
  else if (k === 'i') P.inbox();
  else if (k === 'x') { const key = currentKey(); if (key) cards.get(key).sel.click(); }
});
function moveCard(d) {
  const list = visiblePanes(); if (!list.length) return;
  const i = list.findIndex((p) => p.key === currentKey());
  const p = list[(i + d + list.length) % list.length];
  S.current = p.key;
  if (S.zoom) { S.zoom = p.key; schedule(); sendVisible(); }
  requestAnimationFrame(() => cards.get(p.key)?.title.focus());
}

// ---------- boot ----------
function bindTop() {
  const on = (id, fn) => { $(id).onclick = fn; };
  on('#btn-agent', () => P.startAgent()); on('#btn-shell', () => P.newShell()); on('#btn-folder', () => Q.folderPicker(''));
  on('#btn-layout', () => Q.layoutBuilder()); on('#btn-board', () => Q.board()); on('#btn-team', () => P.team());
  on('#btn-usage', () => P.usage()); on('#btn-inbox', () => P.inbox()); on('#btn-palette', () => P.palette());
  on('#btn-help', () => Q.help()); on('#btn-search', () => Q.search()); on('#btn-agents', () => Q.agentsHub()); on('#btn-tools', () => Q.tools());
  on('#btn-rail', toggleRail);
  $('#safe').onchange = (ev) => setSafe(ev.target.checked);
  for (const b of document.querySelectorAll('[data-cols]')) b.onclick = () => setCols(+b.dataset.cols);
  for (const b of document.querySelectorAll('[data-view]')) b.onclick = () => setView(b.dataset.view);
  $('#bcast').onsubmit = (ev) => { ev.preventDefault(); P.broadcast($('#bcast-text')); };
  $('#bcast-compare').onclick = () => P.compare();
  $('#bcast-clear').onclick = () => { S.selected.clear(); for (const c of cards.values()) c.hash = ''; for (const k of S.panes.keys()) schedule(k); renderTop(); };
}
const ctx = { S, LS, act, schedule, statusOf, nameOf, kindOf, isAgent, lastLine, plainOf, zoom, closePane, stopAgent, card, currentKey,
  filteredPanes, visiblePanes, setCols, setView, setSafe, broadcastKeys, setFilter, toggleRail, railStep, moveCard, renamePane };
async function boot() {
  P.init(ctx); Q.init(ctx); G.init(ctx);
  if (new URLSearchParams(location.search).get('perf') === '1') import('./perf.js').catch((e) => toast('Perf overlay failed: ' + e.message, 'error'));
  P.applyLook();
  bindTop();
  try { await initApi(); } catch (e) { toast('Mock failed to load: ' + e.message, 'error'); return; }
  window.addEventListener('error', (e) => toast('Page error: ' + e.message, 'error'));
  window.addEventListener('unhandledrejection', (e) => toast('Page error: ' + (e.reason?.message || e.reason), 'error'));
  try {
    const r = await api.prefs();
    // Include legacy local settings so the next whole-object write preserves them.
    try { for (let i = 0; i < localStorage.length; i++) { const key = localStorage.key(i); if (key?.startsWith('deck.') && key !== 'deck.mock-prefs') { try { prefs[key.slice(5)] = JSON.parse(localStorage.getItem(key)); } catch { /* invalid legacy preference */ } } } } catch { /* local cache unavailable: backend preferences still apply */ }
    Object.assign(prefs, r.prefs || {});
    for (const [k, v] of Object.entries(prefs)) { try { localStorage.setItem('deck.' + k, JSON.stringify(v)); } catch { /* blocked local cache */ } }
    Object.assign(S, { filter: LS.get('filter', S.filter), cols: LS.get('cols', S.cols), view: LS.get('view', S.view), rail: LS.get('rail', S.rail), history: capHistory(LS.get('history', S.history)), hidden: new Set(LS.get('hiddenPanes', [])), pinned: new Set(LS.get('pinned', [])) });
    P.applyLook(); prefsReady = true;
  } catch (e) { toast('Preferences unavailable: ' + e.message, 'warn'); }
  Q.syncJobs();
  await loadState();
  connect();
  P.resume(); ageTick();
  schedule();
}
boot();
