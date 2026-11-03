"""Host-hop shell packaging and guarded remediation: isolated, non-root contracts."""
import hashlib
import os
import stat
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

from bwmini import online


ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / "bwmini" / "containarium-shell"


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
        (self.stage / "containarium-shell").write_bytes(SOURCE.read_bytes())
        self.state = {"managed_files": {}}
        self.original_stat = Path.stat
        self.atomic_calls = []

    def fake_stat(self, path, *args, **kwargs):
        result = self.original_stat(path, *args, **kwargs)
        if path in (self.host_dir, self.target, self.registry):
            return SimpleNamespace(st_uid=0, st_gid=0, st_mode=result.st_mode)
        return result

    def atomic(self, path, payload, mode=0o600):
        self.atomic_calls.append((path, mode))
        self.assertEqual(path, self.target)
        self.target.write_bytes(payload)
        self.target.chmod(mode)

    def invoke(self, *, external=False, repair=False, rooted=True):
        # Stand in for root-owned filesystem metadata WITHOUT requiring
        # privileged host access in unit tests.
        with patch.object(online, "HOST_SHELL", self.target), \
             patch.object(online, "HOST_SHELLS", self.registry), \
             patch.object(online, "atomic", side_effect=self.atomic), \
             patch.object(Path, "stat", (lambda path, *a, **kw: self.fake_stat(path, *a, **kw)) if rooted else self.original_stat):
            return online.ensure_containarium_shell(
                self.stage, self.state, external=external, repair=repair
            )

    def test_pinned_wrapper_matches_official_v01201_and_parses(self):
        data = SOURCE.read_bytes()
        self.assertEqual(len(data.splitlines()), 106)
        self.assertEqual(hashlib.sha256(data).hexdigest(), online.HOST_SHELL_SHA256)
        self.assertEqual(subprocess.run(["bash", "-n", str(SOURCE)],
                                        capture_output=True, check=False).returncode, 0)
        self.assertIn(b"SSH_ORIGINAL_COMMAND", data)
        self.assertIn(b"sudo incus exec", data)
        self.assertNotIn(b"ALL ALL=(root)", data)
        self.assertNotIn(b"sshd_config", data)

    def test_external_missing_is_read_only(self):
        status = self.invoke(external=True)
        self.assertEqual(status, "missing_external_unmodified")
        self.assertFalse(self.target.exists())
        self.assertEqual(self.atomic_calls, [])
        self.assertEqual(self.state["managed_files"], {})

    def test_external_matching_shell_is_preserved_without_claiming_ownership(self):
        self.target.write_bytes(SOURCE.read_bytes())
        self.target.chmod(0o755)
        self.registry.write_text("/bin/bash\n" + str(self.target) + "\n")
        self.assertEqual(self.invoke(external=True), "present_external")
        self.assertEqual(self.invoke(external=True, repair=True), "present_external")
        self.assertEqual(self.atomic_calls, [])
        self.assertEqual(self.state["managed_files"], {})

    def test_external_unregistered_and_foreign_are_observed_not_mutated(self):
        self.target.write_bytes(SOURCE.read_bytes())
        self.target.chmod(0o755)
        self.assertEqual(self.invoke(external=True), "unregistered_external_unmodified")
        self.assertNotIn(str(self.target), self.registry.read_text())
        self.target.write_text("foreign binary\n")
        self.assertEqual(self.invoke(external=True), "foreign_external_unmodified")
        self.assertEqual(self.atomic_calls, [])

    def test_local_install_adds_only_missing_pinned_wrapper_and_shell_entry(self):
        self.assertEqual(self.invoke(), "installed_registered")
        self.assertEqual(self.target.read_bytes(), SOURCE.read_bytes())
        self.assertEqual(self.target.stat().st_mode & 0o777, 0o755)
        self.assertEqual(self.state["managed_files"], {str(self.target): online.HOST_SHELL_SHA256})
        self.assertEqual(self.registry.read_text(), "/bin/bash\n" + str(self.target) + "\n")
        self.assertEqual(self.invoke(), "present_managed")
        self.assertEqual(len(self.atomic_calls), 1)
        self.assertEqual(self.registry.read_text().count(str(self.target)), 1)

    def test_explicit_external_repair_is_identical_but_not_automatic(self):
        self.assertEqual(self.invoke(external=True, repair=True), "installed_registered")
        self.assertTrue(self.target.exists())
        self.assertEqual(len(self.atomic_calls), 1)

    def test_exact_existing_local_shell_keeps_external_provenance(self):
        self.target.write_bytes(SOURCE.read_bytes())
        self.target.chmod(0o755)
        self.registry.write_text("/bin/bash\n" + str(self.target) + "\n")
        self.assertEqual(self.invoke(), "present_external")
        self.assertEqual(self.state["managed_files"], {})
        self.assertEqual(self.atomic_calls, [])

    def test_existing_wrapper_without_world_execute_is_not_accepted(self):
        self.target.write_bytes(SOURCE.read_bytes())
        self.target.chmod(0o700)
        self.assertEqual(self.invoke(external=True), "foreign_external_unmodified")
        with self.assertRaisesRegex(online.Failure, "FOREIGN_CONTAINARIUM_SHELL_REFUSED"):
            self.invoke()
        self.assertEqual(self.atomic_calls, [])

    def test_foreign_wrapper_or_symlink_fails_closed_on_enabled_paths(self):
        self.target.write_text("foreign shell\n")
        for mode in ((False, False), (True, True)):
            with self.subTest(mode=mode), self.assertRaisesRegex(
                online.Failure, "FOREIGN_CONTAINARIUM_SHELL_REFUSED"
            ):
                self.invoke(external=mode[0], repair=mode[1])
        self.target.unlink()
        self.target.symlink_to(self.base / "not-existing")
        with self.assertRaisesRegex(online.Failure, "FOREIGN_CONTAINARIUM_SHELL_REFUSED"):
            self.invoke()
        self.assertEqual(self.invoke(external=True), "foreign_external_unmodified")
        self.assertEqual(self.atomic_calls, [])

    def test_untrusted_source_hash_fails_before_writing_even_external(self):
        (self.stage / "containarium-shell").write_bytes(b"#!/bin/sh\nexit 0\n")
        for external in (False, True):
            with self.subTest(external=external), self.assertRaisesRegex(
                online.Failure, "CONTAINARIUM_SHELL_PIN_MISMATCH"
            ):
                self.invoke(external=external)
        self.assertEqual(self.atomic_calls, [])

    def test_registry_missing_or_untrusted_refuses_only_mutating_paths(self):
        self.registry.unlink()
        self.assertEqual(self.invoke(external=True), "missing_external_unmodified")
        with self.assertRaisesRegex(online.Failure, "HOST_SHELL_REGISTRY_UNAVAILABLE"):
            self.invoke()
        self.registry.write_text("/bin/bash\n")
        self.registry.chmod(0o666)
        self.assertEqual(self.invoke(external=True), "missing_external_unmodified")
        with self.assertRaisesRegex(online.Failure, "HOST_SHELL_REGISTRY_UNSAFE"):
            self.invoke()
        self.assertEqual(self.atomic_calls, [])

    def test_shell_syntax_failure_stops_before_host_write(self):
        with patch.object(online.subprocess, "run", return_value=SimpleNamespace(returncode=1)):
            with self.assertRaisesRegex(online.Failure, "CONTAINARIUM_SHELL_SYNTAX_INVALID"):
                self.invoke()
        self.assertEqual(self.atomic_calls, [])

    def test_versioned_cli_passes_explicit_repair_flag(self):
        with patch.object(sys, "argv", ["online.py", "install", "--stage", "/tmp/stage",
                                        "--version", "0.42.9",
                                        "--repair-containarium-shell"]), \
             patch.object(online, "install") as install:
            self.assertEqual(online.main(), 0)
            install.assert_called_once_with(Path("/tmp/stage"), "0.42.9", None,
                                            None, None, repair_containarium_shell=True)

        with patch.object(sys, "argv", ["online.py", "install", "--stage", "/tmp/stage",
                                        "--version", "0.42.9"]), \
             patch.object(online, "install") as install:
            self.assertEqual(online.main(), 0)
            install.assert_called_once_with(Path("/tmp/stage"), "0.42.9", None,
                                            None, None, repair_containarium_shell=False)

    def test_bad_directory_ownership_fails_closed(self):
        with self.assertRaisesRegex(online.Failure, "CONTAINARIUM_SHELL_PARENT_UNSAFE"):
            self.invoke(rooted=False)
        self.assertEqual(self.atomic_calls, [])


if __name__ == "__main__":
    unittest.main()

