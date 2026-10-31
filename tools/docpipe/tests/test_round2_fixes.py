"""Round-2 fixes beyond the seven independent regressions (same root causes, other paths)."""
import builtins
import os
import sqlite3
import sys
import tempfile
import threading
import unittest
from decimal import Decimal
from pathlib import Path
from unittest import mock

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.dirname(HERE))
sys.path.insert(0, HERE)
import fixtures as F  # noqa: E402
from docpipe import ooxml, pipeline, sniff, store  # noqa: E402


class Base(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.db = store.connect(str(self.root / "w.db"))

    def tearDown(self):
        self.db.close()
        self.tmp.cleanup()

    def write(self, name, data):
        p = self.root / name
        p.write_bytes(data if isinstance(data, bytes) else data.encode("utf-8"))
        return str(p)


class CompanyPolicy(Base):
    def test_no_company_file_of_any_type_is_read_before_policy(self):
        names = ["a.txt", "b.tsv", "c.xlsb", "d.pdf", "e.bin", "noext", "f.csv", "g.docx"]
        paths = [self.write(n, "opaque") for n in names]
        real_open = builtins.open

        def guarded(file, *a, **k):
            if str(file) in paths:
                raise PermissionError("raw read of a protected file")
            return real_open(file, *a, **k)
        with mock.patch("builtins.open", side_effect=guarded), \
             mock.patch.object(pipeline, "com_extract", return_value=([], None)) as com:
            st = {os.path.basename(p): pipeline.import_file(self.db, p, profile="company_nasca")[1] for p in paths}
        self.assertEqual(st["d.pdf"], "needs-authorized-app")
        self.assertEqual(st["e.bin"], "needs-authorized-app")
        self.assertEqual(st["noext"], "needs-authorized-app")
        for n in ("a.txt", "b.tsv", "c.xlsb", "f.csv", "g.docx"):
            self.assertEqual(st[n], "done", n)
        self.assertEqual(com.call_count, 5)  # every readable company file went to the authorized app
        notes = dict(self.db.execute("SELECT path, note FROM files").fetchall())
        self.assertIn("no authorized PDF reader", notes[paths[3]])

    def test_identity_includes_profile_and_parser_version(self):
        p = self.write("x.csv", "Amount\n5\n")
        pipeline.import_file(self.db, p)
        with mock.patch.object(pipeline, "com_extract", return_value=([], None)):
            _, st = pipeline.import_file(self.db, p, profile="company_nasca")
        self.assertNotEqual(st, "already-imported")  # a personal import is not reused under the company policy
        with mock.patch.object(pipeline, "PARSER_VERSION", "999"):
            _, st = pipeline.import_file(self.db, p)
        self.assertNotEqual(st, "already-imported")  # nor after a parser change

    def test_com_output_is_removed_even_when_the_worker_fails(self):
        wd = self.root / "work"
        seen = {}

        def fake_run(argv, **kw):
            out = argv[argv.index("-Out") + 1]
            Path(out).write_text('{"loc":"S!A1","raw":"secret"}\n')
            Path(out + ".pid").write_text("123")
            seen["out"] = out
            return mock.Mock(returncode=7, stderr="boom", stdout="")
        with mock.patch.object(pipeline.platform, "system", return_value="Windows"), \
             mock.patch.object(pipeline, "_which", return_value="pwsh"), \
             mock.patch.object(pipeline.subprocess, "run", side_effect=fake_run):
            recs, err = pipeline.com_extract("x.xlsx", workdir=str(wd))
        self.assertIsNone(recs)
        self.assertIn("exit 7", err)
        self.assertFalse(os.path.exists(seen["out"]) or os.path.exists(seen["out"] + ".pid"), "extracted company content left on disk")

    def test_blocked_execution_policy_is_reported_not_bypassed(self):
        captured = {}

        def fake_run(argv, **kw):
            captured["argv"] = argv
            return mock.Mock(returncode=1, stderr="File cannot be loaded because running scripts is disabled on this system.", stdout="")
        with mock.patch.object(pipeline.platform, "system", return_value="Windows"), \
             mock.patch.object(pipeline, "_which", return_value="powershell"), \
             mock.patch.object(pipeline.subprocess, "run", side_effect=fake_run):
            recs, err = pipeline.com_extract("x.xlsx", workdir=str(self.root))
        self.assertNotIn("Bypass", " ".join(captured["argv"]))
        self.assertIn("execution policy", err)


class Integrity(Base):
    def test_headers_duplicate_missing_multiline_reconciled(self):
        p = self.write("h.csv", 'Amount,Amount,"Due\nDate",\n1,2,3,4\n')
        fid, st = pipeline.import_file(self.db, p)
        fields = sorted(r[0] for r in self.db.execute("SELECT field FROM records WHERE file_id=?", (fid,)))
        self.assertEqual(fields, ["Amount", "Amount#2", "Due Date", "column 4"])
        notes = " | ".join(r[0] for r in self.db.execute("SELECT note FROM issues WHERE file_id=?", (fid,)))
        for want in ("duplicate header", "line breaks", "no header"):
            self.assertIn(want, notes)
        self.assertIn("4 under headers, 4 records", self.db.execute("SELECT note FROM files WHERE id=?", (fid,)).fetchone()[0])

    def test_source_change_during_import_keeps_nothing(self):
        p = self.write("s.csv", "Amount\n" + "\n".join(str(i) for i in range(5000)) + "\n")
        real = pipeline._csv_records

        def changing(path):
            for i, rec in enumerate(real(path)):
                if i == 10:
                    with open(path, "a") as f:
                        f.write("999\n")
                yield rec
        with mock.patch.object(pipeline, "_csv_records", changing):
            fid, st = pipeline.import_file(self.db, p, batch=100)
        self.assertEqual(st, "source-changed")
        self.assertEqual(self.db.execute("SELECT COUNT(*) FROM cells WHERE file_id=?", (fid,)).fetchone()[0], 0)

    def test_disk_full_is_a_clear_status_and_the_database_stays_sound(self):
        p = self.write("big.csv", "Amount\n" + "\n".join(str(i) * 50 for i in range(20000)) + "\n")
        pages = self.db.execute("PRAGMA page_count").fetchone()[0]
        self.db.execute(f"PRAGMA max_page_count={pages + 20}")
        fid, st = pipeline.import_file(self.db, p, batch=500)
        self.assertEqual(st, "disk-full")
        self.db.execute("PRAGMA max_page_count=1073741823")
        self.assertEqual(store.integrity(self.db), "ok")
        ck = store.checkpoint(self.db, fid)
        n = self.db.execute("SELECT COUNT(*) FROM cells WHERE file_id=?", (fid,)).fetchone()[0]
        self.assertEqual(n, ck, "committed cells and the checkpoint disagree after a full disk")
        _, st = pipeline.import_file(self.db, p, batch=500)  # space is back: it resumes
        self.assertEqual(st, "done")

    def test_second_importer_waits_for_the_lease(self):
        p = self.write("l.csv", "Amount\n1\n")
        ident = pipeline._identity(p, "personal")[0]
        fid = store.add_file(self.db, ident, p, "csv", 1, "native", "importing")
        self.assertTrue(store.take_lease(self.db, fid, "other-process"))
        self.assertEqual(pipeline.import_file(self.db, p)[1], "busy")
        store.release_lease(self.db, fid, "other-process")
        self.assertEqual(pipeline.import_file(self.db, p)[1], "done")

    def test_checkpoint_never_moves_back(self):
        p = self.write("c.csv", "Amount\n1\n")
        fid = store.add_file(self.db, "x", p, "csv", 1, "native", "importing")
        store.write_batch(self.db, fid, 50, [], [], [])
        store.write_batch(self.db, fid, 10, [], [], [])  # a stale importer
        self.assertEqual(store.checkpoint(self.db, fid), 50)

    def test_concurrent_migration_applies_once(self):
        path = str(self.root / "m.db")
        errors = []

        def open_it():
            try:
                store.connect(path).close()
            except Exception as e:  # noqa: BLE001
                errors.append(e)
        ts = [threading.Thread(target=open_it) for _ in range(6)]
        [t.start() for t in ts]
        [t.join() for t in ts]
        self.assertEqual(errors, [])
        db = sqlite3.connect(path)
        self.assertEqual(db.execute("PRAGMA user_version").fetchone()[0], len(store.MIGRATIONS))
        db.close()

    def test_network_database_is_refused(self):
        with self.assertRaises(ValueError):
            store.connect("\\\\server\\share\\w.db")


class Values(Base):
    def test_date_systems_leap_boundary_and_time(self):
        cases = [((1, True), "1904-01-02"), ((59, False), "1900-02-28"), ((61, False), "1900-03-01"),
                 ((46325.5, False), "2026-10-30T12:00:00"), ((0, True), "1904-01-01")]
        for (raw, d1904), want in cases:
            got = pipeline._normalise_com({"raw": raw, "kind": "n", "is_date": True, "date1904": d1904})[1]
            self.assertEqual(got, want, (raw, d1904))
            rec = dict(raw=str(raw), kind="n", formula=False, row_hidden=False, sheet_hidden=False, merged=None,
                       is_date=True, date1904=d1904)
            self.assertEqual(pipeline._cell_from_xlsx(rec)[1], want, (raw, d1904))

    def test_workbook_1904_flag_is_read(self):
        p = str(self.root / "w1904.xlsx")
        F.make_xlsx(p, {"S": [["d"], [("n", "1")]]}, date_cells={("S", "A2")})
        import zipfile
        tmp = p + ".tmp"
        with zipfile.ZipFile(p) as zin, zipfile.ZipFile(tmp, "w") as zout:
            for i in zin.infolist():
                data = zin.read(i.filename)
                if i.filename == "xl/workbook.xml":
                    data = data.replace(b"<sheets>", b'<workbookPr date1904="1"/><sheets>')
                zout.writestr(i, data)
        os.replace(tmp, p)
        recs = [r for r in ooxml.xlsx_records(p) if r["cell"] == "A2"]
        self.assertTrue(recs[0]["date1904"])
        self.assertEqual(pipeline._cell_from_xlsx(recs[0])[1], "1904-01-02")

    def test_unknown_cell_kind_is_kept(self):
        rec = dict(raw="x", kind="weird", formula=False, row_hidden=False, sheet_hidden=False, merged=None, is_date=False, date1904=False)
        num, date, text, *_ , flags = pipeline._cell_from_xlsx(rec)
        self.assertEqual(text, "x")
        self.assertTrue(any("unrecognised" in f for f in flags))

    def test_inexact_total_is_an_error_not_a_rounding(self):
        p = self.write("t.csv", "Amount\n" + "9" * 250 + "\n0.1\n")
        pipeline.import_file(self.db, p)
        with self.assertRaises(Exception):
            pipeline.totals(self.db, "Amount")

    def test_huge_merge_is_not_expanded_and_shared_strings_are_bounded(self):
        p = str(self.root / "m.xlsx")
        F.make_xlsx(p, {"S": [["a"], [1]]}, merged={"S": ["A1:XFD1048576"]})
        recs = list(ooxml.xlsx_records(p))
        self.assertIn("huge merge", recs[0]["merged"])
        with mock.patch.object(ooxml, "MAX_SHARED_STRING_CHARS", 0):
            with self.assertRaises(ooxml.LimitExceeded):
                list(ooxml.xlsx_records(p))


if __name__ == "__main__":
    unittest.main()


class Exports(Base):
    def test_csv_export_neutralises_formulas_and_streams(self):
        from docpipe import export, report
        p = self.write("f.csv", 'Label,Amount\n"=HYPERLINK(""http://x"")",5\n@SUM(A1),-3\n')
        pipeline.import_file(self.db, p)
        out = self.root / "r.csv"
        report.build(self.db, str(self.root / "r.html"), str(out), "t", ["Amount"])
        body = out.read_text(encoding="utf-8-sig")
        self.assertIn("'=HYPERLINK", body)
        self.assertIn("'@SUM", body)
        self.assertIn(",-3,", body)  # numbers stay numbers
        chunks = list(export.stream_csv(self.db, chunk=64))
        self.assertGreater(len(chunks), 1)
        self.assertEqual(b"".join(chunks).decode("utf-8").lstrip("﻿").replace("\r\n", "\n"), body.replace("\r\n", "\n"))

    def test_company_run_without_approval_exports_no_values(self):
        from docpipe import __main__ as cli
        src = self.root / "in"
        src.mkdir()
        (src / "a.csv").write_text("Amount\n5\n")
        out = self.root / "out"
        with mock.patch.object(pipeline, "com_extract", return_value=([
                {"loc": "S!A1", "sheet": "S", "row": 1, "col": 1, "raw": "Amount", "kind": "s"},
                {"loc": "S!A2", "sheet": "S", "row": 2, "col": 1, "raw": 5, "kind": "n"}], None)):
            code = cli.main(["run", "--in", str(src), "--db", str(self.root / "c.db"), "--out", str(out),
                             "--profile", "company_nasca", "--field", "Amount", "--reference", "Amount=5"])
        self.assertEqual(code, 0)
        self.assertFalse((out / "report.html").exists() or (out / "records.csv").exists())
        summary = (out / "summary.json").read_text()
        self.assertIn("withheld", summary)
        self.assertNotIn('"5"', summary.replace('"Amount=5"', ""))

    def test_server_refuses_network_without_flag(self):
        from docpipe import web
        with self.assertRaises(SystemExit):
            web.serve(str(self.root / "w.db"), host="0.0.0.0", port=0)


class UnknownProfile(Base):
    def test_unknown_device_profile_is_strict(self):
        p = self.write("x.csv", "A\n1\n")
        real_open = builtins.open

        def guarded(file, *a, **k):
            if str(file) == p:
                raise PermissionError("raw read")
            return real_open(file, *a, **k)
        with mock.patch("builtins.open", side_effect=guarded), mock.patch.object(pipeline, "com_extract", return_value=(None, "no COM")):
            _, st = pipeline.import_file(self.db, p, profile="unknown")
        self.assertEqual(st, "needs-authorized-app")


class Subtotals(Base):
    def test_formula_total_row_is_not_added_again(self):
        p = str(self.root / "s.xlsx")
        F.make_xlsx(p, {"S": [["Amount"], [100], [50], [("n", "150")]]}, formulas={("S", "A4"): "SUM(A2:A3)"})
        pipeline.import_file(self.db, p)
        t = pipeline.totals(self.db, "Amount")["-"]
        self.assertEqual((t["total"], t["count"], t["formulas"]), (Decimal("150"), 2, 1))
