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
	errNativeBackingUnverified = errors.New("nondefault dir pool backing cannot be distinguished from host-root fallback by restricted actor")
	errNativeSpaceLow          = errors.New("selected native storage free space insufficient")
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
	// Root-backed named dir pools can be valid Incus configurations, but a
	// confined actor cannot read the original (redacted) pool source and so
	// cannot distinguish those from a missing formerly separate mount.
	// Do not claim the mount is missing; refuse readiness as unverified.
	if poolName != "default" && !separate {
		return 0, false, errNativeBackingUnverified
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


// dirProjectQuotaMountReady is a conservative, read-only Linux mount
// precondition. Incus 7.6 silently skips dir volume size quotas if project
// quotas are unsupported; a valid size string alone is not quota evidence.
// Accept only a dedicated ext4/XFS mount whose effective mount options
// explicitly enable project quotas. This is necessary but not sufficient:
// the installed native CI also checks real per-instance EDQUOT, including
// the sibling Exec output spool. No mount/provision/repair occurs here.
func dirProjectQuotaMountReady(selectedPath string, mountinfo string) bool {
	if selectedPath == "" || !filepath.IsAbs(selectedPath) || strings.ContainsAny(selectedPath, " \t\r\n") {
		return false
	}
	longest := -1
	verified := false
	for _, line := range strings.Split(mountinfo, "\n") {
		halves := strings.SplitN(line, " - ", 2)
		if len(halves) != 2 {
			continue
		}
		left, right := strings.Fields(halves[0]), strings.Fields(halves[1])
		if len(left) < 6 || len(right) < 3 {
			continue
		}
		point := left[4]
		if point != selectedPath && !(point == "/" || strings.HasPrefix(selectedPath, point+"/")) {
			continue
		}
		if len(point) <= longest {
			continue
		}
		longest = len(point)
		verified = (right[0] == "ext4" || right[0] == "xfs") &&
			(hasQuotaOption(left[5]) || hasQuotaOption(right[2]))
	}
	return longest >= 0 && verified
}

func hasQuotaOption(options string) bool {
	for _, opt := range strings.Split(options, ",") {
		if opt == "prjquota" || opt == "pquota" {
			return true
		}
	}
	return false
}

func localDirPoolQuotaReady(poolName string) bool {
	if poolName == "" || filepath.Base(poolName) != poolName ||
		strings.ContainsAny(poolName, "/\\\x00") {
		return false
	}
	info, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil || len(info) > 4<<20 {
		return false
	}
	return dirProjectQuotaMountReady(filepath.Join("/var/lib/incus/storage-pools", poolName), string(info))
}
