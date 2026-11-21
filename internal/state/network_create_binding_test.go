package state

import (
	"context"
	"errors"
	"testing"
)

func TestVerifiedCreateBindingAtomic(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	_, box, rec := s4CreateDeleteFixture(t, st)
	b := VerifiedCreateBinding{"bwbr0", "bw-peer", "02:16:3e:11:22:33", "10.243.18.11", "fd43:18::11"}
	if err := st.Claim(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	if ok, e := st.StageCreateBinding(ctx, rec.ID, box.Project, box.Name, b); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if err := st.NativeAck(ctx, rec.ID, "original-native-create"); err != nil {
		t.Fatal(err)
	}
	mustCount := func(q string, want int) {
		t.Helper()
		var n int
		if err := st.db.QueryRow(q).Scan(&n); err != nil || n != want {
			t.Fatalf("count %s got %d want %d: %v", q, n, want, err)
		}
	}
	for _, fault := range []string{
		"CREATE TRIGGER pin_abort BEFORE INSERT ON network_address_fences WHEN NEW.family='ipv6' BEGIN SELECT RAISE(ABORT,'second fence fault'); END",
		"CREATE TRIGGER pin_abort BEFORE UPDATE ON operations WHEN NEW.state='SUCCEEDED' BEGIN SELECT RAISE(ABORT,'terminal fault'); END",
	} {
		if _, err := st.db.Exec(fault); err != nil {
			t.Fatal(err)
		}
		if err := st.FinalizeVerifiedCreateWithBinding(ctx, rec.ID, box.Project, box.Name, box.UUID, b); err == nil {
			t.Fatal("fault must fail")
		}
		mustCount("SELECT COUNT(*) FROM owned_boxes", 0)
		mustCount("SELECT COUNT(*) FROM network_bindings", 0)
		mustCount("SELECT COUNT(*) FROM network_address_fences", 0)
		got, err := st.Get(ctx, rec.ID)
		if err != nil || got.State != "VERIFYING" || got.BoxUUID != "" || got.NativeOperationID != "original-native-create" {
			t.Fatal(got, err)
		}
		if _, err := st.db.Exec("DROP TRIGGER pin_abort"); err != nil {
			t.Fatal(err)
		}
	}
	mismatch := b
	mismatch.IPv4 = "10.243.18.12"
	if err := st.FinalizeVerifiedCreateWithBinding(ctx, rec.ID, box.Project, box.Name, box.UUID, mismatch); err == nil {
		t.Fatal("unreserved binding accepted")
	}
	mustCount("SELECT COUNT(*) FROM owned_boxes", 0)
	mustCount("SELECT COUNT(*) FROM network_bindings", 0)
	if err := st.FinalizeVerifiedCreate(ctx, rec.ID, box.Project, box.Name, box.UUID); err == nil {
		t.Fatal("binding bypass")
	}
	mustCount("SELECT COUNT(*) FROM owned_boxes", 0)
	if err := st.FinalizeVerifiedCreateWithBinding(ctx, rec.ID, box.Project, box.Name, box.UUID, b); err != nil {
		t.Fatal(err)
	}
	got, err := st.Get(ctx, rec.ID)
	if err != nil || got.State != "SUCCEEDED" || got.BoxUUID != box.UUID || got.NativeOperationID != "original-native-create" {
		t.Fatal(got, err)
	}
	ledger, err := st.ReadCompleteNetworkAuditLedger(ctx, box.Project, b.Bridge, b.ACL)
	if err != nil || len(ledger.Boxes) != 1 || ledger.Boxes[0].UUID != box.UUID || ledger.Boxes[0].IPv4 != b.IPv4 || ledger.Boxes[0].IPv6 != b.IPv6 {
		t.Fatal(ledger, err)
	}
	mustCount("SELECT COUNT(*) FROM network_address_fences WHERE state='RESERVED'", 2)
	if err := st.FinalizeVerifiedCreateWithBinding(ctx, rec.ID, box.Project, box.Name, box.UUID, b); !errors.Is(err, ErrState) {
		t.Fatal("duplicate finalization", err)
	}
}
func TestVerifiedCreateBindingRejectsInvalid(t *testing.T) {
	st := newStore(t)
	_, box, rec := s4CreateDeleteFixture(t, st)
	for _, b := range []VerifiedCreateBinding{
		{"bw", "acl", "01:00:5e:00:00:00", "10.1.0.2", "fd01::2"},
		{"bw", "acl", "02:16:3e:11:22:33", "10.1.0.2", ""},
		{"bw", "acl", "02:16:3e:11:22:33", "10.1.0.2", "10.1.0.3"},
	} {
		if err := st.FinalizeVerifiedCreateWithBinding(context.Background(), rec.ID, box.Project, box.Name, box.UUID, b); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid binding accepted", err)
		}
	}
}

