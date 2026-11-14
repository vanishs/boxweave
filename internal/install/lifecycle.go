package install

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// The production paths are fixed by safe(Plan). All actions are explicit,
// host-local and root-only. No Incus or unrelated service restart is attempted.
const serviceName = "boxweave.service"

type BackupReceipt struct {
	Schema             int
	Previous           Manifest
	PreviousUnitSHA256 string
	PreviousDBSHA256   string
	PreviousWALSHA256  string
	PreviousSHMSHA256  string
}

func localApproval(p Plan, approved bool) error {
	if !approved || os.Geteuid() != 0 {
		return errors.New("explicit host-local root authorization required")
	}
	return safe(p)
}
func runLocal(name string, args ...string) error {
	c := exec.Command(name, args...)
	if err := c.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", filepath.Base(name), err)
	}
	return nil
}

// A service can be installed before any root disk, private NIC or Guest
// image exists. Never call the full Box-create doctor from installation.
func doctorAsService(p Plan, binary string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "runuser", "-u", p.User, "--", "env",
		"-u", "INCUS_CONF", "-u", "INCUS_CONFIG_DIR", "-u", "XDG_CONFIG_HOME",
		"HOME=/home/boxweave", "XDG_CONFIG_HOME=/home/boxweave/.config",
		binary, "status", "-db", filepath.Join(p.StateDir, "ledger.sqlite"),
		"-project", p.Project, "-incus-socket", p.Socket)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("restricted service preflight unavailable: %w", err)
	}
	var status struct {
		MCPReady bool `json:"mcp_ready"`
		Native   struct {
			ServiceReady bool   `json:"service_ready"`
			ErrorClass   string `json:"error_class"`
		} `json:"native"`
	}
	if err := json.Unmarshal(out, &status); err != nil {
		return fmt.Errorf("service preflight returned invalid status: %w", err)
	}
	if !status.MCPReady || !status.Native.ServiceReady {
		return fmt.Errorf("service preflight blocked: MCP schema ready=%t; Incus runtime ready=%t (%s)",
			status.MCPReady, status.Native.ServiceReady, status.Native.ErrorClass)
	}
	return nil
}
func serviceActive() error {
	return runLocal("systemctl", "is-active", "--quiet", serviceName)
}
func writeManifest(p Plan, m Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err = atomicWrite(manifestPath(p), append(b, '\n'), 0600); err != nil {
		return err
	}
	uid, gid, err := ServiceIdentity(p)
	if err != nil {
		return err
	}
	return os.Chown(manifestPath(p), uid, gid)
}

// A staged installation is NOT yet active or accepted.
func Activate(p Plan, approved bool) (Manifest, error) {
	if err := localApproval(p, approved); err != nil {
		return Manifest{}, err
	}
	m, err := Status(p)
	if err != nil {
		return m, err
	}
	if err = doctorAsService(p, p.BinaryPath); err != nil {
		return m, err
	}
	if m.RestartRequired {
		if err = runLocal("systemctl", "daemon-reload"); err != nil {
			return m, err
		}
		if err = runLocal("systemctl", "enable", "--now", serviceName); err != nil {
			return m, err
		}
	}
	if err = waitInstalledHealthy(p, m); err != nil {
		return m, fmt.Errorf("activated service is not stable: %w", err)
	}
	m.RestartRequired = false
	if err = writeManifest(p, m); err != nil {
		return m, err
	}
	return Status(p)
}
func backupDir(p Plan) string { return p.StateDir + "-installer-backup" }

