package state

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestS4CReadinessAdmissionFenceReplayRecovery(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	st, e := Open(filepath.Join(dir, "state.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer st.Close()
	_, tok, e := st.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	box := "11111111-1111-4111-8111-111111111111"
	if e = st.RegisterOwnedBox(ctx, box, "user-1001", "bw-ready", box); e != nil {
		t.Fatal(e)
	}
	i, e := NormalizeReadinessIntent(ReadinessIntent{BoxID: box, Checks: []string{"running"}, TimeoutSeconds: 60})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(i)
	a := Admission{Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID, Key: "readiness-original-key-0001", Action: ReadinessAction, BoxUUID: box, Incarnation: box, ResourceKey: "box:bw-ready", SafeIntent: raw, RequestID: "request-1"}
	r, e := st.Admit(ctx, a)
	if e != nil || r.State != "OBSERVING" {
		t.Fatalf("admit: %v %v", r, e)
	}
	result, e := st.Readiness(ctx, r)
	if e != nil || result.Ready || result.Running != "unknown" || result.ExecReady != "not_checked" {
		t.Fatalf("initial: %+v %v", result, e)
	}
	dup, e := st.Admit(ctx, a)
	if e != nil || !dup.Duplicate || dup.ID != r.ID {
		t.Fatal("replay failed")
	}
	newWait := a
	newWait.Key = "another-readiness-key-0002"
	if _, e = st.Admit(ctx, newWait); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("parallel readiness accepted")
	}
	mutation := a
	mutation.Action = "box_start"
	mutation.Key = "independent-start-key-0001"
	mutation.SafeIntent = []byte("{}")
	m, e := st.Admit(ctx, mutation)
	if e != nil {
		t.Fatalf("observation blocked start: %v", e)
	}
	if e = st.ClaimReadiness(ctx, r.ID); !errors.Is(e, ErrResourceLocked) {
		t.Fatalf("probe raced mutation: %v", e)
	}
	if e = st.CancelQueued(ctx, a.Layer, m.ID); e != nil {
		t.Fatal(e)
	}
	if e = st.ClaimReadiness(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	mutation.Key = "independent-start-key-0002"
	if _, e = st.Admit(ctx, mutation); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("probe fence bypassed")
	}
	sample := NewReadinessSample(i)
	sample.Running = "ready"
	sample.CheckedAt = time.Now().UnixNano()
	if e = st.SaveReadiness(ctx, r.ID, "DISPATCHING", "SUCCEEDED", "", sample); e != nil {
		t.Fatal(e)
	}
	got, e := st.Get(ctx, r.ID)
	if e != nil || got.State != "SUCCEEDED" {
		t.Fatal(got, e)
	}
	proof, e := st.Readiness(ctx, got)
	if e != nil || !proof.Ready || proof.ExecReady != "not_checked" {
		t.Fatal(proof, e)
	}
	if _, e = st.Admit(ctx, mutation); e != nil {
		t.Fatal("terminal observation retained lock", e)
	}
	// Restart before any probe is safe; no fabricated native receipt.
	pending, e := st.Admit(ctx, newWait)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = st.Recover(ctx); e != nil {
		t.Fatal(e)
	}
	got, e = st.Get(ctx, pending.ID)
	if e != nil || got.State != "FAILED" || got.NativeOperationID != "" {
		t.Fatal(got, e)
	}
	// A restarted dispatch must retain an original UNKNOWN and a lock.
	newWait.Key = "third-readiness-key-0003"
	inFlight, e := st.Admit(ctx, newWait)
	if e != nil {
		t.Fatal(e)
	}
	if e = st.ClaimReadiness(ctx, inFlight.ID); e != nil {
		t.Fatal(e)
	}
	if e = st.NativeAck(ctx, inFlight.ID, "native-probe-receipt"); e != nil {
		t.Fatal(e)
	}
	if _, e = st.Recover(ctx); e != nil {
		t.Fatal(e)
	}
	got, e = st.Get(ctx, inFlight.ID)
	if e != nil || got.State != "UNKNOWN" || got.NativeOperationID != "native-probe-receipt" {
		t.Fatal(got, e)
	}
	if _, e = st.Admit(ctx, Admission{Layer: a.Layer, ActorID: a.ActorID, TokenID: a.TokenID, Key: "new-start-after-unknown", Action: "box_start", SafeIntent: []byte("{}"), BoxUUID: box, Incarnation: box, ResourceKey: a.ResourceKey, RequestID: "new"}); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("lost probe fence")
	}
}

func TestS4CReadinessIntentBounds(t *testing.T) {
	for _, i := range []ReadinessIntent{{}, {BoxID: "x", TimeoutSeconds: 601}, {BoxID: "x", Checks: []string{"exec", "exec"}}, {BoxID: "x", Checks: []string{"initialized"}}, {BoxID: "x", TimeoutSeconds: -1}} {
		if _, e := NormalizeReadinessIntent(i); e == nil {
			t.Fatal("invalid input accepted")
		}
	}
	i, e := NormalizeReadinessIntent(ReadinessIntent{BoxID: "x"})
	if e != nil || i.TimeoutSeconds != 300 || len(i.Checks) != 2 {
		t.Fatal(i, e)
	}
}
