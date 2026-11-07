"""Optional root-owned fail-closed gateway policy for coordinated mutation mode.

This is a *capability gate*, not an authentication layer or per-Agent RBAC.
The trusted administrator must separately review/provision shared coordination
for app Web, SSH and HTTP. No MCP request can enable this mode.
"""
import json
from pathlib import Path
import stat

POLICY = Path("/var/lib/boxweave-default/public/coordination-policy.json")

# Only known read-only upstream methods. Never use an untrusted Provider's
# textual description or readOnlyHint to authorize a forwarded operation.
_UPSTREAM_READ = frozenset(("list_containers", "get_container", "get_system_info"))
_GATEWAY_READ = frozenset((
    "boxweave_info", "boxweave_diagnose", "boxweave_project_diagnose",
    "boxweave_diagnostic_run",
    "boxweave_box_read_file", "boxweave_box_list_directory",
    "boxweave_task_get", "boxweave_task_list", "boxweave_create_status",
))
_TRACKED_WRITE = frozenset(("boxweave_task_submit", "boxweave_box_lease", "boxweave_box_patch"))
_KNOWN_SPECIAL = frozenset(("boxweave_nesting",))
VISIBLE = _UPSTREAM_READ | _GATEWAY_READ | _TRACKED_WRITE | _KNOWN_SPECIAL


def enabled(path=POLICY):
    """Missing policy preserves legacy mode. Unsafe installed policy FAILS."""
    path=Path(path)
    try:
        p=path.parent.lstat()
        f=path.lstat()
    except FileNotFoundError:
        if path == POLICY:
            return False
        raise ValueError("COORDINATION_POLICY_UNAVAILABLE") from None
    if (not stat.S_ISDIR(p.st_mode) or p.st_uid != 0 or p.st_mode & 0o022
            or not stat.S_ISREG(f.st_mode) or f.st_uid != 0 or f.st_mode & 0o022
            or f.st_nlink != 1 or f.st_size > 1024):
        raise ValueError("COORDINATION_POLICY_UNSAFE")
    try:
        data=path.read_bytes()
        value=json.loads(data)
    except (OSError, UnicodeError, ValueError):
        raise ValueError("COORDINATION_POLICY_INVALID") from None
    later=path.lstat()
    if later.st_ino != f.st_ino or later.st_dev != f.st_dev:
        raise ValueError("COORDINATION_POLICY_UNSAFE")
    if (not isinstance(value,dict) or set(value)!={"version","enabled"}
            or type(value["version"]) is not int or value["version"]!=1
            or type(value["enabled"]) is not bool):
        raise ValueError("COORDINATION_POLICY_INVALID")
    return value["enabled"]


def check_call(name, arguments, *, rest_debug=False, coordinated_rest=False):
    """In protected mode, no direct or unclassified Provider mutations."""
    if not isinstance(name,str) or not isinstance(arguments,dict):
        raise ValueError("COORDINATION_TOOL_UNCLASSIFIED")
    if name == "create_container" and coordinated_rest is True:
        # The selected REST gateway must acquire the exact shared R-04 claim
        # and commit one durable dispatch intent before reaching the Provider.
        return
    if name == "debug_container" and rest_debug is True:
        # Strictly scoped to the selected REST adapter; legacy MCP must not
        # acquire new rights under protected mode.
        return
    if name == "boxweave_nesting":
        if arguments.get("action") == "plan":
            return
        raise ValueError("COORDINATION_TOOL_UNCLASSIFIED")
    if name in _UPSTREAM_READ | _GATEWAY_READ | _TRACKED_WRITE:
        return
    if name == "connect":
        raise ValueError("COORDINATION_USE_TRACKED_TASK")
    raise ValueError("COORDINATION_TOOL_UNCLASSIFIED")


def filter_catalog(message, *, rest_debug=False, coordinated_rest=False):
    """Hide forwarded mutators that this Gateway will refuse."""
    result=message.get("result")
    if not isinstance(result,dict) or not isinstance(result.get("tools"),list):
        raise ValueError("UPSTREAM_CATALOG_INVALID")
    allowed = (VISIBLE | ({"debug_container"} if rest_debug is True else set())
               | ({"create_container"} if coordinated_rest is True else set()))
    result["tools"]=[t for t in result["tools"]
                     if isinstance(t,dict) and t.get("name") in allowed]
    return message
