
"""Mandatory Containarium REST lifecycle control-plane for BoxWeave's app.

All migrated reads and writes use REST. Only unmigrated specialist operations
and fixed-command Box SSH/exec retain MCP. No lifecycle MCP fallback exists.
REST acknowledgements are not final Box-state or readiness evidence.
"""
import contextlib
import contextvars
import json
import os
from pathlib import Path
import re
import stat
import time
import urllib.error
import urllib.parse
import urllib.request

from . import diagnostics, resource_size, rest_wire, rest_routes
from .domain import Fault

_TARGET = re.compile(r"[a-z][a-z0-9-]{0,62}\Z")
_TRACE = contextvars.ContextVar("boxweave_rest_trace", default=None)
_RESPONSE_LIMIT = 524288


@contextlib.contextmanager
def trace_operation(trace, operation_id, actor="operator"):
    """Correlate a Provider HTTP call to its existing durable operation receipt."""
    state = _TRACE.set((trace, operation_id, actor))
    try:
        yield
    finally:
        _TRACE.reset(state)


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


def checked_logical(logical):
    if not isinstance(logical, str) or _TARGET.fullmatch(logical) is None:
        raise Fault("API_TARGET_INVALID", 422)
    if logical.startswith("containarium-core-"):
        raise Fault("TARGET_NOT_ALLOWED", 403)
    return logical


def checked_endpoint(url):
    try:
        p = urllib.parse.urlsplit(url)
        if (not isinstance(url, str) or p.scheme != "http" or
                p.hostname != "127.0.0.1" or p.username is not None or
                p.password is not None or p.netloc != "127.0.0.1:" + str(p.port) or
                not 1024 <= p.port <= 65535 or p.path or p.query or p.fragment):
            raise ValueError()
    except (TypeError, AttributeError, ValueError):
        raise Fault("PROVIDER_API_LOOPBACK_REQUIRED", 422) from None
    return url


def checked_token_file(path):
    if not isinstance(path, str):
        raise Fault("PROVIDER_API_TOKEN_FILE_REQUIRED", 422)
    item = Path(path)
    if not item.is_absolute() or ".." in item.parts or len(path) > 1024:
        raise Fault("PROVIDER_API_TOKEN_FILE_REQUIRED", 422)
    return item


