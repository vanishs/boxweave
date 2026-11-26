"""Source-only negative calibration of the independent original receipt witness."""
import hashlib
import sqlite3
import unittest

from pr191_scoped_sdk_compat import ACTIONS, NAME, originals, verify_originals


class OldSDKOriginalsTests(unittest.TestCase):
    def setUp(self):
        self.db = sqlite3.connect(":memory:")
        self.addCleanup(self.db.close)
        self.db.executescript("""
            CREATE TABLE operations (id,tracking_id,actor_id,token_id,layer,action,state,
                                     box_uuid,incarnation,native_operation_id,key_digest);
            CREATE TABLE owned_boxes (uuid,incarnation,state,deleted_by_operation_id,project,name);
            CREATE TABLE network_bindings (state,project,box_uuid,nic);
            CREATE TABLE exec_outputs (exit_code,operation_id);
        """)
        for i, action in enumerate(ACTIONS, 1):
            key = hashlib.sha256(("s2-old-sdk-positive-" + action).encode()).hexdigest()
            self.db.execute("INSERT INTO operations VALUES (?,?,?,?,?,?,?,?,?,?,?)",
                            (f"{i:032x}", f"{i+10:032x}", "actor", "token", "layer",
                             "box_"+action, "SUCCEEDED", "uuid", "uuid", f"native-{i}", key))
        self.db.execute("INSERT INTO owned_boxes VALUES (?,?,?,?,?,?)",
                        ("uuid", "uuid", "DELETED", f"{6:032x}", "project", NAME))
        self.db.execute("INSERT INTO network_bindings VALUES ('RELEASED','project','uuid','eth0')")
        self.db.execute("INSERT INTO exec_outputs VALUES (0,?)", (f"{4:032x}",))

    def read(self, query, params):
        return self.db.execute(query, params).fetchall()

    def test_exact_originals(self):
        self.assertEqual(len(verify_originals(self.read, "project")), 6)

    def test_refuse_missing_original(self):
        self.db.execute("DELETE FROM operations WHERE action='box_update'")
        with self.assertRaises(AssertionError):
            verify_originals(self.read, "project")

    def test_refuse_duplicate_original_key(self):
        self.db.execute("INSERT INTO operations SELECT * FROM operations WHERE action='box_update'")
        with self.assertRaises(AssertionError):
            originals(self.read)

    def test_refuse_inconsistent_identity_state_and_receipts(self):
        for update in (
                "UPDATE operations SET state='UNKNOWN' WHERE action='box_exec'",
                "UPDATE operations SET box_uuid='other' WHERE action='box_update'",
                "UPDATE operations SET actor_id='other' WHERE action='box_stop'",
                "UPDATE operations SET native_operation_id='' WHERE action='box_delete'",
                "UPDATE operations SET tracking_id=(SELECT tracking_id FROM operations WHERE action='box_create') WHERE action='box_delete'",
                "UPDATE owned_boxes SET deleted_by_operation_id='other'",
                "UPDATE network_bindings SET state='PINNED'",
                "UPDATE exec_outputs SET exit_code=7"):
            with self.subTest(update=update):
                self.db.execute("SAVEPOINT bad")
                try:
                    self.db.execute(update)
                    with self.assertRaises(AssertionError):
                        verify_originals(self.read, "project")
                finally:
                    self.db.execute("ROLLBACK TO bad")
                    self.db.execute("RELEASE bad")


if __name__ == "__main__":
    unittest.main()
