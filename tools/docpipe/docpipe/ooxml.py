"""Stdlib readers for xlsx / docx / pptx (Office Open XML). Streaming (iterparse), so a sheet with a
million rows is read in bounded memory. Every record carries its exact location. Formula cells give the
value Excel stored at the last save: they are marked stored-not-recalculated, never presented as a new
calculation."""
import posixpath
import re
import xml.etree.ElementTree as ET
import zipfile

NS = {
    "m": "http://schemas.openxmlformats.org/spreadsheetml/2006/main",
    "r": "http://schemas.openxmlformats.org/officeDocument/2006/relationships",
    "rel": "http://schemas.openxmlformats.org/package/2006/relationships",
    "w": "http://schemas.openxmlformats.org/wordprocessingml/2006/main",
    "a": "http://schemas.openxmlformats.org/drawingml/2006/main",
    "p": "http://schemas.openxmlformats.org/presentationml/2006/main",
}
R_ID = "{%s}id" % NS["r"]
_BUILTIN_DATE_FMTS = set(range(14, 23)) | {45, 46, 47}
_DATE_FMT = re.compile(r"(?<![\\\"])(d|m|y|h|s)", re.I)


def _q(prefix, tag):
    return "{%s}%s" % (NS[prefix], tag)


def _rels(z, part):
    """Relationship id -> target path for a part."""
    d, f = posixpath.split(part)
    rp = posixpath.join(d, "_rels", f + ".rels")
    if rp not in z.namelist():
        return {}
    root = ET.fromstring(z.read(rp))
    out = {}
    for r in root.findall("rel:Relationship", NS):
        t = r.get("Target")
        out[r.get("Id")] = t.lstrip("/") if t.startswith("/") else posixpath.normpath(posixpath.join(d, t))
    return out


def col_to_num(col):
    n = 0
    for ch in col:
        n = n * 26 + ord(ch) - 64
    return n


_REF = re.compile(r"([A-Z]+)(\d+)")


# ---------- xlsx ----------

# Limits that bound memory before allocation on hostile or broken files.
MAX_MERGE_CELLS = 100_000          # a merged range larger than this is not expanded cell by cell
MAX_SHARED_STRING_CHARS = 200_000_000
MAX_XML_PART_BYTES = 256 << 20     # a docx/pptx part loaded whole must be smaller than this
MAX_ROWS = 5_000_000


class LimitExceeded(Exception):
    pass


def xlsx_records(path):
    """Yield dicts: sheet, sheet_hidden, row, col, cell, raw, kind (n/s/b/d/e/str), formula, row_hidden,
    is_date, date1904, merged (range or None)."""
    with zipfile.ZipFile(path) as z:
        wb = ET.fromstring(z.read("xl/workbook.xml"))
        pr = wb.find("m:workbookPr", NS)
        date1904 = pr is not None and pr.get("date1904") in ("1", "true")
        calc = wb.find("m:calcPr", NS)
        stale = calc is not None and calc.get("fullCalcOnLoad") in ("1", "true")
        rels = _rels(z, "xl/workbook.xml")
        strings = _shared_strings(z)
        datestyles = _date_styles(z)
        for sh in wb.find("m:sheets", NS).findall("m:sheet", NS):
            name, state = sh.get("name"), sh.get("state", "visible")
            part = rels.get(sh.get(R_ID))
            if not part or part not in z.namelist():
                continue
            merged = _merged(z, part)
            for rec in _sheet(z, part, name, state != "visible", strings, datestyles, date1904, merged):
                rec["stale_calc"] = stale
                yield rec


def _shared_strings(z):
    out = []
    if "xl/sharedStrings.xml" not in z.namelist():
        return out
    total = 0
    with z.open("xl/sharedStrings.xml") as f:
        for ev, el in ET.iterparse(f, events=("end",)):
            if el.tag == _q("m", "si"):
                s = "".join(t.text or "" for t in el.iter(_q("m", "t")))
                total += len(s)
                if total > MAX_SHARED_STRING_CHARS:
                    raise LimitExceeded(f"shared strings exceed {MAX_SHARED_STRING_CHARS} characters")
                out.append(s)
                el.clear()
    return out


def _date_styles(z):
    """cellXfs index -> (is_date, currency or None) from the number format."""
    if "xl/styles.xml" not in z.namelist():
        return {}
    root = ET.fromstring(z.read("xl/styles.xml"))
    from .normalize import currency_of
    custom = {}
    nf = root.find("m:numFmts", NS)
    if nf is not None:
        for f in nf.findall("m:numFmt", NS):
            raw = f.get("formatCode", "")
            code = re.sub(r"\[[^\]]*\]|\"[^\"]*\"", "", raw)
            cur = currency_of(re.sub(r"\[\$([^-\]]*)[^\]]*\]", r" \1 ", raw).replace('"', " "))
            custom[int(f.get("numFmtId"))] = (bool(_DATE_FMT.search(code)), cur)
    out = {}
    xfs = root.find("m:cellXfs", NS)
    if xfs is not None:
        for i, xf in enumerate(xfs.findall("m:xf", NS)):
            fid = int(xf.get("numFmtId", "0"))
            is_date, cur = custom.get(fid, (fid in _BUILTIN_DATE_FMTS, None))
            if is_date or cur:
                out[i] = (is_date, cur)
    return out


