"""Exact shared Containarium REST route/argument contract, no transport policy.

Used by the package-owned application and trusted standalone Agent gateway.
Deliberately DOES NOT implement a blind MCP-to-HTTP proxy.
"""
import re

try:
    from . import resource_size
except ImportError:
    import resource_size

class RouteFault(ValueError):
    pass

_NAMES = {
    "list_containers": "list",
    "get_container": "get",
    "debug_container": "debug",
    "create_container": "create",
    "start_container": "start",
    "stop_container": "stop",
    "delete_container": "delete",
    "resize_container": "resize",
}
_NAME = re.compile(r"[a-z][a-z0-9-]{0,62}\Z")
_ALLOWED_CREATE = frozenset((
    "username", "cpu", "memory", "disk", "image", "enable_podman",
    "ssh_keys", "disable_auto_sleep", "idle_stop_minutes",
    "gpu", "gpus", "monitoring", "encrypted", "tenant_id", "pool",
    "backend_id", "region", "network_tenant", "isolation", "os_type",
))
_BOOL = frozenset(("enable_podman", "disable_auto_sleep", "monitoring", "encrypted"))
_STR = frozenset(("cpu", "memory", "disk", "image", "gpu", "tenant_id",
                  "pool", "backend_id", "region", "network_tenant"))
_FIELD = {
    "ssh_keys": "sshKeys", "enable_podman": "enablePodman",
    "disable_auto_sleep": "disableAutoSleep",
    "idle_stop_minutes": "idleStopMinutes", "backend_id": "backendId",
    "tenant_id": "tenantId", "network_tenant": "networkTenant",
    "os_type": "osType",
}
_OSTYPES = {"ubuntu": "OS_TYPE_UBUNTU_2404", "rocky9": "OS_TYPE_ROCKY_9",
            "rhel9": "OS_TYPE_RHEL_9", "windows2022": "OS_TYPE_WINDOWS_2022"}
_ISOLATION = {"container": "ISOLATION_TYPE_CONTAINER", "vm": "ISOLATION_TYPE_VM"}


def logical_name(logical):
    if not isinstance(logical, str) or _NAME.fullmatch(logical) is None:
        raise RouteFault("API_TARGET_INVALID")
    if logical.startswith("containarium-core-"):
        raise RouteFault("TARGET_NOT_ALLOWED")
    return logical


def create_body(args, *, async_create=True):
    """Translate MCP's public schema with strict no-silent-option-loss."""
    if not isinstance(args, dict) or set(args) - _ALLOWED_CREATE:
        raise RouteFault("REST_CREATE_ARGUMENT_UNSUPPORTED")
    username = logical_name(args.get("username"))
    for key in _BOOL & args.keys():
        if type(args[key]) is not bool:
            raise RouteFault("REST_CREATE_ARGUMENT_INVALID")
    for key in _STR & args.keys():
        if not isinstance(args[key], str) or not args[key] or len(args[key]) > 512:
            raise RouteFault("REST_CREATE_ARGUMENT_INVALID")
    if "idle_stop_minutes" in args and (
            type(args["idle_stop_minutes"]) is not int or
            not 0 <= args["idle_stop_minutes"] <= 2147483647):
        raise RouteFault("REST_CREATE_ARGUMENT_INVALID")
    if ("idle_stop_minutes" in args and args["idle_stop_minutes"] > 0 and
            args.get("disable_auto_sleep", False)):
        raise RouteFault("REST_CREATE_ARGUMENT_INVALID")
    for key in ("ssh_keys", "gpus"):
        if key in args and (not isinstance(args[key], list)
                            or len(args[key]) > 16
                            or any(not isinstance(x, str) or not x or len(x) > 512
                                   for x in args[key])):
            raise RouteFault("REST_CREATE_ARGUMENT_INVALID")
    if "ssh_keys" not in args or not args["ssh_keys"]:
        raise RouteFault("MANAGED_SSH_KEY_REQUIRED")
    if "isolation" in args and args["isolation"] not in ("", *tuple(_ISOLATION)):
        raise RouteFault("REST_CREATE_ARGUMENT_UNSUPPORTED")
    if "os_type" in args and args["os_type"] not in ("", *tuple(_OSTYPES)):
        raise RouteFault("REST_CREATE_ARGUMENT_UNSUPPORTED")
    resources = {k: args.get(k, d) for k, d in (
        ("cpu", "4"), ("memory", "4GB"), ("disk", "50GB"))}
    body = {"username": username, "resources": resources,
            "sshKeys": args["ssh_keys"],
            "image": args.get("image", "images:ubuntu/24.04"),
            "enablePodman": args.get("enable_podman", True)}
    for field in _ALLOWED_CREATE - {"username", "ssh_keys", "image", "enable_podman",
                                   "cpu", "memory", "disk", "os_type", "isolation"}:
        if field in args and args[field] not in ("", [], None):
            body[_FIELD.get(field, field)] = args[field]
    if args.get("os_type"):
        body["osType"] = _OSTYPES[args["os_type"]]
    if args.get("isolation"):
        body["isolation"] = _ISOLATION[args["isolation"]]
    if async_create:
        body["async"] = True
    return body


