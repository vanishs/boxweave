"""Explicit official Provider upgrade preflight and authenticated readiness checks.

Publication/backup are owned by install.sh. This helper never installs packages,
reads signing secrets, changes policies or creates workloads.
"""
import argparse
import ipaddress
import json
from pathlib import Path
import re
import subprocess
import time
from urllib.parse import urlsplit

from mcp_client import Client, MCPError
from provenance import version, atomic_json, digest
from webui.tenants import CONTRACT, guard_view, TenantService
from webui.provider import Provider, ProviderError


def local_origin(origin):
    url = urlsplit(origin)
    try:
        local = url.hostname == "localhost" or ipaddress.ip_address(url.hostname).is_loopback
    except ValueError:
        local = False
    if (url.scheme not in ("http", "https") or not local or url.username
            or url.password or url.path not in ("", "/") or url.query or url.fragment):
        raise ValueError("Provider upgrade requires a loopback API origin without credentials or a path")


def target_path(value):
    path = Path(value)
    if (not re.fullmatch(r"/usr/local/[A-Za-z0-9_./-]+", value)
            or ".." in path.parts or path.is_symlink() or not path.is_file()
            or path.resolve() != path):
        raise ValueError("Provider upgrade supports regular binaries under /usr/local only")
    return path


def prepare(directory, daemon, cli, origin, release):
    local_origin(origin)
    target = target_path(daemon)
    # Preserve service arguments and environment; refuse wrappers/another daemon.
    result = subprocess.run(
        ["systemctl", "show", "containarium.service", "-p", "ExecStart", "--value"],
        capture_output=True, text=True, check=True, timeout=10,
    )
    paths = re.findall(r"\bpath=([^ ;]+)", result.stdout)
    if paths != [str(target)]:
        raise ValueError("containarium.service ExecStart must use the configured daemon directly")
    selected = tuple(map(int, release.removeprefix("v").split(".")))
    current = version(str(target))
    if current and tuple(map(int, current.split("-")[0].split("+")[0].split("."))) > selected:
        raise ValueError("Refusing a Provider downgrade")
    cli_path = Path(cli)
    if cli_path.is_symlink() and cli_path.resolve() == target:
        # Upstream's conventional admin CLI alias follows the upgraded daemon.
        cli_target = None
    else:
        cli_target = str(target_path(cli))
        if cli_target == str(target):
            raise ValueError("CLI and daemon targets must be distinct or the upstream symlink")
    dropin = Path("/etc/systemd/system/containarium.service.d/90-boxweave-tenant-guard.conf")
    if dropin.is_symlink() or dropin.parent.resolve() != dropin.parent:
        raise ValueError("Refusing Provider drop-in symlink")
    stage = Path(directory)
    plan = {"release": release, "daemon": str(target), "cli": cli_target,
            "previousDaemonVersion": current, "contract": CONTRACT,
            "cliMode": "daemon-alias" if cli_target is None else "separate-binary"}
    atomic_json(stage / "provider-upgrade.json", plan)
    (stage / "boxweave-tenant-guard.conf").write_text(
        "# Managed by BoxWeave --upgrade-provider. Other settings are preserved.\n"
        "[Service]\nEnvironment=CONTAINARIUM_TENANT_GUARD=enforce\n"
    )
    print("Provider upgrade prepared: " + release + "; " + plan["cliMode"])


def tool_object(value):
    if value.get("isError"):
        raise MCPError("Read-only Provider tool failed")
    for entry in value.get("content", []):
        if entry.get("type") == "text":
            try:
                data = json.loads(entry["text"])
            except (ValueError, KeyError):
                continue
            if isinstance(data, dict):
                return data
    raise MCPError("Provider returned no typed capability/Guard object")


