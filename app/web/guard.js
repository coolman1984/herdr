// "BUILD NOW" rows of tasks/deck-feature-research.md: task readiness check, file reservations,
// bounded launch waves, helper account guard, research route guard, proof checklist, scope drift
// alarm, uncertain delivery guard, failure recovery cards, research citation audit.
// Rules: warn, never invent; never answer a blocked agent; never resend or revert by itself.
import { api } from './api.js';
import { el, $, toast, fail, dialog, confirm, copyText } from './ui.js';
import { icon } from './icons.js';
import * as P from './panels.js';

let X;
export function init(ctx) { X = ctx; }
const missing = (e) => e.status === 404 || e.status === 405 || /^404/.test(e.message);
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
function capacity() {
  const L = limits(), why = [];
  const n = activeAgents();
  if (n >= L.maxAgents) why.push(`${n} active agents (limit ${L.maxAgents})`);
  if (!health) why.push('RAM/disk unknown (health not read yet)');
  else {
    if (health.ram_used_pct > L.ramMaxPct) why.push(`RAM ${health.ram_used_pct}% used (floor: at most ${L.ramMaxPct}%)`);
    const c = (health.disks || []).find((d) => /^c/i.test(d.drive));
    if (c && +c.free_gb < L.diskMinGb) why.push(`C: ${(+c.free_gb).toFixed(1)} GB free (floor ${L.diskMinGb} GB)`);
  }
  return { ok: !why.length, why, n };
}
const queue = [];
function drainQueue() {
  while (queue.length && capacity().ok && !X.S.safe) {
    const job = queue.shift();
    toast(`Launch wave: starting held launch "${job.label}"`, 'info');
    job.run();
  }
  if (queue.length) X.schedule();
}
export const queued = () => queue.slice();
export function queuePanel() {
  P.drawer('queue', 'Launch queue', (b) => {
    const cap = capacity(), L = limits();
    const max = el('input', { type: 'number', min: 1, max: 12, value: L.maxAgents, 'aria-label': 'Max active agents' });
    const ram = el('input', { type: 'number', min: 50, max: 99, value: L.ramMaxPct, 'aria-label': 'Max RAM used %' });
    const disk = el('input', { type: 'number', min: 0, max: 500, value: L.diskMinGb, 'aria-label': 'Min free C: GB' });
    b.append(el('p', { class: 'muted small', text: 'New launches wait here when the limit or a RAM/disk floor is reached. Running work is never stopped to make room.' }),
      el('div', { class: 'form2' }, field('Max active agents', max), field('Max RAM used %', ram), field('Min free C: (GB)', disk)),
      el('button', { text: 'Save limits', onclick: () => { X.LS.set('limits', { maxAgents: +max.value || 4, ramMaxPct: +ram.value || 90, diskMinGb: +disk.value || 0 }); drainQueue(); P.refreshOpen(true); } }),
      el('p', {}, mark(cap.ok), ' ', cap.ok ? `Capacity free (${cap.n} active)` : 'Holding: ' + cap.why.join('; ')),
      el('ul', { class: 'list' }, queue.map((j, i) => el('li', { class: 'li' }, el('span', { class: 'grow', text: j.label }),
        el('button', { text: 'Cancel', onclick: () => { queue.splice(i, 1); P.refreshOpen(true); } })))),
      queue.length ? '' : el('p', { class: 'muted', text: 'Nothing held.' }));
  }, true);
}

// ---------- row 4: helper account guard ----------
function accountGuard(profile, account, override) {
  const kind = X.S.profiles.find((p) => p.name === profile)?.kind || profile;
  if (kind !== 'claude' && profile !== 'next') return { ok: true };
  if (profile === 'next') return { ok: true, note: 'herdr-agents picks the least-used account (lead account excluded by the tool)' };
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
  const S = X.S;
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
    items.push([!conflicts.length, false, conflicts.length ? `Overlap with active reservations: ${conflicts.map((c) => `${c.path} (${c.task}, ${c.owner})`).join('; ')}` : (conflicts.unavailable ? 'Reservations: not available (backend R19); overlap not checked' : 'No overlap with active reservations')]);
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
      const r = (await api.reservations()).reservations || [];
      for (const res of r) for (const p of res.paths) if (paths.includes(p.toLowerCase()) && res.task !== taskSel.value) conflicts.push({ path: p, task: res.task, owner: res.owner });
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
  const record = (key) => recordLaunch(key, { task: chosen, paths, at: Date.now() });
  const reserve = () => api.reservations && api.reserve(chosen, target.value === 'new' ? prof.value : X.nameOf(S.panes.get(target.value) || {}), paths)
    .catch((e) => { if (!missing(e)) toast('Reservation: ' + e.message, 'error'); });
  let run;
  if (target.value === 'headless') run = () => X.act('Headless research', () => api.headless(prof.value, prompt), 'Research started headless');
  else if (target.value === 'new') {
    const cur = S.panes.get(X.currentKey());
    run = async () => { const r = await X.act('Start agent', () => api.start({ profile: prof.value, model: '', effort: '', account: account.value.trim(), cwd: cur?.cwd || 'I:\\herdr', key: cur?.key || '' }), `Starting ${prof.value} for ${chosen}`);
      if (r) { pendingLaunch = { profile: prof.value, task: chosen, paths, prompt, at: Date.now() }; reserve(); toast(`When ${prof.value} is ready, its card offers "Send task"`, 'info', 9000); } };
  } else {
    const key = target.value;
    run = async () => { const r = await sendGuarded(key, prompt); if (r) { record(key); reserve(); } };
  }
  if (verdict.hold) { queue.push({ label: `${chosen} -> ${target.selectedOptions[0].text}`, run }); toast('Held in the launch queue (limits/floors)', 'warn'); queuePanel(); }
  else run();
}
let pendingLaunch = null;
const launches = new Map();
async function recordLaunch(key, rec) {
  try { const g = await api.git(key); rec.baseline = gitPaths(g); } catch { rec.baseline = null; }
  launches.set(key, rec);
}
export function onNewPane(p) {
  if (!pendingLaunch || !(p.name || p.agent) || Date.now() - pendingLaunch.at > 120000) return;
  if (!(p.name || '').startsWith(pendingLaunch.profile)) return;
  const pl = pendingLaunch; pendingLaunch = null;
  recordLaunch(p.key, { task: pl.task, paths: pl.paths, at: Date.now(), prompt: pl.prompt });
  toast(`${X.nameOf(p)} started for ${pl.task}. Send the task from its Proof panel when it is ready.`, 'info', 9000);
}

