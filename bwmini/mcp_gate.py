"""MCP stdio relay: inject a server-owned workload key and enforce SSH trust."""
import base64, json, os, re, stat, subprocess, sys, time, secrets, hashlib
from pathlib import Path

try:
    from .nesting import invoke as nesting_invoke
except ImportError:
    from nesting import invoke as nesting_invoke

try:
    from . import gate_diagnostics as gd
    from . import box_reads, box_patch, gateway_tasks, coordination_bridge, gateway_protection, mcp_compat, restore_guard, resource_size, debug_rest, gateway_rest, gateway_create, rest_mutations
    from .operation_coordination import CoordinationFault
except ImportError:
    import gate_diagnostics as gd
    import box_reads
    import box_patch
    import gateway_tasks
    import coordination_bridge
    import gateway_protection
    import mcp_compat
    import restore_guard
    import resource_size
    import debug_rest
    import gateway_rest
    import gateway_create
    import rest_mutations
    from operation_coordination import CoordinationFault

# Source-backed, read-only MCP self-information; tools/list is runtime truth.
SERVER_INFO_VERSION = "0.50.3"
SERVER_DESCRIPTION = (
    "BoxWeave is a self-hosted environment control plane for AI coding agents. "
    "Use isolated Linux Boxes to execute code, build and test projects, manage "
    "services, and diagnose workloads. Tool availability depends on the "
    "Containarium backend, authenticated scope and operator tool policy."
)
SERVER_INSTRUCTIONS = (
    "Use the read-only boxweave_info tool for architecture and developer "
    "guidance when using or developing BoxWeave. Use tools/list for currently "
    "authorized tools, then list_containers before creating a Box. For "
    "ordinary user work consider an existing suitable Box; for testing "
    "BoxWeave itself, prefer a new disposable Box after local checks and "
    "use historical Boxes only with evidence of environment dependence. "
    "Do not assume SCP/upload or host shell from an SSH/HTTP MCP connection. "
    "Execute builds/tests with connect(exec) when permitted, and verify "
    "exit codes and outputs. The BoxWeave gateway "
    "manages workload SSH public keys; never request or expose private keys. "
    "Diagnose SSH failures with debug_container. Obtain approval for destructive "
    "or network exposure changes. Timed-out writes have UNKNOWN outcomes: "
    "inspect real state before retrying, never blindly replay mutations. "
    "Use boxweave_diagnostic_run for read-only app preflight, REST/MCP "
    "readiness and actor-scoped operation traces; it cannot execute host "
    "shell commands or modify Boxes. "
    "On errors, use boxweave_diagnose to inspect classified evidence; use "
    "boxweave_project_diagnose only within the server-authorized Box scope. "
    "Use boxweave_diagnose(source=compat) to read a narrow, on-demand "
    "Provider MCP schema check when compatibility is suspected. "
    "A diagnosis is not a verified cause or permission to repair. For ordinary "
    "users explain/prepare a private issue draft; for authorized developers "
    "investigate source and tests. Never upload reports or open an external "
    "issue before the user personally reviews its contents and separately "
    "approves final submission."
)
INFO_TOPICS = {
    "overview": {
        "purpose": "Self-hosted control plane for AI-agent development, persistent isolated Linux environments, testing, and application management.",
        "components": ["BoxWeave Python application and Web UI",
                       "SSH/stdio MCP gateway", "optional authenticated HTTP MCP",
                       "Containarium Provider", "Incus/LXC Boxes"],
        "limitation": "This guide is descriptive, not a live runtime, version or permission probe.",
    },
    "architecture": {
        "direct_agent": "Authorized MCP client -> BoxWeave gateway -> Containarium Provider -> Box. Gateway manages workload public SSH key and selected tool rewrites.",
        "web": "Authenticated local Web -> BoxWeave application service and typed Provider adapter -> separately configured upstream MCP launcher and action receipts.",
        "remote": "Authenticated opt-in Streamable HTTP MCP -> loopback listener behind independently configured TLS/tunnel -> same gateway.",
        "exec": "connect(exec) executes inside a managed Box, not in the Web application host.",
        "source": "https://github.com/vanishs/boxweave",
    },
    "development": {
        "steps": [
            "Read AGENTS.md, docs/DEVELOPMENT.md, docs/process/AGENT_TEST_WORKFLOW.md and docs/process/AGENT_TEST_DEPLOYMENT.md at the intended source revision.",
            "Run focused local checks first; for real BoxWeave deployment tests default to a new authorized disposable Box; use a historical Box only when the old environment is demonstrably required.",
            "Prefer exact local modified source; verify bundle hash and transport ability independently of SSH/HTTP MCP, and do not publish uncommitted code to a public GitHub branch.",
            "Run focused unit tests, then python scripts/check_knowledge.py and python scripts/validate.py on a prepared checkout.",
            "Browser fixtures: python scripts/validate.py --browser; fixtures cannot prove live Provider, SSH or network behavior.",
            "Record commit, environment, commands, exit codes, failures and independently verified behavior.",
            "Only publish source-bound artifacts after CI; promote the stable installer pin separately after release verification.",
        ],
    },
    "capabilities": {
        "runtime_discovery": "Call tools/list for the actual authorized tool names and input schemas. For current state use get_system_info, list_backends, list_containers, get_container, get_metrics.",
        "examples": ["list_containers", "create_container", "connect", "debug_container",
                     "start_container", "stop_container", "delete_container",
                     "get_metrics", "boxweave_nesting"],
        "limits": "Provider tools vary by version/scope. Application-side agent-box support is separate from direct Agent tools; no guarantee of general snapshot/clone or automatic Agent test orchestration.",
    },
    "diagnostics": {
        "gateway": "boxweave_diagnose reads bounded gateway and app evidence; gateway capture defaults to 7 days/50 MiB and is configurable by the host operator.",
        "projects": "boxweave_project_diagnose is read-only and defaults deny until root-owned exact Box incarnation and check allowlist is installed.",
        "agent_workflow": "On failure or unknown result diagnose first, never blindly retry writes. For suspected tool-schema drift use boxweave_diagnose(source=compat), an on-demand read-only subset check.",
        "restore": "Whole-Box restore through this Agent gateway is default-denied; the host operator preapproves exact source backup / target / local private key alias. No arbitrary key paths or overwrite."
    },
    "security": {
        "authority": "Single trusted-operator MCP authority, optionally restricted by a root-owned host-wide tool allowlist; not per-Agent/per-Box RBAC.",
        "ssh": "Gateway injects its own managed workload Ed25519 public key. No private/client transport keys should be passed. connect checks registered target identity and rejects conflicting host/user overrides.",
        "mutations": "Require authorization for destructive, trust or externally exposed changes; timeouts imply UNKNOWN outcome and must not trigger blind retry.",
        "http": "Remote HTTP MCP is optional, authenticated and loopback-only; HTTPS/tunnel exposure is configured separately.",
    },
    "documentation": {
        "repository": "https://github.com/vanishs/boxweave",
        "entrypoints": ["README.md", "AGENTS.md", "docs/DEVELOPMENT.md",
                        "docs/process/AGENT_TEST_WORKFLOW.md",
                         "docs/process/AGENT_TEST_DEPLOYMENT.md", "docs/REMOTE_MCP.md",
                        "docs/verification/OPEN_RISKS.md",
                        "docs/verification/REMEDIATION_TEST_QUEUE.md"],
        "rule": "Use documentation matching deployed version; old proposals/source inventories are not runtime evidence.",
    },
}
INFO_TOOL = {
    "name": "boxweave_info",
    "description": (
        "Read-only BoxWeave self-description for coding agents working with or "
        "developing BoxWeave: architecture, developer workflow, capabilities, "
        "security and documentation. Optional topic selects a section; omit "
        "for all. No host state or secrets. Use tools/list and Provider read "
        "tools for actual live capabilities and current environments."
    ),
    "inputSchema": {"type": "object", "additionalProperties": False,
                    "properties": {"topic": {"type": "string",
                         "enum": ["all", *INFO_TOPICS],
                         "description": "Section to read, defaults to all."}},
                    "required": []},
    "annotations": {"readOnlyHint": True, "destructiveHint": False,
                    "idempotentHint": True, "openWorldHint": False},
}

