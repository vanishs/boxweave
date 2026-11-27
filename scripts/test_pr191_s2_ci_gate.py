import copy
from contextlib import redirect_stdout
import io
import hashlib
import json
from pathlib import Path
import struct
import tempfile
from unittest import mock
import unittest

import pr191_s2_ci_gate as g


SOURCE = "a" * 40
EVENT = "b" * 40


def page(key, rows):
    return {key: rows, "total_count": len(rows)}


def fixture():
    repo = {"id": 123, "full_name": g.REPO}
    run = {"id": 10, "run_attempt": 2, "head_sha": SOURCE, "event": "pull_request",
           "path": ".github/workflows/" + g.N2, "repository": repo, "head_repository": repo,
           "pull_requests": [{"number": 191, "head": {"sha": SOURCE, "repo": repo},
                              "base": {"ref": "v1", "repo": repo}}]}
    jobs = page("jobs", [{"id": 11, "run_id": 10, "name": "installed-lifecycle (scoped)",
                          "status": "completed", "conclusion": "success", "steps": [
                              {"name": name, "status": "completed", "conclusion": "success"}
                              for name in ("Retain sanitized lane locator",
                                           "Index mapped lane only after all original lane checks")]}])
    artifacts = page("artifacts", [{"id": 12, "name": "pr191-s2-scoped-attempt-2",
                                    "expired": False, "size_in_bytes": 1024,
                                    "workflow_run": {"id": 10, "head_sha": SOURCE,
                                        "repository_id": 123, "head_repository_id": 123}}])
    return run, jobs, artifacts


