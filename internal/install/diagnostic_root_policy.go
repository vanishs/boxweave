package install

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/vanishs/boxweave/internal/networklimits"
)

type DiagnosticRootPolicy struct {
	State                          string `json:"state"`
	Kernel                         string `json:"kernel_guard"`
	Sequence                       *int   `json:"sequence,omitempty"`
	ActiveDirections               *int   `json:"active_directions,omitempty"`
	HistoryLimit                   int    `json:"history_limit,omitempty"`
	NewGrantHistoryAvailable       *bool  `json:"new_grant_history_available,omitempty"`
	RemainingGrantEventsUpperBound *int   `json:"remaining_grant_events_upper_bound,omitempty"`
	GrantHistoryLevel              string `json:"grant_history_level,omitempty"`
	Conntrack                      string `json:"conntrack_executable"`
}

func diagnosticExecutableState(path string, uid uint32) string {
	st, e := os.Stat(path)
	if os.IsNotExist(e) {
		return "MISSING"
	}
	if e != nil {
		return "UNVERIFIED"
	}
	own, ok := st.Sys().(*syscall.Stat_t)
	if ok && own.Uid == uid && st.Mode().IsRegular() && st.Mode().Perm()&0111 != 0 && st.Mode().Perm()&0022 == 0 {
		return "PRESENT"
	}
	return "UNVERIFIED"
}
func verifyScopedPrerequisiteExecutables() error {
	for _, p := range []string{"/usr/sbin/ip", "/usr/sbin/bridge", "/usr/sbin/nft", "/usr/sbin/conntrack"} {
		if st := diagnosticExecutableState(p, 0); st != "PRESENT" {
			return fmt.Errorf("SCOPED_PREREQUISITE_%s: %s", st, p)
		}
	}
	return nil
}

// No Root lifecycle socket call, boot restore, ACL or conntrack mutation.
// An incomplete policy or unknown journal may be a live transition; never
// label the kernel healthy or recommend a restart solely from this read.
func ObserveDiagnosticRootPolicy(ctx context.Context, project string, identity DiagnosticInstallation) DiagnosticRootPolicy {
	out := DiagnosticRootPolicy{State: "UNVERIFIED", Kernel: "NOT_CHECKED", Conntrack: "NOT_CHECKED"}
	if identity.Mode == "LEGACY" {
		out.State = "NOT_APPLICABLE"
		return out
	}
	if identity.Candidate != "SCOPED_V2" {
		return out
	}
	out.HistoryLimit = networklimits.ScopedPeerHistoryLimit
	out.Conntrack = diagnosticExecutableState("/usr/sbin/conntrack", 0)

	if os.Geteuid() != 0 {
		out.State = "ROOT_AUTHORITY_REQUIRED"
		return out
	}
	if identity.Mode != "SCOPED_V2" || identity.RootProof != "VERIFIED" || ctx.Err() != nil {
		return out
	}
	return observeScopedDiagnosticRootPolicy(ctx, project, out)
}

// Call only after independent root journal/scope proof; runtime need not be active.
func observeScopedDiagnosticRootPolicy(ctx context.Context, project string, out DiagnosticRootPolicy) DiagnosticRootPolicy {
	r, initial, e := nativePeerFreshVerifiedPlan()
	if e != nil || r.Project != project {
		return out
	}
	current, last, e := scopedPeerBootCurrent(nativePeerFreshJournalDir, 0, r, initial, "")
	if e != nil || ctx.Err() != nil {
		out.State = "JOURNAL_INCOMPLETE_OR_UNVERIFIED"
		return out
	}
	seq, active := last.Sequence, len(last.Active)
	available := networklimits.ScopedPeerHistoryCanAppend(seq, active, true)
	out.State = "OBSERVED"
	out.Sequence = &seq
	out.ActiveDirections = &active
	out.NewGrantHistoryAvailable = &available
	remaining := networklimits.ScopedGrantHistoryRemaining(seq, active)
	out.RemainingGrantEventsUpperBound = &remaining
	out.GrantHistoryLevel = networklimits.CapacityLevel(networklimits.ScopedPeerHistoryLimit/2, remaining)
	out.Kernel = "UNVERIFIED"
	if NativePeerGuardKernelReadback(ctx, current) == nil {
		again, latest, e := scopedPeerBootCurrent(nativePeerFreshJournalDir, 0, r, initial, "")
		if e == nil && again == current && latest.Sequence == seq && latest.FinalSHA256 == last.FinalSHA256 && ctx.Err() == nil {
			out.Kernel = "OBSERVED"
		}
	}
	return out
}
