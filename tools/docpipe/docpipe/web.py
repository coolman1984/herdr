"""A small responsive web app over a docpipe database (standard library http.server, read-only):
dashboard -> files -> one file's values with their sources -> search -> CSV download.
Arabic first (RTL), English with ?lang=en; light/dark; works with keyboard and screen readers; pages carry
data-ready="1" when their content is in place (tests wait on that, never on network idle).
   python -m docpipe serve --db work.db [--port 8765] [--host 127.0.0.1]"""
import html
import json
import sqlite3
from contextlib import closing
import urllib.parse
from decimal import Decimal
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from . import pipeline

T = {
    "ar": dict(title="لوحة المستندات", files="الملفات", search="بحث في القيم", totals="المجاميع", values="القيم",
               loc="الموضع", raw="النص الأصلي", num="الرقم", conf="الثقة", method="الطريقة", flags="ملاحظات",
               empty="لا توجد ملفات بعد. شغّل: python -m docpipe run", none="لا نتائج", csv="تنزيل CSV",
               back="رجوع إلى اللوحة", notfound="غير موجود", q="كلمة أو رقم", go="ابحث", loading="جارٍ التحميل…",
               failed="تعذّر التحميل. حاول مرة أخرى.", status="الحالة", kind="النوع", lang="English", more="التالي",
               count="عدد", issues="ملاحظات الجودة", skip="انتقل إلى المحتوى"),
    "en": dict(title="Documents dashboard", files="Files", search="Search values", totals="Totals", values="Values",
               loc="Location", raw="Original text", num="Number", conf="Confidence", method="Method", flags="Notes",
               empty="No files yet. Run: python -m docpipe run", none="No results", csv="Download CSV",
               back="Back to dashboard", notfound="Not found", q="word or number", go="Search", loading="Loading…",
               failed="Could not load. Try again.", status="Status", kind="Kind", lang="العربية", more="Next",
               count="Count", issues="Quality notes", skip="Skip to content"),
}

CSS = """:root{--bg:#f7f7f5;--fg:#1b1b19;--muted:#55554f;--card:#fff;--line:#d6d6d0;--accent:#0a58a8;--accent-fg:#fff}
@media (prefers-color-scheme:dark){:root{--bg:#141413;--fg:#efefec;--muted:#b0b0a9;--card:#1f1f1d;--line:#3b3b38;--accent:#8ab8ff;--accent-fg:#0b0b0a}}
*{box-sizing:border-box}html{-webkit-text-size-adjust:100%}body{margin:0;background:var(--bg);color:var(--fg);font:16px/1.6 system-ui,"Segoe UI",Tahoma,"Noto Naskh Arabic",sans-serif}
a{color:var(--accent)}.skip{position:absolute;inset-inline-start:-9999px}.skip:focus{inset-inline-start:8px;top:8px;background:var(--card);padding:8px;z-index:9}
header{display:flex;gap:12px;align-items:center;justify-content:space-between;flex-wrap:wrap;padding:12px 16px;border-bottom:1px solid var(--line);background:var(--card)}
header nav{display:flex;gap:16px;flex-wrap:wrap}header a{min-height:44px;display:inline-flex;align-items:center}
main{max-width:1100px;margin:0 auto;padding:16px}h1{font-size:1.4rem;margin:.2em 0 .7em}h2{font-size:1.1rem;margin:1.3em 0 .5em}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:12px}
.card{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:14px}.card .v{font-size:1.5rem;font-variant-numeric:tabular-nums;direction:ltr;unicode-bidi:isolate;display:block}
.muted{color:var(--muted)}table{width:100%;border-collapse:collapse;background:var(--card);border:1px solid var(--line)}
th,td{padding:8px 10px;border-bottom:1px solid var(--line);text-align:start;vertical-align:top}.n{font-variant-numeric:tabular-nums;direction:ltr;unicode-bidi:isolate}
.wrap{overflow-x:auto;-webkit-overflow-scrolling:touch}code{direction:ltr;unicode-bidi:isolate;word-break:break-all}
form{display:flex;gap:8px;flex-wrap:wrap}input[type=search]{flex:1 1 220px;min-height:44px;padding:8px 12px;font:inherit;color:var(--fg);background:var(--card);border:1px solid var(--muted);border-radius:8px}
button,.btn{min-height:44px;padding:8px 16px;font:inherit;border:0;border-radius:8px;background:var(--accent);color:var(--accent-fg);cursor:pointer;text-decoration:none;display:inline-flex;align-items:center}
:focus-visible{outline:3px solid var(--accent);outline-offset:2px}[aria-busy=true]{opacity:.6}
.state{padding:16px;border:1px dashed var(--line);border-radius:10px;background:var(--card)}
@media (max-width:600px){th.opt,td.opt{display:none}main{padding:12px}}
@media (prefers-reduced-motion:no-preference){.card{transition:transform .12s ease-out}.card:hover{transform:translateY(-1px)}}"""

