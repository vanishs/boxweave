"""On-demand, read-only Containarium source references pinned to observed runtime.

Nothing here executes upstream instructions, installs binaries, or changes a Provider.
All caches reside in a private developer workspace, never in customer archives.
"""
from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import shutil
import stat
import subprocess
import sys
from urllib.request import Request, urlopen

from .engine import AcceptanceFault, canonical, capture, now, private_mkdir

REPO = "FootprintAI/Containarium"
API = "https://api.github.com/repos/" + REPO + "/"
VERSION = re.compile(r"^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")
SHA = re.compile(r"^[0-9a-f]{40}$")
EXACT = frozenset(("docs/DOCMAP.md", "docs/PROOF-PACK.md",
                   "docs/security/CLAIMS.md"))
PREFIXES = ("docs/testing/", "proto/containarium/v1/", "internal/mcp/",
            "internal/server/", "internal/cmd/", ".github/workflows/")
MAX_REFERENCE_BYTES = 2_000_000


def semver(value):
    m = VERSION.fullmatch(str(value))
    if not m:
        raise AcceptanceFault("UPSTREAM_VERSION_INVALID")
    return ".".join(m.groups())


def safe_path(path):
    if not isinstance(path, str) or len(path) > 250 or "\\" in path:
        raise AcceptanceFault("UPSTREAM_REFERENCE_PATH_INVALID")
    p = PurePosixPath(path)
    if (p.is_absolute() or any(x in (".", "..", "") for x in path.split("/"))
            or str(p) != path or not (path in EXACT or path.startswith(PREFIXES))):
        raise AcceptanceFault("UPSTREAM_REFERENCE_NOT_ALLOWLISTED")
    return path


def get_json(endpoint):
    """Only fixed GitHub HTTPS API routes; bounded JSON, no alternate hosts."""
    if not endpoint.startswith(API) or any(x in endpoint for x in ("#", "\n", "\r", "@")):
        raise AcceptanceFault("UNTRUSTED_UPSTREAM_ORIGIN")
    with urlopen(Request(endpoint, headers={
            "Accept": "application/vnd.github+json", "User-Agent": "BoxWeave-reacceptance/1"
    }), timeout=12) as response:
        raw = response.read(MAX_REFERENCE_BYTES * 2 + 1)
    if len(raw) > MAX_REFERENCE_BYTES * 2:
        raise AcceptanceFault("UPSTREAM_HTTP_RESPONSE_TOO_LARGE")
    return json.loads(raw)


def release_commit(version, fetch=get_json):
    tag = "v" + semver(version)
    item = fetch(API + "git/ref/tags/" + tag)
    obj = item.get("object", {})
    for _ in range(3):
        sha, kind = obj.get("sha"), obj.get("type")
        if not isinstance(sha, str) or not SHA.fullmatch(sha):
            raise AcceptanceFault("UPSTREAM_TAG_SHA_INVALID")
        if kind == "commit":
            return sha
        if kind != "tag":
            raise AcceptanceFault("UPSTREAM_TAG_TARGET_UNSUPPORTED")
        obj = fetch(API + "git/tags/" + sha).get("object", {})
    raise AcceptanceFault("UPSTREAM_TAG_CHAIN_TOO_LONG")


def binary_observation(name, binary):
    path = shutil.which(binary)
    if not path:
        return {"status": "UNAVAILABLE", "reason": "BINARY_NOT_FOUND", "binary": binary}
    path = Path(path).resolve()
    if not path.is_file():
        return {"status": "UNAVAILABLE", "reason": "BINARY_NOT_REGULAR"}
    try:
        if path.stat().st_size > 100_000_000:
            raise AcceptanceFault("VERSION_BINARY_TOO_LARGE")
        checksum = hashlib.sha256(path.read_bytes()).hexdigest()
    except OSError:
        return {"status": "UNAVAILABLE", "reason": "BINARY_UNREADABLE"}
    # Both invocations are read-only version commands. No arbitrary argv is accepted.
    records = []
    for arg in ("version", "--version"):
        result = capture([str(path), arg], timeout=8)
        records.append({"arg": arg, "exit": result["exit_code"],
                        "execution": result["execution"]})
        if result["execution"] != "completed" or result["exit_code"] != 0:
            continue
        m = re.search(r"(?<![A-Za-z0-9])v?(\d+\.\d+\.\d+)(?![A-Za-z0-9])",
                      (result["stdout"] + " " + result["stderr"])[:8192])
        if m:
            return {"status": "OBSERVED", "version": semver(m.group(1)),
                    "binary_path": str(path), "binary_sha256": checksum,
                    "argv": [str(path), arg], "observed_at": now(), "attempts": records}
    return {"status": "UNAVAILABLE", "reason": "VERSION_NOT_EXPOSED",
            "binary_path": str(path), "binary_sha256": checksum, "attempts": records}


