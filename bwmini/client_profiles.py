"""Cross-platform BoxWeave MCP client profiles and self-contained setup generator.

Project artifacts and generated client instructions are English. The catalog
order is alphabetical, regardless of the order profiles were introduced.
No code downloads, private keys, Provider credentials, or Codex AGENTS edits.
"""
import json
import re

# Maintain strict alphabetical ordering (case-insensitive) in every UI/document.
APPS = ("antigravity", "claude", "codex", "copilot", "cursor",
        "opencode", "qoder", "trae", "zcode")
SYSTEMS = ("linux", "macos", "windows")
TRANSPORTS = ("native", "wsl")
# Qoder and TRAE expose documented IDE JSON import, not a stable global file.
MANUAL = frozenset(("qoder", "trae"))
DOCS = {
    "antigravity": "https://antigravity.google/docs/mcp",
    "claude": "https://code.claude.com/docs/en/mcp",
    "codex": "https://developers.openai.com/codex/mcp",
    "copilot": "https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers",
    "cursor": "https://cursor.com/docs/mcp",
    "opencode": "https://opencode.ai/docs/mcp-servers/",
    "qoder": "https://docs.qoder.com/user-guide/chat/model-context-protocol",
    "trae": "https://docs.trae.cn/ide_add-mcp-servers",
    "zcode": "https://zcode.z.ai/en/docs/mcp-services",
}
assert tuple(sorted(APPS, key=str.casefold)) == APPS

