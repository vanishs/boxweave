"""Client enrollment-script generation for BoxWeave.

The generated artifact is a plain, readable script with an explicit EDIT ME
header (SSH host, port, user, pinned server key). The values detected by the
server installer are only a starting point; the address that matters is the
one reachable from the client machine, so the user edits the header before
running. The enrollment logic itself is fetched from the pinned source commit
at run time, so a rerun of the script always repairs with that release's code.
"""
import argparse
import base64
import json
from pathlib import Path
import re
import shlex
import sys

DEFAULT_CONFIG = Path('/etc/boxweave/client.json')
LAUNCHER = '/usr/local/bin/boxweave-mcp'
AGENTS = ('codex', 'cursor', 'claude-code', 'zcode')


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
    if not isinstance(data, dict) or data.get('schema_version') != 2:
        raise ValueError('Unsupported client descriptor; reinstall the server (schema 2 required)')
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
    return data


def publish(config, destination, commit, host_key):
    """Called by the staged installer; do not read any credential contents."""
    d = json.loads(Path(config).read_text())
    data = {'schema_version': 2, 'ssh_host': d['ssh_host'], 'ssh_port': d['ssh_port'],
            'linux_user': d['linux_user'], 'launcher': LAUNCHER,
            'source_repo': d['source_repo'], 'source_commit': commit,
            'server_public_key': Path(host_key).read_text().strip(),
            'container_public_key': Path(d['container_public_key_file']).read_text().strip()}
    validate(data)
    Path(destination).write_text(json.dumps(data, indent=2)+'\n')


def add_parser(commands):
    p = commands.add_parser('client', help='Generate a client enrollment script; interactive by default')
    p.add_argument('--os', choices=('windows', 'macos'))
    p.add_argument('--agent', choices=AGENTS)
    p.add_argument('--claude-provider', choices=('zai', 'zhipu'), help='Configure GLM on the client; key is read from client GLM_API_KEY')
    p.add_argument('--glm-model', help='Optional GLM model identifier for all Claude model tiers')
    p.add_argument('--runtime', choices=('native', 'wsl'))
    p.add_argument('--transport', choices=('native', 'wsl'))
    p.add_argument('--config', type=Path, default=DEFAULT_CONFIG, help='Public client descriptor')
    p.add_argument('--output', type=Path, help='Write the enrollment script to this file instead of stdout')
    p.add_argument('--replace', action='store_true', help='Bake replacement of an existing BoxWeave MCP registration into the script')
    p.add_argument('--json', action='store_true', default=argparse.SUPPRESS)


def choose(title, options):
    print('\n'+title, file=sys.stderr)
    for i, (label, _) in enumerate(options, 1):
        print('  {}) {}'.format(i, label), file=sys.stderr)
    while True:
        print('Select [1-{}]: '.format(len(options)), end='', file=sys.stderr, flush=True)
        answer = sys.stdin.readline()
        if not answer: raise EOFError()
        if answer.strip().isdigit() and 1 <= int(answer) <= len(options):
            return options[int(answer)-1][1]
        print('Enter a listed number.', file=sys.stderr)


def selection(args):
    interactive = sys.stdin.isatty() and not getattr(args, 'json', False)
    def need(value, title, options, flag):
        if value: return value
        if not interactive: raise ValueError('Missing '+flag+'; use an interactive terminal or pass explicit options')
        return choose(title, options)
    os_name = need(args.os, 'Choose your client OS', [('Windows', 'windows'), ('macOS', 'macos')], '--os')
    agent = need(args.agent, 'Choose your agent', [(a, a) for a in AGENTS], '--agent')
    runtime = args.runtime
    transport = args.transport
    if os_name == 'macos':
        runtime, transport = runtime or 'native', transport or 'native'
    else:
        if not runtime:
            runtime = need(runtime, 'Where does the agent run?',
                           [('Windows native app / CLI', 'native')]+
                           ([('CLI inside WSL', 'wsl')] if agent not in ('cursor', 'zcode') else []), '--runtime')
        if runtime == 'wsl':
            transport = transport or 'wsl'
        else:
            transport = need(transport, 'Choose SSH transport',
                             [('Windows OpenSSH (PowerShell setup)', 'native'),
                              ('OpenSSH inside WSL (PowerShell setup)', 'wsl')], '--transport')
    if os_name == 'macos' and (runtime != 'native' or transport != 'native'):
        raise ValueError('macOS supports native runtime and SSH only')
    if runtime == 'wsl' and (agent in ('cursor', 'zcode') or transport != 'wsl'):
        raise ValueError('WSL runtime supports Codex/Claude Code CLI with WSL SSH only')
    return {'os': os_name, 'agent': agent, 'runtime': runtime, 'transport': transport}


