package install

import (
	"strings"
	"testing"
)

func TestPR177D3ScopedExplicitReadOnlySelector(t *testing.T) {
	id := strings.Repeat("a", 32)
	for _, tc := range []struct {
		action, phase string
		recover       bool
		want          string
	}{
		{"box_peer_set", "UNKNOWN", true, "PEER_RECOVER"},
		{"box_peer_set", "DISPATCHING", true, ""},
		{"box_peer_set", "ADMITTED", true, ""},
		{"box_peer_set", "SUCCEEDED", true, ""},
		{"box_peer_set", "UNKNOWN", false, ""},
		{"box_peer_set", "DISPATCHING", false, "PEER_APPLY"},
		{"box_create", "UNKNOWN", true, "CREATE_RECOVER"},
		{"box_delete", "UNKNOWN", true, "DELETE_RECOVER"},
		{"box_create", "UNKNOWN", false, ""},
		{"box_delete", "UNKNOWN", false, ""},
		{"box_create", "DISPATCHING", true, ""},
		{"box_delete", "SUCCEEDED", true, ""},
		{"box_create", "VERIFYING", true, ""},
		{"box_delete", "VERIFYING", true, ""},
		{"box_create", "VERIFYING", false, "CREATE"},
		{"box_delete", "VERIFYING", false, "DELETE"},
	} {
		req := NativePeerRootRequest{OperationID: id, ReconcileApproved: tc.recover}
		if got := scopedLifecycleRequestKind(req, tc.action, tc.phase); got != tc.want {
			t.Fatal(tc, got)
		}
	}
	if err := nativePeerRootValidateRequest(NativePeerRootRequest{OperationID: "invalid"}); err == nil {
		t.Fatal("invalid original accepted")
	}
	t.Log("D3_EXPLICIT_ROOT_SELECTOR_NEVER_DISPATCHES_PASS")
}
