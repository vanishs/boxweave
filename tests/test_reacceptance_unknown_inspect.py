"""Safety regression suite for read-only follow-up of post-dispatch UNKNOWN."""
from pathlib import Path
import json
import os
import socket
import tempfile
import time
import unittest
from unittest.mock import patch

from tools.reacceptance import lifecycle, unknown_inspect
from tools.reacceptance.engine import AcceptanceFault


class OriginalUnknownInspectionTests(unittest.TestCase):
    def fixture(self, root):
        root = Path(root)
        original = root / "original"
        original.mkdir(mode=0o700)
        child = "bwracc-test-unknown-1"
        outer_uuid = "a"*8+"-aaaa-aaaa-aaaa-"+"a"*12
        past = {"uuid": "b"*8+"-bbbb-bbbb-bbbb-"+"b"*12,
                "created_at": "2026-11-06T04:40:07Z"}
        docs = {
            "lab.json": {"scope": lifecycle.SCOPE, "child_name": child,
                         "outer_uuid": outer_uuid,
                         "outer_created_at": "1793939346"},
            "01-create_container.intent.json": {
                "name": child, "tool": "create_container",
                "state": "MAY_BE_DISPATCHED_ONCE",
                "arguments": {"username": child}},
            "01-create_container.attempt.json": {
                "tool": "create_container",
                "request": {"username": child},
                "success_response": False,
                "process": {"execution": "UNKNOWN", "exit_code": None}},
            "01-create_container.oracle.json": {"incarnation": past},
            "result.json": {"state": "UNKNOWN_OR_FAIL",
                            "child_name": child, "no_automatic_retry": True},
        }
        for name, value in docs.items():
            p = original/name
            p.write_text(json.dumps(value))
            p.chmod(0o600)
        grant = root / "grant.json"
        grant.write_text(json.dumps({
            "scope": lifecycle.SCOPE, "approved": True,
            "hostname": socket.gethostname(),
            "machine_id": Path("/etc/machine-id").read_text().strip(),
            "outer_uuid": outer_uuid,
            "outer_created_at": "1793939346",
            "child_name": child,
            "expires_epoch": time.time() + 3600
        }))
        grant.chmod(0o600)
        native = {
            "name": child+"-container", "status": "Running",
            "created_at": past["created_at"],
            "config": {"volatile.uuid": past["uuid"]}}
        return original, grant, child, native

    def test_original_unknown_with_real_present_child_remains_unknown(self):
        with tempfile.TemporaryDirectory() as d:
            original, grant, child, native = self.fixture(d)
            original_before = (original/"result.json").read_bytes()
            with patch.object(unknown_inspect.lifecycle, "native_list",
                              return_value={child+"-container": native}) as read:
                x = unknown_inspect.inspect(original, grant, lifecycle.CONFIRM,
                                            Path(d)/"report")
            read.assert_called_once()
            self.assertEqual(x["state"], "SAME_ORIGINAL_SIDE_EFFECT_EXISTS_STILL_UNKNOWN")
            self.assertFalse(x["cleanup_authorized"])
            self.assertFalse(x["release_acceptance"])
            self.assertFalse(x["original_side_effect_replayed"])
            self.assertEqual((original/"result.json").read_bytes(), original_before)
            self.assertTrue((Path(d)/"report/read-only-reconcile.json").exists())

    def test_replaced_incarnation_is_hard_blocked_not_new_creation(self):
        with tempfile.TemporaryDirectory() as d:
            original, grant, child, native = self.fixture(d)
            native["config"]["volatile.uuid"] = "c"*8+"-cccc-cccc-cccc-"+"c"*12
            with patch.object(unknown_inspect.lifecycle, "native_list",
                              return_value={child+"-container": native}):
                x = unknown_inspect.inspect(original, grant, lifecycle.CONFIRM,
                                            Path(d)/"changed")
            self.assertEqual(x["state"], "INCARNATION_CHANGED_HARD_BLOCK")
            self.assertFalse(x["cleanup_authorized"])

    def test_later_absent_cannot_claim_original_never_existed(self):
        with tempfile.TemporaryDirectory() as d:
            original, grant, _, _ = self.fixture(d)
            with patch.object(unknown_inspect.lifecycle, "native_list",
                              return_value={}):
                x = unknown_inspect.inspect(original, grant, lifecycle.CONFIRM,
                                            Path(d)/"missing")
            self.assertEqual(x["state"], "NOT_PRESENT_AT_LATER_READ_ORIGINAL_UNCERTAIN")
            self.assertEqual(x["original_outcome"], "UNKNOWN_NO_REPLAY")

    def test_refuses_forged_success_and_foreign_grant(self):
        with tempfile.TemporaryDirectory() as d:
            original, grant, child, native = self.fixture(d)
            item=original/"01-create_container.attempt.json"
            x=json.loads(item.read_text());x["success_response"]=True
            item.write_text(json.dumps(x))
            with patch.object(unknown_inspect.lifecycle, "native_list") as read:
                with self.assertRaisesRegex(AcceptanceFault,
                                      "UNKNOWN_ORIGINAL_CASE_NOT_DISPATCHED_OR_INCONSISTENT"):
                    unknown_inspect.inspect(original,grant,lifecycle.CONFIRM,
                                            Path(d)/"rejected")
                read.assert_not_called()
            x["success_response"]=False;item.write_text(json.dumps(x))
            g=json.loads(grant.read_text());g["child_name"]="bwracc-foreign-unknown"
            grant.write_text(json.dumps(g))
            with patch.object(unknown_inspect.lifecycle, "native_list") as read:
                with self.assertRaisesRegex(AcceptanceFault,
                                      "UNKNOWN_ORIGINAL_CASE_NOT_DISPATCHED_OR_INCONSISTENT"):
                    unknown_inspect.inspect(original,grant,lifecycle.CONFIRM,
                                            Path(d)/"other")
                read.assert_not_called()

    def test_refuses_insecure_original_record_permissions(self):
        with tempfile.TemporaryDirectory() as d:
            original, grant, child, native = self.fixture(d)
            (original/"result.json").chmod(0o666)
            with patch.object(unknown_inspect.lifecycle, "native_list") as read:
                with self.assertRaisesRegex(AcceptanceFault,
                                            "UNKNOWN_ORIGINAL_RECORD_UNTRUSTED"):
                    unknown_inspect.inspect(original,grant,lifecycle.CONFIRM,
                                            Path(d)/"insecure")
                read.assert_not_called()


if __name__ == "__main__":
    unittest.main()

