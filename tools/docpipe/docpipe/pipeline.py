"""discover -> route -> extract (native stdlib or authorized Office over COM) -> normalise -> store in
batches with checkpoints -> reconcile -> quality checks. A value is never dropped silently: anything not
stored as a value becomes an issue with its location.

Company policy (docs/COMPANY-ENVIRONMENT-POLICY.md) is applied BEFORE any read of a file's content: under the
company_nasca profile the route is decided from the file's name and metadata alone, no hash, magic bytes,
ZIP listing or safety scan touches its bytes, and only the authorized application reads it."""
import codecs
import csv
import hashlib
import json
import os
import platform
import sqlite3
import subprocess
import tempfile
import time
import uuid
from decimal import Decimal, Inexact, localcontext

from . import normalize as N
from . import ooxml, pdf, sniff, store

PARSER_VERSION = "3"  # bump when extraction changes: earlier imports are then not reused
MAX_SOURCES = 1000    # source pointers kept per total (the count is always exact)


class Crash(Exception):
    """Raised by tests to cut an import between batches."""


# ---------- normalisation of one extracted value ----------

def _date_serial(raw, date1904):
    return N.excel_serial_to_date(Decimal(str(raw)), date1904)


def _cell_from_xlsx(rec):
    raw, kind = rec["raw"], rec["kind"]
    flags = []
    if rec["formula"]:
        flags.append("formula: stored value from the last Excel save, not recalculated" + (
            "; workbook asks for a full recalculation on load, so this cache may be stale" if rec.get("stale_calc") else ""))
    if rec["row_hidden"]:
        flags.append("hidden row")
    if rec["sheet_hidden"]:
        flags.append("hidden sheet")
    if rec["merged"]:
        flags.append("merged " + rec["merged"])
    num = date = text = None
    cur = rec.get("currency")
    conf, method = 1.0, "xlsx-xml"
    if kind == "n" and raw is not None:
        if rec["is_date"]:
            try:
                d = _date_serial(raw, rec["date1904"])
                date = d.date().isoformat() if d.time() == d.min.time() else d.isoformat(timespec="seconds")
                method += " date-serial" + (" 1904" if rec["date1904"] else "")
            except (ValueError, OverflowError, ArithmeticError):
                text, conf = raw, 0.5
                flags.append("date serial out of range")
        else:
            num = str(Decimal(raw))
            if cur:
                method += " currency-format " + cur
    elif kind == "d" and raw is not None:  # ISO 8601 date cell
        d, dc, how = N.parse_date(raw[:10])
        if d:
            date, conf, method = (raw if len(raw) > 10 else d), dc, method + " iso-date-cell"
        else:
            text, conf = raw, 0.5
            flags.append("unreadable ISO date cell")
    elif kind in ("s", "str", "inlineStr") and raw is not None:
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
    if raw is not None and num is None and date is None and text is None:
        text, conf = str(raw), 0.5  # never let a non-empty value vanish
        flags.append(f"unrecognised cell type {kind!r}: kept as text")
    return num, date, text, cur, conf, method, flags


def _normalise_com(r):
    """COM records carry Value2 (numbers as numbers, dates as serials when the cell is a date), the
    workbook's Date1904 flag and the cell's currency when its number format has one."""
    raw, kind = r.get("raw"), r.get("kind", "s")
    num = date = text = None
    cur = r.get("currency")
    conf, method, flags = 1.0, "office-com", []
    if r.get("formula"):
        flags.append("formula: value as Excel holds it now (Excel's own calculation, read via COM)")
    if kind == "n" and raw is not None:
        if r.get("is_date"):
            d = _date_serial(raw, bool(r.get("date1904")))
            date = d.date().isoformat() if d.time() == d.min.time() else d.isoformat(timespec="seconds")
            method += " date-serial" + (" 1904" if r.get("date1904") else "")
        else:
            num = str(Decimal(str(raw)))
    elif raw is not None:
        text = str(raw)
        v, _ = N.parse_number(text)
        if v is not None:
            num, cur, conf = str(v), cur or N.currency_of(text), 0.9
    return num, date, text, cur, conf, method, flags


