#!/usr/bin/env python3
"""Synchronize Incus-authenticated SSH server keys for Containarium project boxes.

Executes as root (for trusted Incus access); writes the user's SSH file via runuser.
Configuration: /etc/boxweave/config.json
"""
import argparse
import fcntl
import ipaddress
import json
import os
from pathlib import Path
import pwd
import subprocess
import sys

CONFIG = Path('/etc/boxweave/config.json')
TYPES = {'ed25519': 'ssh-ed25519', 'ecdsa': 'ecdsa-sha2-nistp256', 'rsa': 'ssh-rsa'}

# This fragment runs only as the target unprivileged user, not as root. Input is
# fully verified public key data; lock + atomic replace avoid partial writes.
USER_WRITE = r'''
import fcntl, os, pathlib, shutil, subprocess, sys, tempfile, time
ip = sys.argv[1]
sshdir = pathlib.Path.home() / '.ssh'
sshdir.mkdir(mode=0o700, exist_ok=True)
sshdir.chmod(0o700)
known = sshdir / 'known_hosts'
lines = sys.stdin.read()
with open(sshdir / '.boxweave-hostkeys.lock', 'a+') as lock:
    fcntl.flock(lock, fcntl.LOCK_EX)
    fd, tmp = tempfile.mkstemp(prefix='.known_hosts.sync.', dir=sshdir)
    try:
        with os.fdopen(fd, 'wb') as out:
            if known.exists():
                out.write(known.read_bytes())
            out.flush()
            os.fsync(out.fileno())
        os.chmod(tmp, 0o600)
        subprocess.run(['ssh-keygen', '-R', ip, '-f', tmp],
                       stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, check=True)
        pathlib.Path(tmp + '.old').unlink(missing_ok=True)
        with open(tmp, 'ab') as out:
            out.write(lines.encode('ascii'))
            out.flush()
            os.fsync(out.fileno())
        if known.exists():
            stamp = time.strftime('%Y%m%d-%H%M%S') + '-' + str(os.getpid())
            backup = sshdir / ('known_hosts.backup-boxweave-' + stamp)
            shutil.copy2(known, backup)
            backup.chmod(0o600)
        os.replace(tmp, known)
    finally:
        pathlib.Path(tmp).unlink(missing_ok=True)
'''


def cmd(argv, timeout=25, input_data=None, check=True):
    return subprocess.run(argv, input=input_data, text=True, capture_output=True,
                          timeout=timeout, check=check)


def as_json(argv):
    return json.loads(cmd(argv).stdout)


def get_ipv4_networks(config):
    # Network CIDRs are discovered from Incus itself, not hardcoded for a home LAN.
    explicit = config.get('allowed_container_cidrs') or []
    cidrs = [ipaddress.ip_network(c, strict=False) for c in explicit]
    if not cidrs:
        for net in as_json(['incus', 'network', 'list', '--format', 'json']):
            if net.get('type') not in ('bridge', 'ovn'):
                continue
            if not net.get('managed', False):
                continue
            value = (net.get('config') or {}).get('ipv4.address', '')
            if value and value.lower() not in ('auto', 'none'):
                cidrs.append(ipaddress.ip_network(value, strict=False))
    # Fail closed: never sync public or loopback IPs.
    cidrs = [x for x in cidrs if x.version == 4 and x.is_private
             and not x.is_link_local and not x.is_loopback]
    if not cidrs:
        raise ValueError('No trusted private Incus bridge subnet found; set allowed_container_cidrs')
    return cidrs


def ip_for(container, cidrs):
    network = (container.get('state') or {}).get('network') or {}
    eth = network.get('eth0') or {}
    values = []
    for entry in eth.get('addresses') or []:
        if entry.get('family') != 'inet' or entry.get('scope') != 'global':
            continue
        ip = ipaddress.ip_address(entry['address'])
        if any(ip in block for block in cidrs):
            values.append(str(ip))
    if len(values) != 1:
        raise ValueError(f'Expected one trusted eth0 IPv4 address, got {values}')
    return values[0]


def project(container):
    name = container.get('name', '')
    cfg = container.get('config') or {}
    return (name.endswith('-container') and not name.startswith('containarium-core-')
            and 'user.containarium.label.os-type' in cfg
            and 'user.containarium.sshd_policy' in cfg
            and container.get('status', '').lower() == 'running')


