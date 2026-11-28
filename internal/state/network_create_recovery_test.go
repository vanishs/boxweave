package state

import (
	"context"
	"errors"
	"testing"

	"github.com/vanishs/boxweave/internal/peerscope"
)

func TestN3StagedCreateRecoveryAtomic(t *testing.T) {
	ctx := context.Background()
	s, original, in, b := p1CreateFixture(t, true)
	if e := s.Unknown(ctx, original.ID, "VERIFYING", "CREATE_TERMINAL_WRITE_UNCERTAIN"); e != nil {
		t.Fatal(e)
	}
	req, target := a4Request(t, s, original.ID)
	if !target.CreatePinRequired || !target.CreatePinStaged {
		t.Fatal(target)
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	exact, e := readScopedCreateOriginalTxPhase(ctx, tx, original.ID, in.Project, in.Bridge, "UNKNOWN")
	tx.Rollback()
	if e != nil {
		t.Fatal(e)
	}
	const uuid = p1UUID
	b.ACL, _ = peerscope.ACLName(in.Project, in.Bridge, uuid)
	proof := p1CompletedProof(t, target, "CREATE", b, ScopedLifecycleOriginalSHA256(exact))
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
	if _, e := s.CommitScopedRecovery(ctx, req, target, bind); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("identity-only accepted", e)
	}
	checkEmpty()
	bad := proof
	bad.CreateBinding.IPv4 = "10.243.18.12"
	if e := s.CheckScopedLifecycleRecovery(ctx, target, bad); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("wrong pin", e)
	}
	if _, e := s.CommitScopedRecovery(ctx, req, target, bad); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("wrong pin commit", e)
	}
	checkEmpty()
	if e := s.CheckScopedLifecycleRecovery(ctx, target, proof); e != nil {
		t.Fatal(e)
	}
	for _, fault := range []string{
		"CREATE TRIGGER abort_n3 BEFORE INSERT ON network_address_fences WHEN NEW.family='ipv6' BEGIN SELECT RAISE(ABORT,'fence fault'); END",
		"CREATE TRIGGER abort_n3 BEFORE INSERT ON operations WHEN NEW.action='operation_recover' BEGIN SELECT RAISE(ABORT,'receipt fault'); END",
	} {
		if _, e := s.db.Exec(fault); e != nil {
			t.Fatal(e)
		}
		if _, e := s.CommitScopedRecovery(ctx, req, target, proof); e == nil {
			t.Fatal("partial commit accepted")
		}
		checkEmpty()
		if _, e := s.db.Exec("DROP TRIGGER abort_n3"); e != nil {
			t.Fatal(e)
		}
	}
	first, e := s.CommitScopedRecovery(ctx, req, target, proof)
	if e != nil || first.Disposition != "SETTLED" || first.Receipt.State != "SUCCEEDED" || first.Receipt.BoxUUID != uuid || first.Receipt.ID == original.ID {
		t.Fatal(first, e)
	}
	got, e := s.Get(ctx, original.ID)
	if e != nil || got.State != "SUCCEEDED" || got.BoxUUID != uuid || got.NativeOperationID != target.NativeID {
		t.Fatal(got, e)
	}
	ledger, e := s.ReadCompleteScopedNetworkAuditLedger(ctx, in.Project, b.Bridge)
	if e != nil || len(ledger.Boxes) != 1 || ledger.Boxes[0].UUID != uuid {
		t.Fatal(ledger, e)
	}
	replay, e := s.CommitScopedRecovery(ctx, req, target, proof)
	if e != nil || !replay.Receipt.Duplicate || replay.Receipt.ID != first.Receipt.ID {
		t.Fatal(replay, e)
	}
}

func TestN3LostCreateACKRetirementLineage(t *testing.T) {
	ctx := context.Background()
	s, orig, in, pin := p1CreateFixture(t, false)
	if e := s.Unknown(ctx, orig.ID, "DISPATCHING", "ACK_LOST"); e != nil {
		t.Fatal(e)
	}
	req, target := a4Request(t, s, orig.ID)
	pin.ACL, _ = peerscope.ACLName(in.Project, in.Bridge, p1UUID)
	legacy := RecoveryProof{Kind: RecoverySettle, Reason: "CREATE_ACK_LOST_STATIC_POSTSTATE_VERIFIED", UUID: p1UUID, Fingerprint: "synthetic", CreateBinding: pin}
	for _, proof := range []RecoveryProof{legacy, p1CompletedProof(t, target, "CREATE", pin, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")} {
		if _, e := s.CommitScopedRecovery(ctx, req, target, proof); e == nil {
			t.Fatal("missing native ACK settled")
		}
	}
	p1AssertOriginal(t, s, orig.ID, "UNKNOWN", "")
	if _, e := s.OwnedBox(ctx, in.Project, p1UUID); !errors.Is(e, ErrNotFound) {
		t.Fatal("invented owner", e)
	}
	var n int
	if e := s.db.QueryRow("SELECT count(*) FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'create-pin:%'", orig.ID).Scan(&n); e != nil || n != 4 {
		t.Fatal("lost original claims", n, e)
	}
	if e := s.db.QueryRow("SELECT count(*) FROM recovery_audit WHERE operation_id=?", orig.ID).Scan(&n); e != nil || n != 0 {
		t.Fatal("invented settlement", n, e)
	}
}
