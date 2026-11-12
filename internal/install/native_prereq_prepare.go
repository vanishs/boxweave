package install

// S2.5 operator-only preparation of owned native Incus resources. The public
// MCP actor cannot invoke this code or acquire the admin socket. Every write
// is preceded by an exact reviewed host/native snapshot and a durable journal.
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

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
)

const nativePrepareJournalPath = "/var/lib/boxweave-native-prereq-v1.json"

type NativePrepareTarget struct {
	Project string `json:"project"`
	Pool    string `json:"pool"`
	Network string `json:"network"`
	ACL     string `json:"acl"`
}
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
func nativePrepareProfile(t NativePrepareTarget, original api.ProfilePut) api.ProfilePut {
	p := original
	p.Devices = api.DevicesMap{
		"root": {"type": "disk", "path": "/", "pool": t.Pool},
		"eth0": {"type": "nic", "name": "eth0", "network": t.Network, "security.acls": t.ACL,
			"security.acls.default.ingress.action": "reject",
			"security.ipv4_filtering":              "true", "security.ipv6_filtering": "true", "security.mac_filtering": "true"},
	}
	return p
}
func nativePrepareProject(original api.ProjectPut) api.ProjectPut {
	p := original
	p.Config = api.ConfigMap{}
	for k, v := range original.Config {
		p.Config[k] = v
	}
	p.Config["restricted.containers.nesting"] = "allow"
	// Incus multi-user bootstrapping can permit home-path mounts and GPUs.
	// This dedicated MCP service needs neither; tighten only its own Project.
	p.Config["restricted.devices.disk"] = "managed"
	p.Config["restricted.devices.gpu"] = "block"
	return p
}