// Copies only private regular files into an exclusive, root-owned backup.
// A backup is not the same thing as proof of a successful rollback.
func copyChecked(src, dst string) (string, error) {
	st, err := os.Lstat(src)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() {
		return "", errors.New("nonregular snapshot input refused")
	}
	hash, err := sha(src)
	if err != nil {
		return "", err
	}
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer f.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(out, f); err != nil {
		_ = out.Close()
		return "", err
	}
	if err = out.Sync(); err != nil {
		_ = out.Close()
		return "", err
	}
	if err = out.Close(); err != nil {
		return "", err
	}
	actual, err := sha(dst)
	if err != nil || actual != hash {
		return "", errors.New("backup checksum mismatch")
	}
	return hash, nil
}
func copyIfPresent(src, dst string) (string, error) {
	if _, err := os.Lstat(src); errors.Is(err, os.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return copyChecked(src, dst)
}
func snapshot(p Plan, old Manifest) (BackupReceipt, error) {
	dir := backupDir(p)
	if err := os.Mkdir(dir, 0700); err != nil {
		return BackupReceipt{}, fmt.Errorf("backup exists or unavailable: %w", err)
	}
	b := BackupReceipt{Schema: 1, Previous: old}
	var err error
	if _, err = copyChecked(p.BinaryPath, filepath.Join(dir, "binary")); err != nil {
		return b, err
	}
	if b.PreviousUnitSHA256, err = copyChecked(p.UnitPath, filepath.Join(dir, "unit")); err != nil {
		return b, err
	}
	path := filepath.Join(p.StateDir, "ledger.sqlite")
	if b.PreviousDBSHA256, err = copyIfPresent(path, filepath.Join(dir, "ledger.sqlite")); err != nil {
		return b, err
	}
	if b.PreviousWALSHA256, err = copyIfPresent(path+"-wal", filepath.Join(dir, "ledger.sqlite-wal")); err != nil {
		return b, err
	}
	if b.PreviousSHMSHA256, err = copyIfPresent(path+"-shm", filepath.Join(dir, "ledger.sqlite-shm")); err != nil {
		return b, err
	}
	data, err := json.Marshal(b)
	if err != nil {
		return b, err
	}
	if err = atomicWrite(filepath.Join(dir, "receipt.json"), data, 0600); err != nil {
		return b, err
	}
	if f, err := os.Open(dir); err == nil {
		_ = f.Sync()
		_ = f.Close()
	}
	return b, nil
}
func checkSource(p Plan) (string, error) {
	if p.SourceBinary == "" || !filepath.IsAbs(p.SourceBinary) || filepath.Clean(p.SourceBinary) != p.SourceBinary {
		return "", errors.New("absolute approved binary source required")
	}
	if len(p.ExpectedDigest) != 64 || strings.Trim(p.ExpectedDigest, "0123456789abcdef") != "" {
		return "", errors.New("operator-reviewed sha256 required")
	}
	st, err := os.Lstat(p.SourceBinary)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() {
		return "", errors.New("source must be a real regular file")
	}
	h, err := sha(p.SourceBinary)
	if err != nil || h != p.ExpectedDigest {
		return "", errors.New("source binary digest mismatch")
	}
	return h, nil
}

// Upgrade is single-generation and fail-closed. Existing Incus and unrelated
// workloads are never upgraded; only the verified BoxWeave unit is stopped.
func Upgrade(p Plan, approved bool) (Manifest, error) {
	if err := localApproval(p, approved); err != nil {
		return Manifest{}, err
	}
	old, err := Status(p)
	if err != nil {
		return old, err
	}
	if old.RestartRequired {
		return old, errors.New("unfinished stage cannot be upgraded")
	}
	if err = serviceActive(); err != nil {
		return old, errors.New("old service is not active")
	}
	h, err := checkSource(p)
	if err != nil {
		return old, err
	}
	build, err := verifiedSourceBuild(p, p.SourceBinary)
	if err != nil {
		return old, err
	}
	if h == old.BinarySHA256 {
		return old, nil
	}
	if err = doctorAsService(p, p.SourceBinary); err != nil {
		return old, err
	}
	if _, err = os.Lstat(backupDir(p)); !errors.Is(err, os.ErrNotExist) {
		return old, errors.New("pending backup must be reviewed before another upgrade")
	}
	if err = runLocal("systemctl", "stop", serviceName); err != nil {
		return old, err
	}
	b, err := snapshot(p, old)
	if err != nil {
		_ = runLocal("systemctl", "start", serviceName)
		return old, fmt.Errorf("snapshot failed without binary modification: %w", err)
	}
	if b.Previous.BinarySHA256 != old.BinarySHA256 {
		return old, errors.New("snapshot previous identity mismatch")
	}
	newBinary, err := os.ReadFile(p.SourceBinary)
	if err != nil {
		_ = runLocal("systemctl", "start", serviceName)
		return old, err
	}
	if err = atomicWrite(p.BinaryPath, newBinary, 0755); err != nil {
		_, rollback := Rollback(p, true)
		return old, fmt.Errorf("upgrade write: %v; rollback: %v", err, rollback)
	}
	next := old
	next.Version = build.Version
	next.BuildCommit = build.Commit
	next.BinarySHA256 = h
	next.RestartRequired = true
	if err = writeManifest(p, next); err != nil {
		_, rollback := Rollback(p, true)
		return old, fmt.Errorf("upgrade manifest: %v; rollback: %v", err, rollback)
	}
	result, err := Activate(p, true)
	if err != nil {
		_, rollback := Rollback(p, true)
		return old, fmt.Errorf("upgrade activation: %v; rollback: %v", err, rollback)
	}
	return result, nil
}
func inspectBackup(p Plan) (BackupReceipt, error) {
	var b BackupReceipt
	dir := backupDir(p)
	st, err := os.Lstat(dir)
	if err != nil {
		return b, err
	}
	info, ok := st.Sys().(*syscall.Stat_t)
	if !ok || !st.IsDir() || st.Mode().Perm() != 0700 || info.Uid != 0 {
		return b, errors.New("backup directory must be root-owned and private")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "receipt.json"))
	if err != nil {
		return b, err
	}
	if err = json.Unmarshal(raw, &b); err != nil {
		return b, err
	}
	m := b.Previous
	if b.Schema != 1 || m.Schema != manifestSchema || m.StateDir != p.StateDir ||
		m.BinaryPath != p.BinaryPath || m.ServiceUser != p.User || m.Project != p.Project ||
		m.Socket != p.Socket || m.BinarySHA256 == "" {
		return b, errors.New("backup belongs to a different installation")
	}
	for _, f := range []struct{ name, hash string }{
		{"binary", m.BinarySHA256}, {"unit", b.PreviousUnitSHA256},
		{"ledger.sqlite", b.PreviousDBSHA256}, {"ledger.sqlite-wal", b.PreviousWALSHA256},
		{"ledger.sqlite-shm", b.PreviousSHMSHA256},
	} {
		if f.hash == "" {
			continue
		}
		h, err := sha(filepath.Join(dir, f.name))
		if err != nil || h != f.hash {
			return b, fmt.Errorf("backup file %s checksum mismatch", f.name)
		}
	}
	return b, nil
}
func restore(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return atomicWrite(dst, data, mode)
}

