#!/usr/bin/env bash
# Source checkout entry point. The installer creates /usr/local/bin/boxweave.
set -euo pipefail
ROOT=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
exec python3 "$ROOT/server/cli.py" "$@"
