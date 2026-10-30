"""Builds a demo database for the web tests: three files (xlsx with Arabic + a big sheet, docx, pptx)."""
import os
import sys
HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.dirname(os.path.dirname(HERE)))
sys.path.insert(0, os.path.dirname(HERE))
import fixtures as F  # noqa: E402
from docpipe import pipeline, store  # noqa: E402

out = sys.argv[1]
rows = int(sys.argv[2]) if len(sys.argv) > 2 else 20000
os.makedirs(os.path.join(out, "in"), exist_ok=True)
F.make_xlsx(os.path.join(out, "in", "sales.xlsx"), {"مبيعات": [["الفرع", "المبلغ"], ["القاهرة", 1250.5], ["الإسكندرية", "١٬٢٠٠٫٢٥"], ["الجيزة", "(300)"]]})
F.make_xlsx(os.path.join(out, "in", "ledger.xlsx"), {"ledger": [["client", "amount"]]},
            stream_rows=("ledger", ([f"عميل {i % 97}", f"{i % 1000}.50"] for i in range(rows))))
F.make_docx(os.path.join(out, "in", "memo.docx"), ["مذكرة الربع الثالث", "إجمالي المبيعات 2,150.75 جنيه"], rtl=True)
F.make_pptx(os.path.join(out, "in", "deck.pptx"), [("الملخص", ["نمو 12%"], None)])
db = store.connect(os.path.join(out, "demo.db"))
for n in sorted(os.listdir(os.path.join(out, "in"))):
    print(pipeline.import_file(db, os.path.join(out, "in", n), batch=5000))
store.connect(os.path.join(out, "empty.db")).close()
