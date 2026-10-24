// Small DOM helpers, toasts and dialogs (native <dialog>: focus trap + Esc for free).
export function el(tag, attrs, ...kids) {
  const n = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (v == null || v === false) continue;
    if (k.startsWith('on')) n.addEventListener(k.slice(2), v);
    else if (k === 'text') n.textContent = v;
    else if (k === 'class') n.className = v;
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
