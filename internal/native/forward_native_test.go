package native

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/lxc/incus/v7/shared/api"
)

func b4NativeFinal(t *testing.T, r Reader, id string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	op, err := r.ObserveUntilFinal(ctx, id)
	if err != nil || op == nil || op.ID != id || op.StatusCode != api.Success {
		t.Fatalf("original native update terminal not independently verified: id=%q op=%+v err=%v", id, op, err)
	}
}

func b4Echo(t *testing.T, protocol string, port int, shouldReply bool) {
	t.Helper()
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	c, err := net.DialTimeout(protocol, address, 2*time.Second)
	if err != nil {
		if shouldReply {
			t.Fatalf("%s did not open: %v", address, err)
		}
		return
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(2 * time.Second))
	message := []byte("b4-exact-proxy-echo")
	_, writeErr := c.Write(message)
	reply := make([]byte, len(message))
	var readErr error
	if writeErr == nil {
		var n int
		n, readErr = c.Read(reply)
		if readErr == nil && string(reply[:n]) != string(message) {
			readErr = fmt.Errorf("payload changed: %q", reply[:n])
		}
	}
	if shouldReply && (writeErr != nil || readErr != nil) {
		t.Fatalf("%s lacked real echo: write=%v read=%v", address, writeErr, readErr)
	}
	if !shouldReply && writeErr == nil && readErr == nil {
		t.Fatalf("%s still forwarded after exact native deletion", address)
	}
}

// This native test can run ONLY in the fresh, disposable one-use CI host.
// It never uses a shared production L0 socket or unrestricted root credentials.
func TestB4NativeOwnedProxyTCPUDP(t *testing.T) {
	if os.Getenv("BOXWEAVE_B4_NATIVE_ONE_USE") != "s4b4-disposable-only-20261119" ||
		os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("requires explicitly disposable B4 runner")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	r := Reader{Socket: "/var/lib/incus/unix.socket.user", Project: "user-" + strconv.Itoa(os.Getuid())}
	if err := r.Strict(ctx); err != nil {
		t.Fatal(err)
	}
	name := "s4b4native"
	c, err := r.confined(ctx)
	if err != nil {
		t.Fatal(err)
	}
	inst, _, err := c.GetInstance(name)
	if err != nil {
		t.Fatal(err)
	}
	uuid := inst.Config["volatile.uuid"]
	if uuid == "" || !nicPrivate(inst.ExpandedDevices) {
		t.Fatal("non-owned or unsafe native fixture")
	}
	for _, tc := range []struct {
		protocol            string
		hostPort, guestPort int
		device              string
	}{
		{"tcp", 23091, 19091, "bwpf-1111111111111111"},
		{"udp", 23092, 19092, "bwpf-2222222222222222"},
	} {
		t.Run(tc.protocol, func(t *testing.T) {
			f := ForwardSpec{Device: tc.device, Protocol: tc.protocol, HostBind: "127.0.0.1", HostPort: tc.hostPort, GuestPort: tc.guestPort}
			if err = r.ObserveOwnedForward(ctx, name, uuid, f, true); err == nil {
				t.Fatal("unexpected preexisting foreign proxy")
			}
			id, err := r.UpdateOwnedForward(ctx, name, uuid, f, true)
			if err != nil || id == "" {
				t.Fatalf("native create rejected: %q %v", id, err)
			}
			b4NativeFinal(t, r, id)
			if err = r.ObserveOwnedForward(ctx, name, uuid, f, true); err != nil {
				t.Fatalf("original create poststate: %v", err)
			}
			b4Echo(t, tc.protocol, tc.hostPort, true)
			t.Logf("S4_B4_NATIVE_%s_ORIGINAL_ETAG_PROXY_PACKET=PASS", tc.protocol)
			removed, err := r.UpdateOwnedForward(ctx, name, uuid, f, false)
			if err != nil || removed == "" {
				t.Fatalf("native delete rejected: %q %v", removed, err)
			}
			b4NativeFinal(t, r, removed)
			if err = r.ObserveOwnedForward(ctx, name, uuid, f, false); err != nil {
				t.Fatalf("original delete poststate: %v", err)
			}
			b4Echo(t, tc.protocol, tc.hostPort, false)
			t.Logf("S4_B4_NATIVE_%s_REMOVAL_PACKET_REFUSED=PASS", tc.protocol)
		})
	}
}