JS = """const f=document.querySelector('#search');if(f){const out=document.querySelector('#results');
f.addEventListener('submit',async e=>{e.preventDefault();const q=f.q.value.trim();if(!q){out.innerHTML='';return}
out.setAttribute('aria-busy','true');out.dataset.ready='0';out.innerHTML='<p class="state" role="status">'+f.dataset.loading+'</p>';
try{const ctl=new AbortController();const t=setTimeout(()=>ctl.abort(),10000);
const r=await fetch('/api/search?lang='+f.dataset.lang+'&q='+encodeURIComponent(q),{signal:ctl.signal});clearTimeout(t);
if(!r.ok)throw new Error(r.status);out.innerHTML=await r.text();}
catch(err){out.innerHTML='<p class="state" role="alert">'+f.dataset.failed+'</p>'}
finally{out.removeAttribute('aria-busy');out.dataset.ready='1';history.replaceState(null,'','?lang='+f.dataset.lang+'&q='+encodeURIComponent(q))}});
const q0=new URLSearchParams(location.search).get('q');if(q0){f.q.value=q0;f.requestSubmit()}}"""


def page(lang, title, body):
    t = T[lang]
    other = "en" if lang == "ar" else "ar"
    return f"""<!doctype html><html lang="{lang}" dir="{'rtl' if lang == 'ar' else 'ltr'}"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>{html.escape(title)} · docpipe</title><style>{CSS}</style></head>
<body><a class="skip" href="#main">{t['skip']}</a><header><strong>docpipe</strong><nav aria-label="main">
<a href="/?lang={lang}">{t['title']}</a><a href="/search?lang={lang}">{t['search']}</a><a href="/records.csv">{t['csv']}</a>
<a href="?lang={other}" lang="{other}">{t['lang']}</a></nav></header><main id="main" data-ready="1">{body}</main><script>{JS}</script></body></html>"""


def fmt(v):
    return f"{Decimal(v):,.2f}"


