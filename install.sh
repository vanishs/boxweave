#!/bin/sh
# BoxWeave v1 distribution coordinator. Uses only Python 3 stdlib.
# No host mutation without --apply plus exact machine-id review.
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


def select_release(tag, arch, allow_prerelease):
    tag = checked_tag(tag)
    rc = "-rc." in tag
    if rc and not allow_prerelease:
        raise InstallError("prerelease install requires explicit --allow-prerelease")
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


def main():
    parser = argparse.ArgumentParser(description="BoxWeave v1 verified external install coordinator")
    parser.add_argument("--version", help="exact immutable v1.X.Y or v1.X.Y-rc.N Release")
    parser.add_argument("--allow-prerelease", action="store_true")
    parser.add_argument("--local-binary", help="approved absolute path to pre-release candidate")
    parser.add_argument("--sha256", help="required 64-hex SHA256 for local candidate")
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--plan", action="store_true", help="read-only plan (default)")
    mode.add_argument("--apply", action="store_true", help="apply on exact approved host")
    parser.add_argument("--upgrade", action="store_true", help="explicitly upgrade an existing verified install")
    parser.add_argument("--project", help="pre-provisioned restricted native Incus Project")
    parser.add_argument("--service-user", default="boxweave")
    parser.add_argument("--incus-socket", default="/var/lib/incus/unix.socket.user")
    parser.add_argument("--confirm-host-sha256", help="exact SHA256 of /etc/machine-id printed by --plan")
    args = parser.parse_args()

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
        binary, version, source, sha = select_release(tag, arch, args.allow_prerelease)

    if not args.service_user or not PROJECT_RE.fullmatch(args.service_user):
        raise InstallError("invalid service identity")
    if args.project is not None and not PROJECT_RE.fullmatch(args.project):
        raise InstallError("invalid project name")
    sock = Path(args.incus_socket)
    if not sock.is_absolute() or ".." in sock.parts or not str(sock).startswith("/var/lib/incus/"):
        raise InstallError("restricted Incus socket must be an approved absolute path")
    if args.upgrade and not args.apply:
        raise InstallError("--upgrade requires --apply")

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
    print("changes=only existing v1 Go installer-owned binary/unit/SQLite; never Incus setup or tunnel")
    print("known_risk=dir project quotas may be unenforced; physical host requires independent preflight")
    if not args.apply:
        print("PLAN_ONLY: no host changes (pass --apply, --project, --confirm-host-sha256 after approval)")
        return
    if os.geteuid() != 0 or args.confirm_host_sha256 != host_digest:
        raise InstallError("--apply requires root and exact current --confirm-host-sha256")
    if args.project is None:
        raise InstallError("--apply requires explicit --project")
    if sock.is_symlink() or not sock.exists() or not stat.S_ISSOCK(sock.stat().st_mode):
        raise InstallError("restricted native Incus socket unavailable")
    user = pwd.getpwnam(args.service_user)
    if user.pw_uid == 0:
        raise InstallError("service identity must be non-root")
    installed_manifest = Path("/var/lib/boxweave/install-manifest.json")
    installed = installed_manifest.exists()
    if installed and not args.upgrade:
        raise InstallError("existing v1 state; refuse implicit upgrade (use --upgrade after rollback review)")
    if not installed and args.upgrade:
        raise InstallError("no verified installation to upgrade")
    if not installed and (Path("/usr/local/bin/boxweave").exists() or Path("/etc/systemd/system/boxweave.service").exists() or Path("/var/lib/boxweave").exists()):
        raise InstallError("unmanaged or legacy main installation detected: refuse adoption/co-installation")
    # The downloaded executable is public but Go installer verifies it under
    # the restricted service account. Use an accessible temporary directory.
    with tempfile.TemporaryDirectory(prefix="boxweave-v1-reviewed-") as folder:
        os.chmod(folder, 0o755)
        candidate = Path(folder) / "boxweave"
        candidate.write_bytes(binary)
        candidate.chmod(0o755)
        command = [
            str(candidate), "upgrade" if args.upgrade else "install-stage",
            "-project", args.project,
            "-service-user", args.service_user,
            "-incus-socket", str(sock),
            "-source-binary", str(candidate),
            "-expect-sha256", sha,
            "-approve-local" if args.upgrade else "-approve-stage",
        ]
        # Only the dedicated Go executable is permitted to own installer
        # writes, backup, service health, and rollback.
        subprocess.run(command, check=True)
        if not args.upgrade:
            subprocess.run([str(candidate), "install-activate", "-project", args.project,
                            "-service-user", args.service_user, "-incus-socket", str(sock),
                            "-approve-local"], check=True)
        subprocess.run([str(candidate), "install-status", "-project", args.project,
                        "-service-user", args.service_user, "-incus-socket", str(sock)], check=True)
    print("INSTALL_HELPER_APPLY_COMPLETE: inspect original installer manifest and native postflight; not HOST_GO PASS")


try:
    main()
except (InstallError, OSError, ValueError, KeyError, urllib.error.URLError, subprocess.CalledProcessError, json.JSONDecodeError) as exc:
    print("boxweave-v1 installer: " + str(exc), file=sys.stderr)
    raise SystemExit(1)
PY_BOXWEAVE_INSTALL
