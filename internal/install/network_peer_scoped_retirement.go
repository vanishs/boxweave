package install

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"syscall"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/peerscope"
	"github.com/vanishs/boxweave/internal/state"
)

const scopedRetirementFence = "/var/lib/boxweave-scoped-retirement-v1.json"

// Startup is fenced throughout retirement, including reboot/interruption.
// The boot guard is intentionally NOT fenced: until its dependency is
// detached, it must keep restoring the original protected policy.
func refuseScopedRetirementInProgress() error {
	if _, err := os.Lstat(scopedRetirementFence); !errors.Is(err, os.ErrNotExist) {
		return errors.New("SCOPED_RETIREMENT_IN_PROGRESS: preserve original audit and review resume")
	}
	return nil
}

type ScopedRetirementReview struct {
	Project      string `json:"project"`
	ReviewSHA256 string `json:"review_sha256"`
	Phase        string `json:"phase"`
	GuardPresent bool   `json:"guard_present"`
	Retired      int    `json:"retired"`
	Table        string `json:"table"`
}

type scopedRetirementNative struct {
	Review                                  ScopedRetirementReview
	Preimage                                scopedRetirementPreimage
	Guard                                   NativePeerGuardPlan
	ProjectTag, ProfileTag                  string
	ProjectBefore, ProfileBefore, ACLAbsent bool
}

func scopedRetirementUnitProperty(ctx context.Context, unit, property string) (string, error) {
	out, err := exec.CommandContext(ctx, "/usr/bin/systemctl", "show", "-p", property, "--value", unit).Output()
	return strings.TrimSpace(string(out)), err
}

func scopedRetirementInactive(ctx context.Context, units ...string) error {
	for _, unit := range units {
		a, err := scopedRetirementUnitProperty(ctx, unit, "ActiveState")
		pid, pe := scopedRetirementUnitProperty(ctx, unit, "MainPID")
		if err != nil || pe != nil || a != "inactive" || pid != "0" {
			return errors.New("retirement requires proven inactive " + unit)
		}
	}
	return nil
}

func scopedRetirementDetached(ctx context.Context) error {
	if _, err := os.Lstat(scopedIncusDropIn); !errors.Is(err, os.ErrNotExist) {
		return errors.New("original Incus guard drop-in still present")
	}
	for _, unit := range []string{"incus.service", "incus-startup.service"} {
		for _, prop := range []string{"Requires", "BindsTo", "Wants", "Requisite"} {
			v, err := scopedRetirementUnitProperty(ctx, unit, prop)
			if err != nil || slices.Contains(strings.Fields(v), scopedGuardUnitName) {
				return errors.New("effective Incus dependency still present or unverified")
			}
		}
	}
	return scopedRetirementInactive(ctx, scopedGuardUnitName)
}

func scopedRetirementFenceMatches(project, digest string) error {
	var receipt struct {
		Schema  int    `json:"schema"`
		Project string `json:"project"`
		Native  string `json:"native_review_sha256"`
		Review  string `json:"review_sha256"`
		Archive string `json:"archive"`
	}
	st, err := os.Lstat(scopedRetirementFence)
	if err != nil {
		return err
	}
	own, ok := st.Sys().(*syscall.Stat_t)
	if !ok || own.Uid != 0 || own.Nlink != 1 || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || st.Size() > 8192 {
		return errors.New("untrusted scoped retirement fence")
	}
	b, err := os.ReadFile(scopedRetirementFence)
	if err != nil || json.Unmarshal(b, &receipt) != nil || receipt.Schema != 1 ||
		receipt.Project != project || receipt.Native != digest || !nativePeerGuardSHA256Valid(receipt.Review) ||
		filepath.Dir(receipt.Archive) != "/var/lib/boxweave-retired" ||
		!regexp.MustCompile(`^retired-[a-z0-9_]{8}$`).MatchString(filepath.Base(receipt.Archive)) {
		return errors.New("exact archived retirement fence required")
	}
	for _, dir := range []string{filepath.Dir(receipt.Archive), receipt.Archive} {
		info, err := os.Lstat(dir)
		if err != nil {
			return err
		}
		owner, ok := info.Sys().(*syscall.Stat_t)
		if !ok || owner.Uid != 0 || !info.IsDir() || info.Mode().Perm() != 0700 {
			return errors.New("untrusted scoped retirement archive")
		}
	}
	var reviewed struct {
		Mode    string                 `json:"mode"`
		Project string                 `json:"project"`
		Review  string                 `json:"review_sha256"`
		Ready   bool                   `json:"ready"`
		Native  ScopedRetirementReview `json:"scoped_native"`
	}
	path := filepath.Join(receipt.Archive, "review.json")
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || owner.Nlink != 1 || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 1024*1024 {
		return errors.New("untrusted archived scoped retirement review")
	}
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &reviewed) != nil || !reviewed.Ready ||
		reviewed.Mode != "SCOPED_PEER_FRESH_V2" || reviewed.Project != project ||
		reviewed.Review != receipt.Review || reviewed.Native.ReviewSHA256 != digest || reviewed.Native.Project != project {
		return errors.New("archived scoped retirement review mismatch")
	}
	return nil
}

