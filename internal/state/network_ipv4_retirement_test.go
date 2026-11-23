package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"
)

func retireFixture(t *testing.T, s *Store, beforeDelete ...func(Admission, OwnedBox)) (Admission, OwnedBox, VerifiedCreateBinding, IPv4RetirementCandidate) {
	t.Helper()
	ctx := context.Background()
	a, b, c := s4CreateDeleteFixture(t, s)
	pin := VerifiedCreateBinding{"bwbr0", "bw-peer", "02:16:3e:11:22:33", "10.24.18.11", "fd43:18::11"}
	if e := s.Claim(ctx, c.ID); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.StageCreateBinding(ctx, c.ID, b.Project, b.Name, pin); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e := s.NativeAck(ctx, c.ID, "original-native-create"); e != nil {
		t.Fatal(e)
	}
	if e := s.FinalizeVerifiedCreateWithBinding(ctx, c.ID, b.Project, b.Name, b.UUID, pin); e != nil {
		t.Fatal(e)
	}
	for _, fn := range beforeDelete {
		fn(a, b)
	}
	d := a
	d.Action = "box_delete"
	d.Key = "n2-retirement-original-delete"
	d.RequestID = "delete"
	d.BoxUUID = b.UUID
	d.Incarnation = b.Incarnation
	d.CleanupDependencies = true
	r, e := s.Admit(ctx, d)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.AuthorizeCascadeFinal(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, r.ID, "original-native-delete"); e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeVerifiedDelete(ctx, r.ID, b); e != nil {
		t.Fatal(e)
	}
	v, e := s.ReadIPv4RetirementCandidate(ctx, b.Project, r.ID)
	if e != nil || v.CreateOperationID != c.ID || v.Binding.UUID != b.UUID || v.Binding.IPv4 != pin.IPv4 {
		t.Fatal(v, e)
	}
	return a, b, pin, v
}

func TestIPv4RetirementAtomicArchiveReuse(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	a, b, pin, v := retireFixture(t, s)
	proof := strings.Repeat("a", 64) // synthetic state-only proof, not native evidence
	if got, e := s.ReadCreateAddressHolds(ctx, b.Project, pin.Bridge); e != nil || !got.IPs[netip.MustParseAddr(pin.IPv4)] {
		t.Fatal(got, e)
	}
	if _, e := s.db.Exec("CREATE TRIGGER retirement_abort BEFORE DELETE ON network_address_fences WHEN OLD.family='ipv6' BEGIN SELECT RAISE(ABORT,'retirement fault'); END"); e != nil {
		t.Fatal(e)
	}
	if e := retireWithProof(s, ctx, v, proof); e == nil {
		t.Fatal("split retirement accepted")
	}
	var n int
	if e := s.db.QueryRow("SELECT COUNT(*) FROM network_ipv4_retirements").Scan(&n); e != nil || n != 0 {
		t.Fatal(n, e)
	}
	if e := s.db.QueryRow("SELECT COUNT(*) FROM network_address_fences WHERE box_uuid=?", b.UUID).Scan(&n); e != nil || n != 2 {
		t.Fatal(n, e)
	}
	if _, e := s.db.Exec("DROP TRIGGER retirement_abort"); e != nil {
		t.Fatal(e)
	}
	if e := retireWithProof(s, ctx, v, proof); e != nil {
		t.Fatal(e)
	}
	if e := retireWithProof(s, ctx, v, proof); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("second retirement", e)
	}
	if got, e := s.ReadCreateAddressHolds(ctx, b.Project, pin.Bridge); e != nil || got.IPs[netip.MustParseAddr(pin.IPv4)] {
		t.Fatal(got, e)
	}
	a.Key = "n2-replacement-create-different-uuid"
	a.ResourceKey = "box:replacement"
	r, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.StageCreateBinding(ctx, r.ID, b.Project, "replacement", pin); e != nil || ok {
		t.Fatal("old MAC/v6 reused", ok, e)
	}
	pin.MAC = "02:16:3e:11:22:44"
	pin.IPv6 = "fd43:18::44"
	if ok, e := s.StageCreateBinding(ctx, r.ID, b.Project, "replacement", pin); e != nil || !ok {
		t.Fatal("verified archived IPv4 refused", ok, e)
	}
	if e = s.NativeAck(ctx, r.ID, "new-native-create"); e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeVerifiedCreateWithBinding(ctx, r.ID, b.Project, "replacement", "new-native-uuid", pin); e != nil {
		t.Fatal(e)
	}
	led, e := s.ReadCompleteNetworkAuditLedger(ctx, b.Project, pin.Bridge, pin.ACL)
	if e != nil || len(led.Boxes) != 1 || led.Boxes[0].UUID != "new-native-uuid" || led.Boxes[0].IPv4 != pin.IPv4 {
		t.Fatal(led, e)
	}
	if e = s.db.QueryRow("SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'create-pin:%'", v.CreateOperationID).Scan(&n); e != nil || n != 4 {
		t.Fatal("historical original lost", n, e)
	}
	if _, e = s.db.Exec("UPDATE network_ipv4_retirements SET native_proof_sha256=?", strings.Repeat("z", 64)); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ReadCompleteNetworkAuditLedger(ctx, b.Project, pin.Bridge, pin.ACL); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("corrupt proof accepted", e)
	}
}

