package install

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/netip"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/vanishs/boxweave/internal/state"
)

var nativePeerGuardTablePattern = regexp.MustCompile(`^bw_pg_[0-9a-f]{16}$`)

func nativePeerGuardDigest(plan NativePeerGuardPlan) string {
	sum := sha256.Sum256([]byte("bw-peer-nft-guard-v2\n" +
		plan.Install + "\n" + plan.Replace + "\n" + plan.ExpectedKernelJSON))
	return hex.EncodeToString(sum[:])
}
func nativePeerGuardMatch(left, right any, op string) any {
	return map[string]any{"match": map[string]any{"op": op, "left": left, "right": right}}
}
func nativePeerGuardPayload(protocol, field string) any {
	return map[string]any{"payload": map[string]any{"protocol": protocol, "field": field}}
}
func nativePeerGuardMetaBridge(bridge string) any {
	return nativePeerGuardMatch(map[string]any{"meta": map[string]any{"key": "ibrname"}}, bridge, "==")
}
func nativePeerGuardPrefix(prefix netip.Prefix) any {
	return map[string]any{"prefix": map[string]any{"addr": prefix.Masked().Addr().String(), "len": prefix.Bits()}}
}
func nativePeerGuardEndpointExpr(bridge, proto, src, dst, sourceMAC, targetMAC string, reply bool) []any {
	expr := []any{
		nativePeerGuardMetaBridge(bridge),
		nativePeerGuardMatch(nativePeerGuardPayload("ether", "daddr"), targetMAC, "=="),
		nativePeerGuardMatch(nativePeerGuardPayload("ether", "saddr"), sourceMAC, "=="),
		nativePeerGuardMatch(nativePeerGuardPayload(proto, "saddr"), src, "=="),
		nativePeerGuardMatch(nativePeerGuardPayload(proto, "daddr"), dst, "=="),
	}
	direction := "original"
	if reply {
		direction = "reply"
	}
	expr = append(expr, nativePeerGuardMatch(map[string]any{"ct": map[string]any{"key": "direction"}}, direction, "=="))
	if reply {
		expr = append(expr, nativePeerGuardMatch(map[string]any{"ct": map[string]any{"key": "state"}}, "established", "in"))
	} else {
		expr = append(expr, nativePeerGuardMatch(map[string]any{"ct": map[string]any{"key": "state"}}, []string{"established", "new"}, "in"))
	}
	return append(expr, map[string]any{"accept": nil})
}
func nativePeerGuardSubnetExpr(bridge, proto string, prefix netip.Prefix) []any {
	return []any{
		nativePeerGuardMetaBridge(bridge),
		nativePeerGuardMatch(nativePeerGuardPayload(proto, "saddr"), nativePeerGuardPrefix(prefix), "=="),
		nativePeerGuardMatch(nativePeerGuardPayload(proto, "daddr"), nativePeerGuardPrefix(prefix), "=="),
		map[string]any{"counter": map[string]any{"packets": 0, "bytes": 0}}, map[string]any{"drop": nil},
	}
}
func nativePeerGuardNDExpr(bridge string) []any {
	return []any{
		nativePeerGuardMetaBridge(bridge),
		nativePeerGuardMatch(nativePeerGuardPayload("ip6", "hoplimit"), 255, "=="),
		nativePeerGuardMatch(nativePeerGuardPayload("icmpv6", "type"), map[string]any{"set": []string{"nd-neighbor-solicit", "nd-neighbor-advert"}}, "=="),
		map[string]any{"accept": nil},
	}
}
func nativePeerGuardIPv6DropExpr(bridge string, base, dest netip.Prefix) []any {
	return []any{
		nativePeerGuardMetaBridge(bridge),
		nativePeerGuardMatch(nativePeerGuardPayload("ip6", "saddr"), map[string]any{"set": []any{nativePeerGuardPrefix(base), nativePeerGuardPrefix(netip.MustParsePrefix("fe80::/10"))}}, "=="),
		nativePeerGuardMatch(nativePeerGuardPayload("ip6", "daddr"), nativePeerGuardPrefix(dest), "=="),
		map[string]any{"counter": map[string]any{"packets": 0, "bytes": 0}}, map[string]any{"drop": nil},
	}
}

