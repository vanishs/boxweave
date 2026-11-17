package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"reflect"
	"strings"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/state"
)

// NativePeerConntrackPlan is only a host-root READ-ONLY review snapshot.
// It is neither an authorization, a replayable write instruction nor an MCP Tool.
type NativePeerConntrackPlan struct {
	Project            string `json:"project"`
	Bridge             string `json:"bridge"`
	ACL                string `json:"acl"`
	SourceUUID         string `json:"source_uuid"`
	TargetUUID         string `json:"target_uuid"`
	HostSHA256         string `json:"host_sha256"`
	LedgerSHA256       string `json:"ledger_sha256"`
	NativeDigestSHA256 string `json:"native_digest_sha256"`
	PlanSHA256         string `json:"plan_sha256"`
	MutationAuthorized bool   `json:"mutation_authorized"`
}

func nativePeerConntrackSealedPlan(ledger state.NetworkAuditLedger, hostSHA, nativeSHA, src, dst string) (NativePeerConntrackPlan, error) {
	var zero NativePeerConntrackPlan
	for _, h := range []string{hostSHA, nativeSHA} {
		if len(h) != 64 || strings.ToLower(h) != h {
			return zero, errors.New("invalid host or native proof digest")
		}
		if _, err := hex.DecodeString(h); err != nil {
			return zero, err
		}
	}
	if len(ledger.Boxes) < 2 || src == "" || dst == "" || src == dst || ledger.Project == "" || ledger.Project == "default" {
		return zero, errors.New("invalid review pair or complete ledger")
	}
	a, b := false, false
	for _, v := range ledger.Boxes {
		if v.UUID == src {
			a = true
		}
		if v.UUID == dst {
			b = true
		}
	}
	if !a || !b {
		return zero, errors.New("review pair missing from owner ledger")
	}
	raw, err := json.Marshal(ledger)
	if err != nil {
		return zero, err
	}
	ledgerSum := sha256.Sum256(append([]byte("boxweave/b2/ledger/v1\n"), raw...))
	result := NativePeerConntrackPlan{
		Project: ledger.Project, Bridge: ledger.Bridge, ACL: ledger.ACL,
		SourceUUID: src, TargetUUID: dst, HostSHA256: hostSHA,
		LedgerSHA256: hex.EncodeToString(ledgerSum[:]), NativeDigestSHA256: nativeSHA,
		MutationAuthorized: false,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return zero, err
	}
	hash := sha256.Sum256(append([]byte("boxweave/b2/conntrack-plan/v1\n"), data...))
	result.PlanSHA256 = hex.EncodeToString(hash[:])
	return result, nil
}

func nativePeerConntrackReviewWithClient(ctx context.Context, s *state.Store, root incus.InstanceServer, hostSHA, project, bridge, acl, sourceUUID, targetUUID string) (NativePeerConntrackPlan, error) {
	var zero NativePeerConntrackPlan
	if !s.IsNetworkAuditReadOnly() || root == nil {
		return zero, errors.New("read-only owner ledger and native root required")
	}
	t, ledger, err := nativePeerConntrackTargetFromStore(ctx, s, project, bridge, acl, sourceUUID, targetUUID)
	if err != nil {
		return zero, err
	}
	first, err := nativePeerBridgeAuditWithClient(ctx, root, t)
	if err != nil {
		return zero, err
	}
	if err = nativePeerConntrackAllStopped(ctx, root, t); err != nil {
		return zero, err
	}
	if err = nativePeerBridgeKernelNoLivePorts(ctx, t.Bridge); err != nil {
		return zero, err
	}
	srcAt, dstAt := -1, -1
	for i, b := range t.Boxes {
		if b.UUID == sourceUUID {
			srcAt = i
		}
		if b.UUID == targetUUID {
			dstAt = i
		}
	}
	if srcAt < 0 || dstAt < 0 || srcAt == dstAt {
		return zero, errors.New("review pair not in exact native scope")
	}
	rules, _, err := root.GetNetworkACL(t.ACL)
	if err != nil {
		return zero, err
	}
	if err = nativePeerACLDropCovered(rules, t.Boxes[srcAt], t.Boxes[dstAt]); err != nil {
		return zero, err
	}
	second, err := nativePeerBridgeAuditWithClient(ctx, root, t)
	if err != nil || first.NativeDigestSHA256 != second.NativeDigestSHA256 {
		return zero, errors.New("native root changed during read-only review")
	}
	if err = nativePeerConntrackAllStopped(ctx, root, t); err != nil {
		return zero, err
	}
	if err = nativePeerBridgeKernelNoLivePorts(ctx, t.Bridge); err != nil {
		return zero, err
	}
	after, err := s.ReadCompleteNetworkAuditLedger(ctx, project, bridge, acl)
	if err != nil || !reflect.DeepEqual(ledger, after) {
		return zero, fmt.Errorf("SQLite owner ledger drift during review: %v", err)
	}
	return nativePeerConntrackSealedPlan(ledger, hostSHA, first.NativeDigestSHA256, sourceUUID, targetUUID)
}

// NativePeerConntrackReview is an operator-side, host-root-only read.
// Future mutation needs separate explicit approval and durable UNKNOWN journal.
func NativePeerConntrackReview(ctx context.Context, s *state.Store, project, bridge, acl, sourceUUID, targetUUID string) (NativePeerConntrackPlan, error) {
	var zero NativePeerConntrackPlan
	if os.Geteuid() != 0 || !s.IsNetworkAuditReadOnly() {
		return zero, errors.New("root and protected read-only SQLite required")
	}
	u, err := user.Lookup("boxweave")
	if err != nil || project != "user-"+u.Uid {
		return zero, errors.New("non-owned root review Project")
	}
	mid, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		return zero, err
	}
	raw := strings.TrimSpace(string(mid))
	if len(raw) != 32 {
		return zero, errors.New("invalid host identity")
	}
	if _, err = hex.DecodeString(raw); err != nil {
		return zero, errors.New("invalid host identity")
	}
	hostSum := sha256.Sum256([]byte(raw))
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		return zero, err
	}
	return nativePeerConntrackReviewWithClient(ctx, s, root, hex.EncodeToString(hostSum[:]), project, bridge, acl, sourceUUID, targetUUID)
}
