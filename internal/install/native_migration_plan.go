package install

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"slices"
	"strings"
)

const nativeMigrationDir = "/var/lib/boxweave-native-migration"

type NativeMigrationPlan struct {
	Mode       string   `json:"mode"`
	OldPackage string   `json:"old_package_version"`
	NewPackage string   `json:"new_package_version"`
	HostDigest string   `json:"host_digest"`
	PlanSHA256 string   `json:"plan_sha256"`
	Projects   []string `json:"observed_projects"`
	AptChanges []string `json:"apt_changes"`
	Packages   []string `json:"packages,omitempty"`
}

func migrationHostDigest() (string, error) {
	data, err := os.ReadFile("/etc/machine-id")
	if err != nil || strings.TrimSpace(string(data)) == "" {
		return "", errors.New("local machine-id required for exact host authorization")
	}
	h := sha256.Sum256([]byte("boxweave-native-host-v1|" + strings.TrimSpace(string(data))))
	return hex.EncodeToString(h[:]), nil
}

func installedClientVersion() (string, error) {
	out, err := exec.Command("dpkg-query", "-W", "-f=${Status} ${Version}", "incus-client").CombinedOutput()
	if err != nil || !strings.HasPrefix(strings.TrimSpace(string(out)), "install ok installed ") {
		return "", errors.New("exact Incus client dpkg status required")
	}
	return strings.TrimPrefix(strings.TrimSpace(string(out)), "install ok installed "), nil
}

// A global daemon cannot be upgraded under other consumers. Reject ALL
// existing instances, including stopped guests, and any foreign Project.
func migrationInventory() ([]string, error) {
	out, err := exec.Command("incus", "list", "--all-projects", "--format", "json").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("native instance inventory unavailable: %w", err)
	}
	var instances []json.RawMessage
	if err := json.Unmarshal(out, &instances); err != nil {
		return nil, err
	}
	if len(instances) != 0 {
		return nil, errors.New("native instances present: shared daemon migration refused")
	}
	out, err = exec.Command("incus", "project", "list", "--format", "json").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("native Project inventory unavailable: %w", err)
	}
	var projects []struct {
		Name   string            `json:"name"`
		Config map[string]string `json:"config"`
	}
	if err := json.Unmarshal(out, &projects); err != nil {
		return nil, err
	}
	u, err := user.Lookup("boxweave")
	if err != nil {
		return nil, errors.New("dedicated native user required")
	}
	own := "user-" + u.Uid
	if len(projects) != 2 {
		return nil, errors.New("foreign or extra Projects prevent native maintenance")
	}
	var ownOK, defOK bool
	var names []string
	for _, p := range projects {
		names = append(names, p.Name)
		switch p.Name {
		case "default":
			defOK = true
		case own:
			ownOK = p.Config["restricted"] == "true"
		default:
			return nil, errors.New("foreign Project prevents migration")
		}
	}
	if !ownOK || !defOK {
		return nil, errors.New("exact restricted service Project and default required")
	}
	out, err = exec.Command("runuser", confinedProjectListArgs(u.HomeDir)...).CombinedOutput()
	trimmed := strings.TrimSpace(string(out))
	if err != nil || strings.Count(trimmed, "\n") != 0 ||
		!strings.HasPrefix(trimmed, own+" ") {
		return nil, errors.New("native service user must have only its own Project")
	}
	if exec.Command("systemctl", "is-active", "--quiet", "boxweave.service").Run() == nil {
		return nil, errors.New("active BoxWeave service requires separate maintenance outage")
	}
	return names, nil
}