def bash_quote(s):
    return shlex.quote(str(s))


def ps_quote(s):
    return "'" + str(s).replace("'", "''") + "'"


POSIX_TEMPLATE = '''#!/usr/bin/env bash
# BoxWeave client enrollment script, generated by `boxweave client`.
# Enrollment logic source: @@BASE@@/clients/posix/setup.py
#
# How to use:
#   1. Review the EDIT ME block below. BOXWEAVE_SSH_HOST/PORT must be the
#      address actually reachable from THIS client machine; the generated
#      values come from the server installer and may be an internal address.
#      After enrollment, the same address can be changed later by editing the
#      "Host boxweave" block in ~/.ssh/config and rerunning this script.
#   2. Run: bash @@FILENAME@@
#      Flags pass through to the enrollment logic: --diagnose
#      --report PATH --bootstrap-identity PATH --agent-home PATH --project-path PATH
#      Replacement mode chosen at generation time is already baked into the
#      invocation below (the --replace flag still works at run time).
set -eu

# ===== EDIT ME: BoxWeave server connection =====
BOXWEAVE_SSH_HOST=@@SSH_HOST@@
BOXWEAVE_SSH_PORT=@@SSH_PORT@@
BOXWEAVE_SSH_USER=@@SSH_USER@@
# Pinned Ubuntu SSH host key (verify the fingerprint out-of-band on first use):
BOXWEAVE_SERVER_HOST_KEY=@@HOST_KEY@@
# ================================================

# Managed Ubuntu-to-container key; pass verbatim to create_container.ssh_keys.
BOXWEAVE_CONTAINER_PUBLIC_KEY=@@CONTAINER_KEY@@

# Enrollment profile chosen when this script was generated.
BOXWEAVE_AGENT=@@AGENT@@
BOXWEAVE_RUNTIME=@@RUNTIME@@
BOXWEAVE_TRANSPORT=@@TRANSPORT@@
# Optional Claude Code GLM settings; credentials stay on this client
# (BOXWEAVE_CLAUDE_PROVIDER requires GLM_API_KEY in the environment at run time).
BOXWEAVE_CLAUDE_PROVIDER=@@PROVIDER@@
BOXWEAVE_GLM_MODEL=@@MODEL@@

export BOXWEAVE_SSH_HOST BOXWEAVE_SSH_PORT BOXWEAVE_SSH_USER BOXWEAVE_SERVER_HOST_KEY
export BOXWEAVE_AGENT BOXWEAVE_RUNTIME BOXWEAVE_TRANSPORT
export BOXWEAVE_CLAUDE_PROVIDER BOXWEAVE_GLM_MODEL

command -v python3 >/dev/null || { echo 'Python 3.11+ is required; install it before setup' >&2; exit 2; }
python3 -c 'import sys; sys.exit(0 if sys.version_info >= (3,11) else 2)' || { echo 'Python 3.11+ is required' >&2; exit 2; }
command -v curl >/dev/null || { echo 'curl is required to fetch the enrollment logic' >&2; exit 2; }
bw_dir=$(mktemp -d) || exit 1
trap 'rm -rf "$bw_dir"' EXIT
curl -fsSL --retry 3 @@BASE@@/clients/posix/setup.py -o "$bw_dir/setup.py" || exit 1
python3 "$bw_dir/setup.py"@@EXTRA_ARGS@@ "$@"
'''

