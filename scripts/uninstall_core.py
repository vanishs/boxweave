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
VALID_NAME = re.compile(r"[a-z][a-z0-9-]{0,62}\Z")
VALID_UUID = re.compile(r"[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}\Z")
VALID_SHA = re.compile(r"[0-9a-f]{64}\Z")
TERMINAL = {"SUCCEEDED", "FAILED", "CANCELLED"}
SERVE_LOCK_NAME = "ledger.sqlite.serve.lock"
EXEC_ARCHIVE_NAME = "ledger.sqlite.exec-archive"
EXEC_ARCHIVE_RE = re.compile(r"[a-f0-9]{32}\.(stdout|stderr)\Z")
STATE_FILES = {"ledger.sqlite", "ledger.sqlite-wal", "ledger.sqlite-shm",
               "install-manifest.json", SERVE_LOCK_NAME}
SCOPES = ("service", "controller", "full")


class UninstallError(Exception):
    pass


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
        for op, layer, box, state, native in conn.execute(
                "SELECT id,layer,box_uuid,state,native_operation_id FROM operations ORDER BY id"):
            operation_hash.update(canonical([op, layer, box, state, native]) + b"\n")
            operation_count += 1
            if state not in TERMINAL:
                # Keep UNKNOWN unchanged. Only retire a reviewed historical
                # Exec with exact successful Delete lineage.
                if guided and version[0] in (5, 6) and state == "UNKNOWN":
                    witness = conn.execute("""
                        SELECT o.id, b.uuid, d.id, r.seq, r.evidence_digest
                        FROM operations o
                        JOIN owned_boxes b ON b.uuid=o.box_uuid AND b.incarnation=o.incarnation
                           AND b.project=o.layer AND b.state='DELETED' AND b.deleted_at>0
                        JOIN operations d ON d.id=b.deleted_by_operation_id AND d.action='box_delete'
                           AND d.state='SUCCEEDED' AND d.layer=b.project
                           AND d.box_uuid=b.uuid AND d.incarnation=b.incarnation
                        JOIN recovery_audit r ON r.operation_id=o.id AND r.decision='RELEASED'
                        WHERE o.id=? AND o.action='box_exec' AND r.at<=b.deleted_at
                        ORDER BY r.seq
                    """, (op,)).fetchall()
                    if len(witness) == 1 and VALID_SHA.fullmatch(witness[0][4]):
                        historical.append(list(witness[0]))
                        operation_hash.update(canonical(witness[0]) + b"\n")
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


