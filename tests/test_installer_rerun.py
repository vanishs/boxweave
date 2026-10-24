"""Run the installer repeatedly in a disposable filesystem with fake host tools.

No real packages, systemd units, SSH credentials or Incus instances are touched.
Only the test copy bypasses root ownership and redirects system paths.
"""
import base64
import hashlib
import json
import os
from pathlib import Path
import pwd
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = (ROOT / 'install.sh').read_text(encoding='utf-8')

FAKE_TOOL = r'''#!/usr/bin/python3
import base64, json, os, pathlib, sys
r = pathlib.Path(os.environ['FAKE_ROOT'])
name = pathlib.Path(sys.argv[0]).name
a = sys.argv[1:]
with (r/'calls.jsonl').open('a') as f:
    f.write(json.dumps([name, *a])+'\n')
if name == 'curl':
    url = next(x for x in a if x.startswith('https://') or x.startswith('http://'))
    if os.environ.get('FAIL_DOWNLOAD') and os.environ['FAIL_DOWNLOAD'] in url:
        sys.exit(22)
    if '-w' in a:
        print('401', end=''); sys.exit(0)
    if url.endswith('/latest'):
        print('{"tag_name":"v0.111.0"}'); sys.exit(0)
    if '/commits/' in url:
        print(json.dumps({'sha':'a'*40}));sys.exit(0)
    assets = json.loads((r/'assets.json').read_text())
    key = next((k for k in assets if url.endswith('/'+k)), None)
    if key is None: sys.exit('Unexpected download: '+url)
    pathlib.Path(a[a.index('-o')+1]).write_bytes(base64.b64decode(assets[key]))
elif name == 'getent':
    if a != ['passwd', os.environ['FAKE_USER']]: sys.exit(2)
    print('%s:x:%s:%s::%s:/bin/bash' % (os.environ['FAKE_USER'], os.getuid(), os.getgid(), r/'home'))
elif name == 'incus':
    if a[0] != '--force-local': sys.exit(40)
    if a[1] == 'info': pass
    elif a[1:3] == ['storage','list']: print('[{"name":"default","driver":"zfs"}]')
    else: sys.exit('Unexpected Incus mutation: '+str(a))
elif name == 'systemctl':
    statefile=r/'unit-state.json'
    state=json.loads(statefile.read_text()) if statefile.exists() else {}
    units=[x for x in a[1:] if x.endswith('.service') or x.endswith('.timer')]
    broken = units and (r/'etc/systemd/system'/units[0]).exists() and 'BROKEN' in (r/'etc/systemd/system'/units[0]).read_text()
    if a[0] == 'show':
        if 'ActiveState' in a: print(state.get(units[0],'inactive'))
        elif os.environ.get('MASK_UNIT') in units: print('masked')
        elif broken: print('bad-setting')
        else: print('loaded' if (r/'etc/systemd/system'/units[0]).exists() else 'not-found')
    elif a[0] == 'is-active':
        sys.exit(0 if all(state.get(u)=='active' for u in units) else 3)
    elif a[0] == 'is-failed':
        sys.exit(0 if any(state.get(u)=='failed' for u in units) else 1)
    elif a[0] == 'reset-failed':
        if any(u not in state for u in units): sys.exit(1)
        for u in units:
            if state[u]=='failed': state[u]='inactive'
        statefile.write_text(json.dumps(state))
    elif a[0] in ('start','restart','stop'):
        if a[0] in ('start','restart') and os.environ.get('FAIL_SERVICE') in units:
            state[os.environ['FAIL_SERVICE']]='failed'
            statefile.write_text(json.dumps(state))
            sys.exit(1)
        if a[0]=='stop' and broken: sys.exit(1)
        for u in units: state[u]='inactive' if a[0]=='stop' else 'active'
        statefile.write_text(json.dumps(state))
    elif a[0] not in ('enable','daemon-reload'):
        sys.exit('Unexpected systemctl action')
elif name == 'ssh-keygen':
    if '-A' in a: sys.exit('Unexpected host identity generation')
    if '-lf' in a: print('256 SHA256:TEST test (ED25519)')
    elif '-y' in a: print('ssh-ed25519 TEST')
    else: sys.exit('Unexpected key generation')
elif name == 'runuser':
    os.execvp(a[3], a[3:])
elif name == 'sshd':
    if '-T' in a: print('port 2222')
elif name == 'dpkg': print('amd64')
elif name == 'ip': print('1.1.1.1 via 192.0.2.1 src 192.0.2.2')
elif name == 'apt-get': sys.exit('Unexpected package installation')
elif name == 'systemd-detect-virt': print('lxc')
elif name not in ('containarium','containariumd'): sys.exit('Unexpected fake tool')
'''


class InstallerRerunTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='boxweave-rerun-')
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.user = pwd.getpwuid(os.getuid()).pw_name
        # Existing test accounts may be root in a developer's disposable runner.
        if self.user == 'root':
            self.skipTest('Run these sandbox tests as a non-root Linux user')
        self.bin = self.root/'bin'; self.bin.mkdir()
        for name in ('curl','getent','incus','systemctl','ssh-keygen','runuser','sshd',
                     'dpkg','ip','apt-get','systemd-detect-virt','containarium','containariumd'):
            tool = self.bin/name; tool.write_text(FAKE_TOOL); tool.chmod(0o755)
        self.config = self.root/'etc/boxweave/config.json'
        for directory in ('etc/boxweave', 'etc/systemd/system', 'run/boxweave-install', 'etc/ssh', 'home/.ssh'):
            (self.root/directory).mkdir(parents=True, exist_ok=True)
        (self.root/'etc/os-release').write_text('ID=ubuntu\n')
        (self.root/'etc/ssh/ssh_host_ed25519_key.pub').write_text('ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICIoY7yK5+VYtbzXBFgHguKKWJFTx98DSeDtmDTfS1zA test-public\n')
        for suffix in ('', '.pub'):
            (self.root/('home/.ssh/containarium_ed25519'+suffix)).write_text('TEST FIXTURE\n')
        (self.root/'home/.ssh/containarium_ed25519.pub').write_text('ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICIoY7yK5+VYtbzXBFgHguKKWJFTx98DSeDtmDTfS1zA test-public\n')
        secret = self.root/'signing-file'
        secret.touch()  # Existence only: no real secret or key material.
        self.secret = secret
        text = SCRIPT.replace('[[ $EUID -eq 0 ]]', '[[ 0 -eq 0 ]]')
        for prefix in ('/usr/local','/etc/boxweave','/etc/systemd/system','/var/backups/boxweave',
                       '/run/boxweave-install','/etc/os-release','/etc/ssh',
                       '/etc/apt','/usr/share/keyrings'):
            text = text.replace(prefix, str(self.root)+prefix)
        text = text.replace('/usr/sbin/sshd', str(self.bin/'sshd'))
        text = text.replace('/usr/bin/ssh-keygen', str(self.bin/'ssh-keygen'))
        text = text.replace('-o root -g root', '-o %s -g %s' % (os.getuid(), os.getgid()))
        self.script = self.root/'install.sh'; self.script.write_text(text)
        assets = {name: (ROOT/name).read_bytes() for name in ('install.sh','server/cli.py','server/hostkeys.py','server/tokens.py',
                  'server/clients.py','clients/windows/setup.ps1','clients/posix/setup.py','server/jobs.py','server/mcp_client.py','server/smoke.py','server/provenance.py','VERSION')}
        backend = b'#!/bin/sh\nexit 0\n'
        assets['mcp-server-linux-amd64'] = backend
        assets['SHA256SUMS.txt'] = (hashlib.sha256(backend).hexdigest()+'  mcp-server-linux-amd64\n').encode()
        for path in ('server/web_install.py','server/web_token.py','server/web_entry.py', *[str(x.relative_to(ROOT)) for x in (ROOT/'server/webui').glob('*') if x.is_file()], 'web/package.json','web/package-lock.json','web/tsconfig.json','web/vite.config.ts','web/index.html','web/src/main.tsx','web/src/tenants.tsx','web/src/model.ts','web/src/style.css'):
            assets[path]=(ROOT/path).read_bytes()
        # This harness covers MCP publication only; web setup has its own isolated tests.
        assets['server/web_install.py']=b'import os,sys\nprint("Web provisioner stub: no host mutation")\nif os.environ.get("FAIL_WEB_PHASE")==sys.argv[1]: sys.exit(23)\n'
        self.assets = assets
        self.write_assets()

    def write_assets(self):
        (self.root/'assets.json').write_text(json.dumps({k: base64.b64encode(v).decode() for k,v in self.assets.items()}))

    def run_install(self, args=(), **extra):
        env = {k:v for k,v in os.environ.items() if not k.startswith('BOXWEAVE_') and k != 'CONTAINARIUM_VERSION'}
        env.update(PATH=str(self.bin)+':'+env['PATH'], FAKE_ROOT=str(self.root), FAKE_USER=self.user, SUDO_USER='root')
        env.update(extra)
        if getattr(self,'pipe_mode',False):
            return subprocess.run(['bash','-s','--',*args], input=self.script.read_text(), env=env, capture_output=True, text=True, timeout=25)
        return subprocess.run(['bash', str(self.script), *args], env=env, capture_output=True, text=True, timeout=25)

    def initial_install(self):
        r = self.run_install(('--user',self.user,'--ssh-host','lab.example','--ssh-port','2222',
                              '--api-url','https://127.0.0.1:8443','--allow-cidr','10.47.0.0/24',
                              '--sync-interval','31','--ref','reviewed','--repo','example/boxweave',
                              '--jwt-secret-file',str(self.secret),'--containariumd-binary',str(self.bin/'containariumd')),
                             CONTAINARIUM_VERSION='v0.110.0')
        self.assertEqual(r.returncode, 0, r.stdout+r.stderr)

    def test_reviewed_local_sources_can_install_without_publishing(self):
        self.initial_install()
        local=self.root/'reviewed-source';local.mkdir()
        for name,data in self.assets.items():
            if name.startswith('mcp-server') or name=='SHA256SUMS.txt':continue
            path=local/name;path.parent.mkdir(parents=True,exist_ok=True);path.write_bytes(data)
        result=self.run_install(('--source-dir',str(local)),FAIL_DOWNLOAD='server/cli.py')
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        manifest=json.loads((self.config.parent/'install-manifest.json').read_text())
        self.assertTrue(manifest['source']['local_tree_staged'])

    def test_web_build_failure_preserves_existing_installation(self):
        self.initial_install()
        before=self.snapshot()
        result=self.run_install(FAIL_WEB_PHASE='prepare')
        self.assertEqual(result.returncode,23,result.stderr)
        self.assertEqual(self.snapshot(),before)
        self.assertFalse((self.config.parent/'update-in-progress').exists())

    def test_web_publish_failure_keeps_mcp_timers_running_and_repair_marker(self):
        self.initial_install()
        result=self.run_install(FAIL_WEB_PHASE='publish')
        self.assertEqual(result.returncode,23,result.stderr)
        state=json.loads((self.root/'unit-state.json').read_text())
        for unit in ('boxweave-token.timer','boxweave-hostkeys.timer'):
            self.assertEqual(state[unit],'active')
        self.assertTrue((self.config.parent/'update-in-progress').exists())
        result=self.run_install()
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertFalse((self.config.parent/'update-in-progress').exists())

    def test_public_client_generator_and_final_instructions(self):
        self.initial_install()
        public=self.config.parent/'client.json'
        self.assertEqual(public.stat().st_mode & 0o777,0o644)
        self.assertEqual(self.config.stat().st_mode & 0o777,0o600)
        data=json.loads(public.read_text())
        self.assertNotIn('jwt_secret_file',data)
        self.assertEqual(data['source_commit'],'a'*40)
        self.assertEqual(data['client_sha256']['clients/windows/setup.ps1'],hashlib.sha256(self.assets['clients/windows/setup.ps1']).hexdigest())
        result=subprocess.run(['python3',str(self.root/'usr/local/libexec/boxweave/cli.py'),'client',
                               '--os','macos','--agent','cursor','--config',str(public),'--json'],
                              capture_output=True,text=True,timeout=10)
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertEqual(json.loads(result.stdout)['paste_into'],'macOS Terminal')
        result=self.run_install()
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertIn('boxweave client --help',result.stdout)
        self.assertNotIn('Invoke-WebRequest',result.stdout)

    def snapshot(self):
        prefixes = ('etc/boxweave','etc/systemd/system','usr/local')
        return {str(p.relative_to(self.root)): p.read_bytes() for prefix in prefixes
                for p in (self.root/prefix).rglob('*') if p.is_file()}

    def test_rerun_preserves_settings_identity_and_extensions(self):
        self.initial_install()
        data=json.loads(self.config.read_text()); data['extension']={'keep':True}; data['default_locale']='future-locale'
        self.config.write_text(json.dumps(data))
        # Adding a higher-priority personal identity must not change the saved key.
        (self.root/'home/.ssh/id_ed25519').write_text('NEW FIXTURE')
        (self.root/'home/.ssh/id_ed25519.pub').write_text('NEW FIXTURE')
        before=self.snapshot()
        r=self.run_install()
        self.assertEqual(r.returncode,0,r.stdout+r.stderr)
        self.assertEqual(json.loads(self.config.read_text()),data)
        self.assertEqual(self.snapshot()['etc/systemd/system/boxweave-hostkeys.timer'],before['etc/systemd/system/boxweave-hostkeys.timer'])
        self.assertEqual(self.config.stat().st_mode & 0o777,0o600)
        self.assertFalse((self.config.parent/'update-in-progress').exists())

    def test_explicit_overrides_update_persistently_and_restart_timer(self):
        self.initial_install()
        r=self.run_install(('--sync-interval','43'), BOXWEAVE_SSH_HOST='new.example',
                           CONTAINARIUM_VERSION='latest',BOXWEAVE_REPO='updated/repo',BOXWEAVE_REF='newref')
        self.assertEqual(r.returncode,0,r.stdout+r.stderr)
        data=json.loads(self.config.read_text())
        self.assertEqual((data['sync_interval'],data['ssh_host'],data['source_repo'],data['source_ref']), (43,'new.example','updated/repo','newref'))
        self.assertEqual((data['mcp_release'],data['mcp_version']),('latest','v0.111.0'))
        self.assertIn('OnUnitInactiveSec=43s',(self.root/'etc/systemd/system/boxweave-hostkeys.timer').read_text())
        calls=[json.loads(line) for line in (self.root/'calls.jsonl').read_text().splitlines()]
        self.assertIn(['systemctl','restart','boxweave-token.timer','boxweave-hostkeys.timer'], calls)

    def test_failed_download_or_invalid_source_keeps_installed_files(self):
        self.initial_install()
        before=self.snapshot()
        r=self.run_install(FAIL_DOWNLOAD='server/tokens.py')
        self.assertNotEqual(r.returncode,0)
        self.assertEqual(self.snapshot(),before)
        self.assets['server/cli.py']=b'not valid python !'
        self.write_assets()
        r=self.run_install()
        self.assertNotEqual(r.returncode,0)
        self.assertEqual(self.snapshot(),before)

    def test_failed_verification_leaves_repair_marker_then_rerun_recovers(self):
        self.initial_install()
        r=self.run_install(FAIL_SERVICE='boxweave-token.service')
        self.assertNotEqual(r.returncode,0)
        self.assertTrue((self.config.parent/'update-in-progress').exists())
        states=json.loads((self.root/'unit-state.json').read_text())
        self.assertEqual(states['boxweave-token.timer'],'inactive')
        self.assertEqual(states['boxweave-hostkeys.timer'],'inactive')
        self.assertIn('remain stopped',r.stderr)
        r=self.run_install()
        self.assertEqual(r.returncode,0,r.stdout+r.stderr)
        self.assertFalse((self.config.parent/'update-in-progress').exists())
        states=json.loads((self.root/'unit-state.json').read_text())
        self.assertNotEqual(states['boxweave-token.service'],'failed')

    def test_corrupt_config_is_not_replaced(self):
        self.initial_install()
        self.config.write_text('{broken')
        before=self.snapshot()
        r=self.run_install()
        self.assertNotEqual(r.returncode,0)
        self.assertEqual(self.snapshot(),before)

    def test_upgrade_and_missing_file_repair(self):
        self.initial_install()
        (self.root/'usr/local/sbin/boxweave-hostkeys').unlink()
        self.assets['VERSION']=b'0.1.0-next\n'
        self.write_assets()
        r=self.run_install()
        self.assertEqual(r.returncode,0,r.stdout+r.stderr)
        self.assertEqual((self.root/'usr/local/sbin/boxweave-hostkeys').read_bytes(),self.assets['server/hostkeys.py'])
        self.assertEqual((self.root/'usr/local/libexec/boxweave/VERSION').read_bytes(),b'0.1.0-next\n')
        self.assertTrue(any((self.root/'var/backups/boxweave').glob('update-*/files/*')))
        manifest=json.loads((self.root/'etc/boxweave/install-manifest.json').read_text())
        self.assertEqual(manifest['state'],'complete')
        self.assertEqual(manifest['source']['resolved_commit'],'a'*40)
        self.assertEqual(manifest['boxweave_version'],'0.1.0-next')
        self.assertEqual(manifest['mcp_backend']['sha256'],hashlib.sha256(self.assets['mcp-server-linux-amd64']).hexdigest())

    def test_checksum_failure_and_concurrent_run_do_not_publish(self):
        import fcntl
        self.initial_install()
        before=self.snapshot()
        self.assets['mcp-server-linux-amd64']=b'tampered backend'
        self.write_assets()
        r=self.run_install()
        self.assertNotEqual(r.returncode,0)
        self.assertEqual(self.snapshot(),before)
        with (self.root/'run/boxweave-install/installer.lock').open('a') as lock:
            fcntl.flock(lock,fcntl.LOCK_EX)
            r=self.run_install()
        self.assertNotEqual(r.returncode,0)
        self.assertIn('Another BoxWeave',r.stderr)
        self.assertEqual(self.snapshot(),before)

    def test_legacy_interval_and_broken_unit_are_repaired(self):
        self.initial_install()
        data=json.loads(self.config.read_text()); del data['sync_interval']
        self.config.write_text(json.dumps(data))
        token_unit=self.root/'etc/systemd/system/boxweave-token.service'
        token_unit.write_text('BROKEN UNIT FIXTURE')
        states=json.loads((self.root/'unit-state.json').read_text()); states['boxweave-token.service']='inactive'
        (self.root/'unit-state.json').write_text(json.dumps(states))
        r=self.run_install()
        self.assertEqual(r.returncode,0,r.stdout+r.stderr)
        self.assertEqual(json.loads(self.config.read_text())['sync_interval'],31)
        self.assertIn('ExecStart=',token_unit.read_text())

    def test_masked_managed_unit_is_not_overridden(self):
        self.initial_install()
        before=self.snapshot()
        r=self.run_install(MASK_UNIT='boxweave-token.timer')
        self.assertNotEqual(r.returncode,0)
        self.assertIn('masked',r.stderr)
        self.assertEqual(self.snapshot(),before)

    def test_atomic_replacement_of_running_executable(self):
        target=self.root/'busy'; shutil.copy2('/bin/sleep',target)
        replacement=self.root/'new'; replacement.write_text('#!/bin/sh\nexit 0\n')
        child=subprocess.Popen([str(target),'20'])
        self.addCleanup(lambda: child.poll() is None and child.kill())
        start=self.script.read_text().index('atomic_install() {')
        end=self.script.read_text().index('\npublish_components() {', start)
        function=self.script.read_text()[start:end]
        r=subprocess.run(['bash','-c','set -euo pipefail; die(){ exit 1; }; '+function+'\natomic_install "$1" "$2" 0755',
                          'test',str(replacement),str(target)],capture_output=True,text=True,timeout=10)
        self.assertEqual(r.returncode,0,r.stderr)
        self.assertEqual(target.read_bytes(),replacement.read_bytes())
        self.assertIsNone(child.poll())
        child.terminate(); child.wait(timeout=5)

    def test_stdin_installer_records_unknown_source_without_reading_pipe(self):
        self.pipe_mode=True
        self.initial_install()
        manifest=json.loads((self.root/'etc/boxweave/install-manifest.json').read_text())
        self.assertEqual(manifest['state'],'complete')
        self.assertIsNone(manifest['installer']['sha256'])
        self.assertIsNone(manifest['installer']['differs_from_source'])


if __name__ == '__main__':
    unittest.main()
