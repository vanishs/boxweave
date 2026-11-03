"""P2 exact same-host backup restore approval and key-boundary tests.

No secret content is ever passed to the gateway; no real restore is attempted.
"""
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from bwmini import restore_guard as guard


class RestoreGuardTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name)
        self.policy_dir = self.base / "etc"
        self.policy_dir.mkdir(mode=0o755)
        self.policy_dir.chmod(0o755)
        self.policy = self.policy_dir / "restore-policy.json"
        self.keydir = self.base / "keys"
        self.keydir.mkdir(mode=0o750)
        self.keydir.chmod(0o750)
        self.key = self.keydir / "box.key"
        self.key.write_text("# test fixture, not a real private key\nAGE-SECRET-KEY-1EXAMPLE\n")
        self.key.chmod(0o640)
        self.entry = {"box": "test", "backup": "test-box-20261103T170000Z",
                      "as": "test-restored", "key_id": "box.key"}
        self.args = {**self.entry, "confirm": self.entry["backup"]}
        self.write_policy([self.entry])

    def write_policy(self, entries):
        self.policy.write_text(json.dumps({"version": 1, "allowed_restores": entries}))
        self.policy.chmod(0o640)

    def call(self, args=None):
        return guard.approve(
            self.args if args is None else args,
            policy_path=self.policy, key_dir=self.keydir,
            owner=os.geteuid(), group=os.getegid())

    def test_authorized_exact_restore_only_returns_fixed_path(self):
        with patch.object(Path, "read_bytes", side_effect=AssertionError("read secret")):
            result = self.call()
        self.assertEqual(result, {"box": "test", "backup": self.entry["backup"],
                                  "as": "test-restored", "age_identity_file": str(self.key),
                                  "force": False})
        self.assertNotIn("confirm", result)
        self.assertNotIn("key_id", result)
        self.assertNotIn("AGE-SECRET-KEY", json.dumps(result))

    def test_exact_operator_approval_and_per_call_confirmation(self):
        for change in (
            {"box": "other"}, {"backup": "different-backup"},
            {"as": "another-target"}, {"key_id": "other.key"},
        ):
            bad = dict(self.args, **change)
            if "backup" in change:
                bad["confirm"] = bad["backup"]
            with self.assertRaisesRegex(ValueError, guard.DENIED):
                self.call(bad)
        with self.assertRaisesRegex(ValueError, guard.INVALID):
            self.call(dict(self.args, confirm="wrong"))
        with self.assertRaisesRegex(ValueError, guard.INVALID):
            self.call(dict(self.args, force=True))
        with self.assertRaisesRegex(ValueError, guard.INVALID):
            self.call(dict(self.args, age_identity_file="/etc/shadow"))
        with self.assertRaisesRegex(ValueError, guard.INVALID):
            self.call(dict(self.args, key_id="../../secret"))
        with self.assertRaisesRegex(ValueError, guard.INVALID):
            self.call(dict(self.args, as_="somewhere"))
        with self.assertRaisesRegex(ValueError, guard.INVALID):
            self.call({k: v for k, v in self.args.items() if k != "as"})

    def test_no_policy_and_foreign_or_unsafe_policy(self):
        self.policy.unlink()
        with self.assertRaisesRegex(ValueError, guard.UNSAFE):
            self.call()
        self.write_policy([self.entry])
        self.policy.chmod(0o666)
        with self.assertRaisesRegex(ValueError, guard.UNSAFE):
            self.call()
        self.policy.chmod(0o640)
        self.policy.unlink()
        self.policy.symlink_to(self.key)
        with self.assertRaisesRegex(ValueError, guard.UNSAFE):
            self.call()

    def test_key_symlink_hardlink_missing_and_world_readable(self):
        self.key.unlink()
        with self.assertRaisesRegex(ValueError, guard.UNSAFE):
            self.call()
        self.key.write_text("opaque fixture\n")
        self.key.chmod(0o640)
        self.key.unlink()
        self.key.symlink_to(self.policy)
        with self.assertRaisesRegex(ValueError, guard.UNSAFE):
            self.call()
        self.key.unlink()
        self.key.write_text("opaque fixture\n")
        self.key.chmod(0o644)
        with self.assertRaisesRegex(ValueError, guard.UNSAFE):
            self.call()
        self.key.chmod(0o640)
        os.link(self.key, self.base / "alias")
        with self.assertRaisesRegex(ValueError, guard.UNSAFE):
            self.call()

    def test_private_directory_and_malformed_policy_denied(self):
        self.keydir.chmod(0o777)
        with self.assertRaisesRegex(ValueError, guard.UNSAFE):
            self.call()
        self.keydir.chmod(0o750)
        self.write_policy([self.entry, self.entry])
        with self.assertRaisesRegex(ValueError, guard.UNSAFE):
            self.call()
        self.write_policy([{"box": "test", "backup": "not/allowed",
                            "as": "test-restored", "key_id": "box.key"}])
        with self.assertRaisesRegex(ValueError, guard.UNSAFE):
            self.call()

    def test_gateway_catalog_hides_host_path_and_requires_confirmation(self):
        upstream = {"name": "restore_box_backup", "description": "upstream", "inputSchema": {
            "type": "object", "properties": {
                "box": {"type": "string"}, "backup": {"type": "string"},
                "age_identity_file": {"type": "string"},
                "as": {"type": "string"}, "force": {"type": "boolean"}},
            "required": ["box", "backup", "age_identity_file"]}}
        adapted = guard.catalog_tool(upstream)
        self.assertNotIn("age_identity_file", adapted["inputSchema"]["properties"])
        self.assertEqual(set(adapted["inputSchema"]["required"]),
                         {"box", "backup", "as", "key_id", "confirm"})
        self.assertNotIn("key_id", upstream["inputSchema"]["properties"])
        self.assertFalse(adapted["inputSchema"]["additionalProperties"])
        self.assertIsNone(guard.catalog_tool({"name": "restore_box_backup",
                                               "inputSchema": {"properties": {}}}))

    def test_local_rejection_cannot_forward_any_upstream_request(self):
        import io
        from contextlib import redirect_stdout
        from bwmini import mcp_gate

        class KeepOpen(io.BytesIO):
            def close(self):
                pass
        class FakeProcess:
            def __init__(self):
                self.stdin = KeepOpen()
                self.stdout = KeepOpen()
            def poll(self):
                return 0

        fake = FakeProcess()
        request = {"jsonrpc": "2.0", "id": 8, "method": "tools/call",
                   "params": {"name": "restore_box_backup", "arguments": self.args}}
        with (patch.object(mcp_gate, "tool_policy", return_value=None),
              patch.object(mcp_gate.subprocess, "Popen", return_value=fake),
              patch.object(mcp_gate.restore_guard, "approve",
                           side_effect=ValueError(guard.DENIED)),
              patch.object(mcp_gate.sys, "stdin", io.StringIO(json.dumps(request) + "\n")),
              redirect_stdout(io.StringIO()) as out):
            mcp_gate.main()
        self.assertEqual(fake.stdin.getvalue(), b"")
        self.assertEqual(json.loads(out.getvalue())["error"]["message"], guard.DENIED)


if __name__ == "__main__":
    unittest.main()

