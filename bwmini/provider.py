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
        self.lifecycle_config = config.get("lifecycle")
        from .lifecycle import Lifecycle
        Lifecycle(None,self)

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

    def lifecycle_inventory(self):
        with self.session() as m:
            m.discover()
            raw=text(m.tool("list_containers",{})).strip()
        if raw in ("No containers found.","No containers found"):
            return set()
        if raw.startswith("{"):
            try:
                value=json.loads(raw)
                rows=value["containers"]
                if not isinstance(rows,list) or value.get("nextCursor"):
                    raise ValueError()
                names=[row["username"] for row in rows]
            except (KeyError,TypeError,ValueError):
                raise Fault("INVENTORY_CONTRACT_UNSUPPORTED") from None
        else:
            match=re.match(r"Found (\d+) container\(s\):",raw)
            if match is None:
                raise Fault("INVENTORY_CONTRACT_UNSUPPORTED")
            names=re.findall(r"(?m)^   Username: ([a-z][a-z0-9-]{0,62})\s*$",raw)
            if len(names)!=int(match[1]):
                raise Fault("INVENTORY_CONTRACT_UNSUPPORTED")
        if len(set(names))!=len(names) or any(not isinstance(n,str) or re.fullmatch(r"[a-z][a-z0-9-]{0,62}",n) is None for n in names):
            raise Fault("INVENTORY_CONTRACT_UNSUPPORTED")
        return set(names)

    def lifecycle_inspect(self, logical):
        if logical not in self.lifecycle_inventory():
            return None
        with self.session() as m:
            m.discover()
            raw=m.tool("get_container",{"username":logical})
        try:
            c=json.loads(text(raw))["container"]
            if c["username"]!=logical:
                raise ValueError()
            target=Target(self.config["provider"],self.config["backend"],self.config["project"],
                          logical,c["name"],str(c["createdAt"]),c["username"],c["name"])
            state=c["state"]
            if not isinstance(state,str):
                raise ValueError()
            return {"target":target,"state":state,"podman_enabled":c.get("podmanEnabled") if type(c.get("podmanEnabled")) is bool else None}
        except (KeyError,ValueError,TypeError):
            raise Fault("TARGET_FORMAT_UNSUPPORTED") from None

    def lifecycle_mutate(self, action, logical, spec, timeout):
        if timeout<=0:
            raise Fault("DEADLINE_EXCEEDED")
        tool={"create":"create_container","start":"start_container","stop":"stop_container","delete":"delete_container"}.get(action)
        if tool is None:
            raise Fault("ACTION_NOT_ALLOWED")
        with MCP(self.argv,min(1800,timeout)) as m:
            tools=m.discover()
            props=tools.get(tool,{}).get("inputSchema",{}).get("properties",{})
            if "username" not in props:
                raise Fault("CONTRACT_UNSUPPORTED")
            args={"username":logical}
            if action=="create":
                required={"cpu","memory","disk","image","ssh_keys","enable_podman"}
                if not required.issubset(props):
                    raise Fault("CONTRACT_UNSUPPORTED")
                args.update(cpu=spec["cpu"],memory=spec["memory"],disk=spec["disk"],
                            image="images:ubuntu/24.04",enable_podman=False,ssh_keys=[spec["ssh_public_key"]])
            elif action=="start":
                if "waitForReady" in props:
                    args["waitForReady"]=False
            else:
                if "force" not in props:
                    raise Fault("CONTRACT_UNSUPPORTED")
                args["force"]=False
            m.tool(tool,args)

    def lifecycle_identity(self, target, timeout):
        with self.session(timeout) as m:
            tools=m.discover()
            props=tools.get("connect",{}).get("inputSchema",{}).get("properties",{})
            key="box" if "box" in props else "username" if "username" in props else None
            if key is None or "exec" not in props:
                raise Fault("CONTRACT_UNSUPPORTED")
            code,out=execution(m.tool("connect",{key:target.logical,"exec":"whoami && hostname"}))
            if code!=0 or out.splitlines()!=[target.user,target.hostname]:
                raise Fault("IDENTITY_MISMATCH")
            return {"user":target.user,"hostname":target.hostname,"stdout":out,"exit_code":code}
