#!/usr/bin/env python3
"""Read-only identity check for one original N2 CI transcript, not acceptance."""
import hashlib
from pathlib import Path
import re
import sys


def evaluate(raw, source):
    if not re.fullmatch(r"[0-9a-f]{40}", source):
        raise ValueError("invalid exact source")
    if not raw or len(raw) > 32 * 1024 * 1024:
        raise ValueError("missing or oversized N2 transcript")
    lines = raw.decode("utf-8", "replace").splitlines()

    def exact(prefix, pattern):
        values = [line[len(prefix):] for line in lines if line.startswith(prefix)]
        if len(values) != 1 or not re.fullmatch(pattern, values[0]):
            raise ValueError("missing, malformed or ambiguous " + prefix)
        return values[0]

    actual = exact("N2_INSTALL_SOURCE=", r"[0-9a-f]{40}")
    built = exact("N2_INSTALL_CANDIDATE_SHA256=", r"[0-9a-f]{64}")
    installed = exact("N2_INSTALLED_CANDIDATE_SHA256=", r"[0-9a-f]{64}")
    if actual != source or built != installed:
        raise ValueError("source or installed candidate mismatch")
    return {
        "source": actual, "candidate_sha256": installed,
        "transcript_sha256": hashlib.sha256(raw).hexdigest(),
    }


if __name__ == "__main__":
    import json
    if len(sys.argv) != 3:
        raise SystemExit("usage: LOG EXACT_SOURCE")
    try:
        print("N2_EVIDENCE_IDENTITY=" + json.dumps(
            evaluate(Path(sys.argv[1]).read_bytes(), sys.argv[2]), sort_keys=True))
    except (OSError, ValueError) as error:
        raise SystemExit(str(error))