def observe_installed():
    return {"schema_version": 1, "captured_at": now(),
            "components": {n: binary_observation(n, cmd) for n, cmd in
                           (("daemon", "containariumd"), ("cli", "containarium"),
                            ("mcp", "mcp-server"))},
            "rest": {"status": "NOT_OBSERVED", "reason": "AUTHENTICATED_ROUTE_PROBE_REQUIRED"}}


def cache_blob(root, sha, path, data):
    path = safe_path(path)
    if not SHA.fullmatch(sha):
        raise AcceptanceFault("UPSTREAM_COMMIT_INVALID")
    if not isinstance(data, bytes) or len(data) > MAX_REFERENCE_BYTES:
        raise AcceptanceFault("UPSTREAM_REFERENCE_TOO_LARGE")
    base = private_mkdir(Path(root) / sha)
    destination = base / path
    private_mkdir(destination.parent)
    if destination.is_symlink():
        raise AcceptanceFault("UPSTREAM_CACHE_SYMLINK")
    h = hashlib.sha256(data).hexdigest()
    if destination.exists():
        if hashlib.sha256(destination.read_bytes()).hexdigest() != h:
            raise AcceptanceFault("UPSTREAM_CACHE_TAMPERED")
    else:
        fd = os.open(destination, os.O_CREAT | os.O_EXCL | os.O_WRONLY |
                     getattr(os, "O_NOFOLLOW", 0), 0o600)
        with os.fdopen(fd, "wb") as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
    return {"path": path, "sha256": h, "bytes": len(data)}


def cached_blob(root, sha, path, checksum):
    path = safe_path(path)
    if not SHA.fullmatch(sha) or not re.fullmatch(r"[a-f0-9]{64}", checksum):
        raise AcceptanceFault("UPSTREAM_CACHE_METADATA_INVALID")
    p = Path(root) / sha / path
    if p.is_symlink() or not p.is_file() or p.stat().st_size > MAX_REFERENCE_BYTES:
        raise AcceptanceFault("UPSTREAM_OFFLINE_REFERENCE_UNAVAILABLE")
    data = p.read_bytes()
    if hashlib.sha256(data).hexdigest() != checksum:
        raise AcceptanceFault("UPSTREAM_CACHE_TAMPERED")
    return data


def retrieve_reference(sha, path, fetch=get_json):
    safe_path(path)
    if not SHA.fullmatch(sha):
        raise AcceptanceFault("UPSTREAM_COMMIT_INVALID")
    # Immutable git-tree commit binding: never use main, latest or a floating tag here.
    endpoint = API + "contents/" + path + "?ref=" + sha
    obj = fetch(endpoint)
    if obj.get("type") != "file" or obj.get("encoding") != "base64":
        raise AcceptanceFault("UPSTREAM_FILE_FORMAT_UNSUPPORTED")
    encoded = obj.get("content")
    if not isinstance(encoded, str) or len(encoded) > MAX_REFERENCE_BYTES * 2:
        raise AcceptanceFault("UPSTREAM_FILE_TOO_LARGE")
    try:
        data = base64.b64decode(encoded, validate=False)
    except (ValueError, TypeError):
        raise AcceptanceFault("UPSTREAM_FILE_BASE64_INVALID") from None
    if len(data) > MAX_REFERENCE_BYTES:
        raise AcceptanceFault("UPSTREAM_FILE_TOO_LARGE")
    # Git Contents SHA is SHA1(blob header + bytes), independent of JSON metadata.
    expected = hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()
    if obj.get("sha") != expected:
        raise AcceptanceFault("UPSTREAM_CONTENTS_BLOB_SHA_MISMATCH")
    return data


