package install

import (
	"errors"
	"github.com/vanishs/boxweave/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIPv4CompactionRootJournalExclusiveIntent(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root-owned temporary journal fixture")
	}
	path := filepath.Join(t.TempDir(), "intent.json")
	plan := retirementCompactionPlan{Original: strings.Repeat("a", 32), Project: "user-1001", Bridge: "bwbr0", ACL: "bw-peer", PolicySHA256: strings.Repeat("b", 64), BeforeSHA256: strings.Repeat("c", 64), AfterSHA256: strings.Repeat("d", 64)}
	j := retirementCompactionJournal{Schema: 1, Status: "DISPATCH_UNKNOWN", Plan: plan, PlanSHA256: compactionPlanDigest(plan)}
	if e := compactionWriteIntent(path, j); e != nil {
		t.Fatal(e)
	}
	if e := compactionWriteIntent(path, j); e == nil {
		t.Fatal("duplicate dispatch intent accepted")
	}
	if got, e := compactionLoadJournal(path); e != nil || got != j {
		t.Fatal(got, e)
	}
	wrong := j
	wrong.Plan.ACL = "other"
	wrong.PlanSHA256 = compactionPlanDigest(wrong.Plan)
	if e := compactionComplete(path, wrong); e == nil {
		t.Fatal("foreign completion accepted")
	}
	if e := compactionComplete(path, j); e != nil {
		t.Fatal(e)
	}
	j.Status = "COMPLETE_VERIFIED"
	if got, e := compactionLoadJournal(path); e != nil || got != j {
		t.Fatal(got, e)
	}
	if e := compactionWriteIntent(path, j); e == nil {
		t.Fatal("verified journal overwritten")
	}
	if e := os.Chmod(path, 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := compactionLoadJournal(path); e == nil {
		t.Fatal("untrusted permissions accepted")
	}
}

func TestIPv4CompactionRootScopeNarrow(t *testing.T) {
	original := strings.Repeat("a", 32)
	cfg := NativePeerRootScope{Approved: true, RetirementReview: true, RetirementCompact: true, DBPath: nativePeerRootInstalledDB, Project: "user-1001", Bridge: "bwbr0", ACL: "bw-peer"}
	unit, e := NativePeerRootUnitText(cfg)
	if e != nil || !strings.Contains(unit, " -retirement-review -retirement-compact -approve-local") || strings.Contains(unit, " -peer-plan-sha256 ") {
		t.Fatal(unit, e)
	}
	if nativePeerRootApprovedRequest(cfg, NativePeerRootRequest{OperationID: original, PlanSHA256: strings.Repeat("b", 64)}) {
		t.Fatal("ordinary Peer operation authorized")
	}
	for _, r := range []NativePeerRootRequest{
		{OperationID: original, RetirementPreview: true, RetirementReview: true},
		{OperationID: original, RetirementPreview: true, PlanSHA256: strings.Repeat("b", 64)},
		{OperationID: original, RetirementPreview: true, ReconcileApproved: true},
	} {
		if nativePeerRootValidateRequest(r) == nil {
			t.Fatal("ambiguous request", r)
		}
	}
	cfg.RetirementReview = false
	if nativePeerRootScopeApproved(cfg) {
		t.Fatal("compaction without explicit review allowed")
	}
}

func TestIPv4RetirementInstallerUnitIsolation(t *testing.T) {
	cfg := NativePeerRootScope{Approved: true, RetirementReview: true, RetirementCompact: true, DBPath: nativePeerRootInstalledDB, Project: "user-1001", Bridge: "bwbr0", ACL: "bw-peer"}
	body, e := NativePeerRootUnitText(cfg)
	if e != nil || !strings.Contains(body, "PartOf=boxweave.service") || !strings.Contains(body, "BindsTo=boxweave.service") || peerRootSocketForScope(cfg) != nativeRetirementSocketPath {
		t.Fatal(body, e)
	}
	cfg.OriginalID = strings.Repeat("a", 32)
	cfg.ApprovedPlanSHA = strings.Repeat("b", 64)
	if peerRootSocketForScope(cfg) != nativePeerRootSocketPath {
		t.Fatal("ordinary exact Peer uses retirement socket")
	}
	p := Plan{Project: "user-1001", RetirementTarget: &NativePrepareTarget{Project: "user-1001", Network: "bwbr0", ACL: "bw-peer"}}
	if strings.Contains(unit(p), nativeRetirementUnitName) || !strings.Contains(unit(p), "Wants="+scopedLifecycleUnitName) {
		t.Fatal("legacy retirement dependency must not be installed")
	}
	p.RetirementTarget = nil
	if strings.Contains(unit(p), nativeRetirementUnitName) {
		t.Fatal("legacy manual install gained root scope")
	}
	if NativeRetirementFailureClass(retirementFailure("RETIREMENT_BRIDGE_NOT_QUIESCENT", nil)) != "RETIREMENT_BRIDGE_NOT_QUIESCENT" || retirementWireClass("private root inventory") != "RETIREMENT_NATIVE_PROOF_UNAVAILABLE" {
		t.Fatal("unsafe diagnostic class")
	}
}

func TestIPv4CompactionUnknownResumeReadOnly(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("root-owned temporary journal fixture")
	}
	path := filepath.Join(t.TempDir(), "intent.json")
	c := state.IPv4RetirementCandidate{DeleteOperationID: strings.Repeat("a", 32), Project: "user-1001", Bridge: "bwbr0", ACL: "bw-peer"}
	plan := retirementCompactionPlan{Original: c.DeleteOperationID, Project: c.Project, Bridge: c.Bridge, ACL: c.ACL, PolicySHA256: strings.Repeat("b", 64), BeforeSHA256: strings.Repeat("c", 64), AfterSHA256: strings.Repeat("d", 64)}
	j := retirementCompactionJournal{Schema: 1, Status: "DISPATCH_UNKNOWN", Plan: plan, PlanSHA256: compactionPlanDigest(plan)}
	if e := compactionWriteIntent(path, j); e != nil {
		t.Fatal(e)
	}
	calls := 0
	prove := func(want retirementCompactionPlan) (NativeIPv4RetirementProof, error) {
		calls++
		if want != plan {
			t.Fatal("wrong plan")
		}
		return NativeIPv4RetirementProof{}, errors.New("unchanged before policy")
	}
	if _, e := compactionResumeJournal(path, j, j.PlanSHA256, c, prove); e == nil || calls != 1 {
		t.Fatal(e, calls)
	}
	if got, e := compactionLoadJournal(path); e != nil || got != j {
		t.Fatal(got, e)
	}
	proof := NativeIPv4RetirementProof{Candidate: c, SHA256: strings.Repeat("e", 64)}
	verified := func(want retirementCompactionPlan) (NativeIPv4RetirementProof, error) {
		calls++
		if want != plan {
			t.Fatal("wrong plan")
		}
		return proof, nil
	}
	if _, e := compactionResumeJournal(path, j, strings.Repeat("f", 64), c, verified); e == nil || calls != 1 {
		t.Fatal(e, calls)
	}
	got, e := compactionResumeJournal(path, j, j.PlanSHA256, c, verified)
	if e != nil || got != proof || calls != 2 {
		t.Fatal(got, e, calls)
	}
	j.Status = "COMPLETE_VERIFIED"
	if actual, e := compactionLoadJournal(path); e != nil || actual != j {
		t.Fatal(actual, e)
	}
	if _, e := compactionResumeJournal(path, j, j.PlanSHA256, c, verified); e != nil || calls != 3 {
		t.Fatal(e, calls)
	}
}
