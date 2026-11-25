package install

import "testing"

func TestScopedRetirementLXCPolicyExact(t *testing.T) {
	p := ScopedRetirementPolicy{knownLXC255PolicyPath, knownLXC255PolicySHA, 0644, "EXACT_LXC255_SCOPED_RETIREMENT"}
	if !knownScopedRetirementPolicy(p) {
		t.Fatal(p)
	}
	for _, q := range []ScopedRetirementPolicy{
		{p.Path + " /tmp/extra", p.SHA, p.Mode, p.Kind}, {p.Path, "bad", p.Mode, p.Kind}, {p.Path, p.SHA, 0666, p.Kind}, {p.Path, p.SHA, p.Mode, "runtime"},
	} {
		if knownScopedRetirementPolicy(q) {
			t.Fatal(q)
		}
	}
	for _, v := range []string{"", "0 0 4294967295", "1 1000 1", "0 1000 0", "0 bad 1"} {
		if mappedRoot(v) {
			t.Fatal(v)
		}
	}
	if !mappedRoot("0 1000000 1001\n1001 1001 1\n1002 1001002 999998998") {
		t.Fatal("mapped root")
	}
}
