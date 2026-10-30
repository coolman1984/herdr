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


def route(path, kind, profile):
    """Which extractor may read this file, and why.

    company_nasca: every Office/PDF file goes to the authorized application (COM) whatever its bytes look
    like - a protected file may be a wrapper, and a parser failing on it does not mean it is damaged.
    Elsewhere: our own stdlib readers; a CFB file (legacy binary or encrypted) needs the application."""
    ext = os.path.splitext(path)[1].lower()
    if profile == "company_nasca" and (ext in OFFICE_EXT or kind in ("xlsx", "docx", "pptx", "pdf", "cfb")):
        if kind == "pdf" or ext == ".pdf":
            return "app-pdf", "company_nasca: PDF needs its authorized reader's automation; none is wired"
        return "com", "company_nasca: authorized Office application over COM"
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
