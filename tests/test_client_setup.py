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
                self.assertIn("'replace': False",output.read_text())
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
                  patch("sys.stdout.isatty",return_value=True),
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
                      patch("sys.stdout.isatty",return_value=True),
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
            self.assertIn("'replace': True",script)
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
        self.assertIn("'replace': False",out.getvalue())
        self.assertIn("# Registration policy: preserve",out.getvalue())
        self.assertIn("# Generation changed no Linux",err.getvalue())

    def test_interactive_no_preserves_entry_and_cancel_writes_nothing(self):
        with (patch.object(client_setup,"load_public_state",return_value=ready_state()),
              patch("sys.stdin.isatty",return_value=True),
              patch("builtins.input",side_effect=["3","3","2","1"]),
              patch("sys.stdout",new_callable=io.StringIO) as out,
              patch("sys.stdout.isatty",return_value=True),
              patch("sys.stderr",new_callable=io.StringIO) as err):
            rc=client_setup.main(["--admin-user","example-admin"])
        self.assertEqual(rc,0)
        self.assertIn("'replace': False",out.getvalue())
        self.assertIn("# Replace an existing conflicting BoxWeave MCP",err.getvalue())
        with (patch.object(client_setup,"load_public_state",return_value=ready_state()),
              patch("sys.stdin.isatty",return_value=True),
              patch("builtins.input",side_effect=["3","3","2",EOFError]),
              patch("sys.stdout",new_callable=io.StringIO) as out,
              patch("sys.stdout.isatty",return_value=True),
              patch("sys.stderr",new_callable=io.StringIO) as err):
            rc=client_setup.main(["--admin-user","example-admin"])
        self.assertEqual(rc,130)
        self.assertEqual(out.getvalue(),"")
        self.assertIn('"state": "cancelled"',err.getvalue())

    def test_captured_output_exec_channel_never_prompts(self):
        # Agent/exec channels expose a TTY stdin but capture stdout; the
        # wizard must treat them as noninteractive instead of waiting forever.
        with (patch.object(client_setup,"load_public_state",return_value=ready_state()),
              patch("sys.stdin.isatty",return_value=True),
              patch("builtins.input",side_effect=AssertionError("prompted")),
              patch("sys.stdout",new_callable=io.StringIO) as out,
              patch("sys.stderr",new_callable=io.StringIO)):
            rc=client_setup.main(["--agent","codex","--transport","wsl",
                                  "--admin-user","example-admin"])
        self.assertEqual(rc,0)
        self.assertTrue(out.getvalue().startswith("#!/usr/bin/env python3\n"))
        self.assertIn("'replace': False",out.getvalue())

    def test_boxweave_batch_forces_noninteractive_despite_full_terminal(self):
        with (patch.object(client_setup,"load_public_state",return_value=ready_state()),
              patch.dict(os.environ,{"BOXWEAVE_BATCH":"1"}),
              patch("sys.stdin.isatty",return_value=True),
              patch("builtins.input",side_effect=AssertionError("prompted")),
              patch("sys.stdout",new_callable=io.StringIO) as out,
              patch("sys.stdout.isatty",return_value=True),
              patch("sys.stderr",new_callable=io.StringIO)):
            rc=client_setup.main(["--agent","codex","--transport","wsl",
                                  "--admin-user","example-admin"])
        self.assertEqual(rc,0)
        self.assertIn("'replace': False",out.getvalue())

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
                self.assertIn("'replace': True",path.read_text())
                self.assertEqual(client_setup.main(["--agent","codex","--transport","wsl",
                    "--admin-user","example-admin","--replace",
                    "--output",str(path)]),2)
                self.assertIn("'replace': True",path.read_text())

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

    def zcode_script(self):
        from bwmini import client_profiles
        script=client_profiles.render({
            "app":"zcode","os":"linux","transport":"native",
            "host":"192.168.2.233","port":22,"server_key":host_key(),
            "admin_user":"example-admin","replace":False})
        ns={"__name__":"fixture"}
        exec(compile(script,"diagnose.py","exec"),ns)
        return ns

    def test_diagnose_reports_missing_identity_without_changes(self):
        import sys
        ns=self.zcode_script()
        with tempfile.TemporaryDirectory() as tmp, \
             patch.object(Path,"home",return_value=Path(tmp)), \
             patch.object(sys,"argv",["boxweave-setup.py","--diagnose"]), \
             patch("sys.stdout",new_callable=io.StringIO) as out, \
             patch("sys.stderr",new_callable=io.StringIO) as err:
            try:
                ns["main"]()
                raised,refused=0,None
            except SystemExit as exit_error:
                raised,refused=exit_error.code,None
            except ValueError as error:
                raised,refused=None,str(error)
            leftovers=list(Path(tmp).iterdir())
        py_ok=sys.version_info[:2]>=(3,11)
        if refused is not None:
            # Interpreters below the client floor are refused before diagnosis.
            self.assertFalse(py_ok)
            self.assertIn("3.11",refused)
            self.assertEqual(leftovers,[])
            return
        self.assertEqual(raised,1)  # registration absent is a definite negative
        summary=json.loads(out.getvalue())
        self.assertEqual(summary["diagnostic_schema"],1)
        self.assertEqual(summary["overall"],"failed")
        self.assertEqual(summary["state"],"failed")
        self.assertEqual(summary["connection_overall"],"incomplete")
        by_id={c["id"]:c for c in summary["checks"]}
        self.assertEqual(by_id["python"]["state"],"passed" if py_ok else "failed")
        self.assertEqual(by_id["identity"]["state"],"not_verified")
        self.assertEqual(by_id["identity"]["code"],"IDENTITY_MISSING")
        self.assertEqual(by_id["ssh_transport"]["state"],"not_verified")
        self.assertEqual(by_id["ssh_transport"]["detail"],"blocked_by=identity")
        self.assertEqual(by_id["mcp_read"]["state"],"not_verified")
        self.assertEqual(by_id["registration"]["state"],"failed")
        self.assertEqual(by_id["registration"]["code"],"REGISTRATION_ABSENT")
        self.assertEqual(by_id["agent_ui"]["state"],"not_verified")
        self.assertFalse(by_id["agent_ui"]["required"])
        self.assertEqual(summary["coverage"]["model_inference"],"not_verified")
        self.assertNotIn("ssh-ed25519 ",out.getvalue())
        self.assertIn("# identity: not_verified IDENTITY_MISSING",err.getvalue())
        self.assertIn("# Diagnosis: failed (connection: incomplete)",err.getvalue())
        self.assertEqual(leftovers,[])

    def test_diagnose_live_stages_registration_and_pin_mismatch(self):
        import subprocess as real_subprocess
        import sys
        ns=self.zcode_script()
        py_ok=sys.version_info[:2]>=(3,11)
        responses="".join(json.dumps(m)+"\n" for m in (
            {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}}}},
            {"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"list_containers"}]}},
            {"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"Found 0 container(s):"}]}}))
        ok=real_subprocess.CompletedProcess([],0,stdout=responses,stderr="")
        leaked=real_subprocess.CompletedProcess([],255,stdout="",
                                                stderr="Warning: identity file SECRET-TOKEN not accessible")
        with tempfile.TemporaryDirectory() as tmp:
            home=Path(tmp)
            root=home/".config"/"boxweave"
            root.mkdir(parents=True,mode=0o700)
            (root/"id_ed25519").write_text("PRIVATE")
            (root/"known_hosts").write_text("192.168.2.233 "+host_key()+"\n")
            identity={"key":str(root/"id_ed25519"),"known_hosts":str(root/"known_hosts")}
            command,args=ns["transport_command"](identity,"")
            config=home/".zcode"/"cli"
            config.mkdir(parents=True)
            entry=ns["app_entry"](command,args)
            (config/"config.json").write_text(json.dumps({"mcp":{"servers":{"boxweave":entry}}}))
            with patch.object(Path,"home",return_value=home), \
                 patch("subprocess.run",return_value=ok):
                code,summary=ns["run_diagnose"]("")
            self.assertEqual(code,0 if py_ok else 2)
            by_id={c["id"]:c for c in summary["checks"]}
            self.assertEqual(by_id["identity"]["state"],"passed")
            self.assertEqual(by_id["ssh_transport"]["state"],"passed")
            self.assertEqual(by_id["mcp_initialize"]["state"],"passed")
            self.assertEqual(by_id["mcp_tools"]["state"],"passed")
            self.assertEqual(by_id["mcp_read"]["state"],"passed")
            self.assertEqual(by_id["registration"]["state"],"passed")
            self.assertEqual(summary["overall"],"passed")
            self.assertEqual(summary["connection_overall"],"passed")
            (config/"config.json").write_text(json.dumps({"mcp":{"servers":{"boxweave":"stale"}}}))
            with patch.object(Path,"home",return_value=home), \
                 patch("subprocess.run",return_value=ok):
                code,summary=ns["run_diagnose"]("")
            by_id={c["id"]:c for c in summary["checks"]}
            self.assertEqual(by_id["registration"]["state"],"failed")
            self.assertEqual(by_id["registration"]["code"],"REGISTRATION_DIFFERS")
            self.assertEqual(summary["overall"],"failed")
            with patch.object(Path,"home",return_value=home), \
                 patch("subprocess.run",return_value=leaked), \
                 patch("sys.stderr",new_callable=io.StringIO) as err:
                code,summary=ns["run_diagnose"]("")
            by_id={c["id"]:c for c in summary["checks"]}
            self.assertEqual(summary["overall"],"failed")
            self.assertEqual(by_id["ssh_transport"]["code"],"SSH_TRANSPORT_FAILED")
            self.assertEqual(by_id["ssh_transport"]["detail"],"ssh exit class 255")
            self.assertEqual(by_id["mcp_initialize"]["state"],"not_verified")
            self.assertEqual(by_id["mcp_initialize"]["detail"],"blocked_by=ssh_transport")
            rendered=json.dumps(summary)+err.getvalue()
            self.assertNotIn("SECRET-TOKEN",rendered)
            (root/"known_hosts").write_text("192.168.2.233 WRONG\n")
            with patch.object(Path,"home",return_value=home):
                code,summary=ns["run_diagnose"]("")
            by_id={c["id"]:c for c in summary["checks"]}
            self.assertEqual(code,1)
            self.assertEqual(by_id["identity"],{"id":"identity","state":"failed","code":"PIN_MISMATCH",
                                                "scope":"client_identity","required":True,
                                                "detail":"pinned host key differs; review out of band"})
            self.assertEqual(by_id["ssh_transport"]["state"],"not_verified")

    def test_diagnose_malformed_stream_fails_handshake_never_leaks(self):
        import subprocess as real_subprocess
        ns=self.zcode_script()
        valid_init=json.dumps({"jsonrpc":"2.0","id":1,
                               "result":{"protocolVersion":"2024-11-05","capabilities":{}}})+"\n"
        cases=(("garbage","not json at all\n","failed","not_verified","not_verified"),
               ("init_only",valid_init,"passed","failed","not_verified"),
               ("secret_noise","SECRET-STREAM-DATA\n"*10,"failed","not_verified","not_verified"),
               ("empty","","failed","not_verified","not_verified"))
        for name,broken,init_state,tools_state,read_state in cases:
            with self.subTest(case=name), tempfile.TemporaryDirectory() as tmp:
                home=Path(tmp)
                root=home/".config"/"boxweave"
                root.mkdir(parents=True,mode=0o700)
                (root/"id_ed25519").write_text("PRIVATE")
                (root/"known_hosts").write_text("192.168.2.233 "+host_key()+"\n")
                completed=real_subprocess.CompletedProcess([],0,stdout=broken,stderr="SECRET-ERR")
                with patch.object(Path,"home",return_value=home), \
                     patch("subprocess.run",return_value=completed):
                    code,summary=ns["run_diagnose"]("")
                by_id={c["id"]:c for c in summary["checks"]}
                self.assertEqual(by_id["ssh_transport"]["state"],"passed")
                self.assertEqual(by_id["mcp_initialize"]["state"],init_state)
                self.assertEqual(by_id["mcp_tools"]["state"],tools_state)
                self.assertEqual(by_id["mcp_read"]["state"],read_state)
                if init_state=="failed":
                    self.assertEqual(by_id["mcp_initialize"]["code"],"MCP_HANDSHAKE_INVALID")
                    self.assertEqual(by_id["mcp_read"]["detail"],"blocked_by=mcp_initialize")
                else:
                    self.assertEqual(by_id["mcp_tools"]["code"],"MCP_READ_TOOL_MISSING")
                    self.assertEqual(by_id["mcp_read"]["detail"],"blocked_by=mcp_tools")
                self.assertEqual(summary["connection_overall"],"failed")
                self.assertNotIn("SECRET",json.dumps(summary))


if __name__ == "__main__":
    unittest.main()
