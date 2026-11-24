package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// DiagnosticInstallation is a redacted, non-atomic observation, never a
// permission to execute an installer/Root operation.
type DiagnosticInstallation struct {
	Mode              string `json:"mode"`
	Candidate         string `json:"candidate,omitempty"`
	Coverage          string `json:"coverage"`
	Manifest          string `json:"manifest"`
	InstalledIdentity string `json:"installed_identity"`
	Runtime           string `json:"runtime"`
	RootJournal       string `json:"root_journal"`
	RootProof         string `json:"root_proof"`
	ScopedArtifacts   string `json:"scoped_artifacts"`
	StaticContract    string `json:"static_contract"`
	Dependencies      string `json:"dependencies"`
	GuardUnit         string `json:"guard_unit"`
	LifecycleUnit     string `json:"lifecycle_unit"`
	ServiceUnit       string `json:"service_unit"`
	Reason            string `json:"reason,omitempty"`
}

func diagnosticReadFile(path string, uid uint32, private bool, limit int64) ([]byte, os.FileInfo, error) {
	fd, e := unix.Openat2(unix.AT_FDCWD, path, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK, Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS})
	if e != nil {
		return nil, nil, e
	}
	f := os.NewFile(uintptr(fd), "diagnostic-install-file")
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return nil, nil, e
	}
	own, ok := st.Sys().(*syscall.Stat_t)
	if !ok || own.Uid != uid || own.Nlink != 1 || !st.Mode().IsRegular() || st.Mode().Perm()&0022 != 0 || private && st.Mode().Perm() != 0600 || st.Size() < 0 || st.Size() > limit {
		return nil, nil, errors.New("unsafe install evidence")
	}
	b, e := io.ReadAll(io.LimitReader(f, limit+1))
	if e != nil || int64(len(b)) != st.Size() {
		return nil, nil, errors.New("changed install evidence")
	}
	again, e := os.Lstat(path)
	if e != nil || !os.SameFile(st, again) || st.Mode() != again.Mode() || st.Size() != again.Size() || !st.ModTime().Equal(again.ModTime()) {
		return nil, nil, errors.New("changed install evidence")
	}
	return b, st, nil
}
func diagnosticHash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func diagnosticSystemd(ctx context.Context, unit string, props ...string) map[string]string {
	args := []string{"show", "--no-pager", unit}
	for _, p := range props {
		args = append(args, "-p", p)
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/systemctl", args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C"}
	b, e := cmd.Output()
	if e != nil || len(b) > 65536 {
		return nil
	}
	out := map[string]string{}
	for _, l := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(l, "=")
		if ok {
			out[k] = v
		}
	}
	return out
}
func diagnosticUnitState(m map[string]string) string {
	if m == nil {
		return "UNVERIFIED"
	}
	switch m["LoadState"] {
	case "not-found":
		return "ABSENT"
	case "loaded":
	default:
		return "UNVERIFIED"
	}
	switch m["ActiveState"] {
	case "active":
		return "ACTIVE"
	case "inactive":
		return "INACTIVE"
	case "failed":
		return "FAILED"
	case "activating", "deactivating", "reloading":
		return "TRANSITIONING"
	}
	return "UNVERIFIED"
}

func diagnosticJournal() string {
	st, e := os.Lstat(nativePeerFreshJournalDir)
	if errors.Is(e, os.ErrNotExist) {
		return "ABSENT"
	}
	if e != nil {
		return "UNVERIFIED"
	}
	own, ok := st.Sys().(*syscall.Stat_t)
	if !ok || own.Uid != 0 || !st.IsDir() || st.Mode().Perm() != 0700 {
		return "UNSAFE"
	}
	return "PRESENT"
}
func diagnosticScopedArtifacts() string {
	count := 0
	for _, p := range []string{scopedGuardUnitPath, scopedLifecycleUnitPath, scopedIncusDropIn} {
		st, e := os.Lstat(p)
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			return "UNVERIFIED"
		}
		own, ok := st.Sys().(*syscall.Stat_t)
		if !ok || own.Uid != 0 || own.Nlink != 1 || !st.Mode().IsRegular() || st.Mode().Perm()&0022 != 0 {
			return "UNSAFE"
		}
		count++
	}
	if count == 0 {
		return "ABSENT"
	}
	if count == 3 {
		return "PRESENT"
	}
	return "PARTIAL"
}

func diagnosticContains(raw, want string) bool {
	for _, v := range strings.Fields(raw) {
		if v == want {
			return true
		}
	}
	return false
}

