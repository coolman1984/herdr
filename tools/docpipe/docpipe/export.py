"""Spreadsheet-safe export. A human-facing CSV opened in Excel must never run a formula: a text cell that
starts with = + - @ (or a tab/CR) gets a leading apostrophe. Numbers stay numbers. Rows are streamed from
the database, never collected in memory."""
import csv

HEADER = ["file", "sheet", "row", "field", "number", "text", "date", "currency", "source"]
QUERY = """SELECT f.path, r.sheet, r.row, r.field, r.num, r.text, r.date, r.currency, r.loc
           FROM records r JOIN files f ON f.id=r.file_id ORDER BY r.file_id, r.sheet, r.row, r.field"""
_DANGER = ("=", "+", "-", "@", "\t", "\r")


def safe(v, numeric=False):
    if v is None or numeric:
        return v
    s = str(v)
    return "'" + s if s.startswith(_DANGER) else s


def safe_row(row):
    return [safe(v, numeric=(i in (2, 4))) for i, v in enumerate(row)]


def write_csv(db, fh):
    w = csv.writer(fh)
    w.writerow(HEADER)
    for row in db.execute(QUERY):
        w.writerow(safe_row(row))


def stream_csv(db, chunk=1 << 16):
    """Yield UTF-8 chunks of the CSV (BOM first) without building the whole file."""
    import io
    buf = io.StringIO()
    w = csv.writer(buf)
    buf.write("\ufeff")
    w.writerow(HEADER)
    for row in db.execute(QUERY):
        w.writerow(safe_row(row))
        if buf.tell() >= chunk:
            yield buf.getvalue().encode("utf-8")
            buf.seek(0)
            buf.truncate()
    yield buf.getvalue().encode("utf-8")
