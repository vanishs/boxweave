"""Current Scoped install wrapper surface; no actual host writes."""
import contextlib
import io
import json
import subprocess
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

SOURCE = Path(__file__).resolve().parents[1] / "install.sh"
NS = {}
code = SOURCE.read_text().split("<<'PY_BOXWEAVE_INSTALL'\n", 1)[1].rsplit("\ntry:\n    main()", 1)[0]
exec(compile(code, str(SOURCE), "exec"), NS)


class CurrentInstallCLIContract(unittest.TestCase):
    def test_help_advertises_only_current_inputs(self):
        output = io.StringIO()
        with patch.object(sys, "argv", ["install.sh", "--help"]), contextlib.redirect_stdout(output):
            with self.assertRaises(SystemExit) as exit:
                NS["main"]()
        self.assertEqual(exit.exception.code, 0)
        for name in ("--upgrade", "--guided-fresh-plan", "--pool", "--network", "--acl"):
            self.assertNotIn(name, output.getvalue())
        for name in ("--plan", "--apply", "--interactive", "--scoped-fresh-review-sha256"):
            self.assertIn(name, output.getvalue())

    def test_obsolete_selectors_refuse_before_candidate_access(self):
        for args, message in [
            (["--upgrade"], "legacy installation and upgrade unsupported"),
            (["--guided-fresh-plan"], "legacy installation and upgrade unsupported"),
            (["--pool", "irrelevant"], "native scope is derived"),
            (["--network", "irrelevant"], "native scope is derived"),
            (["--acl", "irrelevant"], "native scope is derived"),
        ]:
            with self.subTest(args=args), patch.object(sys, "argv", ["install.sh", *args]):
                with patch.dict(NS, {"refuse_scoped_existing": lambda: None}):
                    with patch.object(NS["os"].path, "lexists", return_value=False):
                        with patch.dict(NS, {"get_bytes": lambda *a: self.fail("candidate accessed")}):
                            with self.assertRaisesRegex(NS["InstallError"], message):
                                NS["main"]()

class DoctorPostflightTests(unittest.TestCase):
    def incomplete(self, mode="UNVERIFIED", reason="LOCAL_MCP_UNAVAILABLE", foreign=0):
        d = {"schema_version": 7, "ready": False, "diagnostic": {
            "installation": {"mode": mode, "candidate": "SCOPED_V2", "manifest": "VERIFIED",
                             "installed_identity": "VERIFIED", "service_unit": "ACTIVE"},
            "database": {"state": "OBSERVED"}, "ledger": {"foreign_rows": foreign},
            "native": {"state": "OBSERVED"},
            "mcp": {"state": "UNVERIFIED", "reason": reason},
            "private": "SENTINEL_NOT_LOGGED"}}
        return subprocess.CalledProcessError(1, ["doctor"], output=json.dumps(d))

    def test_transient_reobserves_only_same_doctor(self):
        from unittest.mock import Mock, call
        cmd = ["exact", "doctor"]
        observation = Mock(side_effect=[self.incomplete(), {"ready": True}])
        err = io.StringIO()
        with patch.dict(NS, {"command_json": observation}), patch.object(NS["time"], "sleep") as sleep:
            with contextlib.redirect_stderr(err):
                self.assertEqual(NS["doctor_postflight"](cmd), {"ready": True})
        self.assertEqual(observation.call_args_list, [call(cmd), call(cmd)])
        sleep.assert_called_once_with(1)
        self.assertIn("LOCAL_MCP_UNAVAILABLE", err.getvalue())
        self.assertNotIn("SENTINEL_NOT_LOGGED", err.getvalue())

    def test_conflict_foreign_and_exhaustion_never_succeed(self):
        from unittest.mock import Mock
        for failure, count in [(self.incomplete(mode="CONFLICT"), 1),
                               (self.incomplete(foreign=1), 1),
                               (self.incomplete(reason="LOCAL_MCP_UNEXPECTED_RESPONSE"), 1),
                               (self.incomplete(), 3)]:
            observation = Mock(side_effect=failure)
            with patch.dict(NS, {"command_json": observation}), patch.object(NS["time"], "sleep"):
                with contextlib.redirect_stderr(io.StringIO()):
                    with self.assertRaisesRegex(NS["InstallError"], "INSTALL_PHASE_UNVERIFIED: CONTROLLER_DOCTOR"):
                        NS["doctor_postflight"](["exact", "doctor"])
            self.assertEqual(observation.call_count, count)
