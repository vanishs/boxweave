package state

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestH2AScopedDrainActorNotBearer(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	a := ScopedDrainActor{ID: strings.Repeat("a", 32), Expires: time.Now().Add(time.Hour).Unix()}
	if e := s.EnsureScopedDrainActor(ctx, a); e != nil {
		t.Fatal(e)
	}
	if e := s.EnsureScopedDrainActor(ctx, a); e != nil {
		t.Fatal(e)
	}
	if e := s.RequireActiveTokenID(ctx, a.ID); e != nil {
		t.Fatal(e)
	}
	if _, e := s.VerifyToken(ctx, "boxweave-scoped-drain-no-issued-bearer-v1|"+a.ID); !errors.Is(e, ErrRevoked) {
		t.Fatal("internal identity accepted as bearer", e)
	}
	changed := a
	changed.Expires++
	if e := s.EnsureScopedDrainActor(ctx, changed); !errors.Is(e, ErrConflict) {
		t.Fatal("changed actor accepted", e)
	}
	if _, e := s.ScopedDrainOriginal(ctx, "user-1001", a.ID, "scoped-drain-"+a.ID+"-0"); !errors.Is(e, ErrNotFound) {
		t.Fatal("missing original accepted", e)
	}
}

func TestH2AScopedDrainWitnessRefusesUnknown(t *testing.T) {
	ctx := context.Background()
	s, b, _ := n3RetiredFixture(t)
	l, e := s.ReadScopedDrainWitness(ctx, b.Project)
	if e != nil || len(l.Boxes) != 2 {
		t.Fatal(l, e)
	}
	s2, _, id := pr178ScopedReadyToStage(t, true)
	if e = s2.StageAndClaimScopedPeerOriginal(ctx, id); e != nil {
		t.Fatal(e)
	}
	if e = s2.Unknown(ctx, id, "DISPATCHING", "SCOPED_PEER_V2_ROOT_OUTCOME_UNVERIFIED"); e != nil {
		t.Fatal(e)
	}
	if _, e = s2.ReadScopedDrainWitness(ctx, "user-1001"); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("unknown original not fenced", e)
	}
}
