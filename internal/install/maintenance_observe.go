package install

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/networklimits"
	"github.com/vanishs/boxweave/internal/state"
	"golang.org/x/sys/unix"
)

// MaintenanceEvidence records a bounded observation, not mutation authority.
// A present artifact is NOT necessarily owned or eligible for deletion.
type MaintenanceEvidence struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}
type MaintenanceArtifact struct {
	ID       string              `json:"id"`
	Evidence MaintenanceEvidence `json:"evidence"`
}
type MaintenanceUnit struct {
	Name          string              `json:"name"`
	State         string              `json:"state"`
	EffectiveFile MaintenanceEvidence `json:"effective_file"`
	DropIns       string              `json:"drop_ins"`
}
type MaintenanceDirectory struct {
	ID       string              `json:"id"`
	Evidence MaintenanceEvidence `json:"evidence"`
	Entries  int                 `json:"entries"`
}
type MaintenanceInstallation struct {
	InstalledVersion string `json:"installed_version,omitempty"`
	InstalledCommit  string `json:"installed_commit,omitempty"`
	InstalledSHA256  string `json:"installed_sha256,omitempty"`
	ManifestSHA256   string `json:"manifest_sha256,omitempty"`
	UnitSHA256       string `json:"unit_sha256,omitempty"`

	RootPolicy DiagnosticRootPolicy `json:"root_policy"`

	Directories []MaintenanceDirectory `json:"directories"`

	Identity                    DiagnosticInstallation `json:"identity"`
	Adapter                     string                 `json:"adapter"`
	ArtifactEvidence            []MaintenanceArtifact  `json:"artifacts"`
	Units                       []MaintenanceUnit      `json:"units"`
	Preparation                 MaintenanceEvidence    `json:"preparation"`
	Preimage                    MaintenanceEvidence    `json:"preimage"`
	Mode                        string                 `json:"mode"`
	LegacyReview                string                 `json:"legacy_review,omitempty"`
	bridge, acl                 string
	beforeProject, afterProject api.ProjectPut
	beforeProfile, afterProfile api.ProfilePut
	bridgePut                   *api.NetworkPut
	aclPut                      *api.NetworkACLPut
}

func me(s, r string) MaintenanceEvidence { return MaintenanceEvidence{State: s, Reason: r} }
func fixedMaintenancePlan(p Plan, db string) bool {
	return p.User == "boxweave" && p.StateDir == "/var/lib/boxweave" && p.BinaryPath == "/usr/local/bin/boxweave" && p.UnitPath == "/etc/systemd/system/boxweave.service" && p.Socket == "/var/lib/incus/unix.socket.user" && db == "/var/lib/boxweave/ledger.sqlite"
}

// Inspect only the fixed known entries, never enumerate by prefix to infer
// ownership. NO_XDEV at the leaf detects an independently mounted artifact;
// parent directories may legitimately reside on another local filesystem.
func maintenanceArtifact(path string, uid uint32, kind uint32, perm uint32) MaintenanceEvidence {
	parent, e := unix.Openat2(unix.AT_FDCWD, filepath.Dir(path), &unix.OpenHow{Flags: unix.O_PATH | unix.O_DIRECTORY | unix.O_CLOEXEC, Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS})
	if errors.Is(e, unix.ENOENT) {
		return me("ABSENT", "")
	}
	if e != nil {
		return me("UNVERIFIED", "PARENT_UNSAFE_OR_INACCESSIBLE")
	}
	defer unix.Close(parent)
	fd, e := unix.Openat2(parent, filepath.Base(path), &unix.OpenHow{Flags: unix.O_PATH | unix.O_CLOEXEC | unix.O_NOFOLLOW, Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS | unix.RESOLVE_NO_XDEV})
	if errors.Is(e, unix.ENOENT) {
		return me("ABSENT", "")
	}
	if e != nil {
		return me("UNVERIFIED", "UNSAFE_PATH_OR_MOUNT")
	}
	defer unix.Close(fd)
	var st, after unix.Stat_t
	if unix.Fstat(fd, &st) != nil || unix.Fstatat(parent, filepath.Base(path), &after, unix.AT_SYMLINK_NOFOLLOW) != nil || st != after {
		return me("UNVERIFIED", "ARTIFACT_CHANGED")
	}
	if st.Mode&unix.S_IFMT != kind || st.Uid != uid || kind == unix.S_IFREG && st.Nlink != 1 || perm != 0 && st.Mode&0777 != perm {
		return me("UNSAFE", "ARTIFACT_METADATA_MISMATCH")
	}
	return me("PRESENT", "METADATA_ONLY_NOT_OWNERSHIP")
}

