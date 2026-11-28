#!/bin/sh
# BoxWeave v1 distribution coordinator. Uses only Python 3 stdlib.
# Host writes require explicit root approval: reviewed --apply or terminal consent.
set -eu
exec python3 - "$@" <<'PY_BOXWEAVE_INSTALL'
import argparse
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import platform
import pwd
import re
import socket
import stat
import subprocess
import sys
import tempfile
import time
import urllib.parse
import urllib.request

REPOSITORY = "vanishs/boxweave"
API = "https://api.github.com/repos/" + REPOSITORY
POINTER_URL = "https://raw.githubusercontent.com/" + REPOSITORY + "/v1/V1_PUBLISHED_RELEASE"
TAG_RE = re.compile(r"^v1\.[0-9]+\.[0-9]+(?:-rc\.[1-9][0-9]*)?$")
SHA_RE = re.compile(r"^[0-9a-f]{64}$")
COMMIT_RE = re.compile(r"^[0-9a-f]{40}$")
PROJECT_RE = re.compile(r"^[a-z][a-z0-9-]{0,62}$")
MAX_BINARY = 100 * 1024 * 1024
MAX_METADATA = 1 * 1024 * 1024
STATE_DIR = Path("/var/lib/boxweave")
INSTALLED_BINARY = Path("/usr/local/bin/boxweave")
INSTALLED_UNIT = Path("/etc/systemd/system/boxweave.service")
BACKUP_DIR = Path("/var/lib/boxweave-installer-backup")
RESTRICTED_SOCKET = "/var/lib/incus/unix.socket.user"
SCOPED_PATHS = (
    Path("/var/lib/boxweave-purge-v1.json"),
    Path("/var/lib/boxweave-scoped-retirement-v1.json"),
    Path("/var/lib/boxweave-scoped-drain-v1.json"),
    Path("/var/lib/boxweave-scoped-peer-v2"),
    Path("/etc/systemd/system/boxweave-scoped-guard.service"),
    Path("/etc/systemd/system/boxweave-scoped-lifecycle.service"),
    Path("/etc/systemd/system/incus.service.d/50-boxweave-scoped-guard.conf"),
    Path("/run/boxweave-scoped-lifecycle.sock"),
)


class InstallError(Exception):
    pass


class HTTPSOnlyRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        if urllib.parse.urlsplit(newurl).scheme != "https":
            raise InstallError("non-HTTPS redirect refused")
        return super().redirect_request(req, fp, code, msg, headers, newurl)


OPENER = urllib.request.build_opener(HTTPSOnlyRedirect())


def get_bytes(url, limit):
    if urllib.parse.urlsplit(url).scheme != "https":
        raise InstallError("HTTPS required")
    req = urllib.request.Request(
        url,
        headers={
            "User-Agent": "boxweave-v1-install/1",
            "Accept": "application/vnd.github+json" if url.startswith(API) else "application/octet-stream",
        },
    )
    with OPENER.open(req, timeout=30) as response:
        data = response.read(limit + 1)
    if len(data) > limit:
        raise InstallError("download exceeds authorized maximum size")
    return data


def get_json(url):
    return json.loads(get_bytes(url, MAX_METADATA))


def digest(data):
    return hashlib.sha256(data).hexdigest()


def checked_tag(tag):
    if not TAG_RE.fullmatch(tag):
        raise InstallError("expected v1.X.Y or v1.X.Y-rc.N (never legacy v0.x)")
    return tag


def current_host_digest():
    machine_id = Path("/etc/machine-id").read_text(encoding="ascii").strip()
    if not re.fullmatch(r"[0-9a-f]{32}", machine_id):
        raise InstallError("untrusted or missing /etc/machine-id")
    return hashlib.sha256(machine_id.encode("ascii")).hexdigest()


def asset_from_release(release, name, tag):
    candidates = [x for x in release.get("assets", []) if x.get("name") == name]
    if len(candidates) != 1:
        raise InstallError("missing or duplicate release asset: " + name)
    asset = candidates[0]
    url = "https://github.com/" + REPOSITORY + "/releases/download/" + tag + "/" + name
    if asset.get("browser_download_url") != url:
        raise InstallError("asset URL does not match the pinned GitHub release")
    api_digest = asset.get("digest")
    if not isinstance(api_digest, str) or not re.fullmatch(r"sha256:[0-9a-f]{64}", api_digest):
        raise InstallError("GitHub asset digest unavailable; fail closed")
    return url, api_digest[7:]


