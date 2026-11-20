package state

import (
	"slices"
	"strings"
)

// normalizeAdmissionResourceKeys is a deterministic, bounded set of internal
// resource claims, never an MCP-supplied arbitrary list. It includes the old
// one-Box resource lock for compatibility, and makes cross-Box+ACL operations
// reserve all relevant resources in the same SQLite admission transaction.
func normalizeAdmissionResourceKeys(a Admission) ([]string, error) {
	if len(a.ResourceKeys) > 8 {
		return nil, ErrInvalid
	}
	keys := make([]string, 0, len(a.ResourceKeys)+1)
	if a.ResourceKey != "" {
		keys = append(keys, a.ResourceKey)
	}
	for _, key := range a.ResourceKeys {
		if !validInternalClaimKey(key) {
			return nil, ErrInvalid
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)
	keys = slices.Compact(keys)
	if len(keys) > 8 {
		return nil, ErrInvalid
	}
	return keys, nil
}

func validInternalClaimKey(key string) bool {
	if len(key) < 5 || len(key) > 192 {
		return false
	}
	scope, value, ok := strings.Cut(key, ":")
	if !ok || value == "" {
		return false
	}
	switch scope {
	case "box", "acl", "listener", "address", "nic":
	default:
		return false
	}
	for _, c := range value {
		if c > 127 || !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '.' || c == '-' || c == '_' || c == '/' || c == ':') {
			return false
		}
	}
	return true
}

// Preserve the original S2/S4-A digest byte-for-byte for every legacy
// single-resource operation. Only explicit multi-resource admissions bind the
// canonical set to the intent. Replay and Admit share this exact formula.
func admissionIntentDigest(a Admission, claims []string) string {
	raw := a.Action + "\x00" + string(a.SafeIntent) + "\x00" + a.BoxUUID + "\x00" + a.Incarnation
	if len(a.ResourceKeys) != 0 {
		raw += "\x00" + strings.Join(claims, "\x00")
	}
	return digest(raw)
}
