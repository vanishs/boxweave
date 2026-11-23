package install

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"syscall"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/state"
)

// A fixed-scope plan contains no caller-chosen rules. The entire before and
// after policy is independently compiled from the complete protected ledger.
// Stable identity digest intentionally excludes global SQLite event counters;
// the controller holds a writer reservation across each proof and a durable
// Project fence between preview and application.
type retirementCompactionPlan struct {
	Original, Project, Bridge, ACL                  string
	PolicySHA256, BeforeSHA256, AfterSHA256, ACLTag string
	NativeScope, GlobalScope, RouteScope            string
}
type retirementCompactionJournal struct {
	Schema     int
	Status     string
	Plan       retirementCompactionPlan
	PlanSHA256 string
}

func compactionPlanDigest(p retirementCompactionPlan) string {
	raw, _ := json.Marshal(p)
	return retirementSHA(append([]byte("BoxWeave retired peer compaction plan v1\n"), raw...))
}
func compactionPolicyDigest(p state.IPv4RetirementPolicy) string {
	p.Candidate.OperationEpoch = 0
	p.Candidate.EventEpoch = 0
	raw, _ := json.Marshal(p)
	return retirementSHA(raw)
}
func compactionAfterPolicy(p state.IPv4RetirementPolicy) state.IPv4RetirementPolicy {
	p.Revoked = p.AfterRevoked
	return p
}
func compactionPolicyRead(ctx context.Context, s *state.Store, project, id string) (state.IPv4RetirementPolicy, error) {
	if os.Geteuid() != 0 || s == nil || !s.IsNetworkAuditReadOnly() {
		return state.IPv4RetirementPolicy{}, errors.New("protected root compaction reader required")
	}
	c, e := s.ReadIPv4RetirementCandidate(ctx, project, id)
	if e != nil || !c.RequiresPeerCompaction {
		return state.IPv4RetirementPolicy{}, errors.New("no exact retired peer original")
	}
	return s.ReadIPv4RetirementPolicy(ctx, c)
}
func compactionReadACL(root incus.InstanceServer, p state.IPv4RetirementPolicy) (*api.NetworkACL, string, api.NetworkACLsPost, api.NetworkACLsPost, error) {
	c := p.Candidate
	zero := api.NetworkACLsPost{}
	nw, _, e := root.GetNetwork(c.Bridge)
	if e != nil || nw == nil {
		return nil, "", zero, zero, errors.New("compaction bridge missing")
	}
	current, tag, e := root.GetNetworkACL(c.ACL)
	if e != nil || current == nil || tag == "" {
		return nil, "", zero, zero, errors.New("compaction ACL ETag missing")
	}
	t := NativePrepareTarget{Project: c.Project, Network: c.Bridge, ACL: c.ACL}
	before, e := NativePeerACLFromPolicySnapshot(t, nw, p.Active, p.Revoked)
	if e != nil {
		return nil, "", zero, zero, e
	}
	after, e := NativePeerACLFromPolicySnapshot(t, nw, p.Active, p.AfterRevoked)
	if e != nil {
		return nil, "", zero, zero, e
	}
	if len(p.AfterRevoked) >= len(p.Revoked) || !nativePeerACLPolicyChanges(before, after) {
		return nil, "", zero, zero, errors.New("compaction must remove an exact retired DROP policy")
	}
	return current, tag, before, after, nil
}
func compactionStableObservation(a, b retirementObservation) bool {
	return a.NativeScope == b.NativeScope && a.GlobalScope == b.GlobalScope && a.RouteScope == b.RouteScope && a.ACLHash == b.ACLHash && a.ForwardHistorySHA256 == b.ForwardHistorySHA256 && a.ConntrackScopes == 8 && b.ConntrackScopes == 8
}