class S2CIGateTests(unittest.TestCase):
    def test_exact_run_selection(self):
        run, _, _ = fixture()
        self.assertEqual(g.select_run(page("workflow_runs", [run]), g.N2, SOURCE, 191), run)
        self.assertIsNone(g.select_run(page("workflow_runs", [run]), g.S25, SOURCE, 191))
        self.assertIsNone(g.select_run(page("workflow_runs", [run]), g.N2, SOURCE, 190))
        with self.assertRaises(ValueError):
            g.select_run(page("workflow_runs", [run, run]), g.N2, SOURCE, 191)
        with self.assertRaises(ValueError):
            g.select_run({"workflow_runs": [run], "total_count": 2}, g.N2, SOURCE, 191)

    def test_job_and_artifact_provenance(self):
        run, jobs, artifacts = fixture()
        self.assertEqual(g.checked_lane(run, jobs, artifacts, "scoped", SOURCE)["artifact_id"], 12)
        for field, value in (("conclusion", "failure"), ("status", "in_progress"), ("run_id", 99)):
            bad = copy.deepcopy(jobs)
            bad["jobs"][0][field] = value
            with self.assertRaises(ValueError):
                g.checked_lane(run, bad, artifacts, "scoped", SOURCE)
        bad = copy.deepcopy(jobs)
        bad["jobs"][0]["steps"].pop()
        with self.assertRaises(ValueError):
            g.checked_lane(run, bad, artifacts, "scoped", SOURCE)
        for field, value in (("expired", True), ("name", "pr191-s2-scoped-attempt-1"),
                             ("size_in_bytes", 65537), ("size_in_bytes", 0)):
            bad = copy.deepcopy(artifacts)
            bad["artifacts"][0][field] = value
            with self.assertRaises(ValueError):
                g.checked_lane(run, jobs, bad, "scoped", SOURCE)
        bad = copy.deepcopy(artifacts)
        bad["artifacts"][0]["workflow_run"]["head_sha"] = "c" * 40
        with self.assertRaises(ValueError):
            g.checked_lane(run, jobs, bad, "scoped", SOURCE)

    def test_receipt_original_binding(self):
        loc = g.checked_lane(*fixture(), "scoped", SOURCE)
        receipt = {"report": {"lane": "scoped"}, "ci": {"repository": g.REPO,
            "event": "pull_request", "event_sha": EVENT, "run_id": 10, "run_attempt": 2}}
        receipt["ci"].update(job="installed-lifecycle",
                             workflow_ref=g.REPO+"/.github/workflows/"+g.N2+"@refs/pull/191/merge")
        g.bind_receipt(receipt, loc, EVENT, pr=191)
        for field, value in (("run_id", 11), ("run_attempt", 1), ("event_sha", SOURCE),
                             ("repository", "other/repo"), ("job", "other"),
                             ("workflow_ref", g.REPO+"/.github/workflows/"+g.N2+"@refs/pull/192/merge")):
            bad = copy.deepcopy(receipt)
            bad["ci"][field] = value
            with self.assertRaises(ValueError):
                g.bind_receipt(bad, loc, EVENT, pr=191)


    def test_current_same_repo_pr_context_and_future_pr(self):
        for pr in ("191", "207"):
            ref = "refs/pull/" + pr + "/merge"
            env = {"GITHUB_ACTIONS": "true", "RUNNER_ENVIRONMENT": "github-hosted",
                   "GITHUB_REPOSITORY": g.REPO, "GITHUB_EVENT_NAME": "pull_request",
                   "GITHUB_SHA": EVENT, "GITHUB_REF": ref, "GITHUB_JOB": "s2-native-blackbox",
                   "GITHUB_WORKFLOW_REF": g.REPO+"/.github/workflows/v1-s2-native.yml@"+ref}
            self.assertEqual(g.pr_context(env, SOURCE, pr, EVENT), int(pr))
            for field, value in (("GITHUB_EVENT_NAME", "push"), ("GITHUB_REF", "refs/heads/v1"),
                                 ("GITHUB_JOB", "other"), ("GITHUB_WORKFLOW_REF", "other/ref"),
                                 ("GITHUB_SHA", SOURCE), ("GITHUB_REPOSITORY", "other/repo")):
                with self.subTest(field=field, value=value), self.assertRaises(ValueError):
                    g.pr_context({**env, field: value}, SOURCE, pr, EVENT)
            for wrong in ("0", "001", "other", "208"):
                with self.assertRaises(ValueError):
                    g.pr_context(env, SOURCE, wrong, EVENT)
            run, _, _ = fixture()
            run["pull_requests"][0]["number"] = int(pr)
            self.assertEqual(g.select_run(page("workflow_runs", [run]), g.N2, SOURCE, int(pr)), run)
            run["head_repository"] = {"id": 999, "full_name": g.REPO}
            self.assertIsNone(g.select_run(page("workflow_runs", [run]), g.N2, SOURCE, int(pr)))
        with self.assertRaises(ValueError):
            g.bind_receipt({"report":{"lane":"scoped"}, "ci":{"repository":g.REPO,
                "event":"pull_request", "event_sha":EVENT, "run_id":10, "run_attempt":2}},
                g.checked_lane(*fixture(), "scoped", SOURCE), EVENT, pr=191)


    def test_current_workflow_trigger_parity_and_legacy_route(self):
        root = Path(".github/workflows")
        names = (g.N2, g.S25, "v1-s2-native.yml")
        sources = [(root / name).read_text() for name in names]
        paths = []
        for src in sources:
            selected = set()
            for line in src.split("    paths:\n", 1)[1].splitlines():
                if not line.startswith("      - "):
                    break
                selected.add(line.strip().removeprefix("- ").strip("'"))
            paths.append(selected)
        self.assertTrue(paths[0])
        self.assertEqual(paths[0], paths[1])
        self.assertEqual(paths[0], paths[2])
        self.assertTrue({"scripts/**", "internal/**", "cmd/**", "install.sh",
                         "uninstall.sh", "VERSION", "Makefile", "go.mod", "go.sum"} <= paths[0])
        self.assertFalse(any(p.startswith("docs/") or p == "**" for p in paths[0]))
        self.assertIn("github.event.pull_request.head.repo.full_name == github.repository", sources[1])
        self.assertNotIn("pull_request.number == 191", sources[1])
        self.assertNotIn("pull_request.number == 191", sources[2])
        self.assertIn("if: github.event_name == 'pull_request'", sources[2])
        self.assertIn("if: github.event_name != 'pull_request' && github.ref == 'refs/heads/feat/v1-s2-operations-20261110'", sources[2])
        self.assertIn("github.event.pull_request.head.sha || github.sha", sources[2])
        self.assertIsNone(g.select_run(page("workflow_runs", []), g.N2, SOURCE, 191))
        self.assertFalse((root / "v1-guided-upgrade.yml").exists())
        self.assertFalse(Path("scripts/guided_upgrade_ci.sh").exists())



