"""Pinned Containarium v0.114.2 REST operations. No general proxy."""

import ipaddress
import json
import re
import stat
from pathlib import Path
from urllib.parse import quote, urlsplit
from urllib.request import build_opener, ProxyHandler, HTTPRedirectHandler, Request
from urllib.error import HTTPError, URLError


class ProviderError(Exception):
    def __init__(self, status=502, unknown=False):
        self.status = status
        self.unknown = unknown


def missing_box_response(method, path, error):
    """Normalize only the exact v0.114.2 default-project missing-instance reply.

    A generic 500 must never authorize creation or an automatic retry. Match the
    requested logical name, read a bounded JSON body, and discard all raw text.
    """
    match = re.fullmatch(r"/v1/containers/([a-z][a-z0-9-]{0,31})", path)
    if method != "GET" or error.code != 500 or match is None:
        return False
    try:
        raw = error.read(8193)
        if len(raw) > 8192:
            return False
        data = json.loads(raw)
        expected = (
            "rpc error: code = Unknown desc = failed to get container: "
            "failed to get container: Failed to fetch instance "
            f'"{match.group(1)}-container" in project "default": Instance not found'
        )
        return (
            isinstance(data, dict)
            and data.get("code") == 500
            and data.get("error") == expected
        )
    except (OSError, ValueError, TypeError):
        return False


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


def name(value):
    if (
        not isinstance(value, str)
        or not re.fullmatch(r"[a-z][a-z0-9-]{0,31}", value)
        or value.startswith(("containarium-core-", "core-"))
    ):
        raise ValueError("Invalid or protected box name")
    return value


def limits(data, required=False):
    if not isinstance(data, dict) or set(data) - {"cpu", "memory", "disk"}:
        raise ValueError("Only CPU, memory and disk can be changed")
    if required and set(data) != {"cpu", "memory", "disk"}:
        raise ValueError("CPU, memory and disk required")
    for k, v in data.items():
        pattern = (
            r"(?:[1-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-6])"
            if k == "cpu"
            else r"[1-9][0-9]{0,5}(?:MB|GB|TB)"
        )
        if not isinstance(v, str) or not re.fullmatch(pattern, v):
            raise ValueError("Invalid " + k)
    if not data:
        raise ValueError("Select at least one field")
    return data


def public_box(c):
    result = {
        k: c[k]
        for k in (
            "name",
            "username",
            "state",
            "image",
            "backendId",
            "networkTenant",
            "networkTenantSource",
        )
        if k in c
    }
    for field, allowed in [
        ("resources", ("cpu", "memory", "disk")),
        ("network", ("ipAddress", "ipv6Address", "gateway", "bridge", "sshPort")),
    ]:
        result[field] = {k: v for k, v in c.get(field, {}).items() if k in allowed}
    # Only network attribution keys, never arbitrary labels (which may contain secrets).
    labels = c.get("labels", {})
    result["labels"] = (
        {
            k: v
            for k, v in labels.items()
            if k in ("cloud_org_id", "tenant") and isinstance(v, str) and len(v) <= 200
        }
        if isinstance(labels, dict)
        else {}
    )
    return result


def port(value):
    if type(value) != int or not 1 <= value <= 65535:
        raise ValueError("Invalid port")
    return value


