"""Explicitly configured, exact-Box MCP adapter; never selects SSH from a request.

No agent-box launcher is installed, generated, or discovered automatically. The
operator must configure a trusted pinned-SSH launcher for EACH exact Box and
confirm AGENTBOX_ROOT within that launcher. Missing configuration fails closed.
"""
import os
from pathlib import Path, PurePosixPath
import re
import stat
from .domain import Fault, workspace
from .mcp import MCP
from .provider import text

READ_TOOLS = frozenset(("read_file", "list_directory", "tail_log", "process_list",
                        "compose_discover", "compose_autostart_status"))
WRITE_TOOLS = frozenset(("write_file", "move_file", "delete_file", "shell_exec",
                         "process_start", "process_kill", "compose_autostart_enable",
                         "compose_autostart_disable"))
RESOURCE_URIS = frozenset(("containarium://ci-context", "containarium://ci-prompt"))
PATH_KEYS = {"read_file":("path",), "list_directory":("path",),
             "write_file":("path",), "move_file":("source","destination")}

class BoxAgent:
    def __init__(self, service, logical):
        self.service = service
        if getattr(service.provider,"mode",None)!="mcp":
            raise Fault("AGENT_BOX_LIVE_PROVIDER_REQUIRED")
        self.target = service.observe_target(logical)
        config = getattr(service.provider,"config",{})
        entry = config.get("agent_box", {}).get(logical)
        if not isinstance(entry,dict) or set(entry)!={"target_digest","workspace",
                                                        "launcher","allowed_tools"}:
            raise Fault("AGENT_BOX_NOT_CONFIGURED",404)
        if entry["target_digest"]!=self.target.digest():
            raise Fault("TARGET_CHANGED")
        self.root=workspace(entry["workspace"],self.target.user)
        launcher=entry["launcher"]
        if (not isinstance(launcher,list) or not 1<=len(launcher)<=12 or
                any(not isinstance(a,str) or not a or len(a)>1024 for a in launcher)):
            raise Fault("AGENT_BOX_LAUNCHER_INVALID",422)
        path=Path(launcher[0])
        if not path.is_absolute() or path.is_symlink():
            raise Fault("AGENT_BOX_LAUNCHER_INVALID",422)
        try:
            info=path.stat()
        except OSError:
            raise Fault("AGENT_BOX_NOT_INSTALLED",404) from None
        if (not stat.S_ISREG(info.st_mode) or info.st_uid not in (0,os.geteuid())
                or info.st_mode & 0o022):
            raise Fault("AGENT_BOX_LAUNCHER_UNSAFE",422)
        allowed=entry["allowed_tools"]
        if (not isinstance(allowed,list) or not all(isinstance(x,str) for x in allowed)
                or len(set(allowed))!=len(allowed) or
                any(x not in READ_TOOLS|WRITE_TOOLS for x in allowed)):
            raise Fault("AGENT_BOX_POLICY_INVALID",422)
        self.allowed=frozenset(allowed)
        self.launcher=launcher

    def _path(self, value):
        if (not isinstance(value,str) or not 1<=len(value)<=1024 or
                "\x00" in value or any(ord(c)<32 for c in value)):
            raise Fault("BOX_PATH_DENIED",422)
        raw=PurePosixPath(value)
        root=PurePosixPath(self.root)
        if not raw.is_absolute() or ".." in raw.parts or not raw.is_relative_to(root):
            raise Fault("BOX_PATH_DENIED",403)
        return str(raw)

    def validate(self, name, args):
        if name not in self.allowed:
            raise Fault("AGENT_BOX_TOOL_DENIED",403)
        if not isinstance(args,dict) or len(args)>12:
            raise Fault("INVALID_INPUT",422)
        if name in PATH_KEYS:
            for key in PATH_KEYS[name]:
                if key not in args:raise Fault("BOX_PATH_DENIED",422)
                self._path(args[key])
        if name in ("shell_exec","process_start"):
            # Execution remains limited by Box OS permissions, NOT file roots.
            command=args.get("command")
            if not isinstance(command,str) or not 1<=len(command)<=4096:
                raise Fault("BOX_COMMAND_DENIED",422)
            if "cwd" not in args or self._path(args["cwd"])!=self.root:
                raise Fault("BOX_COMMAND_DENIED",403)
        if name=="write_file":
            if not isinstance(args.get("content"),str) or len(args["content"].encode())>16384:
                raise Fault("BOX_WRITE_LIMIT",422)
        if name in ("delete_file","process_kill","compose_autostart_disable"):
            # All destructive calls are dispatched only through the plan API.
            pass
        return args

    def _open(self, timeout=20):
        return MCP(self.launcher,min(30,max(1,timeout)))

    def capabilities(self):
        with self._open() as m:
            tools=m.discover()
            try:
                resources=m.list_resources()
            except Fault:
                resources=[]  # protocol/resource support is separately measured
        return {"logical":self.target.logical,"target_digest":self.target.digest(),
                "tools":[{"name":n,"available":n in tools,"authorized":n in self.allowed}
                         for n in sorted(READ_TOOLS|WRITE_TOOLS)],
                "resources":[r["uri"] for r in resources
                             if r["uri"] in RESOURCE_URIS],
                "source":"agent_box","execution":"not_tested"}

    def read(self, name, args):
        if name not in READ_TOOLS:raise Fault("AGENT_BOX_READ_ONLY",403)
        self.validate(name,args)
        self.service.observe_target(self.target.logical)  # revalidate target
        with self._open() as m:
            discovered=m.discover()
            if name not in discovered:raise Fault("AGENT_BOX_TOOL_MISSING",409)
            raw=text(m.tool(name,args))
        if len(raw.encode())>16384:raise Fault("OUTPUT_LIMIT")
        return {"logical":self.target.logical,"tool":name,
                "content":raw,"source":"agent_box"}

    def resource(self, uri):
        if uri not in RESOURCE_URIS:raise Fault("RESOURCE_URI_NOT_ALLOWED",403)
        self.service.observe_target(self.target.logical)
        with self._open() as m:
            m.discover()
            if not any(r["uri"]==uri for r in m.list_resources()):
                raise Fault("AGENT_BOX_RESOURCE_MISSING",404)
            content=m.read_resource(uri)
        return {"logical":self.target.logical,"uri":uri,"content":content}

    def execute(self, name, args, timeout=30):
        """Internal dispatch only: caller must hold a durable confirmed plan."""
        if name not in WRITE_TOOLS:raise Fault("AGENT_BOX_WRITE_ONLY",403)
        self.validate(name,args)
        if self.service.observe_target(self.target.logical).digest()!=self.target.digest():
            raise Fault("TARGET_CHANGED")
        with self._open(timeout) as m:
            discovered=m.discover()
            if name not in discovered:raise Fault("AGENT_BOX_TOOL_MISSING",409)
            raw=text(m.tool(name,args))
        if len(raw.encode())>4096:raise Fault("OUTPUT_LIMIT")
        return {"logical":self.target.logical,"tool":name,"summary":"upstream_completed",
                "output_bytes":len(raw.encode()),"source":"agent_box"}
