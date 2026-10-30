"""discover -> route -> extract (native stdlib or authorized Office over COM) -> normalise -> store in
batches with checkpoints -> quality checks. A record is never dropped silently: anything not stored as a
value becomes an issue with its location."""
import json
import os
import platform
import subprocess
import tempfile

from . import normalize as N
from . import ooxml, pdf, sniff, store


class Crash(Exception):
    """Raised by tests to cut an import between batches."""


def _cell_from_xlsx(rec):
    raw, kind = rec["raw"], rec["kind"]
    flags = []
    if rec["formula"]:
        flags.append("formula: stored value from the last Excel save, not recalculated")
    if rec["row_hidden"]:
        flags.append("hidden row")
    if rec["sheet_hidden"]:
        flags.append("hidden sheet")
    if rec["merged"]:
        flags.append("merged " + rec["merged"])
    num = date = text = cur = None
    conf, method = 1.0, "xlsx-xml"
    if kind == "n" and raw is not None:
        if rec["is_date"]:
            try:
                date = N.excel_serial_to_date(raw, rec["date1904"]).date().isoformat()
                method += " date-serial"
            except (ValueError, OverflowError):
                text, conf = raw, 0.5
        else:
            num = str(N.Decimal(raw))
    elif kind in ("s", "str") and raw is not None:
        text = raw
        v, info = N.parse_number(raw)
        if v is not None:
            num, cur, conf = str(v), N.currency_of(raw), 0.9
            method += " number-from-text (" + (info or "plain") + ")"
        else:
            d, dc, how = N.parse_date(raw)
            if d:
                date, conf, method = d, dc, method + " date-from-text " + how
    elif kind == "b":
        text = "TRUE" if raw == "1" else "FALSE"
    elif kind == "e":
        text, conf = raw, 0.0
        flags.append("Excel error value")
    return num, date, text, cur, conf, method, flags


def _records_from(path, kind):
    """Yield (loc, sheet, row, col, raw, kind, num, date, text, currency, conf, method, flags)."""
    if kind == "xlsx":
        for r in ooxml.xlsx_records(path):
            num, date, text, cur, conf, method, flags = _cell_from_xlsx(r)
            yield (f"{r['sheet']}!{r['cell']}", r["sheet"], r["row"], r["col"], r["raw"], r["kind"], num, date, text, cur, conf, method, flags)
    elif kind == "docx":
        for r in ooxml.docx_records(path):
            v, info = N.parse_number(r["text"]) if len(r["text"]) < 40 else (None, "")
            yield (r["where"], None, None, None, r["text"], "text", str(v) if v is not None else None, None, r["text"],
                   N.currency_of(r["text"]), 1.0, "docx-xml", ["rtl"] if r["rtl"] else [])
    elif kind == "pptx":
        for r in ooxml.pptx_records(path):
            v, _ = N.parse_number(r["text"]) if len(r["text"]) < 40 else (None, "")
            yield (r["where"], None, None, None, r["text"], "text", str(v) if v is not None else None, None, r["text"],
                   N.currency_of(r["text"]), 1.0, "pptx-xml", [])
    elif kind == "pdf":
        for r in pdf.pdf_records(path):
            if r.get("status") == "needs-reader":
                yield (f"page{r['page']}", None, r["page"], None, None, "status", None, None, None, None, 0.0,
                       "pdf-stdlib", ["needs-reader: " + r["why"]])
                continue
            yield (f"page{r['page']}:line{r['line']}", None, r["page"], r["line"], r["text"], "text", None, None, r["text"],
                   N.currency_of(r["text"]), r["confidence"], "pdf-stdlib", [] if r["status"] == "ok" else [r["status"]])
    elif kind in ("csv", "txt"):
        import csv
        raw = open(path, "rb").read()
        for enc in ("utf-8-sig", "cp1256", "latin-1"):
            try:
                text = raw.decode(enc)
                break
            except UnicodeDecodeError:
                continue
        try:
            dialect = csv.Sniffer().sniff(text[:4096], delimiters=",;\t|")
        except csv.Error:
            dialect = csv.excel
        for ri, row in enumerate(csv.reader(text.splitlines(), dialect), 1):
            for ci, val in enumerate(row, 1):
                if val == "":
                    continue
                v, info = N.parse_number(val)
                d = N.parse_date(val) if v is None else (None, 0, "")
                yield (f"csv!R{ri}C{ci}", "csv", ri, ci, val, "s", str(v) if v is not None else None, d[0], val,
                       N.currency_of(val), 0.9 if v is not None else (d[1] or 1.0), f"csv ({enc})", [])


