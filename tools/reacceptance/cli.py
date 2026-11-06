"""One-shot source-tree developer CLI; keeps legacy commands available."""
import sys
from . import engine, scenarios, report, upstream, release_gate, surfaces, installed, lifecycle, installed_verifier, private_installer, unknown_inspect

def main(argv=None):
    args = list(sys.argv[1:] if argv is None else argv)
    if args and args[0] == "plan":
        return scenarios.main(args[1:])
    if args and args[0] == "report-system":
        return report.main(args[1:])
    if args and args[0] == "upstream":
        return upstream.main(args[1:])
    if args and args[0] == "unknown-inspect":
        return unknown_inspect.main(args[1:])
    if args and args[0] == "private-install":
        return private_installer.main(args[1:])
    if args and args[0] == "installed-verifier":
        return installed_verifier.main(args[1:])
    if args and args[0] == "lifecycle":
        return lifecycle.main(args[1:])
    if args and args[0] == "installed-readonly":
        return installed.main(args[1:])
    if args and args[0] == "breadth-safe":
        return surfaces.main(args[1:])
    if args and args[0] == "release-gate":
        return release_gate.main(args[1:])
    return engine.main(args)

if __name__ == "__main__":
    raise SystemExit(main())
