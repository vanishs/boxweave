#!/usr/bin/env python3
"""Rotate short-lived MCP JWT without exposing it in process arguments or logs."""
import json
import os
from pathlib import Path
import pwd
import subprocess
import sys
import tempfile

CONFIG = Path('/etc/boxweave/config.json')


def main():
    if os.geteuid() != 0:
        sys.exit('Must run as root')
    cfg = json.loads(CONFIG.read_text())
    username = cfg['linux_user']
    pw = pwd.getpwnam(username)
    home = Path(pw.pw_dir)
    token_file = home / '.config' / 'boxweave' / 'containarium-mcp.jwt'
    secret = Path(cfg.get('jwt_secret_file', '/etc/containarium/jwt.secret'))
    if not secret.is_file():
        sys.exit(f'JWT signing secret not found: {secret}')
    binary = cfg.get('containarium_daemon_binary', '/usr/local/bin/containariumd')
    proc = subprocess.run([binary, 'token', 'generate',
                           '--username', f'boxweave-{username}', '--roles', 'admin',
                           '--expiry', '168h', '--secret-file', str(secret), '--raw'],
                          capture_output=True, text=True, check=True, timeout=30)
    token = proc.stdout.strip()
    if not token.startswith('eyJ') or len(token.split('.')) != 3:
        sys.exit('JWT generation returned unexpected output')
    # Create missing per-user directories without changing existing ownership.
    for directory in [home / '.config', token_file.parent]:
        if not directory.exists():
            directory.mkdir(mode=0o700)
            os.chown(directory, pw.pw_uid, pw.pw_gid)
        if directory.is_symlink():
            sys.exit(f'Refusing symlink for credential directory: {directory}')
    fd, tmp = tempfile.mkstemp(prefix='.boxweave-mcp.jwt.', dir=token_file.parent)
    try:
        with os.fdopen(fd, 'w') as f:
            f.write(token + '\n')
            f.flush()
            os.fsync(f.fileno())
        os.chown(tmp, pw.pw_uid, pw.pw_gid)
        os.chmod(tmp, 0o600)
        os.replace(tmp, token_file)
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)
    print(f'Rotated BoxWeave provider JWT for {username} (secret not logged)')


if __name__ == '__main__':
    main()
