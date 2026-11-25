"""Universal entry and portable envelope validation."""
import argparse
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "dev"))
from harness import work, records
from harness.common import HarnessError


class WorkTests(unittest.TestCase):
    def setUp(self):
        self.data = records.read_json(ROOT / "dev/examples/WORK_ENVELOPE.json")

    def test_plan_no_target_or_mutation(self):
        result = work.plan(ROOT, "operations", "install rc14")
        self.assertEqual(result["status"], "PLAN_ONLY")
        self.assertEqual(result["assets"][0]["id"], "knowledge.maintenance")
        self.assertNotIn("target", result)

    def test_tiny_guidance(self):
        result = work.plan(ROOT, "guidance", "what is box")
        self.assertEqual(result["assets"][0]["id"], "knowledge.index")
        self.assertLessEqual(len(result["assets"]), 7)

    def test_unknown_target_is_not_completion(self):
        self.data.update(state="RECORDED", evidence=["example"], learning={"kind":"none","ref":"No change"})
        with self.assertRaises(HarnessError):
            work.validate(self.data)
        self.data["state"] = "ANSWERED"
        with self.assertRaises(HarnessError):
            work.validate(self.data)
        self.data["task_kind"] = "guidance"
        self.assertEqual(work.validate(self.data)["state"], "ANSWERED")

    def test_invalid_envelope_and_boundary(self):
        for key, value in (("target", {}), ("state", "PASS"), ("learning", {"kind":"auto","ref":"x"}),
                           ("work_ref", ""), ("evidence", ["same","same"]), ("raw_transcript", "not allowed")):
            data = dict(self.data, **{key:value})
            with self.assertRaises(HarnessError):
                work.validate(data)
        data = dict(self.data, state="HANDOFF")
        with self.assertRaises(HarnessError):
            work.validate(data)

    def test_boundary_preserves_scope_and_original_evidence(self):
        old = dict(self.data, revision=1, original_operations=["original-op"], evidence=["old"])
        new = dict(self.data, state="HANDOFF", boundary="handoff", original_operations=["original-op"],
                   evidence=["old", "handoff"], learning={"kind":"pending","ref":"remote unavailable"})
        self.assertEqual(work.advance(old, new, 1)["state"], "HANDOFF")
        self.assertIn("Learning persistence pending", work.summary(new)["issues"])
        for patch in ({"authority":"different"}, {"evidence":[]}, {"original_operations":[]}, {"risk":"low"}):
            with self.assertRaises(HarnessError):
                work.advance(old, dict(new, **patch), 1)
        with self.assertRaises(HarnessError):
            work.advance(old, new, 2)
        with self.assertRaises(HarnessError):
            work.advance(dict(old, state="UNKNOWN", learning=new["learning"]), dict(new, state="ACTIVE"), 1)

    def test_unknown_cannot_be_laundered_by_handoff(self):
        old = dict(self.data, revision=1, state="UNKNOWN", learning={"kind":"none","ref":"review"},
                   original_operations=["op"])
        new = dict(self.data, state="HANDOFF", boundary="handoff", learning=old["learning"],
                   original_operations=["op"])
        with self.assertRaises(HarnessError):
            work.advance(old, new, 1)

    def test_version_change_requires_new_evidence(self):
        old = dict(self.data, revision=1, target={"kind":"host","ref":"lab-host","version":"rc13","source_sha":None},
                   evidence=["initial"])
        new = dict(self.data, target=dict(old["target"],version="rc14"), evidence=["initial"],
                   state="RECORDED", boundary="completion", learning={"kind":"none","ref":"no new gap"})
        with self.assertRaises(HarnessError):
            work.advance(old, new, 1)
        new["evidence"].append("new observed version/poststate")
        self.assertEqual(work.advance(old, new, 1)["target"]["version"], "rc14")
        new["target"]["ref"] = "different-host"
        with self.assertRaises(HarnessError):
            work.advance(old, new, 1)

    def test_cross_process_boundary_and_export(self):
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            history = base / "history"
            boundary = base / "boundary.json"
            exported = base / "export.json"
            def cli(*args):
                return subprocess.run([sys.executable, str(ROOT/"dev/bwdev"), "--json", "work", *map(str,args)],
                                      cwd=base, text=True, capture_output=True)
            self.assertEqual(cli("begin", history, ROOT/"dev/examples/WORK_ENVELOPE.json").returncode, 0)
            boundary.write_text(json.dumps(dict(self.data, state="HANDOFF", boundary="handoff",
                learning={"kind":"pending","ref":"remote not available"})), encoding="utf-8")
            self.assertEqual(cli("boundary", history, boundary, "--expect", 1).returncode, 0)
            self.assertEqual(cli("boundary", history, boundary, "--expect", 1).returncode, 2)
            resumed = cli("resume", history)
            self.assertEqual(resumed.returncode, 1)
            self.assertEqual(json.loads(resumed.stdout)["record"]["revision"], 2)
            self.assertEqual(cli("export", history, exported).returncode, 0)
            self.assertEqual(cli("check", exported).returncode, 0)
            self.assertEqual(cli("export", history, exported).returncode, 2)

    def test_portable_begin_exclusive(self):
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp) / "history"
            args = argparse.Namespace(action="begin", directory=p, file=ROOT/"dev/examples/WORK_ENVELOPE.json")
            result, code = work.run(args, ROOT)
            self.assertEqual(code, 0)
            self.assertEqual(result["record"]["revision"], 1)
            with self.assertRaises(HarnessError):
                work.run(args, ROOT)
            self.assertEqual(records.latest(p)["target"]["source_sha"], None)


if __name__ == "__main__":
    unittest.main()
