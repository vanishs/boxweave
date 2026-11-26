"""Private source and original-lineage calibration for alternate pool."""
import hashlib
import os
from pathlib import Path
import sqlite3
import subprocess
import unittest

from pr191_scoped_altpool import check_absence, KEYS, NAME

ROOT=Path(__file__).resolve().parents[1]


class AlternatePoolOracle(unittest.TestCase):
    def test_refuses_non_hosted(self):
        env=dict(os.environ,GITHUB_ACTIONS="false",RUNNER_ENVIRONMENT="local")
        r=subprocess.run(["sh","-e",str(ROOT/"scripts/s2_p2_alternate_pool_setup.sh")],
                         env=env,capture_output=True,timeout=5)
        self.assertNotEqual(r.returncode,0)
        self.assertNotIn(b"=PASS",r.stdout)

    def test_preserves_original_and_refuses_changed_binding(self):
        project="user-1001"; actor="a"*32; u="1"*8+"-"+"1"*4+"-"+"1"*4+"-"+"1"*4+"-"+"1"*12
        db=sqlite3.connect(":memory:")
        self.addCleanup(db.close)
        db.executescript("""
        CREATE TABLE operations(id,native_operation_id,tracking_id,layer,actor_id,token_id,action,state,box_uuid,incarnation,key_digest);
        CREATE TABLE owned_boxes(uuid,incarnation,state,deleted_by_operation_id,project,name);
        CREATE TABLE network_bindings(project,box_uuid,state);
        """)
        cp={"box_id":u}
        for i,kind in enumerate(("create","delete"),2):
            oid=str(i)*32; native=u.replace("1",str(i)); tracking=str(i+2)*32
            cp.update({kind+"_id":oid,kind+"_native":native,kind+"_tracking":tracking})
            db.execute("INSERT INTO operations VALUES (?,?,?,?,?,?,?,?,?,?,?)",
                       (oid,native,tracking,project,actor,actor,"box_"+kind,"SUCCEEDED",u,u,
                        hashlib.sha256(KEYS[kind].encode()).hexdigest()))
        db.execute("INSERT INTO owned_boxes VALUES (?,?,?,?,?,?)",(u,u,"DELETED",cp["delete_id"],project,NAME))
        db.execute("INSERT INTO network_bindings VALUES (?,?,?)",(project,u,"RELEASED"))
        read=lambda q,a=():db.execute(q,a).fetchall()
        check_absence(read,project,actor,cp)
        for changed in (dict(cp,delete_id="9"*32),dict(cp,create_native=cp["delete_native"])):
            with self.assertRaises(AssertionError):
                check_absence(read,project,actor,changed)
        db.execute("UPDATE network_bindings SET state='PINNED'")
        with self.assertRaises(AssertionError):
            check_absence(read,project,actor,cp)


if __name__=="__main__":
    unittest.main()
