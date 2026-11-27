package state

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/vanishs/boxweave/internal/peerscope"
)

func TestN3LifecycleRecoveryPin(t *testing.T) {
	t.Run("legacy", func(t *testing.T) { lifecycleRecoveryPinMode(t, false) })
	t.Run("scoped", func(t *testing.T) { lifecycleRecoveryPinMode(t, true) })
}

func lifecycleRecoveryPinMode(t *testing.T, scoped bool) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	a := Admission{Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID, Key: "n3-lifecycle-pin-original-create", RequestID: "create", Action: "box_create", ResourceKey: "box:alpha", SafeIntent: []byte(`{"name":"alpha","cpu":1,"memory_mib":1024,"disk_gib":10,"start":false,"nesting":false,"internet":true,"instance_options":{}}`)}
	admit, commit := s.Admit, s.CommitRecovery
	if scoped {
		admit, commit = s.AdmitScoped, s.CommitScopedRecovery
		raw, e := json.Marshal(ScopedCreateIntent{Schema: 2, Project: a.Layer, Bridge: "incusbr-1001", Name: "alpha", CPU: 1, MemoryMiB: 1024, DiskGiB: 10, Internet: true, BootAutostart: true, Options: json.RawMessage("{}")})
		if e != nil {
			t.Fatal(e)
		}
		a.SafeIntent = raw
	}
	c, e := admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, c.ID); e != nil {
		t.Fatal(e)
	}
	pin := VerifiedCreateBinding{"bwbr0", "bw-peer", "02:16:3e:11:22:33", "10.24.18.11", "fd43:18::11"}
	if scoped {
		pin.Bridge, pin.ACL, pin.IPv4 = "incusbr-1001", peerscope.QuarantineACL, "10.243.18.11"
	}
	if ok, e := s.StageCreateBinding(ctx, c.ID, a.Layer, "alpha", pin); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e = s.NativeAck(ctx, c.ID, "native-create"); e != nil {
		t.Fatal(e)
	}
	if scoped {
		pin.ACL, e = peerscope.ACLName(a.Layer, pin.Bridge, rUUID)
		if e != nil {
			t.Fatal(e)
		}
		if e = s.FinalizeVerifiedScopedCreate(ctx, c.ID, a.Layer, "alpha", rUUID, pin); e != nil {
			t.Fatal(e)
		}
	} else if e = s.FinalizeVerifiedCreateWithBinding(ctx, c.ID, a.Layer, "alpha", rUUID, pin); e != nil {
		t.Fatal(e)
	}
	a.Action = "box_start"
	a.Key = "n3-lifecycle-pin-start-original"
	a.BoxUUID = rUUID
	a.Incarnation = rUUID
	a.RequestID = "start"
	a.SafeIntent = []byte(`{"box_id":"` + rUUID + `"}`)
	op, e := admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, op.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, op.ID, rNative); e != nil {
		t.Fatal(e)
	}
	if e = s.Unknown(ctx, op.ID, "VERIFYING", "TEST_TERMINAL_CUT"); e != nil {
		t.Fatal(e)
	}
	target, e := s.ReadRecoveryTarget(ctx, a.Layer, op.ID)
	if e != nil || target.LifecyclePinStatus != "PINNED" || target.LifecycleBinding != pin {
		t.Fatal(target, e)
	}
	req := RecoveryRequest{Layer: a.Layer, ActorID: tok.ID, TokenID: tok.ID, OriginalID: op.ID, Key: "n3-lifecycle-pin-reconcile"}
	p := recoverProof(target)
	if _, e = commit(ctx, req, target, p); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("missing pin witness", e)
	}
	p.LifecycleBinding = pin

	if _, e = s.db.Exec("DELETE FROM network_address_fences WHERE box_uuid=? AND family='ipv4'", rUUID); e != nil {
		t.Fatal(e)
	}
	missing, e := s.ReadRecoveryTarget(ctx, a.Layer, op.ID)
	if e != nil || missing.LifecyclePinStatus != "MISSING" {
		t.Fatal(missing, e)
	}
	if _, e = s.db.Exec("INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,operation_id,observed_at) VALUES(?,'ipv4',?,?,?,?,'RESERVED',?,1)", pin.Bridge, pin.IPv4, a.Layer, rUUID, rUUID, c.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("UPDATE network_address_fences SET state='QUARANTINED' WHERE box_uuid=? AND family='ipv4'", rUUID); e != nil {
		t.Fatal(e)
	}
	if _, e = commit(ctx, req, target, p); !errors.Is(e, ErrConflict) {
		t.Fatal("stale ledger witness accepted", e)
	}
	changed, e := s.ReadRecoveryTarget(ctx, a.Layer, op.ID)
	if e != nil || changed.LifecyclePinStatus != "CONFLICT" {
		t.Fatal(changed, e)
	}
	if _, e = s.db.Exec("UPDATE network_address_fences SET state='RESERVED' WHERE box_uuid=? AND family='ipv4'", rUUID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("UPDATE network_address_fences SET operation_id='wrong' WHERE box_uuid=? AND family='ipv6'", rUUID); e != nil {
		t.Fatal(e)
	}
	changed, e = s.ReadRecoveryTarget(ctx, a.Layer, op.ID)
	if e != nil || changed.LifecyclePinStatus != "CONFLICT" {
		t.Fatal(changed, e)
	}
	if _, e = s.db.Exec("UPDATE network_address_fences SET operation_id=? WHERE box_uuid=? AND family='ipv6'", c.ID, rUUID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("DELETE FROM operation_resource_claims WHERE operation_id=? AND resource_key=?", c.ID, pin.reservationKeys()[0]); e != nil {
		t.Fatal(e)
	}
	changed, e = s.ReadRecoveryTarget(ctx, a.Layer, op.ID)
	if e != nil || changed.LifecyclePinStatus != "CONFLICT" {
		t.Fatal(changed, e)
	}
	if _, e = s.db.Exec("INSERT INTO operation_resource_claims(operation_id,layer,resource_key) VALUES(?,?,?)", c.ID, a.Layer, pin.reservationKeys()[0]); e != nil {
		t.Fatal(e)
	}
	rr, e := commit(ctx, req, target, p)
	if e != nil || rr.Disposition != "SETTLED" {
		t.Fatal(rr, e)
	}
}
