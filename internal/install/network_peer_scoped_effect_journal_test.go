package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPR178N3WholeOriginalJournal(t *testing.T) {
	for _, allow := range []bool{true, false} {
		for _, fault := range []string{"none", "effect0", "effect1", "effect2", "effect3", "post0", "post1", "post2", "post3", "final", "pre-drift", "cancel"} {
			t.Run(fault+map[bool]string{true: "-grant", false: "-revoke"}[allow], func(t *testing.T) {
				p, _, _, _ := pr178CASPlan(t)
				if !allow {
					p = pr178CASReverse(p)
				}
				scope := scopedPeerEffectJournalScope{strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				dir := t.TempDir()
				uid := uint32(os.Getuid())
				prefix := 0
				reads := 0
				w := func(_ context.Context, i int) (string, error) {
					if i != prefix {
						t.Fatalf("witness prefix=%d actual=%d", i, prefix)
					}
					reads++
					if fault == "pre-drift" && reads == 2 {
						return strings.Repeat("d", 64), nil
					}
					if fault == "final" && prefix == 4 && reads == 10 {
						return "", errors.New("late drift")
					}
					if strings.HasPrefix(fault, "post") && prefix > 0 && fault == ("post"+string(rune('0'+prefix-1))) {
						return "", errors.New("postimage unavailable")
					}
					return nativePrepareStateDigest("prefix", i), nil
				}
				steps, e := nativePeerScopedRootEffectSequence(p)
				if e != nil {
					t.Fatal(e)
				}
				apply := func(_ context.Context, step nativePeerScopedEffectStep) error {
					if step != steps[prefix] {
						t.Fatal("wrong effect order", step)
					}
					path := filepath.Join(dir, "peer-"+p.OriginalID)
					d, e := nativePeerFreshJournalOpenAt(path, uid)
					if e != nil {
						t.Fatal(e)
					}
					r, e := scopedPeerEffectRead(d, uid, scopedPeerEffectRecordName(prefix, false))
					d.Close()
					if e != nil || r.Status != "DISPATCH_UNKNOWN" || r.Kind != step.Kind || r.OwnerUUID != step.OwnerUUID || r.PreimageSHA256 != nativePrepareStateDigest("prefix", prefix) {
						t.Fatal("effect before durable exact intent", r, e)
					}
					prefix++
					if fault == "cancel" {
						cancel()
					}
					if fault == ("effect" + string(rune('0'+prefix-1))) {
						return errors.New("effect applied, ACK lost")
					}
					return nil
				}
				receipt, e := nativeScopedPeerEffectsOnce(ctx, dir, uid, p, scope, w, apply)
				if fault == "none" {
					if e != nil || receipt.Status != "COMPLETE_VERIFIED" || receipt.VerifiedSteps != 4 || prefix != 4 {
						t.Fatal(receipt, e, prefix)
					}
					actual, e := nativeScopedPeerEffectsReadOnly(dir, uid, p, scope)
					if e != nil || actual != receipt {
						t.Fatal(actual, e)
					}
					if e = os.WriteFile(filepath.Join(dir, "peer-"+p.OriginalID, "extra"), []byte("unexpected"), 0600); e != nil {
						t.Fatal(e)
					}
					if _, e = nativeScopedPeerEffectsReadOnly(dir, uid, p, scope); e == nil {
						t.Fatal("extra journal accepted")
					}
				} else {
					if e == nil || receipt.Status == "COMPLETE_VERIFIED" {
						t.Fatal("uncertain effect completed", receipt, e)
					}
					if _, e = nativeScopedPeerEffectsReadOnly(dir, uid, p, scope); e == nil {
						t.Fatal("incomplete journal accepted")
					}
				}
				calls := prefix
				if _, e = nativeScopedPeerEffectsOnce(context.Background(), dir, uid, p, scope, func(context.Context, int) (string, error) { return nativePrepareStateDigest("prefix", 0), nil }, apply); e == nil || prefix != calls {
					t.Fatal("original replayed", e, prefix, calls)
				}
			})
		}
	}
}
