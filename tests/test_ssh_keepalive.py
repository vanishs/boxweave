"""Regressions: keepalive must reach persistent MCP registrations, not only probes."""
import ast
import unittest
from pathlib import Path

from bwmini import client_profiles, client_setup
from test_client_setup import host_key

ROOT=Path(__file__).resolve().parents[1]
ALIVE=("ServerAliveInterval=15","ServerAliveCountMax=4")


class PersistentMcpKeepaliveTests(unittest.TestCase):
    def check_ssh(self,argv):
        for setting in ALIVE:
            self.assertEqual(argv.count(setting),1,setting)
            self.assertEqual(argv[argv.index(setting)-1],"-o")
        self.assertLess(argv.index(ALIVE[0]),argv.index(ALIVE[1]))
        self.assertIn("StrictHostKeyChecking=yes",argv)

    def test_all_portable_profiles_register_keepalive(self):
        count=0
        for system in client_profiles.SYSTEMS:
            transports=("native","wsl") if system=="windows" else ("native",)
            for transport in transports:
                for app in client_profiles.APPS:
                    with self.subTest(system=system,transport=transport,app=app):
                        payload=client_profiles.render({
                            "app":app,"os":system,"transport":transport,
                            "host":"example.test","port":2222,
                            "server_key":host_key(),"admin_user":"hostadmin",
                            "replace":False})
                        scope={"__name__":"keepalive_fixture"}
                        exec(compile(payload,"boxweave-setup.py","exec"),scope)
                        command,args=scope["transport_command"](
                            {"key":"/example/key","known_hosts":"/example/known"},"Ubuntu")
                        self.check_ssh(args)
                        self.assertEqual(command,"wsl.exe" if system=="windows" and transport=="wsl" else "ssh")
                        self.assertIn("2222",args)
                        count+=1
        self.assertEqual(count,36)

    def test_windows_codex_wsl_probe_and_registration_both_protected(self):
        source=client_setup.POWERSHELL
        probe=source.split("$probeScript = @'",1)[1].split("'@",1)[0]
        registration=source.split("$sshArgs = @(",1)[1].split("$homeDir =",1)[0]
        for setting in ALIVE:
            self.assertEqual(probe.count(setting),1)
            self.assertEqual(registration.count(setting),1)
        self.assertIn("StrictHostKeyChecking=yes",probe)
        self.assertIn("StrictHostKeyChecking=yes",registration)

    def test_native_windows_registration_and_diagnostic_scripts(self):
        for filename in ("clients/windows/register-native.ps1",
                         "clients/windows/diagnose-native.ps1",
                         "clients/windows/diagnose-connection.ps1"):
            with self.subTest(filename=filename):
                content=(ROOT/filename).read_text()
                for setting in ALIVE:
                    self.assertEqual(content.count("'-o','"+setting+"'"),1)
                self.assertIn("StrictHostKeyChecking=yes",content)

    def test_legacy_descriptor_helper_keeps_both_ssh_options(self):
        source=(ROOT/"clients/profile_client.py").read_text()
        method=next(node for node in ast.parse(source).body
                    if isinstance(node,ast.FunctionDef) and node.name=="ssh_argv")
        text=ast.get_source_segment(source,method)
        for setting in ALIVE:
            self.assertIn("'-o','"+setting+"'",text)
        self.assertIn("StrictHostKeyChecking=yes",text)


if __name__=="__main__":
    unittest.main()
