package state

import (
	"context"
	"errors"
	"testing"
)

func TestN3StagedCreateRecoveryAtomic(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	a := Admission{Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID, Key: "n3-create-recovery-original", RequestID: "n3-original", Action: "box_create", ResourceKey: "box:alpha", SafeIntent: []byte(`{"name":"alpha","cpu":1,"memory_mib":1024,"disk_gib":10,"start":false,"nesting":false,"internet":true,"instance_options":{}}`)}
	original, err := s.Admit(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Claim(ctx, original.ID); err != nil {
		t.Fatal(err)
	}
	b := VerifiedCreateBinding{"bwbr0", "bw-peer", "02:16:3e:11:22:33", "10.243.18.11", "fd43:18::11"}
	if ok, e := s.StageCreateBinding(ctx, original.ID, a.Layer, "alpha", b); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if err = s.NativeAck(ctx, original.ID, "original-native-create"); err != nil {
		t.Fatal(err)
	}
	if err = s.Unknown(ctx, original.ID, "VERIFYING", "CREATE_TERMINAL_WRITE_UNCERTAIN"); err != nil {
		t.Fatal(err)
	}
	target, err := s.ReadRecoveryTarget(ctx, a.Layer, original.ID)
	if err != nil || !target.CreatePinRequired || !target.CreatePinStaged {
		t.Fatal(target, err)
	}
	req := RecoveryRequest{Layer: a.Layer, ActorID: tok.ID, TokenID: tok.ID, OriginalID: original.ID, Key: "n3-original-create-reconcile"}
	const uuid = "11111111-1111-4111-8111-000000000001"
	proof := RecoveryProof{Kind: RecoverySettle, Reason: "CREATE_ORIGINAL_POSTSTATE_VERIFIED", NativeID: target.NativeID, UUID: uuid, Fingerprint: "verified", CreateBinding: b}
	checkEmpty := func() {
		t.Helper()
		for _, table := range []string{"owned_boxes", "network_bindings", "network_address_fences", "recovery_audit"} {
			var n int
			if e := s.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); e != nil || n != 0 {
				t.Fatal(table, n, e)
			}
		}
		got, e := s.Get(ctx, original.ID)
		if e != nil || got.State != "UNKNOWN" || got.BoxUUID != "" {
			t.Fatal(got, e)
		}
	}
	bind := RecoveryProof{Kind: RecoveryBind, Reason: "CREATE_MARKER_EXACT_IDENTITY", UUID: uuid, Fingerprint: "marker"}
	if _, e := s.CommitRecovery(ctx, req, target, bind); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("identity-only accepted", e)
	}
	checkEmpty()
	bad := proof
	bad.CreateBinding.IPv4 = "10.243.18.12"
	if e := s.CheckRecoveryCreateBinding(ctx, target, bad); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("wrong pin", e)
	}
	if _, e := s.CommitRecovery(ctx, req, target, bad); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("wrong pin commit", e)
	}
	checkEmpty()
	if e := s.CheckRecoveryCreateBinding(ctx, target, proof); e != nil {
		t.Fatal(e)
	}
	for _, fault := range []string{
		"CREATE TRIGGER abort_n3 BEFORE INSERT ON network_address_fences WHEN NEW.family='ipv6' BEGIN SELECT RAISE(ABORT,'fence fault'); END",
		"CREATE TRIGGER abort_n3 BEFORE INSERT ON operations WHEN NEW.action='operation_recover' BEGIN SELECT RAISE(ABORT,'receipt fault'); END",
	} {
		if _, e := s.db.Exec(fault); e != nil {
			t.Fatal(e)
		}
		if _, e := s.CommitRecovery(ctx, req, target, proof); e == nil {
			t.Fatal("partial commit accepted")
		}
		checkEmpty()
		if _, e := s.db.Exec("DROP TRIGGER abort_n3"); e != nil {
			t.Fatal(e)
		}
	}
	first, e := s.CommitRecovery(ctx, req, target, proof)
	if e != nil || first.Disposition != "SETTLED" || first.Receipt.State != "SUCCEEDED" || first.Receipt.BoxUUID != uuid || first.Receipt.ID == original.ID {
		t.Fatal(first, e)
	}
	got, e := s.Get(ctx, original.ID)
	if e != nil || got.State != "SUCCEEDED" || got.BoxUUID != uuid || got.NativeOperationID != target.NativeID {
		t.Fatal(got, e)
	}
	ledger, e := s.ReadCompleteNetworkAuditLedger(ctx, a.Layer, b.Bridge, b.ACL)
	if e != nil || len(ledger.Boxes) != 1 || ledger.Boxes[0].UUID != uuid {
		t.Fatal(ledger, e)
	}
	replay, e := s.CommitRecovery(ctx, req, target, proof)
	if e != nil || !replay.Receipt.Duplicate || replay.Receipt.ID != first.Receipt.ID {
		t.Fatal(replay, e)
	}
}

