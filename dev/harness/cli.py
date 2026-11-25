"""Thin CLI dispatcher."""
import argparse

from . import checkpoint, context, environment
from .common import HarnessError, ROOT, emit

COMMANDS = {
    "doctor": (environment, "Observe development prerequisites"),
    "context": (context, "Discover repository assets"),
    "checkpoint": (checkpoint, "Capture or review a client-owned checkpoint"),
}


def main(argv=None):
    parser = argparse.ArgumentParser(prog="bwdev", description="BoxWeave development toolbox")
    parser.add_argument("--json", action="store_true", help="Machine-readable output")
    parser.add_argument("--version", action="version", version="bwdev 0.1.0")
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
