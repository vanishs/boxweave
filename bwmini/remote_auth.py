"""Private single-operator credentials for the optional remote MCP transport.

This is an independent trust boundary. A Web cookie, remote token, URL
capability, OAuth code and upstream Provider JWT are never interchangeable.
"""
import base64
import hashlib
import hmac
import re
import secrets
import time
from urllib.parse import urlsplit

from werkzeug.security import check_password_hash

SCHEMA = """
CREATE TABLE IF NOT EXISTS remote_clients(
 id TEXT PRIMARY KEY, label TEXT NOT NULL, redirects TEXT NOT NULL, created REAL NOT NULL
);
CREATE TABLE IF NOT EXISTS remote_pending(
 id TEXT PRIMARY KEY, client TEXT NOT NULL, redirect TEXT NOT NULL,
 challenge TEXT NOT NULL, resource TEXT NOT NULL, state TEXT NOT NULL,
 expires REAL NOT NULL
);
CREATE TABLE IF NOT EXISTS remote_codes(
 digest TEXT PRIMARY KEY, client TEXT NOT NULL, redirect TEXT NOT NULL,
 challenge TEXT NOT NULL, resource TEXT NOT NULL, expires REAL NOT NULL
);
CREATE TABLE IF NOT EXISTS remote_credentials(
 digest TEXT PRIMARY KEY, kind TEXT NOT NULL, label TEXT NOT NULL,
 client TEXT, resource TEXT, created REAL NOT NULL, expires REAL,
 revoked INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS remote_attempts(
 at REAL NOT NULL, category TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS remote_attempt_time ON remote_attempts(at);
"""

TOKEN_PATTERN = re.compile(r"^[A-Za-z0-9_-]{40,160}$")
CLIENT_PATTERN = re.compile(r"^[A-Za-z0-9_-]{10,160}$")
CHALLENGE_PATTERN = re.compile(r"^[A-Za-z0-9_-]{43,128}$")


class RemoteError(Exception):
    def __init__(self, code="invalid_request", status=400):
        self.code, self.status = code, status
        super().__init__(code)


def digest(value):
    return hashlib.sha256(value.encode("ascii")).hexdigest()


def secret():
    return secrets.token_urlsafe(48)


def valid_url(value):
    """Only registered HTTPS or strict loopback HTTP redirect URIs."""
    if not isinstance(value, str) or not 1 <= len(value) <= 1024:
        return False
    try:
        u = urlsplit(value)
        if u.username or u.password or u.fragment or not u.netloc:
            return False
        if u.scheme == "https":
            return bool(u.hostname and "." in u.hostname)
        return (u.scheme == "http" and u.hostname in
                ("localhost", "127.0.0.1", "[::1]", "::1"))
    except ValueError:
        return False


