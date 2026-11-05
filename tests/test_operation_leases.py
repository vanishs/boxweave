"""R-04 operator lease and durable claim collision regression (fixture only)."""
import hashlib
import os
import sqlite3
import tempfile
import unittest

from bwmini.operation_coordination import (
    OperationCoordinator, CoordinationFault, box_resource, name_resource,
)

IDENTITY_A=hashlib.sha256(b"Box-A").hexdigest()
IDENTITY_B=hashlib.sha256(b"Box-B").hexdigest()
SCOPE=hashlib.sha256(b"scope").hexdigest()
A=box_resource(IDENTITY_A)
B=box_resource(IDENTITY_B)

class LeaseTests(unittest.TestCase):
    def setUp(self):
        self.folder=tempfile.TemporaryDirectory()
        self.addCleanup(self.folder.cleanup)
        self.file=os.path.join(self.folder.name,"coord.sqlite")
        self.now=[1000]
        self.c=OperationCoordinator(lambda:sqlite3.connect(self.file,timeout=2),clock=lambda:self.now[0])
        self.c.initialize()

    def test_nonleased_operations_continue_without_token(self):
        self.c.acquire("1"*32,IDENTITY_A,"receipt_a",[A])
        self.c.dispatch_intent("1"*32)
        with self.assertRaisesRegex(CoordinationFault,"RECOVERY_REQUIRED"):
            self.c.acquire("2"*32,IDENTITY_A,"receipt_b",[A])
        self.c.unknown("1"*32)
        with self.assertRaisesRegex(CoordinationFault,"RECOVERY_REQUIRED"):
            self.c.acquire("2"*32,IDENTITY_A,"receipt_b",[A])

    def test_lease_blocks_other_agent_without_ignoring_different_box(self):
        result=self.c.lease("acquire",A,IDENTITY_A)
        self.assertEqual(result["expires"],1600)
        with self.assertRaisesRegex(CoordinationFault,"RESOURCE_BUSY"):
            self.c.acquire("1"*32,IDENTITY_A,"receipt_a",[A])
        self.c.acquire("2"*32,IDENTITY_B,"receipt_b",[B])
        self.c.dispatch_intent("2"*32)
        self.c.unknown("2"*32)
        self.c.acquire("3"*32,IDENTITY_A,"receipt_c",[A],lease_token=result["lease_token"])
        self.c.dispatch_intent("3"*32)
        self.c.unknown("3"*32)
        self.now[0]=1700
        with self.assertRaisesRegex(CoordinationFault,"RECOVERY_REQUIRED"):
            self.c.acquire("4"*32,IDENTITY_A,"receipt_d",[A])

    def test_renew_release_token_hash_and_expiry(self):
        value=self.c.lease("acquire",A,IDENTITY_A)
        with self.assertRaisesRegex(CoordinationFault,"LEASE_TOKEN_INVALID"):
            self.c.lease("renew",A,IDENTITY_A,token="a"*43)
        with sqlite3.connect(self.file) as db:
            stored=db.execute("SELECT token_digest FROM coordination_leases").fetchone()[0]
        self.assertEqual(len(stored),64)
        self.assertNotEqual(stored,value["lease_token"])
        self.now[0]=1100
        renewed=self.c.lease("renew",A,IDENTITY_A,token=value["lease_token"])
        self.assertEqual(renewed["expires"],1700)
        self.assertEqual(self.c.lease("status",A,IDENTITY_A)["state"],"leased")
        self.assertEqual(self.c.lease("release",A,IDENTITY_A,token=value["lease_token"])["state"],"released")
        self.assertEqual(self.c.lease("status",A,IDENTITY_A)["state"],"free")
        self.now[0]=1800
        with self.assertRaisesRegex(CoordinationFault,"LEASE_EXPIRED"):
            self.c.lease("release",A,IDENTITY_A,token=value["lease_token"])

    def test_expired_lease_does_not_release_unknown_claim(self):
        result=self.c.lease("acquire",A,IDENTITY_A)
        self.c.acquire("1"*32,IDENTITY_A,"receipt_a",[A],lease_token=result["lease_token"])
        self.c.dispatch_intent("1"*32)
        self.c.unknown("1"*32)
        self.now[0]=1700
        with self.assertRaisesRegex(CoordinationFault,"RESOURCE_BUSY"):
            self.c.lease("acquire",A,IDENTITY_A)
        self.assertEqual(self.c.status("1"*32)["state"],"unknown")

    def test_wrong_incarnation_cannot_use_token(self):
        value=self.c.lease("acquire",A,IDENTITY_A)
        with self.assertRaisesRegex(CoordinationFault,"LEASE_ARGUMENT_INVALID"):
            self.c.lease("release",A,IDENTITY_B,token=value["lease_token"])
        with self.assertRaisesRegex(CoordinationFault,"RESOURCE_BUSY"):
            self.c.acquire("9"*32,IDENTITY_B,"receipt_wrong",[A],
                           lease_token=value["lease_token"])
        with self.assertRaisesRegex(CoordinationFault,"RESOURCE_BUSY"):
            self.c.acquire("1"*32,IDENTITY_A,"receipt_a",[A])
        self.assertEqual(self.c.lease("release",A,IDENTITY_A,token=value["lease_token"])["state"],"released")

if __name__=="__main__":
    unittest.main()