// Incus already creates a bounded root disk and a private NIC for new
// restricted users. Only these unchanged defaults can be prepared, never
// an arbitrary host mount, a proxy, foreign bridge or a preexisting ACL.
func nativePrepareProfileEligible(devices api.DevicesMap, t NativePrepareTarget) error {
	if len(devices) > 2 {
		return errors.New("owned profile contains unreviewed devices")
	}
	for name, d := range devices {
		switch name {
		case "root":
			if d["type"] != "disk" || d["path"] != "/" || d["source"] != "" || len(d) != 3 || d["pool"] == "" {
				return errors.New("root device not the safe native default")
			}
		case "eth0":
			if d["type"] != "nic" || d["network"] != t.Network ||
				(d["name"] != "" && d["name"] != "eth0") ||
				d["security.acls"] != "" || d["security.acls.default.ingress.action"] != "" ||
				d["security.ipv4_filtering"] != "" || d["security.ipv6_filtering"] != "" ||
				d["security.mac_filtering"] != "" || len(d) > 4 {
				return errors.New("owned NIC not untouched native bridge default")
			}
		default:
			return errors.New("unexpected profile device or host mount")
		}
	}
	return nil
}
func nativePrepareACL(t NativePrepareTarget) api.NetworkACLsPost {
	return api.NetworkACLsPost{
		NetworkACLPost: api.NetworkACLPost{Name: t.ACL},
		NetworkACLPut: api.NetworkACLPut{Description: "BoxWeave v1 exclusive to " + t.Project,
			Ingress: []api.NetworkACLRule{}, Egress: []api.NetworkACLRule{}, Config: api.ConfigMap{}},
	}
}
func nativePrepareStateDigest(objects ...any) string {
	h := sha256.New()
	for _, v := range objects {
		b, _ := json.Marshal(v)
		_, _ = h.Write(b)
		_, _ = h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}
func prepareClient(ctx context.Context) (incus.InstanceServer, error) {
	return incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
}
func nativePrepareInspect(ctx context.Context, t NativePrepareTarget) (NativePreparePlan, NativePrepareRecord, error) {
	plan := NativePreparePlan{Target: t}
	rec := NativePrepareRecord{Schema: 1, Status: "PLANNED"}
	if err := nativePrepareReadable(t); err != nil {
		return plan, rec, err
	}
	host, err := migrationHostDigest()
	if err != nil {
		return plan, rec, err
	}
	plan.HostSHA256 = host
	c, err := prepareClient(ctx)
	if err != nil {
		return plan, rec, err
	}
	own := c.UseProject(t.Project)
	instances, err := own.GetInstanceNames(api.InstanceTypeAny)
	if err != nil {
		return plan, rec, err
	}
	if len(instances) != 0 {
		return plan, rec, errors.New("owned project has instances: refuse to change live profile")
	}
	project, pETag, err := own.GetProject(t.Project)
	if err != nil {
		return plan, rec, err
	}
	if project.Config["restricted"] != "true" {
		return plan, rec, errors.New("native owned project is not restricted")
	}
	nesting := project.Config["restricted.containers.nesting"]
	if nesting != "" && nesting != "block" && nesting != "allow" {
		return plan, rec, errors.New("unrecognized owned Project nesting setting")
	}
	profile, prETag, err := own.GetProfile("default")
	if err != nil {
		return plan, rec, err
	}
	if err := nativePrepareProfileEligible(profile.Devices, t); err != nil {
		return plan, rec, err
	}
	if len(profile.UsedBy) != 0 {
		return plan, rec, errors.New("owned default profile already used")
	}
	pool, poolETag, err := c.GetStoragePool(t.Pool)
	if err != nil {
		return plan, rec, err
	}
	if pool.Driver != "dir" || pool.Status != "Created" {
		return plan, rec, errors.New("selected dir pool not created")
	}
	free, separate, quota, err := native.InspectNativeDirPool(t.Pool)
	if err != nil || free < 64<<20 {
		return plan, rec, errors.New("selected dir pool backing unavailable")
	}
	if !separate && t.Pool != "default" {
		return plan, rec, errors.New("custom pool backing not independently identified")
	}
	if !separate {
		plan.Warnings = append(plan.Warnings, "DIR_POOL_HOST_ROOT_SHARED: separate capacity isolation is not established")
	}
	if !quota {
		plan.Warnings = append(plan.Warnings, "DIR_QUOTA_UNVERIFIED: official Incus dir quota precondition not observed; no hard size guarantee")
	}
	network, nETag, err := c.GetNetwork(t.Network)
	if err != nil {
		return plan, rec, err
	}
	if !network.Managed || network.Type != "bridge" || network.Status != "Created" {
		return plan, rec, errors.New("selected native network must be a managed live bridge")
	}
	aclNames, err := c.GetNetworkACLNames()
	if err != nil {
		return plan, rec, err
	}
	if slices.Contains(aclNames, t.ACL) {
		return plan, rec, errors.New("ACL name already exists: refuse adoption")
	}
	alias, aETag, err := own.GetImageAlias("boxweave-ubuntu-24.04")
	if err != nil || alias.Target == "" {
		return plan, rec, errors.New("approved local Ubuntu image alias must be operator-prepared")
	}
	img, imgETag, err := own.GetImage(alias.Target)
	if err != nil || !strings.EqualFold(img.Properties["os"], "Ubuntu") ||
		(img.Properties["release"] != "noble" && img.Properties["release"] != "24.04") {
		return plan, rec, errors.New("approved local Ubuntu 24.04 image not present")
	}
	rec.BeforeProject = project.Writable()
	rec.BeforeProfile = profile.Writable()
	rec.AfterProject = nativePrepareProject(rec.BeforeProject)
	rec.AfterProfile = nativePrepareProfile(t, rec.BeforeProfile)
	rec.ACL = nativePrepareACL(t)
	plan.NativeStateSHA256 = nativePrepareStateDigest(
		project, pETag, profile, prETag, pool, poolETag, network, nETag, alias, aETag, img, imgETag, "ACL_ABSENT",
	)
	plan.Actions = []string{
		"Create exact new dedicated ACL " + t.ACL + " (never edit shared network)",
		"Set nesting permission only on empty restricted Project " + t.Project,
		"Set root dir pool and private reject/anti-spoof NIC only in owned empty default profile",
		"Independently read back same owned Project, profile, image, pool, ACL",
	}
	plan.ReviewSHA256 = nativePrepareDigest(plan)
	rec.Plan = plan
	return plan, rec, nil
}
func NativePreparePlanReview(ctx context.Context, t NativePrepareTarget) (NativePreparePlan, error) {
	p, _, e := nativePrepareInspect(ctx, t)
	return p, e
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
func NativePrepareApply(ctx context.Context, t NativePrepareTarget, approvedSHA string, approved bool) (NativePrepareRecord, error) {
	var empty NativePrepareRecord
	if !approved || len(approvedSHA) != 64 {
		return empty, errors.New("explicit root-local reviewed native plan approval required")
	}
	if _, err := os.Lstat(nativePrepareJournalPath); err == nil {
		return empty, errors.New("existing preparation journal: inspect/reconcile or rollback before any new effect")
	} else if !os.IsNotExist(err) {
		return empty, err
	}
	plan, rec, err := nativePrepareInspect(ctx, t)
	if err != nil {
		return empty, err
	}
	if plan.ReviewSHA256 != approvedSHA {
		return empty, errors.New("host/native state or approved plan digest changed")
	}
	c, err := prepareClient(ctx)
	if err != nil {
		return empty, err
	}
	rec.Status = "APPLY_IN_FLIGHT"
	if err = nativePrepareSave(rec); err != nil {
		return empty, err
	}
	// A crash after this point leaves a durable in-flight marker; retrying
	// apply is NEVER allowed. Root-local rollback compares exact after/before.
	if err = c.CreateNetworkACL(rec.ACL); err != nil {
		return rec, fmt.Errorf("ACL creation uncertain; inspect journal, never auto-retry: %w", err)
	}
	rec.Status = "ACL_CREATED"
	if err = nativePrepareSave(rec); err != nil {
		return rec, err
	}
	own := c.UseProject(t.Project)
	proj, etag, err := own.GetProject(t.Project)
	if err != nil {
		return rec, err
	}
	if !reflect.DeepEqual(proj.Writable(), rec.BeforeProject) {
		return rec, errors.New("Project changed after plan: rollback/review")
	}
	rec.Status = "PROJECT_IN_FLIGHT"
	if err = nativePrepareSave(rec); err != nil {
		return rec, err
	}
	if err = own.UpdateProject(t.Project, rec.AfterProject, etag); err != nil {
		return rec, fmt.Errorf("Project mutation uncertain; inspect original: %w", err)
	}
	rec.Status = "PROJECT_UPDATED"
	if err = nativePrepareSave(rec); err != nil {
		return rec, err
	}
	profile, etag, err := own.GetProfile("default")
	if err != nil {
		return rec, err
	}
	if !reflect.DeepEqual(profile.Writable(), rec.BeforeProfile) {
		return rec, errors.New("Profile changed after plan: rollback/review")
	}
	rec.Status = "PROFILE_IN_FLIGHT"
	if err = nativePrepareSave(rec); err != nil {
		return rec, err
	}
	if err = own.UpdateProfile("default", rec.AfterProfile, etag); err != nil {
		return rec, fmt.Errorf("Profile mutation uncertain; inspect original: %w", err)
	}
	// Snapshot readback is not optional. If any check fails retain journal
	// so the operator cannot unknowingly re-dispatch writes.
	report, err := NativePrerequisites(ctx, t.Project)
	if err != nil || !report.ConfigReady || report.ACL != t.ACL || report.Pool != t.Pool || report.Network != t.Network {
		rec.Status = "VERIFY_BLOCKED"
		_ = nativePrepareSave(rec)
		return rec, errors.New("native preparation applied but readback blocked; exact journal requires review")
	}
	rec.Status = "APPLIED"
	if err = nativePrepareSave(rec); err != nil {
		return rec, err
	}
	return rec, nil
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