def boxweave_info(arguments):
    if not isinstance(arguments, dict) or set(arguments) - {"topic"}:
        raise ValueError("INFO_TOPIC_INVALID")
    topic = arguments.get("topic", "all")
    if not isinstance(topic, str) or topic not in ("all", *INFO_TOPICS):
        raise ValueError("INFO_TOPIC_INVALID")
    return {"name": "boxweave", "guide_version": 1,
            "source_version": SERVER_INFO_VERSION,
            "scope": "static source guidance, not live runtime evidence",
            "sections": INFO_TOPICS if topic == "all" else {topic: INFO_TOPICS[topic]}}

NESTING_TOOL = {
    "name": "boxweave_nesting",
    "description": "Inspect or preview/confirm one managed box nesting setting. Runs on its parent host; never restarts. Capability remains unverified until child MCP execution.",
    "inputSchema": {"type": "object", "additionalProperties": False,
        "properties": {"action": {"type":"string","enum":["inspect","plan","apply"]},
                       "logical": {"type":"string"}, "nesting":{"type":"boolean"},
                       "expected":{"type":"object"}, "confirm":{"type":"string"}},
        "required":["action","logical"]}}

PATCH_TOOL = {
    "name": "boxweave_box_patch",
    "description": "Protected-mode tracked compare-and-swap replacement of <=1024 bytes in a trusted Box file; request_key deduplicates dispatch. Return includes metadata-only task receipt; read file for the observed content. UNKNOWN cannot be retried.",
    "inputSchema": {"type":"object","additionalProperties":False,
        "properties":{"box":{"type":"string"},"path":{"type":"string"},
                      "expected_sha256":{"type":"string"},
                      "new_text":{"type":"string"},"request_key":{"type":"string"},
                      "lease_token":{"type":"string"}},
        "required":["box","path","expected_sha256","new_text","request_key"]},
}

LEASE_TOOL = {
    "name": "boxweave_box_lease",
    "description": "Optional protected-mode exact Box lease (acquire/renew/release/status). Ten-minute maximum; not an authentication token. UNKNOWN dispatched claims outlive expiry.",
    "inputSchema": {"type":"object","additionalProperties":False,
        "properties":{"action":{"type":"string","enum":["acquire","renew","release","status"]},
                      "box":{"type":"string"},"token":{"type":"string"},
                      "seconds":{"type":"integer","minimum":1,"maximum":600}},
        "required":["action","box"]},
}

def guard_restore_catalog(message):
    """Rewrite or omit the privileged upstream tool on EVERY catalog page."""
    result = message.get("result")
    if not isinstance(result, dict) or not isinstance(result.get("tools"), list):
        return message
    reviewed = []
    for tool in result["tools"]:
        if isinstance(tool, dict) and tool.get("name") == "restore_box_backup":
            adapted = restore_guard.catalog_tool(tool)
            if adapted is not None:
                reviewed.append(adapted)
        else:
            reviewed.append(tool)
    result["tools"] = reviewed
    return message


def catalog(message):
    if "result" not in message:
        return message
    message=guard_restore_catalog(message)
    tools=message["result"].get("tools")
    if not isinstance(tools,list):
        raise ValueError("UPSTREAM_CATALOG_INVALID")
    rest={t["name"]:t for t in gateway_rest.rest_routes.mcp_tool_catalog()}
    # Source-controlled REST schemas, not forwarded MCP tool definitions.
    rest["create_container"]["description"]=(
        "Create an isolated development Box using one durable REST intent. "
        "BoxWeave automatically supplies its managed workload SSH public key; extra "
        "ssh_keys are optional. A CREATING response is not guest-ready. "
        "If a result is UNKNOWN, inspect the original task without retrying.")
    rest["create_container"]["inputSchema"]["properties"]["ssh_keys"]["description"]=(
        "Optional additional authorized Ed25519 public keys. The server automatically supplies "
        "its own managed workload SSH public key.")
    rest["list_containers"]["description"]=(
        "Read-only REST inventory of existing Boxes and their resources.")
    rest["delete_container"]["description"]=(
        "Permanently delete an exactly identified Box using one tracked REST "
        "intent; do not retry an UNKNOWN outcome.")
    chosen=set()
    converted=[]
    for tool in tools:
        if not isinstance(tool,dict):
            continue
        name=tool.get("name")
        if name in rest:
            if name not in chosen:
                converted.append(rest[name])
                chosen.add(name)
            continue
        if name=="connect":
            tool["description"]=(
                "Connect to a managed Box by exact-target logical name or displayed "
                "name with pinned SSH trust. Exec remains MCP and may mutate "
                "the guest. Host and user overrides are refused.")
        converted.append(tool)
    # A missing upstream MCP descriptor cannot hide an implemented REST route.
    converted.extend(tool for name,tool in rest.items() if name not in chosen)
    extras=(LEASE_TOOL,PATCH_TOOL) if gateway_protection.enabled() else ()
    converted.extend((NESTING_TOOL,INFO_TOOL,gd.DIAG_TOOL,gd.PROJECT_TOOL,
                      gd.COORD_TOOL,*box_reads.TOOLS,*gateway_tasks.TOOLS,*extras))
    message["result"]["tools"]=converted
    return message

