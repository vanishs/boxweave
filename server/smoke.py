"""Plan and run a confirmed inner-MCP create/connect/Python smoke test."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import socket
import subprocess
import time
import uuid

from jobs import private_directory, read_json
from mcp_client import Client, MCPError
from provenance import atomic_json, digest, utc


def plan_hash(plan):
    return hashlib.sha256(json.dumps(plan, sort_keys=True, separators=(',',':')).encode()).hexdigest()


def public_key(path):
    path=Path(path)
    if path.suffix!='.pub' or not path.is_file() or path.stat().st_size>16384:
        raise ValueError('A bounded public .pub file is required')
    text = path.read_text(encoding='utf-8').strip()
    if len(text) > 16384 or not re.fullmatch(r'ssh-(?:ed25519|rsa) [A-Za-z0-9+/=]+(?: [^\r\n]*)?',text):
        raise ValueError('Expected a single full public key line')
    r = subprocess.run(['ssh-keygen','-lf',str(path),'-E','sha256'],capture_output=True,text=True,timeout=5,check=True)
    return text, r.stdout.split()[1]


def make_plan(box, key, launcher, directory, reuse=False, phase_timeout=600):
    if not re.fullmatch(r'[a-z][a-z0-9-]{0,40}',box) or box.startswith('containarium-core-'): raise ValueError('Invalid workload name')
    if not 1 <= phase_timeout <= 600: raise ValueError('Phase timeout must be 1-600 seconds')
    launcher = Path(launcher).resolve(); key = Path(key).resolve()
    _, fp = public_key(key)
    if not launcher.is_file() or not os.access(launcher,os.X_OK): raise ValueError('Executable MCP launcher required')
    p = private_directory(directory)
    plan = {'schema_version':1,'plan_id':uuid.uuid4().hex,'host':socket.gethostname(),
            'launcher':str(launcher),'launcher_sha256':digest(launcher),'public_key_file':str(key),
            'public_key_fingerprint':fp,'box':box,'image':'images:ubuntu/24.04',
            'cpu':'1','memory':'1GB','disk':'8GB','reuse_existing':reuse,
            'phase_timeout_seconds':phase_timeout,'cleanup':'none; leave workload available',
            'steps':['discover inner MCP tools','reconcile inventory','create only if absent and never previously requested',
                     'wait for Running','connect: whoami && hostname','connect: python3 hello world'],
            'success':'exact workload identity, stdout hello world and exit_code 0'}
    target = p / 'plan.json'
    if target.exists(): raise ValueError('Plan already exists; choose a new directory')
    atomic_json(target,plan)
    return {'plan':plan,'path':str(target),'approval_sha256':plan_hash(plan),
            'next':'Review target, changes and time budget with the user; run only after confirmation.'}


def texts(result):
    return '\n'.join(x['text'] for x in result.get('content',[]) if x.get('type')=='text')


def inventory_has(result, box):
    text = texts(result).strip()
    if text == 'No containers found.': return False
    if not re.match(r'^Found \d+ container\(s\):',text): raise MCPError('Unrecognized inventory format')
    return bool(re.search(r'(?m)^\s*\U0001f4e6\s+'+re.escape(box+'-container')+r'\s*$',text))


def metadata(result, box):
    data = json.loads(texts(result))['container']
    if data.get('name') != box+'-container' or data.get('username') != box or not data.get('createdAt'):
        raise MCPError('Container identity mismatch or missing creation identity')
    return {key:data.get(key) for key in ('name','username','createdAt','state')}


def execution(result):
    # This parser is ONLY for the fixed smoke commands, not arbitrary guest output.
    text = texts(result)
    match = re.fullmatch(r'exit_code: (-?\d+)\n\n--- stdout ---\n(.*)',text,re.S)
    if not match: raise MCPError('Unrecognized connect result format')
    return int(match.group(1)), match.group(2)


def run(plan_path, approval, client_factory=Client):
    import fcntl
    path = Path(plan_path).resolve(); p = private_directory(path.parent)
    plan = read_json(path)
    if approval != plan_hash(plan): raise ValueError('Approval digest does not match the reviewed plan')
    if plan['host'] != socket.gethostname() or digest(plan['launcher']) != plan['launcher_sha256']:
        raise ValueError('Plan host or launcher changed; generate and approve a new plan')
    key, fp = public_key(plan['public_key_file'])
    if fp != plan['public_key_fingerprint']: raise ValueError('Public identity changed')
    with open(p/'run.lock','a') as lock:
        fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
        receipt_path = p/'result.json'
        receipt = read_json(receipt_path) if receipt_path.exists() else {'schema_version':1,'plan_sha256':approval,'state':'pending','created_at':utc()}
        monotonic_deadlines={}
        if receipt['plan_sha256'] != approval: raise ValueError('Receipt belongs to another plan')
        if receipt.get('state') == 'succeeded': return receipt
        client = None
        def save(): atomic_json(receipt_path,receipt)
        def phase(name):
            if receipt.get('phase') != name:
                receipt.update(phase=name, phase_deadline_epoch=time.time()+plan['phase_timeout_seconds'])
                monotonic_deadlines[name]=time.monotonic()+plan['phase_timeout_seconds']
                save()
        def budget():
            wall_remaining=receipt['phase_deadline_epoch']-time.time()
            name=receipt['phase']
            if name not in monotonic_deadlines:
                monotonic_deadlines[name]=time.monotonic()+max(0,wall_remaining)
            value=min(wall_remaining,monotonic_deadlines[name]-time.monotonic())
            if value <= 0: raise TimeoutError('Phase deadline exceeded; no automatic deadline reset')
            return value
        box=plan['box']
        try:
            client=client_factory(plan['launcher'])
            available={x['name'] for x in client.call('tools/list',{},20)['tools']}
            if not {'list_containers','get_container','create_container','connect'} <= available:
                raise MCPError('Required MCP tools unavailable')
            receipt['server']=client.initial.get('serverInfo',{})
            phase('provisioning') if 'phase' not in receipt else None
            if not receipt.get('create_requested') and not receipt.get('identity'):
                exists=inventory_has(client.tool('list_containers',{},min(30,budget())),box)
                if exists and not plan['reuse_existing']: raise MCPError('Existing workload requires an approved reuse plan')
                if not exists:
                    receipt.update(create_requested=True,create_requested_epoch=time.time()); save()
                    try:
                        client.tool('create_container',{'username':box,'image':plan['image'],'cpu':plan['cpu'],
                                    'memory':plan['memory'],'disk':plan['disk'],'enable_podman':False,'ssh_keys':[key]},min(150,budget()))
                    except (MCPError,TimeoutError):
                        # Ambiguous create is reconciled, never automatically resubmitted.
                        receipt['create_response']='unknown';save()
            if not receipt.get('identity'):
                while True:
                    budget()
                    d=None
                    try:
                        d=metadata(client.tool('get_container',{'username':box},min(30,budget())),box)
                    except (MCPError,ValueError,KeyError):
                        pass
                    if d:
                        if receipt.get('create_requested') and float(d['createdAt']) < receipt['create_requested_epoch']-5:
                            raise ValueError('Observed workload predates this create request')
                        if d['state']=='CONTAINER_STATE_RUNNING':
                            receipt['identity']=d; save(); break
                    time.sleep(min(15,budget()))
            d=metadata(client.tool('get_container',{'username':box},min(30,budget())),box)
            if any(d[k]!=receipt['identity'][k] for k in ('name','username','createdAt')):
                raise MCPError('Workload replaced since previous observation')
            phase('execution')
            while True:
                budget()
                try:
                    code, output=execution(client.tool('connect',{'box':box,'exec':'whoami && hostname'},min(45,budget())))
                    if code!=0 or output.splitlines()!=[box,box+'-container']: raise MCPError('Connect identity mismatch')
                    break
                except (MCPError,TimeoutError):
                    # Read-only diagnosis; no service restarts or host-key bypasses.
                    if 'debug_container' in available:
                        try: client.tool('debug_container',{'username':box},min(15,budget()))
                        except (MCPError,TimeoutError): pass
                    time.sleep(min(15,budget()))
            code, output=execution(client.tool('connect',{'box':box,'exec':'python3 -c "print(\'hello world\')"'},min(45,budget())))
            if code!=0 or output!='hello world\n': raise MCPError('Python result did not match the smoke contract')
            receipt.update(state='succeeded',exit_code=code,stdout=output,finished_at=utc());save()
        except Exception as error:
            receipt.update(state='timed_out' if isinstance(error,TimeoutError) else 'incomplete',
                           error=type(error).__name__,exit_code=None,updated_at=utc());save()
            if isinstance(error,(MCPError,TimeoutError)):
                receipt['detail']=str(error);save()
        finally:
            if client: client.close()
        return receipt


def main(argv=None):
    p=argparse.ArgumentParser(description=__doc__)
    sub=p.add_subparsers(dest='action',required=True)
    plan=sub.add_parser('plan');plan.add_argument('--box',required=True);plan.add_argument('--public-key',type=Path,required=True)
    plan.add_argument('--launcher',type=Path,default=Path('/usr/local/bin/boxweave-mcp'))
    plan.add_argument('--directory',type=Path,required=True);plan.add_argument('--reuse-existing',action='store_true')
    plan.add_argument('--phase-timeout',type=int,default=600)
    execute=sub.add_parser('run');execute.add_argument('plan',type=Path);execute.add_argument('--approve',required=True)
    query=sub.add_parser('status');query.add_argument('plan',type=Path)
    args=p.parse_args(argv)
    try:
        if args.action=='plan': result=make_plan(args.box,args.public_key,args.launcher,args.directory,args.reuse_existing,args.phase_timeout)
        elif args.action=='run': result=run(args.plan,args.approve)
        else: result=read_json(args.plan.parent/'result.json')
        print(json.dumps(result,indent=2));return 0 if result.get('state') in (None,'succeeded') else 2
    except (OSError,ValueError,subprocess.SubprocessError) as error:
        print(json.dumps({'error':type(error).__name__,'message':'Plan or state unavailable; no automatic recreation.'}));return 2


if __name__=='__main__': raise SystemExit(main())
