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
                 "params":{"name":"resize_container","arguments":{"username":"fixture","cpu":"invalid-cpu"}}}
        process=SimpleNamespace(stdin=KeepBytes(),stdout=io.BytesIO(),poll=lambda:0)
        output=io.StringIO()
        with (patch.object(mcp_gate.subprocess,"Popen",return_value=process),
              patch.object(mcp_gate,"tool_policy",return_value=None),
              patch.object(mcp_gate.sys,"stdin",io.StringIO(json.dumps(request)+"\n")),
              patch.object(mcp_gate.sys,"stdout",output),
              patch.object(mcp_gate.gd,"record")):
            mcp_gate.main()
        result=json.loads(output.getvalue())
        self.assertEqual(result["error"]["code"],-32602)
        self.assertEqual(result["error"]["message"],"RESOURCE_CPU_INVALID")
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
