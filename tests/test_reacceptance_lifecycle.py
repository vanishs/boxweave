"""Safety and success controls for the disposable nested lifecycle driver."""
from pathlib import Path
import json
import os
import socket
import tempfile
import time
import unittest
from unittest.mock import patch

from tools.reacceptance import lifecycle
from tools.reacceptance.engine import AcceptanceFault


class LiveLifecycleTests(unittest.TestCase):
    def create_grant(self, base):
        grant = Path(base) / "grant.json"
        grant.write_text(json.dumps({
            "scope": lifecycle.SCOPE, "approved": True,
            "hostname": socket.gethostname(),
            "machine_id": Path("/etc/machine-id").read_text().strip(),
            "outer_uuid": "a"*8 + "-aaaa-aaaa-aaaa-" + "a"*12,
            "outer_created_at": "1793931912",
            "child_name": "bwracc-scenario-probe-1",
            "expires_epoch": time.time() + 3600,
        }))
        grant.chmod(0o600)
        return grant

    def test_no_unapproved_or_foreign_grant(self):
        with tempfile.TemporaryDirectory() as d:
            grant = self.create_grant(d)
            with self.assertRaisesRegex(AcceptanceFault, "CONFIRMATION"):
                lifecycle.grant_read(grant, "not-approved")
            value = json.loads(grant.read_text())
            value["child_name"] = "production"
            grant.write_text(json.dumps(value))
            with self.assertRaisesRegex(AcceptanceFault, "UNSAFE_CHILD_NAME"):
                lifecycle.grant_read(grant, lifecycle.CONFIRM)

    def test_lifecycle_simulated_native_oracle_limits_pass(self):
        with tempfile.TemporaryDirectory() as d:
            grant = self.create_grant(d)
            name = "bwracc-scenario-probe-1"
            identity = {"volatile.uuid": "a"*8 + "-aaaa-aaaa-aaaa-" + "a"*12}
            def item(status):
                return {name + "-container": {
                    "name": name + "-container", "status": status,
                    "created_at": "2026-11-05T00:00:00Z",
                    "config": identity}}
            observations = [{}, item("Running"), item("Stopped"), item("Running"), {}]
            def one_call(argv, name_, args):
                return {"tool": name_, "request": args,
                        "process": {"execution": "completed", "exit_code": 0},
                        "response": {"result": {"content": []}},
                        "success_response": True}
            with patch.object(lifecycle, "native_list", side_effect=observations), (
                 patch.object(lifecycle, "one_call", side_effect=one_call)):
                answer = lifecycle.run(grant, lifecycle.CONFIRM,
                                       ["python3", "-m", "bwmini.mcp_gate"],
                                       Path(d) / "run")
            self.assertEqual(answer["state"], "PARTIAL")
            self.assertEqual(answer["scenario_id"], "SYS-LIFE-01")
            self.assertIn("resize", answer["not_tested"])
            self.assertTrue((Path(d) / "run" / "04-delete_container.oracle.json").exists())

    def test_confirmed_gateway_preflight_denial_is_not_provider_unknown(self):
        with tempfile.TemporaryDirectory() as d:
            grant = self.create_grant(d)
            def denied(argv,tool,args):
                return {"tool":tool,"request":args,
                        "process":{"execution":"completed","exit_code":0},
                        "response":{"error":{"code":-32602,
                                             "message":"MANAGED_IDENTITY_UNAVAILABLE"}},
                        "success_response":False}
            with patch.object(lifecycle,"native_list",side_effect=[{},{}]), (
                 patch.object(lifecycle,"one_call",side_effect=denied)) as calls:
                with self.assertRaisesRegex(AcceptanceFault,"PREFLIGHT_DENIED"):
                    lifecycle.run(grant,lifecycle.CONFIRM,
                                  ["sudo","-n","-u","boxweave","python3",
                                   "/usr/local/lib/boxweave/default/mcp_gate.py"],
                                  Path(d)/"run")
                self.assertEqual(calls.call_count,1)
                self.assertTrue((Path(d)/"run"/"01-create_container.denied.json").is_file())
                self.assertFalse((Path(d)/"run"/"01-create_container.unknown.json").exists())
                outcome=json.loads((Path(d)/"run"/"result.json").read_text())
                self.assertEqual(outcome["state"],"BLOCKED_PRE_DISPATCH")

    def test_slow_real_create_has_bounded_cold_budget(self):
        from tools.reacceptance import lifecycle
        with patch.object(lifecycle, "mcp_session", return_value={
                "messages": [{"id":"racc-lifecycle",
                              "result":{"content":[]}}],
                "process":{"execution":"completed","exit_code":0}}) as session:
            result = lifecycle.one_call(["sudo", "-n", "-u", "boxweave",
                 "python3", "/usr/local/lib/boxweave/default/mcp_gate.py"],
                 "create_container",
                 {"username":"bwracc-budget-cold", **lifecycle.CHILD_RESOURCES})
        self.assertEqual(session.call_args.kwargs["timeout"], 240)
        self.assertEqual(result["timeout_seconds"], 240)
        self.assertTrue(result["success_response"])
        self.assertEqual(lifecycle.CHILD_RESOURCES["memory"], "1GB")
        with self.assertRaisesRegex(AcceptanceFault, "UNREVIEWED_LIFECYCLE_ACTION"):
            lifecycle.one_call(["python3"], "unreviewed_write", {})

    def test_failed_mcp_write_keeps_original_and_does_not_retry(self):
        with tempfile.TemporaryDirectory() as d:
            grant = self.create_grant(d)
            def one_call(argv, tool, args):
                return {"tool": tool, "request": args,
                        "process": {"execution": "completed", "exit_code": 1},
                        "response": {"error": {"message": "unknown"}},
                        "success_response": False}
            with patch.object(lifecycle, "native_list", side_effect=[
                {}, {"bwracc-scenario-probe-1-container": {
                     "name": "bwracc-scenario-probe-1-container",
                     "status": "Running",
                     "config": {"volatile.uuid": "a"*8+"-aaaa-aaaa-aaaa-"+"a"*12},
                     "created_at": "2026-11-05T00:00:00Z"}}]), (
                 patch.object(lifecycle, "one_call", side_effect=one_call)) as calls:
                with self.assertRaisesRegex(AcceptanceFault, "UNKNOWN_NO_REPLAY"):
                    lifecycle.run(grant, lifecycle.CONFIRM,
                                  ["python3", "-m", "bwmini.mcp_gate"], Path(d)/"run")
                self.assertEqual(calls.call_count, 1)
                self.assertTrue((Path(d)/"run"/"01-create_container.unknown.json").exists())
                verdict=json.loads((Path(d)/"run"/"result.json").read_text())
                self.assertTrue(verdict["no_automatic_retry"])


if __name__ == "__main__":
    unittest.main()