class RemoteCredentials:
    def __init__(self, store, resource, clock=time.time):
        self.store, self.resource, self.clock = store, resource, clock

    def _rate(self, db, category, max_attempts=10):
        now = self.clock()
        db.execute("DELETE FROM remote_attempts WHERE at<?", (now - 60,))
        if db.execute("SELECT count(*) FROM remote_attempts WHERE category=?",
                      (category,)).fetchone()[0] >= max_attempts:
            raise RemoteError("rate_limited", 429)
        db.execute("INSERT INTO remote_attempts(at,category) VALUES(?,?)", (now, category))

    def issue(self, kind, label, client=None, resource=None, expires=None):
        if kind not in ("api", "link", "access", "refresh"):
            raise RemoteError()
        if not isinstance(label, str) or not 1 <= len(label) <= 80:
            raise RemoteError("invalid_label")
        value = secret()
        with self.store.db() as db:
            db.execute("INSERT INTO remote_credentials "
                       "(digest,kind,label,client,resource,created,expires) "
                       "VALUES(?,?,?,?,?,?,?)",
                       (digest(value), kind, label, client, resource,
                        self.clock(), expires))
        return value

    def verify(self, value, kinds, resource=None):
        if not isinstance(value, str) or TOKEN_PATTERN.fullmatch(value) is None:
            return None
        with self.store.db() as db:
            row = db.execute("SELECT kind,client,resource,expires,revoked "
                             "FROM remote_credentials WHERE digest=?",
                             (digest(value),)).fetchone()
        if row is None or row["revoked"] or row["kind"] not in kinds:
            return None
        if row["expires"] is not None and row["expires"] <= self.clock():
            return None
        if row["kind"] == "access" and (resource != row["resource"]):
            return None
        return dict(row)

    def list(self):
        with self.store.db() as db:
            return [dict(r) for r in db.execute(
                "SELECT substr(digest,1,12) AS id,kind,label,client,created,"
                "expires,revoked FROM remote_credentials "
                "ORDER BY created DESC LIMIT 200")]

    def revoke(self, prefix):
        if not isinstance(prefix, str) or not re.fullmatch(r"[a-f0-9]{12}", prefix):
            raise RemoteError()
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            rows = db.execute("SELECT digest FROM remote_credentials "
                              "WHERE substr(digest,1,12)=?", (prefix,)).fetchall()
            if len(rows) != 1:
                raise RemoteError("not_found", 404)
            db.execute("UPDATE remote_credentials SET revoked=1 WHERE digest=?",
                       (rows[0][0],))

    def register(self, label, redirects):
        if not isinstance(label, str) or not 1 <= len(label) <= 80:
            raise RemoteError("invalid_client_metadata")
        if (not isinstance(redirects, list) or not 1 <= len(redirects) <= 8
                or len(set(redirects)) != len(redirects)
                or not all(valid_url(url) for url in redirects)):
            raise RemoteError("invalid_redirect_uri")
        import json
        ident = "bw_" + secrets.token_urlsafe(24)
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            self._rate(db, "register", 20)
            db.execute("INSERT INTO remote_clients VALUES(?,?,?,?)",
                       (ident, label, json.dumps(redirects), self.clock()))
        return ident

    def registered(self, client, redirect):
        import json
        if not isinstance(client, str) or CLIENT_PATTERN.fullmatch(client) is None:
            return False
        with self.store.db() as db:
            row = db.execute("SELECT redirects FROM remote_clients WHERE id=?",
                             (client,)).fetchone()
        return bool(row and redirect in json.loads(row[0]))

    def pending(self, client, redirect, challenge, resource, state):
        if (not self.registered(client, redirect) or
            not isinstance(challenge, str) or not CHALLENGE_PATTERN.fullmatch(challenge)
            or not isinstance(state, str) or not 8 <= len(state) <= 1024
            or resource != self.resource):
            raise RemoteError("invalid_authorization_request")
        ident = secret()
        with self.store.db() as db:
            db.execute("DELETE FROM remote_pending WHERE expires<?", (self.clock(),))
            db.execute("INSERT INTO remote_pending VALUES(?,?,?,?,?,?,?)",
                       (digest(ident), client, redirect, challenge, resource,
                        state, self.clock() + 300))
        return ident

    def approve(self, pending, password):
        if not isinstance(pending, str) or TOKEN_PATTERN.fullmatch(pending) is None:
            raise RemoteError("invalid_request")
        # Record the attempt in its own committed transaction. The pending
        # lookup/password check below can raise RemoteError, and Store.db()
        # rolls back on exceptions. Keeping _rate in that same transaction
        # would silently erase each failed attempt and disable throttling.
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            self._rate(db, "admin_authorize", 5)
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute("SELECT * FROM remote_pending WHERE id=? AND expires>?",
                             (digest(pending), self.clock())).fetchone()
            admin = db.execute("SELECT password FROM admin WHERE id=1").fetchone()
            if (row is None or admin is None or not isinstance(password, str)
                    or len(password) > 1024
                    or not check_password_hash(admin[0], password)):
                raise RemoteError("access_denied", 403)
            db.execute("DELETE FROM remote_pending WHERE id=?", (digest(pending),))
            code = secret()
            db.execute("INSERT INTO remote_codes VALUES(?,?,?,?,?,?)",
                       (digest(code), row["client"], row["redirect"],
                        row["challenge"], row["resource"], self.clock() + 120))
            return code, dict(row)

    def exchange_code(self, code, client, redirect, verifier, resource):
        if (not isinstance(code, str) or TOKEN_PATTERN.fullmatch(code) is None
                or not isinstance(verifier, str)
                or not 43 <= len(verifier) <= 128
                or not re.fullmatch(r"[A-Za-z0-9._~-]+", verifier)):
            raise RemoteError("invalid_grant")
        challenge = base64.urlsafe_b64encode(
            hashlib.sha256(verifier.encode("ascii")).digest()
        ).rstrip(b"=").decode("ascii")
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute("SELECT * FROM remote_codes WHERE digest=?",
                             (digest(code),)).fetchone()
            if (row is None or row["expires"] <= self.clock()
                or not hmac.compare_digest(row["challenge"], challenge)
                or row["client"] != client or row["redirect"] != redirect
                or row["resource"] != resource):
                raise RemoteError("invalid_grant")
            # The one-use authorization code and BOTH credentials must
            # commit together: partial issue must not burn the code or leave
            # an orphan access token that the client never received.
            db.execute("DELETE FROM remote_codes WHERE digest=?", (digest(code),))
            return self._issue_pair(db, client, resource)

    def _issue_pair(self, db, client, resource):
        if resource != self.resource:
            raise RemoteError("invalid_target")
        # One BEGIN IMMEDIATE transaction owns both inserts. A crash,
        # random-token failure, or SQLite constraint failure rolls them back.
        now = self.clock()
        access = secret()
        db.execute("INSERT INTO remote_credentials "
                   "(digest,kind,label,client,resource,created,expires) "
                   "VALUES(?,?,?,?,?,?,?)",
                   (digest(access), "access", "OAuth " + client[:40],
                    client, resource, now, now + 3600))
        refresh = secret()
        db.execute("INSERT INTO remote_credentials "
                   "(digest,kind,label,client,resource,created,expires) "
                   "VALUES(?,?,?,?,?,?,?)",
                   (digest(refresh), "refresh", "Refresh " + client[:40],
                    client, resource, now, now + 30 * 86400))
        return {"access_token": access, "token_type": "Bearer",
                "expires_in": 3600, "refresh_token": refresh,
                "scope": "mcp:operator"}

    def issue_pair(self, client, resource):
        if resource != self.resource:
            raise RemoteError("invalid_target")
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            return self._issue_pair(db, client, resource)

    def refresh(self, value, client, resource):
        if not isinstance(value, str) or TOKEN_PATTERN.fullmatch(value) is None:
            raise RemoteError("invalid_grant")
        with self.store.db() as db:
            db.execute("BEGIN IMMEDIATE")
            row = db.execute("SELECT * FROM remote_credentials WHERE digest=?",
                             (digest(value),)).fetchone()
            if (row is None or row["kind"] != "refresh" or row["revoked"]
                    or row["expires"] <= self.clock() or row["client"] != client
                    or row["resource"] != resource):
                raise RemoteError("invalid_grant")
            # Rotating an existing token is one atomic operation: do not
            # revoke it if issuing the new pair cannot be committed.
            db.execute("UPDATE remote_credentials SET revoked=1 WHERE digest=?",
                       (digest(value),))
            return self._issue_pair(db, client, resource)
