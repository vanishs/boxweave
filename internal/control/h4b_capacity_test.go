package control

import (
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
	"testing"
)

func TestH4BCapacityNavigationDoesNotImplyFaultOrRetry(t *testing.T) {
	f := newToolFailure("box_create", state.ErrScopedIdentityCapacity)
	if f.Code != "SCOPED_RETIRED_IDENTITY_CAPACITY" || f.Category != "CAPACITY" || f.Admission != "NOT_ADMITTED" || f.Next.Tool != "host_status" {
		t.Fatal(f)
	}
	s := HostStatus{Ready: true, AddressCapacity: &native.AddressCapacity{Level: "LOW"}}
	if !s.CapacityNeedsReview() || !s.Ready {
		t.Fatal(s)
	}
	s.AddressCapacity.Level = "AVAILABLE"
	s.ScopedPeerCapacity = &state.DiagnosticPeerCapacity{DirectionLevel: "EXHAUSTED", GrantHistoryLevel: "AVAILABLE"}
	if !s.CapacityNeedsReview() {
		t.Fatal(s)
	}
	s.ScopedPeerCapacity.DirectionLevel = "AVAILABLE"
	if s.CapacityNeedsReview() {
		t.Fatal(s)
	}
}
