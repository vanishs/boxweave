"""Host-safe BoxWeave CLI. Public diagnostics never execute application-owned code as root.

Only public installation metadata is read here. Application files, credentials and
system service changes retain their separate privilege boundaries.
"""
import argparse
import importlib
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
PROVENANCE_STATE = Path("/usr/local/lib/boxweave/default/provenance.json")
IDENTITY = re.compile(r"[A-Za-z0-9][A-Za-z0-9_.-]{0,62}\Z")



def trusted_helper(name):
    """Resolve sibling code in isolated Python without importing user search paths.

    The installed directory and helper files must all be owned by root and
    unmodifiable by non-root callers. Development fixtures run only non-root.
    """
    if name not in ("client_setup", "nesting"):
        raise ValueError("UNKNOWN_HOST_HELPER")
    directory = Path(__file__).absolute().parent
    installed = Path("/usr/local/lib/boxweave/default")
    if directory != installed:
        if os.geteuid() == 0:
            raise ValueError("UNTRUSTED_HOST_CLI_LOCATION")
    else:
        for path in (directory, *directory.parents):
            info = path.lstat()
            if not stat.S_ISDIR(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022:
                raise ValueError("HOST_HELPER_DIRECTORY_UNTRUSTED")
        for filename in ("host_cli.py", "client_setup.py", "client_profiles.py", "online.py", "nesting.py"):
            info = (directory / filename).lstat()
            if (not stat.S_ISREG(info.st_mode) or info.st_uid != 0
                    or info.st_mode & 0o022 or info.st_nlink != 1):
                raise ValueError("HOST_HELPER_CODE_UNTRUSTED")
    # -I keeps the interpreter isolated. Explicitly allow this *verified*
    # root-owned directory only for the bundled helper's local imports.
    location = str(directory)
    sys.path.insert(0, location)
    try:
        return importlib.import_module(name)
    finally:
        sys.path.remove(location)


def published_descriptor(path, validator):
    """Shared trust gate for root-published 0644 JSON: owned, isolated, bounded."""
    try:
        meta = path.lstat()
        if not stat.S_ISREG(meta.st_mode) or meta.st_uid != 0 or meta.st_mode & 0o022 or meta.st_size > 16384:
            raise ValueError("UNTRUSTED_PUBLIC_DESCRIPTOR")
        for parent in path.parents:
            info = parent.lstat()
            if not stat.S_ISDIR(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022:
                raise ValueError("UNTRUSTED_PUBLIC_DESCRIPTOR_DIRECTORY")
        d = json.loads(path.read_text(encoding="utf-8"))
        validator(d)
        return d
    except (OSError, UnicodeError, json.JSONDecodeError, TypeError) as exc:
        raise ValueError("PUBLIC_DESCRIPTOR_NOT_AVAILABLE") from exc


def check_public_state(d):
    if (not isinstance(d, dict) or d.get("schema_version") != 1 or d.get("phase") != "ready"
            or d.get("application_user") != "boxweave"
            or d.get("app_prefix") != "/home/boxweave/boxweave"
            or not isinstance(d.get("version"), str)
            or not re.fullmatch(r"\d+\.\d+\.\d+", d["version"])
            or type(d.get("web_port")) is not int or not 1024 <= d["web_port"] <= 65535):
        raise ValueError("INVALID_PUBLIC_DESCRIPTOR")


def check_provenance(d):
    if (not isinstance(d, dict) or set(d) != {"schema_version", "version", "source", "installed_utc",
                                              "provider_mode", "host", "port", "web_port",
                                              "application_user", "managed_files", "phases"}
            or d["schema_version"] != 1 or not re.fullmatch(r"\d+\.\d+\.\d+", d["version"])
            or not isinstance(d["source"], str) or not re.fullmatch(r"[a-f0-9]{40}", d["source"])
            or not isinstance(d["installed_utc"], str) or len(d["installed_utc"]) > 64
            or d["provider_mode"] not in ("external", "standalone")
            or not isinstance(d["host"], str) or not re.fullmatch(r"[A-Za-z0-9.-]{1,253}", d["host"])
            or type(d["port"]) is not int or not 1 <= d["port"] <= 65535
            or type(d["web_port"]) is not int or not 1024 <= d["web_port"] <= 65535
            or d["application_user"] != "boxweave" or type(d["managed_files"]) is not int
            or not 0 <= d["managed_files"] <= 4096
            or not isinstance(d["phases"], list) or len(d["phases"]) > 64
            or any(not isinstance(p, dict) or set(p) != {"name", "started_utc"}
                   or not isinstance(p["name"], str) or not re.fullmatch(r"[a-z_]{1,64}", p["name"])
                   or not isinstance(p["started_utc"], str) or len(p["started_utc"]) > 64
                   for p in d["phases"])):
        raise ValueError("INVALID_PROVENANCE_DESCRIPTOR")


def public_state():
    return published_descriptor(PUBLIC_STATE, check_public_state)


def provenance_state():
    return published_descriptor(PROVENANCE_STATE, check_provenance)


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


DIAGNOSTIC_SCHEMA = 1


def check(cid, state, code, scope, required=True, detail=None):
    """One bounded diagnostic check: fixed states, fixed codes, fixed scope."""
    item = {"id": cid, "state": state, "code": code, "scope": scope, "required": required}
    if detail:
        item["detail"] = detail
    return item


def overall_state(checks):
    """failed beats incomplete; only every required check passed is passed."""
    required = [c for c in checks if c["required"]]
    if any(c["state"] == "failed" for c in required):
        return "failed"
    if required and all(c["state"] == "passed" for c in required):
        return "passed"
    return "incomplete"


def exit_code_for(overall):
    return {"passed": 0, "failed": 1}.get(overall, 2)


def classify_unit(observed, expect_enabled):
    """Definite failure only from an observed inactive/failed unit; never from a launch failure."""
    if not isinstance(observed, dict) or observed.get("state") == "unavailable" or not observed.get("ActiveState"):
        return "not_verified", "UNIT_STATE_UNAVAILABLE"
    if observed.get("ActiveState") == "failed":
        return "failed", "UNIT_OBSERVED_FAILED"
    if observed.get("ActiveState") == "inactive":
        return "failed", "UNIT_OBSERVED_INACTIVE"
    if observed.get("ActiveState") == "active" and (not expect_enabled or observed.get("UnitFileState") == "enabled"):
        return "passed", "UNIT_OBSERVED"
    return "not_verified", "UNIT_STATE_UNEXPECTED"


def run_bounded(argv, timeout, limit, cwd):
    """Bounded observation that also accepts the reader's typed exit-2 result output."""
    try:
        result = subprocess.run(argv, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                                stderr=subprocess.DEVNULL, timeout=timeout, check=False, cwd=cwd)
    except (OSError, subprocess.TimeoutExpired):
        return None
    if result.returncode not in (0, 2) or len(result.stdout) > limit:
        return None
    return result.stdout.decode("utf-8", "replace")


def app_health(info, mcp):
    """One app-user health transaction (bwmini.distribution health); never app code as root.

    Returns the parsed strict result dict, or a typed not_verified refusal.
    """
    if os.geteuid() != 0:
        return {"state": "not_verified", "code": "APP_HEALTH_REQUIRES_ADMIN",
                "next": "sudo boxweave doctor" + (" --mcp" if mcp else "")}
    account = pwd.getpwnam(info["application_user"])
    prefix = Path(info["app_prefix"])
    argv = ["runuser", "-u", account.pw_name, "--", "env",
            "XDG_RUNTIME_DIR=/run/user/" + str(account.pw_uid),
            str(prefix / "current/.venv/bin/python"), "-m", "bwmini.distribution", "health",
            "--prefix", str(prefix)] + (["--mcp"] if mcp else [])
    result = run_bounded(argv, timeout=140 if mcp else 40, limit=16384, cwd=str(prefix / "current"))
    if result is None:
        return {"state": "not_verified", "code": "APP_HEALTH_READER_UNAVAILABLE"}
    try:
        outcome = json.loads(result)
    except ValueError:
        return {"state": "not_verified", "code": "INVALID_APP_HEALTH_RESULT"}
    if (not isinstance(outcome, dict) or outcome.get("diagnostic_schema") != DIAGNOSTIC_SCHEMA
            or not isinstance(outcome.get("unit_states"), dict)):
        return {"state": "not_verified", "code": "INVALID_APP_HEALTH_RESULT"}
    installed = outcome.get("installed")
    if isinstance(installed, dict) and installed.get("version") != info["version"]:
        return {"state": "not_verified", "code": "APP_RELEASE_DESCRIPTOR_MISMATCH"}
    return outcome


def status(info):
    return {"state": "installed_public_descriptor", "version": info["version"],
            "application_user": info["application_user"],
            "web": web_observation(info["web_port"]),
            "web_on_server": "http://127.0.0.1:" + str(info["web_port"]),
            "external_web_access": "not_configured_by_boxweave",
            "mcp": "not_checked", "app_health": "not_checked"}


def doctor(info, mcp):
    """Explicit-scope readiness: every promised check must pass; gaps are never success."""
    web = web_observation(info["web_port"])
    uid = pwd.getpwnam(info["application_user"]).pw_uid
    cred = service_observation("boxweave-credential-" + info["application_user"] + ".timer")
    trust = service_observation("boxweave-trust.timer")
    manager = service_observation("user@" + str(uid) + ".service")
    checks = [check("public_descriptor", "passed", "PUBLIC_DESCRIPTOR_VALID", "host_public")]
    checks.append(check("web_loopback", "passed" if web.get("reachable") else "failed",
                        "HTTP_REACHABLE" if web.get("reachable") else "WEB_LOOPBACK_UNREACHABLE",
                        "host_loopback"))
    for cid, observed, expect_enabled in (("host_credential_timer", cred, True),
                                          ("host_trust_timer", trust, True),
                                          ("host_user_manager", manager, False)):
        state, code = classify_unit(observed, expect_enabled)
        checks.append(check(cid, state, code, "host_systemd"))
    health = app_health(info, mcp)
    verified = health.get("state") not in ("not_verified",) if isinstance(health, dict) else False
    if not verified:
        checks.append(check("app_release_health", "not_verified", health["code"], "app_user_systemd",
                            detail=health.get("next")))
        for cid in ("app_serve_unit", "app_worker_unit"):
            checks.append(check(cid, "not_verified", "APP_UNIT_VISIBILITY_UNAVAILABLE", "app_user_systemd"))
        mcp_field = {"state": "not_checked", "reason": "APP_USER_PROBE_REQUIRES_ADMIN",
                     "next": "sudo boxweave doctor --mcp"} if not mcp else \
                    {"state": "unavailable", "reason": "APP_USER_PROBE_FAILED"}
    else:
        units = health.get("unit_states", {})
        if health.get("state") == "ready" and units:
            checks.append(check("app_release_health", "passed", "APP_HEALTH_READY", "app_user_systemd"))
        elif health.get("state") == "failed":
            checks.append(check("app_release_health", "failed", "APP_HEALTH_FAILED", "app_user_systemd",
                                detail=health.get("code")))
        elif not units:
            checks.append(check("app_release_health", "not_verified", "APP_UNIT_LIST_EMPTY", "app_user_systemd"))
        else:
            checks.append(check("app_release_health", "not_verified", "APP_HEALTH_" +
                                str(health.get("state", "unexpected")).upper(), "app_user_systemd"))
        unit_codes = {"ready": ("passed", "APP_UNIT_READY"), "inactive": ("failed", "APP_UNIT_INACTIVE"),
                      "failed": ("failed", "APP_UNIT_FAILED"), "unavailable": ("not_verified", "APP_UNIT_STATE_UNAVAILABLE"),
                      "unknown": ("not_verified", "APP_UNIT_UNRECOGNIZED")}
        for cid, role in (("app_serve_unit", "serve"), ("app_worker_unit", "worker")):
            unit_state = units.get(role, "absent")
            state, code = unit_codes.get(unit_state, ("not_verified", "APP_UNIT_ABSENT"))
            checks.append(check(cid, state, code, "app_user_systemd"))
        observed_mcp = health.get("mcp")
        if mcp:
            mcp_field = {"state": "observed", "mcp": observed_mcp,
                         "administrator_initialized": health.get("administrator_initialized")}
            if isinstance(observed_mcp, dict) and observed_mcp.get("state") == "authenticated_read_passed":
                checks.append(check("authenticated_mcp", "passed", "MCP_READ_PASSED", "app_user_mcp"))
            elif health.get("code") == "MCP_READ_FAILED":
                checks.append(check("authenticated_mcp", "failed", "MCP_READ_FAILED", "app_user_mcp"))
            else:
                checks.append(check("authenticated_mcp", "not_verified", "AUTHENTICATED_MCP_UNVERIFIED", "app_user_mcp"))
            initialized = health.get("administrator_initialized")
            if initialized is True:
                checks.append(check("admin_initialized", "passed", "ADMIN_INITIALIZED", "app_user_mcp"))
            elif initialized is False:
                checks.append(check("admin_initialized", "failed", "ADMIN_SETUP_REQUIRED", "app_user_mcp"))
            else:
                checks.append(check("admin_initialized", "not_verified", "ADMIN_INITIALIZATION_UNVERIFIED", "app_user_mcp"))
        else:
            mcp_field = {"state": "not_checked", "reason": "NOT_REQUESTED"}
            checks.append(check("authenticated_mcp", "not_checked", "NOT_REQUESTED", "app_user_mcp", required=False))
    overall = overall_state(checks)
    return {"state": "installed_public_descriptor", "version": info["version"],
            "application_user": info["application_user"],
            "web": web, "web_on_server": "http://127.0.0.1:" + str(info["web_port"]),
            "external_web_access": "not_configured_by_boxweave",
            "mcp": mcp_field,
            "services": {"credential_timer": cred, "trust_timer": trust}, "user_manager": manager,
            "diagnostic_schema": DIAGNOSTIC_SCHEMA, "checks": checks, "overall": overall,
            "coverage": {"app_authenticated_health": "observed" if verified else
                         ("not_verified_requires_admin" if health.get("code") == "APP_HEALTH_REQUIRES_ADMIN"
                          else "unavailable"),
                         "authenticated_mcp": "observed" if mcp and verified else
                         ("not_requested" if not mcp else "unavailable"),
                         "workload_execute": "not_verified", "effective_nesting": "not_verified",
                         "agent_ui": "not_verified", "model_inference": "not_verified"}}


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
            diagnostic = trusted_helper("nesting").diagnostic
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
    print()
    print("==============================================================")
    print("  Configure your client")
    print("==============================================================")
    print()
    print("  boxweave client")
    print("  boxweave client --help  # More options")
    print()
    print("Run as your normal Linux SSH/sudo administrator (no sudo for client setup),")
    print("not the boxweave service account. Key enrollment needs admin approval.")
    print("Scripted/agent setup: pass --replace or --no-replace explicitly, or set BOXWEAVE_BATCH;")
    print("interactive prompts need a full terminal.")
    print("Unattended enrollment: generate with --no-enroll, approve the key on this host with")
    print("  sudo python3 /usr/local/lib/boxweave/default/online.py enroll --public-key KEY,")
    print("then rerun the setup with --already-enrolled.")
    print()
    print("Optional operations:")
    print("  Web on this Linux host: http://127.0.0.1:" + str(port))
    print("    Loopback-only; from your computer: ssh -N -L " + str(port) +
          ":127.0.0.1:" + str(port) + " YOUR_HOST_ADMIN@YOUR_LINUX_HOST")
    print("    Then open http://127.0.0.1:" + str(port) + " on your computer.")
    print("  First Web admin password (private): /home/boxweave/boxweave/initial-admin-password")
    print("  CLI commands: boxweave --help | boxweave status | boxweave doctor")
    print("  Incus inspection: boxweave box inspect NAME [--project PROJECT]")
    print("  This overview does not verify authenticated MCP access.")


def job_command(options):
    """Run one explicitly requested local command as the app user; never scheduled, never root."""
    budget = 60
    if options and options[0] == "--budget":
        if len(options) < 2 or not options[1].isdigit():
            print(json.dumps({"state": "error", "code": "JOB_BUDGET_INVALID"}), file=sys.stderr)
            return 2
        budget = int(options[1])
        options = options[2:]
    if len(options) < 2 or options[0] != "--":
        print("# Usage: boxweave job [--budget SECONDS] -- COMMAND [ARGS...]", file=sys.stderr)
        return 2
    try:
        info = public_state()
    except ValueError as exc:
        print(json.dumps({"state": "unavailable", "code": str(exc)}), file=sys.stderr)
        return 2
    if os.geteuid() != 0:
        print(json.dumps({"state": "error", "code": "JOB_REQUIRES_ROOT",
                          "next": "sudo boxweave job [--budget SECONDS] -- COMMAND [ARGS...]"}), file=sys.stderr)
        return 2
    budget = min(600, max(1, budget))
    account = pwd.getpwnam(info["application_user"])
    prefix = Path(info["app_prefix"])
    argv = ["runuser", "-u", account.pw_name, "--", "env",
            "XDG_RUNTIME_DIR=/run/user/" + str(account.pw_uid),
            str(prefix / "current/.venv/bin/python"), "-m", "bwmini", "job-run",
            "--config", str(prefix / "runtime.json"), "--argv", json.dumps(options[1:]),
            "--budget", str(budget)]
    result = observe(argv, timeout=budget + 60, limit=65536, cwd=str(prefix / "current"))
    if result is None:
        print(json.dumps({"state": "error", "code": "JOB_RUN_FAILED_INSPECT_RECEIPTS"}), file=sys.stderr)
        return 2
    print(result)
    try:
        return 0 if json.loads(result).get("state") == "succeeded" else 1
    except ValueError:
        return 2


def selftest_command(options):
    """Confirmed live smoke through the application: plan, then run with the plan hash."""
    try:
        info = public_state()
    except ValueError as exc:
        print(json.dumps({"state": "unavailable", "code": str(exc)}), file=sys.stderr)
        return 2
    if os.geteuid() != 0:
        print(json.dumps({"state": "error", "code": "SELFTEST_REQUIRES_ROOT",
                          "next": "sudo boxweave selftest plan | sudo boxweave selftest run --plan FILE --confirmation HEX"}), file=sys.stderr)
        return 2
    if not options or options[0] not in ("plan", "run"):
        print("# Usage: boxweave selftest plan [--budget SECONDS]\n"
              "#        boxweave selftest run --plan FILE --confirmation HEX [--budget SECONDS]", file=sys.stderr)
        return 2
    mode, options = options[0], options[1:]
    budget = 900
    passthrough = []
    while options:
        if options[0] == "--budget":
            if len(options) < 2 or not options[1].isdigit():
                print(json.dumps({"state": "error", "code": "SELFTEST_BUDGET_INVALID"}), file=sys.stderr)
                return 2
            budget = int(options[1])
            options = options[2:]
        else:
            passthrough.append(options[0])
            options = options[1:]
    if mode == "run" and ("--plan" not in passthrough or "--confirmation" not in passthrough):
        print("# Usage: boxweave selftest run --plan FILE --confirmation HEX [--budget SECONDS]", file=sys.stderr)
        return 2
    budget = min(1800, max(60, budget))
    account = pwd.getpwnam(info["application_user"])
    prefix = Path(info["app_prefix"])
    argv = ["runuser", "-u", account.pw_name, "--", "env",
            "XDG_RUNTIME_DIR=/run/user/" + str(account.pw_uid),
            str(prefix / "current/.venv/bin/python"), "-m", "bwmini", "selftest-" + mode,
            "--config", str(prefix / "runtime.json")] + passthrough + ["--budget", str(budget)]
    result = observe(argv, timeout=budget + 120, limit=65536, cwd=str(prefix / "current"))
    if result is None:
        print(json.dumps({"state": "error", "code": "SELFTEST_FAILED_INSPECT_STEPS"}), file=sys.stderr)
        return 2
    print(result)
    if mode == "plan":
        return 0
    try:
        return 0 if json.loads(result).get("state") == "passed" else 1
    except ValueError:
        return 2


def main(argv=None):
    argv = list(sys.argv[1:] if argv is None else argv)
    if argv and argv[0] == "client":
        try:
            return trusted_helper("client_setup").main(argv[1:])
        except (ImportError, OSError, ValueError) as exc:
            print(json.dumps({"state": "error", "code": "CLIENT_HELPER_UNAVAILABLE",
                              "reason": str(exc) if isinstance(exc, ValueError) else "IMPORT_FAILED"}),
                  file=sys.stderr)
            return 2
    if argv and argv[0] == "job":
        try:
            return job_command(argv[1:])
        except (OSError, ValueError, KeyError) as exc:
            print(json.dumps({"state": "error", "code": "JOB_UNAVAILABLE",
                              "reason": str(exc) if isinstance(exc, ValueError) else "JOB_CONTEXT_INVALID"}),
                  file=sys.stderr)
            return 2
    if argv and argv[0] == "selftest":
        try:
            return selftest_command(argv[1:])
        except (OSError, ValueError, KeyError) as exc:
            print(json.dumps({"state": "error", "code": "SELFTEST_UNAVAILABLE",
                              "reason": str(exc) if isinstance(exc, ValueError) else "SELFTEST_CONTEXT_INVALID"}),
                  file=sys.stderr)
            return 2
    parser = argparse.ArgumentParser(description=__doc__,
                                     epilog="Run 'boxweave' with no arguments for next-step Web and Agent guidance.")
    sub = parser.add_subparsers(dest="action")
    sub.add_parser("version", help="Installed public release version")
    sub.add_parser("status", help="Public Web reachability and published installation metadata")
    doctor_parser = sub.add_parser("doctor", help="Host/systemd and app readiness checks (no repair)")
    doctor_parser.add_argument("--mcp", action="store_true", help="Authenticated MCP read as app user; requires sudo")
    sub.add_parser("provenance", help="Published installation provenance (no secrets)")
    box = sub.add_parser("box", help="Exact-name Incus read-only inspection (Incus permission required)")
    box_sub = box.add_subparsers(dest="box_action")
    for action in ("inspect", "diagnose"):
        p = box_sub.add_parser(action, help="Inspect exact instance" if action == "inspect" else "Classify exact instance logs")
        p.add_argument("name")
        p.add_argument("--project", default="default")
    sub.add_parser("client", help="Generate reviewed MCP client setup for Windows, Linux or macOS (no sudo)")
    if not argv:
        try:
            onboarding(public_state())
        except ValueError:
            parser.print_help()
            print("Installation not ready or public descriptor unavailable. Run 'sudo boxweave status' only after reviewing the host installation.", file=sys.stderr)
            return 2
        return 0
    args = parser.parse_args(argv)
    if args.action == "provenance":
        try:
            print(json.dumps(provenance_state(), sort_keys=True))
            return 0
        except ValueError as exc:
            print(json.dumps({"state": "unavailable", "code": str(exc)}), file=sys.stderr)
            return 2
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
        result = doctor(info, mcp=args.mcp)
        code = exit_code_for(result["overall"])
    else:
        result = status(info)
        code = 0
    print(json.dumps(result, sort_keys=True))
    return code


if __name__ == "__main__":
    raise SystemExit(main())
