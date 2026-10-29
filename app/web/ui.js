// Small DOM helpers, toasts and dialogs (native <dialog>: focus trap + Esc for free).
export function el(tag, attrs, ...kids) {
  const n = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (v == null || v === false) continue;
    if (k.startsWith('on')) n.addEventListener(k.slice(2), v);
    else if (k === 'text') n.textContent = v;
    else if (k === 'class') n.className = v;
    else if (k === 'value' && 'value' in n) n.value = v;   // a textarea's text is its property, not an attribute
    else if (k in n && typeof v !== 'string') n[k] = v;
    else n.setAttribute(k, v === true ? '' : v);
  }
  for (const c of kids.flat()) if (c != null && c !== false) n.append(c.nodeType ? c : String(c));
  return n;
}
export const $ = (s, r = document) => r.querySelector(s);

// Toasts: polite by default, assertive for blocked agents and errors.
export function toast(msg, kind = 'info', ms = kind === 'error' || kind === 'blocked' ? 9000 : 4000) {
  const box = $(kind === 'error' || kind === 'blocked' ? '#toasts-loud' : '#toasts');
  const t = el('div', { class: 'toast ' + kind, role: kind === 'error' ? 'alert' : 'status' },
    el('span', { text: msg }), el('button', { class: 'x', 'aria-label': 'Dismiss', text: '×', onclick: () => t.remove() }));
  box.append(t);
  while (box.children.length > 5) box.firstChild.remove();
  setTimeout(() => t.remove(), ms);
}
export const fail = (what) => (e) => toast(`${what} failed: ${e.message}`, 'error');

// Generic modal. actions: [{label, value, kind:'primary'|'danger', focus:true}]
// Resolves with the clicked value, or null on Esc/Cancel. `check(value, form)` may return an
// error string to keep the dialog open.
export function dialog({ title, body, actions, check, wide }) {
  return new Promise((resolve) => {
    const err = el('p', { class: 'dlg-err', role: 'alert' });
    const d = el('dialog', { class: 'dlg' + (wide ? ' wide' : ''), 'aria-label': title });
    const form = el('form', { method: 'dialog' },
      el('h2', { text: title }), body, err,
      el('div', { class: 'dlg-actions' }, actions.map((a) =>
        el('button', { type: a.value === null ? 'button' : 'submit', value: String(a.value), class: a.kind || '', 'data-focus': a.focus ? '1' : null,
          text: a.label, onclick: a.value === null ? () => d.close('') : null }))));
    d.append(form);
    let result = null;
    form.addEventListener('submit', (ev) => {
      const v = ev.submitter ? ev.submitter.value : '';
      const msg = check && v !== 'null' ? check(v, form) : '';
      if (msg) { ev.preventDefault(); err.textContent = msg; return; }
      result = v;
    });
    d.addEventListener('close', () => { d.remove(); resolve(result); });
    document.body.append(d);
    d.showModal();
    const f = d.querySelector('[data-focus]') || d.querySelector('input,textarea,select');
    if (f) f.focus();
  });
}

export async function confirm(title, text, okLabel = 'OK', danger = true) {
  const v = await dialog({ title, body: el('p', { class: 'pre', text }),
    actions: [{ label: 'Cancel', value: null, focus: true }, { label: okLabel, value: 'ok', kind: danger ? 'danger' : 'primary' }] });
  return v === 'ok';
}

export async function confirmTyped(title, text, word) {
  const inp = el('input', { 'aria-label': `Type ${word} to confirm`, autocomplete: 'off', spellcheck: false });
  const v = await dialog({ title, body: el('div', {}, el('p', { class: 'pre', text }), el('label', {}, `Type "${word}" to confirm `, inp)),
    actions: [{ label: 'Cancel', value: null }, { label: 'Confirm', value: 'ok', kind: 'danger' }],
    check: () => (inp.value.trim() === word ? '' : `Type "${word}" exactly.`) });
  return v === 'ok';
}

export async function ask(title, label, value = '') {
  const inp = el('input', { value, 'aria-label': label });
  const v = await dialog({ title, body: el('label', { class: 'field' }, label, inp),
    actions: [{ label: 'Cancel', value: null }, { label: 'Save', value: 'ok', kind: 'primary' }] });
  return v === 'ok' ? inp.value : null;
}

