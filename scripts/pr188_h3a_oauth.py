#!/usr/bin/env python3
"""H3A wrapper tests with optional real isolated CLI/SQLite witness.

Never use the installed ledger or a real tunnel. No credential material printed.
"""
import argparse
import contextlib
import io
import json
from pathlib import Path
import subprocess
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import sqlite3
import os
import pwd

ROOT = Path(__file__).resolve().parents[1]
source = (ROOT / "install.sh").read_text().split("<<'PY_BOXWEAVE_INSTALL'\n", 1)[1]
source = source.rsplit("\ntry:\n    main()", 1)[0]
ns = {"__name__": "h3a_test"}
exec(compile(source, "install.sh", "exec"), ns)
ORIGIN = "https://fixture.example"
IDENTITY = ["fixture"]
ARGS = SimpleNamespace(project="user-1002", service_user="boxweave", public_origin=None)


class Wrapper(unittest.TestCase):
    def configure(self, responses, inp, origin=None, failures=None, fallback=None):
        args = SimpleNamespace(**vars(ARGS))
        args.public_origin = origin
        args.oauth_password_configured = fallback
        output = io.StringIO()
        tty = io.StringIO()
        calls = []
        def run(argv, **kwargs):
            calls.append(argv)
            if failures and argv[1] in failures:
                raise subprocess.CalledProcessError(1, argv)
        with patch.dict(ns, {"command_json": lambda _: next(responses)}), patch.object(ns["subprocess"], "run", side_effect=run), contextlib.redirect_stdout(output):
            code = ns["configure_oauth"](IDENTITY, args, io.StringIO(inp), tty)
        return code, calls, output.getvalue(), tty.getvalue()

    def test_origin_contract(self):
        normalize = ns["normalize_oauth_origin"]
        for raw in [ORIGIN, ORIGIN+"/", "https://a-b.example"]:
            self.assertEqual(normalize(raw), raw.rstrip("/"))
        for bad in [ORIGIN+"//", ORIGIN+"/mcp", ORIGIN+"?", ORIGIN+"#", "http://fixture.example",
                    "https://x@fixture.example", "https://fixture.example:443", "https://127.0.0.1",
                    "https://[::1]", "https://x.localhost", "https://UPPER.example", "https://-x.example",
                    "https://x..example", ORIGIN+" ", " "+ORIGIN, ORIGIN+"%2f", "https://x.example:"]:
            with self.subTest(bad=bad), self.assertRaises(ns["InstallError"]):
                normalize(bad)

    def test_retry_preserves_password(self):
        before = dict(Origin="", Enabled=False, PasswordConfigured=False)
        password = dict(before, PasswordConfigured=True)
        done = dict(password, Origin=ORIGIN, Enabled=True)
        code, calls, out, tty = self.configure(iter([before, password, done]), "y\nhttps://bad.example/mcp\n"+ORIGIN+"/\n")
        self.assertEqual(code, 0)
        self.assertEqual([x[1] for x in calls], ["oauth-setup", "oauth-set-origin"])
        self.assertEqual(calls[-1][-1], ORIGIN)
        self.assertIn("OAUTH_ORIGIN_INVALID", out)
        self.assertNotIn("bad.example", out)
        self.assertIn("OAUTH_CONFIGURED", out)

    def test_defer_and_resume(self):
        before = dict(Origin="", Enabled=False, PasswordConfigured=False)
        password = dict(before, PasswordConfigured=True)
        done = dict(password, Origin=ORIGIN, Enabled=True)
        for state, inp, marker in [(before, "N\n", "OAUTH_CONFIGURATION_DEFERRED"), (password, "\n", "OAUTH_ORIGIN_MISSING")]:
            code, calls, out, _ = self.configure(iter([state]), inp)
            self.assertEqual((code, calls), (0, []))
            self.assertIn(marker, out)
        code, calls, out, tty = self.configure(iter([password, done]), "", origin=ORIGIN+"/")
        self.assertEqual((code, [x[1] for x in calls]), (0, ["oauth-set-origin"]))
        self.assertNotIn("Configure a new", tty)

    def test_apply_unverified_not_retried(self):
        before = dict(Origin="", Enabled=False, PasswordConfigured=True)
        code, calls, out, _ = self.configure(iter([before, before]), "", origin=ORIGIN, failures={"oauth-set-origin"})
        self.assertEqual((code, len(calls)), (2, 1))
        self.assertIn("OAUTH_ORIGIN_APPLY_UNVERIFIED", out)
        self.assertIn("oauth-status", out)

    def test_lost_reply_observed(self):
        before = dict(Origin="", Enabled=False, PasswordConfigured=True)
        done = dict(before, Origin=ORIGIN, Enabled=True)
        code, calls, out, _ = self.configure(iter([before, done]), "", origin=ORIGIN, failures={"oauth-set-origin"})
        self.assertEqual((code, len(calls)), (0, 1))
        self.assertIn("OAUTH_CONFIGURED", out)

    def test_retry_limit_and_unknown(self):
        password = dict(Origin="", Enabled=False, PasswordConfigured=True)
        code, calls, out, _ = self.configure(iter([password]), "bad\nbad\nbad\n")
        self.assertEqual((code, calls), (2, []))
        self.assertIn("OAUTH_ORIGIN_RETRY_LIMIT", out)
        code, calls, out, _ = self.configure(iter([dict(Origin="", Enabled=False)]), "")
        self.assertEqual((code, calls), (2, []))
        self.assertIn("OAUTH_PASSWORD_STATE_UNVERIFIED", out)

    def test_install_phase_failure_is_redacted(self):
        out = io.StringIO()
        with patch.dict(ns, {"command_json": lambda _: (_ for _ in ()).throw(ns["InstallError"]("private input"))}), contextlib.redirect_stdout(out):
            with self.assertRaisesRegex(ns["InstallError"], "^INSTALL_PHASE_UNVERIFIED: CONTROLLER_DOCTOR") as raised:
                ns["install_phase"]("CONTROLLER_DOCTOR", ["fixture"], capture=True)
        self.assertNotIn("private input", str(raised.exception))
        self.assertIn("INSTALL_PHASE=CONTROLLER_DOCTOR", out.getvalue())

    def test_existing_enabled_no_write(self):
        done = dict(Origin=ORIGIN, Enabled=True, PasswordConfigured=True)
        code, calls, out, _ = self.configure(iter([done]), "")
        self.assertEqual((code, calls), (0, []))
        self.assertIn("OAUTH_CONFIGURED", out)

    def test_old_candidate_fallback(self):
        old = dict(Origin="", Enabled=False)
        done = dict(Origin=ORIGIN, Enabled=True)
        code, calls, out, _ = self.configure(iter([old, done]), "", origin=ORIGIN, fallback=True)
        self.assertEqual((code, [x[1] for x in calls]), (0, ["oauth-set-origin"]))


