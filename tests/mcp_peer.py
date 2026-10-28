"""Synthetic protocol peer. No network or real provider credentials."""
import json
import sys
mode=sys.argv[1] if len(sys.argv)>1 else "valid"
initialized=False
for line in sys.stdin:
    m=json.loads(line)
    method=m["method"]
    if method=="notifications/initialized":
        initialized=True
        continue
    if method=="initialize":
        result={"protocolVersion":"2024-11-05","capabilities":{"tools":{}}}
    elif not initialized:
        raise SystemExit(4)
    elif method=="tools/list":
        tools=[{"name":"get_container","inputSchema":{"properties":{"username":{}}}},
               {"name":"connect","inputSchema":{"properties":{"box":{},"exec":{}}}}]
        result={"tools":tools}
        if mode=="paged" and "cursor" not in m.get("params",{}):
            result={"tools":[],"nextCursor":"page-two"}
        if mode=="repeat":
            result={"tools":[],"nextCursor":"same"}
    elif method=="tools/call":
        p=m["params"]
        if mode=="malformed":
            result={}
        elif mode=="denied":
            result={"isError":True,"content":[{"type":"text","text":"denied"}]}
        elif p["name"]=="get_container":
            c={"name":"fixture-box-container","username":"fixture-box","createdAt":"123",
               "state":"CONTAINER_STATE_RUNNING"}
            result={"content":[{"type":"text","text":json.dumps({"container":c})}]}
        else:
            out="other-user\nfixture-box-container\n" if mode=="identity" else "fixture-box\nfixture-box-container\n"
            if m["params"]["arguments"]["exec"]!="whoami && hostname":
                out="hello world\n"
            result={"content":[{"type":"text","text":"exit_code: 0\n\n--- stdout ---\n"+out}]}
    else:
        raise SystemExit(3)
    response={"jsonrpc":"2.0","id":m["id"],"result":result}
    if mode=="wrong-id":
        response["id"]+=1
    print(json.dumps(response),flush=True)
