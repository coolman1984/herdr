// Fake backend for /?mock=1: fake state, fake event stream, fake actions. Never loaded otherwise.
// Extra flags: &down=1 starts with Herdr "not running"; &big=1 gives a 2000-line screen;
// &stress=N adds N streaming panes at 20 lines/s each (perf test); &strict=1 makes unknown routes 404.
const q = new URLSearchParams(location.search);
const now = () => new Date().toLocaleTimeString();
const C = (n, s) => `\x1b[${n}m${s}\x1b[0m`;

export function createMock(R) {
  let running = q.get('down') !== '1';
  let rev = 0, n = 5;
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
    ? { session: 'lab', herdr: { running: true, error: '' }, panes: [...panes.values()].map(({ n: _n, ...p }) => p), workspaces, tabs, layouts: layouts(), profiles, error: '' }
    : { session: 'lab', herdr: { running: false, error: 'herdr.sock not found (mock)' }, panes: [], workspaces: [], tabs: [], profiles, error: '' };

  let sink = null;
  const emit = (name, data) => sink && setTimeout(() => sink(name, JSON.parse(JSON.stringify(data))), 0);
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
    emit('panes', { panes: out, removed: [] });
  };
  const ok = (message) => ({ ok: true, message });

  const handlers = {
    [R.state]: state,
    [R.visible]: () => ok(''),
    [R.prompt]: ({ key, text }) => { const p = need(key); append(p, C(36, '> ') + text); if (p.agent) touch(p, { agent_status: 'working' }); return ok('prompt sent'); },
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
    [R.taskRead]: () => ({ file: 'tasks/x.md', text: '# Task: example\n\n## 1. Role and goal\nMock task text.' }),
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
    return ok('started ' + (name || id));
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
    if (!h && (q.get('strict') === '1' || !data)) return delay(new Error('404: mock has no ' + base));
    if (!h) return delay(ok('(mock) ' + base));
    try { return delay(h({ ...Object.fromEntries(new URLSearchParams(query || '')), ...data })); } catch (e) { return delay(e); }
  };
  window.__mock = { setRunning(v) { running = v; emit('herdr', { running: v, error: v ? '' : 'stopped (mock)' }); if (v) emit('state', state()); }, panes };
  return {
    get: (p) => call(p),
    post: (p, d) => call(p, d),
    events(_names, h) {
      sink = h.onEvent;
      setTimeout(() => { h.onOpen(); emit('state', state()); }, 50);
      return () => { sink = null; };
    },
  };
}
