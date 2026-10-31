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

GRAPH_SCHEMA = """
CREATE TABLE graphs (id TEXT PRIMARY KEY, revision INTEGER NOT NULL, document TEXT NOT NULL);
CREATE TABLE graph_copies (id TEXT PRIMARY KEY, graph TEXT NOT NULL REFERENCES graphs(id) ON DELETE CASCADE,
 revision INTEGER NOT NULL, at REAL NOT NULL, status TEXT NOT NULL CHECK(status='reported_success'));
"""

OBSERVATION_SCHEMA = """
CREATE TABLE operation_observations (id TEXT PRIMARY KEY, operation TEXT NOT NULL REFERENCES operations(id),
 actor TEXT NOT NULL, at REAL NOT NULL, target_status TEXT NOT NULL, code TEXT NOT NULL);
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
            db.execute("BEGIN IMMEDIATE")
            version = db.execute("PRAGMA user_version").fetchone()[0]
            if version not in (0, 1, 2, 3, 4, 5, 6, 7, 8):
                raise Fault("SCHEMA_UNSUPPORTED", 422)
            if version == 0:
                if db.execute("SELECT count(*) FROM sqlite_master WHERE type='table'").fetchone()[0]:
                    raise Fault("SCHEMA_UNSUPPORTED", 422)
                for statement in SCHEMA.split(";"):
                    if statement.strip():db.execute(statement)
            if version < 2:
                for statement in GRAPH_SCHEMA.split(";"):
                    if statement.strip():db.execute(statement)
            if version < 3:
                db.execute(OBSERVATION_SCHEMA)
            if version < 4:
                from .lifecycle import SCHEMA as lifecycle_schema
                for statement in lifecycle_schema.split(";"):
                    if statement.strip():db.execute(statement)
            if version < 5:
                from .models import SCHEMA as model_schema
                db.execute(model_schema)
            if version < 6:
                from .access import SCHEMA as access_schema
                for statement in access_schema.split(";"):
                    if statement.strip():db.execute(statement)
                db.execute("PRAGMA user_version=6")
            if version < 7:
                db.execute("CREATE TABLE project_unbindings (project TEXT PRIMARY KEY REFERENCES projects(id))")
                db.execute("PRAGMA user_version=7")
            if version < 8:
                from .jobs import SCHEMA as job_schema
                db.execute(job_schema)
                db.execute("PRAGMA user_version=8")
        os.chmod(self.path, 0o600)

    @contextlib.contextmanager
    def db(self):
        try:
            db = sqlite3.connect(self.path, timeout=2)
        except sqlite3.Error as error:
            raise Fault("STATE_UNAVAILABLE", 503) from error
        db.row_factory = sqlite3.Row
        db.execute("PRAGMA foreign_keys=ON")
        db.execute("PRAGMA synchronous=FULL")
        try:
            yield db
            db.commit()
        except sqlite3.Error as error:
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

    def reset_admin(self, hashed):
        with self.db() as db:
            db.execute("BEGIN IMMEDIATE")
            if not db.execute("SELECT 1 FROM admin").fetchone():
                raise Fault("ADMIN_NOT_INITIALIZED")
            db.execute("UPDATE admin SET password=? WHERE id=1",(hashed,))
            db.execute("DELETE FROM sessions")
            db.execute("DELETE FROM attempts")

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
