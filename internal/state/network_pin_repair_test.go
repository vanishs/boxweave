package state

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func n3OwnedCreateForRepair(t *testing.T) (*Store, Admission, Receipt, VerifiedCreateBinding, RecoveryTarget) {
	t.Helper()
	ctx := context.Background()
	s := newStore(t)
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	a := Admission{Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID, Key: "n3-pin-repair-original-create", RequestID: "create", Action: "box_create", ResourceKey: "box:alpha", SafeIntent: []byte(`{"name":"alpha","cpu":1,"memory_mib":1024,"disk_gib":10,"start":false,"nesting":false,"internet":true,"instance_options":{}}`)}
	c, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, c.ID); e != nil {
		t.Fatal(e)
	}
	pin := VerifiedCreateBinding{"bwbr0", "bw-peer", "02:16:3e:11:22:33", "10.24.18.11", "fd43:18::11"}
	if ok, e := s.StageCreateBinding(ctx, c.ID, a.Layer, "alpha", pin); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e = s.NativeAck(ctx, c.ID, "native-create-receipt"); e != nil {
		t.Fatal(e)
	}
	const uuid = "11111111-1111-4111-8111-000000000001"
	if e = s.FinalizeVerifiedCreateWithBinding(ctx, c.ID, a.Layer, "alpha", uuid, pin); e != nil {
		t.Fatal(e)
	}
	target, e := s.ReadRecoveryTarget(ctx, a.Layer, c.ID)
	if e != nil || target.State != "SUCCEEDED" || target.OwnedState != "OWNED" || !target.CreatePinStaged {
		t.Fatal(target, e)
	}
	return s, a, c, pin, target
}

func TestN3RepairMissingLedgerAtomicAndReplay(t *testing.T) {
	ctx := context.Background()
	s, a, c, pin, target := n3OwnedCreateForRepair(t)
	st, e := s.CreatePinRepairStatus(ctx, target)
	if e != nil || st != "COMPLETE" {
		t.Fatal(st, e)
	}
	req := RecoveryRequest{Layer: a.Layer, ActorID: a.ActorID, TokenID: a.TokenID, OriginalID: c.ID, Key: "n3-repair-missing-native-pin-001"}
	p := RecoveryProof{Kind: RecoveryRepairPin, Reason: "CREATE_PIN_LEDGER_REPAIRED", NativeID: target.NativeID, UUID: target.UUID, Fingerprint: "independently-verified-test", CreateBinding: pin}
	if _, e = s.CommitRecovery(ctx, req, target, p); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("no-op repair succeeded", e)
	}
	if _, e = s.db.Exec("DELETE FROM network_bindings WHERE box_uuid=?", target.UUID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("DELETE FROM network_address_fences WHERE box_uuid=? AND family='ipv6'", target.UUID); e != nil {
		t.Fatal(e)
	}
	st, e = s.CreatePinRepairStatus(ctx, target)
	if e != nil || st != "MISSING" {
		t.Fatal(st, e)
	}
	if _, e = s.ReadCompleteNetworkAuditLedger(ctx, a.Layer, pin.Bridge, pin.ACL); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("damaged ledger accepted", e)
	}
	bad := p
	bad.CreateBinding.IPv4 = "10.24.18.77"
	if e = s.CheckCreatePinRepair(ctx, target, bad); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("wrong address accepted", e)
	}
	other := a
	other.Key = "n3-repair-unresolved-concurrent"
	other.RequestID = "busy"
	other.ResourceKey = "box:other"
	busy, e := s.Admit(ctx, other)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CommitRecovery(ctx, req, target, p); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("busy original admitted", e)
	}
	if e = s.CancelQueued(ctx, a.Layer, busy.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.CheckCreatePinRepair(ctx, target, p); e != nil {
		t.Fatal(e)
	}
	for _, trigger := range []string{
		"CREATE TRIGGER n3_repair_fault BEFORE INSERT ON network_address_fences WHEN NEW.family='ipv6' BEGIN SELECT RAISE(ABORT,'n3 fault'); END",
		"CREATE TRIGGER n3_repair_fault BEFORE INSERT ON operations WHEN NEW.action='operation_recover' BEGIN SELECT RAISE(ABORT,'n3 receipt fault'); END",
	} {
		if _, e = s.db.Exec(trigger); e != nil {
			t.Fatal(e)
		}
		if _, e = s.CommitRecovery(ctx, req, target, p); e == nil {
			t.Fatal("partial repair committed")
		}
		st, e = s.CreatePinRepairStatus(ctx, target)
		if e != nil || st != "MISSING" {
			t.Fatal(st, e)
		}
		if _, e = s.db.Exec("DROP TRIGGER n3_repair_fault"); e != nil {
			t.Fatal(e)
		}
	}
	first, e := s.CommitRecovery(ctx, req, target, p)
	if e != nil || first.Disposition != "SETTLED" || first.Receipt.State != "SUCCEEDED" || first.Receipt.ID == c.ID {
		t.Fatal(first, e)
	}
	original, e := s.Get(ctx, c.ID)
	if e != nil || original.State != "SUCCEEDED" || original.BoxUUID != target.UUID || original.NativeOperationID != target.NativeID {
		t.Fatal(original, e)
	}
	st, e = s.CreatePinRepairStatus(ctx, target)
	if e != nil || st != "COMPLETE" {
		t.Fatal(st, e)
	}
	led, e := s.ReadCompleteNetworkAuditLedger(ctx, a.Layer, pin.Bridge, pin.ACL)
	if e != nil || len(led.Boxes) != 1 || led.Boxes[0].UUID != target.UUID {
		t.Fatal(led, e)
	}
	replay, e := s.CommitRecovery(ctx, req, target, p)
	if e != nil || !replay.Receipt.Duplicate || replay.Receipt.ID != first.Receipt.ID {
		t.Fatal(replay, e)
	}
}

