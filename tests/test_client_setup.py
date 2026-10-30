"""Client setup generator safety: fixture only; no SSH, Codex or privileged writes."""
import base64
import io
import json
import os
from pathlib import Path
import stat
import struct
import tempfile
import unittest
from unittest.mock import patch

from bwmini import client_setup, manage


def host_key():
    data = struct.pack(">I", 11) + b"ssh-ed25519" + struct.pack(">I", 32) + b"x" * 32
    return "ssh-ed25519 " + base64.b64encode(data).decode("ascii")


def ready_state():
    return {"phase": "ready", "host": "192.168.2.233", "port": 22,
            "server_key": host_key(), "managed_public_key": host_key()}


class ClientGeneratorTests(unittest.TestCase):
    def test_public_only_source_bound_script(self):
        payload = client_setup.render(ready_state(), "luka")
        for expected in ("192.168.2.233", host_key(), "'luka'", "online.py enroll",
                         "Codex", "/usr/bin/ssh", "list_containers",
                         "StrictHostKeyChecking=yes", "NoEnroll", "AlreadyEnrolled",
                         "Windows ChatGPT/Codex MCP config:", "PubkeyAuthentication=no",
                         "CODEX_HOME", "config.toml"):
            self.assertIn(expected, payload)
        self.assertNotIn("Get-Command codex", payload)
        self.assertNotIn("Get-Command wsl.exe", payload)
        self.assertIn("$wsl = Join-Path $env:SystemRoot 'System32\\wsl.exe'", payload)
        self.assertIn("Test-Path -LiteralPath $wsl -PathType Leaf", payload)
        self.assertNotIn("codex mcp add", payload)
        self.assertNotIn("jwt.secret", payload)
        self.assertNotIn("StrictHostKeyChecking=no", payload)
        self.assertNotIn("ssh-keyscan", payload)
        self.assertIn("enrollment not confirmed", payload.lower())
        self.assertIn("MCP read verification failed", payload)
        self.assertIn("$sshTarget = 'boxweave@' + $ServerHost", payload)
        self.assertIn("$ManagedWorkloadPublicKey = '", payload)
        self.assertIn("function Merge-BoxWeaveGuidance", payload)
        self.assertIn("MANAGED_PUBLIC_KEY_REQUIRED", payload)
        self.assertIn("if ($WslGuidance)", payload)
        self.assertNotIn("__MANAGED_PUBLIC_KEY__", payload)
        self.assertIn('$sshTarget,$RemoteMcp)', payload)
        self.assertIn("function Merge-BoxWeaveToml(", payload)
        self.assertIn("Nested BoxWeave MCP table without explicit parent", payload)
        self.assertNotIn("$Original -match '(?m)^[ \\t]*\\[mcp_servers\\.boxweave\\.' -or", payload)

    def test_rejects_invalid_public_metadata(self):
        for change in ({"host": "127.0.0.1;rm -rf /"}, {"server_key": "ssh-ed25519 'malicious'"},
                       {"port": -1}, {"port": "22"}):
            with self.subTest(change=change), self.assertRaises(ValueError):
                client_setup.render(ready_state() | change, "luka")
        for managed in (None, "", "SHA256:fingerprint", "ssh-ed25519 AAAA",
                        "ssh-ed25519 AAAA\\nWindows script injection"):
            with self.subTest(managed=managed), self.assertRaises(ValueError):
                client_setup.render(ready_state() | {"managed_public_key": managed}, "luka")
        for admin in ("root;whoami", "bob's", "", "../../tmp"):
            with self.subTest(admin=admin), self.assertRaises(ValueError):
                client_setup.render(ready_state(), admin)

    def test_output_is_new_private_file_with_no_implicit_overwrite(self):
        with tempfile.TemporaryDirectory() as tmp:
            dest = Path(tmp)/"new.ps1"
            with patch.dict(os.environ, {"SUDO_USER": "root"}):
                self.assertEqual(client_setup.publish(dest, "public"), str(dest))
            self.assertEqual(dest.read_text(), "public")
            self.assertEqual(stat.S_IMODE(dest.stat().st_mode), 0o600)
            with self.assertRaisesRegex(ValueError, "NEW_REGULAR_FILE"):
                client_setup.publish(dest, "overwritten")
            self.assertEqual(dest.read_text(), "public")
            path = Path(tmp)/"link.ps1"
            path.symlink_to(dest)
            with self.assertRaises(ValueError):
                client_setup.publish(path, "tamper")
            self.assertEqual(dest.read_text(), "public")

    def test_noninteractive_requires_explicit_profile_and_generates(self):
        with tempfile.TemporaryDirectory() as tmp:
            out = Path(tmp)/"setup.ps1"
            with (patch.object(client_setup.os, "geteuid", return_value=0),
                  patch.object(client_setup.online, "current_state", return_value=ready_state()),
                  patch("sys.stdin.isatty", return_value=False),
                  patch.dict(os.environ, {"SUDO_USER": "root"}),
                  patch("sys.stdout", new_callable=io.StringIO) as stdout):
                code = client_setup.main(["--agent","codex","--transport","wsl",
                                          "--admin-user","luka","--output",str(out)])
                self.assertEqual(code, 0)
                self.assertEqual(json.loads(stdout.getvalue())["state"], "generated")
                self.assertIn("$ServerHost", out.read_text())
                missing = client_setup.main(["--admin-user","luka","--output",str(Path(tmp)/"missing.ps1")])
                self.assertEqual(missing, 2)
                self.assertFalse((Path(tmp)/"missing.ps1").exists())

    def test_interactive_and_noninteractive_share_generator(self):
        with tempfile.TemporaryDirectory() as tmp:
            out = Path(tmp)/"wizard.ps1"
            with (patch.object(client_setup.os,"geteuid",return_value=0),
                  patch.object(client_setup.online,"current_state",return_value=ready_state()),
                  patch("sys.stdin.isatty",return_value=True),
                  patch("builtins.input",side_effect=["1","1"]),
                  patch.dict(os.environ,{"SUDO_USER":"root"}),
                  patch("sys.stdout",new_callable=io.StringIO)):
                code = manage.main(["client","--admin-user","luka","--output",str(out)])
                self.assertEqual(code,0)
                self.assertEqual(out.read_text(),client_setup.render(ready_state(),"luka"))

    def test_generator_refuses_unready_and_nonroot_before_output(self):
        with tempfile.TemporaryDirectory() as tmp:
            target=str(Path(tmp)/"never.ps1")
            with (patch.object(client_setup.os,"geteuid",return_value=1234),
                  patch.object(client_setup.online,"current_state") as state,
                  patch("sys.stderr",new_callable=io.StringIO)):
                code=client_setup.main(["--agent","codex","--transport","wsl",
                                        "--admin-user","luka","--output",target])
            self.assertEqual(code,2)
            state.assert_not_called()
            self.assertFalse(Path(target).exists())


if __name__ == "__main__":
    unittest.main()
