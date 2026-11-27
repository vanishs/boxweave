package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func a4ForwardAdmission(a Admission, port int) Admission {
	a.Action = "port_forward_create"
	a.CleanupDependencies = false
	a.Key = fmt.Sprintf("a4-forward-create-original-%d", port)
	a.RequestID = a.Key
	a.ResourceKeys = []string{"box:" + a.BoxUUID, fmt.Sprintf("listener:tcp:127.0.0.1:%d", port)}
	a.ForwardCreate = &ForwardCreateAdmission{Protocol: "tcp", HostBind: "127.0.0.1", HostPort: port, GuestPort: 19091}
	a.SafeIntent, _ = json.Marshal(map[string]any{"box_id": a.BoxUUID, "protocol": "tcp", "host_bind": "127.0.0.1", "host_port": port, "guest_port": 19091, "confirm_external_exposure": false})
	return a
}
func a4Admit(t *testing.T, s *Store, a Admission) Receipt {
	t.Helper()
	return a4AdmitMode(t, s, a, false)
}
func a4AdmitMode(t *testing.T, s *Store, a Admission, scoped bool) Receipt {
	t.Helper()
	admit := s.Admit
	if scoped {
		admit = s.AdmitScoped
	}
	r, e := admit(context.Background(), a)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func a4Unknown(t *testing.T, s *Store, id string, ack bool) {
	t.Helper()
	ctx := context.Background()
	if e := s.Claim(ctx, id); e != nil {
		t.Fatal(e)
	}
	from := "DISPATCHING"
	if ack {
		if e := s.NativeAck(ctx, id, rNative); e != nil {
			t.Fatal(e)
		}
		from = "VERIFYING"
	}
	if e := s.Unknown(ctx, id, from, "A4_TEST_ACK_LOST"); e != nil {
		t.Fatal(e)
	}
}
func a4Request(t *testing.T, s *Store, id string) (RecoveryRequest, RecoveryTarget) {
	t.Helper()
	ctx := context.Background()
	var layer, actor, token string
	if e := s.db.QueryRow("SELECT layer,actor_id,token_id FROM operations WHERE id=?", id).Scan(&layer, &actor, &token); e != nil {
		t.Fatal(e)
	}
	target, e := s.ReadRecoveryTarget(ctx, layer, id)
	if e != nil {
		t.Fatal(e)
	}
	return RecoveryRequest{Layer: layer, ActorID: actor, TokenID: token, OriginalID: id, Key: "a4-recovery-" + id}, target
}
func a4ForwardProof(t RecoveryTarget) RecoveryProof {
	p := recoverProof(t)
	p.Forward = t.Forward
	return p
}
func a4Settle(t *testing.T, s *Store, r RecoveryRequest, target RecoveryTarget, p RecoveryProof) {
	t.Helper()
	a4SettleMode(t, s, r, target, p, false)
}
func a4SettleMode(t *testing.T, s *Store, r RecoveryRequest, target RecoveryTarget, p RecoveryProof, scoped bool) {
	t.Helper()
	ctx := context.Background()
	first, e := recoveryCommitMode(s, scoped)(ctx, r, target, p)
	if e != nil || first.Disposition != "SETTLED" || first.Receipt.ID == target.ID || first.Receipt.NativeOperationID != "" {
		t.Fatalf("settlement %+v %v", first, e)
	}
	dup, e := s.ReplayRecovery(ctx, r)
	if e != nil || dup.Receipt.ID != first.Receipt.ID || !dup.Receipt.Duplicate {
		t.Fatalf("replay %+v %v", dup, e)
	}
	v, e := s.Get(ctx, target.ID)
	if e != nil || v.State != "SUCCEEDED" || v.NativeOperationID != target.NativeID {
		t.Fatalf("original %+v %v", v, e)
	}
	var n int
	for _, q := range []string{"SELECT count(*) FROM operation_events WHERE operation_id=? AND state='UNKNOWN'", "SELECT count(*) FROM recovery_audit WHERE operation_id=? AND decision='RECOVERY_SETTLED'"} {
		if e = s.db.QueryRow(q, target.ID).Scan(&n); e != nil || n != 1 {
			t.Fatalf("audit count %d %v", n, e)
		}
	}
}
func TestS5A4ForwardRecoveryExactClaimsAndAtomicReceipt(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	b4OwnedFixture(t, s, "user-1001", "a", s4b2RootSrc)
	parent := b5DeleteAdmission(t, s, s4b2RootSrc, "a")
	a := a4ForwardAdmission(parent, 24105)
	original := a4AdmitMode(t, s, a, true)
	a4Unknown(t, s, original.ID, true)
	r, target := a4Request(t, s, original.ID)
	if !target.ForwardReady {
		t.Fatalf("forward snapshot %+v", target)
	}
	p := a4ForwardProof(target)
	bad := p
	bad.Forward.HostPort++
	if _, e := s.CommitScopedRecovery(ctx, r, target, bad); !errors.Is(e, ErrResourceLocked) {
		t.Fatalf("changed proof accepted %v", e)
	}
	if _, e := s.db.Exec("CREATE TRIGGER a4_fault BEFORE INSERT ON operation_events WHEN NEW.safe_detail LIKE 'SETTLED:%' BEGIN SELECT RAISE(ABORT,'a4 fault'); END"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.CommitScopedRecovery(ctx, r, target, p); e == nil {
		t.Fatal("fault accepted")
	}
	f, e := s.GetForward(ctx, a.Layer, original.ID)
	if e != nil || f.State != "PENDING" {
		t.Fatalf("fault lost claim %+v %v", f, e)
	}
	if _, e = s.db.Exec("DROP TRIGGER a4_fault"); e != nil {
		t.Fatal(e)
	}
	a4SettleMode(t, s, r, target, p, true)
	f, e = s.GetForward(ctx, a.Layer, original.ID)
	if e != nil || f.State != "CONFIRMED" {
		t.Fatalf("forward %+v %v", f, e)
	}
	a.Action = "port_forward_delete"
	a.Key = "a4-forward-delete-original-001"
	a.RequestID = a.Key
	a.ForwardCreate = nil
	a.ForwardDeleteID = original.ID
	a.SafeIntent, _ = json.Marshal(map[string]string{"forward_id": original.ID})
	del := a4AdmitMode(t, s, a, true)
	a4Unknown(t, s, del.ID, true)
	r, target = a4Request(t, s, del.ID)
	if !target.ForwardReady {
		t.Fatal("delete lineage not resolved")
	}
	a4SettleMode(t, s, r, target, a4ForwardProof(target), true)
	f, e = s.GetForward(ctx, a.Layer, original.ID)
	if e != nil || f.State != "REVOKED" {
		t.Fatalf("delete %+v %v", f, e)
	}
}
func TestS5A4ForwardMissingAckOrDriftKeepsListener(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"missing-ack", "changed-port", "missing-claim", "generic-release"} {
		t.Run(kind, func(t *testing.T) {
			s := newStore(t)
			b4OwnedFixture(t, s, "user-1001", "a", s4b2RootSrc)
			a := a4ForwardAdmission(b5DeleteAdmission(t, s, s4b2RootSrc, "a"), 24106)
			original := a4AdmitMode(t, s, a, true)
			a4Unknown(t, s, original.ID, kind != "missing-ack")
			r, target := a4Request(t, s, original.ID)
			p := a4ForwardProof(target)
			switch kind {
			case "changed-port":
				if _, e := s.db.Exec("UPDATE forward_claims SET host_port=24107 WHERE operation_id=?", original.ID); e != nil {
					t.Fatal(e)
				}
			case "missing-claim":
				if _, e := s.db.Exec("DELETE FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'listener:%'", original.ID); e != nil {
					t.Fatal(e)
				}
			case "generic-release":
				snap, e := s.RecoveryInspect(ctx, original.ID)
				if e != nil {
					t.Fatal(e)
				}
				ch, e := s.IssueRecoveryChallenge(ctx, snap, "a4-reviewer", "test evidence")
				if e != nil {
					t.Fatal(e)
				}
				if e = s.ReleaseRecoveryClaim(ctx, snap, "a4-reviewer", ch, a.BoxUUID, "test reason is sufficiently long", "test evidence", true); !errors.Is(e, ErrResourceLocked) {
					t.Fatalf("unsafe release %v", e)
				}
				p.Fingerprint = ""
			}
			if _, e := s.CommitScopedRecovery(ctx, r, target, p); e == nil {
				t.Fatal("unsafe settlement")
			}
			f, e := s.GetForward(ctx, a.Layer, original.ID)
			if e != nil || f.State != "PENDING" {
				t.Fatalf("lost reservation %+v %v", f, e)
			}
			v, e := s.Get(ctx, original.ID)
			if e != nil || v.State != "UNKNOWN" {
				t.Fatal("original unlocked")
			}
		})
	}
}
func TestS5A4DependentDeleteLinkedChildrenAndFinalRecovery(t *testing.T) {
	ctx := context.Background()
	t.Run("linked-child-retains-parent", func(t *testing.T) {
		s := newStore(t)
		b4OwnedFixture(t, s, "user-1001", "a", s4b2RootSrc)
		a := b5DeleteAdmission(t, s, s4b2RootSrc, "a")
		fwd := a4Admit(t, s, a4ForwardAdmission(a, 24108))
		if e := s.Claim(ctx, fwd.ID); e != nil {
			t.Fatal(e)
		}
		if e := s.NativeAck(ctx, fwd.ID, rNative); e != nil {
			t.Fatal(e)
		}
		if e := s.FinalizeVerifiedForward(ctx, fwd.ID, true); e != nil {
			t.Fatal(e)
		}
		parent := a4Admit(t, s, a)
		if e := s.Claim(ctx, parent.ID); e != nil {
			t.Fatal(e)
		}
		child, e := s.NextDeleteChild(ctx, parent.ID, true)
		if e != nil {
			t.Fatal(e)
		}
		a4Unknown(t, s, child.Receipt.ID, true)
		r, target := a4Request(t, s, child.Receipt.ID)
		if _, e = s.CommitRecovery(ctx, r, target, a4ForwardProof(target)); e == nil {
			t.Fatal("child settled while parent running")
		}
		if e = s.Unknown(ctx, parent.ID, "DISPATCHING", "A4_PARENT_UNKNOWN"); e != nil {
			t.Fatal(e)
		}
		a4Settle(t, s, r, target, a4ForwardProof(target))
		p, e := s.Get(ctx, parent.ID)
		if e != nil || p.State != "UNKNOWN" || p.NativeOperationID != "" {
			t.Fatalf("parent replayed or unlocked %+v %v", p, e)
		}
		b5CheckBlocked(t, s, a)
		_, pt := a4Request(t, s, parent.ID)
		if e = s.CheckDependentDeleteRecovery(ctx, pt); e == nil {
			t.Fatal("parent without final native ID recoverable")
		}
	})
	t.Run("final-exact-terminal", func(t *testing.T) {
		s := newStore(t)
		s4b2InsertAuditFixture(t, s)
		a := b5DeleteAdmission(t, s, s4b2RootSrc, "a")
		parent := a4Admit(t, s, a)
		if e := s.Claim(ctx, parent.ID); e != nil {
			t.Fatal(e)
		}
		child, e := s.NextDeleteChild(ctx, parent.ID, true)
		if e != nil || child.Receipt.Action != "box_stop" {
			t.Fatalf("child %+v %v", child, e)
		}
		if e = s.Claim(ctx, child.Receipt.ID); e != nil {
			t.Fatal(e)
		}
		if e = s.NativeAck(ctx, child.Receipt.ID, rNative); e != nil {
			t.Fatal(e)
		}
		if e = s.Verified(ctx, child.Receipt.ID); e != nil {
			t.Fatal(e)
		}
		if e = s.AuthorizeCascadeFinal(ctx, parent.ID); e != nil {
			t.Fatal(e)
		}
		if e = s.NativeAck(ctx, parent.ID, rNative); e != nil {
			t.Fatal(e)
		}
		if e = s.Unknown(ctx, parent.ID, "VERIFYING", "A4_FINAL_ACK_LOST"); e != nil {
			t.Fatal(e)
		}
		r, target := a4Request(t, s, parent.ID)
		if e = s.CheckDependentDeleteRecovery(ctx, target); e != nil {
			t.Fatal(e)
		}
		p := recoverProof(target)
		p.Fingerprint = "DELETE_SUCCESS:test-witness"
		if _, e = s.db.Exec("UPDATE operations SET state='UNKNOWN' WHERE id=?", child.Receipt.ID); e != nil {
			t.Fatal(e)
		}
		if _, e = s.CommitRecovery(ctx, r, target, p); e == nil {
			t.Fatal("unknown child bypassed")
		}
		if _, e = s.db.Exec("UPDATE operations SET state='SUCCEEDED' WHERE id=?", child.Receipt.ID); e != nil {
			t.Fatal(e)
		}
		a4Settle(t, s, r, target, p)
		var n int
		if e = s.db.QueryRow("SELECT count(*) FROM network_address_fences WHERE project=? AND box_uuid=? AND state='RELEASED' AND operation_id=?", a.Layer, a.BoxUUID, parent.ID).Scan(&n); e != nil || n != 2 {
			t.Fatalf("fences %d %v", n, e)
		}
		if _, e = s.OwnedBox(ctx, a.Layer, a.BoxUUID); !errors.Is(e, ErrNotFound) {
			t.Fatal("tombstone missing", e)
		}
	})
}
func a4PeerProof(t RecoveryTarget) RecoveryProof {
	p := recoverProof(t)
	p.Reason = "ROOT_COMPLETED_POSTSTATE_VERIFIED"
	p.Peer = RootPeerResult{Status: "RECONCILE_VERIFIED", PlanSHA256: strings.Repeat("a", 64), PostNativeDigestSHA256: strings.Repeat("b", 64), ProbedScopes: 4, MutationComplete: true, KernelVerified: true, VerifiedNICs: 3}
	return p
}
func TestS5A4PeerRootCompletedGrantRevokeAtomic(t *testing.T) {
	ctx := context.Background()
	for _, grant := range []bool{true, false} {
		t.Run(fmt.Sprint(grant), func(t *testing.T) {
			var s *Store
			var id string
			if grant {
				s, _, id = s4b3UnknownGrantRecoveryFixture(t)
			} else {
				s, _, id = s4b2RootOperationFixture(t)
				s4b2PendingExistingGrantToConfirmed(t, s)
				if e := s.StageRootPeerRevocation(ctx, id); e != nil {
					t.Fatal(e)
				}
				a4Unknown(t, s, id, false)
			}
			r, target := a4Request(t, s, id)
			if !target.PeerReady {
				t.Fatalf("root snapshot %+v", target)
			}
			p := a4PeerProof(target)
			bad := p
			bad.Peer.KernelVerified = false
			if _, e := s.CommitRecovery(ctx, r, target, bad); e == nil {
				t.Fatal("partial root proof accepted")
			}
			bad = p
			bad.Peer.PlanSHA256 = "invalid"
			if _, e := s.CommitRecovery(ctx, r, target, bad); e == nil {
				t.Fatal("invalid digest accepted")
			}
			if _, e := s.db.Exec("CREATE TRIGGER a4_peer_fault BEFORE INSERT ON operation_events WHEN NEW.safe_detail LIKE 'SETTLED:%' BEGIN SELECT RAISE(ABORT,'a4 fault'); END"); e != nil {
				t.Fatal(e)
			}
			if _, e := s.CommitRecovery(ctx, r, target, p); e == nil {
				t.Fatal("peer fault accepted")
			}
			var status string
			if e := s.db.QueryRow("SELECT status FROM peer_grants WHERE operation_id=?", id).Scan(&status); e != nil || status != "PENDING" {
				t.Fatal("root claim lost", e)
			}
			if _, e := s.db.Exec("DROP TRIGGER a4_peer_fault"); e != nil {
				t.Fatal(e)
			}
			a4Settle(t, s, r, target, p)
			want := "REVOKED"
			if grant {
				want = "CONFIRMED"
			}
			if e := s.db.QueryRow("SELECT status FROM peer_grants WHERE last_verified_operation_id=?", id).Scan(&status); e != nil || status != want {
				t.Fatalf("peer %s %v", status, e)
			}
		})
	}
}
