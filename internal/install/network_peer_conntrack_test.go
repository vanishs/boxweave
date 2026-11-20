package install

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
)

func s4b2ConntrackPair() (native.PeerACLBox, native.PeerACLBox) {
	return native.PeerACLBox{Name: "s4b2a", UUID: "uuid-a", IPv4: "10.248.83.11", IPv6: "fd83:248::11"},
		native.PeerACLBox{Name: "s4b2b", UUID: "uuid-b", IPv4: "10.248.83.12", IPv6: "fd83:248::12"}
}
func s4b2ConntrackTombstone(a, b native.PeerACLBox) *api.NetworkACL {
	acl := &api.NetworkACL{}
	for _, pair := range []struct {
		x, y string
		bits int
	}{
		{a.IPv4, b.IPv4, 32}, {b.IPv4, a.IPv4, 32}, {a.IPv6, b.IPv6, 128}, {b.IPv6, a.IPv6, 128},
	} {
		r := api.NetworkACLRule{Action: "drop", State: "enabled", Source: fmt.Sprintf("%s/%d", pair.x, pair.bits),
			Destination: fmt.Sprintf("%s/%d", pair.y, pair.bits)}
		acl.Ingress = append(acl.Ingress, r)
		acl.Egress = append(acl.Egress, r)
	}
	return acl
}
func TestS4B2RootConntrackRequiresNativeExactReverseDrops(t *testing.T) {
	a, b := s4b2ConntrackPair()
	acl := s4b2ConntrackTombstone(a, b)
	if err := nativePeerACLDropCovered(acl, a, b); err != nil {
		t.Fatal(err)
	}
	acl.Ingress = acl.Ingress[:3]
	if err := nativePeerACLDropCovered(acl, a, b); err == nil {
		t.Fatal("partial IPv6 reverse-side DROP authorized conntrack purge")
	}
	acl = s4b2ConntrackTombstone(a, b)
	acl.Egress = acl.Egress[:3]
	if err := nativePeerACLDropCovered(acl, a, b); err == nil {
		t.Fatal("missing egress reverse DROP authorized conntrack purge")
	}
	acl = s4b2ConntrackTombstone(a, b)
	acl.Ingress = append(acl.Ingress, api.NetworkACLRule{Action: "allow", Source: "10.248.83.0/24", Destination: "10.248.83.12/32"})
	if err := nativePeerACLDropCovered(acl, a, b); err == nil {
		t.Fatal("CIDR overlapping drop grant accepted")
	}
	acl = s4b2ConntrackTombstone(a, b)
	acl.Egress = append(acl.Egress, api.NetworkACLRule{Action: "allow", Source: "", Destination: ""})
	if err := nativePeerACLDropCovered(acl, a, b); err == nil {
		t.Fatal("wildcard overlapping grant accepted")
	}
	acl = s4b2ConntrackTombstone(a, b)
	acl.Ingress = append(acl.Ingress, api.NetworkACLRule{Action: "allow", Source: "10.248.83.13/32", Destination: "10.248.83.14/32"})
	if err := nativePeerACLDropCovered(acl, a, b); err != nil {
		t.Fatal("unrelated exact foreign pair not preserved", err)
	}
	acl = s4b2ConntrackTombstone(a, b)
	acl.Ingress = append(acl.Ingress, api.NetworkACLRule{Action: "allow", Protocol: "icmp6", ICMPType: "135"})
	if err := nativePeerACLDropCovered(acl, a, b); err != nil {
		t.Fatal("native ICMPv6 neighbor discovery should not reauthorize TCP", err)
	}
}
func TestS4B2RootConntrackDeletesOnlyVerifiedPairAndNeverFlushes(t *testing.T) {
	a, b := s4b2ConntrackPair()
	sessions := map[string]bool{}
	known := map[string]bool{}
	for _, pair := range []struct{ family, x, y string }{
		{"ipv4", a.IPv4, b.IPv4}, {"ipv4", b.IPv4, a.IPv4}, {"ipv6", a.IPv6, b.IPv6}, {"ipv6", b.IPv6, a.IPv6},
	} {
		key := pair.family + "|" + pair.x + "|" + pair.y
		sessions[key] = true
		known[key] = true
	}
	foreign := "ipv4|10.248.83.13|10.248.83.14"
	sessions[foreign] = true
	deletes := 0
	run := func(_ context.Context, args ...string) ([]byte, error) {
		key := ""
		switch args[0] {
		case "-L":
			if len(args) != 7 || args[1] != "-f" ||
				(args[2] != "ipv4" && args[2] != "ipv6") ||
				args[3] != "-s" || args[5] != "-d" {
				t.Fatalf("unbounded root conntrack -L argv %v", args)
			}
			key = args[2] + "|" + args[4] + "|" + args[6]
		case "-D":
			// conntrack-tools v1.4.8 rejects --family with -D. Kernel
			// derives family from the already-validated numeric IPs.
			if len(args) != 5 || args[1] != "-s" || args[3] != "-d" {
				t.Fatalf("invalid root conntrack -D argv %v", args)
			}
			family := "ipv4"
			if strings.Contains(args[2], ":") {
				family = "ipv6"
			}
			key = family + "|" + args[2] + "|" + args[4]
		default:
			t.Fatalf("unsafe root conntrack command %v", args)
		}
		if key == foreign {
			t.Fatal("unrelated tuple touched")
		}
		if _, ok := known[key]; !ok {
			t.Fatal("unexpected native pair scope " + key)
		}
		if args[0] == "-L" {
			if sessions[key] {
				return []byte("tcp 6 390 ESTABLISHED src=" + args[4] + " dst=" + args[6] + " sport=1000 dport=19091"), nil
			}
			return []byte("conntrack v1.4.8: 0 flow entries have been shown"), nil
		}
		deletes++
		delete(sessions, key)
		return []byte("1 flow entries have been deleted"), nil
	}
	receipt, err := nativePeerConntrackPurgeExact(t.Context(), run, a, b)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ReadyForPeer || receipt.ProbedScopes != 4 || receipt.DeletedScopes != 4 ||
		!sessions[foreign] || deletes != 4 {
		t.Fatalf("root conntrack pair isolation failed: %+v remaining=%v calls=%d", receipt, sessions, deletes)
	}
}
func TestS4B2RootConntrackNativeZeroAndUnknownNeverRetry(t *testing.T) {
	a, b := s4b2ConntrackPair()
	listCalls := 0
	noFlow := func(_ context.Context, args ...string) ([]byte, error) {
		if args[0] != "-L" {
			t.Fatal("empty conntrack inventory should not delete")
		}
		listCalls++
		return []byte("conntrack v1.4.8: 0 flow entries have been shown"), errors.New("no matching entries")
	}
	rep, err := nativePeerConntrackPurgeExact(t.Context(), noFlow, a, b)
	if err != nil || listCalls != 4 || rep.DeletedScopes != 0 || rep.ReadyForPeer {
		t.Fatal("exact empty pair state needs observed zero rather than blind deletion", err, rep)
	}
	failCount := 0
	ambiguous := func(_ context.Context, args ...string) ([]byte, error) {
		if args[0] == "-L" {
			return []byte("tcp 6 src=" + a.IPv4 + " dst=" + b.IPv4), nil
		}
		failCount++
		return []byte("native ACK was lost"), errors.New("lost ACK")
	}
	rep, err = nativePeerConntrackPurgeExact(t.Context(), ambiguous, a, b)
	if err == nil || failCount != 1 || rep.DeletedScopes != 0 || !strings.Contains(err.Error(), "uncertain") {
		t.Fatal("uncertain native delete was retried or falsely accepted", rep, err)
	}
	bad := a
	bad.IPv4 = "--flush"
	if _, err := nativePeerConntrackPurgeExact(t.Context(), noFlow, bad, b); err == nil {
		t.Fatal("untrusted host command argument accepted")
	}
	bad = a
	bad.IPv6 = "fd83:248::12"
	if _, err := nativePeerConntrackPurgeExact(t.Context(), noFlow, bad, b); err == nil {
		t.Fatal("duplicate native IPv6 accepted")
	}
	if _, err := nativePeerConntrackPurgeExact(t.Context(), nil, a, b); err == nil {
		t.Fatal("nil privileged runner accepted")
	}
}
