package state

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This private fixture carries a synthetic original/native receipt and claim.
// All production ledger entry points must refuse incompatible state without
// adopting, migrating or dropping its unresolved operation.
func TestIncompatibleSchemaPreservesOriginalAcrossEntryPoints(t *testing.T) {
	for _, tc := range []struct{ name, mutation string }{
		{"exact-v6-with-legacy-history", "UPDATE schema_version SET version=6;" + legacyRetirementTableDDL + ";" + legacyRetirementQueueTableDDL + ";" + legacyCompactionTableDDL + ";" + legacyRetirementIndexDDL + ";" + legacyRetirementQueueIndexDDL},
		{"old-version", fmt.Sprintf("UPDATE schema_version SET version=%d", SchemaVersion-1)},
		{"new-version", fmt.Sprintf("UPDATE schema_version SET version=%d", SchemaVersion+1)},
		{"ambiguous-version", fmt.Sprintf("INSERT INTO schema_version VALUES(%d)", SchemaVersion)},
		{"foreign-table", "CREATE TABLE old_runtime_shadow (id TEXT)"},
		{"schema7-legacy-archive", legacyRetirementTableDDL},
		{"schema7-legacy-queue", legacyRetirementQueueTableDDL},
		{"schema7-legacy-compaction", legacyCompactionTableDDL},
		{"missing-declared-index", "DROP INDEX operations_work_time"},
		{"missing-retained-table", "DROP TABLE network_address_fences"},
		{"changed-retained-index", "DROP INDEX network_address_fences_owner_state; CREATE INDEX network_address_fences_owner_state ON network_address_fences(box_uuid,project,state)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			dir := t.TempDir()
			if err := os.Chmod(dir, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "ledger.sqlite")
			s, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			_, tok, err := s.IssueToken(ctx, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			original, err := s.AdmitScoped(ctx, Admission{
				Layer: "fixture-layer", ActorID: tok.ID, TokenID: tok.ID,
				Key: "schema-original-unknown-001", Action: "test-schema",
				SafeIntent: []byte(`{"fixture":true}`), RequestID: "schema-fixture",
				ResourceKey: "box:schema-fixture",
			})
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Claim(ctx, original.ID); err != nil {
				t.Fatal(err)
			}
			const native = "12345678-1234-1234-1234-123456789abc"
			if err = s.NativeAck(ctx, original.ID, native); err != nil {
				t.Fatal(err)
			}
			if n, e := s.Recover(ctx); e != nil || n != 1 {
				t.Fatalf("fixture recovery: %d %v", n, e)
			}
			var claims int
			if err = s.db.QueryRow("SELECT count(*) FROM operation_resource_claims WHERE operation_id=?", original.ID).Scan(&claims); err != nil || claims == 0 {
				t.Fatalf("fixture claim: %d %v", claims, err)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			db, err := sql.Open("sqlite3", "file:"+path+"?mode=rw")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = db.Exec(tc.mutation); err != nil {
				_ = db.Close()
				t.Fatal(err)
			}
			if tc.name == "exact-v6-with-legacy-history" {
				if _, err = db.Exec("INSERT INTO network_ipv4_retirement_queue(delete_operation_id,project,attempt_order) VALUES(?,?,1)", original.ID, "fixture-layer"); err != nil {
					t.Fatal(err)
				}
			}
			if err = db.Close(); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range []struct {
				name string
				open func(string) (*Store, error)
			}{
				{"controller", Open},
				{"recovery", OpenExistingRecovery},
				{"root-audit", OpenNetworkAuditReadOnly},
			} {
				t.Run(entry.name, func(t *testing.T) {
					rejected, err := entry.open(path)
					if err == nil {
						_ = rejected.Close()
						t.Fatal("incompatible ledger unexpectedly accepted")
					}
					after, err := os.ReadFile(path)
					if err != nil || !bytes.Equal(before, after) {
						t.Fatalf("refusal changed original ledger bytes: %v", err)
					}
					if _, err := os.Lstat(path + ".exec-archive"); !os.IsNotExist(err) {
						t.Fatalf("refusal created an archive or could not observe it: %v", err)
					}
				})
			}
			if snap, err := OpenDiagnosticSnapshot(ctx, path); err == nil {
				snap.Close()
				t.Fatal("incompatible diagnostic snapshot accepted")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("diagnostic refusal changed ledger", err)
			}
			// Inspect the deliberately incompatible private fixture directly;
			// never open or migrate a real old installation to test refusal.
			ro, err := sql.Open("sqlite3", "file:"+path+"?mode=ro&_query_only=1")
			if err != nil {
				t.Fatal(err)
			}
			defer ro.Close()
			var state, tracking, receipt string
			if err = ro.QueryRow("SELECT state,tracking_id,native_operation_id FROM operations WHERE id=?", original.ID).
				Scan(&state, &tracking, &receipt); err != nil || state != "UNKNOWN" || tracking != original.TrackingID || receipt != native {
				t.Fatalf("unresolved original changed: %s %s %s %v", state, tracking, receipt, err)
			}
			var remaining int
			if err = ro.QueryRow("SELECT count(*) FROM operation_resource_claims WHERE operation_id=?", original.ID).
				Scan(&remaining); err != nil || remaining != claims {
				t.Fatalf("original claim changed: %d != %d (%v)", remaining, claims, err)
			}
			if tc.name == "exact-v6-with-legacy-history" {
				var n int
				if err = ro.QueryRow("SELECT count(*) FROM network_ipv4_retirement_queue WHERE delete_operation_id=?", original.ID).Scan(&n); err != nil || n != 1 {
					t.Fatal("legacy history changed", n, err)
				}
			}

		})
	}
}

// A fresh ledger contains only the retained model; retired entrypoints
// must not even invoke their native witness on the new schema.
func TestSchema7FreshLegacyEntryRefusal(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	var version, n int
	if e := s.db.QueryRow("SELECT version FROM schema_version").Scan(&version); e != nil || version != 7 {
		t.Fatal(version, e)
	}
	if e := s.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name LIKE 'network_ipv4_%'").Scan(&n); e != nil || n != 0 {
		t.Fatal(n, e)
	}
	called := false
	if e := s.RetireVerifiedIPv4(ctx, "user-1001", "original", func(context.Context, IPv4RetirementCandidate) (IPv4RetirementCandidate, string, error) {
		called = true
		return IPv4RetirementCandidate{}, "", nil
	}); e == nil || called {
		t.Fatal("retired callback invoked", e, called)
	}
	if ok, e := s.PrepareIPv4Compaction(ctx, "user-1001", "original", func(context.Context, IPv4RetirementCandidate) (string, error) { called = true; return "", nil }); e == nil || ok || called {
		t.Fatal("compaction callback invoked", ok, e, called)
	}
	if e := s.db.QueryRow("SELECT count(*) FROM operations").Scan(&n); e != nil || n != 0 {
		t.Fatal(n, e)
	}
}
