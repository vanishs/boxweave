#!/usr/bin/env python3
"""Root-owned web CLI bootstrap into the staged, immutable release environment."""
import json
import os
import pwd
from pathlib import Path

cfg = json.loads(Path("/etc/boxweave/web.json").read_text())
release = Path(cfg["release"])
if os.geteuid() == 0:
    account = pwd.getpwnam("boxweave-web")
    os.initgroups(account.pw_name, account.pw_gid)
    os.setgid(account.pw_gid)
    os.setuid(account.pw_uid)
os.environ["PYTHONPATH"] = str(release / "server")
os.execv(
    str(release / "venv/bin/python"),
    [
        str(release / "venv/bin/python"),
        "-m",
        "webui.command",
        *__import__("sys").argv[1:],
    ],
)
