#!/usr/bin/env python3
"""Dependency-free, loopback-only MCP 2025-11-25 test client for BoxWeave.

The official MCP SDK remains the acceptance authority for full protocol and
OAuth; this small client is intentionally optimized for reproducible local
black-box probes. It never retries mutating operations or follows redirects.
"""
from __future__ import annotations

from dataclasses import dataclass, field
import http.client
import ipaddress
import json
import os
from pathlib import Path
import re
import time
from typing import Any, Callable, Iterator, Mapping
from urllib.parse import urlsplit

PROTOCOL = "2025-11-25"
MUTATIONS = frozenset({
    "box_create", "box_start", "box_stop", "box_update", "box_delete",
    "box_exec", "box_file_apply", "box_peer_set", "operation_cancel",
})
FINAL_STATES = frozenset({"SUCCEEDED", "FAILED", "CANCELLED", "UNKNOWN"})


class MCPError(RuntimeError):
    """Safe, non-secret summary of a failed MCP call."""


class MCPTransportError(MCPError):
    pass


class MCPProtocolError(MCPError):
    pass


class MCPToolError(MCPError):
    pass


class MCPUnsafeMutation(MCPError):
    pass


def _loopback_endpoint(endpoint: str) -> tuple[str, int, str, str]:
    """Reject remote targets, URL credentials, redirects and ambient proxies."""
    parsed = urlsplit(endpoint)
    if (parsed.scheme != "http" or parsed.username is not None or
            parsed.password is not None or parsed.query or parsed.fragment or
            parsed.path != "/mcp"):
        raise ValueError("only http://127.0.0.1:PORT/mcp or http://[::1]:PORT/mcp is allowed")
    host = parsed.hostname or ""
    try:
        port = parsed.port
    except ValueError as exc:
        raise ValueError("invalid explicit MCP loopback port") from exc
    if port is None or not (1 <= port <= 65535):
        raise ValueError("explicit loopback port required")
    if host == "localhost":
        # Pin the actual TCP destination to a numeric loopback address: do
        # not resolve localhost again after the initial safety check.
        dial_host = "127.0.0.1"
    else:
        try:
            ip = ipaddress.ip_address(host)
        except ValueError as exc:
            raise ValueError("MCP target must be a numeric loopback IP or localhost") from exc
        if host not in ("127.0.0.1", "::1") or not ip.is_loopback:
            raise ValueError("remote and noncanonical loopback MCP targets are forbidden")
        dial_host = host
    authority = f"[{host}]:{port}" if ":" in host else f"{host}:{port}"
    return dial_host, port, authority, parsed.path


def _safe_token(value: str) -> str:
    if not value or len(value) > 8192 or any(c in value for c in "\r\n\x00"):
        raise ValueError("invalid credential string")
    return value


def token_from_env(name: str = "BOXWEAVE_MCP_TOKEN") -> str:
    if not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]*", name):
        raise ValueError("invalid token environment variable name")
    return _safe_token(os.environ.get(name, ""))


def token_from_file(path: str | Path) -> str:
    file = Path(path)
    st = file.stat()
    if not file.is_file() or (st.st_mode & 0o077):
        raise ValueError("token file must be a regular owner-only file (chmod 600)")
    return _safe_token(file.read_text(encoding="utf-8").strip())


def _limit_json(data: bytes, *, max_bytes: int) -> Any:
    if not data or len(data) > max_bytes:
        raise MCPProtocolError("missing or oversized JSON-RPC response")
    try:
        return json.loads(data)
    except (ValueError, UnicodeDecodeError) as exc:
        raise MCPProtocolError("invalid JSON-RPC response") from exc


@dataclass(frozen=True)
class ToolReply:
    name: str
    structured: dict[str, Any]
    content: list[Any]
    raw: dict[str, Any] = field(repr=False)


