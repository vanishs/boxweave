"""Actual Windows four-profile acceptance; use only dedicated, already reviewed test profiles."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile


def main():
    a=argparse.ArgumentParser(description=__doc__);a.add_argument('--descriptors',nargs=4,required=True);a.add_argument('--result',required=True);args=a.parse_args()
    assert os.name=='nt','Actual Windows required'
    root=Path(__file__).resolve().parents[1];helper=root/'clients/profile_client.py'
    records=[]
    def invoke(path,action='diagnose',replace=False):
        argv=[sys.executable,str(helper),action,'--descriptor',str(path)]
        if replace:argv.append('--replace')
        r=subprocess.run(argv,capture_output=True,text=True,encoding='utf-8',timeout=180)
        assert not r.stderr, 'Unexpected diagnostics; inspect locally without publishing raw output'
        return r.returncode,json.loads(r.stdout)
    for item in args.descriptors:
        path=Path(item).resolve();d=json.loads(path.read_text(encoding='utf-8'))
        assert d['source_sha256']==hashlib.sha256(helper.read_bytes()).hexdigest(),'Descriptor must be independently reviewed before running this check'
        profile=Path(d['profile_directory']);config=profile/('config.toml' if d['profile'].startswith('codex/') else '.zcode/config.json')
        # Explicit replacement applies only to these reviewed test profiles.
        code,first=invoke(path,'register',True);assert code==0,first.get('code')
        before=config.read_bytes()
        code,second=invoke(path,'register');assert code==0 and not second['write']['changed']
        code,diagnosis=invoke(path);assert code==0 and diagnosis['connection']['result']=='passed'
        assert config.read_bytes()==before,'Diagnosis changed config'
        negatives=[]
        with tempfile.TemporaryDirectory(prefix='boxweave-profile-negative-') as folder:
            for field,value,expected in [('source_sha256','0'*64,'PAYLOAD_MISMATCH'),('agent_version','not-the-installed-version','AGENT_VERSION_CHANGED')]:
                bad=dict(d);bad[field]=value;p=Path(folder)/'public-descriptor.json';p.write_text(json.dumps(bad),encoding='utf-8')
                code,r=invoke(p,'register',True);assert code!=0 and r['code']==expected
                assert config.read_bytes()==before;negatives.append(expected)
            bad=json.loads(json.dumps(d));bad['connection']['server_key']='ssh-ed25519 AAAA';p.write_text(json.dumps(bad),encoding='utf-8')
            code,r=invoke(p,'register',True);assert code!=0 and config.read_bytes()==before
            negatives.append('wrong_pin_stopped_before_config_write')
            if d['profile'].endswith('/wsl'):
                bad=json.loads(json.dumps(d));bad['connection']['distribution']='NotInstalled-BoxWeave-NoFallback';p.write_text(json.dumps(bad),encoding='utf-8')
                code,r=invoke(p,'register',True);assert code!=0 and config.read_bytes()==before
                negatives.append('missing_distro_no_install_or_fallback')
        records.append({'profile':d['profile'],'agent_version':d['agent_version'],'diagnosis':diagnosis,'idempotent':True,'diagnose_config_unchanged':True,'negatives':negatives})
    result={'result':'passed','os':'Windows','profiles':records,'model':'not_requested','ui':'separate_observation_required'}
    Path(args.result).write_text(json.dumps(result,indent=2),encoding='utf-8');print(json.dumps(result))
if __name__=='__main__':main()
