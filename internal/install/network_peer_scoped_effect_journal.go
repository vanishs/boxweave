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
	"syscall"

	"golang.org/x/sys/unix"
)

// Whole-original durable effect journal. Only a trusted, locked Root
// worker may supply the independently collected proof digests and callbacks.
// No installed RPC or CLI exposes this seam. The journal never authorizes
// a replay, even if no stage was dispatched.
type scopedPeerEffectJournalScope struct {
	HostSHA256, LedgerSHA256, NativeSHA256 string
}
type scopedPeerEffectRecord struct {
	Schema                                                          int
	OriginalID, Project, Bridge, SourceUUID, TargetUUID             string
	Allow                                                           bool
	Scope                                                           scopedPeerEffectJournalScope
	PlanSHA256, BeforeGuardSHA256, AfterGuardSHA256                 string
	Index                                                           int
	Kind                                                            nativePeerScopedEffectKind
	OwnerUUID, GuardSHA256, Status, PreimageSHA256, PostimageSHA256 string
}
type scopedPeerEffectReceipt struct {
	OriginalID, PlanSHA256, Status string
	VerifiedSteps                  int
	FinalSHA256                    string
}
type scopedPeerPrefixWitness func(context.Context, int) (string, error)
type scopedPeerEffectApply func(context.Context, nativePeerScopedEffectStep) error

