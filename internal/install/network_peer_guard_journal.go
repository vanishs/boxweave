package install

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// This path is a proposed, separate Root-only stage journal. It is not
// installed or exposed by the existing one-original Root Helper.
const nativePeerGuardRootJournalDir = "/var/lib/boxweave-native-peer-guard-v1"

// One exact authenticated original and pair's independently verified
// ledger, native and kernel pre/post digests, NEVER raw MCP ACL/IP input.
// Creation and completed status are only one NFT GUARD STEP, not evidence
// that the entire Peer grant/revoke (Incus ACL, conntrack, forwarding,
// application sessions) is safe or complete.
type nativePeerGuardJournalScope struct {
	OriginalID   string `json:"original_id"`
	Project      string `json:"project"`
	Bridge       string `json:"bridge"`
	ACL          string `json:"acl"`
	SourceUUID   string `json:"source_uuid"`
	TargetUUID   string `json:"target_uuid"`
	LedgerSHA256 string `json:"ledger_sha256"`
	NativeSHA256 string `json:"native_sha256"`
	BeforeSHA256 string `json:"before_sha256"`
	AfterSHA256  string `json:"after_sha256"`
}

type nativePeerGuardJournal struct {
	Schema  int                           `json:"schema"`
	Status  string                        `json:"status"`
	Scope   nativePeerGuardJournalScope   `json:"scope"`
	Outcome nativePeerGuardReplaceOutcome `json:"outcome"`
}

type nativePeerGuardJournalReceipt struct {
	Status            string
	OriginalID        string
	GuardStepVerified bool
	NativeAttempted   bool
	BeforeSHA256      string
	AfterSHA256       string
}

