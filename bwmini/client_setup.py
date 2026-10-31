"""Public-only unified Python 3.11+ MCP client setup generator.

Every App/OS/transport profile uses the same self-contained Python client
script. Interactive prompts go to stderr; stdout is valid Python source
starting with an executable shebang.
"""
import argparse
import json
import os
from pathlib import Path
import pwd
import re
import stat
import sys

PUBLIC_DESCRIPTOR = Path("/usr/local/lib/boxweave/default/client-public.json")


def load_public_state(path=None):
    """Only root-published, non-secret metadata; never read the private install marker."""
    path = PUBLIC_DESCRIPTOR if path is None else Path(path)
    meta = path.lstat()
    if (not stat.S_ISREG(meta.st_mode) or meta.st_uid != 0
            or meta.st_mode & 0o022 or meta.st_size > 16384):
        raise ValueError("PUBLIC_DESCRIPTOR_UNTRUSTED")
    for parent in path.parents:
        info = parent.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022:
            raise ValueError("PUBLIC_DESCRIPTOR_PARENT_UNTRUSTED")
    state = json.loads(path.read_text(encoding="utf-8"))
    if (not isinstance(state, dict) or state.get("schema_version") != 1
            or state.get("phase") != "ready"):
        raise ValueError("READY_INSTALL_REQUIRED")
    return state


try:
    from . import online, client_profiles
except ImportError:
    import online
    import client_profiles

HOST = re.compile(r"[A-Za-z0-9][A-Za-z0-9.-]{0,252}\Z")
USER = re.compile(r"[a-z_][a-z0-9_-]{0,30}\Z")
PUB = re.compile(r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}\Z")

def render(state, admin_user, replace=False, *, app="codex", system="windows", transport="wsl"):
    """Emit a single client-side Python entrypoint for every supported profile."""
    if type(replace) is not bool:
        raise ValueError("CLIENT_REPLACE_POLICY_INVALID")
    public = {name: state[name] for name in ("host", "port", "server_key")}
    return client_profiles.render(public | {
        "app": app, "os": system, "transport": transport,
        "admin_user": admin_user, "replace": replace,
    })


