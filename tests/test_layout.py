"""Static regression checks for the BoxWeave namespace and extension layout."""
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class LayoutTests(unittest.TestCase):
    def test_required_components_exist(self):
        required = (
            'install.sh', 'server/hostkeys.py', 'server/tokens.py',
            'clients/windows/setup.ps1', 'clients/macos/README.md',
            'clients/linux/README.md', 'agents/codex/README.md',
            'providers/containarium/README.md', 'docs/LOCALIZATION.md',
        )
        for name in required:
            with self.subTest(name=name):
                self.assertTrue((ROOT / name).is_file())

    def test_install_paths_and_brand(self):
        installer = (ROOT / 'install.sh').read_text(encoding='utf-8')
        self.assertIn('vanishs/boxweave', installer)
        self.assertIn('/etc/boxweave', installer)
        self.assertIn('boxweave-hostkeys.service', installer)
        self.assertIn('boxweave-token.service', installer)
        self.assertIn('/usr/local/bin/boxweave-mcp', installer)
        self.assertIn('clients/windows/setup.ps1', installer)
        self.assertIn('boxweave client --help', installer)
        self.assertNotIn('/etc/codex-containarium-bridge', installer)

    def test_windows_client_has_vendor_neutral_names_and_explicit_gate(self):
        content = (ROOT / 'clients/windows/setup.ps1').read_text(encoding='utf-8')
        self.assertIn("[string]$McpName = 'boxweave'", content)
        self.assertIn("[string]$KeyName = 'boxweave'", content)
        self.assertIn("[ValidateSet('Codex','Cursor','Claude-Code')]", content)
        self.assertIn("[ValidateSet('Containarium')]", content)
        self.assertIn("[ValidateSet('Native','WSL')]", content)
        self.assertNotIn('codex_containarium', content)

    def test_source_and_docs_are_english_only(self):
        for path in ROOT.rglob('*'):
            if path.relative_to(ROOT).parts[0] in ('outputs', '.git'):
                continue
            if path.suffix not in ('.py', '.ps1', '.sh', '.md', '.yml', '.json'):
                continue
            if any(part in ('__pycache__', '.venv', 'web-venv', 'node_modules', 'dist', 'test-results', 'playwright-report') for part in path.parts):
                continue
            with self.subTest(path=str(path.relative_to(ROOT))):
                data = path.read_text(encoding='utf-8')
                self.assertFalse(any('\u4e00' <= c <= '\u9fff' for c in data))


if __name__ == '__main__':
    unittest.main()
