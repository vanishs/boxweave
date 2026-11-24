package state

import (
	"context"
	"path/filepath"
	"testing"
)

func TestScopedRetirementLedgerEmptyAndLiveRefused(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "ledger.sqlite")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
	read := func() (ScopedNetworkAuditLedger, error) {
		r, err := OpenNetworkAuditReadOnly(path)
		if err != nil {
			return ScopedNetworkAuditLedger{}, err
		}
		defer r.Close()
		return r.ReadScopedRetirementLedger(ctx, "user-1001", "bwbr0")
	}
	if got, err := read(); err != nil || len(got.Boxes) != 0 || len(got.Retired) != 0 {
		t.Fatalf("empty retirement ledger refused: %+v %v", got, err)
	}
	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s4b2InsertAuditFixture(t, st)
	st.Close()
	if _, err := read(); err == nil {
		t.Fatal("live or foreign original admitted to retirement")
	}
}