def real_cli(binary, project):
    # Dedicated temp ledger; must not modify existing controller or auth.
    uid = pwd.getpwnam("boxweave")
    with tempfile.TemporaryDirectory(prefix="pr188-h3a-") as d:
        os.chown(d, uid.pw_uid, uid.pw_gid)
        db = str(Path(d) / "ledger.sqlite")
        prefix = ["runuser", "-u", "boxweave", "--", binary]
        def cli(action, *flags, check=True):
            return subprocess.run(prefix + [action, "-project", project, "-db", db, *flags],
                                  text=True, capture_output=True, check=check, timeout=30)
        def status():
            return json.loads(cli("oauth-status").stdout)
        def state():
            with sqlite3.connect("file:"+db+"?mode=ro", uri=True) as conn:
                return conn.execute("SELECT public_origin,password_hash,epoch FROM oauth_config WHERE id=1").fetchone()
        assert status() == dict(Origin="", Enabled=False, PasswordConfigured=False)
        private = str(Path(d)/"approval")
        cli("oauth-setup", "-generate", "-output-file", private)
        assert status() == dict(Origin="", Enabled=False, PasswordConfigured=True)
        before = state()
        # Independent CLI rejects the invalid value without changing state.
        assert cli("oauth-set-origin", "-public-origin", ORIGIN+"/mcp", check=False).returncode != 0
        assert state() == before
        # Drive the real wrapper continuation with existing password.
        args = SimpleNamespace(project=project, service_user="boxweave", public_origin=ORIGIN+"/")
        orig = subprocess.run
        # Call the real CLI while isolating the wrapper's process dispatch.
        def invoke(argv, **kw):
            if argv[0] != "fixture":
                return orig(argv, **kw)
            assert argv[1] == "oauth-set-origin", "continuation must not reset password"
            return orig(prefix + [argv[1], "-project", project, "-db", db, *argv[4:]],
                        text=True, capture_output=True, check=True, timeout=30)
        with patch.dict(ns, {"command_json": lambda _: status()}), patch.object(subprocess, "run", side_effect=invoke):
            assert ns["configure_oauth"](IDENTITY, args, io.StringIO(), io.StringIO()) == 0
        after = state()
        assert after[0] == ORIGIN and after[1] == before[1] and after[2] == before[2]+1
        assert status() == dict(Origin=ORIGIN, Enabled=True, PasswordConfigured=True)
        cli("oauth-disable")
        assert status() == dict(Origin="", Enabled=False, PasswordConfigured=True)
        print("PR188_H3A_REAL_OAUTH_RESUME=PASS")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary")
    parser.add_argument("--project")
    args = parser.parse_args()
    suite = unittest.defaultTestLoader.loadTestsFromTestCase(Wrapper)
    if not unittest.TextTestRunner(verbosity=2).run(suite).wasSuccessful():
        raise SystemExit(1)
    print("PR188_H3A_WRAPPER=PASS")
    if args.binary:
        assert args.project and os.geteuid() == 0
        real_cli(args.binary, args.project)
