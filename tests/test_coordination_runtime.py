"""Negative and exact-identity checks for the opt-in runtime adapter."""
import os
from pathlib import Path
import sqlite3
import tempfile
import unittest

from bwmini.coordination_runtime import (
    app_target_identity, canonical_box_identity, gateway_target_identity,
    private_connection,
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
        self.assertEqual(app_target_identity(target), gateway_target_identity(observed))
        observed["createdAt"] = 124
        self.assertNotEqual(app_target_identity(target), gateway_target_identity(observed))

    def test_missing_or_bad_identity_fails_closed(self):
        self.assert_code("COORDINATION_IDENTITY_INVALID", lambda: gateway_target_identity({}))
        self.assert_code("COORDINATION_IDENTITY_INVALID",
                         lambda: canonical_box_identity("bad/box", "a", "123", "box"))
        self.assert_code("COORDINATION_IDENTITY_INVALID",
                         lambda: canonical_box_identity("box", "\n", "123", "box"))

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
