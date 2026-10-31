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
        self.assertIn("# EDIT ME: use the client-reachable host/IP and SSH port.", payload)
        self.assertNotIn("$ManagedWorkloadPublicKey", payload)
        self.assertNotIn("Merge-BoxWeaveGuidance", payload)
        self.assertNotIn("AGENTS.md", payload)
        self.assertNotIn("WslGuidance", payload)
        self.assertIn('$sshTarget,$RemoteMcp)', payload)
        self.assertIn("function Merge-BoxWeaveToml(", payload)
        self.assertIn("Nested BoxWeave MCP table without explicit parent", payload)
        self.assertNotIn("$Original -match '(?m)^[ \\t]*\\[mcp_servers\\.boxweave\\.' -or", payload)

    def test_rejects_invalid_public_metadata(self):
        for change in ({"host": "127.0.0.1;rm -rf /"}, {"server_key": "ssh-ed25519 'malicious'"},
                       {"port": -1}, {"port": "22"}):
            with self.subTest(change=change), self.assertRaises(ValueError):
                client_setup.render(ready_state() | change, "example-admin")
        # The Linux MCP gateway supplies the server's workload key.
        for value in (None, "", "SHA256:fingerprint"):
            with self.subTest(value=value):
                script = client_setup.render(ready_state() | {"managed_public_key":value}, "example-admin")
                self.assertNotIn("ManagedWorkloadPublicKey", script)
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
                self.assertEqual(stdout.getvalue(), "")
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
                  patch("builtins.input",side_effect=["3","3","2","1"]),
                  patch.dict(os.environ,{"SUDO_USER":"root"}),
                  patch("sys.stdout",new_callable=io.StringIO)):
                code = manage.main(["client","--admin-user","example-admin","--output",str(out)])
                self.assertEqual(code,0)
                self.assertEqual(out.read_text(),client_setup.script_header(False) + client_setup.render(ready_state(),"example-admin"))

    def test_last_wizard_choice_is_mcp_registration_not_script_overwrite(self):
        base = ready_state()
        normal = client_setup.render(base, "example-admin")
        replace = client_setup.render(base, "example-admin", replace=True)
        marker = "if (-not $PSBoundParameters.ContainsKey('Replace')) { $Replace = $true }"
        self.assertNotIn(marker, normal)
        self.assertIn(marker, replace)
        self.assertIn("[switch]$Replace", normal)
        self.assertNotIn("__REPLACE_POLICY__", replace)
        with self.assertRaisesRegex(ValueError, "CLIENT_REPLACE_POLICY_INVALID"):
            client_setup.render(base, "example-admin", replace="yes")

    def test_default_stdout_is_copyable_powershell_and_never_writes_local_script(self):
        with tempfile.TemporaryDirectory() as tmp:
            existing = Path(tmp) / "boxweave-setup.ps1"
            existing.write_text("old script")
            original = os.getcwd()
            try:
                os.chdir(tmp)
                with (patch.object(client_setup, "load_public_state", return_value=ready_state()),
                      patch("sys.stdin.isatty", return_value=True),
                      patch("builtins.input", side_effect=["3", "3", "2", "2"]),
                      patch("sys.stdout", new_callable=io.StringIO) as out,
                      patch("sys.stderr", new_callable=io.StringIO) as err):
                    result = client_setup.main(["--admin-user", "example-admin"])
            finally:
                os.chdir(original)
            self.assertEqual(result, 0)
            self.assertEqual(existing.read_text(), "old script")
            self.assertEqual([p.name for p in Path(tmp).iterdir()], ["boxweave-setup.ps1"])
            ps = out.getvalue()
            self.assertTrue(ps.startswith("# ==="))
            self.assertIn("#requires -version 5.1", ps)
            self.assertIn("$ServerHost", ps)
            self.assertIn("Windows MCP policy: Update", ps)
            self.assertNotIn("Select number:", ps)
            self.assertNotIn('{"state": "generated"', ps)
            self.assertNotIn("__REPLACE_POLICY__", ps)
            self.assertIn("Select number:", err.getvalue())
            self.assertIn("PowerShell (.ps1)", err.getvalue())
            self.assertNotIn("Client script already exists", err.getvalue())

    def test_noninteractive_default_stdout_preserves_conflicts(self):
        with (patch.object(client_setup, "load_public_state", return_value=ready_state()),
              patch("sys.stdin.isatty", return_value=False),
              patch("sys.stdout", new_callable=io.StringIO) as out,
              patch("sys.stderr", new_callable=io.StringIO) as err):
            result = client_setup.main(["--agent", "codex", "--transport", "wsl",
                                        "--admin-user", "example-admin"])
        self.assertEqual(result, 0)
        self.assertIn("Windows MCP policy: Preserve", out.getvalue())
        self.assertNotIn("$Replace = $true", out.getvalue())
        self.assertIn("Generation changed no Linux", err.getvalue())

    def test_interactive_choice_one_preserves_existing_mcp_registration(self):
        with (patch.object(client_setup, "load_public_state", return_value=ready_state()),
              patch("sys.stdin.isatty", return_value=True),
              patch("builtins.input", side_effect=["3", "3", "2", "1"]),
              patch("sys.stdout", new_callable=io.StringIO) as out,
              patch("sys.stderr", new_callable=io.StringIO) as err):
            result = client_setup.main(["--admin-user", "example-admin"])
        self.assertEqual(result, 0)
        self.assertIn("Windows MCP policy: Preserve", out.getvalue())
        self.assertIn("Replace an existing conflicting BoxWeave MCP entry", err.getvalue())
        self.assertNotIn("Client script already exists", err.getvalue())

    def test_interrupted_wizard_writes_no_partial_script(self):
        with (patch.object(client_setup, "load_public_state", return_value=ready_state()),
              patch("sys.stdin.isatty", return_value=True),
              patch("builtins.input", side_effect=["3", "3", "2", EOFError]),
              patch("sys.stdout", new_callable=io.StringIO) as out,
              patch("sys.stderr", new_callable=io.StringIO) as err):
            result = client_setup.main(["--admin-user", "example-admin"])
        self.assertEqual(result, 130)
        self.assertEqual(out.getvalue(), "")
        self.assertIn('"state": "cancelled"', err.getvalue())

    def test_explicit_output_refuses_existing_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            dest = Path(tmp) / "boxweave-setup.ps1"
            dest.write_text("old")
            with (patch.object(client_setup, "load_public_state", return_value=ready_state()),
                  patch("sys.stdin.isatty", return_value=False),
                  patch("sys.stderr", new_callable=io.StringIO) as err):
                result = client_setup.main(["--agent", "codex", "--transport", "wsl",
                                            "--admin-user", "example-admin",
                                            "--output", str(dest)])
            self.assertEqual(result, 2)
            self.assertIn("CLIENT_OUTPUT_MUST_BE_NEW_REGULAR_FILE", err.getvalue())
            self.assertEqual(dest.read_text(), "old")

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
            self.assertEqual(out.getvalue(), "")
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
            self.assertNotIn("ManagedWorkloadPublicKey", Path(target).read_text())

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


    def test_every_cross_platform_profile_compiles(self):
        from bwmini import client_profiles
        self.assertEqual(client_profiles.APPS, tuple(sorted(client_profiles.APPS)))
        self.assertEqual(client_profiles.APPS,
            ("antigravity","claude","codex","copilot","cursor",
             "opencode","qoder","trae","zcode"))
        self.assertEqual(client_profiles.SYSTEMS, ("linux","macos","windows"))
        public=ready_state()
        for system in client_profiles.SYSTEMS:
            for transport in (("native","wsl") if system=="windows" else ("native",)):
                for app in client_profiles.APPS:
                    with self.subTest(system=system,transport=transport,app=app):
                        script=client_profiles.render({
                            "app":app,"os":system,"transport":transport,
                            "host":public["host"],"port":public["port"],
                            "server_key":public["server_key"],
                            "admin_user":"example-admin","replace":False})
                        compile(script, "boxweave-setup.py", "exec")
                        self.assertIn("StrictHostKeyChecking=yes",script)
                        self.assertIn("list_containers",script)
                        self.assertIn("online.py enroll",script)
                        self.assertIn('"boxweave"',script)
                        self.assertNotIn("jwt.secret",script)
        with self.assertRaisesRegex(ValueError,"UNSUPPORTED"):
            client_profiles.render({
                "app":"cursor","os":"macos","transport":"wsl",
                "host":public["host"],"port":22,
                "server_key":public["server_key"],
                "admin_user":"example-admin","replace":False})

    def test_new_profile_generates_portable_python(self):
        with (patch.object(client_setup,"load_public_state",return_value=ready_state()),
              patch("sys.stdin.isatty",return_value=False),
              patch("sys.stdout",new_callable=io.StringIO) as out,
              patch("sys.stderr",new_callable=io.StringIO)):
            rc=client_setup.main(["--app","cursor","--os","macos","--transport","native",
                                  "--admin-user","example-admin","--no-replace"])
        self.assertEqual(rc,0)
        self.assertTrue(out.getvalue().startswith("#!/usr/bin/env python3"))
        self.assertIn("cursor / macos / native",out.getvalue())
        compile(out.getvalue(),"boxweave-setup.py","exec")

    def test_portable_json_registration_preserves_other_servers(self):
        from bwmini import client_profiles
        script=client_profiles.render({
            "app":"cursor","os":"linux","transport":"native",
            "host":"server.internal","port":22,"server_key":host_key(),
            "admin_user":"example-admin","replace":False})
        ns={"__name__":"client_fixture"}
        exec(compile(script,"fixture.py","exec"),ns)
        with tempfile.TemporaryDirectory() as tmp:
            home=Path(tmp)
            folder=home/".cursor"
            folder.mkdir()
            path=folder/"mcp.json"
            path.write_text(json.dumps({"mcpServers":{"other":{"command":"safe"}}}))
            with patch.object(Path,"home",return_value=home):
                ns["update_json"]("ssh",["-T","boxweave@host"])
                data=json.loads(path.read_text())
                self.assertEqual(data["mcpServers"]["other"],{"command":"safe"})
                self.assertEqual(data["mcpServers"]["boxweave"]["command"],"ssh")
                ns["update_json"]("ssh",["-T","boxweave@host"])
                self.assertEqual(len(list(folder.glob("*.bak"))),1)
                with self.assertRaisesRegex(ValueError,"differs"):
                    ns["update_json"]("ssh",["-T","changed@host"])
                self.assertEqual(json.loads(path.read_text()),data)

    def test_qoder_and_trae_use_official_manual_import(self):
        from bwmini import client_profiles
        for app in ("qoder","trae"):
            with self.subTest(app=app):
                script=client_profiles.render({
                    "app":app,"os":"linux","transport":"native",
                    "host":"example.test","port":22,"server_key":host_key(),
                    "admin_user":"example-admin","replace":False})
                ns={"__name__":"fixture"}
                exec(compile(script,"manual.py","exec"),ns)
                with tempfile.TemporaryDirectory() as tmp, \
                     patch.object(Path,"home",return_value=Path(tmp)), \
                     patch("sys.stdout",new_callable=io.StringIO) as output:
                    ns["update_json"]("ssh",["-T","boxweave@host"])
                    self.assertIn('"mcpServers"',output.getvalue())
                    self.assertEqual(list(Path(tmp).iterdir()),[])


if __name__ == "__main__":
    unittest.main()
