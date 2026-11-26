package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"github.com/vanishs/boxweave/internal/diagnostic"
	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/recovery"
	"github.com/vanishs/boxweave/internal/state"
)

const reviewedInputMax = 65536

type reviewedApplyRequest struct {
	Plan          diagnostic.RecoverPlan
	Reviewer, Key string
}
type reviewedApplyResult struct {
	ReviewSHA256    string                          `json:"review_sha256"`
	Recovery        state.RecoveryRecord            `json:"recovery"`
	Postcheck       state.ReviewedRecoveryPostcheck `json:"postcheck"`
	NativeRetried   bool                            `json:"native_retried"`
	NativePostcheck string                          `json:"native_postcheck"`
}

func decodeReviewedInput(r io.Reader, v any) error {
	b, e := io.ReadAll(io.LimitReader(r, reviewedInputMax+1))
	if e != nil || len(b) > reviewedInputMax {
		return state.ErrInvalid
	}
	// Unknown fields or a second JSON value must not select a different leaf.
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(v); e != nil {
		return state.ErrInvalid
	}
	var x any
	if e = d.Decode(&x); e != io.EOF {
		return state.ErrInvalid
	}
	return nil
}
func reviewedRequest(q reviewedApplyRequest) state.RecoveryRequest {
	b := q.Plan.ReviewedBinding()
	return state.RecoveryRequest{Layer: q.Plan.Scope, LocalReviewer: q.Reviewer, OriginalID: q.Plan.OriginalID, Key: q.Key, Review: &b}
}
func validReviewedEnvelope(q reviewedApplyRequest) bool {
	p := q.Plan
	if p.ObserverUID != 0 || !p.ApplyAvailable || !p.ReviewMatches(time.Unix(0, p.IssuedNS)) || !state.SafeDiagnosticProject(p.Scope) || !diagnostic.ValidIssueID(p.IssueID) || p.Issue == nil || p.Issue.ID != p.IssueID || p.Issue.OriginalID != p.OriginalID || diagnostic.IssueID(p.Scope, p.Issue.Domain, p.Issue.Code, p.Issue.Resource) != p.IssueID || p.Witness == nil || p.Witness.OriginalID != p.OriginalID {
		return false
	}
	return state.ValidateReviewedRecoveryRequest(reviewedRequest(q)) == nil
}
func fixedRecoveryArgs(project string) doctorArgs {
	return doctorArgs{DB: "/var/lib/boxweave/ledger.sqlite", Socket: "/var/lib/incus/unix.socket.user", Project: project, ServiceUser: "boxweave"}
}

func recoverApplyCLI(args []string) error {
	f := flag.NewFlagSet("recover --apply", flag.ContinueOnError)
	apply := f.Bool("apply", false, "execute one root-reviewed recovery plan read from stdin")
	issue := f.String("issue", "", "exact reviewed Doctor issue ID")
	project := f.String("project", "", "exact installed confined Project")
	confirm := f.String("confirm-review-sha256", "", "exact approved plan digest")
	approved := f.Bool("approve-local", false, "explicit host-local root authorization")
	reviewer := f.String("reviewer", "", "recorded local operator identity")
	key := f.String("recovery-key", "", "separate durable idempotency key")
	if e := f.Parse(args); e != nil {
		return e
	}
	if !*apply || !*approved || os.Geteuid() != 0 || f.NArg() != 0 || !diagnostic.ValidIssueID(*issue) || !state.SafeDiagnosticProject(*project) || *confirm == "" || *reviewer == "" || *key == "" {
		return errors.New("apply requires root, --approve-local, exact issue/project, review digest, reviewer and separate recovery key")
	}
	var p diagnostic.RecoverPlan
	if e := decodeReviewedInput(os.Stdin, &p); e != nil {
		return e
	}
	q := reviewedApplyRequest{Plan: p, Reviewer: *reviewer, Key: *key}
	if p.IssueID != *issue || p.Scope != *project || p.ReviewSHA256 != *confirm || !validReviewedEnvelope(q) {
		return errors.New("reviewed envelope or explicit confirmation does not match")
	}
	return applyReviewedEnvelope(q)
}