def publish(path, script):
    dest = Path(path).absolute()
    if (dest.is_symlink() or dest.exists() or dest.parent.is_symlink()
        or not dest.parent.is_dir()):
        raise ValueError("CLIENT_OUTPUT_MUST_BE_NEW_REGULAR_FILE")
    # Never write to an untrusted or world-writable output directory.
    parent = dest.parent.stat()
    if parent.st_mode & 0o002:
        raise ValueError("CLIENT_OUTPUT_PARENT_WORLD_WRITABLE")
    fd = os.open(dest, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    try:
        with os.fdopen(fd, "w", encoding="utf-8", newline="\r\n") as stream:
            stream.write(script)
        sudo_user = os.environ.get("SUDO_USER", "")
        if sudo_user not in ("", "root"):
            identity = pwd.getpwnam(sudo_user)
            # Only transfer ownership to the human when writing under their home.
            if dest.is_relative_to(Path(identity.pw_dir).resolve()):
                os.chown(dest, identity.pw_uid, identity.pw_gid)
    except BaseException:
        dest.unlink(missing_ok=True)
        raise
    return str(dest)


class ClientCancelled(Exception):
    """The human declined the reviewed client setup change."""


def choose(prompt, options):
    """Interactive, bounded list: [(label, value), ...]. No implicit yes."""
    print("\n# " + prompt, file=sys.stderr)
    for number, (label, _) in enumerate(options, 1):
        print(f"#   {number}) {label}", file=sys.stderr)
    while True:
        print("# Select number: ", end="", file=sys.stderr, flush=True)
        answer = input().strip()
        if answer.isdecimal() and 1 <= int(answer) <= len(options):
            return options[int(answer) - 1][1]
        print("# Choose one of the listed numbers.", file=sys.stderr)


def select(name, available, prompt, interactive):
    if name:
        if name not in available:
            raise ValueError("UNSUPPORTED_CLIENT_PROFILE")
        return name
    if not interactive:
        raise ValueError("NONINTERACTIVE_CLIENT_SELECTION_REQUIRED")
    return choose(prompt, [(item, item) for item in available])


def main(argv=None):
    parser = argparse.ArgumentParser(
        description="BoxWeave reviewed cross-platform MCP client setup generator",
        epilog="Apps are listed alphabetically; generated scripts require review and client-side execution.")
    parser.add_argument("--agent", "--app", dest="app", choices=client_profiles.APPS,
                        help="Target MCP application (alphabetical)")
    parser.add_argument("--os", dest="system", choices=client_profiles.SYSTEMS,
                        help="Operating system where the client app runs")
    parser.add_argument("--transport", choices=client_profiles.TRANSPORTS,
                        help="Native OpenSSH or Windows-to-WSL OpenSSH")
    parser.add_argument("--admin-user",
                        help="Existing Linux SSH administrator authorized to run sudo online.py enroll")
    destination = parser.add_mutually_exclusive_group()
    destination.add_argument("--output", help="Optional NEW setup script; refuses overwriting existing files")
    destination.add_argument("--stdout", action="store_true",
                             help="Print a copyable script to stdout (the default)")
    group = parser.add_mutually_exclusive_group()
    group.add_argument("--replace", dest="mcp_replace", action="store_const", const=True,
                       help="Replace a conflicting BoxWeave MCP entry only after read verification; backup")
    group.add_argument("--no-replace", dest="mcp_replace", action="store_const", const=False,
                       help="Refuse conflicting BoxWeave MCP entries (noninteractive default)")
    parser.set_defaults(mcp_replace=None)
    args = parser.parse_args(argv)
    try:
        state = load_public_state()
        if state.get("phase") != "ready":
            raise ValueError("READY_INSTALL_REQUIRED")
        interactive = sys.stdin.isatty()
        app = select(args.app, client_profiles.APPS, "Choose MCP application (alphabetical)",
                     interactive)
        # A noninteractive old --transport wsl invocation remains Windows by default.
        system = select(args.system or ("windows" if args.transport == "wsl" else None),
                        client_profiles.SYSTEMS, "Choose client operating system", interactive)
        valid_transports = ("native", "wsl") if system == "windows" else ("native",)
        transport = select(args.transport, valid_transports,
                           "Choose SSH transport", interactive)
        if not client_profiles.profile_valid(app, system, transport):
            raise ValueError("UNSUPPORTED_CLIENT_PROFILE")
        admin = args.admin_user or os.environ.get("SUDO_USER", "") or (
            pwd.getpwuid(os.getuid()).pw_name if os.geteuid() != 0 else "")
        if not USER.fullmatch(admin or ""):
            if not interactive:
                raise ValueError("ADMIN_USER_REQUIRED")
            print("# Existing Linux SSH administrator: ", end="", file=sys.stderr, flush=True)
            admin = input().strip()
        if not USER.fullmatch(admin):
            raise ValueError("ADMIN_USER_INVALID")
        if args.mcp_replace is not None:
            replace_mcp = args.mcp_replace
        elif interactive:
            replace_mcp = choose(
                "Replace an existing conflicting BoxWeave MCP entry on the client?",
                [("No: preserve the existing registration and stop for review", False),
                 ("Yes: replace only BoxWeave after authenticated MCP read; keep backup", True)])
        else:
            replace_mcp = False

        script = render(state, admin, replace_mcp,
                        app=app, system=system, transport=transport)
        shell = "Python 3.11+ (.py)"
        filename = "boxweave-setup.py"
        if args.output:
            published = publish(args.output, script)
            print("# " + shell + " setup saved to: " + published, file=sys.stderr)
        else:
            print("# BoxWeave " + app + " / " + system + " / " + transport +
                  " setup: copy the complete " + shell + " script below.", file=sys.stderr)
            print("# Or redirect stdout to " + filename + " and transfer it to the client.", file=sys.stderr)
            print("# Generation changed no Linux or client settings.", file=sys.stderr)
            sys.stdout.write(script)
        return 0
    except (ClientCancelled, KeyboardInterrupt, EOFError):
        print(json.dumps({"state": "cancelled", "changed": False}), file=sys.stderr)
        return 130
    except (ValueError, OSError, KeyError) as exc:
        print(json.dumps({"state": "error", "code": str(exc)}), file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