func TestN3RepairDoesNotChangeConflictingRecords(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []string{"foreign-fence", "wrong-existing-binding", "missing-original-reservation", "duplicate-stale-claim", "wrong-fence-origin"} {
		t.Run(tc, func(t *testing.T) {
			s, a, c, pin, target := n3OwnedCreateForRepair(t)
			if _, e := s.db.Exec("DELETE FROM network_address_fences WHERE box_uuid=? AND family='ipv4'", target.UUID); e != nil {
				t.Fatal(e)
			}
			switch tc {
			case "foreign-fence":
				_, _ = s.db.Exec("INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES(?,'ipv4',?,'foreign','foreign','foreign','RESERVED',1)", pin.Bridge, pin.IPv4)
			case "wrong-existing-binding":
				_, _ = s.db.Exec("UPDATE network_bindings SET mac='02:16:3e:11:22:44' WHERE box_uuid=?", target.UUID)
			case "missing-original-reservation":
				_, _ = s.db.Exec("DELETE FROM operation_resource_claims WHERE operation_id=? AND resource_key=?", c.ID, pin.reservationKeys()[0])
			case "duplicate-stale-claim":
				other := a
				other.Key = "n3-repair-extra-stale-operation"
				other.RequestID = "extra"
				other.ResourceKey = "box:other"
				rec, err := s.Admit(ctx, other)
				if err != nil {
					t.Fatal(err)
				}
				if err = s.CancelQueued(ctx, a.Layer, rec.ID); err != nil {
					t.Fatal(err)
				}
				if _, err = s.db.Exec("INSERT INTO operation_resource_claims(operation_id,layer,resource_key) VALUES(?,?,?)", rec.ID, a.Layer, pin.reservationKeys()[0]); err != nil {
					t.Fatal(err)
				}
			case "wrong-fence-origin":
				_, _ = s.db.Exec("UPDATE network_address_fences SET operation_id='foreign' WHERE box_uuid=? AND family='ipv6'", target.UUID)
			}
			p := RecoveryProof{Kind: RecoveryRepairPin, Reason: "CREATE_PIN_LEDGER_REPAIRED", NativeID: target.NativeID, UUID: target.UUID, Fingerprint: "verified", CreateBinding: pin}
			req := RecoveryRequest{Layer: a.Layer, ActorID: a.ActorID, TokenID: a.TokenID, OriginalID: c.ID, Key: "n3-repair-conflict-test-001"}
			if _, e := s.CommitRecovery(ctx, req, target, p); !errors.Is(e, ErrResourceLocked) {
				t.Fatal("unsafe repair accepted", e)
			}
			var count int
			if e := s.db.QueryRow("SELECT COUNT(*) FROM recovery_audit WHERE operation_id=?", c.ID).Scan(&count); e != nil || count != 0 {
				t.Fatal(count, e)
			}
		})
	}
}

