#!/bin/sh
# BoxWeave v1 distribution coordinator. Uses only Python 3 stdlib.
# Host writes require explicit root approval: reviewed --apply or terminal consent.
set -eu
exec python3 - "$@" <<'PY_BOXWEAVE_INSTALL'
import argparse
import hashlib
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


def file_digest(path, limit):
    with os.fdopen(os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK), "rb") as f:
        info = os.fstat(f.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o022:
            raise InstallError("unsafe installed file: " + path.name)
        if info.st_size > limit:
            raise InstallError("installed file exceeds size limit")
        data = f.read(limit + 1)
    if len(data) > limit:
        raise InstallError("installed file exceeds size limit")
    return digest(data)


def identity_args(args):
    return ["-project", args.project, "-service-user", args.service_user,
            "-incus-socket", args.incus_socket]


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


def print_scoped_review(review):
    print("Reviewed scoped native preparation:", json.dumps(review, sort_keys=True), flush=True)
    print("scoped_fresh_review_sha256=" + review["ReviewSHA256"], flush=True)


def guided_upgrade_review(args):
    """Inspect the canonical existing install, without opening its SQLite ledger."""
    if os.geteuid() != 0:
        raise InstallError("upgrade review requires root (use sudo --plan --upgrade)")
    uid = pwd.getpwnam("boxweave").pw_uid
    if uid == 0:
        raise InstallError("service identity must be non-root")
    expected = {"schema": 2, "service_user": "boxweave", "native_project": "user-" + str(uid),
                "native_socket": RESTRICTED_SOCKET, "state_dir": str(STATE_DIR),
                "binary_path": str(INSTALLED_BINARY)}
    manifest, manifest_sha = private_json(STATE_DIR / "install-manifest.json", uid)
    if manifest.get("scoped_peer_mode") not in (None, ""):
        raise InstallError("scoped manifest: preserve original, no legacy upgrade")
    if any(manifest.get(k) != v for k, v in expected.items()):
        raise InstallError("not a canonical verified v1 installation; refuse adoption")
    if args.project is not None and args.project != expected["native_project"]:
        raise InstallError("selected project conflicts with installed identity")
    if args.service_user != expected["service_user"] or args.incus_socket != expected["native_socket"]:
        raise InstallError("selected service or socket conflicts with installed identity")
    args.project = expected["native_project"]
    sock = Path(args.incus_socket)
    if sock.is_symlink() or not stat.S_ISSOCK(sock.stat().st_mode):
        raise InstallError("restricted native Incus socket unavailable")
    if (not TAG_RE.fullmatch("v" + str(manifest.get("version", ""))) or
            not COMMIT_RE.fullmatch(str(manifest.get("build_commit", "")))):
        raise InstallError("guided upgrade requires a published v1 installed build; use explicit reviewed apply for development installs")
    if (file_digest(INSTALLED_BINARY, MAX_BINARY) != manifest.get("binary_sha256") or
            file_digest(INSTALLED_UNIT, MAX_METADATA) != manifest.get("unit_sha256")):
        raise InstallError("installed binary/unit does not match its manifest")
    current = command_json([str(INSTALLED_BINARY), "install-status", *identity_args(args)])
    if (any(current.get(k) != v for k, v in manifest.items() if k != "service_active") or
            current.get("service_active") is not True or current.get("restart_required") is not False):
        raise InstallError("installed service is inactive, unsettled or changed; inspect original install")
    previous = None
    if os.path.lexists(BACKUP_DIR):
        info = BACKUP_DIR.lstat()
        if not stat.S_ISDIR(info.st_mode) or stat.S_IMODE(info.st_mode) != 0o700 or info.st_uid != 0:
            raise InstallError("unsafe installer backup directory")
        receipt, receipt_sha = private_json(BACKUP_DIR / "receipt.json", 0)
        old = receipt.get("Previous")
        if (receipt.get("Schema") != 1 or not isinstance(old, dict) or
                any(old.get(k) != v for k, v in expected.items()) or
                not SHA_RE.fullmatch(str(old.get("binary_sha256", ""))) or
                not COMMIT_RE.fullmatch(str(old.get("build_commit", ""))) or
                not re.fullmatch(r"[0-9A-Za-z.+-]{1,80}", str(old.get("version", "")))):
            raise InstallError("previous recovery receipt does not match this installation")
        previous = {"version": old["version"], "binary_sha256": old["binary_sha256"],
                    "build_commit": old["build_commit"], "receipt_sha256": receipt_sha}
    return {"current": current, "manifest_sha256": manifest_sha, "previous": previous}


def release_order(version):
    match = re.fullmatch(r"1\.([0-9]+)\.([0-9]+)(?:-rc\.([1-9][0-9]*))?", version)
    if not match:
        raise InstallError("invalid published upgrade version")
    minor, patch, rc = match.groups()
    return int(minor), int(patch), rc is None, int(rc or 0)


def print_upgrade_review(review, sha):
    current, previous = review["current"], review["previous"]
    print("installed_version=" + current["version"])
    print("installed_commit=" + current["build_commit"])
    print("installed_binary_sha256=" + current["binary_sha256"])
    print("installed_unit_sha256=" + current["unit_sha256"])
    print("installed_service_active=true")
    print("upgrade_change=" + ("already current" if current["binary_sha256"] == sha else "replace and restart controller"))
    if previous is None:
        print("previous_backup=none")
    else:
        print("previous_backup_version=" + previous["version"])
        print("previous_backup_receipt_sha256=" + previous["receipt_sha256"])
        print("previous_backup_policy=" + ("retain unchanged" if current["binary_sha256"] == sha else
              "approval closes this older recovery window using verified install-commit"))
    print("new_backup_policy=retain current installation for rollback after a successful upgrade")
    print("native_policy=preserve existing Incus, service unit/overrides, OAuth and tunnel")
    if current["binary_sha256"] != sha and previous:
        print("WARNING: approval permanently closes the older backup before upgrading; if upgrade later fails, that older rollback is no longer available.")
    print("Stop other maintenance and mutating MCP work during upgrade; rollback cannot undo native effects.", flush=True)


def require_unchanged_upgrade(args, reviewed, host_digest):
    if current_host_digest() != host_digest or guided_upgrade_review(args) != reviewed:
        raise InstallError("host/install/backup changed after review; run a new plan before approval")


def service_health(args, binary, expected):
    status = command_json(service_command(args, binary, "status",
                          "-db", str(STATE_DIR / "ledger.sqlite"), "-project", args.project,
                          "-incus-socket", args.incus_socket))
    if (status.get("version") != expected["version"] or
            status.get("commit") != expected["build_commit"] or
            status.get("mcp_ready") is not True or status.get("native", {}).get("service_ready") is not True):
        raise InstallError("restricted service preflight/postflight not ready or identity changed")
    return status


def apply_guided_upgrade(args, candidate, sha, version, source, reviewed, host_digest):
    original = reviewed["current"]
    require_unchanged_upgrade(args, reviewed, host_digest)
    if original["binary_sha256"] == sha:
        print("UPGRADE_ALREADY_CURRENT: existing backup retained; no service restart")
        return
    with open("/dev/tty", "r") as inp, open("/dev/tty", "w") as out:
        out.write("Type UPGRADE " + args.project + " to approve this exact plan: ")
        out.flush()
        if inp.readline().strip() != "UPGRADE " + args.project:
            raise InstallError("upgrade not approved; no installer changes")
    require_unchanged_upgrade(args, reviewed, host_digest)
    # The explicit candidate hash was reviewed; execute build-info only as
    # the existing restricted service identity, never as root.
    build = command_json(service_command(args, candidate, "build-info"))
    if (not re.fullmatch(r"[0-9A-Za-z.+-]{1,80}", str(build.get("version", ""))) or
            not COMMIT_RE.fullmatch(str(build.get("commit", ""))) or
            (not args.local_binary and (build.get("version") != version or build.get("commit") != source))):
        raise InstallError("candidate build identity differs from reviewed release")
    # Health checks that open SQLite run only after terminal approval.
    service_health(args, INSTALLED_BINARY, original)
    # Never close a previous recovery window for an incompatible candidate.
    # In particular, pre-stable schema 6 refuses the older schema explicitly.
    service_health(args, candidate, {"version": build["version"], "build_commit": build["commit"]})
    require_unchanged_upgrade(args, reviewed, host_digest)
    if reviewed["previous"]:
        # Delegate all backup verification/deletion to the installed Go
        # implementation. Each Go mutation retains its maintenance lock.
        subprocess.run([str(INSTALLED_BINARY), "install-commit", *identity_args(args),
                        "-approve-local"], check=True)
        expected = dict(reviewed, previous=None)
        require_unchanged_upgrade(args, expected, host_digest)
        print("PREVIOUS_BACKUP_COMMITTED", flush=True)
    print("UPGRADE_START: on any failure inspect the original manifest/backup; never retry or rollback blindly.", flush=True)
    subprocess.run([str(candidate), "upgrade", *identity_args(args),
                    "-source-binary", str(candidate), "-expect-sha256", sha,
                    "-approve-local"], check=True)
    # Local dev candidates can be used in an explicitly approved upgrade,
    # but the resulting build must still match the reviewed executable.
    current = command_json([str(INSTALLED_BINARY), "install-status", *identity_args(args)])
    if (current.get("binary_sha256") != sha or current.get("version") != build["version"] or
            current.get("build_commit") != build["commit"] or current.get("unit_sha256") != original["unit_sha256"] or
            current.get("service_active") is not True or current.get("restart_required") is not False or
            any(current.get(k) != original.get(k) for k in ("schema", "native_project", "native_socket", "service_user", "state_dir", "binary_path", "retirement_unit_sha256", "retirement_bridge", "retirement_acl", "retirement_native_review_sha256"))):
        raise InstallError("upgrade postflight mismatch; retain rollback backup and inspect")
    backup, _ = private_json(BACKUP_DIR / "receipt.json", 0)
    if (backup.get("Schema") != 1 or
            any(backup.get("Previous", {}).get(k) != v for k, v in original.items())):
        raise InstallError("new rollback backup does not bind the reviewed previous installation")
    health = service_health(args, INSTALLED_BINARY, current)
    print("Installed:", json.dumps(current), flush=True)
    print("Service status:", json.dumps(health), flush=True)
    print("GUIDED_UPGRADE_COMPLETE: service ready; previous version backup retained; not HOST_GO PASS")


def legacy_fresh_review(candidate, args, native_args):
    retirement_reader_preflight()
    plan = subprocess.check_output([str(candidate), "native-prepare-plan", *native_args],
                                   text=True, timeout=90)
    review = json.loads(plan)
    expected = dict(project=args.project, pool=args.pool, network=args.network, acl=args.acl)
    if review.get("target") != expected or not SHA_RE.fullmatch(review.get("review_sha256", "")):
        raise InstallError("native plan identity missing or mismatched")
    return plan, review


def install_phase(name, command, capture=False):
    print("INSTALL_PHASE=" + name, flush=True)
    try:
        if capture:
            return command_json(command)
        subprocess.run(command, check=True)
    except (OSError, ValueError, subprocess.SubprocessError) as exc:
        raise InstallError("INSTALL_PHASE_UNVERIFIED: " + name +
                           "; retain original files and journals; inspect install-status and doctor locally; do not reinstall blindly") from exc


def optional_oauth(identity, args):
    # The controller is already installed. OAuth is separate, optional
    # configuration, never a reason to restart or reinstall the controller.
    try:
        with open("/dev/tty", "r") as inp, open("/dev/tty", "w") as out:
            out.write("Configure a new OAuth approval password? [y/N] ")
            out.flush()
            if inp.readline().strip().lower() != "y":
                print("OAUTH_CONFIGURATION_DEFERRED: local MCP installation remains complete; use oauth-status and the operator guide")
                return
            subprocess.run(identity + ["oauth-setup", "-project", args.project],
                           stdin=inp, stdout=out, stderr=out, check=True)
            origin = args.public_origin
            if not origin:
                out.write("Existing HTTPS tunnel origin (empty for later): ")
                out.flush()
                origin = inp.readline().strip()
            if origin:
                subprocess.run(identity + ["oauth-set-origin", "-project", args.project,
                                           "-public-origin", origin], check=True)
            current = command_json(identity + ["oauth-status", "-project", args.project])
            if current.get("Enabled") is True and current.get("Origin") == origin and origin:
                print("OAUTH_CONFIGURED: remote reachability and actual client authorization not verified")
                print("MCP URL:", origin + "/mcp")
            elif not origin and current.get("Enabled") is False and current.get("Origin") == "":
                print("OAUTH_ORIGIN_MISSING: password setup succeeded; configure the HTTPS origin later")
            else:
                raise InstallError("OAuth status differs from requested configuration")
    except (OSError, ValueError, InstallError, subprocess.SubprocessError):
        # Never print a command exception containing configuration inputs.
        print("OAUTH_CONFIGURATION_UNVERIFIED: controller installation complete; inspect oauth-status and finish configuration separately", file=sys.stderr)
        raise SystemExit(2)


def main():
    parser = argparse.ArgumentParser(description="BoxWeave v1 verified external install coordinator")
    parser.add_argument("--version", help="exact immutable v1.X.Y or v1.X.Y-rc.N Release")
    parser.add_argument("--local-binary", help="approved absolute path to pre-release candidate")
    parser.add_argument("--sha256", help="required 64-hex SHA256 for local candidate")
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--interactive", action="store_true", help="review and install/upgrade in one terminal")
    mode.add_argument("--plan", action="store_true", help="read-only plan (default)")
    mode.add_argument("--apply", action="store_true", help="apply on exact approved host")
    parser.add_argument("--guided-fresh-plan", action="store_true", help="with --plan: full root-local Legacy fresh preview, no approval or writes")
    parser.add_argument("--scoped-peer-fresh", action="store_true", help="fresh-only scoped Peer v2; no legacy migration")
    parser.add_argument("--scoped-fresh-review-sha256", help="exact root-local native-scoped-fresh-plan review digest for --apply")
    parser.add_argument("--upgrade", action="store_true", help="explicitly upgrade an existing verified install")
    parser.add_argument("--project", help="pre-provisioned restricted native Incus Project")
    parser.add_argument("--pool", help="existing Incus storage pool (guided install)")
    parser.add_argument("--network", help="existing managed Incus bridge (guided install)")
    parser.add_argument("--acl", help="exclusive new ACL (guided install)")
    parser.add_argument("--public-origin", help="optional already-operated HTTPS origin (guided install)")
    parser.add_argument("--service-user", default="boxweave")
    parser.add_argument("--incus-socket", default="/var/lib/incus/unix.socket.user")
    parser.add_argument("--confirm-host-sha256", help="exact SHA256 of /etc/machine-id printed by --plan")
    args = parser.parse_args()
    refuse_scoped_existing()
    if args.guided_fresh_plan and (not args.plan or args.upgrade or args.scoped_peer_fresh or args.public_origin):
        raise InstallError("--guided-fresh-plan requires --plan, Legacy fresh mode and no OAuth options")
    if args.scoped_peer_fresh and args.upgrade:
        raise InstallError("scoped Peer requires a new installation; upgrade refused before backup or health changes")
    if args.scoped_fresh_review_sha256 and (not args.scoped_peer_fresh or not args.apply):
        raise InstallError("scoped review digest requires explicit scoped --apply")
    if args.scoped_peer_fresh and args.apply and not SHA_RE.fullmatch(args.scoped_fresh_review_sha256 or ""):
        raise InstallError("scoped --apply requires exact --scoped-fresh-review-sha256")
    if args.scoped_peer_fresh and any((args.pool,args.network,args.acl)):
        raise InstallError("scoped fresh native scope is derived, never caller-selected")
    # Source is a heredoc: use the controlling terminal for human consent.
    guided = args.interactive or (not args.plan and not args.apply and
                                 os.geteuid() == 0 and sys.stdout.isatty())
    if (guided or args.plan) and args.confirm_host_sha256:
        raise InstallError("interactive or plan-only mode does not accept supplied approval")
    guided_upgrade = args.upgrade and guided
    fresh_guided = guided and not args.upgrade
    legacy_preview = args.guided_fresh_plan
    if not (fresh_guided or legacy_preview) and any((args.pool, args.network, args.acl, args.public_origin)):
        raise InstallError("native/OAuth options require interactive fresh install")
    if args.upgrade and not (guided or args.apply or args.plan):
        raise InstallError("noninteractive upgrade requires --plan or explicit --apply")
    upgrade_review = None
    if args.upgrade and (guided_upgrade or args.plan):
        upgrade_review = guided_upgrade_review(args)

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
        if not binary_path.is_absolute() or not binary_path.is_file() or binary_path.is_symlink():
            raise InstallError("--local-binary must be an absolute regular file")
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
    if fresh_guided or legacy_preview or args.scoped_peer_fresh:
        if os.geteuid() != 0:
            raise InstallError("interactive or scoped native review requires local root")
        if (args.scoped_peer_fresh or legacy_preview or fresh_guided) and args.incus_socket != RESTRICTED_SOCKET:
            raise InstallError("guided fresh review requires the canonical restricted socket")
        if args.scoped_peer_fresh and args.apply and args.project is None:
            raise InstallError("scoped --apply requires explicit --project")
        uid = pwd.getpwnam(args.service_user).pw_uid
        args.project = args.project or ("user-" + str(uid))
        if uid == 0 or args.service_user != "boxweave" or args.project != "user-" + str(uid):
            raise InstallError("guided install requires exact dedicated restricted identity/Project")
        if not args.scoped_peer_fresh:
            args.pool = args.pool or "default"
            args.network = args.network or ("incusbr-" + str(uid))
            args.acl = args.acl or ("bw-" + args.project + "-acl")
            if any(not PROJECT_RE.fullmatch(x) for x in (args.pool, args.network, args.acl)):
                raise InstallError("invalid guided native target")
    if args.project is not None and not PROJECT_RE.fullmatch(args.project):
        raise InstallError("invalid project name")
    sock = Path(args.incus_socket)
    if not sock.is_absolute() or ".." in sock.parts or not str(sock).startswith("/var/lib/incus/"):
        raise InstallError("restricted Incus socket must be an approved absolute path")
    if upgrade_review and not args.local_binary:
        current = upgrade_review["current"]
        if release_order(version) < release_order(current["version"]) or (version == current["version"] and sha != current["binary_sha256"]):
            raise InstallError("refuse downgrade or changed bytes under installed version")
    if not args.upgrade and os.geteuid() == 0 and (fresh_guided or legacy_preview or args.scoped_peer_fresh or args.apply):
        previous = Path("/var/lib/boxweave-uninstall-v1-audit.json")
        if os.path.lexists(previous):
            audit, _ = private_json(previous, 0)
            if audit.get("status") != "COMPLETE":
                raise InstallError("UNINSTALL_AUDIT_STARTED_OR_UNVERIFIED: preserve original audit; inspect matching uninstall --resume-plan, not a fresh install")
    if args.plan and not args.upgrade and os.path.lexists(STATE_DIR / "install-manifest.json"):
        raise InstallError("existing installation; use sudo sh install.sh --plan --upgrade --version TAG")

    print("BoxWeave v1 external distribution plan")
    print("host_machine_id_sha256=" + host_digest)
    print("arch=linux-" + arch)
    print("candidate_version=" + version)
    print("source_commit=" + source)
    print("binary_sha256=" + sha)
    print("project=" + str(args.project or "(not selected)"))
    print("socket=" + str(sock))
    print("service_user=" + args.service_user)
    print("operation=" + ("upgrade" if args.upgrade else "fresh install"))
    if args.scoped_peer_fresh:
        print("changes=fresh-only exact scoped guard, quarantine ACL/profile, fixed systemd units, binary and SQLite; no legacy upgrade or tunnel")
    elif fresh_guided or legacy_preview:
        print("changes=reviewed fresh Legacy Project/Profile/ACL, retirement verifier, binary/unit/SQLite; no shared Incus bootstrap or tunnel")
    else:
        print("changes=only v1 Go installer-owned binary/unit/SQLite; no native preparation or tunnel")
    print("known_risk=Incus storage quotas and remote Guest image downloads require operator review; instance preparation is independent of service installation")
    if args.upgrade:
        print("forward_policy=preserve existing systemd unit and operator overrides; native Project permissions remain unchanged")
        if upgrade_review:
            print_upgrade_review(upgrade_review, sha)
    else:
        print("forward_policy=current fresh installer defaults to all; older candidate binaries keep their original defaults")
        print("forward_risk=explicit external Forward may expose guest services on public and future Host interfaces; default bind stays loopback; reviewed fresh native preparation explicitly grants Project proxy permission; service-only installation does not")
    if not args.apply and not guided and not legacy_preview:
        if args.scoped_peer_fresh:
            with tempfile.TemporaryDirectory(prefix="boxweave-v1-scoped-review-") as folder:
                candidate = Path(folder) / "boxweave"
                candidate.write_bytes(binary)
                candidate.chmod(0o700)
                review = scoped_review(candidate, args.project)
                if current_host_digest() != host_digest:
                    raise InstallError("Host changed during scoped review")
                print_scoped_review(review)
            print("PLAN_ONLY: no persistent host changes (review exact candidate and native digest; use --interactive or explicit --apply with --project, --confirm-host-sha256 and --scoped-fresh-review-sha256)")
        elif args.upgrade:
            print("PLAN_ONLY: no host changes (run sudo sh install.sh --upgrade --version TAG for terminal approval)")
        else:
            print("PLAN_ONLY: no host changes (pass --apply, --project, --confirm-host-sha256 after approval)")
        return
    if os.geteuid() != 0 or (not guided and not legacy_preview and args.confirm_host_sha256 != host_digest):
        raise InstallError("--apply requires root and exact current --confirm-host-sha256")
    if args.project is None:
        raise InstallError("--apply requires explicit --project")
    if sock.is_symlink() or not sock.exists() or not stat.S_ISSOCK(sock.stat().st_mode):
        raise InstallError("restricted native Incus socket unavailable")
    user = pwd.getpwnam(args.service_user)
    if user.pw_uid == 0:
        raise InstallError("service identity must be non-root")
    installed_manifest = Path("/var/lib/boxweave/install-manifest.json")
    installed = os.path.lexists(installed_manifest)
    if installed and not args.upgrade:
        raise InstallError("existing v1 state; refuse implicit upgrade (use --upgrade after rollback review)")
    if not installed and args.upgrade:
        raise InstallError("no verified installation to upgrade")
    if not installed and any(os.path.lexists(path) for path in ("/usr/local/bin/boxweave", "/etc/systemd/system/boxweave.service", "/var/lib/boxweave")):
        raise InstallError("unmanaged or legacy main installation detected: refuse adoption/co-installation")
    if (fresh_guided or legacy_preview) and os.path.lexists("/var/lib/boxweave-native-prereq-v1.json"):
        raise InstallError("existing Native journal; review and reconcile before guided install")
    # The downloaded executable is public but Go installer verifies it under
    # the restricted service account. Use an accessible temporary directory.
    with tempfile.TemporaryDirectory(prefix="boxweave-v1-reviewed-") as folder:
        os.chmod(folder, 0o755)
        candidate = Path(folder) / "boxweave"
        candidate.write_bytes(binary)
        candidate.chmod(0o755)
        if guided_upgrade:
            apply_guided_upgrade(args, candidate, sha, version, source, upgrade_review, host_digest)
            return
        native_args = ["-project", args.project, "-pool", args.pool,
                       "-network", args.network, "-acl", args.acl] if fresh_guided or legacy_preview else []
        if (fresh_guided or legacy_preview) and not args.scoped_peer_fresh:
            plan, review = legacy_fresh_review(candidate, args, native_args)
            print("Reviewed native preparation:", plan, flush=True)
            print("Will install, prepare Native, stage the fixed-Project IPv4 retirement verifier, verify and check doctor; no shared Incus rebuild.", flush=True)
            if legacy_preview:
                if current_host_digest() != host_digest:
                    raise InstallError("host changed during guided preview")
                print("GUIDED_FRESH_PLAN_ONLY: no persistent host changes; use --interactive for fresh review and approval")
                return
            with open("/dev/tty", "r") as inp, open("/dev/tty", "w") as out:
                out.write("Type INSTALL " + args.project + " to approve this exact plan: ")
                out.flush()
                if inp.readline().strip() != "INSTALL " + args.project:
                    raise InstallError("OPERATOR_APPROVAL_REQUIRED: installation not approved; no installation changes")
            if current_host_digest() != host_digest:
                raise InstallError("host changed after approval")
            fresh_plan, _ = legacy_fresh_review(candidate, args, native_args)
            if fresh_plan != plan:
                raise InstallError("native plan changed after approval; review again")
        if args.scoped_peer_fresh:
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
            str(candidate), "upgrade" if args.upgrade else "install-stage",
            "-project", args.project,
            "-service-user", args.service_user,
            "-incus-socket", str(sock),
            "-source-binary", str(candidate),
            "-expect-sha256", sha,
            "-approve-local" if args.upgrade else "-approve-stage",
        ]
        if args.scoped_peer_fresh:
            command += ["-scoped-peer-fresh", "-scoped-fresh-review-sha256", scoped_sha]
        elif fresh_guided:
            command += ["-retirement-review", "-retirement-compact", "-pool", args.pool,
                        "-network", args.network, "-acl", args.acl,
                        "-native-prepare-sha256", review["review_sha256"]]
        # Only the dedicated Go executable is permitted to own installer
        # writes, backup, service health, and rollback.
        install_phase("UPGRADE" if args.upgrade else "STAGE", command)
        if fresh_guided and not args.scoped_peer_fresh:
            prepared = install_phase("NATIVE_PREPARE", [str(candidate), "native-prepare-apply", *native_args,
                 "-native-prepare-sha256", review["review_sha256"], "-approve-local"], capture=True)
            if prepared.get("status") != "APPLIED":
                raise InstallError("native preparation incomplete; inspect original journal")
            prereq = install_phase("NATIVE_READBACK", [str(candidate), "native-prereq-plan", "-project", args.project], capture=True)
            if not prereq.get("config_ready") or not SHA_RE.fullmatch(prereq.get("review_sha256", "")):
                raise InstallError("native configuration not ready; preserve staged installation")
            print("Native prerequisite readback:", json.dumps(prereq), flush=True)
            install_phase("NATIVE_VERIFY", [str(candidate), "native-prereq-verify", "-project", args.project,
                            "-native-prereq-sha256", prereq["review_sha256"], "-approve-local"])
        if not args.upgrade:
            install_phase("ACTIVATE", [str(candidate), "install-activate", "-project", args.project,
                            "-service-user", args.service_user, "-incus-socket", str(sock), "-approve-local"])
        install_phase("INSTALLED_STATUS", [str(candidate), "install-status", "-project", args.project,
                        "-service-user", args.service_user, "-incus-socket", str(sock)])
        if fresh_guided:
            installed = "/usr/local/bin/boxweave"
            identity = ["runuser", "-u", args.service_user, "--", installed]
            doctor = json.loads(subprocess.check_output(identity + ["doctor", "-project", args.project], text=True))
            print("Doctor:", json.dumps(doctor), flush=True)
            if not args.local_binary and (doctor.get("version") != version or doctor.get("commit") != source):
                raise InstallError("installed doctor build identity differs from reviewed release")
            if not (doctor.get("ready") and doctor.get("mcp_ready") and doctor.get("native", {}).get("ready")):
                raise InstallError("doctor not ready; inspect installed service")
            policy = doctor.get("diagnostic", {}).get("forward_policy", {})
            effective = policy.get("reason") if policy.get("state") == "OBSERVED" else "unverified"
            print("RUNNING_SERVICE_FORWARD_POLICY=" + str(effective))
            print("GUIDED_INSTALL_COMPLETE: native config ready; packet/quota/PRS acceptance remains separate", flush=True)
            optional_oauth(identity, args)
    print("INSTALL_HELPER_APPLY_COMPLETE: inspect original installer manifest and native postflight; not HOST_GO PASS")


try:
    main()
except (InstallError, OSError, ValueError, KeyError, urllib.error.URLError, subprocess.SubprocessError, json.JSONDecodeError) as exc:
    print("boxweave-v1 installer: " + str(exc), file=sys.stderr)
    raise SystemExit(1)
PY_BOXWEAVE_INSTALL
