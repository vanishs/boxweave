"""Read-only client command generation and the terminal selection wizard."""
import argparse
import base64
import hashlib
import json
from pathlib import Path
import re
import shlex
import sys

DEFAULT_CONFIG = Path('/etc/boxweave/client.json')
AGENTS = ('codex', 'cursor', 'claude-code')
POLICY = """## BoxWeave remote development (Containarium)
- Use MCP server boxweave to list/provision containers and its dynamic connect tool to work inside a box.
- Develop, install dependencies, test and deploy INSIDE the assigned project LXC. Inspect existing files/Git state and preserve uncommitted work.
- Inspect existing resources first; reuse a suitable project box. Never modify unrelated or containarium-core-* boxes.
- The Ubuntu MCP endpoint is {ssh_host}:{ssh_port}, user {linux_user}. The transport key is restricted to the MCP launcher, not a general host shell credential.
- For create_container.ssh_keys, explicitly supply this FULL managed Ubuntu-to-container public key:
  {container_public_key}
- Never substitute a client transport key, a fingerprint, or another nesting layer's key. Never read/export private keys or JWTs.
- A create timeout is ambiguous: inspect the original logical name before retrying; never duplicate or force-recreate to bypass a timeout.
- Running does not prove SSH readiness. Use one deadline per phase (normally 600 seconds), poll every 15-30 seconds, and verify connect with whoami && hostname before project work. Report actual stdout and exit codes.
- After IP reuse, allow bounded trusted host-key reconciliation. Never disable host verification or edit known_hosts to bypass a mismatch. Diagnose persistent failures read-only.
- Service activity, HTTP 200/401/403 and CLI exit 0 do not prove authenticated MCP execution. Test initialize/initialized, tools/list and an actual read-only tools/call.
- Keep identities separate at each nested MCP host; direct Incus execution is not evidence of nested authenticated MCP execution.
- Use systemd/Compose only for requested persistent workloads; verify health and application autostart. LXC boot.autostart requires separate evidence.
- Publish ports only when requested. Inspect routes/conflicts and verify actual client connectivity. Passthrough does not provide LAN-only source filtering or TLS.
- Obtain explicit approval for destructive/host-wide changes or public exposure. Never change host SSH/firewall/provider credentials to make a test pass.
"""

def public_key(value):
    if not isinstance(value, str) or len(value) > 8192 or '\n' in value or '\r' in value:
        raise ValueError('Invalid public key')
    parts = value.split()
    if len(parts) < 2 or parts[0] not in ('ssh-ed25519', 'ssh-rsa', 'ecdsa-sha2-nistp256'):
        raise ValueError('Invalid public key')
    try:
        blob = base64.b64decode(parts[1], validate=True)
        size = int.from_bytes(blob[:4], 'big')
        if blob[4:4+size].decode('ascii') != parts[0] or len(blob) < size+12:
            raise ValueError()
    except (ValueError, UnicodeError):
        raise ValueError('Invalid public key') from None
    return value

def validate(data):
    if not isinstance(data, dict) or data.get('schema_version') != 1:
        raise ValueError('Unsupported client descriptor')
    fields = {'ssh_host': r'[A-Za-z0-9][A-Za-z0-9.:-]{0,252}',
              'linux_user': r'[a-z_][a-z0-9_-]{0,63}',
              'launcher': r'/[A-Za-z0-9_./-]+',
              'source_repo': r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+',
              'source_commit': r'[a-f0-9]{40}'}
    for key, pattern in fields.items():
        if not isinstance(data.get(key), str) or not re.fullmatch(pattern, data[key]):
            raise ValueError('Invalid descriptor field: '+key)
    if type(data.get('ssh_port')) is not int or not 1 <= data['ssh_port'] <= 65535:
        raise ValueError('Invalid SSH port')
    for key in ('server_public_key', 'container_public_key'):
        public_key(data.get(key))
    if not data['server_public_key'].startswith('ssh-ed25519 '):
        raise ValueError('Server host key must be ED25519')
    assets = data.get('client_sha256')
    required = ('clients/windows/setup.ps1', 'clients/posix/setup.py')
    if not isinstance(assets, dict) or any(not re.fullmatch(r'[a-f0-9]{64}', str(assets.get(p,''))) for p in required):
        raise ValueError('Missing validated client asset hashes')
    return data

