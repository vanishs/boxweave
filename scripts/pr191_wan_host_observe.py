#!/usr/bin/env python3
"""Read-only, aggregate forwarding diagnostics on a disposable N2 runner.

These observations do not attribute individual packets or establish WAN PASS.
Never modify host firewall, daemon, forwarding or Incus policy.
"""
import json
import os
import shlex
import subprocess
import sys

PHASES = {"prebootstrap", "before", "after"}
DOCKER_TARGETS = {"DOCKER-USER", "DOCKER-FORWARD", "DOCKER", "DOCKER-ISOLATION-STAGE-1"}


def summarize_forward(text):
    policy = None
    jumps = set()
    try:
        for line in text.splitlines():
            fields = shlex.split(line)
            if len(fields) == 3 and fields[:2] == ["-P", "FORWARD"]:
                if policy is not None or fields[2] not in {"DROP", "ACCEPT"}:
                    return "UNVERIFIED", "UNVERIFIED"
                policy = fields[2]
            elif len(fields) >= 2 and fields[:2] == ["-A", "FORWARD"]:
                for i, field in enumerate(fields[:-1]):
                    if field in {"-j", "--jump", "-g", "--goto"} and fields[i+1] in DOCKER_TARGETS:
                        jumps.add(fields[i+1])
            else:
                return "UNVERIFIED", "UNVERIFIED"
    except ValueError:
        return "UNVERIFIED", "UNVERIFIED"
    if policy is None:
        return "UNVERIFIED", "UNVERIFIED"
    return policy, "PRESENT" if jumps else "ABSENT"


def summarize_nft(text):
    try:
        rows = json.loads(text)["nftables"]
        if not isinstance(rows, list):
            raise ValueError()
        counts = {"ACCEPT": 0, "DROP": 0, "OTHER": 0}
        for row in rows:
            chain = row.get("chain", {})
            if chain.get("family") in {"ip", "inet"} and chain.get("hook") == "forward":
                policy = chain.get("policy", "").upper()
                counts[policy if policy in counts else "OTHER"] += 1
        return counts
    except (ValueError, KeyError, AttributeError, TypeError):
        return None


def observe(argv):
    try:
        proc = subprocess.run(argv, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                              text=True, timeout=8, check=False)
        if len(proc.stdout) > 1024*1024:
            return None, ""
        return proc.returncode, proc.stdout
    except (OSError, UnicodeError, subprocess.TimeoutExpired):
        return None, ""


def main(phase):
    if (os.geteuid() != 0 or os.environ.get("GITHUB_ACTIONS") != "true"
            or os.environ.get("RUNNER_ENVIRONMENT") != "github-hosted" or phase not in PHASES):
        raise SystemExit("Only the disposable hosted N2 read-only observer is allowed")
    rc, status = observe(["systemctl", "is-active", "docker.service"])
    status = status.strip()
    docker = status.upper() if ((rc == 0 and status == "active") or
        (rc in (3, 4) and status in {"inactive", "failed", "unknown"})) else "UNVERIFIED"
    rc, rules = observe(["iptables", "-w", "3", "-S", "FORWARD"])
    policy, jumps = summarize_forward(rules) if rc == 0 else ("UNVERIFIED", "UNVERIFIED")
    rc, chains = observe(["nft", "-j", "list", "chains"])
    counts = summarize_nft(chains) if rc == 0 else None
    nft = "UNVERIFIED" if counts is None else "%d/%d/%d" % (counts["ACCEPT"], counts["DROP"], counts["OTHER"])
    print("P1B_SCOPED_WAN_HOST_ENV", phase, "docker="+docker,
          "iptables="+policy, "docker_jump="+jumps, "nft_accept_drop_other="+nft, flush=True)


if __name__ == "__main__":
    if len(sys.argv) != 2:
        raise SystemExit("Expected exact observation phase")
    main(sys.argv[1])