func TestIPv4RetirementRefusesStaleOrUnsettledEvidence(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"completed-new-operation", "unknown", "revoked-peer", "revoked-forward", "lost-create-pin", "foreign-fence"} {
		t.Run(kind, func(t *testing.T) {
			s := newStore(t)
			a, b, pin, v := retireFixture(t, s)
			switch kind {
			case "completed-new-operation", "unknown":
				a.Key = "unrelated-operation-after-proof"
				a.ResourceKey = "box:other"
				r, e := s.Admit(ctx, a)
				if e != nil {
					t.Fatal(e)
				}
				if e = s.Claim(ctx, r.ID); e != nil {
					t.Fatal(e)
				}
				if kind == "unknown" {
					e = s.Unknown(ctx, r.ID, "DISPATCHING", "NATIVE_OUTCOME_AMBIGUOUS")
				} else {
					e = s.FailPreDispatch(ctx, r.ID, "SAFE_NATIVE_PREFLIGHT_DENIED")
				}
				if e != nil {
					t.Fatal(e)
				}
			case "revoked-peer":
				if _, e := s.db.Exec("INSERT INTO peer_grants(project,source_uuid,source_incarnation,target_uuid,target_incarnation,desired,status) VALUES(?,?,?,'old-target','old-target',0,'REVOKED')", b.Project, b.UUID, b.Incarnation); e != nil {
					t.Fatal(e)
				}
			case "revoked-forward":
				if _, e := s.db.Exec("INSERT INTO forward_claims(forward_id,project,box_uuid,incarnation,native_device,protocol,host_bind,host_port,guest_port,guest_ip,state) VALUES('old-forward',?,?,?,'bwpf-old','tcp','127.0.0.1',18000,80,'127.0.0.1','REVOKED')", b.Project, b.UUID, b.Incarnation); e != nil {
					t.Fatal(e)
				}
			case "lost-create-pin":
				if _, e := s.db.Exec("DELETE FROM operation_resource_claims WHERE operation_id=? AND resource_key=?", v.CreateOperationID, pin.reservationKeys()[0]); e != nil {
					t.Fatal(e)
				}
			case "foreign-fence":
				if _, e := s.db.Exec("INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES(?,'ipv4','10.24.18.99','foreign','foreign','foreign','RESERVED',1)", pin.Bridge); e != nil {
					t.Fatal(e)
				}
			}
			if e := retireWithProof(s, ctx, v, strings.Repeat("a", 64)); !errors.Is(e, ErrResourceLocked) {
				t.Fatal("unsafe commit", e)
			}
		})
	}
}

