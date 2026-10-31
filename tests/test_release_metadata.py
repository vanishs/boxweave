"""Release cross-file version and public README contract regression."""
import json
from pathlib import Path
import tempfile
import unittest
from scripts.check_knowledge import release_errors, readme_narrative_errors

class ReleaseMetadataTests(unittest.TestCase):
    def test_public_readme_order_and_terminology(self):
        from pathlib import Path
        text = (Path(__file__).resolve().parents[1] / 'README.md').read_text()
        self.assertLess(text.index('## Install (Ubuntu 24.04+ recommended)'), text.index('## Connect an AI coding agent'))
        self.assertLess(text.index('## Connect an AI coding agent'), text.index('## Common management commands'))
        self.assertIn('curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash', text)
        self.assertIn('| Client platform | Agent | Transport | Support |', text)
        self.assertIn('| Command | Purpose |', text)
        self.assertNotIn('## Isolated fixture', text)
        self.assertEqual(readme_narrative_errors(text), [])

    def test_technical_archive_and_protocol_words_are_allowed(self):
        text = """BoxWeave 0.14.0 installs from a release archive.
Use two-phase verification and the upstream API v2 where applicable.
A vendor v2 application protocol is a technical reference, not this product name.
Exact managed identifiers such as `boxweave-v2-service` remain compatible.
```sh
bash install.sh apply --archive /reviewed/boxweave.zip
```
"""
        self.assertEqual(readme_narrative_errors(text), [])

    def test_current_readme_rejects_obsolete_narratives(self):
        for text in ('Phase 1 implementation', 'Phase7 acceptance',
                     'archived 0.5.1', 'archive 0.5.1', '# BoxWeave v2',
                     'This is the v2 application.'):
            with self.subTest(text=text):
                self.assertTrue(readme_narrative_errors(text))

    def test_versions_agree_and_bad_lock_is_rejected(self):
        with tempfile.TemporaryDirectory() as root:
            root = Path(root)
            (root / 'VERSION').write_text('0.13.0\n')
            (root / 'pyproject.toml').write_text('[project]\nversion = "0.13.0"\n')
            (root / 'web').mkdir()
            (root / 'web/package.json').write_text(json.dumps({'version':'0.13.0'}))
            (root / 'web/package-lock.json').write_text(json.dumps({'version':'0.13.0','packages':{'':{'version':'0.13.0'}}}))
            (root / 'README.md').write_text(
                '0.13.0\n## Connect an AI coding agent\n'
                'Antigravity Claude Codex Copilot Cursor OpenCode Qoder TRAE ZCode\n'
                'Linux macOS Windows Native OpenSSH WSL\n')
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
            (root / 'README.md').write_text(
                '0.13.0 Phase7 has not started\n## Connect an AI coding agent\n'
                'Antigravity Claude Codex Copilot Cursor OpenCode Qoder TRAE ZCode\n'
                'Linux macOS Windows Native OpenSSH WSL\n')
            self.assertTrue(any('historical phase' in e for e in release_errors(root)))

if __name__ == '__main__':
    unittest.main()