func exactAptMigration(from, to string, names []string) ([]string, string, error) {
	if !ubuntuIncusVersion.MatchString(from) || !ubuntuIncusVersion.MatchString(to) {
		return nil, "", errors.New("exact pinned Ubuntu Incus versions required")
	}
	if exec.Command("dpkg", "--compare-versions", from, "lt", to).Run() != nil {
		return nil, "", errors.New("target must be an Incus upgrade, not downgrade")
	}
	if !validNativePackageNames(names) {
		return nil, "", errors.New("invalid reviewed native package set")
	}
	args := append([]string{"-s", "--no-install-recommends", "install"}, nativePackageArgs(names, to)...)
	out, err := exec.Command("apt-get", args...).CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("APT signed simulation failed: %w", err)
	}
	simulation := string(out)
	for _, unsafe := range []string{"Remv ", "cannot be authenticated", "essential packages will be removed", "WARNING:"} {
		if strings.Contains(simulation, unsafe) {
			return nil, "", errors.New("unsafe APT change refused")
		}
	}
	var lines []string
	seen := map[string]bool{}
	for _, line := range strings.Split(simulation, "\n") {
		if !strings.HasPrefix(line, "Inst ") {
			continue
		}
		matched := false
		for _, name := range names {
			if strings.HasPrefix(line, "Inst "+name+" ["+from+"] ("+to+" ") && !seen[name] {
				seen[name] = true
				matched = true
				break
			}
		}
		// Rollback backs up only the exact coupled native package set. Refuse
		// unrelated dependency upgrades/removals rather than promise rollback
		// of a package whose original signed bytes were not retained.
		if !matched {
			return nil, "", errors.New("APT migration changes unreviewed dependency or native version")
		}
		lines = append(lines, strings.TrimSpace(line))
	}
	if len(seen) != len(names) {
		return nil, "", errors.New("APT did not plan exact coupled old/new native packages")
	}
	return lines, simulation, nil
}

func NativeMigrationReview(from, to string) (NativeMigrationPlan, error) {
	p := NativeMigrationPlan{Mode: "review-only", OldPackage: from, NewPackage: to}
	if os.Geteuid() != 0 {
		return p, errors.New("root-only native inventory")
	}
	if err := ubuntuNativeBaseline(); err != nil {
		return p, err
	}
	if _, err := os.Lstat(nativeMigrationDir); err == nil {
		return p, errors.New("previous migration journal exists: inspect, do not retry")
	} else if !errors.Is(err, os.ErrNotExist) {
		return p, err
	}
	if out, err := exec.Command("apt-config", "dump").CombinedOutput(); err != nil {
		return p, err
	} else {
		for _, bad := range []string{"AllowUnauthenticated \"true\"", "AllowInsecureRepositories \"true\""} {
			if strings.Contains(string(out), bad) {
				return p, errors.New("insecure apt override")
			}
		}
	}
	version, ok, err := installedIncusPackage()
	if err != nil || !ok || version != from {
		return p, fmt.Errorf("old Incus dpkg mismatch, want %s got %s: %v", from, version, err)
	}
	client, err := installedClientVersion()
	if err != nil || client != from {
		return p, errors.New("native client and server package versions differ")
	}
	if exec.Command("systemctl", "is-active", "--quiet", "incus").Run() != nil {
		return p, errors.New("old native daemon inactive")
	}
	if _, err := os.Lstat("/var/lib/incus/unix.socket.user"); err != nil {
		return p, err
	}
	projects, err := migrationInventory()
	if err != nil {
		return p, err
	}
	oldNames, err := nativePackageNames(from)
	if err != nil {
		return p, err
	}
	newNames, err := nativePackageNames(to)
	if err != nil {
		return p, err
	}
	if !slices.Equal(oldNames, newNames) {
		return p, errors.New("cross-packaging native migration not supported")
	}
	if err = verifyNativePackageVersions(oldNames, from); err != nil {
		return p, err
	}
	p.Packages = oldNames
	changes, simulation, err := exactAptMigration(from, to, oldNames)
	if err != nil {
		return p, err
	}
	host, err := migrationHostDigest()
	if err != nil {
		return p, err
	}
	p.HostDigest = host
	p.Projects = projects
	p.AptChanges = changes
	h := sha256.Sum256([]byte("boxweave-native-migration-v1\n" + host + "\n" + from + "\n" + to +
		"\n" + strings.Join(projects, "\n") + "\n" + strings.Join(p.Packages, "\n") + "\n" + simulation))
	p.PlanSHA256 = hex.EncodeToString(h[:])
	p.Mode = "explicit-migration-approval-required"
	return p, nil
}
