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


def run(provider, plan, budget=900, clock=time.time, sleep=time.sleep, key=None):
    if getattr(provider, "mode", "") != "mcp":
        raise Fault("SELFTEST_LIVE_REQUIRED", 409)
    budget = min(1800, max(60, int(budget)))
    box = plan["box"]
    if key is None:
        key = (Path.home() / ".ssh" / "id_ed25519.pub").read_text(encoding="ascii").strip()
    if not re.fullmatch(r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}(?: [A-Za-z0-9@._-]{1,80})?", key):
        raise Fault("MANAGED_PUBLIC_KEY_REQUIRED", 422)
    deadline = clock() + budget
    steps = []

    def step(name, action):
        started = clock()
        try:
            action()
            steps.append({"step": name, "state": "succeeded", "code": "OK",
                          "seconds": round(clock() - started, 3)})
            return True
        except Fault as error:
            steps.append({"step": name, "state": "failed", "code": error.code,
                          "seconds": round(clock() - started, 3)})
        except Exception:
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
    return {"state": "passed" if ok else "failed", "box": box, "confirmation": confirmation(plan),
            "evidence": "live_mcp", "cleanup": cleanup, "steps": steps}
