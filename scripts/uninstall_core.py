#!/usr/bin/env python3
"""Operator-reviewed BoxWeave host removal. Fixed paths, no silent native ownership."""
import argparse
from contextlib import contextmanager
import fcntl
import hashlib
import json
import os
from pathlib import Path
import pwd
import re
import sqlite3
import shutil
import stat
import subprocess
import sys
import tempfile

STATE = Path("/var/lib/boxweave")
UNIT = Path("/etc/systemd/system/boxweave.service")
BINARY = Path("/usr/local/bin/boxweave")
NATIVE = Path("/var/lib/boxweave-native-prereq-v1.json")
SCRIPTLET = Path("/var/lib/boxweave-native-scriptlet-v1.json")
ROOT_UNIT = Path("/etc/systemd/system/boxweave-native-peer-root.service")
RETIREMENT_UNIT = Path("/etc/systemd/system/boxweave-ipv4-retirement.service")
RETIREMENT_SOCKET = Path("/run/boxweave-ipv4-retirement.sock")
AUDIT = Path("/var/lib/boxweave-uninstall-v1-audit.json")
BACKUP = Path("/var/lib/boxweave-installer-backup")
RETIRED = Path("/var/lib/boxweave-retired")
SCOPED_JOURNAL = Path("/var/lib/boxweave-scoped-peer-v2")
SCOPED_GUARD_UNIT = Path("/etc/systemd/system/boxweave-scoped-guard.service")
SCOPED_LIFECYCLE_UNIT = Path("/etc/systemd/system/boxweave-scoped-lifecycle.service")
SCOPED_INCUS_DROPIN = Path("/etc/systemd/system/incus.service.d/50-boxweave-scoped-guard.conf")
SCOPED_SOCKET = Path("/run/boxweave-scoped-lifecycle.sock")
SCOPED_DRAIN = Path("/var/lib/boxweave-scoped-drain-v1.json")
SCOPED_FENCE = Path("/var/lib/boxweave-scoped-retirement-v1.json")
PURGE_FENCE = Path("/var/lib/boxweave-purge-v1.json")
SCOPED_LOCK = Path("/run/lock/boxweave-scoped-retirement.lock")
MAINTENANCE_LOCK = Path("/run/lock/boxweave-v1-maintenance.lock")
VALID_NAME = re.compile(r"[a-z][a-z0-9-]{0,62}\Z")
VALID_UUID = re.compile(r"[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}\Z")
VALID_SHA = re.compile(r"[0-9a-f]{64}\Z")
TERMINAL = {"SUCCEEDED", "FAILED", "CANCELLED", "TIMED_OUT"}
SERVE_LOCK_NAME = "ledger.sqlite.serve.lock"
EXEC_ARCHIVE_NAME = "ledger.sqlite.exec-archive"
EXEC_ARCHIVE_RE = re.compile(r"[a-f0-9]{32}\.(stdout|stderr)\Z")
STATE_FILES = {"ledger.sqlite", "ledger.sqlite-wal", "ledger.sqlite-shm",
               "install-manifest.json", SERVE_LOCK_NAME}
SCOPES = ("service", "controller", "full")


class UninstallError(Exception):
    pass


def refuse_scoped_retirement():
    """Legacy retirement, including resume, cannot consume a scoped installation."""
    for path in (SCOPED_JOURNAL, SCOPED_GUARD_UNIT, SCOPED_LIFECYCLE_UNIT,
                 SCOPED_INCUS_DROPIN, SCOPED_SOCKET, SCOPED_FENCE, SCOPED_DRAIN):
        try:
            path.lstat()
        except FileNotFoundError:
            continue
        except OSError as exc:
            raise UninstallError("scoped v2 evidence cannot be inspected; preserve native guard and journal") from exc
        raise UninstallError("scoped v2 retirement requires separate reviewed implementation; preserve native guard and journal: " + str(path))
    manifest = read_json(STATE / "install-manifest.json")
    if isinstance(manifest, dict) and manifest.get("scoped_peer_mode") not in (None, ""):
        raise UninstallError("scoped v2 retirement requires separate reviewed implementation; preserve native guard and journal")


def sha(data):
    return hashlib.sha256(data).hexdigest()


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode()


def checked_file(path, kind, max_size=None):
    """lstat, no symlinks, never open unexpected inodes."""
    try:
        info = path.lstat()
    except FileNotFoundError:
        return None
    expected = {"file": stat.S_ISREG, "dir": stat.S_ISDIR}[kind]
    if not expected(info.st_mode) or (max_size is not None and info.st_size > max_size):
        raise UninstallError("unsafe path type or size: " + str(path))
    if info.st_uid not in (0, pwd.getpwnam("boxweave").pw_uid):
        raise UninstallError("foreign-owned path: " + str(path))
    return info


def require_root_owned(path):
    if path.lstat().st_uid != 0:
        raise UninstallError("installer-owned root path has foreign owner: " + str(path))


def validate_state_dir(path, service_uid):
    info = checked_file(path, "dir")
    if info is None:
        raise UninstallError("verified controller state directory missing")
    if info.st_uid != service_uid or stat.S_IMODE(info.st_mode) != 0o700:
        raise UninstallError("state directory owner/mode no longer matches installer policy")
    return info


def require_service_lock_owner(path, service_uid):
    if path.lstat().st_uid != service_uid:
        raise UninstallError("serve lock must belong to exact service UID")


def require_serve_lock_fd_owner(fd, service_uid):
    if os.fstat(fd).st_uid != service_uid:
        raise UninstallError("serve lock FD must belong to service UID")


def verified_serve_lock(path, service_uid):
    info = checked_file(path, "file", 4096)
    if info is None:
        return None
    if stat.S_IMODE(info.st_mode) != 0o600 or info.st_nlink != 1:
        raise UninstallError("unsafe serve lock mode or hardlink")
    require_service_lock_owner(path, service_uid)
    return info


def file_fingerprint(path):
    info = checked_file(path, "file", 120 * 1024 * 1024 if path == BINARY else None)
    if info is None:
        return None
    fd = os.open(str(path), os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0))
    try:
        fdinfo = os.fstat(fd)
        if fdinfo.st_ino != info.st_ino or fdinfo.st_dev != info.st_dev:
            raise UninstallError("path changed during inspection: " + str(path))
        digest = hashlib.sha256()
        while True:
            b = os.read(fd, 1024 * 1024)
            if not b:
                break
            digest.update(b)
        return {"sha256": digest.hexdigest(), "device": info.st_dev, "inode": info.st_ino,
                "mode": stat.S_IMODE(info.st_mode), "size": info.st_size}
    finally:
        os.close(fd)


def read_json(path, cap=1024 * 1024):
    if checked_file(path, "file", cap) is None:
        return None
    fd = os.open(str(path), os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0))
    try:
        raw = os.read(fd, cap + 1)
        if len(raw) > cap:
            raise UninstallError("oversized protected manifest")
        obj = json.loads(raw)
        if not isinstance(obj, dict):
            raise UninstallError("manifest must be JSON object")
        return obj
    finally:
        os.close(fd)


def command(argv, acceptable=(0,), timeout=30):
    try:
        # Never inherit an arbitrary CLI remote, shell PATH, user config or
        # proxy into an operator-root native cleanup invocation.
        safe_env = {"PATH": "/usr/sbin:/usr/bin:/sbin:/bin",
                    "HOME": "/root", "LC_ALL": "C", "LANG": "C"}
        run = subprocess.run(argv, capture_output=True, text=True, timeout=timeout,
                             env=safe_env, check=False)
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise UninstallError("local command unavailable: " + argv[0]) from exc
    if run.returncode not in acceptable:
        raise UninstallError("local command refused: " + " ".join(argv[:2]))
    return run.stdout.strip(), run.returncode


def machine_sha():
    path = Path("/etc/machine-id")
    mid = path.read_text(encoding="ascii").strip()
    if not re.fullmatch(r"[a-f0-9]{32}", mid):
        raise UninstallError("missing/invalid machine ID")
    return sha(mid.encode())


def cold_sqlite_files(dbpath):
    dbpath = Path(dbpath)
    owner = dbpath.parent.lstat()
    if (not dbpath.is_absolute() or dbpath.parent.resolve() != dbpath.parent or
            not stat.S_ISDIR(owner.st_mode) or owner.st_mode & 0o077):
        raise UninstallError("OFFLINE_SQLITE_PRIVATE_DIRECTORY_REQUIRED")
    files = {}
    total = 0
    for suffix in ("", "-wal", "-shm", "-journal"):
        path = Path(str(dbpath) + suffix)
        info = checked_file(path, "file", 32 * 1024 * 1024)
        if info is not None:
            total += info.st_size
            if (total > 32 * 1024 * 1024 or info.st_nlink != 1 or
                    info.st_uid != owner.st_uid or info.st_dev != owner.st_dev or
                    info.st_mode & 0o077):
                raise UninstallError("OFFLINE_SQLITE_SOURCE_IDENTITY_OR_BOUND_UNVERIFIED")
        files[suffix] = file_fingerprint(path)
    return files


@contextmanager
def cold_sqlite_copy(dbpath):
    # Same SQLite Unix VFS cold exclusive DB range as diagnostic_snapshot.go.
    # Never open SQLite on the source, truncate its WAL, or discard evidence.
    dbpath = Path(dbpath)
    before = cold_sqlite_files(dbpath)
    if before[""] is None or before["-journal"] is not None:
        raise UninstallError("OFFLINE_SQLITE_ROLLBACK_JOURNAL_OR_DB_UNVERIFIED")
    owner = dbpath.parent.lstat()
    if not stat.S_ISDIR(owner.st_mode) or owner.st_mode & 0o077:
        raise UninstallError("OFFLINE_SQLITE_PRIVATE_DIRECTORY_REQUIRED")
    fds = {}
    try:
        for suffix, proof in before.items():
            if proof is None:
                continue
            path = Path(str(dbpath) + suffix)
            fd = os.open(path, (os.O_RDWR if suffix == "" else os.O_RDONLY) |
                         os.O_NOFOLLOW | os.O_NONBLOCK)
            fds[suffix] = fd
            st = os.fstat(fd)
            if (not stat.S_ISREG(st.st_mode) or st.st_nlink != 1 or
                    st.st_uid != owner.st_uid or st.st_dev != owner.st_dev or
                    st.st_mode & 0o077 or (st.st_dev, st.st_ino, st.st_size) !=
                    (proof["device"], proof["inode"], proof["size"])):
                raise UninstallError("OFFLINE_SQLITE_SOURCE_IDENTITY_UNVERIFIED")
        if sum(x["size"] for x in before.values() if x) > 32 * 1024 * 1024:
            raise UninstallError("OFFLINE_SQLITE_CAPTURE_LIMIT")
        try:
            fcntl.lockf(fds[""], fcntl.LOCK_EX | fcntl.LOCK_NB, 512, 0x40000000)
        except OSError as exc:
            raise UninstallError("OFFLINE_SQLITE_SOURCE_BUSY") from exc
        with tempfile.TemporaryDirectory(prefix="boxweave-offline-sqlite-") as tmp:
            dst = Path(tmp) / "ledger.sqlite"
            for suffix, fd in fds.items():
                st = os.fstat(fd)
                h = hashlib.sha256()
                # SHM is an index, not durable transaction data. Verify its
                # bytes for archive provenance; rebuild only on private copy.
                target = Path(str(dst) + suffix)
                with target.open("xb") as out:
                    os.fchmod(out.fileno(), 0o600)
                    remaining = before[suffix]["size"]
                    while remaining:
                        chunk = os.read(fd, min(remaining, 1024 * 1024))
                        if not chunk:
                            raise UninstallError("OFFLINE_SQLITE_EPOCH_CHANGED")
                        remaining -= len(chunk)
                        h.update(chunk)
                        if suffix != "-shm":
                            out.write(chunk)
                if suffix == "-shm":
                    target.unlink()
                now = os.fstat(fd)
                current = Path(str(dbpath) + suffix).lstat()
                if (h.hexdigest() != before[suffix]["sha256"] or
                        (st.st_ino, st.st_size, st.st_mtime_ns, st.st_ctime_ns) !=
                        (now.st_ino, now.st_size, now.st_mtime_ns, now.st_ctime_ns) or
                        (current.st_dev, current.st_ino) != (now.st_dev, now.st_ino)):
                    raise UninstallError("OFFLINE_SQLITE_EPOCH_CHANGED")
            # Reconstruct WAL on a private copy; never use immutable=1 here.
            yield dst
    finally:
        for fd in fds.values():
            os.close(fd)
    if cold_sqlite_files(dbpath) != before:
        raise UninstallError("OFFLINE_SQLITE_EPOCH_CHANGED")


def inspect_sqlite(dbpath, guided=False, cold=False):
    """A read-only snapshot; never import a v4/v5 DB into a new controller."""
    if checked_file(dbpath, "file") is None:
        raise UninstallError("no readable controller SQLite evidence")
    if cold:
        with cold_sqlite_copy(dbpath) as copy:
            conn = sqlite3.connect(copy.as_uri() + "?mode=ro", uri=True, timeout=5)
            try:
                if conn.execute("PRAGMA quick_check").fetchall() != [("ok",)]:
                    raise UninstallError("OFFLINE_SQLITE_PRIVATE_INTEGRITY_UNVERIFIED")
            except sqlite3.Error as exc:
                raise UninstallError("OFFLINE_SQLITE_PRIVATE_INTEGRITY_UNVERIFIED") from exc
            finally:
                conn.close()
            return inspect_sqlite(copy, guided=guided)
    conn = sqlite3.connect("file:" + str(dbpath) + "?mode=ro", uri=True, timeout=5)
    try:
        conn.execute("PRAGMA query_only=ON")
        conn.execute("BEGIN")
        version = [x[0] for x in conn.execute("SELECT version FROM schema_version")]
        if len(version) != 1 or version[0] not in (4, 5, 6):
            raise UninstallError("unrecognized controller database schema")
        operation_hash = hashlib.sha256()
        operation_count = 0
        blockers = []
        unresolved = 0
        historical = []
        historical_released_other = []
        for op, layer, box, state, native in conn.execute(
                "SELECT id,layer,box_uuid,state,native_operation_id FROM operations ORDER BY id"):
            operation_hash.update(canonical([op, layer, box, state, native]) + b"\n")
            operation_count += 1
            if state not in TERMINAL:
                # A reviewed UNKNOWN is never rewritten to SUCCEEDED/FAILED.
                # For full retirement, exact successful *later* Box Delete
                # and an earlier local release audit must prove that no live
                # Box is being abandoned. Network/foreign UNKNOWNs remain blocked.
                if guided and version[0] in (5, 6) and state == "UNKNOWN":
                    witness = conn.execute("""
                        SELECT o.id, b.uuid, d.id, r.seq, r.evidence_digest, o.action
                        FROM operations o
                        JOIN owned_boxes b ON b.uuid=o.box_uuid AND b.incarnation=o.incarnation
                           AND b.project=o.layer AND b.state='DELETED' AND b.deleted_at>0
                        JOIN operations d ON d.id=b.deleted_by_operation_id AND d.action='box_delete'
                           AND d.state='SUCCEEDED' AND d.layer=b.project
                           AND d.box_uuid=b.uuid AND d.incarnation=b.incarnation
                        JOIN recovery_audit r ON r.operation_id=o.id AND r.decision='RELEASED'
                        WHERE o.id=? AND o.action IN
                            ('box_exec','box_start','box_stop','box_update','box_wait_ready')
                            AND r.at<=b.deleted_at
                        ORDER BY r.seq
                    """, (op,)).fetchall()
                    if len(witness) == 1 and VALID_SHA.fullmatch(witness[0][4]):
                        proof = list(witness[0])
                        if proof[5] == "box_exec":
                            historical.append(proof[:5])
                        else:
                            historical_released_other.append(proof)
                        operation_hash.update(canonical(proof) + b"\n")
                        continue
                unresolved += 1
                if unresolved <= 24:
                    blockers.append("OPERATION_" + state + ":" + op + ":native=" + (native or "unverified"))
        if unresolved > 24:
            blockers.append("MORE_NONTERMINAL_OPERATIONS:" + str(unresolved - 24))
        boxes = [dict(uuid=u, project=p, name=n, state=s)
                 for u, p, n, s in conn.execute(
                     "SELECT uuid,project,name,state FROM owned_boxes WHERE deleted_at=0 ORDER BY uuid")]
        for table, condition in (
            ("peer_grants", "status NOT IN ('REVOKED')"),
            ("forward_claims", "state NOT IN ('REVOKED')"),
            ("network_address_fences", "state NOT IN ('RELEASED')"),
            ("exec_output_gc", "state NOT IN ('REVIEWED_RELEASED','EXPIRED')" if guided else "state NOT IN ('REVIEWED_RELEASED')"),
        ):
            count = conn.execute("SELECT count(*) FROM " + table + " WHERE " + condition).fetchone()[0]
            if count:
                blockers.append(table + "_NOT_RELEASED:" + str(count))
        if version[0] == 6:
            # An uncertain root ACL update is not an ordinary terminal Delete.
            # Its project fence must survive; do not uninstall away recovery evidence.
            count = conn.execute("SELECT count(*) FROM network_ipv4_compactions").fetchone()[0]
            if count:
                blockers.append("IPV4_COMPACTION_PROTECTED:" + str(count))
        # Exact operation IDs/state/native refs bind the human review to the
        # current durable originals, including terminal operations.
        return {"version": version[0], "operations": operation_count,
                "operations_sha256": operation_hash.hexdigest(), "boxes": boxes,
                "unresolved_operations": unresolved, "historical_released_exec": historical,
                "historical_released_other": historical_released_other,
                "blockers": blockers}
    except sqlite3.Error as exc:
        raise UninstallError("SQLite snapshot cannot be verified: " + str(exc)) from exc
    finally:
        conn.close()


def native_instances(project, guided=False):
    raw, _ = command(["incus", "list", "local:", "--project", project, "--format", "json"], timeout=25)
    try:
        data = json.loads(raw)
    except ValueError as exc:
        raise UninstallError("unreadable Incus instance inventory") from exc
    if not isinstance(data, list):
        raise UninstallError("unexpected Incus inventory")
    result = []
    for entry in data:
        if not isinstance(entry, dict):
            raise UninstallError("invalid Incus instance record")
        name, cfg = entry.get("name"), entry.get("config") or {}
        uuid = cfg.get("volatile.uuid")
        status = entry.get("status")
        if not isinstance(name, str) or not VALID_NAME.fullmatch(name) or \
                not isinstance(uuid, str) or not VALID_UUID.fullmatch(uuid) or \
                not isinstance(status, str):
            raise UninstallError("Incus UUID/name/status uncertain; no native deletion")
        if guided:
            devices = entry.get("expanded_devices")
            if entry.get("type") != "container" or not isinstance(devices, dict):
                raise UninstallError("unverified native container devices")
            if any(not isinstance(dev, dict) or dev.get("type") == "proxy"
                   for dev in devices.values()):
                raise UninstallError("native proxy or unknown device: reconcile before guided removal")
        result.append({"uuid": uuid, "name": name, "state": status})
    return sorted(result, key=lambda x: x["uuid"])


def require_quiescent_native(project):
    require_quiescent_scoped_native(project)


def require_quiescent_scoped_native(project):
    """Incus retains terminal operations briefly after MCP has settled them.
    Check the exact native terminal state, never mistake retention for activity.
    The same exact terminal observation is required for Legacy and Scoped.
    """
    raw, _ = command(["incus", "operation", "list", "local:", "--project",
                      project, "--format", "json"], timeout=25)
    try:
        data = json.loads(raw)
    except ValueError as exc:
        raise UninstallError("scoped native operation inventory unreadable") from exc
    terminal = {200: "Success", 400: "Failure", 401: "Cancelled"}
    if not isinstance(data, list):
        raise UninstallError("scoped native operation inventory unverified")
    seen = set()
    for item in data:
        if (not isinstance(item, dict) or not isinstance(item.get("id"), str) or
                not VALID_UUID.fullmatch(item["id"]) or item["id"] in seen or
                type(item.get("status_code")) is not int or
                terminal.get(item["status_code"]) != item.get("status") or
                item["status_code"] not in terminal or item.get("may_cancel") is not False):
            raise UninstallError("scoped native operation active or unverified: settle original first")
        seen.add(item["id"])


def require_root_journal(path):
    info = checked_file(path, "file", 1024 * 1024)
    if info is not None:
        require_root_owned(path)
        if stat.S_IMODE(info.st_mode) != 0o600:
            raise UninstallError("untrusted root journal permission: " + str(path))


def prepare_record():
    require_root_journal(NATIVE)
    record = read_json(NATIVE)
    if record is None:
        return {"status": "ABSENT", "sha256": None, "review_sha256": None}
    status = record.get("Status", record.get("status"))
    plan = record.get("Plan", record.get("plan")) or {}
    approval = plan.get("ReviewSHA256", plan.get("review_sha256"))
    if record.get("Schema", record.get("schema")) != 1 or \
            status not in ("APPLIED", "ROLLED_BACK") or not isinstance(approval, str) or \
            not VALID_SHA.fullmatch(approval):
        raise UninstallError("unrecognized native preparation journal (preserve it)")
    return {"status": status, "sha256": file_fingerprint(NATIVE)["sha256"],
            "review_sha256": approval}


def native_retirement_precheck(project, native, binary, observer=None):
    """Use the exact installed Go reader; no independent rollback engine."""
    if native["status"] == "ABSENT":
        return None
    if not binary or file_fingerprint(BINARY) != binary:
        raise UninstallError("NATIVE_RETIREMENT_CHECK_UNVERIFIED: exact installed binary required")
    require_root_owned(BINARY)
    args = ["native-prepare-retirement-check", "-project", project,
            "-native-prepare-sha256", native["review_sha256"]]
    if observer:
        # Historical RC13 has no retirement-check CLI. A separately pinned,
        # root-trusted current reader is used ONLY for this observation.
        with maintenance_observer_fd(observer["path"], observer["sha256"]) as (fd, _):
            run = subprocess.run(["/proc/self/fd/" + str(fd), *args], pass_fds=(fd,),
                                 env={"PATH": "/usr/sbin:/usr/bin:/sbin:/bin", "HOME": "/root", "LANG": "C"},
                                 capture_output=True, text=True, timeout=90, check=False)
            output, code = run.stdout, run.returncode
            if code not in (0, 1) or len(output) > 65536:
                raise UninstallError("OFFLINE_NATIVE_READER_UNAVAILABLE")
    else:
        output, code = command([str(BINARY), *args], acceptable=(0, 1), timeout=90)
    try:
        report = json.loads(output)
    except ValueError as exc:
        raise UninstallError("NATIVE_RETIREMENT_CHECK_UNVERIFIED: use the matching installed Release uninstaller for older builds; preserve original state") from exc
    if (not isinstance(report, dict) or report.get("project") != project or
            report.get("review_sha256") != native["review_sha256"] or
            report.get("journal_status") != native["status"]):
        raise UninstallError("NATIVE_RETIREMENT_CHECK_UNVERIFIED: native review identity differs")
    if report.get("reason") == "NATIVE_CONFIG_DRIFT":
        raise UninstallError("NATIVE_CONFIG_DRIFT: Project/Profile/ACL differs from reviewed preparation; inspect original journal and local Doctor; no new retirement effects authorized")
    if (code != 0 or report.get("ready") is not True or report.get("reason") != "OBSERVED" or
            not VALID_SHA.fullmatch(str(report.get("observation_sha256", "")))):
        raise UninstallError("NATIVE_RETIREMENT_CHECK_UNVERIFIED: preserve original native evidence")
    if prepare_record() != native or file_fingerprint(BINARY) != binary:
        raise UninstallError("native retirement evidence changed during read")
    return report


def check_original_native_precondition(plan):
    if plan["scope"] != "full":
        return
    native = plan["native_prepare"]
    if prepare_record() != native:
        raise UninstallError("native journal changed after approval")
    observed = native_retirement_precheck(plan["project"], native, plan["binary"], plan.get("offline_observer")) if plan.get("offline_observer") else native_retirement_precheck(plan["project"], native, plan["binary"])
    if "native_retirement_precheck" in plan and observed != plan["native_retirement_precheck"]:
        raise UninstallError("NATIVE_CONFIG_DRIFT: native retirement observation changed after approval")


def systemd_status():
    state, code = command(["systemctl", "is-active", "boxweave.service"], acceptable=(0, 3, 4))
    if code == 0 and state != "active":
        raise UninstallError("unexpected service manager status")
    if state not in ("active", "inactive", "failed", "unknown"):
        raise UninstallError("unrecognized service status: " + state)
    return state


def effective_unit():
    raw, _ = command(["systemctl", "show", "boxweave.service",
                      "-p", "FragmentPath", "-p", "DropInPaths", "--no-pager"])
    pairs = dict(x.split("=", 1) for x in raw.splitlines() if "=" in x)
    if "FragmentPath" not in pairs or "DropInPaths" not in pairs:
        raise UninstallError("effective systemd unit cannot be verified")
    if pairs["DropInPaths"]:
        raise UninstallError("unreviewed systemd drop-ins present: boxweave.service; inspect doctor --maintenance --summary and effective DropInPaths; preserve inherited policy")
    return pairs["FragmentPath"]


# Narrow retirement contract for the exact Ubuntu systemd-255 distrobuilder
# LXC generated policy. It is NOT a runtime-hardening waiver, an install
# activation check or a generic drop-in allowlist. Live drain additionally
# requires the exact approved scope and independent Go worker proof. Never change the inherited file. New variants fail closed.
LXC_SERVICE_DROPIN = Path("/run/systemd/system/service.d/zzz-lxc-service.conf")
LXC_SERVICE_DROPIN_SHA = "fcbd512cb8212c6f5720f28398f01e85ff96246aebb73e09cf449d2f3aa2c5b0"

def lxc_retirement_policy():
    virt, code = command(["systemd-detect-virt", "--container"], acceptable=(0, 1))
    if code or virt.strip() != "lxc":
        raise UninstallError("LXC_EMPTY_RETIREMENT_CONTAINER_UNVERIFIED")
    for kind in ("uid", "gid"):
        raw = Path("/proc/self/" + kind + "_map").read_text().splitlines()
        rows = [tuple(map(int, line.split())) for line in raw]
        if not rows or any(len(row) != 3 for row in rows) or rows[0][0] != 0 or rows[0][1] <= 0:
            raise UninstallError("LXC_EMPTY_RETIREMENT_PARENT_IDENTITY_UNVERIFIED")
    path = LXC_SERVICE_DROPIN
    for parent in reversed(path.parents):
        st = parent.lstat()
        if not stat.S_ISDIR(st.st_mode) or st.st_uid != 0 or st.st_mode & 0o022:
            raise UninstallError("LXC_EMPTY_RETIREMENT_POLICY_PARENT_UNSAFE")
    before = path.lstat()
    if (not stat.S_ISREG(before.st_mode) or before.st_uid != 0 or before.st_gid != 0 or
            stat.S_IMODE(before.st_mode) != 0o644 or before.st_nlink != 1 or before.st_size > 4096):
        raise UninstallError("LXC_EMPTY_RETIREMENT_POLICY_UNSAFE")
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    try:
        current = os.fstat(fd)
        data = os.read(fd, 4097)
        after = os.fstat(fd)
        def stable(st):
            return (st.st_dev,st.st_ino,st.st_mode,st.st_uid,st.st_gid,st.st_nlink,
                    st.st_size,st.st_mtime_ns,st.st_ctime_ns)
        if stable(current) != stable(before) or stable(current) != stable(after) or stable(path.lstat()) != stable(after) or sha(data) != LXC_SERVICE_DROPIN_SHA:
            raise UninstallError("LXC_EMPTY_RETIREMENT_POLICY_CHANGED_OR_UNKNOWN")
    finally:
        os.close(fd)
    # Generators may rewrite identical bytes during daemon-reload. Bind
    # semantic content/path/mode, not a transient tmpfs inode.
    return {"path": str(path), "sha256": sha(data), "mode": 0o644, "kind": "EXACT_LXC255_SCOPED_RETIREMENT"}

