"""Containarium minimum-version and forward compatibility regression.

No test executes a real upstream binary or modifies the host.
"""
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

from bwmini import online, deploy_check
from bwmini.domain import Fault


class RuntimeVersionTests(unittest.TestCase):
    def version(self, text, *, exit_code=0):
        with patch.object(online, "trusted_provider_binary", return_value=True), \
             patch.object(online.subprocess, "run",
                          return_value=SimpleNamespace(returncode=exit_code,
                                                       stdout=text, stderr="")):
            return online.component_version("/usr/local/bin/containariumd")

    def test_inclusive_minimum_and_open_ended_future(self):
        for text, expected in (("Containarium v0.128.0", "0.128.0"),
                               ("Containarium v0.129.0", "0.129.0"),
                               ("Containarium v0.300.42", "0.300.42"),
                               ("Containarium v1.0.0", "1.0.0")):
            with self.subTest(text=text):
                self.assertEqual(self.version(text), expected)

    def test_old_and_ambiguous_version_refused(self):
        for value in ("Containarium v0.120.1", "Containarium v0.127.999"):
            with self.subTest(value=value), self.assertRaisesRegex(
                    online.Failure, "PROVIDER_COMPONENT_BELOW_MINIMUM"):
                self.version(value)
        for value in ("unknown", "Containarium dev", "v0.129.0 and v0.130.0"):
            with self.subTest(value=value), self.assertRaisesRegex(
                    online.Failure, "PROVIDER_COMPONENT_VERSION_UNKNOWN"):
                self.version(value)
        with self.assertRaisesRegex(online.Failure, "PROVIDER_COMPONENT_VERSION_UNKNOWN"):
            self.version("Containarium v0.128.0", exit_code=1)

    def test_untrusted_runtime_denied_before_execution(self):
        with patch.object(online, "trusted_provider_binary", return_value=False), \
             patch.object(online.subprocess, "run") as run:
            with self.assertRaisesRegex(online.Failure, "PROVIDER_COMPONENT_NOT_TRUSTED"):
                online.component_version("/usr/local/bin/containariumd")
            run.assert_not_called()

    def test_toolchain_is_advisory_not_mutating(self):
        with tempfile.TemporaryDirectory() as d:
            with patch.object(online.Path, "is_dir", return_value=False):
                self.assertEqual(online.toolchain_preflight("0.128.0"),
                                 "missing_for_build_tools")


class MinimumMCPTests(unittest.TestCase):
    def probe(self, reported):
        class FakeMCP:
            def __init__(self, *_args, **_kwargs):
                self.server = {"serverInfo": {"name": "containarium-mcp-server",
                                              "version": reported},
                               "protocolVersion": "2024-11-05"}
                self.calls = []
            def __enter__(self): return self
            def __exit__(self, *_): return False
            def discover(self): return {"list_containers": {}}
            def tool(self, name, args): self.calls.append((name, args))
        p = SimpleNamespace(argv=["/approved/launcher"], rest=None)
        with patch.object(deploy_check, "Provider", return_value=p), \
             patch.object(deploy_check, "Observed", FakeMCP):
            return deploy_check.read_provider({"mode": "mcp"})

    def test_mcp_minimum_and_future_versions(self):
        for version in ("0.128.0", "0.150.0", "1.0.0"):
            with self.subTest(version=version):
                result=self.probe(version)
                self.assertEqual(result["state"], "authenticated_read_passed")
                self.assertEqual(result["minimum_mcp_server"], "0.128.0")

    def test_mcp_unknown_and_old_refused(self):
        for version in ("", "0.127.0", "dev"):
            with self.subTest(version=version), self.assertRaises(Fault) as cm:
                self.probe(version)
            self.assertEqual(cm.exception.code, "MCP_SERVER_BELOW_MINIMUM_OR_UNKNOWN")


if __name__ == "__main__":
    unittest.main()
