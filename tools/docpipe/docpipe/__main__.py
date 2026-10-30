"""python -m docpipe run --in DIR --db FILE --out DIR [--profile company_nasca|personal] [--field Amount]
                         [--reference "Amount (EGP)=1234.50"] [--title T] [--lang ar|en] [--batch N]
python -m docpipe check --db FILE        integrity check
python -m docpipe backup --db FILE --to FILE
Exit 0 = all files imported and every reference total matches; 1 = a mismatch or a file needing the
authorized application; 2 = usage."""
import argparse
import json
import os
import sys

from . import pipeline, report, store


def main(argv=None):
    ap = argparse.ArgumentParser(prog="docpipe")
    sub = ap.add_subparsers(dest="cmd", required=True)
    r = sub.add_parser("run")
    r.add_argument("--in", dest="src", required=True)
    r.add_argument("--db", required=True)
    r.add_argument("--out", required=True)
    r.add_argument("--profile", default=os.environ.get("HERDR_DEVICE_PROFILE", "personal"))
    r.add_argument("--field", action="append", default=[])
    r.add_argument("--reference", action="append", default=[])
    r.add_argument("--title", default="docpipe report")
    r.add_argument("--lang", default="ar")
    r.add_argument("--batch", type=int, default=2000)
    c = sub.add_parser("check")
    c.add_argument("--db", required=True)
    w = sub.add_parser("serve")
    w.add_argument("--db", required=True)
    w.add_argument("--host", default="127.0.0.1")
    w.add_argument("--port", type=int, default=8765)
    b = sub.add_parser("backup")
    b.add_argument("--db", required=True)
    b.add_argument("--to", required=True)
    a = ap.parse_args(argv)
    if a.cmd == "check":
        res = store.integrity(store.connect(a.db))
        print(res)
        return 0 if res == "ok" else 1
    if a.cmd == "serve":
        from . import web
        web.serve(a.db, a.host, a.port)
        return 0
    if a.cmd == "backup":
        store.backup(store.connect(a.db), a.to)
        print("backup", a.to, store.integrity(store.connect(a.to)))
        return 0
    os.makedirs(a.out, exist_ok=True)
    db = store.connect(a.db)
    paths = []
    for root, _, names in os.walk(a.src):
        for n in sorted(names):
            if not n.startswith(("~$", ".")):
                paths.append(os.path.join(root, n))
    statuses = {}
    for p in sorted(paths):
        fid, st = pipeline.import_file(db, p, a.profile, a.batch)
        statuses[p] = st
        print(f"{st:22s} {p}")
    refs = {}
    for ref in a.reference:
        k, v = ref.rsplit("=", 1)
        refs[k.strip()] = v.strip()
    res = report.build(db, os.path.join(a.out, "report.html"), os.path.join(a.out, "records.csv"), a.title, a.field, refs, a.lang)
    with open(os.path.join(a.out, "summary.json"), "w", encoding="utf-8") as f:
        json.dump({"files": statuses, **res}, f, ensure_ascii=False, indent=1, default=str)
    print(json.dumps(res, ensure_ascii=False, default=str))
    bad = [k for k, *_, ok in res["checks"] if not ok]
    pending = [p for p, s in statuses.items() if s in ("needs-authorized-app", "quarantined")]
    if bad:
        print("REFERENCE MISMATCH:", bad, file=sys.stderr)
    if pending:
        print("NOT READ (authorized application needed or quarantined):", pending, file=sys.stderr)
    return 1 if bad or pending else 0


if __name__ == "__main__":
    sys.exit(main())