# This is emitted as client-side Python 3.11+ and never run as root.
PORTABLE = r'''#!/usr/bin/env python3
# BoxWeave MCP setup: __LABEL__.
# Requires local Python 3.11+, OpenSSH client and ssh-keygen.
# Windows WSL transport also needs a selected WSL distribution and Python3 inside it.
# Review the EDIT ME fields. Run: python3 boxweave-setup.py [--already-enrolled | --no-enroll]
# On Windows use: py -3 boxweave-setup.py [--already-enrolled | --no-enroll]
# No BoxWeave project AGENTS.md files are created or modified.
import argparse, base64, json, os, pathlib, re, shutil, stat, subprocess, sys, tempfile, uuid
SETTINGS = __SETTINGS__
# EDIT ME: choose the client-reachable IP/DNS and SSH port. Do NOT change
# the verified host key without checking the real server out of band.
HOST = SETTINGS["host"]
PORT = SETTINGS["port"]
SERVER_KEY = SETTINGS["server_key"]
ADMIN_USER = SETTINGS["admin_user"]
APP = SETTINGS["app"]
SYSTEM = SETTINGS["os"]
TRANSPORT = SETTINGS["transport"]
REPLACE = SETTINGS["replace"]
REMOTE = "/usr/bin/python3 /usr/local/lib/boxweave/default/mcp_gate.py"
SERVER_USER = "boxweave"

def require(value, message):
    if not value: raise ValueError(message)

def checked(argv, **kwargs):
    completed = subprocess.run(argv, check=False, **kwargs)
    require(completed.returncode == 0, "Command failed: " + str(argv[0]))
    return completed

def safe_dir(path):
    require(not path.is_symlink(), "Refusing symlinked directory")
    path.mkdir(parents=True, exist_ok=True, mode=0o700)
    require(path.is_dir(), "Expected directory")
    if os.name != "nt":
        require(path.stat().st_uid == os.getuid() and not path.stat().st_mode & 0o077,
                "Private directory ownership/mode required")

def safe_file(path):
    require(not path.is_symlink(), "Refusing a symlinked file")
    if path.exists():
        require(path.is_file() and path.stat().st_nlink == 1,
                "Regular single-link file required")
        if os.name != "nt":
            require(path.stat().st_uid == os.getuid(), "User-owned file required")

def posix_identity(host, port, host_key):
    root = pathlib.Path.home()/".config"/"boxweave"
    safe_dir(root.parent)
    safe_dir(root)
    private = root/"id_ed25519"
    public = root/"id_ed25519.pub"
    pin = root/"known_hosts"
    for path in (private, public, pin): safe_file(path)
    require(private.exists() == public.exists(), "Partial SSH keypair refused")
    if not private.exists():
        checked(["ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", str(private),
                 "-C", "boxweave-client"])
    if os.name != "nt":
        require(stat.S_IMODE(private.stat().st_mode) == 0o600, "Private key must be mode 0600")
    token = host if port == 22 else "["+host+"]:"+str(port)
    expected = token+" "+host_key+"\n"
    if pin.exists():
        require(pin.read_text() == expected, "SSH host pin differs; review manually")
    else:
        with os.fdopen(os.open(pin, os.O_WRONLY|os.O_CREAT|os.O_EXCL, 0o600), "w") as f:
            f.write(expected)
    if os.name != "nt":
        require(stat.S_IMODE(pin.stat().st_mode) == 0o600, "Host pin must be mode 0600")
    key = " ".join(public.read_text().strip().split()[:2])
    require(re.fullmatch(r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}", key) is not None,
            "Invalid public key")
    return {"key":str(private), "known_hosts":str(pin), "public_key":key}

WSL_IDENTITY = r"""
import json, os, pathlib, re, stat, subprocess, sys
host, port, host_key = sys.argv[1:]
base = pathlib.Path.home()
assert base.is_dir() and not base.is_symlink() and base.stat().st_uid == os.getuid()
config = base/".config"
assert not config.is_symlink()
config.mkdir(mode=0o700,exist_ok=True)
root = config/"boxweave"
assert not root.is_symlink()
root.mkdir(mode=0o700,exist_ok=True)
assert root.stat().st_uid == os.getuid() and not root.stat().st_mode & 0o077
key = root/"id_ed25519"
public = root/"id_ed25519.pub"
pin = root/"known_hosts"
assert all(not p.is_symlink() for p in (key,public,pin))
assert key.exists() == public.exists()
if not key.exists():
    subprocess.run(["/usr/bin/ssh-keygen","-q","-t","ed25519","-N","","-f",str(key),
                    "-C","boxweave-client-wsl"],check=True)
assert stat.S_IMODE(key.stat().st_mode) == 0o600
token = host if int(port)==22 else "["+host+"]:"+port
expected = token+" "+host_key+"\n"
if pin.exists(): assert pin.read_text()==expected, "SSH host pin differs"
else:
    with os.fdopen(os.open(pin,os.O_CREAT|os.O_EXCL|os.O_WRONLY,0o600),"w") as f:
        f.write(expected)
assert stat.S_IMODE(pin.stat().st_mode)==0o600
pubkey=" ".join(public.read_text().strip().split()[:2])
assert re.fullmatch(r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}",pubkey)
print(json.dumps({"key":str(key),"known_hosts":str(pin),"public_key":pubkey}))
"""

def client_ssh_base(identity, distro):
    if SYSTEM=="windows" and TRANSPORT=="wsl":
        return ["wsl.exe","--distribution",distro,"--exec","/usr/bin/ssh"]
    return ["ssh"]

def ssh_opts(identity):
    return ["-F", "/dev/null", "-T", "-o", "BatchMode=yes",
            "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=yes",
            "-o", "ConnectTimeout=15", "-o", "ServerAliveInterval=15",
            "-o", "ServerAliveCountMax=4",
            "-o", "UserKnownHostsFile="+identity["known_hosts"],
            "-i",identity["key"], "-p",str(PORT)]

def client_identity(distro):
    if SYSTEM=="windows" and TRANSPORT=="wsl":
        result=checked(["wsl.exe","--distribution",distro,"--exec",
                        "/usr/bin/python3","-c",WSL_IDENTITY,
                        HOST,str(PORT),SERVER_KEY],capture_output=True,text=True,timeout=60)
        return json.loads(result.stdout.strip())
    return posix_identity(HOST, PORT, SERVER_KEY)

def transport_command(identity, distro):
    base=client_ssh_base(identity,distro)
    return base[0], base[1:]+ssh_opts(identity)+[SERVER_USER+"@"+HOST,REMOTE]

def enroll(identity,distro):
    base=client_ssh_base(identity,distro)
    # Administrator password login is separate from the restricted MCP key.
    opts=["-F","/dev/null","-tt","-o","StrictHostKeyChecking=yes",
          "-o","IdentityAgent=none","-o","PubkeyAuthentication=no",
          "-o","PreferredAuthentications=password,keyboard-interactive",
          "-o","UserKnownHostsFile="+identity["known_hosts"],"-p",str(PORT)]
    pub=identity["public_key"]
    require(re.fullmatch(r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}",pub) is not None,
            "Malformed client public key")
    remote="sudo /usr/bin/python3 /usr/local/lib/boxweave/default/online.py enroll --public-key '"+pub+"'"
    checked(base+opts+[ADMIN_USER+"@"+HOST,remote],timeout=120)

def probe(identity,distro):
    command,args=transport_command(identity,distro)
    requests=[
        {"jsonrpc":"2.0","id":1,"method":"initialize",
         "params":{"protocolVersion":"2024-11-05","capabilities":{},
                   "clientInfo":{"name":"boxweave-client","version":"1.0"}}},
        {"jsonrpc":"2.0","method":"notifications/initialized"},
        {"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}},
        {"jsonrpc":"2.0","id":3,"method":"tools/call",
         "params":{"name":"list_containers","arguments":{}}}]
    result=checked([command]+args,input="".join(json.dumps(x)+"\n" for x in requests),
                   text=True,capture_output=True,timeout=90)
    messages=[json.loads(line) for line in result.stdout.splitlines() if line.strip()]
    by_id={x.get("id"):x for x in messages if isinstance(x,dict) and "id" in x}
    require(by_id[1].get("result",{}).get("protocolVersion") == "2024-11-05",
            "MCP initialize failed")
    require(any(t.get("name")=="list_containers" for t in
                by_id[2].get("result",{}).get("tools",[])),
            "MCP list_containers tool not found")
    read=by_id[3].get("result",{})
    require(not read.get("isError") and read.get("content"),"MCP read failed")
    print("Authenticated MCP initialize/tools/list/list_containers: passed")
    return command,args

def app_entry(command, args):
    if APP=="copilot":
        return {"type":"local","command":command,"args":args,"tools":["*"]}
    if APP=="opencode":
        return {"type":"local","command":[command]+args,"enabled":True,"timeout":180000}
    if APP=="zcode":
        return {"type":"stdio","command":command,"args":args,"timeoutMs":180000}
    return {"command":command,"args":args}

def app_location():
    home=pathlib.Path.home()
    if APP=="antigravity": return home/".gemini"/"config"/"mcp_config.json", ("mcpServers",)
    if APP=="claude": return home/".claude.json", ("mcpServers",)
    if APP=="copilot": return pathlib.Path(os.environ.get("COPILOT_HOME",str(home/".copilot")))/"mcp-config.json", ("mcpServers",)
    if APP=="cursor": return home/".cursor"/"mcp.json", ("mcpServers",)
    if APP=="opencode": return home/".config"/"opencode"/"opencode.json", ("mcp",)
    if APP=="zcode": return home/".zcode"/"cli"/"config.json", ("mcp","servers")
    raise ValueError("App uses manual JSON import")

def update_json(command,args):
    entry=app_entry(command,args)
    if APP in ("qoder","trae"):
        print("Import the following JSON in the app's MCP settings; no files modified:")
        print(json.dumps({"mcpServers":{"boxweave":entry}},indent=2))
        if APP=="trae": print("TRAE: Settings > MCP > Add > Manual, or project .trae/mcp.json")
        if APP=="qoder": print("Qoder IDE: Settings > MCP > My Servers > Add")
        return
    path,keys=app_location()
    require(path.is_absolute(), "App configuration home must be absolute")
    safe_dir(path.parent)
    safe_file(path)
    root=json.loads(path.read_text(encoding="utf-8")) if path.exists() else {}
    require(isinstance(root,dict),"App configuration must be a JSON object")
    section=root
    for key in keys:
        if key not in section: section[key]={}
        section=section[key]
        require(isinstance(section,dict),"Malformed MCP configuration object")
    old=section.get("boxweave")
    if old==entry:
        print("BoxWeave MCP registration already matches: "+str(path)); return
    require(old is None or REPLACE,
            "BoxWeave MCP entry differs; regenerate with --replace after review")
    section["boxweave"]=entry
    backup=path.with_name(path.name+".boxweave-"+uuid.uuid4().hex+".bak")
    if path.exists():
        with os.fdopen(os.open(backup,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600),"wb") as fd:
            fd.write(path.read_bytes())
    with os.fdopen(os.open(path.parent/(".boxweave-"+uuid.uuid4().hex+".tmp"),
                             os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600),"w",encoding="utf-8") as fd:
        temp=pathlib.Path(fd.name) if isinstance(fd.name,str) else None
        fd.write(json.dumps(root,indent=2,ensure_ascii=False)+"\n")
    # Use an owned, freshly created staging file for atomic replacement.
    # The temporary path was created above but fd.name is an integer on Unix;
    # retrieve it by tracking its path directly in the current directory.
    candidates=list(path.parent.glob(".boxweave-*.tmp"))
    require(len(candidates)==1, "Unexpected staging files; manual cleanup required")
    candidates[0].replace(path)
    print("BoxWeave MCP configured: "+str(path))

def update_codex(command,args):
    import tomllib
    path=pathlib.Path(os.environ.get("CODEX_HOME",str(pathlib.Path.home()/".codex")))/"config.toml"
    require(path.is_absolute(),"Codex home must be absolute")
    safe_dir(path.parent); safe_file(path)
    original=path.read_text(encoding="utf-8") if path.exists() else ""
    parsed=tomllib.loads(original)
    root=parsed.get("mcp_servers",{})
    require(isinstance(root,dict),"Invalid Codex MCP configuration")
    entry=root.get("boxweave")
    require(entry is None or isinstance(entry,dict),"Invalid BoxWeave config")
    expected={"command":command,"args":args,"startup_timeout_sec":30,"tool_timeout_sec":180}
    if entry and all(entry.get(k)==v for k,v in expected.items()):
        print("BoxWeave MCP already matches: "+str(path));return
    require(entry is None or REPLACE,
            "BoxWeave MCP transport differs; regenerate with --replace after review")
    import re
    # Only replace the root section, never unrelated servers or nested tables.
    pattern=r"(?ms)^[ \t]*\[mcp_servers\.boxweave\][^\r\n]*\r?\n.*?(?=^[ \t]*\[|\Z)"
    matches=list(re.finditer(pattern,original))
    require(len(matches)<=1,"Duplicate BoxWeave MCP tables")
    require(entry is None or len(matches)==1,"Ambiguous BoxWeave TOML; manual review required")
    require(not (entry is None and re.search(r"(?m)^\[mcp_servers\.boxweave\.",original)),
            "BoxWeave nested-only TOML; manual review")
    block="[mcp_servers.boxweave]\n"+"".join(
        k+" = "+json.dumps(v)+"\n" for k,v in expected.items())
    if matches:
        m=matches[0];updated=original[:m.start()]+block+original[m.end():]
    else:
        updated=original.rstrip()+"\n\n"+block
    require(tomllib.loads(updated)["mcp_servers"]["boxweave"]["args"]==args,
            "Invalid generated TOML")
    if path.exists():
        backup=path.with_name(path.name+".boxweave-"+uuid.uuid4().hex+".bak")
        with os.fdopen(os.open(backup,os.O_CREAT|os.O_EXCL|os.O_WRONLY,0o600),"wb") as f:
            f.write(path.read_bytes())
    temp=path.parent/(".boxweave-"+uuid.uuid4().hex+".tmp")
    with os.fdopen(os.open(temp,os.O_CREAT|os.O_EXCL|os.O_WRONLY,0o600),"w",encoding="utf-8") as f:
        f.write(updated)
    temp.replace(path)
    print("BoxWeave MCP configured: "+str(path))

def main():
    parser=argparse.ArgumentParser(description="Verified BoxWeave MCP client setup")
    flags=parser.add_mutually_exclusive_group()
    flags.add_argument("--already-enrolled",action="store_true")
    flags.add_argument("--no-enroll",action="store_true")
    parser.add_argument("--distro",default="",help="Required when multiple Windows WSL distros exist")
    options=parser.parse_args()
    require(SYSTEM in ("linux","macos","windows"),"Unsupported OS")
    require((SYSTEM=="windows") or TRANSPORT=="native","WSL only supported from Windows")
    require(re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9.-]{0,252}",HOST),"Unsafe host")
    require(type(PORT) is int and 1<=PORT<=65535,"Invalid port")
    require(re.fullmatch(r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}",SERVER_KEY),"Invalid pinned server key")
    require(re.fullmatch(r"[a-z_][a-z0-9_-]{0,30}",ADMIN_USER),"Invalid administrator")
    require(sys.version_info[:2]>=(3,11),"Python 3.11+ required on client")
    distro=options.distro
    if SYSTEM=="windows" and TRANSPORT=="wsl":
        require(shutil.which("wsl.exe"),"wsl.exe not found")
        if not distro:
            found=checked(["wsl.exe","--list","--quiet"],capture_output=True).stdout
            names=[v.strip() for v in found.decode("utf-16-le").splitlines() if v.strip()]
            require(len(names)==1,"Choose an installed WSL distro with --distro NAME")
            distro=names[0]
        require(re.fullmatch(r"[A-Za-z0-9_.-]{1,80}",distro),"Invalid WSL distro")
    identity=client_identity(distro)
    if options.no_enroll:
        print("On the Linux BoxWeave host, authorize the following public key:")
        print("sudo python3 /usr/local/lib/boxweave/default/online.py enroll --public-key '"+
              identity["public_key"]+"'")
        print("Then rerun this script with --already-enrolled.")
        return
    if not options.already_enrolled: enroll(identity,distro)
    command,args=probe(identity,distro)
    if APP=="codex": update_codex(command,args)
    else: update_json(command,args)
    print("Restart your MCP client and verify the BoxWeave tools appear.")

if __name__=="__main__":
    try: main()
    except (ValueError, OSError, KeyError, json.JSONDecodeError,
            subprocess.TimeoutExpired, subprocess.CalledProcessError) as exc:
        print("BoxWeave setup failed: "+str(exc),file=sys.stderr)
        raise SystemExit(2)
'''

def profile_valid(app, system, transport):
    return (app in APPS and system in SYSTEMS and transport in TRANSPORTS
            and (system == "windows" or transport == "native"))

def render(settings):
    """Return a client-side Python program; all server data remains public."""
    if not profile_valid(settings["app"], settings["os"], settings["transport"]):
        raise ValueError("UNSUPPORTED_CLIENT_PROFILE")
    host, port, host_key = settings["host"], settings["port"], settings["server_key"]
    if (not isinstance(host,str) or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9.-]{0,252}",host)
        or type(port) is not int or not 1<=port<=65535
        or not isinstance(host_key,str)
        or not re.fullmatch(r"ssh-ed25519 [A-Za-z0-9+/]+={0,2}",host_key)
        or not re.fullmatch(r"[a-z_][a-z0-9_-]{0,30}",settings["admin_user"])
        or type(settings["replace"]) is not bool):
        raise ValueError("PUBLIC_CLIENT_IDENTITY_INVALID")
    value=repr(dict(settings))
    return PORTABLE.replace("__SETTINGS__",value).replace(
        "__LABEL__",settings["app"]+" / "+settings["os"]+" / "+settings["transport"])
