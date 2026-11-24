package networklimits

// ScopedPeerHistoryLimit bounds the immutable Root policy chain. Reserve
// one future record for revoking each active direction, so the capacity
// limit cannot strand an otherwise valid live grant.
const ScopedPeerHistoryLimit = 8192

func ScopedPeerHistoryCanAppend(used, active int, allow bool) bool {
	if used < 0 || used >= ScopedPeerHistoryLimit || active < 0 || active > ScopedPeerHistoryLimit {
		return false
	}
	after := active - 1
	if allow {
		after = active + 1
	}
	return after >= 0 && used+1+after <= ScopedPeerHistoryLimit
}
