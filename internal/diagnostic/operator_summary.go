package diagnostic

// OperatorSummary is additive presentation only, never recovery authority.
// Existing issue IDs and legacy readiness/exit contracts remain unchanged.
type OperatorSummary struct {
	Status           string   `json:"status"`
	Authority        string   `json:"authority"`
	ConfirmedIssues  int      `json:"confirmed_issues"`
	UnverifiedIssues int      `json:"unverified_issues"`
	PendingIssues    int      `json:"pending_issues"`
	CoverageLimits   int      `json:"coverage_limits"`
	Maintenance      string   `json:"maintenance_readiness"`
	Next             []string `json:"next_actions"`
}

func Summarize(r Report, uid int) *OperatorSummary {
	o := &OperatorSummary{Status: "HEALTHY_OBSERVED", Authority: "LOCAL_ROOT", Maintenance: "SEPARATE_REVIEW_REQUIRED", Next: []string{}}
	if uid != 0 {
		o.Authority = "RESTRICTED_OBSERVER"
	}
	for _, i := range r.Issues {
		switch i.Classification {
		case "HISTORICAL":
			continue
		case "COVERAGE_LIMIT":
			o.CoverageLimits++
		case "IN_PROGRESS":
			o.PendingIssues++
		case "CONFIRMED":
			o.ConfirmedIssues++
		default:
			o.UnverifiedIssues++
		}
	}
	switch {
	case r.Installation.Mode == "CONFLICT" || r.Installation.ServiceUnit == "FAILED" || r.Installation.ServiceUnit == "INACTIVE":
		o.Status = "BLOCKED"
	case o.ConfirmedIssues > 0:
		o.Status = "DEGRADED"
	case o.UnverifiedIssues > 0 || o.CoverageLimits > 0 || r.Database.State != "OBSERVED" || r.Native.State != "OBSERVED" || r.MCP.State != "OBSERVED" || r.ForwardPolicy.State != "OBSERVED" || r.ForwardDevices.State != "OBSERVED":
		o.Status = "UNVERIFIED"
	}
	if uid != 0 && (o.CoverageLimits > 0 || r.RootPolicy.State != "OBSERVED") {
		o.Next = append(o.Next, "Run sudo boxweave doctor --summary for root-only coverage; missing authority is not a fault.")
	}
	if o.Status != "HEALTHY_OBSERVED" {
		o.Next = append(o.Next, "Inspect current issue IDs; use boxweave recover --plan --issue ID or local root --guided for supported originals. Preserve UNKNOWN.")
	}
	if o.PendingIssues > 0 {
		o.Next = append(o.Next, "Pending work or deferred retirement is not itself a fault; inspect original claims and capacity.")
	}
	o.Next = append(o.Next, "For lifecycle readiness use sudo boxweave doctor --maintenance --summary, then the exact uninstall.sh plan or install-status. Purge needs its own verified plan.")
	return o
}
