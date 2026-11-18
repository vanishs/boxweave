package install

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/vanishs/boxweave/internal/state"
	"golang.org/x/sys/unix"
)

// This journal belongs only to the root-only exact-peer conntrack capability.
// Its existence is intentionally permanent until a separately reviewed
// recovery/archive contract exists. Neither MCP nor a service worker can
// delete, replace, or replay this intent.
const nativePeerConntrackJournalPath = "/var/lib/boxweave-native-peer-conntrack-v1.json"
const nativePeerConntrackLockPath = "/var/lib/boxweave-native-peer-conntrack-v1.lock"

type NativePeerConntrackJournalReceipt struct {
	Status                 string `json:"status"`
	PlanSHA256             string `json:"plan_sha256"`
	ProbedScopes           int    `json:"probed_scopes"`
	DeletedScopes          int    `json:"deleted_scopes"`
	ReadyForPeer           bool   `json:"ready_for_peer"`
	MutationComplete       bool   `json:"mutation_complete"`
	KernelVerified         bool   `json:"kernel_verified,omitempty"`
	VerifiedNICs           int    `json:"verified_nics,omitempty"`
	PostNativeDigestSHA256 string `json:"post_native_digest_sha256,omitempty"`
}

type nativePeerConntrackJournal struct {
	Schema int                       `json:"schema"`
	Status string                    `json:"status"`
	Plan   NativePeerConntrackPlan   `json:"plan"`
	Report NativePeerConntrackReport `json:"report"`
}

func nativePeerConntrackReceipt(j nativePeerConntrackJournal) NativePeerConntrackJournalReceipt {
	return NativePeerConntrackJournalReceipt{
		Status: j.Status, PlanSHA256: j.Plan.PlanSHA256, ProbedScopes: j.Report.ProbedScopes,
		DeletedScopes: j.Report.DeletedScopes, ReadyForPeer: false,
		MutationComplete:       j.Status == "COMPLETE_VERIFIED",
		KernelVerified:         j.Status == "COMPLETE_VERIFIED" && j.Report.KernelVerified,
		VerifiedNICs:           j.Report.VerifiedNICs,
		PostNativeDigestSHA256: j.Report.PostNativeDigestSHA256,
	}
}

