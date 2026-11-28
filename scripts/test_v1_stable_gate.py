#!/usr/bin/env python3
"""Stable gate tests with independent disposable keys and no GitHub writes."""
import base64
import copy
import hashlib
import json
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
RUN = 123456
ATTEMPT = 2
EVIDENCE = "b" * 40


def run_data():
    return {"path": gate.WORKFLOW, "event": "workflow_dispatch", "head_branch": "v1",
            "head_sha": SOURCE, "status": "completed", "conclusion": "success",
            "repository": {"full_name": gate.REPO}, "head_repository": {"full_name": gate.REPO},
            "id": RUN, "run_attempt": ATTEMPT}


def job_data():
    return {"total_count": len(gate.JOBS), "jobs": [
        {"name": job, "run_id": RUN, "run_attempt": ATTEMPT, "head_sha": SOURCE,
         "status": "completed", "conclusion": "success"} for job in sorted(gate.JOBS)]}


def create_key(directory, name):
    path = directory / (name + ".pem")
    subprocess.run(["openssl", "genpkey", "-algorithm", "ED25519", "-out", str(path)],
                   capture_output=True, check=True)
    pub = subprocess.check_output(["openssl", "pkey", "-in", str(path), "-pubout"])
    return path, pub


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
        self.owner, self.owner_pub = create_key(self.dir, "owner")
        self.verifier, self.verifier_pub = create_key(self.dir, "verifier")
        self.run = run_data()
        self.jobs = job_data()
        self.s = {"format": 1, "purpose": "BOXWEAVE_STABLE_PUBLICATION", "repository": gate.REPO,
                  "source": SOURCE, "tag": TAG, "candidate_run_id": RUN, "candidate_run_attempt": ATTEMPT,
                  "release_json_sha256": gate.sha((self.assets / "release.json").read_bytes()),
                  "sha256sums_sha256": gate.sha((self.assets / "SHA256SUMS").read_bytes()),
                  "gates": {name: {"status": "PASS", "evidence": [
                      {"ref": f"fixture:{name}", "sha256": "c" * 64}]} for name in gate.GATES},
                  "scope_decisions": []}

    def signatures(self, raw=None, same=False):
        if raw is None:
            raw = json.dumps(self.s, sort_keys=True).encode()
        p = self.dir / "signoff.json"
        p.write_bytes(raw)
        sigs = []
        for key in (self.owner, self.owner if same else self.verifier):
            sigs.append(subprocess.check_output(["openssl", "pkeyutl", "-sign", "-inkey", str(key),
                                                 "-rawin", "-in", str(p)]))
        return raw, *sigs

    def verify(self, signatures=None, keys=None, **kw):
        args = {"directory": self.assets, "root": ROOT, "run": self.run, "jobs": self.jobs,
                "source": SOURCE, "tag": TAG}
        args.update(kw)
        raw, s1, s2 = signatures or self.signatures()
        k1, k2 = keys or (self.owner_pub, self.verifier_pub)
        return gate.verify(**args, raw=raw, owner_sig=s1, verifier_sig=s2, owner_key=k1, verifier_key=k2)

    def test_positive_independent_signed_dry_run(self):
        result = self.verify()
        self.assertEqual(result["verdict"], "STABLE_RELEASE_GATE_PASS")
        self.assertEqual(result["source"], SOURCE)
        self.assertEqual(result["run_attempt"], ATTEMPT)
        self.assertEqual(result["sha256sums_sha256"], self.s["sha256sums_sha256"])

    def test_two_signatures_are_mandatory_and_distinct(self):
        raw, s1, s2 = self.signatures()
        for name, signed, keys in [
            ("unsigned", (raw, b"", b""), None),
            ("one signature", (raw, s1, b""), None),
            ("forged signature", (raw, s1, bytes(64)), None),
            ("wrong signer", (raw, s2, s1), None),
            ("changed payload", (raw+b" ", s1, s2), None),
            ("same signer", self.signatures(same=True), (self.owner_pub, self.owner_pub)),
            ("missing trust", (raw, s1, s2), (b"", self.verifier_pub)),
        ]:
            with self.subTest(name=name), self.assertRaises(ValueError):
                self.verify(signed, keys)

    def test_all_mandatory_gates_fail_closed_even_if_signed(self):
        original = copy.deepcopy(self.s)
        for name in gate.GATES:
            for status in (None, "BLOCKED", "PARTIAL", "SIMULATED_PASS", "WAIVED"):
                with self.subTest(name=name, status=status), self.assertRaises(ValueError):
                    self.s = copy.deepcopy(original)
                    if status is None:
                        del self.s["gates"][name]
                    else:
                        self.s["gates"][name]["status"] = status
                    self.verify()
        self.s = copy.deepcopy(original)
        self.s["gates"]["S5A3"]["evidence"] = []
        with self.assertRaises(ValueError):
            self.verify()

    def test_signed_wrong_binding_and_duplicates_denied(self):
        original = copy.deepcopy(self.s)
        for key, value in [("repository", "foreign/boxweave"), ("purpose", "TEST_ONLY"),
                           ("source", "f"*40), ("tag", TAG+"-rc.9"), ("candidate_run_id", RUN+1),
                           ("candidate_run_attempt", ATTEMPT+1), ("release_json_sha256", "f"*64),
                           ("sha256sums_sha256", "a"*64), ("extra", True)]:
            with self.subTest(key=key), self.assertRaises(ValueError):
                self.s = copy.deepcopy(original)
                self.s[key] = value
                self.verify()
        self.s = original
        raw = json.dumps(self.s).encode().replace(b'"format": 1', b'"format": 1, "format": 1', 1)
        with self.assertRaisesRegex(ValueError, "duplicate"):
            self.verify(self.signatures(raw))

    def test_run_provenance_rejects_rc_fork_wrong_ref_and_failed_jobs(self):
        for k, value in [("path", ".github/workflows/v1-release.yml"), ("event", "push"),
                         ("head_branch", "feat/v1-x"), ("head_sha", "f"*40),
                         ("conclusion", "failure"), ("status", "in_progress"),
                         ("id", True), ("head_repository", {"full_name": "fork/boxweave"})]:
            with self.subTest(k=k), self.assertRaises(ValueError):
                run = run_data()
                run[k] = value
                self.verify(run=run)
        for kind in ("skipped", "missing", "duplicate", "stale", "partial"):
            with self.subTest(kind=kind), self.assertRaises(ValueError):
                j = job_data()
                if kind == "skipped":
                    j["jobs"][0]["conclusion"] = "skipped"
                elif kind == "missing":
                    j["jobs"].pop()
                elif kind == "duplicate":
                    j["jobs"][0] = j["jobs"][1]
                elif kind == "stale":
                    j["jobs"][0]["run_attempt"] = 1
                else:
                    j["total_count"] += 1
                self.verify(jobs=j)

    def test_asset_byte_inventory_symlink_and_rc_refusal(self):
        sigs = self.signatures()
        files = list(self.assets.iterdir())
        for f in files:
            old = f.read_bytes()
            try:
                f.write_bytes(old + b"changed")
                with self.subTest(file=f.name), self.assertRaises(ValueError):
                    self.verify(sigs)
            finally:
                f.write_bytes(old)
        extra = self.assets / "foreign.bin"
        extra.write_bytes(b"x")
        with self.assertRaisesRegex(ValueError, "unexpected"):
            self.verify(sigs)
        extra.unlink()
        p = self.assets / "install.sh"
        p.unlink()
        p.symlink_to(ROOT / "install.sh")
        with self.assertRaisesRegex(ValueError, "non-regular"):
            self.verify(sigs)
        p.unlink()
        p.write_bytes((ROOT / "install.sh").read_bytes())
        with self.assertRaises(ValueError):
            self.verify(sigs, tag=TAG+"-rc.9")

    def test_ci_driver_read_only_and_blocked_never_writes(self):
        raw, s1, s2 = self.signatures()
        evidence = {"signoff.json": raw, "owner.sig": s1, "verifier.sig": s2}
        calls = []
        def api(path):
            calls.append(path)
            if "/attempts/" in path:
                return self.jobs
            if path.endswith("/artifacts?per_page=100"):
                return {"total_count": 1, "artifacts": [{"id": 88, "name": f"v1-stable-assets-{ATTEMPT}",
                        "expired": False, "size_in_bytes": 10000}]}
            if path.endswith(f"/runs/{RUN}"):
                return self.run
            raise AssertionError(path)
        def gh(*args):
            calls.append(args)
            self.assertEqual(args[:2], ("run", "download"))
            dest = Path(args[args.index("--dir")+1])
            for f in self.assets.iterdir():
                shutil.copyfile(f, dest / f.name)
            return b""
        def ef(ref, tag, filename, limit):
            self.assertEqual((ref, tag), (EVIDENCE, TAG))
            return evidence[filename]
        env = {"STABLE_OWNER_PUBLIC_KEY": self.owner_pub.decode(),
               "STABLE_VERIFIER_PUBLIC_KEY": self.verifier_pub.decode()}
        with patch.object(ci, "protected_source", return_value=SOURCE), patch.object(ci, "api", side_effect=api), \
                patch.object(ci, "gh", side_effect=gh), patch.object(ci, "evidence_file", side_effect=ef), \
                patch.object(ci, "not_exists") as absent, patch.dict(os.environ, env, clear=True):
            result = ci.gate(ROOT, str(RUN), EVIDENCE)
            self.assertEqual(result["publication"], "DRY_RUN_NO_PUBLICATION")
            self.assertEqual(absent.call_count, 2)
            with self.assertRaisesRegex(ValueError, "not explicitly authorized"):
                ci.gate(ROOT, str(RUN), EVIDENCE, publish=True)
            evidence["owner.sig"] = b"bad"
            with self.assertRaises(ValueError):
                ci.gate(ROOT, str(RUN), EVIDENCE, publish=True)
            with patch.dict(os.environ, {"STABLE_OWNER_PUBLIC_KEY": ""}):
                with self.assertRaisesRegex(ValueError, "not configured"):
                    ci.gate(ROOT, str(RUN), EVIDENCE)

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
                for active in ("v1-stable-candidate.yml", "v1-stable-release.yml"):
                    self.assertNotIn(path.name, (workflows / active).read_text())

    def test_signed_historical_s5_evidence_does_not_require_workflow_replay(self):
        for name, entry in self.s["gates"].items():
            entry["evidence"] = [{"ref": f"historical:{name}:completed", "sha256": "d" * 64}]
        self.assertEqual(self.verify()["verdict"], "STABLE_RELEASE_GATE_PASS")

    def test_workflow_separates_candidate_readonly_and_publish(self):
        build = (ROOT / ".github/workflows/v1-stable-candidate.yml").read_text()
        release = (ROOT / ".github/workflows/v1-stable-release.yml").read_text()
        self.assertIn("workflow_dispatch:", build)
        self.assertNotIn("contents: write", build)
        self.assertNotIn("gh release create", build)
        self.assertIn('SELECT version FROM schema_version', build)
        self.assertIn('== [(7,)]', build)
        self.assertIn('network_address_fences', build)
        self.assertIn('network_ipv4_retirement', build)
        self.assertIn('SELECT count(*) FROM operations', build)
        self.assertIn('?mode=ro', build)
        self.assertIn("default: false", release)
        self.assertIn("needs: verify", release)
        self.assertIn("if: inputs.publish == true", release)
        self.assertEqual(release.count("environment: v1-stable-release"), 2)
        self.assertIn("contents: write", release)
        self.assertIn("STABLE_OWNER_PUBLIC_KEY", release)
        self.assertIn("STABLE_VERIFIER_PUBLIC_KEY", release)


if __name__ == "__main__":
    unittest.main()