// Runtime scope is selected by actual argv, not the invoking CLI flags or
// a systemd file that has not been daemon-reloaded. Never expose raw argv.
func diagnosticRuntimeArgs(raw []byte, project, socket, db string) (bool, bool) {
	if len(raw) == 0 || len(raw) > 16384 || raw[len(raw)-1] != 0 {
		return false, false
	}
	args := strings.Split(string(raw[:len(raw)-1]), "\x00")
	if len(args) < 2 || args[1] != "serve" {
		return false, false
	}
	got := map[string]string{"db": "/var/lib/boxweave/ledger.sqlite", "incus-socket": "/var/lib/incus/unix.socket.user", "project": "", "scoped-peer-fresh": "false"}
	seen := map[string]bool{}
	// Known serve flags can be followed by a value; unknown or duplicate
	// values are not trustworthy runtime evidence.
	for i := 2; i < len(args); i++ {
		k, v, eq := strings.Cut(strings.TrimLeft(args[i], "-"), "=")
		if !strings.HasPrefix(args[i], "-") || seen[k] {
			return false, false
		}
		seen[k] = true
		switch k {
		case "scoped-peer-fresh":
			if !eq {
				v = "true"
			}
			if v != "true" && v != "false" {
				return false, false
			}
		case "db", "incus-socket", "project", "listen", "service-user":
			if !eq {
				i++
				if i >= len(args) {
					return false, false
				}
				v = args[i]
			}
			if v == "" {
				return false, false
			}
		default:
			return false, false
		}
		got[k] = v
	}
	return got["scoped-peer-fresh"] == "true", got["project"] == project && got["db"] == db && got["incus-socket"] == socket
}

func diagnosticClassify(o DiagnosticInstallation, runtimeScoped bool) DiagnosticInstallation {
	o.Mode = "UNVERIFIED"
	o.Coverage = "PARTIAL"
	if o.Manifest != "VERIFIED" || o.InstalledIdentity != "VERIFIED" {
		return o
	}
	if o.Candidate != "LEGACY" && o.Candidate != "SCOPED_V2" {
		o.Reason = "UNSUPPORTED_MANIFEST_MODE"
		return o
	}
	if o.ScopedArtifacts == "UNSAFE" || o.Candidate == "LEGACY" && (o.ScopedArtifacts == "PRESENT" || o.ScopedArtifacts == "PARTIAL" || o.Dependencies == "CONFLICT") || o.RootJournal == "UNSAFE" || o.Candidate == "LEGACY" && (o.RootJournal == "PRESENT" || o.GuardUnit != "ABSENT" && o.GuardUnit != "UNVERIFIED" || o.LifecycleUnit != "ABSENT" && o.LifecycleUnit != "UNVERIFIED") || o.Candidate == "SCOPED_V2" && o.RootJournal == "ABSENT" {
		o.Mode = "CONFLICT"
		o.Reason = "ARCHITECTURE_EVIDENCE_CONFLICT"
		return o
	}
	if o.Runtime != "VERIFIED" {
		return o
	}
	if runtimeScoped != (o.Candidate == "SCOPED_V2") {
		o.Mode = "CONFLICT"
		o.Reason = "RUNTIME_MODE_CONFLICT"
		return o
	}
	if o.Candidate == "LEGACY" && o.RootJournal == "ABSENT" && o.GuardUnit == "ABSENT" && o.LifecycleUnit == "ABSENT" && o.ScopedArtifacts == "ABSENT" && o.Dependencies == "LEGACY_ABSENT" {
		o.Mode = "LEGACY"
		o.Coverage = "MODE_PROVEN"
		o.Reason = ""
		return o
	}
	if o.Candidate == "SCOPED_V2" && o.RootJournal == "PRESENT" && o.ScopedArtifacts == "PRESENT" && o.RootProof == "VERIFIED" && o.GuardUnit != "ABSENT" && o.GuardUnit != "UNVERIFIED" && o.LifecycleUnit != "ABSENT" && o.LifecycleUnit != "UNVERIFIED" && o.Dependencies == "OBSERVED" {
		o.Mode = "SCOPED_V2"
		o.Coverage = "MODE_PROVEN"
		o.Reason = ""
		return o
	}
	return o
}

