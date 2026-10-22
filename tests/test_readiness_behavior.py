"""Execute only the readiness block against fake tools, never the installer."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = (ROOT / 'install.sh').read_text(encoding='utf-8')
BLOCK = SCRIPT[SCRIPT.index('healthy=0\nstart_time='):SCRIPT.index("\nINSTALL_PHASE='MCP download")]


class ReadinessBehaviorTests(unittest.TestCase):
    def run_case(self, state):
        with tempfile.TemporaryDirectory() as tmp:
            p = Path(tmp)
            tools = {
                'curl': '#!/bin/bash\nif [[ -e "$CALL_MARK" ]]; then printf 401; else touch "$CALL_MARK"; printf 000; fi\n',
                'systemctl': '#!/bin/bash\n[[ "$1" == show ]] || exit 90\nprintf "%s\\n" "$SERVICE_STATE"\n',
                'sleep': '#!/bin/bash\nexit 0\n',
            }
            for name, text in tools.items():
                (p / name).write_text(text); (p / name).chmod(0o755)
            env = dict(os.environ, PATH=str(p) + ':' + os.environ['PATH'], CALL_MARK=str(p / 'called'), SERVICE_STATE=state)
            text = 'set -Eeuo pipefail\nlog(){ echo "$*"; }; die(){ echo "$*" >&2; exit 1; }; API_READY_TIMEOUT=5; API_URL=http://127.0.0.1:8080\n' + BLOCK
            return subprocess.run(['bash', '-c', text], env=env, capture_output=True, text=True, timeout=8)

    def test_activating_then_api_auth_response_is_ready(self):
        r = self.run_case('activating')
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertIn('HTTP 401', r.stdout)

    def test_failed_service_is_reported_before_retry(self):
        r = self.run_case('failed')
        self.assertNotEqual(r.returncode, 0)
        self.assertIn('service is failed', r.stderr)

    def test_inactive_does_not_trigger_service_start(self):
        r = self.run_case('inactive')
        self.assertNotEqual(r.returncode, 0)
        self.assertIn('service is inactive', r.stderr)


if __name__ == '__main__':unittest.main()
