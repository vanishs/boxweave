"""Small POSIX MCP stdio client for the approved smoke test, not a generic gateway."""
import json
import os
import selectors
import signal
import subprocess
import time


class MCPError(RuntimeError): pass


class Client:
    def __init__(self, launcher, timeout=20):
        self.p = subprocess.Popen([str(launcher)], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                  stderr=subprocess.DEVNULL, start_new_session=True)
        self.selector = selectors.DefaultSelector()
        self.selector.register(self.p.stdout, selectors.EVENT_READ)
        os.set_blocking(self.p.stdin.fileno(), False)
        self.buffer, self.ident = b'', 0
        try:
            self.initial = self.call('initialize', {'protocolVersion':'2024-11-05', 'capabilities':{},
                                    'clientInfo':{'name':'boxweave-nested-smoke','version':'1'}}, timeout)
            if self.initial.get('protocolVersion') != '2024-11-05': raise MCPError('Unsupported protocol version')
            self.send({'jsonrpc':'2.0','method':'notifications/initialized'}, time.monotonic()+timeout)
        except Exception:
            self.close(); raise

    def send(self, obj, deadline):
        data = (json.dumps(obj)+'\n').encode()
        if len(data) > 65536: raise MCPError('Request limit exceeded')
        with selectors.DefaultSelector() as selector:
            selector.register(self.p.stdin, selectors.EVENT_WRITE)
            while data:
                remaining = deadline-time.monotonic()
                if remaining <= 0 or not selector.select(remaining): raise TimeoutError('MCP write timeout')
                try: count = os.write(self.p.stdin.fileno(), data)
                except BlockingIOError: continue
                data = data[count:]

    def call(self, method, params, timeout=30):
        self.ident += 1; ident = self.ident; deadline = time.monotonic()+timeout
        self.send({'jsonrpc':'2.0','id':ident,'method':method,'params':params}, deadline)
        while time.monotonic() < deadline:
            if b'\n' not in self.buffer:
                if not self.selector.select(max(0, deadline-time.monotonic())): break
                chunk = os.read(self.p.stdout.fileno(), 65536)
                if not chunk: raise MCPError('MCP stream closed')
                self.buffer += chunk
                if len(self.buffer) > 2097152: raise MCPError('Response limit exceeded')
                continue
            line, self.buffer = self.buffer.split(b'\n',1)
            if not line.strip(): continue
            message = json.loads(line)
            if message.get('id') == ident and 'method' not in message:
                if 'error' in message: raise MCPError('JSON-RPC error')
                return message['result']
            if 'id' in message and 'method' in message:
                self.send({'jsonrpc':'2.0','id':message['id'],'error':{'code':-32601,'message':'Client capability not supported'}}, deadline)
        raise TimeoutError('MCP request timeout')

    def tool(self, name, args, timeout=30):
        value = self.call('tools/call', {'name':name,'arguments':args},timeout)
        if value.get('isError'): raise MCPError('Tool returned an error; reconcile resource state')
        return value

    def close(self):
        try: os.killpg(self.p.pid, signal.SIGTERM)
        except ProcessLookupError: pass
        try: self.p.wait(timeout=2)
        except subprocess.TimeoutExpired:
            try: os.killpg(self.p.pid, signal.SIGKILL)
            except ProcessLookupError: pass
            self.p.wait(timeout=2)
        for stream in (self.p.stdin,self.p.stdout): stream.close()
        self.selector.close()