// Before native PUT, no running guest is stopped by this capability. All
// existing native ownership, absence of old identity/flows and the original
// whole ACL are independently checked. If stopped-port or kernel evidence
// cannot be established, the preview refuses without setting a fence.
func compactionPreview(ctx context.Context, s *state.Store, root incus.InstanceServer, p state.IPv4RetirementPolicy) (retirementCompactionPlan, api.NetworkACLsPost, error) {
	var zero retirementCompactionPlan
	before, e := observeRetirement(ctx, root, p)
	if e != nil {
		return zero, api.NetworkACLsPost{}, e
	}
	middle, e := compactionPolicyRead(ctx, s, p.Candidate.Project, p.Candidate.DeleteOperationID)
	if e != nil || !reflect.DeepEqual(p, middle) {
		return zero, api.NetworkACLsPost{}, errors.New("compaction protected policy changed")
	}
	again, e := observeRetirement(ctx, root, middle)
	if e != nil || !compactionStableObservation(before, again) {
		return zero, api.NetworkACLsPost{}, errors.New("compaction preflight changed")
	}
	current, tag, old, want, e := compactionReadACL(root, p)
	if e != nil || !nativePeerACLMatches(current, old) {
		return zero, api.NetworkACLsPost{}, errors.New("compaction original ACL differs")
	}
	if before.ACLHash != nativePrepareStateDigest(old) {
		return zero, api.NetworkACLsPost{}, errors.New("compaction original policy digest differs")
	}
	c := p.Candidate
	plan := retirementCompactionPlan{Original: c.DeleteOperationID, Project: c.Project, Bridge: c.Bridge, ACL: c.ACL, PolicySHA256: compactionPolicyDigest(p), BeforeSHA256: nativePrepareStateDigest(old), AfterSHA256: nativePrepareStateDigest(want), ACLTag: tag, NativeScope: before.NativeScope, GlobalScope: before.GlobalScope, RouteScope: before.RouteScope}
	return plan, want, nil
}
func nativeIPv4CompactionPreview(ctx context.Context, s *state.Store, project, id string) (string, error) {
	p, e := compactionPolicyRead(ctx, s, project, id)
	if e != nil {
		return "", e
	}
	root, e := prepareClient(ctx)
	if e != nil {
		return "", e
	}
	plan, _, e := compactionPreview(ctx, s, root, p)
	if e != nil {
		return "", e
	}
	return compactionPlanDigest(plan), nil
}

