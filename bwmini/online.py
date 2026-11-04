"""Source-bound BoxWeave install/update/repair for fresh or reviewed existing Providers.
Existing Containarium is upgraded to the declared minimum when needed. Credentials stay private.
"""
import argparse, base64, datetime, fcntl, hashlib, importlib.util, ipaddress, json, os
from pathlib import Path
import platform, pwd, re, secrets, shutil, socket, stat, subprocess, sys, tempfile, time, urllib.request, urllib.error

STATE = Path("/var/lib/boxweave-default")
PUBLIC = Path("/usr/local/lib/boxweave/default")
APP = "boxweave"
# Minimum runtime version; there is no maximum or per-release allowlist.
MIN_PROVIDER_VERSION = "0.128.0"
PROVIDER_VERSION = MIN_PROVIDER_VERSION  # Default baseline for fresh bootstrap.
INCUS_PACKAGES = ("incus", "incus-client", "incus-extra")
KEY_SHA256 = "f1adb44fdf82083f983a392b5b406d34e473dce3d66a51de3118cdc1f51d3f87"
# These hashes authenticate only the initial v0.128.0 clean-host download,
# NOT a compatibility whitelist for already installed upstream binaries.
PROVIDER_ASSETS = {
    "containariumd": "7c19f21b07d8e988ebbfd39243e22cf97d97d9595598b9860f5bc4123e41e09f",
    "containarium": "74560e96c4216cfb83218499682c816a9db78e4f367a4c194a587d650b83f696",
    "mcp-server": "d1c645843ef2989fc5bff53ec57b6612ff931e3b94c9f42d040ee8d7c065b0ba",
}
RELEASE_FILES = ("boxweave.zip", "bootstrap.py", "distribution.py", "online.py", "hostkeys.py", "mcp_gate.py", "resource_size.py", "gate_diagnostics.py","mcp_compat.py","restore_guard.py","operation_coordination.py", "coordination_runtime.py", "coordination_bridge.py", "box_reads.py", "box_patch.py", "gateway_tasks.py", "gateway_protection.py", "nesting.py", "client_setup.py", "client_profiles.py", "host_cli.py", "upstream_version.py")
SECRET = Path("/etc/boxweave-provider/jwt.secret")
UNIT_DIR = Path("/etc/systemd/system")
# The upstream main at e5deb167 and v0.120.1 contain the identical wrapper.
# Source is fetched from immutable upstream revision, NEVER from a BoxWeave Release.
# This does not authorize a Containarium binary upgrade.
HOST_SHELL = Path("/usr/local/bin/containarium-shell")
HOST_SHELLS = Path("/etc/shells")
HOST_SHELL_SHA256 = "2212651e2083b80d76363523bf794ec4d530f8eb74bbac0def0c347bb544caa6"
HOST_SHELL_UPSTREAM_COMMIT = "e5deb167c973033d9a847abdcb2420e917095546"
HOST_SHELL_SOURCE_SHA256 = "93365c56cd70aafac9880bc4dfcc4b1bcbb55dfc86e37b17b207c8245989f04a"
HOST_SHELL_UPSTREAM_URL = ("https://raw.githubusercontent.com/FootprintAI/Containarium/"
                           + HOST_SHELL_UPSTREAM_COMMIT + "/scripts/setup-ssh-container-proxy.sh")


class Failure(Exception): pass


def supported_python(version):
    return tuple(version[:2]) >= (3, 12)


def host_package_preflight(external, system):
    """Check missing host capabilities rather than a Linux distro name.

    Fail before writing managed state if APT is needed but unavailable.
    A separately prepared Provider does not need the host package manager.
    """
    missing_venv = importlib.util.find_spec("ensurepip") is None
    missing_incus = not external and not shutil.which("incus")
    missing_postgres = not external and not any(PG_BINARY_ROOT.glob("*/bin/postgres"))
    if missing_venv or missing_incus or missing_postgres:
        require(bool(shutil.which("apt-get")), "APT_GET_REQUIRED_FOR_MISSING_HOST_DEPENDENCIES")
    if missing_incus:
        codename = system.get("VERSION_CODENAME", "")
        require(isinstance(codename, str) and re.fullmatch(r"[a-z]{4,32}", codename),
                "APT_REPOSITORY_SUITE_REQUIRED_FOR_INCUS")
    return {"apt_needed": bool(missing_venv or missing_incus or missing_postgres),
            "missing_venv": missing_venv, "missing_incus": bool(missing_incus),
            "missing_postgres": bool(missing_postgres)}


def require(value, code):
    if not value: raise Failure(code)


def sha(data): return hashlib.sha256(data).hexdigest()


