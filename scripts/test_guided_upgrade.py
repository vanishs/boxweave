"""Offline tests of actual install.sh Python coordinator; no host writes."""
import argparse
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch, Mock

SOURCE = Path(__file__).resolve().parents[1] / "install.sh"
NS = {}
code = SOURCE.read_text().split("<<'PY_BOXWEAVE_INSTALL'\n", 1)[1].rsplit("\ntry:\n    main()", 1)[0]
exec(compile(code, str(SOURCE), "exec"), NS)
ERR = NS["InstallError"]
SHA, OLD, PREV = "a"*64, "b"*64, "c"*64
SOURCE_SHA, OLD_COMMIT = "d"*40, "e"*40

def args(**kw):
    return argparse.Namespace(project="user-1001", service_user="boxweave",
                              incus_socket=NS["RESTRICTED_SOCKET"], local_binary=None, **kw)

def current():
    return dict(schema=2, version="1.0.0-rc.10", build_commit=OLD_COMMIT,
                binary_sha256=OLD, unit_sha256="f"*64, service_active=True,
                restart_required=False, native_project="user-1001",
                native_socket=NS["RESTRICTED_SOCKET"], service_user="boxweave",
                binary_path=str(NS["INSTALLED_BINARY"]), state_dir=str(NS["STATE_DIR"]))

def review(previous=True):
    return dict(current=current(), manifest_sha256="1"*64,
                previous=dict(version="1.0.0-rc.9", binary_sha256=PREV,
                              build_commit="2"*40, receipt_sha256="3"*64) if previous else None)

class GuidedUpgradeTests(unittest.TestCase):
    def test_release_order(self):
        f = NS["release_order"]
        self.assertLess(f("1.0.0-rc.9"), f("1.0.0-rc.10"))
        self.assertLess(f("1.0.0-rc.10"), f("1.0.0"))
        self.assertLess(f("1.0.0"), f("1.0.1-rc.1"))
        with self.assertRaises(ERR):
            f("0.5.1")

    def test_noop_preserves_backup(self):
        with patch.dict(NS, {"require_unchanged_upgrade": Mock(), "command_json": Mock()}) as _:
            with patch("subprocess.run") as run, patch("builtins.open") as terminal:
                NS["apply_guided_upgrade"](args(), Path("/fixture/candidate"), OLD, "1.0.0-rc.10",
                                          OLD_COMMIT, review(), "host")
                run.assert_not_called()
                terminal.assert_not_called()
                NS["command_json"].assert_not_called()

    def test_decline_no_mutation(self):
        with patch.dict(NS, {"require_unchanged_upgrade": Mock()}):
            with patch("builtins.open", side_effect=[io.StringIO("NO\n"), io.StringIO()]), patch("subprocess.run") as run:
                with self.assertRaisesRegex(ERR, "not approved"):
                    NS["apply_guided_upgrade"](args(), Path("/fixture/candidate"), SHA, "1.0.0-rc.11",
                                              SOURCE_SHA, review(), "host")
                run.assert_not_called()

    def test_changed_plan_refuses(self):
        check = Mock(side_effect=[None, ERR("changed")])
        with patch.dict(NS, {"require_unchanged_upgrade": check}):
            with patch("builtins.open", side_effect=[io.StringIO("UPGRADE user-1001\n"), io.StringIO()]), patch("subprocess.run") as run:
                with self.assertRaisesRegex(ERR, "changed"):
                    NS["apply_guided_upgrade"](args(), Path("/fixture/candidate"), SHA, "1.0.0-rc.11",
                                              SOURCE_SHA, review(), "host")
                run.assert_not_called()

    def test_identity_mismatch_before_commit(self):
        with patch.dict(NS, {"require_unchanged_upgrade": Mock(),
                             "command_json": Mock(return_value={"version":"1.0.0-rc.11", "commit": OLD_COMMIT})}):
            with patch("builtins.open", side_effect=[io.StringIO("UPGRADE user-1001\n"), io.StringIO()]), patch("subprocess.run") as run:
                with self.assertRaisesRegex(ERR, "identity"):
                    NS["apply_guided_upgrade"](args(), Path("/fixture/candidate"), SHA, "1.0.0-rc.11",
                                              SOURCE_SHA, review(), "host")
                run.assert_not_called()

    def test_commit_failure_stops_before_upgrade(self):
        with patch.dict(NS, {"require_unchanged_upgrade": Mock(), "service_health": Mock(),
                             "command_json": Mock(return_value={"version":"1.0.0-rc.11", "commit": SOURCE_SHA})}):
            with patch("builtins.open", side_effect=[io.StringIO("UPGRADE user-1001\n"), io.StringIO()]), patch("subprocess.run", side_effect=NS["subprocess"].CalledProcessError(1, "install-commit")) as run:
                with self.assertRaises(NS["subprocess"].CalledProcessError):
                    NS["apply_guided_upgrade"](args(), Path("/fixture/candidate"), SHA, "1.0.0-rc.11",
                                              SOURCE_SHA, review(), "host")
                self.assertEqual(run.call_count, 1)
                self.assertEqual(run.call_args.args[0][1], "install-commit")

    def test_no_backup_and_pending_backup_success(self):
        for pending in (False, True):
            with self.subTest(pending=pending):
                final = dict(current(), version="1.0.0-rc.11", build_commit=SOURCE_SHA, binary_sha256=SHA)
                verify = Mock()
                with patch.dict(NS, {"require_unchanged_upgrade": verify, "service_health": Mock(return_value={"ready":True}),
                                     "command_json": Mock(side_effect=[{"version":"1.0.0-rc.11", "commit":SOURCE_SHA}, final]),
                                     "private_json": Mock(return_value=({"Schema":1,"Previous":current()}, "4"*64))}):
                    with patch("builtins.open", side_effect=[io.StringIO("UPGRADE user-1001\n"), io.StringIO()]), patch("subprocess.run") as run:
                        NS["apply_guided_upgrade"](args(), Path("/fixture/candidate"), SHA, "1.0.0-rc.11",
                                                  SOURCE_SHA, review(pending), "host")
                        self.assertEqual([c.args[0][1] for c in run.call_args_list],
                                         ["install-commit", "upgrade"] if pending else ["upgrade"])
                        if pending:
                            verify.assert_called_with(unittest.mock.ANY, dict(review(True), previous=None), "host")

    def test_private_metadata_rejects_symlink_and_bad_mode(self):
        with tempfile.TemporaryDirectory() as folder:
            p = Path(folder)/"receipt.json"
            p.write_text("{}")
            p.chmod(0o600)
            self.assertEqual(NS["private_json"](p, p.stat().st_uid)[0], {})
            link = p.with_name("link")
            link.symlink_to(p)
            with self.assertRaises(OSError):
                NS["private_json"](link, p.stat().st_uid)
            p.chmod(0o644)
            with self.assertRaises(ERR):
                NS["private_json"](p, p.stat().st_uid)

if __name__ == "__main__":
    unittest.main()