REGISTRY = Path("/var/lib/boxweave-default/public/targets.json")
LAUNCHER = "/usr/local/lib/boxweave/boxweave/launcher"

# Fixed, non-disclosing failure codes that may be surfaced verbatim; anything
# else stays behind the generic trust-gate message.
GATE_ERROR_CODES = frozenset((
    "MANAGED_IDENTITY_UNAVAILABLE", "SSH_KEYS_ARRAY_REQUIRED",
    "DUPLICATE_SSH_PUBLIC_KEY", "FULL_ED25519_PUBLIC_KEY_REQUIRED",
    "TOO_MANY_PUBLIC_KEYS", "TARGET_CHANGED", "TARGET_READ_FAILED",
    "EXACT_MANAGED_TARGET_REQUIRED", "MANAGED_INSTANCE_IDENTITY_REQUIRED",
    "LOCAL_INCUS_OBSERVATION_OR_CHANGE_FAILED", "NESTING_BOOLEAN_REQUIRED",
    "NESTING_CONFIG_INVALID", "NESTING_REQUEST_INVALID", "REQUEST_LIMIT",
    "LOCAL_UBUNTU_NESTING_ONLY",
    "OWNED_PROVIDER_REQUIRED", "NESTING_HELPER_UNAVAILABLE",
    "NESTING_HELPER_FAILED_INSPECT_ORIGINAL_INSTANCE",
    "STALE_PLAN_OR_EXACT_CONFIRMATION_REQUIRED", "INFO_TOPIC_INVALID",
    "CHANGE_OUTCOME_UNVERIFIED_INSPECT_ORIGINAL_INSTANCE",
    "DIAGNOSTIC_SELECTOR_INVALID", "DIAGNOSTIC_SETTINGS_INVALID",
    "PROJECT_SELECTOR_INVALID", "PROJECT_POLICY_INVALID",
    "PROJECT_DIAG_NOT_AUTHORIZED", "PROJECT_TARGET_CHANGED",
    "COORDINATOR_SELECTOR_INVALID",
    "DIAGNOSTIC_POLICY_UNSAFE", "DIAGNOSTIC_STORAGE_UNSAFE",
    "INVALID_ARGUMENT", "NOT_FOUND", "ALREADY_EXISTS", "FAILED_PRECONDITION",
    "RESOURCE_EXHAUSTED", "UNAVAILABLE", "DEADLINE_EXCEEDED", "UNKNOWN",
    "PERMISSION_DENIED", "UNAUTHENTICATED", "ABORTED", "INTERNAL",
    "REST_APP_CONFIG_UNVERIFIED", "REST_APP_COORDINATION_NOT_ALIGNED",
    "REST_STATE_DB_UNSAFE", "REST_STATE_DB_UNAVAILABLE",
    "RESTORE_APPROVAL_REQUIRED", "RESTORE_KEY_OR_POLICY_UNSAFE", "RESTORE_ARGUMENT_INVALID",
    "PROJECT_COMMAND_UNAVAILABLE",
    "TRUST_REGISTRY_UNAVAILABLE", "TRUST_OBSERVATION_STALE",
    "TRUST_NOT_READY", "BOX_TARGET_AMBIGUOUS", "TARGET_OVERRIDE_REFUSED", "RESOURCE_CPU_INVALID",
    "RESOURCE_DISK_SIZE_UNVERIFIED", "RESOURCE_DISK_SHRINK_REFUSED",
    "TASK_ARGUMENT_INVALID", "TASK_STORAGE_UNSAFE", "TASK_TRACKING_UNAVAILABLE",
    "TASK_SCHEMA_UNSUPPORTED", "TASK_NOT_FOUND", "TASK_CAPACITY_EXCEEDED",
    "TASK_SELECTOR_INVALID", "TASK_STATE_CONFLICT", "TASK_RESULT_UNVERIFIED",
    "REQUEST_KEY_CONFLICT", "GATE_TOOL_NOT_ALLOWED",
    "COORDINATION_SCOPE_UNVERIFIED", "COORDINATION_SCOPE_MISMATCH",
    "COORDINATION_IDENTITY_INVALID", "COORDINATION_UNAVAILABLE",
    "COORDINATION_SCHEMA_UNSUPPORTED", "COORDINATION_STORAGE_UNSAFE",
    "RESOURCE_BUSY", "RECOVERY_REQUIRED",
    "COORDINATION_POLICY_UNAVAILABLE", "COORDINATION_POLICY_INVALID",
    "COORDINATION_POLICY_UNSAFE", "COORDINATION_TOOL_UNCLASSIFIED",
    "COORDINATION_USE_TRACKED_TASK", "COORDINATION_LEASE_DISABLED",
    "LEASE_ARGUMENT_INVALID", "LEASE_TOKEN_INVALID", "LEASE_EXPIRED",
    "COORDINATION_PATCH_DISABLED", "BOX_PATCH_INVALID", "BOX_PATCH_TOO_LARGE",
    "DEBUG_REST_TARGET_INVALID", "DEBUG_REST_ENDPOINT_INVALID",
    "DEBUG_REST_TOKEN_PATH_INVALID", "DEBUG_REST_TOKEN_UNAVAILABLE",
    "DEBUG_REST_TOKEN_UNSAFE", "DEBUG_REST_RESPONSE_INVALID",
    "DEBUG_REST_RESPONSE_OVERSIZED", "DEBUG_REST_HTTP_REJECTED",
    "DEBUG_REST_AUTH_REJECTED", "DEBUG_REST_ROUTE_UNVERIFIED",
    "DEBUG_REST_UNAVAILABLE", "DEBUG_REST_CONFIG_INVALID",
    "DEBUG_REST_POLICY_UNAVAILABLE", "DEBUG_REST_POLICY_UNSAFE",
    "DEBUG_REST_POLICY_INVALID",
    "BOX_READ_INVALID", "BOX_PATH_INVALID", "BOX_READ_UNVERIFIED",
))


