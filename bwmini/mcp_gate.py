"""MCP stdio relay: inject a server-owned workload key and enforce SSH trust."""
import base64, json, os, re, stat, subprocess, sys, time
from pathlib import Path

try:
    from .nesting import invoke as nesting_invoke
except ImportError:
    from nesting import invoke as nesting_invoke

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
            schema = tool["inputSchema"]
            schema["required"] = [field for field in schema.get("required", []) if field != "ssh_keys"]
            schema["properties"].setdefault("ssh_keys", {})["description"] = (
                "Optional additional authorized full Ed25519 public keys. The "
                "BoxWeave MCP host automatically supplies its own managed workload "
                "public key. Do not pass client transport keys, fingerprints or private keys.")
            tool["inputSchema"]["properties"]["enable_nesting"] = {
                "type":"boolean",
                "description":"Independent nesting setting applied after successful creation, without installing Podman or restarting. Omit to retain upstream behavior."}
    message["result"].setdefault("tools", []).append(NESTING_TOOL)
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
    "OWNED_PROVIDER_REQUIRED", "NESTING_HELPER_UNAVAILABLE",
    "NESTING_HELPER_FAILED_INSPECT_ORIGINAL_INSTANCE",
    "STALE_PLAN_OR_EXACT_CONFIRMATION_REQUIRED",
    "CHANGE_OUTCOME_UNVERIFIED_INSPECT_ORIGINAL_INSTANCE",
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
            if request.get("method") == "tools/call":
                params = request.get("params", {}); args = params.get("arguments", {})
                if permitted is not None and params.get("name") not in permitted:
                    print(json.dumps({"jsonrpc":"2.0","id":request.get("id"),
                        "error":{"code":-32601,"message":"GATE_TOOL_NOT_ALLOWED"}}), flush=True)
                    continue
                try:
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
                    if params.get("name") == "connect":
                        registry = json.loads(REGISTRY.read_text())
                        internal = "boxweave-trust-" + str(request["id"])
                        send({"jsonrpc": "2.0", "id": internal, "method": "tools/call",
                              "params": {"name": "get_container", "arguments": {"username": args.get("box")}}})
                        observed = reply(internal, False)
                        result = observed["result"]
                        if result.get("isError"): raise ValueError("TARGET_READ_FAILED")
                        text = "".join(b["text"] for b in result["content"] if b.get("type") == "text")
                        trusted(args, registry, json.loads(text)["container"])
                except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError) as error:
                    # Never disclose actual public/private key data in errors.
                    reason = str(error) if str(error) in GATE_ERROR_CODES else "TRUST_OR_WORKLOAD_KEY_GATE: inspect the exact target and server-managed identity"
                    print(json.dumps({"jsonrpc": "2.0", "id": request.get("id"),
                                      "error": {"code": -32602, "message": reason}}), flush=True)
                    continue
            send(request)
            if "id" in request:
                if nesting is None:
                    reply(request["id"])
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
                            message = {"jsonrpc":"2.0","id":request["id"],"error":{
                                "code":-32603,"message":"BOXWEAVE_NESTING_NOT_APPLIED; box creation itself succeeded and must not be repeated; "+detail}}
                    print(json.dumps(message), flush=True)
    finally:
        if process.poll() is None:
            process.terminate()
            try: process.wait(timeout=3)
            except subprocess.TimeoutExpired: process.kill()
        process.stdin.close(); process.stdout.close()


if __name__ == "__main__": main()
