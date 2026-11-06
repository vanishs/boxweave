"""Trusted developer-only pre-publication Host installer bootstrap.

No candidate-supplied Python executes until ALL immutable release asset bytes
and the source-bound product ZIP verify against out-of-band SHA-256 values.
Only operates inside an explicitly approved disposable Box. Not shipped to
the normal BoxWeave product archive.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import time

from bwmini.online import RELEASE_FILES
from bwmini.distribution import verify as verify_zip, InstallError
from .engine import AcceptanceFault, now, private_mkdir, unique_file

HEX64 = re.compile(r"^[0-9a-f]{64}$")
HEX40 = re.compile(r"^[0-9a-f]{40}$")
VERSION = re.compile(r"^0\.[0-9]+\.[0-9]+$")
CONFIRM = "I_AUTHORIZE_EXACT_PRIVATE_INSTALL_IN_TEST_BOX"
SCOPE = "test_owned_private_candidate_install_only"
MAX_ZIP = 180_000_000
MAX_ASSET = 8_000_000


def require(condition, code):
    if not condition:
        raise AcceptanceFault(code)


def hashed(content):
    return hashlib.sha256(content).hexdigest()


def read_owned_file(path, limit):
    st = path.lstat()
    require(stat.S_ISREG(st.st_mode) and st.st_uid == 0 and
            st.st_nlink == 1 and st.st_mode & 0o022 == 0 and
            0 < st.st_size <= limit,
            "PRIVATE_CANDIDATE_ASSET_UNTRUSTED_" + path.name)
    data = path.read_bytes()
    require(len(data) == st.st_size, "PRIVATE_CANDIDATE_ASSET_CHANGED_" + path.name)
    return data


def exact_stage(stage, version, manifest_sha256, source):
    if not isinstance(manifest_sha256, str) or not HEX64.fullmatch(manifest_sha256):
        raise AcceptanceFault("CANDIDATE_MANIFEST_PIN_REQUIRED")
    if not isinstance(source, str) or not HEX40.fullmatch(source):
        raise AcceptanceFault("CANDIDATE_SOURCE_PIN_REQUIRED")
    if not isinstance(version, str) or not VERSION.fullmatch(version):
        raise AcceptanceFault("CANDIDATE_VERSION_INVALID")
    stage = Path(stage)
    require(stage.is_absolute(), "PRIVATE_CANDIDATE_ABSOLUTE_STAGE_REQUIRED")
    for folder in (stage, *stage.parents):
        st = folder.lstat()
        require(stat.S_ISDIR(st.st_mode) and st.st_uid == 0 and
                st.st_mode & 0o022 == 0,
                "PRIVATE_CANDIDATE_STAGE_DIRECTORY_UNTRUSTED")
    manifest_content = read_owned_file(stage / "release.json", MAX_ASSET)
    require(hashed(manifest_content) == manifest_sha256,
            "CANDIDATE_INDEPENDENT_MANIFEST_SHA_MISMATCH")
    try:
        m = json.loads(manifest_content)
    except ValueError:
        raise AcceptanceFault("CANDIDATE_MANIFEST_NOT_JSON") from None
    require(isinstance(m, dict) and m.get("format") == 1 and
            m.get("version") == version and m.get("source") == source and
            isinstance(m.get("files"), dict) and
            set(m["files"]) == set(RELEASE_FILES),
            "CANDIDATE_MANIFEST_SCOPE_OR_SOURCE_MISMATCH")
    require({entry.name for entry in stage.iterdir()} ==
            {"release.json", *RELEASE_FILES},
            "CANDIDATE_EXTRA_OR_MISSING_ASSETS")
    for name in RELEASE_FILES:
        sha = m["files"][name]
        require(isinstance(sha, str) and HEX64.fullmatch(sha),
                "CANDIDATE_ASSET_SHA_NOT_PINNED")
        data = read_owned_file(stage / name, MAX_ZIP if name == "boxweave.zip"
                               else MAX_ASSET)
        require(hashed(data) == sha, "CANDIDATE_ASSET_SHA_MISMATCH_" + name)
    try:
        zipinfo = verify_zip(stage / "boxweave.zip", m["files"]["boxweave.zip"])
    except (InstallError, ValueError, KeyError, OSError) as exc:
        raise AcceptanceFault("CANDIDATE_ARCHIVE_VERIFY_FAILED") from exc
    require(zipinfo.get("version") == version and
            zipinfo.get("source") == {"commit": source, "state": "clean"},
            "CANDIDATE_ARCHIVE_SOURCE_MISMATCH")
    return {"source_sha": source, "candidate_sha256": m["files"]["boxweave.zip"],
            "manifest_sha256": manifest_sha256, "version": version,
            "verified_files": len(RELEASE_FILES) + 1}


def check_owned_lab(grant):
    p = Path(grant)
    require(not p.is_symlink() and p.is_file(), "PRIVATE_INSTALL_GRANT_NOT_FOUND")
    st = p.stat()
    require(st.st_uid == os.geteuid() and st.st_mode & 0o077 == 0,
            "PRIVATE_INSTALL_GRANT_NOT_PRIVATE")
    try:
        value = json.loads(p.read_text())
    except ValueError:
        raise AcceptanceFault("PRIVATE_INSTALL_GRANT_INVALID") from None
    import socket
    require(isinstance(value, dict) and value.get("scope") == SCOPE and
            value.get("approved") is True and
            value.get("hostname") == socket.gethostname() and
            value.get("machine_id") == Path("/etc/machine-id").read_text().strip() and
            re.fullmatch(r"[a-f0-9-]{36}", str(value.get("outer_uuid", ""))) and
            re.fullmatch(r"\d{9,13}", str(value.get("outer_created_at", ""))) and
            isinstance(value.get("expires_epoch"), (int, float)) and
            0 < value["expires_epoch"] - time.time() < 86400,
            "PRIVATE_INSTALL_DISPOSABLE_IDENTITY_INVALID")
    return value


def run(stage, version, manifest_sha256, source, out, *, apply=False,
        grant=None, confirm=None):
    if os.geteuid() != 0:
        raise AcceptanceFault("PRIVATE_INSTALL_ROOT_OPERATOR_REQUIRED")
    record = exact_stage(stage, version, manifest_sha256, source)
    if apply:
        require(confirm == CONFIRM and grant is not None,
                "PRIVATE_INSTALL_EXPLICIT_CONFIRMATION_REQUIRED")
        identity = check_owned_lab(grant)
    else:
        identity = None
    dest = Path(out)
    require(not dest.exists() and not dest.is_symlink(),
            "PRIVATE_INSTALL_OUTPUT_EXISTS")
    private_mkdir(dest)
    unique_file(dest / "candidate-preflight.json", {
        "verified_at": now(), "status": "BYTES_VERIFIED_NO_CODE_RUN_YET",
        **record, "authorized_lab": bool(identity),
        "mode": "APPLY" if apply else "PREFLIGHT_ONLY"})
    if not apply:
        return {"state": "PREFLIGHT_ONLY", **record}
    unique_file(dest / "dispatch-intent.json", {
        "state": "MAY_HAVE_BEEN_DISPATCHED_ONCE",
        "outer_uuid": identity["outer_uuid"],
        "outer_created_at": identity["outer_created_at"],
        "candidate_sha256": record["candidate_sha256"],
        "recorded_at": now()})
    argv = ["/usr/bin/python3", "-B", str(Path(stage) / "online.py"),
            "install", "--stage", str(stage), "--version", version,
            "--private-manifest-sha256", manifest_sha256]
    try:
        proc = subprocess.run(argv, stdin=subprocess.DEVNULL,
                              capture_output=True, text=True, timeout=1800,
                              check=False)
    except subprocess.TimeoutExpired:
        unique_file(dest / "unknown.json", {
            "state": "UNKNOWN_NO_REPLAY",
            "reason": "HOST_INSTALL_DISPATCH_TIMEOUT"})
        raise AcceptanceFault("PRIVATE_INSTALL_UNKNOWN_NO_REPLAY") from None
    unique_file(dest / "install-result.json", {
        "exit_code": proc.returncode,
        "stdout_sha256": hashed(proc.stdout.encode()),
        "stderr_sha256": hashed(proc.stderr.encode()),
        "completed_at": now()})
    if proc.returncode != 0:
        raise AcceptanceFault("PRIVATE_INSTALL_NOT_SUCCESSFUL_INSPECT_ORIGINAL")
    return {"state": "INSTALL_RETURNED_ZERO_REQUIRES_INDEPENDENT_DOCTOR",
            **record}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--stage", required=True)
    p.add_argument("--version", required=True)
    p.add_argument("--manifest-sha256", required=True)
    p.add_argument("--source", required=True)
    p.add_argument("--out", required=True)
    p.add_argument("--apply", action="store_true")
    p.add_argument("--grant")
    p.add_argument("--confirm")
    args = p.parse_args(argv)
    try:
        result = run(args.stage, args.version, args.manifest_sha256,
                     args.source, args.out, apply=args.apply,
                     grant=args.grant, confirm=args.confirm)
        print(json.dumps(result, sort_keys=True))
        return 0
    except (AcceptanceFault, OSError, ValueError, KeyError) as exc:
        print(json.dumps({"state": "BLOCKED_OR_UNKNOWN", "reason": str(exc)}))
        return 2


if __name__ == "__main__":
    sys.exit(main())

