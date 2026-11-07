"""Optional inbound Streamable HTTP MCP with single-operator authentication.

Not a generic Web tools/call route. Requests go through the existing
server-owned mcp_gate forced-command implementation and its trust checks.
"""
import html
import json
import os
from pathlib import Path
import re
import secrets
import subprocess
import sys
import threading
from urllib.parse import urlencode, urlsplit

from flask import Flask, Response, jsonify, redirect, request

from .remote_auth import RemoteCredentials, RemoteError, TOKEN_PATTERN, valid_url
from .mcp_gate import SERVER_DESCRIPTION, SERVER_INSTRUCTIONS, SERVER_INFO_VERSION
from .remote_create import CreateFault, CreateJournal, validated_request, validate_upstream_payload

PROTOCOL = "2025-11-25"
GATE = "/usr/local/lib/boxweave/default/mcp_gate.py"
METHODS = frozenset(("tools/list", "tools/call", "resources/list", "resources/read",
                     "prompts/list", "prompts/get", "ping"))
MAX_RESULT = 2_097_152


CREATE_STATUS_TOOL = {
    "name": "boxweave_create_status",
    "description": ("Read the durable HTTP MCP create receipt by original task_id, "
                    "request_key or username (exactly one). Read-only; never "
                    "retries a create. UNKNOWN requires separate inspection."),
    "inputSchema": {"type": "object", "additionalProperties": False,
                    "properties": {"task_id": {"type": "string"},
                                   "request_key": {"type": "string"},
                                   "username": {"type": "string"}}},
    "annotations": {"readOnlyHint": True, "destructiveHint": False,
                    "idempotentHint": True},
}


def tool_response(ident, value, *, error=False):
    return {"jsonrpc": "2.0", "id": ident, "result": {
        "content": [{"type": "text", "text": json.dumps(value, sort_keys=True)}],
        "isError": error,
    }}


def augment_catalog(reply):
    """Only modify the authenticated, filtered HTTP catalogue."""
    result = reply.get("result")
    if not isinstance(result, dict) or not isinstance(result.get("tools"), list):
        return reply
    tools = result["tools"]
    for tool in tools:
        if not isinstance(tool, dict) or tool.get("name") != "create_container":
            continue
        schema = tool.get("inputSchema")
        if isinstance(schema, dict) and isinstance(schema.get("properties"), dict):
            schema["properties"]["request_key"] = {
                "type": "string", "pattern": "^[0-9a-f]{32}$",
                "description": ("Optional stable 32-hex creation intent identifier. "
                                "The HTTP gateway also deduplicates the exact username "
                                "and original parameters if ChatGPT sends a new key.")
            }
            schema["properties"]["status_only"] = {
                "type": "boolean",
                "description": ("Set true with username only to read the prior "
                                "durable creation receipt without dispatch. "
                                "Useful when boxweave_create_status is not "
                                "available in a client's capped tool catalog.")
            }
            tool["description"] = (
                tool.get("description", "") +
                " HTTP MCP: create is asynchronous and durable. The call returns "
                "a task_id immediately; poll boxweave_create_status or "
                "call create_container with username and status_only=true "
                "to READ the original receipt without dispatch. Resending "
                "a matching request returns the same receipt, never a second "
                "create. A different request for the same name is refused. "
                "A succeeded receipt only acknowledges upstream creation; "
                "the guest account and SSH may still be provisioning. "
                "An UNKNOWN receipt must be inspected, never replayed."
            )
    if not any(isinstance(t, dict) and t.get("name") == "boxweave_create_status"
               for t in tools):
        tools.append(dict(CREATE_STATUS_TOOL))
    return reply


def public_origin(value):
    if not isinstance(value, str) or len(value) > 255:
        raise ValueError("REMOTE_PUBLIC_URL_INVALID")
    parsed = urlsplit(value)
    if (parsed.scheme != "https" or not parsed.hostname
            or not parsed.netloc or parsed.username or parsed.password
            or parsed.path not in ("", "/") or parsed.query or parsed.fragment):
        raise ValueError("REMOTE_PUBLIC_URL_INVALID")
    return value.rstrip("/")


