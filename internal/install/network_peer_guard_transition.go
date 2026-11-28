package install

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
)

// NativePeerGuardPlan is a deterministic PURE nftables policy proposal.
// Install/Replace must NOT be executed from MCP, a guest, or by any caller
// that has not independently proven ownership and guarded boot/recovery.
// An nftables accept in one bridge chain does not bypass another chain's
// drop. The terminal drop is intentionally restricted to managed-to-managed
// traffic so unrelated host/forward traffic is unaffected.
type NativePeerGuardPlan struct {
	Table, Install, Replace, ExpectedKernelJSON, DigestSHA256 string
}

// nativePeerGuardReplaceOutcome is an *ephemeral* effect witness, NEVER a
// completed Peer original or a durable replacement for the protected root
// journal. A lost process ACK may mean the nft batch committed; the caller
// must persist UNKNOWN and hold all claims, never replay the write.
type nativePeerGuardReplaceOutcome struct {
	Attempted      bool
	KernelVerified bool
	BeforePlanSHA  string
	AfterPlanSHA   string
}

// nativePeerGuardValidateOneDirectionDelta guarantees that one guarded
// transaction can add OR delete exactly one pinned four-rule dual-stack
// direction. All other C/D grants AND the four whole-bridge IPv4/IPv6/link-
// local deny/ND rules must be bit-identical to the preimage. This is an
// additional guard; only an independent root original/SQLite preflight can
// prove the selected delta was actually authorized by the Actor.
func nativePeerGuardValidateOneDirectionDelta(before, after NativePeerGuardPlan) error {
	if !nativePeerGuardTablePattern.MatchString(before.Table) ||
		before.Table != after.Table ||
		nativePeerGuardDigest(before) != before.DigestSHA256 ||
		nativePeerGuardDigest(after) != after.DigestSHA256 ||
		before.ExpectedKernelJSON == after.ExpectedKernelJSON {
		return errors.New("untrusted or unchanged exact bridge guard transition")
	}
	groups := func(p NativePeerGuardPlan) (map[string]bool, [][]any, error) {
		var rules [][]any
		if err := json.Unmarshal([]byte(p.ExpectedKernelJSON), &rules); err != nil ||
			len(rules) < 4 || (len(rules)-4)%4 != 0 || len(rules) > 256 {
			return nil, nil, errors.New("unverified guard direction groups")
		}
		lines := strings.Split(strings.TrimSuffix(p.Replace, "\n"), "\n")
		if len(lines) != len(rules)+1 ||
			lines[0] != "flush chain bridge "+p.Table+" protect" {
			return nil, nil, errors.New("root nft batch is not one bounded exact chain replace")
		}
		for _, line := range lines[1:] {
			if !strings.HasPrefix(line, "add rule bridge "+p.Table+" protect meta ibrname \"") ||
				strings.ContainsAny(line, ";\r") || len(line) > 8192 {
				return nil, nil, errors.New("root nft batch contains unbounded or foreign command")
			}
		}
		groups := map[string]bool{}
		for i := 0; i < len(rules)-4; i += 4 {
			bytes, err := json.Marshal(rules[i : i+4])
			if err != nil {
				return nil, nil, err
			}
			key := string(bytes)
			if groups[key] {
				return nil, nil, errors.New("duplicate native guarded pair direction")
			}
			groups[key] = true
		}
		return groups, rules[len(rules)-4:], nil
	}
	left, baseLeft, err := groups(before)
	if err != nil {
		return err
	}
	right, baseRight, err := groups(after)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(baseLeft, baseRight) {
		return errors.New("whole bridge default DROP/neighbor policy mutated with a Peer direction")
	}
	var removed, added int
	for k := range left {
		if !right[k] {
			removed++
		}
	}
	for k := range right {
		if !left[k] {
			added++
		}
	}
	if (removed != 1 || added != 0) && (removed != 0 || added != 1) {
		return fmt.Errorf("unsafe multi-peer or re-addressed guard mutation: +%d -%d", added, removed)
	}
	return nil
}

type nativePeerGuardWitness func(context.Context, NativePeerGuardPlan) error
type nativePeerGuardBatch func(context.Context, string) error

// nativePeerGuardReplaceOnce is the package-private, non-authorizing kernel
// transition seam. It MUST execute only after the Root dispatcher has
// fsynced the exact original's intent, validated the full native+SQLite
// inventory, live sysfs ports, protected actor and locked all dependencies.
// A successful pre/post witness MUST include this module's exact native
// nft JSON readback; a caller must additionally re-verify unrelated Box
// identities and connection/Forward continuity. This function has no MCP
// entrypoint and is NOT called by the RC13 installed Helper.
//
// A failed or ambiguous batch returns Attempted=true but never Verified.
// Do not roll back or retry an uncertain native write here: only a later
// explicit journaled root reconciliation can decide the next safe action.
func nativePeerGuardReplaceOnce(ctx context.Context, before, after NativePeerGuardPlan,
	witness nativePeerGuardWitness, write nativePeerGuardBatch) (nativePeerGuardReplaceOutcome, error) {
	out := nativePeerGuardReplaceOutcome{
		BeforePlanSHA: before.DigestSHA256, AfterPlanSHA: after.DigestSHA256,
	}
	if witness == nil || write == nil {
		return out, errors.New("no root-only guard witness or batch executor")
	}
	if err := nativePeerGuardValidateOneDirectionDelta(before, after); err != nil {
		return out, err
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if err := witness(ctx, before); err != nil {
		return out, fmt.Errorf("original kernel guard preimage not verified: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	// This call marks the irreversible uncertainty boundary. Even an error
	// from the kernel batch may mean it applied before the ACK was lost.
	out.Attempted = true
	if err := write(ctx, after.Replace); err != nil {
		return out, fmt.Errorf("root nft atomic batch result UNKNOWN, do not retry: %w", err)
	}
	if err := witness(ctx, after); err != nil {
		return out, fmt.Errorf("root nft post-ACK kernel result UNKNOWN, do not retry: %w", err)
	}
	out.KernelVerified = true
	return out, nil
}

// Only an independently approved root-original dispatcher may call this
// fixed local primitive. No shell, generic nft flags, raw Host paths, CLI
// or public MCP tool is ever exposed.
func nativePeerGuardFixedNftBatch(ctx context.Context, script string) error {
	if os.Geteuid() != 0 {
		return errors.New("nft native batch requires trusted local root")
	}
	if len(script) == 0 || len(script) > 256*8192 {
		return errors.New("missing or unbounded protected nft batch")
	}
	cmd := exec.CommandContext(ctx, "/usr/sbin/nft", "-f", "-")
	cmd.Env = []string{"LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
	cmd.Stdin = strings.NewReader(script)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("native nft batch outcome uncertain: %w", err)
	}
	return nil
}
