#!/bin/sh
# BoxWeave reviewed uninstall entrypoint. No args -> help, no host action.
set -eu
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
exec python3 "$SCRIPT_DIR/scripts/uninstall_core.py" "$@"
