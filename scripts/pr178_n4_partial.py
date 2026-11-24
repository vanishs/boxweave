#!/usr/bin/env python3
"""Last test in a disposable lab: leave the partial journal and claims intact."""
import os
import re
import socket
import sqlite3
import subprocess
import sys
import threading
import time
from pathlib import Path

from pr178_n2_scoped_lifecycle import binding
from pr178_n4_native import guest,UDP

def main(project,testbin):
    assert os.geteuid()==0
    with sqlite3.connect("file:/var/lib/boxweave/ledger.sqlite?mode=ro",uri=True) as db:
        ids=dict(db.execute("SELECT name,uuid FROM owned_boxes WHERE state='OWNED' AND name LIKE 'n4-%'"))
    assert set(ids)=={"n4-a","n4-b","n4-c","n4-d"}
    env={"BOXWEAVE_PR178_N4_PARTIAL_LAB":"disposable-scoped-v2","N4_TEST_PROJECT":project}
    base=[testbin,"-test.run","^TestPR178N4InstalledPartialACL$","-test.v","-test.timeout","100s"]
    sockets=[]
    for family,addr in ((socket.AF_INET,"127.0.0.1"),(socket.AF_INET6,"::1")):
        for typ,port in ((socket.SOCK_STREAM,28182),(socket.SOCK_DGRAM,28181)):
            s=socket.socket(family,typ);s.settimeout(2);s.connect((addr,port));sockets.append(s)
    stop=threading.Event();failures=[];count=[0]
    def monitor():
        while not stop.is_set():
            try:
                for s in sockets:
                    data=("partial-"+str(count[0])).encode()
                    s.sendall(data);assert s.recv(4096)==data
                count[0]+=1
            except Exception as e:
                failures.append(repr(e));return
            stop.wait(.05)
    t=threading.Thread(target=monitor,daemon=True);t.start()
    try:
        subprocess.run(["systemctl","stop","boxweave-scoped-lifecycle.service"],check=True,timeout=30)
        stage=subprocess.check_output(["runuser","-u","boxweave","--","env",*[k+"="+v for k,v in env.items()],"N4_TEST_PHASE=stage",*base],text=True,timeout=100)
        print(stage,flush=True)
        original=re.search(r"PR178_N4_PARTIAL_ORIGINAL=([0-9a-f]{32})",stage).group(1)
        subprocess.run(base,env=dict(os.environ,**env,N4_TEST_PHASE="apply",N4_TEST_ORIGINAL=original),check=True,timeout=100)
        for af in ("ipv4","ipv6"):
            guest(project,"n4-a","python3","-c",UDP,binding(ids["n4-b"])[af],"no")
            guest(project,"n4-b","python3","-c",UDP,binding(ids["n4-a"])[af],"no")
            guest(project,"n4-c","python3","-c",UDP,binding(ids["n4-d"])[af],"yes")
        print("PR178_N4_PARTIAL_PACKET_SAFETY=PASS",flush=True)
        # Start the real services again. Incomplete published-policy
        # history must not be treated as a recoverable complete original.
        subprocess.run(["systemctl","start","boxweave.service"],timeout=30,check=False)
        time.sleep(5)
        with sqlite3.connect("file:/var/lib/boxweave/ledger.sqlite?mode=ro",uri=True) as db:
            phase=db.execute("SELECT state FROM operations WHERE id=?",(original,)).fetchone()[0]
        print("N4_PARTIAL_RESTART_PHASE",phase,flush=True)
        # If Root fails startup before controller recovery, the original
        # remains DISPATCHING (fenced). The normal Store.Recover startup
        # routine is tested separately; never manually rewrite its state.
        assert phase in ("DISPATCHING","UNKNOWN"),phase
        if phase=="UNKNOWN":
            subprocess.run(base,env=dict(os.environ,**env,N4_TEST_PHASE="recover",N4_TEST_ORIGINAL=original),check=True,timeout=100)
        journal=Path("/var/lib/boxweave-scoped-peer-v2")/("peer-"+original)
        assert sorted(x.name for x in journal.iterdir())==["step-00-intent.json"]
        check=subprocess.run(["/usr/local/bin/boxweave","native-scoped-guard-boot"],capture_output=True,text=True,timeout=30)
        assert check.returncode!=0
        print("PR178_N4_PARTIAL_BOOT_REFUSAL=PASS",flush=True)
        assert not failures and count[0]>5,(failures,count)
        print("PR178_N4_PARTIAL_FORWARD_CONTINUITY=PASS",count[0],flush=True)
    finally:
        stop.set();t.join(timeout=5)
        for s in sockets:s.close()

if __name__=="__main__":
    main(*sys.argv[1:])
