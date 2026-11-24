#!/usr/bin/env python3
"""Disposable installed acceptance: dual-stack UDP and local Forward continuity."""
import socket
import subprocess
import sys
import threading
import time
from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import op, token
from pr178_n2_scoped_lifecycle import binding, verify, instance

SERVER = """import socketserver,socket,threading
class U(socketserver.BaseRequestHandler):
 def handle(self):
  data,s=self.request;s.sendto(data,self.client_address)
class T(socketserver.BaseRequestHandler):
 def handle(self):
  while True:
   data=self.request.recv(4096)
   if not data:return
   self.request.sendall(data)
class U6(socketserver.ThreadingUDPServer):
 address_family=socket.AF_INET6
 allow_reuse_address=True
class T6(socketserver.ThreadingTCPServer):
 address_family=socket.AF_INET6
 allow_reuse_address=True
for s in (U6(("::",18181),U),T6(("::",18182),T)):
 threading.Thread(target=s.serve_forever,daemon=True).start()
threading.Event().wait()
"""
UDP = """import socket,sys
addr=sys.argv[1];want=sys.argv[2]=="yes"
s=socket.socket(socket.AF_INET6 if ":" in addr else socket.AF_INET,socket.SOCK_DGRAM)
s.settimeout(.6);s.connect((addr,18181))
s.send(b"n4-udp-witness")
try:got=s.recv(1024)==b"n4-udp-witness"
except (TimeoutError,ConnectionRefusedError):got=False
assert got==want,(addr,want,got)
"""
def guest(project, name, *args):
    return subprocess.check_output(["incus","exec",name,"--project",project,"--",*args],text=True,timeout=15)

def main(project, receipt):
    with LoopbackMCP("http://127.0.0.1:18650/mcp",token=token(receipt),allow_mutations=True,timeout=60) as c:
        ids={}
        for n in "abcd":
            ids[n]=op(c,"box_create",{"name":"n4-"+n,"cpu":1,"memory_mib":256,"disk_gib":2,"image":"local:boxweave-ubuntu-24.04","start":True,"internet":True},"n4-create-"+n)["box_uuid"]
            guest(project,"n4-"+n,"python3","-c","import subprocess,sys;subprocess.Popen(['python3','-u','-c',sys.argv[1]],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)",SERVER)
        ips={n:binding(u) for n,u in ids.items()}
        before={n:verify(project,"n4-"+n,u,"true") for n,u in ids.items()}
        def peer(a,b,allow,key):
            return op(c,"box_peer_set",{"source_box_id":ids[a],"target_box_id":ids[b],"allow":allow},key)
        def udp(a,b,want):
            for af in ("ipv4","ipv6"):
                guest(project,"n4-"+a,"python3","-c",UDP,ips[b][af],"yes" if want else "no")
        time.sleep(2)
        udp("a","b",False);udp("b","a",False)
        peer("c","d",True,"n4-cd-grant-01")
        udp("c","d",True)
        forwards=[]
        for proto,bind,port,gport in (("tcp","127.0.0.1",28182,18182),("tcp","::1",28182,18182),("udp","127.0.0.1",28181,18181),("udp","::1",28181,18181)):
            op(c,"port_forward_create",{"box_id":ids["d"],"protocol":proto,"host_bind":bind,"host_port":port,"guest_port":gport},"n4-forward-"+proto+("-v6" if ":" in bind else "-v4"))
            forwards.append((proto,bind,port))
        print("N4_FORWARD_CREATED",flush=True)
        sockets=[]
        for proto,bind,port in forwards:
            s=socket.socket(socket.AF_INET6 if ":" in bind else socket.AF_INET,socket.SOCK_STREAM if proto=="tcp" else socket.SOCK_DGRAM)
            s.settimeout(2);s.connect((bind,port));sockets.append(s)
        def echo(s,tag):
            s.sendall(tag);assert s.recv(4096)==tag
        for s in sockets:echo(s,b"baseline")
        stop=threading.Event();errors=[];count=[0]
        def monitor():
            while not stop.is_set():
                try:
                    for s in sockets:echo(s,("n4-"+str(count[0])).encode())
                    count[0]+=1
                except Exception as exc:
                    errors.append(str(exc));return
                stop.wait(.05)
        t=threading.Thread(target=monitor,daemon=True);t.start()
        try:
            peer("a","b",True,"n4-ab-grant-01")
            udp("a","b",True);udp("b","a",False);udp("c","b",False);udp("a","c",False)
            peer("b","a",True,"n4-ba-grant-01")
            udp("b","a",True)
            peer("a","b",False,"n4-ab-revoke-01")
            udp("a","b",False);udp("b","a",True);udp("c","d",True)
            peer("a","b",True,"n4-ab-regrant-01")
            udp("a","b",True)
        finally:
            stop.set();t.join(timeout=5)
            for s in sockets:s.close()
        assert count[0]>5 and not errors,(count,errors)
        print("PR178_N4_UDP_FORWARD_CONTINUITY=PASS",count[0],flush=True)
        for n,u in ids.items():
            assert verify(project,"n4-"+n,u,"true")==before[n]
            assert instance(project,"n4-"+n)["status"]=="Running"
        rows=c.call("port_forward_list",{"box_id":ids["d"]}).structured["forwards"]
        assert len(rows)==4 and all(x["state"]=="CONFIRMED" and x["native_verified"] for x in rows),rows
        print("PR178_N4_FORWARD_DEVICE_POSTSTATE=PASS",flush=True)
if __name__=="__main__":
    main(*sys.argv[1:])
