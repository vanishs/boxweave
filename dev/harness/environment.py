"""Read-only environment inspection; never install dependencies."""
import os
import shutil
import subprocess
import sys

from .common import observe

REQUIREMENTS = ("git", "go", "make", "base", "gofmt", "incus", "systemctl")


def register(parser):
    parser.add_argument("--require", choices=REQUIREMENTS,
                        action="append", default=[])


def run(args, root):
    tools = {}
    for name, argv in (
        ("git", ["git", "--version"]),
        ("go", ["go", "version"]),
        ("make", ["make", "--version"]),
    ):
        if not shutil.which(name):
            tools[name] = {"status": "MISSING"}
            continue
        try:
            result = subprocess.run(argv, cwd=root, capture_output=True, text=True,
                                    encoding="utf-8", errors="replace", timeout=10, check=False,
                                    env=dict(os.environ, GOTOOLCHAIN="local"))
            tools[name] = {
                "status": "AVAILABLE" if result.returncode == 0 else "UNKNOWN",
                "version": result.stdout.splitlines()[0] if result.stdout else None,
            }
        except (OSError, subprocess.TimeoutExpired):
            tools[name] = {"status": "UNKNOWN"}
    for name in ("gofmt", "incus", "systemctl"):
        tools[name] = {"status": "AVAILABLE" if shutil.which(name) else "MISSING", "version": None}
    source = observe(root)
    required = set(args.require) | {"git"}
    blockers = sorted(name for name in required if name != "base"
                      and tools[name]["status"] != "AVAILABLE")
    if not source["commit"] or source["worktree"] == "unknown":
        blockers.append("git-worktree")
    if "base" in required and not source["base_commit"]:
        blockers.append("origin/v1")
    return {
        "status": "BLOCKED" if blockers else "OBSERVED",
        "python": sys.version.split()[0], "source": source, "tools": tools,
        "blockers": blockers, "native_capability": "UNKNOWN",
        "note": "Tool presence is not native readiness or acceptance.",
    }, 1 if blockers else 0
