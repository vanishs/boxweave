#!/usr/bin/env python3
"""Local source-only staged batch. No remote dispatch or publication."""
import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
import time
from pathlib import Path

from harness.verification import CHECKS

APPEND_DOCS = {
    "dev/LEARNING_INBOX.md",
    "docs/development/CONSOLIDATION_PROGRESS.md",
    "docs/development/CONSOLIDATION_ACCEPTANCE.md",
    "docs/development/RETIREMENT_INVENTORY.md",
    "docs/development/CONSOLIDATION_P1B_WITNESS_CROSSWALK.md",
}
MAX_EXPORT = 4 * 1024 * 1024


class BatchError(ValueError):
    pass


def git(root, *args):
    return subprocess.check_output(["git", *args], cwd=root)


def encoded(value):
    return json.dumps(value, ensure_ascii=True, separators=(",", ":")).encode("ascii")


def sha(value):
    if not re.fullmatch(r"[0-9a-f]{40}", value):
        raise BatchError("Expected exact SHA-1 object ID")
    return value


def snapshot(root):
    parent = sha(git(root, "rev-parse", "HEAD").decode().strip())
    tree = sha(git(root, "write-tree").decode().strip())
    if (subprocess.run(["git", "diff", "--quiet", "--"], cwd=root).returncode != 0
            or git(root, "ls-files", "--others", "--exclude-standard", "-z")):
        raise BatchError("Require staged-only worktree; preserve unstaged/untracked work")
    return parent, tree


def export(root, parent, tree, compact=True):
    """Build a text-only GitHub tree payload from exact local Git objects."""
    base = sha(git(root, "rev-parse", sha(parent) + "^{tree}").decode().strip())
    sha(tree)
    names = git(root, "diff", "--no-renames", "--name-only", "-z", base, tree).split(b"\0")
    rows = []
    for raw in filter(None, names):
        path = raw.decode("utf-8")
        if any(ord(c) < 32 or ord(c) == 127 for c in path):
            raise BatchError("Unsupported control character in path")
        entry = git(root, "ls-tree", "-z", tree, "--", path)
        if not entry or entry.count(b"\0") != 1:
            raise BatchError("Deletion/ambiguous path requires reviewed manual route")
        meta, actual = entry[:-1].split(b"\t", 1)
        mode, kind, blob = meta.decode("ascii").split()
        if actual != raw or kind != "blob" or mode not in ("100644", "100755"):
            raise BatchError("Unsupported Git entry")
        content = git(root, "cat-file", "blob", sha(blob))
        if len(content) > MAX_EXPORT or b"\0" in content:
            raise BatchError("Unsupported binary or oversized text")
        row = dict(path=path, mode=mode, type="blob", content=content.decode("utf-8"))
        if compact and path in APPEND_DOCS:
            prior = git(root, "ls-tree", "-z", base, "--", path)
            if prior:
                old_meta, old_path = prior[:-1].split(b"\t", 1)
                old_mode, old_kind, old_blob = old_meta.decode("ascii").split()
                if old_path == raw and old_kind == "blob" and old_mode == mode:
                    old = git(root, "cat-file", "blob", sha(old_blob))
                    if content.startswith(old) and len(content) > len(old):
                        row.pop("content")
                        row.update(base_blob=old_blob, append=content[len(old):].decode("utf-8"))
        rows.append(row)
    if not rows:
        raise BatchError("No staged changes")
    payload = dict(parent=parent, base=base, tree=tree, rows=rows)
    if len(encoded(payload)) > MAX_EXPORT:
        raise BatchError("Export too large; use bounded reviewed transfer")
    return payload


def framed_export(payload, max_rows=6, max_bytes=65536):
    """Index immutable text frames for caller-selected bounded delivery.

    Offsets are relative to the byte after the index newline. A frame is
    not a commit or a receipt. The caller still verifies the complete tree.
    """
    if not 1 <= max_rows <= 100 or not 1024 <= max_bytes <= MAX_EXPORT:
        raise BatchError("Invalid delivery frame bounds")
    identity = {key: sha(payload[key]) for key in ("parent", "base", "tree")}
    rows = payload["rows"]
    if not rows or len({r["path"] for r in rows}) != len(rows):
        raise BatchError("Empty or duplicate delivery rows")
    frames, index, group = [], [], []
    offset = 0

    def pack(items):
        return encoded(dict(identity, rows=items))

    def flush():
        nonlocal offset, group
        raw = pack(group)
        if len(raw) > max_bytes:
            raise BatchError("Single row exceeds frame bound; use reviewed ordinary export")
        index.append(dict(number=len(index)+1, offset=offset, bytes=len(raw),
                          sha256=hashlib.sha256(raw).hexdigest(), rows=len(group)))
        frames.append(raw)
        offset += len(raw)
        group = []

    for row in rows:
        if group and (len(group) == max_rows or len(pack(group + [row])) > max_bytes):
            flush()
        group.append(row)
    if group:
        flush()
    header = encoded(dict(identity, schema=1, kind="STAGED_EXPORT_FRAMES",
                          scope="LOCAL_EXPORT_ONLY", rows=len(rows), body_bytes=offset,
                          payload_sha256=hashlib.sha256(encoded(payload)).hexdigest(),
                          frames=index))
    if len(header) > 32768 or len(header)+1+offset > MAX_EXPORT:
        raise BatchError("Framed export exceeds bounded index or total size")
    return header + b"\n" + b"".join(frames)


