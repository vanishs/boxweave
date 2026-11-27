import copy
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
                             ("size_in_bytes", 65537)):
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
        g.bind_receipt(receipt, loc, EVENT)
        for field, value in (("run_id", 11), ("run_attempt", 1), ("event_sha", SOURCE),
                             ("repository", "other/repo")):
            bad = copy.deepcopy(receipt)
            bad["ci"][field] = value
            with self.assertRaises(ValueError):
                g.bind_receipt(bad, loc, EVENT)


if __name__ == "__main__":
    unittest.main()
