#!/usr/bin/env python3
"""Live workload and second fresh install in the existing clean CI VM.

No fake network receipts, private credentials in logs, or inherited-policy
override. The caller owns the clean-runner guard and first retirement.
"""
from datetime import datetime, timezone
import hashlib
import http.client
import json
import os
from pathlib import Path
import sqlite3
import subprocess

from mcp_testkit import LoopbackMCP, MCPToolError
from n2_installed_lifecycle import op, pid, token, tty_run
from pr178_n2_scoped_lifecycle import binding

BIN = "/usr/local/bin/boxweave"
DB = "file:/var/lib/boxweave/ledger.sqlite?mode=ro"
PORT = 18080
FORWARD = 18082
# A fixed response, not a directory listing or application data.
SERVER = """from http.server import BaseHTTPRequestHandler, HTTPServer
class Handler(BaseHTTPRequestHandler):
 def do_GET(self):
  self.send_response(200)
  self.end_headers()
  self.wfile.write(b"roundtrip")
HTTPServer(("0.0.0.0",18080),Handler).serve_forever()
"""
APP = "import subprocess,sys; subprocess.Popen([sys.executable,'-c',"+repr(SERVER)+"],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)"
HTTP = """import http.client,sys
try:
 c=http.client.HTTPConnection(sys.argv[1],18080,timeout=3)
 c.request("GET","/")
 r=c.getresponse()
 if r.status!=200: sys.exit(8)
except OSError:
 sys.exit(7)
"""

def guest(c, box, code, key, *args, expected=0):
    d=op(c,"box_exec",{"box_id":box,"argv":["python3","-c",code,*args]},key)
    out=c.operation_output(d["operation_id"])
    assert out["exit_code"]==expected, (key,out["exit_code"])
    return d["operation_id"]

def http_status(port, path="/", bearer=None):
    c=http.client.HTTPConnection("127.0.0.1",port,timeout=10)
    try:
        if bearer is None:
            c.request("GET",path)
        else:
            c.request("POST",path,body="{}",headers={"Authorization":"Bearer "+bearer,
                "Content-Type":"application/json"})
        r=c.getresponse()
        r.read(1024)
        return r.status
    finally:
        c.close()

def network(c, a, b, prefix):
    # Verify each real guest application before interpreting a peer denial.
    for suffix,box in (("a",a),("b",b)):
        guest(c,box,APP,prefix+"-http-"+suffix)
        guest(c,box,"import time;time.sleep(1)",prefix+"-wait-"+suffix)
        guest(c,box,HTTP,prefix+"-local-"+suffix,"127.0.0.1")
    ipv4=binding(b)["ipv4"]
    guest(c,a,HTTP,prefix+"-deny",ipv4,expected=7)
    op(c,"box_peer_set",{"source_box_id":a,"target_box_id":b,"allow":True},prefix+"-grant")
    guest(c,a,HTTP,prefix+"-allowed",ipv4)
    guest(c,b,HTTP,prefix+"-reverse-denied",binding(a)["ipv4"],expected=7)
    f=op(c,"port_forward_create",{"box_id":a,"protocol":"tcp","host_bind":"127.0.0.1",
        "host_port":FORWARD,"guest_port":PORT},prefix+"-forward")
    assert http_status(FORWARD)==200
    print("PR188_ROUNDTRIP_NETWORK=PASS",prefix,flush=True)
    return f

