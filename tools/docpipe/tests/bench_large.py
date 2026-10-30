"""Large-file run under a declared memory bound: N rows x 4 columns streamed into an xlsx, imported in
batches, totals checked exactly, peak memory and time reported. Exit 1 if over the bound.
   python3 tests/bench_large.py [rows] [max_rss_mb]"""
import os
import resource
import shutil
import sys
import tempfile
import time
from decimal import Decimal

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.dirname(HERE))
sys.path.insert(0, HERE)
import fixtures as F  # noqa: E402
from docpipe import pipeline, store  # noqa: E402

rows = int(sys.argv[1]) if len(sys.argv) > 1 else 300000
bound = int(sys.argv[2]) if len(sys.argv) > 2 else 300
d = tempfile.mkdtemp(prefix="docpipe-big-")
p = os.path.join(d, "big.xlsx")
t0 = time.time()
gen = ([f"عميل {i % 997}", f"2026-{1 + i % 12:02d}-{1 + i % 28:02d}", f"{i % 10000}.{i % 100:02d}", i % 7] for i in range(rows))
F.make_xlsx(p, {"data": [["client", "date", "amount", "qty"]]}, stream_rows=("data", gen))
t1 = time.time()
base = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss // 1024
db = store.connect(os.path.join(d, "big.db"))
_, st = pipeline.import_file(db, p, batch=5000)
t2 = time.time()
peak = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss // 1024
tot = pipeline.totals(db, "amount")["-"]
want = sum(Decimal(f"{i % 10000}.{i % 100:02d}") for i in range(rows))
ok = st == "done" and tot["total"] == want and tot["count"] == rows and peak <= bound
print(f"rows {rows}: write {t1 - t0:.1f}s, import {t2 - t1:.1f}s ({rows / max(t2 - t1, 1e-9):.0f} rows/s), "
      f"file {os.path.getsize(p) >> 20} MB, db {os.path.getsize(os.path.join(d, 'big.db')) >> 20} MB, "
      f"peak RSS {peak} MB (start {base} MB, bound {bound} MB), total {tot['total']} == {want}: {tot['total'] == want}")
shutil.rmtree(d)
sys.exit(0 if ok else 1)
