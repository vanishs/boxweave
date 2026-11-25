package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vanishs/boxweave/internal/diagnostic"
	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/recovery"
	"github.com/vanishs/boxweave/internal/state"
)

func planFlagPresent(args []string) bool {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			continue
		}
		name := strings.TrimLeft(strings.SplitN(a, "=", 2)[0], "-")
		if name == "plan" || name == "guided" || name == "issue" || name == "apply" || name == "confirm-review-sha256" {
			return true
		}
	}
	return false
}
func recoverPlanCLI(args []string) error {
	for _, a := range args {
		if a == "--guided" || a == "-guided" {
			return recoverGuidedCLI(args)
		}
		if a == "--apply" || a == "-apply" || strings.HasPrefix(a, "--apply=") || strings.HasPrefix(a, "-apply=") {
			return recoverApplyCLI(args)
		}
	}

	f := flag.NewFlagSet("recover --plan", flag.ContinueOnError)
	plan := f.Bool("plan", false, "read-only issue-bound recovery plan")
	id := f.String("issue", "", "exact current Doctor issue ID")
	project := f.String("project", "", "exact confined project")
	db := f.String("db", "/var/lib/boxweave/ledger.sqlite", "protected local SQLite path")
	socket := f.String("incus-socket", "/var/lib/incus/unix.socket.user", "restricted Incus socket")
	service := f.String("service-user", "boxweave", "dedicated service account")
	if e := f.Parse(args); e != nil {
		return e
	}
	a, e := resolveDoctorScope(doctorArgs{DB: *db, Socket: *socket, Project: *project, ServiceUser: *service})
	if e != nil {
		return e
	}
	if !*plan || !diagnostic.ValidIssueID(*id) || f.NArg() != 0 || !state.SafeDiagnosticProject(a.Project) {
		return errors.New("exact --plan --issue and confined project required")
	}
	return runRecoverPlan(a, *id)
}

type planWorkerRequest struct {
	Project, OriginalID, OriginalSHA256, Mode, ForwardPolicy string
}

func planWitness(ctx context.Context, a doctorArgs, q planWorkerRequest) (recovery.PlanWitness, error) {
	var out recovery.PlanWitness
	if os.Geteuid() == 0 {
		return out, state.ErrInvalid
	}
	snap, e := state.OpenDiagnosticSnapshot(ctx, a.DB)
	if e != nil {
		return out, e
	}
	defer snap.Close()
	orig, e := snap.ReadOriginal(ctx, q.Project, q.OriginalID)
	if e != nil || orig.Digest != q.OriginalSHA256 {
		return out, state.ErrConflict
	}
	r := native.Reader{Project: a.Project, Socket: a.Socket, ExternalForwardAllow: q.ForwardPolicy}
	if st := doctorNativeActive(ctx, r); !st.Ready {
		return out, state.ErrResourceLocked
	}
	svc := recovery.Service{ReadStore: snap, Native: r, Layer: q.Project, PeerWitness: recovery.ObserveCompletedPeer}
	if q.Mode == "SCOPED_V2" {
		svc.ScopedPeerWitness = install.NativeScopedPeerRecoverCall
	}
	w, e := svc.InspectPlan(ctx, orig.Target)
	if e != nil {
		return out, e
	}
	// New capture after native observation. Same snapshot re-read alone
	// cannot establish that the original remains unchanged.
	after, e := state.OpenDiagnosticSnapshot(ctx, a.DB)
	if e != nil {
		return out, e
	}
	next, e := after.ReadOriginal(ctx, q.Project, q.OriginalID)
	ce := after.Close()
	if e != nil || ce != nil || next.Digest != orig.Digest {
		return out, state.ErrConflict
	}
	return w, nil
}

