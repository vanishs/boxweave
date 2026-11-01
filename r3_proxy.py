import base64, hashlib, http.client, json, secrets, ssl, subprocess, sys, threading, time, urllib.parse

# --- TLS proxy: terminate TLS on 127.0.0.1:18901, forward raw to 18844 ---
import socketserver, socket
UPSTREAM=("127.0.0.1",18844)
ctx=ssl.SECLEVEL_FIX=ssl.create_default_context(ssl.Purpose.CLIENT_AUTH)
ctx.load_cert_chain("/tmp/proxy-cert.pem","/tmp/proxy-key.pem")
class Handler(socketserver.BaseRequestHandler):
    def handle(self):
        try:
            tls=ctx.wrap_socket(self.request,server_side=True)
        except ssl.SSLError: return
        up=socket.create_connection(UPSTREAM,timeout=30)
        stop=threading.Event()
        def pump(a,b):
            try:
                while True:
                    d=a.recv(65536)
                    if not d: break
                    b.sendall(d)
            except OSError: pass
            finally:
                try: b.shutdown(socket.SHUT_WR)
                except OSError: pass
        t1=threading.Thread(target=pump,args=(tls,up),daemon=True)
        t2=threading.Thread(target=pump,args=(up,tls),daemon=True)
        t1.start(); t2.start(); t1.join(60); t2.join(1)
        up.close(); tls.close()
class Server(socketserver.ThreadingTCPServer):
    allow_reuse_address=True; daemon_threads=True
srv=Server(("127.0.0.1",18901),Handler)
threading.Thread(target=srv.serve_forever,daemon=True).start()
time.sleep(0.5)
print("proxy up on 127.0.0.1:18901 (TLS, self-signed)")

# --- client through the proxy ---
ORIGIN="https://mcp.ser.boxweave-test.internal"; RESOURCE=ORIGIN+"/mcp"
PW=open("/home/boxweave/boxweave/initial-admin-password").read().strip()
cctx=ssl.create_default_context(); cctx.check_hostname=False; cctx.verify_mode=ssl.CERT_NONE
class NoRedirect(http.client.HTTPRedirectHandler):
    def redirect_request(self,*a,**k): return None
op=urllib.request.build_opener(NoRedirect,urllib.request.HTTPSHandler(context=cctx))
def req(method,path,headers=None,json_body=None,form=None,follow=True):
    host="127.0.0.1:18901"
    h={"Accept":"application/json, text/event-stream","Host":ORIGIN.split("//")[1]}
    if headers: h.update(headers)
    body=None
    if json_body is not None: body=json.dumps(json_body).encode(); h["Content-Type"]="application/json"
    elif form is not None: body=urllib.parse.urlencode(form).encode(); h["Content-Type"]="application/x-www-form-urlencoded"
    r=urllib.request.Request("https://"+host+path,data=body,headers=h,method=method)
    o=op if follow else urllib.request.build_opener(NoRedirect,urllib.request.HTTPSHandler(context=cctx))
    try:
        with o.open(r,timeout=60) as resp: return resp.status, dict(resp.headers), resp.read()
    except urllib.error.HTTPError as e: return e.code, dict(e.headers), e.read()
import urllib.error, urllib.request

s,h,b=req("GET","/.well-known/oauth-protected-resource/mcp")
print("proxy: resource metadata:", s, json.loads(b).get("resource"))
s,h,b=req("GET","/.well-known/oauth-authorization-server")
print("proxy: AS metadata:", s, json.loads(b).get("issuer"))
s,h,b=req("POST","/register",json_body={"client_name":"r3proxy","redirect_uris":["http://127.0.0.1:8765/cb"],"token_endpoint_auth_method":"none"})
CID=json.loads(b)["client_id"]; print("proxy: DCR:", s)
verifier=secrets.token_urlsafe(48)
challenge=base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest()).rstrip(b"=").decode()
qs=urllib.parse.urlencode({"response_type":"code","client_id":CID,"redirect_uri":"http://127.0.0.1:8765/cb",
    "code_challenge":challenge,"code_challenge_method":"S256","state":"proxy-st-1","resource":RESOURCE,"scope":"mcp:operator"})
s,h,b=req("GET","/authorize?"+qs)
pending=b.decode().split('name="pending" value="')[1].split('"')[0]
s,h,b=req("POST","/authorize",form={"pending":pending,"password":PW,"approve":"yes"},headers={"Origin":ORIGIN},follow=False)
loc=h.get("Location",""); code=urllib.parse.parse_qs(urllib.parse.urlsplit(loc).query)["code"][0]
print("proxy: authorize+approve:", s, "| state ok:", "state=proxy-st-1" in loc)
s,h,b=req("POST","/token",form={"grant_type":"authorization_code","code":code,"client_id":CID,
    "redirect_uri":"http://127.0.0.1:8765/cb","code_verifier":verifier,"resource":RESOURCE})
AT=json.loads(b)["access_token"]; print("proxy: token:", s)
INIT={"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"r3","version":"0"}}}
s,h,b=req("POST","/mcp",json_body=INIT,headers={"Authorization":"Bearer "+AT})
print("proxy: initialize:", s, json.loads(b).get("result",{}).get("serverInfo"))
s,h,b=req("POST","/mcp",json_body={"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}},headers={"Authorization":"Bearer "+AT})
print("proxy: tools/list:", s, "count:", len(json.loads(b).get("result",{}).get("tools",[])))
s,h,b=req("POST","/mcp",json_body=INIT,headers={"Authorization":"Bearer "+AT,"Host":"evil.example"})
print("proxy: spoofed Host still rejected:", s, b[:40])
srv.shutdown()
print("PROXY DONE")
