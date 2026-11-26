"""Private calibration of the independent installed-S2 fixed-ledger oracle."""
import hashlib
import sqlite3
import unittest

from pr191_scoped_s2_native import KEYS, MAIN, verify_history, verify_originals, verify_public_wan, by_key


class ScopedS2OracleTests(unittest.TestCase):
    project = "user-1002"
    actor = "a" * 32
    actor2 = "b" * 32
    uuid = "first-incarnation"
    reused = "second-incarnation"

    def setUp(self):
        self.db = sqlite3.connect(":memory:")
        self.addCleanup(self.db.close)
        self.db.executescript("""
        CREATE TABLE operations (id,tracking_id,actor_id,token_id,layer,action,state,box_uuid,incarnation,native_operation_id,key_digest,work_ref);
        CREATE TABLE owned_boxes (uuid,incarnation,state,deleted_by_operation_id,project,name);
        CREATE TABLE network_bindings (project,box_uuid,nic,state);
        CREATE TABLE exec_outputs (operation_id,exit_code);
        """)
        self.seq = 0

    def read(self, sql, args=()):
        return self.db.execute(sql, args).fetchall()

    def add(self, key, action, actor=None, box=None, layer=None,
            state="SUCCEEDED", native=True, work=""):
        self.seq += 1
        oid = f"{self.seq:032x}"
        track = f"{self.seq+1000:032x}"
        actor = actor or self.actor
        box = self.uuid if box is None else box
        native_id = f"native-{self.seq}" if native else ""
        self.db.execute("INSERT INTO operations VALUES (?,?,?,?,?,?,?,?,?,?,?,?)",
                        (oid, track, actor, actor, layer or self.project, action, state, box, box,
                         native_id, hashlib.sha256(key.encode()).hexdigest(), work))
        if action == "box_exec" and state == "SUCCEEDED":
            self.db.execute("INSERT INTO exec_outputs VALUES (?,0)", (oid,))
        return oid

    def seed_originals(self):
        originals = []
        for i, (key, action) in enumerate(KEYS):
            originals.append(self.add(key, action,
                                      actor=self.actor2 if "actor-two" in key else self.actor,
                                      box=self.reused if i >= len(KEYS)-2 else self.uuid))
        for actor in (self.actor, self.actor2):
            self.add("native-s2-same-key-two-independent-admin-actors-01", "box_exec", actor=actor)
        for uuid, deletion in ((self.uuid, originals[-3]), (self.reused, originals[-1])):
            self.db.execute("INSERT INTO owned_boxes VALUES (?,?,'DELETED',?,?,?)",
                            (uuid, uuid, deletion, self.project, MAIN))
            self.db.execute("INSERT INTO network_bindings VALUES (?,?,'eth0','RELEASED')",
                            (self.project, uuid))

    def test_exact_lineage_and_independent_actors(self):
        self.seed_originals()
        result = verify_originals(self.read, self.project, self.actor, self.actor2)
        self.assertEqual(len(result), len(KEYS)+2)

    def test_wrong_actor_or_native_receipt_refused(self):
        self.seed_originals()
        with self.assertRaises(AssertionError):
            verify_originals(self.read, self.project, self.actor2, self.actor)
        self.db.execute("UPDATE operations SET native_operation_id='' WHERE action='box_update'")
        with self.assertRaises(AssertionError):
            verify_originals(self.read, self.project, self.actor, self.actor2)

    def test_stale_incarnation_refused(self):
        self.seed_originals()
        self.db.execute("UPDATE owned_boxes SET deleted_by_operation_id='wrong'")
        with self.assertRaises(AssertionError):
            verify_originals(self.read, self.project, self.actor, self.actor2)

    def wan_record(self,key,code,status):
        self.add(key,"box_exec")
        r=by_key(self.read,key)[0]
        self.db.execute("UPDATE exec_outputs SET exit_code=? WHERE operation_id=?",(code,r[0]))
        return "S2_NATIVE_PUBLIC_IPV4_ORIGINAL "+" ".join((key,r[0],r[9],r[1],r[7],str(code),status))

    def test_wan_positive_negative_and_missing_positive_control(self):
        prefix="native-s2-wan-"
        records=[self.wan_record(prefix+"curl-available-01",0,"CLIENT"),
                 self.wan_record(prefix+"online-before-01",0,"200"),
                 self.wan_record(prefix+"offline-01",28,"000"),
                 self.wan_record(prefix+"online-after-01",0,"204")]
        log="\n".join(records+["S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS"])
        self.assertEqual(verify_public_wan(self.read,self.project,self.actor,self.uuid,log),"PASS")
        with self.assertRaises(AssertionError):
            verify_public_wan(self.read,self.project,self.actor,self.uuid,log.replace("28 000","28 403"))
        with self.assertRaises(AssertionError):
            verify_public_wan(self.read,self.project,self.actor,self.uuid,log.replace("0 204","0 404"))
        with self.assertRaises(AssertionError):
            verify_public_wan(self.read,self.project,self.actor,self.uuid,log.replace(records[2],""))
        self.db.execute("DELETE FROM operations")
        self.db.execute("DELETE FROM exec_outputs")
        records=[self.wan_record(prefix+"curl-available-01",0,"CLIENT")]
        records.extend(self.wan_record(prefix+f"online-before-{i:02d}",28,"000") for i in range(1,5))
        log="\n".join(records+["S2_NATIVE_EXTERNAL_IPV4_WAN_BASELINE=UNVERIFIED","P1B_SCOPED_PUBLIC_IPV4_TLS=UNVERIFIED"])
        self.assertEqual(verify_public_wan(self.read,self.project,self.actor,self.uuid,log),"UNVERIFIED")
        with self.assertRaises(AssertionError):
            verify_public_wan(self.read,self.project,self.actor,self.uuid,log.replace("P1B_SCOPED_PUBLIC_IPV4_TLS=UNVERIFIED","S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS"))

    def test_history_count_and_original_cancel(self):
        for i in range(132):
            self.add(f"history-{i}", "box_exec", box="", native=False,
                     state="CANCELLED", work="lab/history-120")
        queued = self.add("queued", "box_exec", box="", native=False,
                          state="CANCELLED", work="lab/queued-cancel")
        foreign = self.add("foreign", "box_exec", box="", native=False,
                           state="CANCELLED", layer=self.project+"-foreign", work="lab/foreign-layer")
        self.add("native-s2-queued-cancel-tool-original-01", "operation_cancel", box="", native=False)
        verify_history(self.read, self.project, self.actor, queued, foreign)
        with self.assertRaises(AssertionError):
            verify_history(self.read, self.project, self.actor, foreign, queued)
        self.db.execute("UPDATE operations SET native_operation_id='false' WHERE id=?", (queued,))
        with self.assertRaises(AssertionError):
            verify_history(self.read, self.project, self.actor, queued, foreign)


if __name__ == "__main__":
    unittest.main()
