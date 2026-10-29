// Minimal browser boundary for real ES modules; no rendering or network emulation.
export class NodeStub extends EventTarget {
  constructor(name = 'div', text = '') {
    super(); this.nodeName = name.toUpperCase(); this.nodeType = name === '#text' ? 3 : 1;
    this._text = text; this.childNodes = []; this.attrs = new Map(); this.dataset = {};
    this.style = { setProperty() {}, removeProperty() {} }; this.hidden = false;
    this.value = ''; this.checked = false; this.scrollTop = 0; this.scrollHeight = 20;
    const classes = new Set(); this.classList = { add: (...xs) => xs.forEach(x => classes.add(x)), remove: x => classes.delete(x), contains: x => classes.has(x), toggle(x, on) { if (on ?? !classes.has(x)) classes.add(x); else classes.delete(x); } };
  }
  get nodeTypeValue() { return this.nodeType; }
  get children() { return this.childNodes.filter(n => n.nodeType === 1); }
  get firstChild() { return this.childNodes[0]; }
  get nextSibling() { return this.parentNode?.childNodes[this.parentNode.childNodes.indexOf(this) + 1]; }
  get attributes() { return [...this.attrs].map(([name, value]) => ({ name, value })); }
  setAttribute(k, v) { this.attrs.set(k, String(v)); if (k === 'class') this.className = String(v); if (k.startsWith('data-')) this.dataset[k.slice(5)] = String(v); }
  getAttribute(k) { return this.attrs.get(k) ?? null; }
  hasAttribute(k) { return this.attrs.has(k); }
  removeAttribute(k) { this.attrs.delete(k); }
  append(...nodes) { for (let n of nodes) { if (!n?.nodeType) n = new NodeStub('#text', String(n)); n.remove(); this.childNodes.push(n); n.parentNode = this; } }
  appendChild(n) { this.append(n); return n; }
  prepend(n) { this.insertBefore(n, this.firstChild); }
  replaceChildren(...nodes) { for (const n of this.childNodes) n.parentNode = null; this.childNodes = []; this._text = ''; this.append(...nodes); }
  insertBefore(n, ref) { n.remove(); const i = this.childNodes.indexOf(ref); if (i < 0) this.childNodes.push(n); else this.childNodes.splice(i, 0, n); n.parentNode = this; }
  replaceWith(n) { const p = this.parentNode; if (!p) return; const i = p.childNodes.indexOf(this); n.remove(); p.childNodes[i] = n; n.parentNode = p; this.parentNode = null; }
  remove() { if (this.parentNode) { const a = this.parentNode.childNodes; a.splice(a.indexOf(this), 1); this.parentNode = null; } }
  get textContent() { return this._text + this.childNodes.map(n => n.textContent).join(''); }
  set textContent(v) { this.replaceChildren(); this._text = String(v); }
  set innerHTML(v) { this.textContent = v; }
  get innerHTML() { return this.textContent; }
  matches(selector) { return selector.split(',').some(s => { s = s.trim(); if (s.startsWith('.')) return this.className?.split(' ').includes(s.slice(1)); if (s.startsWith('[')) { const m = s.match(/^\[([^=\]]+)(?:="?([^"\]]+)"?)?\]$/); return m && this.hasAttribute(m[1]) && (!m[2] || this.getAttribute(m[1]) === m[2]); } return this.nodeName === s.toUpperCase(); }); }
  querySelectorAll(s) { return this.children.flatMap(n => [...(n.matches(s) ? [n] : []), ...n.querySelectorAll(s)]); }
  querySelector(s) { return this.querySelectorAll(s)[0] || null; }
  closest(s) { return this.matches(s) ? this : this.parentNode?.closest(s) || null; }
  contains(n) { return n === this || this.children.some(c => c.contains(n)); }
  focus() { document.activeElement = this; }
  get isConnected() { return document.body.contains(this); }
  getBoundingClientRect() { return this.rect || { top: 20, bottom: 40, right: 200, left: 0, width: 180, height: 100 }; }
}
export function browser() {
  const ids = new Map(), doc = new EventTarget(), win = new EventTarget();
  doc.body = new NodeStub('body'); doc.documentElement = new NodeStub('html'); doc.hidden = false;
  doc.createElement = n => new NodeStub(n); doc.createElementNS = (_, n) => new NodeStub(n);
  doc.querySelector = s => { if (s.startsWith('#')) { if (!ids.has(s)) { const n = new NodeStub(s.includes('text') ? 'textarea' : 'div'); n.setAttribute('id', s.slice(1)); ids.set(s, n); doc.body.append(n); } return ids.get(s); } return doc.body.querySelector(s); };
  doc.querySelectorAll = s => doc.body.querySelectorAll(s);
  const store = new Map(), timers = new Map(), frames = new Map(); let seq = 0;
  globalThis.document = doc; globalThis.window = win; win.getSelection = () => ({ toString: () => '' });
  globalThis.Element = NodeStub; globalThis.addEventListener = win.addEventListener.bind(win);
  globalThis.location = { search: '', reload() {} }; globalThis.innerWidth = 800; globalThis.innerHeight = 600;
  globalThis.localStorage = { getItem: k => store.get(k) ?? null, setItem: (k,v) => store.set(k,v), key: i => [...store.keys()][i], get length() { return store.size; } };
  globalThis.setTimeout = (fn, ms) => { const id = ++seq; timers.set(id, { fn, ms }); return id; }; globalThis.clearTimeout = id => timers.delete(id);
  globalThis.setInterval = (fn, ms) => { const id = ++seq; timers.set(id, { fn, ms, interval: true }); return id; }; globalThis.clearInterval = globalThis.clearTimeout;
  globalThis.requestAnimationFrame = fn => { const id = ++seq; frames.set(id, fn); return id; };
  globalThis.getComputedStyle = () => ({ getPropertyValue: () => '20' });
  const observers = []; globalThis.ResizeObserver = class { constructor(fn) { this.callback = fn; observers.push(this); } observe() {} unobserve() {} };
  globalThis.fetch = () => new Promise(() => {});
  return { ids, timers, frames, observers, async tick(ms) { const list = [...timers]; for (const [id,t] of list) if (t.ms === ms) { if (!t.interval) timers.delete(id); await t.fn(); } await Promise.resolve(); }, frame() { const list = [...frames.values()]; frames.clear(); for (const fn of list) fn(); } };
}