def second_install(project, first_receipt, candidate, digest, old_ids, incus_pid):
    assert os.geteuid()==0 and os.environ.get("GITHUB_ACTIONS")=="true"
    assert os.environ.get("RUNNER_ENVIRONMENT")=="github-hosted"
    assert hashlib.sha256(Path(candidate).read_bytes()).hexdigest()==digest
    # pr182 already proved its sentinel Project and ACL unchanged. The
    # caller removes only those exact CI-created fixtures, never foreign
    # resources from the production installer/uninstaller.
    assert not Path("/var/lib/boxweave").exists()
    assert not Path("/var/lib/boxweave-retired").exists()
    assert not Path("/usr/local/bin/boxweave").exists()
    assert pid("incus.service")==incus_pid
    tty_run(["sh","install.sh","--interactive","--scoped-peer-fresh","--project",project,
             "--local-binary",candidate,"--sha256",digest],
            [(b"to approve this exact plan:","INSTALL SCOPED "+project),
             (b"Configure a new OAuth approval password?","N")])
    x=json.loads(subprocess.check_output([BIN,"install-status","-project",project],text=True))
    assert x["binary_sha256"]==digest and x["service_active"] and not x["restart_required"]
    with sqlite3.connect(DB,uri=True) as db:
        for table in ("owned_boxes","operations","peer_grants","forward_claims",
                      "network_bindings","network_address_fences"):
            assert db.execute("SELECT count(*) FROM "+table).fetchone()[0]==0,table
    expiration=next(line.split("expires=",1)[1] for line in Path(first_receipt).read_text().splitlines() if line.startswith("id="))
    assert datetime.fromisoformat(expiration.replace("Z","+00:00"))>datetime.now(timezone.utc), "old token expired before purge isolation proof"
    assert http_status(18650,"/mcp",token(first_receipt))==401
    assert pid("incus.service")==incus_pid
    print("PR188_ROUNDTRIP_SCOPED_FRESH_LEDGER_OLD_TOKEN_DENIED=PASS",flush=True)
    receipt=Path(first_receipt).with_name("roundtrip-second.receipt")
    fd=os.open(receipt,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
    with os.fdopen(fd,"w") as out:
        subprocess.run(["runuser","-u","boxweave","--",BIN,"token-create",
                        "-project",project,"-ttl","30m"],stdout=out,check=True)
    # The image alias belongs to the retained Incus Project, not the
    # purged BoxWeave ledger. Reimport only if absent.
    if subprocess.run(["incus","image","info","boxweave-ubuntu-24.04","--project",project],
                      stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode:
        subprocess.run(["incus","image","copy","images:ubuntu/24.04","local:",
                        "--target-project",project,"--alias","boxweave-ubuntu-24.04"],check=True)
    with LoopbackMCP(token=token(receipt),allow_mutations=True,timeout=60) as c:
        assert c.host_status()["ready"]
        for old in old_ids:
            try:
                c.call("box_get",{"box_id":old})
            except MCPToolError:
                pass
            else:
                raise AssertionError("retired original visible in fresh ledger")
        args={"cpu":1,"memory_mib":256,"disk_gib":2,
              "image":"local:boxweave-ubuntu-24.04","internet":True,"start":True}
        # Same names and keys on a genuinely fresh installation, not old
        # receipts replayed into a new incarnation.
        a=op(c,"box_create",dict(args,name="scoped-retirement-live"),"pr182-live")["box_uuid"]
        b=op(c,"box_create",dict(args,name="scoped-retirement-peer"),"pr188-peer-fixture")["box_uuid"]
        assert a!=b and not ({a,b}&set(old_ids))
        network(c,a,b,"roundtrip-second")
        h=c.host_status()
        assert h["project_capacity"]["owned"]==2 and h["scoped_identity_capacity"]["retired"]==0
    assert pid("incus.service")==incus_pid
    # Receipt is private and this VM is ephemeral. Revoke this exact
    # temporary actor rather than altering any other auth configuration.
    token_id=next(line.split()[0][3:] for line in receipt.read_text().splitlines() if line.startswith("id="))
    subprocess.run(["runuser","-u","boxweave","--",BIN,"token-revoke","-project",project,
                    "-id",token_id],check=True,stdout=subprocess.DEVNULL)
    receipt.unlink()
    print("PR188_ROUNDTRIP_SCOPED_REINSTALL_WORKLOAD=PASS",flush=True)
