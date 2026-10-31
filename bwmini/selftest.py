"""Confirmed live smoke: create one box, verify identity+hello, stop, delete.

Restores the 0.5.1 nested selftest on the new architecture. The plan is
durable and hash-confirmed; the run drives the same MCP contract as the
app lifecycle, never schedules itself, and always cleans up its box.
"""
import datetime
import hashlib
import json
import re
import secrets
import time
from pathlib import Path
from .domain import Fault

PLAN_PREFIX = "boxweave-selftest-"
PLAN_KEYS = {"schema_version", "box", "resources", "command", "created_utc"}
HELLO = "hello world\n"


def confirmation(plan):
    return hashlib.sha256(json.dumps(plan, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def build_plan(now=None):
    stamp = datetime.datetime.now(datetime.timezone.utc).isoformat() if now is None else now
    plan = {"schema_version": 1, "box": PLAN_PREFIX + secrets.token_hex(3),
            "resources": {"cpu": "1", "memory": "1GB", "disk": "8GB"},
            "command": "print('hello world')", "created_utc": stamp}
    return plan, confirmation(plan)


def load_plan(raw, expected_confirmation):
    """Accept only a selftest plan for this install, confirmed by its exact hash."""
    if (not isinstance(raw, dict) or set(raw) != PLAN_KEYS
            or raw["schema_version"] != 1
            or not isinstance(raw["box"], str) or not raw["box"].startswith(PLAN_PREFIX)
            or not re.fullmatch(r"[a-z][a-z0-9-]{0,62}", raw["box"])
            or raw["resources"] != {"cpu": "1", "memory": "1GB", "disk": "8GB"}
            or raw["command"] != "print('hello world')"
            or not isinstance(raw["created_utc"], str) or len(raw["created_utc"]) > 64):
        raise Fault("SELFTEST_PLAN_INVALID", 422)
    if (not isinstance(expected_confirmation, str)
            or not re.fullmatch(r"[a-f0-9]{64}", expected_confirmation)
            or expected_confirmation != confirmation(raw)):
        raise Fault("SELFTEST_CONFIRMATION_MISMATCH", 422)
    return raw


def _wait_absent(provider, box, deadline, clock, sleep):
    while True:
        if provider.lifecycle_inspect(box) is None:
            return None
        if clock() >= deadline:
            raise Fault("DEADLINE_EXCEEDED")
        sleep(2)


class Journal:
    """Durable per-step evidence for one confirmed selftest run.

    Rows record intent before each remote effect and the observed outcome
    after it; a killed process therefore leaves an explicit running row with
    the last step. Recording never acts on resources, and a journal write
    failure refuses the run before mutation rather than mutating unjournaled.
    """

    def __init__(self, store, actor="operator", clock=time.time):
        self.store, self.actor, self.clock = store, actor, clock
        self.run_id = None

    def start(self, plan, deadline_epoch):
        self.run_id = "st-" + secrets.token_hex(8)
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("INSERT INTO selftest_runs VALUES(?,?,?,?,?,?,?,?,?)",
                       (self.run_id, self.actor, confirmation(plan), plan["box"], "running",
                        datetime.datetime.now(datetime.timezone.utc).isoformat(timespec="seconds"),
                        datetime.datetime.fromtimestamp(deadline_epoch, datetime.timezone.utc)
                        .isoformat(timespec="seconds"), None, "pending"))
        return self.run_id

    def _mark(self, value):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            changed = db.execute("UPDATE selftest_runs SET last_step=? WHERE id=? AND state='running'",
                                 (value[:64], self.run_id)).rowcount
        if changed != 1:
            raise Fault("SELFTEST_JOURNAL_LOST", 503)

    def step_started(self, name):
        self._mark(name + ":started")

    def step_done(self, name, code):
        self._mark(name + ":" + code)

    def finish(self, state, cleanup):
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("UPDATE selftest_runs SET state=?,cleanup_state=? WHERE id=? AND state='running'",
                       (state, cleanup, self.run_id))

    @staticmethod
    def unreconciled(store):
        with store.db() as db:
            return [dict(row) for row in db.execute(
                "SELECT id,actor,plan_digest,exact_test_box,state,created_utc,deadline_utc,last_step,"
                "cleanup_state FROM selftest_runs WHERE state='running' ORDER BY created_utc")]

    @staticmethod
    def resolve(store, run_id, actor="operator"):
        """Operator-confirmed bookkeeping only; never touches the original box."""
        with store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute("SELECT id FROM selftest_runs WHERE id=? AND state='running'",
                             (run_id,)).fetchone()
            if row is None:
                raise Fault("SELFTEST_RUN_NOT_RUNNING", 404)
            db.execute("UPDATE selftest_runs SET state='resolved',cleanup_state='operator_resolved',"
                       "last_step=? WHERE id=?",
                       ("resolved:" + actor[:48], run_id))


def run(provider, plan, budget=900, clock=time.time, sleep=time.sleep, key=None, journal=None):
    if getattr(provider, "mode", "") != "mcp":
        raise Fault("SELFTEST_LIVE_REQUIRED", 409)
    budget = min(1800, max(60, int(budget)))
    box = plan["box"]
    if key is None:
        key = (Path.home() / ".ssh" / "id_ed25519.pub").read_text(encoding="ascii").strip()
    if not re.fullmatch(r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}(?: [A-Za-z0-9@._-]{1,80})?", key):
        raise Fault("MANAGED_PUBLIC_KEY_REQUIRED", 422)
    deadline = clock() + budget
    if journal is not None:
        if Journal.unreconciled(journal.store):
            raise Fault("SELFTEST_RUN_UNRECONCILED")
        journal.start(plan, deadline)
    steps = []

    def step(name, action):
        started = clock()
        if journal is not None:
            journal.step_started(name)
        try:
            action()
            if journal is not None:
                journal.step_done(name, "OK")
            steps.append({"step": name, "state": "succeeded", "code": "OK",
                          "seconds": round(clock() - started, 3)})
            return True
        except Fault as error:
            if journal is not None:
                journal.step_done(name, error.code)
            steps.append({"step": name, "state": "failed", "code": error.code,
                          "seconds": round(clock() - started, 3)})
        except Exception:
            if journal is not None:
                journal.step_done(name, "OUTCOME_UNKNOWN")
            steps.append({"step": name, "state": "failed", "code": "OUTCOME_UNKNOWN",
                          "seconds": round(clock() - started, 3)})
        return False

    def wait_state(predicate):
        while True:
            value = provider.lifecycle_inspect(box)
            if value is not None and predicate(value):
                return value
            if clock() >= deadline:
                raise Fault("DEADLINE_EXCEEDED")
            sleep(2)

    spec = dict(plan["resources"], ssh_public_key=key)
    ok = True
    ok = step("create", lambda: (provider.lifecycle_mutate("create", box, spec, deadline - clock()),
                                 wait_state(lambda v: v["state"] == "CONTAINER_STATE_RUNNING"), None)) and ok
    target = None
    try:
        value = provider.lifecycle_inspect(box)
        target = value["target"] if value else None
    except Fault:
        pass
    if ok:
        def hello():
            left = deadline - clock()
            if left <= 0:
                raise Fault("DEADLINE_EXCEEDED")
            result = provider.hello(target, "/home/" + target.user, left)
            if (not isinstance(result, dict) or result.get("exit_code") != 0
                    or result.get("stdout") != HELLO):
                raise Fault("COMMAND_FAILED")
        ok = step("hello", hello) and ok
    ok = step("stop", lambda: (provider.lifecycle_mutate("stop", box, spec, deadline - clock()),
                               wait_state(lambda v: v["state"] == "CONTAINER_STATE_STOPPED"), None)) and ok
    ok = step("delete", lambda: (provider.lifecycle_mutate("delete", box, spec, deadline - clock()),
                                 _wait_absent(provider, box, deadline, clock, sleep), None)) and ok
    cleanup = "not_needed" if ok else "best_effort"
    if not ok:
        try:
            if provider.lifecycle_inspect(box) is not None:
                provider.lifecycle_mutate("stop", box, spec, max(30, deadline - clock()))
                provider.lifecycle_mutate("delete", box, spec, max(30, deadline - clock()))
            cleanup = "performed"
        except Exception:
            cleanup = "best_effort_failed"
    if journal is not None:
        journal.finish("completed" if ok else "failed", cleanup)
    return {"state": "passed" if ok else "failed", "box": box, "confirmation": confirmation(plan),
            "evidence": "live_mcp", "cleanup": cleanup, "steps": steps}
