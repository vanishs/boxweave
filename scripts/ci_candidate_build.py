"""Shared exact-source candidate build and real SQLite smoke for CI."""
import json
import os
from pathlib import Path
import re
import sqlite3
import subprocess
import sys
from contextlib import closing
from tempfile import TemporaryDirectory

REQUIRED = {"owned_boxes", "operations", "operation_resource_claims",
            "network_bindings", "network_address_fences"}
RETIRED = ("network_ipv4_retirement", "network_ipv4_compaction")


def require(condition, message):
    if not condition:
        raise ValueError(message)


def smoke(binary, version, source, temp_root):
    actual = json.loads(subprocess.check_output([str(binary), "build-info"]))
    require(actual == {"version": version, "commit": source}, "candidate build-info mismatch")
    # The runner's temp root need not satisfy SQLite's private-parent policy.
    with TemporaryDirectory(prefix="v1-cgo-sqlite.", dir=temp_root) as directory:
        parent = Path(directory)
        parent.chmod(0o700)
        db_path = parent / "ledger.sqlite"
        status = json.loads(subprocess.check_output([
            str(binary), "status", "-db", str(db_path), "-project", "no-project",
            "-incus-socket", "/unavailable-incus"]))
        require(status["ready"] is False, "unexpected native host readiness in isolated CI")
        with closing(sqlite3.connect(db_path.as_uri() + "?mode=ro", uri=True)) as db:
            require(db.execute("SELECT version FROM schema_version").fetchall() == [(7,)],
                    "candidate schema mismatch")
            tables = {row[0] for row in db.execute("SELECT name FROM sqlite_master WHERE type='table'")}
            require(REQUIRED.issubset(tables), "candidate tables missing")
            require(not any(t.startswith(RETIRED) for t in tables), "retired table present")
            require(db.execute("SELECT count(*) FROM operations").fetchone() == (0,),
                    "candidate operations not empty")


def build(version, source, arch, binary, temp_root):
    require(re.fullmatch(r"1\.[0-9]+\.[0-9]+(?:-rc\.[1-9][0-9]*)?", version), "invalid candidate version")
    require(re.fullmatch(r"[a-f0-9]{40}", source), "invalid candidate source")
    require(arch in ("amd64", "arm64"), "invalid candidate architecture")
    require(subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip() == source,
            "candidate source mismatch")
    require(subprocess.check_output(["go", "env", "GOARCH"], text=True).strip() == arch,
            "candidate architecture mismatch")
    require(subprocess.check_output(["go", "env", "CGO_ENABLED"], text=True).strip() == "1",
            "candidate requires CGO")
    subprocess.run(["go", "build", "-trimpath", "-buildvcs=false", "-ldflags",
                    f"-X main.version={version} -X main.commit={source} -buildid=",
                    "-o", str(binary), "./cmd/boxweave"],
                   check=True, env=dict(os.environ, CGO_ENABLED="1"))
    smoke(binary.resolve(strict=True), version, source, temp_root)


def main():
    if len(sys.argv) != 5:
        raise SystemExit("usage: ci_candidate_build.py VERSION SOURCE ARCH BINARY")
    version, source, arch, binary = sys.argv[1:]
    build(version, source, arch, Path(binary), Path(os.environ["RUNNER_TEMP"]).resolve())
    print("CI_CANDIDATE_BUILD_AND_SQLITE=PASS", arch, version, source, flush=True)


if __name__ == "__main__":
    main()
