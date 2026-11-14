package native

import (
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

// The host operator, not BoxWeave, chooses Incus's storage driver.
func TestRC2NativeStorageAdmissionIgnoresDriver(t *testing.T) {
	for _, driver := range []string{"dir", "btrfs", "zfs", "lvm", "ceph"} {
		t.Run(driver, func(t *testing.T) {
			if !StoragePoolReady(&api.StoragePool{Driver: driver, Status: "Created"}) {
				t.Fatalf("Created native pool rejected only because of driver %s", driver)
			}
			if StoragePoolReady(&api.StoragePool{Driver: driver, Status: "Unavailable"}) {
				t.Fatalf("unavailable native pool %s accepted", driver)
			}
		})
	}
	if StoragePoolReady(nil) {
		t.Fatal("missing native pool accepted")
	}
}
