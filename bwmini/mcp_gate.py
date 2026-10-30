"""MCP stdio relay: require explicit creation keys and Incus-authenticated SSH trust."""
import base64, json, re, subprocess, sys, time
from pathlib import Path

REGISTRY = Path("/var/lib/boxweave-default/public/targets.json")
LAUNCHER = "/usr/local/lib/boxweave/boxweave/launcher"


def explicit_keys(args):
    if not isinstance(args, dict): raise ValueError("ARGUMENTS_OBJECT_REQUIRED")
    keys = args.get("ssh_keys")
    if not isinstance(keys, list) or not keys: raise ValueError("EXPLICIT_MANAGED_PUBLIC_KEY_REQUIRED")
    for key in keys:
        if not isinstance(key, str) or not re.fullmatch(r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}(?: [A-Za-z0-9@._-]{1,80})?", key):
            raise ValueError("FULL_ED25519_PUBLIC_KEY_REQUIRED")
        blob = base64.b64decode(key.split()[1], validate=True)
        if len(blob) != 51 or blob[:19] != b"\x00\x00\x00\x0bssh-ed25519\x00\x00\x00\x20":
            raise ValueError("FULL_ED25519_PUBLIC_KEY_REQUIRED")


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


def main():
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
    try:
        for line in sys.stdin:
            if len(line) > 65536: raise ValueError("REQUEST_LIMIT")
            request = json.loads(line)
            if request.get("method") == "tools/call":
                params = request.get("params", {}); args = params.get("arguments", {})
                try:
                    if params.get("name") == "create_container": explicit_keys(args)
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
                except (ValueError, KeyError, TypeError, OSError):
                    print(json.dumps({"jsonrpc": "2.0", "id": request.get("id"),
                                      "error": {"code": -32602, "message": "TRUST_OR_EXPLICIT_KEY_GATE: wait for trusted reconciliation and use the exact managed target/key"}}), flush=True)
                    continue
            send(request)
            if "id" in request: reply(request["id"])
    finally:
        if process.poll() is None:
            process.terminate()
            try: process.wait(timeout=3)
            except subprocess.TimeoutExpired: process.kill()
        process.stdin.close(); process.stdout.close()


if __name__ == "__main__": main()