class Provider:
    def __init__(self, cfg):
        u = urlsplit(cfg["api_url"])
        if (
            u.scheme not in ("http", "https")
            or u.hostname not in ("127.0.0.1", "::1", "localhost")
            or u.username
            or u.path not in ("", "/")
            or u.query
            or u.fragment
        ):
            raise ValueError("Provider must be a configured loopback origin")
        self.origin = cfg["api_url"].rstrip("/")
        self.token = Path(cfg["token_file"])
        self.key = cfg["public_key"]
        if not re.fullmatch(
            r"ssh-(?:ed25519|rsa) [A-Za-z0-9+/=]+(?: [^\r\n]*)?", self.key
        ):
            raise ValueError("Explicit managed public key required")
        self.opener = build_opener(ProxyHandler({}), NoRedirect())

    def request(self, method, path, body=None):
        # Paths are constructed only by the operations below, never by HTTP input.
        try:
            st = self.token.lstat()
            if not stat.S_ISREG(st.st_mode) or st.st_mode & 0o077:
                raise OSError("credential permissions")
            token = self.token.read_text().strip()
            req = Request(
                self.origin + path,
                data=json.dumps(body).encode() if body is not None else None,
                method=method,
                headers={
                    "Authorization": "Bearer " + token,
                    "Content-Type": "application/json",
                },
            )
            with self.opener.open(req, timeout=45) as r:
                raw = r.read(4 * 1024 * 1024 + 1)
                if len(raw) > 4 * 1024 * 1024:
                    raise ProviderError()
                return json.loads(raw)
        except HTTPError as e:
            try:
                missing = missing_box_response(method, path, e)
            finally:
                e.close()
            if missing:
                raise ProviderError(404) from None
            raise ProviderError(
                e.code if e.code in (400, 403, 404, 409) else 502,
                method != "GET" and e.code >= 500,
            ) from None
        except (URLError, TimeoutError, OSError, ValueError):
            raise ProviderError(502, method != "GET") from None

    def boxes(self):
        r = self.request("GET", "/v1/containers")
        return {
            "containers": [public_box(c) for c in r.get("containers", [])],
            "unreachableBackends": [
                {"backendId": x.get("backendId", "unknown")}
                for x in r.get("unreachableBackends", [])
            ],
        }

    def box(self, n):
        return public_box(
            self.request("GET", "/v1/containers/" + quote(name(n), safe=""))[
                "container"
            ]
        )

    def network_guard(self):
        return self.request("GET", "/v1/network/guard")

    def tenant_capabilities(self):
        return self.request("GET", "/v1/network-tenants/capabilities")

    def network_policies(self):
        return self.request("GET", "/v1/network-policies")

    def network_policy(self, key):
        try:
            return self.request("GET", "/v1/network-policies/" + quote(key, safe=""))
        except ProviderError as error:
            if error.status == 404:
                return {"policy": None, "revision": "absent"}
            raise

    def patch_tenant_access(self, changes):
        return self.request(
            "POST", "/v1/network-policies/tenant-access", {"changes": changes}
        )

    def routes(self, kind):
        r = self.request(
            "GET", "/v1/network/" + ("routes" if kind == "route" else "passthrough")
        )
        fields = {
            "domain",
            "fullDomain",
            "targetIp",
            "containerIp",
            "targetPort",
            "port",
            "containerName",
            "description",
            "protocol",
            "externalPort",
            "enabled",
            "active",
        }
        return {
            "routes": [
                {k: v for k, v in x.items() if k in fields} for x in r.get("routes", [])
            ]
        }

    def execute(self, action, item):
        if action == "create":
            return self.request(
                "POST",
                "/v1/containers",
                {
                    "username": item["username"],
                    "resources": item["resources"],
                    "image": item["image"],
                    "sshKeys": [self.key],
                    "async": True,
                    "enablePodman": False,
                    **(
                        {"networkTenant": item["networkTenant"]}
                        if item.get("networkTenant")
                        else {}
                    ),
                },
            )
        if action in ("start", "stop", "resize", "delete"):
            n = name(item["username"])
            path = "/v1/containers/" + n
            return self.request(
                (
                    "DELETE"
                    if action == "delete"
                    else "PUT" if action == "resize" else "POST"
                ),
                path + ("" if action == "delete" else "/" + action),
                None if action == "delete" else item.get("resources", {}),
            )
        if action in ("route-add", "passthrough-add"):
            existing = self.routes("route" if action == "route-add" else "passthrough")[
                "routes"
            ]
            if any(
                (
                    (r.get("fullDomain", r.get("domain")) == item.get("domain"))
                    if action == "route-add"
                    else (
                        r.get("externalPort") == item["externalPort"]
                        and r.get("protocol") == "ROUTE_PROTOCOL_" + item["protocol"]
                    )
                )
                for r in existing
            ):
                raise ValueError("Route already exists; refusing replacement")
            c = self.box(item["username"])
            ip = c.get("network", {}).get("ipAddress", "")
            address = ipaddress.ip_address(ip)
            if not address.is_private or address.is_loopback or address.is_link_local:
                raise ValueError("Target must be a private box IP")
            payload = {
                "targetIp": ip,
                "targetPort": item["targetPort"],
                "containerName": c["name"],
                "description": item.get("description", ""),
            }
            if action == "route-add":
                payload["domain"] = item["domain"]
            else:
                payload.update(
                    externalPort=item["externalPort"],
                    protocol="ROUTE_PROTOCOL_" + item["protocol"],
                )
            return self.request(
                "POST",
                "/v1/network/" + ("routes" if action == "route-add" else "passthrough"),
                payload,
            )
        if action == "route-delete":
            return self.request(
                "DELETE", "/v1/network/routes/" + quote(item["domain"], safe="")
            )
        if action == "passthrough-delete":
            return self.request(
                "DELETE",
                "/v1/network/passthrough/"
                + str(item["externalPort"])
                + "?protocol=ROUTE_PROTOCOL_"
                + item["protocol"],
            )
        raise ValueError("Unsupported operation")