def relay(message, gate=GATE, timeout=45):
    """Bounded, isolated stdio invocation: one request, no cross-client state.

    The existing gateway owns upstream credentials, tool transforms and the
    optional operator allowlist. Never interpret a caller's command or launcher.
    A timed-out write is UNKNOWN; this layer never retries it.
    """
    if not Path(gate).is_file():
        raise RemoteError("gateway_unavailable", 503)
    init = {"jsonrpc": "2.0", "id": "bw-remote-init",
            "method": "initialize", "params": {
                "protocolVersion": PROTOCOL,
                "capabilities": {}, "clientInfo": {"name": "boxweave-remote", "version": "1"}}}
    ready = {"jsonrpc": "2.0", "method": "notifications/initialized"}
    payload = "".join(json.dumps(m, separators=(",", ":")) + "\n"
                      for m in (init, ready, message))
    if len(payload) > 65536:
        raise RemoteError("request_too_large", 413)
    try:
        # -E -s ignore the environment and user site without -P: the gateway
        # imports its sibling `nesting` module, and isolated mode (-I) strips
        # the script directory from sys.path, so every relay would fail.
        completed = subprocess.run([sys.executable, "-E", "-s", gate], input=payload,
                                   text=True, stdout=subprocess.PIPE,
                                   stderr=subprocess.DEVNULL, timeout=timeout, check=False)
    except (OSError, subprocess.TimeoutExpired):
        raise RemoteError("upstream_outcome_unknown", 503) from None
    if len(completed.stdout) > MAX_RESULT:
        raise RemoteError("upstream_response_too_large", 502)
    if completed.returncode:
        raise RemoteError("upstream_unavailable", 503)
    for line in reversed(completed.stdout.splitlines()):
        try:
            item = json.loads(line)
        except (ValueError, TypeError):
            continue
        if (isinstance(item, dict) and item.get("id") == message.get("id")
                and ("result" in item or "error" in item)):
            return item
    raise RemoteError("upstream_protocol_error", 502)


