"""Bounded local Go regression and failure navigation; not acceptance."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys

FAIL = re.compile(r"^--- FAIL: (Test\w+|Fuzz\w+|Example\w*)(?:\s|$)", re.M)
DECL = re.compile(r"^func (Test\w+|Fuzz\w+|Example\w*)\(", re.M)
PACKAGE_FAIL = re.compile(r"^FAIL[\t ]+(\S+)", re.M)


def read_log(path):
    data = path.read_bytes()
    log = data.decode("utf-8", errors="replace")
    return (hashlib.sha256(data).hexdigest(), len(data),
            FAIL.findall(log), PACKAGE_FAIL.findall(log))


def locations(root):
    paths = subprocess.check_output(
        ["git", "ls-files", "-z", "--", "*_test.go"], cwd=root
    ).decode("utf-8").split("\0")
    result = {}
    for name in sorted(set(filter(None, paths))):
        p = root / name
        if p.is_symlink() or not p.is_file():
            continue
        for match in DECL.finditer(p.read_text(encoding="utf-8")):
            result.setdefault(match[1], []).append(name)
    return result


def summarize(root, log, previous=None):
    digest, size, names, packages = read_log(log)
    index = locations(root)
    unique = set(names)
    groups = {}
    for name in sorted(unique):
        hits = index.get(name, [])
        key = hits[0] if len(hits) == 1 else (
            "MISSING" if not hits else "AMBIGUOUS:" + ",".join(hits))
        groups.setdefault(key, []).append(name)
    lines = [
        "NAVIGATION ONLY; not acceptance or proof of complete execution",
        f"log_sha256={digest} bytes={size}",
        f"top_level_failure_lines={len(names)} unique_names={len(unique)}",
        "package_failure_records=" + (",".join(packages) or "NONE_REPORTED"),
    ]
    for path, tests in sorted(groups.items()):
        lines.append(f"{path} ({len(tests)}): " + ", ".join(tests))
    if previous is not None:
        old_hash, _, old_names, _ = read_log(previous)
        absent = set(old_names) - unique
        lines.append("previous_sha256=" + old_hash)
        lines.append("previous_failures_not_reported_now (NOT proof of PASS):")
        for name in sorted(absent):
            hits = index.get(name, [])
            lines.append(f"  {name}: " + (",".join(hits) or "MISSING"))
    return "\n".join(lines) + "\n"


def source_identity(root):
    """Observe tracked and nonignored untracked source; not a Git tree."""
    def git(*args):
        return subprocess.check_output(["git", *args], cwd=root)
    digest = hashlib.sha256()
    for data in (git("diff", "--binary", "HEAD", "--"),
                 git("diff", "--cached", "--binary", "HEAD", "--"),
                 git("status", "--porcelain=v1", "-z", "--untracked-files=all")):
        digest.update(len(data).to_bytes(8, "big") + data)
    for name in git("ls-files", "--others", "--exclude-standard", "-z").split(b"\0"):
        if not name:
            continue
        path = root / os.fsdecode(name)
        if path.is_symlink():
            data = os.fsencode(os.readlink(path))
        elif path.is_file():
            data = path.read_bytes()
        else:
            raise ValueError("unsupported untracked source")
        digest.update(len(name).to_bytes(8, "big") + name)
        digest.update(len(data).to_bytes(8, "big") + data)
    return {"head": git("rev-parse", "HEAD").decode().strip(),
            "local_observation_sha256": digest.hexdigest()}


def run_batch(root, packages, pattern, evidence, previous=None):
    root = root.resolve()
    evidence = evidence.resolve()
    top = subprocess.check_output(["git", "rev-parse", "--show-toplevel"], cwd=root)
    if Path(os.fsdecode(top).strip()).resolve() != root:
        raise ValueError("run from the checkout root")
    if evidence == root or root in evidence.parents:
        raise ValueError("evidence must be outside the checkout")
    if not packages or any(not re.fullmatch(r"\./[A-Za-z0-9_./-]*", p)
                           or ".." in p.split("/") for p in packages):
        raise ValueError("use explicit local package paths")
    before = source_identity(root)
    evidence.mkdir(mode=0o700)
    command = ["go", "test", "-count=1", "-v"]
    if pattern is not None:
        command += ["-run", pattern]
    command += packages
    request = {"argv": command, "source_before": before,
               "cgo_enabled": os.environ.get("CGO_ENABLED", "UNSET")}
    (evidence / "request.json").write_text(json.dumps(request, indent=2) + "\n")
    log = evidence / "go-test.log"
    with log.open("xb") as output:
        proc = subprocess.run(command, cwd=root, stdout=output,
                              stderr=subprocess.STDOUT, check=False)
    after = source_identity(root)
    summary = summarize(root, log, previous)
    data = log.read_text(encoding="utf-8", errors="replace")
    ran = sorted(set(re.findall(r"^=== RUN +((?:Test|Fuzz|Example)\w+)$", data, re.M)))
    passed = sorted(set(re.findall(r"^--- PASS: ((?:Test|Fuzz|Example)\w+)(?:\s|$)", data, re.M)))
    skipped = sorted(set(re.findall(r"^--- SKIP: ((?:Test|Fuzz|Example)\w+)(?:\s|$)", data, re.M)))
    summary += (f"reported_top_level_run={len(ran)} pass={len(passed)} skip={len(skipped)}\n"
                + "reported_pass_names=" + ",".join(passed) + "\n"
                + "reported_skip_names=" + ",".join(skipped) + "\n")
    result = {**request, "source_after": after, "source_drift": before != after,
              "guest_test_exit": proc.returncode, "log_sha256": read_log(log)[0],
              "reported_run": ran, "reported_pass": passed, "reported_skip": skipped}
    (evidence / "result.json").write_text(json.dumps(result, indent=2) + "\n")
    (evidence / "failure-index.txt").write_text(summary)
    print(f"test_exit={proc.returncode} source_drift={before != after} "
          f"evidence={evidence}")
    print(summary[:8192], end="")
    if len(summary) > 8192:
        print("\nProjection truncated; full index retained.")
    if before != after or (proc.returncode == 0 and (not ran or not passed)):
        print("REVIEW_REQUIRED: source drift or no reported passing top-level test.")
        return 2
    return proc.returncode if proc.returncode >= 0 else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("log", nargs="?", type=Path)
    parser.add_argument("--previous", type=Path)
    parser.add_argument("--run", nargs="+", metavar="PACKAGE")
    parser.add_argument("--pattern")
    parser.add_argument("--evidence-dir", type=Path)
    args = parser.parse_args()
    if args.run is not None:
        if args.log or args.evidence_dir is None:
            parser.error("--run requires --evidence-dir and no positional log")
        try:
            return run_batch(Path.cwd(), args.run, args.pattern,
                             args.evidence_dir, args.previous)
        except (OSError, ValueError, subprocess.SubprocessError) as exc:
            print(f"batch unavailable: {exc}", file=sys.stderr)
            return 2
    if args.log is None or args.pattern is not None or args.evidence_dir is not None:
        parser.error("provide a log or use --run")
    print(summarize(Path.cwd(), args.log, args.previous), end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
