package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/vanishs/boxweave/internal/control"
	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type scopedDrainRecord struct {
	Schema         int                       `json:"schema"`
	Project        string                    `json:"project"`
	Binary         string                    `json:"binary_sha256"`
	Review         string                    `json:"review_sha256"`
	Phase          string                    `json:"phase"`
	Index          int                       `json:"index"`
	Steps          []control.ScopedDrainStep `json:"steps"`
	Actor          state.ScopedDrainActor    `json:"actor"`
	ExpectedLedger string                    `json:"expected_ledger_sha256"`
	ExpectedNative string                    `json:"expected_native_sha256"`
}
type scopedDrainRequest struct {
	Project        string
	Actor          state.ScopedDrainActor
	Step           control.ScopedDrainStep
	ExpectedLedger string
}
type scopedDrainResult struct {
	Found   bool           `json:"found"`
	Receipt *state.Receipt `json:"receipt,omitempty"`
}

func drainHex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	b, e := hex.DecodeString(s)
	return e == nil && hex.EncodeToString(b) == s
}

var drainUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func validDrainStep(s control.ScopedDrainStep) bool {
	if !state.ValidCorrelationRef(s.Key) || len(s.Key) < 16 || len(s.Key) > 128 || !drainUUID.MatchString(s.Box) {
		return false
	}
	switch s.Action {
	case "box_peer_set":
		return drainUUID.MatchString(s.Target) && s.Target != s.Box && s.Forward == ""
	case "port_forward_delete":
		return drainHex(s.Forward, 32) && s.Target == ""
	case "box_stop", "box_delete":
		return s.Target == "" && s.Forward == ""
	default:
		return false
	}
}
func scopedDrainRoot(probe bool) error {
	if os.Geteuid() != 0 {
		return errors.New("scoped drain requires exact local root approval")
	}
	if !probe {
		release, e := acquireMaintenanceLock()
		if e != nil {
			return e
		}
		defer release()
	}
	raw, e := install.ReadScopedDrainFence()
	if e != nil {
		return e
	}
	var r scopedDrainRecord
	if json.Unmarshal(raw, &r) != nil || r.Schema != 1 || r.Phase != "DRAINING" || !state.SafeDiagnosticProject(r.Project) || !drainHex(r.Binary, 64) || r.Binary != maintenanceSelfSHA() || !drainHex(r.Review, 64) || !drainHex(r.Actor.ID, 32) || r.Index < 0 || r.Index >= len(r.Steps) || len(r.Steps) > 16384 || !validDrainStep(r.Steps[r.Index]) || !drainHex(r.ExpectedLedger, 64) || !drainHex(r.ExpectedNative, 64) {
		return state.ErrInvalid
	}
	if e = install.CheckScopedRetirementFence(); e != nil {
		return e
	}
	a := fixedRecoveryArgs(r.Project)
	uid, gid, e := install.ServiceIdentity(doctorInstallPlan(a))
	if e != nil || uid == 0 || r.Project != "user-"+strconv.Itoa(uid) {
		return state.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(context.Background(), 17*time.Minute)
	defer cancel()
	q := scopedDrainRequest{r.Project, r.Actor, r.Steps[r.Index], r.ExpectedLedger}
	// No new effect when the original exists, including lost local ACK.
	db, e := state.OpenNetworkAuditReadOnly(a.DB)
	if e != nil {
		return e
	}
	prior, e := db.ScopedDrainOriginal(ctx, r.Project, r.Actor.ID, q.Step.Key)
	ce := db.Close()
	if ce != nil {
		return ce
	}
	if e == nil {
		if prior.Action != q.Step.Action || prior.BoxUUID != q.Step.Box || prior.Incarnation != q.Step.Box || prior.WorkRef != "scoped-drain-"+r.Actor.ID {
			return state.ErrConflict
		}
		if ee := json.NewEncoder(os.Stdout).Encode(scopedDrainResult{true, &prior}); ee != nil {
			return ee
		}
		if prior.State != "SUCCEEDED" {
			return errors.New("SCOPED_DRAIN_ORIGINAL_UNSETTLED: preserve original")
		}
		return nil
	}
	if !errors.Is(e, state.ErrNotFound) {
		return e
	}
	if probe {
		return json.NewEncoder(os.Stdout).Encode(scopedDrainResult{})
	}
	// Independent root observation binds the exact approved current step,
	// native inventory, stopped Controller and still-active policy helpers.
	if e = install.CheckScopedDrainControllerInactive(ctx); e != nil {
		return e
	}
	report := collectMaintenance(ctx, a)
	i, n, l := report.Installation, report.Native, report.Ledger
	if report.Coverage != "BOUNDED_SOURCES_OBSERVED" || i.Adapter != "EXACT_RUNNING_OBSERVER_BINARY" || i.Mode != "SCOPED_V2" || i.InstalledSHA256 != r.Binary || i.Identity.StaticContract != "VERIFIED" || i.Identity.ServiceUnit != "INACTIVE" || i.Identity.GuardUnit != "ACTIVE" || i.Identity.LifecycleUnit != "ACTIVE" || i.RootPolicy.Kernel != "OBSERVED" || report.RootLedgerComparison != "MATCH" || l == nil || len(l.Originals) != 0 || l.ForeignRows != 0 || l.OrphanBindings != 0 || l.OrphanFences != 0 || n.OwnershipComparison != "OBSERVED" || n.Unowned != 0 || len(n.OwnedUnverified) != 0 || n.ActiveOperations != 0 || n.Evidence.SHA256 != r.ExpectedNative || report.LedgerSHA256 != r.ExpectedLedger || n.BoxACLReferences != "OBSERVED" || n.ForwardReferences != "OBSERVED" || n.BridgeConfig != "RECORDED_PREIMAGE_MATCH" || n.BaseACLConfig != "RECORDED_PREIMAGE_MATCH" || n.BridgeConsumers != "OWNED_REFERENCES_ONLY" || (n.BaseACLConsumers != "OWNED_REFERENCES_ONLY" && n.BaseACLConsumers != "NO_REFERENCES") {
		return errors.New("SCOPED_DRAIN_CURRENT_PROOF_UNVERIFIED")
	}
	for _, u := range i.Units {
		if u.DropIns != "ABSENT" || (u.EffectiveFile.State != "OBSERVED" && u.EffectiveFile.State != "ABSENT") {
			return errors.New("SCOPED_DRAIN_EFFECTIVE_UNIT_UNVERIFIED")
		}
	}
	return spawnScopedDrain(ctx, q, uid, gid)
}

func spawnScopedDrain(ctx context.Context, q scopedDrainRequest, uid, gid int) error {
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
	parent := os.NewFile(uintptr(pair[0]), "drain-root")
	child := os.NewFile(uintptr(pair[1]), "drain-child")
	defer parent.Close()
	defer child.Close()
	cmd := exec.CommandContext(ctx, "/proc/self/exe", "scoped-drain-worker")
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
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return e
	}
	_ = syscall.Shutdown(pair[0], syscall.SHUT_WR)
	return cmd.Wait()
}

