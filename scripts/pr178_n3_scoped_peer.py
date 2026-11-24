#!/usr/bin/env python3
"""Real installed MCP peer acceptance in a disposable dedicated lab."""
import select
import subprocess
import sys
import time

from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import op, token
from pr178_n2_scoped_lifecycle import binding, verify, instance

def native(project, name, *args):
    return subprocess.check_output(["incus", "exec", name, "--project", project, "--", *args], text=True, timeout=15)

ECHO = """import socketserver,socket
class H(socketserver.BaseRequestHandler):
 def handle(self):
  while True:
   b=self.request.recv(4096)
   if not b:return
   self.request.sendall(b)
class S(socketserver.ThreadingTCPServer):
 address_family=socket.AF_INET6
 allow_reuse_address=True
 daemon_threads=True
S(("::",18081),H).serve_forever()
"""
CLIENT = """import socket,sys
s=socket.create_connection((sys.argv[1],18081),2)
s.settimeout(1)
print("READY",flush=True)
for line in sys.stdin:
 try:
  data=line.encode()
  s.sendall(data)
  out=b""
  while len(out)<len(data):
   chunk=s.recv(4096)
   if not chunk:raise OSError("closed")
   out+=chunk
  print("OK" if out==data else "BAD",flush=True)
 except OSError:
  print("BLOCKED",flush=True)
"""
def start_flow(project, name, ip):
    proc=subprocess.Popen(["incus","exec",name,"--project",project,"--mode=non-interactive","--","python3","-u","-c",CLIENT,ip],
        stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True,bufsize=1)
    assert select.select([proc.stdout],[],[],10)[0] and proc.stdout.readline().strip()=="READY", (name,ip)
    return proc

def probe_flow(proc, message, expected):
    proc.stdin.write(message+"\n")
    proc.stdin.flush()
    assert select.select([proc.stdout],[],[],10)[0],"flow output timeout"
    assert proc.stdout.readline().strip()==expected,(message,expected)

def main(project, receipt):
    with LoopbackMCP("http://127.0.0.1:18650/mcp", token=token(receipt), allow_mutations=True, timeout=60) as c:
        args = {"cpu": 1, "memory_mib": 256, "disk_gib": 2, "image": "local:boxweave-ubuntu-24.04", "internet": True, "start": True}
        ids = {}
        for n in "abcd":
            ids[n] = op(c, "box_create", dict(args, name="n3-"+n, **{"boot.autostart": n != "b"}), "n3-create-"+n)["box_uuid"]
        before = {n:verify(project, "n3-"+n, ids[n], "false" if n == "b" else "true") for n in ids}
        ips = {n:binding(u)["ipv4"] for n,u in ids.items()}
        ips6 = {n:binding(u)["ipv6"] for n,u in ids.items()}
        def peer(a,b,allow,key):
            result=op(c,"box_peer_set",{"source_box_id":ids[a],"target_box_id":ids[b],"allow":allow},key)
            assert result["state"]=="SUCCEEDED",result
            print("N3_PEER",a,b,allow,result["operation_id"],flush=True)
            return result
        def reachable(a,b,want):
            q=subprocess.run(["incus","exec","n3-"+a,"--project",project,"--","python3","-c",
                "import socket,sys;s=socket.create_connection((sys.argv[1],18080),1);s.sendall(b'GET / HTTP/1.0\\r\\n\\r\\n');assert s.recv(32).startswith(b'HTTP/')",ips[b]],
                stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=10)
            assert (q.returncode==0)==want,(a,b,want,q.returncode)
        for n in ids:
            native(project,"n3-"+n,"sh","-c","nohup python3 -m http.server 18080 --bind 0.0.0.0 >/tmp/n3-http.log 2>&1 </dev/null &")
        for n in ids:
            # Fixture services are confined to these dedicated test guests.
            native(project,"n3-"+n,"python3","-c",
                   "import subprocess,sys;subprocess.Popen(['python3','-u','-c',sys.argv[1]],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)",ECHO)
        time.sleep(2)
        reachable("a","b",False);reachable("b","a",False);reachable("c","d",False)
        peer("c","d",True,"n3-grant-c-to-d-01")
        peer("a","b",True,"n3-grant-a-to-b-01")
        reachable("a","b",True);reachable("b","a",False);reachable("c","d",True)
        peer("b","a",True,"n3-grant-b-to-a-01")
        reachable("b","a",True)
        flows={}
        try:
            for family,addr in ((4,ips),(6,ips6)):
                for a,b in (("a","b"),("b","a"),("c","d")):
                    flows[a+b,family]=start_flow(project,"n3-"+a,addr[b])
                    probe_flow(flows[a+b,family],"before","OK")
            peer("a","b",False,"n3-revoke-a-to-b-01")
            for family in (4,6):
                probe_flow(flows["ab",family],"revoked","BLOCKED")
                probe_flow(flows["ba",family],"retained-reverse","OK")
                probe_flow(flows["cd",family],"retained-unrelated","OK")
            print("PR178_N3_INSTALLED_LIVE_FLOWS=PASS",flush=True)
        finally:
            for proc in flows.values():
                proc.stdin.close()
                proc.terminate()
                proc.wait(timeout=10)
        reachable("a","b",False);reachable("b","a",True);reachable("c","d",True)
        original=peer("a","b",True,"n3-regrant-a-to-b-01")
        assert peer("a","b",True,"n3-regrant-a-to-b-01")["operation_id"]==original["operation_id"]
        reachable("a","b",True)
        extra=op(c,"box_create",dict(args,name="n3-extra",start=False,**{"boot.autostart":False}),"n3-active-extra-create")["box_uuid"]
        op(c,"box_delete",{"box_id":extra},"n3-active-extra-delete")
        reachable("c","d",True)
        print("PR178_N3_ACTIVE_POLICY_LIFECYCLE=PASS",flush=True)
        subprocess.run(["/usr/local/bin/boxweave","native-scoped-guard-boot"],check=True,timeout=30)
        subprocess.run(["systemctl","restart","boxweave-scoped-lifecycle.service"],check=True,timeout=30)
        subprocess.run(["systemctl","is-active","--quiet","boxweave-scoped-lifecycle.service"],check=True,timeout=15)
        subprocess.run(["/usr/local/bin/boxweave","native-scoped-guard-boot"],check=True,timeout=30)
        reachable("a","b",True);reachable("b","a",True);reachable("c","d",True)
        print("PR178_N3_BOOT_POLICY=PASS",flush=True)
        for n in ids:
            assert verify(project,"n3-"+n,ids[n],"false" if n=="b" else "true")==before[n]
            assert instance(project,"n3-"+n)["status"]=="Running"
        print("PR178_N3_INSTALLED_PEER=PASS",flush=True)

if __name__ == "__main__":
    main(*sys.argv[1:])
