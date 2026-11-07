"""Shared fixed-scope localhost REST transport for app and gateway.

No arbitrary URI or credential exposure, retries or mutation replay. Compatible
with both the installed standalone mcp_gate and package-owned application.
"""
import json
import os
from pathlib import Path
import re
import stat
import time
import urllib.error
import urllib.parse
import urllib.request


class WireFault(Exception):
    def __init__(self, code, status=409):
        self.code, self.status = code, status
        super().__init__(code)


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


def checked_endpoint(value):
    try:
        p = urllib.parse.urlsplit(value)
        if (not isinstance(value, str) or p.scheme != "http" or
                p.hostname != "127.0.0.1" or p.username is not None or
                p.password is not None or p.netloc != "127.0.0.1:" + str(p.port) or
                not 1024 <= p.port <= 65535 or p.path or p.query or p.fragment):
            raise ValueError()
    except (ValueError, TypeError, AttributeError):
        raise WireFault("PROVIDER_API_LOOPBACK_REQUIRED", 422) from None
    return value


def checked_token_file(value):
    if (not isinstance(value, str) or not Path(value).is_absolute()
            or ".." in Path(value).parts or len(value) > 1024):
        raise WireFault("PROVIDER_API_TOKEN_FILE_REQUIRED", 422)
    return Path(value)


def checked_logical(value):
    if (not isinstance(value, str) or
            re.fullmatch(r"[a-z][a-z0-9-]{0,62}", value) is None):
        raise WireFault("API_TARGET_INVALID", 422)
    if value.startswith("containarium-core-"):
        raise WireFault("TARGET_NOT_ALLOWED", 403)
    return value


def request(owner, method, path, body=None, timeout=30):
    """The sole transport implementation used by both REST entrypoint adapters.

    `owner` supplies validated .url, .opener and ._credential. An optional
    ._event callback sees bounded status metadata only; never auth or bodies.
    """
    if method not in ("GET", "POST", "PUT", "DELETE"):
        raise WireFault("PROVIDER_API_METHOD_DENIED", 422)
    if (not isinstance(path, str) or not path.startswith("/v1/")
            or "//" in path or ".." in path or "?" in path
            or not 1 <= len(path) <= 192):
        raise WireFault("PROVIDER_API_ROUTE_DENIED", 422)
    if type(timeout) not in (int, float) or not 0 < timeout <= 1800:
        raise WireFault("DEADLINE_EXCEEDED")
    raw = None
    if body is not None:
        try:
            raw = json.dumps(body, ensure_ascii=True, allow_nan=False,
                             separators=(",", ":")).encode()
        except (TypeError, ValueError, OverflowError):
            raise WireFault("PROVIDER_API_REQUEST_INVALID", 422) from None
        if len(raw) > 65536:
            raise WireFault("PROVIDER_API_REQUEST_TOO_LARGE", 422)
    credential = owner._credential()
    req = urllib.request.Request(owner.url + path, data=raw, method=method,
                                 headers={"Authorization": "Bearer " + credential,
                                          "Content-Type": "application/json",
                                          "Accept": "application/json"})
    del credential
    is_write = method != "GET"
    started = time.monotonic()
    code = "API_OUTCOME_UNKNOWN" if is_write else "API_READ_UNAVAILABLE"
    try:
        with owner.opener.open(req, timeout=timeout) as response:
            if not 200 <= response.status < 300:
                raise WireFault("PROVIDER_API_UNEXPECTED_STATUS")
            data = response.read(524289)
            if len(data) > 524288:
                raise WireFault("PROVIDER_API_RESPONSE_TOO_LARGE")
            if data.strip():
                def unique(pairs):
                    out = {}
                    for k, v in pairs:
                        if k in out:
                            raise ValueError("duplicate")
                        out[k] = v
                    return out
                try:
                    value = json.loads(data, object_pairs_hook=unique)
                except (ValueError, UnicodeError):
                    raise WireFault("PROVIDER_API_RESPONSE_INVALID") from None
                if not isinstance(value, dict):
                    raise WireFault("PROVIDER_API_RESPONSE_INVALID")
            else:
                value = {}
            code = "API_RESPONSE_RECEIVED"
            return value
    except urllib.error.HTTPError as error:
        code = ("API_AUTH_REJECTED" if error.code in (401, 403)
                else "API_TARGET_NOT_FOUND" if error.code == 404 and
                method == "GET" and re.fullmatch(
                    r"/v1/containers/[a-z][a-z0-9-]{0,62}", path)
                else "API_ROUTE_UNSUPPORTED" if error.code == 404
                else "API_CONFLICT" if error.code == 409
                else "API_RATE_LIMITED" if error.code == 429
                else "API_HTTP_REJECTED")
        raise WireFault(code) from None
    except (urllib.error.URLError, TimeoutError, OSError):
        code = "API_OUTCOME_UNKNOWN" if is_write else "API_READ_UNAVAILABLE"
        raise WireFault(code) from None
    finally:
        if hasattr(owner, "_event"):
            try:
                owner._event(method, path, None, code,
                             min(3600000, max(0, int(
                                 (time.monotonic() - started) * 1000))), is_write)
            except Exception:
                pass


class Client:
    """Minimal standalone gateway HTTP client, same transport as app RESTClient."""

    def __init__(self, config):
        if not isinstance(config, dict) or set(config) != {"url", "token_file"}:
            raise WireFault("PROVIDER_API_CONFIG_INVALID", 422)
        self.url = checked_endpoint(config["url"])
        self.token_file = checked_token_file(config["token_file"])
        self.opener = urllib.request.build_opener(
            urllib.request.ProxyHandler({}), _NoRedirect())

    def _credential(self):
        flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
        try:
            fd = os.open(self.token_file, flags)
            with os.fdopen(fd, "rb") as stream:
                meta = os.fstat(stream.fileno())
                if (not stat.S_ISREG(meta.st_mode) or meta.st_uid != os.geteuid()
                        or meta.st_mode & 0o077 or meta.st_mode & 0o200
                        or meta.st_nlink != 1 or not 20 <= meta.st_size <= 8192):
                    raise WireFault("PROVIDER_API_TOKEN_UNSAFE")
                token = stream.read(8193).strip()
            if (not 20 <= len(token) <= 8192
                    or any(ch < 33 or ch > 126 for ch in token)):
                raise WireFault("PROVIDER_API_TOKEN_UNSAFE")
            return token.decode("ascii")
        except (OSError, UnicodeError):
            raise WireFault("PROVIDER_API_TOKEN_UNAVAILABLE") from None

    def request(self, method, path, body=None, timeout=30):
        return request(self, method, path, body, timeout)