def publish(config, destination, staged, commit, host_key):
    """Called by the staged installer; do not read any credential contents."""
    config, staged = Path(config), Path(staged)
    d = json.loads(config.read_text())
    data = {'schema_version': 1, 'ssh_host': d['ssh_host'], 'ssh_port': d['ssh_port'],
            'linux_user': d['linux_user'], 'launcher': '/usr/local/bin/boxweave-mcp',
            'source_repo': d['source_repo'], 'source_commit': commit,
            'server_public_key': Path(host_key).read_text().strip(),
            'container_public_key': Path(d['container_public_key_file']).read_text().strip(),
            'client_sha256': {p: hashlib.sha256((staged/p).read_bytes()).hexdigest()
                              for p in ('clients/windows/setup.ps1', 'clients/posix/setup.py')}}
    validate(data)
    Path(destination).write_text(json.dumps(data, indent=2)+'\n')

def add_parser(commands):
    p = commands.add_parser('client', help='Generate a client setup command; interactive by default')
    p.add_argument('--os', choices=('windows','macos'))
    p.add_argument('--agent', choices=AGENTS)
    p.add_argument('--runtime', choices=('native','wsl'))
    p.add_argument('--transport', choices=('native','wsl'))
    p.add_argument('--config', type=Path, default=DEFAULT_CONFIG, help='Public client descriptor')
    p.add_argument('--ssh-host', help='Override rendered address only')
    p.add_argument('--ssh-port', type=int)
    p.add_argument('--bootstrap-identity', help='Existing client SSH identity path (never read by the server)')
    p.add_argument('--agent-home', help='Client agent configuration directory')
    p.add_argument('--project-path', help='Optional local Cursor workspace for its rule')
    p.add_argument('--wsl-distro', help='Distribution for native Windows to WSL SSH')
    p.add_argument('--replace', action='store_true', help='Replace a reviewed boxweave MCP registration')
    p.add_argument('--json', action='store_true', default=argparse.SUPPRESS)

def choose(title, options):
    print('\n'+title, file=sys.stderr)
    for i, (label, _) in enumerate(options, 1):
        print('  {}) {}'.format(i,label), file=sys.stderr)
    while True:
        print('Select [1-{}]: '.format(len(options)), end='', file=sys.stderr, flush=True)
        answer = sys.stdin.readline()
        if not answer: raise EOFError()
        if answer.strip().isdigit() and 1 <= int(answer) <= len(options):
            return options[int(answer)-1][1]
        print('Enter a listed number.', file=sys.stderr)

def selection(args):
    interactive = sys.stdin.isatty() and not getattr(args,'json',False)
    def need(value, title, options, flag):
        if value: return value
        if not interactive: raise ValueError('Missing '+flag+'; use an interactive terminal or pass explicit options')
        return choose(title,options)
    os_name = need(args.os, 'Choose your client OS', [('Windows','windows'),('macOS','macos')], '--os')
    agent = need(args.agent, 'Choose your agent', [(a,a) for a in AGENTS], '--agent')
    runtime = args.runtime
    transport = args.transport
    if os_name == 'macos':
        runtime, transport = runtime or 'native', transport or 'native'
    else:
        if not runtime:
            runtime = need(runtime,'Where does the agent run?',
                           [('Windows native app / CLI','native')]+
                           ([('CLI inside WSL','wsl')] if agent != 'cursor' else []),'--runtime')
        if runtime == 'wsl':
            transport = transport or 'wsl'
        else:
            transport = need(transport,'Choose SSH transport',
                             [('Windows OpenSSH (PowerShell setup)','native'),
                              ('OpenSSH inside WSL (PowerShell setup)','wsl')],'--transport')
    if os_name == 'macos' and (runtime != 'native' or transport != 'native'):
        raise ValueError('macOS supports native runtime and SSH only')
    if runtime == 'wsl' and (agent == 'cursor' or transport != 'wsl'):
        raise ValueError('WSL runtime supports Codex/Claude Code CLI with WSL SSH only')
    return {'os':os_name,'agent':agent,'runtime':runtime,'transport':transport}

def psquote(s):
    return "'"+str(s).replace("'","''")+"'"

