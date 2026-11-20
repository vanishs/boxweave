package install

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestS4B2ConntrackInventoryNeedsNativeZeroOrMatchingOriginal(t *testing.T) {
	a, b := s4b2ConntrackPair()
	flow := fmt.Sprintf("tcp 6 390 ESTABLISHED src=%s dst=%s sport=15000 dport=19091", a.IPv4, b.IPv4)
	zero := "conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown"
	cases := []struct {
		name, out          string
		cmdErr             error
		wantFound, wantErr bool
	}{
		{"empty-success", "", nil, false, true},
		{"explicit-zero", zero, nil, false, false},
		{"explicit-zero-exit-one", zero, errors.New("no matches"), false, false},
		{"nonzero-summary-without-tuples", "conntrack v1.4.8: 10 flow entries have been shown", nil, false, true},
		{"unrelated-original", "tcp 6 src=10.248.83.17 dst=10.248.83.19 sport=11 dport=22", nil, false, true},
		{"permission-diagnostic", "Operation not permitted", nil, false, true},
		{"valid-original", flow, nil, true, false},
		{"valid-with-count", flow + "\nconntrack v1.4.8: 1 flow entries have been shown", nil, true, false},
		{"zero-and-tuple", flow + "\n" + zero, nil, false, true},
		{"partial-error-with-tuple", flow, errors.New("incomplete read"), false, true},
		{"zero-with-error-text", zero + "\npermission denied", errors.New("exit 1"), false, true},
		{"empty-error", "", errors.New("I/O failure"), false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			run := func(_ context.Context, args ...string) ([]byte, error) {
				calls++
				if len(args) != 7 || args[0] != "-L" || args[1] != "-f" || args[2] != "ipv4" || args[3] != "-s" || args[4] != a.IPv4 || args[5] != "-d" || args[6] != b.IPv4 {
					t.Fatalf("unbounded conntrack arguments %v", args)
				}
				return []byte(tc.out), tc.cmdErr
			}
			found, err := nativePeerConntrackList(t.Context(), run, "ipv4", a.IPv4, b.IPv4)
			if calls != 1 || found != tc.wantFound || (err != nil) != tc.wantErr {
				t.Fatalf("malformed native output became a false zero or entry: found=%v error=%v", found, err)
			}
		})
	}
}

func TestS4B2ConntrackUncertainInventoryNeverDeletes(t *testing.T) {
	a, b := s4b2ConntrackPair()
	for _, bad := range []string{"", "conntrack v1.4.8: 10 flow entries have been shown", "permission denied"} {
		deleted := 0
		run := func(_ context.Context, args ...string) ([]byte, error) {
			if args[0] == "-D" {
				deleted++
				t.Fatal("unknown inventory reached native delete")
			}
			return []byte(bad), nil
		}
		result, err := nativePeerConntrackPurgeExact(t.Context(), run, a, b)
		if err == nil || deleted != 0 || result.ProbedScopes != 0 || result.ReadyForPeer {
			t.Fatalf("unsafe unknown inventory accepted: %+v %v", result, err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	called := 0
	if _, err := nativePeerConntrackList(ctx, func(context.Context, ...string) ([]byte, error) { called++; return nil, nil }, "ipv4", a.IPv4, b.IPv4); !errors.Is(err, context.Canceled) || called != 0 {
		t.Fatal("canceled root conntrack precheck incorrectly dispatched")
	}
}
