#!/usr/bin/env python3
"""Offline negative checks for the v1-only Release manifest and installer."""
import importlib.util
import hashlib
import subprocess
from pathlib import Path
import struct
import tempfile
import unittest


ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location("v1_release_manifest", ROOT / "scripts/v1_release_manifest.py")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.dir = Path(self.temp.name)
        self.installer = self.dir / "source-installer.sh"
        self.installer.write_text("#!/bin/sh\n")
        for arch, machine in module.MACHINES.items():
            blob = bytearray(64)
            blob[0:6] = b"\x7fELF\x02\x01"
            struct.pack_into("<H", blob, 18, machine)
            (self.dir / ("boxweave-v1.0.0-rc.1-linux-" + arch)).write_bytes(blob)

    def test_good_manifest_and_integrity(self):
        m = module.assemble(self.dir, "v1.0.0-rc.1", "a" * 40, self.installer)
        self.assertEqual(m["product"], "boxweave-v1")
        lines = (self.dir / "SHA256SUMS").read_text().splitlines()
        self.assertEqual(len(lines), 4)
        self.assertEqual(len(m["assets"]), 2)
        self.assertEqual(m["dependencies"]["go_toolchain"], "1.26.9")
        self.assertTrue(m["dependencies"]["incus_go_sdk"].startswith("v7."))

    def test_wrong_machine_fails(self):
        arm = self.dir / "boxweave-v1.0.0-rc.1-linux-arm64"
        arm.write_bytes((self.dir / "boxweave-v1.0.0-rc.1-linux-amd64").read_bytes())
        with self.assertRaisesRegex(ValueError, "machine mismatch"):
            module.assemble(self.dir, "v1.0.0-rc.1", "b" * 40, self.installer)

    def test_reject_main_or_invalid_tag(self):
        with self.assertRaisesRegex(ValueError, "invalid v1"):
            module.assemble(self.dir, "v0.50.5", "b" * 40, self.installer)

    def test_external_installer_plan_and_wrong_digest(self):
        path = self.dir / "boxweave-v1.0.0-rc.1-linux-amd64"
        valid_sha = hashlib.sha256(path.read_bytes()).hexdigest()
        base = ["sh", str(ROOT / "install.sh"), "--local-binary", str(path),
                "--project", "user-1001", "--sha256"]
        good = subprocess.run(base + [valid_sha, "--plan"], text=True, capture_output=True)
        self.assertEqual(good.returncode, 0, good.stderr)
        self.assertIn("PLAN_ONLY: no host changes", good.stdout)
        bad = subprocess.run(base + ["0" * 64, "--plan"], text=True, capture_output=True)
        self.assertNotEqual(bad.returncode, 0)
        self.assertIn("SHA256 mismatch", bad.stderr)

    def test_external_installer_rejects_legacy_tag_and_unapproved_apply(self):
        bad_tag = subprocess.run(["sh", str(ROOT / "install.sh"), "--version", "v0.50.5",
                                  "--plan"], text=True, capture_output=True)
        self.assertNotEqual(bad_tag.returncode, 0)
        self.assertIn("expected v1", bad_tag.stderr)
        path = self.dir / "boxweave-v1.0.0-rc.1-linux-amd64"
        valid_sha = hashlib.sha256(path.read_bytes()).hexdigest()
        denied = subprocess.run([
            "sh", str(ROOT / "install.sh"), "--local-binary", str(path),
            "--sha256", valid_sha, "--project", "user-1001",
            "--apply", "--confirm-host-sha256", "0" * 64,
        ], text=True, capture_output=True)
        self.assertNotEqual(denied.returncode, 0)
        self.assertIn("exact current --confirm-host-sha256", denied.stderr)

    def test_rc_publishing_requires_explicit_one_shot_branch(self):
        workflow = (ROOT / ".github/workflows/v1-release.yml").read_text()
        self.assertIn('"release/v1.*-rc.*"', workflow)
        self.assertIn("refs/heads/release/v1.", workflow)
        self.assertNotIn("workflow_dispatch", workflow)
        self.assertIn("v1-prerelease-native-S2-isolated-runner", workflow)
        self.assertIn("v1-prerelease-nested-S2.5-isolated-runner", workflow)
        self.assertIn("needs: [source, linux, native-s2-gate, native-s25-gate, candidate]", workflow)
        s2 = workflow.split("\n  native-s2-gate:\n", 1)[1].split("\n  native-s25-gate:\n", 1)[0]
        s25 = workflow.split("\n  native-s25-gate:\n", 1)[1].split("\n  candidate:\n", 1)[0]
        self.assertIn("sh scripts/accept_s2_native.sh", s2)
        self.assertNotIn("sh scripts/s25_bridgehead_lab.sh", s2)
        self.assertIn("sh scripts/s25_bridgehead_lab.sh", s25)
        self.assertNotIn("sh scripts/accept_s2_native.sh", s25)
        self.assertIn("S25_SIMWAN6_BOTH_LAYER_RAW_PACKET_AND_NATIVE_EVIDENCE_ARCHIVED=PASS", s25)
        self.assertIn("runs-on: ubuntu-24.04", s2)
        self.assertIn("runs-on: ubuntu-24.04", s25)

    def test_rc_flag_not_required_and_removed(self):
        script = (ROOT / "install.sh").read_text()
        self.assertNotIn("allow-prerelease", script)

    def test_embedded_installer_python_compiles(self):
        content = (ROOT / "install.sh").read_text()
        self.assertIn("PY_BOXWEAVE_INSTALL", content)
        embedded = content.split("<<'PY_BOXWEAVE_INSTALL'\n", 1)[1].rsplit("\nPY_BOXWEAVE_INSTALL", 1)[0]
        compile(embedded, "install.sh:python", "exec")


if __name__ == "__main__":
    unittest.main()
