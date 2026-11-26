#!/usr/bin/env python3
"""Installed D4 acceptance, only in dedicated disposable test installation."""
import errno
from concurrent.futures import ThreadPoolExecutor
import hashlib
import pty
import select
import tempfile
import json
import os
import re
import socket
import sqlite3
import subprocess
import sys
import threading
import time
from pathlib import Path

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import op, token, wait_status
from pr188_h4a_installed import busy, unknown
from pr178_n4_native import guest, SERVER, UDP
from pr178_n2_scoped_lifecycle import binding

DB = "file:/var/lib/boxweave/ledger.sqlite?mode=ro"
BIN = "/usr/local/bin/boxweave"
CTL = "boxweave.service"
HELPER = "boxweave-scoped-lifecycle.service"

def run(args, **kw):
    result=subprocess.run(args, timeout=100, **kw)
    if result.returncode:
        if result.stdout:print(result.stdout[-10000:],flush=True)
        if result.stderr:print(result.stderr[-4000:],file=sys.stderr,flush=True)
        result.check_returncode()
    return result

def pid():
    return subprocess.check_output(["systemctl","show",CTL,"-p","MainPID","--value"],text=True).strip()

def wait_mcp_listener(timeout=15):
    """Wait for the restarted fixture listener; never retry a tool mutation."""
    deadline = time.monotonic() + timeout
    while True:
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            raise TimeoutError("restarted fixture MCP listener unavailable")
        try:
            with socket.create_connection(("127.0.0.1", 18650), timeout=min(1, remaining)):
                return
        except ConnectionRefusedError:
            time.sleep(min(.25, max(0, deadline - time.monotonic())))


def db_state(original):
    with sqlite3.connect(DB,uri=True) as db:
        row=db.execute("SELECT state,native_operation_id,box_uuid FROM operations WHERE id=?",(original,)).fetchone()
        claims=db.execute("SELECT layer,resource_key FROM operation_resource_claims WHERE operation_id=? ORDER BY layer,resource_key",(original,)).fetchall()
        receipts=db.execute("SELECT count(*) FROM operations WHERE action='operation_recover'").fetchone()[0]
        audits=db.execute("SELECT count(*) FROM recovery_audit WHERE operation_id=?",(original,)).fetchone()[0]
    return row,claims,receipts,audits

def journal(original, mode):
    if mode=="LEGACY":
        return []
    root=Path("/var/lib/boxweave-scoped-peer-v2")/("peer-"+original)
    return [(x.name,hashlib.sha256(x.read_bytes()).hexdigest()) for x in sorted(root.iterdir())]

def recheck_original(client, original, state, native_id):
    # A fresh authenticated client must agree with the local CLI and
    # original receipt. Activity is durable evidence, not native proof.
    op_row=client.operation_get(original)
    evidence=client.activity("evidence",operation_id=original)["evidence"]
    for row in (op_row,evidence["original"]):
        assert row["operation_id"]==original and row["state"]==state,row
        assert row.get("native_operation_id")==native_id,row
    return evidence

def cleanup_legacy(client, ids, fwd):
    op(client,"port_forward_delete",{"forward_id":fwd},"d4-sibling-forward-delete")
    # d4-a was independently verified Stopped; sibling stayed Running
    # for the continuity witness. Never replay the original stop.
    op(client,"box_stop",{"box_id":ids["d4-sibling"]},"d4-sibling-stop")
    for name in ("d4-a","d4-sibling"):
        op(client,"box_delete",{"box_id":ids[name]},"d4-delete-"+name)
    wait_status(client,lambda x:x["address_retirement"]["pending"]==0,120)
    print("PR177_D5_LEGACY_CLEANUP=PASS",flush=True)

