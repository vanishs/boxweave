"""Optional inbound Streamable HTTP MCP with single-operator authentication.

Not a generic Web tools/call route. Requests go through the existing
server-owned mcp_gate forced-command implementation and its trust checks.
"""
import html
import json
import os
from pathlib import Path
import re
import subprocess
import sys
from urllib.parse import urlencode, urlsplit

from flask import Flask, Response, jsonify, redirect, request

from .remote_auth import RemoteCredentials, RemoteError, TOKEN_PATTERN, valid_url

PROTOCOL = "2025-11-25"
GATE = "/usr/local/lib/boxweave/default/mcp_gate.py"
METHODS = frozenset(("tools/list", "tools/call", "resources/list", "resources/read",
                     "prompts/list", "prompts/get", "ping"))
MAX_RESULT = 2_097_152


def public_origin(value):
    if not isinstance(value, str) or len(value) > 255:
        raise ValueError("REMOTE_PUBLIC_URL_INVALID")
    parsed = urlsplit(value)
    if (parsed.scheme != "https" or not parsed.hostname
            or not parsed.netloc or parsed.username or parsed.password
            or parsed.path not in ("", "/") or parsed.query or parsed.fragment):
        raise ValueError("REMOTE_PUBLIC_URL_INVALID")
    return value.rstrip("/")


def relay(message, gate=GATE):
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
                                   stderr=subprocess.DEVNULL, timeout=45, check=False)
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
    """Bind this application to loopback with the dedicated remote-mcp CLI.

    config must contain a canonical HTTPS public_url; a reverse proxy is
    responsible for TLS. No source/config option directly publishes a port.
    """
    origin = public_origin(config["public_url"])
    resource = origin + "/mcp"
    if clock is None:
        import time
        clock = time.time
    credentials = RemoteCredentials(store, resource, clock)
    bridge = dispatch or relay
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
                                   "serverInfo": {"name": "boxweave", "version": "0.24.2"}}},
                       headers={"MCP-Protocol-Version": PROTOCOL})
        if method == "notifications/initialized":
            return Response(status=202, headers={"Cache-Control": "no-store"})
        if method not in METHODS:
            return out({"jsonrpc": "2.0", "id": message.get("id"),
                        "error": {"code": -32601, "message": "Method not found"}})
        if "id" not in message or isinstance(message["id"], (dict, list, bool)):
            raise RemoteError("invalid_json_rpc")
        reply = bridge(message)
        if not isinstance(reply, dict) or reply.get("id") != message["id"]:
            raise RemoteError("upstream_protocol_error", 502)
        return out(reply, headers={"MCP-Protocol-Version": PROTOCOL})

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
    if not isinstance(config, dict) or set(config) - {
            "enabled", "public_url", "port", "allowed_origins"}:
        raise ValueError("REMOTE_CONFIG_INVALID")
    if config.get("enabled") is not True:
        raise ValueError("REMOTE_NOT_ENABLED")
    public_origin(config.get("public_url"))
    if type(config.get("port")) is not int or not 1024 <= config["port"] <= 65535:
        raise ValueError("REMOTE_PORT_INVALID")
    if not isinstance(config.get("allowed_origins", []), list):
        raise ValueError("REMOTE_ORIGIN_INVALID")
    return config
