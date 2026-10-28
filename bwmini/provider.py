"""A fixture is explicit. Missing live credentials never enable a fixture."""
import json
import pathlib
import re
import shlex
from .domain import Target, Fault
from .mcp import MCP

def text(result):
    return "\n".join(b["text"] for b in result["content"])

def execution(result):
    structured = result.get("structuredContent")
    if isinstance(structured, dict):
        code, out = structured.get("exit_code"), structured.get("stdout")
        if type(code) is int and isinstance(out, str) and len(out) <= 4096:
            return code, out
        raise Fault("EXECUTION_FORMAT_UNSUPPORTED")
    # Only the observed platform envelope, never markers parsed out of guest text.
    match = re.fullmatch(r"exit_code: (-?\d+)\n\n--- stdout ---\n([\s\S]{0,4096})", text(result))
    if match is None:
        raise Fault("EXECUTION_FORMAT_UNSUPPORTED")
    return int(match[1]), match[2]

class Fixture:
    mode = "fixture"
    def __init__(self, target, counter=None):
        self.target, self.counter, self.calls = target, counter, []
    def observe(self, logical):
        if logical != self.target.logical:
            raise Fault("TARGET_NOT_ALLOWED", 403)
        return self.target
    def hello(self, target, path, timeout):
        self.calls.append("identity")
        self.calls.append("hello")
        if self.counter:
            with open(self.counter, "a", encoding="utf-8") as f:
                f.write("hello\n")
        return {"stdout":"hello world\n", "exit_code":0, "identity":{"user":target.user,
                "hostname":target.hostname}, "evidence":"fixture", "remote_cleanup":"not_applicable"}

class Provider:
    mode = "mcp"
    def __init__(self, config):
        self.config = config
        argv = config.get("launcher")
        if not isinstance(argv, list) or not argv or any(not isinstance(a, str) for a in argv):
            raise Fault("MCP_LAUNCHER_REQUIRED", 422)
        if not pathlib.Path(argv[0]).is_absolute() or not pathlib.Path(argv[0]).is_file():
            raise Fault("MCP_LAUNCHER_UNAVAILABLE", 422)
        self.argv = argv

    def session(self, timeout=30):
        return MCP(self.argv, min(30, max(1, timeout)))

    def observe(self, logical):
        expected = self.config["targets"].get(logical)
        if expected is None:
            raise Fault("TARGET_NOT_ALLOWED", 403)
        with self.session() as m:
            tools = m.discover()
            if "get_container" not in tools:
                raise Fault("TOOL_UNAVAILABLE")
            result = m.tool("get_container", {"username":logical})
            try:
                c = json.loads(text(result))["container"]
                target = Target(self.config["provider"], self.config["backend"], self.config["project"],
                                logical, c["name"], str(c["createdAt"]), c["username"], c["name"])
            except (KeyError, TypeError, ValueError):
                raise Fault("TARGET_FORMAT_UNSUPPORTED")
            if c.get("state") != "CONTAINER_STATE_RUNNING":
                raise Fault("NOT_CONNECTION_READY")
            if target.data() != Target(**expected).data():
                raise Fault("TARGET_CHANGED")
            return target

    def hello(self, target, path, timeout):
        with self.session(timeout) as m:
            tools = m.discover()
            spec = tools.get("connect", {}).get("inputSchema", {}).get("properties", {})
            key = "box" if "box" in spec else "username" if "username" in spec else None
            if key is None or "exec" not in spec:
                raise Fault("CONNECT_CONTRACT_UNSUPPORTED")
            result = m.tool("connect", {key:target.logical, "exec":"whoami && hostname"})
            code, out = execution(result)
            if code != 0 or out.splitlines() != [target.user, target.hostname]:
                raise Fault("IDENTITY_MISMATCH")
            # No caller-supplied shell command; only the approved workspace and fixed action.
            cmd = "cd -- " + shlex.quote(path) + " && python3 -c " + shlex.quote("print('hello world')")
            code, out = execution(m.tool("connect", {key:target.logical, "exec":cmd}))
            return {"stdout":out, "exit_code":code, "identity":{"user":target.user,
                    "hostname":target.hostname}, "evidence":"live_mcp",
                    "remote_cleanup":"not_verified", "protocol":"2024-11-05"}
