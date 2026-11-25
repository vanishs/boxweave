package install

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// ScopedRetirementPolicy is an exact approved policy witness, not a
// runtime hardening endorsement. Only the fixed, reviewed policy qualifies.
type ScopedRetirementPolicy struct {
	Path string `json:"path"`
	SHA  string `json:"sha256"`
	Mode int    `json:"mode"`
	Kind string `json:"kind"`
}

func knownScopedRetirementPolicy(p ScopedRetirementPolicy) bool {
	return p == (ScopedRetirementPolicy{knownLXC255PolicyPath, knownLXC255PolicySHA, 0644, "EXACT_LXC255_SCOPED_RETIREMENT"})
}

func mappedRoot(raw string) bool {
	rows := strings.Split(strings.TrimSpace(raw), "\n")
	if len(rows) == 0 {
		return false
	}
	for i, row := range rows {
		f := strings.Fields(row)
		if len(f) != 3 {
			return false
		}
		vals := make([]uint64, 3)
		for j, v := range f {
			n, e := strconv.ParseUint(v, 10, 32)
			if e != nil {
				return false
			}
			vals[j] = n
		}
		if vals[2] == 0 || i == 0 && (vals[0] != 0 || vals[1] == 0) {
			return false
		}
	}
	return true
}

// Called only by the local root approved drain worker, after the exact
// installed static contract, durable original, native and ledger proof.
func VerifyScopedRetirementPolicy(ctx context.Context, unit string, p ScopedRetirementPolicy) bool {
	if !knownScopedRetirementPolicy(p) || !knownLXC255DropIn(p.Path) || os.Geteuid() != 0 {
		return false
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/systemd-detect-virt", "--container")
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C"}
	b, e := cmd.Output()
	if e != nil || strings.TrimSpace(string(b)) != "lxc" {
		return false
	}
	for _, name := range []string{"uid", "gid"} {
		b, e := os.ReadFile("/proc/self/" + name + "_map")
		if e != nil || !mappedRoot(string(b)) {
			return false
		}
	}
	for _, dir := range []string{"/", "/run", "/run/systemd", "/run/systemd/system", "/run/systemd/system/service.d"} {
		st, e := os.Lstat(dir)
		if e != nil || !st.IsDir() || st.Mode().Perm()&0022 != 0 {
			return false
		}
		// diagnosticReadFile independently requires root-owned no-symlink file.
		own, ok := st.Sys().(*syscall.Stat_t)
		if !ok || own.Uid != 0 {
			return false
		}
	}
	props := diagnosticSystemd(ctx, unit, "LoadState", "FragmentPath", "DropInPaths", "NeedDaemonReload", "Type", "User", "Group", "KillMode", "UMask", "NoNewPrivileges", "ProtectSystem", "ProtectHome", "PrivateTmp", "CapabilityBoundingSet", "AmbientCapabilities", "RequiredBy", "BoundBy", "RequisiteOf", "ConsistsOf", "PropagatesStopTo", "ExecStart", "ExecStartPre", "ExecStartPost", "ExecCondition", "ExecStop", "ExecStopPost", "ExecReload")
	if props["LoadState"] != "loaded" || props["DropInPaths"] != p.Path || props["NeedDaemonReload"] != "no" || props["NoNewPrivileges"] != "no" || props["KillMode"] != "control-group" || props["UMask"] != "0077" || props["Group"] != "" {
		return false
	}
	switch unit {
	case "boxweave.service":
		if props["FragmentPath"] != "/etc/systemd/system/boxweave.service" || props["Type"] != "simple" || props["User"] != "boxweave" || props["ProtectSystem"] != "strict" || props["ProtectHome"] != "yes" || props["PrivateTmp"] != "yes" || props["CapabilityBoundingSet"] != "" || props["AmbientCapabilities"] != "" {
			return false
		}
	case scopedGuardUnitName:
		if props["FragmentPath"] != scopedGuardUnitPath || props["Type"] != "oneshot" || props["User"] != "" {
			return false
		}
	case scopedLifecycleUnitName:
		if props["FragmentPath"] != scopedLifecycleUnitPath || props["Type"] != "simple" || props["User"] != "" {
			return false
		}
	default:
		return false
	}
	raw, _, e := diagnosticReadFile(props["FragmentPath"], 0, false, 65536)
	if e != nil {
		return false
	}
	starts := []string{}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "ExecStart=") {
			starts = append(starts, strings.TrimPrefix(line, "ExecStart="))
		}
	}
	if len(starts) != 1 || strings.Count(props["ExecStart"], "argv[]=") != 1 || !strings.Contains(props["ExecStart"], "{ path=/usr/local/bin/boxweave ; argv[]="+starts[0]+" ; ignore_errors=no ;") {
		return false
	}
	for _, key := range []string{"ExecStartPre", "ExecStartPost", "ExecCondition", "ExecStop", "ExecStopPost", "ExecReload"} {
		if props[key] != "" {
			return false
		}
	}
	for _, k := range []string{"RequiredBy", "BoundBy", "RequisiteOf", "ConsistsOf", "PropagatesStopTo"} {
		v, ok := props[k]
		if !ok {
			return false
		}
		for _, ref := range strings.Fields(v) {
			if unit != scopedGuardUnitName || (ref != "incus.service" && ref != scopedLifecycleUnitName) {
				return false
			}
		}
	}
	return knownLXC255DropIn(p.Path)
}