class App:
    def __init__(self, db_path, fields=None, export_ok=True):
        self.db_path = db_path
        self.fields = fields
        self.export_ok = export_ok

    def db(self):
        """A read-only connection that closes when the request is done (use with `with closing(...)`)."""
        db = sqlite3.connect(f"file:{self.db_path}?mode=ro", uri=True, timeout=10)
        return db

    def dashboard(self, lang):
        with closing(self.db()) as db:
            return self._dashboard(lang, db)

    def _dashboard(self, lang, db):
        t, e = T[lang], html.escape
        files = db.execute("SELECT id, path, kind, status, records FROM files ORDER BY id").fetchall()
        if not files:
            return page(lang, t["title"], f"<h1>{t['title']}</h1><p class='state' role='status'>{t['empty']}</p>")
        fields = self.fields or [r[0] for r in db.execute(
            "SELECT field FROM records GROUP BY field HAVING SUM(num IS NOT NULL) > 0 AND SUM(num IS NOT NULL) >= COUNT(*) / 2 ORDER BY field")]
        cards = []
        for fld in fields:
            for cur, tt in sorted(pipeline.totals(db, fld).items()):
                label = fld + (f" ({cur})" if cur != "-" else "")
                cards.append(f"<div class='card'><span class='muted'>{e(label)}</span><span class='v' data-total='{tt['total']}'>{fmt(tt['total'])}</span><span class='muted'>{t['count']}: {tt['count']}</span></div>")
        rows = "".join(f"<tr><td class='n'>{fid}</td><td><a href='/file/{fid}?lang={lang}'><code>{e(p.rsplit('/', 1)[-1].rsplit(chr(92), 1)[-1])}</code></a></td><td class='opt'>{e(k)}</td><td>{e(s)}</td><td class='n'>{n}</td></tr>"
                       for fid, p, k, s, n in files)
        issues = pipeline.quality(db)
        il = "".join(f"<li>{e(lv)}: {e(n)}</li>" for lv, n in issues) or f"<li>{t['none']}</li>"
        return page(lang, t["title"], f"""<h1>{t['title']}</h1><section aria-labelledby="h1"><h2 id="h1">{t['totals']}</h2>
<div class="cards">{''.join(cards) or t['none']}</div></section><section aria-labelledby="h2"><h2 id="h2">{t['files']}</h2>
<div class="wrap"><table><thead><tr><th scope="col">#</th><th scope="col">{t['files']}</th><th scope="col" class="opt">{t['kind']}</th>
<th scope="col">{t['status']}</th><th scope="col">{t['values']}</th></tr></thead><tbody>{rows}</tbody></table></div></section>
<section aria-labelledby="h3"><h2 id="h3">{t['issues']}</h2><ul>{il}</ul></section>""")

    def values_table(self, lang, rows):
        t, e = T[lang], html.escape
        if not rows:
            return f"<p class='state' role='status'>{t['none']}</p>"
        body = "".join(f"<tr><td><code>{e(loc)}</code></td><td>{e(raw or '')}</td><td class='n'>{e(num) if num else ''}</td>"
                       f"<td class='n opt'>{conf:.1f}</td><td class='opt'>{e(method)}</td><td class='opt'>{e(flags or '')}</td></tr>"
                       for loc, raw, num, conf, method, flags in rows)
        return f"""<div class="wrap"><table><thead><tr><th scope="col">{t['loc']}</th><th scope="col">{t['raw']}</th><th scope="col">{t['num']}</th>
<th scope="col" class="opt">{t['conf']}</th><th scope="col" class="opt">{t['method']}</th><th scope="col" class="opt">{t['flags']}</th></tr></thead><tbody>{body}</tbody></table></div>"""

    def file(self, lang, fid, offset):
        with closing(self.db()) as db:
            return self._file(lang, fid, offset, db)

    def _file(self, lang, fid, offset, db):
        t, e = T[lang], html.escape
        f = db.execute("SELECT path, kind, route, status, note, sha256 FROM files WHERE id=?", (fid,)).fetchone()
        if not f:
            return None
        rows = db.execute("SELECT loc, raw, num, confidence, method, flags FROM cells WHERE file_id=? ORDER BY id LIMIT 101 OFFSET ?", (fid, offset)).fetchall()
        more = f"<p><a class='btn' href='/file/{fid}?lang={lang}&offset={offset + 100}'>{t['more']}</a></p>" if len(rows) > 100 else ""
        return page(lang, f[0], f"""<p><a href="/?lang={lang}">{t['back']}</a></p><h1><code>{e(f[0])}</code></h1>
<p class="muted">{e(f[1])} · {e(f[2])} · {e(f[3])}{(' — ' + e(f[4])) if f[4] else ''} · SHA-256 <code>{f[5][:16]}…</code></p>
<h2>{t['values']}</h2>{self.values_table(lang, rows[:100])}{more}""")

    def search_page(self, lang):
        t = T[lang]
        return page(lang, t["search"], f"""<h1>{t['search']}</h1><form id="search" role="search" data-lang="{lang}" data-loading="{t['loading']}" data-failed="{t['failed']}">
<label for="q" class="muted">{t['q']}</label><input id="q" name="q" type="search" autocomplete="off" required><button type="submit">{t['go']}</button></form>
<div id="results" aria-live="polite" data-ready="1"></div>""")

    def search(self, lang, q):
        with closing(self.db()) as db:
            return self._search(lang, q, db)

    def _search(self, lang, q, db):
        like = "%" + q.replace("%", "").replace("_", "") + "%"
        rows = db.execute("""SELECT (SELECT path FROM files WHERE id=c.file_id) || ' — ' || c.loc, c.raw, c.num, c.confidence, c.method, c.flags
                             FROM cells c WHERE c.raw LIKE ? OR c.num = ? ORDER BY c.id LIMIT 100""", (like, q)).fetchall()
        return self.values_table(lang, rows)

    def csv_chunks(self):
        from .export import stream_csv
        with closing(self.db()) as db:
            yield from stream_csv(db)


