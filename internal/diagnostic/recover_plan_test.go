package diagnostic

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/recovery"
	"github.com/vanishs/boxweave/internal/state"
)

func d3Fixture() (Report, Issue, state.DiagnosticOriginal, recovery.PlanWitness) {
	r := NewReport("user-1001")
	r.Installation = install.DiagnosticInstallation{Mode: "LEGACY", IdentitySHA256: strings.Repeat("a", 64)}
	id := strings.Repeat("b", 32)
	box := "12345678-1234-4abc-8def-123456789abc"
	r.AddRoutedIssue("operations", "UNRESOLVED_OPERATION", id+"/opaque", "UNVERIFIED", "HOST_REVIEW_REQUIRED")
	r.Issues[0].OriginalID = id
	r.Issues[0].BoxID = box
	o := state.DiagnosticOriginal{Target: state.RecoveryTarget{ID: id, Layer: r.Scope, Action: "box_start", State: "UNKNOWN", UUID: box, Incarnation: box, UpdatedAt: 77, SafeIntent: "private-command"}, Digest: strings.Repeat("c", 64), Claims: []string{"box:private-name"}, CredentialActive: true}
	w := recovery.PlanWitness{OriginalID: id, Kind: state.RecoverySettle, Reason: "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED", ProofSHA256: strings.Repeat("d", 64)}
	return r, r.Issues[0], o, w
}
func TestPR177D3ReviewDigestAndRefusals(t *testing.T) {
	r, i, o, w := d3Fixture()
	now := time.Unix(100, 0)
	p := NewRecoverPlan(r, i.ID, 0, now)
	if !p.SetOriginal(o) {
		t.Fatal(p)
	}
	p.SetWitness(w)
	if p.Status != "REVIEW_READY" || !p.CanRepair || p.ApplyAvailable || p.NativeRetried || !p.ReviewMatches(now) || p.ReviewMatches(now.Add(RecoverPlanTTL)) {
		t.Fatal(p)
	}
	raw, e := json.Marshal(p)
	if e != nil || strings.Contains(string(raw), "private-") {
		t.Fatal("private evidence leaked", e)
	}
	var copy RecoverPlan
	if e = json.Unmarshal(raw, &copy); e != nil || !copy.ReviewMatches(now) {
		t.Fatal("review envelope not reproducible", e)
	}
	for _, change := range []func(*RecoverPlan){
		func(p *RecoverPlan) { p.Scope = "user-1002" }, func(p *RecoverPlan) { p.OriginalRevision++ }, func(p *RecoverPlan) { p.Incarnation = "other" }, func(p *RecoverPlan) { p.Leaf = "release" }, func(p *RecoverPlan) { p.ObserverUID = 1001 }, func(p *RecoverPlan) { p.ExpiresNS++ }, func(p *RecoverPlan) { p.InstallationSHA256 = strings.Repeat("e", 64) }, func(p *RecoverPlan) { p.ApplyAvailable = true },
	} {
		q := p
		change(&q)
		if q.ReviewMatches(now) {
			t.Fatal("changed review accepted", q)
		}
	}
	q := NewRecoverPlan(r, i.ID, 0, now)
	q.SetWitness(w)
	if q.CanRepair || q.ReviewSHA256 != "" {
		t.Fatal("no original proof", q)
	}
	for _, change := range []func(*state.DiagnosticOriginal){
		func(x *state.DiagnosticOriginal) { x.Target.Layer = "user-1002" }, func(x *state.DiagnosticOriginal) { x.Target.UUID = "other" }, func(x *state.DiagnosticOriginal) { x.Target.State = "DISPATCHING" }, func(x *state.DiagnosticOriginal) { x.Target.Released = true },
	} {
		q := NewRecoverPlan(r, i.ID, 0, now)
		x := o
		change(&x)
		if q.SetOriginal(x) {
			t.Fatal("invalid original accepted")
		}
		q.SetWitness(w)
		if q.CanRepair {
			t.Fatal(q)
		}
	}
	q = NewRecoverPlan(r, i.ID, 0, now)
	if !q.SetOriginal(o) {
		t.Fatal(q)
	}
	w.ProofSHA256 = ""
	w.Kind = state.RecoveryBlocked
	w.Reason = "ROOT_PROOF_MISSING"
	q.SetWitness(w)
	if q.CanRepair || q.ReviewSHA256 != "" {
		t.Fatal(q)
	}
	q = NewRecoverPlan(r, "d1-"+strings.Repeat("f", 64), 0, now)
	if q.Status != "BLOCKED_UNPROVEN" || q.Issue != nil {
		t.Fatal(q)
	}
	r.Installation.Mode = "SCOPED_V2"
	o.Target.Action = "box_peer_set"
	o.Target.ScopedPeer = true
	o.CredentialActive = false
	o.Target.SafeIntent = `{"schema":2,"project":"user-1001","bridge":"incusbr-1001","source_uuid":"12345678-1234-4abc-8def-123456789abc","source_incarnation":"12345678-1234-4abc-8def-123456789abc","target_uuid":"87654321-4321-4abc-8def-abcdef012345","target_incarnation":"87654321-4321-4abc-8def-abcdef012345"}`
	o.Claims = []string{"guard", "source-acl", "target-acl", "source-box", "target-box"}
	q = NewRecoverPlan(r, i.ID, 0, now)
	if q.SetOriginal(o) || q.Reason != "SCOPED_ORIGINAL_CREDENTIAL_NOT_ACTIVE" || q.ClaimCount != 5 || len(q.Resources) != 5 || len(q.AffectedBoxes) != 2 || q.OriginalCredentialActive == nil || *q.OriginalCredentialActive {
		t.Fatal(q)
	}
	o.Target.Action = "box_delete"
	o.Target.ScopedPeer = false
	o.CredentialActive = true
	q = NewRecoverPlan(r, i.ID, 0, now)
	if q.SetOriginal(o) || q.Reason != "SCOPED_LIFECYCLE_RECOVERY_UNSUPPORTED" {
		t.Fatal(q)
	}
	t.Log("D3_REVIEW_DIGEST_SCOPE_REVISION_EXPIRY_AND_UNSUPPORTED_PASS")
}
