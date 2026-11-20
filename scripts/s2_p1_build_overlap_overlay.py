#!/usr/bin/env python3
"""Build a disposable overlay at the verified-native, pre-atomic Create boundary.

Committed lifecycle.go and candidate bytes stay unchanged. Successful Create no
longer has a bound-UUID VERIFYING window: UUID/claim/terminal commit atomically.
"""
import hashlib
import json
import os
from pathlib import Path
import sys

if len(sys.argv) != 3:
    raise SystemExit("usage: s2_p1_build_overlap_overlay.py <lifecycle.go> <lab>")
source = Path(sys.argv[1]).resolve(strict=True)
lab = Path(sys.argv[2]).resolve(strict=True)
if not (
    os.environ.get("GITHUB_ACTIONS") == "true"
    and os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    and os.geteuid() != 0
    and str(lab).startswith("/tmp/bw-v1-s2-ghci.")
    and lab.name == "p1-native-overlap"
    and source.parts[-3:] == ("internal", "state", "lifecycle.go")
    and lab.is_dir()
):
    raise SystemExit("refusing overlay outside single-use GitHub-hosted acceptance")

original = source.read_bytes()
before = original.decode("utf-8")
function = "func (s *Store) FinalizeVerifiedCreate(ctx context.Context, opID, project, name, uuid string) error {"
assert before.count(function) == 1, "atomic Create finalizer must be exactly one"
prefix, body = before.split(function)
needle = "\ttx, err := s.db.BeginTx(ctx, nil)"
assert body.count(needle) >= 1, "atomic Create transaction anchor changed"
imports = '\t"errors"\n\t"fmt"'
assert before.count(imports) == 1, "lifecycle import layout changed; fail closed"
extra = '\t"encoding/json"\n\t"errors"\n\t"fmt"\n\t"os"\n\t"path/filepath"'

# Pause before opening the transaction so independently handled HTTP reads
# and denied Deletes can run on the Store's single connection. The caller
# already verified native UUID, original marker, shape and finality.
hook = '''
	// TEST-ONLY GO OVERLAY: never in committed or release lifecycle.go.
	if os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" &&
		os.Geteuid() != 0 && name == "bw-s2-p1-overlap-110x" &&
		os.Getenv("BW_S2_OVERLAP_ACTIVATE") == "one-disposable-original" {
		lab := os.Getenv("BW_S2_OVERLAP_LAB")
		if !strings.HasPrefix(lab, "/tmp/bw-v1-s2-ghci.") ||
			filepath.Base(lab) != "p1-native-overlap" {
			return errors.New("S4_TEST_ONLY_INVALID_OVERLAY_SCOPE")
		}
		receipt, readErr := s.Get(ctx, opID)
		if readErr != nil || receipt.Action != "box_create" ||
			receipt.State != "VERIFYING" || receipt.BoxUUID != "" ||
			receipt.NativeOperationID == "" || len(uuid) != 36 {
			return errors.New("S4_TEST_ONLY_PREATOMIC_IDENTITY_UNVERIFIED")
		}
		evidence, marshalErr := json.Marshal(map[string]string{
			"phase": "NATIVE_VERIFIED_BEFORE_ATOMIC_CREATE",
			"uuid": uuid, "original_id": opID,
			"native_id": receipt.NativeOperationID, "project": project})
		if marshalErr != nil || os.WriteFile(filepath.Join(lab, "held.tmp"), evidence, 0600) != nil ||
			os.Rename(filepath.Join(lab, "held.tmp"), filepath.Join(lab, "held")) != nil {
			return errors.New("S4_TEST_ONLY_PREATOMIC_WITNESS_UNAVAILABLE")
		}
		released := false
		for i := 0; i < 950; i++ {
			if _, err := os.Stat(filepath.Join(lab, "release")); err == nil {
				released = true
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
		if !released {
			return errors.New("S4_TEST_ONLY_PREATOMIC_HOOK_TIMED_OUT")
		}
	}
'''
modified = (prefix + function + body.replace(needle, hook + needle, 1)).replace(imports, extra)
assert modified.count("TEST-ONLY GO OVERLAY") == 1
overlay_file = lab / "lifecycle.test-overlay.go"
json_file = lab / "go-overlay.json"
overlay_file.write_text(modified, encoding="utf-8")
json_file.write_text(
    json.dumps({"Replace": {str(source): str(overlay_file)}}, sort_keys=True),
    encoding="utf-8",
)
assert source.read_bytes() == original, "original lifecycle unexpectedly changed"
print("S4_A_OVERLAY_SOURCE_UNMODIFIED_SHA256=" + hashlib.sha256(original).hexdigest())
print("S4_A_OVERLAY_TEST_ONLY_LIFECYCLE_SHA256=" + hashlib.sha256(modified.encode()).hexdigest())
print("S4_A_OVERLAY_PREATOMIC_EXACT_SINGLE_INSERTION=PASS")
