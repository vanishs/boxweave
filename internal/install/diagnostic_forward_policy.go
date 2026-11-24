package install

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/vanishs/boxweave/internal/native"
)

// Read only the fixed Forward policy from the verified running controller.
// Never expose raw environment, bind rules or other process secrets.
func ObserveDiagnosticForwardPolicy(ctx context.Context, p Plan, identity DiagnosticInstallation) (string, bool) {
	if identity.Runtime != "VERIFIED" || identity.InstalledIdentity != "VERIFIED" {
		return "unverified", false
	}
	svc := diagnosticSystemd(ctx, "boxweave.service", "MainPID", "ActiveState", "FragmentPath")
	if svc == nil || svc["ActiveState"] != "active" || svc["FragmentPath"] != p.UnitPath {
		return "unverified", false
	}
	pid, e := strconv.Atoi(svc["MainPID"])
	if e != nil || pid < 2 {
		return "unverified", false
	}
	uid, _, e := ServiceIdentity(p)
	if e != nil {
		return "unverified", false
	}
	proc := filepath.Join("/proc", strconv.Itoa(pid))
	before, e := os.Stat(proc)
	if e != nil {
		return "unverified", false
	}
	own, ok := before.Sys().(*syscall.Stat_t)
	if !ok || own.Uid != uint32(uid) {
		return "unverified", false
	}
	exe, e := os.Stat(filepath.Join(proc, "exe"))
	if e != nil {
		return "unverified", false
	}
	expected, e := os.Stat(p.BinaryPath)
	if e != nil || !os.SameFile(exe, expected) {
		return "unverified", false
	}
	f, e := os.Open(filepath.Join(proc, "environ"))
	if e != nil {
		return "unverified", false
	}
	b, e := io.ReadAll(io.LimitReader(f, 65537))
	f.Close()
	if e != nil || len(b) > 65536 || len(b) > 0 && b[len(b)-1] != 0 {
		return "unverified", false
	}
	raw := ""
	found := false
	for _, v := range strings.Split(string(b), "\x00") {
		k, value, ok := strings.Cut(v, "=")
		if ok && k == "BOXWEAVE_FORWARD_EXTERNAL_ALLOW" {
			if found {
				return "unverified", false
			}
			found = true
			raw = value
		}
	}
	after, e := os.Stat(proc)
	actual, e2 := os.Stat(filepath.Join(proc, "exe"))
	latest := diagnosticSystemd(ctx, "boxweave.service", "MainPID", "ActiveState", "FragmentPath")
	if e != nil || e2 != nil || !os.SameFile(before, after) || !os.SameFile(exe, actual) || latest == nil || latest["MainPID"] != svc["MainPID"] || latest["ActiveState"] != "active" || latest["FragmentPath"] != p.UnitPath {
		return "unverified", false
	}
	return (native.Reader{ExternalForwardAllow: raw}).ExternalForwardPolicyMode(), true
}