def _merged(z, part):
    out = {}
    with z.open(part) as f:
        for ev, el in ET.iterparse(f, events=("end",)):
            if el.tag == _q("m", "mergeCell"):
                ref = el.get("ref", "")
                if ":" in ref:
                    a, b = ref.split(":")
                    (c1, r1), (c2, r2) = _REF.match(a).groups(), _REF.match(b).groups()
                    area = (int(r2) - int(r1) + 1) * (col_to_num(c2) - col_to_num(c1) + 1)
                    if area > MAX_MERGE_CELLS:
                        out[(int(r1), col_to_num(c1))] = ref + " (huge merge, not expanded)"
                        continue
                    for r in range(int(r1), int(r2) + 1):
                        for c in range(col_to_num(c1), col_to_num(c2) + 1):
                            out[(r, c)] = ref
            elif el.tag == _q("m", "row"):
                el.clear()
    return out


def _sheet(z, part, name, hidden, strings, datestyles, date1904, merged):
    with z.open(part) as f:
        row_hidden = False
        for ev, el in ET.iterparse(f, events=("start", "end")):
            if ev == "start" and el.tag == _q("m", "row"):
                row_hidden = el.get("hidden") in ("1", "true")
                continue
            if ev != "end":
                continue
            if el.tag == _q("m", "c"):
                ref = el.get("r")
                t = el.get("t", "n")
                v = el.find("m:v", NS)
                fml = el.find("m:f", NS)
                raw = None
                if t == "inlineStr":
                    raw = "".join(x.text or "" for x in el.iter(_q("m", "t")))
                    t = "str"
                elif v is not None:
                    raw = v.text
                    if t == "s":
                        raw = strings[int(raw)] if raw is not None and int(raw) < len(strings) else None
                if raw is not None or fml is not None:
                    col, row = _REF.match(ref).groups()
                    r, c = int(row), col_to_num(col)
                    if r > MAX_ROWS:
                        raise LimitExceeded(f"sheet {name} has more than {MAX_ROWS} rows")
                    style = datestyles.get(int(el.get("s", "0")), (False, None))
                    yield {
                        "sheet": name, "sheet_hidden": hidden, "row": r, "col": c, "cell": ref, "raw": raw,
                        "kind": t, "formula": fml is not None, "row_hidden": row_hidden,
                        "is_date": t == "n" and style[0], "currency": style[1], "date1904": date1904,
                        "merged": merged.get((r, c)),
                    }
                el.clear()
            elif el.tag == _q("m", "row"):
                el.clear()


# ---------- docx ----------

def docx_records(path):
    """Yield dicts: where (p<n> or t<n>r<r>c<c>), text, rtl, style."""
    with zipfile.ZipFile(path) as z:
        if z.getinfo("word/document.xml").file_size > MAX_XML_PART_BYTES:
            raise LimitExceeded("word/document.xml is too large to read whole")
        root = ET.fromstring(z.read("word/document.xml"))
    body = root.find("w:body", NS)
    p_no = t_no = 0
    for el in body:
        if el.tag == _q("w", "p"):
            p_no += 1
            text = _wtext(el)
            if text.strip():
                ppr = el.find("w:pPr", NS)
                rtl = ppr is not None and ppr.find("w:bidi", NS) is not None
                style = ppr.find("w:pStyle", NS).get(_q("w", "val")) if ppr is not None and ppr.find("w:pStyle", NS) is not None else ""
                yield {"where": f"p{p_no}", "text": text, "rtl": rtl, "style": style}
        elif el.tag == _q("w", "tbl"):
            t_no += 1
            for ri, tr in enumerate(el.findall("w:tr", NS), 1):
                for ci, tc in enumerate(tr.findall("w:tc", NS), 1):
                    text = " ".join(_wtext(p) for p in tc.findall("w:p", NS)).strip()
                    if text:
                        yield {"where": f"t{t_no}r{ri}c{ci}", "text": text, "rtl": False, "style": "table",
                               "table": f"table{t_no}", "row": ri, "col": ci}


def _wtext(p):
    return "".join(t.text or "" for t in p.iter(_q("w", "t")))


# ---------- pptx ----------

def pptx_records(path):
    """Yield dicts: where (s<n>:<shape name>[:r<r>c<c>]), text."""
    with zipfile.ZipFile(path) as z:
        pres = ET.fromstring(z.read("ppt/presentation.xml"))
        rels = _rels(z, "ppt/presentation.xml")
        lst = pres.find("p:sldIdLst", NS)
        slides = [rels[s.get(R_ID)] for s in (lst.findall("p:sldId", NS) if lst is not None else [])]
        for n, part in enumerate(slides, 1):
            root = ET.fromstring(z.read(part))
            for sp in root.iter():
                if sp.tag not in (_q("p", "sp"), _q("p", "graphicFrame")):
                    continue
                nv = sp.find(".//p:cNvPr", NS)
                name = nv.get("name") if nv is not None else "shape"
                tbl = sp.find(".//a:tbl", NS)
                if tbl is not None:
                    for ri, tr in enumerate(tbl.findall("a:tr", NS), 1):
                        for ci, tc in enumerate(tr.findall("a:tc", NS), 1):
                            text = "".join(t.text or "" for t in tc.iter(_q("a", "t"))).strip()
                            if text:
                                yield {"where": f"s{n}:{name}:r{ri}c{ci}", "text": text, "table": f"s{n}:{name}", "row": ri, "col": ci}
                    continue
                for pi, para in enumerate(sp.iter(_q("a", "p")), 1):
                    text = "".join(t.text or "" for t in para.iter(_q("a", "t"))).strip()
                    if text:
                        yield {"where": f"s{n}:{name}:p{pi}", "text": text}