// The compiler builds a SECOND representation of the intended policy, using
// nftables' observed v1 JSON expression ABI. Kernel readback comparisons use
// this semantic expression vector rather than an SDK response or nft exit 0.
// Unknown kernel schema or ordering FAILS CLOSED; never normalize away allows,
// different source MACs, hidden rules, hooks or a changed action.
func nativePeerGuardExpectedKernelJSON(bridge string, base4, base6 netip.Prefix,
	owners map[string]NativePeerGuardMember, grants []state.PeerGrantBinding) (string, error) {
	pairs := slices.Clone(grants)
	slices.SortFunc(pairs, func(a, b state.PeerGrantBinding) int {
		return strings.Compare(a.SourceUUID+"/"+a.TargetUUID, b.SourceUUID+"/"+b.TargetUUID)
	})
	rules := make([][]any, 0, len(pairs)*4+4)
	for _, p := range pairs {
		src, okSrc := owners[p.SourceUUID]
		dst, okDst := owners[p.TargetUUID]
		if !okSrc || !okDst {
			return "", errors.New("guard kernel expectation missing exact owner")
		}
		for _, f := range []struct{ proto, s, d string }{
			{"ip", src.IPv4, dst.IPv4}, {"ip6", src.IPv6, dst.IPv6},
		} {
			rules = append(rules, nativePeerGuardEndpointExpr(bridge, f.proto, f.s, f.d, src.MAC, dst.MAC, false))
			rules = append(rules, nativePeerGuardEndpointExpr(bridge, f.proto, f.d, f.s, dst.MAC, src.MAC, true))
		}
	}
	rules = append(rules,
		nativePeerGuardSubnetExpr(bridge, "ip", base4.Masked()),
		nativePeerGuardNDExpr(bridge),
		nativePeerGuardIPv6DropExpr(bridge, base6.Masked(), base6.Masked()),
		nativePeerGuardIPv6DropExpr(bridge, base6.Masked(), netip.MustParsePrefix("fe80::/10")),
	)
	encoded, err := json.Marshal(rules)
	return string(encoded), err
}

