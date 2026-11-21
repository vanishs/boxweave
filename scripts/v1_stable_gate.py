#!/usr/bin/env python3
"""Read-only stable-release authorization verifier. No network or publication.

The evidence is signed by separately pinned owner and verifier keys, not by
the build job or the candidate artifact. PASS means reviewed supported scope,
not automatic validation of physical/client evidence by this program.
"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
import stat
import subprocess
import tempfile

from v1_release_manifest import MACHINES, validate_elf

REPO = "vanishs/boxweave"
WORKFLOW = ".github/workflows/v1-stable-candidate.yml"
GATES = tuple(f"S5{phase}{i}" for phase in "AB" for i in range(1, 7))
JOBS = {"stable-candidate-source", "stable-candidate-build-amd64",
        "stable-candidate-build-arm64", "stable-candidate-package"}
TAG = re.compile(r"v1\.[0-9]+\.[0-9]+")
SHA = re.compile(r"[0-9a-f]{40}")
DIGEST = re.compile(r"[0-9a-f]{64}")
MAX_JSON = 256 * 1024
ED25519_SPKI = bytes.fromhex("302a300506032b6570032100")


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


def valid_id(v):
    return type(v) is int and 0 < v < 2**63


def candidate(run, jobs, source):
    """Check actual GitHub Actions REST metadata, not signoff-supplied booleans."""
    require(SHA.fullmatch(source) is not None, "invalid source")
    require(run.get("path") in (WORKFLOW, WORKFLOW + "@refs/heads/v1"), "wrong candidate workflow")
    require(run.get("event") == "workflow_dispatch" and run.get("head_branch") == "v1",
            "candidate was not dispatched from v1")
    require(run.get("head_sha") == source and run.get("status") == "completed"
            and run.get("conclusion") == "success", "candidate source/status mismatch")
    for field in ("repository", "head_repository"):
        require(isinstance(run.get(field), dict) and run[field].get("full_name") == REPO,
                "foreign candidate repository")
    identity = run.get("id"), run.get("run_attempt")
    require(all(valid_id(x) for x in identity), "invalid run identity")
    rows = jobs.get("jobs")
    require(type(rows) is list and jobs.get("total_count") == len(rows) == len(JOBS),
            "incomplete candidate job inventory")
    require({j.get("name") for j in rows} == JOBS, "unexpected or missing candidate jobs")
    for j in rows:
        require(j.get("run_id") == identity[0] and j.get("head_sha") == source
                and j.get("status") == "completed" and j.get("conclusion") == "success",
                "candidate job not successful at exact source")
        require(j.get("run_attempt", identity[1]) == identity[1], "candidate job attempt mismatch")
    return identity


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
    require(data["uninstall.sh"] == b"#!/bin/sh\nset -eu\nexec python3 - \"$@\" <<'PY_BOXWEAVE_UNINSTALL'\n"
            + core + b"\nPY_BOXWEAVE_UNINSTALL\n", "uninstaller source mismatch")
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


def validate_signoff(raw, source, tag, identity, hashes):
    s = obj(raw)
    exact_fields(s, ("format", "purpose", "repository", "source", "tag", "candidate_run_id",
                     "candidate_run_attempt", "release_json_sha256", "sha256sums_sha256",
                     "gates", "scope_decisions"), "signoff")
    require(type(s["format"]) is int and s["format"] == 1 and s["purpose"] == "BOXWEAVE_STABLE_PUBLICATION"
            and s["repository"] == REPO and s["source"] == source and s["tag"] == tag,
            "signoff does not bind exact stable candidate")
    require(valid_id(s["candidate_run_id"]) and valid_id(s["candidate_run_attempt"])
            and (s["candidate_run_id"], s["candidate_run_attempt"]) == identity,
            "signoff run/attempt mismatch")
    require((s["release_json_sha256"], s["sha256sums_sha256"]) == hashes, "signoff artifact mismatch")
    exact_fields(s["gates"], GATES, "acceptance gates")
    for gate in GATES:
        entry = s["gates"][gate]
        exact_fields(entry, ("status", "evidence"), gate)
        require(entry["status"] == "PASS", gate + " not signed PASS")
        validate_refs(entry["evidence"], 1)
    validate_refs(s["scope_decisions"], 0)
    return s


def validate_refs(refs, minimum):
    require(type(refs) is list and minimum <= len(refs) <= 32, "invalid evidence inventory")
    for item in refs:
        exact_fields(item, ("ref", "sha256"), "evidence reference")
        require(type(item["ref"]) is str and 1 <= len(item["ref"]) <= 512
                and all(32 <= ord(c) <= 126 for c in item["ref"])
                and type(item["sha256"]) is str and DIGEST.fullmatch(item["sha256"]) is not None,
                "invalid evidence reference/digest")


def openssl(args):
    p = subprocess.run(["openssl", *args], capture_output=True, timeout=15, check=False)
    require(p.returncode == 0, "pinned signer or signature verification failed")
    return p.stdout


def verify_signatures(raw, signature_owner, signature_verifier, owner_pem, verifier_pem):
    require(len(signature_owner) == len(signature_verifier) == 64, "invalid Ed25519 signature length")
    with tempfile.TemporaryDirectory(prefix="bw-stable-verify-") as tmp:
        d = Path(tmp)
        payload = d / "signoff.json"
        payload.write_bytes(raw)
        keys = []
        for identity, pem, sig in (("owner", owner_pem, signature_owner),
                                   ("verifier", verifier_pem, signature_verifier)):
            require(1 <= len(pem) <= 1024 and pem.startswith(b"-----BEGIN PUBLIC KEY-----"),
                    "pinned public key unavailable")
            key, signature = d / (identity + ".pem"), d / (identity + ".sig")
            key.write_bytes(pem)
            signature.write_bytes(sig)
            der = openssl(["pkey", "-pubin", "-in", str(key), "-outform", "DER"])
            require(len(der) == 44 and der.startswith(ED25519_SPKI), "pinned key is not Ed25519")
            keys.append(der)
            openssl(["pkeyutl", "-verify", "-pubin", "-inkey", str(key), "-rawin",
                     "-in", str(payload), "-sigfile", str(signature)])
        require(keys[0] != keys[1], "owner and independent verifier keys must differ")


def verify(directory, root, run, jobs, source, tag, raw, owner_sig, verifier_sig, owner_key, verifier_key):
    identity = candidate(run, jobs, source)
    hashes = verify_assets(directory, tag, source, root)
    verify_signatures(raw, owner_sig, verifier_sig, owner_key, verifier_key)
    validate_signoff(raw, source, tag, identity, hashes)
    return {"verdict": "STABLE_RELEASE_GATE_PASS", "source": source, "tag": tag,
            "run_id": identity[0], "run_attempt": identity[1], "signoff_sha256": sha(raw),
            "release_json_sha256": hashes[0], "sha256sums_sha256": hashes[1]}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ("assets", "root", "run", "jobs", "signoff", "owner_sig", "verifier_sig", "owner_key", "verifier_key"):
        p.add_argument("--" + name.replace("_", "-"), required=True, type=Path)
    p.add_argument("--source", required=True)
    p.add_argument("--tag", required=True)
    a = p.parse_args()
    try:
        v = verify(a.assets, a.root, obj(read_regular(a.run, MAX_JSON)), obj(read_regular(a.jobs, MAX_JSON)),
                   a.source, a.tag, read_regular(a.signoff, MAX_JSON), read_regular(a.owner_sig, 64),
                   read_regular(a.verifier_sig, 64), read_regular(a.owner_key, 1024), read_regular(a.verifier_key, 1024))
    except (OSError, ValueError, TypeError, subprocess.SubprocessError) as e:
        p.exit(1, "STABLE_RELEASE_GATE_BLOCKED: " + str(e) + "\n")
    print(json.dumps(v, sort_keys=True))


if __name__ == "__main__":
    main()
