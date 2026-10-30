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
        payload = client_setup.render(ready_state(), "example-admin")
        for expected in ("192.168.2.233", host_key(), "'example-admin'", "online.py enroll",
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
                client_setup.render(ready_state() | change, "example-admin")
        for managed in (None, "", "SHA256:fingerprint", "ssh-ed25519 AAAA",
                        "ssh-ed25519 AAAA\\nWindows script injection"):
            with self.subTest(managed=managed), self.assertRaises(ValueError):
                client_setup.render(ready_state() | {"managed_public_key": managed}, "example-admin")
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
                  patch.object(client_setup, "load_public_state", return_value=ready_state()),
                  patch("sys.stdin.isatty", return_value=False),
                  patch.dict(os.environ, {"SUDO_USER": "root"}),
                  patch("sys.stdout", new_callable=io.StringIO) as stdout):
                code = client_setup.main(["--agent","codex","--transport","wsl",
                                          "--admin-user","example-admin","--output",str(out)])
                self.assertEqual(code, 0)
                self.assertEqual(json.loads(stdout.getvalue())["state"], "generated")
                self.assertIn("$ServerHost", out.read_text())
                missing = client_setup.main(["--admin-user","example-admin","--output",str(Path(tmp)/"missing.ps1")])
                self.assertEqual(missing, 2)
                self.assertFalse((Path(tmp)/"missing.ps1").exists())

    def test_interactive_and_noninteractive_share_generator(self):
        with tempfile.TemporaryDirectory() as tmp:
            out = Path(tmp)/"wizard.ps1"
            with (patch.object(client_setup.os,"geteuid",return_value=0),
                  patch.object(client_setup,"load_public_state",return_value=ready_state()),
                  patch("sys.stdin.isatty",return_value=True),
                  patch("builtins.input",side_effect=["1","1","1"]),
                  patch.dict(os.environ,{"SUDO_USER":"root"}),
                  patch("sys.stdout",new_callable=io.StringIO)):
                code = manage.main(["client","--admin-user","example-admin","--output",str(out)])
                self.assertEqual(code,0)
                self.assertEqual(out.read_text(),client_setup.render(ready_state(),"example-admin"))

    def test_last_wizard_choice_is_mcp_registration_not_script_overwrite(self):
        base = ready_state()
        normal = client_setup.render(base, "example-admin")
        replace = client_setup.render(base, "example-admin", replace=True)
        marker = "if (-not $PSBoundParameters.ContainsKey('Replace')) { $Replace = $true }"
        self.assertNotIn(marker, normal)
        self.assertIn(marker, replace)
        self.assertIn("[switch]$Replace,", normal)
        self.assertNotIn("__REPLACE_POLICY__", replace)
        with self.assertRaisesRegex(ValueError, "CLIENT_REPLACE_POLICY_INVALID"):
            client_setup.render(base, "example-admin", replace="yes")

    def test_existing_script_interactively_preserved_with_numbered_output(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "boxweave-setup.ps1"
            path.write_text("original", encoding="utf-8")
            with (patch.object(client_setup, "load_public_state", return_value=ready_state()),
                  patch("sys.stdin.isatty", return_value=True),
                  patch("builtins.input", side_effect=["1", "1", "1", "2"]),
                  patch("sys.stdout", new_callable=io.StringIO) as out):
                code = client_setup.main(["--admin-user", "example-admin",
                                          "--output", str(path)])
            self.assertEqual(code, 0)
            self.assertEqual(path.read_text(), "original")
            result = json.loads(out.getvalue())
            self.assertEqual(result["file"], str(Path(tmp) / "boxweave-setup-2.ps1"))
            self.assertTrue(result["existing_script_preserved"])
            self.assertEqual(result["mcp_registration_policy"], "replace_after_verified_read")
            self.assertIn("$Replace = $true", (Path(tmp) / "boxweave-setup-2.ps1").read_text())
            self.assertEqual(stat.S_IMODE((Path(tmp) / "boxweave-setup-2.ps1").stat().st_mode), 0o600)

    def test_existing_script_interactively_can_cancel_without_mutation(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "boxweave-setup.ps1"
            path.write_text("original")
            with (patch.object(client_setup, "load_public_state", return_value=ready_state()),
                  patch("sys.stdin.isatty", return_value=True),
                  patch("builtins.input", side_effect=["1", "1", "2"]),
                  patch("sys.stderr", new_callable=io.StringIO) as err):
                code = client_setup.main(["--admin-user", "example-admin", "--output", str(path)])
            self.assertEqual(code, 130)
            self.assertEqual(path.read_text(), "original")
            self.assertFalse((Path(tmp) / "boxweave-setup-2.ps1").exists())
            self.assertIn('"state": "cancelled"', err.getvalue())

    def test_existing_script_noninteractive_requires_new_filename(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "boxweave-setup.ps1"
            path.write_text("original")
            with (patch.object(client_setup, "load_public_state", return_value=ready_state()),
                  patch("sys.stdin.isatty", return_value=False),
                  patch("sys.stderr", new_callable=io.StringIO) as err):
                code = client_setup.main(["--agent", "codex", "--transport", "wsl",
                                          "--admin-user", "example-admin", "--output", str(path)])
            self.assertEqual(code, 2)
            self.assertIn("CLIENT_OUTPUT_EXISTS_USE_NEW_OUTPUT_OR_STDOUT", err.getvalue())
            self.assertEqual(path.read_text(), "original")

    def test_noninteractive_mcp_replace_is_explicit_and_cannot_overwrite_script(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "new.ps1"
            with (patch.object(client_setup, "load_public_state", return_value=ready_state()),
                  patch("sys.stdin.isatty", return_value=False),
                  patch("sys.stdout", new_callable=io.StringIO) as out):
                code = client_setup.main(["--agent", "codex", "--transport", "wsl",
                                          "--admin-user", "example-admin", "--replace",
                                          "--output", str(path)])
            self.assertEqual(code, 0)
            self.assertEqual(json.loads(out.getvalue())["mcp_registration_policy"], "replace_after_verified_read")
            self.assertIn("$Replace = $true", path.read_text())
            with patch("sys.stderr", new_callable=io.StringIO):
                self.assertEqual(client_setup.main(["--agent", "codex", "--transport", "wsl",
                                                     "--admin-user", "example-admin", "--replace",
                                                     "--output", str(path)]), 2)
            self.assertIn("$Replace = $true", path.read_text())

    def test_generator_refuses_missing_descriptor_before_output(self):
        with tempfile.TemporaryDirectory() as tmp:
            target = str(Path(tmp) / "never.ps1")
            with patch.object(client_setup, "load_public_state", side_effect=ValueError("PUBLIC_DESCRIPTOR_NOT_AVAILABLE")), \
                 patch("sys.stderr", new_callable=io.StringIO):
                code = client_setup.main(["--agent", "codex", "--transport", "wsl",
                                          "--admin-user", "example-admin", "--output", target])
            self.assertEqual(code, 2)
            self.assertFalse(Path(target).exists())

    def test_nonroot_generator_uses_public_descriptor_without_sudo(self):
        with tempfile.TemporaryDirectory() as tmp:
            target = str(Path(tmp) / "normal.ps1")
            with patch.object(client_setup.os, "geteuid", return_value=1001), \
                 patch.object(client_setup, "load_public_state", return_value=ready_state()), \
                 patch("sys.stdout", new_callable=io.StringIO):
                code = client_setup.main(["--agent", "codex", "--transport", "wsl",
                                          "--admin-user", "example-admin", "--output", target])
            self.assertEqual(code, 0)
            self.assertIn("ManagedWorkloadPublicKey", Path(target).read_text())

    def test_rejects_untrusted_descriptor_not_even_symlink(self):
        with tempfile.TemporaryDirectory() as tmp:
            target = Path(tmp) / "public.json"
            target.write_text(json.dumps(ready_state() | {"schema_version": 1}))
            target.chmod(0o666)
            with self.assertRaisesRegex(ValueError, "PUBLIC_DESCRIPTOR_UNTRUSTED"):
                client_setup.load_public_state(target)
            target.chmod(0o600)
            link = Path(tmp) / "alias.json"
            link.symlink_to(target)
            with self.assertRaisesRegex(ValueError, "PUBLIC_DESCRIPTOR_UNTRUSTED"):
                client_setup.load_public_state(link)


if __name__ == "__main__":
    unittest.main()