def inspect(scope, project, box_uuids, native_sha, guided=False):
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
    if scope != "service" and BACKUP.exists():
        blockers.append("INSTALLER_BACKUP_REQUIRES_SEPARATE_RETENTION_REVIEW")
    inputs = {
        "format": 1, "scope": scope, "guided": guided, "project": project, "host_machine_id_sha256": host,
        "service_user_uid": owner.pw_uid, "service_state": service,
        "unit": unit, "binary": binary, "state_files": files, "exec_archive": exec_archive,
        "retirement_verifier": retirement,
        "effective_unit": fragment, "uninstall_audit": audit,
        "sqlite": snapshot, "native_prepare": native, "scriptlet_present": scriptlet is not None,
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
    # At this point the operator has already approved both exact digests;
    # no automatic retries or implicit "yes" on modified host state.
    if plan["service_state"] == "active":
        command(["systemctl", "stop", "boxweave.service"])
    if systemd_status() == "active":
        raise UninstallError("controller did not stop")
    require_retirement_stopped(plan)
    with hold_stopped_serve_lock(plan):
        return apply_with_serve_lock(plan)


def archive_controller(plan):
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
        out.write(canonical(plan) + b"\n")
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


def apply_with_serve_lock(plan):
    # Stopping may legitimately checkpoint WAL; recheck native and SQLite
    # semantic blockers, not a fabricated claim that WAL bytes are immutable.
    after = inspect_sqlite(STATE / "ledger.sqlite", guided=plan.get("guided", False))
    if after["operations_sha256"] != plan["sqlite"]["operations_sha256"] or \
            after["boxes"] != plan["sqlite"]["boxes"] or after["blockers"]:
        raise UninstallError("SQLite ledger changed during stop; preserve evidence")
    if plan.get("guided") and inspect_exec_archive(plan["service_user_uid"]) != plan.get("exec_archive"):
        raise UninstallError("exec archive changed after approval")
    archive = archive_controller(plan) if plan.get("guided") else None
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
    return {"outcome": "COMPLETE", "scope": plan["scope"],
            "host_machine_id_sha256": plan["host_machine_id_sha256"],
            "approved_review_sha256": plan["review_sha256"],
            "retirement_archive": archive, "retained": plan["retained"]}


def main(argv=None):
    parser = argparse.ArgumentParser(prog="uninstall.sh", description="BoxWeave exact-owned host uninstall (no implicit deletion)")
    action = parser.add_mutually_exclusive_group()
    action.add_argument("--plan", action="store_true", help="read-only host inventory and review hashes")
    action.add_argument("--apply", action="store_true", help="apply one reviewed plan; requires two exact digests")
    action.add_argument("--full", action="store_true", help="guided full retirement with terminal review and backup")
    parser.add_argument("--scope", choices=SCOPES, help="service | controller | full (never shared Incus)")
    parser.add_argument("--project", help="exact restricted service Project (user-UID)")
    parser.add_argument("--box-uuid", action="append", default=[], help="full only: complete exact immutable UUID allowlist")
    parser.add_argument("--native-prepare-sha256", help="full only: exact original native journal review digest")
    parser.add_argument("--confirm-host-sha256", help="apply: exact machine SHA printed by plan")
    parser.add_argument("--confirm-review-sha256", help="apply: exact immutable inventory review SHA")
    args = parser.parse_args(argv)
    if not args.plan and not args.apply and not args.full:
        parser.print_help()
        return 0 if not any((args.scope, args.project, args.box_uuid)) else 2
    if args.full:
        if args.scope or args.confirm_host_sha256 or args.confirm_review_sha256:
            parser.error("--full uses terminal approval, not explicit scope/digests")
        args.scope = "full"
        args.project = args.project or "user-" + str(pwd.getpwnam("boxweave").pw_uid)
    if not args.scope or not args.project:
        parser.error("--scope and --project required for plan/apply")
    if os.geteuid() != 0 or sys.platform != "linux":
        raise UninstallError("root on the reviewed Linux host required for safe ledger/Incus inspection")
    if os.path.lexists("/var/lib/boxweave-scoped-peer-v2"):
        raise UninstallError("scoped v2 retirement requires separate reviewed implementation; preserve native guard and journal")
    plan = inspect(args.scope, args.project, args.box_uuid, args.native_prepare_sha256, guided=args.full)
    if args.plan:
        print(json.dumps(plan, indent=2, sort_keys=True))
        return 0 if plan["ready"] else 2
    if args.full:
        print(json.dumps(plan, indent=2, sort_keys=True), flush=True)
        if not plan["ready"]:
            raise UninstallError("guided retirement blocked by current evidence")
        with open("/dev/tty", "r") as inp, open("/dev/tty", "w") as out:
            out.write("Type REMOVE " + args.project + " to approve deleting all listed Boxes and retiring this controller: ")
            out.flush()
            if inp.readline().strip() != "REMOVE " + args.project:
                raise UninstallError("retirement not approved")
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
    audited_write({"status": "STARTED", "scope": plan["scope"],
                   "host_sha256": plan["host_machine_id_sha256"],
                   "review_sha256": plan["review_sha256"],
                   "actions": plan["actions"],
                   "historical_released_exec": plan["sqlite"]["historical_released_exec"]})
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
