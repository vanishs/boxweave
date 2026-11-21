package native

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// The host root operator can place a bounded, independently measured Btrfs
// qgroup witness here using scripts/host_btrfs_quota_attest.py. A confined
// service has read access only. This is never an MCP mutation or an implicit
// claim that a pool driver alone enforces disk quotas.
const storageQuotaWitnessPath = "/etc/boxweave/storage-quota-witness.json"

const storageQuotaWitnessMaxAge = 4 * time.Hour

// Near-full proof additionally requires an observed EDQUOT on a bounded guest write.
const storageQuotaWitnessMaxSlack = 64 << 10

type storageQuotaWitness struct {
	Schema       int    `json:"schema"`
	Project      string `json:"project"`
	Pool         string `json:"pool"`
	Driver       string `json:"driver"`
	BootID       string `json:"boot_id"`
	MountDev     uint64 `json:"mount_dev"`
	MountIno     uint64 `json:"mount_ino"`
	ObservedUnix int64  `json:"observed_unix"`
	BoxUUID      string `json:"box_uuid"`
	QuotaBytes   uint64 `json:"quota_bytes"`
	Referenced   uint64 `json:"referenced_bytes"`
	QgroupMax    uint64 `json:"qgroup_max_referenced_bytes"`
	WriteRefusal string `json:"write_refusal,omitempty"`
}

func quotaWitnessValid(w storageQuotaWitness, project, pool, driver, boot string,
	dev, ino uint64, now time.Time) bool {
	if (w.Schema != 1 && w.Schema != 2) || project == "" || pool == "" || driver != "btrfs" ||
		w.Project != project || w.Pool != pool || w.Driver != driver ||
		boot == "" || w.BootID != boot ||
		w.MountDev != dev || w.MountIno != ino ||
		len(w.BoxUUID) != 36 || w.QuotaBytes < (2<<30) ||
		w.QgroupMax != w.QuotaBytes {
		return false
	}
	if w.Schema == 1 {
		if w.Referenced < w.QgroupMax || w.WriteRefusal != "" {
			return false
		}
	} else if w.WriteRefusal != "GUEST_EDQUOT_128K" || w.Referenced > w.QgroupMax ||
		w.QgroupMax-w.Referenced > storageQuotaWitnessMaxSlack {
		return false
	}
	when := time.Unix(w.ObservedUnix, 0)
	return !when.After(now.Add(2*time.Minute)) && !when.Before(now.Add(-storageQuotaWitnessMaxAge))
}

func secureRootOwnedRegular(path string, isDir bool) bool {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 ||
		info.Mode().Perm()&0022 != 0 {
		return false
	}
	if isDir && !info.IsDir() || !isDir && !info.Mode().IsRegular() {
		return false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == 0
}

// recentHostQuotaWitness requires a root-owned, non-writable parent and file;
// same kernel boot, same Incus pool mount inode/device, exact Project, pool and
// Btrfs backend, with a four-hour expiration. A stale/forged/missing witness
// never changes readiness and cannot turn a false quota result into true.
func recentHostQuotaWitness(project, pool, driver string) bool {
	if pool == "" || filepath.Base(pool) != pool || strings.ContainsAny(pool, "/\\\x00") ||
		!secureRootOwnedRegular(filepath.Dir(storageQuotaWitnessPath), true) ||
		!secureRootOwnedRegular(storageQuotaWitnessPath, false) {
		return false
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join("/var/lib/incus/storage-pools", pool))
	if err != nil {
		return false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	contents, err := os.ReadFile(storageQuotaWitnessPath)
	if err != nil || len(contents) > 4096 {
		return false
	}
	var w storageQuotaWitness
	if json.Unmarshal(contents, &w) != nil {
		return false
	}
	return quotaWitnessValid(w, project, pool, driver,
		strings.TrimSpace(string(boot)), uint64(st.Dev), st.Ino, time.Now())
}
