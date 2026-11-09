package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

const maintenanceLockFile = "/run/lock/boxweave-v1-maintenance.lock"

// A single kernel lock covers *all* state-changing installer/daemon actions.
// Nested Activate/Rollback calls within one approved upgrade reuse the same
// process and cannot race with a second CLI invocation. No daemon is added.
func acquireMaintenanceLock() (func(), error) {
	if os.Geteuid() != 0 {
		return nil, errors.New("native/installer maintenance requires local root")
	}
	fd, err := syscall.Open(maintenanceLockFile,
		syscall.O_CREAT|syscall.O_RDWR|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot open fixed maintenance lock: %w", err)
	}
	closeFD := func() { _ = syscall.Close(fd) }
	var st syscall.Stat_t
	if err = syscall.Fstat(fd, &st); err != nil {
		closeFD()
		return nil, err
	}
	if st.Mode&syscall.S_IFMT != syscall.S_IFREG || st.Mode&0777 != 0600 ||
		st.Uid != 0 || st.Nlink != 1 {
		closeFD()
		return nil, errors.New("maintenance lock path is unsafe or not owned by root")
	}
	if err = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		closeFD()
		return nil, fmt.Errorf("another local installer/recovery operation holds the maintenance lock: %w", err)
	}
	return func() {
		_ = syscall.Flock(fd, syscall.LOCK_UN)
		closeFD()
	}, nil
}
