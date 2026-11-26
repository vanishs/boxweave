package state

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestP1ScopedAdmissionMixedHistoryAndOriginalReplay(t *testing.T) {
	ctx := context.Background()
	for _, table := range []string{"network_ipv4_retirements", "network_ipv4_retirement_queue", "network_ipv4_compactions"} {
		t.Run(table, func(t *testing.T) {
			s := newStore(t)
			_, actor, err := s.IssueToken(ctx, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			admission := func(project, bridge, name, key string) Admission {
				raw, err := json.Marshal(ScopedCreateIntent{Schema: 2, Project: project, Bridge: bridge, Name: name, CPU: 1, MemoryMiB: 1024, DiskGiB: 10, Options: json.RawMessage("{}")})
				if err != nil {
					t.Fatal(err)
				}
				return Admission{Layer: project, ActorID: actor.ID, TokenID: actor.ID, Key: key, RequestID: key, Action: "box_create", ResourceKey: "box:" + name, SafeIntent: raw}
			}
			original := admission("user-2002", "incusbr-2002", "scoped-original", "p1-scoped-admit-original")
			rec, err := s.AdmitScoped(ctx, original)
			if err != nil || rec.Duplicate {
				t.Fatal(rec, err)
			}
			_, b, _, retired := retireFixture(t, s)
			switch table {
			case "network_ipv4_retirements":
				err = retireWithProof(s, ctx, retired, strings.Repeat("a", 64))
			case "network_ipv4_retirement_queue":
				_, err = s.db.Exec("INSERT OR IGNORE INTO network_ipv4_retirement_queue(delete_operation_id,project,attempt_order) VALUES(?,?,1)", retired.DeleteOperationID, b.Project)
			case "network_ipv4_compactions":
				_, err = s.db.Exec("INSERT INTO network_ipv4_compactions(project,delete_operation_id,plan_sha256,prepared_at) VALUES(?,?,?,1)", b.Project, retired.DeleteOperationID, strings.Repeat("b", 64))
			}
			if err != nil {
				t.Fatal(err)
			}
			var n int
			if err = s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 1 {
				t.Fatal(n, err)
			}
			if got, err := s.AdmitScoped(ctx, original); err != nil || !got.Duplicate || got.ID != rec.ID || got.TrackingID != rec.TrackingID {
				t.Fatal(got, err)
			}
			fresh := admission("user-2003", "incusbr-2003", "scoped-fresh", "p1-scoped-admit-fresh")
			if _, err = s.AdmitScoped(ctx, fresh); !errors.Is(err, ErrResourceLocked) {
				t.Fatal("mixed legacy history admitted", err)
			}
			for _, q := range []string{"SELECT count(*) FROM operations WHERE layer=?", "SELECT count(*) FROM operation_resource_claims WHERE layer=?"} {
				if err = s.db.QueryRow(q, fresh.Layer).Scan(&n); err != nil || n != 0 {
					t.Fatal("partial admission", q, n, err)
				}
			}
		})
	}
}
