package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestH4AExactAdmissionBlockerPriorityAndScope(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	_, token, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	a := Admission{Layer: "user-1001", ActorID: token.ActorID, TokenID: token.ID, Action: "box_create", RequestID: "test", SafeIntent: []byte(`{"name":"same"}`), ResourceKey: "box:same", Key: "h4a-admit-original-0001"}
	orig, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	b := a
	b.Key = "h4a-admit-new-000002"
	expect := func(want string) {
		_, e := s.Admit(ctx, b)
		var block AdmissionBlock
		if !errors.Is(e, ErrResourceLocked) || !errors.As(e, &block) || block.OriginalID != orig.ID || block.State != want {
			t.Fatal(block, e)
		}
	}
	expect("ADMITTED")
	if e = s.Claim(ctx, orig.ID); e != nil {
		t.Fatal(e)
	}
	expect("DISPATCHING")
	if e = s.Unknown(ctx, orig.ID, "DISPATCHING", "NATIVE_ACK_DURABILITY_UNVERIFIED"); e != nil {
		t.Fatal(e)
	}
	expect("UNKNOWN")
	got, e := s.Admit(ctx, a)
	if e != nil || !got.Duplicate || got.ID != orig.ID {
		t.Fatal(got, e)
	}
	b.Layer = "user-2002"
	got, e = s.Admit(ctx, b)
	if e != nil || got.ID == orig.ID {
		t.Fatal(got, e)
	}
}
