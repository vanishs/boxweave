"""Negative and exact-identity checks for the opt-in runtime adapter."""
import os
from pathlib import Path
import sqlite3
import subprocess
import shutil
import sys
import tempfile
import unittest

from bwmini.coordination_runtime import (
    app_target_identity, canonical_box_identity, gateway_target_identity,
    private_connection, scope_digest,
)
from bwmini.domain import Target
from bwmini.operation_coordination import CoordinationFault, OperationCoordinator


class CoordinationRuntimeTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.path = Path(self.tmp.name) / "coordination.sqlite"
        self.path.touch(mode=0o600)
        self.path.chmod(0o600)

    def assert_code(self, code, action):
        with self.assertRaises(CoordinationFault) as error:
            action()
        self.assertEqual(error.exception.code, code)

    def test_same_identity_across_verified_views(self):
        target = Target("containarium", "local", "default",
                        "project-a", "instance-a", "123", "project-a", "instance-a")
        observed = {"username": "project-a", "name": "instance-a", "createdAt": 123}
        verified = {"provider":"containarium", "backend":"local", "project":"default"}
        self.assertEqual(app_target_identity(target), gateway_target_identity(
            observed, scope=verified, logical="project-a"))
        observed["createdAt"] = 124
        self.assertNotEqual(app_target_identity(target), gateway_target_identity(
            observed, scope=verified, logical="project-a"))

    def test_missing_or_bad_identity_fails_closed(self):
        self.assert_code("COORDINATION_IDENTITY_INVALID", lambda: gateway_target_identity({}, scope={}, logical="box"))
        self.assert_code("COORDINATION_IDENTITY_INVALID",
                         lambda: canonical_box_identity("bad/box", "a", "123", "box",
                                                       provider="containarium", backend="local", project="default"))
        self.assert_code("COORDINATION_IDENTITY_INVALID",
                         lambda: canonical_box_identity("box", "\n", "123", "box",
                                                       provider="containarium", backend="local", project="default"))

    def test_cross_scope_and_recreation_are_distinct(self):
        target = Target("containarium", "local", "default", "box", "box-container", "123", "box", "box-container")
        base = app_target_identity(target)
        for scope in (("other", "local", "default"), ("containarium", "remote", "default"),
                      ("containarium", "local", "other")):
            self.assertNotEqual(base, canonical_box_identity(
                "box", "box-container", "123", "box", provider=scope[0],
                backend=scope[1], project=scope[2]))
        self.assertNotEqual(base, canonical_box_identity(
            "box", "box-container", "124", "box", provider="containarium",
            backend="local", project="default"))
        self.assertEqual(len(scope_digest("containarium", "local", "default")), 64)

    def test_gateway_requires_complete_verified_scope(self):
        observed = {"username": "box", "name": "box-container", "createdAt": 123}
        scope = {"provider":"containarium", "backend":"local", "project":"default"}
        self.assert_code("COORDINATION_IDENTITY_INVALID",
                         lambda: gateway_target_identity(observed, scope={}, logical="box"))
        self.assert_code("COORDINATION_IDENTITY_INVALID",
                         lambda: gateway_target_identity(observed, scope=scope, logical="other"))
        self.assert_code("COORDINATION_IDENTITY_INVALID",
                         lambda: gateway_target_identity(observed | {"createdAt":None},
                                                         scope=scope, logical="box"))

    def test_standalone_gateway_sibling_import(self):
        with tempfile.TemporaryDirectory() as folder:
            from bwmini import coordination_runtime, operation_coordination
            for mod in (coordination_runtime, operation_coordination):
                shutil.copyfile(Path(mod.__file__), Path(folder) / Path(mod.__file__).name)
            script = Path(folder) / "probe.py"
            script.write_text("from coordination_runtime import scope_digest, private_connection\n"
                              "print(scope_digest('containarium', 'local', 'default'))\n")
            result = subprocess.run([sys.executable, "-E", "-s", str(script)],
                                    capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(len(result.stdout.strip()), 64)

    def test_sqlite_schema_downgrade_or_app_db_refused(self):
        with sqlite3.connect(self.path) as db:
            db.execute("PRAGMA user_version=12")
        self.assert_code("COORDINATION_SCHEMA_UNSUPPORTED",
                         lambda: private_connection(self.path))
        with sqlite3.connect(self.path) as db:
            db.execute("PRAGMA user_version=0")
            db.execute("CREATE TABLE app_secrets (id TEXT)")
        self.assert_code("COORDINATION_SCHEMA_UNSUPPORTED",
                         lambda: private_connection(self.path))

    def test_unsafe_ancestor_or_sidecar(self):
        sub = Path(self.tmp.name) / "nested"
        sub.mkdir(mode=0o700)
        other = sub / "coord.sqlite"
        other.touch(mode=0o600)
        Path(self.tmp.name).chmod(0o777)
        self.assert_code("COORDINATION_STORAGE_UNSAFE", lambda: private_connection(other))
        Path(self.tmp.name).chmod(0o700)
        side = Path(str(other) + "-wal")
        side.symlink_to(self.path)
        self.assert_code("COORDINATION_STORAGE_UNSAFE", lambda: private_connection(other))

    def test_private_open_and_existing_db(self):
        store = OperationCoordinator(lambda: private_connection(self.path))
        store.initialize()
        row = private_connection(self.path).execute(
            "SELECT version FROM coordination_version").fetchone()
        self.assertEqual(row, (1,))

    def test_missing_storage_does_not_create_it(self):
        self.path.unlink()
        self.assert_code("COORDINATION_UNAVAILABLE", lambda: private_connection(self.path))
        self.assertFalse(self.path.exists())

    def test_directory_permissions(self):
        Path(self.tmp.name).chmod(0o755)
        self.assert_code("COORDINATION_STORAGE_UNSAFE", lambda: private_connection(self.path))

    def test_database_permissions(self):
        self.path.chmod(0o644)
        self.assert_code("COORDINATION_STORAGE_UNSAFE", lambda: private_connection(self.path))

    def test_file_symlink_refused(self):
        link = self.path.with_name("link.sqlite")
        link.symlink_to(self.path)
        self.assert_code("COORDINATION_STORAGE_UNSAFE", lambda: private_connection(link))

    def test_directory_symlink_refused(self):
        link = Path(self.tmp.name).with_name(Path(self.tmp.name).name + "-link")
        link.symlink_to(Path(self.tmp.name), target_is_directory=True)
        self.addCleanup(lambda: link.unlink(missing_ok=True))
        self.assert_code("COORDINATION_STORAGE_UNSAFE",
                         lambda: private_connection(link / "coordination.sqlite"))

    def test_relative_path_refused(self):
        self.assert_code("COORDINATION_STORAGE_UNSAFE",
                         lambda: private_connection("coordination.sqlite"))

    def test_unsafe_directory_structure_refused(self):
        self.assert_code("COORDINATION_STORAGE_UNSAFE",
                         lambda: private_connection(self.path.parent / ".." / self.path.parent.name / self.path.name))


if __name__ == "__main__":
    unittest.main()
