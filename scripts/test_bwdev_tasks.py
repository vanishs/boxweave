"""Task DAG and local record concurrency regressions."""
import argparse
import concurrent.futures
import json
import sys
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "dev"))
from harness import records, tasks
from harness.common import HarnessError


class TaskTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.spec = self.root / "spec.json"
        self.directory = self.root / "history"
        self.parser = argparse.ArgumentParser()
        tasks.register(self.parser)
        self.spec.write_text(json.dumps({
            "work_ref": "test", "objective": "test tasks",
            "tasks": [{"id": "a", "acceptance": "a test", "depends_on": []},
                      {"id": "b", "acceptance": "b test", "depends_on": ["a"]}],
        }), encoding="utf-8")

    def invoke(self, action, *args):
        return tasks.run(self.parser.parse_args([action, str(self.directory), *args]), ROOT)[0]

    def test_dependency_and_handoff(self):
        self.assertEqual(self.invoke("init", str(self.spec))["ready"], ["a"])
        with self.assertRaises(HarnessError):
            self.invoke("transition", "b", "--expect", "1", "--to", "WORKING", "--note", "start")
        self.invoke("transition", "a", "--expect", "1", "--to", "WORKING", "--note", "start")
        self.invoke("transition", "a", "--expect", "2", "--to", "RECORDED", "--note", "fixture evidence")
        self.assertEqual(self.invoke("next")["ready"], ["b"])
        with self.assertRaises(HarnessError):
            self.invoke("transition", "b", "--expect", "2", "--to", "WORKING", "--note", "stale")
        self.invoke("transition", "b", "--expect", "3", "--to", "WORKING", "--note", "start")
        with self.assertRaises(HarnessError):
            self.invoke("transition", "a", "--expect", "4", "--to", "PENDING", "--note", "reopen")
        self.assertEqual(records.latest(self.directory)["revision"], 4)

    def test_next_does_not_stop_at_local_blocker(self):
        spec=json.loads(self.spec.read_text())
        spec["tasks"].append({"id":"independent","acceptance":"separate required proof","depends_on":[]})
        self.spec.write_text(json.dumps(spec))
        self.invoke("init",str(self.spec))
        self.invoke("transition","a","--expect","1","--to","BLOCKED","--note","specific unavailable dependency")
        r=self.invoke("next")
        self.assertEqual(r["continuation"]["decision"],"CONTINUE")
        self.assertEqual(r["continuation"]["actionable"],[{"id":"independent","state":"PENDING","acceptance":"separate required proof"}])
        self.assertEqual(r["continuation"]["blocked"],[{"id":"a","note":"specific unavailable dependency"}])
        self.assertEqual(r["continuation"]["waiting"],[{"id":"b","on":["a"]}])
        self.invoke("transition","independent","--expect","2","--to","WORKING","--note","started")
        r=self.invoke("next")
        self.assertEqual(r["continuation"]["decision"],"CONTINUE")
        self.assertEqual(r["continuation"]["actionable"][0]["state"],"WORKING")
        self.invoke("transition","independent","--expect","3","--to","RECORDED","--note","observed proof")
        r=self.invoke("next")
        self.assertEqual(r["continuation"]["decision"],"REVIEW_BLOCKERS")
        self.assertFalse(r["continuation"]["actionable"])
        self.assertEqual(r["revision"],4)
        self.assertEqual(records.latest(self.directory)["revision"],4)

    def test_next_all_recorded_requires_exit_review(self):
        self.invoke("init",str(self.spec))
        for i,(key,state) in enumerate((("a","WORKING"),("a","RECORDED"),("b","WORKING"),("b","RECORDED")),1):
            self.invoke("transition",key,"--expect",str(i),"--to",state,"--note","fixture proof")
        r=self.invoke("next")
        self.assertEqual(r["continuation"]["decision"],"REVIEW_EXIT")
        self.assertFalse(r["continuation"]["actionable"])
        self.assertFalse(r["continuation"]["waiting"])
        self.assertEqual(r["continuation"]["recorded_count"],2)

    def test_replan_retains_acceptance_and_revision(self):
        self.invoke("init", str(self.spec))
        spec = json.loads(self.spec.read_text())
        spec["tasks"].append({"id": "c", "acceptance": "c test", "depends_on": ["b"]})
        self.spec.write_text(json.dumps(spec))
        self.assertEqual(self.invoke("replan", str(self.spec), "--expect", "1", "--note", "add case")["revision"], 2)
        spec["tasks"][0]["acceptance"] = "weakened"
        self.spec.write_text(json.dumps(spec))
        with self.assertRaises(HarnessError):
            self.invoke("replan", str(self.spec), "--expect", "2", "--note", "invalid")
        self.assertEqual(records.latest(self.directory)["revision"], 2)

    def test_cycle(self):
        spec = json.loads(self.spec.read_text())
        spec["tasks"][0]["depends_on"] = ["b"]
        self.spec.write_text(json.dumps(spec))
        with self.assertRaises(HarnessError):
            self.invoke("init", str(self.spec))
        self.assertFalse(self.directory.exists())

    def test_concurrent_revision(self):
        records.publish(self.directory, 0, {"value": 0})

        def write(value):
            try:
                return records.publish(self.directory, 1, {"value": value})["value"]
            except HarnessError:
                return None

        with concurrent.futures.ThreadPoolExecutor(2) as pool:
            results = list(pool.map(write, [1, 2]))
        self.assertEqual(results.count(None), 1)
        self.assertEqual(records.latest(self.directory)["revision"], 2)
        self.assertEqual(records.read_json(self.directory / "00000001.json")["value"], 0)

    def test_separate_process_conflict(self):
        records.publish(self.directory, 0, {"value": 0})
        runner = (
            "import sys; from harness.records import publish; "
            "from harness.common import HarnessError; "
            "publish(sys.argv[1], 1, {'value': sys.argv[2]})"
        )
        first = (self.directory / "00000001.json").read_bytes()
        procs = [subprocess.Popen([sys.executable, "-c", runner,
                                   str(self.directory), str(i)],
                                  cwd=ROOT / "dev", stdout=subprocess.PIPE,
                                  stderr=subprocess.PIPE) for i in range(2)]
        for proc in procs:
            proc.communicate(timeout=10)
        self.assertEqual(sorted(p.returncode for p in procs), [0, 1])
        self.assertEqual(records.latest(self.directory)["revision"], 2)
        self.assertEqual((self.directory / "00000001.json").read_bytes(), first)

    def test_corrupt_prior_revision_rejected(self):
        records.publish(self.directory, 0, {"value": 0})
        records.publish(self.directory, 1, {"value": 1})
        (self.directory / "00000001.json").write_text('{"revision":9}')
        with self.assertRaises(HarnessError):
            records.latest(self.directory)

    def test_corrupt_history_rejected(self):
        records.publish(self.directory, 0, {"value": 0})
        (self.directory / "00000003.json").write_text('{"revision":3}')
        with self.assertRaises(HarnessError):
            records.latest(self.directory)


if __name__ == "__main__":
    unittest.main()
