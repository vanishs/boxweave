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


if __name__ == "__main__":
    unittest.main()
