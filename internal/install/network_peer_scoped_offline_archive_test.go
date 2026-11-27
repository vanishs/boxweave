package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestP2OfflineNativePrefix(t *testing.T) {
	for _, tc := range []struct {
		states []int
		want   int
		ok     bool
	}{
		{[]int{0, 0}, 0, true}, {[]int{1, 0}, 1, true}, {[]int{2, 1, 0}, 3, true},
		{[]int{2, 2}, 4, true}, {[]int{0, 2}, 0, false}, {[]int{1, 1}, 0, false},
		{[]int{2, 0, 1}, 0, false}, {[]int{-1}, 0, false}, {[]int{3}, 0, false},
	} {
		got, e := scopedOfflinePrefix(tc.states)
		if (e == nil) != tc.ok || tc.ok && got != tc.want {
			t.Fatalf("%v: %d %v", tc.states, got, e)
		}
	}
}

func TestP2OfflineNeverReplayPending(t *testing.T) {
	for _, tc := range []struct {
		c    int
		p    bool
		o, n int
		want string
	}{
		{0, false, 0, 4, "DISPATCH"}, {0, true, 0, 4, ""}, {0, true, 1, 4, "VERIFY_ONLY"},
		{0, true, 2, 4, ""}, {1, false, 0, 4, ""}, {1, false, 2, 4, ""},
		{4, false, 4, 4, "COMPLETE"}, {4, true, 4, 4, ""}, {5, false, 5, 4, ""},
	} {
		got, e := scopedOfflineNext(tc.c, tc.p, tc.o, tc.n)
		if (e == nil) != (tc.want != "") || got != tc.want {
			t.Fatalf("%+v: %q %v", tc, got, e)
		}
	}
}

func TestP2OfflineEffectJournal(t *testing.T) {
	r := ScopedOfflineReview{ReviewSHA256: strings.Repeat("a", 64), Targets: []ScopedOfflineTarget{{UUID: "12345678-1234-1234-1234-123456789012"}}}
	d := t.TempDir()
	check := func(w int, p bool) {
		t.Helper()
		got, pending, e := scopedOfflineJournal(d, r)
		if e != nil || got != w || pending != p {
			t.Fatalf("journal %d %t %v", got, pending, e)
		}
	}
	write := func(i int, status, id string) {
		t.Helper()
		if e := scopedOfflineReceiptWrite(d, scopedOfflineReceipt(r, i, status, id)); e != nil {
			t.Fatal(e)
		}
	}
	check(0, false)
	write(0, "INTENT", "")
	check(0, true)
	if e := scopedOfflineReceiptWrite(d, scopedOfflineReceipt(r, 0, "INTENT", "")); e == nil {
		t.Fatal("intent overwritten")
	}
	write(0, "ACK", "12345678-1234-1234-1234-123456789013")
	check(0, true)
	write(0, "VERIFIED", "")
	check(1, false)
	write(1, "INTENT", "")
	check(1, true)
	write(1, "VERIFIED", "")
	check(2, false)
	name := filepath.Join(d, scopedOfflineReceiptName(1, "VERIFIED"))
	if e := os.WriteFile(name, []byte("{}"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, _, e := scopedOfflineJournal(d, r); e == nil {
		t.Fatal("tampered receipt accepted")
	}
}

func TestP2OfflineEffectJournalRejectsGap(t *testing.T) {
	r := ScopedOfflineReview{ReviewSHA256: strings.Repeat("a", 64), Targets: []ScopedOfflineTarget{{UUID: "12345678-1234-1234-1234-123456789012"}}}
	d := t.TempDir()
	if e := scopedOfflineReceiptWrite(d, scopedOfflineReceipt(r, 1, "INTENT", "")); e != nil {
		t.Fatal(e)
	}
	if _, _, e := scopedOfflineJournal(d, r); e == nil {
		t.Fatal("unrecorded prefix accepted")
	}
}