def create_remote_app(store, config, dispatch=None, clock=None):
    """A loopback-only transport; a public origin requires explicit HTTPS config.

    No public origin means a local OAuth resource bound to loopback. The
    network listener never binds a publicly reachable interface.
    """
    config = validate_remote_config(config)
    origin = (public_origin(config["public_url"]) if config.get("public_url")
              else "http://127.0.0.1:" + str(config["port"]))
    resource = origin + "/mcp"
    if clock is None:
        import time
        clock = time.time
    credentials = RemoteCredentials(store, resource, clock)
    bridge = dispatch or relay
    # A background worker may wait longer than the HTTP client. Provider
    # timeouts still mean UNKNOWN and never trigger an automatic replay.
    background_bridge = dispatch if dispatch is not None else (
        lambda message: relay(message, timeout=900))
    journal = CreateJournal(store, clock)
    journal.recover_interrupted()
    create_slots = threading.BoundedSemaphore(4)
    app = Flask(__name__, static_folder=None)
    app.config["MAX_CONTENT_LENGTH"] = 65536
    origins = frozenset([origin] + list(config.get("allowed_origins", [])))
    if any(not valid_url(u) for u in origins):
        raise ValueError("REMOTE_ORIGIN_INVALID")

    def out(value, status=200, headers=None):
        response = jsonify(value)
        response.status_code = status
        response.headers["Cache-Control"] = "no-store"
        response.headers["Referrer-Policy"] = "no-referrer"
        response.headers["X-Content-Type-Options"] = "nosniff"
        if headers:
            response.headers.update(headers)
        return response

    @app.after_request
    def security_headers(response):
        # A browser form POST to /authorize returns a cross-origin OAuth
        # callback redirect. Chrome applies the form page's form-action CSP
        # to that redirect too. Restrict the extra destination to the exact
        # registered client's callback origin, and only on a valid auth page.
        form_action = "form-action 'self'"
        referrer_policy = "no-referrer"
        if (request.method == "GET" and request.path == "/authorize"
                and response.status_code == 200):
            callback = request.args.get("redirect_uri")
            client = request.args.get("client_id")
            if valid_url(callback) and credentials.registered(client, callback):
                parsed = urlsplit(callback)
                callback_origin = parsed.scheme + "://" + parsed.netloc
                # CSP source syntax is deliberately narrower than OAuth's
                # redirect URI parser; never interpolate unsafe header bytes.
                if re.fullmatch(r"https?://[A-Za-z0-9.\-:\[\]]+", callback_origin):
                    form_action += " " + callback_origin
                    # no-referrer can turn a browser's POST Origin into null.
                    # same-origin keeps the OAuth callback's referrer private.
                    referrer_policy = "same-origin"
        response.headers["Cache-Control"] = "no-store"
        response.headers["Referrer-Policy"] = referrer_policy
        response.headers["X-Content-Type-Options"] = "nosniff"
        response.headers["Content-Security-Policy"] = (
            "default-src 'none'; style-src 'unsafe-inline'; "
            + form_action + "; frame-ancestors 'none'")
        return response

    @app.errorhandler(RemoteError)
    def remote_error(error):
        headers = {}
        if error.status == 401:
            headers["WWW-Authenticate"] = (
                'Bearer resource_metadata="' + origin +
                '/.well-known/oauth-protected-resource/mcp"')
        return out({"error": error.code}, error.status, headers)

    @app.errorhandler(413)
    def too_large(_):
        return out({"error": "request_too_large"}, 413)

    @app.before_request
    def protect_origin():
        # Prevent DNS rebinding and Host spoofing even on the loopback ingress.
        # A trusted proxy may preserve the public Host or use the fixed
        # loopback upstream Host; forwarded-host headers are never trusted.
        incoming_host = request.host.lower()
        allowed_hosts = {urlsplit(origin).netloc.lower(),
                         "localhost:" + str(config["port"]),
                         "127.0.0.1:" + str(config["port"])}
        if incoming_host not in allowed_hosts and incoming_host != "localhost":
            # Flask test clients use localhost; production only uses explicit
            # canonical or loopback authority.
            raise RemoteError("invalid_host", 403)
        supplied = request.headers.get("Origin")
        if supplied and supplied not in origins:
            raise RemoteError("invalid_origin", 403)
        if (request.method == "POST" and request.path in ("/authorize",)
                and request.headers.get("Origin") != origin):
            raise RemoteError("invalid_origin", 403)

    @app.get("/")
    def root_help():
        # Public information only: never disclose operator data or tokens.
        return Response(
            "<!doctype html><title>BoxWeave HTTP MCP</title>"
            "<h1>BoxWeave HTTP MCP</h1><p>Endpoint: /mcp</p>"
            "<p>OAuth metadata: /.well-known/oauth-protected-resource/mcp</p>"
            "<p>Authentication is required for MCP tool access.</p>",
            mimetype="text/html")

    @app.get("/.well-known/oauth-protected-resource")
    @app.get("/.well-known/oauth-protected-resource/mcp")
    def protected_resource():
        return out({"resource": resource,
                    "authorization_servers": [origin],
                    "scopes_supported": ["mcp:operator"],
                    "bearer_methods_supported": ["header"]})

    @app.get("/.well-known/oauth-authorization-server")
    def authorization_server():
        return out({"issuer": origin,
                    "authorization_endpoint": origin + "/authorize",
                    "token_endpoint": origin + "/token",
                    "registration_endpoint": origin + "/register",
                    "response_types_supported": ["code"],
                    "grant_types_supported": ["authorization_code", "refresh_token"],
                    "code_challenge_methods_supported": ["S256"],
                    "token_endpoint_auth_methods_supported": ["none"],
                    "scopes_supported": ["mcp:operator"]})

    @app.post("/register")
    def register():
        data = request.get_json(silent=True)
        if not isinstance(data, dict) or data.get("token_endpoint_auth_method", "none") != "none":
            raise RemoteError("invalid_client_metadata")
        client = credentials.register(data.get("client_name", "MCP client"),
                                      data.get("redirect_uris"))
        return out({"client_id": client, "client_name": data.get("client_name", "MCP client"),
                    "redirect_uris": data["redirect_uris"],
                    "token_endpoint_auth_method": "none"}, 201)

    @app.get("/authorize")
    def authorize():
        args = request.args
        if (args.get("response_type") != "code" or
                args.get("code_challenge_method") != "S256" or
                args.get("scope", "mcp:operator") != "mcp:operator"):
            raise RemoteError("unsupported_response_type")
        ident = credentials.pending(args.get("client_id"), args.get("redirect_uri"),
                                    args.get("code_challenge"), args.get("resource"),
                                    args.get("state"))
        page = """<!doctype html><html lang="en"><meta charset="utf-8">
<title>BoxWeave MCP authorization</title><body>
<h1>Authorize MCP client</h1>
<p>This client will receive the current trusted operator's broad
BoxWeave MCP capability, including mutating tools. Only approve clients you trust.</p>
<p>Client: <strong>%s</strong></p>
<form action="/authorize" method="post">
<input type="hidden" name="pending" value="%s">
<label>BoxWeave administrator password
<input name="password" type="password" required autocomplete="current-password"></label>
<label><input name="approve" type="checkbox" value="yes" required>
I authorize this client to use BoxWeave MCP tools.</label>
<button type="submit">Authorize</button>
</form></body></html>""" % (html.escape(args["client_id"]), html.escape(ident))
        return Response(page, mimetype="text/html",
                        headers={"Cache-Control": "no-store", "Referrer-Policy": "no-referrer"})

    @app.post("/authorize")
    def approve():
        if request.content_length is not None and request.content_length > 4096:
            raise RemoteError("invalid_request")
        if request.form.get("approve") != "yes":
            raise RemoteError("access_denied", 403)
        code, pending = credentials.approve(request.form.get("pending"),
                                            request.form.get("password"))
        return redirect(pending["redirect"] + (
            "&" if "?" in pending["redirect"] else "?") +
            urlencode({"code": code, "state": pending["state"]}), code=302)

    @app.post("/token")
    def token():
        if request.content_length is not None and request.content_length > 8192:
            raise RemoteError("invalid_request")
        data = request.form
        client, target = data.get("client_id"), data.get("resource")
        kind = data.get("grant_type")
        if kind == "authorization_code":
            values = credentials.exchange_code(
                data.get("code"), client, data.get("redirect_uri"),
                data.get("code_verifier"), target)
        elif kind == "refresh_token":
            values = credentials.refresh(data.get("refresh_token"), client, target)
        else:
            raise RemoteError("unsupported_grant_type")
        return out(values)

    def authenticated(link=None):
        if link is not None:
            if credentials.verify(link, ("link",)):
                return True
            raise RemoteError("invalid_token", 401)
        header = request.headers.get("Authorization", "")
        if not header.startswith("Bearer ") or header.count(" ") != 1:
            raise RemoteError("authentication_required", 401)
        if credentials.verify(header[7:], ("api", "access"), resource):
            return True
        raise RemoteError("invalid_token", 401)

    def mcp_request(link=None):
        authenticated(link)
        if request.method == "GET":
            return out({"error": "stream_not_supported"}, 405,
                       {"Allow": "POST, DELETE"})
        if request.method == "DELETE":
            return out({"error": "stateless_server"}, 405,
                       {"Allow": "POST"})
        if request.method != "POST":
            return out({"error": "method_not_allowed"}, 405, {"Allow": "POST"})
        if request.mimetype != "application/json":
            raise RemoteError("unsupported_media_type", 415)
        accepted = request.headers.get("Accept", "")
        if ("application/json" not in accepted or
                "text/event-stream" not in accepted):
            raise RemoteError("not_acceptable", 406)
        version = request.headers.get("MCP-Protocol-Version")
        if version is not None and version != PROTOCOL:
            raise RemoteError("unsupported_protocol_version", 400)
        message = request.get_json(silent=True)
        if (not isinstance(message, dict) or message.get("jsonrpc") != "2.0"
                or not isinstance(message.get("method"), str)
                or not isinstance(message.get("params", {}), dict)
                or "result" in message or "error" in message):
            raise RemoteError("invalid_json_rpc")
        method = message["method"]
        # These RPC methods require a params object. Report a typed client
        # error rather than leaking a KeyError as an HTTP 500.
        if method in ("initialize", "tools/call") and "params" not in message:
            raise RemoteError("invalid_json_rpc")
        if method == "initialize":
            if "id" not in message:
                raise RemoteError("invalid_json_rpc")
            params = message["params"]
            if params.get("protocolVersion") != PROTOCOL:
                raise RemoteError("unsupported_protocol_version", 400)
            return out({"jsonrpc": "2.0", "id": message["id"],
                        "result": {"protocolVersion": PROTOCOL,
                                   "capabilities": {"tools": {"listChanged": False},
                                                    "resources": {"listChanged": False}},
                                   "serverInfo": {"name": "boxweave", "version": SERVER_INFO_VERSION,
                                                  "description": SERVER_DESCRIPTION},
                                   "instructions": SERVER_INSTRUCTIONS}},
                       headers={"MCP-Protocol-Version": PROTOCOL})
        if method == "notifications/initialized":
            return Response(status=202, headers={"Cache-Control": "no-store"})
        if method not in METHODS:
            return out({"jsonrpc": "2.0", "id": message.get("id"),
                        "error": {"code": -32601, "message": "Method not found"}})
        if "id" not in message or isinstance(message["id"], (dict, list, bool)):
            raise RemoteError("invalid_json_rpc")
        if method == "tools/call":
            params = message["params"]
            if params.get("name") == "boxweave_create_status":
                reply = handle_create_status(message)
                return out(reply, headers={"MCP-Protocol-Version": PROTOCOL})
            if params.get("name") == "create_container":
                reply = handle_create(message)
                return out(reply, headers={"MCP-Protocol-Version": PROTOCOL})
        reply = bridge(message)
        if not isinstance(reply, dict) or reply.get("id") != message["id"]:
            raise RemoteError("upstream_protocol_error", 502)
        if method == "tools/list" and not message.get("params", {}).get("cursor"):
            reply = augment_catalog(reply)
        return out(reply, headers={"MCP-Protocol-Version": PROTOCOL})

    def run_create(task_id, message):
        try:
            reply = background_bridge(message)
            result = reply.get("result") if isinstance(reply, dict) else None
            ok = (isinstance(result, dict) and result.get("isError") is not True
                  and "error" not in reply and isinstance(result.get("content"), list))
            if ok:
                # An opt-in unified REST Gateway can acknowledge asynchronous
                # CREATING. Do not promote that to succeeded or guest-ready.
                content = result.get("content", [])
                accepted = None
                if len(content) == 1 and isinstance(content[0], dict) and isinstance(
                        content[0].get("text"), str):
                    try:
                        accepted = json.loads(content[0]["text"])
                    except (ValueError, TypeError):
                        pass
                if (isinstance(accepted, dict)
                        and accepted.get("task_id") == task_id
                        and accepted.get("state") in (
                            "submitted", "accepted", "unknown", "succeeded")):
                    # The single app journal already owns this receipt.
                    # A duplicate/pending status is NEVER proof of success.
                    if accepted["state"] == "accepted":
                        journal.accept(task_id)
                else:
                    journal.complete(task_id, True)
            else:
                journal.complete(task_id, False)
        except Exception:
            # The original write may still finish upstream. Never retry it.
            try:
                journal.complete(task_id, False)
            except Exception:
                pass  # Persistent submitted state is UNKNOWN after restart.
        finally:
            create_slots.release()

    def handle_create(message):
        ident = message["id"]
        args = message["params"].get("arguments")
        try:
            # Some ChatGPT connectors cap the number of exposed MCP tools,
            # so the read-only status tool may be absent even after refresh.
            # Existing create_container can retrieve its own original receipt
            # without performing or authorizing a second write.
            if isinstance(args, dict) and args.get("status_only") is True:
                if set(args) != {"username", "status_only"}:
                    raise CreateFault("CREATE_SELECTOR_INVALID")
                return tool_response(ident, journal.get(username=args["username"]))
            if isinstance(args, dict) and "status_only" in args:
                raise CreateFault("CREATE_ARGUMENTS_INVALID")
            logical, key, payload, digest = validated_request(args)
            # Use the *local* immutable REST schema and the same root-owned
            # allowlist as the stdio Gateway. Never consult an upstream MCP
            # tools/list endpoint for migrated create authorization.
            from . import mcp_gate, rest_routes
            allowed = mcp_gate.tool_policy()
            if allowed is not None and "create_container" not in allowed:
                raise CreateFault("GATE_TOOL_NOT_ALLOWED")
            schema = next(t["inputSchema"] for t in
                          rest_routes.mcp_tool_catalog()
                          if t["name"] == "create_container")
            validate_upstream_payload(payload, schema)
            try:
                journal.get(username=logical)
            except CreateFault as error:
                if error.code != "CREATE_TASK_NOT_FOUND":
                    raise
                existing = False
            else:
                existing = True
            if existing:
                receipt, _ = journal.reserve(logical, key, digest)
                return tool_response(ident, receipt)
            if not create_slots.acquire(blocking=False):
                raise CreateFault("CREATE_CAPACITY_BUSY")
            try:
                receipt, fresh = journal.reserve(logical, key, digest)
                if not fresh:
                    create_slots.release()
                    return tool_response(ident, receipt)
                forwarded = {"jsonrpc": "2.0", "id": "bw-create-" +
                             receipt["task_id"], "method": "tools/call",
                             "params": {"name": "create_container",
                                        "arguments": payload}}
                thread = threading.Thread(target=run_create,
                                          args=(receipt["task_id"], forwarded),
                                          name="bw-http-create", daemon=True)
                try:
                    thread.start()
                except RuntimeError:
                    # Dispatch intent was already committed. Fail safely.
                    journal.complete(receipt["task_id"], False)
                    create_slots.release()
                    receipt = journal.get(task_id=receipt["task_id"])
                return tool_response(ident, receipt)
            except BaseException:
                # A committed intent cannot be safely replayed if the process
                # stopped between reservation and thread startup.
                create_slots.release()
                raise
        except CreateFault as error:
            return tool_response(ident, {"code": error.code}, error=True)

    def handle_create_status(message):
        args = message["params"].get("arguments", {})
        try:
            if not isinstance(args, dict) or set(args) - {
                    "task_id", "request_key", "username"}:
                raise CreateFault("CREATE_SELECTOR_INVALID")
            return tool_response(message["id"], journal.get(**args))
        except CreateFault as error:
            return tool_response(message["id"], {"code": error.code}, error=True)

    @app.route("/mcp", methods=["GET", "POST", "DELETE"])
    def mcp():
        return mcp_request()

    @app.route("/mcp/link/<link>", methods=["GET", "POST", "DELETE"])
    def mcp_link(link):
        if TOKEN_PATTERN.fullmatch(link) is None:
            raise RemoteError("invalid_token", 401)
        return mcp_request(link)

    return app


def validate_remote_config(config):
    """Default to authenticated loopback MCP, never to a public listener."""
    if config is None:
        config = {}
    if not isinstance(config, dict) or set(config) - {
            "enabled", "public_url", "port", "allowed_origins"}:
        raise ValueError("REMOTE_CONFIG_INVALID")
    if type(config.get("enabled", True)) is not bool:
        raise ValueError("REMOTE_CONFIG_INVALID")
    if config.get("enabled", True) is not True:
        raise ValueError("REMOTE_NOT_ENABLED")
    port = config.get("port", 18844)
    if type(port) is not int or not 1024 <= port <= 65535:
        raise ValueError("REMOTE_PORT_INVALID")
    external = config.get("public_url")
    if external is not None:
        public_origin(external)
    origins = config.get("allowed_origins", [])
    if not isinstance(origins, list) or any(
            not isinstance(u, str) or not valid_url(u) or
            u != ("https://" + urlsplit(u).netloc)
            for u in origins):
        raise ValueError("REMOTE_ORIGIN_INVALID")
    return {"enabled": True, "port": port, "allowed_origins": origins,
            **({"public_url": external} if external is not None else {})}
