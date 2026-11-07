"""Policy and client boundaries on the standalone BoxWeave REST gateway."""
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from bwmini import gateway_rest, gateway_create, rest_wire
from bwmini.store import Store
from bwmini.remote_create import CreateJournal


class GatewayRESTTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.state = Store(self.root / "state" / "tasks.sqlite3")
        self.config = {"mode": "coordinated", "url": "http://127.0.0.1:18080",
                       "token_file": str(self.root / "secret"),
                       "state_db": str(self.state.path)}
        self.fake_claim = None

    def test_policy_read_credential_scope_and_fail_closed(self):
        policy = self.root / "lifecycle-rest.json"
        config = {"version":1,"enabled":True,"mode":"read",
                  "url":"http://127.0.0.1:18080","token_file":str(self.root / "secret")}
        policy.write_text(json.dumps(config))
        with patch.object(gateway_rest.os if hasattr(gateway_rest, "os") else __import__("os"), "geteuid", return_value=0):
            with self.assertRaises(gateway_rest.GatewayRESTFault):
                gateway_rest.policy(policy)
        # File is deliberately not root-owned in fixture; only a mocked
        # filesystem can select a fake root-owned policy. No live route.
        with self.assertRaises(gateway_rest.GatewayRESTFault):
            gateway_rest.execute_read(self.config,"create_container",{})

    def test_single_rest_create_and_cross_interface_duplicate(self):
        calls = []
        class Rest:
            def __init__(self, cfg):
                self.cfg = cfg
            def request(self, method, uri, body=None, timeout=30):
                if method == "GET":
                    raise rest_wire.WireFault("API_TARGET_NOT_FOUND")
                calls.append((method,uri,body))
                return {"container":{"username":"wip-unique",
                                     "state":"CONTAINER_STATE_CREATING"}}
        class Claims:
            def acquire_environment(self,*a): pass
            def dispatch_intent(self,*a): pass
            def unknown(self,*a): pass
        args={"username":"wip-unique","cpu":"1"}
        def add_key(x):
            return {**x, "ssh_keys":["ssh-ed25519 AAA"]}
        with patch("bwmini.rest_wire.Client", Rest), \
             patch("bwmini.coordination_bridge.published_scope",
                   return_value={"provider":"containarium","backend":"dev",
                                 "project":"default"}), \
             patch("bwmini.coordination_bridge.SharedClaims",Claims):
            one=gateway_create.dispatch(self.config,args,7,
                                        normalize=add_key,protected=True)
            self.assertEqual(one["state"],"accepted")
            same=gateway_create.dispatch(self.config,args,8,
                                         normalize=add_key,protected=True)
        self.assertEqual(one["task_id"],same["task_id"])
        self.assertEqual(len(calls),1)
        self.assertIs(calls[0][2]["async"],True)
        self.assertFalse(one["guest_user_and_ssh_verified"])

    def test_r04_web_name_claim_blocks_gateway_create_before_rest_post(self):
        from bwmini.coordination_bridge import SharedClaims
        from bwmini.coordination_runtime import private_connection
        from bwmini.operation_coordination import OperationCoordinator
        claims_file=self.root/"claims.sqlite3"
        claims_file.touch(mode=0o600)
        OperationCoordinator(
            lambda:private_connection(claims_file)).initialize()
        web=SharedClaims(claims_file)
        scope={"provider":"containarium","backend":"dev","project":"default"}
        calls=[]
        class Client:
            def __init__(self,config):pass
            def request(self,method,uri,body=None,timeout=30):
                calls.append(method)
                if method=="GET":
                    raise rest_wire.WireFault("API_TARGET_NOT_FOUND")
                raise AssertionError("REST write bypassed shared claim")
        with (patch("bwmini.rest_wire.Client",Client),
              patch("bwmini.coordination_bridge.published_scope",
                    return_value=scope),
              patch("bwmini.coordination_bridge.SharedClaims",
                    side_effect=lambda:SharedClaims(claims_file))):
            web.acquire_environment("e"*32,"environment_"+"e"*32,
                                    scope,"create","wip-unique",None,{})
            with self.assertRaisesRegex(ValueError,"REST_CREATE_PREDISPATCH_UNVERIFIED"):
                gateway_create.dispatch(
                    self.config,{"username":"wip-unique"},7,
                    normalize=lambda x:{**x,"ssh_keys":["ssh-ed25519 AAA"]},
                    protected=True)
        self.assertEqual(calls,["GET"])
        self.assertEqual(SharedClaims(claims_file).coordinator.status(
            "e"*32)["state"],"accepted")

    def test_post_response_loss_frozen_and_duplicate_is_not_retried(self):
        calls=[]
        class Broken:
            def __init__(self, cfg): pass
            def request(self,method,uri,body=None,timeout=30):
                if method == "GET":
                    raise rest_wire.WireFault("API_TARGET_NOT_FOUND")
                calls.append((method,uri))
                raise rest_wire.WireFault("API_OUTCOME_UNKNOWN")
        class Claims:
            def acquire_environment(self,*a):pass
            def dispatch_intent(self,*a):pass
            def unknown(self,*a):pass
        args={"username":"wip-unique"}
        key=lambda x: {**x,"ssh_keys":["ssh-ed25519 AAA"]}
        with patch("bwmini.rest_wire.Client",Broken), \
             patch("bwmini.coordination_bridge.published_scope",
                   return_value={"provider":"containarium","backend":"dev",
                                 "project":"default"}), \
             patch("bwmini.coordination_bridge.SharedClaims",Claims):
            one=gateway_create.dispatch(self.config,args,7,
                                        normalize=key,protected=True)
            other=gateway_create.dispatch(self.config,args,8,
                                          normalize=key,protected=True)
        self.assertEqual(one["state"],"unknown")
        self.assertEqual(other["task_id"],one["task_id"])
        self.assertEqual(len(calls),1)
