package install

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/state"
)

func s4b2AuditRaceStore(t *testing.T) (*state.Store, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "audit.sqlite")
	s, e := state.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	db, e := sql.Open("sqlite3", path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = db.Close() })
	for i := 1; i <= 3; i++ {
		id := fmt.Sprintf("11111111-1111-4111-8111-%012d", i)
		name := fmt.Sprintf("b%d", i)
		v4 := fmt.Sprintf("10.243.18.%d", i+10)
		v6 := fmt.Sprintf("fd43:18::%d", i+10)
		mac := fmt.Sprintf("02:16:3e:11:12:%02x", i)
		now := time.Now().UnixNano()
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{"INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES(?,?,?,?,'OWNED',?)", []any{id, "user-1001", name, id, now}},
			{"INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES(?,?,?,?,?,?,?,?,?,'PINNED',?)", []any{"user-1001", id, id, "eth0", "bwbr0", "bw-s4b2", mac, v4, v6, now}},
			{"INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0','ipv4',?,?,?,?,'RESERVED',?)", []any{v4, "user-1001", id, id, now}},
			{"INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0','ipv6',?,?,?,?,'RESERVED',?)", []any{v6, "user-1001", id, id, now}},
		} {
			if _, e := db.Exec(q.sql, q.args...); e != nil {
				t.Fatal(e)
			}
		}
	}
	return s, db
}
func TestS4B2RootAuditRefusesSQLiteAndNativeInterleaving(t *testing.T) {
	for _, tc := range []struct {
		name     string
		expected int
		fail     bool
	}{
		{"stable", 2, false}, {"mac-drift", 1, true}, {"lease-drift", 2, true}, {"native-drift", 2, true}, {"root-failure", 1, true}, {"wrong-root-scope", 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, db := s4b2AuditRaceStore(t)
			calls := 0
			read := func(_ context.Context, trg NativePeerBridgeAuditTarget) (NativePeerBridgeAuditResult, error) {
				calls++
				if len(trg.Boxes) != 3 {
					t.Fatalf("wrong root owner scope %d", len(trg.Boxes))
				}
				proof := NativePeerBridgeAuditResult{Project: trg.Project, Bridge: trg.Bridge, ACL: trg.ACL, OwnedBoxes: 3, NativeDigestSHA256: strings.Repeat("a", 64)}
				var err error
				switch tc.name {
				case "mac-drift":
					if calls == 1 {
						_, err = db.Exec("UPDATE network_bindings SET mac='02:16:3e:aa:bb:cc' WHERE box_uuid=?", trg.Boxes[0].UUID)
					}
				case "lease-drift":
					if calls == 2 {
						_, err = db.Exec("UPDATE network_address_fences SET state='QUARANTINED' WHERE box_uuid=? AND family='ipv6'", trg.Boxes[1].UUID)
					}
				case "native-drift":
					if calls == 2 {
						proof.NativeDigestSHA256 = strings.Repeat("b", 64)
					}
				case "root-failure":
					return NativePeerBridgeAuditResult{}, errors.New("native error")
				case "wrong-root-scope":
					proof.ACL = "foreign-acl"
				}
				if err != nil {
					t.Fatal(err)
				}
				return proof, nil
			}
			rep, err := nativePeerBridgeAuditFromStoreWithVerifier(t.Context(), s, "user-1001", "bwbr0", "bw-s4b2", read)
			if calls != tc.expected || (err != nil) != tc.fail || (tc.fail && rep.NativeDigestSHA256 != "") || (!tc.fail && rep.NativeDigestSHA256 != strings.Repeat("a", 64)) {
				t.Fatalf("unsafe audit acceptance calls=%d report=%+v err=%v", calls, rep, err)
			}
		})
	}
}
func TestS4B2RootAuditPreventsUnpinnedAndCancelledNativeRead(t *testing.T) {
	s, db := s4b2AuditRaceStore(t)
	calls := 0
	read := func(context.Context, NativePeerBridgeAuditTarget) (NativePeerBridgeAuditResult, error) {
		calls++
		return NativePeerBridgeAuditResult{}, errors.New("should not execute")
	}
	if _, e := db.Exec("UPDATE network_bindings SET state='QUARANTINED'"); e != nil {
		t.Fatal(e)
	}
	if _, e := nativePeerBridgeAuditFromStoreWithVerifier(t.Context(), s, "user-1001", "bwbr0", "bw-s4b2", read); e == nil || calls != 0 {
		t.Fatal("invalid owner sent to root", e)
	}
	if _, e := nativePeerBridgeAuditFromStoreWithVerifier(t.Context(), s, "user-1001", "bwbr0", "bw-s4b2", nil); e == nil {
		t.Fatal("nil native witness")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, e := nativePeerBridgeAuditFromStoreWithVerifier(ctx, s, "user-1001", "bwbr0", "bw-s4b2", read); e == nil || calls != 0 {
		t.Fatal("canceled audit dispatched", e)
	}
}
