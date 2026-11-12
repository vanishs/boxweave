package native

import (
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

// A vanished GET never by itself proves finality. The caller must receive
// two successful native project-scoped active-operation listings and verify
// exact immutable UUID/config postconditions separately.
func TestOriginalCreateOperationListingCannotHideActiveOriginal(t *testing.T) {
	original := "11111111-2222-3333-4444-555555555555"
	cases := []struct {
		name       string
		operations []api.Operation
		active     bool
	}{
		{"empty_after_expiration", nil, false},
		{"unrelated_active", []api.Operation{{ID: "foreign-op-id", StatusCode: api.Running}}, false},
		{"same_original_running", []api.Operation{{ID: original, StatusCode: api.Running}}, true},
		{"same_original_pending", []api.Operation{{ID: original, StatusCode: api.Pending}}, true},
		{"same_original_final_success", []api.Operation{{ID: original, StatusCode: api.Success}}, false},
		{"same_original_final_failure", []api.Operation{{ID: original, StatusCode: api.Failure}}, false},
		{"original_active_with_other_final", []api.Operation{
			{ID: "foreign-op-id", StatusCode: api.Success},
			{ID: original, StatusCode: api.Started},
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := nativeOriginalStillActive(original, tc.operations); got != tc.active {
				t.Fatalf("original native effect activity mislabeled: got=%v want=%v", got, tc.active)
			}
		})
	}
}
