package install

import (
	"context"
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func TestS4B2NativePeerConntrackReadOnlyPlanSealsExactIdentityAndHost(t *testing.T) {
	original := state.NetworkAuditLedger{
		Project: "user-1001", Bridge: "bwbr0", ACL: "bw-s4b2",
		Boxes: []state.NetworkAuditBinding{
			{Name: "a", UUID: "uuid-a", Incarnation: "uuid-a", MAC: "02:11:22:33:44:01", IPv4: "10.243.11.2", IPv6: "fd42::2"},
			{Name: "b", UUID: "uuid-b", Incarnation: "uuid-b", MAC: "02:11:22:33:44:02", IPv4: "10.243.11.3", IPv6: "fd42::3"},
		},
	}
	h, n := strings.Repeat("a", 64), strings.Repeat("b", 64)
	first, err := nativePeerConntrackSealedPlan(original, h, n, "uuid-a", "uuid-b")
	if err != nil || len(first.PlanSHA256) != 64 || first.MutationAuthorized || first.SourceUUID != "uuid-a" {
		t.Fatalf("invalid bounded read-only plan: %+v %v", first, err)
	}
	second, err := nativePeerConntrackSealedPlan(original, h, n, "uuid-a", "uuid-b")
	if err != nil || first != second {
		t.Fatal("same root state must yield deterministic review", err)
	}
	if changed, _ := nativePeerConntrackSealedPlan(original, h, n, "uuid-b", "uuid-a"); changed.PlanSHA256 == first.PlanSHA256 {
		t.Fatal("pair direction not sealed")
	}
	if changed, _ := nativePeerConntrackSealedPlan(original, strings.Repeat("c", 64), n, "uuid-a", "uuid-b"); changed.PlanSHA256 == first.PlanSHA256 {
		t.Fatal("physical host not sealed")
	}
	if changed, _ := nativePeerConntrackSealedPlan(original, h, strings.Repeat("d", 64), "uuid-a", "uuid-b"); changed.PlanSHA256 == first.PlanSHA256 {
		t.Fatal("actual native root inventory not sealed")
	}
	replacement := original
	replacement.Boxes = append([]state.NetworkAuditBinding(nil), original.Boxes...)
	replacement.Boxes[1].MAC = "02:11:22:33:44:fe"
	if changed, _ := nativePeerConntrackSealedPlan(replacement, h, n, "uuid-a", "uuid-b"); changed.PlanSHA256 == first.PlanSHA256 {
		t.Fatal("NIC MAC drift not sealed")
	}
	replacement = original
	replacement.Boxes = append([]state.NetworkAuditBinding(nil), original.Boxes...)
	replacement.Boxes[1].IPv6 = "fd42::99"
	if changed, _ := nativePeerConntrackSealedPlan(replacement, h, n, "uuid-a", "uuid-b"); changed.PlanSHA256 == first.PlanSHA256 {
		t.Fatal("dual-stack reuse drift not sealed")
	}
	for _, p := range [][2]string{{"uuid-a", "uuid-a"}, {"missing", "uuid-b"}, {"uuid-a", "missing"}} {
		if _, err := nativePeerConntrackSealedPlan(original, h, n, p[0], p[1]); err == nil {
			t.Fatalf("unowned pair plan accepted: %v", p)
		}
	}
	if _, err := nativePeerConntrackSealedPlan(original, "not-a-host", n, "uuid-a", "uuid-b"); err == nil {
		t.Fatal("missing canonical host rejected too late")
	}
	if _, err := nativePeerConntrackSealedPlan(original, h, "not-native", "uuid-a", "uuid-b"); err == nil {
		t.Fatal("missing native SHA admitted")
	}
	if _, err := nativePeerConntrackReviewWithClient(context.Background(), nil, nil, h, "user-1001", "bwbr0", "bw-s4b2", "uuid-a", "uuid-b"); err == nil {
		t.Fatal("unprotected state or nil native SDK admitted")
	}
}
