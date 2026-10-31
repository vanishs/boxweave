"""Bounded POSIX MCP stdio client for an operator-configured launcher."""
import json
import os
import selectors
import signal
import subprocess
import time
from . import diagnostics
from .domain import Fault

class MCP:
    def __init__(self, argv, timeout=30):
        self.deadline = time.monotonic() + timeout
        self.ident = 0
        self.total = 0
        self.buffer = b""
        self.p = subprocess.Popen(argv, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                  stderr=subprocess.DEVNULL, start_new_session=True)
        os.set_blocking(self.p.stdin.fileno(), False)
        os.set_blocking(self.p.stdout.fileno(), False)

    def wait(self, stream, event):
        left = self.deadline - time.monotonic()
        if left <= 0:
            raise diagnostics.attach(Fault("DEADLINE_EXCEEDED"), "TIMEOUT")
        with selectors.DefaultSelector() as s:
            s.register(stream, event)
            if not s.select(left):
                raise diagnostics.attach(Fault("DEADLINE_EXCEEDED"), "TIMEOUT")

    def send(self, obj):
        data = (json.dumps(obj) + "\n").encode()
        while data:
            self.wait(self.p.stdin, selectors.EVENT_WRITE)
            try:
                n = os.write(self.p.stdin.fileno(), data)
            except BlockingIOError:
                continue
            data = data[n:]

    def call(self, method, params):
        self.ident += 1
        self.send({"jsonrpc":"2.0", "id":self.ident, "method":method, "params":params})
        for _ in range(256):
            while b"\n" not in self.buffer:
                self.wait(self.p.stdout, selectors.EVENT_READ)
                try:
                    chunk = os.read(self.p.stdout.fileno(), 65536)
                except BlockingIOError:
                    continue
                if not chunk:
                    raise diagnostics.attach(Fault("MCP_STREAM_CLOSED"), "TRANSPORT_EOF")
                self.total += len(chunk)
                self.buffer += chunk
                if self.total > 8388608 or len(self.buffer) > 2097152:
                    raise diagnostics.attach(Fault("OUTPUT_LIMIT"), "OUTPUT_LIMIT")
            line, self.buffer = self.buffer.split(b"\n", 1)
            try:
                msg = json.loads(line)
            except (ValueError, UnicodeError):
                raise diagnostics.attach(Fault("MCP_INVALID"), "MALFORMED_PROTOCOL")
            if not isinstance(msg, dict) or msg.get("jsonrpc") != "2.0":
                raise diagnostics.attach(Fault("MCP_INVALID"), "MALFORMED_PROTOCOL")
            if "method" in msg:
                if "id" in msg:
                    self.send({"jsonrpc":"2.0", "id":msg["id"],
                               "error":{"code":-32601, "message":"Unsupported request"}})
                continue
            if type(msg.get("id")) is not int or msg["id"] != self.ident:
                raise diagnostics.attach(Fault("MCP_INVALID"), "MALFORMED_PROTOCOL")
            if ("result" in msg) == ("error" in msg):
                raise diagnostics.attach(Fault("MCP_INVALID"), "MALFORMED_PROTOCOL")
            if "error" in msg:
                message = str(msg["error"].get("message", "")).lower() if isinstance(msg["error"], dict) else ""
                if any(marker in message for marker in ("knownhosts: key mismatch", "host key verification failed", "remote host identification has changed")):
                    raise diagnostics.attach(Fault("TRUST_FAILED"), "UPSTREAM_JSONRPC_REJECTION")
                if method=="tools/call" and params.get("name")=="get_network_policy":
                    prefix="Tool execution failed: API error (status 404): "
                    original=msg["error"].get("message","") if isinstance(msg["error"],dict) else ""
                    if original.startswith(prefix):
                        try:
                            detail=json.loads(original[len(prefix):])
                            expected='rpc error: code = NotFound desc = no network policy for tenant "'+params["arguments"]["tenant"]+'"'
                            if detail=={"code":404,"error":expected}:raise diagnostics.attach(Fault("POLICY_ABSENT",404), "UPSTREAM_JSONRPC_REJECTION")
                        except (ValueError,KeyError,TypeError):pass
                raise diagnostics.attach(Fault("PROVIDER_REJECTED"), "UPSTREAM_JSONRPC_REJECTION")
            if not isinstance(msg["result"], dict):
                raise diagnostics.attach(Fault("MCP_INVALID"), "MALFORMED_PROTOCOL")
            return msg["result"]
        raise diagnostics.attach(Fault("OUTPUT_LIMIT"), "OUTPUT_LIMIT")

    def discover(self):
        r = self.call("initialize", {"protocolVersion":"2024-11-05", "capabilities":{},
                      "clientInfo":{"name":"boxweave-mini", "version":"0.2.1"}})
        if r.get("protocolVersion") != "2024-11-05":
            raise diagnostics.attach(Fault("PROTOCOL_UNSUPPORTED"), "MALFORMED_PROTOCOL")
        if not isinstance(r.get("capabilities", {}).get("tools"), dict):
            raise diagnostics.attach(Fault("MCP_INVALID"), "MALFORMED_PROTOCOL")
        self.send({"jsonrpc":"2.0", "method":"notifications/initialized"})
        tools, seen, params = {}, set(), {}
        for _ in range(16):
            r = self.call("tools/list", params)
            if not isinstance(r.get("tools"), list):
                raise diagnostics.attach(Fault("MCP_INVALID"), "MALFORMED_PROTOCOL")
            for t in r["tools"]:
                if not isinstance(t, dict) or not isinstance(t.get("name"), str):
                    raise diagnostics.attach(Fault("MCP_INVALID"), "MALFORMED_PROTOCOL")
                tools[t["name"]] = t
            cursor = r.get("nextCursor")
            if cursor is None:
                return tools
            if not isinstance(cursor, str) or not cursor or cursor in seen:
                raise diagnostics.attach(Fault("MCP_INVALID"), "MALFORMED_PROTOCOL")
            seen.add(cursor)
            params = {"cursor":cursor}
        raise diagnostics.attach(Fault("OUTPUT_LIMIT"), "OUTPUT_LIMIT")

    def list_resources(self):
        """Bounded resource listing. Caller must initialize with discover first."""
        output, seen, cursor = [], set(), {}
        for _ in range(16):
            result = self.call("resources/list", cursor)
            if not isinstance(result.get("resources"), list):
                raise Fault("MCP_INVALID")
            for item in result["resources"]:
                if (not isinstance(item, dict) or not isinstance(item.get("uri"),str)
                        or len(item["uri"]) > 512 or not item["uri"]):
                    raise Fault("MCP_INVALID")
                output.append({"uri":item["uri"],"name":item.get("name","") if
                               isinstance(item.get("name",""),str) else ""})
                if len(output)>256:raise Fault("OUTPUT_LIMIT")
            token=result.get("nextCursor")
            if token is None:return output
            if not isinstance(token,str) or not token or token in seen:
                raise Fault("MCP_INVALID")
            seen.add(token);cursor={"cursor":token}
        raise Fault("OUTPUT_LIMIT")

    def read_resource(self, uri):
        if not isinstance(uri,str) or not uri.startswith("containarium://") or len(uri)>512:
            raise Fault("RESOURCE_URI_NOT_ALLOWED",422)
        result=self.call("resources/read",{"uri":uri})
        entries=result.get("contents")
        if not isinstance(entries,list) or len(entries)!=1:raise Fault("MCP_INVALID")
        entry=entries[0]
        if (not isinstance(entry,dict) or entry.get("uri")!=uri or
                not isinstance(entry.get("text"),str) or
                len(entry["text"].encode("utf-8"))>65536):
            raise Fault("MCP_INVALID")
        return entry["text"]

    def tool(self, tool, args):
        r = self.call("tools/call", {"name":tool, "arguments":args})
        if "isError" in r and type(r["isError"]) is not bool:
            raise diagnostics.attach(Fault("MCP_INVALID"), "MALFORMED_PROTOCOL")
        if r.get("isError"):
            raise diagnostics.attach(Fault("PROVIDER_REJECTED"), "TOOL_IS_ERROR")
        if not isinstance(r.get("content"), list):
            raise diagnostics.attach(Fault("MCP_INVALID"), "MALFORMED_PROTOCOL")
        if any(not isinstance(b, dict) or b.get("type") != "text" or not isinstance(b.get("text"), str)
               for b in r["content"]):
            raise diagnostics.attach(Fault("MCP_INVALID"), "MALFORMED_PROTOCOL")
        return r

    def close(self):
        if self.p.poll() is None:
            os.killpg(self.p.pid, signal.SIGTERM)
            try:
                self.p.wait(timeout=2)
            except subprocess.TimeoutExpired:
                os.killpg(self.p.pid, signal.SIGKILL)
                self.p.wait(timeout=2)
        self.p.stdin.close()
        self.p.stdout.close()

    def __enter__(self):
        return self

    def __exit__(self, *_):
        self.close()
