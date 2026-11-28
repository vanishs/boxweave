package install

// Shared bounded conntrack readback and report primitives. Callers must
// independently prove native scope and authorization before any effect.
import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os/exec"
	"strconv"
	"strings"
)

type NativePeerConntrackReport struct {
	NativeDigestSHA256     string `json:"native_digest_sha256"`
	PostNativeDigestSHA256 string `json:"post_native_digest_sha256,omitempty"`
	VerifiedNICs           int    `json:"verified_nics,omitempty"`
	KernelVerified         bool   `json:"kernel_verified,omitempty"`
	PairIPv4               string `json:"pair_ipv4"`
	PairIPv6               string `json:"pair_ipv6"`
	ProbedScopes           int    `json:"probed_scopes"`
	DeletedScopes          int    `json:"deleted_scopes"`
	// No readiness claim is made before an independently checked ACL
	// native per-NIC kernel state and the full controller transaction.
	ReadyForPeer bool `json:"ready_for_peer"`
}

type nativePeerConntrackRun func(context.Context, ...string) ([]byte, error)

func nativePeerConntrackOS(ctx context.Context, args ...string) ([]byte, error) {
	// Deliberately absolute, never PATH resolution or any shell arguments.
	cmd := exec.CommandContext(ctx, "/usr/sbin/conntrack", args...)
	cmd.Env = []string{"LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
	return cmd.CombinedOutput()
}

func nativePeerConntrackAddr(raw string, ipv6 bool) (netip.Addr, error) {
	a, err := netip.ParseAddr(raw)
	if err != nil || !a.IsGlobalUnicast() || a.Is4In6() ||
		(ipv6 && !a.Is6()) || (!ipv6 && !a.Is4()) {
		return netip.Addr{}, errors.New("untrusted static native peer address")
	}
	return a, nil
}

// A successful process exit is NOT evidence that an empty stdout means
// zero kernel connections. Require an explicit conntrack zero diagnostic or
// original source/destination tuples. Unexpected, truncated or contradictory
// output preserves UNKNOWN; never issue a blind -D after an ambiguous read.
func nativePeerConntrackList(ctx context.Context, run nativePeerConntrackRun,
	family, source, target string) (bool, error) {
	if run == nil {
		return false, errors.New("missing bounded conntrack reader")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	output, cmdErr := run(ctx, "-L", "-f", family, "-s", source, "-d", target)
	if err := ctx.Err(); err != nil {
		return false, err
	}
	found, zero, nonzero := false, false, false
	for _, raw := range strings.Split(string(output), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		pos := strings.Index(line, " flow entries have been shown")
		if pos < 0 {
			pos = strings.Index(line, " flow entries shown")
		}
		if pos >= 0 {
			prefix := strings.Fields(strings.TrimSpace(line[:pos]))
			if len(prefix) == 0 {
				return false, errors.New("unparseable conntrack summary")
			}
			count, err := strconv.Atoi(prefix[len(prefix)-1])
			if err != nil || count < 0 {
				return false, errors.New("unparseable conntrack entry count")
			}
			if count == 0 {
				zero = true
			} else {
				nonzero = true
			}
			continue
		}
		firstSource, firstTarget := "", ""
		for _, f := range strings.Fields(line) {
			if strings.HasPrefix(f, "src=") && firstSource == "" {
				firstSource = strings.TrimPrefix(f, "src=")
			}
			if strings.HasPrefix(f, "dst=") && firstTarget == "" {
				firstTarget = strings.TrimPrefix(f, "dst=")
			}
		}
		if firstSource != source || firstTarget != target {
			return false, errors.New("conntrack inventory contains unverified tuple or diagnostic")
		}
		found = true
	}
	if (zero && found) || (zero && nonzero) {
		return false, errors.New("contradictory conntrack inventory readback")
	}
	if cmdErr != nil {
		if zero && !found && !nonzero {
			// Some conntrack-tools builds return code 1 for a filtered zero.
			return false, nil
		}
		return false, fmt.Errorf("bounded conntrack inventory uncertain: %w", cmdErr)
	}
	if found {
		return true, nil
	}
	if zero && !nonzero {
		return false, nil
	}
	return false, errors.New("conntrack readback lacks explicit zero or verified original tuple")
}
