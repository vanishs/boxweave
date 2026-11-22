package state

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/networklimits"
)

func seedCapacityOwned(t *testing.T, s *Store, project string, n int) {
	t.Helper()
	// SQL fixtures only: do not pretend these are native acceptance evidence.
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%s-fixture-%04d", project, i)
		if _, e := s.db.Exec("INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES(?,?,?,?,'OWNED',?)", id, project, id, id, 1); e != nil {
			t.Fatal(e)
		}
	}
}

func TestProjectCapacityConcurrentAdmissionAndReplay(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	a, box, original := s4CreateDeleteFixture(t, st)
	seedCapacityOwned(t, st, box.Project, networklimits.MaxProjectBoxes-2)
	seedCapacityOwned(t, st, "other-project", 1)
	// Includes the original unresolved Create, leaves exactly one slot.
	got, e := st.ReadProjectBoxCapacity(ctx, box.Project)
	if e != nil || got.Remaining != 1 || got.PendingCreates != 1 || got.Owned != networklimits.MaxProjectBoxes-2 {
		t.Fatal(got, e)
	}
	type result struct {
		a Admission
		r Receipt
		e error
	}
	ch := make(chan result, 2)
	for i := 0; i < 2; i++ {
		next := a
		next.Key = fmt.Sprintf("capacity-concurrent-create-%d", i)
		next.ResourceKey = fmt.Sprintf("box:capacity-%d", i)
		next.SafeIntent = []byte(fmt.Sprintf("{\"name\":\"capacity-%d\"}", i))
		go func(in Admission) { r, e := st.Admit(ctx, in); ch <- result{in, r, e} }(next)
	}
	successes, denials := 0, 0
	for i := 0; i < 2; i++ {
		x := <-ch
		if errors.Is(x.e, ErrProjectCapacity) {
			denials++
			continue
		}
		if x.e != nil {
			t.Fatal(x.e)
		}
		successes++
		replay, e := st.Admit(ctx, x.a)
		if e != nil || !replay.Duplicate || replay.ID != x.r.ID {
			t.Fatal(replay, e)
		}
	}
	if successes != 1 || denials != 1 {
		t.Fatal(successes, denials)
	}
	// Reconciliation transfers the same slot from pending to owned.
	if e := st.Claim(ctx, original.ID); e != nil {
		t.Fatal(e)
	}
	if e := st.NativeAck(ctx, original.ID, "native-capacity-original"); e != nil {
		t.Fatal(e)
	}
	if e := st.FinalizeVerifiedCreate(ctx, original.ID, box.Project, box.Name, box.UUID); e != nil {
		t.Fatal(e)
	}
	got, e = st.ReadProjectBoxCapacity(ctx, box.Project)
	if e != nil || got.Remaining != 0 || got.PendingCreates != 1 || got.Owned != networklimits.MaxProjectBoxes-1 {
		t.Fatal(got, e)
	}
	// Historical tombstoned instances are not live slots.
	if _, e = st.db.Exec("UPDATE owned_boxes SET state='DELETED' WHERE uuid=?", box.UUID); e != nil {
		t.Fatal(e)
	}
	got, e = st.ReadProjectBoxCapacity(ctx, box.Project)
	if e != nil || got.Remaining != 1 {
		t.Fatal(got, e)
	}
}

func TestProjectCapacityUnknownCreateStillOccupiesSlot(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	a, box, original := s4CreateDeleteFixture(t, st)
	seedCapacityOwned(t, st, box.Project, networklimits.MaxProjectBoxes-1)
	if e := st.Claim(ctx, original.ID); e != nil {
		t.Fatal(e)
	}
	if e := st.Unknown(ctx, original.ID, "DISPATCHING", "NATIVE_OUTCOME_AMBIGUOUS"); e != nil {
		t.Fatal(e)
	}
	// Even a released outcome is not proof that the native Box does not exist.
	if _, e := st.db.Exec("INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES(?,?,'test','RELEASED','fixture')", original.ID, time.Now().UnixNano()); e != nil {
		t.Fatal(e)
	}
	got, e := st.ReadProjectBoxCapacity(ctx, box.Project)
	if e != nil || got.Remaining != 0 || got.PendingCreates != 1 {
		t.Fatal(got, e)
	}
	a.Key = "capacity-new-create-after-unknown"
	a.ResourceKey = "box:capacity-new"
	a.SafeIntent = []byte("{\"name\":\"capacity-new\"}")
	if _, e := st.Admit(ctx, a); !errors.Is(e, ErrProjectCapacity) {
		t.Fatal(e)
	}
	// The original is replayable at capacity without admitting anything new.
}
