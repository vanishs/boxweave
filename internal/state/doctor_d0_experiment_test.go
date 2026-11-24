package state

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// D0 characterizes existing readers, not a new Doctor implementation.
// All writes belong only to disposable test fixtures.
func d0FileNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

func TestPR177D0ReadOnlyReaderFootprint(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ledger.sqlite")
	if ro, err := OpenNetworkAuditReadOnly(path); err == nil {
		ro.Close()
		t.Fatal("missing database accepted")
	}
	if len(d0FileNames(t, dir)) != 0 {
		t.Fatal("missing database reader changed directory")
	}
	rw, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = rw.db.Exec("INSERT INTO oauth_attempts(at,kind) VALUES(7,'d0-live')"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	names := d0FileNames(t, dir)
	ro, err := OpenNetworkAuditReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err = ro.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_attempts WHERE kind='d0-live'").Scan(&n); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if _, err = ro.db.Exec("DELETE FROM oauth_attempts"); err == nil {
		t.Fatal("read-only reader admitted DML")
	}
	if err = ro.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	end, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(before) != sha256.Sum256(after) || !os.SameFile(info, end) || !info.ModTime().Equal(end.ModTime()) || !reflect.DeepEqual(names, d0FileNames(t, dir)) {
		t.Fatal("live database file or inventory changed")
	}
	t.Log("D0_LIVE_WAL_READ_ONLY_PASS")
	if err = rw.Close(); err != nil {
		t.Fatal(err)
	}
	// SQLite may create or update WAL/SHM sidecars for mode=ro. Measure
	// before, while open, and after; do not label query_only as zero writes.
	coldBefore := d0FileNames(t, dir)
	coldDB, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ro, err = OpenNetworkAuditReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	coldDuring := d0FileNames(t, dir)
	if err = ro.Close(); err != nil {
		t.Fatal(err)
	}
	coldAfter := d0FileNames(t, dir)
	bytes, err := os.ReadFile(path)
	if err != nil || sha256.Sum256(coldDB) != sha256.Sum256(bytes) {
		t.Fatal("cold database main file changed", err)
	}
	t.Logf("D0_COLD_WAL_FOOTPRINT before=%v during=%v after=%v", coldBefore, coldDuring, coldAfter)
}

func TestPR177D0ScopedLedgerDiagnosticBoundary(t *testing.T) {
	ctx := context.Background()
	empty := newStore(t)
	if _, err := empty.ReadCompleteScopedNetworkAuditLedger(ctx, "user-1001", "bwbr0"); !errors.Is(err, ErrResourceLocked) {
		t.Fatal("expected mutation-grade nonempty reader refusal", err)
	}
	tx, err := empty.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	l, err := readCompleteScopedNetworkAuditLedgerModeTx(ctx, tx, "user-1001", "bwbr0", true)
	tx.Rollback()
	if err != nil || len(l.Boxes) != 0 || len(l.Retired) != 0 {
		t.Fatal("empty scope observation", l, err)
	}
	t.Log("D0_EMPTY_SCOPE_REQUIRES_DIAGNOSTIC_READER")
	s, b, _ := n3RetiredFixture(t)
	l, err = s.ReadCompleteScopedNetworkAuditLedger(ctx, b.Project, "incusbr-1001")
	if err != nil || len(l.Boxes) != 2 || len(l.Retired) != 1 {
		t.Fatal(l, err)
	}
	old, err := s.ReadNativeNetworkLedger(ctx, b.Project)
	if err != nil {
		t.Fatal(err)
	}
	if old.OrphanBindings != 0 || old.OrphanFences != 0 {
		t.Fatal("valid retired rows counted as orphans", old.OrphanBindings, old.OrphanFences)
	}
	proof := 0
	for _, item := range old.Boxes {
		if item.OriginalProof {
			proof++
		}
	}
	t.Logf("D0_VALID_RETIRED_HISTORY scoped_owned=%d scoped_retired=%d legacy_view_original_proof=%d/%d orphan_bindings=%d orphan_fences=%d", len(l.Boxes), len(l.Retired), proof, len(old.Boxes), old.OrphanBindings, old.OrphanFences)
}