# ---------- native extraction (personal profile only) ----------

def _detect_encoding(path):
    """Decode the whole file incrementally (bounded memory) to pick an encoding that fits every byte."""
    for enc in ("utf-8-sig", "cp1256"):
        dec = codecs.getincrementaldecoder(enc)()
        try:
            with open(path, "rb") as f:
                for chunk in iter(lambda: f.read(1 << 20), b""):
                    dec.decode(chunk)
                dec.decode(b"", final=True)
            return enc
        except UnicodeDecodeError:
            continue
    return "latin-1"


def _csv_records(path):
    enc = _detect_encoding(path)
    with open(path, encoding=enc, newline="") as f:  # newline="": quoted line breaks stay in the value
        sample = f.read(8192)
        f.seek(0)
        try:
            dialect = csv.Sniffer().sniff(sample, delimiters=",;\t|")
        except csv.Error:
            dialect = csv.excel
        for ri, row in enumerate(csv.reader(f, dialect), 1):
            for ci, val in enumerate(row, 1):
                if val == "":
                    continue
                v, info = N.parse_number(val)
                d = N.parse_date(val) if v is None else (None, 0, "")
                yield (f"csv!R{ri}C{ci}", "csv", ri, ci, val, "s", str(v) if v is not None else None, d[0], val,
                       N.currency_of(val), 0.9 if v is not None else (d[1] or 1.0), f"csv ({enc}, delimiter {dialect.delimiter!r})", [])


def _records_from(path, kind):
    """Yield (loc, sheet, row, col, raw, kind, num, date, text, currency, conf, method, flags)."""
    if kind == "xlsx":
        for r in ooxml.xlsx_records(path):
            num, date, text, cur, conf, method, flags = _cell_from_xlsx(r)
            yield (f"{r['sheet']}!{r['cell']}", r["sheet"], r["row"], r["col"], r["raw"], r["kind"], num, date, text, cur, conf, method, flags)
    elif kind == "docx":
        for r in ooxml.docx_records(path):
            v, info = N.parse_number(r["text"]) if len(r["text"]) < 40 else (None, "")
            table = r.get("table")
            yield (r["where"], table, r.get("row"), r.get("col"), r["text"], "text", str(v) if v is not None else None, None, r["text"],
                   N.currency_of(r["text"]), 1.0, "docx-xml", ["rtl"] if r["rtl"] else [])
    elif kind == "pptx":
        for r in ooxml.pptx_records(path):
            v, _ = N.parse_number(r["text"]) if len(r["text"]) < 40 else (None, "")
            yield (r["where"], r.get("table"), r.get("row"), r.get("col"), r["text"], "text", str(v) if v is not None else None, None, r["text"],
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
        yield from _csv_records(path)


# ---------- the authorized application (Office over COM, through PowerShell) ----------

def _which(name):
    from shutil import which
    return which(name)


def work_dir(db_path=None):
    """Where extracted company content may be written: DOCPIPE_WORK_DIR, else next to the database, 0700."""
    d = os.environ.get("DOCPIPE_WORK_DIR") or os.path.join(os.path.dirname(os.path.abspath(db_path or ".")), ".docpipe-work")
    os.makedirs(d, exist_ok=True)
    try:
        os.chmod(d, 0o700)
    except OSError:
        pass
    return d


def com_extract(path, timeout=300, workdir=None):
    """Run the PowerShell COM worker in its own process with a timeout; read its JSON-lines output as a
    stream. Returns (records iterator or list, error). The device's execution policy is respected: a
    blocked policy is reported as a capability fact, never bypassed."""
    if platform.system() != "Windows":
        return None, "Office COM exists only on Windows; file kept for an authorized Windows run"
    shell = "pwsh" if _which("pwsh") else ("powershell" if _which("powershell") else None)
    if not shell:
        return None, "no PowerShell found: the authorized-application bridge is unavailable"
    worker = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "com-extract.ps1")
    wd = workdir or work_dir()
    os.makedirs(wd, exist_ok=True)
    out = os.path.join(wd, f"com-{uuid.uuid4().hex}.jsonl")
    pidfile = out + ".pid"
    try:
        try:
            p = subprocess.run([shell, "-NoProfile", "-NonInteractive", "-File", worker, "-Path", path, "-Out", out],
                               capture_output=True, text=True, timeout=timeout)
        except subprocess.TimeoutExpired:
            killed = _kill_owned_office(pidfile)
            return None, f"COM worker timed out after {timeout}s (Office dialog or hang); {killed}"
        err = (p.stderr or p.stdout)[-400:]
        if p.returncode != 0:
            if "running scripts is disabled" in err or "execution polic" in err.lower():
                return None, "PowerShell execution policy blocks the worker on this device (capability blocked; not bypassed)"
            return None, f"COM worker failed (exit {p.returncode}): {err}"
        with open(out, encoding="utf-8") as f:
            return [json.loads(line) for line in f if line.strip()], None
    finally:
        for fn in (out, pidfile):  # extracted company content never stays behind, success or not
            try:
                os.remove(fn)
            except OSError:
                pass