export async function copyText(text) {
  try { await navigator.clipboard.writeText(text); toast(`Copied ${text.length} characters`); }
  catch (e) { toast('Copy failed: ' + e.message, 'error'); }
}

export const ago = (t) => {
  const s = Math.round((Date.now() - t) / 1000);
  return s < 60 ? s + ' s' : s < 3600 ? Math.round(s / 60) + ' min' : Math.round(s / 3600) + ' h';
};

// Shared bounded state and scheduling primitives; kept independent of DOM for regression tests.
export const deliveryUnknown = (e) => (e.status === 504 && !!e.dispatch_id) || (!e.status && e.promptSent === true);
export const knownNumber = (v, digits = 1) => v == null ? 'unknown' : Number(v).toFixed(digits);
export const sameValues = (a, b) => JSON.stringify(a) === JSON.stringify(b);
export function queuePatch(p, patch) {
  if (p.needsResync) return false;
  const bytes = JSON.stringify(patch).length * 2;
  if ((p.pq?.length || 0) >= 64 || (p.patchBytes || 0) + bytes > 262144) {
    p.pq = null; p.patchBytes = 0; p.needsResync = true; return false;
  }
  (p.pq ||= []).push(patch); p.patchBytes = (p.patchBytes || 0) + bytes; return true;
}
export function upsertJob(jobs, d) {
  if (!d.id) return;
  const old = jobs.get(d.id);
  const finished = d.done === true || (d.status ? ['done', 'failed', 'completed'].includes(d.status) : typeof d.ok === 'boolean');
  if (old?.done && !finished) return old;
  const job = { label: d.label || d.kind || d.id, t: d.started ? Date.parse(d.started) : Date.now(), ...old, ...d, done: !!(old?.done || finished) };
  if (job.output) job.output = job.output.slice(-32768);
  if (job.message) job.message = job.message.slice(-32768);
  jobs.set(d.id, job);
  while (jobs.size > 100) { const k = [...jobs].find(([, j]) => j.done)?.[0] || jobs.keys().next().value; jobs.delete(k); }
  return job;
}
export class OrderedQueue {
  constructor(send, allowed = () => true) { this.send = send; this.allowed = allowed; this.tails = new Map(); this.tokens = new Map(); this.epoch = 0; }
  push(key, text, keys = []) {
    const epoch = this.epoch, token = this.tokens.get(key) || { cancelled: false };
    this.tokens.set(key, token);
    const next = (this.tails.get(key) || Promise.resolve()).then(() => {
      if (epoch !== this.epoch || token.cancelled || !this.allowed(key)) return;
      return this.send(key, text, keys);
    });
    this.tails.set(key, next);
    next.finally(() => { if (this.tails.get(key) === next) { this.tails.delete(key); this.tokens.delete(key); } }).catch(() => {});
    return next;
  }
  drain(key) { return this.tails.get(key) || Promise.resolve(); }
  clear() { this.epoch++; }
  forget(key) { const token = this.tokens.get(key); if (token) token.cancelled = true; this.tokens.delete(key); this.tails.delete(key); }
}
export function menuPosition(rect, size, viewport) {
  const pad = 8;
  return { left: Math.max(pad, Math.min(rect.right - size.width, viewport.width - size.width - pad)),
    top: Math.max(pad, rect.top - size.height >= pad ? rect.top - size.height : Math.min(rect.bottom, viewport.height - size.height - pad)) };
}
let activeMenuClose = null;
export function openMenu(trigger, menu) {
  activeMenuClose?.();
  const ac = new AbortController(), opts = { signal: ac.signal };
  const close = () => { if (activeMenuClose === close) activeMenuClose = null; ac.abort(); menu.remove(); trigger.setAttribute('aria-expanded', 'false'); if (trigger.isConnected) trigger.focus(); };
  activeMenuClose = close;
  document.body.append(menu);
  menu.classList.add('card-popover');
  const r = menuPosition(trigger.getBoundingClientRect(), menu.getBoundingClientRect(), { width: innerWidth, height: innerHeight });
  menu.style.left = r.left + 'px'; menu.style.top = r.top + 'px';
  trigger.setAttribute('aria-expanded', 'true');
  document.addEventListener('pointerdown', (ev) => { if (!menu.contains(ev.target) && !trigger.contains(ev.target)) close(); }, opts);
  document.addEventListener('keydown', (ev) => { if (ev.key === 'Escape') { ev.preventDefault(); ev.stopImmediatePropagation(); close(); } }, { ...opts, capture: true });
  window.addEventListener('resize', close, opts);
  window.addEventListener('scroll', (ev) => { if (!menu.contains(ev.target)) close(); }, { ...opts, capture: true });
  menu.addEventListener('click', (ev) => { if (ev.target.closest('button')) close(); }, opts);
  menu.querySelector('button')?.focus();
  return close;
}
// A menu that lives in the page (its items keep their ids): the trigger toggles it, arrows move,
// Esc closes and returns focus to the trigger, a click outside or on an item closes it.
export function bindMenu(trigger, menu) {
  let ac = null;
  const items = () => [...menu.querySelectorAll('button')];
  const close = (focus) => {
    if (menu.hidden) return;
    menu.hidden = true; trigger.setAttribute('aria-expanded', 'false'); ac?.abort(); ac = null;
    if (focus) trigger.focus();
  };
  const open = () => {
    activeMenuClose?.();
    menu.hidden = false; trigger.setAttribute('aria-expanded', 'true');
    const r = trigger.getBoundingClientRect();
    menu.style.top = Math.round(r.bottom + 6) + 'px'; menu.style.left = 'auto'; menu.style.right = Math.max(8, Math.round(innerWidth - r.right)) + 'px';
    ac = new AbortController(); const opts = { signal: ac.signal };
    document.addEventListener('pointerdown', (ev) => { if (!menu.contains(ev.target) && !trigger.contains(ev.target)) close(false); }, opts);
    document.addEventListener('keydown', (ev) => {
      const list = items(), i = list.indexOf(document.activeElement);
      if (ev.key === 'Escape') { ev.preventDefault(); ev.stopImmediatePropagation(); close(true); }
      else if (ev.key === 'ArrowDown' || ev.key === 'ArrowUp') { ev.preventDefault(); ev.stopImmediatePropagation(); list[(i + (ev.key === 'ArrowDown' ? 1 : -1) + list.length) % list.length]?.focus(); }
      else if (ev.key === 'Home' || ev.key === 'End') { ev.preventDefault(); ev.stopImmediatePropagation(); list[ev.key === 'Home' ? 0 : list.length - 1]?.focus(); }
      else if (ev.key === 'Tab') close(false);
    }, { ...opts, capture: true });
    window.addEventListener('resize', () => close(false), opts);
    items()[0]?.focus();
  };
  trigger.addEventListener('click', () => (menu.hidden ? open() : close(true)));
  menu.addEventListener('click', (ev) => { if (ev.target.closest?.('button')) close(false); });
  return { open, close };
}
// Patch text/attributes in place. Form controls keep their node, focus and unsaved value.
export function patchContent(target, source) {
  const old = [...target.childNodes], fresh = [...source.childNodes], used = new Set();
  const key = (n) => n.nodeType === 1 ? (n.getAttribute('data-drawer-key') || n.getAttribute('id') || n.getAttribute('aria-label')) : null;
  for (let i = 0; i < fresh.length; i++) {
    const b = fresh[i], id = key(b), current = target.childNodes[i];
    const a = id ? old.find((n) => !used.has(n) && key(n) === id && n.nodeName === b.nodeName)
      : current && !used.has(current) && !key(current) ? current : null;
    if (!a || a.nodeType !== b.nodeType || a.nodeName !== b.nodeName) {
      target.insertBefore(b, current || null); used.add(b); continue;
    }
    used.add(a);
    if (a !== current) target.insertBefore(a, current || null);
    if (a.nodeType === 3) { if (a.textContent !== b.textContent) a.textContent = b.textContent; continue; }
    if (a.matches('input,textarea,select')) continue;
    // Keep handlers that close over the retained controls. A different action gets a fresh node.
    if (a.matches('button') && a.textContent !== b.textContent) { a.replaceWith(b); used.delete(a); used.add(b); continue; }
    for (const attr of [...a.attributes]) if (!b.hasAttribute(attr.name)) a.removeAttribute(attr.name);
    for (const attr of b.attributes) if (a.getAttribute(attr.name) !== attr.value) a.setAttribute(attr.name, attr.value);
    patchContent(a, b);
  }
  for (const a of old) if (!used.has(a)) a.remove();
}
