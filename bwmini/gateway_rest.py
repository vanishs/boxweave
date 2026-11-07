"""Root-published opt-in REST selector for the standalone Agent gateway.

A same-user Token is never exported over MCP. No incoming request can pick URL,
HTTP verb, credential path or authorization mode. This does not grant new MCP
permissions: Gateway's allowlist and protected R-04 checks run first.
"""
import json
import os
from pathlib import Path
import stat

try:
    from . import rest_wire, rest_routes
except ImportError:
    import rest_wire
    import rest_routes

POLICY = Path("/var/lib/boxweave-default/public/lifecycle-rest.json")
ROUTED = frozenset(rest_routes._NAMES)
WRITES = frozenset(("create_container", "start_container", "stop_container",
                    "delete_container", "resize_container"))


class GatewayRESTFault(ValueError):
    pass


def _unique(pairs):
    obj = {}
    for key, value in pairs:
        if key in obj:
            raise ValueError("duplicate")
        obj[key] = value
    return obj


def policy(path=POLICY):
    """Only exact administrator-owned immutable policy selects REST.

    Missing default policy retains legacy behavior; an *unsafe* present
    policy fails closed. This is not a publicly callable configurator.
    """
    path = Path(path)
    try:
        before = path.lstat()
    except FileNotFoundError:
        if path == POLICY:
            return None
        raise GatewayRESTFault("REST_POLICY_UNAVAILABLE") from None
    try:
        parent = path.parent.lstat()
        if (not stat.S_ISDIR(parent.st_mode) or parent.st_uid != 0
                or parent.st_mode & 0o022
                or not stat.S_ISREG(before.st_mode) or before.st_uid != 0
                or before.st_mode & 0o022 or before.st_nlink != 1
                or not 10 <= before.st_size <= 2048):
            raise GatewayRESTFault("REST_POLICY_UNSAFE")
        with path.open("rb") as stream:
            raw = stream.read(2049)
        if len(raw) > 2048:
            raise GatewayRESTFault("REST_POLICY_UNSAFE")
        now = path.lstat()
        if (before.st_dev, before.st_ino) != (now.st_dev, now.st_ino):
            raise GatewayRESTFault("REST_POLICY_UNSAFE")
        value = json.loads(raw, object_pairs_hook=_unique)
    except (OSError, UnicodeError, ValueError) as exc:
        if isinstance(exc, GatewayRESTFault):
            raise
        raise GatewayRESTFault("REST_POLICY_INVALID") from None
    if (not isinstance(value, dict)
            or set(value) != ({"version", "enabled", "mode", "url", "token_file"}
                                 | ({"state_db", "app_config"} if value.get("mode") == "coordinated" else set()))
            or type(value["version"]) is not int or value["version"] != 1
            or type(value["enabled"]) is not bool
            or value["mode"] not in ("read", "coordinated")):
        raise GatewayRESTFault("REST_POLICY_INVALID")
    try:
        rest_wire.Client({"url": value["url"], "token_file": value["token_file"]})
    except rest_wire.WireFault as exc:
        raise GatewayRESTFault(exc.code) from None
    if value["enabled"] and value["mode"] == "coordinated":
        try:
            from . import gateway_create
        except ImportError:
            import gateway_create
        # The app store is the only durable receipt owner. Confirm Web/CLI
        # explicitly opted into the same R-04 coordination core and Provider.
        gateway_create.checked_app_store(value["state_db"])
        app = value["app_config"]
        if not isinstance(app, str) or not app.startswith(
                "/home/boxweave/boxweave/") or ".." in Path(app).parts:
            raise GatewayRESTFault("REST_APP_CONFIG_INVALID")
        path = Path(app)
        try:
            info = path.lstat()
            parent = path.parent.lstat()
            if (not stat.S_ISREG(info.st_mode) or info.st_nlink != 1
                    or info.st_uid != os.geteuid() or info.st_mode & 0o077
                    or not 10 <= info.st_size <= 65536
                    or not stat.S_ISDIR(parent.st_mode)
                    or parent.st_uid != os.geteuid()
                    or parent.st_mode & 0o022):
                raise ValueError()
            with path.open("rb") as stream:
                app_config = json.loads(stream.read(65537), object_pairs_hook=_unique)
            again = path.lstat()
            if (info.st_dev,info.st_ino) != (again.st_dev,again.st_ino):
                raise ValueError()
        except (ValueError, OSError, UnicodeError):
            raise GatewayRESTFault("REST_APP_CONFIG_UNVERIFIED") from None
        if (not isinstance(app_config,dict) or
                app_config.get("mode") != "mcp" or
                app_config.get("coordination") != {"enabled":True} or
                app_config.get("database") != value["state_db"] or
                app_config.get("provider_api") != {
                    "url":value["url"], "token_file":value["token_file"]}):
            raise GatewayRESTFault("REST_APP_COORDINATION_NOT_ALIGNED")
    return value if value["enabled"] else None


def selected(name, configuration):
    return configuration is not None and name in ROUTED


def execute_read(configuration, tool, args):
    """The three REST reads are safe after caller authorization.

    The authenticated REST token is reloaded on every request. All upstream
    data validation happens before it crosses the MCP response boundary.
    """
    if not isinstance(configuration, dict) or tool not in (
            "list_containers", "get_container", "debug_container"):
        raise GatewayRESTFault("REST_READ_NOT_ALLOWED")
    try:
        method, route, body = rest_routes.route(tool, args)
        result = rest_wire.Client({k: configuration[k] for k in ("url", "token_file")}).request(method, route, body, timeout=20)
        return rest_routes.validate_read_result(
            tool, result, username=args.get("username") if isinstance(args, dict) else None)
    except rest_routes.RouteFault as exc:
        raise GatewayRESTFault(str(exc)) from None
    except rest_wire.WireFault as exc:
        # Expose standard Provider semantics, never parse the raw HTTP body.
        if exc.grpc_code is not None:
            code = exc.grpc_code.name
        elif (exc.upstream_http_status is not None
              and exc.upstream_http_status >= 400):
            code = "UNKNOWN"
        else:
            code = exc.code
        raise GatewayRESTFault(code) from None


def plan_write(configuration, tool, args):
    """Validate exact action + payload BEFORE any durable dispatch intent.

    A plan is not permission to execute. The operation coordinator must
    reserve the specific target/name first. No route override or fallback.
    """
    if (not isinstance(configuration, dict) or
            configuration.get("mode") != "coordinated" or tool not in WRITES):
        raise GatewayRESTFault("REST_WRITE_NOT_AUTHORIZED")
    try:
        method, route, body = rest_routes.route(tool, args)
        return {"method": method, "route": route, "body": body}
    except rest_routes.RouteFault as exc:
        raise GatewayRESTFault(str(exc)) from None
