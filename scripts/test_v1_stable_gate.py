#!/usr/bin/env python3
"""Stable release verification and publication tests; no GitHub writes."""
import os
from pathlib import Path
import shutil
import struct
import subprocess
import tempfile
import unittest
from unittest.mock import patch, Mock

import v1_release_manifest as manifest
import v1_stable_gate as gate
import v1_stable_release_ci as ci

ROOT = Path(__file__).resolve().parent.parent
SOURCE = "a" * 40
TAG = "v" + (ROOT / "VERSION").read_text().strip()


class StableGateTests(unittest.TestCase):
    def setUp(self):
        t = tempfile.TemporaryDirectory()
        self.addCleanup(t.cleanup)
        self.dir = Path(t.name)
        self.assets = self.dir / "assets"
        self.assets.mkdir()
        for arch, machine in manifest.MACHINES.items():
            data = bytearray(64)
            data[:6] = b"\x7fELF\x02\x01"
            struct.pack_into("<H", data, 18, machine)
            (self.assets / f"boxweave-{TAG}-linux-{arch}").write_bytes(data)
        manifest.assemble(self.assets, TAG, SOURCE, ROOT / "install.sh")

    def verify(self, **kw):
        args = {"directory": self.assets, "root": ROOT, "source": SOURCE, "tag": TAG}
        args.update(kw)
        return gate.verify_assets(**args)

    def test_exact_candidate_assets(self):
        self.assertEqual(self.verify(), (
            gate.sha((self.assets / "release.json").read_bytes()),
            gate.sha((self.assets / "SHA256SUMS").read_bytes())))
        with self.assertRaises(ValueError):
            self.verify(source="f" * 40)

    def test_asset_byte_inventory_symlink_and_rc_refusal(self):
        files = list(self.assets.iterdir())
        for f in files:
            old = f.read_bytes()
            try:
                f.write_bytes(old + b"changed")
                with self.subTest(file=f.name), self.assertRaises(ValueError):
                    self.verify()
            finally:
                f.write_bytes(old)
        extra = self.assets / "foreign.bin"
        extra.write_bytes(b"x")
        with self.assertRaisesRegex(ValueError, "unexpected"):
            self.verify()
        extra.unlink()
        p = self.assets / "install.sh"
        p.unlink()
        p.symlink_to(ROOT / "install.sh")
        with self.assertRaisesRegex(ValueError, "non-regular"):
            self.verify()
        p.unlink()
        p.write_bytes((ROOT / "install.sh").read_bytes())
        with self.assertRaises(ValueError):
            self.verify(tag=TAG+"-rc.9")


    def test_publish_requires_authorization_and_valid_bytes(self):
        with patch.dict(os.environ, {}, clear=True), patch.object(ci, "gh") as gh:
            with self.assertRaisesRegex(ValueError, "not explicitly authorized"):
                ci.publish(ROOT, self.assets)
            gh.assert_not_called()
        with patch.dict(os.environ, {"STABLE_PUBLISH_AUTHORIZED": "true"}, clear=True), \
                patch.object(ci, "protected_source", return_value=SOURCE), patch.object(ci, "gh") as gh:
            (self.assets / "SHA256SUMS").write_text("wrong")
            with self.assertRaises(ValueError):
                ci.publish(ROOT, self.assets)
            gh.assert_not_called()

    def test_publish_exact_bytes_and_readback(self):
        calls = []

        def gh(*args):
            calls.append(args)
            if args[:2] == ("release", "download"):
                dest = Path(args[args.index("--dir") + 1])
                for path in self.assets.iterdir():
                    shutil.copyfile(path, dest / path.name)
            return b""

        def api(path):
            if "/releases/tags/" in path:
                return {"tag_name": TAG, "draft": False, "prerelease": False,
                        "assets": [{"name": p.name} for p in self.assets.iterdir()]}
            return {"object": {"sha": SOURCE}}

        with patch.dict(os.environ, {"STABLE_PUBLISH_AUTHORIZED": "true"}, clear=True), \
                patch.object(ci, "protected_source", return_value=SOURCE), \
                patch.object(ci, "gh", side_effect=gh), patch.object(ci, "api", side_effect=api), \
                patch.object(ci, "not_exists") as absent:
            result = ci.publish(ROOT, self.assets)
            self.assertEqual(result["publication"], "PUBLISHED_EXACT_READBACK")
            self.assertEqual(absent.call_count, 2)
            self.assertEqual([c[:2] for c in calls],
                             [("api", f"repos/{gate.REPO}/git/refs"),
                              ("release", "create"), ("release", "download")])
            self.assertIn("--verify-tag", calls[1])

    def test_existing_or_unknown_tag_refusal(self):
        for status in (b"", b"gh: unauthorized (HTTP 401)\n", b"gh: forbidden (HTTP 403)\n",
                       b"gh: unavailable (HTTP 503)\n"):
            with patch.object(ci.subprocess, "run", return_value=Mock(returncode=1, stderr=status)):
                with self.assertRaises(ValueError):
                    ci.not_exists("fixture")
        with patch.object(ci.subprocess, "run", return_value=Mock(returncode=0, stderr=b"")):
            with self.assertRaises(ValueError):
                ci.not_exists("fixture")
        with patch.object(ci.subprocess, "run", return_value=Mock(returncode=1, stderr=b"gh: Not Found (HTTP 404)\n")):
            ci.not_exists("fixture")

    def test_protected_source_needs_real_manual_v1_context(self):
        env = {"GITHUB_ACTIONS": "true", "GITHUB_EVENT_NAME": "workflow_dispatch", "GITHUB_REPOSITORY": gate.REPO,
               "GITHUB_REF": "refs/heads/v1", "GITHUB_REF_PROTECTED": "true", "GITHUB_SHA": SOURCE}
        for field in env:
            with patch.dict(os.environ, env, clear=True), patch.dict(os.environ, {field: ""}):
                with self.assertRaises(ValueError):
                    ci.protected_source(ROOT)
        with patch.dict(os.environ, env, clear=True), patch.object(ci.subprocess, "check_output", return_value=SOURCE.encode()), \
                patch.object(ci, "api", return_value={"object": {"sha": "f"*40}}):
            with self.assertRaisesRegex(ValueError, "HEAD advanced"):
                ci.protected_source(ROOT)

    def test_completed_s5_workflows_are_historical_only(self):
        workflows = ROOT / ".github/workflows"
        historical = sorted(workflows.glob("v1-s5*.yml"))
        self.assertTrue(historical)
        for path in historical:
            with self.subTest(workflow=path.name):
                text = path.read_text()
                self.assertTrue(text.startswith("name: Historical "))
                self.assertIn("on:\n  workflow_dispatch:", text)
                self.assertNotIn("\n  push:", text)
                self.assertNotIn("\n  pull_request:", text)
                self.assertNotIn(path.name, (workflows / "v1-stable-release.yml").read_text())

    def test_single_manual_workflow_keeps_publication_separate(self):
        workflows = ROOT / ".github/workflows"
        release = (workflows / "v1-stable-release.yml").read_text()
        self.assertFalse((workflows / "v1-stable-candidate.yml").exists())
        build, publish = release.split("\n  publish:\n", 1)
        self.assertIn("workflow_dispatch:", build)
        self.assertIn("default: false", build)
        self.assertNotIn("contents: write", build)
        self.assertIn("make verify", build)
        self.assertIn("SELECT version FROM schema_version", build)
        self.assertIn("== [(7,)]", build)
        self.assertIn("network_address_fences", build)
        self.assertIn("network_ipv4_retirement", build)
        self.assertIn("SELECT count(*) FROM operations", build)
        self.assertIn("?mode=ro", build)
        self.assertIn("needs: package", publish)
        self.assertIn("if: inputs.publish == true", publish)
        self.assertIn("contents: write", publish)
        self.assertIn("name: v1-stable-assets-${{ github.run_attempt }}", publish)
        self.assertNotIn("secrets.", release)
        self.assertNotIn("environment:", release)
        self.assertNotIn("evidence_commit", release)


if __name__ == "__main__":
    unittest.main()
