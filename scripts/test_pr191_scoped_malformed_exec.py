"""Calibrate malformed-terminal original binding without native effects."""
import sqlite3
import unittest

from pr191_scoped_malformed_exec import check_original, digest, NAME, WORK, KEY, COMPETING


class MalformedExecOracle(unittest.TestCase):
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
        self.cp={"exec_id":"b"*32,"exec_tracking":"c"*32,
                 "native_id":"11111111-1111-1111-1111-111111111111",
                 "box_id":"22222222-2222-2222-2222-222222222222"}
        oid,native,tracking,u=(self.cp[k] for k in ("exec_id","native_id","exec_tracking","box_id"))
        self.db.execute("INSERT INTO operations VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)",
                        (oid,tracking,self.actor,self.actor,self.project,"box_exec","UNKNOWN",native,u,u,
                         "EXEC_TERMINAL_EVIDENCE_UNVERIFIED",digest(KEY),WORK))
        self.db.execute("INSERT INTO operation_resource_claims VALUES (?,?,?)",(oid,self.project,"box:"+NAME))
        self.db.executemany("INSERT INTO operation_events VALUES (?,?,?)",
                            [(oid,i,state) for i,state in enumerate(("ADMITTED","DISPATCHING","VERIFYING","UNKNOWN"))])
        self.accepted="REAL_EXEC_202_NATIVE_ID="+native+"\n"
        self.malformed="MALFORMED_FINAL_NATIVE_ID="+native+"\n"

    def proof(self, accepted=None, malformed=None, duplicate=False):
        read=lambda q,a=():self.db.execute(q,a).fetchall()
        check_original(read,self.project,self.actor,self.cp,
                       self.accepted if accepted is None else accepted,
                       self.malformed if malformed is None else malformed,duplicate)

    def test_exact_unknown(self):
        self.proof()

    def test_different_marker_or_duplicate_refused(self):
        for kwargs in (dict(accepted="other"),dict(malformed="other"),dict(duplicate=True)):
            with self.subTest(kwargs=kwargs),self.assertRaises(AssertionError):
                self.proof(**kwargs)

    def test_fake_terminal_output_refused(self):
        self.db.execute("INSERT INTO exec_outputs VALUES (?)",(self.cp["exec_id"],))
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
