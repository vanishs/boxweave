package native

import (
	"encoding/json"
	"github.com/lxc/incus/v7/shared/api"
	"slices"
)

// Canonicalize complete lease records as JSON before comparing independent
// reads. API ordering is not a stable witness; each lease and its duplicates
// still count. No field is discarded from the native evidence.
func canonicalNetworkLeaseSet(leases []api.NetworkLease) ([]string, error) {
	if len(leases) > 131072 {
		return nil, ErrNativeUnsupported
	}
	out := make([]string, 0, len(leases))
	for _, lease := range leases {
		raw, e := json.Marshal(lease)
		if e != nil {
			return nil, e
		}
		out = append(out, string(raw))
	}
	slices.Sort(out)
	return out, nil
}