// ---------- row 8: uncertain delivery guard ----------
const uncertain = new Map();
export async function sendGuarded(key, text) {
  try {
    const r = await api.prompt(key, text);
    uncertain.delete(key);
    toast(`Sent to ${X.nameOf(X.S.panes.get(key) || {})}`, 'ok');
    return r || {};
  } catch (e) {
    if (e.status === 504 || e.status === 0 || /time(d)? ?out|delivery unknown|Failed to fetch|NetworkError|network/i.test(e.message)) {
      uncertain.set(key, { text, at: Date.now(), err: e.message, dispatch: e.dispatch_id });
      X.schedule(key);
      toast('Delivery UNKNOWN: the prompt may or may not have arrived. Check the screen before resending.', 'warn', 12000);
    } else fail('Prompt')(e);
    return null;
  }
}
export const isUncertain = (key) => uncertain.has(key);

// ---------- row 9: failure recovery cards ----------
function classify(p, c) {
  const s = X.statusOf(p), tail = (c.view.plain.slice(-1500) || '').toLowerCase();
  if (uncertain.has(p.key)) { const u = uncertain.get(p.key); return { kind: 'Delivery unknown', text: `Sent at ${new Date(u.at).toLocaleTimeString()}: "${u.text.slice(0, 80)}" - ${u.err}`, step: 'Read the screen; resend only if it did not arrive.', resend: true }; }
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
export function transportBanner(connected, msg) {
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
      b.append(el('section', { class: 'pslot', 'data-state': r.state }, el('div', { class: 'prow' }, mark(r.state === 'verified', r.state === 'not verified'), el('strong', { text: label }), el('span', { class: 'grow' }), sel), note,
        claim ? el('div', { class: 'small muted' }, 'Agent claims: ', el('span', { class: 'mono', text: claim.trim().slice(0, 140) })) : ''));
    }
    // row 7: scope drift
    const drift = el('div', { class: 'proof-list', text: 'Reading git status…' });
    b.append(el('h3', { text: 'Scope drift' }), drift);
    try {
      const g = await api.git(key), now = gitPaths(g), declared = new Set((L?.paths || []).map((x) => x.toLowerCase()));
      const base = L?.baseline || new Set();
      const extra = [...now].filter((x) => !base.has(x) && !declared.has(x) && ![...declared].some((d) => d.endsWith('/**') ? x.startsWith(d.slice(0, -2)) : d.includes('*') ? new RegExp('^' + d.replace(/[.+^${}()|[\]\\]/g, '\\$&').replace(/\*\*/g, '.*').replace(/\*/g, '[^/]*') + '$').test(x) : x.endsWith('/' + d) || x === d));
      drift.replaceChildren(
        el('div', { class: 'prow' }, mark(L != null, true), el('div', { text: L ? `Task ${L.task}: ${declared.size} declared file(s), baseline ${L.baseline ? L.baseline.size + ' path(s)' : 'unknown'}` : 'No launch record for this pane (launch it via "Launch a task" to compare scope)' })),
        el('div', { class: 'prow' }, mark(!extra.length, !L), el('div', { text: extra.length ? `${extra.length} change(s) outside the declared files (not reverted, not blamed - check who made them):` : 'No changes outside the declared files' })),
        ...extra.slice(0, 30).map((x) => el('div', { class: 'mono small', text: '  ' + x })));
    } catch (e) { drift.textContent = 'Scope drift: ' + (missing(e) ? 'git view unavailable' : e.message); }
    b.append(el('h3', { text: 'Citation audit (this screen)' }), auditBlock(plain));
  });
  function save(slot, state, note) { const s = X.LS.get('proof', {}); (s[id] ||= {})[slot] = { state, note, at: Date.now() }; X.LS.set('proof', s); }
}
