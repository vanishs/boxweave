package state

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestS4B2AddressFenceNeverRecyclesAcrossUUIDIncarnation(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	a := NetworkAddressFence{
		Bridge: "bw-bridge", Family: "ipv4", Address: "10.90.0.42",
		Project: "user-1001", BoxUUID: "original-b-uuid", Incarnation: "original-b-uuid",
		OperationID: "original-lease-operation",
	}
	if err := s.ReserveNetworkAddress(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := s.ReserveNetworkAddress(ctx, a); err != nil {
		t.Fatalf("idempotent same-identity reserve failed: %v", err)
	}
	c := a
	c.BoxUUID = "replacement-c-uuid"
	c.Incarnation = "replacement-c-uuid"
	if err := s.ReserveNetworkAddress(ctx, c); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("old IP grant transferred to different UUID: %v", err)
	}
	if err := s.QuarantineNetworkAddress(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := s.QuarantineNetworkAddress(ctx, a); err != nil {
		t.Fatalf("idempotent quarantine: %v", err)
	}
	if err := s.ReserveNetworkAddress(ctx, a); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("quarantined IP silently reactivated: %v", err)
	}
	if err := s.ReserveNetworkAddress(ctx, c); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("quarantined IP transferred to new Box: %v", err)
	}
	var state string
	if err := s.db.QueryRowContext(ctx, "SELECT state FROM network_address_fences WHERE bridge=? AND family=? AND address=?",
		a.Bridge, a.Family, a.Address).Scan(&state); err != nil || state != "QUARANTINED" {
		t.Fatalf("fence not persisted: %q %v", state, err)
	}
	b4 := a
	b4.Family = "ipv6"
	b4.Address = "fd90:1::42"
	if err := s.ReserveNetworkAddress(ctx, b4); err != nil {
		t.Fatal(err)
	}
	invalid := a
	invalid.Family = "ipv6"
	invalid.Address = "10.90.0.42"
	if err := s.ReserveNetworkAddress(ctx, invalid); !errors.Is(err, ErrInvalid) {
		t.Fatalf("IPv6 claim with IPv4 address: %v", err)
	}
	multicast := a
	multicast.Address = "239.2.3.4"
	if err := s.ReserveNetworkAddress(ctx, multicast); !errors.Is(err, ErrInvalid) {
		t.Fatalf("multicast reserve admitted: %v", err)
	}
}
func TestS4B2AddressFenceAndLifecycleSurviveReopen(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "s4b2-state.sqlite")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a := NetworkAddressFence{Bridge: "bw-bridge", Family: "ipv4", Address: "10.90.0.43",
		Project: "user-1001", BoxUUID: "old-b-uuid", Incarnation: "old-b-uuid",
		OperationID: "unknown-operation"}
	if err := s.ReserveNetworkAddress(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := s.QuarantineNetworkAddress(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	c := a
	c.BoxUUID = "new-c-uuid"
	c.Incarnation = "new-c-uuid"
	if err := reopened.ReserveNetworkAddress(ctx, c); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("new incarnation stole quarantined IP after reboot: %v", err)
	}
	_, tok, err := reopened.IssueToken(ctx,time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	del := Admission{Layer: a.Project, ActorID: tok.ID, TokenID: tok.ID,
		Key: "s4b2-lease-reboot-delete-fence-1116", RequestID: "reboot-delete",
		Action: "box_delete", SafeIntent: []byte("{}"), ResourceKey: "box:old-b",
		BoxUUID: a.BoxUUID, Incarnation: a.Incarnation}
	if _, err := reopened.Admit(ctx, del); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("UNKNOWN lease failed to block delete after restart: %v", err)
	}
}
