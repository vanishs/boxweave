#!/usr/bin/env python3
"""Offline Agent-only local worktree snapshot builder; no MCP or network access.

This source snapshot is NOT an installable BoxWeave release package. Review
sources for secrets before any transfer, even when filename filters pass.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import stat
import subprocess
import sys
import tempfile
import zipfile

MAX_FILES = 4096
MAX_FILE = 24 * 1024 * 1024
MAX_TOTAL = 90 * 1024 * 1024
DENY_NAMES = {".env", ".ssh", ".aws", ".kube", ".gnupg", "id_rsa",
              "id_ed25519", "credentials.json", "secrets.json", "jwt.secret",
              "authorized_keys", "known_hosts"}
DENY_SUFFIXES = (".pem", ".p12", ".pfx", ".key", ".sqlite", ".sqlite3",
                 ".db", ".token", ".secret")
METADATA = "boxweave-source-manifest.json"


class BundleError(Exception):
    pass


def require(ok, code):
    if not ok:
        raise BundleError(code)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def git(root, *args):
    result = subprocess.run(["git", "-C", str(root), *args],
                            capture_output=True, timeout=30, check=False)
    require(result.returncode == 0, "GIT_WORKTREE_UNAVAILABLE")
    return result.stdout


def source_root(repo):
    root = Path(git(repo, "rev-parse", "--show-toplevel").decode().strip()).resolve()
    require(root.is_dir() and (root / "VERSION").is_file(), "BOXWEAVE_WORKTREE_REQUIRED")
    commit = git(root, "rev-parse", "HEAD").decode().strip()
    require(re.fullmatch(r"[a-f0-9]{40}", commit) is not None, "GIT_COMMIT_INVALID")
    state = "modified" if git(root, "status", "--porcelain=v1", "--untracked-files=all") else "clean"
    return root, {"commit": commit, "state": state}


def safe_name(name):
    if not name or "\\" in name or "\x00" in name:
        return False
    parts = name.split("/")
    if PurePosixPath(name).is_absolute() or any(p in ("", ".", "..") for p in parts):
        return False
    for p in parts:
        lower = p.lower()
        if (lower in DENY_NAMES or lower.startswith(".env.")
                or lower.startswith("id_ed25519") or lower.startswith("id_rsa")
                or lower.endswith(DENY_SUFFIXES)):
            return False
    return True


def read_sources(root):
    candidates = set()
    raw = git(root, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
    for value in raw.split(b"\x00"):
        if not value:
            continue
        try:
            name = value.decode("utf-8")
        except UnicodeDecodeError:
            raise BundleError("NON_UTF8_SOURCE_PATH") from None
        require(safe_name(name) and name != METADATA, "UNSAFE_SOURCE_PATH")
        candidates.add(name)
    require(0 < len(candidates) <= MAX_FILES, "SOURCE_FILE_COUNT_INVALID")
    files, total = {}, 0
    for name in sorted(candidates):
        path = root / name
        if not os.path.lexists(path):
            continue  # A tracked deletion is absent from the current worktree.
        check = root
        for part in name.split("/"):
            check = check / part
            require(not check.is_symlink(), "SOURCE_SYMLINK_REFUSED")
        info = path.lstat()
        require(stat.S_ISREG(info.st_mode), "NON_REGULAR_SOURCE_REFUSED")
        require(info.st_size <= MAX_FILE, "SOURCE_FILE_TOO_LARGE")
        data = path.read_bytes()
        require(len(data) <= MAX_FILE, "SOURCE_FILE_TOO_LARGE")
        total += len(data)
        require(total <= MAX_TOTAL, "SOURCE_TOO_LARGE")
        files[name] = data
    require(bool(files), "EMPTY_SOURCE")
    return files


def verify(archive_path, expected):
    require(re.fullmatch(r"[a-f0-9]{64}", expected or "") is not None,
            "EXPECTED_SHA256_REQUIRED")
    archive_path = Path(archive_path)
    require(archive_path.is_file() and not archive_path.is_symlink(),
            "SOURCE_ARCHIVE_UNAVAILABLE")
    require(archive_path.stat().st_size <= MAX_TOTAL + 5 * 1024 * 1024,
            "SOURCE_ARCHIVE_TOO_LARGE")
    require(sha(archive_path.read_bytes()) == expected, "SOURCE_ARCHIVE_DIGEST_MISMATCH")
    with zipfile.ZipFile(archive_path) as archive:
        names = archive.namelist()
        require(len(names) == len(set(names)) and METADATA in names,
                "SOURCE_MANIFEST_MISSING_OR_DUPLICATE")
        manifest = json.loads(archive.read(METADATA))
        require(isinstance(manifest, dict) and set(manifest) == {"format", "source", "files"}
                and manifest["format"] == 1, "SOURCE_MANIFEST_INVALID")
        source = manifest["source"]
        require(isinstance(source, dict) and set(source) == {"commit", "state"}
                and re.fullmatch(r"[a-f0-9]{40}", source["commit"]) is not None
                and source["state"] in ("clean", "modified"), "SOURCE_IDENTITY_INVALID")
        records = manifest["files"]
        require(isinstance(records, dict) and 0 < len(records) <= MAX_FILES,
                "SOURCE_FILE_COUNT_INVALID")
        require(set(names) == {METADATA} | {"source/" + n for n in records},
                "SOURCE_MEMBERS_MISMATCH")
        total = 0
        for name, digest in records.items():
            require(isinstance(name, str) and safe_name(name) and name != METADATA
                    and isinstance(digest, str)
                    and re.fullmatch(r"[a-f0-9]{64}", digest) is not None,
                    "SOURCE_MEMBER_UNSAFE")
            entry = archive.getinfo("source/" + name)
            require(not entry.is_dir() and entry.file_size <= MAX_FILE,
                    "SOURCE_MEMBER_TOO_LARGE")
            content = archive.read(entry)
            total += len(content)
            require(total <= MAX_TOTAL, "SOURCE_TOO_LARGE")
            require(sha(content) == digest, "SOURCE_MEMBER_DIGEST_MISMATCH")
    return {"state": "verified", "sha256": expected, "source": source,
            "files": len(records), "bytes": total}


def pack(repo, output):
    root, source = source_root(repo)
    destination = Path(output).absolute()
    require(destination.parent.is_dir() and not destination.is_symlink()
            and not destination.exists(), "OUTPUT_UNAVAILABLE")
    require(not destination.is_relative_to(root), "OUTPUT_INSIDE_WORKTREE")
    files = read_sources(root)
    manifest = {"format": 1, "source": source,
                "files": {name: sha(payload) for name, payload in files.items()}}
    temp = None
    try:
        with tempfile.NamedTemporaryFile(prefix=".boxweave-source-", suffix=".zip",
                                         dir=destination.parent, delete=False) as handle:
            temp = Path(handle.name)
        with zipfile.ZipFile(temp, "w", compression=zipfile.ZIP_DEFLATED,
                             compresslevel=6, allowZip64=False) as bundle:
            bundle.writestr(METADATA, json.dumps(manifest, sort_keys=True) + "\n")
            for name, payload in files.items():
                bundle.writestr("source/" + name, payload)
        digest = sha(temp.read_bytes())
        result = verify(temp, digest)
        os.chmod(temp, 0o600)
        os.link(temp, destination)  # Exclusive publication; never overwrite another run.
        return {**result, "state": "ready", "archive": str(destination)}
    finally:
        if temp is not None:
            temp.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    modes = parser.add_subparsers(dest="mode", required=True)
    p = modes.add_parser("pack")
    p.add_argument("--repo", default=".")
    p.add_argument("--output", required=True)
    v = modes.add_parser("verify")
    v.add_argument("--archive", required=True)
    v.add_argument("--sha256", required=True)
    args = parser.parse_args()
    try:
        result = pack(args.repo, args.output) if args.mode == "pack" else verify(args.archive, args.sha256)
        print(json.dumps(result, sort_keys=True))
        return 0
    except (BundleError, OSError, UnicodeError, ValueError, KeyError, TypeError,
            subprocess.TimeoutExpired, zipfile.BadZipFile, zipfile.LargeZipFile):
        print(json.dumps({"state": "failed", "code": "SOURCE_BUNDLE_REFUSED"}), file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
