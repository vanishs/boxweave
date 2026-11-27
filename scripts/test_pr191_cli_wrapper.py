"""Current Scoped install wrapper surface; no actual host writes."""
import contextlib
import io
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
