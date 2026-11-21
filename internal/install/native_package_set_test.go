package install

import (
	"slices"
	"testing"
)

func TestNativePackageSetFromMetadata(t *testing.T) {
	v := "1:7.6-ubuntu24.04-202611071352"
	header := "Package: incus\nVersion: " + v + "\n"
	cases := []struct {
		raw  string
		want []string
	}{
		{header + "Depends: libfoo, incus-base (= " + v + "), libbar\n", []string{"incus", "incus-client", "incus-base"}},
		{header + "Depends: libfoo, libbar\n", []string{"incus", "incus-client"}},
		{header + "Depends: incus-base (= wrong)\n", nil},
		{header + "Depends: incus-base | other\n", nil},
		{header + "Depends: incus-base (= " + v + "), incus-base (= " + v + ")\n", nil},
		{header + "Depends: libfoo\n\n" + header + "Depends: incus-base (= " + v + ")\n", nil},
		{"Package: incus\nVersion: wrong\n", nil},
	}
	for i, c := range cases {
		got, err := nativePackageNamesFromMetadata(c.raw, v)
		if c.want == nil {
			if err == nil {
				t.Fatalf("case %d unexpectedly accepted %v", i, got)
			}
		} else if err != nil || !slices.Equal(got, c.want) {
			t.Fatalf("case %d got %v err %v", i, got, err)
		}
	}
}
