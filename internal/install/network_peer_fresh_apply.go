package install

import (
	"context"
	"errors"
	"os"

	"github.com/lxc/incus/v7/shared/api"
)

type NativePeerFreshGuardReview struct {
	Mode, HostSHA256, Project, Bridge, NativeSHA256, GuardSHA256, ReviewSHA256 string
	IPv4CIDR, IPv6CIDR                                                         string
}

func nativePeerFreshGuardReviewFrom(host string, e nativePeerFreshEvidence) NativePeerFreshGuardReview {
	r := NativePeerFreshGuardReview{Mode: nativePeerFreshMode, HostSHA256: host, Project: e.Project, Bridge: e.Bridge,
		NativeSHA256: e.NativeSHA256, GuardSHA256: e.Guard.DigestSHA256, IPv4CIDR: e.IPv4CIDR, IPv6CIDR: e.IPv6CIDR}
	r.ReviewSHA256 = nativePrepareStateDigest("fresh-guard-review-v2", r)
	return r
}

// Internal root-local native setup, not a new MCP operation. A future
// installer must hold the common maintenance lock across the whole setup.
func nativePeerFreshGuardReview(ctx context.Context) (NativePeerFreshGuardReview, error) {
	var zero NativePeerFreshGuardReview
	host, err := migrationHostDigest()
	if err != nil {
		return zero, err
	}
	e, err := nativePeerFreshCollectReadOnly(ctx, false)
	if err != nil {
		return zero, err
	}
	if err = nativePeerFreshGuardAbsent(ctx, e.Guard); err != nil {
		return zero, err
	}
	return nativePeerFreshGuardReviewFrom(host, e), nil
}

func nativePeerFreshGuardApply(ctx context.Context, approvedSHA string, approved bool) (nativePeerFreshJournalRecord, error) {
	var zero nativePeerFreshJournalRecord
	if os.Geteuid() != 0 || !approved || !nativePeerGuardSHA256Valid(approvedSHA) {
		return zero, errors.New("explicit reviewed fresh local root approval required")
	}
	review, err := nativePeerFreshGuardReview(ctx)
	if err != nil {
		return zero, err
	}
	if review.ReviewSHA256 != approvedSHA {
		return zero, errors.New("fresh guard review changed")
	}
	inspect := func(ctx context.Context) (nativePeerFreshEvidence, error) {
		var empty nativePeerFreshEvidence
		host, e := migrationHostDigest()
		if e != nil || host != review.HostSHA256 {
			return empty, errors.New("fresh guard host identity changed")
		}
		observed, e := nativePeerFreshRootSnapshot(ctx)
		if e != nil {
			return empty, e
		}
		if nativePeerFreshGuardReviewFrom(host, observed) != review {
			return empty, errors.New("fresh reviewed native evidence changed")
		}
		return observed, nil
	}
	return nativePeerFreshBootstrapOnce(ctx, nativePeerFreshJournalDir, 0, review.HostSHA256, false, os.Lstat, inspect,
		nativePeerFreshGuardAbsent, nativePeerFreshGuardInstall, NativePeerGuardKernelReadback)
}

func nativePeerFreshPlanFromRecord(r nativePeerFreshJournalRecord) (NativePeerGuardPlan, error) {
	var zero NativePeerGuardPlan
	if !nativePeerFreshJournalRecordValid(r) {
		return zero, errors.New("untrusted fresh guard record")
	}
	nw := &api.Network{Name: r.Bridge, Type: "bridge", Managed: true, Status: "Created",
		NetworkPut: api.NetworkPut{Config: api.ConfigMap{"ipv4.address": r.IPv4CIDR, "ipv6.address": r.IPv6CIDR}}}
	p, err := nativePeerFreshInitialGuard(NativePrepareTarget{Project: r.Project, Network: r.Bridge, ACL: "bw-fresh-initial-guard"}, nw)
	if err != nil {
		return zero, err
	}
	if err = nativePeerFreshEvidenceValid(nativePeerFreshEvidence{Project: r.Project, Bridge: r.Bridge, NativeSHA256: r.NativeSHA256,
		IPv4CIDR: r.IPv4CIDR, IPv6CIDR: r.IPv6CIDR, Guard: p}); err != nil || p.DigestSHA256 != r.GuardSHA256 {
		return zero, errors.New("fresh guard record cannot reproduce exact reviewed policy")
	}
	return p, nil
}

// Reproduce the exact reviewed initial policy from the fixed root-owned
// journal, not from service-owned SQLite or RPC parameters. Until N3, only
// the no-grant initial policy is supported. A conflicting table is never
// flushed or adopted.
func nativePeerFreshVerifiedPlan() (nativePeerFreshJournalRecord, NativePeerGuardPlan, error) {
	var r nativePeerFreshJournalRecord
	var p NativePeerGuardPlan
	if os.Geteuid() != 0 {
		return r, p, errors.New("fresh guard requires local root")
	}
	d, err := nativePeerFreshJournalOpenAt(nativePeerFreshJournalDir, 0)
	if err != nil {
		return r, p, err
	}
	defer d.Close()
	r, err = nativePeerFreshJournalRead(d, "INITIAL_GUARD_VERIFIED")
	if err != nil {
		return r, p, err
	}
	if err = nativePeerFreshJournalVerifiedAt(nativePeerFreshJournalDir, 0, r); err != nil {
		return r, p, err
	}
	h, err := migrationHostDigest()
	if err != nil || h != r.HostSHA256 {
		return r, p, errors.New("fresh guard host changed")
	}
	p, err = nativePeerFreshPlanFromRecord(r)
	return r, p, err
}

func nativePeerFreshGuardRestore(ctx context.Context) error {
	_, p, err := nativePeerFreshVerifiedPlan()
	if err != nil {
		return err
	}
	present, err := nativePeerFreshTableLive(ctx, p)
	if err != nil {
		return err
	}
	if !present {
		if err = nativePeerFreshGuardInstall(ctx, p); err != nil {
			return err
		}
	}
	return NativePeerGuardKernelReadback(ctx, p)
}

func nativePeerFreshGuardCurrent(ctx context.Context, project, bridge string) (nativePeerFreshJournalRecord, error) {
	r, p, err := nativePeerFreshVerifiedPlan()
	if err != nil {
		return r, err
	}
	if r.Project != project || r.Bridge != bridge {
		return r, errors.New("fresh guard scope changed")
	}
	return r, NativePeerGuardKernelReadback(ctx, p)
}
