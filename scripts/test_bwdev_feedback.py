"""Reliable feedback boundary regressions; no native effects."""
import argparse
import json
import shutil
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "dev"))
from harness import feedback, learning, records, work
from harness.common import HarnessError


class FeedbackTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.base = Path(self.tmp.name)
        self.inbox = self.base / "inbox"
        self.inbox.mkdir()
        shutil.copytree(ROOT / "dev/learning/BDH-L014", self.inbox / "BDH-L014")
        self.item = {"asset": "skill.box-development", "outcome": "helped",
                     "evidence": "fixture operation 123", "attempt_ref": "second",
                     "observation": "Verified short command transfer",
                     "decision": "linked", "candidate": "BDH-L014",
                     "reason": "Existing procedure worked"}
        self.row = {"kind": "work", "schema_version": 1, "feedback_protocol": 1,
                    "work_ref": "fixture/current", "attempt_ref": "second",
                    "task_kind": "development", "objective": "fixture", "risk": "standard",
                    "authority": "fixture only", "target": {"kind": "repository", "ref": "fixture",
                    "version": None, "source_sha": None}, "state": "HANDOFF",
                    "boundary": "handoff", "next_safe_action": "review",
                    "used_assets": [self.item["asset"]], "evidence": [self.item["evidence"]],
                    "asset_feedback": [self.item], "learning_links": ["BDH-L014"],
                    "learning": {"kind": "reuse", "ref": "procedure worked"}}

    def review(self, row=None):
        return feedback.review(work.validate(row or self.row), ROOT, self.inbox)

    def observe(self):
        args = argparse.Namespace(action="observe", inbox=self.inbox, id="BDH-L014",
                expect=records.latest(self.inbox / "BDH-L014")["revision"], work_ref=self.row["work_ref"],
                attempt_ref=self.item["attempt_ref"], evidence=self.item["evidence"],
                observation=self.item["observation"], outcome=self.item["outcome"])
        learning.run(args, ROOT)

    def test_actual_link_required_and_exact_match(self):
        result, code = self.review()
        self.assertEqual(code, 1)
        self.assertEqual(result["items"][0]["related"][0]["id"], "BDH-L014")
        self.observe()
        self.assertEqual(self.review()[0]["status"], "FEEDBACK_RECORDED")
        for key, value in (("attempt_ref", "other"), ("observation", "changed"), ("outcome", "misled")):
            changed = dict(self.row, asset_feedback=[dict(self.item, **{key: value})])
            self.assertEqual(self.review(changed)[1], 1)

    def test_related_no_change_and_explicit_not_applicable(self):
        row = dict(self.row, asset_feedback=[dict(self.item, decision="no-change", candidate=None)])
        self.assertEqual(self.review(row)[1], 1)
        row["asset_feedback"][0]["decision"] = "not-applicable"
        row["asset_feedback"][0]["reason"] = "Same skill, different observation not related to candidate"
        self.assertEqual(self.review(row)[1], 0)

    def test_pending_and_legacy(self):
        row = dict(self.row, asset_feedback=[dict(self.item, decision="pending", candidate=None)])
        self.assertEqual(self.review(row)[1], 1)
        row = dict(self.row, asset_feedback=[], learning={"kind": "none", "ref": "No material reuse"})
        self.assertEqual(self.review(row)[1], 0)
        row.pop("feedback_protocol")
        self.assertEqual(self.review(row)[1], 1)

    def test_preserve_feedback_facts(self):
        old = dict(self.row, revision=1)
        for patch in ({"asset_feedback": []}, {"asset_feedback": [dict(self.item, observation="different")]},
                      {"feedback_protocol": 0}, {"asset_feedback": [self.item, self.item]}):
            with self.assertRaises(HarnessError):
                work.advance(old, dict(self.row, **patch), 1)
        updated = dict(self.row, asset_feedback=[dict(self.item, decision="pending", candidate=None)])
        self.assertEqual(work.advance(old, updated, 1)["state"], "HANDOFF")

    def test_boundary_keeps_pending_but_refuses_false_completion(self):
        history = self.base / "history"
        initial = dict(self.row, state="ACTIVE", asset_feedback=[], learning=None)
        initial.pop("boundary")
        records.publish(history, 0, initial)
        file = self.base / "boundary.json"
        file.write_text(json.dumps(dict(self.row, state="RECORDED", boundary="completion")))
        args = argparse.Namespace(action="boundary", directory=history, file=file, expect=1, inbox=self.inbox)
        result, code = work.run(args, ROOT)
        self.assertEqual(code, 1)
        self.assertFalse(result["published"])
        self.assertEqual(records.latest(history)["revision"], 1)
        file.write_text(json.dumps(self.row))
        result, code = work.run(args, ROOT)
        self.assertEqual(code, 0)
        self.assertEqual(result["feedback"]["status"], "REVIEW_REQUIRED")
        self.observe()
        file.write_text(json.dumps(dict(self.row, state="RECORDED", boundary="completion",
                                       evidence=self.row["evidence"] + ["learning revision reconciled"])))
        args.expect = 2
        result, code = work.run(args, ROOT)
        self.assertEqual(code, 0)
        self.assertEqual(result["feedback"]["status"], "FEEDBACK_RECORDED")
        self.assertEqual(records.latest(history)["revision"], 3)

    def test_wrong_link_missing_evidence_and_missing_inbox(self):
        self.observe()
        self.assertEqual(self.review(dict(self.row, learning_links=[]))[1], 1)
        self.assertEqual(self.review(dict(self.row, evidence=[]))[1], 1)
        self.assertEqual(feedback.review(self.row, ROOT, self.base / "absent")[1], 1)
