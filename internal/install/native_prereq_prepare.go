package install

// Historical preparation journal recognition and reviewed rollback only.
// Fresh installations must use the Scoped installer; no Legacy creation.
import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/lxc/incus/v7/shared/api"
)

const nativePrepareJournalPath = "/var/lib/boxweave-native-prereq-v1.json"

type NativePreparePlan struct {
	Target            NativePrepareTarget `json:"target"`
	HostSHA256        string              `json:"host_sha256"`
	NativeStateSHA256 string              `json:"native_state_sha256"`
	ReviewSHA256      string              `json:"review_sha256"`
	Actions           []string            `json:"actions"`
	Warnings          []string            `json:"warnings,omitempty"`
}
type NativePrepareRecord struct {
	Schema        int                 `json:"schema"`
	Status        string              `json:"status"`
	Plan          NativePreparePlan   `json:"plan"`
	BeforeProject api.ProjectPut      `json:"before_project"`
	BeforeProfile api.ProfilePut      `json:"before_profile"`
	AfterProject  api.ProjectPut      `json:"after_project"`
	AfterProfile  api.ProfilePut      `json:"after_profile"`
	ACL           api.NetworkACLsPost `json:"acl"`
}

func nativePrepareDigest(plan NativePreparePlan) string {
	plan.ReviewSHA256 = ""
	b, _ := json.Marshal(plan)
	h := sha256.Sum256(append([]byte("bw-native-prepare-1\n"), b...))
	return hex.EncodeToString(h[:])
}
func nativePrepareReadable(t NativePrepareTarget) error {
	if os.Geteuid() != 0 {
		return errors.New("native preparation requires host-local root")
	}
	u, err := user.Lookup("boxweave")
	if err != nil || t.Project != "user-"+u.Uid || !safePrereqName.MatchString(t.Pool) ||
		!safePrereqName.MatchString(t.Network) || !safePrereqName.MatchString(t.ACL) ||
		strings.HasPrefix(t.ACL, "incus") || !strings.HasPrefix(t.ACL, "bw-") {
		return errors.New("exact owned Project, safe pool/network and bw-prefixed exclusive ACL required")
	}
	if err := nativeServiceActorReady(); err != nil {
		return fmt.Errorf("dedicated confined identity not ready: %w", err)
	}
	return nil
}
// NativePreparePlanReview refuses the retired shared ACL preparation path.
func NativePreparePlanReview(_ context.Context, _ NativePrepareTarget) (NativePreparePlan, error) {
	return NativePreparePlan{}, errors.New("legacy shared ACL preparation unsupported; scoped fresh only")
}
func nativePrepareSave(rec NativePrepareRecord) error {
	if err := os.MkdirAll(filepath.Dir(nativePrepareJournalPath), 0700); err != nil {
		return err
	}
	if st, err := os.Lstat(nativePrepareJournalPath); err == nil {
		if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
			return errors.New("unsafe native preparation journal")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := atomicWrite(nativePrepareJournalPath, b, 0600); err != nil {
		return err
	}
	d, err := os.Open(filepath.Dir(nativePrepareJournalPath))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
func NativePrepareStatus() (NativePrepareRecord, error) {
	var rec NativePrepareRecord
	if os.Geteuid() != 0 {
		return rec, errors.New("host-local root required")
	}
	st, err := os.Lstat(nativePrepareJournalPath)
	if err != nil {
		return rec, err
	}
	if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return rec, errors.New("unsafe native preparation journal")
	}
	b, err := os.ReadFile(nativePrepareJournalPath)
	if err != nil {
		return rec, err
	}
	if err = json.Unmarshal(b, &rec); err != nil {
		return rec, err
	}
	if rec.Schema != 1 || rec.Plan.ReviewSHA256 == "" {
		return rec, errors.New("unknown native preparation journal schema")
	}
	return rec, nil
}
// NativePrepareApply refuses before inspecting or modifying any host state.
func NativePrepareApply(_ context.Context, _ NativePrepareTarget, _ string, _ bool) (NativePrepareRecord, error) {
	return NativePrepareRecord{}, errors.New("legacy shared ACL preparation unsupported; scoped fresh only")
}
func nativePrepareACLIsOwned(acl *api.NetworkACL, rec NativePrepareRecord) bool {
	if acl == nil || !reflect.DeepEqual(acl.Writable(), rec.ACL.NetworkACLPut) {
		return false
	}
	// Only the owned Project default profile can refer to the ACL; on
	// rollback we refuse any attached instance, foreign Project or bridge.
	return len(acl.UsedBy) == 0 || aclExclusiveUse(acl.UsedBy, rec.Plan.Target.Project)
}
func NativePrepareRollback(ctx context.Context, approvedSHA string, approved bool) (NativePrepareRecord, error) {
	rec, err := NativePrepareStatus()
	if err != nil {
		return rec, err
	}
	if !approved || approvedSHA != rec.Plan.ReviewSHA256 {
		return rec, errors.New("exact journal approval required for rollback")
	}
	if rec.Status == "ROLLED_BACK" {
		return rec, nil
	}
	// A running installed controller may dispatch new native work between
	// the empty-instance read and a profile change. Require an explicit
	// operator-controlled service stop before modifying prerequisites.
	if exec.Command("systemctl", "is-active", "--quiet", "boxweave.service").Run() == nil {
		return rec, errors.New("stop the owned BoxWeave service before native prerequisite rollback")
	}
	if err = nativePrepareReadable(rec.Plan.Target); err != nil {
		return rec, err
	}
	host, err := migrationHostDigest()
	if err != nil || host != rec.Plan.HostSHA256 {
		return rec, errors.New("original host digest changed; rollback denied")
	}
	c, err := prepareClient(ctx)
	if err != nil {
		return rec, err
	}
	t := rec.Plan.Target
	own := c.UseProject(t.Project)
	instances, err := own.GetInstanceNames(api.InstanceTypeAny)
	if err != nil || len(instances) != 0 {
		return rec, errors.New("owned instances present or inventory unknown: rollback forbidden")
	}
	prof, profTag, err := own.GetProfile("default")
	if err != nil {
		return rec, err
	}
	proj, projTag, err := own.GetProject(t.Project)
	if err != nil {
		return rec, err
	}
	curP, curJ := prof.Writable(), proj.Writable()
	// Read-back of BOTH values before any rollback effect; unknown third
	// party edits are not overwritten, even if they appear harmless.
	if !reflect.DeepEqual(curP, rec.BeforeProfile) && !reflect.DeepEqual(curP, rec.AfterProfile) {
		return rec, errors.New("profile differs from both reviewed versions; rollback refused")
	}
	if !reflect.DeepEqual(curJ, rec.BeforeProject) && !reflect.DeepEqual(curJ, rec.AfterProject) {
		return rec, errors.New("Project differs from both reviewed versions; rollback refused")
	}
	acl, _, err := c.GetNetworkACL(t.ACL)
	haveACL := err == nil
	if haveACL && !nativePrepareACLIsOwned(acl, rec) {
		return rec, errors.New("ACL is foreign or mutated: rollback refused")
	}
	rec.Status = "ROLLBACK_IN_FLIGHT"
	if err = nativePrepareSave(rec); err != nil {
		return rec, err
	}
	if reflect.DeepEqual(curP, rec.AfterProfile) {
		if err = own.UpdateProfile("default", rec.BeforeProfile, profTag); err != nil {
			return rec, err
		}
	}
	if reflect.DeepEqual(curJ, rec.AfterProject) {
		if err = own.UpdateProject(t.Project, rec.BeforeProject, projTag); err != nil {
			return rec, err
		}
	}
	if haveACL {
		acl, _, err = c.GetNetworkACL(t.ACL)
		if err != nil {
			return rec, err
		}
		if len(acl.UsedBy) > 0 {
			return rec, errors.New("ACL remains used: no blind delete")
		}
		if err = c.DeleteNetworkACL(t.ACL); err != nil {
			return rec, err
		}
	}
	// Verify no remaining BoxWeave-owned ACL and exact original Project/Profile.
	afterP, _, err := own.GetProfile("default")
	if err != nil || !reflect.DeepEqual(afterP.Writable(), rec.BeforeProfile) {
		return rec, errors.New("rollback profile readback differs")
	}
	afterJ, _, err := own.GetProject(t.Project)
	if err != nil || !reflect.DeepEqual(afterJ.Writable(), rec.BeforeProject) {
		return rec, errors.New("rollback Project readback differs")
	}
	names, err := c.GetNetworkACLNames()
	if err != nil || slices.Contains(names, t.ACL) {
		return rec, errors.New("rollback ACL remains or inventory unavailable")
	}
	rec.Status = "ROLLED_BACK"
	if err = nativePrepareSave(rec); err != nil {
		return rec, err
	}
	return rec, nil
}

// NativePrepareArchive retains a reviewed, COMPLETED rollback journal before
// a subsequent Scoped fresh installation. It does not remove or adopt any
// native object. A failed/UNKNOWN in-flight journal cannot be archived.
// The caller holds the same host-local maintenance mutex as apply/rollback.
func NativePrepareArchive(ctx context.Context, expectedSHA string, approved bool) (string, error) {
	rec, err := NativePrepareStatus()
	if err != nil {
		return "", err
	}
	if !approved || expectedSHA != rec.Plan.ReviewSHA256 || len(expectedSHA) != 64 ||
		rec.Status != "ROLLED_BACK" {
		return "", errors.New("only an exact approved fully rolled-back journal can be archived")
	}
	if exec.Command("systemctl", "is-active", "--quiet", "boxweave.service").Run() == nil {
		return "", errors.New("stop the owned service before archiving native rollback")
	}
	if err := nativePrepareReadable(rec.Plan.Target); err != nil {
		return "", err
	}
	host, err := migrationHostDigest()
	if err != nil || host != rec.Plan.HostSHA256 {
		return "", errors.New("archived rollback must belong to exact current host")
	}
	c, err := prepareClient(ctx)
	if err != nil {
		return "", err
	}
	own := c.UseProject(rec.Plan.Target.Project)
	guests, err := own.GetInstanceNames(api.InstanceTypeAny)
	if err != nil || len(guests) != 0 {
		return "", errors.New("owned instances present or inventory unknown")
	}
	pr, _, err := own.GetProject(rec.Plan.Target.Project)
	if err != nil || !reflect.DeepEqual(pr.Writable(), rec.BeforeProject) {
		return "", errors.New("original Project was not fully restored")
	}
	pf, _, err := own.GetProfile("default")
	if err != nil || !reflect.DeepEqual(pf.Writable(), rec.BeforeProfile) {
		return "", errors.New("original Profile was not fully restored")
	}
	names, err := c.GetNetworkACLNames()
	if err != nil || slices.Contains(names, rec.Plan.Target.ACL) {
		return "", errors.New("owned ACL still exists or inventory failed")
	}
	// An already-archived path is never overwritten; retain the exact prior
	// plan and its verified inverse as an immutable 0600 root-only artifact.
	for i := 1; i <= 99; i++ {
		archive := fmt.Sprintf("%s.closed.%s.%02d", nativePrepareJournalPath, expectedSHA[:16], i)
		if err = os.Link(nativePrepareJournalPath, archive); errors.Is(err, os.ErrExist) {
			continue
		} else if err != nil {
			return "", err
		}
		d, err := os.Open(filepath.Dir(nativePrepareJournalPath))
		if err != nil {
			return "", err
		}
		if err = d.Sync(); err != nil {
			_ = d.Close()
			return "", err
		}
		if err = d.Close(); err != nil {
			return "", err
		}
		if err = os.Remove(nativePrepareJournalPath); err != nil {
			return "", fmt.Errorf("immutable archive retained but journal source could not be released: %w", err)
		}
		d, err = os.Open(filepath.Dir(nativePrepareJournalPath))
		if err != nil {
			return "", err
		}
		defer d.Close()
		if err = d.Sync(); err != nil {
			return "", err
		}
		return archive, nil
	}
	return "", errors.New("native rollback archive numbering exhausted; no journal removed")
}
