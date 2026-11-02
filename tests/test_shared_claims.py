"""The Web actor and SSH/HTTP gateway contend on the same durable Box claim."""
import json
from pathlib import Path
import sqlite3
import tempfile
import threading
import unittest
from unittest.mock import patch

from bwmini.domain import Target
from bwmini.operation_coordination import CoordinationFault, OperationCoordinator
from bwmini.coordination_runtime import private_connection, scope_digest
from bwmini.coordination_bridge import SharedClaims, app_box, gateway_box, checked_scope, published_scope

SCOPE={"provider":"containarium","backend":"dev","project":"default"}
A=Target("containarium","dev","default","box-a",
         "box-a-container","123","box-a","box-a-container")
B=Target("containarium","dev","default","box-b",
         "box-b-container","456","box-b","box-b-container")

def observed(target):
    return {"name":target.instance,"createdAt":target.created,
            "username":target.user}

class SharedClaimsTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.path=Path(self.tmp.name)/"claims.sqlite3"
        self.path.touch(mode=0o600)
        OperationCoordinator(lambda:private_connection(self.path)).initialize()
        self.now=[1000.]
        self.web=SharedClaims(self.path,clock=lambda:self.now[0])
        self.gateway=SharedClaims(self.path,clock=lambda:self.now[0])

    def assert_code(self,code,cb):
        with self.assertRaises(CoordinationFault) as e:
            cb()
        self.assertEqual(e.exception.code,code)

    def test_app_and_gateway_same_identity_and_scope(self):
        self.assertEqual(app_box(A,SCOPE),gateway_box(observed(A),SCOPE,A.logical))
        ia,sa,la=app_box(A,SCOPE)
        ib,sb,lb=app_box(B,SCOPE)
        self.assertNotEqual(ia,ib)
        self.assertEqual(sa,sb)
        self.assertNotEqual(la,lb)

    def test_racing_web_gateway_one_box_only_one_writer(self):
        id_,scope,logical=app_box(A,SCOPE)
        outcomes=[]
        start=threading.Barrier(3)
        lock=threading.Lock()
        def contender(which):
            start.wait()
            try:
                (self.web if which==1 else self.gateway).acquire(
                    str(which)*32, "web_"+str(which),id_,scope,logical)
                result="accepted"
            except CoordinationFault as e:
                result=e.code
            with lock:
                outcomes.append(result)
        threads=[threading.Thread(target=contender,args=(n,)) for n in (1,2)]
        for thread in threads:thread.start()
        start.wait()
        for thread in threads:
            thread.join(timeout=6)
            self.assertFalse(thread.is_alive())
        self.assertCountEqual(outcomes,["accepted","RESOURCE_BUSY"])

    def test_web_gateway_different_boxes_concurrent(self):
        for i,target in enumerate((A,B)):
            identity,scope,logical=app_box(target,SCOPE)
            self.web.acquire(str(i+1)*32,"web_"+str(i),identity,scope,logical)
        self.assertEqual(self.gateway.coordinator.status("1"*32)["state"],"accepted")
        self.assertEqual(self.gateway.coordinator.status("2"*32)["state"],"accepted")

    def test_crash_unknown_and_lease_expiry_cannot_steal(self):
        identity,scope,logical=gateway_box(observed(A),SCOPE,A.logical)
        receipt="gateway_"+("a"*32)
        leased=self.gateway.lease("acquire",identity,seconds=1)
        self.gateway.acquire("1"*32,receipt,identity,scope,logical,leased["lease_token"])
        self.gateway.dispatch_intent("1"*32)
        self.gateway.unknown("1"*32)
        self.now[0]=1010.
        self.assert_code("RECOVERY_REQUIRED",lambda:self.web.acquire(
            "2"*32,"web_2",identity,scope,logical))
        self.gateway.observed_terminal("1"*32,receipt)
        self.web.acquire("2"*32,"web_2",identity,scope,logical)

    def test_different_version_or_instance_refuses_same_claim(self):
        identity,scope,logical=app_box(A,SCOPE)
        self.web.acquire("1"*32,"web_a",identity,scope,logical)
        replaced=Target(**{**A.data(),"created":"987"})
        other,scope,logical=app_box(replaced,SCOPE)
        self.assertNotEqual(other,identity)
        self.assert_code("RESOURCE_BUSY",lambda:self.gateway.acquire(
            "2"*32,"mcp_b",other,scope,logical))

    def test_mismatched_scope_denied(self):
        self.assert_code("COORDINATION_SCOPE_MISMATCH",lambda:app_box(
            A,{"provider":"containarium","backend":"other","project":"default"}))
        self.assert_code("COORDINATION_SCOPE_UNVERIFIED",lambda:checked_scope(
            {"backend":"dev","project":"default"}))
        self.assert_code("COORDINATION_IDENTITY_INVALID",lambda:gateway_box(
            {"username":"box-a","name":"box-a-container","createdAt":None},SCOPE,A.logical))

    def test_unprovisioned_shared_claims_fail_closed(self):
        self.path.unlink()
        identity,scope,logical=app_box(A,SCOPE)
        self.assert_code("COORDINATION_UNAVAILABLE",lambda:self.web.acquire(
            "1"*32,"web_1",identity,scope,logical))
        self.assertFalse(self.path.exists())

    def test_public_scope_requires_root_published_file(self):
        scope_file=Path(self.tmp.name)/"targets.json"
        scope_file.write_text(json.dumps({"observed_at":100.,"scope":SCOPE,"targets":{}}))
        self.assert_code("COORDINATION_SCOPE_UNVERIFIED",
                         lambda:published_scope(scope_file,now=lambda:110.))
        # The trusted root writer is tested through a bounded synthetic
        # ownership fixture, not by widening /tmp or changing host privileges.
        original=Path.lstat
        def root_owned(path):
            stat=original(path)
            if path in (scope_file,scope_file.parent):
                from types import SimpleNamespace
                return SimpleNamespace(st_uid=0,st_mode=stat.st_mode,
                                       st_size=stat.st_size,st_nlink=stat.st_nlink)
            return stat
        scope_file.parent.chmod(0o755)
        scope_file.chmod(0o644)
        with patch.object(Path,"lstat",root_owned):
            self.assertEqual(published_scope(scope_file,now=lambda:110.),SCOPE)
            self.assert_code("COORDINATION_SCOPE_UNVERIFIED",
                             lambda:published_scope(scope_file,now=lambda:200.))