def commands(root, payload, packages, registered):
    # Explicit packages only; caller selects affected and adjacent regressions.
    selected = sorted(set(packages))
    for package in selected:
        if not re.fullmatch(r"\./(?:internal|cmd)/[A-Za-z0-9_/-]+", package):
            raise BatchError("Expected explicit source package under internal/ or cmd/")
        if ".." in package.split("/") or not (root / package).is_dir():
            raise BatchError("Unknown source package")
    for row in payload["rows"]:
        if row["path"].endswith(".go") and "./" + str(Path(row["path"]).parent) not in selected:
            raise BatchError("Every changed Go package needs an explicit check")
    result = [("diff-check", ["git", "diff", "--cached", "--check"])]
    if selected:
        result += [("go-test", ["go", "test", "-count=1", *selected]),
                   ("go-vet", ["go", "vet", *selected])]
    for name in sorted(set(registered) - {"format"}) + ["format"]:
        result.append((name, [sys.executable, *CHECKS[name][1:]]))
    return result


def run(root, parent, tree, directory, packages, registered, timeout):
    if not 1 <= timeout <= 600:
        raise BatchError("Timeout per check must be 1..600 seconds")
    expected = (sha(parent), sha(tree))
    if snapshot(root) != expected:
        raise BatchError("Parent or staged tree changed")
    payload = export(root, *expected)
    checks = commands(root, payload, packages, registered)
    directory = Path(directory).absolute()
    if directory.is_relative_to(root) or directory.parent.resolve(strict=True) != directory.parent:
        raise BatchError("Use fresh evidence directory outside checkout")
    directory.mkdir(mode=0o700)
    results = []
    for name, argv in checks:
        if snapshot(root) != expected:
            raise BatchError("Source changed before check")
        start = time.monotonic()
        log_path = directory / (name + ".log")
        with log_path.open("xb") as log:
            os.chmod(log_path, 0o600)
            try:
                code = subprocess.run(argv, cwd=root, stdout=log, stderr=subprocess.STDOUT,
                                      timeout=timeout, check=False,
                                      env=dict(os.environ, PYTHONDONTWRITEBYTECODE="1",
                                               GOTOOLCHAIN="local")).returncode
                status = "PASS" if code == 0 else "FAIL"
            except subprocess.TimeoutExpired:
                code, status = None, "UNKNOWN"
            except OSError:
                code, status = None, "BLOCKED"
        with log_path.open("rb") as log:
            digest = hashlib.sha256()
            for chunk in iter(lambda: log.read(65536), b""):
                digest.update(chunk)
        results.append(dict(check=name, argv=argv, status=status, exit_code=code,
                            seconds=round(time.monotonic()-start, 3),
                            bytes=log_path.stat().st_size, sha256=digest.hexdigest()))
        try:
            unchanged = snapshot(root) == expected
        except BatchError:
            unchanged = False
        if not unchanged:
            results[-1]["status"] = "SOURCE_CHANGED"
            break
        if status != "PASS":
            break
    passed = len(results) == len(checks) and all(r["status"] == "PASS" for r in results)
    record = dict(status="LOCAL_STAGED_PASS" if passed else "NOT_VERIFIED",
                  parent=parent, tree=tree, checks=results, directory=str(directory),
                  binding="STAGED_TREE_ONLY", native_ci="NOT_ESTABLISHED")
    with (directory / "record.json").open("xb") as f:
        os.chmod(f.name, 0o600)
        f.write(encoded(record) + b"\n")
    if passed:
        record["export"] = payload
    return record, 0 if passed else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--parent", required=True)
    parser.add_argument("--tree", required=True)
    parser.add_argument("--evidence", required=True)
    parser.add_argument("--go-package", action="append", default=[])
    parser.add_argument("--check", action="append", choices=sorted(CHECKS), default=[])
    parser.add_argument("--timeout", type=int, default=240)
    parser.add_argument("--framed-export", action="store_true",
                        help="Emit an indexed export after local verification")
    parser.add_argument("--frame-rows", type=int, default=6)
    parser.add_argument("--frame-bytes", type=int, default=65536)
    args = parser.parse_args()
    try:
        if args.framed_export and not (1 <= args.frame_rows <= 100 and
                                       1024 <= args.frame_bytes <= MAX_EXPORT):
            raise BatchError("Invalid delivery frame bounds")
        root = Path(git(Path.cwd(), "rev-parse", "--show-toplevel").decode().strip()).resolve()
        result, code = run(root, args.parent, args.tree, args.evidence,
                           args.go_package, args.check, args.timeout)
        if code == 0 and args.framed_export:
            # The existing private record.json retains all verification facts.
            sys.stdout.buffer.write(framed_export(result["export"], args.frame_rows, args.frame_bytes))
        else:
            print(encoded(result).decode("ascii"))
        return code
    except (BatchError, OSError, subprocess.CalledProcessError, UnicodeError) as exc:
        print(json.dumps(dict(status="REFUSED", reason=str(exc)[:240])))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
