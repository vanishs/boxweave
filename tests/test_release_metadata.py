"""Release cross-file version and public README contract regression."""
import json
from pathlib import Path
import tempfile
import unittest
from scripts.check_knowledge import release_errors

class ReleaseMetadataTests(unittest.TestCase):
    def test_readme_puts_real_installation_before_developer_fixture(self):
        readme = (Path(__file__).resolve().parents[1] / 'README.md').read_text(encoding='utf-8')
        install = readme.index('## Install, update, or repair (real Provider)')
        development = readme.index('## Development & testing')
        fixture = readme.index('### Isolated fixture (synthetic)')
        self.assertLess(install, development)
        self.assertLess(development, fixture)
        self.assertIn('bash install.sh plan', readme[install:development])
        self.assertIn('bash install.sh apply', readme[install:development])
        self.assertIn('not a real installation', readme[fixture:])
        self.assertIn('without contacting or changing a real Incus/Containarium Provider', readme[fixture:])

    def test_versions_agree_and_bad_lock_is_rejected(self):
        with tempfile.TemporaryDirectory() as root:
            root = Path(root)
            (root / 'VERSION').write_text('0.13.0\n')
            (root / 'pyproject.toml').write_text('[project]\nversion = "0.13.0"\n')
            (root / 'web').mkdir()
            (root / 'web/package.json').write_text(json.dumps({'version':'0.13.0'}))
            (root / 'web/package-lock.json').write_text(json.dumps({'version':'0.13.0','packages':{'':{'version':'0.13.0'}}}))
            (root / 'README.md').write_text('0.13.0 Phase 7 Windows Codex ZCode WSL macOS')
            self.assertEqual(release_errors(root),[])
            (root / 'web/package-lock.json').write_text(json.dumps({'version':'0.13.0','packages':{'':{'version':'0.12.0'}}}))
            self.assertTrue(any('version mismatch' in e for e in release_errors(root)))

    def test_removed_old_phase_status_does_not_reappear(self):
        with tempfile.TemporaryDirectory() as root:
            root = Path(root)
            (root / 'VERSION').write_text('0.13.0\n')
            (root / 'pyproject.toml').write_text('[project]\nversion = "0.13.0"\n')
            (root / 'web').mkdir()
            (root / 'web/package.json').write_text('{"version":"0.13.0"}')
            (root / 'web/package-lock.json').write_text('{"version":"0.13.0","packages":{"":{"version":"0.13.0"}}}')
            (root / 'README.md').write_text('0.13.0 Phase7 has not started Windows Codex ZCode WSL macOS')
            self.assertTrue(any('historical phase' in e for e in release_errors(root)))

if __name__ == '__main__':
    unittest.main()
