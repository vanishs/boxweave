package state

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// This characterizes immutable=1; it is NOT a proposed live DB reader.
func TestPR177D0ImmutableIsNotLiveWALReader(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ledger.sqlite")
	rw, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = rw.Close(); err != nil {
		t.Fatal(err)
	}
	uri := (&url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro&immutable=1&_query_only=1"}).String()
	before := d0FileNames(t, dir)
	cold, err := sql.Open("sqlite3", uri)
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err = cold.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_attempts WHERE kind='d0-uncheckpointed'").Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
	if err = cold.Close(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, d0FileNames(t, dir)) {
		t.Fatal("immutable cold read changed file inventory")
	}
	rw, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer rw.Close()
	if _, err = rw.db.Exec("PRAGMA wal_autocheckpoint=0"); err != nil {
		t.Fatal(err)
	}
	if _, err = rw.db.Exec("INSERT INTO oauth_attempts(at,kind) VALUES(99,'d0-uncheckpointed')"); err != nil {
		t.Fatal(err)
	}
	live, err := OpenNetworkAuditReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer live.Close()
	if err = live.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_attempts WHERE kind='d0-uncheckpointed'").Scan(&n); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	stale, err := sql.Open("sqlite3", uri)
	if err != nil {
		t.Fatal(err)
	}
	defer stale.Close()
	if err = stale.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_attempts WHERE kind='d0-uncheckpointed'").Scan(&n); err != nil || n != 0 {
		t.Fatal("immutable unexpectedly observed uncheckpointed data", n, err)
	}
	t.Log("D0_IMMUTABLE_COLD_NO_SIDECAR_BUT_LIVE_WAL_STALE")
}
