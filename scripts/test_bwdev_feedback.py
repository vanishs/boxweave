"""Reliable feedback boundary regressions; no native effects."""
import argparse
import json
import shutil
import subprocess
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
        row["learning_links"] = []
        self.assertEqual(self.review(row)[1], 0)

    def test_pending_and_legacy(self):
        row = dict(self.row, asset_feedback=[dict(self.item, decision="pending", candidate=None)])
        self.assertEqual(self.review(row)[1], 1)
        row = dict(self.row, asset_feedback=[], used_assets=[], learning_links=[],
                   learning={"kind": "none", "ref": "No material reuse"})
        self.assertEqual(self.review(row)[1], 0)
        self.assertEqual(feedback.review(row, ROOT, self.base / "missing")[1], 1)
        corrupt = self.base / "corrupt"
        corrupt.mkdir()
        (corrupt / "bad").mkdir()
        self.assertEqual(feedback.review(row, ROOT, corrupt)[1], 1)
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

    def test_unaccounted_unknown_and_alias_feedback(self):
        self.observe()
        self.assertEqual(self.review(dict(self.row, used_assets=self.row["used_assets"] + ["tool.bwdev"]))[1], 1)
        unknown = dict(self.item, asset="not-a-real-asset", decision="no-change", candidate=None)
        row = dict(self.row, used_assets=[unknown["asset"]], asset_feedback=[unknown], learning_links=[])
        self.assertEqual(self.review(row)[1], 1)
        alias = dict(self.item, asset="dev/skills/box-development/SKILL.md")
        row = dict(self.row, used_assets=self.row["used_assets"] + [alias["asset"]],
                   asset_feedback=[self.item, alias])
        self.assertEqual(self.review(row)[1], 1)
        self.assertEqual(self.review(dict(self.row, learning_links=["BDH-L014", "not-linked"]))[1], 1)

    def test_legacy_begin_and_completion_refused_but_handoff_allowed(self):
        initial = dict(self.row, state="ACTIVE", asset_feedback=[], used_assets=[], learning_links=[], learning=None)
        initial.pop("feedback_protocol")
        initial.pop("boundary")
        history = self.base / "legacy"
        file = self.base / "legacy.json"
        file.write_text(json.dumps(initial))
        args = argparse.Namespace(action="begin", directory=history, file=file, inbox=self.inbox)
        with self.assertRaises(HarnessError):
            work.run(args, ROOT)
        records.publish(history, 0, initial)
        final = dict(initial, state="RECORDED", boundary="completion", learning={"kind": "none", "ref": "fixture"})
        file.write_text(json.dumps(final))
        args.action = "boundary"
        args.expect = 1
        self.assertEqual(work.run(args, ROOT)[1], 1)
        self.assertEqual(records.latest(history)["revision"], 1)
        file.write_text(json.dumps(dict(final, state="HANDOFF", boundary="handoff")))
        self.assertEqual(work.run(args, ROOT)[1], 0)
        self.assertEqual(records.latest(history)["revision"], 2)

    def test_candidate_decision_can_be_corrected_without_losing_prior_fact(self):
        history = self.base / "correction-history"
        initial = dict(self.row, state="ACTIVE", learning=None)
        initial.pop("boundary")
        records.publish(history, 0, initial)
        revised = dict(self.row, asset_feedback=[dict(self.item, decision="not-applicable",
                       candidate=None, reason="Original candidate association was mistaken")],
                       learning_links=[])
        records.publish(history, 1, work.advance(records.latest(history), revised, 1))
        self.assertEqual(work.read_history(history)["learning_links"], [])
        self.assertEqual(records.read_json(history / "00000001.json")["learning_links"], ["BDH-L014"])
        self.assertEqual(self.review(revised)[1], 0)

    def test_history_rejects_changed_original_facts(self):
        history = self.base / "invalid-history"
        initial = dict(self.row, state="ACTIVE", learning=None)
        initial.pop("boundary")
        records.publish(history, 0, initial)
        records.publish(history, 1, dict(self.row, asset_feedback=[dict(self.item, observation="rewritten")]))
        with self.assertRaises(HarnessError):
            work.read_history(history)

    def test_learning_history_cannot_drop_prior_observation(self):
        self.observe()
        directory = self.inbox / "BDH-L014"
        original = records.latest(directory)
        changed = dict(original, observations=original["observations"][1:])
        records.publish(directory, original["revision"], changed)
        self.assertEqual(self.review()[1], 1)
        with self.assertRaises(HarnessError):
            learning.audit(self.inbox, ROOT)

    def test_learning_history_rejects_malformed_prior_revision(self):
        self.observe()
        prior = self.inbox / "BDH-L014" / "00000002.json"
        data = records.read_json(prior)
        prior.write_text(json.dumps(dict(data, kind="invalid")))
        self.assertEqual(self.review()[1], 1)

    def test_improvement_requires_current_implementation_digest(self):
        self.observe()
        directory = self.inbox / "BDH-L014"
        original = records.latest(directory)
        records.publish(directory, original["revision"],
                        dict(original, implementation=dict(original["implementation"], asset_sha256="0"*64)))
        row = dict(self.row, learning={"kind": "improved", "ref": "BDH-L014"})
        result, code = self.review(row)
        self.assertEqual(code, 1)
        self.assertIn("Attached implementation drifted", " ".join(result["issues"]))
        # A historic observation remains a fact even if the implementation later changes.
        self.assertEqual(self.review()[1], 0)

    def test_duplicate_linked_capture_requires_review(self):
        self.observe()
        directory = self.inbox / "BDH-L014"
        duplicate = records.read_json(directory / "00000001.json")
        records.publish(self.inbox / "Lduplicate", 0, dict(duplicate, id="Lduplicate"))
        self.assertEqual(self.review()[1], 1)

    def test_concurrent_observe_and_retry_preserve_original_observations(self):
        directory = self.inbox / "BDH-L014"
        initial = records.latest(directory)
        expected = initial["revision"]

        def command(label, revision):
            return [sys.executable, str(ROOT / "dev/bwdev"), "--json", "learn",
                    "observe", str(self.inbox), "BDH-L014", "--expect", str(revision),
                    "--work-ref", "fixture/" + label, "--attempt-ref", "one",
                    "--evidence", "fixture:" + label, "--observation", "Observed reuse",
                    "--outcome", "helped"]

        labels = ["writer-a", "writer-b"]
        children = [subprocess.Popen(command(label, expected), stdout=subprocess.PIPE,
                                     stderr=subprocess.PIPE, text=True) for label in labels]
        for child in children:
            child.communicate(timeout=20)
        self.assertEqual(sum(child.returncode == 0 for child in children), 1)
        loser = next(label for label, child in zip(labels, children) if child.returncode != 0)
        after = records.latest(directory)
        self.assertEqual(after["revision"], expected + 1)
        self.assertEqual(after["observations"][:-1], initial["observations"])
        retry = subprocess.run(command(loser, expected + 1), capture_output=True, timeout=20)
        self.assertEqual(retry.returncode, 0, retry.stderr.decode())
        final = records.latest(directory)
        self.assertEqual(final["revision"], expected + 2)
        self.assertEqual(final["observations"][:-2], initial["observations"])
        self.assertEqual({x["evidence"] for x in final["observations"][-2:]},
                         {"fixture:writer-a", "fixture:writer-b"})
        replay = subprocess.run(command(loser, expected + 2), capture_output=True, timeout=20)
        self.assertNotEqual(replay.returncode, 0)
        self.assertEqual(records.latest(directory), final)
        self.assertEqual(learning.audit(self.inbox, ROOT)[1], 0)

    def test_wrong_link_missing_evidence_and_missing_inbox(self):
        self.observe()
        self.assertEqual(self.review(dict(self.row, learning_links=[]))[1], 1)
        self.assertEqual(self.review(dict(self.row, evidence=[]))[1], 1)
        self.assertEqual(feedback.review(self.row, ROOT, self.base / "absent")[1], 1)