MANAGED_PUBLIC_KEY_FILE = Path("/home/boxweave/.ssh/id_ed25519.pub")


def normalized_public_key(value):
    if not isinstance(value, str) or len(value) > 512 or not re.fullmatch(
        r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}(?: [A-Za-z0-9@._-]{1,80})?", value
    ):
        raise ValueError("FULL_ED25519_PUBLIC_KEY_REQUIRED")
    try:
        blob = base64.b64decode(value.split()[1], validate=True)
    except ValueError:
        raise ValueError("FULL_ED25519_PUBLIC_KEY_REQUIRED") from None
    if len(blob) != 51 or blob[:19] != b"\x00\x00\x00\x0bssh-ed25519\x00\x00\x00\x20":
        raise ValueError("FULL_ED25519_PUBLIC_KEY_REQUIRED")
    return " ".join(value.split()[:2])


def managed_public_key():
    # Read only the public half of this MCP server's *own* identity. Never
    # substitute the client's transport key or an identity from another layer.
    path = MANAGED_PUBLIC_KEY_FILE
    if path.is_symlink():
        raise ValueError("MANAGED_IDENTITY_UNAVAILABLE")
    try:
        info = path.stat()
        if (not stat.S_ISREG(info.st_mode) or info.st_uid != os.geteuid()
            or info.st_mode & 0o022 or info.st_nlink != 1
            or info.st_size > 512):
            raise ValueError("MANAGED_IDENTITY_UNAVAILABLE")
        return normalized_public_key(path.read_text(encoding="ascii").strip())
    except (OSError, UnicodeError, ValueError):
        raise ValueError("MANAGED_IDENTITY_UNAVAILABLE") from None


def managed_create_arguments(args):
    """Always give the Provider the verified server-owned public key.

    This prevents the upstream missing-key private-key fallback, without
    requiring Codex AGENTS guidance or exposing the server's private key.
    """
    if not isinstance(args, dict):
        raise ValueError("ARGUMENTS_OBJECT_REQUIRED")
    keys = args.get("ssh_keys", [])
    if not isinstance(keys, list) or len(keys) > 16:
        raise ValueError("SSH_KEYS_ARRAY_REQUIRED")
    normalized = [normalized_public_key(key) for key in keys]
    if len(normalized) != len(set(normalized)):
        raise ValueError("DUPLICATE_SSH_PUBLIC_KEY")
    host_key = managed_public_key()
    if host_key not in normalized:
        if len(normalized) >= 16:
            raise ValueError("TOO_MANY_PUBLIC_KEYS")
        normalized.append(host_key)
    return {**args, "ssh_keys": normalized}


def trusted_registry():
    """Fail closed with bounded deployment-readiness codes, never raw key data."""
    try:
        value = json.loads(REGISTRY.read_text())
        if not isinstance(value, dict):
            raise ValueError("TRUST_REGISTRY_UNAVAILABLE")
        return value
    except (OSError, ValueError, UnicodeError):
        raise ValueError("TRUST_REGISTRY_UNAVAILABLE") from None


def resolve_managed_box(box, registry):
    """Resolve only exact trusted display-name aliases, never guessed targets."""
    if not isinstance(box, str) or re.fullmatch(r"[a-z][a-z0-9-]{0,62}", box) is None:
        raise ValueError("EXACT_MANAGED_TARGET_REQUIRED")
    targets = registry.get("targets")
    if not isinstance(targets, dict):
        raise ValueError("TRUST_REGISTRY_UNAVAILABLE")
    direct = targets.get(box)
    alias = None
    if box.endswith("-container"):
        logical = box[:-len("-container")]
        candidate = targets.get(logical)
        if (isinstance(candidate, dict) and candidate.get("name") == box
                and candidate.get("user") == logical):
            alias = logical
    if direct is not None and alias is not None and alias != box:
        raise ValueError("BOX_TARGET_AMBIGUOUS")
    if direct is not None:
        if not isinstance(direct, dict):
            raise ValueError("TRUST_REGISTRY_UNAVAILABLE")
        return box
    if alias is not None:
        return alias
    raise ValueError("TRUST_NOT_READY")


def validated_resize_arguments(args):
    """Reject malformed CPU syntax before any remote Provider mutation.

    Preserve CPU counts, decimal/millicpu quotas, and Incus CPU sets.
    The Provider remains authoritative for capacity and host constraints.
    """
    if not isinstance(args, dict):
        raise ValueError("RESOURCE_CPU_INVALID")
    cpu = args.get("cpu")
    if cpu is None or cpu == "":
        return args
    if not isinstance(cpu, str) or len(cpu) > 64:
        raise ValueError("RESOURCE_CPU_INVALID")
    value = cpu.strip()
    numeric = re.fullmatch(r"(?:[0-9]+(?:\.[0-9]+)?|\.[0-9]+|[0-9]+m)", value)
    cpuset = re.fullmatch(r"[0-9]+(?:-[0-9]+)?(?:,[0-9]+(?:-[0-9]+)?)*", value)
    if not numeric and not cpuset:
        raise ValueError("RESOURCE_CPU_INVALID")
    if numeric:
        count = value[:-1] if value.endswith("m") else value
        try:
            if not (0 < float(count) < 1000000):
                raise ValueError("RESOURCE_CPU_INVALID")
        except (ValueError, OverflowError):
            raise ValueError("RESOURCE_CPU_INVALID") from None
    if cpuset and ("-" in value or "," in value):
        for part in value.split(","):
            if "-" in part:
                lo, hi = map(int, part.split("-"))
                if hi < lo:
                    raise ValueError("RESOURCE_CPU_INVALID")
    return args


