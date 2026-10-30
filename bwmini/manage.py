"""Read-only BoxWeave management CLI; never repairs hosts or exports secrets."""
import argparse
import hashlib
import json
from pathlib import Path
import platform
import re
import subprocess
import sys
from urllib.parse import quote

IDENTITY = re.compile(r"[A-Za-z0-9][A-Za-z0-9_.-]{0,62}\Z")

def version_from(root):
    try:
        value = (Path(root) / "VERSION").read_text(encoding="utf-8").strip()
        if re.fullmatch(r"0\.[0-9]+\.[0-9]+", value):
            return value
    except OSError:
        pass
    return "unknown"

def command(argv, timeout=6):
    """Return a sanitized command observation; never use the shell or stderr."""
    try:
        p = subprocess.run(argv, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                           stderr=subprocess.DEVNULL, timeout=timeout, check=False)
        if len(p.stdout) > 8192:
            return {"state": "error", "reason": "output_too_large"}
        return {"state": "ok" if p.returncode == 0 else "error",
                "text": p.stdout.decode("utf-8", "replace") if p.returncode == 0 else "",
                "reason": None if p.returncode == 0 else "nonzero_exit"}
    except subprocess.TimeoutExpired:
        return {"state": "error", "reason": "timeout"}
    except (OSError, ValueError):
        return {"state": "error", "reason": "not_available"}

def component(binary, argv):
    result = command([binary, *argv])
    message = result.get("text", "").strip()
    match = re.search(r"(?<![A-Za-z0-9])v?([0-9]+\.[0-9]+\.[0-9]+(?:[-+][A-Za-z0-9.-]+)?)", message)
    return {"state": "observed" if result["state"] == "ok" and match else "unavailable",
            "version": match.group(1) if result["state"] == "ok" and match else None}

def inspect_box(name, project):
    if not all(IDENTITY.fullmatch(v) for v in (name, project)):
        raise ValueError("invalid_exact_target")
    uri = "/1.0/instances/" + quote(name, safe="") + "?project=" + quote(project, safe="")
    observation = command(["incus", "--force-local", "query", uri], timeout=10)
    if observation["state"] != "ok":
        return {"state": "unavailable", "reason": observation["reason"], "name": name, "project": project}, 1
    try:
        record = json.loads(observation["text"])
        if not isinstance(record, dict) or record.get("name") != name:
            raise ValueError("identity_mismatch")
        config = record.get("expanded_config", {})
        if not isinstance(config, dict):
            raise ValueError("invalid_config")
        keys = ("security.nesting", "security.privileged", "boot.autostart",
                "security.idmap.isolated", "security.syscalls.intercept.bpf")
        raw = json.dumps({"name": name, "project": project,
                          "expanded_config": config, "expanded_devices": record.get("expanded_devices", {})},
                         sort_keys=True, default=str).encode()
        return {"state": "observed", "name": name, "project": project,
                "instance_state": record.get("status"), "type": record.get("type"),
                "security": {key: config.get(key) for key in keys},
                "config_sha256": hashlib.sha256(raw).hexdigest(),
                "confinement": "not_verified", "note": "Snapshot only; no authorization or parent-host policy proof"}, 0
    except (ValueError, TypeError, AttributeError):
        return {"state": "error", "reason": "invalid_instance_metadata", "name": name, "project": project}, 1

def app_health(prefix, mcp=False):
    from .distribution import health
    return health(Path(prefix), mcp=mcp)

def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="action", required=True)
    v = sub.add_parser("version", help="Installed source version, optionally upstream client versions")
    v.add_argument("--components", action="store_true")
    for name in ("status", "doctor"):
        p = sub.add_parser(name, help="Inspect an app install (no repair)")
        p.add_argument("--prefix", default=str(Path.home() / "boxweave-v2"))
        if name == "doctor":
            p.add_argument("--mcp", action="store_true", help="Run a real read-only authenticated MCP probe")
    b = sub.add_parser("box", help="Read-only exact Incus instance inspection")
    bs = b.add_subparsers(dest="box_action", required=True)
    i = bs.add_parser("inspect")
    i.add_argument("name")
    i.add_argument("--project", default="default")
    n = bs.add_parser("diagnose")
    n.add_argument("name")
    n.add_argument("--project", default="default")
    sub.add_parser("client", help="Print selected client setup boundary")
    args = parser.parse_args(argv)
    exit_code = 0
    if args.action == "version":
        outcome = {"product": "BoxWeave", "version": version_from(Path(__file__).resolve().parents[1]),
                   "source": "local_files", "runtime_verified": False}
        if args.components:
            outcome["components"] = {"incus_cli": component("incus", ["--version"]),
                                     "containarium_cli": component("containarium", ["version"])}
    elif args.action == "box":
        try:
            outcome, exit_code = inspect_box(args.name, args.project)
            if args.box_action == "diagnose" and exit_code == 0:
                from .nesting import diagnostic
                log = command(["incus","--force-local","--project",args.project,"info",args.name,"--show-log"], timeout=15)
                outcome["nesting_diagnostic"] = diagnostic(log.get("text","")) if log["state"]=="ok" else {
                    "state":"unavailable","cause":"not_identified"}

        except ValueError:
            outcome, exit_code = {"state": "error", "reason": "invalid_exact_target"}, 2
    elif args.action == "client":
        outcome = {"state": "guidance_only", "profiles": ["codex-native", "codex-wsl", "zcode-native", "zcode-wsl"],
                   "guide": "clients/README.md", "registration": "not_attempted"}
    else:
        outcome = app_health(args.prefix, mcp=args.action == "doctor" and args.mcp)
        if args.action == "doctor":
            outcome["host_observations"] = {
                "platform": platform.system(), "arch": platform.machine(),
                "python": f"{sys.version_info.major}.{sys.version_info.minor}",
                "incus_cli": component("incus", ["--version"]),
                "containarium_cli": component("containarium", ["version"]),
                "effective_confinement": "not_verified"}
            from .nesting import environment
            outcome["host_observations"]["nesting"] = environment()
        exit_code = 0 if outcome.get("state") == "ready" else 1 if outcome.get("state") == "failed" else 2
    print(json.dumps(outcome, sort_keys=True))
    return exit_code

if __name__ == "__main__":
    raise SystemExit(main())