def download_checked_asset(release, name, tag, size_limit):
    url, expected = asset_from_release(release, name, tag)
    data = get_bytes(url, size_limit)
    if digest(data) != expected:
        raise InstallError("GitHub release asset SHA256 mismatch: " + name)
    return data


def select_release(tag, arch):
    tag = checked_tag(tag)
    rc = "-rc." in tag
    release = get_json(API + "/releases/tags/" + tag)
    if release.get("tag_name") != tag or release.get("draft") or bool(release.get("prerelease")) != rc:
        raise InstallError("expected published release kind/tag not found")
    reference = get_json(API + "/git/ref/tags/" + tag)
    obj = reference.get("object", {})
    if obj.get("type") != "commit" or not COMMIT_RE.fullmatch(str(obj.get("sha", ""))):
        raise InstallError("release tag must be an immutable lightweight commit ref")
    version = tag[1:]
    asset_name = "boxweave-" + tag + "-linux-" + arch
    manifest_bytes = download_checked_asset(release, "release.json", tag, MAX_METADATA)
    manifest = json.loads(manifest_bytes)
    if not isinstance(manifest, dict) or manifest.get("format") != 1 or manifest.get("product") != "boxweave-v1" or manifest.get("tag") != tag or manifest.get("version") != version or manifest.get("source") != obj["sha"]:
        raise InstallError("release manifest identity/tag/source mismatch")
    assets = manifest.get("assets")
    if not isinstance(assets, dict) or set(assets) != {
        "boxweave-" + tag + "-linux-amd64",
        "boxweave-" + tag + "-linux-arm64",
    }:
        raise InstallError("release manifest does not bind exactly two architectures")
    checksums = download_checked_asset(release, "SHA256SUMS", tag, MAX_METADATA)
    sums = {}
    for line in checksums.decode("ascii").splitlines():
        fields = line.split("  ", 1)
        if len(fields) != 2 or not SHA_RE.fullmatch(fields[0]) or "/" in fields[1] or fields[1] in sums:
            raise InstallError("malformed SHA256SUMS")
        sums[fields[1]] = fields[0]
    if sums.get("release.json") != digest(manifest_bytes) or sums.get(asset_name) != assets[asset_name].get("sha256"):
        raise InstallError("release checksums mismatch")
    binary = download_checked_asset(release, asset_name, tag, MAX_BINARY)
    if digest(binary) != sums[asset_name] or len(binary) != assets[asset_name].get("size"):
        raise InstallError("release binary digest or size mismatch")
    return binary, version, obj["sha"], digest(binary)



def private_json(path, owner):
    """Bounded manifest/receipt read; never follow a symlink or print secrets."""
    with os.fdopen(os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK), "rb") as f:
        info = os.fstat(f.fileno())
        if not stat.S_ISREG(info.st_mode) or stat.S_IMODE(info.st_mode) != 0o600 or info.st_uid != owner or info.st_nlink != 1:
            raise InstallError("unsafe installer metadata: " + path.name)
        raw = f.read(MAX_METADATA + 1)
    if len(raw) > MAX_METADATA:
        raise InstallError("installer metadata exceeds size limit")
    value = json.loads(raw)
    if not isinstance(value, dict):
        raise InstallError("invalid installer metadata: " + path.name)
    return value, digest(raw)


def command_json(command):
    result = json.loads(subprocess.check_output(command, text=True, timeout=90))
    if not isinstance(result, dict):
        raise InstallError("invalid installer command result")
    return result


def service_command(args, binary, command, *extra):
    return ["runuser", "-u", args.service_user, "--", "env",
            "-u", "INCUS_CONF", "-u", "INCUS_CONFIG_DIR", "-u", "XDG_CONFIG_HOME",
            "HOME=/home/boxweave", "XDG_CONFIG_HOME=/home/boxweave/.config",
            str(binary), command, *extra]


def refuse_scoped_existing():
    for path in SCOPED_PATHS:
        try:
            path.lstat()
        except FileNotFoundError:
            continue
        except OSError as exc:
            raise InstallError("scoped installation evidence unreadable; preserve original state") from exc
        raise InstallError("scoped installation evidence exists: " + str(path) +
                           "; preserve original state, no legacy adoption or upgrade")
    # Root maintenance must also refuse a surviving manifest even if all
    # scoped native paths are missing. Nonroot distribution previews never
    # open the service's private metadata.
    if os.geteuid() == 0:
        manifest_path = STATE_DIR / "install-manifest.json"
        try:
            manifest_path.lstat()
        except FileNotFoundError:
            return
        uid = pwd.getpwnam("boxweave").pw_uid
        manifest, _ = private_json(manifest_path, uid)
        if manifest.get("scoped_peer_mode") not in (None, ""):
            raise InstallError("scoped manifest: preserve original, no legacy adoption or upgrade")


