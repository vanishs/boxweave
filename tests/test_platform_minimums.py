"""Platform prerequisites: no distribution-name gate, capability-gated dependencies."""
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from bwmini import distribution, online

ROOT = Path(__file__).resolve().parents[1]


class MinimumVersionTests(unittest.TestCase):
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

    def test_non_apt_external_provider_on_any_linux(self):
        with tempfile.TemporaryDirectory() as temp, \
             patch.object(online, "PG_BINARY_ROOT", Path(temp)), \
             patch.object(online.shutil, "which", return_value=None), \
             patch.object(online.importlib.util, "find_spec", return_value=object()):
            for system in ({"ID": "fedora"}, {"ID": "arch"}, {"ID": "debian"},
                           {"ID": "ubuntu", "VERSION_ID": "22.04"}):
                with self.subTest(system=system):
                    self.assertFalse(online.host_package_preflight(True, system)["apt_needed"])

    def test_missing_deps_require_apt_not_ubuntu(self):
        with tempfile.TemporaryDirectory() as temp, \
             patch.object(online, "PG_BINARY_ROOT", Path(temp)), \
             patch.object(online.importlib.util, "find_spec", return_value=object()):
            with patch.object(online.shutil, "which", return_value=None):
                with self.assertRaisesRegex(online.Failure, "APT_GET_REQUIRED"):
                    online.host_package_preflight(False, {"ID": "fedora"})
            with patch.object(online.shutil, "which", side_effect=lambda name:
                              "/usr/bin/apt-get" if name == "apt-get" else None):
                with self.assertRaisesRegex(online.Failure, "APT_REPOSITORY_SUITE_REQUIRED"):
                    online.host_package_preflight(False, {"ID": "fedora"})
                for system in ({"ID": "debian", "VERSION_CODENAME": "trixie"},
                               {"ID": "ubuntu", "VERSION_ID": "22.04", "VERSION_CODENAME": "jammy"}):
                    with self.subTest(system=system):
                        self.assertTrue(online.host_package_preflight(False, system)["apt_needed"])

    def test_preinstalled_managed_dependencies_do_not_require_apt(self):
        with tempfile.TemporaryDirectory() as temp:
            binary = Path(temp)/"16"/"bin"/"postgres"
            binary.parent.mkdir(parents=True)
            binary.write_text("fixture")
            with patch.object(online, "PG_BINARY_ROOT", Path(temp)), \
                 patch.object(online.importlib.util, "find_spec", return_value=object()), \
                 patch.object(online.shutil, "which", side_effect=lambda n:
                              "/usr/bin/incus" if n == "incus" else None):
                self.assertFalse(online.host_package_preflight(False, {"ID": "arch"})["apt_needed"])

    def test_streamed_entry_tracks_release_without_distro_name_gate(self):
        script = (ROOT / "install.sh").read_text()
        source_version = (ROOT / "VERSION").read_text().strip()
        published_version = (ROOT / "PUBLISHED_RELEASE").read_text().strip()
        self.assertRegex(published_version, r"^0\.[0-9]+\.[0-9]+$")
        self.assertLessEqual(tuple(map(int, published_version.split("."))),
                             tuple(map(int, source_version.split("."))))
        self.assertIn('python3 - "' + published_version + '"', script)
        # Unreleased source must not switch streamed installs to missing assets.
        if published_version != source_version:
            self.assertNotIn('python3 - "' + source_version + '"', script)

        self.assertIn("Recommended: Ubuntu >= 24.04", script)
        self.assertNotIn('"$ID" == ubuntu', script)
        self.assertNotIn("Provider provisioning requires Ubuntu", script)
        code = (ROOT / "bwmini/online.py").read_text()
        self.assertNotIn('AUTOMATIC_HOST_PROVISIONING_REQUIRES_UBUNTU', code)
        self.assertNotIn('host_can_use_provider(', code)
        self.assertNotIn('INCUS_PACKAGE =', code)
        self.assertNotIn('INCUS_PIN_CHANGED_REVIEW_REQUIRED', code)


if __name__ == "__main__":
    unittest.main()