func TestCreatePinOriginalCollisionAndUnknown(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	a, box, first := s4CreateDeleteFixture(t, st)
	if err := st.Claim(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	pin := VerifiedCreateBinding{"bwbr0", "bw-peer", "02:16:3e:11:22:33", "10.243.18.11", "fd43:18::11"}
	if ok, e := st.StageCreateBinding(ctx, first.ID, box.Project, box.Name, pin); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if err := st.Unknown(ctx, first.ID, "DISPATCHING", "NATIVE_OUTCOME_AMBIGUOUS"); err != nil {
		t.Fatal(err)
	}
	a.Key = "s5b-new-distinct-create-id"
	a.ResourceKey = "box:another"
	a.SafeIntent = []byte("{\"name\":\"another\"}")
	second, e := st.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = st.Claim(ctx, second.ID); e != nil {
		t.Fatal(e)
	}
	if ok, e := st.StageCreateBinding(ctx, second.ID, box.Project, "another", pin); e != nil || ok {
		t.Fatal("unknown reused", ok, e)
	}
	changed := pin
	changed.IPv4 = "10.243.18.12"
	if ok, e := st.StageCreateBinding(ctx, second.ID, box.Project, "another", changed); e != nil || ok {
		t.Fatal("unknown MAC/v6 reused", ok, e)
	}
	changed.MAC = "02:16:3e:11:22:34"
	changed.IPv6 = "fd43:18::12"
	if ok, e := st.StageCreateBinding(ctx, second.ID, box.Project, "another", changed); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if ok, e := st.StageCreateBinding(ctx, second.ID, box.Project, "another", pin); e == nil || ok {
		t.Fatal("changed staged tuple", ok, e)
	}
	if e = st.NativeAck(ctx, second.ID, "native-other"); e != nil {
		t.Fatal(e)
	}
	if e = st.FinalizeVerifiedCreateWithBinding(ctx, second.ID, box.Project, "another", "uuid-other", changed); e != nil {
		t.Fatal(e)
	}
	got, e := st.Get(ctx, first.ID)
	if e != nil || got.State != "UNKNOWN" || got.BoxUUID != "" {
		t.Fatal(got, e)
	}
	var n int
	if e = st.db.QueryRow("SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'create-pin:%'", first.ID).Scan(&n); e != nil || n != 4 {
		t.Fatal(n, e)
	}
}

func TestCreatePinLifecycleRetainsFences(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	a, box, r := s4CreateDeleteFixture(t, st)
	b := VerifiedCreateBinding{"bwbr0", "bw-peer", "02:16:3e:11:22:33", "10.243.18.11", "fd43:18::11"}
	if e := st.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if ok, e := st.StageCreateBinding(ctx, r.ID, box.Project, box.Name, b); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e := st.NativeAck(ctx, r.ID, "native-create"); e != nil {
		t.Fatal(e)
	}
	if e := st.FinalizeVerifiedCreateWithBinding(ctx, r.ID, box.Project, box.Name, box.UUID, b); e != nil {
		t.Fatal(e)
	}
	a.Action = "box_stop"
	a.Key = "s5b-lifecycle-pinned-stop"
	a.BoxUUID = box.UUID
	a.Incarnation = box.Incarnation
	a.SafeIntent = []byte("{\"box_id\":\"s4-exact-native-uuid\"}")
	stop, e := st.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = st.Claim(ctx, stop.ID); e != nil {
		t.Fatal(e)
	}
	pin, e := st.LifecyclePinnedNIC(ctx, stop.ID, box)
	if e != nil || pin == nil || pin.IPv4 != b.IPv4 {
		t.Fatal(pin, e)
	}
	if e = st.NativeAck(ctx, stop.ID, "native-stop"); e != nil {
		t.Fatal(e)
	}
	if e = st.Verified(ctx, stop.ID); e != nil {
		t.Fatal(e)
	}
	var n int
	if e = st.db.QueryRow("SELECT COUNT(*) FROM network_address_fences WHERE box_uuid=? AND state='RESERVED'", box.UUID).Scan(&n); e != nil || n != 2 {
		t.Fatal(n, e)
	}
	if e = st.QuarantineNetworkAddress(ctx, NetworkAddressFence{Bridge: b.Bridge, Family: "ipv6", Address: b.IPv6, Project: box.Project, BoxUUID: box.UUID, Incarnation: box.Incarnation}); e != nil {
		t.Fatal(e)
	}
	a.Action = "box_start"
	a.Key = "s5b-lifecycle-quarantined-start"
	if _, e = st.Admit(ctx, a); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("quarantine bypass", e)
	}
}