def com_extract(path, timeout=300):
    """Run the PowerShell COM worker in its own process with a timeout. Returns (records, error)."""
    if platform.system() != "Windows":
        return None, "Office COM exists only on Windows; file kept for an authorized Windows run"
    worker = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "com-extract.ps1")
    out = tempfile.NamedTemporaryFile(suffix=".json", delete=False).name
    shell = "pwsh" if _which("pwsh") else "powershell"
    try:
        p = subprocess.run([shell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", worker,
                            "-Path", path, "-Out", out], capture_output=True, text=True, timeout=timeout)
    except subprocess.TimeoutExpired:
        return None, f"COM worker timed out after {timeout}s (Office dialog or hang); its own Office instance is closed by the worker's finally block"
    if p.returncode != 0:
        return None, "COM worker failed: " + (p.stderr or p.stdout)[-400:]
    with open(out, encoding="utf-8") as f:
        recs = json.load(f)
    os.remove(out)
    return recs, None


def _which(name):
    from shutil import which
    return which(name)


def import_file(db, path, profile="personal", batch=2000, crash_after_batches=None):
    """Import one file; returns (file_id, status). Re-running after a cut resumes at the checkpoint;
    re-running after success does nothing (same SHA-256)."""
    sha = sniff.sha256_file(path)
    kind = sniff.kind_of(path)
    route, why = sniff.route(path, kind, profile)
    row = store.file_row(db, sha)
    if row and row[1] == "done":
        return row[0], "already-imported"
    fid = store.add_file(db, sha, path, kind, os.path.getsize(path), route, "importing", why)
    for level, note in sniff.safety(path, kind):
        if level == "block":
            store.write_batch(db, fid, 0, [], [], [(fid, "", "block", note)])
            store.finish_file(db, fid, "quarantined", note)
            return fid, "quarantined"
        db.execute("INSERT INTO issues(file_id,loc,level,note) SELECT ?,?,?,? WHERE NOT EXISTS (SELECT 1 FROM issues WHERE file_id=? AND note=?)",
                   (fid, "", level, note, fid, note))
    if route == "skip":
        store.finish_file(db, fid, "skipped", why)
        return fid, "skipped"
    if route in ("com", "app-pdf"):
        recs, err = com_extract(path) if route == "com" else (None, why)
        if err:
            store.finish_file(db, fid, "needs-authorized-app", err)
            return fid, "needs-authorized-app"
        source = ((r["loc"], r.get("sheet"), r.get("row"), r.get("col"), r.get("raw"), r.get("kind", "s"),
                   *_normalise_com(r)) for r in recs)
    else:
        source = _records_from(path, kind)
    start = store.checkpoint(db, fid)
    header = {}
    cells, records, issues, seq, batches = [], [], [], 0, 0
    for rec in source:
        seq += 1
        loc, sheet, r, c, raw, k, num, date, text, cur, conf, method, flags = rec
        if r is not None and c is not None and sheet is not None and text and num is None and date is None and (sheet, r) not in header and _is_header_row(sheet, r, header):
            pass
        _track_header(header, sheet, r, c, text, num, date)
        if seq <= start:
            continue  # committed before the cut
        cells.append((fid, loc, sheet, r, c, raw, num, date, text, cur, k, conf, method, "; ".join(flags) or None))
        field = header.get((sheet, "cols"), {}).get(c) if sheet is not None and r is not None and r > header.get((sheet, "row"), 1 << 30) else None
        if field:
            records.append((fid, sheet, r, field, num, text if num is None else None, date, cur, loc))
        for fl in flags:
            if fl.startswith(("needs-reader", "Excel error", "low-confidence")):
                issues.append((fid, loc, "warn", fl))
        if conf < 0.7 and not flags:
            issues.append((fid, loc, "warn", f"low confidence {conf} ({method})"))
        if len(cells) >= batch:
            store.write_batch(db, fid, seq, cells, records, issues)
            cells, records, issues = [], [], []
            batches += 1
            if crash_after_batches and batches >= crash_after_batches:
                raise Crash(f"cut after {batches} batches")
    store.write_batch(db, fid, seq, cells, records, issues)
    status = "done"
    if db.execute("SELECT COUNT(*) FROM issues WHERE file_id=? AND note LIKE 'needs-reader%'", (fid,)).fetchone()[0]:
        status = "partial"
    store.finish_file(db, fid, status)
    return fid, status


