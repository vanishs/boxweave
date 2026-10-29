"""Generate a fresh identity inside the explicitly selected WSL distro; public output only."""
import argparse
import json
import os
from pathlib import Path
import re
import stat
import subprocess

def prepare(directory,host,port,server_key):
    p=Path(directory)
    if not p.is_absolute() or p.exists() or not p.parent.is_dir():raise ValueError('Fresh absolute directory required')
    for parent in (p.parent,*p.parent.parents):
        if parent.is_symlink() or parent.stat().st_mode&0o022:raise ValueError('Trusted non-writable parents required')
    if p.parent.stat().st_uid!=os.getuid():raise ValueError('Owned parent required')
    if not re.fullmatch(r'[A-Za-z0-9.-]{1,253}',host) or type(port) is not int or not 1<=port<=65535:raise ValueError('Endpoint invalid')
    if not re.fullmatch(r'ssh-ed25519 [A-Za-z0-9+/]+={0,2}',server_key):raise ValueError('Reviewed public pin required')
    p.mkdir(mode=0o700)
    subprocess.run(['/usr/bin/ssh-keygen','-q','-t','ed25519','-N','','-C','boxweave-wsl-profile','-f',str(p/'id_ed25519')],check=True,stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=30)
    os.chmod(p/'id_ed25519',0o600)
    token=host if port==22 else '['+host+']:'+str(port)
    (p/'known_hosts').write_text(token+' '+server_key+'\n');os.chmod(p/'known_hosts',0o600)
    if stat.S_IMODE((p/'id_ed25519').stat().st_mode)!=0o600:raise ValueError('Private mode invalid')
    return {'identity_file':str(p/'id_ed25519'),'known_hosts':str(p/'known_hosts'),'public_key':(p/'id_ed25519.pub').read_text().strip(),'enrollment':'pending_public_key_authorization'}
if __name__=='__main__':
    a=argparse.ArgumentParser(description=__doc__);a.add_argument('--request',required=True);args=a.parse_args()
    r=json.loads(args.request);print(json.dumps(prepare(r['directory'],r['host'],r['port'],r['server_key'])))
