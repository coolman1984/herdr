"""Synthetic Office/PDF files written with the standard library only (no company data, no packages)."""
import zipfile
from xml.sax.saxutils import escape

CT = '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>\n'


def _col(n):
    s = ""
    while n:
        n, r = divmod(n - 1, 26)
        s = chr(65 + r) + s
    return s


def make_xlsx(path, sheets, hidden_rows=(), date_cells=(), formulas=None, merged=None, hidden_sheets=(), stream_rows=None):
    """sheets: {name: [[cell, ...], ...]} cells are str (shared string), int/float/Decimal-text numbers,
    or ('n', '123') raw numbers. date_cells: {(sheet,'B2')}. formulas: {(sheet,'C5'): 'SUM(C2:C4)'}.
    stream_rows: (sheet, generator of rows) appended after, written without holding all rows in memory."""
    formulas = formulas or {}
    merged = merged or {}
    names = list(sheets) + ([stream_rows[0]] if stream_rows and stream_rows[0] not in sheets else [])
    strings, sidx = [], {}

    def si(s):
        if s not in sidx:
            sidx[s] = len(strings)
            strings.append(s)
        return sidx[s]

    with zipfile.ZipFile(path, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr("[Content_Types].xml", CT + '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">'
                   '<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>'
                   '<Default Extension="xml" ContentType="application/xml"/>'
                   '<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>'
                   + "".join(f'<Override PartName="/xl/worksheets/sheet{i}.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>' for i in range(1, len(names) + 1))
                   + '<Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>'
                   '<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>')
        z.writestr("_rels/.rels", CT + '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
                   '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>')
        z.writestr("xl/workbook.xml", CT + '<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>'
                   + "".join(f'<sheet name="{escape(n)}" sheetId="{i}" r:id="rId{i}"{" state=\"hidden\"" if n in hidden_sheets else ""}/>' for i, n in enumerate(names, 1))
                   + "</sheets></workbook>")
        z.writestr("xl/_rels/workbook.xml.rels", CT + '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
                   + "".join(f'<Relationship Id="rId{i}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet{i}.xml"/>' for i in range(1, len(names) + 1))
                   + f'<Relationship Id="rId{len(names) + 1}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>'
                   f'<Relationship Id="rId{len(names) + 2}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>')
        z.writestr("xl/styles.xml", CT + '<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts><fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills><borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>'
                   '<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="2"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="14" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/></cellXfs><cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>')

        def cell_xml(sheet, ref, v):
            f = formulas.get((sheet, ref))
            fx = f"<f>{escape(f)}</f>" if f else ""
            s = ' s="1"' if (sheet, ref) in date_cells else ""
            if isinstance(v, tuple):
                return f'<c r="{ref}"{s}>{fx}<v>{v[1]}</v></c>'
            if isinstance(v, (int, float)):
                return f'<c r="{ref}"{s}>{fx}<v>{v}</v></c>'
            if v is None:
                return ""
            return f'<c r="{ref}" t="s">{fx}<v>{si(str(v))}</v></c>'

        for i, n in enumerate(names, 1):
            with z.open(f"xl/worksheets/sheet{i}.xml", "w") as w:
                w.write((CT + '<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>').encode())
                rows = list(sheets.get(n, []))
                for ri, row in enumerate(rows, 1):
                    hid = ' hidden="1"' if (n, ri) in hidden_rows else ""
                    w.write((f'<row r="{ri}"{hid}>' + "".join(cell_xml(n, f"{_col(ci)}{ri}", v) for ci, v in enumerate(row, 1)) + "</row>").encode())
                if stream_rows and stream_rows[0] == n:
                    ri = len(rows)
                    for row in stream_rows[1]:
                        ri += 1
                        w.write((f'<row r="{ri}">' + "".join(cell_xml(n, f"{_col(ci)}{ri}", v) for ci, v in enumerate(row, 1)) + "</row>").encode())
                w.write(b"</sheetData>")
                if merged.get(n):
                    w.write(("<mergeCells>" + "".join(f'<mergeCell ref="{m}"/>' for m in merged[n]) + "</mergeCells>").encode())
                w.write(b"</worksheet>")
        z.writestr("xl/sharedStrings.xml", CT + f'<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="{len(strings)}" uniqueCount="{len(strings)}">'
                   + "".join(f"<si><t xml:space=\"preserve\">{escape(s)}</t></si>" for s in strings) + "</sst>")


def make_docx(path, paragraphs, table=None, rtl=False):
    W = 'xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"'
    bidi = "<w:pPr><w:bidi/></w:pPr>" if rtl else ""
    body = "".join(f'<w:p>{bidi}<w:r><w:t xml:space="preserve">{escape(p)}</w:t></w:r></w:p>' for p in paragraphs)
    if table:
        body += "<w:tbl>" + "".join("<w:tr>" + "".join(f"<w:tc><w:p><w:r><w:t>{escape(str(c))}</w:t></w:r></w:p></w:tc>" for c in row) + "</w:tr>" for row in table) + "</w:tbl>"
    with zipfile.ZipFile(path, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr("[Content_Types].xml", CT + '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>')
        z.writestr("_rels/.rels", CT + '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>')
        z.writestr("word/document.xml", CT + f"<w:document {W}><w:body>{body}</w:body></w:document>")


def make_pptx(path, slides):
    """slides: list of (title, [bullets], table or None)."""
    P = ('xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" '
         'xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"')
    with zipfile.ZipFile(path, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr("[Content_Types].xml", CT + '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>'
                   + "".join(f'<Override PartName="/ppt/slides/slide{i}.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>' for i in range(1, len(slides) + 1)) + "</Types>")
        z.writestr("_rels/.rels", CT + '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/></Relationships>')
        z.writestr("ppt/presentation.xml", CT + f"<p:presentation {P}><p:sldIdLst>" + "".join(f'<p:sldId id="{255 + i}" r:id="rId{i}"/>' for i in range(1, len(slides) + 1)) + "</p:sldIdLst></p:presentation>")
        z.writestr("ppt/_rels/presentation.xml.rels", CT + '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
                   + "".join(f'<Relationship Id="rId{i}" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide{i}.xml"/>' for i in range(1, len(slides) + 1)) + "</Relationships>")
        for i, (title, bullets, table) in enumerate(slides, 1):
            sp = (f'<p:sp><p:nvSpPr><p:cNvPr id="2" name="Title 1"/></p:nvSpPr><p:txBody><a:p><a:r><a:t>{escape(title)}</a:t></a:r></a:p></p:txBody></p:sp>'
                  f'<p:sp><p:nvSpPr><p:cNvPr id="3" name="Content 2"/></p:nvSpPr><p:txBody>' + "".join(f"<a:p><a:r><a:t>{escape(b)}</a:t></a:r></a:p>" for b in bullets) + "</p:txBody></p:sp>")
            if table:
                sp += ('<p:graphicFrame><p:nvGraphicFramePr><p:cNvPr id="4" name="Table 3"/></p:nvGraphicFramePr><a:graphic><a:graphicData><a:tbl>'
                       + "".join("<a:tr>" + "".join(f"<a:tc><a:txBody><a:p><a:r><a:t>{escape(str(c))}</a:t></a:r></a:p></a:txBody></a:tc>" for c in row) + "</a:tr>" for row in table)
                       + "</a:tbl></a:graphicData></a:graphic></p:graphicFrame>")
            z.writestr(f"ppt/slides/slide{i}.xml", CT + f"<p:sld {P}><p:cSld><p:spTree>{sp}</p:spTree></p:cSld></p:sld>")


def make_pdf(path, pages):
    """pages: list of lists of ASCII lines. Uncompressed and Flate content alternate to test both."""
    import zlib
    objs = []
    kids = []
    font_id = 3
    objs.append(b"<< /Type /Catalog /Pages 2 0 R >>")
    objs.append(None)  # pages, filled later
    objs.append(b"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
    for pi, lines in enumerate(pages):
        ops = b"BT /F1 12 Tf 72 720 Td " + b" ".join(b"(" + l.replace("\\", "\\\\").replace("(", "\\(").replace(")", "\\)").encode("latin-1") + b") Tj 0 -16 Td" for l in lines) + b" ET"
        if pi % 2:
            data = zlib.compress(ops)
            content = b"<< /Length %d /Filter /FlateDecode >>\nstream\n" % len(data) + data + b"\nendstream"
        else:
            content = b"<< /Length %d >>\nstream\n" % len(ops) + ops + b"\nendstream"
        objs.append(content)
        cid = len(objs)
        objs.append(b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 %d 0 R >> >> /Contents %d 0 R >>" % (font_id, cid))
        kids.append(len(objs))
    objs[1] = b"<< /Type /Pages /Kids [" + b" ".join(b"%d 0 R" % k for k in kids) + b"] /Count %d >>" % len(kids)
    out = bytearray(b"%PDF-1.4\n")
    offs = []
    for i, o in enumerate(objs, 1):
        offs.append(len(out))
        out += b"%d 0 obj\n" % i + o + b"\nendobj\n"
    xref = len(out)
    out += b"xref\n0 %d\n0000000000 65535 f \n" % (len(objs) + 1) + b"".join(b"%010d 00000 n \n" % o for o in offs)
    out += b"trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n" % (len(objs) + 1, xref)
    with open(path, "wb") as f:
        f.write(out)
