#!/usr/bin/env python3
"""Installed authenticated MCP navigation assertions for disposable fixtures."""
import json
import os
import sqlite3
import sys

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import token, pid

DB = "file:/var/lib/boxweave/ledger.sqlite?mode=ro"
CTL = "boxweave.service"

def rejected(c, name, args, code):
    assert c.allow_mutations
    r = c._request("tools/call", {"name": name, "arguments": args})
    assert r.get("isError") is True and not r.get("structuredContent"), r
    parts = r["content"]
    assert len(parts) == 1 and parts[0]["type"] == "text"
    f = json.loads(parts[0]["text"])
    assert f == r["_meta"]["io.boxweave.error"] and f["code"] == code, f
    return f

def count(work):
    with sqlite3.connect(DB, uri=True) as db:
        return db.execute("SELECT count(*) FROM operations WHERE work_ref=?", (work,)).fetchone()[0]

def preflight(project, receipt, scoped):
    assert os.geteuid() == 0
    before = pid(CTL)
    with LoopbackMCP(token=token(receipt), allow_mutations=True, timeout=60) as c:
        status = c.host_status()
        assert status["peer_mode"] == ("SCOPED_V2" if scoped else "LEGACY")
        assert status["readiness_class"] == "READY_OBSERVED" and status["native"]["project"] == project
        tools = c.all_tools()
        assert len(tools) == 23
        for t in tools.values():
            key = t["inputSchema"].get("properties", {}).get("idempotency_key")
            if key:
                assert key["maxLength"] == 128 and "16..128 UTF-8 bytes" in key["description"], key
                if t["name"] != "operation_recover":
                    assert key["minLength"] == 16
        for key in ("x"*15, "x"*129, chr(0x754c)*43):
            f = rejected(c,"box_create",{"name":"h4a-invalid-key","idempotency_key":key,"work_ref":"h4a-invalid"}, "INVALID_IDEMPOTENCY_KEY")
            assert f["admission"]=="NOT_ADMITTED" and "operation_id" not in f["next_action"]
        f = rejected(c,"box_get",{"box_id":"00000000-0000-4000-8000-000000000001"}, "BOX_NOT_FOUND_OR_NOT_OWNED")
        assert "operation_id" not in f["next_action"]
        f = rejected(c,"box_exec",{"box_id":"missing","idempotency_key":"h4a-input-invalid-0001","argv":{"secret":"H4A_DO_NOT_ECHO"}}, "INVALID_INPUT")
        assert "H4A_DO_NOT_ECHO" not in json.dumps(f)
        assert count("h4a-invalid")==0
    assert pid(CTL)==before
    print("PR188_H4A_INSTALLED_PREFLIGHT=PASS",flush=True)

def busy(c, box, mode):
    # Guest execution is deliberately long enough to keep the exact claim.
    # No fixture-specific daemon restart, no fault injection, no native retry.
    controller=pid(CTL)
    args={"box_id":box,"argv":["sh","-c","sleep 8; printf H4A"],"idempotency_key":"pr188-h4a-long-exec-0001","work_ref":"h4a-busy-original"}
    original=c.call("box_exec",args).structured
    oid=original["operation_id"]
    duplicate=c.call("box_exec",args).structured
    assert duplicate["operation_id"]==oid and duplicate["duplicate"]
    follow={"box_id":box,"argv":["true"],"idempotency_key":"pr188-h4a-waited-exec-0001","work_ref":"h4a-busy-refused"}
    f=rejected(c,"box_exec",follow,"RESOURCE_BUSY")
    assert f["admission"]=="NOT_ADMITTED" and f["category"]=="NORMAL_WAIT" and f["next_action"]["operation_id"]==oid and f["next_action"]["tool"]=="operation_get",f
    assert count("h4a-busy-refused")==0 and count("h4a-busy-original")==1
    result=c.poll_operation(oid,seconds=60)
    assert result["state"]=="SUCCEEDED"
    original_output=c.call("operation_output_get",{"operation_id":oid,"stream":"stdout"}).structured
    assert original_output["exit_code"]==0 and original_output["content_base64"]=="SDRB"
    new=c.call("box_exec",follow).structured
    assert new["operation_id"]!=oid and c.poll_operation(new["operation_id"],seconds=60)["state"]=="SUCCEEDED"
    assert count("h4a-busy-refused")==1 and pid(CTL)==controller
    print("PR188_H4A_INSTALLED_BUSY_AND_REPLAY=PASS",mode,flush=True)

def unknown(c, box, original, mode):
    before=pid(CTL)
    f=rejected(c,"box_exec",{"box_id":box,"argv":["true"],"idempotency_key":"pr188-h4a-unknown-refused-0001","work_ref":"h4a-unknown-refused"},"UNKNOWN_BLOCKER")
    assert f["admission"]=="NOT_ADMITTED" and f["next_action"]=={"kind":"INSPECT_RECOVERY","tool":"operation_recover","operation_id":original,"mode":"inspect"},f
    assert count("h4a-unknown-refused")==0
    d=c.call("operation_get",{"operation_id":original}).structured
    assert d["operation"]["state"]=="UNKNOWN" and d["diagnostic"]["status_class"]=="EFFECT_UNVERIFIED"
    assert d["recovery_navigation"]["next_action"]["operation_id"]==original
    r=c.call("operation_recover",{"operation_id":original,"mode":"inspect"}).structured
    assert r["diagnosis"]["disposition"]=="SAFE_RECOVER" and r["diagnosis"]["original_operation_id"]==original and not r["native_retried"],r
    assert r["next_action"]=={"kind":"RECONCILE_ORIGINAL","tool":"operation_recover","operation_id":original,"mode":"reconcile"},r
    assert pid(CTL)==before
    print("PR188_H4A_INSTALLED_UNKNOWN_NAVIGATION=PASS",mode,flush=True)

if __name__=="__main__":
    preflight(sys.argv[1],sys.argv[2],sys.argv[3]=="1")
