"""Gateway tracked-exec integration: real metadata DB, fake Provider, no guest."""
import io
import json
import pathlib
import tempfile
import unittest
from contextlib import redirect_stdout
from unittest.mock import patch

from bwmini import mcp_gate
from bwmini.gateway_tasks import TaskStore
from bwmini.coordination_bridge import SharedClaims, app_box
from bwmini.coordination_runtime import private_connection
from bwmini.operation_coordination import OperationCoordinator
from bwmini.domain import Target

SCOPE = {"provider":"containarium","backend":"dev","project":"default"}
TARGET = Target("containarium","dev","default","box1","box1-container","123","box1","box1-container")

class FakeProvider:
    def __init__(self):
        self.responses=[]
        self.sent=[]
        self.stdin=self
        self.stdout=self
    def write(self,raw):
        outgoing=json.loads(raw)
        self.sent.append(outgoing)
        if outgoing["method"]!="tools/call":return
        tool=outgoing["params"]["name"]
        if tool=="get_container":
            data={"container":{"username":"box1","name":"box1-container",
                               "createdAt":123,"state":"CONTAINER_STATE_RUNNING",
                               "network":{"ipAddress":"10.100.0.2"}}}
            answer={"jsonrpc":"2.0","id":outgoing["id"],
                    "result":{"content":[{"type":"text","text":json.dumps(data)}]}}
        else:
            answer={"jsonrpc":"2.0","id":outgoing["id"],
                    "result":{"structuredContent":{"exit_code":7,"stdout":"sensitive-content"}}}
        self.responses.append(json.dumps(answer)+"\n")
    def flush(self):
        pass
    def readline(self,n):
        return self.responses.pop(0).encode() if self.responses else b""
    def poll(self):
        return 0
    def close(self):
        pass

class GatewayTaskIntegration(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        path=pathlib.Path(self.tmp.name)/"gateway.sqlite"
        path.touch(mode=0o600)
        self.store=TaskStore(path)
        self.store.initialize()
        claim_path=pathlib.Path(self.tmp.name)/"claims.sqlite"
        claim_path.touch(mode=0o600)
        OperationCoordinator(lambda:private_connection(claim_path)).initialize()
        self.claims=SharedClaims(claim_path)

    def run_gateway(self,requests,allowed=None):
        provider=FakeProvider()
        inbox=io.StringIO("".join(json.dumps(x)+"\n" for x in requests))
        out=io.StringIO()
        with patch.object(mcp_gate,"tool_policy",return_value=allowed), \
             patch.object(mcp_gate,"trusted_registry",return_value={"scope":SCOPE}), \
             patch.object(mcp_gate,"trusted",return_value=None), \
             patch.object(mcp_gate.subprocess,"Popen",return_value=provider), \
             patch.object(mcp_gate.gateway_tasks,"TaskStore",return_value=self.store), \
             patch.object(mcp_gate.coordination_bridge,"SharedClaims",return_value=self.claims), \
             patch.object(mcp_gate.coordination_bridge,"published_scope",return_value=SCOPE), \
             patch.object(mcp_gate.gd,"record"), \
             patch.object(mcp_gate.sys,"stdin",inbox), redirect_stdout(out):
            mcp_gate.main()
        responses=[json.loads(line) for line in out.getvalue().splitlines()]
        return provider,responses

    def test_same_key_across_requests_forwards_only_one_exec(self):
        tool={"jsonrpc":"2.0","id":4,"method":"tools/call",
              "params":{"name":"boxweave_task_submit",
                        "arguments":{"box":"box1","request_key":"a"*32,"exec":"pytest -q"}}}
        provider,answers=self.run_gateway([tool,tool|{"id":5}])
        self.assertEqual([x["params"]["name"] for x in provider.sent].count("connect"),1)
        data=[json.loads(x["result"]["content"][0]["text"]) for x in answers]
        self.assertEqual(data[0]["state"],"completed")
        self.assertEqual(data[0]["exit_code"],7)
        self.assertEqual(data[0]["task_id"],data[1]["task_id"])
        self.assertNotIn("sensitive-content",str(answers))

    def test_lookup_after_gateway_restart_never_redispatches(self):
        submit={"jsonrpc":"2.0","id":4,"method":"tools/call",
                "params":{"name":"boxweave_task_submit",
                          "arguments":{"box":"box1","request_key":"b"*32,"exec":"pytest -q"}}}
        _,first=self.run_gateway([submit])
        task=json.loads(first[0]["result"]["content"][0]["text"])
        query={"jsonrpc":"2.0","id":9,"method":"tools/call",
               "params":{"name":"boxweave_task_get",
                         "arguments":{"task_id":task["task_id"]}}}
        next_provider,other=self.run_gateway([query])
        self.assertEqual(next_provider.sent,[])
        result=json.loads(other[0]["result"]["content"][0]["text"])
        self.assertEqual(result["state"],"completed")


    def test_web_mutation_claim_blocks_tracked_gateway_exec(self):
        identity,scope,logical=app_box(TARGET,SCOPE)
        self.claims.acquire("1"*32,"web_1",identity,scope,logical)
        command={"jsonrpc":"2.0","id":12,"method":"tools/call",
                 "params":{"name":"boxweave_task_submit",
                           "arguments":{"box":"box1","request_key":"d"*32,"exec":"echo denied"}}}
        provider,replies=self.run_gateway([command])
        self.assertEqual([x["params"]["name"] for x in provider.sent].count("connect"),0)
        self.assertEqual(replies[0]["error"]["message"],"RESOURCE_BUSY")
        self.assertEqual(self.store.get(request_key="d"*32)["state"],"failed_before_dispatch")

    def test_unknown_web_claim_survives_recovered_gateway(self):
        identity,scope,logical=app_box(TARGET,SCOPE)
        self.claims.acquire("1"*32,"web_1",identity,scope,logical)
        self.claims.dispatch_intent("1"*32)
        self.claims.unknown("1"*32)
        command={"jsonrpc":"2.0","id":12,"method":"tools/call",
                 "params":{"name":"boxweave_task_submit",
                           "arguments":{"box":"box1","request_key":"e"*32,"exec":"echo denied"}}}
        provider,replies=self.run_gateway([command])
        self.assertEqual([x["params"]["name"] for x in provider.sent].count("connect"),0)
        self.assertEqual(replies[0]["error"]["message"],"RECOVERY_REQUIRED")

    def test_missing_connect_allowlist_denies_before_provider_read(self):
        submit={"jsonrpc":"2.0","id":4,"method":"tools/call",
                "params":{"name":"boxweave_task_submit",
                          "arguments":{"box":"box1","request_key":"c"*32,"exec":"echo unsafe"}}}
        provider,answers=self.run_gateway([submit],frozenset(["boxweave_task_submit"]))
        self.assertEqual(provider.sent,[])
        self.assertEqual(answers[0]["error"]["message"],"GATE_TOOL_NOT_ALLOWED")

if __name__=="__main__":
    unittest.main()