def scoped_effective_review(path, allow_lxc=False):
    props = scoped_properties(path.name, "FragmentPath", "DropInPaths")
    if props["FragmentPath"] != str(path):
        raise UninstallError("SCOPED_EFFECTIVE_FRAGMENT_UNVERIFIED:" + path.name)
    if not props["DropInPaths"]:
        return None
    if not allow_lxc or props["DropInPaths"] != str(LXC_SERVICE_DROPIN):
        raise UninstallError("SCOPED_EFFECTIVE_DROPIN_UNREVIEWED:" + path.name)
    proof = lxc_retirement_policy()
    # Exact root-owned fragment is independently matched to manifest /
    # installed Go Status. The fixed policy changes no commands, users,
    # dependencies or stop hooks. Verify manager state is current and
    # ExecStart is still the exact fragment command.
    names = ("FragmentPath", "DropInPaths", "LoadState", "NeedDaemonReload",
             "Type", "User", "Group", "ExecStart", "KillMode", "UMask",
             "RequiredBy", "BoundBy", "RequisiteOf", "ConsistsOf", "PropagatesStopTo",
             "NoNewPrivileges", "ProtectSystem", "ProtectHome", "PrivateTmp",
             "CapabilityBoundingSet", "AmbientCapabilities")
    q = scoped_properties(path.name, *names)
    info = checked_file(path, "file", 65536)
    if info is None or info.st_uid != 0 or info.st_nlink != 1 or stat.S_IMODE(info.st_mode) != 0o644:
        raise UninstallError("SCOPED_LXC_FRAGMENT_UNSAFE")
    fragment = path.read_text().splitlines()
    starts = [line.removeprefix("ExecStart=") for line in fragment if line.startswith("ExecStart=")]
    allowed = {SCOPED_GUARD_UNIT.name: {"incus.service", SCOPED_LIFECYCLE_UNIT.name},
               UNIT.name: set(), SCOPED_LIFECYCLE_UNIT.name: set()}[path.name]
    if (q["FragmentPath"] != str(path) or q["DropInPaths"] != str(LXC_SERVICE_DROPIN) or
            q["LoadState"] != "loaded" or q["NeedDaemonReload"] != "no" or
            len(starts) != 1 or q["ExecStart"].count("argv[]=") != 1 or
            "{ path=/usr/local/bin/boxweave ; argv[]=" + starts[0] + " ; ignore_errors=no ;" not in q["ExecStart"] or
            q["KillMode"] != "control-group" or q["UMask"] != "0077" or q["Group"] or
            q["Type"] != ("oneshot" if path == SCOPED_GUARD_UNIT else "simple") or
            q["User"] != ("boxweave" if path == UNIT else "") or
            any(set(q[key].split()) - allowed for key in ("RequiredBy", "BoundBy", "RequisiteOf", "ConsistsOf", "PropagatesStopTo"))):
        raise UninstallError("SCOPED_LXC_EFFECTIVE_CONTRACT_UNVERIFIED:" + path.name)
    if path == UNIT and (q["CapabilityBoundingSet"] or q["AmbientCapabilities"] or
            q["ProtectSystem"] != "strict" or q["ProtectHome"] != "yes" or q["PrivateTmp"] != "yes"):
        raise UninstallError("SCOPED_LXC_CONTROLLER_IDENTITY_UNVERIFIED")
    # NNP=no is deliberately disclosed; the only permitted operation here
    # is to retire an already installed controller under its exact approved scope. Do not
    # treat this as a successful runtime security attestation.
    if q["NoNewPrivileges"] != "no" or lxc_retirement_policy() != proof:
        raise UninstallError("SCOPED_LXC_EFFECTIVE_POLICY_DRIFT")
    return proof

def scoped_effective_reviews(allow_lxc=False, allow_removed=False):
    out = {}
    for path in (UNIT, SCOPED_GUARD_UNIT, SCOPED_LIFECYCLE_UNIT):
        if allow_removed and not os.path.lexists(path):
            q = scoped_properties(path.name, "LoadState", "ActiveState", "MainPID", "FragmentPath", "DropInPaths")
            if q != {"LoadState":"not-found", "ActiveState":"inactive", "MainPID":"0", "FragmentPath":"", "DropInPaths":""}:
                raise UninstallError("SCOPED_REMOVED_EFFECTIVE_UNIT_UNVERIFIED:" + path.name)
            continue
        proof = scoped_effective_review(path, allow_lxc)
        if proof:
            out[path.name] = proof
    return out

def scoped_verify_effective_reviews(plan, allow_removed=False):
    expected = plan.get("scoped_effective_reviews", {})
    # Match the complete approved unit body, not just the effective command.
    # Keep clean-host historical cleanup/resume semantics unchanged.
    if not expected and allow_removed:
        return
    if expected and any(proof != lxc_retirement_policy() for proof in expected.values()):
        raise UninstallError("SCOPED_EFFECTIVE_RETIREMENT_POLICY_DRIFT")
    for path in (UNIT, SCOPED_GUARD_UNIT, SCOPED_LIFECYCLE_UNIT):
        if allow_removed and not os.path.lexists(path):
            continue
        original = plan["unit"] if path == UNIT else plan["scoped_units"][str(path)]
        if file_fingerprint(path) != original:
            raise UninstallError("SCOPED_EFFECTIVE_UNIT_BODY_DRIFT:" + path.name)
        require_root_owned(path)
    actual = scoped_effective_reviews(bool(expected), allow_removed)
    if allow_removed:
        expected = {name: proof for name, proof in expected.items()
                    if os.path.lexists(UNIT.parent / name)}
    if actual != expected:
        raise UninstallError("SCOPED_EFFECTIVE_RETIREMENT_REVIEW_DRIFT")


def retirement_unit_body(project, bridge, acl):
    for value in (project, bridge, acl):
        if not isinstance(value, str) or not re.fullmatch(r"[a-zA-Z0-9_-]+", value):
            raise UninstallError("unsafe retirement unit scope")
    return ("[Unit]\nDescription=BoxWeave exact retired IPv4 verifier\n"
            "After=incus.service boxweave.service\nRequires=incus.service\n"
            "PartOf=boxweave.service\nBindsTo=boxweave.service\n\n"
            "[Service]\nType=simple\nUser=root\nGroup=root\nUMask=0077\n"
            "ExecStart=/usr/local/bin/boxweave native-peer-root-serve -db /var/lib/boxweave/ledger.sqlite "
            "-project " + project + " -network " + bridge + " -acl " + acl +
            " -retirement-review -retirement-compact -approve-local\n"
            "Restart=on-failure\nRestartSec=1s\nNoNewPrivileges=true\nPrivateTmp=true\n"
            "ProtectHome=true\nProtectSystem=full\nRestrictAddressFamilies=AF_UNIX AF_NETLINK\n"
            "\n[Install]\nWantedBy=multi-user.target\n").encode()


def retirement_service_state():
    raw, _ = command(["systemctl", "show", RETIREMENT_UNIT.name,
                      "-p", "FragmentPath", "-p", "DropInPaths",
                      "-p", "ActiveState", "-p", "MainPID", "--no-pager"])
    pairs = dict(x.split("=", 1) for x in raw.splitlines() if "=" in x)
    if (pairs.get("FragmentPath") != str(RETIREMENT_UNIT) or
            pairs.get("DropInPaths") != "" or
            pairs.get("ActiveState") not in ("active", "inactive", "failed") or
            not re.fullmatch(r"[0-9]+", pairs.get("MainPID", ""))):
        raise UninstallError("effective retirement verifier unit cannot be verified; inspect doctor --maintenance --summary and effective DropInPaths; preserve inherited policy")
    if (int(pairs["MainPID"]) > 1) != (pairs["ActiveState"] == "active"):
        raise UninstallError("retirement verifier process state ambiguous")
    return pairs["ActiveState"]


def inspect_retirement_unit(manifest, project):
    proof = file_fingerprint(RETIREMENT_UNIT)
    digest = manifest.get("retirement_unit_sha256", "")
    if not digest:
        if (proof is not None or os.path.lexists(RETIREMENT_SOCKET) or
                any(manifest.get(x) for x in ("retirement_bridge", "retirement_acl", "retirement_native_review_sha256"))):
            raise UninstallError("unmanaged retirement verifier; inspect original unit")
        return None
    if proof is None or proof["sha256"] != digest or proof["mode"] != 0o644:
        raise UninstallError("installer retirement verifier identity changed")
    require_root_owned(RETIREMENT_UNIT)
    if RETIREMENT_UNIT.lstat().st_nlink != 1:
        raise UninstallError("retirement verifier unit hardlink refused")
    expected = retirement_unit_body(project, manifest.get("retirement_bridge"), manifest.get("retirement_acl"))
    if sha(expected) != digest:
        raise UninstallError("retirement verifier differs from exact installer unit")
    require_root_journal(NATIVE)
    record = read_json(NATIVE) or {}
    plan = record.get("plan") or {}
    target = plan.get("target") or {}
    if (record.get("status") not in ("APPLIED", "ROLLED_BACK") or
            plan.get("review_sha256") != manifest.get("retirement_native_review_sha256") or
            target.get("project") != project or
            target.get("network") != manifest.get("retirement_bridge") or
            target.get("acl") != manifest.get("retirement_acl")):
        raise UninstallError("retirement verifier lacks original prepared scope")
    current = retirement_service_state()
    if os.path.lexists(RETIREMENT_SOCKET):
        info = RETIREMENT_SOCKET.lstat()
        if (current != "active" or not stat.S_ISSOCK(info.st_mode) or info.st_uid != 0 or
                stat.S_IMODE(info.st_mode) != 0o660):
            raise UninstallError("untrusted or stale retirement verifier socket")
    elif current == "active":
        raise UninstallError("active retirement verifier socket missing")
    return {"unit": proof, "state": current}


def require_retirement_stopped(plan):
    proof = plan.get("retirement_verifier")
    if not proof:
        return
    if (file_fingerprint(RETIREMENT_UNIT) != proof["unit"] or
            retirement_service_state() not in ("inactive", "failed") or os.path.lexists(RETIREMENT_SOCKET)):
        raise UninstallError("retirement verifier not stopped or changed; preserve evidence")


def audit_status():
    record = read_json(AUDIT)
    if record is None:
        return {"status": "ABSENT", "sha256": None}
    info = AUDIT.lstat()
    require_root_owned(AUDIT)
    if stat.S_IMODE(info.st_mode) != 0o600:
        raise UninstallError("unsafe permissions on uninstall audit")
    if record.get("status") not in ("STARTED", "COMPLETE"):
        raise UninstallError("unrecognized interrupted uninstall audit")
    return {"status": record["status"], "sha256": file_fingerprint(AUDIT)["sha256"]}


def inspect_installer_backup(project):
    """Verify Go's previous-installation rollback receipt and exact private files."""
    info = checked_file(BACKUP, "dir")
    if info is None:
        return None
    require_root_owned(BACKUP)
    if stat.S_IMODE(info.st_mode) != 0o700:
        raise UninstallError("unsafe installer backup directory")
    receipt_path = BACKUP / "receipt.json"
    receipt = read_json(receipt_path)
    if not isinstance(receipt, dict) or receipt.get("Schema") != 1:
        raise UninstallError("unknown installer backup receipt")
    previous = receipt.get("Previous")
    if not isinstance(previous, dict) or any(previous.get(k) != v for k, v in {
        "schema": 2, "native_project": project, "native_socket": "/var/lib/incus/unix.socket.user",
        "service_user": "boxweave", "binary_path": str(BINARY), "state_dir": str(STATE),
    }.items()):
        raise UninstallError("installer backup belongs to a foreign installation")
    expected = {
        "binary": previous.get("binary_sha256"),
        "unit": receipt.get("PreviousUnitSHA256"),
    }
    for name, field in (("ledger.sqlite", "PreviousDBSHA256"),
                        ("ledger.sqlite-wal", "PreviousWALSHA256"),
                        ("ledger.sqlite-shm", "PreviousSHMSHA256")):
        if receipt.get(field):
            expected[name] = receipt[field]
    if any(not isinstance(v, str) or not VALID_SHA.fullmatch(v) for v in expected.values()):
        raise UninstallError("installer backup digest not verified")
    names = set(expected) | {"receipt.json"}
    if {entry.name for entry in BACKUP.iterdir()} != names:
        raise UninstallError("unrecognized installer backup file inventory")
    files = []
    for name in sorted(names):
        path = BACKUP / name
        info = checked_file(path, "file", 120 * 1024 * 1024 if name in ("binary", "unit", "receipt.json") else None)
        if info is None or info.st_nlink != 1 or stat.S_IMODE(info.st_mode) != 0o600:
            raise UninstallError("unsafe installer backup file")
        require_root_owned(path)
        proof = file_fingerprint(path)
        if name in expected and proof["sha256"] != expected[name]:
            raise UninstallError("installer backup checksum mismatch: " + name)
        files.append({"name": name, "info": proof})
    return {"files": files, "receipt_sha256": file_fingerprint(receipt_path)["sha256"]}


def inspect_exec_archive(uid):
    root = STATE / EXEC_ARCHIVE_NAME
    info = checked_file(root, "dir")
    if info is None:
        return None
    if info.st_uid != uid or stat.S_IMODE(info.st_mode) != 0o700:
        raise UninstallError("unsafe exec archive directory")
    records = []
    total = 0
    for entry in sorted(root.iterdir(), key=lambda x: x.name):
        if len(records) >= 4096 or not EXEC_ARCHIVE_RE.fullmatch(entry.name):
            raise UninstallError("unrecognized or oversized exec archive inventory")
        st = checked_file(entry, "file", 1024 * 1024)
        if st is None or st.st_uid != uid or st.st_nlink != 1 or stat.S_IMODE(st.st_mode) & 0o077:
            raise UninstallError("unsafe exec archive entry")
        total += st.st_size
        if total > 256 * 1024 * 1024:
            raise UninstallError("exec archive exceeds verified retention bound")
        records.append({"name": entry.name, "info": file_fingerprint(entry)})
    return {"device": info.st_dev, "inode": info.st_ino, "files": records}


def inspect(scope, project, box_uuids, native_sha, guided=False, scoped=False, offline_observer=None, offline=False, scoped_empty_lxc=False):
    if not scoped:
        refuse_scoped_retirement()
    if not VALID_NAME.fullmatch(project):
        raise UninstallError("exact valid --project required")
    if len(box_uuids) != len(set(box_uuids)) or any(not VALID_UUID.fullmatch(x) for x in box_uuids):
        raise UninstallError("full scope requires unique exact native UUID allowlist")
    if scope != "full" and (box_uuids or native_sha):
        raise UninstallError("Box/native options are only permitted with --scope full")
    host = machine_sha()
    owner = pwd.getpwnam("boxweave")
    if owner.pw_uid == 0 or project != "user-" + str(owner.pw_uid):
        raise UninstallError("Project is not exact restricted service UID Project")
    state_info = validate_state_dir(STATE, owner.pw_uid)
    manifest = read_json(STATE / "install-manifest.json")
    if manifest is None:
        raise UninstallError("no verified BoxWeave installer manifest (legacy/unmanaged removal blocked)")
    if manifest.get("schema") != 2 or manifest.get("native_project") != project or \
            manifest.get("service_user") != "boxweave" or \
            manifest.get("binary_path") != str(BINARY) or \
            manifest.get("state_dir") != str(STATE) or \
            manifest.get("native_socket") != "/var/lib/incus/unix.socket.user":
        raise UninstallError("installer ownership/provenance mismatch")
    if scoped and manifest.get("scoped_peer_mode") != "SCOPED_PEER_FRESH_V2":
        raise UninstallError("exact installed Scoped Peer v2 manifest required")
    # An original v1 install's exact unit and binary hashes must still match.
    binary = file_fingerprint(BINARY)
    unit = file_fingerprint(UNIT)
    if (binary is None) != (unit is None):
        raise UninstallError("partial/unmanaged binary/unit detected")
    if binary and (binary["sha256"] != manifest.get("binary_sha256") or
                   unit["sha256"] != manifest.get("unit_sha256")):
        raise UninstallError("owned binary or systemd unit modified after install")
    if binary:
        require_root_owned(BINARY)
        require_root_owned(UNIT)
    service = systemd_status()
    if scoped_empty_lxc and scoped:
        scoped_effective_review(UNIT, True)
        fragment = str(UNIT)
    else:
        fragment = effective_unit()
    if (unit and fragment != str(UNIT)) or (not unit and fragment not in ("", str(UNIT))):
        raise UninstallError("effective systemd unit is foreign or unexpectedly replaced")
    if binary is None and service == "active":
        raise UninstallError("unit is absent but service still active")
    if guided and os.path.lexists("/run/boxweave-native-peer-root.sock"):
        raise UninstallError("Root peer socket present: reconcile exact original before retirement")
    if checked_file(ROOT_UNIT, "file") is not None:
        raise UninstallError("separate root peer service exists: manual original audit required")
    retirement = inspect_retirement_unit(manifest, project)
    files = []
    exec_archive = inspect_exec_archive(owner.pw_uid) if guided else None
    installer_backup = inspect_installer_backup(project) if guided and scope == "full" else None
    for entry in sorted(STATE.iterdir(), key=lambda x: x.name):
        if guided and entry.name == EXEC_ARCHIVE_NAME:
            if exec_archive is None:
                raise UninstallError("exec archive changed during planning")
            continue
        if entry.name not in STATE_FILES:
            raise UninstallError("unrecognized owned state file: " + entry.name)
        info = (verified_serve_lock(entry, owner.pw_uid)
                if entry.name == SERVE_LOCK_NAME else checked_file(entry, "file"))
        if stat.S_IMODE(info.st_mode) & 0o077:
            raise UninstallError("controller state file has unsafe permissions: " + entry.name)
        files.append({"name": entry.name, "info": file_fingerprint(entry)})
    snapshot = inspect_sqlite(STATE / "ledger.sqlite", guided=guided, cold=True) if offline else inspect_sqlite(STATE / "ledger.sqlite", guided=guided)
    if any(x["project"] != project for x in snapshot["boxes"]):
        raise UninstallError("foreign Layer in owned Box inventory")
    native = prepare_record()
    require_root_journal(SCRIPTLET)
    scriptlet = read_json(SCRIPTLET)
    if scriptlet is not None and str(scriptlet.get("Status", scriptlet.get("status", ""))) != "ROLLED_BACK":
        # Scriptlet has its own root-local journal; never hand-edit global ACL.
        raise UninstallError("active/unknown global Scriptlet journal: rollback separately")
    blockers = list(snapshot["blockers"])
    audit = audit_status()
    if audit["status"] == "STARTED":
        blockers.append("PRIOR_UNINSTALL_INTERRUPTED_REVIEW_ORIGINAL_AUDIT")
    if scope in ("controller", "full") and snapshot["boxes"] and scope != "full":
        blockers.append("OWNED_BOXES_REMAIN:" + str(len(snapshot["boxes"])))
    if scope != "full" and native["status"] == "APPLIED":
        blockers.append("NATIVE_PREPARE_APPLIED: retain installed rollback binary or choose full")
    incus = []
    if scope == "full":
        if guided:
            if scoped:
                require_quiescent_scoped_native(project)
            else:
                require_quiescent_native(project)
        incus = native_instances(project, guided=guided)
        expected = sorted(x["uuid"] for x in snapshot["boxes"]) if guided else sorted(box_uuids)
        if guided and box_uuids and sorted(box_uuids) != expected:
            blockers.append("EXPLICIT_ALLOWLIST_MISMATCH")
        current = sorted(x["uuid"] for x in incus)
        db_current = sorted(x["uuid"] for x in snapshot["boxes"])
        if current != db_current or expected != current:
            blockers.append("INCUS_DB_ALLOWLIST_MISMATCH")
        if guided and ({(x["uuid"], x["name"]) for x in incus} !=
                       {(x["uuid"], x["name"]) for x in snapshot["boxes"] if x["state"] == "OWNED"}):
            blockers.append("INCUS_DB_NAME_OR_STATE_MISMATCH")
        if any(x["state"] not in (("Stopped", "Running") if guided else ("Stopped",)) for x in incus):
            blockers.append("NATIVE_BOX_NOT_STOPPED")
        if guided and native_sha is None and native["status"] in ("APPLIED", "ROLLED_BACK"):
            native_sha = native["review_sha256"]
        if native["status"] in ("APPLIED", "ROLLED_BACK") and native_sha != native["review_sha256"]:
            blockers.append("NATIVE_PREPARE_DIGEST_REQUIRED")
        if native["status"] == "ABSENT" and native_sha:
            blockers.append("NATIVE_PREPARE_DIGEST_WITHOUT_JOURNAL")
    # Upgrade backups are separate and may contain credential/database copies.
    if scope != "service" and BACKUP.exists() and installer_backup is None:
        blockers.append("INSTALLER_BACKUP_REQUIRES_SEPARATE_RETENTION_REVIEW")
    native_precheck = (native_retirement_precheck(project, native, binary, offline_observer) if offline_observer else native_retirement_precheck(project, native, binary)) if scope == "full" else None
    inputs = {
        "format": 1, "scope": scope, "guided": guided, "project": project, "host_machine_id_sha256": host,
        "service_user_uid": owner.pw_uid, "service_state": service,
        "unit": unit, "binary": binary, "state_files": files, "exec_archive": exec_archive,
        "installer_backup": installer_backup,
        "retirement_verifier": retirement,
        "effective_unit": fragment, "uninstall_audit": audit,
        "sqlite": snapshot, "native_prepare": native, "native_retirement_precheck": native_precheck,
        "scriptlet_present": scriptlet is not None,
        "incus_instances": incus, "allowlist": expected if scope == "full" else [],
        "native_approval": native_sha or "", "blockers": sorted(set(blockers)),
    }
    review = sha(canonical(inputs))
    actions = ["stop exact boxweave.service", "disable exact boxweave.service",
               "unlink validated /etc/systemd/system/boxweave.service",
               "unlink validated /usr/local/bin/boxweave", "systemctl daemon-reload"]
    if retirement:
        actions.insert(1, "verify stopped and remove exact installer-owned IPv4 retirement verifier")
    if scope == "full":
        actions = ["delete exact stopped, DB-owned, explicitly allowlisted Incus UUIDs",
                   "rollback/archive exact approved native preparation journal"] + actions
    if guided:
        actions = ["archive original controller files under root-only retirement directory",
                   "gracefully stop exact approved running Boxes before deletion"] + actions
        if installer_backup:
            actions.insert(1, "archive exact verified prior installer backup before retiring original")
    if scope in ("controller", "full"):
        actions += ["unlink only recognized files from /var/lib/boxweave",
                    "rmdir empty /var/lib/boxweave"]
    return {**inputs, "review_sha256": review, "actions": actions,
            "ready": not blockers, "retained": [
                "Incus daemon/global default storage pool/all other Projects",
                "managed/shared bridge, image cache and ACL except verified native rollback",
                "system boxweave user/group, external Cloudflare tunnel and ChatGPT connector",
                "closed native journals and any historical audit files",
            ]}


def _replace_audit(record):
    if len(canonical(record)) + 1 > 1024 * 1024:
        raise UninstallError("uninstall audit retention bound exceeded")
    old = checked_file(AUDIT, "file", 1024 * 1024)
    if old is None or stat.S_IMODE(old.st_mode) != 0o600:
        raise UninstallError("untrusted existing root uninstall audit")
    require_root_owned(AUDIT)
    fd, tmp = tempfile.mkstemp(dir=str(AUDIT.parent), prefix=".boxweave-uninstall-")
    try:
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, "wb") as dst:
            dst.write(canonical(record) + b"\n")
            dst.flush()
            os.fsync(dst.fileno())
        os.replace(tmp, AUDIT)
        rootdir = os.open(str(AUDIT.parent), os.O_RDONLY | getattr(os, "O_DIRECTORY", 0))
        try:
            os.fsync(rootdir)
        finally:
            os.close(rootdir)
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)


def audited_write(event):
    previous = read_json(AUDIT)
    if previous is not None:
        if previous.get("status") != "COMPLETE":
            raise UninstallError("previous uninstall interrupted; review original audit, never replay")
        if len(previous.get("history", [])) > 100:
            raise UninstallError("audit retention requires manual archiving")
        history = list(previous.get("history", []))
        history.append({k: v for k, v in previous.items() if k != "history"})
        record = {**event, "history": history, "status": "STARTED"}
        _replace_audit(record)
        return
    record = {**event, "status": "STARTED", "history": []}
    payload = canonical(record) + b"\n"
    if len(payload) > 1024 * 1024:
        raise UninstallError("uninstall audit retention bound exceeded")
    fd = os.open(str(AUDIT), os.O_WRONLY | os.O_CREAT | os.O_EXCL |
                 getattr(os, "O_NOFOLLOW", 0), 0o600)
    with os.fdopen(fd, "wb") as dst:
        dst.write(payload)
        dst.flush()
        os.fsync(dst.fileno())
    rootdir = os.open(str(AUDIT.parent), os.O_RDONLY | getattr(os, "O_DIRECTORY", 0))
    try:
        os.fsync(rootdir)
    finally:
        os.close(rootdir)


def audited_complete(review, result):
    record = read_json(AUDIT)
    if record is None or record.get("status") != "STARTED" or record.get("review_sha256") != review:
        raise UninstallError("original uninstall audit changed before completion")
    record["status"] = "COMPLETE"
    record["result"] = result
    _replace_audit(record)


def exact_unlink(path, fingerprint):
    current = file_fingerprint(path)
    if fingerprint is None:
        if current is not None:
            raise UninstallError("unexpected newly installed path: " + str(path))
        return
    if current != fingerprint:
        raise UninstallError("file identity changed before unlink: " + str(path))
    path.unlink()


@contextmanager
def hold_stopped_serve_lock(plan):
    """The persistent lock inode is never unlinked while another serve owns it.

    Keep our exclusive flock through state removal. The file must be removed
    LAST, after the DB and other state, to minimize split-lock exposure.
    """
    path = STATE / SERVE_LOCK_NAME
    original = next((x["info"] for x in plan["state_files"]
                     if x["name"] == SERVE_LOCK_NAME), None)
    if original is None:
        if checked_file(path, "file") is not None:
            raise UninstallError("serve lock appeared after review")
        yield
        return
    if verified_serve_lock(path, plan["service_user_uid"]) is None or file_fingerprint(path) != original:
        raise UninstallError("serve lock inode changed since plan")
    flags = os.O_RDWR | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
    fd = os.open(str(path), flags)
    try:
        state = os.fstat(fd)
        if (state.st_dev != original["device"] or state.st_ino != original["inode"]
                or state.st_nlink != 1 or stat.S_IMODE(state.st_mode) != 0o600):
            raise UninstallError("serve lock changed while opening")
        require_serve_lock_fd_owner(fd, plan["service_user_uid"])
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as exc:
            raise UninstallError("serve lock held by another process; refusing uninstall") from exc
        yield
    finally:
        os.close(fd)


def apply(plan):
    refuse_scoped_retirement()
    check_original_native_precondition(plan)
    # At this point the operator has already approved both exact digests;
    # no automatic retries or implicit "yes" on modified host state.
    if plan["service_state"] == "active":
        command(["systemctl", "stop", "boxweave.service"])
    if systemd_status() == "active":
        raise UninstallError("controller did not stop")
    require_retirement_stopped(plan)
    with hold_stopped_serve_lock(plan):
        return apply_with_serve_lock(plan)


def archive_controller(plan, reviewed_plan=None):
    # Preserve originals before native deletion. Separate names avoid
    # directory/binary basename collisions.
    RETIRED.mkdir(mode=0o700, exist_ok=True)
    info = RETIRED.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.geteuid() or stat.S_IMODE(info.st_mode) != 0o700:
        raise UninstallError("unsafe retirement archive directory")
    dest = Path(tempfile.mkdtemp(prefix="retired-", dir=RETIRED))
    items = [(STATE / x["name"], dest / ("state-" + x["name"])) for x in plan["state_files"]]
    if plan.get("exec_archive"):
        items += [(STATE / EXEC_ARCHIVE_NAME / x["name"], dest / ("exec-" + x["name"]))
                  for x in plan["exec_archive"]["files"]]
    items += [(BINARY, dest / "boxweave.bin"), (UNIT, dest / "boxweave.service"),
              (NATIVE, dest / "native-prepare.json")]
    if plan.get("installer_backup"):
        if inspect_installer_backup(plan["project"]) != plan["installer_backup"]:
            raise UninstallError("installer backup changed before retirement archive")
        items += [(BACKUP / x["name"], dest / ("installer-backup-" + x["name"]))
                  for x in plan["installer_backup"]["files"]]
    if plan.get("retirement_verifier"):
        items += [(RETIREMENT_UNIT, dest / RETIREMENT_UNIT.name)]
    for src, dst in items:
        proof = file_fingerprint(src)
        if proof is None:
            if src.name in ("ledger.sqlite-wal", "ledger.sqlite-shm") or src == NATIVE:
                continue
            raise UninstallError("retirement source disappeared")
        fd = os.open(str(src), os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0))
        with os.fdopen(fd, "rb") as inp, dst.open("xb") as out:
            os.fchmod(out.fileno(), 0o600)
            shutil.copyfileobj(inp, out)
            out.flush()
            os.fsync(out.fileno())
        if file_fingerprint(src) != proof or file_fingerprint(dst)["sha256"] != proof["sha256"]:
            raise UninstallError("retirement copy changed during backup")
    with (dest / "review.json").open("xb") as out:
        os.fchmod(out.fileno(), 0o600)
        out.write(canonical(reviewed_plan or plan) + b"\n")
        out.flush()
        os.fsync(out.fileno())
    for directory in (dest, RETIRED, RETIRED.parent):
        fd = os.open(str(directory), os.O_RDONLY | getattr(os, "O_DIRECTORY", 0))
        try:
            os.fsync(fd)
        finally:
            os.close(fd)
    print("Retirement archive: " + str(dest), flush=True)
    return str(dest)


