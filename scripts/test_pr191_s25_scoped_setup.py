#!/usr/bin/env python3
"""Source-only refusals for the dedicated Scoped nested setup helper."""
import os
import ipaddress
import re
from pathlib import Path
import subprocess
import unittest

from s25_r12_foreign_l1 import BLOCKED4, BLOCKED6, expected_denied_prefixes


class ScopedSetup(unittest.TestCase):
    def test_non_lab_refuses_before_any_candidate_call(self):
        helper = Path(__file__).with_name("pr191_s25_scoped_setup.sh")
        for env in ({}, {"P1B_S25_SCOPED": "1", "BW_S25_SETUP_LAYER": "outer"},
                    {"P1B_S25_SCOPED": "1", "BW_S25_SETUP_LAYER": "invalid"}):
            clean = {k: v for k, v in os.environ.items() if not k.startswith(("P1B_S25_", "BW_S25_", "GITHUB_", "RUNNER_"))}
            clean.update(env)
            with self.subTest(env=env):
                p = subprocess.run(["sh", str(helper), "invalid", "/not-a-candidate", "invalid", "invalid", "/not-a-workdir"],
                                   env=clean, capture_output=True, timeout=5)
                self.assertNotEqual(p.returncode, 0)
                self.assertNotIn(b"=PASS", p.stdout)

    def test_current_entrypoints_refuse_legacy_before_effects(self):
        root = Path(__file__).resolve().parent
        for name in ("s25_bridgehead_lab.sh", "s25_inner_setup.sh"):
            for value in ("", "0", "invalid"):
                env = dict(os.environ, P1B_S25_SCOPED=value)
                # No host access or setup arguments; early selector guard.
                p = subprocess.run(["sh", "-x", str(root / name)], env=env,
                                   capture_output=True, timeout=5)
                self.assertNotEqual(p.returncode, 0)
                self.assertEqual(p.stdout, b"")
                self.assertEqual(len(p.stderr.splitlines()), 2, p.stderr)
                self.assertIn(b" = 1", p.stderr)



class ScopedMountOracle(unittest.TestCase):
    def test_pre_bootstrap_and_post_bootstrap_mount_classification(self):
        source=Path(__file__).with_name("pr191_s25_scoped_setup.sh").read_text()
        rules=re.findall(r"awk '(.*?)' /proc/self/mountinfo",source)
        self.assertEqual(len(rules),2)
        def result(rule,path):
            row="1 2 3 / "+path+" rw - tmpfs tmpfs rw\n"
            return subprocess.run(["awk",rule],input=row,text=True,timeout=5).returncode
        for path in ("/var/lib/incus/devices","/var/lib/incus/shmounts","/var/lib/incus/guestapi"):
            self.assertNotEqual(result(rules[0],path),0)
            self.assertEqual(result(rules[1],path),0)
        for path in ("/var/lib/incus","/var/lib/incus/unix.socket","/var/lib/incus/unix.socket.user","/run/incus","/run/incus/unix.socket.user"):
            for rule in rules:
                self.assertNotEqual(result(rule,path),0)


class ScopedImageWitness(unittest.TestCase):
    def test_exact_source_image_transfer_not_proxy_expansion(self):
        root=Path(__file__).resolve().parent
        outer=(root/"s25_bridgehead_lab.sh").read_text()
        inner=(root/"s25_inner_setup.sh").read_text()
        foreign=(root/"s25_r12_foreign_l1.py").read_text()
        self.assertIn('image export "$image_fingerprint"',outer)
        self.assertIn('cat "$work/staging/s25-ubuntu.meta" "$work/staging/s25-ubuntu.meta.root" | sha256sum',outer)
        self.assertIn('test "$#" = 10',inner)
        self.assertIn("cat /root/s25-image/meta /root/s25-image/rootfs | sha256sum",inner)
        self.assertLess(inner.index("sha256sum -c -",inner.index("mkdir -m 0700 /root/s25-image")),inner.index("incus image import /root/s25-image/meta"))
        self.assertIn('test "$actual" = "$image_fingerprint"',inner)
        self.assertIn('assert h.hexdigest()==image',foreign)
        self.assertIn('"--project",FOREIGN_PROJECT,timeout=180',foreign)
        self.assertIn('"-c","F"',foreign)


class ScopedPolicyOracle(unittest.TestCase):
    def test_exact_gateway_exclusion_not_subnet_exclusion(self):
        for blocked, gateway, peers, permitted in (
            (BLOCKED4, "10.44.8.1", ("10.44.8.0", "10.44.8.2", "192.168.0.1", "198.18.0.1"), ("1.1.1.1",)),
            (BLOCKED6, "fd42:abcd::1", ("fd42:abcd::", "fd42:abcd::2", "::1"), ("fe80::1", "2001:db8:25:2026::2")),
        ):
            ranges=expected_denied_prefixes(blocked,gateway)
            self.assertEqual(len(ranges),len(set(ranges)))
            self.assertTrue(all(not a.overlaps(b) for i,a in enumerate(ranges) for b in ranges[i+1:]))
            self.assertFalse(any(ipaddress.ip_address(gateway) in a for a in ranges))
            for addr in peers:
                self.assertEqual(sum(ipaddress.ip_address(addr) in a for a in ranges),1)
            for addr in permitted:
                self.assertFalse(any(ipaddress.ip_address(addr) in a for a in ranges))
            old=sum(ipaddress.ip_network(x).num_addresses for x in blocked.split(","))
            self.assertEqual(sum(a.num_addresses for a in ranges),old-1)


if __name__ == "__main__":
    unittest.main()
