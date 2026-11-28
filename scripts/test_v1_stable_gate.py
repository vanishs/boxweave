#!/usr/bin/env python3
"""Stable release verification and publication tests; no GitHub writes."""
import json
import os
from pathlib import Path
import struct
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

    def release_fixture(self, tagged=False, draft=None, present=(), starter=None, corrupt=None):
        state = {"tagged": tagged, "release": None, "payloads": {}, "calls": []}
        paths = {p.name: p for p in self.assets.iterdir()}
        notes = (f"Source {SOURCE}. Verified stable candidate.\n"
                 f"release.json SHA256: {self.verify()[0]}\nSHA256SUMS SHA256: {self.verify()[1]}")
        base = f"repos/{gate.REPO}"

        def add(name, payload, asset_state="uploaded"):
            asset_id = 100 + len(state["payloads"])
            while asset_id in state["payloads"]:
                asset_id += 1
            state["payloads"][asset_id] = payload
            state["release"]["assets"].append({"name": name, "id": asset_id,
                                               "size": len(payload), "state": asset_state})

        def new_release(is_draft):
            state["release"] = {"id": 42, "tag_name": TAG, "draft": is_draft,
                                "prerelease": False, "body": notes, "assets": []}

        if draft is not None:
            new_release(draft)
            for name in present:
                add(name, b"bad" if name == corrupt else paths[name].read_bytes())
            if starter is not None:
                add(starter, b"", "starter")

        def optional(path):
            self.assertEqual(path, f"{base}/git/ref/tags/{TAG}")
            if state["tagged"]:
                return {"ref": f"refs/tags/{TAG}", "object": {"type": "commit", "sha": SOURCE}}

        def api(path):
            self.assertEqual(path, f"{base}/releases/42")
            return state["release"]

        def gh(*args):
            state["calls"].append(args)
            if args[:2] == ("api", f"{base}/git/refs"):
                self.assertFalse(state["tagged"])
                state["tagged"] = True
            elif args[:2] == ("api", f"{base}/releases"):
                self.assertIsNone(state["release"])
                self.assertIn("draft=true", args)
                new_release(True)
                return json.dumps(state["release"]).encode()
            elif args[:2] == ("release", "upload"):
                self.assertTrue(state["release"]["draft"])
                self.assertNotIn("--clobber", args)
                path = Path(args[3])
                self.assertNotIn(path.name, {a["name"] for a in state["release"]["assets"]})
                add(path.name, path.read_bytes())
            elif args[0] == "api" and args[1].startswith(f"{base}/releases/assets/"):
                asset_id = int(args[1].rsplit("/", 1)[1])
                if "DELETE" in args:
                    self.assertTrue(state["release"]["draft"])
                    self.assertEqual(state["payloads"][asset_id], b"")
                    state["release"]["assets"] = [a for a in state["release"]["assets"] if a["id"] != asset_id]
                    del state["payloads"][asset_id]
                else:
                    return state["payloads"][asset_id]
            elif args[:2] == ("api", f"{base}/releases/42"):
                self.assertIn("draft=false", args)
                self.assertEqual({a["name"] for a in state["release"]["assets"]}, set(paths))
                state["release"]["draft"] = False
            else:
                self.fail(f"unexpected GitHub call: {args}")
            return b""

        for patcher in (patch.dict(os.environ, {"STABLE_PUBLISH_AUTHORIZED": "true"}, clear=True),
                        patch.object(ci, "protected_source", return_value=SOURCE),
                        patch.object(ci, "optional_api", side_effect=optional),
                        patch.object(ci, "api", side_effect=api),
                        patch.object(ci, "find_release", side_effect=lambda _: state["release"]),
                        patch.object(ci, "gh", side_effect=gh)):
            patcher.start()
            self.addCleanup(patcher.stop)
        return state

    def test_publish_exact_bytes_and_readback(self):
        state = self.release_fixture()
        self.assertEqual(ci.publish(ROOT, self.assets)["publication"], "PUBLISHED_EXACT_READBACK")
        self.assertTrue(state["tagged"])
        self.assertFalse(state["release"]["draft"])
        self.assertEqual(sum(c[:2] == ("release", "upload") for c in state["calls"]), 6)
        calls = state["calls"]
        state["calls"] = []
        self.assertEqual(ci.publish(ROOT, self.assets)["publication"], "PUBLISHED_EXACT_READBACK")
        self.assertTrue(all(c[0] == "api" and c[2:3] == ("-H",) for c in state["calls"]))
        self.assertTrue(any("draft=false" in c for c in calls))

    def test_resume_tag_only(self):
        state = self.release_fixture(tagged=True)
        ci.publish(ROOT, self.assets)
        self.assertFalse(any(c[1].endswith("/git/refs") for c in state["calls"]))
        self.assertFalse(state["release"]["draft"])

    def test_resume_partial_draft_with_starter(self):
        state = self.release_fixture(tagged=True, draft=True, present=("install.sh",), starter="SHA256SUMS")
        ci.publish(ROOT, self.assets)
        uploads = [Path(c[3]).name for c in state["calls"] if c[:2] == ("release", "upload")]
        self.assertEqual(set(uploads), {p.name for p in self.assets.iterdir()} - {"install.sh"})
        self.assertEqual(sum("DELETE" in c for c in state["calls"]), 1)
        self.assertFalse(state["release"]["draft"])

    def test_conflicting_draft_refused_without_mutation(self):
        state = self.release_fixture(tagged=True, draft=True, present=("install.sh",), corrupt="install.sh")
        with self.assertRaisesRegex(ValueError, "existing asset"):
            ci.publish(ROOT, self.assets)
        self.assertTrue(state["release"]["draft"])
        self.assertEqual(state["calls"], [])

    def test_incomplete_public_release_never_modified(self):
        state = self.release_fixture(tagged=True, draft=False, present=("install.sh",))
        with self.assertRaisesRegex(ValueError, "incomplete published"):
            ci.publish(ROOT, self.assets)
        self.assertEqual(state["calls"], [])

    def test_existing_wrong_tag_refused(self):
        with patch.object(ci, "optional_api", return_value={"ref": f"refs/tags/{TAG}",
                          "object": {"type": "commit", "sha": "f" * 40}}):
            with self.assertRaisesRegex(ValueError, "tag/source mismatch"):
                ci.require_tag(TAG, SOURCE)

    def test_only_confirmed_404_means_absence(self):
        for status in (b"", b"gh: unauthorized (HTTP 401)\n", b"gh: forbidden (HTTP 403)\n",
                       b"gh: unavailable (HTTP 503)\n"):
            with patch.object(ci.subprocess, "run", return_value=Mock(returncode=1, stderr=status)):
                with self.assertRaises(ValueError):
                    ci.optional_api("fixture")
        with patch.object(ci.subprocess, "run", return_value=Mock(returncode=0, stdout=b'{"id":1}')):
            self.assertEqual(ci.optional_api("fixture"), {"id": 1})
        with patch.object(ci.subprocess, "run", return_value=Mock(returncode=1, stderr=b"gh: Not Found (HTTP 404)\n")):
            self.assertIsNone(ci.optional_api("fixture"))

    def test_find_release_includes_drafts_and_rejects_duplicates(self):
        draft = {"tag_name": TAG, "draft": True}
        with patch.object(ci, "gh", side_effect=[json.dumps([{"tag_name": "old"}] * 30).encode(),
                                                json.dumps([draft]).encode()]):
            self.assertEqual(ci.find_release(TAG), draft)
        with patch.object(ci, "gh", return_value=json.dumps([draft, draft]).encode()):
            with self.assertRaisesRegex(ValueError, "ambiguous"):
                ci.find_release(TAG)

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
        helper = (ROOT / "scripts/ci_candidate_build.py").read_text()
        self.assertIn('python3 scripts/ci_candidate_build.py "$version" "$GITHUB_SHA" "$ARCH" "$binary"', build)
        self.assertIn("python3 scripts/test_ci_candidate_build.py", build)
        self.assertLess(build.index('python3 scripts/ci_candidate_build.py "$version" "$GITHUB_SHA" "$ARCH" "$binary"'),
                        build.index("go test -count=1 ./..."))
        self.assertIn("SELECT version FROM schema_version", helper)
        self.assertIn("== [(7,)]", helper)
        self.assertIn("network_address_fences", helper)
        self.assertIn("network_ipv4_retirement", helper)
        self.assertIn("SELECT count(*) FROM operations", helper)
        self.assertIn("?mode=ro", helper)
        self.assertIn("needs: package", publish)
        self.assertIn("if: inputs.publish == true", publish)
        self.assertIn("contents: write", publish)
        self.assertIn("name: v1-stable-assets", publish)
        self.assertNotIn("secrets.", release)
        self.assertNotIn("environment:", release)
        self.assertNotIn("evidence_commit", release)


if __name__ == "__main__":
    unittest.main()
