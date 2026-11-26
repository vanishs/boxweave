#!/usr/bin/env python3
"""One-use Scoped nested CI: short-lived dedicated auth boundary witnesses.

Only these newly issued fixture credentials traverse the private operator
pipes; no normal Agent credential, token bytes, or DB is logged or archived.
"""
import json
import os
import re
import subprocess
import sys

BIN = "/usr/local/bin/boxweave"
PROBE = r'''
import http.client,json,sys
own,foreign=json.load(sys.stdin)
for credential,expected in ((own,200),(foreign,401)):
    conn=http.client.HTTPConnection("127.0.0.1",18650,timeout=8)
    try:
        body=json.dumps({"jsonrpc":"2.0","id":1,"method":"initialize",
            "params":{"protocolVersion":"2025-11-25","capabilities":{},
                      "clientInfo":{"name":"s25-isolation-fixture","version":"1"}}})
        conn.request("POST","/mcp",body,{"Authorization":"Bearer "+credential,
            "Content-Type":"application/json","Accept":"application/json, text/event-stream"})
        reply=conn.getresponse()
        assert reply.status==expected,"unexpected fixture HTTP auth status"
        data=reply.read(65537)
        assert len(data)<=65536
    finally:
        conn.close()
'''

def run(prefix,*args,payload=None):
    p=subprocess.run([*prefix,*args],input=payload,text=True,capture_output=True,timeout=30)
    assert p.returncode==0,"dedicated dual-auth witness failed"
    return p.stdout

def issue(prefix,project):
    receipt=run(prefix,"runuser","-u","boxweave","--",BIN,"token-create","-project",project,"-ttl","3m")
    ident=re.search(r"(?m)^id=([0-9a-f]{32})(?: |$)",receipt)
    credential=re.search(r"(?m)^token=(bw1_[^\s]+)$",receipt)
    assert ident and credential,"invalid disposable auth receipt"
    return ident[1],credential[1]

def revoke(prefix,project,ident):
    run(prefix,"runuser","-u","boxweave","--",BIN,"token-revoke","-project",project,"-id",ident)

def verify(project):
    assert os.environ.get("GITHUB_ACTIONS")=="true"
    assert os.environ.get("RUNNER_ENVIRONMENT")=="github-hosted"
    assert os.environ.get("P1B_S25_SCOPED")=="1"
    outer=["sudo"]
    assert project=="user-"+run(outer,"id","-u","boxweave").strip()
    inner=["sudo","incus","exec","bw-s25-mcp-l1","--project",project,"--"]
    inner_project="user-"+run(inner,"id","-u","boxweave").strip()
    for prefix,p in ((outer,project),(inner,inner_project)):
        status=json.loads(run(prefix,BIN,"install-status","-project",p))
        assert status["scoped_peer_mode"]=="SCOPED_PEER_FRESH_V2" and status["service_active"]
        assert status["native_project"]==p
    receipts=[]
    try:
        for prefix,p in ((outer,project),(inner,inner_project)):
            ident,credential=issue(prefix,p)
            receipts.append((prefix,p,ident,credential))
        a,b=receipts
        assert a[2]!=b[2] and a[3]!=b[3]
        for local,other in ((a,b),(b,a)):
            run(local[0],"python3","-c",PROBE,payload=json.dumps([local[3],other[3]]))
    finally:
        failures=[]
        for prefix,p,ident,_ in receipts:
            try:
                revoke(prefix,p,ident)
            except Exception:
                failures.append(ident)
        assert not failures,"dedicated auth fixture revocation failed"
    print("P1B_S25_SCOPED_CROSS_FIXTURE_TOKEN_AUTH_DENIED=PASS")

if __name__=="__main__":
    verify(sys.argv[1])
