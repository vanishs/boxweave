"""Unified client generator fixture tests: every profile emits Python source."""
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

from bwmini import client_setup, client_profiles, manage


def host_key():
    data = struct.pack(">I", 11) + b"ssh-ed25519" + struct.pack(">I", 32) + b"x" * 32
    return "ssh-ed25519 " + base64.b64encode(data).decode("ascii")


def ready_state():
    return {"phase": "ready", "host": "192.168.2.233", "port": 22,
            "server_key": host_key(), "managed_public_key": host_key()}


class ClientGeneratorTests(unittest.TestCase):
    def test_all_apps_share_one_executable_python_script(self):
        for system in client_profiles.SYSTEMS:
            transports=("native","wsl") if system=="windows" else ("native",)
            for transport in transports:
                for app in client_profiles.APPS:
                    with self.subTest(app=app,system=system,transport=transport):
                        script=client_setup.render(ready_state(),"example-admin",
                                                   app=app,system=system,transport=transport)
                        self.assertTrue(script.startswith(
                            "#!/usr/bin/env python3\n# ==="),script[:90])
                        self.assertIn("# BoxWeave MCP setup: ",script.splitlines()[2])
                        self.assertIn("# Registration policy:",script)
                        self.assertIn("# EDIT ME:",script)
                        self.assertIn("ServerAliveInterval=15",script)
                        self.assertIn("StrictHostKeyChecking=yes",script)
                        self.assertIn("list_containers",script)
                        self.assertNotIn("jwt.secret",script)
                        self.assertNotIn("POWERSHELL",script)
                        compile(script,"boxweave-setup.py","exec")

    def test_public_metadata_and_replace_type_validation(self):
        for change in ({"host":"127.0.0.1;rm -rf /"},
                       {"server_key":"ssh-ed25519 'malicious'"},
                       {"port":-1},{"port":"22"}):
            with self.subTest(change=change), self.assertRaises(ValueError):
                client_setup.render(ready_state()|change,"example-admin")
        for value in (None,"","SHA256:fingerprint"):
            with self.subTest(value=value):
                script=client_setup.render(
                    ready_state()|{"managed_public_key":value},"example-admin")
                self.assertNotIn("ManagedWorkloadPublicKey",script)
        for admin in ("root;whoami","bob's","","../../tmp"):
            with self.subTest(admin=admin),self.assertRaises(ValueError):
                client_setup.render(ready_state(),admin)
        with self.assertRaisesRegex(ValueError,"CLIENT_REPLACE_POLICY_INVALID"):
            client_setup.render(ready_state(),"example-admin",replace="yes")

    def test_output_is_new_private_file_never_overwritten(self):
        with tempfile.TemporaryDirectory() as tmp:
            dest=Path(tmp)/"setup.py"
            with patch.dict(os.environ,{"SUDO_USER":"root"}):
                self.assertEqual(client_setup.publish(dest,"public"),str(dest))
            self.assertEqual(dest.read_text(),"public")
            self.assertEqual(stat.S_IMODE(dest.stat().st_mode),0o600)
            with self.assertRaisesRegex(ValueError,"NEW_REGULAR_FILE"):
                client_setup.publish(dest,"tamper")
            alias=Path(tmp)/"alias.py"
            alias.symlink_to(dest)
            with self.assertRaises(ValueError):
                client_setup.publish(alias,"tamper")
            self.assertEqual(dest.read_text(),"public")

    def test_noninteractive_codex_wsl_uses_python_and_requires_explicit_selection(self):
        with tempfile.TemporaryDirectory() as tmp:
            output=Path(tmp)/"new.py"
            with (patch.object(client_setup,"load_public_state",return_value=ready_state()),
                  patch("sys.stdin.isatty",return_value=False),
                  patch("sys.stdout",new_callable=io.StringIO) as out,
                  patch("sys.stderr",new_callable=io.StringIO) as err):
                rc=client_setup.main([
                    "--agent","codex","--os","windows","--transport","wsl",
                    "--admin-user","example-admin","--output",str(output)])
                self.assertEqual(rc,0)
                self.assertEqual(out.getvalue(),"")
                self.assertTrue(output.read_text().startswith("#!/usr/bin/env python3\n"))
                self.assertIn("REPLACE = False",output.read_text())
                self.assertNotIn("PowerShell",output.read_text())
                self.assertEqual(client_setup.main(["--admin-user","example-admin",
                    "--output",str(Path(tmp)/"missing.py")]),2)
                self.assertFalse((Path(tmp)/"missing.py").exists())

    def test_interactive_matches_noninteractive_for_codex_wsl(self):
        with tempfile.TemporaryDirectory() as tmp:
            output=Path(tmp)/"wizard.py"
            with (patch.object(client_setup,"load_public_state",return_value=ready_state()),
                  patch("sys.stdin.isatty",return_value=True),
                  patch("builtins.input",side_effect=["3","3","2","1"]),
                  patch("sys.stdout",new_callable=io.StringIO),
                  patch("sys.stderr",new_callable=io.StringIO) as err):
                rc=manage.main(["client","--admin-user","example-admin",
                                "--output",str(output)])
            self.assertEqual(rc,0)
            self.assertEqual(output.read_text(),client_setup.render(
                ready_state(),"example-admin",False,
                app="codex",system="windows",transport="wsl"))
            self.assertIn("# Choose MCP application",err.getvalue())
            self.assertIn("# Select number: ",err.getvalue())
            for line in err.getvalue().splitlines():
                self.assertTrue(not line.strip() or line.startswith("# "),line)

    def test_stdout_is_only_python_even_during_interactive_yes(self):
        with tempfile.TemporaryDirectory() as tmp:
            previous=Path(tmp)/"boxweave-setup.py"
            previous.write_text("old script")
            old=os.getcwd()
            try:
                os.chdir(tmp)
                with (patch.object(client_setup,"load_public_state",return_value=ready_state()),
                      patch("sys.stdin.isatty",return_value=True),
                      patch("builtins.input",side_effect=["3","3","2","2"]),
                      patch("sys.stdout",new_callable=io.StringIO) as out,
                      patch("sys.stderr",new_callable=io.StringIO) as err):
                    rc=client_setup.main(["--admin-user","example-admin"])
            finally:
                os.chdir(old)
            self.assertEqual(rc,0)
            self.assertEqual(previous.read_text(),"old script")
            self.assertEqual([p.name for p in Path(tmp).iterdir()],
                             ["boxweave-setup.py"])
            script=out.getvalue()
            self.assertTrue(script.startswith("#!/usr/bin/env python3\n# ==="))
            self.assertIn("REPLACE = True",script)
            self.assertIn("# Registration policy: replace",script)
            self.assertNotIn("Select number:",script)
            self.assertNotIn("copy the complete",script)
            self.assertNotIn("PowerShell",script)
            compile(script,"boxweave-setup.py","exec")
            self.assertIn("# Select number:",err.getvalue())
            self.assertIn("# BoxWeave codex / windows / wsl setup:",err.getvalue())
            for line in err.getvalue().splitlines():
                self.assertTrue(not line.strip() or line.startswith("# "),line)

    def test_noninteractive_default_preserves_registration(self):
        with (patch.object(client_setup,"load_public_state",return_value=ready_state()),
              patch("sys.stdin.isatty",return_value=False),
              patch("sys.stdout",new_callable=io.StringIO) as out,
              patch("sys.stderr",new_callable=io.StringIO) as err):
            rc=client_setup.main(["--agent","codex","--transport","wsl",
                                  "--admin-user","example-admin"])
        self.assertEqual(rc,0)
        self.assertTrue(out.getvalue().startswith("#!/usr/bin/env python3\n"))
        self.assertIn("REPLACE = False",out.getvalue())
        self.assertIn("# Registration policy: preserve",out.getvalue())
        self.assertIn("# Generation changed no Linux",err.getvalue())

    def test_interactive_no_preserves_entry_and_cancel_writes_nothing(self):
        with (patch.object(client_setup,"load_public_state",return_value=ready_state()),
              patch("sys.stdin.isatty",return_value=True),
              patch("builtins.input",side_effect=["3","3","2","1"]),
              patch("sys.stdout",new_callable=io.StringIO) as out,
              patch("sys.stderr",new_callable=io.StringIO) as err):
            rc=client_setup.main(["--admin-user","example-admin"])
        self.assertEqual(rc,0)
        self.assertIn("REPLACE = False",out.getvalue())
        self.assertIn("# Replace an existing conflicting BoxWeave MCP",err.getvalue())
        with (patch.object(client_setup,"load_public_state",return_value=ready_state()),
              patch("sys.stdin.isatty",return_value=True),
              patch("builtins.input",side_effect=["3","3","2",EOFError]),
              patch("sys.stdout",new_callable=io.StringIO) as out,
              patch("sys.stderr",new_callable=io.StringIO) as err):
            rc=client_setup.main(["--admin-user","example-admin"])
        self.assertEqual(rc,130)
        self.assertEqual(out.getvalue(),"")
        self.assertIn('"state": "cancelled"',err.getvalue())

    def test_explicit_replace_backups_only_client_mcp_entry(self):
        with tempfile.TemporaryDirectory() as tmp:
            path=Path(tmp)/"approved.py"
            with (patch.object(client_setup,"load_public_state",return_value=ready_state()),
                  patch("sys.stdin.isatty",return_value=False),
                  patch("sys.stdout",new_callable=io.StringIO)):
                rc=client_setup.main(["--agent","codex","--transport","wsl",
                                      "--admin-user","example-admin","--replace",
                                      "--output",str(path)])
                self.assertEqual(rc,0)
                self.assertIn("REPLACE = True",path.read_text())
                self.assertEqual(client_setup.main(["--agent","codex","--transport","wsl",
                    "--admin-user","example-admin","--replace",
                    "--output",str(path)]),2)
                self.assertIn("REPLACE = True",path.read_text())

    def test_nonroot_uses_public_metadata_and_refuses_untrusted_descriptor(self):
        with tempfile.TemporaryDirectory() as tmp:
            target=Path(tmp)/"normal.py"
            with (patch.object(client_setup.os,"geteuid",return_value=1001),
                  patch.object(client_setup,"load_public_state",return_value=ready_state()),
                  patch("sys.stdout",new_callable=io.StringIO)):
                self.assertEqual(client_setup.main(["--agent","codex","--transport","wsl",
                    "--admin-user","example-admin","--output",str(target)]),0)
            self.assertNotIn("ManagedWorkloadPublicKey",target.read_text())
            descriptor=Path(tmp)/"public.json"
            descriptor.write_text(json.dumps(ready_state()|{"schema_version":1}))
            descriptor.chmod(0o666)
            with self.assertRaisesRegex(ValueError,"PUBLIC_DESCRIPTOR_UNTRUSTED"):
                client_setup.load_public_state(descriptor)
            descriptor.chmod(0o600)
            link=Path(tmp)/"alias.json"
            link.symlink_to(descriptor)
            with self.assertRaisesRegex(ValueError,"PUBLIC_DESCRIPTOR_UNTRUSTED"):
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
