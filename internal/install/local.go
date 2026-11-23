// Package install stages an explicitly approved, host-local service without
// taking over Incus, restarting unrelated services, or changing shared policy.
package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Plan struct {
	User                string
	StateDir            string
	BinaryPath          string
	UnitPath            string
	SourceBinary        string
	Project             string
	Socket              string
	Version             string
	ExpectedDigest      string
	RetirementTarget    *NativePrepareTarget
	RetirementReviewSHA string
}
type Manifest struct {
	Schema                    int    `json:"schema"`
	Version                   string `json:"version"`
	BuildCommit               string `json:"build_commit,omitempty"`
	BinarySHA256              string `json:"binary_sha256"`
	UnitSHA256                string `json:"unit_sha256"`
	ServiceActive             bool   `json:"service_active"`
	BinaryPath                string `json:"binary_path"`
	StateDir                  string `json:"state_dir"`
	ServiceUser               string `json:"service_user"`
	Project                   string `json:"native_project"`
	Socket                    string `json:"native_socket"`
	RestartRequired           bool   `json:"restart_required"`
	RetirementUnitSHA256      string `json:"retirement_unit_sha256,omitempty"`
	RetirementBridge          string `json:"retirement_bridge,omitempty"`
	RetirementACL             string `json:"retirement_acl,omitempty"`
	RetirementNativeReviewSHA string `json:"retirement_native_review_sha256,omitempty"`
}

const manifestSchema = 2

var safeValue = regexp.MustCompile(`^[a-zA-Z0-9_./-]+$`)

func safe(p Plan) error {
	if p.User == "" || p.Project == "" || p.Version == "" || p.StateDir == "" || p.BinaryPath == "" || p.UnitPath == "" || p.Socket == "" {
		return errors.New("missing explicit install parameter")
	}
	for _, v := range []string{p.User, p.Project, p.Version, p.StateDir, p.BinaryPath, p.UnitPath, p.Socket} {
		if !safeValue.MatchString(v) {
			return errors.New("unsafe install path or value")
		}
	}
	for _, v := range []string{p.StateDir, p.BinaryPath, p.UnitPath, p.Socket} {
		if !filepath.IsAbs(v) || filepath.Clean(v) != v {
			return errors.New("absolute clean install path required")
		}
	}
	if p.StateDir != "/var/lib/boxweave" || p.BinaryPath != "/usr/local/bin/boxweave" || p.UnitPath != "/etc/systemd/system/boxweave.service" {
		return errors.New("nonstandard install target requires a separate approved installer design")
	}
	return nil
}
func ServiceIdentity(p Plan) (int, int, error) {
	if err := safe(p); err != nil {
		return 0, 0, err
	}
	u, err := user.Lookup(p.User)
	if err != nil {
		return 0, 0, fmt.Errorf("create dedicated non-root incus-only user first: %w", err)
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, 0, err
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return 0, 0, err
	}
	if uid == 0 {
		return 0, 0, errors.New("service root forbidden")
	}
	incusGroup, err := user.LookupGroup("incus")
	if err != nil {
		return 0, 0, errors.New("incus group not installed")
	}
	gs, err := u.GroupIds()
	if err != nil {
		return 0, 0, err
	}
	hasIncus := false
	for _, g := range gs {
		if g == incusGroup.Gid {
			hasIncus = true
		}
	}
	if !hasIncus {
		return 0, 0, errors.New("service identity is not in native incus group")
	}
	for _, forbidden := range []string{"incus-admin", "sudo", "docker", "lxd"} {
		g, e := user.LookupGroup(forbidden)
		if e == nil {
			for _, id := range gs {
				if id == g.Gid {
					return 0, 0, fmt.Errorf("service identity has forbidden group %s", forbidden)
				}
			}
		}
	}
	return uid, gid, nil
}
func sha(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func manifestPath(p Plan) string { return filepath.Join(p.StateDir, "install-manifest.json") }
func ReadManifest(p Plan) (Manifest, error) {
	var m Manifest
	st, err := os.Lstat(manifestPath(p))
	if err != nil {
		return m, err
	}
	if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return m, errors.New("installer manifest is not a private regular file")
	}
	data, err := os.ReadFile(manifestPath(p))
	if err != nil {
		return m, err
	}
	if err = json.Unmarshal(data, &m); err != nil {
		return m, err
	}
	if m.Schema != manifestSchema || m.StateDir != p.StateDir || m.BinaryPath != p.BinaryPath || m.Project != p.Project || m.ServiceUser != p.User || m.Socket != p.Socket {
		return Manifest{}, errors.New("installed identity/manifest conflict: refuse adoption")
	}
	return m, nil
}
func Status(p Plan) (Manifest, error) {
	if err := safe(p); err != nil {
		return Manifest{}, err
	}
	m, err := ReadManifest(p)
	if err != nil {
		return m, err
	}
	for _, path := range []string{p.BinaryPath, p.UnitPath} {
		st, err := os.Lstat(path)
		if err != nil {
			return m, err
		}
		if !st.Mode().IsRegular() || st.Mode().Perm()&0022 != 0 {
			return m, errors.New("installer binary or unit is not a safe regular file")
		}
	}
	h, err := sha(p.BinaryPath)
	if err != nil {
		return m, err
	}
	if h != m.BinarySHA256 {
		return m, errors.New("installed binary digest mismatch")
	}
	unitHash, err := sha(p.UnitPath)
	if err != nil {
		return m, err
	}
	if m.UnitSHA256 == "" || unitHash != m.UnitSHA256 {
		return m, errors.New("installer unit checksum mismatch")
	}
	if m.BuildCommit != "" {
		build, err := verifiedSourceBuild(p, p.BinaryPath)
		if err != nil || build.Version != m.Version || build.Commit != m.BuildCommit {
			return m, errors.New("installed executable version/commit mismatch")
		}
	}
	if err = verifyInstalledRetirementUnit(m); err != nil {
		return m, err
	}
	m.ServiceActive = serviceActive() == nil
	return m, nil
}