def key_from_incus(name, keytype, algorithm):
    pubpath = f'/etc/ssh/ssh_host_{keytype}_key.pub'
    result = cmd(['incus', 'exec', name, '--', 'cat', pubpath])
    bits = result.stdout.split()
    if len(bits) < 2 or bits[0] != algorithm:
        raise ValueError(f'Invalid Incus {keytype} public key')
    return bits[1]


def existing_keys(username, known, ip):
    if not known.exists():
        return set()
    result = cmd(['runuser', '-u', username, '--', 'ssh-keygen', '-F', ip,
                  '-f', str(known)], timeout=10, check=False)
    if result.returncode not in (0, 1):
        raise ValueError(f'Failed to read known_hosts: {result.stderr.strip()}')
    pairs = set()
    for line in result.stdout.splitlines():
        parts = line.split()
        if not line.startswith('#') and len(parts) >= 3:
            pairs.add((parts[1], parts[2]))
    return pairs


def synchronize_one(container, username, home, cidrs, apply):
    name = container['name']
    cfg = container['config']
    uuid = cfg.get('volatile.uuid', '')
    if not uuid:
        raise ValueError('Missing Incus UUID')
    ip = ip_for(container, cidrs)
    expected = {}
    for keytype, algorithm in TYPES.items():
        expected[algorithm] = key_from_incus(name, keytype, algorithm)
    desired = set(expected.items())
    known = home / '.ssh' / 'known_hosts'
    if existing_keys(username, known, ip) == desired:
        print(f'OK      {name} {ip} UUID={uuid}', flush=True)
        return ip

    scan = cmd(['ssh-keyscan', '-T', '5', '-t', 'ed25519,ecdsa,rsa', ip], timeout=25)
    observed = set()
    for line in scan.stdout.splitlines():
        parts = line.split()
        if not line.startswith('#') and len(parts) >= 3:
            observed.add((parts[1], parts[2]))
    if observed != desired:
        raise ValueError('Network-presented host keys differ from trusted Incus host keys')

    # Reject IP or instance-identity changes between verification and update.
    fresh = as_json(['incus', 'list', name, '--format', 'json'])
    if len(fresh) != 1 or fresh[0].get('name') != name:
        raise ValueError('Instance disappeared during verification')
    if fresh[0].get('config', {}).get('volatile.uuid') != uuid:
        raise ValueError('Instance UUID changed during verification')
    if ip_for(fresh[0], cidrs) != ip:
        raise ValueError('Instance IP changed during verification')

    print(f'{"UPDATE" if apply else "DRY-RUN"} {name} {ip} UUID={uuid}', flush=True)
    if apply:
        content = ''.join(f'{ip} {alg} {key}\n' for alg, key in expected.items())
        cmd(['runuser', '-u', username, '--', 'python3', '-c', USER_WRITE, ip],
            timeout=40, input_data=content)
    return ip


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--apply', action='store_true', help='Apply validated changes')
    parser.add_argument('--config', default=str(CONFIG))
    args = parser.parse_args()
    if os.geteuid() != 0:
        parser.error('Root privileges are required for Incus verification')
    config = json.loads(Path(args.config).read_text())
    username = config['linux_user']
    if not (username and username != 'root'):
        parser.error('The target SSH user must not be root')
    pw = pwd.getpwnam(username)
    home = Path(pw.pw_dir)
    containers = as_json(['incus', 'list', '--format', 'json'])
    projects = [c for c in containers if project(c)]
    if not projects:
        # A clean install may have no project LXC and no managed bridge yet.
        print('OK      no running project containers')
        return 0
    cidrs = get_ipv4_networks(config)
    seen_ips = set()
    failures = 0
    for box in projects:
        try:
            ip = ip_for(box, cidrs)
            if ip in seen_ips:
                raise ValueError('Duplicate IP shared by two project containers')
            seen_ips.add(ip)
            synchronize_one(box, username, home, cidrs, args.apply)
        except (ValueError, subprocess.CalledProcessError, subprocess.TimeoutExpired,
                OSError, json.JSONDecodeError) as exc:
            failures += 1
            print(f'ERROR   {box.get("name", "unknown")}: {exc}', file=sys.stderr)
    return 1 if failures else 0


if __name__ == '__main__':
    sys.exit(main())
