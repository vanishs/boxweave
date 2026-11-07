"""Mandatory REST boundary for the eight migrated Agent Gateway tools.

The reviewed application configuration is the sole credential/Provider source.
No operator switch, request field or upstream MCP fallback can select a second
transport for migrated lifecycle methods. Unmigrated specialist tools retain
the separately protected MCP launcher.
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

POLICY = Path("/home/boxweave/boxweave/config.json")
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
    """Load the one installed application REST binding; never return MCP mode.

    Missing, stale, unsafe or uncoordinated configuration fails closed. In
    particular, old lifecycle-rest.json feature switches are ignored.
    """
    path = Path(path)
    try:
        info=path.lstat()
        parent=path.parent.lstat()
        if (not stat.S_ISREG(info.st_mode) or info.st_nlink != 1
                or info.st_uid != os.geteuid() or info.st_mode & 0o077
                or not 10 <= info.st_size <= 65536
                or not stat.S_ISDIR(parent.st_mode)
                or parent.st_uid != os.geteuid() or parent.st_mode & 0o022):
            raise GatewayRESTFault("REST_APP_CONFIG_UNVERIFIED")
        with path.open("rb") as stream:
            app=json.loads(stream.read(65537), object_pairs_hook=_unique)
        again=path.lstat()
        if (info.st_dev,info.st_ino)!=(again.st_dev,again.st_ino):
            raise GatewayRESTFault("REST_APP_CONFIG_UNVERIFIED")
    except (OSError, UnicodeError, ValueError) as exc:
        if isinstance(exc,GatewayRESTFault):
            raise
        raise GatewayRESTFault("REST_APP_CONFIG_UNVERIFIED") from None
    if (not isinstance(app,dict) or app.get("mode")!="mcp"
            or app.get("coordination")!={"enabled":True}
            or not isinstance(app.get("database"),str)
            or not isinstance(app.get("provider_api"),dict)):
        raise GatewayRESTFault("REST_APP_COORDINATION_NOT_ALIGNED")
    try:
        api=app["provider_api"]
        rest_wire.Client(api)
        from . import gateway_create
    except ImportError:
        import gateway_create
        api=app["provider_api"]
        rest_wire.Client(api)
    except rest_wire.WireFault as exc:
        raise GatewayRESTFault(exc.code) from None
    gateway_create.checked_app_store(app["database"])
    return {"mode":"coordinated","url":api["url"],
            "token_file":api["token_file"],"state_db":app["database"],
            "app_config":str(path)}


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
