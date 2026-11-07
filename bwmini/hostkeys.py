"""Trusted local-Incus public host-key reconciliation for a dedicated MCP account."""
import concurrent.futures, datetime, hashlib, ipaddress, json, os, platform, pwd, re, subprocess, sys, tempfile, time
from pathlib import Path

PUBLIC = Path("/var/lib/boxweave-default/public")
HOME = Path("/home/boxweave")


def command(args):
    return subprocess.run(args, stdin=subprocess.DEVNULL, capture_output=True, text=True, check=True, timeout=15).stdout


def atomic(path, data, uid=0, gid=0, mode=0o644):
    fd, name = tempfile.mkstemp(prefix=".trusted-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as out:
            os.fchmod(out.fileno(), mode); os.fchown(out.fileno(), uid, gid)
            out.write(data); out.flush(); os.fsync(out.fileno())
        os.replace(name, path)
    finally:
        if os.path.exists(name): os.unlink(name)


def eligible(record):
    config = record.get("config", {})
    return (re.fullmatch(r"[a-z][a-z0-9-]{0,62}-container", record.get("name", "")) is not None
            and not record["name"].startswith("containarium-core-")
            and "user.containarium.label.os-type" in config
            and record.get("status") == "Running")


def reconcile():
    if os.geteuid() != 0: raise ValueError("ROOT_REQUIRED")
    account = pwd.getpwnam("boxweave")
    PUBLIC.mkdir(parents=True, exist_ok=True); PUBLIC.chmod(0o755)
    inventory = json.loads(command(["incus", "--force-local", "--project", "default", "list", "--format", "json"]))
    # Each native guest read returns exactly three fixed *public* host keys.
    # Bounded concurrency keeps unrelated or slow Boxes from serially
    # delaying publication for every other Box on busy hosts.
    def sample(item):
        if not eligible(item):
            return None
        addresses = item.get("state", {}).get("network", {}).get("eth0", {}).get("addresses", [])
        ips = [a["address"] for a in addresses if a.get("family") == "inet"
               and a.get("scope") == "global" and ipaddress.ip_address(a["address"]).is_private]
        if len(ips) != 1:
            return None
        name = item["name"]
        try:
            raw = command(["incus", "--force-local", "--project", "default", "exec", name, "--",
                           "cat", "/etc/ssh/ssh_host_ed25519_key.pub",
                           "/etc/ssh/ssh_host_ecdsa_key.pub",
                           "/etc/ssh/ssh_host_rsa_key.pub"])
            rows = raw.splitlines()
            if len(rows) != 3:
                return None
            keys=[]
            for row, algorithm in zip(rows,("ssh-ed25519","ecdsa-sha2-nistp256","ssh-rsa")):
                fields=row.split()
                if len(fields) < 2 or fields[0] != algorithm:
                    return None
                keys.append(" ".join(fields[:2]))
            created = int(datetime.datetime.fromisoformat(
                item["created_at"].replace("Z","+00:00")).timestamp())
            logical=name[:-10]
            return logical, {"name":name,"user":logical,"ip":ips[0],
                             "created":str(created),
                             "uuid":item.get("config",{}).get("volatile.uuid"),
                             "keys":keys}, [ips[0]+" "+key for key in keys]
        except (OSError, subprocess.SubprocessError, ValueError, KeyError, TypeError):
            return None
    targets, lines = {}, []
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
        for item in pool.map(sample, inventory):
            if item is None:
                continue
            logical, record, entries=item
            if logical in targets:
                raise ValueError("DUPLICATE_MANAGED_TARGET")
            targets[logical]=record
            lines.extend(entries)
    sshdir = HOME / ".ssh"
    if sshdir.is_symlink(): raise ValueError("UNSAFE_SSH_DIRECTORY")
    sshdir.mkdir(mode=0o700, exist_ok=True)
    path = sshdir / "known_hosts"
    if path.is_symlink(): raise ValueError("UNSAFE_KNOWN_HOSTS")
    # Dedicated account: preserve lines outside our own explicitly delimited block.
    begin, end = "# BEGIN BOXWEAVE INCUS TRUST", "# END BOXWEAVE INCUS TRUST"
    previous = path.read_text() if path.exists() else ""
    if previous.count(begin) != previous.count(end) or previous.count(begin) > 1: raise ValueError("TRUST_BLOCK_INVALID")
    if begin in previous:
        first, tail = previous.split(begin, 1); _, last = tail.split(end, 1); previous = first + last.lstrip("\n")
    data = (previous.rstrip("\n") + ("\n" if previous.strip() else "") +
            begin + "\n" + "\n".join(lines) + "\n" + end + "\n").encode()
    atomic(path, data, account.pw_uid, account.pw_gid, 0o600)
    atomic(PUBLIC / "targets.json", json.dumps({"observed_at": time.time(),
        "scope": {"provider": "containarium", "backend": platform.node(), "project": "default"},
        "targets": targets}, sort_keys=True).encode())
    print(json.dumps({"state": "trusted_public_keys_synchronized", "targets": len(targets), "project": "default"}))


if __name__ == "__main__": reconcile()
