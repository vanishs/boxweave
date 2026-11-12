#!/usr/bin/env python3
"""Disposable isolated-network HTTPS witness; never a public WAN endpoint."""
import http.server
import os
import ssl
import sys

if len(sys.argv) != 4:
    raise SystemExit("expected one-use cert/key/log")
certificate, private_key, hitlog = sys.argv[1:]
class Witness(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path != "/s2-probe":
            self.send_error(404)
            return
        with open(hitlog, "a", encoding="ascii") as hits:
            hits.write("S2_SIMULATED_WAN_HTTPS_GET\n")
            hits.flush()
            os.fsync(hits.fileno())
        body = b"S2_SIMULATED_WAN_HTTPS_OK\n"
        self.send_response(200)
        self.send_header("Content-Type", "text/plain")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(body)
    def log_message(self, fmt, *args):
        pass

server = http.server.ThreadingHTTPServer(("198.18.0.2", 18443), Witness)
context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
context.load_cert_chain(certificate, private_key)
server.socket = context.wrap_socket(server.socket, server_side=True)
server.serve_forever()
