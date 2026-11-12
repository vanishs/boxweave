#!/usr/bin/env python3
"""Make a disposable-only Go source overlay at the exact committed post-verify boundary.

Production worker.go stays byte-identical. The overlay is compiled ONLY as a
CI-specific acceptance binary; it is never included in the release build.
"""
import hashlib
import json
import os
from pathlib import Path
import sys

if len(sys.argv) != 3:
    raise SystemExit("usage: s2_p1_build_overlap_overlay.py <source-worker.go> <lab>")
source = Path(sys.argv[1]).resolve(strict=True)
lab = Path(sys.argv[2]).resolve(strict=True)
if not (
    os.environ.get("GITHUB_ACTIONS") == "true"
    and os.environ.get("RUNNER_ENVIRONMENT") == "github-hosted"
    and os.geteuid() != 0
    and str(lab).startswith("/tmp/bw-v1-s2-ghci.")
    and lab.name == "p1-native-overlap"
    and source.parts[-3:] == ("internal", "state", "worker.go")
    and lab.is_dir()
):
    raise SystemExit("refusing overlay outside single-use GitHub-hosted acceptance")

original = source.read_bytes()
before = original.decode("utf-8")
needle = "\tif err = w.Store.Verified(context.Background(), operationID); err != nil {"
assert before.count(needle) == 1, "post-verify receipt site must be exactly one"
imports = '\t"errors"\n\t"time"'
assert before.count(imports) == 1, "worker import layout changed; fail closed"
extra = '''\t"errors"
\t"os"
\t"path/filepath"
\t"strings"
\t"time"'''

# Test pause ONLY after verify(ctx) returned nil: RegisterCreatedBox already
# committed exact native UUID + owned instance in one SQLite transaction.
# No exported product hook, no alternate API, no injected fake native result.
hook = """
	// TEST-ONLY GO OVERLAY: not in the committed or release worker.go.
	// Preserve exact native effect and all original DB transitions.
	if os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" &&
		os.Geteuid() != 0 &&
		os.Getenv("BW_S2_OVERLAP_ACTIVATE") == "one-disposable-original" {
		lab := os.Getenv("BW_S2_OVERLAP_LAB")
		if strings.HasPrefix(lab, "/tmp/bw-v1-s2-ghci.") &&
			filepath.Base(lab) == "p1-native-overlap" {
			receipt, readErr := w.Store.Get(context.Background(), operationID)
			if readErr == nil && receipt.Action == "box_create" &&
				receipt.State == "VERIFYING" && len(receipt.BoxUUID) == 36 &&
				receipt.NativeOperationID != "" {
				if err := os.WriteFile(filepath.Join(lab, "held"),
					[]byte("POST_REGISTER_CREATED_BOX_COMMIT_VERIFIED_EFFECT_RETURNED\\n"), 0600); err != nil {
					_ = w.Store.Unknown(context.Background(), operationID, "VERIFYING",
						"S2_TEST_ONLY_HOOK_WITNESS_UNAVAILABLE")
					return
				}
				released := false
				for i := 0; i < 950; i++ {
					if _, err := os.Stat(filepath.Join(lab, "release")); err == nil {
						released = true
						break
					}
					time.Sleep(100 * time.Millisecond)
				}
				if !released {
					_ = w.Store.Unknown(context.Background(), operationID, "VERIFYING",
						"S2_TEST_ONLY_HOOK_TIMED_OUT")
					return
				}
			}
		}
	}
"""
modified = before.replace(imports, extra).replace(needle, hook + needle)
assert modified.count("TEST-ONLY GO OVERLAY") == 1
overlay_file = lab / "worker.test-overlay.go"
json_file = lab / "go-overlay.json"
overlay_file.write_text(modified, encoding="utf-8")
json_file.write_text(
    json.dumps({"Replace": {str(source): str(overlay_file)}}, sort_keys=True),
    encoding="utf-8",
)
assert source.read_bytes() == original, "original worker unexpectedly changed"
print("S2_P1_OVERLAY_SOURCE_UNMODIFIED_SHA256=" + hashlib.sha256(original).hexdigest())
print("S2_P1_OVERLAY_TEST_ONLY_WORKER_SHA256=" + hashlib.sha256(modified.encode()).hexdigest())
print("S2_P1_OVERLAY_POST_BIND_ONLY_EXACT_SINGLE_INSERTION=PASS")