WINDOWS_TEMPLATE = '''#requires -version 5.1
# BoxWeave client enrollment script, generated by `boxweave client`.
# Enrollment logic source: @@BASE@@/clients/windows/setup.ps1
#
# How to use:
#   1. Review the EDIT ME block below. BOXWEAVE_SSH_HOST/PORT must be the
#      address actually reachable from THIS client machine; the generated
#      values come from the server installer and may be an internal address.
#      After enrollment, the same address can be changed later by editing the
#      "Host boxweave" block in the SSH config and rerunning this script.
#   2. Run: powershell -NoProfile -ExecutionPolicy Bypass -File @@FILENAME@@
#      Switches pass through to the enrollment logic: -Diagnose
#      -ReportPath PATH -BootstrapIdentityFile PATH -AgentHome PATH -ProjectPath PATH
#      Replacement mode chosen at generation time is already baked into the
#      invocation below (the -Replace switch still works at run time).
$ErrorActionPreference = 'Stop'

# ===== EDIT ME: BoxWeave server connection =====
$env:BOXWEAVE_SSH_HOST = @@SSH_HOST@@
$env:BOXWEAVE_SSH_PORT = @@SSH_PORT@@
$env:BOXWEAVE_SSH_USER = @@SSH_USER@@
# Pinned Ubuntu SSH host key (verify the fingerprint out-of-band on first use):
$env:BOXWEAVE_SERVER_HOST_KEY = @@HOST_KEY@@
# ================================================

# Managed Ubuntu-to-container key; pass verbatim to create_container.ssh_keys.
$env:BOXWEAVE_CONTAINER_PUBLIC_KEY = @@CONTAINER_KEY@@

# Enrollment profile chosen when this script was generated.
$env:BOXWEAVE_AGENT = @@AGENT@@
$env:BOXWEAVE_TRANSPORT = @@TRANSPORT@@   # Native | WSL
$env:BOXWEAVE_WSL_DISTRO = @@WSL_DISTRO@@  # empty = auto-select
# Optional Claude Code GLM settings; credentials stay on this client
# (BOXWEAVE_CLAUDE_PROVIDER requires GLM_API_KEY in the environment at run time).
$env:BOXWEAVE_CLAUDE_PROVIDER = @@PROVIDER@@
$env:BOXWEAVE_GLM_MODEL = @@MODEL@@

$bwDir = Join-Path ([IO.Path]::GetTempPath()) ('boxweave-' + [guid]::NewGuid())
$null = New-Item -ItemType Directory -Path $bwDir
try {
  $bwFile = Join-Path $bwDir 'setup.ps1'
  Invoke-WebRequest -UseBasicParsing @@BASE@@/clients/windows/setup.ps1 -OutFile $bwFile -ErrorAction Stop
  & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $bwFile@@EXTRA_ARGS@@ @args
  if ($LASTEXITCODE -ne 0) { throw 'BoxWeave client setup failed' }
} finally { Remove-Item -LiteralPath $bwDir -Recurse -Force }
'''


