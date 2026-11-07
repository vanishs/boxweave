"""Observation-only recovery of a previously accepted asynchronous REST create.

The only possible follow-up write is the exact previously authorized optional
nesting postcondition, after native instance + guest SSH identity are attested.
No original create is ever retried, including across worker crashes.
"""
from .domain import Fault
from .remote_create import CreateJournal, CreateFault
from . import coordination_bridge, nesting

_READY = "CONTAINER_STATE_RUNNING"
_DEADLINE_SECONDS = 1800


def _hold_unknown(journal, task):
    journal.freeze(task, "READINESS_OUTCOME_UNKNOWN")
    try:
        coordination_bridge.SharedClaims().unknown(task)
    except Exception:
        pass


def observe_one(store, provider, clock):
    """Observe at most one due task; throttle is committed before network reads.

    Returns False if no task is due. The caller need not retain request
    arguments or secret material.
    """
    journal = CreateJournal(store, clock=clock)
    item = journal.accepted_observation()
    if item is None:
        return False
    task, logical = item["task_id"], item["logical"]
    if clock() >= item["created"] + _DEADLINE_SECONDS:
        _hold_unknown(journal, task)
        return True
    if getattr(provider, "rest", None) is None:
        _hold_unknown(journal, task)
        return True
    try:
        current = provider.lifecycle_inspect(logical)
    except Fault:
        return True  # No reply is never evidence of nonexistence.
    if not current or current["state"] != _READY:
        return True
    target = current["target"]
    # The only name that can have been created by an ordinary Containarium
    # user operation. This is necessary, not sufficient, identity evidence.
    if (target.logical != logical or target.instance != logical + "-container"
            or target.user != logical or not target.created):
        _hold_unknown(journal, task)
        return True
    try:
        proof = provider.lifecycle_identity(target, timeout=30)
    except Fault as exc:
        if exc.code in ("IDENTITY_MISMATCH", "TARGET_CHANGED"):
            _hold_unknown(journal, task)
        return True
    if (not isinstance(proof, dict) or proof.get("exit_code") != 0
            or proof.get("user") != logical
            or proof.get("hostname") != target.hostname):
        _hold_unknown(journal, task)
        return True
    wanted = item["requested_nesting"]
    if wanted is not None:
        if item["postcondition"] == "intent":
            # The previous invocation may have changed Incus before a crash.
            _hold_unknown(journal, task)
            return True
        try:
            plan = nesting.invoke({"action":"plan", "logical":logical,
                                   "nesting":bool(wanted)})
            native = plan["target"]
            if (native["logical"] != logical or
                    native["name"] != target.instance or
                    native["created"] != target.created or
                    not native.get("uuid")):
                _hold_unknown(journal, task)
                return True
            if item["postcondition"] == "configured":
                if (native["uuid"] != item["instance_uuid"] or
                        native["created"] != item["instance_created"] or
                        native["nesting"] != bool(wanted)):
                    _hold_unknown(journal, task)
                    return True
            elif item["postcondition"] == "not_started":
                if native["nesting"] == bool(wanted):
                    journal.nesting_already_set(task, native["uuid"],
                                                native["created"])
                else:
                    if not journal.begin_nesting(task):
                        _hold_unknown(journal, task)
                        return True
                    try:
                        result = nesting.invoke({
                            "action": "apply", "logical": logical,
                            "nesting": bool(wanted),
                            "expected": plan["expected"],
                            "confirm": native["name"]})
                        changed = result.get("target", {})
                        if (result.get("state") != "configured"
                                or changed.get("uuid") != native["uuid"]
                                or changed.get("created") != native["created"]
                                or changed.get("nesting") != bool(wanted)):
                            raise ValueError("NESTING_OUTCOME_UNVERIFIED")
                        journal.finish_nesting(task, native["uuid"],
                                               native["created"])
                    except Exception:
                        journal.freeze(task, "NESTING_POSTCREATE_UNKNOWN")
                        try:
                            coordination_bridge.SharedClaims().unknown(task)
                        except Exception:
                            pass
                        return True
            else:
                _hold_unknown(journal, task)
                return True
        except (ValueError, KeyError, TypeError, CreateFault):
            # Before apply, inability to inspect the native instance is
            # nonterminal; the fixed deadline eventually freezes the task.
            return True
    journal.mark_ready(task)
    try:
        coordination_bridge.SharedClaims().observed_terminal(
            task, "gateway_" + task)
    except Exception:
        # A failed release stays locked. Never clear claims from a read.
        pass
    return True