func scopedPeerEffectRecordName(i int, verified bool) string {
	suffix := "intent"
	if verified {
		suffix = "verified"
	}
	return fmt.Sprintf("step-%02d-%s.json", i, suffix)
}
func scopedPeerEffectWrite(d *os.File, name string, r scopedPeerEffectRecord) error {
	b, e := json.Marshal(r)
	if e != nil {
		return e
	}
	fd, e := unix.Openat(int(d.Fd()), name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if e != nil {
		return e
	}
	f := os.NewFile(uintptr(fd), name)
	_, e = f.Write(append(b, '\n'))
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return d.Sync()
}
func scopedPeerEffectRead(d *os.File, uid uint32, name string) (scopedPeerEffectRecord, error) {
	var zero scopedPeerEffectRecord
	fd, e := unix.Openat(int(d.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if e != nil {
		return zero, e
	}
	f := os.NewFile(uintptr(fd), name)
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return zero, e
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok || sys.Uid != uid || sys.Nlink != 1 || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return zero, errors.New("untrusted scoped Peer journal record")
	}
	b, e := io.ReadAll(io.LimitReader(f, 8193))
	if e != nil || len(b) > 8192 {
		return zero, errors.New("invalid scoped Peer journal size")
	}
	var r scopedPeerEffectRecord
	if json.Unmarshal(b, &r) != nil {
		return zero, errors.New("invalid scoped Peer journal")
	}
	raw, e := json.Marshal(r)
	if e != nil || !bytes.Equal(b, append(raw, '\n')) {
		return zero, errors.New("noncanonical scoped Peer journal")
	}
	return r, nil
}
func scopedPeerEffectBase(p NativePeerScopedRootTransition, s scopedPeerEffectJournalScope) (scopedPeerEffectRecord, []nativePeerScopedEffectStep, error) {
	var zero scopedPeerEffectRecord
	steps, e := nativePeerScopedRootEffectSequence(p)
	if e != nil || len(steps) != 4 || !safePrereqName.MatchString(p.Project) || !safePrereqName.MatchString(p.Bridge) {
		return zero, nil, errors.New("untrusted complete Peer effect plan")
	}
	for _, h := range []string{s.HostSHA256, s.LedgerSHA256, s.NativeSHA256} {
		if !nativePeerGuardSHA256Valid(h) {
			return zero, nil, errors.New("missing independently collected Root scope")
		}
	}
	return scopedPeerEffectRecord{Schema: 2, OriginalID: p.OriginalID, Project: p.Project, Bridge: p.Bridge, SourceUUID: p.SourceUUID, TargetUUID: p.TargetUUID, Allow: p.IntentAllow, Scope: s, PlanSHA256: nativePrepareStateDigest("scoped-peer-full-sequence-v2", p), BeforeGuardSHA256: p.GuardBefore.DigestSHA256, AfterGuardSHA256: p.GuardAfter.DigestSHA256}, steps, nil
}
func scopedPeerStageRecord(base scopedPeerEffectRecord, i int, step nativePeerScopedEffectStep, pre string) scopedPeerEffectRecord {
	base.Index = i
	base.Kind = step.Kind
	base.OwnerUUID = step.OwnerUUID
	base.GuardSHA256 = step.GuardMustMatchSHA256
	base.Status = "DISPATCH_UNKNOWN"
	base.PreimageSHA256 = pre
	return base
}
func scopedPeerWitness(ctx context.Context, w scopedPeerPrefixWitness, prefix int) (string, error) {
	if e := ctx.Err(); e != nil {
		return "", e
	}
	h, e := w(ctx, prefix)
	if e != nil || !nativePeerGuardSHA256Valid(h) {
		return "", errors.New("full scoped Peer prefix unverified")
	}
	if e = ctx.Err(); e != nil {
		return "", e
	}
	return h, nil
}
func nativeScopedPeerEffectsOnce(ctx context.Context, dir string, uid uint32, p NativePeerScopedRootTransition, s scopedPeerEffectJournalScope, w scopedPeerPrefixWitness, apply scopedPeerEffectApply) (scopedPeerEffectReceipt, error) {
	rec := scopedPeerEffectReceipt{OriginalID: p.OriginalID, Status: "NOT_DISPATCHED"}
	base, steps, e := scopedPeerEffectBase(p, s)
	if e != nil || w == nil || apply == nil || ctx == nil {
		return rec, errors.New("missing locked scoped Peer worker prerequisites")
	}
	rec.PlanSHA256 = base.PlanSHA256
	h, e := scopedPeerWitness(ctx, w, 0)
	if e != nil {
		return rec, e
	}
	// An existing original is NEVER adopted, resumed, or overwritten.
	path := filepath.Join(dir, "peer-"+p.OriginalID)
	d, e := nativePeerFreshJournalCreateAt(path, uid)
	if e != nil {
		return rec, e
	}
	defer d.Close()
	rec.Status = "DISPATCH_UNKNOWN"
	for i, step := range steps {
		r := scopedPeerStageRecord(base, i, step, h)
		if e = scopedPeerEffectWrite(d, scopedPeerEffectRecordName(i, false), r); e != nil {
			return rec, e
		}
		now, e := scopedPeerWitness(ctx, w, i)
		if e != nil || now != h {
			return rec, errors.New("scoped Peer prefix drift after durable intent")
		}
		if e = apply(ctx, step); e != nil {
			return rec, fmt.Errorf("scoped Peer stage %d UNKNOWN: %w", i, e)
		}
		post, e := scopedPeerWitness(ctx, w, i+1)
		if e != nil {
			return rec, e
		}
		r.Status = "STAGE_VERIFIED"
		r.PostimageSHA256 = post
		if e = scopedPeerEffectWrite(d, scopedPeerEffectRecordName(i, true), r); e != nil {
			return rec, e
		}
		rec.VerifiedSteps = i + 1
		h = post
	}
	final, e := scopedPeerWitness(ctx, w, len(steps))
	if e != nil || final != h {
		return rec, errors.New("scoped Peer final whole-state drift")
	}
	base.Index = len(steps)
	base.Status = "COMPLETE_VERIFIED"
	base.PostimageSHA256 = final
	if e = scopedPeerEffectWrite(d, "complete.json", base); e != nil {
		return rec, e
	}
	verified, e := nativeScopedPeerEffectsReadOnly(dir, uid, p, s)
	if e != nil {
		return rec, e
	}
	return verified, nil
}

// Read-only verification of a terminal journal chain. This is historical
// evidence only. Later recovery must independently reprove live state and
// original authority; this function cannot dispatch an effect.
func nativeScopedPeerEffectsReadOnly(dir string, uid uint32, p NativePeerScopedRootTransition, s scopedPeerEffectJournalScope) (scopedPeerEffectReceipt, error) {
	rec := scopedPeerEffectReceipt{OriginalID: p.OriginalID, Status: "DISPATCH_UNKNOWN"}
	base, steps, e := scopedPeerEffectBase(p, s)
	if e != nil {
		return rec, e
	}
	rec.PlanSHA256 = base.PlanSHA256
	d, e := nativePeerFreshJournalOpenAt(filepath.Join(dir, "peer-"+p.OriginalID), uid)
	if e != nil {
		return rec, e
	}
	defer d.Close()
	names, e := d.Readdirnames(-1)
	if e != nil || len(names) != 2*len(steps)+1 {
		return rec, errors.New("incomplete or unexpected scoped Peer journal inventory")
	}
	prior := ""
	for i, step := range steps {
		a, e := scopedPeerEffectRead(d, uid, scopedPeerEffectRecordName(i, false))
		if e != nil {
			return rec, e
		}
		if !nativePeerGuardSHA256Valid(a.PreimageSHA256) || i > 0 && a.PreimageSHA256 != prior || a != scopedPeerStageRecord(base, i, step, a.PreimageSHA256) {
			return rec, errors.New("scoped Peer journal intent mismatch")
		}
		b, e := scopedPeerEffectRead(d, uid, scopedPeerEffectRecordName(i, true))
		if e != nil {
			return rec, e
		}
		if !nativePeerGuardSHA256Valid(b.PostimageSHA256) {
			return rec, errors.New("missing scoped Peer stage postimage")
		}
		a.Status = "STAGE_VERIFIED"
		a.PostimageSHA256 = b.PostimageSHA256
		if a != b {
			return rec, errors.New("scoped Peer stage receipt mismatch")
		}
		prior = b.PostimageSHA256
		rec.VerifiedSteps = i + 1
	}
	base.Index = len(steps)
	base.Status = "COMPLETE_VERIFIED"
	base.PostimageSHA256 = prior
	final, e := scopedPeerEffectRead(d, uid, "complete.json")
	if e != nil || final != base {
		return rec, errors.New("scoped Peer original terminal receipt mismatch")
	}
	rec.Status = base.Status
	rec.FinalSHA256 = prior
	return rec, nil
}
