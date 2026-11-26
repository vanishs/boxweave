"""Calibrate lost Stop acknowledgement original binding without native effects."""
import sqlite3
import unittest

from pr191_scoped_stop_ack import check_original, digest, NAME, WORK, KEY, COMPETING


class StopAckOracle(unittest.TestCase):
    def setUp(self):
        self.db=sqlite3.connect(":memory:")
        self.addCleanup(self.db.close)
        self.db.executescript("""
        CREATE TABLE operations(id,tracking_id,actor_id,token_id,layer,action,state,native_operation_id,box_uuid,incarnation,error_class,key_digest,work_ref);
        CREATE TABLE operation_resource_claims(operation_id,layer,resource_key);
        CREATE TABLE operation_events(operation_id,seq,state);
        CREATE TABLE exec_outputs(operation_id);
        CREATE TABLE recovery_audit(operation_id);
        """)
        self.project="user-1001"
        self.actor="a"*32
        self.cp={"action_id":"b"*32,"action_tracking":"c"*32,
                 "create_native":"11111111-1111-1111-1111-111111111111",
                 "box_id":"22222222-2222-2222-2222-222222222222"}
        oid,native,tracking,u=(self.cp[k] for k in ("action_id","create_native","action_tracking","box_id"))
        self.db.execute("INSERT INTO operations VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)",
                        (oid,tracking,self.actor,self.actor,self.project,"box_stop","UNKNOWN","",u,u,
                         "NATIVE_OUTCOME_AMBIGUOUS",digest(KEY),WORK))
        self.db.execute("INSERT INTO operation_resource_claims VALUES (?,?,?)",(oid,self.project,"box:"+NAME))
        self.db.executemany("INSERT INTO operation_events VALUES (?,?,?)",
                            [(oid,i,state) for i,state in enumerate(("ADMITTED","DISPATCHING","UNKNOWN"))])
        self.accepted="S2_P1_NATIVE_STOP_202_ACCEPTED_ID=33333333-3333-3333-3333-333333333333"+"\n"

    def proof(self, accepted=None, duplicate=False):
        read=lambda q,a=():self.db.execute(q,a).fetchall()
        check_original(read,self.project,self.actor,self.cp,
                       self.accepted if accepted is None else accepted,
                       duplicate)

    def test_exact_unknown(self):
        self.proof()

    def test_different_marker_or_duplicate_refused(self):
        for kwargs in (dict(accepted="other"),dict(duplicate=True)):
            with self.subTest(kwargs=kwargs),self.assertRaises(AssertionError):
                self.proof(**kwargs)

    def test_invented_native_receipt_refused(self):
        self.db.execute("UPDATE operations SET native_operation_id=?",("33333333-3333-3333-3333-333333333333",))
        with self.assertRaises(AssertionError):
            self.proof()

    def test_seed_native_reuse_refused(self):
        self.cp["create_native"]="33333333-3333-3333-3333-333333333333"
        with self.assertRaises(AssertionError):
            self.proof()

    def test_lost_claim_refused(self):
        self.db.execute("DELETE FROM operation_resource_claims")
        with self.assertRaises(AssertionError):
            self.proof()

    def test_competing_admission_refused(self):
        self.db.execute("INSERT INTO operations(id,key_digest) VALUES (?,?)",("d"*32,digest(COMPETING)))
        with self.assertRaises(AssertionError):
            self.proof()


if __name__=="__main__":
    unittest.main()
