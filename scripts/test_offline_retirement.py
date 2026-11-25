import copy
import unittest
import uninstall_core as u


class OfflineScopeTests(unittest.TestCase):
    def setUp(self):
        self.box = "a" * 32
        self.original = {"id": "b"*32, "state": "UNKNOWN", "action": "box_stop",
                         "retirement_scope": "EXACT_OWNED_POWER_ORIGINAL",
                         "claim_protection": "RETAINED", "recorded_claims": 1,
                         "native_receipt": True, "box_uuid": self.box, "incarnation": self.box}
        self.report = {"installation": {"mode": "LEGACY", "adapter": "PINNED_RC13",
                       "identity": {"installed_identity": "VERIFIED", "service_unit": "INACTIVE"}},
                       "native": {"legacy_lease_proof": "VERIFIED", "base_acl_config": "RECORDED_PREIMAGE_MATCH",
                       "owned_matched": [self.box], "owned_power": {self.box: "Stopped"}},
                       "ledger": {"boxes": [{"uuid": self.box, "incarnation": self.box,
                       "create_proof": True, "binding_state": "PINNED"}],
                       "nonterminal_or_unknown_originals": [self.original]}}
        self.preview = {"blockers": [{"code": "ORIGINAL_UNKNOWN", "resource": self.original["id"], "severity": "HARD"}]}

    def test_exact_unknown_retained(self):
        before = copy.deepcopy(self.report)
        blockers, eligible = u.offline_scope_blockers(self.report, self.preview)
        self.assertEqual(blockers, [])
        self.assertEqual(eligible, [self.original])
        self.assertEqual(self.report, before)

    def test_unknown_ownership_mismatch(self):
        for field, value in (("recorded_claims", 2), ("action", "box_create"),
                             ("native_receipt", False), ("incarnation", "foreign"),
                             ("retirement_scope", "UNPROVEN"), ("state", "RUNNING")):
            with self.subTest(field=field):
                report = copy.deepcopy(self.report)
                report["ledger"]["nonterminal_or_unknown_originals"][0][field] = value
                blockers, eligible = u.offline_scope_blockers(report, self.preview)
                self.assertTrue(blockers)
                self.assertFalse(eligible)

    def test_native_and_service_refusals(self):
        for section, field, value in (("native", "owned_power", {self.box: "Running"}),
                    ("native", "legacy_lease_proof", "UNVERIFIED"),
                    ("native", "base_acl_config", "DRIFT"),
                    ("installation", "mode", "SCOPED_V2"),
                    ("ledger", "peers", [{"state": "UNKNOWN"}]),
                    ("ledger", "forwards", [{"state": "ACTIVE"}]),
                    ("ledger", "legacy_compactions", ["x"])):
            with self.subTest(field=field):
                report = copy.deepcopy(self.report)
                report[section][field] = value
                self.assertTrue(u.offline_scope_blockers(report, self.preview)[0])

    def test_unrelated_hard_blocker_preserved(self):
        self.preview["blockers"].append({"code":"FOREIGN_CONSUMER","resource":"foreign","severity":"HARD"})
        self.assertIn("FOREIGN_CONSUMER:foreign", u.offline_scope_blockers(self.report, self.preview)[0])


class ColdSQLiteTests(unittest.TestCase):
    def setUp(self):
        import os
        from types import SimpleNamespace
        from unittest.mock import patch
        # Disposable test files belong to this process, not a live controller.
        owner = patch.object(u.pwd, "getpwnam", return_value=SimpleNamespace(pw_uid=os.getuid()))
        owner.start()
        self.addCleanup(owner.stop)

    def test_crash_wal_preserves_committed_transaction(self):
        import sqlite3
        import subprocess
        import sys
        import tempfile
        from pathlib import Path
        with tempfile.TemporaryDirectory() as d:
            db = Path(d) / "ledger.sqlite"
            code = """import os,sqlite3,sys
os.umask(0o077)
c=sqlite3.connect(sys.argv[1])
c.execute('pragma journal_mode=wal')
c.execute('pragma wal_autocheckpoint=0')
c.execute('create table original(state text)')
c.execute("insert into original values('UNKNOWN')")
c.commit()
os._exit(0)
"""
            subprocess.run([sys.executable, "-c", code, str(db)], check=True)
            before = u.cold_sqlite_files(db)
            self.assertGreater(before["-wal"]["size"], 0)
            self.assertIsNotNone(before["-shm"])
            for _ in range(3):
                with u.cold_sqlite_copy(db) as copy:
                    conn = sqlite3.connect(copy.as_uri()+"?mode=ro", uri=True)
                    try:
                        self.assertEqual(conn.execute("pragma quick_check").fetchall(), [("ok",)])
                        self.assertEqual(conn.execute("select state from original").fetchall(), [("UNKNOWN",)])
                    finally:
                        conn.close()
                self.assertEqual(u.cold_sqlite_files(db), before)
            code = """import sqlite3,sys
c=sqlite3.connect(sys.argv[1])
c.execute('begin')
c.execute('select * from original').fetchall()
print('READY',flush=True)
sys.stdin.readline()
"""
            child = subprocess.Popen([sys.executable, "-c", code, str(db)], stdin=subprocess.PIPE,
                                     stdout=subprocess.PIPE, text=True)
            try:
                self.assertEqual(child.stdout.readline().strip(), "READY")
                with self.assertRaisesRegex(u.UninstallError, "SOURCE_BUSY"):
                    with u.cold_sqlite_copy(db):
                        self.fail("live copy accepted")
            finally:
                child.communicate("\n", timeout=10)

    def test_rollback_journal_refused_without_effect(self):
        import tempfile
        from pathlib import Path
        with tempfile.TemporaryDirectory() as d:
            db = Path(d) / "ledger.sqlite"
            db.write_bytes(b"fixture")
            db.chmod(0o600)
            journal = Path(str(db)+"-journal")
            journal.write_bytes(b"preserve")
            journal.chmod(0o600)
            before = u.cold_sqlite_files(db)
            with self.assertRaisesRegex(u.UninstallError, "ROLLBACK_JOURNAL"):
                with u.cold_sqlite_copy(db):
                    self.fail("rollback journal accepted")
            self.assertEqual(u.cold_sqlite_files(db), before)


if __name__ == "__main__":
    unittest.main()
