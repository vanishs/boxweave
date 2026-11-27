"""Validate selection against private fixtures, never system sources."""
from pathlib import Path
from tempfile import TemporaryDirectory
import unittest
from pr191_ci_apt_sources import URI, optional_source, select


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


if __name__ == "__main__":
    unittest.main()