def atomic(path, data, mode=0o600, uid=0, gid=0):
    fd, name = tempfile.mkstemp(prefix=".boxweave-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as out:
            os.fchmod(out.fileno(), mode); os.fchown(out.fileno(), uid, gid)
            out.write(data); out.flush(); os.fsync(out.fileno())
        os.replace(name, path)
    finally:
        if os.path.exists(name): os.unlink(name)


def directory(path, mode=0o700):
    require(not path.is_symlink(), "SYMLINK_DIRECTORY_REFUSED")
    path.mkdir(parents=True, exist_ok=True, mode=mode)
    require(path.stat().st_uid == 0, "ROOT_DIRECTORY_REQUIRED")
    path.chmod(mode)


def download(url, target, expected, limit=180_000_000):
    require(url.startswith("https://"), "HTTPS_REQUIRED")
    require(re.fullmatch("[a-f0-9]{64}", expected) is not None, "EXPECTED_SHA256_REQUIRED")
    if target.exists():
        require(not target.is_symlink() and sha(target.read_bytes()) == expected, "CACHED_ARTIFACT_CHANGED")
        return
    request = urllib.request.Request(url, headers={"User-Agent": "BoxWeave-installer"})
    with urllib.request.urlopen(request, timeout=60) as response:
        payload = response.read(limit+1)
    require(len(payload) <= limit and sha(payload) == expected, "DOWNLOAD_DIGEST_MISMATCH")
    atomic(target, payload)


def run(argv, deadline, log=None, env=None, check=True):
    try:
        result = subprocess.run(argv, stdin=subprocess.DEVNULL, stdout=log or subprocess.DEVNULL,
                                stderr=log or subprocess.DEVNULL, env=env,
                                timeout=max(.01, deadline-time.monotonic()))
    except subprocess.TimeoutExpired:
        raise Failure("PHASE_DEADLINE_EXCEEDED_INSPECT_ORIGINAL_PROCESSES") from None
    if check: require(result.returncode == 0, "COMMAND_FAILED:" + Path(argv[0]).name)
    return result.returncode


def public_key(value):
    require(isinstance(value, str) and re.fullmatch(r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}(?: [A-Za-z0-9@._-]{1,80})?", value.strip()),
            "FULL_PUBLIC_ED25519_KEY_REQUIRED")
    value = value.strip()
    blob = base64.b64decode(value.split()[1], validate=True)
    require(len(blob) == 51 and blob[:19] == b"\x00\x00\x00\x0bssh-ed25519\x00\x00\x00\x20", "PUBLIC_KEY_INVALID")
    return " ".join(value.split()[:2])


def manifest_read(path, version):
    m = json.loads(path.read_text())
    require(m.get("format") == 1 and m.get("version") == version and re.fullmatch("[a-f0-9]{40}", m.get("source", "")),
            "RELEASE_IDENTITY_INVALID")
    require(set(m.get("files", {})) == set(RELEASE_FILES), "RELEASE_MEMBERS_INVALID")
    for value in m["files"].values(): require(re.fullmatch("[a-f0-9]{64}", value), "RELEASE_DIGEST_INVALID")
    return m


def current_state():
    path = STATE / "state.json"
    require(path.exists() and not path.is_symlink() and path.stat().st_uid == 0 and not path.stat().st_mode & 0o077,
            "MANAGED_INSTALL_REQUIRED")
    return json.loads(path.read_text())


def trusted_provider_binary(path):
    """Only trust a regular root-controlled executable; do not follow symlinks."""
    path = Path(path)
    if path.is_symlink() or not path.is_file():
        return False
    meta = path.stat()
    return (meta.st_uid == 0 and meta.st_nlink == 1 and
            not meta.st_mode & 0o022 and bool(meta.st_mode & stat.S_IXUSR))


def application_mcp_binary(path):
    """Resolve MCP after local Provider bootstrap, never from a pre-bootstrap decision."""
    require(trusted_provider_binary(path), "INSTALLED_MCP_SERVER_UNTRUSTED_OR_MISSING")
    return str(path)


def component_version(binary, *, minimum=MIN_PROVIDER_VERSION):
    """Fail closed below minimum, with no maximum and no version whitelist."""
    require(trusted_provider_binary(binary), "PROVIDER_COMPONENT_NOT_TRUSTED")
    try:
        result = subprocess.run([str(binary), "version"], capture_output=True,
                                text=True, timeout=10, check=False)
    except (OSError, subprocess.SubprocessError):
        raise Failure("PROVIDER_COMPONENT_VERSION_UNKNOWN") from None
    observed = re.findall(r"(?<![\d.])v?([0-9]+\.[0-9]+\.[0-9]+)(?![\d.])",
                          result.stdout + " " + result.stderr)
    require(result.returncode == 0 and len(set(observed)) == 1,
            "PROVIDER_COMPONENT_VERSION_UNKNOWN")
    version = observed[0]
    require(minimum is None or tuple(map(int, version.split("."))) >= tuple(map(int, minimum.split("."))),
            "PROVIDER_COMPONENT_BELOW_MINIMUM")
    return version


def toolchain_preflight(version):
    """Advisory only: advanced BUILD_TOOLS runs need a managed host toolchain."""
    if tuple(map(int, version.split("."))) < (0, 123, 0):
        return "not_required"
    return ("present" if Path("/opt/containarium/toolchains/current/bin").is_dir()
            else "missing_for_build_tools")


def check_managed(old):
    for name, digest in old.get("managed_files", {}).items():
        p = Path(name)
        # Older releases tracked this optional host-hop helper as a managed
        # file. Permit a truly missing, known pinned helper to be repaired on
        # upgrade; never exempt a changed file or even a dangling symlink.
        if (name == str(HOST_SHELL) and digest == HOST_SHELL_SHA256
                and not p.is_symlink() and not p.exists()):
            continue
        if (old.get("provider_mode") == "standalone" and
                name in {"/usr/local/bin/containariumd", "/usr/local/bin/containarium",
                         "/usr/local/bin/mcp-server"} and trusted_provider_binary(p)):
            # Upstream programs may be operator-upgraded independently. Do
            # not exempt any other managed file from byte-integrity checks.
            continue
        require(not p.is_symlink() and p.is_file() and sha(p.read_bytes()) == digest, "MANAGED_FILE_CHANGED")
    require(old.get("hostname") == socket.gethostname(), "HOST_IDENTITY_CHANGED")


def extract_upstream_containarium_shell(source):
    """Extract ONLY the pinned wrapper; never execute upstream's setup script."""
    marker = b"cat > \"$WRAPPER_SCRIPT\" << 'SHELLEOF'\n"
    terminator = b"\nSHELLEOF\n"
    require(sha(source) == HOST_SHELL_SOURCE_SHA256,
            "CONTAINARIUM_SHELL_SOURCE_PIN_MISMATCH")
    require(source.count(marker) == 1, "CONTAINARIUM_SHELL_SOURCE_FORMAT_CHANGED")
    tail = source.split(marker, 1)[1]
    require(tail.count(terminator) == 1, "CONTAINARIUM_SHELL_SOURCE_FORMAT_CHANGED")
    wrapper = tail.split(terminator, 1)[0] + b"\n"
    require(wrapper.startswith(b"#!/bin/bash\n") and len(wrapper) <= 16_384
            and sha(wrapper) == HOST_SHELL_SHA256, "CONTAINARIUM_SHELL_PIN_MISMATCH")
    return wrapper


def verified_host_shell_provider_version(*, external=False, provider_cli=None):
    """Check only the minimum version before optional host-hop setup."""
    candidate = provider_cli if external else "/usr/local/bin/containariumd"
    require(candidate and Path(candidate).is_absolute(),
            "CONTAINARIUM_SHELL_PROVIDER_VERSION_UNKNOWN")
    component_version(candidate)


def ensure_containarium_shell(release, old, *, external=False, repair=False,
                              provider_cli=None):
    """Install a reviewed upstream dependency on owned hosts, or by opt-in.

    Existing host login shells are never overwritten. A missing wrapper is
    fetched by exact upstream commit + source SHA256 + extracted body SHA256.
    Upstream's full setup-ssh-container-proxy.sh is NEVER executed.
    """
    enabled = not external or repair
    target = HOST_SHELL
    managed = str(target) in old.get("managed_files", {})
    if target.is_symlink():
        require(not enabled, "FOREIGN_CONTAINARIUM_SHELL_REFUSED")
        return "foreign_external_unmodified"
    if target.exists():
        metadata = target.stat()
        safe = (stat.S_ISREG(metadata.st_mode) and metadata.st_uid == 0
                and metadata.st_gid == 0 and not metadata.st_mode & 0o022
                and bool(metadata.st_mode & stat.S_IXOTH)
                and sha(target.read_bytes()) == HOST_SHELL_SHA256)
        if not safe:
            require(not enabled, "FOREIGN_CONTAINARIUM_SHELL_REFUSED")
            return "foreign_external_unmodified"
    elif not enabled:
        return "missing_external_unmodified"
    else:
        parent = target.parent
        info = parent.stat()
        require(parent.is_dir() and not parent.is_symlink() and info.st_uid == 0
                and not info.st_mode & 0o022 and bool(info.st_mode & stat.S_IXOTH),
                "CONTAINARIUM_SHELL_PARENT_UNSAFE")

    registry_available = not HOST_SHELLS.is_symlink() and HOST_SHELLS.is_file()
    if not registry_available:
        require(not enabled, "HOST_SHELL_REGISTRY_UNAVAILABLE")
        return "registry_unavailable_external_unmodified"
    registry_meta = HOST_SHELLS.stat()
    registry_safe = (registry_meta.st_uid == 0
                     and stat.S_ISREG(registry_meta.st_mode)
                     and not registry_meta.st_mode & 0o022)
    if not registry_safe:
        require(not enabled, "HOST_SHELL_REGISTRY_UNSAFE")
        return "registry_unsafe_external_unmodified"
    text = HOST_SHELLS.read_text()
    registered = str(target) in text.splitlines()
    if not enabled:
        return "present_external" if registered else "unregistered_external_unmodified"

    if not target.exists():
        verified_host_shell_provider_version(external=external,
                                             provider_cli=provider_cli)
        source = Path(release) / "containarium-ssh-proxy-source.sh"
        download(HOST_SHELL_UPSTREAM_URL, source, HOST_SHELL_SOURCE_SHA256,
                 limit=65_536)
        payload = extract_upstream_containarium_shell(source.read_bytes())
        # Even correctly downloaded sources must parse before a host write.
        with tempfile.NamedTemporaryFile(prefix="boxweave-shell-validate-",
                                         dir=Path(release)) as candidate:
            candidate.write(payload)
            candidate.flush()
            validation = subprocess.run(["/bin/bash", "-n", candidate.name],
                                        stdin=subprocess.DEVNULL,
                                        stdout=subprocess.DEVNULL,
                                        stderr=subprocess.DEVNULL, timeout=10)
        require(validation.returncode == 0,
                "CONTAINARIUM_SHELL_SYNTAX_INVALID")
        atomic(target, payload, 0o755)
        old.setdefault("managed_files", {})[str(target)] = HOST_SHELL_SHA256
        state = "installed_from_pinned_upstream"
    else:
        state = "present_managed" if managed else "present_external"
    if not registered:
        with HOST_SHELLS.open("a") as out:
            if text and not text.endswith("\n"):
                out.write("\n")
            out.write(str(target) + "\n")
            out.flush()
            os.fsync(out.fileno())
        state += "_registered"
    return state


def optional_containarium_shell_setup(release, old, **kwargs):
    """Record a bounded optional failure; core MCP installation continues."""
    try:
        status = ensure_containarium_shell(release, old, **kwargs)
        error_code = None
    except (Failure, OSError, ValueError, subprocess.SubprocessError) as error:
        status = "incomplete"
        error_code = (str(error) if isinstance(error, Failure)
                      else "CONTAINARIUM_SHELL_SETUP_FAILED")
    return {"status": status, "code": error_code,
            "host_hop_ssh_verified": False,
            "source_commit": HOST_SHELL_UPSTREAM_COMMIT,
            "note": "Optional host-hop setup; direct Box MCP SSH is independent."}


PG_ETC = Path("/etc/postgresql")
PG_DATA = Path("/var/lib/postgresql")
PG_BINARY_ROOT = Path("/usr/lib/postgresql")
PG_DSN = "postgresql:///boxweave_provider?host=/var/run/postgresql"


def prepare_postgres(old, deadline):
    # A dedicated, peer-authenticated local database; no database password or
    # network listener configuration is generated or copied into the app.
    if not old.get("postgres_managed"):
        require(not PG_ETC.exists() or not any(PG_ETC.iterdir()), "FOREIGN_POSTGRES_CONFIGURATION")
        require(not PG_DATA.exists() or not any(PG_DATA.iterdir()), "FOREIGN_POSTGRES_DATA")
        old["postgres_managed"] = True
        atomic(STATE / "state.json", json.dumps(old, sort_keys=True).encode())
    def installed_majors():
        return sorted((int(p.parent.parent.name) for p in PG_BINARY_ROOT.glob("*/bin/postgres")
                       if p.parent.parent.name.isdigit()), reverse=True)
    majors = installed_majors()
    if not majors:
        require(bool(shutil.which("apt-get")), "APT_GET_REQUIRED_FOR_POSTGRES")
        env = dict(os.environ, DEBIAN_FRONTEND="noninteractive")
        os_release = platform.freedesktop_os_release()
        packages = (["postgresql-16", "postgresql-contrib"] if
                    os_release.get("ID") == "ubuntu" and os_release.get("VERSION_ID") == "24.04"
                    else ["postgresql", "postgresql-contrib"])
        with (STATE / "postgres-apt.log").open("ab") as log:
            run(["apt-get", "update", "-qq"], deadline, log, env)
            run(["apt-get", "install", "-y", "-qq", *packages], deadline, log, env)
        majors = installed_majors()
    require(bool(majors), "POSTGRES_BINARY_NOT_FOUND")
    major = old.get("postgres_major", majors[0])
    require(major in majors, "MANAGED_POSTGRES_VERSION_CHANGED")
    old["postgres_major"] = major
    atomic(STATE / "state.json", json.dumps(old, sort_keys=True).encode())
    run(["systemctl", "enable", "--now", "postgresql.service"], deadline)
    run(["systemctl", "start", "postgresql@" + str(major) + "-main.service"], deadline)
    sql = """DO $body$
BEGIN
 IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='root') THEN
  CREATE ROLE root LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE;
 ELSIF EXISTS (SELECT FROM pg_roles WHERE rolname='root' AND (rolsuper OR NOT rolcanlogin)) THEN
  RAISE EXCEPTION 'managed peer role changed';
 END IF;
END
$body$;"""
    configured = subprocess.run(["runuser", "-u", "postgres", "--", "psql", "-X", "-q", "-v", "ON_ERROR_STOP=1",
                                 "--dbname=postgres"], input=sql.encode(), stdout=subprocess.DEVNULL,
                                stderr=subprocess.DEVNULL, timeout=max(.01, deadline-time.monotonic()))
    require(configured.returncode == 0, "LOCAL_POSTGRES_ROLE_FAILED")
    owner = subprocess.run(["runuser", "-u", "postgres", "--", "psql", "-X", "-At", "--dbname=postgres", "--command",
                            "SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname='boxweave_provider'"],
                           capture_output=True, text=True, timeout=15)
    require(owner.returncode == 0, "LOCAL_POSTGRES_DATABASE_OBSERVATION_FAILED")
    if not owner.stdout.strip():
        run(["runuser", "-u", "postgres", "--", "createdb", "--owner=root", "boxweave_provider"], deadline)
    else:
        require(owner.stdout.strip() == "root", "FOREIGN_POSTGRES_DATABASE_OWNER")
    ready = subprocess.run(["psql", "-X", "-At", "--dbname="+PG_DSN, "--command=SELECT 1"],
                           capture_output=True, text=True, timeout=15)
    require(ready.returncode == 0 and ready.stdout.strip() == "1", "LOCAL_POSTGRES_AUTHENTICATED_READ_FAILED")
    old["postgres_dsn"] = PG_DSN
    atomic(STATE / "state.json", json.dumps(old, sort_keys=True).encode())


def existing_provider_options(provider_url=None, provider_secret_file=None, provider_cli=None):
    """Read-only provider selection; an existing Incus alone is not a Provider."""
    incus = shutil.which("incus")
    containarium = shutil.which("containarium")
    daemon = provider_cli or shutil.which("containariumd")
    if not containarium and not daemon:
        require(not any((provider_url, provider_secret_file, provider_cli)),
                "PROVIDER_OPTIONS_REQUIRE_EXISTING_CONTAINARIUM")
        return None
    require(incus and containarium and daemon, "PARTIAL_EXISTING_PROVIDER_REVIEW_REQUIRED")
    cli = str(Path(provider_cli or daemon).resolve())
    url = provider_url or "http://127.0.0.1:8080"
    secret = str(Path(provider_secret_file or "/etc/containarium/jwt.secret"))
    # Never accept an arbitrary remote Provider or a symlinked signer.
    from urllib.parse import urlsplit
    parsed = urlsplit(url)
    require(parsed.scheme == "http" and parsed.hostname == "127.0.0.1" and
            parsed.username is None and parsed.password is None and
            parsed.path == "" and parsed.query == "" and parsed.fragment == "" and
            parsed.port is not None and 1024 <= parsed.port <= 65535,
            "EXISTING_PROVIDER_LOOPBACK_URL_REQUIRED")
    require(Path(secret).is_absolute() and ".." not in Path(secret).parts,
            "EXISTING_PROVIDER_SECRET_PATH_INVALID")
    require(Path(cli).is_absolute(), "EXISTING_PROVIDER_CLI_PATH_INVALID")
    return {"url": url, "secret": secret, "cli": cli}


def verify_existing_provider(provider):
    """No mutations of the installed service, its Incus state or signing secret."""
    secret = Path(provider["secret"])
    require(not secret.is_symlink() and secret.is_file(), "EXISTING_PROVIDER_SECRET_NOT_FOUND")
    mode = secret.stat()
    require(mode.st_uid == 0 and not mode.st_mode & 0o077,
            "EXISTING_PROVIDER_SECRET_NOT_PRIVATE")
    cli = Path(provider["cli"])
    require(not cli.is_symlink() and cli.is_file() and os.access(cli, os.X_OK),
            "EXISTING_PROVIDER_CLI_UNAVAILABLE")
    meta = cli.stat()
    require(meta.st_uid == 0 and not meta.st_mode & 0o022, "EXISTING_PROVIDER_CLI_NOT_TRUSTED")
    require(shutil.which("incus") and shutil.which("containarium"), "EXISTING_PROVIDER_DEPENDENCY_MISSING")
    pools = subprocess.run(["incus", "--force-local", "storage", "list", "--format", "json"],
                           capture_output=True, text=True, timeout=20)
    require(pools.returncode == 0 and isinstance(json.loads(pools.stdout), list) and
            bool(json.loads(pools.stdout)), "EXISTING_INCUS_STORAGE_NOT_READY")
    try:
        with urllib.request.urlopen(provider["url"] + "/v1/containers", timeout=5) as response:
            require(response.status == 200, "EXISTING_PROVIDER_API_UNEXPECTED_STATUS")
    except urllib.error.HTTPError as error:
        require(error.code == 401, "EXISTING_PROVIDER_API_UNEXPECTED_STATUS")
    except (urllib.error.URLError, TimeoutError):
        raise Failure("EXISTING_PROVIDER_API_UNAVAILABLE") from None
    # Check actual version after the authorized native upgrade below.
    component_version(provider["cli"], minimum=None)
    return provider



def install_missing_incus(deadline, log, env):
    """Let APT resolve one coherent dependency set without upgrading existing packages.

    A fresh host needs Incus before the managed Provider can start. In
    particular, do not pin only some Incus packages: incus-base must match
    the selected incus build. APT simulation checks the exact transaction
    before applying it. Hosts with Incus already installed skip this path.
    """
    packages = list(INCUS_PACKAGES)
    run(["apt-get", "-s", "install", "-y", "--no-upgrade", *packages],
        deadline, log, env)
    run(["apt-get", "install", "-y", "-qq", "--no-upgrade", *packages],
        deadline, log, env)


def build_provenance(old, descriptor, installed_utc):
    """Public, non-secret installation record; never includes credentials or digests of private files."""
    return {"schema_version": 1, "version": descriptor["version"], "source": old.get("source"),
            "installed_utc": installed_utc, "provider_mode": old.get("provider_mode"),
            "host": descriptor["host"], "port": descriptor["port"], "web_port": descriptor["web_port"],
            "application_user": descriptor["application_user"],
            "managed_files": len(old.get("managed_files", {})),
            "phases": [{"name": p["name"], "started_utc": p["started_utc"]}
                       for p in old.get("phase_history", [])][-64:]}


def installation_summary(web_port, version, remote_config=None):
    """Human-facing last screen; addresses are configuration, not reachability proof."""
    require(isinstance(version, str) and re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+", version),
            "INSTALLED_VERSION_INVALID")
    port = str(web_port)
    web_url = "http://127.0.0.1:" + port
    remote = remote_config or {}
    require(isinstance(remote, dict), "REMOTE_CONFIG_INVALID")
    http_enabled = remote.get("enabled", True)
    http_port = remote.get("port", 18844)
    require(type(http_enabled) is bool and type(http_port) is int
            and 1024 <= http_port <= 65535, "REMOTE_CONFIG_INVALID")
    local_origin = "http://127.0.0.1:" + str(http_port)
    public_origin = remote.get("public_url")
    lines = [
        "",
        "[boxweave] BoxWeave v" + version + " installed successfully",
        "",
        "==============================================================",
        "  Configure your client",
        "==============================================================",
        "",
        "  boxweave client",
        "  boxweave client --help  # More options",
        "",
        "Use your ordinary Linux administrator login (no sudo).",
        "HTTP MCP is recommended. Use --method http for automation; complete OAuth in the Agent App.",
        "Advanced SSH MCP remains available via --method ssh or existing SSH flags.",
        "SSH public-key enrollment later requires explicit administrator approval.",
        "Scripted/agent setup: pass --replace or --no-replace explicitly, or set BOXWEAVE_BATCH;",
        "interactive prompts need a full terminal.",
        "Unattended enrollment: generate with --no-enroll, approve the key on this host with",
        "  sudo python3 /usr/local/lib/boxweave/default/online.py enroll --public-key KEY,",
        "then rerun the setup with --already-enrolled.",
        "",
        "==============================================================",
        "  HTTP MCP (authenticated, loopback only)",
        "==============================================================",
        "",
    ]
    if not http_enabled:
        lines += [
            "  Disabled by remote_mcp.enabled=false in the private application config.",
            "  To enable: review /home/boxweave/boxweave/config.json and reinstall.",
        ]
    else:
        lines += [
            "  Local MCP URL: " + local_origin + "/mcp",
            "  Local info page: " + local_origin + "/",
            "  Status: boxweave http status",
            "  Diagnostics: sudo boxweave http doctor",
            "  Restart: sudo boxweave http restart",
            "  The endpoint requires credentials; installation does not prove live reachability.",
            "",
        ]
        if public_origin:
            lines += [
                "  Configured public MCP URL: " + public_origin.rstrip("/") + "/mcp",
                "  Public HTTPS/Tunnel availability is NOT verified by installation.",
                "  Use this /mcp URL in ChatGPT; authorize with the BoxWeave Web admin password.",
                "  If the external hostname changed, reconfigure and reconnect the client.",
            ]
        else:
            lines += [
                "  For hosted ChatGPT (optional; public access is NOT enabled by default):",
                "    1. Install cloudflared yourself, then run:",
                "       cloudflared tunnel --url " + local_origin,
                "    2. Copy the HTTPS hostname printed by cloudflared and configure:",
                "       sudo boxweave http configure --public-url https://YOUR_HOST.trycloudflare.com",
                "    3. Register https://YOUR_HOST.trycloudflare.com/mcp in ChatGPT.",
                "       Finish OAuth with the BoxWeave Web admin password.",
                "  Quick Tunnel is temporary; after its hostname changes, reconfigure/reconnect.",
            ]
    lines += [
        "",
        "Optional operations:",
        "  Web dashboard (server-local only): " + web_url,
        "    From your computer: ssh -N -L " + port + ":127.0.0.1:" + port +
        " YOUR_HOST_ADMIN@YOUR_LINUX_HOST",
        "    Then open " + web_url + " on your computer.",
        "  First Web admin password (private): /home/boxweave/boxweave/initial-admin-password",
        "  Host overview: boxweave",
        "  Diagnostics: boxweave status | boxweave doctor",
        "  After client setup: verify actual MCP initialize/tools/read",
    ]
    return "\n".join(lines)

def install(stage, version, provider_url=None, provider_secret_file=None, provider_cli=None,
            repair_containarium_shell=False, upstream_version=None, upstream_latest=False):
    require(os.geteuid() == 0, "ROOT_REQUIRED")
    require(sys.platform == "linux" and platform.machine() == "x86_64" and platform.python_implementation() == "CPython" and supported_python(sys.version_info),
            "LINUX_X86_64_CPYTHON_MIN312_REQUIRED")
    system = platform.freedesktop_os_release()
    # Linux runtime, systemd and actual package/tool availability are checked
    # independently. Distribution identity is not an installation gate.
    require(Path("/run/systemd/system").is_dir(), "SYSTEMD_REQUIRED")
    detected = subprocess.run(["systemd-detect-virt","--container"], capture_output=True, text=True, timeout=5)
    print(json.dumps({"nesting_preflight":{"environment":detected.stdout.strip() or "unknown",
                      "parent_nesting":"not_verified","effective_nested_execution":"not_verified",
                      "guidance":"Inside LXC, configure security.nesting=true on the exact parent instance, restart that outer box and verify a child workload. This installer does not change parent policy."}}), flush=True)
    m = manifest_read(stage / "release.json", version)
    release_url = "https://github.com/vanishs/boxweave/releases/download/v" + version + "/"
    marker = STATE / "state.json"
    if not marker.exists():
        provider = existing_provider_options(provider_url, provider_secret_file, provider_cli)
        host_package_preflight(provider is not None, system)
        if provider is not None:
            verify_existing_provider(provider)
        else:
            require(not Path("/etc/containarium").exists(), "UNMANAGED_PROVIDER_STATE_REVIEW_REQUIRED")
        # A separately prepared Incus (and its storage pool) is never overwritten.
        preexisting_incus = bool(shutil.which("incus") or Path("/var/lib/incus").exists())
        if preexisting_incus and provider is None:
            require(shutil.which("incus"), "EXISTING_INCUS_INCOMPLETE_REVIEW_REQUIRED")
            pools = subprocess.run(["incus", "--force-local", "storage", "list", "--format", "json"],
                                   capture_output=True, text=True, timeout=20)
            require(pools.returncode == 0 and bool(json.loads(pools.stdout)),
                    "EXISTING_INCUS_STORAGE_NOT_READY")
        require(not STATE.exists() and not PUBLIC.exists() and not SECRET.exists(), "FOREIGN_INSTALL_PATH")
        try: pwd.getpwnam(APP)
        except KeyError: pass
        else: raise Failure("EXISTING_APP_ACCOUNT_NOT_ADOPTED")
        directory(STATE, 0o755)
        old = {"format": 1, "hostname": socket.gethostname(), "phase": "created",
               "managed_files": {}, "provider_mode": "external" if provider else "standalone",
               "preexisting_incus": preexisting_incus}
        if provider:
            old["external_provider"] = provider
        atomic(marker, json.dumps(old).encode())
    directory(STATE, 0o755)
    lock = os.open(STATE / ".install.lock", os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        try: fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError: raise Failure("INSTALL_ALREADY_RUNNING") from None
        old = current_state(); check_managed(old)
        external = old.get("provider_mode") == "external"
        host_package_preflight(external, system)
        require(old.get("provider_mode", "standalone") in ("external", "standalone"),
                "UNKNOWN_PROVIDER_MODE")
        if external:
            existing = old["external_provider"]
            # Reruns cannot silently adopt a different Provider/signing authority.
            for requested, saved in ((provider_url, existing["url"]),
                                     (provider_secret_file, existing["secret"]),
                                     (provider_cli, existing["cli"])):
                require(requested is None or requested == saved, "EXISTING_PROVIDER_IDENTITY_CHANGED")
            verify_existing_provider(existing)
        else:
            require(not any((provider_url, provider_secret_file, provider_cli)),
                    "PROVIDER_OPTIONS_REQUIRE_EXISTING_CONTAINARIUM")
        if external:
            provider_version = component_version(old["external_provider"]["cli"], minimum=None)
        elif "/usr/local/bin/containariumd" in old.get("managed_files", {}):
            provider_version = component_version("/usr/local/bin/containariumd", minimum=None)
        else:
            provider_version = PROVIDER_VERSION
        print(json.dumps({"provider_compatibility": {"minimum": MIN_PROVIDER_VERSION,
                  "observed": provider_version, "maximum": None,
                  "managed_toolchain": toolchain_preflight(provider_version),
                  "upgrade_policy": "baseline_or_explicit_tag"}}), flush=True)
        directory(STATE / "releases")
        release = STATE / "releases" / (version + "-" + m["source"])
        directory(release)
        def phase(name, seconds=600):
            old.update(phase=name, phase_started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                       phase_deadline_seconds=seconds, source=m["source"], version=version)
            old.setdefault("phase_history", []).append({"name": name, "started_utc": old["phase_started_utc"]})
            atomic(marker, json.dumps(old, sort_keys=True, indent=2).encode())
            print(json.dumps({"phase":name, "deadline_seconds":seconds}), flush=True)
            return time.monotonic()+seconds
        if importlib.util.find_spec("ensurepip") is None:
            require(bool(shutil.which("apt-get")), "APT_GET_REQUIRED_FOR_PYTHON_VENV")
            deadline = phase("python_venv_prerequisite")
            env = dict(os.environ, DEBIAN_FRONTEND="noninteractive")
            with (STATE / "python-venv-apt.log").open("ab") as log:
                run(["apt-get", "update", "-qq"], deadline, log, env)
                run(["apt-get", "install", "-y", "-qq", "python3-venv"], deadline, log, env)
            importlib.invalidate_caches()
            require(importlib.util.find_spec("ensurepip") is not None,
                    "PYTHON_VENV_PREREQUISITE_REQUIRED")
        phase("release_download")
        for name, expected in m["files"].items():
            if name == "online.py":
                require(sha(Path(__file__).read_bytes()) == expected, "ONLINE_INSTALLER_DIGEST_MISMATCH")
                atomic(release / name, Path(__file__).read_bytes())
            else: download(release_url + name, release / name, expected)
        # This helper is part of the digest-verified release and is loaded only
        # after all its source bytes have passed the release manifest check.
        spec = importlib.util.spec_from_file_location("upstream_version", release / "upstream_version.py")
        require(spec is not None and spec.loader is not None, "UPSTREAM_HELPER_UNAVAILABLE")
        manager = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(manager)
        require(manager.BASELINE == MIN_PROVIDER_VERSION and manager.PINNED == PROVIDER_ASSETS,
                "UPSTREAM_BASELINE_SOURCE_MISMATCH")
        upstream_selected, upstream_digests = manager.release(upstream_version, upstream_latest)
        fresh_provider = not external and "/usr/local/bin/containariumd" not in old.get("managed_files", {})
        bootstrap_stage = release / "upstream-bootstrap"
        if fresh_provider:
            bootstrap_stage.mkdir(mode=0o700, exist_ok=True)
            try:
                manager.download_assets(bootstrap_stage, upstream_selected, upstream_digests)
            except manager.UpgradeFailure as exc:
                raise Failure(str(exc)) from None
        else:
            deadline = phase("upstream_compatibility", seconds=900)
            try:
                upgrade_receipt = manager.ensure(
                    old, cli=(old["external_provider"]["cli"] if external else "/usr/local/bin/containariumd"),
                    url=(old["external_provider"]["url"] if external else "http://127.0.0.1:18080"),
                    secret=(old["external_provider"]["secret"] if external else str(SECRET)),
                    target=upstream_version, latest=upstream_latest, standalone=not external)
            except manager.UpgradeFailure as exc:
                raise Failure(str(exc)) from None
            atomic(marker, json.dumps(old, sort_keys=True, indent=2).encode())
            print(json.dumps({"upstream_upgrade": upgrade_receipt}), flush=True)
        local_mcp = Path("/usr/local/bin/mcp-server")
        use_installed_mcp = local_mcp.exists() and trusted_provider_binary(local_mcp)
        if not use_installed_mcp and not fresh_provider:
            raise Failure("INSTALLED_MCP_SERVER_UNTRUSTED_OR_MISSING")
        if external:
            deadline = phase("external_provider_preflight")
            # Preserve the reviewed endpoint, signing authority and Incus state.
            verify_existing_provider(old["external_provider"])
        else:
            if not shutil.which("incus"):
                require(bool(shutil.which("apt-get")), "APT_GET_REQUIRED_FOR_INCUS")
                deadline = phase("signed_apt_prerequisites")
                env = dict(os.environ, DEBIAN_FRONTEND="noninteractive")
                with (STATE / "apt.log").open("ab") as log:
                    run(["apt-get", "update", "-qq"], deadline, log, env)
                    run(["apt-get", "install", "-y", "-qq", "ca-certificates", "curl", "gnupg", "openssh-client",
                         "python3-venv", "dbus-user-session", "openssh-server", "iproute2"], deadline, log, env)
                    download("https://pkgs.zabbly.com/key.asc", release / "zabbly.asc", KEY_SHA256, 65536)
                    directory(Path("/etc/apt/keyrings"), 0o755)
                    keyring = Path("/etc/apt/keyrings/boxweave-zabbly.asc")
                    source = Path("/etc/apt/sources.list.d/boxweave-incus.sources")
                    require(all(not p.exists() or str(p) in old["managed_files"] for p in (keyring,source)), "FOREIGN_APT_CONFIGURATION")
                    atomic(keyring, (release/"zabbly.asc").read_bytes(), 0o644)
                    codename = system.get("VERSION_CODENAME", "")
                    require(re.fullmatch(r"[a-z]{4,32}", codename), "APT_REPOSITORY_SUITE_REQUIRED_FOR_INCUS")
                    source_text = ("Types: deb\nURIs: https://pkgs.zabbly.com/incus/stable\nSuites: " + codename +
                                   "\nComponents: main\nArchitectures: amd64\nSigned-By: /etc/apt/keyrings/boxweave-zabbly.asc\n")
                    atomic(source, source_text.encode(), 0o644)
                    # Record exact managed files before invoking APT so repair preserves their provenance.
                    old["managed_files"].update({str(keyring):sha(keyring.read_bytes()), str(source):sha(source.read_bytes())})
                    atomic(marker, json.dumps(old, sort_keys=True).encode())
                    run(["apt-get", "update", "-qq"], deadline, log, env)
                    install_missing_incus(deadline, log, env)
            deadline = phase("local_provider")
            observed = subprocess.run(["incus", "--version"], capture_output=True, text=True, timeout=10)
            require(observed.returncode == 0 and re.fullmatch(r"[0-9]+\.[0-9]+(?:\.[0-9]+)?", observed.stdout.strip()),
                    "INCUS_VERSION_UNKNOWN")
            incus_version = tuple(int(n) for n in observed.stdout.strip().split("."))
            require(incus_version >= (7, 5, 1), "INCUS_MIN_VERSION_REQUIRED")
            # Initialize only an inspected empty local daemon; never replace an existing pool/profile.
            pools=subprocess.run(["incus","--force-local","storage","list","--format","json"],capture_output=True,text=True,timeout=20)
            require(pools.returncode==0,"LOCAL_INCUS_NOT_READY")
            if not json.loads(pools.stdout):
                inventory=subprocess.run(["incus","--force-local","list","--all-projects","--format","json"],capture_output=True,text=True,timeout=20)
                require(inventory.returncode==0 and json.loads(inventory.stdout)==[],"INCUS_NONEMPTY_WITHOUT_STORAGE")
                routes=json.loads(subprocess.check_output(["ip","-j","-4","route"],text=True,timeout=10))
                occupied=[ipaddress.ip_network(r["dst"],strict=False) for r in routes if r.get("dst") not in (None,"default")]
                subnet=next((ipaddress.ip_network("10."+str(n)+".0.0/24") for n in range(203,250)
                             if all(not ipaddress.ip_network("10."+str(n)+".0.0/24").overlaps(other) for other in occupied)),None)
                require(subnet is not None,"NO_FREE_PRIVATE_SUBNET")
                address=str(subnet.network_address+1)+"/24"
                preseed={"config":{},"networks":[{"name":"incusbr0","type":"bridge","config":{"ipv4.address":address,"ipv4.nat":"true","ipv6.address":"none"}}],
                         "storage_pools":[{"name":"default","driver":"dir","config":{}}],
                         "profiles":[{"name":"default","devices":{"eth0":{"type":"nic","name":"eth0","network":"incusbr0"},"root":{"type":"disk","path":"/","pool":"default"}}}]}
                prepared=subprocess.run(["incus","--force-local","admin","init","--preseed"],input=json.dumps(preseed).encode(),
                                        stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=60)
                require(prepared.returncode==0,"EMPTY_INCUS_INITIALIZATION_FAILED")
                old["network_subnet"]=address
                atomic(marker,json.dumps(old,sort_keys=True).encode())
            for name, expected in PROVIDER_ASSETS.items():
                target = Path("/usr/local/bin") / name
                if target.exists():
                    require(trusted_provider_binary(target), "PROVIDER_COMPONENT_NOT_TRUSTED")
                else:
                    require(fresh_provider, "UPSTREAM_COMPONENT_MISSING_AFTER_UPGRADE")
                    atomic(target, (bootstrap_stage / name).read_bytes(), 0o755)
                old["managed_files"][str(target)] = sha(target.read_bytes())
            component_version("/usr/local/bin/containariumd")
            if not SECRET.exists():
                directory(SECRET.parent)
                atomic(SECRET, secrets.token_hex(32).encode())
            require(not SECRET.is_symlink() and SECRET.stat().st_uid == 0 and not SECRET.stat().st_mode & 0o077, "SIGNING_FILE_NOT_PRIVATE")
            deadline = phase("local_postgres")
            prepare_postgres(old, deadline)
            deadline = phase("local_provider_start")
            provider = ("[Unit]\nDescription=BoxWeave local Provider\nAfter=incus.service\n[Service]\n"
                        "ExecStart=/usr/local/bin/containariumd daemon --address 127.0.0.1 --http-port 18080 --rest --standalone "
                        "--disable-security-scanner --disable-pentest-scanner --disable-zap-scanner "
                        "--postgres "+PG_DSN+" --network-subnet "+old.get("network_subnet","10.203.0.1/24")+" --jwt-secret-file /etc/boxweave-provider/jwt.secret\nRestart=on-failure\nRestartSec=3\nUMask=0077\n"
                        "[Install]\nWantedBy=multi-user.target\n").encode()
            provider_unit = UNIT_DIR / "boxweave-provider.service"
            require(not provider_unit.exists() or str(provider_unit) in old["managed_files"], "FOREIGN_PROVIDER_UNIT")
            previous = provider_unit.read_bytes() if provider_unit.exists() else None
            atomic(provider_unit, provider, 0o644)
            old["managed_files"][str(provider_unit)] = sha(provider)
            atomic(marker, json.dumps(old, sort_keys=True).encode())
            run(["systemctl", "daemon-reload"], deadline)
            run(["systemctl", "enable", "--now", provider_unit.name], deadline)
            if previous is not None and previous != provider:
                run(["systemctl", "restart", provider_unit.name], deadline)
            # Reachability only. The app installer independently proves authenticated MCP.
            while True:
                try:
                    urllib.request.urlopen("http://127.0.0.1:18080/v1/containers", timeout=3).close(); break
                except urllib.error.HTTPError as e:
                    if e.code == 401: break
                except OSError: pass
                require(time.monotonic() < deadline, "PROVIDER_API_NOT_READY")
                time.sleep(15)
        old["upstream_development_baseline"] = MIN_PROVIDER_VERSION
        old["upstream_version_observed"] = component_version(
            old["external_provider"]["cli"] if external else "/usr/local/bin/containariumd")
        atomic(marker, json.dumps(old, sort_keys=True, indent=2).encode())
        phase("containarium_host_shell")
        # Host-hop SSH is optional; independent app/MCP installation continues
        # after a pinned-source fetch, compatibility or host-ownership failure.
        shell_result = optional_containarium_shell_setup(
            release, old, external=external, repair=repair_containarium_shell,
            provider_cli=old["external_provider"]["cli"] if external else None)
        print(json.dumps({"containarium_shell": shell_result}), flush=True)
        deadline = phase("independent_application")
        mcp_binary = application_mcp_binary(local_mcp)
        provider_binary = old["external_provider"]["cli"] if external else "/usr/local/bin/containariumd"
        provider_digest = sha(Path(provider_binary).read_bytes())
        mcp_digest = sha(Path(mcp_binary).read_bytes())
        secret_file = old["external_provider"]["secret"] if external else str(SECRET)
        api_url = old["external_provider"]["url"] if external else "http://127.0.0.1:18080"
        args = ["/usr/bin/python3", str(release/"bootstrap.py"), "apply", "--app-user", APP,
                "--archive", str(release/"boxweave.zip"), "--sha256", m["files"]["boxweave.zip"],
                "--distribution", str(release/"distribution.py"), "--distribution-sha256", m["files"]["distribution.py"],
                "--mcp-binary", mcp_binary, "--mcp-sha256", mcp_digest,
                "--provider-cli", provider_binary, "--provider-cli-sha256", provider_digest,
                "--secret-file", secret_file, "--provider-url", api_url,
                "--ssh-entry", "--ack-root-changes", "INITIALIZE_BOXWEAVE"]
        with (STATE/"application.log").open("ab") as log:
            code = run(args, deadline, log, check=False)
        if code:
            app_log = Path("/var/lib/boxweave-bootstrap/boxweave/application-install.log")
            # A standalone installer may emit a traceback instead of JSON.
            # Keep the original private log, and emit the actionable typed
            # failure instead of collapsing it into INSTALL_OR_CONFIGURATION_FAILED.
            try:
                info = json.loads(app_log.read_text()) if app_log.exists() else {}
            except (OSError, UnicodeError, ValueError):
                info = {}
            require(isinstance(info, dict) and info.get("code") == "ADMIN_SETUP_REQUIRED",
                    "APPLICATION_INSTALL_INCOMPLETE_INSPECT_PRIVATE_LOG")
            account = pwd.getpwnam(APP)
            admin_script = """import os,secrets,stat
from pathlib import Path
from werkzeug.security import generate_password_hash
from bwmini.__main__ import read_config
from bwmini.store import Store
c=read_config('/home/boxweave/boxweave/runtime.json');store=Store(c['database'])
p=Path('/home/boxweave/boxweave/initial-admin-password')
if p.exists():
    meta=p.lstat()
    assert stat.S_ISREG(meta.st_mode) and meta.st_uid==os.getuid() and not meta.st_mode & 0o077 and meta.st_nlink==1
    password=p.read_text().strip()
else:
    password=secrets.token_urlsafe(32)
    fd=os.open(p,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW,0o600)
    with os.fdopen(fd,'w') as out:out.write(password+chr(10));out.flush();os.fsync(out.fileno())
store.admin(generate_password_hash(password))
"""
            run(["runuser", "-u", APP, "--", "env", "XDG_RUNTIME_DIR=/run/user/"+str(account.pw_uid),
                 "PYTHONPATH=/home/boxweave/boxweave/current", "/home/boxweave/boxweave/current/.venv/bin/python",
                 "-c", admin_script], deadline)
            with (STATE/"application.log").open("ab") as log: run(args, deadline, log)
        account = pwd.getpwnam(APP)
        sshdir = Path(account.pw_dir)/".ssh"
        if not sshdir.exists():
            sshdir.mkdir(mode=0o700); os.chown(sshdir, account.pw_uid, account.pw_gid)
        require(not sshdir.is_symlink() and sshdir.stat().st_uid == account.pw_uid, "APP_SSH_DIRECTORY_UNSAFE")
        private, pub = sshdir/"id_ed25519", sshdir/"id_ed25519.pub"
        if not private.exists() and not pub.exists():
            run(["runuser", "-u", APP, "--", "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "boxweave-managed", "-f", str(private)], deadline)
        require(private.is_file() and pub.is_file() and not private.is_symlink() and not pub.is_symlink(), "MANAGED_IDENTITY_INCOMPLETE")
        managed_public_key = public_key(pub.read_text())
        directory(PUBLIC, 0o755)
        for name in ("online.py","hostkeys.py","mcp_gate.py","resource_size.py","gate_diagnostics.py","mcp_compat.py","restore_guard.py","operation_coordination.py","coordination_runtime.py","coordination_bridge.py","box_reads.py", "box_patch.py","gateway_tasks.py","gateway_protection.py","nesting.py","client_setup.py","client_profiles.py","host_cli.py","upstream_version.py"):
            target = PUBLIC/name
            require(not target.exists() or str(target) in old["managed_files"], "FOREIGN_PUBLIC_HELPER")
            payload = (release/name).read_bytes(); atomic(target,payload,0o755)
            old["managed_files"][str(target)] = sha(payload)
        # Execute only root-owned, isolated stdlib code for all host CLI
        # actions. Application-owned modules must never be run as root.
        entry = Path("/usr/local/bin/boxweave")
        entry_script = (b'#!/bin/sh\n'
                        b'exec /usr/bin/python3 -I /usr/local/lib/boxweave/default/host_cli.py "$@"\n')
        require(not entry.exists() or str(entry) in old["managed_files"], "FOREIGN_BOXWEAVE_ENTRY")
        atomic(entry, entry_script, 0o755)
        old["managed_files"][str(entry)] = sha(entry_script)
        sudoers = Path("/etc/sudoers.d/boxweave-nesting")
        require(not sudoers.exists() or str(sudoers) in old["managed_files"], "FOREIGN_NESTING_SUDOERS")
        payload = b"boxweave ALL=(root) NOPASSWD: /usr/bin/python3 -I /usr/local/lib/boxweave/default/nesting.py\n"
        # Fixed executable/arguments only; the isolated helper reads bounded validated stdin.
        require(shutil.which("visudo") is not None, "SUDO_POLICY_VALIDATOR_REQUIRED")
        candidate = release / "nesting.sudoers"
        atomic(candidate, payload, 0o440)
        run(["visudo","-cf",str(candidate)], deadline)
        atomic(sudoers, payload, 0o440); old["managed_files"][str(sudoers)] = sha(payload)
        sync_service = b"[Unit]\nDescription=Synchronize Incus-authenticated public host keys\n[Service]\nType=oneshot\nUMask=0077\nExecStart=/usr/bin/python3 /usr/local/lib/boxweave/default/hostkeys.py\n"
        sync_timer = b"[Unit]\nDescription=Refresh trusted workload host keys\n[Timer]\nOnBootSec=15s\nOnUnitInactiveSec=15s\nAccuracySec=1s\n[Install]\nWantedBy=timers.target\n"
        for name,payload in (("boxweave-trust.service",sync_service),("boxweave-trust.timer",sync_timer)):
            target=UNIT_DIR/name
            require(not target.exists() or str(target) in old["managed_files"], "FOREIGN_TRUST_UNIT")
            atomic(target,payload,0o644);old["managed_files"][str(target)]=sha(payload)
        atomic(marker,json.dumps(old,sort_keys=True).encode())
        run(["systemctl","daemon-reload"],deadline)
        run(["systemctl","start","boxweave-trust.service"],deadline)
        run(["systemctl","enable","--now","boxweave-trust.timer"],deadline)
        # Existing sshd configuration/host identity is observed, never rewritten.
        require(Path("/usr/sbin/sshd").exists() and Path("/etc/ssh/ssh_host_ed25519_key.pub").is_file(), "PREPARE_SSH_SERVER_SEPARATELY")
        run(["/usr/sbin/sshd","-t"],deadline)
        key=public_key(Path("/etc/ssh/ssh_host_ed25519_key.pub").read_text())
        with socket.socket(socket.AF_INET,socket.SOCK_DGRAM) as detect:
            detect.connect(("1.1.1.1",53));host=detect.getsockname()[0]
        require(ipaddress.ip_address(host).is_private, "PRIVATE_SSH_ENDPOINT_REQUIRED")
        port=22
        sshd=subprocess.run(["/usr/sbin/sshd","-T"],capture_output=True,text=True,timeout=10)
        require(sshd.returncode==0,"SSHD_OBSERVATION_FAILED")
        for line in sshd.stdout.splitlines():
            if line.startswith("port "):port=int(line.split()[1]);break
        # Publish only non-secret connection details in a root-owned 0644
        # descriptor. The app directory and credentials remain private.
        app_cfg = Path("/home/boxweave/boxweave/config.json")
        require(app_cfg.is_file() and not app_cfg.is_symlink(), "APP_PUBLIC_CONFIG_UNAVAILABLE")
        raw_cfg = app_cfg.read_bytes()
        require(len(raw_cfg) <= 65536, "APP_CONFIG_TOO_LARGE")
        cfg = json.loads(raw_cfg)
        web_port = cfg.get("port")
        require(cfg.get("host") == "127.0.0.1" and type(web_port) is int
                and 1024 <= web_port <= 65535, "PUBLIC_WEB_ENDPOINT_NOT_LOOPBACK")
        descriptor = {"schema_version":1,"phase":"ready","version":version,"application_user":APP,
                      "app_prefix":"/home/boxweave/boxweave","web_port":web_port,
                      "host":host,"port":port,"server_key":key,
                      "managed_public_key":managed_public_key}
        public_path = PUBLIC / "client-public.json"
        public_payload = (json.dumps(descriptor,sort_keys=True,indent=2)+"\n").encode()
        atomic(public_path,public_payload,0o644)
        old["managed_files"][str(public_path)] = sha(public_payload)
        provenance_path = PUBLIC / "provenance.json"
        provenance_payload = (json.dumps(build_provenance(old, descriptor, datetime.datetime.now(datetime.timezone.utc).isoformat()),sort_keys=True,indent=2)+"\n").encode()
        require(not provenance_path.exists() or str(provenance_path) in old["managed_files"], "FOREIGN_PROVENANCE_FILE")
        atomic(provenance_path,provenance_payload,0o644)
        old["managed_files"][str(provenance_path)] = sha(provenance_payload)
        old.update(phase="ready",host=host,port=port,server_key=key,managed_public_key=managed_public_key,
                   credential_contents="not_read",provider_mode="external" if external else "standalone",
                   upstream_development_baseline=MIN_PROVIDER_VERSION, version=version,source=m["source"])
        atomic(marker,json.dumps(old,sort_keys=True,indent=2).encode())
        http_config=cfg.get("remote_mcp") or {}
        http_enabled=http_config.get("enabled",True)
        http_port=http_config.get("port",18844)
        print(json.dumps({"next_steps":{
            "client_setup":"boxweave client (no sudo)",
            "client_help":"boxweave client --help",
            "cli":"boxweave", "cli_help":"boxweave --help", "status":"boxweave status",
            "web_on_server":"http://127.0.0.1:"+str(web_port),
            "http_mcp_local":("http://127.0.0.1:"+str(http_port)+"/mcp"
                              if http_enabled else "disabled"),
            "http_mcp_status":"boxweave http status",
            "http_mcp_diagnostics":"sudo boxweave http doctor",
            "http_mcp_public_config":"sudo boxweave http configure --public-url https://YOUR-TUNNEL-HOST",
            "http_mcp_cloudflare_quick_tunnel":"cloudflared tunnel --url http://127.0.0.1:"+str(http_port),
            "http_mcp_note":"Only loopback is enabled automatically. Public HTTPS is opt-in; configure the actual Quick Tunnel origin before connecting a hosted client.",
            "web_access":"Use a reviewed SSH tunnel; Web is loopback-only",
            "client_role":"Use your authorized Linux SSH/sudo login; not the boxweave service account",
            "admin_password_location":"/home/boxweave/boxweave/initial-admin-password",
            "verification":"Run boxweave doctor; verify real client MCP calls"
        }}),flush=True)
        print(json.dumps({"state":"ready","version":version,"host":host,"ssh_port":port,"user":APP,
                          "client_config_command":"/usr/bin/python3 /usr/local/lib/boxweave/default/online.py client-config --identity ABSOLUTE_CLIENT_KEY --known-hosts ABSOLUTE_CLIENT_KNOWN_HOSTS",
                          "enroll_command":"sudo python3 /usr/local/lib/boxweave/default/online.py enroll --public-key FULL_CLIENT_PUBLIC_KEY",
                          "admin_password_file":"/home/boxweave/boxweave/initial-admin-password",
                          "public_routes_created":False,"provider_mode":old["provider_mode"],"reboot":"not_verified"}),flush=True)
        # Human-facing last screen; JSON above remains machine-readable.
        print(installation_summary(web_port, version, cfg.get("remote_mcp")), flush=True)

    finally: os.close(lock)


def enroll(value):
    require(os.geteuid()==0,"ROOT_REQUIRED")
    state=current_state();require(state.get("phase")=="ready","READY_INSTALL_REQUIRED");check_managed(state)
    key=public_key(value)
    account=pwd.getpwnam(APP);path=Path(account.pw_dir)/".ssh/authorized_keys"
    require(not path.is_symlink(),"AUTHORIZATION_SYMLINK_REFUSED")
    old=path.read_text() if path.exists() else ""
    line='restrict,command="/usr/bin/python3 /usr/local/lib/boxweave/default/mcp_gate.py" '+key+' boxweave-client'
    matching=[v for v in old.splitlines() if key.split()[1] in v.split()]
    require(not matching or matching==[line],"CLIENT_KEY_CONFLICT")
    if not matching:
        if path.exists():atomic(path.with_name("authorized_keys.backup-"+str(time.time_ns())),old.encode(),0o600,account.pw_uid,account.pw_gid)
        atomic(path,(old+("" if not old or old.endswith("\n") else "\n")+line+"\n").encode(),0o600,account.pw_uid,account.pw_gid)
    print(json.dumps({"state":"authorized","changed":not matching,"public_key_only":True}))


def client_config(identity,known_hosts):
    state=current_state();require(state.get("phase")=="ready","READY_INSTALL_REQUIRED")
    require(Path(identity).is_absolute() and Path(known_hosts).is_absolute() and
            not any(c in identity+known_hosts for c in "\r\n"),"ABSOLUTE_CLIENT_PATH_REQUIRED")
    host=state["host"];port=state["port"];endpoint=host if port==22 else "["+host+"]:"+str(port)
    args=["-F","/dev/null","-T","-p",str(port),"-o","BatchMode=yes","-o","IdentitiesOnly=yes",
          "-o","StrictHostKeyChecking=yes","-o","UserKnownHostsFile="+known_hosts,"-i",identity,
          APP+"@"+host,"/usr/bin/python3 /usr/local/lib/boxweave/default/mcp_gate.py"]
    print(json.dumps({"mcpServers":{"boxweave":{"command":"/usr/bin/ssh","args":args}},
                      "known_hosts_line":endpoint+" "+state["server_key"],
                      "workload_ssh_public_key":state["managed_public_key"]},indent=2))


def main():
    p=argparse.ArgumentParser(description=__doc__);sub=p.add_subparsers(dest="action",required=True)
    a=sub.add_parser("install");a.add_argument("--stage",required=True);a.add_argument("--version",required=True)
    a.add_argument("--provider-url");a.add_argument("--provider-secret-file");a.add_argument("--provider-cli")
    a.add_argument("--repair-containarium-shell",action="store_true")
    choice = a.add_mutually_exclusive_group()
    choice.add_argument("--upstream-version")
    choice.add_argument("--upstream-latest", action="store_true")
    a=sub.add_parser("enroll");a.add_argument("--public-key",required=True)
    a=sub.add_parser("client-config");a.add_argument("--identity",required=True);a.add_argument("--known-hosts",required=True)
    args=p.parse_args()
    try:
        if args.action=="install":install(Path(args.stage),args.version,args.provider_url,args.provider_secret_file,args.provider_cli,
                                          repair_containarium_shell=args.repair_containarium_shell,
                                           upstream_version=args.upstream_version, upstream_latest=args.upstream_latest)
        elif args.action=="enroll":enroll(args.public_key)
        else:client_config(args.identity,args.known_hosts)
        return 0
    except (Failure,OSError,ValueError,KeyError,subprocess.SubprocessError):
        error=sys.exc_info()[1]
        print(json.dumps({"state":"incomplete","code":str(error) if isinstance(error,Failure) else "INSTALL_OR_CONFIGURATION_FAILED",
                          "evidence":str(STATE),"retry":"Inspect the recorded phase and original processes before rerunning the same install command."}),flush=True)
        return 2


if __name__=="__main__":raise SystemExit(main())
