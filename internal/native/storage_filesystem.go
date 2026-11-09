package native

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const minNativePoolFreeBytes uint64 = 64 << 20

var (
	errNativeMountAbsent = errors.New("selected nondefault dir pool backing mount absent")
	errNativeSpaceLow = errors.New("selected native storage free space insufficient")
)

// localDirPoolSpace checks the ACTUAL filesystem below Incus's canonical pool
// mount. The Incus SDK redacts source from confined actors; the host root
// filesystem is not a substitute for the selected pool's statfs values.
// This is a read-only readiness observation, not a durable allocation promise.
func localDirPoolSpace(poolName string) (uint64, bool, error) {
	if poolName == "" || filepath.Base(poolName) != poolName || poolName == "." ||
		poolName == ".." || strings.ContainsAny(poolName, "/\\\x00") {
		return 0, false, errors.New("invalid selected native storage pool name")
	}
	return inspectDirPoolFilesystem(poolName, filepath.Join("/var/lib/incus/storage-pools", poolName), "/")
}

// Isolated for tests: never mounts, modifies, or repairs any storage.
func inspectDirPoolFilesystem(poolName, path, rootPath string) (uint64, bool, error) {
	st, err := os.Lstat(path)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return 0, false, errors.New("selected native storage directory unavailable")
	}
	root, err := os.Stat(rootPath)
	if err != nil {
		return 0, false, fmt.Errorf("host root filesystem unavailable: %w", err)
	}
	selectedDev, ok1 := st.Sys().(*syscall.Stat_t)
	rootDev, ok2 := root.Sys().(*syscall.Stat_t)
	if !ok1 || !ok2 {
		return 0, false, errors.New("native storage device identity unavailable")
	}
	separate := selectedDev.Dev != rootDev.Dev
	// Default dir on the root filesystem is supported. A separately selected
	// nondefault dir pool must not silently fall back to host root when the
	// backing mount vanishes; fail closed rather than claim readiness.
	if poolName != "default" && !separate {
		return 0, false, errNativeMountAbsent
	}
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return 0, separate, fmt.Errorf("actual selected storage filesystem unavailable: %w", err)
	}
	if fs.Bsize <= 0 || (fs.Bavail != 0 && uint64(fs.Bsize) > math.MaxUint64/fs.Bavail) {
		return 0, separate, errors.New("invalid native storage filesystem capacity")
	}
	free := uint64(fs.Bsize) * fs.Bavail
	if free < minNativePoolFreeBytes {
		return free, separate, fmt.Errorf("%w: %d free bytes below minimum %d",
			errNativeSpaceLow, free, minNativePoolFreeBytes)
	}
	return free, separate, nil
}