# Mirror the Go Stage retirement-reader contract even when installing an
# older immutable Release. Never run these executables or install packages.
RETIREMENT_READERS = (
    ("/usr/sbin/ip", "iproute2"),
    ("/usr/sbin/bridge", "iproute2"),
    ("/usr/sbin/nft", "nftables"),
    ("/usr/sbin/conntrack", "conntrack"),
)


def retirement_reader_preflight(stat_path=os.stat):
    failures = []
    for path, package in RETIREMENT_READERS:
        try:
            info = stat_path(path)
        except FileNotFoundError:
            failures.append("INSTALL_MISSING_PREREQUISITE: " + path +
                            " (Debian/Ubuntu package: " + package + ")")
            continue
        except OSError:
            failures.append("INSTALL_PREREQUISITE_UNVERIFIED: " + path)
            continue
        if (info.st_uid != 0 or not stat.S_ISREG(info.st_mode) or
                info.st_mode & 0o022 or not info.st_mode & 0o111):
            failures.append("INSTALL_PREREQUISITE_UNVERIFIED: " + path)
    if failures:
        raise InstallError("; ".join(failures) +
                           "; resolve locally and review again; no installation changes")
    return True


def scoped_review(candidate, project):
    review = command_json([str(candidate), "native-scoped-fresh-plan"])
    machine = Path("/etc/machine-id").read_text(encoding="ascii").strip()
    native_host = digest(("boxweave-native-host-v1|" + machine).encode("ascii"))
    if (review.get("Mode") != "SCOPED_PEER_FRESH_V2" or review.get("Project") != project or
            review.get("HostSHA256") != native_host or
            not all(SHA_RE.fullmatch(str(review.get(k, ""))) for k in
                    ("ReviewSHA256", "NativeSHA256", "GuardSHA256"))):
        raise InstallError("scoped fresh review mode, Host, Project or digest mismatch")
    return review


# Type-wide systemd service drop-ins can apply only once the unit is
# installed: systemctl show of a not-found unit can report no DropInPaths.
# This is a conservative advisory, NOT ownership proof or a permission to
# ignore an effective drop-in. Never modify inherited systemd policy.
# Read-only advisory. A recognized byte sequence is not a verified effective
# unit or approval. Keep the actual retirement contract in the installed
# observer and exact uninstaller; never change the inherited policy here.
KNOWN_LXC_POLICY_PATH = Path("/run/systemd/system/service.d/zzz-lxc-service.conf")
KNOWN_LXC_POLICY_SHA = "fcbd512cb8212c6f5720f28398f01e85ff96246aebb73e09cf449d2f3aa2c5b0"


def known_lxc_policy_bytes(path):
    before = path.lstat()
    if not stat.S_ISREG(before.st_mode) or before.st_size > 4096:
        return False
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as stream:
        opened = os.fstat(stream.fileno())
        if not stat.S_ISREG(opened.st_mode) or not os.path.samestat(before, opened):
            return False
        data = stream.read(4097)
        after = os.fstat(stream.fileno())
    def identity(st):
        return (st.st_dev, st.st_ino, st.st_mode, st.st_uid, st.st_gid,
                st.st_nlink, st.st_size, st.st_mtime_ns, st.st_ctime_ns)
    return (identity(before) == identity(opened) == identity(after) == identity(path.lstat())
            and len(data) <= 4096 and digest(data) == KNOWN_LXC_POLICY_SHA)


def inherited_service_dropin_hint(paths=None):
    if paths is None:
        paths = ("/etc/systemd/system/service.d", "/run/systemd/system/service.d",
                 "/usr/local/lib/systemd/system/service.d",
                 "/usr/lib/systemd/system/service.d", "/lib/systemd/system/service.d")
    try:
        entries = set()
        for folder in paths:
            try:
                with os.scandir(folder) as listing:
                    entries.update(Path(entry.path) for entry in listing if entry.name.endswith(".conf"))
            except FileNotFoundError:
                continue
        if not entries:
            return "NO_GLOBAL_HINT_NOT_EFFECTIVE_PROOF"
        if entries == {KNOWN_LXC_POLICY_PATH} and known_lxc_policy_bytes(KNOWN_LXC_POLICY_PATH):
            return "KNOWN_LXC255_POLICY_BYTES_REVIEW_REQUIRED"
        return "POTENTIAL_INHERITED_SERVICE_DROPIN"
    except OSError:
        return "INHERITED_SERVICE_DROPIN_UNVERIFIED"