func nativePeerGuardScopeValid(s nativePeerGuardJournalScope) error {
	if _, err := nativePeerConntrackOperationJournalPath(s.OriginalID); err != nil {
		return err
	}
	if !strings.HasPrefix(s.Project, "user-") ||
		!safePrereqName.MatchString(s.Project) ||
		!safePrereqName.MatchString(s.Bridge) ||
		!safePrereqName.MatchString(s.ACL) ||
		!strings.HasPrefix(s.ACL, "bw-") ||
		!scopedPeerUUID.MatchString(s.SourceUUID) ||
		!scopedPeerUUID.MatchString(s.TargetUUID) ||
		s.SourceUUID == s.TargetUUID {
		return errors.New("unsafe original Project/bridge/ACL or native peer identities")
	}
	for _, digest := range []string{s.LedgerSHA256, s.NativeSHA256, s.BeforeSHA256, s.AfterSHA256} {
		if !nativePeerGuardSHA256Valid(digest) {
			return errors.New("missing or untrusted protected original/kernel SHA256")
		}
	}
	if s.BeforeSHA256 == s.AfterSHA256 {
		return errors.New("guard original has no approved exact direction delta")
	}
	return nil
}
func nativePeerGuardSHA256Valid(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
func nativePeerGuardJournalPathAt(dir, original string) (string, error) {
	if _, err := nativePeerConntrackOperationJournalPath(original); err != nil {
		return "", err
	}
	return filepath.Join(dir, original+".json"), nil
}

// Caller supplies a FIXED root-owned directory in production. dir+uid are
// package-private test seams only, not CLI/socket/MCP inputs.
func nativePeerGuardJournalPrepareAt(dir string, uid uint32) error {
	if dir == "" || !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
		return errors.New("unsafe root guard journal directory")
	}
	if err := os.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	st, err := os.Lstat(dir)
	if err != nil || !st.IsDir() || st.Mode().Perm() != 0700 || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("root guard journal directory not exclusive and private")
	}
	stat, ok := st.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uid {
		return errors.New("root guard journal directory ownership mismatch")
	}
	p, err := os.Open(filepath.Dir(dir))
	if err != nil {
		return err
	}
	defer p.Close()
	return p.Sync()
}
func nativePeerGuardJournalIntentAt(dir string, uid uint32, j nativePeerGuardJournal) error {
	if j.Schema != 1 || j.Status != "DISPATCH_UNKNOWN" ||
		nativePeerGuardScopeValid(j.Scope) != nil || j.Outcome.Attempted || j.Outcome.KernelVerified {
		return errors.New("invalid exact root guard original intent")
	}
	if err := nativePeerGuardJournalPrepareAt(dir, uid); err != nil {
		return err
	}
	path, err := nativePeerGuardJournalPathAt(dir, j.Scope.OriginalID)
	if err != nil {
		return err
	}
	content, err := json.Marshal(j)
	if err != nil {
		return err
	}
	content = append(content, '\n')
	fd, err := unix.Open(path, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return fmt.Errorf("root guard original already used or inaccessible: %w", err)
	}
	f := os.NewFile(uintptr(fd), path)
	_, err = f.Write(content)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
func nativePeerGuardJournalLoadAt(dir string, uid uint32, original string) (nativePeerGuardJournal, error) {
	var j nativePeerGuardJournal
	path, err := nativePeerGuardJournalPathAt(dir, original)
	if err != nil {
		return j, err
	}
	st, err := os.Lstat(path)
	if err != nil {
		return j, err
	}
	stat, ok := st.Sys().(*syscall.Stat_t)
	if !ok || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 ||
		st.Mode()&os.ModeSymlink != 0 || stat.Uid != uid || stat.Nlink != 1 {
		return j, errors.New("root guard journal inode owner/mode/links unverified")
	}
	f, err := os.Open(path)
	if err != nil {
		return j, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(data) > 65536 {
		return j, errors.New("root guard journal content untrusted or oversized")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&j); err != nil {
		return j, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return j, errors.New("root journal contains trailing data")
	}
	if j.Schema != 1 || j.Scope.OriginalID != original ||
		nativePeerGuardScopeValid(j.Scope) != nil ||
		(j.Status != "DISPATCH_UNKNOWN" && j.Status != "GUARD_STEP_KERNEL_VERIFIED") {
		return j, errors.New("unrecognized or mismatched guarded root original journal")
	}
	if j.Status == "DISPATCH_UNKNOWN" {
		if j.Outcome.Attempted || j.Outcome.KernelVerified {
			return j, errors.New("unexpected claimed complete effect in UNKNOWN root intent")
		}
	} else if !j.Outcome.Attempted || !j.Outcome.KernelVerified ||
		j.Outcome.BeforePlanSHA != j.Scope.BeforeSHA256 ||
		j.Outcome.AfterPlanSHA != j.Scope.AfterSHA256 {
		return j, errors.New("root guard journal claims incomplete kernel step")
	}
	return j, nil
}
func nativePeerGuardJournalCommitAt(dir string, uid uint32,
	initial nativePeerGuardJournal, outcome nativePeerGuardReplaceOutcome) error {
	previous, err := nativePeerGuardJournalLoadAt(dir, uid, initial.Scope.OriginalID)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(previous, initial) || previous.Status != "DISPATCH_UNKNOWN" ||
		!outcome.Attempted || !outcome.KernelVerified ||
		outcome.BeforePlanSHA != initial.Scope.BeforeSHA256 ||
		outcome.AfterPlanSHA != initial.Scope.AfterSHA256 {
		return errors.New("root guard kernel step or original preimage unverified")
	}
	verified := initial
	verified.Status = "GUARD_STEP_KERNEL_VERIFIED"
	verified.Outcome = outcome
	data, err := json.Marshal(verified)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path, err := nativePeerGuardJournalPathAt(dir, initial.Scope.OriginalID)
	if err != nil {
		return err
	}
	if err := atomicWrite(path, data, 0600); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		return err
	}
	actual, err := nativePeerGuardJournalLoadAt(dir, uid, initial.Scope.OriginalID)
	if err != nil || !reflect.DeepEqual(actual, verified) {
		return errors.New("root guard completed step durable readback mismatch")
	}
	return nil
}

// nativePeerGuardJournalOnce is not exposed through the installed Root
// Helper. The FUTURE trusted caller must already hold original+resource
// locks, verify the independently authenticated exact SQLite Actor and
// complete native/foreign/bridge/nft state, and disallow any other writer.
// No automatic cleanup, replay or optimistic recovery is permitted.
func nativePeerGuardJournalOnce(ctx context.Context, dir string, uid uint32,
	s nativePeerGuardJournalScope, effect func(context.Context) (nativePeerGuardReplaceOutcome, error),
	beforeCommit func() error) (nativePeerGuardJournalReceipt, error) {
	receipt := nativePeerGuardJournalReceipt{
		Status: "NOT_DISPATCHED", OriginalID: s.OriginalID,
		BeforeSHA256: s.BeforeSHA256, AfterSHA256: s.AfterSHA256,
	}
	if effect == nil || nativePeerGuardScopeValid(s) != nil {
		return receipt, errors.New("unapproved root guard journal original or missing bounded effect")
	}
	if err := ctx.Err(); err != nil {
		return receipt, err
	}
	initial := nativePeerGuardJournal{Schema: 1, Status: "DISPATCH_UNKNOWN", Scope: s}
	if err := nativePeerGuardJournalIntentAt(dir, uid, initial); err != nil {
		return receipt, err
	}
	receipt.Status = "DISPATCH_UNKNOWN"
	// After the durable intent has been fsynced, every fault must preserve
	// the same UNKNOWN. The next call cannot use the same original file.
	if err := ctx.Err(); err != nil {
		return receipt, fmt.Errorf("guard original UNKNOWN after intent sync: %w", err)
	}
	outcome, err := effect(ctx)
	receipt.NativeAttempted = outcome.Attempted
	if err != nil {
		return receipt, fmt.Errorf("guard original effect UNKNOWN, never replay: %w", err)
	}
	if !outcome.Attempted || !outcome.KernelVerified ||
		outcome.BeforePlanSHA != s.BeforeSHA256 ||
		outcome.AfterPlanSHA != s.AfterSHA256 {
		return receipt, errors.New("guard original kernel poststate missing; UNKNOWN")
	}
	if beforeCommit != nil {
		if err := beforeCommit(); err != nil {
			return receipt, fmt.Errorf("guard original commit fault; UNKNOWN: %w", err)
		}
	}
	if err := nativePeerGuardJournalCommitAt(dir, uid, initial, outcome); err != nil {
		return receipt, fmt.Errorf("guard original completion not durable; UNKNOWN: %w", err)
	}
	receipt.Status = "GUARD_STEP_KERNEL_VERIFIED"
	receipt.GuardStepVerified = true
	return receipt, nil
}