def trusted(args, registry, container):
    observed_at = registry.get("observed_at")
    if type(observed_at) not in (int, float):
        raise ValueError("TRUST_REGISTRY_UNAVAILABLE")
    if not 0 <= time.time() - observed_at <= 45:
        raise ValueError("TRUST_OBSERVATION_STALE")
    targets = registry.get("targets")
    if not isinstance(targets, dict):
        raise ValueError("TRUST_REGISTRY_UNAVAILABLE")
    target = targets.get(args.get("box"))
    if not isinstance(target, dict) or not target.get("keys"):
        raise ValueError("TRUST_NOT_READY")
    if (container.get("name") != target.get("name")
        or str(container.get("createdAt")) != target.get("created")
        or container.get("username") != target.get("user")
        or not isinstance(container.get("network"), dict)
        or container["network"].get("ipAddress") != target.get("ip")
        or container.get("state") != "CONTAINER_STATE_RUNNING"):
        raise ValueError("TARGET_CHANGED")
    if (args.get("host", target.get("ip")) != target.get("ip")
            or args.get("user", target.get("user")) != target.get("user")):
        raise ValueError("TARGET_OVERRIDE_REFUSED")

POLICY = Path("/etc/boxweave/mcp-tool-policy.json")

def tool_policy(path=POLICY):
    """Host-wide optional allowlist, never a grant or per-user RBAC."""
    try:
        metadata = path.lstat()
    except FileNotFoundError:
        if path == POLICY: return None  # Deliberate legacy compatibility.
        raise ValueError("POLICY_NOT_INSTALLED") from None
    folder = path.parent.lstat()
    if (not stat.S_ISDIR(folder.st_mode) or folder.st_uid != 0 or
        folder.st_mode & 0o022 or not stat.S_ISREG(metadata.st_mode) or
        metadata.st_uid != 0 or metadata.st_mode & 0o077 or
        metadata.st_nlink != 1 or metadata.st_size > 16384):
        raise ValueError("POLICY_UNSAFE")
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, ValueError, UnicodeError):
        raise ValueError("POLICY_INVALID") from None
    if (not isinstance(value, dict) or set(value) != {"version", "allowed_tools"} or
        type(value["version"]) is not int or value["version"] != 1 or
        not isinstance(value["allowed_tools"], list) or len(value["allowed_tools"]) > 256 or
        any(not isinstance(n, str) or not re.fullmatch(r"[a-z][a-z0-9_]{0,127}", n)
            for n in value["allowed_tools"]) or
        len(set(value["allowed_tools"])) != len(value["allowed_tools"])):
        raise ValueError("POLICY_INVALID")
    return frozenset(value["allowed_tools"])

def filter_tools(message, allowed):
    if allowed is None or "result" not in message: return message
    result = message["result"]
    if not isinstance(result, dict) or not isinstance(result.get("tools"), list):
        raise ValueError("UPSTREAM_CATALOG_INVALID")
    result["tools"] = [t for t in result["tools"] if
                       isinstance(t, dict) and t.get("name") in allowed]
    return message

