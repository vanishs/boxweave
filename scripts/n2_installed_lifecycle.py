#!/usr/bin/env python3
"""Exact clean-runner acceptance client; also usable inside a disposable Box.

No production hooks, synthetic native receipts, arbitrary remote access or
credential output. The shell runner owns the clean-host authorization guard.
"""
import errno
import os
from pathlib import Path
import pty
import select
import sqlite3
import subprocess
import sys
import time

from mcp_testkit import LoopbackMCP


HELPER = "boxweave-ipv4-retirement.service"
CTL = "boxweave.service"
DB = "file:/var/lib/boxweave/ledger.sqlite?mode=ro"


def tty_run(argv, prompts):
    child, fd = pty.fork()
    if child == 0:
        os.execvp(argv[0], argv)
    deadline = time.monotonic() + 150
    pending = list(prompts)
    history = b""
    try:
        while time.monotonic() < deadline:
            ready, _, _ = select.select([fd], [], [], 1)
            if not ready:
                continue
            try:
                raw = os.read(fd, 65536)
            except OSError as exc:
                if exc.errno == errno.EIO:
                    break
                raise
            if not raw:
                break
            sys.stdout.buffer.write(raw)
            sys.stdout.buffer.flush()
            history = (history + raw)[-16384:]
            if pending and pending[0][0] in history:
                _, answer = pending.pop(0)
                os.write(fd, answer.encode() + b"\n")
                history = b""
        else:
            os.kill(child, 15)
            raise RuntimeError("bounded terminal fixture timeout")
    finally:
        os.close(fd)
    _, status = os.waitpid(child, 0)
    if os.waitstatus_to_exitcode(status) != 0 or pending:
        raise RuntimeError("reviewed terminal action failed or missing approval prompt")


def pid(unit):
    value = int(subprocess.check_output(["systemctl", "show", unit, "-p", "MainPID", "--value"], text=True))
    assert value > 1
    return value


def token(path):
    assert Path(path).stat().st_mode & 0o077 == 0
    return next(line[6:] for line in Path(path).read_text().splitlines() if line.startswith("token="))


def status(client):
    s = client.host_status()
    assert s["schema_version"] == 6 and s["ready"] and s["mcp_ready"], s
    return s


def wait_status(client, pred, seconds):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        s = status(client)
        if pred(s):
            return s
        time.sleep(1)
    raise RuntimeError("bounded retirement status wait failed")


def op(client, name, args, key):
    r = client.call(name, dict(args, idempotency_key="n2-installed-lifecycle-"+key,
                              work_ref="n2-clean-installed-lifecycle")).structured
    d = client.poll_operation(r["operation_id"], seconds=150)
    print(name, d["operation_id"], d["state"], d.get("native_operation_id"), flush=True)
    assert d["state"] == "SUCCEEDED", d
    return d


def mcp_retry(project, receipt):
    before_pid = pid(CTL)
    before_helper = pid(HELPER)
    with LoopbackMCP("http://127.0.0.1:18650/mcp", token=token(receipt),
                     allow_mutations=True, timeout=60) as c:
        before = status(c)
        assert before["native"]["project"] == project
        assert before["address_retirement"]["pending"] == 0
        capacity = before["address_capacity"]["remaining_ipv4_upper_bound"]
        b = op(c, "box_create", {"name": "n2-installed-retry",
                    "cpu": 1, "memory_mib": 256, "disk_gib": 2, "internet": False,
                    "start": False}, "create")["box_uuid"]
        subprocess.run(["systemctl", "stop", HELPER], check=True)
        assert pid(CTL) == before_pid
        d = op(c, "box_delete", {"box_id": b}, "delete")
        s = wait_status(c, lambda x: x["address_retirement"]["pending"] == 1, 20)
        assert s["address_retirement"]["last_deferred_reason"] == "RETIREMENT_HELPER_NOT_CONFIGURED"
        assert s["address_capacity"]["remaining_ipv4_upper_bound"] == capacity - 1
        subprocess.run(["systemctl", "start", HELPER], check=True)
        s = wait_status(c, lambda x: x["address_retirement"]["pending"] == 0 and
                        x["address_capacity"]["remaining_ipv4_upper_bound"] == capacity, 120)
        assert pid(CTL) == before_pid and pid(HELPER) != before_helper
        with sqlite3.connect(DB, uri=True) as db:
            row = db.execute("SELECT box_uuid, native_proof_sha256 FROM network_ipv4_retirements WHERE delete_operation_id=?",
                             (d["operation_id"],)).fetchone()
            assert row and row[0] == b and len(row[1]) == 64
        original = c.operation_get(d["operation_id"])
        assert original["state"] == "SUCCEEDED" and original["native_operation_id"] == d["native_operation_id"]
        print("N2_INSTALLED_HELPER_OUTAGE_RECOVERY_NO_DELETE_REPLAY=PASS", flush=True)


def main():
    action, project, *args = sys.argv[1:]
    if action == "tty-install":
        path, digest = args
        tty_run(["sudo", "sh", "install.sh", "--interactive", "--project", project,
                 "--local-binary", path, "--sha256", digest],
                [(b"to approve this exact plan:", "INSTALL "+project),
                 (b"Configure a new OAuth approval password?", "N")])
    elif action == "tty-uninstall":
        tty_run(["sudo", "sh", "uninstall.sh", "--full", "--project", project],
                [(b"to approve deleting all listed Boxes and retiring this controller:", "REMOVE "+project)])
    elif action == "mcp-retry":
        mcp_retry(project, args[0])
    elif action == "mcp-read":
        with LoopbackMCP("http://127.0.0.1:18650/mcp", token=token(args[0])) as c:
            s = status(c)
            assert s["native"]["project"] == project and s["address_retirement"]["pending"] == 0
    else:
        raise ValueError("unknown fixture task")


if __name__ == "__main__":
    main()
