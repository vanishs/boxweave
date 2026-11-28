"""Select required sources on disposable hosted CI; never alter trust."""
import os
import re
import stat
import subprocess
import sys
from pathlib import Path

URI = "https://packages.microsoft.com/ubuntu/24.04/prod"
ENTRY = re.compile(r"deb\s+(?:\[[^\]\n]+\]\s+)?"+re.escape(URI)+r"/?\s+noble\s+main")


def optional_source(text):
    active = [line.split("#", 1)[0].strip() for line in text.splitlines()]
    active = [line for line in active if line]
    return len(active) == 1 and ENTRY.fullmatch(active[0]) is not None


def select(root):
    main = root / "sources.list"
    paths = sorted((root / "sources.list.d").glob("*.list"))
    paths += sorted((root / "sources.list.d").glob("*.sources"))
    if main.exists():
        paths.append(main)
    matches = []
    for path in paths:
        text = path.read_text()
        if URI not in text:
            continue
        if path == main or path.suffix != ".list" or not optional_source(text):
            raise ValueError("optional CI repository layout needs review")
        matches.append(path)
    if len(matches) > 1:
        raise ValueError("ambiguous optional CI repository")
    return matches

ZABBLY_FINGERPRINT = "4EFC590696CB15B87C73A3AD82CC8797C838DCFD"


def verify_incus_key(path):
    # Check the first fingerprint exactly, as in the original callers.
    text = subprocess.check_output(["gpg", "--show-keys", "--with-colons", path], text=True)
    for line in text.splitlines():
        fields = line.split(":")
        if fields[0] == "fpr":
            if len(fields) > 9 and fields[9] == ZABBLY_FINGERPRINT:
                return
            break
    raise ValueError("unexpected Incus signing key")


def incus_package():
    # Same first matching Incus 7.6 repository version as the old awk filter.
    # Require the producer command itself to succeed, not just its filter.
    text = subprocess.check_output(["apt-cache", "madison", "incus"], text=True)
    for line in text.splitlines():
        fields = line.split()
        if len(fields) > 2 and "pkgs.zabbly.com" in line and re.search(r":7[.]6[-.]", fields[2]):
            return fields[2]
    raise ValueError("Incus 7.6 package not found in selected repository")


def main():
    if sys.argv[1:] == ["incus-package"]:
        print(incus_package())
        return
    if len(sys.argv) == 3 and sys.argv[1] == "verify-incus-key":
        verify_incus_key(sys.argv[2])
        return
    if len(sys.argv) != 1:
        raise SystemExit("unknown CI APT helper arguments")
    if (os.geteuid() != 0 or os.environ.get("GITHUB_ACTIONS") != "true"
            or os.environ.get("RUNNER_ENVIRONMENT") != "github-hosted"):
        raise SystemExit("disposable hosted CI only")
    root = Path("/etc/apt")
    for path in select(root):
        for parent in (path.parent, root, root.parent):
            info = parent.lstat()
            if (not stat.S_ISDIR(info.st_mode) or info.st_uid != 0
                    or info.st_mode & 0o022):
                raise SystemExit("untrusted source directory")
        info = path.lstat()
        if (not stat.S_ISREG(info.st_mode) or info.st_uid != 0
                or info.st_mode & 0o022 or info.st_nlink != 1):
            raise SystemExit("untrusted source file")
        dest = path.with_suffix(".list.bw-ci-unused")
        if dest.exists() or dest.is_symlink():
            raise SystemExit("previous CI source backup exists")
        path.rename(dest)
        print("CI_OPTIONAL_MS_PROD_SOURCE_NOT_REQUIRED", path.name, flush=True)


if __name__ == "__main__":
    main()
