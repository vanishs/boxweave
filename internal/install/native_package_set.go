package install

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

// The Ubuntu archive packages and Zabbly's split packages differ. Resolve
// the exact coupled package set from APT's package metadata, not a floating
// candidate. Installation and migration keep ordinary APT authentication.
func nativePackageNamesFromMetadata(raw, version string) ([]string, error) {
	if !ubuntuIncusVersion.MatchString(version) {
		return nil, errors.New("invalid native version")
	}
	var result []string
	matched := false
	baseWord := regexp.MustCompile(`(^|[^A-Za-z0-9+.-])incus-base([^A-Za-z0-9+.-]|$)`)
	for _, block := range strings.Split(strings.TrimSpace(raw), "\n\n") {
		var pkg, ver, depends string
		active := false
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "Package: "):
				pkg = strings.TrimSpace(strings.TrimPrefix(line, "Package: "))
				active = false
			case strings.HasPrefix(line, "Version: "):
				ver = strings.TrimSpace(strings.TrimPrefix(line, "Version: "))
				active = false
			case strings.HasPrefix(line, "Depends: "):
				depends = strings.TrimSpace(strings.TrimPrefix(line, "Depends: "))
				active = true
			case active && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")):
				depends += " " + strings.TrimSpace(line)
			default:
				active = false
			}
		}
		if pkg != "incus" || ver != version {
			continue
		}
		names := []string{"incus", "incus-client"}
		foundBase := false
		for _, d := range strings.Split(depends, ",") {
			d = strings.TrimSpace(d)
			if !baseWord.MatchString(d) {
				continue
			}
			if d != "incus-base (= "+version+")" || foundBase {
				return nil, errors.New("unrecognized or ambiguous exact incus-base dependency")
			}
			foundBase = true
			names = append(names, "incus-base")
		}
		if matched && !slices.Equal(result, names) {
			return nil, errors.New("conflicting exact native package metadata")
		}
		matched = true
		result = names
	}
	if !matched {
		return nil, errors.New("exact native package metadata unavailable")
	}
	return result, nil
}
func nativePackageNames(version string) ([]string, error) {
	if !ubuntuIncusVersion.MatchString(version) {
		return nil, errors.New("invalid native version")
	}
	b, err := exec.Command("apt-cache", "show", "incus="+version).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exact native APT metadata unavailable: %w", err)
	}
	return nativePackageNamesFromMetadata(string(b), version)
}
func nativePackageArgs(names []string, version string) []string {
	args := make([]string, 0, len(names))
	for _, name := range names {
		args = append(args, name+"="+version)
	}
	return args
}
func validNativePackageNames(names []string) bool {
	return slices.Equal(names, []string{"incus", "incus-client"}) ||
		slices.Equal(names, []string{"incus", "incus-client", "incus-base"})
}
func verifyNativePackageVersions(names []string, version string) error {
	if !validNativePackageNames(names) || !ubuntuIncusVersion.MatchString(version) {
		return errors.New("invalid native package set")
	}
	for _, name := range names {
		out, err := exec.Command("dpkg-query", "-W", "-f=${Status} ${Version}", name).CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != "install ok installed "+version {
			return fmt.Errorf("exact %s package version postcondition mismatch", name)
		}
	}
	return nil
}