def print_lifecycle_preflight(scoped=False):
    hint = inherited_service_dropin_hint()
    print("maintenance_preflight=" + hint, flush=True)
    if hint == "KNOWN_LXC255_POLICY_BYTES_REVIEW_REQUIRED" and scoped:
        print("Maintenance: recognized LXC service.d policy bytes. This Scoped retirement profile has a reviewed compatibility route; installed effective units, mapped container identity, original scope and policy still require verification. This is not a runtime-hardening or uninstall approval. Do not remove inherited policy.", flush=True)
    elif hint != "NO_GLOBAL_HINT_NOT_EFFECTIVE_PROOF":
        print("Maintenance warning: inherited service drop-ins may block verified activation (Legacy) or full retirement. Review effective units before installation; do not delete inherited policy. A working Scoped controller is not proof of retirement readiness.", flush=True)
    else:
        print("Maintenance preflight is advisory; only the installed effective-unit review can prove retirement readiness.", flush=True)
    return hint


def print_scoped_review(review):
    print("Reviewed scoped native preparation:", json.dumps(review, sort_keys=True), flush=True)
    # The displayed pool is derived from the exact reviewed CIDR, not an
    # extra mutable input or another approval digest. Older/partial reports
    # must never be represented as a verified capacity.
    try:
        prefix = ipaddress.IPv4Interface(review["IPv4CIDR"])
        if not 16 <= prefix.network.prefixlen <= 30:
            raise ValueError("unsupported prefix")
        nominal = prefix.network.num_addresses - 3
        print("Reviewed IPv4 capacity: " + str(prefix) + "; nominal identities=" +
              str(nominal) + "; live Box ledger limit=1024; retired identities remain held.", flush=True)
    except (KeyError, ValueError, TypeError):
        print("Reviewed IPv4 capacity: UNVERIFIED; inspect native review before approval.", flush=True)
    print("No implicit resize or address reuse. Larger exclusive sizing is fresh-bootstrap only.", flush=True)
    print("scoped_fresh_review_sha256=" + review["ReviewSHA256"], flush=True)


def phase_unverified(name):
    return InstallError("INSTALL_PHASE_UNVERIFIED: " + name +
                        "; retain original files and journals; inspect install-status and doctor locally; do not reinstall blindly")


def install_phase(name, command, capture=False):
    print("INSTALL_PHASE=" + name, flush=True)
    try:
        if capture:
            return command_json(command)
        subprocess.run(command, check=True)
    except (OSError, ValueError, InstallError, subprocess.SubprocessError) as exc:
        raise phase_unverified(name) from exc


def doctor_postflight(command):
    """Reobserve only the exact installed Doctor; never repeat installation."""
    print("INSTALL_PHASE=CONTROLLER_DOCTOR", flush=True)
    for attempt in range(3):
        try:
            return command_json(command)
        except subprocess.CalledProcessError as exc:
            # Doctor emits redacted JSON even when preflight is incomplete.
            # Log only bounded typed classifications, never raw stdout/intent.
            try:
                report = json.loads(exc.output or "")
                d = report["diagnostic"]
                i = d["installation"]
                fields = {
                    "mode": i.get("mode"), "candidate": i.get("candidate"),
                    "manifest": i.get("manifest"), "identity": i.get("installed_identity"),
                    "runtime": i.get("runtime"), "installation_reason": i.get("reason"),
                    "guard": i.get("guard_unit"), "lifecycle": i.get("lifecycle_unit"),
                    "root_journal": i.get("root_journal"), "root_proof": i.get("root_proof"),
                    "artifacts": i.get("scoped_artifacts"), "dependencies": i.get("dependencies"),
                    "database": d["database"].get("state"), "database_reason": d["database"].get("reason"),
                    "native": d["native"].get("state"), "native_reason": d["native"].get("reason"),
                    "mcp": d["mcp"].get("state"), "mcp_reason": d["mcp"].get("reason"),
                }
                fields = {k: v if isinstance(v, str) and re.fullmatch(r"[A-Z0-9_]{1,100}", v)
                          else "UNAVAILABLE" for k, v in fields.items()}
                print("CONTROLLER_DOCTOR_INCOMPLETE " + json.dumps(fields, sort_keys=True),
                      file=sys.stderr, flush=True)
                # A startup/observation gap may settle. A legacy/mixed
                # identity, wrong schema or foreign ledger is not retried.
                retry = (i.get("candidate") == "SCOPED_V2" and i.get("mode") in ("SCOPED_V2", "UNVERIFIED")
                         and i.get("manifest") == i.get("installed_identity") == "VERIFIED"
                         and report.get("schema_version") == 7 and d["database"].get("state") == "OBSERVED"
                         and isinstance(d.get("ledger"), dict) and d["ledger"].get("foreign_rows", 0) == 0
                         and i.get("service_unit") == "ACTIVE"
                         and d["mcp"].get("reason") in ("LOCAL_MCP_UNAVAILABLE", "CONTROLLER_RUNTIME_UNVERIFIED"))
            except (ValueError, KeyError, TypeError, AttributeError):
                retry = False
            if not retry or attempt == 2:
                raise phase_unverified("CONTROLLER_DOCTOR") from exc
            time.sleep(1)
        except (OSError, ValueError, InstallError, subprocess.SubprocessError) as exc:
            raise phase_unverified("CONTROLLER_DOCTOR") from exc


