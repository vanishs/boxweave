"""Explicit local administrator jobs with durable receipts; never scheduled, never root."""
import hashlib
import json
import subprocess
import time
import uuid
from .domain import Fault

SCHEMA = """
CREATE TABLE job_runs (id TEXT PRIMARY KEY, actor TEXT NOT NULL, argv TEXT NOT NULL,
 state TEXT NOT NULL, exit_code INTEGER, duration REAL NOT NULL, created REAL NOT NULL,
 output_sha256 TEXT, output_bytes INTEGER NOT NULL, code TEXT NOT NULL);
"""
MAX_ITEMS, MAX_ARG, MAX_TOTAL, MAX_OUTPUT = 64, 4096, 16384, 1048576


def validate(argv):
    if (not isinstance(argv, list) or not 1 <= len(argv) <= MAX_ITEMS
            or any(not isinstance(a, str) or not a or len(a) > MAX_ARG
                   or any(ord(c) < 32 for c in a) for a in argv)
            or sum(len(a) for a in argv) > MAX_TOTAL):
        raise Fault("JOB_ARGV_INVALID", 422)
    return list(argv)


def run(store, argv, actor="operator", budget=60, clock=time.time, spawner=None):
    """Run one explicitly requested local command as the calling (non-root) user.

    The command never runs inside a shell, is never retried, and its output is
    hashed but never stored in the durable receipt.
    """
    argv = validate(argv)
    budget = min(600, max(1, int(budget)))
    ident = uuid.uuid4().hex
    created = clock()
    state, code, exit_code, output = "unknown", "OUTCOME_UNKNOWN", None, b""
    try:
        if spawner is not None:
            completed, output = spawner(argv, budget)
        else:
            completed = subprocess.run(argv, stdin=subprocess.DEVNULL, capture_output=True,
                                       timeout=budget)
            output = completed.stdout[-MAX_OUTPUT:] + completed.stderr[-MAX_OUTPUT:]
        exit_code = completed.returncode
        state, code = ("succeeded", "OK") if exit_code == 0 else ("failed", "JOB_EXITED")
    except subprocess.TimeoutExpired:
        state, code = "failed", "JOB_TIMEOUT"
    except (OSError, ValueError):
        state, code = "failed", "JOB_SPAWN_FAILED"
    duration = max(0.0, clock() - created)
    row = {"id": ident, "actor": actor, "argv": json.dumps(argv), "state": state,
           "exit_code": exit_code, "duration": duration, "created": created,
           "output_sha256": hashlib.sha256(output).hexdigest(),
           "output_bytes": len(output), "code": code}
    with store.db() as db:
        db.execute("INSERT INTO job_runs VALUES(?,?,?,?,?,?,?,?,?,?)", tuple(row.values()))
    return row


def receipts(store, actor="operator", limit=100):
    with store.db() as db:
        return [dict(r) for r in db.execute(
            "SELECT id,argv,state,exit_code,duration,created,output_sha256,output_bytes,code "
            "FROM job_runs WHERE actor=? ORDER BY created DESC,id DESC LIMIT ?", (actor, int(limit)))]