// unit is used only by fresh Stage; Upgrade preserves the reviewed installed
// unit byte-for-byte, including legacy absence of this opt-in. An operator
// drop-in may narrow the effective startup policy. Manual serve stays local.
func unit(p Plan) string {
	header := "[Unit]\nDescription=BoxWeave v1 local MCP controller\nAfter=network.target incus.service\n"
	if p.RetirementTarget != nil {
		header += "Wants=boxweave-ipv4-retirement.service\n"
	}
	return header + "\n[Service]\n" +
		"Type=simple\nUser=" + p.User + "\nUMask=0077\n" +
		"Environment=BOXWEAVE_FORWARD_EXTERNAL_ALLOW=all\n" +
		"ExecStart=" + p.BinaryPath + " serve -db " + p.StateDir + "/ledger.sqlite -incus-socket " + p.Socket + " -project " + p.Project + " -listen 127.0.0.1:18650\n" +
		"Restart=on-failure\nRestartSec=5s\nNoNewPrivileges=true\nPrivateTmp=true\nProtectHome=true\nProtectSystem=strict\nReadWritePaths=" + p.StateDir + "\n" +
		"CapabilityBoundingSet=\nAmbientCapabilities=\n\n[Install]\nWantedBy=multi-user.target\n"
}
func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".boxweave-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(mode); err != nil {
		_ = f.Close()
		return err
	}
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Stage is invoked only by an explicit local root operator after an approved
// host-specific preflight. It does NOT enable/restart systemd or modify Incus.
// A different existing install must use a separately approved upgrade path.
func Stage(p Plan, approve bool) (Manifest, error) {
	if !approve || os.Geteuid() != 0 {
		return Manifest{}, errors.New("local root approval required")
	}
	if err := safe(p); err != nil {
		return Manifest{}, err
	}
	uid, gid, err := ServiceIdentity(p)
	if err != nil {
		return Manifest{}, err
	}
	if _, err = os.Lstat(p.Socket); err != nil {
		return Manifest{}, fmt.Errorf("restricted Incus socket unavailable: %w", err)
	}
	if p.SourceBinary == "" || !filepath.IsAbs(p.SourceBinary) || filepath.Clean(p.SourceBinary) != p.SourceBinary {
		return Manifest{}, errors.New("approved source binary absolute path required")
	}
	if len(p.ExpectedDigest) != 64 || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(p.ExpectedDigest) {
		return Manifest{}, errors.New("operator expected sha256 required")
	}
	st, err := os.Lstat(p.SourceBinary)
	if err != nil {
		return Manifest{}, err
	}
	if !st.Mode().IsRegular() {
		return Manifest{}, errors.New("approved binary source must be a real regular file")
	}
	h, err := sha(p.SourceBinary)
	if err != nil {
		return Manifest{}, err
	}
	if h != p.ExpectedDigest {
		return Manifest{}, errors.New("source digest differs from operator reviewed digest")
	}
	build, err := verifiedSourceBuild(p, p.SourceBinary)
	if err != nil {
		return Manifest{}, err
	}
	if _, err = os.Lstat(p.UnitPath); err == nil {
		old, e := Status(p)
		if e != nil {
			return Manifest{}, fmt.Errorf("existing unit/install not owned or unchanged: %w", e)
		}
		if old.BinarySHA256 != h {
			return Manifest{}, errors.New("different binary: explicit inspected upgrade required")
		}
		if p.RetirementTarget != nil && (old.RetirementBridge != p.RetirementTarget.Network || old.RetirementACL != p.RetirementTarget.ACL || old.RetirementNativeReviewSHA != p.RetirementReviewSHA) {
			return Manifest{}, errors.New("existing installer retirement scope differs; no adoption")
		}
		return old, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, err
	}
	if _, err = os.Lstat(p.BinaryPath); err == nil {
		return Manifest{}, errors.New("target binary exists but is not a verified BoxWeave install")
	}
	if _, e := os.Lstat(p.StateDir); e == nil {
		return Manifest{}, errors.New("existing state directory not governed by a verified manifest; refuse adoption")
	} else if !errors.Is(e, os.ErrNotExist) {
		return Manifest{}, e
	}
	retirementBody := ""
	if p.RetirementTarget != nil {
		if e := verifyRetirementExecutables(); e != nil {
			return Manifest{}, e
		}
		t := *p.RetirementTarget
		if p.User != "boxweave" || t.Project != p.Project || !validRetirementReviewSHA(p.RetirementReviewSHA) {
			return Manifest{}, errors.New("exact reviewed native retirement scope required")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		reviewed, e := NativePreparePlanReview(ctx, t)
		cancel()
		if e != nil || reviewed.ReviewSHA256 != p.RetirementReviewSHA {
			return Manifest{}, errors.New("native retirement scope review drifted")
		}
		retirementBody, e = NativePeerRootUnitText(NativePeerRootScope{DBPath: nativePeerRootInstalledDB, Project: p.Project, Bridge: t.Network, ACL: t.ACL, Approved: true, RetirementReview: true, RetirementCompact: true})
		if e != nil {
			return Manifest{}, e
		}
		if _, e = os.Lstat(nativeRetirementUnitPath); !errors.Is(e, os.ErrNotExist) {
			return Manifest{}, errors.New("existing retirement unit not owned by fresh installation")
		}
		if _, e = os.Lstat(nativeRetirementSocketPath); !errors.Is(e, os.ErrNotExist) {
			return Manifest{}, errors.New("existing retirement socket requires review")
		}
	}
	fmt.Fprintln(os.Stderr, "Fresh install: external Forward policy defaults to all; explicit external binds may expose public interfaces. Default bind is loopback; native Project authorization is unchanged.")
	if err = os.MkdirAll(p.StateDir, 0700); err != nil {
		return Manifest{}, err
	}
	if err = os.Chmod(p.StateDir, 0700); err != nil {
		return Manifest{}, err
	}
	if err = os.Chown(p.StateDir, uid, gid); err != nil {
		return Manifest{}, err
	}
	src, err := os.ReadFile(p.SourceBinary)
	if err != nil {
		return Manifest{}, err
	}
	if err = atomicWrite(p.BinaryPath, src, 0755); err != nil {
		return Manifest{}, err
	}
	digestUnit := sha256.Sum256([]byte(unit(p)))
	m := Manifest{Schema: manifestSchema, Version: build.Version, BuildCommit: build.Commit, BinarySHA256: h, UnitSHA256: hex.EncodeToString(digestUnit[:]), BinaryPath: p.BinaryPath, StateDir: p.StateDir, ServiceUser: p.User, Project: p.Project, Socket: p.Socket, RestartRequired: true}
	if retirementBody != "" {
		dig := sha256.Sum256([]byte(retirementBody))
		m.RetirementUnitSHA256 = hex.EncodeToString(dig[:])
		m.RetirementBridge = p.RetirementTarget.Network
		m.RetirementACL = p.RetirementTarget.ACL
		m.RetirementNativeReviewSHA = p.RetirementReviewSHA
		// Never overwrite or adopt an independently installed privileged unit.
		f, e := os.OpenFile(nativeRetirementUnitPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if e != nil {
			return m, e
		}
		_, e = f.WriteString(retirementBody)
		if e == nil {
			e = f.Sync()
		}
		cerr := f.Close()
		if e != nil {
			return m, e
		}
		if cerr != nil {
			return m, cerr
		}
		dir, e := os.Open(filepath.Dir(nativeRetirementUnitPath))
		if e != nil {
			return m, e
		}
		e = dir.Sync()
		dir.Close()
		if e != nil {
			return m, e
		}
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return m, err
	}
	if err = atomicWrite(manifestPath(p), b, 0600); err != nil {
		return m, err
	}
	if err = os.Chown(manifestPath(p), uid, gid); err != nil {
		return m, err
	}
	if err = atomicWrite(p.UnitPath, []byte(unit(p)), 0644); err != nil {
		return m, err
	}
	// systemctl daemon-reload/enable/start and effective native verification are
	// separate explicit local actions. A staged unit is NOT installed acceptance.
	return m, nil
}

func validRetirementReviewSHA(s string) bool {
	return len(s) == 64 && strings.Trim(s, "0123456789abcdef") == ""
}
func verifyInstalledRetirementUnit(m Manifest) error {
	if m.RetirementUnitSHA256 == "" {
		if m.RetirementBridge != "" || m.RetirementACL != "" || m.RetirementNativeReviewSHA != "" {
			return errors.New("incomplete retirement installer identity")
		}
		return nil
	}
	if !validRetirementReviewSHA(m.RetirementUnitSHA256) || m.RetirementBridge == "" || m.RetirementACL == "" || !validRetirementReviewSHA(m.RetirementNativeReviewSHA) {
		return errors.New("invalid retirement installer identity")
	}
	st, e := os.Lstat(nativeRetirementUnitPath)
	if e != nil {
		return e
	}
	owner, ok := st.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || !st.Mode().IsRegular() || st.Mode().Perm()&0022 != 0 || owner.Nlink != 1 {
		return errors.New("untrusted retirement installer unit")
	}
	h, e := sha(nativeRetirementUnitPath)
	if e != nil || h != m.RetirementUnitSHA256 {
		return errors.New("retirement unit digest mismatch")
	}
	// This is an independently generated fixed-scope unit, not trust in a
	// service-account writable manifest alone.
	body, e := NativePeerRootUnitText(NativePeerRootScope{DBPath: nativePeerRootInstalledDB, Project: m.Project, Bridge: m.RetirementBridge, ACL: m.RetirementACL, Approved: true, RetirementReview: true, RetirementCompact: true})
	if e != nil {
		return e
	}
	expected := sha256.Sum256([]byte(body))
	if hex.EncodeToString(expected[:]) != h {
		return errors.New("retirement unit differs from fixed installer scope")
	}
	return nil
}

func verifyRetirementNativePrepared(m Manifest) error {
	if m.RetirementUnitSHA256 == "" {
		return nil
	}
	st, e := os.Lstat(nativePrepareJournalPath)
	if e != nil {
		return e
	}
	own, ok := st.Sys().(*syscall.Stat_t)
	if !ok || own.Uid != 0 || own.Nlink != 1 || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return errors.New("untrusted root native prepare record")
	}
	r, e := NativePrepareStatus()
	if e != nil {
		return e
	}
	if r.Status != "APPLIED" || r.Plan.Target.Project != m.Project || r.Plan.Target.Network != m.RetirementBridge || r.Plan.Target.ACL != m.RetirementACL || r.Plan.ReviewSHA256 != m.RetirementNativeReviewSHA || nativePrepareDigest(r.Plan) != r.Plan.ReviewSHA256 {
		return errors.New("retirement verifier requires exact APPLIED native preparation")
	}
	return nil
}

func verifyRetirementExecutables() error {
	for _, path := range []string{"/usr/sbin/ip", "/usr/sbin/bridge", "/usr/sbin/nft", "/usr/sbin/conntrack"} {
		st, e := os.Stat(path)
		if e != nil {
			return fmt.Errorf("required retirement reader unavailable: %s", path)
		}
		own, ok := st.Sys().(*syscall.Stat_t)
		if !ok || own.Uid != 0 || !st.Mode().IsRegular() || st.Mode().Perm()&0022 != 0 || st.Mode().Perm()&0111 == 0 {
			return fmt.Errorf("untrusted retirement reader: %s", path)
		}
	}
	return nil
}