// Root receipts must match every retired lineage, and no extra incomplete
// lifecycle original may be silently hidden by a terminal SQLite row.
func scopedRetirementReceipts(r nativePeerFreshJournalRecord, ledger state.ScopedNetworkAuditLedger) ([]scopedLifecycleRecord, error) {
	want := map[string]bool{}
	var receipts []scopedLifecycleRecord
	for _, b := range ledger.Retired {
		a, err := nativeScopedCreateReceipt(b.CreateID)
		if err != nil || !nativeScopedReceiptMatches(r, a, b.CreateID, "CREATE", b.ScopedNetworkAuditBinding) {
			return nil, errors.New("retired Root Create evidence unverified")
		}
		d, err := nativeScopedDeleteReceipt(b.DeleteID)
		if err != nil || !nativeScopedReceiptMatches(r, d, b.DeleteID, "DELETE", b.ScopedNetworkAuditBinding) {
			return nil, errors.New("retired Root Delete evidence unverified")
		}
		want["op-"+b.CreateID], want["op-"+b.DeleteID] = true, true
		receipts = append(receipts, a, d)
	}
	files, err := os.ReadDir(nativePeerFreshJournalDir)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "op-") {
			if !want[f.Name()] || !f.IsDir() {
				return nil, errors.New("unsettled or foreign Root lifecycle original")
			}
			delete(want, f.Name())
		}
	}
	if len(want) != 0 {
		return nil, errors.New("retired Root lifecycle original missing")
	}
	return receipts, nil
}

