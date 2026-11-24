package install

import (
	"reflect"
	"testing"
)

func TestPR178FreshScopedPeerGuardedEffectSequence(t *testing.T) {
	grant, _, _, _ := pr178CASPlan(t)
	got, e := nativePeerScopedRootEffectSequence(grant)
	if e != nil {
		t.Fatal(e)
	}
	expected := []nativePeerScopedEffectKind{
		nativePeerScopedACLSource, nativePeerScopedACLTarget,
		nativePeerScopedExactConntrack, nativePeerScopedGuardBatch,
	}
	if len(got) != 4 {
		t.Fatalf("incomplete fresh grant effect plan: %+v", got)
	}
	for i, want := range expected {
		if got[i].Kind != want {
			t.Fatal("unsafe fresh grant order", got)
		}
	}
	for _, s := range got[:3] {
		if s.GuardMustMatchSHA256 != grant.GuardBefore.DigestSHA256 {
			t.Fatal("grant may write ACL only while kernel still denies A/B")
		}
	}
	if got[0].OwnerUUID != grant.SourceUUID || got[1].OwnerUUID != grant.TargetUUID ||
		got[2].OwnerUUID != "" || got[3].OwnerUUID != "" {
		t.Fatal("fresh grant ordered unrelated owner")
	}
	if got[3].GuardMustMatchSHA256 != grant.GuardBefore.DigestSHA256 {
		t.Fatal("grant must open guard only after preparation and purge under DROP")
	}
	revoke := pr178CASReverse(grant)
	got, e = nativePeerScopedRootEffectSequence(revoke)
	if e != nil {
		t.Fatal(e)
	}
	expected = []nativePeerScopedEffectKind{
		nativePeerScopedGuardBatch, nativePeerScopedExactConntrack,
		nativePeerScopedACLSource, nativePeerScopedACLTarget,
	}
	if len(got) != 4 {
		t.Fatal("incomplete revoke plan", got)
	}
	for i, want := range expected {
		if got[i].Kind != want || got[i].GuardMustMatchSHA256 != revoke.GuardAfter.DigestSHA256 && i > 0 {
			t.Fatalf("revoke allowed unguarded A/B or C/D update at stage %d: %+v", i, got)
		}
	}
	if got[2].OwnerUUID != revoke.SourceUUID || got[3].OwnerUUID != revoke.TargetUUID {
		t.Fatal("revoke touched unrelated ACL")
	}
}

func TestPR178FreshScopedEffectSequenceRejectsUnprovenOwnerOrCDDelta(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*NativePeerScopedRootTransition)
	}{
		{"intent-polarity-mismatch", func(p *NativePeerScopedRootTransition) { p.IntentAllow = false }},
		{"boot-map-key-substitution", func(p *NativePeerScopedRootTransition) {
			delete(p.OriginalBootAutostart, pr178C)
			p.OriginalBootAutostart["foreign"] = "false"
		}},
		{"invalid-boot-value", func(p *NativePeerScopedRootTransition) { p.OriginalBootAutostart[pr178C] = "invalid" }},
		{"foreign-acl-both-images", func(p *NativePeerScopedRootTransition) {
			b, a := p.ExpectedBeforeByUUID[pr178C], p.ExpectedAfterByUUID[pr178C]
			b.Name, a.Name = "foreign", "foreign"
			p.ExpectedBeforeByUUID[pr178C], p.ExpectedAfterByUUID[pr178C] = b, a
		}},
		{"unknown-original", func(p *NativePeerScopedRootTransition) { p.OriginalID = "not-an-original" }},
		{"same-owner", func(p *NativePeerScopedRootTransition) { p.TargetUUID = p.SourceUUID }},
		{"foreign-source", func(p *NativePeerScopedRootTransition) { p.SourceUUID = pr178C }},
		{"forged-ACL-identity", func(p *NativePeerScopedRootTransition) { p.SourceACLName = "foreign" }},
		{"missing-tenant-ACL", func(p *NativePeerScopedRootTransition) { delete(p.ExpectedBeforeByUUID, pr178D) }},
		{"missing-tenant-autostart", func(p *NativePeerScopedRootTransition) { delete(p.OriginalBootAutostart, pr178C) }},
		{"C-ACL-change", func(p *NativePeerScopedRootTransition) {
			a := p.ExpectedAfterByUUID[pr178C]
			a.Description = "mutated C"
			p.ExpectedAfterByUUID[pr178C] = a
		}},
		{"A-no-op", func(p *NativePeerScopedRootTransition) {
			p.ExpectedAfterByUUID[pr178A] = p.ExpectedBeforeByUUID[pr178A]
		}},
		{"forged-nft-batch", func(p *NativePeerScopedRootTransition) { p.GuardAfter.Replace += "flush ruleset\n" }},
		{"forged-guard-digest", func(p *NativePeerScopedRootTransition) { p.GuardAfter.DigestSHA256 = "untrusted" }},
		{"global-partial-tenant", func(p *NativePeerScopedRootTransition) { p.UnrelatedACLsVerifiedUnchanged = 1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, _, _, _ := pr178CASPlan(t)
			tc.mutate(&p)
			got, e := nativePeerScopedRootEffectSequence(p)
			if e == nil || len(got) != 0 {
				t.Fatalf("invalid native effect stages unexpectedly created: %+v %v", got, e)
			}
		})
	}
}
func TestPR178FreshSequenceKeepsAllBootAutostartPreferences(t *testing.T) {
	p, _, _, _ := pr178CASPlan(t)
	original := map[string]string{}
	for k, v := range p.OriginalBootAutostart {
		original[k] = v
	}
	for _, plan := range []NativePeerScopedRootTransition{p, pr178CASReverse(p)} {
		_, e := nativePeerScopedRootEffectSequence(plan)
		if e != nil {
			t.Fatal(e)
		}
		if !reflect.DeepEqual(plan.OriginalBootAutostart, original) {
			t.Fatal("native effect planning modified user-configured autostart")
		}
	}
}
