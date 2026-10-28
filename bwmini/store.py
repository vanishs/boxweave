"""Local transactional state; remote I/O never runs in a transaction."""
import contextlib
import os
import sqlite3
from pathlib import Path
from .domain import Fault

SCHEMA = """
CREATE TABLE admin (id INTEGER PRIMARY KEY CHECK(id=1), password TEXT NOT NULL);
CREATE TABLE sessions (id TEXT PRIMARY KEY, csrf TEXT NOT NULL, expires REAL NOT NULL);
CREATE TABLE attempts (at REAL NOT NULL);
CREATE TABLE projects (id TEXT PRIMARY KEY, name TEXT NOT NULL, target TEXT NOT NULL,
 digest TEXT NOT NULL, workspace TEXT NOT NULL, revision INTEGER NOT NULL);
CREATE TABLE plans (id TEXT PRIMARY KEY, project TEXT NOT NULL REFERENCES projects(id),
 revision INTEGER NOT NULL, digest TEXT NOT NULL, expires REAL NOT NULL,
 actor TEXT NOT NULL, used INTEGER NOT NULL DEFAULT 0);
CREATE TABLE operations (id TEXT PRIMARY KEY, project TEXT NOT NULL REFERENCES projects(id),
 revision INTEGER NOT NULL, digest TEXT NOT NULL, actor TEXT NOT NULL,
 state TEXT NOT NULL, created REAL NOT NULL, deadline REAL NOT NULL,
 result TEXT, code TEXT);
CREATE TABLE audit (at REAL NOT NULL, action TEXT NOT NULL, target TEXT NOT NULL, code TEXT NOT NULL);
"""

class Store:
    def __init__(self, path):
        self.path = Path(path).absolute()
        # Never repair permissions or follow symlinks into an existing state tree.
        if self.path.is_symlink() or any(p.is_symlink() for p in self.path.parents):
            raise Fault("UNSAFE_STATE_PATH", 422)
        if self.path.exists() and (self.path.stat().st_nlink != 1 or self.path.stat().st_mode & 0o077):
            raise Fault("STATE_FILE_NOT_PRIVATE", 422)
        self.path.parent.mkdir(parents=True, mode=0o700, exist_ok=True)
        if os.name == "posix" and self.path.parent.stat().st_mode & 0o077:
            raise Fault("STATE_DIRECTORY_NOT_PRIVATE", 422)
        with self.db() as db:
            version = db.execute("PRAGMA user_version").fetchone()[0]
            if version not in (0, 1):
                raise Fault("SCHEMA_UNSUPPORTED", 422)
            if version == 0:
                if db.execute("SELECT count(*) FROM sqlite_master WHERE type='table'").fetchone()[0]:
                    raise Fault("SCHEMA_UNSUPPORTED", 422)
                db.executescript("BEGIN IMMEDIATE;" + SCHEMA + "PRAGMA user_version=1;COMMIT;")
        os.chmod(self.path, 0o600)

    @contextlib.contextmanager
    def db(self):
        db = sqlite3.connect(self.path, timeout=2)
        db.row_factory = sqlite3.Row
        db.execute("PRAGMA foreign_keys=ON")
        db.execute("PRAGMA synchronous=FULL")
        try:
            yield db
            db.commit()
        except sqlite3.OperationalError as error:
            db.rollback()
            raise Fault("STATE_UNAVAILABLE", 503) from error
        except BaseException:
            db.rollback()
            raise
        finally:
            db.close()

    def admin(self, hashed):
        with self.db() as db:
            db.execute("BEGIN IMMEDIATE")
            if db.execute("SELECT 1 FROM admin").fetchone():
                raise Fault("ADMIN_ALREADY_INITIALIZED")
            db.execute("INSERT INTO admin VALUES(1,?)", (hashed,))

    def project(self, ident):
        with self.db() as db:
            row = db.execute("SELECT * FROM projects WHERE id=?", (ident,)).fetchone()
        if row is None:
            raise Fault("NOT_FOUND", 404)
        return dict(row)

    def receipt(self, ident, actor="operator"):
        with self.db() as db:
            row = db.execute("SELECT * FROM operations WHERE id=? AND actor=?", (ident, actor)).fetchone()
        if row is None:
            raise Fault("NOT_FOUND", 404)
        return dict(row)
