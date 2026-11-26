"""Calibrate the exact fixture-scoped ACK transaction predicates privately."""
import sqlite3
import unittest

from pr191_scoped_ack_loss import ack_fault_ddl


class UpdateAckFaultTests(unittest.TestCase):
    def test_only_original_normal_ack_rolls_back(self):
        for preserve in (False, True):
            with self.subTest(preserve=preserve):
                db = sqlite3.connect(":memory:")
                self.addCleanup(db.close)
                db.executescript("""
                    CREATE TABLE operations (id,action,work_ref,state,native_operation_id);
                    CREATE TABLE operation_events (operation_id,state);
                    INSERT INTO operations VALUES
                    ('original','box_update','p1b/first-ack-box_update','DISPATCHING',''),
                    ('other','box_update','other','DISPATCHING',''),
                    ('create','box_create','p1b/first-ack-box_update','DISPATCHING','');
                """)
                db.execute(ack_fault_ddl("box_update", "p1b/first-ack-box_update", preserve))
                for oid in ("other", "create"):
                    with db:
                        db.execute("UPDATE operations SET state='VERIFYING',native_operation_id='other-id' WHERE id=?", (oid,))
                        db.execute("INSERT INTO operation_events VALUES (?, 'VERIFYING')", (oid,))
                with self.assertRaises(sqlite3.IntegrityError), db:
                    db.execute("UPDATE operations SET state='VERIFYING',native_operation_id='native' WHERE id='original'")
                    db.execute("INSERT INTO operation_events VALUES ('original','VERIFYING')")
                self.assertEqual(db.execute("SELECT state,native_operation_id FROM operations WHERE id='original'").fetchall(),
                                 [("DISPATCHING", "")])
                self.assertEqual(db.execute("SELECT * FROM operation_events WHERE operation_id='original'").fetchall(), [])
                if preserve:
                    with db:
                        db.execute("UPDATE operations SET state='UNKNOWN',native_operation_id='native' WHERE id='original'")
                        db.execute("INSERT INTO operation_events VALUES ('original','UNKNOWN')")
                else:
                    with self.assertRaises(sqlite3.IntegrityError), db:
                        db.execute("UPDATE operations SET state='UNKNOWN',native_operation_id='native' WHERE id='original'")
                    with db:
                        db.execute("UPDATE operations SET state='UNKNOWN' WHERE id='original'")
                        db.execute("INSERT INTO operation_events VALUES ('original','UNKNOWN')")
                self.assertEqual(db.execute("SELECT state,native_operation_id FROM operations WHERE id='original'").fetchall(),
                                 [("UNKNOWN", "native" if preserve else "")])
                self.assertEqual(db.execute("SELECT state FROM operation_events WHERE operation_id='original'").fetchall(), [("UNKNOWN",)])

    def test_invalid_selector_refused(self):
        for action, work, preserve in (("box_exec", "p1b/first-ack-box_exec", False),
                                       ("box_update", "other", False),
                                       ("box_create", "p1b/first-ack-box_create", True)):
            with self.assertRaises(AssertionError):
                ack_fault_ddl(action, work, preserve)


if __name__ == "__main__":
    unittest.main()
