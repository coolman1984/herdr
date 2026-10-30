"""Report and dashboard: one self-contained HTML file (no CDN, works offline and under company policy),
Arabic/RTL aware, light/dark, keyboard and screen-reader friendly. Every number links to the cells it
came from. Also a CSV of the records."""
import csv
import html
import json
from decimal import Decimal

from . import pipeline


def _fmt(d):
    q = Decimal(d).quantize(Decimal("0.01"))
    s = f"{q:,.2f}"
    return s


def build(db, out_html, out_csv, title, fields, references=None, lang="ar"):
    references = references or {}
    files = db.execute("SELECT id, path, kind, route, status, records, note FROM files ORDER BY id").fetchall()
    issues = db.execute("SELECT file_id, loc, level, note FROM issues ORDER BY id").fetchall()
    quality = pipeline.quality(db)
    cards, checks, prov = [], [], {}
    for f in fields:
        for cur, t in sorted(pipeline.totals(db, f).items()):
            key = f"{f} ({cur})" if cur != "-" else f
            cards.append((key, t["total"], t["count"]))
            prov[key] = [loc for _, loc in t["sources"][:500]]
            ref = references.get(key)
            if ref is not None:
                ok = Decimal(str(ref)) == t["total"]
                checks.append((key, Decimal(str(ref)), t["total"], ok))
    by_sheet = db.execute("""SELECT f.path, r.sheet, COUNT(DISTINCT r.row) FROM records r JOIN files f ON f.id=r.file_id
                             GROUP BY r.file_id, r.sheet ORDER BY f.id""").fetchall()
    maxrows = max([n for *_, n in by_sheet] or [1])
    rtl = lang == "ar"
    t = {
        "ar": dict(files="الملفات", status="الحالة", totals="المجاميع", check="مطابقة المجاميع المرجعية", src="المصدر",
                   issues="ملاحظات الجودة", rows="الصفوف لكل ورقة", none="لا يوجد", ok="مطابق", bad="غير مطابق",
                   ref="المرجع", got="المحسوب", count="عدد القيم", kind="النوع", route="المسار", recs="القيم",
                   skip="انتقل إلى المحتوى", show="اعرض المصادر"),
        "en": dict(files="Files", status="Status", totals="Totals", check="Reference totals", src="Source",
                   issues="Quality notes", rows="Rows per sheet", none="None", ok="matches", bad="MISMATCH",
                   ref="Reference", got="Computed", count="Values", kind="Kind", route="Route", recs="Values",
                   skip="Skip to content", show="Show sources"),
    }[lang]
    e = html.escape
    parts = [f"""<!doctype html><html lang="{lang}" dir="{'rtl' if rtl else 'ltr'}"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>{e(title)}</title>
<style>
:root{{--bg:#f7f7f5;--fg:#1d1d1b;--muted:#5d5d58;--card:#fff;--line:#d9d9d4;--accent:#0b5cad;--ok:#1f7a3a;--bad:#b3261e}}
@media (prefers-color-scheme:dark){{:root{{--bg:#151514;--fg:#ececea;--muted:#a9a9a3;--card:#1f1f1d;--line:#3a3a37;--accent:#7fb2ff;--ok:#6fcf8f;--bad:#ff8a80}}}}
*{{box-sizing:border-box}}body{{margin:0;background:var(--bg);color:var(--fg);font:16px/1.55 system-ui,"Segoe UI",Tahoma,sans-serif}}
a{{color:var(--accent)}}.skip{{position:absolute;inset-inline-start:-999px}}.skip:focus{{inset-inline-start:8px;top:8px;background:var(--card);padding:6px}}
main{{max-width:1100px;margin:0 auto;padding:16px}}h1{{font-size:1.5rem;margin:.2em 0 .6em}}h2{{font-size:1.15rem;margin:1.4em 0 .5em}}
.cards{{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px}}
.card{{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:14px}}
.card .v{{font-size:1.6rem;font-variant-numeric:tabular-nums;direction:ltr;unicode-bidi:isolate;display:block}}
.card .k{{color:var(--muted)}}table{{width:100%;border-collapse:collapse;background:var(--card);border:1px solid var(--line)}}
th,td{{padding:8px;border-bottom:1px solid var(--line);text-align:start;vertical-align:top}}td.n{{font-variant-numeric:tabular-nums;direction:ltr;unicode-bidi:isolate}}
.wrap{{overflow-x:auto}}.ok{{color:var(--ok);font-weight:600}}.bad{{color:var(--bad);font-weight:600}}
.bar{{fill:var(--accent)}}svg text{{fill:var(--fg);font-size:12px}}
details summary{{cursor:pointer;min-height:44px;display:flex;align-items:center}}
:focus-visible{{outline:3px solid var(--accent);outline-offset:2px}}
@media (prefers-reduced-motion:no-preference){{.card{{transition:transform .15s}}.card:hover{{transform:translateY(-1px)}}}}
code{{direction:ltr;unicode-bidi:isolate}}
</style></head><body><a class="skip" href="#main">{t['skip']}</a><main id="main" data-ready="1">
<h1>{e(title)}</h1><section aria-labelledby="h-totals"><h2 id="h-totals">{t['totals']}</h2><div class="cards">"""]
    for key, total, n in cards:
        parts.append(f"""<div class="card" data-field="{e(key)}"><span class="k">{e(key)}</span><span class="v" data-total="{total}">{_fmt(total)}</span>
<span class="k">{t['count']}: {n}</span><details><summary>{t['show']}</summary><code>{e(', '.join(prov[key][:50]))}{' …' if len(prov[key]) > 50 else ''}</code></details></div>""")
    if not cards:
        parts.append(f"<p>{t['none']}</p>")
    parts.append("</div></section>")
    if checks:
        parts.append(f"""<section aria-labelledby="h-check"><h2 id="h-check">{t['check']}</h2><div class="wrap"><table><thead><tr>
<th scope="col">{t['totals']}</th><th scope="col">{t['ref']}</th><th scope="col">{t['got']}</th><th scope="col">{t['status']}</th></tr></thead><tbody>""")
        for key, ref, got, ok in checks:
            parts.append(f"""<tr data-check="{e(key)}" data-ok="{int(ok)}"><td>{e(key)}</td><td class="n">{_fmt(ref)}</td><td class="n">{_fmt(got)}</td>
<td class="{'ok' if ok else 'bad'}">{t['ok'] if ok else t['bad']}</td></tr>""")
        parts.append("</tbody></table></div></section>")
    if by_sheet:
        h = 26 * len(by_sheet) + 10
        parts.append(f"""<section aria-labelledby="h-rows"><h2 id="h-rows">{t['rows']}</h2><svg role="img" aria-label="{t['rows']}" viewBox="0 0 600 {h}" width="100%" style="max-width:700px" direction="ltr">""")
        for i, (path, sheet, n) in enumerate(by_sheet):
            w = 360 * n / maxrows
            label = e(f"{path.rsplit('/', 1)[-1].rsplit(chr(92), 1)[-1]} / {sheet}")[:40]
            parts.append(f"""<text x="0" y="{i * 26 + 18}">{label}</text><rect class="bar" x="220" y="{i * 26 + 6}" width="{w:.1f}" height="16" rx="3"><title>{n}</title></rect>
<text x="{226 + w:.1f}" y="{i * 26 + 18}">{n}</text>""")
        parts.append("</svg></section>")
    parts.append(f"""<section aria-labelledby="h-files"><h2 id="h-files">{t['files']}</h2><div class="wrap"><table><thead><tr><th scope="col">#</th>
<th scope="col">{t['files']}</th><th scope="col">{t['kind']}</th><th scope="col">{t['route']}</th><th scope="col">{t['status']}</th><th scope="col">{t['recs']}</th></tr></thead><tbody>""")
    for fid, path, kind, route, status, n, note in files:
        parts.append(f"""<tr><td class="n">{fid}</td><td><code>{e(path)}</code></td><td>{e(kind)}</td><td>{e(route)}</td>
<td>{e(status)}{(' — ' + e(note)) if note and status != 'done' else ''}</td><td class="n">{n}</td></tr>""")
    parts.append(f"""</tbody></table></div></section><section aria-labelledby="h-issues"><h2 id="h-issues">{t['issues']}</h2><ul>""")
    for level, note in quality:
        parts.append(f"<li>{e(level)}: {e(note)}</li>")
    for fid, loc, level, note in issues[:300]:
        parts.append(f"<li>{e(level)}: {t['files']} {fid} <code>{e(loc or '')}</code> {e(note)}</li>")
    if not quality and not issues:
        parts.append(f"<li>{t['none']}</li>")
    parts.append("</ul></section></main></body></html>")
    with open(out_html, "w", encoding="utf-8") as f:
        f.write("".join(parts))
    from .export import write_csv
    with open(out_csv, "w", newline="", encoding="utf-8-sig") as f:
        write_csv(db, f)
    return {"cards": [(k, str(v), n) for k, v, n in cards], "checks": [(k, str(a), str(b), ok) for k, a, b, ok in checks],
            "quality": quality, "issues": len(issues)}
