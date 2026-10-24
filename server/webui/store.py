"""SQLite persistence, optimistic graph revisions and durable operation receipts."""

import json
import sqlite3
import time
import uuid
from contextlib import contextmanager
from pathlib import Path

SCHEMA = 1


class Conflict(Exception):
    pass


class Store:
    def __init__(self, path):
        self.path = str(path)
        Path(path).parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        with self.db() as db:
            version = db.execute("PRAGMA user_version").fetchone()[0]
            if version > SCHEMA:
                raise RuntimeError(
                    "Database schema is newer than this application; restore matching software"
                )
            if version == 0:
                db.executescript(
                    """
                CREATE TABLE admin (id INTEGER PRIMARY KEY CHECK(id=1), hash TEXT NOT NULL);
                CREATE TABLE sessions (id TEXT PRIMARY KEY, csrf TEXT NOT NULL, expires REAL NOT NULL);
                CREATE TABLE attempts (address TEXT NOT NULL, at REAL NOT NULL);
                CREATE TABLE graphs (id TEXT PRIMARY KEY, version INTEGER NOT NULL, data TEXT NOT NULL, updated REAL NOT NULL);
                CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT);
                CREATE TABLE plans (id TEXT PRIMARY KEY, data TEXT NOT NULL, expires REAL NOT NULL, used INTEGER NOT NULL DEFAULT 0);
                CREATE TABLE jobs (id TEXT PRIMARY KEY, data TEXT NOT NULL);
                CREATE TABLE audit (at REAL NOT NULL, action TEXT NOT NULL, target TEXT NOT NULL, result TEXT NOT NULL);
                PRAGMA user_version=1;
                """
                )
        Path(path).chmod(0o600)

    @contextmanager
    def db(self):
        db = sqlite3.connect(self.path, timeout=10)
        db.row_factory = sqlite3.Row
        db.execute("PRAGMA foreign_keys=ON")
        try:
            yield db
            db.commit()
        except BaseException:
            db.rollback()
            raise
        finally:
            db.close()

    def graphs(self):
        with self.db() as db:
            return [
                dict(id=r["id"], version=r["version"], **json.loads(r["data"]))
                for r in db.execute("SELECT * FROM graphs ORDER BY updated DESC,id")
            ]

    def graph(self, gid):
        with self.db() as db:
            r = db.execute("SELECT * FROM graphs WHERE id=?", (gid,)).fetchone()
            if not r:
                raise KeyError(gid)
            return dict(id=gid, version=r["version"], **json.loads(r["data"]))

    def save(self, gid, version, data):
        with self.db() as db:
            if version == 0:
                try:
                    db.execute(
                        "INSERT INTO graphs VALUES (?,?,?,?)",
                        (gid, 1, json.dumps(data), time.time()),
                    )
                except sqlite3.IntegrityError:
                    raise Conflict()
            else:
                row = db.execute(
                    "UPDATE graphs SET version=version+1,data=?,updated=? WHERE id=? AND version=?",
                    (json.dumps(data), time.time(), gid, version),
                )
                if row.rowcount != 1:
                    raise Conflict()
        return self.graph(gid)

    def audit(self, action, target, result):
        with self.db() as db:
            db.execute(
                "INSERT INTO audit VALUES (?,?,?,?)",
                (time.time(), action, target, result),
            )

    def setting(self, key, value=None):
        with self.db() as db:
            if value is not None:
                db.execute("INSERT OR REPLACE INTO settings VALUES (?,?)", (key, value))
            r = db.execute("SELECT value FROM settings WHERE key=?", (key,)).fetchone()
            return r[0] if r else None

    def job(self, jid, data=None):
        with self.db() as db:
            if data is not None:
                db.execute(
                    "INSERT OR REPLACE INTO jobs VALUES (?,?)", (jid, json.dumps(data))
                )
            r = db.execute("SELECT data FROM jobs WHERE id=?", (jid,)).fetchone()
            if not r:
                raise KeyError(jid)
            return json.loads(r[0])
