#!/usr/bin/env python3
"""Offline negative controls for host uninstaller: never touch a real host."""
import contextlib
import fcntl
import io
import json
from pathlib import Path
import sqlite3
import tempfile
import types
import unittest
from unittest import mock

import uninstall_core as u

REAL_NATIVE_INSTANCES = u.native_instances
REAL_REQUIRE_QUIET = u.require_quiescent_native


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
            "RETIREMENT_UNIT": root / "units" / "boxweave-ipv4-retirement.service",
            "RETIREMENT_SOCKET": root / "retirement.sock",
            "AUDIT": root / "uninstall-audit.json",
            "BACKUP": root / "installer-backup",
            "RETIRED": root / "retired",
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
        self.host_state_check = mock.patch.object(u, "validate_state_dir",
                                                  side_effect=lambda path, uid: u.checked_file(path, "dir"))
        self.host_state_check.start()
        self.addCleanup(self.host_state_check.stop)
        self.lock_owner_check = mock.patch.object(u, "require_service_lock_owner")
        self.lock_owner_check.start()
        self.addCleanup(self.lock_owner_check.stop)
        self.lock_fd_owner_check = mock.patch.object(u, "require_serve_lock_fd_owner")
        self.lock_fd_owner_check.start()
        self.addCleanup(self.lock_fd_owner_check.stop)
        self.host = mock.patch.object(u, "machine_sha", return_value="a" * 64)
        self.host.start()
        self.addCleanup(self.host.stop)
        self.service = "inactive"
        self.sys = mock.patch.object(u, "systemd_status", side_effect=lambda: self.service)
        self.sys.start()
        self.addCleanup(self.sys.stop)
        self.native = []
        self.incus = mock.patch.object(u, "native_instances", side_effect=lambda _, guided=False: list(self.native))
        self.incus.start()
        self.addCleanup(self.incus.stop)
        self.cmds = []
        self.quiet = mock.patch.object(u, "require_quiescent_native")
        self.quiet.start()
        self.addCleanup(self.quiet.stop)
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
        CREATE TABLE operations(id TEXT,layer TEXT,box_uuid TEXT,state TEXT,native_operation_id TEXT,
                                action TEXT DEFAULT '', incarnation TEXT DEFAULT '');
        CREATE TABLE owned_boxes(uuid TEXT,project TEXT,name TEXT,state TEXT,deleted_at INTEGER,
                                 incarnation TEXT DEFAULT '', deleted_by_operation_id TEXT DEFAULT '');
        CREATE TABLE recovery_audit(seq INTEGER PRIMARY KEY, operation_id TEXT, decision TEXT, at INTEGER, evidence_digest TEXT);
        CREATE TABLE peer_grants(status TEXT);
        CREATE TABLE forward_claims(state TEXT);
        CREATE TABLE network_address_fences(state TEXT);
        CREATE TABLE exec_output_gc(state TEXT);
        """)
        db.commit()
        db.close()
        self.db.chmod(0o600)
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
        self.paths["STATE"].joinpath("install-manifest.json").chmod(0o600)
        self.box = "11111111-2222-3333-4444-555555555555"

    def fake_command(self, args, **kwargs):
        if args[:2] == ["systemctl", "show"]:
            return ("FragmentPath=" + str(u.UNIT) + "\nDropInPaths=", 0)
        self.cmds.append(args)
        if args[:2] == ["systemctl", "stop"]:
            self.service = "inactive"
        if args[:2] == ["incus", "stop"]:
            self.native = [{**b, "state": "Stopped"} if b["name"] == args[2][6:] else b
                           for b in self.native]
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

    def install_retirement_fixture(self):
        self.insert("UPDATE schema_version SET version=6")
        self.insert("CREATE TABLE network_ipv4_compactions (project TEXT, delete_operation_id TEXT)")
        bridge, acl, digest = "incusbr-1001", "bw-test-1001", "a"*64
        unit = u.retirement_unit_body("user-1001", bridge, acl)
        u.RETIREMENT_UNIT.write_bytes(unit)
        u.RETIREMENT_UNIT.chmod(0o644)
        manifest = u.STATE / "install-manifest.json"
        data = json.loads(manifest.read_text())
        data.update(retirement_unit_sha256=u.sha(unit), retirement_bridge=bridge,
                    retirement_acl=acl, retirement_native_review_sha256=digest)
        manifest.write_text(json.dumps(data))
        u.NATIVE.write_text(json.dumps({"schema": 1, "status": "APPLIED", "plan": {
            "target": {"project": "user-1001", "network": bridge, "acl": acl}, "review_sha256": digest}}))
        u.NATIVE.chmod(0o600)
        p = mock.patch.object(u, "retirement_service_state", return_value="inactive")
        p.start()
        self.addCleanup(p.stop)
        return digest

    def test_schema6_owned_retirement_full_cleanup(self):
        digest = self.install_retirement_fixture()
        plan = self.plan("full", digest=digest)
        self.assertTrue(plan["ready"], plan["blockers"])
        self.assertEqual(plan["sqlite"]["version"], 6)
        self.assertEqual(plan["retirement_verifier"]["state"], "inactive")
        out = u.apply(plan)
        self.assertEqual(out["outcome"], "COMPLETE")
        self.assertFalse(u.RETIREMENT_UNIT.exists())
        self.assertFalse(u.BINARY.exists())
        self.assertFalse(u.STATE.exists())
        self.assertIn(["systemctl", "disable", u.RETIREMENT_UNIT.name], self.cmds)

    def test_schema6_compaction_protects_all_uninstall_scopes(self):
        digest = self.install_retirement_fixture()
        self.insert("INSERT INTO network_ipv4_compactions VALUES(?,?)", ("user-1001", "b"*32))
        for scope in ("service", "controller", "full"):
            p = self.plan(scope, digest=digest if scope=="full" else None)
            self.assertFalse(p["ready"])
            self.assertIn("IPV4_COMPACTION_PROTECTED:1", p["blockers"])
        self.assertEqual(self.cmds, [])
        self.assertTrue(u.RETIREMENT_UNIT.exists())

    def test_schema6_retirement_unit_drift_and_orphan_refused(self):
        digest = self.install_retirement_fixture()
        plan = self.plan("full", digest=digest)
        u.RETIREMENT_UNIT.write_bytes(u.RETIREMENT_UNIT.read_bytes()+b"# altered")
        with self.assertRaisesRegex(u.UninstallError, "identity changed"):
            self.plan("full", digest=digest)
        with self.assertRaisesRegex(u.UninstallError, "not stopped or changed"):
            u.apply(plan)
        self.assertTrue(u.BINARY.exists())

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
        earlier = self.plan()
        self.service = "inactive"
        changed = self.plan()
        self.assertNotEqual(earlier["review_sha256"], changed["review_sha256"])
        with mock.patch.object(u.os, "geteuid", return_value=0), \
                contextlib.redirect_stdout(io.StringIO()):
            with self.assertRaisesRegex(u.UninstallError, "host/scope/inventory drift"):
                u.main(["--apply", "--scope", "service", "--project", "user-1001",
                        "--confirm-host-sha256", earlier["host_machine_id_sha256"],
                        "--confirm-review-sha256", earlier["review_sha256"]])
            self.assertFalse(u.AUDIT.exists())
            self.assertEqual(u.main([
                "--apply", "--scope", "service", "--project", "user-1001",
                "--confirm-host-sha256", changed["host_machine_id_sha256"],
                "--confirm-review-sha256", changed["review_sha256"],
            ]), 0)
        self.assertEqual(u.audit_status()["status"], "COMPLETE")
        self.assertFalse(u.UNIT.exists())
        self.assertFalse(u.BINARY.exists())
        self.assertTrue(self.db.exists())
        self.assertTrue((u.STATE / "install-manifest.json").exists())
        self.assertTrue(u.AUDIT.exists())
        self.assertFalse(any(a[0] == "incus" for a in self.cmds))

    def test_apply_wrong_hash_and_nonroot_refused_before_audit(self):
        p = self.plan()
        args = ["--apply", "--scope", "service", "--project", "user-1001",
                "--confirm-host-sha256", p["host_machine_id_sha256"],
                "--confirm-review-sha256", "f" * 64]
        with mock.patch.object(u.os, "geteuid", return_value=0):
            with self.assertRaisesRegex(u.UninstallError, "drift"):
                u.main(args)
        with mock.patch.object(u.os, "geteuid", return_value=1001):
            with self.assertRaisesRegex(u.UninstallError, "root on"):
                u.main(args)
        self.assertFalse(u.AUDIT.exists())
        self.assertTrue(u.BINARY.exists())

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
        self.insert("INSERT INTO operations(id,layer,box_uuid,state,native_operation_id) VALUES(?,?,?,?,?)",
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
        self.insert("INSERT INTO owned_boxes(uuid,project,name,state,deleted_at) VALUES(?,?,?,?,?)",
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
        self.insert("INSERT INTO owned_boxes(uuid,project,name,state,deleted_at) VALUES(?,?,?,?,?)",
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
        u.NATIVE.chmod(0o600)
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

    def test_untrusted_root_journal_permissions_fail_closed(self):
        u.NATIVE.write_text(json.dumps({"Schema": 1, "Status": "APPLIED",
                                         "Plan": {"ReviewSHA256": "b" * 64}}))
        u.NATIVE.chmod(0o644)
        with self.assertRaisesRegex(u.UninstallError, "untrusted root journal permission"):
            self.plan()
        u.NATIVE.unlink()
        u.SCRIPTLET.write_text(json.dumps({"Status": "ROLLED_BACK"}))
        u.SCRIPTLET.chmod(0o644)
        with self.assertRaisesRegex(u.UninstallError, "untrusted root journal permission"):
            self.plan("full")

    def make_serve_lock(self):
        path = u.STATE / u.SERVE_LOCK_NAME
        path.write_bytes(b"")
        path.chmod(0o600)
        return path

    def test_persistent_serve_lock_is_recognized_in_read_only_plan(self):
        lock = self.make_serve_lock()
        self.service = "active"
        original = lock.stat()
        plan = self.plan("full")
        self.assertTrue(plan["ready"])
        self.assertIn(u.SERVE_LOCK_NAME, [x["name"] for x in plan["state_files"]])
        self.assertEqual(lock.stat().st_ino, original.st_ino)
        self.assertEqual(self.cmds, [])
        self.assertEqual(u.STATE.joinpath("ledger.sqlite").stat().st_size > 0, True)

    def test_service_only_keeps_permanent_serve_lock_and_sqlite(self):
        lock = self.make_serve_lock()
        self.service = "active"
        p = self.plan("service")
        self.assertTrue(p["ready"])
        u.apply(p)
        self.assertTrue(lock.exists())
        self.assertTrue(self.db.exists())
        self.assertEqual(self.service, "inactive")
        self.assertFalse(u.BINARY.exists())

    def test_controller_removes_serve_lock_last_after_sqlite(self):
        lock = self.make_serve_lock()
        plan = self.plan("controller")
        seen = []
        original = u.exact_unlink
        def observe(path, proof):
            seen.append(path.name)
            return original(path, proof)
        with mock.patch.object(u, "exact_unlink", side_effect=observe):
            out = u.apply(plan)
        self.assertEqual(out["outcome"], "COMPLETE")
        self.assertFalse(lock.exists())
        self.assertFalse(u.STATE.exists())
        self.assertLess(seen.index("ledger.sqlite"), seen.index(u.SERVE_LOCK_NAME))
        self.assertEqual(seen[-1], u.SERVE_LOCK_NAME)

    def test_held_serve_lock_blocks_destructive_apply(self):
        lock = self.make_serve_lock()
        plan = self.plan("full")
        with lock.open("r+b") as held:
            fcntl.flock(held, fcntl.LOCK_EX | fcntl.LOCK_NB)
            with self.assertRaisesRegex(u.UninstallError, "serve lock held"):
                u.apply(plan)
        self.assertTrue(lock.exists())
        self.assertTrue(u.BINARY.exists())
        self.assertTrue(self.db.exists())
        self.assertFalse(any(x[:2] == ["systemctl", "disable"] for x in self.cmds))

    def test_replaced_or_new_serve_lock_refuses_apply(self):
        lock = self.make_serve_lock()
        planned = self.plan("controller")
        lock.unlink()
        lock = self.make_serve_lock()
        # Ensure inode drift even on filesystems that quickly recycle.
        lock.write_bytes(b"changed inode or content")
        with self.assertRaisesRegex(u.UninstallError, "serve lock inode changed"):
            u.apply(planned)
        lock.unlink()
        planned = self.plan("controller")
        self.make_serve_lock()
        with self.assertRaisesRegex(u.UninstallError, "serve lock appeared"):
            u.apply(planned)
        self.assertTrue(u.BINARY.exists())
        self.assertTrue(self.db.exists())

    def test_unsafe_serve_lock_symlink_hardlink_and_mode_refused(self):
        lock = self.make_serve_lock()
        lock.chmod(0o644)
        with self.assertRaisesRegex(u.UninstallError, "unsafe serve lock"):
            self.plan("full")
        lock.chmod(0o600)
        hardlink = u.STATE.parent / "another-serve-lock-reference"
        hardlink.hardlink_to(lock)
        with self.assertRaisesRegex(u.UninstallError, "unsafe serve lock"):
            self.plan("full")
        hardlink.unlink()
        lock.unlink()
        lock.symlink_to(self.db)
        with self.assertRaisesRegex(u.UninstallError, "unsafe path"):
            self.plan("full")

    def test_guided_retires_exact_running_box_and_archives(self):
        self.insert("INSERT INTO owned_boxes(uuid,project,name,state,deleted_at) VALUES(?,?,?,?,?)",
                    (self.box, "user-1001", "bw-example", "OWNED", 0))
        self.native = [{"uuid": self.box, "name": "bw-example", "state": "Running"}]
        plan = u.inspect("full", "user-1001", [], None, guided=True)
        self.assertTrue(plan["ready"])
        self.assertEqual(plan["allowlist"], [self.box])
        result = u.apply(plan)
        dest = Path(result["retirement_archive"])
        self.assertTrue((dest / "state-ledger.sqlite").is_file())
        self.assertTrue((dest / "boxweave.bin").is_file())
        self.assertFalse(self.native)
        self.assertLess(next(i for i,a in enumerate(self.cmds) if a[:2] == ["incus","stop"]),
                        next(i for i,a in enumerate(self.cmds) if a[:2] == ["incus","delete"]))

    def test_guided_exec_archive_preserves_contents_and_rejects_unknown_files(self):
        directory = u.STATE / u.EXEC_ARCHIVE_NAME
        directory.mkdir(mode=0o700)
        entry = directory / ("a" * 32 + ".stdout")
        entry.write_bytes(b"original")
        entry.chmod(0o600)
        with mock.patch.object(u.pwd, "getpwnam", return_value=types.SimpleNamespace(pw_uid=directory.stat().st_uid)):
            snapshot = u.inspect_exec_archive(directory.stat().st_uid)
            self.assertEqual(len(snapshot["files"]), 1)
            bad = directory / "unexpected"
            bad.write_text("no")
            with self.assertRaisesRegex(u.UninstallError, "unrecognized"):
                u.inspect_exec_archive(directory.stat().st_uid)
            bad.unlink()
        # Test the archive/retirement path against the validated proof without
        # changing real host ownership.
        real_inspect = u.inspect_exec_archive
        with mock.patch.object(u, "inspect_exec_archive",
                               side_effect=lambda uid: real_inspect(directory.stat().st_uid)):
            plan = u.inspect("full", "user-1001", [], None, guided=True)
            self.assertTrue(plan["ready"])
            result = u.apply(plan)
        self.assertEqual((Path(result["retirement_archive"]) / ("exec-" + entry.name)).read_bytes(), b"original")
        self.assertFalse(u.STATE.exists())

    def test_guided_history_requires_exact_released_exec_and_successful_delete(self):
        self.insert("INSERT INTO operations VALUES(?,?,?,?,?,?,?)",
                    ("e"*32, "user-1001", self.box, "UNKNOWN", "native1", "box_exec", self.box))
        self.insert("INSERT INTO operations VALUES(?,?,?,?,?,?,?)",
                    ("d"*32, "user-1001", self.box, "SUCCEEDED", "native2", "box_delete", self.box))
        self.insert("INSERT INTO owned_boxes VALUES(?,?,?,?,?,?,?)",
                    (self.box, "user-1001", "old", "DELETED", 20, self.box, "d"*32))
        self.assertFalse(u.inspect("full", "user-1001", [], None, guided=True)["ready"])
        self.insert("INSERT INTO recovery_audit VALUES(?,?,?,?,?)",
                    (1, "e"*32, "RELEASED", 10, "a"*64))
        plan = u.inspect("full", "user-1001", [], None, guided=True)
        self.assertTrue(plan["ready"])
        self.assertEqual(len(plan["sqlite"]["historical_released_exec"]), 1)
        self.assertFalse(self.plan("full")["ready"])
        self.insert("UPDATE operations SET state='FAILED' WHERE id=?", ("d"*32,))
        self.assertFalse(u.inspect("full", "user-1001", [], None, guided=True)["ready"])

    def test_guided_plan_reads_exec_archive_without_any_mutation(self):
        directory = u.STATE / u.EXEC_ARCHIVE_NAME
        directory.mkdir(mode=0o700)
        entry = directory / ("a" * 32 + ".stderr")
        entry.write_bytes(b"original-output")
        entry.chmod(0o600)
        original = u.inspect_exec_archive
        # CI synthetic user != host's dedicated uid; preserve all other checks.
        with mock.patch.object(u, "inspect_exec_archive",
                               side_effect=lambda uid: original(directory.stat().st_uid)), \
                mock.patch.object(u.os, "geteuid", return_value=0), \
                contextlib.redirect_stdout(io.StringIO()) as output:
            self.assertEqual(u.main(["--plan", "--guided-full-plan", "--scope", "full",
                                     "--project", "user-1001"]), 0)
        data = json.loads(output.getvalue())
        self.assertTrue(data["ready"])
        self.assertEqual(data["scope"], "full")
        self.assertTrue(data["guided"])
        self.assertEqual(len(data["exec_archive"]["files"]), 1)
        self.assertEqual(entry.read_bytes(), b"original-output")
        self.assertFalse(u.AUDIT.exists())
        self.assertEqual(self.cmds, [])

    def test_guided_terminal_timeout_and_released_historical_start(self):
        self.insert("INSERT INTO operations VALUES(?,?,?,?,?,?,?)",
                    ("s"*32, "user-1001", self.box, "UNKNOWN", "expired", "box_start", self.box))
        self.insert("INSERT INTO operations VALUES(?,?,?,?,?,?,?)",
                    ("w"*32, "user-1001", self.box, "TIMED_OUT", "", "box_wait_ready", self.box))
        self.insert("INSERT INTO operations VALUES(?,?,?,?,?,?,?)",
                    ("d"*32, "user-1001", self.box, "SUCCEEDED", "native", "box_delete", self.box))
        self.insert("INSERT INTO owned_boxes VALUES(?,?,?,?,?,?,?)",
                    (self.box, "user-1001", "retired", "DELETED", 20, self.box, "d"*32))
        # No RELEASED proof => fail closed even though native Box no longer exists.
        self.assertFalse(u.inspect("full", "user-1001", [], None, guided=True)["ready"])
        self.insert("INSERT INTO recovery_audit VALUES(?,?,?,?,?)",
                    (1, "s"*32, "RELEASED", 10, "a"*64))
        plan = u.inspect("full", "user-1001", [], None, guided=True)
        self.assertTrue(plan["ready"], plan["blockers"])
        self.assertEqual(plan["sqlite"]["unresolved_operations"], 0)
        self.assertEqual(len(plan["sqlite"]["historical_released_other"]), 1)
        self.assertFalse(self.plan("full")["ready"])
        self.insert("UPDATE operations SET state='FAILED' WHERE id=?", ("d"*32,))
        self.assertFalse(u.inspect("full", "user-1001", [], None, guided=True)["ready"])

    def test_guided_released_unknown_cannot_bypass_network_original(self):
        self.insert("INSERT INTO operations VALUES(?,?,?,?,?,?,?)",
                    ("p"*32, "user-1001", self.box, "UNKNOWN", "expired", "box_peer_set", self.box))
        self.insert("INSERT INTO operations VALUES(?,?,?,?,?,?,?)",
                    ("d"*32, "user-1001", self.box, "SUCCEEDED", "native", "box_delete", self.box))
        self.insert("INSERT INTO owned_boxes VALUES(?,?,?,?,?,?,?)",
                    (self.box, "user-1001", "retired", "DELETED", 20, self.box, "d"*32))
        self.insert("INSERT INTO recovery_audit VALUES(?,?,?,?,?)",
                    (1, "p"*32, "RELEASED", 10, "a"*64))
        self.assertFalse(u.inspect("full", "user-1001", [], None, guided=True)["ready"])

    def test_guided_archives_verified_prior_installer_backup(self):
        u.BACKUP.mkdir(mode=0o700)
        previous = {"schema": 2, "native_project": "user-1001",
                    "native_socket": "/var/lib/incus/unix.socket.user",
                    "service_user": "boxweave", "binary_path": str(u.BINARY),
                    "state_dir": str(u.STATE),
                    "binary_sha256": u.sha(b"prior-binary")}
        files = {"binary": b"prior-binary", "unit": b"prior-unit"}
        for name, body in files.items():
            (u.BACKUP / name).write_bytes(body)
            (u.BACKUP / name).chmod(0o600)
        receipt = {"Schema": 1, "Previous": previous,
                   "PreviousUnitSHA256": u.sha(files["unit"]),
                   "PreviousDBSHA256": "", "PreviousWALSHA256": "",
                   "PreviousSHMSHA256": ""}
        (u.BACKUP / "receipt.json").write_text(json.dumps(receipt))
        (u.BACKUP / "receipt.json").chmod(0o600)
        self.assertFalse(self.plan("full")["ready"])
        plan = u.inspect("full", "user-1001", [], None, guided=True)
        self.assertTrue(plan["ready"], plan["blockers"])
        self.assertEqual(len(plan["installer_backup"]["files"]), 3)
        result = u.apply(plan)
        saved = Path(result["retirement_archive"])
        for name, body in files.items():
            self.assertEqual((saved / ("installer-backup-" + name)).read_bytes(), body)
        self.assertTrue((saved / "installer-backup-receipt.json").is_file())
        self.assertFalse(u.BACKUP.exists())

    def test_guided_untrusted_backup_cannot_be_retired(self):
        u.BACKUP.mkdir(mode=0o700)
        (u.BACKUP / "unexpected").write_text("foreign")
        with self.assertRaises(u.UninstallError):
            u.inspect("full", "user-1001", [], None, guided=True)
        self.assertTrue((u.BACKUP / "unexpected").exists())

    def prepare_interrupted_native_retirement(self):
        self.install_retirement_fixture()
        plan = u.inspect("full", "user-1001", [], None, guided=True)
        self.assertTrue(plan["ready"], plan["blockers"])
        u.audited_write({"status": "STARTED", "scope": "full",
                         "host_sha256": plan["host_machine_id_sha256"],
                         "review_sha256": plan["review_sha256"],
                         "actions": plan["actions"],
                         "historical_released_exec": plan["sqlite"]["historical_released_exec"],
                         "historical_released_other": plan["sqlite"]["historical_released_other"]})
        archive = u.archive_controller(plan)
        self.assertEqual(u.audit_status()["status"], "STARTED")
        return plan, archive

    def test_interrupted_retirement_exact_archive_resume_and_audit(self):
        plan, archive = self.prepare_interrupted_native_retirement()
        receipt, old_plan = u.inspect_interrupted_retirement("user-1001", archive)
        self.assertTrue(receipt["ready"])
        self.assertEqual(receipt["phase"], "ARCHIVED_BEFORE_NATIVE_ROLLBACK")
        self.assertEqual(old_plan["review_sha256"], plan["review_sha256"])
        self.assertEqual(self.cmds, [])
        result = u.resume_interrupted_retirement(old_plan, archive)
        self.assertEqual(result["outcome"], "COMPLETE")
        self.assertEqual(result["retirement_archive"], archive)
        self.assertEqual(u.audit_status()["status"], "COMPLETE")
        self.assertFalse(u.STATE.exists())
        self.assertFalse(u.BINARY.exists())
        self.assertFalse(u.NATIVE.exists())
        self.assertEqual(len(list(u.RETIRED.glob("retired-*"))), 1)
        self.assertTrue((Path(archive) / "review.json").is_file())

    def test_interrupted_resume_rejects_archive_drift_without_effect(self):
        plan, archive = self.prepare_interrupted_native_retirement()
        (Path(archive) / "boxweave.bin").write_bytes(b"drift")
        with self.assertRaisesRegex(u.UninstallError, "copy changed"):
            u.inspect_interrupted_retirement("user-1001", archive)
        self.assertEqual(self.cmds, [])
        self.assertEqual(u.audit_status()["status"], "STARTED")
        self.assertTrue(u.NATIVE.exists())

    def test_interrupted_resume_rejects_original_audit_drift(self):
        plan, archive = self.prepare_interrupted_native_retirement()
        audit = json.loads(u.AUDIT.read_text())
        audit["review_sha256"] = "c" * 64
        u.AUDIT.write_text(json.dumps(audit))
        with self.assertRaisesRegex(u.UninstallError, "audit and archived plan disagree"):
            u.inspect_interrupted_retirement("user-1001", archive)
        self.assertEqual(self.cmds, [])
        self.assertTrue(u.BINARY.exists())

    def test_interrupted_resume_rejects_native_inflight(self):
        plan, archive = self.prepare_interrupted_native_retirement()
        native = json.loads(u.NATIVE.read_text())
        native["status"] = "ROLLBACK_IN_FLIGHT"
        u.NATIVE.write_text(json.dumps(native))
        with self.assertRaises(u.UninstallError):
            u.inspect_interrupted_retirement("user-1001", archive)
        self.assertEqual(self.cmds, [])
        self.assertEqual(u.audit_status()["status"], "STARTED")

    def test_guided_native_dependency_and_live_operation_refusals(self):
        entry = {"name": "bw-example", "type": "container", "status": "Running",
                 "config": {"volatile.uuid": self.box},
                 "expanded_devices": {"p": {"type": "proxy"}}}
        with mock.patch.object(u, "command", return_value=(json.dumps([entry]), 0)):
            with self.assertRaisesRegex(u.UninstallError, "native proxy"):
                REAL_NATIVE_INSTANCES("user-1001", guided=True)
        with mock.patch.object(u, "command", return_value=('[{"id":"active"}]', 0)):
            with self.assertRaisesRegex(u.UninstallError, "operations present"):
                REAL_REQUIRE_QUIET("user-1001")

    def test_guided_denial_and_drift_do_not_mutate(self):
        with mock.patch.object(u.os, "geteuid", return_value=0), \
                mock.patch("builtins.open", side_effect=[io.StringIO("NO\n"), io.StringIO()]), \
                contextlib.redirect_stdout(io.StringIO()):
            with self.assertRaisesRegex(u.UninstallError, "not approved"):
                u.main(["--full"])
        self.assertFalse(u.AUDIT.exists())
        self.assertTrue(u.BINARY.exists())
        initial = u.inspect("full", "user-1001", [], None, guided=True)
        changed = {**initial, "review_sha256": "0"*64}
        with mock.patch.object(u.os, "geteuid", return_value=0), \
                mock.patch("builtins.open", side_effect=[io.StringIO("REMOVE user-1001\n"), io.StringIO()]), \
                mock.patch.object(u, "inspect", side_effect=[initial, changed]), \
                contextlib.redirect_stdout(io.StringIO()):
            with self.assertRaisesRegex(u.UninstallError, "drift"):
                u.main(["--full"])
        self.assertFalse(u.AUDIT.exists())

    def test_separate_root_peer_or_active_scriptlet_blocks(self):
        u.ROOT_UNIT.write_text("peer root helper")
        with self.assertRaisesRegex(u.UninstallError, "separate root"):
            self.plan()
        u.ROOT_UNIT.unlink()
        u.SCRIPTLET.write_text(json.dumps({"Status": "APPLIED"}))
        u.SCRIPTLET.chmod(0o600)
        with self.assertRaisesRegex(u.UninstallError, "global Scriptlet"):
            self.plan("full")


if __name__ == "__main__":
    unittest.main()