def apply_with_serve_lock(plan, preserved_archive=None, offline=False):
    refuse_scoped_retirement()
    check_original_native_precondition(plan)
    # Stopping may legitimately checkpoint WAL; recheck native and SQLite
    # semantic blockers, not a fabricated claim that WAL bytes are immutable.
    after = (inspect_sqlite(STATE / "ledger.sqlite", guided=True, cold=True) if offline else
             inspect_sqlite(STATE / "ledger.sqlite", guided=plan.get("guided", False)))
    if offline:
        # The separately approved offline branch preserves the original
        # blocked SQLite snapshot verbatim. It never settles/releases it.
        if plan.get("mode") != OFFLINE_MODE or after != plan["sqlite"] or not offline_ledger_unchanged(plan):
            raise UninstallError("OFFLINE_ORIGINAL_LEDGER_CHANGED")
        offline_inactive(plan["project"], plan, require_disabled=True)
    elif after["operations_sha256"] != plan["sqlite"]["operations_sha256"] or \
            after["boxes"] != plan["sqlite"]["boxes"] or after["blockers"]:
        raise UninstallError("SQLite ledger changed during stop; preserve evidence")
    if plan.get("guided") and inspect_exec_archive(plan["service_user_uid"]) != plan.get("exec_archive"):
        raise UninstallError("exec archive changed after approval")
    archive = preserved_archive or (archive_controller(plan) if plan.get("guided") else None)
    archive_proof = legacy_archive_content(Path(archive), plan, historical=False) if archive else None
    if offline:
        offline_archive_checkpoint(plan, archive)
        remaining = list(plan["incus_instances"])
    if plan["scope"] == "full":
        guided = plan.get("guided", False)
        if guided:
            require_quiescent_native(plan["project"])
        if native_instances(plan["project"], guided=guided) != plan["incus_instances"]:
            raise UninstallError("native inventory changed after approval")
        for box in plan["incus_instances"]:
            if offline:
                offline_inactive(plan["project"], plan, require_disabled=True)
                require_quiescent_native(plan["project"])
                check_original_native_precondition(plan)
                if not offline_ledger_unchanged(plan):
                    raise UninstallError("OFFLINE_ORIGINAL_LEDGER_CHANGED")
            current = native_instances(plan["project"], guided=guided)
            if offline and current != remaining:
                raise UninstallError("OFFLINE_APPROVED_NATIVE_PREFIX_CHANGED")
            if offline:
                offline_native_consumers(plan, remaining)
            exact = [x for x in current if x["uuid"] == box["uuid"]]
            if len(exact) != 1 or exact[0] != box:
                raise UninstallError("Incus target changed before deletion")
            if box["state"] == "Running" and plan.get("guided"):
                command(["incus", "stop", "local:" + box["name"], "--project",
                         plan["project"], "--timeout", "60"], timeout=90)
                stopped = [x for x in native_instances(plan["project"], guided=guided) if x["uuid"] == box["uuid"]]
                if stopped != [{**box, "state": "Stopped"}]:
                    raise UninstallError("graceful stop not independently verified")
            if offline:
                require_quiescent_native(plan["project"])
                expected = [{**x, "state": "Stopped"} if x["uuid"] == box["uuid"] else x for x in remaining]
                if native_instances(plan["project"], guided=True) != expected:
                    raise UninstallError("OFFLINE_APPROVED_NATIVE_STOP_PREFIX_CHANGED")
                offline_inactive(plan["project"], plan, require_disabled=True)
                offline_native_consumers(plan, expected)
                offline_checkpoint(plan, "BOX_STOPPED", current_box=box["uuid"])
            command(["incus", "delete", "local:" + box["name"], "--project", plan["project"]], timeout=90)
            if any(x["uuid"] == box["uuid"] for x in native_instances(plan["project"], guided=guided)):
                raise UninstallError("native deletion not verified")
            if offline:
                remaining = [x for x in remaining if x["uuid"] != box["uuid"]]
                require_quiescent_native(plan["project"])
                if native_instances(plan["project"], guided=True) != remaining:
                    raise UninstallError("OFFLINE_APPROVED_NATIVE_DELETE_PREFIX_CHANGED")
                offline_checkpoint(plan, "BOX_DELETED", deleted=box["uuid"])
        if guided:
            require_quiescent_native(plan["project"])
            if native_instances(plan["project"], guided=True):
                raise UninstallError("native Project no longer empty; preserve controller")
        if offline:
            offline_inactive(plan["project"], plan, require_disabled=True)
            check_original_native_precondition(plan)
            if not offline_ledger_unchanged(plan):
                raise UninstallError("OFFLINE_ORIGINAL_LEDGER_CHANGED")
            offline_native_consumers(plan, [])
            offline_checkpoint(plan, "NATIVE_EMPTY")
        native = plan["native_prepare"]
        if native["status"] == "APPLIED":
            if prepare_record() != native:
                raise UninstallError("native journal changed before rollback")
            command([str(BINARY), "native-prepare-rollback", "-native-prepare-sha256",
                     native["review_sha256"], "-approve-local"], timeout=90)
            record = prepare_record()
            if record["status"] != "ROLLED_BACK" or record["review_sha256"] != native["review_sha256"]:
                raise UninstallError("native rollback unverified; preserve binary")
        if native["status"] in ("APPLIED", "ROLLED_BACK"):
            command([str(BINARY), "native-prepare-archive", "-native-prepare-sha256",
                     native["review_sha256"], "-approve-local"], timeout=90)
            if NATIVE.exists():
                raise UninstallError("native journal archive unverified")
    if offline:
        offline_checkpoint(plan, "NATIVE_RESTORED")
        offline_inactive(plan["project"], plan, require_disabled=True)
        if not offline_ledger_unchanged(plan):
            raise UninstallError("OFFLINE_ORIGINAL_LEDGER_CHANGED")
        offline_checkpoint(plan, "CLEANUP_STARTED")
        return offline_cleanup_files(read_json(AUDIT))
    # Do not remove helper binary until all native rollback/archives succeeded.
    command(["systemctl", "disable", "boxweave.service"])
    if systemd_status() == "active":
        raise UninstallError("service restarted during removal")
    if plan.get("retirement_verifier"):
        require_retirement_stopped(plan)
        command(["systemctl", "disable", RETIREMENT_UNIT.name])
        exact_unlink(RETIREMENT_UNIT, plan["retirement_verifier"]["unit"])
    exact_unlink(UNIT, plan["unit"])
    exact_unlink(BINARY, plan["binary"])
    command(["systemctl", "daemon-reload"])
    if plan["scope"] != "service":
        # Close WAL is already enforced by service stop. Never recurse into
        # unrecognized state or delete symlinks, folders or foreign assets.
        for item in plan["state_files"]:
            # Never unlink an actively held flock inode before DB/state cleanup.
            if item["name"] == SERVE_LOCK_NAME:
                continue
            path = STATE / item["name"]
            if item["name"] == "ledger.sqlite":
                # DB may checkpoint during stop, verify type and ownership.
                current = file_fingerprint(path)
                if current is None:
                    raise UninstallError("SQLite disappeared unexpectedly")
                exact_unlink(path, current)
            elif item["name"] in ("ledger.sqlite-wal", "ledger.sqlite-shm"):
                current = file_fingerprint(path)
                if current is not None:
                    exact_unlink(path, current)
            else:
                exact_unlink(path, item["info"])
        if plan.get("exec_archive"):
            if inspect_exec_archive(plan["service_user_uid"]) != plan["exec_archive"]:
                raise UninstallError("exec archive changed before retirement")
            for item in plan["exec_archive"]["files"]:
                exact_unlink(STATE / EXEC_ARCHIVE_NAME / item["name"], item["info"])
            (STATE / EXEC_ARCHIVE_NAME).rmdir()
        lock = next((x for x in plan["state_files"]
                     if x["name"] == SERVE_LOCK_NAME), None)
        if lock is not None:
            # We hold the exclusive original flock from apply(); safe to
            # remove only after the SQLite and all other state disappeared.
            exact_unlink(STATE / SERVE_LOCK_NAME, lock["info"])
        if list(STATE.iterdir()):
            raise UninstallError("new files appeared in state directory")
        STATE.rmdir()
        backup = plan.get("installer_backup")
        if backup:
            # Preserve the fsynced root-private copy; unlink originals last.
            if inspect_installer_backup(plan["project"]) != backup:
                raise UninstallError("installer backup changed before retirement")
            for item in backup["files"]:
                exact_unlink(BACKUP / item["name"], item["info"])
            BACKUP.rmdir()
    if archive and legacy_archive_content(Path(archive), plan, historical=False) != archive_proof:
        raise UninstallError("retirement archive changed after original copy")
    return {"outcome": "COMPLETE", "scope": plan["scope"],
            "host_machine_id_sha256": plan["host_machine_id_sha256"],
            "approved_review_sha256": plan["review_sha256"],
            "retirement_archive": archive, "archive_content_sha256": archive_proof,
            "retained": plan["retained"]}


def inspect_interrupted_retirement(project, archive_path):
    """Read-only, narrowly bound recovery of a stopped, fully archived phase.

    Only a completed original guided archive, original STARTED audit and
    pre-rollback APPLIED native journal are accepted. No partially dispatched
    native rollback, deleted controller artifact or unknown Box is replayed.
    """
    refuse_scoped_retirement()
    if not VALID_NAME.fullmatch(project):
        raise UninstallError("exact project required for retirement recovery")
    archive = Path(archive_path)
    if not archive.is_absolute() or archive.parent != RETIRED or not re.fullmatch(r"retired-[a-z0-9_]{8}", archive.name):
        raise UninstallError("exact original private retirement archive path required")
    for directory in (RETIRED, archive):
        info = directory.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.geteuid() or stat.S_IMODE(info.st_mode) != 0o700:
            raise UninstallError("untrusted retirement directory")
    audit = read_json(AUDIT)
    if audit_status()["status"] != "STARTED" or not isinstance(audit, dict):
        raise UninstallError("no original STARTED retirement audit")
    if audit.get("mode") == OFFLINE_MODE:
        raise UninstallError("OFFLINE_RETIREMENT_INTERRUPTED: preserve original archive and checkpoint; no ordinary retirement replay")
    review_path = archive / "review.json"
    plan = read_json(review_path)
    if not isinstance(plan, dict):
        raise UninstallError("archived original review is absent")
    proof = file_fingerprint(review_path)
    if proof is None or proof["mode"] != 0o600:
        raise UninstallError("unsafe original review")
    require_root_owned(review_path)
    if (not plan.get("guided") or plan.get("scope") != "full" or
            not plan.get("ready") or plan.get("project") != project or
            plan.get("host_machine_id_sha256") != machine_sha() or
            not VALID_SHA.fullmatch(plan.get("review_sha256", ""))):
        raise UninstallError("archive is not a reviewed full retirement for this Host")
    inputs = {k: v for k, v in plan.items()
              if k not in ("review_sha256", "actions", "ready", "retained")}
    if sha(canonical(inputs)) != plan["review_sha256"]:
        raise UninstallError("original retirement plan hash mismatch")
    if (audit.get("review_sha256") != plan["review_sha256"] or
            audit.get("host_sha256") != plan["host_machine_id_sha256"] or
            audit.get("scope") != "full" or audit.get("actions") != plan["actions"] or
            audit.get("historical_released_exec") != plan["sqlite"]["historical_released_exec"] or
            audit.get("historical_released_other") != plan["sqlite"]["historical_released_other"]):
        raise UninstallError("original STARTED audit and archived plan disagree")
    if plan["incus_instances"] or plan["sqlite"]["boxes"]:
        raise UninstallError("resume does not replay prior Box deletion; manual native review required")
    if plan["native_prepare"]["status"] != "APPLIED" or systemd_status() != "inactive":
        raise UninstallError("resume limited to inactive controller before native rollback")
    now = inspect("full", project, [], None, guided=True)
    if now["blockers"] != ["PRIOR_UNINSTALL_INTERRUPTED_REVIEW_ORIGINAL_AUDIT"]:
        raise UninstallError("unresolved blockers beyond original interrupted audit")
    for key in ("host_machine_id_sha256", "sqlite", "incus_instances", "native_prepare",
                "unit", "binary", "exec_archive", "installer_backup", "allowlist"):
        if now[key] != plan[key]:
            raise UninstallError("retirement recovery source drift: " + key)
    if ((now["retirement_verifier"] or {}).get("unit") !=
            (plan["retirement_verifier"] or {}).get("unit")):
        raise UninstallError("retirement verifier drift")
    for item in plan["state_files"]:
        if item["name"] not in ("ledger.sqlite", "ledger.sqlite-wal", "ledger.sqlite-shm"):
            fresh = next((x for x in now["state_files"] if x["name"] == item["name"]), None)
            if fresh != item:
                raise UninstallError("controller state changed after original archive")
    names = {"review.json", "boxweave.bin", "boxweave.service", "native-prepare.json"}
    proofs = [(archive / "boxweave.bin", BINARY),
              (archive / "boxweave.service", UNIT),
              (archive / "native-prepare.json", NATIVE)]
    for item in plan["state_files"]:
        name = "state-" + item["name"]
        if item["name"] in ("ledger.sqlite-wal", "ledger.sqlite-shm") and not (archive / name).exists():
            if (STATE / item["name"]).exists():
                raise UninstallError("missing original WAL/SHM archive")
            continue
        names.add(name)
        proofs.append((archive / name, STATE / item["name"]))
    for item in (plan.get("exec_archive") or {}).get("files", []):
        name = "exec-" + item["name"]
        names.add(name)
        proofs.append((archive / name, STATE / EXEC_ARCHIVE_NAME / item["name"]))
    for item in (plan.get("installer_backup") or {}).get("files", []):
        name = "installer-backup-" + item["name"]
        names.add(name)
        proofs.append((archive / name, BACKUP / item["name"]))
    if plan.get("retirement_verifier"):
        names.add(RETIREMENT_UNIT.name)
        proofs.append((archive / RETIREMENT_UNIT.name, RETIREMENT_UNIT))
    if {entry.name for entry in archive.iterdir()} != names:
        raise UninstallError("archived original file inventory changed")
    for saved, source in proofs:
        proof = file_fingerprint(saved)
        original = file_fingerprint(source)
        if proof is None or original is None or proof["mode"] != 0o600 or proof["sha256"] != original["sha256"]:
            raise UninstallError("original private retirement copy changed")
        require_root_owned(saved)
    return {"ready": True, "phase": "ARCHIVED_BEFORE_NATIVE_ROLLBACK",
            "project": project, "retirement_archive": str(archive),
            "original_review_sha256": plan["review_sha256"],
            "original_uninstall_audit_sha256": audit_status()["sha256"]}, plan


def resume_interrupted_retirement(plan, archive):
    refuse_scoped_retirement()
    # Caller must have independently approved the matching, fresh read-only phase.
    if systemd_status() != "inactive":
        raise UninstallError("stopped original service required for retirement resume")
    require_retirement_stopped(plan)
    with hold_stopped_serve_lock(plan):
        result = apply_with_serve_lock(plan, preserved_archive=str(archive))
    audited_complete(plan["review_sha256"], result)
    return result



# Scoped retirement is a distinct full-only transaction. Ordinary workload
# retirement remains MCP-owned; never use direct native Delete for scoped Boxes.
SCOPED_MODE = "SCOPED_PEER_FRESH_V2"
SCOPED_PHASES = ("APPROVED", "ARCHIVED", "NATIVE_RESTORED", "GUARD_REMOVED", "CLEANUP_STARTED")


def scoped_private_dir(path):
    info = path.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != 0 or stat.S_IMODE(info.st_mode) != 0o700:
        raise UninstallError("untrusted scoped retirement directory: " + str(path))


def scoped_private_file(path, cap=None):
    info = checked_file(path, "file", cap)
    if info is None or info.st_nlink != 1 or info.st_uid != 0 or stat.S_IMODE(info.st_mode) != 0o600:
        raise UninstallError("untrusted scoped retirement evidence: " + str(path))
    return file_fingerprint(path)


def scoped_sync_dir(path):
    fd = os.open(str(path), os.O_RDONLY | getattr(os, "O_DIRECTORY", 0) | getattr(os, "O_NOFOLLOW", 0))
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def scoped_write_json(path, value):
    payload = canonical(value) + b"\n"
    if len(payload) > 1024 * 1024:
        raise UninstallError("scoped retirement receipt exceeds supported bound")
    fd, temp = tempfile.mkstemp(prefix=".scoped-retirement-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as out:
            os.fchmod(out.fileno(), 0o600)
            out.write(payload)
            out.flush()
            os.fsync(out.fileno())
        os.replace(temp, path)
        scoped_sync_dir(path.parent)
    finally:
        if os.path.exists(temp):
            os.unlink(temp)


@contextmanager
def scoped_lock(path):
    fd = os.open(str(path), os.O_CREAT | os.O_RDWR | getattr(os, "O_NOFOLLOW", 0) |
                 getattr(os, "O_CLOEXEC", 0), 0o600)
    try:
        st = os.fstat(fd)
        if st.st_uid != 0 or st.st_nlink != 1 or not stat.S_ISREG(st.st_mode) or stat.S_IMODE(st.st_mode) != 0o600:
            raise UninstallError("untrusted local maintenance lock")
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as exc:
            raise UninstallError("another local maintenance transaction is active") from exc
        yield
    finally:
        os.close(fd)


def scoped_tree(root):
    scoped_private_dir(root)
    h = hashlib.sha256()
    count, total = 0, 0
    def walk(directory, depth):
        nonlocal count, total
        if depth > 3:
            raise UninstallError("unrecognized scoped Root journal depth")
        scoped_private_dir(directory)
        for child in sorted(directory.iterdir(), key=lambda x: x.name):
            if not re.fullmatch(r"[a-z0-9][a-z0-9_.-]{0,95}", child.name):
                raise UninstallError("unrecognized scoped Root journal name")
            rel = str(child.relative_to(root))
            info = child.lstat()
            if stat.S_ISDIR(info.st_mode):
                h.update(canonical([rel, "dir"]) + b"\n")
                walk(child, depth + 1)
            else:
                proof = scoped_private_file(child, 1024 * 1024)
                count += 1
                total += proof["size"]
                if count > 100000 or total > 256 * 1024 * 1024:
                    raise UninstallError("scoped Root journal exceeds verified archive bound")
                h.update(canonical([rel, proof["sha256"], proof["size"]]) + b"\n")
    walk(root, 0)
    return {"sha256": h.hexdigest(), "files": count, "bytes": total}


def scoped_properties(unit, *props):
    raw, _ = command(["systemctl", "show", unit, *[v for p in props for v in ("-p", p)], "--no-pager"])
    values = dict(line.split("=", 1) for line in raw.splitlines() if "=" in line)
    if set(props) - values.keys():
        raise UninstallError("effective service evidence unavailable: " + unit)
    return values


def scoped_units(allow_empty_lxc=False):
    result = {}
    for path in (SCOPED_GUARD_UNIT, SCOPED_LIFECYCLE_UNIT, SCOPED_INCUS_DROPIN):
        proof = file_fingerprint(path)
        if proof is None or proof["mode"] != 0o644 or path.lstat().st_nlink != 1:
            raise UninstallError("exact scoped unit/drop-in missing or changed")
        require_root_owned(path)
        if path != SCOPED_INCUS_DROPIN:
            p = scoped_properties(path.name, "FragmentPath", "DropInPaths", "ActiveState", "UnitFileState")
            if p["FragmentPath"] != str(path) or (p["DropInPaths"] and not allow_empty_lxc) or p["UnitFileState"] != "static" or p["ActiveState"] not in ("active", "inactive", "failed"):
                raise UninstallError("unrecognized effective scoped service: " + path.name + "; inspect doctor --maintenance --summary and effective DropInPaths; preserve inherited policy")
            if allow_empty_lxc:
                scoped_effective_review(path, True)
        result[str(path)] = proof
    p = scoped_properties("incus.service", "Requires", "After", "ActiveState", "MainPID", "DropInPaths")
    if (p["ActiveState"] != "active" or not p["MainPID"].isdigit() or int(p["MainPID"]) <= 1 or
            SCOPED_GUARD_UNIT.name not in p["Requires"].split() or SCOPED_GUARD_UNIT.name not in p["After"].split() or
            str(SCOPED_INCUS_DROPIN) not in p["DropInPaths"].split()):
        raise UninstallError("effective original Incus guard dependency not verified")
    return result


def scoped_native(plan, action):
    if file_fingerprint(BINARY) != plan["binary"]:
        raise UninstallError("exact installed scoped executable changed")
    require_root_owned(BINARY)
    argv = [str(BINARY), "native-scoped-retirement-" + action, "-project", plan["project"]]
    if action in ("restore", "guard"):
        argv += ["-native-prepare-sha256", plan["scoped_native"]["review_sha256"], "-approve-local"]
    raw, code = command(argv, acceptable=(0, 1), timeout=140)
    if code:
        raise UninstallError("SCOPED_NATIVE_PROOF_UNVERIFIED: inspect matching local Doctor/native retirement plan; preserve original evidence")
    report = json.loads(raw)
    if (not isinstance(report, dict) or report.get("project") != plan["project"] or
            not VALID_SHA.fullmatch(str(report.get("review_sha256", ""))) or
            not re.fullmatch(r"bw_pg_[0-9a-f]{16}", str(report.get("table", ""))) or
            report.get("phase") not in ("INSTALLED", "PROFILE_RESTORED", "PROJECT_RESTORED", "NATIVE_RESTORED", "GUARD_ABSENT")):
        raise UninstallError("scoped native retirement response identity invalid")
    original = plan.get("scoped_native")
    if original and any(report.get(k) != original.get(k) for k in ("project", "review_sha256", "retired", "table")):
        raise UninstallError("scoped native review changed")
    return report


def scoped_plan(project):
    if os.path.lexists(SCOPED_FENCE) or audit_status()["status"] == "STARTED":
        raise UninstallError("SCOPED_RETIREMENT_IN_PROGRESS: review --resume-plan --scoped, do not start another")
    # This alternative is only used for an empty Scoped installation.
    if native_instances(project, guided=False):
        raise UninstallError("SCOPED_LXC_EMPTY_RETIREMENT_REQUIRES_MCP_CLEANUP")
    plan = inspect("full", project, [], None, guided=True, scoped=True, scoped_empty_lxc=True)
    if (plan["binary"] is None or plan["unit"] is None or plan["native_prepare"]["status"] != "ABSENT" or
            plan["retirement_verifier"] or plan["scriptlet_present"] or plan["installer_backup"] or os.path.lexists(BACKUP)):
        raise UninstallError("mixed legacy/scoped installation evidence; no retirement")
    for path in (BINARY, UNIT):
        if path.lstat().st_nlink != 1 or stat.S_IMODE(path.lstat().st_mode) != (0o755 if path == BINARY else 0o644):
            raise UninstallError("untrusted installed file mode or hardlink")
    # Go Status validates exact unit contents, manifest and original Root receipts.
    command([str(BINARY), "install-status", "-project", project], timeout=90)
    plan["scoped_units"] = scoped_units(allow_empty_lxc=True)
    reviewed = scoped_effective_reviews(allow_lxc=True)
    if reviewed:
        if plan["sqlite"]["boxes"] or plan["sqlite"]["unresolved_operations"] or plan["incus_instances"]:
            raise UninstallError("SCOPED_LXC_EMPTY_RETIREMENT_REQUIRES_MCP_CLEANUP")
        plan["scoped_effective_reviews"] = reviewed
    plan["scoped_journal"] = scoped_tree(SCOPED_JOURNAL)
    plan["mode"] = SCOPED_MODE
    if plan["sqlite"]["boxes"] or plan["incus_instances"]:
        plan["blockers"].append("SCOPED_LIVE_BOXES: use the exact scoped normal drain coordinator")
    if not os.path.lexists(SCOPED_JOURNAL / "retirement-preimage.json"):
        plan["blockers"].append("SCOPED_RETIREMENT_PREIMAGE_MISSING: older installation has no verified original configuration")
    if not plan["blockers"]:
        try:
            plan["scoped_native"] = scoped_native(plan, "plan")
            if plan["scoped_native"]["phase"] != "INSTALLED" or plan["scoped_native"]["guard_present"] is not True:
                raise UninstallError("original scoped guard not verified")
        except UninstallError as exc:
            plan["blockers"].append(str(exc))
    else:
        plan["scoped_native"] = None
    for k in ("ready", "actions", "retained", "review_sha256"):
        plan.pop(k, None)
    plan["blockers"] = sorted(set(plan["blockers"]))
    plan["review_sha256"] = sha(canonical(plan))
    plan["ready"] = not plan["blockers"]
    plan["actions"] = ["fence controller and Root lifecycle, then stop only these services",
                       "archive original controller, Root journal and exact units",
                       "restore exact original Project/Profile; remove detached quarantine ACL",
                       "remove exact Incus guard drop-in and reload systemd without restarting Incus",
                       "stop owned guard; remove only verified scoped guard table",
                       "archive original Root journal and remove exact owned controller files"]
    plan["retained"] = ["Incus, original Project/Profile/bridge, all other native resources and workloads",
                        "private controller and Root audit archive, dedicated OS identity, external tunnel"]
    if len(canonical(plan)) > 768 * 1024:
        raise UninstallError("scoped retirement plan exceeds audit bound")
    return plan


def scoped_fence_value(record):
    plan = record["scoped_plan"]
    return {"schema": 1, "project": plan["project"], "review_sha256": plan["review_sha256"],
            "native_review_sha256": plan["scoped_native"]["review_sha256"], "archive": record.get("archive", "")}


def scoped_set_fence(record):
    value = scoped_fence_value(record)
    if os.path.lexists(SCOPED_FENCE):
        scoped_private_file(SCOPED_FENCE, 8192)
        prior = read_json(SCOPED_FENCE)
        if any(prior.get(k) != value[k] for k in ("schema", "project", "review_sha256", "native_review_sha256")) or prior.get("archive") not in ("", value["archive"]):
            raise UninstallError("scoped retirement fence belongs to another original")
    scoped_write_json(SCOPED_FENCE, value)


def scoped_audit(project):
    status = audit_status()
    record = read_json(AUDIT)
    if status["status"] not in ("STARTED", "COMPLETE") or not record or record.get("mode") != SCOPED_MODE:
        raise UninstallError("no matching original scoped retirement audit")
    plan = record.get("scoped_plan")
    if (not isinstance(plan, dict) or not plan.get("ready") or plan.get("mode") != SCOPED_MODE or
            plan.get("project") != project or plan.get("host_machine_id_sha256") != machine_sha() or
            record.get("review_sha256") != plan.get("review_sha256") or
            record.get("phase") not in SCOPED_PHASES):
        raise UninstallError("scoped original retirement identity mismatch")
    inputs = {k: v for k, v in plan.items() if k not in ("review_sha256", "ready", "actions", "retained")}
    if sha(canonical(inputs)) != plan["review_sha256"]:
        raise UninstallError("original scoped review digest changed")
    if os.path.lexists(SCOPED_FENCE):
        scoped_private_file(SCOPED_FENCE, 8192)
        fence = read_json(SCOPED_FENCE)
        expected = scoped_fence_value(record)
        if any(fence.get(k) != v for k, v in expected.items() if k != "archive") or fence.get("archive") not in ("", expected["archive"]):
            raise UninstallError("original scoped fence differs")
    elif record["phase"] != "APPROVED" and status["status"] != "COMPLETE":
        raise UninstallError("original scoped retirement fence missing")
    return record


def scoped_advance(record, phase, **updates):
    current = scoped_audit(record["scoped_plan"]["project"])
    if current != record or current["status"] != "STARTED" or SCOPED_PHASES.index(phase) < SCOPED_PHASES.index(record["phase"]):
        raise UninstallError("original scoped retirement audit changed")
    updated = {**record, **updates, "phase": phase}
    _replace_audit(updated)
    return updated


def scoped_copy(src, dst):
    src, dst = Path(src), Path(dst)
    original = file_fingerprint(src)
    if original is None or src.lstat().st_nlink != 1:
        raise UninstallError("scoped archive source missing or linked")
    fd = os.open(str(src), os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0))
    with os.fdopen(fd, "rb") as inp, Path(dst).open("xb") as out:
        os.fchmod(out.fileno(), 0o600)
        shutil.copyfileobj(inp, out)
        out.flush()
        os.fsync(out.fileno())
    if file_fingerprint(src) != original or scoped_private_file(Path(dst))["sha256"] != original["sha256"]:
        raise UninstallError("scoped archive copy changed")


