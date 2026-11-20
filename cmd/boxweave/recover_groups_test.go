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
