// The only file that knows the backend contract (app/API.md). `/?mock=1` swaps in mock.js.
const q = new URLSearchParams(location.search);
export const MOCK = q.get('mock') === '1';
const TOKEN = q.get('t') || '';

export const ROUTES = {
  state: '/api/state', events: '/api/events', visible: '/api/visible',
  raw: '/api/pane/raw', prefs: '/api/prefs', scopeBaseline: '/api/scope/baseline', scopeCheck: '/api/scope/check', prompt: '/api/pane/prompt', keys: '/api/pane/keys', text: '/api/pane/text',
  rename: '/api/pane/rename', close: '/api/pane/close', split: '/api/pane/split', shell: '/api/shell',
  start: '/api/agent/start', stop: '/api/agent/stop', focus: '/api/agent/focus', tabFocus: '/api/tab/focus',
  herdrStart: '/api/herdr/start', reload: '/api/herdr/reload-config',
  tasks: '/api/tasks', taskRead: '/api/tasks/read', usage: '/api/usage', health: '/api/health',
  closeFinished: '/api/panes/close-finished', signin: '/api/signin', signinOpen: '/api/signin/open',
  git: '/api/git', runs: '/api/runs', runRead: '/api/runs/read', docs: '/api/docs',
  lesson: '/api/lessons/append', check: '/api/check', readOnly: '/api/read-only', accounts: '/api/accounts', size: '/api/size',
  // parity requests R8-R16 (API-REQUESTS.md); a 404 means "not built yet" and the UI says so
  paneFocus: '/api/pane/focus', paneZoom: '/api/pane/zoom', resize: '/api/pane/resize', swap: '/api/pane/swap', move: '/api/pane/move',
  tabCreate: '/api/tab/create', tabRename: '/api/tab/rename', tabClose: '/api/tab/close',
  wsRename: '/api/workspace/rename', wsClose: '/api/workspace/close', wsFocus: '/api/workspace/focus', wsOpen: '/api/workspace/open',
  foldersRecent: '/api/folders/recent', foldersList: '/api/folders/list', columns: '/api/columns',
  layouts: '/api/layouts', layoutBuild: '/api/layouts/build', layoutSave: '/api/layouts/save',
  spaces: '/api/spaces', spacesJoin: '/api/spaces/join', spacesSeparate: '/api/spaces/separate', stopAll: '/api/stop-all',
  agentsGet: '/api/agents/', agentsPost: '/api/agents/', headless: '/api/headless', syncSkills: '/api/sync-skills',
  reservations: '/api/reservations', release: '/api/reservations/release', headlessRoutes: '/api/headless/routes', jobs: '/api/jobs',
};

export const missing = (e) => e.status === 404 || e.status === 405 || /^404/.test(e.message);
export const why = (e) => (missing(e) ? 'Not available: the backend has no endpoint for this.' : e.message);

const withToken =(path) => path + (path.includes('?') ? '&' : '?') + 't=' + encodeURIComponent(TOKEN);

async function readJson(res) {
  let text;
  try { text = await res.text(); } catch (e) { e.responseReadFailed = true; throw e; }
  let body = null;
  try { body = text ? JSON.parse(text) : {}; } catch { body = null; }
  if (!res.ok) {
    // A code-like error ("input_not_empty") comes with a readable `message`: show the sentence.
    const code = body && /^[a-z]+(_[a-z]+)+$/.test(body.error || '') ? body.error : '';
    const msg = (body && ((code && body.message) || body.error || body.message)) || text.slice(0, 300) || res.statusText;
    const err = new Error(code ? msg : `${res.status}: ${msg}`);
    Object.assign(err, { status: res.status, code, dispatch_id: body?.dispatch_id, body });
    throw err;
  }
  if (body === null) throw new Error('Backend sent invalid JSON: ' + text.slice(0, 200));
  if (body.ok === false) throw new Error(body.error || body.message || 'Backend said the action failed');
  return body;
}

const real = {
  async get(path, options = {}) {
    return readJson(await fetch(withToken(path), { headers: { 'X-Deck-Token': TOKEN }, cache: 'no-store', ...options }));
  },
  async post(path, data) {
    let res;
    try { res = await fetch(withToken(path), {
      method: 'POST', cache: 'no-store',
      headers: { 'Content-Type': 'application/json', 'X-Deck-Token': TOKEN },
      body: JSON.stringify(data || {}),
    });
    } catch (e) { e.promptSent = path === ROUTES.prompt; throw e; }
    try { return await readJson(res); }
    catch (e) { if (e.responseReadFailed) e.promptSent = path === ROUTES.prompt; throw e; }
  },
  // handlers: {onEvent(name, data), onOpen(), onDown(err)}. Returns a close function.
  events(names, h) {
    const es = new EventSource(withToken(ROUTES.events));
    es.onopen = () => h.onOpen();
    es.onerror = () => h.onDown(new Error('Event stream lost; reconnecting'));
    for (const n of names) es.addEventListener(n, (ev) => {
      let data;
      try { data = JSON.parse(ev.data); } catch { h.onEvent('error', { message: `Bad ${n} event from backend` }); return; }
      h.onEvent(n, data);
    });
    return () => es.close();
  },
};

let impl = real;
export async function initApi() {
  if (MOCK) impl = (await import('./mock.js')).createMock(ROUTES);
}

const get = (p, options) => impl.get(p, options);
const post = (p, d) => impl.post(p, d);
const qs = (o) => '?' + new URLSearchParams(o).toString();

