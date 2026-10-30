"""Source-bound BoxWeave install/update/repair for fresh or reviewed existing Providers.
Existing Containarium/Incus are never replaced or upgraded. Credentials stay private.
"""
import argparse, base64, datetime, fcntl, hashlib, importlib.util, ipaddress, json, os
from pathlib import Path
import platform, pwd, re, secrets, shutil, socket, stat, subprocess, sys, tempfile, time, urllib.request, urllib.error

STATE = Path("/var/lib/boxweave-default")
PUBLIC = Path("/usr/local/lib/boxweave/default")
APP = "boxweave"
PROVIDER_VERSION = "0.120.1"
INCUS_PACKAGE = "1:7.5.1-ubuntu24.04-202610270033"
KEY_SHA256 = "f1adb44fdf82083f983a392b5b406d34e473dce3d66a51de3118cdc1f51d3f87"
PROVIDER_ASSETS = {
    "containariumd": "7d992e5df41d7070c187e8b11d516d61e8b89b4b7be677f7a0880b9cfed5d4bd",
    "containarium": "dc356b07ca2138f28052d021458f8081753b2c6c186fe7508b0645a06a9b121d",
    "mcp-server": "5b12e284b3b103c01bf9eff0cfe2f2a1e3b25476dfcc72aebe8166b4235198a2",
}
RELEASE_FILES = ("boxweave.zip", "bootstrap.py", "distribution.py", "online.py", "hostkeys.py", "mcp_gate.py", "nesting.py", "client_setup.py", "host_cli.py")
SECRET = Path("/etc/boxweave-provider/jwt.secret")
UNIT_DIR = Path("/etc/systemd/system")


class Failure(Exception): pass


def supported_ubuntu(version):
    """Accept Ubuntu 24.04 and newer; do not parse release numbers as floats."""
    return isinstance(version, str) and re.fullmatch(r"[0-9]{2,}\.[0-9]{2}", version) is not None and tuple(map(int, version.split("."))) >= (24, 4)


def supported_python(version):
    return tuple(version[:2]) >= (3, 12)



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


