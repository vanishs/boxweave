#!/usr/bin/env python3
"""Exact installed Doctor/Recover operator UX, no effects."""
import json
import os
from pathlib import Path
import subprocess
import sys

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import token, pid
import uninstall_core as u
from pr188_h4b_installed import check as check_capacity

BIN = "/usr/local/bin/boxweave"

def read(args, good=True):
    r = subprocess.run(args, capture_output=True, text=True, timeout=100)
    assert (r.returncode == 0) == good, (r.returncode, r.stderr[-2000:])
    return json.loads(r.stdout), r.stderr

def main(project, receipt, scoped):
    assert os.geteuid() == 0
    before = pid("boxweave.service")
    installed = u.file_fingerprint(Path(BIN))
    manifest = u.file_fingerprint(u.STATE / "install-manifest.json")
    for _ in range(2):
        root, text = read([BIN, "doctor", "--summary"])
        service, _ = read(["runuser", "-u", "boxweave", "--", BIN, "doctor", "--summary"])
        r, s = root["diagnostic"], service["diagnostic"]
        assert r["scope"] == s["scope"] == project
        assert r["operator_summary"]["status"] == "HEALTHY_OBSERVED", r["operator_summary"]
        assert "Doctor " + project in text
        if scoped:
            assert s["installation"]["root_proof"] == "ROOT_AUTHORITY_REQUIRED"
            assert s["operator_summary"]["coverage_limits"] >= 1
            assert s["operator_summary"]["confirmed_issues"] == 0
        with LoopbackMCP("http://127.0.0.1:18650/mcp", token=token(receipt)) as c:
            mcp = c.host_status()
        prefix = root["address_capacity"]["ipv4_prefix"]
        assert prefix and prefix == service["address_capacity"]["ipv4_prefix"] == mcp["address_capacity"]["ipv4_prefix"]
        assert root["address_capacity"] == mcp["address_capacity"]
        check_capacity(mcp,root,scoped)
        assert r["forward_policy"]["state"] == "OBSERVED"
    m, _ = read([BIN, "doctor", "--maintenance", "--summary"])
    v = m["maintenance_readiness"]
    assert not v["effect_set_complete"] and not v["apply_available"]
    preview = u.maintenance_preview(project, "retain", u.machine_sha(), m, u.maintenance_local_inventory(project, int(project[5:])))
    assert not any(i["code"] == "OBSERVER_BLOCKER_PROJECTION_MISMATCH" for i in preview["blockers"]), preview["blockers"]
    absent = "d1-" + "0"*64
    p, _ = read([BIN,"recover","--guided","--issue",absent],good=False)
    assert p["status"] == "BLOCKED_UNPROVEN" and not p["can_repair"] and not p["apply_available"] and not p["native_retried"]
    assert u.file_fingerprint(Path(BIN)) == installed
    assert u.file_fingerprint(u.STATE / "install-manifest.json") == manifest
    assert pid("boxweave.service") == before
    print("PR188_H3B_INSTALLED_DIAGNOSTIC=PASS", flush=True)
    print("PR188_H4B_INSTALLED_CAPACITY_PARITY=PASS", flush=True)

if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2], sys.argv[3] == "1")
