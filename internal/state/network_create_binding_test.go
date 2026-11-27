package state

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/peerscope"
)

func TestVerifiedCreateBindingAtomic(t *testing.T) {
	ctx := context.Background()
	st, rec, in, b := p1CreateFixture(t, true)
	b.ACL, _ = peerscope.ACLName(in.Project, in.Bridge, p1UUID)
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
		if err := st.FinalizeVerifiedScopedCreate(ctx, rec.ID, in.Project, in.Name, p1UUID, b); err == nil {
			t.Fatal("fault must fail")
		}
		mustCount("SELECT COUNT(*) FROM owned_boxes", 0)
		mustCount("SELECT COUNT(*) FROM network_bindings", 0)
		mustCount("SELECT COUNT(*) FROM network_address_fences", 0)
		got, err := st.Get(ctx, rec.ID)
		if err != nil || got.State != "VERIFYING" || got.BoxUUID != "" || got.NativeOperationID != "fixture-native-create" {
			t.Fatal(got, err)
		}
		if _, err := st.db.Exec("DROP TRIGGER pin_abort"); err != nil {
			t.Fatal(err)
		}
	}
	mismatch := b
	mismatch.IPv4 = "10.243.18.12"
	if err := st.FinalizeVerifiedScopedCreate(ctx, rec.ID, in.Project, in.Name, p1UUID, mismatch); err == nil {
		t.Fatal("unreserved binding accepted")
	}
	mustCount("SELECT COUNT(*) FROM owned_boxes", 0)
	mustCount("SELECT COUNT(*) FROM network_bindings", 0)
	if err := st.FinalizeVerifiedCreate(ctx, rec.ID, in.Project, in.Name, p1UUID); err == nil {
		t.Fatal("binding bypass")
	}
	mustCount("SELECT COUNT(*) FROM owned_boxes", 0)
	if err := st.FinalizeVerifiedScopedCreate(ctx, rec.ID, in.Project, in.Name, p1UUID, b); err != nil {
		t.Fatal(err)
	}
	got, err := st.Get(ctx, rec.ID)
	if err != nil || got.State != "SUCCEEDED" || got.BoxUUID != p1UUID || got.NativeOperationID != "fixture-native-create" {
		t.Fatal(got, err)
	}
	ledger, err := st.ReadCompleteScopedNetworkAuditLedger(ctx, in.Project, b.Bridge)
	if err != nil || len(ledger.Boxes) != 1 || ledger.Boxes[0].UUID != p1UUID || ledger.Boxes[0].IPv4 != b.IPv4 || ledger.Boxes[0].IPv6 != b.IPv6 {
		t.Fatal(ledger, err)
	}
	mustCount("SELECT COUNT(*) FROM network_address_fences WHERE state='RESERVED'", 2)
	if err := st.FinalizeVerifiedScopedCreate(ctx, rec.ID, in.Project, in.Name, p1UUID, b); !errors.Is(err, ErrState) {
		t.Fatal("duplicate finalization", err)
	}
}
func TestVerifiedCreateBindingRejectsInvalid(t *testing.T) {
	st, rec, in, _ := p1CreateFixture(t, true)
	for _, b := range []VerifiedCreateBinding{
		{"bw", "acl", "01:00:5e:00:00:00", "10.1.0.2", "fd01::2"},
		{"bw", "acl", "02:16:3e:11:22:33", "10.1.0.2", ""},
		{"bw", "acl", "02:16:3e:11:22:33", "10.1.0.2", "10.1.0.3"},
	} {
		if err := st.FinalizeVerifiedScopedCreate(context.Background(), rec.ID, in.Project, in.Name, p1UUID, b); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid binding accepted", err)
		}
	}
}

func TestCreatePinOriginalCollisionAndUnknown(t *testing.T) {
	ctx := context.Background()
	st, first, in, pin := p1CreateFixture(t, false)
	if e := st.Unknown(ctx, first.ID, "DISPATCHING", "NATIVE_OUTCOME_AMBIGUOUS"); e != nil {
		t.Fatal(e)
	}
	_, tok, e := st.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	in.Name = "another"
	raw, e := json.Marshal(in)
	if e != nil {
		t.Fatal(e)
	}
	a := Admission{Layer: in.Project, ActorID: tok.ID, TokenID: tok.ID, Key: "s5b-new-distinct-create-id", RequestID: "another", Action: "box_create", ResourceKey: "box:another", SafeIntent: raw}
	if _, e = st.AdmitScoped(ctx, a); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("UNKNOWN allowed sibling admission", e)
	}
	// Deliberately synthetic cross-project common bridge: exercise global
	// SQL reservation collisions separately from the same-project barrier.
	// This is not authorization for a real shared bridge or native dispatch.
	in.Project = "user-2002"
	a.Layer = in.Project
	a.SafeIntent, e = json.Marshal(in)
	if e != nil {
		t.Fatal(e)
	}
	second, e := st.AdmitScoped(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = st.Claim(ctx, second.ID); e != nil {
		t.Fatal(e)
	}
	if ok, e := st.StageCreateBinding(ctx, second.ID, in.Project, "another", pin); e != nil || ok {
		t.Fatal("unknown reused", ok, e)
	}
	changed := pin
	changed.IPv4 = "10.243.18.12"
	if ok, e := st.StageCreateBinding(ctx, second.ID, in.Project, "another", changed); e != nil || ok {
		t.Fatal("unknown MAC/v6 reused", ok, e)
	}
	changed.MAC = "02:16:3e:11:22:34"
	changed.IPv6 = "fd43:18::12"
	if ok, e := st.StageCreateBinding(ctx, second.ID, in.Project, "another", changed); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if ok, e := st.StageCreateBinding(ctx, second.ID, in.Project, "another", pin); e == nil || ok {
		t.Fatal("changed staged tuple", ok, e)
	}
	if e = st.NativeAck(ctx, second.ID, "native-other"); e != nil {
		t.Fatal(e)
	}
	const uuid = "11111111-1111-4111-8111-000000000002"
	changed.ACL, e = peerscope.ACLName(in.Project, in.Bridge, uuid)
	if e != nil {
		t.Fatal(e)
	}
	if e = st.FinalizeVerifiedScopedCreate(ctx, second.ID, in.Project, "another", uuid, changed); e != nil {
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
	st, a, _, b, target := n3OwnedCreateForRepair(t)
	box, e := st.OwnedBox(ctx, a.Layer, target.UUID)
	if e != nil {
		t.Fatal(e)
	}
	a.Action = "box_stop"
	a.Key = "s5b-lifecycle-pinned-stop"
	a.BoxUUID = box.UUID
	a.Incarnation = box.Incarnation
	a.SafeIntent = []byte(`{"box_id":"` + box.UUID + `"}`)
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
