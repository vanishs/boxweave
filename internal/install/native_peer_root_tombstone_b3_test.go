package install

import (
	"github.com/vanishs/boxweave/internal/state"
	"testing"
)

func TestS4B3TombstoneSurvivesNextGrantAndRegrantRemovesOnlyOwnDrop(t *testing.T) {
	row := func(a, b, ip4a, ip4b, ip6a, ip6b string) state.PeerGrantBinding {
		return state.PeerGrantBinding{SourceUUID: a, SourceIncarnation: a, TargetUUID: b, TargetIncarnation: b,
			SourceIPv4: ip4a, TargetIPv4: ip4b, SourceIPv6: ip6a, TargetIPv6: ip6b}
	}
	ab := row("a", "b", "10.55.8.11", "10.55.8.12", "fd55:8::11", "fd55:8::12")
	bc := row("b", "c", "10.55.8.12", "10.55.8.13", "fd55:8::12", "fd55:8::13")
	cb := row("c", "b", "10.55.8.13", "10.55.8.12", "fd55:8::13", "fd55:8::12")
	t0 := NativePrepareTarget{ACL: "bw-s4b2-unit"}
	n := s4b2UnitBridge()
	pre, e := NativePeerACLFromPolicySnapshot(t0, n, []state.PeerGrantBinding{bc}, []state.PeerGrantBinding{ab})
	if e != nil {
		t.Fatal(e)
	}
	post, e := NativePeerACLFromPolicySnapshot(t0, n, []state.PeerGrantBinding{bc, cb}, []state.PeerGrantBinding{ab})
	if e != nil {
		t.Fatal(e)
	}
	for _, w := range []struct{ src, dst string }{
		{"10.55.8.11/32", "10.55.8.12/32"},
		{"10.55.8.12/32", "10.55.8.11/32"},
		{"fd55:8::11/128", "fd55:8::12/128"},
		{"fd55:8::12/128", "fd55:8::11/128"},
	} {
		npre, npost := 0, 0
		for _, r := range pre.Ingress {
			if r.Source == w.src && r.Destination == w.dst && r.Action == "drop" {
				npre++
			}
		}
		for _, r := range post.Ingress {
			if r.Source == w.src && r.Destination == w.dst && r.Action == "drop" {
				npost++
			}
		}
		if npre != 1 || npost != 1 {
			t.Fatalf("historical DROP lost %s>%s: before=%d after=%d", w.src, w.dst, npre, npost)
		}
	}
	for _, w := range []struct{ src, dst string }{
		{"10.55.8.13/32", "10.55.8.12/32"},
		{"fd55:8::13/128", "fd55:8::12/128"},
	} {
		found := false
		for _, r := range post.Ingress {
			if r.Source == w.src && r.Destination == w.dst && r.Action == "allow" {
				found = true
			}
		}
		if !found {
			t.Fatalf("new grant missing %s>%s", w.src, w.dst)
		}
	}
	regrant, e := NativePeerACLFromPolicySnapshot(t0, n, []state.PeerGrantBinding{ab, bc}, nil)
	if e != nil {
		t.Fatal(e)
	}
	allowed := false
	for _, r := range regrant.Ingress {
		if r.Source == "10.55.8.11/32" && r.Destination == "10.55.8.12/32" {
			if r.Action == "drop" {
				t.Fatal("stale DROP blocks exact regrant")
			}
			if r.Action == "allow" {
				allowed = true
			}
		}
	}
	if !allowed {
		t.Fatal("exact regrant lost")
	}
}
