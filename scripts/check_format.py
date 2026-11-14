#!/usr/bin/env python3
"""Repository-wide format guard. No network, no third-party dependencies.

By default inspect tracked working-tree files; --cached inspects *staged blobs*
instead of the working tree so an unstaged correction cannot bypass a commit
check. Markdown's existing explicit two-space hard breaks remain valid, but
new diffs must also pass git diff --check in the hooks/CI.
"""
from __future__ import annotations

import argparse
import pathlib
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
TEXT_SUFFIXES = {".go", ".md", ".yml", ".yaml", ".json", ".sh", ".toml", ".py"}
TEXT_BASENAMES = {
    "VERSION", "Makefile", ".gitignore", ".editorconfig", ".gitattributes",
    ".yamllint", ".markdownlint.yaml", "AGENTS.md", "CONTRIBUTING.md",
    "CODEOWNERS", "pre-commit", "pre-push",
}


def git(*args: str, input_data: bytes | None = None) -> bytes:
    return subprocess.run(
        ["git", *args], cwd=ROOT, input=input_data,
        check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
    ).stdout


def tracked_files() -> list[str]:
    return [x.decode("utf-8") for x in git("ls-files", "-z").split(b"\0") if x]


def source_bytes(name: str, cached: bool) -> bytes:
    if cached:
        return git("show", ":" + name)
    return (ROOT / name).read_bytes()


def check_file(name: str, data: bytes) -> list[str]:
    path = pathlib.PurePosixPath(name)
    if path.suffix.lower() not in TEXT_SUFFIXES and path.name not in TEXT_BASENAMES:
        return []
    failures: list[str] = []
    try:
        data.decode("utf-8")
    except UnicodeDecodeError:
        return [f"{name}: tracked text must be UTF-8"]
    if not data.endswith(b"\n"):
        failures.append(f"{name}: missing final newline")
    if b"\r" in data:
        failures.append(f"{name}: CRLF / carriage return forbidden")
    if b"\0" in data:
        failures.append(f"{name}: NUL byte forbidden")
    for lineno, line in enumerate(data.split(b"\n"), 1):
        if not line:
            continue
        if name.endswith((".md", ".yml", ".yaml")) and b"\t" in line:
            failures.append(f"{name}:{lineno}: tabs forbidden in Markdown/YAML")
        if line.rstrip(b" \t") != line:
            if path.suffix.lower() == ".md" and line.endswith(b"  ") and not line.endswith(b"   "):
                # Pre-existing Markdown explicit hard breaks; new lines still
                # require git diff --check to pass.
                continue
            failures.append(f"{name}:{lineno}: trailing whitespace")
    if path.suffix.lower() == ".json":
        import json
        try:
            json.loads(data)
        except (json.JSONDecodeError, UnicodeDecodeError) as exc:
            failures.append(f"{name}: invalid JSON ({exc})")
    if path.suffix.lower() == ".go":
        process = subprocess.run(
            ["gofmt", "-d"], input=data, capture_output=True, cwd=ROOT
        )
        if process.returncode or process.stdout:
            failures.append(f"{name}: gofmt mismatch (run gofmt -w)\n{process.stdout.decode(\"utf-8\", errors=\"replace\")[:3000]}")
    return failures


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cached", action="store_true", help="validate index, not working tree")
    args = parser.parse_args()
    errors: list[str] = []
    for name in tracked_files():
        try:
            errors += check_file(name, source_bytes(name, args.cached))
        except (OSError, subprocess.CalledProcessError) as exc:
            errors.append(f"{name}: could not inspect tracked file ({exc})")
    if errors:
        for err in errors[:100]:
            print("FORMAT FAIL:", err, file=sys.stderr)
        if len(errors) > 100:
            print(f"FORMAT FAIL: {len(errors) - 100} additional issues", file=sys.stderr)
        return 1
    print("FORMAT PASS: all tracked source, configs and docs")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
