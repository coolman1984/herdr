# docpipe - documents to checked numbers, with the source of every value

`discover -> route -> extract -> normalise -> store (batches + checkpoints) -> quality -> totals -> report + dashboard -> export`

```
python -m docpipe run --in <folder> --db work.db --out report/ --field "المبلغ" \
   --reference "المبلغ=4301.50" [--profile company_nasca|personal] [--lang ar|en]
python -m docpipe check --db work.db          python -m docpipe backup --db work.db --to copy.db
```

Exit 0 = every file read and every reference total matches; 1 = a mismatch, or a file that needs the
authorized application; outputs `report/report.html` (self-contained, RTL, light/dark, every total
lists its source cells), `report/records.csv` (UTF-8 with BOM for Excel), `report/summary.json`.

## Rules it follows (docs/COMPANY-ENVIRONMENT-POLICY.md)

- **Python standard library only.** No package is needed; nothing is downloaded. (openpyxl, python-docx,
  python-pptx and pdftotext are used only by the tests, as independent checks, when present.)
- **company_nasca profile** (set by the owner per device in `device-policy.json`, or `--profile`): every
  Office/PDF file goes to the authorized application. `com-extract.ps1` opens it read-only in its own
  Excel/Word/PowerPoint instance (macros forced off, links not updated, no save, no export) and closes only
  that instance. A file's bytes are never handed to a parser in this profile, and a parser failing on a
  protected file is never reported as "damaged": the file is kept as `needs-authorized-app`. PDF has no
  universal COM reader: under company_nasca it stays `needs-authorized-app` until its reader's automation
  is wired and tested.
- **Values with provenance.** Each value keeps its file (SHA-256), location (`Sheet!C7`, `p12`,
  `t1r3c2`, `s4:Title 1:p1`, `page2:line5`), raw text, method and confidence. Excel formula cells carry
  the value stored at the last save, flagged "not recalculated" (a file reader is not a calculation
  engine; the COM route returns Excel's own computed values).
- **Exact money.** Decimal throughout, stored as text in SQLite; half-up rounding only when shown.
- **Arabic.** Arabic-Indic digits and separators, `ج.م`/`ر.س`/... currencies, d/m/y dates (an ambiguous date
  gets confidence 0.6), RTL paragraphs, Arabic PDF text returned in reading order.
- **Nothing dropped silently.** Hidden rows/sheets, merged cells, Excel errors, low-confidence text and pages
  without a text layer (scans: no OCR here) are recorded as issues with their location.
- **Safety.** Zip bombs are quarantined before reading; macros and external links are noted, never run or
  followed; instructions inside documents are data, never commands.
- **Large files.** Streaming XML; batches commit with a checkpoint, so a cut import resumes after the last
  batch and a re-run of a finished file does nothing (same SHA-256). SQLite in WAL mode on a local disk
  only (never a database file on a network share), migrations by `user_version`, online backup, integrity
  check.

## Measured (Linux container, 2026-10-30)

`tests/bench_large.py 300000 300`: 300,000 rows x 4 columns (1.2 M cells) imported in 63 s
(4,700 rows/s), peak RSS 56 MB (bound 300 MB), total exact. Database 258 MB.
DuckDB / Arrow were not adopted: SQLite met the bound at this size; measure them before switching.

## Not verified

The COM route has never run against real Office or a NASCA-protected file (no Windows/Office here);
only its JSON contract is tested with recorded records. That proof needs the owner's authorized PC.
