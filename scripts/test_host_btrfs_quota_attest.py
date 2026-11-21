#!/usr/bin/env python3
"""Pure tests for the root-only host witness parser; no Incus/root needed."""
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch

MODULE_PATH = Path(__file__).with_name("host_btrfs_quota_attest.py")
SPEC = importlib.util.spec_from_file_location("host_btrfs_quota_attest", MODULE_PATH)
audit = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(audit)


class HostBtrfsQuotaAttestTest(unittest.TestCase):
    def test_exact_btrfs_raw_qgroup_limit(self):
        qgroups = """Qgroupid Referenced Exclusive Max referenced Max exclusive Path
-------- ---------- --------- -------------- ------------- ----
0/256 16384 16384 none none srv/boxweave-incus
0/343 2147483648 1522532352 2147483648 none srv/boxweave-incus/containers/user-1001_quota-probe-1119
"""
        expected = "srv/boxweave-incus/containers/user-1001_quota-probe-1119"
        self.assertEqual(audit.qgroup_limit(qgroups, expected), (2147483648, 2147483648))
        with self.assertRaisesRegex(ValueError, "absent"):
            audit.qgroup_limit(qgroups, "srv/boxweave-incus/containers/user-1002_quota-probe-1119")
        with self.assertRaisesRegex(ValueError, "nonunique"):
            audit.qgroup_limit(qgroups + qgroups.splitlines()[-1] + "\n", expected)

    def test_missing_max_is_not_verified(self):
        path = "srv/boxweave-incus/containers/user-1001_quota-probe-1119"
        with self.assertRaisesRegex(ValueError, "numeric"):
            audit.qgroup_limit("0/343 2147483648 1000 none none " + path, path)

    def test_near_full_boundary(self):
        quota = 2 << 30
        self.assertTrue(audit.near_full(2147467264, quota, quota))
        self.assertTrue(audit.near_full(quota - (64 << 10), quota, quota))
        self.assertFalse(audit.near_full(quota - (64 << 10) - 1, quota, quota))
        self.assertFalse(audit.near_full(quota + 1, quota, quota))
        self.assertFalse(audit.near_full(quota, quota, quota + 1))

    def test_exact_refusal_marker_required(self):
        with patch.object(audit, "run", return_value="GUEST_EDQUOT_128K\n") as run:
            audit.verify_write_refusal("incus", "user-1001", "lab")
            self.assertEqual(run.call_args.args[:7],
                             ("incus", "exec", "lab", "--project", "user-1001",
                              "--mode=non-interactive", "--"))
        for returned in ("ENOSPC", "", "GUEST_EDQUOT_128K extra"):
            with patch.object(audit, "run", return_value=returned):
                with self.assertRaises(ValueError):
                    audit.verify_write_refusal("incus", "user-1001", "lab")

    def test_probe_requires_exact_uuid_before_native_command(self):
        from unittest.mock import Mock
        native = {"config": {"volatile.uuid": "382c4dc2-d82a-4477-a098-1dd2c5d0d640"}}
        with patch.object(audit.shutil, "which", return_value="incus"), \
                patch.object(audit, "incus_json", return_value=native), \
                patch.object(audit, "verify_write_refusal", Mock()) as probe:
            for expected in (None, "00000000-0000-0000-0000-000000000000"):
                with self.assertRaises(ValueError):
                    audit.audit("user-1001", "lab", True, expected)
            probe.assert_not_called()


if __name__ == "__main__":
    unittest.main()