def normalize_oauth_origin(raw):
    # Only a single terminal slash is accepted as a convenience.
    # The installed CLI remains the authoritative validator.
    origin = raw[:-1] if raw.endswith("/") else raw
    if not 12 <= len(origin) <= 255 or not origin.startswith("https://"):
        raise InstallError("OAUTH_ORIGIN_INVALID")
    host = origin[len("https://"):]
    labels = host.split(".")
    if (len(labels) < 2 or host.endswith(".localhost") or
            not all(re.fullmatch(r"[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?", label)
                    for label in labels)):
        raise InstallError("OAUTH_ORIGIN_INVALID")
    try:
        ipaddress.ip_address(host)
    except ValueError:
        return origin
    raise InstallError("OAUTH_ORIGIN_INVALID")


def oauth_next(args, code):
    # No origin or credential is echoed on failure.
    print(code + ": controller installation complete; no reinstall needed")
    prefix = "sudo runuser -u " + args.service_user + " -- /usr/local/bin/boxweave "
    print("Check: " + prefix + "oauth-status -project " + args.project)
    print("If PasswordConfigured is false, use oauth-setup once; otherwise keep the existing password.")
    print("Continue: " + prefix + "oauth-set-origin -project " + args.project +
          " -public-origin HTTPS_ORIGIN")


def configure_oauth(identity, args, inp, out):
    status_cmd = identity + ["oauth-status", "-project", args.project]
    current = command_json(status_cmd)
    # Older candidates lack PasswordConfigured; use the fresh Doctor observation.
    password = current.get("PasswordConfigured", getattr(args, "oauth_password_configured", None))
    if not isinstance(password, bool):
        oauth_next(args, "OAUTH_PASSWORD_STATE_UNVERIFIED")
        return 2
    if not password:
        out.write("Configure a new OAuth approval password? [y/N] ")
        out.flush()
        if inp.readline().strip().lower() != "y":
            oauth_next(args, "OAUTH_CONFIGURATION_DEFERRED")
            return 0
        try:
            subprocess.run(identity + ["oauth-setup", "-project", args.project],
                           stdin=inp, stdout=out, stderr=out, check=True)
        except (OSError, subprocess.SubprocessError):
            oauth_next(args, "OAUTH_PASSWORD_SETUP_UNVERIFIED")
            return 2
        current = command_json(status_cmd)
        if current.get("PasswordConfigured", True) is not True:
            oauth_next(args, "OAUTH_PASSWORD_SETUP_UNVERIFIED")
            return 2
    origin = args.public_origin
    if current.get("Enabled") is True and not origin:
        print("OAUTH_CONFIGURED: existing OAuth configuration retained; remote authorization not verified")
        return 0
    for _ in range(3):
        if not origin:
            out.write("Existing HTTPS tunnel origin (empty for later): ")
            out.flush()
            origin = inp.readline().rstrip("\r\n")
        if not origin:
            oauth_next(args, "OAUTH_ORIGIN_MISSING")
            return 0
        try:
            normalized = normalize_oauth_origin(origin)
        except InstallError:
            print("OAUTH_ORIGIN_INVALID: use an HTTPS DNS origin without path, port, userinfo, query or fragment")
            origin = None
            continue
        try:
            subprocess.run(identity + ["oauth-set-origin", "-project", args.project,
                                       "-public-origin", normalized],
                           stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=True)
        except (OSError, subprocess.SubprocessError):
            # The command might have committed before the reply failed.
            # Observe state instead of blindly repeating a write.
            pass
        current = command_json(status_cmd)
        if current.get("Enabled") is True and current.get("Origin") == normalized:
            print("OAUTH_CONFIGURED: remote reachability and actual client authorization not verified")
            print("MCP URL:", normalized + "/mcp")
            return 0
        oauth_next(args, "OAUTH_ORIGIN_APPLY_UNVERIFIED")
        return 2
    oauth_next(args, "OAUTH_ORIGIN_RETRY_LIMIT")
    return 2