def check_managed(old):
    for name, digest in old.get("managed_files", {}).items():
        p = Path(name)
        require(not p.is_symlink() and p.is_file() and sha(p.read_bytes()) == digest, "MANAGED_FILE_CHANGED")
    require(old.get("hostname") == socket.gethostname(), "HOST_IDENTITY_CHANGED")


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
        env = dict(os.environ, DEBIAN_FRONTEND="noninteractive")
        packages = (["postgresql-16", "postgresql-contrib"] if
                    platform.freedesktop_os_release()["VERSION_ID"] == "24.04"
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
    return provider


def install(stage, version, provider_url=None, provider_secret_file=None, provider_cli=None):
    require(os.geteuid() == 0, "ROOT_REQUIRED")
    require(sys.platform == "linux" and platform.machine() == "x86_64" and platform.python_implementation() == "CPython" and supported_python(sys.version_info),
            "UBUNTU_MIN_2404_X86_64_PYTHON_MIN312_REQUIRED")
    system = platform.freedesktop_os_release()
    require(system.get("ID") == "ubuntu" and supported_ubuntu(system.get("VERSION_ID")), "UBUNTU_MIN_2404_REQUIRED")
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
        directory(STATE / "releases")
        release = STATE / "releases" / (version + "-" + m["source"])
        directory(release)
        def phase(name):
            old.update(phase=name, phase_started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                       phase_deadline_seconds=600, source=m["source"], version=version)
            atomic(marker, json.dumps(old, sort_keys=True, indent=2).encode())
            print(json.dumps({"phase":name, "deadline_seconds":600}), flush=True)
            return time.monotonic()+600
        if importlib.util.find_spec("ensurepip") is None:
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
        # Fixed official upstream assets, never mutable latest or an existing-host upgrade.
        for name, digest in PROVIDER_ASSETS.items():
            # The externally managed Provider's binaries are never downloaded or installed.
            if not external or name == "mcp-server":
                download("https://github.com/FootprintAI/Containarium/releases/download/v" + PROVIDER_VERSION + "/" + name + "-linux-amd64",
                         release / name, digest)
        if external:
            deadline = phase("external_provider_preflight")
            # Do not install, upgrade, restart, or initialize Containarium or Incus.
            verify_existing_provider(old["external_provider"])
        else:
            if not shutil.which("incus"):
                deadline = phase("signed_ubuntu_prerequisites")
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
                    require(re.fullmatch(r"[a-z]{4,32}", codename), "UBUNTU_CODENAME_INVALID")
                    source_text = ("Types: deb\nURIs: https://pkgs.zabbly.com/incus/stable\nSuites: " + codename +
                                   "\nComponents: main\nArchitectures: amd64\nSigned-By: /etc/apt/keyrings/boxweave-zabbly.asc\n")
                    atomic(source, source_text.encode(), 0o644)
                    # Record exact managed files before invoking APT so repair preserves their provenance.
                    old["managed_files"].update({str(keyring):sha(keyring.read_bytes()), str(source):sha(source.read_bytes())})
                    atomic(marker, json.dumps(old, sort_keys=True).encode())
                    run(["apt-get", "update", "-qq"], deadline, log, env)
                    packages = (["incus="+INCUS_PACKAGE, "incus-client="+INCUS_PACKAGE,
                                 "incus-extra="+INCUS_PACKAGE] if system["VERSION_ID"] == "24.04"
                                else ["incus", "incus-client", "incus-extra"])
                    run(["apt-get", "install", "-y", "-qq", *packages], deadline, log, env)
            deadline = phase("local_provider")
            observed = subprocess.run(["incus", "--version"], capture_output=True, text=True, timeout=10)
            require(observed.returncode == 0 and re.fullmatch(r"[0-9]+\.[0-9]+(?:\.[0-9]+)?", observed.stdout.strip()),
                    "INCUS_VERSION_UNKNOWN")
            incus_version = tuple(int(n) for n in observed.stdout.strip().split("."))
            require(incus_version >= (7, 5, 1), "INCUS_MIN_VERSION_REQUIRED")
            if not old.get("preexisting_incus") and system["VERSION_ID"] == "24.04":
                require(observed.stdout.strip() == "7.5.1", "INCUS_PIN_CHANGED_REVIEW_REQUIRED")
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
                if target.exists(): require(not target.is_symlink() and sha(target.read_bytes()) == expected, "PROVIDER_UPGRADE_OR_FOREIGN_BINARY_REFUSED")
                else: atomic(target, (release/name).read_bytes(), 0o755)
                old["managed_files"][str(target)] = expected
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
        deadline = phase("independent_application")
        mcp_binary = str(release / "mcp-server") if external else "/usr/local/bin/mcp-server"
        provider_binary = old["external_provider"]["cli"] if external else "/usr/local/bin/containariumd"
        provider_digest = sha(Path(provider_binary).read_bytes()) if external else PROVIDER_ASSETS["containariumd"]
        secret_file = old["external_provider"]["secret"] if external else str(SECRET)
        api_url = old["external_provider"]["url"] if external else "http://127.0.0.1:18080"
        args = ["/usr/bin/python3", str(release/"bootstrap.py"), "apply", "--app-user", APP,
                "--archive", str(release/"boxweave.zip"), "--sha256", m["files"]["boxweave.zip"],
                "--distribution", str(release/"distribution.py"), "--distribution-sha256", m["files"]["distribution.py"],
                "--mcp-binary", mcp_binary, "--mcp-sha256", PROVIDER_ASSETS["mcp-server"],
                "--provider-cli", provider_binary, "--provider-cli-sha256", provider_digest,
                "--secret-file", secret_file, "--provider-url", api_url,
                "--ssh-entry", "--ack-root-changes", "INITIALIZE_BOXWEAVE"]
        with (STATE/"application.log").open("ab") as log:
            code = run(args, deadline, log, check=False)
        if code:
            app_log = Path("/var/lib/boxweave-bootstrap/boxweave/application-install.log")
            info = json.loads(app_log.read_text()) if app_log.exists() else {}
            require(info.get("code") == "ADMIN_SETUP_REQUIRED", "APPLICATION_INSTALL_INCOMPLETE_INSPECT_PRIVATE_LOG")
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
        for name in ("online.py","hostkeys.py","mcp_gate.py","nesting.py","client_setup.py","host_cli.py"):
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
        old.update(phase="ready",host=host,port=port,server_key=key,managed_public_key=managed_public_key,
                   credential_contents="not_read",provider_mode="external" if external else "standalone",version=version,source=m["source"])
        atomic(marker,json.dumps(old,sort_keys=True,indent=2).encode())
        print(json.dumps({"next_steps":{
            "cli":"boxweave", "cli_help":"boxweave --help", "status":"boxweave status",
            "web_on_server":"http://127.0.0.1:"+str(web_port),
            "web_access":"Use a reviewed SSH tunnel; Web is loopback-only",
            "client_setup":"boxweave client (no sudo)",
            "client_role":"Use your authorized Ubuntu SSH/sudo login; not the boxweave service account",
            "admin_password_location":"/home/boxweave/boxweave/initial-admin-password",
            "verification":"Run boxweave doctor; verify real client MCP calls"
        }}),flush=True)
        print(json.dumps({"state":"ready","version":version,"host":host,"ssh_port":port,"user":APP,
                          "client_config_command":"/usr/bin/python3 /usr/local/lib/boxweave/default/online.py client-config --identity ABSOLUTE_CLIENT_KEY --known-hosts ABSOLUTE_CLIENT_KNOWN_HOSTS",
                          "enroll_command":"sudo python3 /usr/local/lib/boxweave/default/online.py enroll --public-key FULL_CLIENT_PUBLIC_KEY",
                          "admin_password_file":"/home/boxweave/boxweave/initial-admin-password",
                          "public_routes_created":False,"provider_mode":old["provider_mode"],"reboot":"not_verified"}),flush=True)
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
    a=sub.add_parser("enroll");a.add_argument("--public-key",required=True)
    a=sub.add_parser("client-config");a.add_argument("--identity",required=True);a.add_argument("--known-hosts",required=True)
    args=p.parse_args()
    try:
        if args.action=="install":install(Path(args.stage),args.version,args.provider_url,args.provider_secret_file,args.provider_cli)
        elif args.action=="enroll":enroll(args.public_key)
        else:client_config(args.identity,args.known_hosts)
        return 0
    except (Failure,OSError,ValueError,KeyError,subprocess.SubprocessError):
        error=sys.exc_info()[1]
        print(json.dumps({"state":"incomplete","code":str(error) if isinstance(error,Failure) else "INSTALL_OR_CONFIGURATION_FAILED",
                          "evidence":str(STATE),"retry":"Inspect the recorded phase and original processes before rerunning the same install command."}),flush=True)
        return 2


if __name__=="__main__":raise SystemExit(main())
