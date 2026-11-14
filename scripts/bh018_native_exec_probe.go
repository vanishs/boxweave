//go:build ignore

// This is intentionally a characterization test, not a generic claim that
// missing return means startup failed: actual Incus 7.6 Failure operations
// returned metadata.return=127/126. The checks below were corrected after a
// real 21-case native negative run retained all original UUIDs and errors.
// BH-018 direct official Incus SDK native-operation probe for an exclusively
// disposable CI guest. Not a shipped command, an MCP tool, or a recovery path.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

type witness struct {
	Case             string `json:"case"`
	Round            int    `json:"round"`
	OriginalNativeID string `json:"original_native_operation_id,omitempty"`
	Status           string `json:"status"`
	StatusCode       int    `json:"status_code"`
	Error            string `json:"error,omitempty"`
	Return           any    `json:"return,omitempty"`
	HasReturn        bool   `json:"has_return"`
	HasOutput        bool   `json:"has_output"`
	DurationMS       int64  `json:"duration_ms"`
}

type scenario struct {
	name         string
	argv         []string
	cwd          string
	expectStatus string
	expectReturn *int
}

func pint(n int) *int { return &n }

func testOne(ctx context.Context, c incus.InstanceServer, instance string, tc scenario, round int) error {
	began := time.Now()
	op, err := c.ExecInstance(instance, api.InstanceExecPost{
		Command: tc.argv, Cwd: tc.cwd,
		RecordOutput: true, WaitForWS: false,
	}, nil)
	w := witness{Case: tc.name, Round: round, Status: "NO_NATIVE_RECEIPT"}
	if err != nil {
		w.Error = safe(err.Error())
		return emit(w, fmt.Errorf("%s: dispatch returned error without native receipt: %w", tc.name, err))
	}
	w.OriginalNativeID = op.Get().ID
	if w.OriginalNativeID == "" {
		return emit(w, fmt.Errorf("%s: empty native operation ID", tc.name))
	}
	var final *api.Operation
	for time.Since(began) < 25*time.Second {
		value, _, err := c.GetOperation(w.OriginalNativeID)
		if err != nil {
			w.Error = safe(err.Error())
			return emit(w, fmt.Errorf("%s: native operation read failed: %w", tc.name, err))
		}
		if value.StatusCode.IsFinal() {
			final = value
			break
		}
		select {
		case <-ctx.Done():
			return emit(w, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	if final == nil {
		return emit(w, fmt.Errorf("%s: native status was not final", tc.name))
	}
	w.Status = final.Status
	w.StatusCode = int(final.StatusCode)
	w.Error = safe(final.Err)
	w.Return, w.HasReturn = final.Metadata["return"]
	_, w.HasOutput = final.Metadata["output"]
	w.DurationMS = time.Since(began).Milliseconds()
	if tc.expectStatus == "success" && final.StatusCode != api.Success {
		return emit(w, fmt.Errorf("%s: wanted Incus native success, got %s", tc.name, final.Status))
	}
	if tc.expectStatus == "failure" && final.StatusCode == api.Success {
		return emit(w, fmt.Errorf("%s: native operation unexpectedly successful", tc.name))
	}
	if tc.expectReturn != nil {
		if !w.HasReturn || fmt.Sprint(w.Return) != fmt.Sprint(*tc.expectReturn) {
			return emit(w, fmt.Errorf("%s: expected native metadata.return=%d", tc.name, *tc.expectReturn))
		}
	} else if w.HasReturn {
		return emit(w, fmt.Errorf("%s: native startup failure unexpectedly recorded a guest exit code", tc.name))
	}
	if w.OriginalNativeID == "" || (tc.expectStatus == "failure" && strings.TrimSpace(w.Error) == "") {
		return emit(w, fmt.Errorf("%s: missing original receipt/error witness", tc.name))
	}
	return emit(w, nil)
}

func safe(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " ")
	if len(s) > 250 {
		return s[:250]
	}
	return s
}

func emit(w witness, err error) error {
	v, mErr := json.Marshal(w)
	if mErr != nil {
		return errors.Join(err, mErr)
	}
	fmt.Println(string(v))
	return err
}

func main() {
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" || os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "BH018_CI_BLOCKED: exclusively ephemeral GitHub Ubuntu runner with root access")
		os.Exit(81)
	}
	guest := os.Getenv("BH018_GUEST_NAME")
	if guest != "bh018-native-startup-1114" {
		fmt.Fprintln(os.Stderr, "BH018_CI_BLOCKED: unrecognized disposable guest identity")
		os.Exit(82)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	base, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		panic(err)
	}
	c := base.UseProject("default")
	tests := []scenario{
		{name: "true-exit0", argv: []string{"/bin/true"}, expectStatus: "success", expectReturn: pint(0)},
		{name: "false-exit1", argv: []string{"/bin/false"}, expectStatus: "success", expectReturn: pint(1)},
		{name: "missing-binary-native", argv: []string{"/usr/bin/bh018-definitely-missing-executable"}, expectStatus: "failure", expectReturn: pint(127)},
		{name: "non-executable-file-native", argv: []string{"/etc/hostname"}, expectStatus: "failure", expectReturn: pint(126)},
		{name: "missing-cwd-native", argv: []string{"/bin/true"}, cwd: "/bh018-definitely-missing-cwd", expectStatus: "success", expectReturn: pint(0)},
		{name: "shell-missing-exit127", argv: []string{"/bin/sh", "-c", "exec /usr/bin/bh018-definitely-missing-executable"}, expectStatus: "failure", expectReturn: pint(127)},
		{name: "timeout-wrapped-missing-exit127", argv: []string{"/usr/bin/timeout", "--signal=TERM", "--kill-after=5s", "--", "8.000s", "/usr/bin/bh018-definitely-missing-executable"}, expectStatus: "failure", expectReturn: pint(127)},
	}
	failures := 0
	for round := 1; round <= 3; round++ {
		for _, tc := range tests {
			if err := testOne(ctx, c, guest, tc, round); err != nil {
				fmt.Fprintf(os.Stderr, "BH018_NATIVE_CASE_FAILED round=%d case=%s: %v\n", round, tc.name, err)
				failures++
			}
		}
	}
	if failures != 0 {
		fmt.Fprintf(os.Stderr, "BH018_NATIVE_ERRORS=%d; no acceptance claim\n", failures)
		os.Exit(1)
	}
	fmt.Println("BH018_NATIVE_ORIGINAL_21_CASES=PASS")
}
