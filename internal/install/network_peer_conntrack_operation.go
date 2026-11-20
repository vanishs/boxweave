package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"syscall"

	"github.com/vanishs/boxweave/internal/state"
)

// A different immutable original operation has a different journal file.
// The old host-wide journal, including a completed one, still blocks this
// newer path until a separately reviewed upgrade/migration resolves it.
const nativePeerConntrackOperationsDir = "/var/lib/boxweave-native-peer-conntrack-v1"

func nativePeerConntrackOperationJournalPath(original string) (string, error) {
	if len(original) != 32 {
		return "", errors.New("root conntrack requires exact 32-character original operation ID")
	}
	for _, c := range original {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return "", errors.New("root conntrack original operation ID is not canonical lowercase hex")
		}
	}
	return filepath.Join(nativePeerConntrackOperationsDir, original+".json"), nil
}

func nativePeerConntrackPrepareOriginalJournalDirectory() error {
	if os.Geteuid() != 0 {
		return errors.New("only local host root may prepare native original journals")
	}
	if _, err := os.Lstat(nativePeerConntrackJournalPath); err == nil {
		return errors.New("legacy root conntrack journal exists; explicit migration/recovery required")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("legacy original journal identity uncertain: %w", err)
	}
	if err := os.Mkdir(nativePeerConntrackOperationsDir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	st, err := os.Lstat(nativePeerConntrackOperationsDir)
	if err != nil || !st.IsDir() || st.Mode().Perm() != 0700 {
		return errors.New("unsafe root original journal directory type or permissions")
	}
	stat, ok := st.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 {
		return errors.New("root original journal directory ownership uncertain")
	}
	// A fresh root-only directory must itself be durable before any intent
	// created within it can be treated as a no-replay crash fence.
	parent, err := os.Open(filepath.Dir(nativePeerConntrackOperationsDir))
	if err != nil {
		return err
	}
	defer parent.Close()
	return parent.Sync()
}

// NativePeerConntrackReviewOperation may be used by the installed root
// helper to return only the digest for an already-claimed durable revoke.
// This never authorizes or executes a native mutation.
func NativePeerConntrackReviewOperation(ctx context.Context, s *state.Store,
	project, bridge, acl, original string) (NativePeerConntrackPlan, error) {
	var zero NativePeerConntrackPlan
	if os.Geteuid() != 0 || s == nil || !s.IsNetworkAuditReadOnly() {
		return zero, errors.New("read-only protected SQLite and host root required")
	}
	proof, err := s.ReadRootPeerRevocation(ctx, original, project, bridge, acl)
	if err != nil {
		return zero, err
	}
	plan, err := NativePeerConntrackReview(ctx, s, project, bridge, acl,
		proof.Intent.SourceUUID, proof.Intent.TargetUUID)
	if err != nil {
		return zero, err
	}
	again, err := s.ReadRootPeerRevocation(ctx, original, project, bridge, acl)
	if err != nil || !reflect.DeepEqual(proof, again) {
		return zero, errors.New("protected original peer revoke drifted during root review")
	}
	return plan, nil
}

// NativePeerConntrackApplyOperation is available only to an explicitly
// host-admin-started, SO_PEERCRED-checked root daemon. Its caller supplies
// ONLY an already-claimed SQLite operation ID and its reviewed digest.
// The Box pair, bridge, ACL and IPs are never taken from IPC.
func NativePeerConntrackApplyOperation(ctx context.Context, s *state.Store,
	project, bridge, acl, original, planSHA string, approved bool) (NativePeerConntrackJournalReceipt, error) {
	var zero NativePeerConntrackJournalReceipt
	if os.Geteuid() != 0 || s == nil || !s.IsNetworkAuditReadOnly() || !approved {
		return zero, errors.New("root-only installed scope and explicit host approval required")
	}
	path, err := nativePeerConntrackOperationJournalPath(original)
	if err != nil {
		return zero, err
	}
	release, err := nativeScriptletAcquireLock(nativePeerConntrackLockPath)
	if err != nil {
		return zero, err
	}
	defer release()
	proof, err := s.ReadRootPeerRevocation(ctx, original, project, bridge, acl)
	if err != nil {
		return zero, err
	}
	plan, err := NativePeerConntrackReview(ctx, s, project, bridge, acl,
		proof.Intent.SourceUUID, proof.Intent.TargetUUID)
	if err != nil {
		return zero, err
	}
	if plan.PlanSHA256 == "" || planSHA != plan.PlanSHA256 {
		return zero, errors.New("root peer plan digest not approved or stale")
	}
	again, err := s.ReadRootPeerRevocation(ctx, original, project, bridge, acl)
	if err != nil || !reflect.DeepEqual(proof, again) {
		return zero, errors.New("original root operation drifted before journal dispatch")
	}
	if err := nativePeerConntrackPrepareOriginalJournalDirectory(); err != nil {
		return zero, err
	}
	return nativePeerConntrackDispatchOnce(ctx, path, plan, planSHA, approved,
		func(ctx context.Context) (NativePeerConntrackReport, error) {
			report, err := NativePeerConntrackPurgeFromStore(ctx, s, project, bridge, acl,
				proof.Intent.SourceUUID, proof.Intent.TargetUUID)
			if err != nil {
				return report, err
			}
			after, err := s.ReadRootPeerRevocation(ctx, original, project, bridge, acl)
			if err != nil || !reflect.DeepEqual(proof, after) {
				return report, errors.New("root revoke original changed after native effect; journal stays UNKNOWN")
			}
			return report, nil
		})
}
