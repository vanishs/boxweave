"""Select required sources on disposable hosted CI; never alter trust."""
import os
import re
import stat
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


def main():
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
