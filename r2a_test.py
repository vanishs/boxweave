import base64, hashlib, json, secrets, urllib.request, urllib.error, urllib.parse

B="http://127.0.0.1:18844"
ORIGIN="https://mcp.ser.boxweave-test.internal"
RESOURCE=ORIGIN+"/mcp"
PW=open("/home/boxweave/boxweave/initial-admin-password").read().strip()

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self,*a,**k): return None
op=urllib.request.build_opener(NoRedirect)

def req(method,path,data=None,headers=None,json_body=None,form=None,follow=True):
    h={"Accept":"application/json, text/event-stream"}
    if headers: h.update(headers)
    body=None
    if json_body is not None: body=json.dumps(json_body).encode(); h["Content-Type"]="application/json"
    elif form is not None: body=urllib.parse.urlencode(form).encode(); h["Content-Type"]="application/x-www-form-urlencoded"
    elif data is not None: body=data
    r=urllib.request.Request(B+path,data=body,headers=h,method=method)
    o=op if not follow else urllib.request.build_opener()
    try:
        with o.open(r,timeout=60) as resp: return resp.status, dict(resp.headers), resp.read()
    except urllib.error.HTTPError as e: return e.code, dict(e.headers), e.read()

s,h,b=req("GET","/.well-known/oauth-protected-resource/mcp")
rm=json.loads(b); print("resource metadata:", s, rm.get("resource"), rm.get("authorization_servers"))
s,h,b=req("GET","/.well-known/oauth-authorization-server")
am=json.loads(b); print("AS metadata:", s, am.get("issuer"), "| methods:", am.get("code_challenge_methods_supported"))

s,h,b=req("POST","/register",json_body={"client_name":"r2probe","redirect_uris":["http://127.0.0.1:8765/callback"],"token_endpoint_auth_method":"none"})
reg=json.loads(b); print("register:", s, "client_id len:", len(reg.get("client_id","")))
CID=reg["client_id"]

verifier=secrets.token_urlsafe(48)
challenge=base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest()).rstrip(b"=").decode()
qs=urllib.parse.urlencode({"response_type":"code","client_id":CID,"redirect_uri":"http://127.0.0.1:8765/callback",
    "code_challenge":challenge,"code_challenge_method":"S256","state":"st-123456","resource":RESOURCE,"scope":"mcp:operator"})
s,h,b=req("GET","/authorize?"+qs)
pending=b.decode().split('name="pending" value="')[1].split('"')[0]
print("authorize form:", s, "pending len:", len(pending))
s,h,b=req("POST","/authorize",form={"pending":pending,"password":PW},headers={"Origin":ORIGIN})
print("no-approve checkbox:", s, b[:40])
s,h,b=req("GET","/authorize?"+urllib.parse.urlencode({"response_type":"code","client_id":CID,"redirect_uri":"http://evil.example/cb",
    "code_challenge":challenge,"code_challenge_method":"S256","state":"x","resource":RESOURCE}))
print("evil redirect_uri:", s, b[:50])

s,h,b=req("POST","/authorize",form={"pending":pending,"password":PW,"approve":"yes"},headers={"Origin":ORIGIN},follow=False)
loc=h.get("Location","")
print("approve:", s, "location has code:", "code=" in loc, "| state ok:", "state=st-123456" in loc)
code=urllib.parse.parse_qs(urllib.parse.urlsplit(loc).query)["code"][0]

s,h,b=req("POST","/token",form={"grant_type":"authorization_code","code":code,"client_id":CID,
    "redirect_uri":"http://127.0.0.1:8765/callback","code_verifier":secrets.token_urlsafe(48),"resource":RESOURCE})
print("wrong verifier:", s, b[:40])
s,h,b=req("POST","/authorize",form={"pending":pending,"password":PW,"approve":"yes"},headers={"Origin":ORIGIN},follow=False)
print("replayed pending:", s, b[:40])

s,h,b=req("POST","/token",form={"grant_type":"authorization_code","code":code,"client_id":CID,
    "redirect_uri":"http://127.0.0.1:8765/callback","code_verifier":verifier,"resource":RESOURCE})
tok=json.loads(b); print("token:", s, "| access len:", len(tok.get("access_token","")), "| refresh:", "refresh_token" in tok, "| type:", tok.get("token_type"))
AT=tok["access_token"]
s,h,b=req("POST","/token",form={"grant_type":"authorization_code","code":code,"client_id":CID,
    "redirect_uri":"http://127.0.0.1:8765/callback","code_verifier":verifier,"resource":RESOURCE})
print("replayed code:", s, b[:40])

INIT={"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"r2probe","version":"0"}}}
AH={"Authorization":"Bearer "+AT}
s,h,b=req("POST","/mcp",json_body=INIT,headers=AH)
print("initialize:", s, json.loads(b).get("result",{}).get("serverInfo"))
s,h,b=req("POST","/mcp",json_body={"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}},headers=AH)
tools=json.loads(b).get("result",{}).get("tools",[])
print("tools/list:", s, "count:", len(tools))
s,h,b=req("POST","/mcp",json_body={"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_containers","arguments":{}}},headers=AH)
print("tools/call:", s, "hello-py present:", "hello-py" in b.decode())
s,h,b=req("GET","/mcp",headers=AH); print("GET /mcp:", s, b[:40])
s,h,b=req("DELETE","/mcp",headers=AH); print("DELETE /mcp:", s, b[:40])
s,h,b=req("POST","/mcp",json_body=INIT,headers=dict(AH,Accept="text/plain")); print("bad Accept:", s, b[:40])
s,h,b=req("POST","/mcp",json_body=INIT,headers=dict(AH,**{"MCP-Protocol-Version":"2025-01-01"})); print("bad protocol version:", s, b[:40])
s,h,b=req("POST","/mcp",json_body={"jsonrpc":"2.0","method":"notifications/initialized"},headers=AH); print("notification:", s)
s,h,b=req("POST","/mcp",json_body={"jsonrpc":"1.0","id":1,"method":"x","params":{}},headers=AH); print("bad jsonrpc body:", s, b[:40])
s,h,b=req("POST","/mcp",json_body=INIT,headers={"Authorization":"Bearer wrong-token"}); print("wrong bearer:", s, "| WWW-Authenticate:", ("www-authenticate" in {k.lower() for k in h}), b[:40])
if tok.get("refresh_token"):
    s,h,b=req("POST","/token",form={"grant_type":"refresh_token","refresh_token":tok["refresh_token"],"client_id":CID,"resource":RESOURCE})
    print("refresh:", s, "new access:", "access_token" in json.loads(b) if s==200 else b[:40])
    s,h,b=req("POST","/token",form={"grant_type":"refresh_token","refresh_token":tok["refresh_token"],"client_id":CID,"resource":RESOURCE})
    print("replayed refresh:", s, b[:40])
open("/tmp/r2_state.json","w").write(json.dumps({"access_token":AT}))
print("STATE saved")