def _kill_owned_office(pidfile):
    """Kill only the Office process the worker created (its PID file); never the owner's instances."""
    try:
        pid = int(open(pidfile).read().strip())
    except (OSError, ValueError):
        return "no Office process was recorded as created by the worker; nothing killed"
    subprocess.run(["taskkill", "/PID", str(pid), "/T", "/F"], capture_output=True)
    return f"killed the worker's own Office process {pid}"


# ---------- import ----------

def _identity(path, profile):
    """(identity, content_sha or None, stat). company_nasca: metadata only (no content read)."""
    st = os.stat(path)
    if profile != "personal":
        ident = f"meta|{os.path.abspath(path)}|{st.st_size}|{st.st_mtime_ns}|{profile}|{PARSER_VERSION}"
        return _sha(ident), None, st
    content = sniff.sha256_file(path)
    return _sha(f"{content}|{profile}|{PARSER_VERSION}"), content, st


def _sha(text):
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


class Headers:
    """Header row per sheet/table: first all-text row. Duplicate, empty and multi-line names are made
    unique and reported, so two source columns never collapse into one field."""

    def __init__(self):
        self.row, self.cols, self.issues = {}, {}, []

    def see(self, sheet, r, c, text, num, date, loc):
        if sheet is None or r is None or c is None:
            return
        hr = self.row.get(sheet)
        if hr is None and text and num is None and date is None:
            self.row[sheet] = hr = r
        if r == hr and text:
            name = " ".join(text.split())
            if name != text.strip():
                self.issues.append((loc, "warn", f"header {text!r} has line breaks/extra spaces: used {name!r}"))
            used = set(self.cols.setdefault(sheet, {}).values())
            base, n = name, 1
            while name in used:
                n += 1
                name = f"{base}#{n}"
            if n > 1:
                self.issues.append((loc, "warn", f"duplicate header {base!r}: this column is field {name!r}"))
            self.cols[sheet][c] = name

    def field(self, sheet, r, c, loc):
        hr = self.row.get(sheet)
        if sheet is None or r is None or c is None or hr is None or r <= hr:
            return None
        cols = self.cols.setdefault(sheet, {})
        if c not in cols:
            cols[c] = f"column {c}"
            self.issues.append((loc, "warn", f"value under a column with no header: field 'column {c}'"))
        return cols[c]


