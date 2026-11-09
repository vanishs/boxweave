package install

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type NativeMigrationJournal struct {
	Schema       int                 `json:"schema"`
	Phase        string              `json:"phase"`
	Plan         NativeMigrationPlan `json:"approved_plan"`
	SnapshotSHA  string              `json:"native_snapshot_sha256,omitempty"`
	OldServerDeb string              `json:"old_server_deb,omitempty"`
	OldClientDeb string              `json:"old_client_deb,omitempty"`
	ServerSHA    string              `json:"old_server_sha256,omitempty"`
	ClientSHA    string              `json:"old_client_sha256,omitempty"`
	CreatedUnix  int64               `json:"created_unix"`
	ObservedNow  string              `json:"current_observed_package,omitempty"`
}

func migrationReceipt() string { return filepath.Join(nativeMigrationDir, "receipt.json") }

func migrationDirSafe() error {
	st, err := os.Lstat(nativeMigrationDir)
	if err != nil {
		return err
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok || !st.IsDir() || sys.Uid != 0 || st.Mode().Perm() != 0700 || sys.Nlink < 2 {
		return errors.New("root-only migration journal directory invalid")
	}
	return nil
}

func readMigrationJournal() (NativeMigrationJournal, error) {
	var j NativeMigrationJournal
	if err := migrationDirSafe(); err != nil {
		return j, err
	}
	st, err := os.Lstat(migrationReceipt())
	if err != nil {
		return j, err
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok || !st.Mode().IsRegular() || sys.Uid != 0 || st.Mode().Perm() != 0600 {
		return j, errors.New("root-owned 0600 migration receipt required")
	}
	raw, err := os.ReadFile(migrationReceipt())
	if err != nil {
		return j, err
	}
	if err = json.Unmarshal(raw, &j); err != nil {
		return j, err
	}
	if j.Schema != 1 || j.Plan.HostDigest == "" || j.Plan.PlanSHA256 == "" ||
		!ubuntuIncusVersion.MatchString(j.Plan.OldPackage) ||
		!ubuntuIncusVersion.MatchString(j.Plan.NewPackage) {
		return j, errors.New("foreign or unsupported native migration journal")
	}
	host, err := migrationHostDigest()
	if err != nil || host != j.Plan.HostDigest {
		return j, errors.New("migration journal belongs to another host")
	}
	return j, nil
}

func appendMigrationAudit(phase string) error {
	if strings.ContainsAny(phase, "\n\r\t") || len(phase) > 100 {
		return errors.New("unsafe native audit phase")
	}
	fd, err := syscall.Open(filepath.Join(nativeMigrationDir, "audit.log"),
		syscall.O_CREAT|syscall.O_WRONLY|syscall.O_APPEND|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), "native-upgrade-audit")
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return errors.New("unsafe native audit file")
	}
	_, err = io.WriteString(f, fmt.Sprintf("%d %s\n", time.Now().UnixNano(), phase))
	if err != nil {
		return err
	}
	return f.Sync()
}

func setMigrationPhase(j *NativeMigrationJournal, phase string) error {
	j.Phase = phase
	j.ObservedNow = ""
	raw, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	if err = atomicWrite(migrationReceipt(), append(raw, '\n'), 0600); err != nil {
		return err
	}
	if err = appendMigrationAudit(phase); err != nil {
		return err
	}
	if f, err := os.Open(nativeMigrationDir); err == nil {
		_ = f.Sync()
		_ = f.Close()
	}
	return nil
}

func NativeMigrationStatus() (NativeMigrationJournal, error) {
	j, err := readMigrationJournal()
	if err != nil {
		return j, err
	}
	current, ok, err := installedIncusPackage()
	switch {
	case err != nil:
		j.ObservedNow = "PACKAGE_STATUS_UNAVAILABLE"
	case !ok:
		j.ObservedNow = "PACKAGE_NOT_FULLY_INSTALLED"
	default:
		j.ObservedNow = current
	}
	return j, nil
}

func shaNative(path string) (string, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() {
		return "", errors.New("migration backup must be a regular file")
	}
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

func assertNativeBackup(j NativeMigrationJournal) error {
	if j.SnapshotSHA == "" || j.ServerSHA == "" || j.ClientSHA == "" {
		return errors.New("no complete native snapshot; manual inspection required")
	}
	for _, item := range []struct{ path, hash string }{
		{filepath.Join(nativeMigrationDir, "native-state.tar"), j.SnapshotSHA},
		{filepath.Join(nativeMigrationDir, j.OldServerDeb), j.ServerSHA},
		{filepath.Join(nativeMigrationDir, j.OldClientDeb), j.ClientSHA},
	} {
		if strings.ContainsAny(filepath.Base(item.path), "/\\") || item.hash == "" {
			return errors.New("unsafe backup receipt path/hash")
		}
		h, err := shaNative(item.path)
		if err != nil || h != item.hash {
			return errors.New("native backup checksum mismatch")
		}
	}
	return nil
}
