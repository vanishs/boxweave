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
	"slices"
	"strings"
	"syscall"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/networklimits"
	"github.com/vanishs/boxweave/internal/state"
	"golang.org/x/sys/unix"
)

// A separate append-only Root policy log. Its records contain typed
// members/grants, never executable nft syntax. A complete journal without
// a published policy is still uncertain and may not be restored at boot.
type scopedPeerBootPolicy struct {
	Schema, Sequence                                                int
	OriginalID, HostSHA256, Project, Bridge, PreviousSHA256         string
	SourceUUID, TargetUUID                                          string
	Allow                                                           bool
	BeforeSHA256, AfterSHA256, ProofSHA256, PlanSHA256, FinalSHA256 string
	Members                                                         []NativePeerGuardMember
	Active                                                          []state.PeerGrantBinding
}

const scopedPeerBootPolicyLimit = networklimits.ScopedPeerHistoryLimit

func scopedPeerBootPolicyName(n int) string { return fmt.Sprintf("policy-%08d.json", n) }

func scopedPeerBootRead(d *os.File, uid uint32, name string) (scopedPeerBootPolicy, error) {
	var zero scopedPeerBootPolicy
	fd, e := unix.Openat(int(d.Fd()), name, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
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
		return zero, errors.New("untrusted boot policy inode")
	}
	b, e := io.ReadAll(io.LimitReader(f, 262145))
	if e != nil || len(b) > 262144 {
		return zero, errors.New("oversized boot policy")
	}
	var p scopedPeerBootPolicy
	if json.Unmarshal(b, &p) != nil {
		return zero, errors.New("invalid boot policy")
	}
	canonical, e := json.Marshal(p)
	if e != nil || !bytes.Equal(b, append(canonical, '\n')) {
		return zero, errors.New("noncanonical boot policy")
	}
	return p, nil
}
func scopedPeerBootWrite(d *os.File, p scopedPeerBootPolicy) error {
	b, e := json.Marshal(p)
	if e != nil || len(b) > 262143 {
		return errors.New("invalid boot policy length")
	}
	fd, e := unix.Openat(int(d.Fd()), scopedPeerBootPolicyName(p.Sequence), unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if e != nil {
		return e
	}
	f := os.NewFile(uintptr(fd), "boot policy")
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
func scopedPeerBootCompile(r nativePeerFreshJournalRecord, p scopedPeerBootPolicy) (NativePeerGuardPlan, error) {
	if p.Schema != 2 || p.Sequence < 1 || p.Sequence > scopedPeerBootPolicyLimit || !scopedLifecycleID.MatchString(p.OriginalID) || p.HostSHA256 != r.HostSHA256 || p.Project != r.Project || p.Bridge != r.Bridge ||
		!scopedPeerUUID.MatchString(p.SourceUUID) || !scopedPeerUUID.MatchString(p.TargetUUID) || p.SourceUUID == p.TargetUUID {
		return NativePeerGuardPlan{}, errors.New("untrusted boot policy identity")
	}
	for _, h := range []string{p.PreviousSHA256, p.BeforeSHA256, p.AfterSHA256, p.ProofSHA256, p.PlanSHA256, p.FinalSHA256} {
		if !nativePeerGuardSHA256Valid(h) {
			return NativePeerGuardPlan{}, errors.New("invalid boot policy digest")
		}
	}
	nw := &api.Network{Name: r.Bridge, Type: "bridge", Managed: true, Status: "Created", NetworkPut: api.NetworkPut{Config: api.ConfigMap{"ipv4.address": r.IPv4CIDR, "ipv6.address": r.IPv6CIDR}}}
	target := NativePrepareTarget{Project: r.Project, Network: r.Bridge, ACL: "bw-fresh-initial-guard"}
	if len(p.Active) == 0 && len(p.Members) == 0 {
		return nativePeerFreshInitialGuard(target, nw)
	}
	return NativePeerGuardFromSnapshot(target, nw, p.Members, p.Active, nil)
}

// Independently validate the complete immutable step chain against the
// typed boot policy. No current ACL plan or service-owned DB is needed
// before Incus starts. This does not grant any mutation authority.
func scopedPeerBootJournal(dir string, uid uint32, p scopedPeerBootPolicy) error {
	d, e := nativePeerFreshJournalOpenAt(filepath.Join(dir, "peer-"+p.OriginalID), uid)
	if e != nil {
		return e
	}
	defer d.Close()
	names, e := d.Readdirnames(-1)
	if e != nil || len(names) != 9 {
		return errors.New("incomplete boot original journal")
	}
	final, e := scopedPeerEffectRead(d, uid, "complete.json")
	if e != nil || final.Schema != 2 || final.Status != "COMPLETE_VERIFIED" || final.Index != 4 ||
		final.OriginalID != p.OriginalID || final.Project != p.Project || final.Bridge != p.Bridge ||
		final.SourceUUID != p.SourceUUID || final.TargetUUID != p.TargetUUID || final.Allow != p.Allow ||
		final.Scope.HostSHA256 != p.HostSHA256 || final.Scope.LedgerSHA256 != p.ProofSHA256 ||
		!nativePeerGuardSHA256Valid(final.Scope.NativeSHA256) || final.PlanSHA256 != p.PlanSHA256 ||
		final.BeforeGuardSHA256 != p.BeforeSHA256 || final.AfterGuardSHA256 != p.AfterSHA256 ||
		final.PostimageSHA256 != p.FinalSHA256 || final.Kind != "" || final.OwnerUUID != "" || final.GuardSHA256 != "" || final.PreimageSHA256 != "" {
		return errors.New("boot policy terminal journal mismatch")
	}
	deny := p.BeforeSHA256
	if !p.Allow {
		deny = p.AfterSHA256
	}
	a := nativePeerScopedEffectStep{Kind: nativePeerScopedACLSource, OwnerUUID: p.SourceUUID, GuardMustMatchSHA256: deny}
	b := nativePeerScopedEffectStep{Kind: nativePeerScopedACLTarget, OwnerUUID: p.TargetUUID, GuardMustMatchSHA256: deny}
	g := nativePeerScopedEffectStep{Kind: nativePeerScopedGuardBatch, GuardMustMatchSHA256: p.BeforeSHA256}
	c := nativePeerScopedEffectStep{Kind: nativePeerScopedExactConntrack, GuardMustMatchSHA256: deny}
	steps := []nativePeerScopedEffectStep{a, b, c, g}
	if !p.Allow {
		steps = []nativePeerScopedEffectStep{g, c, a, b}
	}
	base := final
	base.Index = 0
	base.Status = ""
	base.PostimageSHA256 = ""
	prior := ""
	for i, s := range steps {
		a, e := scopedPeerEffectRead(d, uid, scopedPeerEffectRecordName(i, false))
		if e != nil || !nativePeerGuardSHA256Valid(a.PreimageSHA256) || i > 0 && a.PreimageSHA256 != prior || a != scopedPeerStageRecord(base, i, s, a.PreimageSHA256) {
			return errors.New("boot policy stage intent mismatch")
		}
		b, e := scopedPeerEffectRead(d, uid, scopedPeerEffectRecordName(i, true))
		if e != nil || !nativePeerGuardSHA256Valid(b.PostimageSHA256) {
			return errors.New("boot policy stage readback missing")
		}
		a.Status = "STAGE_VERIFIED"
		a.PostimageSHA256 = b.PostimageSHA256
		if a != b {
			return errors.New("boot policy stage receipt mismatch")
		}
		prior = b.PostimageSHA256
	}
	if prior != p.FinalSHA256 {
		return errors.New("boot policy final postimage mismatch")
	}
	return nil
}

func scopedPeerBootCurrent(dir string, uid uint32, r nativePeerFreshJournalRecord, initial NativePeerGuardPlan, except string) (NativePeerGuardPlan, scopedPeerBootPolicy, error) {
	var last scopedPeerBootPolicy
	var zero NativePeerGuardPlan
	d, e := nativePeerFreshJournalOpenAt(dir, uid)
	if e != nil {
		return zero, last, e
	}
	defer d.Close()
	names, e := d.Readdirnames(-1)
	if e != nil {
		return zero, last, e
	}
	var files []string
	pending := map[string]bool{}
	for _, name := range names {
		if strings.HasPrefix(name, "policy-") {
			files = append(files, name)
		}
		if strings.HasPrefix(name, "peer-") {
			id := strings.TrimPrefix(name, "peer-")
			if !scopedLifecycleID.MatchString(id) {
				return zero, last, errors.New("invalid boot Peer journal name")
			}
			pending[id] = true
		}
	}
	if len(files) > scopedPeerBootPolicyLimit || len(pending) > scopedPeerBootPolicyLimit {
		return zero, last, errors.New("boot policy history capacity reached")
	}
	slices.Sort(files)
	current := initial
	prev := initial.DigestSHA256
	for i, name := range files {
		if name != scopedPeerBootPolicyName(i+1) {
			return zero, last, errors.New("noncontiguous boot policy history")
		}
		p, e := scopedPeerBootRead(d, uid, name)
		if e != nil || p.Sequence != i+1 || p.PreviousSHA256 != prev || p.BeforeSHA256 != current.DigestSHA256 || !pending[p.OriginalID] || p.OriginalID == except {
			return zero, last, errors.New("boot policy lineage mismatch")
		}
		next, e := scopedPeerBootCompile(r, p)
		if e != nil || next.DigestSHA256 != p.AfterSHA256 || nativePeerGuardValidateOneDirectionDelta(current, next) != nil || !scopedPeerBootDelta(last.Active, p) {
			return zero, last, errors.New("untrusted boot policy direction delta")
		}
		if e = scopedPeerBootJournal(dir, uid, p); e != nil {
			return zero, last, e
		}
		delete(pending, p.OriginalID)
		prev = nativePrepareStateDigest("scoped-peer-boot-policy-v2", p)
		current = next
		last = p
	}
	if except != "" {
		if !pending[except] {
			return zero, last, errors.New("missing pending policy original")
		}
		delete(pending, except)
	}
	if len(pending) != 0 {
		return zero, last, errors.New("unpublished Peer original: refuse boot restore")
	}
	return current, last, nil
}

func scopedPeerBootDelta(before []state.PeerGrantBinding, p scopedPeerBootPolicy) bool {
	a, b := map[string]state.PeerGrantBinding{}, map[string]state.PeerGrantBinding{}
	for _, q := range before {
		key := q.SourceUUID + "/" + q.TargetUUID
		if _, ok := a[key]; ok {
			return false
		}
		a[key] = q
	}
	for _, q := range p.Active {
		key := q.SourceUUID + "/" + q.TargetUUID
		if _, ok := b[key]; ok {
			return false
		}
		b[key] = q
	}
	key := p.SourceUUID + "/" + p.TargetUUID
	_, hasA := a[key]
	_, hasB := b[key]
	if p.Allow && (!hasB || hasA) || !p.Allow && (!hasA || hasB) {
		return false
	}
	delete(a, key)
	delete(b, key)
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// Publish only after all four native stages have a complete root receipt
// and the current kernel guard independently matches the exact after image.
// Root caller holds the common lifecycle/effect lock.
func scopedPeerBootPublish(ctx context.Context, dir string, uid uint32, r nativePeerFreshJournalRecord, initial NativePeerGuardPlan, proof state.ScopedPeerOriginalProof, rec state.ScopedPeerRootReceipt, witness nativePeerGuardWitness) error {
	if witness == nil || proof.RecoveryOnly || rec.Status != "COMPLETE_VERIFIED" || rec.VerifiedSteps != 4 || rec.RecoveryOnly || rec.OriginalID != proof.OriginalID || rec.Project != r.Project || rec.Bridge != r.Bridge || rec.ProofSHA256 != state.ScopedPeerProofSHA256(proof) {
		return errors.New("unverified boot publication original")
	}
	current, last, e := scopedPeerBootCurrent(dir, uid, r, initial, rec.OriginalID)
	if e != nil {
		return e
	}
	p := scopedPeerBootPolicy{Schema: 2, Sequence: last.Sequence + 1, OriginalID: rec.OriginalID, HostSHA256: r.HostSHA256, Project: r.Project, Bridge: r.Bridge, SourceUUID: proof.Intent.SourceUUID, TargetUUID: proof.Intent.TargetUUID, Allow: proof.Intent.Allow, PreviousSHA256: initial.DigestSHA256, BeforeSHA256: current.DigestSHA256, ProofSHA256: rec.ProofSHA256, PlanSHA256: rec.PlanSHA256, FinalSHA256: rec.FinalSHA256, Active: proof.After}
	if last.Sequence != 0 {
		p.PreviousSHA256 = nativePrepareStateDigest("scoped-peer-boot-policy-v2", last)
	}
	for _, b := range proof.Inventory.Boxes {
		p.Members = append(p.Members, NativePeerGuardMember{UUID: b.UUID, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6})
	}
	// Compile without executable text from storage; the journal binds the
	// expected after digest to the complete effect receipt.
	p.AfterSHA256 = strings.Repeat("0", 64)
	next, e := scopedPeerBootCompile(r, p)
	if e != nil {
		return e
	}
	p.AfterSHA256 = next.DigestSHA256
	if !scopedPeerBootDelta(last.Active, p) || nativePeerGuardValidateOneDirectionDelta(current, next) != nil {
		return errors.New("boot policy delta differs from original")
	}
	if e = scopedPeerBootJournal(dir, uid, p); e != nil {
		return e
	}
	if e = witness(ctx, next); e != nil {
		return e
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	d, e := nativePeerFreshJournalOpenAt(dir, uid)
	if e != nil {
		return e
	}
	defer d.Close()
	if e = scopedPeerBootWrite(d, p); e != nil {
		return e
	}
	got, _, e := scopedPeerBootCurrent(dir, uid, r, initial, "")
	if e != nil || got != next {
		return errors.New("boot policy durable readback failed")
	}
	return nil
}
