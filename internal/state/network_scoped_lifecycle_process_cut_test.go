package state

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/vanishs/boxweave/internal/peerscope"
)

// Scoped lifecycle counterpart of the existing BH018 Exec subprocess test.
// Actual SQLite WAL and process interruption, synthetic native/Root proof.
// No installed controller, host service or Incus process is touched.
func TestP1ScopedLifecycleCutChild(t *testing.T) {
	stage := os.Getenv("BW_P1_CUT_STAGE")
	if stage == "" {
		return
	}
	action, path, marker, id := os.Getenv("BW_P1_CUT_ACTION"), os.Getenv("BW_P1_CUT_DB"), os.Getenv("BW_P1_CUT_MARKER"), os.Getenv("BW_P1_CUT_ID")
	if (stage != "before" && stage != "after") || (action != "create" && action != "delete") || path == "" || marker == "" || id == "" {
		t.Fatal("invalid isolated child identity")
	}
	ctx := context.Background()
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if stage == "before" {
		sql.Register("p1_scoped_cut", &sqlite3.SQLiteDriver{ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.RegisterFunc("p1_pause", func() int {
				if err := os.WriteFile(marker, []byte("in-transaction"), 0600); err != nil {
					panic(err)
				}
				select {}
			}, false)
		}})
		db, err := sql.Open("p1_scoped_cut", "file:"+path+"?_busy_timeout=5000&_foreign_keys=on")
		if err != nil {
			t.Fatal(err)
		}
		db.SetMaxOpenConns(1)
		if err := s.db.Close(); err != nil {
			t.Fatal(err)
		}
		s.db = db
		for _, q := range []string{"PRAGMA journal_mode=WAL", "PRAGMA synchronous=FULL",
			"CREATE TEMP TRIGGER p1_cut AFTER INSERT ON main.operation_events WHEN NEW.state='SUCCEEDED' AND NEW.safe_detail IN ('ORIGINAL_CREATE_UUID_ATOMIC','SCOPED_DELETE_ACL_ABSENT_ATOMIC') BEGIN SELECT p1_pause(); END"} {
			if _, err := db.Exec(q); err != nil {
				t.Fatal(err)
			}
		}
	}
	acl, err := peerscope.ACLName(p1Project, p1Bridge, p1UUID)
	if err != nil {
		t.Fatal(err)
	}
	if action == "create" {
		err = s.FinalizeVerifiedScopedCreate(ctx, id, p1Project, "alpha", p1UUID,
			VerifiedCreateBinding{p1Bridge, acl, "02:16:3e:11:22:33", "10.243.18.11", "fd43:18::11"})
	} else {
		box, e := s.OwnedBox(ctx, p1Project, p1UUID)
		if e != nil {
			t.Fatal(e)
		}
		err = s.FinalizeVerifiedScopedDelete(ctx, id, box)
	}
	if err != nil {
		t.Fatal(err)
	}
	if stage != "after" {
		t.Fatal("before-commit trigger returned")
	}
	if err := os.WriteFile(marker, []byte("durable-commit"), 0600); err != nil {
		t.Fatal(err)
	}
	select {}
}

