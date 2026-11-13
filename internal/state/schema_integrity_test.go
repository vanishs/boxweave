package state

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSameVersionSQLiteDDLDriftAndTriggersFailClosed(t *testing.T) {
	for _, tc := range []struct {
		name, mutate string
	}{
		{"same-version-extra-column", "ALTER TABLE credentials ADD COLUMN silent_extra TEXT"},
		{"same-version-renamed-column", "ALTER TABLE operations RENAME COLUMN tracking_id TO broken_tracking_id"},
		{"same-version-trigger", "CREATE TRIGGER foreign_insert AFTER INSERT ON credentials BEGIN DELETE FROM operations; END"},
		{"unsupported-newer-version", "UPDATE schema_version SET version=3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Chmod(dir, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "ledger.sqlite")
			created, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			if err = created.Close(); err != nil {
				t.Fatal(err)
			}
			db, err := sql.Open("sqlite3", "file:"+path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = db.Exec(tc.mutate); err != nil {
				t.Fatal(err)
			}
			if err = db.Close(); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			existing, err := Open(path)
			if err == nil {
				_ = existing.Close()
				t.Fatalf("refused neither version drift nor foreign schema: %s", tc.name)
			}
			if !strings.Contains(err.Error(), "schema") && !strings.Contains(err.Error(), "trigger") &&
				!strings.Contains(err.Error(), "migration") {
				t.Fatalf("missing actionable schema refusal: %v", err)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(before) {
				t.Fatal("failed preflight modified original database bytes")
			}
		})
	}
}