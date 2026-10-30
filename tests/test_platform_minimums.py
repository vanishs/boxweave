"""Lower-bound platform contract: Ubuntu >= 24.04, CPython >= 3.12.

These are pure checks. No host provisioning or root operations are performed.
"""
import sys
import unittest
from pathlib import Path

from bwmini import distribution, online


ROOT = Path(__file__).resolve().parents[1]


class MinimumVersionTests(unittest.TestCase):
    def test_ubuntu_minimum_is_numeric_and_inclusive(self):
        for accepted in ("24.04", "24.10", "25.04", "26.04", "27.10"):
            with self.subTest(version=accepted):
                self.assertTrue(online.supported_ubuntu(accepted))
        for rejected in ("22.04", "24.03", "23.10", "24.4", "invalid", "", None):
            with self.subTest(version=rejected):
                self.assertFalse(online.supported_ubuntu(rejected))

    def test_python_minimum_includes_new_minor_versions(self):
        for version in ((3, 12), (3, 13), (3, 14), (3, 15), (4, 0)):
            with self.subTest(version=version):
                self.assertTrue(online.supported_python(version))
        for version in ((3, 11), (3, 10), (2, 7)):
            with self.subTest(version=version):
                self.assertFalse(online.supported_python(version))

    def test_distribution_does_not_claim_cpython312_only(self):
        self.assertEqual(distribution.PLATFORM, "linux-x86_64-cpython3-min312")
        self.assertGreaterEqual(sys.version_info[:2], (3, 12))

    def test_streamed_entry_tracks_release_version(self):
        script = (ROOT / "install.sh").read_text()
        version = (ROOT / "VERSION").read_text().strip()
        self.assertIn('python3 - "' + version + '"', script)
        self.assertIn("Ubuntu >= 24.04 and Python >= 3.12", script)


if __name__ == "__main__":
    unittest.main()
