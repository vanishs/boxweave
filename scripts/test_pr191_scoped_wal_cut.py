#!/usr/bin/env python3
"""Exercise the read-only commit marker selector on actual private SQLite WALs.

No installed controller, systemd, native daemon or process-cut arm is used.
"""
import ctypes
import os
from pathlib import Path
import sqlite3
import subprocess
import tempfile
import unittest

from pr191_scoped_wal_cut import cold_ledger_rows


class ScopedWALMarkerTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.build = tempfile.TemporaryDirectory(prefix="p1b-wal-parser-")
        cls.addClassCleanup(cls.build.cleanup)
        lib = Path(cls.build.name) / "selector.so"
        subprocess.run(["cc", "-shared", "-fPIC", "-O2", "-std=gnu11", "-Wall", "-Wextra", "-Werror",
                        "-o", str(lib), str(Path(__file__).with_name("pr191_scoped_wal_cut.c")), "-ldl"],
                       check=True, timeout=30)
        cls.selector = ctypes.CDLL(str(lib)).p1b_wal_has_marker
        cls.selector.argtypes = [ctypes.c_int, ctypes.c_uint32, ctypes.c_char_p]
        cls.selector.restype = ctypes.c_int

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="p1b-wal-db-")
        self.addCleanup(self.temp.cleanup)
        self.db_path = Path(self.temp.name) / "ledger.sqlite"
        self.wal = Path(str(self.db_path) + "-wal")
        self.db = sqlite3.connect(self.db_path)
        self.addCleanup(self.db.close)
        self.assertEqual(self.db.execute("PRAGMA journal_mode=WAL").fetchone(), ("wal",))
        self.db.execute("PRAGMA synchronous=FULL")
        self.db.execute("PRAGMA wal_autocheckpoint=0")
        self.db.execute("CREATE TABLE marker (nonce TEXT, original TEXT)")
        self.db.execute("CREATE TABLE unrelated (nonce TEXT)")
        self.marker_page = self.db.execute("SELECT rootpage FROM sqlite_master WHERE name='marker'").fetchone()[0]
        self.other_page = self.db.execute("SELECT rootpage FROM sqlite_master WHERE name='unrelated'").fetchone()[0]
        self.assertEqual(self.db.execute("PRAGMA wal_checkpoint(TRUNCATE)").fetchone(), (0, 0, 0))
        self.nonce = "1234567890abcdef" * 2

    def has(self, page=None, nonce=None, path=None):
        with (path or self.wal).open("rb") as wal:
            return bool(self.selector(wal.fileno(), page or self.marker_page, (nonce or self.nonce).encode()))

    def test_exact_page_and_commit(self):
        self.assertFalse(self.has())
        with self.db:
            self.db.execute("INSERT INTO unrelated VALUES (?)", (self.nonce,))
        self.assertTrue(self.has(self.other_page))
        self.assertFalse(self.has())
        self.db.execute("INSERT INTO marker VALUES (?,?)", (self.nonce, "original"))
        self.assertFalse(self.has())
        self.db.rollback()
        self.assertFalse(self.has())
        with self.db:
            self.db.execute("INSERT INTO marker VALUES (?,?)", (self.nonce, "original"))
        self.assertTrue(self.has())
        self.assertFalse(self.has(nonce="0" * 32))
        self.assertFalse(self.has(self.marker_page + 10000))

    def test_corrupt_or_incomplete_wal_fails_closed(self):
        with self.db:
            self.db.execute("INSERT INTO marker VALUES (?,?)", (self.nonce, "original"))
        data = self.wal.read_bytes()
        self.assertTrue(self.has())
        copy = Path(self.temp.name) / "copy"
        for label, raw in (
            ("truncated_header", data[:31]),
            ("truncated_frame", data[:-1]),
            ("wrong_magic", b"xxxx" + data[4:]),
            ("wrong_salt", data[:40] + bytes(c ^ 255 for c in data[40:48]) + data[48:]),
            ("no_commit_frame", data[:36] + b"\0" * 4 + data[40:]),
        ):
            with self.subTest(label=label):
                copy.write_bytes(raw)
                self.assertFalse(self.has(path=copy))

    def test_cold_ledger_observer_preserves_source(self):
        with self.db:
            self.db.execute("INSERT INTO marker VALUES (?,?)", (self.nonce, "committed"))
        self.db.execute("INSERT INTO marker VALUES (?,?)", ("0" * 32, "uncommitted"))
        source = (self.db_path.read_bytes(), self.wal.read_bytes())
        with cold_ledger_rows(self.db_path) as query:
            self.assertEqual(query("PRAGMA integrity_check"), [("ok",)])
            self.assertEqual(query("SELECT nonce,original FROM marker"), [(self.nonce, "committed")])
            with self.assertRaises(sqlite3.OperationalError):
                query("DELETE FROM marker")
        self.assertEqual(source, (self.db_path.read_bytes(), self.wal.read_bytes()))
        self.assertEqual(self.db.execute("SELECT count(*) FROM marker").fetchone(), (2,))
        self.db.rollback()

    def test_checkpoint_generation_and_new_nonce(self):
        with self.db:
            self.db.execute("INSERT INTO marker VALUES (?,?)", (self.nonce, "original"))
        self.assertTrue(self.has())
        self.assertEqual(self.db.execute("PRAGMA wal_checkpoint(TRUNCATE)").fetchone(), (0, 0, 0))
        self.assertFalse(self.has())
        with self.db:
            self.db.execute("INSERT INTO unrelated VALUES (?)", (self.nonce,))
        self.assertFalse(self.has())


if __name__ == "__main__":
    unittest.main()
