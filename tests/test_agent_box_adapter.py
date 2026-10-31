"""M3: target identity, explicit launcher, path scoping, no live fallback."""
import os
from pathlib import Path
import tempfile
import unittest
from bwmini.agent_box import BoxAgent
from bwmini.domain import Fault, Target

TARGET=Target("containarium","dev","default","a-box","a-box-container",
              "12345","testuser","a-box-container")

class Provider:
    mode="mcp"
    def __init__(self,cfg):self.config=cfg

class Service:
    def __init__(self,cfg):
        self.provider=Provider(cfg)
        self.current=TARGET
    def observe_target(self,logical):
        if logical!=TARGET.logical:raise Fault("TARGET_NOT_ALLOWED",403)
        return self.current

class AgentBoxTests(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory()
        self.launcher=Path(self.temp.name)/"agentbox-launcher"
        self.launcher.write_text("#!/bin/sh\nexit 1\n")
        self.launcher.chmod(0o700)
        self.entry={"target_digest":TARGET.digest(),
                    "workspace":"/home/testuser/work",
                    "launcher":[str(self.launcher)],
                    "allowed_tools":["read_file","write_file","shell_exec"]}
        self.service=Service({"agent_box":{"a-box":self.entry}})
    def tearDown(self):self.temp.cleanup()

    def test_exact_identity_and_workspace(self):
        a=BoxAgent(self.service,"a-box")
        self.assertEqual(a.root,"/home/testuser/work")
        self.assertEqual(a.validate("read_file",{"path":"/home/testuser/work/ok.txt"}),
                         {"path":"/home/testuser/work/ok.txt"})
        with self.assertRaises(Fault):a.validate("read_file",{"path":"/etc/passwd"})
        with self.assertRaises(Fault):a.validate("read_file",{"path":"/home/testuser/work/../.ssh/id"})
        with self.assertRaises(Fault):a.validate("shell_exec",{"command":"echo hello",
                                                                 "cwd":"/tmp"})
    def test_mutation_not_a_read(self):
        a=BoxAgent(self.service,"a-box")
        with self.assertRaises(Fault):a.read("write_file",{"path":"/home/testuser/work/a",
                                                           "content":"x"})
    def test_denies_unconfigured_tool_and_stale_target(self):
        a=BoxAgent(self.service,"a-box")
        with self.assertRaises(Fault):a.validate("delete_file",{"path":"/home/testuser/work/x"})
        self.entry["target_digest"]="wrong"
        with self.assertRaises(Fault):BoxAgent(self.service,"a-box")
    def test_denies_unsafe_launcher(self):
        self.launcher.chmod(0o777)
        with self.assertRaises(Fault):BoxAgent(self.service,"a-box")
        self.launcher.chmod(0o700)
        self.entry["launcher"]=["/does-not-exist"]
        with self.assertRaises(Fault):BoxAgent(self.service,"a-box")

if __name__=="__main__":
    unittest.main()
