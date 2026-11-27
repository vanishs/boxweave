package install

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// The production paths are fixed by safe(Plan). All actions are explicit,
// host-local and root-only. No Incus or unrelated service restart is attempted.
const serviceName = "boxweave.service"

func localApproval(p Plan, approved bool) error {
	if !approved || os.Geteuid() != 0 {
		return errors.New("explicit host-local root authorization required")
	}
	return safe(p)
}
func runLocal(name string, args ...string) error {
	c := exec.Command(name, args...)
	if err := c.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", filepath.Base(name), err)
	}
	return nil
}

// A service can be installed before any root disk, private NIC or Guest
// image exists. Never call the full Box-create doctor from installation.
func doctorAsService(p Plan, binary string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "runuser", "-u", p.User, "--", "env",
		"-u", "INCUS_CONF", "-u", "INCUS_CONFIG_DIR", "-u", "XDG_CONFIG_HOME",
		"HOME=/home/boxweave", "XDG_CONFIG_HOME=/home/boxweave/.config",
		binary, "status", "-db", filepath.Join(p.StateDir, "ledger.sqlite"),
		"-project", p.Project, "-incus-socket", p.Socket)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("restricted service preflight unavailable: %w", err)
	}
	var status struct {
		MCPReady bool `json:"mcp_ready"`
		Native   struct {
			ServiceReady bool   `json:"service_ready"`
			ErrorClass   string `json:"error_class"`
		} `json:"native"`
	}
	if err := json.Unmarshal(out, &status); err != nil {
		return fmt.Errorf("service preflight returned invalid status: %w", err)
	}
	if !status.MCPReady || !status.Native.ServiceReady {
		return fmt.Errorf("service preflight blocked: MCP schema ready=%t; Incus runtime ready=%t (%s)",
			status.MCPReady, status.Native.ServiceReady, status.Native.ErrorClass)
	}
	return nil
}
func serviceActive() error {
	return runLocal("systemctl", "is-active", "--quiet", serviceName)
}
func writeManifest(p Plan, m Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err = atomicWrite(manifestPath(p), append(b, '\n'), 0600); err != nil {
		return err
	}
	uid, gid, err := ServiceIdentity(p)
	if err != nil {
		return err
	}
	return os.Chown(manifestPath(p), uid, gid)
}

// A staged installation is NOT yet active or accepted.
func Activate(p Plan, approved bool) (Manifest, error) {
	if err := localApproval(p, approved); err != nil {
		return Manifest{}, err
	}
	m, err := Status(p)
	if err != nil {
		return m, err
	}
	if m.ScopedPeerMode != nativePeerFreshMode || m.RetirementUnitSHA256 != "" || m.RetirementBridge != "" || m.RetirementACL != "" || m.RetirementNativeReviewSHA != "" {
		return m, errors.New("scoped fresh activation required; legacy installation unsupported")
	}
	if err = doctorAsService(p, p.BinaryPath); err != nil {
		return m, err
	}
	if m.RestartRequired {
		if err = runLocal("systemctl", "daemon-reload"); err != nil {
			return m, err
		}
		if err = activateScopedGuard(); err != nil {
			return m, err
		}
		if err = runLocal("systemctl", "enable", "--now", serviceName); err != nil {
			return m, err
		}
	}
	if err = waitInstalledHealthy(p, m); err != nil {
		return m, fmt.Errorf("activated service is not stable: %w", err)
	}
	m.RestartRequired = false
	if err = writeManifest(p, m); err != nil {
		return m, err
	}
	return Status(p)
}
