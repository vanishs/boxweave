"""Behavior checks for the existing-Incus bootstrap, using a fake Incus CLI.

These tests never run Incus, APT, sudo, or the installer on the test host.
"""
import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest

ROOT = Path(__file__).resolve().parents[1]
INSTALLER = (ROOT / 'install.sh').read_text(encoding='utf-8')
START = 'ensure_incus_initialized() {'
END = '\n# Refuse to run competing legacy synchronizers'
FUNC = INSTALLER[INSTALLER.index(START):INSTALLER.index(END)]


class BootstrapBehaviorTests(unittest.TestCase):
    def run_case(self, instances, has_pool):
        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp)
            cli = base / 'incus'
            cli.write_text(textwrap.dedent('''\
                #!/usr/bin/env bash
                set -e
                [[ "$1" == --force-local ]] || exit 41
                shift
                if [[ "$1" == info ]]; then exit 0; fi
                if [[ "$1" == storage && "$2" == list ]]; then
                  if [[ -f "$MOCK_POOL" ]]; then echo '[{"name":"default","driver":"dir"}]'; else echo '[]'; fi
                  exit 0
                fi
                if [[ "$1" == list && "$2" == --all-projects ]]; then cat "$MOCK_INSTANCES"; exit 0; fi
                if [[ "$1" == admin && "$2" == init && "$3" == --auto ]]; then
                  touch "$MOCK_POOL" "$MOCK_INITIALIZED"
                  exit 0
                fi
                printf 'Unexpected call: %s\\n' "$*" >&2
                exit 40
            '''))
            cli.chmod(0o755)
            instance_file = base / 'instances.json'
            instance_file.write_text(instances)
            pool = base / 'pool'
            if has_pool:
                pool.touch()
            env = dict(os.environ, PATH=str(base) + ':' + os.environ['PATH'],
                       MOCK_INSTANCES=str(instance_file), MOCK_POOL=str(pool),
                       MOCK_INITIALIZED=str(base / 'initialized'))
            script = """set -Eeuo pipefail
log() { :; }
die() { echo "$1" >&2; exit 1; }
""" + FUNC + "\nensure_incus_initialized\n"
            outcome = subprocess.run(['bash', '-c', script], env=env, text=True,
                                     capture_output=True, timeout=10)
            return outcome, (base / 'initialized').exists()

    def test_empty_host_gets_initialization(self):
        outcome, initialized = self.run_case('[]', False)
        self.assertEqual(outcome.returncode, 0, outcome.stderr)
        self.assertTrue(initialized)

    def test_existing_storage_not_reinitialized(self):
        outcome, initialized = self.run_case('[]', True)
        self.assertEqual(outcome.returncode, 0, outcome.stderr)
        self.assertFalse(initialized)

    def test_existing_containers_no_storage_refuses_changes(self):
        outcome, initialized = self.run_case('[{"name":"precious-container"}]', False)
        self.assertNotEqual(outcome.returncode, 0)
        self.assertFalse(initialized)
        self.assertIn('manual review required', outcome.stderr)


if __name__ == '__main__':
    unittest.main()
