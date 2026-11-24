package install

import (
	"context"
	"errors"
	"os"
	"reflect"
	"slices"
	"syscall"

	"github.com/lxc/incus/v7/shared/api"
)

// NativeRetirementObservation is a read-only precondition, never a deletion
// permit. The uninstall coordinator separately binds exact owned inventory.
type NativeRetirementObservation struct {
	Project           string `json:"project"`
	JournalStatus     string `json:"journal_status"`
	ReviewSHA256      string `json:"review_sha256"`
	ObservationSHA256 string `json:"observation_sha256,omitempty"`
	Ready             bool   `json:"ready"`
	Reason            string `json:"reason"`
}

func nativeRetirementSnapshot(rec NativePrepareRecord, project api.ProjectPut, profile api.ProfilePut, acl *api.NetworkACL) (string, error) {
	switch rec.Status {
	case "APPLIED":
		if (!reflect.DeepEqual(project, rec.BeforeProject) && !reflect.DeepEqual(project, rec.AfterProject)) ||
			(!reflect.DeepEqual(profile, rec.BeforeProfile) && !reflect.DeepEqual(profile, rec.AfterProfile)) ||
			!nativePrepareACLIsOwned(acl, rec) {
			return "", errors.New("NATIVE_CONFIG_DRIFT")
		}
		return nativePrepareStateDigest(project, profile, acl.Writable()), nil
	case "ROLLED_BACK":
		if !reflect.DeepEqual(project, rec.BeforeProject) || !reflect.DeepEqual(profile, rec.BeforeProfile) || acl != nil {
			return "", errors.New("NATIVE_CONFIG_DRIFT")
		}
		return nativePrepareStateDigest(project, profile, "ACL_ABSENT"), nil
	default:
		return "", errors.New("NATIVE_JOURNAL_UNVERIFIED")
	}
}

// NativePrepareRetirementCheck does not require an empty Project or a stopped
// controller. Those are separately verified before actual rollback. It detects
// third-party configuration drift BEFORE an operator approves Box retirement.
func NativePrepareRetirementCheck(ctx context.Context, project, review string) (NativeRetirementObservation, error) {
	out := NativeRetirementObservation{Project: project, ReviewSHA256: review, Reason: "NATIVE_RETIREMENT_CHECK_UNVERIFIED"}
	if err := refuseScopedLegacy(); err != nil {
		return out, err
	}
	st, err := os.Lstat(nativePrepareJournalPath)
	if err != nil {
		return out, err
	}
	owner, ok := st.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || owner.Nlink != 1 || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return out, errors.New("untrusted native preparation journal")
	}
	rec, err := NativePrepareStatus()
	if err != nil {
		return out, err
	}
	out.JournalStatus = rec.Status
	if !validRetirementReviewSHA(review) || rec.Plan.ReviewSHA256 != review ||
		nativePrepareDigest(rec.Plan) != review || rec.Plan.Target.Project != project {
		return out, errors.New("native journal identity mismatch")
	}
	if err = nativePrepareReadable(rec.Plan.Target); err != nil {
		return out, err
	}
	host, err := migrationHostDigest()
	if err != nil || host != rec.Plan.HostSHA256 {
		return out, errors.New("native journal host mismatch")
	}
	c, err := prepareClient(ctx)
	if err != nil {
		return out, err
	}
	own := c.UseProject(project)
	read := func() (string, error) {
		p, _, e := own.GetProject(project)
		if e != nil {
			return "", e
		}
		f, _, e := own.GetProfile("default")
		if e != nil {
			return "", e
		}
		names, e := c.GetNetworkACLNames()
		if e != nil {
			return "", e
		}
		var acl *api.NetworkACL
		if slices.Contains(names, rec.Plan.Target.ACL) {
			acl, _, e = c.GetNetworkACL(rec.Plan.Target.ACL)
			if e != nil {
				return "", e
			}
		}
		return nativeRetirementSnapshot(rec, p.Writable(), f.Writable(), acl)
	}
	first, err := read()
	if err != nil {
		if err.Error() == "NATIVE_CONFIG_DRIFT" {
			out.Reason = "NATIVE_CONFIG_DRIFT"
		}
		return out, err
	}
	second, err := read()
	again, journalErr := NativePrepareStatus()
	after, statErr := os.Lstat(nativePrepareJournalPath)
	if err != nil || first != second || journalErr != nil || !reflect.DeepEqual(rec, again) ||
		statErr != nil || !os.SameFile(st, after) || ctx.Err() != nil {
		return out, errors.New("native retirement observation changed; review again")
	}
	out.Ready, out.Reason, out.ObservationSHA256 = true, "OBSERVED", first
	return out, nil
}