func TestN3ModernNetworkIssuesRecoveryNavigation(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct{ name, mutation, reason, tool string }{
		{"missing binding", "DELETE FROM network_bindings WHERE box_uuid=?", "NETWORK_BINDING_MISSING", "operation_recover"},
		{"missing fence", "DELETE FROM network_address_fences WHERE box_uuid=? AND family='ipv4'", "NETWORK_ADDRESS_FENCE_MISSING", "operation_recover"},
		{"wrong original fence", "UPDATE network_address_fences SET operation_id='different' WHERE box_uuid=? AND family='ipv6'", "NETWORK_ADDRESS_FENCE_MISMATCH", "box_activity_get"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, a, c, _, target := n3OwnedCreateForRepair(t)
			if _, e := s.db.Exec(tc.mutation, target.UUID); e != nil {
				t.Fatal(e)
			}
			q := ActivityQuery{Layer: a.Layer, Requester: a.ActorID, BoxUUID: target.UUID, View: "issues", Limit: 10}
			out, e := s.ReadActivity(ctx, q)
			if e != nil {
				t.Fatal(e)
			}
			found := false
			for _, issue := range out.Issues {
				if issue.Reason != tc.reason {
					continue
				}
				found = true
				if issue.OperationID != c.ID || issue.BoxUUID != target.UUID || issue.NextAction == nil || issue.NextAction.Tool != tc.tool {
					t.Fatal(issue)
				}
				if tc.tool == "operation_recover" && (issue.NextAction.Mode != "inspect" || issue.Disposition != "NATIVE_VERIFICATION_REQUIRED") {
					t.Fatal(issue)
				}
			}
			if !found {
				t.Fatal("missing typed diagnosis", out.Issues)
			}
		})
	}
}

func TestN3ConcurrentPinRepairOnlyOneOriginalWriter(t *testing.T) {
	ctx := context.Background()
	s, a, c, pin, target := n3OwnedCreateForRepair(t)
	if _, e := s.db.Exec("DELETE FROM network_bindings WHERE box_uuid=?", target.UUID); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.Exec("DELETE FROM network_address_fences WHERE box_uuid=? AND family='ipv4'", target.UUID); e != nil {
		t.Fatal(e)
	}
	proof := RecoveryProof{Kind: RecoveryRepairPin, Reason: "CREATE_PIN_LEDGER_REPAIRED", NativeID: target.NativeID, UUID: target.UUID, Fingerprint: "concurrent-independent-observation", CreateBinding: pin}
	type result struct {
		rec RecoveryRecord
		err error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			key := "n3-concurrent-repair-first"
			if i == 1 {
				key = "n3-concurrent-repair-second"
			}
			rec, e := s.CommitRecovery(ctx, RecoveryRequest{Layer: a.Layer, ActorID: a.ActorID, TokenID: a.TokenID, OriginalID: c.ID, Key: key}, target, proof)
			results <- result{rec, e}
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	won, refused := 0, 0
	for v := range results {
		if v.err == nil && v.rec.Disposition == "SETTLED" {
			won++
		} else if errors.Is(v.err, ErrResourceLocked) {
			refused++
		} else {
			t.Fatal(v)
		}
	}
	if won != 1 || refused != 1 {
		t.Fatal("non-atomic repair", won, refused)
	}
	var n int
	if e := s.db.QueryRow("SELECT COUNT(*) FROM recovery_audit WHERE operation_id=? AND decision='RECOVERY_SETTLED'", c.ID).Scan(&n); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	status, e := s.CreatePinRepairStatus(ctx, target)
	if e != nil || status != "COMPLETE" {
		t.Fatal(status, e)
	}
}