// Rollback is local and explicit; it preserves the original audit snapshot.
// It never asserts that a previously dispatched native effect was reversed.
func Rollback(p Plan, approved bool) (Manifest, error) {
	if err := localApproval(p, approved); err != nil {
		return Manifest{}, err
	}
	b, err := inspectBackup(p)
	if err != nil {
		return Manifest{}, err
	}
	if err = runLocal("systemctl", "stop", serviceName); err != nil {
		return Manifest{}, err
	}
	dir := backupDir(p)
	if err = restore(filepath.Join(dir, "binary"), p.BinaryPath, 0755); err != nil {
		return Manifest{}, err
	}
	if err = restore(filepath.Join(dir, "unit"), p.UnitPath, 0644); err != nil {
		return Manifest{}, err
	}
	db := filepath.Join(p.StateDir, "ledger.sqlite")
	for _, item := range []struct{ suffix, hash string }{
		{"", b.PreviousDBSHA256}, {"-wal", b.PreviousWALSHA256}, {"-shm", b.PreviousSHMSHA256},
	} {
		dst := db + item.suffix
		if item.hash == "" {
			if err = os.Remove(dst); err != nil && !errors.Is(err, os.ErrNotExist) {
				return Manifest{}, err
			}
			continue
		}
		if err = restore(filepath.Join(dir, "ledger.sqlite"+item.suffix), dst, 0600); err != nil {
			return Manifest{}, err
		}
		uid, gid, e := ServiceIdentity(p)
		if e != nil {
			return Manifest{}, e
		}
		if err = os.Chown(dst, uid, gid); err != nil {
			return Manifest{}, err
		}
	}
	old := b.Previous
	old.RestartRequired = true
	if err = writeManifest(p, old); err != nil {
		return Manifest{}, err
	}
	return Activate(p, true)
}