def import_file(db, path, profile="personal", batch=None, crash_after_batches=None, owner=None):
    """Import one file; returns (file_id, status). Statuses: done, partial, already-imported, busy,
    quarantined, skipped, needs-authorized-app, source-changed, disk-full."""
    batch = batch or int(os.environ.get("DOCPIPE_BATCH_ROWS", "2000"))
    owner = owner or f"{os.getpid()}-{uuid.uuid4().hex[:8]}"
    route, why = sniff.policy_route(path, profile)  # name/metadata only
    ident, content_sha, st0 = _identity(path, profile)
    row = store.file_row(db, ident)
    if row and row[1] == "done":
        return row[0], "already-imported"
    kind = sniff.kind_of(path) if route is None else sniff.kind_from_name(path)
    if route is None:
        route, why = sniff.route(path, kind, profile)
    fid = store.add_file(db, ident, path, kind, st0.st_size, route, "importing", why, content_sha, profile)
    if not store.take_lease(db, fid, owner):
        return fid, "busy"
    try:
        return _import(db, fid, path, kind, route, why, profile, batch, crash_after_batches, st0)
    except (store.DiskFull, sqlite3.OperationalError) as e:
        if not isinstance(e, store.DiskFull) and "full" not in str(e).lower():
            raise
        try:
            store.finish_file(db, fid, "disk-full", str(e))
        except Exception:
            pass
        return fid, "disk-full"
    finally:
        try:
            store.release_lease(db, fid, owner)
        except Exception:
            pass


def _import(db, fid, path, kind, route, why, profile, batch, crash_after_batches, st0):
    if route == "native":
        for level, note in sniff.safety(path, kind):
            if level == "block":
                store.write_batch(db, fid, 0, [], [], [(fid, "", "block", note)])
                store.finish_file(db, fid, "quarantined", note)
                return fid, "quarantined"
            db.execute("INSERT INTO issues(file_id,loc,level,note) SELECT ?,?,?,? WHERE NOT EXISTS (SELECT 1 FROM issues WHERE file_id=? AND note=?)",
                       (fid, "", level, note, fid, note))
    if route in ("skip", "blocked"):
        status = "skipped" if route == "skip" else "needs-authorized-app"
        store.finish_file(db, fid, status, why)
        return fid, status
    if route in ("com", "app-pdf"):
        recs, err = com_extract(path, workdir=work_dir(store.db_path(db))) if route == "com" else (None, why)
        if err:
            store.finish_file(db, fid, "needs-authorized-app", err)
            return fid, "needs-authorized-app"
        source = ((r["loc"], r.get("sheet"), r.get("row"), r.get("col"), r.get("raw"), r.get("kind", "s"),
                   *_normalise_com(r)) for r in recs)
    else:
        source = _records_from(path, kind)
    start = store.checkpoint(db, fid)
    heads = Headers()
    cells, records, issues, seq, batches, n_issue = [], [], [], 0, 0, 0
    for rec in source:
        seq += 1
        loc, sheet, r, c, raw, k, num, date, text, cur, conf, method, flags = rec
        heads.see(sheet, r, c, text, num, date, loc)
        field = heads.field(sheet, r, c, loc)
        while n_issue < len(heads.issues):
            hl, lvl, note = heads.issues[n_issue]
            if seq > start:
                issues.append((fid, hl, lvl, note))
            n_issue += 1
        if seq <= start:
            continue  # committed before the cut
        cells.append((fid, loc, sheet, r, c, raw, num, date, text, cur, k, conf, method, "; ".join(flags) or None))
        if field:
            records.append((fid, sheet, r, field, num, text if num is None else None, date, cur, loc))
        for fl in flags:
            if fl.startswith(("needs-reader", "Excel error", "low-confidence", "unrecognised", "unreadable", "date serial")):
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
    st1 = os.stat(path)
    if (st1.st_size, st1.st_mtime_ns) != (st0.st_size, st0.st_mtime_ns):
        store.discard_file_rows(db, fid)
        store.finish_file(db, fid, "source-changed", "the file changed while it was read; nothing kept, import again")
        return fid, "source-changed"
    note = reconcile(db, fid)
    status = "done"
    if db.execute("SELECT COUNT(*) FROM issues WHERE file_id=? AND (note LIKE 'needs-reader%' OR note LIKE 'reconcile:%')", (fid,)).fetchone()[0]:
        status = "partial"
    store.finish_file(db, fid, status, note)
    return fid, status