func TestIPv4RetirementConcurrentReplacementOneSlot(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	a, b, pin, v := retireFixture(t, s)
	if e := retireWithProof(s, ctx, v, strings.Repeat("b", 64)); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	wins := make(chan bool, 2)
	for i := 1; i <= 2; i++ {
		x := a
		x.Key = fmt.Sprintf("n2-concurrent-replacement-%d", i)
		name := fmt.Sprintf("new-%d", i)
		x.ResourceKey = "box:" + name
		r, e := s.Admit(ctx, x)
		if e != nil {
			t.Fatal(e)
		}
		if e = s.Claim(ctx, r.ID); e != nil {
			t.Fatal(e)
		}
		p := pin
		p.MAC = fmt.Sprintf("02:16:3e:11:33:%02x", i)
		p.IPv6 = fmt.Sprintf("fd43:18::%x", 100+i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, e := s.StageCreateBinding(ctx, r.ID, b.Project, name, p)
			if e != nil {
				t.Error(e)
			}
			wins <- ok
		}()
	}
	wg.Wait()
	close(wins)
	n := 0
	for ok := range wins {
		if ok {
			n++
		}
	}
	if n != 1 {
		t.Fatal("double allocation or lost slot", n)
	}
}

func retireWithProof(s *Store, ctx context.Context, v IPv4RetirementCandidate, proof string) error {
	return s.RetireVerifiedIPv4(ctx, v.Project, v.DeleteOperationID, func(_ context.Context, got IPv4RetirementCandidate) (IPv4RetirementCandidate, string, error) {
		if got != v {
			return IPv4RetirementCandidate{}, "", ErrResourceLocked
		}
		return v, proof, nil
	})
}

func TestIPv4RetirementHeldWriterReservation(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, b, pin, v := retireFixture(t, s)
	path := strings.TrimSuffix(s.ExecArchiveDir(), ".exec-archive")
	ro, e := OpenNetworkAuditReadOnly(path)
	if e != nil {
		t.Fatal(e)
	}
	defer ro.Close()
	competing, e := sql.Open("sqlite3", "file:"+path+"?_busy_timeout=50&_foreign_keys=on")
	if e != nil {
		t.Fatal(e)
	}
	defer competing.Close()
	for _, kind := range []string{"denied", "wrong-original", "wrong-proof", "canceled", "verified"} {
		e = s.RetireVerifiedIPv4(ctx, b.Project, v.DeleteOperationID, func(call context.Context, got IPv4RetirementCandidate) (IPv4RetirementCandidate, string, error) {
			// Independent root read must see committed evidence while the
			// writer fence excludes a different connection's write.
			read, e := ro.ReadIPv4RetirementCandidate(call, b.Project, v.DeleteOperationID)
			if e != nil || read != got || read != v {
				t.Fatal(read, got, e)
			}
			_, e = competing.ExecContext(call, "UPDATE schema_version SET version=version")
			if e == nil {
				t.Fatal("competing writer admitted between proof and commit")
			}
			switch kind {
			case "denied":
				return got, "", ErrResourceLocked
			case "wrong-original":
				got.DeleteOperationID = strings.Repeat("f", 32)
			case "wrong-proof":
				return got, "bad", nil
			case "canceled":
				return got, "", context.Canceled
			}
			return got, strings.Repeat("c", 64), nil
		})
		holds, er := s.ReadCreateAddressHolds(ctx, b.Project, pin.Bridge)
		if er != nil {
			t.Fatal(er)
		}
		held := holds.IPs[netip.MustParseAddr(pin.IPv4)]
		if kind == "verified" {
			if e != nil || held {
				t.Fatal(kind, e, held)
			}
		} else if e == nil || !held {
			t.Fatal(kind, e, held)
		}
	}
	// Reopen the exact ledger. No synthetic proof replay or old identity loss.
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	again, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer again.Close()
	if _, e = again.ReadIPv4RetirementCandidate(ctx, b.Project, v.DeleteOperationID); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("retirement replay after restart", e)
	}
	holds, e := again.ReadCreateAddressHolds(ctx, b.Project, pin.Bridge)
	if e != nil || holds.IPs[netip.MustParseAddr(pin.IPv4)] {
		t.Fatal(holds, e)
	}
	// A later independent admission must not remain stuck after commit.
	testCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if _, e = competing.ExecContext(testCtx, "UPDATE schema_version SET version=version"); e != nil {
		t.Fatal("writer lock retained", e)
	}
}