export const api = {
  state: () => get(ROUTES.state),
  events: (names, h) => impl.events(names, h),
  visible: (keys, client_id) => post(ROUTES.visible, { client_id, keys }),
  raw: (key, text, keys = []) => post(ROUTES.raw, { key, text, keys }),
  prefs: (options) => get(ROUTES.prefs, options),
  savePrefs: (prefs) => post(ROUTES.prefs, { prefs }),
  size: (cols, rows) => post(ROUTES.size, { cols, rows }),
  scopeBaseline: (id, repo, allowed) => post(ROUTES.scopeBaseline, { id, repo, allowed }),
  scopeCheck: (id, options) => get(ROUTES.scopeCheck + qs({ id }), options),
  // expect: 'agent' (default) or 'shell'; the backend refuses agent text when only a shell is left.
  prompt: (key, text, expect = 'agent') => post(ROUTES.prompt, { key, text, expect }),
  keys: (key, keys) => post(ROUTES.keys, { key, keys }),
  text: (key, text) => post(ROUTES.text, { key, text }),
  rename: (key, label) => post(ROUTES.rename, { key, label }),
  close: (key) => post(ROUTES.close, { key }),
  split: (key, cwd) => post(ROUTES.split, { key, cwd }),
  shell: (cwd) => post(ROUTES.shell, { cwd }),
  start: (o) => post(ROUTES.start, o),
  stop: (name) => post(ROUTES.stop, { name }),
  focus: (name) => post(ROUTES.focus, { name }),
  tabFocus: (tab_id) => post(ROUTES.tabFocus, { tab_id }),
  herdrStart: () => post(ROUTES.herdrStart, {}),
  reload: () => post(ROUTES.reload, {}),
  tasks: (options) => get(ROUTES.tasks, options),
  taskRead: (file) => get(ROUTES.taskRead + qs({ file })),
  usage: (options) => get(ROUTES.usage, options),
  health: () => get(ROUTES.health),
  closeFinished: (keys) => post(ROUTES.closeFinished, { keys }),
  signin: (options) => get(ROUTES.signin, options),
  signinOpen: (tool, account) => post(ROUTES.signinOpen, { tool, account }),
  git: (key, options) => get(ROUTES.git + qs({ key }), options),
  runs: (options) => get(ROUTES.runs, options),
  runRead: (file) => get(ROUTES.runRead + qs({ file })),
  docs: (name, options) => get(ROUTES.docs + qs({ name }), options),
  lesson: (text) => post(ROUTES.lesson, { text }),
  check: () => post(ROUTES.check, {}),
  readOnly: (enabled) => post(ROUTES.readOnly, { enabled }),
  accounts: (options) => get(ROUTES.accounts, options),
  paneFocus: (key) => post(ROUTES.paneFocus, { key }),
  paneZoom: (key) => post(ROUTES.paneZoom, { key }),
  splitDir: (key, direction, cwd) => post(ROUTES.split, { key, direction, cwd }),
  resize: (key, direction, amount) => post(ROUTES.resize, { key, direction, amount }),
  swap: (key, other) => post(ROUTES.swap, { key, with: other }),
  move: (key, tab_id) => post(ROUTES.move, { key, tab_id }),
  tabCreate: (workspace_id, cwd) => post(ROUTES.tabCreate, { workspace_id, cwd }),
  tabRename: (tab_id, label) => post(ROUTES.tabRename, { tab_id, label }),
  tabClose: (tab_id) => post(ROUTES.tabClose, { tab_id }),
  wsRename: (workspace_id, label) => post(ROUTES.wsRename, { workspace_id, label }),
  wsClose: (workspace_id) => post(ROUTES.wsClose, { workspace_id }),
  wsFocus: (workspace_id) => post(ROUTES.wsFocus, { workspace_id }),
  wsOpen: (cwd, agent) => post(ROUTES.wsOpen, { cwd, agent }),
  foldersRecent: () => get(ROUTES.foldersRecent),
  foldersList: (path) => get(ROUTES.foldersList + qs({ path })),
  columns: (key, count, direction) => post(ROUTES.columns, { key, count, direction }),
  layouts: () => get(ROUTES.layouts),
  layoutBuild: (plan) => post(ROUTES.layoutBuild, { plan }),
  layoutSave: (name, plan) => post(ROUTES.layoutSave, { name, plan }),
  spaces: () => get(ROUTES.spaces),
  spacesJoin: (workspace_ids) => post(ROUTES.spacesJoin, { workspace_ids }),
  spacesSeparate: () => post(ROUTES.spacesSeparate, {}),
  stopAll: (word) => post(ROUTES.stopAll, { confirm: word }),
  agentsGet: (cmd, q) => get(ROUTES.agentsGet + cmd + (q ? qs(q) : '')),
  agentsPost: (cmd, body) => post(ROUTES.agentsPost + cmd, body),
  headless: (profile, text) => post(ROUTES.headless, { profile, text }),
  syncSkills: () => post(ROUTES.syncSkills, {}),
  reservations: () => get(ROUTES.reservations),
  reserve: (task, owner, paths, overlap = false) => post(ROUTES.reservations, { task, owner, paths, overlap }),
  release: (id) => post(ROUTES.release, { id }),
  headlessRoutes: () => get(ROUTES.headlessRoutes),
  jobs: (options) => get(ROUTES.jobs, options),
};