// ObserveDiagnosticInstallation is a fixed-path, bounded, metadata and
// existing Root-journal observation. Do not call installer Status as the
// restricted worker: that includes root-only checks and executable probes.
func ObserveDiagnosticInstallation(ctx context.Context, p Plan, db string) DiagnosticInstallation {
	o := DiagnosticInstallation{Mode: "UNVERIFIED", Coverage: "PARTIAL", Manifest: "UNVERIFIED", InstalledIdentity: "UNVERIFIED", Runtime: "UNVERIFIED", RootJournal: "UNVERIFIED", RootProof: "UNVERIFIED", Dependencies: "UNVERIFIED", GuardUnit: "UNVERIFIED", LifecycleUnit: "UNVERIFIED", ServiceUnit: "UNVERIFIED"}
	if p.User != "boxweave" || p.StateDir != "/var/lib/boxweave" || p.BinaryPath != "/usr/local/bin/boxweave" || p.UnitPath != "/etc/systemd/system/boxweave.service" || db != filepath.Join(p.StateDir, "ledger.sqlite") || p.Socket != "/var/lib/incus/unix.socket.user" {
		o.Reason = "CUSTOM_OR_UNINSTALLED_SCOPE"
		return o
	}
	o.RootJournal = diagnosticJournal()
	o.ScopedArtifacts = diagnosticScopedArtifacts()
	o.StaticContract = "UNVERIFIED"
	svc := diagnosticSystemd(ctx, "boxweave.service", "LoadState", "ActiveState", "MainPID", "FragmentPath", "Wants", "Requires", "After")
	guard := diagnosticSystemd(ctx, scopedGuardUnitName, "LoadState", "ActiveState")
	helper := diagnosticSystemd(ctx, scopedLifecycleUnitName, "LoadState", "ActiveState")
	o.ServiceUnit = diagnosticUnitState(svc)
	o.GuardUnit = diagnosticUnitState(guard)
	o.LifecycleUnit = diagnosticUnitState(helper)
	uid, _, e := ServiceIdentity(p)
	if e != nil || p.Project != "user-"+strconv.Itoa(uid) {
		o.Reason = "SERVICE_IDENTITY_UNVERIFIED"
		return o
	}
	st, e := os.Lstat(p.StateDir)
	if e != nil || !st.IsDir() || st.Mode().Perm() != 0700 {
		o.Reason = "STATE_DIRECTORY_UNVERIFIED"
		return o
	}
	ownDir, ok := st.Sys().(*syscall.Stat_t)
	if !ok || ownDir.Uid != uint32(uid) {
		o.Reason = "STATE_DIRECTORY_UNVERIFIED"
		return o
	}
	b, _, e := diagnosticReadFile(manifestPath(p), uint32(uid), true, 65536)
	if errors.Is(e, os.ErrNotExist) {
		o.Manifest = "ABSENT"
		o.Reason = "MANIFEST_MISSING"
		if o.RootJournal == "PRESENT" {
			o.Reason = "PARTIAL_SCOPED_INSTALLATION"
		}
		return o
	}
	if e != nil {
		o.Reason = "MANIFEST_UNVERIFIED"
		return o
	}
	var m Manifest
	if json.Unmarshal(b, &m) != nil || m.Schema != manifestSchema || m.StateDir != p.StateDir || m.BinaryPath != p.BinaryPath || m.Project != p.Project || m.ServiceUser != p.User || m.Socket != p.Socket {
		o.Reason = "MANIFEST_SCOPE_UNVERIFIED"
		return o
	}
	o.Manifest = "VERIFIED"
	switch m.ScopedPeerMode {
	case "":
		o.Candidate = "LEGACY"
	case nativePeerFreshMode:
		o.Candidate = "SCOPED_V2"
	default:
		o.Candidate = "UNSUPPORTED"
	}
	unit, _, e := diagnosticReadFile(p.UnitPath, 0, false, 65536)
	if e != nil || diagnosticHash(unit) != m.UnitSHA256 {
		o.Reason = "UNIT_IDENTITY_UNVERIFIED"
		return diagnosticClassify(o, false)
	}
	exe, exeStat, e := diagnosticReadFile(p.BinaryPath, 0, false, 128<<20)
	if e != nil || diagnosticHash(exe) != m.BinarySHA256 {
		o.Reason = "BINARY_IDENTITY_UNVERIFIED"
		return diagnosticClassify(o, false)
	}
	o.InstalledIdentity = "VERIFIED"
	if o.Candidate == "SCOPED_V2" {
		if os.Geteuid() == 0 && o.RootJournal == "PRESENT" {
			r, _, proofErr := nativePeerFreshVerifiedPlan()
			if proofErr == nil && r.Project == p.Project && r.Bridge == "incusbr-"+strconv.Itoa(uid) && nativePeerFreshQuarantineReceipt(r) == nil {
				o.RootProof = "VERIFIED"
			}
			if verifyScopedInstalled(m) == nil {
				o.StaticContract = "VERIFIED"
			}
		} else {
			o.RootProof = "ROOT_AUTHORITY_REQUIRED"
		}
		incus := diagnosticSystemd(ctx, "incus.service", "Requires", "After")
		if svc != nil && incus != nil && diagnosticContains(incus["Requires"], scopedGuardUnitName) && diagnosticContains(incus["After"], scopedGuardUnitName) && diagnosticContains(svc["After"], scopedLifecycleUnitName) && (diagnosticContains(svc["Wants"], scopedLifecycleUnitName) || diagnosticContains(svc["Requires"], scopedLifecycleUnitName)) {
			o.Dependencies = "OBSERVED"
		}
	} else if o.Candidate == "LEGACY" {
		o.RootProof = "NOT_APPLICABLE"
		o.StaticContract = "NOT_APPLICABLE"
		incus := diagnosticSystemd(ctx, "incus.service", "Requires", "After")
		if svc != nil && incus != nil {
			if diagnosticContains(incus["Requires"], scopedGuardUnitName) || diagnosticContains(incus["After"], scopedGuardUnitName) || diagnosticContains(svc["Requires"], scopedLifecycleUnitName) || diagnosticContains(svc["Wants"], scopedLifecycleUnitName) || diagnosticContains(svc["After"], scopedLifecycleUnitName) {
				o.Dependencies = "CONFLICT"
			} else {
				o.Dependencies = "LEGACY_ABSENT"
			}
		}
	}
	if svc == nil || svc["FragmentPath"] != p.UnitPath || o.ServiceUnit != "ACTIVE" {
		o.Reason = "RUNTIME_UNVERIFIED"
		return diagnosticClassify(o, false)
	}
	pid, e := strconv.Atoi(svc["MainPID"])
	if e != nil || pid < 2 {
		o.Reason = "RUNTIME_UNVERIFIED"
		return diagnosticClassify(o, false)
	}
	proc := filepath.Join("/proc", strconv.Itoa(pid))
	ps, e := os.Stat(proc)
	if e != nil {
		o.Reason = "RUNTIME_UNVERIFIED"
		return diagnosticClassify(o, false)
	}
	own, ok := ps.Sys().(*syscall.Stat_t)
	if !ok || own.Uid != uint32(uid) {
		o.Reason = "RUNTIME_ACTOR_UNVERIFIED"
		return diagnosticClassify(o, false)
	}
	actual, e := os.Stat(filepath.Join(proc, "exe"))
	if e != nil || !os.SameFile(exeStat, actual) {
		o.Reason = "RUNTIME_BINARY_UNVERIFIED"
		return diagnosticClassify(o, false)
	}
	// proc cmdline is a pseudo-file with size zero; bound reads explicitly.
	f, e := os.Open(filepath.Join(proc, "cmdline"))
	if e != nil {
		o.Reason = "RUNTIME_ARGV_UNVERIFIED"
		return diagnosticClassify(o, false)
	}
	raw, e := io.ReadAll(io.LimitReader(f, 16385))
	f.Close()
	scoped, valid := diagnosticRuntimeArgs(raw, p.Project, p.Socket, db)
	again := diagnosticSystemd(ctx, "boxweave.service", "MainPID", "ActiveState", "FragmentPath")
	ps2, e2 := os.Stat(proc)
	actual2, e3 := os.Stat(filepath.Join(proc, "exe"))
	// Re-read manifest and root journal before classifying across sources.
	b2, _, e4 := diagnosticReadFile(manifestPath(p), uint32(uid), true, 65536)
	if e != nil || !valid || e2 != nil || e3 != nil || e4 != nil || string(b) != string(b2) || diagnosticJournal() != o.RootJournal || diagnosticScopedArtifacts() != o.ScopedArtifacts || !os.SameFile(ps, ps2) || !os.SameFile(exeStat, actual2) || again == nil || again["MainPID"] != svc["MainPID"] || again["ActiveState"] != "active" || again["FragmentPath"] != p.UnitPath {
		o.Reason = "RUNTIME_SCOPE_OR_EPOCH_UNVERIFIED"
		return diagnosticClassify(o, false)
	}
	o.Runtime = "VERIFIED"
	o.Reason = "MODE_EVIDENCE_INCOMPLETE"
	return diagnosticClassify(o, scoped)
}
