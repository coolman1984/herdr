"""Independent regression specifications, pinned initially to cd7b35e.

Failures are evidence of missing behavior, not permission to weaken expectations.
Company routing tests simulate a denied raw file read; they are NOT a NASCA test.
"""
import builtins
from decimal import Decimal
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from docpipe import pipeline, sniff, store


class IndependentRound2(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.db = store.connect(str(self.root / "data.db"))

    def tearDown(self):
        self.db.close()
        self.tmp.cleanup()

    def csv(self, text):
        path = self.root / "input.csv"
        path.write_text(text, encoding="utf-8", newline="")
        return path

    def test_duplicate_headers_never_silently_discard_values(self):
        fid, status = pipeline.import_file(self.db, str(self.csv("Amount,Amount\n1,2\n")))
        mapped = self.db.execute("SELECT num FROM records WHERE file_id=? AND row=2", (fid,)).fetchall()
        issues = self.db.execute("SELECT note FROM issues WHERE file_id=?", (fid,)).fetchall()
        self.assertTrue(len(mapped) == 2 or status != "done" or issues,
                        "two source values became one report value, marked done with no issue")

    def test_quoted_csv_newline_is_preserved(self):
        pipeline.import_file(self.db, str(self.csv('Label,Amount\n"hello\nworld",2\n')))
        value = self.db.execute("SELECT raw FROM cells WHERE loc='csv!R2C1'").fetchone()[0]
        self.assertEqual(value, "hello\nworld")

    def test_large_decimal_total_keeps_cents(self):
        raw = "123456789012345678901234567890.01"
        pipeline.import_file(self.db, str(self.csv("Amount\n" + raw + "\n")))
        self.assertEqual(pipeline.totals(self.db, "Amount")["-"]["total"], Decimal(raw))

    def test_company_csv_requires_authorized_application(self):
        route, why = sniff.route(str(self.csv("A\n1\n")), "csv", "company_nasca")
        self.assertIn(route, ("com", "app-pdf", "skip"), why)

    def test_company_authorized_route_does_not_open_source_bytes(self):
        path = self.root / "protected.xlsx"
        path.write_bytes(b"opaque protected document")
        real_open = builtins.open
        def guarded_open(file, mode="r", *args, **kwargs):
            if str(file) == str(path):
                raise PermissionError("raw access denied; authorized application may open")
            return real_open(file, mode, *args, **kwargs)
        with patch("builtins.open", side_effect=guarded_open), patch.object(
                pipeline, "com_extract", return_value=([], None)) as com:
            try:
                _, status = pipeline.import_file(self.db, str(path), profile="company_nasca")
            except PermissionError as exc:
                self.fail("protected source was opened before authorized app routing: " + str(exc))
            self.assertTrue(com.called, "authorized app never received the file")
            self.assertIn(status, ("done", "partial"))

    def test_com_1904_workbook_date_system_is_respected(self):
        values = pipeline._normalise_com({"raw": 1, "kind": "n", "is_date": True, "date1904": True})
        self.assertEqual(values[1], "1904-01-02")

    def test_iso_date_cell_is_not_accepted_as_empty(self):
        row = dict(raw="2026-10-30", kind="d", formula=False, row_hidden=False,
                   sheet_hidden=False, merged=None, is_date=True, date1904=False)
        values = pipeline._cell_from_xlsx(row)
        self.assertEqual(values[1], "2026-10-30")


if __name__ == "__main__":
    unittest.main()
