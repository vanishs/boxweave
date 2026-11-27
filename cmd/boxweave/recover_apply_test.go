package main

import (
	"bytes"
	"encoding/json"
	"github.com/vanishs/boxweave/internal/diagnostic"
	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/recovery"
	"github.com/vanishs/boxweave/internal/state"
	"strings"
	"testing"
	"time"
)

func TestPR177D4ReviewEnvelope(t *testing.T) {
	now := time.Now()
	project := "user-1001"
	id := strings.Repeat("a", 32)
	box := "12345678-1234-4abc-8def-123456789abc"
	report := diagnostic.NewReport(project)
	report.Installation = install.DiagnosticInstallation{Mode: "SCOPED_V2", IdentitySHA256: strings.Repeat("b", 64)}
	report.AddRoutedIssue("operations", "UNRESOLVED_OPERATION", id+"/opaque", "UNVERIFIED", "HOST_REVIEW_REQUIRED")
	report.Issues[0].OriginalID = id
	report.Issues[0].BoxID = box
	makePlan := func(now time.Time, uid int) diagnostic.RecoverPlan {
		p := diagnostic.NewRecoverPlan(report, report.Issues[0].ID, uid, now)
		if !p.SetOriginal(state.DiagnosticOriginal{Target: state.RecoveryTarget{ID: id, Layer: project, Action: "box_start", State: "UNKNOWN", UUID: box, Incarnation: box, UpdatedAt: 77}, Digest: strings.Repeat("c", 64), CredentialActive: true}) {
			t.Fatal(p)
		}
		p.LedgerSHA256 = strings.Repeat("d", 64)
		p.EvidenceEpochNS = now.UnixNano()
		p.SetWitness(recovery.PlanWitness{OriginalID: id, Kind: state.RecoverySettle, Reason: "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED", ProofSHA256: strings.Repeat("e", 64)})
		return p
	}
	p := makePlan(now, 0)
	q := reviewedApplyRequest{Plan: p, Reviewer: "d4-operator", Key: "d4-exact-review-key-01"}
	if !validReviewedEnvelope(q) || !p.MatchesFresh(makePlan(now.Add(time.Second), 0), now.Add(time.Second)) {
		t.Fatal(p)
	}
	if p.MatchesFresh(makePlan(now.Add(time.Minute), 0), now.Add(3*time.Minute)) {
		t.Fatal("expired review renewed")
	}
	q.Plan = makePlan(now, 1001)
	if validReviewedEnvelope(q) {
		t.Fatal("nonroot plan became local approval")
	}
	q.Plan = p
	q.Plan.Scope = "user-1002"
	if validReviewedEnvelope(q) {
		t.Fatal("tampered scope")
	}
	q.Plan = p
	q.Plan.Witness.ProofSHA256 = strings.Repeat("f", 64)
	if validReviewedEnvelope(q) {
		t.Fatal("tampered witness")
	}
	p = makePlan(now, 0)
	q.Plan = p
	raw, e := json.Marshal(q)
	if e != nil {
		t.Fatal(e)
	}
	var decoded reviewedApplyRequest
	if e = decodeReviewedInput(bytes.NewReader(raw), &decoded); e != nil || !validReviewedEnvelope(decoded) {
		t.Fatal(e)
	}
	for _, raw := range []string{string(raw) + "{}", `{"unknown":true}`, strings.Repeat("x", reviewedInputMax+1)} {
		if decodeReviewedInput(strings.NewReader(raw), &decoded) == nil {
			t.Fatal("ambiguous/oversize input accepted")
		}
	}
	t.Log("D4_EXACT_LOCAL_REVIEW_ENVELOPE_AND_FRESHNESS_PASS")
}
