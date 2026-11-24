package install

// This fixture is confined to a newly created, initially empty network
// namespace. It never touches the installed BoxWeave/Incus network.
import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

const n3EndpointScript = `
import json,socket,sys,threading
flows={}
def echo(s,udp):
    if udp:
        while True:
            data,addr=s.recvfrom(4096)
            s.sendto(data,addr)
    else:
        while True:
            c,_=s.accept()
            threading.Thread(target=stream,args=(c,),daemon=True).start()
def stream(c):
    with c:
        while True:
            data=c.recv(4096)
            if not data:return
            c.sendall(data)
def setup():
    for af,addr in [(socket.AF_INET,"0.0.0.0"),(socket.AF_INET6,"::")]:
        for udp in (False,True):
            s=socket.socket(af,socket.SOCK_DGRAM if udp else socket.SOCK_STREAM)
            if af==socket.AF_INET6:s.setsockopt(socket.IPPROTO_IPV6,socket.IPV6_V6ONLY,1)
            s.bind((addr,18080))
            if not udp:s.listen()
            threading.Thread(target=echo,args=(s,udp),daemon=True).start()
print(json.dumps({"ok":True}),flush=True)
for line in sys.stdin:
    try:
        c=json.loads(line)
        if c["op"]=="setup":setup()
        elif c["op"]=="open":
            af=socket.AF_INET6 if ":" in c["ip"] else socket.AF_INET
            udp=c["udp"]
            s=socket.socket(af,socket.SOCK_DGRAM if udp else socket.SOCK_STREAM)
            s.settimeout(0.65)
            s.connect((c["ip"],18080))
            flows[c["id"]]=s
            s.sendall(b"open")
            assert s.recv(4096)==b"open"
        elif c["op"]=="probe":
            s=flows[c["id"]]
            data=c["msg"].encode()
            s.sendall(data)
            assert s.recv(4096)==data
        elif c["op"]=="close":
            flows.pop(c["id"]).close()
        else:raise ValueError("unknown fixture command")
        print(json.dumps({"ok":True}),flush=True)
    except (OSError,AssertionError) as e:
        print(json.dumps({"ok":False,"error":str(e)}),flush=True)
`

type n3Endpoint struct {
	cmd   *exec.Cmd
	input io.WriteCloser
	out   *bufio.Scanner
}

