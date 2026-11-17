package install

import "testing"

func TestS4B2ExclusiveNativeBridgeConsumersRefuseForeignAndUnknown(t *testing.T) {
	p := "/1.0/profiles/default?project=user-1001"
	a := "/1.0/instances/box-a?project=user-1001"
	b := "/1.0/instances/box-b?project=user-1001"
	known := map[string]string{"box-a": "a-native-uuid", "box-b": "b-native-uuid"}
	for _, refs := range [][]string{{p}, {p, a}, {p, a, b}} {
		if err := nativePeerBridgeConsumersOwned(refs, "user-1001", known); err != nil {
			t.Fatalf("exact expected consumers rejected %v: %v", refs, err)
		}
	}
	bad := []struct {
		name string
		refs []string
	}{
		{"no references", nil},
		{"foreign default Project managed NIC", []string{p, "/1.0/instances/privilegedbox"}},
		{"foreign direct managed", []string{p, "/1.0/instances/foreign?project=other"}},
		{"foreign raw-parent profile", []string{p, "/1.0/profiles/s4b2rawparent"}},
		{"unknown same Project NIC", []string{p, "/1.0/instances/unknown?project=user-1001"}},
		{"foreign Profile", []string{p, "/1.0/profiles/extra?project=user-1001"}},
		{"missing default Profile", []string{a}},
		{"repeated profile", []string{p, p}},
		{"ambiguous query", []string{p, "/1.0/instances/box-a?project=user-1001&project=other"}},
		{"outside API", []string{p, "/1.0/networks/bwbr0?project=user-1001"}},
		{"absolute URL", []string{p, "https://example.org/1.0/instances/box-a?project=user-1001"}},
		{"fragment", []string{p, a + "#foreign"}},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			if err := nativePeerBridgeConsumersOwned(tt.refs, "user-1001", known); err == nil {
				t.Fatalf("unsafe native consumer accepted: %v", tt.refs)
			}
		})
	}
}
