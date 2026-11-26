"""Integration contracts for v0.5, using source and synthetic CI records."""
import argparse
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "dev"))
from harness import ci, evaluation, profiles, records, tasks, workflow, verification
from harness.common import HarnessError

SHA = "a" * 40
REPO = "owner/repo"


def snapshot(*runs):
    return {"schema_version": 1, "repository": REPO, "head_sha": SHA, "runs": list(runs)}


def run_row(sha=SHA, conclusion="success", status="completed", run_id=1):
    return {"id": run_id, "head_sha": sha, "event": "pull_request",
            "path": ci.WORKFLOWS["quality"], "status": status, "conclusion": conclusion}


class WorkflowTests(unittest.TestCase):
    def test_ci_prior_cancelled_not_current_failure(self):
        result, code = ci.evaluate(snapshot(run_row("b" * 40, "cancelled"),
                                            run_row(run_id=2)), REPO, SHA, ["quality"])
        self.assertEqual(code, 0)
        self.assertEqual(result["ignored_other_head_or_event"], 1)
        self.assertEqual(result["checks"][0]["state"], "REPORTED_SUCCESS")

    def test_ci_missing_cancelled_ambiguous_and_identity(self):
        for rows, expected in (([], "MISSING"), ([run_row(conclusion="cancelled")], "CANCELLED"),
                               ([run_row(), run_row(run_id=2)], "AMBIGUOUS"),
                               ([run_row(conclusion="failure")], "FAIL"),
                               ([run_row(status="in_progress", conclusion=None)], "PENDING")):
            result, code = ci.evaluate(snapshot(*rows), REPO, SHA, ["quality"])
            self.assertEqual(code, 1)
            self.assertEqual(result["checks"][0]["state"], expected)
        with self.assertRaises(HarnessError):
            ci.evaluate(snapshot(), REPO, "b" * 40, ["quality"])

    def test_exact_pr_base_association(self):
        row = run_row()
        row["pull_requests"] = [{"number": 189, "base": {"sha": "c" * 40},
                                "head": {"sha": SHA}}]
        good, code = ci.evaluate(snapshot(row), REPO, SHA, ["quality"], 189, "c" * 40)
        self.assertEqual(code, 0)
        self.assertEqual(good["pr"], 189)
        bad, code = ci.evaluate(snapshot(row), REPO, SHA, ["quality"], 189, "d" * 40)
        self.assertEqual(code, 1)
        self.assertEqual(bad["checks"][0]["state"], "MISSING")

    def test_zero_discovery_refused(self):
        proc = subprocess.run([sys.executable, "dev/harness/check_runner.py",
                               "nonexistent_test_*.py"], cwd=ROOT, capture_output=True)
        self.assertNotEqual(proc.returncode, 0)
        self.assertIn(b"No tests discovered", proc.stderr)

    def test_native_profile_does_not_claim_readiness(self):
        result, code = profiles.observe_profile(ROOT, "native-lab")
        self.assertEqual(code, 1)
        self.assertEqual(result["native_capability"], "UNKNOWN")

    def test_eval_cases(self):
        result, code = evaluation.evaluate(ROOT)
        self.assertEqual(code, 0, result)
        self.assertGreaterEqual(len(result["cases"]), 3)

    def test_clean_handoff_summary_and_wrong_identity(self):
        from harness import checkpoint
        with tempfile.TemporaryDirectory() as tmp:
            parent = Path(tmp)
            root = parent / "repo"
            root.mkdir()
            (root / "dev/learning").mkdir(parents=True)
            def git(*args):
                return subprocess.check_output(["git", "-C", str(root), *args], text=True).strip()
            git("init", "-q")
            git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                "commit", "--allow-empty", "-qm", "fixture")
            sha = git("rev-parse", "HEAD")
            spec = parent / "spec.json"
            history = parent / "history"
            spec.write_text(json.dumps({"work_ref": "fixture/handoff", "objective": "test",
                                        "tasks": [{"id": "a", "acceptance": "result", "depends_on": []}]}))
            tasks.run(argparse.Namespace(action="init", directory=history, spec=spec), root)
            for expected, state in ((1, "WORKING"), (2, "RECORDED")):
                tasks.run(argparse.Namespace(action="transition", directory=history,
                          task_id="a", expect=expected, to=state, note="fixture", checkpoint=None), root)
            cp = parent / "checkpoint.json"
            checkpoint.run(argparse.Namespace(action="capture", file=cp,
                           work_ref="fixture/handoff", attempt_ref="one", objective="test",
                           stage="verify", repository=REPO, next_safe_action="review",
                           kind="local", box_uuid=None, layer=None, operation_id=[]), root)
            evidence = parent / "evidence"
            snap = parent / "ci.json"
            rows = [dict(run_row(sha=sha, run_id=i+1), path=path,
                         pull_requests=[{"number": 1, "base": {"sha": "c"*40}, "head": {"sha": sha}}])
                    for i, path in enumerate(ci.WORKFLOWS.values())]
            snap.write_text(json.dumps({"schema_version": 1, "repository": REPO,
                                        "head_sha": sha, "runs": rows}))
            args = argparse.Namespace(history=history, checkpoint=cp, evidence=[str(evidence)],
                       inbox=None, repo=REPO, ci_snapshot=snap, pr=1, base_sha="c"*40,
                       learning_disposition="no-new-asset: fixture only", export=None)
            work_history = parent / "work"
            envelope = {"kind": "work", "schema_version": 1, "feedback_protocol": 1,
                        "work_ref": "fixture/handoff", "attempt_ref": "one",
                        "task_kind": "development", "objective": "test", "risk": "standard",
                        "authority": "fixture only", "state": "RECORDED", "boundary": "completion",
                        "target": {"kind": "repository", "ref": REPO, "version": None, "source_sha": sha},
                        "next_safe_action": "review", "evidence": ["fixture"], "used_assets": [],
                        "learning": {"kind": "none", "ref": "fixture has no reusable feedback"}}
            initial = dict(envelope, state="ACTIVE", learning=None)
            initial.pop("boundary")
            records.publish(work_history, 0, initial)
            records.publish(work_history, 1, envelope)
            with patch.dict(verification.CHECKS, {"fixture": ("python", "-c", "pass")}):
                verification.execute(root, "fixture", evidence, 10)
                missing, missing_code = workflow.summary(args, root, finishing=True)
                self.assertEqual(missing_code, 1)
                self.assertIn("No structured work feedback history supplied", missing["issues"])
                args.work_history = work_history
                args.learning_disposition = None
                unreviewed, code = workflow.summary(args, root, finishing=True)
                self.assertEqual(code, 1)
                self.assertEqual(unreviewed["finish_gate"]["decision"], "REVIEW_REQUIRED")
                args.operations_reviewed = True
                args.pending_output = ["fixture-original"]
                unread, code = workflow.summary(args, root, finishing=True)
                self.assertEqual(code, 1)
                self.assertEqual(unread["finish_gate"]["decision"], "CONTINUE_OR_RESOLVE")
                self.assertEqual(unread["finish_gate"]["pending_output"], ["fixture-original"])
                args.pending_output = []
                result, code = workflow.summary(args, root, finishing=True)
                self.assertEqual(code, 0, result)
                self.assertEqual(result["finish_gate"]["decision"], "HANDOFF_READY")
                self.assertEqual(result["status"], "HANDOFF_SUMMARY")
                for field, value, message in (
                    ("attempt_ref", "other", "Feedback attempt differs from checkpoint"),
                    ("objective", "other", "Feedback objective differs from task history"),
                    ("target", dict(envelope["target"], source_sha="b"*40),
                     "Feedback target differs from exact delivery repository and source"),
                ):
                    wrong = parent / ("work-" + field)
                    changed = dict(envelope, **{field: value})
                    first = dict(changed, state="ACTIVE", learning=None)
                    first.pop("boundary")
                    records.publish(wrong, 0, first)
                    records.publish(wrong, 1, changed)
                    args.work_history = wrong
                    invalid, invalid_code = workflow.summary(args, root, finishing=True)
                    self.assertEqual(invalid_code, 1)
                    self.assertIn(message, invalid["issues"])
                args.work_history = work_history
                args.base_sha = "d"*40
                self.assertEqual(workflow.summary(args, root, finishing=True)[1], 1)
                args.repo = "wrong/repository"
                with self.assertRaises(HarnessError):
                    workflow.summary(args, root, finishing=True)

    def test_finish_links_observation_from_another_task(self):
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            history = directory / "history"
            spec = directory / "spec.json"
            spec.write_text(json.dumps({"work_ref":"current","objective":"test",
                                       "tasks":[{"id":"a","acceptance":"result","depends_on":[]}]}))
            tasks.run(argparse.Namespace(action="init", directory=history, spec=spec), ROOT)
            args = argparse.Namespace(history=history, checkpoint=None, evidence=[],
                inbox=str(directory), repo=None, ci_snapshot=None, pr=None, base_sha=None,
                learning_disposition="candidate:L1", related_candidate=["L1"], export=None)
            candidate = {"id":"L1","state":"CANDIDATE","work_ref":"prior",
                         "observations":[{"work_ref":"current"}]}
            with patch("harness.learning.audit", return_value=({"records":[candidate],"findings":[]},0)):
                result, _ = workflow.summary(args, ROOT, finishing=True)
                self.assertNotIn("Learning candidate is not present in supplied inbox", result["issues"])
                self.assertNotIn("Related learning candidate missing or not linked to work ref", result["issues"])
                args.related_candidate = ["L2"]
                result, _ = workflow.summary(args, ROOT, finishing=True)
                self.assertIn("Related learning candidate missing or not linked to work ref", result["issues"])

    def test_finish_missing_evidence_and_export(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            spec = root / "spec.json"
            history = root / "history"
            spec.write_text(json.dumps({"work_ref": "example", "objective": "test",
                                        "tasks": [{"id": "a", "acceptance": "result", "depends_on": []}]}))
            tasks.run(argparse.Namespace(action="init", directory=history, spec=spec), ROOT)
            args = argparse.Namespace(history=history, checkpoint=None, evidence=[], inbox=None,
                                      repo=None, ci_snapshot=None, pr=None, base_sha=None, learning_disposition=None,
                                      export=str(root / "export.json"))
            result, code = workflow.summary(args, ROOT, finishing=True)
            self.assertEqual(code, 1)
            self.assertIn("No local evidence supplied", result["issues"])
            self.assertEqual(result["finish_gate"]["decision"], "CONTINUE_OR_RESOLVE")
            self.assertEqual(result["finish_gate"]["ready"], ["a"])
            self.assertEqual(json.loads(Path(args.export).read_text())["work_ref"], "example")
            with self.assertRaises(FileExistsError):
                workflow.summary(args, ROOT, finishing=True)


if __name__ == "__main__":
    unittest.main()
