#!/usr/bin/env python3
"""Single-use LAB-only TLS CONNECT relay for audited upstream package hosts.

No public bind, no arbitrary CONNECT, no request-body/header forwarding
visible to the host, and no production/MCP runtime capability. It is used
only when the CI guest's direct NAT is independently shown to be unavailable.
Use of this relay does NOT satisfy BH-009 direct routed guest WAN acceptance.
"""
import ipaddress
import select
import socket
import socketserver
import sys
from http.server import BaseHTTPRequestHandler

ALLOWED_HOSTS = frozenset({
    "archive.ubuntu.com",
    "security.ubuntu.com",
    "pkgs.zabbly.com",
    "images.linuxcontainers.org",
    "cloud-images.ubuntu.com",
})
ALLOWED_PORT = 443

class PackageOnlyConnect(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"
    timeout = 12

    def log_message(self, fmt, *args):
        # No URLs, credentials, HTTP headers or arbitrary request text in CI logs.
        pass

    def send_refused(self):
        self.send_response(403)
        self.send_header("Content-Length", "0")
        self.send_header("Connection", "close")
        self.end_headers()

    def do_CONNECT(self):
        target = self.path
        host, sep, port = target.rpartition(":")
        if not sep or not port.isascii() or not port.isdecimal() or not (
            host in ALLOWED_HOSTS and int(port) == ALLOWED_PORT
        ):
            self.send_refused()
            return
        try:
            destinations = socket.getaddrinfo(host, ALLOWED_PORT, 0, socket.SOCK_STREAM)
        except OSError:
            self.send_refused()
            return
        # Never let DNS repoint an allowlisted name to the L0 host, a Box,
        # a metadata service, or any non-global internal network.
        candidates = [
            (family, socktype, proto, sockaddr)
            for family, socktype, proto, _, sockaddr in destinations
            if family in (socket.AF_INET, socket.AF_INET6)
            and ipaddress.ip_address(sockaddr[0]).is_global
        ]
        candidates.sort(key=lambda x: x[0] != socket.AF_INET)
        remote = None
        for family, socktype, proto, sockaddr in candidates:
            s = socket.socket(family, socktype, proto)
            s.settimeout(8)
            try:
                s.connect(sockaddr)
            except OSError:
                s.close()
                continue
            remote = s
            break
        if remote is None:
            self.send_refused()
            return
        try:
            self.send_response(200, "Connection Established")
            self.send_header("Connection", "keep-alive")
            self.end_headers()
            self.wfile.flush()
            self.connection.settimeout(None)
            remote.settimeout(None)
            clients = (self.connection, remote)
            while True:
                readable, _, _ = select.select(clients, [], [], 45)
                if not readable:
                    break
                for current in readable:
                    other = remote if current is self.connection else self.connection
                    try:
                        block = current.recv(16384)
                        if not block:
                            return
                        other.sendall(block)
                    except (OSError, ConnectionError):
                        return
        finally:
            remote.close()

    def do_GET(self):
        self.send_refused()

    def do_POST(self):
        self.send_refused()

    def do_HEAD(self):
        self.send_refused()

class SingleUseServer(socketserver.ThreadingTCPServer):
    allow_reuse_address = True
    daemon_threads = True

if __name__ == "__main__":
    if len(sys.argv) != 3:
        raise SystemExit("usage: s25_package_proxy.py <owned-bridge-ip> <port>")
    bind, port_text = sys.argv[1:]
    ip = ipaddress.ip_address(bind)
    if ip.version != 4 or not ip.is_private or not 1024 <= int(port_text) <= 65535:
        raise SystemExit("LAB package proxy must bind only to a private IPv4 bridge")
    with SingleUseServer((bind, int(port_text)), PackageOnlyConnect) as server:
        server.serve_forever()