def route(name, args, *, async_create=True):
    if name not in _NAMES:
        raise RouteFault("REST_TOOL_UNSUPPORTED")
    if not isinstance(args, dict):
        raise RouteFault("REST_ARGUMENTS_INVALID")
    action = _NAMES[name]
    if action == "list":
        if args:
            raise RouteFault("REST_LIST_OPTIONS_NOT_IMPLEMENTED")
        return "GET", "/v1/containers", None
    if action == "create":
        return "POST", "/v1/containers", create_body(args, async_create=async_create)
    logical = logical_name(args.get("username"))
    base = "/v1/containers/" + logical
    if action in ("get", "debug"):
        if set(args) != {"username"}:
            raise RouteFault("REST_ARGUMENTS_INVALID")
        return "GET", base + ("/debug" if action == "debug" else ""), None
    if action == "start":
        if set(args) - {"username", "waitForReady", "wait_for_ready"}:
            raise RouteFault("REST_ARGUMENTS_INVALID")
        ready = args.get("waitForReady", args.get("wait_for_ready", False))
        if type(ready) is not bool:
            raise RouteFault("REST_ARGUMENTS_INVALID")
        return "POST", base + "/start", {"waitForReady": ready}
    if action == "stop":
        if set(args) - {"username", "force"} or type(args.get("force", False)) is not bool:
            raise RouteFault("REST_ARGUMENTS_INVALID")
        return "POST", base + "/stop", {"force": args.get("force", False)}
    if action == "delete":
        if set(args) - {"username", "force"} or type(args.get("force", False)) is not bool:
            raise RouteFault("REST_ARGUMENTS_INVALID")
        if args.get("force", False):
            raise RouteFault("REST_FORCE_DELETE_UNSUPPORTED")
        return "DELETE", base, None
    if action == "resize":
        spec = {k: v for k, v in args.items() if k != "username"}
        if not spec or set(spec) - {"cpu", "memory", "disk"}:
            raise RouteFault("RESIZE_SPEC_INVALID")
        if any(not isinstance(v, str) or not v for v in spec.values()):
            raise RouteFault("RESIZE_SPEC_INVALID")
        return "PUT", base + "/resize", spec
    raise RouteFault("REST_TOOL_UNSUPPORTED")


def validate_read_result(tool, result, *, username=None):
    if not isinstance(result, dict):
        raise RouteFault("REST_RESULT_INVALID")
    if tool == "list_containers":
        rows = result.get("containers")
        if (not isinstance(rows, list) or result.get("nextCursor") or
                result.get("unreachableBackends") or
                ("totalCount" in result and
                 (type(result["totalCount"]) is not int or
                  result["totalCount"] != len(rows)))):
            raise RouteFault("REST_INVENTORY_PARTIAL")
        for row in rows:
            if not isinstance(row, dict) or not isinstance(row.get("username"), str):
                raise RouteFault("REST_INVENTORY_PARTIAL")
        return result
    if tool == "get_container":
        row = result.get("container")
        if (not isinstance(row, dict)
                or row.get("username") != logical_name(username)):
            raise RouteFault("REST_TARGET_CHANGED")
        return result
    if tool == "debug_container":
        try:
            from . import debug_rest
        except ImportError:
            import debug_rest
        return debug_rest.validate(result)
    raise RouteFault("REST_TOOL_UNSUPPORTED")


# Canonical BoxWeave-owned MCP descriptors for REST-migrated operations.
# Never trust an upstream MCP tool catalogue to describe or authorize them.
MIGRATED_TOOLS = frozenset(_NAMES)

def mcp_tool_catalog():
    def spec(fields, *, required=()):
        return {"type":"object","properties":fields,"required":list(required),
                "additionalProperties":False}
    logical={"type":"string","pattern":"^[a-z][a-z0-9-]{0,62}$"}
    string={"type":"string","minLength":1,"maxLength":512}
    boolean={"type":"boolean"}
    create={"username":logical}
    create.update({k:dict(string) for k in _STR})
    create.update({k:dict(boolean) for k in _BOOL})
    create["idle_stop_minutes"]={"type":"integer","minimum":0,"maximum":2147483647}
    create["ssh_keys"]={"type":"array","items":dict(string),"maxItems":16}
    create["gpus"]={"type":"array","items":dict(string),"maxItems":16}
    create["isolation"]={"type":"string","enum":["container","vm",""]}
    create["os_type"]={"type":"string","enum":["ubuntu","rocky9","rhel9","windows2022",""]}
    create["enable_nesting"]={"type":"boolean"}
    entries=(
        ("list_containers",spec({}),True),
        ("get_container",spec({"username":logical},required=("username",)),True),
        ("debug_container",spec({"username":logical},required=("username",)),True),
        ("create_container",spec(create,required=("username",)),False),
        ("start_container",spec({"username":logical,"waitForReady":boolean,
                                 "wait_for_ready":boolean},required=("username",)),False),
        ("stop_container",spec({"username":logical,"force":boolean},
                                required=("username",)),False),
        ("delete_container",spec({"username":logical,"force":boolean},
                                  required=("username",)),False),
        ("resize_container",spec({"username":logical,
                                  "cpu":string,"memory":string,"disk":string},
                                  required=("username",)),False),
    )
    return [{"name":name,"description":"BoxWeave managed REST "+name.replace("_"," "),
             "inputSchema":schema,
             "annotations":{"readOnlyHint":readonly,"destructiveHint":not readonly}}
            for name,schema,readonly in entries]
