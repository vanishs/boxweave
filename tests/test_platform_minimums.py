"""Lower-bound platform contract: Linux CPython >= 3.12, Ubuntu APT provisioning.

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

    def test_preinstalled_provider_does_not_require_ubuntu_label(self):
        # Host compatibility is still subject to Linux/systemd/Incus/venv
        # checks. This only checks that the OS name is not a blanket rejection.
        for distro in ({"ID": "fedora", "VERSION_ID": "42"},
                       {"ID": "debian", "VERSION_ID": "13"},
                       {"ID": "ubuntu", "VERSION_ID": "22.04"}):
            with self.subTest(distro=distro):
                self.assertTrue(online.host_can_use_provider(distro, True))
                self.assertFalse(online.host_can_use_provider(distro, False))
        self.assertTrue(online.host_can_use_provider(
            {"ID": "ubuntu", "VERSION_ID": "26.04"}, False))

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
        self.assertIn("Recommended: Ubuntu >= 24.04", script)
        self.assertIn("Linux x86_64, Python >= 3.12 and systemd required", script)
        self.assertIn("if (( PROVISION )); then", script)


if __name__ == "__main__":
    unittest.main()