def guided(project, issue, approve):
    """Exact local terminal approval, keep the plan and durable key."""
    with tempfile.TemporaryDirectory(prefix="h3b-guided-") as d:
        out, err = Path(d)/"out", Path(d)/"err"
        child, fd = pty.fork()
        if child == 0:
            try:
                for path, dest in ((out, 1), (err, 2)):
                    with path.open("wb") as f:
                        os.dup2(f.fileno(), dest)
                os.execv(BIN, [BIN,"recover","--guided","--issue",issue,"-project",project,
                               "--reviewer","d4-installed-test","--recovery-key","pr177-d4-reviewed-exact-key"])
            finally:
                os._exit(127)
        history = b""
        sent = None
        deadline = time.monotonic()+150
        try:
            while time.monotonic() < deadline:
                ready, _, _ = select.select([fd],[],[],1)
                if not ready:
                    continue
                try:
                    data = os.read(fd,4096)
                except OSError as exc:
                    if exc.errno == errno.EIO:
                        break
                    raise
                if not data:
                    break
                history = (history + data)[-8192:]
                match = re.search(rb"Type RECOVER ([0-9a-f]{64}) to approve: ",history)
                if match and sent is None:
                    sent = match.group(1).decode()
                    os.write(fd, ("RECOVER "+sent+"\n" if approve else "NO\n").encode())
            else:
                os.kill(child,15)
                raise RuntimeError("guided recovery terminal timeout")
        finally:
            os.close(fd)
            _, status = os.waitpid(child,0)
        assert sent is not None, err.read_text()[-3000:]
        plan, _ = json.JSONDecoder().raw_decode(err.read_text().lstrip())
        assert plan["review_sha256"] == sent and plan["status"] == "REVIEW_READY",plan
        assert (os.waitstatus_to_exitcode(status)==0)==approve,err.read_text()[-3000:]
        if not approve:
            assert not out.read_text()
            return plan,None
        return plan,json.loads(out.read_text())


