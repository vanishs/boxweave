package native

import (
	"testing"
	"time"
)

func TestRC8HostRootQuotaWitnessFailClosed(t *testing.T) {
	const id = "4b887448-bc95-424f-9812-da83eb71190a"
	now := time.Unix(1795143691, 0)
	v := storageQuotaWitness{
		Schema: 1, Project: "user-1001", Pool: "default", Driver: "btrfs",
		BootID: "boot-a", MountDev: 23, MountIno: 456,
		ObservedUnix: now.Add(-time.Minute).Unix(),
		BoxUUID:      id, QuotaBytes: 2 << 30, Referenced: 2 << 30, QgroupMax: 2 << 30,
	}
	valid := func(w storageQuotaWitness, project, pool, driver, boot string, dev, ino uint64, when time.Time) bool {
		return quotaWitnessValid(w, project, pool, driver, boot, dev, ino, when)
	}
	if !valid(v, "user-1001", "default", "btrfs", "boot-a", 23, 456, now) {
		t.Fatal("lost exact independently measured host quota proof")
	}
	for _, tc := range []struct {
		name string
		edit func(*storageQuotaWitness)
	}{
		{"wrong-project", func(w *storageQuotaWitness) { w.Project = "user-1002" }},
		{"wrong-pool", func(w *storageQuotaWitness) { w.Pool = "other" }},
		{"unverified-driver", func(w *storageQuotaWitness) { w.Driver = "dir" }},
		{"changed-boot", func(w *storageQuotaWitness) { w.BootID = "boot-b" }},
		{"changed-device", func(w *storageQuotaWitness) { w.MountDev = 99 }},
		{"changed-inode", func(w *storageQuotaWitness) { w.MountIno = 99 }},
		{"too-small", func(w *storageQuotaWitness) { w.QuotaBytes = 1 << 30 }},
		{"unlimited", func(w *storageQuotaWitness) { w.QgroupMax = 0 }},
		{"size-mismatch", func(w *storageQuotaWitness) { w.QgroupMax = 3 << 30 }},
		{"not-full", func(w *storageQuotaWitness) { w.Referenced = 1 << 30 }},
		{"expired", func(w *storageQuotaWitness) { w.ObservedUnix = now.Add(-5 * time.Hour).Unix() }},
		{"future", func(w *storageQuotaWitness) { w.ObservedUnix = now.Add(3 * time.Minute).Unix() }},
		{"missing-box-id", func(w *storageQuotaWitness) { w.BoxUUID = "" }},
		{"wrong-schema", func(w *storageQuotaWitness) { w.Schema = 2 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := v
			tc.edit(&w)
			if valid(w, "user-1001", "default", "btrfs", "boot-a", 23, 456, now) {
				t.Fatal("unverified or stale host qgroup evidence accepted")
			}
		})
	}
	for _, tc := range []struct {
		project, pool, driver, boot string
		dev, ino                    uint64
	}{
		{"user-1002", "default", "btrfs", "boot-a", 23, 456},
		{"user-1001", "other", "btrfs", "boot-a", 23, 456},
		{"user-1001", "default", "dir", "boot-a", 23, 456},
		{"user-1001", "default", "btrfs", "boot-b", 23, 456},
		{"user-1001", "default", "btrfs", "boot-a", 22, 456},
		{"user-1001", "default", "btrfs", "boot-a", 23, 457},
	} {
		if valid(v, tc.project, tc.pool, tc.driver, tc.boot, tc.dev, tc.ino, now) {
			t.Fatalf("foreign witness admitted: %+v", tc)
		}
	}
}
