"""A fixture is explicit. Missing live credentials never enable a fixture."""
import json
import pathlib
import re
import shlex
from .domain import Target, Fault
from .mcp import MCP
from .wire_results import text, execution  # preserve provider.py public imports
from .access_contract import validate_config as validate_access_config
from .lifecycle_contract import validate_config as validate_lifecycle_config

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
        validate_access_config(config.get("access"))
        argv = config.get("launcher")
        if not isinstance(argv, list) or not argv or any(not isinstance(a, str) for a in argv):
            raise Fault("MCP_LAUNCHER_REQUIRED", 422)
        if not pathlib.Path(argv[0]).is_absolute() or not pathlib.Path(argv[0]).is_file():
            raise Fault("MCP_LAUNCHER_UNAVAILABLE", 422)
        self.argv = argv
        # Only REST handles migrated reads/writes. MCP remains for guest
        # identity/exec and unmigrated specialist capabilities.
        self.rest = None
        if "provider_api" in config:
            from .provider_rest import RESTClient
            self.rest = RESTClient(config["provider_api"])
        # Debug-only rollback must not disable existing REST lifecycle.
        choice = config.get("debug_rest_enabled", True)
        if type(choice) is not bool:
            raise Fault("DEBUG_REST_CONFIG_INVALID", 422)
        self.debug_rest_enabled = self.rest is not None and choice
        self.lifecycle_config = config.get("lifecycle")
        validate_lifecycle_config(self.lifecycle_config, set(config.get("targets", {})))

    def session(self, timeout=30):
        return MCP(self.argv, min(30, max(1, timeout)))

    def environment_report(self, logical):
        """E0/E1 exact-target read-only report; no snapshots or guest shell."""
        from .environment_observe import EnvironmentInspector
        return EnvironmentInspector(self).report(logical)

    def observe(self, logical):
        expected = self.config["targets"].get(logical)
        if expected is None:
            raise Fault("TARGET_NOT_ALLOWED", 403)
        if self.rest is None:
            raise Fault("PROVIDER_API_REQUIRED", 422)
        c = self.rest.container(logical)
        try:
            target = Target(self.config["provider"], self.config["backend"],
                            self.config["project"], logical, c["name"],
                            str(c["createdAt"]), c["username"], c["name"])
        except (KeyError, TypeError, ValueError):
            raise Fault("TARGET_FORMAT_UNSUPPORTED") from None
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

    def nesting_check(self, wanted):
        if type(wanted) is not bool:
            raise Fault("NESTING_BOOLEAN_REQUIRED", 422)
        if self.rest is None:
            raise Fault("PROVIDER_API_REQUIRED", 422)
        # Native nesting follows independently attested REST provisioning.

    def lifecycle_inventory(self):
        if self.rest is None:
            raise Fault("PROVIDER_API_REQUIRED", 422)
        return self.rest.inventory()

    def lifecycle_inspect(self, logical):
        if logical not in self.lifecycle_inventory():
            return None
        if self.rest is None:
            raise Fault("PROVIDER_API_REQUIRED", 422)
        c = self.rest.container(logical)
        try:
            if c["username"]!=logical:
                raise ValueError()
            target=Target(self.config["provider"],self.config["backend"],self.config["project"],
                          logical,c["name"],str(c["createdAt"]),c["username"],c["name"])
            state=c["state"]
            if not isinstance(state,str):
                raise ValueError()
            resources=c.get("resources")
            return {"target":target,"state":state,"network_tenant":c.get("networkTenant"),
                    "membership_source":c.get("networkTenantSource"),
                    "account_user":c["username"],
                    "podman_enabled":c.get("podmanEnabled") if type(c.get("podmanEnabled")) is bool else None,
                    "resources":resources if isinstance(resources,dict) else None}
        except (KeyError,ValueError,TypeError):
            raise Fault("TARGET_FORMAT_UNSUPPORTED") from None

    def lifecycle_mutate(self, action, logical, spec, timeout):
        if timeout<=0:
            raise Fault("DEADLINE_EXCEEDED")
        if self.rest is None:
            raise Fault("PROVIDER_API_REQUIRED", 422)
        return self.rest.mutate(action,logical,spec,timeout)

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
