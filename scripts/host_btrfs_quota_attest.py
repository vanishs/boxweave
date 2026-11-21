#!/usr/bin/env python3
"""Host-root Btrfs qgroup witness for the installed BoxWeave service.

Run manually on the trusted host against a disposable Box that has reached its
configured quota. The default audit is read-only except for the witness.
An explicit near-full probe writes at most 128 KiB to a temporary guest file.
Never grants MCP host access or changes qgroups or instance configuration.
"""
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import time
from urllib.parse import quote

OUTPUT = Path("/etc/boxweave/storage-quota-witness.json")
POOL_MOUNT_ROOT = Path("/var/lib/incus/storage-pools")
NAME = re.compile(r"[a-z][a-z0-9-]{1,62}\Z")
PROJECT = re.compile(r"user-[0-9]+\Z")
POOL = re.compile(r"[A-Za-z0-9_-]+\Z")
UUID = re.compile(r"[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}\Z")


MAX_SLACK = 64 << 10
WRITE_REFUSAL = "GUEST_EDQUOT_128K"
# A bounded, automatically unlinked file in the exact guest root filesystem.
# ENOSPC, process start failure, successful write and any other error are not proof.
GUEST_PROBE = """import errno, os, tempfile
try:
    with tempfile.TemporaryFile(dir="/", prefix=".boxweave-quota-probe-", buffering=0) as f:
        data = memoryview(os.urandom(128 << 10))
        while data:
            n = os.write(f.fileno(), data)
            if n <= 0:
                raise RuntimeError("short write without errno")
            data = data[n:]
        os.fsync(f.fileno())
except OSError as exc:
    if exc.errno != errno.EDQUOT:
        raise
    print("GUEST_EDQUOT_128K")
else:
    raise SystemExit("bounded write succeeded")
"""


def near_full(referenced, maximum, quota_bytes):
    return maximum == quota_bytes and 0 <= maximum - referenced <= MAX_SLACK


def verify_write_refusal(incus, project, name):
    result = run(incus, "exec", name, "--project", project,
                 "--mode=non-interactive", "--", "python3", "-c", GUEST_PROBE)
    if result.strip() != WRITE_REFUSAL:
        raise ValueError("exact guest EDQUOT proof unavailable")



def run(*args):
    cp = subprocess.run(args, check=True, capture_output=True, text=True, timeout=25)
    return cp.stdout


def incus_json(incus, api_path):
    val = json.loads(run(incus, "query", api_path))
    return val.get("metadata", val)


def qgroup_limit(output, expected_path):
    found = []
    for line in output.splitlines():
        parts = line.split(maxsplit=5)
        if len(parts) != 6 or not re.fullmatch(r"[0-9]+/[0-9]+", parts[0]):
            continue
        if parts[5] != expected_path:
            continue
        if not (parts[1].isdigit() and parts[3].isdigit()):
            raise ValueError("matched qgroup has no numeric referenced/limit")
        found.append((int(parts[1]), int(parts[3])))
    if len(found) != 1:
        raise ValueError("exact Box qgroup path absent or nonunique")
    return found[0]


def root_owned_secure_directory(path):
    if path.exists() and path.is_symlink():
        raise ValueError("attestation directory is a symlink")
    path.mkdir(mode=0o755, exist_ok=True)
    st = path.lstat()
    if not stat.S_ISDIR(st.st_mode) or st.st_uid != 0 or st.st_mode & 0o022:
        raise ValueError("attestation directory is not root-owned and nonwritable")


