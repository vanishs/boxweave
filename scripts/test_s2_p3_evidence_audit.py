#!/usr/bin/env python3
"""P3 verifier's independent negative controls; no Incus or Box mutations."""
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

SCRIPT = Path(__file__).with_name("s2_p3_evidence_audit.py")
spec = importlib.util.spec_from_file_location("s2_p3_evidence_audit", SCRIPT)
audit = importlib.util.module_from_spec(spec)
spec.loader.exec_module(audit)
SOURCE = "a" * 40
HASH = "b" * 64
BASE = (
    [f"S2_NATIVE_CANDIDATE_SOURCE_SHA={SOURCE}",
     f"{HASH}  /tmp/bw-v1-s2-ghci.test/boxweave"]
    + [f"{marker}=PASS" for marker in audit.MANDATORY]
)

class EvidenceNegativeControls(unittest.TestCase):
    def check_log(self, lines, source=SOURCE, success=False, wan=None):
        with tempfile.TemporaryDirectory(prefix="s2-p3-audit-") as tmp:
            path = Path(tmp) / "native.log"
            out = Path(tmp) / "manifest.json"
            path.write_text("\n".join(lines) + "\n", encoding="utf-8")
            run = subprocess.run(
                [sys.executable, str(SCRIPT), "--log", str(path),
                 "--source", source, "--output", str(out)],
                capture_output=True, text=True, timeout=10, check=False,
            )
            self.assertEqual(run.returncode == 0, success,
                             f"stdout={run.stdout}\nstderr={run.stderr}")
            if success:
                manifest = json.loads(out.read_text(encoding="utf-8"))
                self.assertEqual(manifest["source_sha"], source)
                self.assertEqual(manifest["native_candidate_executable_sha256"], HASH)
                self.assertEqual(manifest["required_markers_checked"], len(audit.MANDATORY))
                if wan is not None:
                    self.assertEqual(manifest["wan_public_ipv4_https"], wan)
            else:
                self.assertFalse(out.exists(), "failed evidence must never emit PASS manifest")

    def test_exact_source_binary_and_required_markers(self):
        self.check_log(BASE, success=True, wan="UNVERIFIED_GUEST_WAN_POSITIVE")

    def test_wrong_source_or_duplicate_source(self):
        self.check_log(BASE, source="c" * 40)
        self.check_log(BASE + [BASE[0]])

    def test_missing_or_duplicated_binary_digest(self):
        self.check_log([line for line in BASE if HASH not in line])
        self.check_log(BASE + [BASE[1]])

    def test_missing_required_marker(self):
        self.check_log([line for line in BASE if line != f"{audit.MANDATORY[-1]}=PASS"])

    def test_wan_claim_without_independent_https_positive(self):
        self.check_log(BASE + ["S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS"])

    def test_wan_requires_both_positive_and_toggle(self):
        self.check_log(BASE + ["S2_NATIVE_EXTERNAL_IPV4_WAN_HTTPS_POSITIVE_CONTROL=PASS"],
                       success=True, wan="UNVERIFIED_GUEST_WAN_POSITIVE")
        self.check_log(BASE + [
            "S2_NATIVE_EXTERNAL_IPV4_WAN_HTTPS_POSITIVE_CONTROL=PASS",
            "S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS",
        ], success=True, wan="PASS")

    def test_malformed_source_and_truncated_log(self):
        self.check_log(BASE, source="invalid")
        self.check_log(BASE[:2])

if __name__ == "__main__":
    unittest.main()
