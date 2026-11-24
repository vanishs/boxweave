#!/usr/bin/env python3
"""An incomplete native original must not take the MCP diagnostic API down."""
import hashlib
import sqlite3
import subprocess
import sys
import time
from pathlib import Path
from mcp_testkit import LoopbackMCP, MCPToolError
from n2_installed_lifecycle import token

def main(project,receipt,original):
    root=Path("/var/lib/boxweave-scoped-peer-v2")/("peer-"+original)
    def snapshot():
        with sqlite3.connect("file:/var/lib/boxweave/ledger.sqlite?mode=ro",uri=True) as db:
            op=db.execute("SELECT state,error_class FROM operations WHERE id=?",(original,)).fetchone()
            claims=db.execute("SELECT resource_key FROM operation_resource_claims WHERE operation_id=? ORDER BY resource_key",(original,)).fetchall()
        return op,claims,[(p.name,hashlib.sha256(p.read_bytes()).hexdigest()) for p in sorted(root.iterdir())]
    before=snapshot()
    assert before[0][0]=="UNKNOWN" and len(before[1])==5 and len(before[2])==1,before
    def prop(unit,key):
        return subprocess.check_output(["systemctl","show","-p",key,"--value",unit],text=True).strip()
    unit="boxweave.service";helper="boxweave-scoped-lifecycle.service"
    assert helper in prop(unit,"Wants").split() and helper not in prop(unit,"Requires").split()
    pid=prop(unit,"MainPID")
    assert int(pid)>0
    with sqlite3.connect("file:/var/lib/boxweave/ledger.sqlite?mode=ro",uri=True) as db:
        ids=dict(db.execute("SELECT name,uuid FROM owned_boxes WHERE state='OWNED'"))
        admitted=db.execute("SELECT COUNT(*) FROM operations WHERE action='box_peer_set'").fetchone()[0]
    with LoopbackMCP("http://127.0.0.1:18650/mcp",token=token(receipt),allow_mutations=True,timeout=60) as c:
        for i in range(12):
            r=c.call("operation_recover",{"operation_id":original,"mode":"inspect"}).structured
            d=r["diagnosis"]
            assert d["original_state"]=="UNKNOWN" and d["disposition"]=="BLOCKED" and d["claim_protection"]=="RETAINED" and not r["native_retried"],r
            try:
                c.call("box_peer_set",{"source_box_id":ids["n4-a"],"target_box_id":ids["n4-b"],"allow":True,"idempotency_key":"n4-degraded-must-stay-locked-"+str(i)})
            except MCPToolError:pass
            else:raise AssertionError("new effect admitted through retained fence")
            assert prop(unit,"MainPID")==pid
            time.sleep(1)
    assert snapshot()==before
    with sqlite3.connect("file:/var/lib/boxweave/ledger.sqlite?mode=ro",uri=True) as db:
        assert db.execute("SELECT COUNT(*) FROM operations WHERE action='box_peer_set'").fetchone()[0]==admitted
    print("PR178_N4_PARTIAL_MCP_FENCED=PASS",flush=True)
    print("PR178_N4_DEGRADED_MCP_STABLE=PASS",flush=True)
if __name__=="__main__":
    main(*sys.argv[1:])