// Directory enumeration provides bounded metadata hints only. Unknown entries
// remain evidence, never an owned file list or an archive provenance proof.
func maintenanceDirectory(id, path string, filter *regexp.Regexp) MaintenanceDirectory {
	o := MaintenanceDirectory{ID: id, Evidence: me("UNVERIFIED", "DIRECTORY_EVIDENCE_UNAVAILABLE")}
	fd, e := unix.Openat2(unix.AT_FDCWD, path, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW, Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS})
	if errors.Is(e, unix.ENOENT) {
		o.Evidence = me("ABSENT", "")
		return o
	}
	if e != nil {
		return o
	}
	f := os.NewFile(uintptr(fd), "maintenance-directory")
	defer f.Close()
	before, e := f.Stat()
	if e != nil {
		return o
	}
	st, ok := before.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 0 || !before.IsDir() || filter == nil && before.Mode().Perm() != 0700 || before.Mode().Perm()&0022 != 0 {
		return o
	}
	list, e := f.ReadDir(32769)
	if e != nil && !errors.Is(e, io.EOF) || len(list) > 32768 {
		return o
	}
	names := []string{}
	for _, v := range list {
		if filter == nil || filter.MatchString(v.Name()) {
			names = append(names, v.Name()+":"+v.Type().String())
		}
	}
	after, e := f.Stat()
	current, ce := os.Lstat(path)
	if e != nil || ce != nil || !os.SameFile(before, after) || !os.SameFile(before, current) || !before.ModTime().Equal(after.ModTime()) || !before.ModTime().Equal(current.ModTime()) {
		return o
	}
	slices.Sort(names)
	raw, _ := json.Marshal(names)
	o.Entries = len(names)
	o.Evidence = MaintenanceEvidence{State: "OBSERVED", Reason: "ENTRY_NAMES_AND_TYPES_ONLY_NOT_OWNERSHIP", SHA256: diagnosticHash(raw)}
	return o
}

const knownLXC255PolicyPath = "/run/systemd/system/service.d/zzz-lxc-service.conf"
const knownLXC255PolicySHA = "fcbd512cb8212c6f5720f28398f01e85ff96246aebb73e09cf449d2f3aa2c5b0"

// This is an observation hint, not authorization. Never classify another
// path or additional override as the reviewed candidate.
func knownLXC255DropIn(paths string) bool {
	if paths != knownLXC255PolicyPath {
		return false
	}
	raw, st, e := diagnosticReadFile(paths, 0, false, 4096)
	return e == nil && st.Mode().Perm() == 0644 && diagnosticHash(raw) == knownLXC255PolicySHA
}

func maintenanceUnit(ctx context.Context, name, path string) MaintenanceUnit {
	p := diagnosticSystemd(ctx, name, "LoadState", "ActiveState", "FragmentPath", "DropInPaths")
	o := MaintenanceUnit{Name: name, State: diagnosticUnitState(p), EffectiveFile: me("UNVERIFIED", "UNIT_NOT_OBSERVED"), DropIns: "UNVERIFIED"}
	if p == nil {
		return o
	}
	if p["LoadState"] == "not-found" {
		o.EffectiveFile = me("ABSENT", "")
	} else if p["FragmentPath"] == path && p["LoadState"] == "loaded" {
		o.EffectiveFile = me("OBSERVED", "EXACT_FRAGMENT_PATH_ONLY")
	}
	if v, ok := p["DropInPaths"]; ok {
		if v == "" {
			o.DropIns = "ABSENT"
		} else {
			o.DropIns = "PRESENT_UNREVIEWED"
			// Hint only. The Python exact retirement coordinator
			// independently checks the whole effective unit and policy.
			if o.EffectiveFile.State == "OBSERVED" && (name == "boxweave.service" || name == scopedGuardUnitName || name == scopedLifecycleUnitName) && knownLXC255DropIn(v) {
				o.DropIns = "KNOWN_LXC255_SCOPED_RETIREMENT_REVIEW_REQUIRED"
			}
		}
	}
	return o
}