func TestP1ScopedLifecycleActualProcessCut(t *testing.T) {
	for _, action := range []string{"create", "delete"} {
		for _, stage := range []string{"before", "after"} {
			t.Run(action+"/"+stage, func(t *testing.T) {
				ctx := context.Background()
				s, original, in, pin := p1CreateFixture(t, true)
				acl, err := peerscope.ACLName(in.Project, in.Bridge, p1UUID)
				if err != nil {
					t.Fatal(err)
				}
				pin.ACL = acl
				nativeID := "fixture-native-create"
				if action == "delete" {
					if err := s.FinalizeVerifiedScopedCreate(ctx, original.ID, in.Project, in.Name, p1UUID, pin); err != nil {
						t.Fatal(err)
					}
					a := b5DeleteAdmission(t, s, p1UUID, in.Name)
					a.Layer, a.SafeIntent = in.Project, []byte(`{"box_id":"`+p1UUID+`"}`)
					original = a4AdmitMode(t, s, a, true)
					if err := s.Claim(ctx, original.ID); err != nil {
						t.Fatal(err)
					}
					if err := s.AuthorizeCascadeFinal(ctx, original.ID); err != nil {
						t.Fatal(err)
					}
					nativeID = "fixture-native-delete"
					if err := s.NativeAck(ctx, original.ID, nativeID); err != nil {
						t.Fatal(err)
					}
				}
				var seq int
				var schema, path string
				if err := s.db.QueryRow("PRAGMA database_list").Scan(&seq, &schema, &path); err != nil || schema != "main" {
					t.Fatal(schema, err)
				}
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
				marker := filepath.Join(t.TempDir(), "boundary")
				cmd := exec.Command(os.Args[0], "-test.run=^TestP1ScopedLifecycleCutChild$")
				cmd.Env = append(os.Environ(), "BW_P1_CUT_STAGE="+stage, "BW_P1_CUT_ACTION="+action,
					"BW_P1_CUT_DB="+path, "BW_P1_CUT_MARKER="+marker, "BW_P1_CUT_ID="+original.ID)
				var output bytes.Buffer
				cmd.Stdout, cmd.Stderr = &output, &output
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
				deadline := time.Now().Add(12 * time.Second)
				wantMarker := "in-transaction"
				if stage == "after" {
					wantMarker = "durable-commit"
				}
				for {
					if raw, err := os.ReadFile(marker); err == nil && string(raw) == wantMarker {
						break
					}
					if time.Now().After(deadline) {
						_ = cmd.Process.Kill()
						_ = cmd.Wait()
						t.Fatalf("boundary not reached: %s", output.String())
					}
					time.Sleep(20 * time.Millisecond)
				}
				if err := cmd.Process.Kill(); err != nil {
					t.Fatal(err)
				}
				if err := cmd.Wait(); err == nil {
					t.Fatal("child unexpectedly exited normally")
				}
				reopened, err := Open(path)
				if err != nil {
					t.Fatal(err)
				}
				defer reopened.Close()
				count := func(q string, args ...any) int {
					t.Helper()
					var n int
					if err := reopened.db.QueryRow(q, args...).Scan(&n); err != nil {
						t.Fatal(err)
					}
					return n
				}
				wantState, wantEvents, wantRecover := "SUCCEEDED", 1, 0
				if stage == "before" {
					wantState, wantEvents, wantRecover = "VERIFYING", 0, 1
				}
				p1AssertOriginal(t, reopened, original.ID, wantState, nativeID)
				if got := count("SELECT count(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED'", original.ID); got != wantEvents {
					t.Fatal("split terminal event", got)
				}
				if n, err := reopened.Recover(ctx); err != nil || n != int64(wantRecover) {
					t.Fatal("restart recovery", n, err)
				}
				if stage == "before" {
					p1AssertOriginal(t, reopened, original.ID, "UNKNOWN", nativeID)
					tx, err := reopened.db.BeginTx(ctx, nil)
					if err != nil {
						t.Fatal(err)
					}
					if action == "create" {
						p, err := readScopedCreateOriginalTxPhase(ctx, tx, original.ID, in.Project, in.Bridge, "UNKNOWN")
						tx.Rollback()
						if err != nil || p.Quarantine.ACL != peerscope.QuarantineACL {
							t.Fatal("lost staged quarantine claim", p, err)
						}
						if count("SELECT count(*) FROM owned_boxes WHERE uuid=?", p1UUID) != 0 {
							t.Fatal("partial ownership committed")
						}
					} else {
						p, err := readScopedDeleteOriginalTxPhase(ctx, tx, original.ID, in.Project, in.Bridge, "UNKNOWN")
						tx.Rollback()
						if err != nil || p.Binding != pin {
							t.Fatal("lost Delete binding", p, err)
						}
						if count("SELECT count(*) FROM network_address_fences WHERE box_uuid=? AND state='RESERVED'", p1UUID) != 2 {
							t.Fatal("partial address release")
						}
					}
				} else if action == "create" {
					if count("SELECT count(*) FROM owned_boxes WHERE uuid=? AND state='OWNED'", p1UUID) != 1 ||
						count("SELECT count(*) FROM network_address_fences WHERE box_uuid=? AND state='RESERVED' AND operation_id=?", p1UUID, original.ID) != 2 ||
						count("SELECT count(*) FROM network_bindings WHERE box_uuid=? AND state='PINNED' AND acl=?", p1UUID, acl) != 1 {
						t.Fatal("committed Create lost ownership/binding/fences")
					}
				} else {
					if count("SELECT count(*) FROM owned_boxes WHERE uuid=? AND state='DELETED'", p1UUID) != 1 ||
						count("SELECT count(*) FROM network_address_fences WHERE box_uuid=? AND state='RELEASED' AND operation_id=?", p1UUID, original.ID) != 2 ||
						count("SELECT count(*) FROM network_bindings WHERE box_uuid=? AND state='RELEASED'", p1UUID) != 1 {
						t.Fatal("committed Delete lost tombstone/binding/fences")
					}
				}
				t.Logf("P1B_SCOPED_SQLITE_PROCESS_CUT=PASS %s %s original=%s (synthetic native proof)", action, stage, original.ID)
			})
		}
	}
}
