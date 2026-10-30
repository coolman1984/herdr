"""Minimal stdlib PDF text reader: objects, FlateDecode streams, page order, text operators (Tj TJ ' ").
It reads simple text PDFs. It does NOT read scanned pages (no OCR), CID fonts without a usable
ToUnicode map, or encrypted PDFs: those come back with status needs-reader and confidence 0, never as
made-up text."""
import re
import zlib

_OBJ = re.compile(rb"(\d+)\s+(\d+)\s+obj\b(.*?)\bendobj", re.S)
_STREAM = re.compile(rb"stream\r?\n(.*?)\r?\nendstream", re.S)
_REF = re.compile(rb"(\d+)\s+0\s+R")


def _objects(data):
    return {int(m.group(1)): m.group(3) for m in _OBJ.finditer(data)}


def _stream(body):
    m = _STREAM.search(body)
    if not m:
        return b""
    raw = m.group(1)
    if b"/FlateDecode" in body.split(b"stream", 1)[0]:
        try:
            return zlib.decompress(raw)
        except zlib.error:
            try:
                return zlib.decompressobj().decompress(raw)
            except zlib.error:
                return b""
    return raw


def _literal(s, i):
    """Parse a (...) string starting after '('; return (bytes, next index)."""
    out, depth = bytearray(), 1
    esc = {ord("n"): 10, ord("r"): 13, ord("t"): 9, ord("b"): 8, ord("f"): 12}
    while i < len(s):
        c = s[i]
        if c == 0x5C:  # backslash
            i += 1
            d = s[i]
            if d in esc:
                out.append(esc[d])
            elif 0x30 <= d <= 0x37:
                j = i
                while j < min(i + 3, len(s)) and 0x30 <= s[j] <= 0x37:
                    j += 1
                out.append(int(s[i:j], 8) & 0xFF)
                i = j - 1
            elif d in (10, 13):
                pass
            else:
                out.append(d)
        elif c == 0x28:
            depth += 1
            out.append(c)
        elif c == 0x29:
            depth -= 1
            if depth == 0:
                return bytes(out), i + 1
            out.append(c)
        else:
            out.append(c)
        i += 1
    return bytes(out), i


def _decode(b):
    if b.startswith(b"\xfe\xff"):
        return b[2:].decode("utf-16-be", "replace")
    return b.decode("latin-1")


def _cmap(data):
    """Parse a ToUnicode CMap: code bytes -> text, and the code length."""
    m, width = {}, 1
    for block in re.findall(rb"beginbfchar(.*?)endbfchar", data, re.S):
        for src, dst in re.findall(rb"<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]*)>", block):
            width = len(src) // 2
            m[bytes.fromhex(src.decode())] = bytes.fromhex(dst.decode()).decode("utf-16-be", "replace")
    for block in re.findall(rb"beginbfrange(.*?)endbfrange", data, re.S):
        for lo, hi, dst in re.findall(rb"<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>\s*(<[0-9A-Fa-f]+>|\[[^\]]*\])", block):
            width = len(lo) // 2
            a, b = int(lo, 16), int(hi, 16)
            if dst.startswith(b"["):
                outs = re.findall(rb"<([0-9A-Fa-f]+)>", dst)
                for i, code in enumerate(range(a, b + 1)):
                    if i < len(outs):
                        m[code.to_bytes(width, "big")] = bytes.fromhex(outs[i].decode()).decode("utf-16-be", "replace")
            else:
                base = bytes.fromhex(dst[1:-1].decode())
                start = int.from_bytes(base, "big")
                for i, code in enumerate(range(a, b + 1)):
                    m[code.to_bytes(width, "big")] = (start + i).to_bytes(len(base), "big").decode("utf-16-be", "replace")
    return m, width


def _apply_cmap(raw, cm):
    if not cm:
        return _decode(raw)
    m, w = cm
    return "".join(m.get(raw[i:i + w], "") for i in range(0, len(raw) - w + 1, w))


def _text_ops(content, cmaps=None):
    """Text shown by Tj/TJ/'/" in a content stream, one string per text line (Td/T*/ET breaks)."""
    lines, cur = [], []
    i, n = 0, len(content)
    pending = []
    cmaps = cmaps or {}
    font = None
    in_array = False
    while i < n:
        c = content[i]
        if c == 0x2F:  # /Name: remember a font name for the next Tf
            j = i + 1
            while j < n and content[j] not in b" \n\r\t()<>[]/":
                j += 1
            name = content[i + 1:j]
            k = j
            while k < n and content[k] in b" \t\r\n0123456789.-":
                k += 1
            if content[k:k + 2] == b"Tf":
                font = name
                i = k + 2
                continue
            i = j
            continue
        if c == 0x28:
            s, i = _literal(content, i + 1)
            pending.append(_apply_cmap(s, cmaps.get(font)))
            continue
        if c == 0x3C and i + 1 < n and content[i + 1] != 0x3C:
            j = content.find(b">", i)
            hx = re.sub(rb"\s", b"", content[i + 1:j])
            if len(hx) % 2:
                hx += b"0"
            try:
                pending.append(_apply_cmap(bytes.fromhex(hx.decode()), cmaps.get(font)))
            except ValueError:
                pass
            i = j + 1
            continue
        m = re.match(rb"(Tj|TJ|'|\"|Td|TD|T\*|ET|Tm)\b", content[i:i + 3]) if c in b"TE'\"" else None
        if m:
            op = m.group(1)
            if op in (b"Tj", b"TJ", b"'", b'"'):
                if op in (b"'", b'"') and cur:
                    lines.append("".join(cur))
                    cur = []
                cur.append("".join(pending))
            elif op in (b"Td", b"TD", b"T*", b"ET", b"Tm") and cur:
                lines.append("".join(cur))
                cur = []
            pending = []
            i += len(op)
            continue
        if c in b"]":
            in_array = False
            i += 1
            continue
        if c in b"[":
            in_array = True
            i += 1
            continue
        if c in b"\n\r\t ":
            i += 1
            continue
        # operands we do not need
        j = i
        while j < n and content[j] not in b" \n\r\t()<>[]/":
            j += 1
        if in_array and re.fullmatch(rb"-?[\d.]+", content[i:j]) and float(content[i:j]) <= -200:
            pending.append(" ")  # a wide gap inside TJ is a word space
            i = j
            continue
        if j == i:
            j += 1
        if content[i:j] not in (b"Tj", b"TJ"):
            if not re.fullmatch(rb"-?[\d.]+", content[i:j]) and content[i:i + 1] != b"/":
                pending = [] if content[i:j] not in (b"Tf",) else pending
        i = j
    if cur:
        lines.append("".join(cur))
    return [l for l in (x.strip() for x in lines) if l]