def _is_header_row(*a):
    return False


def _track_header(header, sheet, r, c, text, num, date):
    """The first row of a sheet holding only text is its header; columns map to field names."""
    if sheet is None or r is None or c is None:
        return
    hr = header.get((sheet, "row"))
    if hr is None:
        if text and num is None and date is None:
            header[(sheet, "row")] = r
            header.setdefault((sheet, "cols"), {})[c] = text.strip()
        return
    if r == hr and text:
        header[(sheet, "cols")][c] = text.strip()


def _normalise_com(r):
    """COM records give Value2 (numbers as numbers, dates as serials when IsDate) and Text."""
    raw, kind = r.get("raw"), r.get("kind", "s")
    num = date = text = cur = None
    conf, method, flags = 1.0, "office-com", []
    if r.get("formula"):
        flags.append("formula: value as Excel holds it (Excel computed; read via COM)")
    if kind == "n" and raw is not None:
        if r.get("is_date"):
            date = N.excel_serial_to_date(raw).date().isoformat()
        else:
            num = str(N.Decimal(str(raw)))
    elif raw is not None:
        text = str(raw)
        v, _ = N.parse_number(text)
        if v is not None:
            num, cur, conf = str(v), N.currency_of(text), 0.9
    return num, date, text, cur, conf, method, flags


def quality(db):
    """Mixed types per field, missing values, duplicate rows, fields that differ between files."""
    out = []
    for field, nums, texts in db.execute("""SELECT field, SUM(num IS NOT NULL), SUM(num IS NULL AND text IS NOT NULL)
                                            FROM records GROUP BY field"""):
        if nums and texts:
            out.append(("warn", f"field '{field}': {nums} numbers and {texts} texts (mixed types)"))
    dup = db.execute("""SELECT COUNT(*) FROM (SELECT file_id, sheet, GROUP_CONCAT(field||'='||COALESCE(num,text,date,''),'|') sig
                         FROM (SELECT * FROM records ORDER BY field) GROUP BY file_id, sheet, row)
                         GROUP BY sig HAVING COUNT(*) > 1""").fetchall()
    if dup:
        out.append(("warn", f"{sum(1 for _ in dup)} duplicated row contents"))
    shapes = db.execute("""SELECT file_id, GROUP_CONCAT(DISTINCT field) FROM records GROUP BY file_id""").fetchall()
    sets = {fid: set(fs.split(",")) for fid, fs in shapes}
    if len({frozenset(s) for s in sets.values()}) > 1:
        out.append(("warn", "column sets differ between files: " + "; ".join(f"file {k}: {sorted(v)}" for k, v in sets.items())))
    return out


def totals(db, field):
    """Exact Decimal totals of a numeric field per currency, with the count and the source cells."""
    out = {}
    for num, cur, loc, fid in db.execute("SELECT num, currency, loc, file_id FROM records WHERE field=? AND num IS NOT NULL", (field,)):
        k = cur or "-"
        t = out.setdefault(k, {"total": N.Decimal(0), "count": 0, "sources": []})
        t["total"] += N.Decimal(num)
        t["count"] += 1
        t["sources"].append((fid, loc))
    return out
