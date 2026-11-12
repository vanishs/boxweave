#!/usr/bin/env python3
"""Disposable, routed IPv6 HTTPS WAN *simulation*, never public IPv6 proof.

Creates only a unique network namespace and veth pair in a reviewed single-use
LAB. Distinct from the Incus guest bridge. HTTPS requires an explicit trusted
test certificate; no -k or TLS bypass. Never change host forwarding, global
Incus ACL, existing network, or an unrelated namespace.
"""
import argparse
import http.server
import ipaddress
import json
import os
from pathlib import Path
import signal
import socket
import ssl
import subprocess
import sys
import time

NS = "s25simwan6"
HOST_IF = "s25s6host"
PEER_IF = "s25s6edge"
HOST_IP = "2001:db8:25:2026::1"
SERVER_IP = "2001:db8:25:2026::2"
PORT = 18479
MARKER = "S25_SIMULATED_IPV6_WAN_HTTPS_OK"

def run(*args):
    return subprocess.run(args, check=True, capture_output=True, text=True).stdout.strip()

def require(condition, message):
    if not condition:
        raise RuntimeError(message)

def clean(work):
    metadata = work / "simwan6.json"
    require(metadata.is_file(), "no owned simulated WAN marker; refuse cleanup")
    state = json.loads(metadata.read_text())
    require(state["namespace"] == NS and state["host_if"] == HOST_IF and
            state["server_ip"] == SERVER_IP and state["workdir"] == str(work),
            "invalid owned simulation marker")
    pid = int(state["pid"])
    cmdline = Path(f"/proc/{pid}/cmdline")
    if cmdline.exists() and "s25_simwan6.py" in cmdline.read_bytes().replace(b"\0", b" ").decode(errors="replace"):
        os.kill(pid, signal.SIGTERM)
    subprocess.run(["ip", "link", "del", HOST_IF], check=True, capture_output=True)
    subprocess.run(["ip", "netns", "del", NS], check=True, capture_output=True)
    metadata.unlink()
    print("S25_SIMWAN6_NAMESPACE_CLEAN=PASS")