def scoped_state_files(plan):
    validate_state_dir(STATE, plan["service_user_uid"])
    files = []
    for path in sorted(STATE.iterdir(), key=lambda x: x.name):
        if path.name == EXEC_ARCHIVE_NAME:
            continue
        if path.name not in STATE_FILES:
            raise UninstallError("unexpected scoped controller state entry")
        info = checked_file(path, "file")
        if info.st_nlink != 1 or stat.S_IMODE(info.st_mode) & 0o077:
            raise UninstallError("unsafe scoped state file")
        files.append({"name": path.name, "info": file_fingerprint(path)})
    if not {"ledger.sqlite", "install-manifest.json", SERVE_LOCK_NAME} <= {x["name"] for x in files}:
        raise UninstallError("incomplete original scoped controller state")
    return files


def scoped_verify_original(plan):
    if file_fingerprint(BINARY) != plan["binary"] or file_fingerprint(UNIT) != plan["unit"]:
        raise UninstallError("original scoped executable/unit drift")
    scoped_verify_effective_reviews(plan)
    if scoped_units(allow_empty_lxc=bool(plan.get("scoped_effective_reviews"))) != plan["scoped_units"] or scoped_tree(SCOPED_JOURNAL) != plan["scoped_journal"]:
        raise UninstallError("original scoped Root journal or unit drift")
    if inspect_sqlite(STATE / "ledger.sqlite", guided=True) != plan["sqlite"] or plan["sqlite"]["blockers"]:
        raise UninstallError("original scoped ledger changed; preserve protected work")
    if native_instances(plan["project"], guided=True):
        raise UninstallError("scoped native consumers remain")
    require_quiescent_scoped_native(plan["project"])
    if inspect_exec_archive(plan["service_user_uid"]) != plan["exec_archive"]:
        raise UninstallError("original scoped Exec archive changed")
    original = {x["name"]: x["info"] for x in plan["state_files"]}
    current = {x["name"]: x["info"] for x in scoped_state_files(plan)}
    for name in ("install-manifest.json", SERVE_LOCK_NAME):
        if current.get(name) != original.get(name):
            raise UninstallError("original scoped state identity drift")
    if scoped_native(plan, "plan") != plan["scoped_native"]:
        raise UninstallError("original scoped native proof changed")


def scoped_inactive(unit):
    p = scoped_properties(unit, "ActiveState", "MainPID")
    if p != {"ActiveState": "inactive", "MainPID": "0"}:
        raise UninstallError("owned service not proven inactive: " + unit)


def scoped_archive(record):
    plan = record["scoped_plan"]
    scoped_verify_original(plan)
    for unit in (UNIT.name, SCOPED_LIFECYCLE_UNIT.name):
        scoped_verify_effective_reviews(plan)
        command(["systemctl", "stop", unit], timeout=90)
        scoped_inactive(unit)
    if os.path.lexists(SCOPED_SOCKET):
        raise UninstallError("scoped lifecycle socket remains after stop")
    with hold_stopped_serve_lock(plan):
        scoped_verify_original(plan)
        cleanup = {"state_files": scoped_state_files(plan), "exec_archive": plan["exec_archive"]}
        actual = {**plan, **cleanup}
        archive = Path(archive_controller(actual, reviewed_plan=plan))
        for raw, proof in plan["scoped_units"].items():
            src = Path(raw)
            if file_fingerprint(src) != proof:
                raise UninstallError("scoped unit changed before archive")
            scoped_copy(src, archive / ("scoped-" + src.name))
        if plan.get("drain_receipt"):
            if scoped_private_file(SCOPED_DRAIN, 1024 * 1024) != plan["drain_receipt"]:
                raise UninstallError("original drain receipt changed before archive")
            scoped_copy(SCOPED_DRAIN, archive / "drain-receipt.json")
        root_copy = archive / "scoped-journal"
        shutil.copytree(SCOPED_JOURNAL, root_copy, symlinks=True, copy_function=scoped_copy)
        # copytree's copy_function preserves neither owner nor modes; the
        # private source directory mode is copied by copytree itself.
        if scoped_tree(root_copy) != plan["scoped_journal"] or scoped_tree(SCOPED_JOURNAL) != plan["scoped_journal"]:
            raise UninstallError("scoped Root journal archive not verified")
        for directory, _, _ in os.walk(root_copy, topdown=False):
            scoped_sync_dir(Path(directory))
        scoped_write_json(archive / "cleanup.json", cleanup)
        scoped_sync_dir(archive)
        scoped_sync_dir(RETIRED)
        record = scoped_advance(record, "ARCHIVED", archive=str(archive),
                                cleanup_sha256=sha(canonical(cleanup)))
        scoped_set_fence(record)
    return record


def scoped_archive_proof(record):
    plan = record["scoped_plan"]
    archive = Path(record.get("archive", ""))
    if archive.parent != RETIRED or not re.fullmatch(r"retired-[a-z0-9_]{8}", archive.name):
        raise UninstallError("exact original scoped retirement archive required")
    scoped_private_dir(RETIRED)
    scoped_private_dir(archive)
    scoped_private_file(archive / "review.json")
    scoped_private_file(archive / "cleanup.json")
    cleanup = read_json(archive / "cleanup.json")
    if read_json(archive / "review.json") != plan or sha(canonical(cleanup)) != record.get("cleanup_sha256"):
        raise UninstallError("original scoped archive review or cleanup receipt changed")
    if scoped_tree(archive / "scoped-journal") != plan["scoped_journal"]:
        raise UninstallError("private scoped Root archive changed")
    for name, proof in [("boxweave.bin", plan["binary"]), ("boxweave.service", plan["unit"])] + [
            ("scoped-" + Path(raw).name, value) for raw, value in plan["scoped_units"].items()] + [
            ("state-" + x["name"], x["info"]) for x in cleanup["state_files"]] + [
            ("exec-" + x["name"], x["info"]) for x in (cleanup.get("exec_archive") or {}).get("files", [])]:
        if scoped_private_file(archive / name)["sha256"] != proof["sha256"]:
            raise UninstallError("original scoped archive copy changed: " + name)
    if plan.get("drain_receipt"):
        if scoped_private_file(archive / "drain-receipt.json", 1024 * 1024)["sha256"] != plan["drain_receipt"]["sha256"]:
            raise UninstallError("archived drain receipt changed")
    return archive, cleanup


def scoped_incus_active():
    p = scoped_properties("incus.service", "ActiveState", "MainPID")
    if p["ActiveState"] != "active" or not p["MainPID"].isdigit() or int(p["MainPID"]) <= 1:
        raise UninstallError("Incus is not independently proven active")
    return p


def scoped_detach_guard(plan):
    scoped_verify_effective_reviews(plan)
    before = scoped_incus_active()
    proof = plan["scoped_units"][str(SCOPED_INCUS_DROPIN)]
    if os.path.lexists(SCOPED_INCUS_DROPIN):
        exact_unlink(SCOPED_INCUS_DROPIN, proof)
        scoped_sync_dir(SCOPED_INCUS_DROPIN.parent)
    command(["systemctl", "daemon-reload"], timeout=90)
    for unit in ("incus.service", "incus-startup.service"):
        p = scoped_properties(unit, "Requires", "Wants", "BindsTo", "Requisite")
        if any(SCOPED_GUARD_UNIT.name in v.split() for v in p.values()):
            raise UninstallError("effective Incus dependency still requires scoped guard")
    p = scoped_properties(SCOPED_GUARD_UNIT.name, "RequiredBy", "BoundBy", "RequisiteOf", "ConsistsOf", "PropagatesStopTo")
    if any(set(v.split()) - {SCOPED_LIFECYCLE_UNIT.name, UNIT.name} for v in p.values()):
        raise UninstallError("foreign service still depends on scoped guard; refuse stop")
    scoped_verify_effective_reviews(plan)
    command(["systemctl", "stop", SCOPED_GUARD_UNIT.name], timeout=90)
    scoped_inactive(SCOPED_GUARD_UNIT.name)
    if scoped_incus_active() != before:
        raise UninstallError("Incus runtime changed during guard detachment")


def scoped_postcheck(plan, archive):
    p = read_json(archive / "scoped-journal" / "retirement-preimage.json")
    project = plan["project"]
    bridge = p["Origin"]["Bridge"]
    if not VALID_NAME.fullmatch(bridge) or bridge != "incusbr-" + str(plan["service_user_uid"]):
        raise UninstallError("archived scoped bridge identity mismatch")
    for uri, expected in (("/1.0/projects/" + project, p["BeforeProject"]),
                          ("/1.0/profiles/default?project=" + project, p["BeforeProfile"]),
                          ("/1.0/networks/" + bridge + "?project=default", p["Bridge"])):
        raw, _ = command(["incus", "query", "local:" + uri])
        actual = json.loads(raw)
        if not isinstance(actual, dict) or any(actual.get(k) != v for k, v in expected.items()):
            raise UninstallError("restored native configuration not independently verified")
    raw, _ = command(["incus", "query", "local:/1.0/network-acls?project=default"])
    names = json.loads(raw)
    if not isinstance(names, list) or any(not isinstance(n, str) or "/bw-" in n for n in names):
        raise UninstallError("owned ACL absence not independently verified")
    raw, _ = command(["/usr/sbin/nft", "-j", "-a", "list", "tables"])
    tables = json.loads(raw).get("nftables")
    if not isinstance(tables, list) or not tables or any(
            item.get("table", {}).get("name", "").startswith("bw_pg_") for item in tables):
        raise UninstallError("scoped guard absence not independently verified")
    require_quiescent_scoped_native(project)
    if native_instances(project, guided=True) or list((Path("/sys/class/net") / bridge / "brif").iterdir()):
        raise UninstallError("native consumers appeared after retirement")
    scoped_incus_active()


def scoped_cleanup(record):
    plan = record["scoped_plan"]
    archive, cleanup = scoped_archive_proof(record)
    scoped_verify_effective_reviews(plan, allow_removed=True)
    for unit in (UNIT.name, SCOPED_LIFECYCLE_UNIT.name, SCOPED_GUARD_UNIT.name):
        scoped_inactive(unit)
    if os.path.lexists(SCOPED_SOCKET) or os.path.lexists(SCOPED_INCUS_DROPIN):
        raise UninstallError("scoped service socket or dependency remains")
    scoped_postcheck(plan, archive)
    before = scoped_incus_active()
    if os.path.lexists(UNIT):
        if file_fingerprint(UNIT) != plan["unit"]:
            raise UninstallError("controller unit drift")
        command(["systemctl", "disable", UNIT.name])
    for path in (UNIT, SCOPED_LIFECYCLE_UNIT, SCOPED_GUARD_UNIT):
        if os.path.lexists(path):
            exact_unlink(path, plan["unit"] if path == UNIT else plan["scoped_units"][str(path)])
            scoped_sync_dir(path.parent)
    command(["systemctl", "daemon-reload"], timeout=90)
    # A previously interrupted cleanup may have removed an exact file. No
    # new path or changed source may be adopted into the original retirement.
    if os.path.lexists(STATE):
        validate_state_dir(STATE, plan["service_user_uid"])
        expected = {x["name"]: x["info"] for x in cleanup["state_files"]}
        if set(x.name for x in STATE.iterdir()) - set(expected) - {EXEC_ARCHIVE_NAME}:
            raise UninstallError("unexpected scoped controller state after archive")
        if os.path.lexists(STATE / EXEC_ARCHIVE_NAME):
            current = inspect_exec_archive(plan["service_user_uid"])
            approved = cleanup.get("exec_archive")
            if not approved or (current["device"], current["inode"]) != (approved["device"], approved["inode"]):
                raise UninstallError("scoped Exec archive inode changed")
            byname = {x["name"]: x["info"] for x in approved["files"]}
            for item in current["files"]:
                if byname.get(item["name"]) != item["info"]:
                    raise UninstallError("unrecognized scoped Exec archive content")
            for item in current["files"]:
                exact_unlink(STATE / EXEC_ARCHIVE_NAME / item["name"], item["info"])
            (STATE / EXEC_ARCHIVE_NAME).rmdir()
        for name, proof in expected.items():
            if name != SERVE_LOCK_NAME and os.path.lexists(STATE / name):
                exact_unlink(STATE / name, proof)
        if os.path.lexists(STATE / SERVE_LOCK_NAME):
            exact_unlink(STATE / SERVE_LOCK_NAME, expected[SERVE_LOCK_NAME])
        STATE.rmdir()
        scoped_sync_dir(STATE.parent)
    original = archive / "scoped-original"
    if os.path.lexists(SCOPED_JOURNAL):
        if os.path.lexists(original) or scoped_tree(SCOPED_JOURNAL) != plan["scoped_journal"]:
            raise UninstallError("original scoped Root journal drift before archive move")
        os.rename(SCOPED_JOURNAL, original)
        scoped_sync_dir(SCOPED_JOURNAL.parent)
        scoped_sync_dir(archive)
    if scoped_tree(original) != plan["scoped_journal"]:
        raise UninstallError("moved original scoped journal not verified")
    if os.path.lexists(BINARY):
        exact_unlink(BINARY, plan["binary"])
        scoped_sync_dir(BINARY.parent)
    if scoped_incus_active() != before:
        raise UninstallError("Incus runtime changed during scoped file cleanup")
    scoped_postcheck(plan, archive)
    for path in (STATE, BINARY, UNIT, SCOPED_JOURNAL, SCOPED_GUARD_UNIT, SCOPED_LIFECYCLE_UNIT, SCOPED_INCUS_DROPIN, SCOPED_SOCKET):
        if os.path.lexists(path):
            raise UninstallError("scoped retirement postflight path remains")
    result = {"outcome": "COMPLETE", "mode": SCOPED_MODE, "scope": "full",
              "approved_review_sha256": plan["review_sha256"], "retirement_archive": str(archive),
              "host_machine_id_sha256": plan["host_machine_id_sha256"], "retained": plan["retained"]}
    audited_complete(plan["review_sha256"], result)
    scoped_finish_fence(scoped_audit(plan["project"]))
    return result


def scoped_finish_fence(record):
    if record["status"] != "COMPLETE":
        raise UninstallError("cannot clear scoped fence before verified COMPLETE")
    if os.path.lexists(SCOPED_FENCE):
        scoped_private_file(SCOPED_FENCE, 8192)
        if read_json(SCOPED_FENCE) != scoped_fence_value(record):
            raise UninstallError("completed scoped original fence mismatch")
        archive, _ = scoped_archive_proof(record)
        scoped_postcheck(record["scoped_plan"], archive)
        for path in (STATE, BINARY, UNIT, SCOPED_JOURNAL, SCOPED_GUARD_UNIT, SCOPED_LIFECYCLE_UNIT, SCOPED_INCUS_DROPIN, SCOPED_SOCKET):
            if os.path.lexists(path):
                raise UninstallError("completed scoped original has remaining path")
        if os.path.lexists(SCOPED_DRAIN):
            proof = record["scoped_plan"].get("drain_receipt")
            if not proof or scoped_private_file(SCOPED_DRAIN, 1024 * 1024) != proof:
                raise UninstallError("completed drain receipt mismatch")
            exact_unlink(SCOPED_DRAIN, proof)
            scoped_sync_dir(SCOPED_DRAIN.parent)
        SCOPED_FENCE.unlink()
        scoped_sync_dir(SCOPED_FENCE.parent)


def scoped_resume_plan(project, archive_hint=None):
    record = scoped_audit(project)
    plan = record["scoped_plan"]
    if archive_hint and archive_hint != record.get("archive"):
        raise UninstallError("not the exact original scoped archive")
    native = None
    if record["status"] == "COMPLETE":
        archive, _ = scoped_archive_proof(record)
        scoped_postcheck(plan, archive)
    elif record["phase"] == "APPROVED":
        scoped_verify_original(plan)
    else:
        archive, _ = scoped_archive_proof(record)
        scoped_verify_effective_reviews(plan, allow_removed=record["phase"]=="CLEANUP_STARTED")
        if record["phase"] == "CLEANUP_STARTED":
            scoped_postcheck(plan, archive)
        else:
            native = scoped_native(plan, "resume-plan")
    return {"ready": True, "mode": SCOPED_MODE, "status": record["status"], "phase": record["phase"],
            "project": project, "review_sha256": plan["review_sha256"], "archive": record.get("archive", ""),
            "audit_sha256": audit_status()["sha256"], "native_observation": native}


def scoped_continue(record):
    plan = record["scoped_plan"]
    if record["status"] == "COMPLETE":
        scoped_finish_fence(record)
        return record["result"]
    with scoped_lock(MAINTENANCE_LOCK):
        scoped_set_fence(record)
    if record["phase"] == "APPROVED":
        record = scoped_archive(record)
    archive, cleanup = scoped_archive_proof(record)
    scoped_verify_effective_reviews(plan, allow_removed=record["phase"]=="CLEANUP_STARTED")
    # Hold the original controller lock through native and file cleanup.
    # After a partial cleanup only the already removed final lock is absent.
    lockplan = {**plan, "state_files": cleanup["state_files"]}
    if record["phase"] == "CLEANUP_STARTED" and not os.path.lexists(STATE / SERVE_LOCK_NAME):
        if os.path.lexists(STATE) and list(STATE.iterdir()):
            raise UninstallError("serve lock disappeared before scoped state cleanup")
        lockplan["state_files"] = []
    with hold_stopped_serve_lock(lockplan):
        if record["phase"] != "CLEANUP_STARTED":
            if scoped_tree(SCOPED_JOURNAL) != plan["scoped_journal"] or inspect_sqlite(STATE / "ledger.sqlite", guided=True) != plan["sqlite"]:
                raise UninstallError("original scoped journal or ledger changed after archive")
            if file_fingerprint(BINARY) != plan["binary"] or file_fingerprint(UNIT) != plan["unit"]:
                raise UninstallError("original scoped installed identity changed after archive")
            for raw, proof in plan["scoped_units"].items():
                if raw == str(SCOPED_INCUS_DROPIN) and record["phase"] != "ARCHIVED" and not os.path.lexists(raw):
                    continue
                if file_fingerprint(Path(raw)) != proof:
                    raise UninstallError("original scoped unit changed after archive")
        for unit in (UNIT.name, SCOPED_LIFECYCLE_UNIT.name):
            scoped_inactive(unit)
        if record["phase"] == "ARCHIVED":
            result = scoped_native(plan, "restore")
            if result["phase"] != "NATIVE_RESTORED" or result["guard_present"] is not True:
                raise UninstallError("original native restore not verified")
            record = scoped_advance(record, "NATIVE_RESTORED")
        if record["phase"] == "NATIVE_RESTORED":
            scoped_detach_guard(plan)
            result = scoped_native(plan, "guard")
            if result["phase"] != "GUARD_ABSENT" or result["guard_present"] is not False:
                raise UninstallError("exact guard retirement not verified")
            record = scoped_advance(record, "GUARD_REMOVED")
        if record["phase"] == "GUARD_REMOVED":
            if scoped_native(plan, "resume-plan")["phase"] != "GUARD_ABSENT":
                raise UninstallError("guard retirement evidence changed")
            scoped_postcheck(plan, archive)
            record = scoped_advance(record, "CLEANUP_STARTED")
        return scoped_cleanup(record)


def scoped_main(args):
    if os.geteuid() != 0 or sys.platform != "linux":
        raise UninstallError("scoped retirement requires local Linux root")
    project = args.project or "user-" + str(pwd.getpwnam("boxweave").pw_uid)
    if not VALID_NAME.fullmatch(project) or project != "user-" + str(pwd.getpwnam("boxweave").pw_uid):
        raise UninstallError("exact dedicated scoped Project required")
    if (args.apply or args.box_uuid or args.native_prepare_sha256 or args.confirm_host_sha256 or
            args.confirm_review_sha256 or args.scope not in (None, "full") or args.guided_full_plan):
        raise UninstallError("scoped retirement is full-only with terminal approval, not legacy apply/scope")
    if os.path.lexists(SCOPED_DRAIN):
        if not (args.resume_plan or args.resume):
            raise UninstallError("SCOPED_DRAIN_IN_PROGRESS: review the original --resume-plan --scoped")
        prior = read_json(AUDIT) if os.path.lexists(AUDIT) else {}
        handed_off = prior.get("scoped_plan", {}).get("drain_receipt")
        if handed_off and scoped_private_file(SCOPED_DRAIN, 1024 * 1024) != handed_off:
            raise UninstallError("drain handoff proof mismatch")
        if not handed_off:
            if args.retirement_archive or os.path.lexists(SCOPED_FENCE):
                raise UninstallError("unrecognized drain handoff evidence")
            snapshot = scoped_drain_resume_preview(project)
            print(json.dumps(snapshot, indent=2, sort_keys=True), flush=True)
            if args.resume_plan:
                return 0 if snapshot["ready"] else 2
            result = scoped_drain_approve(project, snapshot, resume=True)
            print(json.dumps(result, indent=2, sort_keys=True), flush=True)
            return 0
    elif (args.plan or args.full) and not os.path.lexists(SCOPED_FENCE):
        if args.retirement_archive:
            raise UninstallError("retirement archive is only valid for original resume")
        scoped_incus_active()
        if native_instances(project, guided=False):
            snapshot = scoped_drain_plan(project)
            print(json.dumps(snapshot, indent=2, sort_keys=True), flush=True)
            if args.plan:
                return 0 if snapshot["ready"] else 2
            result = scoped_drain_approve(project, snapshot)
            print(json.dumps(result, indent=2, sort_keys=True), flush=True)
            return 0
    if args.resume_plan or args.resume:
        snapshot = scoped_resume_plan(project, args.retirement_archive)
        print(json.dumps(snapshot, indent=2, sort_keys=True), flush=True)
        if args.resume_plan:
            return 0
        prompt = "RESUME SCOPED " + project
    else:
        if not (args.plan or args.full) or args.retirement_archive:
            raise UninstallError("use --scoped --plan or --scoped --full")
        snapshot = scoped_plan(project)
        print(json.dumps(snapshot, indent=2, sort_keys=True), flush=True)
        if args.plan:
            return 0 if snapshot["ready"] else 2
        if not snapshot["ready"]:
            raise UninstallError("scoped retirement blocked; complete original MCP operations first")
        prompt = "REMOVE SCOPED " + project
    with open("/dev/tty", "r") as inp, open("/dev/tty", "w") as out:
        out.write("Type " + prompt + " to approve the exact scoped retirement: ")
        out.flush()
        if inp.readline().strip() != prompt:
            raise UninstallError("OPERATOR_APPROVAL_REQUIRED: scoped retirement not approved")
    with scoped_lock(SCOPED_LOCK):
        if args.resume:
            if scoped_resume_plan(project, args.retirement_archive) != snapshot:
                raise UninstallError("original scoped retirement evidence changed after approval")
            record = scoped_audit(project)
        else:
            with scoped_lock(MAINTENANCE_LOCK):
                if scoped_plan(project) != snapshot:
                    raise UninstallError("scoped retirement evidence changed after approval")
                audited_write({"status": "STARTED", "mode": SCOPED_MODE, "scope": "full",
                               "host_sha256": snapshot["host_machine_id_sha256"], "review_sha256": snapshot["review_sha256"],
                               "phase": "APPROVED", "scoped_plan": snapshot, "archive": ""})
                record = scoped_audit(project)
                scoped_set_fence(record)
        result = scoped_continue(record)
    print(json.dumps(result, indent=2, sort_keys=True), flush=True)
    return 0


# A normal full retirement deliberately retains private evidence, including
# SQLite and credentials. Erasure is a distinct local, reviewed transaction.
# Never delete the audit or journals of a STARTED uninstall to make a new
# installation appear clean.
def purge_absent(allow_completed_offline_fence=False):
    refuse_scoped_retirement()
    if os.path.lexists(OFFLINE_FENCE) and not allow_completed_offline_fence:
        raise UninstallError("OFFLINE_RETIREMENT_IN_PROGRESS: preserve exact original")
    for path in (STATE, UNIT, BINARY, NATIVE, SCRIPTLET, ROOT_UNIT, BACKUP,
                 RETIREMENT_UNIT, RETIREMENT_SOCKET, SCOPED_FENCE, SCOPED_JOURNAL,
                 SCOPED_GUARD_UNIT, SCOPED_LIFECYCLE_UNIT, SCOPED_INCUS_DROPIN, SCOPED_SOCKET,
                 Path("/run/boxweave-native-peer-root.sock"),
                 Path("/var/lib/boxweave-native-peer-guard-v1"),
                 Path("/var/lib/boxweave-native-peer-conntrack-v1"),
                 Path("/var/lib/boxweave-native-peer-conntrack-v1.json"),
                 Path("/var/lib/boxweave-native-peer-conntrack-v1.lock"),
                 Path("/etc/systemd/system/boxweave.service.d")):
        if os.path.lexists(path):
            raise UninstallError("PURGE_ACTIVE_OR_UNREVIEWED_RESIDUE: " + str(path))
    for unit in (UNIT.name, ROOT_UNIT.name, RETIREMENT_UNIT.name,
                 SCOPED_GUARD_UNIT.name, SCOPED_LIFECYCLE_UNIT.name):
        state = scoped_properties(unit, "LoadState", "ActiveState", "MainPID")
        if state != {"LoadState": "not-found", "ActiveState": "inactive", "MainPID": "0"}:
            raise UninstallError("purge requires absent effective service: " + unit)


def purge_native_empty(project):
    require_quiescent_native(project)
    if native_instances(project, guided=True):
        raise UninstallError("purge requires an empty original native Project")
    uid = str(pwd.getpwnam("boxweave").pw_uid)
    bridge = "incusbr-" + uid
    raw, _ = command(["incus", "query", "local:/1.0/projects/" + project])
    pr = json.loads(raw)
    if (not isinstance(pr, dict) or pr.get("name") != project or
            pr.get("config", {}).get("restricted") != "true" or
            pr.get("config", {}).get("restricted.networks.access") != bridge):
        raise UninstallError("original restricted Project identity unverified")
    raw, _ = command(["incus", "query", "local:/1.0/network-acls?project=default"])
    names = json.loads(raw)
    if not isinstance(names, list) or any(not isinstance(n, str) or "/bw-" in n for n in names):
        raise UninstallError("unreviewed native BoxWeave ACL remains")
    raw, _ = command(["/usr/sbin/nft", "-j", "-a", "list", "tables"])
    tables = json.loads(raw).get("nftables")
    if not isinstance(tables, list) or not tables or any(
            item.get("table", {}).get("name", "").startswith("bw_pg_") for item in tables):
        raise UninstallError("unreviewed BoxWeave guard remains")
    if list((Path("/sys/class/net") / bridge / "brif").iterdir()):
        raise UninstallError("original bridge has live ports")
    scoped_incus_active()
    posture = {"project": {k: pr.get(k) for k in ("name", "description", "config")}}
    for key, uri, fields in (
            ("profile", "/1.0/profiles/default?project=" + project, ("name", "config", "description", "devices")),
            ("bridge", "/1.0/networks/" + bridge + "?project=default", ("name", "config", "description", "managed", "type"))):
        raw, _ = command(["incus", "query", "local:" + uri])
        actual = json.loads(raw)
        if not isinstance(actual, dict) or not isinstance(actual.get("config"), dict):
            raise UninstallError("purge native postflight observation unverified")
        posture[key] = {k: actual.get(k) for k in fields}
    return sha(canonical(posture))


def purge_dir_info(path):
    scoped_private_dir(path)
    st = path.lstat()
    return {"device": st.st_dev, "inode": st.st_ino}


def purge_mount_boundary():
    points = maintenance_mounts()
    for root in (RETIRED, AUDIT, PURGE_FENCE, OFFLINE_FENCE):
        if maintenance_under_mount(root, points):
            raise UninstallError("PURGE_MOUNTED_ARTIFACT: " + str(root))
    if any(x.startswith(str(NATIVE) + ".closed.") for x in points):
        raise UninstallError("PURGE_MOUNTED_CLOSED_JOURNAL")


def legacy_archive_content(archive, plan, historical=False):
    """Bind exact archive bytes, names and identities before native effects.

    Live normal retirement may checkpoint SQLite on stop. Older receipts
    without a post-stop proof must match original hashes. Offline always
    requires the exact approved cold database.
    """
    st = archive.lstat()
    if not stat.S_ISDIR(st.st_mode) or st.st_uid != os.geteuid() or stat.S_IMODE(st.st_mode) != 0o700:
        raise UninstallError("untrusted legacy archive directory")
    if read_json(archive / "review.json") != plan:
        raise UninstallError("legacy archive review changed")
    expected = {"review.json": None, "boxweave.bin": plan["binary"]["sha256"],
                "boxweave.service": plan["unit"]["sha256"]}
    native = plan["native_prepare"]
    if native["status"] != "ABSENT":
        expected["native-prepare.json"] = native["sha256"]
    for f in plan["state_files"]:
        name = f["name"]
        transient = name in ("ledger.sqlite-wal", "ledger.sqlite-shm")
        if transient and not os.path.lexists(archive / ("state-" + name)) and not historical and plan.get("mode") != OFFLINE_MODE:
            continue
        exact = historical or plan.get("mode") == OFFLINE_MODE or name not in ("ledger.sqlite", "ledger.sqlite-wal", "ledger.sqlite-shm")
        expected["state-" + name] = f["info"]["sha256"] if exact else None
    for prefix, key in (("exec-", "exec_archive"), ("installer-backup-", "installer_backup")):
        for item in (plan.get(key) or {}).get("files", []):
            expected[prefix + item["name"]] = item["info"]["sha256"]
    if plan.get("retirement_verifier"):
        expected[RETIREMENT_UNIT.name] = plan["retirement_verifier"]["unit"]["sha256"]
    if set(expected) != {x.name for x in archive.iterdir()}:
        raise UninstallError("legacy archive exact entry set differs from review")
    actual = {}
    for name, digest in sorted(expected.items()):
        path = archive / name
        proof = file_fingerprint(path)
        st = path.lstat()
        if proof is None or st.st_uid != os.geteuid() or st.st_nlink != 1 or proof["mode"] != 0o600 or (digest is not None and proof["sha256"] != digest):
            raise UninstallError("legacy archive content differs from original: " + name)
        actual[name] = proof
    return sha(canonical(actual))