def audit(project, name, verify_refusal=False, expected_uuid=None):
    incus, btrfs = shutil.which("incus"), shutil.which("btrfs")
    if not incus or not btrfs:
        raise ValueError("Incus and btrfs-progs must be installed on the host")
    instance = incus_json(incus, "/1.0/instances/" + quote(name) + "?project=" + quote(project))
    uuid = instance.get("config", {}).get("volatile.uuid", "")
    if not UUID.fullmatch(uuid):
        raise ValueError("native Box UUID unavailable")
    if expected_uuid is not None and uuid != expected_uuid:
        raise ValueError("native Box UUID does not match expected disposable fixture")
    if verify_refusal and expected_uuid is None:
        raise ValueError("near-full probe requires the exact disposable Box UUID")
    root = [d for d in instance.get("expanded_devices", {}).values()
            if d.get("type") == "disk" and d.get("path") == "/"]
    if len(root) != 1 or not root[0].get("pool"):
        raise ValueError("exact native root volume not configured")
    pool_name = root[0]["pool"]
    if not POOL.fullmatch(pool_name):
        raise ValueError("invalid native storage pool name")
    size = re.fullmatch(r"([0-9]+)GiB", root[0].get("size", ""))
    if not size or int(size[1]) < 2:
        raise ValueError("native root disk size not explicitly limited")
    quota_bytes = int(size[1]) * (1 << 30)
    pool = incus_json(incus, "/1.0/storage-pools/" + quote(pool_name))
    if pool.get("driver") != "btrfs" or pool.get("status", "").lower() != "created":
        raise ValueError("selected Incus pool is not a ready Btrfs pool")
    source_text = pool.get("config", {}).get("source", "")
    source = Path(source_text)
    if not source.is_absolute() or ".." in source.parts or source.is_symlink() or not source.is_dir():
        raise ValueError("selected Btrfs source is not an exact absolute native directory")
    if source.resolve() != source:
        raise ValueError("pool source traverses symlinks")
    project_state = incus_json(incus, "/1.0/projects/" + quote(project))
    if project_state.get("config", {}).get("restricted") != "true":
        raise ValueError("target Project is not restricted")
    quota = run(btrfs, "quota", "status", str(source)).lower()
    if not re.search(r"enabled:\s*yes", quota) or not re.search(r"inconsistent:\s*no", quota):
        raise ValueError("Btrfs qgroups disabled or inconsistent")
    relative = source.as_posix().lstrip("/") + "/containers/" + project + "_" + name
    referenced, maximum = qgroup_limit(run(btrfs, "qgroup", "show", "-re", "--raw", str(source)), relative)
    if maximum != quota_bytes:
        raise ValueError("native Btrfs qgroup limit differs from requested quota")
    witness_schema = 1
    if referenced < maximum:
        if not verify_refusal or not near_full(referenced, maximum, quota_bytes):
            raise ValueError("Box did not reach exact limit; eligible near-full fixture needs explicit EDQUOT probe")
        verify_write_refusal(incus, project, name)
        referenced, maximum = qgroup_limit(run(btrfs, "qgroup", "show", "-re", "--raw", str(source)), relative)
        if not near_full(referenced, maximum, quota_bytes):
            raise ValueError("native qgroup changed or is not near-full after probe")
        witness_schema = 2
    # Read the same native UUID and effective root size a second time to
    # detect replacement during the read-only audit.
    recheck = incus_json(incus, "/1.0/instances/" + quote(name) + "?project=" + quote(project))
    if recheck.get("config", {}).get("volatile.uuid") != uuid or (
        recheck.get("expanded_devices", {}).get(next(k for k, d in instance["expanded_devices"].items()
            if d.get("type") == "disk" and d.get("path") == "/"), {}).get("size") != size.group(0)
    ):
        raise ValueError("native Box identity/limit changed during audit")
    mount = POOL_MOUNT_ROOT / pool_name
    if mount.is_symlink() or not mount.is_dir():
        raise ValueError("canonical Incus pool mount unavailable")
    st = mount.stat()
    evidence = {
        "schema": witness_schema, "project": project, "pool": pool_name, "driver": "btrfs",
        "box_uuid": uuid, "boot_id": Path("/proc/sys/kernel/random/boot_id").read_text().strip(),
        "mount_dev": st.st_dev, "mount_ino": st.st_ino,
        "observed_unix": int(time.time()), "quota_bytes": quota_bytes,
        "referenced_bytes": referenced, "qgroup_max_referenced_bytes": maximum,
    }
    if witness_schema == 2:
        evidence["write_refusal"] = WRITE_REFUSAL
    return evidence


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--project", required=True)
    parser.add_argument("--box-name", required=True)
    parser.add_argument("--verify-write-refusal", action="store_true",
                        help="explicitly probe only a disposable near-full guest, at most 128 KiB")
    parser.add_argument("--expected-box-uuid", help="exact disposable native UUID for probe")
    args = parser.parse_args()
    if args.expected_box_uuid and not UUID.fullmatch(args.expected_box_uuid):
        parser.error("invalid expected Box UUID")
    if args.verify_write_refusal and not args.expected_box_uuid:
        parser.error("explicit probe requires --expected-box-uuid")
    if os.geteuid() != 0 or not PROJECT.fullmatch(args.project) or not NAME.fullmatch(args.box_name):
        parser.error("host root, exact user-N Project and safe Box name required")
    evidence = audit(args.project, args.box_name, args.verify_write_refusal, args.expected_box_uuid)
    root_owned_secure_directory(OUTPUT.parent)
    fd, temp_name = tempfile.mkstemp(prefix=".quota-witness-", dir=OUTPUT.parent)
    try:
        os.fchmod(fd, 0o644)
        with os.fdopen(fd, "w", encoding="utf-8") as out:
            json.dump(evidence, out, sort_keys=True)
            out.write("\n")
            out.flush()
            os.fsync(out.fileno())
        os.replace(temp_name, OUTPUT)
    finally:
        if os.path.exists(temp_name):
            os.unlink(temp_name)
    print(json.dumps({"evidence": ("HOST_QGROUP_MAX_REACHED" if evidence["schema"] == 1
                                  else "HOST_QGROUP_NEAR_FULL_EDQUOT_VERIFIED"), "project": evidence["project"],
                      "pool": evidence["pool"], "expires_in_hours": 4}))


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, subprocess.SubprocessError) as exc:
        print("QUOTA_WITNESS_UNVERIFIED: " + str(exc), file=sys.stderr)
        sys.exit(2)
