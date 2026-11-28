#!/usr/bin/env python3
"""Verify stable release asset identity, inventory and checksums."""
from __future__ import annotations

import hashlib
import json
from pathlib import Path
import re
import stat

from v1_release_manifest import MACHINES, validate_elf

REPO = "vanishs/boxweave"
TAG = re.compile(r"v1\.[0-9]+\.[0-9]+")
SHA = re.compile(r"[0-9a-f]{40}")
MAX_JSON = 256 * 1024


def require(condition, reason):
    if not condition:
        raise ValueError(reason)


def strict_object(pairs):
    out = {}
    for k, v in pairs:
        require(k not in out, "duplicate JSON key")
        out[k] = v
    return out


def obj(raw):
    require(len(raw) <= MAX_JSON, "oversized JSON")
    v = json.loads(raw, object_pairs_hook=strict_object,
                   parse_constant=lambda _: (_ for _ in ()).throw(ValueError("invalid JSON constant")))
    require(type(v) is dict, "object required")
    return v


def read_regular(path, limit):
    info = path.lstat()
    require(stat.S_ISREG(info.st_mode) and info.st_nlink == 1, "non-regular asset")
    require(0 < info.st_size <= limit, "asset size outside bound")
    return path.read_bytes()


def sha(data):
    return hashlib.sha256(data).hexdigest()


def exact_fields(value, fields, name):
    require(type(value) is dict and set(value) == set(fields), "invalid " + name + " fields")


def verify_assets(directory, tag, source, root):
    require(TAG.fullmatch(tag) is not None and SHA.fullmatch(source) is not None, "not a stable v1 identity")
    require(tag == "v" + (root / "VERSION").read_text().strip(), "VERSION mismatch")
    names = {"boxweave-" + tag + "-linux-" + arch: arch for arch in MACHINES}
    expected = set(names) | {"release.json", "SHA256SUMS", "install.sh", "uninstall.sh"}
    require(directory.is_dir() and not directory.is_symlink() and
            {p.name for p in directory.iterdir()} == expected, "unexpected or missing release asset")
    data = {n: read_regular(directory / n, 100 * 1024 * 1024 if n in names else MAX_JSON)
            for n in expected}
    m = obj(data["release.json"])
    exact_fields(m, ("format", "product", "version", "tag", "source", "platform", "cgo_enabled",
                     "dependencies", "native_incus_runtime", "assets"), "manifest")
    require(type(m["format"]) is int and m["format"] == 1 and m["product"] == "boxweave-v1"
            and m["version"] == tag[1:] and m["tag"] == tag and m["source"] == source
            and m["platform"] == "linux" and m["cgo_enabled"] is True, "manifest identity mismatch")
    require(type(m["assets"]) is dict and set(m["assets"]) == set(names), "binary inventory mismatch")
    for name, arch in names.items():
        payload = data[name]
        validate_elf(payload, arch)
        require(m["assets"][name] == {"sha256": sha(payload), "size": len(payload), "goarch": arch},
                "manifest binary digest mismatch")
    require(data["install.sh"] == read_regular(root / "install.sh", MAX_JSON), "installer source mismatch")
    core = read_regular(root / "scripts/uninstall_core.py", MAX_JSON)
    expected_uninstaller = (
        b'#!/bin/sh\n'
        b'set -eu\n'
        b"exec python3 -c 'import hashlib, sys\n"
        b'source = sys.stdin.buffer.read()\n'
        b'if not source.endswith(b"\\n"):\n'
        b'    raise SystemExit("BoxWeave uninstaller: incomplete embedded core")\n'
        b'source = source[:-1]\n'
        b'__file__ = "<boxweave-uninstall>"\n'
        b'_BOXWEAVE_EMBEDDED_CORE_SHA256 = hashlib.sha256(source).hexdigest()\n'
        b'sys.argv[0] = "-"\n'
        b'exec(compile(source, __file__, "exec"))\' "$@" <<\'PY_BOXWEAVE_UNINSTALL\'\n'
        + core + b'\nPY_BOXWEAVE_UNINSTALL\n'
    )
    require(data["uninstall.sh"] == expected_uninstaller, "uninstaller source mismatch")
    go_mod = (root / "go.mod").read_text(encoding="utf-8")
    dep_patterns = {"go_toolchain": r"^go ([0-9]+\.[0-9]+\.[0-9]+)$",
                    "mcp_go_sdk": r"^[ \t]*github\.com/modelcontextprotocol/go-sdk (v[0-9.]+)$",
                    "incus_go_sdk": r"^[ \t]*github\.com/lxc/incus/v7 (v[0-9.]+)$",
                    "sqlite_cgo": r"^[ \t]*github\.com/mattn/go-sqlite3 (v[0-9.]+)$"}
    deps = {}
    for name, pattern in dep_patterns.items():
        match = re.search(pattern, go_mod, re.MULTILINE)
        require(match is not None, "missing pinned source dependency")
        deps[name] = match.group(1)
    require(m["dependencies"] == deps, "dependency provenance mismatch")
    sums = "".join(sha(data[n]) + "  " + n + "\n" for n in sorted(expected - {"SHA256SUMS"})).encode()
    require(data["SHA256SUMS"] == sums, "checksum inventory/content mismatch")
    return sha(data["release.json"]), sha(sums)
