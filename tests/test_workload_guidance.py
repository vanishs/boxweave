"""Server-owned workload public key and minimal Windows client fixtures."""
import base64
import struct
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch
from bwmini import client_setup, mcp_gate
from test_client_setup import ready_state


def key(letter):
    blob = struct.pack(">I", 11) + b"ssh-ed25519" + struct.pack(">I", 32) + letter * 32
    return "ssh-ed25519 " + base64.b64encode(blob).decode()


class WorkloadIdentityTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        # Gateway fixture expectations must not depend on the host's
        # root-installed protected-mode policy. Individual protection
        # tests explicitly override this default where appropriate.
        base = patch.object(mcp_gate.gateway_protection, "enabled", return_value=False)
        base.start()
        cls.addClassCleanup(base.stop)

    def test_generated_client_leaves_codex_agents_untouched(self):
        script=client_setup.render(ready_state(),"example-admin")
        for forbidden in ("AGENTS.override.md", "WslGuidance",
                          "ManagedWorkloadPublicKey", "Merge-BoxWeaveGuidance"):
            self.assertNotIn(forbidden,script)
        self.assertTrue(script.startswith("#!/usr/bin/env python3\n# ==="))
        self.assertIn("# No BoxWeave project AGENTS.md files are created or modified.",script)
        self.assertIn('HOST = SETTINGS["host"]',script)
        self.assertIn('PORT = SETTINGS["port"]',script)
        self.assertIn("StrictHostKeyChecking=yes",script)

    def test_managed_key_is_host_specific_and_idempotent(self):
        with tempfile.TemporaryDirectory() as tmp:
            path=Path(tmp)/"id_ed25519.pub"
            path.write_text(key(b"a")+"\n")
            path.chmod(0o644)  # Avoid test outcome depending on developer umask.
            with patch.object(mcp_gate,"MANAGED_PUBLIC_KEY_FILE",path):
                first=mcp_gate.managed_create_arguments({"username":"box1"})
                self.assertEqual(first["ssh_keys"],[key(b"a")])
                extra=mcp_gate.managed_create_arguments({"ssh_keys":[key(b"b")]})
                self.assertEqual(extra["ssh_keys"],[key(b"b"),key(b"a")])
                self.assertEqual(mcp_gate.managed_create_arguments(extra),extra)

    def test_invalid_resize_rejected_before_upstream_write(self):
        import io, json
        from types import SimpleNamespace
        class KeepBytes(io.BytesIO):
            def close(self): pass
        request={"jsonrpc":"2.0","id":13,"method":"tools/call",
                 "params":{"name":"resize_container",
                           "arguments":{"username":"fixture","cpu":"invalid-cpu"}}}
        process=SimpleNamespace(stdin=KeepBytes(),stdout=io.BytesIO(),poll=lambda:0)
        output=io.StringIO()
        with (patch.object(mcp_gate.subprocess,"Popen",return_value=process),
              patch.object(mcp_gate,"tool_policy",return_value=None),
              patch.object(mcp_gate.gateway_rest,"policy",
                           return_value={"mode":"coordinated"}),
              patch.object(mcp_gate.rest_mutations,"dispatch") as dispatch,
              patch.object(mcp_gate.sys,"stdin",io.StringIO(json.dumps(request)+"\n")),
              patch.object(mcp_gate.sys,"stdout",output),
              patch.object(mcp_gate.gd,"record")):
            mcp_gate.main()
        result=json.loads(output.getvalue())
        self.assertEqual(result["error"]["code"],-32602)
        self.assertEqual(result["error"]["message"],"RESOURCE_CPU_INVALID")
        dispatch.assert_not_called()
        self.assertEqual(process.stdin.getvalue(),b"")

    def test_resize_cpu_compatible_syntax_and_safe_registry_error(self):
        for cpu in ("3","0.25","1.5","250m","0-3","0,2-4"):
            self.assertEqual(mcp_gate.validated_resize_arguments({"cpu":cpu})["cpu"],cpu)
        for cpu in ("invalid-cpu","0","0m","-1","2-1","0,3-2",{},True):
            with self.subTest(cpu=cpu), self.assertRaisesRegex(ValueError,"RESOURCE_CPU_INVALID"):
                mcp_gate.validated_resize_arguments({"cpu":cpu})
        with tempfile.TemporaryDirectory() as tmp, patch.object(mcp_gate,"REGISTRY",Path(tmp)/"missing"):
            with self.assertRaisesRegex(ValueError,"TRUST_REGISTRY_UNAVAILABLE"):
                mcp_gate.trusted_registry()

    def test_disk_resize_preflight_does_not_dispatch_shrink_or_unknown(self):
        from bwmini import resource_size
        from bwmini.provider_rest import RESTClient
        from bwmini.domain import Fault
        for requested,current,expected in (
                ("24GB","25GB","RESOURCE_DISK_SHRINK_REFUSED"),
                ("xGB","25GB","RESOURCE_DISK_SIZE_UNVERIFIED"),
                ("25GB","n/a","RESOURCE_DISK_SIZE_UNVERIFIED"),
                ("1GiB","2GB","RESOURCE_DISK_SHRINK_REFUSED")):
            with self.subTest(requested=requested,current=current):
                with self.assertRaisesRegex(ValueError,expected):
                    resource_size.require_nonshrinking_disk(requested,current)
        for requested,current in (("25GB","25GB"),("26GB","25GB"),
                                  ("2048MiB","2GiB")):
            resource_size.require_nonshrinking_disk(requested,current)

        for requested,current,expected,expected_writes in (
                ("24GB","25GB","RESOURCE_DISK_SHRINK_REFUSED",0),
                ("bad","25GB","RESOURCE_DISK_SIZE_UNVERIFIED",0),
                ("30GB","unknown","RESOURCE_DISK_SIZE_UNVERIFIED",0),
                ("25GB","25GB",None,1),
                ("26GB","25GB",None,1)):
            with self.subTest(requested=requested,current=current):
                rest=RESTClient({"url":"http://127.0.0.1:18080",
                                 "token_file":"/tmp/unused"})
                calls=[]
                rest.container=lambda logical: {"username":logical,
                                                 "resources":{"disk":current}}
                def request(method,route,body=None,timeout=30):
                    calls.append((method,route,body))
                    return {"state":"accepted"}
                rest.request=request
                if expected:
                    with self.assertRaisesRegex(Fault,expected):
                        rest.mutate("resize","fixture",
                                    {"_resize":{"disk":requested}},60)
                else:
                    rest.mutate("resize","fixture",
                                {"_resize":{"disk":requested}},60)
                self.assertEqual(len(calls),expected_writes)
                if calls:
                    self.assertEqual(calls[0],("PUT","/v1/containers/fixture/resize",
                                              {"disk":requested}))

    def test_specific_safe_trust_codes(self):
        registry={"observed_at":0,"targets":{}}
        with patch.object(mcp_gate.time,"time",return_value=100):
            with self.assertRaisesRegex(ValueError,"TRUST_OBSERVATION_STALE"):
                mcp_gate.trusted({"box":"fixture"},registry,{})
        registry={"observed_at":100,"targets":{}}
        with patch.object(mcp_gate.time,"time",return_value=100):
            with self.assertRaisesRegex(ValueError,"TRUST_NOT_READY"):
                mcp_gate.trusted({"box":"fixture"},registry,{})


if __name__ == "__main__":
    unittest.main()
