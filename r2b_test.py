import hashlib, json, sqlite3, urllib.request, urllib.error, urllib.parse

B="http://127.0.0.1:18844"; W="http://127.0.0.1:18640"
ORIGIN_WEB=json.load(open("/home/boxweave/boxweave/runtime.json"))["origin"]
PW=open("/home/boxweave/boxweave/initial-admin-password").read().strip()
op=urllib.request.build_opener()

def wreq(method,path,body=None,headers=None,cookie=None):
    h={"Origin":ORIGIN_WEB,"Accept":"application/json"}
    if headers: h.update(headers)
    if cookie: h["Cookie"]=cookie
    data=json.dumps(body).encode() if body is not None else None
    if data is not None: h["Content-Type"]="application/json"
    r=urllib.request.Request(W+path,data=data,headers=h,method=method)
    try:
        with op.open(r,timeout=30) as resp: return resp.status, dict(resp.headers), resp.read()
    except urllib.error.HTTPError as e: return e.code, dict(e.headers), e.read()

def mcp(path,token,link=False):
    h={"Content-Type":"application/json","Accept":"application/json, text/event-stream","MCP-Protocol-Version":"2025-11-25"}
    if not link: h["Authorization"]="Bearer "+token
    data=json.dumps({"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}).encode()
    r=urllib.request.Request(B+path,data=data,headers=h,method="POST")
    try:
        with op.open(r,timeout=60) as resp: return resp.status, resp.read()
    except urllib.error.HTTPError as e: return e.code, e.read()

s,h,b=wreq("POST","/api/login",{"password":PW})
cookie=h.get("Set-Cookie","").split(";")[0]
csrf=json.loads(b)["csrf"]
print("web login:", s)
AH={"X-CSRF-Token":csrf}

s,h,b=wreq("POST","/api/remote-mcp/credentials",{"kind":"api","label":"r2b-api"},AH,cookie)
api=json.loads(b); print("issue api:", s, "| token len:", len(api.get("secret","")), "| url:", api.get("connection_url"))
s,h,b=wreq("POST","/api/remote-mcp/credentials",{"kind":"link","label":"r2b-link"},AH,cookie)
link=json.loads(b); print("issue link:", s, "| path contains secret:", "/mcp/link/" in link.get("connection_url",""))
linkurl=link["connection_url"].replace("https://mcp.ser.boxweave-test.internal","http://127.0.0.1:18844")
linkval=linkurl.rsplit("/",1)[1]

print("api bearer tools/list:", mcp("/mcp",api["secret"])[0], json.loads(mcp("/mcp",api["secret"])[1]).get("result",{}).get("tools").__len__())
s,b=mcp(linkurl.replace("http://127.0.0.1:18844","")and"/mcp/link/"+linkval,None,link=True)
print("link tools/list:", s, "count:", len(json.loads(b).get("result",{}).get("tools",[])))
print("guessed link:", mcp("/mcp/link/"+("x"*43),None,link=True)[0])

s,h,b=wreq("GET","/api/remote-mcp",None,None,cookie)
lst=json.loads(b)
print("web list:", s, "| credentials shown:", len(lst.get("credentials",[])), "| raw secret exposed:", any(api["secret"] in json.dumps(c) for c in lst.get("credentials",[])))

db=sqlite3.connect("file:/home/boxweave/boxweave-state/state.sqlite3?mode=ro",uri=True)
cols=[r[1] for r in db.execute("pragma table_info(remote_credentials)")]
rows=db.execute("select * from remote_credentials").fetchall()
blob=json.dumps([dict(zip(cols,r)) for r in rows])
print("db columns:", cols)
print("raw api token in db:", api["secret"] in blob, "| raw link secret in db:", linkval in blob)
print("api digest present:", hashlib.sha256(api["secret"].encode()).hexdigest()[:16] in blob or hashlib.sha256(("https://mcp.ser.boxweave-test.internal/mcp"+api["secret"]).encode()).hexdigest()[:16] in blob or "digest" in cols)

ident_api=[c for c in lst["credentials"] if c.get("label")=="r2b-api"][0]["id"]
ident_link=[c for c in lst["credentials"] if c.get("label")=="r2b-link"][0]["id"]
print("revoke api:", wreq("DELETE","/api/remote-mcp/credentials/"+ident_api,None,AH,cookie)[0])
print("revoke link:", wreq("DELETE","/api/remote-mcp/credentials/"+ident_link,None,AH,cookie)[0])
print("api after revoke:", mcp("/mcp",api["secret"])[0], mcp("/mcp",api["secret"])[1][:40])
print("link after revoke:", mcp("/mcp/link/"+linkval,None,link=True)[0])
