"""Offline calibration of the candidate gate; mocked binary, real fixture SQLite."""
import json
from pathlib import Path
import sqlite3
import subprocess
from tempfile import TemporaryDirectory
import unittest
from unittest.mock import patch

import ci_candidate_build as gate

SOURCE = "a" * 40
VERSION = "1.0.0-rc.15"


class CandidateBuildTests(unittest.TestCase):
    def test_private_sqlite_oracle(self):
        for case in ("ok", "identity", "ready", "schema", "missing", "retired", "operation", "status"):
            with self.subTest(case=case), TemporaryDirectory() as temp:
                root = Path(temp)
                binary = root / "fixture"
                binary.touch()

                def output(args):
                    self.assertEqual(args[0], str(binary))
                    if args[1] == "build-info":
                        return json.dumps({"version": VERSION, "commit": "b"*40 if case == "identity" else SOURCE}).encode()
                    self.assertEqual(args[1:3], ["status", "-db"])
                    self.assertEqual(args[4:], ["-project", "no-project", "-incus-socket", "/unavailable-incus"])
                    path = Path(args[3])
                    self.assertEqual(path.parent.stat().st_mode & 0o777, 0o700)
                    if case == "status":
                        raise subprocess.CalledProcessError(1, args)
                    with sqlite3.connect(path) as db:
                        db.execute("CREATE TABLE schema_version (version INTEGER)")
                        db.execute("INSERT INTO schema_version VALUES (?)", (6 if case == "schema" else 7,))
                        for table in gate.REQUIRED - ({"owned_boxes"} if case == "missing" else set()):
                            db.execute(f"CREATE TABLE {table} (id INTEGER)")
                        if case == "retired":
                            db.execute("CREATE TABLE network_ipv4_retirement_old (id INTEGER)")
                        if case == "operation":
                            db.execute("INSERT INTO operations VALUES (1)")
                    return json.dumps({"ready": case == "ready"}).encode()

                with patch.object(gate.subprocess, "check_output", side_effect=output):
                    if case == "ok":
                        gate.smoke(binary, VERSION, SOURCE, root)
                    else:
                        with self.assertRaises((ValueError, subprocess.CalledProcessError)):
                            gate.smoke(binary, VERSION, SOURCE, root)
                self.assertEqual(list(root.iterdir()), [binary])


    def test_exact_build_arguments(self):
        for arch, version in (("amd64", VERSION), ("arm64", "1.0.0")):
            with self.subTest(arch=arch), TemporaryDirectory() as temp:
                binary = Path(temp) / "candidate"
                binary.touch()
                with patch.object(gate.subprocess, "check_output", side_effect=[SOURCE, arch, "1"]), \
                        patch.object(gate.subprocess, "run") as run, patch.object(gate, "smoke") as smoke:
                    gate.build(version, SOURCE, arch, binary, Path(temp))
                    self.assertEqual(run.call_args.args[0], [
                        "go", "build", "-trimpath", "-buildvcs=false", "-ldflags",
                        f"-X main.version={version} -X main.commit={SOURCE} -buildid=",
                        "-o", str(binary), "./cmd/boxweave"])
                    self.assertTrue(run.call_args.kwargs["check"])
                    self.assertEqual(run.call_args.kwargs["env"]["CGO_ENABLED"], "1")
                    smoke.assert_called_once_with(binary, version, SOURCE, Path(temp))

    def test_refuses_wrong_source_arch_cgo(self):
        for responses in (["b"*40], [SOURCE, "arm64"], [SOURCE, "amd64", "0"]):
            with self.subTest(responses=responses), patch.object(gate.subprocess, "check_output", side_effect=responses), \
                    patch.object(gate.subprocess, "run") as run:
                with self.assertRaises(ValueError):
                    gate.build(VERSION, SOURCE, "amd64", Path("candidate"), Path("."))
                run.assert_not_called()

    def test_invalid_inputs(self):
        for version, source, arch in (("bad", SOURCE, "amd64"), (VERSION, "bad", "amd64"), (VERSION, SOURCE, "386")):
            with patch.object(gate.subprocess, "check_output") as check:
                with self.assertRaises(ValueError):
                    gate.build(version, source, arch, Path("candidate"), Path("."))
                check.assert_not_called()


if __name__ == "__main__":
    unittest.main()