// Only exact published RC13/RC14 binaries or the identical executing
// candidate are supported by this schema-6 offline observation adapter.
// Manifest/file agreement alone is not release provenance.
func maintenanceAdapter(m Manifest, selfSHA, version, commit string) string {
	switch {
	case m.Version == "1.0.0-rc.13" && m.BuildCommit == "6c995687ae38b5cd0e23c464dee84f40a33a01ef" && m.BinarySHA256 == "38307cc8a6b9e39830e489873c31b1e0b99a67e4a3d8f06ee3be58038d6da68e":
		return "PINNED_RC13"
	case m.Version == "1.0.0-rc.14" && m.BuildCommit == "3fb71a6ddf47d13d165e5c039144aa66fad0eb05" && m.BinarySHA256 == "1af57c1f46ed85a822f1f0a1ec2327d5e75746bb57c04ffa9c95b1d81b2389d4":
		return "PINNED_RC14"
	case validRetirementReviewSHA(selfSHA) && m.BinarySHA256 == selfSHA && m.Version == version && m.BuildCommit == commit:
		return "EXACT_RUNNING_OBSERVER_BINARY"
	}
	return "UNSUPPORTED"
}
func ObserveMaintenanceInstallation(ctx context.Context, p Plan, db, selfSHA, commit string) MaintenanceInstallation {
	o := MaintenanceInstallation{Adapter: "UNSUPPORTED", Mode: "UNVERIFIED", Preparation: me("UNVERIFIED", "INSTALLED_IDENTITY_REQUIRED"), Preimage: me("UNVERIFIED", "INSTALLED_IDENTITY_REQUIRED"), ArtifactEvidence: []MaintenanceArtifact{}, Units: []MaintenanceUnit{}}
	if os.Geteuid() != 0 || !fixedMaintenancePlan(p, db) {
		o.Preparation = me("UNVERIFIED", "ROOT_AND_FIXED_SCOPE_REQUIRED")
		return o
	}
	uid, _, e := ServiceIdentity(p)
	if e != nil || p.Project != "user-"+strconv.Itoa(uid) {
		o.Preparation = me("UNVERIFIED", "SERVICE_IDENTITY_UNVERIFIED")
		return o
	}
	o.Identity = ObserveDiagnosticInstallation(ctx, p, db)
	for _, a := range []struct {
		id, path        string
		uid, kind, perm uint32
	}{
		{"controller_binary", p.BinaryPath, 0, unix.S_IFREG, 0755}, {"controller_unit", p.UnitPath, 0, unix.S_IFREG, 0644},
		{"state_directory", p.StateDir, uint32(uid), unix.S_IFDIR, 0700}, {"manifest", manifestPath(p), uint32(uid), unix.S_IFREG, 0600}, {"ledger", db, uint32(uid), unix.S_IFREG, 0600},
		{"legacy_preparation", nativePrepareJournalPath, 0, unix.S_IFREG, 0600}, {"legacy_root_unit", nativePeerRootUnitPath, 0, unix.S_IFREG, 0644}, {"legacy_retirement_unit", nativeRetirementUnitPath, 0, unix.S_IFREG, 0644},
		{"legacy_root_socket", nativePeerRootSocketPath, 0, unix.S_IFSOCK, 0}, {"legacy_retirement_socket", nativeRetirementSocketPath, 0, unix.S_IFSOCK, 0},
		{"legacy_root_operations", nativePeerConntrackOperationsDir, 0, unix.S_IFDIR, 0700}, {"legacy_root_lock", nativePeerConntrackLockPath, 0, unix.S_IFREG, 0600}, {"legacy_root_journal", nativePeerConntrackJournalPath, 0, unix.S_IFREG, 0600},
		{"scoped_root_journal", nativePeerFreshJournalDir, 0, unix.S_IFDIR, 0700}, {"scoped_guard_unit", scopedGuardUnitPath, 0, unix.S_IFREG, 0644}, {"scoped_lifecycle_unit", scopedLifecycleUnitPath, 0, unix.S_IFREG, 0644}, {"scoped_incus_dropin", scopedIncusDropIn, 0, unix.S_IFREG, 0644}, {"scoped_lifecycle_socket", scopedLifecycleSocket, 0, unix.S_IFSOCK, 0},
		{"retirement_fence", scopedRetirementFence, 0, unix.S_IFREG, 0600}, {"purge_fence", retirementPurgeFence, 0, unix.S_IFREG, 0600}, {"retirement_archive_root", "/var/lib/boxweave-retired", 0, unix.S_IFDIR, 0700}, {"retirement_audit", "/var/lib/boxweave-uninstall-v1-audit.json", 0, unix.S_IFREG, 0600}, {"installer_backup", "/var/lib/boxweave-installer-backup", 0, unix.S_IFDIR, 0700},
	} {
		ev := maintenanceArtifact(a.path, a.uid, a.kind, a.perm)
		if ev.State == "PRESENT" && a.kind == unix.S_IFREG && a.uid == 0 && a.id != "controller_binary" {
			raw, _, err := diagnosticReadFile(a.path, 0, a.perm == 0600, 1<<20)
			if err != nil {
				ev = me("UNVERIFIED", "ARTIFACT_CONTENT_UNVERIFIED")
			} else {
				ev.SHA256 = diagnosticHash(raw)
			}
		}
		o.ArtifactEvidence = append(o.ArtifactEvidence, MaintenanceArtifact{a.id, ev})
	}
	o.Directories = []MaintenanceDirectory{
		maintenanceDirectory("legacy_root_operations", nativePeerConntrackOperationsDir, nil),
		maintenanceDirectory("scoped_root_journal", nativePeerFreshJournalDir, nil),
		maintenanceDirectory("retirement_archive_root", "/var/lib/boxweave-retired", nil),
		maintenanceDirectory("legacy_closed_journals", "/var/lib", regexp.MustCompile(`^boxweave-native-prereq-v1[.]json[.]closed[.][0-9a-f]{16}[.][0-9]{2}$`)),
	}
	for _, u := range []struct{ name, path string }{
		{"boxweave.service", p.UnitPath}, {"boxweave-native-peer-root.service", nativePeerRootUnitPath}, {nativeRetirementUnitName, nativeRetirementUnitPath}, {scopedGuardUnitName, scopedGuardUnitPath}, {scopedLifecycleUnitName, scopedLifecycleUnitPath},
	} {
		o.Units = append(o.Units, maintenanceUnit(ctx, u.name, u.path))
	}
	for _, a := range o.ArtifactEvidence[:4] {
		if a.Evidence.State != "PRESENT" {
			return o
		}
	}
	if o.Identity.InstalledIdentity != "VERIFIED" {
		return o
	}
	raw, _, e := diagnosticReadFile(manifestPath(p), uint32(uid), true, 65536)
	var m Manifest
	if e != nil || json.Unmarshal(raw, &m) != nil || m.Project != p.Project || m.Schema != manifestSchema || m.ServiceUser != p.User || m.Socket != p.Socket || m.BinaryPath != p.BinaryPath || m.StateDir != p.StateDir {
		return o
	}
	// Re-read hashes: the manifest may have changed since the first observation.
	bin, _, e := diagnosticReadFile(p.BinaryPath, 0, false, 128<<20)
	if e != nil || diagnosticHash(bin) != m.BinarySHA256 {
		return o
	}
	unit, _, e := diagnosticReadFile(p.UnitPath, 0, false, 65536)
	if e != nil || diagnosticHash(unit) != m.UnitSHA256 {
		return o
	}
	o.Adapter = maintenanceAdapter(m, selfSHA, p.Version, commit)
	if o.Adapter != "UNSUPPORTED" {
		o.InstalledVersion, o.InstalledCommit, o.InstalledSHA256 = m.Version, m.BuildCommit, m.BinarySHA256
		o.ManifestSHA256, o.UnitSHA256 = diagnosticHash(raw), m.UnitSHA256
	}
	if o.Adapter == "UNSUPPORTED" {
		return o
	}
	if m.ScopedPeerMode == "" {
		o.Mode = "LEGACY"
		o.RootPolicy = DiagnosticRootPolicy{State: "NOT_APPLICABLE", Kernel: "NOT_CHECKED", Conntrack: "NOT_CHECKED"}
		raw, _, e := diagnosticReadFile(nativePrepareJournalPath, 0, true, 1<<20)
		if errors.Is(e, os.ErrNotExist) {
			o.Preparation = me("ABSENT", "")
			o.Preimage = me("ABSENT", "")
			return o
		}
		var rec NativePrepareRecord
		host, he := migrationHostDigest()
		if e != nil || he != nil || json.Unmarshal(raw, &rec) != nil || rec.Schema != 1 || rec.Plan.HostSHA256 != host || rec.Plan.Target.Project != p.Project || rec.Plan.Target.Network != "incusbr-"+strconv.Itoa(uid) || !safePrereqName.MatchString(rec.Plan.Target.ACL) || !validRetirementReviewSHA(rec.Plan.ReviewSHA256) || nativePrepareDigest(rec.Plan) != rec.Plan.ReviewSHA256 || m.RetirementNativeReviewSHA != rec.Plan.ReviewSHA256 || m.RetirementBridge != rec.Plan.Target.Network || m.RetirementACL != rec.Plan.Target.ACL {
			o.Preparation = me("UNVERIFIED", "LEGACY_JOURNAL_IDENTITY_UNVERIFIED")
			return o
		}
		if rec.BeforeProject.Config["restricted"] != "true" || rec.BeforeProject.Config["features.networks"] != "false" || rec.BeforeProject.Config["restricted.networks.access"] != rec.Plan.Target.Network ||
			!safePrereqName.MatchString(rec.Plan.Target.Pool) || nativePrepareProfileEligible(rec.BeforeProfile.Devices, rec.Plan.Target) != nil ||
			!reflect.DeepEqual(rec.AfterProject, nativePrepareProject(rec.BeforeProject)) || !reflect.DeepEqual(rec.AfterProfile, nativePrepareProfile(rec.Plan.Target, rec.BeforeProfile)) || rec.ACL.Name != rec.Plan.Target.ACL {
			o.Preparation = me("UNVERIFIED", "LEGACY_PREIMAGE_SHAPE_UNVERIFIED")
			return o
		}
		if rec.Status != "APPLIED" && rec.Status != "ROLLED_BACK" {
			o.Preparation = me("UNVERIFIED", "LEGACY_JOURNAL_NOT_TERMINAL")
			return o
		}
		o.Preparation = MaintenanceEvidence{State: "OBSERVED", Reason: rec.Status, SHA256: diagnosticHash(raw)}
		o.Preimage = me("OBSERVED", "RECORDED_LEGACY_PREIMAGE_NOT_ROLLBACK_AUTHORITY")
		o.LegacyReview = rec.Plan.ReviewSHA256
		o.bridge, o.acl = rec.Plan.Target.Network, rec.Plan.Target.ACL
		o.beforeProject, o.afterProject = rec.BeforeProject, rec.AfterProject
		o.beforeProfile, o.afterProfile = rec.BeforeProfile, rec.AfterProfile
		acl := rec.ACL.NetworkACLPut
		o.aclPut = &acl
	} else if m.ScopedPeerMode == nativePeerFreshMode {
		o.Mode = "SCOPED_V2"
		r, _, e := nativePeerFreshVerifiedPlan()
		if e != nil || r.Project != p.Project || r.Bridge != "incusbr-"+strconv.Itoa(uid) || nativePeerFreshQuarantineReceipt(r) != nil {
			o.Preparation = me("UNVERIFIED", "SCOPED_ROOT_JOURNAL_UNVERIFIED")
			return o
		}
		o.Preparation = me("OBSERVED", "SCOPED_FRESH_AND_QUARANTINE_RECEIPTS_VERIFIED")
		o.RootPolicy = observeScopedDiagnosticRootPolicy(ctx, p.Project, DiagnosticRootPolicy{State: "UNVERIFIED", Kernel: "NOT_CHECKED", Conntrack: diagnosticExecutableState("/usr/sbin/conntrack", 0), HistoryLimit: networklimits.ScopedPeerHistoryLimit})

		pre, e := scopedRetirementPreimageRead(r)
		if e != nil {
			o.Preimage = me("UNVERIFIED", "SCOPED_PREIMAGE_UNVERIFIED")
			return o
		}
		b, _ := json.Marshal(pre)
		o.Preimage = MaintenanceEvidence{State: "OBSERVED", Reason: "RECORDED_SCOPED_PREIMAGE", SHA256: diagnosticHash(b)}
		o.bridge, o.acl = r.Bridge, pre.Quarantine.Name
		o.beforeProject, o.afterProject = pre.BeforeProject, pre.AfterProject
		o.beforeProfile, o.afterProfile = pre.BeforeProfile, pre.AfterProfile
		o.bridgePut = &pre.Bridge
		acl := pre.Quarantine.NetworkACLPut
		o.aclPut = &acl
	}
	return o
}

