#!/usr/bin/env python3
"""H1B exact-head read-only preview/recheck acceptance on disposable installed labs."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

from pr188_h1a_observe import snapshot


def run(*args):
    return subprocess.check_output(args, text=True, timeout=60).rstrip("\n")


def read_plan(base, action, retention, extra=()):
    result = subprocess.run(base + [action, "--retention", retention, *extra],
                            capture_output=True, text=True, timeout=75)
    assert result.returncode in (0, 2), (result.returncode, result.stderr[:300])
    p = json.loads(result.stdout)
    assert p["schema"] == 1 and p["scope"] == "full"
    assert not p["ready"] and not p["apply_available"] and not p["effect_set_complete"]
    assert p["effects_are_unordered_preview"] and not p["cross_source_atomic"]
    assert "review_sha256" not in p and len(p["preview_sha256"]) == 64
    return p, result.returncode


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("project")
    parser.add_argument("phase", choices=("active-empty", "offline-empty", "live"))
    parser.add_argument("--core", default=str(Path(__file__).with_name("uninstall_core.py")))
    parser.add_argument("--observer")
    parser.add_argument("--nested-dropins", action="store_true")
    parser.add_argument("--negative-fixtures", action="store_true")
    args = parser.parse_args()
    base = [sys.executable, args.core, "--project", args.project]
    if args.observer:
        base += ["--observer-binary", args.observer, "--observer-sha256", digest(args.observer)]
    before = snapshot(args.project)
    assert (before[1] == "0") == (args.phase == "offline-empty")
    cold_db = digest("/var/lib/boxweave/ledger.sqlite") if args.phase == "offline-empty" else None
    p, rc = read_plan(base, "--maintenance-plan", "retain")
    again, again_rc = read_plan(base, "--maintenance-plan", "retain")
    assert p == again and rc == again_rc, "stable preview changed"
    assert p["observation"]["coverage"] == "BOUNDED_SOURCES_OBSERVED"
    expected = "OFFLINE_RETIREMENT_REVIEW" if args.phase == "offline-empty" else "NORMAL_CLEANUP_REVIEW"
    assert p["route"] == expected, p["route"]
    hard = {b["code"] for b in p["blockers"] if b["severity"] == "HARD"}
    if args.nested_dropins:
        assert p["status"] == "BLOCKED" and rc == 2 and hard == {"EFFECTIVE_DROPIN_UNREVIEWED"}, p["blockers"]
    else:
        assert p["status"] == "REVIEW_ONLY" and rc == 0 and not hard, p["blockers"]
    l = p["observation"]["ledger"]
    assert len(l["history_sha256"]) == 64 and len(l["retired_boxes"]) == l["retired"]
    kinds = [x["kind"] for x in p["proposed_effects"]]
    assert kinds.count("STOP_DELETE_BOX") == len(l["boxes"])
    assert kinds.count("RETIRE_PEER_DIRECTION") == sum(x["state"] != "REVOKED" for x in l["peers"])
    assert kinds.count("RETIRE_FORWARD") == sum(x["state"] != "REVOKED" for x in l["forwards"])
    if args.phase == "live":
        assert len(l["boxes"]) >= 4 and len(l["retired_boxes"]) >= 1
        assert kinds.count("RETIRE_PEER_DIRECTION") >= 3
    else:
        assert not l["boxes"]
    confirm = ["--confirm-host-sha256", p["host_machine_id_sha256"],
               "--confirm-preview-sha256", p["preview_sha256"]]
    same, same_rc = read_plan(base, "--maintenance-recheck", "retain", confirm)
    assert same["recheck"] == ("MATCH_BLOCKED" if rc else "MATCH_REVIEW_ONLY") and same_rc == rc
    stale, stale_rc = read_plan(base, "--maintenance-recheck", "erase", confirm)
    assert stale["recheck"] == "STALE_PREVIEW" and stale_rc == 2
    assert stale["preview_sha256"] != p["preview_sha256"]
    assert any(x["kind"] == "REVIEW_ERASE_COMPLETED_ARCHIVES" for x in stale["proposed_effects"])
    wrong_host = ["--confirm-host-sha256", "0" * 64,
                  "--confirm-preview-sha256", p["preview_sha256"]]
    stale, stale_rc = read_plan(base, "--maintenance-recheck", "retain", wrong_host)
    assert stale["recheck"] == "STALE_PREVIEW" and stale_rc == 2
    assert before == snapshot(args.project), "preview changed installed files, native or controller PID"
    if cold_db:
        assert cold_db == digest("/var/lib/boxweave/ledger.sqlite")
    # No execution authority can be obtained from a preview digest.
    result = subprocess.run(base + ["--apply", "--confirm-preview-sha256", p["preview_sha256"]],
                            capture_output=True, text=True, timeout=10)
    assert result.returncode == 2
    print("PR188_H1B_" + args.phase.upper().replace("-", "_") + "=PASS", flush=True)
    if args.negative_fixtures:
        negative(args, base, p, confirm, before)


def negative(args, base, original, confirm, before):
    """Exact temporary fixtures, only on caller-selected disposable nested labs/CI."""
    assert os.geteuid() == 0
    # Root-trusted alternate file checks without touching installed binaries.
    observer = args.observer or "/usr/local/bin/boxweave"
    with tempfile.TemporaryDirectory(prefix="pr188-h1b-", dir="/root") as temp:
        bad = Path(temp) / "symlink"
        bad.symlink_to(observer)
        candidate = [sys.executable, args.core, "--project", args.project,
                     "--observer-binary", str(bad), "--observer-sha256", digest(observer)]
        refused, rc = read_plan(candidate, "--maintenance-plan", "retain")
        assert rc == 2 and "OBSERVER_UNAVAILABLE" in {x["code"] for x in refused["blockers"]}
        bad.unlink()
        # A hard-linked test executable is also refused. Use a disposable
        # copy, never increase the link count of the installed executable.
        import shutil
        copy = Path(temp) / "copy"
        shutil.copyfile(observer, copy)
        copy.chmod(0o755)
        os.link(copy, bad)
        refused, rc = read_plan(candidate, "--maintenance-plan", "retain")
        assert rc == 2 and "OBSERVER_UNAVAILABLE" in {x["code"] for x in refused["blockers"]}
    with tempfile.TemporaryDirectory(prefix="pr188-h1b-nonroot-") as temp:
        root = Path(temp)
        root.chmod(0o755)
        path = root / "uninstall_core.py"
        path.write_bytes(Path(args.core).read_bytes())
        path.chmod(0o644)
        r = subprocess.run(["runuser", "-u", "boxweave", "--", sys.executable, str(path),
                            "--maintenance-plan", "--project", args.project, "--retention", "retain"],
                           capture_output=True, text=True, timeout=10)
        assert r.returncode == 1 and "requires local root" in r.stderr
    print("PR188_H1B_UNTRUSTED_OBSERVER_NONROOT=PASS", flush=True)
    if args.phase == "offline-empty":
        marker = Path("/var/lib/boxweave/pr188-h1b-unknown-fixture")
        assert not os.path.lexists(marker)
        marker.symlink_to("/dev/null")
        try:
            changed, rc = read_plan(base, "--maintenance-recheck", "retain", confirm)
            assert rc == 2 and changed["recheck"] == "STALE_PREVIEW"
            assert "STATE_INVENTORY_UNSAFE" in {x["code"] for x in changed["blockers"]}
        finally:
            marker.unlink()
        print("PR188_H1B_UNSAFE_STATE_STALE=PASS", flush=True)
    if args.phase == "live":
        # A new, unattached fixture profile is not the owned default profile.
        # Its ACL/bridge references must be detected, not adopted.
        name = "pr188-h1b-foreign-consumer"
        project = args.project
        profiles = json.loads(run("incus", "query", f"/1.0/profiles?project={project}"))
        assert not any("/profiles/" + name in item for item in profiles)
        acl = run("incus", "profile", "device", "get", "default", "eth0", "security.acls",
                  "--project", project)
        bridge = "incusbr-" + project.removeprefix("user-")
        run("incus", "profile", "create", name, "--project", project)
        try:
            run("incus", "profile", "device", "add", name, "eth0", "nic",
                "network=" + bridge, "security.acls=" + acl, "--project", project)
            changed, rc = read_plan(base, "--maintenance-recheck", "retain", confirm)
            assert rc == 2 and changed["recheck"] == "STALE_PREVIEW"
            codes = {x["code"] for x in changed["blockers"]}
            assert "FOREIGN_OR_UNVERIFIED_CONSUMER" in codes, changed["blockers"]
        finally:
            run("incus", "profile", "delete", name, "--project", project)
        print("PR188_H1B_FOREIGN_CONSUMER_STALE=PASS", flush=True)
    assert before == snapshot(args.project)
    restored, rc = read_plan(base, "--maintenance-recheck", "retain", confirm)
    assert restored["preview_sha256"] == original["preview_sha256"], "fixture cleanup did not restore observed scope"
    assert restored["recheck"] in ("MATCH_BLOCKED", "MATCH_REVIEW_ONLY")
    print("PR188_H1B_NEGATIVE_NO_EFFECT=PASS", flush=True)


if __name__ == "__main__":
    main()