RC_REF = "refs/heads/release/v" + Path("VERSION").read_text().strip() + "-rc.7"
RC_TAG = RC_REF.rsplit("/", 1)[1]


def release_env(workflow=g.RELEASE):
    return {"GITHUB_ACTIONS": "true", "RUNNER_ENVIRONMENT": "github-hosted",
            "GITHUB_REPOSITORY": g.REPO, "GITHUB_EVENT_NAME": "push",
            "GITHUB_SHA": SOURCE, "GITHUB_REF": RC_REF,
            "GITHUB_WORKFLOW_REF": g.REPO + "/.github/workflows/" + workflow + "@" + RC_REF}


def release_run(workflow=g.N2):
    run, jobs, artifacts = fixture()
    run.update(event="push", head_branch=RC_REF.removeprefix("refs/heads/"),
               path=".github/workflows/" + workflow, status="completed", conclusion="success")
    return run, jobs, artifacts


def success_step(name):
    return {"name": name, "status": "completed", "conclusion": "success"}


class ReleaseCIGateTests(unittest.TestCase):
    def test_release_context_no_pr_dispatch_or_other_branch(self):
        for workflow in (g.N2, g.RELEASE):
            env = release_env(workflow)
            self.assertEqual(g.release_context(env, SOURCE, (workflow,)), (RC_REF, RC_TAG))
            for field, value in (("GITHUB_EVENT_NAME", "pull_request"), ("GITHUB_EVENT_NAME", "workflow_dispatch"),
                                 ("GITHUB_REPOSITORY", "other/repo"), ("GITHUB_REF", "refs/heads/v1"),
                                 ("GITHUB_SHA", EVENT), ("RUNNER_ENVIRONMENT", "self-hosted"),
                                 ("GITHUB_WORKFLOW_REF", "other/workflow")):
                with self.subTest(field=field, value=value), self.assertRaises(ValueError):
                    g.release_context({**env, field: value}, SOURCE, (workflow,))
            wrong = "refs/heads/release/v1.999.999-rc.7"
            with self.assertRaises(ValueError):
                g.release_context({**env, "GITHUB_REF": wrong,
                                   "GITHUB_WORKFLOW_REF": env["GITHUB_WORKFLOW_REF"].replace(RC_REF, wrong)},
                                  SOURCE, (workflow,))

    def test_release_run_exact_event_repo_ref_attempt(self):
        run, _, _ = release_run()
        self.assertEqual(g.select_release_run(page("workflow_runs", [run]), g.N2, SOURCE, RC_REF), run)
        self.assertEqual(g.select_release_run(page("workflow_runs", [run]), g.N2, SOURCE, RC_REF, 10, 2), run)
        self.assertIsNone(g.select_release_run(page("workflow_runs", [run]), g.N2, SOURCE, RC_REF, 10, 1))
        for field, value in (("event", "pull_request"), ("head_branch", "v1"),
                             ("head_sha", EVENT), ("path", ".github/workflows/other.yml"),
                             ("head_repository", {"id": 999, "full_name": g.REPO})):
            with self.subTest(field=field):
                self.assertIsNone(g.select_release_run(page("workflow_runs", [{**run, field: value}]),
                                                      g.N2, SOURCE, RC_REF))
        with self.assertRaises(ValueError):
            g.select_release_run(page("workflow_runs", [run, run]), g.N2, SOURCE, RC_REF)


    def test_full_n2_required_including_unmapped_gates(self):
        run, _, _ = release_run()
        import re
        modes = re.search(r"^        mode: \[(.+)\]$", Path(".github/workflows/v1-n2-installed-lifecycle.yml").read_text(), re.M)[1].split(", ")
        rows = [{"id": i+100, "name": "installed-lifecycle ("+lane+")", "run_id": 10,
                 "status": "completed", "conclusion": "success",
                 "steps": [success_step(name) for name in (
                     "Require the exact protected v1 HEAD for a controlled RC producer",
                     "Require original installed lane assertions")]} for i, lane in enumerate(modes)]
        g.require_release_n2(run, page("jobs", rows))
        for i in range(len(rows)):
            with self.assertRaises(ValueError):
                g.require_release_n2(run, page("jobs", rows[:i]+rows[i+1:]))
        for field, value in (("conclusion", "failure"), ("status", "in_progress"), ("run_id", 20)):
            bad = copy.deepcopy(rows)
            bad[-1][field] = value
            with self.assertRaises(ValueError):
                g.require_release_n2(run, page("jobs", bad))
        bad = copy.deepcopy(rows)
        bad[-1]["steps"][1]["conclusion"] = "skipped"
        with self.assertRaises(ValueError):
            g.require_release_n2(run, page("jobs", bad))
        with self.assertRaises(ValueError):
            g.require_release_n2({**run, "conclusion": "failure"}, page("jobs", rows))

    def test_release_s25_current_attempt_and_receipt(self):
        run, jobs, artifacts = release_run(g.RELEASE)
        jobs["jobs"][0]["name"] = "v1-prerelease-nested-S2.5-isolated-runner"
        jobs["jobs"][0]["steps"] = [success_step(x) for x in (
            "Require the exact protected v1 HEAD before touching a disposable VM",
            "Real nested independent Agent/Incus verifier", "Retain sanitized lane locator")]
        artifacts["artifacts"][0]["name"] = "pr191-s2-s25-attempt-2"
        loc = g.checked_lane(run, jobs, artifacts, "s25", SOURCE)
        receipt = {"report": {"lane": "s25", "source": SOURCE}, "ci": {
            "repository": g.REPO, "event": "push", "event_sha": SOURCE, "run_id": 10,
            "run_attempt": 2, "job": "native-s25-gate",
            "workflow_ref": release_env()["GITHUB_WORKFLOW_REF"]}}
        g.bind_receipt(receipt, loc, SOURCE, RC_REF)
        for field, value in (("event", "pull_request"), ("run_attempt", 1), ("run_id", 20),
                             ("job", "s25-bridgehead-lab"), ("workflow_ref", "other/ref")):
            bad = copy.deepcopy(receipt)
            bad["ci"][field] = value
            with self.assertRaises(ValueError):
                g.bind_receipt(bad, loc, SOURCE, RC_REF)
        jobs["jobs"][0]["steps"][1]["conclusion"] = "skipped"
        with self.assertRaises(ValueError):
            g.checked_lane(run, jobs, artifacts, "s25", SOURCE)

    def test_manifest_and_exact_candidate_bytes(self):
        run, jobs, artifacts = release_run(g.RELEASE)
        jobs["jobs"][0]["name"] = "v1-exact-source-two-arch-manifest"
        jobs["jobs"][0]["steps"] = [success_step("Make and self-verify exact candidate manifest")]
        artifacts["artifacts"][0]["name"] = "v1-release-candidate-manifest"
        self.assertEqual(g.checked_release_manifest(run, jobs, artifacts, SOURCE)["artifact_id"], 12)
        for field, value in (("expired", True), ("size_in_bytes", 0), ("name", "wrong")):
            bad = copy.deepcopy(artifacts)
            bad["artifacts"][0][field] = value
            with self.assertRaises(ValueError):
                g.checked_release_manifest(run, jobs, bad, SOURCE)
        with tempfile.TemporaryDirectory() as root:
            directory = Path(root)
            name = "boxweave-" + RC_TAG + "-linux-amd64"
            data = bytearray(64)
            data[:6] = b"\x7fELF\x02\x01"
            struct.pack_into("<H", data, 18, 62)
            (directory / name).write_bytes(data)
            digest = hashlib.sha256(data).hexdigest()
            m = {"source": SOURCE, "tag": RC_TAG, "version": RC_TAG[1:],
                 "platform": "linux", "cgo_enabled": True,
                 "assets": {name: {"sha256": digest, "size": len(data), "goarch": "amd64"}}}
            path = directory / "release.json"
            path.write_text(json.dumps(m))
            self.assertEqual(g.release_candidate_digest(directory, RC_TAG, SOURCE), digest)
            for field, value in (("source", EVENT), ("tag", "wrong"), ("version", "wrong"), ("cgo_enabled", False)):
                path.write_text(json.dumps({**m, field: value}))
                with self.assertRaises(ValueError):
                    g.release_candidate_digest(directory, RC_TAG, SOURCE)
            path.write_text(json.dumps(m))
            data[-1] = 1
            (directory / name).write_bytes(data)
            with self.assertRaises(ValueError):
                g.release_candidate_digest(directory, RC_TAG, SOURCE)
            g.require_release_digests([{"report": {"candidate_sha256": digest}}], digest)
            with self.assertRaises(ValueError):
                g.require_release_digests([{"report": {"candidate_sha256": "b"*64}}], digest)


    def test_unmapped_release_identity_and_negative(self):
        lane = "scoped-retirement"
        run, jobs, artifacts = release_run()
        jobs["jobs"][0]["name"] = "installed-lifecycle (" + lane + ")"
        jobs["jobs"][0]["steps"] = [success_step("Retain sanitized release-only N2 candidate identity")]
        artifacts["artifacts"][0]["name"] = "pr191-n2-identity-" + lane + "-attempt-2"
        loc = g.checked_lane(run, jobs, artifacts, lane, SOURCE, identity_only=True)
        binary = "b" * 64
        raw = ("N2_INSTALL_SOURCE="+SOURCE+"\nN2_INSTALL_CANDIDATE_SHA256="+binary+
               "\nN2_INSTALLED_CANDIDATE_SHA256="+binary+"\nN2_EVIDENCE_LANE="+lane+"\n").encode()
        env = {**release_env(g.N2), "GITHUB_RUN_ID": "10", "GITHUB_RUN_ATTEMPT": "2",
               "GITHUB_JOB": "installed-lifecycle"}
        receipt = g.release_identity_receipt(raw, lane, SOURCE, env)
        g.check_release_identity(receipt, loc, SOURCE, RC_REF, binary)
        with self.assertRaises(ValueError):
            g.check_release_identity(receipt, loc, SOURCE, RC_REF, "c"*64)
        for bad in (raw.replace(lane.encode(), b"scoped-noquota"),
                    raw + ("N2_EVIDENCE_LANE="+lane+"\n").encode()):
            with self.assertRaises(ValueError):
                g.release_identity_receipt(bad, lane, SOURCE, env)
        with self.assertRaises(ValueError):
            g.release_identity_receipt(raw, lane, SOURCE, {**env,"GITHUB_EVENT_NAME":"pull_request"})
        with self.assertRaises(ValueError):
            g.checked_lane({**run,"event":"pull_request"}, jobs, artifacts, lane, SOURCE, identity_only=True)


    def test_release_consumer_complete_model_and_mismatched_unmapped_binary(self):
        # In-memory CI API and receipt model only; no real release or native effects.
        import test_pr191_s2_witness_map as w
        n2, _, _ = release_run()
        own, _, _ = release_run(g.RELEASE)
        own.update(id=20, status="in_progress", conclusion=None)
        payloads = {"git/ref/heads/v1": {"object": {"sha": SOURCE}},
                    "actions/runs/20": own,
                    "actions/workflows/" + g.N2 + "/runs?event=push&head_sha=" + SOURCE +
                    "&per_page=100": page("workflow_runs", [n2])}
        receipts = {}
        for run, lanes in ((n2, sorted((set(w.LANES)-{"s25"})|g.UNMAPPED_N2)), (own, ["s25"])):
            jobs, artifacts = [], []
            def artifact(name, i):
                return {"name": name, "id": run["id"]*1000+i, "expired": False, "size_in_bytes": 1024,
                        "workflow_run": {"id": run["id"], "head_sha": SOURCE,
                                         "repository_id": 123, "head_repository_id": 123}}
            def job(name, i, steps):
                return {"name": name, "id": run["id"]*100+i, "run_id": run["id"],
                        "status": "completed", "conclusion": "success",
                        "steps": [success_step(x) for x in steps]}
            for i, lane in enumerate(lanes):
                extra = lane in g.UNMAPPED_N2
                s25 = lane == "s25"
                steps = (["Require the exact protected v1 HEAD before touching a disposable VM",
                          "Real nested independent Agent/Incus verifier"] if s25 else
                         ["Require the exact protected v1 HEAD for a controlled RC producer",
                          "Require original installed lane assertions"])
                steps += (["Retain sanitized release-only N2 candidate identity"] if extra else
                          ["Retain sanitized lane locator", "Index mapped lane only after all original lane checks"])
                jobs.append(job("v1-prerelease-nested-S2.5-isolated-runner" if s25 else
                                "installed-lifecycle ("+lane+")", i, steps))
                artifacts.append(artifact(("pr191-n2-identity-" if extra else "pr191-s2-")+lane+"-attempt-2", i))
                env = {**release_env(g.RELEASE if s25 else g.N2), "GITHUB_RUN_ID": str(run["id"]),
                       "GITHUB_RUN_ATTEMPT": "2", "GITHUB_JOB": "native-s25-gate" if s25 else "installed-lifecycle"}
                if extra:
                    raw = ("N2_INSTALL_SOURCE="+SOURCE+"\nN2_INSTALL_CANDIDATE_SHA256="+w.BINARY+
                           "\nN2_INSTALLED_CANDIDATE_SHA256="+w.BINARY+"\nN2_EVIDENCE_LANE="+lane+"\n").encode()
                    receipts[lane] = g.release_identity_receipt(raw, lane, SOURCE, env)
                else:
                    receipts[lane] = w.ci_receipt(w.run(lane,w.fixture(lane)), env)
            if run is own:
                jobs.append(job("v1-exact-source-two-arch-manifest", 99, ["Make and self-verify exact candidate manifest"]))
                artifacts.append(artifact("v1-release-candidate-manifest", 99))
            path = "actions/runs/"+str(run["id"])
            payloads[path+"/attempts/2/jobs?per_page=100"] = page("jobs", jobs)
            payloads[path+"/artifacts?per_page=100"] = page("artifacts", artifacts)
        env = {**release_env(), "GITHUB_RUN_ID": "20", "GITHUB_RUN_ATTEMPT": "2", "GITHUB_JOB": "native-s2-gate"}
        with mock.patch.dict(g.os.environ, env, clear=True), mock.patch.object(g, "api", side_effect=payloads.__getitem__), \
                mock.patch.object(g.subprocess, "check_output", return_value=SOURCE+"\n"), \
                mock.patch.object(g, "release_candidate_digest", return_value=w.BINARY), \
                mock.patch.object(g, "download_receipt", side_effect=lambda loc, *_: receipts[loc["lane"]]), \
                redirect_stdout(io.StringIO()) as output:
            g.release_main(Path("unused-test-manifest"))
            self.assertIn("P2_R4_SCOPED_RELEASE_S2_EXACT_CI_CROSSWALK=PASS", output.getvalue())
            receipts["scoped-retirement"]["report"]["candidate_sha256"] = "c"*64
            with self.assertRaisesRegex(ValueError, "unmapped N2 release identity mismatch"):
                g.release_main(Path("unused-test-manifest"))


if __name__ == "__main__":
    unittest.main()