def inspect_mcp(user, timeout):
    client = Client(["runuser", "-u", user, "--", "/usr/local/bin/boxweave-mcp"], timeout=timeout)
    try:
        names, cursor = set(), None
        for _ in range(10):
            page = client.call("tools/list", {"cursor": cursor} if cursor else {}, timeout)
            names.update(t["name"] for t in page.get("tools", []))
            cursor = page.get("nextCursor")
            if not cursor:
                break
        required = {"network_tenant_capabilities", "network_guard_status", "list_containers"}
        if cursor or not required <= names:
            raise MCPError("Installed MCP lacks required native Tenant tools")
        caps = tool_object(client.tool("network_tenant_capabilities", {}, timeout))
        if caps.get("contract") != CONTRACT:
            raise ValueError("Provider contract is not native-network-tenant/v1; verify daemon service path and API origin")
        if caps.get("backendScope") != "local-lxc" or caps.get("nativeMembershipRead") is not True:
            raise ValueError("Provider does not advertise native local-LXC membership")
        guard = guard_view(tool_object(client.tool("network_guard_status", {}, timeout)).get("tenant"))
        if guard["unsupported"] or guard["mode"] == "GUARD_MODE_OFF":
            raise ValueError("Tenant Guard is unsupported or effectively disabled; inspect Provider configuration")
        if not all(caps.get(k) is True for k in (
                "sharedTenantCreate", "atomicTenantAccess", "networkPolicyWrite")):
            raise MCPError("Native Tenant capability is not ready or credential is insufficient")
        if guard["status"] != "ready":
            raise MCPError("Tenant Guard is not fresh, supported and converged in ENFORCE mode")
        client.tool("list_containers", {}, timeout)
        return {"contract": CONTRACT, "backendScope": "local-lxc",
                "sharedTenantCreate": True, "atomicTenantAccess": True,
                "guardStatus": "ready", "connectivityTested": False}
    finally:
        client.close()


def verify(user, timeout, manifest):
    deadline = time.monotonic() + timeout
    while True:
        try:
            evidence = inspect_mcp(user, min(10, max(0.1, (deadline - time.monotonic()) / 16)))
            break
        except ValueError:
            raise  # A definite contract mismatch needs diagnosis, not more waiting.
        except (MCPError, OSError, TimeoutError):
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                raise TimeoutError("Authenticated Tenant readiness deadline exceeded; inspect Provider and Guard logs") from None
            print("[boxweave] Waiting for authenticated native Tenant capabilities and converged Guard", flush=True)
            time.sleep(min(15, remaining))
    path = Path(manifest)
    data = json.loads(path.read_text())
    cfg = json.loads(Path("/etc/boxweave/config.json").read_text())
    data["daemon"] = {"reported_version": version(cfg["containarium_daemon_binary"]),
                      "sha256": digest(cfg["containarium_daemon_binary"])}
    data["tenant_network"] = evidence
    atomic_json(path, data)
    print(json.dumps(evidence, sort_keys=True))
    print("[boxweave] Native Tenant contract and Guard ready; TCP connectivity has not been tested")


def verify_web(manifest):
    # Exercise the exact Web adapter with its independent credential. Provider
    # reads the protected token internally; no token or raw response is logged.
    cfg = json.loads(Path("/etc/boxweave/web.json").read_text())
    provider = Provider(cfg)
    caps = TenantService(None, provider).capabilities()
    if (caps.get("contract") != CONTRACT or not caps.get("sharedTenantCreate")
            or not caps.get("networkPolicyWrite")):
        raise ValueError("Independent Web credential does not authorize the native Tenant contract")
    if guard_view(provider.network_guard().get("tenant"))["status"] != "ready":
        raise ValueError("Independent Web Guard read is not ready")
    provider.network_policies()
    path = Path(manifest)
    data = json.loads(path.read_text())
    data["tenant_network"]["independentWebCredentialVerified"] = True
    atomic_json(path, data)
    print("[boxweave] Independent Web Tenant capability, Guard and policy reads verified; no policies changed")


def main():
    p = argparse.ArgumentParser(description=__doc__)
    sub = p.add_subparsers(dest="action", required=True)
    prepare_cmd = sub.add_parser("prepare")
    for key in ("directory", "daemon", "cli", "origin", "release"):
        prepare_cmd.add_argument("--" + key, required=True)
    check = sub.add_parser("verify")
    check.add_argument("--user", required=True)
    check.add_argument("--timeout", type=int, default=600)
    check.add_argument("--manifest", default="/etc/boxweave/install-manifest.json")
    web = sub.add_parser("verify-web")
    web.add_argument("--manifest", default="/etc/boxweave/install-manifest.json")
    a = vars(p.parse_args())
    action = a.pop("action")
    try:
        {"prepare": prepare, "verify": verify, "verify-web": verify_web}[action](**a)
    except (ValueError, OSError, subprocess.SubprocessError, TimeoutError, MCPError, ProviderError) as exc:
        # Never print raw subprocess stderr, tokens or response bodies.
        raise SystemExit("Provider upgrade/check failed: " + (str(exc) if isinstance(exc, (ValueError, TimeoutError)) else type(exc).__name__)) from None


if __name__ == "__main__":
    main()