func compactionJournalPath(id string) (string, error) {
	// Same strict original-ID validator; separate root-only namespace so a
	// completed Delete's retirement can never consume a Peer operation journal.
	if _, e := nativePeerConntrackOperationJournalPath(id); e != nil {
		return "", e
	}
	return "/var/lib/boxweave-ipv4-compaction-" + id + ".json", nil
}
func compactionJournalSafe(info os.FileInfo, dir bool) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 0 {
		return false
	}
	if dir {
		return info.IsDir() && info.Mode().Perm()&0022 == 0
	}
	return info.Mode().IsRegular() && info.Mode().Perm() == 0600 && st.Nlink == 1
}
func compactionSyncDir(path string) error {
	d, e := os.Open(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer d.Close()
	return d.Sync()
}
func compactionLoadJournal(path string) (retirementCompactionJournal, error) {
	var j retirementCompactionJournal
	f, e := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if e != nil {
		return j, e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil || !compactionJournalSafe(st, false) {
		return j, errors.New("unsafe root compaction journal")
	}
	raw, e := io.ReadAll(io.LimitReader(f, 65537))
	if e != nil || len(raw) > 65536 {
		return j, errors.New("compaction journal unavailable")
	}
	if json.Unmarshal(raw, &j) != nil || j.Schema != 1 || j.PlanSHA256 != compactionPlanDigest(j.Plan) || (j.Status != "DISPATCH_UNKNOWN" && j.Status != "COMPLETE_VERIFIED") {
		return j, errors.New("invalid original compaction journal")
	}
	return j, nil
}
func compactionWriteIntent(path string, j retirementCompactionJournal) error {
	st, e := os.Lstat(filepath.Dir(path))
	if e != nil || !compactionJournalSafe(st, true) || j.Schema != 1 || j.Status != "DISPATCH_UNKNOWN" || j.PlanSHA256 != compactionPlanDigest(j.Plan) {
		return errors.New("unsafe root compaction intent")
	}
	raw, e := json.Marshal(j)
	if e != nil {
		return e
	}
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if e != nil {
		return e
	}
	_, e = f.Write(raw)
	if e == nil {
		e = f.Sync()
	}
	cerr := f.Close()
	if e != nil {
		return e
	}
	if cerr != nil {
		return cerr
	}
	return compactionSyncDir(path)
}
func compactionComplete(path string, j retirementCompactionJournal) error {
	old, e := compactionLoadJournal(path)
	if e != nil || old != j || j.Status != "DISPATCH_UNKNOWN" {
		return errors.New("original compaction journal changed")
	}
	j.Status = "COMPLETE_VERIFIED"
	raw, e := json.Marshal(j)
	if e != nil {
		return e
	}
	if e = atomicWrite(path, append(raw, '\n'), 0600); e != nil {
		return e
	}
	return compactionSyncDir(path)
}
func compactionPostProof(ctx context.Context, s *state.Store, root incus.InstanceServer, p state.IPv4RetirementPolicy, plan retirementCompactionPlan) (NativeIPv4RetirementProof, error) {
	var zero NativeIPv4RetirementProof
	current, _, _, want, e := compactionReadACL(root, p)
	if e != nil || !nativePeerACLMatches(current, want) || nativePrepareStateDigest(want) != plan.AfterSHA256 {
		return zero, errors.New("compaction post-ACL not exact")
	}
	if compactionPolicyDigest(p) != plan.PolicySHA256 {
		return zero, errors.New("compaction protected policy changed")
	}
	after := compactionAfterPolicy(p)
	first, e := observeRetirement(ctx, root, after)
	if e != nil {
		return zero, e
	}
	mid, e := compactionPolicyRead(ctx, s, p.Candidate.Project, p.Candidate.DeleteOperationID)
	if e != nil || !reflect.DeepEqual(mid, p) {
		return zero, errors.New("compaction ledger changed")
	}
	second, e := observeRetirement(ctx, root, compactionAfterPolicy(mid))
	if e != nil || !compactionStableObservation(first, second) || first.ACLHash != plan.AfterSHA256 || first.GlobalScope != plan.GlobalScope || first.RouteScope != plan.RouteScope {
		return zero, errors.New("compaction post-native scope differs")
	}
	final, e := compactionPolicyRead(ctx, s, p.Candidate.Project, p.Candidate.DeleteOperationID)
	if e != nil || !reflect.DeepEqual(final, p) {
		return zero, errors.New("compaction final protected policy changed")
	}
	data, e := json.Marshal(struct {
		Candidate     state.IPv4RetirementCandidate
		PlanSHA256    string
		First, Second retirementObservation
	}{p.Candidate, compactionPlanDigest(plan), first, second})
	if e != nil {
		return zero, e
	}
	return NativeIPv4RetirementProof{Candidate: p.Candidate, SHA256: retirementSHA(append([]byte("BoxWeave retired peer compacted proof v1\n"), data...))}, nil
}

// Only reachable from the explicitly approved fixed-Project root socket.
// A durable SQLite fence must preexist. A root DISPATCH_UNKNOWN journal is
// never replayed, even if the native rules appear unchanged. Either a
// completed journal or independently verified exact poststate allows fresh
// read-only proof after controller/root ACK loss or restart.
func nativeIPv4CompactionApply(ctx context.Context, s *state.Store, project, id string) (NativeIPv4RetirementProof, error) {
	var zero NativeIPv4RetirementProof
	p, e := compactionPolicyRead(ctx, s, project, id)
	if e != nil {
		return zero, e
	}
	approved, e := s.ReadIPv4CompactionPlan(ctx, project, id)
	if e != nil {
		return zero, retirementFailure("RETIREMENT_COMPACTION_PROTECTED", errors.New("missing durable compaction fence"))
	}
	path, e := compactionJournalPath(id)
	if e != nil {
		return zero, e
	}
	root, e := prepareClient(ctx)
	if e != nil {
		return zero, e
	}
	prior, e := compactionLoadJournal(path)
	if e == nil {
		return compactionResumeJournal(path, prior, approved, p.Candidate, func(plan retirementCompactionPlan) (NativeIPv4RetirementProof, error) {
			return compactionPostProof(ctx, s, root, p, plan)
		})
	}
	if !errors.Is(e, os.ErrNotExist) {
		return zero, e
	}
	plan, want, e := compactionPreview(ctx, s, root, p)
	if e != nil {
		return zero, e
	}
	if compactionPlanDigest(plan) != approved {
		return zero, retirementFailure("RETIREMENT_COMPACTION_PROTECTED", errors.New("compaction plan drifted; preserve fence"))
	}
	// ETag and complete original policy must still match at point of use.
	current, tag, before, after, e := compactionReadACL(root, p)
	if e != nil || tag != plan.ACLTag || !nativePeerACLMatches(current, before) || !reflect.DeepEqual(want, after) {
		return zero, retirementFailure("RETIREMENT_COMPACTION_PROTECTED", errors.New("compaction baseline changed"))
	}
	if e = nativePeerBridgeKernelNoLivePorts(ctx, p.Candidate.Bridge); e != nil {
		return zero, e
	}
	pinned, e := s.ReadIPv4CompactionPlan(ctx, project, id)
	if e != nil || pinned != approved {
		return zero, retirementFailure("RETIREMENT_COMPACTION_PROTECTED", errors.New("compaction fence changed"))
	}
	j := retirementCompactionJournal{Schema: 1, Status: "DISPATCH_UNKNOWN", Plan: plan, PlanSHA256: approved}
	if e = compactionWriteIntent(path, j); e != nil {
		return zero, e
	}
	if e = root.UpdateNetworkACL(p.Candidate.ACL, want.NetworkACLPut, tag); e != nil {
		return zero, fmt.Errorf("compaction native effect uncertain: %w", e)
	}
	proof, e := compactionPostProof(ctx, s, root, p, plan)
	if e != nil {
		return zero, e
	}
	if e = compactionComplete(path, j); e != nil {
		return zero, e
	}
	return proof, nil
}

// Resume accepts only the exact durable original and a new independent
// poststate proof. There is no native writer in this path: absence of proof,
// including an unchanged before-policy, retains the original fence.
func compactionResumeJournal(path string, prior retirementCompactionJournal, approved string, c state.IPv4RetirementCandidate, prove func(retirementCompactionPlan) (NativeIPv4RetirementProof, error)) (NativeIPv4RetirementProof, error) {
	var zero NativeIPv4RetirementProof
	if prior.PlanSHA256 != approved || prior.PlanSHA256 != compactionPlanDigest(prior.Plan) || prior.Plan.Original != c.DeleteOperationID || prior.Plan.Project != c.Project || prior.Plan.Bridge != c.Bridge || prior.Plan.ACL != c.ACL || (prior.Status != "COMPLETE_VERIFIED" && prior.Status != "DISPATCH_UNKNOWN") || prove == nil {
		return zero, retirementFailure("RETIREMENT_COMPACTION_PROTECTED", errors.New("compaction original identity changed"))
	}
	proof, e := prove(prior.Plan)
	if e != nil || proof.Candidate != c || !validRetirementReviewSHA(proof.SHA256) {
		return zero, retirementFailure("RETIREMENT_COMPACTION_PROTECTED", e)
	}
	if prior.Status == "DISPATCH_UNKNOWN" {
		if e = compactionComplete(path, prior); e != nil {
			return zero, retirementFailure("RETIREMENT_COMPACTION_PROTECTED", e)
		}
	}
	return proof, nil
}
