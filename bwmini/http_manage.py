"""Non-root managed HTTP MCP setup and diagnostics.

Invoked only through the fixed app identity by the reviewed host CLI.
Never installs tunnels, changes firewall, or exposes a non-loopback listener.
"""
import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import subprocess
import urllib.error
import urllib.request

from .distribution import atomic, config_read, installed, regular
from .remote_mcp import validate_remote_config, public_origin

PREFIX = Path("/home/boxweave/boxweave")


def unit_name(prefix):
    identity = hashlib.sha256(str(prefix).encode()).hexdigest()[:12]
    return "boxweave-v2-" + identity + "-remote-mcp.service"


def user_control(prefix, action):
    name = unit_name(prefix)
    state = installed(prefix)
    if not state or name not in state.get("units", []):
        raise ValueError("REMOTE_SERVICE_NOT_MANAGED")
    proc = subprocess.run(["systemctl", "--user", action, name],
                          capture_output=True, timeout=20)
    if proc.returncode != 0:
        raise ValueError("REMOTE_SERVICE_CONTROL_FAILED")


def read_remote(prefix):
    cfg = json.loads(regular(prefix / "runtime.json"))
    if not isinstance(cfg, dict):
        raise ValueError("REMOTE_CONFIG_INVALID")
    remote = cfg.get("remote_mcp")
    if isinstance(remote, dict) and remote.get("enabled") is False:
        return {"enabled": False, "port": remote.get("port", 18844)}
    return validate_remote_config(remote)


def observation(prefix):
    remote = read_remote(prefix)
    port = remote.get("port", 18844)
    result = {"enabled": remote["enabled"], "listener": "127.0.0.1",
              "port": port, "local_url": "http://127.0.0.1:" + str(port) + "/mcp",
              "public_url": remote.get("public_url"), "external_verified": False,
              "local_discovery": False}
    if not result["enabled"]:
        return result
    url = "http://127.0.0.1:" + str(port) + "/.well-known/oauth-protected-resource/mcp"
    try:
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
        with opener.open(url, timeout=2) as response:
            payload = json.loads(response.read(4096))
        expected = (remote.get("public_url") or "http://127.0.0.1:" + str(port)) + "/mcp"
        result["local_discovery"] = (payload.get("resource") == expected)
    except (OSError, ValueError, TypeError, urllib.error.HTTPError):
        pass
    result["managed_service"] = bool(installed(prefix) and
                                    unit_name(prefix) in installed(prefix).get("units", []))
    return result


def configure(prefix, public_url=None, local_only=False):
    if not local_only:
        public_origin(public_url)
    with (prefix / ".install.lock").open("a") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        if (prefix / "update-in-progress.json").exists():
            raise ValueError("INSTALL_IN_PROGRESS")
        config_path, runtime_path = prefix / "config.json", prefix / "runtime.json"
        original, config = config_read(config_path, prefix)
        original_runtime = regular(runtime_path)
        runtime = json.loads(original_runtime)
        if not isinstance(runtime, dict):
            raise ValueError("REMOTE_CONFIG_INVALID")
        remote = dict(config.get("remote_mcp") or {})
        if remote.get("enabled") is False:
            raise ValueError("REMOTE_EXPLICITLY_DISABLED")
        remote["enabled"] = True
        remote.setdefault("port", 18844)
        if local_only:
            remote.pop("public_url", None)
        else:
            remote["public_url"] = public_url
        remote = validate_remote_config(remote)
        if remote["port"] == config["port"]:
            raise ValueError("REMOTE_PORT_CONFLICT")
        config["remote_mcp"] = remote
        runtime["remote_mcp"] = remote
        serialized = lambda value: (json.dumps(value, sort_keys=True, indent=2) + "\n").encode()
        # Two owned private files; if a write or restart fails, restore both.
        try:
            atomic(config_path, serialized(config))
            atomic(runtime_path, serialized(runtime))
            user_control(prefix, "restart")
        except Exception:
            atomic(config_path, original)
            atomic(runtime_path, original_runtime)
            try:
                user_control(prefix, "restart")
            except Exception:
                pass
            raise
    return observation(prefix)


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("action", choices=("status", "doctor", "configure", "restart"))
    p.add_argument("--public-url")
    p.add_argument("--local-only", action="store_true")
    p.add_argument("--prefix", default=str(PREFIX))
    args = p.parse_args(argv)
    try:
        prefix = Path(args.prefix)
        if args.action == "configure":
            if bool(args.public_url) == bool(args.local_only):
                raise ValueError("CHOOSE_PUBLIC_URL_OR_LOCAL_ONLY")
            result = configure(prefix, args.public_url, args.local_only)
        elif args.action == "restart":
            user_control(prefix, "restart")
            result = observation(prefix)
        else:
            result = observation(prefix)
        print(json.dumps(result, sort_keys=True))
        return 0 if args.action not in ("doctor", "restart", "configure") or (
            not result["enabled"] or result["local_discovery"]) else 2
    except (OSError, ValueError, TypeError, KeyError, subprocess.TimeoutExpired) as exc:
        print(json.dumps({"state": "error", "code": str(exc) if isinstance(exc, ValueError)
                          else "REMOTE_MANAGEMENT_FAILED"}))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