// NativePeerGuardKernelVerifyJSON is a STRICT, read-only proof of exactly ONE
// owned nft bridge table. It rejects extra/missing rules, shadowing accepts,
// changed hook priority or policy, foreign tables/chains and malformed counter
// metadata. Handle and packet/byte counters are the only volatile fields.
// A passing result is necessary but NOT sufficient for the full original:
// root must also recheck native owner+Port+MAC+lease+SQLite and boot fencing.
func NativePeerGuardKernelVerifyJSON(plan NativePeerGuardPlan, raw []byte) error {
	if !nativePeerGuardTablePattern.MatchString(plan.Table) ||
		len(plan.ExpectedKernelJSON) == 0 ||
		len(plan.DigestSHA256) != 64 ||
		plan.DigestSHA256 != nativePeerGuardDigest(plan) ||
		len(raw) == 0 || len(raw) > 1<<20 {
		return errors.New("untrusted root guard plan or bounded nft JSON")
	}
	var expected [][]any
	if err := json.Unmarshal([]byte(plan.ExpectedKernelJSON), &expected); err != nil ||
		len(expected) < 4 || len(expected) > 256 {
		return errors.New("malformed root guard expected expression inventory")
	}
	var envelope struct {
		NFTables []json.RawMessage `json:"nftables"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&envelope); err != nil {
		return fmt.Errorf("nft readback JSON invalid: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF || len(envelope.NFTables) != len(expected)+3 {
		return errors.New("incomplete, extra or ambiguous nft kernel readback")
	}
	seenMeta, seenTable, seenChain := false, false, false
	handles := map[float64]bool{}
	ruleIndex := 0
	for _, item := range envelope.NFTables {
		var one map[string]json.RawMessage
		if err := json.Unmarshal(item, &one); err != nil || len(one) != 1 {
			return errors.New("unexpected native nft object")
		}
		for key, value := range one {
			var v map[string]any
			if err := json.Unmarshal(value, &v); err != nil {
				return errors.New("nft object cannot be inspected")
			}
			switch key {
			case "metainfo":
				if seenMeta || seenTable || len(v) != 3 ||
					v["json_schema_version"] != float64(1) {
					return errors.New("untrusted nft JSON schema/version")
				}
				seenMeta = true
			case "table":
				if !seenMeta || seenTable || seenChain ||
					len(v) != 3 || v["family"] != "bridge" || v["name"] != plan.Table {
					return errors.New("missing exact owned nft bridge table")
				}
				if err := nativePeerGuardValidHandle(v["handle"], map[float64]bool{}); err != nil {
					return err
				}
				seenTable = true
			case "chain":
				if !seenTable || seenChain || ruleIndex != 0 ||
					len(v) != 8 || v["family"] != "bridge" || v["table"] != plan.Table ||
					v["name"] != "protect" || v["type"] != "filter" ||
					v["hook"] != "forward" || v["prio"] != float64(-250) ||
					v["policy"] != "accept" {
					return errors.New("nft base chain hook/priority changed or shadowed")
				}
				if err := nativePeerGuardValidHandle(v["handle"], map[float64]bool{}); err != nil {
					return err
				}
				seenChain = true
			case "rule":
				if !seenChain || ruleIndex >= len(expected) ||
					len(v) != 5 || v["family"] != "bridge" ||
					v["table"] != plan.Table || v["chain"] != "protect" {
					return errors.New("foreign nft guard rule, wrong chain or unexpected rule count")
				}
				if err := nativePeerGuardValidHandle(v["handle"], handles); err != nil {
					return err
				}
				expr, ok := v["expr"].([]any)
				if !ok {
					return errors.New("nft guard rule has no inspected expressions")
				}
				if !nativePeerGuardNormalizeCounters(expr) ||
					!reflect.DeepEqual(expr, expected[ruleIndex]) {
					return fmt.Errorf("nft guard rule %d mismatched independent expected kernel policy", ruleIndex)
				}
				ruleIndex++
			default:
				return errors.New("unrecognized native nft table content")
			}
		}
	}
	if !seenMeta || !seenTable || !seenChain || ruleIndex != len(expected) {
		return errors.New("missing independent full nft policy readback")
	}
	return nil
}
func nativePeerGuardValidHandle(raw any, seen map[float64]bool) error {
	n, ok := raw.(float64)
	if !ok || n <= 0 || math.IsInf(n, 0) || math.IsNaN(n) || math.Trunc(n) != n || seen[n] {
		return errors.New("invalid or repeated native nft handle")
	}
	seen[n] = true
	return nil
}
func nativePeerGuardNormalizeCounters(expr []any) bool {
	for _, item := range expr {
		m, ok := item.(map[string]any)
		if !ok || len(m) != 1 {
			return false
		}
		counter, ok := m["counter"]
		if !ok {
			continue
		}
		c, ok := counter.(map[string]any)
		if !ok || len(c) != 2 {
			return false
		}
		for _, key := range []string{"packets", "bytes"} {
			x, ok := c[key].(float64)
			if !ok || x < 0 || math.Trunc(x) != x || math.IsInf(x, 0) || math.IsNaN(x) {
				return false
			}
			c[key] = float64(0)
		}
	}
	return true
}

// nativePeerGuardNFTRead is the ONLY production external invocation at this
// stage, a root-local, read-only fixed binary with a constrained target.
// There is NO mutation, shell, MCP entrypoint or user-supplied command.
func nativePeerGuardNFTRead(ctx context.Context, table string) ([]byte, error) {
	if !nativePeerGuardTablePattern.MatchString(table) {
		return nil, errors.New("untrusted nft table")
	}
	cmd := exec.CommandContext(ctx, "/usr/sbin/nft", "-j", "-a", "list", "table", "bridge", table)
	cmd.Env = []string{"LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
	var data bytes.Buffer
	limit := int64(1<<20) + 1
	cmd.Stdout = &nativePeerGuardLimitWriter{dst: &data, max: limit}
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("native nft read uncertain: %w", err)
	}
	if data.Len() > 1<<20 {
		return nil, errors.New("nft kernel table readback too large")
	}
	return data.Bytes(), nil
}

type nativePeerGuardLimitWriter struct {
	dst *bytes.Buffer
	max int64
}

func (w *nativePeerGuardLimitWriter) Write(p []byte) (int, error) {
	if int64(w.dst.Len()+len(p)) > w.max {
		return 0, errors.New("native nft JSON output exceeded bound")
	}
	return w.dst.Write(p)
}

// NativePeerGuardKernelReadback is a root-local pre/post effect gate. The
// protected root dispatcher must supply the *just compiled* plan and still
// perform complete SQLite, Incus NIC, sysfs-port and original journal checks.
func NativePeerGuardKernelReadback(ctx context.Context, plan NativePeerGuardPlan) error {
	if os.Geteuid() != 0 {
		return errors.New("kernel nft guard readback requires root-local caller")
	}
	if !nativePeerGuardTablePattern.MatchString(plan.Table) ||
		plan.DigestSHA256 != nativePeerGuardDigest(plan) {
		return errors.New("untrusted root guard digest")
	}
	data, err := nativePeerGuardNFTRead(ctx, plan.Table)
	if err != nil {
		return err
	}
	return NativePeerGuardKernelVerifyJSON(plan, data)
}
