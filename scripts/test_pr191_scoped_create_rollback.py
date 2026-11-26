"""Calibrate the independent read-only original Create rollback snapshot."""
import sqlite3
import unittest

from pr191_scoped_lifecycle_recovery import rollback_snapshot, NAME


class ScopedCreateRollback(unittest.TestCase):
    def setUp(self):
        self.db=sqlite3.connect(":memory:")
        self.addCleanup(self.db.close)
        self.db.executescript("""
        CREATE TABLE operations(id,state,box_uuid,incarnation,native_operation_id,tracking_id,error_class,updated_at);
        CREATE TABLE owned_boxes(uuid);
        CREATE TABLE network_bindings(box_uuid);
        CREATE TABLE network_address_fences(box_uuid);
        CREATE TABLE operation_resource_claims(operation_id,layer,resource_key);
        CREATE TABLE operation_events(operation_id,seq,state,safe_detail);
        CREATE TABLE recovery_audit(operation_id);
        """)
        self.oid="a"*32
        self.uuid="b"*36
        self.project="user-1001"
        self.pin=dict(Bridge="incusbr-1001",IPv4="10.0.0.2",IPv6="fd00::2",MAC="00:16:3e:00:00:01",ACL="bw-peer-owned")
        self.db.execute("INSERT INTO operations VALUES (?,?,?,?,?,?,?,?)",
                        (self.oid,"UNKNOWN","","","c"*36,"d"*32,"POSTCONDITION_UNVERIFIED",1))
        self.db.execute("INSERT INTO operation_events VALUES (?,?,?,?)",(self.oid,1,"UNKNOWN","POSTCONDITION_UNVERIFIED"))
        keys=["box:"+NAME]+["create-pin:"+self.pin["Bridge"]+":"+k+":"+v for k,v in
                            (("v4",self.pin["IPv4"]),("v6",self.pin["IPv6"]),("mac",self.pin["MAC"]),("acl","bw-peer-quarantine"))]
        self.db.executemany("INSERT INTO operation_resource_claims VALUES (?,?,?)",
                            [(self.oid,self.project,k) for k in keys])

    def snap(self):
        return rollback_snapshot(self.oid,self.uuid,self.project,self.pin,
                                 query=lambda q,a=():self.db.execute(q,a).fetchall())

    def test_stable_original(self):
        self.assertEqual(self.snap(),self.snap())

    def test_leaked_partial_commit_refused(self):
        for table in ("owned_boxes","network_bindings","network_address_fences"):
            with self.subTest(table=table):
                self.db.execute("INSERT INTO "+table+" VALUES (?)",(self.uuid,))
                with self.assertRaises(AssertionError):
                    self.snap()
                self.db.execute("DELETE FROM "+table)

    def test_lost_provisional_claim_refused(self):
        self.db.execute("DELETE FROM operation_resource_claims WHERE resource_key LIKE '%:acl:%'")
        with self.assertRaises(AssertionError):
            self.snap()

    def test_split_success_event_refused(self):
        self.db.execute("INSERT INTO operation_events VALUES (?,?,?,?)",(self.oid,2,"SUCCEEDED","ORIGINAL_CREATE_UUID_ATOMIC"))
        with self.assertRaises(AssertionError):
            self.snap()

    def test_new_original_changes_snapshot(self):
        old=self.snap()
        self.db.execute("INSERT INTO operations VALUES (?,?,?,?,?,?,?,?)",
                        ("e"*32,"ADMITTED","","","","f"*32,"",2))
        self.assertNotEqual(old,self.snap())


if __name__=="__main__":
    unittest.main()