func n3Cmd(t *testing.T, ctx context.Context, args ...string) {
	t.Helper()
	out, err := exec.CommandContext(ctx, args[0], args[1:]...).CombinedOutput()
	if err != nil {
		t.Fatalf("fixture %v: %v: %s", args, err, out)
	}
}
func (e *n3Endpoint) call(t *testing.T, cmd map[string]any) bool {
	t.Helper()
	if err := json.NewEncoder(e.input).Encode(cmd); err != nil {
		t.Fatal(err)
	}
	if !e.out.Scan() {
		t.Fatal("endpoint response missing", e.out.Err())
	}
	var r struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(e.out.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if !r.OK {
		t.Log("endpoint:", e.out.Text())
	}
	return r.OK
}
func n3NewEndpoint(t *testing.T, ctx context.Context, i int, m NativePeerGuardMember, bridge string) *n3Endpoint {
	t.Helper()
	cmd := exec.CommandContext(ctx, "/usr/bin/unshare", "--net", "/usr/bin/python3", "-u", "-c", n3EndpointScript)
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { in.Close(); cmd.Process.Kill(); cmd.Wait() })
	e := &n3Endpoint{cmd: cmd, input: in, out: bufio.NewScanner(out)}
	if !e.out.Scan() || !strings.Contains(e.out.Text(), "\"ok\": true") {
		t.Fatal("private endpoint namespace not ready")
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	v, p := fmt.Sprintf("n3v%d", i), fmt.Sprintf("n3p%d", i)
	n3Cmd(t, ctx, "/usr/sbin/ip", "link", "add", v, "type", "veth", "peer", "name", p)
	n3Cmd(t, ctx, "/usr/sbin/ip", "link", "set", v, "master", bridge)
	n3Cmd(t, ctx, "/usr/sbin/ip", "link", "set", v, "up")
	n3Cmd(t, ctx, "/usr/sbin/ip", "link", "set", p, "netns", pid)
	run := func(a ...string) {
		n3Cmd(t, ctx, append([]string{"/usr/bin/nsenter", "-t", pid, "-n", "/usr/sbin/ip"}, a...)...)
	}
	run("link", "set", "lo", "up")
	run("link", "set", p, "address", m.MAC)
	run("link", "set", p, "up")
	run("addr", "add", m.IPv4+"/24", "dev", p)
	run("-6", "addr", "add", m.IPv6+"/64", "dev", p, "nodad")
	if !e.call(t, map[string]any{"op": "setup"}) {
		t.Fatal("echo fixture failed")
	}
	return e
}
func TestPR178N3NativeDirectionGuard(t *testing.T) {
	if os.Getenv("BOXWEAVE_PR178_DIRECTION_NETNS") != "isolated-v1" {
		t.Skip("explicit disposable network namespace fixture only")
	}
	if os.Geteuid() != 0 {
		t.Fatal("fixture requires local root")
	}
	self, err := os.Readlink("/proc/self/ns/net")
	if err != nil {
		t.Fatal(err)
	}
	init, err := os.Readlink("/proc/1/ns/net")
	if err != nil || self == init {
		t.Fatal("fixture must never use init network namespace")
	}
	ifs, err := net.Interfaces()
	if err != nil || len(ifs) != 1 || ifs[0].Name != "lo" {
		t.Fatal("fixture requires empty network namespace")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	bridge := pr178GuardTarget().Network
	n3Cmd(t, ctx, "/usr/sbin/ip", "link", "add", bridge, "type", "bridge")
	n3Cmd(t, ctx, "/usr/sbin/ip", "link", "set", bridge, "up")
	m := pr178GuardMembers()
	a, b, c, d := m[0], m[1], m[2], m[3]
	ab := pr178Grant(a.UUID, b.UUID, a.IPv4, b.IPv4, a.IPv6, b.IPv6)
	ba := pr178Grant(b.UUID, a.UUID, b.IPv4, a.IPv4, b.IPv6, a.IPv6)
	cd := pr178Grant(c.UUID, d.UUID, c.IPv4, d.IPv4, c.IPv6, d.IPv6)
	before := pr178Guard(t, m, []state.PeerGrantBinding{ab, ba, cd}, nil)
	after := pr178Guard(t, m, []state.PeerGrantBinding{ba, cd}, nil)
	ep := make([]*n3Endpoint, 4)
	for i := range m {
		ep[i] = n3NewEndpoint(t, ctx, i, m[i], bridge)
	}
	if err := nativePeerGuardFixedNftBatch(ctx, before.Install); err != nil {
		t.Fatal(err)
	}
	if err := NativePeerGuardKernelReadback(ctx, before); err != nil {
		raw, _ := nativePeerGuardNFTRead(ctx, before.Table)
		t.Log("kernel:", string(raw))
		t.Log("expected:", before.ExpectedKernelJSON)
		t.Fatal(err)
	}
	for _, v6 := range []bool{false, true} {
		for _, udp := range []bool{false, true} {
			id := fmt.Sprintf("%t-%t", v6, udp)
			addr := func(x NativePeerGuardMember) string {
				if v6 {
					return x.IPv6
				}
				return x.IPv4
			}
			for _, f := range []struct {
				src  int
				dst  NativePeerGuardMember
				name string
			}{{0, b, "ab"}, {1, a, "ba"}, {2, d, "cd"}} {
				if !ep[f.src].call(t, map[string]any{"op": "open", "ip": addr(f.dst), "udp": udp, "id": f.name + id}) {
					for _, e := range ep {
						out, _ := exec.Command("/usr/bin/nsenter", "-t", strconv.Itoa(e.cmd.Process.Pid), "-n", "/usr/sbin/ip", "-6", "neigh").CombinedOutput()
						t.Log("neighbor:", string(out))
					}
					raw, _ := exec.Command("/usr/sbin/nft", "list", "table", "bridge", before.Table).CombinedOutput()
					t.Log("guard:", string(raw))
					t.Fatalf("baseline %s %s failed", f.name, id)
				}
			}
		}
	}
	out, err := nativePeerGuardReplaceOnce(ctx, before, after, NativePeerGuardKernelReadback, nativePeerGuardFixedNftBatch)
	if err != nil || !out.KernelVerified {
		t.Fatalf("revocation failed %+v %v", out, err)
	}
	// Under the already-revoked guard, invalidate only the original AB
	// tuples. BA and CD must remain in the kernel, not reconnect silently.
	box := func(m NativePeerGuardMember) native.PeerACLBox {
		return native.PeerACLBox{UUID: m.UUID, IPv4: m.IPv4, IPv6: m.IPv6, MAC: m.MAC}
	}
	report, err := nativeScopedDirectionalConntrackPurge(ctx, nativePeerConntrackOS, box(a), box(b), func(ctx context.Context) error { return NativePeerGuardKernelReadback(ctx, after) })
	if err != nil || report.ProbedScopes != 2 || report.DeletedScopes != 2 {
		t.Fatalf("exact directional purge: %+v %v", report, err)
	}
	for _, p := range []struct{ family, src, dst string }{{"ipv4", b.IPv4, a.IPv4}, {"ipv6", b.IPv6, a.IPv6}, {"ipv4", c.IPv4, d.IPv4}, {"ipv6", c.IPv6, d.IPv6}} {
		found, e := nativePeerConntrackList(ctx, nativePeerConntrackOS, p.family, p.src, p.dst)
		if e != nil || !found {
			t.Fatalf("reverse/unrelated original tuple lost: %+v %v", p, e)
		}
	}
	for _, v6 := range []bool{false, true} {
		for _, udp := range []bool{false, true} {
			id := fmt.Sprintf("%t-%t", v6, udp)
			for _, f := range []struct {
				src  int
				name string
				want bool
			}{{0, "ab", false}, {1, "ba", true}, {2, "cd", true}} {
				got := ep[f.src].call(t, map[string]any{"op": "probe", "id": f.name + id, "msg": "after-revoke-" + id})
				if got != f.want {
					t.Fatalf("post-revoke %s %s: got=%t want=%t", f.name, id, got, f.want)
				}
			}
		}
	}
	if err := NativePeerGuardKernelReadback(ctx, after); err != nil {
		t.Fatal(err)
	}
	t.Log("PR178_N3_TRACKED_DIRECTION_REVOKE_PASS: IPv4 IPv6 TCP UDP, reverse and unrelated live flows retained")
}