func applyReviewedEnvelope(q reviewedApplyRequest) error {
	if os.Geteuid() != 0 || !validReviewedEnvelope(q) {
		return state.ErrInvalid
	}
	p := q.Plan
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	a := fixedRecoveryArgs(p.Scope)
	uid, gid, e := install.ServiceIdentity(doctorInstallPlan(a))
	if e != nil || uid == 0 || a.Project != "user-"+strconv.Itoa(uid) {
		return state.ErrInvalid
	}
	// Exact-key lost-ACK lookup is read-only even after expiry. It cannot
	// authorize a new effect, and is scoped to the original local reviewer.
	snap, e := state.OpenDiagnosticSnapshot(ctx, a.DB)
	if e != nil {
		return e
	}
	r := reviewedRequest(q)
	prior, e := snap.ReplayReviewedRecovery(ctx, r)
	if e == nil {
		post, pe := snap.ReadReviewedRecoveryPostcheck(ctx, r, prior)
		ce := snap.Close()
		if pe != nil || ce != nil {
			post.State = "UNVERIFIED"
		}
		return outputReviewedResult(q, prior, post)
	}
	if ce := snap.Close(); ce != nil {
		return ce
	}
	if !errors.Is(e, state.ErrNotFound) {
		return e
	}
	if !p.ReviewMatches(time.Now()) {
		return errors.New("review expired; no new recovery admitted")
	}
	release, e := acquireMaintenanceLock()
	if e != nil {
		return e
	}
	defer release()
	if e = install.CheckScopedDrainFence(); e != nil {
		return e
	}
	if e = install.CheckOfflineRetirementFence(); e != nil {
		return e
	}
	if e = install.CheckScopedRetirementFence(); e != nil {
		return e
	}
	fresh := collectRecoverPlan(ctx, a, p.IssueID)
	if !p.MatchesFresh(fresh, time.Now()) {
		return errors.New("current evidence differs from approved review; no new recovery admitted")
	}
	// The current Root proof was verified by the fresh plan. Recheck static
	// identity before delegating; the worker repeats restricted identity and
	// native evidence and the transaction checks original/ledger again.
	identity := install.ObserveDiagnosticInstallation(ctx, doctorInstallPlan(a), a.DB)
	if identity.Mode != p.Mode || identity.IdentitySHA256 != p.InstallationSHA256 || !p.ReviewMatches(time.Now()) {
		return fmt.Errorf("reviewed recovery: root installation identity changed: %w", state.ErrConflict)
	}
	return spawnReviewedApply(ctx, q, uid, gid)
}
func spawnReviewedApply(ctx context.Context, q reviewedApplyRequest, uid, gid int) error {
	g, e := user.LookupGroup("incus")
	if e != nil {
		return e
	}
	ig, e := strconv.ParseUint(g.Gid, 10, 32)
	if e != nil || ig == 0 {
		return state.ErrInvalid
	}
	pair, e := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if e != nil {
		return e
	}
	parent := os.NewFile(uintptr(pair[0]), "apply-root")
	child := os.NewFile(uintptr(pair[1]), "apply-child")
	defer parent.Close()
	defer child.Close()
	cmd := exec.CommandContext(ctx, "/proc/self/exe", "recover-apply-worker")
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), Groups: recoveryGroups(uint32(gid), uint32(ig))}}
	cmd.ExtraFiles = []*os.File{child}
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=/home/boxweave", "XDG_CONFIG_HOME=/home/boxweave/.config", "LANG=C"}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e = cmd.Start(); e != nil {
		return e
	}
	child.Close()
	if e = json.NewEncoder(parent).Encode(q); e != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return e
	}
	_ = syscall.Shutdown(pair[0], syscall.SHUT_WR)
	// On a lost ACK inspect/repeat the same exact reviewed key. Do not
	// substitute a new recovery key or retry the original native operation.
	return cmd.Wait()
}
func recoverApplyWorker() error {
	if os.Geteuid() == 0 {
		return state.ErrInvalid
	}
	peer, e := syscall.GetsockoptUcred(3, syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if e != nil || peer.Uid != 0 || peer.Pid <= 0 {
		return errors.New("root-reviewed Unix socket peer required")
	}
	f := os.NewFile(3, "trusted-apply-request")
	if f == nil {
		return state.ErrInvalid
	}
	defer f.Close()
	var q reviewedApplyRequest
	if e = decodeReviewedInput(f, &q); e != nil || !validReviewedEnvelope(q) {
		return state.ErrInvalid
	}
	p := q.Plan
	a := fixedRecoveryArgs("user-" + strconv.Itoa(os.Geteuid()))
	uid, _, e := install.ServiceIdentity(doctorInstallPlan(a))
	if e != nil || uid != os.Geteuid() || p.Scope != a.Project {
		return state.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	recheck := func(ctx context.Context) error {
		i := install.ObserveDiagnosticInstallation(ctx, doctorInstallPlan(a), a.DB)
		if i.Candidate != p.Mode || i.InstalledIdentity != "VERIFIED" || i.Runtime != "VERIFIED" || i.IdentitySHA256 != p.InstallationSHA256 || i.Mode == "CONFLICT" || p.Mode == "LEGACY" && i.Mode != "LEGACY" || !p.ReviewMatches(time.Now()) {
			return fmt.Errorf("reviewed recovery: restricted installation identity changed: %w", state.ErrConflict)
		}
		if p.OriginalAction == "port_forward_create" {
			policy, ok := install.ObserveDiagnosticForwardPolicy(ctx, doctorInstallPlan(a), i)
			if !ok || policy != p.EffectiveForwardPolicy {
				return state.ErrConflict
			}
		}
		return nil
	}
	// The worker must never open/create an absent source DB. The prior
	// root review cannot be used as authority over a replacement scope.
	if e = recheck(ctx); e != nil {
		return e
	}
	db, e := state.OpenExistingRecovery(a.DB)
	if e != nil {
		return e
	}
	defer db.Close()
	policy := ""
	if p.OriginalAction == "port_forward_create" {
		if p.EffectiveForwardPolicy == "all" {
			policy = "all"
		} else if p.EffectiveForwardPolicy != "disabled" {
			return state.ErrConflict
		}
	}
	n := native.Reader{Project: a.Project, Socket: a.Socket, ExternalForwardAllow: policy}
	if st := doctorNativeActive(ctx, n); !st.Ready {
		return state.ErrResourceLocked
	}
	svc := recovery.Service{Store: db, Native: n, Layer: a.Project, PeerWitness: recovery.ObserveCompletedPeer}
	if p.Mode == "SCOPED_V2" {
		svc.ScopedPeerWitness = install.NativeScopedPeerRecoverCall
	}
	r := reviewedRequest(q)
	rec, e := svc.ReconcileReviewed(ctx, r, recheck)
	if e != nil {
		// Concurrent/lost acknowledgement may have committed the same key.
		// A receipt read is the only fallback; no native operation is retried.
		if prior, pe := db.ReplayRecovery(ctx, r); pe == nil {
			rec = prior
		} else {
			return e
		}
	}
	post, pe := db.ReadReviewedRecoveryPostcheck(ctx, r, rec)
	if pe != nil {
		post.State = "UNVERIFIED"
	}
	return outputReviewedResult(q, rec, post)
}
func outputReviewedResult(q reviewedApplyRequest, rec state.RecoveryRecord, post state.ReviewedRecoveryPostcheck) error {
	// Do not claim a current packet/native witness after settlement; the
	// native proof immediately before commit was independently re-observed.
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if e := enc.Encode(reviewedApplyResult{ReviewSHA256: q.Plan.ReviewSHA256, Recovery: rec, Postcheck: post, NativePostcheck: "NOT_REPROBED_AFTER_LEDGER_COMMIT"}); e != nil {
		return e
	}
	if post.State != "DURABLE_ORIGINAL_POSTSTATE_VERIFIED" {
		return errors.New("durable receipt exists; current postcheck not verified; inspect same receipt")
	}
	return nil
}