def main():
    permitted = tool_policy()
    process = subprocess.Popen([LAUNCHER], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
    def send(message):
        process.stdin.write((json.dumps(message) + "\n").encode()); process.stdin.flush()
    def reply(ident, forward=True):
        for _ in range(256):
            line = process.stdout.readline(2_097_153)
            if not line or len(line) > 2_097_152: raise ValueError("UPSTREAM_PROTOCOL_LIMIT")
            message = json.loads(line)
            if message.get("id") == ident and ("result" in message or "error" in message):
                if forward: print(json.dumps(message), flush=True)
                return message
            print(json.dumps(message), flush=True)
        raise ValueError("UPSTREAM_MESSAGE_LIMIT")
    def emit_result(ident, value):
        print(json.dumps({"jsonrpc": "2.0", "id": ident, "result": {
            "content": [{"type": "text", "text": json.dumps(value, sort_keys=True)}],
            "isError": False}}), flush=True)

    def container(logical, ident):
        # Native exact-target, trust and nesting preflights must use the
        # migrated REST read. Never tunnel get_container via upstream MCP.
        config=gateway_rest.policy()
        return gateway_rest.execute_read(
            config, "get_container", {"username": logical})["container"]
    upstream_mcp_version = "not_verified"
    try:
        for line in sys.stdin:
            if len(line) > 65536: raise ValueError("REQUEST_LIMIT")
            request = json.loads(line)
            if request.get("method") == "tools/list":
                send(request)
                message = reply(request["id"], False)
                if not request.get("params", {}).get("cursor"):
                    message = catalog(message)
                else:
                    message = guard_restore_catalog(message)
                message = filter_tools(message, permitted)
                try:
                    try:
                        selected_rest = gateway_rest.policy()
                    except gateway_rest.GatewayRESTFault:
                        # Missing REST credentials must not disable discovery
                        # of unrelated specialist MCP tools. All migrated
                        # calls still fail closed when actually invoked.
                        selected_rest = None
                    selected_debug = selected_rest is not None
                    protected = gateway_protection.enabled()
                    coordinated_rest = (selected_rest is not None and
                                        selected_rest["mode"] == "coordinated")
                    if protected:
                        message = gateway_protection.filter_catalog(
                            message, rest_debug=selected_debug,
                            coordinated_rest=coordinated_rest)
                    if selected_rest is not None:
                        # A selected REST capability never falls back to MCP
                        # when coordination is absent or disabled.
                        if not coordinated_rest:
                            message["result"]["tools"] = [
                                tool for tool in message["result"]["tools"]
                                if tool.get("name") not in gateway_rest.WRITES]
                        else:
                            for tool in message["result"]["tools"]:
                                if tool.get("name") in (
                                        "start_container", "stop_container",
                                        "delete_container", "resize_container"):
                                    tool["inputSchema"].setdefault(
                                        "properties", {})["request_key"] = {
                                        "type": "string", "pattern": "^[0-9a-f]{32}$",
                                        "description": "Optional durable one-shot intent. "
                                        "Use a new key only for a deliberate new operation."}
                            for tool in (rest_mutations.CREATE_STATUS_TOOL,
                                         rest_mutations.STATUS_TOOL):
                                if permitted is None or tool["name"] in permitted:
                                    if not any(t.get("name") == tool["name"]
                                               for t in message["result"]["tools"]):
                                        message["result"]["tools"].append(dict(tool))
                except (debug_rest.DebugRESTError, gateway_rest.GatewayRESTFault) as error:
                    # A malformed root policy cannot silently downgrade or
                    # widen the protected forwarded MCP catalog.
                    print(json.dumps({"jsonrpc":"2.0","id":request.get("id"),
                        "error":{"code":-32602,"message":str(error)}}), flush=True)
                    continue
                print(json.dumps(message), flush=True)
                continue
            started = time.monotonic()
            trace = secrets.token_hex(16)
            if request.get("method") == "tools/call":
                params = request.get("params", {}); args = params.get("arguments", {})
                if permitted is not None and params.get("name") not in permitted:
                    print(json.dumps({"jsonrpc":"2.0","id":request.get("id"),
                        "error":{"code":-32601,"message":"GATE_TOOL_NOT_ALLOWED"}}), flush=True)
                    continue
                try:
                    name=params.get("name")
                    migrated = name in gateway_rest.ROUTED
                    rest_state_tool = name in (
                        "boxweave_rest_status", "boxweave_create_status")
                    # Unmigrated MCP tools remain usable independently of
                    # the REST app binding. Only migrated tools may read the
                    # binding, and may never fall back to MCP if it is absent.
                    rest_config = (gateway_rest.policy()
                                   if migrated or rest_state_tool else None)
                    protected = gateway_protection.enabled()
                    coordinated_rest = (rest_config is not None and
                                        rest_config["mode"] == "coordinated")
                    if protected:
                        gateway_protection.check_call(
                            params.get("name"), args,
                            rest_debug=rest_config is not None,
                            coordinated_rest=coordinated_rest)
                    if rest_config is not None and params.get("name") == "create_container":
                        receipt = gateway_create.dispatch(
                            rest_config, args, request.get("id"),
                            normalize=managed_create_arguments, protected=True)
                        emit_result(request["id"], receipt)
                        gd.record("create_container", {
                            "result": {"content": [], "isError": False}},
                            (time.monotonic()-started)*1000, trace=trace)
                        continue
                    if rest_config is not None and params.get("name") in (
                            "start_container", "stop_container",
                            "delete_container", "resize_container"):
                        if params["name"] == "resize_container":
                            validated_resize_arguments(args)
                        receipt = rest_mutations.dispatch(
                            rest_config, params["name"], args,
                            protected=True)
                        emit_result(request["id"], receipt)
                        gd.record(params["name"],
                                  {"result":{"content":[],"isError":False}},
                                  (time.monotonic()-started)*1000, trace=trace)
                        continue
                    if params.get("name") in (
                            "boxweave_rest_status", "boxweave_create_status"):
                        if rest_config is None or not coordinated_rest:
                            raise gateway_rest.GatewayRESTFault("REST_TASK_UNAVAILABLE")
                        if not isinstance(args,dict):
                            raise gateway_rest.GatewayRESTFault("REST_TASK_SELECTOR_INVALID")
                        if params["name"] == "boxweave_rest_status":
                            if set(args)!={"task_id"}:
                                raise gateway_rest.GatewayRESTFault("REST_TASK_SELECTOR_INVALID")
                            value=rest_mutations.MutationJournal(
                                gateway_create.AppStore(rest_config["state_db"])
                            ).get(task_id=args["task_id"])
                        else:
                            if not args or set(args)-{
                                    "task_id","request_key","username"}:
                                raise gateway_rest.GatewayRESTFault("CREATE_SELECTOR_INVALID")
                            value=gateway_create.journal(rest_config).get(**args)
                        emit_result(request["id"], value)
                        continue
                    if rest_config is not None and params.get("name") in (
                            "list_containers", "get_container", "debug_container"):
                        value = gateway_rest.execute_read(
                            rest_config, params["name"], args)
                        emit_result(request["id"], value)
                        gd.record(params["name"],
                                  {"result": {"content": [], "isError": False}},
                                  (time.monotonic()-started)*1000, trace=trace)
                        continue
                    if params.get("name") == "boxweave_diagnose":
                        if not isinstance(args, dict):
                            raise ValueError("DIAGNOSTIC_SELECTOR_INVALID")
                        if args.get("source") == "gateway":
                            value = gd.query_gateway(args)
                        elif args.get("source") == "app":
                            value = gd.query_app(args)
                        elif args == {"source": "compat"}:
                            try:
                                snapshot = mcp_compat.read_tools(send, reply)
                                value = mcp_compat.analyze(snapshot, allowed=permitted)
                                value["upstream_version"] = upstream_mcp_version
                                value["version_source"] = (
                                    "initialize.serverInfo.version"
                                    if upstream_mcp_version != "not_verified"
                                    else "not_verified")
                            except (ValueError, KeyError, TypeError, OSError):
                                value = mcp_compat.unavailable()
                        else:
                            raise ValueError("DIAGNOSTIC_SELECTOR_INVALID")
                        emit_result(request["id"], value)
                        continue
                    if params.get("name") == "boxweave_diagnostic_run":
                        value = gd.query_coordinator(args)
                        emit_result(request["id"], value)
                        continue
                    if params.get("name") == "boxweave_project_diagnose":
                        # Root-owned policy is checked BEFORE any Provider read.
                        gd.preauthorize_project(args)
                        observed = container(args["box"], request["id"])
                        gd.project_grant(args, observed)
                        registry = trusted_registry()
                        trusted({"box": args["box"]}, registry, observed)
                        internal = "boxweave-d4-" + secrets.token_hex(8)
                        send({"jsonrpc": "2.0", "id": internal, "method": "tools/call",
                              "params": {"name": "connect",
                                         "arguments": {"box": args["box"],
                                                       "exec": gd.command_for_check(args)}}})
                        response = reply(internal, False)
                        value = gd.project_summary(args, response.get("result", {}))
                        value.update(source="project", box=args["box"])
                        emit_result(request["id"], value)
                        continue
                    if params.get("name") == "restore_box_backup":
                        # An Agent cannot supply an arbitrary host file path.
                        # The root-approved exact backup/target/key must match.
                        # No upstream call takes place until the preflight passes.
                        params["arguments"] = restore_guard.approve(args)
                    if params.get("name") == "boxweave_info":
                        value = boxweave_info(args)
                        print(json.dumps({"jsonrpc":"2.0","id":request["id"],
                            "result":{"content":[{"type":"text","text":json.dumps(value)}],
                                      "isError":False}}), flush=True)
                        continue
                    if params.get("name") in ("boxweave_box_read_file", "boxweave_box_list_directory"):
                        # This is a narrowly reviewed, read-only façade over the
                        # existing connect adapter. Never grant authority beyond
                        # the original connect permission or trusted Box registry.
                        if permitted is not None and "connect" not in permitted:
                            raise ValueError("GATE_TOOL_NOT_ALLOWED")
                        verified = box_reads.verified_args(params["name"], args)
                        registry = trusted_registry()
                        logical = resolve_managed_box(verified["box"], registry)
                        observed = container(logical, request["id"])
                        trusted({"box": logical}, registry, observed)
                        verified["box"] = logical
                        internal = "boxweave-box-read-" + secrets.token_hex(8)
                        send({"jsonrpc": "2.0", "id": internal, "method": "tools/call",
                              "params": {"name": "connect",
                                         "arguments": {"box": verified["box"],
                                                       "exec": box_reads.guest_command(verified)}}})
                        outcome = reply(internal, False)
                        emit_result(request["id"], box_reads.typed_output(outcome))
                        continue
                    if params.get("name") in ("boxweave_task_get", "boxweave_task_list"):
                        if not isinstance(args, dict):
                            raise gateway_tasks.TaskFault("TASK_ARGUMENT_INVALID")
                        store = gateway_tasks.TaskStore()
                        if params["name"] == "boxweave_task_get":
                            if set(args) - {"task_id", "request_key"}:
                                raise gateway_tasks.TaskFault("TASK_SELECTOR_INVALID")
                            value = store.get(**args)
                        else:
                            if set(args) - {"box", "limit"} or "box" not in args:
                                raise gateway_tasks.TaskFault("TASK_ARGUMENT_INVALID")
                            value = {"tasks": store.list(args["box"], args.get("limit", 20))}
                        emit_result(request["id"], value)
                        continue
                    if params.get("name") == "boxweave_box_lease":
                        # Only the reviewed protected gateway route may expose
                        # leases; they coordinate trusted operators, never grant auth.
                        if not gateway_protection.enabled():
                            raise ValueError("COORDINATION_LEASE_DISABLED")
                        if permitted is not None and "connect" not in permitted:
                            raise ValueError("GATE_TOOL_NOT_ALLOWED")
                        if not isinstance(args,dict) or set(args)-{"box","action","token","seconds"}:
                            raise ValueError("LEASE_ARGUMENT_INVALID")
                        action=args.get("action")
                        if action not in ("acquire","renew","release","status"):
                            raise ValueError("LEASE_ARGUMENT_INVALID")
                        if action in ("acquire","status") and "token" in args:
                            raise ValueError("LEASE_ARGUMENT_INVALID")
                        if action in ("renew","release") and "token" not in args:
                            raise ValueError("LEASE_TOKEN_INVALID")
                        if action!="acquire" and "seconds" in args and action!="renew":
                            raise ValueError("LEASE_ARGUMENT_INVALID")
                        observed=container(args.get("box"),request["id"])
                        registry=trusted_registry()
                        trusted({"box":args.get("box")},registry,observed)
                        identity, _, _=coordination_bridge.gateway_box(
                            observed,registry.get("scope"),args["box"])
                        value=coordination_bridge.SharedClaims().lease(
                            action,identity,token=args.get("token"),
                            seconds=args.get("seconds",600))
                        emit_result(request["id"],value)
                        continue
                    if params.get("name") in ("boxweave_task_submit","boxweave_box_patch"):
                        # Preserve the original connect permission and target
                        # validation; request_key is NOT an authorization grant.
                        if permitted is not None and "connect" not in permitted:
                            raise ValueError("GATE_TOOL_NOT_ALLOWED")
                        is_patch=params["name"]=="boxweave_box_patch"
                        if is_patch:
                            if not gateway_protection.enabled():
                                raise ValueError("COORDINATION_PATCH_DISABLED")
                            if not isinstance(args,dict):
                                raise ValueError("BOX_PATCH_INVALID")
                            patch_input={k:v for k,v in args.items() if k!="lease_token"}
                            checked=box_patch.verify_patch(patch_input)
                            command=box_patch.guest_command(checked)
                        else:
                            if (not isinstance(args,dict)
                                    or not {"box","request_key","exec"}<=set(args)
                                    or set(args)-{"box","request_key","exec","lease_token"}):
                                raise gateway_tasks.TaskFault("TASK_ARGUMENT_INVALID")
                            command=args["exec"]
                        if "lease_token" in args and (
                                not gateway_protection.enabled()
                                or not isinstance(args["lease_token"],str)):
                            raise gateway_tasks.TaskFault("TASK_ARGUMENT_INVALID")
                        payload_digest=gateway_tasks.command_digest(args["box"],command)
                        registry = trusted_registry()
                        observed = container(args["box"], request["id"])
                        trusted({"box": args["box"]}, registry, observed)
                        identity, scope_id, logical = coordination_bridge.gateway_box(
                            observed, registry.get("scope"), args["box"])
                        store = gateway_tasks.TaskStore()
                        receipt, fresh = store.reserve(args["request_key"], logical,
                                                       identity, payload_digest)
                        if not fresh:
                            emit_result(request["id"], receipt)
                            continue
                        claims = coordination_bridge.SharedClaims()
                        ref = "gateway_" + receipt["task_id"]
                        try:
                            # This atomic claim must succeed before any dispatch.
                            # A Box lease affects BOTH root-published gateways
                            # and the app's identical canonical resource keys.
                            claims.acquire(receipt["task_id"], ref, identity, scope_id, logical,
                                           lease_token=args.get("lease_token"))
                        except CoordinationFault:
                            store.advance(receipt["task_id"], "recorded",
                                          "failed_before_dispatch")
                            raise
                        try:
                            store.advance(receipt["task_id"], "recorded", "dispatch_intent")
                            claims.dispatch_intent(receipt["task_id"])
                        except Exception:
                            # No remote write was sent. If the durable receipt
                            # already says dispatch_intent, it stays UNKNOWN.
                            try:
                                claims.cancel_before_dispatch(receipt["task_id"])
                            except CoordinationFault:
                                pass
                            raise
                        try:
                            internal = "boxweave-task-" + secrets.token_hex(8)
                            send({"jsonrpc": "2.0", "id": internal, "method": "tools/call",
                                  "params": {"name": "connect",
                                             "arguments": {"box": logical, "exec": command}}})
                            result = reply(internal, False)
                            code = gateway_tasks.observed_exit(result)
                            if is_patch and code==0:
                                verified_patch=box_patch.parse_receipt(result)
                                if (verified_patch["sha256"] != hashlib.sha256(
                                        args["new_text"].encode("utf-8")).hexdigest() or
                                        verified_patch["bytes"] != len(args["new_text"].encode("utf-8"))):
                                    raise ValueError("BOX_PATCH_RESULT_UNVERIFIED")
                            # Commit original receipt BEFORE releasing the
                            # separately stored resource claim. A failed release
                            # leaves a conservative claim, never a duplicate run.
                            store.advance(receipt["task_id"], "dispatch_intent", "completed",
                                          exit_code=code)
                            try:
                                claims.observed_terminal(receipt["task_id"], ref)
                            except CoordinationFault:
                                pass
                        except Exception:
                            # Transport loss is UNKNOWN even if the command
                            # might have been sent. Never replay or clear lock.
                            try:
                                claims.unknown(receipt["task_id"])
                            except CoordinationFault:
                                pass
                            try:
                                store.advance(receipt["task_id"], "dispatch_intent", "unknown")
                            except gateway_tasks.TaskFault:
                                pass
                        emit_result(request["id"], store.get(task_id=receipt["task_id"]))
                        continue
                    if params.get("name") == "boxweave_nesting":
                        observed = container(args.get("logical"), request["id"])
                        action = args.get("action")
                        # Require current credential-authorized visibility before root helper access.
                        if observed.get("username") != args.get("logical"): raise ValueError("TARGET_CHANGED")
                        if action == "apply":
                            expected = args.get("expected", {})
                            if (not isinstance(expected, dict) or expected.get("name") != observed.get("name")
                                    or expected.get("created") != str(observed.get("createdAt"))):
                                raise ValueError("TARGET_CHANGED")
                        value = nesting_invoke(args)
                        target = value["target"]
                        if target["name"] != observed["name"] or target["created"] != str(observed["createdAt"]):
                            raise ValueError("TARGET_CHANGED")
                        print(json.dumps({"jsonrpc":"2.0","id":request["id"],"result":{
                            "content":[{"type":"text","text":json.dumps(value)}]}}), flush=True)
                        continue
                    if params.get("name") == "connect":
                        if not isinstance(args, dict):
                            raise ValueError("EXACT_MANAGED_TARGET_REQUIRED")
                        registry = trusted_registry()
                        logical = resolve_managed_box(args.get("box"), registry)
                        observed = container(logical, request["id"])
                        trusted({**args, "box": logical}, registry, observed)
                        # Forward only the verified logical username.
                        params["arguments"] = {**args, "box": logical}
                except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError, gateway_tasks.TaskFault, gateway_create.CreateFault, CoordinationFault) as error:
                    # Never disclose actual public/private key data in errors.
                    reason = str(error) if str(error) in GATE_ERROR_CODES else "TRUST_OR_WORKLOAD_KEY_GATE: inspect the exact target and server-managed identity"
                    error_result = {"jsonrpc": "2.0", "id": request.get("id"),
                                    "error": {"code": -32602, "message": reason}}
                    if params.get("name") not in ("boxweave_diagnose", "boxweave_project_diagnose", "boxweave_diagnostic_run"):
                        # Only the gateway's own reviewed read/preflight
                        # refusals are definitely FAILED. A Provider error,
                        # dispatched write, or unreviewed exception stays
                        # UNKNOWN and is never automatically replayed.
                        gd.record(params.get("name", ""), error_result,
                                  (time.monotonic() - started) * 1000,
                                  trace=trace, local_denial=reason)
                    print(json.dumps(error_result), flush=True)
                    continue
            # Only genuinely unmigrated upstream MCP functions reach this
            # relay. All eight migrated methods have returned above.
            send(request)
            if "id" in request:
                if request.get("method") == "initialize":
                    message = reply(request["id"], False)
                    result = message.get("result")
                    if isinstance(result, dict):
                        original = result.get("serverInfo")
                        version = original.get("version") if isinstance(original, dict) else None
                        if isinstance(version, str) and re.fullmatch(
                                r"v?[0-9]+\.[0-9]+\.[0-9]+(?:[-+][A-Za-z0-9.-]+)?",
                                version) and len(version) <= 64:
                            upstream_mcp_version = version
                        info = result.setdefault("serverInfo", {})
                        info.update({"name":"boxweave","version":SERVER_INFO_VERSION,
                                     "description":SERVER_DESCRIPTION})
                        result["instructions"] = SERVER_INSTRUCTIONS
                    print(json.dumps(message), flush=True)
                else:
                    try:
                        message = reply(request["id"], False)
                    except (ValueError, OSError):
                        if request.get("method") == "tools/call":
                            gd.record(request.get("params", {}).get("name", ""),
                                      {"error": {"code": -32603}},
                                      (time.monotonic() - started) * 1000, trace=trace)
                        raise
                    if request.get("method") == "tools/call":
                        gd.record(request.get("params", {}).get("name", ""), message,
                                  (time.monotonic() - started) * 1000, trace=trace)
                    print(json.dumps(message), flush=True)
    finally:
        if process.poll() is None:
            process.terminate()
            try: process.wait(timeout=3)
            except subprocess.TimeoutExpired: process.kill()
        process.stdin.close(); process.stdout.close()


if __name__ == "__main__": main()
