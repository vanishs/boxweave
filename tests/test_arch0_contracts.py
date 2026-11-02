"""ARCH-0C pure-contract extraction and backwards-compatible import regression."""
import base64
import json
import pathlib
import struct
import sys
import unittest

from bwmini.domain import Fault, Target
from bwmini.provider import Provider, text, execution
from bwmini.wire_results import text as wire_text, execution as wire_execution
from bwmini.access import validate_config as public_access_validate
from bwmini.access_contract import validate_config as core_access_validate
from bwmini.lifecycle import RESIZE as public_resize, Lifecycle
from bwmini.lifecycle_contract import RESIZE as core_resize, validate_config as lifecycle_validate


class ContractExtractionTests(unittest.TestCase):
    def test_provider_result_exports_preserved(self):
        self.assertIs(text, wire_text)
        self.assertIs(execution, wire_execution)
        sample = {"content":[{"type":"text","text":"ok"}]}
        self.assertEqual(text(sample), "ok")

    def test_structured_and_legacy_execution_identical(self):
        self.assertEqual(execution({"structuredContent":{"exit_code":7,"stdout":"failed"}}),
                         (7,"failed"))
        self.assertEqual(execution({"content":[{"text":"exit_code: 0\n\n--- stdout ---\nhello"}]}),
                         (0,"hello"))
        with self.assertRaises(Fault):
            execution({"structuredContent":{"exit_code":True,"stdout":"bad"}})

    def test_access_validator_public_import_stable(self):
        self.assertIs(public_access_validate,core_access_validate)
        self.assertIsNone(public_access_validate(None))
        with self.assertRaises(Fault) as caught:
            core_access_validate({"tenants":[]})
        self.assertEqual(caught.exception.code,"ACCESS_CONFIG_INVALID")

    def test_lifecycle_options_preserved(self):
        self.assertIs(public_resize,core_resize)
        self.assertEqual(core_resize["cpu"], ("1","2","3","4","5","6","7","8"))

    def test_provider_validates_without_worker_or_remote_access(self):
        p=Provider({"provider":"containarium","backend":"local","project":"default",
                    "launcher":[sys.executable],"targets":{}})
        self.assertEqual(p.mode,"mcp")
        with self.assertRaises(Fault) as caught:
            Provider({"provider":"containarium","backend":"local","project":"default",
                      "launcher":[sys.executable],"targets":{},
                      "lifecycle":{"allowed_names":[]}})
        self.assertEqual(caught.exception.code,"LIFECYCLE_CONFIG_INVALID")

    def test_lifecycle_rejects_protected_names(self):
        pub = base64.b64encode(struct.pack(">I",11)+b"ssh-ed25519" +
                              struct.pack(">I",32)+b"x"*32).decode()
        cfg={"allowed_names":["target"],"cpu":"1","memory":"1GB","disk":"8GB",
             "ssh_public_key":"ssh-ed25519 "+pub,"readiness_timeout":60}
        lifecycle_validate(cfg,set())
        with self.assertRaises(Fault):
            lifecycle_validate(cfg,{"target"})

    def test_core_modules_avoid_provider_dependency_inversion(self):
        for name in ("bwmini/access_contract.py","bwmini/lifecycle_contract.py",
                     "bwmini/wire_results.py"):
            path=pathlib.Path(__file__).resolve().parents[1]/name
            source=path.read_text()
            self.assertNotIn("from .provider import",source)
            self.assertNotIn("from .lifecycle import",source)
        from bwmini import access_provider, agent_box
        self.assertIs(access_provider.text,wire_text)
        self.assertIs(agent_box.text,wire_text)


if __name__ == "__main__":
    unittest.main()