def make_handler(app):
    class H(BaseHTTPRequestHandler):
        def log_message(self, *a):
            pass

        def send(self, code, body, ctype="text/html; charset=utf-8", extra=None):
            b = body.encode("utf-8")
            self.send_response(code)
            self.send_header("Content-Type", ctype)
            self.send_header("Content-Length", str(len(b)))
            self.send_header("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'")
            self.send_header("X-Content-Type-Options", "nosniff")
            for k, v in (extra or {}).items():
                self.send_header(k, v)
            self.end_headers()
            self.wfile.write(b)

        def do_GET(self):
            u = urllib.parse.urlparse(self.path)
            qs = urllib.parse.parse_qs(u.query)
            lang = qs.get("lang", ["ar"])[0]
            lang = lang if lang in T else "ar"
            try:
                if u.path == "/":
                    return self.send(200, app.dashboard(lang))
                if u.path == "/search":
                    return self.send(200, app.search_page(lang))
                if u.path == "/api/search":
                    return self.send(200, app.search(lang, qs.get("q", [""])[0][:200]))
                if u.path.startswith("/file/"):
                    try:
                        fid = int(u.path.split("/")[2])
                        off = max(0, int(qs.get("offset", ["0"])[0]))
                    except ValueError:
                        fid, off = -1, 0
                    body = app.file(lang, fid, off)
                    if body:
                        return self.send(200, body)
                if u.path == "/records.csv":
                    if not app.export_ok:
                        return self.send(403, page(lang, "export", "<p class='state' role='alert'>Export is not approved for this data (company policy).</p>"))
                    self.send_response(200)
                    self.send_header("Content-Type", "text/csv; charset=utf-8")
                    self.send_header("Content-Disposition", "attachment; filename=records.csv")
                    self.send_header("X-Content-Type-Options", "nosniff")
                    self.end_headers()
                    for chunk in app.csv_chunks():  # streamed: no whole-file buffer
                        self.wfile.write(chunk)
                    return
                if u.path == "/healthz":
                    return self.send(200, json.dumps({"ok": True}), "application/json")
                t = T[lang]
                return self.send(404, page(lang, t["notfound"], f"<h1>{t['notfound']}</h1><p><a href='/?lang={lang}'>{t['back']}</a></p>"))
            except sqlite3.Error as err:
                return self.send(503, page(lang, "error", f"<p class='state' role='alert'>{html.escape(str(err))}</p>"))
    return H


LOOPBACK = ("127.0.0.1", "::1", "localhost")


def serve(db_path, host="127.0.0.1", port=8765, allow_remote=False, export_ok=True):
    """Loopback only unless allow_remote: this server has no login; on a network it would show the data
    to anyone who can reach the port."""
    if host not in LOOPBACK and not allow_remote:
        raise SystemExit(f"refusing to serve on {host}: no authentication; pass --allow-remote only behind an authenticating proxy")
    srv = ThreadingHTTPServer((host, port), make_handler(App(db_path, export_ok=export_ok)))
    print(f"docpipe web on http://{host}:{srv.server_address[1]}/", flush=True)
    srv.serve_forever()
