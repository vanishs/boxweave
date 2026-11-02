"""MCP stdio relay: inject a server-owned workload key and enforce SSH trust."""
import base64, json, os, re, stat, subprocess, sys, time, secrets, hashlib
from pathlib import Path

try:
    from .nesting import invoke as nesting_invoke
except ImportError:
    from nesting import invoke as nesting_invoke

try:
    from . import gate_diagnostics as gd
    from . import box_reads, gateway_tasks
except ImportError:
    import gate_diagnostics as gd
    import box_reads
    import gateway_tasks

# Source-backed, read-only MCP self-information; tools/list is runtime truth.
SERVER_INFO_VERSION = "0.32.0"
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
    "On errors, use boxweave_diagnose to inspect classified evidence; use "
    "boxweave_project_diagnose only within the server-authorized Box scope. "
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
        "agent_workflow": "On failure or unknown result diagnose first, never blindly retry writes. For disclosure draft privately, user reviews sensitive data, then separately approves final external submission."
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

def catalog(message):
    if "result" not in message: return message
    for tool in message["result"].get("tools", []):
        if tool.get("name") == "create_container":
            tool["description"] = (
                "Create an isolated Linux development Box for coding and testing. "
                "BoxWeave injects its own managed workload SSH public key; "
                "ssh_keys is optional and accepts only additional authorized Ed25519 "
                "public keys. Never supply client transport keys, fingerprints or "
                "private keys. Check existing Boxes first. After creation use "
                "connect(exec) for commands. If creation times out, check state "
                "before retry; diagnose connection failures with debug_container."
            )
            schema = tool["inputSchema"]
            schema["required"] = [field for field in schema.get("required", []) if field != "ssh_keys"]
            schema["properties"].setdefault("ssh_keys", {})["description"] = (
                "Optional additional authorized full Ed25519 public keys. The "
                "BoxWeave MCP host automatically supplies its own managed workload "
                "public key. Do not pass client transport keys, fingerprints or private keys.")
            tool["inputSchema"]["properties"]["enable_nesting"] = {
                "type":"boolean",
                "description":"Independent nesting setting applied after successful creation, without installing Podman or restarting. Omit to retain upstream behavior."}
        elif tool.get("name") == "connect":
            tool["description"] = (
                "Connect to a managed Box with BoxWeave exact-target SSH trust. "
                "Optional exec runs one command inside the Box and returns "
                "stdout/stderr/exit_code; omit exec for human SSH instructions. "
                "Commands may mutate the Box. Host/user overrides inconsistent "
                "with the registered target are rejected."
            )
        elif tool.get("name") == "list_containers":
            tool["description"] = (
                "Read-only list of existing Boxes, state, IP and resources. "
                "Check before creating a development/test Box."
            )
        elif tool.get("name") == "delete_container":
            tool["description"] = (
                "Permanently delete a Box and its data. Destructive: obtain "
                "explicit authorization and verify target; after a timeout "
                "inspect state instead of blindly retrying."
            )
    message["result"].setdefault("tools", []).extend((NESTING_TOOL, INFO_TOOL, gd.DIAG_TOOL, gd.PROJECT_TOOL, *box_reads.TOOLS, *gateway_tasks.TOOLS))
    return message

