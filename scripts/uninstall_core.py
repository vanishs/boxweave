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
SCOPED_FENCE = Path("/var/lib/boxweave-scoped-retirement-v1.json")
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
                 SCOPED_INCUS_DROPIN, SCOPED_SOCKET, SCOPED_FENCE):
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


def inspect_sqlite(dbpath, guided=False):
    """A read-only snapshot; never import a v4/v5 DB into a new controller."""
    if checked_file(dbpath, "file") is None:
        raise UninstallError("no readable controller SQLite evidence")
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
    raw, _ = command(["incus", "operation", "list", "local:", "--project",
                      project, "--format", "json"], timeout=25)
    if json.loads(raw) != []:
        raise UninstallError("native operations present or unreadable: reconcile before retirement")


def require_quiescent_scoped_native(project):
    """Incus retains terminal operations briefly after MCP has settled them.
    Check the exact native terminal state, never mistake retention for activity.
    Legacy removal deliberately keeps its original stricter empty-list rule.
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


def native_retirement_precheck(project, native, binary):
    """Use the exact installed Go reader; no independent rollback engine."""
    if native["status"] == "ABSENT":
        return None
    if not binary or file_fingerprint(BINARY) != binary:
        raise UninstallError("NATIVE_RETIREMENT_CHECK_UNVERIFIED: exact installed binary required")
    require_root_owned(BINARY)
    output, code = command([str(BINARY), "native-prepare-retirement-check",
                            "-project", project, "-native-prepare-sha256",
                            native["review_sha256"]], acceptable=(0, 1), timeout=90)
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
    observed = native_retirement_precheck(plan["project"], native, plan["binary"])
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
        raise UninstallError("unreviewed systemd drop-ins present")
    return pairs["FragmentPath"]


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
        raise UninstallError("effective retirement verifier unit cannot be verified")
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


def inspect(scope, project, box_uuids, native_sha, guided=False, scoped=False):
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
    snapshot = inspect_sqlite(STATE / "ledger.sqlite", guided=guided)
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
    native_precheck = native_retirement_precheck(project, native, binary) if scope == "full" else None
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


def apply_with_serve_lock(plan, preserved_archive=None):
    refuse_scoped_retirement()
    check_original_native_precondition(plan)
    # Stopping may legitimately checkpoint WAL; recheck native and SQLite
    # semantic blockers, not a fabricated claim that WAL bytes are immutable.
    after = inspect_sqlite(STATE / "ledger.sqlite", guided=plan.get("guided", False))
    if after["operations_sha256"] != plan["sqlite"]["operations_sha256"] or \
            after["boxes"] != plan["sqlite"]["boxes"] or after["blockers"]:
        raise UninstallError("SQLite ledger changed during stop; preserve evidence")
    if plan.get("guided") and inspect_exec_archive(plan["service_user_uid"]) != plan.get("exec_archive"):
        raise UninstallError("exec archive changed after approval")
    archive = preserved_archive or (archive_controller(plan) if plan.get("guided") else None)
    if plan["scope"] == "full":
        guided = plan.get("guided", False)
        if guided:
            require_quiescent_native(plan["project"])
        if native_instances(plan["project"], guided=guided) != plan["incus_instances"]:
            raise UninstallError("native inventory changed after approval")
        for box in plan["incus_instances"]:
            current = native_instances(plan["project"], guided=guided)
            exact = [x for x in current if x["uuid"] == box["uuid"]]
            if len(exact) != 1 or exact[0] != box:
                raise UninstallError("Incus target changed before deletion")
            if box["state"] == "Running" and plan.get("guided"):
                command(["incus", "stop", "local:" + box["name"], "--project",
                         plan["project"], "--timeout", "60"], timeout=90)
                stopped = [x for x in native_instances(plan["project"], guided=guided) if x["uuid"] == box["uuid"]]
                if stopped != [{**box, "state": "Stopped"}]:
                    raise UninstallError("graceful stop not independently verified")
            command(["incus", "delete", "local:" + box["name"], "--project", plan["project"]], timeout=90)
            if any(x["uuid"] == box["uuid"] for x in native_instances(plan["project"], guided=guided)):
                raise UninstallError("native deletion not verified")
        if guided:
            require_quiescent_native(plan["project"])
            if native_instances(plan["project"], guided=True):
                raise UninstallError("native Project no longer empty; preserve controller")
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
    return {"outcome": "COMPLETE", "scope": plan["scope"],
            "host_machine_id_sha256": plan["host_machine_id_sha256"],
            "approved_review_sha256": plan["review_sha256"],
            "retirement_archive": archive, "retained": plan["retained"]}


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


def scoped_units():
    result = {}
    for path in (SCOPED_GUARD_UNIT, SCOPED_LIFECYCLE_UNIT, SCOPED_INCUS_DROPIN):
        proof = file_fingerprint(path)
        if proof is None or proof["mode"] != 0o644 or path.lstat().st_nlink != 1:
            raise UninstallError("exact scoped unit/drop-in missing or changed")
        require_root_owned(path)
        if path != SCOPED_INCUS_DROPIN:
            p = scoped_properties(path.name, "FragmentPath", "DropInPaths", "ActiveState", "UnitFileState")
            if p["FragmentPath"] != str(path) or p["DropInPaths"] or p["UnitFileState"] != "static" or p["ActiveState"] not in ("active", "inactive", "failed"):
                raise UninstallError("unrecognized effective scoped service")
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
    plan = inspect("full", project, [], None, guided=True, scoped=True)
    if (plan["binary"] is None or plan["unit"] is None or plan["native_prepare"]["status"] != "ABSENT" or
            plan["retirement_verifier"] or plan["scriptlet_present"] or plan["installer_backup"] or os.path.lexists(BACKUP)):
        raise UninstallError("mixed legacy/scoped installation evidence; no retirement")
    for path in (BINARY, UNIT):
        if path.lstat().st_nlink != 1 or stat.S_IMODE(path.lstat().st_mode) != (0o755 if path == BINARY else 0o644):
            raise UninstallError("untrusted installed file mode or hardlink")
    # Go Status validates exact unit contents, manifest and original Root receipts.
    command([str(BINARY), "install-status", "-project", project], timeout=90)
    plan["scoped_units"] = scoped_units()
    plan["scoped_journal"] = scoped_tree(SCOPED_JOURNAL)
    plan["mode"] = SCOPED_MODE
    if plan["sqlite"]["boxes"] or plan["incus_instances"]:
        plan["blockers"].append("SCOPED_LIVE_BOXES: retire all Boxes, Peer and Forward through MCP first")
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
    if scoped_units() != plan["scoped_units"] or scoped_tree(SCOPED_JOURNAL) != plan["scoped_journal"]:
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
    return archive, cleanup


def scoped_incus_active():
    p = scoped_properties("incus.service", "ActiveState", "MainPID")
    if p["ActiveState"] != "active" or not p["MainPID"].isdigit() or int(p["MainPID"]) <= 1:
        raise UninstallError("Incus is not independently proven active")
    return p


def scoped_detach_guard(plan):
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


def main(argv=None):
    parser = argparse.ArgumentParser(prog="uninstall.sh", description="BoxWeave exact-owned uninstall; explicit full-only Scoped Peer v2 retirement after MCP cleanup")
    action = parser.add_mutually_exclusive_group()
    action.add_argument("--plan", action="store_true", help="read-only host inventory and review hashes")
    action.add_argument("--apply", action="store_true", help="apply one reviewed plan; requires two exact digests")
    action.add_argument("--full", action="store_true", help="guided full retirement with terminal review and backup")
    action.add_argument("--resume-plan", action="store_true", help="read-only review of a stopped, archived STARTED retirement")
    action.add_argument("--resume", action="store_true", help="explicit terminal-approved continuation of the exact archived STARTED retirement")
    parser.add_argument("--scoped", action="store_true", help="explicit full-only Scoped Peer v2 retirement; no native Box deletion")
    parser.add_argument("--retirement-archive", help="exact root-private original retirement archive for resume")
    parser.add_argument("--guided-full-plan", action="store_true",
                        help="with --plan --scope full: use the same guided read-only archive/inventory checks as --full")
    parser.add_argument("--scope", choices=SCOPES, help="service | controller | full (never shared Incus)")
    parser.add_argument("--project", help="exact restricted service Project (user-UID)")
    parser.add_argument("--box-uuid", action="append", default=[], help="full only: complete exact immutable UUID allowlist")
    parser.add_argument("--native-prepare-sha256", help="full only: exact original native journal review digest")
    parser.add_argument("--confirm-host-sha256", help="apply: exact machine SHA printed by plan")
    parser.add_argument("--confirm-review-sha256", help="apply: exact immutable inventory review SHA")
    args = parser.parse_args(argv)
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
