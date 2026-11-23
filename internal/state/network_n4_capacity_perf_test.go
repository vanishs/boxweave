package state

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/networklimits"
)

// N4's one-use 1024-row SQLite-only budget is intentionally separate from
// native Incus, root PRS and real guest scale. A full 1024-host native check
// still requires its own exact-head installation and packet/resource evidence.
func TestN4ProjectCapacityAndNetworkAudit1024LocalBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("N4 opt-in scale fixture")
	}
	if v := os.Getenv("BOXWEAVE_N4_PERF_FIXTURE"); v != "n4-1024-local-only" {
		t.Skip("opt-in SQLite-only N4 scale fixture")
	}
	const project = "user-1001"
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	st := newStore(t)
	tx, e := st.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < networklimits.MaxProjectBoxes; i++ {
		uuid := fmt.Sprintf("11111111-1111-4111-8111-%012d", i+1)
		name := fmt.Sprintf("n4-budget-%04d", i)
		mac := fmt.Sprintf("02:16:3e:11:%02x:%02x", i/256, i%256)
		ip4 := fmt.Sprintf("10.243.%d.%d", 16+(i+2)/256, (i+2)%256)
		ip6 := fmt.Sprintf("fd43:18::%x", i+2)
		if _, e = tx.ExecContext(ctx, "INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES(?, ?, ?, ?, 'OWNED',1)", uuid, project, name, uuid); e != nil {
			t.Fatal(e)
		}
		if _, e = tx.ExecContext(ctx, "INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES(?,?,?,'eth0','bwbr0','bw-s4b2',?,?,?,'PINNED',1)", project, uuid, uuid, mac, ip4, ip6); e != nil {
			t.Fatal(e)
		}
		for _, f := range []struct{ family, address string }{{"ipv4", ip4}, {"ipv6", ip6}} {
			if _, e = tx.ExecContext(ctx, "INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0',?,?,?,?,?,'RESERVED',1)", f.family, f.address, project, uuid, uuid); e != nil {
				t.Fatal(e)
			}
		}
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	// Exclude setup time and GC from each read-only API budget measurement.
	measure := func(name string, run func() error) {
		t.Helper()
		const iterations = 3
		times := make([]time.Duration, iterations)
		var maxAlloc uint64
		for i := 0; i < iterations; i++ {
			runtime.GC()
			var start, end runtime.MemStats
			runtime.ReadMemStats(&start)
			begin := time.Now()
			if e = run(); e != nil {
				t.Fatalf("%s failed: %v", name, e)
			}
			times[i] = time.Since(begin)
			runtime.ReadMemStats(&end)
			if delta := end.TotalAlloc - start.TotalAlloc; delta > maxAlloc {
				maxAlloc = delta
			}
			if times[i] > 15*time.Second {
				t.Fatalf("N4_SLOW_SQLITE_%s duration=%s >15s budget", name, times[i])
			}
		}
		t.Logf("N4_1024_SQLITE_%s=PASS runs=%v max_alloc_bytes=%d", name, times, maxAlloc)
	}
	measure("CAPACITY", func() error {
		result, e := st.ReadProjectBoxCapacity(ctx, project)
		if e != nil {
			return e
		}
		if result.Owned != networklimits.MaxProjectBoxes || result.Remaining != 0 || result.PendingCreates != 0 {
			return fmt.Errorf("capacity mismatch: %+v", result)
		}
		return nil
	})
	measure("NATIVE_LEDGER", func() error {
		result, e := st.ReadNativeNetworkLedger(ctx, project)
		if e != nil {
			return e
		}
		if len(result.Boxes) != networklimits.MaxProjectBoxes || result.OrphanBindings != 0 || result.OrphanFences != 0 || result.Unsettled != 0 {
			return fmt.Errorf("truncated/false orphan: boxes=%d orphans=%d/%d unsettled=%d", len(result.Boxes), result.OrphanBindings, result.OrphanFences, result.Unsettled)
		}
		if result.Boxes[0].OriginalProof {
			return fmt.Errorf("fabricated original Create proof")
		}
		return nil
	})
	measure("PEER_ROOT_LEDGER", func() error {
		result, e := st.ReadCompleteNetworkAuditLedger(ctx, project, "bwbr0", "bw-s4b2")
		if e != nil {
			return e
		}
		if len(result.Boxes) != networklimits.MaxProjectBoxes {
			return fmt.Errorf("peer root audit truncated %d", len(result.Boxes))
		}
		return nil
	})
}