def resolve(observations, cache, paths=(), *, offline=False, fetch=get_json):
    """Pins each observed component independently; never maps baseline to runtime."""
    if not isinstance(observations, dict) or observations.get("schema_version") != 1:
        raise AcceptanceFault("UPSTREAM_OBSERVATION_INVALID")
    components = observations.get("components")
    if not isinstance(components, dict) or not components:
        raise AcceptanceFault("UPSTREAM_COMPONENTS_MISSING")
    paths = [safe_path(p) for p in paths]
    base = private_mkdir(cache)
    manifests = {}
    for name, entry in components.items():
        if not re.fullmatch(r"[a-z][a-z0-9_-]{1,30}", name) or not isinstance(entry, dict):
            raise AcceptanceFault("UPSTREAM_COMPONENT_INVALID")
        if entry.get("status") != "OBSERVED":
            manifests[name] = {"status": "UNAVAILABLE", "reason": entry.get("reason", "NOT_OBSERVED")}
            continue
        version = semver(entry.get("version"))
        pin = base / ("v" + version + ".json")
        if pin.is_symlink():
            raise AcceptanceFault("UPSTREAM_CACHE_SYMLINK")
        seal = base / ("v" + version + ".json.sha256")
        if seal.is_symlink():
            raise AcceptanceFault("UPSTREAM_CACHE_PIN_TAMPERED")
        if pin.exists() != seal.exists():
            raise AcceptanceFault("UPSTREAM_CACHE_PIN_TAMPERED")
        if pin.exists():
            contents = pin.read_bytes()
            if seal.read_text().strip() != hashlib.sha256(contents).hexdigest():
                raise AcceptanceFault("UPSTREAM_CACHE_PIN_TAMPERED")
            metadata = json.loads(contents)
            sha = metadata.get("commit")
            if (metadata.get("tag") != "v" + version or not isinstance(sha, str)
                    or not SHA.fullmatch(sha)):
                raise AcceptanceFault("UPSTREAM_CACHE_PIN_TAMPERED")
        elif offline:
            manifests[name] = {"status": "UNRESOLVED_UPSTREAM_REF", "version": version,
                               "reason": "OFFLINE_PIN_UNAVAILABLE"}
            continue
        else:
            sha = release_commit(version, fetch)
            bytes_pinned = canonical({"tag": "v" + version, "commit": sha}) + b"\n"
            pin.write_bytes(bytes_pinned)
            pin.chmod(0o600)
            seal.write_text(hashlib.sha256(bytes_pinned).hexdigest() + "\n")
            seal.chmod(0o600)
        cached = {}
        for path in paths:
            audit = base / sha / (path + ".sha256")
            if audit.is_symlink():
                raise AcceptanceFault("UPSTREAM_CACHE_SYMLINK")
            if audit.exists():
                checksum = audit.read_text().strip()
                data = cached_blob(base, sha, path, checksum)
                cached[path] = {"sha256": hashlib.sha256(data).hexdigest(),
                                "bytes": len(data), "provenance": "verified_cache"}
            elif offline:
                cached[path] = {"status": "REFERENCE_UNAVAILABLE_OFFLINE"}
            else:
                data = retrieve_reference(sha, path, fetch)
                cached[path] = cache_blob(base, sha, path, data) | {"provenance": "commit_bound_download"}
                audit.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
                audit.write_text(cached[path]["sha256"] + "\n")
                audit.chmod(0o600)
        manifests[name] = {"status": "PINNED", "version": version, "tag": "v" + version,
                           "commit": sha, "reference_paths": cached,
                           "binary_sha256": entry.get("binary_sha256"),
                           "binary_path": entry.get("binary_path")}
    found = {x["version"] for x in manifests.values() if x["status"] == "PINNED"}
    return {"schema_version": 1, "created_at": now(), "upstream_repo": REPO,
            "state": "PINNED" if found else "NO_OBSERVED_VERSION_PIN",
            "split_runtime": len(found) > 1, "components": manifests,
            "contracts_are_reference_not_runtime_proof": True}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    sub = p.add_subparsers(dest="command", required=True)
    ob = sub.add_parser("observe")
    ob.add_argument("--out", required=True)
    rs = sub.add_parser("resolve")
    rs.add_argument("--observations", required=True)
    rs.add_argument("--cache", required=True)
    rs.add_argument("--path", action="append", default=[])
    rs.add_argument("--offline", action="store_true")
    args = p.parse_args(argv)
    try:
        if args.command == "observe":
            value = observe_installed()
        else:
            value = resolve(json.loads(Path(args.observations).read_text()),
                            args.cache, args.path, offline=args.offline)
        path = Path(args.out) if args.command == "observe" else None
        if path:
            path.write_bytes(canonical(value) + b"\n")
        print(json.dumps(value, indent=2, sort_keys=True))
        return 0
    except (AcceptanceFault, OSError, ValueError, KeyError) as exc:
        print(json.dumps({"status": "FAIL_CLOSED", "code": str(exc)}))
        return 2


if __name__ == "__main__":
    sys.exit(main())
