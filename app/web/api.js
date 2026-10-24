// The only file that knows the backend contract (app/API.md). `/?mock=1` swaps in mock.js.
const q = new URLSearchParams(location.search);
export const MOCK = q.get('mock') === '1';
const TOKEN = q.get('t') || '';

export const ROUTES = {
  state: '/api/state', events: '/api/events', visible: '/api/visible',
  prompt: '/api/pane/prompt', keys: '/api/pane/keys', text: '/api/pane/text',
  rename: '/api/pane/rename', close: '/api/pane/close', split: '/api/pane/split', shell: '/api/shell',
  start: '/api/agent/start', stop: '/api/agent/stop', focus: '/api/agent/focus', tabFocus: '/api/tab/focus',
  herdrStart: '/api/herdr/start', reload: '/api/herdr/reload-config',
  tasks: '/api/tasks', taskRead: '/api/tasks/read', usage: '/api/usage', health: '/api/health',
  closeFinished: '/api/panes/close-finished', signin: '/api/signin', signinOpen: '/api/signin/open',
  git: '/api/git', runs: '/api/runs', runRead: '/api/runs/read', docs: '/api/docs',
  lesson: '/api/lessons/append', check: '/api/check', readOnly: '/api/read-only', accounts: '/api/accounts',
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

const withToken = (path) => path + (path.includes('?') ? '&' : '?') + 't=' + encodeURIComponent(TOKEN);

async function readJson(res) {
  const text = await res.text();
  let body = null;
  try { body = text ? JSON.parse(text) : {}; } catch { body = null; }
  if (!res.ok) {
    const msg = (body && (body.error || body.message)) || text.slice(0, 300) || res.statusText;
    const err = new Error(`${res.status}: ${msg}`); err.status = res.status; err.dispatch_id = body?.dispatch_id; err.body = body; throw err;
  }
  if (body === null) throw new Error('Backend sent invalid JSON: ' + text.slice(0, 200));
  if (body.ok === false) throw new Error(body.error || body.message || 'Backend said the action failed');
  return body;
}

const real = {
  async get(path) {
    return readJson(await fetch(withToken(path), { headers: { 'X-Deck-Token': TOKEN }, cache: 'no-store' }));
  },
  async post(path, data) {
    return readJson(await fetch(withToken(path), {
      method: 'POST', cache: 'no-store',
      headers: { 'Content-Type': 'application/json', 'X-Deck-Token': TOKEN },
      body: JSON.stringify(data || {}),
    }));
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

const get = (p) => impl.get(p);
const post = (p, d) => impl.post(p, d);
const qs = (o) => '?' + new URLSearchParams(o).toString();

export const api = {
  state: () => get(ROUTES.state),
  events: (names, h) => impl.events(names, h),
  visible: (keys) => post(ROUTES.visible, { keys }),
  prompt: (key, text) => post(ROUTES.prompt, { key, text }),
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
  tasks: () => get(ROUTES.tasks),
  taskRead: (file) => get(ROUTES.taskRead + qs({ file })),
  usage: () => get(ROUTES.usage),
  health: () => get(ROUTES.health),
  closeFinished: (keys) => post(ROUTES.closeFinished, { keys }),
  signin: () => get(ROUTES.signin),
  signinOpen: (tool, account) => post(ROUTES.signinOpen, { tool, account }),
  git: (key) => get(ROUTES.git + qs({ key })),
  runs: () => get(ROUTES.runs),
  runRead: (file) => get(ROUTES.runRead + qs({ file })),
  docs: (name) => get(ROUTES.docs + qs({ name })),
  lesson: (text) => post(ROUTES.lesson, { text }),
  check: () => post(ROUTES.check, {}),
  readOnly: (enabled) => post(ROUTES.readOnly, { enabled }),
  accounts: () => get(ROUTES.accounts),
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
  reserve: (task, owner, paths) => post(ROUTES.reservations, { task, owner, paths }),
  release: (id) => post(ROUTES.release, { id }),
  headlessRoutes: () => get(ROUTES.headlessRoutes),
  jobs: () => get(ROUTES.jobs),
};