func scopedRetirementNativeRead(ctx context.Context, project string, resume bool) (scopedRetirementNative, error) {
	var zero scopedRetirementNative
	if os.Geteuid() != 0 {
		return zero, errors.New("scoped retirement requires Host-local root")
	}
	r, initial, err := nativePeerFreshVerifiedPlan()
	if err != nil || r.Project != project {
		return zero, errors.New("scoped retirement original Host/Project not verified")
	}
	if err = nativePeerFreshQuarantineReceipt(r); err != nil {
		return zero, err
	}
	p, err := scopedRetirementPreimageRead(r)
	if err != nil {
		return zero, err
	}
	guard, last, err := scopedPeerBootCurrent(nativePeerFreshJournalDir, 0, r, initial, "")
	if err != nil || len(last.Active) != 0 {
		return zero, errors.New("scoped retirement requires terminal empty Peer policy")
	}
	db, err := state.OpenNetworkAuditReadOnly(nativePeerRootInstalledDB)
	if err != nil {
		return zero, err
	}
	defer db.Close()
	ledger, err := db.ReadScopedRetirementLedger(ctx, r.Project, r.Bridge)
	if err != nil {
		return zero, errors.New("SCOPED_RETIREMENT_LIVE_OR_UNSETTLED: clean up through MCP first; preserve original claims")
	}
	receipts, err := scopedRetirementReceipts(r, ledger)
	if err != nil {
		return zero, err
	}
	c, err := prepareClient(ctx)
	if err != nil {
		return zero, err
	}
	c = c.UseProject("default")
	own := c.UseProject(project)
	instances, err := own.GetInstanceNames(api.InstanceTypeAny)
	if err != nil || len(instances) != 0 {
		return zero, errors.New("scoped native Project not proven empty")
	}
	ops, err := own.GetOperations()
	if err != nil {
		return zero, err
	}
	for _, op := range ops {
		if !op.StatusCode.IsFinal() {
			return zero, errors.New("scoped native operation still active")
		}
	}
	nw, _, err := c.GetNetwork(r.Bridge)
	if err != nil || nw == nil || !nw.Managed || nw.Type != "bridge" || nw.Status != "Created" ||
		!reflect.DeepEqual(nw.Writable(), p.Bridge) || nativePeerBridgeConsumersOwned(nw.UsedBy, project, nil) != nil {
		return zero, errors.New("scoped retirement bridge or consumer drift")
	}
	if err = nativePeerBridgeKernelNoLivePorts(ctx, r.Bridge); err != nil {
		return zero, err
	}
	pr, pt, err := own.GetProject(project)
	if err != nil || pr == nil || pt == "" {
		return zero, errors.New("scoped retirement Project unavailable")
	}
	pf, ft, err := own.GetProfile("default")
	if err != nil || pf == nil || ft == "" || len(pf.UsedBy) != 0 {
		return zero, errors.New("scoped retirement Profile has consumers or is unverified")
	}
	profileBefore := reflect.DeepEqual(pf.Writable(), p.BeforeProfile)
	projectBefore := reflect.DeepEqual(pr.Writable(), p.BeforeProject)
	if (!profileBefore && !reflect.DeepEqual(pf.Writable(), p.AfterProfile)) ||
		(!projectBefore && !reflect.DeepEqual(pr.Writable(), p.AfterProject)) {
		return zero, errors.New("SCOPED_RETIREMENT_CONFIG_DRIFT")
	}
	acl, tag, err := c.GetNetworkACL(peerscope.QuarantineACL)
	absent := api.StatusErrorCheck(err, http.StatusNotFound)
	if !absent {
		if err != nil || acl == nil || tag == "" || acl.Project != "default" ||
			!reflect.DeepEqual(acl.Writable(), p.Quarantine.NetworkACLPut) {
			return zero, errors.New("scoped quarantine ACL changed")
		}
		if profileBefore {
			if len(acl.UsedBy) != 0 {
				return zero, errors.New("detached quarantine ACL has a consumer")
			}
		} else if len(acl.UsedBy) != 1 || !nativePeerGuardOwnedURL(acl.UsedBy[0], "profiles", "default", project) {
			return zero, errors.New("quarantine ACL consumer drift")
		}
	}
	names, err := c.GetNetworkACLNames()
	if err != nil {
		return zero, err
	}
	for _, name := range names {
		if strings.HasPrefix(name, "bw-") && (name != peerscope.QuarantineACL || absent) {
			return zero, errors.New("unexpected or retained BoxWeave ACL")
		}
	}
	if absent && (!profileBefore || !projectBefore) {
		return zero, errors.New("quarantine disappeared before reviewed native restore")
	}
	if !profileBefore && projectBefore && !reflect.DeepEqual(p.BeforeProject, p.AfterProject) {
		return zero, errors.New("native state not an approved retirement prefix")
	}
	present, err := nativePeerFreshTableLive(ctx, guard)
	if err != nil {
		return zero, err
	}
	if present {
		if err = NativePeerGuardKernelReadback(ctx, guard); err != nil {
			return zero, err
		}
	} else if !absent || !profileBefore || !projectBefore || !resume {
		return zero, errors.New("scoped guard missing before retirement")
	}
	if !resume {
		if profileBefore && !reflect.DeepEqual(p.BeforeProfile, p.AfterProfile) ||
			projectBefore && !reflect.DeepEqual(p.BeforeProject, p.AfterProject) || absent {
			return zero, errors.New("native state not original installed scoped state")
		}
		if _, _, _, err = nativePeerFreshQuarantineRead(ctx, c, r, ""); err != nil {
			return zero, err
		}
	}
	// This digest binds original preimage, retained identities/receipts,
	// immutable boot history and terminal guard, not mutable service PIDs.
	digest := nativePrepareStateDigest("scoped-retirement-v1", r, p, ledger, receipts, last, guard.DigestSHA256)
	phase := "INSTALLED"
	if profileBefore {
		phase = "PROFILE_RESTORED"
	}
	if profileBefore && projectBefore {
		phase = "PROJECT_RESTORED"
	}
	if absent {
		phase = "NATIVE_RESTORED"
	}
	if !present {
		phase = "GUARD_ABSENT"
	}
	return scopedRetirementNative{
		Review:   ScopedRetirementReview{Project: project, ReviewSHA256: digest, Phase: phase, GuardPresent: present, Retired: len(ledger.Retired), Table: guard.Table},
		Preimage: p, Guard: guard, ProjectTag: pt, ProfileTag: ft,
		ProjectBefore: projectBefore, ProfileBefore: profileBefore, ACLAbsent: absent,
	}, nil
}