func TestIPv4RetirementVerifiedForwardHistory(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	var forwardID, removeID string
	_, b, pin, v := retireFixture(t, s, func(a Admission, b OwnedBox) {
		a.Action = "port_forward_create"
		a.BoxUUID = b.UUID
		a.Incarnation = b.Incarnation
		a.Key = "n2-verified-forward-create"
		a.ResourceKeys = []string{"box:" + b.UUID, "listener:tcp:127.0.0.1:23001"}
		a.ForwardCreate = &ForwardCreateAdmission{Protocol: "tcp", HostBind: "127.0.0.1", HostPort: 23001, GuestPort: 8080}
		a.SafeIntent = []byte(fmt.Sprintf(`{"box_id":%q,"protocol":"tcp","host_bind":"127.0.0.1","host_port":23001,"guest_port":8080,"confirm_external_exposure":false}`, b.UUID))
		r, e := s.Admit(ctx, a)
		if e != nil {
			t.Fatal(e)
		}
		forwardID = r.ID
		if e = s.Claim(ctx, r.ID); e != nil {
			t.Fatal(e)
		}
		if e = s.NativeAck(ctx, r.ID, "native-verified-forward-create"); e != nil {
			t.Fatal(e)
		}
		if e = s.FinalizeVerifiedForward(ctx, r.ID, true); e != nil {
			t.Fatal(e)
		}
		a.Action = "port_forward_delete"
		a.Key = "n2-verified-forward-remove"
		a.ForwardCreate = nil
		a.ForwardDeleteID = r.ID
		a.SafeIntent = []byte(fmt.Sprintf(`{"forward_id":%q}`, r.ID))
		r, e = s.Admit(ctx, a)
		if e != nil {
			t.Fatal(e)
		}
		removeID = r.ID
		if e = s.Claim(ctx, r.ID); e != nil {
			t.Fatal(e)
		}
		if e = s.NativeAck(ctx, r.ID, "native-verified-forward-delete"); e != nil {
			t.Fatal(e)
		}
		if e = s.FinalizeVerifiedForward(ctx, r.ID, false); e != nil {
			t.Fatal(e)
		}
	})
	pol, e := s.ReadIPv4RetirementPolicy(ctx, v)
	if e != nil || len(pol.Forwards) != 1 || pol.Forwards[0].ForwardID != forwardID || pol.Forwards[0].OperationID != removeID {
		t.Fatal(pol, e)
	}
	// A changed historical original must not count as verified revocation.
	if _, e = s.db.Exec("UPDATE forward_claims SET host_port=23002 WHERE forward_id=?", forwardID); e != nil {
		t.Fatal(e)
	}
	if e = retireWithProof(s, ctx, v, strings.Repeat("c", 64)); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("changed claim accepted", e)
	}
	if _, e = s.db.Exec("UPDATE forward_claims SET host_port=23001 WHERE forward_id=?", forwardID); e != nil {
		t.Fatal(e)
	}
	if e = retireWithProof(s, ctx, v, strings.Repeat("c", 64)); e != nil {
		t.Fatal(e)
	}
	holds, e := s.ReadCreateAddressHolds(ctx, b.Project, pin.Bridge)
	if e != nil || holds.IPs[netip.MustParseAddr(pin.IPv4)] {
		t.Fatal(e, holds)
	}
	claim, e := s.GetForward(ctx, b.Project, forwardID)
	if e != nil || claim.State != "REVOKED" || claim.OperationID != removeID {
		t.Fatal(claim, e)
	}
}