func TestN3LostCreateACKRetirementLineage(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	a := Admission{Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID, Key: "n3-ack-retire-create-original", RequestID: "create", Action: "box_create", ResourceKey: "box:alpha", SafeIntent: []byte(`{"name":"alpha","cpu":1,"memory_mib":1024,"disk_gib":10,"start":false,"nesting":false,"internet":true,"instance_options":{}}`)}
	c, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	pin := VerifiedCreateBinding{"bwbr0", "bw-peer", "02:16:3e:11:22:33", "10.24.18.11", "fd43:18::11"}
	if e = s.Claim(ctx, c.ID); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.StageCreateBinding(ctx, c.ID, a.Layer, "alpha", pin); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e = s.Unknown(ctx, c.ID, "DISPATCHING", "NO_NATIVE_OPERATION_RECEIPT"); e != nil {
		t.Fatal(e)
	}
	target, e := s.ReadRecoveryTarget(ctx, a.Layer, c.ID)
	if e != nil {
		t.Fatal(e)
	}
	const uuid = "11111111-1111-4111-8111-000000000001"
	p := RecoveryProof{Kind: RecoverySettle, Reason: "CREATE_ACK_LOST_STATIC_POSTSTATE_VERIFIED", UUID: uuid, Fingerprint: "verified", CreateBinding: pin}
	req := RecoveryRequest{Layer: a.Layer, ActorID: tok.ID, TokenID: tok.ID, OriginalID: c.ID, Key: "n3-ack-retire-reconcile"}
	bad := p
	bad.NativeID = "fabricated"
	if _, e = s.CommitRecovery(ctx, req, target, bad); !errors.Is(e, ErrInvalid) {
		t.Fatal("fabricated receipt", e)
	}
	bad = p
	bad.Reason = "CREATE_ORIGINAL_POSTSTATE_VERIFIED"
	if _, e = s.CommitRecovery(ctx, req, target, bad); !errors.Is(e, ErrInvalid) {
		t.Fatal("wrong proof type", e)
	}
	rr, e := s.CommitRecovery(ctx, req, target, p)
	if e != nil || rr.Disposition != "SETTLED" {
		t.Fatal(rr, e)
	}
	d := a
	d.Action = "box_delete"
	d.Key = "n3-ack-retire-delete-original"
	d.RequestID = "delete"
	d.BoxUUID = uuid
	d.Incarnation = uuid
	d.CleanupDependencies = true
	dr, e := s.Admit(ctx, d)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, dr.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.AuthorizeCascadeFinal(ctx, dr.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, dr.ID, "exact-native-delete"); e != nil {
		t.Fatal(e)
	}
	box := OwnedBox{UUID: uuid, Incarnation: uuid, Name: "alpha", Project: a.Layer, State: "OWNED"}
	if e = s.FinalizeVerifiedDelete(ctx, dr.ID, box); e != nil {
		t.Fatal(e)
	}
	for _, cut := range []struct {
		change, restore string
		args            []any
	}{
		{"UPDATE operation_events SET safe_detail='DAMAGED' WHERE operation_id=? AND safe_detail='RECOVERY_CREATE_ACK_LOST_PIN_ATOMIC_NO_REPLAY'", "UPDATE operation_events SET safe_detail='RECOVERY_CREATE_ACK_LOST_PIN_ATOMIC_NO_REPLAY' WHERE operation_id=? AND safe_detail='DAMAGED'", []any{c.ID}},
		{"UPDATE operations SET resource_key='recovery:other' WHERE id=?", "UPDATE operations SET resource_key='recovery:" + c.ID + "' WHERE id=?", []any{rr.Receipt.ID}},
		{"UPDATE recovery_audit SET decision='RELEASED' WHERE operation_id=?", "UPDATE recovery_audit SET decision='RECOVERY_SETTLED' WHERE operation_id=?", []any{c.ID}},
	} {
		if _, e = s.db.Exec(cut.change, cut.args...); e != nil {
			t.Fatal(e)
		}
		if _, e = s.ReadIPv4RetirementCandidate(ctx, a.Layer, dr.ID); !errors.Is(e, ErrResourceLocked) {
			t.Fatal("incomplete recovery lineage accepted", e)
		}
		if _, e = s.db.Exec(cut.restore, cut.args...); e != nil {
			t.Fatal(e)
		}
	}
	v, e := s.ReadIPv4RetirementCandidate(ctx, a.Layer, dr.ID)
	if e != nil || v.CreateOperationID != c.ID {
		t.Fatal(v, e)
	}
	if e = retireWithProof(s, ctx, v, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ReadCreateAddressHolds(ctx, a.Layer, pin.Bridge); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("UPDATE recovery_audit SET decision='RELEASED' WHERE operation_id=?", c.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ReadCreateAddressHolds(ctx, a.Layer, pin.Bridge); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("damaged archival lineage accepted", e)
	}
}
