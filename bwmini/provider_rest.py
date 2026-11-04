
"""Bounded, opt-in Containarium REST control-plane for BoxWeave's app Provider.

Only the app's own basic lifecycle is migrated. The independent Agent-facing
platform MCP and fixed-command Box SSH/exec remain separate, observable paths.
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

from . import diagnostics
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
        if method not in ("GET", "POST", "PUT", "DELETE"):
            raise Fault("PROVIDER_API_METHOD_DENIED", 422)
        if (not isinstance(path, str) or not path.startswith("/v1/") or
                "//" in path or ".." in path or "?" in path or
                not 1 <= len(path) <= 192):
            raise Fault("PROVIDER_API_ROUTE_DENIED", 422)
        if type(timeout) not in (int, float) or not 0 < timeout <= 1800:
            raise Fault("DEADLINE_EXCEEDED")
        raw = None
        if body is not None:
            raw = json.dumps(body, ensure_ascii=True, separators=(",", ":")).encode()
            if len(raw) > 65536:
                raise Fault("PROVIDER_API_REQUEST_TOO_LARGE", 422)
        # A local credential failure is proven pre-dispatch. Never emit its bytes.
        credential = self._credential()
        request = urllib.request.Request(
            self.url + path, data=raw, method=method,
            headers={"Authorization": "Bearer " + credential,
                     "Content-Type": "application/json", "Accept": "application/json"})
        del credential
        started = time.monotonic()
        is_write = method != "GET"
        code = "API_OUTCOME_UNKNOWN" if is_write else "API_READ_UNAVAILABLE"
        try:
            with self.opener.open(request, timeout=timeout) as response:
                if not 200 <= response.status < 300:
                    raise Fault("PROVIDER_API_UNEXPECTED_STATUS")
                payload = response.read(_RESPONSE_LIMIT + 1)
                if len(payload) > _RESPONSE_LIMIT:
                    raise Fault("PROVIDER_API_RESPONSE_TOO_LARGE")
                if payload.strip():
                    try:
                        # Deny duplicate keys in security-critical API structures.
                        def unique(pairs):
                            result = {}
                            for key, value in pairs:
                                if key in result:
                                    raise ValueError("duplicate")
                                result[key] = value
                            return result
                        result = json.loads(payload, object_pairs_hook=unique)
                    except (ValueError, UnicodeError):
                        raise Fault("PROVIDER_API_RESPONSE_INVALID") from None
                    if not isinstance(result, dict):
                        raise Fault("PROVIDER_API_RESPONSE_INVALID")
                else:
                    result = {}
                code = "API_RESPONSE_RECEIVED"
                return result
        except urllib.error.HTTPError as error:
            code = ("API_AUTH_REJECTED" if error.code in (401, 403)
                    else "API_ROUTE_UNSUPPORTED" if error.code == 404
                    else "API_CONFLICT" if error.code == 409
                    else "API_RATE_LIMITED" if error.code == 429
                    else "API_HTTP_REJECTED")
            # Do not parse/log freeform upstream error bodies or assume a rejected
            # write had no side effects; Operation Core must classify UNKNOWN.
            raise Fault(code, 409) from None
        except (urllib.error.URLError, TimeoutError, OSError):
            code = "API_OUTCOME_UNKNOWN" if is_write else "API_READ_UNAVAILABLE"
            raise Fault(code, 409) from None
        finally:
            self._event(method, path, None, code,
                        min(3600000, max(0, int((time.monotonic() - started) * 1000))),
                        is_write)

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
        logical = checked_logical(logical)
        if action == "create":
            body = {
                "username": logical,
                "resources": {name: spec[name] for name in ("cpu", "memory", "disk")},
                "image": "images:ubuntu/24.04",
                "enablePodman": False,
                "sshKeys": [spec["ssh_public_key"]]}
            if spec.get("_network_tenant") is not None:
                body["networkTenant"] = spec["_network_tenant"]
            # The upstream MCP adds enable_nesting as an out-of-band
            # capability: it is deliberately NOT a REST create field.
            route, method = "/v1/containers", "POST"
        elif action == "start":
            route, method, body = "/v1/containers/" + logical + "/start", "POST", {"waitForReady": False}
        elif action == "stop":
            route, method, body = "/v1/containers/" + logical + "/stop", "POST", {"force": False}
        elif action == "delete":
            route, method, body = "/v1/containers/" + logical, "DELETE", None
        elif action == "resize":
            body = spec.get("_resize")
            if not isinstance(body, dict) or not body or set(body) - {"cpu", "memory", "disk"}:
                raise Fault("RESIZE_SPEC_INVALID", 422)
            route, method = "/v1/containers/" + logical + "/resize", "PUT"
        else:
            raise Fault("ACTION_NOT_ALLOWED")
        return self.request(method, route, body=body, timeout=timeout)
