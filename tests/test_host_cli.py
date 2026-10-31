"""Host CLI contracts: no privileged host changes, network or credential access."""
import io
import json
import os
import subprocess
import sys
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from bwmini import host_cli


def fixture():
    return {"schema_version": 1, "phase": "ready", "version": "0.17.9",
            "application_user": "boxweave", "app_prefix": "/home/boxweave/boxweave",
            "web_port": 18640, "host": "10.0.0.5", "port": 22,
            "server_key": "ssh-ed25519 public", "managed_public_key": "ssh-ed25519 public"}


class HostCliTests(unittest.TestCase):
    def test_installed_style_isolated_interpreter_client_help(self):
        # Exactly the production -I invocation; normal imports in unittest
        # masked this regression in 0.17.9. No root, host state or SSH needed.
        root = Path(__file__).resolve().parents[1]
        env = dict(os.environ, PYTHONPATH=str(root / "bwmini"))
        process = subprocess.run([sys.executable, "-I", str(root / "bwmini/host_cli.py"),
                                  "client", "--help"],
                                 cwd="/", env=env, capture_output=True, text=True, timeout=20)
        self.assertEqual(process.returncode, 0, process.stderr)
        self.assertIn("--admin-user", process.stdout)
        self.assertNotIn("ModuleNotFoundError", process.stderr)
        self.assertNotIn("Traceback", process.stderr)

    def test_host_cli_never_invokes_app_owned_root_interpreter(self):
        online = Path("bwmini/online.py").read_text()
        self.assertIn("exec /usr/bin/python3 -I /usr/local/lib/boxweave/default/host_cli.py", online)
        self.assertNotIn("b'cd /home/boxweave/boxweave/current", online)
        self.assertNotIn('b"cd /home/boxweave/boxweave/current', online)
        self.assertIn('atomic(public_path,public_payload,0o644)', online)

    def test_noargs_is_actionable_without_root_or_private_home(self):
        with patch.object(host_cli, "public_state", return_value=fixture()), \
             patch("sys.stdout", new_callable=io.StringIO) as out:
            self.assertEqual(host_cli.main([]), 0)
        text = out.getvalue()
        self.assertIn("Configure your client", text)
        self.assertRegex(text, r"(?m)^  boxweave client$")
        self.assertRegex(text, r"(?m)^  boxweave client --help  # More options$")
        self.assertLess(text.index("  boxweave client\n"), text.index("Optional operations:"))
        self.assertIn("127.0.0.1:18640", text)
        self.assertIn("ssh -N -L", text)
        self.assertIn("not the boxweave service account", text)
        self.assertIn("BOXWEAVE_BATCH", text)
        self.assertIn("--no-enroll", text)
        self.assertIn("--already-enrolled", text)
        self.assertNotRegex(text, r"(?m)^\\s*[1-9]\\.\\s")

    def test_box_root_shows_help_without_inspecting(self):
        with patch.object(host_cli, "inspect") as inspected, \
             patch("sys.stdout", new_callable=io.StringIO) as out:
            self.assertEqual(host_cli.main(["box"]), 0)
        self.assertIn("diagnose", out.getvalue())
        inspected.assert_not_called()

    def test_only_exact_readonly_incus_names(self):
        with patch.object(host_cli, "observe") as execute:
            for name in ("", "../etc", "bad;cmd", "with/slash"):
                result, code = host_cli.inspect(name, "default")
                self.assertEqual(code, 2)
                self.assertEqual(result["code"], "INVALID_EXACT_TARGET")
            execute.assert_not_called()
        record = {"name": "box1", "status": "Running", "type": "container",
                  "expanded_config": {"security.nesting": "false", "private.token": "never-export"}}
        with patch.object(host_cli, "observe", return_value=json.dumps(record)) as query:
            result, code = host_cli.inspect("box1", "default")
        self.assertEqual(code, 0)
        self.assertEqual(result["name"], "box1")
        self.assertNotIn("never-export", json.dumps(result))
        self.assertEqual(query.call_args.args[0],
                         ["incus", "--force-local", "query", "/1.0/instances/box1?project=default"])

    def test_unprivileged_doctor_mcp_never_runs_app_code(self):
        with patch.object(host_cli.os, "geteuid", return_value=1000), \
             patch.object(host_cli, "observe") as execute:
            result = host_cli.mcp_probe(fixture())
        self.assertEqual(result["reason"], "APP_USER_PROBE_REQUIRES_ADMIN")
        execute.assert_not_called()

    def test_public_state_refuses_symlink_and_writeable_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            descriptor = Path(tmp) / "public.json"
            descriptor.write_text(json.dumps(fixture()))
            descriptor.chmod(0o666)
            with patch.object(host_cli, "PUBLIC_STATE", descriptor):
                with self.assertRaises(ValueError):
                    host_cli.public_state()
            descriptor.chmod(0o600)
            link = Path(tmp) / "link.json"
            link.symlink_to(descriptor)
            with patch.object(host_cli, "PUBLIC_STATE", link):
                with self.assertRaises(ValueError):
                    host_cli.public_state()

    def test_status_observation_is_explicitly_not_authenticated(self):
        with patch.object(host_cli, "web_observation", return_value={"reachable": True, "authenticated": False}):
            result = host_cli.status(fixture())
        self.assertEqual(result["mcp"], "not_checked")
        self.assertEqual(result["app_health"], "not_checked")
        self.assertFalse(result["web"]["authenticated"])


if __name__ == "__main__":
    unittest.main()
