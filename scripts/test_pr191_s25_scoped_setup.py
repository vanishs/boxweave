#!/usr/bin/env python3
"""Source-only refusals for the dedicated Scoped nested setup helper."""
import os
from pathlib import Path
import subprocess
import unittest


class ScopedSetup(unittest.TestCase):
    def test_non_lab_refuses_before_any_candidate_call(self):
        helper = Path(__file__).with_name("pr191_s25_scoped_setup.sh")
        for env in ({}, {"P1B_S25_SCOPED": "1", "BW_S25_SETUP_LAYER": "outer"},
                    {"P1B_S25_SCOPED": "1", "BW_S25_SETUP_LAYER": "invalid"}):
            clean = {k: v for k, v in os.environ.items() if not k.startswith(("P1B_S25_", "BW_S25_", "GITHUB_", "RUNNER_"))}
            clean.update(env)
            with self.subTest(env=env):
                p = subprocess.run(["sh", str(helper), "invalid", "/not-a-candidate", "invalid", "invalid", "/not-a-workdir"],
                                   env=clean, capture_output=True, timeout=5)
                self.assertNotEqual(p.returncode, 0)
                self.assertNotIn(b"=PASS", p.stdout)


if __name__ == "__main__":
    unittest.main()