def offline_completed_archive(record, plan, archive):
    if (record.get("offline_plan") != plan or record.get("phase") != "CLEANUP_STARTED" or
            record.get("archive") != str(archive) or
            record.get("verified_deleted") != [b["uuid"] for b in plan["incus_instances"]] or
            record.get("unresolved_originals") != plan["offline_originals"] or
            record.get("result", {}).get("unresolved_originals") != plan["offline_originals"] or
            record.get("result", {}).get("operation_outcome_changed") is not False or
            record.get("result", {}).get("business_operations_replayed") is not False):
        raise UninstallError("offline retirement completion provenance unverified")
    return legacy_archive_content(archive, plan, historical=True)


def purge_inventory(project):
    purge_mount_boundary()
    purge_absent()
    native_postflight = purge_native_empty(project)
    info = scoped_private_file(AUDIT, 1024 * 1024)
    audit = read_json(AUDIT)
    records = audit.get("history", []) + [audit]
    if len(records) > 102 or audit.get("scope") != "full" or audit.get("status") != "COMPLETE":
        raise UninstallError("purge requires a verified COMPLETE full retirement")
    files = {str(AUDIT): info}
    dirs = {}
    archives = set()
    native_preimages = {}
    if os.path.lexists(RETIRED):
        dirs[str(RETIRED)] = purge_dir_info(RETIRED)
    for record in records:
        result = record.get("result", {})
        review = record.get("review_sha256")
        if (record.get("status") != "COMPLETE" or record.get("host_sha256") != machine_sha() or
                result.get("outcome") != "COMPLETE" or result.get("approved_review_sha256") != review or
                result.get("host_machine_id_sha256") != machine_sha() or
                not VALID_SHA.fullmatch(str(review or ""))):
            raise UninstallError("unverified historical retirement; preserve original evidence")
        if record.get("scope") != "full":
            raise UninstallError("historical retirement scope is not full")
        raw = result.get("retirement_archive")
        if not raw:
            if record.get("mode") in (OFFLINE_MODE, SCOPED_MODE):
                raise UninstallError("verified retirement archive missing")
            continue
        archive = Path(raw)
        if (archive.parent != RETIRED or not re.fullmatch(r"retired-[a-z0-9_]{8}", archive.name)
                or archive in archives):
            raise UninstallError("unrecognized historical retirement archive")
        archives.add(archive)
        dirs[str(archive)] = purge_dir_info(archive)
        review_path = archive / "review.json"
        files[str(review_path)] = scoped_private_file(review_path, 1024 * 1024)
        plan = read_json(review_path)
        inputs = {k: v for k, v in plan.items() if k not in ("review_sha256", "ready", "actions", "retained")}
        if (plan.get("project") != project or plan.get("scope") != "full" or
                plan.get("host_machine_id_sha256") != machine_sha() or plan.get("review_sha256") != review or
                sha(canonical(inputs)) != review or not plan.get("ready")):
            raise UninstallError("historical archive review identity mismatch")
        if record.get("mode") == SCOPED_MODE:
            if record.get("scoped_plan") != plan or record.get("archive") != str(archive):
                raise UninstallError("scoped archive/audit identity mismatch")
            _, cleanup = scoped_archive_proof(record)
            exact_names = {"review.json", "cleanup.json", "boxweave.bin", "boxweave.service",
                           "scoped-journal", "scoped-original"}
            exact_names.update("state-" + x["name"] for x in cleanup["state_files"])
            exact_names.update("exec-" + x["name"] for x in (cleanup.get("exec_archive") or {}).get("files", []))
            exact_names.update("scoped-" + Path(x).name for x in plan["scoped_units"])
            if plan.get("drain_receipt"):
                exact_names.add("drain-receipt.json")
            if {x.name for x in archive.iterdir()} != exact_names:
                raise UninstallError("scoped archive exact entry set differs from receipt")
        elif record.get("mode") == OFFLINE_MODE:
            observed = offline_completed_archive(record, plan, archive)
            if result.get("archive_content_sha256", observed) != observed:
                raise UninstallError("offline archive content receipt mismatch")
        elif record.get("mode") not in (None, ""):
            raise UninstallError("unknown retirement archive mode")
        else:
            content = result.get("archive_content_sha256")
            observed = legacy_archive_content(archive, plan, historical=content is None)
            if content is not None and content != observed:
                raise UninstallError("legacy archive content receipt mismatch")
        if record.get("mode") != SCOPED_MODE and plan.get("native_prepare", {}).get("status") != "ABSENT":
            source = read_json(archive / "native-prepare.json")
            if not source or source.get("status") not in ("APPLIED", "ROLLED_BACK"):
                raise UninstallError("original native preparation archive unverified")
            original = {**source, "status": "ROLLED_BACK"}
            digest = source.get("plan", {}).get("review_sha256")
            if digest != plan["native_prepare"]["review_sha256"] or not closed_native_review_valid(source["plan"]):
                raise UninstallError("archived native preparation digest mismatch")
            native_preimages.setdefault(digest, []).append(original)
        permitted = {"review.json", "boxweave.bin", "boxweave.service", "native-prepare.json",
                     RETIREMENT_UNIT.name}
        permitted.update("state-" + x for x in STATE_FILES)
        permitted.update("exec-" + x["name"] for x in (plan.get("exec_archive") or {}).get("files", []))
        permitted.update("installer-backup-" + x["name"] for x in (plan.get("installer_backup") or {}).get("files", []))
        if record.get("mode") == SCOPED_MODE:
            permitted.update(("cleanup.json", "scoped-journal", "scoped-original"))
            if plan.get("drain_receipt"):
                permitted.add("drain-receipt.json")
            permitted.update("scoped-" + Path(x).name for x in plan["scoped_units"])
        for path in sorted(archive.iterdir()):
            if path.name not in permitted:
                raise UninstallError("unrecognized retirement archive entry: " + str(path))
            if path.name in ("scoped-journal", "scoped-original"):
                if scoped_tree(path) != plan["scoped_journal"]:
                    raise UninstallError("scoped original archive changed")
                for directory, subdirs, names in os.walk(path, followlinks=False):
                    parent = Path(directory)
                    dirs[str(parent)] = purge_dir_info(parent)
                    for name in subdirs:
                        purge_dir_info(parent / name)
                    for name in names:
                        child = parent / name
                        files[str(child)] = scoped_private_file(child, 1024 * 1024)
            else:
                files[str(path)] = scoped_private_file(path)
        for name, proof in (("boxweave.bin", plan.get("binary")), ("boxweave.service", plan.get("unit"))):
            if proof and files.get(str(archive / name), {}).get("sha256") != proof["sha256"]:
                raise UninstallError("retirement archive binary/unit differs from review")
    latest = audit.get("result", {}).get("retirement_archive")
    if latest:
        path = Path(latest)
        if audit.get("mode") == SCOPED_MODE:
            scoped_postcheck(audit["scoped_plan"], path)
        elif os.path.lexists(path / "native-prepare.json"):
            legacy_retired_native_postcheck(project, read_json(path / "native-prepare.json"))
    if os.path.lexists(RETIRED) and set(RETIRED.iterdir()) != archives:
        raise UninstallError("unreferenced retirement archive; review original before purge")
    # NativePrepareArchive creates only closed, exact root journals under
    # this fixed basename. Bind their reviewed host/Project/digest before
    # including them. No open or foreign native journals are erased.
    matched_preimages = set()
    for path in sorted(NATIVE.parent.glob(NATIVE.name + ".closed.*")):
        if not re.fullmatch(re.escape(NATIVE.name) + r"\.closed\.[a-f0-9]{16}\.[0-9]{2}", path.name):
            raise UninstallError("unknown closed native journal filename")
        proof = scoped_private_file(path, 1024 * 1024)
        rec = read_json(path)
        plan = rec.get("plan", {})
        native_host = sha(("boxweave-native-host-v1|" + Path("/etc/machine-id").read_text().strip()).encode())
        if (rec.get("schema") != 1 or rec.get("status") != "ROLLED_BACK" or
                plan.get("target", {}).get("project") != project or plan.get("host_sha256") != native_host or
                not VALID_SHA.fullmatch(str(plan.get("review_sha256", ""))) or
                path.name.split(".")[-2] != plan["review_sha256"][:16] or
                not closed_native_review_valid(plan)):
            raise UninstallError("closed native journal provenance unverified")
        if rec not in native_preimages.get(plan["review_sha256"], []):
            raise UninstallError("closed native journal has no exact verified retirement archive")
        files[str(path)] = proof
        matched_preimages.add(plan["review_sha256"])
    if set(native_preimages) != matched_preimages:
        raise UninstallError("verified retirement lacks its original closed native journal")
    if len(files) > 100000 or len(dirs) > 30000:
        raise UninstallError("purge inventory exceeds supported bound")
    if purge_native_empty(project) != native_postflight:
        raise UninstallError("purge native postflight changed during inventory")
    return {"schema": 1, "project": project, "host_sha256": machine_sha(), "files": files, "directories": dirs,
            "native_postflight_sha256": native_postflight}


def legacy_retired_native_postcheck(project, record):
    target = record["plan"]["target"]
    if target["project"] != project:
        raise UninstallError("retired native Project identity changed")
    for uri, expected in (("/1.0/projects/" + project, record["before_project"]),
                          ("/1.0/profiles/default?project=" + project, record["before_profile"])):
        raw, _ = command(["incus", "query", "local:" + uri])
        current = json.loads(raw)
        if not isinstance(current, dict) or any(current.get(k) != v for k, v in expected.items()):
            raise UninstallError("retired Legacy native preimage changed")


def closed_native_review_valid(plan):
    # Go NativePreparePlan/Target field order is fixed, not sorted JSON.
    try:
        keys = ("project", "pool", "network", "acl")
        target = plan["target"]
        if set(target) != set(keys) or any(not isinstance(target[k], str) for k in keys):
            return False
        if set(plan) - {"target", "host_sha256", "native_state_sha256", "review_sha256", "actions", "warnings"}:
            return False
        v = {"target": {k: target[k] for k in keys}, "host_sha256": plan["host_sha256"],
             "native_state_sha256": plan["native_state_sha256"], "review_sha256": "",
             "actions": plan["actions"]}
        if plan.get("warnings"):
            v["warnings"] = plan["warnings"]
        if not isinstance(v["actions"], list) or any(not isinstance(x, str) for x in v["actions"] + v.get("warnings", [])):
            return False
        raw = json.dumps(v, ensure_ascii=False, separators=(",", ":"))
        raw = raw.replace("&", "\\u0026").replace("<", "\\u003c").replace(">", "\\u003e").replace("\u2028", "\\u2028").replace("\u2029", "\\u2029")
        return sha(b"bw-native-prepare-1\n" + raw.encode()) == plan["review_sha256"]
    except (KeyError, TypeError, ValueError):
        return False


def purge_validate_paths(plan):
    files, dirs = plan["files"], plan["directories"]
    if (plan.get("schema") != 1 or str(AUDIT) not in files or set(files) & set(dirs) or
            len(files) > 100000 or len(dirs) > 30000):
        raise UninstallError("invalid original purge inventory")
    for raw in set(files) | set(dirs):
        path = Path(raw)
        if not path.is_absolute() or str(path) != raw or ".." in path.parts:
            raise UninstallError("invalid purge inventory path")
        if path == AUDIT and raw in files or path == RETIRED and raw in dirs:
            continue
        if path.parent == NATIVE.parent and raw in files and re.fullmatch(re.escape(NATIVE.name) + r"\.closed\.[a-f0-9]{16}\.[0-9]{2}", path.name):
            continue
        if RETIRED not in path.parents:
            raise UninstallError("foreign path in purge receipt")
        relative = path.relative_to(RETIRED)
        if not re.fullmatch(r"retired-[a-z0-9_]{8}", relative.parts[0]) or str(path.parent) not in dirs:
            raise UninstallError("unverified purge receipt parent")


def purge_remaining(plan):
    purge_validate_paths(plan)
    purge_mount_boundary()
    purge_absent()
    observed = purge_native_empty(plan["project"])
    if "native_postflight_sha256" in plan and observed != plan["native_postflight_sha256"]:
        raise UninstallError("PURGE_NATIVE_POSTFLIGHT_CHANGED")
    files = plan["files"]
    dirs = plan["directories"]
    if os.path.lexists(AUDIT) and scoped_private_file(AUDIT, 1024 * 1024) != files.get(str(AUDIT)):
        raise UninstallError("original purge audit changed")
    for raw, proof in dirs.items():
        path = Path(raw)
        if not os.path.lexists(path):
            continue
        if purge_dir_info(path) != proof:
            raise UninstallError("reviewed purge directory identity changed")
        if any(str(child) not in files and str(child) not in dirs for child in path.iterdir()):
            raise UninstallError("unreviewed entry appeared in purge directory")
    for raw, proof in files.items():
        path = Path(raw)
        if os.path.lexists(path) and scoped_private_file(path) != proof:
            raise UninstallError("reviewed purge file identity changed")
    for path in NATIVE.parent.glob(NATIVE.name + ".closed.*"):
        if str(path) not in files:
            raise UninstallError("new closed native journal appeared during purge")
    if os.path.lexists(RETIRED) and str(RETIRED) not in dirs:
        raise UninstallError("new retirement directory appeared during purge")


def purge_review(project):
    if os.path.lexists(PURGE_FENCE):
        scoped_private_file(PURGE_FENCE, 1024 * 1024)
        receipt = read_json(PURGE_FENCE)
        plan = receipt["plan"]
        if (receipt.get("schema") not in (1, 2) or receipt.get("review_sha256") != sha(canonical(plan)) or
                plan.get("host_sha256") != machine_sha() or plan.get("project") != project):
            raise UninstallError("original purge receipt identity mismatch")
        purge_remaining(plan)
        if receipt["schema"] == 2:
            if not VALID_SHA.fullmatch(str(plan.get("native_postflight_sha256", ""))):
                raise UninstallError("original purge native witness missing")
            purge_prefix(plan)
        return receipt
    plan = purge_inventory(project)
    receipt = {"schema": 2, "plan": plan, "review_sha256": sha(canonical(plan))}
    if len(canonical(receipt)) + 1 > 1024 * 1024:
        raise UninstallError("purge receipt exceeds supported bound")
    return receipt


def purge_order(plan):
    files = [Path(x) for x in sorted(plan["files"]) if x != str(AUDIT)]
    dirs = [Path(x) for x in sorted(plan["directories"], key=lambda x: (-len(Path(x).parts), x))]
    return files + dirs + [AUDIT]


def purge_prefix(plan):
    missing = 0
    present = False
    for path in purge_order(plan):
        if os.path.lexists(path):
            present = True
        elif present:
            raise UninstallError("PURGE_NON_PREFIX_MISSING: preserve original receipt")
        else:
            missing += 1
    return missing


def purge_execute(receipt):
    plan = receipt["plan"]
    purge_remaining(plan)
    if receipt["schema"] == 2:
        purge_prefix(plan)
    for path in purge_order(plan):
        if not os.path.lexists(path):
            parent = path.parent
            while not os.path.lexists(parent):
                parent = parent.parent
            scoped_sync_dir(parent)
            continue
        purge_mount_boundary()
        if read_json(PURGE_FENCE) != receipt:
            raise UninstallError("original purge receipt changed")
        parent_proof = plan["directories"].get(str(path.parent))
        if parent_proof and purge_dir_info(path.parent) != parent_proof:
            raise UninstallError("reviewed purge parent changed")
        if str(path) in plan["directories"]:
            if purge_dir_info(path) != plan["directories"][str(path)]:
                raise UninstallError("reviewed purge directory changed")
            path.rmdir()
        else:
            if path == AUDIT:
                purge_remaining(plan)
                if any(os.path.lexists(x) for x in purge_order(plan)[:-1]):
                    raise UninstallError("retirement data remains before final audit erasure")
            exact_unlink(path, plan["files"][str(path)])
        scoped_sync_dir(path.parent)
    purge_remaining(plan)
    if any(os.path.lexists(x) for x in purge_order(plan)):
        raise UninstallError("reviewed purge path remains")
    if read_json(PURGE_FENCE) != receipt:
        raise UninstallError("original purge receipt changed")
    exact_unlink(PURGE_FENCE, scoped_private_file(PURGE_FENCE, 1024 * 1024))
    scoped_sync_dir(PURGE_FENCE.parent)


def purge_main(args):
    if os.geteuid() != 0 or sys.platform != "linux":
        raise UninstallError("purge requires original local Linux root")
    project = args.project or "user-" + str(pwd.getpwnam("boxweave").pw_uid)
    if project != "user-" + str(pwd.getpwnam("boxweave").pw_uid) or not VALID_NAME.fullmatch(project):
        raise UninstallError("exact dedicated purge Project required")
    if any((args.scoped, args.scope, args.retirement_archive, args.guided_full_plan, args.box_uuid,
            args.native_prepare_sha256, args.confirm_host_sha256, args.confirm_review_sha256)):
        raise UninstallError("purge is a separate completed-retirement action")
    receipt = purge_review(project)
    plan = receipt["plan"]
    print(json.dumps({"ready": True, "project": project, "review_sha256": receipt["review_sha256"],
                      "files": sorted(plan["files"]), "directories": sorted(plan["directories"]),
                      "retained": ["Incus, dedicated OS identity, original native Project/bridge, image cache, external tunnel"],
                      "irreversible": True, "retention": "erase",
                      "remaining_files": sum(os.path.lexists(x) for x in plan["files"]),
                      "remaining_directories": sum(os.path.lexists(x) for x in plan["directories"]),
                      "erasure_kind": "EXACT_FILE_UNLINK_NOT_STORAGE_SANITIZATION"}, indent=2, sort_keys=True), flush=True)
    if args.purge_plan:
        return 0
    prompt = "PURGE " + project
    with open("/dev/tty", "r") as inp, open("/dev/tty", "w") as out:
        out.write("Type " + prompt + " to permanently erase the reviewed retirement data: ")
        out.flush()
        if inp.readline().strip() != prompt:
            raise UninstallError("OPERATOR_APPROVAL_REQUIRED: purge not approved")
    with scoped_lock(SCOPED_LOCK), scoped_lock(MAINTENANCE_LOCK):
        if purge_review(project) != receipt:
            raise UninstallError("purge review changed after approval")
        if not os.path.lexists(PURGE_FENCE):
            scoped_write_json(PURGE_FENCE, receipt)
        purge_execute(receipt)
    print(json.dumps({"outcome": "PURGED", "project": project, "review_sha256": receipt["review_sha256"]}))
    return 0


# H1B preview only. This is deliberately not accepted by --apply, --full,
# --scoped, --purge or the existing approved retirement protocol.
MAINTENANCE_PLAN_SCHEMA = 1
MAINTENANCE_OBSERVER_CAP = 128 * 1024 * 1024


@contextmanager
def maintenance_observer_fd(path, expected):
    """Open a root-trusted executable by FD, never hash then exec a pathname."""
    path = Path(path)
    if not path.is_absolute() or ".." in path.parts or path == Path("/"):
        raise UninstallError("maintenance observer requires an absolute clean path")
    fd = os.open("/", os.O_RDONLY | os.O_DIRECTORY | os.O_CLOEXEC)
    try:
        for part in path.parts[1:-1]:
            parent = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW |
                             os.O_CLOEXEC, dir_fd=fd)
            os.close(fd)
            fd = parent
            st = os.fstat(fd)
            if st.st_uid != 0 or st.st_mode & 0o022:
                raise UninstallError("maintenance observer parent is not root-trusted")
        executable = os.open(path.name, os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC |
                             os.O_NONBLOCK, dir_fd=fd)
        try:
            before = os.fstat(executable)
            if (not stat.S_ISREG(before.st_mode) or before.st_uid != 0 or
                    before.st_nlink != 1 or before.st_mode & 0o6022 or
                    not before.st_mode & 0o111 or not 0 < before.st_size <= MAINTENANCE_OBSERVER_CAP):
                raise UninstallError("unsafe maintenance observer executable")
            digest = hashlib.sha256()
            total = 0
            while True:
                data = os.read(executable, 1024 * 1024)
                if not data:
                    break
                total += len(data)
                if total > MAINTENANCE_OBSERVER_CAP:
                    raise UninstallError("maintenance observer exceeds size bound")
                digest.update(data)
            actual = digest.hexdigest()
            if expected and actual != expected:
                raise UninstallError("maintenance observer checksum mismatch")
            if total != before.st_size or maintenance_stat(os.fstat(executable)) != maintenance_stat(before):
                raise UninstallError("maintenance observer changed during hash")
            yield executable, actual
            if maintenance_stat(os.fstat(executable)) != maintenance_stat(before):
                raise UninstallError("maintenance observer changed during observation")
        finally:
            os.close(executable)
    finally:
        os.close(fd)


def maintenance_stat(info):
    return [info.st_dev, info.st_ino, info.st_uid, info.st_gid, info.st_mode,
            info.st_nlink, info.st_size, info.st_mtime_ns, info.st_ctime_ns]


def maintenance_observe(path, expected, project):
    with maintenance_observer_fd(path, expected) as (fd, digest):
        env = {"PATH": "/usr/sbin:/usr/bin:/sbin:/bin",
               "HOME": "/root", "LC_ALL": "C", "LANG": "C"}
        result = subprocess.run(
            ["/proc/self/fd/" + str(fd), "doctor", "-maintenance", "-project", project],
            pass_fds=(fd,), capture_output=True, timeout=60, env=env, check=False)
        if result.returncode not in (0, 1) or len(result.stdout) > 16 * 1024 * 1024:
            raise UninstallError("maintenance observer unavailable")
        report = json.loads(result.stdout)
        if (not isinstance(report, dict) or report.get("version") != 1 or
                report.get("scope") != project or report.get("observer_sha256") != digest or
                report.get("cross_source_atomic") is not False or
                (result.returncode == 0) != (report.get("coverage") == "BOUNDED_SOURCES_OBSERVED")):
            raise UninstallError("maintenance observer contract or identity mismatch")
        return report


def maintenance_mounts():
    # mountinfo also detects bind mounts on the same device. Do not scan
    # through an independently mounted owned artifact (including descendants).
    raw = Path("/proc/self/mountinfo").read_bytes()
    if len(raw) > 4 * 1024 * 1024:
        raise UninstallError("mount inventory exceeds bound")
    points = []
    for line in raw.splitlines():
        fields = line.split()
        if len(fields) < 10 or b"-" not in fields:
            raise UninstallError("mount inventory unverified")
        point = re.sub(rb"\\([0-7]{3})", lambda m: bytes([int(m[1], 8)]), fields[4])
        points.append(os.fsdecode(point))
    return points


def maintenance_under_mount(path, points):
    root = str(path)
    return any(x == root or x.startswith(root + "/") for x in points)


def maintenance_state_inventory(uid):
    """Metadata only for mutable SQLite/WAL; H1A owns all logical DB reads."""
    try:
        fd = os.open(STATE, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC)
    except FileNotFoundError:
        return {"state": "ABSENT"}
    try:
        start = os.fstat(fd)
        if start.st_uid != uid or stat.S_IMODE(start.st_mode) != 0o700:
            return {"state": "UNSAFE"}
        files, unknown = [], 0
        with os.scandir(fd) as entries:
            for count, entry in enumerate(entries, 1):
                if count > 8192:
                    return {"state": "UNVERIFIED", "reason": "STATE_INVENTORY_BOUND"}
                st = entry.stat(follow_symlinks=False)
                name = entry.name
                if name not in STATE_FILES and name != EXEC_ARCHIVE_NAME:
                    unknown += 1
                    continue
                good = (st.st_uid == uid and
                        (stat.S_ISDIR(st.st_mode) and stat.S_IMODE(st.st_mode) == 0o700
                         if name == EXEC_ARCHIVE_NAME else
                         stat.S_ISREG(st.st_mode) and st.st_nlink == 1 and
                         stat.S_IMODE(st.st_mode) == 0o600))
                # For active DB components the exact logical content digest
                # is Go's responsibility; use identity and mode here.
                files.append({"name": name, "safe": good,
                              "identity": maintenance_stat(st)[:6]})
        end = os.fstat(fd)
        if maintenance_stat(start) != maintenance_stat(end) or not os.path.samestat(end, STATE.lstat()):
            return {"state": "UNVERIFIED", "reason": "STATE_DIRECTORY_CHANGED"}
        return {"state": "OBSERVED", "files": sorted(files, key=lambda x: x["name"]),
                "unknown_entries": unknown}
    finally:
        os.close(fd)


def maintenance_local_inventory(project, uid):
    """Reuse existing archive readers only on verified fixed paths.

    Historical retirement provenance remains an independent purge concern.
    Do not infer ownership from a directory name or read unknown archives.
    """
    out = {}
    try:
        points = maintenance_mounts()
    except (UninstallError, OSError, ValueError):
        return {"capture": {"state": "UNVERIFIED", "reason": "MOUNT_INVENTORY_UNAVAILABLE"}}
    roots = {"state": STATE, "scoped_journal": SCOPED_JOURNAL, "backup": BACKUP,
             "retired": RETIRED, "audit": AUDIT, "legacy_journal": NATIVE,
             "scriptlet": SCRIPTLET, "scoped_fence": SCOPED_FENCE,
             "purge_fence": PURGE_FENCE, "drain_fence": SCOPED_DRAIN, "offline_fence": OFFLINE_FENCE, "controller_binary": BINARY, "controller_unit": UNIT,
             "legacy_root_unit": ROOT_UNIT, "legacy_retirement_unit": RETIREMENT_UNIT,
             "scoped_guard_unit": SCOPED_GUARD_UNIT, "scoped_lifecycle_unit": SCOPED_LIFECYCLE_UNIT,
             "scoped_incus_dropin": SCOPED_INCUS_DROPIN,
             "legacy_root_operations": Path("/var/lib/boxweave-native-peer-conntrack-v1")}
    mounted = sorted(k for k, p in roots.items() if maintenance_under_mount(p, points))
    out["mounted_artifacts"] = mounted
    def observe(key, fn):
        if key in mounted:
            out[key] = {"state": "UNSAFE", "reason": "MOUNTED_ARTIFACT"}
            return
        try:
            out[key] = fn()
        except (UninstallError, OSError, ValueError, KeyError, TypeError):
            out[key] = {"state": "UNVERIFIED", "reason": "LOCAL_EVIDENCE_UNAVAILABLE"}
    observe("state", lambda: maintenance_state_inventory(uid))
    def scoped():
        if not os.path.lexists(SCOPED_JOURNAL):
            return {"state": "ABSENT"}
        return {"state": "OBSERVED", "tree": scoped_tree(SCOPED_JOURNAL)}
    observe("scoped_journal", scoped)
    def backup():
        v = inspect_installer_backup(project)
        return {"state": "OBSERVED", "proof": v} if v else {"state": "ABSENT"}
    observe("backup", backup)
    def audit():
        if not os.path.lexists(AUDIT):
            return {"state": "ABSENT"}
        proof = scoped_private_file(AUDIT, 1024 * 1024)
        v = audit_status()
        return {"state": v["status"], "sha256": proof["sha256"]}
    observe("audit", audit)
    # Existing completed archives are not read or adopted by a live preview.
    # The separate exact provenance reader and purge gate own those effects.
    def presence(path):
        try:
            st = path.lstat()
        except FileNotFoundError:
            return {"state": "ABSENT"}
        return {"state": "PRESENT_UNVERIFIED", "identity": maintenance_stat(st)}
    for key in ("retired", "scriptlet", "scoped_fence", "purge_fence", "drain_fence", "offline_fence", "legacy_root_operations"):
        observe(key, lambda key=key: presence(roots[key]))
    # Count even malformed historical names, without disclosing any names.
    try:
        names = []
        with os.scandir(NATIVE.parent) as entries:
            for count, entry in enumerate(entries, 1):
                if count > 32768:
                    raise UninstallError("historical journal inventory bound")
                if entry.name.startswith(NATIVE.name + ".closed."):
                    names.append(entry.name)
        out["closed_journals"] = {
            "state": "PRESENT_UNVERIFIED" if names else "ABSENT", "count": len(names),
            "names_sha256": sha(canonical(sorted(names)))}
    except (UninstallError, OSError):
        out["closed_journals"] = {"state": "UNVERIFIED"}
    return out