def render(d, profile, options=None):
    options = options or {}
    payload = {k:d[k] for k in ('ssh_host','ssh_port','linux_user','launcher','server_public_key','container_public_key')}
    payload.update(profile)
    payload['setup_options'] = options
    payload['policy'] = POLICY.format(**d)
    encoded = base64.b64encode(json.dumps(payload).encode()).decode()
    base = 'https://raw.githubusercontent.com/{}/{}'.format(d['source_repo'],d['source_commit'])
    if profile['os'] == 'windows' and profile['runtime'] == 'native':
        asset = 'clients/windows/setup.ps1'
        command = ("$bwDir = Join-Path ([IO.Path]::GetTempPath()) ('boxweave-' + [guid]::NewGuid()); "
                   "$null = New-Item -ItemType Directory -Path $bwDir; try { "
                   "$bwFile = Join-Path $bwDir 'setup.ps1'; "
                   "Invoke-WebRequest -UseBasicParsing "+psquote(base+'/'+asset)+" -OutFile $bwFile -ErrorAction Stop; "
                   "if ((Get-FileHash -Algorithm SHA256 -LiteralPath $bwFile).Hash.ToLowerInvariant() -ne "+
                   psquote(d['client_sha256'][asset])+") { throw 'BoxWeave client checksum mismatch' }; "
                   "& $bwFile -DataBase64 "+psquote(encoded)+" "
                   "} finally { Remove-Item -LiteralPath $bwDir -Recurse -Force }")
        # Launch a child with process-scoped policy so .ps1 works under standard Windows defaults.
        command = command.replace("& $bwFile -DataBase64", "& powershell.exe -NoProfile -ExecutionPolicy Bypass -File $bwFile -DataBase64")
        command = command.replace(" } finally", "; if ($LASTEXITCODE -ne 0) { throw 'BoxWeave client setup failed' } } finally")
        shell = 'Windows PowerShell'
    else:
        asset = 'clients/posix/setup.py'
        script = ("set -eu; command -v python3 >/dev/null || { echo 'Python 3.11+ is required; install it before setup' >&2; exit 2; }; "
                  "python3 -c 'import sys; sys.exit(0 if sys.version_info >= (3,11) else 2)'; "
                  "bw_dir=$(mktemp -d); trap 'rm -rf \"$bw_dir\"' EXIT; "
                  "curl -fsSL "+shlex.quote(base+'/'+asset)+" -o \"$bw_dir/setup.py\"; "
                  "python3 -c "+shlex.quote("import hashlib,pathlib,sys; sys.exit(0 if hashlib.sha256(pathlib.Path(sys.argv[1]).read_bytes()).hexdigest()==sys.argv[2] else 'BoxWeave client checksum mismatch')")+
                  " \"$bw_dir/setup.py\" "+shlex.quote(d['client_sha256'][asset])+"; "
                  "python3 \"$bw_dir/setup.py\" --data-base64 "+shlex.quote(encoded))
        command = 'bash -c '+shlex.quote(script)
        shell = 'WSL terminal (the Linux user running the CLI)' if profile['runtime']=='wsl' else 'macOS Terminal'
    return {'profile':profile,'paste_into':shell,'command':command,
            'notes':['Setup uses an existing authorized SSH login for public-key enrollment.',
                     'Cursor User Rules require a manual Settings step unless a project rule is explicitly installed.',
                     'Python 3.11+ is required for macOS / WSL CLI setup; nothing is installed automatically.']
                    if asset.endswith('.py') else
                    ['Setup uses an existing authorized SSH login; WSL distribution selection occurs on this client.',
                     'Cursor global User Rules require the displayed manual Settings step.']}

def run(args):
    try:
        profile = selection(args)
        if sys.stdin.isatty() and not getattr(args,'json',False) and not args.replace:
            args.replace = choose('Replace an existing BoxWeave MCP registration?',
                                  [('No: preserve conflicting entries', False),
                                   ('Yes: replace the BoxWeave entry (backups retained)', True)])
        try:
            if args.config.stat().st_size > 65536: raise ValueError('Client descriptor too large')
            data = json.loads(args.config.read_text(encoding='utf-8'))
        except OSError:
            raise ValueError('Public client descriptor unavailable; update the server installer or pass --config PATH') from None
        if args.ssh_host: data['ssh_host'] = args.ssh_host
        if args.ssh_port is not None: data['ssh_port'] = args.ssh_port
        options = {key:getattr(args,key,None) for key in ('bootstrap_identity','agent_home','project_path','wsl_distro','replace')}
        if options['wsl_distro'] and not (profile['os']=='windows' and profile['runtime']=='native' and profile['transport']=='wsl'):
            raise ValueError('--wsl-distro requires a native Windows agent with WSL SSH')
        if options['project_path'] and profile['agent']!='cursor':
            raise ValueError('--project-path is only supported for Cursor')
        if options['agent_home'] and profile['agent']=='claude-code':
            raise ValueError('For Claude Code set CLAUDE_CONFIG_DIR on the client')
        result = render(validate(data),profile,options)
        if getattr(args,'json',False): print(json.dumps(result,indent=2))
        else:
            print('\nPaste into '+result['paste_into']+':\n\n'+result['command']+'\n')
            for note in result['notes']:
                for line in note.splitlines(): print('# '+line)
        return 0
    except (KeyboardInterrupt,EOFError):
        print('\nClient command generation cancelled.',file=sys.stderr); return 130
    except (ValueError,TypeError,KeyError) as error:
        print('boxweave client: '+str(error),file=sys.stderr); return 2

if __name__ == '__main__':
    if len(sys.argv) == 7 and sys.argv[1] == 'publish':
        publish(*sys.argv[2:])
    else:
        raise SystemExit('Use boxweave client --help')
