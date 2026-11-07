"""Bounded Containarium debug_container REST read, standalone Gateway safe.

Shared stdlib-only validation and request code. This module provides exactly
one fixed GET route; it is never a generic HTTP proxy. The Agent Gateway now uses the mandatory application REST binding.
"""
import json
import os
from pathlib import Path
import re
import stat
import urllib.error
import urllib.request
import urllib.parse

_TARGET = re.compile(r"[a-z][a-z0-9-]{0,62}\Z")
_MAX_BYTES = 32768


class DebugRESTError(ValueError):
    """Fixed public error codes only; never include token, upstream text or URI."""


def logical_name(value):
    if not isinstance(value, str) or _TARGET.fullmatch(value) is None:
        raise DebugRESTError("DEBUG_REST_TARGET_INVALID")
    return value


def route(value):
    return "/v1/containers/" + logical_name(value) + "/debug"


def endpoint(value):
    try:
        u = urllib.parse.urlsplit(value)
        if (not isinstance(value, str) or u.scheme != "http"
                or u.netloc != "127.0.0.1:" + str(u.port)
                or u.hostname != "127.0.0.1" or u.username is not None
                or u.password is not None or u.path or u.query or u.fragment
                or not 1024 <= u.port <= 65535):
            raise ValueError()
    except (TypeError, ValueError, AttributeError):
        raise DebugRESTError("DEBUG_REST_ENDPOINT_INVALID") from None
    return value


def token_path(value):
    if (not isinstance(value, str) or len(value) > 1024
            or not Path(value).is_absolute() or ".." in Path(value).parts):
        raise DebugRESTError("DEBUG_REST_TOKEN_PATH_INVALID")
    return Path(value)


def token(value):
    path = token_path(value)
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
    try:
        descriptor = os.open(path, flags)
        with os.fdopen(descriptor, "rb") as stream:
            info = os.fstat(stream.fileno())
            if (not stat.S_ISREG(info.st_mode) or info.st_nlink != 1
                    or info.st_uid != os.geteuid() or info.st_mode & 0o277
                    or not 20 <= info.st_size <= 8192):
                raise DebugRESTError("DEBUG_REST_TOKEN_UNSAFE")
            raw = stream.read(8193).strip()
        if not 20 <= len(raw) <= 8192 or any(c < 33 or c > 126 for c in raw):
            raise DebugRESTError("DEBUG_REST_TOKEN_UNSAFE")
        return raw.decode("ascii")
    except (OSError, UnicodeError):
        raise DebugRESTError("DEBUG_REST_TOKEN_UNAVAILABLE") from None


def unique_pairs(pairs):
    value = {}
    for key, item in pairs:
        if key in value:
            raise ValueError("duplicate key")
        value[key] = item
    return value


def validate(value):
    """Validate REST/MCP-compatible shape; Web applies further safe projection."""
    if (not isinstance(value, dict)
            or value.get("containerState") not in ("running", "stopped", "missing")
            or any(key in value and type(value[key]) is not bool for key in
                   ("hostUserExists", "hostUserShellExists"))
            or "recentSshdRejections" in value and
            (not isinstance(value["recentSshdRejections"], list) or
             any(not isinstance(item, str) for item in value["recentSshdRejections"]))
            or "nextActions" in value and
            (not isinstance(value["nextActions"], list) or
             any(not isinstance(item, str) for item in value["nextActions"]))):
        raise DebugRESTError("DEBUG_REST_RESPONSE_INVALID")
    if len(json.dumps(value, ensure_ascii=False).encode("utf-8")) > _MAX_BYTES:
        raise DebugRESTError("DEBUG_REST_RESPONSE_OVERSIZED")
    return value


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


def read(config, username, *, opener=None):
    """Single authenticated read, no proxy, redirects, retry or secret-bearing logs."""
    if not isinstance(config, dict) or set(config) != {"url", "token_file"}:
        raise DebugRESTError("DEBUG_REST_CONFIG_INVALID")
    uri = endpoint(config["url"]) + route(username)
    credential = token(config["token_file"])
    request = urllib.request.Request(uri, method="GET", headers={
        "Authorization": "Bearer " + credential, "Accept": "application/json"})
    del credential
    transport = opener or urllib.request.build_opener(
        urllib.request.ProxyHandler({}), NoRedirect())
    try:
        with transport.open(request, timeout=20) as response:
            if not 200 <= response.status < 300:
                raise DebugRESTError("DEBUG_REST_HTTP_REJECTED")
            body = response.read(_MAX_BYTES + 1)
            if len(body) > _MAX_BYTES:
                raise DebugRESTError("DEBUG_REST_RESPONSE_OVERSIZED")
            return validate(json.loads(body, object_pairs_hook=unique_pairs))
    except urllib.error.HTTPError as error:
        code = ("DEBUG_REST_AUTH_REJECTED" if error.code in (401, 403)
                else "DEBUG_REST_ROUTE_UNVERIFIED" if error.code == 404
                else "DEBUG_REST_HTTP_REJECTED")
        raise DebugRESTError(code) from None
    except (urllib.error.URLError, TimeoutError, OSError):
        raise DebugRESTError("DEBUG_REST_UNAVAILABLE") from None
    except (ValueError, UnicodeError) as error:
        if isinstance(error, DebugRESTError):
            raise
        raise DebugRESTError("DEBUG_REST_RESPONSE_INVALID") from None