func scopedDrainWorker() error {
	if os.Geteuid() == 0 {
		return state.ErrInvalid
	}
	peer, e := syscall.GetsockoptUcred(3, syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if e != nil || peer.Uid != 0 || peer.Pid <= 0 {
		return errors.New("root-approved scoped drain socket required")
	}
	f := os.NewFile(3, "scoped-drain-request")
	if f == nil {
		return state.ErrInvalid
	}
	defer f.Close()
	var q scopedDrainRequest
	if e = decodeReviewedInput(f, &q); e != nil || !validDrainStep(q.Step) || !drainHex(q.ExpectedLedger, 64) {
		return state.ErrInvalid
	}
	a := fixedRecoveryArgs("user-" + strconv.Itoa(os.Geteuid()))
	uid, _, e := install.ServiceIdentity(doctorInstallPlan(a))
	if e != nil || uid != os.Geteuid() || q.Project != a.Project {
		return state.ErrInvalid
	}
	if install.CheckScopedDrainFence() == nil {
		return errors.New("scoped drain fence absent")
	}
	if e = install.CheckScopedRetirementFence(); e != nil {
		return e
	}
	// Reuse the controller's exact exclusive lock; never create a database
	// or invoke startup recovery on an uncertain original.
	if _, e = os.Lstat(a.DB + ".serve.lock"); e != nil {
		return e
	}
	lock, e := acquireServeLock(a.DB)
	if e != nil {
		return e
	}
	defer lock.Close()
	db, e := state.OpenExistingRecovery(a.DB)
	if e != nil {
		return e
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 16*time.Minute)
	defer cancel()
	n := native.Reader{Project: a.Project, Socket: a.Socket}
	svc := control.Service{Store: db, Native: n, ScopedPeerFresh: true}
	receipt, e := svc.RunScopedDrainStep(ctx, q.Actor, q.Step, q.ExpectedLedger, false)
	if receipt.ID != "" {
		if ee := json.NewEncoder(os.Stdout).Encode(scopedDrainResult{true, &receipt}); ee != nil {
			return ee
		}
	}
	return e
}
