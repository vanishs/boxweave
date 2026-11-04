"""Pinned upstream host-hop fetch, bounded installation, and legacy upgrade tests.

All host filesystem writes are mocked into disposable temp paths. No Provider
version is upgraded, no upstream installer is run, and no host sudoers edited.
"""
import hashlib
import os
import socket
import stat
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

from bwmini import online


class ShellInstallTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.base = Path(self.tmp.name)
        self.host_dir = self.base / "bin"
        self.host_dir.mkdir()
        self.host_dir.chmod(0o755)
        self.target = self.host_dir / "containarium-shell"
        self.registry = self.base / "shells"
        self.registry.write_text("/bin/bash\n")
        self.registry.chmod(0o644)
        self.stage = self.base / "release"
        self.stage.mkdir()
        self.wrapper = b"#!/bin/bash\nexec /bin/true\n"
        self.source = (b"#!/usr/bin/env bash\n"
                       b"cat > \"$WRAPPER_SCRIPT\" << 'SHELLEOF'\n"
                       + self.wrapper +
                       b"SHELLEOF\n"
                       b"# Unrelated root setup must NEVER execute.\n"
                       b"ALL ALL=(root) NOPASSWD: incus exec *\n")
        self.state = {"managed_files": {}}
        self.stat_original = Path.stat
        self.atomic_calls = []
        self.download_calls = []
        self.version_output = "Containarium v0.128.0"
        self.syntax_exit = 0
        self.download_failure = None

    def fake_stat(self, path, *a, **kw):
        actual = self.stat_original(path, *a, **kw)
        if path in (self.host_dir, self.target, self.registry):
            return SimpleNamespace(st_uid=0, st_gid=0, st_mode=actual.st_mode)
        return actual

    def atomic(self, path, payload, mode=0o600):
        self.atomic_calls.append((path, mode))
        self.assertEqual(path, self.target)
        self.target.write_bytes(payload)
        self.target.chmod(mode)

    def fake_download(self, url, path, expected, limit=180_000_000):
        self.download_calls.append((url, expected, limit))
        self.assertEqual(url, online.HOST_SHELL_UPSTREAM_URL)
        self.assertEqual(expected, hashlib.sha256(self.source).hexdigest())
        if self.download_failure:
            raise self.download_failure
        path.write_bytes(self.source)

    def fake_run(self, argv, **kwargs):
        if argv[0] == "/bin/bash":
            self.assertEqual(argv[1], "-n")
            return SimpleNamespace(returncode=self.syntax_exit, stdout="", stderr="")
        self.assertEqual(argv, ["/opt/containariumd", "version"])
        return SimpleNamespace(returncode=0, stdout=self.version_output, stderr="")

    def invoke(self, *, external=False, repair=False, rooted=True,
               provider_cli="/opt/containariumd"):
        # Patching stat/atomic simulates root-owned paths WITHOUT touching
        # /usr/local/bin, system sshd, sudoers, or a real managed Provider.
        with patch.object(online, "HOST_SHELL", self.target), \
             patch.object(online, "HOST_SHELLS", self.registry), \
             patch.object(online, "HOST_SHELL_SHA256", hashlib.sha256(self.wrapper).hexdigest()), \
             patch.object(online, "HOST_SHELL_SOURCE_SHA256", hashlib.sha256(self.source).hexdigest()), \
             patch.object(online, "atomic", side_effect=self.atomic), \
             patch.object(online, "download", side_effect=self.fake_download), \
             patch.object(online.subprocess, "run", side_effect=self.fake_run), \
             patch.object(online, "trusted_provider_binary", return_value=True), \
             patch.object(Path, "stat", (lambda p, *a, **kw: self.fake_stat(p, *a, **kw)) if rooted else self.stat_original):
            return online.ensure_containarium_shell(
                self.stage, self.state, external=external, repair=repair,
                provider_cli=provider_cli
            )

    def test_upstream_source_pin_and_release_asset_exclusion(self):
        self.assertEqual(online.PROVIDER_VERSION, "0.128.0")
        self.assertEqual(online.MIN_PROVIDER_VERSION, "0.128.0")
        self.assertEqual(online.HOST_SHELL_UPSTREAM_COMMIT,
                         "e5deb167c973033d9a847abdcb2420e917095546")
        self.assertEqual(online.HOST_SHELL_SOURCE_SHA256,
                         "93365c56cd70aafac9880bc4dfcc4b1bcbb55dfc86e37b17b207c8245989f04a")
        self.assertEqual(online.HOST_SHELL_SHA256,
                         "2212651e2083b80d76363523bf794ec4d530f8eb74bbac0def0c347bb544caa6")
        self.assertNotIn("containarium-shell", online.RELEASE_FILES)
        self.assertFalse((Path(__file__).resolve().parents[1] / "bwmini/containarium-shell").exists())

    def test_extract_only_wrapper_not_full_upstream_setup(self):
        with patch.object(online, "HOST_SHELL_SHA256", hashlib.sha256(self.wrapper).hexdigest()), \
             patch.object(online, "HOST_SHELL_SOURCE_SHA256", hashlib.sha256(self.source).hexdigest()):
            result = online.extract_upstream_containarium_shell(self.source)
            self.assertEqual(result, self.wrapper)
            self.assertNotIn(b"NOPASSWD", result)
            with self.assertRaisesRegex(online.Failure, "SOURCE_PIN_MISMATCH"):
                online.extract_upstream_containarium_shell(self.source + b"changed")
            changed = self.source.replace(b"SHELLEOF", b"DRIFT")
            with patch.object(online, "HOST_SHELL_SOURCE_SHA256", hashlib.sha256(changed).hexdigest()):
                with self.assertRaisesRegex(online.Failure, "SOURCE_FORMAT_CHANGED"):
                    online.extract_upstream_containarium_shell(changed)

    def test_local_missing_auto_downloads_and_installs_once(self):
        self.assertEqual(self.invoke(), "installed_from_pinned_upstream_registered")
        self.assertEqual(self.target.read_bytes(), self.wrapper)
        self.assertEqual(self.target.stat().st_mode & 0o777, 0o755)
        self.assertEqual(self.state["managed_files"][str(self.target)],
                         hashlib.sha256(self.wrapper).hexdigest())
        self.assertEqual(self.registry.read_text(),
                         "/bin/bash\n" + str(self.target) + "\n")
        self.assertEqual(self.invoke(), "present_managed")
        self.assertEqual(len(self.download_calls), 1)
        self.assertEqual(len(self.atomic_calls), 1)

    def test_external_requires_explicit_opt_in(self):
        self.assertEqual(self.invoke(external=True), "missing_external_unmodified")
        self.assertFalse(self.target.exists())
        self.assertEqual(self.download_calls, [])
        self.assertEqual(self.invoke(external=True, repair=True),
                         "installed_from_pinned_upstream_registered")
        self.assertEqual(len(self.download_calls), 1)

    def test_external_unsupported_or_unknown_version_never_downloads(self):
        self.version_output = "Containarium v0.120.1"
        with self.assertRaisesRegex(online.Failure, "PROVIDER_COMPONENT_BELOW_MINIMUM"):
            self.invoke(external=True, repair=True)
        with self.assertRaisesRegex(online.Failure, "PROVIDER_VERSION_UNKNOWN"):
            self.invoke(external=True, repair=True, provider_cli=None)
        self.assertFalse(self.target.exists())
        self.assertEqual(self.download_calls, [])

    def test_future_runtime_version_does_not_block_verified_wrapper(self):
        self.version_output = "Containarium v0.140.1"
        self.assertEqual(self.invoke(external=True, repair=True),
                         "installed_from_pinned_upstream_registered")

    def test_existing_safe_wrapper_preserves_file_even_if_external(self):
        self.target.write_bytes(self.wrapper)
        self.target.chmod(0o755)
        self.registry.write_text("/bin/bash\n" + str(self.target) + "\n")
        self.assertEqual(self.invoke(), "present_external")
        self.assertEqual(self.invoke(external=True), "present_external")
        self.assertEqual(self.invoke(external=True, repair=True), "present_external")
        self.assertEqual(len(self.download_calls), 0)
        self.assertEqual(len(self.atomic_calls), 0)
        self.assertEqual(self.state["managed_files"], {})

    def test_external_unregistered_does_not_change_registry(self):
        self.target.write_bytes(self.wrapper)
        self.target.chmod(0o755)
        self.assertEqual(self.invoke(external=True), "unregistered_external_unmodified")
        self.assertEqual(self.registry.read_text(), "/bin/bash\n")

    def test_foreign_and_symlink_are_not_overwritten(self):
        self.target.write_text("foreign content\n")
        self.assertEqual(self.invoke(external=True), "foreign_external_unmodified")
        for external, repair in ((False, False), (True, True)):
            with self.assertRaisesRegex(online.Failure, "FOREIGN_CONTAINARIUM_SHELL_REFUSED"):
                self.invoke(external=external, repair=repair)
        self.target.unlink()
        self.target.symlink_to(self.base / "missing")
        self.assertEqual(self.invoke(external=True), "foreign_external_unmodified")
        with self.assertRaisesRegex(online.Failure, "FOREIGN_CONTAINARIUM_SHELL_REFUSED"):
            self.invoke()
        self.assertEqual(self.download_calls, [])

    def test_unsafe_registry_or_directory_stops_before_download(self):
        self.registry.chmod(0o666)
        with self.assertRaisesRegex(online.Failure, "HOST_SHELL_REGISTRY_UNSAFE"):
            self.invoke()
        self.registry.chmod(0o644)
        with self.assertRaisesRegex(online.Failure, "CONTAINARIUM_SHELL_PARENT_UNSAFE"):
            self.invoke(rooted=False)
        self.assertEqual(self.download_calls, [])

    def test_offline_download_or_bad_syntax_is_non_mutating(self):
        self.download_failure = OSError("no network")
        with self.assertRaises(OSError):
            self.invoke()
        self.assertFalse(self.target.exists())
        self.download_failure = None
        self.syntax_exit = 1
        with self.assertRaisesRegex(online.Failure, "CONTAINARIUM_SHELL_SYNTAX_INVALID"):
            self.invoke()
        self.assertFalse(self.target.exists())

    def test_legacy_managed_missing_allows_repair_but_drift_denied(self):
        expected = hashlib.sha256(self.wrapper).hexdigest()
        state = {"hostname": socket.gethostname(),
                 "managed_files": {str(self.target): expected}}
        with patch.object(online, "HOST_SHELL", self.target), \
             patch.object(online, "HOST_SHELL_SHA256", expected):
            online.check_managed(state)
            self.target.write_text("tampered\n")
            with self.assertRaisesRegex(online.Failure, "MANAGED_FILE_CHANGED"):
                online.check_managed(state)
            self.target.unlink()
            self.target.symlink_to(self.base / "missing")
            with self.assertRaisesRegex(online.Failure, "MANAGED_FILE_CHANGED"):
                online.check_managed(state)

    def test_optional_failure_does_not_block_core_installer(self):
        with patch.object(online, "ensure_containarium_shell",
                          side_effect=online.Failure("CONTAINARIUM_SHELL_PIN_MISMATCH")):
            result = online.optional_containarium_shell_setup(self.stage, self.state)
        self.assertEqual(result["status"], "incomplete")
        self.assertEqual(result["code"], "CONTAINARIUM_SHELL_PIN_MISMATCH")
        self.assertFalse(result["host_hop_ssh_verified"])
        with patch.object(online, "ensure_containarium_shell",
                          side_effect=OSError("sensitive network details")):
            result = online.optional_containarium_shell_setup(self.stage, self.state)
        self.assertEqual(result["code"], "CONTAINARIUM_SHELL_SETUP_FAILED")
        self.assertNotIn("sensitive", str(result))
        with patch.object(online, "ensure_containarium_shell",
                          return_value="missing_external_unmodified"):
            result = online.optional_containarium_shell_setup(self.stage, self.state)
        self.assertIsNone(result["code"])

    def test_versioned_cli_legacy_repair_flag_still_works(self):
        with patch.object(sys, "argv", ["online.py", "install", "--stage", "/tmp/stage",
                                        "--version", "0.42.10",
                                        "--repair-containarium-shell"]), \
             patch.object(online, "install") as installer:
            self.assertEqual(online.main(), 0)
            installer.assert_called_once_with(
                Path("/tmp/stage"), "0.42.10", None, None, None,
                repair_containarium_shell=True
            )


if __name__ == "__main__":
    unittest.main()

