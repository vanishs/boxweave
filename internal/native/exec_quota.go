package native

import (
	"strings"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

// A reserved 16 MiB margin is sufficient to create small Incus recorded
// stdout/stderr log files before an Exec. This is a best-effort read-only
// preflight, NOT a reservation: concurrent guest writes can exhaust the
// volume between the query and ExecInstance. The native terminal verifier
// remains authoritative.
const minExecSpoolHeadroomBytes uint64 = 16 << 20

// ErrNativeExecQuotaLow denotes a proven BEFORE-dispatch quota preflight
// refusal. It is never synthesized after an Incus Exec request has been sent.
var ErrNativeExecQuotaLow = errNativeExecQuotaLow("native Exec recorded-output spool quota headroom insufficient")

type errNativeExecQuotaLow string

func (e errNativeExecQuotaLow) Error() string { return string(e) }

func execSpoolHeadroomLow(usage *api.StorageVolumeStateUsage) bool {
	if usage == nil || usage.Total <= 0 {
		return false // Unknown is not proof of low quota.
	}
	maxBytes := uint64(usage.Total)
	return usage.Used >= maxBytes ||
		maxBytes-usage.Used < minExecSpoolHeadroomBytes
}

// preflightExecSpoolHeadroom queries only the exact selected root volume via
// the already-confined project SDK client. Never shell out, guess mount paths,
// or use filesystem-wide df: Btrfs qgroups count the instance and its sibling
// Incus recorded-Exec spool together. An unavailable state API retains the
// previous Exec path, with conservative postcondition verification.
func preflightExecSpoolHeadroom(c incus.InstanceServer, inst *api.Instance, name string) error {
	if c == nil || inst == nil || name == "" {
		return nil
	}
	pool := ""
	for _, d := range inst.ExpandedDevices {
		if d["type"] == "disk" && d["path"] == "/" {
			pool = d["pool"]
			break
		}
	}
	if pool == "" {
		return nil
	}
	state, err := c.GetStoragePoolVolumeState(pool, "container", name)
	if err != nil || state == nil {
		return nil
	}
	if execSpoolHeadroomLow(state.Usage) {
		return ErrNativeExecQuotaLow
	}
	return nil
}

// ExecRecordingQuotaPreStartFailure accepts only the source-verified Incus
// recorded-output OpenFile ENOSPC/EDQUOT failure: Incus opens BOTH log files
// before calling inst.Exec, and has not set a Guest 'return' or 'output'
// metadata on this path. The exact original operation log basename must
// appear in the trusted final native error. No opaque error, guest exit, or
// partial record is converted to a pre-start failure.
func ExecRecordingQuotaPreStartFailure(op *api.Operation, name, nativeID string) bool {
	if op == nil || op.ID != nativeID || op.StatusCode != api.Failure ||
		!safeName.MatchString(name) || len(nativeID) != 36 ||
		strings.ContainsAny(nativeID, "/\\\r\n") {
		return false
	}
	if op.Metadata != nil {
		if _, ok := op.Metadata["return"]; ok {
			return false
		}
		if _, ok := op.Metadata["output"]; ok {
			return false
		}
	}
	if strings.ContainsAny(op.Err, "\r\n") {
		return false
	}
	for _, stream := range []string{"stdout", "stderr"} {
		base := "exec_" + nativeID + "." + stream
		for _, errText := range []string{"disk quota exceeded", "no space left on device"} {
			if strings.HasSuffix(op.Err, base+": "+errText) &&
				(strings.Contains(op.Err, "open "+base+": ") ||
					strings.Contains(op.Err, "open /")) {
				return true
			}
		}
	}
	return false
}
