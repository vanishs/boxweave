#!/usr/bin/env python3
"""Pure tests for the root-only host witness parser; no Incus/root needed."""
import importlib.util
from pathlib import Path
import unittest

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


if __name__ == "__main__":
    unittest.main()
