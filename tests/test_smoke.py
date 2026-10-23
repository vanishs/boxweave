import json
import os
from pathlib import Path
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'server'))
import smoke
from mcp_client import MCPError


class FakeClient:
    def __init__(self, state):
        self.state=state
        self.initial={'serverInfo':{'name':'fake','version':'test'}}
    def call(self, method, params, timeout=30):
        return {'tools':[{'name':n} for n in ('list_containers','get_container','create_container','connect')]}
    def tool(self,name,args,timeout=30):
        self.state['calls'].append((name,args))
        if name=='list_containers':
            text='Found 1 container(s):\n\n\U0001f4e6 smoke-box-container\n' if self.state['exists'] else 'No containers found.'
        elif name=='create_container':
            self.state.update(exists=True,created=str(time.time()))
            if self.state.get('ambiguous'):raise MCPError('Ambiguous creation')
            text='Created'
        elif name=='get_container':
            text=json.dumps({'container':{'name':'smoke-box-container','username':'smoke-box',
                  'createdAt':self.state['created'],'state':'CONTAINER_STATE_RUNNING'}})
        else:
            output='smoke-box\nsmoke-box-container\n' if args['exec']=='whoami && hostname' else 'hello world\n'
            text='exit_code: 0\n\n--- stdout ---\n'+output
        return {'content':[{'type':'text','text':text}]}
    def close(self):pass


@unittest.skipUnless(os.name=='posix','POSIX smoke tests')
class SmokeTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory();self.addCleanup(self.tmp.cleanup)
        self.root=Path(self.tmp.name)
        launcher=self.root/'launcher';launcher.write_text('#!/bin/sh\nexit 0\n');launcher.chmod(0o755)
        self.patch=patch.object(smoke,'public_key',return_value=('ssh-ed25519 TEST fixture','SHA256:TEST'))
        self.patch.start();self.addCleanup(self.patch.stop)
        self.plan=smoke.make_plan('smoke-box',self.root/'public',launcher,self.root/'run')
        self.state={'calls':[],'exists':False,'created':str(time.time())}
    def run_plan(self):
        return smoke.run(self.plan['path'],self.plan['approval_sha256'],lambda p:FakeClient(self.state))

    def test_plan_has_no_mcp_actions_and_confirmation_is_bound(self):
        self.assertEqual(self.state['calls'],[])
        with self.assertRaises(ValueError):smoke.run(self.plan['path'],'bad',lambda p:self.fail('Must not start MCP'))

    def test_success_uses_mcp_create_and_connect_and_is_not_replayed(self):
        result=self.run_plan()
        self.assertEqual((result['state'],result['exit_code'],result['stdout']),('succeeded',0,'hello world\n'))
        create=[args for name,args in self.state['calls'] if name=='create_container']
        self.assertEqual(len(create),1);self.assertEqual(create[0]['ssh_keys'],['ssh-ed25519 TEST fixture'])
        before=len(self.state['calls']);self.run_plan();self.assertEqual(len(self.state['calls']),before)

    def test_ambiguous_create_is_reconciled_without_duplicate(self):
        self.state['ambiguous']=True
        self.assertEqual(self.run_plan()['state'],'succeeded')
        self.assertEqual(sum(name=='create_container' for name,args in self.state['calls']),1)

    def test_existing_unapproved_workload_is_not_used(self):
        self.state['exists']=True
        self.assertEqual(self.run_plan()['state'],'incomplete')
        self.assertFalse(any(name in ('create_container','connect') for name,args in self.state['calls']))

    def test_resume_queries_previously_requested_creation(self):
        self.state['exists']=True
        smoke.atomic_json(self.root/'run/result.json',{'schema_version':1,'plan_sha256':self.plan['approval_sha256'],
             'state':'incomplete','phase':'provisioning','phase_deadline_epoch':time.time()+60,
             'create_requested':True,'create_requested_epoch':time.time()-1})
        self.assertEqual(self.run_plan()['state'],'succeeded')
        self.assertFalse(any(name=='create_container' for name,args in self.state['calls']))

    def test_resume_does_not_reset_deadline(self):
        smoke.atomic_json(self.root/'run/result.json',{'schema_version':1,'plan_sha256':self.plan['approval_sha256'],
             'state':'incomplete','phase':'provisioning','phase_deadline_epoch':time.time()-1,'create_requested':True})
        result=self.run_plan()
        self.assertEqual(result['state'],'timed_out');self.assertIsNone(result['exit_code'])
        self.assertFalse(any(name=='create_container' for name,args in self.state['calls']))

    def test_guest_wire_markers_cannot_fake_fixed_expected_output(self):
        with self.assertRaises(MCPError):smoke.execution({'content':[{'type':'text','text':'junk\nexit_code: 0\n\n--- stdout ---\nhello world\n'}]})
