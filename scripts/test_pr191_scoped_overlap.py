"""Calibrate the read-only Scoped overlap oracle and source overlay."""
from pathlib import Path
import sqlite3
import unittest

from pr191_scoped_overlap import held_proof, final_proof, NAME, QUARANTINE
from s2_p1_build_overlap_overlay import render_overlay, HOOK


class ScopedOverlapOracle(unittest.TestCase):
    def setUp(self):
        self.db=sqlite3.connect(":memory:")
        self.addCleanup(self.db.close)
        self.db.executescript("""
        CREATE TABLE operations(id,state,box_uuid,incarnation,native_operation_id,actor_id,layer,tracking_id);
        CREATE TABLE owned_boxes(uuid,project,name,incarnation,state,deleted_by_operation_id);
        CREATE TABLE network_bindings(box_uuid,state,acl);
        CREATE TABLE network_address_fences(box_uuid,family,state,operation_id);
        CREATE TABLE operation_resource_claims(operation_id,layer,resource_key);
        CREATE TABLE operation_events(operation_id,state,safe_detail);
        """)
        self.project="user-1001"
        self.actor="a"*32
        self.oid="b"*32
        self.native="c"*36
        self.uuid="d"*36
        self.track="e"*32
        self.acl="bw-peer-example"
        self.pin=dict(Bridge="incusbr-1001",IPv4="10.0.0.2",IPv6="fd00::2",MAC="00:16:3e:00:00:01",ACL=self.acl)
        self.db.execute("INSERT INTO operations VALUES (?,?,?,?,?,?,?,?)",
                        (self.oid,"VERIFYING","","",self.native,self.actor,self.project,self.track))
        keys=["box:"+NAME]+["create-pin:"+self.pin["Bridge"]+":"+k+":"+v for k,v in
                           (("v4",self.pin["IPv4"]),("v6",self.pin["IPv6"]),("mac",self.pin["MAC"]),("acl",QUARANTINE))]
        self.db.executemany("INSERT INTO operation_resource_claims VALUES (?,?,?)",
                            [(self.oid,self.project,k) for k in keys])

    def read(self,q,a=()):
        return self.db.execute(q,a).fetchall()

    def held(self):
        held_proof(self.read,self.project,self.actor,self.oid,self.native,self.uuid,self.pin)

    def final(self,delete=None):
        final_proof(self.read,self.project,self.actor,self.oid,self.native,self.track,self.uuid,self.acl,delete)

    def commit(self):
        self.db.execute("UPDATE operations SET state='SUCCEEDED',box_uuid=?,incarnation=?",(self.uuid,self.uuid))
        self.db.execute("INSERT INTO owned_boxes VALUES (?,?,?,?,?,?)",(self.uuid,self.project,NAME,self.uuid,"OWNED",""))
        self.db.execute("INSERT INTO network_bindings VALUES (?,?,?)",(self.uuid,"PINNED",self.acl))
        self.db.executemany("INSERT INTO network_address_fences VALUES (?,?,?,?)",
                            [(self.uuid,f,"RESERVED",self.oid) for f in ("ipv4","ipv6")])
        self.db.execute("INSERT INTO operation_events VALUES (?,?,?)",(self.oid,"SUCCEEDED","ORIGINAL_CREATE_UUID_ATOMIC"))

    def test_held_then_atomic_create_delete(self):
        self.held()
        self.commit()
        self.final()
        delete="f"*32
        self.db.execute("UPDATE owned_boxes SET state='DELETED',deleted_by_operation_id=?",(delete,))
        self.db.execute("UPDATE network_bindings SET state='RELEASED'")
        self.db.execute("UPDATE network_address_fences SET state='RELEASED',operation_id=?",(delete,))
        self.db.execute("INSERT INTO operation_events VALUES (?,?,?)",(delete,"SUCCEEDED","SCOPED_DELETE_ACL_ABSENT_ATOMIC"))
        self.final(delete)

    def test_early_uuid_or_claim_loss_refused(self):
        self.db.execute("UPDATE operations SET box_uuid=?",(self.uuid,))
        with self.assertRaises(AssertionError):
            self.held()
        self.db.execute("UPDATE operations SET box_uuid=''")
        self.db.execute("DELETE FROM operation_resource_claims WHERE resource_key=?",("box:"+NAME,))
        with self.assertRaises(AssertionError):
            self.held()

    def test_split_commit_refused(self):
        self.commit()
        self.db.execute("DELETE FROM network_address_fences WHERE family='ipv6'")
        with self.assertRaises(AssertionError):
            self.final()

    def test_overlay_is_single_scoped_pretransaction_insertion(self):
        original=Path("internal/state/lifecycle.go").read_bytes()
        scoped=render_overlay(original,True)
        legacy=render_overlay(original,False)
        self.assertEqual(scoped.count("TEST-ONLY GO OVERLAY"),1)
        self.assertIn("if scoped && binding != nil",scoped)
        self.assertNotIn("if scoped && binding != nil && os.Getenv",legacy)
        i=scoped.index("func (s *Store) finalizeVerifiedCreate(")
        self.assertLess(i,scoped.index("TEST-ONLY GO OVERLAY"))
        self.assertLess(scoped.index("TEST-ONLY GO OVERLAY"),scoped.index("tx, err := s.db.BeginTx(ctx, nil)",i))
        self.assertEqual(Path("internal/state/lifecycle.go").read_bytes(),original)


if __name__=="__main__":
    unittest.main()