def maintenance_stable_report(report):
    # Only timestamps of observation itself are removed; operation/journal
    # identities and hashes remain bound. Never normalize arbitrary lists.
    result = json.loads(json.dumps(report))
    for key in ("database", "native"):
        result.get(key, {}).pop("epoch_ns", None)
    return result


def maintenance_preview(project, retention, host, report, local, planner_sha256=None):
    """Pure typed planner: aggregate blockers without mutating any source."""
    issues, effects = [], []
    def issue(code, severity, ref, next_action):
        issues.append({"code": code, "severity": severity, "resource": ref,
                       "next_action": next_action})
    def effect(kind, target, proof):
        effects.append({"kind": kind, "target": target, "coverage": proof})
    i = report.get("installation") or {}
    n = report.get("native") or {}
    l = report.get("ledger")
    mode = i.get("mode", "UNVERIFIED")
    runtime = (i.get("identity") or {}).get("runtime")
    offline = runtime != "VERIFIED"
    if not report:
        issue("OBSERVER_UNAVAILABLE", "HARD", "observer", "Run a root-trusted exact supported maintenance observer")
    if i.get("adapter") not in ("PINNED_RC13", "PINNED_RC14", "EXACT_RUNNING_OBSERVER_BINARY"):
        issue("INSTALLATION_IDENTITY_UNVERIFIED", "HARD", "installation", "Inspect exact manifest and binary; do not adopt")
    if (i.get("preimage") or {}).get("state") != "OBSERVED":
        issue("PREIMAGE_UNVERIFIED", "HARD", "native_preimage", "Preserve original preparation journal; no reconstructed baseline")
    if report.get("coverage") != "BOUNDED_SOURCES_OBSERVED":
        issue("SOURCE_COVERAGE_INCOMPLETE", "HARD", "observation", "Repeat read-only observation; missing is not empty")
    if (report.get("database") or {}).get("state") != "OBSERVED" or not isinstance(l, dict):
        issue("LEDGER_UNVERIFIED", "HARD", "ledger", "Inspect durable original ledger without starting controller")
        l = {}
    elif not VALID_SHA.fullmatch(report.get("ledger_observation_sha256", "")):
        issue("LEDGER_WITNESS_MISSING", "HARD", "ledger", "Use H1B observation adapter")
    if (n.get("evidence") or {}).get("state") != "OBSERVED":
        issue("NATIVE_INVENTORY_UNVERIFIED", "HARD", "native", "Inspect Incus locally; never start it for preview")
    if l.get("foreign_scope_rows", 0):
        issue("FOREIGN_LEDGER_SCOPE", "HARD", "ledger", "Preserve foreign rows; do not adopt or remove")
    if (n.get("unowned_or_mismatched", 0) or n.get("owned_unverified") or
            n.get("ownership_comparison") != "OBSERVED"):
        issue("NATIVE_IDENTITY_CONFLICT", "HARD", "native", "Resolve exact UUID/incarnation and original Create evidence")
    if l.get("orphan_bindings", 0) or l.get("orphan_fences", 0):
        issue("ORPHAN_NETWORK_EVIDENCE", "HARD", "ledger", "Inspect original binding and fence lineage")
    if n.get("active_operations", 0):
        offline = True
        issue("NATIVE_OPERATIONS_ACTIVE", "REVIEW", "native", "Drain and independently verify quiescence in a later reviewed phase")
    if n.get("other_projects", 0):
        issue("OTHER_PROJECTS_PRESERVED", "INFO", "native", "Preserve unrelated Projects; fresh-install prerequisites are separate")
    for key in ("project_config", "profile_config"):
        if n.get(key) not in ("BEFORE", "AFTER"):
            issue("NATIVE_CONFIG_UNVERIFIED", "HARD", key, "Compare recorded preimage; no overwrite of drift")
    if n.get("bridge_config") not in ("MANAGED_BRIDGE_OBSERVED", "RECORDED_PREIMAGE_MATCH"):
        issue("BRIDGE_CONFIG_UNVERIFIED", "HARD", "bridge", "Preserve bridge and verify exact native configuration")
    for key in ("bridge_consumers", "base_acl_consumers"):
        if n.get(key) not in ("OWNED_REFERENCES_ONLY", "NO_REFERENCES", "NOT_APPLICABLE"):
            issue("FOREIGN_OR_UNVERIFIED_CONSUMER", "HARD", key, "Inspect references; never alter foreign resources")
    if (n.get("base_acl_config") not in ("RECORDED_PREIMAGE_MATCH", "ABSENT") or
            n.get("base_acl_config") == "ABSENT" and n.get("profile_config") == "AFTER"):
        issue("BASE_ACL_DELTA_UNVERIFIED", "HARD", "base_acl", "Verify exact managed delta before restoration")
    if n.get("box_acl_references") != "OBSERVED":
        issue("BOX_ACL_UNVERIFIED", "HARD", "box_acl", "Verify per-Box ACL ownership and consumer evidence")
    if n.get("forward_references") != "OBSERVED":
        issue("FORWARD_DEVICE_UNVERIFIED", "HARD", "forward", "Verify exact proxy identity and endpoints")
    if mode == "SCOPED_V2":
        root = i.get("root_policy") or {}
        if root.get("kernel_guard") != "OBSERVED" or report.get("root_ledger_comparison") != "MATCH":
            issue("ROOT_POLICY_UNVERIFIED", "HARD", "scoped_guard", "Preserve guard and original journal; do not replay")
    if runtime != "VERIFIED":
        issue("CONTROLLER_RUNTIME_UNVERIFIED", "REVIEW", "controller", "Use offline retirement review, not a forced restart")
    for item in i.get("artifacts", []):
        if item.get("evidence", {}).get("state") not in ("PRESENT", "ABSENT"):
            issue("ARTIFACT_UNSAFE_OR_UNVERIFIED", "HARD", item["id"], "Inspect fixed path metadata and ownership")
    for unit in i.get("units", []):
        if (mode == "SCOPED_V2" and unit.get("drop_ins") == "KNOWN_LXC255_SCOPED_RETIREMENT_REVIEW_REQUIRED"
                and report.get("coverage") == "BOUNDED_SOURCES_OBSERVED"
                and (report.get("database") or {}).get("state") == "OBSERVED"):
            issue("LXC_SCOPED_RETIREMENT_REVIEW_REQUIRED", "REVIEW", unit["name"], "Use exact scoped uninstall plan; verify inherited policy and approved original scope; no runtime hardening approval")
        elif unit.get("drop_ins") != "ABSENT":
            issue("EFFECTIVE_DROPIN_UNREVIEWED", "HARD", unit["name"], "Review effective systemd configuration; do not remove inherited policy")
        if unit.get("effective_file", {}).get("state") not in ("OBSERVED", "ABSENT"):
            issue("EFFECTIVE_UNIT_UNVERIFIED", "HARD", unit["name"], "Verify exact installed fragment")
    present = {a["id"] for a in i.get("artifacts", []) if a.get("evidence", {}).get("state") == "PRESENT"}
    legacy = {"legacy_preparation", "legacy_root_unit", "legacy_retirement_unit",
              "legacy_root_socket", "legacy_retirement_socket", "legacy_root_operations",
              "legacy_root_lock", "legacy_root_journal"}
    scoped = {"scoped_root_journal", "scoped_guard_unit", "scoped_lifecycle_unit",
              "scoped_incus_dropin", "scoped_lifecycle_socket"}
    if ((present & legacy and present & scoped) or
            (mode == "SCOPED_V2" and present & legacy) or (mode == "LEGACY" and present & scoped)):
        issue("MIXED_INSTALLATION_ARTIFACTS", "HARD", "installation", "Retain both original witnesses; no cross-mode adoption")
    if mode == "SCOPED_V2" and (scoped - {"scoped_lifecycle_socket"}) - present:
        issue("SCOPED_ARTIFACTS_INCOMPLETE", "HARD", "installation", "Verify exact installed guard and lifecycle contract")
    if present & {"legacy_root_unit", "legacy_root_socket", "legacy_root_operations", "legacy_root_journal"}:
        offline = True
        issue("LEGACY_ROOT_ORIGINAL_REVIEW_REQUIRED", "HARD", "legacy_root", "Verify exact original stage, unit and journal; preserve UNKNOWN")
    if mode == "LEGACY":
        for original in l.get("legacy_pending_retirements", []):
            offline = True
            issue("LEGACY_RETIREMENT_QUEUE_PRESENT", "REVIEW", original, "Preserve exact original Delete evidence and verify retirement")
    for original in l.get("legacy_compactions", []):
        offline = True
        issue("LEGACY_COMPACTION_PENDING", "HARD", original, "Verify original Root compaction; preserve journal and fences")
    for row in l.get("nonterminal_or_unknown_originals", []):
        offline = True
        issue("ORIGINAL_" + ("UNKNOWN" if row["state"] == "UNKNOWN" else "IN_FLIGHT"),
              "HARD" if row["action"] == "box_create" else "REVIEW", row["id"],
              "Preserve original receipt and claims; review exact native outcome without replay")
    boxes = {(b["uuid"], b["incarnation"]): b for b in l.get("boxes", [])}
    matched = set(n.get("owned_matched") or [])
    for b in sorted(boxes.values(), key=lambda x: (x["uuid"], x["incarnation"])):
        exact = b["uuid"] in matched and b.get("create_proof") and b.get("binding_state") == "PINNED"
        if not exact:
            issue("BOX_ORIGIN_UNVERIFIED", "HARD", b["uuid"], "Inspect original Create, binding and native identity")
        effect("STOP_DELETE_BOX", b, "IDENTITY_OBSERVED" if exact else "UNVERIFIED")
    for p in l.get("peers", []):
        if p["state"] == "REVOKED":
            continue
        exact = ((p["source"], p["source_incarnation"]) in boxes and
                 (p["target"], p["target_incarnation"]) in boxes and
                 p["source"] in matched and p["target"] in matched and p["state"] == "CONFIRMED" and
                 bool(p.get("last_verified_original")))
        if not exact:
            offline = True
            issue("PEER_ORIGINAL_UNVERIFIED", "REVIEW", p["original"] or p["source"], "Review original directional policy; no replay")
        effect("RETIRE_PEER_DIRECTION", p, "IDENTITY_OBSERVED" if exact else "UNVERIFIED")
    for f in l.get("forwards", []):
        if f["state"] == "REVOKED":
            continue
        exact = ((f["box"], f["incarnation"]) in boxes and f["box"] in matched and
                 f["state"] == "CONFIRMED" and n.get("forward_references") == "OBSERVED")
        if not exact:
            offline = True
            issue("FORWARD_ORIGINAL_UNVERIFIED", "REVIEW", f["id"], "Review original proxy without redispatch")
        effect("RETIRE_FORWARD", f, "IDENTITY_OBSERVED" if exact else "UNVERIFIED")
    # H3B Doctor exposes exactly the H1 observation-side blockers. The
    # existing wrapper remains the coordinator for local inventory and
    # effects. A new observer must agree with this compatibility projection;
    # old pinned observers have no such field.
    readiness = report.get("maintenance_readiness")
    if readiness is not None:
        expected = sorted(issues, key=lambda x: ({"HARD": 0, "REVIEW": 1, "INFO": 2}[x["severity"]], x["code"], x["resource"]))
        if (not isinstance(readiness, dict) or readiness.get("observation_blockers") != expected or
                readiness.get("apply_available") is not False or readiness.get("effect_set_complete") is not False):
            issue("OBSERVER_BLOCKER_PROJECTION_MISMATCH", "HARD", "observation", "Inspect exact Doctor and lifecycle planner versions; preserve original evidence")
        else:
            issues = list(readiness["observation_blockers"])
    for key in local.get("mounted_artifacts", []):
        issue("MOUNTED_ARTIFACT", "HARD", key, "Do not traverse or erase independently mounted resources")
    for key, value in local.items():
        if not isinstance(value, dict):
            continue
        if value.get("state") in ("UNSAFE", "UNVERIFIED"):
            issue("LOCAL_ARTIFACT_UNVERIFIED", "HARD", key, "Inspect fixed local evidence without modifying it")
    state = local.get("state", {})
    if any(x["name"] == EXEC_ARCHIVE_NAME for x in state.get("files", [])):
        issue("EXEC_ARCHIVE_POST_FENCE_PROOF_REQUIRED", "REVIEW", "exec_archive", "Review exact output archive files before archival or erasure")
    if state.get("unknown_entries") or any(not x["safe"] for x in state.get("files", [])):
        issue("STATE_INVENTORY_UNSAFE", "HARD", "state", "Preserve unknown or unsafe entries")
    if local.get("audit", {}).get("state") == "STARTED" or any(
            local.get(k, {"state": "ABSENT"}).get("state") != "ABSENT" for k in ("scoped_fence", "purge_fence", "drain_fence", "offline_fence")):
        issue("INTERRUPTED_RETIREMENT", "HARD", "receipt", "Inspect original resume/purge plan; never start a second retirement")
    if local.get("scriptlet", {}).get("state") != "ABSENT":
        issue("SCRIPTLET_PROVENANCE_REQUIRED", "HARD", "scriptlet", "Preserve original global routing evidence")
    if any(local.get(k, {}).get("state") in ("PRESENT_UNVERIFIED", "COMPLETE")
           for k in ("retired", "closed_journals", "audit")):
        issue("HISTORICAL_ARCHIVE_PROVENANCE_REQUIRED", "HARD" if retention == "erase" else "INFO",
              "archives", "Preserve existing archives; separate completed-retirement purge review")
    if retention == "erase":
        issue("ERASURE_REQUIRES_POST_RETIREMENT_PROOF", "REVIEW", "archives", "Review exact completed archive set after retirement")
    for unit in i.get("units", []):
        if unit.get("effective_file", {}).get("state") == "OBSERVED":
            effect("RETIRE_UNIT", unit["name"], "FRAGMENT_OBSERVED_NOT_OWNERSHIP")
    for a in sorted(present):
        if a not in {"retirement_archive_root", "retirement_audit", "retirement_fence", "purge_fence"}:
            effect("REVIEW_OWNED_ARTIFACT", a, "METADATA_ONLY")
    effect("RESTORE_RECORDED_NATIVE_PREIMAGE", project,
           "OBSERVED" if (i.get("preimage") or {}).get("state") == "OBSERVED" else "UNVERIFIED")
    effect("ARCHIVE_ORIGINAL_EVIDENCE", {"retention": retention}, "POST_FENCE_PROOF_REQUIRED")
    if retention == "erase":
        effect("REVIEW_ERASE_COMPLETED_ARCHIVES", project, "POST_RETIREMENT_PROOF_REQUIRED")
    order = {"RETIRE_PEER_DIRECTION": 0, "RETIRE_FORWARD": 1, "STOP_DELETE_BOX": 2,
             "ARCHIVE_ORIGINAL_EVIDENCE": 3, "RETIRE_UNIT": 4,
             "RESTORE_RECORDED_NATIVE_PREIMAGE": 5, "REVIEW_OWNED_ARTIFACT": 6,
             "REVIEW_ERASE_COMPLETED_ARCHIVES": 7}
    # Presentation order only; H2 must derive a safe executable sequence.
    effects.sort(key=lambda x: (order[x["kind"]], canonical(x)))
    issues.sort(key=lambda x: ({"HARD": 0, "REVIEW": 1, "INFO": 2}[x["severity"]], x["code"], x["resource"]))
    blocked = any(x["severity"] == "HARD" for x in issues)
    result = {"schema": MAINTENANCE_PLAN_SCHEMA, "scope": "full", "project": project,
              "host_machine_id_sha256": host, "retention": retention,
              "planner_sha256": planner_sha256,
              "route": "OFFLINE_RETIREMENT_REVIEW" if offline else "NORMAL_CLEANUP_REVIEW",
              "status": "BLOCKED" if blocked else "REVIEW_ONLY", "ready": False,
              "classification": "HARD_REFUSAL" if blocked else ("OFFLINE_REVIEW" if offline else "NORMAL_REVIEW"),
              "apply_available": False, "effect_set_complete": False,
              "cross_source_atomic": False, "blockers": issues, "proposed_effects": effects,
              "effects_are_unordered_preview": True,
              "preserve": ["Incus daemon", "service account", "storage pool", "bridge", "unrelated Projects", "unrelated Host services"],
              "observation": maintenance_stable_report(report), "local_inventory": local,
              "next_action": "Resolve hard blockers; H2 must review exact effects under maintenance fence"
              if blocked else "Review proposed scope; H2 must verify complete effects under maintenance fence"}
    result["preview_sha256"] = sha(b"boxweave-maintenance-preview-v1\n" + canonical(result))
    return result


def maintenance_main(args):
    if os.geteuid() != 0 or sys.platform != "linux":
        raise UninstallError("maintenance preview requires local root")
    uid = pwd.getpwnam("boxweave").pw_uid
    project = args.project or "user-" + str(uid)
    if project != "user-" + str(uid) or uid == 0:
        raise UninstallError("exact dedicated service Project required")
    host = machine_sha()
    path = args.observer_binary or str(BINARY)
    before = maintenance_local_inventory(project, uid)
    try:
        if before.get("mounted_artifacts") or before.get("capture"):
            raise UninstallError("unsafe or unverified mount inventory")
        report = maintenance_observe(path, args.observer_sha256, project)
    except UninstallError as exc:
        report = {}
        before["observer_failure"] = {"state": "UNVERIFIED", "reason": str(exc)}
    except (OSError, ValueError, TypeError, subprocess.TimeoutExpired):
        report = {}
        before["observer_failure"] = {"state": "UNVERIFIED", "reason": "OBSERVER_EXEC_OR_JSON_UNAVAILABLE"}
    failure = before.pop("observer_failure", None)
    after = maintenance_local_inventory(project, uid)
    if before != after or host != machine_sha():
        # Discard cross-source comparison; no favorable preview on drift.
        report = {}
        after["capture"] = {"state": "UNVERIFIED", "reason": "LOCAL_EVIDENCE_CHANGED"}
    if failure:
        after["observer_failure"] = failure
    plan = maintenance_preview(project, args.retention, host, report, after,
                               sha(Path(__file__).read_bytes()))
    if args.maintenance_recheck:
        match = (host == args.confirm_host_sha256 and
                 plan["preview_sha256"] == args.confirm_preview_sha256)
        plan["recheck"] = ("MATCH_BLOCKED" if plan["status"] == "BLOCKED" else "MATCH_REVIEW_ONLY") if match else "STALE_PREVIEW"
        print(json.dumps(plan, sort_keys=True, indent=2))
        return 0 if match and plan["status"] == "REVIEW_ONLY" else 2
    print(json.dumps(plan, sort_keys=True, indent=2))
    return 0 if plan["status"] == "REVIEW_ONLY" else 2


# Normal Scoped V2 live retirement. The preview is not an effect permit:
# this separate exact plan is approved locally, durably fenced, then each
# original is admitted using the existing restricted business handlers.
DRAIN_MODE = "SCOPED_V2_NORMAL_DRAIN"


def scoped_drain_scope(report):
    l = report["ledger"]
    n = report["native"]
    return {"boxes": l["boxes"], "retired": l["retired_boxes"],
            "peers": [p for p in l["peers"] if p["state"] != "REVOKED"],
            "forwards": [f for f in l["forwards"] if f["state"] != "REVOKED"],
            "power": n.get("owned_power") or {}}


def scoped_drain_steps(scope):
    steps = []
    for p in scope["peers"]:
        if (p["state"] != "CONFIRMED" or p["source"] != p["source_incarnation"] or
                p["target"] != p["target_incarnation"] or not p.get("last_verified_original")):
            raise UninstallError("unverified original Peer direction")
        steps.append({"action": "box_peer_set", "box": p["source"], "target": p["target"]})
    for f in scope["forwards"]:
        if f["state"] != "CONFIRMED" or f["box"] != f["incarnation"]:
            raise UninstallError("unverified original Forward")
        steps.append({"action": "port_forward_delete", "box": f["box"], "forward": f["id"]})
    for b in scope["boxes"]:
        if b["uuid"] != b["incarnation"] or not b["create_proof"] or b["binding_state"] != "PINNED":
            raise UninstallError("unverified original Box")
        steps.append({"action": "box_stop", "box": b["uuid"]})
        steps.append({"action": "box_delete", "box": b["uuid"]})
    if len(steps) > 16384:
        raise UninstallError("approved drain effects exceed supported bound")
    return steps


def scoped_drain_global_inventory(project, scope):
    # The current Root lifecycle contract requires exclusive host-wide
    # instance inventory and exact default-Project ACL names. Refuse before
    # admitting a business original, not after creating UNKNOWN.
    raw, _ = command(["incus", "query", "local:/1.0/network-acls?project=default&recursion=1"], timeout=25)
    acls = json.loads(raw)
    expected = {"bw-peer-quarantine"}
    expected.update(b["acl"] for b in scope["boxes"])
    if not isinstance(acls, list) or {a.get("name") for a in acls} != expected or len(acls) != len(expected):
        raise UninstallError("DRAIN_ROOT_ACL_INVENTORY_NOT_EXCLUSIVE")
    raw, _ = command(["incus", "list", "local:", "--all-projects", "--format", "json"], timeout=25)
    actual = json.loads(raw)
    want = {(project, b["name"], b["uuid"]) for b in scope["boxes"]}
    if not isinstance(actual, list) or len(actual) != len(want) or {
            (b.get("project"), b.get("name"), (b.get("config") or {}).get("volatile.uuid"))
            for b in actual} != want:
        raise UninstallError("DRAIN_ROOT_INSTANCE_INVENTORY_NOT_EXCLUSIVE")


def scoped_drain_capture(project, active, observer=None, observer_sha=None, allow_drain=False):
    """Bounded root read only, using the shared H1 observation and existing
    exact scoped unit/archive readers. No approval or effect here."""
    uid = pwd.getpwnam("boxweave").pw_uid
    host = machine_sha()
    before = maintenance_local_inventory(project, uid)
    report = {}
    blockers = []
    try:
        if before.get("mounted_artifacts") or before.get("capture"):
            raise UninstallError("unsafe or unverified mount inventory")
        report = maintenance_observe(observer or BINARY, observer_sha, project)
    except (UninstallError, OSError, ValueError, TypeError, subprocess.TimeoutExpired) as exc:
        blockers.append("DRAIN_OBSERVER_UNAVAILABLE:" + type(exc).__name__)
    local = maintenance_local_inventory(project, uid)
    if before != local or host != machine_sha():
        report = {}
        blockers.append("DRAIN_LOCAL_EVIDENCE_CHANGED")
    # Only the current root-private drain is expected during its resume.
    # A generic preview must still refuse this in-progress transaction.
    classified = dict(local)
    if allow_drain:
        classified["drain_fence"] = {"state": "ABSENT"}
    preview = maintenance_preview(project, "retain", host, report, classified)
    for issue in preview["blockers"]:
        if issue["severity"] == "INFO":
            continue
        if not active and issue["code"] == "CONTROLLER_RUNTIME_UNVERIFIED":
            continue
        if issue["code"] in ("EXEC_ARCHIVE_POST_FENCE_PROOF_REQUIRED", "LXC_SCOPED_RETIREMENT_REVIEW_REQUIRED"):
            continue  # independently verify below
        blockers.append(issue["code"] + ":" + issue["resource"])
    i = report.get("installation") or {}
    n = report.get("native") or {}
    identity = i.get("identity") or {}
    if (i.get("adapter") != "EXACT_RUNNING_OBSERVER_BINARY" or i.get("mode") != "SCOPED_V2" or
            identity.get("static_contract") != "VERIFIED"):
        blockers.append("DRAIN_EXACT_CURRENT_SCOPED_CANDIDATE_REQUIRED")
    if (identity.get("service_unit") != ("ACTIVE" if active else "INACTIVE") or
            active and identity.get("runtime") != "VERIFIED" or
            identity.get("guard_unit") != "ACTIVE" or identity.get("lifecycle_unit") != "ACTIVE"):
        blockers.append("DRAIN_EXACT_SERVICE_STATE_UNVERIFIED")
    if n.get("base_acl_config") != "RECORDED_PREIMAGE_MATCH" or n.get("bridge_config") != "RECORDED_PREIMAGE_MATCH":
        blockers.append("DRAIN_EXACT_NATIVE_PREIMAGE_UNVERIFIED")
    if any(local.get(k, {}).get("state") != "ABSENT" for k in
           ("backup", "scriptlet", "legacy_root_operations", "scoped_fence", "purge_fence")):
        blockers.append("DRAIN_MIXED_OR_INTERRUPTED_INSTALLATION")
    if local.get("audit", {}).get("state") not in ("ABSENT", "COMPLETE"):
        blockers.append("DRAIN_ORIGINAL_AUDIT_UNSETTLED")
    # Do not execute the historical binary or traverse unsafe state just
    # to get a more favorable preview.
    scope, immutable, witness = None, None, None
    if report.get("coverage") == "BOUNDED_SOURCES_OBSERVED":
        try:
            scope = scoped_drain_scope(report)
            if (set(scope["power"]) != {b["uuid"] for b in scope["boxes"]} or
                    any(s not in ("Running", "Stopped") for s in scope["power"].values())):
                blockers.append("DRAIN_NATIVE_POWER_UNVERIFIED")
            if scope["boxes"]:
                try:
                    scoped_drain_global_inventory(project, scope)
                except UninstallError as exc:
                    blockers.append(str(exc))
            if not isinstance(local.get("state"), dict) or local["state"].get("state") != "OBSERVED":
                raise UninstallError("state metadata unavailable")
            state_ids = {f["name"]: f["identity"] for f in local["state"]["files"]
                         if f["name"] in ("ledger.sqlite", "install-manifest.json", SERVE_LOCK_NAME) and f["safe"]}
            if len(state_ids) != 3:
                blockers.append("DRAIN_REQUIRED_STATE_IDENTITY_UNVERIFIED")
            # Existing uninstaller ledger reader adds the Exec GC blocker.
            # Confirmed Peer/Forward/address dependencies are intentionally
            # handled by the exact normal drain, not ignored by retirement.
            db = STATE / "ledger.sqlite"
            cold = not active and not any(os.path.lexists(str(db) + x) for x in ("-wal", "-shm", "-journal"))
            sql = inspect_sqlite(db, guided=True, cold=True) if cold else inspect_sqlite(db, guided=True)
            allowed = ("peer_grants_NOT_RELEASED:", "forward_claims_NOT_RELEASED:",
                       "network_address_fences_NOT_RELEASED:")
            blockers.extend("DRAIN_" + b for b in sql["blockers"] if not b.startswith(allowed))
            if sql["version"] != 6 or sql["unresolved_operations"]:
                blockers.append("DRAIN_UNSETTLED_ORIGINALS")
            exec_archive = inspect_exec_archive(uid)
            # Exact static installed contract and original scope are still
            # required. Hint alone grants no authority.
            reviewed = scoped_effective_reviews(allow_lxc=True)
            if reviewed and (set(reviewed) != {UNIT.name, SCOPED_GUARD_UNIT.name, SCOPED_LIFECYCLE_UNIT.name} or
                    any(u.get("drop_ins") != "KNOWN_LXC255_SCOPED_RETIREMENT_REVIEW_REQUIRED"
                        for u in i.get("units", []) if u.get("name") in reviewed)):
                raise UninstallError("DRAIN_INHERITED_POLICY_OBSERVER_MISMATCH")
            units = scoped_units(allow_empty_lxc=bool(reviewed))
            incus_pid = scoped_incus_active()
            if i.get("adapter") == "EXACT_RUNNING_OBSERVER_BINARY":
                command([str(BINARY), "install-status", "-project", project], timeout=90)
            immutable = {"host": host, "project": project, "installed": i.get("installed_sha256"),
                         "manifest": i.get("manifest_sha256"), "unit": i.get("unit_sha256"),
                         "preimage": i.get("preimage", {}).get("sha256"), "units": units,
                         "state_ids": state_ids, "exec_archive": exec_archive,
                         "incus_pid": incus_pid, "audit": local.get("audit")}
            if reviewed:
                immutable["scoped_effective_reviews"] = reviewed
            witness = {"ledger": report.get("ledger_observation_sha256"),
                       "native": n.get("evidence", {}).get("sha256"),
                       "journal": local.get("scoped_journal", {}).get("tree"),
                       "operations": sql["operations"]}
            if (not all(VALID_SHA.fullmatch(x or "") for x in
                        (witness["ledger"], witness["native"], immutable["preimage"],
                         immutable["installed"], immutable["manifest"], immutable["unit"])) or
                    not witness["journal"]):
                blockers.append("DRAIN_WITNESS_INCOMPLETE")
        except (UninstallError, OSError, ValueError, KeyError, TypeError) as exc:
            blockers.append("DRAIN_STATIC_OR_LEDGER_PROOF_UNVERIFIED:" + type(exc).__name__)
    return {"scope": scope, "immutable": immutable, "witness": witness,
            "blockers": sorted(set(blockers)), "observation": preview}


