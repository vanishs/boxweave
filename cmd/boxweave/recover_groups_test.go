package main

import (
	"reflect"
	"testing"
)

func TestRecoveryWorkerRestrictedGroups(t *testing.T) {
	for _, tc := range []struct {
		primary, restricted uint32
		want                []uint32
	}{
		{1001, 990, []uint32{1001, 990}},
		{990, 990, []uint32{990}},
	} {
		if got := recoveryGroups(tc.primary, tc.restricted); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("restricted groups: got %v want %v", got, tc.want)
		}
	}
}

func TestScopedGCReviewActionBoundary(t *testing.T) {
	for _, a := range []string{"gc-inspect", "gc-challenge", "gc-release"} {
		if !gcReviewAction(a) {
			t.Fatal(a)
		}
	}
	for _, a := range []string{"inspect", "challenge", "release", "create-bind", "delete-confirm", "reconcile", "diagnose", "gc-other", ""} {
		if gcReviewAction(a) {
			t.Fatal("unexpected GC authorization", a)
		}
	}
}
