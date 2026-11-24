package peerscope

import "testing"

func TestPR178ScopedACLNameStableAndUnique(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	first, e := ACLName("user-1001", "incusbr-1001", id)
	if e != nil || len(first) > 63 {
		t.Fatal("invalid bounded peer ACL", e)
	}
	second, e := ACLName("user-1001", "incusbr-1001", id)
	if e != nil || first != second {
		t.Fatal("non-deterministic per-Box ACL")
	}
	for _, tuple := range []struct{ p, b, id string }{
		{"user-1002", "incusbr-1001", id},
		{"user-1001", "incusbr-1002", id},
		{"user-1001", "incusbr-1001", "22222222-2222-4222-8222-222222222222"},
	} {
		other, e := ACLName(tuple.p, tuple.b, tuple.id)
		if e != nil || other == first {
			t.Fatal("different Box/Project/bridge reused ACL name")
		}
	}
	for _, tuple := range []struct{ p, b, id string }{
		{"default", "incusbr-1001", id},
		{"../../root", "incusbr-1001", id},
		{"user-1001", "../network", id},
		{"user-1001", "incusbr-1001", "untrusted"},
		{"user-1001", "incusbr-1001", "11111111-1111-4111-8111-11111111111A"},
	} {
		if _, e := ACLName(tuple.p, tuple.b, tuple.id); e == nil {
			t.Fatal("unsafe per-Box scope accepted")
		}
	}
}