type MaintenanceNative struct {
	LegacyLeaseProof    string            `json:"legacy_lease_proof,omitempty"`
	OwnedPower          map[string]string `json:"owned_power,omitempty"`
	NativeCount         int               `json:"native_count"`
	OwnershipComparison string            `json:"ownership_comparison"`

	Evidence          MaintenanceEvidence `json:"evidence"`
	OwnedMatched      []string            `json:"owned_matched"`
	OwnedUnverified   []string            `json:"owned_unverified"`
	Unowned           int                 `json:"unowned_or_mismatched"`
	ActiveOperations  int                 `json:"active_operations"`
	OtherProjects     int                 `json:"other_projects"`
	ProjectConfig     string              `json:"project_config"`
	ProfileConfig     string              `json:"profile_config"`
	BridgeConfig      string              `json:"bridge_config"`
	BridgeConsumers   string              `json:"bridge_consumers"`
	BaseACLConfig     string              `json:"base_acl_config"`
	BaseACLConsumers  string              `json:"base_acl_consumers"`
	BoxACLReferences  string              `json:"box_acl_references"`
	ForwardReferences string              `json:"forward_references"`
	EpochNS           int64               `json:"epoch_ns,omitempty"`
}

func maintenanceConfig(actual, before, after any) string {
	if reflect.DeepEqual(actual, before) {
		return "BEFORE"
	}
	if reflect.DeepEqual(actual, after) {
		return "AFTER"
	}
	return "DRIFT"
}

