"""SQLite store: one writer, short transactions, schema migrations (PRAGMA user_version), idempotent
imports (a file is identified by its SHA-256), checkpoints for resuming a cut import, online backup and
integrity check. Decimal amounts are stored as TEXT so no float rounding enters the totals."""
import sqlite3
import time

MIGRATIONS = [
    # 1: base schema
    """
    CREATE TABLE files (
        id INTEGER PRIMARY KEY, sha256 TEXT NOT NULL UNIQUE, path TEXT NOT NULL, kind TEXT NOT NULL,
        size INTEGER NOT NULL, route TEXT NOT NULL, status TEXT NOT NULL, note TEXT,
        imported_at TEXT, records INTEGER DEFAULT 0
    );
    CREATE TABLE cells (
        id INTEGER PRIMARY KEY, file_id INTEGER NOT NULL REFERENCES files(id),
        loc TEXT NOT NULL,           -- Sheet!B7 | p12 | t1r3c2 | s4:Title 1:p1 | page2:line5
        sheet TEXT, row INTEGER, col INTEGER,
        raw TEXT, num TEXT, date TEXT, text TEXT, currency TEXT,
        kind TEXT, confidence REAL NOT NULL, method TEXT NOT NULL, flags TEXT,
        UNIQUE(file_id, loc)
    );
    CREATE INDEX cells_file_row ON cells(file_id, sheet, row);
    CREATE TABLE checkpoints (file_id INTEGER PRIMARY KEY, last_loc_seq INTEGER NOT NULL, at TEXT NOT NULL);
    CREATE TABLE issues (id INTEGER PRIMARY KEY, file_id INTEGER, loc TEXT, level TEXT NOT NULL, note TEXT NOT NULL);
    CREATE TABLE runs (id INTEGER PRIMARY KEY, started TEXT, finished TEXT, files INTEGER, note TEXT);
    """,
    # 2: records table for row-shaped data (header-mapped), used by reports
    """
    CREATE TABLE records (
        id INTEGER PRIMARY KEY, file_id INTEGER NOT NULL REFERENCES files(id), sheet TEXT, row INTEGER,
        field TEXT NOT NULL, num TEXT, text TEXT, date TEXT, currency TEXT, loc TEXT NOT NULL,
        UNIQUE(file_id, sheet, row, field)
    );
    CREATE INDEX records_field ON records(field);
    """,
]


def now():
    return time.strftime("%Y-%m-%dT%H:%M:%S")


def connect(path):
    db = sqlite3.connect(path, timeout=30, isolation_level=None)
    db.execute("PRAGMA journal_mode=WAL")  # local disk only: never a shared network file (see README)
    db.execute("PRAGMA synchronous=FULL")
    db.execute("PRAGMA foreign_keys=ON")
    migrate(db)
    return db


def migrate(db):
    v = db.execute("PRAGMA user_version").fetchone()[0]
    for i, sql in enumerate(MIGRATIONS[v:], v + 1):
        db.execute("BEGIN IMMEDIATE")
        try:
            for stmt in [s for s in sql.split(";") if s.strip()]:
                db.execute(stmt)
            db.execute(f"PRAGMA user_version={i}")
            db.execute("COMMIT")
        except Exception:
            db.execute("ROLLBACK")
            raise


def file_row(db, sha):
    return db.execute("SELECT id, status FROM files WHERE sha256=?", (sha,)).fetchone()


def add_file(db, sha, path, kind, size, route, status, note=""):
    db.execute("INSERT OR IGNORE INTO files(sha256,path,kind,size,route,status,note) VALUES(?,?,?,?,?,?,?)",
               (sha, path, kind, size, route, status, note))
    return db.execute("SELECT id FROM files WHERE sha256=?", (sha,)).fetchone()[0]


def checkpoint(db, file_id):
    r = db.execute("SELECT last_loc_seq FROM checkpoints WHERE file_id=?", (file_id,)).fetchone()
    return r[0] if r else 0


def write_batch(db, file_id, seq_end, cells, records, issues):
    """One transaction per batch: cells, records, issues and the checkpoint move together, so a cut
    import resumes exactly after the last committed batch and nothing is inserted twice."""
    db.execute("BEGIN IMMEDIATE")
    try:
        db.executemany("""INSERT OR IGNORE INTO cells(file_id,loc,sheet,row,col,raw,num,date,text,currency,kind,confidence,method,flags)
                          VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)""", cells)
        db.executemany("""INSERT OR IGNORE INTO records(file_id,sheet,row,field,num,text,date,currency,loc)
                          VALUES(?,?,?,?,?,?,?,?,?)""", records)
        db.executemany("INSERT INTO issues(file_id,loc,level,note) VALUES(?,?,?,?)", issues)
        db.execute("INSERT INTO checkpoints(file_id,last_loc_seq,at) VALUES(?,?,?) ON CONFLICT(file_id) DO UPDATE SET last_loc_seq=excluded.last_loc_seq, at=excluded.at",
                   (file_id, seq_end, now()))
        db.execute("COMMIT")
    except Exception:
        db.execute("ROLLBACK")
        raise


def finish_file(db, file_id, status, note=""):
    n = db.execute("SELECT COUNT(*) FROM cells WHERE file_id=?", (file_id,)).fetchone()[0]
    db.execute("UPDATE files SET status=?, note=?, imported_at=?, records=? WHERE id=?", (status, note, now(), n, file_id))


def backup(db, path):
    dst = sqlite3.connect(path)
    with dst:
        db.backup(dst)
    dst.close()


def integrity(db):
    return db.execute("PRAGMA integrity_check").fetchone()[0]