def scoped_drain_plan(project, observer=None, observer_sha=None):
    if os.path.lexists(SCOPED_DRAIN) or os.path.lexists(SCOPED_FENCE) or os.path.lexists(PURGE_FENCE):
        raise UninstallError("SCOPED_DRAIN_IN_PROGRESS: inspect the original resume")
    capture = scoped_drain_capture(project, True, observer, observer_sha)
    steps = []
    try:
        if capture["scope"] is not None:
            steps = scoped_drain_steps(capture["scope"])
    except UninstallError:
        capture["blockers"].append("DRAIN_EXACT_EFFECT_SET_UNVERIFIED")
    if not steps:
        capture["blockers"].append("DRAIN_NO_VERIFIED_LIVE_BOX_SET")
    plan = {"schema": 1, "mode": DRAIN_MODE, "project": project, "scope": "full",
            "host_machine_id_sha256": machine_sha(), "retention": "retain",
            "capture": capture, "steps": steps,
            "actions": ["fence public controller admission; stop only exact Controller",
                        "revoke exact confirmed Peer directions, remove exact Forward proxies",
                        "stop and delete exact owned Boxes via durable business Operations",
                        "archive original receipts; restore native preimage and retire owned services",
                        "keep Incus, shared bridge/pool, dedicated account and unrelated resources"]}
    plan["ready"] = not capture["blockers"] and capture["immutable"] is not None and capture["witness"] is not None
    plan["blockers"] = capture["blockers"]
    plan["review_sha256"] = sha(b"boxweave-scoped-normal-drain-v1\n" + canonical(plan))
    if len(canonical(plan)) > 512 * 1024:
        raise UninstallError("approved normal drain plan exceeds durable receipt bound")
    return plan


def scoped_drain_read(project):
    scoped_private_file(SCOPED_DRAIN, 1024 * 1024)
    r = read_json(SCOPED_DRAIN)
    p = r["approved_plan"]
    review = p["review_sha256"]
    if (r["schema"] != 1 or p["schema"] != 1 or p["mode"] != DRAIN_MODE or not p["ready"] or
            r["project"] != project or p["project"] != project or
            p["host_machine_id_sha256"] != machine_sha() or r["review_sha256"] != review or
            sha(b"boxweave-scoped-normal-drain-v1\n" + canonical({k: v for k, v in p.items() if k != "review_sha256"})) != review or
            r["binary_sha256"] != p["capture"]["immutable"]["installed"] or
            not VALID_SHA.fullmatch(r["expected_ledger_sha256"]) or not VALID_SHA.fullmatch(r["expected_native_sha256"]) or
            not re.fullmatch(r"[a-f0-9]{32}", r["actor"]["id"]) or
            r["phase"] not in ("APPROVED", "DRAINING", "DRAINED") or
            r["index"] != len(r["completed"]) or not 0 <= r["index"] <= len(r["steps"]) or
            r["steps"] != [{**step, "key": "scoped-drain-" + r["actor"]["id"] + "-" + str(n)}
                           for n, step in enumerate(p["steps"])] or
            r["phase"] == "APPROVED" and r["index"] != 0 or
            r["phase"] == "DRAINED" and r["index"] != len(r["steps"])):
        raise UninstallError("exact original normal drain receipt unverified")
    if (len({x["id"] for x in r["completed"]}) != r["index"] or
            any(not re.fullmatch(r"[a-f0-9]{32}", x["id"]) for x in r["completed"])):
        raise UninstallError("normal drain original receipts unverified")
    return r


def scoped_drain_checkpoint(record):
    scoped_write_json(SCOPED_DRAIN, record)


def scoped_drain_begin(plan):
    import secrets
    import time
    if not plan["ready"] or os.path.lexists(SCOPED_DRAIN):
        raise UninstallError("normal drain not approved or original exists")
    actor = {"id": secrets.token_hex(16), "expires": int(time.time()) + 30 * 86400}
    witness = plan["capture"]["witness"]
    r = {"schema": 1, "project": plan["project"], "binary_sha256": plan["capture"]["immutable"]["installed"],
         "review_sha256": plan["review_sha256"], "approved_plan": plan, "phase": "APPROVED",
         "actor": actor, "index": 0, "completed": [],
         "steps": [{**step, "key": "scoped-drain-" + actor["id"] + "-" + str(n)}
                   for n, step in enumerate(plan["steps"])],
         "expected_ledger_sha256": witness["ledger"], "expected_native_sha256": witness["native"],
         "last_witness": witness}
    if len(canonical(r)) > 768 * 1024:
        raise UninstallError("normal drain receipt exceeds durable bound")
    scoped_drain_checkpoint(r)
    return r


def scoped_drain_expected(record, extra=None):
    scope = json.loads(json.dumps(record["approved_plan"]["capture"]["scope"]))
    receipts = record["completed"] + ([extra] if extra is not None else [])
    for step, receipt in zip(record["steps"], receipts):
        action, box = step["action"], step["box"]
        if action == "box_peer_set":
            scope["peers"] = [p for p in scope["peers"] if not (p["source"] == box and p["target"] == step["target"])]
        elif action == "port_forward_delete":
            scope["forwards"] = [f for f in scope["forwards"] if f["id"] != step["forward"]]
        elif action == "box_stop":
            scope["power"][box] = "Stopped"
        elif action == "box_delete":
            b = next(b for b in scope["boxes"] if b["uuid"] == box)
            scope["boxes"] = [x for x in scope["boxes"] if x["uuid"] != box]
            scope["power"].pop(box)
            scope["retired"].append({"uuid": box, "incarnation": b["incarnation"],
                                     "name": b["name"], "delete_original": receipt["id"]})
        else:
            raise UninstallError("unknown approved drain effect")
    scope["retired"].sort(key=lambda b: b["uuid"])
    return scope


def scoped_drain_verify(record, active=False, extra=None, exact=True):
    fresh = scoped_drain_capture(record["project"], active, allow_drain=True)
    if fresh["blockers"] or fresh["immutable"] != record["approved_plan"]["capture"]["immutable"]:
        raise UninstallError("SCOPED_DRAIN_REVIEW_DRIFT:" + ",".join(fresh["blockers"]))
    if fresh["scope"] != scoped_drain_expected(record, extra):
        raise UninstallError("SCOPED_DRAIN_APPROVED_EFFECT_SET_DRIFT")
    if exact and fresh["witness"] != record["last_witness"]:
        raise UninstallError("SCOPED_DRAIN_EXACT_WITNESS_DRIFT")
    if extra is not None:
        if fresh["witness"]["operations"] != record["last_witness"]["operations"] + 1:
            raise UninstallError("SCOPED_DRAIN_UNEXPECTED_ORIGINAL_COUNT")
        step = record["steps"][record["index"]]
        l = fresh["observation"]["observation"]["ledger"]
        if step["action"] == "box_peer_set":
            matches = [p for p in l["peers"] if p["source"] == step["box"] and p["target"] == step["target"]]
            if len(matches) != 1 or matches[0]["state"] != "REVOKED" or matches[0].get("last_verified_original") != extra["id"]:
                raise UninstallError("SCOPED_DRAIN_PEER_ORIGINAL_POSTSTATE_UNVERIFIED")
        if step["action"] == "port_forward_delete":
            matches = [f for f in l["forwards"] if f["id"] == step["forward"]]
            if len(matches) != 1 or matches[0]["state"] != "REVOKED" or matches[0].get("original") != extra["id"]:
                raise UninstallError("SCOPED_DRAIN_FORWARD_ORIGINAL_POSTSTATE_UNVERIFIED")
    return fresh


def scoped_drain_call(record, probe):
    # Root command reads only the fixed root-private current receipt. No
    # arbitrary native args, bearer token, or external MCP session.
    args = [str(BINARY), "native-scoped-drain-step"]
    if probe:
        args.append("--read-only")
    raw, code = command(args, acceptable=(0, 1), timeout=1040)
    try:
        out = json.loads(raw)
    except ValueError:
        raise UninstallError("SCOPED_DRAIN_ORIGINAL_OUTCOME_UNVERIFIED: inspect original receipt")
    if not isinstance(out, dict) or type(out.get("found")) is not bool:
        raise UninstallError("SCOPED_DRAIN_ORIGINAL_OUTCOME_UNVERIFIED")
    if not out["found"] and not code and probe:
        return None
    rec = out.get("receipt") or {}
    step = record["steps"][record["index"]]
    if (code or not out["found"] or rec.get("state") != "SUCCEEDED" or
            rec.get("action") != step["action"] or rec.get("box_uuid") != step["box"] or
            rec.get("incarnation") != step["box"] or rec.get("work_ref") != "scoped-drain-" + record["actor"]["id"] or
            not re.fullmatch(r"[a-f0-9]{32}", rec.get("operation_id") or "")):
        state_value = rec.get("state", "UNVERIFIED")
        error_class = rec.get("error_class", "UNVERIFIED")
        if not isinstance(state_value, str) or not re.fullmatch(r"[A-Z_]{1,64}", state_value):
            state_value = "UNVERIFIED"
        if not isinstance(error_class, str) or not re.fullmatch(r"[A-Z_0-9]{1,96}", error_class):
            error_class = "UNVERIFIED"
        raise UninstallError("SCOPED_DRAIN_ORIGINAL_UNSETTLED: state=" + state_value +
                             "; class=" + error_class + "; preserve exact original; never retry uncertain effect")
    return {"id": rec["operation_id"]}


def scoped_drain_resume_preview(project):
    r = scoped_drain_read(project)
    status, blocker = "REVIEW_ONLY", None
    try:
        if r["phase"] == "APPROVED":
            active = systemd_status() == "active"
            scoped_drain_verify(r, active)
        elif r["index"] < len(r["steps"]):
            prior = scoped_drain_call(r, True)
            scoped_drain_verify(r, extra=prior, exact=prior is None)
        else:
            scoped_drain_verify(r)
    except (UninstallError, OSError, ValueError, KeyError, TypeError) as exc:
        status, blocker = "BLOCKED", str(exc)
    return {"mode": DRAIN_MODE, "project": project, "phase": r["phase"], "status": status,
            "ready": status == "REVIEW_ONLY", "review_sha256": r["review_sha256"],
            "receipt_sha256": file_fingerprint(SCOPED_DRAIN)["sha256"],
            "completed": r["completed"], "remaining": len(r["steps"]) - r["index"],
            "blocker": blocker}


def scoped_drain_continue(record):
    project = record["project"]
    r = scoped_drain_read(project)
    if r != record:
        raise UninstallError("original drain changed")
    if r["phase"] == "APPROVED":
        active = systemd_status() == "active"
        scoped_drain_verify(r, active)
        if active:
            command(["systemctl", "stop", UNIT.name], timeout=90)
        scoped_inactive(UNIT.name)
        scoped_drain_verify(r)
        r["phase"] = "DRAINING"
        scoped_drain_checkpoint(r)
    while r["index"] < len(r["steps"]):
        scoped_inactive(UNIT.name)
        prior = scoped_drain_call(r, True)
        if prior is None:
            scoped_drain_verify(r)
            prior = scoped_drain_call(r, False)
        # A lost local ACK advances only from the exact durable original,
        # and the independently observed expected prefix. UNKNOWN never
        # becomes a new dispatch or a fabricated success.
        after = scoped_drain_verify(r, extra=prior, exact=False)
        r["completed"].append(prior)
        r["index"] += 1
        r["last_witness"] = after["witness"]
        r["expected_ledger_sha256"] = after["witness"]["ledger"]
        r["expected_native_sha256"] = after["witness"]["native"]
        scoped_drain_checkpoint(r)
    scoped_drain_verify(r)
    if r["phase"] != "DRAINED":
        r["phase"] = "DRAINED"
        scoped_drain_checkpoint(r)
    # Transition to the original #182 empty native retirement protocol.
    # Keep the first root fence through the handoff and archive its receipt.
    with scoped_lock(MAINTENANCE_LOCK):
        if scoped_drain_read(project) != r:
            raise UninstallError("drain receipt changed at handoff")
        scoped_drain_verify(r)
        plan = scoped_plan(project)
        if not plan["ready"]:
            raise UninstallError("SCOPED_DRAIN_EMPTY_RETIREMENT_BLOCKED")
        if plan.get("scoped_effective_reviews", {}) != r["approved_plan"]["capture"]["immutable"].get("scoped_effective_reviews", {}):
            raise UninstallError("SCOPED_DRAIN_HANDOFF_POLICY_DRIFT")
        plan["drain_receipt"] = scoped_private_file(SCOPED_DRAIN, 1024 * 1024)
        plan["drain_approved_review"] = r["review_sha256"]
        inputs = {k: v for k, v in plan.items() if k not in ("ready", "actions", "retained", "review_sha256")}
        plan["review_sha256"] = sha(canonical(inputs))
        audited_write({"status": "STARTED", "mode": SCOPED_MODE, "scope": "full",
                       "host_sha256": plan["host_machine_id_sha256"], "review_sha256": plan["review_sha256"],
                       "phase": "APPROVED", "scoped_plan": plan, "archive": ""})
        final = scoped_audit(project)
        scoped_set_fence(final)
    return scoped_continue(final)


def scoped_drain_approve(project, snapshot, resume=False):
    if not snapshot["ready"]:
        raise UninstallError("normal Scoped retirement blocked; preserve exact original")
    prompt = ("RESUME SCOPED " if resume else "REMOVE SCOPED ") + project
    with open("/dev/tty", "r") as inp, open("/dev/tty", "w") as out:
        out.write("Type " + prompt + " to approve the exact scoped retirement: ")
        out.flush()
        if inp.readline().strip() != prompt:
            raise UninstallError("OPERATOR_APPROVAL_REQUIRED")
    with scoped_lock(SCOPED_LOCK):
        with scoped_lock(MAINTENANCE_LOCK):
            fresh = scoped_drain_resume_preview(project) if resume else scoped_drain_plan(project)
            if fresh != snapshot:
                raise UninstallError("normal scoped retirement evidence changed after approval")
            r = scoped_drain_read(project) if resume else scoped_drain_begin(fresh)
        return scoped_drain_continue(r)




# Offline retirement is a distinct authorization, not normal recovery.
# The initial bounded executable leaf is Legacy with its exact baseline ACL,
# no uncertain network effects and a stopped controller. Other modes and
# network originals retain typed blockers and cannot use the normal drain.
OFFLINE_MODE = "LEGACY_OFFLINE_RETIREMENT_V1"
OFFLINE_FENCE = Path("/var/lib/boxweave-offline-retirement-v1.json")


def offline_scope_blockers(report, preview):
    i = report.get("installation") or {}
    n = report.get("native") or {}
    l = report.get("ledger") or {}
    identity = i.get("identity") or {}
    originals = l.get("nonterminal_or_unknown_originals", [])
    blockers = []
    if i.get("mode") != "LEGACY":
        blockers.append("OFFLINE_SCOPED_GUARD_OR_MODE_REQUIRES_SEPARATE_PROOF")
    if i.get("adapter") not in ("PINNED_RC13", "EXACT_RUNNING_OBSERVER_BINARY"):
        blockers.append("OFFLINE_INSTALLED_IDENTITY_UNSUPPORTED")
    if identity.get("installed_identity") != "VERIFIED":
        blockers.append("OFFLINE_INSTALLED_IDENTITY_UNVERIFIED")
    if identity.get("service_unit") not in ("INACTIVE", "FAILED"):
        blockers.append("OFFLINE_CONTROLLER_MUST_BE_STOPPED")
    if n.get("legacy_lease_proof") != "VERIFIED":
        blockers.append("OFFLINE_EXACT_LEASE_PROOF_UNVERIFIED")
    if n.get("base_acl_config") != "RECORDED_PREIMAGE_MATCH":
        blockers.append("OFFLINE_LEGACY_ACL_DELTA_REQUIRES_PROOF")
    if any(p["state"] != "REVOKED" for p in l.get("peers", [])):
        blockers.append("OFFLINE_PEER_POLICY_REQUIRES_PROOF")
    if any(f["state"] != "REVOKED" for f in l.get("forwards", [])):
        blockers.append("OFFLINE_FORWARD_POLICY_REQUIRES_PROOF")
    if l.get("legacy_pending_retirements") or l.get("legacy_compactions"):
        blockers.append("OFFLINE_LEGACY_RETIREMENT_JOURNAL_REQUIRES_PROOF")
    owners = {b["uuid"]: b for b in l.get("boxes", [])}
    matched = set(n.get("owned_matched") or [])
    power = n.get("owned_power") or {}
    exact = (set(owners) == matched == set(power) and
             all(b["uuid"] == b["incarnation"] and b.get("create_proof") and
                 b.get("binding_state") == "PINNED" and power[b["uuid"]] in ("Running", "Stopped")
                 for b in owners.values()))
    if not exact:
        blockers.append("OFFLINE_EXACT_OWNED_INVENTORY_UNVERIFIED")
    eligible = []
    for original in originals:
        # Bounded power-only leaf. A receipt is not proof of success. The
        # stopped controller, independent native quiescence and exact owner
        # scope constrain retirement without resolving the original outcome.
        b = owners.get(original.get("box_uuid"))
        if (original.get("state") == "UNKNOWN" and original.get("action") == "box_stop" and
                original.get("retirement_scope") == "EXACT_OWNED_POWER_ORIGINAL" and
                original.get("claim_protection") == "RETAINED" and original.get("recorded_claims") == 1 and
                original.get("native_receipt") is True and b is not None and
                b["incarnation"] == original.get("incarnation") and power.get(b["uuid"]) == "Stopped"):
            eligible.append(original)
        else:
            blockers.append("OFFLINE_ORIGINAL_SCOPE_UNPROVEN:" + original["id"])
    eligible_ids = {x["id"] for x in eligible}
    for issue in preview["blockers"]:
        if issue["severity"] == "INFO":
            continue
        if issue["code"] == "CONTROLLER_RUNTIME_UNVERIFIED" and identity.get("service_unit") in ("INACTIVE", "FAILED"):
            continue
        if issue["code"] == "ORIGINAL_UNKNOWN" and issue["resource"] in eligible_ids:
            continue
        if issue["code"] == "EXEC_ARCHIVE_POST_FENCE_PROOF_REQUIRED":
            continue  # exact file/archive proof below
        blockers.append(issue["code"] + ":" + issue["resource"])
    return sorted(set(blockers)), eligible


def offline_native_consumers(plan, instances):
    # Never treat a same-Project but unowned profile as an owned consumer.
    # Read the exact original targets; the native rollback reader binds
    # their configuration separately.
    if prepare_record() != plan["native_prepare"]:
        raise UninstallError("OFFLINE_ORIGINAL_NATIVE_CHANGED")
    journal = read_json(NATIVE)
    jplan = journal.get("plan", journal.get("Plan", {}))
    target = jplan.get("target", jplan.get("Target", {}))
    project = plan["project"]
    if target.get("project") != project:
        raise UninstallError("OFFLINE_NATIVE_PROJECT_CHANGED")
    allowed = {"/1.0/profiles/default?project=" + project}
    allowed.update("/1.0/instances/" + b["name"] + "?project=" + project for b in instances)
    for kind, field in (("networks", "network"), ("network-acls", "acl")):
        name = target.get(field)
        if not isinstance(name, str) or not VALID_NAME.fullmatch(name):
            raise UninstallError("OFFLINE_NATIVE_TARGET_UNVERIFIED")
        raw, _ = command(["incus", "query", "local:/1.0/" + kind + "/" + name + "?project=default"], timeout=25)
        refs = json.loads(raw).get("used_by")
        if not isinstance(refs, list) or any(not isinstance(x, str) or x not in allowed for x in refs):
            raise UninstallError("OFFLINE_FOREIGN_NATIVE_CONSUMER")


def offline_ledger_unchanged(plan):
    expected = {f["name"]: f["info"] for f in plan["state_files"]}
    files = cold_sqlite_files(STATE / "ledger.sqlite")
    return (files[""] is not None and all(files[suffix] == expected.get("ledger.sqlite" + suffix)
                                         for suffix in files) and
            inspect_sqlite(STATE / "ledger.sqlite", guided=True, cold=True) == plan["sqlite"])


def offline_inactive(project, plan, require_disabled=False):
    if systemd_status() not in ("inactive", "failed") or effective_unit() != str(UNIT):
        raise UninstallError("OFFLINE_CONTROLLER_INACTIVE_UNVERIFIED")
    props = scoped_properties(UNIT.name, "ActiveState", "MainPID", "UnitFileState")
    if props["ActiveState"] not in ("inactive", "failed") or props["MainPID"] != "0":
        raise UninstallError("OFFLINE_CONTROLLER_PID_UNVERIFIED")
    if require_disabled and props["UnitFileState"] != "disabled":
        raise UninstallError("OFFLINE_CONTROLLER_AUTOSTART_NOT_FENCED")
    require_retirement_stopped(plan)
    if plan.get("retirement_verifier"):
        props = scoped_properties(RETIREMENT_UNIT.name, "ActiveState", "MainPID", "UnitFileState")
        if props["ActiveState"] not in ("inactive", "failed") or props["MainPID"] != "0":
            raise UninstallError("OFFLINE_HELPER_PID_UNVERIFIED")
        if require_disabled and props["UnitFileState"] != "disabled":
            raise UninstallError("OFFLINE_HELPER_AUTOSTART_NOT_FENCED")
    for path in (ROOT_UNIT, Path("/run/boxweave-native-peer-root.sock"),
                 Path("/var/lib/boxweave-native-peer-conntrack-v1"),
                 Path("/var/lib/boxweave-native-peer-conntrack-v1.json"),
                 Path("/var/lib/boxweave-native-peer-conntrack-v1.lock"),
                 Path("/var/lib/boxweave-native-peer-guard-v1")):
        if os.path.lexists(path):
            raise UninstallError("OFFLINE_LEGACY_ROOT_ORIGINAL_REQUIRES_PROOF")
    root = scoped_properties(ROOT_UNIT.name, "LoadState", "ActiveState", "MainPID")
    if root != {"LoadState": "not-found", "ActiveState": "inactive", "MainPID": "0"}:
        raise UninstallError("OFFLINE_ROOT_UNIT_EFFECTIVE_STATE_UNVERIFIED")
    if scoped_incus_active() != plan["offline_incus_pid"]:
        raise UninstallError("OFFLINE_INCUS_RUNTIME_CHANGED")


def offline_plan(project, observer=None):
    uid = pwd.getpwnam("boxweave").pw_uid
    host = machine_sha()
    if uid == 0 or project != "user-" + str(uid):
        raise UninstallError("exact dedicated offline retirement Project required")
    before = maintenance_local_inventory(project, uid)
    report = {}
    failure = None
    try:
        if before.get("mounted_artifacts") or before.get("capture"):
            raise UninstallError("unsafe or unavailable mount inventory")
        report = maintenance_observe(observer["path"] if observer else BINARY,
                                     observer["sha256"] if observer else None, project)
    except (UninstallError, OSError, ValueError, TypeError, subprocess.TimeoutExpired) as exc:
        failure = type(exc).__name__
    local = maintenance_local_inventory(project, uid)
    if before != local or host != machine_sha():
        report = {}
        failure = "LOCAL_EVIDENCE_CHANGED"
    preview = maintenance_preview(project, "retain", host, report, local,
                                  sha(Path(__file__).read_bytes()))
    blockers, originals = offline_scope_blockers(report, preview)
    if failure:
        blockers.append("OFFLINE_OBSERVER_UNAVAILABLE:" + failure)
    if os.path.lexists(OFFLINE_FENCE) or local.get("audit", {}).get("state") == "STARTED":
        blockers.append("OFFLINE_ORIGINAL_RETIREMENT_REQUIRES_REVIEW")
    if not any(x["name"] == SERVE_LOCK_NAME and x["safe"]
               for x in (local.get("state") or {}).get("files", [])):
        blockers.append("OFFLINE_ORIGINAL_SERVE_LOCK_UNVERIFIED")
    if not observer and (report.get("installation") or {}).get("adapter") == "PINNED_RC13":
        blockers.append("OFFLINE_PINNED_CURRENT_READ_ONLY_ADAPTER_REQUIRED")
    if any(local.get(k, {}).get("state") != "ABSENT" for k in
           ("backup", "scriptlet", "legacy_root_operations", "scoped_fence", "purge_fence")):
        blockers.append("OFFLINE_MIXED_OR_INTERRUPTED_INSTALLATION")
    plan = None
    if not blockers:
        try:
            plan = inspect("full", project, [], None, guided=True,
                           offline_observer=observer, offline=True)
            plan["offline_incus_pid"] = scoped_incus_active()
            offline_inactive(project, plan)
            if plan["native_prepare"]["status"] != "APPLIED" or not plan["native_retirement_precheck"]:
                blockers.append("OFFLINE_EXACT_APPLIED_PREIMAGE_REQUIRED")
            if plan.get("installer_backup") or plan.get("scriptlet_present"):
                blockers.append("OFFLINE_MIXED_OR_INTERRUPTED_INSTALLATION")
            if plan["service_state"] not in ("inactive", "failed"):
                blockers.append("OFFLINE_CONTROLLER_MUST_BE_STOPPED")
            if plan["sqlite"]["version"] != 6:
                blockers.append("OFFLINE_SCHEMA_UNSUPPORTED")
            if plan["sqlite"]["unresolved_operations"] != len(originals):
                blockers.append("OFFLINE_ORIGINAL_INVENTORY_CHANGED")
            ids = {x["id"] for x in originals}
            remaining = []
            for b in plan["blockers"]:
                if b.startswith("OPERATION_UNKNOWN:") and b.split(":")[1] in ids:
                    continue
                if b == "network_address_fences_NOT_RELEASED:" + str(2 * len(plan["incus_instances"])):
                    continue  # complete exact live lease proof above
                remaining.append(b)
            blockers.extend(remaining)
            native = report["native"]
            expected = sorted((b["uuid"], b["name"], native["owned_power"][b["uuid"]])
                              for b in report["ledger"]["boxes"])
            actual = sorted((b["uuid"], b["name"], b["state"]) for b in plan["incus_instances"])
            if expected != actual:
                blockers.append("OFFLINE_NATIVE_SCOPE_CHANGED")
            offline_native_consumers(plan, plan["incus_instances"])
            if maintenance_local_inventory(project, uid) != local:
                blockers.append("OFFLINE_LOCAL_EVIDENCE_CHANGED")
        except (UninstallError, OSError, ValueError, KeyError, TypeError, subprocess.TimeoutExpired) as exc:
            detail = str(exc)
            blockers.append(detail if isinstance(exc, UninstallError) and re.fullmatch(r"OFFLINE_SQLITE_[A-Z_]+", detail)
                            else "OFFLINE_STATIC_PROOF_UNVERIFIED:" + type(exc).__name__)
    result = {"schema": 1, "mode": OFFLINE_MODE, "scope": "full", "project": project,
              "host_machine_id_sha256": host, "retention": "retain",
              "ready": not blockers and plan is not None,
              "blockers": sorted(set(blockers)), "observation": preview,
              "unresolved_originals": originals,
              "business_operations_replayed": False,
              "operation_outcome_changed": False}
    if result["ready"]:
        # Bind the actual complete retirement plan, not a presentation-only
        # H1 preview digest. Original SQLite blocker facts remain intact.
        plan["mode"] = OFFLINE_MODE
        plan["offline_observer"] = observer
        plan["offline_originals"] = originals
        plan["offline_ledger_sha256"] = report["ledger_observation_sha256"]
        plan["offline_native_sha256"] = report["native"]["evidence"]["sha256"]
        plan["blockers"] = []
        plan["ready"] = True
        plan["actions"] = ["fence and disable only original inactive controller/helper",
                           "archive unchanged ledger and original UNKNOWN facts",
                           "stop/delete only exact approved owned Boxes",
                           "restore baseline native preparation and retire exact files",
                           "retain all original business outcomes; no recovery replay"]
        plan["review_sha256"] = sha(canonical({k: v for k, v in plan.items()
                                               if k not in ("review_sha256", "ready", "actions", "retained")}))
        result["review_sha256"] = plan["review_sha256"]
        result["plan"] = plan
        if len(canonical(result)) > 768 * 1024:
            raise UninstallError("offline review exceeds durable receipt bound")
    return result


