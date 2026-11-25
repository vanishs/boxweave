package install

import (
	"context"
	"errors"
	"fmt"
	"github.com/lxc/incus/v7/shared/api"
	"io/fs"
	"reflect"
	"strings"
)

// Inert fresh-only preflight, not an installed entrypoint or authorization.
const nativePeerFreshMode = "SCOPED_PEER_FRESH_V2"

type nativePeerFreshEvidence struct {
	Project, Bridge    string
	IPv4CIDR, IPv6CIDR string
	NativeSHA256       string
	Guard              NativePeerGuardPlan
}

// Lstat refuses even dangling links, empty legacy state and unreadable paths.
func nativePeerFreshPathsAbsent(lstat func(string) (fs.FileInfo, error)) error {
	return nativePeerFreshPathsAbsentExceptJournal(lstat, false)
}

func nativePeerFreshPathsAbsentExceptJournal(lstat func(string) (fs.FileInfo, error), ownJournal bool) error {
	if lstat == nil {
		return errors.New("fresh install path inventory unavailable")
	}
	for _, path := range []string{
		"/var/lib/boxweave", "/var/lib/boxweave-installer-backup",
		"/usr/local/bin/boxweave", "/etc/systemd/system/boxweave.service",
		"/etc/systemd/system/boxweave.service.d", nativePrepareJournalPath,
		nativePeerRootUnitPath, nativeRetirementUnitPath, nativeRetirementSocketPath,
		nativePeerGuardRootJournalDir, nativePeerFreshJournalDir,
		scopedGuardUnitPath, scopedLifecycleUnitPath, scopedIncusDropIn, scopedLifecycleSocket, scopedRetirementFence, retirementPurgeFence, ScopedDrainFence, nativePeerRootSocketPath, nativePeerConntrackJournalPath, nativePeerConntrackOperationsDir, nativePeerConntrackLockPath,
	} {
		if ownJournal && path == nativePeerFreshJournalDir {
			continue
		}
		info, err := lstat(path)
		if info != nil || !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("unsupported existing or unreadable BoxWeave installation: %s; no upgrade or adoption", path)
		}
	}
	return nil
}

// No network, database or service effect. A future caller must hold the
// installer setup lock and separately prove guard/boot fencing.
func nativePeerFreshAdmission(ctx context.Context, upgrade bool,
	lstat func(string) (fs.FileInfo, error),
	inspect func(context.Context) (nativePeerFreshEvidence, error)) (nativePeerFreshEvidence, error) {
	var zero nativePeerFreshEvidence
	if upgrade {
		return zero, errors.New("scoped Peer v2 requires fresh installation; existing upgrade unsupported")
	}
	if ctx == nil || inspect == nil {
		return zero, errors.New("missing fresh installation evidence collector")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if err := nativePeerFreshPathsAbsent(lstat); err != nil {
		return zero, err
	}
	first, err := inspect(ctx)
	if err != nil {
		return zero, err
	}
	if err := nativePeerFreshEvidenceValid(first); err != nil {
		return zero, err
	}
	second, err := inspect(ctx)
	if err != nil || !reflect.DeepEqual(first, second) {
		return zero, errors.New("fresh native evidence changed or unavailable between reads")
	}
	if err := nativePeerFreshPathsAbsent(lstat); err != nil {
		return zero, err
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	return first, nil
}

// Detached legacy ACLs are ambiguous too. No native adoption path.
func nativePeerFreshACLInventory(names []string) error {
	for _, name := range names {
		if name == "" || strings.HasPrefix(name, "bw-") {
			return errors.New("existing or ambiguous BoxWeave ACL; fresh scoped installation refuses adoption")
		}
	}
	return nil
}

// Recompile the exact empty policy from independently observed bridge
// addresses. A digest of a caller-constructed script alone is not evidence.
func nativePeerFreshEvidenceValid(e nativePeerFreshEvidence) error {
	if !strings.HasPrefix(e.Project, "user-") || e.Bridge != "incusbr-"+strings.TrimPrefix(e.Project, "user-") ||
		!validRetirementReviewSHA(e.NativeSHA256) {
		return errors.New("fresh scoped native evidence incomplete")
	}
	network := &api.Network{Name: e.Bridge, Type: "bridge", Managed: true, Status: "Created",
		NetworkPut: api.NetworkPut{Config: api.ConfigMap{"ipv4.address": e.IPv4CIDR, "ipv6.address": e.IPv6CIDR}}}
	want, err := nativePeerFreshInitialGuard(NativePrepareTarget{Project: e.Project, Network: e.Bridge, ACL: "bw-fresh-initial-guard"}, network)
	if err != nil || want != e.Guard {
		return errors.New("fresh scoped guard does not match exact empty policy")
	}
	return nil
}
