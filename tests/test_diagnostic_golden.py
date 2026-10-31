"""Shared golden vectors for diagnostic aggregation across trusted origins.

The root-owned host CLI and the generated client script each embed their own
small aggregation logic; identical check sets must classify identically.
These vectors freeze that contract for both implementations.
"""
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from bwmini import host_cli


def c(cid, state, required=True):
    return host_cli.check(cid, state, "CODE_X", "fixture", required=required)


AGGREGATE_VECTORS = (
    ((), "incomplete", 2),
    ((c("a", "passed"),), "passed", 0),
    ((c("a", "failed"),), "failed", 1),
    ((c("a", "passed"), c("b", "passed")), "passed", 0),
    ((c("a", "passed"), c("b", "failed")), "failed", 1),
    ((c("a", "failed"), c("b", "not_verified")), "failed", 1),
    ((c("a", "passed"), c("b", "not_verified")), "incomplete", 2),
    ((c("a", "not_verified"), c("b", "not_checked")), "incomplete", 2),
    ((c("a", "passed"), c("b", "not_checked", required=False)), "passed", 0),
    ((c("a", "failed"), c("b", "not_checked", required=False)), "failed", 1),
    ((c("a", "not_checked"),), "incomplete", 2),
    ((c("a", "unexpected"),), "incomplete", 2),
)


class HostGoldenTests(unittest.TestCase):
    def test_host_cli_aggregation_matches_golden_vectors(self):
        for checks, overall, code in AGGREGATE_VECTORS:
            with self.subTest(overall=overall, size=len(checks)):
                self.assertEqual(host_cli.overall_state(list(checks)), overall)
                self.assertEqual(host_cli.exit_code_for(overall), code)

    def test_check_shape_is_fixed(self):
        self.assertEqual(host_cli.check("x", "passed", "C", "scope"),
                         {"id": "x", "state": "passed", "code": "C",
                          "scope": "scope", "required": True})
        self.assertEqual(host_cli.check("x", "failed", "C", "scope", required=False, detail="d")["detail"], "d")

    def test_unit_classification_definite_negative_only(self):
        healthy = {"LoadState": "loaded", "ActiveState": "active", "UnitFileState": "enabled"}
        self.assertEqual(host_cli.classify_unit(healthy, expect_enabled=True), ("passed", "UNIT_OBSERVED"))
        static = {"LoadState": "loaded", "ActiveState": "active", "UnitFileState": "static"}
        self.assertEqual(host_cli.classify_unit(static, expect_enabled=False), ("passed", "UNIT_OBSERVED"))
        self.assertEqual(host_cli.classify_unit(static, expect_enabled=True)[0], "not_verified")
        self.assertEqual(host_cli.classify_unit({"ActiveState": "inactive"}, True),
                         ("failed", "UNIT_OBSERVED_INACTIVE"))
        self.assertEqual(host_cli.classify_unit({"ActiveState": "failed"}, True),
                         ("failed", "UNIT_OBSERVED_FAILED"))
        self.assertEqual(host_cli.classify_unit({"state": "unavailable"}, True),
                         ("not_verified", "UNIT_STATE_UNAVAILABLE"))
        self.assertEqual(host_cli.classify_unit(None, True), ("not_verified", "UNIT_STATE_UNAVAILABLE"))
        self.assertEqual(host_cli.classify_unit({"LoadState": "not-found"}, True)[0], "not_verified")


def generated_server_key():
    import base64
    import struct
    data = struct.pack(">I", 11) + b"ssh-ed25519" + struct.pack(">I", 32) + b"x" * 32
    return "ssh-ed25519 " + base64.b64encode(data).decode("ascii")


def portable_namespace():
    """Exec one generated client script and return its namespace (main not run)."""
    from bwmini import client_profiles
    script = client_profiles.render({"app": "claude", "os": "linux", "transport": "native",
                                     "host": "example.test", "port": 22,
                                     "server_key": generated_server_key(),
                                     "admin_user": "example-admin", "replace": False})
    namespace = {"__name__": "portable_fixture"}
    exec(compile(script, "boxweave-setup.py", "exec"), namespace)
    return namespace


class PortableGoldenTests(unittest.TestCase):
    def test_generated_client_aggregation_matches_golden_vectors(self):
        namespace = portable_namespace()
        client_overall = namespace["overall_state"]
        for checks, overall, _ in AGGREGATE_VECTORS:
            with self.subTest(overall=overall, size=len(checks)):
                self.assertEqual(client_overall([dict(c) for c in checks]), overall)

    def test_generated_client_check_shape_matches_host(self):
        namespace = portable_namespace()
        self.assertEqual(namespace["dcheck"]("x", "passed", "C", "scope"),
                         host_cli.check("x", "passed", "C", "scope"))
        self.assertEqual(namespace["dcheck"]("x", "failed", "C", "scope", required=False, detail="d"),
                         host_cli.check("x", "failed", "C", "scope", required=False, detail="d"))

    def test_manual_profiles_stay_incomplete_with_passed_connection(self):
        import subprocess as real_subprocess
        from bwmini import client_profiles
        script = client_profiles.render({"app": "qoder", "os": "linux", "transport": "native",
                                         "host": "example.test", "port": 22,
                                         "server_key": generated_server_key(),
                                         "admin_user": "example-admin", "replace": False})
        namespace = {"__name__": "portable_fixture"}
        exec(compile(script, "boxweave-setup.py", "exec"), namespace)
        responses = "".join(json.dumps(m) + "\n" for m in (
            {"jsonrpc": "2.0", "id": 1, "result": {"protocolVersion": "2024-11-05", "capabilities": {}}},
            {"jsonrpc": "2.0", "id": 2, "result": {"tools": [{"name": "list_containers"}]}},
            {"jsonrpc": "2.0", "id": 3, "result": {"content": [{"type": "text", "text": "ok"}]}}))
        completed = real_subprocess.CompletedProcess([], 0, stdout=responses, stderr="")
        with tempfile.TemporaryDirectory() as tmp, \
             patch("pathlib.Path.home", return_value=Path(tmp)), \
             patch("subprocess.run", return_value=completed):
            root = Path(tmp) / ".config" / "boxweave"
            root.mkdir(parents=True, mode=0o700)
            (root / "id_ed25519").write_text("PRIVATE")
            (root / "known_hosts").write_text("example.test " + generated_server_key() + "\n")
            code, summary = namespace["run_diagnose"]("")
        by_id = {c["id"]: c for c in summary["checks"]}
        self.assertEqual(by_id["registration"]["state"], "not_verified")
        self.assertEqual(by_id["registration"]["code"], "REGISTRATION_MANUAL")
        self.assertTrue(by_id["registration"]["required"])
        self.assertEqual(summary["connection_overall"], "passed")
        self.assertEqual(summary["overall"], "incomplete")
        self.assertEqual(code, 2)


if __name__ == "__main__":
    unittest.main()


if __name__ == "__main__":
    unittest.main()
