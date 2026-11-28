package state

import (
	"context"
	"fmt"
	"github.com/vanishs/boxweave/internal/peerscope"
	"net/netip"
	"strings"
	"testing"
)

// Retained identity histories must survive restart.
func TestIPv4RetirementQueuePersistentRoundRobin(t *testing.T) {
	ctx := context.Background()
	s, b, d := n3RetiredFixture(t)
	path := strings.TrimSuffix(s.ExecArchiveDir(), ".exec-archive")
	check := func(s *Store) {
		t.Helper()
		ledger, e := s.ReadCompleteScopedNetworkAuditLedger(ctx, b.Project, purePeerBridge)
		if e != nil || len(ledger.Retired) != 1 || ledger.Retired[0].DeleteID != d.ID {
			t.Fatal(ledger, e)
		}
	}
	check(s)
	if e := s.Close(); e != nil {
		t.Fatal(e)
	}
	again, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer again.Close()
	check(again)
}

// Same-slot reuse is retired; keep the beyond-256 oracle on distinct exact
// retired identities, plus actual admission of a previously unused address.
func TestIPv4RetirementChurnBeyond256Originals(t *testing.T) {
	ctx := context.Background()
	s, b, _ := n3RetiredFixture(t)
	seedReleasedNetworkHistory(t, s, 257)
	ledger, e := s.ReadCompleteScopedNetworkAuditLedger(ctx, b.Project, purePeerBridge)
	if e != nil || len(ledger.Retired) != 258 {
		t.Fatal(len(ledger.Retired), e)
	}
	h, e := s.ReadScopedCreateAddressHolds(ctx, b.Project, purePeerBridge)
	if e != nil {
		t.Fatal(e)
	}
	for _, r := range ledger.Retired {
		if !h.IPs[netip.MustParseAddr(r.IPv4)] || !h.IPs[netip.MustParseAddr(r.IPv6)] || !h.MACs[r.MAC] {
			t.Fatal("lost history", r.UUID)
		}
	}
	r := scopedReplacementOriginal(t, s, purePeerNewToken(t, s), "after-churn")
	pin := VerifiedCreateBinding{purePeerBridge, peerscope.QuarantineACL, "02:16:3e:44:55:99", ledger.Retired[257].IPv4, "fd43:18::99"}
	if ok, e := s.StageCreateBinding(ctx, r.ID, b.Project, "after-churn", pin); ok || e != nil {
		t.Fatal(ok, e)
	}
	pin.IPv4 = "10.43.18.99"
	if ok, e := s.StageCreateBinding(ctx, r.ID, b.Project, "after-churn", pin); !ok || e != nil {
		t.Fatal(ok, e)
	}
	if e = s.NativeAck(ctx, r.ID, "native-after-history"); e != nil {
		t.Fatal(e)
	}
	uuid := fmt.Sprintf("33333333-3333-4333-8333-%012d", 1)
	pin.ACL, e = peerscope.ACLName(b.Project, purePeerBridge, uuid)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeVerifiedScopedCreate(ctx, r.ID, b.Project, "after-churn", uuid, pin); e != nil {
		t.Fatal(e)
	}
	ledger, e = s.ReadCompleteScopedNetworkAuditLedger(ctx, b.Project, purePeerBridge)
	if e != nil || len(ledger.Retired) != 258 || len(ledger.Boxes) != 3 {
		t.Fatal(len(ledger.Retired), len(ledger.Boxes), e)
	}
}

func TestIPv4RetirementStatusScopedDiagnostics(t *testing.T) {
	ctx := context.Background()
	s, b, _ := n3RetiredFixture(t)
	snap := DiagnosticSnapshot{store: s}
	first, e := snap.ReadScopedMaintenanceLedger(ctx, b.Project)
	if e != nil || len(first.LegacyPending) != 0 || len(first.LegacyCompactions) != 0 || first.Retired != 1 || len(first.HistorySHA256) != 64 {
		t.Fatal(first, e)
	}
	after, e := snap.ReadScopedMaintenanceLedger(ctx, b.Project)
	if e != nil || after.ObservationDigest() != first.ObservationDigest() {
		t.Fatal(after, e)
	}
	foreign, e := snap.ReadScopedMaintenanceLedger(ctx, "user-1002")
	if e != nil || len(foreign.RetiredBoxes) != 0 || len(foreign.Boxes) != 0 {
		t.Fatal(foreign, e)
	}
}