func recoverPlanWorker() error {
	if os.Geteuid() == 0 {
		return state.ErrInvalid
	}
	peer, e := syscall.GetsockoptUcred(3, syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if e != nil || peer.Uid != 0 || peer.Pid <= 0 {
		return errors.New("root observation peer required")
	}
	f := os.NewFile(3, "trusted-plan-observation")
	if f == nil {
		return state.ErrInvalid
	}
	defer f.Close()
	dec := json.NewDecoder(io.LimitReader(f, 8192))
	dec.DisallowUnknownFields()
	var q planWorkerRequest
	if e = dec.Decode(&q); e != nil {
		return state.ErrInvalid
	}
	var extra any
	if e = dec.Decode(&extra); e != io.EOF {
		return state.ErrInvalid
	}
	a := doctorArgs{DB: "/var/lib/boxweave/ledger.sqlite", Socket: "/var/lib/incus/unix.socket.user", Project: "user-" + strconv.Itoa(os.Geteuid()), ServiceUser: "boxweave"}
	uid, _, e := install.ServiceIdentity(doctorInstallPlan(a))
	if e != nil || uid != os.Geteuid() || q.Project != a.Project || q.Mode != "SCOPED_V2" && q.Mode != "LEGACY" {
		return state.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
	defer cancel()
	w, e := planWitness(ctx, a, q)
	if e != nil {
		return e
	}
	return json.NewEncoder(os.Stdout).Encode(w)
}

func restrictedPlanWitness(ctx context.Context, a doctorArgs, q planWorkerRequest) (recovery.PlanWitness, error) {
	var out recovery.PlanWitness
	// A candidate observation never delegates arbitrary DB/socket/project.
	if a.DB != "/var/lib/boxweave/ledger.sqlite" || a.Socket != "/var/lib/incus/unix.socket.user" || a.ServiceUser != "boxweave" {
		return out, state.ErrInvalid
	}
	if os.Geteuid() != 0 {
		if a.Project != "user-"+strconv.Itoa(os.Geteuid()) {
			return out, state.ErrInvalid
		}
		return planWitness(ctx, a, q)
	}
	uid, gid, e := install.ServiceIdentity(doctorInstallPlan(a))
	if e != nil || uid == 0 || a.Project != "user-"+strconv.Itoa(uid) {
		return out, state.ErrInvalid
	}
	g, e := user.LookupGroup("incus")
	if e != nil {
		return out, e
	}
	ig, e := strconv.ParseUint(g.Gid, 10, 32)
	if e != nil || ig == 0 {
		return out, state.ErrInvalid
	}
	pair, e := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if e != nil {
		return out, e
	}
	parent := os.NewFile(uintptr(pair[0]), "plan-root")
	child := os.NewFile(uintptr(pair[1]), "plan-child")
	defer parent.Close()
	defer child.Close()
	cmd := exec.CommandContext(ctx, "/proc/self/exe", "recover-plan-worker")
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), Groups: recoveryGroups(uint32(gid), uint32(ig))}}
	cmd.ExtraFiles = []*os.File{child}
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=/home/boxweave", "XDG_CONFIG_HOME=/home/boxweave/.config", "LANG=C"}
	stdout, e := cmd.StdoutPipe()
	if e != nil {
		return out, e
	}
	if e = cmd.Start(); e != nil {
		return out, e
	}
	child.Close()
	if e = json.NewEncoder(parent).Encode(q); e != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return out, e
	}
	_ = syscall.Shutdown(pair[0], syscall.SHUT_WR)
	b, e := io.ReadAll(io.LimitReader(stdout, 16385))
	if e != nil || len(b) > 16384 {
		cmd.Process.Kill()
		cmd.Wait()
		return out, state.ErrResourceLocked
	}
	if e = cmd.Wait(); e != nil {
		return out, e
	}
	if json.Unmarshal(b, &out) != nil || out.OriginalID != q.OriginalID {
		return out, state.ErrResourceLocked
	}
	return out, nil
}

