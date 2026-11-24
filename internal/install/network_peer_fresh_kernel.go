package install

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Successful full table inventory is required for absence. A failed
// list-table command is NOT proof of absence. Refuse any old scoped table.
func nativePeerFreshTablePresent(raw []byte, expected string) (bool, error) {
	if len(raw) == 0 || len(raw) > 1<<20 || !nativePeerGuardTablePattern.MatchString(expected) {
		return false, errors.New("missing or unsafe nft table inventory")
	}
	var root map[string][]map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil || len(root) != 1 || len(root["nftables"]) == 0 || len(root["nftables"]) > 4096 {
		return false, errors.New("invalid nft table inventory")
	}
	found, meta := false, false
	seen := map[string]bool{}
	for _, entry := range root["nftables"] {
		if len(entry) != 1 {
			return false, errors.New("ambiguous nft inventory object")
		}
		for key, value := range entry {
			var obj map[string]any
			if json.Unmarshal(value, &obj) != nil {
				return false, errors.New("invalid nft inventory object")
			}
			if key == "metainfo" {
				if meta || len(seen) > 0 || len(obj) != 3 || obj["json_schema_version"] != float64(1) {
					return false, errors.New("untrusted nft inventory schema")
				}
				meta = true
				continue
			}
			if key != "table" || !meta || len(obj) != 3 {
				return false, errors.New("unexpected nft inventory object")
			}
			family, ok := obj["family"].(string)
			name, ok2 := obj["name"].(string)
			if !ok || !ok2 || family == "" || name == "" || nativePeerGuardValidHandle(obj["handle"], map[float64]bool{}) != nil || seen[family+"/"+name] {
				return false, errors.New("untrusted or duplicate nft table")
			}
			seen[family+"/"+name] = true
			if strings.HasPrefix(name, "bw_pg_") {
				if family != "bridge" || name != expected || found {
					return false, errors.New("foreign or stale scoped guard table")
				}
				found = true
			}
		}
	}
	if !meta {
		return false, errors.New("missing nft inventory metadata")
	}
	return found, nil
}
func nativePeerFreshTableLive(ctx context.Context, p NativePeerGuardPlan) (bool, error) {
	if os.Geteuid() != 0 {
		return false, errors.New("root-local nft inventory required")
	}
	cmd := exec.CommandContext(ctx, "/usr/sbin/nft", "-j", "-a", "list", "tables")
	cmd.Env = []string{"LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
	var data bytes.Buffer
	cmd.Stdout = &nativePeerGuardLimitWriter{dst: &data, max: 1 << 20}
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return false, errors.New("native nft table inventory unavailable")
	}
	return nativePeerFreshTablePresent(data.Bytes(), p.Table)
}
func nativePeerFreshGuardAbsent(ctx context.Context, p NativePeerGuardPlan) error {
	found, err := nativePeerFreshTableLive(ctx, p)
	if err != nil {
		return err
	}
	if found {
		return errors.New("existing scoped guard; no native adoption")
	}
	return nil
}

// create table is exclusive, unlike idempotent add table. One atomic
// batch cannot adopt a table raced into existence between check and write.
func nativePeerFreshExclusiveBatch(p NativePeerGuardPlan) (string, error) {
	if !nativePeerGuardTablePattern.MatchString(p.Table) || p.DigestSHA256 != nativePeerGuardDigest(p) {
		return "", errors.New("unverified scoped guard")
	}
	first := "flush chain bridge " + p.Table + " protect\n"
	if !strings.HasPrefix(p.Replace, first) {
		return "", errors.New("invalid scoped guard replacement body")
	}
	return "create table bridge " + p.Table + "\nadd chain bridge " + p.Table + " protect { type filter hook forward priority -250; policy accept; }\n" + strings.TrimPrefix(p.Replace, first), nil
}
func nativePeerFreshGuardInstall(ctx context.Context, p NativePeerGuardPlan) error {
	batch, err := nativePeerFreshExclusiveBatch(p)
	if err != nil {
		return err
	}
	return nativePeerGuardFixedNftBatch(ctx, batch)
}
