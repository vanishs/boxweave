"""Exact-instance nesting configuration and honest parent-environment observations.

The privileged stdin-only entry point changes one Incus boolean. It never starts,
stops, deletes, installs packages, edits profiles or changes parent-host policy.
"""
import datetime, hashlib, json, os, re, socket, subprocess, sys
from pathlib import Path

INCUS = "/usr/bin/incus"
STATE = Path("/var/lib/boxweave-default/state.json")
HELPER = "/usr/local/lib/boxweave/default/nesting.py"
SUDO = ["/usr/bin/sudo", "-n", "/usr/bin/python3", "-I", HELPER]
RESTART_NOTE = ("Restart the configured outer box before relying on the changed nesting capability; "
                "then verify a child workload through authenticated MCP. No restart is performed.")

def environment():
    try:
        p = subprocess.run(["systemd-detect-virt", "--container"], capture_output=True, text=True, timeout=5)
        kind = p.stdout.strip() if p.returncode == 0 else "none" if p.returncode == 1 else "unknown"
    except (OSError, subprocess.SubprocessError):
        kind = "unknown"
    return {"container": kind, "parent_nesting": "not_verified",
            "effective_nested_execution": "not_verified",
            "parent_action": "Check security.nesting on the parent Incus instance." if kind == "lxc" else None,
            "restart_guidance": RESTART_NOTE if kind == "lxc" else None}

def run(argv):
    p = subprocess.run(argv, stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=20)
    if p.returncode or len(p.stdout) > 262144:
        raise ValueError("LOCAL_INCUS_OBSERVATION_OR_CHANGE_FAILED")
    return p.stdout

def observe(logical):
    if (not isinstance(logical, str) or not re.fullmatch(r"[a-z][a-z0-9-]{0,52}", logical)
            or logical.startswith("containarium-core-")):
        raise ValueError("EXACT_MANAGED_TARGET_REQUIRED")
    name = logical + "-container"
    item = json.loads(run([INCUS, "--force-local", "--project", "default", "query", "/1.0/instances/" + name]))
    config = item.get("expanded_config", {})
    own = item.get("config", {})
    if (item.get("name") != name or item.get("type") != "container"
            or own.get("user.containarium.label.os-type") not in ("ubuntu_2404", "ubuntu")
            or not own.get("volatile.uuid")):
        raise ValueError("MANAGED_INSTANCE_IDENTITY_REQUIRED")
    created = int(datetime.datetime.fromisoformat(item["created_at"].replace("Z", "+00:00")).timestamp())
    value = config.get("security.nesting", "false")
    if value not in ("true", "false"):
        raise ValueError("NESTING_CONFIG_INVALID")
    # Include all effective config/devices in the plan fingerprint, never return them.
    fingerprint = hashlib.sha256(json.dumps({
        "name": name, "created": created, "config": config, "devices": item.get("expanded_devices", {})
    }, sort_keys=True).encode()).hexdigest()
    return {"logical": logical, "name": name, "project": "default", "created": str(created),
            "uuid": own["volatile.uuid"], "config_sha256": fingerprint,
            "nesting": value == "true", "instance_state": item.get("status"),
            "effective_nested_execution": "not_verified"}

def identity(value):
    return {k: value[k] for k in ("logical", "name", "project", "created", "uuid", "config_sha256", "nesting")}

def handle(request):
    if not isinstance(request, dict):
        raise ValueError("REQUEST_OBJECT_REQUIRED")
    action = request.get("action")
    expected_keys = {"action", "logical"} | ({"nesting"} if action == "plan" else
                                           {"nesting", "expected", "confirm"} if action == "apply" else set())
    if set(request) != expected_keys or action not in ("inspect", "plan", "apply"):
        raise ValueError("NESTING_REQUEST_INVALID")
    before = observe(request["logical"])
    if action == "inspect":
        return {"target": before, "restart_guidance": RESTART_NOTE}
    wanted = request["nesting"]
    if type(wanted) is not bool:
        raise ValueError("NESTING_BOOLEAN_REQUIRED")
    plan = {"target": before, "nesting": wanted, "podman": "unchanged",
            "restart_performed": False, "restart_guidance": RESTART_NOTE}
    if action == "plan":
        return plan | {"expected": identity(before)}
    if request["confirm"] != before["name"] or request["expected"] != identity(before):
        raise ValueError("STALE_PLAN_OR_EXACT_CONFIRMATION_REQUIRED")
    changed = before["nesting"] != wanted
    if changed:
        run([INCUS, "--force-local", "--project", "default", "config", "set",
             before["name"], "security.nesting=" + str(wanted).lower()])
    after = observe(before["logical"])
    if after["uuid"] != before["uuid"] or after["created"] != before["created"] or after["nesting"] != wanted:
        raise ValueError("CHANGE_OUTCOME_UNVERIFIED_INSPECT_ORIGINAL_INSTANCE")
    return plan | {"target": after, "changed": changed, "state": "configured",
                   "effective_nested_execution": "not_verified",
                   "restart_required_for_verified_transition": changed}

def privileged_main():
    if os.geteuid() != 0:
        raise ValueError("ROOT_REQUIRED")
    info = STATE.lstat()
    if STATE.is_symlink() or info.st_uid != 0 or info.st_mode & 0o077:
        raise ValueError("OWNED_PROVIDER_REQUIRED")
    state = json.loads(STATE.read_text())
    if state.get("hostname") != socket.gethostname() or not state.get("managed_files"):
        raise ValueError("OWNED_PROVIDER_REQUIRED")
    raw = sys.stdin.buffer.read(8193)
    if len(raw) > 8192:
        raise ValueError("REQUEST_LIMIT")
    print(json.dumps(handle(json.loads(raw))), flush=True)

def invoke(request):
    p = subprocess.run(SUDO, input=json.dumps(request), capture_output=True, text=True, timeout=50)
    if p.returncode or len(p.stdout) > 65536:
        raise ValueError("NESTING_HELPER_FAILED_INSPECT_ORIGINAL_INSTANCE")
    return json.loads(p.stdout)

def diagnostic(log_text):
    blocked = 'Failed to mount "proc"' in log_text and "Operation not permitted" in log_text
    return {"nested_proc_mount_denied": blocked, "parent_nesting": "not_verified",
            "cause": "parent_nesting_or_other_mount_confinement" if blocked else "not_identified",
            "guidance": ("On the parent host, inspect the exact outer instance security.nesting setting. "
                         + RESTART_NOTE) if blocked else None}

if __name__ == "__main__":
    try:
        privileged_main()
    except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError):
        print(json.dumps({"state": "failed", "code": "NESTING_REQUEST_REJECTED_OR_OUTCOME_UNVERIFIED"}), flush=True)
        raise SystemExit(2)
