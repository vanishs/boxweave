"""Host-safe BoxWeave CLI. Public diagnostics never execute application-owned code as root.

Only public installation metadata is read here. Application files, credentials and
system service changes retain their separate privilege boundaries.
"""
import argparse
import json
import os
from pathlib import Path
import pwd
import re
import stat
import subprocess
import sys
from urllib.error import HTTPError, URLError
from urllib.parse import quote
from urllib.request import build_opener, ProxyHandler

PUBLIC_STATE = Path("/usr/local/lib/boxweave/default/client-public.json")
IDENTITY = re.compile(r"[A-Za-z0-9][A-Za-z0-9_.-]{0,62}\Z")


def public_state():
    path = PUBLIC_STATE
    try:
        meta = path.lstat()
        if not stat.S_ISREG(meta.st_mode) or meta.st_uid != 0 or meta.st_mode & 0o022 or meta.st_size > 16384:
            raise ValueError("UNTRUSTED_PUBLIC_DESCRIPTOR")
        # Never follow links through a writable installation namespace.
        for parent in path.parents:
            info = parent.lstat()
            if not stat.S_ISDIR(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022:
                raise ValueError("UNTRUSTED_PUBLIC_DESCRIPTOR_DIRECTORY")
        d = json.loads(path.read_text(encoding="utf-8"))
        if (not isinstance(d, dict) or d.get("schema_version") != 1 or d.get("phase") != "ready"
                or d.get("application_user") != "boxweave"
                or d.get("app_prefix") != "/home/boxweave/boxweave"
                or not isinstance(d.get("version"), str)
                or not re.fullmatch(r"\d+\.\d+\.\d+", d["version"])
                or type(d.get("web_port")) is not int or not 1024 <= d["web_port"] <= 65535):
            raise ValueError("INVALID_PUBLIC_DESCRIPTOR")
        return d
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise ValueError("PUBLIC_DESCRIPTOR_NOT_AVAILABLE") from exc


def observe(argv, timeout=8, limit=16384, cwd=None):
    try:
        result = subprocess.run(argv, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                                stderr=subprocess.DEVNULL, timeout=timeout, check=False, cwd=cwd)
        if len(result.stdout) > limit:
            return None
        return result.stdout.decode("utf-8", "replace") if result.returncode == 0 else None
    except (OSError, subprocess.TimeoutExpired):
        return None


def web_observation(port):
    url = "http://127.0.0.1:" + str(port) + "/api/projects"
    opener = build_opener(ProxyHandler({}))
    try:
        with opener.open(url, timeout=3) as response:
            code = response.status
    except HTTPError as exc:
        code = exc.code
        exc.close()
    except (OSError, URLError):
        return {"reachable": False, "authenticated": False, "scope": "host_loopback"}
    return {"reachable": code in (200, 401, 403), "http_status": code,
            "authenticated": False, "scope": "host_loopback"}


def service_observation(name):
    output = observe(["systemctl", "show", name, "--property=LoadState,ActiveState,UnitFileState"], timeout=4)
    if output is None:
        return {"state": "unavailable"}
    entries = dict(line.split("=", 1) for line in output.splitlines() if "=" in line)
    return {key: entries.get(key) for key in ("LoadState", "ActiveState", "UnitFileState")}


def status(info):
    return {"state": "installed_public_descriptor", "version": info["version"],
            "application_user": info["application_user"],
            "web": web_observation(info["web_port"]),
            "web_on_server": "http://127.0.0.1:" + str(info["web_port"]),
            "external_web_access": "not_configured_by_boxweave",
            "mcp": "not_checked", "app_health": "not_checked"}


def mcp_probe(info):
    if os.geteuid() != 0:
        return {"state": "not_checked", "reason": "APP_USER_PROBE_REQUIRES_ADMIN",
                "next": "sudo boxweave doctor --mcp"}
    account = pwd.getpwnam(info["application_user"])
    prefix = Path(info["app_prefix"])
    result = observe(["runuser", "-u", account.pw_name, "--", "env",
                      "XDG_RUNTIME_DIR=/run/user/" + str(account.pw_uid),
                      str(prefix / "current/.venv/bin/python"), "-m", "bwmini.deploy_check",
                      "--config", str(prefix / "runtime.json"), "--mcp"], timeout=110, limit=8192,
                     cwd=str(prefix / "current"))
    if result is None:
        return {"state": "unavailable", "reason": "APP_USER_PROBE_FAILED"}
    try:
        outcome = json.loads(result)
        return {"state": "observed", "mcp": outcome.get("mcp"),
                "administrator_initialized": outcome.get("admin_initialized")}
    except (TypeError, ValueError):
        return {"state": "unavailable", "reason": "INVALID_PROBE_RESULT"}


def inspect(name, project, diagnose=False):
    if not IDENTITY.fullmatch(name) or not IDENTITY.fullmatch(project):
        return {"state": "error", "code": "INVALID_EXACT_TARGET"}, 2
    path = "/1.0/instances/" + quote(name, safe="") + "?project=" + quote(project, safe="")
    data = observe(["incus", "--force-local", "query", path], timeout=12, limit=262144)
    if data is None:
        return {"state": "unavailable", "code": "LOCAL_INCUS_ACCESS_UNAVAILABLE",
                "hint": "Incus administration is privileged. An authorized operator may use sudo boxweave box inspect NAME."}, 2
    try:
        record = json.loads(data)
        if record.get("name") != name or not isinstance(record.get("expanded_config"), dict):
            raise ValueError()
        config = record["expanded_config"]
        result = {"state": "observed", "name": name, "project": project,
                  "instance_state": record.get("status"), "type": record.get("type"),
                  "security": {key: config.get(key) for key in
                               ("security.nesting", "security.privileged", "boot.autostart")},
                  "confinement": "not_verified"}
        if diagnose:
            # Only a classified diagnosis is returned, never raw instance logs.
            from nesting import diagnostic
            logs = observe(["incus", "--force-local", "--project", project, "info", name, "--show-log"],
                           timeout=15, limit=16384)
            result["nesting_diagnostic"] = diagnostic(logs) if logs is not None else {"state": "unavailable"}
        return result, 0
    except (ValueError, TypeError, AttributeError, KeyError):
        return {"state": "error", "code": "INVALID_INSTANCE_METADATA"}, 2


def onboarding(info):
    version = info["version"]
    port = info["web_port"]
    print("BoxWeave " + version + " — installed host CLI")
    print("  Web on this Ubuntu host: http://127.0.0.1:" + str(port))
    print("  The Web listener is loopback-only, not a public LAN URL.")
    print("  From your own computer, make a reviewed SSH tunnel:")
    print("    ssh -N -L " + str(port) + ":127.0.0.1:" + str(port) +
          " YOUR_SSH_USER@YOUR_UBUNTU_HOST")
    print("  Then open http://127.0.0.1:" + str(port) + " on that computer.")
    print("  CLI: boxweave --help | boxweave status | boxweave doctor")
    print("  Agent setup: boxweave client (Windows Codex / WSL)")
    print("  First Web admin password location: /home/boxweave/boxweave/initial-admin-password")
    print("  Only an authorized administrator should read that private file.")
    print("  SSH account for enrollment is your authorized Ubuntu login, not the boxweave service account.")
    print("  Need an Incus-only inspection? boxweave box inspect NAME [--project PROJECT]")
    print("  No host changes or authenticated MCP test are performed by this guide.")


def main(argv=None):
    argv = list(sys.argv[1:] if argv is None else argv)
    if argv and argv[0] == "client":
        from client_setup import main as client_main
        return client_main(argv[1:])
    parser = argparse.ArgumentParser(description=__doc__,
                                     epilog="Run 'boxweave' with no arguments for next-step Web and Agent guidance.")
    sub = parser.add_subparsers(dest="action")
    sub.add_parser("version", help="Installed public release version")
    sub.add_parser("status", help="Public Web reachability and published installation metadata")
    doctor = sub.add_parser("doctor", help="Host/systemd observations (no repair)")
    doctor.add_argument("--mcp", action="store_true", help="Authenticated MCP read as app user; requires sudo")
    box = sub.add_parser("box", help="Exact-name Incus read-only inspection (Incus permission required)")
    box_sub = box.add_subparsers(dest="box_action")
    for action in ("inspect", "diagnose"):
        p = box_sub.add_parser(action, help="Inspect exact instance" if action == "inspect" else "Classify exact instance logs")
        p.add_argument("name")
        p.add_argument("--project", default="default")
    sub.add_parser("client", help="Generate a user-owned Windows Codex/WSL enrollment script (no sudo)")
    if not argv:
        try:
            onboarding(public_state())
        except ValueError:
            parser.print_help()
            print("Installation not ready or public descriptor unavailable. Run 'sudo boxweave status' only after reviewing the host installation.", file=sys.stderr)
            return 2
        return 0
    args = parser.parse_args(argv)
    if args.action is None:
        parser.print_help()
        return 0
    if args.action == "box" and args.box_action is None:
        box.print_help()
        return 0
    try:
        info = public_state()
    except ValueError as exc:
        print(json.dumps({"state": "unavailable", "code": str(exc)}), file=sys.stderr)
        return 2
    if args.action == "version":
        print("BoxWeave " + info["version"])
        return 0
    if args.action == "box":
        result, code = inspect(args.name, args.project, diagnose=args.box_action == "diagnose")
    elif args.action == "doctor":
        result = status(info)
        result["services"] = {"credential_timer": service_observation("boxweave-credential-"+info["application_user"]+".timer"),
                              "trust_timer": service_observation("boxweave-trust.timer")}
        result["user_manager"] = service_observation("user@" + str(pwd.getpwnam(info["application_user"]).pw_uid) + ".service")
        if args.mcp:
            result["mcp"] = mcp_probe(info)
        code = 0
    else:
        result = status(info)
        code = 0
    print(json.dumps(result, sort_keys=True))
    return code


if __name__ == "__main__":
    raise SystemExit(main())
