#!/usr/bin/env python3
"""Offline negative controls for host uninstaller: never touch a real host."""
import contextlib
import json
from pathlib import Path
import sqlite3
import tempfile
import types
import unittest
from unittest import mock

import uninstall_core as u


class FakeHost(unittest.TestCase):
    def setUp(self):
        self.t = tempfile.TemporaryDirectory()
        self.addCleanup(self.t.cleanup)
        root = Path(self.t.name)
        self.paths = {
            "STATE": root / "state",
            "UNIT": root / "units" / "boxweave.service",
            "BINARY": root / "bin" / "boxweave",
            "NATIVE": root / "native-prep.json",
            "SCRIPTLET": root / "scriptlet.json",
            "ROOT_UNIT": root / "units" / "boxweave-native-peer-root.service",
            "AUDIT": root / "uninstall-audit.json",
            "BACKUP": root / "installer-backup",
        }
        for name, p in self.paths.items():
            patcher = mock.patch.object(u, name, p)
            patcher.start()
            self.addCleanup(patcher.stop)
        self.p = mock.patch.object(u.pwd, "getpwnam",
                                   return_value=types.SimpleNamespace(pw_uid=1001))
        self.p.start()
        self.addCleanup(self.p.stop)
        # GitHub Actions runner is intentionally nonroot; tests use a fake
        # filesystem fixture while real CLI main() requires host-root.
        self.owner_check = mock.patch.object(u, "require_root_owned")
        self.owner_check.start()
        self.addCleanup(self.owner_check.stop)
        self.host = mock.patch.object(u, "machine_sha", return_value="a" * 64)
        self.host.start()
        self.addCleanup(self.host.stop)
        self.service = "inactive"
        self.sys = mock.patch.object(u, "systemd_status", side_effect=lambda: self.service)
        self.sys.start()
        self.addCleanup(self.sys.stop)
        self.native = []
        self.incus = mock.patch.object(u, "native_instances", side_effect=lambda _: list(self.native))
        self.incus.start()
        self.addCleanup(self.incus.stop)
        self.cmds = []
        self.command = mock.patch.object(u, "command", side_effect=self.fake_command)
        self.command.start()
        self.addCleanup(self.command.stop)
        self.paths["STATE"].mkdir()
        self.paths["UNIT"].parent.mkdir()
        self.paths["BINARY"].parent.mkdir()
        self.paths["UNIT"].write_text("[Service]\nExecStart=/usr/local/bin/boxweave\n")
        self.paths["BINARY"].write_bytes(b"boxweave-synthetic-binary")
        self.db = self.paths["STATE"] / "ledger.sqlite"
        db = sqlite3.connect(str(self.db))
        db.executescript("""
        CREATE TABLE schema_version(version INTEGER NOT NULL);
        INSERT INTO schema_version VALUES(5);
        CREATE TABLE operations(id TEXT,layer TEXT,box_uuid TEXT,state TEXT,native_operation_id TEXT);
        CREATE TABLE owned_boxes(uuid TEXT,project TEXT,name TEXT,state TEXT,deleted_at INTEGER);
        CREATE TABLE peer_grants(status TEXT);
        CREATE TABLE forward_claims(state TEXT);
        CREATE TABLE network_address_fences(state TEXT);
        CREATE TABLE exec_output_gc(state TEXT);
        """)
        db.commit()
        db.close()
        self.paths["STATE"].joinpath("install-manifest.json").write_text(json.dumps({
            "schema": 2,
            "native_project": "user-1001",
            "service_user": "boxweave",
            "binary_path": str(u.BINARY),
            "state_dir": str(u.STATE),
            "native_socket": "/var/lib/incus/unix.socket.user",
            "binary_sha256": u.file_fingerprint(u.BINARY)["sha256"],
            "unit_sha256": u.file_fingerprint(u.UNIT)["sha256"],
        }))
        self.box = "11111111-2222-3333-4444-555555555555"

    def fake_command(self, args, **kwargs):
        if args[:2] == ["systemctl", "show"]:
            return ("FragmentPath=" + str(u.UNIT) + "\nDropInPaths=", 0)
        self.cmds.append(args)
        if args[:2] == ["systemctl", "stop"]:
            self.service = "inactive"
        if args[:2] == ["incus", "delete"]:
            self.assertTrue(args[2].startswith("local:"))
            self.native = [b for b in self.native if b["name"] != args[2][6:]]
        if "native-prepare-rollback" in args:
            rec = json.loads(u.NATIVE.read_text())
            rec["Status"] = "ROLLED_BACK"
            u.NATIVE.write_text(json.dumps(rec))
        if "native-prepare-archive" in args:
            u.NATIVE.unlink()
        return ("", 0)

    def insert(self, sql, params=()):
        db = sqlite3.connect(str(self.db))
        db.execute(sql, params)
        db.commit()
        db.close()

    def plan(self, scope="service", boxes=(), digest=None):
        return u.inspect(scope, "user-1001", list(boxes), digest)

    def test_help_default_is_read_only(self):
        with mock.patch("sys.stdout") as output:
            self.assertEqual(u.main([]), 0)
            self.assertTrue(output.write.called)
        self.assertEqual(self.cmds, [])
        self.assertTrue(u.BINARY.exists())

    def test_service_plan_pure_stable_and_no_secret(self):
        a = self.plan()
        b = self.plan()
        self.assertEqual(a["review_sha256"], b["review_sha256"])
        self.assertTrue(a["ready"])
        self.assertEqual(a["sqlite"]["version"], 5)
        self.assertEqual(a["scope"], "service")
        self.assertEqual(self.cmds, [])
        self.assertTrue(self.db.exists())

    def test_service_apply_requires_fresh_approval_and_keeps_state(self):
        self.service = "active"
        plan = self.plan()
        with self.assertRaisesRegex(u.UninstallError, "host/scope/inventory drift"):
            # a change in service state between plan/apply invalidates review.
            self.service = "inactive"
            changed = self.plan()
            self.assertNotEqual(plan["review_sha256"], changed["review_sha256"])
            if changed["review_sha256"] != plan["review_sha256"]:
                raise u.UninstallError("host/scope/inventory drift")
        plan = self.plan()
        u.audited_write({"status": "STARTED", "review": plan["review_sha256"]})
        out = u.apply(plan)
        self.assertEqual(out["outcome"], "COMPLETE")
        self.assertFalse(u.UNIT.exists())
        self.assertFalse(u.BINARY.exists())
        self.assertTrue(self.db.exists())
        self.assertTrue((u.STATE / "install-manifest.json").exists())
        self.assertTrue(u.AUDIT.exists())
        self.assertFalse(any(a[0] == "incus" for a in self.cmds))

    def test_controller_purge_removes_only_known_state(self):
        p = self.plan("controller")
        self.assertTrue(p["ready"])
        out = u.apply(p)
        self.assertEqual(out["scope"], "controller")
        self.assertFalse(u.STATE.exists())
        self.assertFalse(u.UNIT.exists())
        self.assertFalse(u.BINARY.exists())
        self.assertEqual(self.native, [])

    def test_unknown_operation_fails_closed_all_scopes(self):
        self.insert("INSERT INTO operations VALUES(?,?,?,?,?)",
                    ("b" * 32, "user-1001", self.box, "UNKNOWN", "native1"))
        for scope in ("service", "controller", "full"):
            p = self.plan(scope)
            self.assertFalse(p["ready"])
            self.assertTrue(any("UNKNOWN:" in e for e in p["blockers"]))
        self.assertTrue(u.BINARY.exists())

    def test_foreign_manifest_and_symlink_rejected(self):
        path = u.STATE / "install-manifest.json"
        manifest = json.loads(path.read_text())
        manifest["native_project"] = "user-OTHER"
        path.write_text(json.dumps(manifest))
        with self.assertRaisesRegex(u.UninstallError, "provenance"):
            self.plan()
        path.write_text(json.dumps({**manifest, "native_project": "user-1001"}))
        u.UNIT.unlink()
        u.UNIT.symlink_to(self.db)
        with self.assertRaisesRegex(u.UninstallError, "unsafe path"):
            self.plan()

    def test_unmanaged_state_and_unrecognized_extra_file_refused(self):
        (u.STATE / "unexpected-secret").write_bytes(b"private")
        with self.assertRaisesRegex(u.UninstallError, "unrecognized owned state"):
            self.plan("controller")
        (u.STATE / "unexpected-secret").unlink()
        (u.STATE / "install-manifest.json").unlink()
        with self.assertRaisesRegex(u.UninstallError, "verified BoxWeave"):
            self.plan()

    def test_full_exact_box_uuid_allowlist_and_stopped_only(self):
        self.insert("INSERT INTO owned_boxes VALUES(?,?,?,?,?)",
                    (self.box, "user-1001", "bw-example", "Stopped", 0))
        self.native = [{"uuid": self.box, "name": "bw-example", "state": "Stopped"}]
        self.assertFalse(self.plan("full")["ready"])
        p = self.plan("full", [self.box])
        self.assertTrue(p["ready"])
        out = u.apply(p)
        self.assertEqual(out["outcome"], "COMPLETE")
        self.assertFalse(u.STATE.exists())
        self.assertFalse(self.native)
        self.assertTrue(any(cmd[:2] == ["incus", "delete"] for cmd in self.cmds))

    def test_full_running_box_never_autostopped_or_forced(self):
        self.insert("INSERT INTO owned_boxes VALUES(?,?,?,?,?)",
                    (self.box, "user-1001", "bw-example", "Running", 0))
        self.native = [{"uuid": self.box, "name": "bw-example", "state": "Running"}]
        plan = self.plan("full", [self.box])
        self.assertIn("NATIVE_BOX_NOT_STOPPED", plan["blockers"])
        self.assertFalse(plan["ready"])
        self.assertFalse(self.cmds)

    def test_native_rollback_order_uses_original_digest_and_retains_closed_audit(self):
        digest = "c" * 64
        u.NATIVE.write_text(json.dumps({"Schema": 1, "Status": "APPLIED",
                                         "Plan": {"ReviewSHA256": digest}}))
        self.assertFalse(self.plan("service")["ready"])
        self.assertFalse(self.plan("full")["ready"])
        p = self.plan("full", [], digest)
        self.assertTrue(p["ready"])
        u.apply(p)
        roll = next(i for i, a in enumerate(self.cmds) if "native-prepare-rollback" in a)
        arch = next(i for i, a in enumerate(self.cmds) if "native-prepare-archive" in a)
        disable = next(i for i, a in enumerate(self.cmds) if a[:2] == ["systemctl", "disable"])
        self.assertLess(roll, arch)
        self.assertLess(arch, disable)
        self.assertFalse(u.NATIVE.exists())
        self.assertFalse(u.BINARY.exists())

    def test_audit_replay_and_modified_unit_are_blocked(self):
        u.audited_write({"status": "STARTED"})
        with self.assertRaisesRegex(u.UninstallError, "previous uninstall interrupted"):
            u.audited_write({"status": "STARTED"})
        u.UNIT.write_text("malicious unit")
        with self.assertRaisesRegex(u.UninstallError, "modified"):
            self.plan()

    def test_staged_service_then_controller_requires_new_plan_and_retains_history(self):
        first = self.plan("service")
        u.audited_write({"status": "STARTED", "review_sha256": first["review_sha256"]})
        out = u.apply(first)
        u.audited_complete(first["review_sha256"], out)
        self.assertEqual(u.audit_status()["status"], "COMPLETE")
        next_plan = self.plan("controller")
        self.assertTrue(next_plan["ready"])
        self.assertNotEqual(first["review_sha256"], next_plan["review_sha256"])
        u.audited_write({"status": "STARTED", "review_sha256": next_plan["review_sha256"]})
        self.assertEqual(len(u.read_json(u.AUDIT)["history"]), 1)
        out2 = u.apply(next_plan)
        u.audited_complete(next_plan["review_sha256"], out2)
        self.assertFalse(u.STATE.exists())
        audit = u.read_json(u.AUDIT)
        self.assertEqual(audit["status"], "COMPLETE")
        self.assertEqual(audit["result"]["scope"], "controller")
        self.assertEqual(audit["history"][0]["result"]["scope"], "service")

    def test_interrupted_audit_blocks_plan_and_apply(self):
        first = self.plan()
        u.audited_write({"review_sha256": first["review_sha256"]})
        blocked = self.plan()
        self.assertFalse(blocked["ready"])
        self.assertIn("PRIOR_UNINSTALL_INTERRUPTED_REVIEW_ORIGINAL_AUDIT", blocked["blockers"])
        with self.assertRaisesRegex(u.UninstallError, "interrupted"):
            u.audited_write({"review_sha256": first["review_sha256"]})

    def test_overridden_effective_systemd_unit_refused(self):
        with mock.patch.object(u, "effective_unit", return_value="/run/systemd/system/boxweave.service"):
            with self.assertRaisesRegex(u.UninstallError, "effective systemd unit"):
                self.plan()

    def test_separate_root_peer_or_active_scriptlet_blocks(self):
        u.ROOT_UNIT.write_text("peer root helper")
        with self.assertRaisesRegex(u.UninstallError, "separate root"):
            self.plan()
        u.ROOT_UNIT.unlink()
        u.SCRIPTLET.write_text(json.dumps({"Status": "APPLIED"}))
        with self.assertRaisesRegex(u.UninstallError, "global Scriptlet"):
            self.plan("full")


if __name__ == "__main__":
    unittest.main()
