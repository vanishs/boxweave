"""HTTP only adapts typed application actions; no caller-defined commands."""
import hashlib
import hmac
import json
import secrets
import time
from flask import Flask, request, jsonify, send_from_directory
from werkzeug.security import check_password_hash
from .domain import Fault
from .graphs import Graphs

def create_app(service, origin, dist, secure=True, clock=time.time):
    app = Flask(__name__, static_folder=None)
    app.config.update(MAX_CONTENT_LENGTH=16384)
    store = service.store
    graphs = Graphs(store)
    def body(keys):
        data = request.get_json(silent=True)
        if not isinstance(data, dict) or set(data) != set(keys):
            raise Fault("INVALID_INPUT", 400)
        return data
    def session():
        token = request.cookies.get("bw_session", "")
        if not token or len(token)>128:
            return None
        digest = hashlib.sha256(token.encode()).hexdigest()
        with store.db() as db:
            row = db.execute("SELECT * FROM sessions WHERE id=? AND expires>?",(digest,clock())).fetchone()
        return dict(row) if row else None
    @app.before_request
    def protect():
        if not request.path.startswith("/api/"):
            return None
        if request.method != "GET" and request.headers.get("Origin") != origin:
            raise Fault("ORIGIN_DENIED",403)
        if request.path == "/api/login":
            return None
        row = session()
        if row is None:
            raise Fault("AUTH_REQUIRED",401)
        if request.method != "GET" and not hmac.compare_digest(request.headers.get("X-CSRF-Token",""), row["csrf"]):
            raise Fault("CSRF_DENIED",403)
    @app.after_request
    def headers(response):
        response.headers.update({"Cache-Control":"no-store", "X-Content-Type-Options":"nosniff",
            "Content-Security-Policy":"default-src 'self'; script-src 'self'; style-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'",
            "Referrer-Policy":"no-referrer"})
        return response
    @app.errorhandler(Fault)
    def fault(error):
        return jsonify(code=error.code), error.status
    @app.errorhandler(413)
    def oversized(_):
        return jsonify(code="REQUEST_TOO_LARGE"),413
    @app.errorhandler(Exception)
    def internal(_):
        return jsonify(code="INTERNAL_ERROR"),500
    @app.post("/api/login")
    def login():
        data = body(["password"])
        if not isinstance(data["password"],str) or len(data["password"])>1024:
            raise Fault("INVALID_INPUT",400)
        now=clock()
        with store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("DELETE FROM attempts WHERE at<?",(now-60,))
            if db.execute("SELECT count(*) FROM attempts").fetchone()[0]>=5:
                raise Fault("RATE_LIMITED",429)
            db.execute("INSERT INTO attempts VALUES(?)",(now,))
            row=db.execute("SELECT password FROM admin WHERE id=1").fetchone()
        if row is None or not check_password_hash(row[0],data["password"]):
            raise Fault("LOGIN_FAILED",401)
        token,csrf=secrets.token_urlsafe(32),secrets.token_urlsafe(32)
        old=session()
        with store.db() as db:
            db.execute("DELETE FROM sessions WHERE expires<?",(now,))
            if old:
                db.execute("DELETE FROM sessions WHERE id=?",(old["id"],))
            db.execute("INSERT INTO sessions VALUES(?,?,?)",(hashlib.sha256(token.encode()).hexdigest(),csrf,now+3600))
        response=jsonify(csrf=csrf)
        response.set_cookie("bw_session",token,httponly=True,secure=secure,samesite="Strict",max_age=3600)
        return response
    @app.get("/api/session")
    def who():
        return jsonify(csrf=session()["csrf"],actor="operator",mode=service.provider.mode)
    @app.post("/api/logout")
    def logout():
        with store.db() as db:
            db.execute("DELETE FROM sessions WHERE id=?",(session()["id"],))
        response=jsonify(code="OK")
        response.delete_cookie("bw_session")
        return response
    @app.get("/api/resources")
    def resources():
        # Only the explicit operator-reviewed allowlist, not arbitrary provider inventory.
        logicals = ([service.provider.target.logical] if service.provider.mode=="fixture"
                    else list(service.provider.config["targets"]))
        return jsonify(resources=[service.provider.observe(x).data() for x in logicals],
                       mode=service.provider.mode)
    @app.get("/api/projects")
    def projects():
        return jsonify(projects=service.projects())
    @app.post("/api/projects")
    def bind():
        d=body(["name","logical","workspace"])
        return jsonify(service.bind(d["name"],d["logical"],d["workspace"])),201
    @app.put("/api/projects/<pid>")
    def rebind(pid):
        d=body(["name","logical","workspace","revision"])
        return jsonify(service.bind(d["name"],d["logical"],d["workspace"],pid,d["revision"]))
    @app.post("/api/plans")
    def plan():
        d=body(["project","revision"])
        return jsonify(service.plan(d["project"],d["revision"])),201
    @app.post("/api/plans/<pid>/confirm")
    def confirm(pid):
        d=body(["confirm"])
        if d["confirm"] is not True:
            raise Fault("CONFIRMATION_REQUIRED",400)
        return jsonify(service.confirm(pid)),202
    @app.get("/api/operations")
    def receipts():
        return jsonify(operations=service.list_receipts())
    @app.get("/api/operation-history")
    def receipt_history():
        if set(request.args) - {"limit", "cursor"} or any(len(request.args.getlist(key)) != 1 for key in request.args):
            raise Fault("INVALID_QUERY", 400)
        raw_limit = request.args.get("limit", "20")
        if not raw_limit.isascii() or not raw_limit.isdigit() or len(raw_limit) > 3:
            raise Fault("INVALID_LIMIT", 400)
        return jsonify(service.receipt_history(limit=int(raw_limit), cursor=request.args.get("cursor")))
    @app.get("/api/operations/<oid>")
    def receipt(oid):
        r=store.receipt(oid)
        r["result"]=json.loads(r["result"]) if r["result"] else None
        return jsonify(r)
    @app.get("/api/operations/<oid>/observations")
    def observations(oid):
        return jsonify(observations=service.observations(oid))
    @app.post("/api/operations/<oid>/reconcile")
    def reconcile(oid):
        body([])
        return jsonify(service.reconcile(oid))
    @app.get("/api/graphs")
    def graph_list():return jsonify(graphs=graphs.list())
    @app.post("/api/graphs")
    def graph_create():return jsonify(graphs.save(body(["document"])["document"])),201
    @app.get("/api/graphs/<gid>")
    def graph_get(gid):return jsonify(graphs.get(gid))
    @app.put("/api/graphs/<gid>")
    def graph_save(gid):
        d=body(["document","revision"])
        return jsonify(graphs.save(d["document"],gid,d["revision"]))
    @app.delete("/api/graphs/<gid>")
    def graph_delete(gid):
        graphs.delete(gid,body(["revision"])["revision"])
        return jsonify(code="OK")
    @app.get("/api/graphs/<gid>/markdown")
    def graph_export(gid):return jsonify(graphs.export(gid))
    @app.post("/api/graphs/<gid>/copies")
    def graph_copy(gid):return jsonify(graphs.copied(gid,body(["revision"])["revision"]))
    @app.get("/")
    def index():
        return send_from_directory(dist,"index.html")
    @app.get("/assets/<path:asset>")
    def asset(asset):
        return send_from_directory(str(dist)+"/assets",asset)
    return app
