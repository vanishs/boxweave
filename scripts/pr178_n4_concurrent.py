#!/usr/bin/env python3
"""Real concurrent admission and retained original UDP sockets, dedicated lab."""
import concurrent.futures
import select
import sqlite3
import subprocess
import sys
import threading
import time
from mcp_testkit import LoopbackMCP, MCPToolError
from n2_installed_lifecycle import op,token
from pr178_n2_scoped_lifecycle import binding,instance,verify
from pr178_n4_native import guest

UDP_CLIENT = """import socket,sys
ip=sys.argv[1]
s=socket.socket(socket.AF_INET6 if ":" in ip else socket.AF_INET,socket.SOCK_DGRAM)
s.connect((ip,18181));s.settimeout(.8)
print("READY",flush=True)
for line in sys.stdin:
 data=line.encode()
 try:
  s.send(data);out=s.recv(4096)
  print("OK" if out==data else "BAD",flush=True)
 except (TimeoutError,ConnectionRefusedError):
  print("BLOCKED",flush=True)
"""
def start(project,name,ip):
    p=subprocess.Popen(["incus","exec",name,"--project",project,"--mode=non-interactive","--","python3","-u","-c",UDP_CLIENT,ip],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True,bufsize=1)
    assert select.select([p.stdout],[],[],10)[0] and p.stdout.readline().strip()=="READY"
    return p

def probe(p,tag,want):
    p.stdin.write(tag+"\n");p.stdin.flush()
    assert select.select([p.stdout],[],[],10)[0]
    assert p.stdout.readline().strip()==want,(tag,want)

def main(project,receipt):
    with sqlite3.connect("file:/var/lib/boxweave/ledger.sqlite?mode=ro",uri=True) as db:
        ids=dict(db.execute("SELECT name,uuid FROM owned_boxes WHERE state='OWNED' AND name IN ('n4-a','n4-b','n4-c','n4-d')"))
    assert len(ids)==4
    ids={n[-1]:u for n,u in ids.items()}
    def client():
        return LoopbackMCP("http://127.0.0.1:18650/mcp",token=token(receipt),allow_mutations=True,timeout=60)
    def uptime(n):
        return guest(project,"n4-"+n,"cat","/proc/sys/kernel/random/boot_id").strip(),guest(project,"n4-"+n,"stat","-c","%i","/proc/1").strip(),instance(project,"n4-"+n)["config"]["volatile.last_state.power"]
    before={n:uptime(n) for n in ids}
    flows={}
    barrier=threading.Barrier(3)
    def submit(a,b,allow,key):
        with client() as c:
            barrier.wait(timeout=20)
            try:
                r=c.call("box_peer_set",{"source_box_id":ids[a],"target_box_id":ids[b],"allow":allow,"idempotency_key":key,"work_ref":"pr178-n4-concurrency"}).structured
            except MCPToolError as e:
                return {"rejected":str(e)}
            return c.poll_operation(r["operation_id"],seconds=150)
    try:
        for af in ("ipv4","ipv6"):
            for a,b in (("a","b"),("b","a"),("c","d")):
                p=start(project,"n4-"+a,binding(ids[b])[af])
                flows[a+b,af]=p;probe(p,"before","OK")
        with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:
            fs=[pool.submit(submit,"a","b",False,"n4-concurrent-ab-revoke"),pool.submit(submit,"a","b",False,"n4-concurrent-ab-revoke"),pool.submit(submit,"d","c",True,"n4-concurrent-dc-grant")]
            results=[f.result() for f in fs]
        print("N4_CONCURRENT_RESULTS",results,flush=True)
        assert all(r.get("state")=="SUCCEEDED" or "rejected" in r for r in results),results
        assert len({r["operation_id"] for r in results[:2] if "operation_id" in r})<=1
        # A rejected admission has no original; same-key resubmission is
        # permitted only after all competing originals are terminal.
        with client() as c:
            ab=op(c,"box_peer_set",{"source_box_id":ids["a"],"target_box_id":ids["b"],"allow":False},"n4-concurrent-ab-settle") if all("rejected" in r for r in results[:2]) else results[0] if "operation_id" in results[0] else results[1]
            assert ab["state"]=="SUCCEEDED"
            if "rejected" in results[2]:
                op(c,"box_peer_set",{"source_box_id":ids["d"],"target_box_id":ids["c"],"allow":True},"n4-concurrent-dc-settle")
        for af in ("ipv4","ipv6"):
            probe(flows["ab",af],"revoked","BLOCKED")
            probe(flows["ba",af],"retained-reverse","OK")
            probe(flows["cd",af],"retained-unrelated","OK")
        assert before=={n:uptime(n) for n in ids}
        print("PR178_N4_CONCURRENT_ORIGINAL_UDP=PASS",flush=True)
    finally:
        for p in flows.values():
            p.stdin.close();p.terminate()
            try:p.wait(timeout=10)
            except subprocess.TimeoutExpired:p.kill();p.wait()
if __name__=="__main__":
    main(*sys.argv[1:])
