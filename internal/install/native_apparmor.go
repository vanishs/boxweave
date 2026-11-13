package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Native Incus 7.6 needs to load AppArmor profiles for networks and archive
// extraction. security.nesting=true does not independently delegate that power.
//
// Only recognize two *observed* no-go configurations. A successful inspection
// is not proof that an inner Incus/Incus-user workload will work: real native
// postconditions remain mandatory. Never change the kernel security policy.
type nativeAppArmorHostState struct {
	Available bool
	Profile   string
	UserNS    bool
	Stacked   bool
}

func nativeUserNSFromMap(raw string) (bool, error) {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) == 0 || len(lines) > 128 {
		return false, errors.New("unreadable uid namespace mapping")
	}
	if len(lines) != 1 {
		return true, nil
	}
	fields := strings.Fields(lines[0])
	if len(fields) != 3 {
		return false, errors.New("malformed uid namespace mapping")
	}
	inside, err1 := strconv.ParseUint(fields[0], 10, 64)
	outside, err2 := strconv.ParseUint(fields[1], 10, 64)
	count, err3 := strconv.ParseUint(fields[2], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || count == 0 || inside != 0 {
		return false, errors.New("invalid uid namespace mapping")
	}
	// Linux's initial user namespace maps 0..UINT32_MAX-1 to itself.
	return outside != 0 || count != 4294967295, nil
}

func classifyNativeAppArmorHost(s nativeAppArmorHostState) error {
	if !s.Available {
		// Keep the old no-AppArmor baseline. Do not claim AppArmor protection.
		return nil
	}
	if s.Profile == "" {
		return errors.New("NATIVE_APPARMOR_UNVERIFIED: process profile cannot be identified")
	}
	if s.Profile != "unconfined" {
		return fmt.Errorf("NATIVE_APPARMOR_CONFINED: Incus 7.6 cannot safely manage nested network/archive profiles from confined process %q; use an independently verified host, not raw.lxc or security.privileged", s.Profile)
	}
	if s.UserNS && !s.Stacked {
		return errors.New("NATIVE_APPARMOR_NO_STACKING: unprivileged container is unconfined but has no delegated AppArmor policy namespace; nested incus-user/aa-exec profiles may be missing")
	}
	return nil
}

// nativeAppArmorHostPreflight is root-local and read-only. It does not infer
// confinement from the name of the container or the parent project. A missing
// stacking indicator in a user namespace is NOT treated as proof of support.
func nativeAppArmorHostPreflight() error {
	root := "/sys/kernel/security/apparmor"
	info, err := os.Stat(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil // upstream Incus also supports hosts without AppArmor
	}
	if err != nil || !info.IsDir() {
		return errors.New("NATIVE_APPARMOR_UNVERIFIED: cannot inspect AppArmor kernel state")
	}
	profile, err := os.ReadFile("/proc/self/attr/current")
	if err != nil {
		return fmt.Errorf("NATIVE_APPARMOR_UNVERIFIED: cannot inspect process confinement: %w", err)
	}
	mapping, err := os.ReadFile("/proc/self/uid_map")
	if err != nil {
		return fmt.Errorf("NATIVE_APPARMOR_UNVERIFIED: cannot inspect user namespace: %w", err)
	}
	userNS, err := nativeUserNSFromMap(string(mapping))
	if err != nil {
		return fmt.Errorf("NATIVE_APPARMOR_UNVERIFIED: %w", err)
	}
	stackedData, err := os.ReadFile(filepath.Join(root, ".ns_stacked"))
	if err != nil && userNS {
		return fmt.Errorf("NATIVE_APPARMOR_UNVERIFIED: nested AppArmor namespace indicator unavailable: %w", err)
	}
	stacked := strings.TrimSpace(string(stackedData))
	if userNS && stacked != "yes" && stacked != "no" {
		return errors.New("NATIVE_APPARMOR_UNVERIFIED: invalid nested namespace indicator")
	}
	return classifyNativeAppArmorHost(nativeAppArmorHostState{
		Available: true, Profile: strings.TrimSpace(string(profile)),
		UserNS: userNS, Stacked: stacked == "yes",
	})
}
