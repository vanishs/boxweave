"""Validate selection against private fixtures, never system sources."""
from pathlib import Path
from tempfile import TemporaryDirectory
import unittest
from unittest.mock import patch
import subprocess
from pr191_ci_apt_sources import (URI, optional_source, select, verify_incus_key,
                                 incus_package, ZABBLY_FINGERPRINT)


class AptSources(unittest.TestCase):
    def test_exact_optional_source(self):
        self.assertTrue(optional_source("deb [arch=amd64 signed-by=/etc/key.gpg] "+URI+" noble main"))
        self.assertFalse(optional_source("deb "+URI+" jammy main"))
        self.assertFalse(optional_source("deb "+URI+" noble main\ndeb https://archive.ubuntu.com/ubuntu noble main"))

    def test_select(self):
        with TemporaryDirectory() as d:
            root=Path(d)
            parts=root/"sources.list.d"
            parts.mkdir()
            (parts/"ubuntu.sources").write_text("URIs: https://archive.ubuntu.com/ubuntu\n")
            self.assertEqual(select(root),[])
            p=parts/"microsoft-prod.list"
            p.write_text("deb "+URI+" noble main\n")
            self.assertEqual(select(root),[p])
            (parts/"mixed.sources").write_text("URIs: "+URI+"\n")
            with self.assertRaises(ValueError):
                select(root)

    def test_exact_first_key_and_producer_failure(self):
        row = "fpr:::::::::" + ZABBLY_FINGERPRINT + ":"
        for text, good in ((row, True), ("pub::::::::::", False),
                           ("fpr:::::::::" + "0"*40 + ":\n" + row, False),
                           ("fpr:short", False)):
            with self.subTest(text=text), patch("pr191_ci_apt_sources.subprocess.check_output", return_value=text) as cmd:
                if good:
                    verify_incus_key("/fixture/key.asc")
                else:
                    with self.assertRaises(ValueError):
                        verify_incus_key("/fixture/key.asc")
                cmd.assert_called_once_with(["gpg", "--show-keys", "--with-colons", "/fixture/key.asc"], text=True)
        with patch("pr191_ci_apt_sources.subprocess.check_output", side_effect=subprocess.CalledProcessError(2, "gpg")):
            with self.assertRaises(subprocess.CalledProcessError):
                verify_incus_key("/fixture/key.asc")

    def test_first_matching_package_and_producer_failure(self):
        rows = ("incus | 1:7.7-1 | https://pkgs.zabbly.com/incus/stable noble/main amd64 Packages\n"
                "incus | 1:7.6-8 | https://archive.ubuntu.com/ubuntu noble/main amd64 Packages\n"
                "incus | 1:7.6-3 | https://pkgs.zabbly.com/incus/stable noble/main amd64 Packages\n"
                "incus | 1:7.6-2 | https://pkgs.zabbly.com/incus/stable noble/main amd64 Packages\n")
        with patch("pr191_ci_apt_sources.subprocess.check_output", return_value=rows) as cmd:
            self.assertEqual(incus_package(), "1:7.6-3")
            cmd.assert_called_once_with(["apt-cache", "madison", "incus"], text=True)
        for text in ("", "\n".join(rows.splitlines()[:2])):
            with patch("pr191_ci_apt_sources.subprocess.check_output", return_value=text):
                with self.assertRaises(ValueError):
                    incus_package()
        with patch("pr191_ci_apt_sources.subprocess.check_output", side_effect=subprocess.CalledProcessError(2, "apt-cache")):
            with self.assertRaises(subprocess.CalledProcessError):
                incus_package()


if __name__ == "__main__":
    unittest.main()
