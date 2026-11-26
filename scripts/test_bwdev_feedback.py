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
from harness import feedback, learning, records, tasks, work, workflow
from harness.common import HarnessError


class FeedbackTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.base = Path(self.tmp.name)
        self.inbox = self.base / "inbox"
        self.inbox.mkdir()
        shutil.copytree(ROOT / "dev/learning/BDH-L014", self.inbox / "BDH-L014")
        # A resolved learning history requires its paired canonical episode records.
        shutil.copytree(ROOT / "docs/development/BDH_V07_EPISODES", self.base / "episodes")
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

    def use_history_prefix(self, revision):
        # Pin lifecycle-sensitive cases to immutable learning revisions, not
        # the moving current candidate. The canonical histories are untouched.
        source = ROOT / "dev/learning/BDH-L014"
        self.inbox = self.base / f"inbox-{revision}"
        target = self.inbox / "BDH-L014"
        target.mkdir(parents=True)
        for number in range(1, revision + 1):
            name = f"{number:08d}.json"
            shutil.copyfile(source / name, target / name)

    def observe(self, residual=None):
        args = argparse.Namespace(action="observe", inbox=self.inbox, id="BDH-L014",
                expect=records.latest(self.inbox / "BDH-L014")["revision"], work_ref=self.row["work_ref"],
                attempt_ref=self.item["attempt_ref"], evidence=self.item["evidence"],
                observation=self.item["observation"], outcome=self.item["outcome"])
        if residual is not None:
            file = self.base / "residual.json"
            file.write_text(json.dumps(residual))
            args.residual_file = file
        learning.run(args, ROOT)

    def test_actual_link_required_and_exact_match(self):
        # Revision 33 has an outstanding second selection.
        self.use_history_prefix(33)
        result, code = self.review()
        self.assertEqual(code, 1)
        self.assertEqual(result["items"][0]["related"][0]["id"], "BDH-L014")
        self.observe()
        self.assertEqual(self.review()[0]["status"], "FEEDBACK_RECORDED")
        evolve = self.review()[0]["evolve"]
        self.assertEqual(evolve["status"], "PENDING")
        self.assertEqual(evolve["items"][0]["id"], "BDH-L014")
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
        updated = dict(self.row, asset_feedback=[dict(self.item, decision="pending", candidate=None,
                       correction_ref="fixture:review")], evidence=self.row["evidence"]+["fixture:review"])
        self.assertEqual(work.advance(old, updated, 1)["state"], "HANDOFF")

    def test_boundary_keeps_pending_but_refuses_false_completion(self):
        self.use_history_prefix(33)
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
        handoff = records.latest(history)["evolve_handoff"]
        self.assertEqual(handoff["status"], "PENDING")
        self.assertEqual(handoff["items"][0]["status"], "SELECTION_RECORDED")
        self.assertEqual(work.read_history(history)["state"], "RECORDED")

    def test_reopened_history_unrelated_helped_is_not_pending(self):
        # Revision 38 includes the second independent REOPEN.
        self.use_history_prefix(38)
        self.observe()
        result, code = self.review()
        self.assertEqual(code, 0, result)
        self.assertEqual(result["status"], "FEEDBACK_RECORDED")
        self.assertEqual(result["items"][0]["related"][0]["opportunity"]["status"],
                         "REVIEW_DUE")
        self.assertEqual(result["evolve"]["status"], "NO_MATCHED_PENDING")
        self.assertEqual(result["evolve"]["items"], [])

    def test_reopened_history_matching_residual_is_pending(self):
        self.use_history_prefix(38)
        self.observe({"signature": "fixture-residual", "impact": "friction",
                      "environment": None, "cost": None, "uncertainty": "fixture"})
        result, code = self.review()
        self.assertEqual(code, 0, result)
        self.assertEqual(result["status"], "FEEDBACK_RECORDED")
        self.assertEqual(result["evolve"]["status"], "PENDING")
        self.assertEqual(result["evolve"]["items"][0]["status"], "REVIEW_DUE")
        self.assertEqual(result["evolve"]["items"][0]["id"], "BDH-L014")

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
        with self.assertRaises(HarnessError):
            work.advance(records.latest(history), revised, 1)
        revised["asset_feedback"][0]["correction_ref"] = "fixture:reviewed incorrect proposal"
        revised["evidence"] = [*self.row["evidence"], "fixture:reviewed incorrect proposal"]
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
        # Exercise actual asset drift without forging a new attachment event.
        # A current bound implementation has immutable event identity.
        root = self.base / "drift-repo"
        subprocess.run(["git", "clone", "-q", "--no-hardlinks", str(ROOT), str(root)],
                       check=True, capture_output=True)
        asset = root / original["asset"]
        asset.write_bytes((ROOT / original["asset"]).read_bytes() + b"\nFixture asset drift.\n")
        row = dict(self.row, learning={"kind": "improved", "ref": "BDH-L014"})
        result, code = feedback.review(work.validate(row), root, self.inbox)
        self.assertEqual(code, 1)
        self.assertIn("Attached implementation drifted", " ".join(result["issues"]))
        # A historic observation remains a fact even if the implementation later changes.
        self.assertEqual(feedback.review(work.validate(self.row), root, self.inbox)[1], 0)
        if "binding" in original["implementation"]:
            records.publish(directory, original["revision"],
                            dict(original, implementation=dict(original["implementation"], asset_sha256="0"*64)))
            result, code = self.review(row)
            self.assertEqual(code, 1)
            self.assertIn("Implementation binding does not identify its attachment event", " ".join(result["issues"]))

    def test_real_witness_cannot_be_relabelled_or_unlinked(self):
        original = json.loads((ROOT / "docs/development/BDH_V06_INDEPENDENT_TRANSFER_FEEDBACK.json").read_text())
        self.assertEqual(self.review(original)[1], 0)
        improved = dict(original, learning={"kind": "improved", "ref": "BDH-L014"})
        self.assertEqual(self.review(improved)[1], 1)
        unlinked = dict(original, state="RECORDED", boundary="completion",
                        learning={"kind":"none","ref":"no-new-asset"}, learning_links=[],
                        asset_feedback=[dict(original["asset_feedback"][0], decision="not-applicable",
                                             candidate=None, reason="unrelated"),
                                        original["asset_feedback"][1]])
        self.assertEqual(self.review(unlinked)[1], 1)
        # An invented correction reference cannot erase an exact recorded observation.
        unlinked["asset_feedback"][0]["correction_ref"] = "fixture:review"
        unlinked["evidence"] = [*original["evidence"], "fixture:review"]
        self.assertEqual(self.review(unlinked)[1], 1)
        omitted = dict(unlinked, asset_feedback=[], used_assets=[])
        self.assertEqual(self.review(omitted)[1], 1)
        self.assertIn("Recorded work observation missing", " ".join(self.review(omitted)[0]["issues"]))
        # A directly imported terminal history must fail finish too, not only
        # the guarded work boundary. Other delivery checks are deliberately absent.
        spec = self.base / "spec.json"
        spec.write_text(json.dumps({"work_ref":original["work_ref"],"objective":original["objective"],
                                    "tasks":[{"id":"a","acceptance":"fixture","depends_on":[]}]}))
        task_history = self.base / "tasks"
        tasks.run(argparse.Namespace(action="init",directory=task_history,spec=spec),ROOT)
        for bad in (improved, unlinked):
            bad = dict(bad, state="RECORDED", boundary="completion")
            history = self.base / ("bad-" + bad["learning"]["kind"])
            initial = dict(original, state="ACTIVE", learning=None, asset_feedback=[],
                           learning_links=[])
            initial.pop("boundary")
            records.publish(history, 0, initial)
            f = self.base / "boundary.json"
            f.write_text(json.dumps(bad))
            result, code = work.run(argparse.Namespace(action="boundary", directory=history,
                            file=f, expect=1, inbox=self.inbox), ROOT)
            self.assertEqual(code, 1)
            self.assertFalse(result["published"])
            self.assertEqual(records.latest(history)["revision"], 1)
            # Import the bad boundary to exercise finish's independent reconciliation.
            records.publish(history, 1, bad)
            args = argparse.Namespace(history=task_history,checkpoint=None,evidence=[],
                      inbox=str(self.inbox),repo=None,ci_snapshot=None,pr=None,base_sha=None,
                      learning_disposition=None,related_candidate=[],export=None,work_history=history)
            summary, finish_code = workflow.summary(args,ROOT,finishing=True)
            self.assertEqual(finish_code,1)
            self.assertEqual(summary["feedback"]["status"],"REVIEW_REQUIRED")
            self.assertIn("Material learning feedback needs review",summary["issues"])

    def test_later_attempt_does_not_invalidate_prior_snapshot(self):
        self.observe()
        self.assertEqual(self.review()[1], 0)
        directory = self.inbox / "BDH-L014"
        r = records.latest(directory)
        args = argparse.Namespace(action="observe", inbox=self.inbox, id="BDH-L014",
                    expect=r["revision"], work_ref=self.row["work_ref"], attempt_ref="future",
                    evidence="fixture:future", observation="New attempt", outcome="helped")
        learning.run(args, ROOT)
        self.assertEqual(self.review()[1], 0)
        omitted = dict(self.row, attempt_ref="future")
        self.assertEqual(self.review(omitted)[1], 1)

    def test_bound_improvement_and_explicit_later_attempt(self):
        self.observe()
        root = self.base / "repo"
        subprocess.run(["git", "clone", "-q", "--no-hardlinks", str(ROOT), str(root)],
                       check=True, capture_output=True)
        def git(*args):
            return subprocess.check_output(["git","-C",str(root),*args],text=True).strip()
        directory = self.inbox / "BDH-L014"
        revision = records.latest(directory)["revision"]
        # The fixture starts at the same asset bytes as the copied learning
        # history, even when the source under test is an edited worktree.
        asset = root / "dev/skills/box-development/SKILL.md"
        asset.write_bytes((ROOT / "dev/skills/box-development/SKILL.md").read_bytes())
        if git("status", "--porcelain"):
            git("add", "dev/skills/box-development/SKILL.md")
            git("-c","user.name=Fixture","-c","user.email=fixture@example.invalid","commit","-qm","fixture baseline")
        # Historical attachment bytes can differ after a legitimate source
        # retirement. Establish a real fixture-only attachment to the clean
        # local source before testing unchanged-asset refusal; preserve the
        # immutable imported history and never alter canonical records.
        current = learning.asset_digest(root, "dev/skills/box-development/SKILL.md")
        if current != records.latest(directory)["implementation"]["asset_sha256"]:
            baseline = argparse.Namespace(action="attach", inbox=self.inbox, id="BDH-L014",
                expect=revision, implementation_ref="fixture:baseline",
                validation_evidence="fixture:baseline evidence", work_ref=self.row["work_ref"],
                attempt_ref=self.item["attempt_ref"])
            revision = learning.run(baseline, root)[0]["record"]["revision"]
        self.assertEqual(current, records.latest(directory)["implementation"]["asset_sha256"])
        args = argparse.Namespace(action="attach", inbox=self.inbox, id="BDH-L014",
                  expect=revision, implementation_ref="fixture:new patch",
                  validation_evidence="fixture:new test result", work_ref=self.row["work_ref"],
                  attempt_ref=self.item["attempt_ref"])
        with self.assertRaises(HarnessError):
            learning.run(args, root)
        asset = root / "dev/skills/box-development/SKILL.md"
        asset.write_text(asset.read_text()+"\nNew bounded fixture improvement.\n")
        with self.assertRaises(HarnessError):
            learning.run(args, root)
        git("add",".")
        git("-c","user.name=Fixture","-c","user.email=fixture@example.invalid","commit","-qm","improve")
        sha = git("rev-parse","HEAD")
        new_evidence = args.validation_evidence
        args.validation_evidence = records.latest(directory)["implementation"]["evidence_ref"]
        with self.assertRaises(HarnessError):
            learning.run(args, root)
        args.validation_evidence = new_evidence
        attached = learning.run(args, root)[0]["record"]
        claim = {"candidate_revision": attached["revision"], "implementation_ref": args.implementation_ref,
                 "validation_evidence": args.validation_evidence, "handoff_ref": None}
        row = dict(self.row, target=dict(self.row["target"], source_sha=sha),
                   learning={"kind":"improved","ref":"BDH-L014"}, implementation_claim=claim,
                   evidence=[*self.row["evidence"], args.implementation_ref, args.validation_evidence])
        def review(r):
            return feedback.review(work.validate(r),root,self.inbox)
        self.assertEqual(review(row)[1],0,review(row))
        for key,value in (("candidate_revision",revision),("implementation_ref","wrong"),("validation_evidence","wrong")):
            self.assertEqual(review(dict(row,implementation_claim=dict(claim,**{key:value})))[1],1)
        self.assertEqual(review(dict(row,target=dict(row["target"],source_sha="a"*40)))[1],1)
        later = dict(row,attempt_ref="later")
        self.assertEqual(review(later)[1],1)
        later["evidence"] = [*row["evidence"], "fixture:handoff review"]
        later["implementation_claim"] = dict(claim,handoff_ref="fixture:handoff review")
        self.assertEqual(review(later)[1],0)
        args.expect = attached["revision"]
        with self.assertRaises(HarnessError):
            learning.run(args,root)

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
        baseline_findings = learning.audit(self.inbox, ROOT)[0]["findings"]

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
        self.assertEqual(learning.audit(self.inbox, ROOT)[0]["findings"], baseline_findings)

    def test_wrong_link_missing_evidence_and_missing_inbox(self):
        self.observe()
        self.assertEqual(self.review(dict(self.row, learning_links=[]))[1], 1)
        self.assertEqual(self.review(dict(self.row, evidence=[]))[1], 1)
        self.assertEqual(feedback.review(self.row, ROOT, self.base / "absent")[1], 1)