// An UNKNOWN journal is saved to stable storage BEFORE the first native
// side effect. The only permissible transition is original UNKNOWN -> VERIFIED.
// No cleanup/replay/reconcile API exists here.
func nativePeerConntrackWriteIntent(path string, j nativePeerConntrackJournal) error {
	if j.Schema != 1 || j.Status != "DISPATCH_UNKNOWN" ||
		j.Plan.PlanSHA256 == "" || j.Plan.MutationAuthorized {
		return errors.New("invalid root conntrack journal pre-dispatch intent")
	}
	st, err := os.Lstat(filepath.Dir(path))
	if err != nil || !st.IsDir() || st.Mode().Perm()&0022 != 0 {
		return errors.New("root conntrack journal parent missing or writable by untrusted actors")
	}
	content, err := json.Marshal(j)
	if err != nil {
		return err
	}
	content = append(content, '\n')
	fd, err := unix.Open(path, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if err != nil {
		// An existing file always freezes this root scope, even if corrupt,
		// old or from a crash during initial creation.
		return fmt.Errorf("root conntrack journal already exists or cannot be created: %w", err)
	}
	f := os.NewFile(uintptr(fd), path)
	_, err = f.Write(content)
	if err == nil {
		err = f.Sync()
	}
	closed := f.Close()
	if err != nil {
		return err
	}
	if closed != nil {
		return closed
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func nativePeerConntrackLoadJournal(path string) (nativePeerConntrackJournal, error) {
	var j nativePeerConntrackJournal
	info, err := os.Lstat(path)
	if err != nil {
		return j, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return j, errors.New("unsafe root conntrack journal mode or file type")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return j, err
	}
	if err := json.Unmarshal(b, &j); err != nil {
		return j, err
	}
	if j.Schema != 1 || j.Plan.PlanSHA256 == "" || j.Plan.MutationAuthorized ||
		(j.Status != "DISPATCH_UNKNOWN" && j.Status != "COMPLETE_VERIFIED") {
		return j, errors.New("unrecognized root conntrack original intent")
	}
	return j, nil
}

func nativePeerConntrackSaveCompletion(path string, original nativePeerConntrackJournal, report NativePeerConntrackReport) error {
	prior, err := nativePeerConntrackLoadJournal(path)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(original, prior) || prior.Status != "DISPATCH_UNKNOWN" {
		return errors.New("root conntrack intent changed before complete transition")
	}
	completed := original
	completed.Status = "COMPLETE_VERIFIED"
	completed.Report = report
	completed.Report.ReadyForPeer = false
	b, err := json.Marshal(completed)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	// atomicWrite synchronizes the temp payload and renames it; the
	// containing directory is then synced before a completed ACK is returned.
	if err := atomicWrite(path, b, 0600); err != nil {
		return err
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

type nativePeerConntrackEffect func(context.Context) (NativePeerConntrackReport, error)

// Sealed, one-shot root effect only. Callers MUST first hold the exclusive
// local lock and rederive their review from real native root and protected DB.
// This function deliberately cannot retry an existing successful OR UNKNOWN
// original journal, including when a previous native ACK was lost.
func nativePeerConntrackDispatchOnce(ctx context.Context, path string, plan NativePeerConntrackPlan,
	approvedSHA string, approved bool, effect nativePeerConntrackEffect) (NativePeerConntrackJournalReceipt, error) {
	var zero NativePeerConntrackJournalReceipt
	if !approved || approvedSHA == "" || approvedSHA != plan.PlanSHA256 || plan.MutationAuthorized ||
		plan.HostSHA256 == "" || plan.NativeDigestSHA256 == "" || plan.LedgerSHA256 == "" || effect == nil {
		return zero, errors.New("explicit exact reviewed root conntrack intent is required")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	j := nativePeerConntrackJournal{Schema: 1, Status: "DISPATCH_UNKNOWN", Plan: plan}
	if err := nativePeerConntrackWriteIntent(path, j); err != nil {
		return zero, err
	}
	unknown := nativePeerConntrackReceipt(j)
	// After the persisted intent is written, any failure is UNKNOWN, not a
	// reason to retry. This also covers a canceled context before dispatch.
	if err := ctx.Err(); err != nil {
		return unknown, fmt.Errorf("root conntrack original intent is UNKNOWN after cancellation: %w", err)
	}
	report, err := effect(ctx)
	if err != nil {
		return unknown, fmt.Errorf("root conntrack original native effect uncertain; no replay: %w", err)
	}
	if report.ReadyForPeer || report.ProbedScopes != 4 ||
		report.DeletedScopes < 0 || report.DeletedScopes > report.ProbedScopes ||
		report.NativeDigestSHA256 != plan.NativeDigestSHA256 {
		return unknown, errors.New("root conntrack result is incomplete or identity drifted; original UNKNOWN")
	}
	if err := nativePeerConntrackSaveCompletion(path, j, report); err != nil {
		return unknown, fmt.Errorf("root conntrack completion could not be durably confirmed; UNKNOWN: %w", err)
	}
	j.Status = "COMPLETE_VERIFIED"
	j.Report = report
	return nativePeerConntrackReceipt(j), nil
}

// NativePeerConntrackApplyReviewed is private host-root orchestration, not an
// MCP handler or an unbounded host shell. No public CLI is wired until a
// complete installed permission/recovery acceptance is independently signed.
func NativePeerConntrackApplyReviewed(ctx context.Context, s *state.Store,
	project, bridge, acl, sourceUUID, targetUUID, planSHA string, approved bool) (NativePeerConntrackJournalReceipt, error) {
	var zero NativePeerConntrackJournalReceipt
	if os.Geteuid() != 0 || s == nil || !s.IsNetworkAuditReadOnly() || !approved {
		return zero, errors.New("host root, validated read-only SQLite and explicit approval required")
	}
	release, err := nativeScriptletAcquireLock(nativePeerConntrackLockPath)
	if err != nil {
		return zero, err
	}
	defer release()
	plan, err := NativePeerConntrackReview(ctx, s, project, bridge, acl, sourceUUID, targetUUID)
	if err != nil {
		return zero, err
	}
	return nativePeerConntrackDispatchOnce(ctx, nativePeerConntrackJournalPath, plan, planSHA, approved,
		func(ctx context.Context) (NativePeerConntrackReport, error) {
			return NativePeerConntrackPurgeFromStore(ctx, s, project, bridge, acl, sourceUUID, targetUUID)
		})
}