class LoopbackMCP:
    """Small synchronous Streamable HTTP client for BoxWeave v1.

    No auto OAuth/browser flow, no automatic native retries, and no promise
    that a successful MCP receipt means native/Guest command success.
    """

    def __init__(self, endpoint: str = "http://127.0.0.1:18650/mcp", *,
                 token: str | None = None, timeout: float = 12,
                 max_response_bytes: int = 4 * 1024 * 1024,
                 allow_mutations: bool = False, protocol: str = PROTOCOL):
        self.dial_host, self.port, self.authority, self.path = _loopback_endpoint(endpoint)
        self.endpoint = endpoint
        self.token = _safe_token(token or os.environ.get("BOXWEAVE_MCP_TOKEN", ""))
        if not (0.1 <= timeout <= 300):
            raise ValueError("timeout must be between 0.1 and 300 seconds")
        if not (4096 <= max_response_bytes <= 64 * 1024 * 1024):
            raise ValueError("response cap must be 4096..67108864 bytes")
        if protocol != PROTOCOL:
            raise ValueError("only pinned MCP 2025-11-25 is supported by this harness")
        self.timeout = timeout
        self.max_response_bytes = max_response_bytes
        self.allow_mutations = allow_mutations
        self.protocol = protocol
        self.session_id: str | None = None
        self._request_id = 0
        self._initialized = False
        self._tools: dict[str, dict[str, Any]] | None = None
        self.last_status: int | None = None
        self.last_cache_control = ""

    def __enter__(self) -> "LoopbackMCP":
        self.connect()
        return self

    def __exit__(self, *_: object) -> None:
        self.close()

    def close(self) -> None:
        self._initialized = False
        self.session_id = None
        self._tools = None

    def _http(self, payload: dict[str, Any]) -> tuple[int, str, bytes]:
        raw = json.dumps(payload, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
        if len(raw) > self.max_response_bytes:
            raise MCPProtocolError("MCP request exceeds configured size cap")
        headers = {
            "Host": self.authority,
            "Authorization": "Bearer " + self.token,
            "Content-Type": "application/json",
            "Accept": "application/json, text/event-stream",
            "MCP-Protocol-Version": self.protocol,
            "Cache-Control": "no-store",
        }
        if self.session_id:
            headers["Mcp-Session-Id"] = self.session_id
        connection = http.client.HTTPConnection(self.dial_host, self.port, timeout=self.timeout)
        try:
            connection.request("POST", self.path, body=raw, headers=headers)
            res = connection.getresponse()
            self.last_status = res.status
            self.last_cache_control = res.getheader("Cache-Control", "")
            new_session = res.getheader("Mcp-Session-Id")
            if new_session:
                if len(new_session) > 256 or "\r" in new_session or "\n" in new_session:
                    raise MCPProtocolError("malformed MCP session identifier")
                self.session_id = new_session
            content_type = res.getheader("Content-Type", "").split(";", 1)[0].strip().lower()
            data = res.read(self.max_response_bytes + 1)
            if len(data) > self.max_response_bytes:
                raise MCPProtocolError("oversized MCP HTTP response")
            if 300 <= res.status < 400:
                raise MCPTransportError("MCP redirect refused (loopback-only)")
            if res.status not in (200, 202, 204):
                # Deliberately omit private body, token and credentials from
                # failure messages and report output.
                raise MCPTransportError(f"MCP HTTP {res.status} (body redacted)")
            return res.status, content_type, data
        except (TimeoutError, OSError, http.client.HTTPException) as exc:
            raise MCPTransportError("local MCP transport failed (details redacted)") from exc
        finally:
            connection.close()

    def _decode(self, content_type: str, data: bytes, request_id: int) -> dict[str, Any]:
        if content_type == "application/json":
            value = _limit_json(data, max_bytes=self.max_response_bytes)
            if not isinstance(value, dict):
                raise MCPProtocolError("expected JSON-RPC response object")
            if value.get("id") != request_id:
                raise MCPProtocolError("JSON-RPC response id mismatch")
            return value
        if content_type == "text/event-stream":
            # Bounded SSE support for servers that send event-stream bodies
            # rather than a JSONResponse; no resumability or infinite SSE.
            frames = re.split(rb"\r?\n\r?\n", data)
            for frame in frames:
                chunks = [line[5:].strip() for line in frame.splitlines() if line.startswith(b"data:")]
                if not chunks:
                    continue
                try:
                    value = json.loads(b"\n".join(chunks))
                except ValueError:
                    continue
                if isinstance(value, dict) and value.get("id") == request_id:
                    return value
            raise MCPProtocolError("no matching JSON-RPC response in bounded SSE body")
        raise MCPProtocolError("unsupported or missing MCP response media type")

    def _request(self, method: str, params: Mapping[str, Any] | None = None) -> dict[str, Any]:
        self._request_id += 1
        request_id = self._request_id
        payload = {"jsonrpc": "2.0", "id": request_id, "method": method,
                   "params": dict(params or {})}
        status, content_type, data = self._http(payload)
        if status != 200:
            raise MCPProtocolError("MCP request was accepted without a response")
        decoded = self._decode(content_type, data, request_id)
        if decoded.get("jsonrpc") != "2.0":
            raise MCPProtocolError("invalid JSON-RPC version")
        if "error" in decoded:
            err = decoded["error"]
            code = err.get("code", "unknown") if isinstance(err, dict) else "unknown"
            raise MCPProtocolError(f"MCP JSON-RPC rejected request (code={code})")
        result = decoded.get("result")
        if not isinstance(result, dict):
            raise MCPProtocolError("missing MCP JSON-RPC result")
        return result

    def connect(self) -> dict[str, Any]:
        if self._initialized:
            raise MCPProtocolError("client already initialized; call close() before reconnect")
        result = self._request("initialize", {"protocolVersion": self.protocol,
                      "capabilities": {}, "clientInfo": {"name": "boxweave-python-mcp-testkit", "version": "1.0"}})
        if result.get("protocolVersion") != self.protocol:
            raise MCPProtocolError("MCP server negotiated an unsupported protocol")
        # Notifications contain no JSON-RPC id; in 2025 Streamable HTTP the
        # server may respond with HTTP 202, 204 or (rarely) an empty 200.
        status, _, data = self._http({"jsonrpc": "2.0", "method": "notifications/initialized"})
        if status not in (200, 202, 204) or (status == 200 and data):
            raise MCPProtocolError("unexpected MCP initialized notification response")
        self._initialized = True
        return result

    def _require_connected(self) -> None:
        if not self._initialized:
            raise MCPProtocolError("MCP connection not initialized")

    def list_tools(self, cursor: str = "") -> tuple[list[dict[str, Any]], str]:
        self._require_connected()
        result = self._request("tools/list", {"cursor": cursor} if cursor else {})
        tools, next_cursor = result.get("tools"), result.get("nextCursor", "")
        if not isinstance(tools, list) or not isinstance(next_cursor, str) or any(
                not isinstance(tool, dict) or not isinstance(tool.get("name"), str) for tool in tools):
            raise MCPProtocolError("malformed tools/list result")
        return tools, next_cursor

    def all_tools(self, *, max_pages: int = 32) -> dict[str, dict[str, Any]]:
        self._require_connected()
        if not (1 <= max_pages <= 512):
            raise ValueError("max_pages must be 1..512")
        cursor = ""
        seen_cursors: set[str] = set()
        found: dict[str, dict[str, Any]] = {}
        for _ in range(max_pages):
            tools, next_cursor = self.list_tools(cursor)
            for tool in tools:
                if tool["name"] in found:
                    raise MCPProtocolError("duplicate MCP tool name in paged catalog")
                found[tool["name"]] = tool
            if not next_cursor:
                self._tools = found
                return found
            if next_cursor in seen_cursors:
                raise MCPProtocolError("repeated tools/list cursor")
            seen_cursors.add(next_cursor)
            cursor = next_cursor
        raise MCPProtocolError("tools/list page limit reached; catalog incomplete")

    def call(self, name: str, arguments: Mapping[str, Any] | None = None) -> ToolReply:
        self._require_connected()
        if self._tools is None:
            self.all_tools()
        assert self._tools is not None
        if name not in self._tools:
            raise MCPProtocolError("requested tool not advertised")
        annotation = self._tools[name].get("annotations") or {}
        read_only = bool(annotation.get("readOnlyHint") is True)
        if (name in MUTATIONS or not read_only) and not self.allow_mutations:
            raise MCPUnsafeMutation("mutation or unannotated tool blocked; opt in only on a disposable fixture")
        if not isinstance(arguments or {}, Mapping):
            raise ValueError("tool arguments must be an object")
        result = self._request("tools/call", {"name": name, "arguments": dict(arguments or {})})
        if result.get("isError") is True:
            raise MCPToolError(f"MCP tool {name} reported isError=true (details redacted)")
        structured = result.get("structuredContent")
        content = result.get("content", [])
        if structured is None:
            # Pinned older MCP clients may only expose TextContent. Never
            # guess from multiple unrelated pieces of text.
            if isinstance(content, list) and len(content) == 1 and isinstance(content[0], dict) and content[0].get("type") == "text":
                try:
                    structured = json.loads(content[0]["text"])
                except (ValueError, KeyError, TypeError) as exc:
                    raise MCPProtocolError("tool returned non-JSON legacy TextContent") from exc
        if not isinstance(structured, dict) or not isinstance(content, list):
            raise MCPProtocolError("tool did not return structured or compatible JSON text result")
        return ToolReply(name, structured, content, result)

    def host_status(self) -> dict[str, Any]:
        return self.call("host_status").structured

    def operation_get(self, operation_id: str, *, wait_ms: int = 0) -> dict[str, Any]:
        return self.call("operation_get", {"operation_id": operation_id, "wait_ms": wait_ms}).structured["operation"]

    def operation_list(self, *, limit: int = 50, cursor: str = "", **filters: Any) -> dict[str, Any]:
        return self.call("operation_list", {"limit": limit, **({"cursor": cursor} if cursor else {}), **filters}).structured["page"]

    def activity(self, view: str = "overview", *, limit: int = 50, cursor: str = "", **filters: Any) -> dict[str, Any]:
        return self.call("box_activity_get", {"view": view, "limit": limit,
                            **({"cursor": cursor} if cursor else {}), **filters}).structured["activity"]

    def operation_output(self, operation_id: str, *, stream: str = "stdout", offset: int = 0,
                         limit: int = 65536) -> dict[str, Any]:
        if stream not in ("stdout", "stderr"):
            raise ValueError("stream must be stdout or stderr")
        return self.call("operation_output_get", {"operation_id": operation_id,
                          "stream": stream, "offset": offset, "limit": limit}).structured

    def poll_operation(self, operation_id: str, *, seconds: float = 30,
                       interval: float = 0.5) -> dict[str, Any]:
        """Read only. UNKNOWN is a terminal stop requiring host-local review."""
        if not (0 <= seconds <= 3600 and 0.01 <= interval <= 60):
            raise ValueError("poll budget/interval invalid")
        deadline = time.monotonic() + seconds
        while True:
            receipt = self.operation_get(operation_id)
            if receipt.get("state") in FINAL_STATES:
                return receipt
            if time.monotonic() >= deadline:
                return receipt  # timed out, still nonterminal; no fake success
            time.sleep(min(interval, max(0, deadline - time.monotonic())))

    def walk_activity(self, view: str = "issues", *, limit: int = 50,
                      max_pages: int = 20, max_items: int = 1000,
                      **filters: Any) -> Iterator[dict[str, Any]]:
        if view not in ("overview", "issues") or not (1 <= limit <= 50) or not (1 <= max_pages <= 1000) or max_items < 1:
            raise ValueError("bounded overview/issues walks only")
        cursor = ""
        seen_cursors: set[str] = set()
        seen_keys: set[str] = set()
        total = 0
        for _ in range(max_pages):
            result = self.activity(view, limit=limit, cursor=cursor, **filters)
            items = result.get("issues" if view == "issues" else "operations", [])
            if not isinstance(items, list):
                raise MCPProtocolError("Activity page did not contain a list")
            for item in items:
                if not isinstance(item, dict):
                    raise MCPProtocolError("invalid Activity item")
                key = (str(item.get("operation_id", "")) if view == "overview" else
                       "|".join(str(item.get(k, "")) for k in ("reason", "operation_id", "resource_ref")))
                if key in seen_keys:
                    raise MCPProtocolError("duplicate Activity item across signed pages")
                seen_keys.add(key)
                yield item
                total += 1
                if total >= max_items:
                    return
            cursor = result.get("next_cursor", "")
            if not cursor:
                if result.get("truncated"):
                    raise MCPProtocolError("Activity result truncated without a continuation cursor")
                return
            if not result.get("truncated") or cursor in seen_cursors:
                raise MCPProtocolError("stale, inconsistent or cyclic Activity cursor")
            seen_cursors.add(cursor)
        raise MCPProtocolError("Activity page cap reached; increase explicit bounds")

    def walk_operations(self, *, limit: int = 50, max_pages: int = 20,
                        max_items: int = 1000, **filters: Any) -> Iterator[dict[str, Any]]:
        if not (1 <= limit <= 100 and 1 <= max_pages <= 1000 and max_items >= 1):
            raise ValueError("invalid operation history scan bounds")
        cursor = ""
        seen: set[str] = set()
        for _ in range(max_pages):
            page = self.operation_list(limit=limit, cursor=cursor, **filters)
            for op in page.get("operations", []):
                op_id = op.get("operation_id")
                if not isinstance(op_id, str) or op_id in seen:
                    raise MCPProtocolError("invalid or duplicate operation ID")
                seen.add(op_id)
                yield op
                if len(seen) >= max_items:
                    return
            nxt = page.get("next_cursor", "")
            if not nxt:
                return
            if nxt == cursor:
                raise MCPProtocolError("repeating operation list cursor")
            cursor = nxt
        raise MCPProtocolError("operation history page cap reached")

    def recover_work(self, work_ref: str, *, max_items: int = 100) -> dict[str, Any]:
        """Read-only recovery manifest: never replays or considers UNKNOWN safe."""
        if not work_ref or max_items < 1 or max_items > 1000:
            raise ValueError("work_ref and max_items<=1000 required")
        originals = list(self.walk_operations(work_ref=work_ref, max_items=max_items))
        issues = list(self.walk_activity("issues", work_ref=work_ref, max_items=max_items))
        unknown = sorted({x["operation_id"] for x in originals if x.get("state") == "UNKNOWN"})
        evidence = []
        for op_id in unknown[:min(20, max_items)]:
            view = self.activity("evidence", operation_id=op_id, limit=50)
            if view.get("evidence"):
                evidence.append({"operation_id": op_id,
                                 "limitations": view["evidence"].get("limitations", [])})
        return {"work_ref": work_ref, "operation_ids": [x.get("operation_id") for x in originals],
                "unknown_originals": unknown, "issue_count": len(issues),
                "unknown_evidence": evidence, "safe_to_retry_unknown": False,
                "coverage": "BOUNDED_SQLITE_READ_ONLY_NOT_AGENT_PLAN",
                "requires_manual_review": bool(unknown)}


@dataclass
class Check:
    name: str
    status: str
    detail: str = ""


class FaultInjectingMCP(LoopbackMCP):
    """Drop exactly one already-processed *read* reply; never drop mutations.

    This simulates an uncertain read result after the HTTP server executed a
    request. It is NOT a native mutation ACK-loss simulation.
    """

    def __init__(self, *args: Any, **kwargs: Any):
        super().__init__(*args, **kwargs)
        self.drop_next_read_reply = False

    def _http(self, payload: dict[str, Any]) -> tuple[int, str, bytes]:
        method = payload.get("method")
        args = payload.get("params") or {}
        name = args.get("name") if isinstance(args, dict) else None
        should_drop = (self.drop_next_read_reply and method == "tools/call" and
                       name in {"host_status", "operation_get", "operation_list",
                                "box_activity_get", "box_list", "box_get"})
        status, content_type, data = super()._http(payload)
        if should_drop:
            self.drop_next_read_reply = False
            raise MCPTransportError("intentionally dropped post-response read result")
        return status, content_type, data


class ProbeReport:
    """Sanitized scoped checks; never include arguments, tokens or raw output."""

    def __init__(self, suite: str):
        if not re.fullmatch(r"[A-Za-z0-9_.-]{1,100}", suite):
            raise ValueError("report suite must be a nonsecret identifier")
        self.suite = suite
        self.started = time.time()
        self.checks: list[Check] = []

    def check(self, name: str, condition: bool, *, detail: str = "") -> None:
        if not re.fullmatch(r"[A-Za-z0-9_.-]{1,100}", name):
            raise ValueError("check name must be a nonsecret identifier")
        if not condition:
            self.checks.append(Check(name, "FAIL"))
            raise AssertionError(f"probe failed: {name}")
        self.checks.append(Check(name, "PASS"))

    def expected_error(self, name: str, fn: Callable[[], Any], expected: type[Exception]) -> None:
        if not re.fullmatch(r"[A-Za-z0-9_.-]{1,100}", name):
            raise ValueError("check name must be a nonsecret identifier")
        try:
            fn()
        except expected:
            self.checks.append(Check(name, "PASS"))
        else:
            self.checks.append(Check(name, "FAIL"))
            raise AssertionError(f"expected {expected.__name__}: {name}")

    def as_dict(self) -> dict[str, Any]:
        return {"suite": self.suite, "scope": "LOCAL_LOOPBACK_MCP_ONLY",
                "duration_ms": round((time.time() - self.started) * 1000),
                "passed": sum(c.status == "PASS" for c in self.checks),
                "failed": sum(c.status == "FAIL" for c in self.checks),
                "checks": [{"name": c.name, "status": c.status} for c in self.checks],
                "limits": ["NOT_OAUTH_BROWSER_ACCEPTANCE", "NOT_REAL_AGENT_INTENT",
                           "NOT_NATIVE_POSTSTATE", "NOT_RELEASE_SIGNOFF"]}

    def write(self, destination: str | Path) -> None:
        path = Path(destination)
        if not path.parent.is_dir():
            raise ValueError("report parent directory must exist")
        path.write_text(json.dumps(self.as_dict(), indent=2, sort_keys=True) + "\n", encoding="utf-8")
