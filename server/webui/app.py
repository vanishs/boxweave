"""Authenticated same-origin API, graph store and bounded batch executor."""

import hashlib
import json
import secrets
import threading
import time
import uuid
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
from urllib.parse import urlsplit
from argon2 import PasswordHasher
from argon2.exceptions import VerificationError, InvalidHashError
from flask import Flask, request, jsonify, send_from_directory
from werkzeug.exceptions import HTTPException
from .store import Store, Conflict, SCHEMA
from .graph import validate, markdown, text
from .provider import Provider, ProviderError, name, limits, port
from .access import AccessService
from .tenants import TenantService, CAPABILITIES, CREATE_BLOCK, tenant_id, resolve_box

HASHER = PasswordHasher(time_cost=3, memory_cost=65536, parallelism=2)
DUMMY = HASHER.hash("unused authentication timing placeholder")
ACTIONS = {
    "create",
    "start",
    "stop",
    "resize",
    "delete",
    "route-add",
    "route-delete",
    "passthrough-add",
    "passthrough-delete",
}


def create_app(cfg, provider=None):
    app = Flask(__name__, static_folder=None)
    app.config.update(MAX_CONTENT_LENGTH=1024 * 1024)
    store = Store(cfg["database"])
    api = provider or Provider(cfg)
    tenants = TenantService(store, api)
    access = AccessService(store, api, tenants)
    origin = cfg["origin"].rstrip("/")
    secure = origin.startswith("https://")
    pool = ThreadPoolExecutor(max_workers=2, thread_name_prefix="web-batch")
    lock = threading.Lock()
    app.extensions.update(store=store, provider=api, batch_pool=pool)
    # A crash leaves ambiguous receipts. Never automatically replay writes.
    with store.db() as db:
        for row in db.execute("SELECT * FROM jobs").fetchall():
            data = json.loads(row["data"])
            for i in data["items"]:
                if i["status"] in ("queued", "running"):
                    i["status"] = "unknown"
            data["finished"] = True
            db.execute(
                "UPDATE jobs SET data=? WHERE id=?", (json.dumps(data), row["id"])
            )

    def body():
        d = request.get_json()
        if not isinstance(d, dict):
            raise ValueError("JSON object required")
        return d

    def session():
        sid = hashlib.sha256(request.cookies.get("bw_session", "").encode()).hexdigest()
        with store.db() as db:
            return db.execute(
                "SELECT * FROM sessions WHERE id=? AND expires>?", (sid, time.time())
            ).fetchone()

    def same_origin():
        return request.headers.get("Origin") == origin

    @app.before_request
    def authorize():
        if request.path != "/healthz" and request.host != urlsplit(origin).netloc:
            return jsonify(error="Invalid host"), 400
        if request.path.startswith("/api/") and request.path not in (
            "/api/status",
            "/api/login",
        ):
            s = session()
            if not s:
                return jsonify(error="Authentication required"), 401
            if request.method not in ("GET", "HEAD"):
                if not same_origin() or not secrets.compare_digest(
                    request.headers.get("X-CSRF-Token", ""), s["csrf"]
                ):
                    return jsonify(error="CSRF check failed"), 403
        if request.path == "/api/login" and not same_origin():
            return jsonify(error="Origin check failed"), 403

    @app.after_request
    def headers(r):
        r.headers.update(
            {
                "Cache-Control": "no-store",
                "X-Content-Type-Options": "nosniff",
                "Referrer-Policy": "no-referrer",
                "X-Frame-Options": "DENY",
                "Content-Security-Policy": "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'",
            }
        )
        if secure:
            r.headers["Strict-Transport-Security"] = "max-age=31536000"
        return r

    @app.errorhandler(Exception)
    def error(e):
        if isinstance(e, HTTPException):
            return jsonify(error=e.name), e.code
        if isinstance(e, Conflict):
            return (
                jsonify(
                    error="Version conflict. Reload before saving; your local changes are retained."
                ),
                409,
            )
        if isinstance(e, KeyError):
            return jsonify(error="Not found"), 404
        if isinstance(e, ValueError):
            return jsonify(error=str(e)), 400
        if isinstance(e, ProviderError):
            return (
                jsonify(
                    error="Provider unavailable or request rejected",
                    outcome="unknown" if e.unknown else "rejected",
                ),
                e.status,
            )
        # Never include exception messages/tracebacks containing provider data.
        app.logger.error("Web request failed: %s", type(e).__name__)
        return jsonify(error="Internal operation failed"), 500

    @app.get("/healthz")
    def health():
        return jsonify(status="ok", schema=SCHEMA)

    @app.get("/api/status")
    def status():
        with store.db() as db:
            initialized = bool(db.execute("SELECT 1 FROM admin").fetchone())
        s = session()
        return jsonify(
            initialized=initialized,
            authenticated=bool(s),
            csrf=s["csrf"] if s else None,
            https=secure,
        )

    @app.post("/api/login")
    def login():
        d = body()
        if (
            set(d) != {"password"}
            or not isinstance(d["password"], str)
            or len(d["password"]) > 1024
        ):
            raise ValueError("Invalid credentials")
        # Trust only the local proxy's overwritten address; never forwarded headers.
        address = request.remote_addr or "unknown"
        with store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            db.execute("DELETE FROM attempts WHERE at<?", (time.time() - 900,))
            if (
                db.execute(
                    "SELECT count(*) FROM attempts WHERE address=?", (address,)
                ).fetchone()[0]
                >= 5
            ):
                return jsonify(error="Try again in 15 minutes"), 429
            db.execute("INSERT INTO attempts VALUES (?,?)", (address, time.time()))
            admin = db.execute("SELECT hash FROM admin WHERE id=1").fetchone()
        try:
            valid = HASHER.verify(admin[0] if admin else DUMMY, d["password"]) and bool(
                admin
            )
        except (VerificationError, InvalidHashError):
            valid = False
        if not valid:
            return (
                jsonify(error="Invalid credentials or administrator not initialized"),
                401,
            )
        sid = secrets.token_urlsafe(32)
        csrf = secrets.token_urlsafe(32)
        with store.db() as db:
            db.execute("DELETE FROM sessions WHERE expires<?", (time.time(),))
            db.execute(
                "INSERT INTO sessions VALUES (?,?,?)",
                (
                    hashlib.sha256(sid.encode()).hexdigest(),
                    csrf,
                    time.time() + 8 * 3600,
                ),
            )
            db.execute("DELETE FROM attempts WHERE address=?", (address,))
        r = jsonify(csrf=csrf)
        r.set_cookie(
            "bw_session",
            sid,
            secure=secure,
            httponly=True,
            samesite="Strict",
            max_age=8 * 3600,
            path="/",
        )
        return r

    @app.post("/api/logout")
    def logout():
        with store.db() as db:
            db.execute("DELETE FROM sessions WHERE id=?", (session()["id"],))
        r = jsonify(ok=True)
        r.delete_cookie("bw_session", secure=secure, httponly=True, samesite="Strict")
        return r

    @app.get("/api/tenant-access")
    def tenant_access_list():
        return jsonify(access.snapshot(tenants.snapshot(), selected_tenant()))

    @app.post("/api/tenant-access/plans")
    def tenant_access_plan():
        return jsonify(access.plan(body())), 201

    @app.post("/api/tenant-access/plans/<pid>/confirm")
    def tenant_access_confirm(pid):
        return jsonify(access.confirm(pid, body()))

    def selected_tenant():
        if set(request.args) - {"tenant"}:
            raise ValueError("Unsupported resource filter")
        return request.args.get("tenant", "")

    @app.get("/api/tenants")
    @app.get("/api/overview")
    def tenant_list():
        snap = tenants.snapshot()
        return jsonify(snap)

    @app.post("/api/tenant-selection")
    def tenant_selection():
        d = body()
        if set(d) != {"id"} or not isinstance(d["id"], str):
            raise ValueError("Tenant ID required")
        snap = tenants.snapshot()
        if d["id"] and d["id"] != "unresolved":
            tenants.record(snap, d["id"])
        store.setting("current_tenant", d["id"])
        return jsonify(current=d["id"])

    @app.post("/api/tenants")
    def create_tenant():
        d = body()
        if set(d) - {"name", "description"}:
            raise ValueError("Only local Tenant metadata can be created")
        display = text(d.get("name", ""), 120).strip()
        if not display:
            raise ValueError("Tenant name is required")
        result = store.save_tenant(
            "tenant_local_" + uuid.uuid4().hex,
            display,
            text(d.get("description", ""), 2000),
            None,
            0,
        )
        return jsonify(result), 201

    @app.get("/api/tenants/<tid>")
    def get_tenant(tid):
        snap = tenants.snapshot()
        t = tenants.record(snap, tid)
        return jsonify(
            {
                **t,
                "deletionBlockers": tenants.deletion_blockers(snap, t)
                + access.deletion_blockers(snap, t),
                "capabilities": snap["capabilities"],
            }
        )

    @app.patch("/api/tenants/<tid>")
    def edit_tenant(tid):
        d = body()
        if set(d) - {"name", "description", "version"} or type(d.get("version")) != int:
            raise ValueError("Tenant revision and editable metadata required")
        current = tenants.record(tenants.snapshot(), tid)
        if d["version"] != current["version"]:
            raise Conflict()
        display = text(d.get("name", current["name"]), 120).strip()
        if not display:
            raise ValueError("Tenant name is required")
        return jsonify(
            store.save_tenant(
                tid,
                display,
                text(d.get("description", current["description"]), 2000),
                current["providerKey"],
                d["version"],
            )
        )

    @app.delete("/api/tenants/<tid>")
    def delete_tenant(tid):
        d = body()
        if (
            set(d) != {"version", "confirm"}
            or d["confirm"] is not True
            or type(d["version"]) != int
        ):
            raise ValueError(
                "Explicit Tenant deletion confirmation and revision required"
            )
        snap = tenants.snapshot()
        current = tenants.record(snap, tid)
        blockers = tenants.deletion_blockers(snap, current) + access.deletion_blockers(
            snap, current
        )
        if blockers:
            return jsonify(error="; ".join(blockers)), 409
        if current["version"] == 0:
            raise ValueError("No local metadata exists to delete")
        store.delete_tenant(tid, d["version"])
        return jsonify(ok=True)

    @app.get("/api/tenants/<tid>/<view>")
    def tenant_resources(tid, view):
        if view not in ("boxes", "network", "monitoring"):
            raise KeyError(view)
        snap = tenants.snapshot()
        current = tenants.record(snap, tid)
        boxes = tenants.filtered(snap, tid)
        result = {
            "tenant": current,
            "containers": boxes,
            "guard": snap["guard"],
            "unreachableBackends": snap["unreachableBackends"],
            "capabilities": snap["capabilities"],
        }
        if view == "network":
            result.update(
                routes=tenants.routes(snap, "route", tid),
                passthrough=tenants.routes(snap, "passthrough", tid),
                tenantAccess=access.snapshot(snap, tid),
            )
        return jsonify(result)

    @app.get("/api/network/guard")
    def network_guard():
        snap = tenants.snapshot()
        return jsonify(
            guard=snap["guard"],
            tenants=snap["tenants"],
            capabilities=snap["capabilities"],
        )

    @app.get("/api/boxes")
    @app.get("/api/metrics")
    def boxes():
        snap = tenants.snapshot()
        tid = selected_tenant()
        return jsonify(
            containers=tenants.filtered(snap, tid),
            tenants=snap["tenants"],
            guard=snap["guard"],
            capabilities=snap["capabilities"],
            unreachableBackends=snap["unreachableBackends"],
        )

    @app.get("/api/boxes/<n>")
    def box(n):
        result = api.box(n)
        snap = tenants.snapshot()
        return jsonify(
            tenants.annotate(result, snap["guard"]["tenant"], snap["tenants"])
        )

    @app.get("/api/network/<kind>")
    def routes(kind):
        if kind not in ("route", "passthrough"):
            raise KeyError(kind)
        snap = tenants.snapshot()
        return jsonify(
            routes=tenants.routes(snap, kind, selected_tenant()),
            guard=snap["guard"],
            capabilities=snap["capabilities"],
        )

    def validate_items(action, items):
        if (
            action not in ACTIONS
            or not isinstance(items, list)
            or not 1 <= len(items) <= 50
        ):
            raise ValueError("Invalid batch (1–50 items)")
        result = []
        identities = set()
        for i in items:
            allowed = (
                {"username", "resources", "image", "tenantId"}
                if action in ("create", "resize")
                else (
                    {"username"}
                    if action in ("start", "stop", "delete")
                    else (
                        {"username", "domain", "targetPort", "description"}
                        if action == "route-add"
                        else (
                            {
                                "username",
                                "externalPort",
                                "targetPort",
                                "protocol",
                                "description",
                            }
                            if action == "passthrough-add"
                            else (
                                {"domain"}
                                if action == "route-delete"
                                else {"externalPort", "protocol"}
                            )
                        )
                    )
                )
            )
            if not isinstance(i, dict) or set(i) - allowed:
                raise ValueError("Unsupported batch field")
            if "username" in allowed:
                name(i.get("username"))
            if action in ("create", "resize"):
                limits(i.get("resources"), action == "create")
            if "tenantId" in i:
                if action != "create":
                    raise ValueError("Existing Box Tenant reassignment is unavailable")
                if i["tenantId"] is not None and not isinstance(i["tenantId"], str):
                    raise ValueError("Invalid Tenant selection")
            if action == "create":
                image = i.get("image", "images:ubuntu/24.04")
                if not isinstance(image, str) or not __import__("re").fullmatch(
                    r"[a-zA-Z0-9:/._-]{1,150}", image
                ):
                    raise ValueError("Invalid image")
                i = {**i, "image": image}
            if "targetPort" in allowed:
                port(i.get("targetPort"))
            if "externalPort" in allowed:
                port(i.get("externalPort"))
            if "protocol" in allowed and i.get("protocol") not in ("TCP", "UDP"):
                raise ValueError("Select TCP or UDP")
            if "domain" in allowed and not __import__("re").fullmatch(
                r"(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}",
                i.get("domain", ""),
            ):
                raise ValueError("Full domain name required")
            if "description" in i:
                text(i["description"], 500)
            identity = (
                i.get("username")
                if action in ("create", "start", "stop", "resize", "delete")
                else i.get("domain", str(i.get("externalPort")) + i.get("protocol", ""))
            )
            if identity in identities:
                raise ValueError("Duplicate target")
            identities.add(identity)
            result.append(i)
        return result

    @app.post("/api/plans")
    def plan():
        d = body()
        if set(d) != {"action", "items"}:
            raise ValueError("Invalid plan")
        action = d["action"]
        items = validate_items(action, d["items"])
        if action == "create":
            snapshot = tenants.snapshot()
            if snapshot["unreachableBackends"]:
                raise ValueError(
                    "Provider inventory is incomplete; name conflicts cannot be checked"
                )
            existing = {c["username"] for c in snapshot["containers"]}
            if any(i["username"] in existing for i in items):
                raise ValueError("Box name already exists")
        if action == "create":
            resolved = []
            for item in items:
                expected = tenants.creation_target(
                    snapshot, item.get("tenantId"), item["username"]
                )
                normalized = {k: v for k, v in item.items() if k != "tenantId"}
                if item.get("tenantId"):
                    normalized["networkTenant"] = expected["key"]
                resolved.append({**normalized, "expectedTenant": expected})
            items = resolved
        pid = str(uuid.uuid4())
        with store.db() as db:
            db.execute(
                "INSERT INTO plans(id,data,expires) VALUES (?,?,?)",
                (
                    pid,
                    json.dumps({"action": action, "items": items}),
                    time.time() + 300,
                ),
            )
        return jsonify(
            id=pid,
            action=action,
            items=items,
            expiresIn=300,
            capabilities=(
                snapshot["capabilities"]
                if action == "create"
                else tenants.capabilities()
            ),
            guard=snapshot["guard"] if action == "create" else None,
        )

    def run(jid):
        data = store.job(jid)
        for item in data["items"]:
            item["status"] = "running"
            store.job(jid, data)
            try:
                # Check again immediately before create; unknown earlier jobs reserve the name.
                if data["action"] == "create":
                    try:
                        api.box(item["target"]["username"])
                    except ProviderError as e:
                        if e.status != 404:
                            raise
                    else:
                        raise ValueError("Name exists; creation not retried")
                api.execute(data["action"], item["target"])
                item["status"] = (
                    "submitted" if data["action"] == "create" else "success"
                )
            except ProviderError as e:
                item["status"] = "unknown" if e.unknown else "failed"
                if e.unknown and data["action"] == "create":
                    try:
                        c = api.box(item["target"]["username"])
                        item["observedState"] = c.get("state", "unknown")
                        item["message"] = (
                            "Resource found; inspect readiness before continuing"
                        )
                    except ProviderError:
                        item["message"] = (
                            "Outcome unknown. Query the resource before any retry."
                        )
                else:
                    item["message"] = (
                        "Provider rejected or could not complete the operation"
                    )
            except ValueError as e:
                item.update(status="failed", message=str(e))
            except Exception:
                item.update(
                    status="unknown",
                    message="Interrupted operation; inspect actual resource state",
                )
            store.job(jid, data)
            store.audit(
                data["action"],
                item["target"].get(
                    "username",
                    item["target"].get(
                        "domain", str(item["target"].get("externalPort", ""))
                    ),
                ),
                item["status"],
            )
        data["finished"] = True
        store.job(jid, data)

    @app.post("/api/plans/<pid>/confirm")
    def confirm(pid):
        if body() != {"confirm": True}:
            raise ValueError("Explicit confirmation required")
        with lock:
            with store.db() as db:
                db.execute("BEGIN IMMEDIATE")
                row = db.execute(
                    "SELECT * FROM plans WHERE id=? AND used=0 AND expires>?",
                    (pid, time.time()),
                ).fetchone()
                if not row:
                    raise Conflict()
                d = json.loads(row["data"])
                if d["action"] == "tenant-access":
                    raise ValueError(
                        "Use the dedicated Tenant policy confirmation endpoint"
                    )
                if d["action"] == "create":
                    if any(
                        i.get("networkTenant") for i in d["items"]
                    ) and not tenants.snapshot()["capabilities"].get(
                        "sharedTenantCreate"
                    ):
                        raise ValueError(
                            "Provider no longer supports safe shared creation"
                        )
                    for r in db.execute("SELECT data FROM jobs"):
                        old = json.loads(r[0])
                        if old["action"] == "create" and any(
                            x["target"]["username"] == i["username"]
                            and x["status"]
                            in (
                                "queued",
                                "running",
                                "unknown",
                                "submitted",
                                "provisioning",
                                "verification-pending",
                                "degraded",
                            )
                            for x in old["items"]
                            for i in d["items"]
                        ):
                            raise ValueError(
                                "Name has an unresolved creation receipt; reconcile it first"
                            )
                # Bounded queue as well as bounded workers.
                pending = sum(
                    not json.loads(r[0]).get("finished", False)
                    for r in db.execute("SELECT data FROM jobs")
                )
                if pending >= 8:
                    raise ValueError("Batch queue is full")
                db.execute("UPDATE plans SET used=1 WHERE id=?", (pid,))
                jid = str(uuid.uuid4())
                job = {
                    "id": jid,
                    "action": d["action"],
                    "items": [{"target": i, "status": "queued"} for i in d["items"]],
                    "finished": False,
                }
                db.execute("INSERT INTO jobs VALUES (?,?)", (jid, json.dumps(job)))
            pool.submit(run, jid)
        return jsonify(job), 202

    @app.get("/api/jobs")
    def jobs():
        with store.db() as db:
            return jsonify(
                jobs=[
                    json.loads(r[0])
                    for r in db.execute(
                        "SELECT data FROM jobs ORDER BY rowid DESC LIMIT 100"
                    )
                ]
            )

    @app.get("/api/jobs/<jid>")
    def job(jid):
        return jsonify(store.job(jid))

    @app.post("/api/jobs/<jid>/reconcile")
    def reconcile(jid):
        d = store.job(jid)
        if not d.get("finished"):
            raise ValueError("Wait for batch completion")
        if d["action"] != "create":
            raise ValueError("Inspect the resource list to reconcile this operation")
        snap = tenants.snapshot()
        for i in d["items"]:
            if i["status"] in (
                "unknown",
                "submitted",
                "provisioning",
                "verification-pending",
                "degraded",
            ):
                try:
                    c = tenants.annotate(
                        api.box(i["target"]["username"]),
                        snap["guard"]["tenant"],
                        snap["tenants"],
                    )
                    i["observedState"] = c.get("state", "unknown")
                    i["observedTenant"] = c["tenant"]
                    expected = (
                        i["target"]
                        .get("expectedTenant", {})
                        .get("id", tenant_id(i["target"]["username"]))
                    )
                    if c.get("state") in (
                        "CONTAINER_STATE_CREATING",
                        "CONTAINER_STATE_PROVISIONING",
                    ):
                        i["status"] = "provisioning"
                    elif c.get("state") == "CONTAINER_STATE_RUNNING":
                        i["status"] = (
                            "success"
                            if c["tenant"]["id"] == expected
                            and c["tenant"]["guard"]["status"] == "verified"
                            else "verification-pending"
                        )
                        if c["tenant"]["confirmed"] and c["tenant"]["id"] != expected:
                            i["status"] = "degraded"
                    else:
                        i["status"] = "verification-pending"
                    i["verification"] = {
                        "created": True,
                        "provisioning": c.get("state")
                        in ("CONTAINER_STATE_CREATING", "CONTAINER_STATE_PROVISIONING"),
                        "tenantVerified": c["tenant"]["confirmed"]
                        and c["tenant"]["id"] == expected,
                        "guardVerified": c["tenant"]["guard"]["status"] == "verified",
                        "networkConnectivity": "not_tested",
                    }
                    if i["status"] == "success":
                        i["status"] = "guard-verified"
                    i["message"] = c["tenant"]["guard"]["reason"]
                except ProviderError as e:
                    i["message"] = (
                        "Not currently found; absence does not authorize automatic retry"
                        if e.status == 404
                        else "Query failed"
                    )
        store.job(jid, d)
        return jsonify(d)

    @app.get("/api/graphs")
    def graphs():
        return jsonify(graphs=store.graphs(), current=store.setting("current"))

    @app.post("/api/graphs")
    def new_graph():
        g = store.save(str(uuid.uuid4()), 0, validate(body()))
        store.setting("current", g["id"])
        return jsonify(g), 201

    @app.get("/api/graphs/<gid>")
    def graph(gid):
        return jsonify(store.graph(gid))

    @app.put("/api/graphs/<gid>")
    def save_graph(gid):
        d = body()
        version = d.pop("version", None)
        if type(version) != int or version < 1:
            raise ValueError("Revision required")
        return jsonify(store.save(gid, version, validate(d)))

    @app.post("/api/graphs/<gid>/select")
    def select(gid):
        store.graph(gid)
        store.setting("current", gid)
        return jsonify(ok=True)

    @app.delete("/api/graphs/<gid>")
    def delete_graph(gid):
        d = body()
        if set(d) != {"version", "confirm"} or d["confirm"] is not True:
            raise ValueError("Confirmation and revision required")
        with store.db() as db:
            if (
                db.execute(
                    "DELETE FROM graphs WHERE id=? AND version=?", (gid, d["version"])
                ).rowcount
                != 1
            ):
                raise Conflict()
            db.execute("DELETE FROM settings WHERE key=? AND value=?", ("current", gid))
        return jsonify(ok=True)

    @app.get("/api/graphs/<gid>/markdown")
    def export(gid):
        return jsonify(markdown=markdown(store.graph(gid)))

    @app.get("/")
    def index():
        return send_from_directory(cfg["static"], "index.html")

    @app.get("/assets/<path:asset>")
    def asset(asset):
        return send_from_directory(Path(cfg["static"]) / "assets", asset)

    return app
