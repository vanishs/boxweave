package networklimits

import "testing"

func TestScopedPeerHistoryReserveRevocation(t *testing.T) {
	l := ScopedPeerHistoryLimit
	for _, x := range []struct {
		used, active int
		allow, want  bool
	}{
		{0, 0, true, true}, {0, 0, false, false},
		{l - 2, 0, true, true}, {l - 1, 0, true, false},
		{l - 1, 1, false, true}, {l - 1, 1, true, false},
		{l - 2, 2, false, true}, {l - 3, 2, true, false},
		{l, 0, true, false}, {l, 1, false, false},
		{-1, 0, true, false}, {1, -1, true, false}, {0, 1, true, true},
	} {
		if got := ScopedPeerHistoryCanAppend(x.used, x.active, x.allow); got != x.want {
			t.Fatal(x, got)
		}
	}
	// Simulate every reachable capacity/active-count boundary. Every
	// admitted grant must retain room for all subsequent revocations.
	for used := 0; used < l; used++ {
		for active := 0; active <= 32 && active <= used; active++ {
			if ScopedPeerHistoryCanAppend(used, active, true) && used+1+active+1 > l {
				t.Fatal("grant stranded revoke", used, active)
			}
			if active > 0 && used+active <= l && !ScopedPeerHistoryCanAppend(used, active, false) {
				t.Fatal("reserved revoke refused", used, active)
			}
		}
	}
}