func collectRecoverPlan(ctx context.Context, a doctorArgs, id string) diagnostic.RecoverPlan {
	_, report := collectDoctor(ctx, a)
	p := diagnostic.NewRecoverPlan(report, id, os.Geteuid(), time.Now())
	if p.OriginalID != "" && p.Status != "HISTORICAL" {
		// Recompute issue in same private capture as the exact original.
		snap, e := state.OpenDiagnosticSnapshot(ctx, a.DB)
		if e != nil {
			p.Block("FRESH_LEDGER_UNAVAILABLE")
		} else {
			o, e := snap.ReadOriginal(ctx, a.Project, p.OriginalID)
			l, le := snap.ReadLedger(ctx, a.Project, report.Installation.Mode, "incusbr-"+strings.TrimPrefix(a.Project, "user-"))
			current := diagnostic.NewReport(a.Project)
			if le == nil {
				current.ClassifyLedger(l)
			}
			fresh, ok := diagnostic.FindIssue(current, id)
			if e != nil || le != nil || !ok || p.Issue == nil || fresh != *p.Issue {
				p.Block("ISSUE_OR_ORIGINAL_CHANGED_OR_UNPROVED")
			} else if p.SetOriginal(o) {
				p.LedgerSHA256 = l.ObservationDigest()
				p.EvidenceEpochNS = snap.EpochNS
				q := planWorkerRequest{Project: a.Project, OriginalID: o.Target.ID, OriginalSHA256: o.Digest, Mode: p.Mode}
				// Only exact known 'all' and 'disabled' modes are reconstructible
				// from the redacted observation. A restricted policy requires MCP's
				// actual running policy and is conservatively blocked here.
				policyReady := true
				if o.Target.Action == "port_forward_create" {
					p.EffectiveForwardPolicy = report.ForwardPolicy.Reason
					if report.ForwardPolicy.State != "OBSERVED" || report.ForwardPolicy.Reason != "all" && report.ForwardPolicy.Reason != "disabled" {
						policyReady = false
					} else if report.ForwardPolicy.Reason == "all" {
						q.ForwardPolicy = "all"
					}
				}
				if !policyReady {
					p.Block("EFFECTIVE_FORWARD_POLICY_NOT_RECONSTRUCTIBLE")
				} else {
					witness, we := restrictedPlanWitness(ctx, a, q)
					if we != nil {
						p.Block("CURRENT_NATIVE_OR_ROOT_PROOF_UNAVAILABLE")
					} else {
						again, e := state.OpenDiagnosticSnapshot(ctx, a.DB)
						same := false
						if e == nil {
							newest, ne := again.ReadOriginal(ctx, a.Project, p.OriginalID)
							ledger, le := again.ReadLedger(ctx, a.Project, p.Mode, "incusbr-"+strings.TrimPrefix(a.Project, "user-"))
							updated := diagnostic.NewReport(a.Project)
							if le == nil {
								updated.ClassifyLedger(ledger)
							}
							issue, ok := diagnostic.FindIssue(updated, id)
							same = ne == nil && le == nil && ok && issue == *p.Issue && newest.Digest == o.Digest && ledger.ObservationDigest() == l.ObservationDigest()
							if again.Close() != nil {
								same = false
							}
						}
						identity := install.ObserveDiagnosticInstallation(ctx, doctorInstallPlan(a), a.DB)
						if o.Target.Action == "port_forward_create" {
							policy, ok := install.ObserveDiagnosticForwardPolicy(ctx, doctorInstallPlan(a), identity)
							if !ok || policy != report.ForwardPolicy.Reason {
								same = false
							}
						}
						if !same || identity.IdentitySHA256 != p.InstallationSHA256 || identity.Mode != p.Mode || ctx.Err() != nil {
							p.Block("PLAN_EVIDENCE_CHANGED_OR_RECHECK_UNAVAILABLE")
						} else {
							p.SetWitness(witness)
						}
					}
				}
			}
			if snap.Close() != nil {
				p.Block("PRIVATE_CAPTURE_CLEANUP_UNVERIFIED")
			}
		}
	}
	return p
}
func runRecoverPlan(a doctorArgs, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	p := collectRecoverPlan(ctx, a, id)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if e := enc.Encode(p); e != nil {
		return e
	}
	if p.Status == "REVIEW_READY" || p.Status == "NOT_REQUIRED" || p.Status == "HISTORICAL" {
		return nil
	}
	return errors.New("recovery plan not review-ready")
}