def render(d, profile, options=None):
    options = options or {}
    base = 'https://raw.githubusercontent.com/{}/{}'.format(d['source_repo'], d['source_commit'])
    windows_native = profile['os'] == 'windows' and profile['runtime'] == 'native'
    quote = ps_quote if windows_native else bash_quote
    replace_args = ' -Replace' if windows_native else ' --replace'
    values = {'@@BASE@@': base,
              '@@FILENAME@@': 'boxweave-setup.ps1' if windows_native else 'boxweave-setup.sh',
              '@@SSH_HOST@@': quote(d['ssh_host']),
              '@@SSH_PORT@@': str(d['ssh_port']),
              '@@SSH_USER@@': quote(d['linux_user']),
              '@@HOST_KEY@@': quote(d['server_public_key']),
              '@@CONTAINER_KEY@@': quote(d['container_public_key']),
              '@@AGENT@@': quote(profile['agent']),
              '@@RUNTIME@@': quote(profile['runtime']),
              '@@TRANSPORT@@': quote('WSL' if profile['transport'] == 'wsl' else 'Native') if windows_native else quote(profile['transport']),
              '@@WSL_DISTRO@@': ps_quote(''),
              '@@PROVIDER@@': quote(options.get('claude_provider') or ''),
              '@@MODEL@@': quote(options.get('glm_model') or ''),
              '@@EXTRA_ARGS@@': replace_args if options.get('replace') else ''}
    if windows_native:
        script = WINDOWS_TEMPLATE
        shell = 'Windows PowerShell'
        notes = ['Setup uses an existing authorized SSH login; WSL distribution selection occurs on this client.',
                 'Change the address later by editing the "Host boxweave" block in the SSH config and rerunning the script.']
    else:
        script = POSIX_TEMPLATE
        shell = 'WSL terminal (the Linux user running the CLI)' if profile['runtime'] == 'wsl' else 'macOS Terminal'
        notes = ['Setup uses an existing authorized SSH login for public-key enrollment.',
                 'Python 3.11+ and curl are required on this client; nothing is installed automatically.',
                 'Change the address later by editing the "Host boxweave" block in ~/.ssh/config and rerunning the script.']
    for token, value in values.items():
        script = script.replace(token, value)
    return {'profile': profile, 'filename': values['@@FILENAME@@'],
            'paste_into': shell, 'script': script, 'notes': notes}


def run(args):
    try:
        profile = selection(args)
        provider = getattr(args, 'claude_provider', None)
        model = getattr(args, 'glm_model', None)
        if (provider or model) and profile['agent'] != 'claude-code':
            raise ValueError('GLM options require Claude Code setup')
        if model and (not provider or not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_.:\[\]-]{0,127}', model)):
            raise ValueError('--glm-model requires --claude-provider and a valid model identifier')
        try:
            if args.config.stat().st_size > 65536: raise ValueError('Client descriptor too large')
            data = json.loads(args.config.read_text(encoding='utf-8'))
        except OSError:
            raise ValueError('Public client descriptor unavailable; update the server installer or pass --config PATH') from None
        options = {}
        if provider: options['claude_provider'] = provider
        if model: options['glm_model'] = model
        # The replacement question is the last interactive step; the answer is
        # baked into the generated script so rerunning it just works.
        if getattr(args, 'replace', False):
            options['replace'] = True
        elif sys.stdin.isatty() and not getattr(args, 'json', False):
            if choose('Replace an existing BoxWeave MCP registration?',
                      [('No: preserve conflicting entries', False),
                       ('Yes: replace the BoxWeave entry (backups retained)', True)]):
                options['replace'] = True
        result = render(validate(data), profile, options)
        if getattr(args, 'json', False):
            print(json.dumps(result, indent=2))
        elif getattr(args, 'output', None):
            output = Path(args.output)
            if output.is_symlink(): raise ValueError('Refusing output symlink: '+str(output))
            output.parent.mkdir(parents=True, exist_ok=True)
            output.write_text(result['script'], encoding='utf-8')
            print('Wrote '+str(output)+' ('+result['filename']+').', file=sys.stderr)
            print('Review the EDIT ME block, then run it on the client machine.', file=sys.stderr)
        else:
            print('# Save as '+result['filename']+' and run inside '+result['paste_into']+':', file=sys.stderr)
            for note in result['notes']:
                print('# '+note, file=sys.stderr)
            print(result['script'], end='')
        return 0
    except (KeyboardInterrupt, EOFError):
        print('\nClient script generation cancelled.', file=sys.stderr); return 130
    except (ValueError, TypeError, KeyError) as error:
        print('boxweave client: '+str(error), file=sys.stderr); return 2


if __name__ == '__main__':
    if len(sys.argv) == 6 and sys.argv[1] == 'publish':
        publish(*sys.argv[2:])
    else:
        raise SystemExit('Use boxweave client --help')
