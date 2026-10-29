// "BUILD NOW" rows of tasks/deck-feature-research.md: task readiness check, file reservations,
// bounded launch waves, helper account guard, research route guard, proof checklist, scope drift
// alarm, uncertain delivery guard, failure recovery cards, research citation audit.
// Rules: warn, never invent; never answer a blocked agent; never resend or revert by itself.
import { api, missing } from './api.js';
import { el, $, toast, fail, dialog, confirm, deliveryUnknown } from './ui.js';
import { icon } from './icons.js';
import * as P from './panels.js';

let X;
export function init(ctx) { X = ctx; }
const field = (label, input) => el('label', { class: 'field' }, el('span', { text: label }), input);
const mark = (ok, warn) => el('span', { class: 'pmark ' + (ok ? 'ok' : warn ? 'warn' : 'bad'), 'aria-hidden': 'true' }, icon(ok ? 'check' : 'x'));

// ---------- row 1: task readiness check (reads the task text; never fills gaps) ----------
const CHECKS = [
  ['Allowed files / scope', /^#+.*\b(scope|files?)\b|write only|change only|files it may write|allowed files/im],
  ['Answer file or format', /^#+.*answer|answer format|answer file|write (the|your) answer|DONE \| BLOCKED/im],
  ['Completion checks', /^#+.*(definition of done|checks|acceptance)|must pass|exits 0/im],
  ['Stop conditions', /\bBLOCKED\b|stop (when|if)|do not loop|stop condition/im],
];
export function readiness(text) {
  const lines = text.split('\n');
  return CHECKS.map(([name, re]) => {
    const i = lines.findIndex((l) => re.test(l));
    return { name, ok: i >= 0, line: i >= 0 ? lines[i].trim().slice(0, 120) : '' };
  });
}
// Paths the task declares (only from its scope section when it has one).
export function declaredPaths(text) {
  const sec = /^#+.*\b(scope|files?)\b.*$/im.exec(text);
  let part = text;
  if (sec) { const rest = text.slice(sec.index + sec[0].length); const end = rest.search(/^#+ /m); part = end > 0 ? rest.slice(0, end) : rest; }
  const out = new Set();
  for (const m of part.matchAll(/`([^`\s]+)`|(?:^|\s)((?:[\w.-]+\/)+[\w.*-]+|[\w-]+\.(?:ps1|md|js|mjs|go|css|html|json|toml|txt))(?=[\s,.;)]|$)/gm)) {
    const p = (m[1] || m[2] || '').replace(/\\/g, '/').replace(/^\.\//, '').toLowerCase();
    if (p && /[./]/.test(p) && !/^https?:/.test(p) && !p.startsWith('%')) out.add(p);
    if (out.size >= 40) break;
  }
  return [...out];
}
const isResearch = (file, text) => /research/i.test(file) || /^#.*research/im.test(text);

// ---------- row 3: bounded launch waves (frontend admission control) ----------
const limits = () => X.LS.get('limits', { maxAgents: 4, ramMaxPct: 90, diskMinGb: 5 });
let health = null;
export function setHealth(h) { health = h; drainQueue(); }
const activeAgents = () => [...X.S.panes.values()].filter((p) => (p.name || p.agent) && ['working', 'blocked', 'idle'].includes(X.statusOf(p))).length;
function capacity(cwd = X.S.panes.get(X.currentKey())?.cwd || X.S.root) {
  const L = limits(), why = [];
  const n = activeAgents();
  if (n >= L.maxAgents) why.push(`${n} active agents (limit ${L.maxAgents})`);
  if (!health) why.push('RAM/disk unknown (health not read yet)');
  else {
    if (health.ram_used_pct == null) why.push('RAM unknown');
    if (health.ram_used_pct > L.ramMaxPct) why.push(`RAM ${health.ram_used_pct}% used (floor: at most ${L.ramMaxPct}%)`);
    const drive = /^[a-z]:/i.exec(cwd || '')?.[0];
    const c = (health.disks || []).find((d) => d.drive?.toLowerCase() === drive?.toLowerCase());
    if ((!c || c.free_gb == null) && L.diskMinGb > 0) why.push('Disk capacity unknown');
    if (c && c.free_gb != null && +c.free_gb < L.diskMinGb) why.push(`${c.drive} ${(+c.free_gb).toFixed(1)} GB free (floor ${L.diskMinGb} GB)`);
  }
  return { ok: !why.length, why, n };
}
const queue = [];
export const unattachedLaunches = [];
let draining = false;
export async function drainQueue() {
  if (draining) return;
  draining = true;
  try { while (queue.length && capacity(queue[0].cwd).ok && !X.S.safe && !X.S.writeBlocked) {
    const job = queue.shift();
    toast(`Launch wave: starting held launch "${job.label}"`, 'info');
    await job.run();
  }
  } finally { draining = false; }
  if (queue.length) X.schedule();
}
export const queued = () => queue.slice();
export function enqueueStart(label, run, cwd) { if (queue.length >= 32) throw new Error('Launch queue is full'); queue.push({ label, run, cwd }); return drainQueue(); }
export function queuePanel() {
  P.drawer('queue', 'Launch queue', (b) => {
    const cap = capacity(), L = limits();
    const max = el('input', { type: 'number', min: 1, max: 12, value: L.maxAgents, 'aria-label': 'Max active agents' });
    const ram = el('input', { type: 'number', min: 50, max: 99, value: L.ramMaxPct, 'aria-label': 'Max RAM used %' });
    const disk = el('input', { type: 'number', min: 0, max: 500, value: L.diskMinGb, 'aria-label': 'Min free project drive GB' });
    b.append(el('p', { class: 'muted small', text: 'New launches wait here when the limit or a RAM/disk floor is reached. Running work is never stopped to make room.' }),
      el('div', { class: 'form2' }, field('Max active agents', max), field('Max RAM used %', ram), field('Min free project drive (GB)', disk)),
      el('button', { text: 'Save limits', onclick: () => { X.LS.set('limits', { maxAgents: +max.value || 4, ramMaxPct: +ram.value || 90, diskMinGb: +disk.value || 0 }); drainQueue(); P.refreshOpen(true); } }),
      el('p', {}, mark(cap.ok), ' ', cap.ok ? `Capacity free (${cap.n} active)` : 'Holding: ' + cap.why.join('; ')),
      el('ul', { class: 'list' }, queue.map((j, i) => el('li', { class: 'li' }, el('span', { class: 'grow', text: j.label }),
        el('button', { text: 'Cancel', onclick: () => { queue.splice(i, 1); P.refreshOpen(true); } })))),
      ...unattachedLaunches.map((r) => el('p', { class: 'warn', text: `Task ${r.task}: attach manually (start returned no pane_id)` })),
      queue.length ? '' : el('p', { class: 'muted', text: 'Nothing held.' }));
  }, true);
}

// ---------- row 4: helper account guard ----------
function accountGuard(profile, account, override) {
  const kind = X.S.profiles.find((p) => p.name === profile)?.kind || profile;
  if (kind !== 'claude' && profile !== 'next') return { ok: true };
  if (profile === 'next') return { ok: true, note: 'The Deck picks the least-used Claude account that is not the lead\'s (refused when none is left)' };
  const lead = X.S.lead;
  if (!account) return { ok: false, why: 'A Claude helper needs an explicit account (not the lead\'s).' };
  if (lead && lead.known && lead.account === account) return { ok: false, why: `Account ${account} is the lead's own account.` };
  if (!lead || !lead.known) return override ? { ok: true, note: 'Lead account unknown; owner confirmed it is a different account.' }
    : { ok: false, unknown: true, why: 'The lead\'s account is unknown (backend R20), so this launch is held.' };
  return { ok: true, note: `Lead uses ${lead.account || 'the default folder'}; helper uses ${account}.` };
}

// ---------- launch dialog (rows 1-5 together) ----------
export async function launch(file) {
  let tasks = [];
  try { tasks = (await api.tasks()).tasks || []; } catch (e) { return fail('Tasks')(e); }
  const S = X.S, launchOwner = 'launch-' + Date.now() + '-' + Math.random().toString(36).slice(2, 8);
  let existingReservations = [];
  const taskSel = el('select', { 'aria-label': 'Task file' }, tasks.map((t) => el('option', { value: t.file, text: t.file, selected: t.file === file })));
  const agents = [...S.panes.values()].filter((p) => p.name || p.agent);
  const target = el('select', { 'aria-label': 'Target' }, el('option', { value: 'new', text: 'New agent…' }),
    agents.map((p) => el('option', { value: p.key, text: `${X.nameOf(p)} (${X.statusOf(p)})` })), el('option', { value: 'headless', text: 'Headless research run' }));
  const prof = el('select', { 'aria-label': 'Profile' }, S.profiles.map((p) => el('option', { value: p.name, text: `${p.name} (${p.kind})` })));
  const account = el('input', { 'aria-label': 'Account', placeholder: 'e.g. 1984', size: 8, spellcheck: false });
  const override = el('input', { type: 'checkbox' });
  const accept = el('input', { type: 'checkbox' });
  const report = el('div', { class: 'proof-list', 'aria-live': 'polite' });
  const newBox = el('div', { class: 'form2' }, field('Profile', prof), field('Account (Claude helpers)', account));
  let text = '', paths = [], conflicts = [], routes = null, verdict = { ok: false };
  const paint = async () => {
    newBox.hidden = target.value !== 'new';
    const research = isResearch(taskSel.value, text);
    const items = [];
    for (const r of readiness(text)) items.push([r.ok, true, `${r.name}: ${r.ok ? 'found' : 'missing - the agent may guess; fix the task first'}`, r.line]);
    items.push([paths.length > 0, true, `Declared files: ${paths.length ? paths.slice(0, 8).join(', ') + (paths.length > 8 ? ` +${paths.length - 8}` : '') : 'none found'}`]);
    items.push([!conflicts.length && !conflicts.unavailable, false, conflicts.length ? `Overlap with active reservations: ${conflicts.map((c) => `${c.path} (${c.task}, ${c.owner})`).join('; ')}` : (conflicts.unavailable ? 'reservations unavailable' : 'No overlap with active reservations')]);
    const cap = capacity();
    items.push([cap.ok, true, cap.ok ? `Capacity: ${cap.n} active agents, within limits` : 'Launch wave: will be HELD - ' + cap.why.join('; ')]);
    let block = '';
    if (research) {
      const ok = target.value === 'headless' && routes && routes.some((r) => r.verified && r.profile === prof.value);
      items.push([ok, false, ok ? 'Research route: verified headless server-search route' : target.value !== 'headless'
        ? 'Research route: research must run headless (no pane, no browser) - pick "Headless research run"'
        : routes ? `Research route: ${prof.value} is not a verified headless route (verified: ${routes.filter((r) => r.verified).map((r) => r.profile).join(', ') || 'none'})` : 'Research route: unavailable (backend R22 not answered)']);
      if (!ok) block = 'Research needs a verified headless route.';
    }
    if (target.value === 'new') {
      const g = accountGuard(prof.value, account.value.trim(), override.checked);
      items.push([g.ok, false, g.ok ? 'Account guard: ' + (g.note || 'not a Claude helper') : 'Account guard: ' + g.why]);
      if (!g.ok) block = block || g.why;
      override.parentElement.hidden = !g.unknown && !override.checked;
    } else override.parentElement.hidden = true;
    if (conflicts.unavailable) block = 'reservations unavailable';
    if (conflicts.length && !accept.checked) block = block || 'File overlap: tick "I accept the overlap" or pick another task.';
    accept.parentElement.hidden = !conflicts.length;
    verdict = { ok: !block, block, hold: !cap.ok, research };
    report.replaceChildren(...items.map(([ok, soft, t, line]) => el('div', { class: 'prow' }, mark(ok, soft), el('div', {}, el('div', { text: t }), line ? el('div', { class: 'mono small muted clip', text: line }) : ''))));
  };
  const load = async () => {
    report.textContent = 'Checking…';
    try { text = (await api.taskRead(taskSel.value)).text || ''; } catch (e) { text = ''; report.textContent = 'Task: ' + e.message; return; }
    paths = declaredPaths(text);
    conflicts = [];
    try {
      const ledger = await api.reservations();
      if (ledger.load_error) throw new Error('reservations unavailable: ' + ledger.load_error);
      const r = ledger.reservations || []; existingReservations = r;
      const norm = (x) => { const p = x.toLowerCase().replace(/\\/g, '/').replace(/\/\*\*$|\/+$/g, ''); const root = (S.root || '').toLowerCase().replace(/\\/g, '/').replace(/\/+$/, ''); return root && p.startsWith(root + '/') ? p.slice(root.length + 1) : p; };
      const overlap = (a, b) => a === b || a.startsWith(b + '/') || b.startsWith(a + '/');   // same rule as the backend (R19)
      for (const res of r) for (const p of res.paths) { const hit = paths.find((x) => overlap(norm(x), norm(p))); if (hit) conflicts.push({ path: hit, task: res.task, owner: res.owner }); }
    } catch (e) { conflicts.unavailable = true; if (!missing(e)) toast('Reservations: ' + e.message, 'error'); }
    if (routes === null) { try { routes = (await api.headlessRoutes()).routes || []; } catch { routes = undefined; } }
    paint();
  };
  for (const n of [target, prof, account, override, accept]) n.addEventListener('input', paint);
  taskSel.onchange = load;
  load();
  const v = await dialog({ title: 'Launch a task', wide: true,
    body: el('div', { class: 'launch' }, el('div', { class: 'form2' }, field('Task', taskSel), field('Send to', target)), newBox,
      el('h3', { text: 'Readiness' }), report,
      el('label', { class: 'small' }, override, ' I confirm this is NOT the lead\'s Claude account'),
      el('label', { class: 'small' }, accept, ' I accept the file overlap (two writers on the same files)')),
    actions: [{ label: 'Cancel', value: null }, { label: 'Launch', value: 'ok', kind: 'primary' }],
    check: () => (verdict.ok ? '' : verdict.block || 'Still checking…') });
  if (v !== 'ok') return;
  const chosen = taskSel.value, prompt = `Read CLAUDE.md and AGENTS.md, then do the task in ${chosen}`;
  const destination = target.value, profile = prof.value, acct = account.value.trim(), overlap = accept.checked;
  const cur = S.panes.get(X.currentKey()), pathsCopy = paths.slice();
  const owner = destination === 'new' || destination === 'headless' ? profile + ' ' + launchOwner : X.nameOf(S.panes.get(destination));
  const run = () => dispatchTask({ destination, profile, acct, overlap, cur, chosen, pathsCopy, owner, prompt, existingIds: existingReservations.map((r) => r.id) });
  if (verdict.hold && destination === 'new') {
    if (queue.length >= 32) return toast('Launch queue is full', 'warn');
    queue.push({ label: `${chosen} -> ${owner}`, run, cwd: cur?.cwd || S.root }); toast('Held in the launch queue (limits/floors)', 'warn'); queuePanel();
  } else if (destination === 'new') await enqueueStart(chosen, run, cur?.cwd || S.root);
  else await run();
}
export async function dispatchTask({ destination, profile, acct, overlap, cur, chosen, pathsCopy, owner, prompt, existingIds = [] }) {
  const S = X.S;
    if (S.safe || S.writeBlocked) return;
    let reservation;
    const releaseNew = async () => { if (reservation?.id && !existingIds.includes(reservation.id)) await api.release(reservation.id); };
    try {
      reservation = await api.reserve(chosen, owner, pathsCopy, overlap);
    } catch (e) {
      const conflicts = (e.body?.conflicts || []).map((c) => `${c.path} (${c.owner})`).join('; ');
      toast((e.status === 503 ? 'reservations unavailable' : 'Reservation refused: ' + e.message) + (conflicts ? ' - ' + conflicts + '. Reopen Launch and accept overlap to proceed.' : ''), 'error'); return;
    }
    const rec = await prepareLaunch({ task: chosen, paths: pathsCopy, at: Date.now(), prompt }, destination === 'new' ? cur?.cwd || S.root : S.panes.get(destination)?.cwd || S.root);
    try {
      if (S.safe || S.writeBlocked || (destination !== 'new' && destination !== 'headless' && S.panes.get(destination)?.stale)) { await releaseNew(); return; }
      if (destination === 'headless') { await api.headless(profile, prompt); toast('Research started headless', 'ok'); }
      else if (destination === 'new') {
        const r = await api.start({ profile, model: '', effort: '', account: acct, cwd: cur?.cwd || S.root, key: cur?.key || '' });
        if (r.pane_id) {
          const key = [...S.panes.values()].find((p) => p.pane_id === r.pane_id)?.key || S.session + '|' + r.pane_id;
          launches.set(key, rec); X.schedule(key);
          toast(`Task ${chosen} attached to ${r.pane_id}. Send it from Proof when ready.`, 'info', 9000);
        } else { unattachedLaunches.push(rec); if (unattachedLaunches.length > 100) unattachedLaunches.shift(); toast(`Task ${chosen}: attach manually (start returned no pane_id); see Launch queue`, 'warn', 12000); }
      } else {
        launches.set(destination, rec);
        const r = await sendGuarded(destination, prompt);
        if (!r) { if (!isUncertain(destination)) { launches.delete(destination); await releaseNew(); } return; }
        rec.sent = true;
      }
    } catch (e) {
      if (!deliveryUnknown(e) && reservation?.id) {
        try { await releaseNew(); } catch (releaseError) { fail('Release reservation')(releaseError); }
      }
      fail('Launch')(e);
    }
}
export const launches = new Map();
export async function prepareLaunch(rec, repo) {
  rec.scopeId = 'deck-' + Date.now() + '-' + Math.random().toString(36).slice(2);
  try { await api.scopeBaseline(rec.scopeId, repo, rec.paths); rec.baseline = true; }
  catch (e) { rec.baseline = false; toast('Scope baseline unknown: ' + e.message, 'warn'); }
  return rec;
}
export function onNewPane(p) { if (launches.has(p.key)) X.schedule(p.key); }
export function forgetPane(key) { launches.delete(key); uncertain.delete(key); }

// ---------- row 8: uncertain delivery guard ----------
const uncertain = new Map();
export async function sendGuarded(key, text, expect) {
  if (X.S.safe || X.S.writeBlocked || X.S.panes.get(key)?.stale) { toast('Writes disabled: pane or connection is stale', 'warn'); return null; }
  try {
    const r = await api.prompt(key, text, expect);
    uncertain.delete(key);
    toast(`Sent to ${X.nameOf(X.S.panes.get(key) || {})}`, 'ok');
    return r || {};
  } catch (e) {
    // The owner is typing in that agent's box: nothing was sent, and nothing is retried.
    // ... or its agent has exited: nothing was sent to the shell left behind.
    if (e.code === 'input_unknown' || e.code === 'input_not_empty' || e.code === 'agent_exited') toast(e.message, 'warn', 9000);
    else if (deliveryUnknown(e)) {
      uncertain.set(key, { text, at: Date.now(), err: e.body?.detail || e.message, dispatch: e.dispatch_id });
      X.schedule(key);
      toast(`Delivery UNKNOWN${e.dispatch_id ? ' (' + e.dispatch_id + ')' : ''}: the prompt may or may not have arrived. Check the screen before resending.`, 'warn', 12000);
    } else fail('Prompt')(e);
    return null;
  }
}
export const isUncertain = (key) => uncertain.has(key);

// ---------- row 9: failure recovery cards ----------
function classify(p, c) {
  const s = X.statusOf(p), tail = (c.view.plain.slice(-1500) || '').toLowerCase();
  if (uncertain.has(p.key)) { const u = uncertain.get(p.key); return { kind: 'Delivery unknown' + (u.dispatch ? ' · ' + u.dispatch : ''), text: `Sent at ${new Date(u.at).toLocaleTimeString()}: "${u.text.slice(0, 80)}" - ${u.err}`, step: 'Read the screen; resend only if it did not arrive.', resend: true }; }
  if (s === 'blocked') {
    const k = /sign in|log in|login|not logged|authenticat/.test(tail) ? 'Sign-in needed' : /trust/.test(tail) ? 'Folder trust question'
      : /allow|approve|permission|\(y\/n\)|y\/n/.test(tail) ? 'Approval needed' : 'Question for you';
    return { kind: 'Owner decision: ' + k, text: X.lastLine(p), step: 'Zoom the pane and decide yourself. The Deck never answers.', zoom: true };
  }
  if (p.error) {
    const start = /start|launch|exited with code|not found|cannot find|no such/i.test(p.error);
    return { kind: start ? 'Startup failure' : 'Agent / backend error', text: p.error, step: start ? 'Check the profile and folder, then start again from + Agent.' : 'Read the error; the cause is unknown until checked.' };
  }
  if (s === 'error') return { kind: 'Agent error', text: X.lastLine(p), step: 'Read the screen; cause unknown until checked.' };
  return null;
}
export function failureCard(c, p) {
  const f = classify(p, c);
  c.err.hidden = !f;
  if (!f) return;
  c.err.replaceChildren(el('div', { class: 'fail-k' }, icon(f.zoom ? 'bell' : 'x'), f.kind), el('div', { class: 'fail-t mono', text: f.text }),
    el('div', { class: 'fail-s' }, el('span', { text: 'Next: ' + f.step }),
      f.zoom ? el('button', { type: 'button', onclick: () => { X.zoom(p.key); requestAnimationFrame(() => c.screenBox.focus()); } }, 'Open pane') : '',
      f.resend ? el('button', { type: 'button', onclick: () => { c.view.scrollToLine(c.view.lines - 1); } }, 'Show screen end') : '',
      f.resend ? el('button', { type: 'button', 'data-w': '1', onclick: async () => {
        const u = uncertain.get(p.key);
        if (u && await confirm('Resend prompt?', `The first send may have arrived. Resend this to ${X.nameOf(p)} anyway?\n\n${u.text}`, 'Resend', false)) sendGuarded(p.key, u.text);
      } }, 'Resend…') : '',
      f.resend ? el('button', { type: 'button', onclick: () => { uncertain.delete(p.key); X.schedule(p.key); } }, 'It arrived') : ''));
}
let bannerKey = null;
export function transportBanner(connected, msg) {
  const key = connected ? '' : msg || '-';
  if (key === bannerKey) return;   // called every frame: rebuild only when the state changes
  bannerKey = key;
  const b = $('#banner');
  b.hidden = connected;
  if (!connected) b.replaceChildren(el('strong', { text: 'Transport failure: ' }), el('span', { text: (msg || 'event stream lost') + ' - screens may be stale. The Deck reconnects by itself.' }),
    el('button', { onclick: () => location.reload() }, 'Reload page'));
}

// ---------- rows 6, 7, 10: proof panel ----------
const SLOTS = [['checks', 'Checks run', /\b(go test|go vet|pytest|npm (run )?test|check\.ps1|tests? pass|checks?:)/i],
  ['oldfail', 'Fails on the old code', /old code|fail(s|ed)? on (the )?(old|previous|head)|against HEAD|git show HEAD/i],
  ['mutation', 'Mutation check', /mutation|mutant|deliberately broken/i],
  ['review', 'Independent review', /review(ed)? by|reviewer|independent review/i],
  ['live', 'Isolated live test', /lab session|isolated (herdr )?session|--session lab|live test/i]];
function gitPaths(g) {
  if (g.paths) return new Set(g.paths.map((x) => x.path.toLowerCase()));
  return new Set((g.status || g.text || '').split('\n').map((l) => l.slice(3).trim().replace(/\\/g, '/').replace(/^"|"$/g, '').toLowerCase()).filter(Boolean));
}
export function audit(text) {
  const urls = [...new Set((text.match(/https?:\/\/[^\s)>\]"'`]+/g) || []).map((u) => u.replace(/[.,;]+$/, '')))];
  const OFFICIAL = /(^|\.)(microsoft\.com|github\.com|docs\.[\w.-]+|developer\.[\w.-]+|developers\.[\w.-]+|learn\.[\w.-]+|anthropic\.com|openai\.com|google\.com|mozilla\.org|w3\.org|whatwg\.org|python\.org|go\.dev|nodejs\.org|herdr\.dev)$/i;
  return urls.map((u) => { let host = ''; try { host = new URL(u).hostname; } catch { /* malformed url */ } return { url: u, host, official: OFFICIAL.test(host) }; });
}
function auditBlock(text) {
  const a = audit(text);
  return el('div', { class: 'proof-list' },
    el('div', { class: 'prow' }, mark(a.length > 0, true), el('div', { text: a.length ? `${a.length} cited URL(s); every claim stays NOT VERIFIED until checked` : 'No citations found in this text' })),
    ...a.slice(0, 40).map((x) => el('div', { class: 'prow' }, mark(x.official, true), el('div', {}, el('div', { class: 'mono small clip', text: x.url }),
      el('div', { class: 'small muted', text: (x.official ? 'official-looking domain' : 'NOT an official domain - check the source') + ' · not verified' })))));
}
export { auditBlock };
export function proof(p) {
  const key = p.key, store = X.LS.get('proof', {}), id = p.name || key, rec = store[id] || {};
  P.drawer('proof', 'Proof - ' + X.nameOf(p), async (b) => {
    const plain = X.plainOf(key), L = launches.get(key);
    b.append(el('p', { class: 'muted small', text: 'A green agent state is not proof. Each slot stays "not verified" until you (or the lead) mark it with evidence. Agent claims are shown, never trusted.' }));
    if (L && L.prompt && !L.sent) b.append(el('button', { class: 'primary', 'data-w': '1', onclick: async () => { if (await sendGuarded(key, L.prompt)) { L.sent = true; P.refreshOpen(true); } } }, icon('send'), ' Send task ' + L.task));
    for (const [slot, label, re] of SLOTS) {
      const r = rec[slot] || { state: 'not verified', note: '' };
      const claim = plain.split('\n').reverse().find((l) => re.test(l));
      const sel = el('select', { 'aria-label': label + ' state', onchange: () => save(slot, sel.value, note.value) },
        ['not verified', 'verified', 'failed'].map((s) => el('option', { value: s, text: s, selected: r.state === s })));
      const note = el('input', { 'aria-label': label + ' evidence', value: r.note, placeholder: 'evidence: command + result', onchange: () => save(slot, sel.value, note.value) });
      b.append(el('section', { class: 'pslot', 'data-drawer-key': slot, 'data-state': r.state }, el('div', { class: 'prow' }, mark(r.state === 'verified', r.state === 'not verified'), el('strong', { text: label }), el('span', { class: 'grow' }), sel), note,
        claim ? el('div', { class: 'small muted' }, 'Agent claims: ', el('span', { class: 'mono', text: claim.trim().slice(0, 140) })) : ''));
    }
    // row 7: scope drift
    const drift = el('div', { class: 'proof-list', text: 'Reading git status…' });
    b.append(el('h3', { text: 'Scope drift' }), drift);
    try {
      if (!L?.baseline) { drift.textContent = 'Scope drift unknown: no successful pre-dispatch baseline'; }
      else {
        const result = await api.scopeCheck(L.scopeId), extra = result.unexpected || [];
        drift.replaceChildren(el('p', { text: `Task ${L.task}: ${extra.length ? extra.length + ' change(s) outside declared files' : 'No changes outside declared files'}` }),
          el('p', { class: 'muted', text: result.warning || 'Concurrent changes are not blamed or reverted.' }),
          ...extra.slice(0, 30).map((path) => el('div', { class: 'mono small', text: path })));
      }
    } catch (e) { drift.textContent = 'Scope drift: ' + (missing(e) ? 'git view unavailable' : e.message); }
    b.append(el('h3', { text: 'Citation audit (this screen)' }), auditBlock(plain));
  });
  function save(slot, state, note) { const s = X.LS.get('proof', {}); (s[id] ||= {})[slot] = { state, note, at: Date.now() }; X.LS.set('proof', s); }
}
