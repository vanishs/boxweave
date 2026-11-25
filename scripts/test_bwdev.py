"""Offline developer CLI regression, discovered by the existing CI suite."""
import argparse
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "dev"))
from harness import checkpoint, context, environment
from harness.common import HarnessError, repo_path


class HarnessTests(unittest.TestCase):
    def test_context_index_and_skill_discovery(self):
        rows = context.assets(ROOT)
        ids = [row["id"] for row in rows]
        self.assertEqual(len(ids), len(set(ids)))
        self.assertIn("skill.task-resume", ids)

    def test_context_install(self):
        result, code = context.run(argparse.Namespace(
            path=["internal/install"], query="", limit=8, all=False), ROOT)
        self.assertEqual(code, 0)
        self.assertIn("internal/install/AGENTS.md", result["local_guides"])
        ids = [row["id"] for row in result["assets"]]
        self.assertIn("knowledge.maintenance", ids)
        self.assertIn("skill.verification", ids)
        self.assertIn("skill.box-development", ids)

    def test_escape(self):
        with self.assertRaises(HarnessError):
            repo_path(ROOT, "../../outside")
        with self.assertRaises(HarnessError):
            repo_path(ROOT, "/etc/passwd")

    def test_duplicate_checkpoint(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "checkpoint.json"
            path.write_text('{"format_version":1,"format_version":2}', encoding="utf-8")
            with self.assertRaises(HarnessError):
                checkpoint.read_checkpoint(path)

    def test_capture_and_review_separate_processes(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / "repo"
            subprocess.run(["git", "init", "-q", str(root)], check=True)
            subprocess.run(["git", "-C", str(root), "-c", "user.name=Fixture",
                            "-c", "user.email=fixture@example.invalid",
                            "commit", "--allow-empty", "-qm", "fixture"], check=True)
            runner = (
                "import sys; from pathlib import Path; from harness import checkpoint; "
                "import argparse,json; p=argparse.ArgumentParser(); checkpoint.register(p); "
                "r,c=checkpoint.run(p.parse_args(sys.argv[2:]),Path(sys.argv[1])); "
                "print(json.dumps(r)); raise SystemExit(c)"
            )
            path = Path(tmp) / "checkpoint.json"

            def invoke(*args):
                return subprocess.run([sys.executable, "-c", runner, str(root), *args],
                                      cwd=ROOT / "dev", capture_output=True, text=True)

            args = ("capture", str(path), "--work-ref", "fixture/test",
                    "--attempt-ref", "one", "--objective", "handoff",
                    "--stage", "verify", "--repository", "fixture/repo",
                    "--next-safe-action", "inspect original state")
            self.assertEqual(invoke(*args).returncode, 0)
            before = path.read_bytes()
            check = invoke("check", str(path))
            self.assertEqual(check.returncode, 0, check.stderr)
            self.assertEqual(json.loads(check.stdout)["status"], "STRUCTURE_AND_GIT_MATCH")
            self.assertNotEqual(invoke(*args).returncode, 0)
            self.assertEqual(path.read_bytes(), before)
            (root / "dirty").write_text("unsaved", encoding="utf-8")
            check = invoke("check", str(path))
            self.assertEqual(check.returncode, 1)
            self.assertEqual(json.loads(check.stdout)["status"], "REVIEW_REQUIRED")
            (root / "dirty").unlink()
            data = json.loads(before)
            data["original_operations"] = [{"operation_id": "fixture-op"}]
            path.write_text(json.dumps(data), encoding="utf-8")
            check = invoke("check", str(path))
            self.assertEqual(check.returncode, 1)
            self.assertIn("original Operations", check.stdout)

    def test_doctor_missing_base_is_blocked(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            subprocess.run(["git", "init", "-q", str(root)], check=True)
            subprocess.run(["git", "-C", str(root), "-c", "user.name=Fixture",
                            "-c", "user.email=fixture@example.invalid",
                            "commit", "--allow-empty", "-qm", "fixture"], check=True)
            result, code = environment.run(argparse.Namespace(require=["base"]), root)
            self.assertEqual(code, 1)
            self.assertIn("origin/v1", result["blockers"])

    def test_new_skill_discovered_without_index_duplication(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "dev/skills/new-skill").mkdir(parents=True)
            (root / "AGENTS.md").write_text("rules", encoding="utf-8")
            (root / "dev/INDEX.md").write_text(
                "| rules.entry | Rule | [rules](../AGENTS.md) | Existing |\n",
                encoding="utf-8")
            (root / "dev/skills/new-skill/SKILL.md").write_text(
                "---\nname: new-skill\ndescription: sample\n---\n", encoding="utf-8")
            self.assertEqual(context.assets(root)[-1]["status"], "Candidate")

    def test_cli_help(self):
        result = subprocess.run([sys.executable, str(ROOT / "dev/bwdev"), "--help"],
                                capture_output=True, text=True, check=True)
        for name in ("doctor", "context", "checkpoint"):
            self.assertIn(name, result.stdout)


if __name__ == "__main__":
    unittest.main()
