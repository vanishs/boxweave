"""Private SQLite calibration of the shared Exec WAL atomicity oracle."""
import sqlite3
import unittest

from pr191_scoped_wal_cut import NAME, assert_ledger


class ExecWALOracleTests(unittest.TestCase):
    def setUp(self):
        self.db = sqlite3.connect(":memory:")
        self.addCleanup(self.db.close)
        self.db.executescript("""
            CREATE TABLE operation_events (operation_id,state);
            CREATE TABLE owned_boxes (uuid,state);
            CREATE TABLE network_bindings (box_uuid,state,acl);
            CREATE TABLE network_address_fences (box_uuid,state,family);
            CREATE TABLE operation_resource_claims (operation_id);
            CREATE TABLE exec_outputs (operation_id,stdout_path,stderr_path,exit_code);
            INSERT INTO owned_boxes VALUES ('uuid','OWNED');
            INSERT INTO network_bindings VALUES ('uuid','PINNED','acl');
            INSERT INTO network_address_fences VALUES ('uuid','RESERVED','ipv4'),('uuid','RESERVED','ipv6');
            INSERT INTO operation_resource_claims VALUES ('original');
        """)
        self.prefix = "/1.0/instances/" + NAME + "/logs/exec-output/exec_native"

    def check(self, state):
        assert_ledger("box_exec", state, "original", "uuid", "acl",
                      query=lambda sql, params=(): self.db.execute(sql, params).fetchall(),
                      native_id="native")

    def test_complete_and_unknown(self):
        self.check("VERIFYING")
        self.check("UNKNOWN")
        self.db.execute("INSERT INTO operation_events VALUES ('original','SUCCEEDED')")
        self.db.execute("INSERT INTO exec_outputs VALUES (?,?,?,0)",
                        ("original", self.prefix + ".stdout", self.prefix + ".stderr"))
        self.check("SUCCEEDED")

    def test_partial_output_and_event_refused(self):
        self.db.execute("INSERT INTO exec_outputs VALUES (?,?,?,0)",
                        ("original", self.prefix + ".stdout", self.prefix + ".stderr"))
        with self.assertRaises(AssertionError):
            self.check("UNKNOWN")
        with self.assertRaises(AssertionError):
            self.check("SUCCEEDED")
        self.db.execute("INSERT INTO operation_events VALUES ('original','SUCCEEDED')")
        self.db.execute("UPDATE exec_outputs SET exit_code=7")
        with self.assertRaises(AssertionError):
            self.check("SUCCEEDED")

    def test_lost_claim_and_wrong_lineage_refused(self):
        self.db.execute("DELETE FROM operation_resource_claims")
        with self.assertRaises(AssertionError):
            self.check("UNKNOWN")
        self.db.execute("INSERT INTO operation_resource_claims VALUES ('original')")
        self.db.execute("UPDATE network_bindings SET state='RELEASED'")
        with self.assertRaises(AssertionError):
            self.check("UNKNOWN")


if __name__ == "__main__":
    unittest.main()
