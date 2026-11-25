// Package networklimits defines a single owned-Project live Box bound.
// Raising it requires native audit, allocator and PRS scale acceptance.
package networklimits

const MaxProjectBoxes = 1024

// These are existing scoped audit bounds, not native capacity promises.
const ScopedRetiredIdentityLimit = 8192
const ScopedDirectionLimit = 32

// CapacityLevel is advisory only: use the actual transactional predicate
// for admission. Warn at 80 percent consumed (including reserved headroom).
func CapacityLevel(limit, remaining int) string {
	if limit <= 0 || remaining < 0 || remaining > limit {
		return "UNVERIFIED"
	}
	if remaining == 0 {
		return "EXHAUSTED"
	}
	if remaining*5 <= limit {
		return "LOW"
	}
	return "AVAILABLE"
}