// Native identity observation is independent of controller runtime and
// network preflight. Do not connect if Incus is inactive (socket activation).
// Only the exact dedicated Project is queried, plus its pinned bridge/ACLs.
func ObserveMaintenanceNative(ctx context.Context, p Plan, i MaintenanceInstallation, ledger *state.MaintenanceLedger) MaintenanceNative {
	zero := MaintenanceNative{OwnershipComparison: "UNVERIFIED", Evidence: me("UNVERIFIED", "NATIVE_SCOPE_OR_SOURCE_UNVERIFIED")}
	if os.Geteuid() != 0 || i.Adapter == "UNSUPPORTED" || i.Mode == "UNVERIFIED" || !fixedMaintenancePlan(p, "/var/lib/boxweave/ledger.sqlite") {
		return zero
	}
	l := state.MaintenanceLedger{Project: p.Project}
	compare := ledger != nil && ledger.Project == p.Project && ledger.ForeignRows == 0
	if compare {
		l = *ledger
	}
	active := func() bool {
		return diagnosticUnitState(diagnosticSystemd(ctx, "incus.service", "LoadState", "ActiveState")) == "ACTIVE"
	}
	if !active() {
		zero.Evidence = me("UNVERIFIED", "INCUS_INACTIVE_NO_SOCKET_ACTIVATION")
		return zero
	}
	c, e := prepareClient(ctx)
	if e != nil {
		return zero
	}
	c = c.UseProject("default")
	own := c.UseProject(p.Project)
	read := func() (MaintenanceNative, string, error) {
		o := MaintenanceNative{OwnedMatched: []string{}, OwnedUnverified: []string{}, ProjectConfig: "UNVERIFIED", ProfileConfig: "UNVERIFIED", BridgeConfig: "UNVERIFIED", BridgeConsumers: "UNVERIFIED", BaseACLConfig: "UNVERIFIED", BaseACLConsumers: "UNVERIFIED", BoxACLReferences: "UNVERIFIED", ForwardReferences: "UNVERIFIED"}
		names, e := own.GetInstanceNames(api.InstanceTypeAny)
		if e != nil {
			return o, "", e
		}
		inst, e := own.GetInstances(api.InstanceTypeAny)
		if e != nil || len(inst) != len(names) || len(inst) > 1024 {
			return o, "", errors.New("native inventory unavailable")
		}
		slices.Sort(names)
		slices.SortFunc(inst, func(a, b api.Instance) int { return strings.Compare(a.Name, b.Name) })
		found := map[string]api.Instance{}
		for n, v := range inst {
			if names[n] != v.Name || v.Name == "" || v.Project != "" && v.Project != p.Project || n > 0 && names[n-1] == names[n] {
				return o, "", errors.New("native scope unavailable")
			}
			found[v.Name] = v
		}
		owners := map[string]string{}
		for _, b := range l.Network.Boxes {
			v, ok := found[b.Name]
			if !ok || v.Config["volatile.uuid"] != b.UUID || b.Incarnation != b.UUID || !b.OriginalProof || b.OriginalCount != 1 || b.OriginalResource != "box:"+b.Name || v.Config["user.boxweave.create_operation"] != b.OriginalID {
				o.OwnedUnverified = append(o.OwnedUnverified, b.UUID)
				continue
			}
			o.OwnedMatched = append(o.OwnedMatched, b.UUID)
			if o.OwnedPower == nil {
				o.OwnedPower = map[string]string{}
			}
			o.OwnedPower[b.UUID] = v.Status
			owners[b.Name] = b.UUID
		}
		o.NativeCount = len(inst)
		o.OwnershipComparison = "LEDGER_UNAVAILABLE_OR_FOREIGN"
		if compare {
			o.Unowned = len(inst) - len(owners)
			o.OwnershipComparison = "OBSERVED"
		}
		ops, e := own.GetOperations()
		if e != nil || len(ops) > 8192 {
			return o, "", errors.New("native operation inventory unavailable")
		}
		opWitness := make([]string, 0, len(ops))
		for _, v := range ops {
			if !v.StatusCode.IsFinal() {
				opWitness = append(opWitness, v.ID+":"+strconv.Itoa(int(v.StatusCode)))
			}
			if !v.StatusCode.IsFinal() {
				o.ActiveOperations++
			}
		}
		projects, e := c.GetProjectNames()
		if e != nil || len(projects) > 8192 || !slices.Contains(projects, p.Project) || !slices.Contains(projects, "default") {
			return o, "", errors.New("native project inventory unavailable")
		}
		o.OtherProjects = len(projects) - 2
		pr, pt, e := own.GetProject(p.Project)
		if e != nil || pr == nil || pt == "" {
			return o, "", errors.New("native project unavailable")
		}
		pf, ft, e := own.GetProfile("default")
		if e != nil || pf == nil || ft == "" {
			return o, "", errors.New("native profile unavailable")
		}
		if i.Preimage.State == "OBSERVED" {
			o.ProjectConfig = maintenanceConfig(pr.Writable(), i.beforeProject, i.afterProject)
			o.ProfileConfig = maintenanceConfig(pf.Writable(), i.beforeProfile, i.afterProfile)
		}
		acls := map[string]api.NetworkACL{}
		var nw *api.Network
		if i.bridge != "" {
			nw, _, e = c.GetNetwork(i.bridge)
			if e != nil || nw == nil {
				return o, "", errors.New("native bridge unavailable")
			}
			if nw.Managed && nw.Type == "bridge" && nw.Status == "Created" {
				o.BridgeConfig = "MANAGED_BRIDGE_OBSERVED"
				if i.bridgePut != nil {
					if reflect.DeepEqual(nw.Writable(), *i.bridgePut) {
						o.BridgeConfig = "RECORDED_PREIMAGE_MATCH"
					} else {
						o.BridgeConfig = "DRIFT"
					}
				}
			}
			if nativePeerBridgeConsumersOwned(nw.UsedBy, p.Project, owners) == nil {
				o.BridgeConsumers = "OWNED_REFERENCES_ONLY"
			} else {
				o.BridgeConsumers = "FOREIGN_OR_UNVERIFIED"
			}
			an, e := c.GetNetworkACLNames()
			if e != nil || len(an) > 16384 {
				return o, "", errors.New("ACL inventory unavailable")
			}
			if slices.Contains(an, i.acl) {
				a, _, e := c.GetNetworkACL(i.acl)
				if e != nil || a == nil {
					return o, "", errors.New("base ACL unavailable")
				}
				acls[i.acl] = *a
				if i.aclPut != nil && reflect.DeepEqual(a.Writable(), *i.aclPut) {
					o.BaseACLConfig = "RECORDED_PREIMAGE_MATCH"
				} else {
					o.BaseACLConfig = "DELTA_OR_DRIFT_UNVERIFIED"
				}
				if nativePeerBridgeConsumersOwned(a.UsedBy, p.Project, owners) == nil {
					o.BaseACLConsumers = "OWNED_REFERENCES_ONLY"
				} else if len(a.UsedBy) == 0 {
					o.BaseACLConsumers = "NO_REFERENCES"
				} else {
					o.BaseACLConsumers = "FOREIGN_OR_UNVERIFIED"
				}
			} else {
				o.BaseACLConfig = "ABSENT"
				o.BaseACLConsumers = "NOT_APPLICABLE"
			}
			o.BoxACLReferences = "OBSERVED"
			if i.Mode == "LEGACY" {
				o.LegacyLeaseProof = "VERIFIED"
			}
			for _, b := range l.Network.Boxes {
				if i.Mode == "LEGACY" {
					if b.FenceCount != 2 || b.FenceIPv4 != 1 || b.FenceIPv6 != 1 ||
						native.ValidatePeerACLBox(func() *api.Instance { v := found[b.Name]; return &v }(), p.Project, i.bridge, i.acl, native.PeerACLBox{Name: b.Name, UUID: b.UUID, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6}, false) != nil {
						o.LegacyLeaseProof = "UNVERIFIED"
					}
				}
				v := found[b.Name]
				if owners[b.Name] != b.UUID || b.BindingCount != 1 || b.BindingState != "PINNED" || b.Bridge != i.bridge || b.ACL == "" || v.ExpandedDevices["eth0"]["security.acls"] != b.ACL {
					o.BoxACLReferences = "UNVERIFIED"
					continue
				}
				if b.ACL == i.acl && i.Mode == "LEGACY" {
					continue
				}
				want, err := NativePeerScopedACLName(p.Project, i.bridge, b.UUID)
				if i.Mode != "SCOPED_V2" || err != nil || want != b.ACL {
					o.BoxACLReferences = "UNVERIFIED"
					continue
				}
				a, _, e := c.GetNetworkACL(b.ACL)
				if e != nil || a == nil || len(a.UsedBy) != 1 || !nativePeerGuardOwnedURL(a.UsedBy[0], "instances", b.Name, p.Project) {
					o.BoxACLReferences = "FOREIGN_OR_UNVERIFIED"
					continue
				}
				acls[b.ACL] = *a
			}
		}
		// Exact device identity and all extra proxy references; detailed packet
		// and Forward endpoint proof remain the existing diagnostic leaf.
		o.ForwardReferences = "OBSERVED"
		expected := map[string]bool{}
		for _, f := range l.Forwards {
			if f.State == "REVOKED" {
				continue
			}
			name := ""
			for _, b := range l.Boxes {
				if b.UUID == f.Box && b.Incarnation == f.Incarnation {
					name = b.Name
					break
				}
			}
			if name == "" || owners[name] != f.Box || f.State != "CONFIRMED" || !(native.ForwardSpec{Device: f.Device, Protocol: f.Protocol, HostBind: f.Bind, HostPort: f.HostPort, GuestPort: f.GuestPort, ExternalAcknowledged: true}).MatchesDevice(found[name].Devices[f.Device]) {
				o.ForwardReferences = "UNVERIFIED"
				continue
			}
			expected[name+"/"+f.Device] = true
		}
		for name, v := range found {
			for dev, cfg := range v.ExpandedDevices {
				if cfg["type"] == "proxy" && !expected[name+"/"+dev] {
					o.ForwardReferences = "EXTRA_OR_UNVERIFIED_PROXY"
				}
			}
		}
		if !compare {
			o.LegacyLeaseProof = "UNVERIFIED"
			o.BridgeConsumers = "UNVERIFIED"
			o.BaseACLConsumers = "UNVERIFIED"
			o.BoxACLReferences = "UNVERIFIED"
			o.ForwardReferences = "UNVERIFIED"
		}
		// UsedBy is a set, not an ordered configuration field. Incus can
		// return the same consumers in different orders between reads.
		slices.Sort(pr.UsedBy)
		slices.Sort(pf.UsedBy)
		if nw != nil {
			slices.Sort(nw.UsedBy)
		}
		for name, acl := range acls {
			slices.Sort(acl.UsedBy)
			acls[name] = acl
		}
		slices.Sort(projects)
		slices.Sort(opWitness)
		raw, _ := json.Marshal(struct {
			Instances []api.Instance
			Projects  []string
			Project   *api.Project
			Profile   *api.Profile
			Bridge    *api.Network
			ACLs      map[string]api.NetworkACL
			PT, FT    string
			Ops       []string
		}{inst, projects, pr, pf, nw, acls, pt, ft, opWitness})
		return o, diagnosticHash(raw), nil
	}
	a, ad, e := read()
	if e != nil {
		return zero
	}
	b, bd, e := read()
	if e != nil || ad != bd || !reflect.DeepEqual(a, b) || !active() || ctx.Err() != nil {
		zero.Evidence = me("UNVERIFIED", "NATIVE_EPOCH_CHANGED_OR_RECHECK_UNAVAILABLE")
		return zero
	}
	b.Evidence = MaintenanceEvidence{State: "OBSERVED", Reason: "BOUNDED_DOUBLE_READ_NOT_MUTATION_PROOF", SHA256: bd}
	b.EpochNS = time.Now().UnixNano()
	return b
}
