"""Discovery: hash, size, kind from the bytes' first block (never from the extension alone), and the
route each file must take under the device policy."""
import hashlib
import os
import zipfile

CFB = b"\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1"  # OLE compound file: legacy .xls/.doc/.ppt, or an encrypted OOXML/DRM wrapper


def sha256_file(path, block=1 << 20):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(block), b""):
            h.update(chunk)
    return h.hexdigest()


def kind_of(path):
    """xlsx/docx/pptx/pdf/csv/txt/cfb/zip/unknown from content."""
    with open(path, "rb") as f:
        head = f.read(8)
    if head.startswith(b"%PDF"):
        return "pdf"
    if head == CFB:
        return "cfb"
    if head.startswith(b"PK"):
        try:
            with zipfile.ZipFile(path) as z:
                names = set(z.namelist())
        except zipfile.BadZipFile:
            return "unknown"
        if "xl/workbook.xml" in names:
            return "xlsx"
        if "word/document.xml" in names:
            return "docx"
        if "ppt/presentation.xml" in names:
            return "pptx"
        return "zip"
    ext = os.path.splitext(path)[1].lower()
    if ext in (".csv", ".tsv"):
        return "csv"
    if ext in (".txt", ".md"):
        return "txt"
    return "unknown"


OFFICE_EXT = {".xlsx", ".xlsm", ".xls", ".docx", ".docm", ".doc", ".pptx", ".pptm", ".ppt", ".pdf"}


# What the authorized Office applications open on a company PC (Excel also opens CSV/TXT/TSV).
COM_EXT = {".xlsx", ".xlsm", ".xls", ".xlsb", ".csv", ".tsv", ".txt", ".prn",
           ".docx", ".docm", ".doc", ".rtf", ".pptx", ".pptm", ".ppt"}


def policy_route(path, profile):
    """Decide the route from the file's NAME ONLY when a policy requires it - before any byte is read.
    company_nasca: every file goes to an authorized application; a type with no authorized reader is a
    documented blocked capability, never a raw-parser fallback. Returns (route, why) or (None, '')."""
    if profile != "company_nasca":
        return None, ""
    ext = os.path.splitext(path)[1].lower()
    if ext in COM_EXT:
        return "com", "company_nasca: authorized Office application over COM (PowerShell bridge)"
    if ext == ".pdf":
        return "blocked", "company_nasca: no authorized PDF reader automation is wired on this device"
    return "blocked", f"company_nasca: no authorized application is known for {ext or 'files without an extension'}"


def kind_from_name(path):
    """Kind from the extension only (used when content must not be read)."""
    ext = os.path.splitext(path)[1].lower()
    return {".xlsx": "xlsx", ".xlsm": "xlsx", ".docx": "docx", ".docm": "docx", ".pptx": "pptx", ".pptm": "pptx",
            ".pdf": "pdf", ".csv": "csv", ".tsv": "csv", ".txt": "txt"}.get(ext, ext.lstrip(".") or "unknown")


def route(path, kind, profile):
    """Which extractor may read this file, and why.

    company_nasca: every Office/PDF file goes to the authorized application (COM) whatever its bytes look
    like - a protected file may be a wrapper, and a parser failing on it does not mean it is damaged.
    Elsewhere: our own stdlib readers; a CFB file (legacy binary or encrypted) needs the application."""
    if profile == "company_nasca":
        return policy_route(path, profile)
    if kind in ("xlsx", "docx", "pptx", "pdf", "csv", "txt"):
        return "native", "stdlib reader"
    if kind == "cfb":
        return "com", "binary or protected Office file: needs the Office application"
    return "skip", "unsupported kind " + kind


def safety(path, kind, max_ratio=100, max_unpacked=2 << 30):
    """Zip-bomb, macro and external-link checks before any read. Returns list of (level, note)."""
    notes = []
    if kind not in ("xlsx", "docx", "pptx", "zip"):
        return notes
    try:
        with zipfile.ZipFile(path) as z:
            infos = z.infolist()
    except zipfile.BadZipFile:
        return [("block", "damaged zip container")]
    packed = sum(i.compress_size for i in infos) or 1
    unpacked = sum(i.file_size for i in infos)
    if unpacked > max_unpacked or unpacked / packed > max_ratio:
        notes.append(("block", f"compression ratio {unpacked / packed:.0f}x / {unpacked >> 20} MB unpacked: possible zip bomb"))
    names = [i.filename for i in infos]
    if any(n.endswith("vbaProject.bin") for n in names):
        notes.append(("warn", "contains macros (never run; values read only)"))
    if any("externalLink" in n for n in names):
        notes.append(("warn", "external links (not followed; cached values only)"))
    return notes
