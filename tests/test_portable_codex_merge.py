"""Portable Python Codex TOML merger: fixtures only; no local client changes."""
import json
import tempfile
import tomllib
import unittest
from pathlib import Path
from unittest.mock import patch

from bwmini import client_profiles
from test_client_setup import host_key


def fixture(replace=False):
    source=client_profiles.render({
        "app":"codex","os":"windows","transport":"wsl",
        "host":"example.test","port":22,"server_key":host_key(),
        "admin_user":"hostadmin","replace":replace})
    namespace={"__name__":"merge_fixture"}
    exec(compile(source,"setup.py","exec"),namespace)
    return namespace


class PythonCodexMergeTests(unittest.TestCase):
    def test_preserves_other_servers_nested_tables_comments_and_flags(self):
        original='''# Codex desktop config
model="keep"
[mcp_servers.other]
command = "other"
[mcp_servers.boxweave]
command = "old"
args = ["before"]
enabled = true
# keep custom comment

[mcp_servers.boxweave.env]
PRIVATE_OPTION = "keep"
[mcp_servers.boxweave.http_headers]
X_Client = "keep"
[mcp_servers.another]
command = "unrelated"
'''
        with tempfile.TemporaryDirectory() as d:
            home=Path(d)/".codex"
            home.mkdir()
            path=home/"config.toml"
            path.write_text(original)
            env=fixture(replace=False)
            with patch.dict(env["os"].environ,{"CODEX_HOME":str(home)}):
                with self.assertRaisesRegex(ValueError,"differs"):
                    env["update_codex"]("wsl.exe",["--distribution","Ubuntu"])
                self.assertEqual(path.read_text(),original)
            approved=fixture(replace=True)
            with patch.dict(approved["os"].environ,{"CODEX_HOME":str(home)}):
                approved["update_codex"]("wsl.exe",["--distribution","Ubuntu"])
                after=path.read_text()
                obj=tomllib.loads(after)
                self.assertEqual(obj["model"],"keep")
                self.assertEqual(obj["mcp_servers"]["other"]["command"],"other")
                self.assertEqual(obj["mcp_servers"]["another"]["command"],"unrelated")
                self.assertEqual(obj["mcp_servers"]["boxweave"]["command"],"wsl.exe")
                self.assertEqual(obj["mcp_servers"]["boxweave"]["args"],
                                 ["--distribution","Ubuntu"])
                self.assertTrue(obj["mcp_servers"]["boxweave"]["enabled"])
                self.assertEqual(obj["mcp_servers"]["boxweave"]["env"],
                                 {"PRIVATE_OPTION":"keep"})
                self.assertEqual(obj["mcp_servers"]["boxweave"]["http_headers"],
                                 {"X_Client":"keep"})
                self.assertIn("# keep custom comment",after)
                backup=list(home.glob("config.toml.boxweave-*.bak"))
                self.assertEqual(len(backup),1)
                self.assertEqual(backup[0].read_text(),original)
                approved["update_codex"]("wsl.exe",["--distribution","Ubuntu"])
                self.assertEqual(path.read_text(),after)
                self.assertEqual(len(list(home.glob("*.bak"))),1)

    def test_ambiguous_existing_nested_only_and_malformed_are_refused(self):
        for original in (
            "[mcp_servers.boxweave.env]\nPRIVATE_OPTION=\"keep\"\n",
            "[mcp_servers.\"boxweave\"]\ncommand=\"old\"\n",
            "mcp_servers = {boxweave = {command = \"old\"}}\n",
            "[mcp_servers.boxweave]\nargs=[\n\"old\"\n]\ncommand=\"old\"\n",
        ):
            with self.subTest(original=original),tempfile.TemporaryDirectory() as d:
                home=Path(d)/".codex";home.mkdir()
                path=home/"config.toml";path.write_text(original)
                env=fixture(replace=True)
                with patch.dict(env["os"].environ,{"CODEX_HOME":str(home)}):
                    with self.assertRaises((ValueError,tomllib.TOMLDecodeError)):
                        env["update_codex"]("wsl.exe",["--distribution","Ubuntu"])
                self.assertEqual(path.read_text(),original)


if __name__=="__main__":
    unittest.main()
