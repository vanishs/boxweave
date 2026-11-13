#!/usr/bin/env python3
"""Assemble exact-source v1 Linux two-architecture release metadata.

No release/publication operation occurs in this script. All inputs must be
the already tested artifacts from a single Actions commit.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import struct

TAG = re.compile(r"^v1\.[0-9]+\.[0-9]+(?:-rc\.[1-9][0-9]*)?$")
SHA = re.compile(r"^[0-9a-f]{40}$")
MACHINES = {"amd64": 62, "arm64": 183}


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def validate_elf(data, architecture):
    if len(data) < 64 or data[:6] != b"\x7fELF\x02\x01":
        raise ValueError("expected 64-bit little-endian ELF binary")
    machine = struct.unpack_from("<H", data, 18)[0]
    if machine != MACHINES[architecture]:
        raise ValueError("ELF machine mismatch for " + architecture)


def assemble(directory, tag, source, installer):
    if not TAG.fullmatch(tag) or not SHA.fullmatch(source):
        raise ValueError("invalid v1 release tag or source SHA")
    if not installer.is_file():
        raise ValueError("external install.sh must be included")
    directory.mkdir(parents=True, exist_ok=True)
    version = tag[1:]
    assets = {}
    files = {}
    for arch in MACHINES:
        name = "boxweave-" + tag + "-linux-" + arch
        path = directory / name
        data = path.read_bytes()
        if not data or len(data) > 100 * 1024 * 1024:
            raise ValueError("invalid binary size")
        validate_elf(data, arch)
        assets[name] = {"sha256": sha256(data), "size": len(data), "goarch": arch}
        files[name] = data
    files["install.sh"] = installer.read_bytes()
    go_mod = Path("go.mod").read_text(encoding="utf-8")
    def pinned(expression):
        match = re.search(expression, go_mod, re.MULTILINE)
        if match is None:
            raise ValueError("release source must pin required Go/SDK dependency: " + expression)
        return match.group(1)

    dependencies = {
        "go_toolchain": pinned(r"^go ([0-9]+\\.[0-9]+\\.[0-9]+)$"),
        "mcp_go_sdk": pinned(r"^[ \\t]*github\\.com/modelcontextprotocol/go-sdk (v[0-9.]+)$"),
        "incus_go_sdk": pinned(r"^[ \\t]*github\\.com/lxc/incus/v7 (v[0-9.]+)$"),
        "sqlite_cgo": pinned(r"^[ \\t]*github\\.com/mattn/go-sqlite3 (v[0-9.]+)$"),
    }
    manifest = {
        "format": 1,
        "product": "boxweave-v1",
        "version": version,
        "tag": tag,
        "source": source,
        "platform": "linux",
        "cgo_enabled": True,
        "dependencies": dependencies,
        "native_incus_runtime": "must be independently checked on the target host",
        "assets": assets,
    }
    manifest_bytes = (json.dumps(manifest, sort_keys=True, indent=2) + "\n").encode()
    files["release.json"] = manifest_bytes
    sums = "".join(sha256(files[name]) + "  " + name + "\n" for name in sorted(files)).encode()
    (directory / "release.json").write_bytes(manifest_bytes)
    (directory / "SHA256SUMS").write_bytes(sums)
    (directory / "install.sh").write_bytes(files["install.sh"])
    return manifest


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--directory", type=Path, required=True)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--source", required=True)
    parser.add_argument("--installer", type=Path, default=Path("install.sh"))
    args = parser.parse_args()
    manifest = assemble(args.directory, args.tag, args.source, args.installer)
    print(json.dumps({"tag": manifest["tag"], "source": manifest["source"], "assets": manifest["assets"]}, sort_keys=True))


if __name__ == "__main__":
    main()
