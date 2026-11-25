"""Thin CLI dispatcher."""
import argparse

from . import checkpoint, ci, context, environment, evaluation, learning, profiles, tasks, verification, workflow, work
from .common import HarnessError, ROOT, emit

COMMANDS = {
    "work": (work, "Portable task entry and boundary records"),
    "start": (workflow.Start, "Plan or initialize development"),
    "resume": (workflow.Resume, "Review original handoff"),
    "status": (workflow.Status, "Summarize local work and evidence"),
    "finish": (workflow.Finish, "Review delivery conditions"),
    "profile": (profiles, "Observe named environment prerequisites"),
    "ci": (ci, "Review exact-head GitHub run metadata"),
    "eval": (evaluation, "Run deterministic harness behavior cases"),
    "doctor": (environment, "Observe development prerequisites"),
    "context": (context, "Discover repository assets"),
    "learn": (learning, "Capture and review learning candidates"),
    "verify": (verification, "Plan and record registered source checks"),
    "task": (tasks, "Track a task DAG with local revision checks"),
    "checkpoint": (checkpoint, "Capture or review a client-owned checkpoint"),
}


def main(argv=None):
    parser = argparse.ArgumentParser(prog="bwdev", description="BoxWeave development toolbox")
    parser.add_argument("--json", action="store_true", help="Machine-readable output")
    parser.add_argument("--version", action="version", version="bwdev 0.5.1")
    sub = parser.add_subparsers(dest="command", required=True)
    for name, (module, help_text) in COMMANDS.items():
        module.register(sub.add_parser(name, help=help_text))
    args = parser.parse_args(argv)
    try:
        data, code = COMMANDS[args.command][0].run(args, ROOT)
    except (HarnessError, OSError, UnicodeError) as exc:
        data, code = {"status": "ERROR", "error": str(exc)}, 2
    emit(data, args.json)
    return code
