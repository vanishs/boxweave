#!/usr/bin/env python3
"""Check address and policy isolation across actual same-name reincarnation."""
import socket
import sqlite3
import sys
import time
from mcp_testkit import LoopbackMCP
from n2_installed_lifecycle import op,token
from pr178_n2_scoped_lifecycle import binding,verify,acls
from pr178_n4_native import guest,UDP,SERVER

def main(project,receipt):
    with sqlite3.connect("file:/var/lib/boxweave/ledger.sqlite?mode=ro",uri=True) as db:
        ids=dict(db.execute("SELECT name,uuid FROM owned_boxes WHERE state='OWNED' AND name IN ('n4-a','n4-b','n4-c','n4-d')"))
    assert len(ids)==4
    a,b,cid,d=[ids["n4-"+n] for n in "abcd"]
    prior=binding(b)
    sockets=[]
    for family,addr in ((socket.AF_INET,"127.0.0.1"),(socket.AF_INET6,"::1")):
        s=socket.socket(family,socket.SOCK_STREAM);s.settimeout(2);s.connect((addr,28182));sockets.append(s)
    def echo(tag):
        for s in sockets:
            s.sendall(tag);assert s.recv(4096)==tag
    try:
        echo(b"before-reincarnation")
        with LoopbackMCP("http://127.0.0.1:18650/mcp",token=token(receipt),allow_mutations=True,timeout=60) as client:
            op(client,"box_peer_set",{"source_box_id":b,"target_box_id":a,"allow":False},"n4-ba-final-revoke")
            op(client,"box_stop",{"box_id":b},"n4-b-retire-stop")
            op(client,"box_delete",{"box_id":b},"n4-b-retire-delete")
            assert prior["acl"] not in acls()
            echo(b"deleted")
            new=op(client,"box_create",{"name":"n4-b","cpu":1,"memory_mib":256,"disk_gib":2,"image":"local:boxweave-ubuntu-24.04","start":True,"internet":True},"n4-b-recreate")["box_uuid"]
            now=binding(new)
            assert new!=b and all(prior[k]!=now[k] for k in ("acl","mac","ipv4","ipv6"))
            verify(project,"n4-b",new,"true")
            guest(project,"n4-b","python3","-c","import subprocess,sys;subprocess.Popen(['python3','-u','-c',sys.argv[1]],stdin=subprocess.DEVNULL,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,start_new_session=True)",SERVER)
            time.sleep(2)
            for af in ("ipv4","ipv6"):
                guest(project,"n4-a","python3","-c",UDP,now[af],"no")
                guest(project,"n4-b","python3","-c",UDP,binding(a)[af],"no")
                guest(project,"n4-c","python3","-c",UDP,binding(d)[af],"yes")
            # Replay the old admitted revoke even though original B is retired.
            old=op(client,"box_peer_set",{"source_box_id":b,"target_box_id":a,"allow":False},"n4-ba-final-revoke")
            assert old["state"]=="SUCCEEDED"
            echo(b"recreated")
            for af in ("ipv4","ipv6"):
                guest(project,"n4-a","python3","-c",UDP,now[af],"no")
            print("PR178_N4_REINCARNATION=PASS",b,new,flush=True)
    finally:
        for s in sockets:s.close()

if __name__=="__main__":
    main(*sys.argv[1:])
