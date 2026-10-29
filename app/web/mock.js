// Fake backend for /?mock=1: fake state, fake event stream, fake actions. Never loaded otherwise.
// Extra flags: &down=1 starts with Herdr "not running"; &big=1 gives a 2000-line screen;
// &stress=N adds N streaming panes at 20 lines/s each (perf test); &strict=1 makes unknown routes 404.
const q = new URLSearchParams(location.search);
const now = () => new Date().toLocaleTimeString();
const C = (n, s) => `\x1b[${n}m${s}\x1b[0m`;

export function createMock(R) {
  let running = q.get('down') !== '1';
  let rev = 0, n = 5, readOnly = false, prefs = {};
  try { prefs = JSON.parse(localStorage.getItem('deck.mock-prefs') || '{}'); } catch { /* fresh mock backend */ }
  // &prefs={json} seeds saved settings (cols, rail, filter, sizes) for headless screenshots.
  if (q.get('prefs')) prefs = { ...prefs, ...JSON.parse(q.get('prefs')) };
  // &drag=v:-200,h:120,rail:80,click:#btn-inbox,drawer:-100 drags those dividers by N px after load (real
  // pointer handlers); click:<selector> clicks a button first.
  if (q.get('drag')) setTimeout(function dragWhenReady() {
    if (!document.querySelector('#grid .card')) { setTimeout(dragWhenReady, 100); return; }
    const sel = { v: '.divider.gd.v', h: '.divider.gd.h', rail: '#rail-split', drawer: '#drawer-split' };
    for (const step of q.get('drag').split(',')) {
      const [name, by] = step.split(':');
      if (name === 'click') { document.querySelector(by).click(); continue; }   // click:#btn-inbox opens the drawer
      const d = document.querySelector(sel[name]);
      if (!d) { console.error('mock drag: no divider ' + name); continue; }
      if (by === 'dbl') { d.dispatchEvent(new MouseEvent('dblclick', { bubbles: true })); continue; }   // v:dbl resets
      if (by === 'key') { d.focus(); d.dispatchEvent(new KeyboardEvent('keydown', { key: name === 'h' ? 'ArrowDown' : 'ArrowRight', bubbles: true })); continue; }
      const r = d.getBoundingClientRect(), x = r.x + r.width / 2, y = r.y + r.height / 2, dy = name === 'h';
      const at = (t, px) => new PointerEvent(t, { bubbles: true, pointerId: 1, button: 0, clientX: dy ? x : x + px, clientY: dy ? y + px : y });
      d.dispatchEvent(at('pointerdown', 0));
      window.dispatchEvent(at('pointermove', +by / 2)); window.dispatchEvent(at('pointermove', +by)); window.dispatchEvent(at('pointerup', +by));
    }
  }, 1500);
  const reservations = new Map(), scopes = new Map();
  let visible = new Set(), clientId = 'c-mock-1';
  const lines = (arr) => arr.join('\n');
  const panes = new Map();
  const add = (p) => panes.set(p.key, { label: '', title: '', error: '', tokens: {}, blocked: false, revision: 0, screen_version: 1, ...p });
  add({ key: 'lab|w1:p1', pane_id: 'w1:p1', workspace_id: 'w1', tab_id: 'w1:t1', cwd: 'I:\\herdr', name: 'claude-2040',
    agent: 'claude', agent_status: 'working', tokens: { model: 'claude-opus-5-5', context: '23%', total: '230k' },
    screen: lines([C('1;35', '* Claude Code') + ' v3.1', '', C(36, '> ') + 'Read CLAUDE.md, then do tasks/x.md', '',
      C(2, '  Reading 4 files...'), C(32, '  ✓ ') + 'api.js written', C(33, '  … ') + 'building mock']) });
  add({ key: 'lab|w1:p2', pane_id: 'w1:p2', workspace_id: 'w1', tab_id: 'w1:t1', cwd: 'I:\\herdr\\app', name: 'codex-1',
    agent: 'codex', agent_status: 'idle', tokens: { model: 'gpt-6.1-sol', context: '8%', total: '41k' },
    screen: lines([C(1, '>_ OpenAI Codex'), '', '• Done. Changed: app/main.go, app/herdr.go', '  Checks: go test ./... ok', '',
      C('38;5;244', '› Ask Codex to do anything')]) });
  add({ key: 'lab|w1:p3', pane_id: 'w1:p3', workspace_id: 'w1', tab_id: 'w1:t2', cwd: 'I:\\herdr', name: 'gemini-1',
    agent: 'agy', agent_status: 'blocked', blocked: true, tokens: { model: 'gemini-4-argon-high' },
    screen: lines(['Antigravity', '', C('1;33', 'Do you trust this folder?'), '  I:\\herdr', '', C(7, ' > Yes ') + '   No', '',
      C(2, 'enter to confirm · esc to cancel')]) });
  add({ key: 'lab|w2:p1', pane_id: 'w2:p1', workspace_id: 'w2', tab_id: 'w2:t1', cwd: 'C:\\Users\\mflma', name: '',
    agent: '', agent_status: 'unknown', screen: q.get('big') === '1'
      ? Array.from({ length: 2400 }, (_, i) => `line ${i + 1} ` + C(31 + (i % 6), 'x'.repeat(i % 80))).join('\n')
      : lines(['PowerShell 7.6', 'PS C:\\Users\\mflma> ']) });
  const workspaces = [
    { workspace_id: 'w1', label: 'herdr', agent_status: 'blocked', focused: true },
    { workspace_id: 'w2', label: 'home', agent_status: 'unknown', focused: false }];
  const tabs = [
    { tab_id: 'w1:t1', workspace_id: 'w1', label: 'main', agent_status: 'working', focused: true },
    { tab_id: 'w1:t2', workspace_id: 'w1', label: 'trust', agent_status: 'blocked', focused: false },
    { tab_id: 'w2:t1', workspace_id: 'w2', label: '1', agent_status: 'unknown', focused: false }];
  const profiles = ['claude', 'codex', 'gemini', 'gemini-flash', 'cline-ds', 'opencode', 'grok', 'cursor'].map((name) => ({
    name, kind: name.split('-')[0], model: { claude: 'claude-opus-5-5', codex: 'gpt-6.1-sol', gemini: 'gemini-4-argon-high' }[name] || '',
    effort: name === 'claude' || name === 'codex' ? 'medium' : '', accounts: name === 'claude' ? ['2030', '2040', '1984'] : name === 'codex' ? ['2030'] : [] }));

  const layouts = () => tabs.map((t) => {
    const ps = [...panes.values()].filter((p) => p.tab_id === t.tab_id), w = 200, h = 50;
    const each = Math.floor(w / Math.max(1, ps.length));
    return { tab_id: t.tab_id, workspace_id: t.workspace_id, area: { x: 0, y: 0, width: w, height: h }, zoomed: false,
      panes: ps.map((p, i) => ({ pane_id: p.pane_id, rect: { x: each * i, y: 0, width: i === ps.length - 1 ? w - each * i : each, height: h } })) };
  });
  const state = () => running
    ? { session: 'lab', root: 'I:\\herdr', lead: { tool: 'claude', account: '2040', known: true }, herdr: { running: true, error: '' }, panes: [...panes.values()].map(({ n: _n, ...p }) => p), workspaces, tabs, layouts: layouts(), profiles, error: '' }
    : { session: 'lab', herdr: { running: false, error: 'herdr.sock not found (mock)' }, panes: [...panes.values()].map((p) => ({ ...p, stale: true })), workspaces, tabs, layouts: layouts(), profiles, error: 'Herdr offline (mock)' };

  let sink = null;
  const emit = (name, data) => {
    if (!sink) return;
    if (data.panes) data = { ...data, panes: data.panes.map((p) => {
      if (visible.has(p.key)) return p;
      const { screen, screen_patch, ...metadata } = p; return metadata;
    }) };
    setTimeout(() => sink?.(name, JSON.parse(JSON.stringify(data))), 0);
  };
  const touch = (p, patch) => { Object.assign(p, patch, { revision: ++rev }); emit('panes', { panes: [p], removed: [] }); };
  const need = (key) => { const p = panes.get(key); if (!p) throw new Error('404: unknown pane ' + key); return p; };
  // Like the real backend (API.md): new rows travel as screen_patch {from_version, version, start, delete, lines}.
  const append = (p, s) => {
    p.n ??= p.screen.split('\n').length; p.screen_version ??= 1;
    p.screen += '\n' + s; p.n++;
    const out = [{ key: p.key, revision: ++rev, screen_patch: { from_version: p.screen_version, version: ++p.screen_version, start: p.n - 1, delete: 0, lines: [s] } }];
    if (p.n > 2200) {   // keep 2000 rows: drop the oldest in one patch
      const cut = p.n - 2000;
      p.screen = p.screen.split('\n').slice(cut).join('\n'); p.n = 2000;
      out.push({ key: p.key, revision: ++rev, screen_patch: { from_version: p.screen_version, version: ++p.screen_version, start: 0, delete: cut, lines: [] } });
    }
    if (visible.has(p.key)) emit('panes', { panes: out, removed: [] });
  };
  const ok = (message) => ({ ok: true, message });

  const handlers = {
    [R.state]: state,
    [R.visible]: ({ client_id, keys }) => { if (client_id !== clientId) throw new Error('unknown client'); visible = new Set(keys); emit('state', state()); return ok(''); },
    [R.size]: ({ cols, rows }) => ({ cols, rows, attached: true }),
    [R.prefs]: (d) => { if (d.prefs) { prefs = d.prefs; localStorage.setItem('deck.mock-prefs', JSON.stringify(prefs)); } return { prefs }; },
    [R.readOnly]: ({ enabled }) => { readOnly = !!enabled; emit('settings', { read_only: readOnly }); return ok(''); },
    [R.raw]: ({ key, text, keys }) => { if (text) append(need(key), text); if (keys?.length) append(need(key), '[keys ' + keys.join(' ') + ']'); return ok(''); },
    [R.jobs]: () => ({ jobs: [] }),
    [R.scopeBaseline]: ({ id, repo, allowed }) => { scopes.set(id, { repo, allowed }); return { ok: true, id, baseline_changed_paths: 0 }; },
    [R.scopeCheck]: ({ id }) => { if (!scopes.has(id)) throw new Error('scope baseline missing'); return { changed: [], unexpected: [], warning: 'Mock scope only' }; },
    [R.reservations]: (d) => {
      if (!d.task) return { reservations: [...reservations.values()], load_error: '' };
      const existing = [...reservations.values()].find((r) => r.task === d.task && r.owner === d.owner);
      if (existing) return existing;
      const conflicts = [...reservations.values()].filter((r) => r.owner !== d.owner && r.paths.some((a) => d.paths.some((b) => a === b || a.startsWith(b + '/') || b.startsWith(a + '/'))));
      if (conflicts.length && !d.overlap) { const e = new Error('reservation conflict'); e.status = 409; e.body = { conflicts }; throw e; }
      const r = { ...d, id: 'r-' + (reservations.size + 1) }; reservations.set(r.id, r); return r;
    },
    [R.release]: ({ id }) => { reservations.delete(id); return ok(''); },
    [R.headlessRoutes]: () => ({ routes: [{ profile: 'cline-ds', verified: true, search: 'server' }] }),
    [R.prompt]: ({ key, text, expect = 'agent' }) => {
      const pane = need(key);
      if (expect === 'agent' && !pane.agent && !pane.name) { const e = new Error('Agent exited / shell remains'); Object.assign(e, { status:409,code:'agent_exited' }); throw e; }
      if (/input-unknown-test/.test(text)) { const e = new Error('Cannot see its input box - not sent'); Object.assign(e, { status:409,code:'input_unknown' }); throw e; }
      if (/read-timeout-test/.test(text)) { const e = new Error('read timeout'); Object.assign(e, { status:504,code:'read_timeout' }); throw e; }
      if (/timeout-test/.test(text)) { const e = new Error('504: delivery unknown'); e.status = 504; e.dispatch_id = 'd-1'; throw e; } const p = need(key); append(p, C(36, '> ') + text); if (p.agent) touch(p, { agent_status: 'working' }); return ok('prompt sent'); },
    [R.keys]: ({ key, keys }) => { const p = need(key); append(p, C(2, `[keys ${keys.join(' ')}]`));
      if (p.blocked && keys.includes('Enter')) touch(p, { blocked: false, agent_status: 'idle' }); return ok('keys sent'); },
    [R.text]: ({ key, text }) => { append(need(key), C(2, '[text] ') + text); return ok(`${text.length} chars sent`); },
    [R.rename]: ({ key, label }) => { touch(need(key), { label }); return ok('renamed'); },
    [R.close]: ({ key }) => { need(key); panes.delete(key); emit('panes', { panes: [], removed: [key] }); return ok('closed'); },
    [R.split]: ({ key, cwd }) => { const p = need(key); return newPane(p.workspace_id, p.tab_id, cwd || p.cwd, '', ''); },
    [R.shell]: ({ cwd }) => newPane('w1', 'w1:t1', cwd || 'I:\\herdr', '', ''),
    [R.start]: (o) => { if (!o.profile) throw new Error('400: profile missing');
      return newPane('w1', 'w1:t1', o.cwd || 'I:\\herdr', `${o.profile}-${n}`, o.profile, o.model); },
    [R.stop]: ({ name }) => { for (const p of panes.values()) if (p.name === name) touch(p, { agent_status: 'done' }); return ok('stopped ' + name); },
    [R.focus]: ({ name }) => ok('focused ' + name),
    [R.tabFocus]: ({ tab_id }) => ok('focused ' + tab_id),
    [R.herdrStart]: () => { setTimeout(() => { running = true; emit('state', state()); }, 1200); return ok('launcher started'); },
    [R.reload]: () => ok('config reloaded'),
    [R.tasks]: () => ({ tasks: ['herdr-deck-app.md', 'herdr-deck-frontend.md', 'herdr-deck-split.md'].map((f) => ({ file: 'tasks/' + f, title: f, mtime: now() })) }),
    [R.taskRead]: ({ file }) => ({ file, text: file.includes('frontend') ? '# Task\n\n## 3. Scope\n- Write ONLY: `app/web/**`, `app/web/app.js`, `app/FEATURES.md`.\n\n## 6. Definition of done\n- checks pass\n\n## 7. Answer format\nDONE | BLOCKED | PARTIAL' : '# Task: example\n\n## 1. Role and goal\nMock task text.' }),
    [R.usage]: () => ({ text: 'claude 2040: 5h 41% weekly 12%\nclaude 1984: 5h 3% weekly 30%\ncodex: 5h 20%',
      accounts: [{ name: '2030', five_hour: 88, weekly: 64 }, { name: '2040', five_hour: 41, weekly: 12 }, { name: '1984', five_hour: 3, weekly: 30 }],
      codex: { five_hour: 20, weekly: 9 } }),
    [R.health]: () => ({ ram_used_pct: 61 + Math.round(Math.random() * 5), disks: [{ drive: 'C:', free_gb: 12.4 }, { drive: 'G:', free_gb: 230 }, { drive: 'I:', free_gb: 120.5 }], panes: panes.size }),
    [R.closeFinished]: ({ keys }) => { for (const k of keys) { panes.delete(k); } emit('panes', { panes: [], removed: keys }); return ok(`closed ${keys.length}`); },
    [R.signin]: () => ({ tools: [{ tool: 'claude', account: '2030', signed_in: true, detail: 'max plan' }, { tool: 'claude', account: '2040', signed_in: true, detail: '' },
      { tool: 'claude', account: '1984', signed_in: false, detail: 'not logged in' }, { tool: 'codex', account: '', signed_in: true, detail: '' },
      { tool: 'cline', account: '', signed_in: true, detail: '' }, { tool: 'grok', account: '', signed_in: false, detail: '' },
      { tool: 'agy', account: '', signed_in: true, detail: '' }, { tool: 'cursor', account: '', signed_in: false, detail: '' }] }),
    [R.signinOpen]: ({ tool, account }) => newPane('w1', 'w1:t1', 'I:\\herdr', `login-${tool}${account ? '-' + account : ''}`, tool),
    [R.git]: () => ({ text: ' M app/web/app.js\n?? app/web/mock.js\n 2 files changed, 400 insertions(+)' }),
    [R.runs]: () => ({ runs: [{ file: 'agent-runs/codex-1.md', title: 'codex-1 - deck backend', mtime: now() }] }),
    [R.runRead]: () => ({ file: 'agent-runs/codex-1.md', text: '# codex-1\n\nTask: deck backend\n\nAnswer: DONE' }),
    [R.docs]: ({ name }) => ({ name, text: `# ${name}\n\n1. First lesson.\n2. Second lesson about herdr.` }),
    [R.lesson]: ({ text }) => ok('appended ' + text.length + ' chars'),
    [R.check]: () => ({ lines: ['OK   docs present', 'WARN tests slow', 'OK   skills synced'] }),
  };

  function newPane(ws, tab, cwd, name, agent, model) {
    const id = `${ws}:p${n++}`, key = 'lab|' + id;
    add({ key, pane_id: id, workspace_id: ws, tab_id: tab, cwd, name, agent, agent_status: agent ? 'working' : 'unknown',
      tokens: model ? { model } : {}, screen: agent ? `Starting ${agent}...` : 'PS ' + cwd + '> ' });
    emit('panes', { panes: [panes.get(key)], removed: [] });
    return { ...ok('started ' + (name || id)), pane_id: id, name };
  }

  const stress = Math.min(32, +q.get('stress') || 0);
  for (let i = 0; i < stress; i++) add({ key: 'lab|w3:p' + (i + 1), pane_id: 'w3:p' + (i + 1), workspace_id: 'w3', tab_id: 'w3:t1', cwd: 'I:\\herdr', name: 'stress-' + (i + 1),
    agent: ['claude', 'codex', 'agy', 'cline'][i % 4], agent_status: 'working', tokens: { model: 'mock', ctx: `ctx ${10 + i * 7}% (100k/1M)`, out: '12k' }, screen: 'stress pane ' + (i + 1) });
  if (stress) { workspaces.push({ workspace_id: 'w3', label: 'stress', agent_status: 'working' }); tabs.push({ tab_id: 'w3:t1', workspace_id: 'w3', label: 'stress', agent_status: 'working' }); }
  let sn = 0;
  if (stress) setInterval(() => {
    if (!sink || !running) return;
    sn++;
    for (const p of panes.values()) if (p.workspace_id === 'w3') append(p, C(31 + (sn % 6), '[' + sn + ']') + ' ' + p.name + ' streaming line ' + sn + ' ' + 'lorem ipsum dolor sit amet '.repeat(1 + (sn % 3)));
  }, 50);

  // Live activity: working panes print a line every ~1.5 s.
  let tick = 0;
  setInterval(() => {
    if (!sink || !running || document.hidden) return;
    tick++;
    for (const p of panes.values()) {
      if (p.agent_status !== 'working' || p.workspace_id === 'w3') continue;
      append(p, C(2, `  ${now()} `) + ['thinking', 'editing app.js', 'running tests', 'reading LESSONS.md'][tick % 4]);
      if (tick % 12 === 0 && p.agent) touch(p, { agent_status: 'idle' });
    }
  }, 1500);

  const delay = (v) => new Promise((ok_, bad) => setTimeout(() => (v instanceof Error ? bad(v) : ok_(v)), 120));
  const call = (path, data) => {
    const [base, query] = path.split('?');
    const h = handlers[base];
    if (data && readOnly && ![R.visible, R.readOnly, R.prefs, R.scopeBaseline].includes(base)) { const e = new Error('read only'); e.status = 403; return delay(e); }
    if (!h && (q.get('strict') === '1' || !data)) return delay(new Error('404: mock has no ' + base));
    if (!h) return delay(ok('(mock) ' + base));
    try { return delay(h({ ...Object.fromEntries(new URLSearchParams(query || '')), ...data })); } catch (e) { return delay(e); }
  };
  window.__mock = { setRunning(v) { running = v; emit('herdr', { running: v, error: v ? '' : 'stopped (mock)' }); emit('state', state()); }, panes };
  return {
    get: (p) => call(p),
    post: (p, d) => call(p, d),
    events(_names, h) {
      sink = h.onEvent;
      setTimeout(() => { h.onOpen(); emit('hello', { client_id: clientId }); emit('settings', { read_only: readOnly }); emit('state', state()); }, 50);
      return () => { sink = null; };
    },
  };
}