def main(project, receipt, testbin, mode):
    assert os.geteuid()==0 and mode in ("LEGACY","SCOPED_V2")
    names=["d4-a","d4-sibling"]+([] if mode=="LEGACY" else ["d4-b"])
    with LoopbackMCP("http://127.0.0.1:18650/mcp",token=token(receipt),allow_mutations=True,timeout=60) as c:
        ids={}
        for name in names:
            ids[name]=op(c,"box_create",{"name":name,"cpu":1,"memory_mib":256,"disk_gib":2,"image":"local:boxweave-ubuntu-24.04","start":True,"internet":True},name+"-create")["box_uuid"]
        for name in [x for x in names if x!="d4-a"]:
            guest(project,name,"python3","-c","import subprocess,sys;subprocess.Popen(['python3','-u','-c',sys.argv[1]],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)",SERVER)
        time.sleep(1)
        if mode=="LEGACY":
            busy(c,ids["d4-sibling"],mode)
        fwd=op(c,"port_forward_create",{"box_id":ids["d4-sibling"],"protocol":"tcp","host_bind":"127.0.0.1","host_port":28284,"guest_port":18182},"d4-sibling-forward")["operation_id"]
    sock=socket.create_connection(("127.0.0.1",28284),timeout=5)
    sock.settimeout(2)
    stop=threading.Event();errors=[];rounds=[0]
    def monitor():
        while not stop.is_set():
            try:
                data=("d4-"+str(rounds[0])).encode()
                sock.sendall(data)
                out=b""
                while len(out)<len(data):
                    part=sock.recv(4096)
                    if not part:raise RuntimeError("forward disconnected")
                    out+=part
                assert out==data
                rounds[0]+=1
            except Exception as e:
                errors.append(repr(e));return
            stop.wait(.05)
    t=threading.Thread(target=monitor,daemon=True);t.start()
    try:
        # Intentional test fixture interruption BEFORE the recovery plan.
        # No service restart is permitted during actual plan/apply.
        run(["systemctl","stop",CTL])
        if mode=="SCOPED_V2":
            run(["systemctl","stop",HELPER])
        env={"BOXWEAVE_PR177_D4_LAB":"disposable-installed-v1","D4_TEST_PROJECT":project,"D4_TEST_MODE":mode}
        base=[testbin,"-test.run","^TestPR177D4InstalledCompletedOriginal$","-test.v","-test.timeout","100s"]
        staged=subprocess.check_output(["runuser","-u","boxweave","--","env",*[k+"="+v for k,v in env.items()],"D4_TEST_PHASE=stage",*base],text=True,timeout=100)
        print(staged,flush=True)
        original=re.search(r"PR177_D4_COMPLETED_ORIGINAL=([0-9a-f]{32})",staged).group(1)
        if mode=="SCOPED_V2":
            run(base,env=dict(os.environ,**env,D4_TEST_PHASE="apply-native",D4_TEST_ORIGINAL=original))
        run(["systemctl","start",CTL])
        for _ in range(30):
            if db_state(original)[0][0]=="UNKNOWN" and int(pid())>1:
                break
            time.sleep(.5)
        before=db_state(original);oldjournal=journal(original,mode);controller=pid()
        assert before[0][0]=="UNKNOWN" and int(controller)>1,before
        if mode=="SCOPED_V2":
            assert len(before[1])==5 and len(oldjournal)>1
        wait_mcp_listener()
        assert pid()==controller and db_state(original)==before and journal(original,mode)==oldjournal
        def navigation():
            with LoopbackMCP(token=token(receipt),allow_mutations=True,timeout=60) as reader:
                unknown(reader,ids["d4-a"],original,mode)

        def review():
            report=json.loads(run([BIN,"doctor","-project",project],capture_output=True,text=True).stdout)["diagnostic"]
            issue=next(i["id"] for i in report["issues"] if i["code"] in ("PROJECT_PEER_BARRIER","UNRESOLVED_OPERATION") and i.get("original_id")==original)
            return issue,run([BIN,"recover","--plan","--issue",issue,"-project",project],capture_output=True,text=True)

        # Incus 7.6 retains completed native operations for only five seconds.
        # Run independent read/refusal witnesses concurrently, before approval.
        with ThreadPoolExecutor(max_workers=2) as pool:
            nav=pool.submit(navigation)
            issue,planned=review()
            nav.result()
        assert db_state(original)==before and journal(original,mode)==oldjournal and pid()==controller
        p=json.loads(planned.stdout)
        assert p["status"]=="REVIEW_READY" and p["apply_available"] and p["mode"]==mode and p["original_id"]==original,p
        assert p["leaf"]=="MCP_SETTLE_ORIGINAL_NO_REPLAY" and p["original_state"]=="UNKNOWN" and not p["native_retried"]
        assert db_state(original)==before and journal(original,mode)==oldjournal
        print("PR177_D3_INSTALLED_POSITIVE_PLAN=PASS",mode,flush=True)
        base=[BIN,"recover","--apply","--issue",issue,"-project",project,"--approve-local","--reviewer","d4-installed-test","--recovery-key","pr177-d4-reviewed-exact-key","--confirm-review-sha256",p["review_sha256"]]
        def refused(args, payload):
            out=subprocess.run(args,input=payload,text=True,capture_output=True,timeout=100)
            assert out.returncode!=0,(out.stdout,out.stderr)
        wrong=list(base);wrong[-1]="0"*64
        forged=dict(p);forged["original_revision"]+=1
        cases=[(wrong,planned.stdout),
               ([x for x in base if x!="--approve-local"],planned.stdout),
               (["runuser","-u","boxweave","--",*base],planned.stdout),
               (base,json.dumps(forged))]
        with ThreadPoolExecutor(max_workers=5) as pool:
            negatives=[pool.submit(refused,*case) for case in cases]
            declined,_=guided(project,issue,False)
            for check in negatives:
                check.result()
        assert declined["original_id"]==original and db_state(original)==before and journal(original,mode)==oldjournal and pid()==controller
        print("PR188_H3B_GUIDED_DECLINE_NO_EFFECT=PASS",mode,flush=True)
        if mode=="SCOPED_V2":
            # Guided applies the same reviewed transaction. The saved exact
            # envelope and key must replay via the old machine API.
            p,result=guided(project,issue,True)
            base[-1]=p["review_sha256"]
            planned.stdout=json.dumps(p)
            assert p["original_id"]==original
            print("PR188_H3B_GUIDED_APPLY=PASS",mode,flush=True)
        else:
            # The product re-observes all evidence before admission. A
            # redundant fresh plan consumes the native receipt window.
            print("P1B_LEGACY_FIRST_EXACT_REVIEW",flush=True)
            out=subprocess.run(base,input=planned.stdout,text=True,capture_output=True,timeout=100)
            if out.returncode:
                # Diagnostic only: preserve and inspect, never re-apply.
                obs=subprocess.run([BIN,"recover","--plan","--issue",issue,"-project",project],capture_output=True,text=True,timeout=100)
                try:
                    current=json.loads(obs.stdout)
                    print("P1B_LEGACY_REFUSED_REOBSERVATION",current.get("status"),current.get("reason"),(current.get("witness") or {}).get("reason"),flush=True)
                except (ValueError,AttributeError):
                    print("P1B_LEGACY_REFUSED_REOBSERVATION_UNAVAILABLE",flush=True)
                assert db_state(original)==before and journal(original,mode)==oldjournal and pid()==controller
                print(out.stderr[-4000:],file=sys.stderr,flush=True)
                out.check_returncode()
            result=json.loads(out.stdout)
        rec=result["recovery"]
        assert rec["original_operation_id"]==original and rec["disposition"]=="SETTLED" and rec["operation"]["state"]=="SUCCEEDED",result
        assert result["postcheck"]["state"]=="DURABLE_ORIGINAL_POSTSTATE_VERIFIED" and result["postcheck"]["original_state"]=="SUCCEEDED" and not result["native_retried"],result
        after=db_state(original)
        assert after[0][0]=="SUCCEEDED" and after[0][1:]==before[0][1:] and after[1]==before[1] and after[2]==before[2]+1 and after[3]==before[3]+1,(before,after)
        dup=json.loads(run(base,input=planned.stdout,text=True,capture_output=True).stdout)
        assert dup["recovery"]["operation"]["operation_id"]==rec["operation"]["operation_id"] and dup["recovery"]["operation"]["duplicate"],dup
        changed=list(base);changed[changed.index("--recovery-key")+1]="pr177-d4-reviewed-other-key"
        refused(changed,planned.stdout)
        assert db_state(original)==after and journal(original,mode)==oldjournal and pid()==controller
        if mode=="SCOPED_V2":
            for family in ("ipv4","ipv6"):
                guest(project,"d4-a","python3","-c",UDP,binding(ids["d4-b"])[family],"yes")
        else:
            status=json.loads(subprocess.check_output(["incus","list","d4-a","--project",project,"-f","json"],text=True))
            assert len(status)==1 and status[0]["status"]=="Stopped",status
        doc=json.loads(run([BIN,"doctor","-project",project],capture_output=True,text=True).stdout)["diagnostic"]
        assert not any(i.get("original_id")==original and i["id"]==issue for i in doc["issues"])
        with LoopbackMCP("http://127.0.0.1:18650/mcp",token=token(receipt),timeout=60) as reader:
            recheck_original(reader,original,"SUCCEEDED",before[0][1] or None)
            recovery=reader.operation_get(rec["operation"]["operation_id"])
            assert recovery["state"]=="SUCCEEDED",recovery
        assert db_state(original)==after and journal(original,mode)==oldjournal and pid()==controller
        print("PR177_D5_AUTHENTICATED_POSTCHECK=PASS",mode,flush=True)
        assert not errors and rounds[0]>5,(errors,rounds)
        print("PR177_D4_INSTALLED_REVIEWED_RECOVERY_NO_REPLAY=PASS",mode,rounds[0],flush=True)
    finally:
        stop.set();t.join(timeout=5);sock.close()
    # Legacy subsequent upgrade/uninstall acceptance requires no Boxes.
    # Scoped dedicated fixtures may stay; do not alter unrelated peers.
    if mode=="LEGACY":
        with LoopbackMCP("http://127.0.0.1:18650/mcp",token=token(receipt),allow_mutations=True,timeout=60) as c:
            cleanup_legacy(c,ids,fwd)

if __name__=="__main__":
    main(*sys.argv[1:])
