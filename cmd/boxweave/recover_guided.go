package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/vanishs/boxweave/internal/diagnostic"
	"github.com/vanishs/boxweave/internal/state"
)

// Guided recovery is a terminal-only front-end to the same reviewed D4
// transaction. stdout is the result JSON; stderr prints the plan to save
// with the exact idempotency key for an uncertain acknowledgement.
func recoverGuidedCLI(args []string) error {
	f := flag.NewFlagSet("recover --guided", flag.ContinueOnError)
	guided := f.Bool("guided", false, "review and approve one supported local repair")
	issue := f.String("issue", "", "exact current Doctor issue ID")
	project := f.String("project", "", "exact installed confined project")
	reviewer := f.String("reviewer", "local-root", "recorded local operator identity")
	key := f.String("recovery-key", "", "separate durable idempotency key")
	if e := f.Parse(args); e != nil {
		return e
	}
	if !*guided || f.NArg() != 0 || os.Geteuid() != 0 || !diagnostic.ValidIssueID(*issue) {
		return errors.New("guided recovery requires local root and an exact Doctor issue")
	}
	a, e := resolveDoctorScope(fixedRecoveryArgs(*project))
	if e != nil || !state.SafeDiagnosticProject(a.Project) {
		return state.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	p := collectRecoverPlan(ctx, a, *issue)
	cancel()
	enc := json.NewEncoder(os.Stderr)
	enc.SetIndent("", "  ")
	if e = enc.Encode(p); e != nil {
		return e
	}
	if p.Status != "REVIEW_READY" || !p.ApplyAvailable {
		// Still return stable machine JSON for blocked, absent and historical
		// cases. No terminal, no maintenance lock, no effect.
		if e = json.NewEncoder(os.Stdout).Encode(p); e != nil {
			return e
		}
		if p.Status == "HISTORICAL" || p.Status == "NOT_REQUIRED" {
			return nil
		}
		return errors.New("recovery blocked: inspect plan route and preconditions")
	}
	if *key == "" {
		*key = "guided-" + p.ReviewSHA256
	}
	q := reviewedApplyRequest{Plan: p, Reviewer: *reviewer, Key: *key}
	if !validReviewedEnvelope(q) {
		return state.ErrInvalid
	}
	tty, e := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if e != nil {
		return errors.New("local terminal approval required; use existing --apply API for saved plan")
	}
	defer tty.Close()
	fmt.Fprintf(tty, "Recovery key: %s; reviewer: %s. Save exact plan above for lost acknowledgement.\nType RECOVER %s to approve: ", q.Key, q.Reviewer, p.ReviewSHA256)
	if e = readGuidedApproval(tty, p.ReviewSHA256); e != nil {
		return e
	}
	// Apply independently checks expiry and re-observes every witness.
	return applyReviewedEnvelope(q)
}

func readGuidedApproval(r io.Reader, sha string) error {
	line, e := bufio.NewReader(io.LimitReader(r, 256)).ReadString('\n')
	if e != nil || strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r") != "RECOVER "+sha {
		return errors.New("OPERATOR_APPROVAL_REQUIRED: recovery not approved")
	}
	return nil
}