func scopedRetirementStable(ctx context.Context, project string, resume bool) (scopedRetirementNative, error) {
	a, err := scopedRetirementNativeRead(ctx, project, resume)
	if err != nil {
		return a, err
	}
	b, err := scopedRetirementNativeRead(ctx, project, resume)
	if err != nil || !reflect.DeepEqual(a, b) || ctx.Err() != nil {
		return scopedRetirementNative{}, errors.New("scoped retirement observation changed or unavailable")
	}
	return a, nil
}

func NativeScopedRetirementPlan(ctx context.Context, project string, resume bool) (ScopedRetirementReview, error) {
	p, err := scopedRetirementStable(ctx, project, resume)
	return p.Review, err
}

// The operator-approved original must be archived and the restricted
// controller and Root lifecycle stopped. Re-read each exact prefix before
// a single effect. On an error, return immediately; only a fresh reviewed
// resume may continue after independent post-state verification.
func NativeScopedRetirementApply(ctx context.Context, project, digest string, guardOnly, approved bool) (ScopedRetirementReview, error) {
	var zero ScopedRetirementReview
	if !approved || !nativePeerGuardSHA256Valid(digest) {
		return zero, errors.New("exact local scoped retirement approval required")
	}
	if err := scopedRetirementFenceMatches(project, digest); err != nil {
		return zero, err
	}
	read := func() (scopedRetirementNative, error) {
		if err := scopedRetirementInactive(ctx, "boxweave.service", scopedLifecycleUnitName); err != nil {
			return scopedRetirementNative{}, err
		}
		p, err := scopedRetirementStable(ctx, project, true)
		if err != nil {
			return p, err
		}
		if p.Review.ReviewSHA256 != digest {
			return p, errors.New("scoped retirement original evidence changed")
		}
		return p, nil
	}
	for i := 0; i < 4; i++ {
		p, err := read()
		if err != nil {
			return zero, err
		}
		if guardOnly {
			if !p.ACLAbsent || !p.ProfileBefore || !p.ProjectBefore {
				return zero, errors.New("native restore not independently verified")
			}
			if err = scopedRetirementDetached(ctx); err != nil {
				return zero, err
			}
			if !p.Review.GuardPresent {
				return p.Review, nil
			}
			if !nativePeerGuardTablePattern.MatchString(p.Guard.Table) {
				return zero, errors.New("invalid exact guard table")
			}
			if err = nativePeerGuardFixedNftBatch(ctx, "delete table bridge "+p.Guard.Table+"\n"); err != nil {
				return zero, err
			}
			after, err := read()
			if err != nil || after.Review.GuardPresent {
				return zero, errors.New("scoped guard retirement outcome unverified")
			}
			return after.Review, nil
		}
		if p.ACLAbsent {
			return p.Review, nil
		}
		if !p.Review.GuardPresent {
			return zero, errors.New("guard must remain until native restore")
		}
		c, err := prepareClient(ctx)
		if err != nil {
			return zero, err
		}
		c = c.UseProject("default")
		own := c.UseProject(project)
		switch {
		case !p.ProfileBefore:
			err = own.UpdateProfile("default", p.Preimage.BeforeProfile, p.ProfileTag)
		case !p.ProjectBefore:
			err = own.UpdateProject(project, p.Preimage.BeforeProject, p.ProjectTag)
		default:
			err = c.DeleteNetworkACL(peerscope.QuarantineACL)
		}
		if err != nil {
			return zero, fmt.Errorf("scoped native retirement effect outcome unverified; review original: %w", err)
		}
		after, err := read()
		if err != nil || after.Review.Phase == p.Review.Phase {
			return zero, errors.New("native retirement effect not independently verified")
		}
	}
	return zero, errors.New("native retirement phase bound exceeded")
}
