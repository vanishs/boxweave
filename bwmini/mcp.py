"""Bounded POSIX MCP stdio client for an operator-configured launcher."""
import json
import os
import selectors
import signal
import subprocess
import time
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
            raise Fault("DEADLINE_EXCEEDED")
        with selectors.DefaultSelector() as s:
            s.register(stream, event)
            if not s.select(left):
                raise Fault("DEADLINE_EXCEEDED")

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
                    raise Fault("MCP_STREAM_CLOSED")
                self.total += len(chunk)
                self.buffer += chunk
                if self.total > 8388608 or len(self.buffer) > 2097152:
                    raise Fault("OUTPUT_LIMIT")
            line, self.buffer = self.buffer.split(b"\n", 1)
            try:
                msg = json.loads(line)
            except (ValueError, UnicodeError):
                raise Fault("MCP_INVALID")
            if not isinstance(msg, dict) or msg.get("jsonrpc") != "2.0":
                raise Fault("MCP_INVALID")
            if "method" in msg:
                if "id" in msg:
                    self.send({"jsonrpc":"2.0", "id":msg["id"],
                               "error":{"code":-32601, "message":"Unsupported request"}})
                continue
            if type(msg.get("id")) is not int or msg["id"] != self.ident:
                raise Fault("MCP_INVALID")
            if ("result" in msg) == ("error" in msg):
                raise Fault("MCP_INVALID")
            if "error" in msg:
                message = str(msg["error"].get("message", "")).lower() if isinstance(msg["error"], dict) else ""
                if any(marker in message for marker in ("knownhosts: key mismatch", "host key verification failed", "remote host identification has changed")):
                    raise Fault("TRUST_FAILED")
                raise Fault("PROVIDER_REJECTED")
            if not isinstance(msg["result"], dict):
                raise Fault("MCP_INVALID")
            return msg["result"]
        raise Fault("OUTPUT_LIMIT")

    def discover(self):
        r = self.call("initialize", {"protocolVersion":"2024-11-05", "capabilities":{},
                      "clientInfo":{"name":"boxweave-mini", "version":"0.2.1"}})
        if r.get("protocolVersion") != "2024-11-05":
            raise Fault("PROTOCOL_UNSUPPORTED")
        if not isinstance(r.get("capabilities", {}).get("tools"), dict):
            raise Fault("MCP_INVALID")
        self.send({"jsonrpc":"2.0", "method":"notifications/initialized"})
        tools, seen, params = {}, set(), {}
        for _ in range(16):
            r = self.call("tools/list", params)
            if not isinstance(r.get("tools"), list):
                raise Fault("MCP_INVALID")
            for t in r["tools"]:
                if not isinstance(t, dict) or not isinstance(t.get("name"), str):
                    raise Fault("MCP_INVALID")
                tools[t["name"]] = t
            cursor = r.get("nextCursor")
            if cursor is None:
                return tools
            if not isinstance(cursor, str) or not cursor or cursor in seen:
                raise Fault("MCP_INVALID")
            seen.add(cursor)
            params = {"cursor":cursor}
        raise Fault("OUTPUT_LIMIT")

    def tool(self, tool, args):
        r = self.call("tools/call", {"name":tool, "arguments":args})
        if "isError" in r and type(r["isError"]) is not bool:
            raise Fault("MCP_INVALID")
        if r.get("isError"):
            raise Fault("PROVIDER_REJECTED")
        if not isinstance(r.get("content"), list):
            raise Fault("MCP_INVALID")
        if any(not isinstance(b, dict) or b.get("type") != "text" or not isinstance(b.get("text"), str)
               for b in r["content"]):
            raise Fault("MCP_INVALID")
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