def nesting_plan(logical, wanted, observed):
    plan = nesting_invoke({"action":"plan","logical":logical,"nesting":wanted})
    target = plan["target"]
    if target["name"] != observed["name"] or target["created"] != str(observed["createdAt"]):
        raise ValueError("TARGET_CHANGED")
    return plan

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
    "DIAGNOSTIC_POLICY_UNSAFE", "DIAGNOSTIC_STORAGE_UNSAFE",
    "PROJECT_COMMAND_UNAVAILABLE",
    "TRUST_REGISTRY_UNAVAILABLE", "TRUST_OBSERVATION_STALE",
    "TRUST_NOT_READY", "TARGET_OVERRIDE_REFUSED", "RESOURCE_CPU_INVALID",
    "TASK_ARGUMENT_INVALID", "TASK_STORAGE_UNSAFE", "TASK_TRACKING_UNAVAILABLE",
    "TASK_SCHEMA_UNSUPPORTED", "TASK_NOT_FOUND", "TASK_CAPACITY_EXCEEDED",
    "TASK_SELECTOR_INVALID", "TASK_STATE_CONFLICT", "TASK_RESULT_UNVERIFIED",
    "REQUEST_KEY_CONFLICT", "GATE_TOOL_NOT_ALLOWED",
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
    if not 0 <= time.time() - registry.get("observed_at", 0) <= 45: raise ValueError("TRUST_OBSERVATION_STALE")
    target = registry.get("targets", {}).get(args.get("box"))
    if not target or not target.get("keys"): raise ValueError("TRUST_NOT_READY")
    if (container.get("name") != target["name"] or str(container.get("createdAt")) != target["created"]
        or container.get("username") != target["user"]
        or container.get("network", {}).get("ipAddress") != target["ip"]
        or container.get("state") != "CONTAINER_STATE_RUNNING"):
        raise ValueError("TARGET_CHANGED")
    if args.get("host", target["ip"]) != target["ip"] or args.get("user", target["user"]) != target["user"]:
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
        internal = "boxweave-nesting-" + str(ident)
        send({"jsonrpc":"2.0","id":internal,"method":"tools/call",
              "params":{"name":"get_container","arguments":{"username":logical}}})
        result = reply(internal, False)["result"]
        if result.get("isError"): raise ValueError("TARGET_READ_FAILED")
        return json.loads("".join(b["text"] for b in result["content"] if b.get("type")=="text"))["container"]
    try:
        for line in sys.stdin:
            if len(line) > 65536: raise ValueError("REQUEST_LIMIT")
            request = json.loads(line)
            if request.get("method") == "tools/list":
                send(request)
                message = reply(request["id"], False)
                if not request.get("params", {}).get("cursor"): message = catalog(message)
                message = filter_tools(message, permitted)
                print(json.dumps(message), flush=True)
                continue
            nesting = None
            started = time.monotonic()
            trace = secrets.token_hex(16)
            if request.get("method") == "tools/call":
                params = request.get("params", {}); args = params.get("arguments", {})
                if permitted is not None and params.get("name") not in permitted:
                    print(json.dumps({"jsonrpc":"2.0","id":request.get("id"),
                        "error":{"code":-32601,"message":"GATE_TOOL_NOT_ALLOWED"}}), flush=True)
                    continue
                try:
                    if params.get("name") == "boxweave_diagnose":
                        if not isinstance(args, dict):
                            raise ValueError("DIAGNOSTIC_SELECTOR_INVALID")
                        if args.get("source") == "gateway":
                            value = gd.query_gateway(args)
                        elif args.get("source") == "app":
                            value = gd.query_app(args)
                        else:
                            raise ValueError("DIAGNOSTIC_SELECTOR_INVALID")
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
                        observed = container(verified["box"], request["id"])
                        trusted({"box": verified["box"]}, trusted_registry(), observed)
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
                    if params.get("name") == "boxweave_task_submit":
                        # Preserve the original connect permission and target
                        # validation; request_key is NOT an authorization grant.
                        if permitted is not None and "connect" not in permitted:
                            raise ValueError("GATE_TOOL_NOT_ALLOWED")
                        if not isinstance(args, dict) or set(args) != {"box", "request_key", "exec"}:
                            raise gateway_tasks.TaskFault("TASK_ARGUMENT_INVALID")
                        payload_digest = gateway_tasks.command_digest(args["box"], args["exec"])
                        observed = container(args["box"], request["id"])
                        trusted({"box": args["box"]}, trusted_registry(), observed)
                        # Bind to exact observed incarnation. Do not print or
                        # persist commands or private connection material.
                        identity = hashlib.sha256(json.dumps({
                            "name": observed["name"],
                            "created": str(observed["createdAt"]),
                            "username": observed["username"],
                            "ip": observed["network"]["ipAddress"]
                        }, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
                        store = gateway_tasks.TaskStore()
                        receipt, fresh = store.reserve(args["request_key"], args["box"],
                                                       identity, payload_digest)
                        if not fresh:
                            emit_result(request["id"], receipt)
                            continue
                        # A failed durable intent prevents ANY upstream dispatch.
                        store.advance(receipt["task_id"], "recorded", "dispatch_intent")
                        try:
                            internal = "boxweave-task-" + secrets.token_hex(8)
                            send({"jsonrpc": "2.0", "id": internal, "method": "tools/call",
                                  "params": {"name": "connect",
                                             "arguments": {"box": args["box"], "exec": args["exec"]}}})
                            result = reply(internal, False)
                            code = gateway_tasks.observed_exit(result)
                            store.advance(receipt["task_id"], "dispatch_intent", "completed",
                                          exit_code=code)
                        except Exception:
                            # Once dispatch_intent is committed, no transport
                            # failure proves the guest command did not run.
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
                    if params.get("name") == "create_container":
                        args = managed_create_arguments(args)
                        params["arguments"] = args
                        if "enable_nesting" in args:
                            if (any(args.get(k) for k in ("backend_id","pool","region"))
                                    or args.get("isolation") not in (None,"","container")
                                    or args.get("os_type") not in (None,"","ubuntu")):
                                raise ValueError("LOCAL_UBUNTU_NESTING_ONLY")
                            nesting = args.pop("enable_nesting")
                            if type(nesting) is not bool: raise ValueError("NESTING_BOOLEAN_REQUIRED")
                    if params.get("name") == "resize_container":
                        validated_resize_arguments(args)
                    if params.get("name") == "connect":
                        registry = trusted_registry()
                        internal = "boxweave-trust-" + str(request["id"])
                        send({"jsonrpc": "2.0", "id": internal, "method": "tools/call",
                              "params": {"name": "get_container", "arguments": {"username": args.get("box")}}})
                        observed = reply(internal, False)
                        result = observed["result"]
                        if result.get("isError"): raise ValueError("TARGET_READ_FAILED")
                        text = "".join(b["text"] for b in result["content"] if b.get("type") == "text")
                        trusted(args, registry, json.loads(text)["container"])
                except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError, gateway_tasks.TaskFault) as error:
                    # Never disclose actual public/private key data in errors.
                    reason = str(error) if str(error) in GATE_ERROR_CODES else "TRUST_OR_WORKLOAD_KEY_GATE: inspect the exact target and server-managed identity"
                    error_result = {"jsonrpc": "2.0", "id": request.get("id"),
                                    "error": {"code": -32602, "message": reason}}
                    if params.get("name") not in ("boxweave_diagnose", "boxweave_project_diagnose"):
                        gd.record(params.get("name", ""), error_result,
                                  (time.monotonic() - started) * 1000, trace=trace)
                    print(json.dumps(error_result), flush=True)
                    continue
            send(request)
            if "id" in request:
                if nesting is None and request.get("method") == "initialize":
                    message = reply(request["id"], False)
                    result = message.get("result")
                    if isinstance(result, dict):
                        info = result.setdefault("serverInfo", {})
                        info.update({"name":"boxweave","version":SERVER_INFO_VERSION,
                                     "description":SERVER_DESCRIPTION})
                        result["instructions"] = SERVER_INSTRUCTIONS
                    print(json.dumps(message), flush=True)
                elif nesting is None:
                    try:
                        message = reply(request["id"], False)
                    except (ValueError, OSError):
                        # Record an uncertain upstream response; never replay
                        # the operation or replace its original failure.
                        if request.get("method") == "tools/call":
                            gd.record(request.get("params", {}).get("name", ""),
                                      {"error": {"code": -32603}},
                                      (time.monotonic() - started) * 1000, trace=trace)
                        raise
                    if request.get("method") == "tools/call":
                        gd.record(request.get("params", {}).get("name", ""), message,
                                  (time.monotonic() - started) * 1000, trace=trace)
                    print(json.dumps(message), flush=True)
                else:
                    message = reply(request["id"], False)
                    if "error" not in message and not message.get("result", {}).get("isError"):
                        try:
                            logical = args["username"]
                            plan = nesting_plan(logical, nesting, container(logical, request["id"]))
                            value = nesting_invoke({"action":"apply","logical":logical,"nesting":nesting,
                                                    "expected":plan["expected"],"confirm":plan["target"]["name"]})
                            message["result"]["content"].append({"type":"text","text":json.dumps({"boxweave_nesting":value})})
                        except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError) as error:
                            detail = str(error) if str(error) in GATE_ERROR_CODES else "NESTING_CONFIGURATION_UNVERIFIED_INSPECT_ORIGINAL_INSTANCE"
                            # Preserve the upstream *successful creation* receipt.
                            # A JSON-RPC error hides that a real Box exists and can
                            # prompt a client to retry a non-idempotent create.
                            # The post-create setting may have applied even if its
                            # verification failed: report UNKNOWN, never "false".
                            partial = {"state": "partial", "box_created": True,
                                       "requested_nesting": nesting,
                                       "nesting_outcome": "not_verified",
                                       "code": detail, "username": args["username"],
                                       "next_action": "Inspect the original Box; do not repeat create_container."}
                            result = message["result"]
                            result["content"] = [{"type": "text",
                                                  "text": json.dumps({"boxweave_nesting": partial})}
                                                 ] + result.get("content", [])
                            result["isError"] = True
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
