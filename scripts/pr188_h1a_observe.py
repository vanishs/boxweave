#!/usr/bin/env python3
"""Read-only H1A installed observation checks, including offline identity."""
import argparse
import hashlib
import json
import subprocess
from pathlib import Path


def run(*argv):
    return subprocess.check_output(argv, text=True, timeout=45)


def normalized(value):
    if isinstance(value, dict):
        return {k: sorted(v) if k == "used_by" else normalized(v) for k, v in value.items()}
    if isinstance(value, list):
        return [normalized(v) for v in value]
    return value


def snapshot(project):
    instances = json.loads(run("incus", "query", f"/1.0/instances?recursion=1&project={project}"))
    instances.sort(key=lambda x: x["name"])
    native = [instances]
    for path in (f"/1.0/projects/{project}", f"/1.0/profiles/default?project={project}"):
        native.append(json.loads(run("incus", "query", path)))
    pid = run("systemctl", "show", "boxweave.service", "-p", "MainPID", "--value").strip()
    paths = ("/usr/local/bin/boxweave", "/var/lib/boxweave/install-manifest.json",
             "/etc/systemd/system/boxweave.service")
    hashes = [hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in paths]
    return normalized(native), pid, hashes


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("project")
    parser.add_argument("phase", choices=("active-empty", "offline-empty", "live"))
    parser.add_argument("--observer", default="/usr/local/bin/boxweave")
    parser.add_argument("--adapter", default="EXACT_RUNNING_OBSERVER_BINARY")
    args = parser.parse_args()
    before = snapshot(args.project)
    assert (before[1] == "0") == (args.phase == "offline-empty")
    command = [args.observer, "doctor", "-maintenance", "-project", args.project]
    previous = None
    for _ in range(3):
        d = json.loads(run(*command))
        assert d["version"] == 1 and d["scope"] == args.project
        assert d["coverage"] == "BOUNDED_SOURCES_OBSERVED" and not d["cross_source_atomic"]
        i, l, n = d["installation"], d["ledger"], d["native"]
        assert i["adapter"] == args.adapter and i["preimage"]["state"] == "OBSERVED"
        assert d["database"]["state"] == n["evidence"]["state"] == "OBSERVED"
        assert n["ownership_comparison"] == "OBSERVED" and n["unowned_or_mismatched"] == 0
        actual = {v["config"]["volatile.uuid"] for v in before[0][0]}
        assert actual == set(n["owned_matched"]) == {v["uuid"] for v in l["boxes"]}
        assert n["native_count"] == len(actual) and not n["owned_unverified"]
        assert all(v["create_proof"] and v["binding_state"] == "PINNED" for v in l["boxes"])
        assert not l["foreign_scope_rows"] and not l["nonterminal_or_unknown_originals"]
        assert l["nonterminal_recorded_claims"] == 0
        assert n["box_acl_references"] == n["forward_references"] == "OBSERVED"
        if args.phase == "live":
            assert i["mode"] == "SCOPED_V2" and len(actual) >= 4
            assert l["retired"] >= 1 and l["peer_capacity"]["active_directions"] >= 3
            assert l["recorded_claims"] > 0
            assert all(v["last_verified_original"] for v in l["peers"] if v["state"] == "CONFIRMED")
        else:
            assert not actual
        if i["mode"] == "SCOPED_V2":
            assert d["root_ledger_comparison"] == "MATCH"
            assert i["root_policy"]["kernel_guard"] == "OBSERVED"
        current = (l, n["evidence"]["sha256"])
        assert previous is None or previous == current, "stable inventory changed"
        previous = current
    for bad in ([args.observer, "doctor", "-maintenance", "-project", "default"],
                command + ["-approve-local"]):
        result = subprocess.run(bad, capture_output=True, timeout=45)
        assert result.returncode != 0
    if args.observer == "/usr/local/bin/boxweave":
        result = subprocess.run(["runuser", "-u", "boxweave", "--", *command],
                                capture_output=True, timeout=45)
        assert result.returncode != 0
    assert before == snapshot(args.project), "observer changed native configuration, controller or installed files"
    print("PR188_H1A_" + args.phase.upper().replace("-", "_") + "=PASS", flush=True)


if __name__ == "__main__":
    main()