def reconcile(db, fid):
    """Every value below a header must be one record; report the counts and any difference."""
    cells = db.execute("SELECT COUNT(*) FROM cells WHERE file_id=?", (fid,)).fetchone()[0]
    recs = db.execute("SELECT COUNT(*) FROM records WHERE file_id=?", (fid,)).fetchone()[0]
    expected = 0
    for sheet, hr in db.execute("""SELECT sheet, MIN(row) FROM cells WHERE file_id=? AND sheet IS NOT NULL AND row IS NOT NULL
                                   AND text IS NOT NULL AND num IS NULL AND date IS NULL GROUP BY sheet""", (fid,)).fetchall():
        expected += db.execute("SELECT COUNT(*) FROM cells WHERE file_id=? AND sheet=? AND row>? AND col IS NOT NULL", (fid, sheet, hr)).fetchone()[0]
    note = f"reconcile: {cells} cells, {expected} under headers, {recs} records"
    if expected and expected != recs:
        db.execute("INSERT INTO issues(file_id,loc,level,note) VALUES(?,?,?,?)", (fid, "", "error", note + " (values not mapped)"))
    return note.replace("reconcile: ", "")


def quality(db):
    """Mixed types per field, duplicate rows, column sets that differ between files."""
    out = []
    for field, nums, texts in db.execute("""SELECT field, SUM(num IS NOT NULL), SUM(num IS NULL AND text IS NOT NULL)
                                            FROM records GROUP BY field"""):
        if nums and texts:
            out.append(("warn", f"field '{field}': {nums} numbers and {texts} texts (mixed types)"))
    dup = db.execute("""SELECT COUNT(*) FROM (SELECT file_id, sheet, GROUP_CONCAT(field||'='||COALESCE(num,text,date,''),'|') sig
                         FROM (SELECT * FROM records ORDER BY field) GROUP BY file_id, sheet, row)
                         GROUP BY sig HAVING COUNT(*) > 1""").fetchall()
    if dup:
        out.append(("warn", f"{len(dup)} duplicated row contents"))
    shapes = db.execute("""SELECT file_id, GROUP_CONCAT(DISTINCT field) FROM records GROUP BY file_id""").fetchall()
    sets = {fid: set(fs.split(",")) for fid, fs in shapes}
    if len({frozenset(s) for s in sets.values()}) > 1:
        out.append(("warn", "column sets differ between files: " + "; ".join(f"file {k}: {sorted(v)}" for k, v in sets.items())))
    return out


def totals(db, field):
    """Exact totals of a numeric field per currency. Arithmetic runs with 200 significant digits and
    raises if any sum would be rounded (Inexact), so a total is either exact or an error - never rounded
    silently. Keeps the first MAX_SOURCES source cells; the count is always complete."""
    out = {}
    with localcontext() as ctx:
        ctx.prec = 200
        ctx.traps[Inexact] = True
        for num, cur, loc, fid, fl in db.execute("""SELECT r.num, r.currency, r.loc, r.file_id, c.flags FROM records r
                LEFT JOIN cells c ON c.file_id=r.file_id AND c.loc=r.loc WHERE r.field=? AND r.num IS NOT NULL""", (field,)):
            k = cur or "-"
            t = out.setdefault(k, {"total": Decimal(0), "count": 0, "sources": [], "formulas": 0, "formula_sources": []})
            if fl and "formula" in fl:
                # A computed cell (often a subtotal or total row) is not added again: listed separately.
                t["formulas"] += 1
                if len(t["formula_sources"]) < MAX_SOURCES:
                    t["formula_sources"].append((fid, loc))
                continue
            t["total"] += Decimal(num)
            t["count"] += 1
            if len(t["sources"]) < MAX_SOURCES:
                t["sources"].append((fid, loc))
    return out