def _resolve(objs, body, key):
    m = re.search(rb"/" + key + rb"\s*(\d+)\s+0\s+R", body)
    if m:
        return objs.get(int(m.group(1)), b"")
    m = re.search(rb"/" + key + rb"\s*<<", body)
    if not m:
        return b""
    depth, i = 0, m.end() - 2
    while i < len(body) - 1:
        if body[i:i + 2] == b"<<":
            depth += 1
            i += 2
            continue
        if body[i:i + 2] == b">>":
            depth -= 1
            i += 2
            if depth == 0:
                return body[m.end() - 2:i]
            continue
        i += 1
    return b""


def _page_cmaps(objs, page):
    """Font resource name -> ToUnicode map for one page (page or inherited /Resources)."""
    res = _resolve(objs, page, b"Resources")
    fonts = _resolve(objs, res, b"Font")
    out = {}
    for name, ref in re.findall(rb"/([^\s/<>\[\]()]+)\s+(\d+)\s+0\s+R", fonts):
        fobj = objs.get(int(ref), b"")
        tu = re.search(rb"/ToUnicode\s+(\d+)\s+0\s+R", fobj)
        if tu:
            cm = _cmap(_stream(objs.get(int(tu.group(1)), b"")))
            if cm[0]:
                out[name] = cm
    return out


_AR_RUN = re.compile("[\u0621-\u064a\u0671-\u06d3\ufb50-\ufdff\ufe70-\ufefc][\u0621-\u0652\u0671-\u06d3\ufb50-\ufdff\ufe70-\ufefc ]*")


def _logical(text):
    """PDF producers store Arabic glyphs in visual (left-to-right) order. Reverse each Arabic letter run
    back to reading order; digits and Latin keep theirs. Returns (text, reordered?)."""
    if not _AR_RUN.search(text):
        return text, False
    return _AR_RUN.sub(lambda m: m.group(0)[::-1].strip() if m.group(0).strip() else m.group(0), text), True


def pdf_records(path):
    """Yield dicts: page, line, text, confidence. Status records (where='status') explain gaps."""
    with open(path, "rb") as f:
        data = f.read()
    if b"/Encrypt" in data:
        yield {"page": 0, "line": 0, "text": "", "status": "needs-reader", "why": "encrypted PDF"}
        return
    objs = _objects(data)
    pages = [k for k, v in sorted(objs.items()) if re.search(rb"/Type\s*/Page\b", v)]
    # page order from the /Kids tree when present
    root_pages = [v for v in objs.values() if re.search(rb"/Type\s*/Pages\b", v) and b"/Parent" not in v]
    if root_pages:
        order = []

        def walk(body):
            m = re.search(rb"/Kids\s*\[(.*?)\]", body, re.S)
            for ref in _REF.findall(m.group(1) if m else b""):
                k = int(ref)
                b = objs.get(k, b"")
                if re.search(rb"/Type\s*/Pages\b", b):
                    walk(b)
                elif k in objs:
                    order.append(k)
        walk(root_pages[0])
        if order:
            pages = order
    any_text = False
    for pno, k in enumerate(pages, 1):
        body = objs[k]
        m = re.search(rb"/Contents\s*(\[(.*?)\]|(\d+)\s+0\s+R)", body, re.S)
        refs = _REF.findall(m.group(0)) if m else []
        content = b"\n".join(_stream(objs.get(int(r), b"")) for r in refs)
        lines = _text_ops(content, _page_cmaps(objs, body))
        if not lines:
            yield {"page": pno, "line": 0, "text": "", "status": "needs-reader",
                   "why": "no text layer (scanned page or image): OCR not available here"}
            continue
        for ln, text in enumerate(lines, 1):
            printable = sum(ch.isprintable() for ch in text) / max(1, len(text))
            conf = 0.9 if printable > 0.95 else 0.3
            text, reordered = _logical(text)
            if reordered:
                conf = min(conf, 0.8)  # reading order rebuilt from visual order
            any_text = True
            yield {"page": pno, "line": ln, "text": text, "confidence": conf,
                   "status": "ok" if conf > 0.5 else "low-confidence"}
    if not pages:
        yield {"page": 0, "line": 0, "text": "", "status": "needs-reader", "why": "no pages found by the stdlib reader"}