class RESTClient:
    """Default-deny localhost API client with per-request credential rotation.

    No redirects, HTTP proxy, arbitrary endpoints, raw errors or automatic
    retries. The installed app account reads only its own private 0400 token.
    """
    def __init__(self, config):
        if not isinstance(config, dict) or set(config) != {"url", "token_file"}:
            raise Fault("PROVIDER_API_CONFIG_INVALID", 422)
        self.url = checked_endpoint(config["url"])
        self.token_file = checked_token_file(config["token_file"])
        self.store = None
        self.opener = urllib.request.build_opener(
            urllib.request.ProxyHandler({}), _NoRedirect())

    def _credential(self):
        flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
        try:
            fd = os.open(self.token_file, flags)
            with os.fdopen(fd, "rb") as f:
                meta = os.fstat(f.fileno())
                if (not stat.S_ISREG(meta.st_mode) or meta.st_uid != os.geteuid()
                        or meta.st_mode & 0o077 or meta.st_mode & 0o200
                        or meta.st_nlink != 1 or not 20 <= meta.st_size <= 8192):
                    raise Fault("PROVIDER_API_TOKEN_UNSAFE", 409)
                token = f.read(8193).strip()
            if not 20 <= len(token) <= 8192 or any(ch < 33 or ch > 126 for ch in token):
                raise Fault("PROVIDER_API_TOKEN_UNSAFE", 409)
            return token.decode("ascii")
        except (OSError, UnicodeError):
            raise Fault("PROVIDER_API_TOKEN_UNAVAILABLE", 409) from None

    def _event(self, method, route, status, code, elapsed, is_write):
        """Best-effort bounded metadata only: no URL, body, token or exception."""
        if self.store is None:
            return
        linkage = _TRACE.get()
        trace, op_id, actor = linkage if linkage else (diagnostics.new_trace_id(), None, "operator")
        diagnostics.emit(
            self.store, component="provider_api", phase="http_response",
            event_type=("acknowledged" if code == "API_RESPONSE_RECEIVED" and is_write
                        else "observed" if code == "API_RESPONSE_RECEIVED"
                        else "unknown" if is_write else "rejected"),
            outcome=("not_verified" if is_write and code == "API_RESPONSE_RECEIVED"
                     else "passed" if code == "API_RESPONSE_RECEIVED" else
                     "unknown" if is_write else "failed"),
            code=code, trace_id=trace, actor_scope=actor,
            operation_kind="environment" if op_id is not None else None,
            operation_id=op_id, elapsed_ms=elapsed)

    def request(self, method, path, body=None, timeout=30):
        # Transport and failure taxonomy are shared with the standalone Agent
        # REST gateway. App Fault remains the existing public error contract.
        try:
            return rest_wire.request(self, method, path, body, timeout)
        except rest_wire.WireFault as exc:
            # App-facing Provider failures use canonical gRPC status names
            # and their grpc-gateway HTTP mappings when classification is
            # authoritative. Ambiguous failures retain the old transport
            # classification; no application code inspects upstream text.
            if exc.grpc_code is not None:
                fault = Fault(exc.grpc_code.name, exc.http_status)
            elif (exc.upstream_http_status is not None
                  and exc.upstream_http_status >= 400):
                # A bare HTTP 400/404/409/500 does not identify an
                # authoritative gRPC code. Expose canonical UNKNOWN, while
                # retaining the original HTTP status as diagnostic metadata.
                fault = Fault("UNKNOWN", 500)
            else:
                # Local route/configuration/response-contract faults are
                # BoxWeave errors, not claims about Provider semantics.
                fault = Fault(exc.code, exc.status)
            fault.grpc_code = exc.grpc_code
            fault.http_status = exc.http_status
            fault.upstream_http_status = exc.upstream_http_status
            fault.status_origin = exc.status_origin
            raise fault from None

    def debug_container(self, logical):
        """Single exact-target REST read; preserve bounded upstream shape."""
        from . import debug_rest
        try:
            return debug_rest.validate(
                self.request("GET", debug_rest.route(logical), timeout=20))
        except debug_rest.DebugRESTError as exc:
            raise Fault(str(exc), 409) from None

    def inventory(self):
        result = self.request("GET", "/v1/containers", timeout=20)
        rows = result.get("containers", [])
        if (not isinstance(rows, list) or result.get("unreachableBackends")
                or result.get("nextCursor")
                or ("totalCount" in result and
                    (type(result["totalCount"]) is not int or result["totalCount"] != len(rows)))):
            raise Fault("INVENTORY_CONTRACT_UNSUPPORTED")
        names = []
        for row in rows:
            if not isinstance(row, dict) or not isinstance(row.get("username"), str):
                raise Fault("INVENTORY_CONTRACT_UNSUPPORTED")
            names.append(checked_logical(row["username"]))
        if len(names) != len(set(names)):
            raise Fault("INVENTORY_CONTRACT_UNSUPPORTED")
        return set(names)

    def container(self, logical):
        logical = checked_logical(logical)
        result = self.request("GET", "/v1/containers/" + logical, timeout=20)
        row = result.get("container")
        if not isinstance(row, dict) or row.get("username") != logical:
            raise Fault("TARGET_FORMAT_UNSUPPORTED")
        return row

    def mutate(self, action, logical, spec, timeout):
        """All app lifecycle mutations use the same exact route contract.

        Create is an asynchronously *accepted* REST write. The lifecycle worker
        must observe the original guest before reporting completion; a lost
        REST acknowledgement remains UNKNOWN and is never dispatched twice.
        """
        logical = checked_logical(logical)
        if action == "create":
            if "_nesting" in spec and type(spec["_nesting"]) is not bool:
                raise Fault("NESTING_BOOLEAN_REQUIRED", 422)
            args = {
                "username": logical,
                "cpu": spec["cpu"], "memory": spec["memory"], "disk": spec["disk"],
                "image": "images:ubuntu/24.04", "enable_podman": False,
                "ssh_keys": [spec["ssh_public_key"]],
            }
            if spec.get("_network_tenant") is not None:
                args["network_tenant"] = spec["_network_tenant"]
            try:
                method, route, body = rest_routes.route(
                    "create_container", args, async_create=True)
            except rest_routes.RouteFault as exc:
                raise Fault(str(exc), 422) from None
        elif action in ("start", "stop", "delete", "resize"):
            args = {"username": logical}
            if action == "resize":
                extra = spec.get("_resize")
                if not isinstance(extra, dict) or not extra or set(extra) - {
                        "cpu", "memory", "disk"}:
                    raise Fault("RESIZE_SPEC_INVALID", 422)
                if extra.get("disk") not in (None, ""):
                    row = self.container(logical)
                    try:
                        current = row["resources"]["disk"]
                    except (KeyError, TypeError):
                        raise Fault("RESOURCE_DISK_SIZE_UNVERIFIED", 422) from None
                    try:
                        resource_size.require_nonshrinking_disk(extra["disk"], current)
                    except ValueError as exc:
                        raise Fault(str(exc), 422) from None
                args.update(extra)
            elif action == "start":
                args["waitForReady"] = False
            elif action == "stop":
                args["force"] = False
            try:
                method, route, body = rest_routes.route(
                    action + "_container", args, async_create=False)
            except rest_routes.RouteFault as exc:
                raise Fault(str(exc), 422) from None
        else:
            raise Fault("ACTION_NOT_ALLOWED", 422)
        return self.request(method, route, body=body, timeout=timeout)
