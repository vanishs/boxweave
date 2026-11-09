package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// CommitUpgrade explicitly closes the previous-generation recovery window
// after an operator has verified the currently active instance. It does not
// delete SQLite state, alter Incus, or silently discard an upgrade snapshot.
func CommitUpgrade(p Plan, approved bool) (Manifest, error) {
	if err := localApproval(p, approved); err != nil {
		return Manifest{}, err
	}
	m, err := Status(p)
	if err != nil {
		return m, err
	}
	if m.RestartRequired || !m.ServiceActive {
		return m, errors.New("refuse commit until installed service is active and settled")
	}
	if err := doctorAsService(p, p.BinaryPath); err != nil {
		return m, err
	}
	if err := waitInstalledHealthy(p, m); err != nil {
		return m, err
	}
	if _, err := inspectBackup(p); err != nil {
		return m, fmt.Errorf("no complete verified prior backup: %w", err)
	}
	dir := backupDir(p)
	allowed := map[string]bool{
		"binary": true, "unit": true, "receipt.json": true,
		"ledger.sqlite": true, "ledger.sqlite-wal": true, "ledger.sqlite-shm": true,
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return m, err
	}
	for _, entry := range entries {
		if !allowed[entry.Name()] || !entry.Type().IsRegular() {
			return m, errors.New("unexpected backup entry; require manual review")
		}
		info, err := os.Lstat(filepath.Join(dir, entry.Name()))
		if err != nil {
			return m, err
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || st.Uid != 0 || info.Mode().Perm() != 0600 {
			return m, errors.New("backup file no longer private root-owned")
		}
	}
	for _, entry := range entries {
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return m, err
		}
	}
	if err := os.Remove(dir); err != nil {
		return m, err
	}
	return Status(p)
}