def optional_oauth(identity, args):
    # Controller installation is independently complete.
    try:
        with open("/dev/tty", "r") as inp, open("/dev/tty", "w") as out:
            result = configure_oauth(identity, args, inp, out)
    except (OSError, ValueError, InstallError, subprocess.SubprocessError, KeyboardInterrupt):
        oauth_next(args, "OAUTH_CONFIGURATION_UNVERIFIED")
        result = 2
    if result:
        raise SystemExit(result)


def main():
    parser = argparse.ArgumentParser(description="BoxWeave v1 verified external install coordinator")
    parser.add_argument("--version", help="exact immutable v1.X.Y or v1.X.Y-rc.N Release")
    parser.add_argument("--local-binary", help="approved absolute path to pre-release candidate")
    parser.add_argument("--sha256", help="required 64-hex SHA256 for local candidate")
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--interactive", action="store_true", help="review and fresh install in one terminal")
    mode.add_argument("--plan", action="store_true", help="read-only plan (default)")
    mode.add_argument("--apply", action="store_true", help="apply on exact approved host")
    parser.add_argument("--guided-fresh-plan", action="store_true", help=argparse.SUPPRESS)
    parser.add_argument("--scoped-peer-fresh", action="store_true", help="optional assertion of the sole fresh Scoped contract; no legacy migration")
    parser.add_argument("--scoped-fresh-review-sha256", help="exact root-local native-scoped-fresh-plan review digest for --apply")
    parser.add_argument("--upgrade", action="store_true", help=argparse.SUPPRESS)
    parser.add_argument("--project", help="pre-provisioned restricted native Incus Project")
    parser.add_argument("--pool", help=argparse.SUPPRESS)
    parser.add_argument("--network", help=argparse.SUPPRESS)
    parser.add_argument("--acl", help=argparse.SUPPRESS)
    parser.add_argument("--public-origin", help="optional already-operated HTTPS origin (guided install)")
    parser.add_argument("--service-user", default="boxweave")
    parser.add_argument("--incus-socket", default="/var/lib/incus/unix.socket.user")
    parser.add_argument("--confirm-host-sha256", help="exact SHA256 of /etc/machine-id printed by --plan")
    args = parser.parse_args()
    if os.path.lexists("/var/lib/boxweave-offline-retirement-v1.json"):
        raise InstallError("OFFLINE_RETIREMENT_IN_PROGRESS: preserve original local retirement receipt")
    refuse_scoped_existing()
    if args.guided_fresh_plan or args.upgrade:
        raise InstallError("scoped fresh only: legacy installation and upgrade unsupported")
    if args.scoped_fresh_review_sha256 and not args.apply:
        raise InstallError("scoped review digest requires explicit --apply")
    if args.apply and not SHA_RE.fullmatch(args.scoped_fresh_review_sha256 or ""):
        raise InstallError("scoped --apply requires exact --scoped-fresh-review-sha256")
    if any((args.pool, args.network, args.acl)):
        raise InstallError("scoped fresh native scope is derived, never caller-selected")
    guided = args.interactive or (not args.plan and not args.apply and
                                 os.geteuid() == 0 and sys.stdout.isatty())
    if (guided or args.plan) and args.confirm_host_sha256:
        raise InstallError("interactive or plan-only mode does not accept supplied approval")
    if args.public_origin and not guided:
        raise InstallError("OAuth options require interactive fresh install")

    if platform.system() != "Linux":
        raise InstallError("Linux required")
    arch = {"x86_64": "amd64", "aarch64": "arm64"}.get(platform.machine())
    if not arch:
        raise InstallError("only Linux amd64/arm64 supported")
    host_digest = current_host_digest()
    if args.version and args.local_binary:
        raise InstallError("--version and --local-binary are mutually exclusive")
    if args.local_binary:
        if not args.sha256 or not SHA_RE.fullmatch(args.sha256):
            raise InstallError("local candidate requires exact --sha256")
        binary_path = Path(args.local_binary)
        if not binary_path.is_absolute():
            raise InstallError("--local-binary must be an absolute regular file; pass the full path to the actual candidate binary, not a relative path")
        if not binary_path.is_file() or binary_path.is_symlink():
            raise InstallError("--local-binary must be an absolute regular file; the candidate must exist and must not be a symlink")
        if binary_path.stat().st_size > MAX_BINARY:
            raise InstallError("local binary too large")
        binary = binary_path.read_bytes()
        sha = digest(binary)
        if sha != args.sha256:
            raise InstallError("local candidate SHA256 mismatch")
        version = "(local exact binary; verify build-info on apply)"
        source = "(read from binary build-info during apply)"
    else:
        if args.sha256:
            raise InstallError("--sha256 is only for --local-binary")
        tag = args.version
        if not tag:
            tag = get_bytes(POINTER_URL, 128).decode("ascii").strip()
            if tag == "UNPUBLISHED":
                raise InstallError("v1 stable pointer is UNPUBLISHED; use --version for a published RC or --local-binary")
            if "-rc." in tag:
                raise InstallError("stable v1 pointer may not select a prerelease")
        checked_tag(tag)
        binary, version, source, sha = select_release(tag, arch)

    if not args.service_user or not PROJECT_RE.fullmatch(args.service_user):
        raise InstallError("invalid service identity")
    if os.geteuid() != 0:
        raise InstallError("scoped native review requires local root")
    if args.incus_socket != RESTRICTED_SOCKET:
        raise InstallError("fresh review requires the canonical restricted socket")
    if args.apply and args.project is None:
        raise InstallError("scoped --apply requires explicit --project")
    uid = pwd.getpwnam(args.service_user).pw_uid
    args.project = args.project or ("user-" + str(uid))
    if uid == 0 or args.service_user != "boxweave" or args.project != "user-" + str(uid):
        raise InstallError("fresh install requires exact dedicated restricted identity/Project")
    if args.project is not None and not PROJECT_RE.fullmatch(args.project):
        raise InstallError("invalid project name")
    sock = Path(args.incus_socket)
    if not sock.is_absolute() or ".." in sock.parts or not str(sock).startswith("/var/lib/incus/"):
        raise InstallError("restricted Incus socket must be an approved absolute path")
    previous = Path("/var/lib/boxweave-uninstall-v1-audit.json")
    if os.path.lexists(previous):
        audit, _ = private_json(previous, 0)
        if audit.get("status") != "COMPLETE":
            raise InstallError("UNINSTALL_AUDIT_STARTED_OR_UNVERIFIED: preserve original audit; inspect matching uninstall --resume-plan, not a fresh install")
    if os.path.lexists(STATE_DIR / "install-manifest.json"):
        raise InstallError("existing installation; scoped fresh only, no adoption")

    print("BoxWeave v1 external distribution plan")
    print("host_machine_id_sha256=" + host_digest)
    print("arch=linux-" + arch)
    print("candidate_version=" + version)
    print("source_commit=" + source)
    print("binary_sha256=" + sha)
    print("project=" + str(args.project or "(not selected)"))
    print("socket=" + str(sock))
    print("service_user=" + args.service_user)
    print("operation=fresh install")
    print("changes=fresh-only exact scoped guard, quarantine ACL/profile, fixed systemd units, binary and SQLite; no legacy upgrade or tunnel")
    print("known_risk=Incus storage quotas and remote Guest image downloads require operator review; instance preparation is independent of service installation")
    print_lifecycle_preflight(scoped=True)
    print("forward_policy=current fresh installer defaults to all")
    print("forward_risk=explicit external Forward may expose guest services on public and future Host interfaces; default bind stays loopback")
    if not args.apply and not guided:
        with tempfile.TemporaryDirectory(prefix="boxweave-v1-scoped-review-") as folder:
            candidate = Path(folder) / "boxweave"
            candidate.write_bytes(binary)
            candidate.chmod(0o700)
            review = scoped_review(candidate, args.project)
            if current_host_digest() != host_digest:
                raise InstallError("Host changed during scoped review")
            print_scoped_review(review)
        print("PLAN_ONLY: no persistent host changes (review exact candidate and native digest; use --interactive or explicit --apply with --project, --confirm-host-sha256 and --scoped-fresh-review-sha256)")
        return
    if not guided and args.confirm_host_sha256 != host_digest:
        raise InstallError("--apply requires root and exact current --confirm-host-sha256")
    if args.project is None:
        raise InstallError("--apply requires explicit --project")
    if sock.is_symlink() or not sock.exists() or not stat.S_ISSOCK(sock.stat().st_mode):
        raise InstallError("restricted native Incus socket unavailable")
    user = pwd.getpwnam(args.service_user)
    if user.pw_uid == 0:
        raise InstallError("service identity must be non-root")
    if any(os.path.lexists(path) for path in ("/usr/local/bin/boxweave", "/etc/systemd/system/boxweave.service", "/var/lib/boxweave")):
        raise InstallError("existing installation detected: refuse adoption/co-installation")
    # The downloaded executable is public but Go installer verifies it under
    # the restricted service account. Use an accessible temporary directory.
    with tempfile.TemporaryDirectory(prefix="boxweave-v1-reviewed-") as folder:
        os.chmod(folder, 0o755)
        candidate = Path(folder) / "boxweave"
        candidate.write_bytes(binary)
        candidate.chmod(0o755)
        scoped = scoped_review(candidate, args.project)
        scoped_sha = scoped["ReviewSHA256"]
        print_scoped_review(scoped)
        if guided:
            with open("/dev/tty", "r") as inp, open("/dev/tty", "w") as out:
                out.write("Type INSTALL SCOPED " + args.project + " to approve this exact plan: ")
                out.flush()
                if inp.readline().strip() != "INSTALL SCOPED " + args.project:
                    raise InstallError("OPERATOR_APPROVAL_REQUIRED: scoped installation not approved; no installation changes")
        elif scoped_sha != args.scoped_fresh_review_sha256:
            raise InstallError("scoped fresh review digest changed")
        if current_host_digest() != host_digest or scoped_review(candidate, args.project) != scoped:
            raise InstallError("Host or scoped native plan changed after approval; review again")
        command = [
            str(candidate), "install-stage",
            "-project", args.project,
            "-service-user", args.service_user,
            "-incus-socket", str(sock),
            "-source-binary", str(candidate),
            "-expect-sha256", sha,
            "-approve-stage", "-scoped-fresh-review-sha256", scoped_sha,
        ]
        install_phase("STAGE", command)
        install_phase("ACTIVATE", [str(candidate), "install-activate", "-project", args.project,
                        "-service-user", args.service_user, "-incus-socket", str(sock), "-approve-local"])
        installed_status = install_phase("INSTALLED_STATUS", [str(candidate), "install-status", "-project", args.project,
                        "-service-user", args.service_user, "-incus-socket", str(sock)], capture=True)
        print(json.dumps(installed_status), flush=True)
        if (installed_status.get("binary_sha256") != sha or
                installed_status.get("native_project") != args.project or
                installed_status.get("service_user") != args.service_user or
                installed_status.get("scoped_peer_mode") != "SCOPED_PEER_FRESH_V2" or
                installed_status.get("service_active") is not True or
                installed_status.get("restart_required") is not False):
            raise phase_unverified("INSTALLED_STATUS")
        installed = "/usr/local/bin/boxweave"
        identity = ["runuser", "-u", args.service_user, "--", installed]
        doctor = doctor_postflight(service_command(args, Path(installed), "doctor", "-project", args.project))
        print("Doctor:", json.dumps(doctor), flush=True)
        if (doctor.get("version") != installed_status.get("version") or
                doctor.get("commit") != installed_status.get("build_commit") or
                (not args.local_binary and (doctor.get("version") != version or doctor.get("commit") != source)) or
                not (doctor.get("ready") and doctor.get("mcp_ready") and doctor.get("native", {}).get("ready"))):
            raise phase_unverified("CONTROLLER_DOCTOR")
        policy = doctor.get("diagnostic", {}).get("forward_policy", {})
        effective = policy.get("reason") if policy.get("state") == "OBSERVED" else "unverified"
        print("RUNNING_SERVICE_FORWARD_POLICY=" + str(effective))
        if guided:
            print("GUIDED_INSTALL_COMPLETE: native config ready; packet/quota/PRS acceptance remains separate", flush=True)
            args.oauth_password_configured = doctor.get("diagnostic", {}).get("ledger", {}).get("oauth", {}).get("password_configured")
            optional_oauth(identity, args)
    print("INSTALL_HELPER_APPLY_COMPLETE: inspect original installer manifest and native postflight; not HOST_GO PASS")


try:
    main()
except (InstallError, OSError, ValueError, KeyError, urllib.error.URLError, subprocess.SubprocessError, json.JSONDecodeError) as exc:
    print("boxweave-v1 installer: " + str(exc), file=sys.stderr)
    raise SystemExit(1)
PY_BOXWEAVE_INSTALL