def start(work, bridge):
    require(os.geteuid() == 0 and os.environ.get("BW_S25_SIMWAN_LAB") == "1",
            "explicit disposable root-only LAB flag required")
    require(bridge.startswith("incusbr-") and bridge[8:].isdigit(),
            "only a new restricted Incus user bridge permitted")
    require(not (work / "simwan6.json").exists(), "owned prior session exists")
    require(NS not in run("ip", "netns", "list"), "namespace already exists")
    require(subprocess.run(["ip", "link", "show", HOST_IF], capture_output=True).returncode != 0,
            "existing veth, refuse collision")
    bridge_ip = run("incus", "network", "get", bridge, "ipv6.address")
    guest_net = ipaddress.ip_interface(bridge_ip).network
    require(guest_net.version == 6 and guest_net.is_private, "unexpected guest bridge")
    require(not run("ip", "-6", "route", "show", "2001:db8:25:2026::/64"), "WAN route collision")
    work.mkdir(mode=0o700, parents=True, exist_ok=True)
    cert, key = work / "simwan6.crt", work / "simwan6.key"
    require(not cert.exists() and not key.exists(), "pre-existing test certificate")
    run("openssl", "req", "-x509", "-newkey", "ec", "-pkeyopt",
        "ec_paramgen_curve:P-256", "-nodes", "-days", "1",
        "-subj", "/CN=s25simwan6.test", "-addext", f"subjectAltName=IP:{SERVER_IP}",
        "-keyout", str(key), "-out", str(cert))
    key.chmod(0o600)
    commands = [
        ["ip", "netns", "add", NS],
        ["ip", "link", "add", HOST_IF, "type", "veth", "peer", "name", PEER_IF],
        ["ip", "link", "set", PEER_IF, "netns", NS],
        ["ip", "-6", "addr", "add", f"{HOST_IP}/64", "dev", HOST_IF],
        ["ip", "link", "set", HOST_IF, "up"],
        ["ip", "-n", NS, "link", "set", "lo", "up"],
        ["ip", "-n", NS, "-6", "addr", "add", f"{SERVER_IP}/64", "dev", PEER_IF],
        ["ip", "-n", NS, "link", "set", PEER_IF, "up"],
        ["ip", "-n", NS, "-6", "route", "add", str(guest_net), "via", HOST_IP],
    ]
    try:
        for command in commands:
            run(*command)
        # IPv6 Duplicate Address Detection is asynchronous. Wait for the
        # independently namespaced address to be usable, never disable DAD.
        for _ in range(45):
            addr = run("ip", "-n", NS, "-6", "addr", "show", "dev", PEER_IF)
            if SERVER_IP in addr and "tentative" not in addr:
                break
            time.sleep(0.2)
        else:
            raise RuntimeError("upstream IPv6 DAD did not settle")
        log = (work / "server.log").open("ab", buffering=0)
        try:
            process = subprocess.Popen(
                ["ip", "netns", "exec", NS, sys.executable, str(Path(__file__).resolve()),
                 "serve", "--cert", str(cert), "--key", str(key)],
                stdin=subprocess.DEVNULL, stdout=log, stderr=log, start_new_session=True)
        finally:
            log.close()
        state = {"namespace": NS, "host_if": HOST_IF, "server_ip": SERVER_IP,
                 "bridge": bridge, "guest_net": str(guest_net), "workdir": str(work),
                 "pid": process.pid, "url": f"https://[{SERVER_IP}]:{PORT}/witness"}
        (work / "simwan6.json").write_text(json.dumps(state, sort_keys=True)+"\n")
        for _ in range(30):
            try:
                out = run("curl", "-g", "-6", "--noproxy", "*", "--cacert", str(cert),
                          "--connect-timeout", "1", "--max-time", "2", "-fsS", state["url"])
                if out == MARKER:
                    print("S25_SIMWAN6_SEPARATELY_ROUTED_TLS_HOST_CONTROL=PASS")
                    print("S25_SIMWAN6_URL="+state["url"])
                    print("S25_SIMWAN6_CERT="+str(cert))
                    return
            except subprocess.CalledProcessError:
                pass
            time.sleep(0.5)
        raise RuntimeError("TLS response on routed namespace never became ready")
    except BaseException:
        if (work / "simwan6.json").exists():
            clean(work)
        else:
            subprocess.run(["ip", "link", "del", HOST_IF], capture_output=True)
            subprocess.run(["ip", "netns", "del", NS], capture_output=True)
        raise

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path != "/witness":
            self.send_error(404)
            return
        body = MARKER.encode()
        self.send_response(200)
        self.send_header("Content-Type", "text/plain")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)
    def log_message(self, fmt, *args):
        print("SIM_WAN6_REMOTE=%s %s" % (self.client_address[0], fmt % args), flush=True)

def serve(cert, key):
    # Separate Linux network namespace, no Incus guest bridge or mock.
    class V6Server(http.server.HTTPServer):
        address_family = socket.AF_INET6
    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    ctx.minimum_version = ssl.TLSVersion.TLSv1_2
    ctx.load_cert_chain(certfile=str(cert), keyfile=str(key))
    httpd = V6Server((SERVER_IP, PORT), Handler)
    httpd.socket = ctx.wrap_socket(httpd.socket, server_side=True)
    httpd.serve_forever()

def main():
    parser = argparse.ArgumentParser()
    sub = parser.add_subparsers(dest="action", required=True)
    for action in ("start", "stop"):
        p = sub.add_parser(action)
        p.add_argument("--workdir", required=True)
        if action == "start":
            p.add_argument("--bridge", required=True)
    p = sub.add_parser("serve")
    p.add_argument("--cert", required=True)
    p.add_argument("--key", required=True)
    args = parser.parse_args()
    if args.action == "serve":
        serve(args.cert, args.key)
    else:
        require(os.geteuid() == 0 and os.environ.get("BW_S25_SIMWAN_LAB") == "1",
                "disposable-root-only simulation")
        work = Path(args.workdir).resolve()
        if args.action == "start":
            start(work, args.bridge)
        else:
            clean(work)

if __name__ == "__main__":
    main()