def offline_checkpoint(plan, phase, **updates):
    record = read_json(AUDIT)
    if (record is None or record.get("status") != "STARTED" or
            record.get("mode") != OFFLINE_MODE or record.get("review_sha256") != plan["review_sha256"]):
        raise UninstallError("OFFLINE_ORIGINAL_AUDIT_CHANGED")
    if read_json(OFFLINE_FENCE) != {"schema": 1, "project": plan["project"],
                                    "review_sha256": plan["review_sha256"]}:
        raise UninstallError("OFFLINE_ORIGINAL_FENCE_CHANGED")
    if "deleted" in updates:
        deleted = updates.pop("deleted")
        prefix = record.get("verified_deleted", [])
        expected = [x["uuid"] for x in plan["incus_instances"]]
        if prefix != expected[:len(prefix)] or len(prefix) >= len(expected) or deleted != expected[len(prefix)]:
            raise UninstallError("OFFLINE_DELETE_CHECKPOINT_MISMATCH")
        record["verified_deleted"] = prefix + [deleted]
    record.update(updates, phase=phase)
    _replace_audit(record)


def offline_archive_checkpoint(plan, archive):
    path = Path(archive)
    if path.parent != RETIRED or not re.fullmatch(r"retired-[a-z0-9_]{8}", path.name):
        raise UninstallError("OFFLINE_ARCHIVE_PATH_UNVERIFIED")
    scoped_private_dir(RETIRED)
    scoped_private_dir(path)
    if read_json(path / "review.json") != plan:
        raise UninstallError("OFFLINE_ARCHIVE_REVIEW_CHANGED")
    for f in plan["state_files"]:
        proof = scoped_private_file(path / ("state-" + f["name"]))
        if proof["sha256"] != f["info"]["sha256"]:
            raise UninstallError("OFFLINE_ARCHIVED_ORIGINAL_CHANGED")
    offline_checkpoint(plan, "ARCHIVED", archive=str(path), archive_content_sha256=legacy_archive_content(path, plan, historical=True))


def offline_resume_record(project):
    scoped_private_file(AUDIT, 1024 * 1024)
    record = read_json(AUDIT)
    plan = record.get("offline_plan")
    if (record.get("mode") != OFFLINE_MODE or record.get("scope") != "full" or
            record.get("status") not in ("STARTED", "COMPLETE") or not isinstance(plan, dict) or
            plan.get("mode") != OFFLINE_MODE or not plan.get("ready") or plan.get("project") != project or
            plan.get("host_machine_id_sha256") != machine_sha() or record.get("host_sha256") != machine_sha() or
            record.get("review_sha256") != plan.get("review_sha256") or
            sha(canonical({k: v for k, v in plan.items() if k not in ("review_sha256", "ready", "actions", "retained")})) != plan.get("review_sha256")):
        raise UninstallError("OFFLINE_RESUME_ORIGINAL_IDENTITY_UNVERIFIED")
    if record.get("phase") not in ("NATIVE_RESTORED", "CLEANUP_STARTED"):
        raise UninstallError("OFFLINE_RESUME_NATIVE_PREFIX_UNPROVEN: preserve original; earlier native phase requires separate proof")
    expected_fence = {"schema": 1, "project": project, "review_sha256": plan["review_sha256"]}
    if os.path.lexists(OFFLINE_FENCE):
        scoped_private_file(OFFLINE_FENCE, 8192)
        if read_json(OFFLINE_FENCE) != expected_fence:
            raise UninstallError("OFFLINE_RESUME_FENCE_MISMATCH")
    elif record["status"] != "COMPLETE":
        raise UninstallError("OFFLINE_RESUME_FENCE_MISSING")
    if record.get("verified_deleted") != [b["uuid"] for b in plan["incus_instances"]] or record.get("unresolved_originals") != plan["offline_originals"]:
        raise UninstallError("OFFLINE_RESUME_ORIGINAL_PREFIX_UNVERIFIED")
    archive = Path(record.get("archive", ""))
    if archive.parent != RETIRED or not re.fullmatch(r"retired-[a-z0-9_]{8}", archive.name):
        raise UninstallError("OFFLINE_RESUME_ARCHIVE_UNVERIFIED")
    purge_mount_boundary()
    scoped_private_dir(RETIRED)
    scoped_private_dir(archive)
    content = legacy_archive_content(archive, plan, historical=True)
    if record.get("archive_content_sha256", record.get("result", {}).get("archive_content_sha256")) != content:
        raise UninstallError("OFFLINE_RESUME_ARCHIVE_CONTENT_CHANGED")
    return record, content


def offline_restored_native(record):
    plan = record["offline_plan"]
    archive = Path(record["archive"])
    if os.path.lexists(NATIVE):
        raise UninstallError("OFFLINE_RESUME_NATIVE_JOURNAL_NOT_ARCHIVED")
    original = read_json(archive / "native-prepare.json")
    if not original or not closed_native_review_valid(original.get("plan", {})):
        raise UninstallError("OFFLINE_RESUME_NATIVE_PREIMAGE_UNVERIFIED")
    closed = {**original, "status": "ROLLED_BACK"}
    prefix = original["plan"]["review_sha256"][:16]
    matching = []
    for path in NATIVE.parent.glob(NATIVE.name + ".closed." + prefix + ".*"):
        if not re.fullmatch(re.escape(NATIVE.name) + r"\.closed\.[a-f0-9]{16}\.[0-9]{2}", path.name):
            raise UninstallError("OFFLINE_RESUME_CLOSED_JOURNAL_NAME_UNVERIFIED")
        scoped_private_file(path, 1024 * 1024)
        if read_json(path) != closed:
            raise UninstallError("OFFLINE_RESUME_CLOSED_JOURNAL_CHANGED")
        matching.append(path)
    if not matching:
        raise UninstallError("OFFLINE_RESUME_CLOSED_JOURNAL_MISSING")
    legacy_retired_native_postcheck(plan["project"], original)
    return purge_native_empty(plan["project"])


def offline_cleanup_inventory(record):
    plan = record["offline_plan"]
    refuse_scoped_retirement()
    if any(os.path.lexists(x) for x in (ROOT_UNIT, RETIREMENT_SOCKET, BACKUP, SCRIPTLET,
                  Path("/run/boxweave-native-peer-root.sock"), Path("/var/lib/boxweave-native-peer-conntrack-v1"),
                  Path("/var/lib/boxweave-native-peer-conntrack-v1.lock"))):
        raise UninstallError("OFFLINE_RESUME_UNREVIEWED_ARTIFACT")
    points = maintenance_mounts()
    for path in (STATE, UNIT, BINARY, RETIREMENT_UNIT, NATIVE):
        if maintenance_under_mount(path, points):
            raise UninstallError("OFFLINE_RESUME_MOUNTED_ARTIFACT")
    remaining = {}
    for path, proof in ((UNIT, plan["unit"]), (BINARY, plan["binary"]),
                       (RETIREMENT_UNIT, (plan.get("retirement_verifier") or {}).get("unit"))):
        if os.path.lexists(path):
            if proof is None or file_fingerprint(path) != proof or path.lstat().st_nlink != 1:
                raise UninstallError("OFFLINE_RESUME_ARTIFACT_CHANGED")
            remaining[str(path)] = proof
    for unit in (UNIT, RETIREMENT_UNIT, ROOT_UNIT):
        props = scoped_properties(unit.name, "LoadState", "FragmentPath", "DropInPaths", "ActiveState", "MainPID", "UnitFileState")
        if props["ActiveState"] not in ("inactive", "failed") or props["MainPID"] != "0" or props["DropInPaths"]:
            raise UninstallError("OFFLINE_RESUME_SERVICE_NOT_INACTIVE")
        if props["LoadState"] != "not-found" and (props["FragmentPath"] != str(unit) or props["UnitFileState"] not in ("disabled", "")):
            raise UninstallError("OFFLINE_RESUME_EFFECTIVE_UNIT_UNVERIFIED")
        if unit == ROOT_UNIT and props["LoadState"] != "not-found":
            raise UninstallError("OFFLINE_RESUME_ROOT_UNIT_UNREVIEWED")
    if os.path.lexists(STATE):
        validate_state_dir(STATE, plan["service_user_uid"])
        expected = {x["name"]: x["info"] for x in plan["state_files"]}
        if {x.name for x in STATE.iterdir()} - set(expected) - {EXEC_ARCHIVE_NAME}:
            raise UninstallError("OFFLINE_RESUME_STATE_UNKNOWN_ENTRY")
        for name, proof in expected.items():
            path = STATE / name
            if os.path.lexists(path):
                if file_fingerprint(path) != proof or path.lstat().st_nlink != 1:
                    raise UninstallError("OFFLINE_RESUME_STATE_CHANGED")
                remaining[str(path)] = proof
        if os.path.lexists(STATE / EXEC_ARCHIVE_NAME):
            current = inspect_exec_archive(plan["service_user_uid"])
            approved = plan.get("exec_archive")
            if not approved or (current["device"], current["inode"]) != (approved["device"], approved["inode"]):
                raise UninstallError("OFFLINE_RESUME_EXEC_ARCHIVE_CHANGED")
            byname = {x["name"]: x["info"] for x in approved["files"]}
            for item in current["files"]:
                if byname.get(item["name"]) != item["info"]:
                    raise UninstallError("OFFLINE_RESUME_EXEC_FILE_CHANGED")
                remaining[str(STATE / EXEC_ARCHIVE_NAME / item["name"])] = item["info"]
        if not os.path.lexists(STATE / SERVE_LOCK_NAME) and list(STATE.iterdir()):
            raise UninstallError("OFFLINE_RESUME_SERVE_LOCK_MISSING")
    return remaining


def offline_cleanup_files(record):
    plan = record["offline_plan"]
    current, content = offline_resume_record(plan["project"])
    if current != record:
        raise UninstallError("OFFLINE_RESUME_ORIGINAL_CHANGED")
    native = offline_restored_native(record)
    remaining = offline_cleanup_inventory(record)
    if record["phase"] == "NATIVE_RESTORED":
        offline_checkpoint(plan, "CLEANUP_STARTED")
    for unit in (UNIT, RETIREMENT_UNIT):
        if str(unit) in remaining:
            exact_unlink(unit, remaining[str(unit)])
            scoped_sync_dir(unit.parent)
    command(["systemctl", "daemon-reload"])
    for raw, proof in remaining.items():
        path = Path(raw)
        if STATE in path.parents and path != STATE / SERVE_LOCK_NAME:
            exact_unlink(path, proof)
            scoped_sync_dir(path.parent)
    if os.path.lexists(STATE / EXEC_ARCHIVE_NAME):
        (STATE / EXEC_ARCHIVE_NAME).rmdir()
        scoped_sync_dir(STATE)
    if str(STATE / SERVE_LOCK_NAME) in remaining:
        exact_unlink(STATE / SERVE_LOCK_NAME, remaining[str(STATE / SERVE_LOCK_NAME)])
        scoped_sync_dir(STATE)
    if os.path.lexists(STATE):
        STATE.rmdir()
        scoped_sync_dir(STATE.parent)
    if str(BINARY) in remaining:
        exact_unlink(BINARY, remaining[str(BINARY)])
        scoped_sync_dir(BINARY.parent)
    for parent in {STATE.parent, UNIT.parent, BINARY.parent}:
        scoped_sync_dir(parent)
    purge_absent(allow_completed_offline_fence=True)
    if offline_restored_native(record) != native or legacy_archive_content(Path(record["archive"]), plan, historical=True) != content:
        raise UninstallError("OFFLINE_RESUME_POSTFLIGHT_CHANGED")
    return {"outcome": "COMPLETE", "scope": "full", "mode": OFFLINE_MODE,
            "host_machine_id_sha256": plan["host_machine_id_sha256"],
            "approved_review_sha256": plan["review_sha256"], "retirement_archive": record["archive"],
            "archive_content_sha256": content, "retained": plan["retained"],
            "unresolved_originals": plan["offline_originals"], "business_operations_replayed": False,
            "operation_outcome_changed": False}


def offline_resume_preview(project):
    record, content = offline_resume_record(project)
    native = offline_restored_native(record)
    remaining = offline_cleanup_inventory(record)
    if record["status"] == "COMPLETE":
        offline_completed_archive(record, record["offline_plan"], Path(record["archive"]))
        purge_absent(allow_completed_offline_fence=True)
        if remaining:
            raise UninstallError("OFFLINE_RESUME_COMPLETE_HAS_REMAINING_FILES")
    return {"ready": True, "mode": OFFLINE_MODE, "status": record["status"], "phase": record["phase"],
            "project": project, "review_sha256": record["review_sha256"], "archive_content_sha256": content,
            "audit_sha256": scoped_private_file(AUDIT, 1024 * 1024)["sha256"], "native_postflight_sha256": native,
            "remaining_files": remaining, "retention": "retain"}


def offline_resume_main(args):
    if os.geteuid() != 0 or sys.platform != "linux":
        raise UninstallError("offline resume requires original local Linux root")
    project = args.project or "user-" + str(pwd.getpwnam("boxweave").pw_uid)
    if project != "user-" + str(pwd.getpwnam("boxweave").pw_uid):
        raise UninstallError("exact original offline Project required")
    preview = offline_resume_preview(project)
    print(json.dumps(preview, sort_keys=True, indent=2), flush=True)
    if args.offline_resume_plan:
        return 0
    prompt = "RESUME OFFLINE " + project
    with open("/dev/tty", "r") as inp, open("/dev/tty", "w") as out:
        out.write("Type " + prompt + " to continue the original offline retirement: ")
        out.flush()
        if inp.readline().strip() != prompt:
            raise UninstallError("OPERATOR_APPROVAL_REQUIRED")
    with scoped_lock(SCOPED_LOCK), scoped_lock(MAINTENANCE_LOCK):
        if offline_resume_preview(project) != preview:
            raise UninstallError("OFFLINE_RESUME_REVIEW_CHANGED")
        record, _ = offline_resume_record(project)
        plan = record["offline_plan"]
        if record["status"] == "STARTED":
            lockplan = plan if os.path.lexists(STATE / SERVE_LOCK_NAME) else {**plan, "state_files": []}
            with hold_stopped_serve_lock(lockplan):
                result = offline_cleanup_files(record)
                audited_complete(plan["review_sha256"], result)
        else:
            result = record["result"]
        offline_resume_preview(project)
        if os.path.lexists(OFFLINE_FENCE):
            exact_unlink(OFFLINE_FENCE, scoped_private_file(OFFLINE_FENCE, 8192))
            scoped_sync_dir(OFFLINE_FENCE.parent)
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


def offline_main(args):
    if os.geteuid() != 0 or sys.platform != "linux":
        raise UninstallError("offline retirement requires local Linux root")
    project = args.project or "user-" + str(pwd.getpwnam("boxweave").pw_uid)
    observer = ({"path": str(Path(args.observer_binary).absolute()), "sha256": args.observer_sha256}
                if args.observer_binary else None)
    snapshot = offline_plan(project, observer)
    print(json.dumps(snapshot, indent=2, sort_keys=True), flush=True)
    if args.offline_plan:
        return 0 if snapshot["ready"] else 2
    if not snapshot["ready"]:
        raise UninstallError("OFFLINE_RETIREMENT_BLOCKED: preserve originals and review exact blockers")
    prompt = "RETIRE OFFLINE " + project
    with open("/dev/tty", "r") as inp, open("/dev/tty", "w") as out:
        out.write("Type " + prompt + " to approve permanent deletion of the exact listed owned Boxes and retain original evidence: ")
        out.flush()
        if inp.readline().strip() != prompt:
            raise UninstallError("OPERATOR_APPROVAL_REQUIRED: offline retirement not approved")
    with scoped_lock(SCOPED_LOCK):
        plan = snapshot["plan"]
        with hold_stopped_serve_lock(plan):
            if offline_plan(project, observer) != snapshot:
                raise UninstallError("OFFLINE_REVIEW_CHANGED_AFTER_APPROVAL")
            audited_write({"status": "STARTED", "mode": OFFLINE_MODE, "scope": "full",
                           "host_sha256": plan["host_machine_id_sha256"], "review_sha256": plan["review_sha256"],
                           "actions": plan["actions"], "phase": "APPROVED", "verified_deleted": [],
                           "unresolved_originals": plan["offline_originals"], "offline_plan": plan,
                           "historical_released_exec": plan["sqlite"]["historical_released_exec"],
                           "historical_released_other": plan["sqlite"]["historical_released_other"]})
            scoped_write_json(OFFLINE_FENCE, {"schema": 1, "project": project,
                                             "review_sha256": plan["review_sha256"]})
            # Historical binaries do not recognize new fences. Hold their
            # exact original serve flock and disable their exact units before
            # native effects, including across a crash/reboot.
            offline_inactive(project, plan)
            command(["systemctl", "disable", UNIT.name])
            if plan.get("retirement_verifier"):
                command(["systemctl", "disable", RETIREMENT_UNIT.name])
            offline_inactive(project, plan, require_disabled=True)
            offline_checkpoint(plan, "ADMISSION_FENCED")
            result = apply_with_serve_lock(plan, offline=True)
            if scoped_incus_active() != plan["offline_incus_pid"] or native_instances(project, guided=True):
                raise UninstallError("OFFLINE_POSTFLIGHT_NATIVE_UNVERIFIED")
            require_quiescent_native(project)
            for path in (STATE, UNIT, BINARY, NATIVE, RETIREMENT_UNIT, RETIREMENT_SOCKET, ROOT_UNIT):
                if os.path.lexists(path):
                    raise UninstallError("OFFLINE_POSTFLIGHT_ARTIFACT_REMAINS")
            result.update(mode=OFFLINE_MODE, unresolved_originals=plan["offline_originals"],
                          business_operations_replayed=False, operation_outcome_changed=False)
            audited_complete(plan["review_sha256"], result)
            proof = scoped_private_file(OFFLINE_FENCE, 8192)
            if read_json(OFFLINE_FENCE) != {"schema": 1, "project": project,
                                           "review_sha256": plan["review_sha256"]}:
                raise UninstallError("OFFLINE_COMPLETED_FENCE_CHANGED")
            exact_unlink(OFFLINE_FENCE, proof)
            scoped_sync_dir(OFFLINE_FENCE.parent)
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0



def main(argv=None):
    parser = argparse.ArgumentParser(prog="uninstall.sh", description="BoxWeave exact-owned uninstall; explicit full-only Scoped Peer v2 retirement")
    action = parser.add_mutually_exclusive_group()
    action.add_argument("--offline-resume-plan", action="store_true", help="read-only review of original post-native offline retirement")
    action.add_argument("--offline-resume", action="store_true", help="continue exact approved post-native offline retirement")
    action.add_argument("--offline-plan", action="store_true", help="H2B explicit offline retirement review; no effects")
    action.add_argument("--offline-full", action="store_true", help="terminal-approved bounded offline Legacy full retirement; retain originals")
    action.add_argument("--maintenance-plan", action="store_true", help="H1B read-only full retirement preview; no apply authority")
    action.add_argument("--maintenance-recheck", action="store_true", help="read-only preview freshness comparison, not approval")
    action.add_argument("--plan", action="store_true", help="read-only host inventory and review hashes")
    action.add_argument("--apply", action="store_true", help="apply one reviewed plan; requires two exact digests")
    action.add_argument("--full", action="store_true", help="guided full retirement with terminal review and backup")
    action.add_argument("--purge-plan", action="store_true", help="read-only review of completed retirement data erasure")
    action.add_argument("--purge", action="store_true", help="permanently erase reviewed completed retirement data; resume same original if interrupted")
    action.add_argument("--resume-plan", action="store_true", help="read-only review of a stopped, archived STARTED retirement")
    action.add_argument("--resume", action="store_true", help="explicit terminal-approved continuation of the exact archived STARTED retirement")
    parser.add_argument("--scoped", action="store_true", help="explicit full-only Scoped Peer v2 retirement with exact owned Box cleanup")
    parser.add_argument("--retirement-archive", help="exact root-private original retirement archive for resume")
    parser.add_argument("--guided-full-plan", action="store_true",
                        help="with --plan --scope full: use the same guided read-only archive/inventory checks as --full")
    parser.add_argument("--scope", choices=SCOPES, help="service | controller | full (never shared Incus)")
    parser.add_argument("--project", help="exact restricted service Project (user-UID)")
    parser.add_argument("--box-uuid", action="append", default=[], help="full only: complete exact immutable UUID allowlist")
    parser.add_argument("--native-prepare-sha256", help="full only: exact original native journal review digest")
    parser.add_argument("--confirm-host-sha256", help="apply: exact machine SHA printed by plan")
    parser.add_argument("--confirm-review-sha256", help="apply: exact immutable inventory review SHA")
    parser.add_argument("--retention", choices=("retain", "erase"), help="explicit maintenance preview retention intent")
    parser.add_argument("--observer-binary", help="root-local alternate maintenance observer")
    parser.add_argument("--observer-sha256", help="exact SHA256 of alternate observer")
    parser.add_argument("--confirm-preview-sha256", help="maintenance recheck only; never an apply digest")
    args = parser.parse_args(argv)
    if args.offline_resume_plan or args.offline_resume:
        if any((args.scope, args.scoped, args.guided_full_plan, args.retirement_archive, args.box_uuid,
                args.native_prepare_sha256, args.confirm_host_sha256, args.confirm_review_sha256,
                args.confirm_preview_sha256, args.observer_binary, args.observer_sha256, args.retention)):
            parser.error("offline resume uses only the original Project and receipt")
        return offline_resume_main(args)
    if args.offline_plan or args.offline_full:
        if (args.scope not in (None, "full") or args.scoped or args.guided_full_plan or
                args.retirement_archive or args.box_uuid or args.native_prepare_sha256 or
                args.confirm_host_sha256 or args.confirm_review_sha256 or args.confirm_preview_sha256 or
                args.retention not in (None, "retain") or
                bool(args.observer_binary) != bool(args.observer_sha256) or
                (args.observer_sha256 and not VALID_SHA.fullmatch(args.observer_sha256))):
            parser.error("offline retirement requires exact dedicated scope and retention; no apply/force options")
        return offline_main(args)
    if args.maintenance_plan or args.maintenance_recheck:
        if (args.scope not in (None, "full") or args.scoped or args.guided_full_plan or
                args.retirement_archive or args.box_uuid or args.native_prepare_sha256 or
                args.confirm_review_sha256 or not args.retention or
                bool(args.observer_binary) != bool(args.observer_sha256) or
                (args.observer_sha256 and not VALID_SHA.fullmatch(args.observer_sha256))):
            parser.error("maintenance preview requires explicit retention and only exact scope/observer options")
        if args.maintenance_recheck:
            if not VALID_SHA.fullmatch(args.confirm_host_sha256 or "") or not VALID_SHA.fullmatch(args.confirm_preview_sha256 or ""):
                parser.error("maintenance recheck requires exact host and preview SHA256")
        elif args.confirm_host_sha256 or args.confirm_preview_sha256:
            parser.error("preview does not accept confirmation digests")
        return maintenance_main(args)
    if any((args.retention, args.observer_binary, args.observer_sha256, args.confirm_preview_sha256)):
        parser.error("maintenance options cannot be used by existing apply/retirement/purge commands")
    if os.path.lexists(OFFLINE_FENCE):
        raise UninstallError("OFFLINE_RETIREMENT_IN_PROGRESS: review --offline-resume-plan; preserve exact original")
    if args.purge or args.purge_plan:
        return purge_main(args)
    if os.path.lexists(PURGE_FENCE):
        raise UninstallError("PURGE_IN_PROGRESS: review --purge-plan; preserve original receipt")
    if args.scoped:
        return scoped_main(args)
    if not args.plan and not args.apply and not args.full and not args.resume_plan and not args.resume:
        parser.print_help()
        return 0 if not any((args.scope, args.project, args.box_uuid)) else 2
    if args.resume_plan or args.resume:
        if (args.scope or args.guided_full_plan or args.full or args.plan or args.apply or
                args.box_uuid or args.native_prepare_sha256 or
                args.confirm_host_sha256 or args.confirm_review_sha256 or
                not args.retirement_archive):
            parser.error("resume requires only --project and --retirement-archive")
        args.project = args.project or "user-" + str(pwd.getpwnam("boxweave").pw_uid)
        if os.geteuid() != 0 or sys.platform != "linux":
            raise UninstallError("root on the original Host required for retirement recovery")
        snapshot, original = inspect_interrupted_retirement(args.project, args.retirement_archive)
        if args.resume_plan:
            print(json.dumps(snapshot, indent=2, sort_keys=True))
            return 0
        print(json.dumps(snapshot, indent=2, sort_keys=True), flush=True)
        with open("/dev/tty", "r") as inp, open("/dev/tty", "w") as out:
            out.write("Type RESUME " + args.project + " to continue the original audited retirement: ")
            out.flush()
            if inp.readline().strip() != "RESUME " + args.project:
                raise UninstallError("OPERATOR_APPROVAL_REQUIRED: interrupted retirement continuation not approved; original audit unchanged")
        fresh, original = inspect_interrupted_retirement(args.project, args.retirement_archive)
        if fresh != snapshot:
            raise UninstallError("interrupted retirement evidence changed since approval")
        print(json.dumps(resume_interrupted_retirement(original, args.retirement_archive),
                         indent=2, sort_keys=True))
        return 0
    if args.retirement_archive:
        parser.error("--retirement-archive requires --resume or --resume-plan")
    if args.guided_full_plan and (not args.plan or args.scope != "full"):
        parser.error("--guided-full-plan requires --plan --scope full")
    if args.full:
        if args.scope or args.confirm_host_sha256 or args.confirm_review_sha256:
            parser.error("--full uses terminal approval, not explicit scope/digests")
        args.scope = "full"
        args.project = args.project or "user-" + str(pwd.getpwnam("boxweave").pw_uid)
    if not args.scope or not args.project:
        parser.error("--scope and --project required for plan/apply")
    if os.geteuid() != 0 or sys.platform != "linux":
        raise UninstallError("root on the reviewed Linux host required for safe ledger/Incus inspection")
    # A prior STARTED audit must direct operators to the original archive,
    # even when current controller artifacts are already partially removed.
    refuse_scoped_retirement()
    prior_audit = audit_status()
    if prior_audit["status"] == "STARTED":
        hint = "Do not repeat retirement; use --resume-plan --project PROJECT --retirement-archive EXACT_ORIGINAL_ARCHIVE if eligible; otherwise preserve the original audit for local review"
        if args.plan:
            print(json.dumps({"ready": False, "scope": args.scope, "project": args.project,
                              "uninstall_audit": prior_audit, "blockers": ["PRIOR_UNINSTALL_INTERRUPTED_REVIEW_ORIGINAL_AUDIT"],
                              "next_action": hint, "coverage": "ORIGINAL_AUDIT_ONLY"}, indent=2, sort_keys=True))
            return 2
        raise UninstallError("UNINSTALL_AUDIT_STARTED: " + hint)
    plan = inspect(args.scope, args.project, args.box_uuid, args.native_prepare_sha256,
                   guided=args.full or args.guided_full_plan)
    if args.plan:
        print(json.dumps(plan, indent=2, sort_keys=True))
        return 0 if plan["ready"] else 2
    if args.full:
        print(json.dumps(plan, indent=2, sort_keys=True), flush=True)
        if not plan["ready"]:
            raise UninstallError("guided retirement blocked by current evidence; preserve original receipts and inspect plan blockers")
        with open("/dev/tty", "r") as inp, open("/dev/tty", "w") as out:
            out.write("Type REMOVE " + args.project + " to approve deleting all listed Boxes and retiring this controller: ")
            out.flush()
            if inp.readline().strip() != "REMOVE " + args.project:
                raise UninstallError("OPERATOR_APPROVAL_REQUIRED: retirement not approved; no retirement changes")
        fresh = inspect(args.scope, args.project, args.box_uuid, args.native_prepare_sha256, guided=True)
        if fresh["review_sha256"] != plan["review_sha256"] or not fresh["ready"]:
            raise UninstallError("host/scope/inventory drift; review again")
        args.confirm_host_sha256 = plan["host_machine_id_sha256"]
        args.confirm_review_sha256 = plan["review_sha256"]
    if not VALID_SHA.fullmatch(args.confirm_host_sha256 or "") or \
            not VALID_SHA.fullmatch(args.confirm_review_sha256 or ""):
        raise UninstallError("apply requires exact host AND review SHA256 from fresh plan")
    if plan["host_machine_id_sha256"] != args.confirm_host_sha256 or \
            plan["review_sha256"] != args.confirm_review_sha256 or not plan["ready"]:
        raise UninstallError("host/scope/inventory drift or blockers: apply refused")
    refuse_scoped_retirement()
    check_original_native_precondition(plan)
    audited_write({"status": "STARTED", "scope": plan["scope"],
                   "host_sha256": plan["host_machine_id_sha256"],
                   "review_sha256": plan["review_sha256"],
                   "actions": plan["actions"],
                   "historical_released_exec": plan["sqlite"]["historical_released_exec"],
                    "historical_released_other": plan["sqlite"]["historical_released_other"]})
    result = apply(plan)
    # Completion is a separate fsynced audit transition; interruptions keep
    # the original STARTED receipt and require deliberate manual reconciliation.
    audited_complete(plan["review_sha256"], result)
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (UninstallError, OSError, ValueError, KeyError, TypeError) as exc:
        print("BoxWeave uninstaller: " + str(exc), file=sys.stderr)
        sys.exit(1)
